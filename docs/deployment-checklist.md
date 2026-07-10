# GGID Production Deployment Checklist

Use this checklist before deploying GGID to production.

---

## Pre-Deployment

### Infrastructure

- [ ] PostgreSQL 16+ provisioned with SSD storage
- [ ] Redis 7+ provisioned with persistence enabled
- [ ] NATS 2.10+ provisioned with file storage
- [ ] At least 2 CPU cores and 8GB RAM for the GGID stack
- [ ] DNS records configured (e.g., `iam.example.com`)

### Network

- [ ] Firewall rules: only expose Gateway (8080) and Console (3000) externally
- [ ] Internal network: all services on private network (Docker/K8s)
- [ ] PostgreSQL, Redis, NATS NOT exposed to public internet
- [ ] TLS termination configured (nginx/Caddy/Ingress)

---

## Security

### Authentication & Authorization

- [ ] Default admin password changed from `Admin@123456`
- [ ] JWT signing key is 2048-bit RSA minimum
- [ ] JWT access token TTL configured (recommend 15min - 1h)
- [ ] Refresh token TTL configured (recommend 7-30 days)
- [ ] MFA enforced for admin accounts
- [ ] Rate limits configured per tenant (not just per-IP)

### Database

- [ ] Non-default PostgreSQL password (not `ggid`)
- [ ] Application role created with `NOBYPASSRLS` (not superuser)
- [ ] RLS enabled and forced on all multi-tenant tables
- [ ] Database connections use TLS (`sslmode=require` or `verify-full`)
- [ ] Connection pool size configured (recommend 20-50 per service)

### Secrets

- [ ] No secrets in environment files (use Docker Secrets / K8s Secrets / Vault)
- [ ] JWT private key stored in secret manager
- [ ] LDAP bind password in secret
- [ ] SMTP password in secret
- [ ] OAuth client secrets in secret
- [ ] Secret rotation policy defined (90 days for JWT keys)

### Network Security

- [ ] TLS 1.3 enforced (TLS 1.2 minimum with PFS)
- [ ] HSTS header: `max-age=63072000; includeSubDomains; preload`
- [ ] CORS origins configured (not `*`)
- [ ] IP allowlist configured for admin endpoints
- [ ] Internal service communication on private network (not localhost)

---

## Data & Backup

### Backup Strategy

- [ ] Daily `pg_dump` backup automated
- [ ] WAL archiving enabled for Point-In-Time Recovery (PITR)
- [ ] Backups encrypted (AES-256)
- [ ] Backups stored in separate availability zone/region
- [ ] Backup retention defined (minimum 30 days)
- [ ] Backup restoration tested (not just assumed working)
- [ ] JWT signing key backed up separately

### Data Encryption

- [ ] Disk encryption enabled (LUKS / EBS encryption)
- [ ] Database connections over TLS
- [ ] Redis password set (not default)
- [ ] NATS authentication configured

---

## Monitoring & Observability

### Metrics

- [ ] Prometheus scraping Gateway `/metrics`
- [ ] Grafana dashboard imported (deploy/grafana/)
- [ ] Alert rules configured (deploy/prometheus/alerts/)
- [ ] Key metrics monitored:
  - Request rate and error rate
  - p95 latency per endpoint
  - Backend health scores
  - Rate limit hits
  - Circuit breaker state

### Logging

- [ ] Log aggregation configured (ELK / Loki / Datadog)
- [ ] Structured JSON logging verified
- [ ] Request IDs propagated through all services
- [ ] Log retention set (minimum 90 days for security events)

### Alerting

- [ ] Alert: Gateway down (no healthz response)
- [ ] Alert: High error rate (> 5% for 5 minutes)
- [ ] Alert: High latency (p95 > 500ms for 5 minutes)
- [ ] Alert: Backend unhealthy (health score < 50)
- [ ] Alert: Rate limit hits spiking (possible attack)
- [ ] Alert: Disk usage > 80%
- [ ] Alerts routed to PagerDuty / Slack / email

