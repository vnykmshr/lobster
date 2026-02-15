package tester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/1mb-dev/lobster/v2/internal/domain"
)

// Integration tests verify end-to-end workflows and multi-component interactions.
// These tests are skipped in short mode as they require more runtime.
// They are also skipped with race detector as they can timeout under the overhead.

// skipSlowIntegrationTest skips the test in short mode.
// Note: Race detector adds 2-10x overhead. These tests have tight timeouts
// and can fail intermittently under race. Use `go test -short` for quick checks.
func skipSlowIntegrationTest(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

// TestIntegration_FullCrawlWorkflow tests the complete crawl workflow:
// seed URL → discover links → validate all URLs → generate final results
func TestIntegration_FullCrawlWorkflow(t *testing.T) {
	skipSlowIntegrationTest(t)

	// Create a multi-page test site
	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "", "/":
			// Homepage with links to other pages
			_, _ = w.Write([]byte(`<html><body>
				<h1>Homepage</h1>
				<a href="/about">About</a>
				<a href="/contact">Contact</a>
				<a href="/products">Products</a>
			</body></html>`))
		case "/about":
			_, _ = w.Write([]byte(`<html><body>
				<h1>About Page</h1>
				<a href="/team">Team</a>
			</body></html>`))
		case "/contact":
			_, _ = w.Write([]byte(`<html><body><h1>Contact Page</h1></body></html>`))
		case "/products":
			_, _ = w.Write([]byte(`<html><body>
				<h1>Products</h1>
				<a href="/products/widget">Widget</a>
			</body></html>`))
		case "/team":
			_, _ = w.Write([]byte(`<html><body><h1>Team Page</h1></body></html>`))
		case "/products/widget":
			_, _ = w.Write([]byte(`<html><body><h1>Widget Product</h1></body></html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<html><body><h1>404 Not Found</h1></body></html>`))
		}
	}))
	defer server.Close()

	// Configure for full crawl
	config := testConfig(server.URL)
	config.MaxDepth = 2 // Allow 2 levels of crawling
	config.FollowLinks = true
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify crawl results
	if results.URLsDiscovered < 5 {
		t.Errorf("Expected to discover at least 5 URLs, got %d", results.URLsDiscovered)
	}

	if results.TotalRequests < 5 {
		t.Errorf("Expected at least 5 requests, got %d", results.TotalRequests)
	}

	if results.SuccessfulRequests < 5 {
		t.Errorf("Expected at least 5 successful requests, got %d", results.SuccessfulRequests)
	}

	if results.SuccessRate < 80.0 {
		t.Errorf("Expected success rate > 80%%, got %.2f%%", results.SuccessRate)
	}

	// Verify actual HTTP requests were made
	if atomic.LoadInt64(&requestCount) < 5 {
		t.Errorf("Expected at least 5 HTTP requests, got %d", requestCount)
	}

	// Verify URL validations recorded
	if len(results.URLValidations) == 0 {
		t.Error("Expected URL validations to be recorded")
	}

	// Verify response times are populated
	if len(results.ResponseTimes) == 0 {
		t.Error("Expected response times to be recorded")
	}
}

