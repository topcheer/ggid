# GGID Quick Start Guide

Get up and running with GGID in 5 minutes — from zero to a successful authenticated API call.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose v2+
- `curl` (pre-installed on macOS/Linux)

---

## Step 1: Start the Stack

```bash
git clone https://github.com/ggid/ggid.git
cd ggid
cd deploy && docker compose up -d
```

Wait ~30 seconds for all services to become healthy:

```bash
docker compose ps
```

You should see 13 containers with `Up` or `healthy` status.

**Service ports:**

| Service    | Port | Description                    |
|------------|------|--------------------------------|
| Gateway    | 8080 | API entry point (use this)     |
| Console    | 3000 | Admin UI (Next.js)             |
| Identity   | 8081 | User CRUD (internal)           |
| Auth       | 9001 | Login/register (internal)      |
| PostgreSQL | 5432 | Database                       |
| Redis      | 6379 | Session/rate-limit cache       |
| NATS       | 4222 | Event bus for audit            |

---

## Step 2: Register a User

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "demo",
    "email": "demo@example.com",
    "password": "SecurePass@123"
  }'
```

Expected response (HTTP 201):

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "user registered"
}
```

> **Note:** The `X-Tenant-ID` header is required on all requests. The default
> tenant `00000000-0000-0000-0000-000000000001` is created by migrations.

---

## Step 3: Login and Get JWT

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "demo",
    "password": "SecurePass@123"
  }' | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "JWT length: ${#TOKEN} chars"
```

The JWT (RS256-signed) contains claims like `sub`, `tenant_id`, `roles`, and `exp`.

---

## Step 4: Make an Authenticated API Call

```bash
# List users (requires JWT)
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

Without the JWT, you get `401 Unauthorized`:

```bash
# This fails with 401
curl -s http://localhost:8080/api/v1/users
```

---

## Step 5: Explore More APIs

### Create a Role

```bash
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "key": "editor",
    "name": "Content Editor",
    "description": "Can edit content"
  }'
```

### Create an Organization

```bash
curl -s -X POST http://localhost:8080/api/v1/orgs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"name": "Engineering"}'
```

### Query Audit Events

```bash
curl -s "http://localhost:8080/api/v1/audit/events?tenant_id=00000000-0000-0000-0000-000000000001&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Admin Console (Optional)

Open http://localhost:3000 in your browser for the Next.js admin console:
- Dashboard with service health
- User management (CRUD, lock/unlock)
- Role management with tabs
- Organization tree view
- Audit event explorer

---

## Troubleshooting

### 401 Unauthorized
- Ensure the JWT hasn't expired (default 1 hour TTL)
- Include `Authorization: Bearer <token>` header
- Include `X-Tenant-ID` header

### 400 Bad Request — "missing tenant context"
- Add `X-Tenant-ID: 00000000-0000-0000-0000-000000000001` header

### 409 Conflict on register
- Username must be unique within the tenant
- Use a different `username` value

### 429 Too Many Requests
- Auth rate-limits after ~5 failed login attempts
- Restart the auth container: `docker compose restart auth`

### Services not starting
- Check logs: `docker compose logs <service-name>`
- Ensure ports 8080, 5432, 6379 are not in use

---

## Next Steps

- [API Reference (OpenAPI)](./openapi.yaml) — complete endpoint documentation
- [Deployment Guide](./deployment.md) — production deployment instructions
- [Architecture Decision Records](./adr/) — key design decisions
- [Go SDK](../sdk/go/README.md) — integrate GGID into your Go backend
- [Node.js SDK](../sdk/node/README.md) — integrate GGID into your Node.js app

---

## Clean Up

```bash
cd deploy && docker compose down -v  # -v removes volumes (DB data)
```
