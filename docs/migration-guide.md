# Migration Guide: Auth0 / Keycloak / Clerk → GGID

> Step-by-step migration from existing IAM platforms to GGID.

---

## 1. Concept Mapping

| Concept | Auth0 | Keycloak | Clerk | GGID |
|---------|-------|----------|-------|------|
| Tenant | Tenant | Realm | Instance | Tenant (UUID) |
| User | User | User | User | User (Identity Service) |
| Role | Role | Role | Role | Role (Policy Service) |
| Permission | Scope | Role/Scope | Permission | Permission (Policy Service) |
| Organization | Organization | Group | Organization | Organization (Org Service) |
| OAuth Client | Application | Client | Instance | OAuth Client (OAuth Service) |
| Hosted Login | Universal Login | Login Theme | SignIn | Hosted Login Pages (Gateway) |
| Token Format | JWT (RS256) | JWT (RS256) | JWT | JWT (RS256) |
| MFA | Guardian | OTP | MFA | TOTP + WebAuthn |
| Social Login | Connections | Identity Providers | Social | Social Connectors |
| Audit Log | Log Streams | Events | Audit | Audit Service (NATS) |
| SCIM | Provisioning | SCIM 2.0 | — | SCIM 2.0 (Identity Service) |

---

## 2. Auth0 Migration

### 2.1 Export Users

Use the Auth0 Management API:

```bash
curl "https://YOUR_DOMAIN/api/v2/users" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -o users.json
```

### 2.2 Import to GGID

```bash
# Transform and bulk import
curl -X POST http://localhost:8080/api/v1/users/import \
  -H "Authorization: Bearer $GGID_JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "file=@users.csv"
```

CSV format:
```csv
username,email,full_name
john.doe,john@example.com,John Doe
jane.smith,jane@example.com,Jane Smith
```

### 2.3 Key Differences

| Feature | Auth0 | GGID |
|---------|-------|------|
| Password hashing | PBKDF2 | bcrypt (cost 12) |
| Token lifetime | Configurable per API | Configurable per tenant |
| JWKS URL | `tenant.auth0.com/.well-known/jwks.json` | `your-host/.well-known/jwks.json` |
| Rate limits | Tenant-level | Per-tenant configurable |
| Rules | JavaScript | Go webhook hooks |
| Connections | Per-tenant | Social connectors per tenant |

> **Password migration**: Auth0 uses PBKDF2, GGID uses bcrypt. Send password-reset emails after migration.

### 2.4 Rules → Hooks

Auth0 Rules run JavaScript inline. GGID uses configurable webhook hooks:

```json
{
  "event": "post_login",
  "url": "https://your-app.com/webhooks/ggid-post-login",
  "method": "POST",
  "headers": { "Authorization": "Bearer WEBHOOK_SECRET" }
}
```

---

## 3. Keycloak Migration

### 3.1 Export Realm

```bash
# Keycloak admin CLI
/opt/keycloak/bin/kcadm.sh export \
  --realm my-realm \
  --dir ./export/
```

### 3.2 Transform and Import

```bash
# Keycloak JSON → GGID CSV
go run scripts/keycloak-to-ggid.go \
  --input export/my-realm-users-0.json \
  --output users.csv
```

### 3.3 Role Mapping

| Keycloak Role | GGID Equivalent |
|---------------|-----------------|
| `realm-admin` | `tenant_admin` |
| `user` | `member` |
| `manage-users` | `user_manager` |
| `view-realm` | `auditor` |

### 3.4 Client Migration

| Keycloak | GGID |
|----------|------|
| Confidential Client | OAuth Client (client_credentials) |
| Bearer-only Client | OAuth Client (service_account) |
| Public Client | OAuth Client (authorization_code + PKCE) |

---

## 4. Clerk Migration

### 4.1 Export via Clerk API

```bash
curl "https://api.clerk.com/v1/users" \
  -H "Authorization: Bearer sk_test_CLERK_KEY" \
  | jq '.data' > users.json
```

### 4.2 Key Differences

| Feature | Clerk | GGID |
|---------|-------|------|
| Multi-tenant | Instances | Tenant UUID + RLS |
| Organizations | Built-in | Org Service (tree structure) |
| session_token | JWT (RS256) | JWT (RS256) |
| Webhooks | Svix | Webhook system (HMAC delivery) |

---

## 5. DNS Cutover Strategy

### Phase 1: Parallel Run (1-2 weeks)

```
auth.example.com → Auth0 (existing)
auth-new.example.com → GGID (new, tested in parallel)
```

### Phase 2: Gradual Migration (1 week)

```
auth.example.com → Load Balancer → 90% Auth0, 10% GGID
```

### Phase 3: Full Cutover

```
auth.example.com → GGID (100%)
```

### Rollback Plan

Keep Auth0/Keycloak running for 30 days post-migration. DNS TTL: 300s (5 min) for quick rollback.

---

## 6. SDK Integration Changes

### Before (Auth0)

```javascript
import { Auth0Client } from '@auth0/auth0-spa-js';
const auth0 = new Auth0Client({
  domain: 'your-tenant.auth0.com',
  client_id: 'CLIENT_ID',
});
```

### After (GGID)

```javascript
import { GGIDClient } from '@ggid/node-sdk';
const ggid = new GGIDClient({
  baseUrl: 'https://auth.example.com',
  clientId: 'CLIENT_ID',
  tenantId: 'TENANT_UUID',
});
```

### Token validation

```javascript
// GGID Node.js middleware
import { ggidMiddleware } from '@ggid/node-sdk/middleware';
app.use(ggidMiddleware({
  jwksUri: 'https://auth.example.com/.well-known/jwks.json',
  tenantId: 'TENANT_UUID',
}));
```
