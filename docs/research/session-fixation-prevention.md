# Session Fixation Prevention in GGID

> Research document analyzing session fixation attack vectors across OAuth, SAML, and OIDC,
> with a security audit of GGID's current implementation and an implementation roadmap.

## 1. Overview

Session fixation is a class of attacks where an attacker forces a known session
identifier onto a victim **before** the victim authenticates. If the server does
not regenerate the session ID after successful authentication, the attacker's
pre-auth session ID becomes the victim's authenticated session ID. The attacker
then uses that same ID to access the victim's authenticated session.

**Attack flow (simplified):**

```
Attacker → Server: GET / (receives session_id=abc123)
Attacker → Victim:  Sends link with session_id=abc123
Victim  → Server:   POST /login (credentials valid, session NOT regenerated)
Victim  → Server:   Session abc123 now authenticated as victim
Attacker → Server:  Uses session_id=abc123 → authenticated as victim
```

**Root cause:** The session ID is not regenerated (rotated) after authentication.
A pre-auth session identifier persists into the post-auth session.

Session fixation extends beyond traditional web sessions:

- **OAuth state** — if the `state` parameter is predictable or reusable, an
  attacker can forge a callback.
- **SAML InResponseTo** — if the assertion does not bind to the original
  AuthnRequest, unsolicited assertions can hijack sessions.
- **OIDC nonce** — if the `nonce` is absent or not validated, ID tokens can be
  replayed to impersonate a user.

This document examines each protocol's fixation surface and audits GGID's code.

## 2. Attack Vectors

### 2.1 Cookie-Based Fixation

```
1. Attacker visits GGID → receives session cookie: ggid_session=xyz
2. Attacker injects cookie into victim's browser via:
   - XSS payload: document.cookie = "ggid_session=xyz; path=/"
   - Network MITM (no HTTPS): Set-Cookie header injection
   - Social engineering: "Click this link to log in"
3. Victim authenticates → server reuses session ID xyz (FIXATION!)
4. Attacker uses session ID xyz → now authenticated as victim
```

**Prerequisites:**
- Server must accept a client-supplied session ID without regeneration.
- Session cookie must not have HttpOnly (enables XSS injection).
- No HTTPS (enables MITM cookie injection).

### 2.2 URL-Based Fixation

Some frameworks accept session IDs in URL parameters:

```
https://app.ggid.io/login?session_id=abc123
```

An attacker sends the victim a link containing the attacker's session ID.
The victim authenticates, and the server binds the session ID from the URL.
This vector requires explicit server-side support for URL session transport,
which is rare in modern frameworks but historically dangerous (logs, referrer
leakage).

### 2.3 SAML Fixation

SAML assertions carry a `SessionIndex` attribute. If this index is predictable
or not bound to the AuthnRequest:

```
1. Attacker initiates SAML SSO → obtains assertion with SessionIndex=S1
2. Attacker replays the assertion or injects SessionIndex=S1 into victim's flow
3. Victim's session is bound to SessionIndex=S1
4. Attacker uses S1 to access victim's session
```

This is mitigated by validating `InResponseTo` (binding the assertion to the
original request) and by assertion replay detection.

### 2.4 OAuth State Fixation

The OAuth `state` parameter prevents CSRF, but if it is predictable:

```
1. Attacker predicts or sets state=st_fixed for victim's authorize request
2. Attacker initiates their own authorization flow with state=st_fixed
3. Both flows use the same state → callback confusion
4. Attacker's authorization code may be injected into victim's session
```

**Mitigation:** `state` must be cryptographically random, single-use, and
server-validated per request.

## 3. Defense: Session ID Regeneration

### Principle

- **Always** generate a new session ID after successful authentication.
- **Never** reuse a pre-auth session ID for the post-auth session.
- **Invalidate** the pre-auth session before issuing the authenticated one.

### Reference Implementation

```go
func (s *AuthService) Login(ctx context.Context, creds Credentials) (*Session, error) {
	// 1. Validate credentials...

	// 2. CRITICAL: Invalidate any pre-auth session
	if oldSessionID, ok := ctx.Value(SessionIDKey).(string); ok && oldSessionID != "" {
		s.sessionStore.Delete(ctx, oldSessionID) // invalidate old
	}

	// 3. Create a fresh session with a new random token
	token, session, err := s.sessionService.Create(ctx, CreateSessionParams{
		TenantID:  tenantID,
		UserID:    userID,
		IPAddress: ip,
		UserAgent: userAgent,
		TTL:       24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil // new session, new ID — no fixation possible
}
```

### GGID Current State

GGID's `AuthService.Login()` (auth_service.go:83) calls
`s.sessionService.Create()` on **every** successful login, which generates a
fresh 32-byte random token via `crypto.GenerateRandomToken(32)` and a new UUID
session ID via `uuid.New()`. There is no pre-auth session concept — the session
is created only after credential validation succeeds.

**Verdict:** **SAFE.** Session fixation via ID reuse is not possible because:
1. Each login creates a brand-new session (new token, new UUID).
2. No server-side session exists before authentication.
3. Tokens are returned in the JSON response body, not as set-cookies.

