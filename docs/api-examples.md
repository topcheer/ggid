# GGID API Examples

Complete curl examples for every GGID REST API endpoint, grouped by service.

## Setup

```bash
# Common variables
export GW="http://localhost:8080"
export TENANT="00000000-0000-0000-0000-000000000001"

# Login and save token (do this first)
RESPONSE=$(curl -s -X POST $GW/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"demo","password":"SecurePass@123"}')

export TOKEN=$(echo $RESPONSE | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
export AUTH="Authorization: Bearer $TOKEN"
```

---

## Auth Service

### Register a New User

```bash
curl -s -X POST $GW/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "jane.doe",
    "email": "jane@example.com",
    "password": "SecurePass@123"
  }'
```

**201 Created:**
```json
{"user_id":"550e8400-e29b-41d4-a716-446655440000","message":"user registered"}
```

**409 Conflict:**
```json
{"error":"username already exists"}
```

---

### Login

```bash
curl -s -X POST $GW/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"jane.doe","password":"SecurePass@123"}'
```

**200 OK:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "rt_abc123def456...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

**401 Unauthorized:**
```json
{"error":"invalid credentials"}
```

**429 Too Many Requests** (after 5 failed attempts):
```json
{"error":"rate limit exceeded"}
```

---

### Refresh Token

```bash
curl -s -X POST $GW/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"rt_abc123def456..."}'
```

**200 OK:** Returns a new token set (refresh token is rotated).

**401 Unauthorized:** Refresh token expired or already used.

---

### Logout

```bash
curl -s -X POST $GW/api/v1/auth/logout \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -d '{"access_token":"eyJhbGciOiJSUzI1NiIs..."}'
```

**200 OK:** `{"status":"logged out"}`

---

### Forgot Password

```bash
curl -s -X POST $GW/api/v1/auth/password/forgot \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"email":"jane@example.com"}'
```

**200 OK:** `{"status":"reset email sent"}`

> Always returns 200 to prevent email enumeration.

---

### Reset Password

```bash
curl -s -X POST $GW/api/v1/auth/password/reset \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "token":"reset_token_from_email",
    "password":"NewSecurePass@456"
  }'
```

**200 OK:** `{"status":"password reset successful"}`

**400 Bad Request:** `{"error":"invalid or expired token"}`

---

### Change Password (Authenticated)

```bash
curl -s -X POST $GW/api/v1/auth/password/change \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "current_password":"SecurePass@123",
    "new_password":"NewSecurePass@456"
  }'
```

---

### MFA Setup (TOTP)

```bash
curl -s -X POST $GW/api/v1/auth/mfa/setup \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"
```

**200 OK:**
```json
{
  "secret":"JBSWY3DPEHPK3PXP",
  "qr_code":"data:image/png;base64,iVBORw0KGgo..."
}
```

### MFA Verify

```bash
curl -s -X POST $GW/api/v1/auth/mfa/verify \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"code":"123456"}'
```

### MFA Login (second step)

```bash
curl -s -X POST $GW/api/v1/auth/mfa/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"mfa_token":"temp_token_from_login","code":"123456"}'
```

---

### Magic Link (Passwordless)

```bash
# Send magic link
curl -s -X POST $GW/api/v1/auth/magic-link \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"email":"jane@example.com"}'

# Verify magic link token
curl -s -X POST $GW/api/v1/auth/magic-link/verify \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"token":"magic_link_token_from_email"}'
```

---

### Email Verification

```bash
# Send verification email
curl -s -X POST $GW/api/v1/auth/email/resend \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"email":"jane@example.com"}'

# Verify email with token
curl -s -X POST $GW/api/v1/auth/email/verify \
  -H "Content-Type: application/json" \
  -d '{"token":"verification_token_from_email"}'
```

---

### List Active Sessions

```bash
curl -s $GW/api/v1/auth/sessions \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"
```

**200 OK:**
```json
{
  "sessions": [
    {
      "id":"sess_abc123",
      "ip":"192.168.1.100",
      "user_agent":"Mozilla/5.0...",
      "created_at":"2024-01-15T10:00:00Z",
      "last_active":"2024-01-15T10:30:00Z"
    }
  ]
}
```

### Logout All Devices

```bash
curl -s -X POST $GW/api/v1/auth/logout-all \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"
```

---

## Identity Service (Users)

### Create User

```bash
curl -s -X POST $GW/api/v1/users \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username":"john.smith",
    "email":"john@example.com",
    "password":"SecurePass@123",
    "phone":"+1234567890",
    "display_name":"John Smith",
    "locale":"en-US",
    "timezone":"America/New_York"
  }'
```

