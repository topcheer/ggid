# GGID SDK Design Guide — Simplest, Most Flexible Integration

## Design Philosophy

> **One line to init. One line to authenticate. One line to authorize.**

Every GGID SDK must be usable in under 5 lines of code for the common case:

```ruby
# Ruby example — same pattern in every language
ggid = GGID::Client.new(base_url: "https://ggid.iot2.win", tenant_id: "00000000-...")
claims = ggid.verify_token(jwt_string)          # raises on invalid
allowed = ggid.check_permission(token, "products", "read")  # => true/false
```

## Core API Surface (ALL languages must implement)

### 1. Client Initialization
```
GGIDClient(config {
  base_url:    string   # GGID gateway URL
  tenant_id:   string   # tenant UUID
  timeout?:    int      # request timeout (default 10s)
  retries?:    int      # retry count (default 3)
})
```

### 2. Authentication
```
verify_token(token) → Claims
  # Claims: { user_id, tenant_id, roles[], scope, exp, iat, iss }
  # Verifies signature via JWKS, checks expiry
  # Raises InvalidTokenError on failure

get_user_info(token) → UserInfo
  # Calls GET /api/v1/oauth/userinfo
  # UserInfo: { sub, name, email, roles[], picture? }
```

### 3. OAuth/OIDC
```
get_authorize_url(client_id, redirect_uri, scope?, state?) → string
  # Returns full authorize URL for redirect

exchange_code(code, redirect_uri, client_id, client_secret) → TokenResponse
  # TokenResponse: { access_token, refresh_token, id_token?, expires_in, token_type }

refresh_token(refresh_token, client_id, client_secret) → TokenResponse

get_jwks() → { keys: [...] }
get_discovery() → { issuer, authorization_endpoint, token_endpoint, jwks_uri, userinfo_endpoint }
revoke_token(token, client_id, client_secret) → void
```

### 4. RBAC (Role-Based Access Control)
```
check_permission(token, resource, action) → bool
  # Calls POST /api/v1/policies/check with user_id from token

assign_role(token, user_id, role_id) → void
revoke_role(token, user_id, role_id) → void
get_user_roles(token, user_id) → Role[]
list_roles(token) → Role[]
list_permissions(token) → Permission[]
```

### 5. ABAC (Attribute-Based Access Control)
```
evaluate_abac(token, request {
  action: string,
  resource: string,
  conditions: [{ field, operator, value }]
}) → { matched: bool, matched_rules: [...] }

check_policy(token, request {
  action, resource, subject: { user_id, roles, attributes },
  conditions: [...], tenant_id
}) → { allowed: bool, matched_rules: [...], reason: string }
```

### 6. Framework Integration (idiomatic per language)

| Language | Auth Guard | Permission Guard | Role Guard |
|----------|-----------|-----------------|-----------|
| Go | `AuthMiddleware()` | `RequirePermission("res","act")` | `RequireRole("admin")` |
| Node.js | `authMiddleware()` | `requirePermission("res","act")` | `requireRole("admin")` |
| Java | `@RequiresAuth` | `@RequirePermission(resource="res", action="act")` | `@RequireRole("admin")` |
| Python | `@require_auth` | `@require_permission("res","act")` | `@require_role("admin")` |
| Ruby | `before_action :require_auth` | `before_action -> { require_permission!("res","act") }` | `before_action -> { require_role!("admin") }` |
| PHP | `#[RequiresAuth]` attribute / `$app->use(AuthMiddleware::class)` | `#[RequirePermission("res","act")]` | `#[RequireRole("admin")]` |
| C# | `[Authorize]` attribute | `[RequirePermission("res","act")]` | `[RequireRole("admin")]` |
| Rust | `#[ggid::require_auth]` macro / `AuthLayer` | `#[ggid::require_permission("res","act")]` | `#[ggid::require_role("admin")]` |
| Dart | `GGIDAuthMiddleware()` | `requirePermission("res","act")` | `requireRole("admin")` |

## File Structure (per language SDK)

```
sdk/{lang}/
├── README.md              # Quick start, 5-min integration guide
├── {package config}       # Cargo.toml / composer.json / Gemfile / .csproj / pubspec.yaml
├── src/
│   ├── client.{ext}       # Core GGIDClient class
│   ├── auth.{ext}         # JWT verification, OAuth flows
│   ├── rbac.{ext}         # Permission checking, role management
│   ├── abac.{ext}         # Policy evaluation
│   ├── middleware.{ext}   # Framework integration (guards/middleware/decorators)
│   └── types.{ext}        # Shared types/models
├── tests/
│   └── test_*.{ext}       # Unit tests covering all API methods
└── examples/
    └── quickstart.{ext}   # Working example app
```

## Language Support Matrix

| Feature | Go | Node | Java | Python | Ruby | PHP | C# | Rust | Dart |
|---------|-----|------|------|--------|------|-----|-----|------|------|
| JWT Verify | ✅ | ✅ | ✅ | ✅ | NEW | NEW | NEW | NEW | NEW |
| OAuth/OIDC | ✅ | ✅ | ✅ | ✅ | NEW | NEW | NEW | NEW | NEW |
| RBAC | ✅ | ✅ | ✅ | ✅ | NEW | NEW | NEW | NEW | NEW |
| ABAC | ✅ | ✅ | ✅ | ✅ | NEW | NEW | NEW | NEW | NEW |
| Middleware | ✅ | ✅ | ✅ | ✅ | NEW | NEW | NEW | NEW | NEW |
| Tests | ✅ | ✅ | ✅ | ✅ | NEW | NEW | NEW | NEW | NEW |
| Docs | ✅ | ✅ | ✅ | ✅ | NEW | NEW | NEW | NEW | NEW |
