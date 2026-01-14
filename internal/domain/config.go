// Package domain defines core domain types and entities for the load testing tool.
package domain

import (
	"fmt"
	"time"
)

// AuthConfig represents authentication configuration for HTTP requests.
// Supports basic auth, bearer tokens, cookies, and custom headers.
// Credentials should be provided via environment variables or stdin for security.
type AuthConfig struct {
	// Type specifies the authentication method: "basic", "bearer", "cookie", or "header".
	Type string `json:"type"`
	// Username is required for basic auth. Set via -auth-username flag.
	Username string `json:"username"`
	// Password is for basic auth. Set via LOBSTER_AUTH_PASSWORD env var or stdin.
	Password string `json:"password"`
	// Token is for bearer auth. Set via LOBSTER_AUTH_TOKEN env var or stdin.
	Token string `json:"token"`
	// Cookies are key-value pairs sent with each request for cookie-based auth.
	Cookies map[string]string `json:"cookies"`
	// Headers are custom headers sent with each request for header-based auth.
	Headers map[string]string `json:"headers"`
	// CookieFile is the path to a Netscape-format cookie file.
	CookieFile string `json:"cookie_file"`
}

// Config represents the complete test configuration loaded from CLI flags and config files.
// Use DefaultConfig() to get sensible defaults, then override as needed.
type Config struct {
	// PerformanceTargets defines pass/fail thresholds for the test.
	PerformanceTargets PerformanceTargets `json:"performance_targets"`
	// Auth contains authentication configuration (optional).
	Auth *AuthConfig `json:"auth,omitempty"`
	// BaseURL is the starting URL for the stress test (required).
	BaseURL string `json:"base_url"`
	// Duration is the test duration as a Go duration string (e.g., "2m", "30s").
	Duration string `json:"duration"`
	// Timeout is the HTTP request timeout as a Go duration string.
	Timeout string `json:"timeout"`
	// UserAgent is the User-Agent header sent with each request.
	UserAgent string `json:"user_agent"`
	// OutputFile is the path to write the report (HTML or JSON based on extension).
	OutputFile string `json:"output_file"`
	// Rate is the maximum requests per second per worker (0 = unlimited).
	Rate float64 `json:"rate"`
	// Concurrency is the number of parallel workers making requests.
	Concurrency int `json:"concurrency"`
	// MaxDepth is the maximum link depth to crawl (0 = base URL only).
	MaxDepth int `json:"max_depth"`
	// QueueSize is the maximum number of URLs to queue for testing.
	QueueSize int `json:"queue_size"`
	// FollowLinks enables recursive link discovery from HTML pages.
	FollowLinks bool `json:"follow_links"`
	// Respect429 enables exponential backoff on HTTP 429 responses.
	Respect429 bool `json:"respect_429"`
	// DryRun discovers URLs without making test requests.
	DryRun bool `json:"dry_run"`
	// Verbose enables detailed logging output.
	Verbose bool `json:"verbose"`
	// InsecureSkipVerify skips TLS certificate validation (requires LOBSTER_INSECURE_TLS=true).
	InsecureSkipVerify bool `json:"insecure_skip_verify"`
	// IgnoreRobots bypasses robots.txt restrictions.
	IgnoreRobots bool `json:"ignore_robots"`
}

// TesterConfig represents the internal configuration for the stress tester.
// This is converted from Config after parsing and validation.
type TesterConfig struct {
	// RequestTimeout is the parsed timeout duration for HTTP requests.
	RequestTimeout time.Duration
	// Auth contains authentication settings.
	Auth *AuthConfig
	// BaseURL is the starting URL for the stress test.
	BaseURL string
	// UserAgent is the User-Agent header value.
	UserAgent string
	// Rate is the maximum requests per second per worker.
	Rate float64
	// Concurrency is the number of parallel workers.
	Concurrency int
	// MaxDepth is the maximum crawl depth.
	MaxDepth int
	// QueueSize is the URL queue capacity.
	QueueSize int
	// MaxResponseSize is the maximum response body to read (default 10MB).
	MaxResponseSize int64
	// FollowLinks enables link discovery from responses.
	FollowLinks bool
	// Respect429 enables backoff on rate limit responses.
	Respect429 bool
	// DryRun discovers URLs without stress testing.
	DryRun bool
	// InsecureSkipVerify skips TLS certificate validation.
	InsecureSkipVerify bool
	// IgnoreRobots bypasses robots.txt restrictions.
	IgnoreRobots bool
	// Verbose enables detailed logging.
	Verbose bool
	// NoProgress disables the progress bar.
	NoProgress bool
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:            "http://localhost:3000",
		Concurrency:        5,
		Duration:           "2m",
		Timeout:            "30s",
		Rate:               2.0,
		UserAgent:          "Lobster/1.0",
		FollowLinks:        true,
		MaxDepth:           3,
		QueueSize:          10000, // ~80KB per 10K queue (assuming 8 bytes per URLTask)
		Respect429:         true,  // Respect rate limiting by default
		DryRun:             false, // Perform actual tests by default
		OutputFile:         "",
		Verbose:            false,
		PerformanceTargets: DefaultPerformanceTargets(),
	}
}

// Validate checks that all configuration values are valid.
// Returns an error describing the first invalid value found.
func (c *Config) Validate() error {
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be > 0, got %d", c.Concurrency)
	}

	if c.MaxDepth < 0 {
		return fmt.Errorf("max-depth cannot be negative, got %d", c.MaxDepth)
	}

	if c.QueueSize <= 0 {
		return fmt.Errorf("queue-size must be > 0, got %d", c.QueueSize)
	}

	if c.Rate < 0 {
		return fmt.Errorf("rate cannot be negative, got %.2f", c.Rate)
	}

	if c.Duration != "" {
		if _, err := time.ParseDuration(c.Duration); err != nil {
			return fmt.Errorf("invalid duration %q: %w", c.Duration, err)
		}
	}

	if c.Timeout != "" {
		if _, err := time.ParseDuration(c.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
		}
	}

	if c.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	// Validate auth config if present
	if c.Auth != nil {
		if err := c.Auth.Validate(); err != nil {
			return fmt.Errorf("auth config: %w", err)
		}
	}

	return nil
}

// Validate checks that auth configuration values are valid.
func (a *AuthConfig) Validate() error {
	validTypes := map[string]bool{
		"":       true, // Empty is allowed (no auth)
		"basic":  true,
		"bearer": true,
		"cookie": true,
		"header": true,
	}

	if !validTypes[a.Type] {
		return fmt.Errorf("invalid auth type %q: must be one of basic, bearer, cookie, header", a.Type)
	}

	if a.Type == "basic" && a.Username == "" {
		return fmt.Errorf("basic auth requires username")
	}

	if a.Type == "bearer" && a.Token == "" {
		return fmt.Errorf("bearer auth requires token")
	}

	return nil
}
