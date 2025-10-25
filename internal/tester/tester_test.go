package tester

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vnykmshr/lobster/internal/domain"
)

// Test helper to create a test logger
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))
}

// Test helper to create default test config
func testConfig(baseURL string) domain.TesterConfig {
	return domain.TesterConfig{
		BaseURL:        baseURL,
		Concurrency:    2,
		RequestTimeout: 5 * time.Second,
		UserAgent:      "TestAgent/1.0",
		FollowLinks:    false,
		MaxDepth:       1,
		Rate:           0, // No rate limiting for faster tests
		IgnoreRobots:   true, // Skip robots.txt in tests
	}
}

func TestNew_Success(t *testing.T) {
	config := testConfig("http://example.com")
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tester == nil {
		t.Fatal("Expected tester to be created")
	}

	if tester.config.BaseURL != "http://example.com" {
		t.Errorf("Expected BaseURL 'http://example.com', got '%s'", tester.config.BaseURL)
	}

	if tester.client == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if tester.urlQueue == nil {
		t.Error("Expected URL queue to be initialized")
	}

	if tester.validationsCh == nil {
		t.Error("Expected validations channel to be initialized")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	// Use completely invalid URL (no protocol)
	config := testConfig("://invalid")
	logger := testLogger()

	_, err := New(config, logger)
	if err == nil {
		t.Fatal("Expected error for invalid URL, got none")
	}

	if !strings.Contains(err.Error(), "creating crawler") {
		t.Errorf("Expected crawler error, got: %v", err)
	}
}

func TestNew_WithRateLimiter(t *testing.T) {
	config := testConfig("http://example.com")
	config.Rate = 10.0 // 10 req/s
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tester.rateLimiter == nil {
		t.Error("Expected rate limiter to be created for rate > 0")
	}
}

func TestNew_ConfigurableQueueSize(t *testing.T) {
	tests := []struct {
		name              string
		queueSize         int
		expectedCapacity  int
		description       string
	}{
		{
			name:             "Custom queue size",
			queueSize:        5000,
			expectedCapacity: 5000,
			description:      "Should use configured queue size",
		},
		{
			name:             "Default when zero",
			queueSize:        0,
			expectedCapacity: 10000,
			description:      "Should default to 10000 when QueueSize is 0",
		},
		{
			name:             "Default when negative",
			queueSize:        -100,
			expectedCapacity: 10000,
			description:      "Should default to 10000 when QueueSize is negative",
		},
		{
			name:             "Small queue size",
			queueSize:        100,
			expectedCapacity: 100,
			description:      "Should allow small queue sizes",
		},
		{
			name:             "Large queue size",
			queueSize:        50000,
			expectedCapacity: 50000,
			description:      "Should allow large queue sizes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testConfig("http://example.com")
			config.QueueSize = tt.queueSize
			logger := testLogger()

			tester, err := New(config, logger)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			// Verify queue capacity by checking we can send expectedCapacity items
			actualCapacity := cap(tester.urlQueue)
			if actualCapacity != tt.expectedCapacity {
				t.Errorf("%s: expected capacity %d, got %d",
					tt.description, tt.expectedCapacity, actualCapacity)
			}
		})
	}
}

func TestMakeHTTPRequest_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("User-Agent") != "TestAgent/1.0" {
			t.Errorf("Expected User-Agent 'TestAgent/1.0', got '%s'", r.Header.Get("User-Agent"))
		}

		if r.Header.Get("Accept") == "" {
			t.Error("Expected Accept header to be set")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	logger := testLogger()
	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx := context.Background()
	resp, duration, err := tester.makeHTTPRequest(ctx, server.URL)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response to be non-nil")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if duration <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestMakeHTTPRequest_ContextCanceled(t *testing.T) {
	// Create server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
	}))
	defer server.Close()

	config := testConfig(server.URL)
	logger := testLogger()
	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err = tester.makeHTTPRequest(ctx, server.URL)

	if err == nil {
		t.Fatal("Expected error due to context timeout")
	}
}

