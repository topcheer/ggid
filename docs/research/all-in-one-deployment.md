# All-in-One Docker Deployment Guide

*Research document — 2026-07-15*

## Summary

GGID provides an all-in-one Docker image that bundles the entire IAM stack into a single container. This is ideal for **local development, demos, and single-node evaluations** where you want to run PostgreSQL, Redis, NATS, all 7 backend services, and the admin console with one command.

## One-Command Quick Start

Build the image from the repository root (this may take several minutes):

```bash
docker build -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .
```

Run the container:

```bash
docker run -d \
  -p 8080:8080 \
  -p 3000:3000 \
  --name ggid-all-in-one \
  ggid/ggid-all-in-one:latest
```

Wait approximately 15 seconds for PostgreSQL, Redis, NATS, migrations, and all services to start. Then open the console:

- **Admin Console:** http://localhost:3000
- **API Gateway:** http://localhost:8080
- **Default Tenant ID:** `00000000-0000-0000-0000-000000000001`

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

## Exposed Ports

Only the following ports are exposed by default:

- `8080` — API Gateway (REST / proxy)
- `3000` — Admin Console
- `8070` / `8071` / `8072` — Internal service HTTP (optional)
- `9001` / `9005` — Auth / OAuth HTTP (optional)
- `8081` — Identity HTTP (optional)

For local development, `8080` and `3000` are usually sufficient.

## Persistent Data

By default, data is stored inside the container and is lost when the container is removed. To persist PostgreSQL data across restarts, mount a named volume:

```bash
docker run -d \
  -p 8080:8080 \
  -p 3000:3000 \
  -v ggid-data:/var/lib/postgresql/data \
  --name ggid-all-in-one \
  ggid/ggid-all-in-one:latest
```

## First API Call

Register an admin user through the gateway using the default tenant:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "Password123!"
  }'
```

Then log in:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "admin",
    "password": "Password123!"
  }'
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
| `GATEWAY_ADDR` | `:8080` | Gateway bind address |
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
docker stop ggid-all-in-one
docker rm ggid-all-in-one
```

To remove the persisted volume as well:

```bash
docker volume rm ggid-data
```

## Architecture

```text
┌──────────────────────────────────────────────────────────┐
│                    ggid-all-in-one                       │
│  ┌─────────────────────────────────────────────────────┐  │
│  │  supervisord                                        │  │
│  │  ├─ postgresql  ├─ redis  ├─ nats-server          │  │
│  │  ├─ identity-server  ├─ auth-server               │  │
│  │  ├─ oauth-server   ├─ policy-server              │  │
│  │  ├─ org-server     ├─ audit-server               │  │
│  │  ├─ gateway-server  ├─ console                   │  │
│  └─────────────────────────────────────────────────────┘  │
│         Host: 8080 (gateway), 3000 (console)               │
└──────────────────────────────────────────────────────────┘
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

The all-in-one image is intentionally **not** designed for production: it runs all processes as `root` inside a single container, bundles a database, and does not provide high availability or horizontal scaling.

## File References

- `deploy/all-in-one/Dockerfile`
- `deploy/all-in-one/supervisord.conf`
- `deploy/all-in-one/entrypoint.sh`
- `deploy/all-in-one/postgres-start.sh`
- `deploy/all-in-one/wait-for-db.sh`
- `deploy/all-in-one/README.md`

## Related Docs

- `docs/deployment-guide.md` — Full deployment options (Compose, Kubernetes, Helm).
- `docs/docker-deployment-state.md` — Docker Compose multi-service deployment status.
- `docs/research/docker-e2e-infra-gap.md` — Historical Docker E2E infrastructure fixes.
