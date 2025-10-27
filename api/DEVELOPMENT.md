# Hermes API - Development Guide

## Code Quality Tools

This project uses several tools to maintain code quality and catch issues early.

### Quick Start

Run all quality checks before submitting code for review:

```bash
cd api
make check
```

This command runs:
- Code formatting checks (`gofmt`)
- Static analysis (`go vet`)
- Linting (`golangci-lint`)
- All tests

### Individual Commands

#### Format Code

Auto-format all Go files:
```bash
make fmt
```

#### Run Tests

```bash
make test          # Run all tests
make test-race     # Run tests with race detector
make coverage      # Run tests with coverage report
```

#### Linting

```bash
make lint          # Run golangci-lint
make vet           # Run go vet
```

#### Run Server

```bash
make run
```

### Linter Configuration

The project uses `golangci-lint` with a custom configuration (`.golangci.yml`) that includes:

**Critical Linters:**
- `errcheck` - Checks for unchecked errors
- `govet` - Official Go static analysis
- `staticcheck` - Advanced static analysis
- `gosec` - Security-focused checks

**Code Quality:**
- `gofmt` - Code formatting
- `goimports` - Import organization
- `errorlint` - Proper error wrapping
- `goconst` - Repeated strings that should be constants
- `gocyclo` - Function complexity (max 15)

**Style (informational):**
- `gocritic` - Style suggestions
- `revive` - Additional style checks

### Pre-Review Checklist

Before submitting code for review, ensure:

1. ✅ All tests pass: `make test`
2. ✅ No race conditions: `make test-race`
3. ✅ Code is formatted: `make fmt`
4. ✅ No vet issues: `make vet`
5. ✅ No linter errors: `make lint`
6. ✅ Test coverage >80% for new code: `make coverage`

Or simply run: `make check` to verify all at once.

### Installing Tools

If `golangci-lint` is not installed:

```bash
make install-tools
```

Or install manually:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Continuous Integration

These tools should also be run in CI/CD pipelines to ensure code quality is maintained across the team.

### Handling Linter Issues

#### False Positives

If a linter reports a false positive, you can:

1. **Add an inline comment** explaining why the issue is acceptable:
   ```go
   //nolint:errcheck // Deliberately ignoring error here because...
   _ = file.Close()
   ```

2. **Update `.golangci.yml`** if the rule is too strict for the project

#### Common Issues

**Unchecked errors (`errcheck`):**
- Always check errors or explicitly document why you're ignoring them
- Use `_ = func()` with a comment if intentionally ignoring

**Error wrapping (`errorlint`):**
- Use `errors.Is()` instead of `==` for error comparison
- Use `errors.As()` instead of type assertions for errors
- Use `%w` in `fmt.Errorf()` to wrap errors

**High complexity (`gocyclo`):**
- Break complex functions into smaller, more focused functions
- Extract nested logic into helper functions

**Type assertions on errors:**
```go
// Bad
if _, ok := err.(*SomeError); ok {
    // ...
}

// Good
var someErr *SomeError
if errors.As(err, &someErr) {
    // ...
}
```

## Testing

### Test Structure

- **Unit tests**: Test individual functions in isolation
- **Integration tests**: Test component interactions
- **Coverage target**: >80% for new code

### Running Specific Tests

```bash
# Run tests for a specific package
go test ./internal/media/...

# Run a specific test
go test -run TestValidateMedia ./internal/media/...

# Verbose output
go test -v ./...
```

### Test Coverage

```bash
# View coverage by package
make coverage

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Development Workflow

1. **Start working on a task**
   - Read task documentation
   - Understand requirements
   - Review relevant API specs

2. **Write code**
   - Follow Go best practices
   - Write tests alongside implementation
   - Run `make fmt` regularly

3. **Before committing**
   - Run `make check`
   - Fix all linter errors
   - Ensure tests pass
   - Update documentation

4. **Submit for review**
   - All checks must pass
   - Include test coverage report
   - Document any disabled linter rules

## Troubleshooting

### Linter is too slow

```bash
# Run only fast linters
golangci-lint run --fast
```

### Clear linter cache

```bash
golangci-lint cache clean
```

### Update linters

```bash
golangci-lint linters
```

## Resources

- [golangci-lint documentation](https://golangci-lint.run/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

