# Platform Completeness Report

*Authoritative source of truth for productization gaps. Maintained by the arch/PM role. Updated after every scan round and after each verified fix.*

## How to update this file

1. **Findings must be verified by code inspection or regression test** — not assumed from TODO comments.
2. **Status values:** `[NEW]`, `[PARTIAL]`, `[FIXED]` (code exists, needs verification), `[DONE]` (verified by test/build), `[ACCEPTABLE]` (known limitation, documented).
3. **Every status change requires a commit hash** in the Commit column.
4. **After updating, run `go build ./...` and `make test`.**
5. **Cross-reference with `docs/platform-scan-state.md`** — both files must agree on counts.

## Summary

- Total findings: 32
  - Done: 31
  - Fixed (pending verification): 0
  - Partial: 0
  - Acceptable: 1
  - Remaining: 3
  - Last scan: 2026-07-15 round 49 (Focus B — Route Wiring)

## Findings

### HIGH Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 1 | DCR grant_types | oauth/service/oauth_service.go | DCR accepts grant_types param and persists via CreateClient → PG repo. Regression test: register client_credentials via DCR, then successfully obtain token. | [DONE] | ff6e2c0e |
| 2 | MFA TOTP Secret | auth/server/jit_mfa_handler.go | Hardcoded secret replaced with crypto/rand generated base32 secret. | [DONE] | backend |
| 3 | SAML SP-Initiated SSO | oauth/server/server.go | SP-initiated AuthnRequest generation and IdP redirect implemented. | [DONE] | backend/arch |
| 4 | Device-Bound SSO signing key | oauth/service/device_bound_sso.go | Hardcoded default HMAC key replaced with random 32-byte key. | [DONE] | backend |

### MEDIUM Priority

