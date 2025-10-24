// Package domain defines core domain types and entities for the load testing tool.
package domain

import "time"

// Config represents the complete test configuration
type Config struct {
	PerformanceTargets PerformanceTargets `json:"performance_targets"`
	BaseURL            string             `json:"base_url"`
	Duration           string             `json:"duration"`
	Timeout            string             `json:"timeout"`
	UserAgent          string             `json:"user_agent"`
	OutputFile         string             `json:"output_file"`
	Rate               float64            `json:"rate"`
	Concurrency        int                `json:"concurrency"`
	MaxDepth           int                `json:"max_depth"`
	FollowLinks        bool               `json:"follow_links"`
	Verbose            bool               `json:"verbose"`
}

// TesterConfig represents the configuration for the stress tester
type TesterConfig struct {
	RequestTimeout time.Duration
	BaseURL        string
	UserAgent      string
	Rate           float64
	Concurrency    int
	MaxDepth       int
	FollowLinks    bool
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
