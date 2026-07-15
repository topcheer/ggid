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

## Notes

- This image is intended for **local development, demos, and single-node deployments**.
- For production, use the multi-service `docker-compose.yaml` or Kubernetes Helm charts.
- The image size is large because it includes PostgreSQL, Redis, NATS, and all service binaries.
