# ML-DSA for JOSE — Post-Quantum JWT Signing

> **Status**: Research — standard finalizing, GGID has zero PQC signing capability.
> **Date**: 2026-07-17
> **Priority**: P2 (prepare now; mandatory for government/regulated sectors by 2027-2030)

---

## 1. Standard Status (July 2026)

**draft-ietf-cose-dilithium** ("ML-DSA for JOSE and COSE") has reached draft-10
and is in the final stages before RFC publication. It specifies:

- JOSE algorithm identifiers: `ML-DSA-44`, `ML-DSA-65`, `ML-DSA-87`
  (registered in the IANA JOSE Algorithms registry)
- JWK key type: `AKP` (Algorithm Key Pair) with `pub`/`priv` parameters
  — ML-DSA keys do NOT fit the existing `RSA`/`EC`/`OKP` kty values
- Deterministic (hedged) signing as the default mode

This sits on top of **FIPS 204 (ML-DSA, August 2024)** — the NIST-standardized
form of CRYSTALS-Dilithium.

| Parameter set | NIST level | Sig size | Public key | Use case |
|---------------|-----------|----------|------------|----------|
| ML-DSA-44 | 1 | 2,420 B | 1,312 B | General auth tokens |
| ML-DSA-65 | 3 | 3,309 B | 1,952 B | **Recommended default** for IAM |
| ML-DSA-87 | 5 | 4,627 B | 2,592 B | High-security / long-lived |

**Practical implication for IAM**: ML-DSA signatures are ~5-12× larger than
ECDSA P-256 (64 B) and ~2-3× larger than RSA-2048 (256 B→2,420 B+).
A typical GGID access JWT would grow from ~800 B to ~4-5 KB. Still fine for
HTTP headers (8 KB default limits) but matters for cookie-based apps.

## 2. Why This Matters for GGID

- **CNSA 2.0 timeline**: US NSA requires PQC for new national-security
  systems by 2027, full transition by 2030-2033.
- **EU (ETSI/BSI)**: Germany's BSI recommends hybrid (classical + PQC)
  modes during transition; France ANSSI similar.
- **China (GM/T)**: separate track — GGID already has SM2SM3 support
  (alg whitelist commit 7cea65ab). Chinese PQC standards (AIGIS etc.) are
  not internationally standardized yet.
- **Enterprise procurement**: RFPs increasingly ask "PQC roadmap?" — having
  a documented answer is becoming a competitive requirement (Auth0/Okta
  both published PQC positions in 2025).

## 3. Go Library Landscape

| Library | ML-DSA | ML-KEM | Notes |
|---------|--------|--------|-------|
| `crypto/mlkem` (Go stdlib, 1.24+) | No | Yes (ML-KEM-768/1024) | TLS 1.3 hybrid key exchange — **Go 1.25 GGID already gets this for free in TLS** |
| `crypto/mldsa` (Go stdlib) | **Not yet** — targeted for a future release | — | Watch golang/go issues |
| `cloudflare/circl` | Yes (`sign/dilithium`) | Yes | Production-grade, but pre-FIPS API differences; check FIPS 204 final vectors |
| `open-quantum-safe/liboqs-go` | Yes | Yes | CGo binding to liboqs; heaviest but most complete |
| `filippo.io` modules | mlkem only | Yes | No ML-DSA module yet |

**Recommendation**: when implementing, use `cloudflare/circl` (pure Go, no
CGo — consistent with GGID's `CGO_ENABLED=0` builds) or wait for stdlib
`crypto/mldsa`. Validate against NIST ACVP test vectors (FIPS 204 final,
not round-3 Dilithium).

## 4. GGID Integration Points

The good news: GGID's architecture is already prepared for algorithm agility:

1. **KeyProvider abstraction** (`pkg/crypto/key_provider.go`) — SM2 provider
   landed in commit 7cea65ab, proving the pattern. An ML-DSA provider would
   implement the same `KeyProvider` interface (`Sign`, `Public`, `Metadata`).
2. **Algorithm whitelist** (`pkg/crypto/alg_whitelist.go`) — add
   `ML-DSA-65` etc. to `supportedAlgs`.
3. **kid derivation** — just unified across services (commit a3e29625);
   ML-DSA keys get kids the same way.
4. **JWKS endpoint** — must emit `kty: "AKP"` JWKs. Verify SDKs can parse
   unknown kty values gracefully (most JWT libraries allow custom kty).
5. **Gateway verification** (`services/gateway/internal/middleware`) —
   add ML-DSA verification alongside RS256/SM2SM3.

### Suggested hybrid transition mode

Issue tokens signed with **both** classical and PQC signatures during
transition (JWS JSON serialization with multiple signatures, or simpler:
keep RS256 primary and offer ML-DSA as opt-in per tenant via system config —
the sysconfig store built this week is the natural toggle).

## 5. Gap Summary

| Item | Status |
|------|--------|
| PQC research docs | 3 exist (post-quantum-*.md, pqc-*.md) — none cover the JOSE-specific draft |
| TLS hybrid key exchange (ML-KEM) | Free via Go 1.25 stdlib (`crypto/mlkem`, X25519MLKEM768) — verify enabled |
| ML-DSA JWT signing | **NOT IMPLEMENTED** |
| ML-DSA JWKS (`kty: AKP`) | NOT IMPLEMENTED |
| SDK PQC verification | NOT IMPLEMENTED (all 8 SDKs) |
| Tenant-level PQC opt-in config | NOT IMPLEMENTED (sysconfig store ready) |

## 6. Recommended Backlog Items

1. **[P2][Backend]** Verify TLS hybrid key exchange is active
   (X25519MLKEM768) on gateway ingress; document in security guide.
   Effort: 0.5 day (verification + docs).
2. **[P2][Backend]** Spike: ML-DSA KeyProvider using cloudflare/circl —
   sign + verify round-trip test only, no integration. Effort: 1 day.
3. **[P3][Backend]** ML-DSA JWKS support (`kty: AKP` JWK marshaling) +
   alg whitelist extension. Effort: 0.5 day (after item 2).
4. **[P3][SDK]** Go SDK ML-DSA verification behind feature flag. Effort:
   0.5 day (after item 3).
5. **[P3][Docs]** PQC migration guide for tenants (timeline, hybrid mode,
   token size implications). Effort: 0.5 day.

Do NOT start implementation before the spike (item 2) validates the circl
FIPS 204 final-vector compatibility.

## 7. References

- draft-ietf-cose-dilithium-10 — ML-DSA for JOSE and COSE
- FIPS 204 — Module-Lattice-Based Digital Signature Standard
- NIST SP 800-227 (KEM guidance, 2025)
- CNSA 2.0 timeline (NSA, 2022/2025 update)
- Go crypto/mlkem documentation (Go 1.24+)
- Existing research: docs/research/post-quantum-readiness.md,
  docs/research/pqc-post-quantum-cryptography.md
