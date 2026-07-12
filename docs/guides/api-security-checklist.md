# API Security Checklist

This guide maps the OWASP API Security Top 10 to GGID's mitigations, test plans, CI/CD security gates, runtime protections, and audit requirements.

## OWASP API Security Top 10 (2023)

### API1: Broken Object Level Authorization (BOLA)

**Risk**: Users access objects belonging to other users/tenants.

**GGID Mitigations**:
- Row-Level Security (RLS) on all database tables, scoped by `tenant_id`
- JWT `sub` claim verified against requested resource owner
- Tenant ID from JWT claim takes priority over `X-Tenant-ID` header (prevents spoofing)
- All queries include `WHERE tenant_id = ?` automatically via middleware

**Test Plan**:
```bash
# Test: User A cannot access User B's data
TOKEN_A=$(login user_a)
TOKEN_B=$(login user_b)

# User A tries to read User B's profile
curl -H "Authorization: Bearer $TOKEN_A" /api/v1/users/$USER_B_ID
# Expected: 404 Not Found

# Cross-tenant access
curl -H "Authorization: Bearer $TOKEN_A" -H "X-Tenant-ID: tenantB" /api/v1/users
# Expected: Only tenant A users returned (JWT claim wins)
```

**CI/CD Gate**: Automated BOLA test suite runs on every PR.

---

### API2: Broken Authentication

**Risk**: Weak authentication, token theft, credential stuffing.

**GGID Mitigations**:
- JWT signature verification (RS256/ES256/EdDSA)
- Refresh token rotation with reuse detection
- Rate limiting on login (5/min per user, 50/min per IP)
- Password breach check (HIBP API)
- Password pepper (server-side secret)
- MFA required for admin accounts
- DPoP binding for high-security clients

**Test Plan**:
```bash
# Test: Rate limiting on login
for i in $(seq 1 10); do
  curl -X POST /api/v1/auth/login -d '{"username":"test","password":"wrong"}'
done
# Expected: 429 after 5 attempts

# Test: Expired token rejected
EXPIRED_TOKEN="eyJ...expired..."
curl -H "Authorization: Bearer $EXPIRED_TOKEN" /api/v1/users
# Expected: 401 Unauthorized

# Test: Refresh token reuse detection
# Use same refresh token twice → second use should revoke family
```

**CI/CD Gate**: Auth test suite with rate limit, token expiry, reuse detection tests.

---

### API3: Broken Object Property Level Authorization

**Risk**: Users can read/write properties they shouldn't (mass assignment).

**GGID Mitigations**:
- Explicit field whitelisting on all input handlers (no auto-binding)
- Output filtering based on caller permissions
- PII fields (email, phone) only returned with explicit scope
- Write operations validate field-level permissions

**Test Plan**:
```bash
# Test: Mass assignment prevention
curl -X PUT /api/v1/users/me -d '{"name":"Test","is_admin":true}'
# Expected: is_admin field ignored or 400 Bad Request

# Test: PII field suppression without scope
curl -H "Authorization: Bearer $TOKEN_NO_EMAIL_SCOPE" /api/v1/users/$ID
# Expected: email field absent from response
```

**CI/CD Gate**: Static analysis for unbounded struct binding.

---

### API4: Unrestricted Resource Consumption

**Risk**: DoS via large payloads, unbounded queries, no rate limits.

**GGID Mitigations**:
- Rate limiting (token bucket, per-user/IP/tenant)
- Request body size limit (1MB default)
- Pagination on all list endpoints (max 100 per page)
- Query timeout (30s database, 10s API)
- Connection pooling with max connections
- Redis-backed distributed rate limiting

**Test Plan**:
```bash
# Test: Large payload rejected
curl -X POST /api/v1/users -d "$(python3 -c 'print("x"*10000000)')"
# Expected: 413 Request Entity Too Large

# Test: Pagination enforced
curl /api/v1/users?per_page=10000
# Expected: 400 Bad Request or capped at 100

# Test: Rate limiting
for i in $(seq 1 200); do curl /api/v1/users; done
# Expected: 429 after limit exceeded
```

**CI/CD Gate**: Load test with 1000 req/s, verify no resource exhaustion.

---

### API5: Broken Function Level Authorization

**Risk**: Users access admin functions without proper authorization.

**GGID Mitigations**:
- Role-based access control (RBAC) on every endpoint
- `hasAdminScope()` guard on `/api/v1/admin/*` routes
- Permission checks before every write operation
- Default-deny: endpoints require explicit permission

