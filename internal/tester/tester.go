// Package tester implements the core load testing engine with concurrent workers.
package tester

import (
	"context"
	"fmt"
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
)

// Tester orchestrates the stress testing process
type Tester struct {
	config      domain.TesterConfig
	client      *http.Client
	urlQueue    chan domain.URLTask
	results     *domain.TestResults
	rateLimiter bucket.Limiter
	crawler     *crawler.Crawler
	logger      *slog.Logger

	// Thread-safe mutexes for result aggregation
	validationsMu   sync.Mutex
	errorsMu        sync.Mutex
	responseTimesMu sync.Mutex
	slowRequestsMu  sync.Mutex
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

	return &Tester{
		config:      config,
		client:      &http.Client{Timeout: config.RequestTimeout},
		urlQueue:    make(chan domain.URLTask, 10000),
		results:     &domain.TestResults{URLValidations: make([]domain.URLValidation, 0)},
		rateLimiter: rateLimiter,
		crawler:     crawlerInstance,
		logger:      logger,
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

	// Calculate final results
	t.calculateResults(time.Since(startTime))

	return t.results, nil
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

// processURL performs a single URL request and records results
func (t *Tester) processURL(ctx context.Context, task domain.URLTask) { //nolint:gocyclo // Complex but cohesive request handling logic
	// Apply rate limiting using goflow's token bucket
	if t.rateLimiter != nil {
		if err := t.rateLimiter.Wait(ctx); err != nil {
			// Context was canceled or deadline exceeded
			t.recordError(task.URL, fmt.Sprintf("rate limiter wait canceled: %v", err), task.Depth)
			atomic.AddInt64(&t.results.FailedRequests, 1)
			return
		}
	}

	startTime := time.Now()
	atomic.AddInt64(&t.results.TotalRequests, 1)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", task.URL, http.NoBody)
	if err != nil {
		t.recordError(task.URL, fmt.Sprintf("creating request: %v", err), task.Depth)
		atomic.AddInt64(&t.results.FailedRequests, 1)
		return
	}

	// Set user agent
	req.Header.Set("User-Agent", t.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Make request
	resp, err := t.client.Do(req)
	if err != nil {
		t.recordError(task.URL, fmt.Sprintf("making request: %v", err), task.Depth)
		atomic.AddInt64(&t.results.FailedRequests, 1)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseTime := time.Since(startTime)
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

	// Process response body for link discovery
	if t.config.FollowLinks && task.Depth < t.config.MaxDepth &&
		strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		body := make([]byte, 0, 64*1024) // Limit to 64KB for link extraction
		buffer := make([]byte, 4096)
		totalRead := 0

		for totalRead < 64*1024 {
			n, readErr := resp.Body.Read(buffer)
			if n > 0 {
				if totalRead+n > 64*1024 {
					body = append(body, buffer[:64*1024-totalRead]...)
					break
				}
				body = append(body, buffer[:n]...)
				totalRead += n
			}
			if readErr != nil {
				break
			}
		}

		links := t.crawler.ExtractLinks(string(body))
		validation.LinksFound = len(links)

		// Add new URLs to queue
		for _, link := range links {
			if t.crawler.AddURL(link, task.Depth+1, t.urlQueue) {
				t.results.URLsDiscovered = t.crawler.GetDiscoveredCount()
			}
		}
	}

	// Record slow requests (>2 seconds)
	if responseTime > 2*time.Second {
		t.recordSlowRequest(task.URL, responseTime, resp.StatusCode)
	}

	// Add validation to results (thread-safe)
	t.addValidation(validation)

	t.logger.Debug("URL processed",
		"url", task.URL,
		"status", resp.StatusCode,
		"response_time", responseTime,
		"depth", task.Depth,
		"links_found", validation.LinksFound)
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

// Thread-safe methods for adding to slices
func (t *Tester) addValidation(validation domain.URLValidation) { //nolint:gocritic // Passing by value is acceptable for this use case
	t.validationsMu.Lock()
	defer t.validationsMu.Unlock()
	t.results.URLValidations = append(t.results.URLValidations, validation)
}

func (t *Tester) addError(errInfo domain.ErrorInfo) {
	t.errorsMu.Lock()
	defer t.errorsMu.Unlock()
	t.results.Errors = append(t.results.Errors, errInfo)
}

func (t *Tester) addResponseTime(entry domain.ResponseTimeEntry) {
	t.responseTimesMu.Lock()
	defer t.responseTimesMu.Unlock()
	t.results.ResponseTimes = append(t.results.ResponseTimes, entry)
}

func (t *Tester) addSlowRequest(req domain.SlowRequest) {
	t.slowRequestsMu.Lock()
	defer t.slowRequestsMu.Unlock()
	t.results.SlowRequests = append(t.results.SlowRequests, req)
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
func (t *Tester) calculateResults(duration time.Duration) {
	t.results.Duration = duration.String()

	// Calculate response time statistics
	t.responseTimesMu.Lock()
	responseTimes := make([]time.Duration, len(t.results.ResponseTimes))
	for i, entry := range t.results.ResponseTimes {
		responseTimes[i] = entry.ResponseTime
	}
	t.responseTimesMu.Unlock()

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
	t.slowRequestsMu.Lock()
	sort.Slice(t.results.SlowRequests, func(i, j int) bool {
		return t.results.SlowRequests[i].ResponseTime > t.results.SlowRequests[j].ResponseTime
	})
	t.slowRequestsMu.Unlock()
}
