// Package reporter generates test reports in various formats (console, JSON, HTML).
package reporter

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vnykmshr/lobster/internal/domain"
)

//go:embed templates/report.html
var reportTemplate string

// Reporter generates test reports in various formats
type Reporter struct {
	results *domain.TestResults
}

// New creates a new report generator
func New(results *domain.TestResults) *Reporter {
	return &Reporter{results: results}
}

// GenerateHTML creates an HTML report with interactive charts
func (r *Reporter) GenerateHTML(outputPath string) error {
	tmpl := r.getHTMLTemplate()

	// Prepare template data
	data := r.prepareTemplateData()

	// Parse and execute template
	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Create file with secure permissions (0o600) consistent with JSON output
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("cannot create HTML report %s: %w\nCheck directory exists and has write permissions", outputPath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	err = t.Execute(file, data)
	if err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

// GenerateJSON creates a JSON report
func (r *Reporter) GenerateJSON(outputPath string) error {
	data, err := json.MarshalIndent(r.results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	err = os.WriteFile(outputPath, data, 0o600)
	if err != nil {
		return fmt.Errorf("cannot write JSON report %s: %w\nCheck directory exists, has write permissions, and sufficient disk space", outputPath, err)
	}

	return nil
}

// PrintSummary prints a console summary of the results
func (r *Reporter) PrintSummary() {
	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("STRESS TEST RESULTS\n")
	fmt.Printf("%s\n", strings.Repeat("=", 60))
	fmt.Printf("Duration:             %s\n", r.results.Duration)
	fmt.Printf("URLs Discovered:      %d\n", r.results.URLsDiscovered)
	fmt.Printf("Total Requests:       %d\n", r.results.TotalRequests)
	fmt.Printf("Successful Requests:  %d\n", r.results.SuccessfulRequests)
	fmt.Printf("Failed Requests:      %d\n", r.results.FailedRequests)
	fmt.Printf("Average Response Time: %s\n", r.results.AverageResponseTime)
	fmt.Printf("Min Response Time:    %s\n", r.results.MinResponseTime)
	fmt.Printf("Max Response Time:    %s\n", r.results.MaxResponseTime)
	fmt.Printf("Requests/Second:      %.2f\n", r.results.RequestsPerSecond)
	fmt.Printf("Success Rate:         %.2f%%\n", r.results.SuccessRate)

	if len(r.results.Errors) > 0 {
		fmt.Printf("\n%s\n", strings.Repeat("-", 60))
		fmt.Printf("ERRORS SUMMARY\n")
		fmt.Printf("%s\n", strings.Repeat("-", 60))
		errorCounts := make(map[string]int)
		for _, err := range r.results.Errors {
			errorCounts[err.Error]++
		}
		for errorMsg, count := range errorCounts {
			fmt.Printf("  %s: %d occurrence(s)\n", errorMsg, count)
		}
	}

	if len(r.results.SlowRequests) > 0 {
		fmt.Printf("\n%s\n", strings.Repeat("-", 60))
		fmt.Printf("SLOWEST REQUESTS (top 10)\n")
		fmt.Printf("%s\n", strings.Repeat("-", 60))
		for i, req := range r.results.SlowRequests {
			if i >= 10 {
				break
			}
			fmt.Printf("  %s: %s (HTTP %d)\n", req.URL, req.ResponseTime, req.StatusCode)
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("-", 60))
	fmt.Printf("URL VALIDATION SUMMARY\n")
	fmt.Printf("%s\n", strings.Repeat("-", 60))
	statusCounts := make(map[int]int)
	for _, validation := range r.results.URLValidations {
		statusCounts[validation.StatusCode]++
	}

	for status, count := range statusCounts {
		fmt.Printf("HTTP %d: %d URL(s)\n", status, count)
	}
	fmt.Printf("%s\n\n", strings.Repeat("=", 60))
}

// prepareTemplateData prepares data for HTML template rendering
func (r *Reporter) prepareTemplateData() map[string]interface{} {
	// Calculate status distribution
	statusCounts := make(map[int]int)
	for _, validation := range r.results.URLValidations {
		statusCounts[validation.StatusCode]++
	}

	statusDistribution := make([]map[string]interface{}, 0, len(statusCounts))
	totalValidations := len(r.results.URLValidations)
	for status, count := range statusCounts {
		percentage := float64(count) / float64(totalValidations) * 100
		statusGroup := "200"
		if status >= 300 && status < 400 {
			statusGroup = "300"
		} else if status >= 400 {
			statusGroup = "400"
		}

		statusDistribution = append(statusDistribution, map[string]interface{}{
			"StatusCode":  status,
			"Count":       count,
			"Percentage":  percentage,
			"StatusGroup": statusGroup,
		})
	}

	// Sort by status code
	sort.Slice(statusDistribution, func(i, j int) bool {
		return statusDistribution[i]["StatusCode"].(int) < statusDistribution[j]["StatusCode"].(int) //nolint:errcheck // Type is guaranteed in template data
	})

	// Prepare URL validations with status groups
	urlValidations := make([]map[string]interface{}, 0, len(r.results.URLValidations))
	for _, validation := range r.results.URLValidations {
		statusGroup := "200"
		if validation.StatusCode >= 300 && validation.StatusCode < 400 {
			statusGroup = "300"
		} else if validation.StatusCode >= 400 {
			statusGroup = "400"
		}

		urlValidations = append(urlValidations, map[string]interface{}{
			"URL":           validation.URL,
			"StatusCode":    validation.StatusCode,
			"StatusGroup":   statusGroup,
			"ResponseTime":  validation.ResponseTime.String(),
			"ContentLength": validation.ContentLength,
			"LinksFound":    validation.LinksFound,
			"Depth":         validation.Depth,
		})
	}

	// Prepare slow requests
	slowRequests := make([]map[string]interface{}, 0, len(r.results.SlowRequests))
	for _, req := range r.results.SlowRequests {
		statusGroup := "200"
		if req.StatusCode >= 300 && req.StatusCode < 400 {
			statusGroup = "300"
		} else if req.StatusCode >= 400 {
			statusGroup = "400"
		}

		slowRequests = append(slowRequests, map[string]interface{}{
			"URL":          req.URL,
			"ResponseTime": req.ResponseTime.String(),
			"StatusCode":   req.StatusCode,
			"StatusGroup":  statusGroup,
		})
	}

	// Convert response times to milliseconds for charting
	responseTimesMs := make([]float64, 0, len(r.results.ResponseTimes))
	for _, entry := range r.results.ResponseTimes {
		responseTimesMs = append(responseTimesMs, float64(entry.ResponseTime.Nanoseconds())/1000000)
	}

	// Determine success rate class
	successRateClass := "success-high"
	if r.results.SuccessRate < 90 {
		successRateClass = "success-medium"
	}
	if r.results.SuccessRate < 70 {
		successRateClass = "success-low"
	}

	return map[string]interface{}{
		"Timestamp":           time.Now().Format("2006-01-02 15:04:05 MST"),
		"Duration":            r.results.Duration,
		"TotalRequests":       r.results.TotalRequests,
		"SuccessfulRequests":  r.results.SuccessfulRequests,
		"FailedRequests":      r.results.FailedRequests,
		"URLsDiscovered":      r.results.URLsDiscovered,
		"SuccessRate":         r.results.SuccessRate,
		"SuccessRateClass":    successRateClass,
		"RequestsPerSecond":   r.results.RequestsPerSecond,
		"AverageResponseTime": r.results.AverageResponseTime,
		"StatusDistribution":  statusDistribution,
		"URLValidations":      urlValidations,
		"SlowRequests":        slowRequests,
		"Errors":              r.results.Errors,
		"ResponseTimesMs":     responseTimesMs,
	}
}

// getHTMLTemplate returns the HTML template string from embedded file
func (r *Reporter) getHTMLTemplate() string {
	return reportTemplate
}
