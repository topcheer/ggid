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
- [x] Coverage sprint26 — gzip, CORS, cb half-open, metrics, deep health (1c23919)
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
- [x] Audit hash chain verification tests — 13 tests: valid chain, tampering detection, gaps, empty/single/large (e3fd91f)
- [x] Java SDK mvn compile — BUILD SUCCESS (18 .class files, no duplicates)
- [x] Auth i18n expansion — 10 new keys for refresh/logout/forgot/reset/sessions (e3fd91f)
- [x] gRPC TLS/mTLS between services — NewGRPCServer/NewGRPCClientDialer with GRPC_TLS_ENABLED (1703849)
- [ ] pkg/crypto coverage — at ceiling 90.5%, Argon2id tests slow with -race
- [ ] auth service coverage — at ceiling 87.5%, 9-dep constructor limits mock-based tests
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
- [x] Wire RotatingKeyProvider into OAuth startup — already done in fb29546
- [x] Fix WebAuthn hardcoded attachment — reads from authenticator response (2870f9d)
- [x] gRPC TLS PoC — pkg/transport/tlsconfig.go already done in fb29546
- [x] Email OTP MFA — EmailOTPService with SendOTP/VerifyOTP, 7 tests
- [x] Wire botdetect into gateway Handler() chain + fix memory leak (097f6a7)
- [x] Wire i18n translator into auth server — writeErrorT(), 10 i18n keys, en/zh-CN/fr locale files (097f6a7)
- [x] Wire pii.Obfuscate into audit InsertEvent — obfuscates ActorName/ResourceName/IP/Metadata (097f6a7)
- [x] CheckSessionTimeout — already wired in gateway router.go:359,367
- [x] Password pepper — PASSWORD_PEPPER env var wired in auth/cmd/main.go → crypto.SetPepper() (PENDING COMMIT)
- [x] OAuth introspection auth — already done (Basic Auth check, server.go:563-572, 590-599)
- [x] Webhook SSRF protection — already done (NewHTTPDeliverer → NewSSRFSafeDeliverer(DefaultSSRFConfig()))
- [x] API error format — structured APIError type + WriteAPIError/WriteSimpleAPIError helpers in pkg/errors/api_error.go (PENDING COMMIT)
- [x] gRPC TLS — NewGRPCServer/NewGRPCClientDialer with GRPC_TLS_ENABLED env var in pkg/transport/grpc_tls.go (PENDING COMMIT)
- [x] Device Auth RFC 8628 — E2E flow tests: full approve flow, denied, invalid, expired, slow_down (7ee1a32)
- [x] Token Exchange RFC 8693 — E2E flow tests: full exchange, missing token/type, invalid, wrong signature, missing sub (7ee1a32)
- [x] Backchannel Logout — E2E flow tests: valid token, empty, missing sub/sid, missing events, nonce, replay, sid (7ee1a32)
- [x] Password breach check configurable disable — BREACH_CHECK_ENABLED env var (41d8064)
- [x] Java SDK de-duplication — Model.java deleted, inner classes extracted to standalone, no more duplicates (7ee1a32)
- [x] Java SDK RS256 verification — JwtVerifier.java created, all 3 filters use JWKS verification, jwks-rsa dep added (7ee1a32)
- [x] hasAdminScope unit tests — 9 dedicated tests in router/gap_regression_admin_test.go (f27f7b3) Gap #14 MEDIUM→HIGH
- [x] Concurrent session limits verification — 7 tests in session_limit_test.go proving EnforceSessionLimit (f27f7b3) Gap #5 PARTIAL→VERIFIED
- [x] OAuth state store Redis migration — RedisCmdable interface, SetRedisClient, Redis GetDel + sync.Map fallback (f27f7b3)
- [x] HIBP password breach circuit breaker — 3-failure threshold, 30s cooldown, fail-open, 7 tests (f27f7b3) Gap #15 MEDIUM→RESOLVED
- [x] SCIM PATCH compliance tests (RFC 7644) — 7 tests: enterprise ext, filter replace, array remove, multi-op sequence (f27f7b3)
- [x] SCIM URN colon notation — parsePatchPath supports RFC 7644 colon notation (urn:...:User:department), 2 new tests (513548b)
- [x] gRPC TLS integration test — self-signed cert, TLS server+client round trip, health RPC call, 5 tests (513548b)
- [x] Webhook delivery retry E2E — 5 tests: retry-then-success (3x503→200), retries-exhausted, first-success, HMAC sig, ctx cancel (513548b)
- [x] OAuth Redis state store integration test — 7 tests: Redis failure fallback, cross-store isolation, Redis recovery, nil Redis, expiry (513548b)
- [x] Auth error i18n — 37 new keys in en/zh-CN, writeAuthErrorT i18n-aware for login/register/refresh/rate-limit errors (513548b)
- [x] PKCE functional verification — 5 tests: full S256 flow, mismatch rejection, plain method, public client enforcement, VerifyCodeChallenge unit tests (07969ea)
- [x] Device Auth functional verification — 8 tests: full flow, pending, denied, expired, invalid code, user code, verification URI, slow_down (07969ea)
- [x] JWKS endpoint functional verification — 7 tests: valid key set, modulus base64url, exponent AQAB, KID match, public key match, key rotation, discovery URI (07969ea)
- [x] Token Introspection auth verification — 6 tests: active+fields, revoked→inactive, expired→inactive, malformed→inactive, empty→inactive, scope field (07969ea)
- [x] gRPC TLS mTLS handshake rejection — rogue cert rejected by RequireAndVerifyClientCert, valid cert accepted (07969ea)
- [x] Token Exchange (RFC 8693) functional verification — 5 tests: delegation flow, actor token, scope reduction, wrong key, expired (6862e9f)
- [x] SAML SSO functional verification — 7 tests: SP metadata, AuthnRequest, unique IDs, redirect encoding, XML round-trip, cert metadata, full flow (6862e9f)
- [x] Multi-tenant RLS verification — 7 tests: tenant isolation, no-tenant rejection, propagation, MustFromContext, isolation levels, spoofing prevention (6862e9f)
- [x] Rate limiter E2E — 3 tests: 100 requests (10x200 + 90x429), 429 JSON body + Retry-After, separate IP buckets (6862e9f)
- [x] WebAuthn functional verification — 7 tests: begin registration/auth challenge, method check, finish without session, list credentials, .well-known endpoints (6862e9f)
- [x] Token Exchange (RFC 8693) functional — 5 tests: delegation flow, actor token, scope reduction, wrong key, expired (6862e9f)
- [x] WebAuthn registration+auth — 7 tests: begin reg/auth challenge, method check, finish no session, list creds, well-known, delete (6862e9f)
- [x] SAML SSO functional — 7 tests: SP metadata, AuthnRequest, unique IDs, redirect encoding, XML marshal, cert metadata, full SP flow (6862e9f)
- [x] Multi-tenant RLS — 7 tests: tenant isolation, no-tenant rejection, propagation, MustFromContext panic, isolation levels, spoofing prevention, settings (6862e9f)
- [x] Rate limiter E2E — 3 tests: 100 requests (10 OK + 90 rate-limited), 429 JSON body + Retry-After, different IPs separate buckets (6862e9f)
- [x] Password pepper functional — 5 tests: full lifecycle, different peppers → different hashes, backward compat, empty no-op, format stable (314a4f3). Already implemented: SetPepper/applyPepper HMAC-SHA256 in pkg/crypto
- [x] Audit hash chain verify — 7 tests: valid chain, tampered event, empty, single, missing hash, replay, different secret (314a4f3)
- [x] OAuth introspection auth — 5 tests: no creds → 401, wrong secret → 401, correct → 200, form-based, missing token (314a4f3). Already implemented: r.BasicAuth() at server.go:563
- [x] gRPC TLS audit wiring — newGRPCServer() helper with GRPC_TLS_ENABLED/CERT/KEY env vars, fallback to plaintext (314a4f3)
- [ ] **MISSING: Per-tenant branding backend API** — console pages exist (settings/branding, settings/branding-custom, branding) but no backend API endpoint for CRUD. Need: GET/PUT /api/v1/tenants/{id}/branding with logo_url, primary_color, custom_css fields.
- [x] Per-tenant branding CRUD API — GET/PUT /api/v1/tenants/{id}/branding, 6 tests (default, update, get-after-update, method, invalid JSON, isolation) (ed2d347)
- [x] Python SDK update_user — added async update_user(token, user_id, email, phone, status) PATCH method (ed2d347)
- [x] Java SDK updateUser — added updateUser(userId, email, phone) + patch() HTTP helper (ed2d347)
- [x] Go SDK typed errors — ALREADY DONE: ErrUnauthorized/ErrForbidden/ErrNotFound/ErrConflict/ErrRateLimited/ErrBadRequest with Is() support in sdk/go/ggid/errors.go (ed2d347)
- [x] **API error format adoption (auth)** — auth service writeError/writeErrorT now use pkg/errors.WriteSimpleAPIError for unified {\"error\":{\"code\",\"message\"}} format (2cd7602). OAuth/identity/policy/org/audit handlers still need migration.

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
- [x] Error response standardization — error_writer.go + all http.Error replaced (737b5f5)
- [x] Request ID propagation tests — request_id_test.go (9e5f2d9)
- [x] Prometheus metrics verification — metrics_test.go (1c23919)
- [x] Gateway route URLs from env vars + UPSTREAM_TIMEOUT (9e5f2d9)
- [x] Full-chain integration tests — full_chain_test.go (2e044b9)
- [x] Full-chain 429 rate limit + proxy test — full_chain_test.go (9d0d30e)
- [x] Metrics label verification — metrics_test.go (9d0d30e)
- [x] Circuit breaker lifecycle tests — coverage_sprint27_test.go (2e044b9)
- [x] Swagger UI at /docs — templates.go + router.go:211 (existing)
- [x] Per-tenant rate limiter isolation — gateway_infra_test.go (existing)
- [x] Content-Type validation middleware — content_type_validator.go (669d026)
- [x] Gzip compression wired into chain — router.go Handler() (669d026)
- [x] Prometheus metrics naming test — metrics_test.go (669d026)
- [x] Structured request logging with tenant_id+remote_addr — recovery.go RequestLogger (existing)
- [x] WebSocket proxy support — wsproxy.go + wsproxy_enhanced.go (existing)
- [x] Content-Type validator tests — content_type_validator_test.go 8 tests (21a2e0a)
- [x] CORS preflight integration test — cors_integration_test.go 3 tests (21a2e0a)
- [x] Bot detection integration test — botdetect_integration_test.go 2 tests (21a2e0a)
- [x] WASM plugin tests — wasm_plugin_test.go (existing)
- [x] /healthz/deep mock backend tests — healthcheck_deep_test.go 4 tests (existing)
- [x] HTTP→HTTPS redirect middleware — https_redirect.go + 5 tests (0f0a019)
- [x] Content-Type validation — content_type_validator.go (existing, 669d026)
- [x] Per-tenant CORS config — per_tenant_cors.go + 5 tests (existing)
- [x] Tiered rate limiting — token_bucket.go TierOverrides + tier_ratelimit_test.go (existing)
- [x] Request size limit — bodysize.go MaxBodySize + bodysize_test.go (existing)
- [x] Middleware chain order test — chain_order_test.go 3 tests (4b952b2)
- [x] pkg/transport gRPC TLS coverage — grpc_tls_test.go 6 tests (4b952b2)
- [x] Panic recovery test — recovery_test.go 5 tests (existing)
- [x] Tenant header injection test — coverage_sprint16_test.go (existing)
- [x] API key auth test — apikey_ipallowlist_test.go (existing)
- [x] QA: docs/examples SDK API fixes — go-integration.md + express-integration.md (e4c671d)
- [x] QA: docs/guides curl commands verified (consistent)
- [x] QA: docs/quickstart steps verified (correct)
- [x] QA: doc link integrity — all .md links resolve (no 404s)
- [x] QA: OpenAPI vs routes — mismatches recorded in tech-debt.md (e4c671d)
- [ ] Middleware coverage →92% (currently 88.3%)
- [ ] API spec coverage audit — openapi.yaml vs router routes

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
- [x] docs/guides/role-based-access.md — RBAC complete guide: roles, permissions, hierarchy, policy check
- [x] docs/guides/abac-policy.md — ABAC guide: policy syntax, dry-run, compliance templates, export/import
- [x] docs/examples/express-integration.md — Full runnable Express.js demo: JWT auth, scope guard, CRUD
- [x] docs/examples/go-integration.md — Full runnable Go server: SDK middleware, RequirePermission, handlers
- [x] docs/quickstart/k3s-deploy.md — Simplified K3s quick deploy (companion to docs/deploy/k3s.md)
- [x] docs/INDEX.md — Complete documentation index: 362 docs organized by category
- [x] docs/CHANGELOG.md — Updated with gap regression (28 tests), i18n refactor, P0 fixes, Docker E2E 11/11
- [x] docs/examples/ verification — express/go integration verified: all SDK API calls match actual source
- [x] docs/research/gap-closure-report.md — Updated with 2026-07-25 regression verification (3 gaps, 28 tests)
- [x] docs/quickstart/docker-5-min.md — Verified: compose commands and ports match deploy/docker-compose.yaml
- [x] QA fix: go-sdk.md, node-sdk.md, sdk-quickstart.md — Fixed SDK API mismatches (NewVerifier→New, expressAuth)
- [x] QA fix: express.md, gin.md integration guides — Updated to match actual SDK exports
- [x] QA fix: developer-onboarding.md, 5-minute-jwt.md — Fixed SDK snippets and io.Reader bug
- [x] docs/quickstart/3-line-integration.md — 3-line JWT auth for Go/Node/Python/Java
- [x] README.md polish — Added 3-line integration section, updated doc links to latest paths
- [x] docs/api/error-codes.md — Verified complete (57 error codes, all services)
- [x] docs/deploy/production-checklist.md — Verified complete (119 lines, TLS/DB/Redis/NATS/Auth)
- [x] docs/guides/sdk-migration-guide.md — Auth0/Keycloak/Firebase migration with API mapping tables
- [x] docs/examples/python-integration.md — Full runnable FastAPI demo: GGIDMiddleware, get_current_user, CRUD
- [x] docs/examples/java-spring-integration.md — Full runnable Spring Boot: GGIDSecurityFilter, SecurityConfig, REST controller
- [x] docs/architecture/security-overview.md — Auth flow, P0 security, multi-tenant RLS, audit hash chain, STRIDE
- [x] fix(scim): isAlpha redeclaration conflict + URN colon notation single-level sub-attribute parsing (commit 513548b)
- [x] docs/deploy/helm-chart-guide.md — Helm chart deployment: values reference, install, upgrade, rollback, production overrides
- [x] docs/guides/troubleshooting.md — Common issues: JWT, DB, NATS, Gateway 502, tenant isolation, OAuth, SCIM, Docker
- [x] docs/architecture/data-flow.md — Request flow diagrams: register, login, JWT verify, policy check, audit pipeline
- [x] docs/examples/go-gin-integration.md — Full runnable Gin app: auth middleware adapter, role/scope guards, CRUD
- [x] docs/research/gap-closure-report.md — Updated: 24 DONE, 3 PARTIAL, 4 TODO (77% closure) + 10 arch verifications
> 213 docs total. All major topics covered.
> Latest batch: rate-limiting guide, event-driven arch, audit-compliance, CHANGELOG update, sdk-quickstart reverified.

## Sprint: SDK Auto-Refresh + Error Format + Discovery (backend 2cd7602)
- [x] API error format adoption (auth) — auth service writeError/writeErrorT migrated to WriteSimpleAPIError for {"error":{"code","message"}} (2cd7602)
- [x] Node SDK middleware security audit — PASS: uses jose jwtVerify with JWKS (createRemoteJWKSet), algorithms: ['RS256'], issuer check. No jwt.decode. No security issues. (2cd7602)
- [x] Go SDK examples — sdk/examples/go/main.go: 3-line integration, role/scope check, permission check, auto-refresh demo (2cd7602)
- [x] Go SDK auto token refresh — TokenManager with 30s margin, transparent refresh on AccessToken(ctx) (2cd7602)
- [x] Gateway OIDC discovery — /.well-known/openid-configuration proxied to OAuth service (2cd7602)

## Sprint: Error Format Migration + SDK Tests + Node Auto-Refresh (backend 2625a95)
- [x] Identity service error format — writeError/writeServiceError migrated to WriteSimpleAPIError/WriteAPIError (2625a95)
- [x] Policy/Org/Audit error format — writeJSONError migrated to WriteSimpleAPIError, writeServiceError to WriteAPIError (2625a95)
- [x] OAuth error format — NOT CHANGED: follows RFC 6749 ({"error":"...","error_description":"..."}) — correct behavior (2625a95)
- [x] Go SDK TokenManager tests — 10 tests: auto-refresh margin, concurrent safety, no-tokens, no-refresh-token, sentinel errors (2625a95)
- [x] Go SDK JWKS cache tests — 7 tests: cache hit, TTL expiry (15min), key rotation, concurrent access, empty/error response (2625a95)
- [x] Node SDK token refresh — TokenManager class (sdk/node/src/token_manager.ts): 30s margin auto-refresh, concurrent dedup, exported from index.ts (2625a95)

### arch
- [x] SDK coverage tests (sdk/go — 71.4% coverage)
- [x] Docker multi-stage build (deploy/)
- [x] Prometheus /metrics for all services (122873e)
- [x] Structured logging slog for gateway (122873e)
- [x] CI/CD pipeline (GitHub Actions — ci.yml, coverage.yml, release.yml) (commit 22c6e5f)
- [x] Helm chart for Kubernetes (deploy/helm/ggid/ — 12 templates) (commit 22c6e5f)
- [x] SDK quickstart examples for Go/Node/Python/Java (dae3339 — sdk/examples/, 5-minute integration)
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
### Batch 34 — PasskeySync/EUDIWallet/CredentialAgent/AIThreat/SGX (5 docs)
- [x] passkey-sync-security.md (2896 lines, 447a397)
- [x] eu-digital-identity-wallet.md (3734 lines, 6ae3d2a)
- [x] credential-agent-architecture.md (3305 lines, 2b0f3cc)
- [x] ai-threat-detection-iam.md (3547 lines, 49fb88e)
- [x] sgx-confidential-computing-iam.md (2843 lines, 6da65de)

- [x] Also fixed: make test failures (83951b0) — NewChecker signature, duplicate test name, circuit breaker timing

### Batch 35 — Competitive Analysis (5 docs)
- [x] auth0-comparison.md (1595 lines, f46d5b5)
- [x] keycloak-comparison.md (1893 lines, 003f8e3)
- [x] ory-comparison.md (2337 lines, 559dd0f)
- [x] casdoor-comparison.md (1352 lines, ae61de2)
- [x] iam-differentiation-strategy.md (1907 lines, 7ef94cf)

### Batch 36 — Gap Validation Research (5 docs)
- [x] gap-closure-report.md (243 lines, 99c6660) — UPDATED with verification: 11 HIGH confidence, 4 MEDIUM, 3 arch-verified
- [x] auth0-quickstart-comparison.md (886 lines, 260f64c)
- [x] sdk-ecosystem-gap-analysis.md (765 lines, 260f64c)
- [x] market-positioning-analysis.md (1188 lines, 8aed6b0)
- [x] i18n-gap-analysis.md (752 lines, 6307bcf)

### Batch 37 — Gap Verification + Competitive Monitoring (5 docs)
- [x] gap-closure-report.md (310 lines, 695c698) — UPDATED: K3s E2E 10/10 verified, 8 deployment issues documented, 24 DONE (77% closure)
- [x] quickstart-timing-comparison.md (889 lines, c3766a5) — GGID 2m35s vs Auth0 8m20s to first JWT (3.2x faster)
- [x] sdk-ecosystem-gap-analysis.md (1114 lines, c3766a5) — VERIFIED: Java SDK has 8 files (NOT vaporware, but won't compile — duplicate classes). Node 90%, Go 85%, Python 65%. Revised from original.
- [x] i18n-wiring-estimate.md (584 lines, 9e737f1) — 937 hardcoded strings across 7 services. auth: 530. ~62.6h / ~4 weeks to wire.
- [x] competitive-update-2026-07.md (576 lines, 3b1065c) — Keycloak 26 workflows+IGA, Auth0 MCP for AI, Casdoor 8 releases, Stytch→Twilio acquisition

### Batch 38 — Strategic Gap Research (3 docs + 2 updates)
- [x] ai-agent-identity-analysis.md (2108 lines, 81dd633)
- [x] bot-protection-analysis.md (960 lines, ac23146)
- [x] iga-workflows-analysis.md (1279 lines, c803b1b)
- [x] gap-closure-report.md UPDATED with 5 new strategic gaps
- [x] team-backlog.md UPDATED with new strategic gaps

### Batch 39 — Gap Verification + Wire Audit (4 docs + 1 update)
- [x] wire-audit.md (681 lines, 9f9ac4e) — 4 unwired components: botdetect (2h), pii.Obfuscate (4h), CheckSessionTimeout (2h), i18n (62h). Total: 70h.
- [x] openapi-audit.md (846 lines, 04cfc1f)
- [x] auth0-top20-benchmark.md (591 lines, 26f6a73) — 13 DONE, 7 PARTIAL, 0 MISSING. 82.5% readiness.
- [x] console-ux-comparison.md (426 lines, 25307e3) — GGID 6.5/10 vs Auth0 8.7 vs Keycloak 4.9. 30 pages, 11 unique features.
- [x] gap-closure-report.md UPDATED — Added 4 wire-audit items as PARTIAL. Total gaps: 39.

### NEW STRATEGIC GAPS (from competitive monitoring 2026-07)
- [x] **[P0]** AI Agent Identity / MCP Auth — IMPLEMENTED (55ffd6f). Agent registration, RFC 8693 token exchange with delegation chain, MCP server auth, 20 tests, 4 HTTP endpoints, gateway routing. (ai-agent-identity-analysis.md)
- [ ] **[P0]** IGA Workflows — Keycloak 26 shipped. GGID has no governance layer. (iga-workflows-analysis.md) — ASSIGNED to backend
- [ ] **[P1]** Bot Protection — Auth0 + Keycloak have full suite. GGID has botdetect.go (coverage unclear). (bot-protection-analysis.md)
- [ ] **[P1]** Zero-Downtime Patches — Keycloak 26 supports. GGID needs rolling update strategy.
- [ ] **[P1]** Device-Bound SSO — Auth0 shipped. GGID has WebAuthn but no device-bound SSO flow.

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
- [x] gRPC plaintext between all services — FIXED (6a0eced). All gRPC services (audit, policy, org) now support GRPC_TLS_ENABLED env var with cert/key loading.
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

## Sprint: Developer Onboarding Polish (frontend beb47d0)
- [x] Onboarding wizard — already complete (640 lines, 5-step: Welcome/Admin/Auth/Users/Review)
- [x] CopyButton component — reusable clipboard component, 3 variants (icon/button/ghost), masked mode for secrets (beb47d0)
- [x] CopyButton integration — OAuth client secret display + API explorer code snippets (beb47d0)
- [x] API health indicator polish — latency tooltip, 5s retry when disconnected, "Reconnecting..." spinner state (beb47d0)
- [x] Console dark mode audit — CSS auto-fallback layer covering ~1500 hardcoded colors across 30+ pages (beb47d0)
- [x] Console i18n completion — hardcoded sidebar labels fixed (Policies, Sessions, SCIM, Org Analytics, Monitoring, API Explorer → t() keys) (beb47d0)

## Sprint: Integration Quality (frontend 2da3a4e)
- [x] Console API integration test — api.test.ts with 15 mock fetch tests (URL, headers, auth, CRUD, errors) (2da3a4e)
- [x] Console build verification — npm run build passes clean, 0 errors, 0 warnings (2da3a4e)
- [x] Accessibility audit — aria-labels added to 25+ icon-only buttons across 15+ pages (2da3a4e)
- [x] Console env vars docs — .env.example created, README.md updated with complete env var table (2da3a4e)
- [x] Performance: code splitting — lazy-charts.tsx dynamic import for recharts (~400KB), 3 pages updated (2da3a4e)

## Sprint: SDK Examples + Console Polish (frontend e86ecfa, 3bf7f9c)
- [x] SDK quickstart examples — go/main.go, node/index.js, python/main.py (3-line JWT pattern, GGID_URL env var, default ggid.iot2.win) (3bf7f9c)
- [x] Console K8s deployment YAML — deploy/k8s/console-deployment.yaml (Deployment+Service+Ingress, ggid-console.iot2.win, probes, resource limits) (e86ecfa)
- [x] Console dark mode verification — CSS auto-fallback covers 79 bg-white patterns, inline styles in branding pages are intentional dynamic theme preview (e86ecfa)
- [x] API error display polish — parseApiError() with structured title/detail/request_id/code, human-friendly status messages (e86ecfa)
- [x] Console build size audit — 3.3MB total across 70+ pages, recharts lazy-loaded, top chunk 365KB is vendor/framework code (e86ecfa)

## Sprint: K3s Deployment + SDK Verification (frontend 006038f)
- [x] Deploy console to K3s — amd64 image built, pushed to registry.iot2.win/ggid/console:latest, pod Running, live at https://ggid-console.iot2.win (HTTP 200) (006038f)
- [x] Console connect to K3s backend — NEXT_PUBLIC_API_URL=https://ggid.iot2.win, CORS configured, SPA client-side requests work (006038f)
- [x] SDK examples verified against K3s — Go/Node/Python all: Login OK (693 chars), Verify OK (subject: 395cbb75...). Fixed: Go JWKS, Node tsconfig, Python __init__/pyproject/clock-skew (006038f)
- [x] API Explorer page — already complete (329 lines, 8 endpoints, curl/JS/Python/Go snippets, try-it with JWT, CopyButton) (e86ecfa)
- [x] Onboarding wizard verification — uses shared useApi().apiFetch(), no hardcoded URLs, NEXT_PUBLIC_API_URL inherited correctly (006038f)

## Sprint: Console Login E2E + SDK Quality (frontend aa8585c)
- [x] Console login flow E2E — fixed critical bug: NEXT_PUBLIC_API_URL not baked into Docker image. Added ARG to Dockerfile, fixed 3 pages using wrong env var. Browser login → dashboard → users all verified against K3s (aa8585c)
- [x] Go SDK quality audit — 3 improvements: (1) structured APIError with parsed title/detail via NewAPIError(), (2) io.ReadAll instead of manual buffer, (3) 60s clock skew tolerance in JWT verify (aa8585c)
- [x] Node SDK build — dist/ verified, npm run build passes, example runs against K3s with jwksUrl (aa8585c)
- [x] Python SDK pip install — clean pip install -e works, pyproject.toml build backend fixed, example runs against K3s (aa8585c)
- [x] Console Users page — verified against K3s, CRUD UI present, empty state correct (backend returns 0 users for list endpoint) (aa8585c)

## Sprint: Frontend i18n Expansion (i18n bf8f58a, this commit)
- [x] Expand i18n dictionary 30→193 keys — top 10 pages covered (organizations, branding, certificates, MFA, SSO, OAuth clients, API keys, tenant config, login flows, security center) (bf8f58a)
- [x] useTranslations() convenience hook + useCallback optimization (bf8f58a)
- [x] LanguageSwitcher component — standalone, compact + full modes (bf8f58a)
- [x] messages/en.json + zh.json synced with nested structure for next-intl migration (bf8f58a)
- [x] Expand i18n 193→280 keys — organizations, SSO, API keys page strings extracted and wired (this commit)
- [x] Organizations page — 18 strings wired to t() (title, tabs, forms, empty states, messages)
- [x] SSO page — 10 strings wired to t() (title, subtitle, wizard headers, provider list)
- [x] API Keys page — 20 strings wired to t() (title, subtitle, table headers, create form, modal)
- [x] All 280 zh-CN translations verified — 0 missing, 3 acronyms (URL, SAML, SCIM) correctly untranslated
- [x] Fixed pre-existing JSX bug in MembersDetail (unclosed div)
- [x] i18n batch 3: branding (13), certificates (18), tenant-config (20), security-center (9), MFA (11), login-flows (5) — 76 strings wired (this commit)
- [x] Total: 352 keys, 0 duplicates, 0 missing zh translations
- [x] i18n batch 4: dashboard (10), login (15), sidebar (2), nav+common keys (22) — 49 new keys, 401 total (this commit)
- [x] i18n batch 5: 8 new namespaces (permissions, policies, groups, sessions, onboarding, notifications, consent, settingsPage) — 505 total keys (ca347a1)
- [x] i18n batch 6: roles.* expanded (22), passwordPolicy (18), audit.* expanded (9) — 551 total keys, wired roles+password-policy pages (c6b16df)

## Sprint: i18n Coordination + Mobile + Build (frontend ef4cfc8)
- [x] Dashboard i18n — already wired to useTranslations() (5 t() calls, no hardcoded strings remaining) (ef4cfc8)
- [x] Users page i18n — already wired to useTranslations() (34 t() calls) (ef4cfc8)
- [x] Console build verification — npm run build: 0 errors, 0 warnings, all 70+ pages prerender (ef4cfc8)
- [x] Console dark mode — CSS auto-fallback active, no contrast issues on main pages (ef4cfc8)
- [x] Mobile responsive — added overflow-x-auto to 3 table pages (activity, monitoring, organizations), added sidebar backdrop overlay, hamburger menu already existed (ef4cfc8)

## Sprint: i18n Hooks + E2E Verification (frontend 29fec8f)
- [x] Console i18n — roles, audit, activity, monitoring pages wired with useI18n() + t() hook (29fec8f)
- [x] User CRUD E2E — Register works (201), list returns empty (backend identity issue). Console shows correct empty state (29fec8f)
- [x] Settings pages verification — all 8 pages return HTTP 200 against K3s (sso, oauth-clients, api-keys, certificates, branding, tenant-config, mfa, login-flows) (29fec8f)
- [x] Sidebar nav verification — all 14 links resolve to existing pages, zero 404s (29fec8f)
- [x] Error handling — gateway scaled to 0, console shows "Reconnecting...", health dots red, empty states rendered, no white screen (29fec8f)

## Sprint: i18n Remaining Pages + E2E Verification (frontend 29fec8f)
- [x] Console i18n — added useI18n hook to roles, audit, activity, monitoring pages (29fec8f)
- [x] User CRUD E2E — API CRUD works (register 201, list 200, get 200). List returns 0 users (backend identity service issue, not console bug) (29fec8f)
- [x] Console settings pages — all 21 settings subpages load HTTP 200 (sso, oauth-clients, api-keys, certificates, branding, tenant-config, mfa, login-flows, + 13 more) (29fec8f)
- [x] Sidebar navigation — all 14 sidebar links resolve to existing pages, 0 missing/404 (29fec8f)
- [x] Console error handling — gateway down → console shows "Reconnecting..." in sidebar, health dots turn red, all panels show empty states, no white screen (29fec8f)

## Sprint: i18n Remaining Pages + Verification (frontend 29fec8f)
- [x] i18n hooks for roles, audit, activity, monitoring — useI18n() + t() added to all 4 pages (29fec8f)
- [x] User CRUD E2E — register works (201 + user_id), list endpoint returns empty (backend store mismatch, not console bug) (29fec8f)
- [x] Settings pages verification — all 8 settings pages return HTTP 200 (sso, oauth-clients, api-keys, certificates, branding, tenant-config, mfa, login-flows) (29fec8f)
- [x] Sidebar navigation — all 14 links verified, no 404s (29fec8f)
- [x] Error handling — gateway scaled to 0: console shows "Reconnecting...", health dots red, empty states. No white screen. Gateway restored. (29fec8f)

## Sprint: Console Polish — Nav Groups + Empty States + Build Fix (frontend cd3d2ed)
- [x] npm run build 0 errors — TSC clean, monitoring error resolved (cd3d2ed)
- [x] Users empty state — helpful hint text added (cd3d2ed)
- [x] Dashboard data validation — stats show 0 when backend stores separated, documented in tech-debt.md (cd3d2ed)
- [x] Console sidebar nav groups — 4 groups: Overview, Management, Security, System (cd3d2ed)
- [x] Login page — already has centered card, logo, remember checkbox, social login (no changes needed)

## Sprint: i18n String Extraction — 5 Pages (frontend b2c7fac)
- [x] policies/page.tsx — 19 keys added to messages (import/export/delete/edit) (b2c7fac)
- [x] settings/branding-custom/page.tsx — 13 keys added (colors/logo/email/password) (b2c7fac)
- [x] audit/page.tsx — 9 new keys added, 6 strings wired (events24h, actor, results) (b2c7fac)
- [x] activity/page.tsx — 12 keys added, 2 strings wired (event types, results) (b2c7fac)
- [x] monitoring/page.tsx — 9 keys added, 10 strings wired (healthy/checking/unhealthy, stats) (b2c7fac)
- [x] gen-i18n-dicts.py regenerated: 611 EN keys, 611 ZH keys (b2c7fac)

## Sprint: i18n Settings Pages — SSO/OAuth/APIKeys/Certs + TSC Fix (frontend 2e46047)
- [x] password-policy TSC fix — added const { t } = useI18n() destructure (2e46047)
- [x] i18n settings/sso — 9 new keys (active/inactive, connected/failed, activate/deactivate, cert) (2e46047)
- [x] i18n settings/oauth-clients — 13 new keys (created/updated/deleted, rotate, edit/delete) (2e46047)
- [x] i18n settings/api-keys — 7 new keys (read/write/admin scopes, expired, selectScope) (2e46047)
- [x] i18n settings/certificates — 4 new keys (valid, expiringSoon, expired, rotated) (2e46047)
- [x] i18n-dicts.ts regenerated: 631 EN + 631 ZH keys, npm run build 0 errors (2e46047)

## Sprint: i18n Wiring — orgs/security/tenant/branding/users (frontend 2a015c7)
- [x] organizations/page.tsx — 13 edits: confirm dialogs wired, form labels (Org/Name/ParentDept/CreatedBy), select options, empty states, table headers (2a015c7)
- [x] security-center/page.tsx — 15 edits: section titles (FailedLogins7d, RiskyIPs, WebAuthnDevices), table headers (IP/Location/Attempts/Last/Risk), device status, MFA labels (2a015c7)
- [x] settings/tenant-config/page.tsx — 25 edits: FEATURE_FLAGS labels, all 12 setMsg calls, 6 save button texts, toggle labels, MFA descriptions (2a015c7)
- [x] settings/branding/page.tsx — 10 edits: setMsg (saved/savedLocal/uploadSvg/fileTooLarge), Email Template Preview title, Template label, summary labels (2a015c7)
- [x] users/page.tsx — 38 edits: all UI labels/buttons/headers, form fields, search, batch toolbar, table headers, pagination, lock/unlock titles (2a015c7)
- [x] en.json + zh.json: 76 new keys (users 28, security 11, tenant 18, branding 0 existing reused)
- [x] gen-i18n-dicts.py: 631 EN / 631 ZH keys, npm run build: 0 errors (2a015c7)

## Sprint: i18n Wiring — SSO/OAuth/APIKeys/Certs (frontend 27dcd38)
- [x] settings/sso — 45 edits: all toast msgs (samlSaved/oidcSaved/providerDeleted/testSucceeded/testFailed), wizard labels (step1Metadata/step2Attributes/step3Certificate), form fields (providerName/entityId/ssoUrl/discoveryUrl/clientId/clientSecret/scopes), action buttons (test/edit/activate/deactivate), social providers (enabled/disabled) (27dcd38)
- [x] settings/oauth-clients — 29 edits: added useTranslations import+hook, all labels (clientName/scopesComma/redirectUrisHint/grantTypes), buttons (registerClient/createClient/saveChanges), secret modal (secretRevealed/secretWarning), table headers, action titles (edit/rotateSecret/delete), msgs (created/updated/deleted/secretRotated) (27dcd38)
- [x] settings/api-keys — 11 edits: error msgs (enterKeyName/selectScope), demo mode, revoke confirm, SCOPE_OPTIONS labels via t(`apiKeys.${scope.value}`), EXPIRY_OPTIONS via t(`apiKeys.expiry${n}`), status badge (expired/activeStatus), never label (27dcd38)
- [x] settings/certificates — 15 edits: STATUS_CONFIG → key-based t(statusCfg.key), upload msgs (selectOrPaste/certUploaded/certUploadedOffline), rotation msgs (keyRotated/keyRotateFailed), certCount, JWKS table headers (keyId/algorithm/status/created), rotation dialog (confirmRotation/rotationConfirmDesc), Active/Rotated badges (27dcd38)
- [x] en.json + zh.json: 95 new keys (sso 18, oauth 9, apiKeys 8, certs 7, common 8)
- [x] Total: 726 keys, npm run build: 0 errors (27dcd38)

## Sprint: i18n Wiring — roles/mfa/login-flows (frontend 94093e3)
- [x] roles/page.tsx — 63 edits across 6 components: main page (header, create/edit forms, error msgs, role cards), PolicyChecker (labels, results), PermissionAssignment (role select, assigned/available perms, batch assign), RolePermissionMatrix (loading, empty states), RoleHierarchyTree (inheritance labels, expand), ABACConditionBuilder (all labels, save/copy). Added useI18n hooks to 5 sub-components. (94093e3)
- [x] settings/mfa — 21 edits: recovery codes section (download/copyAll/copied/warning), webauthn/passkeys (webauthnDesc/registerPasskey/registeredOn), backup MFA (smsBackup/emailBackup/toggleMsgs), TOTP msgs (scanQrPrompt/totpEnrolledSuccess/enterDigitCode), passkey msgs (passkeyRegistered/enterPasskeyName) (94093e3)
- [x] settings/login-flows — 11 edits: flowPreview title, noActiveSteps, step toggles (enableStep/disableStep), conditions panel (ipRangeCidr/riskThreshold/userRole), addStep section, save messages (flowSavedServer/flowSavedLocal) (94093e3)
- [x] en.json + zh.json: 101 new keys (roles 57, mfa 23, flows 12)
- [x] Total: 827 keys, npm run build: 0 errors (94093e3)
- NOTE: dashboard/page.tsx does NOT exist — root page is login redirect. No i18n needed.

## Sprint: Console Quality — responsive/dark-mode/error-handling/loading (frontend e34896c)
- [x] Responsive: sidebar already has mobile toggle + backdrop (md:hidden). Tables verified with overflow-x-auto wrappers. (e34896c)
- [x] Dark mode: profile/page.tsx +28 dark: classes (cards, inputs, tabs, alerts). oauth-clients/page.tsx deduped to settings/oauth-clients redirect (was 282-line copy without dark/i18n). All 72 pages now have dark: coverage. (e34896c)
- [x] Toast/error handling: existing ToastProvider already wired in layout.tsx. parseApiError in api.ts. SSO page uses showToast. Settings pages use setError/setMsg pattern. (existing)
- [x] Loading states: settings/page.tsx has profileLoaded spinner. All pages with API calls have loading state (checked 20+ pages). (existing)
- [x] make test: 0 FAIL (all 3 flaky tests pass on clean run)

## Sprint: 5-Minute Quickstart — login UX/API/onboarding/error/branding (frontend 6eb7070)
- [x] Login UX: already complete — social login buttons (Google/GitHub/SSO from API), remember me checkbox, forgot password link (/forgot-password), MFA challenge flow (TOTP 6-digit step), WebAuthn passkey conditional mediation. (6eb7070)
- [x] API client consistency: all pages import from api-config.ts (API_BASE_URL + DEFAULT_TENANT_ID). api.ts uses apiFetch wrapper with auth headers. Login page uses same config. No hardcoded URLs. (existing)
- [x] Onboarding wizard: created /onboarding 3-step wizard (Create Org → Add User → Get API Key) with progress indicator, skip support, localStorage completion tracking, dark mode. (6eb7070)
- [x] Auth pages: created /forgot-password (password reset with email confirmation) and /register (self-registration with redirect to login). Both use API_BASE_URL + dark mode. (6eb7070)
- [x] Branding backend: settings/branding/page.tsx GET loader tries /api/v1/tenants/current/branding → /api/v1/settings/branding → localStorage fallback chain. POST already saved to /api/v1/settings/branding. (6eb7070)
- [x] Error handling: existing ToastProvider in layout.tsx, parseApiError in api.ts with status maps. All new pages have try/catch with friendly error messages. (6eb7070)
- [x] make test: 0 FAIL, npm run build: 0 errors

## Sprint: PWA + branding verification + dark mode audit (frontend 4979d41)
- [x] npm run build: 0 errors, all routes compile (4979d41)
- [x] Branding backend: GET loader chain connected (/api/v1/tenants/current/branding → /api/v1/settings/branding → localStorage), POST saves to /api/v1/settings/branding (4979d41)
- [x] Dark mode: activity/page.tsx + permissions/page.tsx dark: classes added. All pages now have dark: coverage. (4979d41)
- [x] PWA support: manifest.json (standalone, themeColor, shortcuts), service worker (cache-first static, network-first API), PWARegister component in layout, appleWebApp meta tags (4979d41)
- [x] Page dedup: apikeys/page.tsx → redirect to settings/api-keys (was 450-line dup). oauth-clients already redirected. (4979d41)
- NOTE: ~500 hardcoded English strings remain in low-traffic pages (saml, exports, activity, permissions, notifications/preview). These are secondary pages not in main sidebar navigation. Core pages (dashboard, users, roles, all settings/*) are fully i18n'd with 827 keys.

## Sprint: Final quality polish (frontend cb13e9b)
- [x] E2E verified: Console live at https://ggid-console.iot2.win (HTTP 200), Gateway healthy, register→login→MFA challenge flow works end-to-end. (cb13e9b)
- [x] Performance: largest chunk 365KB (under 500KB threshold). Total static 3.3MB. recharts already lazy-loaded via next/dynamic. No code splitting needed. (cb13e9b)
- [x] Accessibility: focus-visible outline ring globally, skip-to-content link, main content id. 77 focus styles across pages. 5 aria-labels on key pages. (cb13e9b)
- [x] Mobile: sidebar has mobile toggle + backdrop (md:hidden). Tables use overflow-x-auto wrappers. Forms are responsive (sm:col-span-2). (existing, verified)
- [x] README: updated with Docker build instructions, PWA section, NEXT_PUBLIC_API_URL build-arg docs. (cb13e9b)
- [x] npm run build: 0 errors, make test: 0 FAIL
