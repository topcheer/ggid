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
- [TODO] SAML 2.0 SP-initiated login flow (parse AuthnRequest, create Assertion)
- [TODO] SAML metadata endpoint: GET /saml/metadata
- [TODO] OAuth2 PKCE verification in authorize endpoint (S256 challenge)
- [TODO] Token revocation endpoint: POST /oauth/revoke (RFC 7009)
- [TODO] Back-channel logout: POST /oauth/logout (OIDC Back-Channel Logout 1.0)
- [TODO] OAuth client credentials rotation: rotate client_secret

### P1 — Enterprise
- [TODO] Password history enforcement (reject reused passwords)
- [TODO] Account lockout policy (configurable threshold + duration)
- [TODO] Email verification flow (token + verification endpoint)
- [TODO] Magic link authentication (passwordless email)
- [TODO] Phone-based OTP authentication
- [TODO] LDAP group → role mapping

### P2 — Enhancement
- [TODO] OAuth consent screen (user approves scopes)
- [TODO] JWT claim customization (add custom claims via rules)
- [TODO] Auth service coverage → 85%+ (currently 74.4%)
- [TODO] OAuth service coverage → 70%+ (currently 47.7%)
- [TODO] Social connector: Microsoft, Apple, GitLab, Discord, LinkedIn

### P3 — Innovation
- [TODO] Passkey autofill (WebAuthn conditional mediation)
- [TODO] Step-up authentication (re-challenge for sensitive operations)
- [TODO] Risk-based authentication (IP reputation, device fingerprinting)

---

## Backlog for dev2 (services/gateway, middleware, router, Docker)

### P0 — Core
- [DONE] Webhook system: registration + HMAC delivery + retry
- [DONE] Prometheus metrics endpoint: GET /metrics
- [IN PROGRESS: dev2] Health check aggregation: GET /healthz returns all backend statuses
- [TODO] Request tracing: X-Request-ID propagation + OpenTelemetry spans

### P1 — Security
- [TODO] API key authentication (alternative to JWT for M2M)
- [TODO] IP allowlist middleware (per-tenant configurable)
- [TODO] Bot detection (User-Agent + behavior analysis)
- [TODO] Gateway middleware coverage → 70%+ (currently 53%)

### P2 — Performance
- [TODO] Response caching middleware (ETag + conditional GET)
- [TODO] Connection pooling tuning + keep-alive optimization
- [TODO] Request body size limiting middleware
- [TODO] Compression middleware (gzip/brotli)

### P3 — Innovation
- [TODO] GraphQL proxy support
- [TODO] WebSocket proxy support
- [TODO] Canary deployment routing (percentage-based traffic splitting)

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
- [IN PROGRESS: dev3] Audit log export (CSV/JSON download)
- [TODO] Org service coverage → 70%+ (currently 65.8%)

### P2 — Enhancement
- [TODO] Console: OAuth client management page
- [TODO] Console: Webhook management page
- [TODO] Console: Audit dashboard with charts (recharts)
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
- [TODO] Quick start guide: 5-minute integration tutorial
- [TODO] Go SDK: full client implementation
- [TODO] Node.js SDK: TypeScript client + Express middleware
- [TODO] Java SDK: Spring Boot starter
- [TODO] Docker Compose production hardening (secrets, TLS, volumes)

### P1 — Enterprise
- [TODO] Helm chart for Kubernetes deployment
- [TODO] Terraform module for AWS/GCP deployment
- [TODO] Architecture documentation (C4 model diagrams)
- [TODO] Security whitepaper (threat model + mitigations)

### P2 — Enhancement
- [TODO] Performance benchmark suite (k6 load tests)
- [TODO] Migration guide from Auth0/Keycloak to GGID
- [TODO] Plugin system design (extension points)
- [TODO] Brand customization (login page theming)

### P3 — Innovation
- [TODO] AI-powered anomaly detection in audit logs
- [TODO] Natural language policy query ("can user X access Y?")
- [TODO] Identity graph visualization (user → roles → orgs → permissions)
