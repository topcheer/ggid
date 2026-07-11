# GGID SDK Reference

Complete method reference for Go, Node.js, Java, and Python SDKs.

---

## Go SDK (`sdk/go/`)

### Client

```go
client := ggid.New("https://iam.example.com",
    ggid.WithAPIKey("key"),
    ggid.WithJWKS(15*time.Minute),
    ggid.WithHTTPClient(&http.Client{Timeout: 30*time.Second}),
)
```

| Method | Signature | Returns |
|--------|-----------|---------|
| `Login` | `(ctx, *LoginRequest) → (*TokenSet, error)` | `{AccessToken, RefreshToken}` |
| `Register` | `(ctx, *RegisterRequest) → (*User, error)` | User object |
| `RefreshToken` | `(ctx, refreshToken) → (*TokenSet, error)` | New token pair |
| `VerifyToken` | `(ctx, accessToken) → (*UserInfo, error)` | `{Subject, TenantID, Roles, Scopes}` |
| `Logout` | `(ctx, accessToken) → error` | nil |
| `CreateUser` | `(ctx, *CreateUserRequest) → (*User, error)` | User |
| `GetUser` | `(ctx, userID) → (*User, error)` | User |
| `ListUsers` | `(ctx, *ListOptions) → (*UserList, error)` | `{Users, Total, Page}` |
| `UpdateUser` | `(ctx, userID, *UpdateUserRequest) → (*User, error)` | Updated user |
| `DeleteUser` | `(ctx, userID) → error` | nil |
| `CreateRole` | `(ctx, *CreateRoleRequest) → (*Role, error)` | Role |
| `ListRoles` | `(ctx) → ([]*Role, error)` | Role list |
| `AssignRole` | `(ctx, userID, roleID) → error` | nil |
| `CheckPermission` | `(ctx, userID, resource, action) → (bool, error)` | allowed bool |
| `CreateOrg` | `(ctx, *CreateOrgRequest) → (*Org, error)` | Organization |
| `ListOrgs` | `(ctx) → ([]*Org, error)` | Org list |

### Middleware

```go
handler := client.Middleware(mux, ggid.MiddlewareConfig{
    PublicPaths: []string{"/healthz"},
    TenantID:    "00000000-0000-0000-0000-000000000001",
})
```

| Guard Method | Signature |
|-------------|-----------|
| `RequireRole("admin")` | `func(http.Handler) http.Handler` |
| `RequirePermission("docs:read")` | `func(http.Handler) http.Handler` |
| `RequireScope("write")` | `func(http.Handler) http.Handler` |

### Error Handling

```go
var apiErr *ggid.APIError
if errors.As(err, &apiErr) {
    apiErr.IsNotFound()       // 404
    apiErr.IsUnauthorized()   // 401
    apiErr.IsRateLimited()    // 429
    apiErr.IsRetryable()      // 429, 5xx
    apiErr.StatusCode         // int
    apiErr.Code               // string
}
```

---

## Node.js SDK (`sdk/node/`)

### Client

```typescript
const client = new GGIDClient({
  gatewayUrl: 'https://iam.example.com',
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'ggid',
  tenantId: '00000000-0000-0000-0000-000000000001',
});
```

| Method | Parameters | Returns |
|--------|-----------|---------|
| `login(username, password)` | string, string | `Promise<TokenSet>` |
| `register(data)` | `{username, email, password}` | `Promise<User>` |
| `verifyToken(token)` | string | `Promise<Claims>` |
| `createUser(token, data)` | string, object | `Promise<User>` |
| `getUser(token, id)` | string, string | `Promise<User>` |
| `listUsers(token, pageSize?)` | string, number | `Promise<{users, total}>` |
| `updateUser(token, id, data)` | string, string, object | `Promise<User>` |
| `deleteUser(token, id)` | string, string | `Promise<void>` |
| `createRole(token, data)` | string, `{key, name}` | `Promise<Role>` |
| `listRoles(token)` | string | `Promise<Role[]>` |
| `checkPermission(token, resource, action, userId?)` | string, string, string, string? | `Promise<{allowed}>` |
| `createOrg(token, data)` | string, `{name}` | `Promise<Org>` |
| `listOrgs(token)` | string | `Promise<Org[]>` |

### Middleware

```typescript
import { expressAuth, requirePermission, getClaims } from '@ggid/node';

app.use(expressAuth({ jwksUrl, issuer }));
app.get('/api/profile', requirePermission('users:read'), (req, res) => {
  const user = getClaims(req);
  res.json(user);
});
```

### Error Handling

```typescript
try {
  await client.getUser(token, id);
} catch (err: any) {
  const status = err.message.match(/GGID API (\d+)/)?.[1];
  // status: '404', '401', '429', '500', etc.
}
```

---

## Java SDK (`sdk/java/`)

### Client

```java
GGIDClient.Config config = new GGIDClient.Config("https://iam.example.com");
config.tenantId = "00000000-0000-0000-0000-000000000001";
GGIDClient client = new GGIDClient(config);
```

