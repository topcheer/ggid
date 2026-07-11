# Keycloak Migration Guide

> Step-by-step migration from Keycloak to GGID.

---

## Migration Overview

| Keycloak Concept | GGID Equivalent |
|-----------------|-----------------|
| Realm | Tenant |
| Client | OAuth Client / API Key |
| Realm Role | Role (with permissions) |
| Client Role | Scoped Role |
| Group | Organization |
| User Federation | LDAP Provider |
| Identity Provider | Social Login / SAML IdP |

---

## Step 1: Export Users from Keycloak

```bash
# Export realm users to JSON
/opt/keycloak/bin/kcadm.sh get \
  realms/myrealm/users \
  -r myrealm > keycloak-users.json
```

## Step 2: Transform to GGID Format

```python
import json

kc_users = json.load(open('keycloak-users.json'))
ggid_users = []
for u in kc_users:
    ggid_users.append({
        'username': u['username'],
        'email': u.get('email', ''),
        'display_name': f"{u.get('firstName','')} {u.get('lastName','')}",
        'status': 'active' if u.get('enabled', True) else 'disabled',
    })

json.dump(ggid_users, open('ggid-users.json', 'w'), indent=2)
```

## Step 3: Import to GGID via SCIM

```bash
for user in $(jq -c '.[]' ggid-users.json); do
  curl -X POST http://localhost:8080/scim/v2/Users \
    -H "Authorization: Bearer $ADMIN_JWT" \
    -H "X-Tenant-ID: $TENANT" \
    -H "Content-Type: application/scim+json" \
    -d "$user"
done
```

## Step 4: Migrate Roles

```bash
# Create equivalent roles in GGID
curl -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -d '{"name":"Admin","key":"admin"}'
```

## Step 5: Update Application Code

Replace Keycloak adapter with GGID SDK:

- `keycloak.protect('role')` → `requireRole('role')`
- `req.kauth.grant.access_token` → `getClaims(req).sub`
- Realm config → GGID `GGIDProvider`

---

## Step 6: Migrate SAML/OIDC IdPs

Reconfigure each external IdP to point to GGID instead of Keycloak:
- Update SP entity ID to GGID's
- Update ACS URL to `https://ggid.example.com/saml/acs`
- Import IdP metadata via per-tenant IdP config

---

*See: [SDK Migration Guide](sdk-migration-guide.md) | [Social Login Setup](social-login-setup.md) | [Migration from Keycloak](../migration-from-keycloak.md)*

*Last updated: 2025-07-11*