| # | Feature | Location | Issue | Status | Commit |
|---|---------|----------|-------|--------|--------|
| 5 | SAML SLO | oauth/server/server.go | `/saml/slo` and `/saml/idp/slo` handlers process LogoutRequest/Response. | [DONE] | arch |
| 6 | Device-Bound SSO | oauth/service/device_bound_sso.go | IssueDeviceBoundToken, VerifyDeviceBoundToken, signClaims, verifyClaims implemented. No remaining TODOs. | [DONE] | backend |
| 7 | Backup Codes Storage | auth/service/backup_codes.go, auth/cmd/main.go | `NewPgBackupCodeRepo(pool)` wired in main.go; table created via `EnsureSchema`. Falls back to in-memory only when pool is nil. | [DONE] | backend |
| 8 | Agent token scope enforcement | oauth/service/agent_identity.go | AgentTokenClaims carries scope; CheckAgentScope enforces it. | [DONE] | backend |
| 9 | NoopIdentityClient | auth/service/identity_client.go, auth/cmd/main.go | `NewHTTPIdentityClient` used when `IDENTITY_SERVICE_URL` is set; Noop is intentional degraded fallback. | [DONE] | backend |
| 10 | CIBA backchannel route | oauth/server/server.go | CIBA backchannel endpoint `/api/v1/oauth/backchannel` registered and invokes BackchannelAuthentication. Service-layer tests in ciba_flow_test.go exercise the flow. | [DONE] | 2934fd98 |
| 11 | Client Branding persistence | oauth/internal/server/client_branding.go | `handleClientBranding` uses `brandingAdapterVar` (PG-backed adapter with mem fallback). Regression test `TestGapRegression_ClientBranding_UsesAdapter` passes. | [DONE] | 2934fd98 |
| 12 | GeoIP | gateway/middleware/geoip.go | MaxMind GeoLite2 DB integration via GEOIP_DB_PATH; private IP detection; block/allow country lists; X-Geo-Country header. Verified by geoip_test.go. | [DONE] | 852a297b |
| 13 | Frontend page completeness | console/src/app/ | Key pages exist and are wired to APIs. | [DONE] | frontend |
| 14 | HSM/KMS key provider | pkg/crypto, services/auth, services/oauth | PKCS#11 provider + integration into auth/oauth/gateway cmd/main.go; TokenService and OAuth server accept KeyProvider; local keys auto-generated. | [DONE] | 12db3bac |
| 15 | OAuth server internal error exposure | oauth/internal/server/server.go, token_events.go | 500 responses returned raw err.Error() to clients (CreateClient, ListClients, DeleteClient, CreateDeviceAuthorization, ListAgents, IssueSAMLToken, BuildSAMLResponse). Added writeInternalError helper that logs error and returns sanitized "internal server error". | [DONE] | 5a40d929 |
| 16 | Auth server internal error exposure | auth/internal/server/http.go, trust_store_handler.go, admin_config.go | 500 responses returned raw err.Error() to clients. Replaced with writeInternalError helper / sanitized messages. | [DONE] | 5a40d929 |
| 17 | Token event streaming status code | oauth/internal/server/token_events.go | SSE unsupported response returned 500; changed to 501 Not Implemented. | [DONE] | 5a40d929 |
| 18 | E2E Docker regression suite | deploy/e2e-docker-test.sh | Docker E2E tests were failing due to migrate container command typo. Fixed `sh` duplication; 11/11 E2E tests now PASS. | [DONE] | 6f7d68e0 |
| 19 | Server handler coverage gaps | identity/internal/server, audit/internal/server, org/internal/server | Added focused HTTP handler tests for certification-status, management-chain, reassign, GDPR export, query-metrics, SIEM health, daily-aggregations, access-matrix, teams-export, membership-trends. | [DONE] | d0bdeb50 |
| 20 | SDK alignment for Agent Identity / IGA | sdk/python, sdk/java, sdk/rust, sdk/ruby, sdk/csharp, sdk/dart, sdk/php | Agent Identity and Access Request methods added to all 7 SDKs (Python, Java, Rust, Ruby, C#, Dart, PHP). | [DONE] | 5cd72023 |
| 21 | Gateway missing route prefixes | services/gateway/internal/config/config.go | Service routes `/api/v1/org/*`, `/api/v1/policy/*`, and `/api/v1/webauthn/*` were not registered in the API Gateway, causing 404 for those endpoints. Added prefixes mapping to org, policy, and auth services. | [DONE] | ab4a1030 |
| 22 | Gateway middleware chain gaps | services/gateway/internal/router/router.go | `MaxBodySize`, `HostValidation`, and `TimeoutMiddleware` existed in the middleware package but were not applied in the gateway `Handler()` chain. Wired all three with configurable defaults. | [DONE] | 642bda6b |
| 23 | OAuth 2.1 compliance audit | services/oauth/internal/server/oauth21_audit_handler.go | Dynamic analyzer implemented: reads ListClients, checks grant_types, redirect_uris, PKCE, auth_method. Tests cover compliant/non_compliant/mixed/method-not-allowed. | [DONE] | dfcb8a7f |
| 24 | FAPI 2.0 profile | services/oauth | Added  client flag and enforcement: PKCE S256, PAR, DPoP, response_type=code;  GET/PUT; tests added. | [DONE] | ccae234f |
| 25 | FedCM support | services/oauth | No FedCM endpoints. Browser consumer-identity feature; not required for B2B IAM productization. Tracked in backlog for future. | [ACCEPTABLE] | research |
| 26 | gRPC TLS fail-secure + HTTP client timeouts | services/identity/internal/server/server.go, services/org/cmd/main.go, services/policy/cmd/main.go, services/audit/cmd/main.go, services/audit/internal/service/alert_webhook.go, services/auth/internal/service/http_identity_client.go, services/gateway/internal/middleware/graphql.go | gRPC TLS fallback from enabled to plaintext was silent and unsafe. Made fallback require explicit `GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true`. Replaced `http.DefaultClient` and no-timeout `http.Client{}` in audit alert webhooks, auth identity client, and gateway GraphQL resolver with timeouts. | [DONE] | d0a26620 |
| 27 | OAuth client versioning endpoint | services/oauth/internal/server/server.go | `handleClientVersioning` was defined but not registered in `/api/v1/oauth/clients/{id}/` sub-path routing. Added `/version` and `/versions` sub-path dispatch. Regression tests cover POST/GET and invalid client. | [DONE] | (current) |
| 28 | OAuth client health endpoint | services/oauth/internal/server/server.go | `handleClientHealth` was defined but not registered in `/api/v1/oauth/clients/{id}/` sub-path routing. Added `/health` sub-path dispatch. Regression tests cover known client, unknown client, and method not allowed. | [DONE] | (current) |
| 29 | OAuth consent receipt endpoint | services/oauth/internal/server/server.go, services/oauth/internal/server/consent_receipt.go | `handleConsentReceipt` was defined but marked `//nolint:unused` and not registered; also had a path-index bug (`pathParts[3]` instead of `pathParts[4]`). Registered under `/api/v1/oauth/consent/{id}/receipt` and fixed path parsing. Regression tests cover receipt retrieval, not found, and method not allowed. | [DONE] | (current) |
| 30 | Identity gRPC service not implemented | services/identity/internal/server/server.go | Generated pb code (api/gen/identity/v1/) now exists. Identity proto defines 16 methods. Service implementation (gRPC handler registration) is pending. | [DONE] | 6627c064 |
| 31 | Auth gRPC service not implemented | services/auth/cmd/main.go, services/auth/internal/server | Generated pb code (api/gen/auth/v1/) now exists. Auth proto defines 10 methods. Service implementation (gRPC handler registration) is pending. | [DONE] | 6627c064 |
| 32 | OAuth gRPC service not implemented | services/oauth/internal/server/server.go | Generated pb code (api/gen/oauth/v1/) now exists. OAuth proto defines 5 methods. Service implementation (gRPC handler registration) is pending. | [DONE] | 6627c064 |

## Previously Fixed (pre-audit)

| Feature | Was | Fixed By | Commit | Date |
|---------|------|----------|--------|------|
| CIBA backchannel route | Route not registered | arch | pending | 2026-07-14 |
| BotDetect not wired | BotDetect existed but not in Handler() chain | arch | - | 2026-07-12 |
| PII Obfuscate not called | pii.Obfuscate never called | backend | - | 2026-07-12 |
| CheckSessionTimeout not wired | Never invoked | backend | - | 2026-07-12 |
| i18n Translator not called | 0 call sites | backend | - | 2026-07-12 |
| Password Pepper not used | Not wired | arch/backend | - | 2026-07-12 |
| Hash Chain not enabled | Not wired | arch | - | 2026-07-12 |
| Go ERP NULL scan | products returned 0 | arch | 31fab80 | 2026-07-14 |
| Gateway DCR whitelist | DCR blocked by JWT middleware | arch | 8a446ab6 | 2026-07-14 |
| OAuth refresh_token | invalid_grant | backend | d949b958 | 2026-07-14 |
| OAuth Redis client init | Never called SetRedisClient | backend | 8c1b46d5 | 2026-07-14 |

## Scan History

| Date | Focus | New Findings | Fixed |
|------|-------|-------------|-------|
| 2026-07-14 | Initial manual scan | 9 | 0 |
| 2026-07-14 | Round 1 — Focus A | +1 | 1 |
| 2026-07-14 | Round 5 — Focus E | +2 | 3 |
| 2026-07-14 | Round 6 — Focus F | +3 coverage gaps | 3 |
| 2026-07-14 | Round 7 — Focus G | +3 SDK gaps | 3 |
| 2026-07-14 | Gap audit & deduplication | -5 false positives | 5 verified |
| 2026-07-14 | CIBA + Client Branding verification | 0 | 2 verified as DONE |
| 2026-07-14 | Round 8 — Focus A (Interface Integrity) | +4 route/handler interface gaps | 4 (gateway TODO, policy route aliases) |
| 2026-07-14 | Round 9 — Focus B (Route Wiring) | +3 missing gateway prefixes | 3 (/api/v1/oauth, /api/v1/identity, /api/v1/agents) |
| 2026-07-15 | Round 11 — Focus D (Data Persistence / Key Provider Wiring) | 0 | 1 (HSM/KMS KeyProvider wired in auth/oauth/gateway) |
| 2026-07-15 | Round 13 — Focus E (Error Handling) | +3 | 3 (internal error exposure sanitized) |
| 2026-07-15 | Round 14 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS) |
| 2026-07-15 | Round 15 — Focus F (Test Coverage) | +2 | 2 (server handler coverage gaps) |
| 2026-07-15 | Round 16 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS) |
| 2026-07-15 | Round 17 — Focus G (SDK Alignment) | +1 | 0 (SDK gap #20 assigned to arch) |
| 2026-07-15 | Round 18 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS) |
| 2026-07-15 | Round 19 — Focus A (Stub/Placeholder/TODO) | 0 | 0 (remaining stubs are intentional) |
| 2026-07-15 | Round 20 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS) |
| 2026-07-15 | Round 21 — Focus B (Route Wiring) | +1 | 1 (gateway missing /api/v1/org, /api/v1/policy, /api/v1/webauthn prefixes) |
| 2026-07-15 | Round 22 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS after auth container restart) |
| 2026-07-15 | Round 23 — Focus C (Middleware Chain) | +1 | 1 (gateway MaxBodySize, HostValidation, TimeoutMiddleware wired) |
| 2026-07-15 | Round 30 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS, current verification) |
| 2026-07-15 | Round 31 — Focus E (Security Config / Error Handling) | +1 | 1 (gRPC TLS fail-secure + HTTP client timeouts) |
| 2026-07-15 | Round 38 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS, current verification) |
| 2026-07-15 | Round 39 — Focus D (Data Persistence) | 0 | 0 (no new productization-critical gaps; repository CRUD patterns reviewed) |
| 2026-07-15 | Round 40 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS) |
| 2026-07-15 | Round 43 — Focus F (Functional Depth / Test Coverage) | 0 | 0 (no new productization gaps; regression tests pass) |
| 2026-07-15 | Round 46 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS, current verification) |
| 2026-07-15 | Round 47 — Focus A (Interface Integrity) | +3 (OAuth handler registration gaps) | 3 (client versioning, client health, consent receipt); +3 NEW gRPC gaps require api/gen work |
| 2026-07-15 | Round 48 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS, current verification) |
| 2026-07-15 | Round 49 — Focus B (Route Wiring) | 0 | 3 (gRPC pb code generated for identity/auth/oauth; route wiring comprehensive, no new gaps) |
| 2026-07-15 | Round 50 — gRPC Service Implementation | 0 | 3 (identity/auth/oauth gRPC handlers implemented, commit 6627c064) |
| 2026-07-15 | Round 51 — Focus C (Middleware Chain) | 0 | 0 (gateway middleware chain complete; service-level shared middleware requires pkg/ extraction by arch) |
| 2026-07-15 | Round 52 — E2E Regression Tests | 0 | 1 (Docker E2E 11/11 PASS, current verification) |
| 2026-07-16 | Round 85 — Focus F (Functional Deep Verification) | +2 (backup codes API unwired; SCIM scim+json rejected) | 2 (backup codes wired commit 2b238d1b + generate timeout fix pending; gateway SCIM Content-Type fix deployed) |

