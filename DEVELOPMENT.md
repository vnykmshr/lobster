# Lobster Development Guide

## Project Overview

**Lobster** is an intelligent web application stress testing tool that automatically discovers URLs through crawling and validates performance under concurrent load.

**Repository**: https://github.com/vnykmshr/lobster
**Version**: 0.1.0
**Created**: 2025-10-24
**License**: MIT

## Quick Context

### What Makes Lobster Different

1. **Crawler-First**: Automatically discovers URLs (unlike ab, wrk, hey, vegeta)
2. **Validation-Focused**: Pass/fail criteria vs raw metrics (unlike k6, JMeter)
3. **Zero-Config**: Works out of the box (unlike enterprise tools)
4. **Smart Rate Limiting**: Uses goflow token bucket to respect server capacity

### Origin Story

Graduated from `markgo/examples/stress-test` on 2025-10-24. Originally built to test MarkGo blog performance, evolved into a general-purpose tool.

## Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────────────┐
│          cmd/lobster (CLI)                │
│  - Flag parsing                             │
│  - Orchestration                            │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│          Application Layer                  │
│  - config/loader.go                         │
│  - Main workflow coordination               │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│          Domain Layer                       │
│  - domain/entities.go (core types)          │
│  - domain/config.go (configuration)         │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│          Use Case Layer                     │
│  - crawler/ (URL discovery)                 │
│  - tester/ (stress testing)                 │
│  - reporter/ (report generation)            │
│  - validator/ (performance validation)      │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│          Infrastructure                     │
│  - goflow (rate limiting)                   │
│  - net/http (HTTP client)                   │
│  - html/template (reporting)                │
└─────────────────────────────────────────────┘
```

### Component Responsibilities

**domain/**
- `entities.go`: Core business entities (TestResults, URLTask, etc.)
- `config.go`: Configuration types and defaults

**crawler/**
- `crawler.go`: URL discovery, link extraction, deduplication

**tester/**
- `tester.go`: Load testing engine, worker pool, rate limiting

**reporter/**
- `reporter.go`: HTML/JSON/console report generation

**validator/**
- `validator.go`: Performance target validation

**config/**
- `loader.go`: Configuration file loading and merging

## Development Workflow

### Setup

```bash
cd /Users/vmx/workspace/gocode/src/github.com/vnykmshr/lobster

# Install dependencies
go mod download

# Build
go build -o lobster cmd/lobster/main.go

# Run
./lobster -url http://localhost:3000
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/crawler

# Verbose output
go test -v ./...
```

### Building

```bash
# Development build
go build -o lobster cmd/lobster/main.go

# Production build (optimized)
go build -ldflags="-s -w" -o lobster cmd/lobster/main.go

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o lobster-linux cmd/lobster/main.go
GOOS=darwin GOARCH=arm64 go build -o lobster-macos cmd/lobster/main.go
GOOS=windows GOARCH=amd64 go build -o lobster.exe cmd/lobster/main.go
```

## Key Implementation Details

### Rate Limiting (tester/tester.go)

Uses goflow's token bucket for smooth rate limiting:

```go
rateLimiter, err = bucket.NewSafe(bucket.Limit(config.Rate), burst)
if err := t.rateLimiter.Wait(ctx); err != nil {
    // Handle cancellation
}
```

**Burst capacity**: 2x the rate per second allows short bursts while maintaining overall limit.

### URL Discovery (crawler/crawler.go)

```go
// Pattern: href=["']([^"']+)["']
// Handles: <a href="..."> extraction
// Filters: javascript:, mailto:, # links
// Resolves: Relative URLs to absolute
// Dedupes: sync.Map for discovered URLs
```

### Concurrent Testing (tester/tester.go)

```go
// Worker pool pattern
for i := 0; i < concurrency; i++ {
    wg.Add(1)
    go t.worker(ctx, &wg)
}

// Thread-safe result collection
validationsMutex.Lock()
t.results.URLValidations = append(...)
validationsMutex.Unlock()
```

### Percentile Calculation (validator/validator.go)

```go
// Sort response times
sort.Slice(responseTimes, ...)

