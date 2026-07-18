# Troubleshooting Guide

> Common GGID issues: symptoms, root causes, and solutions.

---

## JWT Verification Failures

### Symptom: `401 Unauthorized` on all API calls

**Cause 1: Expired token**

```bash
# Check token expiry
echo $JWT | cut -d. -f2 | base64 -d 2>/dev/null | jq .exp
# Compare with current time: date +%s
```

**Solution:** Refresh the token:
```bash
NEWTOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq -r .access_token)
```

**Cause 2: Wrong JWKS URL**

```bash
# Verify JWKS endpoint is reachable
curl http://localhost:8080/.well-known/jwks.json | jq .keys[0].kid
```

**Cause 3: Tenant mismatch**

The JWT `tenant_id` must match the `X-Tenant-ID` header. The JWT claim takes priority.

```bash
# Check tenant in token
echo $JWT | cut -d. -f2 | base64 -d 2>/dev/null | jq .tenant_id
```

---

## Database Connection Issues

### Symptom: `connection refused` or `database not reachable`

**Cause: Wrong DATABASE_URL or DB_HOST**

Policy/Org/Audit services use individual env vars (`DB_HOST`, `DB_PORT`, etc.), **not** `DATABASE_URL`. Auth and Identity use `DATABASE_URL`.

```bash
# Verify env vars in container
kubectl exec -it <pod> -n ggid -- env | grep DB_

# PostgreSQL connectivity test
kubectl exec -it <pod> -n ggid -- sh -c 'pg_isready -h $DB_HOST -p $DB_PORT'
```

**Solution:** Set correct env vars:

```yaml
# For Policy/Org/Audit:
DB_HOST: ggid-postgresql
DB_PORT: "5432"
DB_USER: ggid
DB_PASSWORD: ggid
DB_NAME: ggid

# For Auth/Identity:
DATABASE_URL: postgres://ggid:ggid@ggid-postgresql:5432/ggid?sslmode=disable
```

### Symptom: `relation "users" does not exist`

**Cause:** Migrations not run.

```bash
# Run migrations
kubectl exec -it <pod> -n ggid -- /app/migrate up
```

---

## NATS Connection Issues

### Symptom: `nats: no servers available` or audit events not persisting

**Cause 1: NATS not running**

```bash
kubectl get pods -n ggid | grep nats
# If not found, NATS not deployed or crashed
```

**Cause 2: Wrong NATS_URL**

```bash
# Verify NATS is reachable from service pod
kubectl exec -it <audit-pod> -n ggid -- sh -c 'nc -z $NATS_HOST $NATS_PORT && echo OK'
```

**Cause 3: JetStream not enabled**

NATS must be started with JetStream. Check the `-js` flag:

```bash
kubectl exec -it <nats-pod> -n ggid -- nats-server --help 2>&1 | grep jetstream
```

**Solution:** Ensure NATS deployment has `-js -m 8222` flags.

---

## Gateway 502 Bad Gateway

### Symptom: `502 Bad Gateway` from Gateway

**Cause 1: Backend service not ready**

```bash
# Check all service pods
kubectl get pods -n ggid -l app.kubernetes.io/component
```

**Cause 2: Wrong upstream service name/port**

Gateway routes to internal services. Verify the service exists:

```bash
kubectl get svc -n ggid
# ggid-gateway      ClusterIP   8080/TCP
# ggid-identity     ClusterIP   8080/TCP
# ggid-auth         ClusterIP   8080/TCP
# ...
```

**Cause 3: gRPC service not reachable**

Some services use gRPC for inter-service communication. Check:

```bash
# Test gRPC connectivity
kubectl exec -it <gateway-pod> -n ggid -- grpcurl ggid-identity:50051 list
```

---

## Tenant Isolation Issues

### Symptom: User sees data from another tenant

**Cause: `X-Tenant-ID` header spoofed without JWT verification**

The Gateway should resolve tenant from the JWT claim, not the header.

```bash
# Verify JWT tenant_id
echo $JWT | cut -d. -f2 | base64 -d | jq .tenant_id

# Verify Gateway resolves correctly
# Gateway middleware: JWT tenant_id > API key tenant > X-Tenant-ID header
```

**Cause: RLS policy not applied to a table**

```sql
-- Check if RLS is enabled
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class WHERE relname = 'your_table';

-- Enable if missing
ALTER TABLE your_table ENABLE ROW LEVEL SECURITY;
ALTER TABLE your_table FORCE ROW LEVEL SECURITY;
```

---

## Authentication Failures

### Symptom: `429 Too Many Requests` on login

**Cause:** Rate limiting (5 attempts per 60 seconds).

**Solution:** Wait 60 seconds or restart auth container:

```bash
kubectl rollout restart deployment ggid-auth -n ggid
```

### Symptom: `401` with correct credentials after password change

**Cause:** Old refresh tokens invalidated. Need fresh login.

```bash
JWT=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"admin","password":"NewPassword!"}' | jq -r .access_token)
```

---

## OAuth/OIDC Issues

### Symptom: `invalid_state` on OAuth callback

**Cause:** State parameter expired or replayed (one-time use via Redis).

**Solution:** Start a new OAuth flow. Don't reuse callback URLs.

### Symptom: `redirect_uri_mismatch`

**Cause:** Registered redirect URI doesn't match the one in the request.

