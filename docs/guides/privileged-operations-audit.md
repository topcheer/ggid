# Privileged Operations Audit Guide (KB-279)

## Overview

GGID's Privileged Operations Audit provides structured, tamper-evident logging of all elevated-privilege API actions. This satisfies DORA, HIPAA, and SOX compliance requirements for privileged access accountability without requiring terminal session recording.

**Design choice:** Option B (lightweight API-level recording). GGID records privileged API operations through its existing hash-chained audit system. Terminal session recording is delegated to third-party PAM products via webhooks.

## What Gets Recorded

All privileged operations are captured with full context:

| Field | Description |
|-------|-------------|
| `operator_id` | User performing the action |
| `target_id` | Affected user or resource |
| `action` | Operation type (see below) |
| `elevated_role` | Temporary role granted |
| `scopes_delta` | Permission changes (+/- scopes) |
| `before_perms` / `after_perms` | Full permission snapshot |
| `duration_seconds` | Time-limited elevation duration |
| `ip_address` | Source IP |
| `metadata` | Additional structured context |

## Tracked Actions

| Action | Trigger | Context |
|--------|---------|---------|
| `break_glass` | Emergency access activation | reason, scope, duration |
| `jit_elevate` | Just-in-time role elevation | requested role, approval status |
| `user_delete` | User account deletion | target user |
| `policy_change` | Security policy modification | policy name, diff |
| `role_assign` | Role assignment | target user, role |
| `config_change` | System configuration change | config key, old/new values |

## API Endpoint

### Query Privileged Operations

```http
GET /api/v1/identity/privileged-operations?operator_id=<uuid>&action=break_glass&limit=50
X-Tenant-ID: <tenant-uuid>
```

**Query Parameters:**
- `operator_id` — Filter by operator (optional)
- `action` — Filter by action type (optional)
- `limit` — Max results (default: 100, max: 500)

**Response:**
```json
{
  "operations": [
    {
      "id": "op_abc123",
      "operator_id": "uuid-of-admin",
      "target_id": "uuid-of-target",
      "action": "break_glass",
      "elevated_role": "break_glass",
      "scopes_delta": ["+production-db"],
      "before_perms": [],
      "after_perms": ["break_glass:production-db"],
      "duration_seconds": 1800,
      "ip_address": "10.0.1.42",
      "metadata": {
        "reason": "P0 incident - database connectivity",
        "duration_min": 30
      },
      "timestamp": "2026-07-18T10:30:00Z"
    }
  ],
  "count": 1
}
```

## Data Model

### `privileged_operations` Table

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT (PK) | Unique operation ID |
| `tenant_id` | UUID | Tenant scope |
| `operator_id` | TEXT | User performing action |
| `target_id` | TEXT | Affected entity |
| `action` | TEXT | Operation type |
| `elevated_role` | TEXT | Temporary role |
| `scopes_delta` | TEXT[] | Permission changes |
| `duration_seconds` | INT | Elevation duration |
| `ip_address` | TEXT | Source IP |
| `metadata` | JSONB | Additional context |
| `timestamp` | TIMESTAMPTZ | When operation occurred |

**Indexes:**
- `idx_privil_op_tenant` — tenant-scoped queries
- `idx_privil_op_operator` — per-operator audit trail
- `idx_privil_op_timestamp` — chronological ordering

## Break-Glass Integration

Break-glass activations emit structured privileged operation events with full permission context:

```
Break-glass activated →
  Audit event: break_glass.activate
    scopes_before: []
    scopes_after: ["break_glass:production-db"]
    scopes_delta: ["+production-db"]
    elevated_role: "break_glass"
    duration_min: 30
    reason: "P0 incident"
```

This feeds into:
1. **Audit hash-chain** — Tamper-evident record
2. **CAE engine** — Triggers session re-evaluation
3. **SOC webhook** — External SIEM notification
4. **Compliance reports** — DORA/SOX evidence

## Hash-Chain Integrity

All privileged operations are covered by GGID's HMAC-SHA256 hash chain:

1. Each audit event includes `prev_hash` linking to the previous event
2. Any tampering with privileged operation records breaks the chain
3. `VerifyHashChain` validates integrity of the complete audit trail
4. Compliance audits can cryptographically prove no records were modified

## Compliance Mapping

| Framework | Requirement | Coverage |
|-----------|-------------|----------|
| **DORA** | Privileged access logging | All privileged API calls recorded |
| **SOX** | Segregation of duties evidence | Operator + action + timestamp logged |
| **HIPAA** | Access to PHI systems | Before/after permissions captured |
| **ISO 27001** | Privileged access review | Queryable audit trail |

## Best Practices

1. **Monitor break-glass frequency** — Alert if >3 activations/month per user
2. **Review scopes_delta weekly** — Look for unexpected permission expansions
3. **Correlate with CAE** — Ensure risk scores spike on break-glass events
4. **Export to SIEM** — Forward privileged operations to external SOC
5. **Regular hash-chain verification** — Run integrity checks at least daily
6. **Retention alignment** — Keep privileged operation logs for the full compliance retention period (7 years for SOX)
