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

## 9. Next.js Middleware for Session Validation

Validate GGID JWTs in Next.js middleware to protect pages and API routes
without a round-trip to the auth server.

### middleware.ts (App Router)

```typescript
// app/middleware.ts
import { NextRequest, NextResponse } from 'next/server';
import jwt from 'jsonwebtoken';
import jwksClient from 'jwks-rsa';

const client = jwksClient({
  jwksUri: process.env.GGID_JWKS_URL ||
    'https://iam.example.com/.well-known/jwks.json',
  cache: true,
  cacheMaxAge: 600_000, // 10 min
  rateLimit: true,
});

function getSigningKey(header: jwt.JwtHeader): Promise<string> {
  return new Promise((resolve, reject) => {
    client.getSigningKey(header.kid, (err, key) => {
      if (err) reject(err);
      else resolve(key.getPublicKey());
    });
  });
}

export async function middleware(request: NextRequest) {
  // Skip public routes
  if (request.nextUrl.pathname.startsWith('/login') ||
      request.nextUrl.pathname.startsWith('/register') ||
      request.nextUrl.pathname.startsWith('/api/health')) {
    return NextResponse.next();
  }

  // Extract token from cookie or Authorization header
  const token =
    request.cookies.get('access_token')?.value ||
    request.headers.get('authorization')?.replace('Bearer ', '');

  if (!token) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  try {
    const decoded = jwt.decode(token, { complete: true });
    if (!decoded) throw new Error('invalid token');

    const signingKey = await getSigningKey(decoded.header);

    const payload = jwt.verify(token, signingKey, {
      algorithms: ['RS256'],
      issuer: process.env.GGID_ISSUER || 'https://iam.example.com',
    }) as GGIDToken;

    // Check tenant match
    const tenantHeader = request.headers.get('x-tenant-id');
    if (tenantHeader && tenantHeader !== payload.tenant_id) {
      return NextResponse.json({ error: 'tenant mismatch' }, { status: 403 });
    }

    // Attach claims to headers for downstream handlers
    const requestHeaders = new Headers(request.headers);
    requestHeaders.set('x-ggid-user-id', payload.sub);
    requestHeaders.set('x-ggid-tenant-id', payload.tenant_id);
    requestHeaders.set('x-ggid-scopes', payload.scope || '');

    return NextResponse.next({
      request: { headers: requestHeaders },
    });
  } catch (err) {
    // Token expired or invalid — redirect to refresh
    const refreshUrl = new URL('/api/auth/refresh', request.url);
    refreshUrl.searchParams.set('redirect', request.nextUrl.pathname);
    return NextResponse.redirect(refreshUrl);
  }
}

interface GGIDToken {
  sub: string;
  tenant_id: string;
  scope: string;
  exp: number;
}

export const config = {
  matcher: [
    /*
     * Match all paths except:
     * - _next/static, _next/image, favicon.ico
     * - Public assets
     */
    '/((?!_next/static|_next/image|favicon.ico|public).*)',
  ],
};
```

### Token Refresh API Route

```typescript
// app/api/auth/refresh/route.ts
import { NextRequest, NextResponse } from 'next/server';

export async function POST(request: NextRequest) {
  const refreshToken = request.cookies.get('refresh_token')?.value;

  if (!refreshToken) {
    return NextResponse.json({ error: 'no refresh token' }, { status: 401 });
  }

  const res = await fetch(
    `${process.env.GGID_GATEWAY_URL}/api/v1/auth/refresh`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': process.env.GGID_TENANT_ID!,
      },
      body: JSON.stringify({ refresh_token: refreshToken }),
    }
  );

  if (!res.ok) {
    return NextResponse.json({ error: 'refresh failed' }, { status: 401 });
  }

  const tokens = await res.json();

  const response = NextResponse.json(tokens);
  response.cookies.set('access_token', tokens.access_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'strict',
    maxAge: 900, // 15 minutes
    path: '/',
  });
  response.cookies.set('refresh_token', tokens.refresh_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'strict',
    maxAge: 604800, // 7 days
    path: '/',
  });

  return response;
}
```

---

## 10. Express.js Token Refresh Interceptor

Automatically refresh expired access tokens in an Express.js backend, caching
the result to avoid duplicate refresh calls.

### Token Refresh Middleware

