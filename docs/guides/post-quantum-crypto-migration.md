# Post-Quantum Cryptography Migration

NIST PQC standards, Kyber/Dilithium, hybrid classical+PQC, migration timeline, impact on JWT/TLS, and testing.

## Overview

Quantum computers will break RSA and ECDSA. NIST has standardized post-quantum cryptographic algorithms. This guide covers the migration path for GGID.

## NIST PQC Standards (2024)

| Algorithm | Type | NIST Standard | Use Case |
|-----------|------|--------------|----------|
| ML-KEM (Kyber) | Key encapsulation | FIPS 203 | TLS key exchange |
| ML-DSA (Dilithium) | Digital signature | FIPS 204 | JWT signing, certificates |
| SLH-DSA (SPHINCS+) | Digital signature | FIPS 205 | Hash-based fallback |
| FN-DSA (Falcon) | Digital signature | FIPS 206 | Compact signatures |

## Migration Timeline

| Phase | Year | Action |
|-------|------|--------|
| Phase 1: Inventory | 2025 | Audit all crypto usage (JWT, TLS, SAML, DB) |
| Phase 2: Hybrid | 2026 | Deploy hybrid classical+PQC in TLS |
| Phase 3: PQC JWT | 2027 | Support ML-DSA for JWT signing |
| Phase 4: PQC Default | 2028 | PQC algorithms as default |
| Phase 5: Classical Deprecation | 2030+ | Remove RSA/ECDSA support |

## Hybrid Approach (Recommended)

During transition, use both classical and PQC:

```
TLS: ECDHE + Kyber (hybrid key exchange)
JWT: RSA-SHA256 || ML-DSA (dual signatures)
Certificates: RSA + Dilithium (dual cert chain)
```

### Benefits

| Benefit | Detail |
|---------|--------|
| Quantum resistant | Even if one algorithm breaks, other holds |
| Backward compatible | Old clients use classical, new clients use PQC |
| Gradual migration | No flag day cutover |

## Impact on TLS

### Hybrid Key Exchange

```go
// Go (with x/crypto/pqcrypto - future)
conn, err := tls.Dial("tcp", "auth.ggid.dev:443", &tls.Config{
    // Negotiate hybrid: classical + PQC
    CurvePreferences: []tls.CurveID{
        tls.X25519,          // Classical (current)
        tls.Kyber512,        // PQC (future)
        tls.X25519Kyber768,  // Hybrid (transition)
    },
})
```

### TLS 1.3 Hybrid

```
Client Hello:
  key_share: X25519 + Kyber768 (both)

Server processes both:
  → Derives shared secret from ECDHE
  → Derives shared secret from Kyber KEM
  → Final secret = SHA256(ecdh_secret || kyber_secret)
```

## Impact on JWT

### Dual-Signed JWT (Transition)

```json
// Header
{
  "alg": "RS256",
  "kid": "rsa-key-1",
  "pq_alg": "ML-DSA-65",
  "pq_kid": "dilithium-key-1",
  "pq_sig": "base64url-pqc-signature"
}
```

### Verification

```go
func VerifyDualSignedJWT(token string) (Claims, error) {
    // 1. Verify classical signature (for backward compat)
    if claims, err := verifyRS256(token); err == nil {
        // 2. Also verify PQC signature (for quantum resistance)
        if pqSig := header["pq_sig"]; pqSig != "" {
            verifyMLDSA(token, pqSig)
        }
        return claims, nil
    }

    // 3. Try PQC-only (future)
    return verifyMLDSA(token)
}
```

### Key Sizes

| Algorithm | Public Key | Signature | Impact |
|-----------|-----------|-----------|--------|
| RSA-2048 | 256 bytes | 256 bytes | Current |
| ES256 | 64 bytes | 64 bytes | Current |
| ML-DSA-65 | 1,952 bytes | 3,309 bytes | ~12x larger |
| SLH-DSA-128s | 32 bytes | 7,856 bytes | ~30x larger |

JWT tokens will grow significantly with PQC signatures.

## Impact on SAML

SAML uses XML signatures. PQC migration requires:

```
1. New signature algorithm URIs:
   http://www.w3.org/2024/pqc/dilithium3

2. Larger certificate sizes in metadata
3. Dual-signed assertions during transition
4. XML Canonicalization must handle larger payloads
```

## Impact on Database

### pgcrypto Column Encryption

```sql
-- Current: AES-256-GCM (symmetric — not affected by quantum)
-- Symmetric crypto (AES) is less affected by quantum (Grover's = quadratic speedup)
-- AES-256 still provides 128-bit post-quantum security

-- No immediate migration needed for symmetric encryption
```

## Impact on mTLS

```yaml
mTLS_migration:
  phase_2_2026:
    server_cert: "RSA + Dilithium (dual)"
    client_cert: "RSA (backward compat)"

  phase_3_2027:
    server_cert: "Dilithium (PQC primary)"
    client_cert: "RSA + Dilithium (dual)"

  phase_4_2028:
    server_cert: "Dilithium only"
    client_cert: "Dilithium only"
```

## Testing

### PQC Test Vectors

```go
func TestMLDSAVerify(t *testing.T) {
    // Test with NIST PQC test vectors
    pub, priv := ml_dsa.GenerateKey()
    msg := []byte("test message")
    sig := ml_dsa.Sign(priv, msg)

    assert.True(t, ml_dsa.Verify(pub, msg, sig))
    assert.False(t, ml_dsa.Verify(pub, []byte("tampered"), sig))
}
```

### Hybrid Fallback Test

```go
func TestHybridFallback(t *testing.T) {
    // Client supports only classical
    token := signRS256(claims)
    _, err := verify(token)
    assert.NoError(t, err) // Classical still works

    // Client supports hybrid
    tokenDual := signDual(claims)
    _, err = verify(tokenDual)
    assert.NoError(t, err) // Dual verified
}
```

## Crypto Agility

Design GGID to swap algorithms without code changes:

```yaml
crypto_config:
  jwt_signing_alg: "RS256"          # Current
  jwt_pq_signing_alg: "ML-DSA-65"   # Future
  tls_min_version: "1.3"
  tls_kex: ["X25519Kyber768", "X25519"]  # Hybrid first, classical fallback
```

Algorithm identifiers in JWT headers and TLS extensions make swapping transparent.

## Monitoring

| Metric | Alert |
|--------|-------|
| PQC algorithm support (clients) | Track adoption |
| Token size increase | Monitor bandwidth impact |
| Verification latency | PQC sig verify may be slower |
| Cert expiry (dual certs) | Track both expiry dates |

## See Also

- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Token Binding Comparison](token-binding-comparison.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [SAML SP Implementation](saml-sp-implementation.md)
