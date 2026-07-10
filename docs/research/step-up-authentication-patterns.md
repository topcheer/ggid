# Step-Up Authentication Patterns

> Protocol-level step-up authentication: OIDC parameters, session upgrade, and
> per-route ACR enforcement. Focuses on the *protocol mechanics*, not risk scoring
> (see `adaptive-mfa-design.md` for risk-based triggers).
>
> **References:** RFC 9470 (OAuth 2.0 Step-Up Challenge), OIDC Core §3.1.2.1,
> NIST SP 800-63B Rev. 3 §5.2 (AAL2).

---

## 1. What is Step-Up Authentication

Step-up authentication lets a user who is *already authenticated* at a lower
assurance level (AAL1 — single factor, e.g. password) gain access to a resource
that requires a higher assurance level (AAL2 — multi-factor) **without** starting
a brand-new authentication session.

**Key properties:**

- The existing session is preserved — user identity, SSO context, and session
  state remain intact.
- Only the *missing factor* is collected (e.g. TOTP code or WebAuthn
  assertion).
- The result is an elevated token/session, not a fresh login.

**Step-up vs. re-authentication:**

| | Step-Up | Re-Authentication |
|---|---|---|
| Session | Preserved | Destroyed & recreated |
| Factors | Adds missing factor(s) | All factors re-entered |
| User experience | One additional prompt | Full login flow |
| Use case | AAL1→AAL2 elevation | Session timeout, suspicious activity |

---

## 2. OIDC Step-Up Parameters

OIDC and OAuth 2.0 define three request parameters that drive step-up behavior.
These are evaluated by the Authorization Server (AS) during the `/authorize`
request.

### 2.1 `acr_values`

**Authentication Context Class Reference** — declares the minimum ACR level the
RP requires:

```
GET /authorize?response_type=code
  &client_id=banking-app
  &acr_values=urn:mace:incommon:iap:silver
  &redirect_uri=https://app.example.com/callback
```

Common ACR values:

| ACR URI | NIST AAL | Meaning |
|---------|----------|---------|
| `urn:mace:incommon:iap:bronze` | AAL1 | Single-factor (password) |
| `urn:mace:incommon:iap:silver` | AAL1+ | Password + proof of identity |
| `urn:mace:incommon:iap:gold` | AAL2 | MFA required |
| Custom: `aal1`, `aal2` | — | Simple numeric levels |

The AS compares the current session's ACR against the requested `acr_values`. If
the session is insufficient, the AS prompts for the additional factor(s).

### 2.2 `max_age`

Maximum acceptable age of the user's authentication, in seconds:

```
&max_age=300   // User must have authenticated within the last 5 minutes
```

If `auth_time` in the session is older than `max_age`, the AS forces
re-authentication. This implements "recent authentication required" semantics for
sensitive operations (fund transfers, password changes, admin actions).

### 2.3 `prompt=login`

Forces the AS to prompt for re-authentication **regardless of session age**:

```
&prompt=login
```

This ignores any existing SSO session entirely. More aggressive than `max_age`
— useful for highly sensitive operations or when the RP detects potential
session hijacking.

### 2.4 `prompt=consent`

Forces the AS to display a consent screen:

```
&prompt=consent
```

Useful during step-up when the RP requests new scopes that weren't previously
granted. The user sees what additional permissions are being requested alongside
the MFA challenge.

---

## 3. Step-Up Flow Sequence

### 3.1 ASCII Sequence Diagram

```
 User       Browser/RP         Auth Server (GGID)      MFA Svc
  |              |                     |                     |
  | Click        |                     |                     |
  |—"Transfer"→ |                     |                     |
  |              | Redirect /authorize |                     |
  |              | acr_values=gold     |                     |
  |              | max_age=300         |                     |
  |              |————————→————————→——|                     |
  |              |                     | Eval: ACR=silver    |
  |              |                     | Need: gold → MFA    |
  |              | 302 → MFA challenge |                     |
  |              |←——————←——————←——————|                     |
  | Enter TOTP   |                     |                     |
  |———————————→ |                     |                     |
  |              | POST verify (code)  |                     |
  |              |————————→————————→——|—→ VerifyTOTP() ——→|
  |              |                     |   ←——— OK ————←——|
  |              |     Auth code       |                     |
  |              |     acr=gold        |                     |
  |              |←——————←——————←——————|                     |
  |              | Exchange → token    |                     |
  |              |————————→————————→——|                     |
  |              | access_token {      |                     |
  |              |   acr: "gold",      |                     |
  |              |   auth_time: now    |                     |
  |              | }                   |                     |
  |              |←——————←——————←——————|                     |
  | "Allowed"   |                     |                     |
  |←————————————|                     |                     |
```

### 3.2 Step-by-step

