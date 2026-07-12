# OAuth Client Scoped Permissions

Scope hierarchy, consent model, per-scope risk rating, dynamic registration scopes, admin vs user consent, and scope downgrade.

## Scope Hierarchy

GGID uses dot-notation scopes with hierarchical relationships:

```
users                  (meta-scope, never granted directly)
├── users:read         (read any user profile)
├── users:write        (create/update users)
├── users:delete       (delete users)
└── users:admin        (all user operations + admin actions)

openid                 (OIDC base)
├── profile            (display_name, locale, timezone)
├── email              (email, email_verified)
├── phone              (phone_number)
└── address            (formatted address)

roles
├── roles:read
├── roles:write
└── roles:assign       (assign/unassign roles to users)

policy
├── policy:read
├── policy:write
├── policy:evaluate    (dry-run evaluation)
└── policy:admin
```

### Scope Implication

```
users:admin implies → users:write implies → users:read
```

If a client requests `users:admin`, it implicitly gets all subordinate scopes.

## Per-Scope Risk Rating

| Risk Level | Scope Pattern | Consent Model | TTL Limit |
|-----------|--------------|---------------|-----------|
| Low | `*:read`, `openid`, `profile` | User consent | Standard (refresh token) |
| Medium | `*:write`, `roles:assign` | User consent + warning | Standard |
| High | `*:delete`, `*:admin`, `policy:admin` | Admin approval required | Max 8h |
| Critical | `admin:super`, `break-glass` | Dual approval + time-boxed | Max 30m |

## Consent Model

### User Consent Flow

```
1. Client requests scopes: openid profile email users:read
2. GGID shows consent screen:
   ┌─────────────────────────────────────┐
   │ "My App" is requesting access to:    │
   │  ✓ Your profile and email            │
   │  ✓ Read user directory               │
   │  [Deny]              [Allow]         │
   └─────────────────────────────────────┘
3. User clicks Allow → authorization code issued
4. Consent stored, future requests skip (unless scopes change)
```

### Admin Consent

Required for:
- Scopes containing `:admin` or `:delete`
- Scopes requesting data from other users (not just self)
- Dynamic registration with high-risk scopes

```bash
# Admin approves client's scope request
POST /api/v1/oauth/clients/{client_id}/approve-scopes
{
  "scopes": ["users:read", "users:write"],
  "approved_by": "admin@corp.com",
  "expires_at": "2025-12-31"
}
```

### Consent Revocation

```bash
# User revokes consent for a client
DELETE /api/v1/consent/{consent_id}
# → All tokens issued under this consent are revoked
# → Client must re-request authorization
```

## Dynamic Registration Scopes

```bash
POST /api/v1/oauth/register
{
  "client_name": "Analytics Dashboard",
  "scope": "openid profile users:read"
}
```

### Registration Scope Restrictions

| Registration Method | Allowed Scopes |
|-------------------|----------------|
| Open registration | `openid`, `profile`, `email` |
| Authenticated registration | + `users:read`, `roles:read` |
| Admin pre-approved | + `users:write`, `roles:assign` |
| CISO pre-approved | + `*:admin`, `policy:admin` |

## Scope Downgrade (Narrowing)

Clients can request fewer scopes than originally authorized:

```bash
# Originally authorized: openid profile email users:read users:write
# Request token with only users:read
POST /api/v1/oauth/token
{
  "grant_type": "authorization_code",
  "code": "...",
  "scope": "openid users:read"  // Narrowed at token time
}
# → Token has only requested scopes, never broader
```

### Runtime Scope Reduction

```bash
# Client proactively reduces its own scope
POST /api/v1/oauth/token/downgrade
{
  "access_token": "eyJ...",
  "new_scope": "users:read"  // Must be subset
}
# → New token with reduced scope (same expiry)
```

This is used for principle of least privilege — request broad at auth, narrow per-request.

## Scope in JWT

```json
{
  "iss": "https://auth.ggid.dev",
  "sub": "user-uuid",
  "scope": "openid profile users:read",
  "aud": "identity-svc",
  "exp": 1700000900
}
```

## Scope Enforcement

```go
func RequireScope(required string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := getTokenFromContext(r.Context())
            if !token.HasScope(required) {
                http.Error(w, "insufficient scope", 403)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Usage
router.GET("/api/v1/users", RequireScope("users:read")(listUsers))
router.POST("/api/v1/users", RequireScope("users:write")(createUser))
```

## Scope Versioning

When a scope's meaning changes in a breaking way, append version:

```
users:read     → v1 (current)
users:read:v2  → includes metadata field
```

Old clients continue using v1 until migration complete.

## Monitoring

| Metric | Alert |
|--------|-------|
| Scope escalation attempts | Client requesting broader than authorized |
| Unusual scope requests | New scope pattern not seen before |
| High-risk scope usage | `*:admin` token used outside business hours |
| Consent abandonment | User starts consent but doesn't complete |

## See Also

- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [OAuth Scope Design](oauth-scope-design.md)
- [OAuth PAR/JAR/DPoP](oauth-par-jar-dpop.md)
- [Token Exchange Patterns](token-exchange-patterns.md)
- [Conditional Access](conditional-access.md)
