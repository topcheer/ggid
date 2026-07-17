# All-in-One Docker Deployment Guide

*Updated: 2026-07-15 — IPv6 fix, multi-tenant login, onboarding APIs*

## Summary

GGID provides an all-in-one Docker image that bundles the entire IAM stack into a single container. This is ideal for **local development, demos, and single-node evaluations** where you want to run PostgreSQL, Redis, NATS, all 7 backend services, and the admin console with one command.

## One-Command Quick Start

The easiest way to start is with the included launcher script:

```bash
bash deploy/all-in-one/run.sh
```

This script will:
1. Build the Docker image if it doesn't exist yet
2. Remove any stale container
3. Start the container with all ports bound to `127.0.0.1` (IPv4 only)
4. Wait for services to initialize (~20 seconds)
5. Run a health check and print the access URLs

Alternatively, you can build and run manually:

```bash
# Build the image from repository root
docker build -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .

# Run with only gateway + console ports exposed (security: P0 fix Round 91)
# Backend service ports (8081/9001/9005/8070/8071/8072) are NOT exposed
# externally. All traffic must go through the gateway at :8080.
docker run -d \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:3000:3000 \
  --name ggid-all-in-one \
  ggid/ggid-all-in-one:latest
```

After ~20 seconds, access the platform:

- **Admin Console:** http://127.0.0.1:3000
- **API Gateway:** http://127.0.0.1:8080
- **Default credentials:** `admin` / `Password123!`
- **Tenant slug:** `default`

> **Important — use `127.0.0.1`, not `localhost`:** On macOS, Docker port forwarding may resolve `localhost` to an IPv6 address (`::1`) that the container doesn't bind to. Always use `127.0.0.1` in browser URLs and API calls to avoid connection refused errors.

## Default Credentials

The all-in-one image seeds the following default data during initialization:

| Item | Value |
|------|-------|
| Admin username | `admin` |
| Admin password | `Password123!` |
| Admin email | `admin@ggid.dev` |
| Tenant name | `Default` |
| Tenant slug | `default` |
| Tenant ID | `00000000-0000-0000-0000-000000000001` |
| System roles | `admin`, `manager`, `user` |

**Change the admin password immediately after first login in any non-demo environment.**

## IPv6 Fix (2026-07-15)

### Problem

On macOS, Docker's port forwarding may resolve `localhost` to `::1` (IPv6 loopback) while the container's services listen on `0.0.0.0` or `127.0.0.1` (IPv4 only). This caused connection refused errors when services tried to communicate internally via HTTP using `localhost` URLs.

### Fix

All service-to-service HTTP URLs in the Dockerfile have been changed from `localhost` to `127.0.0.1`:

```dockerfile
# Before (IPv6 issue on macOS)
ENV GATEWAY_URL=http://localhost:8080
ENV IDENTITY_SERVICE_URL=http://localhost:8081
ENV AUTH_SERVICE_URL=http://localhost:9001

# After (IPv4 only, macOS compatible)
ENV GATEWAY_URL=http://127.0.0.1:8080
ENV IDENTITY_SERVICE_URL=http://127.0.0.1:8081
ENV AUTH_SERVICE_URL=http://127.0.0.1:9001
```

The `run.sh` script also binds all published ports to `127.0.0.1` explicitly:

```bash
docker run -d \
    -p 127.0.0.1:8080:8080 \
    -p 127.0.0.1:3000:3000 \
    ...
```

Database (`DB_HOST`), Redis, and NATS still use `localhost` internally — these use raw TCP sockets (not HTTP) and are unaffected by the IPv6 issue.

## Exposed Ports

Only gateway and console ports are exposed externally (P0 security fix, Round 91):

| Port | Service | Protocol | Purpose |
|------|---------|----------|---------|
| **8080** | Gateway | HTTP | REST API gateway, reverse proxy to all services |
| **3000** | Console | HTTP | Next.js admin UI |

Backend service ports are internal-only (not published to host):

| Internal Port | Service | Purpose |
|---------------|---------|---------|
| 8081 | Identity | User CRUD, SCIM 2.0, tenant management |
| 9001 | Auth | Login, register, MFA, password policy, sessions |
| 9005 | OAuth | OAuth2/OIDC, JWKS, SAML, discovery |
| 8070 | Policy | RBAC + ABAC engine, roles, permissions |
| 8071 | Org | Organizations, departments, teams, memberships |
| 8072 | Audit | Audit event query, compliance reports |

> **Security note:** Direct backend access bypasses the gateway's JWT
> verification, tenant binding, rate limiting, and bot detection layers.
> The P0 fix (commit 11876559) removed all backend port mappings from
> `docker run`. For debugging, you may add `-p 127.0.0.1:8081:8081`
> temporarily, but never expose backend ports in production.

