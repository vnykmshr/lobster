package domain

import (
	"testing"
	"time"
)

func TestURLTask_Fields(t *testing.T) {
	task := URLTask{
		URL:   "https://example.com",
		Depth: 2,
	}

	if task.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", task.URL)
	}
	if task.Depth != 2 {
		t.Errorf("Expected Depth 2, got %d", task.Depth)
	}
}

func TestURLValidation_IsValidLogic(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		isValid    bool
	}{
		{"2xx status", 200, true},
		{"2xx status OK", 204, true},
		{"3xx redirect", 301, true},
		{"3xx redirect temp", 302, true},
		{"4xx client error", 404, false},
		{"4xx bad request", 400, false},
		{"5xx server error", 500, false},
		{"5xx unavailable", 503, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := URLValidation{ //nolint:govet // Test uses complete struct initialization for realism
				URL:           "https://example.com",
				StatusCode:    tt.statusCode,
				ResponseTime:  100 * time.Millisecond,
				ContentLength: 1024,
				ContentType:   "text/html",
				LinksFound:    5,
				Depth:         1,
				IsValid:       tt.statusCode >= 200 && tt.statusCode < 400,
			}

			if validation.IsValid != tt.isValid {
				t.Errorf("For status %d, expected IsValid=%v, got %v",
					tt.statusCode, tt.isValid, validation.IsValid)
			}
		})
	}
}

func TestErrorInfo_Fields(t *testing.T) {
	now := time.Now()
	errInfo := ErrorInfo{
		URL:       "https://example.com/fail",
		Error:     "connection timeout",
		Timestamp: now,
		Depth:     3,
	}

	if errInfo.URL != "https://example.com/fail" {
		t.Errorf("Expected URL 'https://example.com/fail', got '%s'", errInfo.URL)
	}
	if errInfo.Error != "connection timeout" {
		t.Errorf("Expected Error 'connection timeout', got '%s'", errInfo.Error)
	}
	if !errInfo.Timestamp.Equal(now) {
		t.Errorf("Expected Timestamp %v, got %v", now, errInfo.Timestamp)
	}
	if errInfo.Depth != 3 {
		t.Errorf("Expected Depth 3, got %d", errInfo.Depth)
	}
}

func TestSlowRequest_Fields(t *testing.T) {
	responseTime := 3 * time.Second
	slowReq := SlowRequest{
		URL:          "https://example.com/slow",
		ResponseTime: responseTime,
		StatusCode:   200,
	}

	if slowReq.URL != "https://example.com/slow" {
		t.Errorf("Expected URL 'https://example.com/slow', got '%s'", slowReq.URL)
	}
	if slowReq.ResponseTime != responseTime {
		t.Errorf("Expected ResponseTime %v, got %v", responseTime, slowReq.ResponseTime)
	}
	if slowReq.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200, got %d", slowReq.StatusCode)
	}
}

func TestResponseTimeEntry_Fields(t *testing.T) {
	now := time.Now()
	responseTime := 50 * time.Millisecond
	entry := ResponseTimeEntry{
		URL:          "https://example.com",
		ResponseTime: responseTime,
		Timestamp:    now,
	}

	if entry.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got '%s'", entry.URL)
	}
	if entry.ResponseTime != responseTime {
		t.Errorf("Expected ResponseTime %v, got %v", responseTime, entry.ResponseTime)
	}
	if !entry.Timestamp.Equal(now) {
		t.Errorf("Expected Timestamp %v, got %v", now, entry.Timestamp)
	}
}

func TestPerformanceTarget_Fields(t *testing.T) {
	target := PerformanceTarget{
		Name:        "Requests per Second",
		Target:      "≥ 100 req/s",
		Actual:      "120 req/s",
		Description: "Throughput validation",
		Passed:      true,
	}

	if target.Name != "Requests per Second" {
		t.Errorf("Expected Name 'Requests per Second', got '%s'", target.Name)
	}
	if target.Target != "≥ 100 req/s" {
		t.Errorf("Expected Target '≥ 100 req/s', got '%s'", target.Target)
	}
	if target.Actual != "120 req/s" {
		t.Errorf("Expected Actual '120 req/s', got '%s'", target.Actual)
	}
	if target.Description != "Throughput validation" {
		t.Errorf("Expected Description 'Throughput validation', got '%s'", target.Description)
	}
	if !target.Passed {
		t.Error("Expected Passed to be true")
	}
}

func TestTestResults_Initialization(t *testing.T) {
	results := TestResults{
		Duration:           "2m0s",
		URLsDiscovered:     42,
		TotalRequests:      1000,
		SuccessfulRequests: 980,
		FailedRequests:     20,
		URLValidations:     []URLValidation{},
		Errors:             []ErrorInfo{},
		SlowRequests:       []SlowRequest{},
		ResponseTimes:      []ResponseTimeEntry{},
	}

	if results.Duration != "2m0s" {
		t.Errorf("Expected Duration '2m0s', got '%s'", results.Duration)
	}
	if results.URLsDiscovered != 42 {
		t.Errorf("Expected URLsDiscovered 42, got %d", results.URLsDiscovered)
	}
	if results.TotalRequests != 1000 {
		t.Errorf("Expected TotalRequests 1000, got %d", results.TotalRequests)
	}
	if results.SuccessfulRequests != 980 {
		t.Errorf("Expected SuccessfulRequests 980, got %d", results.SuccessfulRequests)
	}
	if results.FailedRequests != 20 {
		t.Errorf("Expected FailedRequests 20, got %d", results.FailedRequests)
	}
	if results.URLValidations == nil {
		t.Error("Expected URLValidations to be initialized")
	}
	if results.Errors == nil {
		t.Error("Expected Errors to be initialized")
	}
	if results.SlowRequests == nil {
		t.Error("Expected SlowRequests to be initialized")
	}
	if results.ResponseTimes == nil {
		t.Error("Expected ResponseTimes to be initialized")
	}
}
