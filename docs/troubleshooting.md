# GGID Troubleshooting Guide

Common issues and their solutions, organized by category.

---

## Docker / Startup Issues

### Q: `docker compose up` fails with "port is already allocated"

**Cause:** Another process is using a port that GGID needs (8080, 5432, 6379, etc.).

**Fix:** Identify and stop the conflicting process, or remap the port.

```bash
# Find what's using port 8080
sudo lsof -i :8080

# Option 1: Stop the conflicting process
kill <PID>

# Option 2: Remap the port in docker-compose.yaml
# Change "8080:8080" to "9080:8080"
```

---

### Q: Containers start but immediately exit

**Cause:** Missing dependencies (database not ready, RSA keys not generated).

**Fix:** Check logs and ensure init containers ran.

```bash
# Check which containers exited
docker compose ps -a

# Check logs of the failing container
docker compose logs identity
docker compose logs auth

# Common issues:
# - "connection refused" → postgres not ready, wait or restart
# - "no such file" → keygen or migrate didn't run, restart them
```

**Force restart init containers:**
```bash
docker compose up -d keygen migrate
docker compose up -d --force-recreate identity auth
```

---

### Q: `migrate` container fails with "database already initialized"

**Cause:** The idempotent check detected existing tables. This is normal.

**Fix:** No action needed — the container exits with code 0 after printing
"Database already initialized, skipping migrations".

If you need to re-run migrations from scratch:
```bash
docker compose down -v  # WARNING: deletes all data
docker compose up -d
```

---

### Q: Build fails with "no such file or directory" during Docker build

**Cause:** Build context is wrong, or `console/public/` directory is missing.

**Fix:**
```bash
# Ensure console/public exists
mkdir -p console/public/.gitkeep

# Rebuild from scratch
docker compose build --no-cache
```

---

### Q: Console shows a blank page or connection error

**Cause:** Gateway is not running or Console can't reach it.

**Fix:**
```bash
# Verify gateway is healthy
curl http://localhost:8080/healthz

# Check Console logs
docker compose logs console

# Verify the Console env var
# GATEWAY_URL should be "http://gateway:8080" (Docker internal network)
```

---

## Database Issues

### Q: Services fail with "connection refused" to PostgreSQL

**Cause:** PostgreSQL container is not running or not yet healthy.

**Fix:**
```bash
# Check postgres status
docker compose ps postgres

# Check logs
docker compose logs postgres

# Wait for it to become healthy
docker compose up -d postgres
# Wait 10 seconds, then restart dependent services
docker compose restart identity auth policy org audit
```

---

### Q: "relation does not exist" errors

**Cause:** Database migrations haven't been applied.

**Fix:**
```bash
# Run migrations manually
docker compose up migrate

# Or run via psql
docker exec -it ggid-postgres psql -U ggid -d ggid -f /migrations/001_init.sql
```

---

### Q: RLS policy blocks all queries (returns empty results)

**Cause:** The application is using a superuser role (bypasses RLS) in development,
but a non-superuser role in production that requires `SET LOCAL app.tenant_id`.

**Fix:** Ensure `SET LOCAL app.tenant_id` is called at the start of every transaction:

```go
// In your repository code:
_, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
if err != nil {
    return err
}
// Then run queries...
```

> **Note:** `SET LOCAL` does NOT support `$1` parameters in pgx v5.
> Use `fmt.Sprintf` with a validated UUID string.

---

### Q: Database password incorrect

**Fix:** Reset the password and restart:

```bash
docker exec -it ggid-postgres psql -U ggid -c "ALTER USER ggid PASSWORD 'newpassword';"

# Update docker-compose.yaml and restart
docker compose down
docker compose up -d
```

> If you've forgotten the superuser password, you need to recreate the volume:
> `docker volume rm ggid-pgdata` (destroys all data).

---

## JWT / Authentication Issues

### Q: All API calls return 401 Unauthorized

**Cause:** The JWT is missing, expired, or invalid.

**Debugging steps:**
```bash
# 1. Verify the token is present
echo $TOKEN | wc -c  # should be > 100 chars

# 2. Check if the token is expired
echo $TOKEN | cut -d. -f2 | base64 -d 2>/dev/null | jq .exp
# Compare with current time: date +%s

# 3. Verify the token format
curl -s -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  $GW/api/v1/users
```

**Fix:**
```bash
# Get a fresh token
TOKEN=$(curl -s -X POST $GW/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"demo","password":"SecurePass@123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
```

---

### Q: Login returns 429 Too Many Requests

**Cause:** Rate limiting kicks in after ~5 failed login attempts from the same IP.

