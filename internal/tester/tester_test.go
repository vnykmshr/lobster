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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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

// TestRun_SlowRequests skipped - takes >2 seconds to run
// Covered by unit test of recordSlowRequest instead

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

	// Give aggregator time to process
	time.Sleep(100 * time.Millisecond)

	// Close channels and wait for aggregator
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

	// Give aggregator time to process
	time.Sleep(50 * time.Millisecond)

	// Close channels
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

// TestRun_WithRateLimiting skipped - takes 2+ seconds to run reliably
// Rate limiter functionality tested in unit test instead
