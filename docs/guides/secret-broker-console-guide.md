# Zero-Trust Secret Broker — Console Guide

> Console path: `/security/secret-broker` | Feature: F-44 | Commit: 7f4c1855

## Overview

The Secret Broker provides dynamic, short-lived credential issuance for databases, SSH hosts, cloud providers, and API keys. Every credential has a TTL, is bound to a user+role, supports JIT approval linkage, and is fully audit-trailed.

## Console Tabs (5)

### 1. Targets

Manage secret targets — the downstream resources that the broker can issue credentials for.

**Supported types:**

| Type | Icon | Connection Params |
|------|------|-------------------|
| Database (`db`) | Database | host, port, database |
| SSH Host (`ssh`) | Terminal | host, port (default 22) |
| Cloud Provider (`cloud`) | Cloud | provider (aws), region |
| API Key (`api_key`) | Link2 | endpoint URL |

**Actions:**
- **Add Target** — Opens dialog with name, type, TTL (seconds), default role, connection config, enabled toggle
- **Edit** — Click gear icon on a target card
- **Enable/Disable** — Click shield icon to toggle without deleting
- **Delete** — Click trash icon (permanent)

**TTL:** Each target has a default TTL (in seconds). Credentials issued for that target expire automatically after the TTL elapses.

### 2. Issue Credential

Dynamically request a short-lived credential for a specific target+user combination.

**Required fields:**
- **Target** — Select from enabled targets (dropdown)
- **User ID** — The identity receiving the credential (e.g., `user:alice`)

**Optional fields:**
- **Role** — Override the target's default role
- **JIT Request ID** — Link to a pre-approved Just-In-Time access request for audited elevation

**Credential display:**
- Issued credential is shown **once** with a "One-Time View" warning
- Use the eye toggle to reveal/hide the credential value
- Use the copy button to copy to clipboard
- Grant ID, role, and expiry time are displayed for audit

### 3. Active Grants

Real-time view of all active credential grants.

**Features:**
- Auto-refreshes every 15 seconds for live TTL countdown
- Shows user, target name, role, status (active/expired/revoked), TTL remaining
- **Revoke** button for active grants (immediate credential invalidation)
- Expired grants remain visible with "expired" badge until cleanup

### 4. Audit Timeline

Chronological view of credential lifecycle events.

**Event types:**
- **Issued** (green) — Credential was issued to a user
- **Revoked** (red) — Credential was manually revoked
- **Expired** (gray) — TTL elapsed and credential auto-expired

Each event shows the target name, user, role, timestamp, and truncated grant ID.

### 5. Connection Tester

Validate connectivity parameters before creating a production target.

**Usage:**
1. Select target type (DB/SSH/Cloud/API Key)
2. Enter host/endpoint and port (if applicable)
3. Click **Test Connectivity**
4. Results show success/failure with latency in milliseconds

## API Endpoints

All endpoints require `Authorization: Bearer <token>` and `X-Tenant-ID` headers.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/identity/secret-broker/targets` | List all targets |
| `POST` | `/api/v1/identity/secret-broker/targets` | Create a target |
| `PUT` | `/api/v1/identity/secret-broker/targets/:id` | Update a target |
| `DELETE` | `/api/v1/identity/secret-broker/targets/:id` | Delete a target |
| `POST` | `/api/v1/identity/secret-broker/broker` | Issue a credential |
| `GET` | `/api/v1/identity/secret-broker/active` | List active grants |
| `POST` | `/api/v1/identity/secret-broker/revoke` | Revoke a grant |

## curl Examples

### Create a database target

```bash
curl -X POST https://ggid.example.com/api/v1/identity/secret-broker/targets \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prod-postgres",
    "type": "db",
    "connection_config": {"host": "10.0.0.5", "port": "5432", "database": "appdb"},
    "ttl_seconds": 3600,
    "default_role": "readonly",
    "enabled": true
  }'
```

### Issue a credential

```bash
curl -X POST https://ggid.example.com/api/v1/identity/secret-broker/broker \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "target_id": "target-uuid-here",
    "user_id": "user:alice",
    "role": "readonly",
    "jit_request_id": "jit-approval-uuid"
  }'
```

**Response:**
```json
{
  "grant_id": "grant-uuid",
  "target_id": "target-uuid",
  "user_id": "user:alice",
  "role": "readonly",
  "credential": "dynamic-credential-value",
  "expires_at": "2026-07-18T02:00:00Z"
}
```

### List active grants

```bash
curl https://ggid.example.com/api/v1/identity/secret-broker/active \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Revoke a grant

```bash
curl -X POST https://ggid.example.com/api/v1/identity/secret-broker/revoke \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"grant_id": "grant-uuid-here"}'
```

## Security Best Practices

1. **Short TTLs** — Use the minimum TTL that satisfies the use case (15min for ad-hoc, 1h for services)
2. **JIT Approval** — For sensitive targets, require a pre-approved JIT request ID
3. **Revoke promptly** — Revoke credentials when work is complete, don't wait for TTL
4. **Audit trail** — All broker events are logged for compliance (SOC2/GDPR)
5. **One-time display** — Credentials are shown once; if lost, revoke and re-issue
6. **Disable unused targets** — Prevents accidental credential issuance

## Architecture

```
Console UI (Next.js)
    │ REST API
    ▼
Identity Service
    │ pgxpool
    ▼
PostgreSQL
  ├─ secret_broker_targets (target definitions)
  └─ secret_broker_grants  (issued credentials + lifecycle)
```

The Secret Broker is fully DB-backed (no in-memory storage). All targets and grants are persisted in PostgreSQL with automatic cleanup of expired grants via `CleanupExpired()`.

## Related Documentation

- [Secret Management](secret-management.md) — Vault patterns, rotation automation
- [Access Broker Design](../architecture/access-broker-design.md) — ZTNA architecture
- [Secrets Rotation Automation](secrets-rotation-automation.md) — Automated rotation policies