**Fix:**
```bash
# Wait for the rate limit window to expire (60 seconds)
sleep 60

# Or restart the auth container to clear the rate limiter
docker compose restart auth

# Then retry login
```

---

### Q: JWT verification fails in the SDK

**Cause:** The SDK can't reach the JWKS endpoint, or the key has rotated.

**Fix:**
```bash
# 1. Verify JWKS endpoint is accessible
curl http://localhost:8080/.well-known/jwks.json

# 2. Check that the kid in the JWT header matches a key in JWKS
echo $TOKEN | cut -d. -f1 | base64 -d 2>/dev/null | jq .kid

# 3. If keys were rotated, clear the SDK cache (restart your app)
```

---

### Q: Register returns 409 Conflict

**Cause:** The username already exists within the tenant.

**Fix:** Use a different username, or delete the existing user first.

```bash
# Check if user exists
curl -s "$GW/api/v1/users?search=jane" \
  -H "$AUTH" -H "X-Tenant-ID: $TENANT"

# Delete if needed
curl -X DELETE "$GW/api/v1/users/USER_ID" \
  -H "$AUTH" -H "X-Tenant-ID: $TENANT"
```

---

### Q: Register returns 500 Internal Server Error

**Cause:** The `username` field is empty. The register handler reads `username`
(not `email`) as the credential identifier.

**Fix:** Ensure the request includes a non-empty `username` field:

```bash
curl -s -X POST $GW/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "username": "jane.doe",   <-- THIS IS REQUIRED
    "email": "jane@example.com",
    "password": "SecurePass@123"
  }'
```

---

## Gateway Issues

### Q: API returns 502 Bad Gateway

**Cause:** The Gateway can reach the backend service, but the backend returned an error
or is misconfigured.

**Fix:**
```bash
# Check gateway logs
docker compose logs gateway

# Check which backend is failing
curl http://localhost:8080/healthz/ready

# Restart the failing service
docker compose restart identity  # or auth, policy, org, audit
```

---

### Q: API returns 503 Service Unavailable

**Cause:** A backend service is not responding to health checks.

**Fix:**
```bash
# Check all container statuses
docker compose ps

# Check health check failures
docker inspect ggid-auth --format='{{json .State.Health}}' | jq .

# Common cause: service is OOM-killed
docker compose logs auth | grep -i "killed\|oom"
```

---

### Q: Gateway returns 404 for an endpoint that should work

**Cause:** The route is not configured in the Gateway, or the path prefix is wrong.

**Fix:**
```bash
# Check gateway route configuration
docker compose logs gateway | grep "route"

# Verify the service is registered
# Gateway routes:
#   /api/v1/auth    → auth:9001
#   /api/v1/users   → identity:8080
#   /api/v1/roles   → policy:8070
#   /api/v1/policies → policy:8070
#   /api/v1/orgs    → org:8071
#   /api/v1/audit   → audit:8072

# If adding a new route, update config.Default() in
# services/gateway/internal/config/config.go
```

---

### Q: Gateway injects tenant_id but backend still returns 400 "missing tenant"

**Cause:** The backend service expects `tenant_id` in a different location
(query param vs JSON body) than what the Gateway injects.

**Fix:** The Gateway injects tenant_id as:
- Query param for GET requests
- JSON body field for POST/PUT/PATCH

If your endpoint reads it from a header instead, add explicit parsing:
```go
// In your handler
tenantIDStr := r.Header.Get("X-Tenant-ID")
```

---

## Policy / RBAC Issues

### Q: Create role returns 500 Internal Server Error

**Cause:** The `key` field is empty. The roles table has a `UNIQUE(tenant_id, key)`
constraint — an empty key conflicts with existing rows.

**Fix:** Always provide a unique, non-empty `key`:

```bash
curl -s -X POST $GW/api/v1/roles \
  -H "$AUTH" -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -d '{"key":"editor","name":"Content Editor"}'
#                  ^^^^ must be unique within tenant
```

---

### Q: Permission check always returns `allowed: false`

**Cause:** The user has no roles assigned, or the policy doesn't match.

**Debugging:**
```bash
# 1. List user's roles
curl -s "$GW/api/v1/users/$USER_ID/roles" -H "$AUTH" -H "X-Tenant-ID: $TENANT"

# 2. Check role permissions
curl -s "$GW/api/v1/roles/$ROLE_ID/permissions" -H "$AUTH"

# 3. List policies
curl -s "$GW/api/v1/policies?tenant_id=$TENANT" -H "$AUTH"

# 4. Test with explicit context
curl -s -X POST $GW/api/v1/policies/check \
  -H "$AUTH" -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -d '{"user_id":"...","resource":"documents:drafts","action":"read","context":{"department":"engineering"}}'
```

