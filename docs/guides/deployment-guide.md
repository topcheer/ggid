# GGID Deployment Guide

> Complete guide for deploying GGID in development and production environments.

## Table of Contents

1. [Docker All-in-One Deployment](#1-docker-all-in-one-deployment)
2. [k3s Production Deployment](#2-k3s-production-deployment)
3. [Console Build & Deploy](#3-console-build--deploy)
4. [Environment Variables](#4-environment-variables)
5. [Database Migrations](#5-database-migrations)
6. [Health Checks & Troubleshooting](#6-health-checks--troubleshooting)

---

## 1. Docker All-in-One Deployment

The simplest deployment — all services in a single container via supervisord.

### Prerequisites

- Docker 27+ with BuildKit
- 4GB RAM minimum

### Build

```bash
cd deploy/all-in-one
docker build --platform linux/amd64 -t ggid/all-in-one:latest .
```

### Run

```bash
docker run -d \
  --name ggid \
  -p 8080:8080 \
  -p 3000:3000 \
  -e DATABASE_URL="postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable" \
  -e REDIS_ADDR="localhost:6379" \
  -e INTERNAL_AUTH_SECRET="$(openssl rand -hex 32)" \
  -e PASSWORD_PEPPER="$(openssl rand -hex 32)" \
  ggid/all-in-one:latest
```

### Ports

| Port | Service |
|------|---------|
| 8080 | API Gateway |
| 3000 | Console (Next.js) |
| 9001 | Auth Service |
| 8081 | Identity Service |
| 9005 | OAuth Service |
| 8070 | Policy Service |
| 8072 | Audit Service |

### Docker Compose (Multi-Container)

```bash
cd deploy
docker compose -f docker-compose.yaml up -d
```

This starts PostgreSQL, Redis, and all GGID services as separate containers.

---

## 2. k3s Production Deployment

### Prerequisites

- k3s cluster (v1.31+)
- `kubectl` configured
- Container registry access (registry.iot2.win)

### Deploy All Services

```bash
export KUBECONFIG=~/.kube/config.k3s

# Apply manifests
kubectl apply -f deploy/k8s/ -n ggid

# Verify pods
kubectl get pod -n ggid
```

### Expected Pods

```
ggid-gateway     Running   1/1
ggid-auth        Running   1/1
ggid-identity    Running   1/1
ggid-oauth       Running   1/1
ggid-policy      Running   1/1
ggid-audit       Running   1/1
ggid-console     Running   1/1
ggid-postgres    Running   1/1
ggid-redis       Running   1/1
```

### Ingress

GGID uses nginx ingress controller (bundled with k3s). TLS is terminated at the ingress layer.

Domains:
- `ggid.iot2.win` — API Gateway + Console (SPA served from gateway)
- `ggid-console.iot2.win` — Console (direct access)
- `erp-api.iot2.win`, `erp-go.iot2.win`, `erp-java.iot2.win`, `erp-python.iot2.win` — ERP test instances

---

## 3. Console Build & Deploy

The console is a Next.js standalone application built into a Docker image.

### Build

```bash
cd /Users/zhanju/ggai/ggid
docker build --platform linux/amd64 -f console/Dockerfile -t registry.iot2.win/ggid/console:latest .
```

### Push

```bash
docker push registry.iot2.win/ggid/console:latest
```

### Deploy

```bash
export KUBECONFIG=~/.kube/config.k3s

# Rolling restart (pulls new image)
kubectl rollout restart deployment/ggid-console -n ggid
kubectl rollout status deployment/ggid-console -n ggid --timeout=120s

# Or delete pod (forces immediate recreation)
kubectl delete pod -n ggid -l app=ggid-console
```

### Build Individual Backend Services

```bash
# Example: Auth service
docker build --platform linux/amd64 -f services/auth/Dockerfile -t registry.iot2.win/ggid/auth:latest .
docker push registry.iot2.win/ggid/auth:latest
kubectl rollout restart deployment/ggid-auth -n ggid
```

Service Dockerfiles:

| Service | Dockerfile |
|---------|-----------|
| Gateway | `services/gateway/Dockerfile` |
| Auth | `services/auth/Dockerfile` |
| Identity | `services/identity/Dockerfile` |
| OAuth | `services/oauth/Dockerfile` |
| Policy | `services/policy/Dockerfile` |
| Audit | `services/audit/Dockerfile` |
| Org | `services/org/Dockerfile` |
| Console | `console/Dockerfile` |

---

## 4. Environment Variables

### Core

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_ADDR` | Yes | Redis address (e.g., `localhost:6379`) |
| `NATS_URL` | No | NATS messaging URL (optional, falls back to direct gRPC) |
| `INTERNAL_AUTH_SECRET` | Yes | Secret for inter-service auth (min 32 bytes) |
| `PASSWORD_PEPPER` | Yes | Pepper for password hashing (min 32 bytes) |

### OAuth

| Variable | Required | Description |
|----------|----------|-------------|
| `OAUTH_PRIVATE_KEY_PATH` | No | Path to OAuth signing private key |
| `OAUTH_PUBLIC_KEY_PATH` | No | Path to OAuth signing public key |

### gRPC

| Variable | Description |
|----------|-------------|
| `GRPC_TLS_ENABLED` | Enable TLS for gRPC |
| `GRPC_TLS_CERT` | Path to TLS certificate |
| `GRPC_TLS_KEY` | Path to TLS key |
| `GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK` | Allow plaintext gRPC in dev |

### LDAP (Optional)

| Variable | Description |
|----------|-------------|
| `LDAP_URL` | LDAP server URL |
| `LDAP_BIND_DN` | Bind DN |
| `LDAP_BIND_PASSWORD` | Bind password |
| `LDAP_BASE_DN` | Base DN for user search |
| `LDAP_USER_FILTER` | User search filter |
| `LDAP_START_TLS` | Enable StartTLS |
| `LDAP_AUTO_PROVISION` | Auto-provision users on first login |

### Key Management

| Variable | Description |
|----------|-------------|
| `GGID_KEY_PROVIDER` | Key provider: `local`, `aws-kms`, `gcp-kms`, `azure-kv`, `vault`, `pkcs11` |
| `GGID_ENCRYPTION_KEY` | Local KEK material (when provider=local) |

---

## 5. Database Migrations

GGID services use `pgxpool` with `EnsureSchema()` for automatic schema management. Most services create their tables on startup.

### Running Migrations

#### Automatic (on startup)

Services run `EnsureSchema()` on boot — tables are created if they don't exist.

#### Manual (migration-only mode)

```bash
# Run audit migrations only
docker exec ggid-audit /app/audit-service -migrate-only

# Or from source
go run ./services/audit/cmd/ -migrate-only
```

#### SQL Migration Files

Formal migrations are in `services/*/migrations/`:

```bash
# Apply identity migrations
psql "$DATABASE_URL" -f services/identity/migrations/000001_initial_schema.up.sql
psql "$DATABASE_URL" -f services/identity/migrations/004_scim_groups.sql

# Apply oauth migrations
psql "$DATABASE_URL" -f services/oauth/migrations/000001_initial_schema.up.sql
```

### Rollback

```bash
psql "$DATABASE_URL" -f services/oauth/migrations/000001_initial_schema.down.sql
```

---

## 6. Health Checks & Troubleshooting

### Health Endpoints

Each service exposes:

```bash
# Health check
curl -k https://ggid.iot2.win/healthz

# Readiness check
curl -k https://ggid.iot2.win/readyz

# Metrics (Prometheus)
curl -k https://ggid.iot2.win/metrics
```

### Verify All Services

```bash
TOKEN=$(curl -sk -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123456","tenant_id":"00000000-0000-0000-0000-000000000001"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))")

# Core endpoints
for ep in users roles audit/events; do
  echo "$ep: $(curl -sk -o /dev/null -w '%{http_code}' "https://ggid.iot2.win/api/v1/$ep" -H "Authorization: Bearer $TOKEN")"
done

# ERP instances
for erp in erp-api erp-go erp-java erp-python; do
  echo "$erp: $(curl -sk -o /dev/null -w '%{http_code}' "https://$erp.iot2.win/health")"
done
```

### Pod Troubleshooting

```bash
export KUBECONFIG=~/.kube/config.k3s

# Check pod status
kubectl get pod -n ggid

# View logs
kubectl logs -n ggid deploy/ggid-auth --tail=50
kubectl logs -n ggid deploy/ggid-gateway --tail=50

# Exec into pod
kubectl exec -it -n ggid deploy/ggid-auth -- /bin/sh

# Check resource usage
kubectl top pod -n ggid
```

### Common Issues

| Problem | Cause | Solution |
|---------|-------|--------|
| Service won't start | DATABASE_URL incorrect or DB unreachable | Verify PostgreSQL is running; check connection string |
| 401 on all API calls | Token expired or JWT signing key mismatch | Verify `OAUTH_PRIVATE_KEY_PATH`; re-login |
| Console 404 on pages | Console image not rebuilt after frontend changes | Rebuild console: `docker build -f console/Dockerfile ...` then redeploy |
| Migrations fail | Database version conflict or permissions | Check DB user has CREATE TABLE permissions; run migrations manually |
| gRPC connection refused | Service not started or wrong port | Check service is running: `kubectl get pod`; verify gRPC port |
| High memory usage | Large audit event backlog | Check audit service logs; increase pod memory limits |

### Performance Tuning

```bash
# Check database connections
kubectl exec -it -n ggid deploy/ggid-postgres -- \
  psql -U ggid -c "SELECT count(*) FROM pg_stat_activity;"

# Redis memory
kubectl exec -it -n ggid deploy/ggid-redis -- redis-cli info memory

# Pod resource limits
kubectl get pod -n ggid -o custom-columns=NAME:.metadata.name,CPU:.spec.containers[0].resources.requests.cpu,MEM:.spec.containers[0].resources.requests.memory
```
