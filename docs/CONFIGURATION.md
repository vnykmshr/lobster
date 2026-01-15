# Configuration Reference

Complete reference for all Lobster configuration options, including CLI flags, configuration files, and environment variables.

## Overview

Lobster can be configured through:
1. **CLI flags** - Highest priority, override all other sources
2. **Environment variables** - For sensitive data like credentials
3. **Configuration file** - JSON format for persistent settings
4. **Defaults** - Sensible defaults when no other value is provided

## CLI Flags

### Core Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-url` | string | (required) | Base URL to test |
| `-concurrency` | int | 5 | Number of concurrent workers |
| `-duration` | string | "2m" | Test duration (e.g., "30s", "5m", "1h") |
| `-timeout` | string | "30s" | HTTP request timeout |
| `-rate` | float | 2.0 | Requests per second limit per worker |
| `-user-agent` | string | "Lobster/1.0" | User-Agent header for requests |

### Crawling Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-follow-links` | bool | true | Discover and follow links from HTML pages |
| `-max-depth` | int | 3 | Maximum crawl depth (0 = base URL only) |
| `-queue-size` | int | 10000 | URL queue buffer capacity |
| `-ignore-robots` | bool | false | Ignore robots.txt directives |

### Request Behavior

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-respect-429` | bool | true | Respect HTTP 429 with exponential backoff |
| `-dry-run` | bool | false | Discover URLs without making test requests |

### Security Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-insecure-skip-verify` | bool | false | Skip TLS certificate verification (requires `LOBSTER_INSECURE_TLS=true`) |
| `-allow-private-ips` | bool | false | Allow private/localhost IPs (bypasses SSRF protection) |

### Authentication Flags

| Flag | Type | Description |
|------|------|-------------|
| `-auth-type` | string | Authentication type: `basic`, `bearer`, `cookie`, `header` |
| `-auth-username` | string | Username for basic authentication |
| `-auth-password-stdin` | bool | Read password from stdin (one line) |
| `-auth-token-stdin` | bool | Read bearer token from stdin (one line) |
| `-auth-header` | string | Custom header in "Name:Value" format |

### Output Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-output` | string | "" | Output file for results (JSON or HTML based on extension) |
| `-verbose` | bool | false | Enable verbose JSON logging |
| `-no-progress` | bool | false | Disable progress bar updates |
| `-compare` | string | "" | Compare against target (e.g., "Ghost", "WordPress") |

### Other Flags

| Flag | Description |
|------|-------------|
| `-config` | Path to configuration file (JSON) |
| `-version` | Show version information |
| `-help` | Show help message |

## Environment Variables

Environment variables are used for sensitive configuration that shouldn't appear in command-line arguments (which are visible in process lists).

| Variable | Description | Required For |
|----------|-------------|--------------|
| `LOBSTER_AUTH_PASSWORD` | Password for basic authentication | Basic auth |
| `LOBSTER_AUTH_TOKEN` | Bearer token for API authentication | Bearer auth |
| `LOBSTER_AUTH_COOKIE` | Cookie value for session authentication | Cookie auth |
| `LOBSTER_INSECURE_TLS` | Set to "true" to allow `--insecure-skip-verify` | Insecure TLS |

### Secure Credential Handling

Credentials are **never** passed as CLI flags to prevent exposure in:
- Process listings (`ps aux`)
- Shell history
- Log files

Instead, use:
```bash
# Via environment variables
export LOBSTER_AUTH_PASSWORD="secret"
lobster -url https://example.com -auth-type basic -auth-username admin

# Via stdin
echo "secret" | lobster -url https://example.com -auth-type basic -auth-username admin -auth-password-stdin
```

## Configuration File

Create a JSON configuration file for persistent settings:

```json
{
  "base_url": "https://example.com",
  "concurrency": 10,
  "duration": "5m",
  "timeout": "30s",
  "rate": 5.0,
  "user_agent": "Lobster/1.0",
  "follow_links": true,
  "max_depth": 3,
  "queue_size": 10000,
  "respect_429": true,
  "dry_run": false,
  "verbose": false,
  "ignore_robots": false,
  "output_file": "results.html",
  "auth": {
    "type": "basic",
    "username": "admin",
    "password": "${LOBSTER_AUTH_PASSWORD}"
  },
  "performance_targets": {
    "requests_per_second": 100,
    "avg_response_time_ms": 50,
    "p95_response_time_ms": 100,
    "p99_response_time_ms": 200,
    "success_rate": 99.0,
    "error_rate": 1.0
  }
}
```

