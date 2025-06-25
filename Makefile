# Makefile for controller

# Variables
BINARY_NAME=k8s-controller
BUILD_DIR=build
MAIN_PATH=main.go
CONFIG_PATH=pkg/common/envs/.env

APP=$(shell basename $(shell git remote get-url origin) |cut -d '.' -f1)
REGISTRY ?=ghcr.io
REPOSITORY ?=vanelin
TARGETOS ?=linux
TARGETARCH ?=arm64
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
BUILD_FLAGS = -v -o $(APP) -ldflags "-X github.com/vanelin/$(APP).git/cmd.appVersion=$(VERSION) -X github.com/vanelin/$(APP).git/cmd.buildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')"

.PHONY: all clean test run help server server-debug server-trace format get build build-linux docker-build docker-build-multi docker-run docker-clean clean-all push

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
	@echo "Building $(BINARY_NAME) for $(TARGETOS)/$(TARGETARCH)..."
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) $(GOBUILD) $(BUILD_FLAGS) $(MAIN_PATH)
	@echo "Build completed: $(BINARY_NAME)"

# Build for different platforms
build-linux:
	@echo "Building for Linux..."
	mkdir -p $(BUILD_DIR)
	$(MAKE) build TARGETOS=linux TARGETARCH=amd64
	mv $(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	$(MAKE) build TARGETOS=linux TARGETARCH=arm64
	mv $(BINARY_NAME) $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.html coverage.out
	rm -rf $(BUILD_DIR)
	@echo "Clean completed"

# Clean Docker images
docker-clean:
	@echo "Cleaning Docker images..."
	@docker images $(REGISTRY)/$(REPOSITORY)/$(APP) --format "table {{.Repository}}:{{.Tag}}" | grep -v "REPOSITORY:TAG" | xargs -r docker rmi || echo "No Docker images found to remove"

# Clean everything (build artifacts + Docker images)
clean-all: clean docker-clean
	@echo "All clean completed"

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

# Docker build for single architecture
docker-build:
	@echo "Building Docker image for $(TARGETOS)/$(TARGETARCH)..."
	docker buildx build \
		--platform $(TARGETOS)/$(TARGETARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		--build-arg SERVER_PORT=$(SERVER_PORT) \
		--build-arg LOGGING_LEVEL=$(LOGGING_LEVEL) \
		--load \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):$(VERSION)-$(TARGETOS)-$(TARGETARCH) \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):latest-$(TARGETOS)-$(TARGETARCH) \
		.
	@echo "Docker image built: $(REGISTRY)/$(REPOSITORY)/$(APP):$(VERSION)-$(TARGETOS)-$(TARGETARCH)"

# Multi-arch Docker build (creates manifest automatically)
docker-build-multi:
	@echo "Building multi-arch Docker image..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg SERVER_PORT=$(SERVER_PORT) \
		--build-arg LOGGING_LEVEL=$(LOGGING_LEVEL) \
		--push \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):$(VERSION) \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):latest \
		.
	@echo "Multi-arch Docker image built and pushed: $(REGISTRY)/$(REPOSITORY)/$(APP):$(VERSION)"

# Push Docker image
push:
	@echo "Pushing Docker image..."
	docker push $(REGISTRY)/$(REPOSITORY)/$(APP):$(VERSION)-$(TARGETOS)-$(TARGETARCH)
	docker push $(REGISTRY)/$(REPOSITORY)/$(APP):latest-$(TARGETOS)-$(TARGETARCH)

# Docker run
docker-run: docker-build
	@echo "Running Docker container interactively (external:internal ports)..."
	@echo -n "Enter FastHTTP server port (default: $(SERVER_PORT)): "; read int_port; \
	echo -n "Enter external port (host port, default: $(SERVER_PORT)): "; read ext_port; \
	echo -n "Enter log level (debug/info/warn/error/trace, default: $(LOGGING_LEVEL)): "; read loglevel; \
	echo "Starting container with external port: $${ext_port:-$(SERVER_PORT)}, internal port: $${int_port:-$(SERVER_PORT)}, LOGGING_LEVEL: $${loglevel:-$(LOGGING_LEVEL)}..."; \
	docker run --rm -p $${ext_port:-$(SERVER_PORT)}:$${int_port:-$(SERVER_PORT)} \
		-e PORT=$${int_port:-$(SERVER_PORT)} \
		-e LOGGING_LEVEL=$${loglevel:-$(LOGGING_LEVEL)} \
		$(REGISTRY)/$(REPOSITORY)/$(APP):latest-$(TARGETOS)-$(TARGETARCH) server

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Build commands:"
	@echo "  build          - Build the application (use TARGETOS/TARGETARCH for cross-compilation)"
	@echo "  build-linux    - Build for Linux (amd64, arm64)"
	@echo "  clean          - Clean build artifacts"
	@echo ""
	@echo "Cross-compilation example:"
	@echo "  make build TARGETOS=linux TARGETARCH=arm64"
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
	@echo "Docker commands:"
	@echo "  docker-build   - Build single-arch Docker image"
	@echo "  docker-build-multi - Build and push multi-arch Docker image (CI/CD)"
	@echo "  docker-run     - Build and run single-arch Docker container"
	@echo "  docker-clean   - Clean Docker images"
	@echo "  clean-all      - Clean build artifacts and Docker images"
	@echo "  push           - Push single-arch Docker image"
	@echo ""
	@echo "Help:"
	@echo "  help           - Show this help message" 