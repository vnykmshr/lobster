// Package config handles loading and saving configuration files for load tests.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/vnykmshr/lobster/internal/domain"
)

// envVarPattern matches ${VAR_NAME} or ${VAR_NAME:-default} syntax
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// Loader handles loading configuration from various sources
type Loader struct{}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadFromFile loads configuration from a JSON file.
// Supports environment variable substitution using ${VAR_NAME} syntax.
// Optional default values can be specified with ${VAR_NAME:-default}.
func (l *Loader) LoadFromFile(path string) (*domain.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file %s: %w\nCheck if file exists and has read permissions", path, err)
	}

	// Substitute environment variables before parsing JSON
	expandedData, err := substituteEnvVars(string(data))
	if err != nil {
		return nil, fmt.Errorf("environment variable substitution failed: %w", err)
	}

	var config domain.Config
	err = json.Unmarshal([]byte(expandedData), &config)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON in config file: %w\nVerify JSON syntax at %s", err, path)
	}

	return &config, nil
}

// substituteEnvVars replaces ${VAR_NAME} patterns with environment variable values.
// Supports ${VAR_NAME:-default} syntax for default values when env var is not set.
// Returns an error if a required env var (no default) is not set.
// Note: ${VAR:-} with empty default is valid and means "use empty string if VAR is unset".
func substituteEnvVars(content string) (string, error) {
	var missingVars []string

	result := envVarPattern.ReplaceAllStringFunc(content, func(match string) string {
		submatches := envVarPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		varName := submatches[1]
		// Check if default syntax was used by looking for :- in the match
		// This properly handles ${VAR:-} where default is empty string
		hasDefault := strings.Contains(match, ":-")
		defaultValue := ""
		if hasDefault && len(submatches) > 2 {
			defaultValue = submatches[2]
		}

		// Use LookupEnv to distinguish between unset and empty env vars
		value, isSet := os.LookupEnv(varName)
		if !isSet {
			if hasDefault {
				return defaultValue
			}
			missingVars = append(missingVars, varName)
			return match // Keep original placeholder for error reporting
		}
		return value
	})

	if len(missingVars) > 0 {
		return "", fmt.Errorf("missing required environment variables: %v\nSet these variables or provide defaults using ${VAR:-default} syntax", missingVars)
	}

	return result, nil
}

// SaveToFile saves configuration to a JSON file
func (l *Loader) SaveToFile(config *domain.Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	err = os.WriteFile(path, data, 0o600)
	if err != nil {
		return fmt.Errorf("cannot write config file %s: %w\nCheck directory exists and has write permissions", path, err)
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
	if config.QueueSize == 0 {
		config.QueueSize = defaults.QueueSize
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
