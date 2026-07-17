# Verifiable Credentials, NIST 800-63B-4, and WebAuthn Hybrid Transport — GGID Gap Analysis

> Research covering OpenID4VP/SD-JWT VC (EU Digital Identity Wallet), NIST SP 800-63B Rev 4 (Aug 2025), and WebAuthn hybrid/cross-device authentication.

---

## 1. Verifiable Credentials + OpenID4VP

**Industry Status (2025):** EU Digital Identity Wallet (EUDI) mandates OpenID4VP for credential presentation by 2026. OpenID4VP 1.0 is final. SD-JWT (Selective Disclosure JWT) is the preferred VC format. Key specs: OpenID4VP, OpenID4VCI (issuance), DCQL (Digital Credential Query Language).

**GGID Status:** **NOT IMPLEMENTED**
- RAR handler mentions "Issue Credential" in consent text (rar_handler.go:196) but no actual VC issuance/presentation logic
- No SD-JWT support, no credential wallet endpoints, no OpenID4VP verifier flow

**This is a significant market gap** — EU organizations will need OIDC providers that support verifiable credential issuance and verification by 2026.

## 2. NIST SP 800-63B Revision 4 (August 2025 Final)

**Key changes from Rev 3:**
- **Phishing-resistant authenticators prioritized** — FIDO2/WebAuthn at AAL2 and AAL3
- **Password requirements relaxed** — no mandatory special characters, no forced rotation
- **AAL2 allows passwordless** — WebAuthn alone satisfies AAL2 (no password required)
- **Federation assurance** — new Federation Assurance Level (FAL)
- **Session management** — stricter requirements on idle timeout, session binding

**GGID Status:**

| Requirement | NIST 800-63B-4 | GGID |
|-------------|----------------|------|
| Phishing-resistant auth (WebAuthn) | AAL2+ expected | DONE — WebAuthn with attestation |
| Password policy (no forced rotation) | Don't force rotation | PARTIAL — has rotation policy (should be optional) |
| Passwordless AAL2 | FIDO2 alone = AAL2 | NOT MAPPED — GGID doesn't expose AAL levels in tokens |
| Session idle timeout | Configurable per AAL | DONE — session timeout middleware exists |
| Federation (FAL) | New assurance level | NO — no federation metadata/R&P |
| Breach password checking | Compromised password detection | DONE — HIBP breach detection |

**Gap:** GGID doesn't expose AAL (Authenticator Assurance Level) claims in JWT tokens. Step-up authentication exists but doesn't map to standardized AAL1/AAL2/AAL3 levels. `acr` and `amr` claims in OIDC tokens should reflect the actual assurance level achieved.

## 3. WebAuthn Hybrid Transport (Cross-Device Authentication)

**Industry Status:** Hybrid transport (QR code + BLE) is the standard for cross-device passkey authentication. User scans QR on desktop → phone authenticates via biometrics → desktop gets credential. Supported by all major platforms (Apple, Google, Microsoft).

**GGID Status:**
- WebAuthn registration/login implemented with `mediation: "conditional"` support
- **No explicit hybrid transport configuration** — the browser handles transport selection, but GGID doesn't optimize for cross-device flows
- **Missing:** Cross-device analytics (how many users authenticate via hybrid vs platform), conditional UI autofill integration

**Assessment:** GGID's WebAuthn implementation is server-side correct — hybrid transport works automatically when the browser supports it. The gap is in frontend UX (conditional UI for autofill, which was already flagged in a previous research cycle).

---

## Summary: New Backlog Items

1. **[P1] AAL/AMR Claims in JWT + Step-Up Mapping** — NIST 800-63B-4 requires standardized assurance level claims. Map GGID's step-up authentication to AAL1/AAL2/AAL3 and include `acr`/`amr` in tokens.

2. **[P2] SD-JWT Verifiable Credential Support** — EU EUDI 2026 mandate. Implement SD-JWT issuance (OpenID4VCI) and verification (OpenID4VP) for credential exchange.

3. **[P3] Federation Assurance Level (FAL)** — NIST 800-63-4 new FAL requirements. Federation metadata, R&P profiles.
