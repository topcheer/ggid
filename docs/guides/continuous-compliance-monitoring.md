# Continuous Compliance Monitoring Guide (KB-280)

## Overview

GGID's Continuous Compliance Monitoring (CCM) engine automatically evaluates security controls against compliance frameworks (SOC 2, ISO 27001, HIPAA, DORA) on a scheduled basis. Results are persisted to PostgreSQL for audit evidence and dashboard display.

## How It Works

```
Scheduler triggers → CCM engine runs 15 controls → Results stored to ccm_results → Dashboard updates
```

## 15 Compliance Controls

| Control ID | Category | What It Checks |
|-----------|----------|----------------|
| `MFA_ENFORCEMENT` | Access Control | All admin accounts have MFA enabled |
| `PASSWORD_POLICY` | Access Control | Password policy meets minimum requirements |
| `BREAK_GLASS_REVIEW` | Privileged Access | Break-glass activations reviewed within 24h |
| `JIT_EXPIRY` | Privileged Access | No JIT elevations exceeding 8h duration |
| `SESSION_TIMEOUT` | Session Mgmt | Idle session timeout < 30 minutes |
| `AUDIT_INTEGRITY` | Audit | Hash-chain verification passes |
| `AUDIT_RETENTION` | Audit | Audit logs retained per policy (90d+) |
| `ROLE_HYGIENE` | RBAC | No dormant roles (>90d unused) |
| `ORPHAN_ACCOUNTS` | IAM | No accounts without owner/manager |
| `DEPARTED_USERS` | IAM | No active accounts for departed employees |
| `PRIVILEGE_CREEP` | RBAC | No users with accumulated excess permissions |
| `SOD_VIOLATIONS` | SoD | No unresolved segregation-of-duties violations |
| `API_KEY_ROTATION` | Secrets | No API keys older than rotation policy |
| `CERTIFICATE_VALIDITY` | PKI | No certificates expiring within 30 days |
| `DLP_EGRESS_RULES` | Data Protection | DLP egress policies active and monitored |

## API Endpoints

### Run Full Scan

```http
POST /api/v1/audit/ccm/scan
X-Tenant-ID: <tenant-uuid>
```

Triggers evaluation of all 15 controls. Returns immediately with scan ID.

### Get Latest Results

```http
GET /api/v1/audit/ccm/latest
X-Tenant-ID: <tenant-uuid>
```

Returns the most recent result for each control.

**Response:**
```json
{
  "results": [
    {
      "control_id": "MFA_ENFORCEMENT",
      "control_name": "MFA Enforcement",
      "category": "Access Control",
      "status": "pass",
      "metric_value": 100,
      "threshold": 100,
      "threshold_dir": ">=",
      "details": "All 12 admin accounts have MFA enabled",
      "checked_at": "2026-07-18T10:00:00Z"
    },
    {
      "control_id": "SOD_VIOLATIONS",
      "control_name": "SoD Violations",
      "category": "SoD",
      "status": "fail",
      "metric_value": 3,
      "threshold": 0,
      "threshold_dir": "<=",
      "details": "3 unresolved SoD violations detected",
      "checked_at": "2026-07-18T10:00:00Z"
    }
  ],
  "summary": {
    "pass": 13,
    "fail": 1,
    "warn": 1
  }
}
```

### Get Control History

```http
GET /api/v1/audit/ccm/history?control_id=MFA_ENFORCEMENT&limit=30
X-Tenant-ID: <tenant-uuid>
```

Returns historical results for trend analysis.

## Data Model

### `ccm_results` Table

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT (PK) | Unique result ID |
| `tenant_id` | UUID | Tenant scope |
| `control_id` | TEXT | Control identifier |
| `control_name` | TEXT | Human-readable name |
| `category` | TEXT | Control category |
| `status` | TEXT | `pass`, `fail`, `warn` |
| `metric_value` | FLOAT | Measured value |
| `threshold` | FLOAT | Expected threshold |
| `threshold_dir` | TEXT | `>=`, `<=`, `==` |
| `details` | JSONB | Additional context |
| `checked_at` | TIMESTAMPTZ | Evaluation timestamp |

**Indexes:**
- `idx_ccm_tenant_time` — tenant + time descending
- `idx_ccm_control` — tenant + control_id + time (history queries)

## Control Status

| Status | Meaning | Action |
|--------|---------|--------|
| `pass` | Control meets threshold | None |
| `fail` | Control violates threshold | Remediate immediately |
| `warn` | Control nears threshold | Monitor closely |

## Compliance Framework Mapping

| Control | SOC 2 | ISO 27001 | HIPAA | DORA |
|---------|-------|-----------|-------|------|
| MFA_ENFORCEMENT | CC6.1 | A.9.4.2 | 164.312(d) | Art. 9 |
| AUDIT_INTEGRITY | CC7.2 | A.12.4 | 164.312(b) | Art. 11 |
| SOD_VIOLATIONS | CC6.3 | A.6.1.2 | — | Art. 8 |
| PRIVILEGE_CREEP | CC6.3 | A.9.4.4 | — | Art. 8 |
| SESSION_TIMEOUT | CC6.1 | A.9.4.5 | 164.312(a)(2)(iii) | — |

## Best Practices

1. **Schedule daily scans** — Run full CCM evaluation at least once per day
2. **Alert on status changes** — Configure webhook for `pass → fail` transitions
3. **Export for audits** — Generate compliance reports from `ccm_results` for external auditors
4. **Track trends** — Use history endpoint to show improvement/degradation over time
5. **Correlate with privileged ops** — Cross-reference with `privileged_operations` for root-cause analysis
6. **Auto-remediation** — Wire failing controls to SOAR playbooks for automated response