**201 Created:**
```json
{
  "id":"a1b2c3d4-...",
  "tenant_id":"00000000-...",
  "username":"john.smith",
  "email":"john@example.com",
  "status":"active",
  "email_verified":false,
  "created_at":"2024-01-15T10:00:00Z"
}
```

---

### List Users

```bash
# Basic list
curl -s "$GW/api/v1/users?page_size=10" \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"

# Search by name/email
curl -s "$GW/api/v1/users?search=john&page_size=20" \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"
```

**200 OK:**
```json
{
  "users": [...],
  "total": 42,
  "next_offset": 10
}
```

---

### Get User by ID

```bash
curl -s $GW/api/v1/users/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"
```

**404 Not Found:**
```json
{"error":"user not found","code":"NOT_FOUND"}
```

---

### Update User

```bash
curl -s -X PATCH $GW/api/v1/users/a1b2c3d4-... \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "email":"new.email@example.com",
    "phone":"+9876543210",
    "display_name":"John A. Smith"
  }'
```

---

### Delete User

```bash
curl -s -X DELETE $GW/api/v1/users/a1b2c3d4-... \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"
```

**200 OK:** `{"status":"deleted"}`

---

### Lock / Unlock / Activate / Deactivate

```bash
USER_ID="a1b2c3d4-..."

curl -s -X POST $GW/api/v1/users/$USER_ID/lock \
  -H "$AUTH" -H "X-Tenant-ID: $TENANT"

curl -s -X POST $GW/api/v1/users/$USER_ID/unlock \
  -H "$AUTH" -H "X-Tenant-ID: $TENANT"

curl -s -X POST $GW/api/v1/users/$USER_ID/deactivate \
  -H "$AUTH" -H "X-Tenant-ID: $TENANT"

curl -s -X POST $GW/api/v1/users/$USER_ID/activate \
  -H "$AUTH" -H "X-Tenant-ID: $TENANT"
```

---

### Bulk Import (CSV)

```bash
cat > /tmp/users.csv << 'EOF'
username,email,password,display_name
alice,alice@example.com,SecurePass@123,Alice Wang
bob,bob@example.com,SecurePass@456,Bob Chen
EOF

curl -s -X POST $GW/api/v1/users/import \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: text/csv" \
  --data-binary @/tmp/users.csv
```

**200 OK:**
```json
{"imported":2,"failed":0,"errors":[]}
```

---

## Policy Service (Roles, Permissions, Policies)

### Create Role

```bash
curl -s -X POST $GW/api/v1/roles \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "key":"editor",
    "name":"Content Editor",
    "description":"Can edit and publish content"
  }'
```

**201 Created:**
```json
{
  "id":"role-uuid-here",
  "tenant_id":"00000000-...",
  "key":"editor",
  "name":"Content Editor",
  "description":"Can edit and publish content"
}
```

**409 Conflict:** `{"error":"role key already exists"}`

---

### List Roles

```bash
curl -s "$GW/api/v1/roles?tenant_id=$TENANT" \
  -H "$AUTH"
```

---

### Get / Delete Role

```bash
ROLE_ID="role-uuid-here"

curl -s $GW/api/v1/roles/$ROLE_ID \
  -H "$AUTH"

curl -s -X DELETE $GW/api/v1/roles/$ROLE_ID \
  -H "$AUTH"
```

---

### Add Permission to Role

```bash
curl -s -X POST $GW/api/v1/roles/$ROLE_ID/permissions \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"resource":"documents:drafts","action":"write"}'
```

---

### Set Parent Role (Hierarchy)

```bash
curl -s -X POST $GW/api/v1/roles/$ROLE_ID/parent \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"parent_role_id":"parent-role-uuid"}'
```

---

### Create Policy

```bash
curl -s -X POST $GW/api/v1/policies \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name":"Engineering Read Access",
    "effect":"allow",
    "actions":["read"],
    "resources":["documents:*"],
    "conditions":{"department":"engineering"}
  }'
```

---

### Check Permission

```bash
curl -s -X POST $GW/api/v1/policies/check \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "user_id":"a1b2c3d4-...",
    "resource":"documents:sensitive",
    "action":"read"
  }'
```

**200 OK:**
```json
{
  "allowed":true,
  "reason":"Role 'editor' grants 'read' on 'documents:*'"
}
```

---

### Export / Import Policies

```bash
# Export all policies as JSON
curl -s "$GW/api/v1/policies/export?tenant_id=$TENANT" \
  -H "$AUTH" > policies_backup.json

# Import policies
curl -s -X POST $GW/api/v1/policies/import \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d @policies_backup.json
```

---

## Org Service

### Create Organization

```bash
curl -s -X POST $GW/api/v1/orgs \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Engineering","description":"Engineering Division"}'
```

**201 Created:**
```json
{
  "id":"org-uuid-here",
  "tenant_id":"00000000-...",
  "name":"Engineering",
  "description":"Engineering Division"
}
```

