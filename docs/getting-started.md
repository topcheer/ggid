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
28d6fe98-adeb-4c0c-b49b-20c6695bbca6
```

---

## Step 2: Register a User

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6" \
  -d '{
    grant_type=password&username= "alice",
    "email": "alice@example.com",
    "password": "SecurePass123!"
  }' | jq .
```

**Expected:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  grant_type=password&username= "alice",
  "email": "alice@example.com",
  "status": "active"
}
```

---

## Step 3: Login and Get JWT

```bash
# Login and save JWT
JWT=$(curl -s -X POST http://localhost:8080/oauth/token \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6" \
  -d '{
    grant_type=password&username= "alice",
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
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6" | jq .
```

### Without JWT (should get 401)

```bash
curl -s http://localhost:8080/api/v1/users \
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6"
# Expected: 401 Unauthorized
```

---

## Step 5: Create a Role and Assign

```bash
# Create admin role
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6" \
  -d '{
    "key": "editor",
    "name": "Editor",
    "permissions": ["users:read", "users:write"]
  }' | jq .

# List roles
curl -s http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: 28d6fe98-adeb-4c0c-b49b-20c6695bbca6" | jq .
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
| Login | `POST /oauth/token (grant_type=password)` |
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

---

## Step 6: First Console Login

The admin console is at `http://localhost:3000`.

1. Open `http://localhost:3000` in your browser
2. Login with the credentials you just created:
   - Username: `admin`
   - Password: `SecurePass123!`
3. You should see the dashboard with overview metrics

### Console Pages

| Page | Purpose |
|------|---------|
| **Dashboard** | System overview, request rates, active sessions |
| **Users** | View, create, edit, delete, suspend users |
| **Roles** | Manage RBAC roles and permissions |
| **Organizations** | Manage org hierarchy and membership |
| **Audit** | Query audit events by date, user, or path |
| **Settings** | System configuration, security settings |

---

## Step 7: Explore OAuth 2.1 Flows

### Authorization Code Flow with PKCE

```bash
# 1. Register an OAuth client
curl -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "my-app",
    "client_secret": "my-secret",
    "redirect_uris": ["http://localhost:3001/callback"],
    "grant_types": ["authorization_code"],
    "response_types": ["code"],
    "scope": "openid profile email"
  }'

# 2. Direct user to authorization endpoint
# Open in browser:
# http://localhost:8080/api/v1/oauth/authorize?
#   response_type=code&
#   client_id=my-app&
#   redirect_uri=http://localhost:3001/callback&
#   scope=openid profile&
#   code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
#   code_challenge_method=S256&
#   state=xyz123

# 3. Exchange code for token
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=AUTH_CODE_FROM_REDIRECT" \
  -d "redirect_uri=http://localhost:3001/callback" \
  -d "client_id=my-app" \
  -d "code_verifier=YOUR_CODE_VERIFIER"
```

### Token Introspection

```bash
# Check if a token is valid (RFC 7662)
curl -X POST http://localhost:8080/api/v1/oauth/introspect \
  -u "my-app:my-secret" \
  -d "token=$ACCESS_TOKEN"

# Response:
# {"active":true,"scope":"read:users","client_id":"my-app","exp":1699999999}
```

---

## Step 8: Configure MFA

```bash
# Enable TOTP for your user
curl -X POST http://localhost:8080/api/v1/auth/mfa/setup \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"method":"totp"}'

# Response includes secret + QR code
# Scan QR code with Google Authenticator

# Verify MFA enrollment
curl -X POST http://localhost:8080/api/v1/auth/mfa/verify \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"method":"totp","code":"123456"}'
```

---

## Step 9: Query Audit Events

```bash
# Get recent audit events for your tenant
curl "http://localhost:8080/api/v1/audit/events?limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Filter by date range
curl "http://localhost:8080/api/v1/audit/events?from=2025-07-01&to=2025-07-11" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Filter by status code (errors only)
curl "http://localhost:8080/api/v1/audit/events?status_code=401" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

---

## Step 10: Set Up Webhooks

```bash
# Register a webhook endpoint
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/my-webhook",
    "events": ["user.created", "user.deleted", "auth.login_failed"],
    "description": "SIEM integration"
  }'

# Save the webhook secret for signature verification
# The secret is only shown once in the response
```

See: [Webhook Guide](./webhook-guide.md) for HMAC verification code.

---

## Using the SDK

### Go SDK

```go
import "github.com/ggid/ggid/sdk/go"

client := ggid.NewClient("http://localhost:8080", "your-jwt-token")

// List users
users, err := client.Users.List(ctx, &ggid.ListUsersRequest{
    TenantID: "28d6fe98-adeb-4c0c-b49b-20c6695bbca6",
    Page:     1,
    PerPage:  20,
})

// Create user
user, err := client.Users.Create(ctx, &ggid.CreateUserRequest{
    Username: "newuser",
    Email:    "new@example.com",
    Password: "SecurePass123!",
})
```

### Node.js SDK

```javascript
const { GGIDClient } = require('@ggid/sdk-node');

const client = new GGIDClient({
  baseURL: 'http://localhost:8080',
  token: 'your-jwt-token'
});

// List users
const users = await client.users.list({
  tenantId: '28d6fe98-adeb-4c0c-b49b-20c6695bbca6'
});

// Create role
const role = await client.roles.create({
  name: 'Editor',
  key: 'editor',
  permissions: ['read:users', 'write:content']
});
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `docker compose up` fails | Check ports 5432, 6379, 4222 aren't already in use |
| Login returns 401 | Verify `username` field (not `email`) is used as credential identifier |
| Register returns 409 | Username must be unique within tenant — include `username` field |
| Create role returns 500 | Include unique `key` field (UNIQUE constraint on tenant_id + key) |
| 429 Too Many Requests | Auth rate limits after ~5 failed logins — restart auth container in dev |
| Gateway returns 502 | Backend service not ready — wait 30s or check `docker compose ps` |
| NATS healthcheck fails | Ensure NATS has `-m 8222` flag for monitoring port |

---

## What's Next?

- [Tutorials](./tutorials/) — Step-by-step guides for common scenarios
- [Authentication Guide](./authentication-guide.md) — All auth methods
- [RBAC Guide](./rbac-guide.md) — Role-based access control
- [Multi-Tenancy](./multi-tenancy.md) — Tenant management
- [OAuth Flows](./oauth-flows-guide.md) — OAuth 2.1 + OIDC details
- [Operations Runbook](./operations-runbook.md) — Production operations
