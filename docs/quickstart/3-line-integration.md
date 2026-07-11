# 3-Line Integration

> Add GGID JWT authentication to your app in 3 lines of code. Pick your language.

---

## Go

```bash
go get github.com/ggid/ggid/sdk/go@latest
```

```go
import ggid "github.com/ggid/ggid/sdk/go"

client := ggid.New("http://localhost:8080", ggid.WithJWKS(15*time.Minute))
userInfo, err := client.VerifyToken(ctx, accessToken)
// userInfo.UserID, userInfo.TenantID, userInfo.Roles, userInfo.Scopes
```

### Protect an HTTP handler

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/me", client.RequirePermission("users", "read",
    func(w http.ResponseWriter, r *http.Request) {
        user := ggid.UserFromContext(r.Context())
        fmt.Fprintf(w, "Hello, %s", user.Username)
    },
))
handler := client.Middleware(mux, ggid.MiddlewareConfig{
    PublicPaths: []string{"/healthz"},
})
http.ListenAndServe(":8081", handler)
```

---

## Node.js

```bash
npm install @ggid/node
```

```javascript
const { expressAuth, getClaims } = require('@ggid/node');

app.use('/api', expressAuth({
  jwksUrl: 'http://localhost:8080/.well-known/jwks.json',
  issuer: 'http://localhost:8080',
}));

app.get('/api/me', (req, res) => {
  const claims = getClaims(req);
  res.json({ userID: claims.sub, tenantID: claims.tenant_id });
});
```

---

## Python

```bash
pip install ggid
```

```python
from ggid import GGIDClient

client = GGIDClient(
    gateway_url="http://localhost:8080",
    tenant_id="00000000-0000-0000-0000-000000000001",
)
claims = await client.verify_token(token)
# claims['sub'], claims['tenant_id'], claims['scope']
```

### Protect a FastAPI route

```python
from fastapi import FastAPI, Depends
from ggid import GGIDMiddleware, get_current_user

app = FastAPI()
app.add_middleware(
    GGIDMiddleware,
    gateway_url="http://localhost:8080",
    jwks_url="http://localhost:8080/.well-known/jwks.json",
)

@app.get("/api/me")
async def me(user = Depends(get_current_user)):
    return {"user": user}
```

---

## Java

### Maven

```xml
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

```java
import dev.ggid.sdk.*;

GGIDClient client = new GGIDClient(new GGIDClient.Config("http://localhost:8080"));
GGIDUser user = client.getUser("user-id");
```

### Spring Security Filter

```java
@Bean
public FilterRegistrationBean<GGIDSecurityFilter> ggidFilter() {
    FilterRegistrationBean<GGIDSecurityFilter> bean = new FilterRegistrationBean<>();
    bean.setFilter(new GGIDSecurityFilter(client, "http://localhost:8080/.well-known/jwks.json"));
    bean.addUrlPatterns("/api/*");
    return bean;
}
```

---

## What You Get

| Feature | Go | Node.js | Python | Java |
|---------|-----|---------|--------|------|
| JWT verification | `VerifyToken()` | `expressAuth()` | `verify_token()` | `JwtVerifier.verify()` |
| HTTP middleware | `Middleware()` | Express middleware | `GGIDMiddleware` | `GGIDAuthFilter` |
| Permission check | `RequirePermission()` | `requirePermission()` | `check_permission()` | `checkPermission()` |
| User from context | `UserFromContext()` | `getClaims()` | `get_current_user()` | `req.getAttribute()` |
| JWKS caching | `WithJWKS(ttl)` | Built-in | Built-in | Built-in |

---

*See: [SDK Quickstart](sdk-quickstart.md) | [Go SDK](go-sdk.md) | [Node SDK](node-sdk.md) | [Integration Guides](../integration-guides/)*

*Last updated: 2025-07-11*