**Test Plan**:
```bash
# Test: Regular user cannot access admin endpoints
curl -H "Authorization: Bearer $REGULAR_TOKEN" /api/v1/admin/tenants
# Expected: 403 Forbidden

# Test: User without permission cannot create roles
curl -X POST -H "Authorization: Bearer $USER_TOKEN" /api/v1/roles
# Expected: 403 Forbidden
```

**CI/CD Gate**: Authorization matrix test (every role × every endpoint).

---

### API6: Unrestricted Access to Sensitive Business Flows

**Risk**: Automation of sensitive flows (account creation, password reset, MFA enrollment).

**GGID Mitigations**:
- Rate limiting on registration (10/hour per IP)
- Rate limiting on password reset (3/hour per user)
- Rate limiting on MFA recovery (3/hour per user)
- CAPTCHA on registration (configurable)
- Email verification required for new accounts
- Admin approval for bulk user creation

**Test Plan**:
```bash
# Test: Registration rate limit
for i in $(seq 1 20); do
  curl -X POST /api/v1/auth/register -d "{\"username\":\"test$i\",\"email\":\"test$i@x.com\"}"
done
# Expected: 429 after 10 registrations
```

**CI/CD Gate**: Rate limit regression tests on all sensitive endpoints.

---

### API7: Server Side Request Forgery (SSRF)

**Risk**: Server makes requests to attacker-controlled URLs.

**GGID Mitigations**:
- Webhook URL validation (block private IP ranges, localhost, metadata endpoints)
- Allowlist for outbound HTTP requests
- DNS rebinding protection (validate resolved IP before request)
- No raw user input in outbound URLs

```go
var blockedCIDRs = []string{
    "127.0.0.0/8",      // Loopback
    "10.0.0.0/8",       // Private
    "172.16.0.0/12",    // Private
    "192.168.0.0/16",   // Private
    "169.254.0.0/16",   // Link-local (AWS metadata)
    "0.0.0.0/8",        // Reserved
    "::1/128",          // IPv6 loopback
    "fc00::/7",         // IPv6 unique local
}
```

**Test Plan**:
```bash
# Test: Webhook to internal IP blocked
curl -X POST /api/v1/webhooks -d '{"url":"http://169.254.169.254/latest/meta-data/"}'
# Expected: 400 Bad Request

# Test: DNS rebinding protection
curl -X POST /api/v1/webhooks -d '{"url":"http://evil.com"}'
# Where evil.com resolves to 127.0.0.1
# Expected: 400 Bad Request
```

**CI/CD Gate**: SSRF test suite with known bypass attempts.

---

### API8: Security Misconfiguration

**Risk**: Default credentials, unnecessary features, verbose errors.

**GGID Mitigations**:
- No default credentials (JWT secret must be set, `log.Fatal` if empty)
- Security headers on all responses (HSTS, X-Content-Type-Options, X-Frame-Options, CSP)
- TLS 1.2+ only (TLS 1.0/1.1 disabled)
- gRPC TLS between all services
- CORS configured with explicit allowed origins (no wildcard in production)
- Error messages don't leak internal details
- Debug endpoints disabled in production

**Security Headers**:
```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

**Test Plan**:
```bash
# Test: Security headers present
curl -I https://auth.ggid.example.com/api/v1/users
# Verify HSTS, X-Content-Type-Options, X-Frame-Options headers

# Test: TLS version
openssl s_client -connect auth.ggid.example.com:443 -tls1_1
# Expected: Connection refused (TLS 1.1 disabled)

# Test: CORS not wildcard
curl -H "Origin: https://evil.com" -I /api/v1/users
# Expected: No Access-Control-Allow-Origin for evil.com
```

**CI/CD Gate**: Security header validation in integration tests.

---

### API9: Improper Inventory Management

**Risk**: Stale API versions, undocumented endpoints, shadow APIs.

**GGID Mitigations**:
- OpenAPI 3.0 specification maintained and version-controlled
- All endpoints documented and tested
- API versioning (`/api/v1/`, `/api/v2/`)
- Deprecated endpoints have sunset timeline
- Automated inventory check in CI/CD

**Test Plan**:
```bash
# Test: All routes are in OpenAPI spec
diff <(ggid routes --list) <(openapi routes --extract spec.yaml)
# Expected: No differences

