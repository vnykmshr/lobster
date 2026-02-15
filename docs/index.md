---
title: Home
nav_order: 1
---

# Lobster

Intelligent web stress testing with automated URL discovery.

Lobster crawls your application, discovers all linked pages, tests them under load, and validates performance against targets. Point it at your app, and it handles the rest.

## Key Features

- **Auto URL Discovery**: Crawls and finds all linked pages automatically
- **Concurrent Testing**: Configurable workers with rate limiting
- **Performance Validation**: Pass/fail against targets (p95, p99, success rate)
- **Multi-format Reports**: HTML, JSON, and console output
- **Security Hardened**: SSRF protection, secure credential handling

## Quick Start

### Installation

```bash
go install github.com/1mb-dev/lobster/cmd/lobster@latest
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

See the [Quick Start Guide](quickstart) for more examples.

## Documentation

| Guide | Description |
|-------|-------------|
| [Quick Start](quickstart) | Get up and running in 5 minutes |
| [Configuration](configuration) | CLI flags, environment variables, and config files |
| [Architecture](architecture) | Technical deep-dive into how Lobster works |
| [Limitations](limitations) | Known limitations and workarounds |

## For Contributors

| Guide | Description |
|-------|-------------|
| [Development](development) | Project structure and development workflow |
| [Testing](testing) | Testing strategy and coverage goals |
| [Contributing](contributing) | How to contribute to Lobster |

## Responsible Use

Lobster is a powerful tool. Only test systems you own or have explicit permission to test. See the [Responsible Use](responsible-use) guidelines.

## Version

Current stable release: **v2.0.0**

See the [Changelog](https://github.com/1mb-dev/lobster/blob/main/CHANGELOG.md) for release history and migration guides.

## Links

- [GitHub Repository](https://github.com/1mb-dev/lobster)
- [Issue Tracker](https://github.com/1mb-dev/lobster/issues)
- [Discussions](https://github.com/1mb-dev/lobster/discussions)
