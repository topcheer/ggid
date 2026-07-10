# GGID IAM Platform — Development Roadmap

> Derived from the [Feature Comparison Matrix](./feature-matrix.md).
> Aligns 157 benchmarked features into phased delivery milestones.

---

## Current State (As of Phase 8)

| Metric | Value |
|--------|-------|
| Services | 7 microservices + Console + Gateway |
| Feature Coverage | 58/157 fully implemented (37%) |
| Test Coverage | 250+ test cases, 15 packages, 0 FAIL |
| Docker | 13 containers, 11/11 E2E tests pass |
| SDKs | Go, Node.js, Java |

### Strengths
- RBAC + ABAC policy engine (60% coverage — strongest category)
- Organization tree with multi-tenant RLS (58%)
- User lifecycle management (42%)
- gRPC + REST dual protocol (unique advantage)
- Temporary role assignment with TTL (no competitor has this)

### Critical Gaps
- No social login (blocks B2C)
- No pre-built login UI (blocks developer adoption)
- No enterprise SSO connectors (blocks B2B)
- WebAuthn only skeleton
- Missing basic web security (CORS, CSRF, cookie flags)

---

## Phase 9 — Foundation Hardening (P0 Blockers)

> **Goal:** Close table-stakes gaps that block GA release.
> **Timeline:** 4-6 weeks
> **Theme:** "Every competitor has these; we must too."

### 9.1 Social Login Connectors
**Priority:** P0 | **Effort:** Medium | **Reference:** Logto connectors, Clerk social

Deliverables:
- [x] Pluggable connector interface (`pkg/social/connector.go`)
- [x] Google OAuth2 connector
- [x] GitHub OAuth2 connector
- [ ] Microsoft / Apple connectors
- [x] OIDC generic connector (any compliant IdP)
- [x] Social login callback handler in Auth Service
- [ ] Identity linking flow (link social account to existing user)
- [ ] JIT provisioning for social logins

**Design:** Follow Logto's connector pattern — each provider is a self-contained package implementing a common `Connector` interface. Store connector configs per-tenant in the database.

### 9.2 WebAuthn / Passkey Full Implementation
**Priority:** P0 | **Effort:** Medium | **Reference:** Clerk passkeys, go-webauthn

Deliverables:
- [ ] Replace skeleton WebAuthn handler with full implementation
- [ ] Registration flow (attestation creation + verification)
- [ ] Authentication flow (assertion creation + verification)
- [ ] Credential storage in PostgreSQL (per-user, per-device)
- [ ] Passkey login as passwordless alternative
- [ ] Console UI for passkey management

**Current state:** `services/auth/internal/webauthn/handler.go` exists but passes `nil` credential store. Need a PostgreSQL-backed `CredentialStore` implementation.

### 9.3 Pre-built Hosted Login UI
**Priority:** P0 | **Effort:** Medium | **Reference:** Auth0 Universal Login, Clerk

Deliverables:
- [ ] Standalone login page served by Gateway or Auth Service
- [ ] Username/password form
- [ ] Social login buttons (rendered from configured connectors)
- [ ] MFA challenge form (TOTP code entry)
- [ ] Password reset flow UI
- [ ] Registration form
- [ ] Customizable branding (logo, colors, CSS per tenant)
- [ ] Localization framework (Chinese + English)

### 9.4 OIDC Completeness
**Priority:** P0 | **Effort:** Small

Deliverables:
- [ ] UserInfo endpoint returns real user data (currently stub)
- [ ] Token introspection returns real token status (currently stub)
- [ ] OAuth authorize endpoint implements real redirect flow (currently stub)
- [ ] Client credentials grant type (M2M tokens)

### 9.5 Web Security Baseline
**Priority:** P0 | **Effort:** Small-Medium

Deliverables:
- [ ] CORS middleware (configurable allowed origins per tenant)
- [x] Cookie security flags (HttpOnly/Secure/SameSite) (HttpOnly, Secure, SameSite=Lax/Strict)
- [ ] CSRF protection (double-submit cookie pattern)
- [ ] Security headers middleware (X-Content-Type-Options, X-Frame-Options, CSP)
- [ ] TLS configuration guidance + HSTS header

### 9.6 API Documentation
**Priority:** P0 | **Effort:** Small

Deliverables:
- [ ] OpenAPI 3.0 spec for all REST endpoints
- [ ] Swagger UI served at `/docs`
- [ ] Postman collection export
- [ ] API reference documentation in `docs/api/`

### 9.7 Python SDK
**Priority:** P0 | **Effort:** Medium

Deliverables:
- [ ] JWT verification middleware (FastAPI, Django, Flask)
- [ ] Permission checking client
- [ ] Auto-generate from OpenAPI spec if possible
- [ ] PyPI package publish pipeline

---

## Phase 10 — Enterprise Readiness (P1)

> **Goal:** Win B2B enterprise deals.
> **Timeline:** 6-8 weeks after Phase 9
> **Theme:** "SSO, SCIM, and directory sync for enterprise IT."

