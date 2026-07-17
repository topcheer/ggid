# OAuth 2.1 Compliance, AI Agent Identity Hardening, and PQC Migration — GGID Gap Analysis

> Research covering OAuth 2.1 (draft-15), MCP Agent Auth (2026-07-28 spec), and NIST PQC FIPS 203/204/205 adoption status.

---

## 1. OAuth 2.1 Compliance (draft-ietf-oauth-v2-1-15)

**Status:** OAuth 2.1 is nearing finalization. Key requirements:
- **PKCE mandatory** for all authorization code flows (no exceptions)
- **Implicit grant removed** — must use authorization code + PKCE
- **ROPC (Resource Owner Password Credentials) removed** — password grant deprecated
- **Exact redirect URI matching** required (no wildcard)
- **DPoP or mTLS** for sender-constrained tokens (refresh token binding)
- **Refresh token rotation** recommended

**GGID Status:**

| Requirement | OAuth 2.1 | GGID |
|-------------|-----------|------|
| PKCE mandatory | Required for all flows | DONE — PKCE enforced |
| Implicit grant | Removed | AUDIT EXISTS — oauth21_audit_handler checks, but login page still has password input |
| ROPC removed | Deprecated | PARTIAL — grant_type handling exists, unclear if ROPC is still accepted |
| DPoP sender-constrained tokens | Required or mTLS | PARTIAL — dpop_pg.go has PG store + config handler, but no actual proof verification at token endpoint |
| Exact redirect URI | Required | DONE — redirect URI validation exists |
| Refresh token rotation | Recommended | DONE — rotation implemented |

**Gap:** DPoP proof verification exists as endpoint (/dpop/verify, /token/dpop-bind) but is NOT enforced at the token endpoint itself. Token issuance doesn't validate DPoP header. This is the critical gap for OAuth 2.1 compliance.

## 2. AI Agent Identity / MCP Authentication (2026-07-28 spec)

**Status:** MCP 2026-07-28 spec makes MCP servers formal OAuth 2.1 resource servers. Key requirements:
- **Agent registration** with scoped capabilities (per-action authorization)
- **MCP token brokering** — vault-based credential dispatch
- **Non-human identity (NHI)** lifecycle management — separate from human identities
- **Delegation chains** — agent acts on behalf of user with traceable delegation
- **24,008 secrets exposed** in MCP config files in 2025 — urgent security concern

**GGID Status:**

| Requirement | MCP 2026 Spec | GGID |
|-------------|---------------|------|
| Agent registration | Required | PARTIAL — AI Agent Identity section exists (server.go:1716), 14 references to agent handlers |
| MCP OAuth 2.1 resource server | Required | NOT VERIFIED — unclear if MCP service enforces OAuth 2.1 token validation |
| Per-action authorization | Required | NOT FOUND — no capability scoping or per-action approval |
| Delegation chains | Required | NOT FOUND — no on-behalf-of (OBO) or delegation chain tracking |
| NHI lifecycle management | Required | NOT FOUND — no separate machine identity lifecycle |
| Secret vault brokering | Required | NOT FOUND — no vault-based credential dispatch |

**Gap:** GGID has agent registration scaffolding but lacks the core MCP auth requirements: per-action authorization, delegation chains, and NHI lifecycle. This is a P1 gap for AI agent use cases.

## 3. Post-Quantum Cryptography Migration (FIPS 203/204/205)

**Status:** NIST finalized 3 PQC standards (Aug 2024):
- **FIPS 203: ML-KEM** (Module-Lattice-Based Key Encapsulation) — replaces ECDH/Diffie-Hellman
- **FIPS 204: ML-DSA** (Module-Lattice-Based Digital Signature) — replaces RSA/ECDSA
- **FIPS 205: SLH-DSA** (Stateless Hash-Based Digital Signature) — backup signature
- **HQC selected** (Mar 2025) for 4th round standardization
- Migration timeline: US federal agencies must complete by 2035, practical adoption starting 2025-2026

**GGID Status:**

| Requirement | NIST PQC | GGID |
|-------------|----------|------|
| ML-KEM key exchange | FIPS 203 | NOT FOUND — no Kyber/ML-KEM implementation |
| ML-DSA signatures | FIPS 204 | **FALSE POSITIVE** — pqc_signature_handler.go uses ed25519, NOT ML-DSA. "PQC" label is misleading |
| TLS PQC negotiation | Required | NOT FOUND — no hybrid PQC TLS |
| JWT PQC signing | Recommended | NOT FOUND — JWT uses RSA/ECDSA only |

**Critical Finding:** GGID's `pqc_signature_handler.go` is **NOT actually post-quantum**. It uses ed25519 (classical cryptography). The "PQC" naming is misleading and should be corrected or actual ML-DSA implemented.

---

## Summary: New Backlog Items

1. **[P1] DPoP Proof Enforcement at Token Endpoint** — OAuth 2.1 requires sender-constrained tokens. GGID has DPoP storage/config but doesn't validate DPoP proof header during token issuance.

2. **[P1] AI Agent Per-Action Authorization + Delegation Chains** — MCP 2026 spec requires capability scoping and traceable delegation. GGID has registration but no authorization layer.

3. **[P0] Fix Misleading PQC Label** — `pqc_signature_handler.go` uses ed25519 (classical), not ML-DSA (post-quantum). Either rename or implement actual FIPS 204.

4. **[P2] ML-KEM/ML-DSA Crypto Package** — Implement FIPS 203/204 algorithms for future-proof JWT signing and key exchange.

5. **[P2] ROPC Grant Deprecation** — OAuth 2.1 removes ROPC. Audit and deprecate password grant_type acceptance.
