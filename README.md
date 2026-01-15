# Lobster

**Lobster** is an intelligent web stress testing tool that automatically discovers URLs by crawling your application and validates performance under load. Point it at your app, and it handles the rest—zero configuration required.

## Why Lobster?

- **Zero Config**: Point and shoot
- **Auto-Discovery**: Crawls and finds all URLs
- **Pass/Fail Validation**: Not just metrics
- **Beautiful Reports**: HTML + JSON + console
- **CI/CD Ready**: Fast, reliable, scriptable

## Features

- **Auto URL Discovery**: Crawls and discovers all linked pages
- **Concurrent Testing**: Configurable workers with rate limiting
- **Performance Validation**: Pass/fail against targets (p95, p99, success rate)
- **Rich Reports**: HTML (charts), JSON (API), console (real-time)
- **Smart Rate Limiting**: Token bucket via [goflow](https://github.com/vnykmshr/goflow)
- **robots.txt Compliance**: Respects website preferences by default

## Quick Start

### Installation

```bash
go install github.com/vnykmshr/lobster/cmd/lobster@latest
```

Or clone and build:

```bash
git clone https://github.com/vnykmshr/lobster.git
cd lobster
go build -o lobster cmd/lobster/main.go
```

### Basic Usage

Test a public website:

```bash
lobster -url https://example.com -duration 30s
```

Test your local application:

```bash
lobster -url http://localhost:3000 -allow-private-ips
```

Lobster crawls your app, tests all discovered URLs under load, and generates reports.

For more examples, see [QUICKSTART.md](docs/QUICKSTART.md).

## Configuration

Key flags: `-url`, `-concurrency`, `-duration`, `-rate`, `-max-depth`, `-output`

Use `-config config.json` for complex setups. See `examples/config.example.json` for template.

## How It Works

### 1. URL Discovery Phase
- Starts with the base URL
- Parses HTML responses for `<a href>` links
- Resolves relative URLs to absolute
- Filters to same-domain links only
- Maintains a queue of discovered URLs
- Respects max depth configuration

### 2. Concurrent Testing Phase
- Spawns configurable number of worker goroutines
- Distributes URLs across workers
- Applies rate limiting via token bucket
- Tracks response times and status codes
- Continues until duration expires

### 3. Analysis Phase
- Calculates aggregate metrics
- Computes percentiles (p95, p99)
- Validates against performance targets
- Identifies slow requests and errors

### 4. Reporting Phase
- Generates JSON output with detailed metrics
- Creates HTML report with interactive charts
- Provides console summary with pass/fail indicators
- Outputs actionable recommendations

## Architecture

Lobster follows **Clean Architecture** principles:

```
lobster/
├── cmd/lobster/       # CLI entry point
├── internal/
│   ├── cli/            # CLI utilities
│   ├── config/         # Configuration loading
│   ├── crawler/        # URL discovery
│   ├── domain/         # Core entities
│   ├── reporter/       # Report generation
│   ├── robots/         # robots.txt parsing
│   ├── tester/         # Load testing engine
│   ├── util/           # Shared utilities
│   └── validator/      # Performance validation
├── docs/               # Documentation
└── examples/           # Example configs
```

**Domain** → **Crawler** → **Tester** → **Reporter** + **Validator**

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for details.

## Reports

### Console Output
Real-time progress updates and detailed summary:

```
=== STRESS TEST RESULTS ===
Duration: 2m0s
URLs Discovered: 42
Total Requests: 2,450
Successful Requests: 2,442
Failed Requests: 8
Average Response Time: 18.7ms
Requests/Second: 20.4
Success Rate: 99.67%

🎯 PERFORMANCE TARGET VALIDATION
============================================================
✅ PASS Requests per Second:         20.4 req/s
✅ PASS Average Response Time:        18.7ms
✅ PASS 95th Percentile Response Time: 35.2ms
✅ PASS Success Rate:                 99.67%

Overall: 4/4 targets met (100.0%)
🎉 ALL PERFORMANCE TARGETS MET!
```

### HTML Report
Beautiful, interactive report with:
- Overview dashboard with key metrics
- Response status distribution (pie charts)
- URL validation table (sortable)
- Response time histogram
- Error analysis
- Slow request identification

### JSON Report
Machine-readable format for integration:
```json
{
  "duration": "2m0s",
  "urls_discovered": 42,
  "total_requests": 2450,
  "successful_requests": 2442,
  "failed_requests": 8,
  "average_response_time": "18.7ms",
  "requests_per_second": 20.4,
  "success_rate": 99.67,
  "performance_validation": {
    "targets_met": 4,
    "total_targets": 4,
    "overall_status": "PRODUCTION_READY"
  }
}
```

## Use Cases

- Development testing and regression detection
- Pre-deployment validation
- CI/CD performance gates
- Capacity planning

## Roadmap

- **v0.1**: Core functionality ✅
- **v0.2-0.3**: Enhanced reporting, configuration
- **v0.4-0.6**: Authentication (cookie, JWT, OAuth)
- **v0.7-0.9**: GraphQL, WebSockets, HAR replay
- **v1.0+**: Distributed testing, CI/CD integration

See [ROADMAP.md](docs/ROADMAP.md) for details.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

### Development

```bash
git clone https://github.com/vnykmshr/lobster.git
cd lobster
make help      # See all targets
make test      # Run tests
make build     # Build binary
```

See [DEVELOPMENT.md](docs/DEVELOPMENT.md) and [CONTRIBUTING.md](docs/CONTRIBUTING.md).

## Responsible Use

⚠️ **Only test systems you own or have written permission to test.**

Unauthorized testing may violate computer fraud laws. Respect robots.txt (enabled by default), use rate limits, handle reports securely.

See [RESPONSIBLE_USE.md](docs/RESPONSIBLE_USE.md) for full guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/vnykmshr/lobster/issues)
- **Discussions**: [GitHub Discussions](https://github.com/vnykmshr/lobster/discussions)

Built with [goflow](https://github.com/vnykmshr/goflow) • MIT License
