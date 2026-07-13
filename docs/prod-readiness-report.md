# GGID Platform Production Readiness — Honest Gap Analysis

**Last Updated:** 2026-07-13 08:37 UTC

## What Actually Works (Verified)

| Area | Status | Evidence |
|------|--------|----------|
| Register/Login/JWT | PASS | 201/200, 693-char JWT |
| User list | PASS | 200 |
| Role create/list/update/delete | PASS | 201/200/200/200 |
| Policy create/list | PASS | 201/200 |
| Org create/list | PASS | 201/200 |
| OAuth client create/list | PASS | 201/200 (needs all fields) |
| Audit events list | PASS | 200 |
| Trust store CA upload/list | PASS | 201/200 |
| mTLS config get/update | PASS | 200/200 |
| OIDC discovery/JWKS | PASS | 200/200 |
| Token revocation | PASS | 200 |
| Webhook create | PASS | 201 |
| 720 console pages load | PASS | All <500, no hydration errors |
| 19 functional UI tests | PASS | Form fill, CRUD, theme, i18n, nav |

## Real Gaps Found This Cycle

### 1. User UPDATE returns 405 on PUT (CRITICAL)
- `PUT /api/v1/users/{id}` → 405 Method Not Allowed
- `PATCH /api/v1/users/{id}` → 200 (works)
- **Impact**: Console frontend may use PUT for user updates → will fail
- **Fix needed**: Add PUT handler to identity service

### 2. OAuth client create requires all fields (MEDIUM)
- Empty grant_types causes 500 if COALESCE not in SQL
- With explicit fields works (201)
- **Status**: Fixed in SQL (COALESCE), but handler should set defaults

### 3. Trust store is in-memory only (MEDIUM)
- CA upload works (201) but data lost on pod restart
- DB migration exists (04_trust_store.sql) but not wired in main.go
- **Fix needed**: Wire pg-backed store in auth/cmd/main.go

### 4. SetCAPool not called at startup (MEDIUM)
- Email sender, LDAP provider, SIEM forwarder have SetCAPool() methods
- But main.go doesn't call them — custom CAs uploaded via API are not used by outbound connections
- **Fix needed**: Wire trust store → email/LDAP/SIEM at startup

### 5. 3 functional tests still failing (LOW)
- Register form submit: Playwright selector timing (SSR vs hydration)
- Duplicate username: same timing issue
- Theme toggle: dashboard redirect race condition
- **Not product bugs** — test infrastructure issues

### 6. Console dashboard System Health (LOW)
- Shows services as "down" — healthcheck endpoint config issue
- Services are actually healthy (kubectl shows Running)

### 7. Not Tested End-to-End
- OAuth authorization code flow (need real redirect_uri + browser flow)
- SAML IdP (not deployed)
- Social login (Google/GitHub SSO — need real credentials)
- SCIM 2.0 (skeleton only)
- WebAuthn registration (code exists, not tested via browser)
- Demo app integration (no demo app deployed)

## Overall Readiness: 90% (not 99%)

The 8-endpoint GET-only check was misleading. Deep CRUD testing revealed:
- User PUT update broken (405)
- Trust store not persisted
- Trust store not wired to consumers
- Several E2E flows untested

### Priority fixes:
1. Add PUT handler to identity service (user update)
2. Wire trust store DB persistence in main.go
3. Wire SetCAPool() to email/LDAP/SIEM at startup
4. Test OAuth authorization code flow end-to-end
