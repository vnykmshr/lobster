package crawler

import (
	"testing"

	"github.com/vnykmshr/lobster/internal/domain"
)

func TestNew_Success(t *testing.T) {
	c, err := New("http://example.com", 3)
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if c == nil {
		t.Fatal("Expected non-nil crawler")
	}
	if c.maxDepth != 3 {
		t.Errorf("Expected maxDepth 3, got %d", c.maxDepth)
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New(":", 3)
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestExtractLinks_BasicHTML(t *testing.T) {
	c, _ := New("http://example.com", 3)

	html := `<html>
		<body>
			<a href="/page1">Page 1</a>
			<a href="/page2">Page 2</a>
			<a href="http://example.com/page3">Page 3</a>
		</body>
	</html>`

	links := c.ExtractLinks(html)

	if len(links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(links))
	}

	expectedLinks := map[string]bool{
		"/page1":                   true,
		"/page2":                   true,
		"http://example.com/page3": true,
	}

	for _, link := range links {
		if !expectedLinks[link] {
			t.Errorf("Unexpected link: %s", link)
		}
	}
}

func TestExtractLinks_DoubleQuotes(t *testing.T) {
	c, _ := New("http://example.com", 3)

	html := `<a href="http://example.com/double">Link</a>`
	links := c.ExtractLinks(html)

	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}
	if links[0] != "http://example.com/double" {
		t.Errorf("Expected 'http://example.com/double', got '%s'", links[0])
	}
}

func TestExtractLinks_SingleQuotes(t *testing.T) {
	c, _ := New("http://example.com", 3)

	html := `<a href='http://example.com/single'>Link</a>`
	links := c.ExtractLinks(html)

	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}
	if links[0] != "http://example.com/single" {
		t.Errorf("Expected 'http://example.com/single', got '%s'", links[0])
	}
}

func TestExtractLinks_HTMLEntities(t *testing.T) {
	c, _ := New("http://example.com", 3)

	html := `<a href="/path?param1=value&amp;param2=value">Link</a>`
	links := c.ExtractLinks(html)

	if len(links) != 1 {
		t.Fatalf("Expected 1 link, got %d", len(links))
	}

	// Should decode &amp; to &
	expected := "/path?param1=value&param2=value"
	if links[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, links[0])
	}
}

func TestExtractLinks_InvalidLinks(t *testing.T) {
	c, _ := New("http://example.com", 3)

	html := `<html>
		<body>
			<a href="javascript:void(0)">JS Link</a>
			<a href="mailto:test@example.com">Email</a>
			<a href="#">Fragment only</a>
			<a href="">Empty</a>
			<a href="   ">Whitespace</a>
		</body>
	</html>`

	links := c.ExtractLinks(html)

	if len(links) != 0 {
		t.Errorf("Expected 0 valid links, got %d: %v", len(links), links)
	}
}

func TestIsValidLink(t *testing.T) {
	c, _ := New("http://example.com", 3)

	tests := []struct {
		name  string
		link  string
		valid bool
	}{
		{"valid http", "http://example.com/page", true},
		{"valid relative", "/page", true},
		{"valid relative with query", "/page?query=value", true},
		{"javascript", "javascript:void(0)", false},
		{"mailto", "mailto:test@example.com", false},
		{"fragment only", "#section", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.isValidLink(tt.link)
			if result != tt.valid {
				t.Errorf("isValidLink(%s) = %v, want %v", tt.link, result, tt.valid)
			}
		})
	}
}

func TestAddURL_SameDomain(t *testing.T) {
	c, _ := New("http://example.com", 3)
	urlQueue := make(chan domain.URLTask, 10)

	added := c.AddURL("http://example.com/page1", 1, urlQueue)
	if !added {
		t.Error("Expected URL to be added")
	}

	if len(urlQueue) != 1 {
		t.Errorf("Expected 1 URL in queue, got %d", len(urlQueue))
	}

	task := <-urlQueue
	if task.URL != "http://example.com/page1" {
		t.Errorf("Expected URL 'http://example.com/page1', got '%s'", task.URL)
	}
	if task.Depth != 1 {
		t.Errorf("Expected Depth 1, got %d", task.Depth)
	}
}

func TestAddURL_DifferentDomain(t *testing.T) {
	c, _ := New("http://example.com", 3)
	urlQueue := make(chan domain.URLTask, 10)

	added := c.AddURL("http://different.com/page", 1, urlQueue)
	if added {
		t.Error("Expected URL from different domain to be rejected")
	}

	if len(urlQueue) != 0 {
		t.Errorf("Expected 0 URLs in queue, got %d", len(urlQueue))
	}
}

