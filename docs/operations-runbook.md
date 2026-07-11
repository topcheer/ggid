# Operations Runbook

> Step-by-step guide for operators: deploy, configure, manage tenants, rotate keys, perform backups, monitor health, and troubleshoot GGID in production.

---

## Table of Contents

1. [Deployment](#deployment)
2. [Service Configuration](#service-configuration)
3. [Tenant Management](#tenant-management)
4. [Key Rotation](#key-rotation)
5. [Backup and Restore](#backup-and-restore)
6. [Health Monitoring](#health-monitoring)
7. [Log Management](#log-management)
8. [Scaling Operations](#scaling-operations)
9. [Database Maintenance](#database-maintenance)
10. [Security Operations](#security-operations)
11. [Common Issues and Solutions](#common-issues-and-solutions)
12. [Emergency Procedures](#emergency-procedures)

---

## Deployment

### Docker Compose (Recommended for Dev/Staging)

```bash
# Clone and enter deploy directory
cd deploy

# Start the full stack (13 containers)
docker compose up -d

# Verify all containers are healthy
docker compose ps

# Expected: all services "healthy"
# Gateway:     8080
# Identity:    8081
# Auth:        9001
# OAuth:       9005
# Policy:      8070, 9070
# Org:         8071, 9071
# Audit:       8072, 9072
# Console:     3000
# PostgreSQL:  5432
# Redis:       6379
# NATS:        4222, 8222
# OpenLDAP:    389
```

### Verify Deployment

```bash
# Gateway health
curl http://localhost:8080/healthz
# Expected: {"status":"ok"}

# Deep health (checks all backends)
curl http://localhost:8080/healthz/deep
# Expected: {"status":"ok","services":{"identity":"ok","auth":"ok",...}}

# Run E2E tests
bash deploy/e2e-docker-test.sh
# Expected: 11/11 PASS
```

### Kubernetes (Production)

```bash
# Apply Helm chart
helm install ggid deploy/helm/ggid \
  --set gateway.replicas=3 \
  --set auth.replicas=2 \
  --set database.enabled=true

# Verify pods
kubectl get pods -n ggid

# Check service endpoints
kubectl get svc -n ggid
```

### Rolling Update

```bash
# Docker Compose — zero downtime (run new alongside old)
docker compose up -d --no-deps --build gateway

# Kubernetes — rolling update
kubectl set image deployment/ggid-gateway gateway=ggid-gateway:v1.1.0 -n ggid
kubectl rollout status deployment/ggid-gateway -n ggid
```

---

## Service Configuration

### Environment Variables Reference

#### Gateway

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_LISTEN` | `:8080` | Listen address |
| `JWT_SECRET` | **required** | HMAC signing key |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection |
| `NATS_URL` | `nats://localhost:4222` | NATS connection |
| `RATE_LIMIT_RPM` | `1000` | Rate limit per minute |
| `CIRCUIT_BREAKER_THRESHOLD` | `5` | Failures before open |
| `LOG_LEVEL` | `info` | debug/info/warn/error |

#### Auth Service

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_LISTEN` | `:9001` | Listen address |
| `JWT_SECRET` | **required** | Must match gateway |
| `DATABASE_URL` | **required** | PostgreSQL connection |
| `REDIS_URL` | **required** | Session storage |
| `LDAP_URL` | (empty) | If set, enables LDAP provider |
| `LDAP_BIND_DN` | (empty) | LDAP service account |
| `LDAP_BIND_PASSWORD` | (empty) | LDAP service password |
| `LDAP_BASE_DN` | (empty) | LDAP search base |
| `LDAP_USER_FILTER` | `(uid=%s)` | LDAP user search filter |
| `LDAP_START_TLS` | `false` | Enable START_TLS |
| `LDAP_AUTO_PROVISION` | `true` | Auto-create LDAP users locally |

#### Policy / Org / Audit

These services use individual DB variables (NOT `DATABASE_URL`):

| Variable | Default |
|----------|---------|
| `DB_HOST` | `localhost` |
| `DB_PORT` | `5432` |
| `DB_USER` | `ggid` |
| `DB_PASSWORD` | (required) |
| `DB_NAME` | `ggid` |
| `DB_SSLMODE` | `disable` |

#### OAuth Service

| Variable | Default | Description |
|----------|---------|-------------|
| `OAUTH_LISTEN` | `:9005` | Listen address |
| `JWT_SECRET` | **required** | Must match gateway |
| `DATABASE_URL` | **required** | PostgreSQL |

---

## Tenant Management

### Create a New Tenant

```bash
# Create tenant (requires super-admin JWT)
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <super-admin-JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "plan": "enterprise",
    "max_users": 10000
  }'

# Response includes tenant UUID
# {"id":"55000000-0000-0000-0000-000000000002","name":"Acme Corporation",...}
```

### Create First Admin User for Tenant

```bash
# Register admin user with new tenant ID
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "X-Tenant-ID: 55000000-0000-0000-0000-000000000002" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@acme.com",
    "password": "SecurePass123!"
  }'

# Assign super_admin role
curl -X POST http://localhost:8080/api/v1/users/{user_id}/roles \
  -H "Authorization: Bearer <super-admin-JWT>" \
  -H "X-Tenant-ID: 55000000-0000-0000-0000-000000000002" \
  -H "Content-Type: application/json" \
  -d '{"role_id": "role_super_admin"}'
```

### Suspend a Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants/{tenant_id}/suspend \
  -H "Authorization: Bearer <super-admin-JWT>" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Non-payment"}'
```

### Delete a Tenant (Irreversible)

```bash
# Export tenant data first!
curl http://localhost:8080/api/v1/tenants/{tenant_id}/export \
  -H "Authorization: Bearer <super-admin-JWT>" \
  -o tenant_backup.json

# Then delete
curl -X DELETE http://localhost:8080/api/v1/tenants/{tenant_id} \
  -H "Authorization: Bearer <super-admin-JWT>"
```

### Default Tenant

- UUID: `00000000-0000-0000-0000-000000000001`
- Pre-seeded during migration
- Used for single-tenant deployments

---

## Key Rotation

### JWT Signing Secret Rotation

**Current process** (manual — zero-downtime rotation is planned):

```bash
# 1. Generate new secret
NEW_SECRET=$(openssl rand -base64 32)
echo "New JWT secret: $NEW_SECRET"

# 2. Update environment variable on ALL services
#    Gateway, Auth, OAuth, Identity, Policy, Org, Audit
#    ALL must use the same secret

# 3. Restart services one at a time (rolling)
docker compose restart gateway
sleep 5
docker compose restart auth
sleep 5
docker compose restart oauth
# ... continue for all services

# 4. All existing tokens become invalid
#    Users will need to re-authenticate

# 5. Verify
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}'
# Should get new token signed with new secret
```

### Database Password Rotation

```bash
# 1. Generate new password
NEW_DB_PASS=$(openssl rand -base64 24)

# 2. Update PostgreSQL user
psql -c "ALTER USER ggid WITH PASSWORD '$NEW_DB_PASS';"

# 3. Update environment variables on all services
#    DB_PASSWORD, DATABASE_URL

# 4. Restart services
docker compose restart
```

### Redis Password Rotation

```bash
# 1. Generate new password
NEW_REDIS_PASS=$(openssl rand -base64 24)

# 2. Update Redis config
redis-cli CONFIG SET requirepass "$NEW_REDIS_PASS"

# 3. Update REDIS_URL on all services
#    redis://:$NEW_REDIS_PASS@localhost:6379

# 4. Restart services
docker compose restart
```

### TLS Certificate Renewal

```bash
# Let's Encrypt (automated via cert-manager in K8s)
certbot renew --deploy-hook "docker compose restart gateway"

# Manual
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
# Update cert paths in load balancer / gateway config
```

---

## Backup and Restore

### Automated Backups

The backup script (`deploy/scripts/backup.sh`) runs as a cron job:

```bash
# Install cron job (daily at 2 AM)
echo "0 2 * * * /opt/ggid/scripts/backup.sh /var/backups/ggid" | crontab -
```

**Features**:
- Compressed `pg_dump` with timestamp
- SHA-256 checksum verification
- Retention: 7 daily, 4 weekly, 12 monthly snapshots
- Optional S3 upload with SSE encryption
- Webhook notification on success/failure

### Manual Backup

```bash
# Run backup script directly
DB_HOST=localhost DB_PORT=5432 DB_NAME=ggid DB_USER=ggid DB_PASSWORD=ggid \
  bash deploy/scripts/backup.sh /var/backups/ggid

# Verify backup
sha256sum -c /var/backups/ggid/ggid_20250711_020000.sha256
```

### Restore from Backup

```bash
# 1. Stop services that write to DB
docker compose stop gateway auth identity oauth policy org audit

# 2. Keep PostgreSQL running
docker compose start postgres

# 3. Restore
PGPASSWORD=ggid pg_restore \
  -h localhost -p 5432 -U ggid -d ggid \
  --clean --if-exists \
  /var/backups/ggid/ggid_20250711_020000.sql.gz

# 4. Run migrations to ensure schema is current
go run ./cmd/migrate up

# 5. Restart services
docker compose up -d
```

### Backup Verification (Monthly Drill)

```bash
# 1. Restore backup to test database
createdb ggid_test
PGPASSWORD=ggid pg_restore -d ggid_test /var/backups/ggid/ggid_latest.sql.gz

# 2. Verify row counts
psql -d ggid_test -c "SELECT COUNT(*) FROM users;"
psql -d ggid_test -c "SELECT COUNT(*) FROM audit_events;"

# 3. Verify data integrity
psql -d ggid_test -c "SELECT tenant_id, COUNT(*) FROM users GROUP BY tenant_id;"

# 4. Clean up
dropdb ggid_test
```

---

## Health Monitoring

### Health Endpoints

| Endpoint | Purpose | Checks |
|----------|---------|--------|
| `GET /healthz` | Liveness probe | Process is running |
| `GET /healthz/deep` | Readiness probe | All backends reachable |
| `GET /metrics` | Prometheus metrics | CPU, memory, request rates |
| `NATS: GET http://:8222/healthz` | NATS health | JetStream status |
| `Redis: redis-cli PING` | Redis health | `PONG` response |

### Deep Health Check

```bash
curl http://localhost:8080/healthz/deep | jq .
```

Response:
```json
{
  "status": "ok",
  "services": {
    "identity": {"status": "ok", "latency_ms": 2},
    "auth": {"status": "ok", "latency_ms": 1},
    "oauth": {"status": "ok", "latency_ms": 3},
    "policy": {"status": "ok", "latency_ms": 1},
    "org": {"status": "ok", "latency_ms": 2},
    "audit": {"status": "ok", "latency_ms": 1}
  },
  "infrastructure": {
    "postgresql": "ok",
    "redis": "ok",
    "nats": "ok"
  }
}
```

### Prometheus Metrics

All services expose `/metrics` for Prometheus scraping:

```
# Request rate
ggid_http_requests_total{service="gateway",method="GET",path="/api/v1/users",status="200"}

# Latency histogram
ggid_http_request_duration_seconds{service="gateway",path="/api/v1/users"}

# Circuit breaker state
ggid_circuit_breaker_state{backend="auth-service",state="closed"} 1

# Rate limiter
ggid_rate_limit_hits_total{tenant_id="00000000-..."}
```

### Alerting Rules

```yaml
# Prometheus alert rules
groups:
- name: ggid
  rules:
  - alert: ServiceDown
    expr: up{job="ggid"} == 0
    for: 1m
    annotations:
      summary: "GGID service {{ $labels.instance }} is down"

  - alert: HighErrorRate
    expr: rate(ggid_http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High 5xx error rate on {{ $labels.service }}"

  - alert: CircuitBreakerOpen
    expr: ggid_circuit_breaker_state{state="open"} == 1
    for: 1m
    annotations:
      summary: "Circuit breaker open for {{ $labels.backend }}"

  - alert: DatabaseConnectionsHigh
    expr: pg_stat_activity_count > 80
    for: 5m
    annotations:
      summary: "PostgreSQL connections > 80"
```

---

## Log Management

### Log Levels

| Level | When to Use | Example |
|-------|-------------|---------|
| `debug` | Development only | Request body, SQL queries |
| `info` | Production default | Request summary, login success |
| `warn` | Potential issues | Rate limit hit, circuit breaker opening |
| `error` | Failures | Database error, auth failure |

### Structured Logging

GGID uses `slog` structured logging:

```json
{
  "time": "2025-07-11T12:00:00.123Z",
  "level": "INFO",
  "msg": "request completed",
  "method": "GET",
  "path": "/api/v1/users",
  "status": 200,
  "duration_ms": 15,
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "user_id": "usr_abc123",
  "client_ip": "192.168.1.50"
}
```

### Viewing Logs

```bash
# Docker Compose
docker compose logs -f gateway
docker compose logs -f auth --since 5m

# Kubernetes
kubectl logs -f deployment/ggid-gateway -n ggid
kubectl logs -f deployment/ggid-auth -n ggid --tail=100

# Filter for errors
docker compose logs gateway 2>&1 | grep '"level":"ERROR"'
```

### Log Retention

- Docker: Configured via logging driver (default: json-file with 10MB rotation)
- Kubernetes: Configured via Fluentd/Loki (recommended 30-day retention)
- Audit events in PostgreSQL: 90-day default (configurable via `AUDIT_RETENTION_DAYS`)

---

## Scaling Operations

### Horizontal Scaling

```bash
# Docker Compose — scale gateway
docker compose up -d --scale gateway=3

# Kubernetes — scale deployments
kubectl scale deployment ggid-gateway --replicas=3 -n ggid
kubectl scale deployment ggid-auth --replicas=2 -n ggid
```

### When to Scale Which Service

| Symptom | Scale This Service | Why |
|---------|-------------------|-----|
| High gateway latency | Gateway | Add more proxy capacity |
| Login timeouts | Auth | Add more auth processing |
| Slow user list | Identity | Add more identity workers |
| Audit lag | Audit | Add more NATS consumers |
| High DB CPU | PostgreSQL (vertical) | Add CPU/RAM, read replicas |

### Database Connection Pool Sizing

```
Rule of thumb:
  pool_size = (service_instances × pool_per_instance) < max_connections - 10

Example:
  PostgreSQL max_connections = 100
  3 gateway instances × 10 pool = 30
  2 auth instances × 5 pool = 10
  2 identity instances × 5 pool = 10
  2 policy instances × 5 pool = 10
  Total: 60 connections (safe margin)
```

---

## Database Maintenance

### Vacuum and Analyze

```bash
# Weekly vacuum (automated by autovacuum, but manual for large tables)
psql -c "VACUUM ANALYZE users;"
psql -c "VACUUM ANALYZE audit_events;"
psql -c "VACUUM ANALYZE refresh_tokens;"
```

### Index Maintenance

```sql
-- Check for bloated indexes
SELECT schemaname, tablename, indexname, pg_size_pretty(pg_relation_size(indexname::text))
FROM pg_tables
JOIN pg_stat_user_indexes ON tablename = relname
WHERE pg_relation_size(indexname::text) > 100000000
ORDER BY pg_relation_size(indexname::text) DESC;

-- Rebuild bloated indexes (online, no locks)
REINDEX INDEX CONCURRENTLY users_pkey;
REINDEX INDEX CONCURRENTLY audit_events_timestamp_idx;
```

### Migration Management

```bash
# Check migration status
go run ./cmd/migrate status

# Apply new migrations
go run ./cmd/migrate up

# Rollback last migration (careful!)
go run ./cmd/migrate down

# Create new migration
go run ./cmd/migrate create add_new_feature
```

---

## Security Operations

### Daily Security Checks

```bash
# 1. Check for failed login spikes (brute force)
curl ".../api/v1/audit/events?event_type=auth.login_failed&from=$(date -d '1 day ago' +%Y-%m-%d)" \
  -H "Authorization: Bearer <admin-JWT>" | jq '.events | length'

# 2. Check for 403 responses (unauthorized access attempts)
curl ".../api/v1/audit/events?status_code=403&from=$(date -d '1 day ago' +%Y-%m-%d)" \
  -H "Authorization: Bearer <admin-JWT>" | jq '.events | length'

# 3. Check active session count
redis-cli keys "session:*" | wc -l

# 4. Verify backups succeeded
ls -la /var/backups/ggid/ | tail -5

# 5. Check certificate expiry
echo | openssl s_client -connect localhost:443 2>/dev/null | \
  openssl x509 -noout -dates
```

### Rate Limit Clearance

If a legitimate user is rate-limited:

```bash
# Option 1: Restart auth service (clears all rate limits)
docker compose restart auth

# Option 2: Delete specific Redis key (surgical)
redis-cli del "rate_limit:192.168.1.50"
```

---

## Common Issues and Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| Gateway returns 502 | Backend service down | `docker compose ps`, restart failing service |
| Login returns 429 | Rate limit hit (>5 failures) | Restart auth: `docker compose restart auth` |
| Register returns 409 | Missing `username` field | Include `username` in request body |
| Create role returns 500 | Missing `key` field | Include unique `key` in request body |
| NATS healthcheck fails | Missing `-m 8222` flag | Add `command: ["-m", "8222"]` in compose |
| Policy won't start | Wrong DB env vars | Use `DB_HOST`/`DB_PORT`, NOT `DATABASE_URL` |
| Audit returns 404 | Route mismatch | Gateway routes `/api/v1/audit` → audit `/api/v1/audit/events` |
| High DB connections | Pool too large | Reduce pool size or scale instances down |
| Slow audit query | Missing index | `CREATE INDEX CONCURRENTLY` on `audit_events(tenant_id, timestamp)` |
| Token replay error | JTI already used | Expected behavior — user needs to re-authenticate |
| Migration fails | Dirty migration state | `UPDATE schema_migrations SET dirty=false;` then retry |
| `too many errors` compile | Stale Go cache | `go clean -testcache && go build ./...` |
| Migrations skip | DB already initialized | Idempotent init container handles this |

---

## Emergency Procedures

### Service Outage (All Users Affected)

```
1. Check health endpoints
   curl http://localhost:8080/healthz/deep

2. Identify which service is down
   docker compose ps

3. Check logs for the failing service
   docker compose logs --tail=100 <failing-service>

4. If database issue:
   a. Check PostgreSQL is running
   b. Check connection pool
   c. Restore from backup if data corruption

5. If Redis issue:
   a. Check Redis is running
   b. Sessions will be lost — users re-authenticate

6. If NATS issue:
   a. Check NATS is running
   b. Audit events queue up locally, deliver when NATS recovers

7. Communicate status
   a. Update status page
   b. Notify customers if SEV-1
```

### Rollback a Deployment

```bash
# Docker Compose — revert to previous image
docker compose down
git checkout <previous-release-tag>
docker compose up -d

# Kubernetes — rollback
kubectl rollout undo deployment/ggid-gateway -n ggid
kubectl rollout undo deployment/ggid-auth -n ggid
```

### Database Rollback

```bash
# 1. Stop all services
docker compose stop gateway auth identity oauth policy org audit

# 2. Restore from pre-deployment backup
PGPASSWORD=ggid pg_restore \
  -h localhost -p 5432 -U ggid -d ggid \
  --clean --if-exists \
  /var/backups/ggid/ggid_pre_deploy.sql.gz

# 3. Restart services
docker compose up -d
```

---

*Last updated: 2025-07-11*
