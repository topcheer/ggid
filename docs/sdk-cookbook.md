# SDK Cookbook

> Practical integration recipes for the GGID Go, Node.js, and Java SDKs.
> Each recipe is a self-contained scenario with copy-paste-ready code.

---

## Table of Contents

1. [JWT Middleware Setup](#1-jwt-middleware-setup)
2. [Permission Check API](#2-permission-check-api)
3. [Refresh Token Rotation](#3-refresh-token-rotation)
4. [Social Login Integration](#4-social-login-integration)
5. [SCIM User Synchronization](#5-scim-user-synchronization)
6. [Multi-Tenant Request Scoping](#6-multi-tenant-request-scoping)
7. [Audit Event Query](#7-audit-event-query)
8. [Webhook Signature Verification](#8-webhook-signature-verification)

---

## 1. JWT Middleware Setup

Protect your API endpoints by verifying GGID-issued JWTs.

### Go (net/http)

```go
package main

import (
    "net/http"

    "github.com/ggid/ggid/sdk/go"
)

func main() {
    // Configure middleware
    mw := ggid.NewJWTMiddleware(ggid.Config{
        JWKSURL:   "https://iam.example.com/.well-known/jwks.json",
        TenantID:  "00000000-0000-0000-0000-000000000001",
    })

    mux := http.NewServeMux()

    // Protected route
    mux.Handle("/api/profile", mw.Protect(http.HandlerFunc(profileHandler)))

    // Public routes (no JWT required)
    mux.HandleFunc("/health", healthHandler)

    http.ListenAndServe(":8080", mux)
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
    // Get user info from JWT claims
    claims := ggid.ClaimsFromContext(r.Context())
    userID := claims["sub"].(string)
    email := claims["email"].(string)

    w.Write([]byte(`{"user_id":"` + userID + `","email":"` + email + `"}`))
}
```

### Node.js (Express)

```typescript
import express from 'express';
import { GGIDMiddleware } from '@ggid/node-sdk';

const app = express();

// Configure middleware
const ggid = new GGIDMiddleware({
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  tenantId: '00000000-0000-0000-0000-000000000001',
});

// Apply to all /api routes
app.use('/api', ggid.verify());

// Or apply to specific routes
app.get('/api/profile', ggid.verify(), (req, res) => {
  const { sub: userId, email } = req.ggid.claims;
  res.json({ user_id: userId, email });
});

// Public routes
app.get('/health', (req, res) => res.json({ status: 'ok' }));

app.listen(3000);
```

### Java (Servlet Filter)

```java
import io.ggid.sdk.GGIDAuthFilter;
import io.ggid.sdk.GGIDConfig;

@WebListener
public class AppConfig implements ServletContextInitializer {

    @Override
    public void onStartup(ServletContext ctx) {
        GGIDConfig config = GGIDConfig.builder()
            .jwksUrl("https://iam.example.com/.well-known/jwks.json")
            .tenantId("00000000-0000-0000-0000-000000000001")
            .build();

        // Register filter
        ctx.addFilter("ggidAuth", new GGIDAuthFilter(config))
           .addMappingForUrlPatterns(null, false, "/api/*");
    }
}
```

---

## 2. Permission Check API

Check if a user has permission to perform an action before processing.

### Go

```go
// Check permission before creating a user
func createUserHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    claims := ggid.ClaimsFromContext(ctx)
    userID := claims["sub"].(string)

    // Check permission
    allowed, err := ggidClient.Policy.Check(ctx, &ggid.CheckRequest{
        Subject:  userID,
        Action:   "write",
        Resource: "users",
    })
    if err != nil {
        http.Error(w, "policy check failed", 500)
        return
    }
    if !allowed {
        http.Error(w, "forbidden", 403)
        return
    }

    // Proceed to create user
    // ...
}
```

### Node.js

```typescript
// Express route with permission check
app.post('/api/users', ggid.verify(), async (req, res) => {
  const userId = req.ggid.claims.sub;

  try {
    const allowed = await client.policy.check({
      subject: userId,
      action: 'write',
      resource: 'users',
    });

    if (!allowed.allow) {
      return res.status(403).json({ error: 'insufficient permissions' });
    }

    // Create user
    res.status(201).json({ created: true });
  } catch (err) {
    res.status(500).json({ error: 'policy service unavailable' });
  }
});
```

### Java

```java
// Spring Boot controller with permission check
@PostMapping("/api/users")
public ResponseEntity<?> createUser(@RequestHeader("Authorization") String auth) {
    String userId = GGIDAuthFilter.getUserId(auth);

    CheckResponse response = ggidClient.policy().check(
        CheckRequest.builder()
            .subject(userId)
            .action("write")
            .resource("users")
            .build()
    );

    if (!response.isAllowed()) {
        return ResponseEntity.status(403).body(Map.of("error", "forbidden"));
    }

    return ResponseEntity.ok(Map.of("created", true));
}
```

---

## 3. Refresh Token Rotation

Implement secure refresh token rotation with automatic access token renewal.

### Go

```go
// TokenManager handles automatic refresh
type TokenManager struct {
    client       *ggid.Client
    refreshTok   string
    accessTok    string
    expiresAt    time.Time
    mu           sync.RWMutex
}

func (tm *TokenManager) ensureValid(ctx context.Context) (string, error) {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    if time.Now().Before(tm.expiresAt.Add(-30 * time.Second)) {
        return tm.accessTok, nil
    }

    resp, err := tm.client.Auth.Refresh(ctx, tm.refreshTok)
    if err != nil {
        return "", err  // Re-login required
    }

    tm.accessTok = resp.AccessToken
    tm.refreshTok = resp.RefreshToken  // Rotated!
    tm.expiresAt = time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
    return tm.accessTok, nil
}

// Usage with background refresh
func (tm *TokenManager) Start(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for {
            select {
            case <-ticker.C:
                tm.ensureValid(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()
}
```

### Node.js

```typescript
class TokenRotator {
  private client: GGIDClient;
  private accessToken: string | null = null;
  private refreshToken: string;
  private expiresIn: number = 0;

  constructor(client: GGIDClient, refreshToken: string) {
    this.client = client;
    this.refreshToken = refreshToken;
  }

  async getToken(): Promise<string> {
    if (this.accessToken && Date.now() < this.expiresIn - 30_000) {
      return this.accessToken;
    }
    return this.rotate();
  }

  private async rotate(): Promise<string> {
    const resp = await this.client.auth.refresh(this.refreshToken);
    this.accessToken = resp.access_token;
    this.refreshToken = resp.refresh_token; // Rotated!
    this.expiresIn = Date.now() + resp.expires_in * 1000;
    return this.accessToken;
  }
}
```

---

## 4. Social Login Integration

Integrate Google/GitHub/Microsoft social login with GGID.

### Node.js (Express + OAuth)

```typescript
import express from 'express';
import { GGIDClient } from '@ggid/node-sdk';

const app = express();
const client = new GGIDClient({ baseUrl: 'https://iam.example.com' });

// Step 1: Redirect to GGID social login
app.get('/auth/google', (req, res) => {
  const authUrl = client.oauth.getAuthUrl('google', {
    redirectUri: 'https://app.example.com/auth/callback',
    state: req.session.csrfToken,
  });
  res.redirect(authUrl);
});

// Step 2: Handle callback
app.get('/auth/callback', async (req, res) => {
  const { code, state } = req.query;

  if (state !== req.session.csrfToken) {
    return res.status(403).send('Invalid state');
  }

  // Exchange code for GGID JWT
  const tokens = await client.oauth.handleCallback('google', code);
  // tokens.access_token, tokens.refresh_token

  // Set secure cookie
  res.cookie('access_token', tokens.access_token, {
    httpOnly: true,
    secure: true,
    sameSite: 'strict',
    maxAge: 900000, // 15 min
  });

  res.redirect('/dashboard');
});
```

### Go

```go
// Social login redirect
func googleLoginHandler(w http.ResponseWriter, r *http.Request) {
    state := generateCSRFToken()
    session, _ := store.Get(r, "ggid-session")
    session.Values["oauth_state"] = state
    session.Save(r, w)

    authURL := ggidClient.OAuth.GetAuthURL("google", state,
        "https://app.example.com/auth/callback")
    http.Redirect(w, r, authURL, 302)
}

// Social login callback
func googleCallbackHandler(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")

    // Verify state
    session, _ := store.Get(r, "ggid-session")
    expectedState := session.Values["oauth_state"].(string)
    if state != expectedState {
        http.Error(w, "invalid state", 403)
        return
    }

    // Exchange for JWT
    tokens, err := ggidClient.OAuth.HandleCallback(r.Context(), "google", code)
    if err != nil {
        http.Error(w, "auth failed", 500)
        return
    }

    // Set cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "access_token",
        Value:    tokens.AccessToken,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })

    http.Redirect(w, r, "/dashboard", 302)
}
```

---

## 5. SCIM User Synchronization

Sync users from your system to GGID via SCIM 2.0.

### Go

```go
// Sync users from HR system to GGID
func syncUsers(ctx context.Context, hrUsers []HRUser) error {
    scim := ggidClient.SCIM

    for _, hr := range hrUsers {
        // Check if user exists
        existing, err := scim.GetUserByUserName(ctx, hr.Email)
        if err == nil {
            // Update
            existing.Name.GivenName = hr.FirstName
            existing.Name.FamilyName = hr.LastName
            existing.Active = hr.Active
            scim.UpdateUser(ctx, existing.ID, existing)
        } else {
            // Create
            scim.CreateUser(ctx, &ggid.SCIMUser{
                UserName: hr.Email,
                Emails: []ggid.SCIMEmail{{
                    Value:   hr.Email,
                    Primary: true,
                }},
                Name: ggid.SCIMName{
                    GivenName:  hr.FirstName,
                    FamilyName: hr.LastName,
                },
                Active: true,
            })
        }
    }
    return nil
}
```

### Node.js

```typescript
// Bulk sync with pagination
async function syncAllUsers(users: HRUser[]): Promise<void> {
  for (const user of users) {
    const existing = await client.scim.findUser(user.email);

    if (existing) {
      await client.scim.updateUser(existing.id, {
        name: { givenName: user.firstName, familyName: user.lastName },
        active: user.active,
      });
    } else {
      await client.scim.createUser({
        userName: user.email,
        emails: [{ value: user.email, primary: true }],
        active: true,
      });
    }
  }
  console.log(`Synced ${users.length} users`);
}
```

### Java

```java
// Spring Batch SCIM sync
@Component
public class UserSyncTasklet implements Tasklet {

    @Override
    public RepeatStatus execute(StepContribution contribution, ChunkContext chunkContext) {
        List<HRUser> users = hrService.getAllUsers();

        for (HRUser hr : users) {
            SCIMUser existing = scimClient.findUser(hr.getEmail());

            if (existing != null) {
                existing.setName(new SCIMName(hr.getFirstName(), hr.getLastName()));
                existing.setActive(hr.isActive());
                scimClient.updateUser(existing);
            } else {
                SCIMUser user = SCIMUser.builder()
                    .userName(hr.getEmail())
                    .email(hr.getEmail())
                    .active(true)
                    .build();
                scimClient.createUser(user);
            }
        }

        return RepeatStatus.FINISHED;
    }
}
```

---

## 6. Multi-Tenant Request Scoping

Make API calls scoped to a specific tenant.

### Go

```go
// Set tenant context for all API calls
ctx := context.Background()
ctx = ggid.WithTenant(ctx, "00000000-0000-0000-0000-000000000001")

// All subsequent calls are scoped to this tenant
users, err := client.Users.List(ctx, &ggid.ListOptions{Limit: 20})
roles, err := client.Roles.List(ctx)
```

### Node.js

```typescript
// Set tenant on client
const tenantClient = client.forTenant('00000000-0000-0000-0000-000000000001');

// All calls scoped
const users = await tenantClient.users.list({ limit: 20 });
const roles = await tenantClient.roles.list();
```

---

## 7. Audit Event Query

Query audit events for compliance reporting.

### Go

```go
events, total, err := client.Audit.List(ctx, &ggid.AuditFilter{
    TenantID:  tenantID,
    Action:    "user.login",       // Filter by action
    StartTime: &startTime,
    EndTime:   &endTime,
    Page:      1,
    Limit:     100,
})

for _, e := range events {
    fmt.Printf("[%s] %s by %s from %s\n",
        e.Timestamp.Format(time.RFC3339),
        e.Action,
        e.ActorID,
        e.RemoteIP,
    )
}
```

### Node.js

```typescript
const { events, total } = await client.audit.list({
  action: 'user.login',
  startTime: '2024-01-01T00:00:00Z',
  endTime: '2024-01-31T23:59:59Z',
  page: 1,
  limit: 100,
});

events.forEach(e => {
  console.log(`[${e.timestamp}] ${e.action} by ${e.actor_id} from ${e.remote_ip}`);
});
```

---

## 8. Webhook Signature Verification

Verify incoming webhooks from GGID.

### Go

```go
func verifyWebhook(body []byte, sig, ts string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(ts + "." + string(body)))
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(sig), []byte(expected))
}
```

### Node.js

```typescript
import crypto from 'crypto';

function verifyWebhook(body: Buffer, signature: string, timestamp: string, secret: string): boolean {
  const data = `${timestamp}.${body.toString()}`;
  const expected = 'sha256=' + crypto
    .createHmac('sha256', secret)
    .update(data)
    .digest('hex');
  return crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));
}
```

### Python

```python
import hmac, hashlib

def verify_webhook(body: bytes, signature: str, timestamp: str, secret: str) -> bool:
    data = f"{timestamp}.".encode() + body
    expected = "sha256=" + hmac.new(
        secret.encode(), data, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)
```

---

## References

- [SDK Guide](./sdk-guide.md) — Installation and basic usage
- [API Reference](./api-reference.md) — REST endpoints
- [Webhooks Guide](./webhooks-guide.md) — Event types and configuration
- [Integration Guide](./integration-guide.md) — Architecture patterns
