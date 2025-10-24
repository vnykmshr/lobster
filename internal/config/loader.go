package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vnykmshr/lobster/internal/domain"
)

// Loader handles loading configuration from various sources
type Loader struct{}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadFromFile loads configuration from a JSON file
func (l *Loader) LoadFromFile(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config domain.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	return &config, nil
}

// SaveToFile saves configuration to a JSON file
func (l *Loader) SaveToFile(config *domain.Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// MergeWithDefaults merges provided config with defaults
func (l *Loader) MergeWithDefaults(config *domain.Config) *domain.Config {
	defaults := domain.DefaultConfig()

	if config.BaseURL == "" {
		config.BaseURL = defaults.BaseURL
	}
	if config.Concurrency == 0 {
		config.Concurrency = defaults.Concurrency
	}
	if config.Duration == "" {
		config.Duration = defaults.Duration
	}
	if config.Timeout == "" {
		config.Timeout = defaults.Timeout
	}
	if config.Rate == 0 {
		config.Rate = defaults.Rate
	}
	if config.UserAgent == "" {
		config.UserAgent = defaults.UserAgent
	}
	if config.MaxDepth == 0 {
		config.MaxDepth = defaults.MaxDepth
	}

	// Merge performance targets
	if config.PerformanceTargets.RequestsPerSecond == 0 {
		config.PerformanceTargets.RequestsPerSecond = defaults.PerformanceTargets.RequestsPerSecond
	}
	if config.PerformanceTargets.AvgResponseTimeMs == 0 {
		config.PerformanceTargets.AvgResponseTimeMs = defaults.PerformanceTargets.AvgResponseTimeMs
	}
	if config.PerformanceTargets.P95ResponseTimeMs == 0 {
		config.PerformanceTargets.P95ResponseTimeMs = defaults.PerformanceTargets.P95ResponseTimeMs
	}
	if config.PerformanceTargets.P99ResponseTimeMs == 0 {
		config.PerformanceTargets.P99ResponseTimeMs = defaults.PerformanceTargets.P99ResponseTimeMs
	}
	if config.PerformanceTargets.SuccessRate == 0 {
		config.PerformanceTargets.SuccessRate = defaults.PerformanceTargets.SuccessRate
	}
	if config.PerformanceTargets.ErrorRate == 0 {
		config.PerformanceTargets.ErrorRate = defaults.PerformanceTargets.ErrorRate
	}

	return config
}
