# API Key Management Guide

This guide covers API key generation, scoping, rotation, storage, revocation, monitoring, and security best practices.

## Overview

API keys provide service-to-service authentication without user interaction. They are used for backend integrations, CI/CD, and automation.

## Key Generation

### Create API Key

```bash
curl -X POST https://api.ggid.example.com/api/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "CI/CD Pipeline",
    "scopes": ["users:read", "roles:read"],
    "expires_at": "2026-01-24T00:00:00Z"
  }'
```

**Response** (201):
```json
{
  "id": "key-uuid",
  "key": "ggid_live_a1b2c3d4e5f6...",
  "name": "CI/CD Pipeline",
  "scopes": ["users:read", "roles:read"],
  "created_at": "2025-01-24T00:00:00Z",
  "expires_at": "2026-01-24T00:00:00Z"
}
```

> **The key value is shown only once.** Store it immediately.

## Key Format

```
ggid_<environment>_<32-char-random-token>

ggid_live_a1b2c3d4e5f6g7h8i9j0...
ggid_test_x1y2z3w4v5u6t7s8r9q0...
```

Environment prefix enables easy identification and prevents test keys in production.

## Scoping

### Principle of Least Privilege

| Use Case | Required Scopes |
|----------|----------------|
| CI/CD (read-only) | `users:read`, `roles:read` |
| User sync | `users:read`, `users:write` |
| Audit export | `audit:read` |
| Policy check | `policy:check` |
| Full admin | `*` (avoid!) |

### Scope Enforcement

Every API key is validated on each request:

```
Request: Authorization: Bearer ggid_live_a1b2...
  ↓
Gateway: Look up key → Get scopes → Check if scope includes required permission
  ↓
  Has scope → Allow
  Missing scope → 403 Forbidden
```

## Rotation Policy

### Recommended Schedule

| Key Type | Rotation Frequency |
|----------|-------------------|
| Production | 90 days |
| CI/CD | 90 days |
| Test | 180 days |
| Break-glass | After each use |

### Rotation Procedure (Zero-Downtime)

1. **Create new key** alongside old key
2. **Deploy** new key to your service
3. **Verify** new key works
4. **Revoke** old key
5. **Monitor** for errors (old key may be cached)

```bash
# Create new key
curl -X POST https://api.ggid.example.com/api/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"name":"CI/CD Pipeline (v2)","scopes":["users:read","roles:read"]}'

# After deploying new key, revoke old
curl -X DELETE https://api.ggid.example.com/api/v1/api-keys/$OLD_KEY_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Storage Best Practices

| Storage Method | Rating | Notes |
|---------------|--------|-------|
| Secrets manager (Vault, AWS SM) | Best | Auto-rotation, audit trail |
| Kubernetes Secret | Good | Encrypted at rest (etcd encryption) |
| CI/CD secret variable | Good | Masked in logs |
| Environment variable | OK | Use .env files, not in source |
| Source code | NEVER | Will be leaked |
| Chat/Slack/email | NEVER | No access control |

## Revocation

### Immediate Revocation

```bash
curl -X DELETE https://api.ggid.example.com/api/v1/api-keys/$KEY_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

Revocation is **immediate** — subsequent requests with the key return 401.

### Auto-Revocation Triggers

- Key expiry date reached
- Key usage anomaly detected (unusual IP, volume spike)
- Security incident
- User/tenant deletion

## Usage Monitoring

### List Keys

```bash
curl https://api.ggid.example.com/api/v1/api-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Usage Statistics

```bash
curl https://api.ggid.example.com/api/v1/api-keys/$KEY_ID/usage \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

**Response**:
```json
{
  "key_id": "uuid",
  "name": "CI/CD Pipeline",
  "total_requests": 154823,
  "requests_30d": 5021,
  "last_used": "2025-01-24T14:30:00Z",
  "top_endpoints": [
    {"path": "/api/v1/users", "count": 3000},
    {"path": "/api/v1/roles", "count": 2021}
  ],
  "unique_ips": ["10.0.1.5"],
  "scopes_used": ["users:read", "roles:read"]
}
```

### Alerts

| Alert | Condition |
|-------|-----------|
| Key about to expire | < 7 days to expiry |
| Unused key | No usage in 30 days |
| Anomalous usage | 10x normal request volume |
| New IP usage | Key used from new IP |
| Off-hours usage | Key used 22:00-06:00 |

## Security Checklist

- [ ] Keys generated with minimal scopes
- [ ] Keys stored in secrets manager
- [ ] Rotation policy enforced (90 days)
- [ ] Unused keys revoked
- [ ] Production keys never in test/dev
- [ ] Key usage monitored
- [ ] Expiry alerts configured
- [ ] Keys never committed to source control
- [ ] `.gitignore` includes `.env*`
- [ ] Breach response: revoke all keys on compromise

## See Also

- [Onboarding Guide](onboarding-guide.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [REST API Reference](../api/rest-api.md)
- [Key Management Lifecycle](../research/key-management-lifecycle.md)