1. User holds AAL1 session (password).
2. User clicks a sensitive action (e.g. "Transfer Funds").
3. RP redirects to `/authorize` with `acr_values=gold` + `max_age=300`.
4. AS evaluates: session ACR=silver, required=gold → gap.
5. AS shows MFA challenge (TOTP or WebAuthn).
6. User completes the second factor.
7. AS issues authorization code bound to `acr=gold`.
8. RP exchanges code for tokens with `acr: "gold"` and fresh `auth_time`.
9. RP validates `acr` claim and grants access.

---

## 4. Session Upgrade vs. Token Reissuance

### 4.1 Session Upgrade

The AS upgrades the existing session's AAL. All subsequent requests within that
session operate at the elevated level.

**Risk:** Session downgrade attack — clearing cookies forces AAL1 re-auth.

**Mitigation:** Session remembers *maximum achieved AAL*; never downgrade.

### 4.2 Token Reissuance

AS issues a new token with elevated `acr` claim. Old token stays valid at lower
ACR. RP selects the correct token per operation sensitivity.

```go
// issueStepUpToken — issues a short-lived token with elevated ACR
func (s *AuthService) issueStepUpToken(ctx context.Context, userID uuid.UUID, acr string) (string, error) {
    claims := jwt.MapClaims{
        "sub": userID.String(),
        "acr": acr,                // e.g. "urn:mace:incommon:iap:gold"
        "iat": time.Now().Unix(),
        "exp": time.Now().Add(5 * time.Minute).Unix(), // short TTL
    }
    return s.signToken(claims)
}
```

### 4.3 Mixed Approach (Recommended)

Combine both: upgrade the session AAL **and** issue a short-lived scoped token.
The token provides per-request proof; the session provides continuity.

| Aspect | Session Only | Token Only | Mixed |
|--------|-------------|------------|-------|
| Downgrade resistance | Weak | Strong | Strong |
| Per-request proof | No | Yes | Yes |
| SSO continuity | Yes | No | Yes |
| Complexity | Low | Medium | Medium |

---

## 5. GGID Current Capabilities

GGID already has a functional step-up framework in
`services/auth/internal/service/stepup.go`. Here is the gap analysis:

| Capability | GGID Status | Gap |
|---|---|---|
| `acr_values` parsing in `/authorize` | **Partial** — `ACRStepUpCheck()` compares levels | Not wired to OAuth `/authorize` endpoint |
| `max_age` enforcement | **Not implemented** | No `auth_time` comparison logic |
| `prompt=login` | **Not implemented** | No forced re-auth path |
| `prompt=consent` | **Not implemented** | No consent screen flow |
| Session AAL tracking | **Partial** — step-up token in Redis, not in JWT/session | No persistent AAL in session |
| Step-up token issuance | **Implemented** — `VerifyStepUp()` issues 5-min token | Token is opaque, not a JWT with `acr` claim |
| Step-up token validation | **Implemented** — `ValidateStepUpToken()` checks Redis | Gateway doesn't call it |
| Gateway ACR enforcement | **Not implemented** | No per-route ACR middleware |
| MFA (TOTP) | **Implemented** — `mfaService.VerifyUserCode()` | Works, used by step-up |
| WebAuthn for step-up | **Not wired** — exists as separate flow | Could be second factor for step-up |

### 5.1 Existing Code Highlights

- **InitStepUp** — Creates challenge token in Redis (`ggid:stepup:<challenge>`), 5-min TTL. Stores `tenantID:userID:method`.
- **VerifyStepUp** — Validates password or TOTP code. On success, issues step-up token (`ggid:stepup-token:<token>`), 5-min TTL.
- **ValidateStepUpToken** — Checks if step-up token is valid for a given user.
- **ACRStepUpCheck** — Compares `currentACR` vs `requestedACR` via `acrLevel()`. If insufficient, calls `InitStepUp` (password for level 1, MFA for level 2+).

**Key gap:** Step-up token is an opaque random string in Redis — **not** a JWT with `acr` claim. RPs must call `ValidateStepUpToken` on each request rather than verifying cryptographically.

---

## 6. Implementation Design

### 6.1 ACR Evaluator Interface

```go
// pkg/auth/acr.go
type ACREvaluator interface {
    Evaluate(ctx context.Context, sessionACR, requiredACR string) (ok bool, challenge *StepUpRequirement, err error)
}

type StepUpRequirement struct {
    CurrentACR  string `json:"current_acr"`
    RequiredACR string `json:"required_acr"`
    Method      string `json:"method"` // "password" or "mfa"
    MaxAge      int    `json:"max_age,omitempty"`
}
```

### 6.2 Auth Service: Wire ACR to `/authorize`

