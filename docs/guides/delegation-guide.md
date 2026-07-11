# Delegation & Token Exchange Guide

This guide covers role delegation, AI agent token exchange, expiry, revocation, and audit trails in GGID.

## Overview

GGID implements two types of delegation:

1. **Role delegation** — Admin grants a role to another user (manual)
2. **Token delegation** — AI agent receives delegated access token (automated, RFC 8693 pattern)

## Role Delegation

### Assign Role

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/$USER_ID/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"role_id": "role-uuid", "expires_at": "2025-12-31T23:59:59Z"}'
```

### Time-Limited Roles

Roles can be assigned with an expiry:

```json
{
  "role_id": "admin-role-uuid",
  "expires_at": "2025-06-30T23:59:59Z"
}
```

After expiry, the role is automatically revoked and an audit event is logged.

### Delegation Chain

```
Super Admin
  └─ assigns 'admin' role to Manager A
       └─ Manager A assigns 'developer' to User B
            └─ Who assigned this? → Delegation audit trail
```

### View Delegation History

```bash
curl https://api.ggid.example.com/api/v1/audit/events?event_type=role.assigned \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Revoke Role

```bash
curl -X DELETE https://api.ggid.example.com/api/v1/users/$USER_ID/roles/$ROLE_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

Revocation triggers:
1. Role removed from user
2. Audit event: `role.revoked`
3. Active sessions for user re-evaluated (scope narrowed on next request)

## AI Agent Token Delegation

### Register Agent

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "deployment-agent",
    "type": "service",
    "scopes": ["users:read", "policy:check"],
    "max_delegation_depth": 3
  }'
```

### Exchange Token (Delegation)

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/$AGENT_ID/token \
  -H "Authorization: Bearer $SUBJECT_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"scopes": ["users:read"]}'
```

**Response**:
```json
{
  "access_token": "eyJhbG...",
  "agent_id": "agent-uuid",
  "delegation_chain": [
    { "agent_id": "agent-uuid", "scopes": ["users:read"], "depth": 1 }
  ],
  "expires_in": 900
}
```

### Delegation Chain Rules

| Rule | Description |
|------|-------------|
| Scope narrowing | Delegated token can only have a subset of the agent's registered scopes |
| Max depth | Each delegation increments depth; rejected if exceeds `max_delegation_depth` |
| No scope expansion | Cannot delegate a scope the parent doesn't have |
| Token expiry | Delegated tokens expire independently |

### Verify Agent Token

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/verify \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"token": "agent-jwt-token"}'
```

**Response**:
```json
{
  "valid": true,
  "agent_id": "agent-uuid",
  "delegation_chain": [...],
  "scopes": ["users:read"],
  "depth": 2
}
```

### Suspend Agent

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/$AGENT_ID/suspend \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"reason": "Security investigation"}'
```

Suspension triggers:
1. Agent marked suspended
2. All active agent tokens invalidated (jti blacklist)
3. New token exchanges rejected
4. Audit event: `agent.suspended`

## Audit Trail

Every delegation action is logged:

| Event | Trigger | Key Fields |
|-------|---------|------------|
| `role.assigned` | Role granted | user_id, role_id, assigned_by, expires_at |
| `role.revoked` | Role removed | user_id, role_id, revoked_by |
| `agent.registered` | Agent created | agent_id, name, scopes |
| `agent.token_exchanged` | Token delegated | agent_id, scopes, depth, subject |
| `agent.suspended` | Agent disabled | agent_id, reason |

## Delegation vs Impersonation

| Aspect | Delegation | Impersonation |
|--------|-----------|---------------|
| Identity | Agent acts as itself | Agent acts AS the user |
| Audit | Shows agent_id + delegation chain | Shows impersonated user |
| Scopes | Subset of agent scopes | Full user scopes |
| Token claim | `agent_id` in JWT | `act` (actor) claim in JWT |
| Use case | Service-to-service | Admin troubleshooting |

## See Also

- [AI Agent Identity](ai-agent-identity.md)
- [Token Exchange RFC 8693](../research/token-exchange-rfc8693.md)
- [Access Reviews](access-reviews.md)
- [REST API Reference](../api/rest-api.md)
