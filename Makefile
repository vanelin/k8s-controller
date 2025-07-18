# Makefile for controller

# Variables
BINARY_NAME=k8s-controller
BUILD_DIR=build
MAIN_PATH=main.go

APP=$(shell basename $(shell git remote get-url origin) |cut -d '.' -f1)
REGISTRY ?=ghcr.io
REPOSITORY ?=vanelin
TARGETOS ?=linux

# Viper envs
CONFIG_PATH=pkg/common/envs/.env
SERVER_PORT ?=8080
METRIC_PORT ?=8081
LOGGING_LEVEL ?=debug
KUBECONFIG ?=~/.kube/config
IN_CLUSTER ?=false
NAMESPACE ?=default
ENABLE_LEADER_ELECTION ?=true
LEADER_ELECTION_NAMESPACE ?=default

# Detect system architecture
ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
    TARGETARCH ?= amd64
else ifeq ($(ARCH),aarch64)
    TARGETARCH ?= arm64
else
    TARGETARCH ?= amd64
endif

# envtest
ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_VERSION ?= latest
LOCALBIN ?= $(shell pwd)/bin
USE_EXISTING_CLUSTER ?= false

# Version calculation (matching CI workflow logic)
LATEST_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")
SHORT_SHA := $(shell git rev-parse --short HEAD)
RAW_TAG := $(shell git describe --tags --exact-match 2>/dev/null || echo "")
LATEST_TAG_CLEAN := $(subst v,,$(LATEST_TAG))

# Version logic matching CI workflow
ifeq ($(RAW_TAG),)
    # For commits, use latest tag + short SHA (remove "v" prefix from tag)
    VERSION ?=$(LATEST_TAG_CLEAN)-$(SHORT_SHA)
    APP_VERSION ?=$(LATEST_TAG)-$(SHORT_SHA)
    DOCKER_TAG ?=$(VERSION)
else
    # If this is a tag, use the tag as version (remove "v" prefix)
    VERSION ?=$(subst v,,$(RAW_TAG))
    APP_VERSION ?=$(RAW_TAG)
    DOCKER_TAG ?=$(VERSION)
endif

# Go related variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
ENVTEST_VERSION ?= release-0.19

envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))
	@echo "Detected architecture: $(ARCH) -> Using envtest arch: $(TARGETARCH)"
	@echo "Installing envtest binaries for $(TARGETARCH)..."
	@$(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path > /dev/null 2>&1 || \
		$(ENVTEST) use --bin-dir $(LOCALBIN) -p path > /dev/null 2>&1
	@chmod +x $(LOCALBIN)/k8s/*/etcd $(LOCALBIN)/k8s/*/kube-apiserver $(LOCALBIN)/k8s/*/kubectl 2>/dev/null || true


# Build flags
BUILD_FLAGS = -v -o $(APP) -ldflags "-X github.com/vanelin/$(APP)/cmd.appVersion=$(APP_VERSION)"

.PHONY: all build build-linux clean test test-coverage test-informer test-ctrl test-config format fmt get lint server list list-namespace check-env dev-server dev prod docker-build docker-build-multi docker-clean clean-all push help vulncheck version-info envtest

# Default target
all: clean build

# Format code
format:
	@echo "Formatting code..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w ./; \
	else \
		echo "goimports not found, installing..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
		goimports -w ./; \
	fi

# Alias for format
fmt: format

# Get dependencies
get:
	@echo "Getting dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	$(GOMOD) verify

# Build the application
build: format get lint
	@echo "Building $(BINARY_NAME) for $(TARGETOS)/$(TARGETARCH) with version $(VERSION)..."
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) $(GOBUILD) $(BUILD_FLAGS) $(MAIN_PATH)
	@echo "Build completed: $(BINARY_NAME)"

# Build for different platforms
build-linux:
	@echo "Building for Linux with version $(VERSION)..."
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
	rm -f coverage.html coverage.out report.xml coverage.xml
	rm -rf $(BUILD_DIR)
	@echo "Clean completed"

# Clean Docker images
docker-clean:
	@echo "Cleaning Docker images..."
	@docker images $(REGISTRY)/$(REPOSITORY)/$(APP) --format "table {{.Repository}}:{{.Tag}}" | grep -v "REPOSITORY:TAG" | xargs -r docker rmi || echo "No Docker images found to remove"

# Clean everything (build artifacts + Docker images)
clean-all: clean docker-clean
	@echo "All clean completed"

# Run all tests (unit + Kubernetes integration tests)
test: envtest
	@echo "Running all tests with envtest..."
	@echo "Using KUBEBUILDER_ASSETS: $(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)"
	go install gotest.tools/gotestsum@latest
	USE_EXISTING_CLUSTER=$(USE_EXISTING_CLUSTER) KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)" gotestsum --junitfile report.xml --format testname ./... ${TEST_ARGS}

# Test Deployment informer with envtest
test-informer: envtest
	@echo "Testing Deployment informer with envtest..."
	@echo "Using KUBEBUILDER_ASSETS: $(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)"
	USE_EXISTING_CLUSTER=$(USE_EXISTING_CLUSTER) KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)" go test ./pkg/informer -v

# Test Deployment controller with envtest
test-ctrl: envtest
	@echo "Testing Deployment controller with envtest..."
	@echo "Using KUBEBUILDER_ASSETS: $(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)"
	USE_EXISTING_CLUSTER=$(USE_EXISTING_CLUSTER) KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)" go test ./pkg/ctrl -v

# Test config with envtest
test-config: envtest
	@echo "Testing config with envtest..."
	@echo "Using KUBEBUILDER_ASSETS: $(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)"
	USE_EXISTING_CLUSTER=$(USE_EXISTING_CLUSTER) KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)" go test ./pkg/common/config -v

# Run all tests with coverage
test-coverage: envtest
	@echo "Running all tests with coverage..."
	@echo "Using KUBEBUILDER_ASSETS: $(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)"
	go install github.com/boumenot/gocover-cobertura@latest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use --arch $(TARGETARCH) --bin-dir $(LOCALBIN) -p path)" go test -coverprofile=coverage.out -covermode=count ./...
	go tool cover -func=coverage.out
	gocover-cobertura < coverage.out > coverage.xml

# Run server command with Deployment controller and informer and leader election enabled
server: build
	@echo "Starting FastHTTP server with Deployment informer..."
	./$(BINARY_NAME) server \
		--kubeconfig $(KUBECONFIG) \
		--namespace $(NAMESPACE) \
		--enable-leader-election $(ENABLE_LEADER_ELECTION) \
		--leader-election-namespace $(LEADER_ELECTION_NAMESPACE) \
		--metric-port $(METRIC_PORT) \
		--port $(SERVER_PORT) \
		--log-level $(LOGGING_LEVEL)

# Run list command
list: build
	@echo "Listing Kubernetes deployments..."
	./$(BINARY_NAME) list

# Run list with custom namespace
list-namespace: build
	@echo "Listing Kubernetes deployments in custom namespace..."
	@read -p "Enter namespace (default: default): " namespace; \
	./$(BINARY_NAME) list --namespace $${namespace:-default}

# Check if .env file exists and create if missing
check-env:
	@if [ ! -f $(CONFIG_PATH) ] || [ ! -s $(CONFIG_PATH) ]; then \
		echo "Warning: $(CONFIG_PATH) not found or empty. Creating default..."; \
		mkdir -p pkg/common/envs; \
		echo "# Server Configuration" > $(CONFIG_PATH); \
		echo "PORT=$(SERVER_PORT)" >> $(CONFIG_PATH); \
		echo "LOGGING_LEVEL=$(LOGGING_LEVEL)" >> $(CONFIG_PATH); \
		echo "METRIC_PORT=$(METRIC_PORT)" >> $(CONFIG_PATH); \
		echo "ENABLE_LEADER_ELECTION=true" >> $(CONFIG_PATH); \
		echo "LEADER_ELECTION_NAMESPACE=$(LEADER_ELECTION_NAMESPACE)" >> $(CONFIG_PATH); \
		echo "" >> $(CONFIG_PATH); \
		echo "# Kubernetes Configuration" >> $(CONFIG_PATH); \
		echo "NAMESPACE=$(NAMESPACE)" >> $(CONFIG_PATH); \
		echo "KUBECONFIG=$(KUBECONFIG)" >> $(CONFIG_PATH); \
		echo "IN_CLUSTER=$(IN_CLUSTER)" >> $(CONFIG_PATH); \
		echo "Default .env file created at $(CONFIG_PATH)"; \
	else \
		echo "Configuration file found: $(CONFIG_PATH)"; \
	fi

# Lint code
lint:
	@if ! command -v golangci-lint &> /dev/null && [ ! -f "$$(go env GOPATH)/bin/golangci-lint" ]; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.2.1; \
		echo "Please add $$(go env GOPATH)/bin to your PATH if not already done"; \
		echo "You can run: export PATH=$$PATH:$$(go env GOPATH)/bin"; \
	fi
	@echo "Linting code..."
	golangci-lint run

# Check for vulnerabilities
vulncheck:
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		echo "govulncheck found, running vulnerability scan..."; \
		govulncheck ./... || true; \
	else \
		echo "govulncheck not installed. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		echo "govulncheck installed, running vulnerability scan..."; \
		govulncheck ./... || true; \
	fi

# Development workflow
dev: check-env get format lint test vulncheck build

# Server development workflow
dev-server: check-env get format lint test vulncheck server

# Release build
prod: clean get test build

# Docker build for single architecture
docker-build:
	@echo "Building Docker image for $(TARGETOS)/$(TARGETARCH) with version $(DOCKER_TAG)..."
	docker buildx build \
		--platform $(TARGETOS)/$(TARGETARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg TARGETOS=$(TARGETOS) \
		--build-arg TARGETARCH=$(TARGETARCH) \
		--build-arg SERVER_PORT=$(SERVER_PORT) \
		--build-arg LOGGING_LEVEL=$(LOGGING_LEVEL) \
		--build-arg NAMESPACE=$(NAMESPACE) \
		--build-arg KUBECONFIG=$(KUBECONFIG) \
		--build-arg IN_CLUSTER=$(IN_CLUSTER) \
		--build-arg METRIC_PORT=$(METRIC_PORT) \
		--build-arg ENABLE_LEADER_ELECTION=$(ENABLE_LEADER_ELECTION) \
		--build-arg LEADER_ELECTION_NAMESPACE=$(LEADER_ELECTION_NAMESPACE) \
		--load \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):$(DOCKER_TAG)-$(TARGETOS)-$(TARGETARCH) \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):latest-$(TARGETOS)-$(TARGETARCH) \
		.
	@echo "Docker image built: $(REGISTRY)/$(REPOSITORY)/$(APP):$(DOCKER_TAG)-$(TARGETOS)-$(TARGETARCH)"

# Multi-arch Docker build (creates manifest automatically)
docker-build-multi:
	@echo "Building multi-arch Docker image with version $(DOCKER_TAG)..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg SERVER_PORT=$(SERVER_PORT) \
		--build-arg LOGGING_LEVEL=$(LOGGING_LEVEL) \
		--build-arg NAMESPACE=$(NAMESPACE) \
		--build-arg KUBECONFIG=$(KUBECONFIG) \
		--build-arg IN_CLUSTER=$(IN_CLUSTER) \
		--build-arg METRIC_PORT=$(METRIC_PORT) \
		--build-arg ENABLE_LEADER_ELECTION=$(ENABLE_LEADER_ELECTION) \
		--build-arg LEADER_ELECTION_NAMESPACE=$(LEADER_ELECTION_NAMESPACE) \
		--provenance=false \
		--push \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):$(DOCKER_TAG) \
		-t $(REGISTRY)/$(REPOSITORY)/$(APP):latest \
		.
	@echo "Multi-arch Docker image built and pushed: $(REGISTRY)/$(REPOSITORY)/$(APP):$(DOCKER_TAG)"

# Push Docker image
push:
	@echo "Pushing Docker image..."
	docker push $(REGISTRY)/$(REPOSITORY)/$(APP):$(DOCKER_TAG)-$(TARGETOS)-$(TARGETARCH)
	docker push $(REGISTRY)/$(REPOSITORY)/$(APP):latest-$(TARGETOS)-$(TARGETARCH)

# Show version info
version-info:
	@echo "Version Information:"
	@echo "  Latest Tag: $(LATEST_TAG)"
	@echo "  Short SHA: $(SHORT_SHA)"
	@echo "  Version: $(VERSION)"
	@echo "  App Version: $(APP_VERSION)"
	@echo "  Docker Tag: $(DOCKER_TAG)"
	@echo "  System Architecture: $(ARCH)"
	@echo "  Target Architecture: $(TARGETARCH)"

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

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
	@echo "  test           - Run all tests with envtest (unit + Kubernetes integration)"
	@echo "  test-coverage  - Run all tests with coverage report and XML output"
	@echo "  test-informer  - Test Deployment informer with envtest"
	@echo "  test-ctrl      - Test Deployment controller with envtest"
	@echo "  test-config    - Test configuration package with envtest"
	@echo "  envtest        - Download setup-envtest tool for Kubernetes testing"
	@echo ""
	@echo "Dependency commands:"
	@echo "  get            - Get dependencies (download, tidy, verify)"
	@echo "  format         - Format code with goimports"
	@echo "  fmt            - Alias for format"
	@echo "  lint           - Lint code with golangci-lint"
	@echo "  vulncheck      - Check for vulnerabilities in dependencies"
	@echo ""
	@echo "Server commands (with Deployment Informer):"
	@echo "  server         - Build and start FastHTTP server with Deployment informer"
	@echo "                   (includes leader election and metrics)"
	@echo ""
	@echo "List commands:"
	@echo "  list           - List Kubernetes deployments"
	@echo "  list-namespace - List Kubernetes deployments in custom namespace (interactive)"
	@echo ""
	@echo "Development commands:"
	@echo "  check-env      - Check/create .env file with default configuration"
	@echo "  dev            - Development workflow (check-env, get, format, lint, test, build)"
	@echo "  dev-server     - Server development workflow (check-env, get, format, lint, test, server)"
	@echo "  prod           - Release build (clean, get, test, build)"
	@echo "  version-info   - Show version information"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build   - Build single-arch Docker image"
	@echo "  docker-build-multi - Build and push multi-arch Docker image (CI/CD)"
	@echo "  docker-clean   - Clean Docker images"
	@echo "  clean-all      - Clean build artifacts and Docker images"
	@echo "  push           - Push single-arch Docker image"
	@echo ""
	@echo "Help:"
	@echo "  help           - Show this help message" 