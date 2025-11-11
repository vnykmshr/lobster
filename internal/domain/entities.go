package domain

import "time"

// URLTask represents a URL to be tested with its depth in the crawl tree
type URLTask struct {
	URL   string
	Depth int
}

// TestResults contains comprehensive test execution results
type TestResults struct {
	URLValidations        []URLValidation        `json:"url_validations"`
	Errors                []ErrorInfo            `json:"errors"`
	SlowRequests          []SlowRequest          `json:"slow_requests"`
	ResponseTimes         []ResponseTimeEntry    `json:"response_times"`
	PerformanceValidation map[string]interface{} `json:"performance_validation,omitempty"`
	Duration              string                 `json:"duration"`
	AverageResponseTime   string                 `json:"average_response_time"`
	MinResponseTime       string                 `json:"min_response_time"`
	MaxResponseTime       string                 `json:"max_response_time"`
	TotalRequests         int64                  `json:"total_requests"`
	SuccessfulRequests    int64                  `json:"successful_requests"`
	FailedRequests        int64                  `json:"failed_requests"`
	RequestsPerSecond     float64                `json:"requests_per_second"`
	SuccessRate           float64                `json:"success_rate"`
	URLsDiscovered        int                    `json:"urls_discovered"`
}

// URLValidation represents the validation result for a single URL
type URLValidation struct {
	ResponseTime  time.Duration `json:"response_time"`
	ContentLength int64         `json:"content_length"`
	URL           string        `json:"url"`
	ContentType   string        `json:"content_type"`
	Error         string        `json:"error,omitempty"`
	StatusCode    int           `json:"status_code"`
	LinksFound    int           `json:"links_found"`
	Depth         int           `json:"depth"`
	IsValid       bool          `json:"is_valid"`
}

// ErrorInfo represents an error encountered during testing
type ErrorInfo struct {
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	Error     string    `json:"error"`
	Depth     int       `json:"depth"`
}

// SlowRequest represents a request that exceeded the slow threshold
type SlowRequest struct {
	ResponseTime time.Duration `json:"response_time"`
	URL          string        `json:"url"`
	StatusCode   int           `json:"status_code"`
}

// ResponseTimeEntry represents a single response time measurement
type ResponseTimeEntry struct {
	Timestamp    time.Time     `json:"timestamp"`
	ResponseTime time.Duration `json:"response_time"`
	URL          string        `json:"url"`
}

// PerformanceTarget represents a performance validation target
type PerformanceTarget struct {
	Name        string
	Target      string
	Actual      string
	Description string
	Passed      bool
}

// PerformanceTargets defines configurable performance criteria
type PerformanceTargets struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	AvgResponseTimeMs float64 `json:"avg_response_time_ms"`
	P95ResponseTimeMs float64 `json:"p95_response_time_ms"`
	P99ResponseTimeMs float64 `json:"p99_response_time_ms"`
	SuccessRate       float64 `json:"success_rate"`
	ErrorRate         float64 `json:"error_rate"`
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