### 10.1 SAML SP (Service Provider)
**Priority:** P0 (enterprise) | **Effort:** Large | **Reference:** Keycloak SP

Deliverables:
- [ ] SAML assertion parser and validator
- [ ] XML signature verification
- [ ] SP-initiated SSO redirect binding
- [ ] SP-initiated SSO POST binding
- [ ] SAML attribute mapping to user profile
- [ ] Per-tenant SAML IdP configuration
- [ ] Metadata exchange (SP metadata endpoint)

### 10.2 Enterprise SSO Connectors
**Priority:** P0 (enterprise) | **Effort:** Large | **Reference:** WorkOS 50+ connectors

Deliverables:
- [ ] Okta SSO connector template
- [ ] Azure AD / Entra ID connector template
- [ ] Google Workspace connector template
- [ ] Generic SAML connector (any SAML 2.0 IdP)
- [ ] Admin UI for SSO configuration per organization
- [ ] Connection testing endpoint

### 10.3 Per-Tenant SSO Configuration
**Priority:** P0 | **Effort:** Medium | **Reference:** WorkOS/Logto per-org SSO

Deliverables:
- [ ] Database schema: tenant SSO configurations table
- [ ] API: CRUD for tenant SSO settings
- [ ] Gateway: route auth requests to correct IdP based on tenant
- [ ] Console UI: SSO settings page per organization

### 10.4 SAML IdP Full Implementation
**Priority:** P1 | **Effort:** Large | **Reference:** Keycloak IdP

Deliverables:
- [ ] SAML response generation with XML signing
- [ ] IdP-initiated SSO
- [ ] IdP metadata with real signing keys
- [ ] SAML SLO (Single Logout)

### 10.5 Directory Sync (SCIM Outbound)
**Priority:** P1 | **Effort:** Large | **Reference:** WorkOS directory sync

Deliverables:
- [ ] SCIM outbound provisioning engine
- [ ] Push user changes to external systems (Slack, Google, etc.)
- [ ] Deprovisioning on user disable/delete
- [ ] Sync status dashboard in Console

### 10.6 Delegated Administration
**Priority:** P1 | **Effort:** Medium | **Reference:** WorkOS Admin Portal

Deliverables:
- [ ] Organization admin roles (scoped to org)
- [ ] Admin portal embeddable in customer apps
- [ ] Org-scoped user management API
- [ ] Org-scoped SSO/SCIM configuration

### 10.7 Webhooks & Event Subscriptions
**Priority:** P1 | **Effort:** Medium

Deliverables:
- [ ] Webhook configuration API (per-tenant event subscriptions)
- [ ] Event delivery with retry (exponential backoff)
- [ ] Webhook signing (HMAC)
- [ ] Event types: user.created, user.deleted, role.assigned, etc.
- [ ] NATS consumer → webhook dispatcher

### 10.8 Additional MFA Methods
**Priority:** P1 | **Effort:** Medium

Deliverables:
- [ ] Email OTP MFA
- [ ] SMS OTP MFA (Twilio integration)
- [ ] MFA enforcement policy (per-tenant: required, optional, disabled)
- [ ] Adaptive/step-up MFA (trigger based on risk score)

### 10.9 Passwordless Authentication
**Priority:** P1 | **Effort:** Medium | **Reference:** Logto/Ory

Deliverables:
- [ ] Magic link (email) login flow
- [ ] SMS OTP passwordless flow
- [ ] Unified passwordless + password + social login page

### 10.10 Kubernetes Deployment
**Priority:** P1 | **Effort:** Medium

Deliverables:
- [ ] Helm chart for all 7 services + infrastructure
- [ ] K8s manifests with health checks, readiness probes
- [ ] ConfigMap/Secret management
- [ ] Horizontal Pod Autoscaler templates
- [ ] Production deployment guide

---

## Phase 11 — Security & Compliance (P1)

> **Goal:** Pass enterprise security audits.
> **Timeline:** 4-6 weeks after Phase 10
> **Theme:** "SOC2-ready, breach-resistant, audit-complete."

### 11.1 Breached Password Detection
**Priority:** P1 | **Effort:** Small | **Reference:** Auth0, Clerk (HIBP)

- [ ] Integrate HaveIBeenPwned API
- [ ] Check on registration and password change
- [ ] Block or warn based on policy

### 11.2 Key Rotation
**Priority:** P1 | **Effort:** Medium

- [ ] JWT signing key rotation with overlapping validity
- [ ] JWKS endpoint serves current + previous keys
- [ ] Admin API to trigger rotation
- [ ] Automated rotation schedule

### 11.3 Anomaly Detection
**Priority:** P1 | **Effort:** Large | **Reference:** Auth0 Attack Protection

- [ ] Impossible travel detection (geo-velocity)
- [ ] New device/location alerting
- [ ] Suspicious IP detection (known bad IPs)
- [ ] Risk score calculation on login

### 11.4 Audit & Compliance Enhancement
**Priority:** P1 | **Effort:** Medium

