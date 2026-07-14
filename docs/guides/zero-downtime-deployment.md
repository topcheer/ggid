# Zero-Downtime Deployment Guide

> GGID IAM Platform supports zero-downtime patches through rolling updates,
> health-gated deployments, and database migration strategies.

## Overview

Zero-downtime deployment ensures the platform remains available during
upgrades. GGID achieves this through:

1. **Multi-instance architecture** — All services are stateless and horizontally scalable
2. **Health-gated rolling updates** — New pods are only promoted when healthy
3. **Backward-compatible migrations** — Database changes are applied in two phases
4. **Graceful shutdown** — In-flight requests complete before termination

## Kubernetes Rolling Update

### Standard Deployment (recommended)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-auth
spec:
  replicas: 3                      # minimum 2 for HA
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0            # never reduce available pods
      maxSurge: 1                  # create 1 extra during update
  selector:
    matchLabels:
      app: ggid-auth
  template:
    spec:
      terminationGracePeriodSeconds: 60  # allow in-flight requests
      containers:
      - name: auth
        image: ggid/auth:latest
        readinessProbe:            # gates traffic routing
          httpGet:
            path: /readyz
            port: 9001
          initialDelaySeconds: 5
          periodSeconds: 5
        livenessProbe:             # restart unhealthy pods
          httpGet:
            path: /healthz
            port: 9001
          initialDelaySeconds: 10
          periodSeconds: 10
        lifecycle:
          preStop:                  # graceful drain
            exec:
              command: ["sleep", "10"]  # let load balancer deregister
```

### Update Process

```bash
# 1. Apply new image
kubectl set image deployment/ggid-auth auth=ggid/auth:v2.0.0

# 2. Watch rollout (blocks until all pods are healthy)
kubectl rollout status deployment/ggid-auth

# 3. Rollback if needed
kubectl rollout undo deployment/ggid-auth
```

### Rolling All Services

```bash
#!/bin/bash
# deploy/rolling-update.sh — zero-downtime update of all GGID services

SERVICES="gateway identity auth oauth policy org audit"

for svc in $SERVICES; do
    echo "=== Updating ggid-$svc ==="
    kubectl set image deployment/ggid-$svc $svc=ggid/$svc:$1
    kubectl rollout status deployment/ggid-$svc --timeout=300s

    if [ $? -ne 0 ]; then
        echo "ROLLBACK: ggid-$svc failed, rolling back"
        kubectl rollout undo deployment/ggid-$svc
        exit 1
    fi
done

echo "All services updated successfully"
```

## Docker Compose Rolling Update

```bash
# Pull new images
docker compose pull

# Recreate services one at a time
for svc in gateway identity auth oauth policy org audit; do
    docker compose up -d --no-deps $svc
    # Wait for healthcheck
    until [ "$(docker inspect --format='{{.State.Health.Status}}' ggid-$svc)" = "healthy" ]; do
        sleep 2
    done
done
```

## Database Migration Strategy

GGID uses **expand-contract** migrations to prevent downtime:

### Phase 1: Expand (backward-compatible)

```sql
-- Add new column (nullable, has default)
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enforced BOOLEAN DEFAULT false;

-- Add new table
CREATE TABLE IF NOT EXISTS access_requests (...);
```

**Deploy new code version** — old code ignores new columns/tables.

### Phase 2: Migrate

```sql
-- Backfill data
UPDATE users SET mfa_enforced = true WHERE tenant_id IN (
    SELECT id FROM tenants WHERE mfa_required = true
);
```

### Phase 3: Contract (after old version is retired)

```sql
-- Remove old column (only when no code references it)
ALTER TABLE users DROP COLUMN IF EXISTS mfa_required;
```

### Migration Command

```bash
# Run as init container (idempotent — skips if already applied)
kubectl apply -f deploy/k8s/migration-job.yaml

# Or standalone
./ggid-migrate --database-url=$DATABASE_URL --migrate-only
```

## Health Check Endpoints

Every GGID service exposes two health endpoints:

| Endpoint | Purpose | Used By |
|----------|---------|---------|
| `/healthz` | Liveness — process is running | Kubernetes liveness probe |
| `/readyz` | Readiness — service can handle requests | Kubernetes readiness probe |

The `/readyz` endpoint checks:
- Database connection is alive
- Redis connection (if applicable) is alive
- NATS connection (if applicable) is alive
- gRPC server is accepting connections

## Graceful Shutdown

GGID services handle `SIGTERM` by:

1. **Stop accepting new requests** — HTTP server calls `Shutdown()`
2. **Drain in-flight requests** — Wait up to 60s for completion
3. **Close connections** — Database, Redis, NATS, gRPC
4. **Exit cleanly** — Process exits with code 0

```go
// Already implemented in all services:
ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
defer cancel()
// ... server.Shutdown(ctx) on signal
```

## Blue-Green Deployment (advanced)

For maximum safety, use blue-green deployments:

```bash
# Deploy new version to green environment
kubectl apply -f deploy/k8s/green/ -l version=v2

# Wait for green to be healthy
kubectl wait --for=condition=ready pods -l version=v2 --timeout=300s

# Switch traffic (instant rollback capability)
kubectl patch service ggid-gateway -p '{"spec":{"selector":{"version":"v2"}}}'

# Keep blue running for quick rollback
# When confident, scale down blue:
kubectl scale deployment ggid-auth-v1 --replicas=0
```

## Monitoring During Deployment

```bash
# Watch all pods during rollout
kubectl get pods -w -l app.kubernetes.io/part-of=ggid

# Check service health
curl http://gateway:8080/healthz
curl http://gateway:8080/readyz

# Monitor error rate (should stay at 0)
kubectl logs -l app=ggid-gateway --tail=100 | grep -c "error"
```

## Checklist Before Deployment

- [ ] All tests pass (`make test`)
- [ ] Docker images built and pushed
- [ ] Database migrations are backward-compatible
- [ ] At least 2 replicas per service
- [ ] Health/readiness probes configured
- [ `terminationGracePeriodSeconds` >= 60
- [ ] Monitoring and alerting active
- [ ] Rollback plan documented

## Checklist After Deployment

- [ ] All pods healthy (`kubectl get pods`)
- [ ] No 5xx errors in gateway logs
- [ ] E2E test passes (`bash deploy/e2e-docker-test.sh`)
- [ ] Key metrics within normal range
- [ ] Old version scaled down (after observation period)
