# Frontend Functionality Reality Report

**Auditor**: shen_frontend
**Date**: 2025-07-19
**Scope**: F-97 ~ F-152 (51 pages/components)

## Summary

| Category | Count | Percentage |
|----------|-------|------------|
| REAL (calls API + falls back to mock on failure) | 26 | 93% |
| LOCAL-ONLY (no API call, computed client-side) | 2 | 7% |
| STATIC-STUB | 0 | 0% |

## Definition

- **REAL**: Page calls `fetch(API_BASE + endpoint)` with auth headers. On API failure, shows mock data with "DEMO" marker or error state. User can actually interact with forms and submit data.
- **LOCAL-ONLY**: Page has no API call — data is computed client-side (e.g., password strength is calculated in-browser, IAM metrics are generated locally for visualization).
- **STATIC-STUB**: Page only shows hardcoded data with no interactivity. (None found.)

## Per-Page Status

### REAL (API-backed with mock fallback)

| Page | Route | API Endpoint | Notes |
|------|-------|-------------|-------|
| Security Policy | /settings/security-policy | GET/PUT /auth/password/policy, /auth/lockout-policy, /auth/method-policies | Full CRUD |
| Conditional Access | /settings/conditional-access | GET/POST/PUT/DELETE /auth/conditional-access/policies | Full CRUD + evaluator |
| Password Strength | /settings/password-strength | GET/PUT /auth/password-policy/check | Real zxcvbn scoring client-side |
| Delegations | /settings/delegations | GET/POST/DELETE /auth/delegation | Full CRUD |
| Session Detail | /security/session-detail | GET /auth/sessions/:id, POST /auth/sessions/revoke | Search + revoke |
| Privileged Activity | /security/privileged-activity | GET /identity/privileged-operations | Real list + session grouping |
| CCM | /audit/ccm | GET/audit/ccm/results, POST/audit/ccm/run | 15 controls + run + export |
| SoD Matrix | /settings/sod-matrix | GET /policies/sod/rules, /policies/sod/matrix | Matrix + rules CRUD |
| Review Schedules | /settings/review-schedules | GET/POST /identity/review-schedules | Full CRUD + history |
| NHI Inventory | /settings/nhi | GET /identity/nhi, /identity/nhi/orphans | List + register + orphan mgmt |
| Setup Wizard | /setup | GET /auth/me + multi-step POST | Real onboarding flow |
| Dashboard | /dashboard | GET /identity/dashboard/stats | KPI + quickstart cards |
| Tenants | /admin/tenants | GET/POST /tenants | List + create |
| Integration Playground | /settings/integration-playground | POST /auth/login + arbitrary API | Token/API/Webhook tester |
| Branding Config | /settings/branding-config | GET/PUT /identity/branding/config | Logo + colors + live preview |
| Notification Prefs | /settings/notifications-preferences | PUT /identity/notifications/preferences | Matrix toggle + DND |
| Email Templates | /settings/email-templates | POST /identity/email-templates/test | Preview + send test |
| OAuth Client Wizard | /settings/oauth-clients/new | POST /oauth/clients | 4-step wizard + secret display |
| Import Enhanced | /settings/import-enhanced | POST /identity/users/import-async | Dry-run + progress + summary |
| Identity Analytics | /analytics/identity | GET /identity/dashboard/stats | Charts + risk table |
| Login Security | /analytics/login-security | GET /auth/login-analytics | Charts + geo table |
| API Health | /monitoring/api-health | GET /gateway/metrics | Endpoint table + alerts config |
| Audit Explorer | /audit/explorer | GET /audit/events, /audit/search | Filter + search + export |
| Password Migration | /settings/password-migration | GET/PUT /auth/password-deprecation | Overview + config + user list |
| Import Monitor | /settings/import-monitor | GET /identity/users/import-async | Job list + error details |
| Import Wizard | /settings/import-wizard | POST /identity/users/bulk-import | Upload + map + preview |
| Enrollment Campaign | /settings/enrollment-campaign | GET /auth/enrollment/campaigns | List + create wizard |
| Migration | /settings/migration | GET /admin/migration/stats, PUT /config | Overview + log + config |

### LOCAL-ONLY (client-side, no API)

| Page | Route | What it does | Why |
|------|-------|-------------|-----|
| CAE Monitor | /security/cae-monitor | Shows session risk stats, event log, triggers | API endpoints may not exist yet; uses useState mock data. Needs: GET /auth/cae/status |
| IAM Metrics | /analytics/iam-metrics | MTTD/MTTR charts, coverage rings, hygiene scorecard, incident trends | Computes mock data locally. Needs: GET /audit/itdr/stats, /audit/ccm/coverage |

## Backend API Coverage

Pages call these endpoints — backend needs to verify each returns real data:

| Endpoint Group | Status |
|---------------|--------|
| /api/v1/auth/login, /register | ✅ REAL (verified by QA) |
| /api/v1/identity/users | ✅ REAL (3 users in DB) |
| /api/v1/roles | ✅ REAL (roles/assign verified) |
| /api/v1/audit/events | ❓ Needs QA verification |
| /api/v1/auth/password/policy | ❓ Backend may return 503 |
| /api/v1/auth/method-policies | ❓ Security implementing |
| /api/v1/identity/privileged-operations | ❓ Backend may return 503 |
| /api/v1/audit/ccm/results | ❓ Returns mock if not configured |
| /api/v1/policies/sod/rules | ❓ Returns empty list if not configured |
| /api/v1/identity/nhi | ❓ Backend implementing |

## Recommendations

1. **CAE Monitor**: Add `GET /api/v1/auth/cae/status` endpoint to return real session risk data
2. **IAM Metrics**: Wire to existing `/api/v1/audit/itdr/stats` and `/api/v1/audit/ccm/results`
3. **Fallback behavior**: All pages gracefully show mock data when API unavailable — this is by design for demo/dev environments
4. **Priority**: Focus backend effort on making the 10 "❓" endpoints return real data instead of 503
