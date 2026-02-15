// Package config handles loading and saving configuration files for load tests.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/1mb-dev/lobster/v2/internal/domain"
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

// mergeString returns value if non-empty, otherwise returns fallback.
func mergeString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// mergeInt returns value if non-zero, otherwise returns fallback.
func mergeInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

// mergeFloat64 returns value if non-zero, otherwise returns fallback.
func mergeFloat64(value, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

// MergeWithDefaults merges provided config with defaults.
// Zero values in config are replaced with corresponding default values.
func (l *Loader) MergeWithDefaults(config *domain.Config) *domain.Config {
	defaults := domain.DefaultConfig()

	config.BaseURL = mergeString(config.BaseURL, defaults.BaseURL)
	config.Concurrency = mergeInt(config.Concurrency, defaults.Concurrency)
	config.Duration = mergeString(config.Duration, defaults.Duration)
	config.Timeout = mergeString(config.Timeout, defaults.Timeout)
	config.Rate = mergeFloat64(config.Rate, defaults.Rate)
	config.UserAgent = mergeString(config.UserAgent, defaults.UserAgent)
	config.MaxDepth = mergeInt(config.MaxDepth, defaults.MaxDepth)
	config.QueueSize = mergeInt(config.QueueSize, defaults.QueueSize)

	// Merge performance targets
	pt := &config.PerformanceTargets
	dt := &defaults.PerformanceTargets
	pt.RequestsPerSecond = mergeFloat64(pt.RequestsPerSecond, dt.RequestsPerSecond)
	pt.AvgResponseTimeMs = mergeFloat64(pt.AvgResponseTimeMs, dt.AvgResponseTimeMs)
	pt.P95ResponseTimeMs = mergeFloat64(pt.P95ResponseTimeMs, dt.P95ResponseTimeMs)
	pt.P99ResponseTimeMs = mergeFloat64(pt.P99ResponseTimeMs, dt.P99ResponseTimeMs)
	pt.SuccessRate = mergeFloat64(pt.SuccessRate, dt.SuccessRate)
	pt.ErrorRate = mergeFloat64(pt.ErrorRate, dt.ErrorRate)

	return config
}
