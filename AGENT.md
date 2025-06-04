# Terminal WakaTime Agent Guide

## Build/Test Commands
Run `make help` to see all available targets. Key commands:

**Primary Commands:**
- `make build` - Build the main binary
- `make test` - Run full test suite (unit + coverage)
- `make test-integration` - Run integration tests
- `make test-shell-integration` - Run shell integration tests
- `make check` - Quick dev check (fmt + vet + short tests)
- `make clean` - Clean build artifacts

**Direct Go Commands (for specific needs):**
- `go test ./pkg/tracker -v` - Run tests for specific package with verbose output
- `go test -run TestName` - Run single test by name
- `go test -tags=integration ./tests/` - Run integration tests only
- `go test -short ./...` - Run only fast unit tests (skip integration)
- `go test -race ./...` - Run tests with race detection

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
