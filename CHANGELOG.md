# Changelog

All notable changes to Lobster will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] - 2026-01-15

Major security hardening release with significant performance improvements.

### Added

- **Environment variable substitution in config files**: Use `${VAR_NAME}` or `${VAR_NAME:-default}` syntax to reference environment variables in JSON config files
- **Secure credential input**: `--auth-password-stdin` and `--auth-token-stdin` flags for piping secrets safely
- **Environment variable auth support**: `LOBSTER_AUTH_PASSWORD`, `LOBSTER_AUTH_TOKEN`, `LOBSTER_AUTH_COOKIE` environment variables
- **URL validation with SSRF protection**: Blocks `file://`, `ftp://`, and private IP ranges by default
- **`--allow-private-ips` flag**: Explicitly enable testing against internal/localhost URLs when needed
- **Response size validation**: `--max-response-size` flag (default 10MB) prevents memory exhaustion from large responses
- **Dropped URL tracking**: Warns when URLs are dropped due to queue overflow with actionable hints
- **Config and auth validation methods**: Early detection of invalid configuration before test execution

### Changed

- **Credential handling (breaking)**: Passwords, tokens, and cookies can no longer be passed via CLI flags. Use environment variables or stdin instead
- **Insecure TLS confirmation (breaking)**: `--insecure-skip-verify` now requires `LOBSTER_INSECURE_TLS=true` environment variable confirmation
- **HTTP connection pooling**: Significantly improved throughput with proper `http.Transport` configuration
- **GetDiscoveredCount performance**: Changed from O(n) to O(1) using atomic counters
- **Result channel buffering**: Dynamic buffer sizing based on concurrency to prevent blocking under load
- **HTTP/2 enabled by default**: `ForceAttemptHTTP2: true` for HTTPS connections
- **429 retry timing**: Response time now correctly reflects only the actual request duration, not retry wait time
- **CLI architecture**: Extracted CLI functions into `internal/cli` package for improved testability and maintainability

### Removed

- **`--auth-password` CLI flag**: Use `LOBSTER_AUTH_PASSWORD` env var or `--auth-password-stdin` instead
- **`--auth-token` CLI flag**: Use `LOBSTER_AUTH_TOKEN` env var or `--auth-token-stdin` instead
- **`--auth-cookie` CLI flag**: Use `LOBSTER_AUTH_COOKIE` env var instead

**Note:** `--auth-header` flag remains available as header names are not secrets.

### Fixed

- Error message sanitization to prevent leaking internal infrastructure details (IPs, hostnames)
- Robots.txt wildcard pattern matching now correctly handles `*` patterns per RFC specification
- Username no longer logged in debug output (prevents enumeration)
- Cookie and header validation now provides explicit error messages for malformed input
- Mutual exclusivity check for `--auth-password-stdin` and `--auth-token-stdin`
- Environment variable substitution properly distinguishes empty values from unset variables
- EOF handling uses `errors.Is(err, io.EOF)` instead of string comparison

### Security

- **Credentials no longer exposed in process list**: Passwords and tokens removed from CLI flags
- **SSRF protection**: Private IP ranges (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16) blocked by default
- **URL scheme validation**: Only `http://` and `https://` schemes permitted
- **Insecure TLS requires explicit consent**: Two-factor confirmation (flag + env var) for skipping TLS verification
- **Response size limits**: Prevents memory exhaustion from malicious large responses
- **Error message sanitization**: Internal infrastructure details redacted from user-facing output

### Migration Guide

#### Credential Handling

Before (v1.0.0):
```bash
lobster -url https://api.example.com -auth-type basic -auth-username admin -auth-password secret
```

After (v2.0.0):
```bash
# Option 1: Environment variable
LOBSTER_AUTH_PASSWORD=secret lobster -url https://api.example.com -auth-type basic -auth-username admin

# Option 2: Stdin (for scripts)
echo "secret" | lobster -url https://api.example.com -auth-type basic -auth-username admin -auth-password-stdin
```

#### Config File Credentials

Before (v1.0.0):
```json
{
  "auth": {
    "type": "bearer",
    "token": "my-secret-token"
  }
}
```

After (v2.0.0):
```json
{
  "auth": {
    "type": "bearer",
    "token": "${API_TOKEN}"
  }
}
```

Then run with:
```bash
API_TOKEN=my-secret-token lobster -config config.json
```

#### Insecure TLS

Before (v1.0.0):
```bash
lobster -url https://self-signed.example.com -insecure-skip-verify
```

After (v2.0.0):
```bash
LOBSTER_INSECURE_TLS=true lobster -url https://self-signed.example.com -insecure-skip-verify
```

#### Testing Internal Services

Before (v1.0.0):
```bash
lobster -url http://localhost:8080
```

After (v2.0.0):
```bash
lobster -url http://localhost:8080 -allow-private-ips
```

---

## [1.0.0] - 2025-10-26

Initial production release of Lobster, a lightweight HTTP load testing and link validation tool.

### Added

- HTTP stress testing with configurable concurrency and rate limiting
- Recursive link discovery and validation
- Support for basic, bearer, cookie, and header-based authentication
- robots.txt compliance (with `-ignore-robots` override)
- HTTP 429 rate limit handling with exponential backoff
- Real-time progress bar with request statistics
- HTML and JSON report generation
- Dry-run mode for URL discovery without load testing
- Configurable request timeout and test duration
- Performance targets with P95/P99 latency thresholds

[Unreleased]: https://github.com/vnykmshr/lobster/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/vnykmshr/lobster/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/vnykmshr/lobster/releases/tag/v1.0.0
