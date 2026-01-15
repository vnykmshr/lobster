// Package crawler provides URL discovery and link extraction for web crawling.
package crawler

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/vnykmshr/lobster/internal/domain"
)

// Crawler handles URL discovery and link extraction
type Crawler struct {
	discoveredURLs sync.Map
	baseURL        *url.URL
	urlPattern     *regexp.Regexp
	maxDepth       int
	discoveredCnt  atomic.Int64 // O(1) counter for discovered URLs
	droppedCnt     atomic.Int64 // Counter for URLs dropped due to queue full
}

// New creates a new crawler
func New(baseURL string, maxDepth int) (*Crawler, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Crawler{
		baseURL:    parsedURL,
		urlPattern: regexp.MustCompile(`href=["']([^"']+)["']`),
		maxDepth:   maxDepth,
	}, nil
}

// ExtractLinks extracts all links from HTML body
func (c *Crawler) ExtractLinks(body string) []string {
	matches := c.urlPattern.FindAllStringSubmatch(body, -1)
	links := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			link := strings.TrimSpace(match[1])
			if c.isValidLink(link) {
				// Decode HTML entities (e.g., &amp; -> &, &quot; -> ")
				decodedLink := html.UnescapeString(link)
				links = append(links, decodedLink)
			}
		}
	}

	return links
}

// isValidLink checks if a link should be followed
func (c *Crawler) isValidLink(link string) bool {
	if link == "" {
		return false
	}

	// Skip javascript:, mailto:, and fragment-only links
	if strings.HasPrefix(link, "javascript:") ||
		strings.HasPrefix(link, "mailto:") ||
		strings.HasPrefix(link, "#") {
		return false
	}

	return true
}

// AddURL adds a URL to the discovery queue if it's valid and not already discovered
func (c *Crawler) AddURL(rawURL string, depth int, urlQueue chan<- domain.URLTask) domain.AddURLResult {
	// Parse and validate URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return domain.AddURLResult{Added: false, Reason: domain.AddURLParseError}
	}

	// Make relative URLs absolute
	if !parsedURL.IsAbs() {
		parsedURL = c.baseURL.ResolveReference(parsedURL)
	}

	// Only process URLs from the same host
	if parsedURL.Host != c.baseURL.Host {
		return domain.AddURLResult{Added: false, Reason: domain.AddURLInvalidHost}
	}

	// Clean URL (remove fragment, normalize)
	parsedURL.Fragment = ""
	cleanURL := parsedURL.String()

	// Check if already discovered
	if _, exists := c.discoveredURLs.LoadOrStore(cleanURL, true); exists {
		return domain.AddURLResult{Added: false, Reason: domain.AddURLDuplicate}
	}

	// Track discovered URL count (O(1) instead of iterating sync.Map)
	c.discoveredCnt.Add(1)

	// Check depth limit
	if depth > c.maxDepth {
		return domain.AddURLResult{Added: false, Reason: domain.AddURLDepthExceeded}
	}

	// Add to queue
	select {
	case urlQueue <- domain.URLTask{URL: cleanURL, Depth: depth}:
		return domain.AddURLResult{Added: true, Reason: domain.AddURLSuccess}
	default:
		// Queue full - track dropped URLs for visibility
		c.droppedCnt.Add(1)
		return domain.AddURLResult{Added: false, Reason: domain.AddURLQueueFull}
	}
}

// GetDiscoveredCount returns the number of discovered URLs (O(1) operation)
func (c *Crawler) GetDiscoveredCount() int {
	return int(c.discoveredCnt.Load())
}

// GetDroppedCount returns the number of URLs dropped due to queue full
func (c *Crawler) GetDroppedCount() int {
	return int(c.droppedCnt.Load())
}
