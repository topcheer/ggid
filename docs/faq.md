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

See the [Deployment Guide](./deployment.md#backup-strategy) for full details:

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

See the [Developer Guide > Adding a New API Endpoint](./developer-guide.md#adding-a-new-api-endpoint)
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
