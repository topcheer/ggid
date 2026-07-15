# IAM Trends 2026 — Research Summary

## Top 3 Structural Shifts

### 1. Passkey Mainstream Adoption
- Passkeys (WebAuthn/FIDO2) becoming default authentication method
- Cross-device sync via Apple/Google/Microsoft password managers
- **GGID Status**: WebAuthn implemented with 6 attestation formats, passkey autofill (conditional mediation) endpoint exists
- **Gap**: No passkey health dashboard to track enrollment rates and recovery risk

### 2. AI Agent Identity Governance
- Agentic AI systems need first-class identity: registration, token exchange, delegation chains
- MCP (Model Context Protocol) auth becoming standard
- **GGID Status**: AI Agent Identity fully implemented (RegisterAgent, ExchangeAgentToken, delegation chains, scope enforcement)
- **Gap**: Agent lifecycle persistence (in-memory registry needs DB-backed store)

### 3. Post-Quantum Cryptography (PQC) Migration
- NIST finalized ML-DSA (signatures) and ML-KEM (key exchange) standards
- IAM vendors must inventory crypto dependencies (JWT signing, SAML assertions, TLS)
- Hybrid TLS (classical + PQC) recommended during migration window
- **GGID Status**: RSA/ECDSA only in JWT signing; no PQC support
- **Gap**: Need hybrid JWT signing (RSA + ML-DSA) in pkg/crypto

## Additional Trends

### Edge Computing IAM
- Identity verification moving to edge (Cloudflare Workers, Vercel Edge)
- Lightweight JWT verification at CDN edge nodes
- **GGID Status**: Go SDK has JWT middleware; could be compiled to WASM for edge

### NIS2/CRA Compliance (EU)
- Mandatory security incident reporting (NIS2)
- Cyber Resilience Act (CRA) requires SBOM and vulnerability disclosure
- **GGID Status**: Research complete (docs/research/nis2-cra-pipl-compliance.md); compliance dashboard page exists

## Backlog Items Generated

| Priority | Item | Owner | Driver |
|----------|------|-------|--------|
| P2 | PQC hybrid JWT signing (ML-DSA) | arch | NIST PQC finalization |
| P2 | Passkey health dashboard | frontend | Passkey adoption tracking |
| P2 | Agent registry DB persistence | backend | AI agent lifecycle |
| P3 | Edge-compatible JWT verification (WASM) | arch | Edge computing trend |
