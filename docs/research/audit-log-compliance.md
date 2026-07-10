# Audit Log Compliance: Requirements, Integrity, and SIEM Integration

## 1. Overview

Audit logging in an Identity and Access Management (IAM) system answers four
questions for every action: **who** did **what**, **when**, and **from where**.
Modern compliance frameworks mandate these records as non-negotiable evidence
of security controls.

GGID uses **NATS JetStream** for asynchronous audit event distribution:
microservices publish events to the `audit.events` subject via
`pkg/audit.Publisher`, and the dedicated **Audit Service** consumes, persists,
and exposes them through a gRPC query API backed by PostgreSQL with monthly
range partitions.

This document covers:
- Compliance requirements across six major frameworks
- A recommended audit event taxonomy for IAM
- Log integrity and tamper-detection strategies (hash chain, Merkle tree, WORM)
- Tiered retention architecture
- SIEM integration patterns and an implementation sketch
- Gap analysis of GGID's current implementation
- A phased implementation roadmap

---

## 2. Compliance Requirements by Framework

### SOC 2 Type II
- **CC7.2**: Monitor system performance and security; detect anomalous activity.
- **Required events**: All authentication events (login, logout, MFA), admin
  actions, configuration changes, data access.
- **Retention**: Minimum 1 year (auditors expect at least 12 months online).
- **Access**: Auditors require read-only access; logs must be tamper-evident.

### ISO 27001
- **A.12.4.1**: Event logging — record user activities, exceptions, and
  security events.
- **Required events**: Security events, access control changes, privilege
  escalation, system configuration changes.
- **Retention**: Typically 12 months (no explicit mandate, but auditors expect it).
- **Log protection**: Tamper-proof, access-controlled, clock-synchronized.

### GDPR
- **Article 30**: Records of processing activities (ROPA).
- **Article 33**: Breach notification within 72 hours — requires an audit trail
  to determine scope and timeline.
- **Required events**: Data access logs, consent changes, data exports/deletions.
- **Retention**: Not specified, but logs must support data subject rights.
  The "legal obligation" basis (Art. 6(1)(c)) permits retention despite
  erasure requests.

### HIPAA
- **164.312(b)**: Audit controls — record and examine activity in systems
  containing electronic PHI.
- **Required events**: All PHI access, authentication events, admin actions.
- **Retention**: 6 years from creation or last effective date.

### SOX (Sarbanes-Oxley)
- **Section 404**: Internal controls over financial reporting.
- **Required events**: Financial system access, privileged account activity,
  configuration changes to financial systems.
- **Retention**: 7 years.

### PCI DSS v4.0
- **Requirement 10**: Track and monitor all access to network resources and
  cardholder data.
- **Required events**: All access to cardholder data, admin actions, failed
  authentications, changes to audit trails.
- **Retention**: 1 year minimum — at least 3 months immediately available
  online, remainder archived.

### Summary Matrix

| Framework | Required Events | Min Retention | Key Requirement |
|-----------|----------------|---------------|-----------------|
| SOC 2 Type II | Auth, admin, config changes | 1 year | Tamper-evident, auditor read access |
| ISO 27001 | Security events, access changes | ~12 months | Clock sync, access-controlled |
| GDPR | Data access, consent changes | Not specified | Breach detection <72h, ROPA |
| HIPAA | PHI access, auth, admin | 6 years | Examination capability |
| SOX | Financial system access, privileged | 7 years | Internal control evidence |
| PCI DSS v4.0 | Cardholder data access, auth | 1 year | 3mo online + 1yr archive |

---

## 3. Audit Event Taxonomy for IAM

GGID's `audit.Event.Action` field uses a dotted convention (e.g.,
`user.login`). The following taxonomy covers the event categories that
compliance frameworks expect.

### Authentication Events
| Action | Description | SOC2 | HIPAA | PCI |
|--------|-------------|------|-------|-----|
| `user.login` | Login attempt (success/failure) | Yes | Yes | Yes |
| `user.logout` | Session termination | Yes | Yes | — |
| `mfa.challenge` | MFA prompt issued | Yes | Yes | — |
| `mfa.success` / `mfa.failed` | MFA result | Yes | Yes | — |
| `token.issued` | JWT or OAuth token issued | Yes | — | Yes |
| `token.refreshed` | Refresh token used | Yes | — | — |
| `token.revoked` | Token manually revoked | Yes | Yes | — |
| `session.expired` | Session timed out | Yes | — | — |

### Identity Management Events
| Action | Description |
|--------|-------------|
| `user.created` / `user.deleted` | User lifecycle |
| `user.disabled` / `user.enabled` | Account state change |
| `role.assigned` / `role.revoked` | Privilege change |
| `group.joined` / `group.left` | Membership change |
| `password.changed` / `password.reset` | Credential events |
| `mfa.enrolled` / `mfa.removed` | MFA lifecycle |