### Environment Variable Substitution

Configuration files support `${VAR_NAME}` syntax for environment variable substitution:

```json
{
  "auth": {
    "type": "bearer",
    "token": "${API_TOKEN}"
  }
}
```

Usage:
```bash
export API_TOKEN="your-secret-token"
lobster -config config.json
```

### Auth Configuration Options

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Auth type: `basic`, `bearer`, `cookie`, `header` |
| `username` | string | Username for basic auth |
| `password` | string | Password (use `${ENV_VAR}`) |
| `token` | string | Bearer token (use `${ENV_VAR}`) |
| `cookies` | object | Key-value pairs for cookie auth |
| `headers` | object | Key-value pairs for header auth |
| `cookie_file` | string | Path to Netscape-format cookie file |

### Performance Targets

Define pass/fail thresholds for automated testing:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `requests_per_second` | float | 100 | Minimum acceptable throughput |
| `avg_response_time_ms` | float | 50 | Maximum average response time |
| `p95_response_time_ms` | float | 100 | Maximum 95th percentile response time |
| `p99_response_time_ms` | float | 200 | Maximum 99th percentile response time |
| `success_rate` | float | 99.0 | Minimum success rate percentage |
| `error_rate` | float | 1.0 | Maximum error rate percentage |

## Precedence

Configuration values are merged in this order (later overrides earlier):

1. **Defaults** - Built-in sensible defaults
2. **Configuration file** - Values from `-config` file
3. **Environment variables** - For credentials and sensitive settings
4. **CLI flags** - Highest priority, always wins

Example:
```bash
# Config file sets concurrency=5
# CLI flag overrides to concurrency=20
lobster -config config.json -concurrency 20
```

## Examples

### Basic Usage

```bash
# Simple test with defaults
lobster -url https://example.com

# Custom duration and concurrency
lobster -url https://example.com -duration 5m -concurrency 10

# Rate-limited test
lobster -url https://example.com -rate 10 -duration 2m
```

### With Authentication

```bash
# Basic authentication
export LOBSTER_AUTH_PASSWORD="secret"
lobster -url https://api.example.com -auth-type basic -auth-username admin

# Bearer token authentication
export LOBSTER_AUTH_TOKEN="your-api-token"
lobster -url https://api.example.com -auth-type bearer

# Read token from stdin (e.g., from vault)
vault kv get -field=token secret/api | \
  lobster -url https://api.example.com -auth-type bearer -auth-token-stdin
```

### URL Discovery (Dry Run)

```bash
# Discover all URLs without stress testing
lobster -url https://example.com -dry-run -max-depth 5 -output urls.json
```

### Testing Internal Services

```bash
# Allow localhost/private IPs (development only!)
lobster -url http://localhost:3000 -allow-private-ips

# Skip TLS verification for self-signed certs
export LOBSTER_INSECURE_TLS=true
lobster -url https://dev.internal:8443 -insecure-skip-verify -allow-private-ips
```

### CI/CD Integration

```bash
# Exit with error code on performance target failure
lobster -url https://staging.example.com \
  -config performance-targets.json \
  -output results.html \
  -no-progress

# Check exit code: 0 = pass, 1 = fail
echo "Exit code: $?"
```

## Performance Tuning

### Concurrency

- **Low concurrency (1-5)**: Gentle testing, accurate latency measurements
- **Medium (10-20)**: Balanced load testing
- **High (50+)**: Stress testing, may saturate network/CPU

### Rate Limiting

- **No limit (`-rate 0`)**: Maximum throughput
- **Rate < 1**: Sub-request-per-second (e.g., 0.5 = one request every 2 seconds)
- **Rate >= 1**: Requests per second per worker

Total RPS = `rate × concurrency`

### Queue Size

Default 10,000 is suitable for most sites. Increase for:
- Large sites with many pages
- Deep crawling (`-max-depth` > 5)

Watch for "URLs dropped due to queue overflow" warning.

### Memory Considerations

- Each queued URL uses ~80 bytes
- Queue of 100,000 URLs ≈ 8MB
- Response times are stored for percentile calculation
- Consider `-max-depth` to limit crawl scope

### robots.txt

By default, Lobster respects `robots.txt`. Use `-ignore-robots` only when:
- You own the target site
- You have explicit permission
- Testing internal services with no robots.txt