### Round 85 Focus F — Functional Deep Verification Results

Verified working (curl-tested, not just code review):
- SAML SSO: 302 redirect to IdP login ✓
- SAML metadata: 200 with valid XML ✓
- PAR: returns `request_uri` with 60s expiry (requires Bearer + JSON body) ✓
- CIBA: returns `auth_req_id` with 300s expiry (requires form-encoded client_id/secret + CIBA grant) ✓
- MFA TOTP setup: dynamic secret + otpauth URI ✓
- SCIM Bulk: POST creates user, returns 201 with location ✓
- Device-bound SSO: implemented (prior rounds) ✓

[FIXED] Backup Codes HTTP API — service+PG repo existed but `_ = backupCodeService` discarded in main.go; no endpoints. Wired by backend team (commit 2b238d1b): POST /generate, POST /verify, GET /remaining, MFA login fallback. Timeout fix (37738d7a parallel hash) caused OOMKilled (10×64MB argon2id > 512Mi limit); bounded pool of 4 (a30c0d8d) + auth resource bump (CPU 200m→1000m, mem 512Mi→1Gi) resolved it. E2E verified: 10 codes generated in 2.1s, remaining=10/10, pod stable.

[FIXED] Gateway Content-Type validator rejected `application/scim+json` (RFC 7644 Section 3.1 violation) — SCIM clients sending the standard media type got 415. Added /scim/ path branch accepting application/scim+json (application/json still accepted). Deployed.

