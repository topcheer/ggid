# Keycloak Migration Guide

Step-by-step guide for migrating from Keycloak to GGID. Covers realm export,
client migration, role mapping, and custom SPI replacement.

---

## Table of Contents

- [Overview](#overview)
- [Step 1: Export Keycloak Realm](#step-1-export-keycloak-realm)
- [Step 2: Convert to GGID Format](#step-2-convert-to-ggid-format)
- [Step 3: Migrate Clients](#step-3-migrate-clients)
- [Step 4: Migrate Roles](#step-4-migrate-roles)
- [Step 5: Migrate Users](#step-5-migrate-users)
- [Step 6: Replace Custom SPIs](#step-6-replace-custom-spis)
- [Step 7: Cutover](#step-7-cutover)

---

## Overview

| Aspect | Keycloak | GGID |
|--------|----------|------|
| Hosting | Self-hosted (JBoss/WildFly) | Self-hosted (Go binary, 30MB) |
| Realms | Multi-realm | Multi-tenant (RLS) |
| Clients | Keycloak Clients | OAuth Clients |
| Roles | Realm + Client roles | RBAC roles per tenant |
| User Federation | LDAP/AD Federation | LDAP Provider (built-in) |
| SPI | Java SPI plugins | Go plugin interfaces |
| Identity Brokering | OIDC/SAML brokering | Social connectors + SAML IdP |
| Admin Console | Keycloak Admin | GGID Admin Console (Next.js) |

---

## Step 1: Export Keycloak Realm

```bash
# Export realm with users (run on Keycloak server)
/opt/keycloak/bin/kc.sh export \
  --realm my-realm \
  --dir /tmp/realm-export \
  --users realm_file

# Output files:
# /tmp/realm-export/my-realm-realm.json
# /tmp/realm-export/my-realm-users-0.json
```

### Realm JSON Structure

```json
{
  "realm": "my-realm",
  "enabled": true,
  "users": [
    {
      "username": "john.doe",
      "email": "john@example.com",
      "enabled": true,
      "emailVerified": true,
      "firstName": "John",
      "lastName": "Doe",
      "realmRoles": ["user", "editor"],
      "clientRoles": { "my-app": ["app-admin"] }
    }
  ],
  "roles": {
    "realm": [
      { "name": "user" },
      { "name": "admin" },
      { "name": "editor" }
    ]
  },
  "clients": [
    {
      "clientId": "my-app",
      "enabled": true,
      "protocol": "openid-connect",
      "redirectUris": ["https://app.example.com/callback"],
      "secret": "******",
      "standardFlowEnabled": true,
      "serviceAccountsEnabled": true
    }
  ],
  "groups": [
    { "name": "Engineering", "subGroups": [{ "name": "Backend" }] }
  ]
}
```

---

## Step 2: Convert to GGID Format

```python
#!/usr/bin/env python3
"""Convert Keycloak realm export to GGID import format."""
import json, sys

def convert(kc):
    return {
        "format": "ggid-import-v1",
        "users": [{
            "username": u.get("username", u["email"]),
            "email": u.get("email", ""),
            "name": f"{u.get('firstName','')} {u.get('lastName','')}".strip(),
            "status": "active" if u.get("enabled", True) else "suspended",
            "require_password_reset": True,
            "roles": u.get("realmRoles", []),
        } for u in kc.get("users", [])],
        "roles": [{
            "key": r["name"],
            "name": r.get("description", r["name"]),
        } for r in kc.get("roles", {}).get("realm", [])],
        "oauth_clients": [{
            "name": c["clientId"],
            "client_id": c["clientId"],
            "client_secret": c.get("secret", ""),
            "redirect_uris": c.get("redirectUris", []),
            "grant_types": convert_grants(c),
        } for c in kc.get("clients", []) if c.get("enabled") and c["clientId"] not in
            ["account", "admin-cli", "broker", "realm-management", "security-admin-console"]],
        "organizations": [{
            "name": g["name"],
            "display_name": g["name"],
        } for g in kc.get("groups", [])],
    }

def convert_grants(c):
    grants = []
    if c.get("standardFlowEnabled"): grants.append("authorization_code")
    if c.get("directAccessGrantsEnabled"): grants.append("password")
    if c.get("serviceAccountsEnabled"): grants.append("client_credentials")
    return grants

if __name__ == "__main__":
    with open(sys.argv[1]) as f:
        data = json.load(f)
    json.dump(convert(data), sys.stdout, indent=2)
```

---

## Step 3: Migrate Clients

```bash
# Convert and import clients
python3 convert_kc.py my-realm-realm.json > ggid-import.json

# Create OAuth clients
jq -c '.oauth_clients[]' ggid-import.json | while read client; do
  curl -s -X POST $API/api/v1/oauth/clients \
    -H "Authorization: Bearer $JWT" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "Content-Type: application/json" \
    -d "$client"
done
```

### Client Secret Rotation

Keycloak client secrets cannot be extracted in plaintext. Generate new secrets
in GGID and update applications:

```bash
# Generate new secret
curl -s -X POST $API/api/v1/oauth/clients/$CLIENT_ID/rotate-secret \
  -H "Authorization: Bearer $JWT" \
  | jq -r '.client_secret'
```

---

## Step 4: Migrate Roles

| Keycloak | GGID |
|----------|------|
| `realmRoles` | RBAC roles (via `/api/v1/roles`) |
| `clientRoles` | Scoped roles per OAuth client |
| `compositeRoles` | Role hierarchy |
| `groups` | Organizations |

```bash
# Import roles
jq -c '.roles[]' ggid-import.json | while read role; do
  curl -s -X POST $API/api/v1/roles \
    -H "Authorization: Bearer $JWT" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "$role"
done
```

---

## Step 5: Migrate Users

Keycloak stores passwords as PBKDF2-SHA256. GGID uses Argon2id/bcrypt. Passwords
**cannot** be directly migrated.

### Strategy: Force Password Reset

```bash
# Import users (all will need password reset)
jq -c '.users[]' ggid-import.json | while read user; do
  curl -s -X POST $API/api/v1/users \
    -H "Authorization: Bearer $JWT" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "$user"
done

# Send password reset emails
curl -s -X POST $API/api/v1/auth/password-reset/bulk \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID"
```

---

## Step 6: Replace Custom SPIs

Keycloak Service Provider Interfaces (SPIs) map to GGID plugin types:

| Keycloak SPI | GGID Equivalent |
|--------------|----------------|
| `AuthenticationSpi` | Auth Provider Plugin |
| `UserStorageProvider` | LDAP Provider (built-in) |
| `EventListenerProvider` | Event Subscriber / Webhook |
| `RequiredActionProvider` | Login Flow Policy |
| `IdentityProvider` | Social Connector / SAML SP |
| `ThemeSelectorProvider` | Brand Customization |

---

## Step 7: Cutover

1. Deploy GGID alongside Keycloak (parallel run)
2. Update applications to use GGID JWKS (`/.well-known/jwks.json`)
3. Send password reset emails to all users
4. Switch DNS/load balancer from Keycloak to GGID
5. Monitor for 48 hours
6. Decommission Keycloak

### Migration Checklist

- [ ] Export Keycloak realm
- [ ] Run conversion script
- [ ] Create GGID tenant
- [ ] Import roles
- [ ] Import users (password reset)
- [ ] Create OAuth clients (rotate secrets)
- [ ] Import groups as organizations
- [ ] Configure LDAP (if Keycloak used federation)
- [ ] Map SPIs to GGID plugins
- [ ] Update JWKS URL in applications
- [ ] Test login flows
- [ ] Cutover DNS
- [ ] Decommission Keycloak

---

## References

- [Auth0 Migration](./auth0-migration-guide.md)
- [Clerk Migration](./migration-from-clerk.md)
- [API Reference](./api-reference.md)
- [Plugin Development](./plugin-development.md)
