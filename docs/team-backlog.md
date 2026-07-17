# GGID Team Backlog

*Last updated: 2026-07-17 (Round 44: Data Migration research complete — 8 new backlog items)*

## Current Stats

- **Docs**: 757 markdown files
- **Console pages**: 629 page.tsx
- **React hooks**: 492 use*.ts
- **Go SDK**: 27 files, 154+ test functions
- **Go services**: 271+ source files, 293+ test files
- **Build**: `go build ./...` = CLEAN
- **Tests**: 45/45 packages PASS, 0 FAIL
- **Real productization gaps**: 0

## Gap Closure Priority Queue

### P1 — Real productization gaps (from platform-completeness-report.md)

| # | Feature | Owner | Location | Status | Next Action |
|---|---------|-------|----------|--------|-------------|
| 1 | OAuth 2.1 compliance audit is a stub | backend | services/oauth/internal/server/oauth21_audit_handler.go | [NEW] | Implement dynamic analyzer reading real client configs |
| 2 | FAPI 2.0 profile not exposed | backend | services/oauth | [NEW] | Add fapi_2_0 client flag and enforce PAR+PKCE+DPoP+response_type=code |
| 3 | FedCM not implemented | backend | services/oauth | [NEW] | Add FedCM config/accounts/login endpoints (future, low priority) |

### P2 — Research-driven competitive/compliance gaps

| # | Feature | Owner | Driver | Notes |
|---|---------|-------|--------|-------|
| 4 | OAuth 2.1 enforcement mode | backend | RFC 9700 / OAuth 2.1 | [DONE] dfcb8a7f |
| 5 | FAPI 2.0 profile | backend | OpenID FAPI | [DONE] ccae234f |
| 6 | FedCM support | backend | Chrome/Edge default | [ACCEPTABLE] future consumer identity |
| 7 | PIPL/NIS2/CRA compliance research | docs/arch | PIPL amended CSL / NIS2 / CRA | [DONE] docs/research/nis2-cra-pipl-compliance.md |
| 8 | Passkey health dashboard | frontend | passkey adoption | Console page showing passkey enrollment status, recovery risk |
| 9 | PQC migration (ML-DSA / ML-KEM) | arch | NIST PQC | Hybrid TLS + JWT signing in pkg/crypto |
| 10 | NIS2 / CRA compliance dashboard | frontend | EU regulation | Security incident reporting, SBOM, vulnerability tracking |
| 11 | AI agent identity lifecycle | backend | agentic AI | Persistent registry, consent flow, credential rotation, drift detection |
| 12 | Fraud: TOR/VPN/proxy detection | backend | ITDR/fraud | IP intelligence integration, geo-velocity anomaly |
| 13 | **ReBAC tuple store** (P0) | backend | Google Zanzibar / fine-grained authz | PostgreSQL tuple store + repository layer. See docs/research/rebac-zanzibar-fine-grained-authz.md §13 |
| 14 | **ReBAC schema DSL parser** (P0) | backend | Zanzibar schema language | Parse `definition type { relation, permission }` syntax |
| 15 | **ReBAC graph traversal engine** (P0) | backend | Relationship graph | Recursive check with depth limiting, memoization |
| 16 | **ReBAC check/write API** (P0) | backend | REST + gRPC | /api/v1/policy/rebac/check, /tuples, /list-objects |
| 17 | **ReBAC evaluator integration** (P0) | backend | Policy service | Wire ReBAC as 3rd layer in evaluator.Check() pipeline |
| 18 | **ReBAC Redis caching** (P1) | backend | Performance | Tuple cache with write invalidation |
| 19 | **ReBAC console UI** (P2) | frontend | Developer experience | Schema editor, tuple browser, check playground |
| 20 | **Journey definition store** (P0) | backend | Configurable auth flows | PostgreSQL tables for journey_definitions, bindings, executions, sessions |
| 21 | **Journey engine + JDL parser** (P0) | backend | Identity orchestration | YAML Journey Definition Language + state machine executor. See docs/research/identity-orchestration-journeys.md |
| 22 | **Journey core nodes** (P0) | backend | Auth extensibility | password_verify, risk_assessment, mfa_orchestrate, issue_tokens, conditional (CEL) |
| 23 | **Journey management + execution API** (P0) | backend | REST + gRPC | CRUD journeys, bindings, start/resume execution, dry-run |
| 24 | **Auth service integration** (P0) | backend | Replace hardcoded Login() | Wire Journey Engine as auth flow dispatcher (backward-compatible default journey) |
| 25 | **Journey templates + dry-run** (P1) | backend | Developer experience | Pre-built templates (login, registration, recovery) + simulation mode |
| 26 | **Journey visual builder** (P2) | frontend | Console UI | Drag-and-drop canvas with React Flow, node config panel, edge labels |
| 27 | **Journey analytics** (P3) | backend | Observability | Conversion rates, drop-off points, per-step latency dashboards |
| 28 | **Cloud federation data model + service** (P0) | backend | Multi-cloud identity | PostgreSQL tables for configs, role mappings, attribute mappings. See docs/research/cloud-iam-federation.md |
| 29 | **Claim mapping engine** (P0) | backend | SAML/OIDC claims | Transform GGID attributes to AWS/Azure/GCP-specific SAML attributes |
| 30 | **AWS SAML federation module** (P0) | backend | AWS IAM | Role ARN generation, https://aws.amazon.com/SAML/Attributes/* attributes |
| 31 | **Azure SAML federation module** (P0) | backend | Azure AD | Azure claim URIs, app role mapping |
| 32 | **Federation login + Terraform snippet API** (P0) | backend | Developer experience | POST /cloud-federation/{id}/login + GET /{id}/terraform |
| 33 | **GCP workforce federation module** (P1) | backend | GCP IAM | SAML attributes for Workforce Identity Pool, CEL attribute mapping |
| 34 | **SCIM client** (P1) | backend | Auto-provisioning | Push user changes to AWS IAM Identity Center via SCIM 2.0 |
| 35 | **Federation health monitoring** (P1) | backend | Operations | Periodic checks: metadata access, cert expiry, SCIM connectivity |
| 36 | **Federation setup wizard** (P2) | frontend | Console UI | Multi-step wizard with metadata download, Terraform copy, test login |
| 37 | **Multi-hash password verifier** (P0) | backend | Migration compatibility | bcrypt, PBKDF2, scrypt, LDAP SSHA, SHA256 verification. See docs/research/data-migration-bulk-import.md |
| 38 | **Transparent rehashing** (P0) | backend | Password security | Auto-upgrade legacy hashes to Argon2id on successful login |
| 39 | **Bulk import pipeline** (P0) | backend | User migration | Async job-based JSON/CSV import with batch processing, progress tracking |
| 40 | **Dry-run validation** (P0) | backend | Safety | Validate import data without committing; return error report |
| 41 | **Lazy migration engine** (P1) | backend | Zero-downtime migration | Legacy DB connector, per-tenant config, JIT user creation on login |
| 42 | **Attribute + role mapping engine** (P1) | backend | Data transformation | Configurable field + role mapping from legacy schema to GGID schema |
| 43 | **Import wizard + dashboard** (P2) | frontend | Console UI | Multi-step wizard (upload, map, validate, import) + migration dashboard with stats |

