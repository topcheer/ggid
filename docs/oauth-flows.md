# OAuth 2.1 Flows Reference

Quick-reference card for all OAuth 2.1 grant types with sequence diagrams
and error scenarios.

> **See also**: [OAuth Flows Guide](oauth-flows-guide.md) for detailed
> configuration and code examples, [Token Lifecycle](token-lifecycle.md)
> for rotation and introspection.

---

## Flow Selection

```
User involved?
├── No → Client Credentials (M2M)
└── Yes
    ├── Input-constrained device (TV, IoT)? → Device Code
    ├── Back-channel only (banking)? → CIBA
    └── Standard browser/mobile? → Auth Code + PKCE

Need to delegate token downstream? → Token Exchange (RFC 8693)
```

---

## Authorization Code + PKCE

```
Client                         Auth Server           Browser
  │ 1. Generate code_verifier       │                    │
  │    code_challenge               │                    │
  │                                 │                    │
  │ 2. GET /authorize               │                    │
  │    ?response_type=code          │                    │
  │    &code_challenge=...          │                    │
  │    &code_challenge_method=S256  │                    │
  ├────────────────────────────────►│                    │
  │                                 │ 3. Show login      │
  │                                 │◄───────────────────┤
  │ 4. Redirect ?code=...           │                    │
  │◄────────────────────────────────┤                    │
  │                                 │                    │
  │ 5. POST /token                  │                    │
  │    code + code_verifier         │                    │
  ├────────────────────────────────►│                    │
  │ 6. access_token + refresh_token │                    │
  │◄────────────────────────────────┤                    │
```

### Error Scenarios

| Error | When | HTTP |
|-------|------|------|
| `invalid_request` | Missing `code_challenge` (PKCE required) | 400 |
| `invalid_grant` | Code expired or already used | 400 |
| `invalid_grant` | `code_verifier` doesn't match `code_challenge` | 400 |
| `access_denied` | User denied consent | 400 |
| `mismatching_state` | `state` parameter doesn't match | 400 |

---

## Client Credentials

```
Service A                     Auth Server
  │ POST /token                    │
  │ grant_type=client_credentials  │
  │ client_id + client_secret      │
  ├───────────────────────────────►│
  │ access_token (no refresh)      │
  │◄───────────────────────────────┤
```

### Error Scenarios

| Error | When |
|-------|------|
| `invalid_client` | Wrong client_secret |
| `invalid_scope` | Requested scope not allowed for client |

---

## Device Flow (RFC 8628)

```
Device              Auth Server           User's Phone
  │ POST /device/code    │                    │
  ├─────────────────────►│                    │
  │ device_code          │                    │
  │ user_code: ABCD-1234 │                    │
  │ verify_uri           │                    │
  │◄─────────────────────┤                    │
  │                      │                    │
  │ Display code         │                    │
  │                      │ User visits URI    │
  │                      │ enters code        │
  │                      │◄───────────────────┤
  │                      │ User authenticates │
  │                      │                    │
  │ POST /token (poll)   │                    │
  ├─────────────────────►│                    │
  │ authorization_pending│                    │
  │◄─────────────────────┤                    │
  │                      │                    │
  │ POST /token (poll)   │                    │
  ├─────────────────────►│                    │
  │ access_token         │                    │
  │◄─────────────────────┤                    │
```

### Polling Errors

| Error | Action |
|-------|--------|
| `authorization_pending` | Wait `interval` seconds, retry |
| `slow_down` | Increase interval by 5s |
| `access_denied` | User denied — stop polling |
| `expired_token` | Device code expired — restart flow |

---

## CIBA (RFC 9101)

```
App                 Auth Server           User Device
  │ POST /bc-authorize   │                    │
  │ login_hint=user@...  │                    │
  │ binding_message=...  │                    │
  ├─────────────────────►│ Push notification  │
  │ auth_req_id          ├───────────────────►│
  │◄─────────────────────┤                    │
  │                      │ User approves      │
  │                      │◄───────────────────┤
  │ POST /token (poll)   │                    │
  ├─────────────────────►│                    │
  │ pending              │                    │
  │◄─────────────────────┤                    │
  │ POST /token (poll)   │                    │
  ├─────────────────────►│                    │
  │ access_token         │                    │
  │◄─────────────────────┤                    │
```

### Error Scenarios

| Error | When |
|-------|------|
| `authorization_pending` | User hasn't responded |
| `authorization_expired` | auth_req_id expired |
| `access_denied` | User rejected |
| `expired_token` | auth_req_id used after expiry |

---

## Token Exchange (RFC 8693)

```
API Gateway              Auth Server            Downstream API
  │ Has token (aud=gw)       │                      │
  │ POST /token              │                      │
  │ grant_type=token-exchange│                      │
  │ subject_token=jwt        │                      │
  │ audience=downstream-api  │                      │
  │ scope=read-only          │                      │
  ├─────────────────────────►│                      │
  │ New token (aud=ds_api)   │                      │
  │ scope=read-only          │                      │
  │ act={sub:gw_client}      │                      │
  │◄─────────────────────────┤                      │
  │                          │                      │
  │ API call with new token  │                      │
  ├──────────────────────────────────────────────────►│
```

### Delegation Chain

```json
{
  "sub": "user-123",
  "aud": "billing-service",
  "act": {
    "sub": "api-gateway"
  }
}
```

### Error Scenarios

| Error | When |
|-------|------|
| `invalid_grant` | subject_token invalid or expired |
| `invalid_target` | audience not in allowed list |
| `invalid_scope` | Requested scope exceeds subject scope |

---

## Refresh Token Rotation

```
RT-A → (rotate) → RT-B → (rotate) → RT-C
  │
  └── Reuse RT-A → DETECTED → Revoke entire family
```

### Error Scenarios

| Error | When |
|-------|------|
| `invalid_grant` | Refresh token expired or revoked |
| `invalid_grant` | Reuse detected — all tokens in family revoked |
| `invalid_grant` | Client ID mismatch |

---

## PAR (RFC 9126)

Push parameters to server via back-channel, redirect with short request_uri.

```
Client                 Auth Server
  │ POST /par               │
  │ (all auth params)       │
  ├────────────────────────►│
  │ request_uri             │
  │◄────────────────────────┤
  │                         │
  │ Redirect: /authorize    │
  │   ?request_uri=...      │
  ├────────────────────────►│
  │ Normal auth flow        │
```

**Benefit**: Prevents URL tampering, parameter injection, and oversized
URLs for Verifiable Credentials.
