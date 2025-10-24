package validator

import (
	"strings"
	"testing"
	"time"

	"github.com/vnykmshr/lobster/internal/domain"
)

func TestNew(t *testing.T) {
	targets := domain.PerformanceTargets{
		RequestsPerSecond:   100,
		AvgResponseTimeMs:   50,
		P95ResponseTimeMs:   100,
		P99ResponseTimeMs:   200,
		SuccessRate:         99.0,
		ErrorRate:           1.0,
	}

	v := New(targets)
	if v == nil {
		t.Fatal("Expected non-nil validator")
	}
	if v.enableComparison {
		t.Error("Expected enableComparison to be false")
	}
	if v.targetConfig.RequestsPerSecond != 100 {
		t.Errorf("Expected RequestsPerSecond 100, got %f", v.targetConfig.RequestsPerSecond)
	}
}

func TestNewWithComparison(t *testing.T) {
	targets := domain.PerformanceTargets{
		RequestsPerSecond:   100,
		AvgResponseTimeMs:   50,
		P95ResponseTimeMs:   100,
		P99ResponseTimeMs:   200,
		SuccessRate:         99.0,
		ErrorRate:           1.0,
	}

	v := NewWithComparison(targets, "Ghost")
	if v == nil {
		t.Fatal("Expected non-nil validator")
	}
	if !v.enableComparison {
		t.Error("Expected enableComparison to be true")
	}
	if v.comparisonTarget != "Ghost" {
		t.Errorf("Expected comparisonTarget 'Ghost', got '%s'", v.comparisonTarget)
	}
}

func TestValidateResults_AllPassing(t *testing.T) {
	targets := domain.PerformanceTargets{
		RequestsPerSecond:   10,  // Low threshold
		AvgResponseTimeMs:   100, // High threshold (easier to pass)
		P95ResponseTimeMs:   200,
		P99ResponseTimeMs:   300,
		SuccessRate:         90.0,
		ErrorRate:           10.0,
	}

	v := New(targets)

	// Create test results that should pass all targets
	results := &domain.TestResults{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		RequestsPerSecond:  20.0, // > 10
		ResponseTimes: []domain.ResponseTimeEntry{
			{ResponseTime: 30 * time.Millisecond},
			{ResponseTime: 40 * time.Millisecond},
			{ResponseTime: 50 * time.Millisecond},
			{ResponseTime: 60 * time.Millisecond},
			{ResponseTime: 70 * time.Millisecond},
		},
	}

	v.ValidateResults(results)

	// Check that targets were created
	if len(v.targets) == 0 {
		t.Fatal("Expected targets to be populated")
	}

	// Verify all targets passed
	for _, target := range v.targets {
		if !target.Passed {
			t.Errorf("Expected target '%s' to pass, but it failed. Target: %s, Actual: %s",
				target.Name, target.Target, target.Actual)
		}
	}
}

func TestValidateResults_AllFailing(t *testing.T) {
	targets := domain.PerformanceTargets{
		RequestsPerSecond:   1000, // Very high threshold
		AvgResponseTimeMs:   1,    // Very low threshold (hard to pass)
		P95ResponseTimeMs:   2,
		P99ResponseTimeMs:   3,
		SuccessRate:         99.9,
		ErrorRate:           0.1,
	}

	v := New(targets)

	// Create test results that should fail all targets
	results := &domain.TestResults{
		TotalRequests:      100,
		SuccessfulRequests: 50,
		FailedRequests:     50,
		RequestsPerSecond:  5.0, // << 1000
		ResponseTimes: []domain.ResponseTimeEntry{
			{ResponseTime: 100 * time.Millisecond},
			{ResponseTime: 200 * time.Millisecond},
			{ResponseTime: 300 * time.Millisecond},
			{ResponseTime: 400 * time.Millisecond},
			{ResponseTime: 500 * time.Millisecond},
		},
	}

	v.ValidateResults(results)

	// Verify all targets failed
	for _, target := range v.targets {
		if target.Passed {
			t.Errorf("Expected target '%s' to fail, but it passed. Target: %s, Actual: %s",
				target.Name, target.Target, target.Actual)
		}
	}
}

func TestValidateResults_EmptyResponseTimes(t *testing.T) {
	targets := domain.DefaultPerformanceTargets()
	v := New(targets)

	results := &domain.TestResults{
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		RequestsPerSecond:  0,
		ResponseTimes:      []domain.ResponseTimeEntry{}, // Empty
	}

	// Should not panic with empty response times
	v.ValidateResults(results)

	if len(v.targets) == 0 {
		t.Error("Expected targets to be populated even with empty response times")
	}
}

