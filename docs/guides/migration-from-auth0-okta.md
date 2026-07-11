# Migration from Auth0 / Okta to GGID

This guide provides a complete migration path from Auth0 and Okta to GGID, including user export, data mapping, application reconfiguration, and cutover strategy.

> **Related**: [Keycloak Migration](keycloak-migration.md), [SDK Migration Guide](sdk-migration-guide.md), [Migration from Auth0](../migration-from-auth0.md)

## Pre-Migration Assessment

### Inventory Checklist

Before starting, inventory your current setup:

- [ ] List all applications using Auth0/Okta (client IDs, redirect URIs)
- [ ] List all social connections / identity providers
- [ ] List all SAML/OIDC enterprise IdP federations
- [ ] List all custom rules/actions/hooks
- [ ] Document user count and growth rate
- [ ] Export current role/permission model
- [ ] Identify API integrations (Management API usage)
- [ ] Document branding/customization (email templates, logos)

## Auth0 to GGID Migration

### Concept Mapping

| Auth0 Concept | GGID Equivalent | Notes |
|---------------|-----------------|-------|
| Tenant | Tenant | Same concept |
| Connection | Auth Provider | Social: Google, GitHub, Microsoft |
| Application | OAuth Client | Register with redirect URIs |
| API | Scope Set | Define resource scopes |
| Role | Role | RBAC roles with key field |
| Permission | Permission | `resource:action` format |
| Rule | Webhook / Auth Hook | Server-side logic |
| Action | Webhook | Event-driven |
| Management API | Admin API | `/api/v1/*` endpoints |
| User Metadata | User Attributes | JSONB column |
| Email Template | Branding Config | Per-tenant email templates |
| Guardian (MFA) | MFA Service | TOTP + WebAuthn |
| Log Streams | SIEM Forwarder | Splunk/Datadog/ES |

### Step 1: Export Users from Auth0

```bash
# Export all users via Auth0 Management API
curl "https://$TENANT.auth0.com/api/v2/users?per_page=100&page=0&include_totals=true" \
  -H "Authorization: Bearer $MGMT_TOKEN" | \
  jq '[.[] | {
    username: .email,
    email: .email,
    name: .name,
    created_at: .created_at,
    last_login: .last_login,
    app_metadata: .app_metadata,
    user_metadata: .user_metadata,
    identities: .identities
  }]' > auth0-users.json

# Paginate through all users
TOTAL_PAGES=$(jq '.total / 100 | ceil' auth0-users.json)
for i in $(seq 1 $((TOTAL_PAGES - 1))); do
  curl "https://$TENANT.auth0.com/api/v2/users?per_page=100&page=$i" \
    -H "Authorization: Bearer $MGMT_TOKEN" >> auth0-users-page-$i.json
done
```

### Step 2: Export Roles and Permissions

```bash
# Export roles
curl "https://$TENANT.auth0.com/api/v2/roles" \
  -H "Authorization: Bearer $MGMT_TOKEN" | \
  jq '[.[] | {key: .name, name: .name, description: .description}]' > auth0-roles.json

# Export permissions per role
for role_id in $(jq -r '.[].id' auth0-roles-raw.json); do
  curl "https://$TENANT.auth0.com/api/v2/roles/$role_id/permissions" \
    -H "Authorization: Bearer $MGMT_TOKEN" >> auth0-permissions.json
done
```

### Step 3: Import Users to GGID via SCIM

```bash
# Import users via SCIM 2.0 bulk
jq -c '.[]' auth0-users.json | while read user; do
  username=$(echo "$user" | jq -r '.username')
  email=$(echo "$user" | jq -r '.email')
  name=$(echo "$user" | jq -r '.name // .email')

  curl -X POST "https://api.ggid.example.com/scim/v2/Users" \
    -H "Authorization: Bearer $GGID_ADMIN_TOKEN" \
    -H "Content-Type: application/scim+json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{
      \"schemas\": [\"urn:ietf:params:scim:schemas:core:2.0:User\"],
      \"userName\": \"$username\",
      \"emails\": [{\"value\": \"$email\", \"primary\": true}],
      \"displayName\": \"$name\",
      \"password\": \"TempPassword123!\"
    }"
done

echo "Migration complete. Users must reset passwords on first login."
```

> **Note**: Auth0 stores passwords as bcrypt. GGID uses Argon2id. Passwords cannot be directly migrated — users must reset via the password reset flow.

### Step 4: Import Roles

```bash
# Create each role in GGID
jq -c '.[]' auth0-roles.json | while read role; do
  key=$(echo "$role" | jq -r '.key')
  name=$(echo "$role" | jq -r '.name')
  desc=$(echo "$role" | jq -r '.description // ""')

  curl -X POST "https://api.ggid.example.com/api/v1/roles" \
    -H "Authorization: Bearer $GGID_ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d "{\"key\": \"$key\", \"name\": \"$name\", \"description\": \"$desc\"}"
done
```

### Step 5: Migrate Social Connections

| Auth0 Connection | GGID Env Vars |
|-----------------|---------------|
| `google-oauth2` | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` |
| `github` | `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` |
| `windowslive` | `MICROSOFT_CLIENT_ID`, `MICROSEP_CLIENT_SECRET` |
| `apple` | `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID` |

### Step 6: Migrate Application Code

```javascript
// BEFORE: Auth0 SDK
import { Auth0Client } from '@auth0/auth0-spa-js';
const auth0 = new Auth0Client({
  domain: 'your-tenant.auth0.com',
  client_id: 'xxx',
  redirect_uri: window.location.origin,
});