| Method | Parameters | Returns |
|--------|-----------|---------|
| `login(username, password)` | String, String | `TokenSet` |
| `register(username, email, password)` | String, String, String | `User` |
| `verifyToken(token)` | String | `UserInfo` |
| `createUser(username, email, password)` | String, String, String | `User` |
| `getUser(id)` | String | `User` |
| `listUsers(page, pageSize)` | int, int | `UserList` |
| `updateUser(id, fields)` | String, Map | `User` |
| `deleteUser(id)` | String | void |
| `createRole(key, name)` | String, String | `Role` |
| `listRoles()` | — | `List<Role>` |
| `assignRole(userId, roleId)` | String, String | void |
| `checkPermission(userId, resource, action)` | String, String, String | `PermissionResult` |
| `createOrg(name)` | String | `Org` |
| `listOrgs()` | — | `List<Org>` |

### Servlet Filter

```java
@WebFilter("/api/*")
public class AuthFilter extends GGIDAuthFilter {
    @Override
    protected void configure() {
        jwksUrl("https://iam.example.com/.well-known/jwks.json");
        issuer("ggid");
        publicPaths("/healthz");
    }
}
```

### Error Handling

```java
try {
    client.getUser(id);
} catch (GGIDException e) {
    e.getStatusCode();    // 404
    e.getMessage();       // "User not found"
    e.isNotFound();       // true
    e.isRetryable();      // false
}
```

---

## Python SDK (`sdk/python/`)

### Client

```python
from ggid import GGIDClient

client = GGIDClient(
    gateway_url="https://iam.example.com",
    tenant_id="00000000-0000-0000-0000-000000000001",
)
```

| Method | Parameters | Returns |
|--------|-----------|---------|
| `login(username, password)` | str, str | `dict (tokens)` |
| `register(username, email, password)` | str, str, str | `dict (user)` |
| `verify_token(token)` | str | `dict (claims)` |
| `create_user(token, **fields)` | str, kwargs | `dict (user)` |
| `get_user(token, id)` | str, str | `dict (user)` |
| `list_users(token, page_size)` | str, int | `dict (users)` |
| `create_role(token, key, name)` | str, str, str | `dict (role)` |
| `check_permission(token, resource, action, user_id)` | str, str, str, str | `dict (result)` |
| `create_org(token, name)` | str, str | `dict (org)` |

### Flask Middleware

```python
from ggid.middleware import GGIDAuthMiddleware

app.wsgi_app = GGIDAuthMiddleware(
    app.wsgi_app,
    jwks_url="https://iam.example.com/.well-known/jwks.json",
    public_paths=["/healthz"],
)
```

---

## Feature Comparison

| Feature | Go | Node.js | Java | Python |
|---------|:--:|:-------:|:----:|:------:|
| Login/Register | Yes | Yes | Yes | Yes |
| Token Refresh | Yes | Yes | Yes | Yes |
| JWT Verification (JWKS) | Yes | Yes | Yes | Yes |
| User CRUD | Yes | Yes | Yes | Yes |
| Role CRUD | Yes | Yes | Yes | No |
| Assign Role | Yes | No | Yes | No |
| Check Permission | Yes | Yes | Yes | Yes |
| Org CRUD | Yes | Yes | Yes | No |
| HTTP Middleware | Yes (net/http) | Yes (Express) | Yes (Servlet) | Yes (Flask) |
| Role Guard | Yes | No | No | No |
| Permission Guard | Yes | Yes | No | No |
| Scope Guard | Yes | No | No | No |
| Structured Errors | Yes (APIError) | Yes (regex) | Yes (GGIDException) | No |

---

## Error Handling

### Go

```go
user, err := client.Users.Get(ctx, "user-id")
if err != nil {
    var apiErr *ggid.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Code {
        case "auth.token_expired":
            // Refresh and retry
        case "identity.user_not_found":
            log.Printf("Not found: %s", apiErr.Message)
        case "gateway.rate_limited":
            time.Sleep(time.Duration(apiErr.RetryAfter) * time.Second)
        }
    }
}
```

### Node.js

```typescript
try {
  await client.users.get('user-id');
} catch (err) {
  if (err instanceof ggid.APIError) {
    if (err.code === 'gateway.rate_limited') {
      await sleep(err.retryAfter * 1000);
    }
  }
}
```

### APIError Structure

```json
{
  "code": "identity.user_not_found",
  "message": "User not found",
  "status": 404,
  "retry_after": null
}
```

---

## Pagination

### Cursor-Based (SDK)

```go
// Go
users, cursor, _ := client.Users.List(ctx, &ggid.ListOptions{Limit: 50})
for cursor != "" {
    users, cursor, _ = client.Users.List(ctx, &ggid.ListOptions{
        Limit: 50, Cursor: cursor,
    })
}
```

```typescript
// Node.js — async iterator (auto-paginates)
for await (const user of client.users.list({ limit: 50 })) {
  console.log(user.username);
}
```

---

## Retry Configuration

```go
client := ggid.NewClient("https://iam.example.com", ggid.WithRetry(
    ggid.RetryConfig{
        MaxAttempts:   3,
        InitialDelay:  1 * time.Second,
        MaxDelay:      30 * time.Second,
        Multiplier:    2,
        RetryOnStatus: []int{429, 500, 502, 503, 504},
    },
))
```

| Attempt | Delay |
|---------|-------|
| 1 | Immediate |
| 2 | 1s |
| 3 | 2s |
| 4 (max) | 4s |

> Retries respect `Retry-After` header on 429 responses.
