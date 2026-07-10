# GGID Integration Guide

A comprehensive guide for third-party developers integrating GGID IAM into
their applications.

## Table of Contents

- [Overview](#overview)
- [Architecture from the Client Perspective](#architecture-from-the-client-perspective)
- [Step 1: Obtain Your Tenant ID](#step-1-obtain-your-tenant-id)
- [Step 2: Choose Your SDK](#step-2-choose-your-sdk)
- [Step 3: Authentication Flow](#step-3-authentication-flow)
- [Step 4: JWT Verification](#step-4-jwt-verification)
- [Step 5: Middleware Configuration](#step-5-middleware-configuration)
- [Step 6: Multi-Tenant Configuration](#step-6-multi-tenant-configuration)
- [Step 7: Error Handling Best Practices](#step-7-error-handling-best-practices)
- [Step 8: Permission-Based Authorization](#step-8-permission-based-authorization)
- [Integration Patterns](#integration-patterns)

---

## Overview

GGID is a multi-tenant IAM platform. Your application integrates with it by:

1. Redirecting users to GGID for login (or calling the login API directly)
2. Receiving a JWT access token
3. Verifying the token on each request to your API
4. Checking permissions via the policy engine

```
┌──────────┐     login        ┌──────────┐     verify JWT     ┌─────────────┐
│  Client   │ ──────────────► │  GGID    │ ◄──────────────── │  Your App   │
│ (Browser/ │                 │  Auth    │                    │  Backend    │
│  Mobile)  │ ◄────────────── │  Service │ ──────────────►   │             │
│           │   JWT token      └──────────┘   user info       └─────────────┘
│           │                                                       │
│           │ ──────────────────────────────────────────────────► │
│           │  API request + Bearer token                           │
└──────────┘                                                       │
```

## Architecture from the Client Perspective

All API calls go through the **Gateway** (port 8080). The Gateway:

- Verifies JWT signatures (RS256) using the public key or JWKS endpoint
- Injects `X-Tenant-ID`, `X-User-ID`, and `X-Request-ID` headers
- Rate-limits requests (default: per-IP on auth endpoints)
- Reverse-proxies to the appropriate backend service

**You never need to talk to backend services directly.** Use the Gateway URL
for everything.

## Step 1: Obtain Your Tenant ID

Every GGID deployment has at least one tenant. The default tenant is:

```
00000000-0000-0000-0000-000000000001
```

For multi-tenant setups, contact your GGID administrator to create a tenant
and get its UUID. You will need this ID for:

- The `X-Tenant-ID` header on every request
- SDK configuration (`tenantId` parameter)

## Step 2: Choose Your SDK

| Language | Package | Install |
|----------|---------|---------|
| Go | `github.com/ggid/ggid/sdk/go` | `go get github.com/ggid/ggid/sdk/go` |
| Node.js | `@ggid/node` | `npm install @ggid/node jose` |
| Java | `sdk/java` | Maven/Gradle (see `sdk/java/README.md`) |
| curl | — | No SDK needed |

### Go

```go
import ggid "github.com/ggid/ggid/sdk/go"

client := ggid.New("https://iam.example.com",
    ggid.WithAPIKey("your-api-key"),
    ggid.WithJWKS(15*time.Minute),
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
});
```

### Direct HTTP (curl)

```bash
curl -X POST https://iam.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"demo","password":"SecurePass@123"}'
```

## Step 3: Authentication Flow

### Password Login (Resource Owner Password Credentials)

```
Client ──POST /api/v1/auth/login──► GGID
       ◄──{ access_token, refresh_token }──
```

```bash
RESPONSE=$(curl -s -X POST https://iam.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"demo","password":"SecurePass@123"}')

ACCESS_TOKEN=$(echo $RESPONSE | jq -r .access_token)
REFRESH_TOKEN=$(echo $RESPONSE | jq -r .refresh_token)
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "rt_abc123...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### Token Refresh

Access tokens expire after 1 hour. Use the refresh token to get a new pair:

```bash
curl -s -X POST https://iam.example.com/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"'$REFRESH_TOKEN'"}'
```

> Refresh tokens are **rotated** — each refresh returns a new refresh token.
> The old one becomes invalid.

### OAuth2 Authorization Code Flow

For third-party apps and SPAs:

```
1. Redirect user to:
   https://iam.example.com/oauth/authorize?
     response_type=code&
     client_id=YOUR_CLIENT_ID&
     redirect_uri=https://yourapp.com/callback&
     scope=openid profile email&
     state=RANDOM_STRING

2. User logs in at GGID, gets redirected back:
   https://yourapp.com/callback?code=AUTH_CODE&state=RANDOM_STRING

3. Exchange code for tokens:
   POST https://iam.example.com/oauth/token
   grant_type=authorization_code
   code=AUTH_CODE
   client_id=YOUR_CLIENT_ID
   client_secret=YOUR_CLIENT_SECRET
```

## Step 4: JWT Verification

### Token Structure

The JWT (RS256-signed) contains these claims:

| Claim | Description |
|-------|-------------|
| `sub` | User UUID |
| `tenant_id` | Tenant UUID |
| `username` | Username |
| `email` | Email address |
| `roles` | Array of role keys (e.g. `["admin","editor"]`) |
| `scope` | Space-separated OAuth scopes |
| `iss` | Issuer (`ggid-auth`) |
| `exp` | Expiry timestamp |
| `iat` | Issued-at timestamp |

### Verification Methods

**Method A: JWKS (Recommended)**

Fetch the public key from GGID's JWKS endpoint and verify locally:

```go
// Go — JWKS cached for 15 minutes
client := ggid.New("https://iam.example.com",
    ggid.WithJWKS(15*time.Minute),
)
userInfo, err := client.VerifyToken(ctx, accessToken)
```

```typescript
// Node.js — jose handles JWKS caching automatically
const verifier = new JWTVerifier({
  jwksUrl: 'https://iam.example.com/.well-known/jwks.json',
  issuer: 'ggid',
});
const claims = await verifier.verify(token);
```

JWKS endpoint: `GET https://iam.example.com/.well-known/jwks.json`

**Method B: Offline (No Signature Verification)**

Parse claims without verifying the signature. Use only when you trust
the transport layer (e.g., behind the Gateway which already verifies):

```go
// Go — no WithJWKS option
client := ggid.New("https://iam.example.com")
userInfo, err := client.VerifyToken(ctx, accessToken)
```

> **Warning:** Offline mode does NOT verify that the token was signed by GGID.
> Only use behind the Gateway or in trusted environments.

## Step 5: Middleware Configuration

### Go (net/http)

```go
mux := http.NewServeMux()

// Public routes (no JWT required)
mux.HandleFunc("/healthz", healthHandler)
mux.HandleFunc("/public", publicHandler)

// Protected routes
mux.HandleFunc("/api/profile", profileHandler)
mux.HandleFunc("/api/admin", adminHandler)

// Wrap with GGID JWT verification
handler := client.Middleware(mux, ggid.MiddlewareConfig{
    PublicPaths: []string{"/healthz", "/public"},
    TenantID:    "00000000-0000-0000-0000-000000000001",
})

http.ListenAndServe(":8080", handler)
```

Inside handlers, access user info from context:

```go
func profileHandler(w http.ResponseWriter, r *http.Request) {
    user := ggid.UserFromContext(r.Context())
    if user == nil {
        http.Error(w, "not authenticated", http.StatusUnauthorized)
        return
    }
    json.NewEncoder(w).Encode(map[string]any{
        "user_id":  user.UserID,
        "username": user.Username,
        "email":    user.Email,
        "roles":    user.Roles,
    })
}
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
  res.json({
    user_id: user.sub,
    username: user.username,
    email: user.email,
  });
});
```

Public paths are automatically skipped: `/healthz`, `/api/v1/auth/*`, `/oauth/*`.

### Node.js (Fastify)

```typescript
app.addHook('preHandler', async (request, reply) => {
  if (request.url.startsWith('/healthz')) return;

  const auth = request.headers.authorization;
  if (!auth?.startsWith('Bearer ')) {
    return reply.code(401).send({ error: 'missing token' });
  }

  const verifier = new JWTVerifier({
    jwksUrl: process.env.GGID_JWKS_URL!,
  });
  try {
    request.ggUser = await verifier.verify(auth.slice(7));
  } catch {
    return reply.code(401).send({ error: 'invalid token' });
  }
});
```

## Step 6: Multi-Tenant Configuration

### Single-Tenant (Simple)

If your app serves one tenant, hardcode the tenant ID:

```go
client := ggid.New("https://iam.example.com",
    ggid.WithAPIKey("key"),
).Middleware(mux, ggid.MiddlewareConfig{
    TenantID: "00000000-0000-0000-0000-000000000001",
})
```

### Multi-Tenant (Dynamic)

If your app serves multiple tenants, extract the tenant ID from the JWT:

```go
func tenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := ggid.UserFromContext(r.Context())
        if user != nil && user.TenantID != "" {
            r.Header.Set("X-Tenant-ID", user.TenantID)
        }
        next.ServeHTTP(w, r)
    })
}
```

In a multi-tenant setup:
1. Users register under a specific tenant
2. The JWT contains `tenant_id` in its claims
3. Your app reads `tenant_id` from the verified token
4. All downstream API calls include it as `X-Tenant-ID`

## Step 7: Error Handling Best Practices

### Go

```go
user, err := client.GetUser(ctx, userID)
if err != nil {
    var apiErr *ggid.APIError
    if errors.As(err, &apiErr) {
        switch {
        case apiErr.IsNotFound():
            // 404 — user doesn't exist
            return fmt.Errorf("user not found")

        case apiErr.IsUnauthorized():
            // 401 — token expired, try refresh
            newTokens, refreshErr := client.RefreshToken(ctx, refreshToken)
            if refreshErr != nil {
                // Refresh also failed — user must re-login
                return fmt.Errorf("session expired, please login again")
            }
            // Retry with new token
            saveTokens(newTokens)
            return client.GetUser(ctx, userID) // retry

        case apiErr.IsForbidden():
            // 403 — insufficient permissions
            return fmt.Errorf("access denied")

        case apiErr.IsConflict():
            // 409 — duplicate resource
            return fmt.Errorf("user already exists")

        case apiErr.IsRateLimited():
            // 429 — too many requests
            time.Sleep(5 * time.Second)
            return client.GetUser(ctx, userID) // retry with backoff
        }
    }
    return fmt.Errorf("unexpected error: %w", err)
}
```

### Node.js

```typescript
async function withRetry<T>(fn: () => Promise<T>, refreshToken?: string): Promise<T> {
  try {
    return await fn();
  } catch (err: any) {
    const status = err.message.match(/GGID API (\d+)/)?.[1];

    if (status === '401' && refreshToken) {
      // Token expired — refresh and retry
      const newTokens = await client.refreshToken(refreshToken);
      saveTokens(newTokens);
      return await fn(); // retry once
    }

    if (status === '429') {
      // Rate limited — wait and retry
      await new Promise(r => setTimeout(r, 5000));
      return await fn();
    }

    throw err;
  }
}
```

### Token Expiry Handling Pattern

```go
// Wrap API calls with automatic token refresh
func (s *Session) Do(ctx context.Context, fn func(string) error) error {
    // Check if token is about to expire (within 5 minutes)
    if s.tokenExpiry.Before(time.Now().Add(5 * time.Minute)) {
        tokens, err := s.client.RefreshToken(ctx, s.refreshToken)
        if err != nil {
            return fmt.Errorf("token refresh failed: %w", err)
        }
        s.accessToken = tokens.AccessToken
        s.refreshToken = tokens.RefreshToken
        s.tokenExpiry = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
    }
    return fn(s.accessToken)
}
```

## Step 8: Permission-Based Authorization

### Check Permission via API (RBAC + ABAC)

```go
allowed, err := client.CheckPermission(ctx, user.UserID, "documents:sensitive", "read")
if err != nil {
    return fmt.Errorf("permission check failed: %w", err)
}
if !allowed {
    return ErrForbidden
}
```

```typescript
const result = await client.checkPermission(accessToken, 'documents:sensitive', 'read');
if (!result.allowed) {
  return res.status(403).json({ error: 'access denied' });
}
```

### Local Role Check (from JWT claims)

For simple role checks, use the roles already in the JWT — no API call needed:

```go
// Go
handler := client.RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
    // Only users with "admin" role reach here
    w.Write([]byte("Admin dashboard"))
})
```

```typescript
// Node.js
app.delete('/api/users/:id',
  requirePermission('iam:users', 'delete'),
  handler,
);
```

### Middleware Chaining

```go
// Chain: JWT verification → Role check → Permission check → Handler
protectedHandler := client.Middleware(
    client.RequireRole("admin",
        client.RequirePermission("settings:security", "write",
            settingsHandler,
        ),
    ),
    ggid.MiddlewareConfig{
        TenantID: "00000000-0000-0000-0000-000000000001",
    },
)
```

## Integration Patterns

### Pattern 1: SPA + Backend API

```
Browser ──login──► GGID Auth ──JWT──► Browser
Browser ──API call + JWT──► Your Backend ──verify JWT──► GGID JWKS
```

- Frontend stores JWT in memory or httpOnly cookie
- Backend verifies JWT using JWKS endpoint
- No SDK needed on frontend — just use fetch/axios

### Pattern 2: Server-Side Rendered (SSR)

```
Browser ──request──► Your Server ──login if needed──► GGID
Your Server ──JWT in session──► GGID API (data calls)
Your Server ──HTML──► Browser
```

- Server handles login and stores tokens server-side
- Server makes all API calls to GGID

### Pattern 3: Microservices (Service-to-Service)

```
Gateway ──JWT──► Service A ──check permission──► Policy API
                                          ──user info──► Identity API
```

- Each service verifies the JWT (using JWKS)
- Services call the Policy API for authorization decisions
- No shared session state needed

### Pattern 4: Machine-to-Machine (Client Credentials)

```
Service A ──client_credentials──► GGID OAuth ──JWT──► Service A
Service A ──JWT──► GGID API (data calls)
```

- No user interaction
- Service authenticates with `client_id` + `client_secret`
- Receives a JWT with service-level scopes

## Further Reading

- [API Reference (OpenAPI)](./openapi.yaml)
- [Quick Start Guide](./quick-start.md)
- [Go SDK README](../sdk/go/README.md)
- [Node.js SDK README](../sdk/node/README.md)
- [Deployment Guide](./deployment.md)
- [Troubleshooting](./troubleshooting.md)
