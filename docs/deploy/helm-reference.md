# Helm Values Reference

> Complete reference for every value in `deploy/helm/ggid/values.yaml`.

---

## Global

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `global.imageRegistry` | string | `""` | Override container image registry (e.g., `ghcr.io/ggid`) |
| `global.imagePullSecrets` | list | `[]` | List of Kubernetes Secret names for private registry auth |
| `global.storageClass` | string | `""` | Default StorageClass for all PVCs |

## PostgreSQL

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `postgresql.enabled` | bool | `true` | Deploy bundled PostgreSQL instance |
| `postgresql.image.registry` | string | `docker.io` | Image registry |
| `postgresql.image.repository` | string | `postgres` | Image repository |
| `postgresql.image.tag` | string | `16-alpine` | Image tag |
| `postgresql.auth.username` | string | `ggid` | Database user |
| `postgresql.auth.password` | string | `ggid` | Database password (override in production!) |
| `postgresql.auth.database` | string | `ggid` | Database name |
| `postgresql.primary.persistence.size` | string | `20Gi` | PVC size |
| `postgresql.params` | list | `[max_connections=200, shared_buffers=256MB]` | PostgreSQL tuning parameters |

## Redis

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `redis.enabled` | bool | `true` | Deploy bundled Redis |
| `redis.image.tag` | string | `7-alpine` | Image tag |
| `redis.auth.enabled` | bool | `true` | Enable password authentication |
| `redis.auth.password` | string | `ggid-redis` | Redis password |
| `redis.master.persistence.size` | string | `5Gi` | PVC size |

## NATS

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `nats.enabled` | bool | `true` | Deploy bundled NATS |
| `nats.image.tag` | string | `2-alpine` | Image tag |
| `nats.jetstream.enabled` | bool | `true` | Enable JetStream for audit pipeline |
| `nats.jetstream.storage.size` | string | `10Gi` | JetStream storage PVC |

## Gateway

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `gateway.enabled` | bool | `true` | Enable gateway service |
| `gateway.image.repository` | string | `ggid/gateway` | Container image |
| `gateway.image.tag` | string | `latest` | Image tag |
| `gateway.image.pullPolicy` | string | `IfNotPresent` | Pull policy |
| `gateway.replicaCount` | int | `2` | Number of replicas |
| `gateway.service.type` | string | `ClusterIP` | Service type |
| `gateway.service.port` | int | `8080` | Service port |
| `gateway.resources.limits.cpu` | string | `500m` | CPU limit |
| `gateway.resources.limits.memory` | string | `256Mi` | Memory limit |
| `gateway.resources.requests.cpu` | string | `100m` | CPU request |
| `gateway.resources.requests.memory` | string | `128Mi` | Memory request |
| `gateway.env` | map | `{}` | Additional environment variables |
| `gateway.autoscaling.enabled` | bool | `true` | Enable HPA |
| `gateway.autoscaling.minReplicas` | int | `2` | Minimum replicas |
| `gateway.autoscaling.maxReplicas` | int | `10` | Maximum replicas |
| `gateway.autoscaling.targetCPUUtilizationPercentage` | int | `70` | CPU target for scale-up |

## Auth / Identity / OAuth / Policy / Org / Audit

Each service follows the same structure as Gateway with service-specific defaults:

| Service | Default Replicas | CPU Limit | Memory Limit |
|---------|-----------------|-----------|-------------|
| auth | 1 | 500m | 256Mi |
| identity | 1 | 500m | 256Mi |
| oauth | 1 | 300m | 256Mi |
| policy | 1 | 200m | 128Mi |
| org | 1 | 200m | 128Mi |
| audit | 1 | 200m | 128Mi |

## Console

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `console.enabled` | bool | `true` | Enable admin console |
| `console.image.repository` | string | `ggid/console` | Container image |
| `console.service.port` | int | `3000` | Service port |
| `console.env.NEXT_PUBLIC_GGID_URL` | string | — | Gateway URL for API calls |

## Ingress

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `ingress.enabled` | bool | `false` | Enable ingress |
| `ingress.className` | string | `nginx` | Ingress controller class |
| `ingress.annotations` | map | `{}` | Annotations (e.g., cert-manager issuer) |
| `ingress.hosts[].host` | string | — | Hostname |
| `ingress.hosts[].paths[].path` | string | `/` | URL path |
| `ingress.hosts[].paths[].pathType` | string | `Prefix` | Path type |
| `ingress.tls[].secretName` | string | — | TLS secret name |
| `ingress.tls[].hosts` | list | — | TLS hostnames |

## values-k3s.yaml Example

```yaml
global:
  imageRegistry: "registry.iot2.win/ggid"
  storageClass: "local-path"

gateway:
  image:
    pullPolicy: Always
  service:
    type: NodePort
    nodePort: 30080

ingress:
  enabled: true
  className: "traefik"
```

---

*Last updated: 2025-07-11*