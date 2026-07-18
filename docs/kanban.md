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
| KB-111 | **GraphQL SDL schema + gqlgen setup** | backend | P0 | graphql-api | 3d |
| KB-112 | **Query resolvers + dataloaders** (nested resolution, batch) | backend | P0 | graphql-api | 5d |
| KB-113 | **Field-level @auth directive** (PDP integration) | backend | P0 | graphql-api | 3d |
| KB-114 | **Query complexity analysis** (depth + cost limits) | backend | P0 | graphql-api | 2d |
| KB-115 | **Mutation resolvers** (CRUD via GraphQL) | backend | P1 | graphql-api | 4d |
| KB-116 | **Persisted queries + GraphQL query log** | backend | P1 | graphql-api | 3d |
| KB-117 | **Console GraphQL playground + migration** | frontend | P2 | graphql-api | 5d |
| KB-118 | **API key DB-backed storage + hashing** (replace in-memory) | backend | P0 | api-key-lifecycle | 2d |
| KB-119 | **API key repository + CRUD handler** (replace in-memory handler) | backend | P0 | api-key-lifecycle | 2d |
| KB-120 | **Gateway API key validator** (DB-backed + Redis cache + expiry) | backend | P0 | api-key-lifecycle | 3d |
| KB-121 | **Key rotation with grace period** (dual-key + auto-revoke) | backend | P1 | api-key-lifecycle | 3d |
| KB-122 | **Scope enforcement + per-key rate limiting** (gateway) | backend | P1 | api-key-lifecycle | 2d |
| KB-123 | **IP allow-list + usage tracking** | backend | P1 | api-key-lifecycle | 2d |
| KB-124 | **Console key management + rotation UI** | frontend | P2 | api-key-lifecycle | 3d |
| KB-125 | **Recovery DB schema + repository** (replace in-memory) | backend | P0 | credential-recovery | 2d |
| KB-126 | **Recovery API + multi-factor verification pipeline** | backend | P0 | credential-recovery | 4d |
| KB-127 | **Temporary Access Pass (TAP)** (15-min single-use) | backend | P0 | credential-recovery | 2d |
| KB-128 | **Risk-based graduated delay + admin approval** | backend | P1 | credential-recovery | 3d |
| KB-129 | **Recovery rate limiting + notifications + audit** | backend | P1 | credential-recovery | 2d |
| KB-130 | **Console self-service + admin recovery UI** | frontend | P2 | credential-recovery | 3d |
| KB-131 | **Consent DB schema + repository** (replace mock + in-memory) | backend | P0 | consent-platform | 2d |
| KB-132 | **Consent CRUD API + withdrawal cascade** | backend | P0 | consent-platform | 4d |
| KB-133 | **Replace mock consent registry + in-memory OAuth store** | backend | P0 | consent-platform | 3d |
| KB-134 | **DSR workflow** (access/deletion/portability) | backend | P1 | consent-platform | 4d |
| KB-135 | **GPC detection + policy versioning + re-consent** | backend | P1 | consent-platform | 3d |
| KB-136 | **Console preference center + DSR admin + cookie banner** | frontend | P2 | consent-platform | 4d |
| KB-137 | **Unified Risk Engine (URE)** — composite scorer + signal collectors | backend | P0 | adaptive-auth | 5d |
| KB-138 | **Risk assessment + config API** (replace 3 in-memory engines) | backend | P0 | adaptive-auth | 4d |
| KB-139 | **Decision policy + step-up integration** (risk → action) | backend | P1 | adaptive-auth | 3d |
| KB-140 | **Replace 12 hardcoded risk handlers** (real URE data) | backend | P1 | adaptive-auth | 3d |
| KB-141 | **Behavioral baselines + risk analytics** | backend | P1 | adaptive-auth | 3d |
| KB-142 | **Risk dashboard + continuous evaluation middleware** | backend+frontend | P2 | adaptive-auth | 4d |
| KB-143 | **Agent registry + DB-backed consent** (replace in-memory) | backend | P0 | ai-agent-identity | 4d |
| KB-144 | **Token exchange integration + workload attestation** | backend | P0 | ai-agent-identity | 4d |
| KB-145 | **Multi-agent delegation chains + cycle detection** | backend | P1 | ai-agent-identity | 4d |
| KB-146 | **Per-agent rate limiting + dual-attribution audit** | backend | P1 | ai-agent-identity | 3d |
| KB-147 | **Agent behavioral anomaly detection** | backend | P2 | ai-agent-identity | 3d |
| KB-148 | **Console agent management UI** | frontend | P2 | ai-agent-identity | 4d |
| KB-149 | **DB-backed VC/DID core** (replace in-memory VCIssuer + DIDResolver) | backend | P0 | decentralized-vc | 4d |
| KB-150 | **Credential schema registry + StatusList2021 revocation** | backend | P0 | decentralized-vc | 3d |
| KB-151 | **Asymmetric SD-JWT** (ES256/EdDSA, replace HMAC) | backend | P0 | decentralized-vc | 2d |
| KB-152 | **OID4VCI credential issuance flow** | backend | P1 | decentralized-vc | 4d |
| KB-153 | **OID4VP presentation verification** | backend | P1 | decentralized-vc | 3d |
| KB-154 | **Console credential manager + did:ebsi** | frontend+backend | P2 | decentralized-vc | 4d |
| KB-155 | **Continuous evaluation middleware (CAE)** — per-request risk re-score | backend | P0 | zero-trust | 3d |
| KB-156 | **MDM integration** (Intune/Jamf connector) + device cert auth | backend | P0 | zero-trust | 5d |
| KB-157 | **CMK / KMS integration** (AWS KMS / Vault per-tenant keys) | backend | P0 | zero-trust | 5d |
| KB-158 | **DLP at egress** (response scanning middleware) | backend | P0 | zero-trust | 4d |
| KB-159 | **Service-to-service mTLS enforcement** + network policy engine | backend | P1 | zero-trust | 5d |
| KB-160 | **Access review / certification** (periodic recertification campaigns) | backend | P1 | zero-trust | 5d |
| KB-161 | **Unified PDP** — combine RBAC+ABAC+ReBAC+risk in one Authorize() RPC | backend | P0 | continuous-authz | 4d |
| KB-162 | **Gateway PEP middleware** — per-request authorization on every API call | backend | P0 | continuous-authz | 3d |
| KB-163 | **Redis decision cache** (5s TTL + invalidation on role/policy change) | backend | P0 | continuous-authz | 3d |
| KB-164 | **DB-backed decision audit** (replace in-memory log + async via NATS) | backend | P0 | continuous-authz | 3d |
| KB-165 | **Risk overlay in evaluator** (risk score upgrades decision to step_up) | backend | P1 | continuous-authz | 3d |
| KB-166 | **Decision audit query API** (replace hardcoded decision stats) | backend | P1 | continuous-authz | 2d |
| KB-167 | **MDM connector framework** (Intune + Jamf API integration) | backend | P0 | mdm-integration | 5d |
| KB-168 | **Compliance policy engine** (configurable rules + scoring) | backend | P0 | mdm-integration | 4d |
| KB-169 | **Compliance webhooks + CAE integration** (real-time revocation) | backend | P1 | mdm-integration | 4d |
| KB-170 | **SCEP certificate provisioning** (internal CA + device certs) | backend | P1 | mdm-integration | 3d |
| KB-171 | **Hardware attestation verification** (TPM + Apple + Android) | backend | P2 | mdm-integration | 5d |
| KB-172 | **Fleet compliance dashboard** (Console) | frontend | P2 | mdm-integration | 3d |
| KB-173 | **DataKeyProvider + envelope encryption** (DEK/KEK per-tenant) | backend | P0 | cmk-kms | 4d |
| KB-174 | **Per-tenant CMK management API** (AWS KMS + Vault + Azure) | backend | P0 | cmk-kms | 4d |
| KB-175 | **BYOK key import + automated rotation** | backend | P1 | cmk-kms | 3d |
| KB-176 | **SM4 data encryption** (China GM compliance) | backend | P1 | cmk-kms | 3d |
| KB-177 | **Field-level encryption** (PII columns AES-256-GCM) | backend | P1 | cmk-kms | 3d |
| KB-178 | **HYOK + key audit + Console UI** | backend+frontend | P2 | cmk-kms | 4d |
| KB-179 | **DLP egress middleware + PII detection + redaction** (gateway) | backend | P0 | dlp-egress | 4d |
| KB-180 | **Egress redaction engine** (mask/partial/tokenize/redact/hash) | backend | P0 | dlp-egress | 3d |
| KB-181 | **Egress policy DSL + classification auto-mask** | backend | P1 | dlp-egress | 3d |
| KB-182 | **Replace auth DLP hardcoded mock + egress analytics** | backend | P1 | dlp-egress | 2d |
| KB-183 | **Istio mTLS STRICT** (auto sidecar injection on k3s) | devops | P0 | service-mesh | 3d |
| KB-184 | **Microsegmentation zone policies** (default-deny + explicit allows) | devops | P0 | service-mesh | 3d |
| KB-185 | **ExtAuthz adapter** (Envoy → GGID PDP for east-west authz) | backend | P1 | service-mesh | 3d |
| KB-186 | **SPIFFE ID registry + mesh policy management API** | backend | P1 | service-mesh | 3d |
| KB-187 | **Distributed tracing + mesh observability** (Jaeger + Prometheus) | devops | P2 | service-mesh | 2d |
| KB-188 | **Hierarchical rate limiting** (per-user/key/IP/endpoint + burst/sustained) | backend | P0 | gateway-hardening | 4d |
| KB-189 | **Circuit breaker in router** (per-backend, auto-trip + half-open) | backend | P0 | gateway-hardening | 3d |
| KB-190 | **Request validation + payload sanitization** (JSON schema, SQLi/XSS) | backend | P1 | gateway-hardening | 3d |
| KB-191 | **Enhanced observability** (P50/P95/P99 histograms + hardening dashboard) | backend | P1 | gateway-hardening | 2d |
| KB-192 | **8 new ITDR detection rules** (consent phishing, MFA fatigue, token theft, session hijack, mass creation, federation anomaly, MFA bypass, mass export) | backend | P0 | itdr-maturity | 4d |
| KB-193 | **Attack simulation API** (purple team, 7 scenarios, coverage validation) | backend | P1 | itdr-maturity | 4d |
| KB-194 | **SOAR webhook + auto-response** (session revoke + account lock on critical) | backend | P1 | itdr-maturity | 3d |
| KB-195 | **ML-based UEBA** (isolation forest, per-user models, anomaly scoring) | backend | P2 | itdr-maturity | 4d |
| KB-196 | **MITRE coverage dashboard + detection-as-code** (YAML versioned rules) | backend | P2 | itdr-maturity | 3d |
| KB-197 | **Asymmetric SD-JWT + erasure pipeline** (EdDSA + cascade deletion) | backend | P0 | pets-privacy | 4d |
| KB-198 | **BBS+ signatures** (unlinkable selective disclosure) | backend | P1 | pets-privacy | 4d |
| KB-199 | **Differential privacy + crypto-shredding** (analytics noise + DEK deletion) | backend | P1 | pets-privacy | 3d |
| KB-200 | **Pseudonymization vault + data minimization** (CMK tokenization) | backend | P2 | pets-privacy | 4d |
| KB-201 | **HR connector framework** (Workday/BambooHR + JML webhooks) | backend | P0 | lifecycle-automation | 5d |
| KB-202 | **Dormant account detection + ghost reconciliation** (cron + auto-stage) | backend | P0 | lifecycle-automation | 3d |
| KB-203 | **Approval workflow engine** (multi-step manager/security approval) | backend | P1 | lifecycle-automation | 4d |
| KB-204 | **SCIM 2.0 outbound + bulk operations** (push to downstream apps) | backend | P1 | lifecycle-automation | 3d |
| KB-205 | **Evidence collection engine + CCM** (5 collectors + continuous monitoring) | backend | P0 | compliance-automation | 5d |
| KB-206 | **Replace 12 hardcoded compliance handlers** (real CCM data) | backend | P0 | compliance-automation | 3d |
| KB-207 | **Framework mapping + PDF reports** (SOC2/ISO/NIST + evidence-attached) | backend | P1 | compliance-automation | 3d |
| KB-208 | **Trust center + gap detection** (public compliance status) | backend | P1 | compliance-automation | 2d |
| KB-209 | **Replace session hijack/anomaly/inspect hardcoded handlers** (real detection) | backend | P0 | session-security | 3d |
| KB-210 | **Device fingerprint session binding + CAE risk re-evaluation** | backend | P1 | session-security | 3d |
| KB-211 | **DB-backed session store + DPoP binding** (audit trail + PoP) | backend | P1 | session-security | 3d |
| KB-212 | **PostgreSQL RLS migration** (30+ tables ENABLE + FORCE + BYPASSRLS role) | backend | P0 | rls-implementation | 3d |
| KB-213 | **RLSPool wrapper + test suite** (WithTenant + cross-tenant isolation tests) | backend | P0 | rls-implementation | 3d |
| KB-214 | **Enable FORCE RLS + performance verification** (all tables enforced) | backend | P1 | rls-implementation | 2d |
| KB-215 | **Expand Python SDK** (auth + pagination + errors + all modules) | backend | P0 | sdk-parity | 5d |
| KB-216 | **Token manager + DPoP in all SDKs** (auto-refresh + PoP) | backend | P0 | sdk-parity | 4d |
| KB-217 | **OpenAPI spec + package publishing CI/CD** (npm/PyPI/Maven) | backend | P1 | sdk-parity | 4d |
| KB-218 | **Vault + cert-manager + security headers** (P0 production blockers) | devops | P0 | prod-hardening | 4d |
| KB-219 | **Graceful shutdown + PG backup + Redis config + alerting** | devops | P0 | prod-hardening | 4d |
| KB-220 | **OpenTelemetry tracing + Grafana + load testing baseline** | devops | P1 | prod-hardening | 4d |
| KB-221 | **Pre-commit hooks + parallel CI + branch protection + lint** | devops | P0 | cicd-hardening | 3d |
| KB-222 | **Container scanning + multi-stage Dockerfile + migration CI** | devops | P0 | cicd-hardening | 3d |
| KB-223 | **ArgoCD GitOps for k3s** (auto-deploy from main) | devops | P1 | cicd-hardening | 3d |
| KB-224 | **PG backup + WAL archiving + PITR** (encrypted to S3) | devops | P0 | dr-backup | 3d |
| KB-225 | **Redis persistence + off-site backup + restore testing** | devops | P0 | dr-backup | 3d |
| KB-226 | **PG streaming replication + DR runbook** (automated failover) | devops | P1 | dr-backup | 4d |
| KB-227 | **k6 load test suite** (5 flows: login/token/user/policy/risk) | devops | P0 | load-testing | 4d |
| KB-228 | **Baseline metrics + capacity model** (1K/10K/100K users → infra) | devops | P0 | load-testing | 2d |
| KB-229 | **Soak + spike + DB index audit** (24h + 20x burst + tuning) | devops | P1 | load-testing | 4d |
| KB-230 | **PG streaming replication + failover** (active-passive) | devops | P0 | multi-region | 4d |
| KB-231 | **GeoDNS + region discovery API** (geo-routing) | devops | P1 | multi-region | 2d |
| KB-232 | **Bidirectional logical replication + LWW conflict resolution** (active-active) | devops | P2 | multi-region | 5d |
| KB-233 | **Per-tenant data residency** (publication filters for GDPR/PIPL) | backend | P2 | multi-region | 3d |
| KB-234 | **swag annotations on all 786+ handlers** (per-service swag init) | backend | P0 | openapi-spec | 5d |
| KB-235 | **Gateway aggregator + Swagger UI** (merged spec + /docs) | backend | P0 | openapi-spec | 2d |
| KB-236 | **Contract tests + SDK auto-generation** (openapi-generator pipeline) | backend | P1 | openapi-spec | 3d |
| KB-237 | **Registration + password reset + profile self-edit** | backend | P0 | self-service | 4d |
| KB-238 | **Device/session/MFA self-service** (list + revoke + enroll) | backend | P0 | self-service | 3d |
| KB-239 | **Account linking + privacy center** (GDPR export/delete) | backend | P1 | self-service | 4d |
| KB-240 | **README.md + CONTRIBUTING.md + LICENSE** (professional GitHub presence) | docs | P0 | readme-guide | 2d |
| KB-241 | **Screenshots + repo badges + Mermaid architecture** | docs+frontend | P1 | readme-guide | 1d |
| KB-242 | **Conventional commits + git-cliff + GitHub Releases** (auto-changelog) | devops | P0 | changelog-release | 2d |
| KB-243 | **Deprecation headers + migration guide framework** (RFC 8594) | backend+docs | P1 | changelog-release | 2d |
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

