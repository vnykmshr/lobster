package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vnykmshr/lobster/internal/domain"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("Expected NewLoader() to return non-nil Loader")
	}
}

func TestLoadFromFile_Success(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	configJSON := `{
		"base_url": "http://example.com",
		"concurrency": 10,
		"duration": "5m",
		"timeout": "60s",
		"rate": 5.0,
		"user_agent": "TestAgent/1.0",
		"follow_links": false,
		"max_depth": 5,
		"output_file": "test-results.json",
		"verbose": true,
		"performance_targets": {
			"requests_per_second": 200,
			"avg_response_time_ms": 25,
			"p95_response_time_ms": 50,
			"p99_response_time_ms": 100,
			"success_rate": 99.5,
			"error_rate": 0.5
		}
	}`

	err := os.WriteFile(configPath, []byte(configJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	config, err := loader.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() returned error: %v", err)
	}

	// Verify loaded values
	if config.BaseURL != "http://example.com" {
		t.Errorf("Expected BaseURL 'http://example.com', got '%s'", config.BaseURL)
	}
	if config.Concurrency != 10 {
		t.Errorf("Expected Concurrency 10, got %d", config.Concurrency)
	}
	if config.Duration != "5m" {
		t.Errorf("Expected Duration '5m', got '%s'", config.Duration)
	}
	if config.Timeout != "60s" {
		t.Errorf("Expected Timeout '60s', got '%s'", config.Timeout)
	}
	if config.Rate != 5.0 {
		t.Errorf("Expected Rate 5.0, got %f", config.Rate)
	}
	if config.UserAgent != "TestAgent/1.0" {
		t.Errorf("Expected UserAgent 'TestAgent/1.0', got '%s'", config.UserAgent)
	}
	if config.FollowLinks {
		t.Errorf("Expected FollowLinks false, got true")
	}
	if config.MaxDepth != 5 {
		t.Errorf("Expected MaxDepth 5, got %d", config.MaxDepth)
	}
	if config.OutputFile != "test-results.json" {
		t.Errorf("Expected OutputFile 'test-results.json', got '%s'", config.OutputFile)
	}
	if !config.Verbose {
		t.Errorf("Expected Verbose true, got false")
	}
	if config.PerformanceTargets.RequestsPerSecond != 200 {
		t.Errorf("Expected RequestsPerSecond 200, got %f", config.PerformanceTargets.RequestsPerSecond)
	}
}

func TestLoadFromFile_NonExistentFile(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadFromFile("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.json")

	invalidJSON := `{
		"base_url": "http://example.com",
		"concurrency": "not-a-number"
	}`

	err := os.WriteFile(configPath, []byte(invalidJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	loader := NewLoader()
	_, err = loader.LoadFromFile(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestSaveToFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "save-test-config.json")

	config := &domain.Config{
		BaseURL:     "http://test.com",
		Concurrency: 15,
		Duration:    "10m",
		Timeout:     "45s",
		Rate:        7.5,
		UserAgent:   "SaveTest/1.0",
		FollowLinks: true,
		MaxDepth:    4,
		OutputFile:  "output.json",
		Verbose:     false,
		PerformanceTargets: domain.PerformanceTargets{
			RequestsPerSecond:   150,
			AvgResponseTimeMs:   30,
			P95ResponseTimeMs:   75,
			P99ResponseTimeMs:   150,
			SuccessRate:         98.0,
			ErrorRate:           2.0,
		},
	}

	loader := NewLoader()
	err := loader.SaveToFile(config, configPath)
	if err != nil {
		t.Fatalf("SaveToFile() returned error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveToFile() did not create file")
	}

	// Load back and verify
	loadedConfig, err := loader.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.BaseURL != config.BaseURL {
		t.Errorf("Saved BaseURL mismatch: expected '%s', got '%s'",
			config.BaseURL, loadedConfig.BaseURL)
	}
	if loadedConfig.Concurrency != config.Concurrency {
		t.Errorf("Saved Concurrency mismatch: expected %d, got %d",
			config.Concurrency, loadedConfig.Concurrency)
	}
}

func TestMergeWithDefaults_EmptyConfig(t *testing.T) {
	loader := NewLoader()
	config := &domain.Config{}

	merged := loader.MergeWithDefaults(config)

	defaults := domain.DefaultConfig()

	if merged.BaseURL != defaults.BaseURL {
		t.Errorf("Expected merged BaseURL '%s', got '%s'", defaults.BaseURL, merged.BaseURL)
	}
	if merged.Concurrency != defaults.Concurrency {
		t.Errorf("Expected merged Concurrency %d, got %d", defaults.Concurrency, merged.Concurrency)
	}
	if merged.Duration != defaults.Duration {
		t.Errorf("Expected merged Duration '%s', got '%s'", defaults.Duration, merged.Duration)
	}
	if merged.UserAgent != defaults.UserAgent {
		t.Errorf("Expected merged UserAgent '%s', got '%s'", defaults.UserAgent, merged.UserAgent)
	}
}

func TestMergeWithDefaults_PartialConfig(t *testing.T) {
	loader := NewLoader()
	config := &domain.Config{
		BaseURL:     "http://custom.com",
		Concurrency: 20,
		// Duration, Timeout, etc. are zero values - should be filled with defaults
	}

	merged := loader.MergeWithDefaults(config)

	// Custom values should be preserved
	if merged.BaseURL != "http://custom.com" {
		t.Errorf("Expected merged BaseURL 'http://custom.com', got '%s'", merged.BaseURL)
	}
	if merged.Concurrency != 20 {
		t.Errorf("Expected merged Concurrency 20, got %d", merged.Concurrency)
	}

	// Default values should be filled in
	defaults := domain.DefaultConfig()
	if merged.Duration != defaults.Duration {
		t.Errorf("Expected merged Duration '%s' (default), got '%s'", defaults.Duration, merged.Duration)
	}
	if merged.Timeout != defaults.Timeout {
		t.Errorf("Expected merged Timeout '%s' (default), got '%s'", defaults.Timeout, merged.Timeout)
	}
	if merged.UserAgent != defaults.UserAgent {
		t.Errorf("Expected merged UserAgent '%s' (default), got '%s'", defaults.UserAgent, merged.UserAgent)
	}
}

func TestMergeWithDefaults_FullConfig(t *testing.T) {
	loader := NewLoader()
	config := &domain.Config{
		BaseURL:     "http://full.com",
		Concurrency: 25,
		Duration:    "15m",
		Timeout:     "90s",
		Rate:        12.0,
		UserAgent:   "FullTest/2.0",
		FollowLinks: false,
		MaxDepth:    10,
		OutputFile:  "custom-output.json",
		Verbose:     true,
		PerformanceTargets: domain.PerformanceTargets{
			RequestsPerSecond:   300,
			AvgResponseTimeMs:   20,
			P95ResponseTimeMs:   40,
			P99ResponseTimeMs:   80,
			SuccessRate:         99.9,
			ErrorRate:           0.1,
		},
	}

	merged := loader.MergeWithDefaults(config)

	// All custom values should be preserved
	if merged.BaseURL != "http://full.com" {
		t.Errorf("Custom BaseURL not preserved")
	}
	if merged.Concurrency != 25 {
		t.Errorf("Custom Concurrency not preserved")
	}
	if merged.Duration != "15m" {
		t.Errorf("Custom Duration not preserved")
	}
	if merged.PerformanceTargets.RequestsPerSecond != 300 {
		t.Errorf("Custom RequestsPerSecond not preserved")
	}
}
