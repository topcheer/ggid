# Migration from Auth0/Okta

> Step-by-step migration guide from Auth0 and Okta to GGID.

---

## Auth0 Migration

### Concept Mapping

| Auth0 | GGID |
|-------|------|
| Tenant | Tenant |
| Connection | Auth Provider |
| Application | OAuth Client |
| API | API (scope set) |
| Role | Role |
| Rule/Action | Auth Hook / Webhook |
| Management API | Admin API |

### Step 1: Export Users

```bash
# Auth0 Management API
curl https://$TENANT.auth0.com/api/v2/users \
  -H "Authorization: Bearer $MGMT_TOKEN" \
  | jq '[.[] | {username, email, name: .name, created_at}]' \
  > auth0-users.json
```

### Step 2: Import to GGID

```bash
for user in $(jq -c '.[]' auth0-users.json); do
  curl -X POST http://localhost:8080/scim/v2/Users \
    -H "Authorization: Bearer $GGID_JWT" \
    -H "Content-Type: application/scim+json" \
    -d "$user"
done
```

### Step 3: Migrate Social Connections

| Auth0 Connection | GGID Provider |
|-----------------|---------------|
| `google-oauth2` | `GOOGLE_*` env |
| `github` | `GITHUB_*` env |
| `windowslive` | `MICROSOFT_*` env |

### Step 4: Update App Code

```javascript
// Auth0
const auth0 = new Auth0Client({ domain, clientId });

// GGID
const { expressAuth } = require('@ggid/node');
app.use('/api', expressAuth({ jwksUrl, issuer }));
```

---

## Okta Migration

### Step 1: Export Users via Okta API

```bash
curl https://$OKTA.okta.com/api/v1/users \
  -H "Authorization: SSWS $OKTA_API_TOKEN" \
  | jq '[.[] | {username: .profile.login, email: .profile.email}]' \
  > okta-users.json
```
### Step 2-4: Same as Auth0 migration above.

---

## Post-Migration Checklist

- [ ] All users imported and can login
- [ ] Social login providers reconfigured
- [ ] App code updated to GGID SDK
- [ ] SAML/OIDC IdPs reconfigured
- [ ] Old Auth0/Okta tenant deactivated (monitor 30 days)

---

*See: [SDK Migration Guide](sdk-migration-guide.md) | [Keycloak Migration](keycloak-migration.md)*

*Last updated: 2025-07-11*
