# Testing Strategy

This document describes the testing approach for Lobster, including test categories, coverage goals, and how to run tests locally.

## Test Categories

### Unit Tests

Unit tests verify individual functions and methods in isolation. They should:

- Test a single unit of functionality
- Use minimal setup and teardown
- Run quickly (milliseconds)
- Not depend on external services
- Mock external dependencies when needed

Examples:
- URL validation logic
- Response time calculations
- Configuration parsing
- Report data preparation

### Integration Tests

Integration tests verify that components work together correctly. They may:

- Test multiple components interacting
- Use test servers for HTTP requests
- Verify end-to-end workflows
- Take longer to execute (seconds)

Examples:
- Tester making real HTTP requests to test server
- Reporter generating actual HTML/JSON files
- robots.txt fetching and parsing
- Authentication flows

### Table-Driven Tests

Use table-driven tests for testing multiple inputs and expected outputs:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"valid URL", "http://example.com", true},
        {"invalid URL", "not-a-url", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Validate(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Coverage Goals

Target coverage by package:

- Overall: 70%+ (currently 30.2%)
- Critical packages: tester 86.9%, reporter TBD, validator 51.2%
- Domain models: 100%
- Utility packages: 90%+

Run coverage reports:

```bash
# Fast tests only (skips slow integration tests)
go test -short ./... -cover

# Full test suite (includes slow tests)
go test ./... -cover

# Detailed coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### What to Test

- Business logic and algorithms
- Error handling paths (HTTP 429 retry, context cancellation, etc.)
- Edge cases and boundary conditions
- Public APIs and exported functions
- Critical user workflows (crawl → test → report)

### What Not to Test

- Third-party library internals
- Standard library behavior
- Trivial functions

## Running Tests

### Run All Tests

```bash
# Fast tests only (recommended for development)
go test -short ./...

# Full test suite including slow integration tests
go test ./...
```

### Run Tests with Coverage

```bash
go test -short ./... -cover
```

### Run Tests with Race Detector

```bash
go test -short ./... -race
```

### Run Specific Package

```bash
go test ./internal/tester -v
```

### Run Specific Test

```bash
go test ./internal/tester -run TestProcessURL
```

## Test Categories

### Unit Tests
Fast, isolated tests of individual functions. See `internal/tester/tester_test.go` for examples.

### Integration Tests
End-to-end tests with real HTTP servers. Use `testing.Short()` to skip in fast mode:

```go
func TestIntegration_FullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // ... test code
}
```

### Slow Tests
Tests taking >2 seconds (rate limiting, slow requests). Always skipped with `-short` flag.

## Continuous Integration

CI pipeline (in progress):
- Run fast tests: `go test -short ./...`
- Check coverage
- Run linters: `golangci-lint run`

## Current Test Suite

**Package Coverage** (as of last run):
- `internal/tester`: 86.9%
- `internal/config`: 95.2%
- `internal/crawler`: 94.9%
- `internal/validator`: 51.2%
- `internal/domain`: 100%
- **Overall**: 30.2%

**Test Breakdown**:
- Unit tests: ~50 tests
- Integration tests: 6 tests (skipped with `-short`)
- Slow tests: 2 tests (skipped with `-short`)

Run `go test -short ./... -cover` to verify current coverage.
