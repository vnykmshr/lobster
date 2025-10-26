# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- CHANGELOG.md to track version history following Keep a Changelog format
- Makefile with comprehensive build targets (build, test, lint, coverage, install, CI)
- Linter configuration (.golangci.yml) with 15+ enabled linters
- EditorConfig (.editorconfig) for consistent code style across editors
- Unit tests for domain package (100% coverage)
- Unit tests for config package (95.2% coverage)
- Unit tests for crawler package (94.9% coverage)
- Unit tests for validator package (51.2% coverage)
- Unit tests for tester package (86.9% coverage)
- Overall test coverage: 30.2% of statements
- Shared test fixtures in internal/testutil package
- 6 comprehensive integration tests for end-to-end workflows
- Skippable slow tests using testing.Short() pattern
- HTTP 429 retry error path coverage tests
- GitHub Actions CI/CD pipeline (pending)
- Version injection support via ldflags in Makefile

### Changed
- Moved global package-level mutexes into Tester struct for better testability and isolation
- Improved testing infrastructure with coverage reporting
- Fixed flaky tests by replacing time.Sleep with proper channel synchronization
- Converted 7 authentication test functions to table-driven approach
- Condensed README.md from 423→216 lines (49% reduction)
- Condensed QUICKSTART.md from 267→130 lines (51% reduction)
- Fixed coverage contradictions and claims in TESTING.md
- Updated coverage numbers in DEVELOPMENT.md
- Improved tester package coverage from 71.2% to 86.9%

### Removed
- Empty pkg/ directory that served no purpose
- Trivial domain entity tests (187 lines providing no value)
- Duplicate test helper functions (consolidated to testutil)
- Redundant documentation sections (use cases, tips, troubleshooting)
- AI-generated content and emojis from documentation

## [0.1.0] - 2025-10-24

### Added
- Initial release after graduating from markgo/examples/stress-test
- CLI interface with comprehensive flags
- Automatic URL discovery through intelligent crawling
- Concurrent load testing with configurable workers
- Token bucket rate limiting via goflow
- HTML report generation with interactive charts
- JSON report output for programmatic analysis
- Console summary with real-time progress monitoring
- Performance validation against configurable targets
- Competitive benchmarking support
- Configuration file support (JSON)
- MIT License
- Comprehensive documentation (README, CONTRIBUTING, DEVELOPMENT)
- Quick start guide and roadmap

### Changed
- Rebranded from WebStress to Lobster
- Updated module path: github.com/vnykmshr/webstress → github.com/vnykmshr/lobster
- Updated user agent: WebStress/1.0 → Lobster/1.0
- Updated all documentation with new branding

[Unreleased]: https://github.com/vnykmshr/lobster/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/vnykmshr/lobster/releases/tag/v0.1.0
