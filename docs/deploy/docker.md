# Docker Deployment Guide

> Complete guide to deploying GGID with Docker Compose: prerequisites, configuration, health checks, E2E test, and troubleshooting.

---

## Prerequisites

| Requirement | Minimum | Recommended |
|-------------|---------|------------|
| Docker | 24.0+ | Latest |
| Docker Compose | v2.20+ | Latest |
| RAM | 4 GB | 8 GB |
| Disk | 10 GB | 50 GB |
| Ports | 8080, 5432, 6379, 4222, 8222, 3000 | Same |

---

## Quick Start

```bash
# Clone
mkdir -p deploy

# Start all services
cd deploy && docker compose up -d

# Wait for healthchecks (30-60 seconds)
sleep 30

# Verify all containers are healthy
docker compose ps
```

### Expected Output

All 12 containers should show `Up (healthy)`:

| Container | Image | Port(s) | Healthcheck |
|-----------|-------|---------|-------------|
| ggid-postgres | postgres:16-alpine | 5432 | pg_isready |
| ggid-redis | redis:7-alpine | 6379 | redis-cli ping |
| ggid-nats | nats:2-alpine | 4222, 8222 | wget /healthz |
| ggid-ldap | osixia/openldap:1.5.0 | 389, 636 | — |
| ggid-gateway | ggid-gateway:latest | 8080 | curl /healthz |
| ggid-identity | ggid-identity:latest | 8081 | curl /healthz |
| ggid-auth | ggid-auth:latest | 9001 | curl /healthz |
| ggid-oauth | ggid-oauth:latest | 9005 | curl /healthz |
| ggid-policy | ggid-policy:latest | 8070, 9070 | curl /healthz |
| ggid-org | ggid-org:latest | 8071, 9071 | curl /healthz |
| ggid-audit | ggid-audit:latest | 8072, 9072 | curl /healthz |
| ggid-console | ggid-console:latest | 3000 | curl / |

---

## docker-compose.yaml Explained

### Infrastructure Layer

```yaml
# PostgreSQL 16 — primary database with RLS
postgres:
  image: postgres:16-alpine
  environment:
    POSTGRES_USER: ggid
    POSTGRES_PASSWORD: ggid
    POSTGRES_DB: ggid
  volumes:
    - ggid-pgdata:/var/lib/postgresql/data  # persistent data

# Redis 7 — sessions, rate limiting, JTI anti-replay
redis:
  image: redis:7-alpine

# NATS 2 — JetStream audit event pipeline
nats:
  image: nats:2-alpine
  command: ["-js", "-m", "8222"]  # JetStream + monitoring

# OpenLDAP — optional, for LDAP auth
ldap:
  image: osixia/openldap:1.5.0
```

### Service Layer

Each GGID service follows the same pattern:

```yaml
gateway:
  build:
    context: ..
    dockerfile: services/gateway/Dockerfile
  ports:
    - "8080:8080"
  depends_on:
    postgres:
      condition: service_healthy
    redis:
      condition: service_healthy
  environment:
    DATABASE_URL: "postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable"
    REDIS_URL: "redis://redis:6379"
    JWT_SECRET: "change-me-in-production"
    # ... service-specific env vars
```

### Database Migration

An init container runs migrations idempotently:

```yaml
migrate:
  image: ggid-migrate:latest
  depends_on:
    postgres:
      condition: service_healthy
  environment:
    DATABASE_URL: "postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable"
  # Runs: psql -f 01_all_up.sql; psql -f 02_add_webauthn.sql; ...
  # Skips if tables already exist (idempotent)
```

---

## Environment Variables

### Required (All Services)

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `REDIS_URL` | — | Redis connection string |
| `JWT_SECRET` | — | HMAC-SHA256 signing key (MUST set in production) |
| `TENANT_ID` | `00000000-...001` | Default tenant UUID |

### Service-Specific

| Service | Key Variables |
|---------|---------------|
| Gateway | `NATS_URL`, `JWKS_REFRESH_INTERVAL`, `LOG_LEVEL` |
| Auth | `LDAP_URL`, `LDAP_BIND_DN`, `LDAP_BIND_PASSWORD`, `LDAP_BASE_DN`, `LDAP_USER_FILTER`, `LDAP_START_TLS`, `LDAP_AUTO_PROVISION` |
| OAuth | `OAUTH_ISSUER`, `OAUTH_CODE_TTL`, `OAUTH_ACCESS_TOKEN_TTL` |
| Policy | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` |
| Org | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` |
| Audit | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `NATS_URL` |