```bash
# Check registered redirect URIs
curl -s http://localhost:8080/api/v1/oauth/clients/$CLIENT_ID \
  -H "Authorization: Bearer $JWT" | jq .redirect_uris
```

---

## SCIM 2.0 Issues

### Symptom: `400 Bad Request` on PATCH operation

**Cause:** Incorrect path syntax.

SCIM PATCH supports both dot and colon notation:

```
# Dot notation (most common):
urn:ietf:params:scim:schemas:extension:enterprise:2.0:User.department

# Colon notation (RFC 7644 §3.10):
urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department
```

---

## Docker Compose Issues

### Symptom: Services restart in a loop

**Cause:** Init container (migration) waiting for PostgreSQL.

```bash
# Check migration logs
docker logs ggid-migrate

# Check PostgreSQL is running
docker ps | grep postgres
```

### Symptom: `Cannot connect to Redis`

```bash
# Verify Redis is running
docker exec -it ggid-redis redis-cli ping
# → PONG
```

---

## Quick Diagnostic Commands

```bash
# Check all pods
kubectl get pods -n ggid

# Check recent events
kubectl get events -n ggid --sort-by='.lastTimestamp' | tail -20

# Check service logs
kubectl logs -n ggid -l app.kubernetes.io/name=ggid --tail=50

# Port-forward for local testing
kubectl port-forward -n ggid svc/ggid-gateway 8080:8080

# Check Gateway health
curl http://localhost:8080/healthz
```

---

## SAML Federation Issues

### Symptom: SAML response rejected — "invalid issuer"

- **Cause**: Entity ID in GGID metadata doesn't match what the SP expects
- **Fix**: Download GGID metadata at `/api/v1/identity/saml/metadata` and compare `entityID` with SP configuration

### Symptom: "Signature validation failed" on SAML assertion

- **Cause**: SP has an old GGID signing certificate
- **Fix**: Download new certificate from `/api/v1/auth/certificates?type=saml` and upload to SP

### Symptom: SAML attribute mapping not working

- **Cause**: GGID sends attributes with different names than SP expects
- **Fix**: Check `/api/v1/identity/saml/attribute-mapping` and align with SP requirements (e.g., AWS expects `email`, `first_name`, `last_name`, `groups`)

### Symptom: SAML redirect loop between GGID and SP

- **Cause**: NameID format mismatch (SP expects `emailAddress`, GGID sends `unspecified`)
- **Fix**: Configure GGID to use `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`

## Conditional Access (CAE) Issues

### Symptom: Login returns 403 "access_blocked_by_conditional_access_policy"

- **Cause**: A CAP policy matched the login context (IP/geo/risk/device posture)
- **Fix**: Check conditional access policies at `/api/v1/auth/conditional-access/policies`. Disable the matching policy or add the IP to the allowlist

### Symptom: Login returns 200 but `mfa_required: true`

- **Cause**: CAP policy action is `require_mfa` — the token is valid but MFA step-up is needed
- **Fix**: App should redirect to MFA enrollment/verification flow when `mfa_required` is true

## Docker Startup Failures

### Symptom: Container exits immediately with "database connection refused"

- **Cause**: PostgreSQL container isn't ready when GGID tries to connect
- **Fix**: Ensure `depends_on` with health check in docker-compose:
```yaml
depends_on:
  postgres:
    condition: service_healthy
```

### Symptom: "AES_KEY must be 32 bytes hex"

- **Cause**: `AES_KEY` environment variable is not set or wrong format
- **Fix**: Generate with `openssl rand -hex 32` and set in `.env`

### Symptom: Services keep restarting — OOMKilled

- **Cause**: Container memory limit too low for the service
- **Fix**: Increase to minimum 4GB for all-in-one, 2GB per individual service

### Symptom: "port already in use"

- **Cause**: Host ports 8080/3000/5432 are occupied
- **Fix**: Change port mapping in docker-compose or stop conflicting process: `lsof -i :8080`

## Console UI Issues

### Symptom: Console shows white screen (blank page)

- **Cause**: JavaScript bundle failed to load — often a CORS or build issue
- **Fix**:
  1. Open browser DevTools (F12) → Console tab for errors
  2. Verify `CONSOLE_URL` matches the actual URL
  3. Check Next.js is running: `docker compose logs console`
  4. Ensure `NEXT_PUBLIC_API_URL` points to the gateway

### Symptom: Console 404 on all pages

- **Cause**: Console service not started or wrong port configured
- **Fix**: Verify port 3000 is mapped and the console container is running:
```bash
docker compose ps
curl -I http://localhost:3000
```

### Symptom: API calls from Console fail with CORS

- **Cause**: Gateway CORS not configured for the Console origin
- **Fix**: Set `CORS_ALLOWED_ORIGINS=https://console.yourcompany.com` in gateway config

### Symptom: Login redirects to wrong URL after authentication

- **Cause**: `GATEWAY_URL` or `CONSOLE_URL` misconfigured
- **Fix**: Ensure both match your actual domain:
```bash
GATEWAY_URL=https://auth.yourcompany.com
CONSOLE_URL=https://console.yourcompany.com
```

---

*See: [Deploy Troubleshooting](../deploy/troubleshooting.md) | [FAQ](../faq.md) | [Docker 5-Minute](../quickstart/docker-5-min.md) | [Self-Hosting Guide](self-hosting.md)*

*Last updated: 2025-07-18*