### JWT Inherent Resistance

GGID uses self-contained JWT access tokens. JWTs are signed by the server and
carry their own claims (subject, expiry, tenant, session ID). An attacker cannot
"fix" a JWT because the client never assigns the token — the server mints it
after authentication. The session ID embedded in the JWT is generated server-side
and is not controllable by the client.

## 4. Defense: SAML InResponseTo

### SAML Session Protection

The `InResponseTo` attribute binds a SAML response to the specific AuthnRequest
that initiated the flow:

```xml
<samlp:Response InResponseTo="_a1b2c3d4-..." Destination="...">
  <saml:Assertion>
    <saml:AuthnStatement SessionIndex="_session-abc123"/>
  </saml:Assertion>
</samlp:Response>
```

**Validation steps (SP side):**
1. SP generates AuthnRequest with unique `ID` (e.g., `_a1b2c3d4-...`).
2. SP stores the request ID (Redis or database) with a short TTL.
3. IdP returns a Response with `InResponseTo="_a1b2c3d4-..."`.
4. SP validates: `response.InResponseTo` matches a stored request ID.
5. SP deletes the stored request ID (single-use).

### IdP-Initiated SSO Risk

IdP-initiated SSO has **no AuthnRequest**, so there is no `InResponseTo` to
validate. An attacker can craft an unsolicited assertion and push it to the
victim's ACS endpoint:

```
Attacker → IdP: Authenticates as themselves
IdP → Victim ACS: Unsolicited assertion (no InResponseTo)
Victim ACS: If IdP-initiated allowed → accepts assertion → session fixation
```

**Defense:** Disable IdP-initiated SSO by default. If required, enforce
additional context validation (issuer allowlist, audience restriction, time
window).

### GGID Current State

| Defense | Status | Notes |
|---------|--------|-------|
| InResponseTo validation | **Gap** | Design documented in `multi-tenant-saml.md` and `saml-sp-initiated-sso-design.md`, but no request ID tracking implemented in code |
| IdP-initiated SSO | **Gap** | No explicit enable/disable flag — should default to disabled |
| Assertion replay detection | **Partial** | Assertion IDs should be cached and rejected on replay |

The SAML research docs (`saml-sp-initiated-sso-design.md:333`) explicitly call
out: "InResponseTo validation — **Gap** — No request ID tracking."

## 5. Defense: OAuth State Binding

### State as Anti-Fixation

The `state` parameter in OAuth 2.0 prevents CSRF and session fixation on the
authorization callback:

```
RP → AS:  GET /authorize?response_type=code&client_id=...&state=RANDOM&redirect_uri=...
AS → RP:  GET /callback?code=AUTH_CODE&state=RANDOM
RP:       Validate callback.state == expected state (stored at step 1)
```

### Fixation Prevention Requirements

| Requirement | Why |
|-------------|-----|
| Cryptographically random | Prevents prediction (use crypto/rand, not math/rand) |
| Single-use | Prevents replay — delete after validation |
| Time-limited | Expires if user abandons the flow (e.g., 10 min) |
| Server-stored | RP must verify against server-side value, not just echo back |
| Bound to session | State should be linked to the RP's session (not just a random string) |

### GGID Current State

GGID's social login handler (`auth/internal/server/http.go:1988`) generates state:

```go
state := uuid.New().String()
```

**UUIDv4** has 122 bits of entropy, which is cryptographically sufficient against
prediction. However, there are gaps:

| Requirement | Status | Notes |
|-------------|--------|-------|
| Cryptographically random | **Pass** | `uuid.New()` uses crypto/rand internally (UUIDv4) |
| Single-use | **Gap** | State is not stored server-side; callback does not validate against stored value |
| Time-limited | **Gap** | No TTL enforcement |
| Server-stored | **Gap** | State returned in JSON to client; callback passes it through to connector |
| Session-bound | **Gap** | State is not linked to a server-side pre-auth session |

**Risk:** Without server-side validation, an attacker could replay a callback
with a known state and authorization code. This is partially mitigated because
authorization codes are single-use, but full CSRF protection requires
server-side state validation.

## 6. Defense: Cookie Attributes

### HttpOnly

Prevents JavaScript from reading the cookie via `document.cookie`, blocking
XSS-based session theft. Must be set on **all** session and authentication
cookies.

### Secure

Ensures the cookie is only transmitted over HTTPS, preventing network-level
interception (MITM, open Wi-Fi sniffing).

### SameSite

| Value | Behavior | Use Case |
|-------|----------|----------|
| Strict | Cookie sent only on same-site requests | Strongest — blocks all cross-site |
| Lax | Cookie sent on top-level GET navigations | Allows OAuth/SAML redirects, blocks CSRF POST |
| None | Cookie sent on all cross-site requests | Requires Secure; weakest |

**Recommendation:** `Lax` for session cookies — allows OAuth redirect flows
while blocking CSRF on POST/PUT/DELETE.

### Path

Session cookies should use `Path=/` (root scope) to ensure they are sent on
all authenticated requests. Scoping to `/auth` limits cookie exposure but
requires careful path management.

### GGID Cookie Audit

