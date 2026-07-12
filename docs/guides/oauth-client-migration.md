# OAuth Client Migration Guide

Migrate OAuth clients from Auth0, Okta, Keycloak to GGID — metadata mapping, redirect URI migration, credential rotation, scope mapping, and phased cutover.

## Migration Sources

| Source | Tenant Model | Client Config Location |
|--------|-------------|----------------------|
| Auth0 | Per-tenant | Management API |
| Okta | Per-org | Admin Console / API |
| Keycloak | Per-realm | realm JSON / admin API |
| Cognito | Per-user-pool | AWS API |

## Metadata Mapping

### Auth0 → GGID

| Auth0 Field | GGID Field | Notes |
|-------------|-----------|-------|
| `client_id` | New ID generated | Can't preserve |
| `client_secret` | New secret generated | Rotate during migration |
| `name` | `client_name` | Direct |
| `callbacks` | `redirect_uris` | Array mapping |
| `allowed_logout_urls` | `post_logout_redirect_uris` | |
| `scopes` | `scope` | Map to GGID scope names |
| `grant_types` | `grant_types` | Direct |
| `token_endpoint_auth_method` | `token_endpoint_auth_method` | Direct |
| `is_first_party` | `trusted` | true → trusted client |

### Okta → GGID

| Okta Field | GGID Field | Notes |
|------------|-----------|-------|
| `client_id` | New ID | |
| `client_secret` | New secret | Rotate |
| `label` | `client_name` | |
| `redirect_uris` | `redirect_uris` | Direct |
| `response_types` | `response_types` | Direct |
| `grant_types` | `grant_types` | Direct |
| ` scopes` | `scope` | Okta custom → GGID scope |

### Keycloak → GGID

| Keycloak Field | GGID Field | Notes |
|---------------|-----------|-------|
| `clientId` | New ID | |
| `secret` | New secret | Rotate |
| `name` | `client_name` | |
| `redirectUris` | `redirect_uris` | |
| `webOrigins` | CORS config | |
| `standardFlowEnabled` | `response_types: [code]` | If true |
| `directGrantsEnabled` | `grant_types: [password]` | If true |
| `serviceAccountsEnabled` | `grant_types: [client_credentials]` | If true |

## Scope Mapping

```yaml
scope_mapping:
  # Auth0 scopes
  "profile": "profile"
  "email": "email"
  "openid": "openid"
  "read:users": "users:read"
  "write:users": "users:write"
  "delete:users": "users:delete"
  
  # Okta scopes
  "okta.users.read": "users:read"
  "okta.users.manage": "users:write"
  "okta.groups.read": "roles:read"
  
  # Keycloak roles → GGID scopes
  "realm-admin": "admin:tenant"
  "view-realm": "users:read"
```

## Redirect URI Migration

```bash
# Export redirect URIs from source
# Auth0
GET https://TENANT.auth0.com/api/v2/clients
# Okta
GET https://ORG.okta.com/api/v1/apps
# Keycloak
GET https://KC/auth/admin/realms/REALM/clients

# Import to GGID
POST /api/v1/oauth/register
{
  "client_name": "Migrated App",
  "redirect_uris": ["https://app.example.com/callback"],
  ...
}
```

### Redirect URI Checklist

- [ ] All URIs are HTTPS (except localhost dev)
- [ ] No wildcards (exact match only)
- [ ] Update application config to new GGID endpoints
- [ ] Test each redirect URI after migration

## Credential Rotation

```bash
# Step 1: Register client in GGID (new credentials)
POST /api/v1/oauth/register
# → {client_id: "ggid-123", client_secret: "ggid-secret"}

# Step 2: Update application config (dual-write period)
# Application tries new GGID first, falls back to old provider
config:
  primary:
    issuer: "https://auth.ggid.dev"
    client_id: "ggid-123"
    client_secret: "ggid-secret"
  fallback:
    issuer: "https://TENANT.auth0.com"
    client_id: "auth0-old"
    client_secret: "auth0-old-secret"

# Step 3: Monitor fallback usage
# Step 4: After 7 days with zero fallback usage, remove old config
```

## Phased Cutover

### Phase 1: Parallel (Week 1-2)

```
Application → GGID (new clients configured, not used)
Application → Auth0 (existing, serving traffic)
```

- Register clients in GGID
- Configure redirect URIs
- Test in staging environment

### Phase 2: Canary (Week 3)

```
10% of users → GGID login
90% of users → Auth0 login
```

```nginx
# Split traffic by cookie or header
map $cookie_ab_test $auth_upstream {
  "ggid"   auth.ggid.dev;
  default  tenant.auth0.com;
}
```

### Phase 3: Majority (Week 4)

```
90% of users → GGID
10% of users → Auth0 (rollback capability)
```

### Phase 4: Complete (Week 5)

```
100% → GGID
Auth0 clients deactivated (not deleted, keep for 30 days)
```

### Phase 5: Cleanup (Week 8)

```
Auth0 account suspended
Redirect URIs removed
Credentials securely deleted
```

## JWT Migration

Tokens issued by old provider remain valid until expiry. During cutover:

```go
func verifyToken(token string) (Claims, error) {
    // Try GGID first
    if claims, err := ggidVerify(token); err == nil {
        return claims, nil
    }
    // Fallback to old provider (during migration only)
    if claims, err := auth0Verify(token); err == nil {
        return claims, nil
    }
    return nil, ErrInvalidToken
}
```

## Rollback Plan

```bash
# If migration fails, revert DNS + config
# Step 1: Route all traffic back to old provider
kubectl patch configmap app-config -p '{"data":{"AUTH_PROVIDER":"auth0"}}'

# Step 2: Rolling restart
kubectl rollout restart deployment/app

# Step 3: GGID clients remain configured but unused (no data loss)
```

## Monitoring During Migration

| Metric | Target | Alert |
|--------|--------|-------|
| Login success rate | >98% | <95% → pause migration |
| Fallback usage | Decreasing | Not decreasing → stuck clients |
| JWT verification errors | <1% | Spike → JWKS sync issue |
| Token refresh failures | <0.5% | Spike → client misconfigured |

## See Also

- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [OAuth Dynamic Client Registration](oauth-dynamic-client-registration.md)
- [Competitive Analysis](competitive-analysis.md)
- [Keycloak Migration Guide](../research/keycloak-migration.md)
