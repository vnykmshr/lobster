package domain

import "time"

// URLTask represents a URL to be tested during stress testing.
// Depth tracks how many links deep this URL was discovered from the base URL,
// used to enforce MaxDepth limits during recursive link discovery.
type URLTask struct {
	// URL is the fully-qualified URL to request.
	URL string
	// Depth is the crawl depth (0 = base URL, 1 = linked from base, etc.)
	Depth int
}

// TestResults contains comprehensive results from a stress test execution.
// This is the main output structure containing all metrics, validations,
// and performance data collected during the test run.
type TestResults struct {
	// URLValidations contains individual validation results for each URL tested.
	URLValidations []URLValidation `json:"url_validations"`
	// Errors contains all errors encountered during testing.
	Errors []ErrorInfo `json:"errors"`
	// SlowRequests contains requests that exceeded the slow threshold (default 2s).
	SlowRequests []SlowRequest `json:"slow_requests"`
	// ResponseTimes contains individual response time measurements for analysis.
	ResponseTimes []ResponseTimeEntry `json:"response_times"`
	// PerformanceValidation contains pass/fail status for each performance target.
	PerformanceValidation map[string]any `json:"performance_validation,omitempty"`
	// Duration is the total test execution time as a human-readable string.
	Duration string `json:"duration"`
	// AverageResponseTime is the mean response time as a human-readable string.
	AverageResponseTime string `json:"average_response_time"`
	// MinResponseTime is the fastest response time recorded.
	MinResponseTime string `json:"min_response_time"`
	// MaxResponseTime is the slowest response time recorded.
	MaxResponseTime string `json:"max_response_time"`
	// TotalRequests is the total number of HTTP requests made.
	TotalRequests int64 `json:"total_requests"`
	// SuccessfulRequests is the count of requests with 2xx/3xx status codes.
	SuccessfulRequests int64 `json:"successful_requests"`
	// FailedRequests is the count of requests that failed or returned 4xx/5xx.
	FailedRequests int64 `json:"failed_requests"`
	// RequestsPerSecond is the average throughput during the test.
	RequestsPerSecond float64 `json:"requests_per_second"`
	// SuccessRate is the percentage of successful requests (0-100).
	SuccessRate float64 `json:"success_rate"`
	// URLsDiscovered is the count of unique URLs found during link discovery.
	URLsDiscovered int `json:"urls_discovered"`
}

// URLValidation represents the validation result for a single URL request.
// A request is considered valid if it completes without error and returns
// a successful HTTP status code (2xx or 3xx).
type URLValidation struct {
	// ResponseTime is how long the request took to complete.
	ResponseTime time.Duration `json:"response_time"`
	// ContentLength is the size of the response body in bytes.
	ContentLength int64 `json:"content_length"`
	// URL is the fully-qualified URL that was requested.
	URL string `json:"url"`
	// ContentType is the Content-Type header from the response.
	ContentType string `json:"content_type"`
	// Error contains the error message if the request failed, empty otherwise.
	Error string `json:"error,omitempty"`
	// StatusCode is the HTTP status code returned (0 if request failed).
	StatusCode int `json:"status_code"`
	// LinksFound is the count of valid links extracted from the response body.
	LinksFound int `json:"links_found"`
	// Depth is how deep in the crawl tree this URL was discovered.
	Depth int `json:"depth"`
	// IsValid is true if the request succeeded with a 2xx/3xx status.
	IsValid bool `json:"is_valid"`
}

// ErrorInfo represents an error encountered during stress testing.
// Errors are collected for reporting and debugging purposes.
type ErrorInfo struct {
	// Timestamp is when the error occurred.
	Timestamp time.Time `json:"timestamp"`
	// URL is the URL that caused the error.
	URL string `json:"url"`
	// Error is the error message (may be sanitized to hide internal details).
	Error string `json:"error"`
	// Depth is how deep in the crawl tree this URL was discovered.
	Depth int `json:"depth"`
}

// SlowRequest represents a request that exceeded the slow threshold (default 2s).
// These are tracked separately to help identify performance bottlenecks.
type SlowRequest struct {
	// ResponseTime is how long the slow request took.
	ResponseTime time.Duration `json:"response_time"`
	// URL is the slow endpoint.
	URL string `json:"url"`
	// StatusCode is the HTTP status returned.
	StatusCode int `json:"status_code"`
}

// ResponseTimeEntry represents a single response time measurement.
// Used for calculating percentile statistics and time-series analysis.
type ResponseTimeEntry struct {
	// Timestamp is when the response was received.
	Timestamp time.Time `json:"timestamp"`
	// ResponseTime is the request duration.
	ResponseTime time.Duration `json:"response_time"`
	// URL is the endpoint that was requested.
	URL string `json:"url"`
}

// PerformanceTarget represents the result of validating a performance criterion.
// Used in reports to show pass/fail status for each configured target.
type PerformanceTarget struct {
	// Name is a short identifier (e.g., "P95 Response Time").
	Name string
	// Target is the expected value as a string (e.g., "< 100ms").
	Target string
	// Actual is the measured value as a string.
	Actual string
	// Description explains what this target measures.
	Description string
	// Passed is true if the actual value met the target.
	Passed bool
}

// PerformanceTargets defines configurable performance criteria for pass/fail validation.
// Tests are evaluated against these targets to determine overall success.
type PerformanceTargets struct {
	// RequestsPerSecond is the minimum acceptable throughput.
	RequestsPerSecond float64 `json:"requests_per_second"`
	// AvgResponseTimeMs is the maximum acceptable average response time in milliseconds.
	AvgResponseTimeMs float64 `json:"avg_response_time_ms"`
	// P95ResponseTimeMs is the maximum acceptable 95th percentile response time.
	P95ResponseTimeMs float64 `json:"p95_response_time_ms"`
	// P99ResponseTimeMs is the maximum acceptable 99th percentile response time.
	P99ResponseTimeMs float64 `json:"p99_response_time_ms"`
	// SuccessRate is the minimum acceptable success percentage (0-100).
	SuccessRate float64 `json:"success_rate"`
	// ErrorRate is the maximum acceptable error percentage (0-100).
	ErrorRate float64 `json:"error_rate"`
}

// DefaultPerformanceTargets returns sensible default performance targets
func DefaultPerformanceTargets() PerformanceTargets {
	return PerformanceTargets{
		RequestsPerSecond: 100,
		AvgResponseTimeMs: 50,
		P95ResponseTimeMs: 100,
		P99ResponseTimeMs: 200,
		SuccessRate:       99.0,
		ErrorRate:         1.0,
	}
}

// AddURLResult constants for URL addition outcomes.
const (
	AddURLSuccess       = "success"
	AddURLDuplicate     = "duplicate"
	AddURLQueueFull     = "queue_full"
	AddURLDepthExceeded = "depth_exceeded"
	AddURLInvalidHost   = "invalid_host"
	AddURLParseError    = "parse_error"
)

// AddURLResult represents the result of attempting to add a URL to the crawl queue.
// Provides detailed feedback about why a URL was or wasn't added.
type AddURLResult struct {
	// Added is true if the URL was successfully added to the queue.
	Added bool
	// Reason explains why the URL was or wasn't added.
	// Values: "success", "duplicate", "queue_full", "depth_exceeded", "invalid_host", "parse_error"
	Reason string
}
