package reporter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vnykmshr/lobster/internal/domain"
	"github.com/vnykmshr/lobster/internal/testutil"
)

func TestNew(t *testing.T) {
	results := testutil.SampleResults()
	reporter := New(results)

	if reporter == nil {
		t.Fatal("Expected reporter to be non-nil")
	}

	if reporter.results != results {
		t.Error("Expected reporter to store the provided results")
	}
}

func TestGenerateJSON_Success(t *testing.T) {
	results := testutil.SampleResults()
	reporter := New(results)

	// Create temp file
	tmpfile, err := os.CreateTemp("", "lobster-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpfile.Name()); removeErr != nil {
			t.Logf("Warning: failed to remove temp file: %v", removeErr)
		}
	}()
	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("Failed to close temp file: %v", closeErr)
	}

	// Generate JSON
	err = reporter.GenerateJSON(tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file exists and has content
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("Expected generated JSON to have content")
	}

	// Verify it's valid JSON
	var decoded domain.TestResults
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Verify key fields
	if decoded.TotalRequests != 100 {
		t.Errorf("Expected TotalRequests 100, got %d", decoded.TotalRequests)
	}

	if decoded.SuccessRate != 95.0 {
		t.Errorf("Expected SuccessRate 95.0, got %.1f", decoded.SuccessRate)
	}
}

// testFilePermissions is a helper that tests generated file permissions.
// It creates a temp file, calls the generate function, and verifies 0o600 mode.
func testFilePermissions(t *testing.T, ext string, generateFn func(string) error) {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "lobster-test-*."+ext)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("Failed to close temp file: %v", closeErr)
	}
	defer func() {
		if removeErr := os.Remove(tmpfile.Name()); removeErr != nil {
			t.Logf("Warning: failed to remove temp file: %v", removeErr)
		}
	}()

	err = generateFn(tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check file permissions (0o600 = -rw-------)
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	expectedMode := os.FileMode(0o600)
	if info.Mode() != expectedMode {
		t.Errorf("Expected file mode %v, got %v", expectedMode, info.Mode())
	}
}

func TestGenerateJSON_FilePermissions(t *testing.T) {
	reporter := New(testutil.SampleResults())
	testFilePermissions(t, "json", reporter.GenerateJSON)
}

func TestGenerateHTML_Success(t *testing.T) {
	results := testutil.SampleResults()
	reporter := New(results)

	tmpfile, err := os.CreateTemp("", "lobster-test-*.html")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if removeErr := os.Remove(tmpfile.Name()); removeErr != nil {
			t.Logf("Warning: failed to remove temp file: %v", removeErr)
		}
	}()
	if closeErr := tmpfile.Close(); closeErr != nil {
		t.Fatalf("Failed to close temp file: %v", closeErr)
	}

	err = reporter.GenerateHTML(tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify file has content
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	html := string(data)

	if html == "" {
		t.Fatal("Expected generated HTML to have content")
	}

	// Verify HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Expected HTML to have DOCTYPE")
	}

	if !strings.Contains(html, "<html") {
		t.Error("Expected HTML to have html tag")
	}

	if !strings.Contains(html, "Lobster Test Report") {
		t.Error("Expected HTML to contain report title")
	}

	// Verify data is included
	if !strings.Contains(html, "example.com") {
		t.Error("Expected HTML to contain test URL")
	}

	// Verify Chart.js is included
	if !strings.Contains(html, "Chart") {
		t.Error("Expected HTML to include Chart.js code")
	}
}

func TestGenerateHTML_FilePermissions(t *testing.T) {
	reporter := New(testutil.SampleResults())
	testFilePermissions(t, "html", reporter.GenerateHTML)
}

