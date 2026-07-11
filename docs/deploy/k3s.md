# K3s Deployment Guide

> Deploy GGID on K3s (lightweight Kubernetes) — OrbStack or standalone K3s with Traefik ingress.

---

## Prerequisites

- [OrbStack](https://orbstack.dev/) (macOS) or [K3s](https://k3s.io/) (Linux)
- Helm 3.12+
- kubectl
- Access to a container registry (e.g., `registry.iot2.win`)

---

## Step 1: K3s Cluster Setup

### OrbStack (macOS)

```bash
# OrbStack includes a K8s cluster
eval "$(orb stack kubectl env)"
kubectl get nodes
# → orbstack Ready control-plane
```

### Standalone K3s (Linux)

```bash
curl -sfL https://get.k3s.io | sh -

# Get kubeconfig
sudo cat /etc/rancher/k3s/k3s.yaml > ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config
kubectl get nodes
```

---

## Step 2: Build and Push Images

```bash
# Build all service images
cd /Users/zhanju/ggai/ggid
for svc in gateway identity auth oauth policy org audit; do
  docker build -f services/$svc/Dockerfile -t registry.iot2.win/ggid/$svc:latest .
  docker push registry.iot2.win/ggid/$svc:latest
done

# Build console
docker build -f console/Dockerfile -t registry.iot2.win/ggid/console:latest .
docker push registry.iot2.win/ggid/console:latest
```

---

## Step 3: Helm Deploy with values-k3s.yaml

```bash
# Create K3s-specific values
cat > deploy/helm/ggid/values-k3s.yaml <<'EOF'
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
EOF

# Install
helm install ggid deploy/helm/ggid \
  -f deploy/helm/ggid/values.yaml \
  -f deploy/helm/ggid/values-k3s.yaml \
  -n ggid --create-namespace
```

---

## Step 4: DNS and Access

### OrbStack

OrbStack auto-resolves `*.local.orbstack.dev` to the K3s cluster:

```bash
# Port-forward gateway
kubectl port-forward -n ggid svc/ggid-gid-gateway 8080:8080

curl http://localhost:8080/healthz
# → {"status":"ok"}
```

### External Domain

For `ggid.iot2.win`:

```bash
# Create Traefik IngressRoute or standard Ingress
kubectl apply -f - <<'EOF'
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ggid-ingress
  namespace: ggid
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
spec:
  ingressClassName: traefik
  rules:
  - host: ggid.iot2.win
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: ggid-gid-gateway
            port:
              number: 8080
EOF
```

---

## Step 5: E2E Verification

```bash
export GGID_URL="http://localhost:8080"
export TENANT="00000000-0000-0000-0000-000000000001"

# Health
curl $GGID_URL/healthz | jq .

# Register
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

# Protected endpoint
curl -s $GGID_URL/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `ErrImagePull` | Verify images pushed to registry, check `imageRegistry` in values |
| `CrashLoopBackOff` | `kubectl logs <pod>` — usually missing env var (DB_HOST, JWT_SECRET) |
| `Pending` pods | StorageClass mismatch — K3s uses `local-path` by default |
| 502/Connection refused | Service name mismatch — verify with `kubectl get svc -n ggid` |
| DB connection error | Policy/Org/Audit use `DB_HOST` not `DATABASE_URL` |

### Useful Commands

```bash
# Pod status
kubectl get pods -n ggid

# Service endpoints
kubectl get svc -n ggid

# Detailed pod info
kubectl describe pod <pod-name> -n ggid

# Logs
kubectl logs -n ggid deploy/ggid-gid-gateway -f
kubectl logs -n ggid deploy/ggid-gid-auth -f

# Uninstall
helm uninstall ggid -n ggid
```

---

*Last updated: 2025-07-11*