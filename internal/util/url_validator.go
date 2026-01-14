package util

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// URLValidationError represents a URL validation failure
type URLValidationError struct {
	URL    string
	Reason string
}

func (e *URLValidationError) Error() string {
	return fmt.Sprintf("invalid URL %q: %s", e.URL, e.Reason)
}

// ValidateBaseURL validates a URL for use as a load test target.
// It checks for:
// - Valid URL syntax
// - HTTP or HTTPS scheme only (blocks file://, ftp://, gopher://, etc.)
// - Non-empty host
// - Optional: blocks private/localhost IPs unless allowPrivateIPs is true
func ValidateBaseURL(rawURL string, allowPrivateIPs bool) error {
	if rawURL == "" {
		return &URLValidationError{URL: rawURL, Reason: "URL cannot be empty"}
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return &URLValidationError{URL: rawURL, Reason: fmt.Sprintf("invalid URL syntax: %v", err)}
	}

	// Validate scheme
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return &URLValidationError{
			URL:    rawURL,
			Reason: fmt.Sprintf("unsupported scheme %q (only http and https allowed)", parsed.Scheme),
		}
	}

	// Validate host is present
	if parsed.Host == "" {
		return &URLValidationError{URL: rawURL, Reason: "missing host"}
	}

	// Extract hostname (without port)
	hostname := parsed.Hostname()
	if hostname == "" {
		return &URLValidationError{URL: rawURL, Reason: "missing hostname"}
	}

	// Check for private/localhost IPs unless allowed
	if !allowPrivateIPs {
		if err := validateNotPrivateIP(hostname); err != nil {
			return &URLValidationError{URL: rawURL, Reason: err.Error()}
		}
	}

	return nil
}

// validateNotPrivateIP checks if the hostname resolves to a private or localhost IP
func validateNotPrivateIP(hostname string) error {
	// Check for localhost variants
	lowerHost := strings.ToLower(hostname)
	if lowerHost == "localhost" || lowerHost == "localhost.localdomain" {
		return fmt.Errorf("localhost is blocked (use --allow-private-ips for internal testing)")
	}

	// Try to parse as IP address directly
	ip := net.ParseIP(hostname)
	if ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("private IP %s is blocked (use --allow-private-ips for internal testing)", ip)
		}
		return nil
	}

	// For hostnames, resolve to IPs and check each
	// DNS errors are intentionally ignored - allow URLs that can't be resolved now
	// The actual HTTP request will fail later if the host is unreachable
	addrs, _ := net.LookupIP(hostname)
	for _, addr := range addrs {
		if isPrivateIP(addr) {
			return fmt.Errorf("hostname %s resolves to private IP %s (use --allow-private-ips for internal testing)", hostname, addr)
		}
	}

	return nil
}

// isPrivateIP checks if an IP is private, loopback, or otherwise not routable
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}

	// Check loopback (127.0.0.0/8 for IPv4, ::1 for IPv6)
	if ip.IsLoopback() {
		return true
	}

	// Check private ranges
	if ip.IsPrivate() {
		return true
	}

	// Check link-local (169.254.0.0/16 for IPv4, fe80::/10 for IPv6)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Check unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	// Additional checks for IPv4 special ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8 (current network)
		if ip4[0] == 0 {
			return true
		}
		// 100.64.0.0/10 (carrier-grade NAT)
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return true
		}
		// 192.0.0.0/24 (IETF protocol assignments)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0 {
			return true
		}
		// 192.0.2.0/24 (TEST-NET-1)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 {
			return true
		}
		// 198.51.100.0/24 (TEST-NET-2)
		if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 {
			return true
		}
		// 203.0.113.0/24 (TEST-NET-3)
		if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 {
			return true
		}
		// 224.0.0.0/4 (multicast)
		if ip4[0] >= 224 && ip4[0] <= 239 {
			return true
		}
		// 240.0.0.0/4 (reserved)
		if ip4[0] >= 240 {
			return true
		}
	}

	return false
}
