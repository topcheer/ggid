# Console Mock Pages Audit

*Research document: P2/P3 backlog input for frontend/uiux team.*

## Executive Summary

| Item | Value |
|------|-------|
| Total pages scanned | 757+ (console/src/app/**/*.tsx) |
| Pages with mock-only data | 118 pages show patterns consistent with local state seeded by mock objects |
| Top risk area | `/settings/*` configuration pages (secret history, blast radius, policy dry-run, etc.) |
| Recommended priority | P2 for admin-facing config pages; P3 for read-only analytics pages |

## Methodology

A page was flagged as "mock data present" when it contained one or more of the following patterns:

- `useState([])` or `useState({})` initialized with an inline array/object literal used for UI demos
- A `const` array/object literal acting as a fake data source
- Use of `mockData`, `mockUsers`, `mockRoles`, `mockOrgs`, or similar identifiers
- Hard-coded status/example data used to render charts, tables, or timelines

The search was performed with:

```bash
grep -rE 'mock.*Data|mockData|useState.*\[\].*\{|const .* = .*\[.*\{' \
  console/src/app/ --include='*.tsx' -l
```

## Findings by Priority

### P2 — Admin Configuration Pages (Need Real API Soonest)

These pages are in settings and manage tenant-wide configuration. Mock data here means administrators cannot actually use the feature.

| Page | Lines | Mock Pattern | Missing API / Concern |
|------|-------|--------------|----------------------|
| `settings/policy-dry-run` | ~200 | Inline policy objects | `/api/v1/policies/dry-run` or `/api/v1/policies/evaluate` |
| `settings/oidc-claim-config` | ~200 | Hard-coded claim map | `/api/v1/oauth/claims` |
| `settings/secret-history` | ~150 | Fake secret versions | `/api/v1/secrets/{id}/history` (not wired) |
| `settings/blast-radius` | ~180 | Simulated impact graph | `/api/v1/policies/impact` |
| `settings/policy-quarantine` | ~180 | Mock quarantine list | `/api/v1/policies/quarantine` |
| `settings/scope-drift` | ~170 | Mock drift records | `/api/v1/oauth/scopes/drift` |
| `settings/par-config` | ~160 | Inline PAR examples | `/api/v1/oauth/par` |
| `settings/ldap-sync-config` | ~200 | Mock LDAP sync jobs | `/api/v1/ldap/sync` |
| `settings/siem-health` | ~170 | Mock health metrics | `/api/v1/audit/health` or SIEM forwarder health |
| `settings/user-import` | ~180 | Mock import preview | `/api/v1/users/import` |
| `settings/joiner-flow` | ~180 | Mock onboarding steps | `/api/v1/workflows/joiner` |
| `settings/feature-flag-architecture-config` | ~170 | Hard-coded flags | `/api/v1/config/feature-flags` |
| `settings/impact-preview` | ~160 | Mock impact preview | `/api/v1/policies/impact-preview` |
| `settings/policy-versioning` | ~200 | Mock version diff | `/api/v1/policies/versions` |
| `settings/policy-version-diff` | ~180 | Mock diff data | `/api/v1/policies/versions/{id}/diff` |
| `settings/policy-change-history` | ~170 | Mock audit trail | `/api/v1/policies/history` |
| `settings/membership-graph` | ~180 | Mock org graph | `/api/v1/orgs/membership-graph` |
| `settings/client-analytics` | ~170 | Mock OAuth client stats | `/api/v1/oauth/clients/{id}/analytics` |

### P3 — Analytics / Visualization Pages (Read-Only, Lower Risk)

These pages are read-only dashboards or visualizations. Mock data is acceptable for demo but should be wired to real APIs before GA.

