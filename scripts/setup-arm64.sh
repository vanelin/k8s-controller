#!/bin/bash

# Exit on error
set -e

echo "Setting up Kubernetes control plane for ARM64..."

# Function to check if a process is running
is_running() {
    pgrep -f "$1" >/dev/null
}

# Function to check if all components are running
check_running() {
    is_running "etcd" && \
    is_running "kube-apiserver" && \
    is_running "kube-controller-manager" && \
    is_running "kube-scheduler" && \
    is_running "kubelet" && \
    is_running "containerd-k8s"
}

# Function to kill process if running
stop_process() {
    if is_running "$1"; then
        echo "Stopping $1..."
        sudo pkill -f "$1" || true
        while is_running "$1"; do
            sleep 1
        done
    fi
}

download_components() {
    # Create necessary directories if they don't exist
    sudo mkdir -p ./kubebuilder/bin
    sudo mkdir -p /etc/cni/net.d
    sudo mkdir -p /var/lib/kubelet
    sudo mkdir -p /etc/kubernetes/manifests
    sudo mkdir -p /var/log/kubernetes
    sudo mkdir -p /etc/containerd-k8s/
    sudo mkdir -p /run/containerd-k8s
    sudo mkdir -p /var/lib/containerd-k8s

    # Download kubebuilder tools if not present
    if [ ! -f "kubebuilder/bin/etcd" ]; then
        echo "Downloading kubebuilder tools..."
        curl -L https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.30.0-linux-arm64.tar.gz -o /tmp/kubebuilder-tools.tar.gz
        sudo tar -C ./kubebuilder --strip-components=1 -zxf /tmp/kubebuilder-tools.tar.gz
        rm /tmp/kubebuilder-tools.tar.gz
        sudo chmod -R 755 ./kubebuilder/bin
    fi

    if [ ! -f "kubebuilder/bin/kubelet" ]; then
        echo "Downloading kubelet..."
        sudo curl -L "https://dl.k8s.io/v1.30.0/bin/linux/arm64/kubelet" -o kubebuilder/bin/kubelet
        sudo chmod 755 kubebuilder/bin/kubelet
    fi

    # Install CNI components if not present
    if [ ! -d "/opt/cni" ]; then
        sudo mkdir -p /opt/cni
        
        echo "Installing containerd for k8s..."
        wget https://github.com/containerd/containerd/releases/download/v2.0.5/containerd-static-2.0.5-linux-arm64.tar.gz -O /tmp/containerd.tar.gz
        sudo tar zxf /tmp/containerd.tar.gz -C /opt/cni/
        rm /tmp/containerd.tar.gz

        echo "Installing runc..."
        sudo curl -L "https://github.com/opencontainers/runc/releases/download/v1.2.6/runc.arm64" -o /opt/cni/bin/runc

        echo "Installing CNI plugins..."
        wget https://github.com/containernetworking/plugins/releases/download/v1.6.2/cni-plugins-linux-arm64-v1.6.2.tgz -O /tmp/cni-plugins.tgz
        sudo tar zxf /tmp/cni-plugins.tgz -C /opt/cni/bin/
        rm /tmp/cni-plugins.tgz

        # Set permissions for all CNI components
        sudo chmod -R 755 /opt/cni
    fi

    if [ ! -f "kubebuilder/bin/kube-controller-manager" ]; then
        echo "Downloading additional components..."
        sudo curl -L "https://dl.k8s.io/v1.30.0/bin/linux/arm64/kube-controller-manager" -o kubebuilder/bin/kube-controller-manager
        sudo curl -L "https://dl.k8s.io/v1.30.0/bin/linux/arm64/kube-scheduler" -o kubebuilder/bin/kube-scheduler
        sudo chmod 755 kubebuilder/bin/kube-controller-manager
        sudo chmod 755 kubebuilder/bin/kube-scheduler
    fi
}

