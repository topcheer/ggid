# GGID Deployment Guide

Production deployment instructions for the GGID IAM Platform.

## Table of Contents

- [Docker Compose (Recommended for Small Deployments)](#docker-compose)
- [Environment Variables](#environment-variables)
- [Database Initialization](#database-initialization)
- [TLS Configuration](#tls-configuration)
- [Kubernetes (Helm)](#kubernetes-helm)
- [Backup Strategy](#backup-strategy)
- [Monitoring](#monitoring)
- [Security Hardening Checklist](#security-hardening-checklist)

---

## Docker Compose

The simplest production deployment uses Docker Compose with the provided
`deploy/docker-compose.yaml`.

### Quick Deploy

```bash
cd deploy

# 1. Create a production .env file (see Environment Variables below)
cp .env.example .env
# Edit .env with production secrets

# 2. Start all services
docker compose up -d

# 3. Verify health
curl http://localhost:8080/healthz/ready
```

### Architecture

```
                    ┌──────────────┐
   Clients ────────►│   Gateway    │:8080
                    │  (JWT verify) │
                    └──────┬───────┘
           ┌───────┬───────┼───────┬────────┐
           ▼       ▼       ▼       ▼        ▼
        ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐
        │Auth │ │Ident│ │Policy│ │ Org │ │Audit│
        │:9001│ │:8081│ │:8070│ │:8071│ │:8072│
        └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘
           │       │       │       │       │
           ▼       ▼       ▼       ▼       ▼
        ┌─────────────────────────────────────┐
        │         PostgreSQL 16 (:5432)        │
        └─────────────────────────────────────┘
                    │                  │
           ┌────────▼───┐    ┌────────▼───┐
           │ Redis 7    │    │ NATS JS    │
           │ (:6379)    │    │ (:4222)    │
           └────────────┘    └────────────┘
```

### Scaling Individual Services

Edit `deploy/docker-compose.yaml` and add `deploy.replicas`:

```yaml
services:
  auth:
    deploy:
      replicas: 3
```

Or use `docker compose up --scale auth=3`.

> The Gateway load-balances across all auth replicas via Docker's DNS round-robin.

---

## Environment Variables

### Gateway

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_ADDR` | `:8080` | Listen address |
| `JWT_PUBLIC_KEY_PATH` | `configs/rsa_public.pem` | RSA public key for JWT verification |
| `GATEWAY_JWKS_URL` | _(empty = use local key)_ | JWKS endpoint URL |
| `GATEWAY_JWT_ISSUER` | `ggid-auth` | Expected JWT issuer |
| `AUTH_SERVICE_URL` | `http://auth:9001` | Auth backend URL |
| `IDENTITY_SERVICE_URL` | `http://identity:8080` | Identity backend URL |
| `POLICY_SERVICE_URL` | `http://policy:8070` | Policy backend URL |
| `ORG_SERVICE_URL` | `http://org:8071` | Org backend URL |
| `AUDIT_SERVICE_URL` | `http://audit:8072` | Audit backend URL |
| `OAUTH_SERVICE_URL` | `http://oauth:9005` | OAuth backend URL |

### Auth Service

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | _(required)_ | PostgreSQL connection string |
| `REDIS_ADDR` | `redis:6379` | Redis address for rate limiting |
| `AUTH_HTTP_ADDR` | `:9001` | HTTP listen address |
| `JWT_PRIVATE_KEY_PATH` | `/configs/rsa_private.pem` | RSA private key for signing JWTs |
| `JWT_PUBLIC_KEY_PATH` | `/configs/rsa_public.pem` | RSA public key |
| `LDAP_URL` | _(empty = disabled)_ | LDAP server URL |
| `LDAP_BIND_DN` | — | LDAP bind DN |
| `LDAP_BIND_PASSWORD` | — | LDAP bind password |
| `LDAP_BASE_DN` | — | LDAP search base DN |
| `LDAP_USER_FILTER` | — | LDAP user filter template |
| `LDAP_AUTO_PROVISION` | `false` | Auto-create users on first LDAP login |

### Identity Service

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | _(required)_ | PostgreSQL connection string |

### Policy / Org / Audit Services

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `postgres` | Database host |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `ggid` | Database user |
| `DB_PASSWORD` | _(required)_ | Database password |
| `DB_DATABASE` | `ggid` | Database name |
| `DB_SSL_MODE` | `disable` | PostgreSQL SSL mode |
| `NATS_URL` | `nats://nats:4222` | NATS connection URL |
| `POLICY_HTTP_ADDR` / `ORG_HTTP_ADDR` / `AUDIT_HTTP_ADDR` | `:807x` | HTTP listen address |

### OAuth Service

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | _(required)_ | PostgreSQL connection string |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `OAUTH_PRIVATE_KEY_PATH` | `/configs/rsa_private.pem` | RSA private key |
| `OAUTH_PUBLIC_KEY_PATH` | `/configs/rsa_public.pem` | RSA public key |

### Infrastructure

| Variable | Description |
|----------|-------------|
| `POSTGRES_USER` | PostgreSQL superuser |
| `POSTGRES_PASSWORD` | PostgreSQL password (**change in production**) |
| `POSTGRES_DB` | Default database name |

---

## Database Initialization

Migrations are applied automatically by the `migrate` init container.

### Manual Migration

```bash
# Run migrations manually
docker compose run --rm migrate

# Or with psql
psql "postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable" \
  -f deploy/migrations/001_init.sql
```

### Migration Files

| File | Description |
|------|-------------|
| `001_init.sql` | Base schema: tenants, users, credentials, roles |
| `002_orgs.sql` | Organizations, departments, teams, memberships |
| `003_audit.sql` | Audit events table + indexes |
| `004_rls.sql` | Row-Level Security policies |

### Non-Superuser for Production

Docker Compose uses a superuser (bypasses RLS). For production:

```sql
-- Create a limited role
CREATE ROLE ggid_app WITH LOGIN PASSWORD 'strong-password';
GRANT CONNECT ON DATABASE ggid TO ggid_app;
GRANT USAGE ON SCHEMA public TO ggid_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ggid_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ggid_app;

-- Ensure RLS is enforced (not bypassed)
ALTER ROLE ggid_app NOBYPASSRLS;
```

Update `DATABASE_URL` / `DB_USER` / `DB_PASSWORD` to use `ggid_app`.

---

## TLS Configuration

### Option 1: Reverse Proxy (Recommended)

Use nginx, Caddy, or Traefik as a TLS-terminating reverse proxy in front
of the Gateway:

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name iam.example.com;

    ssl_certificate     /etc/ssl/certs/iam.crt;
    ssl_certificate_key /etc/ssl/private/iam.key;

    location / {
        proxy_pass http://gateway:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Option 2: Caddy (Automatic TLS)

```Caddyfile
iam.example.com {
    reverse_proxy gateway:8080
}
```

Caddy automatically provisions Let's Encrypt certificates.

### Option 3: Internal TLS

For service-to-service TLS, configure each service to listen on HTTPS:

```bash
# Set TLS cert/key paths via environment
TLS_CERT_PATH=/certs/service.crt
TLS_KEY_PATH=/certs/service.key
```

> Internal TLS requires certificate management (cert-manager in Kubernetes,
> or mutual TLS via a service mesh like Linkerd/Istio).

---

## Kubernetes Helm

### Prerequisites

- Kubernetes 1.28+
- Helm 3.14+
- cert-manager (for TLS)
- Ingress controller (nginx-ingress or Traefik)

### Install

```bash
# Add the GGID Helm repository
helm repo add ggid https://charts.ggid.dev
helm repo update

# Install with production values
helm install ggid ggid/ggid \
  --namespace ggid-system \
  --create-namespace \
  -f values-production.yaml
```

### values-production.yaml

```yaml
global:
  domain: iam.example.com

# Gateway
gateway:
  replicas: 2
  ingress:
    enabled: true
    className: nginx
    tls:
      enabled: true
      certManager: true
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi

# Auth service
auth:
  replicas: 2
  env:
    LDAP_URL: "ldap://ldap.ggid-system.svc:389"
  resources:
    requests:
      cpu: 100m
      memory: 128Mi

# Identity service
identity:
  replicas: 2

# Policy service
policy:
  replicas: 2

# PostgreSQL
postgresql:
  enabled: true
  primary:
    persistence:
      size: 50Gi
      storageClass: fast-ssd
  auth:
    postgresPassword: "change-me-in-production"

# Redis
redis:
  enabled: true
  architecture: replication
  master:
    persistence:
      size: 8Gi

# NATS
nats:
  enabled: true
  jetstream:
    enabled: true
    fileStore:
      enabled: true
      size: 10Gi
```

### Verify Deployment

```bash
kubectl get pods -n ggid-system
kubectl get ingress -n ggid-system
curl https://iam.example.com/healthz
```

---

## Backup Strategy

### PostgreSQL

**Daily logical backup:**

```bash
#!/bin/bash
# backup-pg.sh — run via cron daily
DATE=$(date +%Y%m%d_%H%M%S)
docker exec ggid-postgres pg_dump -U ggid ggid | gzip > /backups/ggid_${DATE}.sql.gz

# Retention: keep 30 days
find /backups -name "ggid_*.sql.gz" -mtime +30 -delete
```

**Continuous WAL archiving (for PITR):**

```yaml
# postgresql.conf
archive_mode = on
archive_command = 'aws s3 cp %p s3://ggid-wal-archive/%f'
```

### Redis

Redis is used for rate limiting and session cache. No persistent backup needed —
data is ephemeral and reconstructable.

### NATS JetStream

JetStream data is ephemeral (audit events are persisted to PostgreSQL by the
audit consumer). No backup needed — if NATS restarts, it re-establishes the
stream and continues consuming.

### RSA Key Pair

**Critical:** Back up the RSA key pair used for JWT signing.

```bash
# Copy keys from the config volume
docker cp ggid-auth:/configs/rsa_private.pem /secure-backup/
docker cp ggid-auth:/configs/rsa_public.pem /secure-backup/
```

> Store these in a secrets manager (HashiCorp Vault, AWS Secrets Manager).
> Losing the private key invalidates all issued JWTs.

---

## Monitoring

### Health Checks

| Endpoint | Type | Description |
|----------|------|-------------|
| `/healthz` | Basic | Gateway is running |
| `/healthz/live` | Liveness | Process is alive (no backend check) |
| `/healthz/ready` | Readiness | All backend services healthy |

### Prometheus Metrics

```yaml
# Prometheus scrape config
scrape_configs:
  - job_name: ggid-gateway
    static_configs:
      - targets: ['gateway:8080']
    metrics_path: /metrics
```

Key metrics exposed at `/metrics`:
- Request count, latency histogram (per route)
- Error rate (4xx/5xx)
- JWT verification count and failure rate
- Proxy backend latency

### Grafana Dashboard

Recommended panels:
- Request rate by service
- p50/p95/p99 latency
- Error rate (4xx vs 5xx)
- Active sessions
- Failed login attempts (security monitoring)

### Logging

All services log to stdout in JSON format:

```json
{"level":"info","msg":"login success","user":"admin","ip":"10.0.0.1","ts":"2024-01-15T10:30:00Z"}
```

Forward to ELK/Loki/Datadog via Docker logging driver or Fluent Bit.

---

## Security Hardening Checklist

- [ ] **Change default passwords** — PostgreSQL, LDAP, Redis
- [ ] **Generate fresh RSA key pair** — do not use development keys
- [ ] **Enable TLS** — terminate at ingress or reverse proxy
- [ ] **Use non-superuser DB role** — create `ggid_app` with `NOBYPASSRLS`
- [ ] **Configure CORS** — restrict origins to your frontend domains
- [ ] **Set up rate limiting** — auth service default is 5 attempts per IP
- [ ] **Enable audit logging** — verify NATS and audit consumer are running
- [ ] **Review LDAP config** — disable auto-provision if not needed
- [ ] **Network policies** — restrict inter-service communication (Kubernetes)
- [ ] **Secrets management** — use Kubernetes Secrets or external secret store
- [ ] **Regular security updates** — rebuild Docker images with latest base
- [ ] **Backup RSA keys** — store in Vault/Secrets Manager
- [ ] **Configure retention** — set audit log retention (default: 90 days)
- [ ] **Monitor failed logins** — alert on spike in 401 responses