---

### List Organizations

```bash
curl -s "$GW/api/v1/orgs?tenant_id=$TENANT" \
  -H "$AUTH"
```

---

### Get / Update / Delete Org

```bash
ORG_ID="org-uuid-here"

# Get
curl -s $GW/api/v1/orgs/$ORG_ID -H "$AUTH"

# Update
curl -s -X PUT $GW/api/v1/orgs/$ORG_ID \
  -H "$AUTH" -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Engineering & Platform"}'

# Delete
curl -s -X DELETE $GW/api/v1/orgs/$ORG_ID -H "$AUTH"
```

---

### Add Member to Org

```bash
curl -s -X POST $GW/api/v1/orgs/$ORG_ID/members \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"user_id":"user-uuid","title":"Senior Engineer"}'
```

---

### List Org Members

```bash
curl -s "$GW/api/v1/orgs/$ORG_ID/members?tenant_id=$TENANT" \
  -H "$AUTH"
```

---

### Get Org Tree (Sub-organizations)

```bash
curl -s "$GW/api/v1/orgs/$ORG_ID/tree?tenant_id=$TENANT" \
  -H "$AUTH"
```

---

## Audit Service

### Query Audit Events

```bash
# All events (last 24h implied by default ordering)
curl -s "$GW/api/v1/audit/events?tenant_id=$TENANT&page_size=10" \
  -H "$AUTH"

# Filter by action
curl -s "$GW/api/v1/audit/events?tenant_id=$TENANT&action=user.login" \
  -H "$AUTH"

# Filter by result and time range
curl -s "$GW/api/v1/audit/events?tenant_id=$TENANT&result=failure&start_time=2024-01-15T00:00:00Z&end_time=2024-01-15T23:59:59Z" \
  -H "$AUTH"

# Filter by actor
curl -s "$GW/api/v1/audit/events?tenant_id=$TENANT&actor_id=user-uuid-here" \
  -H "$AUTH"
```

**200 OK:**
```json
{
  "events":[
    {
      "id":"event-uuid",
      "tenant_id":"00000000-...",
      "actor_id":"user-uuid",
      "actor_name":"jane.doe",
      "action":"user.login",
      "result":"success",
      "resource_type":"auth",
      "ip_address":"192.168.1.100",
      "created_at":"2024-01-15T10:30:00Z"
    }
  ],
  "total":156
}
```

---

### Get Single Event

```bash
curl -s $GW/api/v1/audit/events/event-uuid \
  -H "$AUTH" \
  -H "X-Tenant-ID: $TENANT"
```

---

### Audit Statistics

```bash
curl -s "$GW/api/v1/audit/stats?tenant_id=$TENANT" \
  -H "$AUTH"
```

**200 OK:**
```json
{
  "total_events": 1542,
  "events_by_action": {"user.login": 450, "role.create": 12},
  "events_by_result": {"success": 1480, "failure": 62},
  "hourly_distribution": [{"hour":"2024-01-15T10:00:00Z","count":45}],
  "top_actors": [{"actor_id":"...","actor_name":"admin","count":230}]
}
```

---

### Export Audit Events (CSV)

```bash
curl -s "$GW/api/v1/audit/export?tenant_id=$TENANT" \
  -H "$AUTH" > audit_export.csv
```

---

### Update Retention

```bash
curl -s -X PUT $GW/api/v1/audit/retention \
  -H "$AUTH" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"days":180,"enabled":true}'
```

---

## OAuth / OIDC

### JWKS Endpoint

```bash
curl -s $GW/.well-known/jwks.json
```

### OIDC Discovery

```bash
curl -s $GW/.well-known/openid-configuration
```

### OAuth Authorize (Browser Redirect)

```
https://iam.example.com/oauth/authorize?
  response_type=code&
  client_id=YOUR_CLIENT_ID&
  redirect_uri=https://yourapp.com/callback&
  scope=openid%20profile%20email&
  state=random_state_string
```

### OAuth Token Exchange

```bash
curl -s -X POST $GW/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=AUTH_CODE&client_id=YOUR_ID&client_secret=YOUR_SECRET"
```

---

## Error Response Reference

All errors follow the same format:

| Status | Meaning | Example Cause |
|--------|---------|---------------|
| 400 | Bad Request | Invalid JSON, missing required field |
| 401 | Unauthorized | Missing/expired JWT |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 405 | Method Not Allowed | Wrong HTTP method for endpoint |
| 409 | Conflict | Duplicate username/email/role key |
| 429 | Too Many Requests | Rate limited (auth endpoints) |
| 500 | Internal Error | Server bug (check logs) |
| 502 | Bad Gateway | Backend service down |
| 503 | Service Unavailable | Backend unhealthy |
