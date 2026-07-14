# OAuth Scope Tiering

Identity scopes vs API scopes vs admin scopes, tier-based consent, scope packages, scope inheritance, least privilege defaults, and migration from flat scopes.

## Scope Tiers

### Tier 1: Identity Scopes (OIDC Standard)

| Scope | Claims Released | Consent | Risk |
|-------|----------------|---------|------|
| `openid` | `sub` | Required (implicit) | None |
| `profile` | `display_name`, `first_name`, `last_name`, `locale`, `timezone` | User | Low |
| `email` | `email`, `email_verified` | User | Low |
| `phone` | `phone_number`, `phone_verified` | User | Low |
| `address` | `formatted_address` | User | Low |

### Tier 2: API Scopes (Data Access)

| Scope | Access | Consent | Risk |
|-------|--------|---------|------|
| `users:read` | List/view users | User | Medium |
| `users:write` | Create/modify users | User + admin | Medium |
| `roles:read` | View roles | User | Low |
| `roles:assign` | Assign roles to users | Admin | High |
| `groups:read` | View groups | User | Low |
| `groups:write` | Manage groups | Admin | Medium |

### Tier 3: Admin Scopes

| Scope | Access | Consent | Risk |
|-------|--------|---------|------|
| `users:delete` | Delete users | Admin + CISO | Critical |
| `users:admin` | All user ops | Admin | Critical |
| `policy:admin` | Modify policies | Admin | Critical |
| `admin:tenant` | Tenant management | Super admin | Critical |
| `admin:super` | Everything | Dual approval | Max |

## Scope Packages

Pre-bundled scope sets for common app types:

```yaml
scope_packages:
  - name: "read-only-dashboard"
    scopes: ["openid", "profile", "email", "users:read", "roles:read"]
    tier: 2
    consent: "user"

  - name: "user-management"
    scopes: ["openid", "profile", "email", "users:read", "users:write", "roles:assign"]
    tier: 3
    consent: "admin"

  - name: "audit-viewer"
    scopes: ["openid", "audit:read"]
    tier: 2
    consent: "admin"

  - name: "full-admin"
    scopes: ["openid", "profile", "users:admin", "policy:admin"]
    tier: 3
    consent: "admin+ciso"
```

### Usage

```bash
# Client requests a package instead of individual scopes
GET /authorize?scope=package:read-only-dashboard&...
# → GGID expands to: openid profile email users:read roles:read
```

## Scope Inheritance

```
users:admin → implies → users:write → implies → users:read
roles:admin → implies → roles:assign → implies → roles:read
```

```go
func resolveEffectiveScopes(granted []string) []string {
    effective := make(map[string]bool)

    for _, scope := range granted {
        effective[scope] = true
        // Add all parent-implied scopes
        for _, implied := range scopeTree[scope] {
            effective[implied] = true
        }
    }

    return keys(effective)
}
```

### Scope Tree

```yaml
scope_hierarchy:
  "users:admin": ["users:write", "users:read", "users:delete"]
  "users:write": ["users:read"]
  "users:delete": ["users:read"]

  "roles:admin": ["roles:assign", "roles:read"]
  "roles:assign": ["roles:read"]

  "admin:tenant": ["users:admin", "roles:admin", "policy:admin"]
  "admin:super": ["admin:tenant", "audit:read", "audit:export"]
```

## Tier-Based Consent

| Tier | Consent Model | TTL |
|------|--------------|-----|
| Tier 1 (Identity) | User, one-time | 1 year |
| Tier 2 (API) | User, per-scope | 6 months |
| Tier 3 (Admin) | Admin approval required | 3 months |

### Consent Escalation

```bash
# Client has Tier 1, requests Tier 2
GET /authorize?scope=openid+users:read
# → Consent screen shows: "This app wants to read your user directory"
# → User approves → Tier 2 scope added
```

```bash
# Client has Tier 2, requests Tier 3
GET /authorize?scope=openid+users:write
# → Consent screen: "This app wants to modify users — requires admin approval"
# → Request queued for admin review
# → Admin approves → scope granted with 3-month TTL
```

## Least Privilege Defaults

### Default Scope Assignment

```yaml
default_scopes:
  new_client:
    tier: 1
    scopes: ["openid", "profile", "email"]

  authenticated_client:
    tier: 2
    scopes: ["openid", "profile", "email", "users:read"]

  admin_approved_client:
    tier: 2
    scopes: ["openid", "profile", "email", "users:read", "users:write"]
```

### Scope Reduction at Runtime

```bash
# Client authorized for: openid users:read users:write
# For a read-only operation, request narrowed scope:
POST /api/v1/oauth/token
{
  "grant_type": "authorization_code",
  "code": "...",
  "scope": "openid users:read"  // Narrowed
}
# → Token has only users:read (even though users:write is authorized)
```

## Migration from Flat Scopes

### Before (Flat)

```yaml
# All scopes at same tier, no hierarchy
old_scopes: ["read", "write", "admin"]
```

### After (Tiered)

```yaml
# Migration mapping
migration_map:
  "read": "users:read"           # Tier 2
  "write": "users:write"         # Tier 2
  "admin": "users:admin"         # Tier 3

# Old tokens accepted during grace period
# New tokens use tiered scopes
```

### Migration Timeline

```
Phase 1: Deploy tiered scopes alongside flat (both work)
Phase 2: Console warning for clients using flat scopes
Phase 3: Flat scopes deprecated (logged but still work)
Phase 4: Flat scopes rejected (clients must migrate)
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Tier 3 scope requests | Track for unusual patterns |
| Scope escalation attempts | Blocked + logged |
| Unused authorized scopes | Flag for cleanup (>90 days unused) |
| Consent denial rate per tier | High → UX or trust issue |

## See Also

- [OAuth Client Scoped Permissions](oauth-client-scoped-permissions.md)
- [OAuth Scope Design](oauth-scope-design.md)
- [Consent Management Design](consent-management-design.md)
- [OAuth 2.1 Compliance Checklist](oauth-2-1-compliance-checklist.md)
