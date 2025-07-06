# Kubernetes Controller

![Visitor](https://visitor-badge.laobi.icu/badge?page_id=vanelin.k8s-controller)
[![Go Reference](https://pkg.go.dev/badge/github.com/vanelin/k8s-controller.svg?style=flat-square)](https://pkg.go.dev/github.com/vanelin/k8s-controller)
[![Go Report Card](https://goreportcard.com/badge/github.com/vanelin/k8s-controller)](https://goreportcard.com/report/github.com/vanelin/k8s-controller)
[![CI](https://img.shields.io/github/actions/workflow/status/vanelin/k8s-controller/ci.yml?branch=main&style=flat-square&logo=githubactions&logoColor=white&label=CI)](https://github.com/vanelin/k8s-controller/actions/workflows/ci.yml)
![Repo size](https://img.shields.io/github/repo-size/vanelin/k8s-controller?style=flat-square)
[![Updates](https://img.shields.io/github/last-commit/vanelin/k8s-controller.svg?style=flat-square&logo=git&logoColor=white&color=blue)](https://github.com/vanelin/k8s-controller/commits/main/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)

A Go-based Kubernetes controller with structured logging, environment configuration using Viper, FastHTTP server, and Deployment informer capabilities.

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
- **REST API** - JSON API endpoints for deployment information with multi-namespace support
- **Deployment Informer** - Real-time Kubernetes Deployment event monitoring using client-go informers
- **Controller-runtime Deployment Controller** - Production-grade Kubernetes controller using controller-runtime framework with reconciliation loops
- **Kubernetes Integration** - List deployments and manage Kubernetes resources with namespace support
- **Smart Configuration** - Load from `.env` files, environment variables, or CLI flags with proper priority
- **Structured Logging** - Zero-config logging with zerolog and controller-runtime integration
- **Development Tools** - Comprehensive Makefile with development workflows
- **Multi-arch Docker** - Official images for `linux/amd64` and `linux/arm64`
- **Helm Chart** - Easy deployment to Kubernetes
- **Comprehensive Testing** - Unit tests with coverage reporting and envtest integration
- **Graceful Shutdown** - Proper signal handling and resource cleanup
- **Metrics Server** - Prometheus metrics endpoint for controller monitoring

## Prerequisites

- [Go](https://golang.org/dl/) 1.24 or newer
- [Make](https://www.gnu.org/software/make/) (install via package manager)
- [curl](https://curl.se/download.html) (for installing golangci-lint)
- [Docker](https://docs.docker.com/engine/install/) (for building images)
- [Helm](https://helm.sh/docs/intro/install/) (for packaging/deploying charts)

## Project Structure

```
k8s-controller/
├── cmd/
│   ├── root.go                    # Main CLI application
│   ├── server.go                  # FastHTTP server command with informer and controller
│   ├── server_test.go
│   ├── list.go                    # Kubernetes deployments list command
│   └── list_test.go
├── pkg/
│   ├── common/
│   │   ├── config/                # Configuration management
│   │   │   ├── config.go
│   │   │   └── config_test.go
│   │   ├── utils/                 # Utility functions
│   │   │   └── k8s.go
│   │   └── envs/                  # Environment files
│   │       └── .env
│   ├── handlers/                  # HTTP handlers for API endpoints
│   │   ├── handlers.go
│   │   ├── handlers_test.go
│   │   └── handlers_env_test.go
│   ├── informer/                  # Deployment informer implementation
│   │   ├── informer.go
│   │   └── informer_test.go
│   ├── ctrl/                      # Controller-runtime implementations
│   │   ├── deployment_controller.go
│   │   └── deployment_controller_test.go
│   └── testutil/                  # Testing utilities and envtest setup
│       ├── envtest.go
│       └── envtest_test.go
├── main.go                        # Application entry point
├── Makefile                       # Development and build commands
├── charts/app/                    # Helm chart
└── README.md
```

## Configuration

### Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `METRIC_PORT` | Controller-runtime metrics server port | `8081` |
| `KUBECONFIG` | Path to Kubernetes configuration file | `~/.kube/config` |
| `IN_CLUSTER` | Use in-cluster Kubernetes config | `false` |
| `NAMESPACE` | Kubernetes namespace(s) for operations (comma-separated, e.g., "kube-system,monitoring") | `default` |
| `LOGGING_LEVEL` | Logging level (trace, debug, info, warn, error) | `info` |

### Configuration Priority

All commands follow the same configuration priority:

1. **CLI flags** (`--port`, `--metric-port`, `--log-level`, `--kubeconfig`, `--in-cluster`, `--namespace`) - highest priority
2. **Environment variables** (`PORT`, `METRIC_PORT`, `LOGGING_LEVEL`, `KUBECONFIG`, `IN_CLUSTER`, `NAMESPACE`)
3. **`.env` file** values (`pkg/common/envs/.env`)
4. **Default values** (PORT=8080, METRIC_PORT=8081, LOGGING_LEVEL=info, KUBECONFIG=~/.kube/config, IN_CLUSTER=false, NAMESPACE=default)

## Quick Start

### Using Makefile (Recommended)

```bash
# Show all available commands
make help

# Start FastHTTP server with Deployment informer
make server

# List Kubernetes deployments
make list

# List deployments in custom namespace
make list-namespace

# Start server with multiple namespaces via environment variable
export NAMESPACE=kube-system,monitoring && make server

# Test controller-runtime Deployment controller
make test-ctrl

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

# Build single-arch Docker image
make docker-build
```

## Commands

### FastHTTP Server with Deployment Informer

The main feature of this application is a high-performance FastHTTP server that can optionally run a Deployment informer to monitor Kubernetes events in real-time.

#### Basic Usage

```bash
# Development mode with informer
go run main.go server --kubeconfig ~/.kube/config

# With custom configuration
go run main.go server --port 9090 --metric-port 9091 --log-level debug --kubeconfig ~/.kube/config --namespace kube-system

# Using in-cluster configuration
go run main.go server --in-cluster --namespace kube-system

# Multiple namespaces (comma-separated)
go run main.go server --namespace kube-system,monitoring,default

# Using environment variable for multiple namespaces
export NAMESPACE=kube-system,monitoring,default
go run main.go server

# List deployments
go run main.go list

# List deployments with custom namespace
go run main.go list --namespace kube-system

# List deployments in multiple namespaces
go run main.go list --namespace kube-system,test

# Using environment variable for multiple namespaces
export NAMESPACE=kube-system,test
go run main.go list

# Server with environment variables
export PORT=9090 && export METRIC_PORT=9091 && export LOGGING_LEVEL=debug && go run main.go server

# List with environment variables
export KUBECONFIG=~/.kube/config-prod && export NAMESPACE=monitoring && go run main.go list
```

#### What it does

- Starts a FastHTTP server on the specified port (default: 8080)
- Starts a controller-runtime manager with Deployment controller on the specified metrics port (default: 8081)
- Provides JSON API endpoints for deployment information:
  - `/` - Root endpoint with version information
  - `/namespaces` - List all watched namespaces
  - `/deployments` - List deployments from all watched namespaces
  - `/deployments/{namespace}` - List deployments in specific namespace
- Provides Prometheus metrics endpoint at `:8081/metrics` for controller monitoring

#### API Examples

```bash
# Start server with multiple namespaces
./k8s-controller server --kubeconfig ~/.kube/config --port 8080 --metric-port 8081 -n kube-system,monitoring

# Get all watched namespaces
curl -s http://localhost:8080/namespaces
# Output: {"namespaces":["kube-system","monitoring"],"count":2}

# Get deployments from all watched namespaces
curl -s http://localhost:8080/deployments
# Output: {
#   "namespaces": [
#     {
#       "namespace": "kube-system",
#       "deployments": ["system-1"],
#       "count": 1
#     },
#     {
#       "namespace": "monitoring",
#       "deployments": ["grafana", "loki", "prometheus"],
#       "count": 3
#     }
#   ],
#   "total_count": 4
# }

# Get deployments in specific namespace
curl -s http://localhost:8080/deployments/monitoring
# Output: {"namespace":"monitoring","deployments":["grafana","loki","prometheus"],"count":3}

curl -s http://localhost:8080/deployments/kube-system
# Output: {"namespace":"kube-system","deployments":["system-1"],"count":1}

# Get root endpoint with version info
curl -s http://localhost:8080/
# Output: {"endpoints":{"deployments":"/deployments","namespaces":"/namespaces"},"message":"Kubernetes Controller API","version":"v0.1.2"}
```

**Note:** The `/deployments` endpoint returns deployments from all namespaces being watched by the informer, not just the default namespace. This provides a comprehensive view of all deployments across monitored namespaces.
- **Deployment Informer**: Watches for Deployment events (add, update, delete) in the specified namespace(s)
- **Controller-runtime Deployment Controller**: Provides production-grade reconciliation loops for Deployments with proper error handling and retry logic
- Uses structured logging with configurable levels (zerolog integration for both FastHTTP server and controller-runtime)
- Supports hot-reload configuration via environment variables
- Implements graceful shutdown with proper signal handling
- Provides Prometheus metrics for monitoring controller performance

### Controller-runtime Deployment Controller

The project includes a production-grade Kubernetes controller built using the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) framework. This controller provides:

- **Reconciliation Loops**: Proper handling of Deployment events with retry logic and exponential backoff
- **Namespace Filtering**: Controller only processes Deployments in specified namespaces
- **Structured Logging**: Integration with zerolog for consistent logging across the application
- **Metrics**: Prometheus metrics endpoint for monitoring controller performance
- **Leader Election**: Support for high availability deployments (future enhancement)

#### Controller Features

- **Event Logging**: Logs each reconciliation event for Deployments
- **Error Handling**: Proper error handling with retry mechanisms
- **Resource Validation**: Validates Deployment specifications
- **Metrics Collection**: Tracks reconciliation duration, success/failure rates
- **Graceful Shutdown**: Proper cleanup when the controller stops

#### Testing the Controller

```bash
# Test the controller with envtest
make test-ctrl

# Run controller tests with verbose output
make test-ctrl TEST_ARGS="-v"
```

### Kubernetes List Command

List Kubernetes deployments in the specified namespace(s) with configurable kubeconfig and comprehensive error handling.

#### Basic Usage

```bash
# Run kubernetes-control-plane using script (choose your architecture)
./scripts/setup-arm64.sh start  # For ARM64 systems
./scripts/setup-amd64.sh start  # For AMD64/x86_64 systems

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

# List deployments in multiple namespaces
./k8s-controller list --namespace kube-system,test
```

#### What it does

Lists deployments in the specified namespace(s) with error handling and logging. Supports multiple namespaces separated by commas. When multiple namespaces are specified, deployments are listed per namespace with a total count.

#### List Command Configuration

```bash
# Build the application first
make build

# Default configuration
./k8s-controller list

# Custom kubeconfig
./k8s-controller list --kubeconfig ~/.kube/config-prod

# Custom namespace
./k8s-controller list --namespace kube-system

# Multiple namespaces
./k8s-controller list --namespace kube-system,monitoring

# Using environment variable for multiple namespaces
export NAMESPACE=kube-system,monitoring
./k8s-controller list

# Environment variable for kubeconfig
export KUBECONFIG=/path/to/kubeconfig && ./k8s-controller list

# Mixed configuration
export KUBECONFIG=/path/to/kubeconfig && ./k8s-controller list --namespace monitoring --log-level debug
```

## Testing with envtest and Inspecting with kubectl

This project uses [envtest](https://book.kubebuilder.io/reference/envtest.html) to spin up a local Kubernetes API server for integration tests. The test environment writes a kubeconfig to `/tmp/envtest.kubeconfig` so you can inspect the in-memory cluster with `kubectl` while tests are running.

### How to Run and Inspect

1. **Run the informer test:**
   ```sh
   make test-informer
   ```
   This will:
   - Start envtest and create sample Deployments
   - Write a kubeconfig to `/tmp/envtest.kubeconfig`
   - Sleep for 5 minutes at the end of the test so you can inspect the cluster

2. **Run the controller test:**
   ```sh
   make test-ctrl
   ```
   This will:
   - Start envtest and create sample Deployments
   - Test the controller-runtime Deployment controller
   - Verify reconciliation logic and error handling

2. **In another terminal, use kubectl:**
   ```sh
   kubectl --kubeconfig=/tmp/envtest.kubeconfig get all -A
   kubectl --kubeconfig=/tmp/envtest.kubeconfig get deployments -n default
   kubectl --kubeconfig=/tmp/envtest.kubeconfig describe pod -n default
   ```
   You can use any standard kubectl commands to inspect resources created by the test.

3. **Notes:**
   - The envtest cluster only exists while the test is running. Once the test finishes, the API server is shut down and the kubeconfig is no longer valid.
   - You can adjust the sleep duration in `TestStartDeploymentInformer` if you need more or less time for inspection.

## Makefile Commands

For a complete list of available commands and their descriptions, run:
```bash
make help
```

### Important Notes

**Architecture Detection:** The Makefile automatically detects your system architecture and sets appropriate defaults:
- `x86_64` >> `TARGETARCH=amd64`
- `aarch64` >> `TARGETARCH=arm64`

**Variable Override:** You can override any Makefile variable:
```bash
# Override default values
make server SERVER_PORT=9090 METRIC_PORT=9091 LOGGING_LEVEL=debug NAMESPACE=kube-system,monitoring

# Cross-compilation
make build TARGETOS=linux TARGETARCH=amd64

# Custom Docker registry
make docker-build-multi REGISTRY=my-registry.com REPOSITORY=my-org
```

**Environment Variables:** The Makefile respects environment variables with the same names:
```bash
export SERVER_PORT=9090
export METRIC_PORT=9091
export LOGGING_LEVEL=debug
export KUBECONFIG=~/.kube/config-prod
make server
```

**Test Environment:** Test commands automatically set up `KUBEBUILDER_ASSETS` for envtest integration - no manual configuration needed.

## Deployment Examples

This section provides examples of how to deploy the controller in different environments. Note that this is a development/experimental project and should not be used in production environments without proper testing and validation.

### Using Docker

```bash
# Pull the latest image
docker pull ghcr.io/vanelin/k8s-controller:latest

# Run with custom configuration
docker run --rm \
  --name k8s-controller \
  -v ~/.kube:/root/.kube:ro \
  -e KUBECONFIG=/root/.kube/config \
  -e IN_CLUSTER=false \
  -e LOGGING_LEVEL=debug \
  -e NAMESPACE=kube-system,monitoring,default \
  -e METRIC_PORT=8081 \
  -p 8080:8080 \
  -p 8081:8081 \
  ghcr.io/vanelin/k8s-controller:latest server
```

### Using Helm Chart

The Helm chart is located in the `charts/app/` directory.

#### Package the Chart
```bash
helm package charts/app --version <version> --app-version <version>
```

#### Install the Chart
```bash
# Basic installation
helm upgrade --install k8s-controller ./k8s-controller-helm-chart.tgz \
  --namespace k8s-controller \
  --create-namespace

# With custom values
helm upgrade --install k8s-controller ./k8s-controller-helm-chart.tgz \
  --namespace k8s-controller \
  --create-namespace \
  --set server.port=9090 \
  --set server.metricPort=9091 \
  --set server.logLevel=info \
  --set server.namespace=monitoring \
  --set server.inCluster=true
```

## Getting Help

```bash
# General help
./k8s-controller --help

# Server command help
./k8s-controller server --help

# List command help
./k8s-controller list --help
```

## Troubleshooting

### Common Issues

#### Permission Denied Errors
```bash
# If you get permission errors with kubeconfig
chmod 600 ~/.kube/config

# For Docker builds
sudo usermod -aG docker $USER
```

#### Port Already in Use
```bash
# Check what's using the port
sudo netstat -tulpn | grep :8080

# Use a different port
./k8s-controller server --port 8081 --metric-port 8082
```

#### Kubernetes Connection Issues
```bash
# Test kubectl connection
./kubebuilder/bin/kubectl cluster-info  

# Check if kubeconfig is valid
./kubebuilder/bin/kubectl config view --raw --minify --flatten

# Use in-cluster config if running inside Kubernetes
./k8s-controller server --in-cluster
```

#### Build Issues
```bash
# Clean and rebuild
make clean
make build

# Update dependencies
make get
```

### Debug Mode

Enable debug logging to get more detailed information:
```bash
# Set debug level
export LOGGING_LEVEL=debug

# Or use CLI flag
./k8s-controller server --log-level debug
```

## License

MIT License. See [LICENSE](LICENSE) for details.