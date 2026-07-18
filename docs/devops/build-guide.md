# GGID DevOps Build Guide

Build, deploy, and troubleshoot GGID Docker images.

---

## Critical: Platform Flag

**Mac Studio (arm64) → K8s (amd64)**: Always specify `--platform linux/amd64` when building for deployment.

```bash
# WRONG — builds arm64 on Mac Studio, fails on amd64 K8s nodes
docker build -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .

# CORRECT — builds amd64 regardless of host architecture
docker build --platform linux/amd64 -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .
```

CI (`.github/workflows/ci.yml`) and Makefile targets already include `--platform linux/amd64`.

---

## Dockerfiles

| Service | Dockerfile | Image Name |
|---------|-----------|------------|
| All-in-one | `deploy/all-in-one/Dockerfile` | `ggid/ggid-all-in-one` |
| Console (Next.js) | `console/Dockerfile` | `ggid/ggid-console` |
| Auth | `services/auth/Dockerfile` | `ggid/ggid-auth` |
| Identity | `services/identity/Dockerfile` | `ggid/ggid-identity` |
| Gateway | `services/gateway/Dockerfile` | `ggid/ggid-gateway` |
| OAuth | `services/oauth/Dockerfile` | `ggid/ggid-oauth` |
| Policy | `services/policy/Dockerfile` | `ggid/ggid-policy` |
| Audit | `services/audit/Dockerfile` | `ggid/ggid-audit` |
| Org | `services/org/Dockerfile` | `ggid/ggid-org` |

---

## Build Commands

### All-in-One (Single Container)

```bash
# Build
make docker-build-allinone

# Tag + push to registry
make docker-push

# Manual build with platform flag
docker build --platform linux/amd64 -f deploy/all-in-one/Dockerfile \
  -t ggid/ggid-all-in-one:latest .

# Tag for remote registry
docker tag ggid/ggid-all-in-one:latest registry.iot2.win/ggid/all-in-one:latest

# Push
docker push registry.iot2.win/ggid/all-in-one:latest
```

### Console

```bash
make docker-build
# or
docker build --platform linux/amd64 -f console/Dockerfile \
  -t ggid/ggid-console:latest .
```

### Individual Services

```bash
# Example: auth service
docker build --platform linux/amd64 -f services/auth/Dockerfile \
  -t ggid/ggid-auth:latest .
```

---

## K8s Deployment

### Restart Pods After Image Update

```bash
# Rollout restart (zero downtime)
kubectl rollout restart deployment ggid-all-in-one -n ggid

# Check rollout status
kubectl rollout status deployment ggid-all-in-one -n ggid

# View pods
kubectl get pods -n ggid
```

### Pull Latest Image

If K8s uses `imagePullPolicy: Always` or the tag is `latest`:

```bash
# Force pull new image
kubectl delete pods -l app=ggid-all-in-one -n ggid
```

---

## Troubleshooting

### Image runs locally but crashes on K8s

**Cause**: Architecture mismatch (arm64 image on amd64 node)

**Fix**: Rebuild with `--platform linux/amd64`

```bash
docker build --platform linux/amd64 -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .
```

### `exec format error` in container logs

**Cause**: Same architecture mismatch

**Fix**: Same as above

### Pod stuck in `ImagePullBackOff`

**Cause**: Wrong image name or registry auth

**Fix**:
```bash
kubectl describe pod <pod-name> -n ggid
# Check Events section for pull error details
```

### Build is slow on Mac

**Cause**: QEMU emulation for cross-platform builds

**Fix**: Use Docker Buildx with cache:
```bash
docker buildx build --platform linux/amd64 \
  --cache-from type=local,src=/tmp/.docker-cache \
  --cache-to type=local,dest=/tmp/.docker-cache \
  -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .
```

---

## CI/CD

The CI pipeline (`.github/workflows/ci.yml`) includes:

1. **Lint gate**: golangci-lint v1.64.8 (pre-build)
2. **Mod tidy check**: fails if go.mod/go.sum not clean
3. **Build**: `go build ./...`
4. **Test**: `make test` with coverage
5. **Console TS check**: `npx tsc --noEmit`
6. **Docker build** (periodic): `platforms: linux/amd64`

---

*Last updated: 2025-07-18*
