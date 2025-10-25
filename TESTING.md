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

- Overall: 70%+ (currently 90%+)
- Critical packages (tester, reporter, validator): 70%+
- Domain models: 100% (pure logic, easy to test)
- Utility packages: 90%+

Run coverage reports:

```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### What to Test

Focus on:
- Business logic and algorithms
- Error handling paths
- Edge cases and boundary conditions
- Public APIs and exported functions
- Critical user workflows

### What Not to Test

Avoid testing:
- Third-party library internals
- Standard library behavior
- Simple getters/setters
- Trivial helper functions
- Implementation details that may change

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test ./... -cover
```

### Run Tests with Race Detector

```bash
go test ./... -race
```

The race detector finds data races and concurrency bugs. Always run before committing changes to concurrent code.

### Run Specific Package Tests

```bash
go test ./internal/tester -v
go test ./internal/reporter -v -cover
```

### Run Specific Test Function

```bash
go test ./internal/tester -run TestProcessURL
go test ./internal/validator -run TestValidation/valid_targets
```

### Watch Mode

Use a tool like `entr` for continuous testing during development:

```bash
find . -name '*.go' | entr -c go test ./...
```

## Test Helpers

### Test Server Setup

For HTTP testing, use the httptest package:

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("test response"))
}))
defer server.Close()
```

### Test Configuration

Use helper functions for common test configurations:

```go
func testConfig(baseURL string) domain.TesterConfig {
    return domain.TesterConfig{
        BaseURL:        baseURL,
        Concurrency:    2,
        RequestTimeout: 5 * time.Second,
        UserAgent:      "TestAgent/1.0",
        IgnoreRobots:   true, // Avoid network calls in tests
    }
}
```

### Temporary Files

Use t.TempDir() for test files:

```go
func TestReportGeneration(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    outputPath := filepath.Join(tmpDir, "report.html")

    err := reporter.GenerateHTML(outputPath)
    if err != nil {
        t.Fatalf("GenerateHTML failed: %v", err)
    }
}
```

## Writing Good Tests

### Test Names

Use descriptive names that explain what is being tested:

```go
// Good
func TestMakeHTTPRequest_Success(t *testing.T)
func TestParse_WithWildcardPatterns(t *testing.T)
func TestValidation_FailsWhenResponseTimeTooSlow(t *testing.T)

// Bad
func TestHTTP(t *testing.T)
func TestParse(t *testing.T)
func TestValidation(t *testing.T)
```

### Assertions

Keep assertions simple and clear:

```go
// Good
if got != want {
    t.Errorf("CalculateAverage() = %v, want %v", got, want)
}

// Bad
if got != want {
    t.Error("failed")
}
```

### Setup and Teardown

Use t.Cleanup for resource cleanup:

```go
func TestWithCleanup(t *testing.T) {
    resource := acquireResource()
    t.Cleanup(func() {
        resource.Release()
    })

    // Test code here
}
```

### Test Isolation

Each test should be independent:

- Don't rely on test execution order
- Don't share state between tests
- Clean up resources after each test
- Use fresh instances for each test

## Continuous Integration

Tests run automatically on every commit via GitHub Actions. The CI pipeline:

1. Runs all tests with race detector
2. Generates coverage reports
3. Checks for lint errors
4. Builds the binary

Pull requests must pass all tests before merging.

## Test Performance

Keep tests fast:

- Unit tests should complete in milliseconds
- Integration tests should complete in seconds
- The full test suite should run in under 10 seconds

If tests are slow:
- Check for unnecessary sleeps or waits
- Mock slow external dependencies
- Reduce test server response times
- Parallelize independent tests with t.Parallel()

## Troubleshooting

### Flaky Tests

If tests fail intermittently:
- Check for race conditions (run with -race)
- Look for timing dependencies
- Verify proper cleanup
- Check for shared state

### Coverage Gaps

To find untested code:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -v 100.0%
```

Focus on covering error paths and edge cases first.

## Contributing Tests

When contributing:

1. Add tests for new features
2. Add tests for bug fixes
3. Maintain or improve coverage
4. Run tests locally before pushing
5. Ensure tests pass with -race flag
6. Keep tests simple and readable

See CONTRIBUTING.md for more details.
