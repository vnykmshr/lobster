# Lobster

> The stress tester that thinks like a user

**Lobster** is an intelligent web application stress testing tool that automatically discovers your application's URLs through crawling and validates performance under concurrent load. Unlike traditional stress testing tools that require manual URL configuration, Lobster explores your application like a real user would, providing comprehensive performance validation with zero configuration.

## Why Lobster?

Traditional stress testing tools are either too simple (just hammer a single endpoint) or too complex (requiring extensive scripting and setup). Lobster fills the gap:

- **Zero Configuration**: Point it at your app and go
- **Intelligent Discovery**: Automatically crawls and discovers all URLs
- **Production-Ready Validation**: Get clear pass/fail criteria, not just raw metrics
- **Beautiful Reports**: HTML and JSON reports with visualizations out of the box
- **Developer-Friendly**: Built for CI/CD pipelines and local development

## Features

### Core Capabilities

🔍 **Automatic URL Discovery**
- Intelligent crawling that follows links like a real user
- Configurable depth and same-domain restriction
- Deduplication and cycle prevention

⚡ **Concurrent Load Testing**
- Configurable concurrency levels
- Smart rate limiting to respect server capacity
- Token bucket algorithm for smooth load distribution

📊 **Comprehensive Reporting**
- Real-time progress monitoring
- HTML reports with interactive charts
- JSON output for programmatic analysis
- Response time distribution and percentile analysis

🎯 **Performance Validation**
- Configurable performance targets
- Automatic pass/fail criteria
- Competitive benchmarking support
- Production-readiness scoring

📈 **Rich Metrics**
- Response time statistics (avg, min, max, p95, p99)
- Throughput (requests per second)
- Success/failure rates
- Slow request identification
- Error categorization and tracking

### Advanced Features

