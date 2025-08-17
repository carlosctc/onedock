# Contributing to OneDock

We love your input! We want to make contributing to OneDock as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

### Branch Naming

- `main` - Production-ready code
- `develop` - Development branch for integration
- `feature/feature-name` - Feature branches
- `fix/bug-description` - Bug fix branches
- `hotfix/critical-fix` - Critical production fixes

## Pull Request Process

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code lints
6. Issue that pull request!

### Pull Request Guidelines

- **Fill out the template**: Use the provided PR template
- **Keep it small**: Smaller PRs are easier to review and merge
- **Write descriptive commit messages**: Follow conventional commit format
- **Test your changes**: Ensure all tests pass and add new tests if needed
- **Update documentation**: Update README, API docs, or comments as needed

## Code Style

### Go Code Style

We follow standard Go conventions:

- Use `gofmt` to format your code
- Use `golint` and `go vet` to catch common issues
- Follow effective Go practices
- Write comprehensive tests for new functionality
- Add comments for exported functions and types

### Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

Examples:
```
feat(api): add health check endpoint
fix(docker): resolve container port mapping issue
docs: update API documentation
test(service): add unit tests for scaling functionality
```

## Development Setup

### Prerequisites

- Go 1.24 or higher
- Docker
- Git

### Local Development

1. **Clone and setup**
   ```bash
   git clone https://github.com/aichy126/onedock.git
   cd onedock
   go mod tidy
   ```

2. **Run tests**
   ```bash
   go test ./...
   ```

3. **Run locally**
   ```bash
   ./dev.sh
   # or
   go run main.go
   ```

4. **Generate documentation**
   ```bash
   swag init
   ```

### Code Quality

Before submitting a PR, ensure your code passes:

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate swagger docs
swag init
```

## Testing

- Write unit tests for all new functionality
- Maintain or improve code coverage
- Test both success and error cases
- Use table-driven tests where appropriate
- Mock external dependencies

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./service/

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...
```

## Documentation

- Update README.md for user-facing changes
- Update API documentation in Swagger comments
- Add inline code comments for complex logic
- Update CHANGELOG.md for notable changes

## Reporting Bugs

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/aichy126/onedock/issues/new?template=bug_report.md).

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## Feature Requests

We use GitHub issues to track feature requests. Request a feature by [opening a new issue](https://github.com/aichy126/onedock/issues/new?template=feature_request.md).

**Great Feature Requests** include:

- A clear and concise description of the problem
- A description of the solution you'd like
- Any alternative solutions you've considered
- Additional context or screenshots

## Security Vulnerabilities

Please do not report security vulnerabilities publicly. Instead, send an email to security@onedock.dev with:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes

We will respond as quickly as possible and work with you to resolve the issue.

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Getting Help

- Check existing [GitHub Issues](https://github.com/aichy126/onedock/issues)
- Start a [GitHub Discussion](https://github.com/aichy126/onedock/discussions)
- Read the documentation in [README.md](README.md)

## Recognition

Contributors who make significant contributions will be:

- Added to the CONTRIBUTORS.md file
- Mentioned in release notes
- Invited to join the maintainer team for sustained contributions

Thank you for contributing to OneDock! 🎉