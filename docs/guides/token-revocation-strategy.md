# Token Revocation Strategy Guide

Guide for token revocation in GGID — revocation endpoints, backchannel logout, token family revocation, cascade revocation, Redis invalidation, propagation delay.

## Revocation Endpoints

### Revoke Access Token (jti)

```bash
curl -X POST https://api.ggid.example.com/oauth/revoke \
  -d "token=$ACCESS_TOKEN&client_id=$CID&client_secret=$SECRET"
```

Adds `jti` to Redis blacklist. Subsequent requests return 401.

### Revoke All User Sessions

```bash
curl -X DELETE "https://api.ggid.example.com/api/v1/sessions?user_id=$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Token Family Revocation

When refresh token rotation detects reuse (theft):

```
Token A → rotated to B → B rotated to C
  ↓
Attacker uses A (stolen) after B was issued
  ↓
GGID detects: A already used → REVOKE ENTIRE FAMILY (A, B, C)
  ↓
User must re-authenticate
```

## Cascade Revocation

| Trigger | What Gets Revoked |
|--------|------------------|
| Password change | All sessions + all tokens |
| Account lock | All sessions + all tokens |
| Account delete | All sessions + all tokens + all OAuth consents |
| MFA removed | All sessions (force re-auth with new MFA) |
| Role revoked | Token scopes re-evaluated (narrowed) |
| Agent suspended | All agent tokens (jti blacklist) |
| Trust revoked | All cross-tenant tokens from that tenant |

## Backchannel Logout (OIDC)

GGID supports OIDC Back-Channel Logout (draft):

```
User logs out at GGID → GGID sends POST to all registered clients:
  POST https://app.example.com/backchannel-logout
  Content-Type: application/x-www-form-urlencoded
  
  logout_token=<JWT containing sub and iat>
  
Client verifies JWT → destroys local session
```
### Logout Token Claims

```json
{
  "iss": "https://api.ggid.example.com",
  "sub": "user-uuid",
  "iat": 1706104200,
  "jti": "unique-id",
  "events": {
    "http://schemas.openid.net/event/backchannel-logout": {}
  }
}
```

## Redis Invalidation

```
# Revoke single jti
DEL jti:{tenant}:{jti}

# Revoke all user jtis
SMEMBERS sessions:{tenant}:{user_id} → for each session: DEL jti:{tenant}:{jti}
DEL sessions:{tenant}:{user_id}

# Revoke all tenant tokens (emergency)
SCAN jti:{tenant}:* → DEL each
```

## Propagation Delay

| Component | Delay | Reason |
|----------|-------|--------|
| Redis (same AZ) | < 1ms | Synchronous |
| Redis (cross-AZ) | 1-5ms | Network latency |
| JWKS cache | 0-15min | TTL-based cache |
| Client-side cache | 0-5min | Depends on client |
| CDN edge | 0-60s | TTL-based |

**Mitigation**: Keep access token TTL short (15 min) to limit propagation window.

## Emergency Revocation

```bash
# Revoke ALL tokens for ALL users (emergency)
curl -X POST https://api.ggid.example.com/api/v1/admin/revoke-all \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -d '{"reason":"Security incident","force_reauth":true}'
```

All users must re-authenticate. Use only for major incidents.

## See Also

- [Session Management](session-management-guide.md)
- [OAuth API](../api/oauth.md)
- [JWT Verification](jwt-verification.md)
