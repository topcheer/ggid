# DPoP Implementation Roadmap for GGID

> **Scope:** Concrete Go implementation guidance for DPoP (RFC 9449) in GGID.
> For RFC spec analysis, JWT format, and protocol flows, see
> [dpop-rfc9449.md](./dpop-rfc9449.md) (2,943 lines).
> For comparison with mTLS and Token Binding, see
> [token-binding-and-dpop.md](./token-binding-and-dpop.md) (340 lines).

**RFCs:** 9449 (DPoP), 7638 (JWK Thumbprint), 7800 (`cnf` claim)
**Status:** Draft | **Target Services:** OAuth (`services/oauth/`), Gateway (`services/gateway/`)

---

## 1. DPoP Nonce Binding

RFC 9449 Section 10.1 introduces AS-issued nonces to prevent replay attacks
where a pre-computed DPoP proof could be captured and replayed by an attacker.
The authorization server (GGID's OAuth service) issues a `DPoP-Nonce` challenge
header; the client must include the nonce in its next DPoP proof JWT's `nonce`
claim.

### Nonce Lifecycle

```
Client                         Authorization Server (GGID)
  |                                    |
  |  POST /oauth/token                 |
  |  DPoP: <proof-jwt>                 |
  |  ─────────────────────────────────►|
  |                                    | Verify proof → nonce missing/invalid
  |  ◄──── 400 invalid_dpop_nonce ────|
  |        DPoP-Nonce: <abc123>        |
  |                                    |
  |  POST /oauth/token (retry)         |
  |  DPoP: <proof-jwt with nonce>      |
  |  ─────────────────────────────────►|
  |                                    | Verify nonce == <abc123> in Redis
  |  ◄──── 200 {access_token, ...} ───|
  |        DPoP-Nonce: <def456>        |
```

### Go Implementation: Redis-Backed Nonce Store

```go
package dpop

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NonceManager issues and validates DPoP nonces using Redis with atomic
// single-use semantics. Each nonce can be used at most once within its TTL.
type NonceManager struct {
	rdb *redis.Client
	ttl time.Duration // default: 5 minutes
}

func NewNonceManager(rdb *redis.Client) *NonceManager {
	return &NonceManager{rdb: rdb, ttl: 5 * time.Minute}
}

// Issue generates a new nonce, stores it in Redis, and returns the value.
// The nonce is stored with a SET NX pattern so it can only be consumed once.
func (nm *NonceManager) Issue(ctx context.Context) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(b)

	// Store as "unconsumed" — the client will present it in the next proof.
	key := fmt.Sprintf("dpop:nonce:%s", nonce)
	if err := nm.rdb.Set(ctx, key, "0", nm.ttl).Err(); err != nil {
		return "", fmt.Errorf("store nonce: %w", err)
	}
	return nonce, nil
}

// ValidateAndConsume atomically marks the nonce as consumed. Returns an error
// if the nonce is unknown, already consumed, or expired.
func (nm *NonceManager) ValidateAndConsume(ctx context.Context, nonce string) error {
	key := fmt.Sprintf("dpop:nonce:%s", nonce)

	// Lua script for atomic check-and-mark-consumed.
	// Returns 1 on success, 0 if already consumed, -1 if not found.
	script := redis.NewScript(`
		local v = redis.call("GET", KEYS[1])
		if v == false then return -1 end
		if v == "1" then return 0 end
		redis.call("SET", KEYS[1], "1", "PX", ARGV[1])
		return 1
	`)

	result, err := script.Run(ctx, nm.rdb, []string{key},
		nm.ttl.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("nonce validation: %w", err)
	}

	switch result {
	case 1:
		return nil
	case 0:
		return ErrNonceAlreadyConsumed
	default:
		return ErrNonceUnknown
	}
}

var (
	ErrNonceUnknown         = fmt.Errorf("dpop: nonce unknown or expired")
	ErrNonceAlreadyConsumed = fmt.Errorf("dpop: nonce already consumed")
)
```

### Challenge Response in Token Endpoint

When a DPoP proof is received without a valid nonce, the server returns a
challenge with a fresh nonce:

```go
// In the token endpoint handler (services/oauth/internal/server/server.go):
func challengeDPoPNonce(w http.ResponseWriter, r *http.Request, nm *dpop.NonceManager) {
	ctx := r.Context()
	nonce, err := nm.Issue(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "server_error",
		})
		return
	}
	w.Header().Set("DPoP-Nonce", nonce)
	w.Header().Set("WWW-Authenticate",
		`DPoP error="use_dpop_nonce", error_description="DPoP nonce required"`)
	writeJSON(w, http.StatusBadRequest, map[string]string{
		"error":             "use_dpop_nonce",
		"error_description": "Authorization server requires nonce in DPoP proof",
	})
}
```

---

## 2. Token Endpoint DPoP Verification

The token endpoint (`/oauth/token` in `services/oauth/internal/server/server.go`,
line 293) must validate the `DPoP` header before issuing tokens. The full
verification checks the JWT structure, signature, freshness, and replay
prevention.

### Go Implementation: VerifyDPoPProof

```go
package dpop

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// DPoPProofClaims represents the required claims in a DPoP proof JWT.
type DPoPProofClaims struct {
	HTM   string `json:"htm"`             // HTTP method (e.g., "POST")
	HTU   string `json:"htu"`             // HTTP URI (no query/fragment)
	IAT   int64  `json:"iat"`             // Issued-at timestamp (unix seconds)
	JTI   string `json:"jti"`             // Unique JWT ID for replay prevention
	Ath   string `json:"ath,omitempty"`   // Access token hash (resource requests only)
	Nonce string `json:"nonce,omitempty"` // AS-issued nonce (when nonce binding active)
	jwt.RegisteredClaims
}

// VerifyDPoPProof validates a DPoP proof JWT against RFC 9449 requirements.
// Parameters:
//   - proofStr:  raw value of the DPoP HTTP header
//   - method:    expected HTTP method (e.g., "POST")
//   - uri:       expected request URI (no query string)
//   - maxAge:    maximum acceptable proof age (default 60s per RFC 9449)
//   - jtiStore:  Redis client for jti replay prevention
func VerifyDPoPProof(
	ctx context.Context,
	proofStr, method, uri string,
	maxAge time.Duration,
	jtiStore *redis.Client,
) (string, *ecdsa.PublicKey, error) {

	if proofStr == "" {
		return "", nil, ErrMissingDPoPHeader
	}

	// 1. Parse the JWT without verification first to extract the header key.
	parser := jwt.NewParser()
	token, parts, err := parser.ParseUnverified(proofStr, &DPoPProofClaims{})
	if err != nil {
		return "", nil, fmt.Errorf("parse DPoP proof: %w", err)
	}

	// 2. Check typ header — must be exactly "dpop+jwt" (RFC 9449 Section 4.2).
	typ, ok := token.Header["typ"].(string)
	if !ok || !strings.EqualFold(typ, "dpop+jwt") {
		return "", nil, ErrInvalidTyp
	}

	// 3. Extract the public key from the JWT header (jwk claim).
	jwkBytes, err := json.Marshal(token.Header["jwk"])
	if err != nil {
		return "", nil, fmt.Errorf("marshal jwk: %w", err)
	}
	pubKey, err := jwkToPublicKey(jwkBytes)
	if err != nil {
		return "", nil, fmt.Errorf("extract key from jwk: %w", err)
	}

	// 4. Verify the JWT signature using the embedded public key.
	claims := &DPoPProofClaims{}
	_, err = jwt.ParseWithClaims(proofStr, claims, func(t *jwt.Token) (any, error) {
		// DPoP proofs must use asymmetric signing (ES256, RS256, EdDSA).
		switch t.Method.(type) {
		case *jwt.SigningMethodECDSA, *jwt.SigningMethodRSA, *jwt.SigningMethodEd25519:
			return pubKey, nil
		default:
			return nil, fmt.Errorf("unsupported signing method: %v", t.Header["alg"])
		}
	})
	if err != nil {
		return "", nil, fmt.Errorf("verify DPoP signature: %w", err)
	}

	// 5. Verify htm (HTTP method) matches the request.
	if !strings.EqualFold(claims.HTM, method) {
		return "", nil, fmt.Errorf("%w: expected %s, got %s", ErrMethodMismatch, method, claims.HTM)
	}

	// 6. Verify htu (HTTP URI) matches — normalize by stripping query/fragment.
	htu, err := normalizeHTU(claims.HTU)
	if err != nil {
		return "", nil, fmt.Errorf("normalize htu: %w", err)
	}
	expectedHTU, _ := normalizeHTU(uri)
	if htu != expectedHTU {
		return "", nil, fmt.Errorf("%w: expected %s, got %s", ErrURIMismatch, expectedHTU, htu)
	}

	// 7. Verify iat freshness — proof must be issued within ±maxAge window.
	if maxAge == 0 {
		maxAge = 60 * time.Second
	}
	now := time.Now()
	iatTime := time.Unix(claims.IAT, 0)
	if now.Sub(iatTime).Abs() > maxAge {
		return "", nil, fmt.Errorf("%w: iat %s, now %s, maxAge %s",
			ErrProofExpired, iatTime.Format(time.RFC3339),
			now.Format(time.RFC3339), maxAge)
	}

	// 8. Replay prevention via jti — store in Redis with TTL = maxAge.
	if claims.JTI == "" {
		return "", nil, ErrMissingJTI
	}
	jtiKey := fmt.Sprintf("dpop:jti:%s", claims.JTI)
	if jtiStore != nil {
		set, err := jtiStore.SetNX(ctx, jtiKey, "1", maxAge).Result()
		if err != nil {
			return "", nil, fmt.Errorf("jti replay check: %w", err)
		}
		if !set {
			return "", nil, ErrReplayedJTI
		}
	}

	// 9. Compute the JWK thumbprint for token binding.
	jkt, err := ComputeJWKThumbprint(jwkBytes)
	if err != nil {
		return "", nil, fmt.Errorf("compute thumbprint: %w", err)
	}

	return jkt, pubKey, nil
}

// normalizeHTU strips query parameters and fragments from a URI and
// lowercases the scheme and host per RFC 9449 Section 4.3.
func normalizeHTU(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	return u.String(), nil
}
```

---

## 3. Resource Server DPoP Verification

The API Gateway (`services/gateway/`) acts as the resource server. It must
validate DPoP proofs for every API request that carries a DPoP-bound access
token. The key difference from token-endpoint verification is that the gateway
must handle **both** bearer tokens and DPoP-bound tokens simultaneously.

### Dual-Mode Auth Middleware

The existing `JWTAuth` middleware at `services/gateway/internal/middleware/middleware.go`
line 499 only checks bearer tokens. We add a `DPoPAuth` wrapper that runs after
JWT validation to enforce DPoP on tokens containing a `cnf.jkt` claim.

```go
package dpop

// DPoPAuth wraps the existing JWTAuth middleware to add DPoP enforcement.
// If the access token has a "cnf.jkt" claim, the request MUST include a valid
// DPoP proof whose key thumbprint matches. If no cnf claim exists, the token
// is treated as a standard bearer token (no DPoP proof required).
func DPoPAuth(jtiStore *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract claims from context (set by JWTAuth middleware).
			jkt, ok := DPoPThumbprintFromContext(r.Context())
			if !ok || jkt == "" {
				// Standard bearer token — no DPoP enforcement.
				next.ServeHTTP(w, r)
				return
			}

			// DPoP-bound token — proof is mandatory.
			proof := r.Header.Get("DPoP")
			if proof == "" {
				w.Header().Set("WWW-Authenticate",
					`DPoP error="invalid_dpop_proof"`)
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error":             "invalid_dpop_proof",
					"error_description": "DPoP-bound token requires DPoP proof header",
				})
				return
			}

			// Verify the proof for this API request.
			ctx := r.Context()
			proofJKT, _, err := VerifyDPoPProof(ctx, proof,
				r.Method, constructHTU(r), 60*time.Second, jtiStore)

			if err != nil {
				w.Header().Set("WWW-Authenticate",
					fmt.Sprintf(`DPoP error="invalid_dpop_proof" error_description="%s"`,
						sanitizeErr(err)))
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error":             "invalid_dpop_proof",
					"error_description": err.Error(),
				})
				return
			}

			// 10. Verify key thumbprint matches the token's cnf.jkt.
			if proofJKT != jkt {
				w.Header().Set("WWW-Authenticate",
					`DPoP error="invalid_token"`)
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error":             "invalid_token",
					"error_description": "DPoP key does not match token binding",
				})
				return
			}

			// 11. Verify ath (access token hash) matches the presented token.
			accessToken := extractBearerToken(r.Header.Get("Authorization"))
			expectedATH := hashAccessToken(accessToken)
			claims := parseProofClaims(proof) // unverified parse for ath check
			if claims.Ath != expectedATH {
				w.Header().Set("WWW-Authenticate",
					`DPoP error="invalid_dpop_proof"`)
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error":             "invalid_dpop_proof",
					"error_description": "ath claim does not match access token hash",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// hashAccessToken computes the base64url(SHA-256) hash of the access token,
// as required by RFC 9449 Section 7.2 for the ath claim.
func hashAccessToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// constructHTU builds the canonical htu for the incoming request.
// Uses the forwarded scheme/host if behind a proxy.
func constructHTU(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "https"
	}
	host := r.Host
	return fmt.Sprintf("%s://%s%s", scheme, host, r.URL.Path)
}
```

### Integration in JWTAuth Middleware

Modify `JWTAuth()` in `middleware.go` to extract `cnf.jkt` from token claims
and inject it into context for the `DPoPAuth` wrapper:

```go
// Inside JWTAuth(), after extracting claims (around line 566):
if cnf, ok := claims["cnf"].(map[string]any); ok {
	if jkt, ok := cnf["jkt"].(string); ok {
		ctx = context.WithValue(ctx, DPoPThumbprintKey, jkt)
	}
}
```

The handler chain in `router.go` (line 330-334) would then wrap:

```go
// Before: jwtMW(gw).ServeHTTP(w, r)
// After:
jwtMW := middleware.JWTAuth(gw.jwks, isPublic, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
dpopMW := dpop.DPoPAuth(gw.jtiStore)
dpopMW(jwtMW(gw)).ServeHTTP(w, r)
```

---

## 4. DPoP-Bound Refresh Tokens

Refresh tokens in GGID are stored as opaque tokens in PostgreSQL (see
`oauth_service.go` line 690, `RefreshToken()` method). To bind a refresh token
to a DPoP key, the key thumbprint is stored alongside the token record and
verified on each refresh request.

### Data Model Extension

```go
// In services/oauth/internal/domain/types.go (or equivalent):
type RefreshTokenRecord struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	ClientID  uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	Scope     []string
	ExpiresAt time.Time
	Used      bool
	Revoked   bool

	// DPoP binding — set when the original token request included a DPoP proof.
	// Empty string means no DPoP binding (standard bearer refresh token).
	DPoPThumbprint string `json:"dpop_jkt,omitempty"`
}
```

### Refresh Token Validation with DPoP

```go
// Modified RefreshToken method in oauth_service.go:
func (s *OAuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenResponse, error) {
	// ... existing client lookup (lines 691-727) ...

	// NEW: If the refresh token record has a DPoP thumbprint, require a
	// valid DPoP proof from the same key on the refresh request.
	if record.DPoPThumbprint != "" {
		if req.DPoPProof == "" {
			return nil, errors.Unauthenticated("DPoP proof required for DPoP-bound refresh token")
		}

		// Verify the DPoP proof at the token endpoint.
		jkt, _, err := dpop.VerifyDPoPProof(ctx, req.DPoPProof,
			"POST", s.issuer+"/oauth/token", 60*time.Second, s.jtiStore)
		if err != nil {
			return nil, errors.Unauthenticated("invalid DPoP proof: " + err.Error())
		}

		// The proof's key thumbprint MUST match the one bound to the refresh token.
		if jkt != record.DPoPThumbprint {
			// Key mismatch → possible token theft. Revoke all tokens for this client.
			_ = s.tokenRepo.RevokeAllRefreshTokens(ctx, req.TenantID, client.ID)
			return nil, errors.Unauthenticated("DPoP key mismatch — all tokens revoked")
		}
	}

	// ... existing token rotation logic (lines 729-760) ...

	// NEW: Propagate the DPoP binding to the new refresh token.
	newRecord.DPoPThumbprint = record.DPoPThumbprint

	// NEW: Add cnf.jkt to the new access token if DPoP-bound.
	if newRecord.DPoPThumbprint != "" {
		// Inject into access token claims in issueAccessToken()
		accessToken = s.issueDPoPBoundAccessToken(userID, tenantID, audience, newRecord.DPoPThumbprint)
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "DPoP", // RFC 9449: token_type is "DPoP" not "Bearer"
		ExpiresIn:   expiresIn,
		RefreshToken: newRefreshToken,
	}, nil
}

// issueDPoPBoundAccessToken adds the cnf claim to the JWT.
func (s *OAuthService) issueDPoPBoundAccessToken(userID, tenantID uuid.UUID, audience, jkt string) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	claims := jwt.MapClaims{
		"iss":       s.issuer,
		"sub":       userID.String(),
		"aud":       audience,
		"iat":       now.Unix(),
		"exp":       expiresAt.Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": tenantID.String(),
		"cnf": map[string]string{
			"jkt": jkt, // RFC 7800 confirmation claim with JWK thumbprint
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyProvider.KeyID()
	signed, err := token.SignedString(s.keyProvider.PrivateKey())
	if err != nil {
		return "", 0, fmt.Errorf("sign DPoP-bound access token: %w", err)
	}
	return signed, int(expiresAt.Sub(now).Seconds()), nil
}
```

---

## 5. JWK Thumbprint Computation

RFC 7638 defines the JWK thumbprint: a canonical hash of a JSON Web Key that
serves as a stable identifier for a DPoP key pair. The `cnf.jkt` claim in
access tokens and the proof JWT's embedded key are linked via this thumbprint.

### Canonical Rules

1. **Lexicographic key ordering**: JSON members must be sorted alphabetically.
2. **No whitespace**: No spaces, newlines, or indentation.
3. **Required members only**: Only the minimum required key parameters are
   included. For EC keys: `crv`, `kty`, `x`, `y`. For RSA: `e`, `kty`, `n`.
   For OKP (Ed25519): `crv`, `kty`, `x`.
4. **SHA-256 hash**: Compute SHA-256 over the canonical JSON bytes.
5. **Base64url encode**: No padding.

### Go Implementation

```go
package dpop

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// ComputeJWKThumbprint computes the RFC 7638 thumbprint of a JWK.
// Input is the raw JSON of the "jwk" member from the DPoP proof header.
func ComputeJWKThumbprint(jwkJSON []byte) (string, error) {
	// Parse the JWK to determine key type and extract required members.
	var jwk map[string]json.RawMessage
	if err := json.Unmarshal(jwkJSON, &jwk); err != nil {
		return "", fmt.Errorf("unmarshal jwk: %w", err)
	}

	kty, err := rawString(jwk["kty"])
	if err != nil {
		return "", fmt.Errorf("missing kty: %w", err)
	}

	// Build the canonical JSON with ONLY required members in sorted order.
	// Go's json.Marshal sorts map keys lexicographically for map[string]any,
	// but we use an explicit struct for safety and clarity.
	var canonical []byte
	switch kty {
	case "EC":
		crv, _ := rawString(jwk["crv"])
		x, _ := rawString(jwk["x"])
		y, _ := rawString(jwk["y"])
		canonical, err = json.Marshal(struct {
			Crv string `json:"crv"`
			Kty string `json:"kty"`
			X   string `json:"x"`
			Y   string `json:"y"`
		}{crv, kty, x, y})
		if err != nil {
			return "", fmt.Errorf("marshal EC canonical: %w", err)
		}

	case "RSA":
		e, _ := rawString(jwk["e"])
		n, _ := rawString(jwk["n"])
		canonical, err = json.Marshal(struct {
			E   string `json:"e"`
			Kty string `json:"kty"`
			N   string `json:"n"`
		}{e, kty, n})
		if err != nil {
			return "", fmt.Errorf("marshal RSA canonical: %w", err)
		}

	case "OKP":
		crv, _ := rawString(jwk["crv"])
		x, _ := rawString(jwk["x"])
		canonical, err = json.Marshal(struct {
			Crv string `json:"crv"`
			Kty string `json:"kty"`
			X   string `json:"x"`
		}{crv, kty, x})
		if err != nil {
			return "", fmt.Errorf("marshal OKP canonical: %w", err)
		}

	default:
		return "", fmt.Errorf("unsupported key type: %s", kty)
	}

	// SHA-256 hash of canonical JSON, base64url encoded without padding.
	h := sha256.Sum256(canonical)
	return base64.RawURLEncoding.EncodeToString(h[:]), nil
}

// rawString extracts a JSON string value from a RawMessage.
func rawString(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}

// jwkToPublicKey converts a JWK JSON blob to a crypto.PublicKey.
func jwkToPublicKey(jwkJSON []byte) (any, error) {
	var jwk struct {
		Kty string `json:"kty"`
		Crv string `json:"crv"`
		X   string `json:"x"`
		Y   string `json:"y"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	if err := json.Unmarshal(jwkJSON, &jwk); err != nil {
		return nil, err
	}

	switch jwk.Kty {
	case "EC":
		xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil {
			return nil, err
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
		if err != nil {
			return nil, err
		}
		var curve elliptic.Curve
		switch jwk.Crv {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("unsupported EC curve: %s", jwk.Crv)
		}
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}, nil

	case "RSA":
		nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			return nil, err
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil {
			return nil, err
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}, nil

	case "OKP":
		xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil {
			return nil, err
		}
		return ed25519.PublicKey(xBytes), nil

	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}
```

### Edge Cases

| Edge Case | Handling |
|-----------|----------|
| Extra JWK members (e.g., `kid`, `use`, `alg`) | **Excluded** from canonical JSON. Only RFC 7638 required members participate. |
| Different member ordering in input JWK | **Irrelevant**. The canonical form always sorts lexicographically. |
| URL-safe vs standard base64 in input | Use `base64.RawURLEncoding` for all JWK member values. |
| EC key with `d` (private exponent) | **Must ignore** `d` — thumbprint is of the public key only. |
| RSA key with private members (`d`, `p`, `q`) | **Must ignore** all private parameters. |
| Key rotation (client generates new key pair) | Old `cnf.jkt` tokens remain valid until expiry. New tokens get new thumbprint. |

---

## 6. GGID DPoP Integration Points

Based on analysis of the actual GGID source files, DPoP verification hooks into
the following locations:

### 6.1 Token Endpoint (`services/oauth/internal/server/server.go`)

**Location:** Line 293 — `mux.HandleFunc("/oauth/token", ...)`

The token endpoint handler processes all grant types (authorization_code,
refresh_token, client_credentials, device_code, jwt-bearer). DPoP proof
verification must occur **before** the grant-type switch at line 325:

```go
// INSERT BEFORE LINE 325 (grantType switch):
dpopProof := r.Header.Get("DPoP")
var dpopJKT string
if dpopProof != "" {
	jkt, _, err := dpop.VerifyDPoPProof(ctx, dpopProof, "POST",
		cfg.Issuer+"/oauth/token", 60*time.Second, jtiStore)
	if err != nil {
		// If nonce binding is enabled, challenge with a fresh nonce.
		challengeDPoPNonce(w, r, nonceManager)
		return
	}
	dpopJKT = jkt
}
// Pass dpopJKT to each grant-type handler for token binding.
```

### 6.2 Access Token Issuance (`services/oauth/internal/service/oauth_service.go`)

**Location:** Line 402 — `issueAccessToken()`

Add an optional `cnf` claim when a DPoP key is present:

```go
// Modified signature:
func (s *OAuthService) issueAccessToken(userID, tenantID uuid.UUID, audience, dpopJKT string) (string, int, error) {
	// ... existing claims map (lines 416-424) ...
	if dpopJKT != "" {
		claimsMap["cnf"] = map[string]string{"jkt": dpopJKT}
	}
	// ... rest of signing logic ...
}
```

### 6.3 Gateway JWT Middleware (`services/gateway/internal/middleware/middleware.go`)

**Location:** Line 499 — `JWTAuth()` function, specifically around line 566
where claims are processed.

Extract `cnf.jkt` and inject into context for downstream DPoP validation:

```go
// After line 573 (tenant_id extraction):
if cnf, ok := claims["cnf"].(map[string]any); ok {
	if jkt, ok := cnf["jkt"].(string); ok {
		ctx = context.WithValue(ctx, DPoPThumbprintKey, jkt)
	}
}
```

### 6.4 Gateway Handler Chain (`services/gateway/internal/router/router.go`)

**Location:** Lines 328-336 — the inner handler that applies JWTAuth.

The existing chain (line 339):
```
PanicRecovery → SecurityHeaders → CORS → RequestID → Logging → RateLimit → TenantResolver
  → inner(JWTAuth → proxy)
```

DPoP validation wraps around the proxy, **after** JWTAuth:

```go
// Lines 330-335 become:
jwtMW := middleware.JWTAuth(gw.jwks, isPublic, gw.cfg.JWTIssuer, gw.cfg.JWTAudience)
dpopMW := middleware.DPoPAuth(gw.jtiStore) // new
dpopMW(jwtMW(gw)).ServeHTTP(w, r)
```

### 6.5 JWKS Endpoint (`services/gateway/internal/middleware/middleware.go`)

**Location:** Line 475 — `JWKSHandler()`, served at `/.well-known/jwks.json`
(router.go line 187).

No change required. The JWKS endpoint serves the AS signing key for JWT
verification. DPoP client keys are self-signed and embedded directly in the
proof JWT header — they are not published in a JWKS endpoint.

### 6.6 OAuth Discovery Document

**Location:** `oauth_service.go` line 360 — `GetDiscoveryDocument()`.

Add DPoP-related metadata fields:

```go
// In the discovery response:
DPoPSigningAlgValuesSupported: []string{"ES256", "RS256", "EdDSA"},
DPoPBoundAccessTokenRequired:  false, // set true when DPoP becomes mandatory
```

### 6.7 Redis Dependency

GGID's gateway already uses Redis for rate limiting and caching. The same Redis
instance serves double duty for:
- `dpop:nonce:<nonce>` — nonce lifecycle (TTL 5min)
- `dpop:jti:<jti>` — proof replay prevention (TTL 60s)

No new infrastructure required.

---

## 7. Testing Strategy

### 7.1 Unit Tests

```go
package dpop_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/dpop"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// Helper: generate an EC P-256 key pair and build a DPoP proof JWT.
func makeDPoPProof(t *testing.T, privKey *ecdsa.PrivateKey, htm, htu string,
	iat time.Time, jti string, nonce string) string {
	t.Helper()

	pubJWK := map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(privKey.X.Bytes()),
		"y":   base64.RawURLEncoding.EncodeToString(privKey.Y.Bytes()),
	}

	claims := &dpop.DPoPProofClaims{
		HTM: htm,
		HTU: htu,
		IAT: iat.Unix(),
		JTI: jti,
	}
	if nonce != "" {
		claims.Nonce = nonce
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = pubJWK

	signed, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("sign proof: %v", err)
	}
	return signed
}