// AFTER: GGID Node SDK
import { GGIDClient, authMiddleware } from '@ggid/node';
const client = new GGIDClient({
  gatewayUrl: 'https://api.ggid.example.com',
  tenantId: '00000000-0000-0000-0000-000000000001',
});
app.use('/api', authMiddleware(client.config));
```

```python
# BEFORE: Auth0 Python
from auth0.v3.authentication import GetToken
token = GetToken('your-tenant.auth0.com')

# AFTER: GGID Python (REST API)
import requests
resp = requests.post('https://api.ggid.example.com/api/v1/auth/login',
    json={'username': 'user@example.com', 'password': 'pass'},
    headers={'X-Tenant-ID': tenant_id})
token = resp.json()['access_token']
```

## Okta to GGID Migration

### Concept Mapping

| Okta Concept | GGID Equivalent | Notes |
|--------------|-----------------|-------|
| Org | Tenant | Top-level isolation |
| Application | OAuth Client | OIDC/OAuth apps |
| Group | Group / Org | User grouping |
| User Type | N/A | Schema extension (roadmap) |
| MFA Factor | MFA Device | TOTP, WebAuthn |
| Identity Provider | Auth Provider | SAML, OIDC, social |
| Authorization Server | OAuth Service | Token issuance |
| Access Policies | Policy Service | RBAC + ABAC |
| Profile Mapping | N/A | Direct field mapping |
| Lifecycle Policy | Access Request | Approval workflows |
| System Log | Audit Service | NATS + hash chain |
| API Token | API Key | Per-user service tokens |

### Step 1: Export Users from Okta

```bash
# Export all users
curl "https://$OKTA.okta.com/api/v1/users?limit=200" \
  -H "Authorization: SSWS $OKTA_API_TOKEN" | \
  jq '[.[] | {
    username: .profile.login,
    email: .profile.email,
    first_name: .profile.firstName,
    last_name: .profile.lastName,
    status: .status,
    created: .created,
    last_login: .lastLogin,
    groups: .credentials
  }]' > okta-users.json

# Paginate (cursor-based)
NEXT_URL=$(curl -sI "https://$OKTA.okta.com/api/v1/users?limit=200" \
  -H "Authorization: SSWS $OKTA_API_TOKEN" | grep -i link | grep next | \
  sed 's/.*<\(.*\)>;.*/\1/')
```

### Step 2: Export Groups and Assignments

```bash
# Export groups
curl "https://$OKTA.okta.com/api/v1/groups?limit=200" \
  -H "Authorization: SSWS $OKTA_API_TOKEN" | \
  jq '[.[] | select(.profile.name != "Everyone") | {
    name: .profile.name,
    description: .profile.description
  }]' > okta-groups.json

# Export group memberships
for group_id in $(curl -s "https://$OKTA.okta.com/api/v1/groups?limit=200" \
  -H "Authorization: SSWS $OKTA_API_TOKEN" | jq -r '.[].id'); do
  curl "https://$OKTA.okta.com/api/v1/groups/$group_id/users" \
    -H "Authorization: SSWS $OKTA_API_TOKEN" | \
    jq -r '.[].id' > "group-$group_id-members.txt"
done
```

### Step 3: Export SAML Apps

```bash
# List SAML apps
curl "https://$OKTA.okta.com/api/v1/apps?type=SAML_2_0" \
  -H "Authorization: SSWS $OKTA_API_TOKEN" | \
  jq '[.[] | {
    name: .name,
    label: .label,
    sso_url: .settings.signOn.ssoAcsUrl,
    issuer: .settings.signOn.issuer,
    cert: .credentials.signing.kid
  }]' > okta-saml-apps.json
```

### Step 4: Import to GGID

Same SCIM import process as Auth0 migration (Step 3 above).

## Cutover Strategy

### Option A: Big Bang

1. Export all users (maintenance window)
2. Import to GGID
3. Send password reset emails
4. Switch DNS to GGID
5. Deactivate Auth0/Okta

**Risk**: High — if GGID has issues, users can't authenticate.

### Option B: Phased Migration (Recommended)

1. **Week 1**: Deploy GGID alongside Auth0/Okta
2. **Week 2**: Migrate non-critical app
3. **Week 3**: Migrate 50% of users (random selection)
4. **Week 4**: Migrate remaining users
5. **Week 5**: Monitor, then deactivate old IdP

### Option C: Dual-Running with Gradual Cutover

1. Configure GGID as secondary IdP
2. New users go to GGID only
3. Existing users migrate on next password reset
4. After 90 days, deactivate old IdP

## Post-Migration Checklist

- [ ] All users imported and can reset password
- [ ] Social login providers reconfigured
- [ ] SAML apps reconfigured with GGID metadata
- [ ] Application code updated to GGID SDK
- [ ] Custom rules/actions migrated to webhooks
- [ ] MFA re-enrollment flow tested
- [ ] Email templates customized
- [ ] Audit logging verified (hash chain)
- [ ] SIEM forwarder reconfigured
- [ ] Old Auth0/Okta tenant kept active for 30 days (monitor)
- [ ] DNS TTL lowered before cutover
- [ ] Rollback plan documented

## See Also

- [Keycloak Migration](keycloak-migration.md)
- [SDK Migration Guide](sdk-migration-guide.md)
- [SCIM Provisioning](scim-provisioning.md)
- [SSO Configuration](sso-providers.md)
