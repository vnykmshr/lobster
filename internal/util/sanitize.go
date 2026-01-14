// Package util provides utility functions for the Lobster load testing tool.
package util

import (
	"net/url"
	"regexp"
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

// ipv4Pattern matches IPv4 addresses
var ipv4Pattern = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})(:\d+)?\b`)

// ipv6Pattern matches IPv6 addresses (simplified - common formats)
var ipv6Pattern = regexp.MustCompile(`\[([0-9a-fA-F:]+)\](:\d+)?`)

// internalHostPattern matches common internal hostname patterns
var internalHostPattern = regexp.MustCompile(`\b(localhost|internal|private|staging|dev|corp|local)(\.[a-zA-Z0-9.-]+)?\b`)

// pathPattern matches file paths that might reveal internal structure
var pathPattern = regexp.MustCompile(`(/[a-zA-Z0-9._-]+){3,}`)

// SanitizeError redacts potentially sensitive information from error messages.
// This includes IP addresses, internal hostnames, and file paths.
// Use verbose mode for full error details during debugging.
func SanitizeError(errMsg string) string {
	if errMsg == "" {
		return ""
	}

	result := errMsg

	// Redact IPv4 addresses (keep port info for context)
	result = ipv4Pattern.ReplaceAllStringFunc(result, func(match string) string {
		if strings.Contains(match, ":") {
			parts := strings.Split(match, ":")
			return "[IP]:" + parts[1]
		}
		return "[IP]"
	})

	// Redact IPv6 addresses
	result = ipv6Pattern.ReplaceAllStringFunc(result, func(match string) string {
		if strings.Contains(match, "]:") {
			parts := strings.Split(match, "]:")
			return "[IPv6]:" + parts[1]
		}
		return "[IPv6]"
	})

	// Redact internal hostnames
	result = internalHostPattern.ReplaceAllString(result, "[internal-host]")

	// Redact deep file paths (3+ levels)
	result = pathPattern.ReplaceAllString(result, "[path]")

	return result
}

// SanitizeErrorForDisplay returns a user-friendly error message.
// If verbose is true, returns the full error. Otherwise sanitizes it.
func SanitizeErrorForDisplay(errMsg string, verbose bool) string {
	if verbose {
		return errMsg
	}
	return SanitizeError(errMsg)
}
