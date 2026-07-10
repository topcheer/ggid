# GDPR Consent Management for IAM Systems

> **Scope**: This document covers the GDPR consent lifecycle, consent receipts,
> granular consent, consent fatigue, revocation cascades, and audit trails.
> OAuth scope-level consent flow is covered in `oidc-scope-management.md`.
> Data Subject Rights (DSR) fulfillment is covered in `data-residency-iam.md`.

---

## 1. GDPR Consent Requirements

### Legal Basis and Conditions

Under the General Data Protection Regulation (GDPR), consent is one of six legal
bases for processing personal data (Article 6(1)(a)). When consent is the chosen
basis, it must satisfy strict conditions defined in **Article 7**:

| Condition | Requirement | Invalid Example |
|-----------|-------------|-----------------|
| **Freely given** | Data subject has a genuine choice; no coercion or detriment for refusal | "Accept all cookies or leave the site" |
| **Specific** | Each purpose must be separately consented to | Bundling marketing + analytics into one checkbox |
| **Informed** | Data subject knows who, what, why, how long, and their rights | "We use your data to improve services" without specifics |
| **Unambiguous** | Clear affirmative action; no silence, pre-ticked boxes, or inactivity | Pre-checked consent checkbox |

### Article 7(1) — Demonstrate Consent

> "Where processing is based on consent, the controller **shall be able to
> demonstrate** that the data subject has consented to processing of his or her
> personal data."

This creates a **documentation obligation**: the IAM system must store not just
that consent was given, but what policy version the user saw, when, through what
interface, and for what purposes.

### Article 7(3) — Right to Withdraw

> "The data subject shall have the right to withdraw his or her consent at any
> time. [...] It shall be **as easy to withdraw** as to give consent."

This mandates:
- A withdrawal UI as accessible as the consent UI
- Immediate cessation of processing based on that consent
- A cascade effect: downstream systems, tokens, and shared data must be addressed

### Article 8 — Children's Data

- For information society services offered directly to a child, consent is valid
  only if the child is at least **16 years old** (member states may lower to 13).
- Below the threshold, consent must be given or authorized by the holder of
  parental responsibility.
- IAM systems must verify age and implement parental consent flows.

### Valid vs Invalid Consent — EDPB Guidelines

The European Data Protection Board (EDPB) has ruled against:

- **Consent walls** that block access unless all cookies are accepted
- **Dark patterns** that make "Accept" prominent and "Reject" hidden
- **Scroll-to-continue** wrapped consent banners
- **Implied consent** from continued use of a service

### Re-consent Triggers

Consent must be re-obtained when:
1. The **purpose** of processing changes
2. New **data categories** are introduced
3. New **third parties** receive data
4. The **retention period** is extended
5. A **policy version** materially changes

---

## 2. Consent Lifecycle

### State Machine

```
                    ┌──────────┐
                    │  PENDING │  ← consent required, not yet shown
                    └────┬─────┘
                         │ user views consent screen
                         ▼
                    ┌──────────┐
          ┌────────│ REQUESTED │  ← consent screen displayed, awaiting decision
          │         └────┬─────┘
          │              │ user makes decision
          │     ┌────────┴────────┐
          │     ▼                 ▼
          │ ┌────────┐      ┌─────────┐
          │ │ GRANTED│      │ DENIED  │
          │ └───┬────┘      └────┬────┘
          │     │                  │ user later changes mind
          │     │ processing       ▼
          │     │ active      (back to REQUESTED)
          │     │
          │     │ user withdraws
          │     ▼
          │ ┌──────────┐
          │ │WITHDRAWN │  ← processing stopped, data deletion scheduled
          │ └────┬─────┘
          │      │ retention period expires
          │      ▼
          │ ┌──────────┐
          └▶│ EXPIRED  │  ← data deleted, record retained for audit
            └──────────┘
```

### Capture

At capture time, the system records:
- **What data** is being collected (data categories: identity, contact, location, behavioral)
- **For what purpose** (authentication, marketing, analytics, third-party sharing)
- **To whom** it may be disclosed (data processors, third-party services)
- **How long** the data will be retained
- **Which policy version** the user was shown

### Storage

The consent record is an immutable, append-only artifact containing timestamp,
policy version, capture method (web form, API, in-app), and the exact text
presented to the user. Every modification creates a new version — the record is
never overwritten.

### Withdrawal

