---
title: Development
nav_order: 6
---

# Development Guide

## Project Overview

**Lobster** is an intelligent web application stress testing tool that automatically discovers URLs through crawling and validates performance under concurrent load.

**Repository**: https://github.com/vnykmshr/lobster
**Version**: 2.0.0
**License**: MIT

## What Makes Lobster Different

1. **Crawler-First**: Automatically discovers URLs (unlike ab, wrk, hey, vegeta)
2. **Validation-Focused**: Pass/fail criteria vs raw metrics (unlike k6, JMeter)
3. **Zero-Config**: Works out of the box (unlike enterprise tools)
4. **Smart Rate Limiting**: Uses goflow token bucket to respect server capacity
5. **Security Hardened**: SSRF protection, secure credential handling

## Architecture

### Clean Architecture Layers

```
cmd/lobster/            # CLI entry point, flag parsing
  │
internal/cli/           # CLI utilities, auth handling
  │
internal/config/        # Configuration loading and merging
  │
internal/domain/        # Core entities and types
  │
├── internal/crawler/   # URL discovery
├── internal/tester/    # Load testing engine
├── internal/reporter/  # Report generation
├── internal/validator/ # Performance validation
├── internal/robots/    # robots.txt parsing
└── internal/util/      # Shared utilities (URL validation, sanitization)
```

### Component Responsibilities

| Package | Purpose |
|---------|---------|
| `domain/` | Core business entities (TestResults, URLTask, Config) |
| `crawler/` | URL discovery, link extraction, deduplication |
| `tester/` | Load testing engine, worker pool, rate limiting |
| `reporter/` | HTML/JSON/console report generation |
| `validator/` | Performance target validation |
| `config/` | Configuration file loading and merging |
| `cli/` | CLI utilities, auth handling, stdin reading |
| `robots/` | robots.txt parsing with wildcard support |
| `util/` | URL validation, error sanitization |

## Development Workflow

### Setup

```bash
git clone https://github.com/vnykmshr/lobster.git
cd lobster
go mod download
make build
./lobster -url http://localhost:3000 -allow-private-ips
```

See `Makefile` for all build targets: `make help`

### Running Tests

```bash
make test          # Fast tests only
make test-verbose  # Full suite with output
make coverage      # Coverage report
make lint          # Run golangci-lint
```

See [testing](testing) for the complete testing guide.

## Key Implementation Details

### Rate Limiting

Uses goflow's token bucket for smooth rate limiting:

```go
rateLimiter, err = bucket.NewSafe(bucket.Limit(config.Rate), burst)
if err := t.rateLimiter.Wait(ctx); err != nil {
    // Handle cancellation
}
```

**Burst capacity**: 2x the rate per second allows short bursts while maintaining overall limit.

### URL Discovery

```go
// Pattern: href=["']([^"']+)["']
// Handles: <a href="..."> extraction
// Filters: javascript:, mailto:, # links
// Resolves: Relative URLs to absolute
// Dedupes: sync.Map for discovered URLs
```

### Concurrent Testing

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

## Key Dependencies

**goflow** (v1.0.3)
- Purpose: Token bucket rate limiting
- Location: `internal/tester/tester.go`

## Common Tasks

### Adding a Feature

1. Update domain models in `internal/domain/`
2. Implement logic in appropriate package (`tester/`, `crawler/`, etc.)
3. Add tests achieving 70%+ coverage
4. Update CLI flags in `cmd/lobster/main.go`
5. Document in README.md and relevant docs

### Release Checklist

1. Update version in `cmd/lobster/main.go`
2. Update `CHANGELOG.md` with changes
3. Update `docs/roadmap.md` to mark completed items
4. Run all tests: `go test ./...`
5. Build for all platforms
6. Create git tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
7. Push tag: `git push origin vX.Y.Z`
8. Create GitHub release with binaries

## Key Files

```
cmd/lobster/main.go      # CLI entry point
internal/cli/            # CLI utilities
internal/tester/         # Core testing engine
internal/crawler/        # URL discovery
internal/reporter/       # Report generation
internal/domain/         # Business entities
internal/util/           # URL validation, sanitization
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
```

### Branch Strategy

- `main`: Stable, releasable code
- `feature/*`: New features
- `fix/*`: Bug fixes
- `docs/*`: Documentation updates

## Troubleshooting

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

- [goflow docs](https://github.com/vnykmshr/goflow)
- [Go testing](https://go.dev/doc/tutorial/add-a-test)
- [Semantic versioning](https://semver.org/)

## Contact

- **Issues**: https://github.com/vnykmshr/lobster/issues
- **Discussions**: https://github.com/vnykmshr/lobster/discussions
- **Pull Requests**: Welcome! See [contributing](contributing)
