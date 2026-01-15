---
title: Limitations
nav_order: 5
---

# Limitations

Lobster is designed as a single-machine load testing tool for web applications. This document outlines its limitations and when to use alternative approaches.

## Single-Machine Architecture

Lobster runs on a single machine and cannot distribute load across multiple servers.

### What This Means

- Maximum concurrent connections limited by your machine's resources
- Network bandwidth capped at your connection speed
- CPU and memory constrained to single machine
- Load originates from one IP address

### Practical Limits

On a typical modern machine (8-core CPU, 16GB RAM), Lobster can handle:

- 100-500 concurrent connections comfortably
- 1000-5000 requests per second
- 10,000+ discovered URLs (depending on depth and site size)

Performance degrades beyond these limits due to:
- Operating system file descriptor limits
- Network stack overhead
- Context switching costs
- Memory pressure

### When Single-Machine is Sufficient

Lobster works well for:

- Testing applications during development
- Load testing staging environments
- Validating performance targets before deployment
- Regression testing in CI/CD pipelines
- Small to medium-scale load testing
- Most web applications with moderate traffic

### When to Use Distributed Tools

Consider distributed load testing tools when you need:

- More than 1000 concurrent connections
- Load from multiple geographic locations
- Very high requests per second (10,000+)
- Simulating massive user bases
- Testing CDN behavior across regions
- Bypassing rate limiting by IP

Distributed alternatives: k6, Gatling, JMeter with distributed mode, or cloud services like AWS Load Testing.

## Memory Usage

Lobster stores discovered URLs and test results in memory during execution.

### Memory Consumption

Approximate memory usage per URL:

- URL string: 100-500 bytes (average 200 bytes)
- Validation result: 150 bytes
- Response time entry: 50 bytes
- Total per URL: ~400 bytes

### Sizing Guidelines

| URLs Discovered | Estimated RAM | Recommended System RAM |
|-----------------|---------------|------------------------|
| 1,000           | ~4 MB         | 4 GB                   |
| 10,000          | ~40 MB        | 8 GB                   |
| 50,000          | ~200 MB       | 16 GB                  |
| 100,000         | ~400 MB       | 32 GB                  |

These are conservative estimates. Actual usage depends on:

- URL length (query parameters increase size)
- Number of errors collected
- Response time samples stored
- Slow request tracking

### Controlling Memory Usage

Use these strategies to reduce memory consumption:

**Limit Crawl Depth**

Shallow crawls discover fewer URLs:

```bash
lobster -url http://example.com -max-depth 2
```

Recommended depth by site size:
- Small sites (<1000 pages): depth 5+
- Medium sites (1000-10000 pages): depth 3-4
- Large sites (10000+ pages): depth 1-2

**Limit Queue Size**

Prevent unbounded URL queue growth:

```bash
lobster -url http://example.com -queue-size 5000
```

Default queue size is 10,000 URLs (~80KB overhead).

**Use Dry-Run First**

Preview URL discovery before testing:

```bash
lobster -url http://example.com -dry-run
```

This shows how many URLs will be discovered without consuming memory for test results.

**Shorter Test Duration**

Reduce result accumulation:

```bash
lobster -url http://example.com -duration 2m
```

**Disable Link Following**

Test specific URLs without crawling:

```bash
lobster -url http://example.com -follow-links=false
```

### Out of Memory Scenarios

If Lobster runs out of memory:

1. Reduce -max-depth
2. Lower -queue-size
3. Shorten -duration
4. Disable -follow-links
5. Test subsets of your site separately

Future versions may add result streaming to disk for very large tests.

## Network Limitations

### Rate Limiting

Lobster respects rate limits when configured properly:

```bash
lobster -url http://example.com -rate 10 -concurrency 5
```

But all requests come from one IP address, which may trigger:
- IP-based rate limiting
- WAF rules
- DDoS protection systems

Use the -respect-429 flag (enabled by default) to back off when rate limited.

### Bandwidth

Single machine bandwidth limits throughput:

- Home connection: 10-100 Mbps
- Office connection: 100-1000 Mbps
- Data center: 1-10 Gbps

High concurrency with large responses can saturate your connection.

## Protocol Support

Currently supported:
- HTTP/1.1
- HTTPS with TLS 1.2+

Not currently supported:
- HTTP/2
- HTTP/3 / QUIC
- WebSockets
- Server-Sent Events
- gRPC

## Testing Scope

### Link Discovery

Lobster discovers links by parsing HTML:

- Only follows <a href="..."> links
- Ignores JavaScript-rendered content
- Ignores dynamically loaded content
- Misses AJAX endpoints
- Misses form submissions

For complete API testing, use explicit URL lists or API-specific tools.

### Same-Domain Only

Lobster only follows links within the same domain:

- http://example.com/ → http://example.com/page (followed)
- http://example.com/ → http://other.com/ (not followed)
- http://example.com/ → http://api.example.com/ (not followed - different subdomain)

This prevents tests from accidentally crawling external sites.

### Authentication

Supported authentication methods:

- Basic HTTP authentication
- Bearer token
- Cookie-based sessions
- Custom headers

Not supported:
- OAuth flows requiring browser redirects
- Multi-factor authentication
- CAPTCHA
- Dynamic client-side authentication

For complex auth flows, obtain tokens manually and provide via command-line flags.

## Timing and Accuracy

### Response Time Measurement

Lobster measures total request time including:

- DNS lookup
- TCP connection
- TLS handshake
- Request transmission
- Server processing
- Response transmission

This differs from server-side metrics which only measure processing time.

### Clock Precision

Response times measured with millisecond precision. Nanosecond-level accuracy is not guaranteed across all operations.

### Concurrency Model

Lobster uses goroutines for concurrency:

- Lightweight compared to threads
- Can handle thousands of concurrent requests
- Limited by OS resources, not goroutine overhead

## File System

### Report Generation

Reports written to local filesystem:

- HTML reports: single file, self-contained
- JSON reports: machine-readable results
- File permissions: 0600 (owner read/write only)

Large result sets create large report files:
- 10,000 URLs: ~5-10 MB HTML report
- 100,000 URLs: ~50-100 MB HTML report

### Temporary Files

Lobster does not create temporary files during testing. All state is in memory until final report generation.

## Working Around Limitations

### Multiple Instance Coordination

While Lobster doesn't have built-in distributed support, you can manually coordinate multiple instances:

**Run from Different Machines**

```bash
# Machine 1
lobster -url http://example.com -output results-1.json

# Machine 2
lobster -url http://example.com -output results-2.json

# Combine results manually
```

**Test Different Subsets**

```bash
# Instance 1: Marketing pages
lobster -url http://example.com/marketing

# Instance 2: Application pages
lobster -url http://example.com/app

# Instance 3: API endpoints
lobster -url http://example.com/api
```

**Stagger Start Times**

Avoid coordinated load spikes:

```bash
lobster -url http://example.com &
sleep 30
lobster -url http://example.com &
```

### Memory-Constrained Environments

For very large sites that exceed memory:

1. Test in phases (different sections separately)
2. Use shallower crawl depth
3. Disable link following and provide URL list
4. Increase system RAM
5. Use dry-run to estimate scope first

## Getting Help

If you hit limitations not documented here:

- Check existing [GitHub issues](https://github.com/vnykmshr/lobster/issues)
- Open a new issue with your use case
- Consider if your needs require distributed testing

We're always interested in understanding real-world usage and limitations.