// --- Positive Test ---
func TestVerifyDPoPProof_Valid(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	jti := "unique-jti-001"
	proof := makeDPoPProof(t, privKey, "POST",
		"https://as.example.com/oauth/token", time.Now(), jti, "")

	// Use miniredis for testing — see github.com/alicebob/miniredis/v2
	mr, rdb := setupTestRedis(t)
	defer mr.Close()

	jkt, pubKey, err := dpop.VerifyDPoPProof(context.Background(), proof,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, rdb)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if jkt == "" {
		t.Error("expected non-empty JWK thumbprint")
	}
	if pubKey == nil {
		t.Error("expected non-nil public key")
	}
}

// --- Negative Tests ---

func TestVerifyDPoPProof_WrongTyp(t *testing.T) {
	// Build proof with typ=JWT instead of dpop+jwt
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	token := jwt.NewWithClaims(jwt.SigningMethodES256, &dpop.DPoPProofClaims{
		HTM: "POST", HTU: "https://as.example.com/oauth/token",
		IAT: time.Now().Unix(), JTI: "x",
	})
	token.Header["typ"] = "JWT" // wrong
	token.Header["jwk"] = makeECJWK(privKey)
	proof, _ := token.SignedString(privKey)

	_, _, err := dpop.VerifyDPoPProof(context.Background(), proof,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, nil)
	if err == nil {
		t.Error("expected error for wrong typ")
	}
}

