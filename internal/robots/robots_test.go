package robots

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParse_BasicDisallow(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /admin/
Disallow: /private/
`
	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	tests := []struct {
		url     string
		allowed bool
	}{
		{"/", true},
		{"/index.html", true},
		{"/admin/", false},
		{"/admin/users", false},
		{"/private/data", false},
		{"/public/page", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			parser.robotsTxtFound = true // Simulate that robots.txt was found
			allowed := parser.IsAllowed(tt.url)
			if allowed != tt.allowed {
				t.Errorf("IsAllowed(%s) = %v, want %v", tt.url, allowed, tt.allowed)
			}
		})
	}
}

func TestParse_UserAgentMatching(t *testing.T) {
	robotsTxt := `
User-agent: Googlebot
Disallow: /private/

User-agent: *
Disallow: /admin/
`
	parser := New("MyBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	parser.robotsTxtFound = true

	// MyBot should match wildcard (*), not Googlebot
	if !parser.IsAllowed("/private/test") {
		t.Error("Expected /private/ to be allowed for MyBot")
	}

	if parser.IsAllowed("/admin/test") {
		t.Error("Expected /admin/ to be disallowed for wildcard")
	}
}

func TestParse_AllowRules(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /admin/
Allow: /admin/public/
`
	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	parser.robotsTxtFound = true

	// Allow should override Disallow
	if !parser.IsAllowed("/admin/public/page.html") {
		t.Error("Expected /admin/public/ to be allowed (Allow rule)")
	}

	if parser.IsAllowed("/admin/private/page.html") {
		t.Error("Expected /admin/private/ to be disallowed")
	}
}

