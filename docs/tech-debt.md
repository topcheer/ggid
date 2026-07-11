# GGID Technical Debt

## P0 — Security Debt

### Resolved (20)
- [x] CSRF token predictable entropy → crypto/rand (arch 29b51c1)
- [x] Rate limiter not wired into handler chain (arch fc20c41)
- [x] SecurityHeaders not wired into handler chain (arch 64991a6)
- [x] Tenant spoofing via X-Tenant-ID header (arch 5bcbfce)
- [x] Admin API no role check — any authenticated user (arch 66ef1db)
- [x] OAuth state param not validated on token exchange (dev 72edaa5)
- [x] No iss parameter in auth redirect (dev 72edaa5)
- [x] JWTSecret empty → silent auth bypass (dev 72edaa5)
- [x] HasScope() always returns true for JWT (dev 72edaa5)
- [x] JWT jti not tracked — tokens replayable (dev 72edaa5)
- [x] Webhook HTTPDeliverer SSRF — SSRF-safe transport (arch b52bafd)
- [x] Audit hash chain — HMAC-SHA256 tamper detection (arch fe5b025, wired in repository layer)
- [x] Host header validation — host_validation.go (uiux ed4124b)
- [x] Access token scope claim — issueAccessToken scope param (dev b8bfa05)
- [x] Password pepper — SetPepper() + applyPepper() wired (dev ce9c29f)
- [x] WebAuthn attestation verification — 6 format verifiers (dev ce9c29f)
- [x] OAuth introspection endpoint auth — client_secret_basic required (dev ce9c29f)
- [x] ValidateClientAssertion JWT signature verification (dev 8098f1c)
- [x] ForgotPassword token leak — tokens removed from HTTP response (dev 7742916)
- [x] UserRole.ExpiresAt enforcement — evaluator filters expired roles (dev 7742916)
- [x] Database backup automation — backup.sh + restore.sh (arch a9b56da)

### Outstanding (3)
- [ ] gRPC plaintext between all services (need mTLS; researched in grpc-security-iam.md)
- [ ] JWT key rotation automation (key_rotation.go in progress by dev)
- [ ] Password breach check at login (HIBP API integration; dev assigned)

## P1 — Architecture Debt

### Resolved
- [x] JWT key persistence + kid header (loadOrCreatePrivateKey)
- [x] JWKS endpoint (oauth /oauth/jwks)
- [x] Deep health check aggregation (/healthz/deep)
- [x] Per-route timeout middleware
- [x] Circuit breaker per backend
- [x] IP allowlist for admin endpoints
- [x] Body size limit middleware
- [x] GraphQL proxy middleware
- [x] WebSocket upgrade support
- [x] gRPC server reflection

### Outstanding
- [ ] Auto-generated RSA keys on boot (multi-instance mismatch — need shared key store)
- [ ] Hardcoded default DB credentials (need env-only in production)
- [ ] SHA-256 KDF instead of HKDF
- [ ] NATS subject flat (audit.events) — no tenant namespacing
- [ ] Audit service has no RLS
- [ ] JWT stored raw in gRPC interceptor (parsing TODO at line 89)
- [ ] grpc_interceptor.go exists but never wired
- [ ] pii.Obfuscate() zero callers in production code (dev working on pii_logging.go)
- [ ] CheckSessionTimeout dead code not wired (uiux assigned)

## P2 — Code Quality

### Resolved
- [x] Console pages loading/error states — reusable components created (frontend 6ddd6fa)
- [x] API config for Docker deployment — lib/api-config.ts (frontend 6ddd6fa)

### Outstanding
- [ ] Coverage →95% across all packages (currently: errors/tenant/i18n 100%, authprovider 97%, pii 96.6%, notification 95.7%)
- [ ] make test intermittent timeout when teammates edit concurrently
- [ ] Missing input validation on several API endpoints
- [x] Dark mode needs testing across all Console pages — verified dashboard, users, settings, organizations in dark mode via browser. CSS auto-fallback layer covers 77 bg-white patterns. No contrast issues. (frontend 29fec8f)

## P3 — Infrastructure

### Resolved
- [x] CI/CD pipeline (ci.yml, coverage.yml, release.yml)
- [x] Helm chart (deploy/helm/ggid/ — 12 templates)
- [x] Docker multi-stage build for all 7 services + console
- [x] Prometheus /metrics for all services
- [x] Structured logging slog for gateway
- [x] Database backup automation scripts
- [x] Integration test suite (31 E2E tests across 3 files)

### Outstanding
- [ ] OpenTelemetry distributed tracing (otel middleware exists, needs collector setup)
- [ ] Multi-region active-active deployment
- [ ] Vault/KMS integration
- [ ] FIDO2 certification
- [ ] Plugin system architecture

## Notes

- WIP test files: teammates occasionally create uncommitted test files
  referencing non-existent functions. Pattern resolved by requiring go vet
  before commit and backlog maintenance rules.

- SCIM test file from dev: coverage_test.go referenced 10+ undefined functions.
  Dev fixed by checking actual function signatures.

- Recurring lesson: always run `go build ./...` before `go test` to catch
  compilation errors from concurrent edits.
