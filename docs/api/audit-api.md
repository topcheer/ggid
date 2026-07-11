# Audit API Reference

Complete reference for GGID's audit service REST API — event querying, SSE streaming, export, integrity verification, compliance reports, access reviews, alerting, and retention.

**Base URL**: `https://api.ggid.example.com/api/v1/audit`

## Authentication

All endpoints require a valid JWT with `audit:read` scope (admin endpoints require `audit:write`).

```
Authorization: Bearer <access_token>
X-Tenant-ID: <tenant-uuid>
```

## Query Events

### `GET /events`

Query audit events with filters.

**Query Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `event_type` | string | Filter by type (e.g., `user.login`) |
| `actor_id` | UUID | Filter by actor (who performed the action) |
| `resource_id` | UUID | Filter by target resource |
| `start_date` | ISO 8601 | Events after this datetime |
| `end_date` | ISO 8601 | Events before this datetime |
| `result` | string | Filter by result (`success` / `failure`) |
| `page` | int | Page number (1-based, default 1) |
| `page_size` | int | Results per page (max 100, default 50) |

**Example**:
```bash
curl -X GET "https://api.ggid.example.com/api/v1/audit/events?event_type=user.login&start_date=2025-01-01T00:00:00Z&page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response**:
```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "event_type": "user.login",
      "actor_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "tenant_id": "00000000-0000-0000-0000-000000000001",
      "resource_type": "user",
      "resource_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "action": "login",
      "result": "success",
      "ip_address": "192.168.1.50",
      "user_agent": "Mozilla/5.0...",
      "timestamp": "2025-01-24T14:30:00.123Z",
      "metadata": {
        "method": "password",
        "mfa_used": true
      }
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 154823,
  "total_pages": 7742
}
```

## SSE Event Stream

### `GET /events/stream`

Real-time event stream via Server-Sent Events.

```bash
curl -N "https://api.ggid.example.com/api/v1/audit/events/stream" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response** (SSE format):
```
data: {"event_type":"user.login","actor_id":"...","timestamp":"2025-01-24T14:30:00Z"}

data: {"event_type":"role.assigned","actor_id":"...","timestamp":"2025-01-24T14:31:00Z"}

```

**JavaScript client**:
```javascript
const stream = new EventSource('https://api.ggid.example.com/api/v1/audit/events/stream', {
  withCredentials: true,
});
stream.onmessage = (e) => {
  const event = JSON.parse(e.data);
  console.log(event.event_type, event.timestamp);
};
```

## Export

### `GET /export`

Export audit events to CSV or JSON.

**Query Parameters**: Same as `/events` plus:

| Parameter | Type | Description |
|-----------|------|-------------|
| `format` | string | `csv` or `json` (default: `json`) |

**Example**:
```bash
# CSV export
curl -X GET "https://api.ggid.example.com/api/v1/audit/export?format=csv&start_date=2025-01-01" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit_january.csv

# JSON export
curl -X GET "https://api.ggid.example.com/api/v1/audit/export?format=json&event_type=user.login" \
  -H "Authorization: Bearer $TOKEN" \
  -o logins.json
```

**CSV format**:
```csv
id,event_type,actor_id,action,result,ip_address,timestamp
550e8400...,user.login,a1b2c3d4...,login,success,192.168.1.50,2025-01-24T14:30:00Z
```

## Integrity Verification

### `GET /integrity/verify`

Verify the audit hash chain for tamper detection.

