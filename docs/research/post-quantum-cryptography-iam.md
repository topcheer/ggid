# Post-Quantum Cryptography (PQC) and Its Impact on IAM

> Research document examining how quantum computing threatens current cryptographic
> primitives in GGID and the NIST-standardized PQC algorithms that will replace them.

---

## 1. Overview

Quantum computing threatens the foundation of public-key cryptography used across
modern IAM systems. **Shor's algorithm** can factor integers and compute discrete
logarithms in polynomial time given a sufficiently large quantum computer, breaking
RSA and ECDSA — the two algorithms underlying virtually all JWT signing, TLS
certificates, and OAuth client authentication today.

Current estimates place a cryptographically relevant quantum computer (CRQC)
capable of breaking RSA-2048 in the **2030-2040** timeframe. NSA, ANSSI, and BSI
all recommend beginning migration preparations now.

The most urgent driver is not future decryption of future data, but the
**"harvest now, decrypt later"** threat: adversaries are already recording
encrypted TLS traffic and storing JWTs signed with RSA/ECDSA keys today. When a
quantum computer arrives, they retroactively forge tokens and decrypt sessions.

NIST completed its Post-Quantum Cryptography (PQC) standardization process in
August 2024, publishing three Federal Information Processing Standards (FIPS).

For GGID, the affected components are:
- **JWT signing**: RS256 (RSA-2048) and ES256 (ECDSA P-256) in auth/oauth services
- **TLS certificates**: service-to-service mTLS and external HTTPS
- **JWKS endpoints**: RSA/ECDSA public keys are exposed and forgeable post-quantum
- **OAuth client auth**: `private_key_jwt` uses RSA/ECDSA signing

Symmetric primitives in GGID (Argon2id passwords, AES-256-GCM encryption,
HMAC-SHA256 client secrets, random session tokens) are **not** at risk.

---

## 2. NIST PQC Standardization Results

In August 2024, the U.S. Secretary of Commerce approved three FIPS standards:
**FIPS 203**, **FIPS 204**, and **FIPS 205**.

### Key Encapsulation Mechanisms (KEM) — for key exchange

| Standard | Algorithm (new name) | Former name | FIPS | Status |
|----------|---------------------|-------------|------|--------|
| FIPS 203 | **ML-KEM** | Kyber | Published Aug 2024 | Final |

- **Use case**: TLS key exchange, replacing ECDHE/X25519
- **Parameters**: ML-KEM-512, ML-KEM-768, ML-KEM-1024 (security levels 1, 3, 5)
- Already deployed in hybrid form by Chrome 116+ and Firefox 127+

### Digital Signatures — for JWT signing, certificates

| Standard | Algorithm (new name) | Former name | FIPS | Status |
|----------|---------------------|-------------|------|--------|
| FIPS 204 | **ML-DSA** | Dilithium (CRYSTALS) | Published Aug 2024 | Final |
| FIPS 205 | **SLH-DSA** | SPHINCS+ | Published Aug 2024 | Final |
| TBD | **FN-DSA** | Falcon | Expected 2025 | Round 4 |

- **ML-DSA**: balanced performance/size, lattice-based, recommended primary choice
- **SLH-DSA**: conservative hash-based fallback (if lattices are broken), larger signatures
- **FN-DSA**: fastest verification, compact signatures, more complex to implement

### Signature Size Comparison

| Algorithm | Public Key | Signature | Equiv. Security | Notes |
|-----------|-----------|-----------|-----------------|-------|
| RSA-2048 | 256 B | 256 B | ~112-bit | Current GGID default |
| ECDSA P-256 | 32 B | 64 B | ~128-bit | Current GGID option |
| ML-DSA-44 | 1312 B | 2420 B | 128-bit | NIST level 2 |
| ML-DSA-65 | 1952 B | 3309 B | 192-bit | NIST level 3 (recommended) |
| ML-DSA-87 | 2592 B | 4595 B | 256-bit | NIST level 5 |
| SLH-DSA-SHA2-128s | 32 B | 7856 B | 128-bit | Conservative choice |
| FN-DSA-512 | 897 B | ~666 B | 128-bit | Compact sig, complex impl |

