# Cross-Tenant Trust Guide

This guide covers establishing trust between GGID tenants — trust establishment, scope delegation, token exchange, auditing, and revocation.

## Overview

In multi-tenant deployments, tenants may need to share resources or allow cross-tenant access. GGID provides controlled trust mechanisms to enable this safely.

## Trust Establishment

### Trust Model

```
Tenant A (source) ──trusts──> Tenant B (target)
  ↓                                ↓
Defines scopes             Accepts delegated tokens
allowed for B               from A's users
```

### Create Trust Relationship

```bash
curl -X POST https://api.ggid.example.com/api/v1/tenants/$TENANT_A_ID/trust \
  -H "Authorization: Bearer $ADMIN_TOKEN_A" \
  -H "X-Tenant-ID: $TENANT_A_ID" \
  -d '{
    "trusted_tenant_id": "tenant-b-uuid",
    "allowed_scopes": ["users:read", "policy:check"],
    "max_delegation_depth": 2,
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

### List Trust Relationships

```bash
curl https://api.ggid.example.com/api/v1/tenants/$TENANT_A_ID/trust \
  -H "Authorization: Bearer $ADMIN_TOKEN_A" \
  -H "X-Tenant-ID: $TENANT_A_ID"
```

**Response**:
```json
{
  "trusts": [
    {
      "trusted_tenant_id": "tenant-b-uuid",
      "trusted_tenant_name": "Partner Corp",
      "allowed_scopes": ["users:read"],
      "max_delegation_depth": 2,
      "created_at": "2025-01-01T00:00:00Z",
      "expires_at": "2025-12-31T23:59:59Z",
      "status": "active"
    }
  ]
}
```

### Revoke Trust

```bash
curl -X DELETE https://api.ggid.example.com/api/v1/tenants/$TENANT_A_ID/trust/$TENANT_B_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN_A"
```

## Token Exchange Between Tenants

### Cross-Tenant Token Request

```bash
curl -X POST https://api.ggid.example.com/api/v1/agents/cross-tenant-token \
  -H "Authorization: Bearer $USER_TOKEN_A" \
  -H "X-Tenant-ID: $TENANT_A_ID" \
  -d '{
    "target_tenant_id": "tenant-b-uuid",
    "requested_scopes": ["users:read"]
  }'
```

**Response**:
```json
{
  "access_token": "eyJhbG...",
  "token_type": "Bearer",
  "expires_in": 900,
  "issuer_tenant": "tenant-a-uuid",
  "target_tenant": "tenant-b-uuid",
  "scopes": ["users:read"],
  "delegation_chain": [
    {
      "user_id": "user-uuid",
      "tenant_id": "tenant-a-uuid",
      "scopes": ["users:read"],
      "depth": 1
    }
  ]
}
```

### Token Validation (Target Tenant)

The target tenant (B) validates the cross-tenant token:

1. Verify JWT signature (from tenant A's JWKS)
2. Check trust relationship exists (A trusts B)
3. Verify scopes are within allowed set
4. Check delegation depth <= max
5. Verify token not expired
6. Check jti not replayed

## Scope Delegation Rules

| Rule | Description |
|------|-------------|
| Scope narrowing | Requested scopes must be subset of allowed |
| Depth limiting | Each delegation increments depth counter |
| Tenant binding | Token explicitly bound to target tenant |
| Time-limited | Cross-tenant tokens have shorter TTL (5-15 min) |
| No chain extension | Target tenant cannot re-delegate |

## Auditing

### Cross-Tenant Audit Events

| Event | Description |
|-------|-------------|
| `trust.created` | Trust relationship established |
| `trust.revoked` | Trust relationship terminated |
| `cross_tenant.token_issued` | Cross-tenant token exchanged |
| `cross_tenant.token_used` | Cross-tenant token used at target |
| `cross_tenant.token_denied` | Token rejected (scope/depth/expiry) |

### Audit Query

```bash
curl "https://api.ggid.example.com/api/v1/audit/events?event_type=cross_tenant" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

## Security Considerations

| Risk | Mitigation |
|------|-----------|
| Privilege escalation | Strict scope narrowing |
| Token theft | Short TTL + jti anti-replay |
| Trust滥用 | Audit all cross-tenant access |
| Delegation chains too deep | Max depth enforcement |
| Stale trust | Expiry dates on all trust relationships |

## Use Cases

| Scenario | Trust Type |
|----------|-----------|
| Parent/subsidiary companies | Full trust, broad scopes |
| B2B partner integration | Limited trust, specific scopes |
| Shared service (multi-tenant SaaS) | Service-specific trust |
| M&A transition | Temporary full trust |

## See Also

- [Delegation Guide](delegation-guide.md)
- [Multi-Tenant Architecture](multi-tenant-architecture.md)
- [Token Exchange RFC 8693](../research/token-exchange-rfc8693.md)
- [AI Agent Lifecycle](ai-agent-lifecycle.md)
