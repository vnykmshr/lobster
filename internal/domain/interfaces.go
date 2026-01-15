// Package domain defines core domain types and interfaces for the load testing tool.
package domain

import "context"

// URLCrawler defines the interface for URL discovery and link extraction.
// Implementations handle URL validation, deduplication, and queue management.
type URLCrawler interface {
	// ExtractLinks parses HTML body and returns valid links found.
	ExtractLinks(body string) []string

	// AddURL adds a URL to the discovery queue if valid and not already discovered.
	// Returns an AddURLResult with the outcome and reason.
	AddURL(rawURL string, depth int, queue chan<- URLTask) AddURLResult

	// GetDiscoveredCount returns the total number of unique URLs discovered.
	GetDiscoveredCount() int

	// GetDroppedCount returns the number of URLs dropped due to queue overflow.
	GetDroppedCount() int
}

// RobotsChecker defines the interface for robots.txt compliance checking.
// Implementations parse robots.txt and enforce path-based access rules.
type RobotsChecker interface {
	// FetchAndParse fetches and parses robots.txt from the given base URL.
	// Returns an error if the robots.txt cannot be fetched or parsed.
	FetchAndParse(ctx context.Context, baseURL string) error

	// IsAllowed returns true if the given URL path is allowed by robots.txt rules.
	IsAllowed(urlPath string) bool

	// RobotsTxtFound returns true if robots.txt was found and parsed successfully.
	RobotsTxtFound() bool
}

// RateLimiter defines the interface for rate limiting concurrent requests.
// Implementations control request throughput using token bucket or similar algorithms.
type RateLimiter interface {
	// Wait blocks until a token is available or the context is canceled.
	// Returns an error if the context is canceled or times out.
	Wait(ctx context.Context) error
}
