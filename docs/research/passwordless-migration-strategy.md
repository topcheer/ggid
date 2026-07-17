# Passwordless Migration: Enterprise Strategy, Policy Engine, and Phased Rollout for GGID

> **Focus**: How GGID enables organizations to migrate from password-based authentication to phishing-resistant passkeys — covering the authentication method policy engine, phased migration tooling, enrollment nudging, password deprecation enforcement, metrics dashboards, and account recovery design.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `passwordless-auth-iam.md` covers individual passwordless methods (magic link, WebAuthn, OTP). This document covers the **enterprise migration program** — the policy engine, rollout phases, and organizational change management.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Why Passwords Must Die](#2-why-passwords-must-die)
3. [Phishing-Resistant Authentication Primer](#3-phishing-resistant-authentication-primer)
4. [Migration Strategy Overview](#4-migration-strategy-overview)
5. [Industry Landscape](#5-industry-landscape)
6. [GGID Current State Analysis](#6-ggid-current-state-analysis)
7. [Gap Analysis](#7-gap-analysis)
8. [Authentication Method Policy Engine](#8-authentication-method-policy-engine)
9. [Enrollment Nudge System](#9-enrollment-nudge-system)
10. [Password Deprecation Enforcement](#10-password-deprecation-enforcement)
11. [Account Recovery for Passwordless](#11-account-recovery-for-passwordless)
12. [Database Schema](#12-database-schema)
13. [API Design](#13-api-design)
14. [Metrics and Analytics](#14-metrics-and-analytics)
15. [Console UI Design](#15-console-ui-design)
16. [Security Considerations](#16-security-considerations)
17. [Competitive Differentiation](#17-competitive-differentiation)
18. [Implementation Backlog](#18-implementation-backlog)

---

## 1. Executive Summary

The FIDO Alliance estimates **5 billion passkeys** are in active use in 2026, with **87% of enterprises** deploying them for workforce authentication. Passwords remain the root cause behind most breaches. Every major IAM platform now supports passwordless authentication — but the challenge is not the technology, it's the **migration program**: how to move thousands of users from passwords to passkeys without lockouts, helpdesk floods, or security gaps.

GGID implements WebAuthn/passkey registration (`services/auth/internal/service/mfa_service.go`), password hashing with Argon2id (`pkg/crypto/crypto.go:68`), and password policy enforcement (`services/auth/internal/service/password_service.go`). However, GGID is **missing the migration management layer**:

1. **No authentication method policy engine** — cannot express "require passkey for Finance team after Oct 1"
2. **No enrollment nudging** — no configurable banners/prompts to encourage passkey enrollment
3. **No password deprecation enforcement** — cannot disable password login for users who have passkeys
4. **No migration metrics dashboard** — cannot track enrollment rate, password usage trends
5. **No passwordless account recovery** — no secure recovery flow when a passkey device is lost
6. **No conditional access based on auth method** — cannot require phishing-resistant auth for sensitive resources

**Recommendation**: Build an **Authentication Method Policy Engine** with phased migration tooling, enrollment nudging, password deprecation enforcement, recovery flows, and a metrics dashboard — making GGID the only open-source IAM with a complete passwordless migration toolkit.

**Estimated effort**: 4 sprints for MVP (policy engine + nudging + deprecation + metrics + recovery).

---

## 2. Why Passwords Must Die

### 2026 Threat Landscape

| Metric | Value | Source |
|--------|-------|--------|
| Breaches involving credentials | 80%+ | Verizon DBIR 2025 |
| Phishing success rate | 14-24% | FIDO Alliance 2026 |
| Password reset helpdesk tickets | 20-50% of all IT tickets | Gartner |
| Average cost per password reset | $70 (helpdesk labor + lost productivity) | Forrester |
| Passkeys in active use | 5 billion | FIDO Alliance 2026 |
| Enterprises deploying passkeys | 87% | FIDO Alliance survey 2026 |
| Helpdesk cost reduction post-passwordless | 35% | FIDO Alliance survey |
| Login time improvement | 45% faster | FIDO Alliance survey |

### Why Passwords Fail

```
┌─────────────────────────────────────────────────────────────────┐
│                    THE PASSWORD PROBLEM                          │
│                                                                 │
│  1. Phishing:       Fake login pages steal credentials          │
│  2. Credential stuffing:  Reused passwords from breaches        │
│  3. Password spray:  Common passwords across many accounts     │
│  4. Brute force:     Automated guessing                        │
│  5. Keyloggers:      Malware captures typed passwords           │
│  6. Shoulder surf:   Physical observation                       │
│  7. Database breach:  Server-side hash extraction               │
│  8. Password fatigue: Users reuse, simplify, write down        │
│  9. Reset cost:      $70 per reset, 20-50% of helpdesk tickets  │
│ 10. Compliance:      PCI DSS 4.0 requires MFA; passwords alone  │
│                      no longer satisfy auditors                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                  THE PASSKEY SOLUTION                            │
│                                                                 │
│  1. Phishing-proof:   Cryptographic origin binding prevents    │
│                       replay against fake sites                  │
│  2. No shared secret:  Private key never leaves device          │
│  3. No server breach:  Server stores only public key            │
│  4. No reuse:         Unique key per site (no cross-site risk)  │
│  5. No typing:        Biometric/PIN unlock, no password to steal│
│  6. No resets:        Key-based, not knowledge-based            │
│  7. Fast:             <2 seconds vs 8-15 seconds for password   │
│  8. AAL2/AAL3:        Meets NIST 800-63B requirements           │
│  9. FIDO2 certified:  Open standard, cross-platform             │
│ 10. Cost saving:      35% lower helpdesk costs                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Phishing-Resistant Authentication Primer

### Synced vs Device-Bound Passkeys

| Property | Synced Passkey | Device-Bound Passkey |
|----------|---------------|---------------------|
| **Key storage** | Encrypted, synced via platform cloud (Apple/Google/MS) | Locked to single authenticator (TPM, hardware key) |
| **NIST AAL** | AAL2 (phishing-resistant) | AAL3 (highest assurance) |
| **Attestation** | Limited/none | Full (via FIDO Metadata Service, AAGUID) |
| **Account recovery** | Easy (cloud restore, multi-device) | Harder (backup key or admin reset) |
| **Best for** | General workforce, BYOD | Admins, privileged, regulated roles |
| **Cost** | Free (platform built-in) | $25-70 per hardware key |
| **Cross-platform** | Yes (Apple ↔ Google ecosystem) | Yes (roaming security keys) |

### WebAuthn Authentication Flow

```
Registration:
  Browser ←→ Authenticator (device)
     1. Server sends challenge + relying party ID
     2. Authenticator generates key pair (scoped to RP ID)
     3. Private key stored in secure enclave / TPM
     4. Public key + attestation returned to server
     5. Server stores public key against user account

Authentication:
  Browser ←→ Authenticator (device)
     1. Server sends challenge bound to origin
     2. User unlocks authenticator (biometric / PIN)
     3. Authenticator signs challenge with private key
     4. Signature returned to server
     5. Server verifies signature against stored public key
     → Access granted
```

**Key security property**: The signed challenge is bound to the real origin domain. A phishing site on a look-alike domain cannot produce a valid assertion — this is the structural reason WebAuthn defeats scalable phishing.

---

## 4. Migration Strategy Overview

### The 4-Phase Enterprise Model

```
Phase 1: Assess & Inventory (Weeks 1-2)
├── Map applications to auth methods
├── Identify WebAuthn-capable apps
├── Catalog browser/device coverage
├── Set AAL targets per role/group
└── Baseline password reset costs

Phase 2: Pilot (Weeks 3-5)
├── Enroll IT staff + admins first (device-bound keys)
├── Define passkey profiles (attestation, AAGUID allow-lists)
├── Build account-recovery flows
├── Test fallback mechanisms
└── Measure enrollment UX

Phase 3: Workforce Rollout (Weeks 6-10)
├── Enable synced passkeys for all staff
├── Self-service enrollment portal
├── Enrollment nudging (banners, emails)
├── Monitor enrollment rate per cohort
└── Helpdesk training + playbooks

Phase 4: Enforce & Retire (Weeks 11-12+)
├── Disable password fallback for enrolled populations
├── Conditional access: require passkey for sensitive apps
├── Monitor sign-in telemetry
├── Decommission legacy auth paths
└── Report ROI metrics
```

### Decision Matrix: Which Approach

| Factor | Big Bang | Phased (Recommended) | Hybrid |
|--------|---------|--------------------|---------|
| User count | <500 | 500-100K | 100K+ |
| Risk tolerance | High | Medium | Low |
| Downtime tolerance | Some | None | None |
| Helpdesk capacity | Strong | Adequate | Limited |
| Legacy app count | Few | Some | Many |
| Timeline | Days | 8-12 weeks | 3-6 months |

---

## 5. Industry Landscape

### Comparison Matrix

| Feature | Okta | Microsoft Entra | Auth0 | Keycloak | **GGID (target)** |
|---------|------|-----------------|-------|----------|-------------------|
| **Passkey enrollment** | Yes (FastPass) | Yes (WHfB) | Yes | Yes | **Yes** |
| **Auth method policy** | App sign-in policy | Conditional Access | Actions | Auth flow config | **Policy engine (new)** |
| **Enrollment nudging** | Yes (dashboard) | Yes (Registration Campaigns) | Custom | No | **Configurable banners** |
| **Password deprecation** | Yes (passwordless mode) | Yes (password disable) | Custom | No | **Policy-based enforcement** |
| **Recovery flows** | Yes (admin reset) | Yes (temporary access pass) | Custom | No | **Multi-factor recovery** |
| **Metrics dashboard** | Yes (adoption reports) | Yes (usage insights) | Custom | No | **Migration dashboard** |
| **Device-bound attestation** | Yes | Yes | Yes | Partial | **Yes (existing WebAuthn)** |
| **Conditional access** | Yes | Yes | Via Actions | No | **Via policy service** |
| **Open source** | No | No | No | Yes | **Yes (Apache 2.0)** |

### Key Differentiator Gap

Okta and Microsoft Entra have **the most mature passwordless migration tooling**: enrollment campaigns, password disable policies, conditional access, recovery passes, and adoption dashboards. Keycloak and Auth0 require custom development for migration management. GGID has the raw WebAuthn capability but lacks the management layer.

---

## 6. GGID Current State Analysis

### Existing Passwordless Infrastructure

| Component | File | Status |
|-----------|------|--------|
| WebAuthn registration | `services/auth/internal/service/mfa_service.go` | **Implemented** — passkey enrollment |
| WebAuthn verification | `services/auth/internal/service/mfa_service.go` | **Implemented** — assertion verification |
| Attestation verification | `services/auth/internal/service/` (6 format verifiers) | **Implemented** — attestation format checking |
| Password hashing | `pkg/crypto/crypto.go:68` | **Implemented** — Argon2id with pepper |
| Password policy | `services/auth/internal/service/password_service.go:48` | **Implemented** — min length, complexity, history, expiry |
| Password update API | `services/auth/internal/service/auth_service.go:531` | **Implemented** — `UpdatePasswordPolicy()` |
| Magic link auth | `services/auth/internal/service/` | **Implemented** — passwordless email login |
| Email OTP | `services/auth/internal/service/email_otp.go` | **Implemented** — one-time password via email |
| TOTP MFA | `services/auth/internal/service/mfa_service.go` | **Implemented** — RFC 6238 authenticator apps |
| Step-up auth | `services/auth/internal/service/stepup.go:31` | **Implemented** — `InitStepUp()` for elevated sessions |
| Device tracking | `services/auth/internal/service/device_tracking.go` | **Implemented** — known device records |
| Conditional access | `services/policy/internal/server/conditional_access_handler.go` | **Implemented** — policy-based conditions |
| Risk assessment | `services/auth/internal/service/risk_auth.go:36` | **Implemented** — login risk scoring |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No auth method policy engine** | Cannot enforce "passkey required for group X after date Y" |
| 2 | **No enrollment nudging** | Users not prompted to enroll passkeys during login |
| 3 | **No password deprecation** | Cannot disable password login even when user has passkey |
| 4 | **No migration metrics** | Cannot track enrollment rate, password usage trends |
| 5 | **No passwordless recovery** | No secure recovery flow when passkey device is lost |
| 6 | **No AAL assessment** | Login flow doesn't compute/emit the user's Authenticator Assurance Level |
| 7 | **No enrollment status API** | No way to query "what auth methods does this user have?" |
| 8 | **No passkey profiles** | Cannot restrict to specific AAGUIDs (authenticator allow-list) |

---

## 7. Gap Analysis

### Real-World Scenarios That Fail Today

| # | Scenario | Current Behavior | Expected Behavior |
|---|----------|-----------------|-------------------|
| 1 | "Require passkey for Finance team starting Oct 1" | Cannot express this policy | AuthMethodPolicy enforces: if user in Finance and date >= Oct 1, require passkey |
| 2 | "Show enrollment banner to users without passkey" | No banner system | Configurable nudge: "Go passwordless" banner with one-click enrollment |
| 3 | "Disable password login for users who have a passkey" | Password always works | Policy: if user.has_passkey && policy.disable_password_for_passkey_users → block password |
| 4 | "Track what % of users have enrolled passkeys" | No metrics endpoint | Dashboard: enrollment rate, weekly trend, per-group breakdown |
| 5 | "User lost phone with passkey — recover account" | No recovery flow | Secure admin-assisted recovery with identity verification + backup factor |
| 6 | "Only allow YubiKey 5 (AAGUID) for admin enrollment" | No AAGUID restriction | Passkey profile: allowed_aaguibs = ["cb69481c-50b0-..."] |
| 7 | "Require AAL2 (phishing-resistant) for /admin endpoints" | No AAL concept in Gateway | Conditional access: resource.sensitivity requires AAL >= 2 |

---

## 8. Authentication Method Policy Engine

### Design

The AuthMethodPolicy engine is a declarative rule system that governs which authentication methods are allowed, required, or forbidden for specific users/groups/applications at specific times.

```go
// services/auth/internal/service/auth_method_policy.go

// AuthMethodPolicy defines which auth methods are allowed for a context.
type AuthMethodPolicy struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    Name            string              // "Finance passkey required"
    Description     string

    // Scope: who does this policy apply to?
    UserGroups      []string            // ["finance", "admin"]
    UserIDs         []uuid.UUID         // specific users
    Applications    []string            // OAuth client IDs (empty = all)

    // Rules: what auth methods are required/forbidden?
    RequireMethods  []string            // ["passkey", "webauthn_platform"]
    AllowMethods    []string            // ["passkey", "totp", "email_otp"]
    ForbidMethods   []string            // ["password", "sms_otp"]

    // Phased enforcement
    EnforceAfter    *time.Time          // Policy is advisory until this date, then mandatory
    GracePeriodEnd  *time.Time          // Users get a warning during grace period

    // Conditions
    MinAAL          int                 // Minimum Authenticator Assurance Level (1, 2, 3)
    RequirePhishingResistant bool       // Require WebAuthn-based methods only

    // Settings
    Priority        int                 // Higher = evaluated first
    Enabled         bool
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// Evaluate checks a user's auth methods against the policy.
func (s *PolicyService) Evaluate(
    ctx context.Context,
    tenantID uuid.UUID,
    userID uuid.UUID,
    attemptedMethod string,
    context *PolicyContext,
) (*PolicyResult, error) {
    // 1. Get applicable policies
    policies := s.getApplicablePolicies(ctx, tenantID, userID, context)

    // 2. Evaluate by priority
    for _, policy := range policies {
        if policy.isEnforced(context.Time) {
            if policy.forbids(attemptedMethod) {
                return &PolicyResult{
                    Allowed: false,
                    Reason:  fmt.Sprintf("method %s forbidden by policy %s", attemptedMethod, policy.Name),
                    RequiredMethods: policy.RequireMethods,
                }, nil
            }
            if !policy.allows(attemptedMethod) {
                return &PolicyResult{
                    Allowed: false,
                    Reason:  fmt.Sprintf("method %s not in allowed list for policy %s", attemptedMethod, policy.Name),
                    RequiredMethods: policy.RequireMethods,
                }, nil
            }
        }
    }

    return &PolicyResult{Allowed: true}, nil
}
```

### Policy Evaluation Flow

```
User attempts login with password
         │
         ▼
┌─────────────────────────┐
│ AuthMethodPolicy Engine │
│                         │
│ 1. Get user's groups    │
│ 2. Get applicable rules │
│ 3. Check enforce date   │
│ 4. Evaluate:            │
│    - Is password        │
│      forbidden?         │
│    - Is passkey         │
│      required?          │
│    - Is AAL sufficient? │
└────────┬────────────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
 ALLOW     DENY + redirect to
 login    passkey enrollment
```

---

## 9. Enrollment Nudge System

### Design

A configurable notification/banner system that encourages users to enroll passkeys:

```go
// services/auth/internal/service/enrollment_nudge.go

type NudgeConfig struct {
    TenantID        uuid.UUID
    Enabled         bool
    BannerText      string              // "Go passwordless with passkeys"
    BannerStyle     string              // "info", "prominent", "urgent"
    ActionURL       string              // "/settings/security/passkey/enroll"
    Dismissable     bool                // Can user dismiss?
    DismissDays     int                 // Re-show after N days if dismissed
    Trigger         NudgeTrigger        // When to show
    Segment         NudgeSegment        // Who to show to
    StartDate       *time.Time
    EndDate         *time.Time
}

type NudgeTrigger string
const (
    NudgeAfterLogin     NudgeTrigger = "after_login"      // Show on dashboard after login
    NudgeDuringLogin    NudgeTrigger = "during_login"      // Show during login flow
    NudgePeriodic       NudgeTrigger = "periodic"          // Show every N days
    NudgeOnAccess       NudgeTrigger = "on_sensitive_access" // Show before accessing sensitive resources
)

type NudgeSegment string
const (
    NudgeAllUsers       NudgeSegment = "all"
    NudgeNoPasskey      NudgeSegment = "no_passkey"       // Only users without passkeys
    NudgePasswordUsers  NudgeSegment = "password_only"     // Users who last logged in with password
    NudgeGroupMembers   NudgeSegment = "group"             // Specific group
)
```

### Post-Login Nudge Check

```go
// During login response, check if nudge should be shown
func (s *AuthService) checkEnrollmentNudge(ctx context.Context, userID uuid.UUID) *NudgeResult {
    // 1. Does user already have a passkey?
    credentials, _ := s.webauthnService.GetCredentials(ctx, userID)
    hasPasskey := len(credentials) > 0
    if hasPasskey {
        return nil // No nudge needed
    }

    // 2. Get nudge config
    config := s.getNudgeConfig(ctx, tenantID)
    if !config.Enabled {
        return nil
    }

    // 3. Check segment applicability
    if !s.userMatchesSegment(ctx, userID, config.Segment) {
        return nil
    }

    // 4. Check dismiss history
    if s.isDismissed(ctx, userID, config.DismissDays) {
        return nil
    }

    // 5. Return nudge for frontend to display
    return &NudgeResult{
        BannerText:  config.BannerText,
        ActionURL:   config.ActionURL,
        Dismissable: config.Dismissable,
    }
}
```

---

## 10. Password Deprecation Enforcement

### Design

Once users have enrolled passkeys, the system can progressively disable password-based login:

```go
// services/auth/internal/service/password_deprecation.go

type DeprecationLevel string
const (
    DeprecationOff        DeprecationLevel = "off"          // Password fully enabled
    DeprecationWarn       DeprecationLevel = "warn"          // Show warning, still allow
    DeprecationSecondary  DeprecationLevel = "secondary"     // Password UI hidden, still works via API
    DeprecationDisabled   DeprecationLevel = "disabled"      // Password rejected entirely
)

type PasswordDeprecationPolicy struct {
    TenantID                    uuid.UUID
    Level                       DeprecationLevel
    ApplyToUsersWithPasskey     bool    // Only deprecate for users who have a passkey
    ApplyToGroups               []string
    EffectiveDate               *time.Time
    WarningPeriodDays           int     // Days of warning before full disable
}
```

### Enforcement in Login Flow

```go
// In AuthService.Login(), after password verification succeeds:
func (s *AuthService) checkDeprecationPolicy(ctx context.Context, userID uuid.UUID) error {
    policy := s.getDeprecationPolicy(ctx, tenantID)
    if policy.Level == DeprecationOff {
        return nil
    }

    if policy.ApplyToUsersWithPasskey {
        hasPasskey, _ := s.webauthnService.HasCredential(ctx, userID)
        if !hasPasskey {
            return nil // User has no passkey, password still allowed
        }
    }

    if policy.Level == DeprecationDisabled {
        return ErrPasswordDeprecated // Reject password login
    }

    if policy.Level == DeprecationWarn {
        // Log warning, add response header, but allow login
        s.logDeprecationWarning(ctx, userID)
    }

    return nil
}
```

---

## 11. Account Recovery for Passwordless

### The Recovery Paradox

> You cannot reset a private key. When a user loses their passkey device, they need a recovery path that doesn't rely on the lost credential.

### Recovery Design

```
User loses device with passkey
         │
         ▼
POST /api/v1/auth/recovery/initiate
  { email, tenant_id }
         │
         ▼
┌─────────────────────────────────┐
│ Step 1: Identity Verification   │
│                                 │
│ Options (at least 2 required):  │
│  ☐ Email OTP                    │
│  ☐ TOTP backup (if enrolled)    │
│  ☐ Backup codes (if generated)  │
│  ☐ Admin approval               │
│  ☐ Identity proofing (KYC)      │
│  ☐ Another device's passkey     │
└────────┬────────────────────────┘
         │ (verification passes)
         ▼
┌─────────────────────────────────┐
│ Step 2: Temporary Access Pass   │
│                                 │
│ - Short-lived (15 min)          │
│ - Single-use                    │
│ - Bound to IP + user agent      │
│ - Requires new passkey enroll   │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│ Step 3: New Passkey Enrollment  │
│                                 │
│ - User registers new passkey    │
│ - Old passkey marked revoked    │
│ - Audit event recorded          │
│ - Optional: admin notification  │
└─────────────────────────────────┘
```

### Temporary Access Pass (TAP)

Similar to Microsoft Entra's Temporary Access Pass:

```go
type TemporaryAccessPass struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    UserID      uuid.UUID
    PassCode    string        // 8-digit alphanumeric
    ExpiresAt   time.Time     // 15-minute default
    UsedAt      *time.Time
    UsedFromIP  string
    CreatedBy   uuid.UUID     // Admin who approved, or system
    Reason      string        // "Device loss", "Account recovery"
}
```

---

## 12. Database Schema

```sql
-- Authentication method policies
CREATE TABLE auth_method_policies (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    name                        VARCHAR(128) NOT NULL,
    description                 TEXT,

    -- Scope
    user_groups                 JSONB DEFAULT '[]',           -- ["finance", "admin"]
    user_ids                    JSONB DEFAULT '[]',           -- specific user UUIDs
    applications                JSONB DEFAULT '[]',           -- OAuth client IDs

    -- Rules
    require_methods             JSONB DEFAULT '[]',           -- ["passkey"]
    allow_methods               JSONB DEFAULT '[]',           -- ["passkey", "totp"]
    forbid_methods              JSONB DEFAULT '[]',           -- ["password"]
    min_aal                     INT DEFAULT 1,                -- 1, 2, or 3
    require_phishing_resistant  BOOLEAN DEFAULT false,

    -- Phased enforcement
    enforce_after               TIMESTAMPTZ,                  -- Advisory until this date
    grace_period_end            TIMESTAMPTZ,                  -- Warning during grace period

    -- Config
    priority                    INT DEFAULT 0,
    enabled                     BOOLEAN DEFAULT true,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Password deprecation policies
CREATE TABLE password_deprecation_policies (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL UNIQUE,
    level                       VARCHAR(32) NOT NULL,         -- 'off', 'warn', 'secondary', 'disabled'
    apply_to_users_with_passkey BOOLEAN DEFAULT true,
    apply_to_groups             JSONB DEFAULT '[]',
    effective_date              TIMESTAMPTZ,
    warning_period_days         INT DEFAULT 14,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enrollment nudge configuration
CREATE TABLE enrollment_nudge_configs (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    enabled                     BOOLEAN DEFAULT true,
    banner_text                 VARCHAR(512),
    banner_style                VARCHAR(32) DEFAULT 'info',   -- 'info', 'prominent', 'urgent'
    action_url                  VARCHAR(512),
    dismissable                 BOOLEAN DEFAULT true,
    dismiss_days                INT DEFAULT 7,
    trigger                     VARCHAR(64) DEFAULT 'after_login',
    segment                     VARCHAR(64) DEFAULT 'no_passkey',
    segment_groups              JSONB DEFAULT '[]',
    start_date                  TIMESTAMPTZ,
    end_date                    TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- Nudge dismissals (per user)
CREATE TABLE enrollment_nudge_dismissals (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    user_id                     UUID NOT NULL,
    nudge_config_id             UUID NOT NULL REFERENCES enrollment_nudge_configs(id),
    dismissed_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, user_id, nudge_config_id)
);

-- Temporary access passes (for passwordless recovery)
CREATE TABLE temporary_access_passes (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    user_id                     UUID NOT NULL,
    pass_code_hash              VARCHAR(256) NOT NULL,        -- bcrypt hash
    expires_at                  TIMESTAMPTZ NOT NULL,
    used_at                     TIMESTAMPTZ,
    used_from_ip                VARCHAR(45),
    used_from_ua                TEXT,
    created_by                  UUID NOT NULL,
    created_by_name             VARCHAR(256),
    reason                      VARCHAR(256),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Passkey profiles (authenticator restrictions)
CREATE TABLE passkey_profiles (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    name                        VARCHAR(128) NOT NULL,
    description                 TEXT,
    allowed_aaguuids            JSONB DEFAULT '[]',           -- AAGUID allow-list
    forbidden_aaguuids          JSONB DEFAULT '[]',
    require_attestation         BOOLEAN DEFAULT false,
    min_attestation_level       VARCHAR(32),                  -- 'none', 'self', 'basic', 'attca'
    user_verification           VARCHAR(32) DEFAULT 'preferred', -- 'required', 'preferred', 'discouraged'
    allowed_attachment          VARCHAR(32) DEFAULT 'all',    -- 'platform', 'cross-platform', 'all'
    apply_to_groups             JSONB DEFAULT '[]',
    enabled                     BOOLEAN DEFAULT true,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Auth method enrollment tracking (for metrics)
CREATE TABLE auth_method_enrollments (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    user_id                     UUID NOT NULL,
    method                      VARCHAR(32) NOT NULL,         -- 'passkey', 'totp', 'email_otp', 'sms_otp'
    method_detail               JSONB,                        -- platform, aaguid, authenticator name
    is_primary                  BOOLEAN DEFAULT false,
    enrolled_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at                TIMESTAMPTZ,
    revoked_at                  TIMESTAMPTZ,

    UNIQUE(tenant_id, user_id, method, method_detail)
);

-- Auth method usage events (for metrics/analytics)
CREATE TABLE auth_method_usage_events (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL,
    user_id                     UUID NOT NULL,
    method                      VARCHAR(32) NOT NULL,
    result                      VARCHAR(16) NOT NULL,         -- 'success', 'failure'
    ip_address                  VARCHAR(45),
    user_agent                  TEXT,
    aal                         INT,                          -- Assessed AAL after auth
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_auth_method_policies_tenant ON auth_method_policies (tenant_id, enabled, priority);
CREATE INDEX idx_deprecation_tenant ON password_deprecation_policies (tenant_id);
CREATE INDEX idx_nudge_dismissals_user ON enrollment_nudge_dismissals (tenant_id, user_id);
CREATE INDEX idx_temp_pass_user ON temporary_access_passes (tenant_id, user_id, expires_at);
CREATE INDEX idx_passkey_profiles_tenant ON passkey_profiles (tenant_id, enabled);
CREATE INDEX idx_enrollments_user ON auth_method_enrollments (tenant_id, user_id, method);
CREATE INDEX idx_enrollments_method ON auth_method_enrollments (tenant_id, method, enrolled_at);
CREATE INDEX idx_usage_events_tenant_date ON auth_method_usage_events (tenant_id, created_at);
CREATE INDEX idx_usage_events_user ON auth_method_usage_events (tenant_id, user_id, created_at);
```

---

## 13. API Design

### Auth Method Policy Management

```
# Create policy
POST /api/v1/auth/method-policies
{
    "name": "Finance passkey required",
    "description": "Finance team must use passkeys after Oct 1, 2026",
    "user_groups": ["finance"],
    "require_methods": ["passkey"],
    "allow_methods": ["passkey", "totp"],
    "forbid_methods": ["password"],
    "min_aal": 2,
    "require_phishing_resistant": true,
    "enforce_after": "2026-10-01T00:00:00Z",
    "grace_period_end": "2026-10-15T00:00:00Z",
    "priority": 10
}

# List policies
GET /api/v1/auth/method-policies?tenant_id={tenant}

# Evaluate policy for a user (preview/check)
POST /api/v1/auth/method-policies/evaluate
{
    "user_id": "uuid",
    "attempted_method": "password",
    "context": { "ip": "10.0.1.50", "application": "web-app" }
}

Response:
{
    "allowed": false,
    "reason": "Method 'password' forbidden by policy 'Finance passkey required'",
    "required_methods": ["passkey"],
    "redirect_url": "/auth/passkey/enroll"
}
```

### Password Deprecation

```
# Get current deprecation policy
GET /api/v1/auth/password-deprecation?tenant_id={tenant}

Response:
{
    "level": "warn",
    "apply_to_users_with_passkey": true,
    "effective_date": "2026-08-01T00:00:00Z",
    "warning_period_days": 14
}

# Update deprecation policy
PUT /api/v1/auth/password-deprecation
{
    "level": "disabled",
    "apply_to_users_with_passkey": true,
    "effective_date": "2026-10-01T00:00:00Z",
    "warning_period_days": 30
}
```

### Enrollment Nudge

```
# Configure nudge
PUT /api/v1/auth/enrollment-nudge
{
    "enabled": true,
    "banner_text": "Secure your account with passkeys — faster and phishing-proof",
    "banner_style": "prominent",
    "action_url": "/settings/security/passkey/enroll",
    "trigger": "after_login",
    "segment": "no_passkey"
}

# Get nudge state for current user (called by frontend after login)
GET /api/v1/auth/enrollment-nudge/status

Response:
{
    "show_nudge": true,
    "banner_text": "Secure your account with passkeys",
    "banner_style": "prominent",
    "action_url": "/settings/security/passkey/enroll",
    "dismissable": true,
    "user_has_passkey": false,
    "enrollment_url": "/api/v1/auth/webauthn/register/begin"
}

# Dismiss nudge
POST /api/v1/auth/enrollment-nudge/dismiss
```

### Temporary Access Pass

```
# Admin creates TAP for user
POST /api/v1/auth/recovery/temporary-access-pass
{
    "user_id": "uuid",
    "reason": "Device loss - account recovery",
    "ttl_minutes": 15
}

Response:
{
    "pass_code": "ABCD-1234-EFGH",
    "expires_at": "2026-07-17T11:15:00Z",
    "usage_url": "/auth/recovery?tap=ABCD-1234-EFGH"
}

# User uses TAP to authenticate
POST /api/v1/auth/recovery/use-tap
{
    "pass_code": "ABCD-1234-EFGH",
    "tenant_id": "uuid"
}

Response:
{
    "access_token": "...",
    "must_enroll_passkey": true,
    "enroll_url": "/api/v1/auth/webauthn/register/begin"
}
```

### User Auth Methods

```
# Get all enrolled auth methods for a user
GET /api/v1/auth/methods?user_id={uuid}

Response:
{
    "methods": [
        {
            "method": "passkey",
            "detail": { "platform": "macOS", "aaguid": "...", "name": "MacBook Pro Touch ID" },
            "is_primary": true,
            "enrolled_at": "2026-06-01T10:00:00Z",
            "last_used_at": "2026-07-17T09:00:00Z"
        },
        {
            "method": "totp",
            "detail": { "name": "Google Authenticator" },
            "is_primary": false,
            "enrolled_at": "2026-01-15T10:00:00Z"
        }
    ],
    "has_passkey": true,
    "has_password": true,
    "current_aal": 2
}
```

---

## 14. Metrics and Analytics

### Migration Dashboard Metrics

| Metric | Description | Formula |
|--------|-------------|---------|
| **Passkey enrollment rate** | % of users with at least one passkey | `users_with_passkey / total_users * 100` |
| **Weekly enrollment trend** | New passkey enrollments per week | `COUNT(enrollments WHERE method='passkey' AND week=current)` |
| **Password usage rate** | % of logins using password (vs passkey) | `password_logins / total_logins * 100` |
| **Phishing-resistant rate** | % of users at AAL2+ | `users_at_aal2plus / total_users * 100` |
| **Helpdesk ticket reduction** | Password reset tickets before/after | `current_resets / baseline_resets` |
| **Enrollment funnel** | Started → Completed enrollment | `completed / started * 100` |
| **Recovery usage** | TAP usage per week | `COUNT(temporary_access_passes WHERE week=current)` |
| **Deprecation impact** | Users blocked by deprecation policy | `COUNT(blocked_login_attempts)` |

### API

```
# Get migration metrics
GET /api/v1/auth/passwordless/metrics?period=30d

Response:
{
    "total_users": 5000,
    "users_with_passkey": 3200,
    "enrollment_rate": 64.0,
    "enrollment_rate_trend": "+8.5%",
    "weekly_enrollments": [45, 62, 78, 95, 110, 125, 140],
    "password_login_rate": 36.0,
    "password_login_rate_trend": "-12.3%",
    "aal_distribution": {
        "aal1": 1800,    // Password only
        "aal2": 3100,    // Password + MFA or passkey
        "aal3": 100      // Hardware key
    },
    "helpdesk_reset_tickets": {
        "current_month": 42,
        "previous_month": 68,
        "reduction": 38.2
    },
    "recovery_usage": {
        "this_week": 3,
        "this_month": 12
    }
}
```

---

## 15. Console UI Design

### Passwordless Migration Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  Passwordless Migration                                          │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │  Enrollment    │  │  Password      │  │  Phishing-     │     │
│  │  Rate          │  │  Usage         │  │  Resistant     │     │
│  │  64% ↑8.5%    │  │  36% ↓12.3%   │  │  64%           │     │
│  │  3,200 / 5,000 │  │  of logins     │  │  of users      │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│                                                                  │
│  Enrollment Trend (30 days)                                      │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │     ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇▆                             │  │
│  │ Week 1      Week 2      Week 3      Week 4                │  │
│  │ +45/wk      +78/wk      +110/wk     +140/wk               │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  AAL Distribution                                                │
│  AAL3 ████░░░░░░░░░░░░░░░░░░░░░  100 (2%)                       │
│  AAL2 ████████████████████████░  3,100 (62%)                    │
│  AAL1 ████████░░░░░░░░░░░░░░░░  1,800 (36%)                    │
│                                                                  │
│  Policies                                                        │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ ● Finance passkey required    Active   Enforce: Oct 1     │  │
│  │ ● Admin AAL3 required         Active   Enforce: Now       │  │
│  │ ● Password deprecation        Warning  Since: Aug 1       │  │
│  │ + Add Policy                                                │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Enrollment Nudge                                                │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ Banner: "Secure your account with passkeys"     [Edit]    │  │
│  │ Segment: Users without passkey                              │  │
│  │ Trigger: After login    Style: Prominent                    │  │
│  │ Dismissals this week: 23 (of 1,800 unenrolled)             │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 16. Security Considerations

### Recovery Security

| Risk | Mitigation |
|------|-----------|
| Social engineering of helpdesk for TAP | Require multi-factor identity verification before issuing TAP |
| TAP brute-force | Short TTL (15 min), single-use, rate-limited, bcrypt-hashed |
| Passkey enrollment during recovery | Require new biometric enrollment ceremony, log device info |
| Admin TAP abuse | All TAPs audited, dual approval for privileged accounts |

### Policy Enforcement Security

| Risk | Mitigation |
|------|-----------|
| Policy bypass via direct API | Policy checked in auth service (server-side), not just frontend |
| Race condition during policy change | Policies are versioned; active version determined atomically |
| Deprecation lockout | Grace period mandatory; TAP always available as fallback |
| AAL spoofing | AAL computed server-side from actual auth method used, not client-supplied |

---

## 17. Competitive Differentiation

| Feature | GGID (target) | Okta | Microsoft Entra | Auth0 | Keycloak |
|---------|---------------|------|-----------------|-------|----------|
| **Auth method policy engine** | **Declarative YAML + API** | App sign-in policy | Conditional Access | Actions (JS) | Auth flow config |
| **Enrollment nudging** | **Configurable banners** | Yes (dashboard) | Registration Campaigns | Custom | No |
| **Password deprecation** | **4 levels** (off→disabled) | Yes | Yes (disable) | Custom | No |
| **Temporary Access Pass** | **Yes** | Yes | Yes | Custom | No |
| **Passkey profiles (AAGUID)** | **Yes** | Yes | Yes | Yes | Partial |
| **Migration metrics** | **Built-in dashboard** | Yes | Yes | Custom | No |
| **Open source** | **Yes (Apache 2.0)** | No | No | No | Yes |

**Key differentiator**: GGID would be the only **open-source IAM** with a complete passwordless migration toolkit — policy engine, enrollment nudging, password deprecation, recovery passes, and metrics dashboard.

---

## 18. Implementation Backlog

### P0 — Core Policy Engine (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | Auth method policy data model | PostgreSQL tables for policies, deprecation, nudge configs | 2 days |
| 2 | Auth method policy service | CRUD + evaluation engine | 4 days |
| 3 | Login flow integration | Policy check in AuthService.Login() | 2 days |
| 4 | Password deprecation enforcement | Block/warn password based on policy | 2 days |
| 5 | Enrollment nudge system | Post-login nudge check + dismiss tracking | 3 days |
| 6 | Policy management API | CRUD endpoints for policies, deprecation, nudges | 3 days |
| 7 | Auth methods enrollment tracking | Track enrolled methods per user | 2 days |
| 8 | Unit tests | 90%+ coverage | 3 days |

### P1 — Recovery & Metrics (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 9 | Temporary Access Pass | Admin-issued recovery pass with TTL, single-use | 4 days |
| 10 | Recovery flow | Multi-step identity verification + TAP + new enrollment | 3 days |
| 11 | Passkey profiles (AAGUID) | Authenticator allow-list enforcement | 3 days |
| 12 | Migration metrics API | Aggregated enrollment/usage/AAL metrics | 3 days |
| 13 | Usage event logging | Log auth method used per login (for analytics) | 2 days |
| 14 | Integration tests | End-to-end policy enforcement + recovery | 3 days |

### P2 — Console UI (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 15 | Migration dashboard | Enrollment rate, trend chart, AAL distribution | 4 days |
| 16 | Policy editor | Create/edit auth method policies with live preview | 3 days |
| 17 | Deprecation controls | Toggle password deprecation level with schedule | 2 days |
| 18 | Nudge configuration | Banner text editor, segment selector, trigger config | 2 days |
| 19 | Recovery console | Issue TAP, view recovery history | 2 days |
| 20 | User auth methods view | Per-user enrolled methods list | 2 days |

### P3 — Advanced Features (Future)

| # | Task | Description |
|---|------|-------------|
| 21 | AAL-based conditional access | Gateway middleware: require AAL2 for sensitive routes |
| 22 | Progressive enrollment | Enforce enrollment during sensitive access (JIT enrollment) |
| 23 | Migration simulator | Simulate full migration rollout with synthetic data |
| 24 | Backup codes | Generate single-use backup codes for passwordless recovery |
| 25 | Shared workstation pattern | Roaming security key support for kiosk/shared device environments |
| 26 | Cross-device passkey transfer | FIDO Credential Exchange Protocol support |
| 27 | Password purge | Automated password hash deletion after sunset period |

---

## References

- [FIDO Alliance 2026 Passkey Adoption Report](https://fidoalliance.org/) — 5B passkeys, 87% enterprise adoption
- [NIST SP 800-63B](https://pages.nist.gov/800-63-3/sp800-63b.html) — Authenticator Assurance Levels (AAL1/2/3)
- [NIST SP 800-63B Supplement 1](https://pages.nist.gov/800-63-3/sp800-63b.html) — Synced passkeys recognized at AAL2
- [WebAuthn Level 3 (W3C)](https://www.w3.org/TR/webauthn-3/) — Web Authentication API specification
- [CTAP 2.1 (FIDO Alliance)](https://fidoalliance.org/specs/fido-v2.1-rd-20191017/fido-client-to-authenticator-protocol-v2.1-rd-20191017.html) — Client to Authenticator Protocol
- [Okta App Sign-In Policies](https://help.okta.com/oie/en-us/content/topics/identity-engine/policies/app-signin-policy.htm) — Passwordless policy configuration
- [Microsoft Entra Registration Campaigns](https://learn.microsoft.com/en-us/entra/identity/authentication/how-to-plan-prerequisites-phishing-resistant-passwordless-authentication) — Phishing-resistant deployment guide
- [Microsoft Temporary Access Pass](https://learn.microsoft.com/en-us/entra/identity/authentication/how-to-authentication-temporary-access-pass) — Recovery pass design
- [FIDO Metadata Service](https://fidoalliance.org/metadata/) — AAGUID attestation database
- [Enterprise Passwordless Migration (KKRF)](https://kkrfgroup.com/enterprise-passwordless-authentication/) — 8-12 week phased playbook
- [GGID WebAuthn Implementation](../guides/webauthn-deep-dive.md) — Existing WebAuthn/attestation support
- [GGID Password Policy](../password-policy.md) — Current password policy configuration
- [GGID Passwordless Auth](./passwordless-auth-iam.md) — Magic link, OTP, WebAuthn implementation details
