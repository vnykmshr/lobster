// Package util provides utility functions for the Lobster load testing tool.
package util

import (
	"net/url"
	"strings"
)

// DefaultSensitiveParams contains common sensitive query parameter names
var DefaultSensitiveParams = []string{
	"api_key", "apikey", "api-key",
	"token", "access_token", "auth_token", "auth",
	"password", "passwd", "pwd",
	"secret", "client_secret",
	"key", "private_key",
	"authorization",
	"session", "session_id", "sessionid",
	"credential", "credentials",
}

// SanitizeURL redacts sensitive query parameters from a URL for safe logging.
// Parameters matching the sensitive list (case-insensitive) are replaced with "[REDACTED]".
func SanitizeURL(rawURL string, sensitiveParams []string) string {
	if rawURL == "" {
		return ""
	}

	// Use default params if none provided
	if len(sensitiveParams) == 0 {
		sensitiveParams = DefaultSensitiveParams
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		// If URL parsing fails, return as-is (better than losing the URL entirely)
		return rawURL
	}

	// If no query parameters, return as-is
	if parsedURL.RawQuery == "" {
		return rawURL
	}

	// Parse query parameters
	query := parsedURL.Query()
	modified := false

	// Create case-insensitive sensitive params map for faster lookup
	sensitiveMap := make(map[string]bool, len(sensitiveParams))
	for _, param := range sensitiveParams {
		sensitiveMap[strings.ToLower(param)] = true
	}

	// Redact sensitive parameters
	for key := range query {
		if sensitiveMap[strings.ToLower(key)] {
			query.Set(key, "[REDACTED]")
			modified = true
		}
	}

	// If nothing was redacted, return original URL
	if !modified {
		return rawURL
	}

	// Reconstruct URL with sanitized query
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

// SanitizeURLDefault redacts sensitive parameters using the default list
func SanitizeURLDefault(rawURL string) string {
	return SanitizeURL(rawURL, nil)
}