Withdrawal triggers:
1. Identify all data collected under this consent (by purpose + time range)
2. Stop all processing immediately (tokens, scheduled jobs, data pipelines)
3. Notify downstream service providers (back-channel logout, SAML/OIDC RP events)
4. Schedule data deletion within the legal retention window
5. Record the withdrawal timestamp and method

### Audit

Every state transition is logged with actor (user or system), timestamp, and
context. The audit trail serves as legal evidence under Article 7(1) — the
"ability to demonstrate" requirement.

---

## 3. Consent Record Schema

### PostgreSQL Schema

```sql
CREATE TABLE consent_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    tenant_id       UUID NOT NULL,

    -- What was consented to
    purpose         TEXT NOT NULL,              -- e.g., 'marketing', 'analytics'
    scopes          TEXT[] NOT NULL DEFAULT '{}', -- OAuth scopes if applicable
    data_categories TEXT[] NOT NULL DEFAULT '{}', -- e.g., '{"email","location","behavioral"}'
    third_parties   TEXT[] NOT NULL DEFAULT '{}', -- e.g., '{"google-analytics","salesforce"}'

    -- Retention
    retention_period INTERVAL,                  -- e.g., '2 years'
    expires_at      TIMESTAMPTZ,                -- computed from granted_at + retention

    -- Policy context
    policy_version  TEXT NOT NULL,              -- e.g., 'privacy-policy-v3.2'
    policy_url      TEXT NOT NULL,

    -- Lifecycle timestamps
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_via     TEXT NOT NULL,              -- 'web_form', 'api', 'in_app'
    withdrawn_at    TIMESTAMPTZ,
    withdrawn_via   TEXT,

    -- Receipt
    receipt_id      TEXT UNIQUE,                -- Kantara consent receipt ID
    receipt_jwt     TEXT,                       -- signed receipt JWT

    -- Hash chain for tamper evidence
    prev_hash       TEXT NOT NULL,
    record_hash     TEXT NOT NULL,

    -- Metadata
    ip_address      INET,
    user_agent      TEXT,

    CONSTRAINT chk_withdrawn_order CHECK (
        (withdrawn_at IS NULL) OR (withdrawn_at >= granted_at)
    )
);

CREATE INDEX idx_consent_user_tenant ON consent_records(user_id, tenant_id);
CREATE INDEX idx_consent_purpose     ON consent_records(purpose) WHERE withdrawn_at IS NULL;
CREATE INDEX idx_consent_active      ON consent_records(expires_at)
    WHERE withdrawn_at IS NULL;

-- Append-only: prevent UPDATE/DELETE
CREATE RULE consent_no_update AS ON UPDATE TO consent_records DO INSTEAD NOTHING;
CREATE RULE consent_no_delete AS ON DELETE TO consent_records DO INSTEAD NOTHING;
```

### Go Struct

```go
package consent

import (
    "time"

    "github.com/google/uuid"
)

// Record represents an immutable consent entry.
type Record struct {
    ID              uuid.UUID  `json:"id"`
    UserID          uuid.UUID  `json:"user_id"`
    TenantID        uuid.UUID  `json:"tenant_id"`

    Purpose         string     `json:"purpose"`
    Scopes          []string   `json:"scopes"`
    DataCategories  []string   `json:"data_categories"`
    ThirdParties    []string   `json:"third_parties"`

    RetentionPeriod string     `json:"retention_period,omitempty"` // PostgreSQL interval
    ExpiresAt       *time.Time `json:"expires_at,omitempty"`

    PolicyVersion   string     `json:"policy_version"`
    PolicyURL       string     `json:"policy_url"`

    GrantedAt       time.Time  `json:"granted_at"`
    GrantedVia      string     `json:"granted_via"`
    WithdrawnAt     *time.Time `json:"withdrawn_at,omitempty"`
    WithdrawnVia    string     `json:"withdrawn_via,omitempty"`

    ReceiptID       string     `json:"receipt_id,omitempty"`
    ReceiptJWT      string     `json:"receipt_jwt,omitempty"`

    PrevHash        string     `json:"prev_hash"`
    RecordHash      string     `json:"record_hash"`

    IPAddress       string     `json:"ip_address,omitempty"`
    UserAgent       string     `json:"user_agent,omitempty"`
}

// Status derives the current lifecycle state of a consent record.
func (r *Record) Status() string {
    if r.WithdrawnAt != nil {
        return "withdrawn"
    }
    if r.ExpiresAt != nil && time.Now().After(*r.ExpiresAt) {
        return "expired"
    }
    return "granted"
}
```

