# SDK Quickstart

> Pick your language. Each example verifies a JWT in under 10 lines.

---

## Choose Your SDK

| Language | Package | Install |
|----------|---------|--------|
| [Go](#go) | `github.com/ggid/ggid/sdk/go` | `go get` |
| [Node.js](#nodejs) | `@ggid/sdk-node` | `npm install` |
| [Python](#python) | `ggid` | `pip install` |
| [Java](#java) | `dev.ggid:ggid-sdk-java` | Maven / Gradle |

---

## Go

### Install

```bash
go get github.com/ggid/ggid/sdk/go@latest
```

### Verify JWT (3 lines)

```go
import ggid "github.com/ggid/ggid/sdk/go"

verifier := ggid.NewVerifier("http://localhost:8080", "jwt-secret")
claims, err := verifier.Verify(ctx, tokenString)
// claims.UserID, claims.TenantID, claims.Scope
```
### Full Example

```go
package main

import (
    "fmt"
    "net/http"
    ggid "github.com/ggid/ggid/sdk/go"
)

func main() {
    verifier := ggid.NewVerifier("http://localhost:8080", "secret")
    mw := ggid.NewMiddleware(verifier)
    http.Handle("/api/me", mw.Protect(http.HandlerFunc(meHandler)))
    http.ListenAndServe(":8081", nil)
}
```

**[Full Go SDK Reference →](../../sdk/go/README.md)**

---

## Node.js

### Install

```bash
npm install @ggid/sdk-node
```

### Verify JWT (3 lines)

```javascript
const { GGIDVerifier } = require('@ggid/sdk-node');

const verifier = new GGIDVerifier({ gatewayURL: 'http://localhost:8080', secret: 'jwt-secret' });
const claims = await verifier.verify(token);
// claims.sub, claims.tenant_id, claims.scope
```

### Full Example

```javascript
const { GGIDMiddleware } = require('@ggid/sdk-node');
const express = require('express');

const app = express();
app.use(GGIDMiddleware({ gatewayURL: 'http://localhost:8080', secret: 'jwt-secret' }));

app.get('/api/me', (req, res) => {
    res.json({ userID: req.ggid.userID });
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
from ggid import GGIDVerifier

verifier = GGIDVerifier(gateway_url="http://localhost:8080", secret="jwt-secret")
claims = verifier.verify(token)
# claims.user_id, claims.tenant_id, claims.scope
```

### Full Example

```python
from ggid import GGIDMiddleware
from flask import Flask, request, jsonify

app = Flask(__name__)
app.wsgi_app = GGIDMiddleware(app.wsgi_app,
    gateway_url="http://localhost:8080", secret="jwt-secret")

@app.route('/api/me')
def me():
    return jsonify(user_id=request.ggid.user_id)
```

**[Full Python SDK Reference →](../../sdk/python/README.md)**

---

## Java

### Install (Maven)

```xml
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Verify JWT (3 lines)

```java
import dev.ggid.sdk.GGIDVerifier;

GGIDVerifier verifier = new GGIDVerifier("http://localhost:8080", "jwt-secret");
GGIDClaims claims = verifier.verify(token);
// claims.getUserId(), claims.getTenantId(), claims.getScopes()
```

### Full Example (Spring Boot)

```java
// See: docs/integration-guides/spring-boot.md
@Configuration
@EnableWebSecurity
public class SecurityConfig {
    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http.addFilterBefore(new GGIDJwtFilter(verifier),
            UsernamePasswordAuthenticationFilter.class);
        return http.build();
    }
}
```

**[Full Java SDK Reference →](../../sdk/java/README.md)**

---

## Feature Comparison

| Feature | Go | Node.js | Python | Java |
|---------|-----|---------|--------|------|
| JWT Verification | ✓ | ✓ | ✓ | ✓ |
| HTTP Middleware | ✓ | ✓ (Express) | ✓ (Flask/Django) | ✓ (Servlet Filter) |
| Gin Integration | ✓ | — | — | — |
| Spring Integration | — | — | — | ✓ |
| JWKS Refresh | ✓ | ✓ | ✓ | ✓ |
| Permission Check | ✓ | ✓ | ✓ | ✓ |

---

*See: [Integration Guides](../integration-guides/) | [API Reference](../api-reference.md)*

*Last updated: 2025-07-11*