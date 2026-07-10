# OIDC Logout Specification Analysis

> Research document analyzing OIDC logout mechanisms and GGID's implementation gaps.
> Date: 2025-01-20 · Status: Research

---

## 1. Overview

In a single-app world, logout is simple: destroy the server session. In an OIDC
federated world, the user holds sessions at **multiple Relying Parties (RPs)** and at
the **OpenID Provider (OP)**. Logging out of one does not log out of others.

The OIDC logout problem has three dimensions:

1. **OP-initiated** — the OP starts the logout (e.g., admin forces logout).
2. **RP-initiated** — an RP requests logout via redirect to the OP.
3. **Cross-domain propagation** — how does logout propagate to all RPs?

The OIDC Foundation defines three specifications to address these:

| Spec | Status | Transport | Reliability |
|------|--------|-----------|------------|
| RP-Initiated Logout 1.0 | **Final** | Browser redirect | Medium |
| Back-Channel Logout 1.0 | **Final** (errata set 1) | Server-to-server POST | High |
| Front-Channel Logout 1.0 | **Final** | Browser iframes | Low |

All three are final specifications (not drafts). Back-Channel is the recommended
mechanism for production multi-RP deployments; Front-Channel is a fallback for
RPs that cannot expose a server-to-server endpoint.

---

## 2. RP-Initiated Logout (OIDC RP-Initiated Logout 1.0)

**Spec:** openid-connect-rpinitiated-1_0.html — **Final**

### Flow

The RP redirects the user's browser to the OP's `end_session_endpoint`. The OP
validates the request, destroys the OP-side session, optionally shows a logout
confirmation page, then redirects back to the RP's pre-registered
`post_logout_redirect_uri`.

### Request Parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `id_token_hint` | Recommended | ID Token previously issued to the RP; lets OP identify the session |
| `post_logout_redirect_uri` | Optional | Where to redirect after logout; must be pre-registered |
| `state` | Optional | OIDC state value passed through for CSRF protection |
| `ui_locales` | Optional | Preferred locales for the logout confirmation page |
| `client_id` | Optional | Identifies the RP when `id_token_hint` is absent |

### Security Considerations

- `post_logout_redirect_uri` **must be pre-registered** during client registration.
  The OP must validate it against the allow-list before redirecting.
- If `id_token_hint` is absent, the OP **should** show a confirmation page to
  prevent logout CSRF (an attacker forcing logout).
- `state` mitigates logout CSRF when no confirmation page is shown.

### Sequence Diagram

```
  User      RP Browser      OP (end_session_endpoint)
   |            |                    |
   | Click      |  302 Redirect:     |
   | logout     |  /oauth/end_session|
   |----------->|  ?id_token_hint=.. |
   |            |  &post_logout=...  |
   |            |------------------->|
   |            |                    | Validate + destroy
   |            |  302 Redirect to   |
   |            |  post_logout_uri   |
   |            |<-------------------|
   |            | RP destroys local  |
   |<-----------| session + cookies  |
```

### GGID Status: NOT IMPLEMENTED

GGID's `GetDiscoveryConfig()` (oauth_service.go:304) does **not** advertise an
`end_session_endpoint`. No `/oauth/end_session` or equivalent route exists in the
OAuth server. Users cannot initiate a spec-compliant RP-initiated logout.

---

## 3. Back-Channel Logout 1.0

**Spec:** openid-connect-backchannel-1_0.html — **Final** (errata set 1)

### Flow

When the OP destroys a session (via RP-initiated logout, session timeout, or
admin action), the OP sends an HTTP POST to each affected RP's
`backchannel_logout_uri`. This communication happens **server-to-server** — the
browser is not involved.

### Logout Token (JWT)

The logout token is a special JWT with these claims:

```json
{
  "iss": "https://oauth.ggid.dev",
  "aud": "client-app-123",
  "iat": 1737400000,
  "jti": "unique-token-id-abc123",
  "sub": "user-uuid-456",
  "sid": "session-uuid-789",
  "events": {
    "http://schemas.openid.net/event/backchannel-logout": {}
  }
}
```

