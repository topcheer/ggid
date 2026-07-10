# Cross-Site Request Forgery (CSRF) Defense for IAM Systems

> Research document for the GGID project — a Go-based multi-tenant IAM suite.
> Covers CSRF attack mechanics, defense patterns, Go implementation examples,
> and a gap analysis of the current GGID gateway CSRF posture.

---

## 1. CSRF Attack Mechanics

### How CSRF Works Against IAM Endpoints

Cross-Site Request Forgery (CSRF) exploits the browser's automatic credential
attachment behavior. When a user is logged into an IAM system, their browser
holds a session cookie (or JWT cookie). If the user visits a malicious site
while that cookie is still valid, the attacker's page can issue a cross-origin
request to the IAM endpoint — the browser automatically attaches the session
cookie, making the request appear authenticated.

```
1. User logs into https://iam.example.com → browser stores session cookie
2. User visits https://evil.example.com
3. evil.example.com contains:
     <form action="https://iam.example.com/api/v1/users/admin" method="POST">
       <input type="hidden" name="role" value="superadmin">
     </form>
     <script>document.forms[0].submit()</script>
4. Browser sends POST to iam.example.com WITH the session cookie
5. Server executes the request as the authenticated user
```

### Why Cookies Make IAM Vulnerable

IAM endpoints are uniquely dangerous CSRF targets because they manage
high-privilege operations: user creation, role assignment, session revocation,
password changes, and policy modifications. The vulnerability vector is
structural:

- **Cookie-based sessions** are sent automatically by the browser — no
  JavaScript required for `POST` form submissions.
