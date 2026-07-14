# Compliance Automation

Evidence collection pipeline, continuous control monitoring, framework mapping automation, drift detection, remediation triggers, and audit readiness dashboard.

## Overview

Manual compliance is slow, error-prone, and expensive. GGID automates evidence collection and continuous monitoring to maintain always-audit-ready state.

## Evidence Collection Pipeline

```
Services → Metrics/Logs → Evidence Collector → Evidence Store → Audit-Ready Export
                              │
                              ├── Config snapshots
                              ├── Access reviews
                              ├── Policy decisions
                              ├── Security scans
                              └── Incident reports
```

### Automated Evidence Sources

| Evidence Type | Source | Frequency |
|--------------|--------|-----------|
| Access control config | `services/policy` config | Hourly snapshot |
| User access logs | `services/audit` events | Real-time |
| Encryption status | `pkg/crypto` config | Daily check |
| Vulnerability scan | External scanner | Weekly |
| Code review records | Git history + CI | Per merge |
| Change management | CI/CD pipeline logs | Per deploy |
| Backup verification | Backup job logs | Daily |
| Incident reports | Incident system | Per incident |

## Continuous Control Monitoring

```go
type ControlMonitor struct {
    controls []Control
}

type Control struct {
    ID          string
    Framework   string  // SOC2, ISO27001, GDPR
    Description string
    Check       func() ControlResult
    Schedule    string  // cron
}

func (m *ControlMonitor) Run() {
    for _, control := range m.controls {
        result := control.Check()

        switch result.Status {
        case "pass":
            storeEvidence(control, result)
        case "fail":
            triggerRemediation(control, result)
            alert.Send("control_failed", control, result)
        case "warning":
            storeEvidence(control, result)
            log.Warn("control warning", control)
        }
    }
}
```

### Built-in Control Checks

| Control | Check | Pass Criteria |
|---------|-------|---------------|
| CC6.1 (SOC2) | RLS enabled on all tenant tables | `pg_tables WHERE rowsecurity = true` |
| CC6.2 (SOC2) | MFA enforced for admins | `admin_users WHERE mfa_enabled = true` |
| CC6.7 (SOC2) | TLS 1.3 enforced | Config check: `min_version >= 1.3` |
| CC6.8 (SOC2) | Encryption at rest enabled | `pgcrypto extension installed` |
| A.6.3 (ISO) | JWT algorithm restricted | `allowed_algs = [RS256, ES256]` |
| Art. 32 (GDPR) | Data retention enforced | `retention_job_last_run < 25h` |

## Framework Mapping Automation

```yaml
control_mapping:
  "RLS_ENABLED":
    soc2: "CC6.1"
    iso27001: "A.8.3"
    gdpr: "Art. 32"
    hipaa: "164.312(a)(1)"

  "MFA_ENFORCED":
    soc2: "CC6.2"
    iso27001: "A.8.5"
    gdpr: "Art. 32(1)(b)"
    hipaa: "164.312(d)"

  "ENCRYPTION_AT_REST":
    soc2: "CC6.8"
    iso27001: "A.6.3"
    gdpr: "Art. 32(1)(a)"
    hipaa: "164.312(a)(2)(iv)"
```

### Automated Cross-Framework Report

```bash
GET /api/v1/admin/compliance/status?frameworks=soc2,iso27001,gdpr
# → {
#   "soc2": {"total": 12, "passing": 11, "failing": 1, "last_check": "2025-01-15T03:00:00Z"},
#   "iso27001": {"total": 10, "passing": 10, "failing": 0},
#   "gdpr": {"total": 11, "passing": 11, "failing": 0}
# }
```

## Drift Detection

Detects configuration drift from approved baseline:

```go
func detectDrift() []Drift {
    baseline := loadApprovedBaseline()
    current := captureCurrentConfig()

    drifts := []Drift{}

    // Compare each config item
    for key, approved := baseline {
        actual := current[key]
        if approved != actual {
            drifts = append(drifts, Drift{
                Config: key,
                Approved: approved,
                Actual: actual,
                Detected: time.Now(),
            })
        }
    }

    return drifts
}
```

### Drift Examples

| Config | Approved | Drift (Alert) |
|--------|----------|---------------|
| JWT TTL | 900s | 3600s (someone increased) |
| Rate limit | 100/min | 0 (disabled) |
| Allowed origin | *.ggid.dev | * (opened up) |
| mTLS required | true | false (disabled) |

## Remediation Triggers

```yaml
remediation:
  RLS_DISABLED:
    trigger: "control_check failed"
    action:
      - "Re-enable RLS: ALTER TABLE ... ENABLE ROW LEVEL SECURITY"
      - "Alert: security team"
      - "Audit log: drift remediated"

  JWT_ALG_DRIFT:
    trigger: "allowed_algs contains 'none'"
    action:
      - "Revert config to approved baseline"
      - "Restart affected services"
      - "Page on-call (P0 security event)"

  MFA_NOT_ENFORCED:
    trigger: "admin without MFA detected"
    action:
      - "Force MFA enrollment for admin"
      - "Suspend admin access until enrolled"
      - "Notify security team"
```

## Audit Readiness Dashboard

```
┌─────────────────────────────────────────────┐
│         COMPLIANCE DASHBOARD                 │
│                                              │
│  SOC2:     ████████████░ 92% (11/12)       │
│  ISO27001: ██████████████ 100% (10/10)     │
│  GDPR:     ██████████████ 100% (11/11)     │
│  HIPAA:    ████████████░ 88% (7/8)         │
│  PIPL:     █████████████░ 90% (9/10)       │
│                                              │
│  Last evidence: 2h ago                       │
│  Drift alerts: 1 (JWT TTL)                   │
│  Failed controls: 1                          │
│  Upcoming audits: SOC2 Type II (Mar 2025)    │
│                                              │
│  [Export Evidence]  [View Drift]  [Remediate]│
└─────────────────────────────────────────────┘
```

### Evidence Export

```bash
# Generate audit evidence package
POST /api/v1/admin/compliance/export
{
  "framework": "soc2",
  "period": "2025-Q1",
  "format": "zip"
}
# → ZIP with:
#   - control_status.csv
#   - evidence/screenshots/
#   - evidence/logs/
#   - evidence/configs/
#   - framework_mapping.json
#   - attestation_letter.pdf
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Control pass rate | >95% | <90% → investigate |
| Drift detection count | 0 | Any → unauthorized change |
| Evidence collection lag | <1h | >4h → pipeline broken |
| Remediation success | 100% | <100% → manual intervention |

## See Also

- [Compliance Framework Mapping](compliance-framework-mapping.md)
- [Audit Log Architecture](audit-log-architecture.md)
- [Policy Hot Reload](policy-hot-reload.md)
- [Identity Threat Detection](identity-threat-detection.md)
