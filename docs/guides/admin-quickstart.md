# Admin Quickstart — 10 Minutes to Master GGID

This guide gets an administrator productive with GGID in 10 minutes. You'll learn the core workflows: user management, roles, OAuth clients, audit, and security policies.

## Prerequisites

- GGID running (all-in-one Docker or docker-compose)
- Admin credentials: username=`admin`, password=`Admin@123456`
- Tenant ID: `00000000-0000-0000-0000-000000000001`

## Minute 0-1: Login & Get Your Token

```bash
# Set your base URL
export GGID=http://localhost:8080
export TENANT=00000000-0000-0000-0000-000000000001

# Login
TOKEN=$(curl -s -X POST $GGID/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"admin","password":"Admin@123456"}' | jq -r '.access_token')

echo "Token: ${TOKEN:0:40}..."

# Verify
curl -s $GGID/api/v1/auth/profile \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

Or open the console at `http://localhost:3000` and log in with the same credentials.

## Minute 1-3: User Management

### Create Users

```bash
# Create a developer
curl -s -X POST $GGID/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "email": "alice@corp.com",
    "password": "SecureDev@123",
    "name": "Alice Chen",
    "username": "alice"
  }' | jq '.id'

# Create a manager
curl -s -X POST $GGID/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "email": "bob@corp.com",
    "password": "SecureMgr@123",
    "name": "Bob Smith",
    "username": "bob"
  }' | jq '.id'
```

### List & Search Users

```bash
# List all users
curl -s $GGID/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.users[] | {username, email, status}'

# Get a specific user
USER_ID=$(curl -s $GGID/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq -r '.users[] | select(.username=="alice") | .id')

curl -s $GGID/api/v1/users/$USER_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

## Minute 3-5: Roles & Permissions

### View Built-in Roles

```bash
curl -s $GGID/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.roles[] | select(.system_role==true) | {key, name}'
```

You'll see: `admin`, `editor`, `viewer` (system roles).

### Create a Custom Role

```bash
curl -s -X POST $GGID/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "key": "developer",
    "name": "Developer",
    "description": "Application developer with code repository access"
  }' | jq .
```

### Assign Role to User

```bash
curl -s -X POST $GGID/api/v1/roles/assign \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"role_id\": \"$(curl -s $GGID/api/v1/roles -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" | jq -r '.roles[] | select(.key=="developer") | .id')\"
  }" | jq .
```

## Minute 5-7: OAuth Client Setup

### Register a Web Application

```bash
curl -s -X POST $GGID/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "client_name": "My Web App",
    "redirect_uris": ["http://localhost:3001/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "token_endpoint_auth_method": "client_secret_post",
    "scope": "openid profile email"
  }' | jq .
```

Save the `client_id` and `client_secret`.

### Register a Machine-to-Machine Client

```bash
curl -s -X POST $GGID/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "client_name": "Backend API Service",
    "grant_types": ["client_credentials"],
    "token_endpoint_auth_method": "client_secret_post",
    "scope": "users:read roles:read"
  }' | jq .
```

## Minute 7-8: Audit Trail

### Query Audit Events

```bash
# Recent events
curl -s "$GGID/api/v1/audit/events?limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.events[] | {timestamp, type, action, user}'

# Verify hash-chain integrity
curl -s "$GGID/api/v1/audit/integrity" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

## Minute 8-9: Security Configuration

### Check Active Sessions

```bash
curl -s $GGID/api/v1/auth/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.sessions[] | {session_id, ip, created_at}'
```

### Enforce MFA (via Console)

1. Open `http://localhost:3000/settings`
2. Navigate to **Security → MFA**
3. Enable **Require MFA for admins**
4. Select TOTP as the primary method

### View Conditional Access Policies

```bash
curl -s $GGID/api/v1/auth/conditional-access/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

## Minute 9-10: Verify Your Setup

### Quick Health Check

```bash
echo "=== System Health ==="
curl -s $GGID/healthz | jq .

echo "=== User Count ==="
curl -s $GGID/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.total'

echo "=== OAuth Clients ==="
curl -s $GGID/api/v1/oauth/clients \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.clients | length'

echo "=== Roles ==="
curl -s $GGID/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.roles | length'

echo "=== Audit Integrity ==="
curl -s $GGID/api/v1/audit/integrity \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" | jq '.valid'
```

## Console Tour

| Page | URL | What You Can Do |
|------|-----|-----------------|
| Dashboard | `/dashboard` | Overview: users, sessions, risk, alerts |
| Users | `/users` | Create, edit, lock, delete users |
| Roles | `/settings/roles` | Manage roles and permissions |
| OAuth Clients | `/settings/oauth` | Register and manage client apps |
| Audit Trail | `/audit` | Search and export audit events |
| Security Center | `/security` | ITDR alerts, risk analytics |
| Settings | `/settings` | MFA, policies, branding, i18n |

## Common Operations Cheat Sheet

```bash
# Lock a user account
curl -X POST $GGID/api/v1/users/$USER_ID/lock \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Unlock
curl -X POST $GGID/api/v1/users/$USER_ID/unlock \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Delete a user
curl -X DELETE $GGID/api/v1/users/$USER_ID \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Revoke a session
curl -X DELETE $GGID/api/v1/auth/sessions/$SESSION_ID \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT"

# Check a permission
curl -s -X POST $GGID/api/v1/policies/check \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"user_id":"USER_ID","resource":"users","action":"read"}' | jq .
```

## Next Steps

- [Integration Guide](integration-guide.md) — Connect your app via OAuth/SAML/WebAuthn
- [API Cookbook](../api-cookbook.md) — 20 practical curl recipes
- [Conditional Access](conditional-access-guide.md) — Configure security policies
- [MFA Architecture](mfa-architecture.md) — Multi-factor authentication setup
- [Deployment Guide](deployment-guide.md) — Production deployment