setup_configs() {
    # Generate certificates and tokens if they don't exist
    if [ ! -f "/tmp/sa.key" ]; then
        openssl genrsa -out /tmp/sa.key 2048
        openssl rsa -in /tmp/sa.key -pubout -out /tmp/sa.pub
    fi

    if [ ! -f "/tmp/token.csv" ]; then
        TOKEN="1234567890"
        echo "${TOKEN},admin,admin,system:masters" > /tmp/token.csv
    fi

    # Always regenerate and copy CA certificate to ensure it exists
    echo "Generating CA certificate..."
    openssl genrsa -out /tmp/ca.key 2048
    openssl req -x509 -new -nodes -key /tmp/ca.key -subj "/CN=kubelet-ca" -days 365 -out /tmp/ca.crt
    sudo mkdir -p /var/lib/kubelet/pki
    sudo cp /tmp/ca.crt /var/lib/kubelet/ca.crt
    sudo cp /tmp/ca.crt /var/lib/kubelet/pki/ca.crt

    # Create kubeconfig directory
    mkdir -p ~/.kube
    sudo mkdir -p /root/.kube

    # Set up kubeconfig
    export KUBECONFIG=~/.kube/config
    
    # Set up kubeconfig if not already configured
    if ! kubebuilder/bin/kubectl config current-context 2>/dev/null | grep -q "test-context"; then
        kubebuilder/bin/kubectl config set-credentials test-user --token=1234567890
        kubebuilder/bin/kubectl config set-cluster test-env --server=https://127.0.0.1:6443 --insecure-skip-tls-verify
        kubebuilder/bin/kubectl config set-context test-context --cluster=test-env --user=test-user --namespace=default 
        kubebuilder/bin/kubectl config use-context test-context
    fi
    
    # Copy to root for sudo operations
    sudo cp ~/.kube/config /root/.kube/config 2>/dev/null || true

    # Configure CNI
    cat <<EOF | sudo tee /etc/cni/net.d/10-mynet.conf
{
    "cniVersion": "0.3.1",
    "name": "mynet",
    "type": "bridge",
    "bridge": "cni0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.22.0.0/16",
        "routes": [
            { "dst": "0.0.0.0/0" }
        ]
    }
}
EOF

    # Configure containerd-k8s (separate from Docker's containerd)
    cat <<EOF | sudo tee /etc/containerd-k8s/config.toml
version = 3

[grpc]
  address = "/run/containerd-k8s/containerd.sock"

[plugins.'io.containerd.content.v1.content']
  path = "/var/lib/containerd-k8s/content"

[plugins.'io.containerd.metadata.v1.bolt']
  content_sharing_policy = "shared"

[plugins.'io.containerd.snapshotter.v1.native']
  root_path = "/var/lib/containerd-k8s/snapshots/native"

[plugins.'io.containerd.snapshotter.v1.overlayfs']
  root_path = "/var/lib/containerd-k8s/snapshots/overlayfs"

[plugins.'io.containerd.cri.v1.runtime']
  enable_selinux = false
  enable_unprivileged_ports = true
  enable_unprivileged_icmp = true
  device_ownership_from_security_context = false
  root = "/var/lib/containerd-k8s/cri"
  state = "/run/containerd-k8s/cri"

[plugins.'io.containerd.cri.v1.images']
  snapshotter = "native"
  disable_snapshot_annotations = true

[plugins.'io.containerd.cri.v1.runtime'.cni]
  bin_dir = "/opt/cni/bin"
  conf_dir = "/etc/cni/net.d"

[plugins.'io.containerd.cri.v1.runtime'.containerd]
  snapshotter = "native"
  default_runtime_name = "runc"

[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"

[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc.options]
  SystemdCgroup = false
  Root = "/var/lib/containerd-k8s/runc"
EOF

    # Ensure containerd-k8s data directory exists with correct permissions
    sudo mkdir -p /var/lib/containerd-k8s/{content,snapshots/{native,overlayfs},cri,runc}
    sudo mkdir -p /run/containerd-k8s/cri
    sudo chmod -R 755 /var/lib/containerd-k8s
    sudo chmod -R 755 /run/containerd-k8s

    # Configure kubelet
    cat << EOF | sudo tee /var/lib/kubelet/config.yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
authentication:
  anonymous:
    enabled: true
  webhook:
    enabled: true
  x509:
    clientCAFile: "/var/lib/kubelet/ca.crt"
authorization:
  mode: AlwaysAllow
clusterDomain: "cluster.local"
clusterDNS:
  - "10.0.0.10"
resolvConf: "/etc/resolv.conf"
runtimeRequestTimeout: "15m"
failSwapOn: false
seccompDefault: true
serverTLSBootstrap: false
containerRuntimeEndpoint: "unix:///run/containerd-k8s/containerd.sock"
staticPodPath: "/etc/kubernetes/manifests"
address: "0.0.0.0"
port: 10250
readOnlyPort: 10255
cgroupDriver: "cgroupfs"
maxPods: 10
registerNode: true
EOF

    # Create required directories with proper permissions
    sudo mkdir -p /var/lib/kubelet/pods
    sudo chmod 750 /var/lib/kubelet/pods
    sudo mkdir -p /var/lib/kubelet/plugins
    sudo chmod 750 /var/lib/kubelet/plugins
    sudo mkdir -p /var/lib/kubelet/plugins_registry
    sudo chmod 750 /var/lib/kubelet/plugins_registry

    # Ensure proper permissions
    sudo chmod 644 /var/lib/kubelet/ca.crt
    sudo chmod 644 /var/lib/kubelet/config.yaml

    # Generate self-signed kubelet serving certificate if not present
    if [ ! -f "/var/lib/kubelet/pki/kubelet.crt" ] || [ ! -f "/var/lib/kubelet/pki/kubelet.key" ]; then
        echo "Generating self-signed kubelet serving certificate..."
        sudo openssl req -x509 -newkey rsa:2048 -nodes \
            -keyout /var/lib/kubelet/pki/kubelet.key \
            -out /var/lib/kubelet/pki/kubelet.crt \
            -days 365 \
            -subj "/CN=$(hostname)"
        sudo chmod 600 /var/lib/kubelet/pki/kubelet.key
        sudo chmod 644 /var/lib/kubelet/pki/kubelet.crt
    fi
}

