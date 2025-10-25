package reporter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vnykmshr/lobster/internal/domain"
)

// Test helper to create sample test results
func sampleResults() *domain.TestResults {
	return &domain.TestResults{
		Duration:            "2m30s",
		URLsDiscovered:      10,
		TotalRequests:       100,
		SuccessfulRequests:  95,
		FailedRequests:      5,
		AverageResponseTime: "150ms",
		MinResponseTime:     "50ms",
		MaxResponseTime:     "500ms",
		RequestsPerSecond:   0.67,
		SuccessRate:         95.0,
		URLValidations: []domain.URLValidation{
			{
				URL:           "http://example.com",
				StatusCode:    200,
				ResponseTime:  100 * time.Millisecond,
				ContentLength: 1024,
				ContentType:   "text/html",
				LinksFound:    5,
				Depth:         0,
				IsValid:       true,
			},
			{
				URL:           "http://example.com/404",
				StatusCode:    404,
				ResponseTime:  50 * time.Millisecond,
				ContentLength: 0,
				ContentType:   "text/html",
				LinksFound:    0,
				Depth:         1,
				IsValid:       false,
			},
		},
		Errors: []domain.ErrorInfo{
			{
				URL:       "http://example.com/error",
				Error:     "connection timeout",
				Timestamp: time.Now(),
				Depth:     1,
			},
		},
		SlowRequests: []domain.SlowRequest{
			{
				URL:          "http://example.com/slow",
				ResponseTime: 3 * time.Second,
				StatusCode:   200,
			},
		},
		ResponseTimes: []domain.ResponseTimeEntry{
			{
				URL:          "http://example.com",
				ResponseTime: 100 * time.Millisecond,
				Timestamp:    time.Now(),
			},
			{
				URL:          "http://example.com/fast",
				ResponseTime: 50 * time.Millisecond,
				Timestamp:    time.Now(),
			},
		},
	}
}

func TestNew(t *testing.T) {
	results := sampleResults()
	reporter := New(results)

	if reporter == nil {
		t.Fatal("Expected reporter to be non-nil")
	}

	if reporter.results != results {
		t.Error("Expected reporter to store the provided results")
	}
}

func TestGenerateJSON_Success(t *testing.T) {
	results := sampleResults()
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

func TestGenerateJSON_FilePermissions(t *testing.T) {
	results := sampleResults()
	reporter := New(results)

	tmpfile, err := os.CreateTemp("", "lobster-test-*.json")
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

	err = reporter.GenerateJSON(tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	// 0o600 = -rw-------
	expectedMode := os.FileMode(0o600)
	if mode != expectedMode {
		t.Errorf("Expected file mode %v, got %v", expectedMode, mode)
	}
}

func TestGenerateHTML_Success(t *testing.T) {
	results := sampleResults()
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
	results := sampleResults()
	reporter := New(results)

	tmpfile, err := os.CreateTemp("", "lobster-test-*.html")
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

	err = reporter.GenerateHTML(tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0o600)
	if mode != expectedMode {
		t.Errorf("Expected file mode %v, got %v", expectedMode, mode)
	}
}

func TestPrepareTemplateData(t *testing.T) {
	results := sampleResults()
	reporter := New(results)

	data := reporter.prepareTemplateData()

	// Verify required keys exist
	requiredKeys := []string{
		"Timestamp",
		"Duration",
		"TotalRequests",
		"SuccessfulRequests",
		"FailedRequests",
		"URLsDiscovered",
		"SuccessRate",
		"SuccessRateClass",
		"RequestsPerSecond",
		"AverageResponseTime",
		"StatusDistribution",
		"URLValidations",
		"SlowRequests",
		"Errors",
		"ResponseTimesMs",
	}

	for _, key := range requiredKeys {
		if _, ok := data[key]; !ok {
			t.Errorf("Expected key '%s' in template data", key)
		}
	}

	// Verify SuccessRateClass logic
	successRateClass, ok := data["SuccessRateClass"].(string)
	if !ok {
		t.Fatal("Expected SuccessRateClass to be string")
	}

	// 95% should be "success-high"
	if successRateClass != "success-high" {
		t.Errorf("Expected SuccessRateClass 'success-high' for 95%%, got '%s'", successRateClass)
	}

	// Verify StatusDistribution
	statusDist, ok := data["StatusDistribution"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected StatusDistribution to be slice of maps")
	}

	if len(statusDist) == 0 {
		t.Error("Expected StatusDistribution to have entries")
	}

	// Verify ResponseTimesMs conversion
	responseTimesMs, ok := data["ResponseTimesMs"].([]float64)
	if !ok {
		t.Fatal("Expected ResponseTimesMs to be []float64")
	}

	if len(responseTimesMs) != len(results.ResponseTimes) {
		t.Errorf("Expected %d response times, got %d", len(results.ResponseTimes), len(responseTimesMs))
	}

	// Verify conversion to milliseconds (100ms → 100.0)
	if responseTimesMs[0] != 100.0 {
		t.Errorf("Expected first response time 100.0ms, got %.1f", responseTimesMs[0])
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
			results := sampleResults()
			results.SuccessRate = tt.successRate
			reporter := New(results)

			data := reporter.prepareTemplateData()
			successRateClass, ok := data["SuccessRateClass"].(string)
			if !ok {
				t.Fatal("Expected SuccessRateClass to be string")
			}

			if successRateClass != tt.expectedClass {
				t.Errorf("Expected class '%s' for %.1f%%, got '%s'",
					tt.expectedClass, tt.successRate, successRateClass)
			}
		})
	}
}