**Mandatory claims:** `iss`, `aud`, `iat`, `jti`, `events`
**At least one of:** `sub` or `sid`
**Must NOT contain:** `nonce`

### RP Validation Steps

1. Verify JWT signature using OP's published JWKS.
2. Verify `iss` matches the expected issuer.
3. Verify `aud` matches the RP's `client_id`.
4. Verify `events` contains `http://schemas.openid.net/event/backchannel-logout`.
5. Confirm `nonce` claim is **absent**.
6. Check `jti` for replay protection (track seen JTI values).
7. Destroy the local session for the identified `sub`/`sid`.

### Response Codes

| Code | Meaning | OP Action |
|------|---------|-----------|
| 200 | Session destroyed | Done, no retry |
| 4xx | Token invalid or error | Log and stop |
| 5xx | Transient failure | Retry with backoff |

### Sequence Diagram

```
  User   RP Browser   OP          RP-A         RP-B         RP-C
   |         |          |            |            |            |
   | Logout  |--302---->|            |            |            |
   |         |          |--POST----->|            |            |
   |         |          | logout_tok |            |            |
   |         |          |       Destroy session   |            |
   |         |          |<--200 OK---|            |            |
   |         |          |            |            |            |
   |         |          |--POST------------------>|            |
   |         |          | logout_tok              |            |
   |         |          |                       Destroy session |
   |         |          |<-------200 OK----------|            |
   |         |          |--POST------------------------------->|
   |         |          | logout_tok                           |
   |         |          |                                     Destroy
   |         |          |<-------------------200 OK-----------|
   |         |<--302----|                                     |
```

### Benefits

- **Reliable**: server-to-server with retry on failure (5xx → backoff retry).
- **Cross-domain**: no browser involvement, no CORS/cookie issues.
- **No timing dependency**: RPs can process asynchronously.

### Challenges

- RP must expose a publicly reachable HTTP endpoint.
- NAT/firewall may block inbound traffic (common for on-prem apps).
- Requires RP to implement JTI replay cache (distributed state).
- OP must track all active sessions and their RP associations.

### GGID Status: PARTIAL — Skeleton Only

GGID has `/api/v1/oauth/backchannel-logout` that accepts `logout_token` and calls
`ParseBackchannelLogoutToken()`. However:

- **No signature verification**: `ParseUnverified` is used (security risk).
- **No RP notification**: `BackchannelLogout()` only marks an in-memory
  `sync.Map`; it does **not** POST logout tokens to registered RP endpoints.
- **No `backchannel_logout_uri`** field in client registration model.
- **No session-to-RP mapping**: cannot determine which RPs have active sessions.

---

## 4. Front-Channel Logout 1.0

**Spec:** openid-connect-frontchannel-1_0.html — **Final**

### Flow

The OP renders an HTML page containing hidden `<iframe>` elements, one per RP.
Each iframe loads the RP's `frontchannel_logout_uri`. The RP's logout page (loaded
in the iframe) executes JavaScript to destroy the session cookie.

### Iframe Transport

The OP renders HTML with one hidden iframe per RP:

```html
<iframe src="https://rp-a.example.com/logout?frontchannel=true"
        style="display:none"></iframe>
<iframe src="https://rp-b.example.com/logout?frontchannel=true"
        style="display:none"></iframe>
<script>
  setTimeout(() => window.location.href = postLogoutUri, 5000);
</script>
```

### Sequence Diagram

```
  User   RP Browser   OP           RP-A         RP-B
   |         |          |             |            |
   | Logout  |--302---->|             |            |
   |         |<--200+HTML(2 iframes)  |            |
   |         |          |             |            |
   |         |--GET iframe----------------------->|
   |         |          |             | Clear cookie|
   |         |--GET iframe----------->|            |
   |         |          |             | Clear cookie|
   |         |          |             |            |
   |         | All iframes loaded     |            |
   |         | Redirect to post_logout_redirect    |
   |<--------|          |             |            |
```

### Benefits

