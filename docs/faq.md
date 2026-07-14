# GGID FAQ

Frequently asked questions about the GGID IAM Platform.

---

## General

### Q: What is GGID?

GGID is a production-grade Identity and Access Management (IAM) platform built
in Go. It provides authentication, authorization, user management, audit logging,
and SSO via 7 microservices with a Next.js admin console.

### Q: How is GGID different from Keycloak?

| Aspect | Keycloak | GGID |
|--------|----------|------|
| Language | Java | Go (faster startup, lower memory) |
| Architecture | Monolith | Microservices (independent scaling) |
| Authorization | RBAC only | RBAC + ABAC hybrid engine |
| Multi-tenancy | Realms (separate DB schemas) | RLS (shared tables, tenant isolation) |
| Audit | Database writes | NATS JetStream event pipeline |
| Image size | ~600MB | ~20-35MB per service |
| Startup time | 10-30s | < 2s per service |
| Admin UI | Built-in (Angular) | Next.js 15 (modern, responsive) |
| SDK | Java only | Go, Node.js, Java, Python |

### Q: How is GGID different from Auth0?

| Aspect | Auth0 | GGID |
|--------|-------|------|
| Deployment | SaaS only (cloud) | Self-hosted (Docker/K8s) or SaaS |
| Cost | Per-user pricing | Free (Apache 2.0 open source) |
| Data residency | Auth0's servers | Your infrastructure |
| Customization | Actions (limited JS) | Go plugins + webhooks |
| ABAC | Not supported | Built-in policy engine |
| Vendor lock-in | High | None (open source) |

### Q: What license is GGID released under?

**Apache License 2.0.** You can use it commercially, modify it, and distribute
it freely. No copyleft restrictions (unlike GPL).

### Q: Is GGID production-ready?

GGID v1.0 includes:
- 250+ test cases with 0 failures
- Docker Compose deployment (13 containers)
- Kubernetes Helm chart
- E2E test suite (11/11 passing)
- Security scanning (govulncheck + Trivy)

For enterprise production, review the [Security Hardening Guide](./security-hardening.md).

---

## Authentication & SSO

### Q: What SSO protocols does GGID support?

| Protocol | Status | Notes |
|----------|--------|-------|
| OAuth 2.0 | Production | Authorization Code, Client Credentials, Refresh Token |
| OpenID Connect (OIDC) | Production | Discovery, JWKS, ID tokens |
| SAML 2.0 | Production | Service Provider with metadata exchange |
| LDAP/AD | Production | Auth provider chain (Local + LDAP) |
| SCIM 2.0 | Production | User provisioning (/scim/v2/Users) |

### Q: What social login providers are supported?

Google, GitHub, Discord, LinkedIn, Slack, Microsoft, GitLab, and any
OIDC-compliant provider via the generic OIDC connector.

### Q: Does GGID support MFA?

Yes, three types:
- **TOTP** (RFC 6238) — Google Authenticator, Authy
- **Email OTP** — one-time passwords via email
- **WebAuthn/Passkey** — FIDO2 hardware keys (YubiKey) and platform authenticators

### Q: Can I use passwordless authentication?

Yes. GGID supports magic links (email-based login) and WebAuthn-only accounts
(no password required).

### Q: How does JWT verification work?

The Auth service signs JWTs with RSA 2048-bit keys (RS256). The public key is
published at `/.well-known/jwks.json`. The Gateway and SDKs verify tokens
locally — no per-request call to the Auth service.

---

## Multi-Tenancy

### Q: How does multi-tenancy work?

GGID uses PostgreSQL Row-Level Security (RLS) for tenant isolation:

1. Every table has a `tenant_id UUID NOT NULL` column
2. RLS policies enforce `tenant_id = current_setting('app.tenant_id')`
3. The application sets the tenant context per transaction via `SET LOCAL`
4. Even if application code has a bug, the database prevents cross-tenant access

See [ADR-004](./adr/ADR-004-rls-for-multi-tenancy.md) for the full design.

### Q: Can I use database-per-tenant instead of RLS?

