# Compliance Automation for GGID

> Research document: continuous compliance, OSCAL, automated control testing, evidence
> collection, drift detection, and policy-as-code integration for the GGID IAM suite.

---

## 1. Overview

Traditional compliance relies on point-in-time audits — an external auditor reviews evidence
quarterly or annually, producing a snapshot that is already stale by the time it is certified.
This model is expensive, labor-intensive, and provides only a lagging indicator of control
effectiveness.

**Continuous compliance** replaces the snapshot model with always-on, automated control
verification. The system continuously verifies that security controls are in place, alerts
security teams on violations, and generates evidence packages on demand or on schedule.

Key drivers:

- **SOC 2 Type II** — requires evidence that controls operate effectively *throughout* the period
- **ISO 27001** — Annex A controls increasingly mapped to automated verification
- **FedRAMP** — mandates continuous monitoring (3PAO monthly assessments)
- **HIPAA** — 164.312(b) audit controls, 164.308(a)(1)(ii)(D) information system activity review
- **GDPR Art. 32** — ongoing security of processing, encryption, regular testing

GGID already has the foundational infrastructure: NATS JetStream audit pipeline with HMAC hash
chain tamper evidence, an RBAC + ABAC policy engine, and per-tenant multi-tenancy. This document
maps a path from that foundation to full continuous compliance automation.

---

## 2. OSCAL (NIST)

### What is OSCAL

The **Open Security Controls Assessment Language** (OSCAL) is a NIST-led standard providing
machine-readable formats (XML, JSON, YAML) for security controls, implementations, assessment
plans, and results. OSCAL reduces audit durations by automating control-based risk assessments.

### OSCAL Models (layered)

```
Catalog ──► Profile ──► Component Definition ──► SSP ──► Assessment Plan ──► Assessment Results
 (source)   (selection)   (how I implement)       (what)   (how to test)        (test outcomes)
```

| Model | Purpose | GGID Mapping |
|-------|---------|--------------|
| **Catalog** | Control definitions (e.g., NIST 800-53 Rev 5) | Reference — do not modify |
| **Profile** | Which controls apply to this system | Select controls relevant to IAM (AC, AU, IA, SC families) |
| **Component Definition** | How GGID implements each control | Each GGID feature = one implementation statement |
| **System Security Plan (SSP)** | System description + control implementations | Generated from GGID configuration |
| **Assessment Plan** | How to test the controls | Automated test definitions |
| **Assessment Results** | Machine-readable test outcomes | Evidence package output |

### GGID + OSCAL: Control Mapping Examples

| NIST 800-53 Control | GGID Feature | Implementation |
|---------------------|-------------|----------------|
| IA-2(1) MFA for privileged access | TOTP/WebAuthn | Enforce MFA for admin roles |
| AC-2 Account management | Identity service | CRUD lifecycle, deactivation |
| AU-2 Event logging | NATS audit pipeline | All authz decisions logged |
| SC-13 Cryptographic protection | Argon2id + TLS 1.3 | Password hashing, transport |
| IA-5 Password authentication | Auth service | NIST 800-63B password policy |
| AC-12 Session termination | JWT refresh rotation | Token expiry, revocation |

### OSCAL Export Architecture

```
GGID Config API ──► ControlMapper ──► OSCAL Component Definition (JSON)
                        │
                        ├──► SSP Generator ──► System Security Plan
                        │
                        └──► Assessment Results (from ControlTester) ──► Evidence Package
```

```go
type ComponentDefinition struct {
    UUID        string                   `json:"uuid"`
    Type        string                   `json:"type"` // "software"
    Title       string                   `json:"title"`
    ControlImpl []ControlImplementation  `json:"control-implementations"`
}
type ControlImplementation struct {
    Source string                   `json:"source"` // catalog URL
    Impls  []ImplementedRequirement `json:"implemented-requirements"`
}
type ImplementedRequirement struct {
    ControlID   string `json:"control-id"`
    Description string `json:"description"`
}
```

