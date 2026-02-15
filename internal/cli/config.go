package cli

import (
	"fmt"

	"github.com/1mb-dev/lobster/v2/internal/config"
	"github.com/1mb-dev/lobster/v2/internal/domain"
)

// LoadConfiguration loads configuration from file (if provided) and merges with CLI options.
// CLI flags override file configuration values.
func LoadConfiguration(configPath string, opts *ConfigOptions) (*domain.Config, error) {
	loader := config.NewLoader()

	var cfg *domain.Config

	if configPath != "" {
		// Load from file
		loadedCfg, err := loader.LoadFromFile(configPath)
		if err != nil {
			return nil, err
		}
		cfg = loadedCfg
	} else {
		// Start with defaults
		defaultCfg := domain.DefaultConfig()
		cfg = &defaultCfg
	}

	// Override with CLI flags (if provided)
	if opts.BaseURL != "" {
		cfg.BaseURL = opts.BaseURL
	}
	if opts.Concurrency != 0 {
		cfg.Concurrency = opts.Concurrency
	}
	if opts.Duration != "" {
		cfg.Duration = opts.Duration
	}
	if opts.Timeout != "" {
		cfg.Timeout = opts.Timeout
	}
	if opts.Rate != 0 {
		cfg.Rate = opts.Rate
	}
	if opts.UserAgent != "" {
		cfg.UserAgent = opts.UserAgent
	}
	if opts.MaxDepth != 0 {
		cfg.MaxDepth = opts.MaxDepth
	}
	if opts.QueueSize != 0 {
		cfg.QueueSize = opts.QueueSize
	}
	if opts.OutputFile != "" {
		cfg.OutputFile = opts.OutputFile
	}
	cfg.FollowLinks = opts.FollowLinks
	cfg.Respect429 = opts.Respect429
	cfg.DryRun = opts.DryRun
	cfg.Verbose = opts.Verbose
	cfg.InsecureSkipVerify = opts.InsecureSkipVerify
	cfg.IgnoreRobots = opts.IgnoreRobots

	// Build authentication configuration from CLI flags and environment variables
	authCfg, err := BuildAuthConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("authentication configuration: %w", err)
	}
	if authCfg != nil {
		cfg.Auth = authCfg
	}

	// Merge with defaults for any missing values
	cfg = loader.MergeWithDefaults(cfg)

	return cfg, nil
}
