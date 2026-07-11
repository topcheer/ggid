# GGID Team Backlog

## Current Status: 39 packages PASS, 0 FAIL

## CRITICAL: Unwired Features (Backend — in progress)
Features with code but NOT wired into server.go (can't be called via HTTP):
- [ ] FrontChannelLogout → oauth/server
- [ ] ConditionalUI → auth/server  
- [ ] NotificationProvider → auth/server
- [ ] RiskEngine → audit/server
- [ ] Impersonation → auth/server
- [ ] SoD → policy/server
- [ ] AccessReview → audit/server
- [ ] ExpiryNotification → auth/server
- [ ] Gateway routes for all above
- [ ] Integration test for all new routes

## Frontend — Stub Pages (3 lines = placeholder!)
- [ ] console/src/app/apikeys/page.tsx (3 lines → full API key management page)
- [ ] console/src/app/oauth-clients/page.tsx (3 lines → full OAuth client management)
- [ ] console/src/app/sso/page.tsx (3 lines → full SSO configuration page)

## Frontend — Missing Feature Pages
- [ ] console/src/app/audit/risk-score/page.tsx (risk score dashboard)
- [ ] console/src/app/audit/access-reviews/page.tsx (access certification workflow)
- [ ] console/src/app/settings/notification/page.tsx (notification provider config)
- [ ] console/src/app/settings/sod/page.tsx (SoD rule management)
- [ ] console/src/app/audit/impersonation-log/page.tsx (impersonation audit trail)
- [ ] console/src/app/settings/introspection/page.tsx (token introspection cache)

## Frontend — SDK Enhancements
- [ ] useIdPConfig: add test connection button
- [ ] useSecurityCenter: add live threat map
- [ ] useSessions: add revoke all + force logout
- [ ] useUsers: add bulk import/export
- [ ] usePolicies: add policy simulator/dry-run UI
- [ ] useAuditEvents: add advanced filtering + saved searches

## Docs — Missing Tutorials (docs/tutorials/ or docs/examples/)
- [ ] react-integration.md (React SPA with GGID auth)
- [ ] nextjs-integration.md (Next.js middleware + SSR)
- [ ] kubernetes-deploy.md (K8s deployment guide)
- [ ] terraform-deploy.md (IaC deployment)
- [ ] helm-chart.md (Helm chart guide)

## Docs — API Reference Expansion
- [ ] docs/api/rest-api.md: expand to cover ALL 50+ endpoints with examples (currently 274 lines)
- [ ] docs/api/audit-api.md: NEW — audit query, stream, export, integrity, compliance, access reviews
- [ ] docs/api/policy-api.md: NEW — roles, permissions, SoD, ABAC check, dry-run, delegation
- [ ] docs/api/oauth-api.md: NEW — authorize, token, introspect, PAR, dynamic registration, front-channel

## Arch Deep Work
- [x] OpenAPI 3.0 spec (deploy/openapi.yaml)
- [x] Go SDK service clients (policy/identity/org)
- [x] SIEM regression (7 tests)
- [x] ABAC condition groups regression (6 tests)
- [x] SoD regression (9 tests)
- [x] Policy dry-run regression (3 tests)
- [ ] Integration test for unwired features (after backend wiring)
- [ ] Console i18n extraction (1051 hardcoded strings)