---

## 3. Automated Control Testing

### Control Types

- **Preventive**: access control (RBAC/ABAC), input validation, encryption at rest/transit
- **Detective**: audit logging, anomaly detection, SIEM alerting
- **Corrective**: incident response runbooks, backup recovery procedures

### Automated Test Matrix

| Control | NIST Ref | Test Method | GGID Integration Point |
|---------|----------|-------------|----------------------|
| Access Control | AC-2 | Query users without role assignments | Identity service `GET /api/v1/users` + Policy `GET /roles/assignments` |
| MFA | IA-2(1) | Verify MFA enrollment rate per tenant | Auth service: count enrolled vs total users |
| Audit Logging | AU-2 | Verify all required events are emitted | Audit service: check event coverage by action type |
| Encryption | SC-13 | Verify TLS config + Argon2id params | Auth config: hash memory/time parameters |
| Session Mgmt | AC-12 | Verify session timeout enforcement | JWT config: access/refresh token TTL |
| Password Policy | IA-5 | Verify NIST 800-63B compliance | Auth config: min length, breach check, no composition rules |
| Rate Limiting | SC-5 | Verify limits on auth endpoints | Gateway config: per-route rate limit |

### Test Execution

```go
type ControlTest struct {
    ID          string
    ControlID   string // e.g., "IA-2(1)"
    Run         func(ctx context.Context, deps Deps) TestResult
}
type TestResult struct {
    ControlID string    `json:"control_id"`
    Status    string    `json:"status"` // pass | fail | warning
    Evidence  []Evidence `json:"evidence"`
    Details   string    `json:"details,omitempty"`
}

// Example: AC-2 — no orphan accounts
var OrphanAccountCheck = ControlTest{
    ID: "ac-2-orphan-check", ControlID: "AC-2",
    Run: func(ctx context.Context, deps Deps) TestResult {
        users, _ := deps.Identity.ListUsers(ctx, tenantID)
        var orphans []string
        for _, u := range users {
            if roles, _ := deps.Policy.GetUserRoles(ctx, u.ID); len(roles) == 0 {
                orphans = append(orphans, u.Email)
            }
        }
        if len(orphans) > 0 {
            return TestResult{ControlID: "AC-2", Status: "fail",
                Details: fmt.Sprintf("%d users without role", len(orphans))}
        }
        return TestResult{ControlID: "AC-2", Status: "pass"}
    },
}
```

**Scheduling**: daily scheduled runs via NATS-triggered job, on-change triggers after config
mutations, on-demand API for auditors.

---

## 4. Evidence Collection from Audit Logs

### What Auditors Need

Auditors require evidence that controls operated effectively during the audit period:

- Who accessed what, when, from which IP
- Privileged actions (role assignments, policy changes) with full context
- Failed access attempts and denial events
- Configuration changes (before/after)
- Evidence of periodic access reviews

### GGID Audit Infrastructure (Current)

GGID's `pkg/audit` publisher emits `audit.Event` structs to NATS JetStream with HMAC hash-chain
tamper evidence (`AuditEvent.Hash`). The audit service persists events and exposes filtering
via `domain.ListFilter` (tenant, actor, action, resource type, result, time range).

```go
type EvidenceCollector struct{ auditSvc *service.AuditService }

func (c *EvidenceCollector) Collect(ctx context.Context, tenantID uuid.UUID,
    controls []string, from, to time.Time) (*EvidencePackage, error) {
    filter := domain.ListFilter{TenantID: tenantID, StartTime: &from, EndTime: &to}
    events, total, err := c.auditSvc.ListEvents(ctx, filter, 1, 10000)
    if err != nil {
        return nil, err
    }
    return &EvidencePackage{
        TenantID: tenantID, Controls: controls, Period: Period{From: from, To: to},
        EventCount: total, Events: events, GeneratedAt: time.Now().UTC(),
    }, nil
}
```

