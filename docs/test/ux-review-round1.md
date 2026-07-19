# UX Review Round 1 — 2026-07-19

## Methodology
- Browser inspection of all core pages (admin login)
- Source code review for data flow, error handling, loading states
- UX Golden Rules: high-frequency visibility, wizard patterns, sensible defaults, real-time feedback, empty states, actionable errors

## Page-by-Page Results

### Login (/login) — PASS
- Flow: credentials → optional MFA → role-based redirect
- Real-time password strength feedback (L437-441)
- Rate limit countdown timer (L101-106)
- Error messages translated to user-friendly text (L127-135)
- Conditional passkey autofill support
- **Fixed**: Removed misleading hardcoded "Demo: admin / Admin@123456" hint
- **Minor**: Tenant field hidden by default, only shown via ?tenant= URL param (good UX)

### Dashboard (/dashboard) — PASS
- KPI cards load from real API (`/api/v1/identity/dashboard/stats`)
- New user quick-start cards when totalUsers <= 1
- Loading spinner while data fetches
- Graceful fallback to zeros on API failure
- Admin vs user role detection (`useUserRole`)

### Users (/users) — PASS (1 bug fixed)
- Search filter (username + email)
- Pagination (10 per page)
- Batch role assignment
- CSV import with column mapping wizard
- Export dropdown
- **Fixed (committed)**: Roles dropdown was empty — `useCallback` without `useEffect` meant roles never loaded. Changed to `useEffect` pattern.
- **UX note**: Create user form could benefit from role selection dropdown + password strength meter (assigned to shen_frontend)

### Roles (/roles) — PASS
- Tabbed interface (roles/permissions/checker/matrix/hierarchy/abac)
- Create/edit/delete with confirmation
- Error states with retry
- Role hierarchy visualization
- Permission checker tool

### Profile (/profile) — FAIL (3 issues, assigned to shen_frontend)
1. **Hardcoded data (P0)**: name="Alice Chen", email="alice@company.com", phone="+1-555-0100" — never loads from API/JWT
2. **Fake save (P1)**: `saveProfile` is `setTimeout` without API call
3. **TS error**: `s.trusted === true` compares string to boolean (L108)
4. **Missing**: No password change form in Security tab

### Audit (/audit) — PASS
- Dual view: dashboard (charts) + events (table)
- Filters: action, actor, result, IP, date range
- URL query param sync for shareable filter views
- Expandable rows for metadata
- Pagination
- Export functionality

### OAuth Clients (/oauth-clients) — PASS (2 improvements assigned)
- Create/edit/delete flow
- Secret reveal with copy button
- PascalCase to camelCase mapping (recently fixed)
- **Assigned to shen_frontend**: Add search box, empty name → show client_id[:8]

## Pages with Known Issues (from earlier verification)

### Settings — 2 FAIL
| Page | Issue | Status |
|------|-------|--------|
| /settings/password-policy | Stuck "Loading..." — API not responding | Needs backend fix |
| /settings/ldap-sync-config | Was crash — recently fixed (commit d9cb3e36) | Verify after deploy |

### Security — 1 FAIL
| Page | Issue | Status |
|------|-------|--------|
| /security/risk-score | Was crash — recently fixed (commit 2876de91) | Verify after deploy |
| /security/cae-monitor | PASS — real data | |
| /security/posture | PASS — real data | |

## Fixes Applied This Round

| File | Fix | Commit |
|------|-----|--------|
| console/src/app/users/page.tsx | Roles dropdown: useCallback→useEffect | Committed |
| console/src/app/login/page.tsx | Remove hardcoded demo hint | Committed |

## RBAC Verification

| User | Scopes | Sidebar | Result |
|------|--------|---------|--------|
| admin | ["admin"] | Full nav (DASHBOARD/IDENTITY/SECURITY/GOVERNANCE/AUDIT/PLATFORM) | PASS |
| bob | ["user"] | OVERVIEW only (Dashboard, My Sessions, Access Requests) | PASS |

## Next Steps
1. Wait for shen_frontend: Profile data loading, Users form, OAuth search
2. Wait for backend: GET /api/v1/users/me, POST /api/v1/auth/change-password
3. Redeploy console + verify all fixes
4. Round 2 UX review (cron-1 will trigger at :30 and :00)
