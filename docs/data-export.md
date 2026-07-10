# Data Export Guide

Export audit logs, user data, and reports from GGID in CSV and JSON formats.

---

## Audit Log Export

### CSV Export

```bash
GET /api/v1/audit/export?format=csv&start=2024-07-01&end=2024-07-31
Authorization: Bearer <token>
X-Tenant-ID: <tenant-uuid>
```

Downloads a CSV file:

```csv
timestamp,actor_id,actor_name,action,result,resource_type,resource_id,ip_address
2024-07-10T12:00:00Z,550e8400-...,admin,user.login,success,auth,,192.168.1.100
2024-07-10T12:01:00Z,550e8400-...,admin,role.create,success,role,role-uuid,192.168.1.100
```

### JSON Export

```bash
GET /api/v1/audit/export?format=json&start=2024-07-01&end=2024-07-31
```

### Filters

| Parameter | Type | Example |
|-----------|------|---------|
| `start` | ISO date | `2024-07-01` |
| `end` | ISO date | `2024-07-31` |
| `action` | string | `user.login` |
| `result` | string | `success` / `failure` |
| `actor_id` | UUID | `550e8400-...` |
| `resource_type` | string | `auth` |

### Combined Example

```bash
GET /api/v1/audit/export?format=csv&action=user.login&result=failure&start=2024-07-01&end=2024-07-10
```

Exports all failed login attempts in the first 10 days of July.

---

## Audit Statistics

```bash
GET /api/v1/audit/stats?start=2024-07-01&end=2024-07-31
```

Response:
```json
{
  "total_events": 15420,
  "by_action": {
    "user.login": 5230,
    "user.login_failed": 142,
    "role.assign": 45,
    "policy.check": 9850,
    "user.create": 12
  },
  "by_result": {
    "success": 15278,
    "failure": 142
  },
  "top_actors": [
    {"name": "admin", "count": 5230},
    {"name": "system", "count": 9850}
  ]
}
```

---

## User Data Export (GDPR)

### Export Single User

```bash
GET /api/v1/users/{user_id}/export
```

Returns all data associated with the user as a JSON bundle:

```json
{
  "user": {
    "id": "550e8400-...",
    "username": "john.doe",
    "email": "john@example.com",
    "display_name": "John Doe",
    "created_at": "2024-01-15T08:00:00Z"
  },
  "credentials": [
    {"type": "password", "created_at": "2024-01-15T08:00:00Z"},
    {"type": "webauthn", "device_name": "YubiKey"}
  ],
  "roles": [
    {"key": "editor", "name": "Content Editor", "assigned_at": "2024-02-01T10:00:00Z"}
  ],
  "organizations": [
    {"name": "Engineering", "title": "Senior Engineer"}
  ],
  "sessions": [
    {"device": "MacBook Pro", "ip": "192.168.1.100", "last_active": "2024-07-10T11:00:00Z"}
  ],
  "audit_events": [
    {"action": "user.login", "timestamp": "2024-07-10T12:00:00Z", "ip": "192.168.1.100"}
  ]
}
```

### Export Format

| Format | Parameter | Content |
|--------|-----------|---------|
| JSON | `?format=json` | Single JSON document |
| CSV | `?format=csv` | ZIP with separate CSV files per data type |

```bash
# ZIP export (multiple CSVs)
GET /api/v1/users/{user_id}/export?format=csv
# Returns: user-data.zip containing user.csv, roles.csv, sessions.csv, audit.csv
```

### Export All Users

```bash
GET /api/v1/users/export?format=csv
```

Returns a CSV of all users in the tenant (requires admin role).

---

## API Pagination

### Offset-Based (Default)

```bash
GET /api/v1/users?page=1&page_size=50
```

Response:
```json
{
  "users": [...],
  "total": 234,
  "page": 1,
  "page_size": 50,
  "total_pages": 5
}
```

### Cursor-Based (Large Datasets)

```bash
GET /api/v1/audit/events?cursor=eyJpZCI6IjU1MGU4NDAwIn0&limit=100
```

Response:
```json
{
  "events": [...],
  "next_cursor": "eyJpZCI6IjU1MGU4NDAxIn0",
  "has_more": true
}
```

### When to Use Each

| Method | Best For | Limitation |
|--------|----------|------------|
| Offset | Small datasets (< 10K rows) | Slow for deep pages (OFFSET 10000) |
| Cursor | Large datasets (> 10K rows) | No random access (can't jump to page 50) |

---

## Batch Download

### Audit Export (Large Date Range)

For exports spanning months, use async export:

```bash
# Step 1: Request export
POST /api/v1/audit/export/async
{
  "format": "csv",
  "start": "2024-01-01",
  "end": "2024-06-30",
  "notify_url": "https://yourapp.com/export-done"
}

# Response
{
  "export_id": "exp-abc123",
  "status": "processing"
}

# Step 2: Poll status
GET /api/v1/audit/export/status/exp-abc123
# {"status": "completed", "download_url": "https://..."}
```

---

## SCIM Bulk Export

```bash
GET /scim/v2/Users?startIndex=1&count=100
```

Returns SCIM-formatted user list with pagination:

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
  "totalResults": 234,
  "startIndex": 1,
  "itemsPerPage": 100,
  "Resources": [...]
}
```

---

## Best Practices

1. **Use cursor pagination for audit logs** — Audit tables grow large quickly
2. **Set date ranges** — Never export "all time" without limits
3. **Use async export for large datasets** — Avoid HTTP timeouts
4. **Encrypt exports** — User data exports contain PII
5. **Delete export files after download** — Don't leave PII on disk
6. **Log exports** — Audit who exported what data
7. **Rate limit exports** — Prevent abuse (max 1 export per minute)
