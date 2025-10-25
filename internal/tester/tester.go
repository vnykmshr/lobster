// Package tester implements the core load testing engine with concurrent workers.
package tester

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vnykmshr/goflow/pkg/ratelimit/bucket"
	"github.com/vnykmshr/lobster/internal/crawler"
	"github.com/vnykmshr/lobster/internal/domain"
	"github.com/vnykmshr/lobster/internal/robots"
	"github.com/vnykmshr/lobster/internal/util"
)

// Tester orchestrates the stress testing process
type Tester struct {
	config        domain.TesterConfig
	client        *http.Client
	urlQueue      chan domain.URLTask
	results       *domain.TestResults
	rateLimiter   bucket.Limiter
	crawler       *crawler.Crawler
	robotsParser  *robots.Parser
	logger        *slog.Logger

	// Result channels for lock-free aggregation
	validationsCh   chan domain.URLValidation
	errorsCh        chan domain.ErrorInfo
	responseTimesCh chan domain.ResponseTimeEntry
	slowRequestsCh  chan domain.SlowRequest
}

// New creates a new stress tester
func New(config domain.TesterConfig, logger *slog.Logger) (*Tester, error) {
	// Create crawler
	crawlerInstance, err := crawler.New(config.BaseURL, config.MaxDepth)
	if err != nil {
		return nil, fmt.Errorf("creating crawler: %w", err)
	}

	// Create token bucket rate limiter using goflow
	var rateLimiter bucket.Limiter
	if config.Rate > 0 {
		// Create token bucket with burst capacity of 2x the rate per second
		burst := int(config.Rate * 2)
		if burst < 1 {
			burst = 1
		}

		rateLimiter, err = bucket.NewSafe(bucket.Limit(config.Rate), burst)
		if err != nil {
			logger.Error("Failed to create rate limiter", "error", err)
			rateLimiter = nil
		}
	}

	// Use configured queue size, default to 10000 if not set
	queueSize := config.QueueSize
	if queueSize <= 0 {
		queueSize = 10000
	}

	// Create HTTP client with optional TLS skip verify
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
	}

	// Configure TLS if InsecureSkipVerify is enabled
	if config.InsecureSkipVerify {
		logger.Warn("⚠️  INSECURE: TLS certificate verification is disabled. Use only for testing with self-signed certificates!")
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // Intentionally insecure for testing self-signed certs
			},
		}
	}

	// Create robots.txt parser and fetch robots.txt
	robotsParser := robots.New(config.UserAgent)
	if !config.IgnoreRobots {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := robotsParser.FetchAndParse(ctx, config.BaseURL); err != nil {
			logger.Warn("Failed to fetch robots.txt, proceeding with caution", "error", err)
		} else if robotsParser.RobotsTxtFound() {
			logger.Info("robots.txt found and parsed successfully")
		} else {
			logger.Debug("No robots.txt found, all paths allowed")
		}
	} else {
		logger.Warn("⚠️  Ignoring robots.txt directives. Please ensure you have permission to test this site!")
	}

	return &Tester{
		config:          config,
		client:          httpClient,
		urlQueue:        make(chan domain.URLTask, queueSize),
		results:         &domain.TestResults{URLValidations: make([]domain.URLValidation, 0)},
		rateLimiter:     rateLimiter,
		crawler:         crawlerInstance,
		robotsParser:    robotsParser,
		logger:          logger,
		validationsCh:   make(chan domain.URLValidation, 1000),
		errorsCh:        make(chan domain.ErrorInfo, 1000),
		responseTimesCh: make(chan domain.ResponseTimeEntry, 1000),
		slowRequestsCh:  make(chan domain.SlowRequest, 100),
	}, nil
}

// Run executes the stress test
func (t *Tester) Run(ctx context.Context) (*domain.TestResults, error) {
	startTime := time.Now()

	// Initialize results
	t.results.ResponseTimes = make([]domain.ResponseTimeEntry, 0)
	t.results.Errors = make([]domain.ErrorInfo, 0)
	t.results.SlowRequests = make([]domain.SlowRequest, 0)

	var wg sync.WaitGroup
	var aggregatorWg sync.WaitGroup

	// Start result aggregator
	aggregatorWg.Add(1)
	go t.aggregator(&aggregatorWg)

	// Start workers
	for i := 0; i < t.config.Concurrency; i++ {
		wg.Add(1)
		go t.worker(ctx, &wg)
	}

	// Start URL discovery with the base URL
	t.crawler.AddURL(t.config.BaseURL, 0, t.urlQueue)
	t.results.URLsDiscovered = t.crawler.GetDiscoveredCount()

	// Start monitoring
	go t.monitor(ctx)

	// Wait for context cancellation or completion
	<-ctx.Done()

	// Close URL queue and wait for workers to finish
	close(t.urlQueue)
	wg.Wait()

	// Close result channels and wait for aggregator to finish
	close(t.validationsCh)
	close(t.errorsCh)
	close(t.responseTimesCh)
	close(t.slowRequestsCh)
	aggregatorWg.Wait()

	// Calculate final results
	t.calculateResults(time.Since(startTime))

	return t.results, nil
}

