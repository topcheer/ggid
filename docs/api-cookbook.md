# GGID API Cookbook

20 essential curl recipes for integrating with the GGID Platform API.

> **Base URL**: `http://localhost:8080` (adjust for your deployment)  
> **Auth**: All authenticated endpoints require `Authorization: Bearer <token>`  
> **Tenant**: Most endpoints require `X-Tenant-ID: <tenant-uuid>`

---

## Quick Start

```bash
# Set these once per session
export GGID=http://localhost:8080
export TENANT=00000000-0000-0000-0000-000000000001
```

---

## 1. Register a New User

```bash
curl -sS -X POST "$GGID/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "Str0ng#Pass2024!"
  }'
```

## 2. Login (Get JWT Tokens)

```bash
curl -sS -X POST "$GGID/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "alice",
    "password": "Str0ng#Pass2024!"
  }'

# Save the token:
export TOKEN=$(curl -sS -X POST "$GGID/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"Str0ng#Pass2024!"}' | jq -r .access_token)
```

## 3. Refresh Access Token

```bash
curl -sS -X POST "$GGID/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<your-refresh-token>"}'
```

## 4. Create a User (Admin)

```bash
curl -sS -X POST "$GGID/api/v1/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "bob",
    "email": "bob@example.com",
    "first_name": "Bob",
    "last_name": "Smith",
    "status": "active"
  }'
```

## 5. List Users

```bash
curl -sS "$GGID/api/v1/users?page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

## 6. Create a Group

```bash
curl -sS -X POST "$GGID/api/v1/groups" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Engineering",
    "description": "Engineering team"
  }'
```

## 7. Create a Role

```bash
curl -sS -X POST "$GGID/api/v1/roles" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "app-developer",
    "description": "Application developer role",
    "scopes": ["app:read", "app:write"]
  }'
```

## 8. Assign Role to User

```bash
curl -sS -X POST "$GGID/api/v1/roles/assign" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "user_id": "<user-uuid>",
    "role_id": "<role-uuid>"
  }'
```

## 9. OAuth Authorize Flow

```bash
# Step 1: Redirect user to authorize URL (browser)
open "$GGID/api/v1/oauth/authorize?response_type=code&client_id=my-client&redirect_uri=http://localhost:3000/callback&scope=openid+profile&state=random123"

# Step 2: Exchange code for token
curl -sS -X POST "$GGID/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code" \
  -d "code=<auth-code-from-redirect>" \
  -d "redirect_uri=http://localhost:3000/callback" \
  -d "client_id=my-client" \
  -d "client_secret=<client-secret>"
```

## 10. Client Credentials Token

```bash
curl -sS -X POST "$GGID/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=my-service" \
  -d "client_secret=<client-secret>" \
  -d "scope=api:read"
```

## 11. Create an OAuth Client

```bash
curl -sS -X POST "$GGID/api/v1/oauth/clients" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "My Web App",
    "redirect_uris": ["http://localhost:3000/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "scopes": ["openid", "profile", "email"]
  }'
```

## 12. Begin WebAuthn Registration

```bash
curl -sS -X POST "$GGID/api/v1/auth/webauthn/begin" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"user_id": "<user-uuid>"}'
```

## 13. Finish WebAuthn Registration

```bash
curl -sS -X POST "$GGID/api/v1/auth/webauthn/finish" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "credential_id": "<base64url-credential-id>",
    "public_key": "<base64url-public-key>",
    "aaguid": "cb69481e-8ff7-4039-93ec-0a2729a154a8",
    "session_id": "<session-from-begin>"
  }'
```

## 14. Query Audit Events

```bash
curl -sS "$GGID/api/v1/audit/events?action=user.login&page_size=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

## 15. Evaluate a Policy

```bash
curl -sS -X POST "$GGID/api/v1/policies/evaluate" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "principal_id": "user-uuid",
    "resources": ["document:123"],
    "actions": ["read"]
  }'
```

## 16. Check Single Permission

```bash
curl -sS -X POST "$GGID/api/v1/policies/check" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "principal_id": "user-uuid",
    "resource": "document:123",
    "action": "read"
  }'
```

## 17. List Active Sessions

```bash
curl -sS "$GGID/api/v1/auth/sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

## 18. Revoke a Session

```bash
curl -sS -X DELETE "$GGID/api/v1/auth/sessions/<session-id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

## 19. Enroll MFA (TOTP)

```bash
curl -sS -X POST "$GGID/api/v1/auth/mfa/enroll" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"method": "totp"}'
```

## 20. Export Audit Events (CSV)

```bash
curl -sS "$GGID/api/v1/audit/export?format=csv&start=2025-01-01T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -o audit-export.csv
```

---

## Tips

- **Swagger UI**: Visit `http://localhost:8080/docs` for interactive API exploration with try-it-out.
- **OpenAPI Spec**: Download at `http://localhost:8080/swagger.json`.
- **OIDC Discovery**: `http://localhost:8080/.well-known/openid-configuration`.
- **JWKS**: `http://localhost:8080/.well-known/jwks.json` for JWT signature verification.
- **Password Strength**: Test passwords before registration via `POST /api/v1/auth/password/strength`.
