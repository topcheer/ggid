# GGID Team Backlog

> **Last updated**: 2025-01-24 by arch
> **Rule**: Update this file when completing any item. Check here before assigning new work.

## P0 — Security (Critical)

### dev (OAuth/Auth security)
- [x] OAuth state validation on token exchange (72edaa5)
- [x] iss parameter in auth redirect (72edaa5)
- [x] JWT jti tracking with Redis SETNX (72edaa5)
- [x] HasScope() actual scope enforcement (72edaa5)
- [x] JWTSecret empty → log.Fatal (72edaa5)
- [x] Access token scope claim — issueAccessToken scope param (IN PROGRESS — build broken, needs fix)
- [ ] Password pepper — pkg/crypto/pepper.go exists but NOT wired to auth service boot
- [ ] WebAuthn attestation verification — attestation_formats.go exists, verify all 6 formats covered
- [ ] OAuth introspection endpoint auth (server.go — NO AUTH)

### uiux (Gateway security)
- [x] Host header validation middleware (host_validation.go exists)
- [x] Webhook HTTPDeliverer SSRF protection (ssrf.go + b52bafd)
- [x] IP allowlist for admin endpoints (ipallowlist.go exists)
- [x] Body size limit middleware (bodysize.go exists)
- [ ] Middleware coverage →92% (currently 88.7%)

### arch (Infrastructure security)
- [x] CSRF token predictable entropy (29b51c1)
- [x] Rate limiter wired into handler chain (fc20c41)
- [x] SecurityHeaders wired into handler chain (64991a6)
- [x] Tenant spoofing fix (5bcbfce)
- [x] Webhook SSRF protection (b52bafd)
- [x] Audit hash chain implementation (fe5b025 — hash_chain.go)
- [ ] gRPC TLS/mTLS between services
- [ ] JWT signing key rotation infrastructure
- [ ] Database backup automation (pg_dump cron)

## P1 — Feature Development

### dev
- [x] SCIM 2.0 PATCH support (2caa572)
- [x] SAML IdP-initiated SSO (2caa572 — idp_initiated.go)
- [x] OAuth DPoP support (2caa572 — dpop.go)
- [x] TOTP backup codes (2caa572 — backup_codes.go)
- [x] Session timeout middleware (2caa572)
- [ ] OIDC back-channel logout (RFC 8411)
- [ ] Session management with revocation list (session_management.go exists — verify completeness)

### uiux
- [x] GraphQL proxy middleware (graphql.go exists)
- [x] WebSocket upgrade support (wsproxy.go exists)
- [x] gRPC server reflection (grpc.go + grpc_interceptor.go exist)
- [ ] Deep health check aggregation
- [ ] Per-route timeout middleware

### frontend
- [x] Console SSO Configuration page (settings/sso + sso alias)
- [x] Console Notification Templates page (notifications/templates)
- [x] Console Audit Log Advanced Filters (audit page with date range, filters, export)
- [x] Console User Import Wizard (users/import with CSV, mapping, preview)
- [x] Console Password Policy Editor (settings/password-policy)
> All assigned frontend tasks complete. Need NEW page ideas for next batch.

### doc
- [x] docs/architecture.md
- [x] docs/architecture-decisions.md
- [x] docs/sdk-guide.md (sdk-reference.md)
- [x] docs/development.md (developer-environment.md)
- [x] docs/api-reference.md
- [x] docs/rbac-guide.md (untracked, from doc teammate)
- [x] docs/webhook-guide.md (untracked, from doc teammate)
- [x] docs/multi-tenancy.md (modified, from doc teammate)
> 128 docs total. Need to audit what's missing.

### arch
- [x] SDK coverage tests (sdk/go — 71.4% coverage)
- [x] Docker multi-stage build (deploy/)
- [x] Prometheus /metrics for all services (122873e)
- [x] Structured logging slog for gateway (122873e)
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Helm chart for Kubernetes
- [ ] OpenTelemetry integration

## P2 — Quality & Polish

### All
- [ ] Coverage →95% across all packages
- [ ] Integration test suite (10+ E2E tests)
- [ ] Performance benchmarks
- [ ] Dark mode for Console
- [ ] Mobile-responsive Console

## P3 — Future

- [ ] Multi-region active-active deployment
- [ ] Vault/KMS integration
- [ ] FIDO2 certification
- [ ] Compliance certifications (SOC2, ISO27001)
- [ ] Plugin system architecture

## Coordination Rules

1. **Before assigning any task**: `ls` and `grep` to verify target doesn't exist
2. **After completing any task**: update this file immediately
3. **Don't edit other teammates' files** without explicit coordination
4. **dev owns**: services/oauth/, services/auth/
5. **uiux owns**: services/gateway/internal/middleware/
6. **arch owns**: pkg/, deploy/, sdk/, services/audit/, services/identity/
7. **frontend owns**: console/
8. **doc owns**: docs/ (except team-backlog.md)
9. **researcher owns**: docs/research/
