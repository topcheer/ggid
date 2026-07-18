# GGID Penetration Test Baseline

**Version**: v1.0-stable  
**Test Suite**: `scripts/security-test.sh`  
**Scope**: OWASP API Top 10 coverage for core endpoints  

---

## Executive Summary

| Category | Tests | Expected | Status |
|----------|-------|----------|--------|
| SQL Injection | 10 endpoints × 5 payloads | 400/500 rejection | PASS |
| XSS Reflection | 4 payloads | No script reflection | PASS |
| IDOR (Insecure Direct Object Reference) | 2 scenarios | 403/404 for cross-user | PASS |
| JWT Tampering | 3 attacks | 401 rejection | PASS |
| Rate Limit Bypass | 2 endpoints | 429 after threshold | PASS |

**Overall**: 0 critical vulnerabilities. 3 low-severity warnings (rate limit tuning).

---

## 1. SQL Injection (OWASP A03:2021)

### Test Methodology

5 classic SQL injection payloads tested against 10 endpoints:

| Payload | Type |
|---------|------|
| `' OR '1'='1` | Authentication bypass |
| `'; DROP TABLE users; --` | Data destruction |
| `' UNION SELECT * FROM users --` | Data exfiltration |
| `admin'--` | Comment-based bypass |
| `1; EXEC xp_cmdshell('dir') --` | OS command injection |

### Endpoints Tested

1. `GET /api/v1/users`
2. `GET /api/v1/users?filter=name`
3. `GET /api/v1/roles`
4. `GET /api/v1/audit`
5. `GET /api/v1/orgs`
6. `GET /api/v1/policies`
7. `POST /api/v1/auth/login`
8. `GET /api/v1/users/:id`
9. `GET /api/v1/oauth/clients`
10. `GET /api/v1/agents`

### Expected Behavior

All SQL injection payloads should result in:
- HTTP 400 (Bad Request) — if input validation catches the pattern
- HTTP 500 (Internal Server Error) — if the parameterized query fails safely
- **Never** HTTP 200 with data returned

### Result: PASS

GGID uses parameterized queries (pgx) throughout all PostgreSQL interactions. Input is never directly concatenated into SQL strings. The `pgxpool` driver enforces prepared statements.

### Evidence

```go
// repository/pg_repo.go — parameterized query
row := r.pool.QueryRow(ctx,
    `SELECT id, tenant_id, client_id FROM oauth_clients WHERE tenant_id = $1 AND client_id = $2`,
    tenantID, clientID)  // parameters are escaped by pgx
```

---

## 2. XSS — Cross-Site Scripting (OWASP A03:2021)

### Test Methodology

4 XSS payloads tested for reflection in responses:

| Payload | Vector |
|---------|--------|
| `<script>alert(1)</script>` | Classic script injection |
| `"><img src=x onerror=alert(1)>` | IMG tag injection |
| `javascript:alert(1)` | Protocol handler |
| `<svg/onload=alert(1)>` | SVG event handler |

### Tested Inputs

- `POST /api/v1/auth/register` — `name` field
- `POST /api/v1/auth/login` — `username` field
- `POST /api/v1/users` — `name` field

### Expected Behavior

- No `<script>` tags or event handlers should appear in any API response
- Input should be stored as-is (raw text) but never rendered as HTML by the API
- Console frontend should use React's built-in JSX escaping (automatic)

### Result: PASS

API responses use `Content-Type: application/json`, preventing browser HTML rendering. The console (Next.js/React) auto-escapes all variable content in JSX.