| ID | Title | Owner | Status |
|----|-------|-------|--------|
| Batch 5c | oauth 15 in-memory maps → PG | backend | in progress |
| KB-179/180 | DLP Egress Control + PII Redaction backend | IAMExpert | in progress |
| — | Console rebuild + deploy (all i18n + crash fixes + new pages) | techwriter | in progress |
| — | Service Mesh & Microsegmentation research | researcher | in progress |
| — | F-50/F-51/F-52 console pages shipped, next TBD | frontend | active |
| — | Batch 5b verification done, next TBD | UIAutomationExpert | active |

## Batch 5: In-Memory Map Migration Progress

| Service | Maps | Status | Verified |
|---------|------|--------|----------|
| auth (5a) | 15 | DONE (commit a540a98d) | Pending |
| policy (5b) | 10 | DONE (commit 946562fb) | PASS by UIAutomationExpert |
| identity (5b) | 6 | DONE (commit 4f7e5378) | PASS by UIAutomationExpert |
| oauth (5c) | 15 | In progress | — |
| audit | 5 | Pending | — |
| org | 2 | Pending | — |
| **Total** | **53** | **31 done, 15 in progress, 7 pending** | |

## Done (Recent)

| Item | Detail | Verified |
|------|--------|----------|
| Batch 5a auth maps | commit a540a98d — 15 auth handlers → PG. OTP/passkey/biometric/credential vault + 11 more. 25+ JSONB tables | Pending verification |
| Batch 5b policy maps | commit 946562fb — 10 policy maps → policyMapRepo | PASS by UIAutomationExpert |
| Batch 5b identity maps | commit 4f7e5378 — 6 identity maps → identityPolicyMapRepo | PASS by UIAutomationExpert |
| F-44 risk-score crash fix | commit 18591819 — useState → useEffect (arch fix) | Pushed |
| P0 useState crash fix | commit 5bfad424 — 44 pages fixed | grep verified 0 remaining |
| P0 i18n (9 pages) | commit 39d920df — 35 new keys | Pushed |
| P1 i18n (13 pages) | commit 3865ad6a — 112 new keys | Pushed |
| Sidebar + login creds | commit 4d562985 — 6 sidebar entries + creds fix | Verified |
| F-50 Consent Management | commit 4926388a — 4 tabs | Pushed |
| F-51 NIS2/CRA Dashboard | commit a6d0949a — 5 tabs | Pushed |
| F-52 Federation Wizard | commit 6dfd4fa6 — 4 tabs + 3-step wizard | Pushed |
| KB-131b OAuth consent PG | commit 71f8b7fc — pgConsentStore | Pushed |
| B-37 Threat Intel Hub | commit c83a4ec9 | PASS by arch |
| B-37b ITDR/CAE integration | commit 2be3e84b | PASS by arch |
| KB-131/132 Consent backend | commit a8e57c93 | PASS by arch |
| URE Design Doc | commit 6576be96 — 503 lines | Pushed |
| Research: 24 rounds | Consent, Adaptive Auth, AI Agent, DID/VC, ZTMM, PDP, MDM, CMK/KMS, DLP Egress — 182 backlog items | Pushed |
| AI Agent Identity research | commit e8fb71a0 — 33KB + 2108-line analysis, 6 backlog items | Pushed |
| F-47 Passkey Health Dashboard | commit fb1aadb7 — 4 tabs (KB-024) | Pushed |
| F-48 ReBAC Console UI | commit ffab77d2 — schema editor + playground + tuples + graph (KB-029) | Pushed |
| Console deploy F-44/F-45 | /security/risk-score + /security/threat-intel live (200 OK) | Verified by techwriter |
| i18n audit report | 9 P0 files + 13 P1 files + P2 missing keys + P3 Chinese-in-EN bugs | Reported by UIAutomationExpert |
| Consent Management research | commit c5452b5b — 38KB doc + 6 backlog items (KB-131 to KB-136) | Pushed |

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
| F-44 | Zero-Trust Secret Broker (5 tabs, Vault-level, JIT linkage) | frontend | 7f4c1855 |
| F-45 | Threat Intelligence Hub (5 tabs, CrowdStrike-level, ITDR correlation) | frontend | 52fe00bd |
| F-46 | API Key Lifecycle Management (4 tabs, create/rotate/revoke/scopes) | frontend | 34997441 |
| F-47 | Passkey Health Dashboard (4 tabs, health checks, enforcement policy) | frontend | fb1aadb7 |
| F-48 | ReBAC Console UI — Schema Editor, Playground, Tuple Store (KB-029) | frontend | ffab77d2 |

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
| GraphQL API Layer for Identity Queries | DONE | docs/research/graphql-api-layer.md |
| API Key Lifecycle Management | DONE | docs/research/api-key-lifecycle.md |
| Credential Recovery Workflow Engine | DONE | docs/research/credential-recovery-workflow.md |
| Consent Management Platform (GDPR/CCPA) | DONE | docs/research/consent-management-platform.md |
| Risk-Based Adaptive Authentication Engine | DONE | docs/research/risk-adaptive-auth-engine.md |
| AI Agent Identity & Delegated Access | DONE | docs/research/ai-agent-identity.md |
| Decentralized Identity & Verifiable Credentials (W3C DID/VC) | DONE | docs/research/decentralized-identity-vc.md |
| Zero Trust Maturity Assessment & Gap Analysis | DONE | docs/research/zero-trust-maturity-assessment.md |
| Continuous Authorization & PDP | DONE | docs/research/continuous-authorization-pdp.md |
| MDM Integration (Intune/Jamf/Android) | DONE | docs/research/mdm-integration.md |
| Customer-Managed Keys (CMK) & KMS Integration | DONE | docs/research/customer-managed-keys-kms.md |
| DLP Egress Control & PII Redaction | DONE | docs/research/dlp-egress-pii-redaction.md |
| Service Mesh & Microsegmentation | DONE | docs/research/service-mesh-microsegmentation.md |
| API Gateway Hardening & Rate Limiting | DONE | docs/research/api-gateway-hardening.md |
| ITDR Maturity & MITRE ATT&CK Mapping | DONE | docs/research/itdr-maturity-mitre-attack.md |
| Privacy-Enhancing Technologies (PETs) | DONE | docs/research/pets-privacy-by-design.md |
| Identity Lifecycle Automation & HR-Driven Provisioning | DONE | docs/research/identity-lifecycle-automation.md |
| Compliance Automation & Audit Evidence | DONE | docs/research/compliance-automation-audit-evidence.md |
| Audit Tamper-Evidence (Hash Chain Verification) | DONE | Already implemented (hash_chain.go HMAC-SHA256 + tamper detection + API) |
| Session Management & Token Lifecycle Security | DONE | docs/research/session-token-lifecycle-security.md |
| Multi-Tenant Architecture & Data Isolation | DONE | docs/research/multi-tenant-isolation.md |
| PostgreSQL RLS Implementation Guide | DONE | docs/research/postgresql-rls-implementation.md |
| SDK Parity & Developer Experience | DONE | docs/research/sdk-parity-developer-experience.md |
| Production Hardening & Security Checklist | DONE | docs/research/production-hardening-checklist.md |
| CI/CD Pipeline & GitOps Hardening | DONE | docs/research/cicd-pipeline-hardening.md |
| Disaster Recovery & Backup Automation | DONE | docs/research/disaster-recovery-backup.md |
| Load Testing Strategy & Capacity Planning | DONE | docs/research/load-testing-capacity-planning.md |
| Multi-Region Active-Active Deployment | DONE | docs/research/multi-region-active-active.md |
| OpenAPI 3.1 Spec Generation & API Docs | DONE | docs/research/openapi-spec-generation.md |
| User Self-Service & Onboarding | DONE | docs/research/user-self-service-onboarding.md |
| README & Quickstart Guide | DONE | docs/research/readme-quickstart-guide.md |
| Changelog & Release Notes Automation | DONE | docs/research/changelog-release-automation.md |
| OAuth Client Lifecycle Management (DCR) | NEXT | — |

---

*Board maintained by researcher (ggcxf). Implementation items are ready for pickup by backend/frontend/arch.*
