# Zero Trust Architecture Guide

> How GGID implements Zero Trust IAM principles: never trust, always verify.

---

## Zero Trust Pillars in GGID

### 1. Continuous Verification

Every request is verified — no implicit trust:

```
Request → Gateway verifies JWT (signature + exp + jti) → Backend verifies scope → DB enforces RLS
```

- **JWT anti-replay**: `jti` tracked via Redis SETNX
- **Token expiry**: 15-minute access tokens
- **Session validation**: Every request hits Redis for session status

### 2. Least Privilege

```bash
# RBAC: User gets only their role's permissions
curl -X POST .../api/v1/policies/check \
  -d '{"user_id":"usr_abc","action":"write","resource":"users"}'
# {"allowed": true, "reason": "role_permission_match"}

# ABAC: Context-aware denial
curl -X POST .../api/v1/policies/evaluate \
  -d '{
    "user_id":"usr_abc",
    "action":"write",
    "resource":"financial_data",
    "attributes": {"time":"02:00", "ip":"10.0.0.5"}
  }'
# {"allowed": false, "reason": "deny:outside_business_hours"}
```

### 3. Microsegmentation

- Each service owns its data (no shared DB access)
- gRPC TLS between services (commit 6a0eced)
- Tenant isolation via PostgreSQL RLS
- Network policies in K8s (NetworkPolicy resources)

### 4. Assume Breach

- Audit hash chain (tamper detection)
- Rate limiting (DDoS protection)
- PII obfuscation in audit logs
- Secrets in key manager (not env files)

---

## Implementation Checklist

| Control | Implementation | Status |
|---------|--------------|--------|
| mTLS between services | gRPC TLS | Done |
| JWT verification | RS256 + JWKS + jti | Done |
| RBAC | Role-based checks | Done |
| ABAC | Policy engine | Done |
| Tenant isolation | PostgreSQL RLS | Done |
| Rate limiting | Token bucket + adaptive | Done |
| Audit integrity | Hash chain | Done |
| Secret management | keys.env / K8s secrets | Done |

---

*See: [Security Overview](../architecture/security-overview.md) | [ABAC Policy](abac-policy.md) | [Multi-Tenant Guide](multi-tenant-guide.md)*

*Last updated: 2025-07-11*
