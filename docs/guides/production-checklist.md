# GGID Production Readiness Checklist

This checklist ensures your GGID deployment is ready for production traffic. Complete every item before launch.

## TLS & Network Security

- [ ] TLS 1.2+ enforced for all external endpoints
- [ ] TLS certificate valid and auto-renewed (Let's Encrypt / cert-manager)
- [ ] HSTS header: `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- [ ] gRPC TLS between gateway and services (policy, org — extend to all)
- [ ] No plaintext protocols exposed externally (disable HTTP port)
- [ ] Firewall rules: only gateway port (8080/443) exposed externally
- [ ] Database port (5432) NOT exposed externally
- [ ] Redis port (6379) NOT exposed externally
- [ ] NATS port (4222) NOT exposed externally

**Verify**:
```bash
# Check TLS
openssl s_client -connect api.ggid.example.com:443 -tls1_2
# Check HSTS
curl -sI https://api.ggid.example.com/ | grep Strict-Transport
# Check no internal ports exposed
nmap -p 5432,6379,4222,389 api.ggid.example.com  # Should show closed
```

## Database

- [ ] PostgreSQL 16+ with tuned `postgresql.conf` (shared_buffers, work_mem)
- [ ] Database backups running daily + WAL archiving
- [ ] Backup restore tested (not just backup — actual restore!)
- [ ] Connection pool sized per service (total < max_connections)
- [ ] RLS enabled with FORCE ROW LEVEL SECURITY on all tenant tables
- [ ] Indexes on `tenant_id` for all tenant-scoped tables
- [ ] Autovacuum configured and running
- [ ] `pg_stat_statements` enabled for query analysis
- [ ] Non-superuser database role for app (BYPASSRLS only for migrate role)
- [ ] Read replica configured for read-heavy workloads

**Verify**:
```bash
# Test backup restore (on staging)
pg_restore --dbname=ggid_test --list backup.dump | head -10

# Check RLS
psql -c "SELECT relname, relrowsecurity, relforcerowsecurity FROM pg_class WHERE relrowsecurity = true"

# Check autovacuum
psql -c "SELECT relname, last_autovacuum FROM pg_stat_user_tables WHERE last_autovacuum IS NULL"
```

## Redis

- [ ] Redis 7+ with persistence enabled (AOF or RDB)
- [ ] Maxmemory policy: `allkeys-lru` or `volatile-lru`
- [ ] Password authentication enabled
- [ ] TLS configured (if Redis is on shared network)
- [ ] Sentinel or Cluster for HA (if applicable)
- [ ] Connection pool sized (PoolSize: 20+)

## NATS JetStream

- [ ] NATS 2.10+ with file-based storage
- [ ] JetStream enabled with `--store_dir`
- [ ] Monitoring port enabled (`-m 8222`)
- [ ] Stream retention limits configured (max_age, max_msgs, max_bytes)
- [ ] Consumer durability configured (durable name)
- [ ] NATS clustering for HA (if multi-node)
- [ ] NATS backup strategy (snapshot stream files)

**Verify**:
```bash
# Check JetStream
nats stream info AUDIT_EVENTS
# Check stream health
curl http://nats:8222/jsz
```

## Secrets Management

- [ ] JWT signing key NOT in source code or Docker image
- [ ] JWT secret loaded from secrets manager (Vault, AWS SM, K8s Secret)
- [ ] Database password loaded from secrets manager
- [ ] Redis password set
- [ ] API keys generated with crypto/rand (not hardcoded)
- [ ] OAuth client secrets loaded from secrets manager
- [ ] No secrets in environment variable files committed to git
- [ ] `.gitignore` includes `*.env`, `keys/`, `*.key`

**Verify**:
```bash
# Check for committed secrets
git log --all -p | grep -iE "password|secret|api_key|token" | head -5
# Check secrets not in Docker image
docker history ggid-auth:latest | grep -i secret
```

## Authentication & Authorization

- [ ] JWT signed with RS256 or ES256 (not HS256)
- [ ] Access token expiry: 5-15 minutes
- [ ] Refresh token rotation (single-use)
- [ ] `jti` anti-replay tracking in Redis
- [ ] OAuth `state` parameter validated (Redis-backed)
- [ ] `iss` parameter in OAuth responses
- [ ] Account lockout after N failed attempts
- [ ] Password policy: min 12 chars, complexity, history
- [ ] MFA available and recommended for admin accounts
- [ ] Session timeout (absolute: 8h, idle: 30m)

## Gateway Security

- [ ] Rate limiting wired into production handler chain
- [ ] CORS: specific origins (not `*`)
- [ ] Host header validation (DNS rebinding defense)
- [ ] Body size limits configured
- [ ] Circuit breaker per upstream
- [ ] SSRF protection on webhook URLs
- [ ] Security headers: X-Content-Type-Options, X-Frame-Options, CSP, Referrer-Policy
- [ ] Compression enabled (gzip/brotli)
- [ ] TLS termination at gateway (services behind firewall)

**Verify**:
```bash
# Security headers
curl -sI https://api.ggid.example.com/ | grep -cE \
  'Strict-Transport|X-Frame|X-Content-Type|Content-Security|Referrer'

# Rate limiting
for i in $(seq 1 200); do
  curl -s -o /dev/null -w '%{http_code}\n' https://api.ggid.example.com/api/v1/users \
    -H "Authorization: Bearer $TOKEN"
done | sort | uniq -c
```

## Monitoring & Alerting

- [ ] Health check endpoints monitored (`/healthz` per service)
- [ ] Prometheus metrics exposed (`/metrics`)
- [ ] Grafana dashboard for key metrics (latency, error rate, throughput)
- [ ] Alerts for: high error rate, service down, rate limit spike, circuit open
- [ ] Audit SIEM forwarder configured (Splunk/Datadog/ES)
- [ ] Log aggregation (ELK / Datadog / CloudWatch)
- [ ] Audit hash chain verification alert
- [ ] Certificate expiry alert (30 days before)
- [ ] Database connection pool exhaustion alert
- [ ] NATS consumer lag alert

## Compliance

- [ ] Audit log retention policy configured (per regulation)
- [ ] GDPR erasure workflow tested
- [ ] PII obfuscation verified in audit events
- [ ] Data classification documented
- [ ] Access review process defined (quarterly)
- [ ] Incident response plan documented
- [ ] Penetration test scheduled
- [ ] Privacy policy published

## Backup & Disaster Recovery

- [ ] PostgreSQL: daily `pg_basebackup` + continuous WAL archiving
- [ ] Redis: RDB snapshot (hourly, 24h retention)
- [ ] NATS: stream file backup (daily)
- [ ] JWT keys: backed up to secrets manager
- [ ] Configuration: git repository with off-site backup
- [ ] DR drill conducted (quarterly)
- [ ] RTO documented and measured
- [ ] RPO documented and measured
- [ ] Failover procedure documented and tested

**Verify**:
```bash
# Test backup
pg_basebackup -h localhost -U backup -D /tmp/test-restore -X stream -P
ls /tmp/test-restore  # Should have PostgreSQL data files

# Verify WAL archiving
aws s3 ls s3://ggid-wal-archive/ | tail -5  # Recent WAL files present
```

## Performance

- [ ] Load tested at 2x expected peak traffic
- [ ] Login latency: < 100ms p99
- [ ] JWT verify: < 5ms p99 (with JWKS cache)
- [ ] API throughput: 5000+ req/s per gateway instance
- [ ] Audit publish: < 1ms p99
- [ ] PostgreSQL: no slow queries (> 250ms)
- [ ] Connection pools sized correctly
- [ ] Auto-scaling configured (HPA or equivalent)

## Docker / Kubernetes

- [ ] Resource limits set for every container (CPU + memory)
- [ ] Liveness probe configured (`/healthz`)
- [ ] Readiness probe configured
- [ ] Pod anti-affinity (services on different nodes for HA)
- [ ] Horizontal Pod Autoscaler configured
- [ ] Rolling update strategy (maxSurge: 1, maxUnavailable: 0)
- [ ] ConfigMaps for non-secret config
- [ ] Secrets for all sensitive data
- [ ] NetworkPolicy restricting pod-to-pod communication

## Go Live Checklist (Final)

- [ ] All above items completed
- [ ] DNS configured and propagated (lower TTL before cutover)
- [ ] SSL certificate valid and trusted
- [ ] Smoke test passed: register → login → CRUD → audit query
- [ ] E2E test suite: 11/11 PASS (`bash deploy/e2e-docker-test.sh`)
- [ ] Team on-call schedule defined
- [ ] Rollback plan documented
- [ ] Status page configured
- [ ] Support channels published

## Post-Launch Monitoring (First 48 Hours)

- [ ] Check error rate every 2 hours
- [ ] Monitor p99 latency
- [ ] Watch for rate limit triggers
- [ ] Verify audit events flowing to SIEM
- [ ] Check NATS consumer lag
- [ ] Monitor database connection pool
- [ ] Review failed login patterns
- [ ] Verify backups completing successfully

## See Also

- Production Readiness Checklist (research)
- [Security Audit Checklist](security-audit-checklist.md)
- [Docker Deployment](docker-deployment.md)
- [Kubernetes Deployment](kubernetes-deployment.md)
- [Disaster Recovery](../research/disaster-recovery.md)
- [Performance Tuning](performance-tuning.md)