func TestDiscoverLinksFromResponse_HTML(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<a href="/page1">Link 1</a>
			<a href="/page2">Link 2</a>
			<a href="http://example.com/page3">Link 3</a>
		</body>
		</html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.FollowLinks = true
	config.MaxDepth = 5
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	// Make request
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get test page: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	task := domain.URLTask{URL: server.URL, Depth: 0}
	linksFound := tester.discoverLinksFromResponse(resp, task)

	if linksFound == 0 {
		t.Error("Expected to find links in HTML response")
	}

	if linksFound < 2 {
		t.Errorf("Expected at least 2 links, got %d", linksFound)
	}
}

func TestDiscoverLinksFromResponse_NotHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key": "value"}`))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.FollowLinks = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get test page: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	task := domain.URLTask{URL: server.URL, Depth: 0}
	linksFound := tester.discoverLinksFromResponse(resp, task)

	if linksFound != 0 {
		t.Errorf("Expected 0 links from non-HTML response, got %d", linksFound)
	}
}

func TestDiscoverLinksFromResponse_MaxDepthReached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<a href="/page">Link</a>`))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.FollowLinks = true
	config.MaxDepth = 2
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get test page: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Task at max depth
	task := domain.URLTask{URL: server.URL, Depth: 2}
	linksFound := tester.discoverLinksFromResponse(resp, task)

	if linksFound != 0 {
		t.Errorf("Expected 0 links when max depth reached, got %d", linksFound)
	}
}

func TestDiscoverLinksFromResponse_FollowLinksDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<a href="/page">Link</a>`))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.FollowLinks = false // Disabled
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get test page: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	task := domain.URLTask{URL: server.URL, Depth: 0}
	linksFound := tester.discoverLinksFromResponse(resp, task)

	if linksFound != 0 {
		t.Errorf("Expected 0 links when FollowLinks disabled, got %d", linksFound)
	}
}

func TestRun_BasicWorkflow(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.Concurrency = 2
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	// Run for very short duration
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	results, err := tester.Run(ctx)

	if err != nil {
		t.Fatalf("Expected no error from Run, got: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results to be non-nil")
	}

	// Should have made at least 1 request (the base URL)
	if results.TotalRequests < 1 {
		t.Errorf("Expected at least 1 request, got %d", results.TotalRequests)
	}

	// Should have successful requests
	if results.SuccessfulRequests < 1 {
		t.Errorf("Expected at least 1 successful request, got %d", results.SuccessfulRequests)
	}

	// Duration should be populated
	if results.Duration == "" {
		t.Error("Expected duration to be populated")
	}

	// Should have URL validations
	if len(results.URLValidations) < 1 {
		t.Error("Expected at least one URL validation")
	}
}

func TestRun_ErrorHandling(t *testing.T) {
	// Server that always returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := testConfig(server.URL)
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	results, err := tester.Run(ctx)

	if err != nil {
		t.Fatalf("Expected no error from Run, got: %v", err)
	}

	// Should still have results even with errors
	if results.TotalRequests < 1 {
		t.Error("Expected requests to be made")
	}

	// URLValidations should show invalid status
	if len(results.URLValidations) > 0 {
		validation := results.URLValidations[0]
		if validation.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", validation.StatusCode)
		}
		if validation.IsValid {
			t.Error("Expected validation.IsValid to be false for 500 status")
		}
	}
}

// TestRun_SlowRequests tests that slow requests (>2s) are properly recorded.
// This test is skipped in short mode due to runtime requirements.
func TestRun_SlowRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow test in short mode")
	}

	// Create a test server that responds slowly
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2100 * time.Millisecond) // Just over 2 seconds
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>Slow response</body></html>"))
	}))
	defer slowServer.Close()

	config := testConfig(slowServer.URL)
	config.MaxDepth = 0 // Don't crawl, just test the seed URL
	config.NoProgress = true // Disable progress output in tests
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	// Use timeout to ensure test doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify the slow request was recorded
	if len(results.SlowRequests) != 1 {
		t.Errorf("Expected 1 slow request, got %d", len(results.SlowRequests))
	}

	if len(results.SlowRequests) > 0 {
		if results.SlowRequests[0].ResponseTime < 2*time.Second {
			t.Errorf("Expected response time >= 2s, got %v", results.SlowRequests[0].ResponseTime)
		}
	}
}