func TestAddURL_RelativeURL(t *testing.T) {
	c, _ := New("http://example.com", 3)
	urlQueue := make(chan domain.URLTask, 10)

	added := c.AddURL("/page2", 1, urlQueue)
	if !added {
		t.Error("Expected relative URL to be added")
	}

	task := <-urlQueue
	// Should be converted to absolute
	if task.URL != "http://example.com/page2" {
		t.Errorf("Expected absolute URL 'http://example.com/page2', got '%s'", task.URL)
	}
}

func TestAddURL_Deduplication(t *testing.T) {
	c, _ := New("http://example.com", 3)
	urlQueue := make(chan domain.URLTask, 10)

	// Add same URL twice
	added1 := c.AddURL("http://example.com/page", 1, urlQueue)
	added2 := c.AddURL("http://example.com/page", 1, urlQueue)

	if !added1 {
		t.Error("Expected first URL to be added")
	}
	if added2 {
		t.Error("Expected duplicate URL to be rejected")
	}

	if len(urlQueue) != 1 {
		t.Errorf("Expected 1 URL in queue (deduplicated), got %d", len(urlQueue))
	}
}

func TestAddURL_FragmentRemoval(t *testing.T) {
	c, _ := New("http://example.com", 3)
	urlQueue := make(chan domain.URLTask, 10)

	// Add URLs with fragments - they should be treated as the same URL
	added1 := c.AddURL("http://example.com/page#section1", 1, urlQueue)
	added2 := c.AddURL("http://example.com/page#section2", 1, urlQueue)

	if !added1 {
		t.Error("Expected first URL to be added")
	}
	if added2 {
		t.Error("Expected URL with different fragment to be deduplicated")
	}

	task := <-urlQueue
	// Fragment should be removed
	if task.URL != "http://example.com/page" {
		t.Errorf("Expected URL without fragment 'http://example.com/page', got '%s'", task.URL)
	}
}

func TestAddURL_MaxDepthExceeded(t *testing.T) {
	c, _ := New("http://example.com", 2) // max depth is 2
	urlQueue := make(chan domain.URLTask, 10)

	// Try to add URL at depth 3 (exceeds maxDepth)
	added := c.AddURL("http://example.com/page", 3, urlQueue)
	if added {
		t.Error("Expected URL exceeding max depth to be rejected")
	}

	if len(urlQueue) != 0 {
		t.Errorf("Expected 0 URLs in queue, got %d", len(urlQueue))
	}
}

func TestGetDiscoveredCount(t *testing.T) {
	c, _ := New("http://example.com", 3)
	urlQueue := make(chan domain.URLTask, 10)

	initialCount := c.GetDiscoveredCount()
	if initialCount != 0 {
		t.Errorf("Expected initial count 0, got %d", initialCount)
	}

	c.AddURL("http://example.com/page1", 1, urlQueue)
	c.AddURL("http://example.com/page2", 1, urlQueue)
	c.AddURL("http://example.com/page1", 1, urlQueue) // duplicate, shouldn't count

	count := c.GetDiscoveredCount()
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
}

func TestGetDroppedCount(t *testing.T) {
	c, _ := New("http://example.com", 3)
	// Create a queue with capacity of 2
	urlQueue := make(chan domain.URLTask, 2)

	// Initial dropped count should be 0
	if dropped := c.GetDroppedCount(); dropped != 0 {
		t.Errorf("Expected initial dropped count 0, got %d", dropped)
	}

	// Fill the queue
	c.AddURL("http://example.com/page1", 1, urlQueue)
	c.AddURL("http://example.com/page2", 1, urlQueue)

	// Queue is full - this should be dropped
	c.AddURL("http://example.com/page3", 1, urlQueue)
	c.AddURL("http://example.com/page4", 1, urlQueue)

	// Should have 2 dropped URLs
	droppedCount := c.GetDroppedCount()
	if droppedCount != 2 {
		t.Errorf("Expected dropped count 2, got %d", droppedCount)
	}

	// Discovered count should still include all unique URLs
	// (they were seen even if not queued)
	discoveredCount := c.GetDiscoveredCount()
	if discoveredCount != 4 {
		t.Errorf("Expected discovered count 4, got %d", discoveredCount)
	}
}