- **Bearer tokens in `Authorization` headers** are NOT automatically attached,
  making API-only architectures with header-based auth immune to classical CSRF.
  However, many IAM systems (including GGID's console) use cookies for the admin
  UI while APIs use Bearer tokens — creating a mixed-trust surface.
- **`SameSite=None` cookies** (required for cross-site iframe scenarios) are
  sent on every cross-origin request, maximizing CSRF exposure.

### Login CSRF vs. Session CSRF

| Aspect | Login CSRF | Session CSRF |
|--------|-----------|--------------|
| Pre-conditions | Victim is NOT logged in | Victim IS logged in |
| Attack | Forcer logs victim into attacker's account | Forger submits state-changing request as victim |
| Impact | Attacker observes victim's actions through their own account | Unauthorized privilege escalation, data modification |
| Detection | Hard to detect — user sees "their" account | Audit logs show user performing actions they didn't initiate |

**Login CSRF** is particularly insidious for IAM: the attacker forces the victim
to log into the attacker's account. The victim, believing they are in their own
account, may enter sensitive data, link accounts, or configure MFA — all visible
to the attacker who controls the account.

### Real-World CSRF Vulnerabilities in IAM Products

- **OAuth 2.0 `state` parameter omission** — COUNTLESS OAuth implementations
  have been found omitting or not validating the `state` parameter, enabling
  login CSRF and code injection. This is the #1 OAuth vulnerability class.
- **Keycloak** (CVE-2020-10765) — CSRF in the admin console allowed
  authenticated attackers to perform actions as another admin.
- **Auth0** — Historical CSRF issues in password reset flows where the reset
  token could be triggered by a cross-site request.
- **Spring Security OAuth** — Multiple CSRF bypasses through custom header
  allowlists that inadvertently excluded the CSRF check for certain content types.

---

## 2. Double-Submit Cookie Pattern

### How It Works

The double-submit pattern requires the client to send the CSRF token in two
places: a cookie and a request header. The server compares them — if they match,
the request is genuine.

```
Client (JS reads cookie)                    Server
  │                                           │
  ├── Cookie: csrf_token=abc123 ─────────────►│
  ├── Header: X-CSRF-Token: abc123 ──────────►│
  │                                           │
  │                             Compare cookie vs header ──► Match? Allow / Reject
```

### Why Stateless

The server stores no CSRF state. It simply verifies that the cookie value and
header value are identical. This makes it ideal for distributed deployments
where sessions are not shared across nodes — no Redis lookup needed.

### Weaknesses

1. **Subdomain cookie injection** — If `evil.iam.example.com` can set cookies
   for the `.iam.example.com` domain, it can plant a known `csrf_token` value.
   When the victim visits `iam.example.com`, the attacker's CSRF token is in
   both the cookie and the forged request header.
2. **No per-session binding** — The token is not tied to the user's session.
   If an attacker can predict or leak the token, they can forge requests.
3. **Requires JavaScript** — The token must be read from the cookie by JS and
   added to headers. This means the CSRF cookie cannot be `HttpOnly`.

### Go Middleware Implementation

```go
package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

// DoubleSubmitCSRF implements the double-submit cookie CSRF pattern.
// On safe requests (GET/HEAD/OPTIONS), it sets/refreshes the csrf cookie.
// On unsafe requests, it validates cookie == header.
type DoubleSubmitConfig struct {
	CookieName string // default: "csrf_token"
	HeaderName string // default: "X-CSRF-Token"
	Secure     bool   // cookie Secure flag (true in production)
	SameSite   http.SameSite
}

func DefaultDoubleSubmitConfig() DoubleSubmitConfig {
	return DoubleSubmitConfig{
		CookieName: "csrf_token",
		HeaderName: "X-CSRF-Token",
		Secure:     true,
		SameSite:   http.SameSiteLaxMode,
	}
}

func DoubleSubmitCSRF(cfg DoubleSubmitConfig) func(http.Handler) http.Handler {
	if cfg.CookieName == "" {
		cfg = DefaultDoubleSubmitConfig()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				setDoubleSubmitCookie(w, cfg)
				next.ServeHTTP(w, r)
				return
			}

			cookieToken, err := r.Cookie(cfg.CookieName)
			if err != nil || cookieToken.Value == "" {
				rejectCSRF(w, "missing CSRF token cookie")
				return
			}

			headerToken := r.Header.Get(cfg.HeaderName)
			if headerToken == "" {
				rejectCSRF(w, "missing CSRF token header")
				return
			}

			if subtle.ConstantTimeCompare(
				[]byte(cookieToken.Value), []byte(headerToken),
			) != 1 {
				rejectCSRF(w, "CSRF token mismatch")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func setDoubleSubmitCookie(w http.ResponseWriter, cfg DoubleSubmitConfig) {
	token := generateRandomToken(32)
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: false, // Must be readable by JavaScript
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	})
}

func generateRandomToken(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet ||
		method == http.MethodHead ||
		method == http.MethodOptions
}

func rejectCSRF(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
```

> **Note:** Use `crypto/rand.Read` for token generation, NEVER `time.Now()`-
> derived entropy. GGID's current `generateCSRFToken()` in `middleware.go`
> uses time-shifted bytes — this is predictable and must be replaced.

---

## 3. Synchronizer Token Pattern

### Why More Secure Than Double-Submit

The synchronizer token pattern stores the CSRF token server-side, bound to the
user's session. The token is provided to the client (typically in a `<meta>` tag
or a separate API call) and must be sent back as a header. The server validates
it against the stored session-bound value.

Key advantages:
- **Not forgeable via subdomain** — The token is server-generated and stored,
  not derivable from any cookie the attacker can plant.
- **Per-form tokens** — Each form or action can have its own token, preventing
  token replay across different operations.
- **Cryptographically random** — No client-generated component.

### Per-Form vs. Per-Session Tokens

| Strategy | Security | Complexity |
|----------|----------|------------|
| Per-session | One token per login session | Low — store once, validate repeatedly |
| Per-form | Unique token per form/action | High — must track valid tokens, expire after use |

Per-form tokens provide defense-in-depth: even if a token leaks, it can only
authorize one specific action. They are recommended for the highest-risk
endpoints (password change, role assignment, MFA enrollment).

### Go Implementation with Session Store

