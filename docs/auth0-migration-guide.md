# Auth0 Migration Guide

Step-by-step guide for migrating from Auth0 to GGID. Covers user export/import,
rule-to-policy mapping, social connector configuration, JWKS migration, and
a phased cutover plan.

---

## Table of Contents

- [Overview](#overview)
- [Phase 1: User Export and Import](#phase-1-user-export-and-import)
- [Phase 2: Rule-to-Policy Mapping](#phase-2-rule-to-policy-mapping)
- [Phase 3: Social Connector Configuration](#phase-3-social-connector-configuration)
- [Phase 4: JWKS Migration](#phase-4-jwks-migration)
- [Phase 5: Cutover Plan](#phase-5-cutover-plan)
- [Post-Migration Checklist](#post-migration-checklist)

---

## Overview

| Aspect | Auth0 | GGID |
|--------|-------|------|
| Hosting | SaaS | Self-hosted (Docker/K8s) |
| User store | Auth0 managed | PostgreSQL (yours) |
| JWT signing | Auth0 keys | Your RS256/EdDSA keys |
| Rules | JavaScript (Auth0 pipeline) | Policy engine + webhooks |
| Connections | Auth0 Connections | Social connectors + LDAP |
| Pricing | Per MAU | Free (open source) |

---

## Phase 1: User Export and Import

### Export from Auth0 Management API

```python
#!/usr/bin/env python3
"""Export Auth0 users for GGID import."""
import requests, json, sys

AUTH0_DOMAIN = sys.argv[1]  # tenant.auth0.com
MGMT_TOKEN = sys.argv[2]

headers = {"Authorization": f"Bearer {MGMT_TOKEN}"}
users = []
page = 0

while True:
    resp = requests.get(
        f"https://{AUTH0_DOMAIN}/api/v2/users",
        headers=headers,
        params={"per_page": 100, "page": page, "include_totals": True}
    )
    data = resp.json()
    batch = data.get("users", [])
    if not batch:
        break
    users.extend(batch)
    print(f"Exported {len(users)}/{data.get('total', '?')} users...", file=sys.stderr)
    if len(users) >= data.get("total", 0):
        break
    page += 1

# Convert to GGID format
ggid_users = []
for u in users:
    email = u.get("email", "")
    ggid_users.append({
        "username": u.get("username") or u.get("nickname") or email,
        "email": email,
        "name": u.get("name", ""),
        "status": "suspended" if u.get("blocked") else "active",
        "email_verified": u.get("email_verified", False),
        "require_password_reset": True,  # Can't migrate Auth0 hashes
        "roles": [],
    })

with open("ggid-users-import.json", "w") as f:
    json.dump(ggid_users, f, indent=2)

print(f"Wrote {len(ggid_users)} users to ggid-users-import.json")
```

### Import to GGID

```bash
TENANT_ID="00000000-0000-0000-0000-000000000001"
API="http://localhost:8080"
JWT=$(curl -s -X POST $API/api/v1/auth/login \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"admin","password":"AdminPass123!"}' | jq -r '.access_token')

# Import each user
jq -c '.[]' ggid-users-import.json | while read user; do
  curl -s -X POST $API/api/v1/users \
    -H "Authorization: Bearer $JWT" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "Content-Type: application/json" \
    -d "$user"
done

echo "Import complete. Send password reset emails:"
curl -s -X POST $API/api/v1/auth/password-reset/bulk \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID"
```

---

## Phase 2: Rule-to-Policy Mapping

Auth0 Rules are JavaScript functions in the auth pipeline. Map them to GGID
policy engine rules and webhooks:

| Auth0 Rule Pattern | GGID Equivalent |
|--------------------|-----------------|
| Add custom JWT claim | Custom claims config |
| Role-based access | RBAC policy engine |
| IP allowlist | Gateway IP filter middleware |
| Call external API | Webhook on `auth.login` |
| Email domain restriction | ABAC policy (deny domain) |

### Example: Custom Claims

**Auth0 Rule:**
```javascript
function(user, context, callback) {
    context.accessToken['https://app.com/department'] = user.user_metadata.department;
    callback(null, user, context);
}
```

**GGID:**
```bash
curl -X PUT $API/api/v1/settings/jwt-claims \
  -H "Authorization: Bearer $JWT" \
  -d '{"custom_claims": {"department": "{{user.department}}"}}'
```

### Example: Email Domain Restriction

**Auth0 Rule:**
```javascript
function(user, context, callback) {
    if (!user.email.endsWith('@company.com')) {
        return callback(new UnauthorizedError('Access denied'));
    }
    callback(null, user, context);
}
```

**GGID Policy:**
```bash
curl -X POST $API/api/v1/policies \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "name": "deny-external-email",
    "effect": "deny",
    "conditions": [{
      "attribute": "email",
      "operator": "not_ends_with",
      "value": "@company.com"
    }],
    "applies_to": ["auth.login"]
  }'
```

---

## Phase 3: Social Connector Configuration

Map Auth0 Connections to GGID social connectors:

| Auth0 Connection | GGID Connector | Config |
|------------------|---------------|--------|
| google-oauth2 | Google | Client ID + Secret |
| github | GitHub | Client ID + Secret |
| windowslive | Microsoft | Client ID + Secret |
| facebook | Facebook | Client ID + Secret |
| linkedin | LinkedIn | Client ID + Secret |
| apple | Apple | Team ID + Key |

### Migrate Social Connection

```bash
# Configure Google connector in GGID
curl -X PUT $API/api/v1/settings/auth-providers \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "google": {
      "enabled": true,
      "client_id": "YOUR_GOOGLE_CLIENT_ID",
      "client_secret": "YOUR_GOOGLE_CLIENT_SECRET",
      "scopes": ["openid", "email", "profile"]
    }
  }'
```

> **Note:** Social provider Client IDs and Secrets from Auth0 work directly
> in GGID. No need to re-register with Google/GitHub/etc.

---

## Phase 4: JWKS Migration

Applications verifying Auth0 JWTs must point to GGID's JWKS endpoint.

### Update JWKS URL

| Setting | Auth0 | GGID |
|---------|-------|------|
| JWKS URL | `https://tenant.auth0.com/.well-known/jwks.json` | `https://iam.example.com/.well-known/jwks.json` |
| Issuer | `https://tenant.auth0.com/` | `https://iam.example.com` |
| Algorithm | RS256 | RS256 |

### Next.js Example

**Before (Auth0):**
```typescript
import { jwtVerify, createRemoteJWKSet } from 'jose';

const JWKS = createRemoteJWKSet(new URL('https://tenant.auth0.com/.well-known/jwks.json'));
const { payload } = await jwtVerify(token, JWKS, { issuer: 'https://tenant.auth0.com/' });
```

**After (GGID):**
```typescript
const JWKS = createRemoteJWKSet(new URL('https://iam.example.com/.well-known/jwks.json'));
const { payload } = await jwtVerify(token, JWKS, { issuer: 'https://iam.example.com' });
```

---

## Phase 5: Cutover Plan

### Phased Cutover (Zero Downtime)

```
Week 1: Preparation
├── Export Auth0 users → Import to GGID
├── Configure social connectors
├── Map rules to policies
├── Deploy GGID (not receiving traffic yet)
└── Send password reset emails

Week 2: Dual-Run (Both Auth0 + GGID active)
├── New registrations → GGID
├── Existing users still on Auth0
├── Verify GGID login flows work
└── Monitor for issues

Week 3: Migration
├── Auth0 login page redirects to GGID
├── All new sessions use GGID JWTs
├── Auth0 handles old (un-expired) sessions
└── Update all apps to verify GGID JWKS

Week 4: Decommission
├── All sessions now GGID
├── Disable Auth0 tenant
├── Keep Auth0 data export as backup
└── Remove Auth0 SDK from codebase
```

### Rollback Plan

If issues arise during cutover:

1. Revert DNS/load balancer to point to Auth0
2. Auth0 still handles old sessions (never deleted)
3. Users who migrated to GGID need password reset on Auth0
4. Fix GGID issues, re-attempt cutover

---

## Post-Migration Checklist

- [ ] Export Auth0 users
- [ ] Import users to GGID (with password reset)
- [ ] Configure all social connectors
- [ ] Map all Auth0 Rules to GGID policies/webhooks
- [ ] Deploy GGID infrastructure
- [ ] Update JWKS URL in all consuming applications
- [ ] Update issuer claim in JWT verification
- [ ] Test login flows (password, social, MFA)
- [ ] Test RBAC policy enforcement
- [ ] Run dual-run period (1 week minimum)
- [ ] Switch DNS/load balancer to GGID
- [ ] Monitor error rates for 48 hours
- [ ] Disable Auth0 tenant
- [ ] Remove Auth0 SDK from codebase
- [ ] Delete Auth0 data export (after retention period)

---

## References

- [Migration from Keycloak](./migration-from-keycloak.md)
- [Migration from Clerk](./migration-from-clerk.md)
- [API Reference](./api-reference.md)
- [SDK Guide](./sdk-guide.md)
