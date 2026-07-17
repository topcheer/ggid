# GGID Kanban Board

*Last updated: 2026-07-15 by ggcxf (researcher) — Round 2: Identity Orchestration journeys*

## Backlog (Ready for Implementation)

### P0 — Critical Security & Core Features

| ID | Title | Owner | Priority | Source | Est |
|----|-------|-------|----------|--------|-----|
| KB-001 | OAuth state validation on token exchange (P0) | backend | P0 | tech-debt.md | 1d |
| KB-002 | JWT jti tracking — anti-replay (P0) | backend | P0 | tech-debt.md | 3d |
| KB-003 | HasScope() actual scope enforcement (P0) | backend | P0 | tech-debt.md | 2d |
| KB-004 | gRPC mTLS between all services | backend | P0 | tech-debt.md | 5d |
| KB-005 | **ReBAC tuple store** (PostgreSQL) | backend | P0 | rebac-zanzibar | 3d |
| KB-006 | **ReBAC schema DSL parser** | backend | P0 | rebac-zanzibar | 4d |
| KB-007 | **ReBAC graph traversal engine** | backend | P0 | rebac-zanzibar | 5d |
| KB-008 | **ReBAC check/write REST + gRPC API** | backend | P0 | rebac-zanzibar | 4d |
| KB-009 | **ReBAC evaluator integration** (3rd authz layer) | backend | P0 | rebac-zanzibar | 2d |

### P1 — High Value Features

| ID | Title | Owner | Priority | Source | Est |
|----|-------|-------|----------|--------|-----|
| KB-010 | Password breach check (HIBP API) | backend | P1 | tech-debt.md | 2d |
| KB-011 | JWT key rotation automation | backend | P1 | tech-debt.md | 3d |
| KB-012 | Auto-generated RSA keys / shared key store | backend | P1 | tech-debt.md | 3d |
| KB-013 | NATS subject tenant namespacing | backend | P1 | tech-debt.md | 2d |
| KB-014 | Audit service RLS enforcement | backend | P1 | tech-debt.md | 2d |
| KB-015 | SCIM 2.0 server implementation | backend | P1 | roadmap.md | 5d |
| KB-016 | Session management with revocation | backend | P1 | roadmap.md | 5d |
| KB-017 | SAML IdP-initiated SSO | backend | P1 | roadmap.md | 3d |
| KB-018 | PQC migration (ML-DSA / ML-KEM) | arch | P1 | NIST PQC | 10d |
| KB-019 | AI agent identity lifecycle | backend | P1 | agentic AI | 8d |
| KB-020 | Fraud detection: TOR/VPN/proxy | backend | P1 | ITDR | 5d |
| KB-021 | **ReBAC Redis caching** | backend | P1 | rebac-zanzibar | 2d |
| KB-022 | **ReBAC caveats** (conditional permissions) | backend | P1 | rebac-zanzibar | 4d |
| KB-023 | **ReBAC ListObjects/ListSubjects API** | backend | P1 | rebac-zanzibar | 3d |

### P2 — Enhancement & Quality

| ID | Title | Owner | Priority | Source | Est |
|----|-------|-------|----------|--------|-----|
| KB-024 | Passkey health dashboard | frontend | P2 | backlog | 3d |
| KB-025 | NIS2/CRA compliance dashboard | frontend | P2 | EU regulation | 5d |
| KB-026 | Console loading/error states (remaining) | frontend | P2 | tech-debt.md | 3d |
| KB-027 | i18n extraction (1051 strings) | frontend | P2 | tech-debt.md | 5d |
| KB-028 | SDK parity completion | arch | P2 | backlog | 5d |
| KB-029 | **ReBAC console UI** (schema editor, playground) | frontend | P2 | rebac-zanzibar | 5d |
| KB-030 | **ReBAC migration tooling** (RBAC→tuple sync) | backend | P2 | rebac-zanzibar | 3d |
| KB-031 | **ReBAC Watch API** (cache invalidation) | backend | P2 | rebac-zanzibar | 4d |

### P3 — Future / Research

