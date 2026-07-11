# GGID Team Backlog

> **Last updated**: 2025-01-24 by uiux
> **Rule**: Update this file when completing any item. Check here before assigning new work.

## ⚠️ Backlog Maintenance Rules — ALL TEAM MEMBERS MUST FOLLOW

### 1. 开工前 (Before starting work)
- 读这个文件，确认你的任务还在 Outstanding 区
- `ls` / `grep` 验证目标文件确实不存在

### 2. 完成后 (After completing work) — 必须做
- 把你的任务从 `- [ ]` 改成 `- [x]`
- 加上 commit hash：`- [x] Task name (commit abc1234)`
- 如果是新功能，在对应 P1/P2 区新增条目
- `git add docs/team-backlog.md && git commit -m "docs: mark <task> done"`

### 3. 发现重复时 (When finding duplicates)
- 不要创建已存在的文件
- 在 DM 中回复 "DUPLICATE: <task> already exists at <path>"
- 不要写入任何内容

### 4. 文件归属 (Ownership)
- **dev**: services/oauth/, services/auth/
- **uiux**: services/gateway/internal/middleware/
- **arch**: pkg/, deploy/, sdk/, services/audit/, services/identity/
- **frontend**: console/
- **doc**: docs/ (except team-backlog.md)
- **researcher**: docs/research/
- **不要修改其他人的文件**，除非有明确协调

## P0 — Security (Critical)

### dev (OAuth/Auth security)
- [x] OAuth state validation on token exchange (72edaa5)
- [x] iss parameter in auth redirect (72edaa5)
- [x] JWT jti tracking with Redis SETNX (72edaa5)
- [x] HasScope() actual scope enforcement (72edaa5)
- [x] JWTSecret empty → log.Fatal (72edaa5)
- [x] Access token scope claim — issueAccessToken scope param (b8bfa05, test fix 5c51080)
- [x] Password pepper — SetPepper() + applyPepper() wired (ce9c29f)
- [x] WebAuthn attestation verification — 6 format verifiers in attestation_formats.go (ce9c29f)
- [x] OAuth introspection endpoint auth — client_secret_basic required (ce9c29f)
- [x] ValidateClientAssertion JWT signature verification — replaced ParseUnverified with ParseWithClaims (8098f1c)
- [x] ForgotPassword token leak — tokens removed from HTTP response bodies (7742916)
- [x] UserRole.ExpiresAt enforcement — evaluator filters expired roles (7742916)
- [x] Email template href injection — safeURL() blocks javascript:/data: protocols (7742916)

### uiux (Gateway security)
- [x] Host header validation middleware (host_validation.go, ed4124b)
- [x] Webhook HTTPDeliverer SSRF protection (ssrf.go, ed4124b)
- [x] IP allowlist for admin endpoints (ipallowlist.go exists)
- [x] Body size limit middleware (bodysize.go exists)
- [x] Circuit breaker per backend (circuitbreaker.go, aa49848)
- [x] Middleware coverage →90% (coverage_sprint24_test.go, 80b9378)
- [ ] Middleware coverage →92% (currently 88.7%)

### arch (Infrastructure security)
- [x] CSRF token predictable entropy (29b51c1)
- [x] Rate limiter wired into handler chain (fc20c41)
- [x] SecurityHeaders wired into handler chain (64991a6)
- [x] Tenant spoofing fix (5bcbfce)
- [x] Webhook SSRF protection (b52bafd)
- [x] Audit hash chain implementation (fe5b025 — hash_chain.go)
- [ ] gRPC TLS/mTLS between services
- [x] JWT key persistence + kid header (loadOrCreatePrivateKey + kid in JWT)
- [x] JWKS endpoint (oauth /oauth/jwks)
- [x] Database backup automation (arch a9b56da — backup.sh + restore.sh)
- [ ] JWT key rotation (generate new key, keep old for grace period)

## P1 — Feature Development

### dev
- [x] SCIM 2.0 PATCH support (2caa572)
- [x] SAML IdP-initiated SSO (2caa572 — idp_initiated.go)
- [x] OAuth DPoP support (2caa572 — dpop.go)
- [x] TOTP backup codes (2caa572 — backup_codes.go)
- [x] Session timeout middleware (2caa572)
- [x] OIDC back-channel logout (dev ce9c29f)
- [x] Session management with revocation list (dev ce9c29f)
- [x] Password pepper wired (dev ce9c29f)
- [x] WebAuthn attestation formats (dev ce9c29f)
- [x] OAuth introspection auth (dev ce9c29f)
- [x] Token scope claim (dev ce9c29f)

