# K8s Controller Helm Chart

A Helm chart for deploying the Kubernetes Controller application with multi-architecture support.

## Prerequisites

- Kubernetes 1.30+
- Helm 3.0+

## Installation

### Basic Installation

```bash
helm install k8s-controller ./charts/app
```

### Installation with Custom Values

```bash
helm install k8s-controller ./charts/app \
  --set image.tag="v1.0.0-abc123" \
  --set app.port=9090 \
  --set deployment.replicas=3
```

## Multi-Architecture Support

This chart supports multi-architecture deployments. The Docker images are built for both `amd64` and `arm64` architectures using Docker manifests.

### Automatic Architecture Selection

By default, Kubernetes will automatically select the appropriate image for each node's architecture. The chart uses a single image tag that contains both architectures.

### Manual Architecture Selection

If you need to deploy to specific architectures, you can use node selectors:

```bash
# Deploy only to AMD64 nodes
helm install k8s-controller ./charts/app \
  --set nodeSelector."kubernetes\.io/arch"=amd64

# Deploy only to ARM64 nodes  
helm install k8s-controller ./charts/app \
  --set nodeSelector."kubernetes\.io/arch"=arm64
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `image.repository` | Docker image repository | `ghcr.io/vanelin/k8s-controller` |
| `image.tag` | Docker image tag | `0.1.0` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `nodeSelector` | Node selector for pod placement | `{}` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `app.port` | Application port | `8080` |
| `app.logLevel` | Log level | `info` |
| `env.KUBECONFIG` | Kubeconfig path | `~/.kube/config` |
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `80` |
| `deployment.replicas` | Number of replicas | `1` |
| `deployment.resources.limits.cpu` | CPU limit | `500m` |
| `deployment.resources.limits.memory` | Memory limit | `512Mi` |
| `deployment.resources.requests.cpu` | CPU request | `100m` |
| `deployment.resources.requests.memory` | Memory request | `128Mi` |

## Values

```yaml
image:
  repository: ghcr.io/vanelin/k8s-controller
  tag: "0.1.0"
  pullPolicy: IfNotPresent

# Node selector for multi-arch support
nodeSelector: {}
  # kubernetes.io/arch: amd64
  # kubernetes.io/arch: arm64

# Image pull secrets (if needed for private registry)
imagePullSecrets: []

app:
  port: 8080
  logLevel: "info"

env:
  KUBECONFIG: "~/.kube/config"

service:
  type: ClusterIP
  port: 80

deployment:
  replicas: 1
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi
```

## Upgrading

```bash
helm upgrade k8s-controller ./charts/app
```

## Uninstalling

```bash
helm uninstall k8s-controller
```

## Troubleshooting

### Check Pod Architecture

```bash
kubectl get pods -o wide
```

### Check Image Architecture

```bash
kubectl describe pod <pod-name>
```

### Verify Multi-Arch Manifest

```bash
docker manifest inspect ghcr.io/vanelin/k8s-controller:latest
```