# Continuous Access Evaluation (CAE) Guide (KB-081)

## Overview

GGID's Continuous Access Evaluation (CAE) engine performs real-time policy re-evaluation on active sessions. Instead of relying solely on login-time authentication, CAE continuously assesses risk signals and access policies throughout the session lifecycle.

## How CAE Works

```
Login → Session Created → CAE Monitors → Policy Change / Risk Event → Re-evaluate → Allow / Step-up / Revoke
```

### Evaluation Triggers

| Trigger | Source | Action |
|---------|--------|--------|
| Policy update | Admin console | Re-evaluate all matching sessions |
| Risk score change | Risk engine | Step-up or revoke if threshold exceeded |
| IP address change | Session telemetry | Flag suspicious geo-velocity |
| Role revocation | HR lifecycle | Immediate access removal |
| Device posture change | MDM/EDR | Restrict if posture degrades |

## API Endpoints

### Status
```http
GET /api/v1/auth/cae/status
```
Returns summary statistics for recent CAE evaluations (last 15 minutes by default).

**Response:**
```json
{
  "status": "active",
  "evaluations_last_15m": 142,
  "by_action": {
    "allow": 120,
    "step_up": 18,
    "revoke": 4
  }
}
```

### Manual Evaluation Run
```http
POST /api/v1/auth/cae/run
Content-Type: application/json

{
  "session_id": "sess_abc123",
  "reason": "policy_change"
}
```

Triggers a manual CAE evaluation for a specific session. Returns the evaluation result including action and risk score.

### Evaluation Log
```http
GET /api/v1/auth/cae/log?limit=50
```

Retrieves recent CAE evaluation records for audit and troubleshooting.

## Data Model

### `cae_evaluations` Table

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID (PK) | Unique evaluation ID |
| `tenant_id` | UUID | Tenant scope |
| `session_id` | TEXT | Associated session |
| `user_id` | TEXT | User being evaluated |
| `action` | TEXT | `allow`, `step_up`, `revoke` |
| `policy_name` | TEXT | Policy that triggered evaluation |
| `ip_address` | TEXT | Client IP at evaluation time |
| `risk_score` | INT | Computed risk score (0-100) |
| `evaluated_at` | TIMESTAMPTZ | When evaluation occurred |

**Indexes:**
- `idx_cae_tenant_time` — tenant + time descending (dashboard queries)
- `idx_cae_session` — session lookup (per-session audit trail)

## Actions

| Action | Behavior |
|--------|----------|
| `allow` | Session continues normally |
| `step_up` | User must complete MFA before continuing |
| `revoke` | Session is immediately terminated |

## Integration with Conditional Access

CAE works in conjunction with GGID's conditional access policies:

1. **Conditional Access** defines the rules (IP ranges, device posture, risk thresholds)
2. **CAE** continuously re-evaluates those rules against active sessions
3. **Attribute Mapping** (KB-063) ensures claims are normalized before evaluation

## Session-Level Evaluation

Each CAE evaluation is tied to a specific `session_id`, enabling:

- **Per-session audit trail** — full history of evaluations for any session
- **Targeted revocation** — revoke a single compromised session without affecting others
- **Risk trend analysis** — track risk score changes over a session's lifetime

## Best Practices

- Enable CAE for all privileged accounts (administrators, service owners)
- Set risk thresholds conservatively initially (revoke > 80, step-up > 50)
- Monitor `cae_evaluations` table for anomalous patterns
- Integrate CAE revocation events with SIEM for incident response
- Test policy changes with dry-run before enabling CAE enforcement

## Monitoring

Use the CAE dashboard in the admin console to monitor:
- Evaluation volume over time
- Action distribution (allow / step-up / revoke)
- Top triggered policies
- High-risk sessions requiring investigation
