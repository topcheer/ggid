# Cross-Tenant Security

This guide covers tenant isolation architecture, cross-tenant attack vectors, protective measures, and testing in GGID's multi-tenant environment.

## Tenant Isolation Architecture

### Isolation Models

| Model | Description | Security | GGID Usage |
|---|---|---|---|
| Row-Level Security (RLS) | Shared DB, shared schema, tenant_id filter | High | Primary |
| Schema-per-tenant | Shared DB, separate schema per tenant | Very High | Optional |
| DB-per-tenant | Separate database per tenant | Maximum | Enterprise |

### Row-Level Security (RLS)

GGID uses PostgreSQL RLS as the primary isolation mechanism:

```sql
-- Enable RLS on all tenant-scoped tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_clients ENABLE ROW LEVEL SECURITY;

-- Policy: users can only see their tenant's data
CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Apply to all tenant-scoped tables
CREATE POLICY tenant_isolation ON roles
  FOR ALL
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Tenant Context Injection

Every request sets the tenant context before database access:

```go
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get tenant ID from JWT (priority) or header
        tenantID := getTenantFromJWT(r)
        if tenantID == "" {
            tenantID = r.Header.Get("X-Tenant-ID")
        }
        if tenantID == "" {
            writeError(w, 401, "missing_tenant")
            return
        }

        // Set tenant context for DB queries
        ctx := context.WithValue(r.Context(), TenantKey, tenantID)
        // Execute SET LOCAL app.tenant_id in DB transaction
        ctx = WithTenantContext(ctx, tenantID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Cross-Tenant Attack Vectors

### 1. Insecure Direct Object Reference (IDOR)

**Attack**: User in Tenant A requests a resource by ID that belongs to Tenant B.

**Example**:
```bash
# Tenant A user tries to access Tenant B's user
GET /api/v1/users/660e8400-tenantB-user-id
Authorization: Bearer <tenantA-token>
```

**GGID Defense**: RLS policy rejects the query — `WHERE tenant_id = 'tenantA'` is automatically applied, so Tenant B's user is not found (404).

### 2. Header Injection (Tenant Spoofing)

**Attack**: User sends a forged `X-Tenant-ID` header to impersonate another tenant.

```bash
GET /api/v1/users
Authorization: Bearer <tenantA-token>
X-Tenant-ID: tenantB-uuid  # Forged header
```

**GGID Defense**: JWT `tenant_id` claim takes priority over the header:

```go
func getTenantID(r *http.Request) string {
    // JWT claim always wins over header
    if claims := getJWTClaims(r); claims != nil {
        if tid, ok := claims["tenant_id"].(string); ok {
            return tid  // Use JWT, ignore header
        }
    }
    // Header only trusted if no JWT (e.g., service-to-service)
    return r.Header.Get("X-Tenant-ID")
}
```

### 3. Token Replay Across Tenants

**Attack**: Stolen token from Tenant A used to access Tenant B.

**GGID Defense**:
- JWT `aud` claim validated against expected audience
- JWT `tenant_id` claim bound to DB session
- Token revocation is tenant-scoped

### 4. Query Parameter Injection

**Attack**: Tenant ID injected via query parameter to bypass RLS.

```bash
GET /api/v1/users?tenant_id=tenantB
```

**GGID Defense**: Gateway injects tenant_id from JWT, not from client input:

```go
// Gateway proxy.Director
func director(req *http.Request) {
    tenantID := getTenantFromJWT(req)
    // Inject as query param for downstream services
    q := req.URL.Query()
    q.Set("tenant_id", tenantID)  // Overwrite any client-provided value
    req.URL.RawQuery = q.Encode()
}
```

### 5. Cross-Tenant Token Forgery

**Attack**: Attacker modifies JWT payload to change tenant_id.

**GGID Defense**: JWT signature verification prevents tampering. Any modification to `tenant_id` invalidates the signature.

### 6. Shared Resource Leakage

**Attack**: Shared infrastructure (Redis, NATS) leaks data across tenants.

**GGID Defense**:
- Redis keys are tenant-prefixed: `ggid:tenant:{tenantID}:session:{sessionID}`
- NATS subjects include tenant ID: `audit.events.{tenantID}`
- Connection pools are shared but all operations are tenant-scoped

## Protective Measures

### Tenant Context Validation

```go
func ValidateTenantContext(r *http.Request, resourceTenantID string) error {
    requestTenantID := getTenantID(r)
    if requestTenantID != resourceTenantID {
        // Log potential cross-tenant access attempt
        audit.Log(AuditEvent{
            Type:   "cross_tenant_access_attempt",
            UserID: getUserID(r),
            TenantID: requestTenantID,
            ResourceTenantID: resourceTenantID,
            IP: clientIP(r),
        })
        return ErrCrossTenantAccess
    }
    return nil
}
```

### JWT Claim Binding

| Claim | Binding | Validation |
|---|---|---|
| `sub` | User ID | Match against resource owner |
| `tenant_id` | Tenant scope | Match against resource tenant |
| `aud` | Intended audience | Match expected API |
| `iss` | Token issuer | Match GGID issuer URL |
| `scope` | Authorized scopes | Check required scope |

### Query Param Injection Prevention

Gateway-level tenant_id injection ensures downstream services receive the correct tenant:

```go
// Gateway: always overwrite tenant_id from JWT, never trust client input
func injectTenantID(req *http.Request) {
    claims := extractJWTClaims(req)
    tenantID := claims["tenant_id"].(string)

    // Query params
    q := req.URL.Query()
    q.Set("tenant_id", tenantID)
    req.URL.RawQuery = q.Encode()

    // JSON body (for POST/PUT/PATCH)
    if hasJSONBody(req) {
        injectTenantIntoBody(req, tenantID)
    }

    // Headers
    req.Header.Set("X-Tenant-ID", tenantID)
}
```

### Tenant Spoofing Detection

```go
func DetectTenantSpoofing(r *http.Request) bool {
    jwtTenant := getTenantFromJWT(r)
    headerTenant := r.Header.Get("X-Tenant-ID")

    // If both present and different, it's spoofing
    if jwtTenant != "" && headerTenant != "" && jwtTenant != headerTenant {
        audit.Log(AuditEvent{
            Type:       "tenant_spoofing_detected",
            JWTTenant:  jwtTenant,
            HeaderTenant: headerTenant,
            IP:         clientIP(r),
            Severity:   "high",
        })
        return true
    }
    return false
}
```

## Multi-Tenant API Design Principles

### 1. Tenant Context Always from JWT

Never accept tenant_id from client-controlled input as the primary source. JWT claim is authoritative.

### 2. Tenant-Scoped Resources

All resources are scoped under a tenant. No global resources accessible via API (except platform admin).

### 3. Tenant-Scoped Audit

All audit events include `tenant_id`. Audit queries are automatically scoped.

### 4. No Cross-Tenant References

Resources cannot reference resources in other tenants. Foreign keys include `tenant_id` in composite keys.

### 5. Tenant-Scoped Rate Limits

Rate limits are per-tenant, preventing one tenant's traffic from affecting another.

### 6. Tenant-Scoped Encryption

If using tenant-specific encryption keys, keys are derived from tenant ID + master key.

## Tenant Isolation Testing Plan

### Automated Tests

```go
func TestCrossTenantAccess(t *testing.T) {
    // Setup: two tenants, two users
    tenantA := createTenant("A")
    tenantB := createTenant("B")
    userA := createUser(tenantA, "userA")
    userB := createUser(tenantB, "userB")

    tokenA := login(userA)
    tokenB := login(userB)

    // Test 1: User A cannot read User B
    resp := getRequest(tokenA, "/api/v1/users/"+userB.ID)
    assert.Equal(t, 404, resp.StatusCode)

    // Test 2: User A cannot list Tenant B users
    resp = getRequest(tokenA, "/api/v1/users", "X-Tenant-ID", tenantB.ID)
    assert.Equal(t, 200, resp.StatusCode) // Returns Tenant A users only

    // Test 3: User A cannot modify User B
    resp = putRequest(tokenA, "/api/v1/users/"+userB.ID, `{"name":"hacked"}`)
    assert.Equal(t, 404, resp.StatusCode)

    // Test 4: Header spoofing rejected
    resp = getRequest(tokenA, "/api/v1/users", "X-Tenant-ID", tenantB.ID)
    body := parseBody(resp)
    for _, user := range body.Users {
        assert.Equal(t, tenantA.ID, user.TenantID)
    }

    // Test 5: Audit logs are tenant-scoped
    resp = getRequest(tokenA, "/api/v1/audit/events")
    for _, event := range parseBody(resp).Events {
        assert.Equal(t, tenantA.ID, event.TenantID)
    }
}
```

### Manual Verification

```bash
# 1. Create two tenants
TENANT_A=$(create_tenant "Acme")
TENANT_B=$(create_tenant "Globex")

# 2. Create users in each
USER_A=$(create_user $TENANT_A "alice@acme.com")
USER_B=$(create_user $TENANT_B "bob@globex.com")

# 3. Get tokens
TOKEN_A=$(login $USER_A)
TOKEN_B=$(login $USER_B)

# 4. Verify isolation
curl -H "Authorization: Bearer $TOKEN_A" /api/v1/users/$USER_B
# Expected: 404

# 5. Verify header spoofing blocked
curl -H "Authorization: Bearer $TOKEN_A" -H "X-Tenant-ID: $TENANT_B" /api/v1/users
# Expected: Only Tenant A users

# 6. Verify audit isolation
curl -H "Authorization: Bearer $TOKEN_A" /api/v1/audit/events | jq '.events[].tenant_id'
# Expected: All events have Tenant A ID
```

### RLS Verification

```sql
-- Verify RLS is enabled
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE relname IN ('users', 'roles', 'audit_events')
AND relrowsecurity = true;

-- Test isolation directly
SET app.tenant_id = 'tenant-a-uuid';
SELECT count(*) FROM users;  -- Should return tenant A count only

SET app.tenant_id = 'tenant-b-uuid';
SELECT count(*) FROM users;  -- Should return tenant B count only
```

## Best Practices

1. **JWT is authoritative** — never trust client headers over JWT claims
2. **RLS on every table** — defense in depth, even if app-layer filters exist
3. **Tenant-prefixed cache keys** — prevent Redis cross-tenant leakage
4. **Tenant-scoped NATS subjects** — prevent event bus cross-tenant access
5. **Audit cross-tenant attempts** — log and alert on any mismatch
6. **Test isolation continuously** — automated tests in CI/CD
7. **Gateway injection** — tenant_id injected at gateway, not trusted from client
8. **Platform admin is special** — separate auth scope for cross-tenant operations
9. **No shared OAuth clients** — each tenant has its own clients
10. **Encrypt per-tenant** — if feasible, derive encryption keys per tenant