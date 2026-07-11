# GGID Team Backlog

> **Last updated**: 2025-01-24 by researcher (research backlog sync)
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
- [x] Coverage sprint25 — error_writer, circuit breaker (737b5f5)
- [ ] Middleware coverage →92% (currently 88.7%)

### arch (Infrastructure security)
- [x] CSRF token predictable entropy (29b51c1)
- [x] Rate limiter wired into handler chain (fc20c41)
- [x] SecurityHeaders wired into handler chain (64991a6)
- [x] Tenant spoofing fix (5bcbfce)
- [x] Webhook SSRF protection (b52bafd)
- [x] Audit hash chain implementation (fe5b025 — hash_chain.go)
- [x] Audit hash chain wired into service startup (74f7feb — AUDIT_HASH_CHAIN_SECRET env var)
- [x] Audit hash chain wired into service startup (74f7feb — SetHashChainSecret + AUDIT_HASH_CHAIN_SECRET env)
- [ ] gRPC TLS/mTLS between services
- [x] JWT key persistence + kid header (loadOrCreatePrivateKey + kid in JWT)
- [x] JWKS endpoint (oauth /oauth/jwks)
- [x] Database backup automation (arch a9b56da — backup.sh + restore.sh)
- [x] JWT key rotation — RotatingKeyProvider with 24h grace period (key_rotation.go)

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
- [x] Password breach check at login — HIBP k-anonymity wired in Login() (f5b8f2c)
- [x] JWT key rotation with grace period — RotatingKeyProvider 24h (f5b8f2c — key_rotation.go)
- [x] PII obfuscation wired in auth/oauth — obfuscateForLog/obfuscateEmail (f5b8f2c)
- [x] Auth service coverage tests — 9 tests in coverage_auth_test.go (f5b8f2c)
- [x] CIBA flow tests — 9 tests in ciba_flow_test.go (f5b8f2c)
- [x] Password breach check at login — HIBP k-anonymity, wired in Login() (f5b8f2c)
- [x] JWT key rotation with grace period — RotatingKeyProvider (f5b8f2c — key_rotation.go)
- [x] Wire pii.Obfuscate() in auth/oauth handlers — obfuscateForLog/obfuscateEmail (f5b8f2c)
- [x] Auth service coverage tests — 10 tests (f5b8f2c — coverage_auth_test.go)
- [x] OAuth CIBA flow tests — 9 tests (f5b8f2c — ciba_flow_test.go)
- [x] Wire RotatingKeyProvider into OAuth startup — 24h grace + ticker (fb29546)
- [x] gRPC TLS config (P0) — pkg/transport/tlsconfig.go LoadServerTLS/LoadClientTLS/LoadMutualTLS (fb29546)
- DUPLICATE: Token exchange RFC 8693 tests — already 8 tests in coverage_boost2/3, sprint14_test.go
- DUPLICATE: DPoP proof verification tests — already 15 tests in dpop_test.go

