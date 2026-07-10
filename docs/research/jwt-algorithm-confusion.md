# JWT Algorithm Confusion Attack

> Research document examining JWT algorithm confusion vulnerabilities, with a security audit of GGID's JWT validation implementation.

---

## 1. Overview

JWT algorithm confusion is a critical authentication bypass where an attacker tricks the server into using the wrong cryptographic algorithm for token verification. The result: the attacker can forge valid JWTs without possessing the signing key.

**Classic attack vector:**
- Server expects RS256 (asymmetric): signs with RSA private key, verifies with RSA public key.
- Attacker sends HS256 (symmetric) token, using the RSA **public key** as the HMAC secret.
- If the server derives the algorithm from the JWT header rather than enforcing an allowlist, it accepts the forged token.

**Root cause:** JWT libraries often select the verification method based on the `alg` field in the token header. If the application does not explicitly restrict which algorithms are acceptable, the library follows whatever the attacker sends.

**Historical CVEs:**
- **CVE-2015-9235** — `jsonwebtoken` (Node.js) accepted `alg: "none"`, skipping signature verification entirely.
- **CVE-2017-17426** — `jose4j` (Java) was susceptible to RS256→HS256 confusion.
- **CVE-2022-21449** — Java ECDSA "Psychic Signatures" (r=0, s=0 accepted as valid).
- **CVE-2025-30204** — `golang-jwt` `ParseUnverified` issue (DoS via untrusted input splitting).

Multiple libraries across ecosystems have been affected. The vulnerability class persists because the JWT specification (RFC 7519) allows arbitrary algorithms in the header by design.

---

## 2. The Attack Explained

### Setup

```
Authorization Server (AS)                Resource Server (RS)
  ├─ RSA private key (secret)             ├─ RSA public key
  ├─ Signs tokens with RS256              ├─ Verifies tokens with RS256
  └─ Publishes public key via JWKS        └─ Fetches key from JWKS endpoint
```

The RSA public key is **public** — anyone can fetch it from the `/.well-known/jwks.json` endpoint. This is by design; RSA security depends on the private key remaining secret.

### Attack Steps

