# Kubernetes Controller

![Visitor](https://visitor-badge.laobi.icu/badge?page_id=vanelin.k8s-controller)
[![Go Reference](https://pkg.go.dev/badge/github.com/vanelin/k8s-controller.svg?style=flat-square)](https://pkg.go.dev/github.com/vanelin/k8s-controller)
[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/vanelin/k8s-controller/ci.yml?branch=main&style=flat-square&logo=githubactions&logoColor=white&label=test-n-build)](https://github.com/vanelin/k8s-controller/actions/workflows/ci.yml)
![Repo size](https://img.shields.io/github/repo-size/vanelin/k8s-controller?style=flat-square)
[![Updates](https://img.shields.io/github/last-commit/vanelin/k8s-controller.svg?style=flat-square&logo=git&logoColor=white&color=blue)](https://github.com/vanelin/k8s-controller/commits/main/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)

A Go-based Kubernetes controller with structured logging, environment configuration using Viper, and a FastHTTP server.

## 📦 Release Artifacts

- **Docker Image (multi-arch):**
  ```bash
  docker pull ghcr.io/vanelin/k8s-controller:<version>
  # Example: docker pull ghcr.io/vanelin/k8s-controller:0.1.1
  ```
- **Binary Archives:**
  - [Linux (amd64)](https://github.com/vanelin/k8s-controller/releases/latest/download/k8s-controller-linux-amd64.tar.gz)
  - [Linux (arm64)](https://github.com/vanelin/k8s-controller/releases/latest/download/k8s-controller-linux-arm64.tar.gz)
- **Helm Chart:**
  - [k8s-controller-helm-chart.tgz](https://github.com/vanelin/k8s-controller/releases/latest/download/k8s-controller-helm-chart.tgz)

## Features

- **FastHTTP Server** - High-performance HTTP server with configurable port and logging
- **Kubernetes Integration** - List deployments and manage Kubernetes resources with namespace support
- **Smart Configuration** - Load from `.env` files, environment variables, or CLI flags with proper priority
- **Structured Logging** - Zero-config logging with zerolog
- **Development Tools** - Comprehensive Makefile with development workflows
- **Multi-arch Docker** - Official images for `linux/amd64` and `linux/arm64`
- **Helm Chart** - Easy deployment to Kubernetes
- **Comprehensive Testing** - Unit tests with coverage reporting

## Prerequisites

- Go 1.24 or newer
- Make
- curl (for installing golangci-lint)
- Docker (for building images)
- Helm (for packaging/deploying charts)
- Kubernetes cluster access (for list command)

## Project Structure

```
k8s-controller/
├── cmd/
│   ├── root.go          # Main CLI application
│   ├── server.go        # FastHTTP server command
│   ├── server_test.go   # Tests for server command
│   ├── list.go          # Kubernetes deployments list command
│   └── list_test.go     # Tests for list command
├── pkg/
│   └── common/
│       ├── config/      # Configuration management
│       │   ├── config.go
│       │   └── config_test.go
│       └── envs/        # Environment files
│           └── .env
├── main.go              # Application entry point
├── Makefile             # Development and build commands
├── charts/app/          # Helm chart
└── README.md
```

## Quick Start

### Using Makefile (Recommended)

```bash
# Show all available commands
make help

# Start FastHTTP server (most common use case)
make server

# Start server with debug logging
make server-debug

# Start server with trace logging
make server-trace

# List Kubernetes deployments
make list

# List deployments in custom namespace
make list-namespace

# Development workflow
make dev-server

# Production build
make prod

# Build multi-arch Docker image (amd64, arm64)
make docker-build-multi VERSION=0.1.1

# Build and package Helm chart
make build-linux VERSION=0.1.1
helm package charts/app --version 0.1.1 --app-version 0.1.1
```

### Manual Build

```bash
# Build the application
make build

# Build for Linux (amd64, arm64)
make build-linux

# Start FastHTTP server
./k8s-controller server

# Start server with custom port and log level
./k8s-controller server --port 9090 --log-level debug

# List Kubernetes deployments
./k8s-controller list

# List deployments in custom namespace
./k8s-controller list --namespace kube-system

# Using short flags
./k8s-controller server -p 8080 -l debug
./k8s-controller list -n kube-system
```

## Commands

### FastHTTP Server

The main feature of this application is a high-performance FastHTTP server that can be configured through multiple methods.

#### Basic Usage

```bash
# Development mode
go run main.go server

# Production mode
./k8s-controller server

# With custom configuration
./k8s-controller server --port 9090 --log-level debug
```

#### What it does

- Starts a FastHTTP server on the specified port (default: 8080)
- Responds with "Hello from FastHTTP!" to any HTTP request
- Uses structured logging with configurable levels
- Supports hot-reload configuration via environment variables

### Kubernetes List Command

List Kubernetes deployments in the specified namespace with configurable kubeconfig and comprehensive error handling.

#### Basic Usage

```bash
# Run kubernetes-control-plane using script
./scripts/setup-arm64.sh start

# Creating test deployment
kubebuilder/bin/kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-nginx-deployment
  labels:
    app: test-nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-nginx
  template:
    metadata:
      labels:
        app: test-nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
        ports:
        - containerPort: 80
        securityContext:
          privileged: true
        resources: {}
EOF

# List deployments using default kubeconfig and namespace
./k8s-controller list

# List deployments with debug logging
./k8s-controller list --log-level debug

# List deployments in custom namespace
./k8s-controller list --namespace kube-system
```

#### What it does

- Connects to Kubernetes cluster using specified kubeconfig with proper priority handling
- Validates namespace existence and provides helpful error messages
- Lists all deployments in the specified namespace (default: 'default')
- Shows available namespaces if the requested namespace doesn't exist
- Uses structured logging for connection and error reporting
- Supports multiple configuration sources with proper priority

#### Error Handling

The list command provides comprehensive error handling:

- **Invalid kubeconfig**: Shows clear error message with kubeconfig path
- **Non-existent namespace**: Lists all available namespaces to help user choose correct one
- **Connection issues**: Provides detailed error messages for troubleshooting
- **Permission issues**: Clear indication of authentication/authorization problems

## Configuration

### Configuration Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `KUBECONFIG` | Path to Kubernetes configuration file | `~/.kube/config` | No |
| `LOGGING_LEVEL` | Logging level (trace, debug, info, warn, error) | `info` | No |

### Configuration Priority

Both server and list commands follow the same configuration priority:

1. **CLI flags** (`--port`, `--log-level`, `--kubeconfig`, `--namespace`) - highest priority
2. **Environment variables** (`PORT`, `LOGGING_LEVEL`, `KUBECONFIG`)
3. **`.env` file** values
4. **Default values** (PORT=8080, LOGGING_LEVEL=info, KUBECONFIG=~/.kube/config, namespace=default)

### Configuration Examples

#### Server Configuration

```bash
# Zero-configuration (uses defaults)
./k8s-controller server

# CLI flag override
./k8s-controller server --port 9090 --log-level debug

# Environment variables
export PORT=9090 && export LOGGING_LEVEL=debug && ./k8s-controller server

# .env file
cat <<EOF > pkg/common/envs/.env
PORT=7070
LOGGING_LEVEL=debug
EOF
./k8s-controller server

# Mixed configuration
export PORT=9090 && ./k8s-controller server --log-level trace
```

#### List Command Configuration

```bash
# Default configuration
./k8s-controller list

# Custom kubeconfig
./k8s-controller list --kubeconfig ~/.kube/config-prod

# Custom namespace
./k8s-controller list --namespace kube-system

# Environment variable for kubeconfig
export KUBECONFIG=/path/to/kubeconfig && ./k8s-controller list

# Mixed configuration
export KUBECONFIG=/path/to/kubeconfig && ./k8s-controller list --namespace monitoring --log-level debug
```

## Development

### Development Mode

```bash
# Run server directly
go run main.go server

# Run list command directly
go run main.go list

# With custom settings
go run main.go server --port 8080 --log-level debug
go run main.go list --kubeconfig /path/to/config --log-level debug
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Full development workflow
make dev-server
```

## Makefile Commands

### Server Commands
- `make server` - Build and start FastHTTP server
- `make server-debug` - Start server with debug logging
- `make server-trace` - Start server with trace logging

### List Commands
- `make list` - Build and run list command
- `make list-namespace` - Run list command with custom namespace (interactive)

### Test Commands
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report

### Dependency Commands
- `make get` - Get and verify dependencies
- `make format` - Format code
- `make lint` - Lint code
- `make vulncheck` - Check for vulnerabilities in dependencies (auto-installs govulncheck if needed)

### Development Commands
- `make dev` - Development workflow (check-env, get, format, lint, test, vulncheck, build)
- `make dev-server` - Server development workflow (check-env, get, format, lint, test, server)
- `make check-env` - Check/create .env file
- `make prod` - Production build (clean, get, test, build)

### Build Commands
- `make build` - Build the application
- `make build-linux` - Build for Linux (amd64, arm64)
- `version-info` - Show version information"
- `make clean` - Clean build artifacts
- `make clean-all` - Clean build artifacts and Docker images

### Docker Commands
- `make docker-build` - Build single-arch Docker image
- `make docker-build-multi` - Build and push multi-arch Docker image (amd64, arm64)
- `make docker-clean` - Clean Docker images
- `make push` - Push single-arch Docker image

### Cross-compilation Examples
```bash
make build TARGETOS=linux TARGETARCH=arm64
make build TARGETOS=linux TARGETARCH=amd64
```

## Helm Chart

- The Helm chart is located in the `charts/app/` directory.
- To package the chart:
  ```bash
  helm package charts/app --version <version> --app-version <version>
  ```
- To install the chart:
  ```bash
  helm upgrade --install k8s-controller ./k8s-controller-helm-chart.tgz \
    --namespace <your-namespace> \
    --create-namespace
  ```

## Getting Help

```bash
# General help
./k8s-controller --help

# Server command help
./k8s-controller server --help

# List command help
./k8s-controller list --help

# Makefile help
make help
```

## License

MIT License. See [LICENSE](LICENSE) for details.