> **Note**: Policy, Org, and Audit use individual `DB_*` env vars, NOT `DATABASE_URL`.

---

## Health Checks

```bash
# Quick health check
curl http://localhost:8080/healthz
# → {"status":"ok"}

# Deep health check (checks all dependencies)
curl http://localhost:8080/healthz/deep
# → {"status":"ok","postgres":"ok","redis":"ok","nats":"ok"}

# NATS monitoring
curl http://localhost:8222/healthz

# Console
open http://localhost:3000
```

---

## E2E Test Walkthrough

```bash
# Run the Docker E2E test script
bash deploy/e2e-docker-test.sh
```

The script verifies:

| Test | Expected | Description |
|------|----------|-------------|
| Gateway healthz | 200 | Gateway responds |
| Register user | 201 | POST /api/v1/auth/register |
| Login + JWT | 200 + token | POST /api/v1/auth/login |
| 401 without JWT | 401 | Protected endpoint blocks unauthenticated |
| List users | 200 | GET /api/v1/users with JWT |
| Create role | 201 | POST /api/v1/roles (requires `key` field) |
| List roles | 200 | GET /api/v1/roles |
| Create org | 201 | POST /api/v1/orgs |
| Audit query | 200 | GET /api/v1/audit/events |
| Wrong password | 401 | Login with bad credentials |
| Duplicate register | 409 | Same username again |

**Expected result**: `11/11 ALL PASS`

---

## Building Images

```bash
# Build all images
cd deploy && docker compose build

# Build specific service
docker compose build gateway

# Images are named: ggid-gateway, ggid-auth, ggid-identity, etc.
```

### Image Sizes

| Image | Size | Base |
|-------|------|------|
| ggid-gateway | 18.3 MB | scratch |
| ggid-auth | 27.4 MB | scratch |
| ggid-identity | 31.8 MB | scratch |
| ggid-oauth | 23.6 MB | scratch |
| ggid-policy | 34.3 MB | scratch |
| ggid-org | 34.3 MB | scratch |
| ggid-audit | 34.2 MB | scratch |
| ggid-console | 212 MB | node:20-alpine |

---

## Production Configuration

Use `docker-compose.prod.yaml`:

```bash
docker compose -f docker-compose.prod.yaml up -d
```

Production differences:
- TLS enabled
- Stronger passwords/secrets
- Resource limits enforced
- No debug logging
- Volume backups configured

### Override File

Create `docker-compose.override.yml` for local customizations:

```yaml
services:
  gateway:
    environment:
      LOG_LEVEL: debug
    ports:
      - "8080:8080"
      - "40000:40000"  # Delve debug port
```

---

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| Container won't start | Port conflict | `lsof -i :8080` to find conflicting process |
| Migration fails | DB not ready | Wait 30s, restart migrate container |
| 401 on all requests | JWT secret mismatch | Ensure JWT_SECRET is same across all services |
| 503 from gateway | Backend down | `docker logs ggid-auth` to check backend |
| Auth rate limited (429) | Too many login attempts | Restart auth container: `docker restart ggid-auth` |
| Register returns 409 | Username exists | Use different username |
| Create role returns 500 | Empty `key` field | Provide unique `key` value |
| Audit returns 404 | Route mismatch | Use `/api/v1/audit` (alias) not `/api/v1/audit/events` |
| NATS unhealthy | Missing `-m 8222` | Ensure NATS command includes `-m 8222` for monitoring |
| Policy/Org/Audit DB error | Wrong env format | These services use `DB_HOST` not `DATABASE_URL` |

### Useful Commands

```bash
# View logs
docker compose logs -f gateway
docker compose logs -f auth

# Restart a service
docker compose restart gateway

# Shell into a container
docker compose exec postgres psql -U ggid

# Clean start (warning: deletes data)
docker compose down -v
docker compose up -d
```

---

## Default Tenant

All Docker deployments include a default tenant:

```
Tenant ID: 00000000-0000-0000-0000-000000000001
```

All API requests must include the `X-Tenant-ID` header with this UUID.

---

*Last updated: 2025-07-11*