# Lobster Development Guide

## Project Overview

**Lobster** is an intelligent web application stress testing tool that automatically discovers URLs through crawling and validates performance under concurrent load.

**Repository**: https://github.com/vnykmshr/lobster
**Version**: 0.1.0
**Created**: 2024-10-24
**License**: MIT

## Quick Context

### What Makes Lobster Different

1. **Crawler-First**: Automatically discovers URLs (unlike ab, wrk, hey, vegeta)
2. **Validation-Focused**: Pass/fail criteria vs raw metrics (unlike k6, JMeter)
3. **Zero-Config**: Works out of the box (unlike enterprise tools)
4. **Smart Rate Limiting**: Uses goflow token bucket to respect server capacity

### Origin Story

Graduated from `markgo/examples/stress-test` on 2024-10-24. Originally built to test MarkGo blog performance, evolved into a general-purpose tool.

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
git clone https://github.com/vnykmshr/lobster.git
cd lobster
go mod download
make build
./lobster -url http://localhost:3000
```

See `Makefile` for all build targets: `make help`

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
- No authentication support (planned for v0.4)
- HTTP/HTTPS only
- Single-machine (no distributed testing)
- Basic HTML parsing (no JavaScript)

### Test Coverage
- `internal/tester`: 86.9%
- `internal/config`: 95.2%
- `internal/crawler`: 94.9%
- `internal/validator`: 51.2%
- `internal/domain`: 100%
- **Overall**: 30.2%

See TESTING.md for details.

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

## Common Tasks

### Adding a Feature
1. Update domain models in `internal/domain/`
2. Implement logic in appropriate package (`tester/`, `crawler/`, etc.)
3. Add tests achieving 70%+ coverage
4. Update CLI flags in `cmd/lobster/main.go`
5. Document in README.md

### Running Tests
See TESTING.md for complete testing guide.

```bash
make test          # Fast tests only
make test-verbose  # Full suite with output
make coverage      # Coverage report
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

## Key Files

```
cmd/lobster/main.go      # CLI entry point
internal/tester/         # Core testing engine
internal/crawler/        # URL discovery
internal/reporter/       # Report generation
internal/domain/         # Business entities
Makefile                 # Build targets
.golangci.yml            # Linter config
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

## Runbook

### Out of Memory
**Symptoms**: Process killed, OOM errors
**Fix**: Reduce `-max-depth`, lower `-queue-size`, or use `-dry-run` to preview URL count first

### High Error Rate
**Symptoms**: Many 429 or timeout errors
**Fix**: Reduce `-concurrency`, lower `-rate`, increase `-timeout`

### No URLs Discovered
**Symptoms**: Only tests base URL
**Fix**: Check HTML has `<a>` links, verify `-follow-links=true`, ensure same domain

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

**Last Updated**: 2024-10-25
**Maintainer**: @vnykmshr
**Status**: Active Development
