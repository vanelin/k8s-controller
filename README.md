# Kubernetes Controller

A Go-based Kubernetes controller with structured logging and environment configuration using Viper.

## Features

- Load configuration from `.env` files
- Support for system environment variables
- Automatic environment variable detection
- Type-safe configuration struct
- Graceful fallback when `.env` file is not found
- Structured logging with zerolog
- Kubernetes configuration support

## Project Structure

```
controller/
├── cmd/
│   └── root.go
├── pkg/
│   └── common/
│       ├── config/
│       │   ├── config.go
│       │   └── example.go
│       └── envs/
│           └── .env
├── main.go
├── Makefile
└── README.md
```

## Quick Start

### Using Makefile (Recommended)

```bash
# Show all available commands
make help

# Build and run the application
make run

# Run with debug logging
make run-debug

# Run with trace logging
make run-trace

# Run with custom environment
make run-env

# Development workflow
make dev

# Production build
make prod
```

### Manual Build

```bash
# Build the application
go build -o controller

# Run the application
./controller
```

## Configuration Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `:8080` | No |
| `KUBECONFIG` | Path to Kubernetes configuration file | `~/.kube/config` | No |
| `LOGGING_LEVEL` | Logging level (trace, debug, info, warn, error) | `info` | No |

## Usage

### Basic Usage

```go
import "github.com/vanelin/k8s-controller.git/pkg/common/config"

// Load configuration
config, err := config.LoadConfig(config.GetConfigPath())
if err != nil {
    log.Fatal(err)
}

// Use configuration
fmt.Printf("Server port: %s\n", config.Port)
fmt.Printf("Logging level: %s\n", config.LoggingLevel)
fmt.Printf("Kubeconfig path: %s\n", config.KUBECONFIG)
```

### Setting Environment Variables

#### 1. Using .env file

Create a `.env` file in `pkg/common/envs/`:

```env
PORT=:8080
KUBECONFIG=~/.kube/config
LOGGING_LEVEL=debug
```

#### 2. Using system environment variables

```bash
export PORT=:8080
export KUBECONFIG=~/.kube/config
export LOGGING_LEVEL=debug
```

#### 3. Using command line

```bash
PORT=:8080 KUBECONFIG=~/.kube/config LOGGING_LEVEL=trace ./controller
```

#### 4. Using CLI flag (overrides config)

```bash
./controller --log-level debug
```

## Priority Order

1. CLI flags (highest priority)
2. System environment variables
3. `.env` file values
4. Default values (lowest priority)

## Makefile Commands

### Build Commands
- `make build` - Build the application
- `make build-linux` - Build for Linux (amd64, arm64)
- `make build-darwin` - Build for macOS (amd64, arm64)
- `make build-all` - Build for all platforms

### Development Commands
- `make run` - Build and run the application
- `make run-debug` - Run with debug logging
- `make run-trace` - Run with trace logging
- `make run-env` - Run with custom environment variables
- `make dev` - Complete development workflow

### Utility Commands
- `make clean` - Clean build artifacts
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report
- `make deps` - Install dependencies
- `make fmt` - Format code
- `make lint` - Lint code
- `make check-env` - Check/create .env file
- `make install-lint` - Install golangci-lint
- `make delete-lint` - Remove golangci-lint

### Production Commands
- `make prod` - Production build (clean, deps, test, build)

## Integration with CLI

The configuration is automatically loaded when running the CLI application. You can see the current configuration by running:

```bash
./controller
```

This will display the current configuration values including the KUBECONFIG path.

### Logging Level Examples

```bash
# Use config default (info)
./controller

# Override with CLI flag
./controller --log-level debug

# Override with environment variable
LOGGING_LEVEL=trace ./controller

# Override with .env file
echo "LOGGING_LEVEL=warn" >> pkg/common/envs/.env
./controller
```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make
- curl (for installing golangci-lint)

### Initial Setup

```bash
# Clone the repository
git clone <repository-url>
cd controller

# Install dependencies
make deps

# Install linter
make install-lint

# Create default environment file
make check-env

# Run development workflow
make dev
```

### Code Quality

```bash
# Format code
make fmt

# Lint code
make lint

# Run tests
make test

# Run tests with coverage
make test-coverage
```

## License

MIT License. See [LICENSE](LICENSE) for details. 