# Test: No undocumented endpoints
# Scan with tool like arjun or param-miner
```

**CI/CD Gate**: OpenAPI spec validation, route coverage check.

---

### API10: Unsafe Consumption of APIs

**Risk**: Trusting third-party API responses without validation.

**GGID Mitigations**:
- IdP metadata validation (certificate chain, signature)
- SCIM source validation (schema compliance check)
- Webhook response validation (status code, content type)
- Input validation on all external API responses
- Timeout on all outbound requests (10s default)

**Test Plan**:
```bash
# Test: Invalid IdP metadata rejected
curl -X POST /api/v1/admin/saml/metadata -d '<invalid-xml/>'
# Expected: 400 Bad Request

# Test: Webhook timeout
# Configure webhook with 60s response time
# Expected: GGID times out at 10s, marks as failed
```

**CI/CD Gate**: External API interaction tests with mock servers.

## CI/CD Security Gates

### Pipeline Stages

```
1. Static Analysis (go vet, gosec, staticcheck)
2. Dependency Scan (govulncheck, nancy)
3. Unit Tests (including security tests)
4. Integration Tests (BOLA, auth, rate limit, SSRF)
5. DAST Scan (OWASP ZAP baseline)
6. Container Scan (trivy)
7. Secret Scan (gitleaks)
8. OpenAPI Validation
```

### Configuration

```yaml
# .github/workflows/security.yml
security-gates:
  - name: gosec
    command: "gosec ./..."
    fail_on: "high"
  - name: govulncheck
    command: "govulncheck ./..."
    fail_on: "high"
  - name: gitleaks
    command: "gitleaks detect --source ."
    fail_on: "any"
  - name: trivy
    command: "trivy image ggid:latest"
    fail_on: "critical"
  - name: zap-baseline
    command: "zap-baseline.py -t https://staging.ggid.example.com"
    fail_on: "high"
```

## Runtime Protection

### Web Application Firewall (WAF) Rules

| Rule | Protection |
|---|---|
| SQL injection | Pattern matching + parameterized queries |
| XSS | Input sanitization + CSP headers |
| Path traversal | File path validation |
| Command injection | No shell exec in request path |
| Bot detection | Behavioral analysis + rate limiting |

### Real-time Monitoring

| Metric | Alert Threshold |
|---|---|
| Login failure rate | >20% over 5 minutes |
| 429 rate limit hits | >1000/hour |
| 5xx error rate | >5% |
| Auth token reuse | Any occurrence |
| SSRF attempt | Any occurrence |
| Admin endpoint access by non-admin | Any occurrence |

## Audit Requirements

### What to Audit

| Event | Fields Logged | Retention |
|---|---|---|
| Login success/failure | user_id, ip, user_agent, method, timestamp | 1 year |
| Token issuance | user_id, client_id, scopes, ip | 1 year |
| Token revocation | user_id, jti, reason, admin_id | 1 year |
| MFA enrollment/reset | user_id, method, ip | 2 years |
| Role assignment change | user_id, role, admin_id, ip | 2 years |
| Tenant config change | tenant_id, setting, old_value, new_value, admin_id | 2 years |
| API access (admin) | admin_id, endpoint, method, ip, response_code | 1 year |
| Rate limit violation | ip, user_id, endpoint, count | 90 days |
| Security event | type, severity, source, details | 3 years |

### Audit Log Integrity

- Hash chain: Each audit event includes hash of previous event
- Append-only: Audit logs cannot be modified or deleted
- Replication: Audit logs replicated to separate storage
- Export: SIEM forwarder for external analysis

## Summary Checklist

| Category | Items |
|---|---|
| BOLA | RLS enabled, JWT tenant validation, cross-tenant tests |
| Auth | JWT verification, refresh rotation, rate limits, MFA, DPoP |
| Property Auth | Field whitelisting, PII scope filtering, mass assignment prevention |
| Resource Limits | Rate limiting, body size, pagination, timeouts |
| Function Auth | RBAC on every endpoint, admin scope guard |
| Sensitive Flows | Rate limits on registration/reset/recovery |
| SSRF | URL validation, IP blocklist, DNS rebinding protection |
| Misconfiguration | Security headers, TLS 1.2+, no default creds, CORS |
| Inventory | OpenAPI spec, route coverage, versioning |
| API Consumption | Input validation on external responses, timeouts |
| CI/CD | SAST, dependency scan, DAST, container scan, secret scan |
| Runtime | WAF, real-time monitoring, alerting |
| Audit | Comprehensive logging, hash chain, SIEM export |