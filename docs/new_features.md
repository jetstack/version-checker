# Kubernetes Version Monitoring

version-checker now includes built-in Kubernetes cluster version monitoring capabilities. This feature automatically compares your cluster's current Kubernetes version against the latest available versions from official Kubernetes release channels.

### How It Works

The Kubernetes version checker:
- Fetches the current cluster version using the Kubernetes Discovery API
- Compares it against the latest version from the configured Kubernetes release channel (using official `https://dl.k8s.io/release/` endpoints)
- Exposes the comparison as Prometheus metrics for monitoring and alerting
- Strips metadata from versions for accurate semantic version comparison (e.g., `v1.28.2-gke.1` becomes `v1.28.2`)

### Configuration

You can configure the Kubernetes version checking behavior using the following CLI flags:

- `--kube-channel`: Specifies which Kubernetes release channel to check against (default: `"stable"`)
  - Examples: `stable`, `latest`, `stable-1.28`, `latest-1.29`
- `--kube-interval`: How often to check for Kubernetes version updates (default: same as `--cache-sync-period`, 5 hours)

### Metrics

The Kubernetes version monitoring exposes the following Prometheus metric:

```
version_checker_is_latest_kube_version{current_version="1.28.2", latest_version="1.29.1", channel="stable"} 0
```

- Value `1`: Cluster is running the latest version from the specified channel
- Value `0`: Cluster is not running the latest version (update available)

### Supported Channels

version-checker uses official Kubernetes release channels:

- `stable` - Latest stable Kubernetes release (recommended)
- `latest` - Latest Kubernetes release (including pre-releases)
- `latest-1.28` - Latest patch for Kubernetes 1.28.x
- `latest-1.27` - Latest patch for Kubernetes 1.27.x

### Examples

```bash
# Check against latest stable Kubernetes
version-checker --kube-version-channel=stable

# Check against latest Kubernetes (including alpha/beta)
version-checker --kube-version-channel=latest

# Check against latest 1.28.x patch
version-checker --kube-version-channel=latest-1.28

# Monitor against a specific version channel with custom interval
./version-checker --kube-channel=stable-1.28 --kube-interval=1h
```

### Managed Kubernetes Support

Works with all managed Kubernetes services:
- **Amazon EKS**: Compares `v1.28.2-eks-abc123` against upstream `v1.28.2`
- **Google GKE**: Compares `v1.28.2-gke.1034000` against upstream `v1.28.2`  
- **Azure AKS**: Compares `v1.28.2-aks-xyz789` against upstream `v1.28.2`