func TestPrepareTemplateData(t *testing.T) {
	results := testutil.SampleResults()
	reporter := New(results)

	data := reporter.prepareTemplateData()

	// Verify required fields are populated (typed struct ensures existence)
	if data.Timestamp == "" {
		t.Error("Expected Timestamp to be set")
	}
	if data.Duration == "" {
		t.Error("Expected Duration to be set")
	}

	// Verify SuccessRateClass logic - 95% should be "success-high"
	if data.SuccessRateClass != "success-high" {
		t.Errorf("Expected SuccessRateClass 'success-high' for 95%%, got '%s'", data.SuccessRateClass)
	}

	// Verify StatusDistribution
	if len(data.StatusDistribution) == 0 {
		t.Error("Expected StatusDistribution to have entries")
	}

	// Verify ResponseTimesMs conversion
	if len(data.ResponseTimesMs) != len(results.ResponseTimes) {
		t.Errorf("Expected %d response times, got %d", len(results.ResponseTimes), len(data.ResponseTimesMs))
	}

	// Verify conversion to milliseconds (100ms → 100.0)
	if data.ResponseTimesMs[0] != 100.0 {
		t.Errorf("Expected first response time 100.0ms, got %.1f", data.ResponseTimesMs[0])
	}
}

func TestPrepareTemplateData_SuccessRateClasses(t *testing.T) {
	tests := []struct {
		name          string
		successRate   float64
		expectedClass string
	}{
		{"High success rate", 95.0, "success-high"},
		{"Medium success rate", 85.0, "success-medium"},
		{"Low success rate", 65.0, "success-low"},
		{"Edge case 90%", 90.0, "success-high"},
		{"Edge case 89%", 89.0, "success-medium"},
		{"Edge case 70%", 70.0, "success-medium"},
		{"Edge case 69%", 69.0, "success-low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := testutil.SampleResults()
			results.SuccessRate = tt.successRate
			reporter := New(results)

			data := reporter.prepareTemplateData()
			if data.SuccessRateClass != tt.expectedClass {
				t.Errorf("Expected class '%s' for %.1f%%, got '%s'",
					tt.expectedClass, tt.successRate, data.SuccessRateClass)
			}
		})
	}
}

func TestGetHTMLTemplate(t *testing.T) {
	results := testutil.SampleResults()
	reporter := New(results)

	tmpl := reporter.getHTMLTemplate()

	if tmpl == "" {
		t.Fatal("Expected template to have content")
	}

	// Verify it's HTML
	if !strings.Contains(tmpl, "<!DOCTYPE html>") {
		t.Error("Expected template to be HTML")
	}

	// Verify template placeholders
	expectedPlaceholders := []string{
		"{{.Timestamp}}",
		"{{.Duration}}",
		"{{.TotalRequests}}",
		"{{.URLsDiscovered}}",
		"{{range .StatusDistribution}}",
		"{{range .URLValidations}}",
		"{{range .SlowRequests}}",
		"{{range .Errors}}",
	}

	for _, placeholder := range expectedPlaceholders {
		if !strings.Contains(tmpl, placeholder) {
			t.Errorf("Expected template to contain '%s'", placeholder)
		}
	}

	// Verify Chart.js reference
	if !strings.Contains(tmpl, "chart.js") {
		t.Error("Expected template to reference Chart.js")
	}
}

func TestPrepareTemplateData_StatusDistribution(t *testing.T) {
	results := &domain.TestResults{
		URLValidations: []domain.URLValidation{
			{StatusCode: 200},
			{StatusCode: 200},
			{StatusCode: 301},
			{StatusCode: 404},
			{StatusCode: 500},
		},
	}
	reporter := New(results)

	data := reporter.prepareTemplateData()

	// Should have 4 unique status codes
	if len(data.StatusDistribution) != 4 {
		t.Errorf("Expected 4 unique status codes, got %d", len(data.StatusDistribution))
	}

	// Verify status code 200 has count 2
	found200 := false
	for _, entry := range data.StatusDistribution {
		if entry.StatusCode == 200 {
			found200 = true
			if entry.Count != 2 {
				t.Errorf("Expected count 2 for status 200, got %d", entry.Count)
			}
			// 2 out of 5 = 40%
			if entry.Percentage != 40.0 {
				t.Errorf("Expected 40%% for status 200, got %.1f", entry.Percentage)
			}
		}
	}

	if !found200 {
		t.Error("Expected to find status code 200 in distribution")
	}
}