func TestGetHTMLTemplate(t *testing.T) {
	results := sampleResults()
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
	statusDist, ok := data["StatusDistribution"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected StatusDistribution to be []map[string]interface{}")
	}

	// Should have 4 unique status codes
	if len(statusDist) != 4 {
		t.Errorf("Expected 4 unique status codes, got %d", len(statusDist))
	}

	// Verify status code 200 has count 2
	found200 := false
	for _, entry := range statusDist {
		if entry["StatusCode"] == 200 {
			found200 = true
			if entry["Count"] != 2 {
				t.Errorf("Expected count 2 for status 200, got %v", entry["Count"])
			}
			// 2 out of 5 = 40%
			percentage, percentageOK := entry["Percentage"].(float64)
			if !percentageOK {
				t.Fatal("Expected Percentage to be float64")
			}
			if percentage != 40.0 {
				t.Errorf("Expected 40%% for status 200, got %.1f", percentage)
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
	urlValidations, ok := data["URLValidations"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected URLValidations to be []map[string]interface{}")
	}

	// Verify status groups
	statusGroups := make(map[int]string)
	for _, v := range urlValidations {
		statusCode, codeOK := v["StatusCode"].(int)
		if !codeOK {
			t.Fatal("Expected StatusCode to be int")
		}
		statusGroup, groupOK := v["StatusGroup"].(string)
		if !groupOK {
			t.Fatal("Expected StatusGroup to be string")
		}
		statusGroups[statusCode] = statusGroup
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
	results := sampleResults()
	reporter := New(results)

	// Try to write to invalid path
	err := reporter.GenerateJSON("/nonexistent/directory/file.json")
	if err == nil {
		t.Fatal("Expected error for invalid path, got none")
	}
}

func TestGenerateHTML_InvalidPath(t *testing.T) {
	results := sampleResults()
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

	statusDist, ok := data["StatusDistribution"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected StatusDistribution to be []map[string]interface{}")
	}
	if len(statusDist) != 0 {
		t.Errorf("Expected empty status distribution, got %d entries", len(statusDist))
	}

	responseTimesMs, ok := data["ResponseTimesMs"].([]float64)
	if !ok {
		t.Fatal("Expected ResponseTimesMs to be []float64")
	}
	if len(responseTimesMs) != 0 {
		t.Errorf("Expected empty response times, got %d entries", len(responseTimesMs))
	}
}

func TestPrintSummary(t *testing.T) {
	results := sampleResults()
	reporter := New(results)

	// PrintSummary outputs to stdout, just verify it doesn't panic
	reporter.PrintSummary()

	// This test mainly ensures the function executes without error
	// Visual output would need to be verified manually
}

func TestPrintSummary_WithErrors(t *testing.T) {
	results := sampleResults()
	results.Errors = []domain.ErrorInfo{
		{URL: "http://example.com/err1", Error: "timeout", Timestamp: time.Now()},
		{URL: "http://example.com/err2", Error: "connection refused", Timestamp: time.Now()},
		{URL: "http://example.com/err3", Error: "timeout", Timestamp: time.Now()},
	}
	reporter := New(results)

	// Should print error summary
	reporter.PrintSummary()
}

func TestPrintSummary_WithSlowRequests(t *testing.T) {
	results := sampleResults()
	results.SlowRequests = []domain.SlowRequest{
		{URL: "http://example.com/slow1", ResponseTime: 5 * time.Second, StatusCode: 200},
		{URL: "http://example.com/slow2", ResponseTime: 4 * time.Second, StatusCode: 200},
		{URL: "http://example.com/slow3", ResponseTime: 3 * time.Second, StatusCode: 500},
	}
	reporter := New(results)

	// Should print slow requests section
	reporter.PrintSummary()
}