---

## 4. Consent Receipts

### Kantara Initiative Specification

The Kantara Initiative Consent Receipt (CR) specification defines a
machine-readable proof of consent. A consent receipt is a **signed JSON Web
Token (JWT)** containing structured metadata about the consent interaction.

Key fields per Kantara CR spec:

| Field | Description |
|-------|-------------|
| `jti` | Receipt ID (unique identifier) |
| `iss` | Jurisdiction (e.g., `EU-GDPR`) |
| `sub` | Data subject pseudonymous identifier |
| `iat` | Consent timestamp |
| `exp` | Receipt expiry |
| `controller` | Data controller (name, email, address) |
| `purposes` | Array of purpose definitions |
| `consentType` | `explicit` or `implicit` |
| `dataElements` | Data categories collected |
| `thirdParties` | Recipients of the data |

### Consent Receipt Generation (Go)

```go
package consent

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
)

// ReceiptClaims maps to the Kantara Consent Receipt JWT structure.
type ReceiptClaims struct {
    JTI             string         `json:"jti"`
    Iss             string         `json:"iss"`       // jurisdiction
    Sub             string         `json:"sub"`       // subject (pseudonymous)
    Iat             int64          `json:"iat"`       // issued at (unix)
    Exp             int64          `json:"exp"`       // receipt expiry (unix)
    Controller      ControllerInfo `json:"controller"`
    Purposes        []PurposeEntry `json:"purposes"`
    ConsentType     string         `json:"consentType"`
    DataElements    []string       `json:"dataElements"`
    ThirdParties    []string       `json:"thirdParties,omitempty"`
    PolicyVersion   string         `json:"policyVersion"`
}

type ControllerInfo struct {
    Name    string `json:"name"`
    Email   string `json:"email"`
    Address string `json:"address,omitempty"`
}

type PurposeEntry struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

// GenerateReceipt creates a signed consent receipt JWT.
func GenerateReceipt(r *Record, controller ControllerInfo, signingKey []byte) (string, error) {
    now := time.Now()
    receiptID := "cr_" + uuid.New().String()

    purposes := make([]PurposeEntry, 0, 1)
    purposes = append(purposes, PurposeEntry{
        ID:          r.Purpose,
        Name:        r.Purpose,
        Description: "Processing for " + r.Purpose,
    })

    claims := ReceiptClaims{
        JTI:          receiptID,
        Iss:          "EU-GDPR",
        Sub:          "user_" + r.UserID.String()[:8],
        Iat:          now.Unix(),
        Exp:          now.Add(365 * 24 * time.Hour).Unix(),
        Controller:   controller,
        Purposes:     purposes,
        ConsentType:  "explicit",
        DataElements: r.DataCategories,
        ThirdParties: r.ThirdParties,
        PolicyVersion: r.PolicyVersion,
    }

    // Build JWT manually (header.payload.signature)
    header := map[string]string{"alg": "HS256", "typ": "JWT"}
    headerJSON, _ := json.Marshal(header)
    payloadJSON, _ := json.Marshal(claims)

    headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
    payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
    signingInput := headerB64 + "." + payloadB64

    mac := hmac.New(sha256.New, signingKey)
    mac.Write([]byte(signingInput))
    sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

    return fmt.Sprintf("%s.%s", signingInput, sigB64), nil
}

// VerifyReceipt validates the signature of a consent receipt JWT.
func VerifyReceipt(receipt string, signingKey []byte) (*ReceiptClaims, error) {
    parts := splitJWT(receipt)
    if len(parts) != 3 {
        return nil, fmt.Errorf("invalid receipt format")
    }

    signingInput := parts[0] + "." + parts[1]
    mac := hmac.New(sha256.New, signingKey)
    mac.Write([]byte(signingInput))
    expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

    if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
        return nil, fmt.Errorf("invalid receipt signature")
    }

    payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
    if err != nil {
        return nil, fmt.Errorf("decode payload: %w", err)
    }

    var claims ReceiptClaims
    if err := json.Unmarshal(payloadBytes, &claims); err != nil {
        return nil, fmt.Errorf("unmarshal claims: %w", err)
    }

    if time.Now().Unix() > claims.Exp {
        return nil, fmt.Errorf("receipt expired")
    }

    return &claims, nil
}

func splitJWT(jwt string) []string {
    var parts []string
    start := 0
    for i, c := range jwt {
        if c == '.' {
            parts = append(parts, jwt[start:i])
            start = i + 1
        }
    }
    parts = append(parts, jwt[start:])
    return parts
}
```

