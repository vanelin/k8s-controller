# Makefile for controller

# Variables
BINARY_NAME=k8s-controller
BUILD_DIR=build
MAIN_PATH=main.go
CONFIG_PATH=pkg/common/envs/.env

APP=$(shell basename $(shell git remote get-url origin) |cut -d '.' -f1)
REGISTRY ?=vanelin
TARGETOS ?=linux
TARGETOSARCH ?=arm64
VERSION ?=$(shell git describe --tags --always --dirty)
SERVER_PORT ?=8080
LOGGING_LEVEL ?=debug

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

.PHONY: all clean test run help server server-debug server-trace format get build build-linux build-darwin build-all

# Default target
all: clean build

# Format code
format:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Get dependencies
get:
	@echo "Getting dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Build the application
build: format get
	@echo "Building $(BINARY_NAME) for $(TARGETOS)/$(TARGETOSARCH)..."
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETOSARCH) $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build completed: $(BINARY_NAME)"

# Build for different platforms
build-linux:
	@echo "Building for Linux..."
	mkdir -p $(BUILD_DIR)
	$(MAKE) build TARGETOS=linux TARGETOSARCH=amd64
	mv $(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	$(MAKE) build TARGETOS=linux TARGETOSARCH=arm64
	mv $(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64

build-darwin:
	@echo "Building for macOS..."
	mkdir -p $(BUILD_DIR)
	$(MAKE) build TARGETOS=darwin TARGETOSARCH=amd64
	mv $(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	$(MAKE) build TARGETOS=darwin TARGETOSARCH=arm64
	mv $(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64

# Build for all platforms
build-all: build-linux build-darwin
	@echo "Build for all platforms completed"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.html coverage.out
	rm -rf $(BUILD_DIR)
	@echo "Clean completed"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Run with debug logging
run-debug: build
	@echo "Running $(BINARY_NAME) with debug logging..."
	./$(BINARY_NAME) --log-level debug

# Run with trace logging
run-trace: build
	@echo "Running $(BINARY_NAME) with trace logging..."
	./$(BINARY_NAME) --log-level trace

# Run server command
server: build
	@echo "Starting FastHTTP server..."
	./$(BINARY_NAME) server

# Run server with debug logging
server-debug: build
	@echo "Starting FastHTTP server with debug logging..."
	./$(BINARY_NAME) server --log-level debug

# Run server with trace logging
server-trace: build
	@echo "Starting FastHTTP server with trace logging..."
	./$(BINARY_NAME) server --log-level trace

# Run server on custom port
server-port: build
	@echo "Starting FastHTTP server on custom port..."
	@read -p "Enter port number: " port; \
	./$(BINARY_NAME) server --port $$port

# Run server with custom environment variables
server-env: build
	@echo "Starting FastHTTP server with custom environment..."
	@read -p "Enter port number (default: $(SERVER_PORT)): " port; \
	read -p "Enter log level (debug/info/warn/error/trace, default: $(LOGGING_LEVEL)): " loglevel; \
	PORT=$${port:-$(SERVER_PORT)} LOGGING_LEVEL=$${loglevel:-$(LOGGING_LEVEL)} ./$(BINARY_NAME) server

# Check if .env file exists
check-env:
	@if [ ! -f $(CONFIG_PATH) ] || [ ! -s $(CONFIG_PATH) ]; then \
		echo "Warning: $(CONFIG_PATH) not found or empty. Creating default..."; \
		mkdir -p pkg/common/envs; \
		echo "PORT=$(SERVER_PORT)" > $(CONFIG_PATH); \
		echo "KUBECONFIG=~/.kube/config" >> $(CONFIG_PATH); \
		echo "LOGGING_LEVEL=$(LOGGING_LEVEL)" >> $(CONFIG_PATH); \
		echo "Default .env file created at $(CONFIG_PATH)"; \
	else \
		echo "Configuration file found: $(CONFIG_PATH)"; \
	fi

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Install golangci-lint if not present
install-lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.1.6; \
		echo "Please add $$(go env GOPATH)/bin to your PATH if not already done"; \
	else \
		echo "golangci-lint already installed"; \
	fi

# Uninstall golangci-lint
delete-lint:
	@if command -v golangci-lint &> /dev/null; then \
		echo "Uninstalling golangci-lint..."; \
		rm -f $$(go env GOPATH)/bin/golangci-lint; \
		echo "golangci-lint uninstalled"; \
	fi

# Development workflow
dev: check-env get format lint test build run

# Server development workflow
dev-server: check-env get format lint test server

# Production build
prod: clean get test build

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Build commands:"
	@echo "  build          - Build the application (use TARGETOS/TARGETOSARCH for cross-compilation)"
	@echo "  build-linux    - Build for Linux (amd64, arm64)"
	@echo "  build-darwin   - Build for macOS (amd64, arm64)"
	@echo "  build-all      - Build for all platforms"
	@echo "  clean          - Clean build artifacts"
	@echo ""
	@echo "Cross-compilation examples:"
	@echo "  make build TARGETOS=linux TARGETOSARCH=arm64"
	@echo "  make build TARGETOS=darwin TARGETOSARCH=arm64"
	@echo ""
	@echo "Test commands:"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo ""
	@echo "Dependency commands:"
	@echo "  get            - Get dependencies"
	@echo "  format         - Format code"
	@echo ""
	@echo "Run commands:"
	@echo "  run            - Build and run the application"
	@echo "  run-debug      - Run with debug logging"
	@echo "  run-trace      - Run with trace logging"
	@echo ""
	@echo "Server commands:"
	@echo "  server         - Build and start FastHTTP server"
	@echo "  server-debug   - Start server with debug logging"
	@echo "  server-trace   - Start server with trace logging"
	@echo "  server-port    - Start server on custom port (interactive)"
	@echo "  server-env     - Start server with custom environment"
	@echo ""
	@echo "Development commands:"
	@echo "  check-env      - Check/create .env file"
	@echo "  lint           - Lint code"
	@echo "  install-lint   - Install golangci-lint"
	@echo "  delete-lint    - Delete golangci-lint"
	@echo "  dev            - Development workflow (check-env, get, format, lint, test, build, run)"
	@echo "  dev-server     - Server development workflow (check-env, get, format, lint, test, server)"
	@echo "  prod           - Production build (clean, get, test, build)"
	@echo ""
	@echo "Help:"
	@echo "  help           - Show this help message" 