🔒 **Smart Rate Limiting**
- Token bucket algorithm via [goflow](https://github.com/vnykmshr/goflow)
- Respects server rate limits
- Prevents overwhelming applications during testing

🔗 **Link Following**
- HTML parsing for href extraction
- Relative URL resolution
- Configurable crawl depth
- Internal link filtering

⏱️ **Flexible Duration**
- Time-based test execution
- Context-aware cancellation
- Graceful shutdown

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

Test your local application:

```bash
lobster -url http://localhost:3000
```

That's it! Lobster will:
1. Crawl your application starting from the base URL
2. Discover all linked pages
3. Generate concurrent requests
4. Validate performance
5. Generate beautiful HTML and JSON reports

### Common Use Cases

**Development Validation**
```bash
# Quick check during development
lobster -url http://localhost:3000 -duration 30s -concurrency 5
```

**Pre-Deployment Testing**
```bash
# Comprehensive test before release
lobster -url https://staging.example.com \
  -duration 10m \
  -concurrency 25 \
  -max-depth 5 \
  -output pre-deploy-results.json
```

**CI/CD Integration**
```bash
# Run in CI pipeline
lobster -url http://localhost:3000 \
  -duration 2m \
  -concurrency 10 \
  -output ci-results.json
```

**High-Load Stress Testing**
```bash
# Simulate high traffic
lobster -url https://example.com \
  -concurrency 100 \
  -duration 15m \
  -rate 50 \
  -output load-test.json
```

## Configuration

### Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `-url` | `http://localhost:3000` | Base URL to test |
| `-concurrency` | `5` | Number of concurrent workers |
| `-duration` | `2m` | Test duration (e.g., 30s, 5m, 1h) |
| `-timeout` | `30s` | Request timeout |
| `-rate` | `2.0` | Requests per second limit |
| `-user-agent` | `Lobster/1.0` | User agent string |
| `-follow-links` | `true` | Follow links found in pages |
| `-max-depth` | `3` | Maximum crawl depth |
| `-output` | - | Output file for results (JSON) |
| `-verbose` | `false` | Verbose logging |
| `-config` | - | Path to configuration file |

### Configuration File

Create a JSON configuration file for complex scenarios:

```json
{
  "base_url": "http://localhost:3000",
  "concurrency": 20,
  "duration": "5m",
  "timeout": "30s",
  "rate": 10.0,
  "user_agent": "Lobster/1.0",
  "follow_links": true,
  "max_depth": 3,
  "output_file": "results.json",
  "verbose": true,
  "performance_targets": {
    "requests_per_second": 1000,
    "avg_response_time_ms": 30,
    "p95_response_time_ms": 50,
    "p99_response_time_ms": 100,
    "success_rate": 99.0,
    "error_rate": 1.0
  }
}
```

Use with:
```bash
lobster -config config.json
```

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
├── cmd/lobster/          # Application entry point
├── internal/
│   ├── domain/            # Core entities and interfaces
│   ├── crawler/           # URL discovery logic
│   ├── tester/            # Load testing engine
│   ├── reporter/          # Report generation
│   ├── validator/         # Performance validation
│   └── config/            # Configuration management
├── pkg/                   # Public reusable packages
├── docs/                  # Documentation
└── examples/              # Example configurations
```

### Key Components

- **Domain Layer**: Core business entities (TestResults, URLTask, etc.)
- **Crawler**: Intelligent URL discovery and link extraction
- **Tester**: Concurrent request execution with rate limiting
- **Reporter**: HTML and JSON report generation
- **Validator**: Performance target validation and scoring
- **Config**: Configuration loading and management

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

### Development Testing
Validate changes during development:
- Quick smoke tests
- Performance regression detection
- Link validation

### Pre-Deployment Validation
Ensure production readiness:
- Comprehensive stress testing
- Performance target validation
- Error detection

### CI/CD Integration
Automate testing in pipelines:
- Fail builds on performance degradation
- Track metrics over time
- Generate compliance reports

### Capacity Planning
Understand application limits:
- Determine max concurrent users
- Identify bottlenecks
- Plan infrastructure scaling

### Competitive Analysis
Benchmark against competitors:
- Compare response times
- Measure throughput differences
- Validate performance claims

## Roadmap

See [ROADMAP.md](docs/ROADMAP.md) for detailed development plans.

### Phase 1: Foundation (v0.1.0) ✅ Current
- Core crawling and stress testing
- Basic reporting (HTML/JSON)
- Performance validation
- Rate limiting

### Phase 2: Authentication & Sessions (v0.2.0)
- Cookie-based authentication
- JWT support
- Session management
- Custom headers

### Phase 3: Advanced Testing (v0.3.0)
- GraphQL support
- WebSocket testing
- Request replay from HAR files
- Custom scenarios

### Phase 4: Enterprise Features (v0.4.0)
- Distributed testing
- Historical trend analysis
- Advanced CI/CD integration
- Plugin architecture

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/vnykmshr/lobster.git
cd lobster

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o lobster cmd/lobster/main.go

# Run locally
./lobster -url http://localhost:3000
```

## Responsible Use

⚠️ **Important**: Lobster is a powerful testing tool that must be used responsibly and ethically.

**Key Requirements:**
- ✅ **Only test systems you own** or have explicit written permission to test
- ✅ **Respect robots.txt** directives (enabled by default)
- ✅ **Configure appropriate rate limits** to avoid service disruption
- ✅ **Handle test reports securely** - they may contain sensitive URLs and data

**Unauthorized load testing may be illegal** and can result in:
- Criminal prosecution under computer fraud laws
- Civil lawsuits for damages
- Termination of employment or contracts
- Permanent legal and professional consequences

Please read our comprehensive [Responsible Use Guidelines](RESPONSIBLE_USE.md) before using Lobster.

**When in doubt, get it in writing.** A simple email confirmation from a system owner can protect you legally.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [goflow](https://github.com/vnykmshr/goflow) for rate limiting
- Inspired by the need for simple yet powerful web stress testing
- Created to fill the gap between simple tools and enterprise solutions

## Support

- 📖 Documentation: [docs/](docs/)
- 🐛 Bug Reports: [GitHub Issues](https://github.com/vnykmshr/lobster/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/vnykmshr/lobster/discussions)
- 🌟 Star the project if you find it useful!

---

**Made with ❤️ for developers who value simplicity and power**