start() {
    if check_running; then
        echo "Kubernetes components are already running"
        return 0
    fi

    HOST_IP=$(hostname -I | awk '{print $1}')
    
    # Download components if needed
    download_components
    
    # Setup configurations
    setup_configs

    # Start components if not running
    if ! is_running "etcd"; then
        echo "Starting etcd..."
        sudo kubebuilder/bin/etcd \
            --advertise-client-urls http://$HOST_IP:2379 \
            --listen-client-urls http://0.0.0.0:2379 \
            --data-dir ./etcd \
            --listen-peer-urls http://0.0.0.0:2380 \
            --initial-cluster default=http://$HOST_IP:2380 \
            --initial-advertise-peer-urls http://$HOST_IP:2380 \
            --initial-cluster-state new \
            --initial-cluster-token test-token 2>&1 | tee -a /tmp/etcd.log &
        sleep 3
    fi

    if ! is_running "kube-apiserver"; then
        echo "Starting kube-apiserver..."
        sudo kubebuilder/bin/kube-apiserver \
            --etcd-servers=http://$HOST_IP:2379 \
            --service-cluster-ip-range=10.0.0.0/24 \
            --bind-address=0.0.0.0 \
            --secure-port=6443 \
            --advertise-address=$HOST_IP \
            --authorization-mode=AlwaysAllow \
            --token-auth-file=/tmp/token.csv \
            --enable-priority-and-fairness=false \
            --allow-privileged=true \
            --profiling=false \
            --storage-backend=etcd3 \
            --storage-media-type=application/json \
            --v=0 \
            --service-account-issuer=https://kubernetes.default.svc.cluster.local \
            --service-account-key-file=/tmp/sa.pub \
            --service-account-signing-key-file=/tmp/sa.key 2>&1 | tee -a /tmp/kube-apiserver.log &
        sleep 5
    fi

    # Start containerd-k8s (separate from Docker's containerd)
    if ! is_running "containerd-k8s"; then
        echo "Starting containerd-k8s..."
        export PATH=$PATH:/opt/cni/bin:kubebuilder/bin
        sudo PATH=$PATH:/opt/cni/bin:/usr/sbin /opt/cni/bin/containerd \
            -c /etc/containerd-k8s/config.toml \
            --root /var/lib/containerd-k8s \
            --state /run/containerd-k8s 2>&1 | tee -a /tmp/containerd-k8s.log &
        sleep 5
    fi

    # Wait for API server to be ready
    echo "Waiting for API server to be ready..."
    export KUBECONFIG=~/.kube/config
    for i in {1..30}; do
        if kubebuilder/bin/kubectl cluster-info 2>/dev/null | grep -q "running"; then
            echo "API server is ready"
            break
        fi
        echo "Waiting for API server... ($i/30)"
        sleep 2
    done

    # Create service account and configmap if they don't exist
    kubebuilder/bin/kubectl create sa default 2>/dev/null || true
    kubebuilder/bin/kubectl create configmap kube-root-ca.crt --from-file=ca.crt=/tmp/ca.crt -n default 2>/dev/null || true

    # Set up kubelet kubeconfig
    sudo cp ~/.kube/config /var/lib/kubelet/kubeconfig

    if ! is_running "kubelet"; then
        echo "Starting kubelet..."
        sudo PATH=$PATH:/opt/cni/bin:/usr/sbin kubebuilder/bin/kubelet \
            --kubeconfig=/var/lib/kubelet/kubeconfig \
            --config=/var/lib/kubelet/config.yaml \
            --root-dir=/var/lib/kubelet \
            --cert-dir=/var/lib/kubelet/pki \
            --tls-cert-file=/var/lib/kubelet/pki/kubelet.crt \
            --tls-private-key-file=/var/lib/kubelet/pki/kubelet.key \
            --hostname-override=$(hostname) \
            --pod-infra-container-image=registry.k8s.io/pause:3.10 \
            --node-ip=$HOST_IP \
            --cgroup-driver=cgroupfs \
            --register-node=true \
            --v=2 2>&1 | tee -a /tmp/kubelet.log &
        sleep 5
    fi

    if ! is_running "kube-controller-manager"; then
        echo "Starting kube-controller-manager..."
        sudo PATH=$PATH:/opt/cni/bin:/usr/sbin kubebuilder/bin/kube-controller-manager \
            --kubeconfig=/var/lib/kubelet/kubeconfig \
            --leader-elect=false \
            --allocate-node-cidrs=true \
            --cluster-cidr=10.22.0.0/16 \
            --service-cluster-ip-range=10.0.0.0/24 \
            --cluster-name=kubernetes \
            --root-ca-file=/var/lib/kubelet/ca.crt \
            --service-account-private-key-file=/tmp/sa.key \
            --use-service-account-credentials=true \
            --v=2 2>&1 | tee -a /tmp/kube-controller-manager.log &
        sleep 3
    fi

    if ! is_running "kube-scheduler"; then
        echo "Starting kube-scheduler..."
        sudo kubebuilder/bin/kube-scheduler \
            --kubeconfig=/var/lib/kubelet/kubeconfig \
            --leader-elect=false \
            --v=2 \
            --bind-address=0.0.0.0 2>&1 | tee -a /tmp/kube-scheduler.log &
        sleep 3
    fi

    echo "Waiting for all components to be ready..."
    sleep 10

    # Label the node so static pods with nodeSelector can be scheduled
    NODE_NAME=$(hostname)
    export KUBECONFIG=~/.kube/config
    kubebuilder/bin/kubectl label node "$NODE_NAME" node-role.kubernetes.io/master="" --overwrite || true

    echo "Verifying setup..."

    # Wait for node to register
    echo "Waiting for node to register..."
    for i in {1..30}; do
        if kubebuilder/bin/kubectl get nodes 2>/dev/null | grep -E "Ready|NotReady" | grep -v "NAME"; then
            echo "Node registered successfully"
            break
        fi
        echo "Waiting for node registration... ($i/30)"
        sleep 3
    done

    echo "Final status:"
    kubebuilder/bin/kubectl get nodes -o wide
    kubebuilder/bin/kubectl get all -A
    kubebuilder/bin/kubectl get componentstatuses 2>/dev/null || true
    kubebuilder/bin/kubectl get --raw='/readyz?verbose' || true
    
    # Check containerd-k8s status
    echo ""
    echo "Containerd-k8s status:"
    sudo /opt/cni/bin/ctr --address /run/containerd-k8s/containerd.sock version || echo "Failed to get containerd-k8s version"
}

