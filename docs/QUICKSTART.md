# Lobster Quick Start Guide

Get up and running with Lobster in 5 minutes.

## Installation

### From Source

```bash
git clone https://github.com/vnykmshr/lobster.git
cd lobster
go build -o lobster cmd/lobster/main.go
```

### Using Go Install

```bash
go install github.com/vnykmshr/lobster/cmd/lobster@latest
```

## Basic Usage

### 1. Test Your Local Application

Start your application on port 3000, then run:

```bash
./lobster -url http://localhost:3000
```

This will:
- Crawl your application for 2 minutes (default duration)
- Use 5 concurrent workers (default concurrency)
- Automatically discover all linked pages
- Generate a comprehensive report

### 2. Customize the Test

```bash
./lobster \
  -url http://localhost:3000 \
  -duration 5m \
  -concurrency 10 \
  -rate 5.0 \
  -output results.json
```

Parameters:
- `-duration 5m`: Run for 5 minutes
- `-concurrency 10`: Use 10 concurrent workers
- `-rate 5.0`: Limit to 5 requests per second
- `-output results.json`: Save results to file

### 3. View Results

After running with `-output`, you'll get:
- `results.json`: Machine-readable results
- `results.html`: Beautiful interactive report

Open the HTML report in your browser:
```bash
open results.html  # macOS
xdg-open results.html  # Linux
```

### 4. Configure Performance Targets

Create a configuration file `config.json`:

```json
{
  "base_url": "http://localhost:3000",
  "concurrency": 10,
  "duration": "5m",
  "rate": 10.0,
  "output_file": "results.json",
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

Run with config:
```bash
./lobster -config config.json
```

**Expected Output:**
Lobster validates performance against configured targets and reports pass/fail status.

## Understanding Results

Console output includes summary statistics and performance validation:

```
=== STRESS TEST RESULTS ===
Duration: 2m0s | URLs Discovered: 42 | Total Requests: 2,450
Successful: 2,442 | Failed: 8 | Success Rate: 99.67%
Average Response Time: 18.7ms | Requests/Second: 20.4

🎯 PERFORMANCE TARGET VALIDATION
✅ PASS Requests per Second:         20.4 req/s
✅ PASS Average Response Time:        18.7ms
✅ PASS 95th Percentile Response:    35.2ms
✅ PASS Success Rate:                99.67%
Overall: 4/4 targets met (100.0%)
```

**Key Metrics:**
- **Success Rate**: >99% for production
- **p95 Response Time**: 95% of requests faster than this value
- **Requests/Second**: Sustained throughput capacity

## Next Steps

- Read the [full documentation](../README.md)
- Check out [example configurations](../examples/)
- Review the [roadmap](ROADMAP.md) for upcoming features
- [Contribute](../CONTRIBUTING.md) to the project

## Getting Help

- [Full Documentation](../README.md)
- [GitHub Discussions](https://github.com/vnykmshr/lobster/discussions)
- [Report Issues](https://github.com/vnykmshr/lobster/issues)
