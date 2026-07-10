# GGID IAM Platform Roadmap

## Current Status

### Phase 9 (Complete) — Social Login, Hosted Login, API Docs
- 7 microservices + Admin Console + SDK (Go/Node/Java)
- OAuth2.1, OIDC, SAML, SCIM 2.0
- RBAC + ABAC policy engine
- Docker Compose full stack
- 31/31 test packages PASS

### Phase 10 (In Progress) — Enterprise Features
- [x] Webhooks
- [x] Audit timeline + report builder
- [x] SCIM provisioning dashboard
- [x] OAuth client registry UI
- [x] SAML SP config UI
- [ ] SAML IdP-initiated SSO
- [ ] SCIM 2.0 server implementation
- [ ] Session management with revocation
- [ ] Multi-tenant isolation hardening

### Phase 11 (In Progress) — Security Hardening
- [x] CSRF token crypto/rand entropy (P0 fix)
- [x] Rate limiter wired into handler chain (P0 fix)
- [x] SecurityHeaders wired into handler chain (P0 fix)
- [x] Tenant spoofing fix — JWT claim priority (P0 fix)
- [ ] OAuth state validation on token exchange (P0)
- [ ] JWT jti tracking (anti-replay) (P0)
- [ ] HasScope() actual scope enforcement (P0)
- [ ] Webhook SSRF protection (P0)
- [ ] Host header validation (DNS rebinding) (P0)
- [ ] Password pepper (P1)
- [ ] WebAuthn attestation verification (P1)
- [ ] JWT signing key rotation (P1)
- [ ] Audit hash chain (P1)
- [ ] gRPC TLS/mTLS between services (P1)
- [ ] Vault/KMS integration (P2)
- [ ] Multi-region active-active (P2)

## Coverage Targets

| Package | Current | Target | Owner |
|---------|---------|--------|-------|
| pkg/saml | 91.1% | 95% | dev |
| pkg/crypto | 89.4% | 92% | arch |
| pkg/audit | 83.3% | 90% | arch |
| pkg/email | 80.2% | 85% | arch |
| gateway/middleware | 89.2% | 92% | uiux |
| auth/service | 86.9% | 90% | dev |
| identity/scim | 65.5% | 80% | dev |
| auth/webauthn | 88.7% | 92% | dev |

## Research Docs Progress
- 100 files, 68K+ lines in docs/research/
- 15 P0 findings identified, 4 fixed by arch
- 25 P1 findings identified

## Team Capacity
- arch: pkg/, sdk/, console/, deploy/, docs/, .github/
- dev: services/identity/, services/auth/, services/oauth/, pkg/authprovider/
- uiux: services/gateway/ (middleware, router)
- doc: docs/ (all non-research docs)
- researcher: docs/research/ (security analysis)
- frontend: console/src/app/ (all pages)
