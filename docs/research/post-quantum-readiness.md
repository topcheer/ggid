# Post-Quantum Readiness

> NIST PQC algorithms and GGID's migration path for JWT, TLS, and WebAuthn.

---

## NIST PQC Standards (2024-2025)

| Algorithm | Purpose | Standard |
|-----------|---------|----------|
| ML-KEM (Kyber) | Key encapsulation | FIPS 203 |
| ML-DSA (Dilithium) | Digital signatures | FIPS 204 |
| SLH-DSA (SPHINCS+) | Hash-based signatures | FIPS 205 |

### Timeline
- **2024**: NIST published final standards
- **2025**: Libraries available (BoringSSL, liboqs)
- **2026-2028**: Early adoption by TLS/VPN vendors
- **2030+**: NSA requires PQC for national security systems

---

## GGID Components at Risk

### 1. JWT Signing (RS256)

**Risk**: Quantum computer could factor RSA keys (Shor's algorithm)

**Migration path**:
1. Short term: RS256 → ES256 (ECDSA — also quantum-vulnerable but smaller keys)
2. Medium term: ES256 → ML-DSA (when Go crypto/x/crypto supports it)
3. Long term: Hybrid RS256 + ML-DSA during transition

```go
// Future: JWT with PQC signature
// token := jwt.NewWithClaims(jwt.SigningMethodMLDSA, claims)
```

### 2. TLS Connections

**Risk**: Harvest-now-decrypt-later attacks on TLS traffic

**Migration path**:
1. Go 1.24+ supports X25519Kyber768 hybrid KEM in TLS
2. Set `tls.Config.CurvePreferences` to include hybrid KEM

```go
tlsConfig := &tls.Config{
    CurvePreferences: []tls.CurveID{
        tls.X25519Kyber768, // Post-quantum hybrid
        tls.X25519,          // Classical fallback
    },
}
```

### 3. WebAuthn / FIDO2

**Risk**: FIDO2 uses ECDSA for authenticator signatures

**Migration path**: FIDO Alliance has PQC working group. WebAuthn Level 3 may add PQC algorithms. GGID should monitor and adopt when available.

---

## Readiness Assessment

| Component | Current | PQC Ready | Priority |
|-----------|---------|-----------|----------|
| JWT signing | RS256 | No | Medium (2030+)
| TLS | TLS 1.3 | Partial (Go 1.24 hybrid) | Low |
| WebAuthn | ECDSA | No (waiting on FIDO) | Low |
| Database | PostgreSQL | N/A | N/A |
| Password hashing | Argon2id | Quantum-resistant (already) | Done |

---

## Recommendation

1. **Monitor Go PQC support** — crypto/mlkem package expected in Go 1.25+
2. **Enable TLS hybrid KEM** when available in Go stdlib
3. **Plan JWT algorithm migration** — add ML-DSA as alternative signer
4. **No urgent action needed** — quantum threat is 10-15 years out for RSA-2048

Priority: P3 (monitoring, no implementation needed yet).

---

*See: [Security Overview](../architecture/security-overview.md) | [JWT Algorithm Confusion](jwt-algorithm-confusion.md)*

*Last updated: 2025-07-11*
