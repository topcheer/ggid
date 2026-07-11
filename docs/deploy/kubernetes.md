# Kubernetes / Helm Deployment Guide

> Deploy GGID to Kubernetes using the Helm chart: installation, configuration, TLS, ingress, autoscaling.

---

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Kubernetes | 1.28+ |
| Helm | 3.12+ |
| kubectl | Latest |
| Ingress controller | nginx-ingress or traefik |
| cert-manager | v1.13+ (for TLS) |
| StorageClass | default or specified |

---

## Quick Install

```bash
# Add GGID Helm repo (when published)
helm repo add ggid https://charts.ggid.dev
helm repo update

# Or use local chart
helm install ggid deploy/helm/ggid
```

### Verify Installation

```bash
kubectl get pods -n ggid
# All pods should be Running within 60 seconds

kubectl get svc -n ggid
# gateway ClusterIP service on port 8080
```

---

## values.yaml Configuration

### Global Settings

| Key | Default | Description |
|-----|---------|-------------|
| `global.imageRegistry` | `""` | Override image registry (e.g., `ghcr.io/ggid`) |
| `global.imagePullSecrets` | `[]` | List of secret names for private registries |
| `global.storageClass` | `""` | Default StorageClass for PVCs |

### PostgreSQL

| Key | Default | Description |
|-----|---------|-------------|
| `postgresql.enabled` | `true` | Deploy bundled PostgreSQL |
| `postgresql.auth.username` | `ggid` | DB username |
| `postgresql.auth.password` | `ggid` | DB password (override in production!) |
| `postgresql.auth.database` | `ggid` | Database name |
| `postgresql.primary.persistence.size` | `20Gi` | PVC size |
| `postgresql.params` | `[max_connections=200, shared_buffers=256MB]` | PostgreSQL tuning |

### Redis

| Key | Default | Description |
|-----|---------|-------------|
| `redis.enabled` | `true` | Deploy bundled Redis |
| `redis.auth.enabled` | `true` | Require password |
| `redis.auth.password` | `ggid-redis` | Redis password |
| `redis.master.persistence.size` | `5Gi` | PVC size |

### NATS

| Key | Default | Description |
|-----|---------|-------------|
| `nats.enabled` | `true` | Deploy bundled NATS |
| `nats.jetstream.enabled` | `true` | Enable JetStream for audit pipeline |
| `nats.jetstream.storage.size` | `10Gi` | JetStream storage PVC |

### Per-Service Configuration

Each service (gateway, auth, identity, oauth, policy, org, audit) supports:

| Key | Default | Description |
|-----|---------|-------------|
| `.{service}.enabled` | `true` | Enable/disable service |
| `.{service}.replicaCount` | `2` (gateway), `1` (others) | Number of replicas |
| `.{service}.image.repository` | `ggid/{service}` | Container image |
| `.{service}.image.tag` | `latest` | Image tag |
| `.{service}.image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `.{service}.resources.limits.cpu` | `500m` | CPU limit |
| `.{service}.resources.limits.memory` | `256Mi` | Memory limit |
| `.{service}.resources.requests.cpu` | `100m` | CPU request |
| `.{service}.resources.requests.memory` | `128Mi` | Memory request |
| `.{service}.env` | `{}` | Additional environment variables |

### Autoscaling (HPA)

```yaml
gateway:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    # targetMemoryUtilizationPercentage: 80
```

---

## Custom values.yaml

Create `my-values.yaml`:

```yaml
# Production configuration
global:
  storageClass: "gp3"

postgresql:
  auth:
    password: "super-secure-password"
  primary:
    persistence:
      size: 100Gi
  params:
    - max_connections=500
    - shared_buffers=1GB

redis:
  auth:
    password: "redis-secure-password"

gateway:
  replicaCount: 3
  autoscaling:
    maxReplicas: 20
  resources:
    limits:
      cpu: 1000m
      memory: 512Mi

auth:
  env:
    JWT_SECRET: "your-production-jwt-secret"
    LDAP_URL: "ldap://ldap.corporate.internal:389"
    LDAP_BIND_DN: "cn=ggid,dc=corp,dc=com"
    LDAP_BIND_PASSWORD: "ldap-service-password"
    LDAP_BASE_DN: "dc=corp,dc=com"
    LDAP_USER_FILTER: "(sAMAccountName=%s)"

audit:
  env:
    AUDIT_RETENTION_DAYS: "180"
```

### Install with Custom Values

```bash
helm install ggid deploy/helm/ggid \
  -f my-values.yaml \
  -n ggid --create-namespace
```

---

## Ingress Configuration

```yaml
# Add to values.yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
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

## TLS with cert-manager

```bash
# Install cert-manager
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  -n cert-manager --create-namespace \
  --set installCRDs=true

# Create ClusterIssuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF
```

---

## Resource Recommendations

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit | Min Replicas |
|---------|------------|-----------|---------------|-------------|-------------|
| Gateway | 100m | 500m | 128Mi | 256Mi | 2 |
| Auth | 100m | 500m | 128Mi | 256Mi | 2 |
| Identity | 100m | 500m | 128Mi | 256Mi | 1 |
| OAuth | 100m | 300m | 128Mi | 256Mi | 1 |
| Policy | 50m | 200m | 64Mi | 128Mi | 1 |
| Org | 50m | 200m | 64Mi | 128Mi | 1 |
| Audit | 50m | 200m | 64Mi | 128Mi | 1 |
| Console | 100m | 300m | 128Mi | 256Mi | 1 |
| PostgreSQL | 250m | 1000m | 256Mi | 1Gi | 1 |
| Redis | 50m | 200m | 64Mi | 128Mi | 1 |
| NATS | 50m | 200m | 64Mi | 128Mi | 1 |

---

## Upgrading

```bash
# Pull latest chart
helm repo update

# Upgrade
helm upgrade ggid deploy/helm/ggid -f my-values.yaml -n ggid

# Check rollout status
kubectl rollout status deployment/ggid-gateway -n ggid
```

---

## Troubleshooting

```bash
# Pod not starting
kubectl describe pod <pod-name> -n ggid
kubectl logs <pod-name> -n ggid

# Service not reachable
kubectl get svc -n ggid
kubectl exec -it <pod> -n ggid -- curl http://gateway:8080/healthz

# Database connection issues
kubectl exec -it ggid-postgres-0 -n ggid -- psql -U ggid -c "\dt"

# Check PVCs
kubectl get pvc -n ggid
```

---

*Last updated: 2025-07-11*