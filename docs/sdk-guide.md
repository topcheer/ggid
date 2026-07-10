# GGID SDK Usage Guide

Side-by-side comparison of the same operations across Go, Node.js, and Java SDKs.
Use this guide to quickly switch languages or compare implementations.

---

## Table of Contents

- [Installation](#installation)
- [Initialization](#initialization)
- [Login](#login)
- [Verify JWT](#verify-jwt)
- [Create User](#create-user)
- [List Users](#list-users)
- [Create Role](#create-role)
- [Check Permission](#check-permission)
- [Create Organization](#create-organization)
- [Error Handling](#error-handling)
- [HTTP Middleware](#http-middleware)
- [Feature Comparison Matrix](#feature-comparison-matrix)

---

## Installation

### Go

```bash
go get github.com/ggid/ggid/sdk/go
```

### Node.js

```bash
npm install @ggid/node jose
# Requires Node.js 18+
```

### Java

```xml
<!-- Maven -->
<dependency>
    <groupId>dev.ggid</groupId>
    <artifactId>ggid-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

```groovy
// Gradle
implementation 'dev.ggid:ggid-sdk:1.0.0'
```

---

## Initialization

### Go

```go
import ggid "github.com/ggid/ggid/sdk/go"

client := ggid.New("https://iam.example.com",
    ggid.WithAPIKey("your-api-key"),
    ggid.WithJWKS(15*time.Minute),
    ggid.WithHTTPClient(&http.Client{Timeout: 30*time.Second}),
)
```

### Node.js

```typescript
import { GGIDClient } from '@ggid/node';

const client = new GGIDClient({
  gatewayUrl: 'https://iam.example.com',
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  tenantId: '00000000-0000-0000-0000-000000000001',
  issuer: 'ggid',
  timeout: 30000,
});
```

### Java

```java
import dev.ggid.sdk.GGIDClient;

GGIDClient.Config config = new GGIDClient.Config("https://iam.example.com");
config.tenantId = "00000000-0000-0000-0000-000000000001";
config.apiKey = "your-api-key";

GGIDClient client = new GGIDClient(config);
```

---

## Login

### Go

```go
tokens, err := client.Login(ctx, &ggid.LoginRequest{
    Username: "admin",
    Password: "Admin@123456",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Access token: %s\n", tokens.AccessToken)
```

### Node.js

```typescript
const tokens = await client.login('admin', 'Admin@123456');
console.log('Access token:', tokens.access_token);
```

### Java

```java
GGIDClient.TokenSet tokens = client.login("admin", "Admin@123456");
System.out.println("Access token: " + tokens.accessToken);
```

---

## Verify JWT

### Go

```go
// With WithJWKS enabled — verifies RS256 signature against JWKS
userInfo, err := client.VerifyToken(ctx, accessToken)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User: %s, Tenant: %s, Roles: %v\n",
    userInfo.Username, userInfo.TenantID, userInfo.Roles)
```

### Node.js

```typescript
const claims = await client.verifyToken(accessToken);
console.log(`User: ${claims.sub}, Roles: ${claims.roles}`);
```

### Java

```java
GGIDClient.UserInfo userInfo = client.verifyToken(accessToken);
System.out.println("User: " + userInfo.username + ", Roles: " + userInfo.roles);
```

---

## Create User

### Go

```go
user, err := client.CreateUser(ctx, &ggid.CreateUserRequest{
    Username:    "john.doe",
    Email:       "john@example.com",
    Password:    "SecurePass@123",
    DisplayName: "John Doe",
    Phone:       "+1234567890",
})
```

### Node.js

```typescript
const user = await client.createUser({
  username: 'john.doe',
  email: 'john@example.com',
  password: 'SecurePass@123',
  display_name: 'John Doe',
  phone: '+1234567890',
});
```

### Java

```java
GGIDClient.User user = client.createUser(
    "john.doe",              // username
    "john@example.com",      // email
    "SecurePass@123"         // password
);
// Additional fields set via update:
client.updateUser(user.id, Map.of("display_name", "John Doe"));
```

---

## List Users

### Go

```go
result, err := client.ListUsers(ctx, &ggid.ListOptions{
    PageSize: 50,
    Search:   "john",
})
for _, u := range result.Users {
    fmt.Printf("%s (%s)\n", u.Username, u.Email)
}
```

### Node.js

```typescript
const { users, total } = await client.listUsers(accessToken, 50);
users.forEach(u => console.log(`${u.username} (${u.email})`));
```

### Java

```java
GGIDClient.UserList result = client.listUsers(1, 50); // page, pageSize
for (GGIDClient.User u : result.users) {
    System.out.println(u.username + " (" + u.email + ")");
}
```

---

## Create Role

### Go

```go
role, err := client.CreateRole(ctx, &ggid.CreateRoleRequest{
    Key:         "editor",
    Name:        "Content Editor",
    Description: "Can edit and publish content",
})
```

### Node.js

```typescript
const role = await client.createRole(accessToken, {
  key: 'editor',
  name: 'Content Editor',
  description: 'Can edit and publish content',
});
```

### Java

```java
GGIDClient.Role role = client.createRole("editor", "Content Editor");
```

---

## Check Permission

### Go

```go
allowed, err := client.CheckPermission(ctx, user.UserID, "documents:sensitive", "read")
if err != nil {
    log.Fatal(err)
}
if allowed {
    fmt.Println("Access granted")
} else {
    fmt.Println("Access denied")
}
```

### Node.js

```typescript
const result = await client.checkPermission(
  accessToken,
  'documents:sensitive',
  'read',
  userId,
);
console.log(result.allowed ? 'Access granted' : 'Access denied');
```

### Java

```java
GGIDClient.PermissionResult result = client.checkPermission(
    userId, "documents:sensitive", "read"
);
System.out.println(result.allowed ? "Access granted" : "Access denied");
```

---

## Create Organization

### Go

```go
org, err := client.CreateOrg(ctx, &ggid.CreateOrgRequest{
    Name:        "Engineering",
    Description: "Engineering Division",
})
```

### Node.js

```typescript
const org = await client.createOrg(accessToken, {
  name: 'Engineering',
  description: 'Engineering Division',
});
```

### Java

```java
GGIDClient.Org org = client.createOrg("Engineering");
```

---

## Error Handling

### Go

```go
_, err := client.GetUser(ctx, "nonexistent-id")
if err != nil {
    var apiErr *ggid.APIError
    if errors.As(err, &apiErr) {
        switch {
        case apiErr.IsNotFound():
            fmt.Println("User not found")
        case apiErr.IsUnauthorized():
            // Token expired — try refresh
        case apiErr.IsRateLimited():
            time.Sleep(5 * time.Second)
        }
    }
}
```

### Node.js

```typescript
try {
  await client.getUser(accessToken, 'nonexistent-id');
} catch (err: any) {
  const status = err.message.match(/GGID API (\d+)/)?.[1];
  if (status === '404') console.log('User not found');
  if (status === '401') // token expired
  if (status === '429') // rate limited
}
```

### Java

```java
try {
    client.getUser("nonexistent-id");
} catch (GGIDException e) {
    if (e.isNotFound()) {
        System.out.println("User not found");
    } else if (e.isRateLimited()) {
        // Wait and retry
    } else if (e.isConflict()) {
        // Resource already exists
    }
    System.out.println(e.getStatusCode() + ": " + e.getMessage());
}
```

---

## HTTP Middleware

### Go (net/http)

```go
mux := http.NewServeMux()
mux.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
    user := ggid.UserFromContext(r.Context())
    json.NewEncoder(w).Encode(user)
})

handler := client.Middleware(mux, ggid.MiddlewareConfig{
    PublicPaths: []string{"/healthz"},
    TenantID:    "00000000-0000-0000-0000-000000000001",
})

http.ListenAndServe(":8080", handler)
```

### Node.js (Express)

```typescript
import { expressAuth, getClaims } from '@ggid/node';

app.use(expressAuth({
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'ggid',
}));

app.get('/api/profile', (req, res) => {
  const user = getClaims(req);
  res.json({ id: user.sub, email: user.email });
});
```

### Java (Servlet Filter)

```java
import dev.ggid.sdk.GGIDAuthFilter;

@WebFilter("/api/*")
public class AuthFilter extends GGIDAuthFilter {
    @Override
    protected void configure() {
        jwksUrl("https://iam.example.com/.well-known/jwks.json");
        issuer("ggid");
        publicPaths("/healthz", "/public");
        tenantId("00000000-0000-0000-0000-000000000001");
    }
}

// In servlet:
GGIDPrincipal principal = (GGIDPrincipal) request.getUserPrincipal();
String userId = principal.getSubject();
```

---

## Feature Comparison Matrix

| Feature | Go SDK | Node.js SDK | Java SDK |
|---------|--------|-------------|----------|
| Login | ✅ | ✅ | ✅ |
| Register | ✅ | ✅ | ✅ |
| Token refresh | ✅ | ✅ | ✅ |
| JWT verification (JWKS) | ✅ | ✅ (jose) | ✅ (java-jwt) |
| JWT verification (offline) | ✅ | ❌ | ✅ |
| Create / Get / Delete user | ✅ | ✅ | ✅ |
| List users | ✅ | ✅ | ✅ |
| Create / List roles | ✅ | ✅ | ✅ |
| Assign role | ✅ | ❌ | ✅ |
| Check permission | ✅ | ✅ | ✅ |
| Create / List orgs | ✅ | ✅ | ✅ |
| HTTP middleware | ✅ (net/http) | ✅ (Express) | ✅ (Servlet Filter) |
| Permission guard middleware | ✅ (`RequirePermission`) | ✅ (`requirePermission`) | ❌ |
| Role guard middleware | ✅ (`RequireRole`) | ❌ | ❌ |
| Scope guard middleware | ✅ (`RequireScope`) | ❌ | ❌ |
| Structured errors | ✅ (`APIError`) | ✅ (regex match) | ✅ (`GGIDException`) |
| Token expiry auto-refresh | ✅ (manual) | ✅ (manual) | ❌ |
| Rate limit retry | ✅ (manual) | ✅ (manual) | ❌ |
| Custom HTTP client | ✅ (`WithHTTPClient`) | ❌ | ❌ |
| TypeScript types | N/A | ✅ | N/A |

---

## Language-Specific Notes

### Go

- Synchronous API (context-based)
- Best for high-performance backend services
- JWKS caching with configurable TTL
- Full middleware suite (JWT, role, permission, scope guards)

### Node.js

- Async/await API (returns Promises)
- Best for SPAs, serverless, and Node.js backends
- Uses `jose` library for JWT verification (browser-compatible)
- Works with Express, Fastify, Hono, Next.js

### Java

- Synchronous API (OkHttp under the hood)
- Best for enterprise Java / Spring Boot applications
- Uses `java-jwt` (Auth0) for JWT verification
- Integrates as Servlet Filter or Spring Security filter

---

## Python SDK (Bonus)

GGID also includes a Python SDK for Flask/FastAPI applications:

```bash
pip install ggid-sdk
```

```python
from ggid import GGIDClient

client = GGIDClient(
    gateway_url="https://iam.example.com",
    tenant_id="00000000-0000-0000-0000-000000000001",
)

# Login
tokens = client.login("admin", "Admin@123456")

# Verify JWT
claims = client.verify_token(tokens["access_token"])

# Check permission
result = client.check_permission(user_id, "documents:sensitive", "read")
print("Allowed:", result["allowed"])
```

### Flask Middleware

```python
from ggid.middleware import GGIDAuthMiddleware

app.wsgi_app = GGIDAuthMiddleware(
    app.wsgi_app,
    jwks_url="https://iam.example.com/.well-known/jwks.json",
    public_paths=["/healthz", "/public"],
)
```

---

## Token Refresh Automation

### Go — Auto-Refreshing Token Manager

```go
package main

import (
    "context"
    "sync"
    "time"

    "github.com/ggid/ggid/sdk/go"
)

type TokenManager struct {
    client       *ggid.Client
    refreshTok   string
    accessTok    string
    expiresAt    time.Time
    mu           sync.RWMutex
}

func NewTokenManager(client *ggid.Client, refreshToken string) *TokenManager {
    return &TokenManager{
        client:     client,
        refreshTok: refreshToken,
    }
}

// GetToken returns a valid access token, refreshing if needed.
func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
    tm.mu.RLock()
    if time.Now().Before(tm.expiresAt.Add(-30 * time.Second)) {
        tok := tm.accessTok
        tm.mu.RUnlock()
        return tok, nil
    }
    tm.mu.RUnlock()

    // Refresh
    tm.mu.Lock()
    defer tm.mu.Unlock()

    // Double-check after acquiring write lock
    if time.Now().Before(tm.expiresAt.Add(-30 * time.Second)) {
        return tm.accessTok, nil
    }

    resp, err := tm.client.Auth.Refresh(ctx, tm.refreshTok)
    if err != nil {
        return "", fmt.Errorf("token refresh failed: %w", err)
    }

    tm.accessTok = resp.AccessToken
    tm.refreshTok = resp.RefreshToken
    tm.expiresAt = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
    return tm.accessTok, nil
}

// Usage
func main() {
    client := ggid.New("https://iam.example.com", "tenant-uuid")
    tm := NewTokenManager(client, "initial-refresh-token")

    // The token manager auto-refreshes before expiry
    tok, _ := tm.GetToken(context.Background())
    fmt.Println("Using token:", tok[:20], "...")
}
```

### Node.js — Auto-Refresh Interceptor

```typescript
import { GGIDClient } from '@ggid/node-sdk';

class AutoRefreshClient {
  private client: GGIDClient;
  private accessToken: string | null = null;
  private refreshToken: string;
  private expiresAt: number = 0;

  constructor(baseUrl: string, tenantId: string, refreshToken: string) {
    this.client = new GGIDClient({ baseUrl, tenantId });
    this.refreshToken = refreshToken;
  }

  async getToken(): Promise<string> {
    // Refresh if token expires in < 30 seconds
    if (this.accessToken && Date.now() < this.expiresAt - 30_000) {
      return this.accessToken;
    }

    const resp = await this.client.auth.refresh(this.refreshToken);
    this.accessToken = resp.access_token;
    this.refreshToken = resp.refresh_token;
    this.expiresAt = Date.now() + resp.expires_in * 1000;
    return this.accessToken;
  }

  // Drop-in replacement for authenticated API calls
  async getUsers() {
    const token = await this.getToken();
    return this.client.users.list({ headers: { Authorization: `Bearer ${token}` } });
  }
}
```

---

## Webhook Handling

GGID sends HMAC-signed webhooks for events like `user.created`, `user.deleted`,
`role.assigned`. This section shows how to verify and process webhooks.

### Webhook Payload Structure

```json
{
  "event_id": "evt-abc123",
  "event_type": "user.created",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "user_id": "...",
    "username": "newuser",
    "email": "user@test.com"
  }
}
```

### Headers

```
X-GGID-Signature: sha256=<hex-hmac>
X-GGID-Timestamp: 1699999999
X-GGID-Event-Type: user.created
```

### Go — Webhook Verification

```go
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

const webhookSecret = "your-webhook-secret"

type WebhookEvent struct {
    EventID   string          `json:"event_id"`
    EventType string          `json:"event_type"`
    TenantID  string          `json:"tenant_id"`
    Timestamp time.Time      `json:"timestamp"`
    Data      json.RawMessage `json:"data"`
}

func verifySignature(body []byte, signature string, timestamp string, secret string) bool {
    // Check timestamp freshness (5-minute window)
    ts, err := strconv.ParseInt(timestamp, 10, 64)
    if err != nil || time.Now().Unix()-ts > 300 {
        return false
    }

    // Compute HMAC-SHA256
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(timestamp + "." + string(body)))
    expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

    return hmac.Equal([]byte(signature), []byte(expectedSig))
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    signature := r.Header.Get("X-GGID-Signature")
    timestamp := r.Header.Get("X-GGID-Timestamp")

    if !verifySignature(body, signature, timestamp, webhookSecret) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }

    var event WebhookEvent
    if err := json.Unmarshal(body, &event); err != nil {
        http.Error(w, "bad json", http.StatusBadRequest)
        return
    }

    // Process event
    switch event.EventType {
    case "user.created":
        var data struct {
            UserID string `json:"user_id"`
            Email  string `json:"email"`
        }
        json.Unmarshal(event.Data, &data)
        fmt.Printf("New user: %s (%s)\n", data.UserID, data.Email)

    case "user.deleted":
        fmt.Println("User deleted")

    case "role.assigned":
        fmt.Println("Role assigned")

    default:
        fmt.Printf("Unknown event: %s\n", event.EventType)
    }

    w.WriteHeader(http.StatusOK)
}
```

### Node.js — Webhook Verification (Express)

```typescript
import express from 'express';
import crypto from 'crypto';

const app = express();
const WEBHOOK_SECRET = process.env.GGID_WEBHOOK_SECRET!;

// Raw body for signature verification
app.use('/webhook', express.raw({ type: 'application/json' }));

app.post('/webhook', (req, res) => {
  const signature = req.headers['x-ggid-signature'] as string;
  const timestamp = req.headers['x-ggid-timestamp'] as string;

  // Verify timestamp (5-minute window)
  const ts = parseInt(timestamp, 10);
  if (Math.floor(Date.now() / 1000) - ts > 300) {
    return res.status(401).json({ error: 'stale webhook' });
  }

  // Verify HMAC
  const body = `${timestamp}.${req.body.toString()}`;
  const expected = 'sha256=' + crypto
    .createHmac('sha256', WEBHOOK_SECRET)
    .update(body)
    .digest('hex');

  if (!crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  )) {
    return res.status(401).json({ error: 'invalid signature' });
  }

  const event = JSON.parse(req.body.toString());

  switch (event.event_type) {
    case 'user.created':
      console.log('User created:', event.data.user_id);
      break;
    case 'user.deleted':
      console.log('User deleted:', event.data.user_id);
      break;
    case 'role.assigned':
      console.log('Role assigned:', event.data);
      break;
  }

  res.status(200).json({ received: true });
});

app.listen(3000, () => console.log('Webhook server on :3000'));
```

### Webhook Retry Strategy

GGID retries failed webhooks with exponential backoff:

| Attempt | Delay | Total Elapsed |
|---------|-------|---------------|
| 1 | 0s | 0s |
| 2 | 30s | 30s |
| 3 | 2m | 2m30s |
| 4 | 10m | 12m30s |
| 5 | 1h | 1h12m30s |
| 6 | 6h | 7h12m30s |
| 7 | 24h | 31h12m30s |

After 7 failed attempts, the webhook is placed in a dead-letter queue for
manual inspection.

**Best practices:**
1. Return `200 OK` immediately, process asynchronously
2. Use idempotency (deduplicate by `event_id`)
3. Handle events in order (use `timestamp` for ordering)
4. Log all webhook receipts for audit trail

---

## SDK Feature Comparison (Updated)

| Feature | Go | Node.js | Java | Python |
|---------|-----|---------|------|--------|
| Client init | Full | Full | Full | Full |
| Login/Register | Full | Full | Full | Full |
| JWT verify | Full | Full | Full | Full |
| User CRUD | Full | Full | Full | Full |
| Role management | Full | Full | Full | Full |
| Policy check | Full | Full | Full | Full |
| Org management | Full | Full | Full | Full |
| HTTP middleware | Full | Express | Servlet | Flask |
| Token refresh (auto) | Full | Full | Manual | Manual |
| Webhook verification | Example | Example | — | — |
| gRPC client | Full | — | — | — |
| Error types | Typed | Typed | Exceptions | Exceptions |

---

## References

- [Go SDK Source](../sdk/go/) — `github.com/ggid/ggid/sdk/go`
- [Node.js SDK Source](../sdk/node/) — `@ggid/node-sdk`
- [Java SDK Source](../sdk/java/) — `io.ggid:sdk`
- [API Reference](./api-reference.md) — REST endpoint documentation
- [Webhooks Guide](./webhooks-guide.md) — Webhook event types and configuration
- [Integration Guide](./integration-guide.md) — App integration patterns