```bash
curl -X GET "https://api.ggid.example.com/api/v1/audit/integrity/verify" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response**:
```json
{
  "valid": true,
  "events_verified": 154823,
  "chain_head_hash": "sha256:a1b2c3...",
  "first_event": "2024-01-01T00:00:00Z",
  "last_event": "2025-01-24T14:30:00Z",
  "broken_at": null
}
```

If tampered:
```json
{
  "valid": false,
  "events_verified": 50000,
  "broken_at": {
    "event_id": "abc123...",
    "expected_hash": "sha256:def456...",
    "actual_hash": "sha256:ghi789..."
  }
}
```

## Compliance Reports

### `GET /compliance/report`

Generate a compliance report for a specific framework.

**Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `type` | string | `soc2`, `hipaa`, `gdpr` |
| `start_date` | ISO 8601 | Report period start |
| `end_date` | ISO 8601 | Report period end |

```bash
curl -X GET "https://api.ggid.example.com/api/v1/audit/compliance/report?type=soc2&start_date=2025-01-01&end_date=2025-03-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Response**:
```json
{
  "type": "soc2",
  "period": { "start": "2025-01-01", "end": "2025-03-31" },
  "summary": {
    "total_events": 154823,
    "unique_users": 342,
    "failed_logins": 1287,
    "privilege_changes": 45,
    "admin_actions": 312
  },
  "sections": [
    {
      "name": "Access Control (CC6)",
      "status": "pass",
      "controls": ["CC6.1", "CC6.2", "CC6.3"],
      "findings": []
    },
    {
      "name": "Change Management (CC8)",
      "status": "pass",
      "controls": ["CC8.1"],
      "findings": []
    }
  ]
}
```

## Access Reviews

### `GET /access-reviews`

List access review certification campaigns.

### `POST /access-reviews`

Create a new certification campaign.

```bash
curl -X POST "https://api.ggid.example.com/api/v1/audit/access-reviews" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Q1 2025 Manager Review",
    "type": "manager_review",
    "scope": { "roles": ["admin", "developer"] },
    "deadline": "2025-03-31T23:59:59Z"
  }'
```

### `POST /access-reviews/{id}/decision`

Submit a review decision (approve/revoke).

```bash
curl -X POST "https://api.ggid.example.com/api/v1/audit/access-reviews/$REVIEW_ID/decision" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"decision": "revoke", "reason": "No longer needed"}'
```

## Alert Rules

### `GET /alerts/rules`

List configured alert rules.

### `POST /alerts/rules`

Create an alert rule.

```bash
curl -X POST "https://api.ggid.example.com/api/v1/audit/alerts/rules" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "Failed login burst",
    "condition": "event_type=user.login AND result=failure COUNT > 10 IN 5m",
    "action": "email",
    "recipients": ["security@company.com"],
    "enabled": true
  }'
```

### `POST /alerts/test`

Send a test alert to verify configuration.

## Retention

### `GET /retention`

Get current retention policy.

```bash
curl -X GET "https://api.ggid.example.com/api/v1/audit/retention" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Response**:
```json
{
  "max_age_days": 365,
  "max_count": 1000000,
  "enabled": true
}
```

### `PUT /retention`

Update retention policy.

```bash
curl -X PUT "https://api.ggid.example.com/api/v1/audit/retention" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"max_age_days": 730, "max_count": 5000000}'
```

### `POST /retention/apply`

Manually trigger retention cleanup.

```bash
curl -X POST "https://api.ggid.example.com/api/v1/audit/retention/apply" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Response**:
```json
{
  "deleted_by_age": 15423,
  "deleted_by_count": 0,
  "total_deleted": 15423
}
```

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_request` | Missing or invalid parameters |
| 401 | `unauthorized` | Missing or invalid token |
| 403 | `forbidden` | Insufficient scope (need `audit:read`) |
| 404 | `not_found` | Event or resource not found |
| 429 | `rate_limit_exceeded` | Too many requests |
| 500 | `internal_error` | Server error |

```json
{
  "error": {
    "code": "forbidden",
    "message": "Insufficient scope: audit:read required"
  }
}
```

## See Also

- [REST API Reference](rest-api.md)
- [Audit & SIEM Guide](../guides/audit-siem-guide.md)
- [Webhook Events Guide](../guides/webhook-events-guide.md)
- [Data Retention Policy](../guides/data-retention-policy.md)
