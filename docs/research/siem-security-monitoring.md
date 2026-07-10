# SIEM Integration & Security Monitoring for GGID

> Patterns for forwarding GGID audit events to SIEM platforms, building
> correlation rules, mapping detections to MITRE ATT&CK, and designing
> security dashboards.

---

## 1. Overview

SIEM (Security Information and Event Management) platforms aggregate
security logs, correlate events in real time, and alert on suspicious
patterns. For an identity provider like GGID, SIEM integration is
critical — every authentication, role change, and policy edit is a
potential indicator of compromise.

GGID's audit pipeline (NATS JetStream → Audit Service → PostgreSQL) is
the authoritative event source. This document covers:

- SIEM integration patterns (CEF, LEEF, JSON, syslog)
- NATS → SIEM bridge architecture
- Correlation rules for common identity attacks
- MITRE ATT&CK technique mapping
- Pre-built dashboard designs

**Key platforms:** Splunk (HEC + CEF), Elastic SIEM (JSON + Kibana),
Datadog (JSON API), Sumo Logic (HTTP collector), IBM QRadar (LEEF/syslog).

Compliance frameworks and hash-chain integrity are covered in
`docs/research/audit-log-compliance.md`.

---

## 2. GGID Audit Event Catalog

GGID's `audit.Event` (in `pkg/audit/publisher.go`) uses dot-notation
actions (e.g. `user.login`) with results (`success | failure | denied`).
The following catalog maps all SIEM-relevant actions.

### Authentication

| Action | Result | Severity | SIEM Use Case | MITRE |
|--------|--------|----------|---------------|-------|
| `user.login` | success | low | Session tracking, geo | T1078 |
| `user.login` | failure | medium | Brute force detection | T1110 |
| `user.logout` | info | info | Session lifecycle | — |
| `mfa.challenge` | — | low | MFA fatigue detection | T1621 |
| `mfa.verify` | success | info | MFA enrollment metrics | — |
| `mfa.verify` | failure | medium | MFA bypass attempts | T1110 |
| `token.issued` | success | low | Token anomaly | T1528 |
| `token.refreshed` | success | info | Token replay detection | — |
| `token.revoked` | success | medium | Forced logout (IR) | — |
| `session.revoked` | success | medium | Incident response | — |

### Identity & Admin

| Action | Severity | MITRE | Action | Severity | MITRE |
|--------|----------|-------|--------|----------|-------|
| `user.create` | medium | T1136 | `config.changed` | high | T1556 |
| `user.update` | medium | T1098 | `api_key.created` | medium | T1552 |
| `user.delete` | high | T1531 | `policy.changed` | high | — |
| `role.assign` | high | T1098 | `tenant.created` | medium | — |
| `role.revoke` | medium | — | `oauth_client.registered` | high | — |
| `group.modify` | medium | T1098 | `api_key.revoked` | medium | — |

### Security

| Action | Severity | MITRE | Trigger |
|--------|----------|-------|---------|
| `ratelimit.exceeded` | medium | T1110 | Rate limiter threshold hit |
| `account.lockout` | high | T1110 | 10+ failed logins |
| `suspicious.ip` | high | — | Threat intel match |
| `bruteforce.detected` | critical | T1110 | Correlation rule fires |
| `privilege.escalation` | critical | T1098 | Unauthorized admin role |

---

## 3. Export Formats

### CEF (Common Event Format) — Splunk, QRadar, ArcSight

```
CEF:0|GGID|AuthService|1.0|1001|Login Failed|6|src=192.168.1.50 suser=admin@example.com act=login_failed dtz=2025-01-15T10:30:00Z
```

```go
func formatCEF(e audit.Event) string {
    sev := cefSeverity(e.Result)
    name := strings.ReplaceAll(e.Action, ".", "_")
    ext := fmt.Sprintf("src=%s suser=%s act=%s dtz=%s",
        e.IPAddress, e.ActorName, name,
        e.CreatedAt.UTC().Format(time.RFC3339))
    return fmt.Sprintf("CEF:0|GGID|AuthService|1.0|%s|%s|%d|%s",
        signatureID(e.Action), name, sev, ext)
}
```

### LEEF (Log Event Extended Format) — IBM QRadar

```
LEEF:2.0|GGID|AuthService|1.0|1001|^src=192.168.1.50\tusrName=admin\taction=login_failed
```

### JSON (Universal) — Elastic, Datadog, Sumo Logic

GGID already emits JSON natively via `json.Marshal(event)`. No
transformation needed — SIEM ingests directly.

### Syslog Transport (RFC 5424 / 5425)

```
<134>1 2025-01-15T10:30:00Z ggid-auth auth 1001 - [ggid@1.0 action="user.login" result="failure"] Login failed
```

