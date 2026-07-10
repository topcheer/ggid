# PAR and JAR Analysis — OAuth 2.0 Authorization Request Security

> **Scope**: Analysis of RFC 9101 (JAR) and RFC 9126 (PAR) for protecting
> OAuth authorization request parameters, with concrete GGID integration points.
>
> **References**: RFC 6749, RFC 9101, RFC 9126, RFC 9701 (FAPI 2.0), OAuth 2.1 draft.

---

## 1. The Parameter Tampering Problem

In standard OAuth 2.0 (RFC 6749), the authorization request is sent as
plain URL query parameters to `/authorize`:

```
GET /oauth/authorize?client_id=s6BhdRkqt3&redirect_uri=https://client.example.com/cb
  &response_type=code&scope=openid%20profile%20email&state=xyz123&nonce=n-0S6_WzA2Mj
```

**Attack surfaces exposed:**

| Threat | Description |
|---|---|
| **Parameter tampering (MITM)** | Attacker intercepting the redirect can modify `redirect_uri`, `scope`, or `client_id` before the user reaches the AS. |
| **Malicious `redirect_uri` injection** | Attacker substitutes their own redirect URI, causing the authorization code to be delivered to an attacker-controlled endpoint. |
| **Scope downgrade/upgrade** | Attacker adds scopes (`payments:write`) or removes scopes, changing effective permissions. |
| **Browser history exposure** | Parameters persist in browser history, Referer headers, and proxy logs. |
| **URL length limits** | Complex requests (large `claims`, multiple `acr_values`) can exceed browser URL limits (~2000–8192 bytes). |
| **Log exposure** | Reverse proxies, WAFs, and load balancers log full URLs including query parameters in plaintext. |

**Root cause**: the authorization request transits through the user agent
(browser) in plaintext, giving the browser — and anything observing it —
full visibility and mutability.

---

## 2. JAR — JWT-Secured Authorization Request (RFC 9101)

RFC 9101 defines **JWT-Secured Authorization Request (JAR)**: instead of
sending authorization parameters as individual query parameters, the
client encodes them into a signed JWT.

### How It Works

1. The client constructs a JWT containing all authorization request parameters as claims.
2. The client signs the JWT with its private key (confidential clients) or sends it to the AS for signing (public clients via PAR).
3. The client sends the JWT to `/authorize` via either:
   - **`request` parameter**: JWT passed directly (base64url-encoded).
   - **`request_uri` parameter**: URL pointing to the JWT (AS fetches it, or pre-registered).

### JWT Claims

```json
{
  "iss": "s6BhdRkqt3",
  "aud": "https://as.ggid.dev",
  "response_type": "code",
  "client_id": "s6BhdRkqt3",
  "redirect_uri": "https://client.example.com/cb",
  "scope": "openid profile email",
  "state": "xyz123",
  "nonce": "n-0S6_WzA2Mj",
  "code_challenge": "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
  "code_challenge_method": "S256",
  "iat": 1719000000,
  "exp": 1719000360,
  "jti": "unique-request-id-001"
}
```

### Request Flow

```
Client                                  Authorization Server
  |  GET /authorize?client_id=...&request=<signed-JWT>     |
  |-------------------------------------------------------->|
  |  1. Verify JWT signature  2. iss == client_id          |
  |  3. aud == AS issuer      4. exp not expired           |
  |  5. Extract claims → auth request  6. Process          |
  |<--------------------------------------------------------|
  |  302 redirect with code                                 |
```

### Benefits

- **Integrity protection**: parameters are signed and cannot be tampered with in transit.
- **Non-repudiation**: the AS can cryptographically prove which client originated the request.
- **Confidentiality** (optional): if the JWT is encrypted (JWE), parameters are hidden from the browser.
- **Large payloads**: JWTs carry complex claims that would exceed URL length limits.

### Limitations

- **Client key management**: confidential clients must register a signing key (or JWKS URI). Public clients (SPAs, mobile apps) cannot sign JWTs and must use PAR.
- **Increased complexity**: JWT construction, signing, and validation add overhead.
- **Replay risk**: without `jti` tracking, a captured JWT can be replayed within its validity window.
- **No browser-side protection**: the `request` JWT still appears in the URL. Only `request_uri` removes it, and that requires the AS to host the JWT.

---

## 3. PAR — Pushed Authorization Requests (RFC 9126)

RFC 9126 defines the **Pushed Authorization Request (PAR)** endpoint.
The client pushes the full authorization request to the AS via a
back-channel `POST`, receives an opaque `request_uri`, then redirects
the browser to `/authorize` with only `client_id` + `request_uri`.

### How It Works

