package crawler

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/vnykmshr/webstress/internal/domain"
)

// Crawler handles URL discovery and link extraction
type Crawler struct {
	baseURL        *url.URL
	discoveredURLs sync.Map
	urlPattern     *regexp.Regexp
	maxDepth       int
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
func (c *Crawler) AddURL(rawURL string, depth int, urlQueue chan domain.URLTask) bool {
	// Parse and validate URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Make relative URLs absolute
	if !parsedURL.IsAbs() {
		parsedURL = c.baseURL.ResolveReference(parsedURL)
	}

	// Only process URLs from the same host
	if parsedURL.Host != c.baseURL.Host {
		return false
	}

	// Clean URL (remove fragment, normalize)
	parsedURL.Fragment = ""
	cleanURL := parsedURL.String()

	// Check if already discovered
	if _, exists := c.discoveredURLs.LoadOrStore(cleanURL, true); exists {
		return false
	}

	// Check depth limit
	if depth > c.maxDepth {
		return false
	}

	// Add to queue
	select {
	case urlQueue <- domain.URLTask{URL: cleanURL, Depth: depth}:
		return true
	default:
		// Queue full, skip
		return false
	}
}

// GetDiscoveredCount returns the number of discovered URLs
func (c *Crawler) GetDiscoveredCount() int {
	count := 0
	c.discoveredURLs.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
