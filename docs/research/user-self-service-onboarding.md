# User Self-Service & Onboarding Experience: Production Implementation Guide for GGID

> **Focus**: End-user self-service capabilities — registration, password reset, profile management, account linking, device management, session management, MFA enrollment, and privacy center — building on GGID's existing auth infrastructure.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `auth/server/` (existing handlers), `identity/server/` (user profiles), `consent-management-platform.md`.
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State](#2-ggid-current-state)
3. [Gap Analysis](#3-gap-analysis)
4. [Self-Service Architecture](#4-self-service-architecture)
5. [Registration & Email Verification](#5-registration--email-verification)
6. [Password Reset Flow](#6-password-reset-flow)
7. [Profile Management](#7-profile-management)
8. [Account Linking](#8-account-linking)
9. [Device & Session Self-Service](#9-device--session-self-service)
10. [MFA Self-Enrollment](#10-mfa-self-enrollment)
11. [Privacy Center](#11-privacy-center)
12. [Endpoint Precondition Check](#12-endpoint-precondition-check)
13. [Implementation Backlog with DoD](#13-implementation-backlog-with-dod)
14. [Competitive Differentiation](#14-competitive-differentiation)

---

## 1. Executive Summary

GGID has admin-driven user management but lacks end-user self-service. Enterprise IAM requires users to manage their own accounts — registration, password reset, MFA enrollment, device management, session revocation, and data export (GDPR).

**Existing self-service components:**
- Password reset (partial — `auth/service/` has flows) ⚠️
- MFA enrollment (JIT + admin-driven, `jit_mfa_handler.go:22`) ✅
- WebAuthn/passkey registration (`auth/webauthn/`) ✅
- Device bindings (`auth/service/device_binding.go`) ✅
- Session management (`auth/service/session_management.go`) ✅
- Break-glass access ✅

**Missing:**
1. **No self-service registration** — Admin must create accounts
2. **No email verification flow** — No verification token send/verify
3. **No profile self-edit** — Users can't change own email/phone
4. **No account linking** — Can't link Google + password + passkey
5. **No device self-service** — Users can't see/revoke own devices
6. **No session list** — Users can't see all active sessions
7. **No data export** — GDPR portability not user-accessible
8. **No notification preferences** — Users can't opt out of emails

**Recommendation**: Build a complete self-service portal with 8 flows, leveraging existing auth/identity infrastructure.

---

## 2. GGID Current State

| Component | File | Status |
|-----------|------|--------|
| Password reset | `auth/service/` | ⚠️ Partial |
| MFA JIT enrollment | `auth/server/jit_mfa_handler.go:22` | ✅ Admin-triggered |
| WebAuthn registration | `auth/webauthn/handler.go:477` | ✅ |
| Device bindings | `auth/service/device_binding.go` | ✅ DB-backed |
| Session management | `auth/service/session_management.go` | ✅ |
| User CRUD | `identity/server/` | ✅ Admin-only |
| Passwordless | `auth/server/passwordless_*.go` | ✅ |
| Break-glass | `auth/server/break_glass_*.go` | ✅ |
| Impersonation | `auth/service/auth_service.go:29` | ✅ |
| Preferences | `identity/server/preferences_handler.go` | ✅ |

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | No self-registration | Users can't sign up |
| 2 | No email verification | Account ownership unverified |
| 3 | No profile self-edit | Users depend on admin |
| 4 | No account linking | Multiple accounts per person |
| 5 | No device self-service | Lost devices can't be revoked by user |
| 6 | No session self-service | Users can't see/revoke sessions |
| 7 | No data export | GDPR Art. 20 not user-accessible |
| 8 | No notification prefs | Users can't control notifications |

---

## 4. Self-Service Architecture

```
┌──────────────────────────────────────────────┐
│         Self-Service Portal (Console)         │
│                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Register │ │ Profile  │ │ Security │     │
│  │ +Verify  │ │ +Edit    │ │ +MFA     │     │
│  └──────────┘ └──────────┘ └──────────┘     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Devices  │ │ Sessions │ │ Privacy  │     │
│  │ +Revoke  │ │ +Revoke  │ │ +Export  │     │
│  └──────────┘ └──────────┘ └──────────┘     │
│  ┌──────────┐                                │
│  │ Linked   │                                │
│  │ Accounts │                                │
│  └──────────┘                                │
└──────────────────────────────────────────────┘
```

---

## 5. Registration & Email Verification

### Flow

```
1. User submits: POST /api/v1/self-service/register
   { email, password, display_name }
2. GGID creates user (status=pending_verification)
3. GGID sends verification email with token
4. User clicks link: GET /verify?token=xxx
5. GGID verifies token → sets status=active
6. User can now log in
```

### Security: Rate limit registration (max 5/hour per IP), verify email format, password complexity.

---

## 6. Password Reset Flow

### Flow (verify existing)

```
1. User requests: POST /api/v1/self-service/password/reset-request
   { email }
2. GGID sends reset email with token (30-min TTL)
3. User clicks: POST /api/v1/self-service/password/reset
   { token, new_password }
4. GGID validates token → updates password → revoke all sessions
5. User must re-authenticate
```

---

## 7. Profile Management

```bash
# User updates own profile
PUT /api/v1/self-service/profile
{
  "display_name": "Alice Chen",
  "phone": "+1-415-555-0123",
  "avatar_url": "https://..."
}

# Email change requires verification
POST /api/v1/self-service/profile/email-change
{ "new_email": "alice@new.com" }
# → Sends verification to new email
# → User must click link to confirm
```

---

## 8. Account Linking

### Link Multiple Auth Methods

```bash
# Link Google account to existing GGID account
POST /api/v1/self-service/accounts/link
{ "provider": "google", "id_token": "eyJ..." }

# Link passkey
POST /api/v1/self-service/accounts/link
{ "provider": "passkey", "credential": {...} }

# List linked accounts
GET /api/v1/self-service/accounts/linked
# → [{ "provider": "password" }, { "provider": "google" }, { "provider": "passkey" }]

# Unlink
DELETE /api/v1/self-service/accounts/google
```

---

## 9. Device & Session Self-Service

```bash
# List user's devices
GET /api/v1/self-service/devices
# → [{ device_id, name, platform, last_seen, trusted }]

# Revoke a device
DELETE /api/v1/self-service/devices/{id}

# List active sessions
GET /api/v1/self-service/sessions
# → [{ session_id, ip, geo, device, created_at, last_activity }]

# Revoke a session
DELETE /api/v1/self-service/sessions/{id}
```

---

## 10. MFA Self-Enrollment

```bash
# List enrolled MFA methods
GET /api/v1/self-service/mfa
# → [{ type: "totp", enrolled_at: "..." }, { type: "passkey", ... }]

# Enroll new TOTP
POST /api/v1/self-service/mfa/totp/enroll
# → { secret, qr_code_url }

# Verify enrollment
POST /api/v1/self-service/mfa/totp/verify
{ "code": "123456" }

# Remove MFA method (requires current MFA challenge)
DELETE /api/v1/self-service/mfa/totp
```

---

## 11. Privacy Center

```bash
# Export user data (GDPR Art. 20)
POST /api/v1/self-service/privacy/export
# → { export_id, status: "processing", estimated_time: "5min" }

# Download when ready
GET /api/v1/self-service/privacy/export/{id}/download
# → JSON file with all user data

# Delete account (GDPR Art. 17)
POST /api/v1/self-service/privacy/delete-account
{ "reason": "user_request", "confirm_password": "..." }
# → Triggers erasure pipeline (see pets-privacy-by-design.md)
```

---

## 12. Endpoint Precondition Check

### Existing (Reuse)

| Component | File | Status | Reuse |
|----------|------|--------|-------|
| Session management | `auth/service/session_management.go` | ✅ | Session list/revoke |
| Device bindings | `auth/service/device_binding.go` | ✅ | Device list/revoke |
| WebAuthn reg | `auth/webauthn/handler.go:477` | ✅ | Passkey enrollment |
| JIT MFA | `auth/server/jit_mfa_handler.go:22` | ✅ | MFA enrollment pattern |
| Preferences | `identity/server/preferences_handler.go` | ✅ | Notification prefs |
| User CRUD | `identity/server/` | ✅ | Profile edit |

### New Endpoints

| Endpoint | Method | Priority |
|----------|--------|----------|
| `/api/v1/self-service/register` | POST | P0 |
| `/api/v1/self-service/verify-email` | POST | P0 |
| `/api/v1/self-service/password/reset-request` | POST | P0 |
| `/api/v1/self-service/password/reset` | POST | P0 |
| `/api/v1/self-service/profile` | GET/PUT | P0 |
| `/api/v1/self-service/devices` | GET/DELETE | P0 |
| `/api/v1/self-service/sessions` | GET/DELETE | P0 |
| `/api/v1/self-service/mfa` | GET/POST/DELETE | P0 |
| `/api/v1/self-service/accounts/link` | POST | P1 |
| `/api/v1/self-service/privacy/export` | POST | P1 |
| `/api/v1/self-service/privacy/delete-account` | POST | P1 |

---

## 13. Implementation Backlog with DoD

### P0 — Core Self-Service (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Registration + email verification | ✅ POST register ✅ Email send ✅ Token verify ✅ DB-backed ✅ ≥3 tests | 4d |
| 2 | Password reset (verify + fix flow) | ✅ Request → email → reset → revoke ✅ DB-backed ✅ ≥3 tests | 3d |
| 3 | Profile self-edit + email change | ✅ Update own profile ✅ Email change verification ✅ ≥3 tests | 3d |
| 4 | Device + session self-service | ✅ List own devices/sessions ✅ Revoke individual ✅ DB-backed ✅ ≥3 tests | 3d |
| 5 | MFA self-enrollment | ✅ List/enroll/remove ✅ Re-auth required for removal ✅ ≥3 tests | 3d |

### P1 — Account Linking + Privacy (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 6 | Account linking | ✅ Link Google/passkey/password ✅ Unlink ✅ ≥3 tests | 4d |
| 7 | Data export (GDPR Art. 20) | ✅ JSON export ✅ Async processing ✅ ≥3 tests | 3d |
| 8 | Account deletion (GDPR Art. 17) | ✅ Cascade deletion ✅ Confirm password ✅ ≥3 tests | 2d |
| 9 | Notification preferences | ✅ Opt-in/out categories ✅ DB-backed ✅ ≥3 tests | 2d |

### P2 — Console UI (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 10 | Self-service portal UI | ✅ 7 pages (profile/security/devices/sessions/privacy/accounts/notifications) ✅ Responsive | 5d |

---

## 14. Competitive Differentiation

| Feature | GGID (target) | Auth0 | Okta | Microsoft Entra | Keycloak |
|---------|---------------|-------|------|-----------------|----------|
| **Self-registration** | Target | Yes | Yes | Yes | Yes |
| **Email verification** | Target | Yes | Yes | Yes | Yes |
| **Profile self-edit** | Target | Yes | Yes | Yes | Yes |
| **Account linking** | Target | Yes | Yes | Yes | Partial |
| **Device self-service** | Target | Custom | Yes | Yes | No |
| **Session self-service** | Target | Custom | Yes | Yes | No |
| **MFA self-enroll** | Existing ✅ | Yes | Yes | Yes | Yes |
| **Data export (GDPR)** | Target | Yes | Yes | Yes | No |
| **Open source** | Yes | No | No | No | Yes |

---

## References

- [Auth0 Self-Service](https://auth0.com/docs/customize/self-service-portal)
- [Okta End-User Dashboard](https://help.okta.com/en-us/Content/Topics/End-user/end-user-home.htm)
- [Microsoft MyAccount](https://myaccount.microsoft.com/)
- [GDPR Art. 20 (Portability)](https://gdpr-info.eu/art-20-gdpr/)
- [GGID Session Management](../services/auth/internal/service/session_management.go)
- [GGID Device Bindings](../services/auth/internal/service/device_binding.go)
- [GGID WebAuthn](../services/auth/internal/webauthn/handler.go) — At line 477
- [GGID JIT MFA](../services/auth/internal/server/jit_mfa_handler.go) — At line 22
- [GGID Preferences](../services/identity/internal/server/preferences_handler.go)
- [GGID PETs Privacy](./pets-privacy-by-design.md) — Erasure pipeline
- [GGID Consent Management](./consent-management-platform.md) — GDPR compliance