---

## 5. Granular Consent

### Per-Purpose Model

Instead of a single "I agree" checkbox, granular consent presents each purpose
as an independent decision:

```
┌─────────────────────────────────────────────────────┐
│  Manage Your Privacy Preferences                     │
├─────────────────────────────────────────────────────┤
│                                                      │
│  [✓] Essential (Required)                            │
│      Authentication, session management              │
│                                                      │
│  [✓] Analytics (Optional)                           │
│      Usage data to improve the platform              │
│                                                      │
│  [ ] Marketing (Optional)                            │
│      Product updates and promotional emails          │
│                                                      │
│  [ ] Third-Party Sharing (Optional)                  │
│      Share data with integration partners            │
│                                                      │
│         [ Save Preferences ]    [ Reject All ]       │
└─────────────────────────────────────────────────────┘
```

### Handling Denied Consent

When a user denies consent for a purpose:
- **Essential**: cannot be denied (required for service operation)
- **Optional**: feature degrades gracefully (no analytics tracking, no marketing
  emails, no data sharing)
- The system must not fail or block the user for denying optional consent

### Consent Upgrade / Downgrade

Users can change their consent at any time:

```go
package consent

import (
    "context"
    "time"
)

// PurposeCategory defines consent granularity.
type PurposeCategory string

const (
    PurposeEssential PurposeCategory = "essential"
    PurposeAnalytics PurposeCategory = "analytics"
    PurposeMarketing PurposeCategory = "marketing"
    PurposeThirdParty PurposeCategory = "third_party"
)

// GranularConsent holds per-purpose decisions for a user.
type GranularConsent struct {
    UserID  uuid.UUID
    TenantID uuid.UUID
    Decisions map[PurposeCategory]bool
    GrantedAt time.Time
}

// UpdateConsent applies a user's per-purpose consent change.
// It creates a NEW consent record (append-only) rather than modifying the old one.
func UpdateConsent(
    ctx context.Context,
    store Store,
    auditEmitter AuditEmitter,
    userID, tenantID uuid.UUID,
    changes map[PurposeCategory]bool,
    policyVersion, policyURL string,
    ip, userAgent string,
) ([]*Record, error) {

    var records []*Record
    now := time.Now()

    for purpose, granted := range changes {
        r := &Record{
            UserID:        userID,
            TenantID:      tenantID,
            Purpose:       string(purpose),
            PolicyVersion: policyVersion,
            PolicyURL:     policyURL,
            GrantedAt:     now,
            GrantedVia:    "web_form",
            IPAddress:     ip,
            UserAgent:     userAgent,
        }

        if !granted {
            // If previously granted and now denied, record withdrawal
            existing, err := store.GetActiveConsent(ctx, userID, tenantID, string(purpose))
            if err == nil && existing != nil {
                with := now
                existing.WithdrawnAt = &with
                existing.WithdrawnVia = "web_form"
                // existing is immutable — store creates a new versioned record
                if err := store.RecordWithdrawal(ctx, existing); err != nil {
                    return nil, err
                }
            }
            continue
        }

        // Grant new consent
        if err := store.Create(ctx, r); err != nil {
            return nil, err
        }
        records = append(records, r)
    }

    // Emit audit event for the change
    _ = auditEmitter.Emit(ctx, AuditEvent{
        Action:   "consent.update",
        ActorID:  userID,
        Changes:  changes,
        Records:  records,
    })

    return records, nil
}

// Store abstracts the persistence layer for consent records.
type Store interface {
    Create(ctx context.Context, r *Record) error
    GetActiveConsent(ctx context.Context, userID, tenantID, purpose string) (*Record, error)
    RecordWithdrawal(ctx context.Context, r *Record) error
    ListByUser(ctx context.Context, userID, tenantID uuid.UUID) ([]*Record, error)
}
```

---

## 6. Consent Fatigue Mitigation

### The Problem

Users face an average of **10-30 consent prompts per day** across modern web
services. Research shows:

- **85%** of users click "Accept All" without reading (CNIL 2019 study)
- **70%** ignore cookie banners entirely when possible
- **Prompt blindness**: users develop a muscle-memory "accept" reflex
- **Decision fatigue**: cognitive load degrades decision quality after 5+ prompts

This creates a paradox: more consent prompts lead to **less meaningful consent**,
undermining the GDPR's goal of informed choice.

### Mitigation Strategies

#### 1. Progressive Consent

Ask for consent **only when the purpose becomes relevant**, not upfront:

```
Registration → "Create Account" (no consent prompt)
                ↓
First analytics event → "Allow analytics to improve the platform?"
                        [ Allow ]  [ Not Now ]
                ↓
Marketing email → "Would you like product updates?"
                   [ Yes ]  [ No Thanks ]
```

This reduces initial friction and asks at the point of context.

#### 2. Remember Consent (Don't Re-Ask)

Once a user has made a decision, store it and **never re-prompt** unless:
- The policy version changes materially
- A new purpose or data category is introduced
- The user explicitly requests to review settings

```go
// ShouldPrompt decides whether to show a consent prompt.
type PromptPolicy struct {
    RememberDuration time.Duration // e.g., 12 months
    RePromptOnPolicyChange bool
}

func (p *PromptPolicy) ShouldPrompt(
    userID uuid.UUID,
    purpose string,
    currentPolicyVersion string,
    lastRecord *Record,
) bool {
    if lastRecord == nil {
        return true // never consented
    }
    if p.RePromptOnPolicyChange && lastRecord.PolicyVersion != currentPolicyVersion {
        return true // policy changed
    }
    if time.Since(lastRecord.GrantedAt) > p.RememberDuration {
        return true // stale
    }
    return false // user already decided
}
```

#### 3. Smart Defaults

Set opt-in defaults based on the **minimum necessary**:
- Essential: always on (required)
- Analytics: off by default (opt-in)
- Marketing: off by default (opt-in)
- Third-party sharing: off by default (opt-in)

Never pre-tick optional consent checkboxes — this violates EDPB guidelines.

#### 4. Layered Consent UI

Instead of overwhelming users with all details at once:

```
Layer 1: Quick choice — "Accept All" / "Reject All" / "Customize"
         ↓ (click "Customize")
Layer 2: Per-purpose toggles — granular control
         ↓ (click any purpose)
Layer 3: Details — exact data collected, third parties, retention
```

#### 5. A/B Testing Consent UX

Measure consent rates and user behavior:
- Variant A: Standard banner
- Variant B: Progressive disclosure
- Metric: opt-in rate for optional purposes
- **Critical**: A/B testing must not manipulate users into consenting (no dark
  patterns). Test for usability, not for coercion.

### Anti-Patterns (Avoid)

- **Consent walls**: blocking all access until consent is given
- **Unequal prominence**: "Accept" button large/green, "Reject" hidden in a submenu
- **Infinite scroll**: burying the reject option below the fold
- **Pre-ticked boxes**: explicitly prohibited by GDPR
- **Cookie refresh**: re-prompting every 24 hours to wear down resistance

---

## 7. Consent Revocation Cascade

When a user withdraws consent, the IAM system must propagate the withdrawal
across all dependent systems:

```
User withdraws consent for "marketing"
         │
         ▼
┌──────────────────────────┐
│ 1. Revoke tokens          │  Find all access tokens / refresh tokens
│    issued under this      │  issued with marketing-related scopes
│    consent                │  → add jti to revocation list
└────────────┬──────────────┘
             ▼
┌──────────────────────────┐
│ 2. Back-channel logout    │  Notify downstream service providers
│    to downstream SPs      │  via OIDC back-channel logout or
│                           │  SAML single logout
└────────────┬──────────────┘
             ▼
┌──────────────────────────┐
│ 3. Stop processing        │  Remove user from marketing lists,
│                           │  stop analytics tracking, disable
│                           │  data pipelines
└────────────┬──────────────┘
             ▼
┌──────────────────────────┐
│ 4. Schedule data deletion │  Within retention window (e.g., 30 days),
│                           │  delete all data collected under this
│                           │  consent
└────────────┬──────────────┘
             ▼
┌──────────────────────────┐
│ 5. Audit + receipt update │  Record withdrawal in audit trail,
│                           │  issue withdrawal receipt
└──────────────────────────┘
```