### Administrative Events
`config.changed`, `tenant.created`, `api_key.created`, `api_key.revoked`,
`saml.config_updated`, `oauth.client_registered`, `policy.changed`.

### Data Access Events
`data.exported`, `data.deleted`, `pii.accessed`, `consent.granted`,
`consent.revoked`.

### Security Events
`rate_limit.exceeded`, `account.lockout`, `suspicious_activity_detected`.

---

## 4. Log Integrity and Tamper Detection

Compliance frameworks require logs to be tamper-evident or tamper-proof.
GGID's `domain.AuditEvent` already has a `Hash string` field (labeled
"HMAC chain hash for tamper detection"), but it is **never populated** in the
current Insert path. This section describes how to activate it.

### Hash Chain

Each entry includes a hash of the previous entry, forming a chain:

```
entry_n.hash = SHA256(entry_n.data || entry_{n-1}.hash)
```

Tampering with any entry breaks the chain. A genesis entry uses a fixed
known hash.

### Go Implementation Sketch

```go
package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// AuditChain computes and verifies a hash chain over audit events.
type AuditChain struct {
	secret []byte // HMAC key for additional protection
}

// ComputeHash returns the HMAC-SHA256 chain hash for an event given
// the previous entry's hash. Use "genesis" for the first entry.
func (c *AuditChain) ComputeHash(event *Event, prevHash string) string {
	payload, _ := json.Marshal(event)
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(prevHash))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyChain walks the chain from the first entry and returns true if
// every hash matches.
func (c *AuditChain) VerifyChain(events []*Event) bool {
	prev := "genesis"
	for _, e := range events {
		expected := c.ComputeHash(e, prev)
		if !hmac.Equal([]byte(expected), []byte(e.Hash)) {
			return false
		}
		prev = e.Hash
	}
	return true
}
```

**Trade-off**: A sequential hash chain creates a single point of failure — if
one entry is missing the chain breaks. In practice, per-tenant chains with
gap-tolerant verification are more resilient.

### Merkle Tree Alternative

Batch entries into a Merkle tree and publish the root periodically. This is
more efficient for verification (O(log N) proof per entry) and is the
approach used by Certificate Transparency logs.

### WORM Storage (Write Once Read Many)

Cloud-native WORM backends prevent modification after writing:
- **AWS S3 Object Lock** (compliance mode) or **GCP Bucket Lock**
- Combined with hash chain: provides cryptographic + physical immutability

---

## 5. Log Retention Strategy

### Tiered Storage

| Tier | Period | Storage | Query Speed |
|------|--------|---------|-------------|
| Hot | 0-90 days | PostgreSQL (current) | <100ms |
| Warm | 90-365 days | Compressed JSON in object storage (S3/GCS) | Seconds |
| Cold | 1-7 years | Archive storage (Glacier/Archive) | Minutes-hours |

### Automated Lifecycle

- **Cron job**: Move logs older than 90 days to warm tier; archive after 1 year.
- **NATS TTL**: GGID currently sets `MaxAge: 72h` on the JetStream stream.
  This is a message-broker buffer, not the retention policy — the Audit Service
  persists to PostgreSQL before messages expire.
- **PostgreSQL partitions**: Monthly range partitions exist for 2025 only.
  A `pg_partman` extension or cron job should auto-create future partitions
  and detach/drop expired ones.
- **Legal hold**: Suspend automated deletion for specific records during
  litigation. `DeleteOlderThan` exists in the repository but is not wired to
  a scheduled job.

### PII in Audit Logs

Audit logs naturally contain PII: user emails, IP addresses, actor names.

