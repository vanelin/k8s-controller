# Kubernetes Controller

![Visitor](https://visitor-badge.laobi.icu/badge?page_id=vanelin.k8s-controller)
[![Go Reference](https://pkg.go.dev/badge/github.com/vanelin/k8s-controller.svg?style=flat-square)](https://pkg.go.dev/github.com/vanelin/k8s-controller)
[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/vanelin/k8s-controller/ci.yml?branch=main&style=flat-square&logo=githubactions&logoColor=white&label=test-n-build)](https://github.com/vanelin/k8s-controller/actions/workflows/ci.yml)
![Repo size](https://img.shields.io/github/repo-size/vanelin/k8s-controller?style=flat-square)
[![Updates](https://img.shields.io/github/last-commit/vanelin/k8s-controller.svg?style=flat-square&logo=git&logoColor=white&color=blue)](https://github.com/vanelin/k8s-controller/commits/main/)

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
- **Smart Configuration** - Load from `.env` files, environment variables, or CLI flags
- **Structured Logging** - Zero-config logging with zerolog
- **Kubernetes Integration** - Built-in Kubernetes configuration support
- **Development Tools** - Comprehensive Makefile with development workflows
- **Multi-arch Docker** - Official images for `linux/amd64` and `linux/arm64`
- **Helm Chart** - Easy deployment to Kubernetes

## Prerequisites

- Go 1.24 or newer
- Make
- curl (for installing golangci-lint)
- Docker (for building images)
- Helm (for packaging/deploying charts)

## Project Structure

```
k8s-controller/
├── cmd/
│   ├── root.go          # Main CLI application
│   └── server.go        # FastHTTP server command
├── pkg/
│   └── common/
│       ├── config/      # Configuration management
│       │   └── config.go
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

# Start server with custom port (interactive)
make server-port

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

# Using short flags
./k8s-controller server -p 8080 -l debug
```

## FastHTTP Server

The main feature of this application is a high-performance FastHTTP server that can be configured through multiple methods.

### Basic Usage

```bash
# Development mode
go run main.go server

# Production mode
./k8s-controller server

# With custom configuration
./k8s-controller server --port 9090 --log-level debug
```

### What it does

- Starts a FastHTTP server on the specified port (default: 8080)
- Responds with "Hello from FastHTTP!" to any HTTP request
- Uses structured logging with configurable levels
- Supports hot-reload configuration via environment variables

### Configuration Priority

1. **CLI flags** (`--port`, `--log-level`) - highest priority
2. **Environment variables** (`PORT`, `LOGGING_LEVEL`)
3. **`.env` file** values
4. **Default values** (PORT=8080, LOGGING_LEVEL=info)

## Configuration

### Configuration Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `KUBECONFIG` | Path to Kubernetes configuration file | `~/.kube/config` | No |
| `LOGGING_LEVEL` | Logging level (trace, debug, info, warn, error) | `info` | No |

### Configuration Examples

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

## Development

### Development Mode

```bash
# Run server directly
go run main.go server

# With custom settings
go run main.go server --port 8080 --log-level debug
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
- `make server-port` - Start server on custom port (interactive)
- `make server-env` - Start server with custom environment (interactive)

### Development Commands
- `make dev` - Complete development workflow
- `make dev-server` - Server development workflow
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage
- `make format` - Format code
- `make lint` - Lint code

### Build Commands
- `make build` - Build the application
- `make build-linux` - Build for Linux (amd64, arm64)
- `make prod` - Production build

### Docker Commands
- `make docker-build` - Build single-arch Docker image
- `make docker-build-multi` - Build and push multi-arch Docker image (amd64, arm64)
- `make docker-run` - Build and run single-arch Docker container
- `make docker-clean` - Clean Docker images
- `make clean-all` - Clean build artifacts and Docker images
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

# Makefile help
make help
```

## License

MIT License. See [LICENSE](LICENSE) for details.