### Revocation Cascade (Go)

```go
package consent

import (
    "context"
    "time"

    "github.com/google/uuid"
)

// RevocationCascade handles the full withdrawal workflow.
type RevocationCascade struct {
    consentStore   Store
    tokenRevoker   TokenRevoker
    spNotifier     SPNotifier
    dataDeleter    DataDeleter
    auditEmitter   AuditEmitter
    retentionDelay time.Duration // grace period before deletion
}

type TokenRevoker interface {
    RevokeByConsent(ctx context.Context, consentID uuid.UUID) error
    ListByConsent(ctx context.Context, consentID uuid.UUID) ([]TokenInfo, error)
}

type SPNotifier interface {
    BackChannelLogout(ctx context.Context, userID uuid.UUID, clientID string) error
}

type DataDeleter interface {
    ScheduleDeletion(ctx context.Context, userID uuid.UUID, purpose string, delay time.Duration) error
}

type TokenInfo struct {
    JTI       string
    ClientID  string
    ExpiresAt time.Time
}

// ExecuteRevocation performs the full withdrawal cascade.
func (rc *RevocationCascade) ExecuteRevocation(
    ctx context.Context,
    userID, tenantID uuid.UUID,
    purpose string,
) error {
    // 1. Find the active consent record
    record, err := rc.consentStore.GetActiveConsent(ctx, userID, tenantID, purpose)
    if err != nil {
        return fmt.Errorf("find consent: %w", err)
    }
    if record == nil {
        return fmt.Errorf("no active consent for purpose %q", purpose)
    }

    // 2. Revoke tokens issued under this consent
    tokens, err := rc.tokenRevoker.ListByConsent(ctx, record.ID)
    if err != nil {
        return fmt.Errorf("list tokens: %w", err)
    }
    if err := rc.tokenRevoker.RevokeByConsent(ctx, record.ID); err != nil {
        return fmt.Errorf("revoke tokens: %w", err)
    }

    // 3. Notify downstream service providers
    notifiedClients := make(map[string]bool)
    for _, tok := range tokens {
        if notifiedClients[tok.ClientID] {
            continue
        }
        if err := rc.spNotifier.BackChannelLogout(ctx, userID, tok.ClientID); err != nil {
            // Log but don't fail — SP may be offline
            _ = rc.auditEmitter.Emit(ctx, AuditEvent{
                Action:  "consent.revocation.sp_notify_failed",
                ActorID: userID,
                Metadata: map[string]any{
                    "client_id": tok.ClientID,
                    "error":     err.Error(),
                },
            })
        }
        notifiedClients[tok.ClientID] = true
    }

    // 4. Mark consent as withdrawn (append-only: creates new version)
    now := time.Now()
    record.WithdrawnAt = &now
    record.WithdrawnVia = "user_request"
    if err := rc.consentStore.RecordWithdrawal(ctx, record); err != nil {
        return fmt.Errorf("record withdrawal: %w", err)
    }

    // 5. Schedule data deletion after retention grace period
    if err := rc.dataDeleter.ScheduleDeletion(ctx, userID, purpose, rc.retentionDelay); err != nil {
        return fmt.Errorf("schedule deletion: %w", err)
    }

    // 6. Emit audit event
    _ = rc.auditEmitter.Emit(ctx, AuditEvent{
        Action:   "consent.withdrawn",
        ActorID:  userID,
        Metadata: map[string]any{
            "purpose":      purpose,
            "consent_id":   record.ID,
            "tokens_revoked": len(tokens),
            "sps_notified":  len(notifiedClients),
            "deletion_at":   now.Add(rc.retentionDelay),
        },
    })

    return nil
}
```

---

## 8. Consent Audit Trail

### Requirements

The audit trail must satisfy **legal evidence quality** standards:

1. **Immutable**: records cannot be modified or deleted (append-only)
2. **Tamper-evident**: hash-chaining detects any alteration
3. **Complete**: every consent state transition is recorded
4. **Verifiable**: independent auditor can reconstruct the full history

### Hash Chaining

Each consent record includes the hash of the previous record, creating a chain:

```
Record 1: hash = SHA256(data_1)
Record 2: prev_hash = hash_1, hash = SHA256(prev_hash || data_2)
Record 3: prev_hash = hash_2, hash = SHA256(prev_hash || data_3)
```

