# OAuth Mix-Up Attack Defense

> Research document examining the OAuth 2.0 mix-up attack vector against
> GGID's authorization code flow, with code-level analysis and an
> implementation roadmap aligned to RFC 9207 and RFC 9700.

---

## 1. Overview

The **mix-up attack** is an OAuth flow-manipulation vulnerability where an
attacker confuses the Relying Party (RP/client) about which Authorization
Server (AS) it is communicating with.

Modern applications commonly support **multi-provider SSO** — a user can
authenticate via Google, Okta, GitHub, or GGID within the same application.
Each provider is a separate AS with its own authorize endpoint, token
endpoint, and credential namespace.

In a mix-up attack, the adversary exploits this multi-AS environment to
**inject their own AS's authorization code** into the legitimate flow. The
RP, unable to distinguish which AS issued the code, exchanges it at the wrong
endpoint — or at an attacker-controlled endpoint — and receives a token for
the **attacker's account** instead of the victim's.

**Impact**: The victim is silently logged into the attacker's account. Any
data the victim creates or uploads (documents, messages, payment info) is
accessible to the attacker, who controls the account. This can lead to data
exfiltration and credential theft through deceptive UI.

The attack was formally described by Christian Mainka, Daniel Fett, and
Jorg Schwenk at IETF 95 (April 2016) and published in peer-reviewed research
in 2017. It directly led to two RFCs:

- **RFC 9207** (2022): `iss` parameter in authorization responses
- **RFC 9700** (2025): OAuth 2.0 Security Best Current Practice (makes
  issuer identification a SHOULD-level requirement)

---

## 2. Attack Scenario

### Step-by-Step Attack

```
  Victim                RP (Client)          AS_A (Honest)        AS_B (Attacker)
    |                      |                      |                    |
    |--- login ----------->|                      |                    |
    |                      |--- /authorize ------>|                    |
    |                      |   (client_id,        |                    |
    |                      |    redirect_uri,     |                    |
    |                      |    state=ABC)        |                    |
    |                      |                      |                    |
    |    Attacker intercepts initial login,          launches parallel  |
    |    flow with AS_B:                                                |
    |                      |--------------------------------------->   |
    |                      |--- /authorize ----------------------->   |
    |                      |   (client_id at AS_B)                 AS_B
    |                      |                                       issues
    |                      |<------- code_B + state --------------  code_B
    |                      |                                          |
    |<--- redirect --------|                                          |
    |     (attacker replaces                                       AS_A issues
    |      code_A with code_B)                                      code_A
    |                      |                                          |
    |    RP receives code_B + state=ABC (looks valid)                 |
    |                      |                                          |
    |                      |--- /token (code_B) --> AS_A: FAILS       |
    |                      |    OR                                     |
    |                      |--- /token (code_B) -------> AS_B: OK     |
    |                      |<------ attacker's token -----            |
    |                      |                                          |
    |<--- logged into      |                                          |
    |     ATTACKER's       |                                          |
    |     account          |                                          |
```

1. The RP supports multiple AS: **AS_A** (e.g., Google) and **AS_B**
   (attacker-controlled, registered as a valid provider).
2. The victim initiates login; the RP creates an authorization request for
   AS_A with `state=ABC`.
3. The attacker — who has an active session with the RP — simultaneously
   initiates a login flow at AS_B, obtaining `code_B` and `state=ABC`
   (state may be predictable or reused).
4. The victim authenticates at AS_A and receives `code_A`.
5. The attacker swaps `code_A` with `code_B` in the redirect (e.g., via an
   open redirect, XSS, or session fixation).
6. The RP receives `code_B` + `state=ABC`. The state validates, so the RP
   proceeds. It either:
   - Exchanges `code_B` at AS_A's token endpoint — fails (code not found),
     **or**
   - Falls back to AS_B's token endpoint — succeeds, returning the
     attacker's token.
7. The victim is now logged into the **attacker's account**. The attacker
   monitors the account and exfiltrates any data the victim uploads.

