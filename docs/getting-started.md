# GGID Getting Started (5-Minute Quickstart)

Get GGID running locally in 5 minutes with Docker Compose. Register a user,
log in, get a JWT, and call a protected API endpoint.

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- `curl` and `jq` (for API calls)
- Ports available: 8080, 5432, 6379, 4222

---

## Step 1: Start GGID

```bash
# Clone the repository
git clone https://github.com/ggid/ggid.git
cd ggid

# Start all services
cd deploy && docker compose up -d

# Wait for services to be healthy (about 30 seconds)
sleep 30

# Verify Gateway is running
curl -s http://localhost:8080/healthz | jq .
```

**Expected output:**
```json
{
  "status": "ok",
  "service": "gateway"
}
```

### Default Tenant

All requests require a tenant ID. The default tenant is:
```
00000000-0000-0000-0000-000000000001
```

---

## Step 2: Register a User

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "SecurePass123!"
  }' | jq .
```

**Expected:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "email": "alice@example.com",
  "status": "active"
}
```

---

## Step 3: Login and Get JWT

```bash
# Login and save JWT
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "alice",
    "password": "SecurePass123!"
  }' | jq -r '.access_token')

# Verify you got a token
echo "JWT length: ${#JWT} chars"
echo "JWT (first 30): ${JWT:0:30}..."
```

---

## Step 4: Call a Protected API

```bash
# List users (requires JWT)
curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
```

### Without JWT (should get 401)

```bash
curl -s http://localhost:8080/api/v1/users \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
# Expected: 401 Unauthorized
```

---

## Step 5: Create a Role and Assign

```bash
# Create admin role
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "key": "editor",
    "name": "Editor",
    "permissions": ["users:read", "users:write"]
  }' | jq .

# List roles
curl -s http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
```

---

## Quick Reference

| Action | Command |
|--------|---------|
| Start | `cd deploy && docker compose up -d` |
| Stop | `cd deploy && docker compose down` |
| Logs | `docker logs ggid-gateway -f` |
| Health | `curl localhost:8080/healthz` |
| Register | `POST /api/v1/auth/register` |
| Login | `POST /api/v1/auth/login` |
| Refresh | `POST /api/v1/auth/refresh` |
| List Users | `GET /api/v1/users` |

### Default Ports

| Service | Port |
|---------|------|
| Gateway | 8080 |
| Identity | 8081 |
| Auth | 9001 |
| OAuth | 9005 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| NATS | 4222 |

---

## Next Steps

- [Configuration Reference](./configuration-reference.md) — All env vars
- [API Reference](./api-reference.md) — Complete REST API
- [SDK Guide](./sdk-guide.md) — Go/Node/Java SDKs
- [Deployment Guide](./deployment-guide.md) — Production setup