GGID does **not** use cookies for session transport. Access tokens and refresh
tokens are returned in the JSON response body, and clients are expected to
include them as `Authorization: Bearer <token>` headers. This is an API-first
design (SPA/mobile clients, not browser-form-based sessions).

| Attribute | Set? | Notes |
|-----------|------|-------|
| HttpOnly | N/A | No session cookies set by auth or oauth services |
| Secure | N/A | No session cookies |
| SameSite | N/A | No session cookies |
| Path | N/A | No session cookies |
| Domain | N/A | No session cookies |

**Note:** If GGID adds browser-based session cookies in the future (e.g., for
the admin console), all four attributes must be configured correctly. The
gateway's canary router (`canary.go:79`) already sets `HttpOnly: true` and
`SameSite: Lax` on its sticky cookie, which can serve as a template.

## 7. GGID Session Fixation Audit

| Defense | Protocol | GGID Status | Gap? | Priority |
|---------|----------|-------------|------|----------|
| Session ID regeneration | Login | **Safe** — new session + token on every login | None | — |
| JWT self-contained tokens | All | **Safe** — server-signed, not client-assigned | None | — |
| OAuth state generation | Social login | **Partial** — uuid.New() (sufficient entropy) | No server-side validation | P1 |
| OAuth state validation | Social login | **Gap** — not validated against stored value | Yes | P1 |
| SAML InResponseTo | SAML | **Gap** — documented in design, not implemented | Yes | P1 |
| SAML assertion replay | SAML | **Gap** — no assertion ID cache | Yes | P2 |
| OIDC nonce enforcement | OIDC | **Safe** — required for id_token flows, embedded in token | None | — |
| OIDC nonce validation | OIDC | **Safe** — nonce stored in auth code, included in ID token | None | — |
| Cookie HttpOnly | All | N/A — no session cookies | Future | P2 |
| Cookie Secure | All | N/A — no session cookies | Future | P2 |
| Cookie SameSite | All | N/A — no session cookies | Future | P2 |
| Pre-auth session invalidation | Login | **Safe** — no pre-auth session exists | None | — |

### Key Findings

1. **JWT is the primary fixation defense.** GGID's architecture is inherently
   resistant to session fixation because it uses server-signed JWTs, not
   client-assigned session IDs. The token is minted after authentication and
   cannot be fixed by the client.

2. **No cookie surface.** Without browser cookies, cookie-based fixation and
   XSS-based session theft are not currently applicable. This changes if cookie
   support is added for the console.

3. **OAuth state is the weakest link.** State is generated with sufficient
   entropy but not validated server-side. An attacker cannot predict the state,
   but the system cannot detect replay or CSRF without server-side tracking.

4. **SAML InResponseTo is the highest-risk gap.** Without request ID tracking,
   SAML responses cannot be bound to their originating requests, enabling
   unsolicited assertion attacks.

## 8. Implementation Roadmap

| Phase | Task | Priority | Effort | Risk Reduction |
|-------|------|----------|--------|----------------|
| 1 | Store OAuth `state` in Redis with TTL, validate on callback | P1 | 1 day | Eliminates CSRF + replay on social login |
| 2 | Implement SAML `InResponseTo` tracking (store AuthnRequest ID in Redis, validate on ACS) | P1 | 2 days | Eliminates unsolicited assertion attacks |
| 3 | Add SAML assertion replay detection (cache assertion IDs, reject duplicates) | P2 | 1 day | Defense-in-depth for SAML |
| 4 | Default IdP-initiated SSO to disabled (opt-in per tenant) | P1 | 0.5 day | Removes SAML fixation surface |
| 5 | Cookie attribute policy (if browser sessions added: HttpOnly + Secure + SameSite=Lax) | P2 | 1 day | Future-proofs against XSS session theft |

**Total estimated effort:** ~5.5 days across P1 and P2 items.

### Phase 1: OAuth State Validation (P1)

Store `state` in Redis on login begin with a 10-min TTL. On callback, use
`GetDel` (atomic single-use) to validate the state exists and matches the
expected provider. Reject if missing or mismatched.

### Phase 2: SAML InResponseTo (P1)

Store `AuthnRequest.ID` in Redis (`saml:pending:{tenant}:{id}`, 5-min TTL) on
request generation. On ACS, validate `response.InResponseTo` is non-empty and
matches a stored key. Delete the key after validation (single-use). Reject
missing/expired InResponseTo as potential unsolicited assertions.

---

## Appendix: Protocol Fixation Summary

| Protocol | Fixation Vector | GGID Defense | Status |
|----------|----------------|--------------|--------|
| Password login | Cookie/session ID reuse | JWT minted after auth, no pre-auth session | **Protected** |
| OAuth social | State prediction/replay | uuid.New() (122-bit entropy) | **Partial** — needs server validation |
| SAML SP-init | Unsolicited assertion | InResponseTo binding (design only) | **Gap** |
| SAML IdP-init | No request binding | N/A — should be disabled by default | **Gap** |
| OIDC | Nonce replay | Nonce required + embedded in ID token | **Protected** |
| OIDC backchannel logout | Token replay | Nonce rejected in logout token | **Protected** |