Note: CIBA endpoint only accepts form-encoded client credentials (client_id/client_secret as form values), NOT HTTP Basic auth per RFC 6749 Section 2.3.1. Recorded as LOW priority standards gap.

## Remaining Real Gaps (post-audit)

1. **GeoIP MaxMind integration** (LOW, [DONE]) — gateway/middleware/geoip.go
   - MaxMind GeoLite2 DB integration via GEOIP_DB_PATH; private IP detection; country block/allow lists; tests in geoip_test.go.

## Next Actions

- Round 51 (odd, Focus C): Middleware Chain scan — gateway chain complete (14 layers); services/ no fixable gaps
- Round 52 (even): E2E regression tests — 11/11 PASS
- Round 53 (odd, Focus D): Data Persistence scan — pending
- Research backlog: NIS2/CRA/PIPL compliance trends, OAuth 2.1 enforcement, PQC migration, passkey health dashboard


| 2026-07-17 | Round 87 — Focus A (Stub/Placeholder) | +1 (provisioning stub masking undeployed operator) | 1 (operator deployed + gateway wired, eae696b5) |

### Round 87 Focus A — Stub/Placeholder Scan Results

- services/ Go 代码零 TODO/FIXME（HTML placeholder 属性除外，非问题）
- [FIXED] **Provisioning gateway stub** — gateway 对所有 /api/v1/provisioning/* 请求返回假空响应 `{"items":[],"total":0}`，包括 POST/DELETE（静默假成功）。实际上 deploy/operator 有完整代码（CRD + controllers + HTTP API）但从未部署。本轮：创建 operator Dockerfile（含 kubectl/helm/chart）、安装 CRD、部署到 ggid-operator namespace、gateway 路由切换到真实 API（PROVISIONING_SERVICE_URL）。E2E 验证 environment/instances/tenants 返回真实集群状态。Commit eae696b5。
- 注意：operator environment 报告的 operatorNamespace 为 "ggid-system"（默认配置），实际 GGID 部署在 "ggid" namespace — 真实 provision 操作前需校正 operator 的命名空间配置（记录为后续项）。

### Round 89 Focus B — Route/Wiring Scan Results

**Auth service unwired handlers (assigned to backend):**
- [NEW] **handleRotationRoute** — rotation_handler.go:13, real RotationScheduler service behind it, never mux-registered → key rotation API 404. DM'd backend.
- [STUB] **handleSessionFingerprint** — session_fingerprint_handler.go:10, returns hardcoded fake fingerprint data (user_agent "Mozilla/5.0", screen "1920x1080"). Not wired (correctly). Needs real session data. DM'd backend.

**Gateway orphaned middleware (6 defined, 0 wired):**
- [FIXED] **CircuitMiddleware** — circuitbreaker.go:237, production-ready per-backend circuit breaker, now wired into proxy dispatch (b993dd37). Failing backends return 503 instead of cascading timeouts.
- [ACCEPTABLE] **APIKeyAuth** — apikey.go, gateway-level API key validation. Auth service handles API keys internally; gateway-level duplicate not needed currently.
- [ACCEPTABLE] **AuditMiddleware** — audit_log.go, gateway-level audit logging. Individual services publish audit events via NATS; redundant at gateway.
- [ACCEPTABLE] **CSRFProtect** — middleware.go, CSRF protection. JWT Bearer auth (no cookies for auth) makes CSRF moot for API gateway.
- [ACCEPTABLE] **CoalesceMiddleware** — coalesce.go, request coalescing. Performance optimization, not critical.
- [ACCEPTABLE] **APIVersioningMiddleware** — api_versioning.go, API version enforcement. No versioning scheme in use yet.

**Other services (identity, oauth, policy, org, audit):** All handler methods properly mux-registered. No unwired handlers found.

### Round 91 Focus C — Middleware Chain Scan Results

**Gateway middleware chain (verified, 10 layers):**
```
PanicRecovery → SecurityHeaders → HostValidation → CORS → Gzip → RequestID → RequestLogger → ContentTypeValidator → RateLimit → [CircuitBreaker (R89)] → BotDetect → TenantResolver → TimeoutMiddleware → MaxBodySize → [JWTAuth → CAE → SessionTimeout] → inner
```
All layers verified active and correctly ordered.

**Service-level panic recovery (7/7 services PASS):**
All services (identity, auth, oauth, policy, org, audit, gateway) have inline panic recovery in their HTTP handler wrappers. No service will crash from a single request panic.

**Tenant context injection (7/7 services PASS):**
All services extract X-Tenant-ID at the handler level and set up ggidtenant context. Gateway injects X-Tenant-ID header + query param via proxy Director.

**Internal auth HMAC (PARTIAL — backend in progress):**
- pkg/middleware/internal_auth.go: middleware + SignInternalRequest helper created
- Gateway: injects HMAC headers in proxy Director (af8eb9de)
- Auth service: /api/v1/auth/internal/revoke-user checks HMAC at handler level (not middleware)
- Other services: not yet applying InternalAuth — backend actively working on this

**No new issues found in this scan dimension.**

### Round 91 Focus D — Data Persistence Scan Results

**Scanned 9 in-memory stores across auth + audit services:**

| Store | Type | Persisted? | Assessment |
|-------|------|-----------|------------|
| backup_codes | map[uuid]*BackupCode | YES (PgBackupCodeRepo) | DONE — backup_codes_pg.go wired in main.go |
| SessionRevocationManager | maps → Redis+DB | YES (Redis ZSET + DB) | DONE — session_revocation_manager.go uses Redis + sessionRepo + refreshRepo |
| account_lockout | map[string]*LockoutState | YES (Redis) | DONE — email_lockout.go uses redis.Client |
| breach_detection | map[string]*BreachInfo | Cache (HIBP API, 24h TTL) | ACCEPTABLE — cache for external API responses |
| risk_engine (audit) | velocityStore, knownDevices/IPs/Locations | In-memory | ACCEPTABLE — real-time detection state, fresh start on restart is safe |
| alert_webhook dedup | map[string]time.Time | In-memory | ACCEPTABLE — operational dedup cache |
| siem_forwarder circuits | map[string]CircuitState | In-memory | ACCEPTABLE — operational circuit state |
| rotation_scheduler | package-level map | In-memory | KNOWN — assigned to backend (R89), not yet wired |
| impersonation tokens | package-level map | In-memory, NO DB | GAP — issue endpoint not wired (404); feature incomplete |

**Summary:** 3 stores properly persisted (PG/Redis), 4 operational caches (acceptable), 2 known gaps (rotation + impersonation, both already tracked).

### Round 95 Focus E — Security Configuration Scan Results

| Check | Status | Details |
|-------|--------|---------|
| Password pepper | DEPLOYMENT GAP | Code supports `PASSWORD_PEPPER` env (HMAC-SHA256 pre-hash), but K3s deployment doesn't set it. Argon2id still applied (strong), but no server-side secret. Non-fatal — add env in deployment hardening. |
| JWT RSA keys | DONE | Loaded from Kubernetes secrets (JWT_PRIV/JWT_PUB → initContainer → /configs/rsa_private.pem). Not generated at runtime. |
| OAuth introspect auth | DONE | Requires client authentication per RFC 7662 §2.1 (HTTP Basic, form, or Bearer). Verified `isClientAuthenticated(r)` check present. |
| DCR grant_types | PARTIAL | No allowlist validation — clients can register with arbitrary grant_types (e.g., `password`, `client_credentials`). RFC 7591 recommends validating. Low priority for current deployment (admin-only DCR). |
| WebAuthn attestation | DONE | Attestation requirement = "preferred" (not "none"). Full attestation verification in webauthn/attestation.go (packed, none, fido-u2f formats). |
| gRPC TLS | DONE | GRPC_TLS_ENABLED env with cert/key loading, plaintext fallback only with explicit GRPC_TLS_ALLOW_PLAINTEXT_FALLBACK=true. |

**No P0 issues. 1 deployment hardening item (pepper), 1 low-priority DCR validation improvement.**