- **Works through NAT/firewall**: browser-initiated, no inbound to RP needed.
- **Simple RP implementation**: just serve a logout page that clears cookies.

### Challenges

- **Iframe blocking**: `X-Frame-Options: DENY` or `Content-Security-Policy:
  frame-ancestors` on the RP logout page blocks the iframe entirely.
- **Timing unreliable**: if an iframe fails to load (network, CSP), the session
  cookie may not be cleared. The OP cannot detect this.
- **Third-party cookie restrictions**: if the RP's logout cookie is SameSite=Lax
  or SameSite=Strict, the iframe request won't include it.
- **Browser throttling**: modern browsers throttle/throttle background iframes.

### Security Considerations

- RP logout page should be a dedicated path with no session-dependent content.
- The `frontchannel_logout_uri` must **not** require authentication.
- Consider `<iframe sandbox="allow-scripts">` to limit RP page access.

### GGID Status: NOT IMPLEMENTED

No front-channel logout code exists anywhere in GGID. No
`frontchannel_logout_uri` field in client registration. No iframe-rendering
endpoint in the OAuth service.

---

## 5. Token Revocation + Session Cleanup

A complete logout strategy combines all mechanisms plus token revocation (RFC 7009).

### Full Logout Sequence

```
  User   RP    OP        Gateway/Redis   RP-A (BC)   RP-B (FC)
   |      |     |             |              |            |
   | 1.Logout  |             |              |            |
   |----->| 2.302 end_session|              |            |
   |      |---->|            |              |            |
   |      |     | 3.DEL session----------->  |            |
   |      |     | 4.POST logout_token------->|            |
   |      |     |<---200 OK (session destroyed)|          |
   |      |     | 5.Render iframes for FC RPs   |        |
   |      |<----| HTML+iframe-----------------------------  |
   |      |     |                          |  Clear cookie |
   |      |     | 6.Revoke access+refresh tokens (RFC 7009)|
   |      |     | 7.302 post_logout_redirect_uri           |
   |<-----|     |             |              |            |
```

### Step Breakdown

| Step | Mechanism | What Happens |
|------|-----------|-------------|
| 1 | RP-Initiated | User clicks logout; RP redirects to OP |
| 2 | OP Session | OP destroys its own session (Redis DEL) |
| 3 | Back-Channel | OP POSTs logout_token to each RP's back-channel URI |
| 4 | Front-Channel | OP renders iframes for RPs without back-channel support |
| 5 | Token Revocation | OP revokes access_token + refresh_token (RFC 7009) |
| 6 | Redirect | OP redirects user to `post_logout_redirect_uri` |

### Failure Modes

| Failure | Impact | Mitigation |
|---------|--------|------------|
| RP back-channel endpoint unreachable | RP session survives | Retry with exponential backoff; queue in NATS |
| Front-channel iframe blocked by CSP | RP session survives | Log warning; rely on back-channel as primary |
| Token revocation fails | Tokens remain valid until expiry | In-memory revocation + Redis persistent store |
| Redis connection lost | Sessions may survive longer | TTL on session keys (eventual expiry) |
| User closes browser before redirect | OP session destroyed, RP may survive | Back-channel + token TTL as safety net |

---

## 6. GGID Gap Analysis

