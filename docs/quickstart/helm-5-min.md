# Helm 5-Minute Quickstart

> Deploy GGID to Kubernetes with Helm in 5 minutes.

---
## Prerequisites

- Kubernetes cluster (K3s, EKS, GKE, AKS, or local)
- `kubectl` configured
- `helm` 3.8+

---

## 1. Add Namespace

```bash
kubectl create namespace ggid
```

## 2. Deploy

```bash
helm install ggid deploy/helm/ggid \
  --namespace ggid \
  --set global.domain=localhost:8080 \
  --set global.imageRegistry=ghcr.io/ggid
```

## 3. Wait for Pods

```bash
kubectl -n ggid rollout status deploy/ggid-gateway --timeout=120s
kubectl -n ggid get pods
```

**Expected:** All pods Running.

```
NAME                            READY   STATUS    AGE
ggid-gateway-xxx                1/1     Running   2m
ggid-identity-xxx               1/1     Running   2m
ggid-auth-xxx                   1/1     Running   2m
ggid-oauth-xxx                  1/1     Running   2m
ggid-policy-xxx                 1/1     Running   2m
ggid-org-xxx                    1/1     Running   2m
ggid-audit-xxx                  1/1     Running   2m
ggid-console-xxx                1/1     Running   2m
postgresql-xxx                  1/1     Running   2m
redis-xxx                       1/1     Running   2m
nats-xxx                        1/1     Running   2m
```

## 4. Port Forward

```bash
kubectl -n ggid port-forward svc/ggid-gateway 8080:8080 &
```

## 5. Test

```bash
# Health check
curl http://localhost:8080/healthz

# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"t@t.com","password":"Test1234!"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"Test1234!"}' | jq -r .access_token)

echo "JWT: ${#TOKEN} chars"
```

---

## Using External Database

```bash
helm install ggid deploy/helm/ggid \
  --namespace ggid \
  --set postgresql.enabled=false \
  --set redis.enabled=false \
  --set nats.enabled=false \
  --set externalDatabase.host=prod-db.internal \
  --set externalDatabase.password=secret \
  --set externalRedis.host=prod-redis.internal \
  --set externalNats.url=nats://prod-nats:4222
```

---

## Upgrade

```bash
helm upgrade ggid deploy/helm/ggid --namespace ggid --set global.imageTag=v1.1.0
```

## Rollback

```bash
helm rollback ggid 1 --namespace ggid
```

---

## Values File

Create `values.yaml` for complex configs:

```yaml
global:
  domain: iam.example.com
  imageRegistry: ghcr.io/ggid
  imageTag: latest

ingress:
  enabled: true
  tls: true

policy:
  replicaCount: 2

gateway:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
```

```bash
helm install ggid deploy/helm/ggid --namespace ggid -f values.yaml
```

---

*See: [Helm Chart Guide](../deploy/helm-chart-guide.md) | [Helm Reference](../deploy/helm-reference.md) | [Docker Quickstart](docker-5-min.md)*

*Last updated: 2025-07-11*