// TestIntegration_DryRunMode verifies dry-run mode discovers links without load testing
func TestIntegration_DryRunMode(t *testing.T) {
	skipSlowIntegrationTest(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "" || r.URL.Path == "/" {
			_, _ = w.Write([]byte(`<html><body>
				<a href="/page1">Page 1</a>
				<a href="/page2">Page 2</a>
				<a href="/page3">Page 3</a>
			</body></html>`))
		} else {
			_, _ = w.Write([]byte(`<html><body>Page content</body></html>`))
		}
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.MaxDepth = 1
	config.FollowLinks = true
	config.DryRun = true
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify URLs were discovered even in dry-run mode
	if results.URLsDiscovered < 3 {
		t.Errorf("Expected to discover at least 3 URLs in dry-run, got %d", results.URLsDiscovered)
	}

	// In dry-run mode, requests are made for discovery but not for load testing
	// So we should see some requests
	if results.TotalRequests == 0 {
		t.Error("Expected some requests in dry-run mode for link discovery")
	}
}

// TestIntegration_AuthenticationWorkflow tests end-to-end authentication
func TestIntegration_AuthenticationWorkflow(t *testing.T) {
	skipSlowIntegrationTest(t)

	// Create a server that requires authentication
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for correct auth header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer secret-token-123" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`<html><body>Unauthorized</body></html>`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>
			<h1>Authenticated Content</h1>
			<a href="/private/data">Private Data</a>
		</body></html>`))
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.MaxDepth = 1
	config.FollowLinks = true
	config.NoProgress = true
	config.Auth = &domain.AuthConfig{
		Type:  "bearer",
		Token: "secret-token-123",
	}
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify authenticated requests succeeded
	if results.SuccessfulRequests == 0 {
		t.Error("Expected successful authenticated requests")
	}

	// Should not have failed requests if auth is working
	if results.FailedRequests > 0 {
		t.Errorf("Expected no failed requests with proper auth, got %d failures", results.FailedRequests)
	}
}

// TestIntegration_RobotsTxtCompliance tests robots.txt enforcement
func TestIntegration_RobotsTxtCompliance(t *testing.T) {
	skipSlowIntegrationTest(t)

	var allowedCount, blockedCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		switch r.URL.Path {
		case "/robots.txt":
			// Disallow /admin paths
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /admin\n"))
		case "", "/":
			atomic.AddInt64(&allowedCount, 1)
			_, _ = w.Write([]byte(`<html><body>
				<a href="/public">Public Page</a>
				<a href="/admin">Admin Page</a>
			</body></html>`))
		case "/public":
			atomic.AddInt64(&allowedCount, 1)
			_, _ = w.Write([]byte(`<html><body>Public content</body></html>`))
		case "/admin":
			// This should not be requested due to robots.txt
			atomic.AddInt64(&blockedCount, 1)
			_, _ = w.Write([]byte(`<html><body>Admin content</body></html>`))
		}
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.MaxDepth = 1
	config.FollowLinks = true
	config.IgnoreRobots = false // Respect robots.txt
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify robots.txt compliance
	if atomic.LoadInt64(&blockedCount) > 0 {
		t.Errorf("Expected /admin to be blocked by robots.txt, but %d requests were made", blockedCount)
	}

	if atomic.LoadInt64(&allowedCount) < 2 {
		t.Errorf("Expected at least 2 allowed requests, got %d", allowedCount)
	}

	// Should have discovered URLs but skipped blocked ones
	if results.URLsDiscovered < 2 {
		t.Errorf("Expected to discover at least 2 URLs, got %d", results.URLsDiscovered)
	}
}

// TestIntegration_ErrorHandlingAndRecovery tests error scenarios
func TestIntegration_ErrorHandlingAndRecovery(t *testing.T) {
	skipSlowIntegrationTest(t)

	var requestNum int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		num := atomic.AddInt64(&requestNum, 1)

		switch r.URL.Path {
		case "", "/":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body>
				<a href="/success">Success Page</a>
				<a href="/error404">404 Page</a>
				<a href="/error500">500 Page</a>
			</body></html>`))
		case "/success":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body>Success!</body></html>`))
		case "/error404":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<html><body>Not Found</body></html>`))
		case "/error500":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`<html><body>Server Error</body></html>`))
		case "/timeout":
			// Simulate slow response
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body>Slow response</body></html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}

		// Simulate occasional server issues
		if num%7 == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.MaxDepth = 1
	config.FollowLinks = true
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify error handling
	if results.TotalRequests == 0 {
		t.Error("Expected some requests to be made")
	}

	// Should have successful requests
	if results.SuccessfulRequests == 0 {
		t.Error("Expected some successful requests")
	}

	// HTTP error codes (404, 500) are successful HTTP requests but invalid validations
	// FailedRequests is only for network errors, timeouts, etc.
	// So we verify invalid validations instead

	// Success rate should be calculated correctly
	expectedRate := float64(results.SuccessfulRequests) / float64(results.TotalRequests) * 100
	if results.SuccessRate != expectedRate {
		t.Errorf("Expected success rate %.2f%%, got %.2f%%", expectedRate, results.SuccessRate)
	}

	// Verify URL validations include both valid and invalid
	hasValid := false
	hasInvalid := false
	for _, v := range results.URLValidations {
		if v.IsValid {
			hasValid = true
		} else {
			hasInvalid = true
		}
	}

	if !hasValid {
		t.Error("Expected at least one valid URL validation")
	}

	if !hasInvalid {
		t.Error("Expected at least one invalid URL validation")
	}
}

// TestIntegration_ConcurrentCrawling tests concurrent worker behavior
func TestIntegration_ConcurrentCrawling(t *testing.T) {
	skipSlowIntegrationTest(t)

	var concurrentRequests int64
	var maxConcurrent int64
	var currentConcurrent int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track concurrent requests
		current := atomic.AddInt64(&currentConcurrent, 1)
		atomic.AddInt64(&concurrentRequests, 1)

		// Update max concurrent
		for {
			currentMax := atomic.LoadInt64(&maxConcurrent)
			if current <= currentMax || atomic.CompareAndSwapInt64(&maxConcurrent, currentMax, current) {
				break
			}
		}

		// Simulate some processing time
		time.Sleep(50 * time.Millisecond)

		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "" || r.URL.Path == "/" {
			// Generate multiple links to test concurrency
			links := []string{}
			for i := 1; i <= 10; i++ {
				links = append(links, `<a href="/page`+string(rune('0'+i))+`">Page</a>`)
			}
			_, _ = w.Write([]byte(`<html><body>` + strings.Join(links, "\n") + `</body></html>`))
		} else {
			_, _ = w.Write([]byte(`<html><body>Page content</body></html>`))
		}

		atomic.AddInt64(&currentConcurrent, -1)
	}))
	defer server.Close()

	config := testConfig(server.URL)
	config.MaxDepth = 1
	config.FollowLinks = true
	config.Concurrency = 3 // Use 3 workers
	config.NoProgress = true
	logger := testLogger()

	tester, err := New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create tester: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := tester.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify concurrent processing occurred
	maxConc := atomic.LoadInt64(&maxConcurrent)
	if maxConc < 2 {
		t.Errorf("Expected concurrent processing (max concurrent: %d), but got %d", config.Concurrency, maxConc)
	}

	// Should not exceed configured concurrency limit significantly
	// Allow some margin due to timing
	if maxConc > int64(config.Concurrency+1) {
		t.Errorf("Max concurrent requests %d exceeded concurrency limit %d", maxConc, config.Concurrency)
	}

	// Verify crawl completed successfully
	if results.TotalRequests < 5 {
		t.Errorf("Expected at least 5 requests with concurrent crawling, got %d", results.TotalRequests)
	}
}