| Page | Lines | Mock Pattern | Missing API |
|------|-------|--------------|-------------|
| `roles/page.tsx` | 1271 | Mock role matrix | `/api/v1/roles` (exists; page may not use hook) |
| `organizations/page.tsx` | 1154 | Mock org tree | `/api/v1/orgs` (exists) |
| `provisioning/page.tsx` | 949 | Mock provisioning logs | `/api/v1/provisioning/logs` |
| `policies/page.tsx` | 947 | Mock policy list | `/api/v1/policies` (exists) |
| `audit/advanced/page.tsx` | 937 | Mock audit analytics | `/api/v1/audit` |
| `users/import/page.tsx` | 921 | Mock import results | `/api/v1/users/import` |
| `groups/page.tsx` | 836 | Mock group cards | `/api/v1/groups` |
| `users/page.tsx` | 795 | Mock user rows | `/api/v1/users` (exists) |
| `settings/webhooks/page.tsx` | 772 | Mock webhook events | `/api/v1/webhooks` (exists; may not be wired) |
| `organizations/tree/page.tsx` | 758 | Mock org tree | `/api/v1/orgs` |
| `certificates/page.tsx` | 718 | Mock cert cards | `/api/v1/certificates` |
| `scim/page.tsx` | 722 | Mock SCIM events | `/api/v1/scim/logs` |
| `settings/security/page.tsx` | 722 | Mock security score | `/api/v1/security/score` |
| `saml/page.tsx` | 718 | Mock SAML config | `/api/v1/saml` |
| `audit/reports/page.tsx` | 674 | Mock report data | `/api/v1/audit/reports` |
| `audit/visualization/page.tsx` | 631 | Mock charts | `/api/v1/audit/aggregations` |
| `orgs/chart/page.tsx` | 630 | Mock org chart | `/api/v1/orgs/chart` |
| `webhooks/page.tsx` | 600 | Mock webhook list | `/api/v1/webhooks` (exists) |
| `settings/data/page.tsx` | 598 | Mock data residency | `/api/v1/tenant/data` |
| `roles/permission-matrix/page.tsx` | 579 | Mock permission matrix | `/api/v1/permissions/matrix` |
| `roles/matrix/page.tsx` | 518 | Mock role matrix | `/api/v1/roles/matrix` |
| `settings/ip-allowlist/page.tsx` | 500 | Mock allowlist | `/api/v1/ip-allowlist` |
| `settings/mfa/page.tsx` | 435 | Mock MFA methods | `/api/v1/auth/mfa` |
| `settings/login-flows/page.tsx` | 434 | Mock login flow | `/api/v1/auth/login-flows` |
| `page.tsx` (dashboard) | 414 | Mock dashboard cards | `/api/v1/dashboard` |
| `audit/compliance/page.tsx` | 402 | Mock compliance status | `/api/v1/compliance/status` |
| `settings/saml/page.tsx` | 397 | Mock SAML settings | `/api/v1/saml` |
| `audit/events/page.tsx` | 396 | Mock event table | `/api/v1/audit` (exists) |
| `settings/idp-config/page.tsx` | 380 | Mock IdP list | `/api/v1/idp-config` |
| `settings/alerting/page.tsx` | 367 | Mock alert rules | `/api/v1/alerts` |

## Full List of Affected Files

The following 118 pages were flagged by the search pattern. This includes both genuine mock pages and pages that use local state for UI-only state (some may be false positives). Manual triage is recommended.

