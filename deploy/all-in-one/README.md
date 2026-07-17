# GGID All-in-One

A single Docker image that runs the entire GGID IAM stack: PostgreSQL, Redis, NATS, all 7 backend services (gateway, identity, auth, oauth, policy, org, audit), and the admin console.

## Quick Start

```bash
docker build -f deploy/all-in-one/Dockerfile -t ggid/ggid-all-in-one:latest .
docker run -d -p 8080:8080 -p 3000:3000 --name ggid-all-in-one ggid/ggid-all-in-one:latest
```

Wait ~15 seconds for all services to start, then open:

- Admin Console: http://localhost:3000
- API Gateway: http://localhost:8080
- Default tenant: `00000000-0000-0000-0000-000000000001`

## Default Credentials

After first boot, the system is ready to register users via the API. Use the API Gateway with the default tenant header:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","email":"admin@example.com","password":"Password123!"}'
```

## Included Services

| Service | Internal Port | Description |
|---------|---------------|-------------|
| Gateway | 8080 | API Gateway |
| Identity | 8081 | User/tenant/SCIM management |
| Auth | 9001 | Login, MFA, sessions, password policy |
| OAuth | 9005 | OAuth/OIDC/IDP endpoints |
| Policy | 8070 | RBAC/ABAC policy engine |
| Org | 8071 | Organizations, teams, departments |
| Audit | 8072 | Audit events, SIEM, hash chain |
| Console | 3000 | Next.js admin UI |
| PostgreSQL | 5432 | Embedded database |
| Redis | 6379 | Sessions/cache |
| NATS | 4222 | Audit message bus |

## Persistent Data

Data is stored inside the container at `/var/lib/postgresql/data`. To persist across container restarts, mount a volume:

```bash
docker run -d -p 8080:8080 -p 3000:3000 \
  -v ggid-data:/var/lib/postgresql/data \
  --name ggid-all-in-one ggid/ggid-all-in-one:latest
```

## Logs

All service logs are written to `/var/log/supervisor/` inside the container:

```bash
docker exec ggid-all-in-one tail -f /var/log/supervisor/gateway-server.log
```

## Environment Variables

Key variables you can override at runtime:

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_USER` | `ggid` | PostgreSQL user |
| `DB_PASSWORD` | `ggid` | PostgreSQL password |
| `DB_DATABASE` | `ggid` | PostgreSQL database |
| `GATEWAY_ADDR` | `:8080` | Gateway bind address |
| `NEXT_PUBLIC_TENANT_ID` | `00000000-0000-0000-0000-000000000001` | Default tenant ID |
| `GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK` | `true` | Allow plaintext gRPC fallback (dev only) |
| `PASSWORD_PEPPER` | _(unset)_ | HMAC-SHA256 pre-hash pepper. Set for production to add server-side password secret. |
| `INTERNAL_AUTH_SECRET` | _(unset)_ | HMAC secret for service-to-service internal auth. Set when enabling CAE pipeline. |
| `AUDIT_HASH_CHAIN_SECRET` | _(unset)_ | Secret for tamper-evident audit log hash chain. Set for production audit integrity. |

## Security Considerations

### Port Exposure (P0 — Fixed)

Only the gateway (8080) and console (3000) ports should be exposed externally.
All backend service ports (8081-8072, 9001, 9005, 5432, 6379, 4222) must remain
internal. The all-in-one Dockerfile exposes only `8080` and `3000` — do not add
additional `-p` port mappings in production deployments.

### CAE (Continuous Access Evaluation) Redis Dependency

The CAE pipeline uses Redis for the JTI (JWT ID) blocklist. When a user's sessions
are revoked (e.g., by ITDR threat detection or admin action), the JTI is added
to a Redis sorted set. The gateway's CAE middleware checks this set on every
request (~0.3ms overhead).

**Requirements for CAE:**
- Redis must be running (embedded in all-in-one image)
- `INTERNAL_AUTH_SECRET` must be set for ITDR → CAE trigger pipeline
- Gateway pod must have Redis connectivity

If Redis is unavailable, CAE checks are skipped (fail-open) — access tokens
remain valid until natural expiry. This is acceptable for dev but not production.

### Password Pepper

Set `PASSWORD_PEPPER` to a strong random string (32+ characters) for production.
Without it, passwords are still hashed with Argon2id (strong), but lack the
additional server-side secret that protects against database-only breaches.

```bash
docker run -d -p 8080:8080 -p 3000:3000 \
  -e PASSWORD_PEPPER=$(openssl rand -hex 32) \
  -e INTERNAL_AUTH_SECRET=$(openssl rand -hex 32) \
  -e AUDIT_HASH_CHAIN_SECRET=$(openssl rand -hex 32) \
  -v ggid-data:/var/lib/postgresql/data \
  --name ggid-all-in-one ggid/ggid-all-in-one:latest
```

### gRPC TLS

gRPC services default to plaintext (`GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true`).
For production, set:
- `GRPC_TLS_ENABLED=true`
- `GRPC_TLS_CERT=/path/to/cert.pem`
- `GRPC_TLS_KEY=/path/to/key.pem`
- Remove `GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK` (or set to `false`)

## Notes

- This image is intended for **local development, demos, and single-node deployments**.
- For production, use the multi-service `docker-compose.yaml` or Kubernetes Helm charts.
- The image size is large because it includes PostgreSQL, Redis, NATS, and all service binaries.
