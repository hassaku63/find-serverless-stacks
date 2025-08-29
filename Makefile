# find_serverless_stacks Makefile

# Variables
BINARY_NAME=find_serverless_stacks
MAIN_PACKAGE=./cmd/find_serverless_stacks
GO_VERSION=1.19
VERSION?=dev
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

# Build for multiple platforms
.PHONY: build-cross
build-cross: clean
	@echo "Cross-compiling for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf build/
	go clean

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
.PHONY: test-verbose
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests (requires AWS credentials)
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2"; \
		exit 1; \
	fi

# Vet code
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Update dependencies
.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	go mod tidy
	go mod download

# Show dependencies
.PHONY: deps-list
deps-list:
	@echo "Listing dependencies..."
	go list -m all

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin/..."
	cp $(BINARY_NAME) $(GOPATH)/bin/

# Uninstall the binary
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Development run with sample parameters
.PHONY: run-dev
run-dev: build
	@echo "Running development version..."
	./$(BINARY_NAME) --region us-east-1 --profile default --output json

# Run with help flag
.PHONY: run-help
run-help: build
	./$(BINARY_NAME) --help

# Docker build
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-cross   - Cross-compile for multiple platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run unit tests"
	@echo "  test-verbose  - Run unit tests with verbose output"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-integration - Run integration tests (requires AWS credentials)"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter (requires golangci-lint)"
	@echo "  vet           - Run go vet"
	@echo "  deps-update   - Update and tidy dependencies"
	@echo "  deps-list     - List all dependencies"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  uninstall     - Remove binary from GOPATH/bin"
	@echo "  run-dev       - Run development version with sample parameters"
	@echo "  run-help      - Show application help"
	@echo "  docker-build  - Build Docker image"
	@echo "  help          - Show this help message"