| ID | Title | Owner | Priority | Source |
|----|-------|-------|----------|--------|
| KB-032 | OpenTelemetry distributed tracing | arch | P3 | tech-debt.md |
| KB-033 | Multi-region active-active | arch | P3 | tech-debt.md |
| KB-034 | Vault/KMS integration | arch | P3 | tech-debt.md |
| KB-035 | Plugin system architecture | arch | P3 | tech-debt.md |
| KB-036 | ~~**Identity orchestration** (adaptive journeys)~~ → Promoted to P0/P1 below | — | DONE (research) | — |
| KB-037 | ~~**Cloud IAM federation** (AWS/Azure/GCP)~~ → Promoted to P0/P1 below | — | DONE (research) | — |
| KB-049 | **Cloud federation data model + service** | backend | P0 | cloud-iam-federation | 5d |
| KB-050 | **Claim mapping engine** (GGID attr → cloud SAML/OIDC) | backend | P0 | cloud-iam-federation | 4d |
| KB-051 | **AWS SAML federation module** (role ARNs, attributes) | backend | P0 | cloud-iam-federation | 3d |
| KB-052 | **Azure SAML federation module** (app roles, claim URIs) | backend | P0 | cloud-iam-federation | 3d |
| KB-053 | **Federation login + Terraform snippet API** | backend | P0 | cloud-iam-federation | 5d |
| KB-054 | **GCP workforce federation module** | backend | P1 | cloud-iam-federation | 3d |
| KB-055 | **SCIM client** (push users to AWS IAM Identity Center) | backend | P1 | cloud-iam-federation | 4d |
| KB-056 | **Federation health monitoring** | backend | P1 | cloud-iam-federation | 3d |
| KB-057 | **Federation setup wizard** (Console multi-cloud) | frontend | P2 | cloud-iam-federation | 5d |
| KB-058 | **Multi-hash password verifier** (bcrypt/PBKDF2/scrypt/SSHA) | backend | P0 | data-migration | 4d |
| KB-059 | **Transparent rehashing** (auto-upgrade to Argon2id) | backend | P0 | data-migration | 2d |
| KB-060 | **Bulk import pipeline** (async job, JSON/CSV) | backend | P0 | data-migration | 5d |
| KB-061 | **Dry-run validation** (test import without committing) | backend | P0 | data-migration | 2d |
| KB-062 | **Lazy migration engine** (JIT from legacy DB) | backend | P1 | data-migration | 5d |
| KB-063 | **Attribute + role mapping engine** | backend | P1 | data-migration | 3d |
| KB-064 | **Import wizard** (Console multi-step) | frontend | P2 | data-migration | 5d |
| KB-065 | **Migration dashboard** (stats, progress, errors) | frontend | P2 | data-migration | 3d |
| KB-066 | **Device posture API + evaluation engine** | backend | P0 | ztna-integration | 5d |
| KB-067 | **Gateway device posture middleware** (per-request check) | backend | P0 | ztna-integration | 2d |
| KB-068 | **SAML groups claim standardization** | backend | P0 | ztna-integration | 2d |
| KB-069 | **SCIM outbound client** (push to ZTNA brokers) | backend | P0 | ztna-integration | 4d |
| KB-070 | **CAEP event transmitter** (continuous verification) | backend | P1 | ztna-integration | 4d |
| KB-071 | **Provider setup guide generator** (Terraform snippets) | backend | P1 | ztna-integration | 3d |
| KB-072 | **ZTNA dashboard + provider wizard** (Console) | frontend | P2 | ztna-integration | 5d |
| KB-073 | **Auth method policy engine** (require/forbid methods per group) | backend | P0 | passwordless-migration | 4d |
| KB-074 | **Password deprecation enforcement** (4 levels: off→disabled) | backend | P0 | passwordless-migration | 2d |
| KB-075 | **Enrollment nudge system** (banners, segments, triggers) | backend | P0 | passwordless-migration | 3d |
| KB-076 | **Temporary Access Pass** (passwordless recovery) | backend | P1 | passwordless-migration | 4d |
| KB-077 | **Migration metrics API** (enrollment rate, AAL distribution) | backend | P1 | passwordless-migration | 3d |
| KB-078 | **Passkey profiles** (AAGUID allow-list enforcement) | backend | P1 | passwordless-migration | 3d |
| KB-079 | **Passwordless migration dashboard** (Console) | frontend | P2 | passwordless-migration | 4d |
| KB-080 | **Universal JIT engine** (SAML/OIDC/LDAP/SCIM) | backend | P0 | jit-provisioning | 5d |
| KB-081 | **Attribute mapping DSL parser** (YAML declarative) | backend | P0 | jit-provisioning | 3d |
| KB-082 | **SAML JIT integration** (auto-create from assertion) | backend | P0 | jit-provisioning | 3d |
| KB-083 | **OIDC JIT integration** (auto-create from claims) | backend | P0 | jit-provisioning | 3d |
| KB-084 | **Role/group mapping engine** (external groups → GGID roles) | backend | P0 | jit-provisioning | 3d |
| KB-085 | **JIT update + deprovisioning** (sync on login + disable on removal) | backend | P1 | jit-provisioning | 3d |
| KB-086 | **SCIM inbound JIT** (enhance SCIM handler for external push) | backend | P1 | jit-provisioning | 3d |
| KB-087 | **JIT config + dry-run API** (CRUD + simulation) | backend | P1 | jit-provisioning | 3d |
| KB-088 | **JIT provisioning dashboard** (Console) | frontend | P2 | jit-provisioning | 3d |
| KB-089 | **Delegation DB persistence** (replace in-memory store) | backend | P0 | delegation-patterns | 3d |
| KB-090 | **Delegation management API** (REST CRUD + check) | backend | P0 | delegation-patterns | 3d |
| KB-091 | **Policy evaluator integration** (check delegated permissions) | backend | P0 | delegation-patterns | 2d |
| KB-092 | **Per-resource delegation scoping** | backend | P0 | delegation-patterns | 2d |
| KB-093 | **Delegation policy DSL** (YAML who→what→whom) | backend | P1 | delegation-patterns | 3d |
| KB-094 | **JWT act claim injection** (delegation in access token) | backend | P1 | delegation-patterns | 3d |
| KB-095 | **Delegation approval workflow** (sensitive delegations) | backend | P1 | delegation-patterns | 3d |
| KB-096 | **Delegation dashboard** (Console self-service UI) | frontend | P2 | delegation-patterns | 3d |
| KB-097 | **Plugin DB schema + repository** (PostgreSQL-backed) | backend | P0 | wasm-plugin | 2d |
| KB-098 | **Plugin management API** (upload/enable/disable/test) | backend | P0 | wasm-plugin | 3d |
| KB-099 | **Per-tenant runtime isolation + resource limits** | backend | P0 | wasm-plugin | 3d |
| KB-100 | **Hook dispatcher** (10 hooks: auth/token/policy/jit/gateway) | backend | P0 | wasm-plugin | 4d |
| KB-101 | **Auth/token/policy hook integration** | backend | P1 | wasm-plugin | 5d |
| KB-102 | **Plugin SDK** (Go + Rust bindings) | backend | P1 | wasm-plugin | 3d |
| KB-103 | **Plugin dashboard + upload wizard** (Console) | frontend | P2 | wasm-plugin | 3d |
| KB-104 | **Analytics event pipeline** (NATS collector + DB store) | backend | P0 | analytics-reporting | 4d |
| KB-105 | **Aggregation jobs + overview API** (hourly/daily rollups) | backend | P0 | analytics-reporting | 5d |
| KB-106 | **Auth method + trends analytics API** | backend | P0 | analytics-reporting | 3d |
| KB-107 | **Anomaly detection engine** (3σ + impossible travel) | backend | P1 | analytics-reporting | 4d |
| KB-108 | **Scheduled compliance reports** (SOC2/GDPR/ISO PDF) | backend | P1 | analytics-reporting | 4d |
| KB-109 | **Analytics dashboard** (Console) | frontend | P2 | analytics-reporting | 4d |
| KB-110 | **Data export + custom dashboards** | backend | P2 | analytics-reporting | 4d |
| KB-038 | **Journey definition store** (PostgreSQL) | backend | P0 | identity-orchestration | 3d |
| KB-039 | **Journey definition parser** (YAML JDL) | backend | P0 | identity-orchestration | 4d |
| KB-040 | **Journey engine** (state machine executor) | backend | P0 | identity-orchestration | 5d |
| KB-041 | **Core node executors** (password, risk, MFA, tokens) | backend | P0 | identity-orchestration | 4d |
| KB-042 | **CEL condition evaluator** | backend | P0 | identity-orchestration | 3d |
| KB-043 | **Journey management + execution API** | backend | P0 | identity-orchestration | 5d |
| KB-044 | **Auth service integration** (replace hardcoded flow) | backend | P0 | identity-orchestration | 3d |
| KB-045 | **Dry-run / simulation mode** | backend | P1 | identity-orchestration | 3d |
| KB-046 | **Journey templates** (login, registration, recovery) | backend | P1 | identity-orchestration | 2d |
| KB-047 | **Visual flow builder** (Console drag-and-drop) | frontend | P2 | identity-orchestration | 5d |
| KB-048 | **Journey analytics** (conversion, drop-off, latency) | backend | P3 | identity-orchestration | 3d |