```typescript
// middleware/token-refresh.ts
import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';

const refreshCache = new Map<string, { tokens: TokenResponse; expiry: number }>();

interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export async function autoRefreshToken(
  req: Request & { ggid?: { claims: any } },
  res: Response,
  next: NextFunction
) {
  const accessToken = req.headers.authorization?.replace('Bearer ', '');
  const refreshToken = req.headers['x-refresh-token'] as string;
  const tenantId = req.headers['x-tenant-id'] as string;

  if (!accessToken) {
    return next();
  }

  // Check if access token is still valid
  try {
    const decoded = jwt.decode(accessToken) as { exp: number };
    const now = Date.now() / 1000;

    // Token still valid for more than 60s — proceed
    if (decoded.exp - now > 60) {
      req.ggid = { claims: decoded };
      return next();
    }
  } catch {
    // Token malformed — let it fail downstream
    return next();
  }

  // Token is expiring soon — try to refresh
  if (!refreshToken) {
    return res.status(401).json({
      error: 'access token expired',
      code: 'TOKEN_EXPIRED',
    });
  }

  // Check cache to prevent concurrent refresh calls
  const cached = refreshCache.get(refreshToken);
  if (cached && cached.expiry > Date.now()) {
    res.setHeader('X-New-Access-Token', cached.tokens.access_token);
    req.headers.authorization = `Bearer ${cached.tokens.access_token}`;
    req.ggid = { claims: jwt.decode(cached.tokens.access_token) };
    return next();
  }

  // Call GGID refresh endpoint
  try {
    const response = await fetch(
      `${process.env.GGID_GATEWAY_URL}/api/v1/auth/refresh`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      }
    );

    if (!response.ok) {
      return res.status(401).json({
        error: 'refresh token invalid',
        code: 'REFRESH_FAILED',
      });
    }

    const tokens: TokenResponse = await response.json();

    // Cache for the lifetime of the access token
    refreshCache.set(refreshToken, {
      tokens,
      expiry: Date.now() + tokens.expires_in * 1000 - 60_000, // 1 min before expiry
    });

    // Return new tokens to client via headers
    res.setHeader('X-New-Access-Token', tokens.access_token);
    res.setHeader('X-New-Refresh-Token', tokens.refresh_token);

    req.headers.authorization = `Bearer ${tokens.access_token}`;
    req.ggid = { claims: jwt.decode(tokens.access_token) };
    next();
  } catch (err) {
    return res.status(502).json({
      error: 'auth service unavailable',
      code: 'AUTH_UNAVAILABLE',
    });
  }
}
```

### Usage

```typescript
import express from 'express';
import { autoRefreshToken } from './middleware/token-refresh';

const app = express();

// Apply to protected routes
app.use('/api', autoRefreshToken);

app.get('/api/profile', (req, res) => {
  const userId = req.ggid?.claims.sub;
  res.json({ userId });
});

// Client-side interceptor (Axios example)
axios.interceptors.response.use(
  (response) => {
    // Check for rotated tokens in response headers
    const newAccessToken = response.headers['x-new-access-token'];
    const newRefreshToken = response.headers['x-new-refresh-token'];
    if (newAccessToken) {
      localStorage.setItem('access_token', newAccessToken);
    }
    if (newRefreshToken) {
      localStorage.setItem('refresh_token', newRefreshToken);
    }
    return response;
  },
  async (error) => {
    if (error.response?.status === 401 &&
        error.response.data?.code === 'TOKEN_EXPIRED') {
      // Redirect to login
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

---

## 11. Go Service-to-Service Authentication

For microservices calling each other through the GGID Gateway, use a shared
service token or client credentials grant.

### Client Credentials Flow

```go
// internal/auth/service_token.go
package auth

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "sync"
    "time"
)

type ServiceTokenProvider struct {
    gatewayURL  string
    clientID    string
    clientSecret string
    tenantID    string

    mu          sync.Mutex
    cachedToken string
    expiresAt   time.Time
}

func NewServiceTokenProvider(gatewayURL, clientID, clientSecret, tenantID string) *ServiceTokenProvider {
    return &ServiceTokenProvider{
        gatewayURL:  gatewayURL,
        clientID:    clientID,
        clientSecret: clientSecret,
        tenantID:    tenantID,
    }
}