// aggregator collects results from workers via channels (lock-free)
func (t *Tester) aggregator(wg *sync.WaitGroup) {
	defer wg.Done()

	// Track which channels are still open
	validationsClosed := false
	errorsClosed := false
	responseTimesClosed := false
	slowRequestsClosed := false

	for {
		// Exit when all channels are closed
		if validationsClosed && errorsClosed && responseTimesClosed && slowRequestsClosed {
			return
		}

		select {
		case validation, ok := <-t.validationsCh:
			if !ok {
				validationsClosed = true
				continue
			}
			t.results.URLValidations = append(t.results.URLValidations, validation)

		case errInfo, ok := <-t.errorsCh:
			if !ok {
				errorsClosed = true
				continue
			}
			t.results.Errors = append(t.results.Errors, errInfo)

		case responseTime, ok := <-t.responseTimesCh:
			if !ok {
				responseTimesClosed = true
				continue
			}
			t.results.ResponseTimes = append(t.results.ResponseTimes, responseTime)

		case slowReq, ok := <-t.slowRequestsCh:
			if !ok {
				slowRequestsClosed = true
				continue
			}
			t.results.SlowRequests = append(t.results.SlowRequests, slowReq)
		}
	}
}

// worker processes URLs from the queue
func (t *Tester) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case task, ok := <-t.urlQueue:
			if !ok {
				return
			}
			t.processURL(ctx, task)
		case <-ctx.Done():
			return
		}
	}
}

// processDryRun handles URL discovery in dry-run mode (no actual HTTP requests)
func (t *Tester) processDryRun(task domain.URLTask) {
	atomic.AddInt64(&t.results.TotalRequests, 1)

	// In dry-run mode, we mark all URLs as "discovered"
	validation := domain.URLValidation{
		URL:        task.URL,
		StatusCode: 0, // No actual request made
		Depth:      task.Depth,
		IsValid:    false, // Unknown in dry-run
	}

	t.addValidation(validation)

	t.logger.Info("URL discovered (dry-run)",
		"url", util.SanitizeURLDefault(task.URL),
		"depth", task.Depth)
}

