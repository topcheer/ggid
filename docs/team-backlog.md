# GGID Engineering Team — Autonomous Backlog

**每位 teammate 完成手头任务后，从这个 backlog 中按优先级认领下一个任务。**
**如果 backlog 中的任务都不在你的技能范围，自主研究竞品新功能并创建自己的任务。**

## 自主工作规则

1. 完成一个任务 → commit → 立即认领下一个，不等 arch 分配
2. 每完成 3 个任务，花 5 分钟做竞品研究（web search），发现新功能就实现
3. 如果发现 bug 或技术债，记录在 `docs/tech-debt.md` 并修复
4. 如果发现测试覆盖率低于 70% 的包，主动补充测试
5. 如果发现文档缺失，主动补充

## Task Status Convention
- `[TODO]` — 待认领
- `[IN PROGRESS: <name>]` — 正在做
- `[DONE]` — 完成已提交

---

## Backlog for dev (services/identity, auth, oauth, pkg/authprovider, pkg/social)

### P0 — Core
- [DONE] SAML 2.0 SP-initiated login flow (parse AuthnRequest, create Assertion)
- [DONE] SAML metadata endpoint: GET /saml/metadata
- [DONE] OAuth2 PKCE verification in authorize endpoint (S256 challenge)
- [DONE] Token revocation endpoint: POST /oauth/revoke (RFC 7009)
- [DONE] Back-channel logout: POST /oauth/logout (OIDC Back-Channel Logout 1.0)
- [DONE] OAuth client credentials rotation: rotate client_secret

### P1 — Enterprise
- [DONE] Password history enforcement (reject reused passwords)
- [DONE] Account lockout policy (configurable threshold + duration)
- [DONE] Email verification flow (token + verification endpoint)
- [DONE] Magic link authentication (passwordless email)
- [DONE] Phone-based OTP authentication
- [DONE] LDAP group → role mapping

### P2 — Enhancement
- [DONE] OAuth consent screen (user approves scopes)
- [DONE] JWT claim customization (add custom claims via rules)
- [TODO] Auth service coverage → 85%+ (currently 80.4%)
- [TODO] OAuth service coverage → 70%+ (currently 59.9%)
- [DONE] Social connector: Microsoft, Apple, GitLab, Discord, LinkedIn

### P3 — Innovation
- [TODO] Passkey autofill (WebAuthn conditional mediation)
- [DONE] Step-up authentication (re-challenge for sensitive operations)
- [DONE] Risk-based authentication (IP reputation, device fingerprinting)
- [DONE] Password expiration forced reset (max_age_days policy)

---

## Backlog for dev2 (services/gateway, middleware, router, Docker)

### P0 — Core
- [DONE] Webhook system: registration + HMAC delivery + retry
- [DONE] Prometheus metrics endpoint: GET /metrics
- [DONE] Health check aggregation: GET /healthz returns all backend statuses
- [DONE] Health check split: /healthz/live + /healthz/ready (Kubernetes probes)
- [DONE] Request tracing: X-Request-ID propagation + W3C traceparent spans

### P1 — Security
- [DONE] API key authentication (alternative to JWT for M2M)
- [DONE] IP allowlist middleware (per-tenant configurable)
- [DONE] Bot detection (User-Agent + behavior analysis)
- [DONE] Gateway middleware coverage → 70%+ (currently 71.3%)

### P2 — Performance
- [DONE] Response caching middleware (ETag + conditional GET)
- [DONE] Connection pooling tuning + keep-alive optimization
- [DONE] Request body size limiting middleware
- [DONE] Compression middleware (gzip/brotli)
- [DONE] Per-route timeout configuration (RouteConfig + RouteTimeout)

### P3 — Innovation
- [TODO] GraphQL proxy support
- [DONE] WebSocket proxy support (HTTP hijack → bidirectional TCP tunnel)
- [DONE] Canary deployment routing (percentage-based traffic splitting with header/cookie override)
- [DONE] Circuit breaker pattern (closed/open/half-open with per-backend registry)
- [DONE] Request ID propagation to backends via X-Request-ID header in proxy Director
- [TODO] gRPC-Web protocol translation middleware

---

## Backlog for dev3 (services/policy, org, audit, console pages)

### P0 — Core
- [DONE] Policy engine: ABAC condition evaluation (resource attributes)
- [TODO] Role hierarchy: parent role inherits child permissions
- [TODO] Bulk user-role assignment API
- [TODO] Audit real-time streaming via WebSocket

### P1 — Enterprise
- [TODO] Policy export/import (JSON format for CI/CD integration)
- [TODO] Audit log retention policy + scheduled cleanup
- [DONE] Audit log export (CSV/JSON download)
- [TODO] Org service coverage → 70%+ (currently 65.8%)

### P2 — Enhancement
- [TODO] Console: OAuth client management page
- [TODO] Console: Webhook management page
- [DONE] Console: Audit dashboard with charts (recharts)
- [TODO] Console: User activity timeline
- [TODO] Console: Dark mode support
- [TODO] Console: i18n (Chinese + English)

### P3 — Innovation
- [TODO] Policy decision logging (record every allow/deny)
- [TODO] Permission analyzer (visualize which roles can access what)
- [TODO] Org chart visualization (interactive tree graph)

---

## Backlog for arch (pkg, sdk, console, deploy, docs, CI/CD)

### P0 — Core
- [DONE] Quick start guide: 5-minute integration tutorial
- [DONE] Go SDK: full client implementation (42 tests, 65.8% coverage)
- [DONE] Node.js SDK: TypeScript client + Express middleware
- [DONE] Java SDK: client + exception + README
- [TODO] Docker Compose production hardening (secrets, TLS, volumes)

### P1 — Enterprise
- [TODO] Helm chart for Kubernetes deployment
- [TODO] Terraform module for AWS/GCP deployment
- [TODO] Architecture documentation (C4 model diagrams)
- [TODO] Security whitepaper (threat model + mitigations)

### P2 — Enhancement
- [TODO] Performance benchmark suite (k6 load tests)
- [DONE] Migration guide: Auth0 (docs/migration-from-auth0.md assigned to doc)
- [DONE] Migration guide: Keycloak (docs/migration-from-keycloak.md assigned to doc)
- [DONE] Plugin system design (docs/plugin-api-reference.md + docs/plugin-development.md)
- [TODO] Brand customization (login page theming)

### P3 — Innovation
- [TODO] AI-powered anomaly detection in audit logs
- [TODO] Natural language policy query ("can user X access Y?")
- [TODO] Identity graph visualization (user → roles → orgs → permissions)