func TestVerifyDPoPProof_ExpiredProof(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// iat is 2 minutes ago — outside the 60s window.
	oldTime := time.Now().Add(-2 * time.Minute)
	proof := makeDPoPProof(t, privKey, "POST",
		"https://as.example.com/oauth/token", oldTime, "jti-old", "")

	_, _, err := dpop.VerifyDPoPProof(context.Background(), proof,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, nil)
	if err == nil {
		t.Error("expected error for expired proof")
	}
}

func TestVerifyDPoPProof_WrongHTM(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	proof := makeDPoPProof(t, privKey, "POST",
		"https://as.example.com/oauth/token", time.Now(), "jti-htm", "")

	_, _, err := dpop.VerifyDPoPProof(context.Background(), proof,
		"GET", "https://as.example.com/oauth/token", 60*time.Second, nil)
	if err == nil {
		t.Error("expected method mismatch error")
	}
}

func TestVerifyDPoPProof_WrongHTU(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	proof := makeDPoPProof(t, privKey, "POST",
		"https://evil.example.com/oauth/token", time.Now(), "jti-htu", "")

	_, _, err := dpop.VerifyDPoPProof(context.Background(), proof,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, nil)
	if err == nil {
		t.Error("expected URI mismatch error")
	}
}

func TestVerifyDPoPProof_ReplayedJTI(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	proof := makeDPoPProof(t, privKey, "POST",
		"https://as.example.com/oauth/token", time.Now(), "jti-replay", "")

	// First use — should succeed.
	_, _, err := dpop.VerifyDPoPProof(context.Background(), proof,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, rdb)
	if err != nil {
		t.Fatalf("first use failed: %v", err)
	}

	// Second use — should fail (replay).
	_, _, err = dpop.VerifyDPoPProof(context.Background(), proof,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, rdb)
	if err == nil {
		t.Error("expected replay error")
	}
}

