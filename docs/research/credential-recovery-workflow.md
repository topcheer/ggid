# Credential Recovery Workflow Engine: Multi-Factor, DB-Backed Recovery for GGID

> **Focus**: A production-grade credential recovery system — replacing the current in-memory placeholder with a DB-backed, multi-step verification workflow supporting password reset, passkey device loss, MFA lockout, social login account recovery, and admin-assisted recovery with full audit trails.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§11), curl commands (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [The Recovery Problem](#2-the-recovery-problem)
3. [GGID Current State Analysis](#3-ggid-current-state-analysis)
4. [Gap Analysis](#4-gap-analysis)
5. [Proposed Architecture](#5-proposed-architecture)
6. [Endpoint Precondition Check](#6-endpoint-precondition-check)
7. [API Design + Curl Commands](#7-api-design--curl-commands)
8. [Database Schema](#8-database-schema)
9. [Recovery Scenarios](#9-recovery-scenarios)
10. [Security Considerations](#10-security-considerations)
11. [Implementation Backlog with DoD](#11-implementation-backlog-with-dod)
12. [Competitive Differentiation](#12-competitive-differentiation)

---

## 1. Executive Summary

Account recovery is the **weakest link** in identity security. Attackers target recovery flows because they bypass primary authentication. A well-designed recovery system must verify identity through multiple independent factors, enforce time delays for sensitive operations, maintain complete audit trails, and support various recovery scenarios — from forgotten passwords to lost passkey devices.

GGID has two recovery-related components:
- **Password reset flow** (`auth_service.go:324` — `ForgotPassword()` + `ResetPassword()`) — **DB-backed**, works correctly
- **Identity recovery service** (`identity_recovery.go:47` — `IdentityRecoveryService`) — **In-memory** (`map[string]*RecoveryRequest` + `sync.RWMutex`), violates acceptance checklist

The in-memory recovery service has critical issues:
1. **Lost on restart** — all recovery requests disappear when service restarts
2. **Weak token generation** — `rtok_{seq}_{timestamp}` — guessable, not cryptographic
3. **No multi-factor verification** — single-factor email/phone/backup only
4. **No DB-backed audit** — audit entries in `[]RecoveryAuditEntry` slice
5. **No handler/API** — no REST endpoints for recovery operations
6. **No passkey-specific recovery** — no Temporary Access Pass for device loss
7. **No admin-assisted recovery** — no admin approval workflow
8. **No rate limiting** — recovery initiation not rate-limited
9. **24h mandatory wait for all** — no graduated wait based on risk
10. **No notification** — no security alert when recovery initiated

**Recommendation**: Replace the in-memory service with a PostgreSQL-backed recovery engine with multi-step verification, cryptographic tokens, Temporary Access Pass (TAP), admin approval workflow, graduated time delays, rate limiting, and full DB-backed audit.

**Estimated effort**: 3 sprints for MVP (DB + API + multi-factor verification + Console UI).

---

## 2. The Recovery Problem

### The Recovery Paradox

> Recovery must be **easy enough** for legitimate users to regain access, but **hard enough** that attackers can't social-engineer their way in. This tension is the core design challenge.

### Recovery Threat Model

| Threat | Vector | Mitigation |
|--------|--------|-----------|
| **Email compromise** | Attacker controls user's email → receives reset link | Require additional factor beyond email |
| **SIM swap** | Attacker ports phone number → receives SMS OTP | Don't rely on SMS alone; require backup codes or admin approval |
| **Social engineering** | Attacker tricks helpdesk into resetting account | Require verification checklist + dual approval for privileged accounts |
| **Token guessing** | Brute-force reset tokens | Cryptographic random tokens (32 bytes) + rate limiting |
| **Token interception** | MITM captures reset link | HTTPS + short TTL (15 min) + single-use tokens |
| **Race condition** | Multiple simultaneous recovery attempts | One active recovery per user |

### Recovery Scenarios by Credential Type

| Scenario | Credential Lost | Recovery Method |
|----------|----------------|-----------------|
| Forgot password | Password | Email reset link → new password |
| Lost passkey device | WebAuthn credential | Backup factor → Temporary Access Pass → new passkey enrollment |
| Lost phone (TOTP) | TOTP secret | Backup codes → new TOTP enrollment |
| Lost phone (SMS) | Phone number | Email + backup codes → new phone verification |
| Social account lockout | Google/Microsoft login | Email verification → local password fallback |
| Admin lockout | Admin locked out | Break-glass: second admin approval + hardware token |
| All factors lost | Everything | Admin-assisted identity proofing + manual verification |

---

## 3. GGID Current State Analysis

### Existing Recovery Infrastructure

| Component | File:Line | Status | Issue |
|-----------|-----------|--------|-------|
| Password reset (ForgotPassword) | `auth_service.go:324` | **DB-backed** ✅ | Works correctly |
| Password reset (ResetPassword) | `auth_service.go:377` | **DB-backed** ✅ | Works correctly |
| Reset token issuance | `password_service.go` | **DB-backed** ✅ | Crypto token via Redis |
| IdentityRecoveryService | `identity_recovery.go:47` | **In-memory** ❌ | `map[string]*RecoveryRequest` |
| RecoveryMethod enum | `identity_recovery.go:9` | **Defined** | email, phone, backup_codes |
| RecoveryStatus enum | `identity_recovery.go:17` | **Defined** | initiated, verified, completed, expired, cancelled |
| RecoveryRequest struct | `identity_recovery.go:27` | **Defined** | Has WaitUntil for time-delayed recovery |
| RecoveryAuditEntry | `identity_recovery.go:39` | **Defined** | But stored in-memory slice |
| CleanupExpired | `identity_recovery.go:145` | **Implemented** | In-memory only |
| Step-up auth | `stepup.go:31` | **Implemented** | `InitStepUp()` for elevated sessions |
| Impersonation | `impersonation.go:29` | **Implemented** | Admin can impersonate users |
| Backup codes | `backup_codes_pg.go:27` | **DB-backed** ✅ | EnsureSchema + pg repo |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **In-memory storage** | Recovery requests lost on restart |
| 2 | **Weak tokens** | `rtok_{seq}_{timestamp}` — guessable |
| 3 | **No multi-factor verification** | Single factor only |
| 4 | **No API endpoints** | No REST CRUD for recovery operations |
| 5 | **No TAP (Temporary Access Pass)** | No secure re-enrollment path for lost passkeys |
| 6 | **No admin approval workflow** | No dual-control for sensitive recovery |
| 7 | **No rate limiting** | Recovery initiation unlimited |
| 8 | **No notifications** | No security alert on recovery initiation |
| 9 | **Fixed 24h wait** | No risk-based graduated delay |
| 10 | **No DB-backed audit** | Audit trail lost on restart |

---

## 4. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "User lost phone with passkey — needs to recover account" | In-memory recovery, weak token, no TAP | Multi-factor verification → TAP → new passkey enrollment |
| 2 | "Admin needs to approve recovery for privileged account" | No approval workflow | Admin reviews + approves; dual control |
| 3 | "Track recovery audit trail across restarts" | Lost | DB-persisted, queryable audit |
| 4 | "Rate-limit recovery attempts to prevent abuse" | No limit | 3 attempts per hour per user/IP |
| 5 | "Notify security team when admin recovery initiated" | No notification | Email + webhook alert |
| 6 | "Risk-based wait: immediate for low-risk, 72h for high-risk" | Fixed 24h | Graduated: 0min (low) → 24h (medium) → 72h (high) |

---

## 5. Proposed Architecture

```
                    ┌──────────────────────────────────────────────┐
                    │     Credential Recovery Engine                │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Recovery Request Store (PostgreSQL)  │    │
                    │  │  - recovery_requests table             │    │
                    │  │  - recovery_verification_steps table   │    │
                    │  │  - recovery_audit_log table            │    │
                    │  │  - temporary_access_passes table       │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Multi-Step Verification Pipeline     │    │
                    │  │                                      │    │
                    │  │  Step 1: Initiate (email/username)    │    │
                    │  │     ↓                                │    │
                    │  │  Step 2: Primary verification         │    │
                    │  │     - Email OTP / Backup codes /      │    │
                    │  │       Another device passkey          │    │
                    │  │     ↓                                │    │
                    │  │  Step 3: Risk assessment              │    │
                    │  │     - IP geo, device, behavior        │    │
                    │  │     → Determines wait period          │    │
                    │  │     ↓                                │    │
                    │  │  Step 4: Secondary (if high risk)     │    │
                    │  │     - Admin approval / Identity       │    │
                    │  │       proofing / Additional factor    │    │
                    │  │     ↓                                │    │
                    │  │  Step 5: Temporary Access Pass (TAP)  │    │
                    │  │     - 15-min single-use credential    │    │
                    │  │     ↓                                │    │
                    │  │  Step 6: New credential enrollment    │    │
                    │  │     - New passkey / password / TOTP   │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Supporting Services                  │    │
                    │  │  ├── Rate Limiter (Redis)             │    │
                    │  │  ├── Notification Service (email)     │    │
                    │  │  ├── Risk Assessment Service          │    │
                    │  │  └── Audit Publisher (NATS)           │    │
                    │  └──────────────────────────────────────┘    │
                    └──────────────────────────────────────────────┘
```

---

## 6. Endpoint Precondition Check

### Existing Endpoints (Reusable)

| Endpoint | File:Line | Status | Reusable? |
|----------|-----------|--------|-----------|
| `POST /api/v1/auth/forgot-password` | `auth_service.go:324` | **DB-backed** ✅ | Yes — password reset pipeline |
| `POST /api/v1/auth/reset-password` | `auth_service.go:377` | **DB-backed** ✅ | Yes — token verification |
| `POST /api/v1/auth/step-up/init` | `stepup.go:31` | **DB-backed** ✅ | Yes — step-up challenge |
| `POST /api/v1/auth/step-up/verify` | `stepup.go:65` | **DB-backed** ✅ | Yes — verification |
| Backup codes | `backup_codes_pg.go:27` | **DB-backed** ✅ | Yes — recovery factor |
| Impersonation | `impersonation.go:29` | **Implemented** | Yes — admin-assisted recovery |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/auth/recovery/initiate` | POST | Start recovery workflow | P0 |
| `/api/v1/auth/recovery/{id}/verify` | POST | Submit verification factor | P0 |
| `/api/v1/auth/recovery/{id}/status` | GET | Check recovery status | P0 |
| `/api/v1/auth/recovery/{id}/complete` | POST | Complete recovery + enroll new credential | P0 |
| `/api/v1/auth/recovery/{id}/cancel` | POST | Cancel recovery | P0 |
| `/api/v1/auth/recovery/tap` | POST | Use Temporary Access Pass | P0 |
| `/api/v1/auth/recovery/admin/list` | GET | List pending admin approvals | P1 |
| `/api/v1/auth/recovery/admin/{id}/approve` | POST | Admin approve recovery | P1 |
| `/api/v1/auth/recovery/admin/{id}/reject` | POST | Admin reject recovery | P1 |
| `/api/v1/auth/recovery/history` | GET | Recovery audit trail | P1 |

---

## 7. API Design + Curl Commands

### Initiate Recovery

```bash
curl -X POST https://ggid.corp.com/api/v1/auth/recovery/initiate \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "alice@corp.com",
    "reason": "lost_passkey_device"
  }'

# Response:
{
  "recovery_id": "rec_7f3a2b1c-...",
  "status": "verification_required",
  "verification_options": [
    { "method": "email_otp", "target": "a***@corp.com" },
    { "method": "backup_codes", "available": true },
    { "method": "another_device", "device_hint": "MacBook Pro" }
  ],
  "expires_at": "2026-07-17T11:00:00Z"
}
```

### Submit Verification

```bash
curl -X POST https://ggid.corp.com/api/v1/auth/recovery/rec_7f3a2b1c/verify \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "method": "email_otp",
    "code": "482916"
  }'

# Response (low-risk user, immediate):
{
  "status": "approved",
  "tap": {
    "pass_code": "TAP-ABCD-1234-EFGH",
    "expires_at": "2026-07-17T10:30:00Z",
    "must_enroll_credential": true
  }
}

# Response (high-risk user, delayed):
{
  "status": "pending_admin_approval",
  "estimated_wait_hours": 24,
  "notification_sent": true
}
```

### Complete Recovery (Enroll New Credential)

```bash
curl -X POST https://ggid.corp.com/api/v1/auth/recovery/rec_7f3a2b1c/complete \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "tap_code": "TAP-ABCD-1234-EFGH",
    "new_credential_type": "passkey",
    "webauthn_attestation": "{...}"
  }'

# Response:
{
  "status": "completed",
  "new_credential_enrolled": true,
  "old_credentials_revoked": true,
  "audit_id": "aud_9e8f7g6h"
}
```

### Admin Approval Flow

```bash
# List pending approvals
curl https://ggid.corp.com/api/v1/auth/recovery/admin/list \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Approve
curl -X POST https://ggid.corp.com/api/v1/auth/recovery/admin/rec_7f3a2b1c/approve \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"note": "Verified via video call with employee"}'

# Reject
curl -X POST https://ggid.corp.com/api/v1/auth/recovery/admin/rec_7f3a2b1c/reject \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"reason": "Could not verify identity"}'
```

---

## 8. Database Schema

```sql
-- Recovery requests
CREATE TABLE recovery_requests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    
    -- Recovery context
    reason              VARCHAR(64) NOT NULL,        -- 'lost_passkey', 'forgot_password', 'mfa_lockout'
    requested_identifier VARCHAR(256),               -- email/username submitted
    
    -- Verification
    verification_factors_required INT DEFAULT 1,      -- Number of factors needed
    verification_factors_completed INT DEFAULT 0,
    
    -- Risk assessment
    risk_level          VARCHAR(16) DEFAULT 'medium', -- 'low', 'medium', 'high'
    risk_score          INT DEFAULT 50,
    requires_admin_approval BOOLEAN DEFAULT false,
    admin_approver_id   UUID,
    admin_approved_at   TIMESTAMPTZ,
    admin_note          TEXT,
    
    -- Time-delayed recovery
    wait_until          TIMESTAMPTZ,                  -- Earliest completion allowed
    
    -- Temporary Access Pass
    tap_code_hash       VARCHAR(256),                -- bcrypt hash of TAP
    tap_expires_at      TIMESTAMPTZ,
    tap_used_at         TIMESTAMPTZ,
    
    -- State
    status              VARCHAR(32) NOT NULL DEFAULT 'initiated',
    -- 'initiated', 'verifying', 'pending_admin', 'approved', 'tap_issued', 'completed', 'expired', 'cancelled'
    
    completed_at        TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ NOT NULL,         -- Overall recovery request expiry
    
    -- Audit
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address          VARCHAR(45),
    user_agent          TEXT
);

-- Recovery verification steps (multi-factor)
CREATE TABLE recovery_verification_steps (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recovery_request_id UUID NOT NULL REFERENCES recovery_requests(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL,
    method              VARCHAR(32) NOT NULL,         -- 'email_otp', 'backup_codes', 'another_device'
    target_hint         VARCHAR(128),                 -- 'a***@corp.com' (masked)
    status              VARCHAR(16) NOT NULL DEFAULT 'pending',
    verified_at         TIMESTAMPTZ,
    attempt_count       INT DEFAULT 0,
    code_hash           VARCHAR(256),                 -- bcrypt hash of verification code
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Recovery audit log (DB-backed, replaces in-memory slice)
CREATE TABLE recovery_audit_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    recovery_request_id UUID NOT NULL REFERENCES recovery_requests(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL,
    action              VARCHAR(64) NOT NULL,         -- 'initiate', 'verify', 'approve', 'complete', 'cancel'
    method              VARCHAR(32),
    actor_id            UUID,                         -- Who performed the action (user or admin)
    actor_type          VARCHAR(16) DEFAULT 'user',   -- 'user', 'admin', 'system'
    detail              JSONB DEFAULT '{}',
    ip_address          VARCHAR(45),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Temporary Access Passes (standalone table for admin-issued TAPs)
CREATE TABLE temporary_access_passes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    pass_code_hash      VARCHAR(256) NOT NULL,        -- bcrypt hash
    expires_at          TIMESTAMPTZ NOT NULL,         -- 15-60 min default
    used_at             TIMESTAMPTZ,
    used_from_ip        VARCHAR(45),
    used_from_ua        TEXT,
    created_by          UUID NOT NULL,
    created_by_name     VARCHAR(256),
    reason              VARCHAR(256),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_recovery_tenant_user ON recovery_requests (tenant_id, user_id, status);
CREATE INDEX idx_recovery_status ON recovery_requests (tenant_id, status) WHERE status IN ('pending_admin', 'initiated', 'verifying');
CREATE INDEX idx_recovery_steps_request ON recovery_verification_steps (recovery_request_id);
CREATE INDEX idx_recovery_audit_tenant_time ON recovery_audit_log (tenant_id, created_at DESC);
CREATE INDEX idx_recovery_audit_user ON recovery_audit_log (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_tap_user_active ON temporary_access_passes (tenant_id, user_id) WHERE used_at IS NULL;
```

---

## 9. Recovery Scenarios

### Scenario 1: Low-Risk Password Reset

```
User: "I forgot my password"
Risk: Low (known device, known IP, no prior suspicious activity)

Flow:
1. POST /recovery/initiate → email_otp sent
2. POST /recovery/{id}/verify (email_otp code) → verified
3. Immediate TAP issued (no admin approval, no wait)
4. POST /recovery/{id}/complete → set new password
5. Old password hash invalidated
6. Audit: "password reset via email verification"
Total time: ~2 minutes
```

### Scenario 2: Lost Passkey Device

```
User: "I lost my phone with my passkey"
Risk: Medium (new device, but known IP)

Flow:
1. POST /recovery/initiate → verification options: email_otp + backup_codes
2. POST /recovery/{id}/verify (backup_codes) → verified
3. POST /recovery/{id}/verify (email_otp) → verified (2 factors)
4. Risk assessment: medium → 2h wait
5. After wait → TAP issued
6. POST /recovery/{id}/complete → enroll new passkey
7. Old passkey credentials revoked
8. Audit: "passkey recovery via backup_codes + email_otp"
Total time: ~2 hours
```

### Scenario 3: High-Risk Admin Recovery

```
User: "I lost everything — phone, laptop, no backup codes"
Risk: High (unknown device, unknown IP)

Flow:
1. POST /recovery/initiate → risk score = 85 (high)
2. Requires admin approval
3. Admin reviews: verifies via video call, employee ID
4. Admin approves via /recovery/admin/{id}/approve
5. 72h mandatory wait (time-delayed recovery)
6. After wait → TAP issued
7. POST /recovery/{id}/complete → identity proofing + new credentials
8. Security team notified
9. Audit: "full account recovery via admin approval + identity proofing"
Total time: 72+ hours
```

---

## 10. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Recovery account takeover** | Multi-factor verification (≥2 factors for medium/high risk) |
| **Weak token brute-force** | crypto/rand 32 bytes + bcrypt hash + rate limiting (3 attempts/hour) |
| **TAP abuse** | 15-min TTL, single-use, IP+UA binding |
| **Admin social engineering** | Dual approval for privileged accounts, verification checklist |
| **Information disclosure** | Masked targets (a***@corp.com), generic errors ("if account exists...") |
| **Race condition** | One active recovery per user (cancel previous on new initiate) |
| **Audit tampering** | Append-only audit log with NATS publication |
| **Rate limiting** | Redis: 3 initiations/hour per IP, 5/hour per user |

---

## 11. Implementation Backlog with DoD

### P0 — DB-Backed Engine + API (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Recovery DB schema | ✅ CREATE TABLE in migration ✅ go build PASS ✅ No in-memory map | 1d |
| 2 | Recovery repository | ✅ CRUD backed by pgx ✅ `if err != nil` guards ✅ ≥3 tests | 3d |
| 3 | Replace IdentityRecoveryService | ✅ Uses repo (not map) ✅ No sync.RWMutex ✅ Crypto tokens (crypto/rand) ✅ ≥3 tests | 3d |
| 4 | Recovery API endpoints | ✅ 6 endpoints registered in http.go ✅ From handler to repo chain works ✅ curl test PASS ✅ ≥3 tests | 4d |
| 5 | Multi-factor verification pipeline | ✅ ≥2 factors for medium risk ✅ Step tracking ✅ ≥3 tests | 3d |
| 6 | Temporary Access Pass (TAP) | ✅ 15-min TTL ✅ Single-use ✅ bcrypt hash ✅ ≥3 tests | 2d |

### P1 — Risk Assessment + Admin Workflow (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | Risk-based graduated delay | ✅ Risk score → wait period (0min/2h/24h/72h) ✅ ≥3 tests | 3d |
| 8 | Admin approval workflow | ✅ List/approve/reject endpoints ✅ Dual approval for privileged ✅ ≥3 tests | 3d |
| 9 | Rate limiting + notification | ✅ Redis: 3/hr per IP ✅ Email alert on initiation ✅ ≥3 tests | 2d |
| 10 | Recovery audit trail (DB-backed) | ✅ Append-only audit log ✅ Queryable via API ✅ ≥3 tests | 2d |

### P2 — Console UI (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 11 | Self-service recovery page | ✅ Initiate + verify flow ✅ Status tracking ✅ TAP entry | 3d |
| 12 | Admin recovery console | ✅ Pending approvals list ✅ Approve/reject with notes ✅ Audit trail view | 3d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 13 | Identity proofing (KYC) | Government ID verification for high-risk recovery |
| 14 | Break-glass recovery | Hardware token + second admin for emergency access |
| 15 | Recovery analytics | Success rate, time-to-complete, by reason |
| 16 | WebAuthn multi-device recovery | Use another device's passkey to authorize recovery |
| 17 | Automated risk scoring | ML model for risk score based on login patterns |

---

## 12. Competitive Differentiation

| Feature | GGID (target) | Okta | Microsoft Entra | Auth0 | Keycloak |
|---------|---------------|------|-----------------|-------|----------|
| **Password reset** | **DB-backed** ✅ | Yes | Yes | Yes | Yes |
| **Passkey device loss recovery** | **TAP + multi-factor** | Yes (FastPass) | Yes (SSAR) | Custom | No |
| **Multi-factor verification** | **≥2 for high risk** | Yes | Yes | Custom | No |
| **Admin approval workflow** | **Dual control** | Yes | Yes | No | No |
| **Temporary Access Pass** | **15-min single-use** | Yes | Yes (TAP) | No | No |
| **Risk-based delay** | **0min→72h graduated** | Yes | Yes | No | Fixed |
| **DB-backed audit** | **PostgreSQL** | Yes | Yes | Yes | Yes |
| **Rate limiting** | **Redis-backed** | Yes | Yes | Custom | No |
| **Open source** | **Yes (Apache 2.0)** | No | No | No | Yes |

**Key differentiator**: GGID would be the only open-source IAM with risk-based graduated recovery + multi-factor verification + admin approval workflow + TAP — covering all recovery scenarios from password reset to complete credential loss.

---

## References

- [Microsoft Entra Self-Service Account Recovery](https://techcommunity.microsoft.com/blog/coreinfrastructureandsecurityblog/the-future-of-identity-self-service-account-recovery-preview-in-microsoft-entra/4499749) — SSAR for passwordless users
- [MojoAuth: Passwordless Recovery Mechanisms](https://mojoauth.com/ciam-101/passwordless-recovery-mechanisms) — Recovery design patterns
- [NIST SP 800-63B §6.1](https://pages.nist.gov/800-63-3/sp800-63b.html) — Account recovery requirements
- [GGID IdentityRecoveryService](../services/identity/internal/service/identity_recovery.go) — Current in-memory implementation at line 47
- [GGID ForgotPassword](../services/auth/internal/service/auth_service.go) — DB-backed password reset at line 324
- [GGID StepUp Auth](../services/auth/internal/service/stepup.go) — Step-up challenge at line 31
- [GGID Backup Codes](../services/auth/internal/service/backup_codes_pg.go) — DB-backed backup codes at line 27
- [GGID Impersonation](../services/auth/internal/service/impersonation.go) — Admin impersonation at line 29