**Key takeaway**: PQC signatures are **10-50x larger** than current RSA/ECDSA.

---

## 3. JWT Algorithm Migration

### Current State in GGID

GGID's JWT infrastructure (in auth and oauth services) currently uses:
- **Signing**: RS256 (RSA-2048) as default, ES256 (ECDSA P-256) as alternative
- **Verification**: JWKS endpoint serving RSA/ECDSA public keys
- **OAuth client auth**: `client_secret` (HMAC-SHA256) or `private_key_jwt` (RS256)

The `pkg/crypto` package itself handles symmetric operations (Argon2id, AES-256-GCM,
random tokens) — these are quantum-safe and require no migration. The JWT layer
lives in the auth/oauth services and uses `golang-jwt`.

### Proposed PQC JWT Algorithms

```
"ML-DSA-65"         — pure PQC, Dilithium security level 3
"SLH-DSA-128s"      — pure PQC, SPHINCS+ for conservative deployments
"RS256+ML-DSA-65"   — hybrid: sign with both, verify both (transition)
```

### Impact on JWT

| Metric | RS256 JWT | ML-DSA-65 JWT | Impact |
|--------|----------|---------------|--------|
| Token size | ~800 bytes | ~5,000+ bytes | 6x larger |
| JWKS key size | 256 bytes/key | 1,952 bytes/key | 8x larger |
| Authorization header | 1 KB | 7 KB | Exceeds some proxy limits |
| Cookie storage | Fits 4 KB | Exceeds 4 KB | Breaks cookie-based sessions |

Practical implications:
- HTTP proxies with 8 KB header limits will reject PQC JWTs in Authorization headers
- Browser cookie-based JWT sessions become infeasible (>4 KB)
- Database column types storing JWTs may need TEXT instead of VARCHAR

### Go Implementation Sketch

```go
// Using github.com/cloudflare/circl/sign/ml_dsa (or equivalent FIPS-compliant lib)
import "github.com/cloudflare/circl/sign/ml_dsa"

// Key generation
pub, priv, _ := ml_dsa.GenerateKey(ml_dsa.ParametersID65, rand.Reader)

// Signing (replaces rsa.SignPSS or ecdsa.Sign)
sig := ml_dsa.Sign(priv, messageBytes) // returns 3309-byte signature

// Verification
valid := ml_dsa.Verify(pub, messageBytes, sig)

// Register with golang-jwt as custom algorithm
jwt.RegisterSigningMethod("ML-DSA-65", &PQCSigner{
    key: priv,
    verifier: ml_dsa.Verify,
})
```

### Hybrid Signer (transition strategy)

```go
// Sign with RS256 AND ML-DSA-65, embed both signatures
sigRSA := rsa.SignPSS(rand.Reader, rsaKey, crypto.SHA256, hash, nil)
sigPQ  := ml_dsa.Sign(pqKey, messageBytes)

// Signature format: base64url(sigRSA) + "." + base64url(sigPQ)
hybridSig := base64.RawURLEncoding.EncodeToString(sigRSA) + "." +
             base64.RawURLEncoding.EncodeToString(sigPQ)
```

---

## 4. TLS Certificate Migration

### Current TLS in GGID

