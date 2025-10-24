# Contributing to WebStress

Thank you for your interest in contributing to WebStress! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and professional in all interactions. We're building a welcoming community for everyone.

## How to Contribute

### Reporting Bugs

1. Check existing issues to avoid duplicates
2. Create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)
   - Relevant logs or error messages

### Suggesting Features

1. Check existing feature requests
2. Create a new issue with:
   - Clear use case
   - Expected behavior
   - Why this benefits users
   - Potential implementation approach (optional)

### Submitting Code

1. **Fork the repository**
   ```bash
   git clone https://github.com/vnykmshr/webstress.git
   cd webstress
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow the existing code style
   - Add tests for new functionality
   - Update documentation as needed
   - Keep commits atomic and well-described

4. **Run tests**
   ```bash
   go test ./...
   go vet ./...
   ```

5. **Commit your changes**
   ```bash
   git commit -m "Add feature: description"
   ```

6. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Create a Pull Request**
   - Provide a clear description
   - Link related issues
   - Ensure CI passes

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `go vet` before committing
- Keep functions focused and concise
- Add comments for exported functions

### Testing

- Write unit tests for new functions
- Maintain test coverage >70%
- Test edge cases and error conditions
- Use table-driven tests where appropriate

### Architecture

WebStress follows Clean Architecture:

```
internal/
├── domain/     # Core entities and business logic
├── crawler/    # URL discovery
├── tester/     # Load testing engine
├── reporter/   # Report generation
├── validator/  # Performance validation
└── config/     # Configuration management
```

- Keep domain logic independent
- Use interfaces for dependencies
- Avoid circular dependencies

### Commit Messages

Follow conventional commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `chore`: Build/tooling changes

Examples:
```
feat(crawler): add support for JavaScript-rendered pages
fix(tester): correct percentile calculation
docs(readme): update installation instructions
```

### Pull Request Guidelines

- Keep PRs focused on a single feature/fix
- Update relevant documentation
- Add tests for new functionality
- Ensure all tests pass
- Address review feedback promptly
- Squash commits before merging (if requested)

## Project Structure

```
webstress/
├── cmd/webstress/          # CLI entry point
├── internal/
│   ├── domain/            # Core business entities
│   ├── crawler/           # URL discovery
│   ├── tester/            # Stress testing engine
│   ├── reporter/          # Report generation
│   ├── validator/         # Performance validation
│   └── config/            # Configuration
├── docs/                  # Documentation
├── examples/              # Example configurations
└── README.md
```

## Getting Help

- 💬 [GitHub Discussions](https://github.com/vnykmshr/webstress/discussions)
- 🐛 [Issue Tracker](https://github.com/vnykmshr/webstress/issues)
- 📖 [Documentation](docs/)

## Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes for significant contributions
- GitHub contributors page

Thank you for making WebStress better! 🚀