### P3 — Quality/infrastructure improvements

| # | Feature | Owner | Notes |
|---|---------|-------|-------|
| 10 | Console loading/error states | frontend | Remaining pages: ip-allowlist, tenant-config, branding-custom, settings/page, notifications/templates |
| 11 | i18n extraction | frontend | 1051 hardcoded strings -> messages/en.json, zh.json |
| 12 | SDK parity completion | arch | Node SDK admin extensions, React hooks for risk/SOD/PAR |

## Currently Dispatched (Next 24h)

### Backend
1. (standby)

### Arch
1. Research OAuth 2.1 / FAPI 2.0 / FedCM gaps → DONE (docs/research/oauth21-fapi-fedcm-gap.md)
2. Round 26 E2E regression test
3. Release v0.2.5 verification

### Frontend
1. Console loading/error states for remaining pages
2. Passkey health dashboard

### Docs/Research
1. OAuth 2.1 / FAPI 2.0 research → DONE
2. Console mock-pages audit (continuing)
3. ReBAC / Zanzibar fine-grained authz → DONE (docs/research/rebac-zanzibar-fine-grained-authz.md) — 7 backlog items added
4. Identity Orchestration / Auth Journeys → DONE (docs/research/identity-orchestration-journeys.md) — 8 backlog items added
5. Cloud IAM Federation → DONE (docs/research/cloud-iam-federation.md) — 9 backlog items added
6. Data Migration / Bulk Import → DONE (docs/research/data-migration-bulk-import.md) — 8 backlog items added

## Rules

- Each task owner must report: commit hash + make test result
- No modification of other teammates' directories
- Gap status changes require verification (see docs/gap-maintenance-rules.md)
- Research findings go to docs/research/*.md before entering backlog
- All dependencies use @latest

## Research Pipeline

Active research topics:
- Data Migration / Bulk User Import → DONE (docs/research/data-migration-bulk-import.md)
- Cloud IAM Federation (AWS/Azure/GCP) → DONE (docs/research/cloud-iam-federation.md)
- Identity Orchestration / Configurable Auth Journeys → DONE (docs/research/identity-orchestration-journeys.md)
- ReBAC / Google Zanzibar fine-grained authorization → DONE (docs/research/rebac-zanzibar-fine-grained-authz.md)
- OAuth 2.1 / FAPI 2.0 compliance gap analysis
- PQC migration for IAM systems
- AI agent identity governance patterns
- NIS2 / CRA compliance for IAM vendors
- Console mock data audit
- **Next**: Zero Trust Network Access (ZTNA) integration / Passwordless migration

See docs/research/ for full research docs.