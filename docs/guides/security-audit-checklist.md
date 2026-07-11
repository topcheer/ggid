# Security Audit Checklist

This checklist provides a comprehensive security verification guide for GGID deployments, covering STRIDE threat model, OWASP Top 10, JWT security, CSRF, SQL injection prevention, and operational security.

## STRIDE Threat Model

### Spoofing

- [ ] JWT signed with RS256 or ES256 (asymmetric — private key only on auth service)
- [ ] JWT `iss` claim verified on every request
- [ ] JWT `aud` claim verified per service
- [ ] JWT `jti` tracked in Redis for anti-replay
- [ ] JWT secret is non-empty (`log.Fatal` on empty at startup)
- [ ] OAuth `state` parameter validated (Redis-backed, crypto/rand generated)
- [ ] Tenant ID from JWT claim overrides `X-Tenant-ID` header (prevents spoofing)
- [ ] LDAP START_TLS configured for directory connections
- [ ] WebAuthn origin verification on attestation/assertion

**Verification**:
```bash
# Check JWT algorithm
echo $JWT | cut -d. -f1 | base64 -d 2>/dev/null | jq .alg
# Must output: "RS256" or "ES256"

# Verify jti tracking
redis-cli KEYS "jti:*" | wc -l  # Should be > 0
```

### Tampering

- [ ] All database queries use parameterized statements (pgx `$1`)
- [ ] Audit log hash chain implemented and verified
- [ ] Webhook payloads signed with HMAC-SHA256
- [ ] Configuration loaded at startup (no runtime config API)
- [ ] PostgreSQL RLS enabled with `FORCE ROW LEVEL SECURITY`

**Verification**:
```bash
# Verify hash chain
curl -s https://api.ggid.example.com/api/v1/audit/integrity/verify \
  -H "Authorization: Bearer $TOKEN" | jq .valid
# Must output: true

# Check RLS enforcement
psql -c "SELECT relrowsecurity FROM pg_class WHERE relname = 'users'"
# Must output: t (true)
```

### Repudiation

- [ ] Every auth event publishes to NATS → audit log
- [ ] Policy changes audited with actor ID and diff
- [ ] JWT `jti` provides per-request attribution
- [ ] Admin actions logged (role assign, config change, key rotation)
- [ ] Audit events include source IP and user agent

### Information Disclosure

- [ ] PII obfuscation (`pii.Obfuscate`) applied to audit payloads
- [ ] PostgreSQL RLS prevents cross-tenant data access
- [ ] Error responses use standardized error codes (no stack traces)
- [ ] SSRF protection on webhook URLs (private IP blocking)
- [ ] TLS for all external traffic
- [ ] gRPC TLS for inter-service communication
- [ ] No sensitive data in URL parameters

**Verification**:
```bash
# Check SSRF protection
curl -X POST https://api.ggid.example.com/api/v1/webhooks \
  -d '{"url":"http://127.0.0.1:8080"}' \
  -H "Authorization: Bearer $TOKEN"
# Should be rejected (private IP)

# Check error response (no stack trace)
curl -s https://api.ggid.example.com/api/v1/users/invalid-uuid | jq .
# Should return structured error, not stack trace
```

### Denial of Service

- [ ] Rate limiting wired into production handler chain (token bucket)
- [ ] Account lockout after N failed attempts
- [ ] Body size limits enforced
- [ ] DNS rebinding defense (host header validation)
- [ ] Circuit breaker per upstream service
- [ ] PostgreSQL `max_connections` set with headroom

**Verification**:
```bash
# Test rate limiting
for i in $(seq 1 200); do
  curl -s -o /dev/null -w '%{http_code}\n' https://api.ggid.example.com/api/v1/users \
    -H "Authorization: Bearer $TOKEN"
done | sort | uniq -c
# Should show mix of 200s and 429s

# Test body size limit
curl -X POST https://api.ggid.example.com/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -d "$(head -c 2000000 /dev/urandom | base64)"
# Should return 413
```

### Elevation of Privilege

- [ ] `HasScope()` enforces actual scope (not always-true)
- [ ] Admin API guarded by `hasAdminScope()`
- [ ] JWT signed with asymmetric key (forgery impossible without private key)
- [ ] SCIM PATCH operations validated against RBAC
- [ ] AI agent `max_delegation_depth` enforced
- [ ] OAuth introspection endpoint requires authentication (P0 — check status)

**Verification**:
```bash
# Test scope enforcement
USER_TOKEN=$(curl -s .../auth/login -d '{"username":"regular","password":"pass"}' | jq -r .access_token)
curl -s -o /dev/null -w '%{http_code}' https://api.ggid.example.com/api/v1/admin/users \
  -H "Authorization: Bearer $USER_TOKEN"
# Should output: 403
```

## OWASP Top 10 (2021)

### A01: Broken Access Control

- [ ] RBAC + ABAC policy engine enforces authorization
- [ ] Tenant isolation via PostgreSQL RLS (database-enforced)
- [ ] Gateway injects tenant_id from JWT (not from client header)
- [ ] Direct object references prevented (every query includes tenant_id filter)

### A02: Cryptographic Failures

- [ ] Passwords hashed with Argon2id (memory-hard)
- [ ] JWT signed with RS256/ES256 (not HS256)
- [ ] TLS 1.2+ for all connections
- [ ] gRPC mTLS between services (policy, org done; others in progress)
- [ ] No hardcoded secrets in source code
- [ ] Secrets stored in secrets manager / keys.env

### A03: Injection

