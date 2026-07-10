# GGID Technical Debt

## P0 — Security Debt

### Resolved
- [x] CSRF token predictable entropy → crypto/rand (commit 29b51c1)
- [x] Rate limiter not wired into handler chain (commit fc20c41)
- [x] SecurityHeaders not wired into handler chain (commit 64991a6)
- [x] Tenant spoofing via X-Tenant-ID header (commit 5bcbfce)

### Outstanding
- [ ] OAuth state param not validated on token exchange
- [ ] Gateway HasScope() returns true for any JWT
- [ ] Access token issued without scope claim
- [ ] JWT jti not tracked — tokens fully replayable
- [ ] Webhook HTTPDeliverer has zero SSRF protection
- [ ] No Host header validation (DNS rebinding)
- [ ] JWTSecret empty → silent auth bypass
- [ ] gRPC plaintext between all services
- [ ] No audit hash chain
- [ ] No database backup automation
- [ ] 5/6 WebAuthn attestation formats unverified
- [ ] No password pepper

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
