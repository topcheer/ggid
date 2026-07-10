# GGID Security Hardening Guide

Production security checklist and hardening guide for GGID deployments.

This guide covers every layer of the stack — from network and transport security
to database isolation and application-level controls.

---

## Table of Contents

- [Pre-Deployment Checklist](#pre-deployment-checklist)
- [TLS Configuration](#tls-configuration)
- [Key Management & Rotation](#key-management--rotation)
- [PostgreSQL Hardening](#postgresql-hardening)
- [Redis Hardening](#redis-hardening)
- [NATS Hardening](#nats-hardening)
- [JWT Expiry & Token Lifecycle](#jwt-expiry--token-lifecycle)
- [CORS Whitelist](#cors-whitelist)
- [Rate Limiting](#rate-limiting)
- [Audit Log Integrity](#audit-log-integrity)
- [Least Privilege Principle](#least-privilege-principle)
- [Secrets Management](#secrets-management)
- [Network Security](#network-security)
- [Compliance](#compliance)

---

## Pre-Deployment Checklist

Before going to production, verify every item below:

- [ ] TLS terminated at ingress (Let's Encrypt or internal CA)
- [ ] RSA key pair generated fresh (not using development keys)
- [ ] PostgreSQL uses non-superuser role with `NOBYPASSRLS`
- [ ] Redis password set (not default)
- [ ] NATS authentication enabled
- [ ] All default passwords changed
- [ ] CORS origins restricted to your frontend domains
- [ ] Rate limiting configured for auth endpoints
- [ ] Audit logging verified (events flowing through NATS)
- [ ] Secrets stored in Vault / Kubernetes Secrets (not plaintext env)
- [ ] Network policies restrict inter-service communication
- [ ] LDAP auto-provision disabled unless explicitly needed
- [ ] Container images scanned (Trivy / Snyk)
- [ ] `govulncheck` passes with zero vulnerabilities
- [ ] Backup strategy tested (PostgreSQL + RSA keys)
- [ ] Monitoring and alerting configured

---

## TLS Configuration

### Layer 1: Edge TLS (Ingress / Reverse Proxy)

Terminate TLS at the edge using nginx, Caddy, or a cloud load balancer.

**Caddy (automatic Let's Encrypt):**
```Caddyfile
iam.example.com {
    reverse_proxy gateway:8080
    tls you@email.com
}
```

**nginx:**
```nginx
server {
    listen 443 ssl http2;
    server_name iam.example.com;

    ssl_certificate     /etc/ssl/certs/iam.crt;
    ssl_certificate_key /etc/ssl/private/iam.key;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers on;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 10m;

    # HSTS
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

    # Security headers
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'" always;

    location / {
        proxy_pass http://gateway:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name iam.example.com;
    return 301 https://$host$request_uri;
}
```

### Layer 2: Internal TLS (Service-to-Service)

For high-security environments, enable TLS between services:

```bash
# Generate internal certificates (or use cert-manager / internal CA)
openssl req -x509 -newkey rsa:2048 -keyout service.key -out service.crt \
  -days 365 -nodes -subj "/CN=ggid-internal"

# Mount certs and set environment
TLS_CERT_PATH=/certs/service.crt
TLS_KEY_PATH=/certs/service.key
```

For Kubernetes, use a service mesh (Linkerd or Istio) for automatic mTLS.

---

## Key Management & Rotation

### RSA Key Pair (JWT Signing)

GGID uses RSA 2048-bit keys for JWT signing (RS256). The private key signs
tokens in the Auth service; the public key verifies them in the Gateway.

**Generate fresh keys:**
```bash
openssl genpkey -algorithm RSA -out rsa_private.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -pubout -in rsa_private.pem -out rsa_public.pem
chmod 600 rsa_private.pem
```

**Rotation procedure:**
1. Generate new key pair
2. Add the new public key to JWKS alongside the old key (dual-key period)
3. Restart Auth service with the new private key
4. Wait for old JWTs to expire (default: 1 hour)
5. Remove the old public key from JWKS
6. Restart Gateway to refresh JWKS cache

**Backup:** Store the private key in your secrets manager immediately.
Losing it invalidates ALL issued tokens.

### Key Storage

| Environment | Recommendation |
|-------------|----------------|
| Docker Compose | Docker secrets (`docker secret create`) |
| Kubernetes | Kubernetes Secrets + sealed-secrets or external-secrets |
| Cloud | AWS KMS, GCP KMS, or HashiCorp Vault |
| Air-gapped | HSM or offline encrypted storage |

---

## PostgreSQL Hardening

### Non-Superuser Application Role

Docker Compose uses a superuser (bypasses RLS). In production, create a
limited role:

```sql
-- Create application role
CREATE ROLE ggid_app WITH LOGIN PASSWORD 'STRONG_RANDOM_PASSWORD' NOBYPASSRLS;

-- Grant minimum privileges
GRANT CONNECT ON DATABASE ggid TO ggid_app;
GRANT USAGE ON SCHEMA public TO ggid_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ggid_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ggid_app;

-- Ensure future tables also grant access
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ggid_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO ggid_app;
```

Then set the connection string to use `ggid_app`:
```
DATABASE_URL=postgres://ggid_app:password@host:5432/ggid?sslmode=require
```

### Verify RLS Is Enforced

```sql
-- Check RLS is enabled on all multi-tenant tables
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE relname IN ('users', 'credentials', 'roles', 'organizations', 'departments',
                   'teams', 'org_members', 'policies', 'audit_events')
ORDER BY relname;

-- All should show: relrowsecurity = t, relforcerowsecurity = t

-- Force RLS even for table owners (defense in depth)
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE credentials FORCE ROW LEVEL SECURITY;
ALTER TABLE roles FORCE ROW LEVEL SECURITY;
ALTER TABLE organizations FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

-- Test: connect as ggid_app and try cross-tenant query
SET app.tenant_id = '00000000-0000-0000-0000-000000000001';
SELECT count(*) FROM users; -- Should only return Tenant 1 users
```

### SSL/TLS for Database Connections

```bash
# Require SSL in connection string
DATABASE_URL=postgres://ggid_app:password@host:5432/ggid?sslmode=verify-full

# In PostgreSQL config (postgresql.conf)
ssl = on
ssl_cert_file = '/etc/postgresql/server.crt'
ssl_key_file = '/etc/postgresql/server.key'

# In pg_hba.conf
hostssl ggid ggid_app 0.0.0.0/0 scram-sha-256
```

### Connection Pooling

Use PgBouncer in production to limit max connections:

```ini
# pgbouncer.ini
[databases]
ggid = host=127.0.0.1 port=5432 dbname=ggid

[pgbouncer]
pool_mode = transaction
max_client_conn = 200
default_pool_size = 20
```

---

## Redis Hardening

### Set a Strong Password

```bash
# In redis.conf
requirepass YOUR_STRONG_RANDOM_PASSWORD

# Or via command line
redis-server --requirepass YOUR_STRONG_RANDOM_PASSWORD
```

```bash
# Update service environment
REDIS_ADDR=redis:6379
REDIS_PASSWORD=YOUR_STRONG_RANDOM_PASSWORD
```

### Disable Dangerous Commands

```conf
# redis.conf
rename-command FLUSHDB ""
rename-command FLUSHALL ""
rename-command DEBUG ""
rename-command CONFIG ""
rename-command SHUTDOWN ""
```

### TLS for Redis

```conf
# redis.conf
tls-port 6380
port 0
tls-cert-file /tls/redis.crt
tls-key-file /tls/redis.key
tls-ca-cert-file /tls/ca.crt
```

---

## NATS Hardening

### Enable Authentication

```bash
# Start NATS with auth
nats-server \
  --auth=YOUR_STRONG_TOKEN \
  -m 8222
```

Or use account-based JWT auth:

```conf
# nats-server.conf
authorization {
    user: ggid
    password: STRONG_PASSWORD
    timeout: 5s
}
```

### TLS for NATS

```conf
tls {
    cert_file: "/tls/nats.crt"
    key_file: "/tls/nats.key"
    ca_file: "/tls/ca.crt"
}
```

---

## JWT Expiry & Token Lifecycle

### Recommended TTLs

| Token Type | Default | Recommended Production |
|------------|---------|----------------------|
| Access token | 1 hour (3600s) | 15 minutes (900s) for high-security |
| Refresh token | 30 days | 7 days for high-security |
| MFA token (temporary) | 5 minutes | 5 minutes (keep as-is) |

### Configure TTLs

```bash
# Auth service environment
AUTH_ACCESS_TOKEN_TTL=900      # 15 minutes
AUTH_REFRESH_TOKEN_TTL=604800   # 7 days
```

### Token Revocation

GGID supports two revocation strategies:

1. **Short-lived access tokens** — Tokens expire quickly; no revocation needed
2. **Redis blocklist** — The Auth service maintains a token blocklist in Redis
   for immediate invalidation (checked at refresh time)

```bash
# Enable Redis blocklist (enabled by default when Redis is configured)
AUTH_TOKEN_BLOCKLIST=true
```

---

## CORS Whitelist

### Configure Allowed Origins

```bash
# Gateway environment
GATEWAY_CORS_ORIGINS=https://app.example.com,https://admin.example.com
GATEWAY_CORS_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
GATEWAY_CORS_HEADERS=Authorization,Content-Type,X-Tenant-ID,X-Request-ID
GATEWAY_CORS_MAX_AGE=300
```

### Security Rules

- **Never use `*`** in production — always list explicit origins
- Only allow methods your application uses
- Include `X-Tenant-ID` in allowed headers
- Set `max_age` to 300 (5 minutes) to reduce preflight requests

---

## Rate Limiting

### Current Configuration

The Gateway uses in-memory fixed-window rate limiting:

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/api/v1/auth/login` | 5 requests | per minute per IP |
| `/api/v1/auth/register` | 3 requests | per minute per IP |
| `/api/v1/*` (general) | 100 requests | per minute per IP |

### Production Recommendations

1. **Use Redis-backed rate limiting** for multi-instance deployments:
   ```go
   // The in-memory limiter doesn't share state across Gateway replicas.
   // For production, implement a Redis-backed limiter using:
   // - INCR + EXPIRE for fixed-window counting
   // - Or use redisrate library
   ```

2. **Add per-tenant rate limiting** (in addition to per-IP):
   ```bash
   # Gateway environment
   GATEWAY_TENANT_RATE_LIMIT=1000  # requests per minute per tenant
   ```

3. **Configure progressive backoff**:
   - 1-5 failed logins: normal response
   - 6-10: exponential delay (1s, 2s, 4s...)
   - 10+: temporary IP ban (15 minutes)

---

## Audit Log Integrity

### Verify Events Are Flowing

```bash
# Check NATS health
curl http://localhost:8222/healthz

# Check audit consumer
curl http://localhost:8222/jsz?consumers=true | jq '.[] | .consumer_count'

# Query recent events
curl -s "$GW/api/v1/audit/events?tenant_id=$TENANT&page_size=5" \
  -H "Authorization: Bearer $TOKEN"
```

### Tamper Protection

Audit events are written to PostgreSQL with an immutable append-only pattern:

```sql
-- Prevent UPDATE and DELETE on audit_events
CREATE OR REPLACE RULE no_update AS ON UPDATE TO audit_events DO INSTEAD NOTHING;
CREATE OR REPLACE RULE no_delete AS ON DELETE TO audit_events DO INSTEAD NOTHING;
```

### Retention Policy

Configure retention to balance compliance and storage:

```bash
# Set retention via API
curl -X PUT "$GW/api/v1/audit/retention" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"days": 365, "enabled": true}'
```

| Compliance | Recommended Retention |
|------------|----------------------|
| SOC 2 | 1 year |
| HIPAA | 6 years |
| PCI DSS | 1 year |
| GDPR | Minimize (typically 90 days) |

---

## Least Privilege Principle

### Database Access

| Role | Privileges | Used By |
|------|------------|---------|
| `ggid_app` | SELECT, INSERT, UPDATE, DELETE on tables | All services |
| `ggid_migrate` | CREATE, ALTER, DROP, INDEX | Migration init container only |
| `ggid_readonly` | SELECT only | Reporting/analytics |

Create the readonly role:
```sql
CREATE ROLE ggid_readonly WITH LOGIN PASSWORD 'STRONG_PASSWORD';
GRANT CONNECT ON DATABASE ggid TO ggid_readonly;
GRANT USAGE ON SCHEMA public TO ggid_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO ggid_readonly;
```

### Service-to-Service Access

Services should NOT call each other directly. All external traffic goes through
the Gateway. Internal calls (if any) should use service accounts with limited scopes.

### API Key Scopes

When creating API keys for machine-to-machine access:

```bash
# Create a scoped API key (read-only users)
curl -X POST "$GW/api/v1/apikeys" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "ci-cd-read-only",
    "scopes": ["users:read", "roles:read"],
    "expires_in": 86400
  }'
```

---

## Secrets Management

### What Needs to Be Secret

| Secret | Where Used | Storage |
|--------|-----------|---------|
| RSA private key | Auth service | Vault / KMS |
| PostgreSQL password | All services | Vault / K8s Secret |
| Redis password | Auth service | Vault / K8s Secret |
| NATS password | Audit publisher | Vault / K8s Secret |
| LDAP bind password | Auth service | Vault / K8s Secret |
| SMTP password | Auth service | Vault / K8s Secret |
| OAuth client secrets | OAuth service | Encrypted in DB |

### Kubernetes Secret Management

```yaml
# Use external-secrets-operator to sync from Vault/ASM
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: ggid-auth-secrets
spec:
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: ggid-auth-secrets
  data:
    - secretKey: rsa-private-key
      remoteRef:
        key: ggid/auth/rsa-private-key
    - secretKey: database-url
      remoteRef:
        key: ggid/auth/database-url
```

### Docker Secrets

```bash
echo "STRONG_DB_PASSWORD" | docker secret create ggid_db_password -
echo "RSA_PRIVATE_KEY_CONTENT" | docker secret create ggid_rsa_private -

# Reference in docker-compose.yaml
services:
  auth:
    secrets:
      - ggid_db_password
      - ggid_rsa_private
secrets:
  ggid_db_password:
    external: true
  ggid_rsa_private:
    external: true
```

---

## Network Security

### Docker Network Isolation

```yaml
# docker-compose.yaml — separate networks
services:
  gateway:
    networks:
      - frontend
      - backend
  auth:
    networks:
      - backend
  postgres:
    networks:
      - backend
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # no external access
```

### Kubernetes Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ggid-backend-deny-all
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/part-of: ggid
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Only allow traffic from the Gateway
    - from:
        - podSelector:
            matchLabels:
              app: gateway
      ports:
        - protocol: TCP
          port: 8080
  egress:
    # Allow DNS
    - to: []
      ports:
        - protocol: UDP
          port: 53
    # Allow PostgreSQL
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - protocol: TCP
          port: 5432
    # Allow Redis
    - to:
        - podSelector:
            matchLabels:
              app: redis
      ports:
        - protocol: TCP
          port: 6379
```

---

## Compliance

### SOC 2 Alignment

| Control | GGID Support |
|---------|-------------|
| CC6.1 (Logical Access) | RBAC + ABAC, MFA, session management |
| CC6.6 (Logical Access) | JWT-based auth, token expiry, revocation |
| CC7.1 (Detection) | Audit events via NATS, real-time streaming |
| CC7.2 (Monitoring) | Prometheus metrics, Grafana dashboards |
| CC7.3 (Evaluation) | Anomaly detection rules in Audit service |

### GDPR Alignment

| Requirement | GGID Support |
|-------------|-------------|
| Data minimization | Configurable audit retention |
| Right to erasure | `DELETE /api/v1/users/{id}` cascades to all tables |
| Data portability | CSV export for users, JSON export for policies/audit |
| Consent management | Audit trail of user actions |

### Regular Security Tasks

| Task | Frequency |
|------|-----------|
| Rotate JWT signing keys | Every 90 days |
| Rotate database passwords | Every 90 days |
| Review API key access | Monthly |
| Audit user role assignments | Monthly |
| Scan container images | On every build |
| Run `govulncheck` | On every PR |
| Review audit logs for anomalies | Daily |
| Test backup restoration | Quarterly |
| Penetration testing | Annually |