### Security Headers (Defense in Depth)

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
```

---

## 3. IDOR — Insecure Direct Object Reference (OWASP A01:2021)

### Test Methodology

1. Create two users (user1, user2) in the same tenant
2. Login as user1 (non-admin)
3. Attempt to access user2's profile: `GET /api/v1/users/{user2_id}`

### Expected Behavior

- Non-admin user should receive 403 (Forbidden) or 404 (Not Found)
- Admin users can access any user in their tenant
- Cross-tenant access is blocked by tenant isolation

### Result: PASS

**Tenant isolation**: All queries include `tenant_id` in the WHERE clause. The gateway injects `X-Tenant-ID` from the JWT claims, preventing cross-tenant data access.

**RBAC enforcement**: The policy service evaluates authorization before returning user data. Non-admin roles (viewer, user) cannot access other users' profiles.

---

## 4. JWT Tampering (OWASP A02:2021)

### Test 4a: Signature Tampering

| Step | Action |
|------|--------|
| 1 | Decode JWT payload (base64) |
| 2 | Modify `role` → `superadmin`, `is_admin` → `true` |
| 3 | Re-encode payload, keep original signature |
| 4 | Send modified token to `GET /api/v1/users` |

**Expected**: 401 Unauthorized (RSA signature mismatch)  
**Result**: PASS

### Test 4b: None Algorithm Attack

| Step | Action |
|------|--------|
| 1 | Create token with `alg: none` header |
| 2 | Set `sub: admin, role: superadmin` |
| 3 | Send to protected endpoint |

**Expected**: 401 Unauthorized (Go JWT library rejects `alg: none`)  
**Result**: PASS

### Test 4c: Expired Token

| Step | Action |
|------|--------|
| 1 | Modify `exp` claim to 1 hour in the past |
| 2 | Send expired token |

**Expected**: 401 Unauthorized (token expired)  
**Result**: PASS

---

## 5. Rate Limit Bypass (OWASP A04:2019 API Abuse)

### Test 5a: Login Brute-Force

| Metric | Value |
|--------|-------|
| Endpoint | `POST /api/v1/auth/login` |
| Rapid attempts | 30 (wrong password) |
| Expected limit | 5/min → 429 |
| Result | 429 after ~5-10 attempts |

### Test 5b: OAuth Token Brute-Force

| Metric | Value |
|--------|-------|
| Endpoint | `POST /oauth/token` |
| Rapid attempts | 20 (fake credentials) |
| Expected limit | 10/min → 429 |
| Result | 429 after ~10 attempts |

### Rate Limiting Architecture

| Layer | Scope | Mechanism |
|-------|-------|-----------|
| Gateway fixed-window | Per-IP, per-endpoint | `RateLimiter` (in-memory) |
| Tenant token bucket | Per-tenant + IP | `TenantBucketLimiter` |
| 5-Dimensional | tenant/user/key/IP/endpoint | `MultiDimRateLimiter` |

---

## 6. Additional Security Controls Verified

### 6.1 CORS Validation

- Disallowed origins (`https://evil.com`) receive no `Access-Control-Allow-Origin` header
- Wildcard `*` only used without credentials
- Per-tenant CORS configuration supported

### 6.2 Tenant Header Enforcement

- Missing `X-Tenant-ID` header → 400 Bad Request
- Invalid tenant UUID → 400 Bad Request (no longer leaks header name)
- Cross-tenant JWT → 403 Forbidden

### 6.3 Password Security

- Argon2id hashing (19MB, 2 iterations, OWASP compliant)
- Password pepper support (`AUTH_PASSWORD_PEPPER`)
- Password history enforcement
- Account lockout after failed attempts

### 6.4 Client Secret Security

- Argon2id hashed storage (no plaintext)
- Secret shown only on creation (RFC 7591)
- Secret rotation with old-key verification
- `omitempty` JSON tag prevents accidental exposure

---

## Running the Tests

```bash
# Ensure services are running
curl -s http://localhost:8080/healthz | jq .

# Run security test suite
bash scripts/security-test.sh

# Against deployed environment
bash scripts/security-test.sh https://ggid.iot2.win
```

### Interpreting Results

- **PASS**: Security control working as expected
- **WARN**: Potential weakness — investigate but not critical
- **FAIL**: Critical security issue — fix immediately

---

## Recommendations

1. **Add CSP reporting**: Enable `Content-Security-Policy-Report-Only` to detect attempted XSS
2. **Regular re-testing**: Run `security-test.sh` in CI weekly
3. **External pentest**: Commission a professional penetration test before each major release
4. **Rate limit tuning**: Monitor production traffic patterns and adjust limits
5. **Secret scanning**: Add TruffleHog or GitGuardian to CI for API key/secret detection

---

## References

- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [OAuth 2.0 Security Best Current Practice](https://tools.ietf.org/html/draft-ietf-oauth-security-topics)
