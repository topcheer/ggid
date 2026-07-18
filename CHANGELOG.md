# Changelog

All notable changes to GGID are documented here.
The format is based on [Conventional Commits](https://conventionalcommits.org).

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

### 📊 Session Statistics

- **Console pages**: 53 feature pages (F-42 to F-94)
- **Research rounds**: 48
- **Backlog items**: 254 (KB-001 to KB-254)
- **Research documents**: 48+ deep-dive analyses
- **Test packages**: 61/61 passing
- **API endpoints**: 786+
- **SDKs**: 11 languages
- **Code migrations**: 37 SQL migration files

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