stop() {
    echo "Stopping Kubernetes components..."
    stop_process "kube-controller-manager"
    stop_process "kubelet"
    stop_process "kube-scheduler"
    stop_process "kube-apiserver"
    stop_process "containerd-k8s"
    stop_process "etcd"
    echo "All components stopped"
}

cleanup() {
    stop
    echo "Cleaning up..."

    # Kill containerd shims and runc processes
    sudo pkill -f 'containerd-shim.*k8s\.io' 2>/dev/null || true
    sudo pkill -f 'runc --root /run/containerd-k8s' 2>/dev/null || true
    sleep 1
    
    # Find and unmount all nested mounts
    for base in /var/lib/kubelet/pods /run/containerd-k8s /var/lib/containerd-k8s; do
        if [ -d "$base" ]; then
            echo "Unmounting under $base ..."
            sudo grep "$base" /proc/mounts | awk '{print $2}' \
                 | sort -r | xargs -r -n1 sudo umount -l 2>/dev/null || true
        fi
    done
    
    # Additional cleanup for specific containerd-k8s mounts
    sudo umount -l /run/containerd-k8s/io.containerd.grpc.v1.cri/sandboxes/*/shm 2>/dev/null || true
    sudo umount -l /run/containerd-k8s/io.containerd.runtime.v2.task/*/rootfs 2>/dev/null || true
    sleep 2

    sudo rm -rf ./etcd
    sudo rm -rf /var/lib/kubelet/*
    sudo rm -rf /run/containerd-k8s/*
    sudo rm -rf /var/lib/containerd-k8s/*
    sudo rm -f /tmp/sa.key /tmp/sa.pub /tmp/token.csv /tmp/ca.key /tmp/ca.crt
    sudo rm -f /tmp/*.log
    echo "Cleanup complete"
}

case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    cleanup)
        cleanup
        ;;
    *)
        echo "Usage: $0 {start|stop|cleanup}"
        exit 1
        ;;
esac