```go
package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// SynchronizerCSRF stores server-side CSRF tokens keyed by session ID.
type SynchronizerCSRF struct {
	store    TokenStore
	header   string
	lifetime time.Duration
}

// TokenStore abstracts the backing store for CSRF tokens.
type TokenStore interface {
	Get(sessionID, token string) (bool, error)
	Set(sessionID, token string, ttl time.Duration) error
	Delete(sessionID, token string) error
}

// RedisTokenStore implements TokenStore using Redis.
type RedisTokenStore struct {
	rdb *redis.Client
}

func (s *RedisTokenStore) Get(sessionID, token string) (bool, error) {
	key := "csrf:" + sessionID + ":" + token
	n, err := s.rdb.Exists(nil, key).Result() //nolint:staticcheck
	return n > 0, err
}

func (s *RedisTokenStore) Set(sessionID, token string, ttl time.Duration) error {
	key := "csrf:" + sessionID + ":" + token
	return s.rdb.Set(nil, key, "1", ttl).Err() //nolint:staticcheck
}

func (s *RedisTokenStore) Delete(sessionID, token string) error {
	key := "csrf:" + sessionID + ":" + token
	return s.rdb.Del(nil, key).Err() //nolint:staticcheck
}

// NewSynchronizerCSRF creates a synchronizer-pattern CSRF middleware.
func NewSynchronizerCSRF(store TokenStore) *SynchronizerCSRF {
	return &SynchronizerCSRF{
		store:    store,
		header:   "X-CSRF-Token",
		lifetime: 30 * time.Minute,
	}
}

// IssueToken generates a new CSRF token for the given session.
// Call this after successful authentication or on page load.
func (sc *SynchronizerCSRF) IssueToken(sessionID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if err := sc.store.Set(sessionID, token, sc.lifetime); err != nil {
		return "", err
	}
	return token, nil
}

// Middleware returns HTTP middleware that validates synchronizer CSRF tokens.
func (sc *SynchronizerCSRF) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		sessionID, ok := getSessionID(r) // from JWT claims or X-Session-ID
		if !ok {
			rejectCSRF(w, "no session for CSRF validation")
			return
		}

		token := r.Header.Get(sc.header)
		if token == "" {
			rejectCSRF(w, "missing CSRF token header")
			return
		}

		exists, err := sc.store.Get(sessionID, token)
		if err != nil || !exists {
			rejectCSRF(w, "invalid or expired CSRF token")
			return
		}

		// Optional: single-use tokens (delete after validation)
		_ = sc.store.Delete(sessionID, token)

		next.ServeHTTP(w, r)
	})
}

func getSessionID(r *http.Request) (string, bool) {
	// In GGID, session ID comes from JWT claims or X-Session-ID header
	sid := r.Header.Get("X-Session-ID")
	return sid, sid != ""
}
```

---

## 4. SameSite Cookie Attribute

### SameSite=Strict vs. Lax vs. None

| Value | Cross-Site GET | Cross-Site POST | Top-Level Navigation | When to Use |
|-------|---------------|-----------------|----------------------|-------------|
| `Strict` | Blocked | Blocked | Blocked | Never send cookies cross-site at all |
| `Lax` | Blocked | Blocked | Sent (top-level GET only) | Default recommendation |
| `None` | Sent | Sent | Sent | Cross-site iframe / OAuth redirects (requires `Secure`) |

### Impact on OAuth Redirects

**`SameSite=Strict` breaks OAuth redirect flows.** When the IdP redirects back
to `iam.example.com/callback?code=xyz` after authorization, the browser treats
this as a cross-site navigation. With `Strict`, the session cookie is NOT sent,
and the user appears logged out — the callback fails.

**`SameSite=Lax` is the right default.** It allows top-level GET navigations
(which includes the OAuth redirect-back), while blocking cross-site `POST`
requests (the primary CSRF vector). Lax provides strong CSRF defense without
breaking standard flows.

### Go Cookie Setting with SameSite

```go
// SetSessionCookie sets a session cookie with secure SameSite defaults.
func SetSessionCookie(w http.ResponseWriter, name, value, domain string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   domain,
		MaxAge:   maxAge,
		HttpOnly: true,           // Prevent JavaScript access
		Secure:   true,           // HTTPS only
		SameSite: http.SameSiteLaxMode, // Allow OAuth redirects, block CSRF POST
	})
}

// SetRefreshTokenCookie sets the refresh token cookie with Strict for maximum
// protection — refresh tokens should never be sent on cross-site navigation.
func SetRefreshTokenCookie(w http.ResponseWriter, value, domain string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		Path:     "/api/v1/auth/refresh", // Scopes to refresh endpoint only
		Domain:   domain,
		MaxAge:   7 * 24 * 3600, // 7 days
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}
```

