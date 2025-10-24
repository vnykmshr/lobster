# WebStress Quick Start Guide

Get up and running with WebStress in 5 minutes.

## Installation

### From Source

```bash
git clone https://github.com/vnykmshr/webstress.git
cd webstress
go build -o webstress cmd/webstress/main.go
```

### Using Go Install

```bash
go install github.com/vnykmshr/webstress/cmd/webstress@latest
```

## Basic Usage

### 1. Test Your Local Application

Start your application on port 3000, then run:

```bash
./webstress -url http://localhost:3000
```

This will:
- Crawl your application for 2 minutes (default duration)
- Use 5 concurrent workers (default concurrency)
- Automatically discover all linked pages
- Generate a comprehensive report

### 2. Customize the Test

```bash
./webstress \
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
./webstress -config config.json
```

## Common Use Cases

### Development Validation

Quick check during development:
```bash
./webstress -url http://localhost:3000 -duration 30s -concurrency 5
```

### Pre-Deployment Testing

Comprehensive test before release:
```bash
./webstress \
  -url https://staging.example.com \
  -duration 10m \
  -concurrency 25 \
  -max-depth 5 \
  -output pre-deploy.json
```

### Load Testing

Simulate high traffic:
```bash
./webstress \
  -url https://example.com \
  -concurrency 100 \
  -duration 15m \
  -rate 50 \
  -output load-test.json
```

### CI/CD Integration

```bash
#!/bin/bash
# Start your app
./start-app.sh &
APP_PID=$!

# Wait for app to be ready
sleep 5

# Run stress test
./webstress \
  -url http://localhost:3000 \
  -duration 2m \
  -concurrency 10 \
  -output ci-results.json

# Parse results and fail if targets not met
if ! grep -q '"overall_status":"PRODUCTION_READY"' ci-results.json; then
  echo "Performance targets not met!"
  kill $APP_PID
  exit 1
fi

kill $APP_PID
```

## Understanding Results

### Console Output

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
```

### Performance Validation

```
🎯 PERFORMANCE TARGET VALIDATION
============================================================
✅ PASS Requests per Second:         20.4 req/s
✅ PASS Average Response Time:        18.7ms
✅ PASS 95th Percentile Response Time: 35.2ms
✅ PASS Success Rate:                 99.67%

Overall: 4/4 targets met (100.0%)
🎉 ALL PERFORMANCE TARGETS MET!
```

### Key Metrics

- **Success Rate**: Should be >99% for production
- **Avg Response Time**: Lower is better (<50ms is excellent)
- **p95 Response Time**: 95% of requests faster than this
- **Requests/Second**: Throughput capacity
- **Error Rate**: Should be <1% for production

## Tips & Best Practices

### 1. Start Small

Begin with low concurrency and short duration:
```bash
./webstress -url http://localhost:3000 -concurrency 2 -duration 30s
```

### 2. Respect Server Limits

Use rate limiting to avoid overwhelming your server:
```bash
./webstress -url http://localhost:3000 -rate 2.0
```

### 3. Monitor Your Application

While running tests, monitor:
- CPU usage
- Memory consumption
- Database connections
- Response times

### 4. Test Realistic Scenarios

- Enable link following to simulate real user behavior
- Adjust concurrency to match expected traffic
- Use appropriate rate limits

### 5. Analyze Failures

If you see failures:
1. Check the error details in the report
2. Look for patterns (specific URLs, times)
3. Review server logs
4. Adjust concurrency or rate limits

## Next Steps

- Read the [full documentation](../README.md)
- Check out [example configurations](../examples/)
- Review the [roadmap](ROADMAP.md) for upcoming features
- [Contribute](../CONTRIBUTING.md) to the project

## Troubleshooting

### High Error Rate

- Reduce concurrency: `-concurrency 2`
- Lower request rate: `-rate 1.0`
- Increase timeout: `-timeout 60s`
- Check server resources

### Slow Performance

- Monitor server CPU/memory
- Check database query performance
- Review application logs
- Consider caching strategies

### No URLs Discovered

- Verify application is running
- Check base URL is correct
- Ensure HTML contains links
- Try disabling link following: `-follow-links=false`

## Getting Help

- 📖 [Full Documentation](../README.md)
- 💬 [GitHub Discussions](https://github.com/vnykmshr/webstress/discussions)
- 🐛 [Report Issues](https://github.com/vnykmshr/webstress/issues)

Happy stress testing! 🚀
