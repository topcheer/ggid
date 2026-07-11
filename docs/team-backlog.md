# GGID Team Backlog

> Last updated: session audit — reflects actual codebase state

## P0 — Security (Critical)

### Resolved (13/20)
- [x] CSRF token predictable entropy → crypto/rand (arch 29b51c1)
- [x] Rate limiter wired into handler chain (arch fc20c41)
- [x] SecurityHeaders wired into handler chain (arch 64991a6)
- [x] Tenant spoofing fix (arch 5bcbfce)
- [x] Admin API role check (arch 66ef1db)
- [x] OAuth state validation on token exchange (dev 72edaa5)
- [x] iss parameter in auth redirect (dev 72edaa5)
- [x] JWT jti tracking with Redis SETNX (dev 72edaa5)
- [x] HasScope() scope enforcement (dev 72edaa5)
- [x] JWTSecret empty → log.Fatal (dev 72edaa5)
- [x] Webhook SSRF protection (arch b52bafd)
- [x] Host header validation middleware (uiux — host_validation.go exists)
- [x] Audit hash chain (arch fe5b025 — hash_chain.go, 11 tests)

### Outstanding (7) — VERIFIED MISSING
1. **OAuth introspection endpoint has NO AUTH** — server.go introspection handler requires no client credentials (dev)
2. **Access token missing scope claim** — issueAccessToken omits scope in JWT claims (dev — in progress)
3. **Password pepper not wired** — SetPepper() exists but not called at boot (dev)
4. **gRPC TLS/mTLS between services** — all gRPC is plaintext (arch)
5. **WebAuthn attestation verification** — attestation_formats.go exists but 5/6 formats unverified (dev)
6. **JWT signing key rotation** — no kid header, no JWKS rotation (arch)
7. **Database backup automation** — no pg_dump cron (arch)

## P1 — Feature Development

### Completed (VERIFIED EXISTING)
- [x] SCIM 2.0 PATCH support — services/identity/internal/scim/patch.go + filter.go (dev 2caa572)
- [x] Session timeout middleware — session_timeout.go (dev 2caa572)
- [x] SAML IdP-initiated SSO — pkg/saml/idp_initiated.go (dev 2caa572)
- [x] OAuth DPoP support — services/oauth/internal/service/dpop.go (dev 2caa572)
- [x] TOTP backup codes — backup_codes.go (dev 2caa572)
- [x] GraphQL proxy middleware — graphql.go (uiux)
- [x] WebSocket upgrade — wsproxy.go (uiux)
- [x] gRPC server reflection — grpc_interceptor.go (uiux)
- [x] Circuit breaker per backend — circuitbreaker.go (uiux)
- [x] IP allowlist middleware — ipallowlist.go (uiux)
- [x] Prometheus /metrics all services (uiux 122873e)
- [x] Structured logging (slog) — 6 gateway middleware files (uiux 122873e)
- [x] Console: all 33 pages (frontend — password-policy, user-import, audit, notifications, SSO, branding, API explorer, certs, etc.)
- [x] Console: security dashboard, webhook tester, user activity
- [x] Docs: api-reference, architecture, developer-environment, security-architecture, admin-guide, audit-guide, configuration, 128 total
- [x] Research: 135 docs covering all major IAM topics

### Outstanding — VERIFIED MISSING
1. **OIDC back-channel logout** — no logout_token JWT validation (dev)
2. **Deep health check aggregation** — no aggregate endpoint (uiux)
3. **Per-route timeout middleware** — no configurable per-route timeouts (uiux)
4. **CI/CD pipeline** — no .github/workflows (arch)
5. **Helm chart** — no deploy/helm/ (arch)
6. **Performance benchmarks** — no bench tests (all)

## P2 — Quality & Polish

### Outstanding
- [ ] Coverage →95% (currently avg ~90%, scim 65%, webauthn 75%)
- [ ] OpenAPI/Swagger spec generation
- [ ] Mobile-responsive Console
- [ ] Dark mode Console

## Assignment Rules

1. **Before assigning any task**: check if the file/feature already exists with `ls` or `grep`
2. **Update this file** when tasks complete — move items from Outstanding to Completed
3. **Stay in your lane**: dev owns oauth/auth, uiux owns gateway/middleware, arch owns shared pkg + infra
4. **Never edit another teammate's files without coordination**
