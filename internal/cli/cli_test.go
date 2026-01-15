package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfiguration_Defaults(t *testing.T) {
	opts := &ConfigOptions{}
	cfg, err := LoadConfiguration("", opts)
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	if cfg.BaseURL == "" {
		t.Error("Expected default BaseURL to be set")
	}
	if cfg.Concurrency == 0 {
		t.Error("Expected default Concurrency to be set")
	}
	if cfg.Duration == "" {
		t.Error("Expected default Duration to be set")
	}
}

func TestLoadConfiguration_CLIOverrides(t *testing.T) {
	opts := &ConfigOptions{
		BaseURL:     "http://custom.example.com",
		Concurrency: 25,
		Duration:    "10m",
		Rate:        15.0,
	}
	cfg, err := LoadConfiguration("", opts)
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	if cfg.BaseURL != "http://custom.example.com" {
		t.Errorf("Expected BaseURL 'http://custom.example.com', got '%s'", cfg.BaseURL)
	}
	if cfg.Concurrency != 25 {
		t.Errorf("Expected Concurrency 25, got %d", cfg.Concurrency)
	}
	if cfg.Duration != "10m" {
		t.Errorf("Expected Duration '10m', got '%s'", cfg.Duration)
	}
	if cfg.Rate != 15.0 {
		t.Errorf("Expected Rate 15.0, got %f", cfg.Rate)
	}
}

func TestLoadConfiguration_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	configJSON := `{
		"base_url": "http://file.example.com",
		"concurrency": 50,
		"duration": "30m"
	}`

	err := os.WriteFile(configPath, []byte(configJSON), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	opts := &ConfigOptions{}
	cfg, err := LoadConfiguration(configPath, opts)
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	if cfg.BaseURL != "http://file.example.com" {
		t.Errorf("Expected BaseURL 'http://file.example.com', got '%s'", cfg.BaseURL)
	}
	if cfg.Concurrency != 50 {
		t.Errorf("Expected Concurrency 50, got %d", cfg.Concurrency)
	}
}

func TestLoadConfiguration_NonExistentFile(t *testing.T) {
	opts := &ConfigOptions{}
	_, err := LoadConfiguration("/nonexistent/path/config.json", opts)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestBuildAuthConfig_NoAuth(t *testing.T) {
	opts := &ConfigOptions{}
	cfg, err := BuildAuthConfig(opts)
	if err != nil {
		t.Fatalf("BuildAuthConfig() error = %v", err)
	}
	if cfg != nil {
		t.Error("Expected nil config when no auth is provided")
	}
}

func TestBuildAuthConfig_Basic(t *testing.T) {
	t.Setenv("LOBSTER_AUTH_PASSWORD", "testpass")

	opts := &ConfigOptions{
		AuthType:     "basic",
		AuthUsername: "testuser",
	}
	cfg, err := BuildAuthConfig(opts)
	if err != nil {
		t.Fatalf("BuildAuthConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if cfg.Type != "basic" {
		t.Errorf("Expected Type 'basic', got '%s'", cfg.Type)
	}
	if cfg.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("Expected Password 'testpass', got '%s'", cfg.Password)
	}
}

func TestBuildAuthConfig_MutuallyExclusiveStdin(t *testing.T) {
	opts := &ConfigOptions{
		AuthPasswordStdin: true,
		AuthTokenStdin:    true,
	}
	_, err := BuildAuthConfig(opts)
	if err == nil {
		t.Error("Expected error for mutually exclusive stdin flags")
	}
}

func TestBuildAuthConfig_Header(t *testing.T) {
	opts := &ConfigOptions{
		AuthType:   "header",
		AuthHeader: "X-API-Key:secret123",
	}
	cfg, err := BuildAuthConfig(opts)
	if err != nil {
		t.Fatalf("BuildAuthConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if cfg.Headers == nil {
		t.Fatal("Expected non-nil Headers map")
	}
	if cfg.Headers["X-API-Key"] != "secret123" {
		t.Errorf("Expected header value 'secret123', got '%s'", cfg.Headers["X-API-Key"])
	}
}

func TestBuildAuthConfig_InvalidHeader(t *testing.T) {
	opts := &ConfigOptions{
		AuthType:   "header",
		AuthHeader: "invalid-no-colon",
	}
	_, err := BuildAuthConfig(opts)
	if err == nil {
		t.Error("Expected error for invalid header format")
	}
}

func TestBuildAuthConfig_Cookie(t *testing.T) {
	t.Setenv("LOBSTER_AUTH_COOKIE", "session=abc123")

	opts := &ConfigOptions{
		AuthType: "cookie",
	}
	cfg, err := BuildAuthConfig(opts)
	if err != nil {
		t.Fatalf("BuildAuthConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
	if cfg.Cookies == nil {
		t.Fatal("Expected non-nil Cookies map")
	}
	if cfg.Cookies["session"] != "abc123" {
		t.Errorf("Expected cookie value 'abc123', got '%s'", cfg.Cookies["session"])
	}
}

func TestBuildAuthConfig_InvalidCookie(t *testing.T) {
	t.Setenv("LOBSTER_AUTH_COOKIE", "invalid-no-equals")

	opts := &ConfigOptions{
		AuthType: "cookie",
	}
	_, err := BuildAuthConfig(opts)
	if err == nil {
		t.Error("Expected error for invalid cookie format")
	}
}

func TestValidateRateLimit_Zero(t *testing.T) {
	rate := 0.0
	err := ValidateRateLimit(&rate)
	if err != nil {
		t.Errorf("ValidateRateLimit() error = %v", err)
	}
	// Rate should remain 0 (unlimited)
	if rate != 0 {
		t.Errorf("Expected rate to remain 0, got %f", rate)
	}
}

func TestValidateRateLimit_BelowMinimum(t *testing.T) {
	rate := 0.05 // Below MinRate of 0.1
	err := ValidateRateLimit(&rate)
	if err != nil {
		t.Errorf("ValidateRateLimit() error = %v", err)
	}
	// Rate should be adjusted to MinRate
	if rate != MinRate {
		t.Errorf("Expected rate to be adjusted to %f, got %f", MinRate, rate)
	}
}

func TestValidateRateLimit_Normal(t *testing.T) {
	rate := 5.0
	err := ValidateRateLimit(&rate)
	if err != nil {
		t.Errorf("ValidateRateLimit() error = %v", err)
	}
	// Rate should remain unchanged
	if rate != 5.0 {
		t.Errorf("Expected rate to remain 5.0, got %f", rate)
	}
}

func TestCenterText(t *testing.T) {
	tests := []struct {
		text     string
		width    int
		expected string
	}{
		{"test", 10, "   test   "},
		{"hello", 5, "hello"},
		{"x", 3, " x "},
		{"longer text", 5, "longer text"}, // Text longer than width
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := CenterText(tt.text, tt.width)
			if result != tt.expected {
				t.Errorf("CenterText(%q, %d) = %q, want %q", tt.text, tt.width, result, tt.expected)
			}
		})
	}
}

func TestIsInteractiveTerminal(_ *testing.T) {
	// In a test environment, stdin is typically not a terminal
	// We can't assert the exact value since it depends on how tests are run,
	// but we can verify it doesn't panic
	_ = IsInteractiveTerminal()
}