// processURL performs a single URL request and records results
func (t *Tester) processURL(ctx context.Context, task domain.URLTask) {
	// Check robots.txt compliance (unless ignoring)
	if !t.config.IgnoreRobots && !t.robotsParser.IsAllowed(task.URL) {
		t.logger.Debug("URL blocked by robots.txt", "url", util.SanitizeURLDefault(task.URL))
		// Record as skipped, not as an error
		atomic.AddInt64(&t.results.TotalRequests, 1)
		return
	}

	// In dry-run mode, just record the URL without making requests
	if t.config.DryRun {
		t.processDryRun(task)
		return
	}

	// Apply rate limiting using goflow's token bucket
	if t.rateLimiter != nil {
		if err := t.rateLimiter.Wait(ctx); err != nil {
			// Context was canceled or deadline exceeded
			t.recordError(task.URL, fmt.Sprintf("rate limiter wait canceled: %v", err), task.Depth)
			atomic.AddInt64(&t.results.FailedRequests, 1)
			return
		}
	}

	atomic.AddInt64(&t.results.TotalRequests, 1)

	// Make HTTP request with 429 retry logic
	resp, responseTime, err := t.makeHTTPRequestWithRetry(ctx, task.URL)
	if err != nil {
		t.recordError(task.URL, fmt.Sprintf("making request: %v", err), task.Depth)
		atomic.AddInt64(&t.results.FailedRequests, 1)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	atomic.AddInt64(&t.results.SuccessfulRequests, 1)

	// Record response time
	t.recordResponseTime(task.URL, responseTime)

	// Create validation record
	validation := domain.URLValidation{
		URL:           task.URL,
		StatusCode:    resp.StatusCode,
		ResponseTime:  responseTime,
		ContentLength: resp.ContentLength,
		ContentType:   resp.Header.Get("Content-Type"),
		Depth:         task.Depth,
		IsValid:       resp.StatusCode >= 200 && resp.StatusCode < 400,
	}

	// Discover links if configured
	validation.LinksFound = t.discoverLinksFromResponse(resp, task)

	// Record slow requests (>2 seconds)
	if responseTime > 2*time.Second {
		t.recordSlowRequest(task.URL, responseTime, resp.StatusCode)
	}

	// Add validation to results (thread-safe)
	t.addValidation(validation)

	t.logger.Debug("URL processed",
		"url", util.SanitizeURLDefault(task.URL),
		"status", resp.StatusCode,
		"response_time", responseTime,
		"depth", task.Depth,
		"links_found", validation.LinksFound)
}

// makeHTTPRequestWithRetry wraps makeHTTPRequest with exponential backoff retry for 429 responses
func (t *Tester) makeHTTPRequestWithRetry(ctx context.Context, url string) (*http.Response, time.Duration, error) {
	const (
		maxRetries    = 4              // Max retry attempts for 429
		initialBackoff = 1 * time.Second
		maxBackoff    = 30 * time.Second
	)

	var totalDuration time.Duration
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, duration, err := t.makeHTTPRequest(ctx, url)
		totalDuration += duration

		// If request failed (network error, etc), return error immediately
		if err != nil {
			return nil, totalDuration, err
		}

		// If not 429 or Respect429 is disabled, return response
		if resp.StatusCode != http.StatusTooManyRequests || !t.config.Respect429 {
			return resp, totalDuration, nil
		}

		// Close the 429 response body before retrying
		_ = resp.Body.Close()

		// If this was the last attempt, return the 429 response
		if attempt == maxRetries {
			// Re-make request one final time to return a valid response object
			return t.makeHTTPRequest(ctx, url)
		}

		// Log the backoff
		t.logger.Info("Received 429 Too Many Requests, backing off",
			"url", util.SanitizeURLDefault(url),
			"attempt", attempt+1,
			"backoff", backoff,
			"max_retries", maxRetries)

		// Wait for backoff period or context cancellation
		select {
		case <-time.After(backoff):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, totalDuration, ctx.Err()
		}

		// Exponential backoff: double each time, cap at maxBackoff
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	// Should never reach here, but return error just in case
	return nil, totalDuration, fmt.Errorf("exceeded max retries for 429")
}

// makeHTTPRequest creates and executes an HTTP request, returning the response and duration
func (t *Tester) makeHTTPRequest(ctx context.Context, url string) (*http.Response, time.Duration, error) {
	startTime := time.Now()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", t.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Apply authentication
	if err := t.applyAuthentication(req); err != nil {
		return nil, 0, fmt.Errorf("applying authentication: %w", err)
	}

	// Execute request
	resp, err := t.client.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		return nil, responseTime, err
	}

	return resp, responseTime, nil
}

// applyAuthentication applies configured authentication to the HTTP request
func (t *Tester) applyAuthentication(req *http.Request) error {
	if t.config.Auth == nil {
		return nil
	}

	auth := t.config.Auth

	switch auth.Type {
	case "basic":
		// Basic authentication: username:password
		if auth.Username != "" {
			req.SetBasicAuth(auth.Username, auth.Password)
			t.logger.Debug("Applied basic authentication", "username", auth.Username)
		}

	case "bearer":
		// Bearer token authentication
		if auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+auth.Token)
			t.logger.Debug("Applied bearer token authentication")
		}

	case "cookie":
		// Cookie-based authentication
		for name, value := range auth.Cookies {
			req.AddCookie(&http.Cookie{
				Name:  name,
				Value: value,
			})
			t.logger.Debug("Applied cookie", "name", name)
		}

	case "header":
		// Custom header-based authentication
		for name, value := range auth.Headers {
			req.Header.Set(name, value)
			t.logger.Debug("Applied custom header", "name", name)
		}

	case "":
		// No authentication type specified, check for individual fields
		if auth.Username != "" {
			req.SetBasicAuth(auth.Username, auth.Password)
		} else if auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+auth.Token)
		}
		if len(auth.Cookies) > 0 {
			for name, value := range auth.Cookies {
				req.AddCookie(&http.Cookie{
					Name:  name,
					Value: value,
				})
			}
		}
		if len(auth.Headers) > 0 {
			for name, value := range auth.Headers {
				req.Header.Set(name, value)
			}
		}

	default:
		return fmt.Errorf("unknown authentication type: %s", auth.Type)
	}

	return nil
}

