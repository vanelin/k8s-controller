# k8s-controller Helm Chart

This Helm chart deploys the Kubernetes Controller application with FastHTTP server support.

## Features

- **FastHTTP Server** - High-performance HTTP server with configurable port and logging
- **Structured Logging** - Zero-config logging with zerolog
- **Kubernetes Integration** - Built-in Kubernetes configuration support
- **Configurable Resources** - Adjustable CPU and memory limits/requests

## Usage

### Basic Installation

```bash
# Install with default values
helm install k8s-controller ./charts/app
```

### Custom Configuration

```bash
# Override application settings (affects both app config and env vars)
helm install k8s-controller ./charts/app \
  --set app.port=9090 \
  --set app.logLevel=debug \
  --set deployment.replicas=3

# Override KUBECONFIG path
helm install k8s-controller ./charts/app \
  --set env.KUBECONFIG=/custom/path/kubeconfig

# Override resource limits
helm install k8s-controller ./charts/app \
  --set deployment.resources.limits.cpu=1000m \
  --set deployment.resources.limits.memory=1Gi
```

### Upgrade Existing Deployment

```bash
# Upgrade to new version
helm upgrade --install k8s-controller ./charts/app \
  --set image.tag=v1.1.0

# Upgrade with new configuration
helm upgrade --install k8s-controller ./charts/app \
  --set app.logLevel=info \
  --set deployment.replicas=2
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Docker image repository | `ghcr.io/vanelin/k8s-controller` |
| `image.tag` | Docker image tag | `0.1.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `app.port` | Application port (used for container port and PORT env) | `8080` |
| `app.logLevel` | Logging level (used for LOGGING_LEVEL env) | `info` |
| `env.KUBECONFIG` | Environment variable KUBECONFIG | `~/.kube/config` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `80` |
| `deployment.replicas` | Number of replicas | `1` |
| `deployment.resources.limits.cpu` | CPU limit | `500m` |
| `deployment.resources.limits.memory` | Memory limit | `512Mi` |
| `deployment.resources.requests.cpu` | CPU request | `100m` |
| `deployment.resources.requests.memory` | Memory request | `128Mi` |

## CI/CD Integration

The image tag is automatically set by CI/CD to the Git tag (if present) or the commit SHA when a release is created.

## Uninstall

```bash
helm uninstall k8s-controller
```