### uiux
- [x] GraphQL proxy middleware (graphql.go exists)
- [x] WebSocket upgrade support (wsproxy.go exists)
- [x] gRPC server reflection (grpc.go + grpc_interceptor.go exist)
- [x] Deep health check aggregation (/healthz/deep wired in router.go, 348d61f)
- [x] Per-route timeout middleware (route_timeout.go, 348d61f)
- [x] OTel tracing middleware (TracingMiddleware in otel.go:311)
- [x] Performance benchmarks (benchmark_test.go — 6 benchmarks, 348d61f)
- [x] Wire CheckSessionTimeout into middleware chain (737b5f5)
- [x] Request ID propagation (request_id.go + router.go:120,706)
- [x] Gateway integration tests (8 router test files already exist)
- [x] Error response standardization — error_writer.go (737b5f5)
- [ ] Middleware coverage →92% (currently 88.3%)

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
- [x] docs/architecture-decisions.md (commit a0ac967)
- [x] docs/security-architecture.md (commit a0ac967)
- [x] docs/developer-environment.md (commit a0ac967)
- [x] docs/sdk-guide.md (sdk-reference.md)
- [x] docs/development.md (developer-environment.md)
- [x] docs/api-reference.md (expanded, commit 1142575)
- [x] docs/rbac-guide.md (commit 1142575)
- [x] docs/webhook-guide.md (commit 1142575)
- [x] docs/multi-tenancy.md (expanded, commit 1142575)
- [x] docs/api-gateway.md (commit c006ba0)
- [x] docs/changelog.md / CHANGELOG.md (expanded, commit 7f1462f)
- [x] docs/faq.md (expanded, commit 7f1462f)
- [x] docs/contributing.md (verified complete, 620 lines)
- [x] docs/troubleshooting.md (verified complete, 1398 lines)
- [x] docs/database-schema.md (verified complete, 734 lines)
- [x] docs/deployment-guide.md (verified complete, 1311 lines)
- [x] docs/sdk-reference.md (verified complete, 330 lines, all 4 SDKs)
- [x] docs/authentication-guide.md (commit 5061709)
- [x] docs/data-protection.md (commit 5061709)
- [x] docs/incident-response.md (commit 5061709)
- [x] docs/network-security.md (commit 5061709)
- [x] docs/vulnerability-management.md (commit 5061709)
- [x] docs/operations-runbook.md (commit 883583d)
- [x] docs/performance-tuning.md expanded (commit 883583d)
- [x] docs/api-error-codes.md expanded (commit 883583d)
- [x] docs/upgrade-guide.md (verified complete, 438 lines)
- [x] docs/design/adr-001-database-choice.md (commit 883583d)
- [x] docs/design/adr-002-event-driven-audit.md (commit 883583d)
- [x] docs/design/adr-003-provider-chain.md (commit 883583d)
> 139 docs total. All major topics covered. No outstanding doc tasks.

### arch
- [x] SDK coverage tests (sdk/go — 71.4% coverage)
- [x] Docker multi-stage build (deploy/)
- [x] Prometheus /metrics for all services (122873e)
- [x] Structured logging slog for gateway (122873e)
- [x] CI/CD pipeline (GitHub Actions — ci.yml, coverage.yml, release.yml) (commit 22c6e5f)
- [x] Helm chart for Kubernetes (deploy/helm/ggid/ — 12 templates) (commit 22c6e5f)
- [ ] OpenTelemetry integration (otel middleware exists in gateway, needs distributed tracing setup)

## P2 — Quality & Polish

### All
- [ ] Coverage →95% across all packages
- [x] Integration test suite (31 E2E tests across 3 files: e2e_test.go, gateway_e2e_test.go, oauth_e2e_test.go)
- [x] Performance benchmarks (benchmark_test.go — 6 benchmarks, 348d61f)
- [x] Dark mode for Console (already complete — theme.tsx toggle, verified 6ddd6fa)
- [x] Mobile-responsive Console (already complete — sidebar hamburger, breakpoints, verified 6ddd6fa)

## P3 — Future

- [ ] Multi-region active-active deployment
- [ ] Vault/KMS integration
- [ ] FIDO2 certification
- [ ] Compliance certifications (SOC2, ISO27001)
- [ ] Plugin system architecture

## Research — docs/research/ (researcher)

> 137 files, ~119K+ lines total. All docs include Go code examples + GGID source analysis.

### Completed Batches 1-18 (75 docs)
- [x] OAuth/OIDC spec analysis (2.1 migration, PKCE, DPoP, RFC 8693, RFC 8628, RFC 8707, RFC 8414, RFC 8252, RFC 6750, RFC 9700, RFC 7591/7592, PAR/JAR)
- [x] WebAuthn/FIDO (passkey best practices, recovery, roadmap v2, attestation chains, FIDO MDS)
- [x] Security (JWT alg confusion, session fixation, credential theft, anomaly detection ML, API checklist, audit compliance, SIEM)
- [x] Architecture (zero trust, edge IAM, privacy-enhancing tech, compliance automation, gateway patterns, lifecycle automation)
- [x] Competitive (Auth0/Keycloak/GGID matrix, Ory, Clerk/Logto/Casdoor)
- [x] Other (SCIM conformance, multi-tenant SAML, LDAP/AD, OIDC Federation, CAEP, OID4VCI/VP, CIAM, post-quantum)

### Batch 19 — SPA/Mobile/CSRF/SQLi/Certs (5 docs)
- [x] grafeb-spa-security.md (683 lines, 1ef19a7)
- [x] mobile-biometric-iam.md (408 lines, adcd2fb)
- [x] cross-site-request-forgery-iam.md (809 lines, 05c9215)
- [x] sql-injection-iam-defense.md (748 lines, 7a56ae8)
- [x] certificate-pinning-iam.md (932 lines, 6e8d59c)