// Calculate percentile index
p95Index := int(float64(len(responseTimes)) * 0.95)
p95ResponseTime = responseTimes[p95Index]
```

## Current State (v0.1.0)

### Completed Features
- ✅ Core stress testing engine
- ✅ URL crawling and discovery
- ✅ Rate limiting via goflow
- ✅ HTML/JSON/console reports
- ✅ Performance validation
- ✅ CLI with comprehensive flags
- ✅ Configuration file support
- ✅ Documentation (README, QUICKSTART, ROADMAP)

### Known Limitations
- No authentication support
- HTTP/HTTPS only (no WebSocket, GraphQL)
- Single-machine only (no distributed testing)
- Basic HTML parsing (no JavaScript rendering)
- No request recording/replay

### Technical Debt
- Need comprehensive unit tests (currently 0%)
- No integration tests
- HTML template is inline (should be separate file)
- No benchmarks for performance testing
- Global mutexes in tester (could be per-Tester instance)

## Roadmap Context

### Phase 1 Priorities (v0.1-v0.3)
Focus on stability and usability:
1. Add comprehensive test coverage (>70%)
2. Improve error handling and messages
3. Add CSV export format
4. Configurable report templates

### Phase 2 Priorities (v0.4-v0.6)
Authentication and real-world scenarios:
1. Cookie-based auth (HIGHEST PRIORITY - RICE: 85)
2. HAR file replay (HIGH PRIORITY - RICE: 72)
3. Custom headers and request bodies

### Key Dependencies

**goflow** (v1.0.3)
- Purpose: Token bucket rate limiting
- Location: `internal/tester/tester.go`
- Why: Your project, works well, no reason to replace
- Alternative considered: stdlib rate.Limiter (less flexible)

## Common Development Tasks

### Adding a New Report Format

1. Add method to `reporter/reporter.go`:
   ```go
   func (r *Reporter) GenerateCSV(outputPath string) error
   ```

2. Update `cmd/lobster/main.go` to detect format and call method

3. Add example to `docs/QUICKSTART.md`

### Adding a New Performance Target

1. Add field to `domain.PerformanceTargets`:
   ```go
   MaxMemoryMB float64 `json:"max_memory_mb"`
   ```

2. Update `domain.DefaultPerformanceTargets()`

3. Add validation in `validator.ValidateResults()`:
   ```go
   v.targets = append(v.targets, domain.PerformanceTarget{...})
   ```

### Adding Authentication Support

1. Add auth config to `domain.Config`:
   ```go
   AuthType string `json:"auth_type"` // "cookie", "jwt", "basic"
   Credentials map[string]string `json:"credentials"`
   ```

2. Create `internal/auth/` package with auth handlers

3. Integrate in `tester/tester.go` before making requests

4. Update examples and docs

## Testing Strategy

### Unit Tests (Target: >70% coverage)

```bash
internal/
├── crawler/
│   └── crawler_test.go      # URL extraction, deduplication
├── validator/
│   └── validator_test.go    # Percentile calculations, validation
├── reporter/
│   └── reporter_test.go     # Template rendering, data prep
└── config/
    └── loader_test.go       # Config loading, merging
