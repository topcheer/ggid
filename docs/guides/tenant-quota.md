# Tenant Quota Management — Technical Guide

> Feature: Tenant Quota Engine
> Location: `services/identity/internal/server/quota_handler.go`
> Console: `/settings/tenant-quota`

## What It Does

The Tenant Quota Engine tracks per-tenant resource consumption against plan-based limits. It prevents individual tenants from exceeding their allocated resources (users, API keys, sessions, storage, API calls) in multi-tenant deployments.

## Plan Tiers

| Resource | Free | Pro | Enterprise |
|----------|------|-----|------------|
| Max Users | 100 | 1,000 | 50,000 |
| Max API Keys | 5 | 50 | 500 |
| Max Sessions | 50 | 500 | 5,000 |
| Max Storage | 1 GB | 10 GB | 100 GB |
| Max API Calls/Day | 10,000 | 100,000 | 1,000,000 |

## Data Model

```go
type TenantQuota struct {
    TenantID     string `json:"tenant_id"`
    Plan         string `json:"plan"`              // free, pro, enterprise
    MaxUsers     int    `json:"max_users"`
    MaxAPIKeys   int    `json:"max_api_keys"`
    MaxSessions  int    `json:"max_sessions"`
    MaxStorageMB int    `json:"max_storage_mb"`
    MaxAPICallsDay int  `json:"max_api_calls_per_day"`
}

type TenantUsage struct {
    UserCount     int `json:"user_count"`
    APIKeyCount   int `json:"api_key_count"`
    SessionCount  int `json:"session_count"`
    StorageMB     int `json:"storage_mb"`
    APICallsToday int `json:"api_calls_today"`
}
```

## Enforcement

Quota checks happen before resource creation:

1. **User creation**: Check `user_count < max_users` before allowing register.
2. **API key creation**: Check `api_key_count < max_api_keys`.
3. **Session creation**: Check `session_count < max_sessions` (evict oldest if exceeded).
4. **API calls**: Gateway middleware checks daily counter before processing.
5. **Storage**: Checked during file/upload operations.

When exceeded, the system returns HTTP 429 with a descriptive error.

## Database Schema

```sql
CREATE TABLE tenant_quotas (
    tenant_id TEXT UNIQUE,
    plan TEXT DEFAULT 'free',
    max_users INT DEFAULT 100,
    max_api_keys INT DEFAULT 5,
    max_sessions INT DEFAULT 50,
    max_storage_mb INT DEFAULT 1024,
    max_api_calls_per_day INT DEFAULT 10000
);

CREATE TABLE tenant_usage (
    tenant_id TEXT,
    metric TEXT,
    value INT DEFAULT 0,
    updated_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, metric)
);
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/tenants/:id/quota` | GET | Get quota + current usage |
| `/api/v1/tenants/:id/quota` | PUT | Update plan/limits |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Get current quota and usage
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/tenants/$TENANT/quota" \
  -H "Authorization: Bearer $TOKEN"

# Upgrade to enterprise plan
curl -k -H 'Accept-Encoding: identity' \
  -X PUT "https://ggid.iot2.win/api/v1/tenants/$TENANT/quota" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plan":"enterprise","max_users":50000,"max_api_keys":500,"max_sessions":5000,"max_storage_mb":102400,"max_api_calls_per_day":1000000}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| HTTP 429 quota_exceeded | Tenant hit resource limit | Upgrade plan or clean up unused resources |
| Usage not updating | Usage tracking lag | Usage updates are async; allow up to 60s |
| Default limits too low | New tenant on free plan | Upgrade to pro/enterprise |

## Best Practices

- **Monitor usage trends**: Track usage growth to forecast plan upgrades.
- **Clean up inactive users**: Reduce user count by running dormant detection.
- **Rotate unused API keys**: Remove expired/revoked keys to free quota.
- **Set alerts**: Configure notifications at 80% quota usage.
