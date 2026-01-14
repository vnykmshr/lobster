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

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected valid default config, got error: %v", err)
	}
}

func TestConfig_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr string
	}{
		{
			name:    "negative concurrency",
			modify:  func(c *Config) { c.Concurrency = -1 },
			wantErr: "concurrency must be > 0",
		},
		{
			name:    "zero concurrency",
			modify:  func(c *Config) { c.Concurrency = 0 },
			wantErr: "concurrency must be > 0",
		},
		{
			name:    "negative max depth",
			modify:  func(c *Config) { c.MaxDepth = -1 },
			wantErr: "max-depth cannot be negative",
		},
		{
			name:    "zero queue size",
			modify:  func(c *Config) { c.QueueSize = 0 },
			wantErr: "queue-size must be > 0",
		},
		{
			name:    "negative rate",
			modify:  func(c *Config) { c.Rate = -1.0 },
			wantErr: "rate cannot be negative",
		},
		{
			name:    "invalid duration",
			modify:  func(c *Config) { c.Duration = "not-a-duration" },
			wantErr: "invalid duration",
		},
		{
			name:    "invalid timeout",
			modify:  func(c *Config) { c.Timeout = "bad" },
			wantErr: "invalid timeout",
		},
		{
			name:    "empty base URL",
			modify:  func(c *Config) { c.BaseURL = "" },
			wantErr: "base URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		auth    AuthConfig
		wantErr string
	}{
		{
			name:    "valid empty",
			auth:    AuthConfig{},
			wantErr: "",
		},
		{
			name:    "valid basic",
			auth:    AuthConfig{Type: "basic", Username: "user"},
			wantErr: "",
		},
		{
			name:    "valid bearer",
			auth:    AuthConfig{Type: "bearer", Token: "token123"},
			wantErr: "",
		},
		{
			name:    "invalid type",
			auth:    AuthConfig{Type: "oauth"},
			wantErr: "invalid auth type",
		},
		{
			name:    "basic without username",
			auth:    AuthConfig{Type: "basic"},
			wantErr: "basic auth requires username",
		},
		{
			name:    "bearer without token",
			auth:    AuthConfig{Type: "bearer"},
			wantErr: "bearer auth requires token",
		},
		{
			name:    "valid cookie with cookies",
			auth:    AuthConfig{Type: "cookie", Cookies: map[string]string{"session": "abc"}},
			wantErr: "",
		},
		{
			name:    "valid cookie with cookie_file",
			auth:    AuthConfig{Type: "cookie", CookieFile: "/path/to/cookies.txt"},
			wantErr: "",
		},
		{
			name:    "cookie without cookies or file",
			auth:    AuthConfig{Type: "cookie"},
			wantErr: "cookie auth requires cookies or cookie_file",
		},
		{
			name:    "valid header",
			auth:    AuthConfig{Type: "header", Headers: map[string]string{"X-API-Key": "secret"}},
			wantErr: "",
		},
		{
			name:    "header without headers",
			auth:    AuthConfig{Type: "header"},
			wantErr: "header auth requires headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.auth.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErr)
					return
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
