.PHONY: build install clean test run help release snapshot fmt lint deps cross

# Version info injected at build time
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS   := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

BINARY    := dcode
CMD       := ./cmd/dcode

# Build the dcode binary for the current platform
build:
	@echo "Building $(BINARY) $(VERSION) ($(COMMIT))..."
	@go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(CMD)
	@echo "Build complete: ./$(BINARY)"

# Install dcode to /usr/local/bin
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	@sudo mv $(BINARY) /usr/local/bin/
	@echo "Installation complete! Run 'dcode' to start."

# Cross-compile for all release targets
cross:
	@echo "Cross-compiling for all platforms..."
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)_linux_x86_64   $(CMD)
	GOOS=linux   GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)_linux_arm64    $(CMD)
	GOOS=darwin  GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)_darwin_x86_64  $(CMD)
	GOOS=darwin  GOARCH=arm64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)_darwin_arm64   $(CMD)
	GOOS=windows GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY)_windows_x86_64.exe $(CMD)
	@echo "Cross-compile complete. Binaries in ./dist/"

# Create a local snapshot release (requires goreleaser)
snapshot:
	@echo "Building snapshot release..."
	@goreleaser release --snapshot --clean

# Create a full release (requires goreleaser + GITHUB_TOKEN)
release:
	@echo "Creating release $(VERSION)..."
	@goreleaser release --clean

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY)
	@rm -rf dist/
	@go clean
	@echo "Clean complete!"

# Run tests with race detector
test:
	@echo "Running tests..."
	@go test -race -count=1 -v ./...

# Run tests with coverage report
cover:
	@echo "Running tests with coverage..."
	@go test -race -count=1 -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run dcode
run: build
	@./$(BINARY)

# Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies ready!"

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@echo "Format complete!"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "Lint complete!"

# Show help
help:
	@echo "dcode Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build     - Build the dcode binary for the current platform"
	@echo "  install   - Install dcode to /usr/local/bin"
	@echo "  cross     - Cross-compile for Linux, macOS, and Windows"
	@echo "  snapshot  - Build a local snapshot release (goreleaser)"
	@echo "  release   - Publish a full release via goreleaser"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run tests with race detector"
	@echo "  cover     - Run tests and generate HTML coverage report"
	@echo "  run       - Build and run dcode"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  fmt       - Format Go source code"
	@echo "  lint      - Run golangci-lint"
	@echo "  help      - Show this help message"
