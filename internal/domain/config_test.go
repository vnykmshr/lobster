package domain

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify non-zero values
	if cfg.BaseURL == "" {
		t.Error("Expected BaseURL to be set")
	}
	if cfg.Concurrency == 0 {
		t.Error("Expected Concurrency to be non-zero")
	}
	if cfg.Duration == "" {
		t.Error("Expected Duration to be set")
	}
	if cfg.Timeout == "" {
		t.Error("Expected Timeout to be set")
	}
	if cfg.Rate == 0 {
		t.Error("Expected Rate to be non-zero")
	}
	if cfg.UserAgent == "" {
		t.Error("Expected UserAgent to be set")
	}
	if cfg.MaxDepth == 0 {
		t.Error("Expected MaxDepth to be non-zero")
	}

	// Verify expected defaults
	expectedDefaults := map[string]interface{}{
		"BaseURL":     "http://localhost:3000",
		"Concurrency": 5,
		"Duration":    "2m",
		"Timeout":     "30s",
		"Rate":        2.0,
		"UserAgent":   "Lobster/1.0",
		"MaxDepth":    3,
		"FollowLinks": true,
		"Verbose":     false,
	}

	if cfg.BaseURL != expectedDefaults["BaseURL"] {
		t.Errorf("Expected BaseURL %v, got %v", expectedDefaults["BaseURL"], cfg.BaseURL)
	}
	if cfg.Concurrency != expectedDefaults["Concurrency"] {
		t.Errorf("Expected Concurrency %v, got %v", expectedDefaults["Concurrency"], cfg.Concurrency)
	}
	if cfg.Duration != expectedDefaults["Duration"] {
		t.Errorf("Expected Duration %v, got %v", expectedDefaults["Duration"], cfg.Duration)
	}
	if cfg.Timeout != expectedDefaults["Timeout"] {
		t.Errorf("Expected Timeout %v, got %v", expectedDefaults["Timeout"], cfg.Timeout)
	}
	if cfg.Rate != expectedDefaults["Rate"] {
		t.Errorf("Expected Rate %v, got %v", expectedDefaults["Rate"], cfg.Rate)
	}
	if cfg.UserAgent != expectedDefaults["UserAgent"] {
		t.Errorf("Expected UserAgent %v, got %v", expectedDefaults["UserAgent"], cfg.UserAgent)
	}
	if cfg.MaxDepth != expectedDefaults["MaxDepth"] {
		t.Errorf("Expected MaxDepth %v, got %v", expectedDefaults["MaxDepth"], cfg.MaxDepth)
	}
	if cfg.FollowLinks != expectedDefaults["FollowLinks"] {
		t.Errorf("Expected FollowLinks %v, got %v", expectedDefaults["FollowLinks"], cfg.FollowLinks)
	}
	if cfg.Verbose != expectedDefaults["Verbose"] {
		t.Errorf("Expected Verbose %v, got %v", expectedDefaults["Verbose"], cfg.Verbose)
	}
}

func TestDefaultPerformanceTargets(t *testing.T) {
	targets := DefaultPerformanceTargets()

	// Verify all fields are non-zero
	if targets.RequestsPerSecond == 0 {
		t.Error("Expected RequestsPerSecond to be non-zero")
	}
	if targets.AvgResponseTimeMs == 0 {
		t.Error("Expected AvgResponseTimeMs to be non-zero")
	}
	if targets.P95ResponseTimeMs == 0 {
		t.Error("Expected P95ResponseTimeMs to be non-zero")
	}
	if targets.P99ResponseTimeMs == 0 {
		t.Error("Expected P99ResponseTimeMs to be non-zero")
	}
	if targets.SuccessRate == 0 {
		t.Error("Expected SuccessRate to be non-zero")
	}
	if targets.ErrorRate == 0 {
		t.Error("Expected ErrorRate to be non-zero")
	}

	// Verify expected values
	if targets.RequestsPerSecond != 100 {
		t.Errorf("Expected RequestsPerSecond 100, got %v", targets.RequestsPerSecond)
	}
	if targets.AvgResponseTimeMs != 50 {
		t.Errorf("Expected AvgResponseTimeMs 50, got %v", targets.AvgResponseTimeMs)
	}
	if targets.P95ResponseTimeMs != 100 {
		t.Errorf("Expected P95ResponseTimeMs 100, got %v", targets.P95ResponseTimeMs)
	}
	if targets.P99ResponseTimeMs != 200 {
		t.Errorf("Expected P99ResponseTimeMs 200, got %v", targets.P99ResponseTimeMs)
	}
	if targets.SuccessRate != 99.0 {
		t.Errorf("Expected SuccessRate 99.0, got %v", targets.SuccessRate)
	}
	if targets.ErrorRate != 1.0 {
		t.Errorf("Expected ErrorRate 1.0, got %v", targets.ErrorRate)
	}
}

func TestDefaultPerformanceTargets_Consistency(t *testing.T) {
	targets := DefaultPerformanceTargets()

	// Verify P95 <= P99
	if targets.P95ResponseTimeMs > targets.P99ResponseTimeMs {
		t.Errorf("P95 (%v) should be <= P99 (%v)", targets.P95ResponseTimeMs, targets.P99ResponseTimeMs)
	}

	// Verify SuccessRate + ErrorRate = 100%
	totalRate := targets.SuccessRate + targets.ErrorRate
	if totalRate != 100.0 {
		t.Errorf("SuccessRate + ErrorRate should equal 100, got %v", totalRate)
	}
}