## CAE (Continuous Access Evaluation) Redis Dependency

GGID's CAE subsystem uses Redis for the JTI blocklist (session revocation).
When CAE is enabled, ensure Redis has sufficient memory for the sorted-set:

```bash
# Redis must be running (included in all-in-one image)
# JTI blocklist uses ~100 bytes per revoked token
# Tokens auto-expire from the sorted set when the JWT expires
```

For standalone deployments, configure Redis with:
```bash
redis-cli CONFIG SET maxmemory-policy allkeys-lru
redis-cli CONFIG SET maxmemory 256mb
```

## New API Endpoints

### System Initialization Check

```bash
# Check if the system has been initialized (any users exist)
curl http://127.0.0.1:8080/api/v1/system/initialized

# Response when not initialized:
# { "initialized": false }

# Response after seeding:
# { "initialized": true }
```

This endpoint is **unauthenticated** — the console uses it on load to decide whether to redirect to the onboarding wizard or the login page.

### Tenant Resolution

```bash
# Resolve a tenant slug to its ID and metadata
curl http://127.0.0.1:8080/api/v1/tenants/resolve?slug=default

# Response:
# {
#   "id": "00000000-0000-0000-0000-000000000001",
#   "name": "Default",
#   "slug": "default",
#   "plan": "enterprise",
#   "status": "active"
# }
```

This enables the multi-tenant login flow: users enter their workspace slug, the console resolves it to a tenant ID, then includes it in the `X-Tenant-ID` header for authentication.

### Multi-Tenant Login

Login now supports an optional `tenant_slug` field for tenant resolution:

```bash
# Login with tenant_slug (recommended for multi-tenant)
curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Password123!",
    "tenant_slug": "default"
  }'

# Login with X-Tenant-ID header (traditional)
curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "admin",
    "password": "Password123!"
  }'
```

If both `tenant_slug` and `X-Tenant-ID` are provided, the header takes precedence.

## What Is Included

The all-in-one image runs the following processes under `supervisord`:

| Process | Role | Internal Port |
|---------|------|---------------|
| PostgreSQL | Embedded database | 5432 |
| Redis | Sessions / cache | 6379 |
| NATS (JetStream) | Audit message bus | 4222 |
| identity-server | User/tenant/SCIM management | 8081 (HTTP), 50051 (gRPC) |
| auth-server | Login, MFA, sessions, password policy | 9001 (HTTP) |
| oauth-server | OAuth/OIDC/IDP endpoints | 9005 (HTTP), 50055 (gRPC) |
| policy-server | RBAC/ABAC policy engine | 8070 (HTTP), 9070 (gRPC) |
| org-server | Organizations, teams, departments | 8071 (HTTP), 9071 (gRPC) |
| audit-server | Audit events, SIEM, hash chain | 8072 (HTTP), 9072 (gRPC) |
| gateway-server | API Gateway | 8080 |
| console | Next.js admin UI | 3000 |

## Persistent Data

By default, data is stored inside the container and is lost when the container is removed. To persist PostgreSQL data across restarts, mount a named volume:

```bash
docker run -d \
  -p 127.0.0.1:8080:8080 \
  -p 127.0.0.1:3000:3000 \
  -v ggid-data:/var/lib/postgresql/data \
  --name ggid-all-in-one \
  ggid/ggid-all-in-one:latest
```

## Environment Variables

You can override the following variables at runtime with `-e KEY=VALUE`:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_USER` | `ggid` | PostgreSQL user |
| `DB_PASSWORD` | `ggid` | PostgreSQL password |
| `DB_DATABASE` | `ggid` | PostgreSQL database |
| `DATABASE_URL` | `postgres://ggid:ggid@localhost:5432/ggid?sslmode=disable` | Full DB connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `NATS_URL` | `nats://localhost:4222` | NATS connection string |
| `GATEWAY_URL` | `http://127.0.0.1:8080` | Gateway URL (IPv4) |
| `IDENTITY_SERVICE_URL` | `http://127.0.0.1:8081` | Identity service URL (IPv4) |
| `AUTH_SERVICE_URL` | `http://127.0.0.1:9001` | Auth service URL (IPv4) |
| `OAUTH_SERVICE_URL` | `http://127.0.0.1:9005` | OAuth service URL (IPv4) |
| `POLICY_SERVICE_URL` | `http://127.0.0.1:8070` | Policy service URL (IPv4) |
| `ORG_SERVICE_URL` | `http://127.0.0.1:8071` | Org service URL (IPv4) |
| `AUDIT_SERVICE_URL` | `http://127.0.0.1:8072` | Audit service URL (IPv4) |
| `NEXT_PUBLIC_TENANT_ID` | `00000000-0000-0000-0000-000000000001` | Default tenant shown in console |
| `GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK` | `true` | Allow plaintext gRPC fallback (dev only) |
| `JWT_PUBLIC_KEY_PATH` | `/app/configs/rsa_public.pem` | JWT public key for verification |
| `OAUTH_PRIVATE_KEY_PATH` | `/app/configs/rsa_private.pem` | JWT signing key |
| `OAUTH_PUBLIC_KEY_PATH` | `/app/configs/rsa_public.pem` | OAuth public key |

