# Post-Quantum Cryptography (PQC) — GGID Gap Analysis

*Research date: 2026-07-12*

## Overview

NIST has published three final PQC standards (August 2024):
- **FIPS 203 (ML-KEM)**: Module-Lattice-Based Key-Encapsulation Mechanism (formerly Kyber)
- **FIPS 204 (ML-DSA)**: Module-Lattice-Based Digital Signature Algorithm (formerly Dilithium)
- **FIPS 205 (SLH-DSA)**: Stateless Hash-Based Digital Signature Algorithm (formerly SPHINCS+)

NIST expects these three standards to "provide the foundation for most deployments of post-quantum cryptography" and urges adoption now.

## GGID Current State: COMPLETELY MISSING

- No PQC algorithm support in `pkg/crypto/`
- JWT signing only supports RSA, ECDSA, Ed25519 — no ML-DSA
- TLS connections use classical key exchange — no ML-KEM hybrid
- No PQC migration planning or crypto-agility framework

## Gap Analysis

### P1: ML-DSA for JWT Signing (Backend)
- Add ML-DSA (Dilithium) as a JWT signing algorithm option
- `pkg/crypto/` needs ML-DSA implementation via Go PQC library
- Token header `alg: ML-DSA-65` support
- Backward compatibility: support both classical + PQC during migration

### P1: Hybrid Key Exchange for TLS (Gateway)
- Enable hybrid classical+PQC TLS (X25519+ML-KEM) in Go 1.25
- Go 1.25 has experimental PQC TLS support via `crypto/tls`
- Gateway and inter-service gRPC should support hybrid mode

### P2: Crypto-Agility Framework (pkg/crypto)
- Centralized algorithm registry
- Runtime algorithm selection
- Migration tracking (which keys use which algorithms)
- Rollback capability

### P2: SLH-DSA for Long-Term Signatures (Audit)
- Hash-based signatures for long-term audit log integrity
- SLH-DSA is stateless and suitable for archival signatures
- Complement existing HMAC hash chain with PQC-safe signatures

## Competitive Landscape
- Auth0/Okta: Announced PQC roadmap, no production support yet
- AWS Cognito: No PQC support announced
- Keycloak: Community discussion, no implementation
- **Opportunity**: GGID can be first-mover among open-source IAM

## Recommended Libraries
- `cloudflare/circl`: Go PQC primitives (ML-KEM, ML-DSA)
- Go 1.25 built-in: `crypto/mlkem` (standard library PQC)

## Backlog Items Generated
- [ ] **P1** Backend: ML-DSA JWT signing in pkg/crypto (services/)
- [ ] **P1** Backend: Hybrid PQC TLS in gateway (services/gateway/)
- [ ] **P2** Backend: Crypto-agility registry in pkg/crypto
- [ ] **P2** Backend: SLH-DSA audit log signatures (services/audit/)
- [ ] **P2** Docs: PQC migration guide (docs/guides/)