- [ ] All SQL queries use parameterized pgx (no string concatenation)
- [ ] `SET LOCAL` uses validated UUID input (not `$1`)
- [ ] Input validation on all API endpoints
- [ ] No OS command execution from user input

**Verification**:
```bash
# Search for SQL injection patterns
grep -rn "fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*DELETE" services/ --include="*.go" | grep -v test
# Only acceptable: SET LOCAL with validated UUID
```

### A04: Insecure Design

- [ ] Threat-modeled using STRIDE
- [ ] Defense in depth (RLS + JWT + scope + rate limit)
- [ ] Fail-safe defaults (deny by default)
- [ ] No security through obscurity

### A05: Security Misconfiguration

- [ ] Security headers: HSTS, X-Frame-Options, X-Content-Type-Options, CSP
- [ ] No default passwords (keygen generates on first run)
- [ ] Debug endpoints (`/debug/pprof`) restricted to internal network
- [ ] CORS configured with specific origins (not `*`)

**Verification**:
```bash
# Check security headers
curl -sI https://api.ggid.example.com/ | grep -E \
  'Strict-Transport|X-Frame|X-Content-Type|Content-Security|Referrer-Policy'
```

### A06: Vulnerable Components

- [ ] `go mod tidy` run regularly
- [ ] Dependabot or similar dependency scanning enabled
- [ ] `govulncheck` run in CI
- [ ] No pinned outdated dependencies (use `@latest`)

### A07: Identification and Authentication Failures

- [ ] Rate limiting on login (token bucket)
- [ ] Account lockout after N failures
- [ ] MFA (TOTP + WebAuthn) available
- [ ] Password history enforcement
- [ ] Session timeout (absolute + idle)
- [ ] Refresh token rotation (single-use)

### A08: Software and Data Integrity Failures

- [ ] Audit hash chain for tamper detection
- [ ] Webhook payloads signed (HMAC-SHA256)
- [ ] CI/CD pipeline with signed artifacts
- [ ] No unsigned third-party libraries

### A09: Security Logging and Monitoring Failures

- [ ] All security events logged to audit service
- [ ] NATS JetStream for reliable delivery
- [ ] SIEM forwarder configured (Splunk/Datadog/ES)
- [ ] Alert rules for suspicious activity
- [ ] Log retention policy enforced

### A10: Server-Side Request Forgery (SSRF)

- [ ] Webhook URL validation (private IP blocking)
- [ ] Protocol restriction (http/https only)
- [ ] DNS resolution check (reject if resolves to private IP)
- [ ] No outbound requests from user-controlled URLs

## JWT Security Checklist

- [ ] RS256 or ES256 (never HS256 in production)
- [ ] Access token expiry: 5-15 minutes
- [ ] Refresh token: single-use with rotation
- [ ] `jti` claim present and tracked
- [ ] `iss` claim verified
- [ ] `aud` claim verified
- [ ] `tenant_id` claim present
- [ ] Key rotation: every 90 days
- [ ] Old keys destroyed after grace period
- [ ] JWKS endpoint accessible to all services
- [ ] No sensitive data in JWT payload (it's base64, not encrypted)

## CSRF Protection

- [ ] OAuth `state` parameter generated with `crypto/rand`
- [ ] `state` validated via Redis (not just cookie comparison)
- [ ] SameSite=Strict or SameSite=Lax on session cookies
- [ ] Origin/Referer header validation on state-changing requests
- [ ] No GET requests for state-changing operations

## SQL Injection Prevention

- [ ] All queries use pgx parameterized statements
- [ ] No `fmt.Sprintf` with user input in SQL
- [ ] `SET LOCAL` uses validated UUID (regex check)
- [ ] Input validation before database access
- [ ] Query builders used where possible

## Operational Security

- [ ] TLS certificates monitored for expiry
- [ ] JWT signing keys rotated per schedule
- [ ] Admin accounts require MFA
- [ ] API keys have expiration dates
- [ ] Unused accounts deactivated (access review)
- [ ] Incident response plan documented
- [ ] Penetration testing conducted annually

## Verification Script

```bash
#!/bin/bash
# security-audit.sh — Automated security checks

BASE_URL="${1:-https://api.ggid.example.com}"
TOKEN="${2:?Usage: $0 <url> <token>}"

echo "=== Security Audit: $BASE_URL ==="

echo -n "1. JWT Algorithm: "
echo "$TOKEN" | cut -d. -f1 | base64 -d 2>/dev/null | jq -r .alg

echo -n "2. Rate Limiting: "
for i in $(seq 1 50); do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/v1/users" \
    -H "Authorization: Bearer $TOKEN")
done
echo "$code (expect 429)"

echo -n "3. Security Headers: "
curl -sI "$BASE_URL/" | grep -c "Strict-Transport\|X-Frame\|X-Content-Type"

echo -n "4. Audit Hash Chain: "
curl -s "$BASE_URL/api/v1/audit/integrity/verify" \
  -H "Authorization: Bearer $TOKEN" | jq -r .valid

echo -n "5. SSRF Protection: "
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/api/v1/webhooks" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"url":"http://127.0.0.1"}')
echo "$code (expect 400/403)"

echo "=== Audit Complete ==="
```

## See Also

- [STRIDE Threat Analysis](../research/stride-analysis.md)
- [Security Overview](../architecture/security-overview.md)
- [Key Management Lifecycle](../research/key-management-lifecycle.md)
- [Gateway Configuration](gateway-config.md)
- [Rate Limiting](rate-limiting.md)
