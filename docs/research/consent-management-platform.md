# Consent Management Platform: GDPR/CCPA-Compliant Consent Lifecycle for GGID

> **Focus**: A production-grade consent management platform — replacing hardcoded mock data and in-memory OAuth consent with a DB-backed, GDPR/CCPA-compliant system supporting consent grants, purpose-based scoping, withdrawal cascading, DSR (Data Subject Request) integration, preference centers, and audit trails.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§12), curl commands (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Regulatory Requirements](#2-regulatory-requirements)
3. [GGID Current State Analysis](#3-ggid-current-state-analysis)
4. [Gap Analysis](#4-gap-analysis)
5. [Proposed Architecture](#5-proposed-architecture)
6. [Consent Lifecycle](#6-consent-lifecycle)
7. [Endpoint Precondition Check](#7-endpoint-precondition-check)
8. [API Design + Curl Commands](#8-api-design--curl-commands)
9. [Database Schema](#9-database-schema)
10. [DSR Integration](#10-dsr-integration)
11. [Console UI Requirements](#11-console-ui-requirements)
12. [Implementation Backlog with DoD](#12-implementation-backlog-with-dod)
13. [Competitive Differentiation](#13-competitive-differentiation)
14. [Security Considerations](#14-security-considerations)

---

## 1. Executive Summary

Consent management is a legal requirement under GDPR (EU), CCPA (California), PIPL (China), LGPD (Brazil), and 20+ other privacy regulations. Organizations must track **what data** they collect, **for what purpose**, **who consented**, **when**, and **how to withdraw** — with full audit trails for regulatory compliance.

GGID has two consent-related components, both with critical gaps:

1. **Identity consent registry** (`identity/server/consent_registry_handler.go:26`) — **Hardcoded mock data**. Returns 4 fake consent records. No DB, no CRUD, no real data. Pure placeholder.

2. **OAuth consent store** (`oauth/service/consent.go:35`) — **In-memory** (`map[string]*ConsentRecord` + `sync.RWMutex`). Violates acceptance checklist. Consent decisions lost on restart.

Additionally, GGID has **14 consent-related API endpoints** scattered across services (consent screen, dashboard, analytics, config, history, receipt, admin override, RAR preview) — most returning hardcoded or mock data.

**Recommendation**: Build a unified consent management platform with:
- PostgreSQL-backed consent store replacing both mock and in-memory
- Purpose-based consent model (marketing, analytics, third-party sharing, profiling)
- GDPR Article 7 compliance (freely given, specific, informed, withdrawable)
- GDPR Article 17 integration (consent withdrawal cascades to data deletion)
- CCPA opt-out signal support (`Global Privacy Control` header)
- DSR (Data Subject Request) workflow integration
- Preference center UI + cookie consent banner
- Bulk consent import/export for compliance reporting

**Estimated effort**: 4 sprints for MVP (DB + API + DSR + Console UI).

---

## 2. Regulatory Requirements

### GDPR (General Data Protection Regulation)

| Article | Requirement | GGID Implementation |
|---------|-------------|---------------------|
| **Art. 6** | Lawful basis for processing | Consent must be explicit, recorded |
| **Art. 7** | Conditions for consent | Freely given, specific, informed, unambiguous, withdrawable |
| **Art. 7(3)** | Right to withdraw consent | As easy to withdraw as to give |
| **Art. 9** | Special category data | Explicit consent for health, biometrics, etc. |
| **Art. 13-14** | Information to data subject | Privacy notice linked to consent |
| **Art. 17** | Right to erasure ("right to be forgotten") | Consent withdrawal triggers data deletion workflow |
| **Art. 20** | Right to data portability | Export consent records + associated data |
| **Art. 25** | Privacy by design | Consent is default-off (opt-in) |
| **Art. 28** | Processor obligations | Consent records track data processors |
| **Art. 33** | Breach notification (72h) | Consent records needed to identify affected users |

### CCPA (California Consumer Privacy Act)

| Right | Requirement | GGID Implementation |
|-------|-------------|---------------------|
| **Right to know** | What personal info collected and why | Consent records list purposes + data categories |
| **Right to delete** | Delete personal info | DSR deletion workflow triggered by consent withdrawal |
| **Right to opt-out** | Stop sale/sharing of personal info | Consent withdrawal for "third_party_share" purpose |
| **Right to non-discrimination** | No penalty for exercising rights | Users who opt out get same service |
| **Global Privacy Control** | Browser-level opt-out signal | Gateway detects `Sec-GPC: 1` header |

### Other Regulations

| Regulation | Region | Key Consent Requirement |
|-----------|--------|------------------------|
| **PIPL** | China | Separate consent for sensitive data, cross-border transfer |
| **LGPD** | Brazil | 9 specific legal bases; consent must be written or electronic |
| **PDPA** | Singapore | Deemed consent for notification; withdrawal must be honored |
| **APPI** | Japan | Opt-out allowed for some data; sensitive data needs consent |

---

## 3. GGID Current State Analysis

### Existing Consent Components

| Component | File:Line | Status | Issue |
|-----------|-----------|--------|-------|
| Consent registry (GDPR) | `identity/server/consent_registry_handler.go:26` | **Hardcoded mock** ❌ | 4 fake records, no DB |
| OAuth consent store | `oauth/service/consent.go:35` | **In-memory** ❌ | `map[string]*ConsentRecord` + sync.RWMutex |
| OAuth consent screen | `oauth/server/consent_screen_handler.go:24` | **Implemented** | OIDC consent screen for authorization flow |
| Consent dashboard | `oauth/server/consent_dashboard_handler.go:24` | **Hardcoded** ❌ | Mock statistics |
| Consent analytics | `oauth/server/consent_analytics_handler.go:9` | **Hardcoded** ❌ | Mock data |
| Consent config | `oauth/server/consent_config_handler.go:33` | **Hardcoded** ❌ | Mock config |
| Consent history | `oauth/server/consents_history_handler.go:35` | **Hardcoded** ❌ | Mock history |
| Consent receipt | `oauth/server/consent_receipt.go:53` | **Implemented** | Receipt PDF generation |
| Admin override | `oauth/server/consent_override_handler.go:29` | **Implemented** | Admin consent override |
| Consent management service | `oauth/service/consent_management.go:12` | **Implemented** | Consent grant tracking |
| Auth consent handler | `auth/server/missing_handlers.go:207` | **Placeholder** | Stub handler |
| Identity consent registry route | `identity/server/http.go:176` | **Wired** | Route exists but handler returns mock |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **Hardcoded consent data** | Registry returns fake records — not usable |
| 2 | **In-memory OAuth consent** | Consent decisions lost on service restart |
| 3 | **No GDPR consent model** | No purpose-based consent (marketing/analytics/sharing) |
| 4 | **No withdrawal cascade** | Withdrawing consent doesn't trigger data operations |
| 5 | **No DSR integration** | No connection between consent and data subject requests |
| 6 | **No CCPA/GPC support** | No browser-level opt-out detection |
| 7 | **No preference center** | Users can't self-manage consent preferences |
| 8 | **No cookie consent** | No cookie consent banner integration |
| 9 | **No consent expiry enforcement** | Expired consents still active |
| 10 | **No consent versioning** | Privacy policy changes don't invalidate old consents |

---

## 4. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "User withdraws marketing consent" | No API | POST /consent/withdraw → cascade to unsubscribe |
| 2 | "Show all active consents for a user" | Hardcoded mock | DB query returns real active consents |
| 3 | "User sends GPC opt-out header" | Ignored | Auto-withdraw all sale/sharing consent |
| 4 | "Regulator audits consent records" | Can't produce | DB-backed audit trail with timestamps |
| 5 | "Privacy policy updated — re-consent needed" | No mechanism | Version bump invalidates old consents |
| 6 | "User requests data deletion (GDPR Art. 17)" | Manual | Consent withdrawal → DSR deletion workflow |

---

## 5. Proposed Architecture

```
                    ┌──────────────────────────────────────────────┐
                    │       Consent Management Platform             │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Consent Store (PostgreSQL)           │    │
                    │  │                                      │    │
                    │  │  Tables:                             │    │
                    │  │  ├── consent_records (grants)        │    │
                    │  │  ├── consent_purposes (catalog)      │    │
                    │  │  ├── consent_policies (versions)     │    │
                    │  │  ├── consent_withdrawal_log (audit)  │    │
                    │  │  └── dsr_requests (data subject)     │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Consent Lifecycle Engine             │    │
                    │  │                                      │    │
                    │  │  Grant → Manage → Withdraw → Audit   │    │
                    │  │     ↓               ↓                │    │
                    │  │  Record consent   Cascade effects:   │    │
                    │  │  + policy version - Unsubscribe      │    │
                    │  │  + purpose          - Data deletion  │    │
                    │  │  + expiry           - Token revoke   │    │
                    │  │                     - Notify 3rdparty│    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────┐  ┌─────────────────┐   │
                    │  │  DSR Workflow     │  │  GPC Detector   │   │
                    │  │  (Art. 17/20)     │  │  (CCPA opt-out) │   │
                    │  └────────┬─────────┘  └────────┬────────┘   │
                    │           │                     │            │
                    │  ┌────────▼─────────────────────▼────────┐   │
                    │  │  Notification + Audit Publisher        │   │
                    │  │  (NATS events → audit service)         │   │
                    │  └───────────────────────────────────────┘   │
                    └──────────────────────────────────────────────┘
```

---

## 6. Consent Lifecycle

```
                    ┌─────────────┐
                    │  No Consent │
                    └──────┬──────┘
                           │ User views privacy notice
                           ▼
                    ┌─────────────┐
                    │  Presented  │  Privacy policy shown
                    └──────┬──────┘
                           │ User clicks "Accept" or "Customize"
                           ▼
                    ┌─────────────┐
                    │   Granted   │  ← Record: purpose, scope, policy_version, timestamp
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │  Active  │ │ Modified │ │ Expired  │
        │  (in use)│ │ (scope   │ │ (TTL     │
        │          │ │  changed)│ │  elapsed)│
        └────┬─────┘ └────┬─────┘ └──────────┘
             │            │
             │     ┌──────┘ (re-grant with new scope)
             │     │
             ▼     ▼
        ┌──────────┐
        │ Withdrawn│  ← User revokes consent
        └────┬─────┘
             │ CASCADE:
             │ ├── Cancel marketing emails
             │ ├── Stop analytics tracking
             │ ├── Revoke OAuth tokens for that purpose
             │ ├── Notify third-party processors
             │ └── If Art. 17: trigger DSR deletion
             ▼
        ┌──────────┐
        │  Audited │  ← Immutable record in consent_withdrawal_log
        └──────────┘
```

---

## 7. Endpoint Precondition Check

### Existing Endpoints (Replace Mock/In-Memory)

| Endpoint | File:Line | Current | Target |
|----------|-----------|---------|--------|
| `GET /api/v1/identity/consent/registry` | `identity/http.go:176` | **Mock** | DB-backed |
| `GET /api/v1/oauth/consents/dashboard` | `oauth/server.go:1601` | **Mock** | DB-backed |
| `GET /api/v1/oauth/consent/analytics` | `oauth/server.go:1937` | **Mock** | DB-backed |
| `GET /api/v1/oauth/consent/config` | `oauth/server.go:1654` | **Mock** | DB-backed |
| `GET /api/v1/oauth/consents/history` | `oauth/server.go:1889` | **Mock** | DB-backed |
| OAuth ConsentStore | `oauth/service/consent.go:35` | **In-memory** | DB-backed |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/consent/records` | GET | List user's consent records | P0 |
| `/api/v1/consent/records` | POST | Grant consent for a purpose | P0 |
| `/api/v1/consent/records/{id}` | GET | Get specific consent record | P0 |
| `/api/v1/consent/records/{id}/withdraw` | POST | Withdraw consent (cascade) | P0 |
| `/api/v1/consent/purposes` | GET | List available consent purposes | P0 |
| `/api/v1/consent/preferences` | GET | Get user's preference summary | P0 |
| `/api/v1/consent/preferences` | PUT | Update preferences (bulk) | P0 |
| `/api/v1/consent/export` | GET | Export consent data (GDPR Art. 20) | P1 |
| `/api/v1/consent/import` | POST | Bulk import consent records | P1 |
| `/api/v1/consent/dsr` | POST | Submit Data Subject Request | P1 |
| `/api/v1/consent/dsr/{id}` | GET | Check DSR status | P1 |
| `/api/v1/consent/dsr/{id}/process` | POST | Admin process DSR | P1 |

---

## 8. API Design + Curl Commands

### List Consent Records

```bash
curl "https://ggid.corp.com/api/v1/consent/records?user_id=$USER_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Response:
{
  "records": [
    {
      "id": "cr_7f3a2b1c-...",
      "user_id": "uuid",
      "purpose": "marketing",
      "purpose_label": "Marketing Emails",
      "scopes": ["email", "profile"],
      "status": "active",
      "policy_version": "2026.01",
      "granted_at": "2026-01-15T10:00:00Z",
      "expires_at": "2027-01-15T10:00:00Z",
      "withdrawal_url": "/api/v1/consent/records/cr_7f3a.../withdraw"
    }
  ],
  "summary": { "active": 3, "expired": 1, "withdrawn": 2 }
}
```

### Grant Consent

```bash
curl -X POST https://ggid.corp.com/api/v1/consent/records \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "purpose": "analytics",
    "scopes": ["behavior", "device_info"],
    "policy_version": "2026.01",
    "expires_in_days": 365
  }'

# Response:
{
  "id": "cr_9e8f7g6h-...",
  "status": "active",
  "granted_at": "2026-07-17T10:00:00Z",
  "expires_at": "2027-07-17T10:00:00Z"
}
```

### Withdraw Consent (with Cascade)

```bash
curl -X POST https://ggid.corp.com/api/v1/consent/records/cr_7f3a2b1c/withdraw \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"reason": "user_request"}'

# Response:
{
  "status": "withdrawn",
  "withdrawn_at": "2026-07-17T11:00:00Z",
  "cascade_effects": [
    { "action": "unsubscribe_email", "target": "marketing", "status": "completed" },
    { "action": "revoke_tokens", "target": "client_abc", "status": "completed" },
    { "action": "notify_processor", "target": "analytics_vendor", "status": "pending" }
  ],
  "audit_id": "aud_5a4b3c2d"
}
```

### Bulk Update Preferences

```bash
curl -X PUT https://ggid.corp.com/api/v1/consent/preferences \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "preferences": [
      { "purpose": "marketing", "granted": true },
      { "purpose": "analytics", "granted": false },
      { "purpose": "third_party_share", "granted": false },
      { "purpose": "profiling", "granted": false }
    ]
  }'
```

### Submit Data Subject Request (DSR)

```bash
curl -X POST https://ggid.corp.com/api/v1/consent/dsr \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "type": "deletion",
    "reason": "GDPR Article 17 - Right to erasure",
    "verify_identity": true
  }'

# Response:
{
  "dsr_id": "dsr_3b2c1d4e-...",
  "status": "pending_verification",
  "verification_required": "email_otp",
  "estimated_completion_days": 30
}
```

### GPC (Global Privacy Control) Detection

```bash
# Browser sends: Sec-GPC: 1
curl https://ggid.corp.com/api/v1/identity/profile \
  -H "Authorization: Bearer $TOKEN" \
  -H "Sec-GPC: 1"

# Gateway detects GPC header → auto-withdraw third_party_share consent
# Response includes GPC status:
{
  "data": { ... },
  "meta": {
    "gpc_detected": true,
    "gpc_action_taken": "auto_withdraw_consent",
    "gpc_affected_purposes": ["third_party_share", "sale_of_data"]
  }
}
```

---

## 9. Database Schema

```sql
-- Consent purposes catalog (admin-configured)
CREATE TABLE consent_purposes (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    key                 VARCHAR(64) NOT NULL,         -- 'marketing', 'analytics', 'third_party_share'
    label               VARCHAR(256) NOT NULL,        -- 'Marketing Emails'
    description         TEXT,                         -- Privacy notice text
    data_categories     JSONB DEFAULT '[]',           -- ["email", "device_info", "location"]
    legal_basis         VARCHAR(32) DEFAULT 'consent', -- 'consent', 'contract', 'legitimate_interest'
    requires_explicit_consent BOOLEAN DEFAULT true,
    default_granted     BOOLEAN DEFAULT false,        -- Privacy by design: opt-in
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, key)
);

-- Consent records (user grants)
CREATE TABLE consent_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    purpose_key         VARCHAR(64) NOT NULL,
    scopes              JSONB DEFAULT '[]',           -- ["email", "profile"]
    status              VARCHAR(16) NOT NULL DEFAULT 'active',
    -- 'active', 'expired', 'withdrawn', 'superseded'

    -- Policy tracking
    policy_version      VARCHAR(16) NOT NULL,         -- Privacy policy version at time of consent
    privacy_notice_url  TEXT,                         -- URL to exact version of notice shown

    -- Timestamps
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,                  -- NULL = no expiry
    withdrawn_at        TIMESTAMPTZ,
    withdrawn_reason    VARCHAR(256),

    -- Grant context
    granted_via         VARCHAR(32) DEFAULT 'preference_center',
    -- 'preference_center', 'oauth_consent', 'cookie_banner', 'api'
    ip_address          VARCHAR(45),
    user_agent          TEXT,

    -- Supersession (when re-granted)
    supersedes_id       UUID REFERENCES consent_records(id),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Consent policy versions (privacy policy change tracking)
CREATE TABLE consent_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    version             VARCHAR(16) NOT NULL,         -- '2026.01'
    title               VARCHAR(256) NOT NULL,
    content_url         TEXT NOT NULL,                -- URL to privacy policy
    summary             TEXT,
    published_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    requires_reconsent  BOOLEAN DEFAULT false,        -- If true, old consents superseded
    UNIQUE(tenant_id, version)
);

-- Consent withdrawal log (immutable audit trail)
CREATE TABLE consent_withdrawal_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    consent_record_id   UUID NOT NULL REFERENCES consent_records(id),
    user_id             UUID NOT NULL,
    purpose_key         VARCHAR(64) NOT NULL,
    withdrawn_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    withdrawn_reason    VARCHAR(256),
    cascade_results     JSONB DEFAULT '[]',           -- [{action, target, status}]
    ip_address          VARCHAR(45),
    user_agent          TEXT
);

-- Data Subject Requests (DSR)
CREATE TABLE dsr_requests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    type                VARCHAR(32) NOT NULL,         -- 'access', 'deletion', 'portability', 'correction', 'opt_out'
    status              VARCHAR(32) NOT NULL DEFAULT 'pending_verification',
    -- 'pending_verification', 'verified', 'processing', 'completed', 'rejected', 'cancelled'

    reason              TEXT,
    legal_basis         VARCHAR(64),                  -- 'GDPR Art. 17', 'CCPA §1798.105'

    -- Verification
    verification_method VARCHAR(32),                  -- 'email_otp', 'identity_proof'
    verified_at         TIMESTAMPTZ,
    verified_by         UUID,

    -- Processing
    processed_by        UUID,
    processed_at        TIMESTAMPTZ,
    completion_data     JSONB,                        -- Export URL, deletion confirmation, etc.

    -- Deadlines (GDPR: 1 month response)
    requested_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deadline_at         TIMESTAMPTZ NOT NULL,         -- requested_at + 30 days
    completed_at        TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- GPC (Global Privacy Control) log
CREATE TABLE gpc_signals (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID,
    ip_address          VARCHAR(45),
    user_agent          TEXT,
    action_taken        VARCHAR(64),                  -- 'auto_withdraw', 'logged'
    affected_purposes   JSONB DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_consent_tenant_user ON consent_records (tenant_id, user_id, status);
CREATE INDEX idx_consent_purpose ON consent_records (tenant_id, purpose_key, status);
CREATE INDEX idx_consent_expiry ON consent_records (tenant_id, expires_at) WHERE status = 'active';
CREATE INDEX idx_consent_supersedes ON consent_records (supersedes_id);
CREATE INDEX idx_withdrawal_tenant_time ON consent_withdrawal_log (tenant_id, withdrawn_at DESC);
CREATE INDEX idx_withdrawal_user ON consent_withdrawal_log (tenant_id, user_id, withdrawn_at DESC);
CREATE INDEX idx_dsr_tenant_status ON dsr_requests (tenant_id, status, deadline_at);
CREATE INDEX idx_dsr_user ON dsr_requests (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_purposes_tenant ON consent_purposes (tenant_id, key);
CREATE INDEX idx_policies_tenant ON consent_policies (tenant_id, version DESC);
CREATE INDEX idx_gpc_tenant_time ON gpc_signals (tenant_id, created_at DESC);
```

---

## 10. DSR Integration

### Data Subject Request Workflow

```
User submits DSR (deletion/portability/access)
         │
         ▼
┌────────────────────────┐
│ 1. Identity Verification │
│    (email OTP / step-up) │
└────────────┬───────────┘
             │ verified
┌────────────▼───────────┐
│ 2. Queue DSR             │
│    (status: processing)  │
└────────────┬───────────┘
             │
┌────────────▼───────────┐
│ 3. Execute DSR type:     │
│    ├── Access: collect   │
│    │   all user data     │
│    ├── Portability:      │
│    │   export JSON       │
│    └── Deletion: cascade │
│        ├── Delete profile│
│        ├── Delete audit  │
│        │   (retain min)  │
│        ├── Revoke tokens │
│        ├── Cancel subs   │
│        └── Notify 3rdpty │
└────────────┬───────────┘
             │
┌────────────▼───────────┐
│ 4. Complete + Notify     │
│    (status: completed)   │
│    Email user with result│
└──────────────────────────┘
```

### DSR Deadlines by Regulation

| Regulation | Response Deadline | Extension |
|-----------|-------------------|-----------|
| GDPR | 1 month | +2 months (with justification) |
| CCPA | 45 days | +45 days (with notice) |
| LGPD | 15 days | None |
| PIPL | "Without delay" | Not specified |

---

## 11. Console UI Requirements

### Preference Center

```
┌──────────────────────────────────────────────────────────────────┐
│  Privacy Preferences                                             │
│                                                                  │
│  Manage how your data is used. You can change these at any time. │
│  Last updated: July 17, 2026                                    │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ ● Marketing Communications          [ON]  ●─               │  │
│  │   Receive product updates and newsletters.                 │  │
│  │   Granted: Jan 15, 2026 · Expires: Jan 15, 2027           │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● Analytics & Tracking             [OFF] ─●                │  │
│  │   Usage analytics to improve our products.                 │  │
│  │   Withdrawn: Mar 10, 2026                                  │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● Third-Party Data Sharing         [OFF] ─●                │  │
│  │   Share data with selected partners.                       │  │
│  │   Never granted                                             │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● Personalized Profiling           [ON]  ●─               │  │
│  │   Tailor content based on your activity.                   │  │
│  │   Granted: Jul 1, 2026                                     │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  [Export My Data] [Delete My Account] [Download Consent Receipt] │
└──────────────────────────────────────────────────────────────────┘
```

### Admin DSR Console

```
┌──────────────────────────────────────────────────────────────────┐
│  Data Subject Requests                                           │
│                                                                  │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐     │
│  │ Pending    │ │ Processing │ │ Completed  │ │ Overdue    │     │
│  │ 3          │ │ 2          │ │ 147        │ │ 1 ⚠       │     │
│  └────────────┘ └────────────┘ └────────────┘ └────────────┘     │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ DSR-0042  Deletion   Alice Chen  3 days left  [Process]   │  │
│  │ DSR-0041  Access     Bob Smith  12 days left  [Process]   │  │
│  │ DSR-0040  Portability Carol Lee  20 days left [Process]   │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 12. Implementation Backlog with DoD

### P0 — DB-Backed Consent Store + API (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Consent DB schema | ✅ CREATE TABLE in migration ✅ go build PASS ✅ No in-memory map | 2d |
| 2 | Consent repository | ✅ CRUD backed by pgx ✅ `if err != nil` guards ✅ ≥3 tests | 3d |
| 3 | Consent API (grant/list/withdraw) | ✅ 7 endpoints registered ✅ From handler to repo ✅ curl test PASS ✅ ≥3 tests | 4d |
| 4 | Replace hardcoded consent registry | ✅ identity/consent_registry uses repo ✅ No mock data ✅ ≥3 tests | 2d |
| 5 | Replace in-memory OAuth consent store | ✅ oauth/service/consent.go uses repo ✅ No sync.RWMutex ✅ ≥3 tests | 3d |
| 6 | Withdrawal cascade engine | ✅ Withdraw triggers cascade (email, tokens, notify) ✅ DB-backed audit ✅ ≥3 tests | 3d |

### P1 — DSR + GPC + Policy Versioning (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | DSR workflow (access/deletion/portability) | ✅ Submit → verify → process → complete ✅ Deadline tracking ✅ ≥3 tests | 4d |
| 8 | GPC detection in gateway | ✅ `Sec-GPC: 1` header detected ✅ Auto-withdraw sale/sharing ✅ ≥3 tests | 2d |
| 9 | Policy versioning + re-consent | ✅ Version bump supersedes old consents ✅ Users prompted for re-consent ✅ ≥3 tests | 3d |
| 10 | Consent analytics (replace mock) | ✅ dashboard/analytics/history use real data ✅ DB-backed ✅ ≥3 tests | 2d |

### P2 — Console UI (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 11 | Preference center UI | ✅ Toggle consents ✅ Granular per-purpose ✅ Receipt download ✅ ≥3 tests | 4d |
| 12 | Admin DSR console | ✅ List/process DSRs ✅ Deadline tracking ✅ Export/download ✅ ≥3 tests | 3d |
| 13 | Cookie consent banner | ✅ Cookie consent widget ✅ Preference categories ✅ GPC detection ✅ ≥3 tests | 2d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 14 | Consent analytics dashboard | Trends: grant rate, withdrawal rate, by purpose |
| 15 | Third-party processor registry | Track data processors with consent status |
| 16 | Consent webhooks | Notify external systems on grant/withdrawal |
| 17 | Multi-language consent notices | Localized privacy policy versions |
| 18 | Consent-based access control | Gate features behind consent status |
| 19 | Children's data protection (COPPA/GDPR-K) | Age-gate + parental consent flow |

---

## 13. Competitive Differentiation

| Feature | GGID (target) | OneTrust | Cookiebot | TrustArc | Auth0 | Okta |
|---------|---------------|---------|-----------|----------|-------|------|
| **Purpose-based consent** | **Yes** | Yes | Yes | Yes | No | No |
| **GDPR Art. 7 (withdraw)** | **Cascade engine** | Yes | Partial | Yes | No | No |
| **GDPR Art. 17 (erasure)** | **DSR workflow** | Module | No | Module | No | No |
| **GDPR Art. 20 (portability)** | **JSON export** | Yes | No | Yes | No | No |
| **CCPA / GPC detection** | **Gateway-level** | Yes | No | Yes | No | No |
| **Consent versioning** | **Policy version** | Yes | Partial | Yes | No | No |
| **Withdrawal cascade** | **Email+token+3rdpty** | Partial | No | Partial | No | No |
| **Preference center** | **Built-in Console** | Widget | Widget | Widget | No | No |
| **Cookie consent banner** | **Yes** | Yes | Yes | Yes | No | No |
| **IAM-integrated** | **Yes (native)** | No (3rd party) | No (3rd party) | No (3rd party) | No | No |
| **Open source** | **Yes (Apache 2.0)** | No | No | No | No | No |

**Key differentiator**: GGID would be the only open-source IAM with **natively integrated** consent management — no third-party SaaS dependency. OneTrust/Cookiebot/TrustArc are standalone tools that bolt onto IAM; GGID makes consent a first-class IAM capability.

---

## 14. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Consent forgery** | All consent grants require authenticated session + audit trail |
| **Silent withdrawal** | Withdrawal logged to immutable consent_withdrawal_log + NATS publication |
| **Regulatory audit failure** | Complete audit trail: who, what, when, policy version, IP |
| **GPC header spoofing** | GPC signal logged; user must be authenticated for auto-withdraw |
| **DSR abuse (mass deletion)** | Rate limited + identity verification required + admin review for large deletions |
| **Stale consent after policy update** | Policy version tracking → re-consent required on version bump |
| **Cross-tenant consent leakage** | All queries scoped by tenant_id |
| **Data retention violation** | Expired consents auto-superseded; data purged per retention policy |

---

## References

- [GDPR Official Text](https://gdpr-info.eu/) — Full regulation text
- [CCPA Official Text](https://oag.ca.gov/privacy/ccpa) — California Attorney General
- [Global Privacy Control Specification](https://globalprivacycontrol.org/) — Browser opt-out signal
- [OneTrust](https://www.onetrust.com/) — Industry-leading consent platform
- [Cookiebot](https://www.cookiebot.com/) — Cookie consent automation
- [TrustArc](https://www.trustarc.com/) — Privacy management platform
- [GGID Consent Registry Handler](../services/identity/internal/server/consent_registry_handler.go) — Hardcoded mock at line 26
- [GGID OAuth Consent Store](../services/oauth/internal/service/consent.go) — In-memory store at line 35
- [GGID Consent Screen Handler](../services/oauth/internal/server/consent_screen_handler.go) — OIDC consent at line 24
- [GGID Consent Dashboard](../services/oauth/internal/server/consent_dashboard_handler.go) — Mock dashboard at line 24
- [GGID DSR/Consent Research](./consent-management.md) — Previous theoretical research (1048 lines)
- [GGID GDPR Compliance](./gdpr-compliance.md) — Earlier GDPR gap analysis