### Why It Works

- The RP **does not verify which AS issued the code**.
- The `state` parameter is validated against the RP's session but **not
  bound to a specific AS** — `state=ABC` is valid regardless of whether the
  code came from AS_A or AS_B.
- The token endpoint: the RP doesn't verify the issuer before accepting the
  token. If it tries AS_A and fails, it may retry with AS_B without flagging
  the inconsistency.

### Real-World Example

The attack was demonstrated against major IdPs (Google, Microsoft, Facebook)
in the 2017 paper *"On the Security of Multi-Provider Single Sign-On"* by
Fett, Kormann, and Schwenk. Several IdPs patched their flows after
disclosure. The research directly motivated RFC 9207 (iss parameter, 2022)
and the issuer identification requirements in RFC 9700 (2025).

---

## 3. Defense: State + Nonce Binding

### state Parameter

The `state` parameter is the first line of defense. Requirements:

- **Unique per request**: never reuse a state value.
- **Cryptographically random**: at least 128 bits of entropy.
- **Validated on callback**: the RP must check that the returned state
  matches what it sent.
- **Bound to AS** (critical for mix-up defense): encode which AS this
  request targets, so a state from AS_A cannot be used for AS_B.

```go
// AS-bound state encoding
type AuthRequest struct {
    State     string // 32 random bytes, base64url
    ASID      string // "google", "okta", "ggid"
    SessionID string // RP's session identifier
}

// Encode: state = base64url(random + "|" + ASID + "|" + SessionID)
// On callback: decode state, verify ASID matches the AS that issued the code
func ValidateState(state string, expectedASID string) error {
    parts := decode(state)
    if parts[1] != expectedASID {
        return errors.New("state bound to different AS — mix-up detected")
    }
    return nil
}
```

### nonce Parameter (OIDC)

For OpenID Connect flows, the `nonce` adds a second binding layer:

- RP generates a random nonce per auth request.
- The AS includes the nonce in the `id_token` claims.
- RP validates: `id_token.nonce == expected_nonce`.
- If the token was issued by a different AS, the nonce will not match.

### GGID Current State

| Check | Status | Evidence |
|-------|--------|----------|
| `state` required | Implemented | `oauth_service.go:223` — returns error if `req.State == ""` |
| `state` validated on callback | **Not implemented** | `ExchangeAuthorizationCode` (line 291) does not check state at all |
| `state` bound to AS | **Not implemented** | State is passed through but never encoded with AS identity |
| `nonce` required for OIDC | Implemented | `oauth_service.go:228` — enforced for `id_token` flows |
| `nonce` in id_token | Implemented | `oauth_service.go:344` — `code.Nonce` included in id_token |
| `nonce` validated by AS | N/A (RP responsibility) | GGID as AS correctly embeds nonce; RP-side validation is RP's job |

**Gap**: GGID requires `state` at authorize time but does not validate it on
token exchange. State serves only as a pass-through value. It is not bound
to a specific AS, so a mix-up substitution is not detectable.

---

## 4. Defense: Issuer Identification (RFC 9207 + RFC 9700)

### RFC 9207: `iss` in Authorization Response

RFC 9207 requires the AS to include its issuer identifier in the
authorization response:

```
HTTP/1.1 302 Found
Location: https://app.example.com/callback?
    code=SplxlOBeZQQYbYS6WxSbIA
    &state=abc123
    &iss=https://auth.ggid.io
```

The RP validates `iss` **before exchanging the code**:

```go
func (rp *RelyingParty) HandleCallback(r *http.Request) error {
    iss := r.URL.Query().Get("iss")
    if iss != rp.expectedASIssuer {
        return errors.New("issuer mismatch — mix-up attack detected")
    }
    // Safe to exchange code
    return rp.ExchangeCode(r.URL.Query().Get("code"))
}
```

### RFC 9700: `issuer` in Token Response