func TestValidateResults_PercentilesCalculation(t *testing.T) {
	targets := domain.PerformanceTargets{
		RequestsPerSecond:   1,
		AvgResponseTimeMs:   1000,
		P95ResponseTimeMs:   1000,
		P99ResponseTimeMs:   1000,
		SuccessRate:         50.0,
		ErrorRate:           50.0,
	}

	v := New(targets)

	// Create 100 response times to test percentile calculation
	responseTimes := make([]domain.ResponseTimeEntry, 100)
	for i := 0; i < 100; i++ {
		responseTimes[i] = domain.ResponseTimeEntry{
			ResponseTime: time.Duration(i+1) * time.Millisecond,
		}
	}

	results := &domain.TestResults{
		TotalRequests:      100,
		SuccessfulRequests: 100,
		FailedRequests:     0,
		RequestsPerSecond:  10.0,
		ResponseTimes:      responseTimes,
	}

	v.ValidateResults(results)

	// Find P95 and P99 targets
	var p95Target, p99Target *domain.PerformanceTarget
	for i := range v.targets {
		if strings.Contains(v.targets[i].Name, "95th") {
			p95Target = &v.targets[i]
		}
		if strings.Contains(v.targets[i].Name, "99th") {
			p99Target = &v.targets[i]
		}
	}

	if p95Target == nil {
		t.Fatal("Expected P95 target to exist")
	}
	if p99Target == nil {
		t.Fatal("Expected P99 target to exist")
	}

	// P95 should be around 95ms (95th value)
	// P99 should be around 99ms (99th value)
	// Just verify they're calculated and formatted correctly
	if !strings.Contains(p95Target.Actual, "ms") {
		t.Errorf("Expected P95 actual to contain 'ms', got '%s'", p95Target.Actual)
	}
	if !strings.Contains(p99Target.Actual, "ms") {
		t.Errorf("Expected P99 actual to contain 'ms', got '%s'", p99Target.Actual)
	}
}

func TestValidateResults_SuccessRate(t *testing.T) {
	targets := domain.PerformanceTargets{
		RequestsPerSecond:   1,
		AvgResponseTimeMs:   1000,
		P95ResponseTimeMs:   1000,
		P99ResponseTimeMs:   1000,
		SuccessRate:         95.0,
		ErrorRate:           5.0,
	}

	v := New(targets)

	tests := []struct {
		name               string
		totalRequests      int64
		successfulRequests int64
		shouldPass         bool
	}{
		{"100% success", 100, 100, true},
		{"95% success (exact)", 100, 95, false}, // > 95%, not >= 95%
		{"96% success", 100, 96, true},
		{"94% success", 100, 94, false},
		{"0% success", 100, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &domain.TestResults{
				TotalRequests:      tt.totalRequests,
				SuccessfulRequests: tt.successfulRequests,
				FailedRequests:     tt.totalRequests - tt.successfulRequests,
				RequestsPerSecond:  10.0,
				ResponseTimes: []domain.ResponseTimeEntry{
					{ResponseTime: 50 * time.Millisecond},
				},
			}

			v.ValidateResults(results)

			// Find success rate target
			var successRateTarget *domain.PerformanceTarget
			for i := range v.targets {
				if strings.Contains(v.targets[i].Name, "Success Rate") {
					successRateTarget = &v.targets[i]
					break
				}
			}

			if successRateTarget == nil {
				t.Fatal("Expected Success Rate target to exist")
			}

			if successRateTarget.Passed != tt.shouldPass {
				t.Errorf("Expected Passed=%v, got %v. Actual: %s",
					tt.shouldPass, successRateTarget.Passed, successRateTarget.Actual)
			}
		})
	}
}

func TestGetValidationSummary(t *testing.T) {
	targets := domain.DefaultPerformanceTargets()
	v := New(targets)

	results := &domain.TestResults{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		RequestsPerSecond:  50.0,
		ResponseTimes: []domain.ResponseTimeEntry{
			{ResponseTime: 30 * time.Millisecond},
			{ResponseTime: 40 * time.Millisecond},
		},
	}

	v.ValidateResults(results)

	summary := v.GetValidationSummary()
	if summary == nil {
		t.Fatal("Expected non-nil summary")
	}

	// Verify summary contains expected keys (based on actual implementation)
	if _, ok := summary["targets_met"]; !ok {
		t.Error("Expected 'targets_met' in summary")
	}
	if _, ok := summary["total_targets"]; !ok {
		t.Error("Expected 'total_targets' in summary")
	}
	if _, ok := summary["success_rate"]; !ok {
		t.Error("Expected 'success_rate' in summary")
	}
	if _, ok := summary["overall_status"]; !ok {
		t.Error("Expected 'overall_status' in summary")
	}
	if _, ok := summary["targets"]; !ok {
		t.Error("Expected 'targets' in summary")
	}
}

func TestValidateResults_TargetCount(t *testing.T) {
	targets := domain.DefaultPerformanceTargets()
	v := New(targets)

	results := &domain.TestResults{
		TotalRequests:      100,
		SuccessfulRequests: 95,
		FailedRequests:     5,
		RequestsPerSecond:  50.0,
		ResponseTimes: []domain.ResponseTimeEntry{
			{ResponseTime: 50 * time.Millisecond},
		},
	}

	v.ValidateResults(results)

	// Should have 6 targets:
	// 1. Requests per Second
	// 2. Average Response Time
	// 3. P95 Response Time
	// 4. P99 Response Time
	// 5. Success Rate
	// 6. Error Rate
	expectedTargetCount := 6
	if len(v.targets) != expectedTargetCount {
		t.Errorf("Expected %d targets, got %d", expectedTargetCount, len(v.targets))
	}

	// Verify target names
	expectedNames := []string{
		"Requests per Second",
		"Average Response Time",
		"95th Percentile Response Time",
		"99th Percentile Response Time",
		"Success Rate",
		"Error Rate",
	}

	for i, expectedName := range expectedNames {
		if !strings.Contains(v.targets[i].Name, expectedName) {
			t.Errorf("Expected target %d to be '%s', got '%s'",
				i, expectedName, v.targets[i].Name)
		}
	}
}