### SameSite Layered Defense

SameSite is a browser-enforced defense that works independently of CSRF tokens.
Even with perfect SameSite configuration, defense-in-depth recommends also
validating `Origin`/`Referer` and using CSRF tokens — older browsers and some
edge cases may not enforce SameSite correctly.

---

## 5. Origin/Referer Header Validation

### Why the Origin Header Is Reliable

The `Origin` header is set by the browser for cross-origin requests and CANNOT
be overridden by JavaScript. If a CSRF attack originates from `evil.example.com`,
the `Origin` header will be `https://evil.example.com` — the browser guarantees
this.

### Checking Origin Against Allowlist

```go
package middleware

import (
	"net/http"
	"strings"
)

// OriginValidator checks Origin (and falls back to Referer) against an allowlist.
type OriginValidator struct {
	allowedOrigins []string
	// unsafeMethodsWithoutOrigin — for these methods, Origin/Referer is required
}

func NewOriginValidator(origins []string) *OriginValidator {
	return &OriginValidator{allowedOrigins: origins}
}

func (ov *OriginValidator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			// Fallback to Referer header
			referer := r.Header.Get("Referer")
			if referer == "" {
				// No Origin AND no Referer on an unsafe method → reject
				rejectCSRF(w, "missing Origin and Referer headers")
				return
			}
			origin = extractOrigin(referer)
		}

		if !ov.isAllowed(origin) {
			rejectCSRF(w, "origin not allowed: "+origin)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (ov *OriginValidator) isAllowed(origin string) bool {
	for _, allowed := range ov.allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// extractOrigin parses a Referer URL and returns just the origin (scheme://host).
func extractOrigin(referer string) string {
	// Minimal extraction: find scheme://host[:port]
	idx := strings.Index(referer, "://")
	if idx < 0 {
		return ""
	}
	scheme := referer[:idx]
	rest := referer[idx+3:]
	// Find the first / or end of string
	slashIdx := strings.Index(rest, "/")
	if slashIdx >= 0 {
		rest = rest[:slashIdx]
	}
	return scheme + "://" + rest
}
```

### When to Use Origin Validation

Origin validation is an excellent first line of defense because it requires no
token management, no client cooperation, and works for all unsafe methods. It
should be combined with token-based defense for defense-in-depth.

---

## 6. OAuth/OIDC CSRF via `state` Parameter

### Why the `state` Parameter Is Critical

The OAuth `state` parameter serves two purposes:
1. **CSRF defense** — Binds the authorization request to the user's session.
2. **Correlation** — Allows the client to match the callback to the original request.

Without `state` validation, an attacker can inject their own authorization code
into the victim's session (login CSRF), or trick the callback endpoint into
processing a code from a different flow.

### Binding State to Session

```go
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
)

// StateManager generates, stores, and validates OAuth state parameters.
type StateManager struct {
	store StateStore // Redis or in-memory
}

type StateStore interface {
	Set(ctx context.Context, sessionID, state string) error
	ValidateAndDelete(ctx context.Context, sessionID, state string) (bool, error)
}

// GenerateState creates a random state and binds it to the session.
func (sm *StateManager) GenerateState(ctx context.Context, sessionID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(raw)
	if err := sm.store.Set(ctx, sessionID, state); err != nil {
		return "", fmt.Errorf("store state: %w", err)
	}
	return state, nil
}

// ValidateState checks that the state parameter matches the one stored for the session.
func (sm *StateManager) ValidateState(ctx context.Context, sessionID, state string) bool {
	if state == "" || sessionID == "" {
		return false
	}
	valid, err := sm.store.ValidateAndDelete(ctx, sessionID, state)
	if err != nil {
		return false
	}
	return valid // single-use: deleted after validation
}

// OAuthCallbackHandler validates the state before exchanging the code.
func OAuthCallbackHandler(sm *StateManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		if code == "" || state == "" {
			http.Error(w, "missing code or state", http.StatusBadRequest)
			return
		}

		sessionID, ok := getSessionID(r)
		if !ok {
			http.Error(w, "no session", http.StatusUnauthorized)
			return
		}

		if !sm.ValidateState(r.Context(), sessionID, state) {
			http.Error(w, "invalid state parameter", http.StatusBadRequest)
			return
		}

		// State is valid — proceed with code exchange
		next.ServeHTTP(w, r)
	})
}
```

