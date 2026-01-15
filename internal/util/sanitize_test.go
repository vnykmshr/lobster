package util

import (
	"strings"
	"testing"
)

func TestSanitizeURL_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No query parameters",
			input:    "http://example.com/path",
			expected: "http://example.com/path",
		},
		{
			name:     "No sensitive parameters",
			input:    "http://example.com/path?page=1&limit=10",
			expected: "http://example.com/path?page=1&limit=10",
		},
		{
			name:     "API key redacted",
			input:    "http://example.com/api?api_key=secret123",
			expected: "http://example.com/api?api_key=%5BREDACTED%5D",
		},
		{
			name:     "Token redacted",
			input:    "http://example.com/api?token=abc123&page=1",
			expected: "http://example.com/api?page=1&token=%5BREDACTED%5D",
		},
		{
			name:     "Multiple sensitive params",
			input:    "http://example.com/api?api_key=key1&password=pass1&page=1",
			expected: "http://example.com/api?api_key=%5BREDACTED%5D&page=1&password=%5BREDACTED%5D",
		},
		{
			name:     "Case insensitive matching",
			input:    "http://example.com/api?API_KEY=secret&Token=abc",
			expected: "http://example.com/api?API_KEY=%5BREDACTED%5D&Token=%5BREDACTED%5D",
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURLDefault(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSanitizeURL_AllDefaultParams(t *testing.T) {
	// Test each default sensitive parameter
	sensitiveTests := []string{
		"api_key", "apikey", "api-key",
		"token", "access_token", "auth_token",
		"password", "passwd", "pwd",
		"secret", "client_secret",
		"key", "private_key",
		"session", "session_id",
	}

	for _, param := range sensitiveTests {
		t.Run(param, func(t *testing.T) {
			input := "http://example.com/api?" + param + "=sensitive_value"
			result := SanitizeURLDefault(input)

			if !strings.Contains(result, "REDACTED") {
				t.Errorf("Parameter %s was not redacted in: %s", param, result)
			}
			if strings.Contains(result, "sensitive_value") {
				t.Errorf("Sensitive value still present in: %s", result)
			}
		})
	}
}

func TestSanitizeURL_CustomParams(t *testing.T) {
	customParams := []string{"custom_secret", "internal_token"}

	input := "http://example.com/api?custom_secret=value1&normal=value2&internal_token=value3"
	result := SanitizeURL(input, customParams)

	if !strings.Contains(result, "REDACTED") {
		t.Error("Custom sensitive parameters were not redacted")
	}
	if strings.Contains(result, "value1") || strings.Contains(result, "value3") {
		t.Error("Sensitive values still present after sanitization")
	}
	if !strings.Contains(result, "value2") {
		t.Error("Non-sensitive parameter was incorrectly redacted")
	}
}

func TestSanitizeURL_PreservesStructure(t *testing.T) {
	input := "https://user:pass@example.com:8080/path?api_key=secret&page=1#fragment"
	result := SanitizeURLDefault(input)

	// Check that URL structure is preserved
	if !strings.HasPrefix(result, "https://") {
		t.Error("Scheme was not preserved")
	}
	if !strings.Contains(result, "example.com:8080") {
		t.Error("Host and port were not preserved")
	}
	if !strings.Contains(result, "/path") {
		t.Error("Path was not preserved")
	}
	if !strings.Contains(result, "page=1") {
		t.Error("Non-sensitive query parameter was not preserved")
	}
	if !strings.Contains(result, "#fragment") {
		t.Error("Fragment was not preserved")
	}
	if !strings.Contains(result, "REDACTED") {
		t.Error("Sensitive parameter was not redacted")
	}
}

func TestSanitizeURL_InvalidURL(t *testing.T) {
	// Invalid URLs should be returned as-is rather than causing errors
	invalid := "not a valid url with spaces"
	result := SanitizeURLDefault(invalid)

	if result != invalid {
		t.Errorf("Invalid URL should be returned as-is, got: %s", result)
	}
}

func TestSanitizeURL_ComplexQueryString(t *testing.T) {
	input := "http://example.com/api?param1=value1&api_key=secret&param2=value2&token=abc&param3=value3"
	result := SanitizeURLDefault(input)

	// Verify sensitive params are redacted
	if !strings.Contains(result, "api_key=%5BREDACTED%5D") {
		t.Error("api_key was not properly redacted")
	}
	if !strings.Contains(result, "token=%5BREDACTED%5D") {
		t.Error("token was not properly redacted")
	}

	// Verify non-sensitive params are preserved
	if !strings.Contains(result, "param1=value1") {
		t.Error("param1 was incorrectly redacted")
	}
	if !strings.Contains(result, "param2=value2") {
		t.Error("param2 was incorrectly redacted")
	}
	if !strings.Contains(result, "param3=value3") {
		t.Error("param3 was incorrectly redacted")
	}
}

func TestSanitizeURL_EmptyParamsList(t *testing.T) {
	// When empty params list is provided, should use defaults
	input := "http://example.com/api?api_key=secret"
	result := SanitizeURL(input, []string{})

	if !strings.Contains(result, "REDACTED") {
		t.Error("Should use default sensitive params when empty list provided")
	}
}

func TestSanitizeURL_NoSensitiveMatch(t *testing.T) {
	input := "http://example.com/api?page=1&limit=10&sort=desc"
	result := SanitizeURLDefault(input)

	// URL should be unchanged when no sensitive params found
	if result != input {
		t.Errorf("URL without sensitive params should be unchanged, got: %s", result)
	}
}

func TestSanitizeError_IPv4(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple IP",
			input:    "connection refused to 192.168.1.100",
			expected: "connection refused to [IP]",
		},
		{
			name:     "IP with port",
			input:    "dial tcp 10.0.0.1:8080: connection refused",
			expected: "dial tcp [IP]:8080: connection refused",
		},
		{
			name:     "multiple IPs",
			input:    "forwarded from 172.16.0.1 to 172.16.0.2",
			expected: "forwarded from [IP] to [IP]",
		},
		{
			name:     "loopback",
			input:    "connect 127.0.0.1:3000 refused",
			expected: "connect [IP]:3000 refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_InternalHostnames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "localhost",
			input:    "cannot connect to localhost:8080",
			expected: "cannot connect to [internal-host]:8080",
		},
		{
			name:     "internal subdomain",
			input:    "error from internal.company.com",
			expected: "error from [internal-host]",
		},
		{
			name:     "staging server",
			input:    "staging.api.example.com returned 500",
			expected: "[internal-host] returned 500",
		},
		{
			name:     "dev environment",
			input:    "dev-server.local failed",
			expected: "[internal-host]-server.[internal-host] failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_FilePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "deep path",
			input:    "file not found: /var/log/app/errors.log",
			expected: "file not found: [path]",
		},
		{
			name:     "short path preserved",
			input:    "error in /var/log",
			expected: "error in /var/log",
		},
		{
			name:     "home directory",
			input:    "loading /home/user/config/settings.json",
			expected: "loading [path]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_Empty(t *testing.T) {
	result := SanitizeError("")
	if result != "" {
		t.Errorf("SanitizeError(\"\") = %q, want \"\"", result)
	}
}

func TestSanitizeError_NoSensitiveData(t *testing.T) {
	input := "request timeout after 30s"
	result := SanitizeError(input)
	if result != input {
		t.Errorf("SanitizeError() = %q, want %q", result, input)
	}
}

func TestSanitizeErrorForDisplay_Verbose(t *testing.T) {
	input := "connection to 192.168.1.1:8080 failed"
	result := SanitizeErrorForDisplay(input, true)

	// In verbose mode, original error should be returned
	if result != input {
		t.Errorf("SanitizeErrorForDisplay(verbose=true) = %q, want %q", result, input)
	}
}

func TestSanitizeErrorForDisplay_NonVerbose(t *testing.T) {
	input := "connection to 192.168.1.1:8080 failed"
	expected := "connection to [IP]:8080 failed"
	result := SanitizeErrorForDisplay(input, false)

	if result != expected {
		t.Errorf("SanitizeErrorForDisplay(verbose=false) = %q, want %q", result, expected)
	}
}