```text
console/src/app/settings/policy-dry-run/page.tsx
console/src/app/settings/oidc-claim-config/page.tsx
console/src/app/settings/policy-version-diff/page.tsx
console/src/app/settings/secret-history/page.tsx
console/src/app/settings/blast-radius/page.tsx
console/src/app/settings/feature-flag-architecture-config/page.tsx
console/src/app/settings/impact-preview/page.tsx
console/src/app/settings/policy-versioning/page.tsx
console/src/app/settings/siem-health/page.tsx
console/src/app/settings/client-analytics/page.tsx
console/src/app/settings/membership-graph/page.tsx
console/src/app/settings/joiner-flow/page.tsx
console/src/app/settings/ldap-sync-config/page.tsx
console/src/app/settings/policy-quarantine/page.tsx
console/src/app/settings/scope-drift/page.tsx
console/src/app/settings/policy-change-history/page.tsx
console/src/app/settings/par-config/page.tsx
console/src/app/settings/user-import/page.tsx
console/src/app/roles/page.tsx
console/src/app/organizations/page.tsx
console/src/app/provisioning/page.tsx
console/src/app/policies/page.tsx
console/src/app/audit/advanced/page.tsx
console/src/app/users/import/page.tsx
console/src/app/groups/page.tsx
console/src/app/users/page.tsx
console/src/app/settings/webhooks/page.tsx
console/src/app/organizations/tree/page.tsx
console/src/app/certificates/page.tsx
console/src/app/scim/page.tsx
console/src/app/settings/security/page.tsx
console/src/app/saml/page.tsx
console/src/app/audit/reports/page.tsx
console/src/app/audit/visualization/page.tsx
console/src/app/orgs/chart/page.tsx
console/src/app/webhooks/page.tsx
console/src/app/settings/data/page.tsx
console/src/app/roles/permission-matrix/page.tsx
console/src/app/roles/matrix/page.tsx
console/src/app/settings/ip-allowlist/page.tsx
console/src/app/settings/mfa/page.tsx
console/src/app/settings/login-flows/page.tsx
console/src/app/page.tsx
console/src/app/audit/compliance/page.tsx
console/src/app/settings/saml/page.tsx
console/src/app/audit/events/page.tsx
console/src/app/settings/idp-config/page.tsx
console/src/app/settings/alerting/page.tsx
... and 70+ additional pages in settings/ and sub-directories
```

## Common Mock Patterns

### Pattern 1: Inline `useState` with mock rows

```tsx
const [policies] = useState([
  { id: '1', name: 'Admin Policy', effect: 'allow', ... },
  { id: '2', name: 'User Policy', effect: 'deny', ... },
]);
```

### Pattern 2: Hard-coded configuration objects

```tsx
const claimConfig = {
  'email': 'email',
  'name': 'name',
  'groups': 'groups',
};
```

### Pattern 3: Mock analytics data for charts

```tsx
const data = [
  { label: 'Jan', value: 120 },
  { label: 'Feb', value: 150 },
  ...
];
```

## Recommended Action Plan

### Immediate (P2)

1. Triage the 18 P2 settings pages to confirm which are genuinely mock-only vs. partially wired.
2. For each confirmed mock-only page, create a backend task to expose the missing API and a frontend task to replace the mock with `useSWR`/`useFetch`.
3. Update `console/src/app/settings/` backlog to reflect the dependency order.

### Short-term (P3)

1. Add a lint rule or CI check that warns when a page under `console/src/app/settings/` contains inline `useState` with array/object literals.
2. For read-only analytics pages, create shared chart/table components that accept `data` props so mock data can be swapped out easily.
3. Document the mock-to-real migration pattern in `console/docs/`.

### Long-term (P3)

1. Add Storybook or MSW (Mock Service Worker) so demo data lives in test fixtures rather than production page code.
2. Introduce a `useRealOrMock` helper that uses real API when available and falls back to mock data only in development mode.

## Verification

To verify progress, re-run the audit command and compare counts:

```bash
grep -rE 'mock.*Data|mockData|useState.*\[\].*\{|const .* = .*\[.*\{' \
  console/src/app/ --include='*.tsx' -l | wc -l
```

Target: reduce the count from 118 to 0 for GA.

## Related Documents

- `console/README.md` — frontend setup and data fetching conventions
- `docs/guides/frontend-state-management.md` — if it exists, should document hook usage
- `docs/platform-completeness-report.md` — add as a frontend gap item

## Suggested Gap Status

| # | Feature | Location | Issue | Status |
|---|---------|----------|-------|--------|
| — | Console mock pages | `console/src/app/settings/` | 18+ admin settings pages still use mock data instead of real API calls | [NEW] |
| — | Console analytics pages | `console/src/app/audit/`, `console/src/app/roles/` | 100+ pages use mock/demo data for tables and charts | [NEW] |