func TestCalculateResults(t *testing.T) {
	config := testConfig("http://example.com")
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	// Populate test data
	tester.results.ResponseTimes = []domain.ResponseTimeEntry{
		{ResponseTime: 100 * time.Millisecond},
		{ResponseTime: 200 * time.Millisecond},
		{ResponseTime: 300 * time.Millisecond},
	}
	tester.results.TotalRequests = 10
	tester.results.SuccessfulRequests = 8
	tester.results.FailedRequests = 2

	tester.results.SlowRequests = []domain.SlowRequest{
		{ResponseTime: 5 * time.Second},
		{ResponseTime: 3 * time.Second},
	}

	// Calculate results
	tester.calculateResults(2 * time.Second)

	// Check duration
	if tester.results.Duration == "" {
		t.Error("Expected duration to be set")
	}

	// Check response time stats
	if tester.results.MinResponseTime != "100ms" {
		t.Errorf("Expected min 100ms, got %s", tester.results.MinResponseTime)
	}

	if tester.results.MaxResponseTime != "300ms" {
		t.Errorf("Expected max 300ms, got %s", tester.results.MaxResponseTime)
	}

	if tester.results.AverageResponseTime != "200ms" {
		t.Errorf("Expected avg 200ms, got %s", tester.results.AverageResponseTime)
	}

	// Check rates
	if tester.results.RequestsPerSecond != 5.0 {
		t.Errorf("Expected 5 req/s, got %.1f", tester.results.RequestsPerSecond)
	}

	if tester.results.SuccessRate != 80.0 {
		t.Errorf("Expected 80%% success rate, got %.1f", tester.results.SuccessRate)
	}

	// Check slow requests are sorted (descending)
	if len(tester.results.SlowRequests) >= 2 {
		if tester.results.SlowRequests[0].ResponseTime < tester.results.SlowRequests[1].ResponseTime {
			t.Error("Expected slow requests to be sorted in descending order")
		}
	}
}

func TestAggregator_ChannelCollection(t *testing.T) {
	config := testConfig("http://example.com")
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	// Initialize results
	tester.results.ResponseTimes = make([]domain.ResponseTimeEntry, 0)
	tester.results.Errors = make([]domain.ErrorInfo, 0)
	tester.results.SlowRequests = make([]domain.SlowRequest, 0)
	tester.results.URLValidations = make([]domain.URLValidation, 0)

	// Start aggregator
	var wg sync.WaitGroup
	wg.Add(1)
	go tester.aggregator(&wg)

	// Send test data
	validation := domain.URLValidation{URL: "http://test.com", StatusCode: 200}
	tester.validationsCh <- validation

	errInfo := domain.ErrorInfo{URL: "http://error.com", Error: "test error"}
	tester.errorsCh <- errInfo

	responseTime := domain.ResponseTimeEntry{URL: "http://test.com", ResponseTime: 100 * time.Millisecond}
	tester.responseTimesCh <- responseTime

	slowReq := domain.SlowRequest{URL: "http://slow.com", ResponseTime: 3 * time.Second}
	tester.slowRequestsCh <- slowReq

	// Close channels to signal completion and wait for aggregator to finish
	// The aggregator processes all channel data until channels are closed
	close(tester.validationsCh)
	close(tester.errorsCh)
	close(tester.responseTimesCh)
	close(tester.slowRequestsCh)
	wg.Wait()

	// Verify results were collected
	if len(tester.results.URLValidations) != 1 {
		t.Errorf("Expected 1 validation, got %d", len(tester.results.URLValidations))
	}

	if len(tester.results.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(tester.results.Errors))
	}

	if len(tester.results.ResponseTimes) != 1 {
		t.Errorf("Expected 1 response time, got %d", len(tester.results.ResponseTimes))
	}

	if len(tester.results.SlowRequests) != 1 {
		t.Errorf("Expected 1 slow request, got %d", len(tester.results.SlowRequests))
	}
}

