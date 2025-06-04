# Terminal WakaTime Agent Guide

## Build/Test Commands
- `go build -o terminal-wakatime ./cmd/terminal-wakatime` - Build the main binary
- `go test ./...` - Run all tests  
- `go test ./pkg/tracker -v` - Run tests for specific package with verbose output
- `go test -run TestName` - Run single test by name
- `go test -race ./...` - Run tests with race detection
- `go test -cover ./...` - Run tests with coverage
- `go test -tags=integration ./...` - Run integration tests only
- `go test -short ./...` - Run only fast unit tests (skip integration)
- `make test` - Run full test suite including mocked wakatime-cli tests
- `make test-coverage` - Generate coverage report
- `make test-integration` - Run integration tests with real shell environments
- `go mod tidy` - Clean up dependencies
- `go fmt ./...` - Format all code
- `go vet ./...` - Run static analysis
- `golangci-lint run` - Run linter (if available)

## Code Style Guidelines
- Use `gofmt` for formatting - no tabs vs spaces debates
- Package names: lowercase, single word, no underscores (e.g., `tracker`, `config`, `shell`)
- File names: lowercase with underscores for multi-word (e.g., `shell_monitor.go`)
- Types: PascalCase for exported, camelCase for private
- Functions: PascalCase for exported, camelCase for private  
- Variables: camelCase, descriptive names (avoid single letters except loop counters)
- Constants: ALL_CAPS with underscores for exported, camelCase for private
- Imports: stdlib first, third-party second, local last, separated by blank lines
- Error handling: explicit checks, wrap with `fmt.Errorf("operation failed: %w", err)`
- Context: pass `context.Context` as first parameter to functions that need it
- Interfaces: small, focused, end with 'er' suffix when possible

## Git Best Practices
- **Multiple agents warning**: Multiple agents may be working on this repository simultaneously
- **Always use specific file paths** with `git add` instead of `git add .` or `git add -A`
- Example: `git add pkg/tracker/tracker.go .github/workflows/test.yml` instead of `git add .`
- This prevents accidentally committing changes made by other agents or processes
- Check `git status` before committing to ensure only intended files are staged
