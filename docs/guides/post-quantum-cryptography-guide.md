# Post-Quantum Cryptography Guide

This guide covers NIST PQC standards, migration timeline, hybrid approach, impact on TLS/JWT/signing, crypto agility, and GGID's PQC migration plan.

## NIST PQC Standards

| Algorithm | Type | NIST Status | Use Case |
|---|---|---|---|
| CRYSTALS-Kyber (ML-KEM) | Key encapsulation | FIPS 203 (finalized) | Key exchange |
| CRYSTALS-Dilithium (ML-DSA) | Digital signature | FIPS 204 (finalized) | Signatures |
| SPHINCS+ (SLH-DSA) | Hash-based signature | FIPS 205 (finalized) | Signatures (fallback) |
| FALCON | Lattice signature | Pending | Signatures (compact) |

## Harvest-Now-Decrypt-Later Threat

Attackers collect encrypted traffic today to decrypt when quantum computers become available.

| Data Sensitivity | Quantum Risk | Action |
|---|---|---|
| Long-term secrets (10+ years) | High | Migrate first |
| PII (retained 7 years) | Medium | Migrate by 2030 |
| Short-lived tokens (15 min) | Low | Migrate last |
| Audit logs (retained 3 years) | Medium | Migrate by 2030 |

## Migration Timeline

| Phase | Timeline | Action |
|---|---|---|
| Assessment | 2026 | Inventory crypto usage, assess risk |
| Hybrid deployment | 2027-2028 | Deploy classical + PQC hybrid |
| PQC-native | 2029-2030 | PQC-only where supported |
| Full migration | 2032+ | Remove classical-only crypto |

## Hybrid Approach (Classical + PQC)

```
TLS handshake:
1. Classical key exchange (ECDHE) → shared secret A
2. PQC key exchange (Kyber) → shared secret B
3. Final key = HKDF(secret_A || secret_B)
```

Security: attacker must break BOTH classical AND PQC to compromise.

## Impact on GGID Components

| Component | Current | PQC Migration | Priority |
|---|---|---|---|
| TLS connections | ECDHE + RSA | Hybrid ECDHE+Kyber | High |
| JWT signing | RS256/ES256 | Dilithium + ES256 hybrid | Medium |
| SAML signing | RSA-SHA256 | Dilithium + RSA hybrid | Medium |
| gRPC mTLS | TLS 1.3 | PQC TLS hybrid | High |
| Database encryption | AES-256-GCM | AES-256-GCM (symmetric, PQC-safe) | Low |
| WebAuthn | ES256/RS256 | Dilithium support (when available) | Low |

## Crypto Agility Design

```yaml
crypto_agility:
  enabled: true
  algorithm_registry:
    jwt_signing:
      current: ["RS256", "ES256", "EdDSA"]
      pqc: ["ML-DSA-65", "ML-DSA-87"]
      hybrid: ["ES256+ML-DSA-65"]
    tls:
      current: ["X25519", "P-256"]
      pqc: ["X25519Kyber768"]
      hybrid: true
  key_rotation:
    algorithm_change_supported: true
    overlap_period: 90d
  jwks:
    publish_pqc_keys: true
    hybrid_keys: true
```

## GGID PQC Migration Plan

```yaml
pqc_migration:
  phase: "assessment"  # assessment → hybrid → native → complete
  inventory:
    - component: "tls"
      current: "TLS 1.3 ECDHE"
      target: "TLS 1.3 hybrid Kyber"
      priority: "high"
    - component: "jwt_signing"
      current: "RS256"
      target: "ES256+ML-DSA-65 hybrid"
      priority: "medium"
    - component: "grpc_mtls"
      current: "TLS 1.3"
      target: "PQC TLS hybrid"
      priority: "high"
    - component: "database"
      current: "AES-256-GCM"
      target: "AES-256-GCM (PQC-safe)"
      priority: "low"
  timeline:
    assessment_complete: "2026-12"
    hybrid_deployment: "2027-06"
    pqc_native: "2029-01"
    full_migration: "2032-01"
```

## Best Practices

1. **Start with assessment** — Know where crypto is used
2. **Deploy hybrid first** — Classical + PQC during transition
3. **Design for agility** — Algorithms should be configurable, not hardcoded
4. **Prioritize long-lived secrets** — These are most at risk from harvest-now
5. **Monitor NIST standards** — FIPS 203/204/205 finalized, more coming
6. **Test interoperability** — Not all clients support PQC yet
7. **Plan key migration** — PQC keys are larger, plan storage
8. **Update JWKS** — Publish PQC public keys alongside classical
9. **Don't rush to PQC-only** — Hybrid is safer during transition
10. **Document algorithm choices** — Track why each algorithm was chosen