func TestRecordError(t *testing.T) {
	config := testConfig("http://example.com")
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	tester.results.Errors = make([]domain.ErrorInfo, 0)

	// Start aggregator
	var wg sync.WaitGroup
	wg.Add(1)
	go tester.aggregator(&wg)

	// Record error
	tester.recordError("http://test.com", "test error message", 1)

	// Close channels to signal completion and wait for aggregator to finish
	close(tester.validationsCh)
	close(tester.errorsCh)
	close(tester.responseTimesCh)
	close(tester.slowRequestsCh)
	wg.Wait()

	// Verify error was recorded
	if len(tester.results.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(tester.results.Errors))
	}

	errorInfo := tester.results.Errors[0]
	if errorInfo.URL != "http://test.com" {
		t.Errorf("Expected URL 'http://test.com', got '%s'", errorInfo.URL)
	}

	if errorInfo.Error != "test error message" {
		t.Errorf("Expected error 'test error message', got '%s'", errorInfo.Error)
	}

	if errorInfo.Depth != 1 {
		t.Errorf("Expected depth 1, got %d", errorInfo.Depth)
	}
}

func TestRun_ConcurrentWorkers(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// Small delay to ensure concurrent execution
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.Concurrency = 5 // Multiple workers
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)

	if err != nil {
		t.Fatalf("Expected no error from Run, got: %v", err)
	}

	// With 5 workers and 1 second, should have made multiple requests
	if results.TotalRequests < 1 {
		t.Error("Expected concurrent workers to process requests")
	}

	// Verify no race conditions (tests run with -race flag)
	if len(results.URLValidations) != int(results.TotalRequests) {
		t.Errorf("Expected %d validations, got %d", results.TotalRequests, len(results.URLValidations))
	}
}

// TestRun_WithRateLimiting tests that rate limiting properly throttles requests.
// This test is skipped in short mode due to runtime requirements.
func TestRun_WithRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow test in short mode")
	}

	requestTimes := []time.Time{}
	var mu sync.Mutex

	// Create a test server that records request timestamps and returns links
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
		// Return HTML with links to create multiple requests
		if r.URL.Path == "" || r.URL.Path == "/" {
			_, _ = w.Write([]byte(`<html><body>
				<a href="/page1">Page 1</a>
				<a href="/page2">Page 2</a>
			</body></html>`))
		} else {
			_, _ = w.Write([]byte("<html><body>Subpage</body></html>"))
		}
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.MaxDepth = 1 // Allow crawling to discover links
	config.FollowLinks = true // Must follow links to make multiple requests
	config.Concurrency = 1 // Single worker to test rate limiting properly
	config.Rate = 1.0 // 1 request per second = 1000ms between requests
	config.NoProgress = true // Disable progress output in tests
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	// Use timeout to ensure test doesn't hang
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	_, err = tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify rate limiting worked by checking time between requests
	mu.Lock()
	defer mu.Unlock()

	if len(requestTimes) < 2 {
		t.Fatalf("Expected at least 2 requests, got %d", len(requestTimes))
	}

	// Check that at least some requests are rate-limited
	// With burst capacity, first few may be fast, but subsequent should be throttled
	// Look for at least one gap >= 800ms (allowing 200ms tolerance for 1 req/s)
	hasThrottledRequest := false
	for i := 1; i < len(requestTimes); i++ {
		timeBetween := requestTimes[i].Sub(requestTimes[i-1])
		if timeBetween >= 800*time.Millisecond {
			hasThrottledRequest = true
			break
		}
	}

	if !hasThrottledRequest {
		t.Errorf("No throttled requests found. Request times: %v. Expected at least one gap >= 800ms", requestTimes)
	}
}

func TestApplyAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		authConfig     *domain.AuthConfig
		wantErr        bool
		errContains    string
		checkAuth      func(t *testing.T, req *http.Request)
	}{
		{
			name: "Basic Auth",
			authConfig: &domain.AuthConfig{
				Type:     "basic",
				Username: "testuser",
				Password: "testpass",
			},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				auth := req.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Basic ") {
					t.Errorf("Expected Basic auth header, got: %s", auth)
				}
			},
		},
		{
			name: "Bearer Token",
			authConfig: &domain.AuthConfig{
				Type:  "bearer",
				Token: "test-token-123",
			},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				auth := req.Header.Get("Authorization")
				expected := "Bearer test-token-123"
				if auth != expected {
					t.Errorf("Expected '%s', got '%s'", expected, auth)
				}
			},
		},
		{
			name: "Cookie Auth",
			authConfig: &domain.AuthConfig{
				Type: "cookie",
				Cookies: map[string]string{
					"session": "abc123",
					"csrf":    "xyz789",
				},
			},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				cookies := req.Cookies()
				if len(cookies) != 2 {
					t.Fatalf("Expected 2 cookies, got %d", len(cookies))
				}
				cookieMap := make(map[string]string)
				for _, cookie := range cookies {
					cookieMap[cookie.Name] = cookie.Value
				}
				if cookieMap["session"] != "abc123" {
					t.Errorf("Expected session cookie 'abc123', got '%s'", cookieMap["session"])
				}
				if cookieMap["csrf"] != "xyz789" {
					t.Errorf("Expected csrf cookie 'xyz789', got '%s'", cookieMap["csrf"])
				}
			},
		},
		{
			name: "Custom Headers",
			authConfig: &domain.AuthConfig{
				Type: "header",
				Headers: map[string]string{
					"X-API-Key":     "secret-key",
					"X-Custom-Auth": "custom-value",
				},
			},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				if req.Header.Get("X-API-Key") != "secret-key" {
					t.Errorf("Expected X-API-Key 'secret-key', got '%s'", req.Header.Get("X-API-Key"))
				}
				if req.Header.Get("X-Custom-Auth") != "custom-value" {
					t.Errorf("Expected X-Custom-Auth 'custom-value', got '%s'", req.Header.Get("X-Custom-Auth"))
				}
			},
		},
		{
			name:       "No Auth",
			authConfig: nil,
			wantErr:    false,
			checkAuth: func(t *testing.T, req *http.Request) {
				if req.Header.Get("Authorization") != "" {
					t.Error("Expected no Authorization header")
				}
			},
		},
		{
			name: "Auto-detect Basic",
			authConfig: &domain.AuthConfig{
				Type:     "", // Empty type, should auto-detect
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				auth := req.Header.Get("Authorization")
				if !strings.HasPrefix(auth, "Basic ") {
					t.Errorf("Expected auto-detected Basic auth, got: %s", auth)
				}
			},
		},
		{
			name: "Invalid Type",
			authConfig: &domain.AuthConfig{
				Type: "invalid-type",
			},
			wantErr:     true,
			errContains: "unknown authentication type",
			checkAuth:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := testConfig("http://example.com")
			config.Auth = tt.authConfig
			logger := testLogger()

			tester, err := New(config, logger)
			if err != nil {
				t.Fatalf("Failed to create tester: %v", err)
			}

			req, err := http.NewRequest("GET", "http://example.com", http.NoBody)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			err = tester.applyAuthentication(req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got none")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if tt.checkAuth != nil {
				tt.checkAuth(t, req)
			}
		})
	}
}

