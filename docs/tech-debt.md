# GGID Technical Debt

## P0 — Security Debt

### Resolved (14)
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
- [x] Audit hash chain — HMAC-SHA256 tamper detection (arch fe5b025)
- [x] Host header validation — host_validation.go (uiux)
- [x] Access token scope claim — issueAccessToken scope param (dev — in progress)

### Outstanding (6)
- [ ] OAuth introspection endpoint has NO AUTH (server.go)
- [ ] gRPC plaintext between all services
- [ ] No database backup automation
- [ ] 5/6 WebAuthn attestation formats unverified
- [ ] No password pepper (pepper.go exists, not wired)
- [ ] No JWT signing key rotation infrastructure

## P1 — Architecture Debt

- [ ] Auto-generated RSA keys on boot (multi-instance mismatch)
- [ ] Hardcoded default DB credentials
- [ ] SHA-256 KDF instead of HKDF
- [ ] No JWT signing key rotation infrastructure
- [ ] NATS subject flat (audit.events) — no tenant namespacing
- [ ] Audit service has no RLS
- [ ] JWT stored raw in gRPC interceptor (parsing TODO at line 89)
- [ ] grpc_interceptor.go exists but never wired

## P2 — Code Quality

- [ ] Recurring WIP test file issue — teammates create uncommitted test files
      referencing non-existent functions. Must delete + notify.
- [ ] make test intermittent timeout on OAuth when dev edits concurrently
- [ ] No structured logging in several services
- [ ] Missing input validation on several API endpoints
- [ ] Console pages need loading/error states standardization

## Notes

- WIP test files from uiux: coverage_sprint20_test.go, coverage_sprint21_test.go
  repeatedly broke make test. Pattern: create test → fails to compile →
  arch deletes → uiux recreates. Resolved by uiux learning to vet before commit.

- SCIM test file from dev: coverage_test.go referenced 10+ undefined functions.
  Dev fixed by checking actual function signatures.
