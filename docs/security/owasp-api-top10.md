# OWASP API Security Top 10 Assessment

**Date**: 2025-07-18  
**Version**: v1.0-stable  

---

## API1: Broken Object Level Authorization (BOLA)

**Score: 8/10 ✅**

- All user-scoped endpoints require `X-Tenant-ID` header
- PostgreSQL Row-Level Security (RLS) enforces tenant isolation at DB level
- Object-level auth: `/users/{id}` verifies caller tenant matches object tenant
- Policy PDP checks resource ownership before allowing access

**Gaps**: 
- Verify RLS policies are tested with cross-tenant queries
- Some endpoints (e.g., `/groups/{id}/members`) may not check tenant ownership of the group

## API2: Broken Authentication

**Score: 9/10 ✅**

- JWT with RS256 (asymmetric, JWKS-compatible)
- Refresh token rotation (single-use with replay detection)
- MFA: TOTP + WebAuthn + backup codes
- Brute-force protection: per-user + per-IP rate limiting
- Password spray detection: 15 unique users/10min threshold
- Password strength: zxcvbn ≥2 required
- Session binding: DPoP proof-of-possession
- Break-glass: reason required, full audit trail

**Gaps**:
- JWT TTL should be verified ≤15min in production config

## API3: Excessive Data Exposure

**Score: 7/10 ⚠️**

- `/oauth/userinfo` returns profile + roles + groups + permissions (intentional)
- `/oauth/introspect` returns full token context (intentional for downstream)
- User list returns full user objects (not filtered by caller's need-to-know)
- Audit events may expose IP addresses and user agents

**Gaps**:
- Consider field-level filtering on user list API
- Audit event export should redact sensitive PII for non-admin callers

## API4: Lack of Resources & Rate Limiting

**Score: 8/10 ✅**

| Endpoint | Rate Limited |
|----------|-------------|
| Login | Per-user + per-IP |
| Register | Per-IP |
| Password reset | Per-email |
| Token refresh | Per-token |
| OAuth authorize | Per-client |
| Global API | Per-tenant bucket |
| GraphQL | Per-user (100/min) |

**Gaps**:
- OAuth `/token` endpoint rate-limited only via gateway middleware
- `DELETE /login-attempts/:username` has no rate limit (admin-only)

## API5: Broken Function Level Authorization (BFLA)

**Score: 7/10 ⚠️**

- RBAC enforced at gateway: JWT claims → role → allowed endpoints
- Admin endpoints require `admin` role (verified at gateway)
- Policy PDP checks action-level permissions
- Some endpoints rely on header (`X-Is-Admin`) rather than JWT claim verification

**Gaps**:
- Verify ALL admin endpoints check role claim (not just header)
- Some new endpoints (TAP, CAP, CCM) may not have explicit role checks
- Consider adding function-level authz middleware to all `/admin/*` paths

## API6: Mass Assignment

**Score: 8/10 ✅**

- User update endpoint uses explicit field mapping (not blind struct binding)
- JSON decoder targets specific request structs with limited fields
- No `json.Unmarshal(req)` into domain model directly

## API7: Security Misconfiguration

**Score: 6/10 ⚠️**

- CORS configurable but must verify no wildcard in production
- `/api/v1/system/bootstrap` exposed after setup
- Debug endpoints may be accessible in production if not properly configured
- Error messages sometimes expose internal structure names

## API8: Improper Assets Management

**Score: 7/10 ⚠️**

- OpenAPI spec covers 704/864 endpoints (81.5%)
- Swagger UI accessible at `/docs`
- Some newer endpoints (CCM, NHI risk) may not be in spec yet

## API9: Insufficient Logging & Monitoring

**Score: 9/10 ✅**

- Comprehensive audit events for all sensitive operations
- Hash chain integrity verification
- CCM (Continuous Compliance Monitoring) with 15 controls
- CAE (Continuous Access Evaluation) session re-evaluation
- Prometheus metrics with ggid_ prefix
- SIEM export (JSON/CSV)

## API10: SSRF

**Score: 9/10 ✅**

- No outbound URL fetching from user input
- Webhook URLs are validated but outbound calls are server-initiated
- LDAP/SMTP connectors use admin-configured URLs (not user-supplied)

---

## Overall Score: **7.8/10 (B+)**

| Category | Score | Priority |
|----------|-------|----------|
| API1 BOLA | 8/10 | Low |
| API2 Broken Auth | 9/10 | Low |
| API3 Excessive Data | 7/10 | Medium |
| API4 Rate Limiting | 8/10 | Low |
| API5 Broken Func Authz | 7/10 | Medium |
| API6 Mass Assignment | 8/10 | Low |
| API7 Misconfiguration | 6/10 | High |
| API8 Asset Mgmt | 7/10 | Medium |
| API9 Logging/Monitoring | 9/10 | Low |
| API10 SSRF | 9/10 | Low |

### Top 3 priorities for v1.0-stable:
1. **API7**: Disable bootstrap endpoint, verify CORS, fix error messages
2. **API5**: Audit all admin endpoints for proper role-based authz
3. **API3**: Add field-level filtering on sensitive list endpoints
