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
	defer resp.Body.Close()

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

// Parse parses robots.txt content from a reader
func (p *Parser) Parse(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
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

// matchesPath checks if a URL path matches a robots.txt path pattern
func matchesPath(urlPath, robotsPath string) bool {
	// Handle wildcard patterns
	if strings.Contains(robotsPath, "*") {
		// Pattern with wildcard at the end: /temp* matches /temp, /temporary, etc.
		if strings.HasSuffix(robotsPath, "*") {
			prefix := strings.TrimSuffix(robotsPath, "*")
			return strings.HasPrefix(urlPath, prefix)
		}

		// Pattern with wildcard at the start: /*.php matches /index.php, /data.php, etc.
		if strings.HasPrefix(robotsPath, "/") && strings.Contains(robotsPath, "*") {
			// Split on wildcard
			parts := strings.SplitN(robotsPath, "*", 2)
			before := parts[0]
			after := ""
			if len(parts) > 1 {
				after = parts[1]
			}

			// Check if URL starts with the part before * and ends with the part after *
			if !strings.HasPrefix(urlPath, before) {
				return false
			}
			if after != "" && !strings.HasSuffix(urlPath, after) {
				return false
			}
			return true
		}

		// For other complex wildcards, do simple contains check
		pattern := strings.ReplaceAll(robotsPath, "*", "")
		return strings.Contains(urlPath, pattern)
	}

	// Exact prefix match
	return strings.HasPrefix(urlPath, robotsPath)
}