- [ ] Compliance report templates (SOC2, GDPR)
- [ ] Log retention policies (configurable per event type)
- [ ] SIEM integration (Splunk, Datadog log streaming)
- [ ] Admin activity logging (all console actions)
- [ ] Per-user login history
- [ ] Immutable audit trail (hash-chain verification)

### 11.5 IP Allowlisting / Blocklisting
**Priority:** P1 | **Effort:** Medium

- [ ] Per-tenant IP allow/deny lists
- [ ] Gateway middleware to enforce IP rules
- [ ] Admin Console UI for IP management

### 11.6 User Import / Export
**Priority:** P1 | **Effort:** Medium

- [ ] CSV/JSON bulk user import API
- [ ] User export API (GDPR data portability)
- [ ] Import job status tracking
- [ ] Dry-run / validation mode

### 11.7 User Search
**Priority:** P1 | **Effort:** Medium

- [ ] Full-text search across user attributes
- [ ] PostgreSQL tsvector or Elasticsearch integration
- [ ] Search API with relevance scoring
- [ ] Console search bar

---

## Phase 12 — Growth & Polish (P2)

> **Goal:** Delight developers and match best-in-class UX.
> **Timeline:** Ongoing after Phase 11

### 12.1 Developer Experience
- [ ] Quick start guides (Next.js, Express, FastAPI, Spring Boot)
- [ ] Sample applications with GGID integration
- [ ] Interactive API playground
- [ ] SDK for Ruby
- [ ] SDK for PHP
- [ ] CLI tool for management (`ggid-cli`)
- [ ] Terraform provider for infrastructure-as-code

### 12.2 Advanced Features
- [ ] Push notification MFA (mobile app)
- [ ] ReBAC (Google Zanzibar model)
- [ ] GraphQL API
- [ ] User impersonation (admin tool)
- [ ] Bot detection / CAPTCHA integration
- [ ] Custom domain support per tenant
- [ ] Remember Me / long-lived sessions
- [ ] Single Logout (SLO) across applications

### 12.3 Compliance & Certification
- [ ] SOC2 Type II audit preparation
- [ ] ISO 27001 alignment
- [ ] HIPAA compliance documentation
- [ ] Data residency (multi-region deployment)

### 12.4 Mobile SDKs
- [ ] iOS SDK (Swift)
- [ ] Android SDK (Kotlin)
- [ ] React Native integration guide
- [ ] Flutter integration guide

---

## Milestone Summary

| Phase | Theme | Duration | Key Outcomes |
|-------|-------|----------|--------------|
| **9** | Foundation Hardening | 4-6 weeks | Social login, WebAuthn, Login UI, OIDC complete, Web security, OpenAPI, Python SDK |
| **10** | Enterprise Readiness | 6-8 weeks | SAML SP, SSO connectors, Per-tenant SSO, SCIM outbound, Delegated admin, Webhooks, MFA methods, Passwordless, K8s |
| **11** | Security & Compliance | 4-6 weeks | Breached password check, Key rotation, Anomaly detection, Audit enhancement, IP rules, Import/export, Search |
| **12** | Growth & Polish | Ongoing | Quick starts, Advanced features, Compliance cert, Mobile SDKs |

---

## Coverage Projection

| Phase | Projected Coverage | Delta |
|-------|--------------------|-------|
| Current (Phase 8) | 37% (58/157) | — |
| After Phase 9 | ~55% (~86/157) | +28 features |
| After Phase 10 | ~72% (~113/157) | +27 features |
| After Phase 11 | ~85% (~134/157) | +21 features |
| After Phase 12 | ~95% (~149/157) | +15 features |

---

## Team Allocation Suggestions

Based on existing team structure and file ownership:

| Teammate | Phase 9 Focus | Phase 10 Focus |
|----------|---------------|----------------|
| **dev** (Identity + Auth Provider) | 9.2 WebAuthn, 9.4 OIDC completeness | 10.8 MFA methods, 10.9 Passwordless |
| **dev2** (Auth Service + Gateway) | 9.1 Social login, 9.5 Web security | 10.1 SAML SP, 10.3 Per-tenant SSO |
| **dev3** (Policy + Org + Audit) | 9.7 Python SDK (policy/client) | 10.7 Webhooks (NATS consumer), 11.4 Audit enhancement |
| **arch** (Shared + SDK + Console + Infra) | 9.3 Login UI, 9.6 OpenAPI, 9.7 Python SDK | 10.2 SSO connectors, 10.6 Delegated admin, 10.10 K8s Helm |

---

## Success Metrics

| Metric | Current | Phase 9 Target | Phase 10 Target | Phase 11 Target |
|--------|---------|----------------|-----------------|-----------------|
| Feature coverage | 37% | 55% | 72% | 85% |
| E2E tests | 11 | 30+ | 50+ | 70+ |
| SDK languages | 3 | 4 (+Python) | 4 | 5+ |
| Social login providers | 0 | 4+ | 8+ | 10+ |
| Docker containers | 13 | 13 | 14+ (connectors) | 14+ |
| P0 gaps closed | 0/12 | 12/12 | 12/12 | 12/12 |
| P1 gaps closed | 0/19 | 0/19 | 12/19 | 19/19 |