## Logs

All services write logs to `/var/log/supervisor/` inside the container. Tail a specific service:

```bash
# Gateway logs
docker exec ggid-all-in-one tail -f /var/log/supervisor/gateway-server.log

# All supervisor logs
docker exec ggid-all-in-one tail -f /var/log/supervisor/supervisord.log
```

## Restarting Services

Because `supervisord` manages all processes, you can restart individual services without restarting the whole container:

```bash
docker exec ggid-all-in-one supervisorctl restart auth-server
```

List all managed processes:

```bash
docker exec ggid-all-in-one supervisorctl status
```

## Stopping and Removing

```bash
docker rm -f ggid-all-in-one
```

To remove the persisted volume as well:

```bash
docker volume rm ggid-data
```

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                    ggid-all-in-one                            │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  supervisord                                           │  │
│  │  ├─ postgresql  ├─ redis  ├─ nats-server             │  │
│  │  ├─ identity-server  ├─ auth-server                   │  │
│  │  ├─ oauth-server   ├─ policy-server                   │  │
│  │  ├─ org-server     ├─ audit-server                    │  │
│  │  ├─ gateway-server  ├─ console                        │  │
│  └────────────────────────────────────────────────────────┘  │
│   Host: 127.0.0.1:8080 (gateway), 127.0.0.1:3000 (console)   │
│         + 8081, 9001, 9005, 8070, 8071, 8072                  │
└──────────────────────────────────────────────────────────────┘
```

## Troubleshooting

### Connection Refused on macOS

If you get "connection refused" when accessing `http://localhost:3000` or `http://localhost:8080`:

1. Use `http://127.0.0.1:3000` instead — macOS may resolve `localhost` to IPv6 (`::1`)
2. Verify the container is running: `docker ps | grep ggid-all-in-one`
3. Check service health: `curl http://127.0.0.1:8080/healthz`

### Services Not Ready

The first startup takes ~20 seconds for PostgreSQL, migrations, and all services to initialize. If health check fails:

```bash
# Check supervisor status
docker exec ggid-all-in-one supervisorctl status

# Check specific service logs
docker exec ggid-all-in-one tail -50 /var/log/supervisor/identity-server.log
```

### Port Already in Use

If a port is already allocated:

```bash
# Find what's using the port (e.g., 8080)
lsof -i :8080

# Stop the conflicting process or change the port mapping
```

## When to Use

Use the all-in-one image when you want to:

- Evaluate GGID locally without installing PostgreSQL, Redis, or NATS.
- Run a demo or proof-of-concept.
- Develop against the full stack without managing multiple containers.

## When NOT to Use

For production, prefer one of the following:

- **Docker Compose** (`deploy/docker-compose.yaml`) — separates infrastructure and services.
- **Kubernetes / Helm** — horizontal scaling, rolling updates, and pod isolation.
- **Managed PostgreSQL / Redis / NATS** — reduces operational burden.

The all-in-one image is intentionally **not** designed for production: it runs all processes in a single container, bundles a database, and does not provide high availability or horizontal scaling.

## File References

- `deploy/all-in-one/Dockerfile` — multi-stage build with IPv4-only service URLs
- `deploy/all-in-one/run.sh` — one-command launcher script
- `deploy/all-in-one/supervisord.conf` — process manager configuration
- `deploy/all-in-one/entrypoint.sh` — container startup sequence
- `deploy/all-in-one/postgres-start.sh` — PostgreSQL initialization
- `deploy/all-in-one/wait-for-db.sh` — DB readiness probe
- `deploy/all-in-one/README.md` — quick reference

## Related Docs

- `docs/research/onboarding-and-multi-tenant-design.md` — Onboarding wizard and multi-tenant login design
- `docs/deployment-guide.md` — Full deployment options (Compose, Kubernetes, Helm)
- `docs/guides/docker-deployment.md` — Docker Compose multi-service deployment guide
- `docs/research/docker-e2e-infra-gap.md` — Historical Docker E2E infrastructure fixes
