# Cookie Security for IAM Systems

> **Research Document** — GGID IAM Suite
> **Topic**: Cookie attributes, prefix enforcement, CSRF defense, session lifecycle, and OAuth/OIDC implications
> **Audience**: Security engineers, backend developers, Platform/SRE teams

---

## Table of Contents

1. [Cookie Attributes Security](#1-cookie-attributes-security)
2. [SameSite Deep Dive](#2-samesite-deep-dive)
3. [`__Host-` Prefix](#3-__host--prefix)
4. [`__Secure-` Prefix](#4-__secure--prefix)
5. [Cookie Prefix Attacks](#5-cookie-prefix-attacks)
6. [Session Cookie vs CSRF Token Cookie](#6-session-cookie-vs-csrf-token-cookie)
7. [OAuth State Cookie](#7-oauth-state-cookie)
8. [Cookie-Based Session Revocation](#8-cookie-based-session-revocation)
9. [Multiple Cookie Jars (Multi-Tenant)](#9-multiple-cookie-jars-multi-tenant)
10. [GGID Cookie Audit](#10-ggid-cookie-audit)
11. [Gap Analysis & Recommendations](#11-gap-analysis--recommendations)

---

## 1. Cookie Attributes Security

Every `Set-Cookie` header can carry attributes that control how the browser handles the cookie. For IAM systems where cookies transport session identifiers, CSRF tokens, and OAuth state, incorrect attributes lead directly to account takeover.

### 1.1 HttpOnly

**Purpose**: Prevents JavaScript (`document.cookie`) from reading the cookie value. This is the primary defense against **cookie theft via XSS**.

- If `HttpOnly` is absent, any XSS payload can exfiltrate the session cookie to an attacker-controlled server.
- HttpOnly does NOT prevent the browser from *sending* the cookie — it only blocks the `document.cookie` API.

### 1.2 Secure

**Purpose**: The cookie is only transmitted over HTTPS connections.

- Without `Secure`, a cookie set on `https://app.example.com` will also be sent to `http://app.example.com` (if the user types HTTP or is downgraded by an MITM).
- In production IAM, **every cookie should have `Secure`**. The only exception is local development over `localhost`.

### 1.3 SameSite

**Purpose**: Controls whether the cookie is sent on cross-site requests (CSRF defense).

- `SameSite=Strict` — never sent on cross-site requests.
- `SameSite=Lax` — sent on top-level navigations (GET), NOT on cross-site POST/PUT/DELETE.
- `SameSite=None` — always sent cross-site, **requires `Secure`**.

See [Section 2](#2-samesite-deep-dive) for the full analysis.

### 1.4 Path

**Purpose**: Restricts the cookie to a URL path prefix.

- Setting `Path=/admin` means the cookie is only sent to `/admin/*` requests.
- **Pitfall**: Path is matched by prefix, not exact match. A cookie with `Path=/api` is sent to `/api`, `/api-users`, `/apiv2`. Always end scoped paths with `/` (e.g. `Path=/api/`).
- For IAM session cookies, use `Path=/` unless you have a strong reason to scope.

### 1.5 Domain

**Purpose**: Controls which hosts the cookie is sent to.

- **No Domain attribute**: cookie is "host-only" — sent only to the exact host that set it. This is the most secure.
- **Domain=.example.com**: cookie is sent to ALL subdomains (`app.example.com`, `blog.example.com`, `evil.example.com`). This is dangerous if any subdomain is compromised or user-controlled.
- **IAM rule**: Never set `Domain` on session cookies unless you specifically need cross-subdomain SSO.

### 1.6 Max-Age vs Expires

| Attribute | Description |
|-----------|-------------|
| `Max-Age` | Seconds until expiry. Preferred — resilient to clock skew. |
| `Expires` | Absolute timestamp (HTTP date format). Fallback for older browsers. |

- If both are set, `Max-Age` takes precedence.
- For session cookies, use `Max-Age` with a value aligned to your token TTL (e.g. access token: 15 min, refresh token: 30 days).
- **Session cookies** (browser closes → cookie deleted) are created by setting *neither* attribute. Do NOT use these for IAM — sessions should expire deterministically.

### 1.7 Go Code: Setting All Attributes Correctly

```go
package cookieutil

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

// SessionCookieConfig holds the security parameters for an IAM session cookie.
type SessionCookieConfig struct {
	Name     string
	Value    string
	Path     string
	Domain   string // leave empty for host-only
	MaxAge   int    // seconds; 0 means session cookie
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// SetSecureCookie writes a cookie with explicit security attributes.
func SetSecureCookie(w http.ResponseWriter, cfg SessionCookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.Name,
		Value:    cfg.Value,
		Path:     cfg.Path,
		Domain:   cfg.Domain, // empty = host-only (most secure)
		MaxAge:   cfg.MaxAge,
		Expires:  time.Now().Add(time.Duration(cfg.MaxAge) * time.Second),
		Secure:   cfg.Secure,
		HttpOnly: cfg.HttpOnly,
		SameSite: cfg.SameSite,
	})
}

// ClearCookie deletes a cookie by setting Max-Age=0 and an expired date.
func ClearCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:    name,
		Value:   "",
		Path:    path,
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
}

// GenerateCookieValue creates a cryptographically random cookie value.
func GenerateCookieValue() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
```

---

## 2. SameSite Deep Dive

SameSite is the most nuanced cookie attribute for IAM systems. Choosing the wrong value either breaks OAuth/OIDC flows or leaves the system vulnerable to CSRF.

### 2.1 SameSite=Strict

- **Behavior**: The cookie is **never** sent on cross-site requests, including top-level navigations (clicking a link from another site).
- **Security**: Strongest CSRF protection. Even `<a href="https://bank.com/transfer">` from an attacker site won't carry the cookie.
- **Problem for IAM**: Breaks **OAuth/OIDC redirect flows**. When the IdP redirects back to `https://app.example.com/callback?code=xxx`, the browser treats this as a cross-site navigation and will NOT send `SameSite=Strict` cookies. The user appears logged out.
- **Use case**: Internal admin panels with no external redirect flows.

### 2.2 SameSite=Lax (Browser Default)

- **Behavior**: Cookie is sent on **safe** top-level navigations (GET via link click, address bar). NOT sent on cross-site POST/PUT/DELETE or iframe navigations.
- **Security**: Protects against CSRF POST attacks. A cross-site form POST to `/transfer` will not carry the cookie.
- **IAM compatibility**: Works for OAuth/OIDC redirect callbacks because the IdP redirect is a top-level GET navigation. The callback endpoint receives the cookie.
- **Default**: Modern browsers (Chrome 80+) default to `Lax` when SameSite is unspecified.

### 2.3 SameSite=None + Secure

- **Behavior**: Cookie is sent on **all** cross-site requests, including POST and iframe.
- **Security**: Weakest. The `Secure` flag is mandatory — browsers reject `SameSite=None` without `Secure`.
- **Use case**: Third-party cookies, SSO iframe-based silent renew, cross-site embedded widgets.
- **Warning**: Chrome is progressively blocking third-party cookies. `SameSite=None` cookies in third-party contexts may be blocked entirely in the near future.

### 2.4 OAuth/OIDC Flow Compatibility Matrix

| Flow | Strict | Lax | None+Secure |
|------|--------|-----|-------------|
| OAuth redirect callback (GET) | Broken | Works | Works |
| OIDC silent renew (iframe) | Broken | Broken | Works |
| Same-site form POST | Works | Works | Works |
| Cross-site form POST | Blocked | Blocked | Works |
| SAML POST binding | Broken | Blocked | Works |

### 2.5 Go Code: SameSite Selection Per Endpoint

```go
package cookieutil

import "net/http"

// SameSiteForEndpoint returns the appropriate SameSite mode based on the
// request path and whether the flow involves cross-site redirects.
func SameSiteForEndpoint(path string, isHTTPS bool) http.SameSite {
	switch {
	// OAuth/OIDC callback endpoints — must allow top-level cross-site GET.
	case path == "/oauth/callback" || path == "/oidc/callback":
		return http.SameSiteLaxMode // works for redirect GET

	// Silent renew iframe — needs None for cross-site iframe access.
	case path == "/oauth/silent-renew":
		if isHTTPS {
			return http.SameSiteNoneMode
		}
		// Can't use None without HTTPS — fall back to Lax.
		return http.SameSiteLaxMode

	// API endpoints that receive cross-site POST — strictest safe default.
	case path == "/api/" || path == "/api/v1/":
		return http.SameSiteLaxMode // blocks cross-site POST, allows same-site POST

	// Admin panel — no external redirects needed.
	case path == "/admin/":
		return http.SameSiteStrictMode

	// Default: Lax is safe for most flows.
	default:
		return http.SameSiteLaxMode
	}
}
```

---

## 3. `__Host-` Prefix

The `__Host-` prefix is a browser-enforced security mechanism defined in [RFC 6265bis](https://datatracker.ietf.org/doc/html/draft-ietf-httpbis-rfc6265bis). When a cookie name starts with `__Host-`, the browser **rejects** the cookie unless ALL of the following are true:

1. `Secure` is set.
2. `Path=/` (root path).
3. **No `Domain` attribute** (host-only).

### 3.1 Why IAM Session Cookies Should Use `__Host-`

Without the prefix, a compromised subdomain can inject a cookie on the parent domain:

```
Attacker controls evil.example.com
Attacker sets: Set-Cookie: session=attacker_value; Domain=.example.com; Path=/
Victim's browser now has TWO "session" cookies for example.com
Server receives the attacker's cookie first (or in unpredictable order)
```

With `__Host-session`, the browser guarantees:
- The cookie was set by the exact host (no Domain).
- It's only sent over HTTPS.
- It's scoped to the root path.
- A subdomain **cannot** set or overwrite a `__Host-` prefixed cookie.

### 3.2 Go Code: `__Host-` Prefixed Cookies

```go
package cookieutil

import "net/http"

// SetHostPrefixedCookie sets a __Host- prefixed cookie with enforced attributes.
// The browser will reject this cookie if Secure is false, Path is not "/",
// or Domain is set.
func SetHostPrefixedCookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-" + name,
		Value:    value,
		Path:     "/",           // REQUIRED for __Host-
		Domain:   "",            // MUST be empty for __Host-
		MaxAge:   maxAge,
		Secure:   true,          // REQUIRED for __Host-
		HttpOnly: true,          // recommended for session cookies
		SameSite: http.SameSiteLaxMode,
	})
}

// Usage in login handler:
// SetHostPrefixedCookie(w, "session", sessionToken, 900) // 15 minutes
// Browser receives: __Host-session=<value>; Secure; HttpOnly; SameSite=Lax; Path=/
```

---

## 4. `__Secure-` Prefix

The `__Secure-` prefix is less strict than `__Host-`. When a cookie name starts with `__Secure-`, the browser only requires:

1. `Secure` is set.

That's it. `Path` and `Domain` are unrestricted.

### 4.1 When to Use `__Secure-` vs `__Host-`

| Scenario | Prefix | Reason |
|----------|--------|--------|
| Session cookie (single host) | `__Host-` | Strictest — no subdomain injection |
| Cross-subdomain SSO cookie | `__Secure-` | Needs `Domain=.example.com` — `__Host-` forbids Domain |
| CSRF double-submit cookie | `__Host-` | Should be host-scoped |
| Analytics/tracking cookie | `__Secure-` | May need non-root Path |
| Third-party iframe cookie | `__Secure-` | Needs Domain or non-root Path |

**Rule of thumb**: Use `__Host-` unless you specifically need `Domain` or non-root `Path`. Then use `__Secure-`.

### 4.2 Go Code: `__Secure-` Prefixed Cookies

```go
package cookieutil

import "net/http"

// SetSecurePrefixedCookie sets a __Secure- prefixed cookie.
// Only requires Secure=true. Domain and Path are allowed.
func SetSecurePrefixedCookie(w http.ResponseWriter, name, value, domain, path string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "__Secure-" + name,
		Value:    value,
		Path:     path,
		Domain:   domain,
		MaxAge:   maxAge,
		Secure:   true,          // REQUIRED for __Secure-
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Example: cross-subdomain SSO cookie
// SetSecurePrefixedCookie(w, "sso", ssoToken, ".example.com", "/", 86400)
```

---

## 5. Cookie Prefix Attacks

### 5.1 Subdomain Cookie Injection

If an attacker controls any subdomain of the cookie's scope, they can inject cookies that appear on the parent domain:

```
Attacker controls: blog.example.com (compromised WordPress plugin)
Attacker sets: Set-Cookie: session=evil; Domain=.example.com; Path=/
User visits app.example.com → browser sends attacker's "session" cookie
```

**Defense**: Use `__Host-` prefix. Browsers enforce host-only (no Domain) for prefixed cookies, making subdomain injection impossible.

### 5.2 Cookie Shadowing

When multiple cookies share the same name, the browser sends them all in the `Cookie` header:

```
Cookie: session=legitimate; session=evil; session=legitimate
```

The server's `r.Cookie("session")` returns the **first** one, but the ordering is implementation-dependent and can vary between browsers.

**Defense**:
1. Use `__Host-` prefix to prevent subdomains from creating same-named cookies.
2. When reading cookies, validate ALL values, not just the first.
3. Reject requests with duplicate cookie names.

### 5.3 Cookie Tossing via Wildcard DNS

If the IAM system uses wildcard DNS (`*.app.example.com` resolves to the same server), an attacker who can register `evil-app.app.example.com` can toss cookies:

```
evil-app.app.example.com sets: Set-Cookie: __Host-session=evil
Wait — __Host- prefix PREVENTS this because the cookie is host-scoped.
Without __Host-: evil-app.app.example.com sets session=evil; Domain=.app.example.com
```

### 5.4 Go Code: Cookie Prefix Enforcement

```go
package cookieutil

import (
	"net/http"
	"strings"
)

// ReadHostPrefixedCookie reads a __Host- prefixed cookie, rejecting requests
// that contain duplicate or shadowed cookies with the same base name.
func ReadHostPrefixedCookie(r *http.Request, name string) (string, error) {
	fullName := "__Host-" + name
	var cookies []string
	for _, c := range r.Cookies() {
		if c.Name == fullName {
			cookies = append(cookies, c.Value)
		}
	}
	switch len(cookies) {
	case 0:
		return "", http.ErrNoCookie
	case 1:
		return cookies[0], nil
	default:
		// Duplicate cookies detected — possible shadowing attack.
		// Reject the request entirely.
		return "", ErrDuplicateCookie
	}
}

// ValidateNoUnprefixed checks that no session-related cookies exist without
// the __Host- or __Secure- prefix. Call this on incoming requests to detect
// injection attempts.
func ValidateNoUnprefixed(r *http.Request, names []string) error {
	for _, name := range names {
		for _, c := range r.Cookies() {
			if c.Name == name {
				// Unprefixed version of a sensitive cookie — reject.
				return ErrUnprefixedCookie
			}
		}
	}
	return nil
}

var ErrDuplicateCookie = &cookieError{"duplicate cookie detected — possible shadowing attack"}
var ErrUnprefixedCookie = &cookieError{"unprefixed sensitive cookie detected"}

type cookieError struct{ msg string }

func (e *cookieError) Error() string { return e.msg }

// IsPrefixed returns true if the cookie name has a security prefix.
func IsPrefixed(name string) bool {
	return strings.HasPrefix(name, "__Host-") || strings.HasPrefix(name, "__Secure-")
}
```

---

## 6. Session Cookie vs CSRF Token Cookie

IAM systems need **two distinct cookies** with **different security attributes**. Mixing them up is a common vulnerability.

### 6.1 Session Cookie

| Attribute | Value | Reason |
|-----------|-------|--------|
| `HttpOnly` | `true` | Prevent XSS from stealing the session |
| `Secure` | `true` | HTTPS only |
| `SameSite` | `Lax` | Allow OAuth redirect, block CSRF POST |
| `Path` | `/` | Available to all endpoints |
| `Domain` | (none) | Host-only — use `__Host-` |
| Prefix | `__Host-` | Prevent subdomain injection |
| Lifetime | Aligned with access token TTL (15 min) | Short |

### 6.2 CSRF Double-Submit Cookie

| Attribute | Value | Reason |
|-----------|-------|--------|
| `HttpOnly` | `false` | **JavaScript must read this** to set the X-CSRF-Token header |
| `Secure` | `true` | HTTPS only |
| `SameSite` | `Lax` | Match session cookie behavior |
| `Path` | `/` | Same scope as session |
| `Domain` | (none) | Host-only — use `__Host-` |
| Prefix | `__Host-` | Prevent subdomain injection |
| Lifetime | Session or short-lived | Regenerated frequently |

**Why different?** The session cookie must be HttpOnly (XSS can't steal it). The CSRF cookie must NOT be HttpOnly (JS reads it to set the header). If both are HttpOnly, the double-submit pattern doesn't work — JS can't extract the token. If neither is HttpOnly, XSS can steal the session.

### 6.3 Go Code: Both Cookie Types

```go
package cookieutil

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
)

// SetSessionCookie writes the IAM session cookie with full security attributes.
func SetSessionCookie(w http.ResponseWriter, sessionToken string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-ggid_session",
		Value:    sessionToken,
		Path:     "/",
		Domain:   "", // host-only
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true, // JS CANNOT read this
		SameSite: http.SameSiteLaxMode,
	})
}

// SetCSRFCookie writes the CSRF double-submit cookie.
// HttpOnly MUST be false — the frontend JavaScript reads this cookie and
// sends the value in the X-CSRF-Token header on state-changing requests.
func SetCSRFCookie(w http.ResponseWriter, csrfToken string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-ggid_csrf",
		Value:    csrfToken,
		Path:     "/",
		Domain:   "", // host-only
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: false, // JS MUST read this
		SameSite: http.SameSiteLaxMode,
	})
}

// GenerateCSRFToken creates a random CSRF token.
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	h := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ValidateCSRF checks that the X-CSRF-Token header matches the CSRF cookie.
// Uses constant-time comparison to prevent timing attacks.
func ValidateCSRF(r *http.Request) bool {
	cookie, err := r.Cookie("__Host-ggid_csrf")
	if err != nil || cookie.Value == "" {
		return false
	}
	headerToken := r.Header.Get("X-CSRF-Token")
	if headerToken == "" {
		return false
	}
	// Constant-time comparison.
	return subtleEqual(cookie.Value, headerToken)
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
```

---

## 7. OAuth State Cookie

The OAuth `state` parameter prevents CSRF on the authorization code flow. It can be stored server-side (in Redis) or in a short-lived cookie. The cookie approach is simpler for stateless deployments.

### 7.1 Cookie Requirements for OAuth State

| Attribute | Value | Reason |
|-----------|-------|--------|
| `HttpOnly` | `true` | JS doesn't need to read it |
| `Secure` | `true` | HTTPS only |
| `SameSite` | `Lax` | **NOT Strict** — the IdP redirect is a cross-site GET navigation |
| `MaxAge` | 600 (10 min) | Short-lived — state should expire quickly |
| Prefix | `__Host-` | Prevent subdomain injection |

**Critical**: If you use `SameSite=Strict`, the cookie will NOT be sent when the IdP redirects back to `/callback`. The user will see an error because the state validation fails. This is the #1 SameSite mistake in OAuth implementations.

### 7.2 Go Code: OAuth State Cookie

```go
package oauthutil

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

// SetOAuthStateCookie stores the OAuth state parameter in a short-lived cookie.
// The cookie is read back on the /callback endpoint to validate the state.
func SetOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-oauth_state",
		Value:    state,
		Path:     "/",
		Domain:   "",
		MaxAge:   600, // 10 minutes
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode, // NOT Strict — breaks redirect callback
	})
}

// ValidateOAuthStateCookie reads the state cookie and compares it to the
// state parameter from the callback URL. Returns true if they match.
func ValidateOAuthStateCookie(r *http.Request, urlState string) bool {
	cookie, err := r.Cookie("__Host-oauth_state")
	if err != nil || cookie.Value == "" {
		return false
	}
	if urlState == "" {
		return false
	}
	// Constant-time comparison.
	if len(cookie.Value) != len(urlState) {
		return false
	}
	var result byte
	for i := 0; i < len(cookie.Value); i++ {
		result |= cookie.Value[i] ^ urlState[i]
	}
	return result == 0
}

// ClearOAuthStateCookie removes the state cookie after successful validation.
func ClearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "__Host-oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// GenerateState creates a cryptographically random OAuth state value.
func GenerateState() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
```

### 7.3 Flow Diagram

```
1. User clicks "Login with Google"
2. Server generates state=xyz, sets __Host-oauth_state=xyz cookie
3. Server redirects to Google: /authorize?state=xyz&client_id=...
4. User authenticates at Google
5. Google redirects to /callback?code=abc&state=xyz
6. Server reads __Host-oauth_state cookie (value=xyz)
7. Server compares cookie state (xyz) with URL state (xyz) → match
8. Server clears state cookie, exchanges code for tokens
```

---

## 8. Cookie-Based Session Revocation

### 8.1 The Revocation Problem

A cookie lives on the client until it expires or the user clears it. **You cannot force a browser to delete a cookie** — you can only send a `Set-Cookie` with `Max-Age=0`, which only works if the browser makes another request.

This means: if a user's session is revoked server-side, the cookie still exists on the client until the next request. The cookie will be sent, but the server must **reject it** based on its session table.

### 8.2 Server-Side Session Table Pattern

The cookie value is NOT the session itself — it's an **opaque identifier** that maps to a server-side session record:

```
Cookie: __Host-ggid_session = abc123...
                         ↓
Server session table (PostgreSQL):
  session_id | user_id | expires_at | revoked_at | device_info
  -----------+---------+-------------+------------+------------
  abc123     | user-1  | 2025-01-15  | NULL       | Chrome/macOS
```

On every request:
1. Read cookie → extract opaque session ID.
2. Look up session in the table.
3. Check `revoked_at IS NULL` and `expires_at > NOW()`.
4. If valid, proceed. If not, return 401.

### 8.3 Go Code: Server-Side Session Management

```go
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// SessionStore is the interface for server-side session persistence.
type SessionStore interface {
	Get(ctx context.Context, tokenHash string) (*Session, error)
	Create(ctx context.Context, s *Session) error
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// Session represents a server-side session record.
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	IPAddress string
	UserAgent string
}

var ErrSessionInvalid = errors.New("session is invalid or revoked")

// CreateSession issues a new session: generates a random token, stores its
// hash server-side, and sets the opaque token in the cookie.
func CreateSession(w http.ResponseWriter, store SessionStore, userID uuid.UUID) (string, error) {
	token := generateOpaqueToken()
	tokenHash := hashToken(token)

	sess := &Session{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		IPAddress: "",
		UserAgent: "",
	}

	ctx := context.Background()
	if err := store.Create(ctx, sess); err != nil {
		return "", err
	}

	// Set cookie with the OPAQUE token (not the hash).
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-ggid_session",
		Value:    token,
		Path:     "/",
		MaxAge:   900,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return token, nil
}

// ValidateSession checks the cookie against the server-side session table.
// Returns the session if valid, or ErrSessionInvalid if revoked/expired.
func ValidateSession(r *http.Request, store SessionStore) (*Session, error) {
	cookie, err := r.Cookie("__Host-ggid_session")
	if err != nil {
		return nil, ErrSessionInvalid
	}

	tokenHash := hashToken(cookie.Value)
	sess, err := store.Get(r.Context(), tokenHash)
	if err != nil {
		return nil, ErrSessionInvalid
	}

	if sess.RevokedAt != nil {
		return nil, ErrSessionInvalid
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, ErrSessionInvalid
	}

	return sess, nil
}

// RevokeSession marks the session as revoked server-side. The cookie still
// exists on the client but will be rejected on the next request.
func RevokeSession(r *http.Request, store SessionStore) error {
	cookie, err := r.Cookie("__Host-ggid_session")
	if err != nil {
		return nil // already logged out
	}
	tokenHash := hashToken(cookie.Value)
	return store.Revoke(r.Context(), tokenHash)
}

// RevokeAllForUser logs out the user across ALL devices.
func RevokeAllForUser(ctx context.Context, store SessionStore, userID uuid.UUID) error {
	return store.RevokeAllForUser(ctx, userID)
}

func generateOpaqueToken() string {
	// In production, use crypto/rand (see cookieutil.GenerateCookieValue).
	return uuid.New().String()
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
```

### 8.4 Session Sliding Expiration

For long-lived sessions, slide the expiration on each valid request:

```go
func SlideExpiration(ctx context.Context, store SessionStore, sess *Session) error {
	sess.ExpiresAt = time.Now().Add(30 * time.Minute) // extend by 30 min
	return store.Create(ctx, sess) // upsert
}
```

---

## 9. Multiple Cookie Jars (Multi-Tenant)

In a multi-tenant IAM system, multiple tenants may share the same domain (e.g. `app.ggid.dev`). Cookie names must be namespaced per tenant to prevent collision.

### 9.1 The Collision Problem

Without namespacing:
```
Tenant A login → Cookie: session=tenant_a_session
Tenant B login → Cookie: session=tenant_b_session (OVERWRITES tenant A!)
```

The user can't be logged into two tenants simultaneously.

### 9.2 Tenant-Scoped Cookie Names

Include the tenant ID in the cookie name:

```
Cookie: __Host-session_tenant_001 = <value>
Cookie: __Host-session_tenant_002 = <value>
```

### 9.3 Go Code: Tenant-Scoped Cookies

```go
package tenant

import (
	"fmt"
	"net/http"
	"strings"
)

// TenantCookieName builds a tenant-scoped cookie name with the __Host- prefix.
func TenantCookieName(baseName, tenantID string) string {
	// Sanitize tenant ID for cookie name safety.
	safe := strings.ReplaceAll(tenantID, "-", "_")
	return fmt.Sprintf("__Host-%s_tenant_%s", baseName, safe)
}

// SetTenantSessionCookie sets a session cookie scoped to a specific tenant.
func SetTenantSessionCookie(w http.ResponseWriter, tenantID, sessionToken string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     TenantCookieName("session", tenantID),
		Value:    sessionToken,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadTenantSessionCookie reads the session cookie for a specific tenant.
func ReadTenantSessionCookie(r *http.Request, tenantID string) (string, error) {
	name := TenantCookieName("session", tenantID)
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// ListTenantSessions returns all tenant session cookies present in the request.
// Useful for detecting which tenants the user is logged into.
func ListTenantSessions(r *http.Request) map[string]string {
	sessions := make(map[string]string)
	prefix := "__Host-session_tenant_"
	for _, cookie := range r.Cookies() {
		if strings.HasPrefix(cookie.Name, prefix) {
			tenantID := strings.TrimPrefix(cookie.Name, prefix)
			sessions[tenantID] = cookie.Value
		}
	}
	return sessions
}

// ClearTenantSessionCookie removes a specific tenant's session cookie.
func ClearTenantSessionCookie(w http.ResponseWriter, tenantID string) {
	http.SetCookie(w, &http.Cookie{
		Name:   TenantCookieName("session", tenantID),
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}
```

### 9.4 Subdomain-Based Tenant Isolation

An alternative approach uses subdomains: `tenant-a.ggid.dev` and `tenant-b.ggid.dev`. With host-only cookies (`__Host-` prefix), each subdomain gets its own cookie jar automatically. No namespacing needed. This is the preferred approach when subdomain routing is available.

---

## 10. GGID Cookie Audit

This section audits the actual cookie handling in the GGID codebase.

### 10.1 Gateway Middleware — CSRF Cookie (`middleware.go`)

**File**: `services/gateway/internal/middleware/middleware.go` (lines 191–202)

```go
func setCSRFCookie(w http.ResponseWriter) {
	token := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: false, // Must be readable by JavaScript for double-submit
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
```

**Assessment**:

| Attribute | Current | Recommended | Status |
|-----------|---------|-------------|--------|
| Name | `csrf_token` | `__Host-ggid_csrf` | Missing prefix |
| HttpOnly | `false` | `false` | Correct (JS reads it) |
| Secure | `true` | `true` | Correct |
| SameSite | `Lax` | `Lax` | Correct |
| Path | `/` | `/` | Correct |
| Domain | (unset) | (unset) | Correct (host-only) |
| MaxAge | 3600 (1 hour) | 3600 | Reasonable |

**Finding**: The CSRF cookie is missing the `__Host-` prefix. This means a compromised subdomain could inject a `csrf_token` cookie with an attacker-controlled value, defeating the double-submit pattern.

### 10.2 Gateway Middleware — Sticky Session Cookie (`sticky.go`)

**File**: `services/gateway/internal/middleware/sticky.go` (lines 116–124)

```go
http.SetCookie(w, &http.Cookie{
	Name:     sr.config.CookieName, // "ggid_sticky"
	Value:    key,
	Path:     "/",
	MaxAge:   int(sr.config.TTL.Seconds()),
	HttpOnly: true,
	SameSite: http.SameSiteLaxMode,
})
```

**Assessment**:

| Attribute | Current | Recommended | Status |
|-----------|---------|-------------|--------|
| Name | `ggid_sticky` | `__Host-ggid_sticky` | Missing prefix |
| HttpOnly | `true` | `true` | Correct |
| Secure | **MISSING** | `true` | Vulnerable |
| SameSite | `Lax` | `Lax` | Correct |

**Finding**: The sticky session cookie is **missing `Secure: true`**. This cookie could be sent over plaintext HTTP, exposing the sticky routing key. While this is not a session credential, it leaks routing metadata.

### 10.3 Gateway Middleware — Canary Cookie (`canary.go`)

**File**: `services/gateway/internal/middleware/canary.go` (lines 75–82)

```go
func SetCanaryCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
```

**Assessment**:

| Attribute | Current | Recommended | Status |
|-----------|---------|-------------|--------|
| Secure | **MISSING** | `true` | Vulnerable |
| Prefix | None | `__Secure-` | Missing |

**Finding**: The canary cookie is missing `Secure`. Since this cookie controls traffic routing (canary vs. stable), an MITM could inject a canary cookie to route traffic to the canary backend.

### 10.4 Auth Service — Token Issuance (`http.go`, `token.go`)

**File**: `services/auth/internal/server/http.go` (line 292)

The auth service returns tokens as JSON in the response body:

```go
writeJSON(w, http.StatusOK, tokens)
// tokens = { access_token, refresh_token, token_type: "Bearer", expires_in, session_id }
```

**Assessment**: The auth service does **NOT set any cookies**. Access tokens and refresh tokens are returned as JSON. The client (SPA or mobile app) must store them. This is the **Bearer token pattern**, not the cookie-based session pattern.

**Implications**:
- No `Set-Cookie` headers → browser cookie attributes are irrelevant for the auth service itself.
- The gateway's CSRF middleware protects state-changing API calls.
- Token storage security depends on the client (localStorage = XSS-vulnerable; HttpOnly cookie = preferred).
- **Recommendation**: Consider returning the access token in an `__Host-ggid_session` HttpOnly cookie for browser clients, keeping the refresh token in a separate `__Host-ggid_refresh` cookie.

### 10.5 OAuth Service — State Validation (`oauth_service.go`)

**File**: `services/oauth/internal/service/oauth_service.go` (lines 266–273, 671–689)

The OAuth state is stored in a **server-side `sync.Map`** (in-memory), NOT in a cookie:

```go
// Store state for CSRF validation during token exchange.
if req.State != "" {
	stateKey := fmt.Sprintf("oauth:state:%s:%s", req.ClientID, req.State)
	stateStore.Store(stateKey, time.Now().Add(10 * time.Minute))
}
```

And validated server-side:

```go
func (s *OAuthService) ValidateState(clientID, state string) bool {
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)
	val, ok := stateStore.Load(stateKey)
	if !ok {
		return false
	}
	expiry, ok := val.(time.Time)
	if !ok || time.Now().After(expiry) {
		stateStore.Delete(stateKey)
		return false
	}
	stateStore.Delete(stateKey) // one-time use
	return true
}
```

**Assessment**: State validation is **correctly implemented server-side** with one-time use and expiry. However, the in-memory `sync.Map` does not survive restarts and does not work across multiple OAuth service instances. For production, use Redis.

### 10.6 Summary of Cookie Audit

| Cookie | Location | HttpOnly | Secure | SameSite | Prefix | Issues |
|--------|----------|----------|--------|----------|--------|--------|
| `csrf_token` | Gateway middleware | false (correct) | true | Lax | Missing | No `__Host-` prefix |
| `ggid_sticky` | Gateway sticky router | true | **false** | Lax | Missing | Missing Secure, no prefix |
| Canary cookie | Gateway canary router | true | **false** | Lax | Missing | Missing Secure, no prefix |
| Session cookie | Not implemented | N/A | N/A | N/A | N/A | Auth returns tokens as JSON, no cookie |
| OAuth state | Not a cookie (server-side) | N/A | N/A | N/A | N/A | Uses in-memory sync.Map (not Redis) |

---

## 11. Gap Analysis & Recommendations

### Gap 1: No `__Host-` Prefix on CSRF Cookie (P1 — Medium Effort)

**Problem**: The CSRF double-submit cookie (`csrf_token`) lacks the `__Host-` prefix. A compromised subdomain can inject a matching CSRF cookie+header pair, bypassing CSRF protection.

**Fix**: Rename cookie to `__Host-ggid_csrf` in `setCSRFCookie()` and update `CSRFProtect()` to read the new name.

**Effort**: 2 hours (code change + test update)

```go
// Before
Name: "csrf_token"
// After
Name: "__Host-ggid_csrf"
```

### Gap 2: Missing `Secure` Flag on Sticky and Canary Cookies (P1 — Low Effort)

**Problem**: The sticky session cookie and canary routing cookie are missing `Secure: true`. These cookies will be transmitted over plaintext HTTP if the user is downgraded.

**Fix**: Add `Secure: true` to both cookie setters.

**Effort**: 30 minutes

```go
// sticky.go — SetStickyCookie
http.SetCookie(w, &http.Cookie{
	Name:     sr.config.CookieName,
	Value:    key,
	Path:     "/",
	MaxAge:   int(sr.config.TTL.Seconds()),
	HttpOnly: true,
	Secure:   true,            // ADD THIS
	SameSite: http.SameSiteLaxMode,
})

// canary.go — SetCanaryCookie
http.SetCookie(w, &http.Cookie{
	Name:     name,
	Value:    value,
	Path:     "/",
	HttpOnly: true,
	Secure:   true,            // ADD THIS
	SameSite: http.SameSiteLaxMode,
})
```

### Gap 3: No Cookie-Based Session for Browser Clients (P2 — Medium Effort)

**Problem**: The auth service returns access/refresh tokens as JSON. SPAs typically store these in `localStorage`, which is vulnerable to XSS exfiltration. There is no option for HttpOnly cookie-based sessions.

**Fix**: Add a cookie-based response mode to the auth service. When `Accept: text/html` or a query parameter `?cookie=true` is present, set tokens in HttpOnly cookies instead of the JSON body.

**Effort**: 4 hours (handler changes + integration tests)

```go
// In login handler, after successful authentication:
if useCookieResponse(r) {
	SetHostPrefixedCookie(w, "ggid_session", tokens.AccessToken, tokens.ExpiresIn)
	SetHostPrefixedCookie(w, "ggid_refresh", tokens.RefreshToken, 30*24*3600)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
} else {
	writeJSON(w, http.StatusOK, tokens) // backward compatible
}
```

### Gap 4: OAuth State Stored in Memory, Not Redis (P2 — Low Effort)

**Problem**: The OAuth state parameter is validated using an in-memory `sync.Map`. This breaks under multi-instance deployments — state set on instance A may be validated on instance B (fails). It also does not survive service restarts.

**Fix**: Move state storage to Redis with the same key format and TTL.

**Effort**: 2 hours (add Redis client to OAuth service, replace `stateStore.Store/Load` with Redis SET/GET with TTL)

```go
// Store
stateKey := fmt.Sprintf("oauth:state:%s:%s", req.ClientID, req.State)
rdb.Set(ctx, stateKey, "1", 10*time.Minute)

// Validate
_, err := rdb.GetDel(ctx, stateKey).Result()
return err == nil // one-time use
```

### Gap 5: No Cookie Attribute Validation on Incoming Requests (P3 — Low Effort)

**Problem**: The gateway does not validate that incoming cookies have security prefixes. An attacker who manages to inject an unprefixed `session` cookie (e.g. via a legacy subdomain) could shadow the legitimate `__Host-session` cookie.

**Fix**: Add a middleware that rejects requests containing unprefixed sensitive cookies.

**Effort**: 1 hour

```go
func RejectUnprefixedCookies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sensitive := []string{"session", "csrf_token", "oauth_state"}
		for _, name := range sensitive {
			for _, c := range r.Cookies() {
				if c.Name == name { // unprefixed version exists
					writeForbidden(w, "invalid cookie state")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
```

### Effort Summary

| # | Gap | Priority | Effort | Impact |
|---|-----|----------|--------|--------|
| 1 | `__Host-` prefix on CSRF cookie | P1 | 2h | Eliminates subdomain CSRF bypass |
| 2 | `Secure` on sticky/canary cookies | P1 | 0.5h | Prevents HTTP cookie leakage |
| 3 | Cookie-based session mode | P2 | 4h | Eliminates localStorage XSS risk |
| 4 | OAuth state in Redis | P2 | 2h | Multi-instance reliability |
| 5 | Unprefixed cookie rejection middleware | P3 | 1h | Defense-in-depth |
| **Total** | | | **9.5h** | |

---

## References

- [RFC 6265bis — Cookie Prefixes](https://datatracker.ietf.org/doc/html/draft-ietf-httpbis-rfc6265bis)
- [RFC 6749 §10.12 — OAuth CSRF (state parameter)](https://datatracker.ietf.org/doc/html/rfc6749#section-10.12)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [OWASP CSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [MDN — Set-Cookie](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie)
- [Chrome SameSite Changes](https://developer.chrome.com/blog/samesite-cookie-changes)
- [Google Web.Dev — SameSite Recipes](https://web.dev/articles/samesite-cookie-recipes)

---

*Document version: 1.0 | Last updated: 2025-01-15 | Author: GGID Security Research*
