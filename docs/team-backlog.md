# GGID Team Backlog

*Last updated: 2026-07-12 20:30 (mid-cycle)*

## Current Stats
- **Docs**: 487 markdown files
- **Console pages**: 105 page.tsx
- **React hooks**: 40 use*.ts
- **Go SDK**: 23 files, 137 test functions
- **Go services**: 271 source files, 293 test files
- **Build**: `go build ./services/... && go build ./pkg/...` = CLEAN
- **Tests**: 38-39 packages PASS

## Completed This Session (All Teams)

### Arch Commits
- `9e0123f`: Go SDK OAuth/OIDC client (14 methods, 14 tests)
- `5e5a0ae`: Node SDK OAuth/OIDC methods (9 methods)
- `352edc6`: Go SDK admin extensions (18 methods, 13 tests)
- OpenAPI spec: 20+ new endpoints documented

### Backend Commits
- `7082072`: Permission inheritance + alert webhook
- `ad0a2ec`: Permission tree endpoint, rate limits, alert webhooks, session stream, introspection config
- `2bc64da`: Delegation validate, password history, SIEM health, account linking, consent mgmt
- `8016175`: ABAC evaluate, compliance schedule CRUD, import validate
- `44f6f63`: Role templates, correlation route confirm, bulk status, RFC 7592

### Frontend Commits
- `f07011f`: Rate limits page, hash chain page, usePermissionTree/useRateLimits
- `6cabea1`: Account linking page, useSIEMForwarder/useConsent hooks
- `81a3acb`: Login security, compliance reports, tokens pages + hooks
- `32655d8`: Role templates page, event correlation page + hooks

### Docs Commits
- `05a582e`: Policy/OAuth/Identity API refs, onboarding, session mgmt (306 docs)
- Batch 2 in progress: Keycloak migration, WebAuthn deploy, audit API, multi-tenant arch, zero-trust research

## Currently Dispatched (In Progress)

### Backend (Batch 6)
1. Policy versioning (versions + rollback)
2. Session device binding
3. Real-time alert evaluation
4. User deprovisioning workflow
5. OAuth scope management CRUD

### Frontend (Batch 5)
1. Audit deprovisioning page
2. Settings scopes page
3. Settings policy versions page
4. useDeprovisioning hook
5. useScopes hook

### Docs (Batch 2)
1. migration-keycloak.md
2. webauthn-deploy.md
3. audit-api.md (expanded)
4. multi-tenant-architecture.md
5. zero-trust-architecture.md

## Pending Backlog (Not Yet Dispatched)

### Backend (Next)
- [ ] WebAuthn attestation verification (5/6 formats unverified)
- [ ] gRPC service reflection
- [ ] Health check aggregation across services
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Database migration versioning tool

### Frontend (Next)
- [ ] Settings - Email templates editor
- [ ] Settings - WebAuthn configuration
- [ ] Dashboard - System health overview widget
- [ ] Users - Bulk import wizard (CSV upload + preview)
- [ ] Audit - Real-time event heatmap

### Docs (Next)
- [ ] docs/guides/high-availability.md
- [ ] docs/guides/backup-disaster-recovery.md
- [ ] docs/api/org-api.md
- [ ] docs/guides/oauth-migration.md
- [ ] docs/research/iam-market-landscape-2026.md

### SDK (Next)
- [ ] Java SDK OAuth methods (matching Go/Node)
- [ ] Go SDK: policy versioning, device binding, deprovisioning, scope mgmt
- [ ] Node SDK: admin extensions (matching Go)
- [ ] React SDK: usePolicyVersions, useDeviceBinding hooks
