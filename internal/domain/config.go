package domain

import "time"

// Config represents the complete test configuration
type Config struct {
	BaseURL            string             `json:"base_url"`
	Concurrency        int                `json:"concurrency"`
	Duration           string             `json:"duration"`
	Timeout            string             `json:"timeout"`
	Rate               float64            `json:"rate"`
	UserAgent          string             `json:"user_agent"`
	FollowLinks        bool               `json:"follow_links"`
	MaxDepth           int                `json:"max_depth"`
	OutputFile         string             `json:"output_file"`
	Verbose            bool               `json:"verbose"`
	PerformanceTargets PerformanceTargets `json:"performance_targets"`
}

// TesterConfig represents the configuration for the stress tester
type TesterConfig struct {
	BaseURL        string
	Concurrency    int
	RequestTimeout time.Duration
	UserAgent      string
	FollowLinks    bool
	MaxDepth       int
	Rate           float64
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
		OutputFile:         "",
		Verbose:            false,
		PerformanceTargets: DefaultPerformanceTargets(),
	}
}