func TestParse_Wildcards(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /*.php
Disallow: /temp*
`
	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	parser.robotsTxtFound = true

	// Test wildcard patterns
	if parser.IsAllowed("/index.php") {
		t.Error("Expected /*.php to disallow /index.php")
	}

	if parser.IsAllowed("/temp/file") {
		t.Error("Expected /temp* to disallow /temp/file")
	}

	if parser.IsAllowed("/temporary/data") {
		t.Error("Expected /temp* to disallow /temporary/data")
	}
}

func TestParse_CrawlDelay(t *testing.T) {
	robotsTxt := `
User-agent: *
Crawl-delay: 2.5
Disallow: /admin/
`
	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	delay := parser.GetCrawlDelay()
	expected := 2500 * time.Millisecond // 2.5 seconds
	if delay != expected {
		t.Errorf("GetCrawlDelay() = %v, want %v", delay, expected)
	}
}

func TestParse_Comments(t *testing.T) {
	robotsTxt := `
# This is a comment
User-agent: *
Disallow: /admin/  # inline comment should be ignored
# Another comment
Disallow: /private/
`
	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	parser.robotsTxtFound = true

	if parser.IsAllowed("/admin/page") {
		t.Error("Expected /admin/ to be disallowed despite comments")
	}
}

func TestParse_EmptyDisallow(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow:
`
	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	parser.robotsTxtFound = true

	// Empty Disallow means allow everything
	if !parser.IsAllowed("/anything") {
		t.Error("Expected all paths to be allowed with empty Disallow")
	}
}

func TestParse_DisallowAll(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /
`
	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(robotsTxt))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	parser.robotsTxtFound = true

	// Disallow: / means block everything
	if parser.IsAllowed("/") {
		t.Error("Expected / to be disallowed")
	}

	if parser.IsAllowed("/anything") {
		t.Error("Expected /anything to be disallowed")
	}
}

func TestFetchAndParse_NotFound(t *testing.T) {
	// Server that returns 404 for robots.txt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	parser := New("TestBot/1.0")
	ctx := context.Background()

	err := parser.FetchAndParse(ctx, server.URL)
	if err != nil {
		t.Fatalf("FetchAndParse failed: %v", err)
	}

	// robots.txt not found means allow everything
	if !parser.IsAllowed("/anything") {
		t.Error("Expected all paths to be allowed when robots.txt not found")
	}

	if parser.RobotsTxtFound() {
		t.Error("Expected RobotsTxtFound() to be false")
	}
}

func TestFetchAndParse_Success(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /admin/
Allow: /admin/public/
`
	// Server that returns robots.txt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(robotsTxt))
		}
	}))
	defer server.Close()

	parser := New("TestBot/1.0")
	ctx := context.Background()

	err := parser.FetchAndParse(ctx, server.URL)
	if err != nil {
		t.Fatalf("FetchAndParse failed: %v", err)
	}

	if !parser.RobotsTxtFound() {
		t.Error("Expected RobotsTxtFound() to be true")
	}

	// Check that rules are applied
	if parser.IsAllowed("/admin/private") {
		t.Error("Expected /admin/private to be disallowed")
	}

	if !parser.IsAllowed("/admin/public/page") {
		t.Error("Expected /admin/public/ to be allowed")
	}
}

func TestFetchAndParse_ServerError(t *testing.T) {
	// Server that returns 500 for robots.txt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	parser := New("TestBot/1.0")
	ctx := context.Background()

	err := parser.FetchAndParse(ctx, server.URL)
	if err == nil {
		t.Error("Expected error when server returns 500")
	}

	// Should be conservative and disallow everything
	if parser.IsAllowed("/anything") {
		t.Error("Expected all paths to be disallowed after server error")
	}
}

func TestIsAllowed_NoRobotsTxt(t *testing.T) {
	parser := New("TestBot/1.0")
	// Don't fetch robots.txt

	// When no robots.txt, allow everything
	if !parser.IsAllowed("/anything") {
		t.Error("Expected all paths to be allowed when no robots.txt")
	}

	if !parser.IsAllowed("/admin/secret") {
		t.Error("Expected all paths to be allowed when no robots.txt")
	}
}

func TestIsAllowed_InvalidURL(t *testing.T) {
	robotsTxt := `
User-agent: *
Disallow: /admin/
`
	parser := New("TestBot/1.0")
	_ = parser.Parse(strings.NewReader(robotsTxt))
	parser.robotsTxtFound = true

	// Invalid URL should be disallowed (conservative)
	if parser.IsAllowed("://invalid-url") {
		t.Error("Expected invalid URL to be disallowed")
	}
}

func TestMatchesPath(t *testing.T) {
	tests := []struct {
		urlPath    string
		robotsPath string
		matches    bool
	}{
		// Basic prefix matching
		{"/admin/users", "/admin/", true},
		{"/admin", "/admin/", false},
		{"/public/page", "/admin/", false},

		// Wildcard at end (suffix matching)
		{"/temp/file", "/temp*", true},
		{"/temporary/data", "/temp*", true},
		{"/test", "/temp*", false},
		{"/admin", "/admin*", true},
		{"/admin/users", "/admin*", true},
		{"/administrator", "/admin*", true},

		// Wildcard matching .php anywhere in path
		{"/data.php", "/*.php", true},
		{"/dir/file.php", "/*.php", true},
		{"/deep/nested/path/script.php", "/*.php", true},
		{"/data.html", "/*.php", false},
		{"/index.php/extra", "/*.php", true}, // .php is in path

		// Middle wildcard
		{"/admin/users/123", "/admin/*/123", true},
		{"/admin/x/123", "/admin/*/123", true},
		{"/admin/123", "/admin/*/123", false}, // needs something between

		// $ anchor (end of URL)
		{"/index.html", "/*.html$", true},
		{"/index.html/extra", "/*.html$", false}, // doesn't end with .html
		{"/page", "/page$", true},
		{"/page/extra", "/page$", false},

		// Multiple wildcards
		{"/a/b/c", "/a/*/c", true},
		{"/a/x/y/c", "/a/*/c", true}, // * matches "x/y"

		// Edge cases
		{"", "/", false},       // empty path doesn't match /
		{"/", "/", true},       // root matches root
		{"/foo", "", false},    // empty pattern matches nothing
		{"/path", "/path", true}, // exact match
	}

	for _, tt := range tests {
		t.Run(tt.urlPath+"_"+tt.robotsPath, func(t *testing.T) {
			result := matchesPath(tt.urlPath, tt.robotsPath)
			if result != tt.matches {
				t.Errorf("matchesPath(%s, %s) = %v, want %v",
					tt.urlPath, tt.robotsPath, result, tt.matches)
			}
		})
	}
}

func TestParse_SizeLimit(t *testing.T) {
	// Create a robots.txt larger than the limit
	largeContent := "User-agent: *\nDisallow: /test\n"
	// Repeat until we exceed maxRobotsTxtSize (1MB)
	for len(largeContent) < 2*1024*1024 {
		largeContent += "Disallow: /path" + strings.Repeat("x", 1000) + "\n"
	}

	parser := New("TestBot/1.0")
	err := parser.Parse(strings.NewReader(largeContent))

	// Should not error - just truncate
	if err != nil {
		t.Errorf("Parse with large content should not error, got: %v", err)
	}

	// The first rule should still be parsed
	parser.robotsTxtFound = true
	if parser.IsAllowed("/test") {
		t.Error("Expected /test to be disallowed")
	}
}

func TestMatchesPath_MalformedPatterns(t *testing.T) {
	// Test that malformed patterns don't crash
	tests := []struct {
		urlPath    string
		robotsPath string
	}{
		{"/path", "***"},
		{"/path", "*/*/*/*"},
		{"/path", "/$$$"},
		{"/path", "/*$*"},
	}

	for _, tt := range tests {
		t.Run(tt.robotsPath, func(_ *testing.T) {
			// Should not panic
			_ = matchesPath(tt.urlPath, tt.robotsPath)
		})
	}
}
