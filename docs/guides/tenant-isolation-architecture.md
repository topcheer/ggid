# Tenant Isolation Architecture

RLS enforcement, per-tenant schema vs shared schema, query-level isolation, Redis/NATS namespacing, file storage isolation, and testing isolation.

## Isolation Model

GGID uses **shared-schema with Row-Level Security (RLS)** — the most scalable approach for multi-tenant SaaS.

| Model | Isolation | Complexity | Cost | GGID Decision |
|-------|-----------|-----------|------|---------------|
| Database per tenant | Strongest | High (many DBs) | $$$ | ❌ |
| Schema per tenant | Strong | Medium | $$ | ❌ |
| Shared schema + RLS | Strong (if correct) | Low | $ | ✅ Chosen |

## PostgreSQL Row-Level Security

### Enable RLS

```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
-- All tables with tenant_id have RLS
```

### Policy

```sql
CREATE POLICY tenant_isolation ON users
  FOR ALL
  USING (tenant_id = current_setting('app.tenant_id')::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
```

### Set Tenant Context

Every request sets the tenant context at the transaction level:

```go
func WithTenant(ctx context.Context, db *pgxpool.Pool, tenantID string, fn func(*pgx.Tx) error) error {
    tx, err := db.Begin(ctx)
    if err != nil { return err }
    defer tx.Rollback(ctx)

    // SET LOCAL applies only to this transaction
    _, err = tx.Exec(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
    if err != nil { return err }

    // All queries in this tx are filtered by RLS
    if err := fn(tx); err != nil { return err }

    return tx.Commit(ctx)
}
```

### RLS Enforcement

```
Query: SELECT * FROM users
  ↓
PostgreSQL RLS rewrites to:
  SELECT * FROM users WHERE tenant_id = current_setting('app.tenant_id')
  ↓
Result: Only rows for the current tenant
```

**Even if a query omits tenant_id filter, RLS enforces isolation.** This is defense-in-depth.

## Query-Level Isolation

### JWT Claim as Source of Truth

```go
func extractTenantID(r *http.Request) string {
    // JWT claim takes priority over header (prevent spoofing)
    claims := getClaimsFromContext(r.Context())
    return claims["tenant_id"].(string)
}
```

### Tenant Middleware

```go
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := extractTenantID(r)
        if tenantID == "" {
            http.Error(w, "missing tenant", 401)
            return
        }

        ctx := context.WithValue(r.Context(), "tenant_id", tenantID)

        // Verify tenant exists and is active
        tenant := getTenant(tenantID)
        if tenant.Status != "active" {
            http.Error(w, "tenant suspended", 403)
            return
        }

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Redis Namespacing

```
session:{tenant_id}:{session_id}
user:{tenant_id}:{user_id}:sessions
ratelimit:{tenant_id}:{client_ip}
cache:{tenant_id}:{key}
```

### Per-Tenant Rate Limiting

```go
func rateLimitKey(tenantID, clientID string) string {
    return fmt.Sprintf("ratelimit:%s:%s", tenantID, clientID)
}
// Each tenant has independent rate limit buckets
```

## NATS Namespacing

```
audit.{tenant_id}.{action}           # Audit events
provisioning.{tenant_id}.{event}     # Provisioning
webhook.{tenant_id}.{event_type}     # Webhook delivery
```

### Per-Tenant Subject Filtering

```go
// Consumer only receives events for its tenant
js.Subscribe("audit."+tenantID+".>", handler)
```

## File Storage Isolation

```
s3://ggid-storage/
  └── tenants/
      └── {tenant_id}/
          ├── exports/
          ├── avatars/
          ├── documents/
          └── backups/
```

### Access Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Deny",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::ggid-storage/tenants/*",
    "Condition": {
      "StringNotEquals": {
        "s3:prefix": "${jwt:tenant_id}/*"
      }
    }
  }]
}
```

## Cross-Tenant Prevention

### RLS Bypass Prevention

```sql
-- Only service role can bypass RLS
ALTER TABLE users FORCE ROW LEVEL SECURITY;
-- Even table owner is subject to RLS

-- Migration role bypasses RLS (DDL only)
GRANT BYPASSRLS TO ggid_migrate;
-- Application role never has BYPASSRLS
```

### Global Admin (Cross-Tenant)

```sql
-- Super admin role bypasses RLS
CREATE ROLE ggid_super_admin BYPASSRLS;
-- Only assigned to specific service account, heavily audited
```

```go
// Super admin operations are logged
func crossTenantQuery(ctx context.Context, query string) {
    audit.Log("cross_tenant_query", map[string]interface{}{
        "query": query,
        "caller": getCallerID(ctx),
        "timestamp": time.Now(),
    })
    // Execute with BYPASSRLS role
}
```

## Testing Tenant Isolation

### Automated Test

```go
func TestTenantIsolation(t *testing.T) {
    // Create user in tenant A
    tx1 := beginTx(tenantA)
    tx1.Exec("INSERT INTO users (id, tenant_id, email) VALUES ('u1', $1, 'a@corp.com')", tenantA)
    tx1.Commit()

    // Query from tenant B — should NOT see tenant A's user
    tx2 := beginTx(tenantB)
    var count int
    tx2.QueryRow("SELECT COUNT(*) FROM users WHERE email = 'a@corp.com'").Scan(&count)

    assert.Equal(t, 0, count, "tenant B should not see tenant A's data")
}
```

### Penetration Test

```
1. Authenticate as tenant A user
2. Try to query tenant B's resources (user IDs, role IDs)
3. Verify all return empty / 404
4. Try to set X-Tenant-ID header to tenant B
5. Verify JWT claim overrides header
6. Try SQL injection to bypass RLS
7. Verify RLS prevents bypass
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Cross-tenant query attempts | Any → security incident |
| RLS policy violations | 0 (should never happen) |
| Missing tenant_id in query | Any → bug in middleware |
| Tenant data count anomaly | Tenant with unexpected row count |

## See Also

- [Tenant Provisioning API](tenant-provisioning-api.md)
- [Database Security](database-security.md)
- [Data Residency Architecture](data-residency-architecture.md)
- [Gateway Architecture](gateway-architecture.md)
