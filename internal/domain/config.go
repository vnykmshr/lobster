// Package domain defines core domain types and entities for the load testing tool.
package domain

import "time"

// AuthConfig represents authentication configuration for HTTP requests
type AuthConfig struct {
	Type        string            `json:"type"`         // "basic", "bearer", "cookie", "header"
	Username    string            `json:"username"`     // For basic auth
	Password    string            `json:"password"`     // For basic auth
	Token       string            `json:"token"`        // For bearer token auth
	Cookies     map[string]string `json:"cookies"`      // For cookie-based auth
	Headers     map[string]string `json:"headers"`      // For custom header-based auth
	CookieFile  string            `json:"cookie_file"`  // Path to cookie file (Netscape format)
}

// Config represents the complete test configuration
type Config struct {
	PerformanceTargets PerformanceTargets `json:"performance_targets"`
	Auth               *AuthConfig        `json:"auth,omitempty"`
	BaseURL            string             `json:"base_url"`
	Duration           string             `json:"duration"`
	Timeout            string             `json:"timeout"`
	UserAgent          string             `json:"user_agent"`
	OutputFile         string             `json:"output_file"`
	Rate               float64            `json:"rate"`
	Concurrency        int                `json:"concurrency"`
	MaxDepth           int                `json:"max_depth"`
	QueueSize          int                `json:"queue_size"`
	FollowLinks        bool               `json:"follow_links"`
	Respect429         bool               `json:"respect_429"`
	DryRun             bool               `json:"dry_run"`
	Verbose            bool               `json:"verbose"`
	InsecureSkipVerify bool               `json:"insecure_skip_verify"`
}

// TesterConfig represents the configuration for the stress tester
type TesterConfig struct {
	RequestTimeout     time.Duration
	Auth               *AuthConfig
	BaseURL            string
	UserAgent          string
	Rate               float64
	Concurrency        int
	MaxDepth           int
	QueueSize          int
	FollowLinks        bool
	Respect429         bool // Respect HTTP 429 (Too Many Requests) with exponential backoff
	DryRun             bool // Discover URLs without making actual test requests
	InsecureSkipVerify bool // Skip TLS certificate validation (INSECURE - for testing only)
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
