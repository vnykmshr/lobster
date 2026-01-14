// Package domain defines core domain types and entities for the load testing tool.
package domain

import (
	"fmt"
	"time"
)

// AuthConfig represents authentication configuration for HTTP requests
type AuthConfig struct {
	Type       string            `json:"type"`        // "basic", "bearer", "cookie", "header"
	Username   string            `json:"username"`    // For basic auth
	Password   string            `json:"password"`    // For basic auth
	Token      string            `json:"token"`       // For bearer token auth
	Cookies    map[string]string `json:"cookies"`     // For cookie-based auth
	Headers    map[string]string `json:"headers"`     // For custom header-based auth
	CookieFile string            `json:"cookie_file"` // Path to cookie file (Netscape format)
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
	IgnoreRobots       bool               `json:"ignore_robots"`
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
	MaxResponseSize    int64 // Maximum response body size to read (default 10MB)
	FollowLinks        bool
	Respect429         bool // Respect HTTP 429 (Too Many Requests) with exponential backoff
	DryRun             bool // Discover URLs without making actual test requests
	InsecureSkipVerify bool // Skip TLS certificate validation (INSECURE - for testing only)
	IgnoreRobots       bool // Ignore robots.txt directives (use responsibly)
	Verbose            bool // Enable verbose logging
	NoProgress         bool // Disable progress updates
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
