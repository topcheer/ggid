# GGID Migration Guide

How to migrate from Auth0, Keycloak, or other IAM platforms to GGID.

---

## Table of Contents

- [Migration Overview](#migration-overview)
- [Phase 1: Assessment](#phase-1-assessment)
- [Phase 2: Export Source Data](#phase-2-export-source-data)
- [Phase 3: Transform & Import](#phase-3-transform--import)
- [Phase 4: SSO / Federation Migration](#phase-4-sso--federation-migration)
- [Phase 5: Cutover](#phase-5-cutover)
- [Auth0 → GGID](#auth0--ggid)
- [Keycloak → GGID](#keycloak--ggid)
- [API Endpoint Mapping](#api-endpoint-mapping)
- [Post-Migration Validation](#post-migration-validation)

---

## Migration Overview

```
┌─────────────┐    Export     ┌──────────────┐    Transform    ┌───────────┐
│  Source IAM  │ ──────────► │  JSON / CSV   │ ──────────────► │   GGID    │
│  (Auth0/KC) │              │  Intermediate │                 │  Import   │
└─────────────┘              └──────────────┘                 └───────────┘
```

### Timeline

| Phase | Duration (1K users) | Duration (10K users) |
|-------|--------------------|--------------------|
| Assessment | 1 day | 2 days |
| Export | 1 hour | 4 hours |
| Transform | 2 hours | 1 day |
| Import | 30 minutes | 2 hours |
| Validation | 1 day | 2 days |
| SSO migration | 1 day | 2 days |
| Cutover | 1 hour | 2 hours |

---

## Phase 1: Assessment

### Inventory Your Current Setup

Document before migrating:

- [ ] Total users count
- [ ] User attributes used (email, phone, custom fields)
- [ ] Authentication methods (password, social, LDAP, SAML)
- [ ] Roles and permissions structure
- [ ] SSO/OAuth clients (client_id, redirect URIs, grant types)
- [ ] SAML federation / IdP connections
- [ ] MFA enrollment status
- [ ] Custom claims / hooks / rules / actions
- [ ] API endpoints your application calls

### Compatibility Check

| Feature | Auth0 | Keycloak | GGID |
|---------|-------|----------|------|
| Password login | ✅ | ✅ | ✅ |
| Social login (Google, GitHub) | ✅ | ✅ | ✅ |
| LDAP/AD | ✅ | ✅ | ✅ |
| SAML 2.0 SP | ✅ | ✅ | ✅ |
| OIDC IdP | ✅ | ✅ | ✅ |
| MFA TOTP | ✅ | ✅ | ✅ |
| WebAuthn/Passkey | ✅ | ❌ | ✅ |
| RBAC | ✅ (Actions) | ✅ | ✅ |
| ABAC | ❌ | ❌ | ✅ |
| SCIM 2.0 | ✅ | ✅ | ✅ |
| Multi-tenant | ✅ (Organizations) | ✅ (Realms) | ✅ (RLS) |

---

## Phase 2: Export Source Data

### From Auth0

#### Export Users

```bash
# Use the Auth0 Management API
# Requires: npm install -g auth0-deploy-cli

# Export configuration
cat > config.json << 'EOF'
{
  "AUTH0_DOMAIN": "YOUR tenant.auth0.com",
  "AUTH0_CLIENT_ID": "YOUR_M2M_CLIENT_ID",
  "AUTH0_CLIENT_SECRET": "YOUR_M2M_CLIENT_SECRET"
}
EOF

# Export all users
a0cli export users --config config.json --output users.json
```

Or via Management API directly:

```bash
# Get all users (paginated)
curl "https://YOUR_TENANT.auth0.com/api/v2/users?per_page=100&page=0" \
  -H "Authorization: Bearer YOUR_MGMT_API_TOKEN" \
  | jq '.[] | {
      user_id, email, email_verified, name,
      nickname, given_name, family_name,
      created_at, last_login, app_metadata, user_metadata
    }' > auth0_users.json
```

#### Export Roles

```bash
curl "https://YOUR_TENANT.auth0.com/api/v2/roles" \
  -H "Authorization: Bearer YOUR_MGMT_API_TOKEN" \
  | jq '.[] | {id, name, description, permissions}' > auth0_roles.json
```

#### Export Connections (SSO)

```bash
curl "https://YOUR_TENANT.auth0.com/api/v2/connections" \
  -H "Authorization: Bearer YOUR_MGMT_API_TOKEN" \
  | jq '.[] | {name, strategy, options}' > auth0_connections.json
```

### From Keycloak

#### Export Realm Data

```bash
# Using Keycloak CLI (inside the Keycloak container)
docker exec keycloak /opt/keycloak/bin/kc.sh export \
  --dir /tmp/export \
  --realm my-realm \
  --users realm_file

# Or via REST API
curl "http://localhost:8080/admin/realms/my-realm/users" \
  -H "Authorization: Bearer $KEYCLOAK_TOKEN" > kc_users.json

curl "http://localhost:8080/admin/realms/my-realm/roles" \
  -H "Authorization: Bearer $KEYCLOAK_TOKEN" > kc_roles.json
```

#### Export Full Realm (JSON)

```bash
# Full realm export includes users, roles, clients, identity providers
docker exec keycloak /opt/keycloak/bin/kc.sh export \
  --dir /tmp/export \
  --realm my-realm \
  --users same_file

# Copy the export out
docker cp keycloak:/tmp/export/my-realm-realm.json ./keycloak_realm.json
```

---

## Phase 3: Transform & Import

### User Data Transformation

#### Auth0 → GGID Format

**Auth0 user JSON:**
```json
{
  "user_id": "auth0|abc123",
  "email": "user@example.com",
  "email_verified": true,
  "name": "John Doe",
  "created_at": "2023-01-15T10:00:00.000Z",
  "app_metadata": { "roles": ["admin"] },
  "user_metadata": { "department": "engineering" }
}
```

**Transform to GGID format:**
```json
{
  "username": "user@example.com",
  "email": "user@example.com",
  "display_name": "John Doe",
  "status": "active",
  "email_verified": true,
  "password": "TEMPORARY_PASSWORD_MUST_RESET",
  "tenant_id": "00000000-0000-0000-0000-000000000001"
}
```

**Transform script (Python):**
```python
import json

with open('auth0_users.json') as f:
    auth0_users = json.load(f)

ggid_users = []
for u in auth0_users:
    ggid_users.append({
        "username": u.get("email", u.get("user_id")),
        "email": u["email"],
        "display_name": u.get("name", u.get("nickname", "")),
        "status": "active",
        "email_verified": u.get("email_verified", False),
        "password": "TempPass@123Reset",  # forces password reset on first login
        "tenant_id": "00000000-0000-0000-0000-000000000001"
    })

with open('ggid_users_import.json', 'w') as f:
    json.dump(ggid_users, f, indent=2)

print(f"Transformed {len(ggid_users)} users")
```

#### Keycloak → GGID Format

**Keycloak user JSON:**
```json
{
  "id": "kc-uuid-here",
  "username": "jdoe",
  "email": "jdoe@example.com",
  "firstName": "John",
  "lastName": "Doe",
  "enabled": true,
  "emailVerified": true,
  "attributes": {"department": ["engineering"]}
}
```

**Transform:**
```json
{
  "username": "jdoe",
  "email": "jdoe@example.com",
  "display_name": "John Doe",
  "status": "active",
  "email_verified": true,
  "password": "TempPass@123Reset"
}
```

### Import Users to GGID

```bash
TOKEN="your-admin-jwt"
GW="http://localhost:8080"
TENANT="00000000-0000-0000-0000-000000000001"

# Method 1: Bulk import via CSV
python3 transform_users.py  # converts to CSV
curl -X POST "$GW/api/v1/users/import" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: text/csv" \
  --data-binary @ggid_users.csv

# Method 2: Individual registration (for small batches)
for user in $(jq -c '.[]' ggid_users_import.json); do
  curl -s -X POST "$GW/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT" \
    -d "$user"
done
```

### Role Mapping

#### Auth0 → GGID

| Auth0 Concept | GGID Equivalent |
|---------------|-----------------|
| Role | Role (key = Auth0 role name) |
| Permission | Role permission (resource:action) |
| App Metadata `roles` | Role assignment via API |

```bash
# Create roles from Auth0 export
for role in $(jq -c '.[]' auth0_roles.json); do
  ROLE_NAME=$(echo $role | jq -r .name)
  ROLE_DESC=$(echo $role | jq -r .description)

  curl -s -X POST "$GW/api/v1/roles" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT" \
    -d "{
      \"key\": \"$(echo $ROLE_NAME | tr ' ' '_' | tr '[:upper:]' '[:lower:]')\",
      \"name\": \"$ROLE_NAME\",
      \"description\": \"$ROLE_DESC\"
    }"
done
```

#### Keycloak → GGID

| Keycloak Concept | GGID Equivalent |
|------------------|-----------------|
| Realm Role | Role |
| Client Role | Role (prefixed with client name) |
| Composite Role | Role hierarchy (parent_role_id) |
| Group | Organization |
| Group Membership | Org membership |

```bash
# Map Keycloak groups to GGID organizations
for group in $(jq -c '.[]' kc_groups.json); do
  GROUP_NAME=$(echo $group | jq -r .name)

  curl -s -X POST "$GW/api/v1/orgs" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT" \
    -d "{\"name\": \"$GROUP_NAME\"}"
done
```

### Password Migration

**Option A: Force Password Reset (Recommended)**

Set all imported users with a temporary password. On first login, GGID
prompts for a password change:

```json
{
  "password": "TempPass@123Reset",
  "require_change": true
}
```

**Option B: Migrate Password Hashes (Advanced)**

If you have access to password hashes:

1. Write a custom migration script that inserts directly into PostgreSQL:

```sql
-- Auth0 uses bcrypt; GGID uses Argon2id
-- Passwords cannot be directly converted between hash algorithms

-- Instead, store the old hash temporarily and do lazy migration:
INSERT INTO credentials (id, tenant_id, user_id, type, secret, created_at)
VALUES (
  gen_random_uuid(),
  '00000000-0000-0000-0000-000000000001',
  'user-uuid-here',
  'password',
  'BCRYPT_HASH_HERE',   -- old hash, verified on first login
  NOW()
);
```

2. On first login, verify against the old hash. If successful, re-hash with
   Argon2id and update the record:

```sql
UPDATE credentials SET secret = 'NEW_ARGON2ID_HASH' WHERE user_id = '...';
```

> **Note:** GGID uses Argon2id by default. Auth0 uses bcrypt. Keycloak uses
> PBKDF2. None are directly compatible, so lazy migration is the only option
> that preserves passwords.

---

## Phase 4: SSO / Federation Migration

### Social Login Connections

| Provider | Auth0 Config | GGID Config |
|----------|-------------|-------------|
| Google | Connection (strategy: google-oauth2) | `AUTH_GOOGLE_CLIENT_ID` / `AUTH_GOOGLE_CLIENT_SECRET` |
| GitHub | Connection (strategy: github) | `AUTH_GITHUB_CLIENT_ID` / `AUTH_GITHUB_CLIENT_SECRET` |
| Microsoft | Connection (strategy: microsoft) | `AUTH_MICROSOFT_CLIENT_ID` / `AUTH_MICROSOFT_CLIENT_SECRET` |

```bash
# Set social login credentials as environment variables in Auth service
AUTH_GOOGLE_CLIENT_ID=your_google_client_id
AUTH_GOOGLE_CLIENT_SECRET=your_google_client_secret
AUTH_GOOGLE_REDIRECT_URL=https://iam.example.com/api/v1/auth/social/google/callback
```

### SAML Federation

Migrate SAML IdP connections:

```bash
# Create IdP federation config in GGID
curl -X POST "$GW/api/v1/idp/config" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Corporate ADFS",
    "protocol": "saml",
    "metadata_url": "https://adfs.corp.example.com/FederationMetadata/2007-06/FederationMetadata.xml"
  }'
```

### OAuth Client Migration

Migrate OAuth clients from Auth0/Keycloak:

| Auth0/Keycloak | GGID |
|----------------|------|
| Application (client_id + client_secret) | OAuth client registration |
| Callback URLs | Redirect URIs |
| Scopes | OIDC scopes (openid profile email) |

```bash
# Register OAuth client in GGID
curl -X POST "$GW/api/v1/oauth/clients" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "My Web App",
    "redirect_uris": ["https://myapp.com/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "scopes": ["openid", "profile", "email"]
  }'
```

---

## Phase 5: Cutover

### Cutover Strategy

1. **Freeze writes** on the source IAM (maintenance window)
2. **Export final delta** (users created since last export)
3. **Import delta** into GGID
4. **Update DNS** to point to GGID Gateway
5. **Update application config** to use new JWKS URL
6. **Test login flow** end-to-end
7. **Announce** migration complete

### Rollback Plan

Keep the source IAM running for 48 hours as fallback:

1. If critical issues arise, revert DNS to the source IAM
2. Users who haven't logged in since cutover are unaffected
3. Users who changed data in GGID may need re-sync

---

## Auth0 → GGID

### Quick Migration Script

```bash
#!/bin/bash
# migrate-from-auth0.sh
set -euo pipefail

AUTH0_DOMAIN="${1:?Usage: $0 <auth0_domain> <mgmt_token> <ggid_gateway> <ggid_token>}"
MGMT_TOKEN="$2"
GW="$3"
GGID_TOKEN="$4"
TENANT="00000000-0000-0000-0000-000000000001"

echo "=== Auth0 → GGID Migration ==="

# Step 1: Export users from Auth0
echo "[1/4] Exporting users..."
curl -s "https://$AUTH0_DOMAIN/api/v2/users?per_page=100&page=0" \
  -H "Authorization: Bearer $MGMT_TOKEN" > auth0_users_raw.json

# Step 2: Transform to GGID format
echo "[2/4] Transforming users..."
python3 -c "
import json, sys
users = json.load(open('auth0_users_raw.json'))
output = []
for u in users:
    output.append({
        'username': u.get('email', ''),
        'email': u['email'],
        'display_name': u.get('name', ''),
        'password': 'TempPass@123Reset',
        'tenant_id': '$TENANT'
    })
json.dump(output, open('ggid_import.json', 'w'), indent=2)
print(f'Transformed {len(output)} users')
"

# Step 3: Import to GGID
echo "[3/4] Importing users..."
SUCCESS=0; FAIL=0
for user in $(jq -c '.[]' ggid_import.json); do
  STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GW/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT" \
    -d "$user")
  if [ "$STATUS" = "201" ] || [ "$STATUS" = "409" ]; then
    SUCCESS=$((SUCCESS + 1))
  else
    FAIL=$((FAIL + 1))
  fi
done
echo "  Imported: $SUCCESS, Failed: $FAIL"

# Step 4: Export & import roles
echo "[4/4] Migrating roles..."
curl -s "https://$AUTH0_DOMAIN/api/v2/roles" \
  -H "Authorization: Bearer $MGMT_TOKEN" | \
  jq -c '.[]' | while read -r role; do
    KEY=$(echo "$role" | jq -r '.name' | tr ' ' '_' | tr '[:upper:]' '[:lower:]')
    NAME=$(echo "$role" | jq -r '.name')
    curl -s -X POST "$GW/api/v1/roles" \
      -H "Authorization: Bearer $GGID_TOKEN" \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: $TENANT" \
      -d "{\"key\":\"$KEY\",\"name\":\"$NAME\"}"
  done

echo "=== Migration complete ==="
```

---

## Keycloak → GGID

### Quick Migration Script

```bash
#!/bin/bash
# migrate-from-keycloak.sh
set -euo pipefail

KC_URL="${1:?Usage: $0 <keycloak_url> <kc_admin> <kc_pass> <ggid_gateway> <ggid_token>}"
KC_REALM="$2"
KC_ADMIN="$3"
KC_PASS="$4"
GW="$5"
GGID_TOKEN="$6"
TENANT="00000000-0000-0000-0000-000000000001"

echo "=== Keycloak → GGID Migration ==="

# Get admin token
KC_TOKEN=$(curl -s -X POST "$KC_URL/realms/master/protocol/openid-connect/token" \
  -d "grant_type=password&username=$KC_ADMIN&password=$KC_PASS&client_id=admin-cli" \
  | jq -r .access_token)

# Export users
echo "[1/3] Exporting users..."
curl -s "$KC_URL/admin/realms/$KC_REALM/users?max=1000" \
  -H "Authorization: Bearer $KC_TOKEN" > kc_users.json

# Transform & import
echo "[2/3] Importing users..."
jq -c '.[]' kc_users.json | while read -r u; do
  USERNAME=$(echo "$u" | jq -r '.username')
  EMAIL=$(echo "$u" | jq -r '.email // empty')
  FIRST=$(echo "$u" | jq -r '.firstName // empty')
  LAST=$(echo "$u" | jq -r '.lastName // empty')

  curl -s -X POST "$GW/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT" \
    -d "{
      \"username\": \"$USERNAME\",
      \"email\": \"${EMAIL:-$USERNAME@example.com}\",
      \"password\": \"TempPass@123Reset\",
      \"display_name\": \"$FIRST $LAST\"
    }" > /dev/null
done

# Export & map roles
echo "[3/3] Migrating roles..."
curl -s "$KC_URL/admin/realms/$KC_REALM/roles" \
  -H "Authorization: Bearer $KC_TOKEN" | \
  jq -c '.[]' | while read -r role; do
    NAME=$(echo "$role" | jq -r '.name')
    KEY=$(echo "$NAME" | tr '[:upper:]' '[:lower:]')
    curl -s -X POST "$GW/api/v1/roles" \
      -H "Authorization: Bearer $GGID_TOKEN" \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: $TENANT" \
      -d "{\"key\":\"$KEY\",\"name\":\"$NAME\"}" > /dev/null
done

echo "=== Migration complete ==="
```

---

## API Endpoint Mapping

### Auth0 → GGID

| Auth0 API | GGID API |
|-----------|----------|
| `POST /api/v2/users` | `POST /api/v1/users` |
| `GET /api/v2/users` | `GET /api/v1/users` |
| `GET /api/v2/users/{id}` | `GET /api/v1/users/{id}` |
| `PATCH /api/v2/users/{id}` | `PATCH /api/v1/users/{id}` |
| `DELETE /api/v2/users/{id}` | `DELETE /api/v1/users/{id}` |
| `POST /api/v2/roles` | `POST /api/v1/roles` |
| `GET /api/v2/roles` | `GET /api/v1/roles` |
| `POST /api/v2/users/{id}/roles` | Assign via `POST /api/v1/users/{id}` (role_ids) |
| `POST /oauth/token` (login) | `POST /api/v1/auth/login` |
| `POST /oauth/token` (refresh) | `POST /api/v1/auth/refresh` |
| `POST /dbconnections/signup` | `POST /api/v1/auth/register` |
| `POST /api/v2/tickets/password-reset` | `POST /api/v1/auth/password/forgot` |
| `GET /.well-known/jwks.json` | `GET /.well-known/jwks.json` |

### Keycloak → GGID

| Keycloak API | GGID API |
|--------------|----------|
| `POST /admin/realms/{r}/users` | `POST /api/v1/users` |
| `GET /admin/realms/{r}/users` | `GET /api/v1/users` |
| `PUT /admin/realms/{r}/users/{id}` | `PATCH /api/v1/users/{id}` |
| `DELETE /admin/realms/{r}/users/{id}` | `DELETE /api/v1/users/{id}` |
| `POST /admin/realms/{r}/roles` | `POST /api/v1/roles` |
| `GET /admin/realms/{r}/roles` | `GET /api/v1/roles` |
| `POST /realms/{r}/protocol/openid-connect/token` | `POST /api/v1/auth/login` |
| `POST /realms/{r}/protocol/openid-connect/token` (refresh) | `POST /api/v1/auth/refresh` |
| `GET /realms/{r}/protocol/openid-connect/certs` | `GET /.well-known/jwks.json` |

---

## Post-Migration Validation

### Test Checklist

- [ ] All imported users can log in with temporary password
- [ ] Password reset flow works for imported users
- [ ] Role assignments match source IAM
- [ ] Social login (Google/GitHub) works
- [ ] SAML federation works (if applicable)
- [ ] OAuth clients can complete authorization code flow
- [ ] JWKS endpoint returns valid keys
- [ ] Audit events are being recorded
- [ ] Token refresh works
- [ ] MFA enrollment works for imported users

### Verification Script

```bash
#!/bin/bash
# verify-migration.sh
set -euo pipefail

GW="http://localhost:8080"
TENANT="00000000-0000-0000-0000-000000000001"

echo "=== Migration Verification ==="

# Count imported users
USER_COUNT=$(curl -s "$GW/api/v1/users?page_size=1" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  | jq '.total')
echo "Total users: $USER_COUNT"

# Count roles
ROLE_COUNT=$(curl -s "$GW/api/v1/roles?tenant_id=$TENANT" \
  -H "Authorization: Bearer $TOKEN" | jq '.roles | length')
echo "Total roles: $ROLE_COUNT"

# Test login with a known user
echo "Testing login..."
LOGIN=$(curl -s -X POST "$GW/api/v1/auth/login" \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"test_user","password":"TempPass@123Reset"}')

if echo "$LOGIN" | jq -e '.access_token' > /dev/null 2>&1; then
  echo "✓ Login successful"
else
  echo "✗ Login failed: $(echo $LOGIN | jq -r .error)"
fi

# Verify JWKS
echo "Verifying JWKS..."
JWKS=$(curl -s "$GW/.well-known/jwks.json")
if echo "$JWKS" | jq -e '.keys[0]' > /dev/null 2>&1; then
  echo "✓ JWKS endpoint returns valid keys"
else
  echo "✗ JWKS endpoint error"
fi

echo "=== Verification complete ==="
```

---

## Clerk → GGID

### Export Users from Clerk

Clerk provides a Backend API for user export:

```bash
# Export all users via Clerk Backend API
curl -s "https://api.clerk.com/v1/users?limit=500" \
  -H "Authorization: Bearer $CLERK_SECRET_KEY" \
  | jq '.' > clerk_users.json

# Paginate if needed
curl -s "https://api.clerk.com/v1/users?limit=500&offset=500" \
  -H "Authorization: Bearer $CLERK_SECRET_KEY" \
  | jq '.' >> clerk_users_all.json
```

### Clerk User → GGID Transform

```python
#!/usr/bin/env python3
"""Transform Clerk users to GGID import format."""

import json, hashlib

with open("clerk_users.json") as f:
    clerk = json.load(f)

ggid_users = []
for u in clerk:
    # Clerk stores password hashes in a non-standard format
    # Users must reset password on first login
    primary_email = next((e["email_address"] for e in u.get("email_addresses", []) if e.get("id") == u.get("primary_email_address_id")), "")

    ggid_users.append({
        "username": u.get("username") or primary_email,
        "email": primary_email,
        "first_name": u.get("first_name", ""),
        "last_name": u.get("last_name", ""),
        "active": "active" if u.get("active", True) else "suspended",
        "external_id": u["id"],  # Clerk user ID for reference
        "metadata": {
            "migrated_from": "clerk",
            "clerk_id": u["id"],
            "created_at": u.get("created_at"),
        }
    })

with open("ggid_import.json", "w") as f:
    json.dump({"users": ggid_users}, f, indent=2)

print(f"Transformed {len(ggid_users)} users")
```

### Clerk Role Mapping

| Clerk Role | GGID Role Key | GGID Permissions |
|------------|---------------|------------------|
| `org:admin` | `admin` | `*` (all) |
| `org:member` | `member` | `read:users`, `read:orgs` |
| (custom) | Map by name | Configure per requirements |

### Clerk OAuth Client Migration

```bash
# Clerk social connections → GGID OAuth providers
# For each social provider in Clerk:
# 1. Get provider config from Clerk dashboard
# 2. Set the same Client ID / Secret in GGID

# Google
ggid-env OAUTH_GOOGLE_CLIENT_ID=$GOOGLE_CID
ggid-env OAUTH_GOOGLE_CLIENT_SECRET=$GOOGLE_SECRET

# GitHub
ggid-env OAUTH_GITHUB_CLIENT_ID=$GITHUB_CID
ggid-env OAUTH_GITHUB_CLIENT_SECRET=$GITHUB_SECRET

# Update redirect URI in provider dashboard:
#   Clerk:  https://your-app.clerk.accounts.dev/oauth/callback
#   GGID:   https://api.your-domain.com/oauth/callback/google
```

### Clerk → GGID Feature Mapping

| Clerk Feature | GGID Equivalent | Notes |
|---------------|-----------------|-------|
| User Management | Identity Service | Full CRUD + SCIM 2.0 |
| Authentication | Auth Service | JWT, bcrypt, MFA |
| Sessions | Redis sessions | TTL-based, revocable |
| Organizations | Org Service | Tree structure, departments, teams |
| RBAC | Policy Service | RBAC + ABAC engine |
| Webhooks | Gateway Webhooks | HMAC-signed, retry, dead-letter |
| JWT Templates | Custom Claims | Per-tenant JWT customization |
| Social Login | pkg/social | 9 connectors (Google, GitHub, etc.) |
| MFA (TOTP) | Auth MFA | TOTP + WebAuthn |
| B2B (Organizations) | Multi-tenant | RLS isolation per tenant |

---

## Pre-Migration Checklist

Complete this checklist before starting any migration:

### Infrastructure

- [ ] GGID deployed and healthy (all containers/services running)
- [ ] Database backups configured and tested
- [ ] DNS record for new GGID instance (e.g., `iam.example.com`)
- [ ] TLS certificate provisioned for new domain
- [ ] Load balancer configured with health checks
- [ ] Monitoring and alerting in place (Prometheus + Grafana)

### User Data

- [ ] Export user directory from source system (CSV/JSON/SCIM)
- [ ] Verify user count and data integrity
- [ ] Map source attributes to GGID schema (email, name, phone, department)
- [ ] Identify service accounts and API keys
- [ ] Document password reset strategy (force reset on first login)

### Role & Group Mapping

- [ ] Inventory all roles/groups in source system
- [ ] Create GGID roles matching source roles
- [ ] Document group-to-role mapping table
- [ ] Plan orphaned role cleanup (roles with no users)

### Application Inventory

- [ ] List all apps using the source IdP
- [ ] For each app: document SAML/OIDC config, redirect URIs, cert
- [ ] Identify apps that can be migrated independently
- [ ] Identify apps with hard dependencies (must migrate together)
- [ ] Test SAML SP metadata import for each app

### Signing Key Strategy

- [ ] Export JWT signing key from source system (if possible)
- [ ] Plan key overlap period (dual-key acceptance window)
- [ ] Document fallback if tokens signed with old key fail

### Rollback Plan

- [ ] Define rollback criteria (error rate threshold, login success rate)
- [ ] Keep source IdP running in read-only mode for 7 days
- [ ] Document DNS cutover reversal procedure
- [ ] Test rollback in staging environment

---

## JWT Signing Key Migration

### Strategy: Dual-Key Overlap

During migration, both the old and new signing keys are accepted. This allows
existing tokens (signed by the source system) to remain valid until they
expire, while new tokens are signed by GGID.

```
Phase 1 (Pre-cutover):
  Source IdP signs with Key-A
  GGID accepts Key-A (imported) + Key-B (new)

Phase 2 (Cutover):
  GGID signs with Key-B
  Source IdP disabled
  GGID still accepts Key-A for lingering tokens

Phase 3 (Post-migration):
  GGID accepts only Key-B
  Key-A removed after all tokens expired
```

### Importing External Signing Keys

```bash
# Export public key from source IdP (e.g., Auth0)
# Auth0: Settings → Advanced → Signing Key → Download Certificate

# Register in GGID for dual-key acceptance
curl -X POST https://iam.example.com/api/v1/admin/oauth/signing-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "key=@auth0-signing-cert.pem" \
  -F "status=accept_only"
```

### Key Overlap Configuration

```yaml
oauth:
  jwt:
    # Primary signing key (GGID's own)
    signing_key: "/etc/ggid/keys/ggid-signing.key"
    signing_cert: "/etc/ggid/keys/ggid-signing.crt"
    
    # Additional accepted keys (for migration overlap)
    accepted_keys:
      - cert: "/etc/ggid/keys/auth0-signing.crt"
        label: "auth0-migration"
        remove_after: "2024-02-15T00:00:00Z"  # Auto-remove date
      - cert: "/etc/ggid/keys/keycloak-signing.crt"
        label: "keycloak-migration"
        remove_after: "2024-02-15T00:00:00Z"
```

---

## Phased Cutover Strategy

### Phase 1: Parallel (Days 1-7)

```
                    ┌──────────────────┐
  User Login ──────►│  Load Balancer    │
                    │  (weighted:       │
                    │   90% → Source    │
                    │   10% → GGID)     │
                    └───┬──────────┬───┘
                        │          │
                 90%    ▼    10%   ▼
              ┌──────────┐    ┌──────────┐
              │ Source IdP│    │   GGID   │
              │ (active)  │    │ (shadow) │
              └──────────┘    └──────────┘
```

- Both systems active simultaneously
- 10% of traffic routed to GGID
- Monitor error rates, login success rates
- Both systems share user directory (LDAP or synced)

### Phase 2: Majority Cutover (Days 8-14)

```
Load Balancer: 80% → GGID, 20% → Source
```

- Most users now authenticate via GGID
- Source IdP kept as fallback
- Migrate one application group at a time

### Phase 3: Full Cutover (Day 15)

```
Load Balancer: 100% → GGID
```

- All traffic through GGID
- Source IdP kept running in read-only mode (rollback safety)
- Users who haven't logged in are force-reset

### Phase 4: Decommission (Day 22)

- Source IdP shut down
- Remove dual-key acceptance for old signing keys
- Final data reconciliation

### Application Migration Groups

Migrate apps in dependency order:

| Group | Apps | When |
|-------|------|------|
| 1 | Internal tools (lowest risk) | Phase 1 |
| 2 | Admin console, dashboards | Phase 2 |
| 3 | Customer-facing apps | Phase 2-3 |
| 4 | Critical integrations (billing, security) | Phase 3 |
