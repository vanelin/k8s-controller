# Kubernetes Controller

A Go-based Kubernetes controller with structured logging and environment configuration using Viper.

### Prerequisites

- Go 1.24 or newer
- Make
- curl (for installing golangci-lint)

## Features

- Load configuration from `.env` files
- Support for system environment variables
- Automatic environment variable detection
- Type-safe configuration struct
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
│       │   └── config.go
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

## Configuration System

The application uses a **smart configuration system** that ensures it always has valid settings, even without any configuration files or environment variables.

### Configuration Priority

The system follows this priority order (highest to lowest):

1. **CLI flags** - Override everything else
2. **System environment variables** - Set via `export` or command line
3. **`.env` file** - Local configuration file
4. **Default values** - Built-in fallback values

### Configuration Examples

```bash
# Zero-configuration (uses defaults)
./controller

# CLI flag override
./controller --log-level debug

# Environment variables
export PORT=:9090 && export LOGGING_LEVEL=debug && ./controller

# .env file
echo "PORT=:7070" > pkg/common/envs/.env && ./controller

# Mixed configuration
export PORT=:9090 && ./controller --log-level trace
```

### Zero-Configuration Setup

The application works out-of-the-box without any configuration:

```bash
# Just run it - all defaults will be used
./controller

# Output will show:
# Configuration:
#   PORT: :8080
#   LOGGING_LEVEL: info
#   KUBECONFIG: ~/.kube/config
```

## Usage

### Basic Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/vanelin/k8s-controller.git/pkg/common/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(config.GetConfigPath())
	if err != nil {
		log.Fatal(err)
	}

	// Use configuration
	fmt.Printf("Port: %s\n", cfg.Port)
	fmt.Printf("Log Level: %s\n", cfg.LoggingLevel)
	fmt.Printf("Kubeconfig: %s\n", cfg.KUBECONFIG)
}
```

### Setting Environment Variables

You can configure the application using different methods:

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

## Makefile Commands

### Quick Commands
- `make run` - Build and run the application
- `make run-debug` - Run with debug logging
- `make run-trace` - Run with trace logging
- `make dev` - Complete development workflow (check-env, deps, fmt, lint, test, build, run)

### Build Commands
- `make build` - Build the application
- `make build-linux` - Build for Linux (amd64, arm64)
- `make build-darwin` - Build for macOS (amd64, arm64)
- `make build-all` - Build for all platforms
- `make prod` - Production build (clean, deps, test, build)

### Development Commands
- `make clean` - Clean build artifacts
- `make deps` - Install dependencies
- `make fmt` - Format code
- `make lint` - Lint code
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage report

### Setup Commands
- `make check-env` - Check/create .env file
- `make install-lint` - Install golangci-lint
- `make delete-lint` - Remove golangci-lint
- `make help` - Show all available commands

## License

MIT License. See [LICENSE](LICENSE) for details. 