1. **Attacker fetches the public key** from the JWKS endpoint (it's public).
2. **Attacker crafts a malicious JWT header:** `{"alg": "HS256", "typ": "JWT", "kid": "rsa-key-1"}`
3. **Attacker signs the JWT** using HMAC-SHA256, with the RSA public key (PEM bytes) as the HMAC secret.
4. **Vulnerable server** reads `alg: "HS256"` from the header → switches to HMAC verification.
5. Server retrieves the RSA public key associated with `kid: "rsa-key-1"` and uses it as the HMAC secret.
6. **HMAC verification succeeds** — the attacker computed the same HMAC using the same public key.

### Why It Works

```
Attacker computes:  HMAC-SHA256(header.payload, publicKeyPEM)
Server computes:    HMAC-SHA256(header.payload, publicKeyPEM)  // same key!
→ Signatures match → token accepted
```

- HMAC is deterministic and symmetric: anyone with the secret can both sign and verify.
- The RSA public key IS the HMAC secret in this attack — and the attacker has it.
- The server's key lookup returns an `rsa.PublicKey`, which a vulnerable library may silently convert to bytes for HMAC use.

### Payload

The attacker can set any claims:
```json
{
  "sub": "admin",
  "role": "superuser",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "exp": 9999999999
}
```

This grants full administrative access with arbitrary identity, tenant, and expiration claims.

---

## 3. Variant: None Algorithm

### The Attack

1. Attacker creates a JWT with header: `{"alg": "none", "typ": "JWT"}`
2. No signature is needed (empty signature segment).
3. Vulnerable server: `alg: "none"` → skips signature verification entirely.
4. Attacker's forged JWT is accepted as valid.

```
Header:   eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0           (base64url of {"alg":"none",...})
Payload:  eyJzdWIiOiJhZG1pbiIsInJvbGUiOiJzdXBlcnVzZXIifQ  ({"sub":"admin","role":"superuser"})
Signature: (empty)
Token:    header.payload.
```

### Historical Context

- **CVE-2015-9235**: The `jsonwebtoken` Node.js library accepted `alg: "none"` when no `algorithms` parameter was passed to `jwt.verify()`.
- Many libraries have since fixed this, but misconfiguration or outdated versions can still enable it.
- Best practice: libraries should reject `"none"` by default in production deployments.

### Python Example

```python
# VULNERABLE — no algorithm specified, accepts "none"
payload = jwt.decode(token, verify=False)

# SECURE — explicit algorithm allowlist
payload = jwt.decode(token, key=public_key, algorithms=["RS256"])
```

---

## 4. Preventive Measures

### 1. Algorithm Whitelist (CRITICAL)

The server must specify which algorithms are acceptable. Never derive the algorithm from the JWT header.

```go
// VULNERABLE — uses whatever alg the header says
token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
    return publicKey, nil // No method check!
})

// SECURE — explicit method enforcement in keyFunc
token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return publicKey, nil
})

// BEST — also use WithValidMethods parser option (belt + suspenders)
token, err := jwt.Parse(tokenString, keyFunc,
    jwt.WithValidMethods([]string{"RS256"}))
```

### 2. Key Type Enforcement

| Key Type | Valid Algorithms | Invalid Algorithms |
|----------|-----------------|-------------------|
| RSA public key | RS256, RS384, RS512 | HS256, HS384, HS512 |
| ECDSA public key | ES256, ES384, ES512 | HS*, RS* |
| HMAC secret | HS256, HS384, HS512 | RS*, ES* |

Never use an asymmetric public key as an HMAC secret. Enforce this at the type level.

### 3. kid Separation

Each key has a `kid` that should be algorithm-specific. JWKS entries include an `alg` field:

```json
{
  "kid": "rsa-key-1",
  "alg": "RS256",
  "kty": "RSA",
  "use": "sig",
  "n": "...",
  "e": "AQAB"
}
```

Server should verify that the key type associated with a `kid` matches the algorithm declared in the token header (or better, the algorithm the server expects).

### 4. Reject "none"

Always reject `alg: "none"` in production. There is no legitimate use case for unsigned access tokens.

### 5. Library Usage

- **Always** pass the `algorithms` parameter or use `WithValidMethods`.
- `golang-jwt` v5: check `t.Method` type in `keyFunc` AND use `jwt.WithValidMethods()`.
- Never call `jwt.ParseUnverified()` for security-sensitive operations.
- Keep libraries updated to patch known CVEs.

---

## 5. GGID JWT Validation Audit

### Library

GGID uses **`github.com/golang-jwt/jwt/v5`** — the current maintained version of the Go JWT library.

### Code Review

From `services/gateway/internal/middleware/middleware.go` (lines 519–540):

```go
tokenStr := strings.TrimSpace(parts[1])
parseOpts := []jwt.ParserOption{
    jwt.WithValidMethods([]string{"RS256"}),  // ← Algorithm whitelist enforced
}
if issuer != "" {
    parseOpts = append(parseOpts, jwt.WithIssuer(issuer))
}
if audience != "" {
    parseOpts = append(parseOpts, jwt.WithAudience(audience))
}
token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
    if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {  // ← Type check
        return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
    }
    keyID, _ := token.Header["kid"].(string)
    if keyID == "" {
        keyID = jwks.KeyID()
    }
    return jwks.GetKey(keyID)
}, parseOpts...)
```

**Key observations:**
- **Algorithm whitelist**: `jwt.WithValidMethods([]string{"RS256"})` — only RS256 accepted.
- **Type assertion**: `token.Method.(*jwt.SigningMethodRSA)` — rejects non-RSA methods.
- **Belt and suspenders**: Both the parser option AND the keyFunc type check enforce RSA-only.
- **Key lookup**: `jwks.GetKey(keyID)` returns `*rsa.PublicKey` — Go's type system prevents it from being used as an HMAC `[]byte` secret.
- **JWKS fetch**: Only fetches from configured `jwksURL` — does NOT follow `jku` or `x5u` header values.

### JWKS Client

From `middleware.go` (lines 339–430): `JWKSClient` only fetches keys from the configured `jwksURL` set at initialization. It does not read `jku`, `x5u`, or `x5c` from JWT headers. Key refresh uses the same trusted URL.

### Audit Table

| Check | GGID Status | Vulnerable? | Fix Needed? |
|-------|-------------|-------------|-------------|
| Algorithm whitelist enforced | `WithValidMethods(["RS256"])` + type assertion | **No** | None |
| "none" algorithm rejected | Not in valid methods list | **No** | None |
| RS256→HS256 confusion | Type assertion blocks non-RSA; `GetKey` returns `*rsa.PublicKey` | **No** | None |
| Key type matches algorithm | JWKS only accepts `kty: "RSA"`, `use: "sig"` | **No** | None |
| kid consistency | kid used to select key from RSA-only map | **No** | None |
| jku/x5u header injection | Server ignores JWT header URLs; uses configured JWKS | **No** | None |
| Constant-time comparison | `golang-jwt` v5 uses `subtle.ConstantTimeCompare` | **No** | None |
| Library version | `golang-jwt/jwt/v5` (current) | **No** | Keep updated |

### Test Coverage

`jwt_validation_test.go` includes `TestJWT_WrongSigningMethod` (line 330) which verifies that an HS256-signed token is rejected with HTTP 401. This test directly validates protection against the RS256→HS256 confusion attack.

### Assessment

**GGID is NOT vulnerable to JWT algorithm confusion attacks.** The implementation uses all three defense layers:
1. Parser-level: `WithValidMethods(["RS256"])`
2. KeyFunc-level: `*jwt.SigningMethodRSA` type assertion
3. Type-system-level: `GetKey` returns `*rsa.PublicKey`, preventing HMAC misuse

---

## 6. Additional JWT Vulnerabilities

### JKU/X5U Header Injection

- **jku** (JWK Set URL): JWT header field pointing to a JWKS endpoint. If the server fetches keys from this URL, an attacker can point it to their own JWKS with attacker-controlled keys.
- **x5u/x5c** (X.509 URL/certificate chain): Same attack vector — server fetches attacker's certificate.

**Defense:** Server must NEVER fetch keys from JWT header values. Use only pre-configured, trusted JWKS endpoints.

**GGID status:** Safe. `JWKSClient` only fetches from the `jwksURL` set during initialization. JWT header `jku`/`x5u` fields are never read.

### Key Confusion (ECDSA)

- Using an ES256 key (P-256 curve) with ES384 (P-384) or vice versa can lead to cross-curve signature forgery.
- **Defense:** Enforce specific curve parameters per algorithm.
- **GGID status:** N/A — GGID uses RS256 only, no ECDSA support in the gateway.

### Timing Attacks

- Signature comparison must use constant-time comparison to prevent timing side-channels.
- `golang-jwt` v5 uses `crypto/subtle.ConstantTimeCompare` internally.
- **GGID status:** Safe — inherits constant-time comparison from the library.

---

## 7. Testing for Vulnerability

### Manual Test

```bash
# 1. Fetch JWKS public key
curl https://gateway.example.com/.well-known/jwks.json

# 2. Craft HS256 token using public key as HMAC secret (jwt_tool)
python3 jwt_tool.py \
  -t "https://gateway.example.com/api/v1/users" \
  -rh "Authorization: Bearer <token>" \
  -X alg_confusion

# 3. Craft "none" algorithm token
python3 jwt_tool.py \
  -t "https://gateway.example.com/api/v1/users" \
  -rh "Authorization: Bearer <token>" \
  -X alg_none
```

If the server returns 200 OK with either forged token → **VULNERABLE**.

### Automated Tooling

| Tool | Purpose |
|------|---------|
| [jwt_tool](https://github.com/ticarpi/jwt_tool) | Automated JWT analysis, algorithm confusion, none bypass |
| Burp Suite JWT Plugin | Intercept and modify JWTs in-flight |
| [jwt-cracker](https://github.com/brendan-rius/c-jwt-cracker) | Brute-force HMAC secrets |
| Custom Go test | Programmatic algorithm confusion test (see below) |

### Go Test Example

```go
func TestAlgorithmConfusion_Rejected(t *testing.T) {
    // Create HS256 token with RSA public key as HMAC secret
    pubKeyPEM := pem.EncodeToMemory(&pem.Block{
        Type:  "PUBLIC KEY",
        Bytes: x509.MarshalPKCS1PublicKey(rsaPub),
    })
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    token.Header["kid"] = "rsa-key-1"
    forgedToken, _ := token.SignedString(pubKeyPEM)

    req := httptest.NewRequest("GET", "/api/v1/test", nil)
    req.Header.Set("Authorization", "Bearer "+forgedToken)

    // Assert: server returns 401, handler NOT called
}
```

GGID already has this test: `TestJWT_WrongSigningMethod` in `jwt_validation_test.go`.

---

## 8. Remediation Roadmap

| Phase | Task | Priority | Effort |
|-------|------|----------|--------|
| 1 | Audit JWT validation code for algorithm whitelist | P0 | ~0.5 day |
| 2 | Add explicit algorithm enforcement (`WithValidMethods`) if missing | P0 | ~0.5 day |
| 3 | Verify `"none"` algorithm rejection | P0 | ~0.5 day |
| 4 | Reject `jku`/`x5u` header values (if fetching from headers) | P1 | ~0.5 day |
| 5 | Add automated test for algorithm confusion attack | P1 | ~1 day |

**GGID status: All phases already implemented.** The codebase enforces RS256-only via both `WithValidMethods` and type assertions, ignores JWT header URLs, and has test coverage for wrong-algorithm rejection.

**Estimated effort for new deployments:** 2–3 days to audit and harden.
**Estimated effort for GGID:** 0 — already secure.

---

## References

- [JWT Algorithm Confusion Attack — Sourcery](https://www.sourcery.ai/vulnerabilities/jwt-algorithm-confusion)
- [JWT Algorithm Confusion (RS256 → HS256) — JWTForge](https://jwtforge.com/guides/jwt-algorithm-confusion)
- [CVE-2015-9235 — jsonwebtoken "none" algorithm](https://nvd.nist.gov/vuln/detail/CVE-2015-9235)
- [CVE-2025-30204 — golang-jwt ParseUnverified](https://nvd.nist.gov/vuln/detail/CVE-2025-30204)
- [RFC 7519 — JSON Web Token (JWT)](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 8725 — JSON Web Token Best Current Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [golang-jwt/jwt/v5 — GitHub](https://github.com/golang-jwt/jwt)