## In Progress

*(None currently)*

## Done (Recent)

| ID | Title | Owner | Commit |
|----|-------|-------|--------|
| — | Fine-grained Delegation research doc | researcher | pending commit |
| — | JIT User Provisioning research doc | researcher | 91938ddd |
| — | Passwordless Migration research doc | researcher | ddf0fbe4 |
| — | ZTNA Broker Integration research doc | researcher | 9c8ed3cf |
| — | Data Migration research doc | researcher | 02c52040 |
| — | Cloud IAM Federation research doc | researcher | 5880e4ac |
| — | Identity Orchestration / Journeys research doc | researcher | 4f76fbdc |
| — | ReBAC/Zanzibar research doc | researcher | 4ce3b8ba |
| — | PIPL/NIS2/CRA compliance research | arch | done |
| — | OAuth 2.1 enforcement mode | backend | dfcb8a7f |
| — | FAPI 2.0 profile | backend | ccae234f |
| F-42 | WASM Plugin Management Console (5 tabs, 10 hooks, upload wizard) | frontend | f7355534 |
| F-43 | Identity Analytics & Reporting Dashboard (5 tabs, SVG charts, compliance gen) | frontend | fa83a950 |

## Research Pipeline

| Topic | Status | Doc |
|-------|--------|-----|
| ReBAC / Google Zanzibar | DONE | docs/research/rebac-zanzibar-fine-grained-authz.md |
| OAuth 2.1 / FAPI 2.0 | DONE | docs/research/oauth21-fapi-fedcm-gap.md |
| PIPL/NIS2/CRA | DONE | docs/research/nis2-cra-pipl-compliance.md |
| Identity Orchestration | DONE | docs/research/identity-orchestration-journeys.md |
| Cloud IAM Federation | DONE | docs/research/cloud-iam-federation.md |
| Data Migration / Bulk Import | DONE | docs/research/data-migration-bulk-import.md |
| Zero Trust Network Access (ZTNA) | DONE | docs/research/ztna-broker-integration.md |
| Passwordless Migration | DONE | docs/research/passwordless-migration-strategy.md |
| JIT User Provisioning | DONE | docs/research/jit-user-provisioning.md |
| Fine-grained Delegation Patterns | DONE | docs/research/fine-grained-delegation-patterns.md |
| WASM Plugin Architecture | DONE | docs/research/wasm-plugin-architecture.md |
| Identity Analytics & Reporting | DONE | docs/research/identity-analytics-reporting.md |
| GraphQL API Layer for Identity Queries | NEXT | — |
| PQC Migration | IN PROGRESS | docs/guides/post-quantum-crypto-migration.md |

---

*Board maintained by researcher (ggcxf). Implementation items are ready for pickup by backend/frontend/arch.*
