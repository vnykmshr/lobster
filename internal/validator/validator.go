// Package validator validates test results against performance targets.
package validator

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/1mb-dev/lobster/internal/domain"
)

// Validator validates test results against performance targets
type Validator struct {
	targets          []domain.PerformanceTarget
	targetConfig     domain.PerformanceTargets
	enableComparison bool
	comparisonTarget string
}

// New creates a new performance validator
func New(targets domain.PerformanceTargets) *Validator {
	return &Validator{
		targets:          make([]domain.PerformanceTarget, 0),
		targetConfig:     targets,
		enableComparison: false,
	}
}

// NewWithComparison creates a validator with competitive comparison enabled
func NewWithComparison(targets domain.PerformanceTargets, comparisonTarget string) *Validator {
	return &Validator{
		targets:          make([]domain.PerformanceTarget, 0),
		targetConfig:     targets,
		enableComparison: true,
		comparisonTarget: comparisonTarget,
	}
}

// ValidateResults validates the test results against performance targets
func (v *Validator) ValidateResults(results *domain.TestResults) {
	v.targets = make([]domain.PerformanceTarget, 0)

	// Parse response times for calculations
	responseTimes := make([]time.Duration, len(results.ResponseTimes))
	for i, entry := range results.ResponseTimes {
		responseTimes[i] = entry.ResponseTime
	}

	var avgResponseTime, p95ResponseTime, p99ResponseTime time.Duration
	if len(responseTimes) > 0 {
		// Calculate average
		var total time.Duration
		for _, rt := range responseTimes {
			total += rt
		}
		avgResponseTime = total / time.Duration(len(responseTimes))

		// Calculate percentiles using proper sorting
		sort.Slice(responseTimes, func(i, j int) bool {
			return responseTimes[i] < responseTimes[j]
		})

		p95Index := int(float64(len(responseTimes)) * 0.95)
		p99Index := int(float64(len(responseTimes)) * 0.99)

		if p95Index >= len(responseTimes) {
			p95Index = len(responseTimes) - 1
		}
		if p99Index >= len(responseTimes) {
			p99Index = len(responseTimes) - 1
		}

		p95ResponseTime = responseTimes[p95Index]
		p99ResponseTime = responseTimes[p99Index]
	}

	// Throughput validation
	v.targets = append(v.targets, domain.PerformanceTarget{
		Name:        "Requests per Second",
		Target:      fmt.Sprintf("≥ %.1f req/s", v.targetConfig.RequestsPerSecond),
		Actual:      fmt.Sprintf("%.1f req/s", results.RequestsPerSecond),
		Description: fmt.Sprintf("Target: >%.1f req/s for optimal throughput", v.targetConfig.RequestsPerSecond),
		Passed:      results.RequestsPerSecond >= v.targetConfig.RequestsPerSecond,
	})

	// Average response time
	avgMs := float64(avgResponseTime.Nanoseconds()) / 1e6
	v.targets = append(v.targets, domain.PerformanceTarget{
		Name:        "Average Response Time",
		Target:      fmt.Sprintf("< %.1fms", v.targetConfig.AvgResponseTimeMs),
		Actual:      fmt.Sprintf("%.1fms", avgMs),
		Description: fmt.Sprintf("Target: <%.1fms for excellent user experience", v.targetConfig.AvgResponseTimeMs),
		Passed:      avgMs < v.targetConfig.AvgResponseTimeMs,
	})

	// 95th percentile response time
	p95Ms := float64(p95ResponseTime.Nanoseconds()) / 1e6
	v.targets = append(v.targets, domain.PerformanceTarget{
		Name:        "95th Percentile Response Time",
		Target:      fmt.Sprintf("< %.1fms", v.targetConfig.P95ResponseTimeMs),
		Actual:      fmt.Sprintf("%.1fms", p95Ms),
		Description: fmt.Sprintf("Target: <%.1fms for consistent performance", v.targetConfig.P95ResponseTimeMs),
		Passed:      p95Ms < v.targetConfig.P95ResponseTimeMs,
	})

	// 99th percentile response time
	p99Ms := float64(p99ResponseTime.Nanoseconds()) / 1e6
	v.targets = append(v.targets, domain.PerformanceTarget{
		Name:        "99th Percentile Response Time",
		Target:      fmt.Sprintf("< %.1fms", v.targetConfig.P99ResponseTimeMs),
		Actual:      fmt.Sprintf("%.1fms", p99Ms),
		Description: fmt.Sprintf("Target: <%.1fms even for worst-case scenarios", v.targetConfig.P99ResponseTimeMs),
		Passed:      p99Ms < v.targetConfig.P99ResponseTimeMs,
	})

	// Success rate
	successRate := (float64(results.SuccessfulRequests) / float64(results.TotalRequests)) * 100
	v.targets = append(v.targets, domain.PerformanceTarget{
		Name:        "Success Rate",
		Target:      fmt.Sprintf("> %.1f%%", v.targetConfig.SuccessRate),
		Actual:      fmt.Sprintf("%.2f%%", successRate),
		Description: fmt.Sprintf("Target: >%.1f%% successful requests for production readiness", v.targetConfig.SuccessRate),
		Passed:      successRate > v.targetConfig.SuccessRate,
	})

	// Error rate
	errorRate := float64(results.FailedRequests) / float64(results.TotalRequests) * 100
	v.targets = append(v.targets, domain.PerformanceTarget{
		Name:        "Error Rate",
		Target:      fmt.Sprintf("< %.1f%%", v.targetConfig.ErrorRate),
		Actual:      fmt.Sprintf("%.2f%%", errorRate),
		Description: fmt.Sprintf("Target: <%.1f%% for production reliability", v.targetConfig.ErrorRate),
		Passed:      errorRate < v.targetConfig.ErrorRate,
	})
}