- **UDP 514** — fire-and-forget, no delivery guarantee
- **TCP 6587** — reliable delivery with backpressure
- **TLS (RFC 5425)** — encrypted syslog, recommended for production

---

## 4. NATS → SIEM Bridge Architecture

```
┌─────────────┐     ┌──────────┐     ┌──────────────────┐
│  GGID Auth  │────▶│   NATS   │────▶│  Audit Service   │
│  GGID Policy│     │JetStream │     │  (PostgreSQL)    │
│  GGID Org   │     │  AUDIT   │     └──────────────────┘
└─────────────┘     │  stream  │            │
                    └────┬─────┘            │ DB query
                         │ subscribe        │
                         ▼                  ▼
                  ┌──────────────┐   ┌──────────────┐
                  │ SIEM Bridge  │   │  Console     │
                  │              │   │  Audit Page  │
                  │ CEF/LEEF/JSON│   └──────────────┘
                  │ HTTP/Syslog  │
                  └──────┬───────┘
            ┌────────────┼────────────┐
            ▼            ▼            ▼
       ┌────────┐  ┌─────────┐  ┌─────────┐
       │ Splunk │  │ Elastic │  │ Datadog │
       └────────┘  └─────────┘  └─────────┘
```

The bridge subscribes to `audit.events` as a **separate consumer** — it
does not interfere with the Audit Service's durable consumer.

### Go Implementation

```go
type Formatter interface {
    Format(e audit.Event) ([]byte, error)
}
type Transport interface {
    Send(ctx context.Context, data [][]byte) error
}

type SIEMBridge struct {
    nc        *nats.Conn
    subject   string
    formatter Formatter
    transport Transport
    batch     [][]byte
    batchSize int
    flushInt  time.Duration
}

func (b *SIEMBridge) Start(ctx context.Context) error {
    ticker := time.NewTicker(b.flushInt)
    defer ticker.Stop()

    _, err := b.nc.Subscribe(b.subject, func(msg *nats.Msg) {
        var event audit.Event
        if err := json.Unmarshal(msg.Data, &event); err != nil {
            return
        }
        formatted, _ := b.formatter.Format(event)
        b.batch = append(b.batch, formatted)
        if len(b.batch) >= b.batchSize {
            b.flush(ctx)
        }
    })

    for {
        select {
        case <-ctx.Done():
            b.flush(ctx)
            return nil
        case <-ticker.C:
            b.flush(ctx)
        }
    }
}
```

**Transports:** `HTTPTransport` (Splunk HEC, Datadog API), `SyslogTransport`
(RFC 5424/5425), `KafkaTransport` (log aggregation pipeline). Ring buffer
(10K events) retains undelivered events during SIEM downtime.

---

## 5. Correlation Rules

### Brute Force Detection

> 5+ `user.login` (failure) from same IP within 60 seconds

```splunk
index=ggid action="user.login" result="failure"
| bucket _time span=1m | stats count by ip_address, _time | where count > 5
```

**GGID action:** Gateway rate-limiter auto-blocks IP for 15 min;
`account.lockout` fires after 10 failed attempts on a single user.

### Impossible Travel

> Successful login from US, then JP within a window impossible for travel

```splunk
index=ggid action="user.login" result="success"
| iplocation ip_address | stats values(Country) as c, range(_time) as span by actor_name
| where mvcount(c) > 1 AND span < 3600
```

**GGID action:** trigger step-up MFA; require TOTP before session is established.

### Credential Stuffing

> 20+ `user.login` (failure) with **different** usernames from same IP in 5 min

```splunk
index=ggid action="user.login" result="failure"
| bucket _time span=5m | stats dc(actor_name) as users by ip_address, _time
| where users > 20
```

**GGID action:** aggressive IP rate-limit (1 req/min); emit `bruteforce.detected`.

### Privilege Escalation

> `role.assign` with admin-level role outside business hours

```splunk
index=ggid action="role.assign"
| eval after_hours=if(datehour < 8 OR datehour > 18, 1, 0)
| where after_hours=1 OR result="denied"
```

**GGID action:** alert security team; require approval workflow.

### Account Takeover Chain

> `password.change` + `mfa.disable` + `user.login` (new device) within 10 min

**GGID action:** lock account immediately; notify user via out-of-band
channel; require admin verification to unlock.

### Correlation Rule Summary

| Rule | Trigger | Severity | MITRE | GGID Action |
|------|---------|----------|-------|-------------|
| Brute Force | >5 fails/IP/60s | high | T1110 | Auto-block IP |
| Impossible Travel | 2 geo-logins impossible time | high | T1078 | Step-up MFA |
| Credential Stuffing | >20 users/IP/5min | critical | T1110 | Aggressive rate-limit |
| Privilege Escalation | Admin role off-hours | critical | T1098 | Alert + approval |
| Account Takeover | Pwd+MFA+new device chain | critical | T1098 | Lock + notify |
| New Device/Location | Unknown device + new geo | medium | T1078 | Step-up MFA |