If any record is modified, its hash changes, breaking all subsequent links.

### Consent Audit Emitter (Go)

```go
package consent

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"

    auditevents "ggid.dev/ggid/pkg/audit"
)

// AuditEvent represents a consent lifecycle event for the audit trail.
type AuditEvent struct {
    Action   string                 // consent.granted, consent.withdrawn, consent.modified
    ActorID  uuid.UUID
    TenantID uuid.UUID
    Changes  map[PurposeCategory]bool
    Records  []*Record
    Metadata map[string]any
}

// AuditEmitter publishes consent audit events.
type AuditEmitter interface {
    Emit(ctx context.Context, event AuditEvent) error
}

// NATSAuditEmitter integrates with the GGID audit service via NATS.
type NATSAuditEmitter struct {
    publisher *auditevents.Publisher
}

// Emit publishes a consent audit event to NATS JetStream.
func (e *NATSAuditEmitter) Emit(ctx context.Context, event AuditEvent) error {
    auditEvent := auditevents.Event{
        ID:           uuid.New(),
        TenantID:     event.TenantID,
        ActorType:    "user",
        ActorID:      event.ActorID,
        Action:       event.Action,
        ResourceType: "consent",
        Result:       "success",
        CreatedAt:    time.Now().UTC(),
        Metadata:     event.Metadata,
    }

    // Attach hash-chained consent record info to metadata
    for _, r := range event.Records {
        chainHash, err := ComputeHash(r)
        if err != nil {
            return fmt.Errorf("compute hash: %w", err)
        }
        key := fmt.Sprintf("consent_record_%s", r.ID)
        auditEvent.Metadata[key] = map[string]any{
            "consent_id":    r.ID,
            "purpose":       r.Purpose,
            "record_hash":   chainHash,
            "prev_hash":     r.PrevHash,
            "granted_at":    r.GrantedAt,
            "withdrawn_at":  r.WithdrawnAt,
            "policy_version": r.PolicyVersion,
        }
    }

    return e.publisher.Publish(ctx, auditEvent)
}

// ComputeHash generates SHA-256 hash for a consent record.
func ComputeHash(r *Record) (string, error) {
    payload := struct {
        ID            uuid.UUID  `json:"id"`
        UserID        uuid.UUID  `json:"user_id"`
        TenantID      uuid.UUID  `json:"tenant_id"`
        Purpose       string     `json:"purpose"`
        Scopes        []string   `json:"scopes"`
        PolicyVersion string     `json:"policy_version"`
        GrantedAt     time.Time  `json:"granted_at"`
        WithdrawnAt   *time.Time `json:"withdrawn_at"`
        PrevHash      string     `json:"prev_hash"`
    }{
        ID:            r.ID,
        UserID:        r.UserID,
        TenantID:      r.TenantID,
        Purpose:       r.Purpose,
        Scopes:        r.Scopes,
        PolicyVersion: r.PolicyVersion,
        GrantedAt:     r.GrantedAt,
        WithdrawnAt:   r.WithdrawnAt,
        PrevHash:      r.PrevHash,
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return "", err
    }

    h := sha256.Sum256(data)
    return hex.EncodeToString(h[:]), nil
}

// VerifyChain validates the hash chain across a list of records.
func VerifyChain(records []*Record) error {
    for i, r := range records {
        expectedHash, err := ComputeHash(r)
        if err != nil {
            return fmt.Errorf("record %d: compute hash: %w", i, err)
        }
        if r.RecordHash != expectedHash {
            return fmt.Errorf("record %d: hash mismatch (expected %s, got %s)",
                i, expectedHash, r.RecordHash)
        }
        if i > 0 && r.PrevHash != records[i-1].RecordHash {
            return fmt.Errorf("record %d: broken chain link", i)
        }
    }
    return nil
}
```

---

## 9. GGID Consent Gap Analysis

### What Exists

The GGID OAuth service (`services/oauth/internal/server/server.go`) has a basic
consent flow:

1. **Consent detection in `/oauth/authorize`** (lines 233-254): When a client
   requests non-basic scopes beyond `openid`, `profile`, `email`, and
   `offline_access`, the server returns a `consent_required` response with the
   requested scopes and a consent URL.

2. **Consent screen endpoint `/oauth/consent`** (lines 595-637): A GET handler
   that returns the scopes and client info, and a POST handler that accepts
   `decision=approve|deny`. On approval, it redirects to `/oauth/authorize` with
   `consent=true`.

