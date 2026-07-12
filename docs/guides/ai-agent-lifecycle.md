# AI Agent Identity Lifecycle Management

This guide covers the complete lifecycle of AI agent identities in GGID — onboarding, provisioning, monitoring, revocation, and audit trail.

## Overview

GGID provides native AI agent identity management for MCP (Model Context Protocol) servers, automated workflows, and service-to-service delegation.

## Lifecycle Stages

```
Register → Provision Scopes → Deploy → Monitor → Rotate → Suspend/Revoke
```

## 1. Registration (Onboarding)

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "deployment-agent",
    "type": "service",
    "description": "Handles automated deployments",
    "scopes": ["users:read", "policy:check"],
    "max_delegation_depth": 3,
    "mcp_servers": ["https://mcp.internal.com"]
  }'
```

**Agent Types**:
| Type | Use Case | Delegation |
|------|----------|------------|
| `service` | Service-to-service API access | Yes (depth-limited) |
| `mcp` | MCP server authentication | Yes |
| `workflow` | Automated workflow identity | Yes |

**Response**:
```json
{
  "id": "agent-uuid",
  "name": "deployment-agent",
  "type": "service",
  "status": "active",
  "client_id": "generated-client-id",
  "client_secret": "generated-secret"
}
```

## 2. Provisioning (Scope Assignment)

### Initial Scopes

Scopes are assigned at registration. Admin can update later:

```bash
curl -X PUT https://api.ggid.example.com/api/v1/agents/$AGENT_ID/scopes \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"scopes": ["users:read", "users:write", "audit:read"]}'
```

### Scope Narrowing Rules

When an agent exchanges a token, it can only request a **subset** of its registered scopes. It cannot expand.

## 3. Token Exchange (Authentication)

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/$AGENT_ID/token \
  -H "Authorization: Bearer $SUBJECT_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"scopes": ["users:read"]}'
```

**Token Claims**:
```json
{
  "iss": "https://api.ggid.example.com",
  "sub": "agent-uuid",
  "agent_id": "agent-uuid",
  "agent_type": "service",
  "delegation_chain": [
    { "agent_id": "agent-uuid", "scopes": ["users:read"], "depth": 1 }
  ],
  "mcp_servers": ["https://mcp.internal.com"],
  "max_delegation_depth": 3,
  "scope": "users:read",
  "exp": 1706105100
}
```

## 4. Monitoring

### Agent Activity Metrics

```bash
curl https://api.ggid.example.com/api/v1/agents/$AGENT_ID/metrics \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

| Metric | Description |
|--------|-------------|
| `tokens_exchanged` | Total token exchanges |
| `active_tokens` | Currently valid tokens |
| `delegation_depth_avg` | Average delegation depth |
| `scope_usage` | Most-used scopes |
| `last_active` | Last API call timestamp |
| `error_rate` | Failed API calls / total |

### Audit Trail

All agent actions are audited:

| Event | Trigger |
|-------|---------|
| `agent.registered` | Agent created |
| `agent.token_exchanged` | Token delegated |
| `agent.scopes_changed` | Scopes updated |
| `agent.suspended` | Agent disabled |
| `agent.activated` | Agent re-enabled |
| `agent.deleted` | Agent removed |

Query agent activity:
```bash
curl "https://api.ggid.example.com/api/v1/audit/events?actor_id=$AGENT_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## 5. Rotation

### Rotate Agent Secret

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/$AGENT_ID/rotate-secret \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response**:
```json
{
  "client_secret": "new-secret",
  "old_secret_expires_at": "2025-01-25T00:00:00Z"
}
```

Old secret remains valid for 24-hour grace period, then invalidated.

## 6. Suspension & Revocation

### Suspend Agent

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/$AGENT_ID/suspend \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"reason": "Security investigation"}'
```

Suspension:
1. Agent marked `suspended`
2. All active tokens invalidated (jti blacklist)
3. New token exchanges rejected (403)
4. Audit event: `agent.suspended`

### Reactivate Agent

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/$AGENT_ID/activate \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Delete Agent

```bash
curl -X DELETE https://api.ggid.example.com/api/v1/agents/$AGENT_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

Deletion is permanent. All tokens revoked. Audit history retained.

## Security Best Practices

- [ ] Minimum scopes (principle of least privilege)
- [ ] `max_delegation_depth` <= 3
- [ ] Secret rotation every 90 days
- [ ] Monitor for anomalous token exchange patterns
- [ ] Alert on delegation depth exceeded
- [ ] Suspend unused agents (> 30 days inactive)
- [ ] MCP server URLs validated against allowlist
- [ ] All agent actions audited with full delegation chain

## See Also

- [Delegation Guide](delegation-guide.md)
- [AI Agent Identity](ai-agent-identity.md)
- [Token Exchange RFC 8693](../research/token-exchange-rfc8693.md)
- [REST API Reference](../api/rest-api.md)
