# GGID Team Backlog

*Last updated: 2026-07-13 00:30 (hourly cycle)*

## Current Stats
- **Docs**: 640 markdown files
- **Console pages**: 611 page.tsx
- **React hooks**: 492 use*.ts
- **Go SDK**: 32 files, 154+ test functions
- **Go services**: 271+ source files, 293+ test files
- **Build**: `go build ./...` = CLEAN
- **Tests**: 40/40 packages PASS, 0 FAIL
- **Session commits**: 250+ across all teams

## Completed This Session (All Teams)

### Arch Commits (This Cycle)
- OpenAPI spec updated: +5 config endpoints (user-lifecycle, ABAC condition, SCIM provisioning, export schedule, token rotation)
- Gap regression tests: 42 tests (policy 17, audit 14, oauth 11)
- Go SDK analytics extensions: 17 methods (commit `28f1180`)
- Research backlog: ITDR/fraud/agent-lifecycle/PIPL/OAuth2.1 (29 items)

### Backend Commits
- `05c1a4d`: User lifecycle config, ABAC condition config, SCIM provisioning config, export schedule config, token rotation config (40/40 PASS)
- `ffa6ad4`: CIBA, JAR, federation metadata, session binding, PAR config
- `d26f702`: Permission inheritance, alert webhook, permission tree, rate limits, session stream
- `c1d6ba8`: Delegation validate, password history, SIEM health, account linking, consent mgmt
- `a01d849`: ABAC evaluate, compliance schedule CRUD, import validate
- `4c10363`: Role templates, correlation route confirm, bulk status, RFC 7592

### Frontend Commits
- `997a952`: gRPC interceptor + connection pool + feature flag + log aggregation + health check config pages
- `1814f4f`: Distributed tracing + canary deployment + database migration + OAuth scope tiering + secret sprawl config pages
- `c1d6ba8`: Agent delegation + zero trust + PQC migration + compliance automation + identity recovery config pages
- `d24051d`: OAuth 2.1 checker + identity schema + error catalog + webhook catalog + tenant isolation config pages
- `6029adf`: Introspection cache + token prefetch + policy hot-reload + audit query optimization + OAuth backpressure pages

### Docs Commits
- `d03e391`: mTLS between services, webhook retry, config management, gRPC streaming, API rate limit tuning (454 docs)
- `f66e005`: Performance benchmarking, blue-green deploy, IaC, cost monitoring, security hardening (449 docs)
- `1fd0528`: Auto-scaling, service mesh observability, DR testing, incident command, SRE practices (444 docs)
- `4c10363`: gRPC interceptors, connection pool tuning, feature flags, log aggregation, health checks (439 docs)
- `c075395`: Distributed tracing, canary deployment, DB migration, OAuth scope tiering, secret sprawl (434 docs)

## Currently Dispatched (In Progress)

### Backend (Batch — dispatched, awaiting report)
1. Risk scoring config (risk factors, weights, thresholds, action mapping)
2. SOD conflict detection config (rules, sensitivity, auto-remediate)
3. PAR config (require_par_for_scopes/clients, enforce PKCE, cache)
4. SIEM forwarder config (destinations, filters, retry, circuit breaker)
5. Password policy config (complexity, history, breach detection, pepper)

### Frontend (Batch — re-dispatched, awaiting report)
1. Auto-scaling config page
2. Circuit breaker dashboard page
3. Service dependency graph page
4. API gateway routes management page
5. K8s deployment management page

### Docs (Batch — dispatched, awaiting report)
1. Credential vault architecture
2. Adaptive authentication
3. Digital identity lifecycle
4. Token binding strategies
5. API versioning strategy

## Pending Backlog (Not Yet Dispatched)

### Backend (Next)
- [ ] ITDR: detection rules catalog (services/auth/internal/server/itdr_handler.go)
- [ ] ITDR: automated response playbooks (block→revoke→notify→ticket)
- [ ] ITDR: lateral movement detection + privilege escalation detection
- [ ] Fraud: device fingerprinting service (pkg/fraud/)
- [ ] Fraud: velocity rules engine + synthetic identity detection
- [ ] Agent: persistent registry (database-backed) + behavioral monitoring
- [ ] **P1** ML-DSA JWT signing in pkg/crypto (PQC)
- [ ] **P1** Hybrid PQC TLS in gateway (ML-KEM)
- [ ] **P1** Workload identity federation (SPIFFE/SPIRE)
- [ ] **P1** OAuth 2.1 enforcement (mandatory PKCE, reject implicit)
- [ ] **P2** Crypto-agility registry in pkg/crypto
- [ ] **P2** Geo-velocity anomaly detection
- [ ] **P2** Device posture API + conditional access integration

### Frontend (Next)
- [ ] ITDR dashboard (console/src/app/settings/itdr-dashboard/)
- [ ] Fraud detection dashboard (console/src/app/settings/fraud-detection/)
- [ ] Agent lifecycle dashboard (console/src/app/settings/agent-lifecycle/)
- [ ] Settings - Email templates editor
- [ ] Users - Bulk import wizard (CSV upload + preview)
- [ ] Audit - Real-time event heatmap
- [ ] **P2** Machine identity inventory dashboard

### Docs (Next)
- [ ] docs/guides/itdr-implementation.md
- [ ] docs/guides/fraud-detection.md
- [ ] docs/guides/ai-agent-lifecycle.md
- [ ] docs/guides/pipl-compliance.md
- [ ] docs/guides/high-availability.md
- [ ] docs/api/org-api.md
- [ ] docs/research/iam-market-landscape-2026.md

### SDK (Next)
- [ ] Go SDK: risk scoring, SOD, PAR, SIEM forwarder, password policy client methods
- [ ] Node SDK: admin extensions (matching Go)
- [ ] React SDK: useRiskScoring, useSODConfig, usePARConfig hooks

---

## Research-Driven Backlog
*Source: docs/research/itdr-fraud-agent-lifecycle-gaps.md*

### Backend (P1)
- [ ] **P1** ITDR: MITRE ATT&CK identity mapping
- [ ] **P1** Fraud: TOR/VPN/proxy detection
- [ ] **P1** Agent: consent flow (user approves agent scope)
- [ ] **P1** Agent: credential rotation automation

### Backend (P2)
- [ ] **P2** PIPL: data handling rules for Chinese users
- [ ] **P2** PIPL: cross-border transfer assessment
- [ ] **P2** OAuth 2.1: compliance audit tool + deprecation enforcement
- [ ] **P2** Passkey: health dashboard API

### Frontend (P2)
- [ ] **P2** Passkey health dashboard
- [ ] **P2** OAuth 2.1 compliance checker page
- [ ] **P2** PIPL compliance config page

### Docs (P2)
- [ ] **P2** docs/guides/pqc-migration-guide.md
- [ ] **P2** docs/oauth-2-1-compliance-statement.md
