# Audit API Reference

> **Deprecated:** This document is superseded by [docs/api/audit-api.md](docs/api/audit-api.md). See the updated version for the latest information.


Endpoints, query parameters, response schema, pagination, filtering, rate limiting, export, and hash chain verification.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/audit/events` | Query events |
| GET | `/api/v1/audit/events/{id}` | Get single event |
| GET | `/api/v1/audit/events/stream` | SSE real-time stream |
| GET | `/api/v1/audit/aggregations` | Aggregated stats |
| POST | `/api/v1/audit/export` | Start async export |
| GET | `/api/v1/audit/export/{id}` | Get export status |
| GET | `/api/v1/audit/verify-chain` | Verify hash chain |

## Query Parameters (GET /events)

| Parameter | Type | Example | Description |
|-----------|------|---------|-------------|
| `tenant_id` | UUID | `?tenant_id=uuid` | Filter by tenant |
| `actor_id` | UUID | `?actor_id=uuid` | Filter by actor user |
| `resource_type` | string | `?resource_type=user` | Filter by resource type |
| `action` | string | `?action=user.login` | Exact action match |
| `action_prefix` | string | `?action_prefix=user.` | Action prefix match |
| `result` | string | `?result=denied` | Filter by result |
| `severity` | string | `?severity=critical` | Filter by severity |
| `start_time` | ISO8601 | `?start_time=2025-01-01T00:00:00Z` | Range start |
| `end_time` | ISO8601 | `?end_time=2025-01-31T23:59:59Z` | Range end |
| `ip` | string | `?ip=10.0.1.5` | Filter by actor IP |
| `q` | CEL | `?q=action=="user.delete" && risk_score>50` | Advanced query |
| `limit` | int | `?limit=100` | Max 1000, default 100 |
| `cursor` | string | `?cursor=eyJpZCI6...` | Cursor pagination |
| `sort` | string | `?sort=created_at:desc` | Sort order |

## Response Schema

```json
{
  "events": [
    {
      "event_id": "evt-uuid",
      "tenant_id": "uuid",
      "actor": {
        "user_id": "uuid",
        "ip": "10.0.1.5",
        "user_agent": "Mozilla/5.0..."
      },
      "action": "user.login",
      "resource": {
        "type": "user",
        "id": "uuid"
      },
      "result": "success",
      "severity": "info",
      "risk_score": 12,
      "details": {
        "method": "password+mfa",
        "mfa_type": "totp"
      },
      "trace_id": "0af7651916cd43dd8448eb211c80319c",
      "created_at": "2025-01-15T10:30:00.123Z",
      "chain": {
        "sequence": 1452390,
        "hash": "abc123...",
        "prev_hash": "def456..."
      }
    }
  ],
  "total": 1547,
  "limit": 100,
  "next_cursor": "eyJpZCI6ImV2dC0xNDUyMzkwIn0="
}
```

## Pagination

### Cursor Format

```
cursor = base64url({"id": "evt-1452390"})
```

Next page: `?cursor=<next_cursor>&limit=100`

### Limits

| Mode | Max | Default |
|------|-----|---------|
| Offset | 10000 | 100 |
| Cursor | Unlimited | 100 |

## Filtering Operators (CEL)

| Operator | Example |
|----------|---------|
| `==` | `result == "denied"` |
| `!=` | `action != "user.login"` |
| `in` | `action in ["user.create", "user.delete"]` |
| `>` / `<` | `risk_score > 50` |
| `&&` | `result == "denied" && severity == "critical"` |
| `\|\|` | `action == "user.create" \|\| action == "role.assign"` |
| `.contains()` | `action.contains("delete")` |
| `.startsWith()` | `actor.ip.startsWith("10.0.")` |

## Rate Limiting

| Endpoint | Limit | Burst |
|----------|-------|-------|
| GET /events | 100/min | 200 |
| GET /events/stream | 5 concurrent | — |
| GET /aggregations | 30/min | 60 |
| POST /export | 10/hour | 20 |
| GET /verify-chain | 10/hour | 20 |

Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

## Export

```bash
POST /api/v1/audit/export
{
  "query": "action in [\"user.create\", \"user.delete\"]",
  "start_time": "2025-01-01T00:00:00Z",
  "end_time": "2025-01-31T23:59:59Z",
  "format": "csv",
  "include_chain": true
}
# → 202 {"export_id": "exp-uuid", "status": "processing"}

GET /api/v1/audit/export/exp-uuid
# → {"status": "complete", "download_url": "https://...", "expires_at": "..."}
```

| Format | Max Rows |
|--------|----------|
| CSV | 1M |
| JSON | 1M |
| JSONL | 10M |
| Parquet | 100M |

## Hash Chain Verification

```bash
GET /api/v1/audit/verify-chain
# → {
#   "status": "valid",
#   "entries_verified": 1452390,
#   "broken_links": 0,
#   "verification_time_ms": 3400
# }

GET /api/v1/audit/verify-chain?start_time=2025-01-01&end_time=2025-01-31
# → Range verification
```

## See Also

- [Audit Query API](audit-query-api.md)
- [Audit Log Architecture](audit-log-architecture.md)
- [Audit Tamper Detection](audit-tamper-detection.md)
- [Audit Query Optimization](audit-query-optimization.md)