func TestApplyAuthentication_IntegrationWithHTTPRequest(t *testing.T) {
	// Test that authentication is properly applied in an actual HTTP request
	authReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for bearer token
		auth := r.Header.Get("Authorization")
		if auth == "Bearer test-integration-token" {
			authReceived = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.Auth = &domain.AuthConfig{
		Type:  "bearer",
		Token: "test-integration-token",
	}
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Make a request
	resp, _, err := tester.makeHTTPRequest(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to make HTTP request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if !authReceived {
		t.Error("Authentication token was not sent to server")
	}
}

// TestNew_InsecureSkipVerify tests the InsecureSkipVerify configuration
func TestNew_InsecureSkipVerify(t *testing.T) {
	config := testConfig("https://example.com")
	config.InsecureSkipVerify = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify HTTP client has custom transport
	if tester.client.Transport == nil {
		t.Error("Expected custom transport for InsecureSkipVerify")
	}
}

// TestMakeHTTPRequestWithRetry_429Backoff tests HTTP 429 retry with exponential backoff
func TestMakeHTTPRequestWithRetry_429Backoff(t *testing.T) {
	var requestCount int64
	retryTimes := []time.Time{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)
		mu.Lock()
		retryTimes = append(retryTimes, time.Now())
		mu.Unlock()

		// Return 429 for first 2 requests, then 200
		if count <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("Too Many Requests"))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Success"))
		}
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.Respect429 = true // Enable 429 retry
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, _, err := tester.makeHTTPRequestWithRetry(ctx, server.URL)
	if err != nil {
		t.Fatalf("Expected successful retry, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify retries happened
	if atomic.LoadInt64(&requestCount) != 3 {
		t.Errorf("Expected 3 requests (2 retries + 1 success), got %d", requestCount)
	}

	// Verify status code is eventually 200
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 after retry, got %d", resp.StatusCode)
	}

	// Verify exponential backoff timing
	mu.Lock()
	defer mu.Unlock()
	if len(retryTimes) >= 2 {
		// Time between first and second request should be ~1 second (initial backoff)
		timeBetween := retryTimes[1].Sub(retryTimes[0])
		if timeBetween < 900*time.Millisecond {
			t.Errorf("Expected ~1s backoff between first two requests, got %v", timeBetween)
		}
	}
}

// TestMakeHTTPRequestWithRetry_429MaxRetries tests that max retries is enforced
func TestMakeHTTPRequestWithRetry_429MaxRetries(t *testing.T) {
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		// Always return 429
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Too Many Requests"))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.Respect429 = true
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, _, err := tester.makeHTTPRequestWithRetry(ctx, server.URL)
	if err != nil {
		t.Fatalf("Expected response (not error) after max retries, got: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should have made 5 attempts (initial + 4 retries) + 1 final request
	count := atomic.LoadInt64(&requestCount)
	if count != 6 {
		t.Errorf("Expected 6 requests (5 attempts + 1 final), got %d", count)
	}

	// Final response should still be 429
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected final status 429, got %d", resp.StatusCode)
	}
}

// TestMakeHTTPRequestWithRetry_ContextCancellation tests context cancellation during retry backoff
func TestMakeHTTPRequestWithRetry_ContextCancellation(t *testing.T) {
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Too Many Requests"))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.Respect429 = true
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, _, err = tester.makeHTTPRequestWithRetry(ctx, server.URL)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}

	// Should have made at least one request before context cancellation
	count := atomic.LoadInt64(&requestCount)
	if count < 1 {
		t.Errorf("Expected at least 1 request before cancellation, got %d", count)
	}
}

// TestMakeHTTPRequestWithRetry_Respect429Disabled tests that 429 retry is skipped when disabled
func TestMakeHTTPRequestWithRetry_Respect429Disabled(t *testing.T) {
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Too Many Requests"))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.Respect429 = false // Disable 429 retry
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, _, err := tester.makeHTTPRequestWithRetry(ctx, server.URL)
	if err != nil {
		t.Fatalf("Expected response, got error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should only make 1 request (no retries)
	count := atomic.LoadInt64(&requestCount)
	if count != 1 {
		t.Errorf("Expected exactly 1 request with Respect429=false, got %d", count)
	}

	// Status should be 429 (no retry)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", resp.StatusCode)
	}
}
