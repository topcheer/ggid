# Audit Events API Reference

Complete reference for GGID's audit event query, filter, pagination, and export endpoints.

**Base URL**: `https://api.ggid.example.com/api/v1/audit`

## Query Events

### `GET /events`

| Parameter | Type | Description |
|-----------|------|-------------|
| `event_type` | string | Filter by type (e.g., `user.login`) |
| `actor_id` | UUID | Who performed the action |
| `resource_id` | UUID | Target resource UUID |
| `resource_type` | string | `user`, `role`, `org`, `policy`, `agent` |
| `result` | string | `success` or `failure` |
| `start_date` | ISO 8601 | Events after this time |
| `end_date` | ISO 8601 | Events before this time |
| `ip_address` | string | Filter by source IP |
| `page` | int | Page number (default 1) |
| `page_size` | int | Per page (max 100, default 50) |
| `sort` | string | `timestamp` or `-timestamp` (desc) |

**Request**:
```bash
curl "https://api.ggid.example.com/api/v1/audit/events?event_type=user.login&start_date=2025-01-01T00:00:00Z&page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response** (200):
```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "event_type": "user.login",
      "actor_id": "user-uuid",
      "tenant_id": "tenant-uuid",
      "resource_type": "user",
      "resource_id": "user-uuid",
      "action": "login",
      "result": "success",
      "ip_address": "192.168.1.50",
      "user_agent": "Mozilla/5.0...",
      "timestamp": "2025-01-24T14:30:00.123Z",
      "metadata": {
        "method": "password",
        "mfa_used": true,
        "mfa_method": "totp",
        "session_id": "sess-abc123"
      }
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 154823,
  "total_pages": 7742
}
```

## Event Types Reference

### Authentication
| Type | Trigger | Metadata |
|------|---------|----------|
| `user.login` | Successful login | method, mfa_used, session_id |
| `user.login_failed` | Failed login | reason, attempt_count |
| `user.logout` | User logout | session_id |
| `user.register` | New registration | source (web/scim/api) |
| `user.locked` | Account locked | reason, attempts |
| `user.unlocked` | Admin unlock | admin_id |

### User Management
| Type | Trigger |
|------|---------|
| `user.created` | User created |
| `user.updated` | Profile updated (fields_changed[]) |
| `user.deleted` | User deleted |
| `user.suspended` | Account suspended |

### Roles & Policies
| Type | Trigger |
|------|---------|
| `role.created` | Role created |
| `role.assigned` | Role assigned to user |
| `role.revoked` | Role revoked |
| `policy.created` | Policy created |
| `policy.updated` | Policy updated |
| `policy.deleted` | Policy removed |

### OAuth & Agents
| Type | Trigger |
|------|---------|
| `oauth.consent_granted` | OAuth consent |
| `oauth.token_issued` | Token issued |
| `oauth.token_revoked` | Token revoked |
| `agent.registered` | Agent created |
| `agent.token_exchanged` | Agent token delegated |
| `agent.suspended` | Agent disabled |

### Security
| Type | Trigger |
|------|---------|
| `security.rate_limited` | Rate limit triggered |
| `security.circuit_open` | Circuit breaker opened |
| `security.suspicious_activity` | Anomaly detected |

## SSE Event Stream

### `GET /events/stream`

Real-time Server-Sent Events stream.

```bash
curl -N "https://api.ggid.example.com/api/v1/audit/events/stream" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response** (SSE):
```
data: {"event_type":"user.login","actor_id":"...","timestamp":"2025-01-24T14:30:00Z"}

data: {"event_type":"role.assigned","actor_id":"...","timestamp":"2025-01-24T14:31:00Z"}

```

**JavaScript client**:
```javascript
const stream = new EventSource('https://api.ggid.example.com/api/v1/audit/events/stream', {
  withCredentials: true,
});
stream.onmessage = (e) => console.log(JSON.parse(e.data));
```

## Export

### `GET /export`

| Parameter | Type | Description |
|-----------|------|-------------|
| `format` | string | `csv` or `json` (default: `json`) |
| `start_date` | ISO 8601 | Export from this date |
| `end_date` | ISO 8601 | Export to this date |
| `event_type` | string | Filter by event type |

```bash
# CSV export
curl "https://api.ggid.example.com/api/v1/audit/export?format=csv&start_date=2025-01-01" \
  -H "Authorization: Bearer $TOKEN" \
  -o audit_export.csv

# JSON export (filtered)
curl "https://api.ggid.example.com/api/v1/audit/export?format=json&event_type=role.assigned" \
  -H "Authorization: Bearer $TOKEN" \
  -o role_changes.json
```

**CSV format**:
```csv
id,event_type,actor_id,action,result,ip_address,timestamp
550e8400...,user.login,a1b2c3d4...,login,success,192.168.1.50,2025-01-24T14:30:00Z
```

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_request` | Invalid filter parameters |
| 401 | `unauthorized` | Missing/invalid token |
| 403 | `forbidden` | Need `audit:read` scope |
| 429 | `rate_limit_exceeded` | Too many queries |

## See Also

- [Audit API (full)](audit-api.md)
- [REST API Reference](rest-api.md)
- [Audit & SIEM Guide](../guides/audit-siem-guide.md)