func TestVerifyDPoPProof_KeyMismatch(t *testing.T) {
	// Token is bound to key A, but proof uses key B.
	privKeyA, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privKeyB, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	proofA := makeDPoPProof(t, privKeyA, "POST",
		"https://as.example.com/oauth/token", time.Now(), "jti-a", "")
	jktA, _, _ := dpop.VerifyDPoPProof(context.Background(), proofA,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, nil)

	proofB := makeDPoPProof(t, privKeyB, "POST",
		"https://as.example.com/oauth/token", time.Now(), "jti-b", "")
	jktB, _, _ := dpop.VerifyDPoPProof(context.Background(), proofB,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, nil)

	if jktA == jktB {
		t.Error("expected different thumbprints for different keys")
	}
}

// --- JWK Thumbprint Tests ---

func TestComputeJWKThumbprint_EC(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	jwkJSON := fmt.Sprintf(`{"crv":"P-256","kty":"EC","x":"%s","y":"%s"}`,
		base64.RawURLEncoding.EncodeToString(privKey.X.Bytes()),
		base64.RawURLEncoding.EncodeToString(privKey.Y.Bytes()))

	jkt, err := dpop.ComputeJWKThumbprint([]byte(jwkJSON))
	if err != nil {
		t.Fatalf("compute thumbprint: %v", err)
	}
	if len(jkt) != 43 { // SHA-256 = 32 bytes → 43 chars base64url
		t.Errorf("expected 43-char thumbprint, got %d chars: %s", len(jkt), jkt)
	}
}

