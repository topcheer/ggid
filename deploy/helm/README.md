# GGID Helm Chart

Production-ready Helm chart for deploying GGID (Global Governance Identity Dashboard) to Kubernetes.

## Quick Start

```bash
# Install with defaults
helm install ggid deploy/helm/ggid/

# Install with custom values
helm install ggid deploy/helm/ggid/ -f my-values.yaml

# Install in a namespace
helm install ggid deploy/helm/ggid/ -n ggid --create-namespace

# Dry-run to see what will be deployed
helm template deploy/helm/ggid/ > rendered.yaml
```

## Configuration

### Core Settings

| Key | Default | Description |
|-----|---------|-------------|
| `global.imageRegistry` | `""` | Global image registry override |
| `global.imagePullSecrets` | `[]` | List of image pull secrets |
| `global.storageClass` | `""` | Default storage class for PVCs |

### Infrastructure

| Key | Default | Description |
|-----|---------|-------------|
| `postgresql.enabled` | `true` | Deploy bundled PostgreSQL |
| `postgresql.auth.password` | `ggid` | PostgreSQL password (change in production!) |
| `redis.enabled` | `true` | Deploy bundled Redis |
| `nats.enabled` | `true` | Deploy bundled NATS |

### Service Images

Each service (`gateway`, `identity`, `auth`, `oauth`, `policy`, `org`, `audit`) supports:

| Key | Description |
|-----|-------------|
| `.enabled` | Enable/disable the service |
| `.image.repository` | Container image |
| `.image.tag` | Image tag |
| `.replicaCount` | Number of replicas |
| `.service.type` | Kubernetes Service type |
| `.service.httpPort` | HTTP port |
| `.resources` | CPU/memory requests and limits |

### Security

| Key | Default | Description |
|-----|---------|-------------|
| `networkPolicy.enabled` | `true` | Restrict inter-pod traffic |
| `jwt.secret` | `""` | JWT signing secret (auto-generated if empty) |

### External Databases

Set `postgresql.enabled=false` / `redis.enabled=false` / `nats.enabled=false` and configure:

```yaml
externalDatabase:
  host: "prod-db.example.com"
  port: 5432
  username: "ggid"
  password: "secure-password"
  database: "ggid"

externalRedis:
  host: "prod-redis.example.com"
  port: 6379
  password: "secure-password"

externalNats:
  url: "nats://prod-nats:4222"
```

## Upgrade

```bash
helm upgrade ggid deploy/helm/ggid/ -f my-values.yaml
```

## Uninstall

```bash
helm uninstall ggid
```

## K3s / Local Development

```bash
helm install ggid deploy/helm/ggid/ -f deploy/helm/ggid/values-k3s.yaml
```