---

## NATS / Audit Issues

### Q: Audit events are not being recorded

**Cause:** NATS is down, or the audit consumer is not running.

**Fix:**
```bash
# 1. Check NATS health
curl http://localhost:8222/healthz

# 2. Check NATS is running
docker compose ps nats

# 3. Check audit service logs
docker compose logs audit | grep -i "nats\|error"

# 4. Restart NATS and audit
docker compose restart nats
docker compose restart audit
```

> **Note:** Audit event publishing is best-effort. If NATS is down when an
> event occurs, that event is lost (not retried). This is by design.

---

### Q: NATS fails to start with "stream already exists"

**Cause:** A previous NATS instance left stale JetStream data.

**Fix:**
```bash
docker compose down
docker volume rm deploy_ggid-nats-data  # if using a named volume
# Or
docker compose up -d --force-recreate nats
```

---

### Q: Audit query returns events from all tenants

**Cause:** RLS is not being enforced (superuser connection), or `tenant_id`
filter is missing.

**Fix:** Ensure the audit service's `ListEvents` uses the tenant_id from the
filter, and the database connection is using a non-superuser role.

---

## LDAP Issues

### Q: LDAP login fails with "invalid credentials"

**Cause:** LDAP server is not running, or bind credentials are wrong.

**Fix:**
```bash
# 1. Check LDAP is running
docker compose ps ldap

# 2. Test LDAP connectivity
docker exec ggid-ldap ldapsearch -x -H ldap://localhost:389 \
  -D "cn=admin,dc=corp,dc=local" -w admin123 -b "dc=corp,dc=local"

# 3. Verify env vars in auth service
docker compose exec auth env | grep LDAP
```

---

### Q: LDAP users can't auto-provision

**Cause:** `LDAP_AUTO_PROVISION` is not set to `true`.

**Fix:**
```yaml
# In docker-compose.yaml, auth service environment:
LDAP_AUTO_PROVISION: "true"
```

Then restart auth: `docker compose restart auth`

---

## Performance Issues

### Q: API responses are slow (>1s latency)

**Cause:** Database queries without indexes, or connection pool exhaustion.

**Fix:**
```bash
# 1. Check Gateway metrics
curl http://localhost:8080/metrics | grep -i "latency\|duration"

# 2. Check PostgreSQL slow queries
docker exec ggid-postgres psql -U ggid -c \
  "SELECT query, mean_exec_time, calls FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"

# 3. Add indexes if missing (check your migration files)
# 4. Increase connection pool size in service configs
```

---

### Q: Auth service uses too much memory

**Cause:** JWKS cache, rate limiter, or session store consuming memory.

**Fix:**
```bash
# Check container memory usage
docker stats ggid-auth

# Reduce JWKS cache TTL
# Reduce session TTL
# Increase container memory limit in docker-compose.yaml
```

---

## FAQ

### Q: How do I reset everything to a clean state?

```bash
cd deploy
docker compose down -v  # removes all volumes (DB, LDAP, configs)
docker compose up -d    # starts fresh with new keys and migrations
```

### Q: How do I change the default tenant ID?

The default tenant `00000000-0000-0000-0000-000000000001` is created by
the migration scripts. To use a different tenant, create one via the API:

```bash
# Create a new tenant (requires admin access)
curl -s -X POST $GW/api/v1/orgs \
  -H "$AUTH" -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"My Company"}'
```

### Q: Can I run GGID without Docker?

Yes. Build each service binary and run them directly:

```bash
go build -o bin/identity ./services/identity/cmd
go build -o bin/auth ./services/auth/cmd
# ... etc

# Set environment variables and run
DATABASE_URL="postgres://..." ./bin/auth
```

You still need PostgreSQL, Redis, and NATS running.

### Q: How do I rotate JWT signing keys?

```bash
# 1. Generate a new key pair
openssl genpkey -algorithm RSA -out new_private.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in new_private.pem -out new_public.pem

# 2. Replace the keys in the configs volume
docker cp new_private.pem ggid-auth:/configs/rsa_private.pem
docker cp new_public.pem ggid-auth:/configs/rsa_public.pem

# 3. Restart auth and gateway
docker compose restart auth gateway

# 4. All existing JWTs become invalid — users must re-login
```

### Q: How do I see what's happening in real-time?

```bash
# All service logs
docker compose logs -f

# Specific service
docker compose logs -f auth

# Gateway access logs
docker compose logs -f gateway | grep "status"
```