### PKCE as CSRF Defense for Code Interception

PKCE (Proof Key for Code Exchange) is primarily designed to protect
authorization codes in transit for public clients. However, it also provides
CSRF-like defense: even if an attacker intercepts the authorization code, they
cannot exchange it without the PKCE verifier.

```go
// GeneratePKCEPair creates a code_verifier and code_challenge (S256).
func GeneratePKCEPair() (verifier, challenge string, err error) {
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(verifierBytes)

	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// At the token exchange endpoint, validate the PKCE verifier:
func ValidatePKCE(verifier, challenge, method string) bool {
	if method != "S256" || verifier == "" {
		return false
	}
	h := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}
```

> **Note:** PKCE is NOT a replacement for the `state` parameter. They protect
> against different threats. Use both together.

---

## 7. GGID Gateway CSRF Middleware Analysis

### Existing CSRF Protection

GGID's gateway already has a `CSRFProtect` middleware in
`services/gateway/internal/middleware/middleware.go` (lines 152-216). It
implements the **double-submit cookie pattern**:

**What works:**
- Safe methods (GET/HEAD/OPTIONS) receive the CSRF cookie without validation.
- Unsafe methods (POST/PUT/PATCH/DELETE) require the `X-CSRF-Token` header to
  match the `csrf_token` cookie.
- Comparison uses `subtle.ConstantTimeCompare` (timing-attack safe).
- Cookie has `HttpOnly: false` (correct — JS must read it for double-submit).
- Cookie has `Secure: true` and `SameSite: http.SameSiteLaxMode`.

**Critical vulnerabilities found:**

1. **Predictable token generation (CRITICAL)**

   ```go
   func generateCSRFToken() string {
       b := make([]byte, 32)
       for i := range b {
           b[i] = byte(time.Now().UnixNano() >> uint(i))
       }
       hash := sha256.Sum256(b)
       return base64.RawURLEncoding.EncodeToString(hash[:])
   }
   ```

   This uses `time.Now().UnixNano()` shifted by byte index — the output is
   entirely predictable. An attacker who knows the approximate server time can
   reconstruct the token. This must be replaced with `crypto/rand.Read`.

2. **No Origin/Referer validation** — The middleware does not validate request
   origin, relying solely on the double-submit token. An attacker who can plant
   a cookie (via subdomain) defeats the defense entirely.

3. **CORS allows `*` by default** — `DefaultCORSConfig()` sets
   `AllowedOrigins: ["*"]`. When combined with `AllowCredentials: true` in the
   per-tenant CORS middleware, this creates a dangerous configuration where any
   origin can make credentialed requests.

4. **OAuth `state` is passed through but not validated** — The OAuth server
   (`oauth_service.go`) checks `if req.State == ""` and rejects empty state, but
   it does NOT validate that the state matches a value previously issued for
   this session. The state is simply echoed back in the redirect URL. This
   means the `state` parameter provides zero actual CSRF protection.

5. **No SameSite on session/auth cookies** — The gateway does not set
   SameSite attributes on authentication-related cookies. Only the CSRF cookie
   and sticky/canary cookies have `SameSite: Lax`. Session cookies set by the
   auth service have no explicit SameSite attribute.

6. **CSRF middleware not in the main chain** — `CSRFProtect` exists but may not
   be wired into the default middleware chain for all unsafe-method routes.
   It must be verified that every state-changing endpoint passes through it.

### Cookie Handling in Gateway

| Cookie | SameSite | HttpOnly | Secure | Same Path | Notes |
|--------|----------|----------|--------|-----------|-------|
| `csrf_token` | Lax | false | true | `/` | Correct for double-submit |
| `sticky` (canary) | Lax | true | true | `/` | Infra cookie, low risk |
| `canary` | Lax | true | true | `/` | Infra cookie, low risk |
| Session/JWT cookies | Not set | ? | ? | ? | **GAP: no explicit SameSite** |