### Evidence Package Format (OSCAL Assessment Results)

```json
{
  "assessment-results": {
    "uuid": "550e8400-e29b-41d4-a716-446655440000",
    "metadata": { "title": "GGID Continuous Compliance Assessment" },
    "results": [{
      "reviewed-controls": [{
        "control-id": "AU-2",
        "result": "satisfied",
        "remarks": "100% of required events logged with HMAC hash chain"
      }]
    }]
  }
}
```

---

## 5. Drift Detection

### Configuration Drift

Compare current configuration against an approved baseline. Example: rate limit changed from
100/min to 1000/min — is this an approved change, or unauthorized drift?

### Policy Drift

- New RBAC role created with wildcard permissions (`actions: ["*"]`)
- New ABAC policy with `Effect: "deny"` removed
- Alert: "unexpected policy change detected" with before/after diff

### Implementation

```go
type Snapshot struct {
    TenantID uuid.UUID       `json:"tenant_id"`
    Config   map[string]any  `json:"config"`
    Roles    []domain.Role   `json:"roles"`
    Policies []domain.Policy `json:"policies"`
    Hash     string          `json:"hash"` // SHA-256
}

type DriftDetector struct {
    store     SnapshotStore
    publisher *audit.Publisher
}

func (d *DriftDetector) Compare(ctx context.Context, tenantID uuid.UUID) ([]Diff, error) {
    current, _ := d.TakeSnapshot(ctx, tenantID)
    baseline, _ := d.store.GetBaseline(ctx, tenantID)
    diffs := computeDiffs(baseline, current)
    if len(diffs) > 0 {
        _ = d.publisher.Publish(ctx, audit.Event{
            Action: "compliance.drift_detected", Result: "warning",
            Metadata: map[string]any{"diffs": diffs},
        })
    }
    return diffs, nil
}
```

**Alerting**: NATS event on drift → notification service → Slack/email to security team.
**Prevention**: require change-approval workflow (GitHub PR + CODEOWNERS) for policy mutations.

---

## 6. Policy-as-Code (OPA/Rego)

### What is Policy-as-Code

**Rego** (the policy language for the Open Policy Agent) defines security policies as code.
Policies are version-controlled in Git, reviewed via PRs, and evaluated in real-time at
authorization decision points.

### GGID + OPA Integration

GGID's existing ABAC engine (`domain.CheckRequest`/`CheckResult`) can be augmented with OPA:

```rego
# require_mfa_for_admin.rego
package ggid.compliance

default allow := true

deny[msg] {
    input.action == "admin"
    not input.user.mfa_enrolled
    msg := sprintf("MFA required for admin operations (user: %s)", [input.user.id])
}

# soc2_cc61_user_role_check.rego
deny[msg] {
    count(input.user.roles) == 0
    msg := sprintf("SOC2 CC6.1 violation: user %s has no role assignment", [input.user.id])
}

# gdpr_article32_encryption.rego
deny[msg] {
    input.config.password_hash_algo != "argon2id"
    msg := "GDPR Art.32: password hashing must use Argon2id"
}

# blocked_country.rego
deny[msg] {
    geoip_country(input.client_ip) in blocked_countries
    msg := sprintf("Access denied from blocked country (IP: %s)", [input.client_ip])
}
```

### OPA Deployment Architecture

```
Git (Rego policies) ──► CI ──► OPA Bundle Server
                                      │ bundle pull (30s)
                                      ▼
  GGID Gateway + OPA Sidecar (localhost:8181)
  ┌──────────────┐    ┌──────────────┐
  │ Gateway MW   │───►│ OPA Decision │──► allow/deny
  │ (authz)      │◄───│ Point        │
  └──────────────┘    └──────┬───────┘
                             │ decision logs
                             ▼
                    NATS audit.events
```

**Decision logs**: every OPA evaluation emits a JSON decision log into GGID's audit pipeline —
full policy decision provenance for auditors.