- **Mitigation**: Hash or tokenize PII fields before logging.
- **Trade-off**: Anonymization reduces forensic value (can't correlate by email).
- **GDPR tension**: "Right to erasure" (Art. 17) vs. legal retention obligation.
  Article 6(1)(c) (legal obligation) provides the lawful basis to retain audit
  logs even when a user requests deletion of their personal data.

---

## 6. SIEM Integration

### SIEM Platforms

Splunk, Elastic SIEM, IBM QRadar, Sumo Logic, Datadog — all aggregate,
correlate, and alert on security events across infrastructure.

### Integration Patterns

| Pattern | Mechanism | Latency | Complexity |
|---------|-----------|---------|------------|
| Push | GGID sends to SIEM via HTTP webhook or syslog | Real-time | Low |
| Pull | SIEM queries GGID audit API on schedule | Minutes | Low |
| Streaming | NATS → Kafka → SIEM connector | Near real-time | Medium |

### GGID NATS-to-SIEM Bridge

The bridge subscribes to the `audit.events` NATS subject, transforms each
event to a SIEM-compatible format, and publishes to the SIEM endpoint:

```
┌──────────┐    NATS     ┌──────────────┐    HTTP/Syslog    ┌──────────┐
│ GGID     │──audit.────▶│  SIEM Bridge │──────────────────▶│ Splunk / │
│ Services │   events    │  (Go daemon) │   CEF / LEEF      │ Elastic  │
└──────────┘             └──────────────┘                   └──────────┘
```

```go
// Format adapter interface
type SIEMFormatter interface {
	Format(event *audit.Event) ([]byte, error)
}

// CEFFormat renders events in Common Event Format (used by Splunk, ArcSight).
type CEFFormat struct{}

func (f *CEFFormat) Format(e *audit.Event) ([]byte, error) {
	cef := fmt.Sprintf(
		"CEF:0|GGID|IAM|1.0|%s|%s|%s|src=%s suser=%s act=%s rt=%s",
		e.Action, e.Action, severity(e.Result),
		e.IPAddress, e.ActorName, e.Action, e.CreatedAt.Format(time.RFC3339),
	)
	return []byte(cef), nil
}
```

### Alert Rules

| Rule | Condition | Severity |
|------|-----------|----------|
| Brute force | 5+ `user.login` failures from same IP in 5 min | High |
| Off-hours admin | Admin action between 22:00-06:00 local | Medium |
| Mass deletion | 10+ `user.deleted` in 1 hour | Critical |
| Privilege escalation | `role.assigned` with admin-level role | High |
| Token anomaly | `token.issued` count spike >3x baseline | Medium |

---

## 7. GGID Current Audit Implementation

Based on source code examination:

**Architecture**: Services publish `audit.Event` structs (JSON) to NATS
JetStream subject `audit.events`. The Audit Service runs a durable consumer
that fetches batches, decodes, and persists each event to PostgreSQL.
Events are queryable via gRPC (`ListEvents`, `GetEvent`) with filtering by
tenant, actor, action, resource type, result, and time range.

**Storage**: PostgreSQL with monthly range partitions (`audit_events_2025_01`
through `_2025_12`). The `ip_address` column uses `inet` type; `metadata`
is JSONB. The repository exposes `DeleteOlderThan` for manual cleanup.

**Domain model**: `AuditEvent` includes `Hash string` for HMAC chain
tamper detection, but the field is **not populated** by the consumer or
repository Insert path.

| Capability | Current State | Compliance Gap |
|-----------|---------------|----------------|
| Event coverage | Auth, identity, basic admin actions | Missing MFA lifecycle, consent, data access events |
| Hash chain / tamper detection | Field exists, never populated | **High** — no integrity guarantee |
| NATS retention | `MaxAge: 72h`, `MaxBytes: 1GB` | None (buffer only, not retention policy) |
| PostgreSQL retention | Manual `DeleteOlderThan`, no scheduler | **High** — no automated retention enforcement |
| Partition automation | 2025 partitions only, manual | **Medium** — will fail for 2026+ without automation |
| SIEM export | Not implemented | **High** — no external alerting/correlation |
| Immutable storage | None | **Medium** — logs are mutable in PostgreSQL |
| Query API | gRPC with filtering + stats dashboard | **Adequate** for current scope |
| PII handling | Raw PII stored (email, IP) | **Medium** — GDPR erasure tension |
| Webhook delivery | HMAC-signed HTTP POST per event type | **Adequate** — could feed SIEM via webhook |

---

## 8. Implementation Roadmap

| Phase | Task | Priority | Effort | Outcome |
|-------|------|----------|--------|---------|
| 1 | **Event taxonomy completion** — add MFA lifecycle, consent, data access events across all services | P0 | 2-3 weeks | Full compliance event coverage |
| 2 | **Hash chain activation** — populate `AuditEvent.Hash` at consume time, add `VerifyChain` API | P1 | 1 week | Tamper-evident log integrity |
| 3 | **SIEM bridge** — NATS subscriber daemon with CEF/LEEF/syslog adapters, configurable endpoint | P1 | 2 weeks | External monitoring and alerting |
| 4 | **Tiered retention** — cron scheduler for hot→warm→cold migration, `pg_partman` for auto-partitioning | P2 | 2-3 weeks | Automated multi-year retention |
| 5 | **WORM storage** — S3 Object Lock export for archived logs, periodic snapshot to immutable bucket | P2 | 1-2 weeks | Physical immutability |
| 6 | **Compliance report generator** — pre-built SOC 2 / HIPAA / PCI DSS audit trail exports | P3 | 2 weeks | Auditor self-service reports |

### Effort Summary

- **P0** (blocking compliance): Phase 1 — ~3 weeks
- **P1** (critical for audit-readiness): Phases 2-3 — ~3 weeks
- **P2** (mature compliance posture): Phases 4-5 — ~4 weeks
- **P3** (convenience): Phase 6 — ~2 weeks

Total estimated effort: **10-12 engineer-weeks** for full compliance-grade
audit infrastructure.