// PrintValidationReport prints a detailed performance validation report
func (v *Validator) PrintValidationReport() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PERFORMANCE TARGET VALIDATION")
	fmt.Println(strings.Repeat("=", 60))

	passed := 0
	total := len(v.targets)

	for _, target := range v.targets {
		status := "FAIL"
		if target.Passed {
			status = "PASS"
			passed++
		}

		fmt.Printf("%s %-30s %s\n", status, target.Name+":", target.Actual)
		fmt.Printf("    %s\n", target.Description)
		fmt.Println()
	}

	fmt.Printf("Overall: %d/%d targets met (%.1f%%)\n", passed, total, float64(passed)/float64(total)*100)

	switch {
	case passed == total:
		fmt.Println("ALL PERFORMANCE TARGETS MET! Application is production-ready.")
	case passed >= total*3/4:
		fmt.Println("WARNING: Most targets met, but some optimization may be needed.")
	default:
		fmt.Println("Significant performance improvements needed before production.")
	}

	if v.enableComparison && v.comparisonTarget != "" {
		v.printCompetitiveAnalysis()
	}
}

// printCompetitiveAnalysis prints competitive comparison if enabled
func (v *Validator) printCompetitiveAnalysis() {
	fmt.Println("\nCOMPETITIVE ANALYSIS")
	fmt.Println(strings.Repeat("=", 60))

	// Check if we have performance metrics
	hasGoodPerf := false
	hasGoodThroughput := false

	for _, target := range v.targets {
		if target.Name == "95th Percentile Response Time" && target.Passed {
			fmt.Printf("Meeting p95 latency targets vs %s\n", v.comparisonTarget)
			hasGoodPerf = true
		}
		if target.Name == "Requests per Second" && target.Passed {
			fmt.Printf("Meeting throughput targets vs %s\n", v.comparisonTarget)
			hasGoodThroughput = true
		}
	}

	if !hasGoodPerf {
		fmt.Printf("WARNING: Response times need improvement to match %s\n", v.comparisonTarget)
	}
	if !hasGoodThroughput {
		fmt.Printf("WARNING: Throughput needs improvement to match %s\n", v.comparisonTarget)
	}
}

// GetValidationSummary returns a summary of the validation results
func (v *Validator) GetValidationSummary() map[string]interface{} {
	passed := 0
	total := len(v.targets)

	targetDetails := make([]map[string]interface{}, len(v.targets))
	for i, target := range v.targets {
		if target.Passed {
			passed++
		}
		targetDetails[i] = map[string]interface{}{
			"name":        target.Name,
			"target":      target.Target,
			"actual":      target.Actual,
			"description": target.Description,
			"passed":      target.Passed,
		}
	}

	return map[string]interface{}{
		"targets_met":    passed,
		"total_targets":  total,
		"success_rate":   float64(passed) / float64(total) * 100,
		"overall_status": v.getOverallStatus(passed, total),
		"targets":        targetDetails,
	}
}

// getOverallStatus returns the overall validation status
func (v *Validator) getOverallStatus(passed, total int) string {
	switch {
	case passed == total:
		return "PRODUCTION_READY"
	case passed >= total*3/4:
		return "MOSTLY_READY"
	default:
		return "NEEDS_IMPROVEMENT"
	}
}
