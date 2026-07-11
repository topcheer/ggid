# Custom JWT Claims

> How to add custom claims to JWT tokens, read them in SDKs, and use them in the policy engine.

---

## Standard JWT Claims

GGID JWTs include these standard claims:

```json
{
  "sub": "usr_abc123",           // User ID
  "tenant_id": "00000000-...",   // Tenant ID
  "scope": "read:users write:users",  // OAuth scopes
  "roles": ["admin", "editor"],  // Assigned roles
  "iss": "ggid-auth",            // Issuer
  "aud": "ggid-gateway",         // Audience
  "exp": 1700003600,             // Expiry
  "iat": 1700000000,             // Issued at
  "jti": "unique-token-id"      // JWT ID (anti-replay)
}
```

---

## Adding Custom Claims

### Via Auth Hooks

Use the `pre-token-issue` hook to inject custom claims:

```go
// Register a pre-token-issue hook
authService.RegisterHook("pre-token-issue", func(ctx context.Context, user *User, claims *Claims) error {
    // Add custom claims
    claims.Custom["department"] = user.Department
    claims.Custom["clearance_level"] = user.ClearanceLevel
    claims.Custom["employee_id"] = user.EmployeeID
    return nil
})
```

### Via OAuth Token Exchange

When using token exchange (RFC 8693), additional claims are included:

```json
{
  "sub": "usr_human123",
  "act": {
    "sub": "agent-claude",
    "type": "ai_agent"
  },
  "scope": "read:tools"
}
```

---

## Reading Custom Claims in SDK

### Go SDK

```go
verifier := ggid.NewVerifier("http://localhost:8080", "secret")
claims, err := verifier.Verify(ctx, token)

// Standard claims
userID := claims.UserID
tenantID := claims.TenantID
scope := claims.Scope

// Custom claims
dept := claims.Custom["department"]
clearance := claims.Custom["clearance_level"]
```

### Node.js SDK

```javascript
const claims = await verifier.verify(token);

// Standard
const userID = claims.sub;
const tenantID = claims.tenant_id;

// Custom
const dept = claims.department;
const clearance = claims.clearance_level;
```

### Python SDK

```python
claims = verifier.verify(token)

# Standard
user_id = claims.user_id
tenant_id = claims.tenant_id

# Custom
dept = claims.get("department")
clearance = claims.get("clearance_level")
```

---

## Using Custom Claims in Policy Engine

### ABAC Rules with Custom Claims

Custom claims can be used in ABAC attribute-based rules:

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "department_isolation",
    "effect": "allow",
    "conditions": "request.user.department == resource.department"
  }'
```

### Clearance Level Check

```json
{
  "name": "confidential_access",
  "effect": "allow",
  "conditions": "request.user.clearance_level >= 3 AND resource.classification <= request.user.clearance_level"
}
```

---

## Claim Transformation

Claims can be transformed during token issuance:

| Transformation | Example |
|---------------|---------|
| Add role-based scope | `if user.role == 'admin' then scope += 'admin:*'` |
| Add tenant context | `claims.tenant_name = tenant.name` |
| Add time-limited scope | `claims.exp = now + 15m` (short-lived) |
| Strip sensitive data | `delete claims.ssn` |

---

## Security Considerations

| Risk | Mitigation |
|------|------------|
| Claim injection | Only auth service can sign JWTs |
| Claim bloat | Limit custom claims to <500 bytes total |
| Sensitive data in JWT | JWT is base64, not encrypted — don't put secrets in claims |
| Claim tampering | HMAC signature protects all claims |

---

*Last updated: 2025-07-11*