---

## 6. MITRE ATT&CK Mapping

| Technique | Name | GGID Events | Detection | Severity |
|-----------|------|-------------|-----------|----------|
| T1110 | Brute Force | `user.login` (failure) | >5 fails/IP/60s | high |
| T1078 | Valid Accounts | `user.login` (success) | Impossible travel, new device | high |
| T1098 | Account Manipulation | `role.assign`, `group.modify` | Off-hours or unauthorized | critical |
| T1136 | Create Account | `user.create` | Unauthorized creator or source | medium |
| T1531 | Account Access Removal | `user.disable`, `user.delete` | Mass disable | high |
| T1621 | MFA Request Generation | `mfa.challenge` | >10 challenges/user/5min | medium |
| T1556 | Modify Auth Process | `config.changed` | Any auth config change | critical |
| T1552 | Unsecured Credentials | `api_key.created` | Key without expiration | medium |
| T1528 | Steal Application Access Token | `token.issued` | Token from new device + geo | high |

### Embedding MITRE Metadata in Events

```go
event := audit.NewEvent("user.login", "failure", tenantID, actorID)
event.IPAddress = clientIP
event.Metadata = map[string]any{
    "mitre_technique": "T1110",
    "mitre_tactic":    "credential-access",
    "risk_score":      7,
}
```

---

## 7. SIEM Dashboard Design

| Panel | Viz | Splunk SPL | Elastic KQL |
|-------|-----|-----------|-------------|
| Login Trend | timechart | `action="user.login" \| timechart count by result` | `event.action:"user.login"` grouped by `event.result` |
| Top Failed IPs | bar chart | `result="failure" \| top ip_address` | `event.result:"failure"` group by `source.ip` |
| Auth by Country | world map | `\| iplocation \| geostats count by Country` | geoip enrichment + map viz |
| MFA Enrollment | gauge | `action="mfa.verify" \| stats dc(actor_name)` | `event.action:"mfa.verify"` unique count |
| Privileged Actions | table | `action IN ("role.assign","policy.changed")` | `event.action:(role.assign OR policy.changed)` |
| Security Alerts | real-time table | `severity IN ("high","critical")` | `severity:(high OR critical)` |

**Datadog equivalent:** `@action:user.login @result:failure group by @ip_address`

### Dashboard Layout

```
┌──────────────────────┬───────────────────────────┐
│ Login Trend          │ Top Failed IPs            │
│ (timechart)          │ (bar chart)               │
├──────────────────────┼───────────────────────────┤
│ Geo Map              │ MFA Enrollment Gauge      │
│ (world map)          │ (per tenant)              │
├──────────────────────┴───────────────────────────┤
│ Privileged Actions Timeline (table)               │
├───────────────────────────────────────────────────┤
│ Real-Time Security Alerts (streaming table)       │
└───────────────────────────────────────────────────┘
```

---

## 8. GGID Current SIEM Capability

| Capability | Current State | Gap |
|-----------|---------------|-----|
| Event emission | `user.login`, `user.create`, `role.assign` via `audit.PublishAsync` | Missing: `mfa.*`, `token.*`, `ratelimit.*`, `bruteforce.*` — many auth flows not instrumented |
| SIEM export | NATS → PostgreSQL only | No SIEM bridge; no syslog/CEF/LEEF |
| Webhook support | Auth service `post-login` webhooks (HMAC-signed) | Limited event types; no batch; no retry queue |
| Correlation rules | None built-in | All detection in SIEM (manual) |
| MITRE mapping | Not present | Events lack technique metadata |
| Dashboards | Console audit page (basic table) | No Splunk/Elastic/Datadog templates |

**Key gaps:** only ~15% of the event catalog in Section 2 is currently
emitted. No NATS subscriber that formats and forwards events exists.
The webhook system is closest but is event-type-limited.

---

## 9. Implementation Roadmap

| Phase | Scope | Effort |
|-------|-------|--------|
| 1. JSON Export | Extend webhooks to all event types; batch POST with HMAC | 1 sprint |
| 2. CEF/LEEF | Add format options; syslog transport (RFC 5424/5425) | 1 sprint |
| 3. Correlation Rules | Brute force, impossible travel, credential stuffing | 1-2 sprints |
| 4. MITRE Metadata | Add technique/tactic/risk to event `Metadata` map | 3 days |
| 5. Dashboard Templates | Splunk JSON, Elastic NDJSON, Datadog JSON | 3 days |

**Total: ~4 sprints (8 weeks).** Phases 1-2 can run in parallel with 3.

---

*See also: `docs/research/audit-log-compliance.md` for compliance
frameworks and hash-chain integrity.*
