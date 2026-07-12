# Audit Query API Guide

Endpoints, query DSL, filters, aggregation, pagination, export formats, SSE streaming, and rate limits.

## Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/audit/events` | GET | Query events with filters |
| `/api/v1/audit/events/{id}` | GET | Get single event |
| `/api/v1/audit/events/stream` | GET (SSE) | Real-time event stream |
| `/api/v1/audit/export` | GET | Export events (async) |
| `/api/v1/audit/aggregations` | GET | Aggregated statistics |
| `/api/v1/audit/verify-chain` | GET | Verify hash chain |

## Basic Queries

### List Events

```bash
GET /api/v1/audit/events?limit=50&sort=created_at:desc
```

### Filter by Actor

```bash
GET /api/v1/audit/events?user_id=uuid-here
GET /api/v1/audit/events?ip=10.0.1.5
GET /api/v1/audit/events?session_id=sess-abc
```

### Filter by Action

```bash
GET /api/v1/audit/events?action=user.create
GET /api/v1/audit/events?action_prefix=user.   # All user.* actions
```

### Filter by Time Range

```bash
GET /api/v1/audit/events?from=2025-01-01T00:00:00Z&to=2025-01-31T23:59:59Z
```

### Filter by Result

```bash
GET /api/v1/audit/events?result=denied         # Access denials only
GET /api/v1/audit/events?result=failure        # Failed operations
```

## Query DSL (Advanced)

For complex queries, use the `q` parameter with CEL syntax:

```bash
# Multiple conditions
GET /api/v1/audit/events?q=action == "user.create" && result == "success"

# In list
GET /api/v1/audit/events?q=action in ["user.create", "user.delete", "role.assign"]

# Nested actor
GET /api/v1/audit/events?q=actor.ip.startsWith("10.0.") && result == "denied"

# Time comparison
GET /api/v1/audit/events?q=timestamp > "2025-01-15T10:00:00Z" && risk_score > 50

# Resource type
GET /api/v1/audit/events?q=resource.type == "user" && action.contains("delete")
```

### CEL Operators

| Operator | Example |
|----------|---------|
| `==` / `!=` | `result == "denied"` |
| `>` / `<` / `>=` / `<=` | `risk_score > 50` |
| `in` | `action in ["a", "b"]` |
| `&&` / `\|\|` | `x && y` |
| `.startsWith()` | `actor.ip.startsWith("10.")` |
| `.contains()` | `action.contains("user")` |
| `matches()` | `action.matches("user\\..*")` |

## Pagination

### Offset-based (default)

```bash
# Page 1
GET /api/v1/audit/events?limit=100&offset=0
# → {"events": [...], "total": 1547, "limit": 100, "offset": 0}

# Page 2
GET /api/v1/audit/events?limit=100&offset=100
```

### Cursor-based (large datasets)

```bash
GET /api/v1/audit/events?limit=100&cursor=eyJpZCI6ImV2dC0xMjMifQ==
# → {"events": [...], "next_cursor": "eyJpZCI6ImV2dC0yMjMifQ=="}

# Next page
GET /api/v1/audit/events?limit=100&cursor=eyJpZCI6ImV2dC0yMjMifQ==
```

Cursor-based is preferred for large result sets — no performance degradation at high offsets.

## Aggregations

### Event Count by Action

```bash
GET /api/v1/audit/aggregations?group_by=action&from=2025-01-01&to=2025-01-31
# → {
#   "buckets": [
#     {"key": "user.create", "count": 142},
#     {"key": "user.login", "count": 8453},
#     {"key": "role.assign", "count": 47}
#   ]
# }
```

### Denied Actions by User

```bash
GET /api/v1/audit/aggregations?group_by=actor.user_id&result=denied&from=2025-01-01
```

### Time Series (hourly)

```bash
GET /api/v1/audit/aggregations?group_by=hour&from=2025-01-15T00:00:00Z&to=2025-01-15T23:59:59Z
# → {"buckets": [{"hour": "2025-01-15T10:00:00Z", "count": 245}, ...]}
```

## SSE Streaming (Real-Time)

```bash
GET /api/v1/audit/events/stream
Accept: text/event-stream
Authorization: Bearer <token>

# Server sends events as they occur:
data: {"event_id":"evt-1","action":"user.login","result":"success",...}

data: {"event_id":"evt-2","action":"role.assign","result":"denied",...}
```

### Stream Filters

```bash
# Stream only denied actions
GET /api/v1/audit/events/stream?filter=result eq "denied"

# Stream only for specific tenant
GET /api/v1/audit/events/stream?tenant_id=uuid
```

Client must send heartbeat every 30s to keep connection alive.

## Export

### Async Export (Large Datasets)

```bash
# Start export
POST /api/v1/audit/export
{
  "query": "action in [\"user.create\", \"user.delete\"]",
  "from": "2025-01-01",
  "to": "2025-01-31",
  "format": "csv",       // or json, jsonl, parquet
  "include_chain": true  // Include hash chain for evidence
}
# → 202 {"export_id": "exp-abc", "status": "processing"}

# Poll for completion
GET /api/v1/audit/export/exp-abc
# → {"status": "complete", "download_url": "https://...", "expires_at": "..."}

# Download
GET <download_url>
# → Signed URL, valid 1 hour
```

### Export Formats

| Format | Use Case | Max Rows |
|--------|----------|----------|
| CSV | Spreadsheet analysis | 1M |
| JSON | Programmatic processing | 1M |
| JSONL | Streaming log ingestion | 10M |
| Parquet | Big data analytics | 100M |

## Rate Limits

| Endpoint | Limit | Burst |
|----------|-------|-------|
| `GET /events` | 100 req/min | 200 |
| `GET /events/stream` | 5 concurrent connections | — |
| `GET /aggregations` | 30 req/min | 60 |
| `POST /export` | 10 req/hour | 20 |
| `GET /export/{id}` | 60 req/min | 120 |

Rate limit headers:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1700000600
```

## See Also

- [Audit Log Architecture](audit-log-architecture.md)
- [SIEM Integration](siem-integration.md)
- [Identity Threat Detection](identity-threat-detection.md)
- [Audit API Reference](../api/audit-api.md)