RFC 9700 (OAuth 2.0 Security BCP) recommends the AS include `issuer` in the
token response JSON:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "issuer": "https://auth.ggid.io"
}
```

If the RP receives a token response where `issuer` does not match the
expected AS, it rejects the token — the mix-up is detected at the token
endpoint.

### GGID Compliance

| Requirement | RFC | GGID Status | Action Needed |
|-------------|-----|-------------|---------------|
| `iss` in auth response | 9207 | **Missing** | Add `iss` param to authorization redirect |
| `issuer` in token response | 9700 | **Missing** | Add `Issuer` field to `TokenResponse` struct |
| `iss` in id_token | OIDC Core | Implemented | `oauth_service.go:450` — `"iss": s.issuer` |
| `iss` in access token JWT | 9068 | Implemented | `oauth_service.go:417` — `"iss": s.issuer` |
| `issuer` in discovery | OIDC Discovery | Implemented | `oauth_service.go:360` — `Issuer: s.issuer` |

GGID embeds the issuer in **all token artifacts** (access JWT, id_token,
discovery document) but omits it from the two response surfaces where it
matters most for mix-up defense: the authorization response redirect and the
token response JSON.

---

## 5. Defense: Redirect URI Validation

Redirect URI validation prevents code injection by ensuring each AS uses a
distinct callback path. If AS_A uses `/callback/google` and AS_B uses
`/callback/ggid`, a code from AS_B cannot be delivered to AS_A's callback
without a redirect manipulation that the browser would block.

### RFC Requirements

- **Exact match only** (RFC 6749 Section 3.1.2.3, reinforced by RFC 9700):
  no wildcard, no path-prefix, no trailing-slash normalization.
- **Per-AS redirect URIs**: different callback paths for different providers.
- **Validated on both endpoints**: authorize (redirect validation) and token
  (redirect_uri must match the one used at authorize).

### GGID Validation

| Check | Status | Evidence |
|-------|--------|----------|
| Registered redirect URIs | Implemented | `CreateClient` stores `RedirectURIs` list |
| Exact match at authorize | Implemented | `oauth_service.go:212` — `client.ValidateRedirectURI()` |
| Match at token exchange | Implemented | `oauth_service.go:318` — `code.RedirectURI != req.RedirectURI` |
| Per-client registration | Implemented | Each client has its own `RedirectURIs` array |

**Assessment**: GGID's redirect URI handling is correct. The exact-match
validation on both endpoints closes the most common code-injection vectors.
However, redirect URI validation alone does not prevent mix-up if the
attacker and victim share the same RP registration or the RP uses the same
callback for all AS.

---

## 6. Defense Summary Table

| Defense | What It Prevents | GGID Status | Priority |
|---------|-----------------|-------------|----------|
| `state` validation | CSRF + basic session binding | Partial (required, not verified) | P1 |
| AS-bound state | Mix-up (code from wrong AS) | **Not implemented** | P1 |
| `nonce` validation (OIDC) | Token injection across AS | Implemented (AS-side) | Done |
| `issuer` in token response | Mix-up (RFC 9700) | **Not implemented** | P0 |
| `iss` in auth response | Mix-up (RFC 9207) | **Not implemented** | P0 |
| Per-client redirect URI | Code confusion | Implemented | Done |
| PKCE | Code interception | Implemented | Done |
| Exact redirect match (both endpoints) | Code injection | Implemented | Done |

---

## 7. GGID Mitigation Gaps

### Gap Analysis Table

| Gap | Current Behavior | Risk | Fix | Effort |
|-----|-----------------|------|-----|--------|
| No `iss` in auth response | Authorization redirect omits issuer ID | RP cannot detect code from wrong AS at callback | Append `&iss={issuer}` to redirect URL | 0.5 day |
| No `issuer` in token response | `TokenResponse` struct has no issuer field | RP cannot verify AS at token exchange | Add `Issuer` field to `TokenResponse` | 0.5 day |
| State not verified on exchange | `ExchangeAuthorizationCode` ignores state | CSRF weakened; state provides no session binding | Store state in code, verify on exchange | 1 day |
| State not bound to AS | State is opaque, no AS identity encoded | Mix-up undetectable via state | Encode `ASID` in state or store alongside code | 1 day |
| No nonce validation (RP-side) | GGID is AS, nonce validation is RP's job | N/A for GGID as AS | Document expectation for RP clients | 0.5 day |

### Code Analysis: oauth_service.go

**CreateAuthorizationCode** (line 202-267):
- Validates: client exists, client enabled, redirect URI registered,
  response_type allowed, `state` present (non-empty), `nonce` for OIDC,
  PKCE for public clients.
- Stores in `AuthorizationCode`: `CodeHash`, `ClientID`, `UserID`,
  `RedirectURI`, `Scope`, `CodeChallenge`, `CodeChallengeMethod`, `Nonce`,
  `ExpiresAt`.
- **Does NOT store**: `State` (it is required but discarded — never
  persisted with the code for later verification).

**ExchangeAuthorizationCode** (line 291-352):
- Validates: client credentials, code existence (atomic consume), code
  matches client, redirect URI matches, PKCE verifier.
- **Does NOT validate**: `state` (not even present in the request struct),
  issuer identity.
- Returns: `TokenResponse` with `AccessToken`, `TokenType`, `ExpiresIn`,
  `RefreshToken`, `IDToken`, `Scope` — **no `issuer` field**.

**Key observation**: The `TokenExchangeRequest` struct (line 270-278) has no
`State` field at all. Even if the RP sends `state` in the callback, GGID's
token endpoint has no mechanism to receive or verify it. State validation
is entirely delegated to the RP — which is architecturally correct for the
CSRF use case but insufficient for mix-up defense.

---

## 8. Implementation Roadmap

### Phase 1: Add `issuer` to Token Response — P0 (~0.5 day)

```go
// oauth_service.go — TokenResponse
type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
    RefreshToken string `json:"refresh_token,omitempty"`
    IDToken      string `json:"id_token,omitempty"`
    Scope        string `json:"scope,omitempty"`
    Issuer       string `json:"issuer"` // NEW: RFC 9700
}