### Tracing

- [ ] OpenTelemetry collector deployed
- [ ] OTLP endpoint configured in Gateway (`OTEL_EXPORTER_OTLP_ENDPOINT`)
- [ ] Trace sampling configured (recommend 10% in production)

---

## Performance

### Database

- [ ] Indexes verified (`tenant_id` first on all multi-tenant tables)
- [ ] `pg_stat_statements` enabled for query analysis
- [ ] `autovacuum` configured and running
- [ ] Connection pool size tuned for workload
- [ ] Slow query log threshold set (> 100ms)

### Gateway

- [ ] Multiple Gateway replicas (minimum 2)
- [ ] Load balancer health check path set to `/healthz`
- [ ] Graceful shutdown timeout configured (30s)
- [ ] Request body size limit set
- [ ] Compression enabled (gzip)

### Redis

- [ ] `maxmemory` policy configured (recommend `allkeys-lru`)
- [ ] Persistence mode set (AOF or RDB)
- [ ] Password authentication enabled

### NATS

- [ ] File storage configured (not memory)
- [ ] Max age set for audit stream (recommend 7 days)
- [ ] Monitoring endpoint enabled (`:8222`)

---

## High Availability

- [ ] Minimum 2 Gateway replicas
- [ ] Minimum 2 Auth service replicas
- [ ] PostgreSQL: primary + replica (or managed DB)
- [ ] Redis: replica or cluster mode
- [ ] NATS: 3-node cluster (RAFT)
- [ ] PodDisruptionBudget configured (K8s)
- [ ] Health check readiness probes configured
- [ ] Auto-scaling configured (HPA)

---

## Compliance

### GDPR

- [ ] Data export endpoint tested (`GET /api/v1/users/{id}?format=json`)
- [ ] Right to erasure tested (DELETE cascades to all tables)
- [ ] Audit log retention policy documented
- [ ] Data residency confirmed (data stays in specified region)
- [ ] Privacy policy references GGID data handling

### SOC 2

- [ ] Audit logging verified for all security events
- [ ] Access controls (RBAC) verified
- [ ] MFA enforced for privileged accounts
- [ ] Session timeout configured
- [ ] Change management process documented

### HIPAA (if applicable)

- [ ] All data encrypted at rest
- [ ] All data encrypted in transit
- [ ] Audit trail covers PHI access
- [ ] BAA signed with infrastructure provider

---

## Docker / Container Security

- [ ] Container images scanned (Trivy / Grype)
- [ ] govulncheck run on all Go packages
- [ ] Containers run as non-root user
- [ ] Container filesystem read-only (except volumes)
- [ ] No secrets in Docker images
- [ ] Image tags pinned to specific version (not `latest`)

---

## Go Live

### Smoke Test

```bash
# 1. Health check
curl https://iam.example.com/healthz

# 2. Register test user
curl -X POST https://iam.example.com/api/v1/auth/register \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"smoketest","email":"smoke@test.com","password":"Test@12345"}'

# 3. Login
TOKEN=$(curl -s -X POST https://iam.example.com/api/v1/auth/login \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"smoketest","password":"Test@12345"}' | jq -r '.access_token')

# 4. API call with token
curl https://iam.example.com/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# 5. Clean up
curl -X DELETE https://iam.example.com/api/v1/users/smoketest \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Rollback Plan

- [ ] Previous version images available
- [ ] Database migration rollback scripts tested
- [ ] DNS rollback procedure documented
- [ ] Rollback decision maker identified
- [ ] Communication plan for stakeholders

---

## Post-Deployment

- [ ] Monitor error rates for 1 hour after deployment
- [ ] Verify audit events are flowing
- [ ] Check Prometheus targets are all up
- [ ] Verify Grafana dashboards show data
- [ ] Test alert routing (trigger a test alert)
- [ ] Document deployment in change log
