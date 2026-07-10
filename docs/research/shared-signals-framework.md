# OpenID Shared Signals Framework (SSF) — SET Transport Protocols & GGID Integration

**Document type:** Technical Research & Architecture Design
**Status:** Draft for Architecture Review
**Date:** 2025-01-20
**Author:** Research Team
**Related specs:**
- [OpenID Shared Signals Framework 1.0](https://openid.net/specs/openid-sharedsignals-framework-1_0.html)
- [OpenID CAEP Specification 1.0](https://openid.net/specs/openid-caep-spec-1_0.html)
- [RFC 8417 — Security Event Token (SET)](https://www.rfc-editor.org/rfc/rfc8417)
- [RFC 8935 — Push-Based SET Delivery Using HTTP](https://www.rfc-editor.org/rfc/rfc8935)
- [RFC 8936 — Poll-Based SET Delivery Using HTTP](https://www.rfc-editor.org/rfc/rfc8936)

**Related GGID documents:**
- [CAEP Analysis](./caep-analysis.md) — CAEP event types and GGID integration design

---

## Table of Contents

1. [Overview](#1-overview)
2. [Security Event Token (SET) — RFC 8417](#2-security-event-token-set--rfc-8417)
3. [RISC (Risk and Incident Security Claims)](#3-risc-risk-and-incident-security-claims)
4. [SET Delivery — RFC 8935 (Push)](#4-set-delivery--rfc-8935-push)
5. [SET Delivery — RFC 8936 (Poll)](#5-set-delivery--rfc-8936-poll)
6. [OpenID SSF (Shared Signals Framework)](#6-openid-ssf-shared-signals-framework)
7. [GGID as Transmitter](#7-ggid-as-transmitter)
8. [GGID as Receiver](#8-ggid-as-receiver)
9. [NATS Integration](#9-nats-integration)
10. [Implementation Roadmap](#10-implementation-roadmap)
11. [Comparison with Commercial SSF Implementations](#11-comparison-with-commercial-ssf-implementations)

---

## 1. Overview

### 1.1 The Shared Signals and Events (SSE) Working Group

The **Shared Signals and Events (SSE)** Working Group at the IETF (now carried forward under the OpenID Foundation) was formed to solve a fundamental problem in modern identity infrastructure: **how do independent organizations communicate security state changes to each other in real time, using a standard, interoperable format?**

In today's federated identity ecosystems, a user's security posture is evaluated by multiple independent services — an Identity Provider (IdP), relying parties (RPs), security information and event management (SIEM) systems, zero-trust network access (ZTNA) brokers, and mobile device management (MDM) platforms. When a credential is compromised, a device falls out of compliance, or a session is revoked, that information must propagate to all interested parties as quickly as possible.

**The problem without SSE:**

- Token revocation latency: JWTs remain valid until expiry (typically 15-60 minutes). Relying parties cannot detect mid-session state changes without aggressive token refresh, which adds latency and load.
- No standard vocabulary: Each vendor uses proprietary webhooks, APIs, or log formats. Cross-organization event sharing requires custom integrations per partner.
- Manual correlation: Security operations teams manually correlate logs across disparate systems to build a complete picture of an incident.
- Federation blind spots: In B2B federation (e.g., a company federates to a SaaS app), the SaaS app has no way to know when the IdP disables a user account until the next login attempt.

**The SSE solution:**

The SSE Working Group defined a layered set of specifications:

```
┌──────────────────────────────────────────────────────────────┐
│                      Application Layer                        │
│                                                               │
│  ┌─────────┐  ┌──────────┐  ┌─────────────────────────────┐ │
│  │  CAEP   │  │   RISC   │  │  Custom Event Profiles      │ │
│  │ Events  │  │  Events  │  │  (Risk Mgmt, Compliance)    │ │
│  └────┬────┘  └────┬─────┘  └──────────┬──────────────────┘ │
│       │            │                    │                     │
│  ═════╧════════════╧════════════════════╧═════════════════   │
│              Shared Signals Framework (SSF)                    │
│  • SET format (RFC 8417)                                      │
│  • Push delivery (RFC 8935)                                   │
│  • Poll delivery (RFC 8936)                                   │
│  • Subject principals & claims                                │
│  • Transmitter/Receiver configuration metadata                │
│  • Stream management API                                      │
│  • Event verification flow                                    │
│  • Subject type negotiation                                   │
└──────────────────────────────────────────────────────────────┘
```

### 1.2 Core Specifications

| Specification | Scope | Status | Key Contribution |
|---|---|---|---|
| **RFC 8417** | SET event format | RFC (Informational) | Defines the JWT-based Security Event Token — the wire format for all security events |
| **RFC 8935** | SET delivery — push | RFC (Proposed Standard) | HTTP POST push delivery with retry, backoff, and error semantics |
| **RFC 8936** | SET delivery — poll | RFC (Proposed Standard) | HTTP GET poll delivery with cursor-based pagination and acknowledgment |
| **OpenID SSF 1.0** | Transport layer framework | Draft (implementer's draft) | Profiles SET for OIDC ecosystems: stream config, metadata discovery, verification, subject negotiation |
| **OpenID CAEP 1.0** | Event type vocabulary | Draft (implementer's draft) | Defines session/access-focused security events (session-revoked, credential-change, device-compliance-change, etc.) |
| **OpenID RISC 1.0** | Event type vocabulary | Draft | Defines account-lifecycle-focused security events (account-deleted, credential-change, tokens-revoked) |

### 1.3 Design Principles

The SSE specifications are built on several core design principles:

1. **SETs are not access tokens** — A SET is a JWT that carries security event data. It must never be presented to a resource server for authorization. The `typ` header is `secevent-jwt`, not `JWT`, to prevent confusion.

2. **Decoupled transport** — The SET format (RFC 8417) is independent of the delivery mechanism. A SET can be delivered via push (RFC 8935), poll (RFC 8936), or any future transport.

3. **Signature independence** — The SET's JWT signature is verified independently of transport-layer authentication (mTLS, OAuth bearer). This provides end-to-end integrity even if the transport is compromised.

4. **At-least-once delivery** — Both push and poll delivery provide at-least-once semantics. Receivers must handle duplicates via the `jti` claim (unique token identifier).

5. **Privacy-preserving subjects** — Events should use the minimum-necessary subject identifiers. Opaque identifiers are preferred over PII (email, phone) where possible.

6. **Receiver-driven configuration** — Receivers subscribe to specific event types and subjects. The transmitter only sends events the receiver has opted into.

---

## 2. Security Event Token (SET) — RFC 8417

### 2.1 SET Format

A **Security Event Token (SET)** is a JWT (RFC 7519) that conveys information about a single security event. Unlike authentication JWTs, SETs carry **event notifications**, not authorization grants. They are consumed by automated systems, not presented to resource servers.

#### SET Header

```json
{
  "typ": "secevent-jwt",
  "alg": "RS256",
  "kid": "transmitter-signing-key-1"
}
```

The `typ` header **must** be `secevent-jwt` to distinguish SETs from other JWT types (access tokens, ID tokens). This is a critical security measure — it prevents SETs from being accidentally accepted as access tokens by resource servers.

**Supported algorithms:** RS256, RS384, RS512, ES256, ES384, ES512. Symmetric algorithms (HS256) are discouraged for inter-organizational use because key sharing is impractical.

#### SET Payload

```json
{
  "jti": "5d54a1f2-9432-4e89-aa33-2488eff87f12",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "toe": 1737379199,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "event_timestamp": 1737379199,
      "reason": {
        "reason": "admin_revocation",
        "description": "Administrator revoked session"
      }
    }
  }
}
```

#### SET Claims Reference

| Claim | Required | Type | Description |
|---|---|---|---|
| `jti` | Yes | string | Unique SET identifier (UUID recommended). Used for dedup. Must be globally unique within the issuer. |
| `iss` | Yes | string (URL) | Issuer of the SET — the Transmitter's identifier URL. Must match the `iss` configured in the receiver's stream. |
| `aud` | Recommended | string or array | Intended audience — the Receiver's identifier. Used to route SETs to the correct stream. |
| `iat` | Yes | integer (epoch) | Issued-at timestamp — when the SET was created (not when the event occurred). |
| `toe` | Optional | integer (epoch) | Time of event — when the underlying security event actually occurred. If absent, receiver assumes `iat`. |
| `events` | Yes | object | JSON object mapping event type URIs to event payloads. Each SET may contain one or more events, though single-event SETs are strongly recommended. |
| `sub` | Conditional | object | Subject identifier — may appear at the top level or within each event payload. See [2.3 Subject Identifiers](#23-subject-identifiers). |

#### SET is NOT an Access Token

This distinction is the most critical security property of SETs. The RFC is explicit:

- A SET **must not** contain `scope`, `client_id`, or other OAuth authorization claims.
- A SET **must not** be presented to a resource server's authorization endpoint.
- The `typ` header value `secevent-jwt` enables resource servers to reject SETs if accidentally presented.
- A receiver that receives a SET at an API endpoint **must not** use it for authorization decisions — it should log the event and return an error.

**Validation flow for a resource server receiving a JWT:**

```
Received JWT
    │
    ▼
┌─────────────────┐    typ != "secevent-jwt"    ┌──────────────────────┐
│ Parse JWT header │ ─────────────────────────> │ Validate as access    │
│ Check typ claim  │                             │ token (normal OAuth)  │
└─────────────────┘                             └──────────────────────┘
    │
    │ typ == "secevent-jwt"
    ▼
┌─────────────────────────────────────────┐
│ REJECT: This is a SET, not an access    │
│ token. Return 401. Log security event.  │
│ Do NOT process authorization.           │
└─────────────────────────────────────────┘
```

### 2.2 Event URIs

Each event in a SET is identified by a **URI** that serves as the key in the `events` object. The URI identifies the event type and the specification that defines its payload structure.

#### Standard Event URI Namespaces

**CAEP Events** (Continuous Access Evaluation Protocol):

| Event URI | Description |
|---|---|
| `urn:ietf:params:sse:caep:session-revoked` | A user session has been revoked |
| `urn:ietf:params:sse:caep:credential-change` | User credentials have changed (password reset, MFA enrollment) |
| `urn:ietf:params:sse:caep:device-compliance-change` | A device's compliance status changed |
| `urn:ietf:params:sse:caep:assurance-level-change` | User's authentication assurance level changed |
| `urn:ietf:params:sse:caep:identifier-changed` | A user identifier changed (email, phone) |
| `urn:ietf:params:sse:caep:token-claims-change` | Claims in issued tokens have changed |

> Note: The OpenID SSF and CAEP specs also use the `https://schemas.openid.net/secevent/caep/event-type/` URI form. Both URN and URL forms are valid; receivers should accept both.

**RISC Events** (Risk and Incident Security Claims):

| Event URI | Description |
|---|---|
| `urn:ietf:params:sse:events:risc:account-deleted` | User account has been deleted |
| `urn:ietf:params:sse:events:risc:account-disabled` | User account has been disabled |
| `urn:ietf:params:sse:events:risc:account-enabled` | User account has been re-enabled |
| `urn:ietf:params:sse:events:risc:credential-change` | Credentials have been changed |
| `urn:ietf:params:sse:events:risc:sessions-revoked` | All sessions for a user have been revoked |
| `urn:ietf:params:sse:events:risc:tokens-revoked` | All tokens for a user have been revoked |

**Verification Events:**

| Event URI | Description |
|---|---|
| `urn:ietf:params:sse:verification` | Stream verification test event — sent during stream setup to confirm delivery works |

**Custom Events:**

Organizations may define custom event types using their own URI namespace:

```json
{
  "events": {
    "https://ggid.dev/events/risk-score-change": {
      "subject": { "subject_type": "iss_sub", "iss": "https://idp.ggid.dev", "sub": "user-123" },
      "risk_score": 85,
      "factors": ["impossible_travel", "new_device", "anonymous_proxy"]
    }
  }
}
```

#### Multiple Events in a Single SET

While single-event SETs are recommended for simplicity, the format supports multiple events:

```json
{
  "events": {
    "urn:ietf:params:sse:caep:session-revoked": { ... },
    "urn:ietf:params:sse:caep:credential-change": { ... }
  }
}
```

> Best practice: Send one event per SET. This simplifies error handling (one event fails → one SET retries), logging, and dedup.

### 2.3 Subject Identifiers

Every security event applies to a **subject** — the entity whose security state changed. The SSE Framework defines multiple subject identifier formats to accommodate different deployment models and privacy requirements.

#### Subject Identifier Formats

| Format | `subject_type` | Fields | Privacy Level | Example |
|---|---|---|---|---|
| **iss_sub** | `iss_sub` | `iss`, `sub` | High (opaque sub) | `{"subject_type":"iss_sub","iss":"https://idp.ggid.dev","sub":"550e8400-..."}` |
| **Email** | `email` | `email` | Low (PII) | `{"subject_type":"email","email":"user@example.com"}` |
| **Phone** | `phone` | `phone` (E.164) | Low (PII) | `{"subject_type":"phone","phone":"+15551234567"}` |
| **Opaque** | `opaque` | `id` | Highest | `{"subject_type":"opaque","id":"acct:user-123:ggid"}` |
| **DNSSub** | `dnssub` | `dns_domain`, `sub` | Medium | `{"subject_type":"dnssub","dns_domain":"example.com","sub":"user-123"}` |
| **JWT ID** | `jwt_id` | `iss`, `jti` | Medium | `{"subject_type":"jwt_id","iss":"https://idp.ggid.dev","jti":"token-id-123"}` |
| **URI** | `uri` | `uri` | Medium | `{"subject_type":"uri","uri":"https://idp.ggid.dev/users/123"}` |

#### Subject Identifier in Event Payload

The subject appears within each event payload (not necessarily at the SET top level):

```json
{
  "jti": "...",
  "iss": "https://idp.ggid.dev",
  "iat": 1737379200,
  "events": {
    "urn:ietf:params:sse:caep:session-revoked": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "session": {
        "id": "session:a1b2c3d4",
        "created_at": 1737000000
      },
      "reason": {
        "reason": "admin_revocation"
      }
    }
  }
}
```

#### Subject Filters

Receivers can configure **subject filters** on their event stream to only receive events for specific subjects. This is useful for:

- **Tenant-scoped streams:** Only receive events for users in a specific tenant
- **Application-scoped streams:** Only receive events relevant to one application
- **Compliance scopes:** Only receive events for regulated users

Subject filters are configured during stream creation:

```json
{
  "subject_filters": [
    {
      "subject_type": "iss_sub",
      "iss": "https://idp.ggid.dev"
    }
  ]
}
```

#### Privacy Principle

The SSE Framework recommends using **opaque or iss_sub** identifiers wherever possible. Email and phone identifiers should only be used when the receiver does not have a mapping from opaque identifiers to local user records, and only with explicit consent.

> GGID recommendation: Use `iss_sub` as the default subject format. GGID's internal user UUIDs serve as the `sub` value — no PII leaves the system.

### 2.4 SET Validation Checklist

A receiver must perform the following validation steps when processing a SET:

1. **Parse JWT** — Decode header and payload. If parsing fails, reject with 400.
2. **Check `typ`** — Header `typ` must be `secevent-jwt`. If not, reject with 400.
3. **Verify `iss`** — The `iss` claim must match the expected transmitter configured for this stream. If not, reject with 401.
4. **Verify `aud`** — The `aud` claim (if present) must match the receiver's identifier. If not, reject with 403.
5. **Verify signature** — Fetch the transmitter's JWKS (cached), find the key by `kid`, verify the JWT signature. If invalid, reject with 401.
6. **Check `iat`** — The `iat` timestamp should be recent (within a configured window, e.g., 5 minutes). If too old, reject with 400 (stale event).
7. **Check `jti` dedup** — Look up `jti` in the dedup cache. If already processed, return 200 (idempotent ack) but do not re-process.
8. **Process events** — For each event URI in the `events` object, dispatch to the appropriate handler.
9. **Acknowledge** — Return 200 OK (or 202 Accepted for async processing).

---

## 3. RISC (Risk and Incident Security Claims)

### 3.1 RISC Overview

RISC (Risk and Incident Security Claims) is one of the two primary event profiles built on top of SSF. While CAEP focuses on **session and access-level** events (is this session/device still trustworthy?), RISC focuses on **account lifecycle** events (does this account still exist? are its credentials valid?).

### 3.2 RISC Event Types

| Event | URI | Trigger | Receiver Action |
|---|---|---|---|
| **account-deleted** | `urn:ietf:params:sse:events:risc:account-deleted` | User account permanently deleted | Remove all local data, invalidate sessions and tokens |
| **account-disabled** | `urn:ietf:params:sse:events:risc:account-disabled` | Account temporarily disabled (policy violation, security hold) | Block new logins, revoke active sessions |
| **account-enabled** | `urn:ietf.params:sse:events:risc:account-enabled` | Account re-enabled after being disabled | Allow logins, restore access |
| **credential-change** | `urn:ietf:params:sse:events:risc:credential-change` | Password reset, MFA enrolled/removed, credential added/removed | Invalidate cached credential state, force re-auth on sensitive operations |
| **sessions-revoked** | `urn:ietf:params:sse:events:risc:sessions-revoked` | All sessions for a user revoked (global logout) | Invalidate all session tokens for the subject |
| **tokens-revoked** | `urn:ietf:params:sse:events:risc:tokens-revoked` | All OAuth tokens for a user revoked | Invalidate access and refresh tokens, force re-authorization |

### 3.3 RISC Event Example

**Account disabled event:**

```json
{
  "jti": "risc-evt-9b8a7f6c-1234-5678-9abc-def012345678",
  "iss": "https://idp.ggid.dev",
  "aud": "https://saas.example.com",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/risc/event-type/account-disabled": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "reason": "security_policy_violation"
    }
  }
}
```

**Credential change event:**

```json
{
  "jti": "risc-evt-3c4d5e6f-7890-abcd-ef12-345678901234",
  "iss": "https://idp.ggid.dev",
  "aud": "https://saas.example.com",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/risc/event-type/credential-change": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "credential_type": "password",
      "change_type": "reset",
      "event_timestamp": 1737379195
    }
  }
}
```

### 3.4 RISC vs CAEP Comparison

| Dimension | RISC | CAEP |
|---|---|---|
| **Focus** | Account lifecycle | Session and access evaluation |
| **Granularity** | User-level (whole account) | Session/device/token-level |
| **Temporal** | Discrete state changes | Continuous evaluation |
| **Primary consumers** | Account management, identity sync | Access control, zero-trust brokers |
| **Typical latency req** | Near real-time (seconds) | Ultra-low latency (sub-second) |
| **Events overlap** | `credential-change`, `sessions-revoked`, `tokens-revoked` also exist in CAEP | Same events but with different payload semantics |

**Overlap resolution:** The OpenID specs acknowledge that some events exist in both RISC and CAEP namespaces. In practice:

- Use **CAEP** `credential-change` when the change affects ongoing sessions (and you want session-level impact details).
- Use **RISC** `credential-change` when the change is informational (account state sync).
- A transmitter may emit both for the same underlying event, targeting different receiver types.

> **GGID recommendation:** Emit CAEP events for session/access-impacting changes (primary). Optionally emit RISC events for account lifecycle sync (Phase 4).

---

## 4. SET Delivery — RFC 8935 (Push)

RFC 8935 defines **push-based SET delivery**: the Transmitter sends SETs to the Receiver via HTTP POST over TLS. This is the primary delivery mechanism for real-time security event notification.

### 4.1 Push Flow

```
┌──────────────────┐                              ┌──────────────────┐
│    Transmitter    │   POST /sse/events           │     Receiver      │
│     (GGID)        │ ──────────────────────────>  │   (RP / SIEM)     │
│                   │   Content-Type:               │                   │
│                   │     application/secevent+jwt  │                   │
│                   │   Authorization: Bearer xxx   │                   │
│                   │   Body: <signed SET JWT>      │                   │
│                   │                              │                   │
│                   │   <- 200 OK ──────────────── │  Validate SET     │
│                   │      (acknowledged)           │  Process event   │
│                   │                              │  Return 2xx       │
└──────────────────┘                              └──────────────────┘
```

#### HTTP Request

```
POST /sse/events HTTP/1.1
Host: receiver.example.com
Content-Type: application/secevent+jwt
Authorization: Bearer eyJhbGciOi...receiver-access-token...

eyJraWQiOiJ0cmFuc21pdHRlci1zaWduaW5nLWtleS0xIiwidHlwIjoic2VjZXZlbnQtand0IiwiYWxnIjoiUlMyNTYifQ.eyJqdGkiOiI1ZDU0YTFmMi05NDMyLTRlODktYWEzMy0yNDg4ZWZmODdmMTIiLCJpc3MiOiJodHRwczovL2lkcC5nZ2lkLmRldiIsImF1ZCI6Imh0dHBzOi8vcmVjZWl2ZXIuZXhhbXBsZS5jb20iLCJpYXQiOjE3MzczNzkyMDAsImV2ZW50cyI6eyJ1cm46aWV0ZjpwYXJhbXM6c3NlOmNhZXA6c2Vzc2lvbi1yZXZva2VkIjp7InN1YmplY3QiOnsic3ViamVjdF90eXBlIjoiaXNzX3N1YiIsImlzcyI6Imh0dHBzOi8vaWRwLmdnaWQuZGV2Iiwic3ViIjoiNTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAwIn0sInNlc3Npb24iOnsiaWQiOiJzZXNzaW9uOmExYjJjM2Q0In19fQ.signature...
```

Key HTTP details:

- **Content-Type**: `application/secevent+jwt` — this is the standard media type for SET delivery.
- **Body**: The raw JWT string (compact serialization: `header.payload.signature`).
- **Authorization**: Bearer token for transport-layer auth (separate from SET signature verification).
- **Method**: POST only. GET requests must be rejected with 405.

#### HTTP Response

```
HTTP/1.1 200 OK
Content-Type: application/json

{}
```

On success, the receiver returns:
- **200 OK** — SET received, validated, and processed.
- **202 Accepted** — SET received and queued for async processing (accepted but not yet fully processed).

The response body should be empty or contain an empty JSON object `{}`.

### 4.2 Push Error Handling

RFC 8935 defines a comprehensive error response taxonomy. Each error category triggers different transmitter behavior.

#### Error Response Format

Errors use the standard SET error response format:

```json
{
  "err": "jwtAudValidation",
  "description": "The SET was intended for a different receiver"
}
```

| Field | Required | Description |
|---|---|---|
| `err` | Yes | Error code from the standard set below |
| `description` | No | Human-readable description |

#### Error Response Codes

**Non-Retriable Errors (drop the SET):**

| HTTP Status | Error Code | Meaning | Transmitter Action |
|---|---|---|---|
| **400 Bad Request** | `badRequest` | Malformed SET, invalid JSON, missing required claims | Log error, mark SET as permanently failed, do not retry |
| **400 Bad Request** | `jwtParse` | SET JWT could not be parsed | Log, drop SET |
| **400 Bad Request** | `jwtTyp` | `typ` header is not `secevent-jwt` | Log, drop SET |
| **400 Bad Request** | `jwtAudValidation` | `aud` claim does not match receiver | Log, drop SET |
| **400 Bad Request** | `jwtClaimValidation` | Other claim validation failure | Log, drop SET |
| **401 Unauthorized** | `authReq` | Missing or invalid transport auth (bearer token, mTLS) | Log, alert ops, do not retry until auth is fixed |
| **403 Forbidden** | `forbidden` | Transmitter is not authorized to send to this receiver | Log, alert ops, do not retry |
| **404 Not Found** | `notFound` | Push endpoint URL is wrong or stream no longer exists | Log, update stream config, do not retry |
| **410 Gone** | `gone` | Stream has been permanently deleted | Log, mark stream as inactive |

**Retriable Errors (retry with backoff):**

| HTTP Status | Error Code | Meaning | Transmitter Action |
|---|---|---|---|
| **429 Too Many Requests** | `tooManyRequests` | Receiver is rate-limiting. Should include `Retry-After` header. | Back off, wait for `Retry-After` or exponential backoff, retry |
| **500 Internal Server Error** | `serverError` | Receiver had an internal error | Retry with exponential backoff |
| **502 Bad Gateway** | — | Proxy/gateway error | Retry with backoff |
| **503 Service Unavailable** | `serviceUnavailable` | Receiver is temporarily down. May include `Retry-After`. | Retry with backoff |

#### Retry Strategy

```
Attempt 1: Immediate
    │ fail (5xx/429)
    ▼
Wait 1s
    │ Attempt 2
    │ fail
    ▼
Wait 2s
    │ Attempt 3
    │ fail
    ▼
Wait 4s
    │ Attempt 4
    │ fail
    ▼
Wait 8s
    │ Attempt 5
    │ fail
    ▼
Wait 16s ... up to max retry count (default: 5)
    │
    ▼
Give up: log to dead-letter queue, alert ops
```

**Retry parameters (RFC 8935 recommendations):**

| Parameter | Default | Description |
|---|---|---|
| Max retries | 5 | Maximum delivery attempts before giving up |
| Initial backoff | 1 second | Wait before first retry |
| Max backoff | 60 seconds | Cap on backoff interval |
| Backoff multiplier | 2.0 | Each retry waits previous * multiplier |
| Jitter | ±20% | Randomized jitter to avoid thundering herd |
| `Retry-After` | — | If receiver sends `Retry-After` header, use that value instead of computed backoff |

#### ACK Semantics

A SET is considered **delivered** when the receiver returns any 2xx status code. Once delivered:

1. The transmitter removes the SET from its delivery queue.
2. The transmitter does not retry delivery.
3. The event is logged as `delivered`.

If the receiver returns a non-2xx status and retries are exhausted, the SET is moved to a **dead-letter queue** for manual investigation.

### 4.3 Push Security

#### Transport-Layer Authentication

The HTTP connection between transmitter and receiver must be authenticated. RFC 8935 supports two primary mechanisms:

**Mutual TLS (mTLS):**

```
┌──────────────────┐                              ┌──────────────────┐
│   Transmitter     │  ── mTLS handshake ──────>  │     Receiver      │
│  client cert:     │  server verifies client     │  server cert:     │
│  trans-cert.pem   │  cert against trust store    │  receiver-cert    │
│                   │  <── mTLS handshake ──────  │                   │
│                   │  transmitter verifies        │                   │
│                   │  server cert                 │                   │
│                   │                              │                   │
│                   │  POST /sse/events            │                   │
│                   │  (no Authorization header    │                   │
│                   │   needed — mTLS proves       │                   │
│                   │   identity)                  │                   │
└──────────────────┘                              └──────────────────┘
```

- Both sides present X.509 certificates during TLS handshake.
- Certificates are validated against a pre-configured trust store.
- No additional `Authorization` header needed — mTLS proves both identities.
- Best for: B2B federation where both parties have PKI infrastructure.

**OAuth 2.0 Bearer Token:**

```
POST /sse/events HTTP/1.1
Authorization: Bearer eyJhbGciOi...receiver-access-token...
```

- Transmitter obtains an access token from the receiver's token endpoint (client credentials grant).
- Token is included in the `Authorization` header.
- Receiver validates the token before processing the SET.
- Best for: SaaS-to-SaaS integrations where mTLS is impractical.

#### SET Signature Verification

Independent of transport auth, the receiver must verify the SET's JWT signature:

1. Parse the `kid` from the JWT header.
2. Fetch the transmitter's JWKS from the configured `jwks_uri` (cache aggressively, refresh on key rotation).
3. Find the key matching `kid`.
4. Verify the JWT signature using the matching public key.
5. If `kid` is not found in JWKS, force a JWKS refresh and retry once.

This two-layer security model ensures:

- **Transport auth** proves the HTTP connection is from the expected transmitter.
- **SET signature** proves the SET content was not tampered with since the transmitter signed it.
- Even if the transport is compromised (e.g., a proxy is breached), the SET signature prevents forgery.

#### Replay Prevention

Since delivery is at-least-once, receivers must handle duplicate SETs. The `jti` claim is the dedup key:

```
Incoming SET
    │
    ▼
Extract jti
    │
    ▼
Check dedup cache (Redis, with TTL = 24h)
    │
    ├── jti found ──> Return 200 OK (already processed), skip event handler
    │
    └── jti not found ──> Process event, store jti in cache, return 200 OK
```

- **Dedup TTL:** 24 hours is recommended (long enough to cover retry windows, short enough to avoid unbounded growth).
- **Dedup storage:** Redis with `SET jti:xxx 1 EX 86400` is ideal for high-throughput receivers.
- **Idempotency:** Event handlers should be idempotent as a defense-in-depth measure.

---

## 5. SET Delivery — RFC 8936 (Poll)

RFC 8936 defines **poll-based SET delivery**: the Receiver polls the Transmitter's events endpoint to retrieve batches of SETs. This is the alternative to push delivery, optimized for receivers that cannot or prefer not to expose an inbound HTTP endpoint.

### 5.1 Poll Flow

```
┌──────────────────┐                              ┌──────────────────┐
│     Receiver      │   GET /sse/poll?            │    Transmitter    │
│   (RP / SIEM)     │   resultsAfter=cursor123    │     (GGID)        │
│                   │ ──────────────────────────> │                   │
│                   │   Authorization: Bearer xxx │                   │
│                   │                              │  Find events after cursor
│                   │                              │  Package into batch
│                   │   <- 200 OK ───────────────  │                   │
│                   │   [{ sets: [...],            │                   │
│                   │     moreResults: false }]    │                   │
│                   │                              │                   │
│  Validate SETs     │                              │                   │
│  Process events    │                              │                   │
│                   │   POST /sse/poll/ack         │                   │
│                   │   { jti: ["jti1","jti2"] }   │                   │
│                   │ ──────────────────────────> │  Remove acknowledged
│                   │                              │  events from queue
│                   │   <- 200 OK ─────────────── │                   │
└──────────────────┘                              └──────────────────┘
```

#### Poll Request

```
GET /sse/poll?resultsAfter=eyJjdXJzb3IiOiIyMDI1LTAxLTIwVDEwOjAwOjAwWiJ9 HTTP/1.1
Host: transmitter.ggid.dev
Authorization: Bearer eyJhbGciOi...transmitter-access-token...
Accept: application/secevent+jwt
```

Query parameters:

| Parameter | Required | Description |
|---|---|---|
| `resultsAfter` | Yes (after first poll) | Opaque cursor returned by the previous poll response. Omit on first poll to get the oldest unacknowledged events. |
| `limit` | No | Maximum number of SETs to return in this batch. Default is implementation-defined (e.g., 100). Max is also implementation-defined. |

#### Poll Response

```
HTTP/1.1 200 OK
Content-Type: application/json

{
  "sets": [
    "eyJraWQiOi...base64url-encoded-SET-JWT-1...",
    "eyJraWQiOi...base64url-encoded-SET-JWT-2...",
    "eyJraWQiOi...base64url-encoded-SET-JWT-3..."
  ],
  "moreResults": false,
  "resultsAfter": "eyJjdXJzb3IiOiIyMDI1LTAxLTIwVDEwOjAxOjAwWiJ9"
}
```

| Field | Required | Description |
|---|---|---|
| `sets` | Yes | Array of SET JWT strings (compact serialization). May be empty if no events are pending. |
| `moreResults` | Yes | `true` if more unacknowledged events are available beyond this batch. Receiver should poll again immediately. |
| `resultsAfter` | Yes | Opaque cursor for the next poll request. Pass this as `resultsAfter` in the next `GET /sse/poll`. |

#### Acknowledgment Request

After processing the batch, the receiver acknowledges:

```
POST /sse/poll/ack HTTP/1.1
Host: transmitter.ggid.dev
Authorization: Bearer eyJhbGciOi...transmitter-access-token...
Content-Type: application/json

{
  "jti": [
    "5d54a1f2-9432-4e89-aa33-2488eff87f12",
    "3c4d5e6f-7890-abcd-ef12-345678901234",
    "9b8a7f6c-1234-5678-9abc-def012345678"
  ]
}
```

The transmitter removes the acknowledged SETs from its delivery queue. Unacknowledged SETs remain in the queue and will be returned on subsequent polls.

#### Acknowledgment Response

```
HTTP/1.1 200 OK
Content-Type: application/json

{}
```

### 5.2 Poll Cursors

The `resultsAfter` cursor is an **opaque token** — the receiver must not interpret its contents. The transmitter generates it and it may encode:

- A timestamp of the last event in the batch
- A database sequence number
- A Redis sorted set score
- An opaque encrypted position marker

**Cursor lifecycle:**

1. First poll: omit `resultsAfter` (or pass an initial cursor from stream configuration).
2. Transmitter returns events starting from the stream's beginning (or from the last acknowledged position).
3. Each response includes a new `resultsAfter` cursor.
4. Receiver stores the cursor and uses it in the next poll.
5. If the receiver loses the cursor, it can restart from the beginning (but will receive already-processed events — dedup via `jti` handles this).

**Limit parameter:**

```
GET /sse/poll?resultsAfter=cursor123&limit=50
```

Controls batch size. Larger batches reduce HTTP overhead but increase memory pressure and latency for individual events. Typical values: 25-200.

### 5.3 Poll Error Handling

The poll endpoint uses the same error response format as push:

| HTTP Status | Meaning | Receiver Action |
|---|---|---|
| **400 Bad Request** | Invalid cursor, malformed request | Reset cursor, re-poll from start |
| **401 Unauthorized** | Invalid transport auth | Refresh token, retry |
| **403 Forbidden** | Not authorized for this stream | Alert ops, stop polling |
| **404 Not Found** | Stream deleted | Stop polling, alert ops |
| **429 Too Many Requests** | Rate limited | Back off, respect `Retry-After` |
| **500/502/503** | Transmitter error | Retry with backoff |

### 5.4 Poll vs Push Tradeoffs

| Dimension | Push (RFC 8935) | Poll (RFC 8936) |
|---|---|---|
| **Latency** | Ultra-low (sub-second to seconds) | Higher (poll interval dependent, typically 1-30s) |
| **Receiver requirements** | Must expose inbound HTTPS endpoint | No inbound endpoint needed |
| **Firewall friendliness** | Requires receiver to be reachable from transmitter | Works behind NAT/firewall (outbound only) |
| **Backpressure** | Receiver handles backpressure via 429 | Receiver controls pace via poll interval and batch size |
| **Reliability** | Transmitter manages retry queue | Transmitter manages delivery queue until ack |
| **State** | Transmitter tracks delivery status per SET | Transmitter tracks cursor + ack per stream |
| **Complexity** | Moderate (retry logic, backoff, dead-letter) | Simpler (poll loop, ack batch) |
| **Throughput** | Higher (parallel POST per event) | Moderate (batch retrieval) |
| **Use case** | Real-time session revocation, zero-trust | SIEM ingestion, batch processing, compliance archiving |
| **Connection model** | Transmitter initiates connection to receiver | Receiver initiates connection to transmitter |
| **Failure mode** | If receiver is down, events queue at transmitter | If transmitter is down, receiver re-polls with backoff |

**Decision matrix:**

| If receiver is... | Use |
|---|---|
| Behind corporate firewall (no inbound) | **Poll** |
| A cloud SaaS (always reachable) | **Push** |
| A SIEM with batch ingestion pipeline | **Poll** |
| A zero-trust broker needing sub-second revocation | **Push** |
| A compliance archive (daily batches OK) | **Poll** |
| Both options wanted (dual delivery for HA) | **Both** (configure two streams) |

> **GGID recommendation:** Support both push and poll. Default new streams to push. Offer poll for partners behind firewalls.

---

## 6. OpenID SSF (Shared Signals Framework)

The OpenID SSF specification profiles the IETF SET specs (RFC 8417/8935/8936) for use in OIDC-based identity ecosystems. It adds: metadata discovery, stream management, event verification, and subject type negotiation.

### 6.1 SSF Metadata Discovery

The Transmitter publishes its SSF configuration at a well-known endpoint. This enables receivers to auto-discover endpoints, supported events, and signing keys.

#### Discovery Endpoint

```
GET /.well-known/ssf-configuration HTTP/1.1
Host: idp.ggid.dev
```

#### Metadata Response

```json
{
  "issuer": "https://idp.ggid.dev",
  "jwks_uri": "https://idp.ggid.dev/.well-known/jwks.json",
  "delivery_methods_supported": [
    "https://schemas.openid.net/secevent/risc/delivery-method/push",
    "https://schemas.openid.net/secevent/risc/delivery-method/poll"
  ],
  "push_endpoint": "https://idp.ggid.dev/ssf/events",
  "poll_endpoint": "https://idp.ggid.dev/ssf/poll",
  "ack_endpoint": "https://idp.ggid.dev/ssf/poll/ack",
  "verification_endpoint": "https://idp.ggid.dev/ssf/verify",
  "stream_management_endpoint": "https://idp.ggid.dev/ssf/streams",
  "subject_types_supported": [
    "iss_sub",
    "email",
    "phone",
    "opaque"
  ],
  "events_supported": [
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "https://schemas.openid.net/secevent/caep/event-type/credential-change",
    "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change",
    "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change",
    "https://schemas.openid.net/secevent/caep/event-type/identifier-changed",
    "https://schemas.openid.net/secevent/caep/event-type/token-claims-change",
    "https://schemas.openid.net/secevent/risc/event-type/account-deleted",
    "https://schemas.openid.net/secevent/risc/event-type/account-disabled",
    "https://schemas.openid.net/secevent/risc/event-type/account-enabled",
    "https://schemas.openid.net/secevent/risc/event-type/sessions-revoked",
    "https://schemas.openid.net/secevent/risc/event-type/tokens-revoked"
  ]
}
```

#### Metadata Fields

| Field | Required | Description |
|---|---|---|
| `issuer` | Yes | The Transmitter's issuer identifier (URL). Must match the `iss` claim in emitted SETs. |
| `jwks_uri` | Yes | URL of the Transmitter's JSON Web Key Set (public signing keys). |
| `delivery_methods_supported` | Yes | Array of supported delivery method URIs. |
| `push_endpoint` | Conditional | Receiver's push endpoint URL (if push is supported). |
| `poll_endpoint` | Conditional | Transmitter's poll endpoint URL (if poll is supported). |
| `ack_endpoint` | Conditional | Transmitter's acknowledgment endpoint for poll delivery. |
| `verification_endpoint` | Yes | URL for triggering stream verification. |
| `stream_management_endpoint` | Yes | URL for creating/updating/deleting event streams. |
| `subject_types_supported` | Yes | Array of subject identifier formats the transmitter can emit. |
| `events_supported` | Yes | Array of event type URIs the transmitter can emit. |

### 6.2 Stream Configuration

An **event stream** is a configured relationship between a Transmitter and a Receiver. It defines:

- Which events the receiver wants (event subscription)
- Which subjects the receiver cares about (subject filters)
- How events are delivered (push URL or poll endpoint)
- What subject identifier formats to use

#### Create Stream

```
POST /ssf/streams HTTP/1.1
Host: idp.ggid.dev
Authorization: Bearer eyJhbGciOi...management-token...
Content-Type: application/json

{
  "delivery": {
    "method": "https://schemas.openid.net/secevent/risc/delivery-method/push",
    "endpoint_url": "https://receiver.example.com/sse/events"
  },
  "events_requested": [
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "https://schemas.openid.net/secevent/caep/event-type/credential-change",
    "https://schemas.openid.net/secevent/risc/event-type/account-disabled"
  ],
  "subject_types": ["iss_sub"],
  "subject_filters": [
    {
      "subject_type": "iss_sub",
      "iss": "https://idp.ggid.dev"
    }
  ],
  "description": "Production security event stream for SaaS app XYZ"
}
```

#### Stream Response

```
HTTP/1.1 201 Created
Content-Type: application/json

{
  "stream_id": "stream-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "description": "Production security event stream for SaaS app XYZ",
  "delivery": {
    "method": "https://schemas.openid.net/secevent/risc/delivery-method/push",
    "endpoint_url": "https://receiver.example.com/sse/events"
  },
  "events_requested": [
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "https://schemas.openid.net/secevent/caep/event-type/credential-change",
    "https://schemas.openid.net/secevent/risc/event-type/account-disabled"
  ],
  "events_enabled": [
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "https://schemas.openid.net/secevent/caep/event-type/credential-change",
    "https://schemas.openid.net/secevent/risc/event-type/account-disabled"
  ],
  "subject_types": ["iss_sub"],
  "subject_filters": [
    {
      "subject_type": "iss_sub",
      "iss": "https://idp.ggid.dev"
    }
  ],
  "status": "enabled",
  "created_at": "2025-01-20T10:00:00Z",
  "updated_at": "2025-01-20T10:00:00Z"
}
```

#### Stream Lifecycle

| Operation | Method | Endpoint | Description |
|---|---|---|---|
| Create stream | `POST` | `/ssf/streams` | Create a new event stream |
| Get stream | `GET` | `/ssf/streams/{stream_id}` | Retrieve stream configuration |
| Update stream | `PATCH` | `/ssf/streams/{stream_id}` | Update delivery URL, events, filters |
| Delete stream | `DELETE` | `/ssf/streams/{stream_id}` | Permanently remove stream |
| Verify stream | `POST` | `/ssf/verify` | Trigger a verification test event |
| Pause stream | `PATCH` | `/ssf/streams/{stream_id}` | Set status to `paused` (events queue but not delivered) |
| Resume stream | `PATCH` | `/ssf/streams/{stream_id}` | Set status to `enabled` (resume delivery) |

#### Stream Status

| Status | Events emitted? | Events queued? | Description |
|---|---|---|---|
| `enabled` | Yes | N/A (pushed immediately) | Active stream, events flowing |
| `paused` | No | Yes (queued for poll or later push) | Temporarily suspended |
| `disabled` | No | No (dropped) | Permanently disabled |

### 6.3 Event Verification

Before a stream goes live, the receiver should verify that delivery works end-to-end. The SSF defines a **verification flow** for this.

#### Verification Flow

```
┌──────────┐                          ┌──────────────┐
│ Receiver  │  POST /ssf/verify        │ Transmitter   │
│           │  { stream_id: "...",     │               │
│           │    state: "random123" }  │               │
│           │ ──────────────────────>  │               │
│           │                          │  Generate verification SET
│           │  <- 200 OK ────────────  │  with state claim
│           │                          │               │
│           │                          │  Push verification SET to
│           │                          │  receiver's push endpoint
│           │  POST /sse/events        │               │
│           │  <─────────────────────  │               │
│           │  (verification SET)      │               │
│           │                          │               │
│  Validate SET                         │               │
│  Check state matches                 │               │
│  Return 200 OK ──────────────────>   │               │
│           │                          │  Stream verified!  │
│           │                          │  Status: enabled  │
└──────────┘                          └──────────────┘
```

#### Verification Request

```
POST /ssf/verify HTTP/1.1
Host: idp.ggid.dev
Authorization: Bearer eyJhbGciOi...management-token...
Content-Type: application/json

{
  "stream_id": "stream-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "state": "verification-state-random-xyz789"
}
```

The `state` parameter is a receiver-generated random string. It will be included in the verification SET so the receiver can correlate the test event with this verification request.

#### Verification SET

The transmitter sends a special verification event:

```json
{
  "jti": "verify-evt-12345678-1234-5678-abcd-123456789012",
  "iss": "https://idp.ggid.dev",
  "aud": "https://receiver.example.com",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/risc/event-type/verification": {
      "state": "verification-state-random-xyz789"
    }
  }
}
```

> Note: The verification event does NOT contain a subject claim — it is a transport-layer test, not a real security event.

The receiver:
1. Validates the SET signature.
2. Checks that the `state` matches the value it sent in the verification request.
3. Returns 200 OK to acknowledge receipt.
4. Optionally calls the transmitter's verification status endpoint to confirm success.

### 6.4 Subject Principal Negotiation

Different receivers have different capabilities for mapping subjects. A SaaS app may know users by email; a zero-trust broker may only understand opaque identifiers. SSF provides a negotiation mechanism.

#### Negotiation Flow

During stream creation, the receiver specifies which subject formats it supports:

```json
{
  "subject_types": ["iss_sub", "email"]
}
```

The transmitter intersects this with its own supported formats (`subject_types_supported` from metadata). The resulting stream configuration includes the agreed-upon formats:

```json
{
  "subject_types": ["iss_sub"]
}
```

If no overlap exists, stream creation fails with a negotiation error.

#### Subject Type Format Negotiation Matrix

| Receiver supports | Transmitter supports | Negotiated | Notes |
|---|---|---|---|
| `iss_sub`, `email` | `iss_sub`, `opaque` | `iss_sub` | Best privacy option |
| `email` only | `email`, `iss_sub` | `email` | Falls back to PII (receiver limited) |
| `opaque` only | `opaque`, `iss_sub` | `opaque` | Good privacy |
| `phone` only | `iss_sub`, `email` | FAIL | No overlap — stream creation rejected |

#### Multi-format Streams

A stream can negotiate multiple subject formats. The transmitter then uses the most privacy-preserving format available for each event:

```json
{
  "subject_types": ["iss_sub", "email", "phone"]
}
```

Priority order (transmitter preference):
1. `opaque` (highest privacy)
2. `iss_sub`
3. `dnssub`
4. `jwt_id`
5. `email`
6. `phone` (lowest privacy, most PII)

---

## 7. GGID as Transmitter

GGID's audit service already captures security-relevant events via NATS JetStream. Adding SSF transmitter capability enables external partners to subscribe to GGID's security events in real time.

### 7.1 Transmitter Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       GGID Platform                          │
│                                                               │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌──────────────┐   │
│  │  Auth   │  │ Policy  │  │   Org   │  │   OAuth       │   │
│  │ Service │  │ Service │  │ Service │  │   Service     │   │
│  └────┬────┘  └────┬────┘  └────┬────┘  └──────┬───────┘   │
│       │            │            │               │            │
│       └────────────┴────────────┴───────────────┘            │
│                          │                                    │
│                    NATS JetStream                             │
│                  (audit.events subject)                       │
│                          │                                    │
│           ┌──────────────┼──────────────┐                    │
│           │              │              │                     │
│    ┌──────▼──────┐ ┌────▼─────┐ ┌──────▼──────┐             │
│    │   Audit     │ │   SSF    │ │  Future     │             │
│    │  Consumer   │ │Transmitter│ │  Consumers  │             │
│    │ (persist)   │ │  (push)   │ │             │             │
│    └─────────────┘ └────┬─────┘ └─────────────┘             │
│                         │                                    │
│    ┌────────────────────┼────────────────────┐              │
│    │            SSF Transmitter API           │              │
│    │  ┌─────────────┐  ┌──────────────┐      │              │
│    │  │ /.well-known│  │ /ssf/streams │      │              │
│    │  │ /ssf-config │  │ /ssf/verify  │      │              │
│    │  │ /ssf/events │  │ /ssf/poll    │      │              │
│    │  └─────────────┘  └──────────────┘      │              │
│    └─────────────────────────────────────────┘              │
└─────────────────────────────────────────────────────────────┘
                         │
                    HTTPS / mTLS
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐   ┌─────▼─────┐   ┌────▼────┐
    │ Partner  │   │   SIEM    │   │ Zero    │
    │   SaaS   │   │ (Splunk)  │   │ Trust   │
    └─────────┘   └───────────┘   └─────────┘
```

#### New Endpoints Required

| Endpoint | Method | Purpose |
|---|---|---|
| `/.well-known/ssf-configuration` | GET | Metadata discovery |
| `/ssf/streams` | POST, GET | Create/list event streams |
| `/ssf/streams/{id}` | GET, PATCH, DELETE | Manage individual stream |
| `/ssf/verify` | POST | Trigger verification event |
| `/ssf/events` | POST | Push delivery target (receiver-facing, not used by transmitter) |
| `/ssf/poll` | GET | Poll delivery — receiver retrieves SET batches |
| `/ssf/poll/ack` | POST | Poll delivery — receiver acknowledges SETs |
| `/.well-known/jwks.json` | GET | Transmitter's signing keys (may reuse OIDC JWKS) |

### 7.2 Push Delivery Implementation

#### Core Go Types

```go
// Package ssf implements the Shared Signals Framework transmitter.
package ssf

import (
    "context"
    "crypto/rsa"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "github.com/nats-io/nats.go/jetstream"
)

// SETPayload represents the claims in a Security Event Token.
type SETPayload struct {
    JTI    string                 `json:"jti"`
    ISS    string                 `json:"iss"`
    AUD    string                 `json:"aud,omitempty"`
    IAT    int64                  `json:"iat"`
    TOE    int64                  `json:"toe,omitempty"`
    Events map[string]interface{} `json:"events"`
}

// SETHeader is the JWT header for a Security Event Token.
type SETHeader struct {
    Typ string `json:"typ"` // Always "secevent-jwt"
    Alg string `json:"alg"` // e.g., "RS256"
    KID string `json:"kid"` // Signing key ID
}

// EventStream represents a configured transmitter-to-receiver relationship.
type EventStream struct {
    ID              string        `json:"stream_id"`
    TenantID        uuid.UUID     `json:"-"`
    Description     string        `json:"description"`
    DeliveryMethod  string        `json:"delivery_method"` // push or poll
    EndpointURL     string        `json:"endpoint_url"`     // for push
    EventsRequested []string      `json:"events_requested"`
    EventsEnabled   []string      `json:"events_enabled"`
    SubjectTypes    []string      `json:"subject_types"`
    SubjectFilters  []SubjectFilter `json:"subject_filters"`
    Status          string        `json:"status"` // enabled, paused, disabled
    CreatedAt       time.Time     `json:"created_at"`
    UpdatedAt       time.Time     `json:"updated_at"`
    mux             sync.RWMutex
}

// SubjectFilter restricts which subjects a stream receives events for.
type SubjectFilter struct {
    SubjectType string `json:"subject_type"`
    ISS         string `json:"iss,omitempty"`
    Sub         string `json:"sub,omitempty"`
}

// SETTransmitter signs and delivers SETs to external receivers.
type SETTransmitter struct {
    issuer      string
    signingKey  *rsa.PrivateKey
    keyID       string
    httpClient  *http.Client
    streams     map[string]*EventStream
    dedupCache  DedupCache
    js          jetstream.JetStream // NATS connection for event source
    mu          sync.RWMutex
}

// NewSETTransmitter creates a new SSF transmitter.
func NewSETTransmitter(issuer string, key *rsa.PrivateKey, keyID string, js jetstream.JetStream) *SETTransmitter {
    return &SETTransmitter{
        issuer:     issuer,
        signingKey: key,
        keyID:      keyID,
        httpClient: &http.Client{Timeout: 10 * time.Second},
        streams:    make(map[string]*EventStream),
        dedupCache: NewRedisDedupCache(),
        js:         js,
    }
}
```

#### Signing a SET

```go
// SignSET creates a signed JWT SET from the given payload.
func (t *SETTransmitter) SignSET(payload SETPayload) (string, error) {
    // Enforce required fields
    if payload.JTI == "" {
        payload.JTI = uuid.New().String()
    }
    if payload.ISS == "" {
        payload.ISS = t.issuer
    }
    if payload.IAT == 0 {
        payload.IAT = time.Now().UTC().Unix()
    }

    // Create the JWT token with secevent-jwt type
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
        "jti":    payload.JTI,
        "iss":    payload.ISS,
        "aud":    payload.AUD,
        "iat":    payload.IAT,
        "toe":    payload.TOE,
        "events": payload.Events,
    })

    // Set the secevent-jwt type header
    token.Header["typ"] = "secevent-jwt"
    token.Header["kid"] = t.keyID

    // Sign and return compact serialization
    signed, err := token.SignedString(t.signingKey)
    if err != nil {
        return "", fmt.Errorf("sign SET: %w", err)
    }

    return signed, nil
}
```

#### Push Delivery to a Single Receiver

```go
// PushDelivery delivers a single SET to a receiver's push endpoint.
func (t *SETTransmitter) PushDelivery(ctx context.Context, stream *EventStream, setJWT string) error {
    t.mu.RLock()
    endpoint := stream.EndpointURL
    status := stream.Status
    t.mu.RUnlock()

    if status != "enabled" {
        return fmt.Errorf("stream %s is not enabled (status=%s)", stream.ID, status)
    }

    // Build the HTTP request
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(setJWT))
    if err != nil {
        return fmt.Errorf("create push request: %w", err)
    }
    req.Header.Set("Content-Type", "application/secevent+jwt")
    req.Header.Set("Authorization", "Bearer "+stream.pushAccessToken)

    // Send with retry
    return t.sendWithRetry(ctx, req, stream)
}

// sendWithRetry sends the push request with exponential backoff.
func (t *SETTransmitter) sendWithRetry(ctx context.Context, req *http.Request, stream *EventStream) error {
    maxRetries := 5
    backoff := 1 * time.Second

    for attempt := 0; attempt <= maxRetries; attempt++ {
        resp, err := t.httpClient.Do(req)
        if err != nil {
            if attempt < maxRetries {
                select {
                case <-ctx.Done():
                    return ctx.Err()
                case <-time.After(backoff):
                    backoff *= 2
                    if backoff > 60*time.Second {
                        backoff = 60 * time.Second
                    }
                    continue
                }
            }
            return fmt.Errorf("push delivery failed after %d attempts: %w", attempt+1, err)
        }
        defer resp.Body.Close()

        // Check response code
        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            // Success — SET acknowledged
            return nil
        }

        if isRetriable(resp.StatusCode) {
            // Retriable error — back off and retry
            retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
            if retryAfter > 0 {
                backoff = time.Duration(retryAfter) * time.Second
            }
            if attempt < maxRetries {
                select {
                case <-ctx.Done():
                    return ctx.Err()
                case <-time.After(backoff):
                    backoff *= 2
                    if backoff > 60*time.Second {
                        backoff = 60 * time.Second
                    }
                    continue
                }
            }
            return fmt.Errorf("push delivery failed: receiver returned %d after %d retries",
                resp.StatusCode, attempt+1)
        }

        // Non-retriable error — drop SET
        return fmt.Errorf("push delivery permanently failed: receiver returned %d", resp.StatusCode)
    }

    return fmt.Errorf("push delivery exhausted retries")
}

func isRetriable(statusCode int) bool {
    return statusCode == 429 || statusCode >= 500
}
```

#### NATS to SET Bridge

```go
// StartPushBridge subscribes to NATS audit events and pushes them as SETs
// to all configured push-delivery streams.
func (t *SETTransmitter) StartPushBridge(ctx context.Context) error {
    // Create a NATS consumer for SSF events
    cons, err := t.js.CreateOrUpdateConsumer(ctx, "AUDIT_EVENTS", jetstream.ConsumerConfig{
        Name:          "ssf-push-transmitter",
        Durable:       "ssf-push-transmitter",
        FilterSubject: "audit.events.>",
        AckPolicy:     jetstream.AckExplicitPolicy,
        MaxDeliver:    5,
    })
    if err != nil {
        return fmt.Errorf("create NATS consumer: %w", err)
    }

    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
            }

            batch, err := cons.FetchNoWait(10)
            if err != nil {
                if err == jetstream.ErrNoMessages {
                    time.Sleep(500 * time.Millisecond)
                    continue
                }
                time.Sleep(time.Second)
                continue
            }

            for msg := range batch.Messages() {
                if err := t.processNATSMessage(ctx, msg); err != nil {
                    msg.Nak() // Re-deliver
                } else {
                    msg.Ack()
                }
            }
        }
    }()

    return nil
}

// processNATSMessage converts a NATS audit event to a SET and pushes it
// to all matching streams.
func (t *SETTransmitter) processNATSMessage(ctx context.Context, msg jetstream.Msg) error {
    // Decode the NATS message (GGID audit event format)
    var auditEvent AuditEventPayload
    if err := json.Unmarshal(msg.Data(), &auditEvent); err != nil {
        return nil // Can't decode — ack and drop (don't retry forever)
    }

    // Map GGID audit event to CAEP/RISC event URI
    eventURI := mapAuditEventToSSFURI(auditEvent.Action)
    if eventURI == "" {
        return nil // Not a security event worth transmitting
    }

    // Build the SET payload
    payload := SETPayload{
        JTI: uuid.New().String(),
        ISS: t.issuer,
        IAT: time.Now().UTC().Unix(),
        TOE: auditEvent.CreatedAt.Unix(),
        Events: map[string]interface{}{
            eventURI: map[string]interface{}{
                "subject": map[string]interface{}{
                    "subject_type": "iss_sub",
                    "iss":          t.issuer,
                    "sub":          auditEvent.ActorID.String(),
                },
                "event_timestamp": auditEvent.CreatedAt.Unix(),
                "reason": map[string]interface{}{
                    "reason":      auditEvent.Action,
                    "description": auditEvent.ResourceName,
                },
            },
        },
    }

    // Sign the SET
    setJWT, err := t.SignSET(payload)
    if err != nil {
        return fmt.Errorf("sign SET: %w", err)
    }

    // Push to all matching enabled push streams
    t.mu.RLock()
    streams := make([]*EventStream, 0, len(t.streams))
    for _, s := range t.streams {
        if s.Status == "enabled" &&
           s.DeliveryMethod == "push" &&
           streamMatchesEvent(s, eventURI, auditEvent.ActorID) {
            streams = append(streams, s)
        }
    }
    t.mu.RUnlock()

    for _, stream := range streams {
        if err := t.PushDelivery(ctx, stream, setJWT); err != nil {
            // Log error but don't fail the NATS ack — other streams may succeed
            log.Printf("SSF push to stream %s failed: %v", stream.ID, err)
        }
    }

    return nil
}

// mapAuditEventToSSFURI maps GGID audit actions to SSF event type URIs.
func mapAuditEventToSSFURI(action string) string {
    switch action {
    case "session.revoke", "session.logout":
        return "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
    case "user.password_reset", "user.mfa_enroll", "user.mfa_unenroll":
        return "https://schemas.openid.net/secevent/caep/event-type/credential-change"
    case "user.disable":
        return "https://schemas.openid.net/secevent/risc/event-type/account-disabled"
    case "user.enable":
        return "https://schemas.openid.net/secevent/risc/event-type/account-enabled"
    case "user.delete":
        return "https://schemas.openid.net/secevent/risc/event-type/account-deleted"
    case "user.token_revoke":
        return "https://schemas.openid.net/secevent/risc/event-type/tokens-revoked"
    default:
        return "" // Not a security event
    }
}
```

### 7.3 Poll Delivery Implementation

For poll delivery, the transmitter stores events in a persistent queue and serves them via the poll endpoint.

#### Event Queue (PostgreSQL)

```sql
-- SSF event delivery queue for poll-based streams
CREATE TABLE ssf_event_queue (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id   UUID NOT NULL REFERENCES ssf_streams(id) ON DELETE CASCADE,
    jti         TEXT NOT NULL UNIQUE,          -- SET unique identifier
    set_jwt     TEXT NOT NULL,                  -- The signed SET JWT string
    event_uri   TEXT NOT NULL,                  -- Event type URI
    subject_sub TEXT,                           -- Subject sub (for filtering)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acked_at    TIMESTAMPTZ,                    -- NULL = not yet acknowledged
    deliver_count INT NOT NULL DEFAULT 0        -- Times polled (for TTL)
);

CREATE INDEX idx_ssf_queue_stream_acked
    ON ssf_event_queue(stream_id, acked_at, created_at);
```

#### Poll Delivery Implementation

```go
// PollResult is the response body for GET /ssf/poll.
type PollResult struct {
    Sets         []string `json:"sets"`
    MoreResults  bool     `json:"moreResults"`
    ResultsAfter string   `json:"resultsAfter"`
}

// PollEvents returns a batch of unacknowledged SETs for a poll-based stream.
func (t *SETTransmitter) PollEvents(ctx context.Context, streamID string, resultsAfter string, limit int) (*PollResult, error) {
    if limit <= 0 || limit > 200 {
        limit = 100
    }

    // Query unacknowledged events from the queue
    // Cursor (resultsAfter) is an opaque base64-encoded timestamp
    cursorTime, err := decodeCursor(resultsAfter)
    if err != nil {
        cursorTime = time.Time{} // Start from beginning
    }

    rows, err := t.db.QueryContext(ctx, `
        SELECT set_jwt, created_at
        FROM ssf_event_queue
        WHERE stream_id = $1 AND acked_at IS NULL AND created_at > $2
        ORDER BY created_at ASC
        LIMIT $3
    `, streamID, cursorTime, limit+1) // Fetch limit+1 to check for more
    if err != nil {
        return nil, fmt.Errorf("query event queue: %w", err)
    }
    defer rows.Close()

    var sets []string
    var lastCreatedAt time.Time

    for rows.Next() {
        var setJWT string
        var createdAt time.Time
        if err := rows.Scan(&setJWT, &createdAt); err != nil {
            return nil, err
        }
        sets = append(sets, setJWT)
        lastCreatedAt = createdAt
    }

    // Check if there are more results
    moreResults := len(sets) > limit
    if moreResults {
        sets = sets[:limit] // Trim to requested limit
    }

    // Generate cursor for next poll
    nextCursor := encodeCursor(lastCreatedAt)

    return &PollResult{
        Sets:         sets,
        MoreResults:  moreResults,
        ResultsAfter: nextCursor,
    }, nil
}

// AcknowledgeEvents removes acknowledged SETs from the queue.
func (t *SETTransmitter) AcknowledgeEvents(ctx context.Context, streamID string, jtiList []string) error {
    if len(jtiList) == 0 {
        return nil
    }

    // Mark events as acknowledged
    _, err := t.db.ExecContext(ctx, `
        UPDATE ssf_event_queue
        SET acked_at = NOW()
        WHERE stream_id = $1 AND jti = ANY($2)
    `, streamID, jtiList)
    if err != nil {
        return fmt.Errorf("acknowledge events: %w", err)
    }

    // Optionally delete old acknowledged events (retention cleanup)
    _, _ = t.db.ExecContext(ctx, `
        DELETE FROM ssf_event_queue
        WHERE acked_at IS NOT NULL AND acked_at < NOW() - INTERVAL '24 hours'
    `)

    return nil
}
```

#### Poll Delivery HTTP Handler

```go
// handlePoll handles GET /ssf/poll
func (t *SETTransmitter) handlePoll(w http.ResponseWriter, r *http.Request) {
    streamID := r.URL.Query().Get("stream_id")
    if streamID == "" {
        // stream_id may be in the path: /ssf/poll/{stream_id}
        streamID = chi.URLParam(r, "stream_id")
    }
    resultsAfter := r.URL.Query().Get("resultsAfter")
    limitStr := r.URL.Query().Get("limit")

    limit := 100
    if limitStr != "" {
        if n, err := strconv.Atoi(limitStr); err == nil {
            limit = n
        }
    }

    result, err := t.PollEvents(r.Context(), streamID, resultsAfter, limit)
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "poll failed")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

// handlePollAck handles POST /ssf/poll/ack
func (t *SETTransmitter) handlePollAck(w http.ResponseWriter, r *http.Request) {
    var req struct {
        StreamID string   `json:"stream_id"`
        JTI      []string `json:"jti"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSONError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    if err := t.AcknowledgeEvents(r.Context(), req.StreamID, req.JTI); err != nil {
        writeJSONError(w, http.StatusInternalServerError, "ack failed")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{}"))
}
```

### 7.4 Metadata Discovery Handler

```go
// handleSSFConfiguration handles GET /.well-known/ssf-configuration
func (t *SETTransmitter) handleSSFConfiguration(w http.ResponseWriter, r *http.Request) {
    config := map[string]interface{}{
        "issuer":        t.issuer,
        "jwks_uri":      t.issuer + "/.well-known/jwks.json",
        "delivery_methods_supported": []string{
            "https://schemas.openid.net/secevent/risc/delivery-method/push",
            "https://schemas.openid.net/secevent/risc/delivery-method/poll",
        },
        "push_endpoint":              t.issuer + "/ssf/events",
        "poll_endpoint":              t.issuer + "/ssf/poll",
        "ack_endpoint":               t.issuer + "/ssf/poll/ack",
        "verification_endpoint":      t.issuer + "/ssf/verify",
        "stream_management_endpoint": t.issuer + "/ssf/streams",
        "subject_types_supported":    []string{"iss_sub", "email", "phone", "opaque"},
        "events_supported":           t.supportedEvents(),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(config)
}
```

### 7.5 Stream Management Handler

```go
// handleCreateStream handles POST /ssf/streams
func (t *SETTransmitter) handleCreateStream(w http.ResponseWriter, r *http.Request) {
    var req CreateStreamRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSONError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // Validate delivery method
    if req.Delivery.Method != "push" && req.Delivery.Method != "poll" {
        writeJSONError(w, http.StatusBadRequest, "invalid delivery method")
        return
    }

    // Negotiate subject types
    negotiatedTypes := negotiateSubjectTypes(req.SubjectTypes, t.supportedSubjectTypes())
    if len(negotiatedTypes) == 0 {
        writeJSONError(w, http.StatusBadRequest, "no common subject types")
        return
    }

    // Validate event types
    for _, eventURI := range req.EventsRequested {
        if !t.supportsEvent(eventURI) {
            writeJSONError(w, http.StatusBadRequest,
                fmt.Sprintf("unsupported event type: %s", eventURI))
            return
        }
    }

    stream := &EventStream{
        ID:              "stream-" + uuid.New().String(),
        Description:     req.Description,
        DeliveryMethod:  req.Delivery.Method,
        EndpointURL:     req.Delivery.EndpointURL,
        EventsRequested: req.EventsRequested,
        EventsEnabled:   req.EventsRequested,
        SubjectTypes:    negotiatedTypes,
        SubjectFilters:  req.SubjectFilters,
        Status:          "enabled",
        CreatedAt:       time.Now().UTC(),
        UpdatedAt:       time.Now().UTC(),
    }

    t.mu.Lock()
    t.streams[stream.ID] = stream
    t.mu.Unlock()

    // Persist to database
    if err := t.repo.CreateStream(r.Context(), stream); err != nil {
        writeJSONError(w, http.StatusInternalServerError, "failed to create stream")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(stream)
}
```

---

## 8. GGID as Receiver

GGID can also act as a SET receiver — consuming security events from external transmitters (e.g., a partner IdP, a device management platform, or a threat intelligence service).

### 8.1 Receiver Architecture

```
┌─────────────────────────────────────────────────────┐
│                    GGID Platform                      │
│                                                       │
│  ┌──────────────────────────────────────────────┐   │
│  │              SSF Receiver                      │   │
│  │                                                │   │
│  │  POST /ssf/receive  ◄── External Transmitter  │   │
│  │     (push delivery)                            │   │
│  │                                                │   │
│  │  ┌─────────────┐    ┌──────────────┐          │   │
│  │  │  SET Verify  │──> │  Dedup Check  │          │   │
│  │  │  (signature) │    │  (jti cache)  │          │   │
│  │  └─────────────┘    └──────┬───────┘          │   │
│  │                             │                   │   │
│  │                    ┌────────▼────────┐         │   │
│  │                    │ Event Dispatcher │         │   │
│  │                    └────────┬────────┘         │   │
│  │                             │                   │   │
│  │              ┌──────────────┼──────────────┐   │   │
│  │              │              │              │    │   │
│  │     ┌────────▼───┐  ┌──────▼─────┐  ┌────▼───┐│   │
│  │     │  Session    │  │ Credential │  │ Device ││   │
│  │     │  Revoker    │  │ Invalidator│  │  Trust ││   │
│  │     │ (Redis del) │  │ (JWT block) │  │ Update ││   │
│  │     └────────────┘  └────────────┘  └────────┘│   │
│  └──────────────────────────────────────────────┘   │
│                         │                             │
│                   NATS JetStream                      │
│                 (internal broadcast)                  │
│                         │                             │
│              ┌──────────┼──────────┐                 │
│              │          │          │                  │
│         ┌────▼───┐ ┌───▼────┐ ┌───▼────┐            │
│         │ Audit  │ │ Auth   │ │ Policy │            │
│         │Logger  │ │Service │ │Engine  │            │
│         └────────┘ └────────┘ └────────┘            │
└─────────────────────────────────────────────────────┘
```

### 8.2 Receiver Implementation

#### Core Receiver Type

```go
// SETReceiver validates, deduplicates, and processes incoming SETs.
type SETReceiver struct {
    jwksCache   map[string]*JWKSProvider // keyed by issuer
    dedupCache  DedupCache
    dispatcher  EventDispatcher
    httpClient  *http.Client
    mu          sync.RWMutex
}

// NewSETReceiver creates a new SET receiver.
func NewSETReceiver(dedup DedupCache, dispatcher EventDispatcher) *SETReceiver {
    return &SETReceiver{
        jwksCache:  make(map[string]*JWKSProvider),
        dedupCache: dedup,
        dispatcher: dispatcher,
        httpClient: &http.Client{Timeout: 5 * time.Second},
    }
}

// ReceiveSET processes an incoming pushed SET.
func (r *SETReceiver) ReceiveSET(ctx context.Context, rawSET string) error {
    // Step 1: Parse the JWT (without verifying yet — need kid to find key)
    token, err := jwt.Parse(rawSET, func(token *jwt.Token) (interface{}, error) {
        // Verify typ header
        if typ, ok := token.Header["typ"].(string); !ok || typ != "secevent-jwt" {
            return nil, fmt.Errorf("invalid typ header: expected secevent-jwt")
        }
        // Get signing method
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        // Get key ID
        kid, _ := token.Header["kid"].(string)
        // Get issuer
        claims := token.Claims.(jwt.MapClaims)
        iss, _ := claims["iss"].(string)
        // Fetch JWKS
        return r.getSigningKey(ctx, iss, kid)
    })
    if err != nil {
        if !token.Valid {
            return &SETError{Code: "jwtValidation", Description: err.Error()}
        }
        return &SETError{Code: "jwtParse", Description: err.Error()}
    }

    claims := token.Claims.(jwt.MapClaims)

    // Step 2: Extract jti for dedup
    jti, _ := claims["jti"].(string)
    if jti == "" {
        return &SETError{Code: "badRequest", Description: "missing jti"}
    }

    // Step 3: Check dedup cache
    seen, err := r.dedupCache.CheckAndSet(ctx, jti)
    if err != nil {
        return fmt.Errorf("dedup check: %w", err)
    }
    if seen {
        // Already processed — return nil (caller returns 200 OK idempotently)
        return nil
    }

    // Step 4: Dispatch events
    events, ok := claims["events"].(map[string]interface{})
    if !ok {
        return &SETError{Code: "badRequest", Description: "missing events claim"}
    }

    for eventURI, payload := range events {
        event := &SecurityEvent{
            URI:     eventURI,
            Payload: payload.(map[string]interface{}),
            JTI:     jti,
            ISS:     claims["iss"].(string),
            IAT:     int64(claims["iat"].(float64)),
        }
        if toe, ok := claims["toe"].(float64); ok {
            event.TOE = int64(toe)
        }
        if err := r.dispatcher.Dispatch(ctx, event); err != nil {
            log.Printf("SSF: event dispatch failed for %s: %v", eventURI, err)
            // Don't fail the whole SET — one bad event shouldn't block others
        }
    }

    return nil
}
```

#### Event Dispatcher

```go
// SecurityEvent represents a parsed security event from a SET.
type SecurityEvent struct {
    URI     string
    Payload map[string]interface{}
    JTI     string
    ISS     string
    IAT     int64
    TOE     int64
}

// EventHandler processes a single security event type.
type EventHandler interface {
    Handle(ctx context.Context, event *SecurityEvent) error
}

// EventDispatcher routes events to registered handlers.
type EventDispatcher struct {
    handlers map[string]EventHandler
    mu       sync.RWMutex
}

// NewEventDispatcher creates a new dispatcher with default handlers.
func NewEventDispatcher(redis *redis.Client, js jetstream.JetStream) *EventDispatcher {
    d := &EventDispatcher{
        handlers: make(map[string]EventHandler),
    }
    // Register default handlers
    d.Register("https://schemas.openid.net/secevent/caep/event-type/session-revoked",
        &SessionRevokedHandler{redis: redis})
    d.Register("https://schemas.openid.net/secevent/caep/event-type/credential-change",
        &CredentialChangeHandler{redis: redis, js: js})
    d.Register("https://schemas.openid.net/secevent/caep/event-type/device-compliance-change",
        &DeviceComplianceHandler{redis: redis})
    d.Register("https://schemas.openid.net/secevent/risc/event-type/account-disabled",
        &AccountDisabledHandler{redis: redis, js: js})
    d.Register("https://schemas.openid.net/secevent/risc/event-type/tokens-revoked",
        &TokensRevokedHandler{redis: redis})
    return d
}

func (d *EventDispatcher) Register(eventURI string, handler EventHandler) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.handlers[eventURI] = handler
}

func (d *EventDispatcher) Dispatch(ctx context.Context, event *SecurityEvent) error {
    d.mu.RLock()
    handler, ok := d.handlers[event.URI]
    d.mu.RUnlock()

    if !ok {
        log.Printf("SSF: no handler for event type %s", event.URI)
        return nil // Unknown event — ack and ignore
    }

    return handler.Handle(ctx, event)
}
```

#### Session Revoked Handler

```go
// SessionRevokedHandler revokes a user session in Redis when it receives
// a CAEP session-revoked event.
type SessionRevokedHandler struct {
    redis *redis.Client
}

func (h *SessionRevokedHandler) Handle(ctx context.Context, event *SecurityEvent) error {
    payload := event.Payload

    // Extract subject
    subject, ok := payload["subject"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("missing subject")
    }
    userSub, _ := subject["sub"].(string)

    // Extract session ID (if present)
    session, _ := payload["session"].(map[string]interface{})
    sessionID, _ := session["id"].(string)

    if sessionID != "" {
        // Revoke specific session
        key := fmt.Sprintf("session:%s", sessionID)
        if err := h.redis.Del(ctx, key).Err(); err != nil {
            return fmt.Errorf("revoke session %s: %w", sessionID, err)
        }
        log.Printf("SSF: revoked session %s for user %s", sessionID, userSub)
    } else {
        // No specific session ID — revoke all sessions for the user
        pattern := fmt.Sprintf("session:user:%s:*", userSub)
        iter := h.redis.Scan(ctx, 0, pattern, 100).Iterator()
        var deleted int64
        for iter.Next(ctx) {
            h.redis.Del(ctx, iter.Val())
            deleted++
        }
        log.Printf("SSF: revoked %d sessions for user %s", deleted, userSub)
    }

    // Broadcast to internal NATS for other services to pick up
    // (e.g., gateway can invalidate JWT cache)
    return nil
}
```

#### HTTP Handler for Receiving Pushed SETs

```go
// handleReceive handles POST /ssf/receive
// This is the push delivery endpoint for external transmitters.
func (s *SSFServer) handleReceive(w http.ResponseWriter, r *http.Request) {
    // Verify transport auth (bearer token or mTLS)
    if !s.verifyTransportAuth(r) {
        writeSETError(w, http.StatusUnauthorized, "authReq", "invalid or missing auth")
        return
    }

    // Read SET from body
    body, err := io.ReadAll(io.LimitReader(r.Body, 256*1024)) // 256KB max
    if err != nil {
        writeSETError(w, http.StatusBadRequest, "badRequest", "failed to read body")
        return
    }

    setJWT := strings.TrimSpace(string(body))

    // Process the SET
    if err := s.receiver.ReceiveSET(r.Context(), setJWT); err != nil {
        if setErr, ok := err.(*SETError); ok {
            status := mapSETErrorToHTTPStatus(setErr.Code)
            writeSETError(w, status, setErr.Code, setErr.Description)
            return
        }
        writeSETError(w, http.StatusInternalServerError, "serverError", err.Error())
        return
    }

    // Acknowledge
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte("{}"))
}

func mapSETErrorToHTTPStatus(code string) int {
    switch code {
    case "badRequest", "jwtParse", "jwtTyp":
        return http.StatusBadRequest
    case "jwtAudValidation", "jwtClaimValidation":
        return http.StatusBadRequest
    case "authReq":
        return http.StatusUnauthorized
    case "forbidden":
        return http.StatusForbidden
    case "tooManyRequests":
        return http.StatusTooManyRequests
    default:
        return http.StatusInternalServerError
    }
}

func writeSETError(w http.ResponseWriter, status int, code, description string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{
        "err":         code,
        "description": description,
    })
}
```

### 8.3 External Poll Client

For receiving events via poll (when GGID polls an external transmitter):

```go
// PollClient polls an external transmitter's poll endpoint.
type PollClient struct {
    transmitterURL string
    streamID       string
    accessToken    string
    receiver       *SETReceiver
    pollInterval   time.Duration
    cursor         string // resultsAfter cursor
    httpClient     *http.Client
}

// NewPollClient creates a poll-mode client for an external transmitter.
func NewPollClient(transmitterURL, streamID, accessToken string, receiver *SETReceiver) *PollClient {
    return &PollClient{
        transmitterURL: transmitterURL,
        streamID:       streamID,
        accessToken:    accessToken,
        receiver:       receiver,
        pollInterval:   5 * time.Second,
        httpClient:     &http.Client{Timeout: 30 * time.Second},
    }
}

// Start begins polling in a background goroutine.
func (c *PollClient) Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(c.pollInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                if err := c.pollOnce(ctx); err != nil {
                    log.Printf("SSF poll error: %v", err)
                    // Back off on error
                    time.Sleep(10 * time.Second)
                }
            }
        }
    }()
}

func (c *PollClient) pollOnce(ctx context.Context) error {
    // Build poll URL
    u := fmt.Sprintf("%s/ssf/poll?stream_id=%s", c.transmitterURL, c.streamID)
    if c.cursor != "" {
        u += "&resultsAfter=" + url.QueryEscape(c.cursor)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+c.accessToken)
    req.Header.Set("Accept", "application/secevent+jwt")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("poll request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusTooManyRequests {
        retryAfter := resp.Header.Get("Retry-After")
        if retryAfter != "" {
            if secs, err := strconv.Atoi(retryAfter); err == nil {
                time.Sleep(time.Duration(secs) * time.Second)
            }
        }
        return nil
    }

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("poll returned %d", resp.StatusCode)
    }

    var result PollResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("decode poll response: %w", err)
    }

    // Update cursor
    c.cursor = result.ResultsAfter

    // Process each SET
    var jtiList []string
    for _, setJWT := range result.Sets {
        if err := c.receiver.ReceiveSET(ctx, setJWT); err != nil {
            log.Printf("SSF: failed to process polled SET: %v", err)
            continue // Don't ack failed SETs — they'll be re-delivered
        }
        // Extract jti for acknowledgment
        // (In practice, the receiver stores jti during processing)
        jtiList = append(jtiList, extractJTI(setJWT))
    }

    // Acknowledge processed SETs
    if len(jtiList) > 0 {
        if err := c.acknowledge(ctx, jtiList); err != nil {
            return fmt.Errorf("acknowledge: %w", err)
        }
    }

    // If more results are available, poll again immediately
    if result.MoreResults {
        go c.pollOnce(ctx)
    }

    return nil
}

func (c *PollClient) acknowledge(ctx context.Context, jtiList []string) error {
    ackURL := fmt.Sprintf("%s/ssf/poll/ack", c.transmitterURL)
    body, _ := json.Marshal(map[string]interface{}{
        "stream_id": c.streamID,
        "jti":       jtiList,
    })

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, ackURL, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Authorization", "Bearer "+c.accessToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("ack returned %d", resp.StatusCode)
    }

    return nil
}
```

---

## 9. NATS Integration

GGID's existing NATS JetStream infrastructure serves as the internal event bus. SSF extends event flow to external organizations. The two layers are complementary:

### 9.1 Dual-Flow Architecture

```
                         EXTERNAL                              INTERNAL
                    ═══════════════════                  ═══════════════════

┌─────────────┐                    ┌─────────────────────────────────────────────┐                    ┌──────────────┐
│  External    │   1. Push SET      │                GGID Platform                 │   5. NATS msg     │  Internal     │
│ Transmitter  │ ─────────────────> │                                              │ ────────────────> │  Consumers    │
│ (Partner IdP)│   POST /ssf/receive│  ┌──────────────┐    ┌─────────────────┐   │   audit.events.*  │  (Audit, Auth,│
│              │                    │  │ SSF Receiver  │    │ NATS JetStream  │   │                   │   Gateway)   │
└─────────────┘                    │  │              │──> │                 │   │                    └──────────────┘
                                   │  │ Verify SET    │    │ audit.events.*  │   │
                                   │  │ Dedup jti     │    │                 │   │
                                   │  │ Dispatch      │    └─────────────────┘   │                    ┌──────────────┐
                                   │  └──────────────┘                          │   4. NATS msg     │  External     │
                                   │                    ▲                        │ <──────────────── │  Receivers    │
                                   │                    │                        │                   │  (Partner,    │
                                   │  ┌──────────────┐  │ 3. Publish event        │                   │   SIEM, ZTNA) │
                                   │  │ SSF          │──┘                        │                    └──────────────┘
                                   │  │ Transmitter  │                           │
┌─────────────┐                    │  │              │   2. Push SET             │                    ┌──────────────┐
│  External    │  <────────────────│  │ Sign SET     │ <───────────────────────  │                    │  External    │
│  Receiver    │   POST /ssf/events│  │ Push/Poll    │   NATS consumer fires      │                    │  Poll Client │
│ (SaaS, SIEM) │                   │  └──────────────┘                           │                    └──────────────┘
└─────────────┘                    └─────────────────────────────────────────────┘
```

### 9.2 Outbound Flow (GGID as Transmitter)

```
1. Internal event occurs (e.g., admin revokes session)
   │
   ▼
2. Service publishes to NATS: audit.events.session.revoke
   │  (existing pattern — see services/audit/internal/consumer/nats_consumer.go)
   ▼
3. SSF Transmitter's NATS consumer picks up the message
   │
   ▼
4. Transmitter maps audit action → SSF event URI
   │  (session.revoke → caep:session-revoked)
   ▼
5. Transmitter wraps event in SET, signs with RS256
   │
   ▼
6. Transmitter pushes SET to all matching enabled streams
   │  (POST to receiver endpoints, with retry/backoff)
   ▼
7. Receiver validates SET signature, processes event
```

### 9.3 Inbound Flow (GGID as Receiver)

```
1. External transmitter pushes SET to GGID's /ssf/receive endpoint
   │  (or GGID's poll client retrieves SETs from external poll endpoint)
   ▼
2. Receiver validates SET signature (fetches external JWKS)
   │
   ▼
3. Receiver checks jti dedup (Redis)
   │
   ▼
4. Receiver dispatches event to handler
   │  (session-revoked → Redis DEL session:xxx)
   │  (credential-change → publish to NATS for auth service)
   ▼
5. Handler may publish internal NATS event for broader consumption
   │  (e.g., security.external.session_revoked)
   ▼
6. Internal consumers (gateway, auth, audit) process the event
```

### 9.4 NATS Subject Mapping

GGID's existing NATS subjects can be extended for SSF:

| NATS Subject | Direction | Purpose |
|---|---|---|
| `audit.events.{action}` | Internal | Existing audit event subjects |
| `ssf.outbound.{event_type}` | Internal | Events queued for external transmission |
| `ssf.inbound.{event_type}` | Internal | External events received and broadcast internally |
| `ssf.stream.{stream_id}.delivery` | Internal | Per-stream delivery status tracking |

### 9.5 Backpressure and Reliability

NATS JetStream provides built-in durability and backpressure:

```
┌──────────────────────────────────────────────────────────────────┐
│                     NATS JetStream Configuration                  │
│                                                                   │
│  Stream: AUDIT_EVENTS                                             │
│  Subjects: audit.events.>, ssf.outbound.>, ssf.inbound.>         │
│  Retention: LimitsPolicy                                          │
│  MaxAge: 72h (events expire after 72 hours if not consumed)      │
│  MaxBytes: 1GB                                                    │
│  Storage: FileStorage (durable — survives restart)                │
│                                                                   │
│  Consumer: ssf-push-transmitter                                   │
│  FilterSubject: ssf.outbound.>                                    │
│  AckPolicy: AckExplicit                                           │
│  MaxDeliver: 5 (retry 5 times, then dead-letter)                 │
│  MaxAckPending: 1000 (max 1000 in-flight events)                 │
│                                                                   │
│  Consumer: ssf-receiver-broadcast                                 │
│  FilterSubject: ssf.inbound.>                                     │
│  AckPolicy: AckExplicit                                           │
│  MaxDeliver: 3                                                    │
│  MaxAckPending: 500                                               │
└──────────────────────────────────────────────────────────────────┘
```

This matches GGID's existing `consumer.Config` pattern in `services/audit/internal/consumer/nats_consumer.go`:

```go
cons, err := c.js.CreateOrUpdateConsumer(ctx, c.cfg.StreamName, jetstream.ConsumerConfig{
    Name:          c.cfg.Consumer,
    Durable:       c.cfg.Consumer,
    FilterSubject: c.cfg.Subject,
    AckPolicy:     jetstream.AckExplicitPolicy,
    MaxDeliver:    c.cfg.MaxDeliver,
})
```

---

## 10. Implementation Roadmap

### Phase 1: SET Format + Push Transmitter (Weeks 1-3)

**Goal:** GGID can emit signed CAEP events to external receivers via push delivery.

**Deliverables:**

| Item | Description | Effort |
|---|---|---|
| SET signing module | `pkg/ssf/set.go` — SignSET, SETPayload, SETHeader | 2 days |
| GGID audit → SSF event mapping | Map `AuditEvent.Action` to CAEP/RISC event URIs | 1 day |
| NATS consumer for SSF outbound | New consumer subscribing to `ssf.outbound.>` | 2 days |
| Push delivery engine | PushDelivery, sendWithRetry, dead-letter queue | 3 days |
| JWKS endpoint | `/.well-known/jwks.json` (reuse OIDC keys) | 1 day |
| Metadata discovery | `GET /.well-known/ssf-configuration` | 1 day |
| Integration tests | Mock receiver, SET signature verification, retry logic | 3 days |

**Endpoints added:**

- `GET /.well-known/ssf-configuration`
- `GET /.well-known/jwks.json` (if not already present)

**Success criteria:**

- GGID can emit a `session-revoked` SET when an admin revokes a session
- External receiver validates SET signature using GGID's JWKS
- SETs are delivered with at-least-once semantics (retried on failure)

### Phase 2: Stream Management + Verification (Weeks 4-5)

**Goal:** External partners can create, configure, and verify event streams.

**Deliverables:**

| Item | Description | Effort |
|---|---|---|
| Stream data model | PostgreSQL `ssf_streams` table | 1 day |
| Stream management API | CRUD: POST/GET/PATCH/DELETE `/ssf/streams` | 3 days |
| Verification flow | POST `/ssf/verify` + verification SET emission | 2 days |
| Subject type negotiation | Intersect transmitter/receiver subject types | 1 day |
| Event filtering | Only emit events matching stream's `events_requested` | 1 day |
| Subject filtering | Only emit events for subjects matching `subject_filters` | 1 day |
| API tests | Stream lifecycle, verification, filtering | 2 days |

**Endpoints added:**

- `POST /ssf/streams`
- `GET /ssf/streams/{id}`
- `PATCH /ssf/streams/{id}`
- `DELETE /ssf/streams/{id}`
- `POST /ssf/verify`

**Success criteria:**

- Partner creates a stream requesting `session-revoked` events
- Partner triggers verification, receives verification SET, confirms
- Stream enters `enabled` status and begins receiving real events
- Event and subject filtering work correctly

### Phase 3: Poll Delivery + Receiver Capability (Weeks 6-8)

**Goal:** GGID supports both delivery methods and can receive external SETs.

**Deliverables:**

| Item | Description | Effort |
|---|---|---|
| Event queue (PostgreSQL) | `ssf_event_queue` table with cursor-based pagination | 2 days |
| Poll endpoint | GET `/ssf/poll` with `resultsAfter` cursor and `limit` | 2 days |
| Ack endpoint | POST `/ssf/poll/ack` with `jti` batch | 1 day |
| SET receiver | SETReceiver: verify, dedup, dispatch | 3 days |
| Push receive endpoint | POST `/ssf/receive` with transport auth | 2 days |
| Event handlers | SessionRevokedHandler, CredentialChangeHandler, DeviceComplianceHandler | 3 days |
| External JWKS caching | Fetch and cache transmitter JWKS, refresh on key rotation | 2 days |
| External poll client | PollClient for polling external transmitters | 2 days |
| Integration tests | Full push/poll/receive cycle tests | 3 days |

**Endpoints added:**

- `GET /ssf/poll`
- `POST /ssf/poll/ack`
- `POST /ssf/receive`

**Success criteria:**

- Partner can poll GGID's poll endpoint and receive SET batches
- Partner can acknowledge SETs to remove them from the queue
- GGID can receive pushed SETs from an external transmitter
- Received `session-revoked` events trigger Redis session invalidation
- External JWKS is cached and refreshed correctly

### Phase 4: Subject Negotiation + Multi-Tenant Isolation (Weeks 9-10)

**Goal:** Production-grade multi-tenant support with per-tenant stream isolation.

**Deliverables:**

| Item | Description | Effort |
|---|---|---|
| Multi-subject format support | Emit SETs with different subject types per stream | 2 days |
| Per-tenant stream scoping | Streams are scoped to a tenant; events from other tenants are not delivered | 2 days |
| Tenant-aware subject mapping | Map GGID tenant_id to subject `iss` claim | 1 day |
| Stream rate limiting | Per-stream rate limits to prevent flooding receivers | 1 day |
| Dead-letter management | UI/API for inspecting and replaying failed deliveries | 2 days |
| Metrics + monitoring | Prometheus metrics for SET delivery, ack rates, retry counts | 2 days |
| Documentation | API docs, integration guide for partners | 2 days |
| Security hardening | mTLS support, JWKS rotation, audit logging of stream operations | 2 days |

**Success criteria:**

- Multiple tenants can have independent streams
- Events from tenant A are never delivered to tenant B's streams
- Per-stream rate limits prevent receiver flooding
- Prometheus dashboard shows delivery health
- Partner integration guide is published

### Effort Summary

| Phase | Duration | Team Size | Total Person-Days |
|---|---|---|---|
| Phase 1: Push Transmitter | 3 weeks | 1 dev | 13 |
| Phase 2: Stream Management | 2 weeks | 1 dev | 11 |
| Phase 3: Poll + Receiver | 3 weeks | 1 dev | 20 |
| Phase 4: Multi-Tenant | 2 weeks | 1 dev | 14 |
| **Total** | **10 weeks** | **1 dev** | **58** |

With 2 developers: ~5-6 weeks elapsed.

---

## 11. Comparison with Commercial SSF Implementations

### 11.1 Vendor Comparison Matrix

| Feature | Okta | Microsoft Entra ID | Google Cloud Identity | Ping Identity | **GGID (planned)** |
|---|---|---|---|---|---|
| **CAEP support** | Yes (session-revoked, credential-change) | Yes (CAE — session revocation, conditional access) | Limited (RISC only) | Yes (full CAEP) | Yes (Phase 1) |
| **RISC support** | Yes (account lifecycle) | Partial (via Graph API webhooks) | Yes (account-disabled, tokens-revoked) | Yes (full RISC) | Yes (Phase 1) |
| **Push delivery** | Yes | Yes (via Graph subscriptions) | Yes | Yes | Yes (Phase 1) |
| **Poll delivery** | No | No | No | Yes | Yes (Phase 3) |
| **Stream management** | Yes (via Admin API) | Yes (via Graph API) | Limited | Yes (full SSF API) | Yes (Phase 2) |
| **Verification flow** | Yes | Manual | No | Yes | Yes (Phase 2) |
| **Subject negotiation** | iss_sub, email | email, oid | email | iss_sub, email, opaque | iss_sub, email, phone, opaque (Phase 4) |
| **Custom events** | No | Yes (via Graph change notifications) | No | Yes | Yes |
| **JWKS discovery** | Yes | Yes (via OpenID config) | Yes | Yes | Yes |
| **mTLS support** | No (OAuth only) | No (OAuth only) | No | Yes | Yes (Phase 4) |
| **Multi-tenant** | Yes (per-org streams) | Yes (per-tenant) | Yes | Yes | Yes (Phase 4) |
| **Dead-letter queue** | Internal only | Internal only | Internal only | Yes (API accessible) | Yes (Phase 4) |
| **Pricing model** | Included in Workforce Identity | Included in Entra ID P2 | Included in Cloud Identity Premium | Included in PingFederate | Open source (self-hosted) |

### 11.2 Okta SSF Implementation

**Strengths:**
- Mature CAEP implementation — session-revoked and credential-change are production-ready
- Well-documented stream management API
- Automatic verification flow during stream setup
- Integrated with Okta's event hook system

**Limitations:**
- Push-only (no poll delivery)
- No mTLS support (OAuth bearer only)
- Limited custom event support
- No dead-letter queue visibility for receivers

**Relevant API:** Okta's `/api/v1/securityEvents` and `/api/v1/eventHooks` endpoints implement SSF-compatible event delivery.

### 11.3 Microsoft Entra ID (Azure AD)

**Strengths:**
- Continuous Access Evaluation (CAE) is built into the token validation pipeline
- Real-time session revocation propagates in <1 minute to Microsoft 365 and Azure
- Graph API change notifications support custom event types
- Cross-tenant event sharing via B2B collaboration

**Limitations:**
- CAE is proprietary (not fully SSF-compliant — uses Microsoft's own event format)
- No poll delivery
- No standard SSF metadata discovery endpoint
- Subject identification is Microsoft-specific (OID-based)

**Note:** Microsoft has expressed intent to adopt SSF standards, but as of 2025, their CAE implementation predates and diverges from the OpenID SSF specification.

### 11.4 Google Cloud Identity

**Strengths:**
- RISC implementation for account lifecycle events (account-disabled, tokens-revoked)
- Integrated with Google Workspace security center
- Well-documented JWKS and event verification

**Limitations:**
- No CAEP support (session-level events not available)
- RISC only (account lifecycle events)
- No stream management API (single stream per project)
- No poll delivery
- Limited subject type support (email only)

### 11.5 Ping Identity

**Strengths:**
- Most complete SSF implementation among commercial vendors
- Full CAEP + RISC support
- Both push and poll delivery
- Stream management with subject negotiation
- mTLS and OAuth bearer support
- Dead-letter queue with API access
- Custom event profiles

**Limitations:**
- Enterprise pricing (not accessible for small organizations)
- Proprietary extensions beyond standard SSF
- Complex configuration (reflects full feature set)

### 11.6 GGID's Positioning

GGID's SSF implementation targets the **open-source, self-hosted IAM** market segment. Key differentiators:

| Dimension | GGID | Commercial vendors |
|---|---|---|
| **Deployment** | Self-hosted (Docker, Kubernetes) | SaaS or managed |
| **Pricing** | Free (Apache 2.0) | Per-user licensing |
| **Customization** | Full source code, custom event profiles | Limited or enterprise-only |
| **Data residency** | Full control (events never leave your infrastructure) | Vendor cloud |
| **Compliance** | Self-auditable, SOC2/GDPR ready | Vendor certifications |
| **Integration depth** | Native Go, NATS, gRPC | REST APIs only |
| **Community** | Open source contributors | Vendor engineering teams only |

**Target users:**
- Organizations that need SSF but cannot use SaaS IdPs (data sovereignty, compliance)
- DevSecOps teams building zero-trust architectures
- Open-source identity platforms seeking interoperability
- Research and academic institutions

---

## Appendix A: Complete SET Examples

### A.1 Session Revoked (CAEP)

**Header:**
```json
{
  "typ": "secevent-jwt",
  "alg": "RS256",
  "kid": "ggid-signing-key-2025-01"
}
```

**Payload:**
```json
{
  "jti": "5d54a1f2-9432-4e89-aa33-2488eff87f12",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "toe": 1737379199,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "session": {
        "id": "session:a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "created_at": 1737000000
      },
      "reason": {
        "reason": "admin_revocation",
        "description": "Session revoked by administrator"
      }
    }
  }
}
```

### A.2 Credential Change (CAEP)

```json
{
  "jti": "3c4d5e6f-7890-abcd-ef12-345678901234",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/credential-change": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "credential": {
        "type": "password",
        "change_type": "reset"
      },
      "event_timestamp": 1737379195
    }
  }
}
```

### A.3 Device Compliance Change (CAEP)

```json
{
  "jti": "7e8f9a0b-1234-5678-9abc-def012345678",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "device": {
        "id": "device:550e8400-device-uuid",
        "name": "Jane's iPhone"
      },
      "compliance_status": {
        "compliant": false,
        "reasons": ["jailbroken", "os_version_too_old"]
      },
      "event_timestamp": 1737379198
    }
  }
}
```

### A.4 Account Disabled (RISC)

```json
{
  "jti": "9b8a7f6c-1234-5678-9abc-def012345678",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/risc/event-type/account-disabled": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://idp.ggid.dev",
        "sub": "550e8400-e29b-41d4-a716-446655440000"
      },
      "reason": "security_policy_violation"
    }
  }
}
```

### A.5 Verification Event

```json
{
  "jti": "verify-12345678-1234-5678-abcd-123456789012",
  "iss": "https://idp.ggid.dev",
  "aud": "https://rp.example.com",
  "iat": 1737379200,
  "events": {
    "https://schemas.openid.net/secevent/risc/event-type/verification": {
      "state": "verification-state-random-xyz789"
    }
  }
}
```

---

## Appendix B: Error Response Reference

### B.1 Push Error Responses (RFC 8935)

| Error Code | HTTP Status | Retriable | Description |
|---|---|---|---|
| `badRequest` | 400 | No | Malformed SET or request |
| `jwtParse` | 400 | No | JWT could not be parsed |
| `jwtTyp` | 400 | No | `typ` is not `secevent-jwt` |
| `jwtSig` | 401 | No | Signature verification failed |
| `jwtIssValidation` | 401 | No | `iss` claim invalid |
| `jwtAudValidation` | 400 | No | `aud` claim does not match |
| `jwtReplay` | 400 | No | `jti` already processed (treated as ack) |
| `jwtClaimValidation` | 400 | No | Other claim validation failure |
| `authReq` | 401 | No | Transport auth missing or invalid |
| `forbidden` | 403 | No | Transmitter not authorized |
| `notFound` | 404 | No | Endpoint or stream not found |
| `tooManyRequests` | 429 | Yes | Rate limited |
| `serverError` | 500 | Yes | Internal server error |

### B.2 Poll Error Responses (RFC 8936)

| Error Code | HTTP Status | Retriable | Description |
|---|---|---|---|
| `badRequest` | 400 | No | Invalid cursor or parameters |
| `authReq` | 401 | No | Transport auth invalid |
| `forbidden` | 403 | No | Not authorized for stream |
| `notFound` | 404 | No | Stream deleted |
| `tooManyRequests` | 429 | Yes | Rate limited |
| `serverError` | 500 | Yes | Internal error |

---

## Appendix C: GGID Configuration Reference

### C.1 Environment Variables (Planned)

```bash
# SSF Transmitter Configuration
SSF_ENABLED=true
SSF_ISSUER=https://idp.ggid.dev
SSF_SIGNING_KEY_PATH=/etc/ggid/keys/ssf-signing.key
SSF_SIGNING_KEY_ID=ggid-ssf-2025-01
SSF_JWKS_URI=https://idp.ggid.dev/.well-known/jwks.json

# SSF Push Delivery
SSF_PUSH_MAX_RETRIES=5
SSF_PUSH_INITIAL_BACKOFF_MS=1000
SSF_PUSH_MAX_BACKOFF_MS=60000
SSF_PUSH_TIMEOUT_MS=10000

# SSF Poll Delivery
SSF_POLL_DEFAULT_LIMIT=100
SSF_POLL_QUEUE_TTL_HOURS=24

# SSF Receiver Configuration
SSF_RECEIVER_ENABLED=true
SSF_RECEIVER_DEDUP_TTL_HOURS=24
SSF_RECEIVER_MAX_BODY_BYTES=262144

# SSF Stream Management
SSF_STREAM_MAX_PER_TENANT=50
SSF_STREAM_DEFAULT_RATE_LIMIT=1000  # events per minute
```

### C.2 NATS Configuration (Existing Pattern)

GGID's existing NATS configuration in `services/audit/internal/config/` already supports the JetStream pattern needed for SSF. The SSF transmitter adds a new consumer:

```go
// SSF outbound consumer configuration
ssfConfig := consumer.Config{
    URL:        cfg.NATS.URL,        // Reuse existing NATS connection
    StreamName: "AUDIT_EVENTS",      // Same stream as audit
    Subject:    "ssf.outbound.>",    // New subject for SSF events
    Consumer:   "ssf-push-transmitter",
    MaxDeliver: 5,
    BatchSize:  10,
}
```

### C.3 Database Schema (Planned)

```sql
-- SSF event streams
CREATE TABLE ssf_streams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    description     TEXT,
    delivery_method TEXT NOT NULL CHECK (delivery_method IN ('push', 'poll')),
    endpoint_url    TEXT,                    -- For push delivery
    events_requested JSONB NOT NULL DEFAULT '[]',
    events_enabled   JSONB NOT NULL DEFAULT '[]',
    subject_types   JSONB NOT NULL DEFAULT '[]',
    subject_filters JSONB NOT NULL DEFAULT '[]',
    status          TEXT NOT NULL DEFAULT 'enabled'
                    CHECK (status IN ('enabled', 'paused', 'disabled')),
    push_access_token TEXT,                  -- Encrypted bearer token for push auth
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ssf_streams_tenant ON ssf_streams(tenant_id, status);

-- SSF event delivery queue (for poll delivery)
CREATE TABLE ssf_event_queue (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id   UUID NOT NULL REFERENCES ssf_streams(id) ON DELETE CASCADE,
    jti         TEXT NOT NULL UNIQUE,
    set_jwt     TEXT NOT NULL,
    event_uri   TEXT NOT NULL,
    subject_sub TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acked_at    TIMESTAMPTZ,
    deliver_count INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_ssf_queue_stream_acked
    ON ssf_event_queue(stream_id, acked_at, created_at);

-- SSF delivery audit log (for dead-letter tracking)
CREATE TABLE ssf_delivery_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id   UUID NOT NULL REFERENCES ssf_streams(id) ON DELETE CASCADE,
    jti         TEXT NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('delivered', 'failed', 'dead_letter')),
    http_status INT,
    error_code  TEXT,
    attempts    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ssf_delivery_log_stream_status
    ON ssf_delivery_log(stream_id, status, created_at);
```

---

## Appendix D: Testing Strategy

### D.1 Unit Tests

| Component | Test Cases |
|---|---|
| `SignSET` | Valid payload, missing jti (auto-generate), missing iat (auto-fill), signing error |
| `mapAuditEventToSSFURI` | All known actions, unknown action (returns empty) |
| `PushDelivery` | 200 OK, 400 (non-retriable), 429 (retriable), 500 (retriable), timeout |
| `sendWithRetry` | Success on first try, success after retry, max retries exhausted, context cancel |
| `PollEvents` | Empty queue, partial batch, full batch, moreResults flag, cursor decode error |
| `AcknowledgeEvents` | All acked, partial ack, already acked, unknown jti |
| `ReceiveSET` | Valid SET, invalid typ, invalid signature, missing jti, dedup hit, dispatch error |
| `SessionRevokedHandler` | Specific session ID, no session ID (all sessions), Redis error |
| `negotiateSubjectTypes` | Full overlap, partial overlap, no overlap (fail), empty inputs |

### D.2 Integration Tests

| Test | Description |
|---|---|
| End-to-end push delivery | NATS event → SET → push to mock receiver → 200 OK → verify SET content |
| End-to-end poll delivery | NATS event → SET → poll → ack → verify queue cleared |
| Verification flow | Create stream → verify → receive verification SET → confirm state match |
| Stream lifecycle | Create → update events → pause → resume → delete |
| Retry on failure | Mock receiver returns 500 → verify retry with backoff → eventual success |
| Dedup on retry | Mock receiver returns 500, then 200 on retry → verify handler called once |
| Subject filtering | Two streams with different subject filters → verify correct delivery |
| Multi-event SET | SET with 2 events → verify both dispatched |
| External JWKS caching | Fetch JWKS → cache hit → cache miss → key rotation |
| Rate limiting | Push 100 events rapidly → verify 429 handling and Retry-After |

### D.3 Mock Receiver (Go)

```go
// MockSETReceiver is a test HTTP server that receives SETs.
type MockSETReceiver struct {
    server     *httptest.Server
    received   []string           // Received SET JWTs
    jtiSeen    map[string]bool    // Dedup tracking
    statusCode int                // Status to return (configurable)
    mu         sync.Mutex
}

func NewMockSETReceiver(statusCode int) *MockSETReceiver {
    m := &MockSETReceiver{
        jtiSeen:    make(map[string]bool),
        statusCode: statusCode,
    }
    m.server = httptest.NewServer(http.HandlerFunc(m.handle))
    return m
}

func (m *MockSETReceiver) handle(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    setJWT := string(body)

    m.mu.Lock()
    defer m.mu.Unlock()
    m.received = append(m.received, setJWT)

    // Extract jti for dedup
    // (In production, this would be done after signature verification)
    jti := extractJTI(setJWT)
    if m.jtiSeen[jti] {
        w.WriteHeader(m.statusCode) // Return configured status even for dup
        return
    }
    m.jtiSeen[jti] = true

    w.WriteHeader(m.statusCode)
    w.Write([]byte("{}"))
}

func (m *MockSETReceiver) URL() string  { return m.server.URL }
func (m *MockSETReceiver) Close()       { m.server.Close() }
func (m *MockSETReceiver) ReceivedCount() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    return len(m.received)
}
```

---

## References

| # | Specification | URL |
|---|---|---|
| 1 | RFC 8417 — Security Event Token (SET) | https://www.rfc-editor.org/rfc/rfc8417 |
| 2 | RFC 8935 — Push-Based SET Delivery Using HTTP | https://www.rfc-editor.org/rfc/rfc8935 |
| 3 | RFC 8936 — Poll-Based SET Delivery Using HTTP | https://www.rfc-editor.org/rfc/rfc8936 |
| 4 | OpenID Shared Signals Framework 1.0 | https://openid.net/specs/openid-sharedsignals-framework-1_0.html |
| 5 | OpenID CAEP Specification 1.0 | https://openid.net/specs/openid-caep-spec-1_0.html |
| 6 | OpenID CAEP Interoperability Profile 1.0 | https://openid.net/specs/openid-caep-interoperability-profile-1_0.html |
| 7 | RFC 7519 — JSON Web Token (JWT) | https://www.rfc-editor.org/rfc/rfc7519 |
| 8 | RFC 7517 — JSON Web Key (JWK) | https://www.rfc-editor.org/rfc/rfc7517 |
| 9 | GGID CAEP Analysis | ./caep-analysis.md |
| 10 | GGID Audit Service Source | `services/audit/` |
| 11 | GGID Gateway Middleware | `services/gateway/internal/middleware/` |

---

## Glossary

| Term | Definition |
|---|---|
| **SET** | Security Event Token — a JWT that carries security event data (RFC 8417) |
| **SSF** | Shared Signals Framework — OpenID specification profiling SET for OIDC ecosystems |
| **CAEP** | Continuous Access Evaluation Protocol — event profile for session/access evaluation |
| **RISC** | Risk and Incident Security Claims — event profile for account lifecycle |
| **Transmitter** | The entity that emits SETs (typically an IdP or security service) |
| **Receiver** | The entity that consumes SETs (typically an RP, SIEM, or zero-trust broker) |
| **Event Stream** | A configured relationship between a transmitter and receiver, defining which events are delivered |
| **Push Delivery** | Transmitter sends SETs to receiver via HTTP POST (RFC 8935) |
| **Poll Delivery** | Receiver retrieves SET batches from transmitter via HTTP GET (RFC 8936) |
| **jti** | JWT Token Identifier — unique ID for dedup in SET processing |
| **JWKS** | JSON Web Key Set — public keys for SET signature verification |
| **Subject Principal** | The entity a security event applies to (user, device, session) |
| **Subject Filter** | Stream configuration restricting which subjects' events are delivered |
| **Verification Flow** | Test event exchange to confirm stream delivery works before going live |
| **Dead-Letter Queue** | Persistent storage for SETs that exhausted delivery retries |
| **Cursor** | Opaque token for paginating poll-based event retrieval |
| **Backpressure** | Mechanism for handling receiver overload (429, rate limiting, queue depth) |

---

*End of document.*