// In ExchangeAuthorizationCode, after building resp:
resp.Issuer = s.issuer
```

### Phase 2: Add `iss` to Authorization Response — P0 (~0.5 day)

In the HTTP handler that performs the 302 redirect after
`CreateAuthorizationCode`:

```go
redirectURL := fmt.Sprintf("%s?code=%s&state=%s&iss=%s",
    redirectURI, code, state, url.QueryEscape(s.issuer))
```

### Phase 3: Verify State on Token Exchange — P1 (~1 day)

- Add `State` field to `TokenExchangeRequest`.
- Store state hash alongside the authorization code.
- On exchange, verify `req.State == storedState`.

### Phase 4: Bind State to AS — P1 (~1 day)

- Encode AS identifier in state: `state = base64(random + ":" + asID)`.
- On callback, decode and verify `asID` matches expected issuer.
- Provides defense-in-depth alongside Phase 2's `iss` parameter.

### Phase 5: Documentation — P2 (~0.5 day)

- Document RP-side nonce validation requirements.
- Document per-AS redirect URI best practices.
- Add mix-up attack section to security guide.

**Total estimated effort: ~3.5 days**. Phases 1-2 (P0) are single-line
changes that close the most critical gaps and bring GGID into RFC 9207/9700
compliance.

---

## References

- **RFC 9207**: OAuth 2.0 Authorization Server Issuer Identification
  (https://www.rfc-editor.org/rfc/rfc9207.html)
- **RFC 9700**: Best Current Practice for OAuth 2.0 Security
  (https://www.ietf.org/rfc/rfc9700.html)
- Daniel Fett, "Mix-Up, Revisited" (https://danielfett.de/2020/05/04/mix-up-revisited/)
- IETF 95 slides: OAuth Mix-Up Attack
  (https://www.ietf.org/proceedings/95/slides/slides-95-oauth-5.pdf)
- RFC 6749 Section 10.12: OAuth 2.0 Security Considerations
