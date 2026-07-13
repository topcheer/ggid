# GGID SDK + Demo E2E Test Report

## Test Cycle: 2026-07-14 01:22 CST

### Pod Health
All 18 pods Running, 0 restarts.

### Step 3: 4-Language ERP Backend Tests

| Backend | Health | Products | Create | Customers | Dashboard | No-Auth | Viewer-POST |
|---------|--------|----------|--------|-----------|-----------|---------|-------------|
| Node.js | ok | 5 items | 403 (policy) | 3 items | OK | blocked | 403 |
| Go | ok | 5 items | 403 (policy) | 3 items | OK | blocked | 403 |
| Java | ok | 5 items | 403 (role) | 3 items | OK | blocked | 403 |
| Python | ok | 5 items | 403 (policy) | 3 items | OK | blocked | 403 |

**Note**: Product create returns 403 for admin because GGID policy engine denies `products:create`. This is correct RBAC behavior — the deny-all policy was deleted, but no explicit allow rule for `products:create` exists.

### Step 4: OAuth/OIDC Tests

| Test | Result | Notes |
|------|--------|-------|
| A. Authorization Code Flow | PASS | Returns 200 (login page) |
| B. Device Code (RFC 8628) | PASS | Returns device_code, user_code, verification_uri |
| C. DCR (RFC 7591) | FAIL | `missing tenant context` — assigned to backend |
| D1. OIDC Discovery | PASS | issuer now `https://ggid.iot2.win` (fixed) |
| D2. JWKS | PASS | 1 key present |
| D3. UserInfo | PASS | sub=ecb72e20-bef0-4aaf-a183-ce204f647ebe |
| E1. Refresh Token | FAIL | `invalid refresh token` — token pair mismatch (test harness issue) |
| E2. Revoke | PASS | HTTP 200 |
| E3. Introspect | PASS | active=true |

### Step 5: SDK Tests

| SDK | Tests | Result |
|-----|-------|--------|
| Go | cached | PASS |
| Rust | 11 | PASS |
| Ruby | 22 | PASS |
| Java | 16 | PASS |
| Python | 16 | PASS |
| Node.js | tsc exit 0 | PASS |
| C# | exit 0 | PASS |
| Dart | 25 | PASS |

### Bugs Found & Status

| # | Bug | Owner | Status |
|---|-----|-------|--------|
| 1 | DCR endpoint `missing tenant context` | backend | Assigned |
| 2 | Java ERP @RequireRole fails (no roles claim in JWT) | frontend | Assigned |
| 3 | OIDC Discovery issuer = internal IP | arch | **FIXED** (set OAUTH_ISSUER=https://ggid.iot2.win) |
| 4 | Refresh token `invalid_grant` | arch | Test harness issue (used different login sessions) |

### Team Task Assignment

| Member | Task | Status |
|--------|------|--------|
| ggcxf_backend | Fix DCR tenant context bug in services/oauth/ | Pending |
| ggcxf_frontend | Fix Java ERP SecurityAspect role extraction | Pending |
| ggcxf_docs | (idle, no tasks this cycle) | - |
