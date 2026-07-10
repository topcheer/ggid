# GGID Security Audit Checklist

OWASP Top 10 (2021) alignment for the GGID IAM Platform.

---

## How to Use

For each OWASP category:
1. Review the **GGID Protection** column
2. Run the **Verification Method** to confirm
3. Mark status: **Pass / Fail / N/A**

---

## A01:2021 — Broken Access Control

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| Tenant isolation | PostgreSQL RLS (`FORCE ROW LEVEL SECURITY`) on all tables | `SELECT relrowsecurity FROM pg_class WHERE relname='users'` returns `t` | |
| JWT verification | Gateway verifies RS256 signature via JWKS before any API access | `curl -H "Authorization: Bearer invalid" $GW/api/v1/users` → 401 | |
| RBAC enforcement | Roles + permissions checked via Policy API | Create user without `users:write` role, attempt POST → 403 | |
| ABAC conditions | Attribute-based policies with deny-override | Configure deny policy, verify access blocked | |
| IDOR prevention | All resource access scoped by `tenant_id` from JWT | Attempt to GET user from different tenant → 404 | |
| API rate limiting | Per-IP (login: 5/min) + per-tenant (configurable) | Send 10 rapid logins → 429 after 5th | |

## A02:2021 — Cryptographic Failures

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| Password hashing | Argon2id (memory-hard, RFC 9106) | Check `credentials.secret` format starts with `$argon2id$` | |
| JWT signing | RSA 2048-bit (RS256) | Decode JWT header: `alg: RS256` | |
| TLS in transit | Configurable TLS at ingress (nginx/Caddy) | `curl -v https://iam.example.com` shows TLS 1.2+ | |
| DB encryption at rest | Disk-level (LUKS/EBS) or PostgreSQL TDE | Verify PostgreSQL data directory is on encrypted volume | |
| Redis password | `requirepass` enforced | `redis-cli -a wrongpass ping` → AUTH failed | |
| Sensitive field encryption | AES-256-GCM via `pkg/crypto` | Review code: `crypto.Encrypt()` used for sensitive fields | |

## A03:2021 — Injection

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| SQL injection | Parameterized queries via pgx v5 (`$1`, `$2`) | `grep -r "fmt.Sprintf.*SELECT" services/` — only in `SET LOCAL` with UUIDs | |
| NoSQL injection | N/A (no NoSQL databases) | — | N/A |
| LDAP injection | `go-ldap` escapes filter values | Review LDAP filter construction in `authprovider` | |
| Command injection | No `os/exec` calls with user input | `grep -r "os/exec" services/ pkg/` | |
| XSS | React/Next.js auto-escaping, CSP headers | `grep -r "dangerouslySetInnerHTML" console/` | |

## A04:2021 — Insecure Design

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| Threat modeling | STRIDE analysis in architecture docs | Review `docs/design/` | |
| Defense in depth | RLS + app-level tenant_id + Gateway injection | Test: disable app-level filter, RLS still blocks | |
| Fail-safe defaults | Default-deny policy engine (configurable) | With no roles, all permission checks return `false` | |
| Rate limiting | Auth endpoints rate-limited by default | 5 login attempts → 429 | |

## A05:2021 — Security Misconfiguration

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| Default credentials | No hardcoded passwords; all via env vars | `grep -rn "password" services/ --include="*.go" | grep -v "Password\|password\|_test"` | |
| Debug mode off | No debug endpoints in production | `curl $GW/debug/pprof` → 404 | |
| CORS whitelist | Configurable origins (no `*` in prod) | Check Gateway config: `GATEWAY_CORS_ORIGINS` | |
| Error handling | Generic errors to client, detailed logs server-side | Trigger 500 → response has no stack trace | |
| Security headers | HSTS, X-Frame-Options, X-Content-Type-Options | `curl -I $GW` check response headers | |

## A06:2021 — Vulnerable and Outdated Components

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| Dependency scanning | `govulncheck` in CI | `govulncheck ./...` → 0 vulns | |
| Container scanning | Trivy in CI | `trivy image deploy-auth:latest` | |
| Go version | 1.25+ (latest) | `go version` | |
| Dependencies at @latest | No pinned outdated versions | `go list -m all \| grep -v latest` | |

## A07:2021 — Identification and Authentication Failures

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| Password complexity | Configurable policy (min 8, upper/lower/digit/special) | Try register with `weak` → 400 | |
| Account lockout | Rate limiting (5 failures → lockout) | Send 5 wrong passwords → 429 | |
| Session management | JWT (1h TTL) + refresh rotation | Refresh a token → old refresh invalid | |
| MFA | TOTP + WebAuthn + Email OTP | Enable MFA, login without code → requires MFA step | |
| Credential recovery | Token-based reset with expiry | `POST /password/forgot` → always 200 (no enumeration) | |

## A08:2021 — Software and Data Integrity Failures

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| CI/CD pipeline | All code passes `go build` + `make test` before merge | Run `make test` | |
| Audit log integrity | Hash chain verification (`GET /audit/integrity`) | Call integrity endpoint | |
| Hook HMAC signing | Webhook payloads signed with HMAC-SHA256 | Verify `X-GGID-Signature` header | |
| Dependency integrity | `go.sum` verifies module hashes | `go mod verify` | |

## A09:2021 — Security Logging and Monitoring Failures

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| Audit logging | All security events via NATS → PostgreSQL | Login → check audit events | |
| Log format | Structured JSON to stdout | `docker logs ggid-auth \| jq .` | |
| Monitoring | Prometheus metrics + Grafana dashboard | `curl $GW/metrics` | |
| Alerting | Prometheus alert rules (latency, error rate) | Check `deploy/prometheus-alerts.yaml` | |
| Anomaly detection | Audit rules API | `GET /api/v1/audit/rules` | |

## A10:2021 — Server-Side Request Forgery (SSRF)

| Control | GGID Protection | Verification | Status |
|---------|----------------|-------------|--------|
| No outbound requests from user input | Services don't fetch URLs from user data | `grep -r "http.Get\|http.Post" services/` — only in webhook calls | |
| Webhook URL validation | Admin-only configuration, not user-provided | Webhook URLs set via authenticated admin API | |
| Internal network protection | Docker network isolation / K8s NetworkPolicies | Review `deploy/docker-compose.yaml` network config | |

---

## Remediation Priority

| Priority | Action | Deadline |
|----------|--------|----------|
| Critical | Fix any Fail items in A01 (Access Control) | Immediate |
| High | Fix any Fail items in A02 (Crypto) | 1 week |
| High | Fix any Fail items in A07 (Auth) | 1 week |
| Medium | Fix any Fail items in A05 (Misconfig) | 2 weeks |
| Low | Fix any Fail items in A06 (Dependencies) | Monthly |

---

## References

- [OWASP Top 10 (2021)](https://owasp.org/Top10/)
- [GGID Security Hardening Guide](./security-hardening.md)
- [GGID ADR-003: JWT](./adr/ADR-003-jwt-over-server-sessions.md)
- [GGID ADR-004: RLS](./adr/ADR-004-rls-for-multi-tenancy.md)
