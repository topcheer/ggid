# OAuth 2.0 Device Authorization Grant (RFC 8628)

**Research & Implementation Design for GGID**

| | |
|---|---|
| **RFC** | [RFC 8628](https://www.rfc-editor.org/info/rfc8628/) — OAuth 2.0 Device Authorization Grant |
| **Published** | August 2019 |
| **Category** | Standards Track |
| **Status** | GGID: Partially implemented (in-memory), production hardening needed |
| **Priority** | P1 — Critical for CLI/SDK adoption |

---

## Table of Contents

1. [Overview](#1-overview)
2. [Device Flow Protocol](#2-device-flow-protocol)
3. [User Code Design](#3-user-code-design)
4. [Polling Strategy](#4-polling-strategy)
5. [GGID Implementation Design](#5-ggid-implementation-design)
6. [Console Integration](#6-console-integration)
7. [CLI Integration Example](#7-cli-integration-example)
8. [Comparison with Other Implementations](#8-comparison-with-other-implementations)
9. [GGID Roadmap](#9-ggid-roadmap)

---

## 1. Overview

### 1.1 What is the Device Authorization Grant?

RFC 8628 defines the **OAuth 2.0 Device Authorization Grant** (commonly called "device flow"), an extension to OAuth 2.0 designed for internet-connected devices that either:

- **Lack a browser** to perform a user-agent-based authorization, or
- Are **input constrained** — requiring the user to type text credentials is impractical.

Instead of authenticating on the device itself, the user authenticates on a **separate device** (typically a smartphone or laptop) that does have a browser and full input capabilities. The constrained device displays a short code, the user enters it elsewhere, and after approval the device receives its tokens.

### 1.2 Typical Use Cases

| Use Case | Example |
|---|---|
| **CLI tools** | `gh auth login` (GitHub CLI), `aws sso login` (AWS CLI), `docker login` (Docker CLI) |
| **Smart TVs / Media** | Google TV, Apple TV, Roku, Chromecast |
| **Game consoles** | Xbox, PlayStation network linking |
| **IoT devices** | Smart displays, digital picture frames, printers |
| **Embedded systems** | Raspberry Pi, industrial controllers without keyboards |

### 1.3 Why Not Other Grant Types?

| Grant Type | Problem for Input-Constrained Devices |
|---|---|
| Authorization Code | Requires a browser redirect on the device |
| Resource Owner Password | Requires typing credentials on the device (terrible UX) |
| Client Credentials | No user context — machine-to-machine only |
| **Device Authorization** | No browser needed; user authenticates on a secondary device |

### 1.4 Key Properties

- **No client secret required** — public clients (CLI tools) can use it safely
- **Out-of-band authentication** — user interacts on a different device
- **Polling-based** — device polls the token endpoint at controlled intervals
- **Short-lived codes** — device_code and user_code expire (typically 15 min)
- **Tenant-aware** — codes are scoped to a specific tenant in multi-tenant systems

---

## 2. Device Flow Protocol

### 2.1 Protocol Sequence Diagram

```
 +----------+                                +----------------+
 |          | >--(A) device authorization -->|                |
 |  Device  |                                |  Authorization |
 |  (CLI,   | <-(B) device_code, user_code --|    Server      |
 |  TV...)  |                                |  (GGID OAuth)  |
 |          |                                |                |
 |          |  +--------+   +----------+     |                |
 |          |  |  User  |   | Browser  |     |                |
 |          |  |  Phone |   | (on user |     |                |
 |          |  |  /Laptop|  |  device) |     |                |
 |          |  +---+----+   +-----+----+     |                |
 |          |       |              |         |                |
 |          | >-(C) display code & URI ----->|                |
 |          |       |              |         |                |
 |          |       | <-(D) visit URI, enter code ------------|
 |          |       |              |         |                |
 |          |       | >-(E) authenticate & approve ---------->|
 |          |       |              |         |                |
 |          | <-(F) poll token -----+--------|                |
 |          | >-(G) access_token ---+------->|                |
 +----------+                                +----------------+
```

### 2.2 Step 1: Device Authorization Request

The device requests a new authorization session by POSTing to the device authorization endpoint.

**Request:**

```http
POST /oauth/device_authorization HTTP/1.1
Host: auth.example.com
Content-Type: application/x-www-form-urlencoded
X-Tenant-ID: 00000000-0000-0000-0000-000000000001

client_id=my-cli-app&scope=openid%20profile%20email
```

| Parameter | Required | Description |
|---|---|---|
| `client_id` | Yes | The client identifier registered with the authorization server |
| `scope` | No | Space-delimited list of requested scopes |
| `client_secret` | No* | Required only for confidential clients (*not recommended for device flow*) |

**Response (HTTP 200):**

```json
{
  "device_code": "GmRh...8zCnKqL2WpXvNbM4QtRsUaHfDgJkLzXcVbNmQwEr",
  "user_code": "WDJB-MJHQ",
  "verification_uri": "https://auth.example.com/device",
  "verification_uri_complete": "https://auth.example.com/device?user_code=WDJB-MJHQ",
  "expires_in": 900,
  "interval": 5
}
```

| Field | Description |
|---|---|
| `device_code` | High-entropy device verification code (not shown to user) |
| `user_code` | Short, human-readable code the user enters on the verification page |
| `verification_uri` | The URL the user should visit to enter the user_code |
| `verification_uri_complete` | URL with user_code embedded (for QR codes / deep links) |
| `expires_in` | Lifetime in seconds (recommended: 900 = 15 min) |
| `interval` | Minimum seconds between polling attempts (recommended: 5) |

> **Note:** `verification_uri_complete` is optional but highly recommended for QR code workflows.

### 2.3 Step 2: User Interaction

The device displays the user_code and verification_uri to the user:

```
    ╔══════════════════════════════════════╗
    ║                                      ║
    ║    Go to: https://auth.example.com   ║
    ║                                      ║
    ║    Enter code: WDJB-MJHQ             ║
    ║                                      ║
    ║    (or scan QR code)                 ║
    ║                                      ║
    ╚══════════════════════════════════════╝
```

The user then:
1. Opens `verification_uri` in a browser on a **different device** (phone/laptop)
2. Enters the `user_code` (or scans the QR code from `verification_uri_complete`)
3. Authenticates with their credentials (password, MFA, WebAuthn, etc.)
4. Reviews the requested scopes and **approves** or **denies** the request

If the QR code (containing `verification_uri_complete`) is scanned, the user_code is pre-filled and the user can skip step 2.

### 2.4 Step 3: Device Polls for Token

While the user is interacting with the verification page, the device **polls** the token endpoint at the specified `interval`:

**Request:**

```http
POST /oauth/token HTTP/1.1
Host: auth.example.com
Content-Type: application/x-www-form-urlencoded
X-Tenant-ID: 00000000-0000-0000-0000-000000000001

grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code
&device_code=GmRh...8zCnKqL2WpXvNbM4QtRsUaHfDgJkLzXcVbNmQwEr
&client_id=my-cli-app
```

The `grant_type` for device flow uses the URN: `urn:ietf:params:oauth:grant-type:device_code`

**Possible Responses:**

| Status | Error Code | Description |
|---|---|---|
| 200 OK | — | User approved; returns standard token response |
| 400 | `authorization_pending` | User hasn't completed interaction yet |
| 400 | `slow_down` | Polling too fast; increase interval by 5 seconds |
| 400 | `access_denied` | User denied the request |
| 400 | `expired_token` | Device code has expired |

**Pending Response (HTTP 400):**
```json
{
  "error": "authorization_pending"
}
```

**Slow Down Response (HTTP 400):**
```json
{
  "error": "slow_down"
}
```

**Success Response (HTTP 200):**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "rTjK...mNpQ",
  "scope": "openid profile email",
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

### 2.5 Step 4: Token Refresh

Once the device has the initial token set, it can use `refresh_token` normally:

```http
POST /oauth/token HTTP/1.1
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token=rTjK...mNpQ
&client_id=my-cli-app
```

This follows the standard OAuth 2.0 refresh token flow (RFC 6749 Section 6).

---

## 3. User Code Design

### 3.1 Character Set

RFC 8628 Section 6.1 recommends using a **reduced character set** to avoid ambiguous characters. The commonly used set is **base20** — 20 consonants selected to eliminate visual ambiguity:

```
BCDFGHJKLMNPQRSTVWXZ
```

**Characters excluded:**
- All vowels (A, E, I, O, U) — prevents accidental profanity
- S, Z — visually similar to S/Z in some fonts (implementation-dependent)
- Numbers 0, 1 — confused with O, I
- L — confused with I or 1

> **Note:** GGID currently uses `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` (base32 without 0/1/O/I). This is also acceptable but has a larger character set. The RFC-compliant approach is to use consonants only.

### 3.2 Format and Length

| Property | Value |
|---|---|
| Length | 8 characters (typical) |
| Display format | `XXXX-XXXX` (hyphen-separated for readability) |
| Case | Case-insensitive (display uppercase) |
| Entropy (base20, 8 chars) | 20^8 = 25,600,000,000 (~31 bits) |
| Entropy (base32, 8 chars) | 32^8 = 1,099,511,627,776 (~40 bits) |

### 3.3 Entropy vs. Brute Force

With ~31 bits of entropy and rate limiting on the verification endpoint:

| Rate Limit | Time to Exhaust |
|---|---|
| 10 attempts/min/IP | ~487,000 years per IP |
| 100 attempts/min globally | ~487 years globally |

With proper rate limiting, brute-forcing the user_code is computationally infeasible.

### 3.4 Rate Limiting on Verification Endpoint

The verification endpoint (`POST /device/approve` or the page that accepts user_code) **must** enforce:

- **Per-IP rate limiting** — e.g., 10 attempts per minute
- **Global rate limiting** — e.g., 1000 attempts per minute across all IPs
- **Exponential backoff** — after N failed attempts, increase delay
- **CAPTCHA** — after repeated failures from the same IP

---

## 4. Polling Strategy

### 4.1 Client-Side Polling Algorithm

```
func deviceFlowPoll(deviceCode, clientID, interval, expiresIn):
    deadline = now() + expiresIn
    currentInterval = interval

    while now() < deadline:
        sleep(currentInterval)

        response = POST /oauth/token with:
            grant_type = urn:ietf:params:oauth:grant-type:device_code
            device_code = deviceCode
            client_id = clientID

        switch response.error:
            case nil:
                return response.tokens  // Success!

            case "authorization_pending":
                continue  // Poll again after currentInterval

            case "slow_down":
                currentInterval += 5  // Increase interval by 5s
                continue

            case "access_denied":
                return error("user denied authorization")

            case "expired_token":
                return error("device code expired")

    return error("timed out waiting for authorization")
```

### 4.2 Key Polling Rules

| Rule | RFC Reference |
|---|---|
| Initial interval from server `interval` field | Section 3.4 |
| On `slow_down`: increase interval by 5 seconds (permanently) | Section 3.5 |
| Never poll faster than the current interval | Section 3.4 |
| Stop polling after `expires_in` has elapsed | Section 3.4 |
| Honor `expired_token` — do not retry | Section 3.5 |

### 4.3 Recommended Defaults

| Parameter | Recommended Value |
|---|---|
| `interval` | 5 seconds |
| `expires_in` | 900 seconds (15 minutes) |
| Maximum poll attempts | `expires_in / interval` = 180 |
| Additional backoff on slow_down | +5 seconds per occurrence |

---

## 5. GGID Implementation Design

### 5.1 Current State

GGID already has a **working device flow implementation** with the following components:

| Component | Location | Status |
|---|---|---|
| Device authorization handler | `services/oauth/internal/server/server.go` (line ~833) | Working (in-memory) |
| Device token poll handler | `services/oauth/internal/server/server.go` (line ~351) | Working |
| Device approve handler | `services/oauth/internal/server/server.go` (line ~873) | Working |
| Service layer | `services/oauth/internal/service/oauth_service.go` (line ~950) | Working |
| User code generator | `services/oauth/internal/service/oauth_service.go` (line ~1142) | Working |
| Device code generator | `services/oauth/internal/service/oauth_service.go` (line ~1132) | Working |
| Storage | In-memory maps with mutex | **Needs Redis for production** |
| Console verification page | Not yet implemented | **TODO** |
| Rate limiting on approve endpoint | Not implemented | **TODO** |
| `verification_uri_complete` | Not in response | **TODO** |

### 5.2 New / Enhanced Endpoints

#### Existing Endpoints (already wired)

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/oauth/device_authorization` | POST | Issue device_code + user_code |
| `/oauth/token` (grant_type=device_code) | POST | Device polls for token |
| `/api/v1/oauth/device/approve` | POST | User submits code + approves |

#### Proposed Additions

| Endpoint | Method | Description |
|---|---|---|
| `/oauth/device_authorization` | POST | RFC-compliant path (without `/api/v1` prefix) |
| `/device` (Console) | GET | User-facing verification page |
| `/device/verify` (Console) | POST | User submits code on Console |
| `/device/deny` | POST | User explicitly denies the request |

### 5.3 Data Model

#### Current Implementation (In-Memory)

```go
// services/oauth/internal/service/oauth_service.go (line ~970)

type DeviceCodeInfo struct {
    DeviceCode string
    UserCode   string
    ClientID   string
    TenantID   uuid.UUID
    UserID     *uuid.UUID // set when user authorizes
    Scope      []string
    Status     string // "pending", "approved", "denied", "expired"
    CreatedAt  time.Time
    ExpiresAt  time.Time
}

// Storage: in-memory maps (not production-safe)
var (
    deviceCodeMu    sync.RWMutex
    deviceCodeStore = make(map[string]*DeviceCodeInfo) // device_code -> info
    userCodeIndex   = make(map[string]string)          // user_code -> device_code
)
```

#### Proposed PostgreSQL Schema

```sql
CREATE TABLE oauth_device_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    device_code     VARCHAR(128) NOT NULL UNIQUE,  -- hashed
    user_code       VARCHAR(20) NOT NULL UNIQUE,
    client_id       VARCHAR(128) NOT NULL,
    scope           TEXT[],
    status          VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending|approved|denied|expired
    user_id         UUID,                             -- NULL until approved
    interval_seconds INT NOT NULL DEFAULT 5,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    approved_at     TIMESTAMPTZ,
    CONSTRAINT chk_status CHECK (status IN ('pending', 'approved', 'denied', 'expired'))
);

CREATE INDEX idx_device_codes_user_code ON oauth_device_codes (user_code) WHERE status = 'pending';
CREATE INDEX idx_device_codes_expires ON oauth_device_codes (expires_at);
```

#### Proposed Redis Schema (Recommended)

Redis is the **preferred storage** for device codes because:

1. **Natural TTL** — keys auto-expire, no cleanup needed
2. **Low latency** — polling endpoint gets sub-ms reads
3. **Simple** — no schema migrations
4. **Scalable** — works across multiple OAuth service instances

```
# Key structure
device_code:{device_code_hash} → {
    "user_code": "WDJB-MJHQ",
    "client_id": "my-cli-app",
    "tenant_id": "00000000-...",
    "scope": ["openid", "profile"],
    "status": "pending",
    "user_id": null,
    "created_at": "2024-01-15T10:30:00Z",
    "expires_at": "2024-01-15T10:45:00Z"
}
# TTL: 900 seconds (15 min)

user_code:{USER_CODE} → {device_code_hash}
# TTL: 900 seconds (15 min)
# Used for O(1) lookup when user enters code on verification page
```

### 5.4 Implementation Code

#### 5.4.1 Device Authorization Handler

> Already implemented at `services/oauth/internal/service/oauth_service.go:989`

```go
// CreateDeviceAuthorization generates device_code + user_code and stores them.
func (s *OAuthService) CreateDeviceAuthorization(req *DeviceAuthorizationRequest) (*DeviceAuthorizationResponse, error) {
    deviceCode := generateDeviceCode(40)
    userCode := generateUserCode()

    info := &DeviceCodeInfo{
        DeviceCode: deviceCode,
        UserCode:   userCode,
        ClientID:   req.ClientID,
        TenantID:   req.TenantID,
        Scope:      req.Scope,
        Status:     "pending",
        CreatedAt:  time.Now(),
        ExpiresAt:  time.Now().Add(15 * time.Minute),
    }

    deviceCodeMu.Lock()
    deviceCodeStore[deviceCode] = info
    userCodeIndex[userCode] = deviceCode
    deviceCodeMu.Unlock()

    verificationURI := req.Issuer + "/device"

    return &DeviceAuthorizationResponse{
        DeviceCode:      deviceCode,
        UserCode:        userCode,
        VerificationURI: verificationURI,
        ExpiresIn:       900,
        Interval:        5,
    }, nil
}
```

**Enhancement: Add `verification_uri_complete`**

```go
type DeviceAuthorizationResponse struct {
    DeviceCode              string `json:"device_code"`
    UserCode                string `json:"user_code"`
    VerificationURI         string `json:"verification_uri"`
    VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
    ExpiresIn               int    `json:"expires_in"`
    Interval                int    `json:"interval"`
}

// In CreateDeviceAuthorization:
return &DeviceAuthorizationResponse{
    DeviceCode:              deviceCode,
    UserCode:                userCode,
    VerificationURI:         verificationURI,
    VerificationURIComplete: verificationURI + "?user_code=" + userCode,
    ExpiresIn:               900,
    Interval:                5,
}, nil
```

#### 5.4.2 User Code Generator

> Already implemented at `services/oauth/internal/service/oauth_service.go:1142`

```go
// generateUserCode creates an 8-character user code in XXXX-XXXX format.
func generateUserCode() string {
    const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no confusing chars
    part1 := make([]byte, 4)
    part2 := make([]byte, 4)
    for i := range part1 {
        part1[i] = charset[cryptoRandInt(len(charset))]
    }
    for i := range part2 {
        part2[i] = charset[cryptoRandInt(len(charset))]
    }
    return string(part1) + "-" + string(part2)
}
```

**RFC 8628-compliant version (base20 consonants only):**

```go
// generateUserCodeRFC8628 generates a user code using only consonants.
// Character set: BCDFGHJKLMNPQRSTVWXZ (20 chars, no vowels, no ambiguous chars)
func generateUserCodeRFC8628() string {
    const charset = "BCDFGHJKLMNPQRSTVWXZ"
    b := make([]byte, 8)
    for i := range b {
        b[i] = charset[cryptoRandInt(len(charset))]
    }
    return string(b[:4]) + "-" + string(b[4:])
}
```

#### 5.4.3 Device Code Generator

> Already implemented at `services/oauth/internal/service/oauth_service.go:1132`

```go
// generateDeviceCode creates a random alphanumeric device code (40 chars).
func generateDeviceCode(length int) string {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, length)
    for i := range b {
        b[i] = charset[cryptoRandInt(len(charset))]
    }
    return string(b)
}
```

**Enhancement: Use crypto/rand for 256-bit entropy:**

```go
// generateDeviceCode256 generates a 256-bit device code (43 chars base64url).
func generateDeviceCode256() string {
    b := make([]byte, 32) // 256 bits
    if _, err := crypto_rand.Read(b); err != nil {
        panic(err) // crypto/rand should never fail
    }
    return base64.RawURLEncoding.EncodeToString(b) // 43 chars, no padding
}
```

#### 5.4.4 Device Poll Handler (Token Endpoint)

> Already implemented at `services/oauth/internal/server/server.go:351` and
> `services/oauth/internal/service/oauth_service.go:1024`

The token endpoint handles `grant_type=urn:ietf:params:oauth:grant-type:device_code`:

```go
case "urn:ietf:params:oauth:grant-type:device_code":
    resp, tokenErr = oauthSvc.PollDeviceToken(ctx, r.FormValue("device_code"), clientID)
    if tokenErr != nil {
        errMsg := tokenErr.Error()
        switch errMsg {
        case "authorization_pending":
            writeJSON(w, http.StatusBadRequest,
                map[string]string{"error": "authorization_pending"})
        case "slow_down":
            writeJSON(w, http.StatusBadRequest,
                map[string]string{"error": "slow_down"})
        case "expired_token":
            writeJSON(w, http.StatusBadRequest,
                map[string]string{"error": "expired_token"})
        case "access_denied":
            writeJSON(w, http.StatusBadRequest,
                map[string]string{"error": "access_denied"})
        default:
            writeJSON(w, http.StatusBadRequest,
                map[string]string{"error": "invalid_grant",
                    "error_description": errMsg})
        }
        return
    }
```

#### 5.4.5 Device Approve Handler

> Already implemented at `services/oauth/internal/server/server.go:873` and
> `services/oauth/internal/service/oauth_service.go:1080`

```go
// ApproveDeviceCode is called when the user enters their user_code and approves.
func (s *OAuthService) ApproveDeviceCode(userCode string, userID uuid.UUID) error {
    deviceCodeMu.Lock()
    defer deviceCodeMu.Unlock()

    deviceCode, ok := userCodeIndex[userCode]
    if !ok {
        return fmt.Errorf("invalid user_code")
    }

    info, ok := deviceCodeStore[deviceCode]
    if !ok {
        return fmt.Errorf("device code not found")
    }

    if time.Now().After(info.ExpiresAt) {
        delete(deviceCodeStore, deviceCode)
        delete(userCodeIndex, userCode)
        return fmt.Errorf("expired user_code")
    }

    info.Status = "approved"
    info.UserID = &userID
    return nil
}
```

#### 5.4.6 Device Deny Handler (Proposed)

```go
// DenyDeviceCode is called when the user explicitly denies the authorization.
func (s *OAuthService) DenyDeviceCode(userCode string) error {
    deviceCodeMu.Lock()
    defer deviceCodeMu.Unlock()

    deviceCode, ok := userCodeIndex[userCode]
    if !ok {
        return fmt.Errorf("invalid user_code")
    }

    info, ok := deviceCodeStore[deviceCode]
    if !ok {
        return fmt.Errorf("device code not found")
    }

    info.Status = "denied"
    return nil
}
```

### 5.5 Security Considerations

#### 5.5.1 Device Code Entropy

The `device_code` must be high-entropy to prevent guessing:

| Implementation | Entropy | Recommendation |
|---|---|---|
| 40-char alphanumeric (current) | ~239 bits | More than sufficient |
| 32 random bytes base64url (proposed) | 256 bits | RFC recommended minimum |
| 16-char alphanumeric | ~95 bits | Minimum acceptable |

**Never** store the raw device_code in the database — always store a **hash** (SHA-256 or Argon2id).

#### 5.5.2 Rate Limiting

| Endpoint | Limit | Purpose |
|---|---|---|
| `/device_authorization` | 10/min/client_id | Prevent code spam |
| `/oauth/token` (device_code) | 12/min/device_code | Enforce interval |
| `/device/approve` | 10/min/IP | Prevent user_code brute force |
| `/device/approve` | 100/min/global | Distributed brute force protection |

GGID has a sliding window rate limiter at `services/gateway/internal/middleware/sliding_ratelimit.go` that can be applied to these endpoints.

#### 5.5.3 Client ID Binding

The `device_code` must be bound to the `client_id` that requested it. The poll handler must verify that the polling `client_id` matches:

```go
// In PollDeviceToken — verify client_id matches
if info.ClientID != clientID {
    return nil, fmt.Errorf("invalid_device_code") // Don't leak info
}
```

#### 5.5.4 Tenant Isolation

Device codes are scoped to a tenant. The `X-Tenant-ID` header is required on all device flow endpoints, and the stored `DeviceCodeInfo.TenantID` must match:

```go
// When approving, verify tenant context matches
if info.TenantID != requestTenantID {
    return fmt.Errorf("invalid user_code") // Cross-tenant attempt
}
```

#### 5.5.5 CSRF Protection on Verification Page

The verification page (Console `/device`) must implement CSRF protection:

- Generate a CSRF token tied to the user session
- Include it in the approval form as a hidden field
- Validate on POST before processing approval

#### 5.5.6 PKCE-like Binding (Future Enhancement)

For additional security, the device authorization request could include a `code_challenge` (PKCE). The poll endpoint would then verify `code_verifier` before issuing tokens:

```
Device → POST /device_authorization (with code_challenge)
User   → Approves on verification page
Device → POST /token (with device_code + code_verifier)
```

This prevents token interception even if the device_code is leaked.

#### 5.5.7 Cleanup of Expired Codes

Expired device codes must be cleaned up to prevent memory leaks and storage growth:

| Storage | Cleanup Strategy |
|---|---|
| In-memory (current) | Lazy deletion on access (already implemented) |
| Redis | Automatic TTL expiry (no code needed) |
| PostgreSQL | Scheduled cleanup job (every 5 min): `DELETE FROM oauth_device_codes WHERE expires_at < NOW()` |

---

## 6. Console Integration

### 6.1 New Page: `/device`

Add a new page to the GGID Admin Console at `console/src/app/device/page.tsx`:

**User Flow:**
1. User navigates to `https://console.example.com/device` (or scans QR code)
2. If URL contains `?user_code=XXXX-XXXX`, pre-fill the code field
3. User enters/verifies the code
4. Console calls `POST /api/v1/oauth/device/approve` with the user's session token
5. Show success or denial confirmation

### 6.2 Console Component Structure

```
console/src/app/device/
├── page.tsx              # Main device verification page
├── components/
│   ├── CodeInput.tsx     # User code input field (auto-uppercase, auto-hyphen)
│   ├── ApprovalCard.tsx  # Shows client info + requested scopes
│   └── SuccessScreen.tsx # Confirmation after approval
```

### 6.3 API Calls from Console

```typescript
// 1. Look up device code info (to show what client is requesting)
const res = await fetch(`/api/v1/oauth/device/info?user_code=${code}`, {
  headers: { 'X-Tenant-ID': tenantId }
});
const info = await res.json();
// → { client_id, client_name, scope, created_at }

// 2. Approve the device
const res = await fetch('/api/v1/oauth/device/approve', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/x-www-form-urlencoded',
    'X-Tenant-ID': tenantId,
    'X-User-ID': userId,
  },
  body: `user_code=${code}`
});
// → { status: "approved" }
```

### 6.4 UX Enhancements

| Feature | Description |
|---|---|
| Auto-uppercase input | Convert lowercase to uppercase as user types |
| Auto-hyphen | Insert hyphen after 4th character |
| Code lookup | Show client name and requested scopes before approval |
| QR code display | If device shows QR, Console can also show a QR for the user_code |
| Session check | Require authenticated session before showing approval UI |

---

## 7. CLI Integration Example

### 7.1 Go CLI Tool Using GGID Device Flow

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ggidAuthURL  = "https://auth.example.com"
	tenantID     = "00000000-0000-0000-0000-000000000001"
	clientID     = "my-cli-app"
)

// DeviceAuthResponse represents the device authorization response.
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// TokenResponse represents the OAuth token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token"`
}

func main() {
	// Step 1: Request device authorization
	fmt.Println("Requesting device authorization...")
	deviceResp, err := requestDeviceAuth()
	if err != nil {
		fmt.Fatalf("Failed: %v", err)
	}

	// Step 2: Display code to user
	fmt.Println()
	fmt.Println("  ┌──────────────────────────────────────────┐")
	fmt.Println("  │                                          │")
	fmt.Printf("  │  Go to: %-33s│\n", deviceResp.VerificationURI)
	fmt.Println("  │                                          │")
	fmt.Printf("  │  Enter code: %-28s│\n", deviceResp.UserCode)
	fmt.Println("  │                                          │")
	fmt.Println("  │  Waiting for authorization...            │")
	fmt.Println("  │                                          │")
	fmt.Println("  └──────────────────────────────────────────┘")
	fmt.Println()

	// Step 3: Poll for token
	token, err := pollForToken(
		deviceResp.DeviceCode,
		deviceResp.Interval,
		deviceResp.ExpiresIn,
	)
	if err != nil {
		fmt.Fatalf("Authorization failed: %v", err)
	}

	// Step 4: Use the token
	fmt.Println("  ✓ Authorization successful!")
	fmt.Printf("  Access token: %s...%s\n",
		token.AccessToken[:20], token.AccessToken[len(token.AccessToken)-8:])
	fmt.Println()

	// Make an authenticated API call
	resp, err := apiCall(token.AccessToken)
	if err != nil {
		fmt.Fatalf("API call failed: %v", err)
	}
	fmt.Printf("  API response: %s\n", resp)
}

func requestDeviceAuth() (*DeviceAuthResponse, error) {
	data := url.Values{
		"client_id": {clientID},
		"scope":     {"openid profile email"},
	}

	req, _ := http.NewRequest("POST",
		ggidAuthURL+"/api/v1/oauth/device_authorization",
		strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var result DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func pollForToken(deviceCode string, interval, expiresIn int) (*TokenResponse, error) {
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	currentInterval := time.Duration(interval) * time.Second

	for time.Now().Before(deadline) {
		time.Sleep(currentInterval)

		data := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode},
			"client_id":   {clientID},
		}

		req, _ := http.NewRequest("POST",
			ggidAuthURL+"/oauth/token",
			strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Tenant-ID", tenantID)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			var token TokenResponse
			json.NewDecoder(resp.Body).Decode(&token)
			return &token, nil
		}

		// Parse error response
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)

		switch errResp.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			currentInterval += 5 * time.Second
			continue
		case "access_denied":
			return nil, fmt.Errorf("user denied authorization")
		case "expired_token":
			return nil, fmt.Errorf("device code expired")
		}
	}

	return nil, fmt.Errorf("timed out waiting for authorization")
}

func apiCall(accessToken string) (string, error) {
	req, _ := http.NewRequest("GET",
		"https://api.example.com/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}
```

**Sample CLI Output:**

```
$ ggid login

  ┌──────────────────────────────────────────┐
  │                                          │
  │  Go to: https://auth.example.com/device  │
  │                                          │
  │  Enter code: WDJB-MJHQ                   │
  │                                          │
  │  Waiting for authorization...            │
  │                                          │
  └──────────────────────────────────────────┘

  ✓ Authorization successful!
  Access token: eyJhbGciOiJSUzI1NiIs...XcVbNmQw

  API response: {"id":"...","email":"user@example.com","name":"Test User"}
```

---

## 8. Comparison with Other Implementations

### 8.1 Feature Matrix

| Feature | GGID (current) | GitHub | Google | Keycloak | Auth0 |
|---|---|---|---|---|---|
| RFC 8628 compliant | Partial | Yes | Yes | Yes | Yes |
| Device authorization endpoint | Yes | `/login/device/code` | `/device/code` | `/realms/{r}/protocol/openid-connect/auth/devices` | `/oauth/device/authorization` |
| `verification_uri_complete` | No | Yes | Yes | Yes | Yes |
| QR code support | No | Yes | Yes | Yes | No |
| User code charset | base32 | base20+ | base20 | base32 | base20 |
| User code format | XXXX-XXXX | XXXX-XXXX | XXXX-XXXX | XXXX-XXXX | XXXX-XXXX |
| Polling interval | 5s | 5s | 5s | 5s | 5s |
| Expiry | 15 min | 15 min | 15 min | Configurable | 15 min |
| Storage | In-memory | Redis | Proprietary | DB/Infinispan | Redis |
| Rate limiting | No | Yes | Yes | Yes | Yes |
| Multi-tenant | Yes | N/A | N/A | Per realm | Per domain |
| Deny flow | No | Yes | Yes | Yes | Yes |
| ID Token (OIDC) | No | N/A | Yes | Yes | Yes |
| PKCE on device flow | No | No | No | Optional | No |

### 8.2 GitHub CLI Device Flow

GitHub's implementation is the gold standard for CLI device flow:

```
$ gh auth login
# Choose: GitHub.com → HTTPS → Login with a web browser

! First copy your one-time code: WDJB-MJHQ
Press Enter to open github.com in your browser...
✓ Authentication complete.
```

Key features:
- Uses device flow with automatic browser opening via `xdg-open`/`open`
- Displays code in terminal with visual emphasis
- `verification_uri_complete` for seamless browser handoff
- Rate-limited verification page
- Supports both github.com and GitHub Enterprise

### 8.3 Google Device Flow

Google's device flow (`oauth2/v3/devicecode`) is used by Google TV, Chromecast, and the Google Cloud CLI:

- Endpoint: `https://oauth2.googleapis.com/device/code`
- Verification: `https://www.google.com/device`
- Supports `verification_uri_complete` with embedded user_code
- Displays a TV-friendly code input page at `google.com/device`
- Rate limited with CAPTCHA on suspicious activity

### 8.4 Keycloak Device Flow

Keycloak provides full RFC 8628 support since version 12+:

- Device endpoint: `/realms/{realm}/protocol/openid-connect/auth/devices`
- Admin-configurable polling interval and expiry
- Per-realm isolation (analogous to GGID's per-tenant)
- Supports OIDC (issues ID Token)
- Verification page served by Keycloak or custom

### 8.5 Auth0 Device Flow

Auth0 supports device flow as a first-class grant type:

- Endpoint: `https://{tenant}.auth0.com/oauth/device/authorization`
- Automatic rate limiting on all endpoints
- Custom domain support for verification page
- Breached password detection on verification
- MFA integration during approval

---

## 9. GGID Roadmap

### 9.1 Implementation Status

| Component | Status | Priority |
|---|---|---|
| Device authorization endpoint | **Done** — in-memory | — |
| Token polling endpoint | **Done** | — |
| Approve endpoint | **Done** — in-memory | — |
| `verification_uri_complete` | **TODO** | P1 |
| Redis storage (replace in-memory) | **TODO** | P1 |
| Console verification page (`/device`) | **TODO** | P1 |
| Rate limiting on approve endpoint | **TODO** | P1 |
| Deny flow | **TODO** | P2 |
| ID Token issuance on device flow | **TODO** | P2 |
| Device code lookup API (for Console) | **TODO** | P2 |
| QR code generation on device display | **TODO** | P3 |
| PKCE binding for device flow | **TODO** | P3 |
| PostgreSQL persistence | **TODO** | P3 |
| Unit + integration tests | **TODO** | P1 |
| SDK support (Go SDK device flow) | **TODO** | P2 |

### 9.2 Effort Estimates

| Task | Effort | Dependencies |
|---|---|---|
| Add `verification_uri_complete` | 0.5 day | None |
| Redis storage migration | 1-2 days | Redis infra (already available) |
| Console `/device` page | 2-3 days | Device code lookup API |
| Rate limiting | 0.5 day | Existing rate limiter |
| Deny flow | 0.5 day | None |
| ID Token in device response | 0.5 day | Existing IDTokenIssuer |
| Full test coverage | 1-2 days | Redis storage |
| Go SDK device flow helper | 1 day | Stable API |
| **Total** | **~7-10 days** | — |

### 9.3 Priority Assessment

**Priority: P1** — Device flow is critical for CLI and SDK adoption.

Without device flow, CLI tools like `ggid login` must either:
- Embed a browser (heavyweight, not always available on servers)
- Use password grant (deprecated, poor UX)
- Use a manual token paste (bad UX)

Device flow is the standard approach used by every major platform (GitHub, Google, AWS, Docker) for CLI authentication. GGID already has the core implementation; the remaining work is production hardening.

### 9.4 Dependencies

| Dependency | Status |
|---|---|
| Redis infrastructure | Already deployed (Docker Compose includes Redis) |
| OAuth client management | Already implemented |
| JWT signing (RS256) | Already implemented (shared with Auth Service) |
| Tenant context | Already implemented (`pkg/tenant`) |
| Rate limiter middleware | Already implemented (`sliding_ratelimit.go`) |
| Admin Console framework | Already implemented (Next.js 15) |

**No blocking dependencies.** This is a standalone feature.

### 9.5 OIDC Discovery Integration

After full implementation, update the OIDC discovery document to advertise device flow support:

```go
// In OIDCDiscoveryConfig (services/oauth/internal/domain/models.go:135):

type OIDCDiscoveryConfig struct {
    // ... existing fields ...
    DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
}

// In GetDiscoveryConfig:
config.DeviceAuthorizationEndpoint = cfg.Issuer + "/oauth/device_authorization"

// Add to GrantTypesSupported:
grantTypesSupported: []string{
    "authorization_code",
    "refresh_token",
    "client_credentials",
    "urn:ietf:params:oauth:grant-type:device_code",  // ← ADD THIS
}
```

---

## References

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://www.rfc-editor.org/info/rfc8628/)
- [RFC 6749: OAuth 2.0 Framework](https://datatracker.ietf.org/doc/html/rfc6749) — Section 6 (Refresh Token)
- [RFC 7591: OAuth 2.0 Dynamic Client Registration](https://datatracker.ietf.org/doc/html/rfc7591)
- [RFC 7636: PKCE](https://datatracker.ietf.org/doc/html/rfc7636)
- [GitHub CLI: Device Flow](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app)
- [Google Device Flow](https://developers.google.com/identity/protocols/oauth2/limited-input-device)
- [Auth0 Device Authorization Flow](https://auth0.com/docs/get-started/authentication-and-authorization-flow/device-authorization-flow)
- [Keycloak Device Flow](https://www.keycloak.org/docs/latest/securing_apps/#device-authorization-grant)
- [Microsoft Entra ID Device Code Flow](https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-device-code)

---

*Document last updated: 2025-01-20*
*GGID commit reference: current main*
