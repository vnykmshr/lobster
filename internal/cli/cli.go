// Package cli provides command-line interface utilities for the Lobster tool.
// It extracts configuration loading, authentication, display, and validation
// functions from the main package to improve testability and separation of concerns.
package cli

import (
	"github.com/vnykmshr/lobster/internal/domain"
)

// ConfigOptions holds command-line flag values for configuration.
// These are passed to LoadConfiguration to build the final Config.
type ConfigOptions struct {
	BaseURL            string
	Duration           string
	Timeout            string
	UserAgent          string
	OutputFile         string
	Rate               float64
	Concurrency        int
	MaxDepth           int
	QueueSize          int
	FollowLinks        bool
	Respect429         bool
	DryRun             bool
	Verbose            bool
	InsecureSkipVerify bool
	IgnoreRobots       bool
	AuthType           string
	AuthUsername       string
	AuthHeader         string
	AuthPasswordStdin  bool
	AuthTokenStdin     bool
}

// Result holds the loaded configuration and any warnings generated during loading.
type Result struct {
	Config   *domain.Config
	Warnings []string
}