---

## 8. Gap Analysis and Recommendations

### What GGID Currently Lacks

| Gap | Severity | Impact |
|-----|----------|--------|
| Predictable CSRF token entropy | **CRITICAL** | Attacker can forge CSRF tokens |
| OAuth `state` not validated against session | **HIGH** | Login CSRF via OAuth flow |
| No Origin/Referer validation | **HIGH** | No defense-in-depth against CSRF |
| No SameSite on auth/session cookies | **HIGH** | Cookies sent on cross-site POST |
| CORS `*` with credentials in tenant config | **MEDIUM** | Any origin can make credentialed requests |
| No per-form CSRF tokens for high-risk endpoints | **MEDIUM** | Token replay across actions |
| CSRFProtect may not cover all routes | **MEDIUM** | Some endpoints unprotected |

### Implementation Roadmap

**Action Item 1: Fix CSRF Token Generation (CRITICAL, effort: 1 hour)**

Replace `generateCSRFToken()` with `crypto/rand`-based generation:

```go
import "crypto/rand"

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
```

**Action Item 2: Implement OAuth `state` Validation (HIGH, effort: 4 hours)**

- Store the generated `state` in Redis keyed by session ID before redirecting
  to the authorization endpoint.
- On callback, validate `state` against the stored value and delete it
  (single-use).
- Reject callbacks where `state` is missing, doesn't match, or has expired.

**Action Item 3: Add Origin/Referer Validation Middleware (HIGH, effort: 2 hours)**

- Add an `OriginValidator` middleware to the gateway chain, positioned before
  CSRF token validation.
- Configure allowed origins per-tenant (reuse the `TenantCORSStore`).
- Apply to all unsafe methods (POST/PUT/PATCH/DELETE).

**Action Item 4: Set SameSite on All Auth Cookies (HIGH, effort: 2 hours)**

- Audit all `http.SetCookie` calls in the gateway and auth service.
- Set `SameSite: Lax` on session/JWT cookies (allows OAuth redirects).
- Set `SameSite: Strict` on refresh token cookies (never needed cross-site).
- Ensure `Secure: true` and `HttpOnly: true` on all auth cookies.

**Action Item 5: Harden CORS Configuration (MEDIUM, effort: 2 hours)**

- Remove `*` from `DefaultCORSConfig()` — require explicit origin lists.
- When `AllowCredentials: true`, never use wildcard origins.
- Validate that the per-tenant CORS store has origins configured before enabling
  credentials.

**Action Item 6: Synchronizer Token for High-Risk Endpoints (LOW, effort: 1 day)**

For the most sensitive endpoints (password change, role assignment, MFA
enrollment, session revocation), add server-side synchronizer tokens in addition
to the existing double-submit pattern. Use Redis (already available) as the
token store, keyed by session ID.

### Summary Defense-in-Depth Stack

```
Request → TLS → SameSite Cookie → Origin/Referer Check → Double-Submit Token
  → Synchronizer Token (high-risk only) → JWT Auth → Rate Limit → Handler
```

Each layer blocks different attack vectors:
- **SameSite** — Browser-level CSRF prevention for modern browsers
- **Origin/Referer** — Verifies the request origin without token management
- **Double-Submit** — Stateless CSRF token for all unsafe methods
- **Synchronizer** — Server-side per-session tokens for critical actions
- **OAuth `state`** — CSRF defense specific to OAuth/OIDC flows

Together, these layers ensure that even if one defense fails, others provide
coverage — the principle of defense-in-depth applied to CSRF.

---

## References

- OWASP CSRF Prevention Cheat Sheet
- RFC 6749 — The OAuth 2.0 Authorization Framework (Section 10.12: CSRF)
- RFC 9700 (OAuth 2.0 Security Best Current Practice) — mandates `state` and PKCE
- MDN: SameSite cookie attribute
- Go standard library: `net/http.Cookie` SameSite field
- Google Browser Security Research: SameSite cookie behavior

---

*Document version: 1.0 | GGID project | Apache 2.0 License*
