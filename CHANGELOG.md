# Changelog

All notable changes to GGID are documented here.
The format is based on [Conventional Commits](https://conventionalcommits.org).

## [v1.0-stable] - 2026-07-19

### Performance
- perf(auth): Argon2id env-tunable params + Redis session cache → login 258ms → 148ms (-43%)
- perf(gateway): Enhanced metrics middleware with per-route histograms
- perf(identity): PG indexes + Redis cache for user lookups

### OpenAPI / API
- feat(gateway): OpenAPI 3.1 spec — 723 paths, 47 requestBodies, 222 responseSchemas, 29 component schemas
- feat(gateway): EnhancedOperation builder for schema-rich endpoint definitions (top 80 endpoints)
- feat(gateway): Dynamic spec generation replaces static templates.go (fixes JSON parse errors)
- feat(gateway): Swagger UI at /docs with interactive API explorer
- feat(gateway): 4 Grafana dashboards (overview, auth-metrics, api-performance, security-overview)

### Security & Compliance
- feat(audit): Compliance framework mapping — SOC2 Type II (15 controls, 5 trust principles)
- feat(audit): ISO 27001 Annex A (10 key controls A.5~A.16)
- feat(audit): CCM cross-references (15 existing controls linked to framework mappings)
- feat(audit): GET /api/v1/audit/compliance-mapping?framework=soc2|iso27001
- feat(gateway): CORS strict defaults — no wildcard, dev-mode only
- feat(auth): Session invalidation triggers (password change, role revoke, admin lockout)
- feat(auth): Multi-hash password verifier (bcrypt/PBKDF2/scrypt/SSHA/Argon2id)
- feat(auth): Transparent password rehash on login (legacy → Argon2id)
- feat(auth): Conditional access policy integration
- feat(policy): SoD (Separation of Duties) → PG persistence
- feat(identity): NHI (Non-Human Identity) memory→PG dual-write migration
- feat(identity): Privilege creep detection with PG repo
- feat(identity): PAM session recording (API-level)
- feat(identity): Access review scheduling
- feat(oauth): JWT 90-day auto-rotation with grace period + audit callback

### DevOps & Infrastructure
- feat(deploy): Prebuilt binary Docker build pipeline (scripts/build-all-images.sh)
- feat(deploy): `make docker-push-services` — one-command 8-image build + push
- feat(deploy): Dockerfile.service — lightweight alpine + healthcheck for Go services
- feat(deploy): 38 SQL migrations (including compliance_mappings, RLS, dormant accounts)
- fix(gateway): defaultPrivateKeyPath off-by-one bug (len 11 → 10, "public.pem")

### Testing & Quality
- fix(test): Data race in key_rotation_test (sync.Mutex on auditCalls, channel-based wait)
- fix(test): Timing flakiness in StartAutoRotation tests (2s timeout for slow CI)
- test: go vet ./... zero warnings
- test: golangci-lint v1.64.8 CI green
- test: E2E 11/11 PASS
- test: UI smoke 822/828 (99.3%)

### Frontend
- feat(console): i18n lazy loading (3.3MB → per-language on-demand)
- feat(console): Dark mode support
- feat(console): Mobile responsive layout
- feat(console): Pagination on all list views
- feat(console): Version badge in navigation

### Documentation
- docs: Production deployment guide (AWS/GCP/Azure)
- docs: Build verification checklist
- docs: API conventions guide
- docs: Security hardening guide
- docs: Performance baseline document
- docs: Testing automation guide

## [Unreleased]

### ✨ Features

#### Authentication
- feat(auth): WebAuthn/FIDO2 enterprise features — Conditional UI, AAGUID allowlist, Device Public Key
- feat(auth): Temporary Access Pass (TAP) for passkey recovery
- feat(auth): JIT MFA enrollment for high-risk users
- feat(auth): Break-glass emergency access
- feat(auth): Impossible travel detection
- feat(auth): VPN/proxy detection
- feat(auth): Adaptive MFA with risk-based step-up
- feat(auth): Session management with risk-based timeout
- feat(auth): Device posture compliance checks
- feat(auth): Passwordless authentication (magic link, biometric)

#### Authorization
- feat(policy): Unified PDP (Policy Decision Point) — RBAC + ABAC + ReBAC + risk overlay
- feat(policy): ReBAC engine (Zanzibar-style) with Redis caching
- feat(policy): PAM JIT (zero standing privilege)
- feat(policy): Token Exchange (RFC 8693) with delegation chains
- feat(identity): PostgreSQL Row-Level Security (27 tables, tenant isolation)

#### OAuth / OIDC
- feat(oauth): OAuth 2.1 with PKCE, PAR, JAR, DPoP
- feat(oauth): Rich Authorization Requests (RAR)
- feat(oauth): Dynamic Client Registration (RFC 7591)
- feat(oauth): Client versioning + lifecycle management
- feat(oauth): Consent cascade (GDPR Art. 17, token/session revocation)
- feat(oauth): SCIM 2.0 outbound provisioning
- feat(oauth): AI Agent Identity (RFC 8693 agent token exchange)

#### Security
- feat(audit): ITDR with 15 MITRE ATT&CK detection rules
- feat(audit): UEBA behavioral baseline deviation (isolation forest)
- feat(audit): SOAR integration with automated response playbooks
- feat(audit): Hash-chained audit trail (HMAC-SHA256, tamper-evident)
- feat(audit): WORM storage (append-only PG + S3 Object Lock)
- feat(audit): Merkle tree accumulation (hourly roots)
- feat(audit): Continuous tamper detection + alerting
- feat(audit): Webhook engine (HMAC signed, retry, dead-letter)

#### Zero Trust
- feat(gateway): ZTNA Access Broker with posture-gated routing
- feat(gateway): CAE (Continuous Access Evaluation) middleware
- feat(gateway): DLP egress control (PII detection + redaction)
- feat(gateway): Hierarchical rate limiting (per-user/key/IP/endpoint)
- feat(gateway): Circuit breaker per-backend
- feat(gateway): WASM plugin architecture (wazero)
- feat(gateway): Security headers (CORS, HSTS, CSP, X-Frame-Options)
- feat(identity): MDM integration (Intune + Jamf connectors)
- feat(identity): Device certificate provisioning (SCEP + internal CA)
- feat(identity): Secret Broker (zero-trust secret injection)
- feat(identity): CMK/KMS (7 providers) with field-level encryption

#### Platform
- feat(identity): Tenant quota engine (5 dimensions, 3 tiers)
- feat(identity): HR-driven JML (Joiner-Mover-Leaver) automation
- feat(identity): Dormant account detection + ghost account reconciliation
- feat(identity): Decentralized Identity (W3C DID/VC) with OID4VCI/OID4VP
- feat(audit): Compliance automation (SOC2/ISO27001/NIST evidence collection)
- feat(org): Multi-tenant with RLS isolation
- feat(all): Graceful shutdown (SIGTERM handling)
- feat(all): Distributed tracing (W3C + OpenTelemetry)
- feat(all): Prometheus metrics (14 alert rules + Grafana dashboards)

### 🔒 Security

- security(audit): 8 new ITDR detection rules (consent phishing, MFA fatigue, token theft, session hijack, mass creation, federation anomaly, MFA bypass, mass export)
- security(gateway): CORS + security headers enforcement
- security(identity): PostgreSQL RLS on 27 tenant tables
- security(auth): Device fingerprint session binding
- security(crypto): Automated key rotation (dual-key pattern)

### 🐛 Bug Fixes
- fix(oauth): TestGapRegression_ClientVersioning_CRUD — oauthMapRepo in-memory fallback
- fix(gateway): SecurityHeadersConfigurable undefined symbols
- fix(gateway): GetCORSConfig → GetTenantCORS rename
- fix(md): Jamf test assertion logic
- fix(console): 49 useState crash bugs fixed (all pages now use proper useEffect)

### 📚 Documentation

- docs: 48 research documents covering Zero Trust, MDM, CMK/KMS, DLP, Service Mesh, ITDR, PETs, compliance, and more
- docs: 254 backlog items (KB-001 to KB-254)
- docs: Console feature pages F-42 through F-94
- docs: User guides for SCEP, HR lifecycle, webhook delivery, RLS, backup/DR

### 🔒 v1.0-beta Stability Phase

#### Quality & Testing
- test: 52 API security tests (auth/authz boundary cases across 25+ endpoints)
- test: 33 E2E integration tests (full gateway request lifecycle)
- test: Data race detection — 2 races found and fixed (atomic.Bool, atomic.Int32)
- test: go test -race ./services/gateway/... — clean
- test: 43/43 console page regression verification (all 200)
- docs: Quality baseline report (API latency, race detection, coverage)
- docs: Testing strategy (4-layer pyramid, CI pipeline, coverage targets)

#### Performance
- perf: API latency baseline — all 5 core endpoints < 200ms (25-167ms)
- perf: Login 167ms, Users 54ms, Policies 37ms, Audit 59ms, Sessions 25ms

#### Frontend
- feat: F-140 i18n audit — hardcoded strings → t() across console
- feat: F-141 a11y improvements — aria-label, label, alt text
- feat: Navigation system refactor (8 functional domains + search + collapse)
- feat: First-time setup wizard (5-step guided onboarding)
- feat: Console experience polish (404/error boundary/loading skeleton/page titles)
- fix: tsc TS7006 errors 834 → 4 (-99.5%)

#### Backend
- fix: KB-312 error handling unification (writeError → writeJSONError)
- fix: Unused imports cleanup
- fix: Data race in TimeoutMiddleware and JWKS refresh tests
- fix: NHI repo nil pool guard (EnsureSchema panic prevention)

#### Documentation
- docs: Documentation completeness audit — 95.7% feature coverage (45/47)
- docs: China GM (SM2/SM3/SM4) compliance guide
- docs: Temporary Access Pass (TAP) guide
- docs: Product overview, admin quickstart, integration guide
- docs: Getting started, testing strategy, quality baseline
- docs: GAP convergence report — 12 critical gaps resolved

#### Security
- docs: KB-313 Security checklist
- test: API security test coverage: no-token, invalid-token, cross-tenant, rate-limit, JSON injection, oversized body, header injection
- fix: 0 make(map) in non-cache production code

### 📊 v1.0-beta Final Statistics

- **Console pages**: 825
- **Test functions**: 4461 (including 85+ API security, 33 E2E)
- **API endpoints**: 864+
- **OpenAPI paths**: 704 (81% coverage)
- **SDKs**: 11 languages
- **Code migrations**: 32 SQL migration files
- **User guides**: 364
- **Research documents**: 292
- **CI pass rate**: >90%
- **tsc TS7006**: 4 (from 834, -99.5%)
- **Documentation coverage**: 95.7%

## [v0.1.0] - Initial Release

### Added
- Core microservices: gateway, auth, identity, oauth, policy, audit, org
- OAuth 2.1 with PKCE
- RBAC + ABAC authorization
- PostgreSQL + Redis + NATS infrastructure
- Docker Compose deployment
- K8s deployment manifests
- Go SDK (production-ready)
- React Console (504 files)

---

*This changelog will be auto-generated from conventional commits in future releases using git-cliff.*