| Feature | Spec Requirement | GGID Status | Effort |
|---------|-----------------|-------------|--------|
| `end_session_endpoint` | RP-Initiated Logout 1.0 | **Missing** — not in discovery, no route | Low (2-3 days) |
| `post_logout_redirect_uri` validation | RP-Initiated Logout 1.0 | **Missing** — no registration field | Low (1 day) |
| `id_token_hint` validation | RP-Initiated Logout 1.0 | **Missing** — no handler | Low (1 day) |
| `backchannel_logout_uri` | Back-Channel Logout 1.0 | **Missing** — no client registration field | Low (1 day) |
| Logout token signature verification | Back-Channel Logout 1.0 | **Missing** — uses `ParseUnverified` | Medium (1-2 days) |
| POST logout_token to RPs | Back-Channel Logout 1.0 | **Missing** — only marks in-memory flag | Medium (2-3 days) |
| Logout token JTI replay cache | Back-Channel Logout 1.0 | **Missing** — no replay protection | Low (1 day) |
| `frontchannel_logout_uri` | Front-Channel Logout 1.0 | **Missing** — no registration, no iframe rendering | Medium (2-3 days) |
| Session-to-RP mapping | All logout specs | **Missing** — no tracking of which RPs have active sessions per user | Medium (2-3 days) |
| Token revocation (RFC 7009) | RFC 7009 | **Implemented** — `/oauth/revoke` works | Done |
| Access token revocation check | RFC 7009 | **Partial** — in-memory `sync.Map`; not Redis-backed | Low (1 day) |
| Refresh token revocation | RFC 7009 | **Implemented** — `RevokeRefreshToken` + `RevokeAllRefreshTokens` in pg_repo | Done |
| Gateway session revocation | Session management | **Implemented** — Redis-backed `SessionManager` with revoke/list | Done |
| Persistent revocation store | All logout specs | **Missing** — `revokedTokens` and `backchannelLogoutList` are in-memory `sync.Map`, lost on restart | Medium (1-2 days) |

### Key Gaps Summary

1. **No RP-Initiated Logout endpoint** — the most user-facing gap. Users cannot
   log out from the OP via a redirect. This is Phase 1 priority.

2. **Back-channel is a skeleton** — `ParseBackchannelLogoutToken` exists but
   parses unverified tokens. `BackchannelLogout` only stores a flag in memory.
   No actual RP notification occurs.

3. **No session tracking** — GGID cannot determine which RPs have active
   sessions for a given user. Without this, back-channel notification has no
   targets.

4. **In-memory revocation** — `revokedTokens` (sync.Map) and
   `backchannelLogoutList` (sync.Map) are lost on process restart. Production
   needs Redis or database-backed persistence.

---

## 7. Roadmap

### Phase 1: RP-Initiated Logout (1-2 weeks)

- Add `end_session_endpoint` to OIDC discovery config.
- Add `post_logout_redirect_uri` field to client registration model + DB migration.
- Implement `/oauth/end_session` handler: validate `id_token_hint`, destroy OP
  session via Gateway's Redis `SessionManager`, redirect to
  `post_logout_redirect_uri`.
- Add `state` parameter passthrough for CSRF protection.

### Phase 2: Token Revocation Hardening (1 week)

- Move `revokedTokens` from in-memory `sync.Map` to Redis (survive restarts).
- On RP-Initiated logout, automatically revoke all access/refresh tokens for the
  user via existing `RevokeAllRefreshTokens`.
- Add introspection check: `IsTokenRevoked` should query Redis.

### Phase 3: Back-Channel Logout (2-3 weeks)

- Add `backchannel_logout_uri` to client registration model.
- Add `sid` (session ID) tracking: issue `sid` in ID Token, track in DB.
- Implement session-to-RP mapping table: `oidc_sessions(user_id, client_id, sid)`.
- Fix `ParseBackchannelLogoutToken` to verify JWT signature using JWKS.
- Implement outbound POST to each RP's `backchannel_logout_uri` with retry via
  NATS JetStream.
- Add JTI replay cache in Redis.

### Phase 4: Front-Channel Logout (1-2 weeks)

- Add `frontchannel_logout_uri` to client registration model.
- Implement iframe-rendering endpoint: returns HTML with one iframe per RP that
  has front-channel but no back-channel support.
- Short timeout (3-5s) then redirect to `post_logout_redirect_uri`.

### Effort Summary

| Phase | Feature | Duration | Priority |
|-------|---------|----------|----------|
| 1 | RP-Initiated Logout | 1-2 weeks | P0 |
| 2 | Token Revocation Hardening | 1 week | P0 |
| 3 | Back-Channel Logout | 2-3 weeks | P1 |
| 4 | Front-Channel Logout | 1-2 weeks | P2 |

Phases 1 and 2 deliver immediate user-facing value. Phase 3 is the most
impactful for multi-RP deployments. Phase 4 is a fallback for RPs that cannot
expose server-to-server endpoints.