---

## 7. SOC 2 / ISO 27001 Continuous Monitoring

### SOC 2 Trust Services Criteria

SOC 2 defines five categories: **Security, Availability, Processing Integrity,
Confidentiality, Privacy**. Continuous monitoring (CC Principle 7.2) requires controls
operating 24/7 with automated evidence collection replacing manual screenshots and spreadsheets.

### ISO 27001:2022 Annex A

Annex A controls are organized into four domains: A.5 Organizational, A.6 People, A.7 Physical,
A.8 Technological. Technical controls (encryption A.8.24, access control A.8.2-8.5, logging
A.8.15-8.17) can be verified automatically. Organizational controls (awareness training A.6.3,
supplier relationships A.5.19) require human attestation.

### GGID Compliance Dashboard

- **Real-time**: per-control status (green/yellow/red) updated every 5 minutes
- **Metrics**: MFA enrollment %, failed auth rate, session count, policy changes, drift alerts
- **Compliance score**: trending over time with control failure history
- **Alerts**: control failure → Slack/email within 60 seconds
- **Export**: on-demand evidence package (JSON + PDF) for external auditor

---

## 8. GGID Current Compliance Posture

| Compliance Area | Current State | Automation Status |
|----------------|--------------|-------------------|
| Audit logging | NATS JetStream + HMAC hash chain | Coverage gaps — not all events emit |
| RBAC | Policy service with role inheritance | No automated verification |
| ABAC | AWS IAM-style policies with conditions | No compliance mapping |
| MFA | TOTP + WebAuthn | No enrollment rate monitoring |
| Encryption | Argon2id (passwords), TLS 1.3 (transport) | No automated config check |
| Rate limiting | Gateway middleware (token bucket) | No compliance reporting |
| Password policy | Partial — composition rules present | Not NIST 800-63B compliant |
| Access review | Manual | Not automated |
| Drift detection | — | Not implemented |
| Evidence export | — | Not implemented |

**Strengths**: HMAC hash-chain tamper evidence on audit events, multi-tenant isolation, RBAC +
ABAC engine already operational, NATS JetStream durable streaming with 72h retention.

**Gaps**: No automated control verification, no evidence package generation, no drift detection,
no compliance dashboard, password policy needs NIST 800-63B alignment.

---

## 9. Roadmap

| Phase | Deliverable | Priority | Effort |
|-------|------------|----------|--------|
| **1** | Audit event coverage completion + Evidence Collector API | P0 | 2 weeks |
| **2** | Automated control tests (SOC 2 mapped, 10 controls) | P1 | 3 weeks |
| **3** | Drift detection (config + policy snapshot diff) | P1 | 2 weeks |
| **4** | Compliance dashboard (real-time control status) | P2 | 3 weeks |
| **5** | OPA/Rego policy-as-code integration | P2 | 4 weeks |
| **6** | OSCAL export (machine-readable SSP + assessment results) | P3 | 3 weeks |

**Phase 1** closes the biggest gap — ensure every authz decision, privileged action, and config
change emits to the audit pipeline. **Phase 2** builds the `ControlTester` framework with 10
SOC 2 / NIST 800-53 mapped control tests. **Phase 3** adds snapshot storage and a diff engine.
**Phases 4-6** deliver visualization, policy-as-code, and machine-readable export, transforming
GGID from compliance-ready into continuously compliant.

---

### References

- [NIST OSCAL](https://pages.nist.gov/OSCAL/) — Open Security Controls Assessment Language
- [NIST SP 800-53 Rev. 5](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final) — Security and Privacy Controls
- [Open Policy Agent / Rego](https://www.openpolicyagent.org/) — Policy-as-code engine
- [AICPA SOC 2 TSC](https://www.aicpa-cima.com/topic/audit-assurance) — Trust Services Criteria
- [ISO/IEC 27001:2022](https://www.iso.org/standard/27001) — ISMS standard
