# GGID Team Backlog

*Last updated: 2026-07-17 (Round 64: API Gateway Hardening research complete — 4 new backlog items)*

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
| 44 | **Device posture API + evaluation engine** (P0) | backend | ZTNA integration | Standardized posture API for ZTNA brokers + configurable policy engine. See docs/research/ztna-broker-integration.md |
| 45 | **Gateway device posture middleware** (P0) | backend | Per-request enforcement | DevicePosture middleware in gateway chain, Redis-cached posture check |
| 46 | **SAML groups claim standardization** (P0) | backend | ZTNA policy | Standardized groups attribute in SAML assertions for ZTNA broker policies |
| 47 | **SCIM outbound client** (P0) | backend | ZTNA provisioning | Push users/groups to Zscaler/Cloudflare/Twingate/Tailscale via SCIM 2.0 |
| 48 | **CAEP event transmitter** (P1) | backend | Continuous verification | Push CAEP SET tokens (session-revoked, credential-change) to ZTNA brokers |
| 49 | **ZTNA provider setup guide generator** (P1) | backend | Developer experience | Auto-generate provider-specific Terraform/config snippets |
| 50 | **ZTNA dashboard + provider wizard** (P2) | frontend | Console UI | Provider status cards, posture compliance, CAEP events, setup wizard |
| 51 | **Auth method policy engine** (P0) | backend | Passwordless migration | Declarative policy: require/forbid auth methods per group/app/date. See docs/research/passwordless-migration-strategy.md |
| 52 | **Password deprecation enforcement** (P0) | backend | Password phase-out | 4 levels: off → warn → secondary → disabled; auto-disable for passkey users |
| 53 | **Enrollment nudge system** (P0) | backend | User adoption | Configurable banners with segments (no_passkey), triggers (after_login), dismiss tracking |
| 54 | **Temporary Access Pass** (P1) | backend | Passwordless recovery | Admin-issued short-lived recovery pass for lost passkey devices; TAP + new enrollment |
| 55 | **Migration metrics API** (P1) | backend | Analytics | Enrollment rate, AAL distribution, password usage trend, helpdesk ticket reduction |
| 56 | **Passkey profiles** (P1) | backend | Authenticator restriction | AAGUID allow-list, attestation requirements, user verification mode enforcement |
| 57 | **Passwordless migration dashboard** (P2) | frontend | Console UI | Enrollment rate gauge, trend chart, AAL distribution, policy list, nudge config |
| 58 | **Universal JIT engine** (P0) | backend | Multi-source provisioning | Protocol-agnostic JIT from SAML/OIDC/LDAP/SCIM. See docs/research/jit-user-provisioning.md |
| 59 | **Attribute mapping DSL** (P0) | backend | Data transformation | Declarative YAML mapping: external attr → GGID fields, per-IdP configurable |
| 60 | **SAML JIT integration** (P0) | backend | SAML federation | Auto-create/update user from SAML assertion at /saml/acs |
| 61 | **OIDC JIT integration** (P0) | backend | OIDC federation | Auto-create/update from OIDC ID Token / UserInfo claims |
| 62 | **Role/group mapping** (P0) | backend | Authorization | Map external groups (LDAP CN, SAML groups, OIDC claims) to GGID roles |
| 63 | **JIT update + deprovisioning** (P1) | backend | Lifecycle | Update attrs on login; disable users removed from IdP |
| 64 | **SCIM inbound JIT + dry-run API** (P1) | backend | SCIM provisioning | Enhance SCIM handler; JIT simulation/test mode |
| 65 | **JIT provisioning dashboard** (P2) | frontend | Console UI | Source cards, event log, config editor, test mode |
| 66 | **Delegation DB persistence** (P0) | backend | Delegation framework | Replace in-memory store with PostgreSQL. See docs/research/fine-grained-delegation-patterns.md |
| 67 | **Delegation management API + evaluator** (P0) | backend | Self-service delegation | REST CRUD + policy evaluator checks delegated permissions |
| 68 | **Per-resource delegation scoping** (P0) | backend | Fine-grained control | Scope delegation to specific resource_type + resource_id, not just permission keys |
| 69 | **Delegation policy DSL + JWT act claim** (P1) | backend | Policy enforcement | YAML who→what→whom policy + act claim in delegated access tokens |
| 70 | **Delegation approval workflow** (P1) | backend | Sensitive delegations | Manager/director approval for sensitive permission delegation |
| 71 | **Delegation dashboard** (P2) | frontend | Console UI | Granted by me / granted to me lists, create wizard, activity log |
| 72 | **Plugin DB schema + repository** (P0) | backend | Plugin persistence | PostgreSQL-backed plugin registry with .wasm binary storage. See docs/research/wasm-plugin-architecture.md |
| 73 | **Plugin management API** (P0) | backend | Plugin CRUD | Upload/enable/disable/test/version plugins via REST |
| 74 | **Per-tenant runtime isolation + resource limits** (P0) | backend | Sandbox security | Separate wazero runtime per tenant + ResourceLimiter (memory/fuel/timeout) |
| 75 | **Universal hook dispatcher** (P0) | backend | Pipeline integration | 10 hooks across auth/token/policy/jit/gateway services |
| 76 | **Auth/token/policy hook integration** (P1) | backend | Custom logic | post_login claim injection, pre_issue token modify, pre_check policy override |
| 77 | **Plugin SDK** (P1) | backend | Developer experience | Go + Rust bindings, example plugins, documentation |
| 78 | **Plugin dashboard** (P2) | frontend | Console UI | Plugin list, upload wizard, hook selector, test mode, execution stats |
| 79 | **Analytics event pipeline** (P0) | backend | Data collection | NATS event collector → analytics_events table. See docs/research/identity-analytics-reporting.md |
| 80 | **Aggregation jobs + analytics APIs** (P0) | backend | Metrics rollups | Hourly/daily rollup jobs + overview/trends/method APIs |
| 81 | **Anomaly detection engine** (P1) | backend | Threat detection | 3σ behavioral baselining, impossible travel, new location/IP detection |
| 82 | **Scheduled compliance reports** (P1) | backend | Compliance | SOC2/GDPR/ISO PDF report generation with cron scheduling + email delivery |
| 83 | **Analytics dashboard + export** (P2) | frontend+backend | Console UI | Overview cards, trend charts, method distribution, CSV/JSON export |
| 84 | **GraphQL SDL schema + gqlgen setup** (P0) | backend | API modernization | Typed schema for identity/policy/oauth/audit domains. See docs/research/graphql-api-layer.md |
| 85 | **GraphQL resolvers + dataloaders** (P0) | backend | Query optimization | Query/nested resolvers with dataloader batch resolution (prevents N+1) |
| 86 | **Field-level @auth directive** (P0) | backend | Authorization | @auth(requires: ADMIN) on field-level with PDP integration |
| 87 | **Query complexity analysis** (P0) | backend | DoS prevention | Depth limit (≤10) + cost limit (≤1000) to prevent nested query attacks |
| 88 | **GraphQL mutations + persisted queries** (P1) | backend | Full CRUD via GraphQL | Create/update/disable user, assign/revoke role via mutations; persisted query allow-list |
| 89 | **Console GraphQL playground** (P2) | frontend | Developer experience | GraphiQL in dev mode, Console dashboard migration to GraphQL |
| 90 | **API key DB storage + hashing** (P0) | backend | Replace in-memory | Replace api_keys_handler.go in-memory map with PostgreSQL + SHA-256 hashing. See docs/research/api-key-lifecycle.md |
| 91 | **API key CRUD + gateway validator** (P0) | backend | M2M auth | DB-backed CRUD handler + gateway APIKeyValidator with Redis cache + expiry enforcement |
| 92 | **Key rotation + scope enforcement** (P1) | backend | Lifecycle | Rotation with 24h grace period + per-route scope checking + per-key Redis rate limiting |
| 93 | **IP allow-list + usage tracking** (P1) | backend | Security | CIDR IP binding + async last_used/usage_count tracking |
| 94 | **Console key management UI** (P2) | frontend | Self-service | Key create/list/revoke + rotation with grace countdown + usage analytics |
| 95 | **Recovery DB schema + repository** (P0) | backend | Replace in-memory | Replace identity_recovery.go in-memory map with PostgreSQL. See docs/research/credential-recovery-workflow.md |
| 96 | **Recovery API + multi-factor pipeline** (P0) | backend | Recovery engine | 6 REST endpoints + multi-factor verification (≥2 for medium risk) + crypto tokens |
| 97 | **Temporary Access Pass** (P0) | backend | Passkey recovery | 15-min single-use TAP for lost passkey device → new credential enrollment |
| 98 | **Risk-based delay + admin approval** (P1) | backend | Security | Graduated wait (0min→72h) based on risk score + dual-control admin approval |
| 99 | **Recovery console UI** (P2) | frontend | Self-service | Self-service recovery page + admin approvals console + audit trail |
| 100 | **Consent DB schema + repository** (P0) | backend | Replace mock | Replace consent_registry_handler.go mock + consent.go in-memory with PostgreSQL. See docs/research/consent-management-platform.md |
| 101 | **Consent CRUD API + withdrawal cascade** (P0) | backend | Consent lifecycle | Grant/list/withdraw APIs + cascade engine (email + token revoke + 3rd-party notify) |
| 102 | **DSR workflow + GPC detection** (P1) | backend | GDPR/CCPA compliance | Data Subject Request pipeline (access/deletion/portability) + Global Privacy Control auto-withdraw |
| 103 | **Policy versioning + re-consent** (P1) | backend | Compliance | Privacy policy version bump → supersede old consents → prompt re-consent |
| 104 | **Console preference center + cookie banner** (P2) | frontend | Self-service | Preference center UI + admin DSR console + cookie consent banner with GPC detection |
| 105 | **Unified Risk Engine** (P0) | backend | Adaptive auth | Composite risk scorer aggregating 5 signal categories (device/geo/network/behavioral/session). See docs/research/risk-adaptive-auth-engine.md |
| 106 | **Risk assessment + config API** (P0) | backend | Replace 3 engines | Unified API replacing 3 in-memory risk engines + 12 hardcoded handlers with DB-backed URE |
| 107 | **Decision policy + step-up** (P1) | backend | Adaptive decisions | Risk → action mapping (allow/step_up/step_up_strong/block) + step-up integration |
| 108 | **Behavioral baselines + analytics** (P1) | backend | Continuous auth | 30-day per-user baselines (hours/locations/devices) + risk score analytics dashboard |
| 109 | **Risk dashboard + CAE middleware** (P2) | backend+frontend | UI + continuous | Risk analytics dashboard + continuous evaluation middleware for active sessions |
| 110 | **Agent registry + DB-backed consent** (P0) | backend | Agent identity | Agent CRUD API + replace in-memory agent_consent_handler.go with PostgreSQL. See docs/research/ai-agent-identity.md |
| 111 | **Token exchange + workload attestation** (P0) | backend | Delegated auth | RFC 8693 token exchange with act claim + k8s SA/SPIFFE workload attestation |
| 112 | **Multi-agent delegation + rate limiting** (P1) | backend | Agent chains | Max depth enforcement + cycle detection + scope narrowing + per-agent Redis rate limiting |
| 113 | **Agent behavioral anomaly + console** (P2) | backend+frontend | Security | Per-agent behavioral baselines + anomaly detection + dual-attribution audit + Console UI |
| 114 | **DB-backed VC/DID core** (P0) | backend | Replace in-memory | Replace vc_issuer.go (sync.RWMutex + 4 maps) + did_resolver.go (sync.RWMutex + map) with PostgreSQL. See docs/research/decentralized-identity-vc.md |
| 115 | **Credential schema + StatusList2021** (P0) | backend | Revocation | Schema registry CRUD + bitmap revocation status list (RFC 9114) + GZIP compression |
| 116 | **OID4VCI + OID4VP flows** (P1) | backend | Credential exchange | OID4VCI issuance (offer + pre-authz code) + OID4VP presentation verification + asymmetric SD-JWT |
| 117 | **Console credential manager + did:ebsi** (P2) | frontend+backend | EU compliance | VC issue/list/revoke UI + did:ebsi resolution (eIDAS 2.0 compatibility) |
| 118 | **CAE middleware** (P0) | backend | Zero Trust | Continuous evaluation — per-request risk re-score + session risk every 15min. See docs/research/zero-trust-maturity-assessment.md |
| 119 | **MDM integration + device certs** (P0) | backend | Zero Trust | Intune/Jamf MDM connector for device compliance + device certificate issuance/validation at gateway |
| 120 | **CMK/KMS + DLP egress** (P0) | backend | Zero Trust | Per-tenant encryption keys (AWS KMS/Vault) + DLP response scanning middleware at API egress |
| 121 | **Network policy + mTLS enforcement** (P1) | backend | Zero Trust | Service-to-service mTLS enforcement + declarative network policy engine + microsegmentation |
| 122 | **Access review / certification** (P1) | backend | Governance | Periodic access recertification campaigns + manager review workflow |
| 123 | **Unified PDP** (P0) | backend | Continuous authz | Combine RBAC+ABAC+ReBAC+risk in one Authorize() RPC. See docs/research/continuous-authorization-pdp.md |
| 124 | **Gateway PEP + Redis cache** (P0) | backend | Per-request authz | Gateway middleware calls PDP on every request + 5s Redis decision cache (<1ms cached, <5ms uncached) |
| 125 | **DB-backed decision audit** (P0) | backend | Compliance | Replace in-memory decision log (evaluator.go:54 sync.Mutex+slice) with PostgreSQL + NATS async publish |
| 126 | **Risk overlay + decision analytics** (P1) | backend | Adaptive authz | Risk score upgrades decision to step_up + decision audit trail query API (replace hardcoded stats) |
| 127 | **MDM connector framework** (P0) | backend | Device compliance | Intune Graph API + Jamf Pro API connectors for device compliance posture. See docs/research/mdm-integration.md |
| 128 | **Compliance policy engine** (P0) | backend | Device security | Configurable rules (min OS, encryption, jailbreak) + scoring + per-platform policies |
| 129 | **Compliance webhooks + SCEP + attestation** (P1-P2) | backend | Real-time + certs | MDM webhooks → CAE session revocation + SCEP device cert provisioning + TPM/Apple/Android attestation |
| 130 | **DataKeyProvider + envelope encryption** (P0) | backend | Data security | Extend key_provider.go with GenerateDataKey/Encrypt/Decrypt (DEK/KEK hierarchy). See docs/research/customer-managed-keys-kms.md |
| 131 | **Per-tenant CMK management** (P0) | backend | Data security | CRUD API for per-tenant customer-managed keys (AWS KMS + Vault + Azure) + BYOK import + rotation |
| 132 | **SM4 + field-level encryption** (P1) | backend | China + PII | SM4 symmetric encryption (GM compliance) + AES-256-GCM field-level encryption for PII columns |
| 133 | **DLP egress middleware + PII redaction** (P0) | backend | Data egress | Gateway middleware inspecting API responses for PII + redaction engine. See docs/research/dlp-egress-pii-redaction.md |
| 134 | **Egress policy DSL + replace mock** (P1) | backend | DLP rules | Declarative redaction rules + classification-driven auto-masking + replace auth DLP hardcoded mock |
| 135 | **Istio mTLS + microsegmentation** (P0) | devops | Network pillar | Istio sidecar auto-injection on k3s + mTLS STRICT + zone-based default-deny segmentation. See docs/research/service-mesh-microsegmentation.md |
| 136 | **ExtAuthz + mesh policy API** (P1) | backend | East-west authz | Envoy ExtAuthz adapter calling GGID PDP for inter-service authz + SPIFFE ID registry + mesh policy CRUD |
| 137 | **Mesh observability** (P2) | devops | Visibility | Jaeger distributed tracing + Prometheus mesh metrics + traffic visualization |
| 138 | **Hierarchical rate limiting + circuit breaker** (P0) | backend | Gateway hardening | Per-user/key/IP/endpoint rate limits (Redis) with burst/sustained + per-backend circuit breaker. See docs/research/api-gateway-hardening.md |
| 139 | **Request validation + observability** (P1) | backend | Gateway hardening | JSON schema validation + SQLi/XSS payload sanitization + P50/P95/P99 latency histograms |

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
7. ZTNA Broker Integration → DONE (docs/research/ztna-broker-integration.md) — 7 backlog items added
8. Passwordless Migration → DONE (docs/research/passwordless-migration-strategy.md) — 7 backlog items added
9. JIT User Provisioning → DONE (docs/research/jit-user-provisioning.md) — 9 backlog items added
10. Fine-grained Delegation → DONE (docs/research/fine-grained-delegation-patterns.md) — 8 backlog items added
11. WASM Plugin Architecture → DONE (docs/research/wasm-plugin-architecture.md) — 7 backlog items added
12. Identity Analytics & Reporting → DONE (docs/research/identity-analytics-reporting.md) — 7 backlog items added
13. GraphQL API Layer → DONE (docs/research/graphql-api-layer.md) — 7 backlog items added
14. API Key Lifecycle Management → DONE (docs/research/api-key-lifecycle.md) — 7 backlog items added
15. Credential Recovery Workflow Engine → DONE (docs/research/credential-recovery-workflow.md) — 6 backlog items added
16. Consent Management Platform → DONE (docs/research/consent-management-platform.md) — 6 backlog items added
17. Risk-Based Adaptive Authentication Engine → DONE (docs/research/risk-adaptive-auth-engine.md) — 6 backlog items added
18. AI Agent Identity & Delegated Access → DONE (docs/research/ai-agent-identity.md) — 6 backlog items added
19. Decentralized Identity & Verifiable Credentials → DONE (docs/research/decentralized-identity-vc.md) — 6 backlog items added
20. Zero Trust Maturity Assessment → DONE (docs/research/zero-trust-maturity-assessment.md) — 6 backlog items added
21. Continuous Authorization & PDP → DONE (docs/research/continuous-authorization-pdp.md) — 6 backlog items added
22. MDM Integration → DONE (docs/research/mdm-integration.md) — 6 backlog items added
23. CMK & KMS Integration → DONE (docs/research/customer-managed-keys-kms.md) — 6 backlog items added
24. DLP Egress Control & PII Redaction → DONE (docs/research/dlp-egress-pii-redaction.md) — 4 backlog items added
25. Service Mesh & Microsegmentation → DONE (docs/research/service-mesh-microsegmentation.md) — 5 backlog items added
26. API Gateway Hardening & Rate Limiting → DONE (docs/research/api-gateway-hardening.md) — 4 backlog items added