func (p *ServiceTokenProvider) GetToken(ctx context.Context) (string, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Return cached token if still valid
    if time.Now().Before(p.expiresAt.Add(-30 * time.Second)) {
        return p.cachedToken, nil
    }

    // Request new token via client_credentials grant
    data := url.Values{
        "grant_type":    {"client_credentials"},
        "client_id":     {p.clientID},
        "client_secret": {p.clientSecret},
        "scope":         {"users:read users:write policy:check"},
    }

    req, _ := http.NewRequestWithContext(ctx, "POST",
        p.gatewayURL+"/oauth/token",
        strings.NewReader(data.Encode()),
    )
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("X-Tenant-ID", p.tenantID)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("requesting service token: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("token endpoint returned %d", resp.StatusCode)
    }

    var result struct {
        AccessToken string `json:"access_token"`
        ExpiresIn   int    `json:"expires_in"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("decoding token response: %w", err)
    }

    p.cachedToken = result.AccessToken
    p.expiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

    return p.cachedToken, nil
}

// AuthenticatedHTTPClient wraps http.Client with automatic service token injection
type AuthenticatedHTTPClient struct {
    provider *ServiceTokenProvider
    client   *http.Client
}

func (c *AuthenticatedHTTPClient) Do(req *http.Request) (*http.Response, error) {
    token, err := c.provider.GetToken(req.Context())
    if err != nil {
        return nil, fmt.Errorf("getting service token: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("X-Tenant-ID", c.provider.tenantID)

    return c.client.Do(req)
}
```

### Usage

```go
func main() {
    tokenProvider := auth.NewServiceTokenProvider(
        "https://iam.example.com",
        "my-service-client",
        os.Getenv("OAUTH_CLIENT_SECRET"),
        "00000000-0000-0000-0000-000000000001",
    )

    client := &auth.AuthenticatedHTTPClient{
        provider: tokenProvider,
        client:   &http.Client{Timeout: 30 * time.Second},
    }

    // Call another GGID service
    req, _ := http.NewRequest("GET",
        "https://iam.example.com/api/v1/users?limit=10", nil)

    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    var users []User
    json.NewDecoder(resp.Body).Decode(&users)
    fmt.Printf("Retrieved %d users\n", len(users))
}
```

---

## 12. Spring Boot SSO Integration

Integrate GGID as a SAML or OIDC identity provider for a Spring Boot
application using Spring Security.

### OIDC Integration (application.yml)

```yaml
spring:
  security:
    oauth2:
      client:
        registration:
          ggid:
            client-id: my-spring-app
            client-secret: ${OAUTH_CLIENT_SECRET}
            scope: openid, profile, email
            redirect-uri: "{baseUrl}/login/oauth2/code/{registrationId}"
            client-name: GGID
            authorization-grant-type: authorization_code
        provider:
          ggid:
            issuer-uri: https://iam.example.com
            authorization-uri: https://iam.example.com/oauth/authorize
            token-uri: https://iam.example.com/oauth/token
            user-info-uri: https://iam.example.com/oauth/userinfo
            jwk-set-uri: https://iam.example.com/.well-known/jwks.json
            user-name-attribute: sub
```

### Security Configuration

```java
@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/", "/login**", "/webjars/**", "/error").permitAll()
                .requestMatchers("/admin/**").hasAuthority("ROLE_admin")
                .requestMatchers("/api/**").authenticated()
                .anyRequest().authenticated()
            )
            .oauth2Login(oauth2 -> oauth2
                .loginPage("/login")
                .defaultSuccessUrl("/dashboard", true)
                .userInfoEndpoint(userInfo -> userInfo
                    .oidcUserService(new GGIDOidcUserService())
                )
            )
            .oauth2ResourceServer(oauth2 -> oauth2
                .jwt(jwt -> jwt
                    .jwkSetUri("https://iam.example.com/.well-known/jwks.json")
                    .jwtAuthenticationConverter(new GGIDJwtConverter())
                )
            )
            .logout(logout -> logout
                .logoutSuccessUrl("https://iam.example.com/oauth/logout?redirect_uri=http://localhost:8080")
                .invalidateHttpSession(true)
                .deleteCookies("JSESSIONID")
            )
            .csrf(csrf -> csrf.ignoringRequestMatchers("/api/**"))
            .sessionManagement(session -> session
                .sessionCreationPolicy(SessionCreationPolicy.IF_REQUIRED)
                .maximumSessions(5)
            );

        return http.build();
    }
}
```

### Custom JWT Converter (Extract GGID Claims)

```java
public class GGIDJwtConverter implements Converter<Jwt, AbstractAuthenticationToken> {

    @Override
    public AbstractAuthenticationToken convert(Jwt jwt) {
        Collection<GrantedAuthority> authorities = new ArrayList<>();

        // Extract roles from GGID JWT
        String roles = jwt.getClaimAsString("roles");
        if (roles != null) {
            for (String role : roles.split(",")) {
                authorities.add(new SimpleGrantedAuthority("ROLE_" + role.trim()));
            }
        }

        // Extract scopes
        String scope = jwt.getClaimAsString("scope");
        if (scope != null) {
            for (String s : scope.split(" ")) {
                authorities.add(new SimpleGrantedAuthority("SCOPE_" + s));
            }
        }

        return new JwtAuthenticationToken(jwt, authorities, jwt.getSubject());
    }
}
```

### Controller Example

```java
@RestController
@RequestMapping("/api")
public class UserController {

    @GetMapping("/me")
    public Map<String, Object> getCurrentUser(@AuthenticationPrincipal Jwt jwt) {
        return Map.of(
            "user_id", jwt.getSubject(),
            "email", jwt.getClaimAsString("email"),
            "tenant_id", jwt.getClaimAsString("tenant_id"),
            "roles", jwt.getClaimAsString("roles")
        );
    }

    @GetMapping("/users")
    @PreAuthorize("hasRole('admin')")
    public List<User> listUsers() {
        // Only accessible by users with admin role
        return userService.findAll();
    }
}
```

---

## References

- [SDK Guide](./sdk-guide.md) — Installation and basic usage
- [API Reference](./api-reference.md) — REST endpoints
- [Webhooks Guide](./webhooks-guide.md) — Event types and configuration
- [Integration Guide](./integration-guide.md) — Architecture patterns