- **Service-to-service**: mTLS with RSA/ECDSA certs (internal PKI)
- **External TLS**: HTTPS terminated at Gateway with RSA certs (Let's Encrypt / CA)
- **gRPC**: TLS between gateway and microservices

### PQC TLS (Hybrid Approach)

- X.509 certificates with PQC public keys: defined in `draft-ietf-pquip-x509-pkix-pq`
- Hybrid TLS handshake: traditional (X25519) + PQC (ML-KEM-768) in single key exchange
- Security: attacker must break BOTH algorithms to compromise the session
- Browser support: Chrome 116+ (X25519MLKEM768), Firefox 127+, Cloudflare edge

### Certificate Authority Timeline

| Provider | PQC Status | Timeline |
|----------|-----------|----------|
| DigiCert | Pilot PQC certificates | 2024-2025 |
| Sectigo | Pilot program | 2024-2025 |
| Let's Encrypt | No announced timeline | TBD |
| OpenSSL 3.4+ | PQC key generation | Available now |
| Go crypto/tls | Hybrid X25519MLKEM768 | Go 1.23+ |

### GGID TLS Migration Phases

- **Phase 1** (2025-2026): Hybrid key exchange (X25519 + ML-KEM) — no cert changes
- **Phase 2** (2027-2028): PQC certificates for internal service-to-service mTLS
- **Phase 3** (2029+): PQC certificates for external HTTPS (when CAs support it)

---

## 5. Hybrid Migration Strategy

### Why Hybrid First

PQC algorithms are mathematically new and may harbor undiscovered vulnerabilities.
Recent examples (e.g., the 2022 SIDH attack breaking SIKE) demonstrate the risk.
Traditional algorithms (RSA, ECDSA) are battle-tested over decades. **Hybrid mode**
guarantees security if EITHER algorithm remains unbroken.

### JWT Hybrid Signing Format

```
Header: {
  "alg": "RS256+ML-DSA-65",
  "typ": "JWT",
  "kid": "rsa-key-2025",
  "kid_pq": "pq-key-2025"
}
Signature: base64url(RS256_sig).base64url(ML-DSA-65_sig)
```

### Migration Path

| Phase | Issuer | Verifier (PQ-aware) | Verifier (legacy) |
|-------|--------|--------------------|--------------------|
| **A: Hybrid issue** | Signs RS256 + ML-DSA | Verifies both | Verifies RS256 only |
| **B: Require PQ** | Signs RS256 + ML-DSA | Rejects RS256-only | Must upgrade |
| **C: Pure PQC** | Signs ML-DSA only | Verifies ML-DSA | Incompatible |

### Key Management

- Generate PQC key pairs alongside existing RSA/ECDSA in the key vault
- JWKS endpoint serves both key types with `kty` field extension
- Rotate PQC keys independently (different cycle than RSA/ECDSA)
- Store PQC private keys using the existing AES-256-GCM encryption in `pkg/crypto`

### Timeline

| Period | Strategy |
|--------|----------|
| 2025-2027 | Hybrid deployment (PQC + traditional) |
| 2027-2030 | PQC preferred (traditional as fallback only) |
| 2030+ | Pure PQC (traditional algorithms deprecated/removed) |

---

## 6. Impact on OAuth/OIDC Flows

### Token Size

| Token Type | Current | PQC-Signed | Mitigation |
|-----------|---------|-----------|------------|
| Access token (JWT) | ~800 B | ~5 KB+ | Use opaque tokens |
| ID token | ~1 KB | ~6 KB+ | Front-channel: use fragment |
| Refresh token | ~200 B | ~5 KB+ | Store server-side, opaque |
| Authorization code | ~50 B | ~50 B | Not affected (random) |

**Recommendation**: GGID should support opaque access tokens (reference to
server-side session) as the default for PQC, with JWT introspection via
the token endpoint. This sidesteps the 4 KB cookie / 8 KB header problem.

### Client Authentication

- **`private_key_jwt`**: client signs assertion with PQC key (ML-DSA-65)
- **mTLS bound tokens (RFC 8705)**: client cert carries PQC public key
- **`client_secret` (HMAC)**: not affected — symmetric, quantum-safe

### JWKS Endpoint

| Key Set | Size | Notes |
|---------|------|-------|
| 2 RSA keys | ~1 KB | Current |
| 2 RSA + 2 ML-DSA | ~9 KB | Hybrid period |
| 2 ML-DSA only | ~8 KB | Pure PQC |

HTTP response size is not a concern (Gzip handles it), but cache TTL and
CDN payload limits should be reviewed.

---

## 7. Hash Function Migration

| Algorithm | Quantum Security | Status |
|-----------|-----------------|--------|
| SHA-256 | 128 -> 64 bits (Grover) | Sufficient for now |
| SHA-384 | 192 -> 96 bits | Recommended post-quantum |
| SHA-512 | 256 -> 128 bits | Recommended post-quantum |
| SHAKE256 | N/A (variable) | Used internally by ML-DSA/SLH-DSA |

**Grover's algorithm** halves the effective security of symmetric hashes, but
64-bit security (SHA-256 post-quantum) is still considered adequate for most
applications. For long-term archives, SHA-384 or SHA-512 is recommended.

GGID-specific:
- **Argon2id** (password hashing): not quantum-relevant — memory-hard, symmetric
- **AES-256-GCM** (`pkg/crypto`): quantum-safe (128-bit post-Grover)
- **HMAC-SHA256** (client secrets): quantum-safe
- **Random session tokens**: quantum-safe (based on CSPRNG)

**Recommendation**: No urgent hash migration needed. SHA-256 is sufficient.

---

## 8. GGID Current Crypto Audit

| Operation | Current Algorithm | PQ Vulnerable? | Priority | Target |
|-----------|------------------|----------------|----------|--------|
| JWT signing (auth) | RS256 (RSA-2048) | YES (Shor's) | P1 | 2027 |
| JWT signing (oauth) | RS256 / ES256 | YES (Shor's) | P1 | 2027 |
| JWKS public keys | RSA / ECDSA | YES (keys forgeable) | P1 | 2027 |
| TLS service mesh | RSA / ECDSA certs | YES | P2 | 2028 |
| TLS external | RSA (Let's Encrypt) | YES | P2 | 2029 |
| OAuth `private_key_jwt` | RS256 | YES | P1 | 2027 |
| Password hashing | Argon2id | NO | None | — |
| Token encryption | AES-256-GCM | NO | None | — |
| Session ID | CSPRNG random | NO | None | — |
| HMAC (client_secret) | HMAC-SHA256 | NO | None | — |

**Summary**: 6 out of 10 cryptographic operations are quantum-vulnerable.
All are in the asymmetric (public-key) layer; the symmetric layer is safe.

---

## 9. Roadmap

| Phase | Year | Activity | Priority |
|-------|------|----------|----------|
| 1 | 2025 | Monitor Go PQC library maturity (CIRCL, FIPS-certified libs) | P3 (Low) |
| 2 | 2026 | Add experimental PQC JWT algorithm support (not default) | P2 |
| 3 | 2027 | Hybrid JWT signing (RS256 + ML-DSA-65) in auth/oauth | P1 |
| 4 | 2027-2028 | Hybrid TLS for internal service-to-service mTLS | P1 |
| 5 | 2028-2029 | PQC preferred (traditional as fallback) | P1 |
| 6 | 2030+ | Pure PQC — remove RSA/ECDSA signing | P1 |

### Key Decisions Needed

1. **Opaque vs JWT access tokens**: PQC JWTs are 5+ KB. Switching to opaque tokens
   with server-side session lookup avoids header/cookie size problems.
2. **PQC library selection**: CIRCL (Cloudflare, well-audited) vs Go standard library
   `crypto/fips140` (expected in Go 1.25+).
3. **Key storage**: PQC private keys are 4 KB+ (vs RSA ~1.2 KB). Vault/HSM capacity
   planning needed.
4. **Hybrid token format**: Define `alg` naming convention and dual-signature encoding.

### Threat Timeline Assessment

- **2025**: No practical quantum threat. Priority is LOW (P3).
- **2027**: Priority rises to P1. Hybrid signing should be live.
- **2030**: "Harvest now, decrypt later" window closes. Pure PQC recommended.
- **2035**: RSA-2048 may be breakable. Must have completed migration.

The "harvest now, decrypt later" threat is a **10-year** concern. Data with
long-term confidentiality requirements (secrets, credentials stored in encrypted
form) should be re-encrypted with PQC algorithms sooner rather than later.

### References

- NIST FIPS 203: ML-KEM (formerly Kyber) — Aug 2024
- NIST FIPS 204: ML-DSA (formerly Dilithium) — Aug 2024
- NIST FIPS 205: SLH-DSA (formerly SPHINCS+) — Aug 2024
- NIST PQC Standardization: https://csrc.nist.gov/projects/post-quantum-cryptography
- Cloudflare CIRCL library: https://github.com/cloudflare/circl
- IETF draft-ietf-pquip-x509-pkix-pq: PQC in X.509 certificates
- IETF draft-ietf-tls-hybrid-design: Hybrid key exchange in TLS 1.3
