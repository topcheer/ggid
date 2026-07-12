# GGID Team Backlog

*Last updated: 2026-07-12 23:00 (hourly cycle)*

## Current Stats
- **Docs**: 490 markdown files
- **Console pages**: 174 page.tsx
- **React hooks**: 88 use*.ts
- **Go SDK**: 23 files, 137 test functions
- **Go services**: 389 source files
- **Build**: `go build ./services/... && go build ./pkg/...` = CLEAN
- **Tests**: 40/40 packages PASS, 0 FAIL
- **Session commits**: 136+ across all teams

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
- [ ] **P1** ML-DSA JWT signing in pkg/crypto (PQC) — see docs/research/pqc-post-quantum-cryptography.md
- [ ] **P1** Hybrid PQC TLS in gateway (ML-KEM) — see docs/research/pqc-post-quantum-cryptography.md
- [ ] **P1** Workload identity federation (SPIFFE/SPIRE) — see docs/research/non-human-workload-identity.md
- [ ] **P1** Service account lifecycle + rotation — see docs/research/non-human-workload-identity.md
- [ ] **P1** OAuth 2.1 enforcement (mandatory PKCE, reject implicit) — see docs/research/oauth-2-1-continuous-verification.md
- [ ] **P1** Continuous session validation goroutine — see docs/research/oauth-2-1-continuous-verification.md
- [ ] **P2** Crypto-agility registry in pkg/crypto
- [ ] **P2** SLH-DSA audit log signatures
- [ ] **P2** Geo-velocity anomaly detection
- [ ] **P2** Device posture API + conditional access integration
- [ ] **P2** Agent-to-agent delegation policies

### Frontend (Next)
- [ ] Settings - Email templates editor
- [ ] Settings - WebAuthn configuration
- [ ] Dashboard - System health overview widget
- [ ] Users - Bulk import wizard (CSV upload + preview)
- [ ] Audit - Real-time event heatmap
- [ ] **P1** Service accounts management page — see docs/research/non-human-workload-identity.md
- [ ] **P2** Machine identity inventory dashboard
- [ ] **P2** Device posture dashboard

### Docs (Next)
- [ ] docs/guides/high-availability.md
- [ ] docs/guides/backup-disaster-recovery.md
- [ ] docs/api/org-api.md
- [ ] docs/guides/oauth-migration.md
- [ ] docs/research/iam-market-landscape-2026.md
- [ ] **P2** docs/guides/pqc-migration-guide.md
- [ ] **P2** docs/oauth-2-1-compliance-statement.md

### SDK (Next)
- [ ] Java SDK OAuth methods (matching Go/Node)
- [ ] Go SDK: policy versioning, device binding, deprovisioning, scope mgmt
- [ ] Node SDK: admin extensions (matching Go)
- [ ] React SDK: usePolicyVersions, useDeviceBinding hooks

---

## Research-Driven Backlog (2026-07-26)
*Source: docs/research/itdr-fraud-agent-lifecycle-gaps.md*

### Backend (P1)
- [ ] **P1** ITDR: detection rules catalog (services/auth/internal/server/itdr_handler.go)
- [ ] **P1** ITDR: automated response playbooks (block→revoke→notify→ticket)
- [ ] **P1** ITDR: lateral movement detection
- [ ] **P1** ITDR: privilege escalation detection
- [ ] **P1** ITDR: MITRE ATT&CK identity mapping
- [ ] **P1** Fraud: device fingerprinting service (pkg/fraud/)
- [ ] **P1** Fraud: velocity rules engine (max registrations/logins per IP)
- [ ] **P1** Fraud: synthetic identity detection + disposable email blocklist
- [ ] **P1** Fraud: TOR/VPN/proxy detection
- [ ] **P1** Agent: lifecycle management (onboard→provision→monitor→revoke)
- [ ] **P1** Agent: persistent registry (database-backed)
- [ ] **P1** Agent: behavioral monitoring + per-tenant rate limiting
- [ ] **P1** Agent: consent flow (user approves agent scope)
- [ ] **P1** Agent: credential rotation automation

### Frontend (P1)
- [ ] **P1** ITDR dashboard (console/src/app/settings/itdr-dashboard/)
- [ ] **P1** Fraud detection dashboard (console/src/app/settings/fraud-detection/)
- [ ] **P1** Agent lifecycle dashboard (console/src/app/settings/agent-lifecycle/)

### Backend (P2)
- [ ] **P2** PIPL: data handling rules for Chinese users
- [ ] **P2** PIPL: cross-border transfer assessment
- [ ] **P2** OAuth 2.1: compliance audit tool + deprecation enforcement
- [ ] **P2** Passkey: health dashboard API (registered passkeys per user)

### Frontend (P2)
- [ ] **P2** Passkey health dashboard (console/src/app/settings/passkey-health/)
- [ ] **P2** OAuth 2.1 compliance checker page
- [ ] **P2** PIPL compliance config page

### Docs (P2)
- [ ] **P2** docs/guides/itdr-implementation.md
- [ ] **P2** docs/guides/fraud-detection.md
- [ ] **P2** docs/guides/ai-agent-lifecycle.md
- [ ] **P2** docs/guides/pipl-compliance.md
