# K3s Quick Deploy

> Get GGID running on K3s in under 10 minutes. For the full guide with troubleshooting, see [K3s Deployment Guide](../deploy/k3s.md).

---

## Prerequisites

- [K3s](https://k3s.io/) or [OrbStack](https://orbstack.dev/) (macOS)
- `kubectl` configured
- Container registry access (or build locally)

---

## 1. Start K3s

**Linux:**

```bash
curl -sfL https://get.k3s.io | sh -
sudo cat /etc/rancher/k3s/kubeconfig > ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config
```

**macOS (OrbStack):**

```bash
eval "$(orb stack kubectl env)"
```

Verify:

```bash
kubectl get nodes
# → orbstack  Ready  control-plane
```

---

## 2. Build and Push Images

```bash
cd /path/to/ggid

# Build all services
for svc in gateway identity auth oauth policy org audit; do
  docker build -f services/$svc/Dockerfile \
    -t registry.iot2.win/ggid/$svc:latest .
  docker push registry.iot2.win/ggid/$svc:latest
done

# Build console
docker build -f console/Dockerfile \
  -t registry.iot2.win/ggid/console:latest .
docker push registry.iot2.win/ggid/console:latest
```

> If using a different registry, replace `registry.iot2.win/ggid` with your registry path.

---

## 3. Deploy with Helm

```bash
# Create K3s-specific values
cat > deploy/helm/ggid/values-k3s.yaml <<'EOF'
global:
  imageRegistry: "registry.iot2.win/ggid"
  storageClass: "local-path"

gateway:
  service:
    type: NodePort
    nodePort: 30080

ingress:
  enabled: true
  className: "traefik"
EOF

# Install
helm install ggid deploy/helm/ggid \
  -f deploy/helm/ggid/values.yaml \
  -f deploy/helm/ggid/values-k3s.yaml \
  -n ggid --create-namespace
```

---

## 4. Verify Deployment

```bash
# Check pods are running
kubectl get pods -n ggid

# Expected: all pods Running or Completed
# NAME                              READY   STATUS
# ggid-gateway-xxx                  1/1     Running
# ggid-identity-xxx                 1/1     Running
# ggid-auth-xxx                     1/1     Running
# ggid-policy-xxx                   1/1     Running
# ggid-org-xxx                      1/1     Running
# ggid-audit-xxx                    1/1     Running
# ggid-oauth-xxx                    1/1     Running
# ggid-console-xxx                  1/1     Running
# ggid-postgresql-xxx               1/1     Running
# ggid-redis-xxx                    1/1     Running
# ggid-nats-xxx                     1/1     Running
```

---

## 5. Access the Gateway

```bash
# Port-forward to access locally
kubectl port-forward -n ggid svc/ggid-gateway 8080:8080 &

# Health check
curl http://localhost:8080/healthz
# → {"status":"ok"}
```

---

## 6. Quick E2E Test

```bash
export GGID_URL="http://localhost:8080"
export TENANT="00000000-0000-0000-0000-000000000001"

# Register a user
curl -s -X POST $GGID_URL/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"k3stest","email":"k3s@test.com","password":"Kr3sSecure!2025"}' | jq .

# Login
JWT=$(curl -s -X POST $GGID_URL/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"k3stest","password":"Kr3sSecure!2025"}' | jq -r .access_token)

echo "JWT: ${JWT:0:20}..."

# List users
curl -s $GGID_URL/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq '.users[0].username'
# → "k3stest"
```

---

## Common Issues

| Problem | Fix |
|---------|-----|
| `ErrImagePull` | Verify images pushed to registry; check `imageRegistry` in values |
| `CrashLoopBackOff` | Check `kubectl logs <pod> -n ggid` — usually missing env vars |
| DB connection error | Policy/Org/Audit use `DB_HOST` not `DATABASE_URL` |
| `Pending` pods | StorageClass mismatch — K3s uses `local-path` by default |

---

## Teardown

```bash
helm uninstall ggid -n ggid
kubectl delete namespace ggid
```

---

*Full guide: [K3s Deployment](../deploy/k3s.md) | [Docker Quick Deploy](./docker-5-min.md) | [Kubernetes Guide](../deploy/kubernetes.md)*

*Last updated: 2025-07-11*