The architecture supports it as a deployment variant for tenants requiring
physical isolation (regulated industries). However, the default deployment
uses shared tables with RLS for simplicity and scalability.

### Q: How many tenants can GGID handle?

The RLS approach has been tested with 10,000+ tenants. Performance depends on
index quality (`tenant_id` as first column in all indexes). The practical
limit is PostgreSQL's storage capacity, not the number of tenants.

---

## Data Storage

### Q: Where is data stored?

| Data Type | Storage |
|-----------|--------|
| User profiles, credentials | PostgreSQL 16 |
| Roles, permissions, policies | PostgreSQL 16 |
| Organizations, memberships | PostgreSQL 16 |
| Audit events | PostgreSQL 16 (via NATS) |
| Rate limit buckets, sessions | Redis 7 |
| Password reset tokens | Redis 7 (TTL-based expiry) |
| Audit event stream (transit) | NATS JetStream (7-day retention) |

### Q: How is data encrypted?

- **At rest:** PostgreSQL TDE or disk-level encryption (LUKS, AWS EBS encryption)
- **In transit:** TLS between all services (configurable)
- **Passwords:** Argon2id (memory-hard, side-channel resistant)
- **Sensitive fields:** AES-256-GCM via `pkg/crypto`

### Q: Do you store passwords in plaintext?

Never. Passwords are hashed with Argon2id before storage. The hash includes
a random salt per password. Even database administrators cannot recover
the original password.

---

## Backup & Recovery

### Q: How do I backup GGID?

See the Deployment Guide for full details:

1. **PostgreSQL** — `pg_dump` daily + WAL archiving for PITR
2. **RSA keys** — store private key in Vault/KMS
3. **Redis** — ephemeral (no backup needed)
4. **NATS** — ephemeral (events are persisted to PostgreSQL by audit consumer)

```bash
# Daily backup script
docker exec ggid-postgres pg_dump -U ggid ggid | gzip > backup_$(date +%Y%m%d).sql.gz
find /backups -name "backup_*.sql.gz" -mtime +30 -delete  # 30-day retention
```

### Q: How do I restore from backup?

```bash
# Restore PostgreSQL
gunzip < backup_20240710.sql.gz | docker exec -i ggid-postgres psql -U ggid -d ggid

# Restore RSA keys
docker cp rsa_private.pem ggid-auth:/configs/rsa_private.pem
docker compose restart auth gateway
```

---

## Scaling

### Q: How do I scale GGID horizontally?

Each service is stateless and can be scaled independently:

```bash
# Docker Compose
docker compose up --scale auth=3 --scale gateway=2

# Kubernetes
kubectl scale deployment ggid-auth --replicas=5
```

The Gateway load-balances across all instances of a service.

### Q: Which services should I scale first?

| Service | Scale When | Reason |
|---------|-----------|--------|
| Gateway | High traffic | Entry point for all requests |
| Auth | Login spikes | CPU-intensive (Argon2id hashing) |
| Identity | User management load | Read-heavy |
| Policy | Frequent permission checks | Compute-intensive |
| Audit | High event volume | NATS consumer throughput |
| Org | Rarely | Low traffic |
| OAuth | Rarely | Low traffic |

### Q: What's the rate limit configuration?

