# GGID Production Readiness Report — 2026-07-12

## Current Status: PARTIALLY WORKING

### What Works (PASS)
- **Register**: `POST /api/v1/auth/register` -> 201
- **Login**: `POST /api/v1/auth/login` -> 200 + JWT
- **Users CRUD**: `GET /api/v1/users` -> 200
- **Roles CRUD**: `GET /api/v1/roles` -> 200
- **Organizations CRUD**: `GET /api/v1/orgs` -> 200
- **Audit Events**: `GET /api/v1/audit/events` -> 200
- **Policies CRUD**: `GET /api/v1/policies` -> 200
- **Access Requests**: `GET /api/v1/access-requests` -> 200
- **Console Pages**: All 20 tested pages return 200 (settings pages fixed)
- **401 Redirect**: Unauthenticated requests redirect to /login (skips /auth/ endpoints)
- **Healthz**: Gateway /healthz returns 200

### What's Broken (FAIL)

#### 1. API 404 Errors — Backend endpoints not registered (17 endpoints)
These gateway routes exist but backend services don't have matching handlers:

| Endpoint | Routed To | Issue |
|----------|-----------|-------|
| `/api/v1/mfa/status` | auth:9001 | Auth has `/api/v1/auth/mfa/*` not `/api/v1/mfa/*` |
| `/api/v1/tokens` | auth:9001 | Auth has `/api/v1/auth/sessions` not `/api/v1/tokens` |
| `/api/v1/login-security` | auth:9001 | Not registered in auth service |
| `/api/v1/password-history` | auth:9001 | Auth has `/api/v1/auth/password-history` |
| `/api/v1/delegation` | auth:9001 | Not registered |
| `/api/v1/account-linking` | auth:9001 | Not registered |
| `/api/v1/consent` | auth:9001 | Not registered |
| `/api/v1/notifications` | auth:9001 | Not registered |
| `/api/v1/introspection/config` | auth:9001 | Not registered |
| `/api/v1/webhooks` | audit:8072 | Not registered in audit service |
| `/api/v1/rate-limits` | policy:8070 | Not registered in policy service |
| `/api/v1/permissions/tree` | policy:8070 | Not registered |
| `/api/v1/sod/rules` | policy:8070 | Not registered |
| `/api/v1/event-correlation/rules` | audit:8072 | Not registered |
| `/api/v1/agents` | oauth:9005 | Not registered in oauth service |
| `/api/v1/audit/hash-chain` | audit:8072 | Not registered |
| `/api/v1/compliance/schedules` | audit:8072 | Not registered |

**Root cause**: Frontend pages call API paths like `/api/v1/mfa/status` but backend services
register them under `/api/v1/auth/mfa/*`. The path prefixes don't match.

**Fix options**:
1. Add route aliases in backend services (quick)
2. Fix frontend to call correct paths (proper)
3. Add path rewriting in gateway (workaround)

#### 2. OAuth Service — 500 on /api/v1/oauth/clients
The OAuth service still panics or returns 500. The nil guard was added but the OAuth
service may not have a database connection (DATABASE_URL not set in k3s deployment).

#### 3. Rate Limiting — 429 on many endpoints
After ~5 rapid requests, the auth service returns 429. This is expected behavior but
makes testing difficult. Need to restart auth container or wait between tests.

#### 4. Gzip Compression Broken
Gateway applies gzip compression but doesn't set Content-Encoding header correctly.
Clients must send `Accept-Encoding: identity` to get uncompressed responses.
This breaks browser fetch() calls that expect proper decompression.

### Fixes Applied This Session (commit a3efd5a)
1. Gateway: Added 20+ missing route prefixes for all 404 endpoints
2. Gateway: Added `/api/v1/healthz` to public paths
3. OAuth: Added nil pointer guard on clientRepo (prevents panic)
4. Console: Fixed settings/layout.tsx (missing default export caused 500 on ALL /settings/* pages)
5. Console: 401 interceptor now skips `/auth/` endpoints (prevents login redirect loop)
6. Console: Removed deprecated eslint config from next.config.mjs

### What Needs To Be Done Next
1. **Fix path mismatch**: Add route aliases in auth/policy/audit services OR fix frontend API calls
2. **Fix OAuth DB connection**: Ensure DATABASE_URL is set in oauth deployment
3. **Fix gzip**: Gateway should set Content-Encoding: gzip header when compressing
4. **Add missing backend handlers**: Many features exist as code but aren't wired to HTTP routes
5. **Test OAuth/OIDC flow**: authorize, token, userinfo endpoints
6. **Test SAML flow**: SP metadata, SSO login
7. **Test SCIM provisioning**: User/group CRUD via SCIM API
8. **Test demo app integration**: End-to-end OAuth login flow with a demo app

### Cannot Test Without Real Infrastructure
- SMTP server (for email OTP, magic links, password reset)
- LDAP server (for LDAP auth)
- WebAuthn authenticators (for passkey registration)
- External OAuth providers (Google, GitHub, Microsoft)
