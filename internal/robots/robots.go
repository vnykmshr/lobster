// Package robots provides robots.txt parsing and compliance checking.
package robots

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Parser handles robots.txt parsing and URL compliance checking
type Parser struct {
	userAgent      string
	disallowPaths  []string
	allowPaths     []string
	crawlDelay     time.Duration
	robotsTxtFound bool
}

// New creates a new robots.txt parser
func New(userAgent string) *Parser {
	return &Parser{
		userAgent:      userAgent,
		disallowPaths:  make([]string, 0),
		allowPaths:     make([]string, 0),
		robotsTxtFound: false,
	}
}

// FetchAndParse fetches and parses robots.txt from the given base URL
func (p *Parser) FetchAndParse(ctx context.Context, baseURL string) error {
	// Parse base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL %s: %w\nUse format: http://example.com or https://example.com", baseURL, err)
	}

	// Construct robots.txt URL
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", p.userAgent)

	// Fetch robots.txt
	resp, err := client.Do(req)
	if err != nil {
		// If robots.txt doesn't exist or network error, allow crawling
		return nil
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// If robots.txt not found (404), allow crawling
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	// If we got robots.txt successfully
	if resp.StatusCode == http.StatusOK {
		p.robotsTxtFound = true
		return p.Parse(resp.Body)
	}

	// For other status codes (403, 500, etc.), be conservative and block crawling
	if resp.StatusCode >= 400 {
		p.robotsTxtFound = true
		p.disallowPaths = append(p.disallowPaths, "/") // Disallow everything
		return fmt.Errorf("robots.txt returned status %d - disallowing all paths", resp.StatusCode)
	}

	return nil
}

// Maximum size of robots.txt to parse (1MB)
const maxRobotsTxtSize = 1 * 1024 * 1024

// Parse parses robots.txt content from a reader
func (p *Parser) Parse(reader io.Reader) error {
	// Limit reading to prevent memory exhaustion from malicious robots.txt
	limitedReader := io.LimitReader(reader, maxRobotsTxtSize)
	scanner := bufio.NewScanner(limitedReader)
	inMatchingUserAgent := false
	foundAnyUserAgent := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove inline comments
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
			if line == "" {
				continue
			}
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		field := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch field {
		case "user-agent":
			foundAnyUserAgent = true
			// Check if this user-agent matches ours
			if value == "*" || strings.Contains(strings.ToLower(p.userAgent), strings.ToLower(value)) {
				inMatchingUserAgent = true
			} else {
				inMatchingUserAgent = false
			}

		case "disallow":
			if inMatchingUserAgent && value != "" {
				p.disallowPaths = append(p.disallowPaths, value)
			}

		case "allow":
			if inMatchingUserAgent && value != "" {
				p.allowPaths = append(p.allowPaths, value)
			}

		case "crawl-delay":
			if inMatchingUserAgent {
				var delay float64
				if _, err := fmt.Sscanf(value, "%f", &delay); err == nil {
					p.crawlDelay = time.Duration(delay * float64(time.Second))
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading robots.txt: %w", err)
	}

	// If no rules found for our user-agent and wildcard exists, use wildcard rules
	if !foundAnyUserAgent {
		// No robots.txt rules at all - allow crawling
		return nil
	}

	return nil
}

// IsAllowed checks if the given URL path is allowed by robots.txt
func (p *Parser) IsAllowed(urlPath string) bool {
	// If no robots.txt was found, allow all paths
	if !p.robotsTxtFound {
		return true
	}

	// If no rules were specified, allow all paths
	if len(p.disallowPaths) == 0 && len(p.allowPaths) == 0 {
		return true
	}

	// Parse URL to get path
	parsedURL, err := url.Parse(urlPath)
	if err != nil {
		// If we can't parse, be conservative and disallow
		return false
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// Check Allow rules first (more specific)
	for _, allowPath := range p.allowPaths {
		if matchesPath(path, allowPath) {
			return true
		}
	}

	// Then check Disallow rules
	for _, disallowPath := range p.disallowPaths {
		if matchesPath(path, disallowPath) {
			return false
		}
	}

	// If no rules matched, allow by default
	return true
}

// GetCrawlDelay returns the crawl delay specified in robots.txt
func (p *Parser) GetCrawlDelay() time.Duration {
	return p.crawlDelay
}

// RobotsTxtFound returns whether robots.txt was found and parsed
func (p *Parser) RobotsTxtFound() bool {
	return p.robotsTxtFound
}

// matchesPath checks if a URL path matches a robots.txt path pattern.
// Supports wildcards per Google's robots.txt specification:
// - * matches any sequence of characters
// - $ matches end of URL path
// - Patterns are matched against URL path (case-sensitive)
func matchesPath(urlPath, robotsPath string) bool {
	// Empty pattern matches nothing
	if robotsPath == "" {
		return false
	}

	// Handle $ anchor at end (matches end of URL)
	if strings.HasSuffix(robotsPath, "$") {
		// Remove $ and check if pattern matches exactly
		pattern := strings.TrimSuffix(robotsPath, "$")
		if strings.Contains(pattern, "*") {
			return matchWildcard(urlPath, pattern, true)
		}
		return urlPath == pattern
	}

	// Handle wildcard patterns
	if strings.Contains(robotsPath, "*") {
		return matchWildcard(urlPath, robotsPath, false)
	}

	// Exact prefix match (standard robots.txt behavior)
	return strings.HasPrefix(urlPath, robotsPath)
}

// matchWildcard matches a path against a pattern with wildcards.
// exactEnd means the pattern must match the end of urlPath ($ anchor).
func matchWildcard(urlPath, pattern string, exactEnd bool) bool {
	// Split pattern on wildcards
	parts := strings.Split(pattern, "*")

	// Empty parts from consecutive wildcards or leading/trailing wildcards
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}

		// Find this part in the remaining path
		idx := strings.Index(urlPath[pos:], part)
		if idx == -1 {
			return false
		}

		// For the first part, it must match at the start (prefix matching)
		if i == 0 && idx != 0 {
			return false
		}

		pos += idx + len(part)
	}

	// If exactEnd ($), the pattern must have consumed the entire path
	if exactEnd {
		return pos == len(urlPath)
	}

	return true
}