Default Gateway rate limits (per IP):

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/api/v1/auth/login` | 5 | per minute |
| `/api/v1/auth/register` | 3 | per minute |
| `/api/v1/*` | 100 | per minute |

For multi-instance deployments, use Redis-backed rate limiting (see
[Performance Guide](./performance.md)).

---

## Security

### Q: Is GGID SOC 2 / GDPR compliant?

GGID provides features that support compliance:

**SOC 2:** RBAC+ABAC, MFA, audit logging, session management, token revocation

**GDPR:** Data export (CSV/JSON), right to erasure (DELETE cascade),
configurable retention, consent tracking via audit trail

Compliance certification depends on your deployment and organizational policies.
See [Security Hardening Guide](./security-hardening.md).

### Q: How are JWT keys rotated?

```bash
# 1. Generate new key pair
openssl genpkey -algorithm RSA -out new_private.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in new_private.pem -out new_public.pem

# 2. Add new public key to JWKS (dual-key period)
# 3. Restart Auth with new private key
# 4. Wait for old JWTs to expire (1 hour)
# 5. Remove old key from JWKS
```

Recommended rotation frequency: every 90 days.

### Q: Can I revoke a JWT immediately?

Access tokens are short-lived (default 1 hour). For immediate revocation:
- The Auth service maintains a Redis token blocklist
- Blocked tokens are rejected at refresh time
- For instant Gateway-level blocking, reduce access token TTL to 15 minutes

---

## Development

### Q: Can I contribute to GGID?

Yes! See the [Developer Guide](./developer-guide.md) for:
- Code structure and conventions
- Testing strategy
- PR workflow
- File ownership matrix

### Q: What Go version is required?

Go 1.25 or later. The project uses modern Go features including generics
and the latest standard library improvements.

### Q: How do I add a new API endpoint?

See the Developer Guide > Adding a New API Endpoint
for a step-by-step walkthrough.

### Q: How do I generate protobuf code?

```bash
make proto
# Requires buf CLI: https://buf.build
```

---

## Deployment

### Q: What's the minimum hardware requirement?

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 2 cores | 4 cores |
| RAM | 4 GB | 8 GB |
| Disk | 20 GB | 100 GB (SSD) |
| PostgreSQL | 15+ | 16 |
| Docker | 24+ | Latest |

### Q: Can I run GGID without Docker?

Yes. Build binaries and run directly:

```bash
go build -o bin/auth ./services/auth/cmd
DATABASE_URL=postgres://... ./bin/auth
```

You still need PostgreSQL, Redis, and NATS running.

### Q: Does GGID work on Kubernetes?

Yes. A Helm chart is included in `deploy/helm/`:

```bash
helm install ggid deploy/helm/ggid -f values-production.yaml
```

Includes Deployments, Services, Ingress, HPA, PDB, NetworkPolicy, Secrets.

---

## Roadmap

### Q: What features are planned for future releases?

| Phase | Features |
|-------|----------|
| Phase 11 | Risk-based authentication, adaptive MFA, device trust |
| Phase 12 | Plugin SDK (Go plugins), gRPC sidecar pattern, marketplace |
| Future | FIDO2 Enterprise Attestation, step-up auth flows, delegated admin |

### Q: Is there a hosted/SaaS version?

A hosted version is planned. The open-source self-hosted version is fully
functional and production-ready.

### Q: How can I request a feature?

Open an issue on GitHub with the `feature-request` label, or contact the team
via the GGID community channels.

---

## JWT Key Rotation

### Q: How do I rotate JWT signing keys?

Current process (manual):

1. Generate new secret: `openssl rand -base64 32`
2. Update `JWT_SECRET` environment variable on all services
3. Restart all services (existing tokens become invalid)
4. Users must re-authenticate

**Planned improvement**: Support multiple signing keys simultaneously, allowing zero-downtime rotation.

### Q: What happens if JWT_SECRET is empty?

The auth service calls `log.Fatal()` — it will not start. This prevents silent bypass of token verification. Never deploy with an empty `JWT_SECRET`.

### Q: Can I use RS256 (asymmetric) instead of HMAC?

The gateway supports RS256 + JWKS verification. The auth service signs tokens with the configured algorithm. For HMAC (HS256), use `JWT_SECRET`. For RS256, use the private key.

---

## SCIM Provisioning

### Q: How do I configure SCIM provisioning?

SCIM 2.0 endpoints are at `/api/v1/scim/v2/Users` and `/api/v1/scim/v2/Groups`.

1. Create a service account with SCIM permissions
2. Generate a bearer token for the SCIM client
3. Configure your SCIM client (Okta, Azure AD, Workday):
   - SCIM URL: `https://your-ggid.example.com/scim/v2/Users`
   - Auth: Bearer token
   - User filter mapping: `userName` → `email`

### Q: Does SCIM support PATCH operations?

Yes. PATCH follows RFC 7644. The `patchUser` handler uses the `ApplyPatch` engine for replace, add, and remove operations on user attributes.

### Q: How does SCIM deprovisioning work?

When a SCIM client sends `DELETE /scim/v2/Users/{id}` or `PATCH { active: false }`:
1. User's `active` flag is set to false
2. All active sessions are revoked (Redis)
3. All refresh tokens are invalidated
4. User cannot authenticate
5. Webhook `user.deleted` or `user.suspended` is emitted

### Q: Why am I getting 404 on SCIM endpoints?

The SCIM endpoints are registered at `/scim/v2/` under the Identity service. If accessing through the gateway, the full path is `/api/v1/scim/v2/Users`. The gateway must have the route configured.

---

## Docker Deployment

### Q: How do I start the full stack?

```bash
cd deploy && docker compose up -d
sleep 30  # wait for healthchecks
bash deploy/e2e-docker-test.sh  # verify 11/11 tests pass
```

### Q: Why is the NATS healthcheck failing?

NATS must start with the `-m 8222` flag to enable the monitoring endpoint:

```yaml
nats:
  command: ["-m", "8222"]
```

Without this, the healthcheck at `http://localhost:8222/healthz` returns connection refused.

### Q: Why does auth return 429 after a few login attempts?

The auth service rate limits after ~5 failed login attempts per IP within a short window. This is working as designed to prevent brute force attacks.

To clear the rate limit during development:
```bash
docker compose restart auth
```

### Q: Why does register return 500 instead of 409 for duplicate emails?

This was a bug. The auth handler reads the `username` field (not `email`) as the credential identifier. When `username` is empty, all registrations conflict on an empty key.

**Fix**: Always include a unique `username` field in the registration payload.

### Q: Why does create role return 500?

The roles table has `UNIQUE(tenant_id, key)`. An empty `key` conflicts with existing roles.

**Fix**: Always provide a unique `key` field:
```json
{ "name": "Editor", "key": "editor" }
```

### Q: Policy/Org/Audit containers won't start — DB connection error

These services use individual DB environment variables (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`), NOT `DATABASE_URL`. Make sure all 5 variables are set in docker-compose.yml.

---

## OAuth / OIDC

### Q: How do I configure SSO with Azure AD?

1. Register an app in Azure AD
2. Get the client ID and client secret
3. Configure GGID as a relying party:
   ```
   OAUTH_CLIENT_ID=<azure-client-id>
   OAUTH_CLIENT_SECRET=<azure-client-secret>
   OAUTH_ISSUER=https://login.microsoftonline.com/<tenant>/v2.0
   ```
4. Users authenticate at `/api/v1/oauth/authorize?client_id=<id>&redirect_uri=<url>`

### Q: Does GGID support PKCE?

Yes. PKCE (Proof Key for Code Exchange) is **mandatory** in OAuth 2.1. All authorization code flows require a code challenge and verifier.

### Q: How does token introspection work?

`POST /api/v1/oauth/introspect` accepts a client credentials or bearer token and returns the token's active status, scopes, and expiration in RFC 7662 format.

### Q: Can I use DPoP (RFC 9449)?

Yes. GGID validates DPoP proof JWTs for sender-constrained tokens. Include a `DPoP` header with the signed JWT proof on token requests.

---

## Audit & Compliance

### Q: How long are audit events retained?

Default retention is 90 days. Configure via the `AUDIT_RETENTION_DAYS` environment variable. Events older than the retention period are automatically deleted.

### Q: Can I export audit events?

Yes. Use the Audit API:

```bash
curl ".../api/v1/audit/events?from=2025-01-01&to=2025-07-11&format=csv" \
  -H "Authorization: Bearer <JWT>" \
  -o audit_export.csv
```

### Q: Are audit events tamper-proof?

Currently, audit events are stored in PostgreSQL without cryptographic chaining. A hash chain is planned that would make modification detectable.

### Q: Can I forward audit events to a SIEM?

Yes. Configure a webhook with security event types (`auth.login_failed`, `auth.account_locked`). See the [Webhook Guide](webhook-guide.md) for SIEM integration examples (Splunk, ELK, Datadog).

---

*Last updated: 2025-07-11*