### Batch 20 — Logout/Replay/RateLimit/DNS/SupplyChain (5 docs)
- [x] openid-connect-logout.md (1085 lines, 8d7ddc1)
- [x] token-replay-defense.md (1081 lines, 1360aaa)
- [x] rate-limiting-iam.md (1267 lines, 5ed7b66)
- [x] dns-rebinding-iam.md (1174 lines, 85453c5)
- [x] supply-chain-iam.md (981 lines, fa5bffc)

### Batch 21 — ZeroTrust/GatewaySec/DataResidency/Secrets/DR (5 docs)
- [x] zero-trust-iam.md (1198 lines, 6529124)
- [x] api-gateway-security.md (1103 lines, 76b707a)
- [x] data-residency-iam.md (1158 lines, 2c6c276)
- [x] secret-management-iam.md (1086 lines, 7b75dc1)
- [x] disaster-recovery-iam.md (1529 lines, ed25bc6)

### Batch 22 — DPoP/AuditChain/Tenant/ gRPC/KeyRotation (5 docs)
- [x] oauth-dpop-support.md (1270 lines, c56dec2)
- [x] audit-tampering-detection.md (1443 lines, a6e776a)
- [x] multi-tenant-isolation.md (1147 lines, be20476)
- [x] grpc-security-iam.md (1129 lines, 987430b)
- [x] key-rotation-iam.md (1373 lines, 9117f26)

### Batch 23 — WebAuthn/State/IP/Password/Scope (5 docs)
- [x] webauthn-attestation-verification.md (1151 lines, 0700747)
- [x] oauth-state-csrf.md (1012 lines, c4e0776)
- [x] ip-reputation-iam.md (1488 lines, 4c533d5)
- [x] password-cracking-defense.md (1041 lines, f8131ce)
- [x] oidc-scope-management.md (1051 lines, ffec9d3)

### Batch 24 — Observability/Federation/Consent/Lifecycle/STRIDE (5 docs)
- [x] observability-iam.md (1114 lines, 57061b7)
- [x] federation-iam.md (1138 lines, 9369ab4)
- [x] consent-management.md (1048 lines, acf119b)
- [x] identity-lifecycle.md (1328 lines, 9e58c13)
- [x] threat-model-iam.md (783 lines, fd0edd5)

### Batch 25 — Passwordless/Introspection/Versioning/Onboarding/AuditCompliance (5 docs)
- [x] passwordless-auth-iam.md (1358 lines, 151986f)
- [x] token-introspection-iam.md (1103 lines, 151986f)
- [x] api-versioning-iam.md (1037 lines, ecc1b2d)
- [x] tenant-onboarding-iam.md (2256 lines, abbba11)
- [x] audit-compliance-iam.md (957 lines, 4909b77)

### Batch 26 — PAM/ABAC/Email/Session/Headless (5 docs)
- [x] priveleged-access-management.md (1443 lines, bf13b27)
- [x] abac-attribute-engine.md (1049 lines, ce9c29f)
- [x] email-security-iam.md (1182 lines, 8e82bf1)
- [x] session-management-iam.md (1183 lines, 8e82bf1)
- [x] headless-auth-iam.md (1553 lines, 8e82bf1)

### Batch 27 — JWT/CA/DLP/IR (4 docs, PKCE dup)
- [x] jwt-claim-validation.md (875 lines, 1ed56a3)
- [x] certificate-authority-iam.md (1351 lines, 82afb5c)
- [x] data-loss-prevention-iam.md (1315 lines, 9f21563)
- [x] incident-response-iam.md (1226 lines, f20e5f0)
- [x] DUPLICATE: oauth-pkce-deep-dive.md already existed (318 lines)

### Batch 28 — ClientCred/TokenExchange/Discovery/Keys/Entropy (5 docs)
- [x] oauth-client-credentials-security.md (916 lines, 7742916)
- [x] token-exchange-iam.md (1028 lines, 3c83071)
- [x] oidc-discovery-security.md (1244 lines, 36235b9)
- [x] access-key-management.md (1403 lines, 26e53da)
- [x] entropy-audit-iam.md (717 lines, 536dca7)

