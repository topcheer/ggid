# GGID Team Backlog

## P0 — Security (Critical)

### dev (OAuth/Auth security)
- [ ] OAuth state validation on token exchange — store state in Redis, verify on callback
- [ ] iss parameter in auth redirect (RFC 6749 §10.14)
- [ ] JWT jti tracking with Redis SETNX (anti-replay)
- [ ] HasScope() actual scope enforcement — verify JWT scope claim
- [ ] Access token missing scope claim — issueAccessToken omits scope
- [ ] JWTSecret empty → log.Fatal (silent auth bypass fix)
- [ ] Password pepper (HMAC-SHA256 pre-hash)
- [ ] WebAuthn attestation verification (5/6 formats unverified)

### uiux (Gateway security)
- [ ] Host header validation middleware (DNS rebinding defense)
- [ ] Webhook HTTPDeliverer SSRF protection (private IP blocking)
- [ ] Body size limit middleware (default 10MB)
- [ ] IP allowlist middleware for admin endpoints
- [ ] Middleware coverage →92%

### arch (Infrastructure security)
- [x] CSRF token predictable entropy (crypto/rand fix)
- [x] Rate limiter wired into handler chain
- [x] SecurityHeaders wired into handler chain
- [x] Tenant spoofing fix (JWT claim priority)
- [ ] gRPC TLS/mTLS between services
- [ ] Audit hash chain implementation
- [ ] JWT signing key rotation infrastructure
- [ ] Database backup automation (pg_dump cron)

## P1 — Feature Development

### dev
- [ ] SCIM 2.0 server implementation (create/update/delete users via SCIM)
- [ ] Session management with revocation list
- [ ] SAML IdP-initiated SSO
- [ ] OIDC back-channel logout
- [ ] OAuth DPoP support (RFC 9449)

### uiux
- [ ] GraphQL proxy middleware (ResolveQuery)
- [ ] WebSocket upgrade support in gateway
- [ ] gRPC server reflection for debugging
- [ ] Deep health check aggregation
- [ ] Per-route timeout middleware

### frontend
- [ ] Console SSO Configuration page
- [ ] Console Notification Templates page
- [ ] Console Audit Log Advanced Filters
- [ ] Console User Import Wizard
- [ ] Console Password Policy Editor

### doc
- [ ] docs/architecture.md — system architecture overview
- [ ] docs/database-schema.md — full DB schema
- [ ] docs/deployment-guide.md — production deployment
- [ ] docs/sdk-guide.md — SDK usage guide
- [ ] docs/development.md — developer guide

### arch
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] SDK coverage tests (Go/Node/Java)
- [ ] Docker multi-stage build optimization
- [ ] Helm chart for Kubernetes
- [ ] OpenTelemetry integration

## P2 — Quality & Polish

### All
- [ ] Coverage →95% across all packages
- [ ] Integration test suite (10+ E2E tests)
- [ ] Performance benchmarks
- [ ] API documentation (OpenAPI/Swagger)
- [ ] Upgrade guide documentation
- [ ] Log structured everywhere
- [ ] Error message internationalization
- [ ] Dark mode for Console
- [ ] Mobile-responsive Console

## P3 — Future
- [ ] Multi-region active-active deployment
- [ ] Vault/KMS integration
- [ ] FIDO2 certification
- [ ] Compliance certifications (SOC2, ISO27001)
- [ ] Plugin system architecture