```
Client                                  Authorization Server
  |  POST /oauth/par (back-channel)                       |
  |  client_id, redirect_uri, response_type, scope, ...   |
  |------------------------------------------------------->|
  |  1. Authenticate client  2. Validate redirect_uri      |
  |  3. Store request in Redis (TTL 60s)  4. Gen request_uri |
  |<-------------------------------------------------------|
  |  201 { "request_uri": "urn:...:abc123", "expires_in": 60 } |
  |                                                        |
  |  --- browser redirect (front-channel) ---              |
  |  GET /authorize?client_id=...&request_uri=urn:...      |
  |------------------------------------------------------->|
  |  1. Look up request_uri  2. Validate client_id match   |
  |  3. Check not expired/consumed  4. Use stored params   |
  |<-------------------------------------------------------|
  |  302 redirect with code                                |
```

### `request_uri` Properties

- **Single-use**: consumed on first lookup; cannot be replayed.
- **Short-lived**: typically 60 seconds TTL (RFC recommends <= 60s).
- **Bound to `client_id`**: must match between PAR push and browser redirect.
- **Opaque format**: `urn:ietf:params:oauth:request_uri:<random>`.

### AS Validation at PAR Endpoint

Before storing the request, the AS validates:
- Client is authenticated (client_secret for confidential, PKCE `code_challenge` for public clients).
- `redirect_uri` matches a registered URI for the client.
- `response_type` is allowed for the client.
- Request size is within limits (RFC recommends max 10KB).

### Error Handling

| Error Code | HTTP Status | Cause |
|---|---|---|
| `invalid_request_uri` | 400 | `request_uri` not found or invalid format |
| `expired_request_uri` | 400 | TTL has expired |
| `invalid_client` | 401 | Client authentication failed at PAR endpoint |
| `invalid_request` | 400 | Missing required parameters or validation failure |

### Benefits

- **Parameters hidden from browser**: no sensitive data in URL, history, or logs.
- **Works for public clients**: authenticate at PAR via PKCE (no signing key needed).
- **Pre-validation**: AS validates the request **before** the user sees the consent page.
- **No client key management**: unlike JAR, clients don't need registered signing keys.
- **URL-length friendly**: large requests pushed via POST body, not URL.

---

## 4. PAR + JAR Combination

| Alone | Limitation |
|---|---|
| **JAR alone** | Signed parameters, but the JWT still transits the browser URL. |
| **PAR alone** | Parameters pushed and hidden, but not cryptographically signed. |

**PAR + JAR together**:

1. The client constructs a signed JWT (JAR) with all authorization parameters.
2. The client pushes the JWT to the PAR endpoint via back-channel POST.
3. The AS validates the JWT signature, stores the request, returns a `request_uri`.
4. The browser redirect carries only `client_id` + `request_uri`.

**Result**: parameters are both **signed (tamper-proof)** and **invisible
to the browser**. This is the OAuth 2.1 recommended pattern and a baseline
requirement for FAPI 2.0 (RFC 9701) financial-grade APIs.

---

## 5. Comparison Table

| Feature | Plain OAuth (RFC 6749) | JAR (RFC 9101) | PAR (RFC 9126) | PAR + JAR |
|---|---|---|---|---|
| **Parameter protection** | None — plaintext in URL | Signed JWT | Pushed to AS (opaque ref) | Signed + pushed |
| **Integrity / signature** | None | Yes (client-signed JWT) | None (AS-validated only) | Yes (client-signed JWT) |
| **Browser visibility** | Full (query params) | Partial (JWT or request_uri) | Minimal (client_id + request_uri) | Minimal (client_id + request_uri) |
| **Public client support** | Yes (PKCE) | No (needs signing key) | Yes (PKCE at PAR) | Yes (PKCE at PAR; JAR by AS) |
| **Client complexity** | Low | High (JWT signing) | Medium (extra POST) | High |
| **AS validation timing** | At `/authorize` (user present) | At `/authorize` (JWT verify) | At PAR endpoint (pre-user) | At PAR endpoint (pre-user) |
| **Pre-user rejection** | No | Partial | Yes | Yes |
| **Non-repudiation** | No | Yes | No | Yes |
| **FAPI 2.0 compliant** | No | No | No | Yes |

---

## 6. GGID OAuth Integration Points

### Current State

GGID's authorize endpoint (`server.go:157`) reads parameters directly from URL query strings:

```go
clientID := r.URL.Query().Get("client_id")
redirectURI := r.URL.Query().Get("redirect_uri")
responseType := r.URL.Query().Get("response_type")
// ... no request_uri or request support
```

There is **no PAR endpoint** and **no `request`/`request_uri` handling**.
The service validates `redirect_uri` at code creation (`oauth_service.go:159`),
but all parameters arrive via plaintext URL.

### 6.1 PAR Endpoint (New)

Add route `POST /oauth/par`:

```go
mux.HandleFunc("/oauth/par", func(w http.ResponseWriter, r *http.Request) {
    _ = r.ParseForm()
    clientID := r.FormValue("client_id")
    redirectURI := r.FormValue("redirect_uri")
    codeChallenge := r.FormValue("code_challenge")

    // Validate client and redirect_uri.
    client, err := oauthSvc.GetClient(ctx, clientID)
    if err != nil || !client.ValidateRedirectURI(redirectURI) {
        writeJSON(w, http.StatusBadRequest, map[string]string{
            "error": "invalid_request", "error_description": "invalid client or redirect_uri"})
        return
    }
    // Public clients require PKCE at PAR endpoint.
    if client.RequiresPKCE() && codeChallenge == "" {
        writeJSON(w, http.StatusBadRequest, map[string]string{
            "error": "invalid_request", "error_description": "code_challenge required"})
        return
    }
    // Generate opaque request_uri, store in Redis (TTL 60s).
    requestURI := "urn:ietf:params:oauth:request_uri:" + generateToken(32)
    rdb.Set(ctx, "par:"+requestURI, marshalFormData(r.PostForm), 60*time.Second)

    writeJSON(w, http.StatusCreated, map[string]any{"request_uri": requestURI, "expires_in": 60})
})
```

**Redis key**: `par:{request_uri}`, 60-second TTL.
**Service method**: `oauthSvc.PushAuthorizationRequest(ctx, req)`.

### 6.2 Authorize Endpoint Changes

At the top of the `/oauth/authorize` handler, add `request_uri` support:

```go
requestURI := r.URL.Query().Get("request_uri")
if requestURI != "" {
    parData, err := rdb.Get(ctx, "par:"+requestURI).Result()
    if err == redis.Nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{
            "error": "invalid_request_uri", "error_description": "expired or not found"})
        return
    }
    // Validate client_id matches, then delete (single-use).
    rdb.Del(ctx, "par:"+requestURI)
    // Override query params with stored PAR data.
    redirectURI = parData["redirect_uri"]
    responseType = parData["response_type"]
}
```

### 6.3 JAR Support

Accept a `request` parameter (JWT) at `/oauth/authorize`:

```go
requestParam := r.URL.Query().Get("request")
if requestParam != "" {
    claims, err := verifyRequestJWT(requestParam, client)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{
            "error": "invalid_request_object", "error_description": err.Error()})
        return
    }
    redirectURI = claims["redirect_uri"].(string)
    responseType = claims["response_type"].(string)
}
```

**Key verification**: use the client's registered JWKS URI. Requires a new
`client_jwks_uri` field on the `OAuthClient` domain model.

### 6.4 Discovery Document Updates

Add to `/.well-known/openid-configuration`:

```json
{
  "pushed_authorization_request_endpoint": "https://as.ggid.dev/oauth/par",
  "request_parameter_supported": true,
  "request_uri_parameter_supported": true,
  "require_pushed_authorization_requests": false
}
```

---

## 7. Implementation Priority

| Phase | Feature | Effort | Rationale |
|---|---|---|---|
| **Phase 1** | PAR endpoint + `request_uri` in `/authorize` | 3–5 days | Simplest win — no client key management, works for all clients. Redis already in stack. |
| **Phase 2** | JAR `request` parameter support | 3–5 days | For high-security clients. Requires `client_jwks_uri` on `OAuthClient` + JWKS fetching/caching. |
| **Phase 3** | PAR + JAR combined | 1–2 days | Stacking — push the signed JWT via PAR endpoint. |
| **Phase 4** | Enforce PAR per-client (`require_par` flag) | 1 day | FAPI 2.0 profile compliance. |

**Recommendation**: Start with Phase 1 (PAR). Highest security-to-effort
ratio, works for all client types, no changes to client key registration.

---

## 8. Security Impact

| Threat Mitigated | Mechanism |
|---|---|
| **Parameter tampering** | PAR stores params server-side; JAR signs them cryptographically. |
| **Malicious `redirect_uri` injection** | AS validates `redirect_uri` at PAR endpoint before user interaction. |
| **Scope manipulation** | Parameters are opaque (`request_uri`) or signed (JWT). |
| **Browser history/log exposure** | No sensitive params in URL — only `client_id` + opaque `request_uri`. |
| **URL length overflow** | Large requests pushed via POST body, not URL query string. |
| **Pre-user attack rejection** | AS validates at PAR endpoint, rejecting malformed requests before user sees consent page. |
| **FAPI 2.0 compliance** | PAR + JAR is a baseline requirement for financial-grade API profiles. |

**Bottom line**: PAR is the single highest-impact security improvement
available for GGID's OAuth authorization flow. It eliminates the plaintext
parameter exposure that is the root cause of the most common OAuth
authorization request vulnerabilities.
