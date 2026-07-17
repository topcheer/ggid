# Identity Lifecycle Automation & HR-Driven Provisioning: JML Engine for GGID

> **Focus**: Extending GGID's existing JML (Joiner-Mover-Leaver) engine with HR system integration, dormant account detection, ghost account prevention, approval workflows, and SCIM outbound — making identity lifecycle fully automated from hire to retire.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§9).

---

## 1. Executive Summary

Identity lifecycle automation eliminates manual account management — when an employee joins, moves roles, or leaves, their identity, access, and accounts are automatically provisioned, modified, and deprovisioned based on HR data.

GGID has a **working JML engine** (`identity/server/jml_engine.go:42` — 353 lines) with:
- LifecycleEvent struct with HR event → JML trigger mapping ✅
- Joiner dashboard handler ✅
- SCIM provisioning config handler (hardcoded) ⚠️
- Lifecycle handler (222 lines) ✅
- User provisioning service (130 lines) ✅

**Gaps**: No HR system connectors, dormant detection is manual, no ghost account prevention, no approval workflows, SCIM config hardcoded.

**Recommendation**: Add HR connectors (Workday/BambooHR/SuccessFactors), dormant detection cron, ghost account reconciliation, approval workflow engine, and real SCIM outbound.

---

## 2. GGID Current State

| Component | File:Line | Status |
|-----------|-----------|--------|
| JMLEngine | `jml_engine.go:42` | ✅ Event-driven trigger mapping |
| LifecycleEvent | `jml_engine.go:18` | ✅ Struct with event type + user data |
| Lifecycle handler | `lifecycle_handler.go` | ✅ 222 lines |
| Joiner dashboard | `joiner_dashboard_handler.go` | ✅ Works |
| SCIM config | `scim_provisioning_config_handler.go` | ⚠️ Hardcoded |
| User provisioning | `user_provisioning.go` | ✅ 130 lines |
| JIT provisioning | (researched + implemented) | ✅ |
| Delegation | `delegation_pg.go` | ✅ DB-backed |

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | No HR connectors | Can't auto-provision from Workday/BambooHR |
| 2 | No dormant detection | Inactive accounts stay active indefinitely |
| 3 | No ghost account prevention | Orphaned accounts undetected |
| 4 | No approval workflows | Role changes/auto-provisioning unreviewed |
| 5 | SCIM config hardcoded | Not usable in production |
| 6 | No bulk operations | Mass import/export/offboarding manual |
| 7 | No HR event webhooks | Can't receive real-time HR changes |

---

## 4. HR-Driven Provisioning Architecture

```
HR System (Workday/BambooHR/SuccessFactors)
    │
    ├── Webhook: employee.created → GGID JML: Joiner
    ├── Webhook: employee.transferred → GGID JML: Mover
    ├── Webhook: employee.terminated → GGID JML: Leaver
    │
    ▼
GGID JML Engine
    │
    ├── Joiner: create user + assign role + provision apps (SCIM)
    ├── Mover: update role + revoke old access + assign new access
    └── Leaver: disable user + revoke all sessions + deprovision apps
    │
    ▼
SCIM 2.0 Outbound → downstream apps (Slack, GitHub, Salesforce, etc.)
```

### HR Connector Interface

```go
type HRConnector interface {
    Sync(ctx context.Context) ([]LifecycleEvent, error)
    Subscribe(webhookURL string) error  // Register webhook
}
```

### HR Systems

| HR System | API | Auth | Event Mechanism |
|-----------|-----|------|-----------------|
| Workday | SOAP/REST | OAuth 2.0 | RaaS (Reports as Services) + webhook |
| BambooHR | REST API | API Key | Webhook (new hire, termination) |
| SAP SuccessFactors | OData API | OAuth 2.0 | Event notification |
| Azure AD | Graph API | OAuth 2.0 | Delta query + change notifications |
| Generic | Webhook | Shared secret | POST events to GGID |

---

## 5. Dormant Account Detection

```
Cron job (daily):
  1. Query: users WHERE last_login_at < NOW() - INTERVAL '90 days' AND status = 'active'
  2. For each dormant user:
     a. Mark status = 'dormant'
     b. Disable active sessions
     c. Revoke OAuth tokens
     d. Notify manager
     e. After 30 more days → auto-disable (status = 'disabled')
     f. After 90 more days → auto-archive
```

### Dormancy Policy

| Stage | Inactivity | Action |
|-------|-----------|--------|
| Active → Dormant | 90 days no login | Mark dormant, revoke sessions |
| Dormant → Disabled | +30 days (120 total) | Disable account |
| Disabled → Archived | +90 days (210 total) | Archive, strip roles |

---

## 6. Ghost Account Prevention

```
Reconciliation job (weekly):
  1. Fetch all active users from GGID
  2. Fetch all active employees from HR system
  3. Find: users in GGID but NOT in HR → ghost accounts
  4. For each ghost:
     a. Alert admin (potential orphaned identity)
     b. Auto-disable if policy configured
     c. Log to audit: "ghost account detected"
```

