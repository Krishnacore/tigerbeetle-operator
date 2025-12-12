# TigerBeetle Operator

A Kubernetes operator for managing [TigerBeetle](https://tigerbeetle.com/) clusters.

## Description

This operator automates the deployment and management of TigerBeetle database clusters on Kubernetes. It handles:

- StatefulSet creation with proper init containers for data file formatting
- Headless service for pod DNS discovery
- Persistent volume claims for data storage
- Pod anti-affinity and topology spread constraints
- Cluster lifecycle management

## Getting Started

### Prerequisites

- Kubernetes cluster v1.28+
- kubectl configured to access your cluster
- cert-manager (for webhook certificates, if using webhooks)

### Installation

1. Install the CRDs:

```bash
make install
```

2. Deploy the operator:

```bash
make deploy IMG=<your-registry>/tigerbeetle-operator:tag
```

### Create a TigerBeetle Cluster

```yaml
apiVersion: database.tigerbeetle.com/v1alpha1
kind: TigerBeetle
metadata:
  name: my-tigerbeetle
spec:
  clusterId: 0
  replicas: 3
  image:
    repository: ghcr.io/tigerbeetle/tigerbeetle
    tag: latest
  storage:
    size: 10Gi
  podAntiAffinity:
    type: soft
    topologyKey: kubernetes.io/hostname
```

Apply with:

```bash
kubectl apply -f config/samples/database_v1alpha1_tigerbeetle.yaml
```

## Configuration

### TigerBeetleSpec

| Field | Description | Default |
|-------|-------------|---------|
| `clusterId` | TigerBeetle cluster identifier | `0` |
| `replicas` | Number of replicas (1-6) | `3` |
| `image.repository` | Container image repository | `ghcr.io/tigerbeetle/tigerbeetle` |
| `image.tag` | Container image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `storage.size` | PVC storage size | `10Gi` |
| `storage.storageClassName` | StorageClass name | (default) |
| `resources` | Container resource limits/requests | (none) |
| `podAntiAffinity.type` | `soft`, `hard`, or empty | `soft` |
| `podAntiAffinity.topologyKey` | Anti-affinity topology key | `kubernetes.io/hostname` |
| `topologySpreadConstraints` | Pod topology spread config | (none) |

## Development

### Build

```bash
make build
```

### Run locally

```bash
make install
make run
```

### Run tests

```bash
make test
```

### Build and push Docker image

```bash
make docker-build docker-push IMG=<your-registry>/tigerbeetle-operator:tag
```

## License

Apache License 2.0