```go
// services/oauth/internal/server/server.go
func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
    acrValues := r.URL.Query().Get("acr_values")
    maxAgeStr := r.URL.Query().Get("max_age")
    prompt := r.URL.Query().Get("prompt")

    session := s.getSession(r)
    ok, challenge, err := s.acrEvaluator.Evaluate(r.Context(),
        session.ACR, acrValues)
    if err != nil {
        s.writeError(w, http.StatusInternalServerError, err)
        return
    }

    // Enforce max_age
    if maxAgeStr != "" {
        maxAge, _ := strconv.Atoi(maxAgeStr)
        if time.Since(session.AuthTime) > time.Duration(maxAge)*time.Second {
            ok = false
            challenge = &StepUpRequirement{Method: "reauth"}
        }
    }

    // prompt=login forces re-auth
    if prompt == "login" {
        ok = false
        challenge = &StepUpRequirement{Method: "reauth"}
    }

    if !ok {
        s.redirectToStepUp(w, r, challenge) // → MFA challenge page
        return
    }
    s.issueAuthorizationCode(w, r, session) // session sufficient
}
```

### 6.3 Gateway: ACR Enforcement Middleware

```go
// services/gateway/internal/middleware/acr_enforcement.go

// RouteACR maps route prefixes to required ACR levels.
type RouteACR map[string]string // "/admin/*" → "urn:mace:incommon:iap:gold"

func ACREnforcementMiddleware(routes RouteACR) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            requiredACR := matchRoute(routes, r.URL.Path)
            if requiredACR == "" {
                next.ServeHTTP(w, r)
                return
            }

            // Extract ACR from JWT claims (set by JWT middleware)
            tokenACR, _ := r.Context().Value("acr").(string)

            if acrLevel(tokenACR) < acrLevel(requiredACR) {
                w.Header().Set("WWW-Authenticate",
                    `Bearer error="insufficient_acr", `+
                        `acr_values="`+requiredACR+`"`)
                w.WriteHeader(http.StatusUnauthorized)
                json.NewEncoder(w).Encode(map[string]string{
                    "error":             "insufficient_authentication",
                    "required_acr":      requiredACR,
                    "step_up_endpoint":  "/api/v1/auth/stepup",
                })
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func matchRoute(routes RouteACR, path string) string {
    for pattern, acr := range routes {
        matched, _ := filepath.Match(pattern, path)
        if matched {
            return acr
        }
    }
    return ""
}
```

### 6.4 Configuration

```yaml
# config.yaml
gateway:
  acr_enforcement:
    routes:
      "/api/v1/admin/*": "urn:mace:incommon:iap:gold"
      "/api/v1/billing/*": "urn:mace:incommon:iap:gold"
      "/api/v1/users/*": "urn:mace:incommon:iap:silver"
```

---

## 7. NIST AAL2 Requirements Checklist

Per NIST SP 800-63B Rev. 3 §5.2, AAL2 step-up must satisfy:

- [ ] **Two distinct authentication factors** — memorized secret + OTP/WebAuthn device
- [ ] **Verified authenticators** — second factor bound to subscriber's account
- [ ] **FIPS 140-2 validated cryptography** — token signing uses validated crypto module
- [ ] **Session binding** — `acr` claim must be signed, not just a flag
- [ ] **New cryptographic proof** — fresh signed token, not a boolean flag
- [ ] **Replay resistance** — step-up tokens short-lived (≤5 min), single-use or scoped
- [ ] **Channel binding** (optional) — bind step-up proof to TLS channel

---

## 8. Competitor Approaches

| Platform | Mechanism | Notes |
|----------|-----------|-------|
| **Auth0** | Actions (post-login rules) | Evaluate `acr` in Actions; redirect to MFA if insufficient. Custom rule `context.acr`. |
| **Keycloak** | `STEP_UP_MECHANISM` | Auth session attribute; `acr` claim in token. Per-flow required actions. |
| **Azure AD** | Conditional Access | Policy-based: require MFA for specific apps/conditions. Evaluated at sign-in. |
| **Okta** | Inline Hooks / `acr_values` | Pre-defined step-up via policy; also supports OIDC `acr_values` natively. |
| **AWS Cognito** | `custom:auth_challenge` | Lambda triggers for custom auth flows; no native ACR support. |

---

## 9. Roadmap

### Phase 1: ACR Parsing + Gateway Enforcement (Week 1-2)

- Wire `ACRStepUpCheck` to OAuth `/authorize` endpoint
- Parse `acr_values` parameter in authorize request
- Add `acr` claim to JWT tokens issued after step-up
- Implement `ACREnforcementMiddleware` in Gateway
- Per-route ACR configuration

### Phase 2: `max_age` + `prompt` Support (Week 2)

- Track `auth_time` in session (Redis)
- Enforce `max_age` comparison in `/authorize`
- Implement `prompt=login` (force re-auth) and `prompt=consent`
- Add `auth_time` claim to ID tokens

### Phase 3: Session AAL Tracking + No-Downgrade (Week 3)

- Store `max_achieved_aal` in session (never downgrade within session lifetime)
- Migrate step-up token from opaque string to signed JWT with `acr` claim
- Wire WebAuthn as step-up second factor (currently TOTP-only)
- Integration tests: full step-up flow through Gateway

**Estimated effort:** 2-3 weeks (1 developer)