---

## 7. Approval Workflows

| Event | Required Approval | Auto-Approve Conditions |
|-------|-------------------|------------------------|
| New account (Joiner) | Manager | HR-driven = auto-approved |
| Role change (Mover) | Manager + Role owner | Same-level move = auto |
| Privileged role | Manager + Security team | Never auto-approved |
| Access request | Resource owner | Read-only = auto for team |
| Offboarding (Leaver) | HR-confirmed | HR-driven = auto-approved |

---

## 8. Endpoint Precondition Check

### Existing (Enhance)

| Component | File:Line | Current | Target |
|----------|-----------|---------|--------|
| JMLEngine | `jml_engine.go:42` | ✅ | Add HR connectors |
| SCIM config | `scim_provisioning_config_handler.go` | Hardcoded | DB-backed |
| Lifecycle handler | `lifecycle_handler.go` | ✅ | Add dormant + ghost |

### New Components

| Component | Priority |
|-----------|----------|
| HR connector framework (Workday/BambooHR) | P0 |
| Dormant detection cron | P0 |
| Ghost account reconciliation | P1 |
| Approval workflow engine | P1 |
| SCIM outbound (real) | P1 |
| Bulk operations API | P2 |

---

## 9. Implementation Backlog with DoD

### P0 — HR Connectors + Dormant Detection (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | HR connector framework | ✅ Interface + Workday + BambooHR ✅ DB-backed config ✅ ≥3 tests | 5d |
| 2 | JML webhook receiver | ✅ POST /hr/events ✅ Webhook signature verification ✅ ≥3 tests | 3d |
| 3 | Dormant account detection | ✅ Cron job ✅ Configurable threshold ✅ Auto-stage transitions ✅ ≥3 tests | 3d |
| 4 | Replace hardcoded SCIM config | ✅ DB-backed CRUD ✅ No hardcoded ✅ ≥3 tests | 2d |

### P1 — Ghost Detection + Approvals + SCIM (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Ghost account reconciliation | ✅ Weekly diff GGID vs HR ✅ Alert + auto-disable ✅ ≥3 tests | 3d |
| 6 | Approval workflow engine | ✅ Manager approval flow ✅ Multi-step for privileged ✅ ≥3 tests | 4d |
| 7 | SCIM 2.0 outbound | ✅ Push user changes to downstream apps ✅ ≥3 tests | 3d |

### P2 — Bulk Operations + Console (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 8 | Bulk import/export API | ✅ CSV import ✅ Batch role assign ✅ ≥3 tests | 3d |
| 9 | Bulk offboarding | ✅ Select N users → disable + revoke + deprovision ✅ ≥3 tests | 2d |
| 10 | Lifecycle dashboard | ✅ JML metrics ✅ Dormant list ✅ Ghost list ✅ Approval queue | 3d |

---

## 10. Competitive Differentiation

| Feature | GGID (target) | Okta Lifecycle | Entra ID Governance | SailPoint |
|---------|---------------|----------------|---------------------|-----------|
| HR connectors | Workday/BambooHR/SF | Yes | Yes (Workday native) | Yes (broadest) |
| JML automation | ✅ Existing | Yes | Yes | Yes |
| Dormant detection | **Configurable** | Yes | Yes | Yes |
| Ghost accounts | **Reconciliation** | Partial | Yes | Yes (strongest) |
| Approval workflows | **Multi-step** | Yes | Yes | Yes (advanced) |
| SCIM outbound | **2.0** | Yes | Yes | Yes |
| Open source | **Yes** | No | No | No |

---

## References

- [SCIM 2.0 Protocol (RFC 7644)](https://datatracker.ietf.org/doc/html/rfc7644) — Provisioning protocol
- [Workday API](https://community.workday.com/api) — HR system integration
- [BambooHR API](https://documentation.bamboohr.com/) — HR webhooks
- [Okta Lifecycle Management](https://help.okta.com/en-us/Content/Topics/Provisioning/lifecycle/lifecycle-workflows.htm) — Reference
- [Microsoft Entra ID Governance](https://learn.microsoft.com/en-us/entra/id-governance/) — Reference
- [SailPoint IdentityNow](https://www.sailpoint.com/products/identity-security-cloud) — Enterprise IGA
- [GGID JML Engine](../services/identity/internal/server/jml_engine.go) — At line 42
- [GGID Lifecycle Handler](../services/identity/internal/server/lifecycle_handler.go) — 222 lines
- [GGID SCIM Config](../services/identity/internal/server/scim_provisioning_config_handler.go) — Hardcoded
- [GGID JIT Provisioning](./jit-user-provisioning.md) — JIT research
- [GGID Identity Orchestration](./identity-orchestration-journeys.md) — Journey research