## Rules

- Each task owner must report: commit hash + make test result
- No modification of other teammates' directories
- Gap status changes require verification (see docs/gap-maintenance-rules.md)
- Research findings go to docs/research/*.md before entering backlog
- All dependencies use @latest

## Research Pipeline

Active research topics:
- Decentralized Identity & Verifiable Credentials → DONE (docs/research/decentralized-identity-vc.md)
- Zero Trust Maturity Assessment → DONE (docs/research/zero-trust-maturity-assessment.md)
- Continuous Authorization & PDP → DONE (docs/research/continuous-authorization-pdp.md)
- MDM Integration → DONE (docs/research/mdm-integration.md)
- CMK & KMS Integration → DONE (docs/research/customer-managed-keys-kms.md)
- DLP Egress Control → DONE (docs/research/dlp-egress-pii-redaction.md)
- Service Mesh & Microsegmentation → DONE (docs/research/service-mesh-microsegmentation.md)
- API Gateway Hardening → DONE (docs/research/api-gateway-hardening.md)
- AI Agent Identity & Delegated Access → DONE (docs/research/ai-agent-identity.md)
- Risk-Based Adaptive Authentication Engine → DONE (docs/research/risk-adaptive-auth-engine.md)
- Consent Management Platform → DONE (docs/research/consent-management-platform.md)
- Credential Recovery Workflow Engine → DONE (docs/research/credential-recovery-workflow.md)
- API Key Lifecycle Management → DONE (docs/research/api-key-lifecycle.md)
- GraphQL API Layer → DONE (docs/research/graphql-api-layer.md)
- Identity Analytics & Reporting → DONE (docs/research/identity-analytics-reporting.md)
- WASM Plugin Architecture → DONE (docs/research/wasm-plugin-architecture.md)
- Fine-grained Delegation Patterns → DONE (docs/research/fine-grained-delegation-patterns.md)
- JIT User Provisioning → DONE (docs/research/jit-user-provisioning.md)
- Passwordless Migration Strategy → DONE (docs/research/passwordless-migration-strategy.md)
- ZTNA Broker Integration → DONE (docs/research/ztna-broker-integration.md)
- Data Migration / Bulk User Import → DONE (docs/research/data-migration-bulk-import.md)
- Cloud IAM Federation (AWS/Azure/GCP) → DONE (docs/research/cloud-iam-federation.md)
- Identity Orchestration / Configurable Auth Journeys → DONE (docs/research/identity-orchestration-journeys.md)
- ReBAC / Google Zanzibar fine-grained authorization → DONE (docs/research/rebac-zanzibar-fine-grained-authz.md)
- OAuth 2.1 / FAPI 2.0 compliance gap analysis
- PQC migration for IAM systems
- AI agent identity governance patterns
- NIS2 / CRA compliance for IAM vendors
- Console mock data audit
- **Next**: Verifiable Credentials / W3C DID

See docs/research/ for full research docs.
---

## Access Broker / Identity-Aware Proxy (ZTNA) (2026-07-17 第12小时研究) - Priority: P1 - Status: Proposed - Suggested: backend + arch

**市场背景**: Identity-Aware Proxy (IAP) / ZTNA 是 2025 企业安全最高优先级投资。Cloudflare Access、Google IAP、AWS Verified Access 全部增长 40%+。核心价值：**替代 VPN** — 任何内部应用（Jenkins/Grafana/Jupyter/internal dashboard）放在 GGID 身份层后面，按身份+上下文授权访问，无需 VPN。

**GGID 现状审计**: gateway 已有 reverse proxy + route config（path_prefix→backend URL）+ JWT 验证 + ABAC + CAE。**基础设施完全就绪**，差一个"受保护应用注册"层：
- 当前 routes 硬编码为 GGID 自己的 7 个服务（identity/auth/oauth/...）
- 无法注册"外部应用"（如 https://ggid.example.com/grafana/ → 代理到 Grafana 实例 + GGID JWT 验证）
- 无 per-application access policy（"Grafana 只允许 SRE 角色 + 工作时间 + 设备可信"）

**业务价值**: HIGH
- GGID 从"IAM 平台"升级为"Zero Trust Access Platform"（市场天花板提升 3-5x）
- 替代 VPN 是 CISO 2025 第一预算项
- GGID 已有全部组件：reverse proxy + JWT + ABAC + CAE + device trust + ITDR
- 差异化：国内无同类产品（竹云/玉籽数科均无 ZTNA 能力）

**实现难度**: Medium
- 实现路径（完整一步到位，不降级）：
  1. **受保护应用注册**：protected_apps 表（name, path_prefix, backend_url, access_policy_id, tenant_id, auth_mode[jwt/session/anonymous]）
  2. **Gateway 动态路由**：启动时 + 热加载（Redis pub/sub 或 DB 轮询）从 protected_apps 表构建路由表，与现有静态路由合并
  3. **Per-app access policy**：复用 ABAC PDP（$security.* + $data.classification），增加 $app.name / $app.backend 属性
  4. **Session passthrough**：对不支持 OAuth 的传统应用（Jenkins/Grafana），GGID 完成认证后注入 header（X-Authenticated-User + X-Tenant-ID），后端配置 trusted header 模式
  5. **Access logs**：每次受保护应用访问写 audit 事件（与 ITDR 联动：异常访问模式自动检测）
  6. **Console 管理页面**：受保护应用列表 + CRUD + 实时访问日志 + access policy 配置器（可视化条件构建器）

**兼容性**: gateway 已有全部基础设施（reverse proxy + JWT + ABAC + CAE），纯增量注册层 + 动态路由
**工作量**: ~5d（gateway 动态路由 2d + protected_apps API 1d + session passthrough 1d + console UI 1d）

---

## Quantum-Safe Access (后量子密码学准备) (2026-07-17 第12小时研究) - Priority: P3 - Status: Watch - Suggested: IAMExpert

**描述**: Cloudflare Access 2025 已支持 quantum-safe access。NIST PQC 标准（ML-KEM/ML-DSA）2024 年最终化。GGID 已有 gmsm 库（可扩展 ML-DSA JOSE 支持，researcher a07e344c 已标记 gap）。Harvest Now Decrypt Later 威胁要求长寿命凭证提前迁移。跟踪标准落地，不立即编码。

---

## SMS OTP Provider Integration (2026-07-17 研究驱动) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: Auth 服务有 SMS OTP 代码验证逻辑（http.go:1598）但只有 DEV 模式日志输出（log.Printf("[DEV] phone OTP..."），没有真正的 SMS 提供商集成。Passwordless 场景需要 SMS 验证码。

**业务价值**: MEDIUM — 补齐 passwordless 方法链 | **实现难度**: Low
- 实现路径：
  1. pkg/notification/sms.go — SMS provider 接口
  2. Twilio provider 实现（或 AWS SNS）
  3. auth/cmd/main.go 注入 provider（env TWILIO_ACCOUNT_SID + TWILIO_AUTH_TOKEN）
  4. 替换 http.go:1598 的 DEV 日志为 provider.Send()
- 参考: docs/research/ciam-2025-trends-gap-analysis.md

---

## Consent Versioning + Withdrawal API (2026-07-17 研究驱动) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: OAuth consent 只有一 checkbox 级别。GDPR/CCPA 要求版本化、可撤回、可审计的 consent 记录。当前 consent analytics 端点存在但没有 consent 版本管理或撤回 API。

**业务价值**: MEDIUM-HIGH — 合规要求 | **实现难度**: Medium
- 实现路径：
  1. consent_records 表（user_id, client_id, scopes, version, granted_at, withdrawn_at）
  2. POST /api/v1/oauth/consent/withdraw — 撤回 consent
  3. GET /api/v1/oauth/consent/history — consent 变更历史
  4. Consent 版本绑定到 scope 定义变更
- 参考: docs/research/ciam-2025-trends-gap-analysis.md

---

## B2B CIAM 增强：客户身份管理 + 自助注册 + 品牌定制 (2026-07-17 第13小时研究) - Priority: P1 - Status: Proposed - Suggested: backend + frontend

**市场背景**: Auth0 Customer Identity Trends Report 2025 — CIAM 市场 $27B（2025），4 大趋势：passwordless 默认化、AI agent 身份、身份欺诈防护、B2B CIAM（组织/租户层级管理）。GGID 当前是 workforce IAM，B2B CIAM 扩展是市场天花板翻倍路径。

**GGID 现状审计（CIAM 能力部分就绪）**：
- ✓ 社交登录（social providers stats handler）
- ✓ OAuth consent screen + consent override
- ✓ 租户品牌（branding_pg.go + client_branding.go）
- ✓ 密码自助重置（forgot + reset）
- ✓ 组织管理（org 服务 LTREE 树）
- ✓ 注册页面（register/page.tsx）
- ✗ 缺：客户自助组织管理（B2B 客户自己注册组织 + 管理成员）
- ✗ 缺：渐进式注册（progressive profiling — 首次只收 email，后续逐步补全）
- ✗ 缺：MFA 强制策略（CIAM 场景：高风险操作才要求 MFA，而非每次登录）
- ✗ 缺：客户身份欺诈防护（step-up based on risk score + bot detection）

**业务价值**: HIGH（从 workforce 扩展到 CIAM = 市场翻倍，$27B TAM）
**实现难度**: Medium
- 完整实现路径（不降级）：
  1. **B2B 自助注册**：POST /api/v1/auth/register-organization — 客户自己创建组织（自动分配 tenant + admin 角色 + 组织根节点）
  2. **渐进式注册**：user.profile_completeness 字段 + 登录后检测缺失字段 + 前端引导补全
  3. **品牌深度定制**：登录/注册/MFA 页面主题色 + Logo + CSS + 自定义域名（CNAME 验证 → tenant_branding 表扩展）
  4. **风险驱动 MFA**：risk_engine.Evaluate > 0.5 → step-up MFA（低风险无缝登录，高风险才要求 MFA）
  5. **身份欺诈防护**：bot detection（已有 middleware）+ velocity check（已有 risk_engine）+ disposable email 阻止 + email verification 强制
  6. **客户旅程分析**：/api/v1/auth/journey-analytics — 注册转化漏斗、MFA 放弃率、社交登录偏好

**兼容性**: 全部基于现有组件扩展；oauth 服务已有品牌+consent+社交+注册基础设施

---

## 可验证凭证 (Verifiable Credentials) / EUDI Wallet 支持 (2026-07-17 第13小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + IAMExpert

**市场背景**: KuppingerCole EIC 2025 CIAM 第一推荐：EUDI Wallet（欧盟数字身份钱包 2026 强制）+ W3C Verifiable Credentials。去中心化身份从概念进入落地期。Auth0 已推出 VC 支持。

**GGID 现状**: SCIM（同步）+ OAuth/OIDC（联邦）+ SAML（企业 SSO）已完整。缺：VC 签发/验证/呈现。

**业务价值**: MEDIUM-HIGH（欧盟市场准入 + 政府/金融场景差异化）
**实现难度**: High（DID 方法选择 + VC JSON-LD 签名 + VP 验证 + Wallet 交互协议）
- 完整路径：DID:web 方法 → VC签发（JSON-LD + Ed25519/SM2 签名）→ VP 验证端点 → statusList 吊销 → Console 签发管理

---

## PKCE Global Mandatory (2026-07-17 研究驱动 OAuth 2.1) - Priority: P1 - Status: Proposed - Suggested: backend

**描述**: OAuth 2.1 (draft-15) 要求所有客户端使用 PKCE。当前 PKCE 在 RequirePKCE=true 或 public client 时强制，但不是全局默认。应改为默认强制，仅允许受信任的 confidential client opt-out。

**业务价值**: HIGH — OAuth 2.1 合规基础要求 | **实现难度**: Low
- 实现路径：
  1. oauth/internal/conf: `RequirePKCE` 默认 true
  2. server.go:331-332 — 移除 RequirePKCE 条件检查，改为全局检查（除非 client 显式标记 PKCE 豁免）
  3. 客户端管理增加 `pkce_optional` 字段（仅 trusted confidential clients）
- 参考: docs/research/oauth21-fapi2-mcp-auth-gap.md

---

## RAR (Rich Authorization Requests) (2026-07-17 研究驱动 FAPI 2.0) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: FAPI 2.0 要求 RAR 支持 — `authorization_details` 参数允许细粒度、类型化的授权请求（如 "transfer 100 EUR to account X"），比传统 scope 更精确。GGID 未实现。

**业务价值**: MEDIUM-HIGH — 金融场景必需 | **实现难度**: Medium
- 实现路径：
  1. OAuth authorize 端点接收 `authorization_details` JSON 数组
  2. 类型验证（每种 type 有独立 schema）
  3. consent screen 展示 RAR details
  4. token 中嵌入 authorization_details claim
  5. 资源服务验证 details
- 参考: docs/research/oauth21-fapi2-mcp-auth-gap.md

---

## Agent Token Revocation + Audit (2026-07-17 研究驱动 MCP Auth) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: GGID 有 AI agent 注册和 token 发放，但缺少：(1) agent token 主动撤销 API (2) MCP server 访问审计（哪个 agent 访问了哪个 server）。Agent 安全面需要这些能力。

**业务价值**: MEDIUM — AI agent 安全闭环 | **实现难度**: Medium
- 实现路径：
  1. POST /api/v1/agents/{id}/revoke — 撤销 agent 所有 token
  2. 审计事件：agent token 发放 + MCP server 访问写入 audit_events
  3. Console 页面：agent 活动审计日志
- 参考: docs/research/oauth21-fapi2-mcp-auth-gap.md

---

## Device Posture as Policy Input (2026-07-17 研究驱动 ZT) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: ZT posture score (当前 65/100) 仅作为信息展示，未接入 Access Broker PDP 决策。NIST SP 800-207 要求设备状态作为访问策略输入（如 score < 50 时拒绝访问）。

**业务价值**: HIGH — ZT 闭环核心 | **实现难度**: Medium
- 实现路径：
  1. Access Broker evaluateAccessPolicy() 增加 device_posture 输入
  2. 策略条件支持 `device_posture >= N` 表达式
  3. posture 低于阈值时返回 deny + 原因
- 参考: docs/research/zero-trust-maturity-compliance-automation.md

---

## Compliance Evidence Export API (2026-07-17 研究驱动) - Priority: P2 - Status: Proposed - Suggested: backend + docs

**描述**: SOC2/ISO27001 审计需要结构化合规证据（用户访问权限、MFA 覆盖率、策略变更历史）。GGID 有全部数据但无导出 API。CCM 工具（Drata/Vanta）需要 API 集成。

**业务价值**: HIGH — 企业合规必需 | **实现难度**: Medium
- 实现路径：
  1. GET /api/v1/audit/compliance/export?framework=soc2&period=Q2-2025
  2. 返回 JSON：access_reviews[], mfa_coverage, policy_changes[], privileged_access_log[]
  3. 支持 SOC2 (Trust Services Criteria) 和 ISO 27001 (Annex A) 框架
  4. 文档：docs/compliance-evidence-guide.md
- 参考: docs/research/zero-trust-maturity-compliance-automation.md

---

## Scheduled IGA Campaigns (2026-07-17 研究驱动) - Priority: P3 - Status: Proposed - Suggested: backend

**描述**: IGA access certification campaigns 需要手动触发。企业合规要求季度自动执行（SOC2 CC6.1, CC6.2）。

**业务价值**: MEDIUM — 合规自动化 | **实现难度**: Low
- 实现路径：
  1. cron/scheduler 在每季度首日自动创建 campaign
  2. 通知所有 certifiers（邮件 + console）
  3. 记录超时未完成审核
- 参考: docs/research/zero-trust-maturity-compliance-automation.md

---

## OWASP MCP Top 10 合规：GGID MCP Server 安全加固 (2026-07-17 第14小时研究) - Priority: P1 - Status: Proposed - Suggested: backend + IAMExpert

**市场背景**: OWASP MCP Top 10 (2025) 发布 — 2026 年 1-2 月研究者提交 30+ MCP CVE（含 CVSS 9.6 mcp-remote，43.7 万下载）。MCP 已成 AI agent 与企业系统的默认连接协议。GGID 有自己的 MCP Server（13 个 LLM 管理 tools），必须对标 OWASP MCP Top 10 加固。

**OWASP MCP Top 10 与 GGID 对照**：

| OWASP # | 风险 | GGID 现状 | 缺口 |
|---------|------|-----------|------|
| MCP01 | Token 管理 | Client 用静态 Bearer token | 需短时 OAuth 2.1 token + scope |
| MCP02 | 权限蔓延 | Tools 无 scope/per-tool authz | 需 per-tool RBAC + scope 过期 |
| MCP03 | Tool Poisoning | 无签名/版本锁定 | 需 tool 签名 + schema 固定 |
| MCP04 | 供应链 | 无 AIBOM | 需 MCP 依赖清单 + provenance |
| MCP05 | 命令注入 | Tools 参数化较好 | 需输入校验审计 |
| MCP06 | Prompt Injection | 无 context 隔离 | 需 instruction quarantine |
| MCP07 | 认证不足 | **无 MCP auth 中间件**（grep 无结果）| 需 OAuth 2.1 + per-server audience |
| MCP08 | 审计缺失 | 无 tool invocation 审计 | 需 immutable audit log + behavioral monitoring |
| MCP09 | Shadow MCP | 无发现机制 | 需 allowlist + discovery |
| MCP10 | Context 泄露 | 无 context 隔离 | 需 per-session context scope |

**最紧急（MCP07）**：GGID MCP Server **无认证中间件**（grep 零结果）— 任何能访问 MCP 端口的人可调用 13 个管理 tool（创建用户/角色/策略）。这与之前 SCIM 直连同型问题。

**业务价值**: HIGH
- MCP 安全是 AI 时代的 OWASP Top 10 等价物 — 2026 合规刚需
- GGID 作为 IAM 平台保护 AI agent 身份 = 差异化叙事
- MCP07 + MCP02 + MCP08 三项是安全底线

**实现难度**: Medium（基于已有 OAuth 2.1 + RBAC + audit 基础设施）
- 完整实现路径（按 OWASP 优先级）：
  1. **MCP07 认证**：MCP Server 加 JWT 验证（复用 gateway JWTAuth）+ per-tool scope 检查（tool 需声明 required_scope）
  2. **MCP08 审计**：每次 tool invocation 写 audit 事件（tool_name/caller/params/result/duration）
  3. **MCP01 Token**：静态 Bearer → OAuth 2.1 token exchange（RFC 8693 — 已在 backlog B-06 中）
  4. **MCP02 Scope**：per-tool RBAC — tool 注册时声明 required_permissions，调用时检查 caller 是否有权限
  5. **MCP10 Context**：per-session context scope + ephemeral memory
  6. **MCP03/04 供应链**：tool 签名 + AIBOM
  7. **MCP09 Discovery**：MCP allowlist + shadow detection
  8. **MCP06 Isolation**：instruction quarantine（retrieved content 标记 untrusted）
  9. **MCP05 Injection**：输入校验审计（已有基础，补审计）

**兼容性**: 复用 OAuth 2.1 (RFC 8693 B-06)、RBAC (policy 服务)、audit publisher (NATS)

---

## AAL/AMR Claims in JWT (2026-07-17 研究驱动 NIST 800-63B-4) - Priority: P1 - Status: Proposed - Suggested: backend

**描述**: NIST SP 800-63B Rev 4 (2025年8月发布) 要求 OIDC token 包含标准化认证保证级别。GGID 有 step-up 认证但不暴露 AAL1/AAL2/AAL3 级别映射到 JWT `acr`/`amr` claims。WebAuthn 单独应满足 AAL2（phishing-resistant）。

**业务价值**: HIGH — 联邦认证和合规场景必需 | **实现难度**: Medium
- 实现路径：
  1. 定义 AAL 级别映射：password=AAL1, WebAuthn=AAL2, WebAuthn+PIN=AAL3
  2. JWT token 添加 `acr` (Assurance Context Reference) 和 `amr` (Authentication Methods References) claims
  3. step-up trigger 时升级 AAL 并签发新 token
  4. 资源服务可根据 AAL 要求拒绝低级别访问
- 参考: docs/research/verifiable-credentials-nist80063-webauthn-hybrid.md

---

## SD-JWT Verifiable Credential Support (2026-07-17 研究驱动 EUDI) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: EU Digital Identity Wallet 2026 强制要求 OpenID4VP 凭证展示。GGID 当前不支持 SD-JWT (Selective Disclosure JWT) 可验证凭证的签发 (OpenID4VCI) 或验证 (OpenID4VP)。RAR handler 提到 "Issue Credential" 但无实际实现。

**业务价值**: HIGH — 欧盟市场合规 | **实现难度**: High
- 实现路径：
  1. SD-JWT 签发：POST /api/v1/oauth/credential-issue — 生成选择性披露凭证
  2. OpenID4VP 验证：POST /api/v1/oauth/credential-verify — 验证第三方凭证
  3. DCQL 查询语言支持 — 按属性查询凭证
  4. Console：凭证管理页面（签发/撤销/查看）
- 参考: docs/research/verifiable-credentials-nist80063-webauthn-hybrid.md

---

## Federation Assurance Level (FAL) (2026-07-17 研究驱动 NIST 800-63-4) - Priority: P3 - Status: Proposed - Suggested: backend

**描述**: NIST 800-63-4 新增 Federation Assurance Level (FAL) 要求 — 联邦认证场景需要 metadata 签名、R&P (Resolution and Policy) profiles、trust frameworks。GGID 有 SAML/OIDC 联邦但未映射到 FAL1/FAL2/FAL3 级别。

**业务价值**: MEDIUM — 政府联邦认证场景 | **实现难度**: Medium
- 参考: docs/research/verifiable-credentials-nist80063-webauthn-hybrid.md

---

## Identity-First Ransomware 防护（MITRE D3FEND 联动） (2026-07-17 第15小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + IAMExpert

**市场背景**: 2025 勒索软件攻击 80%+ 从身份入侵开始（初始访问→凭据窃取→横向移动→部署勒索）。CrowdStrike/Sophos/CyberArk 全部推出 identity-first ransomware 防护。MITRE ATT&CK + D3FEND 框架明确身份层的检测与防御映射。

**GGID 协同优势（全部已实现/在建）**：
- ITDR 检测引擎 → brute_force/credential_stuffing/impossible_travel/baseline_deviation ✓
- CAE 持续评估 → critical detection → session <1s 吊销 ✓
- PAM JIT → 零常驻权限 → 即使 admin 被攻破也无可常驻权限 ✓
- JML → leaver 自动撤销 → 离职者 <3s 失效 ✓
- UEBA → 行为基线偏差 → 低慢攻击检测 ✓
- **缺最后一块拼图**：ransomware playbook（多信号联动自动响应链）

**完整实现路径（不降级）**：
1. **复合检测规则**（ITDR 新类型）：多个 medium detection 在短时间内累积 → 升级为 critical "possible ransomware precursor"
   - 场景：baseline_deviation + new_device + offhours_admin + privilege_escalation 4 信号在 1h 内 → ransomware alert
2. **自动隔离 playbook**（CAE + JML + PAM 联动）：
   - ransomware alert → CAE RevokeUser（全 session 吊销）+ PAM 取消所有 active JIT + identity 锁定账户 + SOC webhook critical
3. **ransomware kill chain 可视化**：console 事件时间线（初始访问→凭据窃取→横向→部署），SOC 可回溯
4. **恢复就绪**：audit chain hash 验证（已有 HMAC chain）→ 确保证据链完整可用于取证

**业务价值**: HIGH（保险合规 + 企业 CISO 必查项）
**实现难度**: Medium（全部基于已建组件，复合规则 + playbook 联动是新增量）
**工作量**: ~3d（复合检测规则 + playbook + 时间线 UI）

---

## Adaptive Authentication Choreography (动态认证编排) (2026-07-17 第16小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + IAMExpert

**市场背景**: Auth0/CyberArk/Stytch 2025 推出 "adaptive auth step-up" — 认证不是固定流程，而是基于实时风险信号的动态编排。B-28 刚交付 AAL/AMR claims（NIST 800-63B），但 AAL 目前是静态的（登录时确定）。Adaptive Choreography 使 AAL **动态升降**：会话期间风险升高（ITDR detection / device untrust / geo anomaly）→ AAL2 自动降为 AAL1（受限访问）或要求 step-up 恢复 AAL2。

**GGID 协同**：B-28 AAL/AMR claims ✓ + ITDR 引擎 ✓ + CAE ✓ + risk_engine ✓ + device_posture ✓ — 全部就绪，差一个 **AAL 状态机 + 动态降级** 层。

**完整实现路径（不降级）**：
1. **AAL 状态机**：每用户会话维护 current_aal（1/2/3）+ expires_at。AAL2 有效期可配（如 4h 后自动降为 AAL1 需 re-MFA）
2. **动态降级触发**：CAE callback 中增加 aal_downgrade 动作 — ITDR medium detection → aal=AAL1（受限）→ 受保护应用 PDP 检查 $security.current_aal
3. **Step-up 恢复**：用户完成 MFA → AAL 恢复 2 → session 继续。无需重新登录
4. **PDP 集成**：$security.current_aal 属性 → per-app policy 可要求 min_aal=2
5. **Console 可视化**：用户当前 AAL 状态 + 降级历史 + step-up 记录

**业务价值**: MEDIUM-HIGH（B-28 AAL claims 的动态化延伸，企业 SSO 核心体验）
**实现难度**: Medium（状态机 + CAE callback + step-up API + PDP 属性）
**工作量**: ~3d

---

## Cross-Domain Identity Federation Hub (联邦身份中枢) (2026-07-17 第17小时研究) - Priority: P2 - Status: Proposed - Suggested: backend + IAMExpert

**市场背景**: 大型企业/政府/高校的身份联邦场景：一个 GGID 实例作为 Hub，连接 N 个 IdP（Okta/Entra/Azure AD/ADFS/飞书/钉钉）+ N 个 SP（内部应用/云服务/SaaS）。用户从任意 IdP 登录 → Hub 转换断言 → 访问任意 SP。Auth0/Ping Identity 的核心商业价值就是 Federation Hub。W3C VC + DID:web 正在将联邦从协议层（SAML/OIDC）升级为信任层（VC verification）。

**GGID 现状（联邦基础设施最完整的一次盘点）**：
- ✓ SAML SP（ACS 断言解析）+ SAML IdP（签发 XMLDSig 签名断言）
- ✓ OIDC Discovery + JWKS + Dynamic Registration
- ✓ OIDC Federation config handler（存在但深度未验证）
- ✓ DID:web resolver + handler（注册/解析/列表/停用）
- ✓ SCIM 2.0 inbound（IdP → GGID provisioning）
- ✓ Social login（Google/GitHub/WeChat providers）
- ✓ LDAP/AD integration（authprovider chain）
- ✓ VC 设计文档（vc-design.md — DID:web + SM2 签发）
- ✗ 缺：**Trust Chain 自动建立**（IdP A 信任 GGID，GGID 信任 SP B → A 的用户自动可访问 B，无需 SP B 单独配置 A）
- ✗ 缺：**Assertion Transformation Engine**（SAML 断言 → OIDC token 转换 / OIDC → VC 转换 / 属性映射 DSL）
- ✗ 缺：**Federation Metadata Aggregate**（统一发布 `/federation/metadata` — 所有 IdP + SP 元数据聚合，新 SP 自动发现）

**完整实现路径（不降级）**：
1. **Trust Chain Registry**：trusted_idps 表（entity_id + protocol[SAML/OIDC/VC] + trust_direction[inbound/outbound/bidirectional] + auto_discovery）
2. **Assertion Transformation Engine**：规则 DSL `source.protocol → target.protocol`（SAML attribute "eduPersonAffiliation" → OIDC claim "role" → SP header "X-WebAuth-Role"）
3. **Federation Metadata Aggregate**：`GET /federation/metadata` 返回所有注册 IdP/SP 元数据聚合 XML/JSON
4. **Discovery Service**：`GET /discovery` — 用户选择来源 IdP（WAYF/IdP picker 页面），或基于 email domain 自动路由
5. **VC Verification Endpoint**：外部系统提交 VP → GGID 验证 → 返回 claims（去中心化联邦）
6. **Console Federation Manager**：IdP/SP 信任关系拓扑图 + 属性映射配置器 + 实时 SSO 测试

**业务价值**: HIGH（GGID 从单点 IAM 升级为联邦中枢 = Auth0/Ping 级产品定位）
**实现难度**: Medium-High（Trust Chain + Assertion Transformation 是核心复杂度）
**工作量**: ~7d（Trust Chain 2d + Transformation 2d + Metadata 1d + Discovery 1d + Console 1d）

---

## Per-Tenant API Rate Limiting + Quota Management (2026-07-17 第18小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**市场背景**: Multi-tenant SaaS 核心能力 — 防止 noisy neighbor（一个租户耗尽 API 配额影响其他租户）。Auth0 每租户可配 per-endpoint RPS/burst/daily quota，超额 429 + Retry-After。Cloudflare/AWS API Gateway 标准功能。

**GGID 现状**: WSConnLimiter（per-tenant WebSocket 连接数）✓ + IPFilter（per-tenant IP 白名单）✓。缺 HTTP API 级别 per-tenant 限流（当前 gateway TenantBucketLimiter 可能是全局配置非 per-tenant 可调）。

**完整实现路径（不降级）**：
1. tenant_rate_limits 表（tenant_id + endpoint_pattern + rps_limit + burst_limit + daily_quota + strategy[token_bucket/sliding_window] + enabled）
2. gateway 中间件：每请求查 tenant_rate_limits（Redis 缓存 30s）→ token bucket / sliding window 计数（Redis INCR + EXPIRE）→ 超额 429 + RateLimit-* headers
3. quota 超额告警（80%/95%/100% webhook + Console 通知）
4. Console：per-tenant 限流配置 + 实时用量 dashboard + top endpoints by RPS
5. 默认策略：免费租户 100 RPM，付费 1000 RPM，企业无限制（可配）

**业务价值**: MEDIUM-HIGH（multi-tenant SaaS 刚需，客户差异化定价依据）
**实现难度**: Medium（Redis 计数器 + DB 配置 + gateway 中间件）
**工作量**: ~3d

---

## Identity-Based Data Loss Prevention (DLP) (2026-07-17 第19小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**市场背景**: DLP 从网络层（防火墙/邮件网关）迁移到身份层（谁在什么时间导出了什么数据）。Forrester 2025 将 Identity-Centric DLP 列为增长最快的安全细分。核心：基于身份+上下文（device posture/ITDR/data classification）的实时数据外发控制。

**GGID 现状审计**：dlp_policies_handler.go 存在但仅 45 行（stub）。audit export handler 有。data_classifications 有。全部安全信号（device/ITDR/risk/classification）已就绪。

**完整实现路径（不降级）**：
1. dlp_policies 表（tenant_id + name + trigger[export/api_access/bulk_query] + conditions[AND data_classification=core AND user_role=viewer] + action[block/require_approval/mask/log]）
2. gateway 中间件：导出/bulk 端点 → DLP 评估（who + what data + context → policy match → action）
3. 审批流：require_approval → DM manager → approve/reject
4. 实时告警：block + webhook SOC + ITDR 检测（异常导出模式）
5. Console：DLP 策略配置 + 导出审计日志 + 实时拦截事件流

**端点清单**：
- GET/POST /api/v1/identity/dlp/policies — 策略 CRUD
- GET /api/v1/audit/dlp/events — 拦截事件日志
- POST /api/v1/identity/dlp/policies/{id}/test — 策略模拟

**验收 curl**：
```bash
curl -X POST /api/v1/identity/dlp/policies -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Block core data export by non-admin","trigger":"export","conditions":{"and":[{"data_classification":"core"},{"user_role":{"$ne":"admin"}}]},"action":"block"}'
curl /api/v1/audit/dlp/events?severity=blocked
```

**业务价值**: MEDIUM-HIGH（Identity-DLP 差异化，与数据安全法+ITDR+CAE 协同）
**实现难度**: Medium
**工作量**: ~4d

---

## Zero-Trust Secret Brokering (动态凭据分发) (2026-07-18 第20小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**市场背景**: HashiCorp Vault Identity Brokering + CyberArk Dynamic Secrets 是 ZTNA 数据面的核心 — JIT 授权不仅给"访问权"，还给"临时凭据"（动态 DB 密码/TLS 证书/SSH key/Cloud STS token）。Vault 作为身份代理：验证身份 → 签发短 TTL 凭据 → 自动轮换 → 到期吊销。2025 趋势是将 secret broker 从独立系统整合进 IAM 平台。

**GGID 现状**：
- JIT user provisioning engine ✓（B-31，自动创建/更新用户）
- PAM JIT grant 系统 ✓（零常驻权限 + 时间限制授权）
- SCIM 2.0 provisioning ✓
- Gateway PDP evaluatePolicy ✓
- **缺**：动态凭据分发 — JIT 授权后不产生临时凭据（只给权限，不给 credential）

**完整实现路径（不降级）**：
1. secret_broker_targets 表（tenant_id + target_name + target_type[db/ssh/cloud/api_key] + connection_config JSONB + default_ttl + rotation_policy）
2. Broker 引擎：PAM JIT grant 批准 → secret_broker 生成短 TTL 凭据 → 返回给请求者
   - DB: 动态生成用户+密码（CREATE USER 'jit_xxx' WITH PASSWORD '...' VALID UNTIL '...')
   - SSH: 短 TTL 证书签名（复用 vc-design.md 的 Ed25519 签名）
   - Cloud: 假定角色（STS AssumeRole）token 分发
   - API Key: 随机生成 + 过期自动删除
3. 自动清理：TTL 到期 → revoke 凭据（DROP USER / 证书吊销 / token 失效）
4. 审计链：每次凭据签发 + 吊销写入 audit_events（HMAC chain）
5. Console：目标配置 + 实时活动凭据列表 + TTL 倒计时

**端点清单**：
- GET/POST/DELETE /api/v1/identity/secrets/targets — 目标 CRUD
- POST /api/v1/identity/secrets/broker — 签发动态凭据（关联 JIT grant）
- GET /api/v1/identity/secrets/active — 当前活动凭据列表
- DELETE /api/v1/identity/secrets/{id} — 手动吊销

**验收 curl**：
```bash
# 注册 DB 目标
curl -X POST /api/v1/identity/secrets/targets -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"prod-postgres","type":"db","connection":{"host":"db.internal","port":5432},"default_ttl":"15m"}'

# JIT 授权后签发动态凭据
curl -X POST /api/v1/identity/secrets/broker -H "Authorization: Bearer $TOKEN" \
  -d '{"target":"prod-postgres","jit_grant_id":"jgr_abc123","ttl":"15m"}'

# 查看活动凭据
curl /api/v1/identity/secrets/active -H "Authorization: Bearer $TOKEN"

# 手动吊销
curl -X DELETE /api/v1/identity/secrets/sec_xyz789 -H "Authorization: Bearer $TOKEN"
```

**业务价值**: HIGH（ZTNA 最后一块 — 零信任 = 零常驻权限 + 零常驻凭据）
**实现难度**: Medium-High（DB 动态用户 + SSH 证书签名 + Cloud STS）
**工作量**: ~5d

---

## DPoP Proof Enforcement at Token Endpoint (2026-07-18 研究驱动 OAuth 2.1) - Priority: P1 - Status: Proposed - Suggested: backend

**描述**: OAuth 2.1 (draft-15) 要求 sender-constrained tokens。GGID 有 DPoP PG store + config handler，但 token endpoint 不验证 DPoP proof header。token 签发时不绑定 client 的 public key。

**业务价值**: HIGH — OAuth 2.1 合规 | **实现难度**: Medium
- 实现路径：
  1. Token endpoint: 解析 `DPoP` header（JWT proof）
  2. 验证 proof signature + htm/htu/iat/jti claims
  3. 绑定 access_token `cnf.jkt` claim（thumbprint of client public key）
  4. Resource server: 验证 `cnf` claim matches DPoP proof
  5. 可选 per-client 强制（RequireDPoP config 已存在）
- 参考: RFC 9449, docs/research/oauth21-mcp-agent-auth-pqc-migration.md

---

## AI Agent Per-Action Authorization + Delegation Chains (2026-07-18 研究驱动 MCP 2026) - Priority: P1 - Status: Proposed - Suggested: backend

**描述**: MCP 2026-07-28 spec 要求 per-action authorization（每个工具调用需授权）和 delegation chains（agent 代表用户操作的可追溯委托链）。GGID 有 agent registration 但无授权层。

**业务价值**: HIGH — AI agent 安全核心 | **实现难度**: High
- 实现路径：
  1. Agent capability scoping: 每个 agent 注册时声明可调用的工具/资源列表
  2. Per-action approval: 工具调用前检查 agent 是否有权限
  3. Delegation chain: JWT 包含 `act` claim 链（user → agent → sub-agent）
  4. On-behalf-of (OBO) flow: agent 用 user token 换取 scoped agent token
  5. Audit: 记录每个 agent action + delegation chain
- 参考: MCP 2026 spec, docs/research/oauth21-mcp-agent-auth-pqc-migration.md

---

## Fix Misleading PQC Label (2026-07-18 研究驱动 NIST PQC) - Priority: P0 - Status: Proposed - Suggested: backend

**描述**: `pqc_signature_handler.go` 使用 ed25519（经典密码学），但命名为 "PQC"（后量子密码学）。这具有误导性——ed25519 不抗量子攻击。需要：(1) 重命名为 `ed25519_signature_handler.go`，或 (2) 实现真正的 ML-DSA (FIPS 204)。

**业务价值**: HIGH — 安全准确性 | **实现难度**: Low (rename) / High (implement)
- 选项 A: 重命名 + 更新所有引用（Low）
- 选项 B: 集成 Go `crypto/mlkem` + `crypto/mldsa`（Go 1.24+）实现真正 PQC（High）
- 参考: FIPS 203/204/205, docs/research/oauth21-mcp-agent-auth-pqc-migration.md

---

## ML-KEM/ML-DSA Crypto Package (2026-07-18 研究驱动 NIST PQC) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: 实现 FIPS 203 (ML-KEM) 和 FIPS 204 (ML-DSA) 后量子算法包，用于 JWT 签名、TLS key exchange、audit log signatures。Go 1.24+ 提供 `crypto/mlkem` 和 `crypto/mldsa` 标准库支持。

**业务价值**: MEDIUM — 未来-proofing，2035 联邦强制 | **实现难度**: Medium
- 参考: FIPS 203/204, docs/research/oauth21-mcp-agent-auth-pqc-migration.md

---

## ROPC Grant Deprecation (2026-07-18 研究驱动 OAuth 2.1) - Priority: P2 - Status: Proposed - Suggested: backend

**描述**: OAuth 2.1 移除 Resource Owner Password Credentials (ROPC) grant。GGID 应审计并弃用 `grant_type=password` 接受。迁移用户到 authorization_code + PKCE 或 device_code flow。

**业务价值**: MEDIUM — OAuth 2.1 合规 | **实现难度**: Low
- 参考: draft-ietf-oauth-v2-1-15, docs/research/oauth21-mcp-agent-auth-pqc-migration.md

---

## Threat Intelligence Integration Hub (ITDR 外部情报集成) (2026-07-18 第21小时研究) - Priority: P2 - Status: Proposed - Suggested: backend

**市场背景**: CrowdStrike/Splunk/IBM QRadar 2025 核心能力 — ITDR 检测引擎 + 外部威胁情报源（AlienVault OTX/AbuseIPDB/ HaveIBeenPwned/MISP）联动。GGID ITDR 引擎有 6+ 内部检测规则（brute_force/credential_stuffing/impossible_travel/baseline_deviation），但缺外部情报源接入。增加已知恶意 IP/泄露凭据/代理检测 → 提升检测准确率 + 降低误报。

**GGID 现状**：threat_intel_feed_handler.go 存在但仅 69 行（stub）。ITDR 引擎不查询外部情报。CAE risk engine 不消费 threat feed。

**完整实现路径（不降级）**：
1. threat_intel_sources 表（tenant_id + name + type[ip_hash/credential/domain] + api_endpoint + api_key_ref + poll_interval + enabled）
2. threat_indicators 表（source_id + indicator_type + value + severity + first_seen + last_seen + confidence）
3. 情报采集器：定时 poll 外部 API（OTX/AbuseIPDB/HIBP）→ 解析 → 存 threat_indicators → TTL 过期清理
4. ITDR 联动：每次 login/access → 查 threat_indicators（IP/email/hash）→ hit → 提升 risk score + ITDR detection
5. CAE 联动：threat indicator hit → risk_engine 提高 → 可能触发 step-up MFA 或 session revoke
6. Console：情报源配置 + indicators 统计 + hit 热力图

**端点清单**：
- GET/POST /api/v1/audit/threat-intel/sources — 情报源 CRUD
- GET /api/v1/audit/threat-intel/indicators — indicators 查询
- POST /api/v1/audit/threat-intel/check — 手动检查（IP/email → threat match）
- GET /api/v1/audit/threat-intel/stats — 统计（source count / indicator count / hit count 24h）

**验收 curl**：
```bash
# 注册情报源
curl -X POST /api/v1/audit/threat-intel/sources -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"AbuseIPDB","type":"ip","api_endpoint":"https://api.abuseipdb.com/api/v2/check","poll_interval":"1h"}'

# 检查 IP
curl -X POST /api/v1/audit/threat-intel/check -H "Authorization: Bearer $TOKEN" \
  -d '{"indicator":"192.168.1.100","type":"ip"}'

# 统计
curl /api/v1/audit/threat-intel/stats -H "Authorization: Bearer $TOKEN"
```

**业务价值**: HIGH（ITDR 从纯内部规则 → 内外联动，检测准确率显著提升）
**实现难度**: Medium（外部 API 适配 + 定时采集 + ITDR 注入）
**工作量**: ~4d
