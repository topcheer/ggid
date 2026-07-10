# Token Replay Defense for IAM Systems

> **Research Document** — GGID IAM Suite
> **Topic**: JWT/access-token replay attack vectors and sender-constrained token defenses
> **Audience**: Security architects, platform engineers, IAM developers
> **Status**: Reference research + GGID gap analysis

---

## Table of Contents

1. [JWT Replay Attack Vectors](#1-jwt-replay-attack-vectors)
2. [jti Tracking with Redis](#2-jti-tracking-with-redis)
3. [DPoP (RFC 9449) Sender Constraint](#3-dpop-rfc-9449-sender-constraint)
4. [mTLS Sender Constraint (RFC 8705)](#4-mtls-sender-constraint-rfc-8705)
5. [Token Binding Comparison Matrix](#5-token-binding-comparison-matrix)
6. [Replay Window Detection](#6-replay-window-detection)
7. [GGID JWT Replay Surface Analysis](#7-ggid-jwt-replay-surface-analysis)
8. [Gap Analysis & Recommendations](#8-gap-analysis--recommendations)

---

## 1. JWT Replay Attack Vectors

### 1.1 Why Stateless JWTs Are Inherently Replayable

A stateless JWT is self-contained: the resource server validates the signature and
trusts the claims without any server-side state lookup. This is the core architectural
tradeoff — scalability comes at the cost of revocability. Once issued, the token is
valid until its `exp` claim expires, regardless of what happens on the server side.

A **replay attack** occurs when an attacker captures a valid token and reuses it to
impersonate the legitimate user. Because the token is a bearer credential (RFC 6750),
possession equals authorization — there is no proof that the presenter is the original
owner.

### 1.2 Token Theft Scenarios

| Scenario | Attack Vector | Mitigation Gap |
|---|---|---|
| **Network interception** | Man-in-the-middle on non-TLS or compromised CA | TLS/HSTS helps, but doesn't protect against endpoint compromise |
| **Log leakage** | Tokens logged in access logs, error traces, APM tools | Common in misconfigured reverse proxies and debug builds |
| **Browser history** | Tokens in URL fragments (implicit flow) or localStorage | XSS can read localStorage; fragments persist in browser history |
| **Memory dump** | Process memory of compromised client application | Difficult to defend; DPoP/mTLS raise the bar |
| **Referrer leakage** | Token in query param leaked via Referer header to third-party | Avoid token-in-URL patterns; use Authorization header |
| **WebRTC/WebSocket leak** | Token used to authenticate WS connection, intercepted by browser extension | CSP and extension isolation gaps |
| **SSRF / proxy chain** | Internal service forwards the token to an attacker-controlled endpoint | Network segmentation + token audience scoping |

### 1.3 Real-World Replay Attack Examples

**OAuth implicit flow token leak (2015–2019)**: The OAuth 2.0 Security BCP (RFC 9700)
deprecated the implicit flow precisely because tokens in URL fragments were leaked via
Referer headers, browser extensions, and log files. Stolen tokens were replayed against
resource APIs with full validity until expiration.

**GitHub OAuth token leak via Travis CI (2018)**: Build logs exposed OAuth tokens that
were injected as environment variables. Attackers replayed the tokens to access private
repositories. The tokens were bearer tokens with no sender-constraint mechanism.

**Firebase JWT replay (2020 CVE)**: Misconfigured Firebase projects allowed unlimited
JWT lifetime. Attackers who intercepted a single token could replay it indefinitely.

**Key lesson**: Every bearer token system has a replay window equal to the token's
lifetime. The only defenses are (a) making the window short, (b) tracking usage, or
(c) binding the token to a cryptographic proof of possession.

---

## 2. jti Tracking with Redis

### 2.1 JWT ID (`jti`) Claim

RFC 7519 defines `jti` (JWT ID) as a unique identifier for the token. It is a string
value intended to be used as a nonce to prevent replay. The `jti` value MUST be assigned
in a manner that ensures there is a negligible probability that the same value will be
accidentally assigned to a different token.

### 2.2 One-Time-Use Enforcement Pattern

The strategy is simple: on first use, record the `jti` in a fast store (Redis). On
subsequent requests, if the `jti` is already present, the token has been replayed.

For **one-time-use tokens** (e.g., authorization codes, back-channel logout tokens):
the first use consumes the token, and any subsequent presentation is a replay.

For **multi-use access tokens** (the common case): `jti` tracking enables two modes:
- **Replay alerting**: detect that the same token was used from different
  IPs/sessions simultaneously (section 6).
- **Revocation**: explicitly blacklist a `jti` when a token is stolen, even before
  `exp` elapses.

### 2.3 Redis Implementation with TTL

The key insight is that the Redis key's TTL should equal the token's remaining lifetime
(`exp - now`). Once the token expires naturally, the Redis entry is garbage-collected.

```go
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
)

// JTITracker provides Redis-backed JWT ID replay detection.
type JTITracker struct {
	rdb *redis.Client
}

func NewJTITracker(rdb *redis.Client) *JTITracker {
	return &JTITracker{rdb: rdb}
}

// CheckAndStore atomically checks if jti was seen, then stores it.
// Returns true if the token is a replay (jti already seen).
// Uses Redis SET NX (atomic) to handle race conditions.
func (t *JTITracker) CheckAndStore(ctx context.Context, jti string, expiresAt time.Time) (bool, error) {
	key := fmt.Sprintf("ggid:jti:%s", jti)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// Token already expired — not a replay, just expired.
		return false, fmt.Errorf("token expired")
	}

	// SET key 1 NX EX <ttl>
	// If NX succeeds (ok=true), this is the first use — not a replay.
	// If NX fails (ok=false), the jti was already seen — replay.
	ok, err := t.rdb.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("jti check failed: %w", err)
	}
	return !ok, nil // true = replay
}

// Blacklist explicitly marks a jti as revoked (e.g., after token theft).
func (t *JTITracker) Blacklist(ctx context.Context, jti string, expiresAt time.Time) error {
	key := fmt.Sprintf("ggid:jti_blacklist:%s", jti)
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // already expired
	}
	return t.rdb.Set(ctx, key, "revoked", ttl).Err()
}

// IsBlacklisted checks if a jti has been explicitly revoked.
func (t *JTITracker) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("ggid:jti_blacklist:%s", jti)
	val, err := t.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "revoked", nil
}

// ReplayDetectionMiddleware wraps JWTAuth to add jti-based replay detection.
// This is for MULTI-USE tokens: it records first-use and alerts on reuse
// from different IP addresses. It does NOT block legitimate concurrent requests
// from the same IP.
func (t *JTITracker) ReplayDetectionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract claims from context (set by JWTAuth)
		claimsRaw := r.Context().Value(JWTClaimsKey)
		claims, ok := claimsRaw.(jwt.MapClaims)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		jti, _ := claims["jti"].(string)
		if jti == "" {
			// No jti claim — cannot track. Allow through (log warning).
			next.ServeHTTP(w, r)
			return
		}

		// Check blacklist first
		blacklisted, err := t.IsBlacklisted(r.Context(), jti)
		if err == nil && blacklisted {
			http.Error(w, `{"error":"token revoked"}`, http.StatusUnauthorized)
			return
		}

		// Check if this jti was seen from a different IP
		clientIP := extractClientIP(r)
		ipKey := fmt.Sprintf("ggid:jti_ip:%s", jti)

		pipe := t.rdb.TxPipeline()
		storedIP := pipe.Get(r.Context(), ipKey)
		pipe.Set(r.Context(), ipKey, clientIP, 5*time.Minute) // rolling window
		_, _ = pipe.Exec(r.Context())

		if storedIP.Err() == nil {
			knownIP := storedIP.Val()
			if knownIP != clientIP {
				// Same token from different IP within window — possible replay
				// Log alert, optionally block
				logReplayAlert(jti, knownIP, clientIP, r.URL.Path)
				http.Error(w, `{"error":"token replay detected"}`, http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.SplitN(xff, ",", 2)[0]
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}
```

### 2.4 Race Condition Handling

The critical race condition: two concurrent requests with the same token arrive within
milliseconds. A naive GET-then-SET approach would see both succeed because the first
GET returns "not found" before the first SET executes.

**Solution**: Redis `SET NX` (Set if Not eXists) is atomic at the Redis level. Even under
concurrency, exactly one request will succeed in the `SET NX` and the other will see the
key already exists. The `SETNX` command executes as a single atomic operation within
Redis's single-threaded event loop — no race window exists.

For one-time-use tokens (authorization codes), `SETNX` gives perfect replay prevention:
the first request consumes the code, the second gets rejected.

---

## 3. DPoP (RFC 9449) Sender Constraint

### 3.1 How DPoP Works

DPoP (Demonstration of Proof-of-Possession) binds an access token to a client-held
private key. The client generates an EC key pair (P-256) and sends a DPoP proof JWT
with every request. The proof JWT is signed by the private key and includes:

- `htm`: HTTP method (e.g., "GET")
- `htu`: HTTP URI (without query string, normalized)
- `iat`: Issued at timestamp
- `jti`: Unique ID (for replay prevention of the proof itself)
- `ath`: Base64url(SHA-256(access_token)) — binds proof to token

At the token endpoint, the authorization server records the client's public key
thumbprint (`jkt`) in the token's `cnf.jkt` claim. At the resource server, the proof
JWT's public key MUST match the `cnf.jkt` in the access token.

### 3.2 Why DPoP Prevents Replay

A stolen bearer token is useless to an attacker who doesn't have the corresponding
private key. Even if the attacker captures the token from network traffic, they cannot
generate a valid DPoP proof for each request. The proof is request-specific: it covers
the HTTP method, URI, and access token hash, and has a short freshness window.

### 3.3 DPoP Verification at the Resource Server

```go
package middleware

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DPoPVerifier validates DPoP proof JWTs at the resource server.
type DPoPVerifier struct {
	// AllowedClockSkew for iat validation.
	clockSkew    time.Duration
	// MaxProofAge is the maximum age of a DPoP proof (default: 60s).
	maxProofAge  time.Duration
}

func NewDPoPVerifier() *DPoPVerifier {
	return &DPoPVerifier{
		clockSkew:   30 * time.Second,
		maxProofAge: 60 * time.Second,
	}
}

// DPoPClaims represents the claims in a DPoP proof JWT.
type DPoPClaims struct {
	HTM string `json:"htm"` // HTTP method
	HTU string `json:"htu"` // HTTP URI
	IAT int64  `json:"iat"` // Issued at (Unix timestamp)
	JTI string `json:"jti"` // Proof JWT ID
	ATH string `json:"ath"` // Access token hash
	jwt.RegisteredClaims
}

// VerifyDPoPProof validates the DPoP header against the request and access token.
// Returns the key thumbprint (jkt) for comparison with cnf.jkt in the token.
func (v *DPoPVerifier) VerifyDPoPProof(r *http.Request, accessToken string) (string, error) {
	dpopHeader := r.Header.Get("DPoP")
	if dpopHeader == "" {
		return "", fmt.Errorf("missing DPoP header")
	}

	// Parse the proof JWT (unsigned header/payload, extract key from header)
	parts := strings.SplitN(dpopHeader, ".", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid DPoP JWT format")
	}

	// Decode header to get the public key (jwk)
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid DPoP header encoding: %w", err)
	}

	var dpopHeader struct {
		Typ string          `json:"typ"`
		Alg string          `json:"alg"`
		JWK json.RawMessage `json:"jwk"`
	}
	if err := json.Unmarshal(headerBytes, &dpopHeader); err != nil {
		return "", fmt.Errorf("invalid DPoP header: %w", err)
	}

	if dpopHeader.Typ != "dpop+jwt" {
		return "", fmt.Errorf("invalid DPoP typ: expected 'dpop+jwt', got '%s'", dpopHeader.Typ)
	}

	// Parse the JWK to get an ECDSA public key
	pubKey, err := parseJWK(dpopHeader.JWK)
	if err != nil {
		return "", fmt.Errorf("invalid DPoP JWK: %w", err)
	}

	// Verify signature with the embedded public key
	claims := &DPoPClaims{}
	token, err := jwt.ParseWithClaims(dpopHeader, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("DPoP must use ECDSA, got %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid DPoP proof signature: %w", err)
	}

	// Verify htm matches request method
	if !strings.EqualFold(claims.HTM, r.Method) {
		return "", fmt.Errorf("DPoP htm mismatch: expected %s, got %s", r.Method, claims.HTM)
	}

	// Verify htu matches request URI (normalized: no query/fragment, lowercase host)
	expectedHTU := normalizeHTU(r)
	if claims.HTU != expectedHTU {
		return "", fmt.Errorf("DPoP htu mismatch: expected %s, got %s", expectedHTU, claims.HTU)
	}

	// Verify iat is within acceptable window
	proofTime := time.Unix(claims.IAT, 0)
	now := time.Now()
	if now.Add(v.clockSkew).Before(proofTime) || now.Add(-v.maxProofAge).After(proofTime) {
		return "", fmt.Errorf("DPoP proof expired or from future")
	}

	// Verify ath matches the access token hash
	tokenHash := sha256.Sum256([]byte(accessToken))
	expectedATH := base64.RawURLEncoding.EncodeToString(tokenHash[:])
	if claims.ATH != expectedATH {
		return "", fmt.Errorf("DPoP ath mismatch: proof not bound to this token")
	}

	// Compute the key thumbprint (jkt) = base64url(SHA-256(JWK))
	jkt, err := computeJKT(dpopHeader.JWK)
	if err != nil {
		return "", fmt.Errorf("compute jkt: %w", err)
	}

	return jkt, nil
}

// VerifySenderConstrainedToken checks that the access token's cnf.jkt
// matches the DPoP proof's key thumbprint.
func (v *DPoPVerifier) VerifySenderConstrainedToken(r *http.Request, claims jwt.MapClaims) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing Authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid Authorization header")
	}
	accessToken := strings.TrimSpace(parts[1])

	// Extract cnf.jkt from access token
	cnfRaw, ok := claims["cnf"]
	if !ok {
		// Token is not sender-constrained — no DPoP required
		return nil
	}
	cnf, ok := cnfRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid cnf claim")
	}
	boundJKT, ok := cnf["jkt"].(string)
	if !ok || boundJKT == "" {
		return nil // cnf exists but no jkt — not DPoP-bound
	}

	// Verify the DPoP proof
	proofJKT, err := v.VerifyDPoPProof(r, accessToken)
	if err != nil {
		return fmt.Errorf("DPoP verification failed: %w", err)
	}

	// The proof's key must match the token's bound key
	if proofJKT != boundJKT {
		return fmt.Errorf("DPoP key mismatch: token bound to different key")
	}

	return nil
}

// normalizeHTU extracts the normalized HTTP URI from the request.
// Per RFC 9449: scheme + host + path (no query, no fragment, host lowercase).
func normalizeHTU(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" {
		scheme = xfp
	}
	host := strings.ToLower(r.Host)
	// Strip query/fragment from path
	path := r.URL.Path
	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}

// parseJWK parses a JSON Web Key into an ECDSA public key.
func parseJWK(jwkBytes json.RawMessage) (*ecdsa.PublicKey, error) {
	var jwk struct {
		Kty string `json:"kty"`
		Crv string `json:"crv"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	if err := json.Unmarshal(jwkBytes, &jwk); err != nil {
		return nil, err
	}
	if jwk.Kty != "EC" || jwk.Crv != "P-256" {
		return nil, fmt.Errorf("DPoP key must be EC P-256, got %s %s", jwk.Kty, jwk.Crv)
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("invalid x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid y: %w", err)
	}
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)
	return &ecdsa.PublicKey{Curve: crypto.P256(), X: x, Y: y}, nil
}

// computeJKT computes the JWK thumbprint per RFC 7638.
func computeJKT(jwkBytes json.RawMessage) (string, error) {
	var jwk struct {
		Crv string `json:"crv"`
		Kty string `json:"kty"`
		X   string `json:"x"`
		Y   string `json:"y"`
	}
	if err := json.Unmarshal(jwkBytes, &jwk); err != nil {
		return "", err
	}
	// RFC 7638 canonical order: crv, kty, x, y
	canonical := fmt.Sprintf(`{"crv":"%s","kty":"%s","x":"%s","y":"%s"}`,
		jwk.Crv, jwk.Kty, jwk.X, jwk.Y)
	h := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(h[:]), nil
}
```

### 3.4 Replay Defense Angle

DPoP's replay defense properties:
1. **Per-request proof**: each API call requires a fresh DPoP JWT with a unique `jti`
   and timestamp. A captured token alone is insufficient.
2. **Replay window**: proofs are valid for ~60 seconds. Captured proofs can be replayed
   within this window, so resource servers should also track proof `jti` values in Redis
   for full one-time-use semantics.
3. **Token theft resilience**: even with full token compromise, the attacker needs the
   private key. This is the strongest bearer-token defense short of hardware-bound keys.

---

## 4. mTLS Sender Constraint (RFC 8705)

### 4.1 How mTLS Token Binding Works

RFC 8705 binds access tokens to the client's TLS certificate. At the token endpoint,
the authorization server extracts the client certificate's SHA-256 thumbprint
(`x5t#S256`) and embeds it in the token's `cnf` claim:

```json
{
  "sub": "user-123",
  "cnf": {
    "x5t#S256": "x5t#S256:base64url-sha256-of-cert-der"
  }
}
```

At the resource server, the TLS connection's client certificate thumbprint MUST match
the `cnf.x5t#S256` in the token.

### 4.2 Why mTLS-Bound Tokens Can't Be Replayed

An attacker who steals the token cannot use it without the corresponding TLS client
certificate and private key. The TLS handshake fails before any HTTP request is
processed. This is defense at the transport layer — the attacker never even reaches
the application.

### 4.3 GGID's Existing mTLS Support

GGID already implements the core mTLS binding primitives in
`services/oauth/internal/service/jar_mtls.go`:

```go
// ExtractCertThumbprint extracts the x5t#S256 thumbprint from a TLS client
// certificate's DER-encoded bytes.
func ExtractCertThumbprint(certDER []byte) string {
    if len(certDER) == 0 {
        return ""
    }
    return "x5t#S256:" + hashTokenSHA256(string(certDER))
}

// ValidateMTLSClientAuth validates that the access token's cnf.x5t#S256 claim
// matches the TLS client certificate thumbprint from the request.
func ValidateMTLSClientAuth(claims jwt.MapClaims, certThumbprint string) error {
    if certThumbprint == "" {
        return fmt.Errorf("no client certificate provided")
    }
    cnfRaw, ok := claims["cnf"]
    if !ok {
        return fmt.Errorf("token is not sender-constrained (missing cnf claim)")
    }
    cnf, ok := cnfRaw.(map[string]any)
    if !ok {
        return fmt.Errorf("invalid cnf claim format")
    }
    x5t, ok := cnf["x5t#S256"].(string)
    if !ok || x5t == "" {
        return fmt.Errorf("token not bound to client certificate (no x5t#S256)")
    }
    if !strings.EqualFold(x5t, certThumbprint) {
        return fmt.Errorf("client certificate thumbprint mismatch")
    }
    return nil
}
```

### 4.4 mTLS Verification Middleware for the Gateway

The following middleware wires mTLS binding into the gateway's request pipeline:

```go
// MTLSBindingMiddleware validates that sender-constrained tokens are used
// with the correct client certificate. If a token contains a cnf.x5t#S256
// claim, the request MUST include a matching TLS client certificate.
func MTLSBindingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract claims from context (set by JWTAuth)
		claimsRaw := r.Context().Value(JWTClaimsKey)
		claims, ok := claimsRaw.(jwt.MapClaims)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// Check if token is sender-constrained
		cnfRaw, hasCNF := claims["cnf"]
		if !hasCNF {
			// Not bound — bearer token, no mTLS check needed.
			next.ServeHTTP(w, r)
			return
		}
		cnf, ok := cnfRaw.(map[string]any)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if _, hasX5T := cnf["x5t#S256"]; !hasX5T {
			// cnf exists but no x5t — might be DPoP-bound instead.
			next.ServeHTTP(w, r)
			return
		}

		// Token is mTLS-bound — extract client certificate from TLS connection
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, `{"error":"client certificate required for mTLS-bound token"}`,
				http.StatusUnauthorized)
			return
		}

		cert := r.TLS.PeerCertificates[0]
		// Extract DER-encoded certificate
		certDER := cert.Raw

		// Compute x5t#S256 thumbprint
		h := sha256.Sum256(certDER)
		thumbprint := "x5t#S256:" + base64.RawURLEncoding.EncodeToString(h[:])

		// Validate against token's cnf claim
		if err := ValidateMTLSClientAuth(claims, thumbprint); err != nil {
			http.Error(w,
				fmt.Sprintf(`{"error":"mTLS binding failed: %s"}`, err.Error()),
				http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

### 4.5 Infrastructure Requirements

mTLS sender-constrained tokens require:
- **TLS termination with client cert passthrough**: The load balancer/reverse proxy must
  pass the client certificate to the backend. In NGINX, this requires
  `proxy_set_header X-SSL-Cert $ssl_client_escaped_cert` or similar.
- **Certificate management**: Clients must obtain certificates from a trusted CA. For
  internal services, a private PKI (e.g., step-ca, Vault PKI) is typical.
- **Revocation**: Certificate revocation (CRL/OCSP) must be checked during TLS handshake.
  GGID should integrate `r.TLS.PeerCertificates[0].CheckCRLSignature` or OCSP stapling.

---

## 5. Token Binding Comparison Matrix

| Feature | Bearer Token | jti Tracking | DPoP (RFC 9449) | mTLS (RFC 8705) | Token Status List |
|---|---|---|---|---|---|
| **Server-side state** | None | Redis (per-token) | None (proof is stateless) | None (cert is stateless) | Redis/DB (status index) |
| **Replay prevention** | None | Yes (if one-time-use) | Strong (per-request proof) | Strong (needs private key) | None (revocation only) |
| **Client complexity** | Minimal | None (transparent) | High (key management + proof JWT) | Medium (cert management) | Low |
| **Server complexity** | Minimal | Medium (Redis ops) | High (proof verification, key cache) | Medium (TLS config) | Medium (status polling) |
| **Performance overhead** | None | ~1ms (Redis SETNX) | ~2-5ms (ECDSA verify per request) | ~0ms (handled in TLS) | ~1ms (status lookup) |
| **Browser/JS client support** | Yes | Yes (transparent) | Yes (WebCrypto API) | No (can't manage certs) | Yes |
| **Mobile client support** | Yes | Yes | Yes (platform keystore) | Yes (cert in app keystore) | Yes |
| **Service-to-service** | Yes | Yes | Possible but overkill | Ideal (machine identity) | Yes |
| **Revocation granularity** | None (until exp) | Per-token | None (until exp) | None (until exp) | Per-token or per-session |
| **Standard** | RFC 6750 | RFC 7519 (jti) | RFC 9449 | RFC 8705 | RFC 9472 (draft) |

### When to Use Each

| Use Case | Recommended Approach | Rationale |
|---|---|---|
| Public SPA client | Short-lived bearer + jti anomaly detection | DPoP possible but complex; short TTL limits replay window |
| Native mobile app | DPoP | Platform keystore provides secure key storage |
| Service-to-service (internal) | mTLS | Machine identity via cert; no key management for apps |
| High-security API (financial) | mTLS + short TTL | Defense in depth; strongest replay prevention |
| Legacy client migration | Bearer + jti tracking | Incremental hardening without breaking clients |
| Real-time revocation needed | Token status list | Allows revocation without waiting for exp |

---

## 6. Replay Window Detection

### 6.1 Time-Based Windows (NBF/EXP)

The most fundamental replay defense is short token lifetimes. GGID currently issues:
- **Auth service access tokens**: configurable via `JWTConfig.AccessTokenTTL`
- **OAuth service access tokens**: 15 minutes (hardcoded in `issueAccessToken`)
- **OAuth service ID tokens**: 1 hour (hardcoded in `issueIDToken`)

Best practices:
- Access tokens: 5-15 minutes
- ID tokens: short lifetime (they're consumed at login)
- Refresh tokens: 30 days with rotation (GGID already does this)

### 6.2 Anomaly Detection: Same Token, Different IP

When the same `jti` is used from two different IP addresses within the token's lifetime,
this is a strong replay signal. The detection must distinguish:

- **Legitimate**: user switches from WiFi to cellular (IP change) — the old request
  completes before the new one starts.
- **Suspicious**: two simultaneous requests from different IPs/regions.

```go
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
)

// ReplayAnomalyDetector scores requests for replay-like behavior.
type ReplayAnomalyDetector struct {
	rdb         *redis.Client
	windowSize  time.Duration // rolling window for IP tracking
	maxScore    int           // score threshold for blocking
}

func NewReplayAnomalyDetector(rdb *redis.Client) *ReplayAnomalyDetector {
	return &ReplayAnomalyDetector{
		rdb:        rdb,
		windowSize: 5 * time.Minute,
		maxScore:   60, // block if score >= 60
	}
}

// ReplayScore holds the computed risk score for a request.
type ReplayScore struct {
	Score       int
	Reasons     []string
	Action      string // "allow", "challenge", "block"
}

// Evaluate checks the request for replay anomalies.
func (d *ReplayAnomalyDetector) Evaluate(r *http.Request, claims jwt.MapClaims) ReplayScore {
	score := 0
	var reasons []string

	jti, _ := claims["jti"].(string)
	if jti == "" {
		return ReplayScore{Score: 0, Action: "allow"}
	}

	clientIP := extractClientIP(r)
	ctx := r.Context()

	// 1. Check if this jti was seen from a different IP
	ipKey := fmt.Sprintf("ggid:replay:jti_ip:%s", jti)
	prevIP, err := d.rdb.Get(ctx, ipKey).Result()
	if err == nil && prevIP != clientIP {
		score += 40
		reasons = append(reasons, fmt.Sprintf("jti %s seen from %s, now from %s", jti, prevIP, clientIP))
	}

	// Update IP tracking
	d.rdb.Set(ctx, ipKey, clientIP, d.windowSize)

	// 2. Check for impossible travel
	if sub, _ := claims["sub"].(string); sub != "" {
		userLocKey := fmt.Sprintf("ggid:replay:user_loc:%s", sub)
		prevLocJSON, err := d.rdb.Get(ctx, userLocKey).Result()
		if err == nil {
			var prevLoc UserLocation
			if json.Unmarshal([]byte(prevLocJSON), &prevLoc) == nil {
				travelScore := assessImpossibleTravel(prevLoc, clientIP, time.Since(prevLoc.Timestamp))
				if travelScore > 0 {
					score += travelScore
					reasons = append(reasons, fmt.Sprintf("impossible travel: %s → %s in %v",
						prevLoc.IP, clientIP, time.Since(prevLoc.Timestamp)))
				}
			}
		}
		// Update last known location
		loc := UserLocation{IP: clientIP, Timestamp: time.Now()}
		locJSON, _ := json.Marshal(loc)
		d.rdb.Set(ctx, userLocKey, locJSON, 1*time.Hour)
	}

	// 3. Check token age (reusing a very old token is suspicious)
	if iat, ok := claims["iat"].(float64); ok {
		tokenAge := time.Since(time.Unix(int64(iat), 0))
		if tokenAge > 10*time.Minute {
			score += 10
			reasons = append(reasons, fmt.Sprintf("token age %v exceeds expected", tokenAge))
		}
	}

	// 4. High-frequency same-jti requests
	usageKey := fmt.Sprintf("ggid:replay:jti_count:%s", jti)
	count, _ := d.rdb.Incr(ctx, usageKey).Result()
	d.rdb.Expire(ctx, usageKey, d.windowSize)
	if count > 50 {
		score += 20
		reasons = append(reasons, fmt.Sprintf("jti used %d times in %v", count, d.windowSize))
	}

	// Determine action
	action := "allow"
	if score >= d.maxScore {
		action = "block"
	} else if score >= d.maxScore/2 {
		action = "challenge" // require MFA re-auth
	}

	return ReplayScore{Score: score, Reasons: reasons, Action: action}
}

// assessImpossibleTravel returns a risk score for geo-impossible travel.
// Uses a simplified distance/speed heuristic (production would use GeoIP DB).
func assessImpossibleTravel(prev UserLocation, currentIP string, elapsed time.Duration) int {
	// In production, resolve both IPs to coordinates via GeoIP (MaxMind)
	// and compute great-circle distance / speed.
	// For this example, return a fixed score if IPs are in different /8 blocks
	// and elapsed is very short.
	prevParts := strings.Split(prev.IP, ".")
	currParts := strings.Split(currentIP, ".")
	if len(prevParts) >= 1 && len(currParts) >= 1 {
		if prevParts[0] != currParts[0] && elapsed < 2*time.Minute {
			return 30 // Different /8 within 2 minutes is suspicious
		}
	}
	return 0
}

type UserLocation struct {
	IP        string    `json:"ip"`
	Latitude  float64   `json:"lat,omitempty"`
	Longitude float64   `json:"lng,omitempty"`
	Timestamp time.Time `json:"ts"`
}

// AnomalyMiddleware wraps the request pipeline with replay anomaly scoring.
func (d *ReplayAnomalyDetector) AnomalyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claimsRaw := r.Context().Value(JWTClaimsKey)
		claims, ok := claimsRaw.(jwt.MapClaims)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		result := d.Evaluate(r, claims)
		switch result.Action {
		case "block":
			// Log security event
			logSecurityEvent("replay_blocked", claims, result)
			http.Error(w, `{"error":"access denied: anomalous activity detected"}`,
				http.StatusForbidden)
			return
		case "challenge":
			// Require step-up authentication
			w.Header().Set("WWW-Authenticate", `DPoP realm="ggid", error="insufficient_auth"`)
			http.Error(w, `{"error":"step-up authentication required"}`,
				http.StatusUnauthorized)
			return
		default:
			next.ServeHTTP(w, r)
		}
	})
}
```

---

## 7. GGID JWT Replay Surface Analysis

### 7.1 Gateway JWT Validation (`services/gateway/internal/middleware/middleware.go`)

The `JWTAuth` middleware validates:
- Signature (RS256 via JWKS or static public key)
- Issuer (`iss` claim)
- Audience (`aud` claim)
- Expiry (`exp`, `nbf`, `iat` — validated by jwt/v5 defaults)

**What it does NOT validate**:
- `jti` claim is never extracted or checked
- No Redis-backed replay tracking
- No IP-based anomaly detection
- No `cnf` claim inspection (mTLS/DPoP binding not enforced)
- No session-level revocation check against Redis (separate `SessionMiddleware` exists
  but is not mandatory)

The middleware extracts only `sub` and `tenant_id` from claims. The `jti` is silently
ignored.

### 7.2 Auth Service Token Generation (`services/auth/internal/service/token_service.go`)

`IssueAccessToken` generates a `jti` via `uuid.New().String()` (line 85):
```go
claims := AccessTokenClaims{
    RegisteredClaims: jwt.RegisteredClaims{
        ID: uuid.New().String(),  // ← jti IS set
        ...
    },
}
```

**Assessment**: The `jti` IS generated, but because the gateway never checks it, replayed
tokens pass validation silently. The `jti` serves as a unique identifier for logging but
provides zero replay defense.

Refresh token rotation includes replay detection: if a revoked refresh token is reused,
the entire session is revoked. This is correct and effective.

### 7.3 OAuth Service Token Generation (`services/oauth/internal/service/oauth_service.go`)

`issueAccessToken` includes `jti: uuid.New().String()` in the claims map (line 422).
The `ParseAccessToken` method (line 486) validates signature, issuer, and expiry but
**does not check `jti`**.

`issueIDToken` also includes `jti` (line 1274 in the device flow path).

Back-channel logout has a **partial** jti replay check using an in-memory `sync.Map`
(`backchannelLogoutList`). This is:
- **In-memory only**: lost on process restart, so replays succeed after restart
- **Per-instance**: each OAuth service replica has its own map, so replays across
  instances succeed
- **Never garbage-collected**: grows unboundedly (memory leak over time)

### 7.4 mTLS Binding Implementation (`services/oauth/internal/service/jar_mtls.go`)

The OAuth service implements `ValidateMTLSClientAuth` and `ValidateMTLSBinding` with
correct `cnf.x5t#S256` verification logic. However:
- These functions exist but are **not wired into the gateway middleware pipeline**
- The gateway's `JWTAuth` does not call `ValidateMTLSClientAuth`
- No middleware exists to extract client certificates and enforce binding

### 7.5 Session Middleware (`services/gateway/internal/middleware/session.go`)

`SessionManager.Middleware` checks Redis for session validity. This provides:
- Session revocation (delete from Redis → subsequent requests fail)
- Session listing and revocation API

But it does NOT provide:
- Token-level (jti-based) revocation
- Replay detection (same jti from different IPs)
- The session check fails open on Redis errors (line 56-58: "Redis error — fail open")

### 7.6 Identified Replay Vulnerabilities

| # | Vulnerability | Severity | File |
|---|---|---|---|
| V1 | Gateway does not track or validate `jti` — tokens are fully replayable | **High** | `middleware.go:JWTAuth` |
| V2 | No mTLS binding enforcement at gateway despite OAuth service having the logic | **High** | `middleware.go` |
| V3 | No DPoP support anywhere in the system | **Medium** | System-wide |
| V4 | Back-channel logout jti tracking is in-memory, not durable, not shared | **Medium** | `oauth_service.go:ParseBackchannelLogoutToken` |
| V5 | Session check fails open on Redis errors — revoked sessions may pass | **Medium** | `session.go:56` |
| V6 | No IP-based anomaly or impossible-travel detection | **Low** | System-wide |
| V7 | Token lifetime hardcoded at 15min/1hr in OAuth service — not configurable | **Low** | `oauth_service.go:404,447` |
| V8 | No token status list / revocation endpoint for access tokens (only refresh) | **Medium** | System-wide |

---

## 8. Gap Analysis & Recommendations

### 8.1 Current State Summary

GGID generates `jti` on all access tokens and ID tokens, which is the prerequisite for
replay tracking. However, the `jti` is never consumed at the validation layer. The system
has:
- Bearer-only token enforcement at the gateway (V1)
- Unused mTLS binding logic in the OAuth service (V2)
- No DPoP support (V3)
- Session-level revocation but no token-level revocation (V8)
- Anomaly detection is entirely absent (V6)

The refresh token flow is the one area with proper replay defense: rotation detects
replayed refresh tokens and revokes the entire session chain.

### 8.2 Implementation Roadmap

#### Action 1: Add jti-Based Replay Tracking to Gateway Middleware
**Effort**: 2-3 days

1. Create `JTITracker` type in `services/gateway/internal/middleware/jti_tracker.go`
   using Redis `SETNX` as shown in section 2.3.
2. Wire `JTITracker.ReplayDetectionMiddleware` into the gateway middleware chain,
   after `JWTAuth` and before the proxy handler.
3. Add jti blacklist support for explicit token revocation (e.g., on logout).
4. Extract `jti` from claims in `JWTAuth` and store in context.
5. Add Prometheus metrics: `replay_detected_total`, `jti_blacklist_hits_total`.

**Impact**: Closes V1, enables V8 (partial).

#### Action 2: Enforce mTLS Binding at the Gateway
**Effort**: 3-5 days

1. Create `MTLSBindingMiddleware` as shown in section 4.4.
2. Configure the gateway's TLS listener to require client certificates
   (`tls.Config{ClientAuth: tls.RequestClientCert}`).
3. Extract `cnf.x5t#S256` from JWT claims and compare with the TLS connection's
   client certificate.
4. Add a configuration flag to make mTLS binding optional (per-client or per-route).
5. Update the OAuth token endpoint to embed `cnf.x5t#S256` when the client uses
   `tls_client_auth`.

**Impact**: Closes V2. Leverages existing `jar_mtls.go` logic.

#### Action 3: Implement DPoP Verification Middleware
**Effort**: 5-7 days

1. Create `DPoPVerifier` as shown in section 3.3.
2. Add `DPoP` header parsing in the gateway middleware chain.
3. Verify proof JWT signature, `htm`, `htu`, `iat`, and `ath`.
4. Compare proof key thumbprint (`jkt`) with the token's `cnf.jkt`.
5. Add DPoP proof jti tracking in Redis (60-second TTL) for proof replay prevention.
6. Update the OAuth token endpoint to support DPoP-bound token issuance
   (extract client JWK from `DPoP` header, compute `jkt`, embed in `cnf`).

**Impact**: Closes V3. Highest replay defense for browser/mobile clients.

#### Action 4: Fix Back-Channel Logout jti Tracking
**Effort**: 0.5 days

1. Replace the in-memory `sync.Map` (`backchannelLogoutList`) with Redis-backed tracking.
2. Use `SETNX` with TTL matching the logout token's `exp`.
3. This survives process restarts and works across multiple OAuth service instances.

**Impact**: Closes V4.

#### Action 5: Add Replay Anomaly Detection
**Effort**: 3-5 days

1. Implement `ReplayAnomalyDetector` as shown in section 6.2.
2. Integrate GeoIP (MaxMind GeoLite2) for impossible-travel detection.
3. Wire into the gateway middleware chain as a scoring layer (log-only initially,
   then escalate to block/challenge).
4. Emit audit events for anomalous activity.
5. Add admin dashboard for replay alerts in the console.

**Impact**: Closes V6. Improves detection of stolen-token usage.

### 8.3 Priority Matrix

| Action | Impact | Effort | Priority |
|---|---|---|---|
| 1. jti Replay Tracking | High | Low | **P0** — immediate |
| 2. mTLS Enforcement | High | Medium | **P1** — this sprint |
| 4. BC Logout jti Fix | Medium | Low | **P1** — quick win |
| 3. DPoP Support | High | High | **P2** — next sprint |
| 5. Anomaly Detection | Medium | Medium | **P2** — next sprint |

### 8.4 Key Takeaway

GGID's token infrastructure has the right building blocks (`jti` generation, mTLS
binding logic, refresh token replay detection) but lacks the gateway-level enforcement
that ties them together. The highest-impact, lowest-effort fix is adding jti tracking
to the gateway's `JWTAuth` middleware — a 2-3 day investment that closes the most
significant replay vulnerability. The mTLS enforcement leverages existing code and
requires only middleware wiring. DPoP is the strongest defense for client-facing APIs
but requires the most implementation effort.

---

## References

- [RFC 7519] JSON Web Token (JWT) — defines `jti` claim
- [RFC 6750] OAuth 2.0 Bearer Token Usage
- [RFC 8705] OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens
- [RFC 9449] OAuth 2.0 Demonstration of Proof-of-Possession at the Application Layer (DPoP)
- [RFC 9700] OAuth 2.0 Security Best Current Practice
- [RFC 7638] JSON Web Key (JWK) Thumbprint
- [RFC 9472] Token Status List (draft)

### GGID Source Files Reviewed

- `services/gateway/internal/middleware/middleware.go` — JWTAuth, JWKSClient
- `services/gateway/internal/middleware/session.go` — SessionManager
- `services/auth/internal/service/token_service.go` — IssueAccessToken, IssueRefreshToken, RotateRefreshToken
- `services/oauth/internal/service/oauth_service.go` — issueAccessToken, ParseAccessToken, ParseBackchannelLogoutToken
- `services/oauth/internal/service/jar_mtls.go` — ValidateMTLSClientAuth, ValidateMTLSBinding, ExtractCertThumbprint
- `services/oauth/internal/service/rfc7523.go` — ValidateClientAssertion (jti extraction)
- `services/oauth/internal/repository/pg_repo.go` — oidc_id_tokens table with jti column
