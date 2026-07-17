# CIAM 2025 Trends — GGID Gap Analysis

> Research round covering refresh token rotation, passwordless adoption, and consent management standards.

---

## 1. Refresh Token Rotation (RFC 6749 + OWASP)

**Industry Standard (2025):** OWASP and OAuth 2.1 draft mandate refresh token rotation — each use issues a new refresh token and invalidates the old one. Detection of reuse (old token used after rotation) signals token theft.

**GGID Status:** **DONE** — `auth_service.go:288` implements `RotateRefreshToken()`: validates old token, issues new refresh + access token, revokes old. Token reuse detection exists in `hijack_check_handler.go` (token_reuse_post_rotation rule).

**Gap:** None. GGID is compliant with OWASP refresh token rotation best practices.

---

## 2. Passwordless Authentication Adoption

**Industry Trend (2025):** Passwordless market reached $24.1B (18.24% CAGR). 48% of consumers abandon purchases due to auth friction. Passkeys going mainstream. Enterprise expectations: WebAuthn + magic links + SMS/Email OTP as standard CIAM methods.

**GGID Status:**

| Method | Implemented | Notes |
|--------|-------------|-------|
| WebAuthn/Passkeys | YES (49 refs in http.go) | Full attestation verification, L3 Signal API, credential persistence |
| Email OTP | YES (email_otp_handler.go) | In-memory store (acceptable — short-lived codes) |
| SMS OTP | YES (http.go:1598) | DEV mode logging only — no real SMS provider integration |
| Magic Links | YES (login_orchestrator.go:52) | Config handler exists (passwordless_config_handler.go) |

**Gap:** SMS OTP only has DEV logging (`log.Printf("[DEV] phone OTP...")`). No Twilio/AWS SNS integration. Magic link delivery mechanism unclear (needs email provider wiring). These are deployment configuration gaps, not code architecture gaps.

---

## 3. OAuth Token Expiration Draft (draft-ietf-oauth-refresh-token-expiration)

**Draft Status:** IETF draft proposes refresh tokens MUST NOT outlive user authorization. Servers SHOULD enforce absolute lifetime caps.

**GGID Status:** Refresh tokens have configurable TTL via `AUTH_REFRESH_TOKEN_TTL` env var. Old tokens are revoked on rotation. Compliant with draft direction.

---

## 4. Consent Management

**Industry Trend:** GDPR/CCPA enforcement driving granular consent — not just "I agree" checkbox but versioned, withdrawable, auditable consent records.

**GGID Status:** OAuth consent screen exists. Consent analytics endpoint (`/api/v1/oauth/consent/analytics`). Admin override capability. Missing: consent versioning, withdrawal API, consent audit trail.

---

## Summary: New Backlog Items

1. **[P2] SMS OTP Provider Integration** — Replace DEV logging with Twilio/AWS SNS provider. Backend task.
2. **[P2] Consent Versioning + Withdrawal API** — Versioned consent records, withdrawal endpoint, audit trail. Backend task.
3. **[P3] parStore Redis Migration** — Move PAR request_uri store to Redis for HA multi-replica. Backend task.
