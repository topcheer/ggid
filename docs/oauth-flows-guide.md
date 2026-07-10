# OAuth 2.1 Flows Guide

Complete reference for all OAuth 2.1 grant types supported by GGID. Each flow
includes sequence diagrams, when to use it, configuration, and code examples.

---

## Table of Contents

- [Flow Selection Matrix](#flow-selection-matrix)
- [Authorization Code + PKCE](#authorization-code--pkce)
- [Client Credentials](#client-credentials)
- [Device Authorization (RFC 8628)](#device-authorization-rfc-8628)
- [CIBA (RFC 9101)](#ciba-rfc-9101)
- [PAR — Pushed Authorization Request (RFC 9126)](#par--pushed-authorization-request-rfc-9126)
- [Token Exchange (RFC 8693)](#token-exchange-rfc-8693)
- [Refresh Token](#refresh-token)

---

## Flow Selection Matrix

| Flow | User Interaction | Client Type | Use Case |
|------|:----------------:|-------------|----------|
| Auth Code + PKCE | Yes (browser) | All clients | Web apps, SPAs, mobile |
| Client Credentials | No | Confidential | Server-to-server (M2M) |
| Device Code | Yes (separate device) | Public | Smart TVs, IoT, CLI |
| CIBA | Yes (async, poll) | Confidential | Banking, mobile-first |
| PAR | Yes (browser) | All clients | Enhanced security layer |
| Token Exchange | No | Confidential | Delegation, tiered APIs |
| Refresh Token | No | All clients | Renew expired access token |

### Decision Tree

```
Is there a user involved?
├── No → Client Credentials (M2M)
└── Yes
    ├── Can the client keep a secret?
    │   ├── No (public: SPA, mobile, TV)
    │   │   ├── Input-constrained device (TV, IoT)?
    │   │   │   └── Device Code
    │   │   └── Standard browser/mobile
    │   │       └── Auth Code + PKCE
    │   └── Yes (confidential: server-side)
    │       ├── Back-channel auth (no browser redirect)?
    │       │   └── CIBA
    │       └── Standard web app
    │           └── Auth Code + PKCE
    └── Need to delegate token to downstream API?
        └── Token Exchange
```

---

## Authorization Code + PKCE

The primary OAuth 2.1 flow. Required for all applications involving user
interaction.

### When to Use

- Web applications (server-rendered)
- Single Page Applications (SPA)
- Mobile apps (iOS/Android)
- Desktop applications

### Sequence Diagram

```
User        Client (App)         GGID Auth Server      Browser
 │              │                      │                   │
 │ 1. Access    │                      │                   │
 ├────────────►│                      │                   │
 │              │ 2. Generate PKCE     │                   │
 │              │    code_verifier     │                   │
 │              │    code_challenge    │                   │
 │              │                      │                   │
 │              │ 3. Redirect to auth  │                   │
 │              │    ?response_type=code                  │
 │              │    &code_challenge=...                   │
 │              │    &code_challenge_method=S256           │
 │              ├─────────────────────────────────────────►│
 │              │                      │ 4. Show login     │
 │              │                      │◄──────────────────┤
 │ 5. Login     │                      │                   │
 ├─────────────────────────────────────────────────────────►│
 │              │                      │ 6. User consents  │
 │              │                      │ 7. Redirect with  │
 │              │                      │    code           │
 │              │◄─────────────────────────────────────────┤
 │              │ 8. Exchange code     │                   │
 │              │    + code_verifier   │                   │
 │              │    for tokens        │                   │
 │              ├─────────────────────►│                   │
 │              │                      │ 9. Verify PKCE    │
 │              │                      │    issue tokens   │
 │              │ 10. access_token     │                   │
 │              │     id_token         │                   │
 │              │     refresh_token    │                   │
 │              │◄────────────────────┤                   │
 │ 11. App      │                      │                   │
 │     loaded   │                      │                   │
 │◄────────────┤                      │                   │
```

### Step-by-Step

#### 1. Generate PKCE

```python
import secrets, hashlib, base64

code_verifier = secrets.token_urlsafe(64)
code_challenge = base64.urlsafe_b64encode(
    hashlib.sha256(code_verifier.encode()).digest()
).rstrip(b'=').decode()
```

#### 2. Authorization Request

```
GET /oauth/authorize?
    response_type=code
    &client_id=your-client-id
    &redirect_uri=https://app.example.com/callback
    &scope=openid profile email
    &state=random-state
    &code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM
    &code_challenge_method=S256
```

#### 3. Token Exchange

```bash
curl -X POST https://iam.example.com/oauth/token \
  -d "grant_type=authorization_code" \
  -d "code=received-auth-code" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "client_id=your-client-id" \
  -d "code_verifier=original-code-verifier"
```

#### 4. Token Response

```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "rt-xxx",
  "id_token": "eyJhbG...",
  "scope": "openid profile email"
}
```

### Configuration

```yaml
oauth:
  clients:
    - client_id: "web-app"
      grant_types: ["authorization_code", "refresh_token"]
      response_types: ["code"]
      redirect_uris: ["https://app.example.com/callback"]
      pkce_required: true
      access_token_lifetime: "15m"
      refresh_token_lifetime: "24h"
```

---

## Client Credentials

Server-to-server flow with no user involvement.

### When to Use

- Microservice-to-microservice communication
- Background jobs and cron tasks
- CI/CD pipelines
- API gateways fetching configuration

### Sequence Diagram

```
Service A                    GGID Auth Server           Target API
   │                               │                        │
   │ 1. POST /oauth/token          │                        │
   │    grant_type=client_credentials                       │
   │    client_id + client_secret                           │
   ├──────────────────────────────►│                        │
   │                               │ 2. Validate client     │
   │ 3. access_token (no refresh)  │                        │
   │◄──────────────────────────────┤                        │
   │                               │                        │
   │ 4. API call with Bearer token │                        │
   ├───────────────────────────────────────────────────────►│
   │ 5. API response               │                        │
   │◄───────────────────────────────────────────────────────┤
```

### Request

```bash
curl -X POST https://iam.example.com/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=service-account-id" \
  -d "client_secret=service-secret" \
  -d "scope=users:read users:write"
```

### Response

```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "users:read users:write"
}
```

> No `refresh_token` is issued — the client simply requests a new token when
> the current one expires.

### Configuration

```yaml
oauth:
  clients:
    - client_id: "backend-service"
      grant_types: ["client_credentials"]
      client_secret: "${SERVICE_CLIENT_SECRET}"
      token_endpoint_auth_method: "client_secret_post"
      access_token_lifetime: "1h"
      scopes: ["users:read", "users:write"]
```

### mTLS Client Auth (RFC 8705)

For enhanced security, use mutual TLS instead of client_secret:

```bash
curl -X POST https://iam.example.com/oauth/token \
  --cert client.crt \
  --key client.key \
  -d "grant_type=client_credentials" \
  -d "client_id=service-account-id"
```

---

## Device Authorization (RFC 8628)

For devices with limited input capability (smart TVs, game consoles, CLI tools).

### When to Use

- Smart TVs and streaming devices
- IoT devices with no keyboard
- CLI tools (gcloud, aws CLI pattern)

### Sequence Diagram

```
Device              GGID Auth Server          User's Phone/Browser
  │                       │                          │
  │ 1. Request device     │                          │
  │    code               │                          │
  ├──────────────────────►│                          │
  │ 2. device_code        │                          │
  │    user_code          │                          │
  │    verification_uri   │                          │
  │◄──────────────────────┤                          │
  │                       │                          │
  │ 3. Display:           │                          │
  │    "Visit example.com │                          │
  │     /device           │                          │
  │     and enter code:   │                          │
  │     ABCD-1234"        │                          │
  │                       │                          │
  │                       │ 4. User visits URI       │
  │                       │    enters code           │
  │                       │◄─────────────────────────┤
  │                       │ 5. User authenticates    │
  │                       │    approves device        │
  │                       │                          │
  │ 6. Poll token endpoint│                          │
  │    (every 5 seconds)  │                          │
  ├──────────────────────►│                          │
  │ 7. authorization_pending                          │
  │◄──────────────────────┤                          │
  │                       │                          │
  │ 8. Poll again         │                          │
  ├──────────────────────►│                          │
  │ 9. access_token       │                          │
  │◄──────────────────────┤                          │
```

### Step 1: Request Device Code

```bash
curl -X POST https://iam.example.com/oauth/device/code \
  -d "client_id=device-client-id" \
  -d "scope=openid profile"
```

```json
{
  "device_code": "GmRh...",
  "user_code": "ABCD-1234",
  "verification_uri": "https://iam.example.com/device",
  "verification_uri_complete": "https://iam.example.com/device?user_code=ABCD-1234",
  "expires_in": 1800,
  "interval": 5
}
```

### Step 2: User Authenticates

The user visits `verification_uri` on a separate device (phone/laptop), enters
the `user_code`, and approves.

### Step 3: Poll for Token

```bash
curl -X POST https://iam.example.com/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=GmRh..." \
  -d "client_id=device-client-id"
```

**While pending:**
```json
{ "error": "authorization_pending" }
```

**After approval:**
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "rt-xxx",
  "id_token": "eyJhbG..."
}
```

### Polling Rules

| Response | Action |
|----------|--------|
| `authorization_pending` | Wait `interval` seconds, poll again |
| `slow_down` | Increase interval by 5 seconds |
| `access_denied` | User denied — stop polling |
| `expired_token` | Device code expired — restart flow |

### Configuration

```yaml
oauth:
  clients:
    - client_id: "smart-tv-app"
      grant_types: ["urn:ietf:params:oauth:grant-type:device_code", "refresh_token"]
      pkce_required: false
      device_code:
        expires_in: "30m"
        polling_interval: 5
        user_code:
          charset: "BCDFGHJKLMNPQRSTVWXZ"  # no ambiguous chars
          length: 8
```

---

## CIBA (RFC 9101)

Client-Initiated Backchannel Authentication. The user authenticates on a
separate device (e.g., biometric on phone) without a browser redirect.

### When to Use

- Banking apps with mobile-first authentication
- High-security apps requiring step-up auth
- Scenarios where browser redirects are undesirable

### Sequence Diagram

```
App (Client)         GGID Auth Server         User's Device
   │                       │                       │
   │ 1. Backchannel auth   │                       │
   │    request (login_hint)                       │
   ├──────────────────────►│                       │
   │                       │ 2. Send push          │
   │                       │    notification       │
   │                       ├──────────────────────►│
   │ 3. auth_req_id        │                       │
   │◄──────────────────────┤                       │
   │                       │                       │
   │                       │ 4. User approves      │
   │                       │    via biometric      │
   │                       │◄──────────────────────┤
   │                       │                       │
   │ 5. Poll token         │                       │
   │    (with auth_req_id) │                       │
   ├──────────────────────►│                       │
   │ 6. pending            │                       │
   │◄──────────────────────┤                       │
   │                       │                       │
   │ 7. Poll again         │                       │
   ├──────────────────────►│                       │
   │ 8. access_token       │                       │
   │◄──────────────────────┤                       │
```

### Step 1: Initiate Authentication

```bash
curl -X POST https://iam.example.com/oauth/bc-authorize \
  -d "client_id=ciba-client" \
  -d "client_secret=secret" \
  -d "login_hint=user@example.com" \
  -d "scope=openid profile" \
  -d "binding_message=Login to Banking App" \
  -d "user_code=1234"
```

```json
{
  "auth_req_id": "ciba-req-id-xxx",
  "expires_in": 600,
  "interval": 3
}
```

### Step 2: Poll for Token

```bash
curl -X POST https://iam.example.com/oauth/token \
  -d "grant_type=urn:openid:params:grant-type:ciba" \
  -d "auth_req_id=ciba-req-id-xxx" \
  -d "client_id=ciba-client"
```

| Response | Meaning |
|----------|---------|
| `authorization_pending` | User hasn't responded yet |
| `access_token` | User approved authentication |

### Configuration

```yaml
oauth:
  ciba:
    enabled: true
    default_expires_in: "10m"
    polling_interval: 3
    max_polling_interval: 10
    delivery_methods: ["poll", "ping"]
    binding_message_required: true
```

---

## PAR — Pushed Authorization Request (RFC 9126)

PAR pushes authorization request parameters directly to the authorization
server via a back-channel POST, returning a `request_uri` that replaces the
long front-channel URL. This prevents URL tampering and parameter injection.

### When to Use

- High-security applications
- When authorization requests contain sensitive parameters
- When request objects are too large for URL (e.g., Verifiable Credentials)

### Sequence Diagram

```
Client                    GGID Auth Server              Browser
  │                            │                            │
  │ 1. POST /oauth/par         │                            │
  │    (all auth params)       │                            │
  ├───────────────────────────►│                            │
  │                            │ 2. Validate request        │
  │ 3. request_uri             │                            │
  │◄───────────────────────────┤                            │
  │                            │                            │
  │ 4. Redirect browser        │                            │
  │    /authorize?request_uri= │                            │
  ├────────────────────────────────────────────────────────►│
  │                            │ 5. Server retrieves params │
  │                            │    from PAR cache           │
  │                            │ 6. Normal auth flow         │
```

### Step 1: Push Request

```bash
curl -X POST https://iam.example.com/oauth/par \
  -d "client_id=high-security-app" \
  -d "response_type=code" \
  -d "redirect_uri=https://app.example.com/callback" \
  -d "code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" \
  -d "code_challenge_method=S256" \
  -d "state=random-state" \
  -d "scope=openid profile"
```

```json
{
  "request_uri": "urn:ietf:params:oauth:request_uri:abc123xyz",
  "expires_in": 60
}
```

### Step 2: Redirect with request_uri

```
GET /oauth/authorize?client_id=high-security-app&request_uri=urn:ietf:params:oauth:request_uri:abc123xyz
```

The authorization server retrieves the original parameters from its cache.

### Configuration

```yaml
oauth:
  par:
    enabled: true
    require_par: false          # Set true to mandate PAR for all clients
    request_uri_lifetime: "60s"
```

---

## Token Exchange (RFC 8693)

Exchange one token for another with different scope or audience. Used for
delegation and tiered API access.

### When to Use

- A frontend calls API A, which needs to call API B on behalf of the user
- Downgrading token scope (least privilege per hop)
- Changing token audience for different downstream APIs

### Sequence Diagram

```
Frontend          API Gateway          Downstream API
   │                  │                      │
   │ 1. Has token     │                      │
   │    (aud=gateway) │                      │
   │                  │                      │
   │ 2. API call      │                      │
   ├─────────────────►│                      │
   │                  │ 3. Exchange token    │
   │                  │    (aud=downstream)  │
   │                  ├─────────────────────►│ (Auth Server)
   │                  │ 4. New token         │
   │                  │    (aud=downstream,  │
   │                  │     reduced scope)   │
   │                  │◄─────────────────────┤
   │                  │                      │
   │                  │ 5. Call downstream   │
   │                  │    with new token    │
   │                  ├─────────────────────►│
   │                  │ 6. Response          │
   │                  │◄─────────────────────┤
   │ 7. Response      │                      │
   │◄─────────────────┤                      │
```

### Request

```bash
curl -X POST https://iam.example.com/oauth/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "subject_token=original-access-token" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "audience=downstream-api-resource" \
  -d "scope=users:read" \
  -d "requested_token_type=urn:ietf:params:oauth:token-type:access_token"
```

### Response

```json
{
  "access_token": "eyJhbG...new-token...",
  "issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
  "token_type": "Bearer",
  "expires_in": 900,
  "scope": "users:read",
  "audience": "downstream-api-resource"
}
```

### Delegation Pattern

The exchanged token includes an `act` (actor) claim identifying the original
subject:

```json
{
  "sub": "user-uuid",
  "aud": "downstream-api",
  "scope": "users:read",
  "act": {
    "sub": "api-gateway-client-id"
  }
}
```

### Configuration

```yaml
oauth:
  token_exchange:
    enabled: true
    max_delegation_depth: 3
    allowed_audiences:
      - "internal-api"
      - "billing-service"
      - "notification-service"
```

---

## Refresh Token

Exchange a refresh token for a new access token without user re-authentication.

### Request

```bash
curl -X POST https://iam.example.com/oauth/token \
  -d "grant_type=refresh_token" \
  -d "refresh_token=rt-xxx" \
  -d "client_id=your-client-id" \
  -d "scope=openid profile"
```

### Response with Rotation

```json
{
  "access_token": "eyJhbG...new-access...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "rt-yyy-new-refresh",
  "scope": "openid profile"
}
```

> The old refresh token is invalidated. If reused, the entire token family is
> revoked (reuse detection).

### Rotation Flow

```
Token Family:
  AT-1 ← RT-A → (rotate) → AT-2 ← RT-B → (rotate) → AT-3 ← RT-C

If RT-A is reused after rotation:
  → Detect reuse
  → Revoke RT-A, RT-B, RT-C (entire family)
  → Force user re-authentication
```

### Configuration

```yaml
oauth:
  refresh_token:
    rotation: true
    reuse_detection: true
    lifetime: "24h"
    idle_timeout: "8h"           # Revoked if not used within 8h
    grace_period: "0s"           # Strict — no overlap allowed
```
