# Temporary Access Pass (TAP) Guide

## Overview

Temporary Access Pass (TAP) is a time-limited, single-use credential that allows a user to authenticate without their primary password. TAP is commonly used for account recovery, onboarding, and break-glass scenarios where MFA enrollment is needed but the user cannot complete primary authentication.

## How TAP Works

```
Admin issues TAP → User receives passcode → User authenticates with TAP →
JWT issued → TAP consumed (single-use) → Session established
```

## TAP Lifecycle

| Phase | Description |
|-------|-------------|
| **Issued** | Admin or automated workflow creates TAP |
| **Active** | TAP is valid within time window |
| **Consumed** | User authenticates — TAP marked used |
| **Expired** | Time window passed without use |
| **Revoked** | Admin cancels before use |

## API Endpoints

### Issue a TAP

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"Admin@123456"}' | jq -r '.access_token')

curl -X POST http://localhost:8080/api/v1/auth/tap \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "lifetime_minutes": 60,
    "is_usable_once": true
  }' | jq .
```

**Response:**
```json
{
  "tap_id": "tap_abc123",
  "passcode": "12345678",
  "expires_at": "2026-07-18T13:00:00Z",
  "is_usable_once": true
}
```

### Batch Issue TAPs

```bash
curl -X POST http://localhost:8080/api/v1/auth/tap/batch \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "user_ids": ["user-1-uuid", "user-2-uuid"],
    "lifetime_minutes": 30
  }' | jq .
```

### Authenticate with TAP

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "targetuser",
    "password": "12345678",
    "auth_method": "tap"
  }' | jq .
```

### Get TAP Policy

```bash
curl -s http://localhost:8080/api/v1/auth/tap/policy \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" | jq .
```

### Update TAP Policy

```bash
curl -X PUT http://localhost:8080/api/v1/auth/tap/policy \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "default_lifetime_minutes": 60,
    "max_lifetime_minutes": 480,
    "min_lifetime_minutes": 5,
    "default_is_usable_once": true,
    "max_active_per_user": 1
  }' | jq .
```

## Policy Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `default_lifetime_minutes` | 60 | Default validity window |
| `max_lifetime_minutes` | 480 | Maximum allowed lifetime (8h) |
| `min_lifetime_minutes` | 5 | Minimum allowed lifetime |
| `default_is_usable_once` | true | Single-use by default |
| `max_active_per_user` | 1 | Max concurrent active TAPs |

## Security Considerations

- TAPs are **audit-logged** with issuer, target, and consumption event
- Single-use TAPs are consumed on first authentication attempt
- Expired and consumed TAPs cannot be reused
- TAP issuance requires admin scope
- TAP authentication triggers CAE risk evaluation
- Rate-limited to prevent brute force

## Common Use Cases

| Scenario | TAP Configuration |
|----------|-------------------|
| New employee onboarding | 60min, single-use, auto-enroll MFA after |
| Account recovery | 30min, single-use, require new password |
| Break-glass access | 15min, single-use, full audit + alert |
| Temporary contractor | 480min, multi-use, scoped role |

## Verify (curl)

```bash
# Issue and consume TAP
TAP_RESP=$(curl -s -X POST http://localhost:8080/api/v1/auth/tap \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"user_id":"USER_UUID","lifetime_minutes":30}')
PASSCODE=$(echo $TAP_RESP | jq -r '.passcode')
echo "TAP passcode: $PASSCODE"
```