```

### Integration Tests

```bash
tests/
├── integration/
│   ├── end_to_end_test.go   # Full workflow
│   ├── crawling_test.go     # Multi-page crawl
│   └── reporting_test.go    # Report generation
```

### Test Server

Create a simple test HTTP server for integration tests:

```go
func TestStressTest(t *testing.T) {
    server := httptest.NewServer(...)
    defer server.Close()

    // Run lobster against test server
    // Verify results
}
```

## Release Process

### Version Numbering

Following Semantic Versioning (semver):
- **0.x.y**: Pre-1.0, may have breaking changes in minor versions
- **1.x.y**: Stable API, breaking changes only in major versions

### Release Checklist

1. Update version in `cmd/lobster/main.go`
2. Update `CHANGELOG.md` with changes
3. Update `docs/ROADMAP.md` to mark completed items
4. Run all tests: `go test ./...`
5. Build for all platforms
6. Create git tag: `git tag -a v0.x.y -m "Release v0.x.y"`
7. Push tag: `git push origin v0.x.y`
8. Create GitHub release with binaries
9. Update documentation if needed
10. Announce on relevant channels

## Important File Locations

```
lobster/
├── cmd/lobster/main.go              # CLI entry point - 337 lines
├── internal/
│   ├── domain/
│   │   ├── entities.go                # Core types - 95 lines
│   │   └── config.go                  # Config types - 45 lines
│   ├── crawler/crawler.go             # URL discovery - 130 lines
│   ├── tester/tester.go               # Testing engine - 320 lines
│   ├── reporter/reporter.go           # Report generation - 440 lines
│   ├── validator/validator.go         # Validation - 200 lines
│   └── config/loader.go               # Config loading - 90 lines
├── docs/
│   ├── ROADMAP.md                     # Development roadmap
│   └── QUICKSTART.md                  # Quick start guide
├── examples/
│   └── config.example.json            # Example configuration
└── README.md                          # Main documentation
```

## Git Workflow

### Commit Message Format

```
type(scope): description

[optional body]

[optional footer]
```

**Types**: feat, fix, docs, style, refactor, test, chore

**Examples**:
```
feat(auth): add cookie-based authentication support
fix(crawler): handle malformed URLs gracefully
docs(readme): update installation instructions
test(validator): add percentile calculation tests
```

### Branch Strategy

- `main`: Stable, releasable code
- `feature/*`: New features
- `fix/*`: Bug fixes
- `docs/*`: Documentation updates

## Performance Considerations

### Memory Usage

- Current: ~10MB base + ~1KB per discovered URL
- With 1000 URLs: ~11MB
- With 10,000 URLs: ~20MB

### Optimization Opportunities

1. **Body reading**: Currently reads up to 64KB for link extraction
   - Could be optimized with streaming parser

2. **Response time storage**: Stores all response times
   - Could use sampling for large tests

3. **HTML parsing**: Uses regex for href extraction
   - Could use golang.org/x/net/html for complex pages

## Debugging Tips

### Enable Verbose Logging

```bash
./lobster -url http://localhost:3000 -verbose
```

### Common Issues

**No URLs discovered**:
- Check HTML contains `<a href="...">` links
- Verify base URL is accessible
- Try `-follow-links=false` for single URL

**High error rate**:
- Reduce concurrency: `-concurrency 2`
- Lower rate: `-rate 1.0`
- Increase timeout: `-timeout 60s`

**Slow performance**:
- Check server resources
- Monitor with: `top`, `htop`, or Activity Monitor
- Review server logs

## Next Session TODO

When resuming work on Lobster:

1. **Immediate**:
   - [ ] Create GitHub repository
   - [ ] Push code to GitHub
   - [ ] Add unit tests for crawler
   - [ ] Add unit tests for validator

2. **Week 1**:
   - [ ] Set up GitHub Actions CI
   - [ ] Write comprehensive tests
   - [ ] Create demo video/GIF
   - [ ] Write blog post

3. **Week 2-4**:
   - [ ] Gather user feedback
   - [ ] Implement Phase 2 features
   - [ ] Release v0.2.0

## Resources

- **goflow docs**: https://github.com/vnykmshr/goflow
- **Go testing**: https://go.dev/doc/tutorial/add-a-test
- **Semantic versioning**: https://semver.org/
- **Clean Architecture**: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html

## Contact & Contribution

- **Issues**: https://github.com/vnykmshr/lobster/issues
- **Discussions**: https://github.com/vnykmshr/lobster/discussions
- **Pull Requests**: Welcome! See CONTRIBUTING.md

---

**Last Updated**: 2025-10-24
**Maintainer**: @vnykmshr
**Status**: Active Development