### uiux
- [x] GraphQL proxy middleware (graphql.go exists)
- [x] WebSocket upgrade support (wsproxy.go exists)
- [x] gRPC server reflection (grpc.go + grpc_interceptor.go exist)
- [x] Deep health check aggregation (/healthz/deep wired in router.go, 348d61f)
- [x] Per-route timeout middleware (route_timeout.go, 348d61f)
- [x] OTel tracing middleware (TracingMiddleware in otel.go:311)
- [x] Performance benchmarks (benchmark_test.go — 6 benchmarks, 348d61f)

### frontend
- [x] Console User Profile Settings /settings/profile (de8093e)
- [x] Console Security Settings /settings/security (108589b)
- [x] Console Activity Log /activity (a6502a1)
- [x] Console Role Permissions Matrix /roles/matrix (d1f4785)
- [x] Console Organization Chart /orgs/chart (e47ed3d)
- [x] Console Tenant Settings /settings/tenant (0da3db8)
- [x] Console Audit Timeline /audit/timeline (01b75f4)
- [x] Console OAuth Flow Visualizer /oauth/flows (16212b2)
- [x] Console Password Policy /settings/password-policy (6dfb2a7)
- [x] Console IP Allowlist /settings/ip-allowlist (0955029)
- [x] Console Certificate Management /settings/certificates (fbf281b)
- [x] Console OAuth Consent Screen /oauth/consent (7a1c8c0)
- [x] Console Admin Impersonation /admin/impersonate (ce70bc7)
- [x] Console Email/SMS Preview /settings/notifications/preview (0246aef)
- [x] Console User Onboarding Wizard /onboarding (c1174b5)
- [x] Console Audit Timeline Visualization /audit/visualization (2909966)
- [x] Console Policy Editor Visual Builder /policies/builder (0b47af3)
- [x] Console SCIM Provisioning Dashboard /scim (9aeb111)
- [x] Console Security Center /security-center (d1f2cad)
- [x] Console Branding Customization /settings/branding-custom (874b78b)
- [x] Console Role Permission Matrix /roles/permission-matrix (a8c50b2)
- [x] Console Login Flow Builder /flows (fc13057)
- [x] Console Data Export Center /exports (b326c65)
- [x] Console API Key Management /apikeys (fc6e78a)
- [x] Console Webhook Management /webhooks (b1d668f)
- [x] Console User Sessions Manager /sessions (0f6c239)
- [x] Console Tenant Settings Enhanced /settings/tenant-config (a9e1d47)
- [x] Console OAuth Client Registry /oauth/clients (ec7f5e2)
- [x] Console SAML SP Config /saml (3162a11)
- [x] Console Audit Report Builder /audit/reports (c5d8bf0)
- [x] Console SSO Configuration /settings/sso + /sso alias (25fdbec, fe5b025)
- [x] Console Notification Templates /notifications/templates (69da107)
- [x] Console Audit Advanced Filters /audit/advanced (b82dabc)
- [x] Console User Import Wizard /users/import (c48238d)
- [x] Console Password Policy Editor /settings/password (228adec)
- [x] Console Organization Tree /organizations/tree (86f607e)
- [x] Console Permission Explorer /permissions (536dca7)
- [x] Console Group Management Enhanced /groups (9b64ed7)
- [x] Console OAuth Consent Enhanced /oauth/consent (76a7639)
- [x] Console Access Keys /access-keys (7fecb69)
- [x] Console Certificate Manager /certificates (7fecb69)
- [x] Console Impersonation Enhanced /admin/impersonate (7fecb69)
- [x] Console Notification Preview /notifications/preview (7fecb69)
- [x] Console Branding /branding (47f12f8)
- [x] Console Webhook Tester /webhooks/test (47f12f8)
- [x] Console User Activity Timeline /users/[id]/activity (47f12f8)
- [x] Console API Explorer Enhanced /api-explorer (47f12f8)
- [x] Console Security Dashboard /security (verified complete — 561 lines)
- [x] Console Monitoring /monitoring (verified complete — 240 lines)
> 50+ console pages implemented across all batches. All assigned frontend tasks complete.

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
- [x] Performance benchmarks (benchmark_test.go — 6 benchmarks, 348d61f)
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