3. **CIBA consent** (`services/oauth/internal/service/ciba.go` line 194):
   `ApproveCIBAAuth` handles user approval of a CIBA authentication request,
   serving as a consent checkpoint for device-initiated login.

### What's Missing

| Gap | Description | Impact |
|-----|-------------|--------|
| **No consent persistence** | Consent decisions are not stored — `consent=true` is a query parameter, not a record | Cannot demonstrate consent (Art. 7(1)) |
| **No consent record schema** | No database table for consent records | No audit trail |
| **No consent receipt** | No Kantara-compliant receipt generation | No verifiable proof of consent |
| **No withdrawal mechanism** | No endpoint to revoke previously given consent | Violates Art. 7(3) |
| **No granular consent** | All-or-nothing approve/deny — no per-purpose toggles | Violates "specific" requirement |
| **No revocation cascade** | Withdrawing consent does not revoke tokens or notify SPs | Processing continues after withdrawal |
| **No policy versioning** | Consent does not record which privacy policy version was shown | Cannot prove informed consent |
| **No re-consent trigger** | No mechanism to force re-consent on policy changes | Stale consent is legally void |
| **No consent audit trail** | Consent changes are not published to the audit service | No evidence for regulators |
| **No hash chaining** | No tamper-evidence for consent records | Audit trail can be disputed |
| **No `pkg/consent` package** | Zero consent utility code exists in shared packages | No reusable abstractions |

---

## 10. Gap Analysis and Recommendations

### Priority Action Items

| # | Action | Effort | Priority |
|---|--------|--------|----------|
| 1 | **Create `pkg/consent` package** with `Record` struct, `Store` interface, and hash-chaining utilities | M (3-4 days) | P0 |
| 2 | **Add consent record persistence** — PostgreSQL migration with append-only rules, integrate with OAuth consent endpoint to store records | M (3-5 days) | P0 |
| 3 | **Implement consent withdrawal endpoint** (`DELETE /oauth/consent/{purpose}`) with revocation cascade: revoke tokens, back-channel logout, schedule deletion | L (5-7 days) | P0 |
| 4 | **Add consent receipt generation** — Kantara JWT receipts on grant and withdrawal, stored alongside the record | M (3-4 days) | P1 |
| 5 | **Integrate consent events with audit service** — emit `consent.granted` / `consent.withdrawn` events via `pkg/audit.Publisher`, hash-chained for tamper evidence | S-M (2-3 days) | P1 |
| 6 | **Add policy versioning** — track privacy policy versions, trigger re-consent when version changes, display version in consent UI | M (3-4 days) | P1 |
| 7 | **Build granular consent UI** — per-purpose toggles (essential/analytics/marketing/third-party), smart defaults, progressive disclosure | L (5-7 days) | P2 |

### Estimated Total Effort

- **P0 items** (consent storage + withdrawal + cascade): ~11-16 days
- **P1 items** (receipts + audit + versioning): ~8-11 days
- **P2 items** (granular UI): ~5-7 days
- **Total**: ~24-34 engineering days for full GDPR-compliant consent management

### Dependencies

- Action 3 (withdrawal cascade) depends on Action 1 (pkg/consent) and Action 2 (persistence)
- Action 5 (audit integration) depends on Action 1 (hash-chaining utilities)
- Action 7 (granular UI) depends on Action 2 (persistence) and Action 6 (versioning)

### Risk Assessment

- **Legal risk**: Without consent persistence, GGID cannot demonstrate compliance
  under GDPR Art. 7(1). This is a **blocking gap** for any EU deployment.
- **Operational risk**: Without revocation cascade, withdrawn consent does not
  stop processing — a direct GDPR violation with fines up to 4% of global revenue.
- **UX risk**: Without granular consent, users cannot make specific choices —
  consent may be deemed invalid by regulators.

---

## References

- GDPR Article 7: Conditions for consent
- GDPR Article 8: Conditions applicable to child's consent
- EDPB Guidelines 05/2020 on consent under Regulation 2016/679
- Kantara Initiative Consent Receipt Specification v1.1
- CNIL (French DPA) study on cookie consent fatigue (2019)
- ISO/IEC 29184:2020 — Privacy notices and consent
- IETF RFC 7049 — Back-Channel Logout (OIDC)