// discoverLinksFromResponse extracts links from HTML responses and adds them to the crawl queue
func (t *Tester) discoverLinksFromResponse(resp *http.Response, task domain.URLTask) int {
	// Only process HTML responses
	if !t.config.FollowLinks || task.Depth >= t.config.MaxDepth ||
		!strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return 0
	}

	// Limit body reading to 64KB for link extraction
	limitedReader := io.LimitReader(resp.Body, 64*1024)
	body, readErr := io.ReadAll(limitedReader)
	if readErr != nil && readErr != io.EOF {
		t.logger.Debug("Error reading response body for link extraction",
			"url", util.SanitizeURLDefault(task.URL),
			"error", readErr)
		return 0
	}

	// Extract and queue links
	links := t.crawler.ExtractLinks(string(body))
	for _, link := range links {
		if t.crawler.AddURL(link, task.Depth+1, t.urlQueue) {
			t.results.URLsDiscovered = t.crawler.GetDiscoveredCount()
		}
	}

	return len(links)
}

// recordError records an error encountered during testing
func (t *Tester) recordError(url, errMsg string, depth int) {
	errorInfo := domain.ErrorInfo{
		URL:       url,
		Error:     errMsg,
		Timestamp: time.Now(),
		Depth:     depth,
	}
	t.addError(errorInfo)
}

// recordResponseTime records a response time measurement
func (t *Tester) recordResponseTime(url string, responseTime time.Duration) {
	entry := domain.ResponseTimeEntry{
		URL:          url,
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
	}
	t.addResponseTime(entry)
}

// recordSlowRequest records a slow request
func (t *Tester) recordSlowRequest(url string, responseTime time.Duration, statusCode int) {
	slowReq := domain.SlowRequest{
		URL:          url,
		ResponseTime: responseTime,
		StatusCode:   statusCode,
	}
	t.addSlowRequest(slowReq)
}

// Lock-free methods for sending results to aggregator via channels
func (t *Tester) addValidation(validation domain.URLValidation) {
	t.validationsCh <- validation
}

func (t *Tester) addError(errInfo domain.ErrorInfo) {
	t.errorsCh <- errInfo
}

func (t *Tester) addResponseTime(entry domain.ResponseTimeEntry) {
	t.responseTimesCh <- entry
}

func (t *Tester) addSlowRequest(req domain.SlowRequest) {
	t.slowRequestsCh <- req
}

// monitor provides real-time progress updates
func (t *Tester) monitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			total := atomic.LoadInt64(&t.results.TotalRequests)
			successful := atomic.LoadInt64(&t.results.SuccessfulRequests)
			failed := atomic.LoadInt64(&t.results.FailedRequests)
			discovered := t.results.URLsDiscovered

			t.logger.Info("Progress update",
				"total_requests", total,
				"successful_requests", successful,
				"failed_requests", failed,
				"urls_discovered", discovered,
				"queue_size", len(t.urlQueue))
		case <-ctx.Done():
			return
		}
	}
}

// calculateResults computes final statistics
// Note: Safe to access results directly since aggregator has finished
func (t *Tester) calculateResults(duration time.Duration) {
	t.results.Duration = duration.String()

	// Calculate response time statistics
	responseTimes := make([]time.Duration, len(t.results.ResponseTimes))
	for i, entry := range t.results.ResponseTimes {
		responseTimes[i] = entry.ResponseTime
	}

	if len(responseTimes) > 0 {
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})

		t.results.MinResponseTime = responseTimes[0].String()
		t.results.MaxResponseTime = responseTimes[len(responseTimes)-1].String()

		// Calculate average
		var total time.Duration
		for _, rt := range responseTimes {
			total += rt
		}
		t.results.AverageResponseTime = (total / time.Duration(len(responseTimes))).String()
	}

	// Calculate rates
	if duration.Seconds() > 0 {
		t.results.RequestsPerSecond = float64(t.results.TotalRequests) / duration.Seconds()
	}

	if t.results.TotalRequests > 0 {
		t.results.SuccessRate = (float64(t.results.SuccessfulRequests) / float64(t.results.TotalRequests)) * 100
	}

	// Sort slow requests by response time
	sort.Slice(t.results.SlowRequests, func(i, j int) bool {
		return t.results.SlowRequests[i].ResponseTime > t.results.SlowRequests[j].ResponseTime
	})
}