func TestComputeJWKThumbprint_CanonicalOrdering(t *testing.T) {
	// Same key, different JSON ordering — must produce same thumbprint.
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	x := base64.RawURLEncoding.EncodeToString(privKey.X.Bytes())
	y := base64.RawURLEncoding.EncodeToString(privKey.Y.Bytes())

	jwk1 := fmt.Sprintf(`{"crv":"P-256","kty":"EC","x":"%s","y":"%s"}`, x, y)
	jwk2 := fmt.Sprintf(`{"y":"%s","x":"%s","kty":"EC","crv":"P-256"}`, y, x)

	jkt1, _ := dpop.ComputeJWKThumbprint([]byte(jwk1))
	jkt2, _ := dpop.ComputeJWKThumbprint([]byte(jwk2))

	if jkt1 != jkt2 {
		t.Errorf("thumbprint must be order-independent: %s != %s", jkt1, jkt2)
	}
}

// --- Nonce Tests ---

func TestNonceManager_IssueAndConsume(t *testing.T) {
	mr, rdb := setupTestRedis(t)
	defer mr.Close()

	nm := dpop.NewNonceManager(rdb)
	ctx := context.Background()

	nonce, err := nm.Issue(ctx)
	if err != nil {
		t.Fatalf("issue nonce: %v", err)
	}

	// First consume — success.
	if err := nm.ValidateAndConsume(ctx, nonce); err != nil {
		t.Errorf("consume valid nonce: %v", err)
	}

	// Second consume — should fail.
	if err := nm.ValidateAndConsume(ctx, nonce); err != dpop.ErrNonceAlreadyConsumed {
		t.Errorf("expected ErrNonceAlreadyConsumed, got: %v", err)
	}

	// Unknown nonce — should fail.
	if err := nm.ValidateAndConsume(ctx, "nonexistent"); err != dpop.ErrNonceUnknown {
		t.Errorf("expected ErrNonceUnknown, got: %v", err)
	}
}

