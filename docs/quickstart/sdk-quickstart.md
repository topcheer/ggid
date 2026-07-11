# SDK Quickstart

> Pick your language. Each example verifies a JWT in under 10 lines.

---

## Choose Your SDK

| Language | Package | Install |
|----------|---------|--------|
| [Go](#go) | `github.com/ggid/ggid/sdk/go` | `go get` |
| [Node.js](#nodejs) | `@ggid/node` | `npm install` |
| [Python](#python) | `ggid` | `pip install` |
| [Java](#java) | `dev.ggid:ggid-sdk` | Maven / Gradle |

---

## Go

### Install

```bash
go get github.com/ggid/ggid/sdk/go@latest
```

### Verify JWT (3 lines)

```go
import ggid "github.com/ggid/ggid/sdk/go"

client := ggid.New("http://localhost:8080", ggid.WithJWKS(15*time.Minute))
userInfo, err := client.VerifyToken(ctx, accessToken)
// userInfo.UserID, userInfo.TenantID, userInfo.Roles
```

### Full Example

```go
package main

import (
    "fmt"
    "net/http"
    "time"

    ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
    client := ggid.New("http://localhost:8080", ggid.WithJWKS(15*time.Minute))

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
}
```

**[Full Go SDK Reference →](../../sdk/go/README.md)**

---

## Node.js

### Install

```bash
npm install @ggid/node
```

### Verify JWT (3 lines)

```javascript
const { JWTVerifier } = require('@ggid/node');

const verifier = new JWTVerifier({ jwksUrl: 'http://localhost:8080/.well-known/jwks.json', issuer: 'http://localhost:8080' });
const claims = await verifier.verify(token);
// claims.sub, claims.tenant_id, claims.scope
```

### Full Example

```javascript
const { expressAuth, getClaims, requireRole } = require('@ggid/node');
const express = require('express');

const app = express();
app.use(expressAuth({
  jwksUrl: 'http://localhost:8080/.well-known/jwks.json',
  issuer: 'http://localhost:8080',
}));

app.get('/api/me', (req, res) => {
    const claims = getClaims(req);
    res.json({ userID: claims.sub, email: claims.email });
});

app.listen(3000);
```

**[Full Node.js SDK Reference →](../../sdk/node/README.md)**

---

## Python

### Install

```bash
pip install ggid
```

### Verify JWT (3 lines)

```python
from ggid import GGIDClient

client = GGIDClient(gateway_url="http://localhost:8080", tenant_id="00000000-0000-0000-0000-000000000001")
claims = await client.verify_token(token)
# claims['sub'], claims['tenant_id'], claims['scope']
```

### Full Example (FastAPI)

```python
from fastapi import FastAPI, Depends
from ggid import GGIDMiddleware, get_current_user

app = FastAPI()
app.add_middleware(
    GGIDMiddleware,
    gateway_url="http://localhost:8080",
    jwks_url="http://localhost:8080/.well-known/jwks.json",
    tenant_id="00000000-0000-0000-0000-000000000001",
)

@app.get("/api/me")
async def me(user = Depends(get_current_user)):
    return {"user": user}
```

**[Full Python SDK Reference →](../../sdk/python/README.md)**

---

## Java

### Install (Maven)

```xml
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Verify JWT (3 lines)

```java
import dev.ggid.sdk.*;

GGIDClient client = new GGIDClient(new GGIDClient.Config("http://localhost:8080"));
GGIDUser user = client.getUser("user-id");
// user.getId(), user.getEmail(), user.getRoles()
```

### Full Example (Servlet Filter)

```java
import dev.ggid.sdk.GGIDAuthFilter;

// Register the filter in web.xml or programmatically
GGIDAuthFilter filter = new GGIDAuthFilter();
filter.setGatewayUrl("http://localhost:8080");
// The filter verifies JWT on every request and sets request attributes
```

**[Full Java SDK Reference →](../../sdk/java/README.md)**

---

## Feature Comparison

| Feature | Go | Node.js | Python | Java |
|---------|-----|---------|--------|------|
| JWT Verification | `client.VerifyToken()` | `JWTVerifier` | `GGIDClient` | `GGIDClient` |
| HTTP Middleware | `client.Middleware()` | `expressAuth()` | `GGIDMiddleware` | `GGIDAuthFilter` |
| Gin Integration | `ggidmw.Auth()` | — | — | — |
| Spring Integration | — | — | — | `GGIDSecurityFilter` |
| JWKS Refresh | `WithJWKS(ttl)` | Built-in | Built-in | Built-in |
| Permission Check | `RequirePermission()` | `requirePermission()` | `check_permission()` | `checkPermission()` |

---

*See: [Integration Guides](../integration-guides/) | [API Reference](../api-reference.md)*

*Last updated: 2025-07-11*
