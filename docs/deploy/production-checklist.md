# Production Deployment Checklist

> Pre-flight checklist for production GGID deployments. Every item MUST be verified before go-live.

---

## Critical (Must Fix Before Launch)

### Secrets & Encryption

- [ ] **JWT_SECRET** is 32+ random characters (`openssl rand -base64 32`), shared across gateway/auth/oauth
- [ ] **JWT_SECRET** stored in secrets manager (Vault, AWS Secrets Manager, K8s Secret)
- [ ] **PASSWORD_PEPPER** is set (32+ random chars, different from JWT_SECRET)
- [ ] **DB password** is strong (not `ggid`), stored in secrets manager
- [ ] **Redis password** is set (not default)
- [ ] No secrets in source code, Docker images, or environment files committed to git

### Database

- [ ] **DB user is NOT superuser** — create a dedicated role with limited privileges
- [ ] **RLS enabled and forced** on all tenant-scoped tables
- [ ] **DB connections use SSL** (`sslmode=verify-full` in production)
- [ ] **Backups automated** — daily `pg_dump` + S3 upload (reference: `deploy/scripts/backup.sh`)
- [ ] **Backup encryption** — AES-256-CBC backup file encryption
- [ ] **Backup restore tested** — verified at least once

### Network & TLS

- [ ] **TLS 1.3 enforced** at load balancer / ingress
- [ ] **TLS certificate** valid, auto-renewal configured (cert-manager / Let's Encrypt)
- [ ] **HSTS header** enabled (`Strict-Transport-Security: max-age=31536000; includeSubDomains`)
- [ ] **Security headers** present (X-Content-Type-Options, X-Frame-Options, CSP)
- [ ] **Internal service traffic** isolated (VPC, private subnet, or mTLS)
- [ ] **NATS authenticated** — NATS auth required (not open access)

### Authentication & Authorization

- [ ] **MFA enforced** for admin users (TOTP or WebAuthn)
- [ ] **Rate limiting enabled** and tuned (`RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`)
- [ ] **CORS restricted** to known origins (not `*`)
- [ ] **RBAC roles follow least privilege** — no admin role assigned to regular users
- [ ] **Default tenant admin password** changed from default

---

## Recommended (Strongly Advised)

### Monitoring & Alerting

- [ ] **Prometheus `/metrics` endpoint** scraped by monitoring system
- [ ] **Health check alerting** — `/healthz/deep` monitored, alert on non-200
- [ ] **Log aggregation** — logs shipped to ELK, Loki, or CloudWatch
- [ ] **Audit log retention** configured (default 90 days, adjust per compliance)
- [ ] **Grafana dashboards** created (request rate, error rate, latency, DB connections)

### Performance & Scaling

- [ ] **DB connection pool** sized correctly (max_connections vs pool size)
- [ ] **Redis maxmemory policy** set (`allkeys-lru` recommended)
- [ ] **NATS JetStream** storage sized for expected audit volume
- [ ] **HPA enabled** for stateless services (gateway, auth)
- [ ] **Resource limits** set for all containers/pods

### Security Hardening

- [ ] **Webhook SSRF protection** enabled (blocks private IP ranges)
- [ ] **WebAuthn attestation verification** enabled
- [ ] **OAuth introspection endpoint** requires client authentication
- [ ] **Password policy** enforced (min length, complexity, breach check)
- [ ] **Session timeout** configured (8 hours active, 7 days max)
- [ ] **Circuit breaker** enabled and tuned

### Operational

- [ ] **Runbook** accessible to on-call team (reference: `docs/operations-runbook.md`)
- [ ] **Incident response plan** documented (reference: `docs/incident-response.md`)
- [ ] **Key rotation procedure** documented and tested
- [ ] **Disaster recovery** plan tested (RTO < 4h, RPO < 24h)
- [ ] **Changelog** updated for release

---

## Compliance (If Applicable)

- [ ] **GDPR data export** endpoint tested (`GET /api/v1/users/{id}/export`)
- [ ] **GDPR right to erasure** tested (`DELETE /api/v1/users/{id}?hard=true`)
- [ ] **Audit log retention** meets regulatory requirement (SOX: 7 years, HIPAA: 6 years)
- [ ] **Data residency** enforced if required (EU, China, etc.)
- [ ] **PII encryption at rest** verified

---

## Quick Verification Script

```bash
#!/bin/bash
# Run this against your production deployment
URL="https://iam.example.com"

echo "=== Health ==="
curl -sf "$URL/healthz/deep" | jq . || echo "FAIL: healthz"

echo "=== TLS ==="
echo | openssl s_client -connect iam.example.com:443 -servername iam.example.com 2>/dev/null \
  | openssl x509 -noout -dates 2>/dev/null || echo "FAIL: TLS"

echo "=== Security Headers ==="
curl -sI "$URL" | grep -i "strict-transport-security" || echo "WARN: HSTS missing"
curl -sI "$URL" | grep -i "x-content-type-options" || echo "WARN: X-Content-Type-Options missing"

echo "=== Rate Limiting ==="
for i in $(seq 1 15); do curl -s -o /dev/null -w "%{http_code} " "$URL/api/v1/users"; done
echo ""
echo "(should see 401s then 429)"
```

## External Infrastructure

If using external databases/middleware (not bundled containers):

- [ ] `DB_HOST` set to external PostgreSQL host
- [ ] `DB_PORT` set (default 5432)
- [ ] `DB_PASSWORD` set to production password
- [ ] `REDIS_HOST` set to external Redis host
- [ ] `NATS_URL` set to external NATS (e.g. `nats://prod-nats:4222`)
- [ ] `LDAP_URL` set if using LDAP auth provider (e.g. `ldap://prod-ldap:389`)
- [ ] Database has SSL enabled (`DB_SSL_MODE=require`)
- [ ] Redis has AUTH password set

---

*Last updated: 2025-07-11*