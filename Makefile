# Makefile for controller

# Variables
BINARY_NAME=controller
BUILD_DIR=build
MAIN_PATH=main.go
CONFIG_PATH=pkg/common/envs/.env

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(shell git describe --tags --always --dirty) -X main.BuildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

.PHONY: all clean test deps run help

# Default target
all: clean build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build completed: $(BINARY_NAME)"

# Build for different platforms
build-linux:
	@echo "Building for Linux..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)

build-darwin:
	@echo "Building for macOS..."
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

# Build for all platforms
build-all: build-linux build-darwin
	@echo "Build for all platforms completed"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
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

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

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

# Run with custom environment variables
run-env: build
	@echo "Running $(BINARY_NAME) with custom environment..."
	PORT=:8080 LOGGING_LEVEL=debug ./$(BINARY_NAME)

# Check if .env file exists
check-env:
	@if [ ! -f $(CONFIG_PATH) ]; then \
		echo "Warning: $(CONFIG_PATH) not found. Creating default..."; \
		mkdir -p pkg/common/envs; \
		echo "PORT=:8080" > $(CONFIG_PATH); \
		echo "KUBECONFIG=~/.kube/config" >> $(CONFIG_PATH); \
		echo "LOGGING_LEVEL=info" >> $(CONFIG_PATH); \
		echo "Default .env file created at $(CONFIG_PATH)"; \
	else \
		echo "Configuration file found: $(CONFIG_PATH)"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

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
dev: check-env deps fmt lint test build run

# Production build
prod: clean deps test build

# Show help
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  build-linux    - Build for Linux (amd64, arm64)"
	@echo "  build-darwin   - Build for macOS (amd64, arm64)"
	@echo "  build-all      - Build for all platforms"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  deps           - Install dependencies"
	@echo "  run            - Build and run the application"
	@echo "  run-debug      - Run with debug logging"
	@echo "  run-trace      - Run with trace logging"
	@echo "  run-env        - Run with custom environment variables"
	@echo "  check-env      - Check/create .env file"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  install-lint   - Install golangci-lint"
	@echo "  delete-lint    - Delete golangci-lint"
	@echo "  dev            - Development workflow (check-env, deps, fmt, lint, test, build, run)"
	@echo "  prod           - Production build (clean, deps, test, build)"
	@echo "  help           - Show this help message" 