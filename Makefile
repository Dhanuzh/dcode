.PHONY: build install clean test run help

# Build the dcode binary
build:
	@echo "Building dcode..."
	@go build -o dcode ./cmd/dcode
	@echo "Build complete! Binary: ./dcode"

# Install dcode to /usr/local/bin
install: build
	@echo "Installing dcode to /usr/local/bin..."
	@sudo mv dcode /usr/local/bin/
	@echo "Installation complete! Run 'dcode' to start."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f dcode
	@go clean
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run dcode
run: build
	@echo "Running dcode..."
	@./dcode

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies ready!"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete!"

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "Lint complete!"

# Show help
help:
	@echo "DCode Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build    - Build the dcode binary"
	@echo "  install  - Install dcode to /usr/local/bin"
	@echo "  clean    - Clean build artifacts"
	@echo "  test     - Run tests"
	@echo "  run      - Build and run dcode"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  fmt      - Format code"
	@echo "  lint     - Run linter"
	@echo "  help     - Show this help message"
