# Unified Policy Decision Point (PDP) — Technical Guide

> Feature: Unified Authorization Engine (RBAC + ABAC + ReBAC + Risk)
> Location: `services/policy/internal/server/unified_pdp.go`
> Endpoint: `POST /api/v1/policy/authorize`

## What It Does

The Unified PDP consolidates all authorization decision-making into a single endpoint. Instead of separately querying RBAC, ABAC, ReBAC, and risk scoring, the PDP evaluates all layers in sequence and returns a single allow/deny decision with a full audit trail.

## Decision Flow

```
Authorization Request
(subject, resource, action, context)
         ↓
   ┌─────────────────┐
   │  1. RBAC Check   │ ← roles + permissions
   └────────┬────────┘
            ↓
   ┌─────────────────┐
   │  2. ABAC Check   │ ← attribute policies
   └────────┬────────┘
            ↓
   ┌─────────────────┐
   │  3. ReBAC Check  │ ← relationship tuples
   └────────┬────────┘
            ↓
   ┌─────────────────┐
   │  4. Risk Overlay │ ← real-time risk score
   └────────┬────────┘
            ↓
   Final Decision
   (allow / deny / step_up)
```

## Authorization Layers

### Layer 1: RBAC (Role-Based)
- Checks if the subject's role includes the required permission.
- Example: `admin` role has `document:delete` permission.

### Layer 2: ABAC (Attribute-Based)
- Evaluates attribute-based policies against request context.
- Example: Only allow access if `request.time` is within business hours AND `resource.department` matches `subject.department`.

### Layer 3: ReBAC (Relationship-Based)
- Checks Zanzibar-style relationship tuples.
- Example: `document:report-q4 can_view user:alice` — Alice has a direct relationship.

### Layer 4: Risk Overlay
- Evaluates real-time risk score for the subject.
- Three outcomes:
  - `none` — normal risk, proceed with RBAC/ABAC/ReBAC decision.
  - `step_up` — require additional authentication (MFA challenge).
  - `block` — deny regardless of other layers (critical risk user).

## Decision Caching

The PDP caches recent decisions in-memory to reduce latency:

- **Cache key**: SHA-256 hash of `(subject, resource, action)`.
- **TTL**: 60 seconds (configurable).
- **Invalidation**: Cache is cleared when policies, roles, or relationship tuples change.
- **Cache hit response**: Includes `"cache_hit": true` in the response.

## Audit Trail

Every authorization decision is logged to the `policy_decisions` PostgreSQL table:

```sql
CREATE TABLE policy_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    subject TEXT, resource TEXT, action TEXT,
    decision TEXT,           -- allow, deny
    deny_reason TEXT,
    risk_score INT,
    risk_overlay TEXT,       -- none, step_up, block
    context JSONB,
    evaluated_by TEXT[],     -- [rbac, abac, rebac, risk]
    cache_hit BOOLEAN,
    latency_ms INT,
    created_at TIMESTAMPTZ
);
```

Each decision includes a `decision_id` (UUID) for traceability.

## API Endpoint

### POST `/api/v1/policy/authorize`

**Request:**
```json
{
  "subject": "user:alice",
  "resource": "document:report-q4",
  "action": "read",
  "context": {
    "ip": "192.168.1.50",
    "device": "laptop",
    "time": "2026-07-18T10:00:00Z"
  }
}
```

**Response:**
```json
{
  "allowed": true,
  "deny_reason": "",
  "risk_overlay": "none",
  "risk_score": 15,
  "evaluated_by": ["rbac", "abac", "rebac", "risk"],
  "cache_hit": false,
  "latency_ms": 3,
  "decision_id": "a1b2c3d4-..."
}
```

### curl Example

```bash
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/policy/authorize" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"subject":"user:alice","resource":"document:report-q4","action":"read","context":{"ip":"192.168.1.50"}}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Expected allow but denied | Missing RBAC permission or ABAC policy blocks | Check evaluated_by to see which layer denied |
| High latency | Cache miss + complex ReBAC expansion | Check latency_ms; warm cache with repeated calls |
| risk_overlay=step_up always | User risk score consistently high | Review risk factors on the Risk Score dashboard |
| Decision not audited | pdpRepo nil or DB unreachable | Check policy_decisions table exists; verify DB connection |

## Best Practices

- **Always pass context**: Include IP, device, time for accurate ABAC and risk evaluation.
- **Monitor latency**: PDP latency should be <10ms (cache hit) or <50ms (miss).
- **Handle step_up gracefully**: When risk_overlay=step_up, trigger MFA challenge before proceeding.
- **Audit regularly**: Query policy_decisions table to review authorization patterns.
- **Invalidate cache after policy changes**: Restart policy pod after major policy updates to clear stale decisions.
