# Terminal WakaTime Makefile

.PHONY: all build test test-unit test-integration test-coverage clean fmt vet lint deps install

# Variables
BINARY_NAME=terminal-wakatime
CMD_DIR=./cmd/terminal-wakatime
PKG_DIR=./pkg/...
MAIN_PKG=./cmd/terminal-wakatime

# Build variables
VERSION ?= $(shell git describe --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -X github.com/hackclub/terminal-wakatime/pkg/config.PluginVersion=$(VERSION)

# Default target
all: fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PKG)

# Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	go install -ldflags "$(LDFLAGS)" $(MAIN_PKG)

# Run all tests
test: test-unit test-coverage

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -v $(PKG_DIR)
	go test -v ./tests/...

# Run integration tests (requires built binary and real shell environments)
test-integration:
	@echo "Running integration tests..."
	go test -tags=integration -v ./tests/

# Run shell integration tests (tests full shell lifecycle)
test-shell-integration:
	@echo "Running shell integration tests..."
	go test -v ./tests/ -run TestShell

# Run all integration tests
test-full-integration: test-integration test-shell-integration
	@echo "All integration tests completed"

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover -coverprofile=coverage.out $(PKG_DIR)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race $(PKG_DIR)

# Run short tests only (skip network-dependent tests)
test-short:
	@echo "Running short tests..."
	go test -short $(PKG_DIR)

# Format code
fmt:
	@echo "Formatting code..."
	go fmt $(PKG_DIR)
	go fmt $(MAIN_PKG)

# Vet code
vet:
	@echo "Vetting code..."
	go vet $(PKG_DIR)
	go vet $(MAIN_PKG)

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	go clean -cache
	go clean -testcache

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 $(MAIN_PKG)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 $(MAIN_PKG)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PKG)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PKG)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PKG)

# Development commands
dev-setup:
	@echo "Setting up development environment..."
	go mod download
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

# Run a quick development check
check: fmt vet test-short

# Test with mocked wakatime-cli (creates mock binaries for testing)
test-mocked:
	@echo "Creating mock wakatime-cli for testing..."
	@mkdir -p ./testbin
	@echo '#!/bin/bash\necho "wakatime-cli v1.73.0 (mock)"\nexit 0' > ./testbin/wakatime-cli
	@chmod +x ./testbin/wakatime-cli
	@PATH="$(PWD)/testbin:$$PATH" go test -v $(PKG_DIR)
	@rm -rf ./testbin

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not found. Install it with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem $(PKG_DIR)

# Profile tests
profile:
	@echo "Running tests with CPU profiling..."
	go test -cpuprofile=cpu.prof -memprofile=mem.prof $(PKG_DIR)
	@echo "Profiles generated: cpu.prof, mem.prof"
	@echo "View with: go tool pprof cpu.prof"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  install       - Install the binary"
	@echo "  test          - Run all tests"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-shell-integration - Run shell integration tests"  
	@echo "  test-full-integration - Run all integration tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-short    - Run short tests (skip network tests)"
	@echo "  test-mocked   - Run tests with mocked wakatime-cli"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
	@echo "  lint          - Run linter"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  dev-setup     - Set up development environment"
	@echo "  check         - Quick development check (fmt, vet, test-short)"
	@echo "  docs          - Start godoc server"
	@echo "  bench         - Run benchmarks"
	@echo "  profile       - Run tests with profiling"
	@echo "  version       - Show version information"
	@echo "  help          - Show this help"

version:
	@echo "Current version: $(VERSION)"
	@if [ -f "$(BINARY_NAME)" ]; then ./$(BINARY_NAME) version; fi
