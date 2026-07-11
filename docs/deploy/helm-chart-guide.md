# Helm Chart Deployment Guide

> Deploy GGID to Kubernetes using the official Helm chart. Covers installation, configuration, upgrades, and rollback.

---

## Prerequisites

- Kubernetes 1.28+ (or K3s)
- Helm 3.12+
- `kubectl` configured
- Container registry with GGID images

---

## Chart Structure

```
deploy/helm/ggid/
├── Chart.yaml          # Chart metadata (v0.1.0, appVersion 0.1.0)
├── values.yaml         # Default configuration
└── templates/
    ├── _helpers.tpl    # Template helpers
    ├── configmap.yaml  # Service config
    ├── deployments.yaml # 7 service deployments
    ├── hpa.yaml         # Horizontal Pod Autoscaler
    ├── ingress.yaml     # Ingress controller
    ├── networkpolicy.yaml # Network isolation
    ├── pdb.yaml         # Pod Disruption Budget
    ├── secrets.yaml     # JWT secrets, DB passwords
    └── services.yaml    # ClusterIP services
```

---

## Installation

### 1. Create Namespace

```bash
kubectl create namespace ggid
```

### 2. Install the Chart

```bash
helm install ggid deploy/helm/ggid \
  --namespace ggid \
  --set global.imageRegistry=registry.iot2.win/ggid \
  --set jwt.secret=$(openssl rand -base64 32)
```

### 3. Verify Deployment

```bash
kubectl get pods -n ggid
# NAME                          READY   STATUS    RESTARTS
# ggid-gateway-xxx              1/1     Running   0
# ggid-identity-xxx             1/1     Running   0
# ggid-auth-xxx                 1/1     Running   0
# ggid-policy-xxx               1/1     Running   0
# ggid-org-xxx                  1/1     Running   0
# ggid-audit-xxx                1/1     Running   0
# ggid-postgresql-xxx           1/1     Running   0
# ggid-redis-xxx                1/1     Running   0
# ggid-nats-xxx                 1/1     Running   0
```

---

## Configuration (values.yaml)

### Global Settings

| Key | Default | Description |
|-----|---------|-------------|
| `global.imageRegistry` | `""` | Container registry prefix |
| `global.imagePullSecrets` | `[]` | Registry auth secrets |
| `global.storageClass` | `""` | StorageClass for PVCs |

### Infrastructure

| Key | Default | Description |
|-----|---------|-------------|
| `postgresql.enabled` | `true` | Deploy bundled PostgreSQL 16 |
| `postgresql.auth.password` | `ggid` | DB password (change in production!) |
| `postgresql.primary.persistence.size` | `20Gi` | DB volume size |
| `redis.enabled` | `true` | Deploy bundled Redis 7 |
| `redis.auth.password` | `ggid-redis` | Redis password |
| `nats.enabled` | `true` | Deploy bundled NATS 2 (JetStream) |

### Using External Infrastructure

```bash
helm install ggid deploy/helm/ggid \
  --namespace ggid \
  --set postgresql.enabled=false \
  --set externalDatabase.host=prod-db.internal \
  --set externalDatabase.password=prod-secret \
  --set redis.enabled=false \
  --set externalRedis.host=prod-redis.internal \
  --set nats.enabled=false \
  --set externalNats.host=prod-nats.internal
```

### Service Configuration

Each service (gateway, identity, auth, policy, org, audit) supports:

| Key | Default | Description |
|-----|---------|-------------|
| `enabled` | `true` | Enable/disable the service |
| `replicaCount` | `2` | Number of replicas (1 for audit) |
| `image.repository` | `ggid/<svc>` | Container image |
| `image.tag` | `latest` | Image tag |
| `service.type` | `ClusterIP` | Service type |
| `service.port` | `8080` | HTTP port |
| `resources.limits.cpu` | `500m` | CPU limit |
| `resources.limits.memory` | `256Mi` | Memory limit |

### Gateway-Specific

| Key | Default | Description |
|-----|---------|-------------|
| `gateway.autoscaling.enabled` | `true` | Enable HPA |
| `gateway.autoscaling.minReplicas` | `2` | Minimum pods |
| `gateway.autoscaling.maxReplicas` | `10` | Maximum pods |
| `gateway.autoscaling.targetCPUUtilizationPercentage` | `70` | Scale-up threshold |
| `gateway.podDisruptionBudget.enabled` | `true` | Enable PDB |
| `gateway.networkPolicy.enabled` | `true` | Enable NetworkPolicy |

### JWT Configuration

```yaml
jwt:
  secret: ""  # Auto-generated if empty (use --set for production)
```

### Ingress

```yaml
ingress:
  enabled: true
  className: nginx  # or traefik for K3s
  hosts:
    - host: iam.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: ggid-tls
      hosts:
        - iam.example.com
```

---

## Production Values Override

Create `values-prod.yaml`:

```yaml
global:
  imageRegistry: "registry.iot2.win/ggid"
  storageClass: "fast-ssd"

postgresql:
  auth:
    password: "${POSTGRES_PASSWORD}"
  primary:
    persistence:
      size: 100Gi

redis:
  auth:
    password: "${REDIS_PASSWORD}"

gateway:
  replicaCount: 3
  autoscaling:
    minReplicas: 3
    maxReplicas: 20

identity:
  replicaCount: 3

auth:
  replicaCount: 3

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: iam.prod.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: ggid-prod-tls
      hosts:
        - iam.prod.example.com
```

Install:

```bash
helm install ggid deploy/helm/ggid \
  -f deploy/helm/ggid/values.yaml \
  -f values-prod.yaml \
  --namespace ggid
```

---

## Upgrade

```bash
# Upgrade with new image tags
helm upgrade ggid deploy/helm/ggid \
  --namespace ggid \
  --set global.imageRegistry=registry.iot2.win/ggid \
  --set gateway.image.tag=v1.1.0 \
  --set identity.image.tag=v1.1.0 \
  --set auth.image.tag=v1.1.0

# Upgrade with new values file
helm upgrade ggid deploy/helm/ggid \
  -f deploy/helm/ggid/values.yaml \
  -f values-prod.yaml \
  --namespace ggid
```

### Rolling Update Strategy

All deployments use `RollingUpdate` strategy:

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0  # Zero downtime
```

---

## Rollback

```bash
# View release history
helm history ggid -n ggid
# REVISION  STATUS      DESCRIPTION
# 1         superseded  Install complete
# 2         deployed    Upgrade complete

# Rollback to previous revision
helm rollback ggid 1 -n ggid

# Rollback automatically rolls back all services
```

---

## Uninstall

```bash
helm uninstall ggid -n ggid

# Clean up PVCs (WARNING: deletes all data)
kubectl delete pvc -n ggid --all
```

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `ImagePullBackOff` | Check `global.imageRegistry` and registry credentials |
| `CrashLoopBackOff` | `kubectl logs <pod> -n ggid` — usually missing env or DB unreachable |
| Pending pods | Check `storageClass` exists; default uses `standard` |
| `Pending` PVC | StorageClass not available — use `local-path` for K3s |

---

*See: [Kubernetes Guide](kubernetes.md) | [K3s Deploy](../quickstart/k3s-deploy.md) | [Helm Reference](helm-reference.md)*

*Last updated: 2025-07-11*