### Batch 29 — SAMLMeta/Logout/Grant/DeviceFlow (4 docs, key-binding dup)
- [x] saml-metadata-security.md (1405 lines, f23744b)
- [x] oidc-back-channel-logout-security.md (1184 lines, 807329b)
- [x] grant-type-validation.md (1053 lines, 346110d)
- [x] oauth-device-flow-security.md (1320 lines)
- [x] DUPLICATE: key-binding-tokens.md (3 DPoP docs exist: 2943+340+1270 lines)

### Batch 30 — MFA/CredentialStuffing/ServiceMesh (3 docs, 2 dups)
- [x] mfa-bypass-prevention.md (1628 lines, ad2902e)
- [x] credential-stuffing-iam.md (1694 lines, 9c7d66b)
- [x] service-mesh-iam.md (1086 lines, d19e123)
- [x] DUPLICATE: oauth-device-flow-security.md (done batch 29)
- [x] DUPLICATE: api-key-lifecycle-iam.md (access-key-management.md covers same topics)

### Batch 31 — CIBA/Rotation/Cookie/Idempotency (4 docs, 1 dup)
- [x] oidc-ciba-security.md (1073 lines, 392001c)
- [x] rotating-credentials-iam.md (1570 lines, 1040497)
- [x] cookie-security-iam.md (1176 lines, 2b9e7d0)
- [x] idempotency-iam.md (1235 lines, d831e7d)
- [x] DUPLICATE: dns-rebinding-iam.md (exists, batch 20, 1174 lines)

### Batch 32 — PPA-ARC/SOC2 (2 docs, 3 dups)
- [x] oidc-ppa-arc.md (1136 lines, f706ca9)
- [x] compliance-soc2-iam.md (1278 lines, 561ecff)
- [x] DUPLICATE: zero-trust-iam.md (exists, batch 21)
- [x] DUPLICATE: iam-disaster-recovery.md (disaster-recovery-iam.md exists, batch 21)
- [x] DUPLICATE: credential-theft-defense.md (already exists)

### Batch 33 — HSM/FIDO2/Gateway (3 docs, 2 dups)
- [x] hsm-kms-integration.md (3965 lines, f5eeefe)
- [x] fido2-certification-guide.md (3026 lines, cb3b772)
- [x] api-gateway-patterns-comparison.md (2640 lines, 5015bd3)
- [x] DUPLICATE: post-quantum-iam.md (post-quantum-cryptography-iam.md exists, 349 lines)
- [x] DUPLICATE: adaptive-authentication.md (adaptive-mfa-design.md 368 lines + step-up-authentication-patterns.md 399 lines)

### Key P0 Findings Driven to Remediation
- [x] CSRF predictable entropy → fixed (29b51c1)
- [x] Rate limiter not wired → fixed (fc20c41)
- [x] Tenant spoofing via header → fixed (5bcbfce)
- [x] SecurityHeaders not wired → fixed (64991a6)
- [x] OAuth state never validated → fixed (72edaa5)
- [x] HasScope() always true → fixed (72edaa5)
- [x] JWTSecret empty bypass → fixed (72edaa5)
- [x] jti replay not tracked → fixed (72edaa5)
- [x] ValidateClientAssertion ParseUnverified → fixed (8098f1c)
- [x] Email template HTML injection → fixed (3399a2a)
- [x] Admin API no role check → fixed (66ef1db/749f809)
- [x] ForgotPassword token leak → fixed (7742916)
- [x] UserRole.ExpiresAt not enforced → fixed (7742916)

### Outstanding P0s Still Open (from research findings)
- [ ] gRPC plaintext between all services (grpc-security-iam.md) — ONLY GENUINELY OPEN P0
- [x] Introspection endpoint auth — fixed (dev ce9c29f, client_secret_basic required)
- [x] JWT key rotation automation — RotatingKeyProvider with 24h grace (dev f5b8f2c — key_rotation.go)
- [x] pii.Obfuscate() zero callers — wired as obfuscateForLog/obfuscateEmail (dev f5b8f2c)
- [x] CheckSessionTimeout dead code — wired into middleware chain (uiux 737b5f5)
- [x] Password breach check at login — HIBP k-anonymity wired in Login() (dev f5b8f2c)

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