// --- Integration Test: Full DPoP Flow ---

func TestIntegration_DPoPFlow(t *testing.T) {
	// This test exercises the full flow:
	// 1. Client generates EC key pair
	// 2. Client sends DPoP proof to token endpoint
	// 3. Server verifies proof, issues DPoP-bound access token
	// 4. Client calls resource server with token + DPoP proof
	// 5. Gateway validates token, extracts cnf.jkt, verifies proof

	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	mr, rdb := setupTestRedis(t)
	defer mr.Close()

	// Step 2: Token endpoint proof
	tokenProof := makeDPoPProof(t, privKey, "POST",
		"https://as.example.com/oauth/token", time.Now(), "token-jti", "")

	jkt, _, err := dpop.VerifyDPoPProof(context.Background(), tokenProof,
		"POST", "https://as.example.com/oauth/token", 60*time.Second, rdb)
	if err != nil {
		t.Fatalf("token endpoint proof: %v", err)
	}

	// Step 4: Resource request proof
	apiProof := makeDPoPProof(t, privKey, "GET",
		"https://api.example.com/v1/users", time.Now(), "api-jti", "")

	apiJKT, _, err := dpop.VerifyDPoPProof(context.Background(), apiProof,
		"GET", "https://api.example.com/v1/users", 60*time.Second, rdb)
	if err != nil {
		t.Fatalf("API proof: %v", err)
	}

	// Step 5: Thumbprints must match
	if apiJKT != jkt {
		t.Fatalf("thumbprint mismatch: token=%s, proof=%s", jkt, apiJKT)
	}
}

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Skipf("miniredis unavailable: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}
```

### 7.2 Test Matrix

| Test | Input | Expected |
|------|-------|----------|
| Valid proof | Correct typ, sig, htm, htu, iat, jti | Pass, return JKT |
| Missing DPoP header | Empty string | `ErrMissingDPoPHeader` |
| Wrong typ | `typ: "JWT"` | `ErrInvalidTyp` |
| Expired proof | `iat` 2min ago | `ErrProofExpired` |
| Future proof | `iat` 2min ahead | `ErrProofExpired` |
| Wrong htm | Proof POST, request GET | `ErrMethodMismatch` |
| Wrong htu | Different host | `ErrURIMismatch` |
| Replayed jti | Same jti twice | `ErrReplayedJTI` |
| Key mismatch | Token bound to key A, proof from key B | 401 at gateway |
| Missing ath | Resource request without ath claim | 401 at gateway |
| Nonce consumed | Second use of same nonce | `ErrNonceAlreadyConsumed` |
| Nonce unknown | Random nonce not from AS | `ErrNonceUnknown` |

---

## 8. Implementation Roadmap

### Phase 1: Core DPoP Package (3-4 days)

**Deliverable:** `pkg/dpop/` package with proof verification and thumbprint computation.

| Item | Effort | Dependencies |
|------|--------|--------------|
| `ComputeJWKThumbprint()` (RFC 7638) | 0.5 day | None |
| `VerifyDPoPProof()` (full validation) | 1.5 days | `ComputeJWKThumbprint` |
| `NonceManager` (Redis-backed) | 0.5 day | Redis (already deployed) |
| `jwkToPublicKey()` converter | 0.5 day | None |
| Unit tests (12+ test cases) | 1 day | All above |

**File structure:**
```
pkg/dpop/
  thumbprint.go      — JWK thumbprint computation
  verify.go          — VerifyDPoPProof
  nonce.go           — NonceManager
  jwk.go             — JWK-to-PublicKey conversion
  errors.go          — Sentinel errors
  dpop_test.go       — Unit tests
```

### Phase 2: Token Endpoint Integration (2 days)

**Deliverable:** DPoP proof verification at `/oauth/token`, DPoP-bound token issuance.

| Item | Effort | Dependencies |
|------|--------|--------------|
| Add DPoP header parsing in token handler (server.go:293) | 0.5 day | Phase 1 |
| Add `cnf.jkt` to `issueAccessToken()` (oauth_service.go:402) | 0.5 day | Phase 1 |
| Add `DPoPThumbprint` to `RefreshTokenRecord` domain model | 0.5 day | DB migration |
| Add nonce challenge response for nonce binding mode | 0.5 day | `NonceManager` |

**Files modified:**
- `services/oauth/internal/server/server.go` (token handler)
- `services/oauth/internal/service/oauth_service.go` (token issuance)
- `services/oauth/internal/domain/types.go` (RefreshTokenRecord)
- Migration: `ALTER TABLE oauth_refresh_tokens ADD COLUMN dpop_jkt TEXT`

### Phase 3: Gateway Resource Server Middleware (2 days)

**Deliverable:** Dual-mode DPoP/bearer auth middleware in the gateway.

| Item | Effort | Dependencies |
|------|--------|--------------|
| Extract `cnf.jkt` in `JWTAuth()` (middleware.go:499) | 0.5 day | Phase 2 |
| `DPoPAuth()` middleware for resource requests | 1 day | Phase 1 |
| Wire into handler chain (router.go:330) | 0.25 day | DPoPAuth |
| `ath` claim verification | 0.25 day | DPoPAuth |

### Phase 4: DPoP-Bound Refresh Tokens (1 day)

**Deliverable:** Refresh token rotation with DPoP key binding.

| Item | Effort | Dependencies |
|------|--------|--------------|
| Verify DPoP proof on refresh (oauth_service.go:690) | 0.5 day | Phase 2 |
| Propagate binding to new refresh token | 0.5 day | Phase 2 |

### Phase 5: Discovery and Documentation (0.5 day)

| Item | Effort | Dependencies |
|------|--------|--------------|
| Add DPoP metadata to discovery document | 0.25 day | Phase 2 |
| Update API documentation (OpenAPI/Swagger) | 0.25 day | Phase 3 |

### Total Effort Estimate: 8-10 days

### Dependencies on Existing GGID Infrastructure

| Dependency | Status | Notes |
|------------|--------|-------|
| Redis | **Deployed** | Already used for rate limiting and caching |
| `golang-jwt/jwt/v5` | **In use** | Already imported in OAuth service and gateway |
| `redis/go-redis/v9` | **In use** | Gateway and auth service already import this |
| PostgreSQL | **Deployed** | `oauth_refresh_tokens` table needs column add |
| JWKS endpoint | **Deployed** | No change needed — DPoP keys are self-contained |
| Gateway handler chain | **Active** | DPoP middleware wraps around existing JWTAuth |

### Risk Mitigations

1. **Backward compatibility**: DPoP is opt-in. Tokens without `cnf.jkt` remain
   standard bearer tokens. No existing client breaks.
2. **Performance**: ECDSA verification adds ~0.1ms per request. Redis SET NX
   for `jti` adds ~0.3ms. Total overhead < 1ms per API call.
3. **Key rotation**: Clients rotating their DPoP key pair must re-authenticate.
   Existing tokens remain valid until expiry. No server-side key store needed.
4. **Nonce binding**: Start with nonce binding **disabled** (optional). Enable
   per-tenant once client SDKs are updated to handle nonce challenges.
