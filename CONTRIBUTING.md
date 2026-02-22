# Contributing to Goclaw

Thank you for your interest in contributing to Goclaw! We welcome contributions from the community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/goclaw.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -am 'Add some feature'`
7. Push to your fork: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

```bash
# Install dependencies
make deps

# Run tests
make test

# Build the project
make build

# Run linters
make lint
```

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Write clear commit messages
- Add tests for new features
- Update documentation as needed

## Reporting Issues

If you find a bug or have a feature request, please create an issue on GitHub with:

- A clear title and description
- Steps to reproduce (for bugs)
- Expected vs actual behavior
- Your environment details (OS, Go version, etc.)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
