# GGID Quick Start Guide

Get up and running with GGID in 5 minutes — from zero to a successful
authenticated API call through the Admin Console.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose v2+
- `curl` (pre-installed on macOS/Linux)
- Modern web browser (for Admin Console)

---

## Step 1: Clone and Start (2 min)

```bash
git clone https://github.com/ggid/ggid.git
cd ggid

# Start all services (7 microservices + PostgreSQL + Redis + NATS + Console)
docker compose -f deploy/docker-compose.yaml up -d

# Wait for healthchecks to pass (~30 seconds)
docker compose -f deploy/docker-compose.yaml ps --format "table {{.Name}}\t{{.Status}}"
```

Verify all containers are healthy:

```
NAME                STATUS
ggid-postgres       Up (healthy)
ggid-redis          Up (healthy)
ggid-nats           Up (healthy)
ggid-gateway        Up (healthy)
ggid-auth           Up (healthy)
ggid-identity       Up (healthy)
ggid-policy         Up (healthy)
ggid-org            Up (healthy)
ggid-audit          Up (healthy)
ggid-oauth          Up (healthy)
ggid-console        Up (healthy)
```

**Troubleshooting:** If containers aren't healthy after 60s, check logs:
```bash
docker compose -f deploy/docker-compose.yaml logs gateway --tail 20
docker compose -f deploy/docker-compose.yaml logs auth --tail 20
```

Common issues:
- **Port conflict:** Ensure ports 8080, 3000, 5432, 6379, 4222 are free
- **Database not ready:** The migrate init container handles this; wait longer
- **OOM:** Docker Desktop needs at least 4GB RAM allocated

---

## Step 2: Register a User (30 sec)

```bash
export GW="http://localhost:8080"
export TENANT="00000000-0000-0000-0000-000000000001"

curl -sX POST "$GW/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "admin",
    "email": "admin@test.com",
    "password": "Sup3rSecure!Pass"
  }' | jq .
```

Expected response (201 Created):
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "user registered"
}
```

**Troubleshooting:**
- `409 Conflict` — username already exists, use a different one
- `400 Bad Request` — password doesn't meet policy (min 12 chars, upper+lower+digit+special)
- `000 (connection refused)` — Gateway not started yet, wait and retry

---

## Step 3: Login and Get JWT (10 sec)

```bash
export TOKEN=$(curl -sX POST "$GW/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "admin",
    "password": "Sup3rSecure!Pass"
  }' | jq -r '.access_token')

echo "JWT length: ${#TOKEN} chars"
# JWT length: 693 chars
```

**Troubleshooting:**
- `401 Unauthorized` — wrong password or user not found
- `429 Too Many Requests` — too many login attempts, wait 15 min or restart auth container:
  ```bash
  docker compose -f deploy/docker-compose.yaml restart auth
  ```

---

## Step 4: Make an Authenticated API Call (10 sec)

```bash
# List all users (should see the admin we just registered)
curl -s "$GW/api/v1/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

```bash
# Try without JWT — should get 401
curl -s "$GW/api/v1/users" \
  -H "X-Tenant-ID: $TENANT"

# Response: {"error":"unauthorized"}
```

---

## Step 5: Open the Admin Console (30 sec)

Open your browser to: **http://localhost:3000**

1. Login with username `admin` and password `Sup3rSecure!Pass`
2. Explore the Dashboard — shows user count, active sessions, recent events
3. Navigate to **Users** to see the user you just registered
4. Navigate to **Audit** to see the login event you just created

---

## What's Next?

Now that you have a running GGID instance:

- **[API Reference](./api-reference.md)** — Complete REST endpoint documentation
- **[API Examples](./api-examples.md)** — End-to-end walkthrough with curl
- **[Integration Guide](./integration-guide.md)** — Integrate GGID with your app
- **[Developer Guide](./developer-guide.md)** — Set up local development
- **[SDK Guide](./sdk-guide.md)** — Use Go/Node/Java/Python SDKs
- **[Deployment](./deployment.md)** — Production deployment guide
- **[Console README](../console/README.md)** — Admin Console usage guide

---

## Stop and Clean Up

```bash
# Stop all containers (data preserved in volumes)
docker compose -f deploy/docker-compose.yaml down

# Stop and delete all data (fresh start)
docker compose -f deploy/docker-compose.yaml down -v
```

---

## Common Issues

| Problem | Solution |
|---------|----------|
| `docker compose` not found | Install Docker Compose v2+ or use `docker-compose` |
| Port 5432 already in use | Stop local PostgreSQL or change port in docker-compose |
| Console shows blank page | Wait for Next.js build, check `docker logs ggid-console` |
| Gateway returns 502 | Backend service not ready, wait 30s and retry |
| `X-Tenant-ID` missing error | All API calls need this header (use the default tenant ID) |
| Rate limited (429) | Auth limits to 10 logins/min; restart auth container to reset |

---

## Next Steps: SDK Integration

### Go SDK

```bash
go get github.com/ggid/ggid/pkg/sdk/go@latest
```

```go
package main

import (
    "context"
    "fmt"
    ggid "github.com/ggid/ggid/pkg/sdk/go"
)

func main() {
    client := ggid.NewClient("http://localhost:8080", ggid.WithTenantID("00000000-0000-0000-0000-000000000001"))

    // Login
    resp, err := client.Auth.Login(context.Background(), &ggid.LoginRequest{
        Username: "admin",
        Password: "Sup3rSecure!Pass",
    })
    if err != nil {
        panic(err)
    }

    // Use the token for authenticated calls
    authClient := client.WithToken(resp.AccessToken)

    users, err := authClient.Users.List(context.Background())
    if err != nil {
        panic(err)
    }

    for _, u := range users {
        fmt.Printf("- %s (%s)\n", u.Username, u.Email)
    }
}
```

### Node.js / TypeScript SDK

```bash
npm install @ggid/sdk
```

```typescript
import { GGIDClient } from '@ggid/sdk';

const client = new GGIDClient({
  baseURL: 'http://localhost:8080',
  tenantID: '00000000-0000-0000-0000-000000000001',
});

// Login
const { access_token } = await client.auth.login({
  username: 'admin',
  password: 'Sup3rSecure!Pass',
});

// Authenticated calls
const users = await client.users.list(access_token);
console.log(users);
```

### curl One-Liners (Copy-Paste)

```bash
# Set variables
export GW="http://localhost:8080"
export TENANT="00000000-0000-0000-0000-000000000001"

# Register
curl -sX POST "$GW/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"testuser","email":"test@test.com","password":"Test1234!"}' | jq .

# Login
export TOKEN=$(curl -sX POST "$GW/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"testuser","password":"Test1234!"}' | jq -r .access_token)

# List users
curl -s "$GW/api/v1/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq .

# Create role
curl -sX POST "$GW/api/v1/roles" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"name":"viewer","key":"viewer","description":"Read-only access"}' | jq .

# Query audit log
curl -s "$GW/api/v1/audit/events?limit=10" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq .

# Create organization
curl -sX POST "$GW/api/v1/orgs" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"name":"Engineering","description":"Engineering Department"}' | jq .
```

### Using Docker Compose E2E Test

```bash
# Run the built-in E2E test suite
bash deploy/e2e-docker-test.sh

# Expected: 11/11 PASS
# Tests: health, register, login, JWT, 401, users, roles, orgs, audit, 401, 409
```