func TestPrepareTemplateData_StatusGroups(t *testing.T) {
	results := &domain.TestResults{
		URLValidations: []domain.URLValidation{
			{StatusCode: 200}, // Group: 200
			{StatusCode: 204}, // Group: 200
			{StatusCode: 301}, // Group: 300
			{StatusCode: 302}, // Group: 300
			{StatusCode: 404}, // Group: 400
			{StatusCode: 500}, // Group: 400 (actually 500+, but treated as 400+)
		},
	}
	reporter := New(results)

	data := reporter.prepareTemplateData()

	// Verify status groups
	statusGroups := make(map[int]string)
	for _, v := range data.URLValidations {
		statusGroups[v.StatusCode] = v.StatusGroup
	}

	if statusGroups[200] != "200" {
		t.Errorf("Expected status group '200' for 200, got '%s'", statusGroups[200])
	}

	if statusGroups[301] != "300" {
		t.Errorf("Expected status group '300' for 301, got '%s'", statusGroups[301])
	}

	if statusGroups[404] != "400" {
		t.Errorf("Expected status group '400' for 404, got '%s'", statusGroups[404])
	}

	if statusGroups[500] != "400" {
		t.Errorf("Expected status group '400' for 500, got '%s'", statusGroups[500])
	}
}

func TestGenerateJSON_InvalidPath(t *testing.T) {
	results := testutil.SampleResults()
	reporter := New(results)

	// Try to write to invalid path
	err := reporter.GenerateJSON("/nonexistent/directory/file.json")
	if err == nil {
		t.Fatal("Expected error for invalid path, got none")
	}
}

func TestGenerateHTML_InvalidPath(t *testing.T) {
	results := testutil.SampleResults()
	reporter := New(results)

	err := reporter.GenerateHTML("/nonexistent/directory/file.html")
	if err == nil {
		t.Fatal("Expected error for invalid path, got none")
	}
}

func TestPrepareTemplateData_EmptyResults(t *testing.T) {
	results := &domain.TestResults{
		URLValidations: []domain.URLValidation{},
		Errors:         []domain.ErrorInfo{},
		SlowRequests:   []domain.SlowRequest{},
		ResponseTimes:  []domain.ResponseTimeEntry{},
	}
	reporter := New(results)

	// Should not panic with empty data
	data := reporter.prepareTemplateData()

	if len(data.StatusDistribution) != 0 {
		t.Errorf("Expected empty status distribution, got %d entries", len(data.StatusDistribution))
	}

	if len(data.ResponseTimesMs) != 0 {
		t.Errorf("Expected empty response times, got %d entries", len(data.ResponseTimesMs))
	}
}

func TestPrintSummary(t *testing.T) {
	_ = t // Test verifies no panic occurs
	results := testutil.SampleResults()
	reporter := New(results)
	reporter.PrintSummary()
}

func TestPrintSummary_WithErrors(t *testing.T) {
	_ = t // Test verifies no panic occurs
	results := testutil.SampleResults()
	results.Errors = []domain.ErrorInfo{
		{URL: "http://example.com/err1", Error: "timeout", Timestamp: time.Now()},
		{URL: "http://example.com/err2", Error: "connection refused", Timestamp: time.Now()},
		{URL: "http://example.com/err3", Error: "timeout", Timestamp: time.Now()},
	}
	reporter := New(results)
	reporter.PrintSummary()
}

func TestPrintSummary_WithSlowRequests(t *testing.T) {
	_ = t // Test verifies no panic occurs
	results := testutil.SampleResults()
	results.SlowRequests = []domain.SlowRequest{
		{URL: "http://example.com/slow1", ResponseTime: 5 * time.Second, StatusCode: 200},
		{URL: "http://example.com/slow2", ResponseTime: 4 * time.Second, StatusCode: 200},
		{URL: "http://example.com/slow3", ResponseTime: 3 * time.Second, StatusCode: 500},
	}
	reporter := New(results)
	reporter.PrintSummary()
}
