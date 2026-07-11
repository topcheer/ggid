# External Database Setup

> Connect GGID to your own PostgreSQL, Redis, NATS, and LDAP instead of bundled containers.

---

## Quick Start

Set these environment variables before starting any GGID deployment:

```bash
DB_HOST=prod-db.internal
DB_PORT=5432
DB_USER=ggid
DB_PASSWORD=secure-password
DB_NAME=ggid
DB_SSL_MODE=require

REDIS_HOST=prod-redis.internal
REDIS_PORT=6379
REDIS_PASSWORD=redis-password

NATS_URL=nats://prod-nats.internal:4222

LDAP_URL=ldap://prod-ldap.internal:389
LDAP_BIND_DN=cn=admin,dc=example,dc=com
LDAP_BIND_PASSWORD=ldap-password
LDAP_BASE_DN=ou=users,dc=example,dc=com
```

---

## Docker Compose

### Option A: `.env` file

Create `deploy/.env`:

```bash
# External PostgreSQL
DB_HOST=prod-db.internal
DB_PORT=5432
DB_USER=ggid
DB_PASSWORD=secure-password
DB_SSL_MODE=require

# External Redis
REDIS_HOST=prod-redis.internal
REDIS_PORT=6379

# External NATS
NATS_URL=nats://prod-nats.internal:4222

# External LDAP (optional)
LDAP_URL=ldap://prod-ldap.internal:389
LDAP_BASE_DN=ou=users,dc=example,dc=com
```

Start only GGID services (skip bundled infra):

```bash
cd deploy && docker compose up -d gateway identity auth oauth policy org audit console
```

### Option B: Override file

Create `deploy/docker-compose.override.yml`:

```yaml
services:
  # Remove bundled infrastructure
  postgres:
    profiles: ["bundled"]
  redis:
    profiles: ["bundled"]
  nats:
    profiles: ["bundled"]
  ldap:
    profiles: ["bundled"]

  # Point to external services
  gateway:
    environment:
      DB_HOST: prod-db.internal
      REDIS_HOST: prod-redis.internal
      NATS_URL: nats://prod-nats.internal:4222

  auth:
    environment:
      DB_HOST: prod-db.internal
      REDIS_HOST: prod-redis.internal
      LDAP_URL: ldap://prod-ldap.internal:389

  policy:
    environment:
      DB_HOST: prod-db.internal

  org:
    environment:
      DB_HOST: prod-db.internal

  audit:
    environment:
      DB_HOST: prod-db.internal
      NATS_URL: nats://prod-nats.internal:4222
```

---

## Helm / Kubernetes

Create `values-external.yaml`:

```yaml
# Disable bundled infrastructure
postgresql:
  enabled: false

redis:
  enabled: false

nats:
  enabled: false

# Point to external services
externalDatabase:
  host: prod-db.internal
  port: 5432
  user: ggid
  password: secure-password
  database: ggid
  sslMode: require

externalRedis:
  host: prod-redis.internal
  port: 6379
  password: redis-password

externalNats:
  url: nats://prod-nats.internal:4222

# All GGID services inherit these automatically
```

Deploy:

```bash
helm install ggid deploy/helm/ggid -f values-external.yaml
```

---

## Terraform

In `terraform.tfvars`:

```hcl
external_database_host = "prod-db.internal"
external_database_port = 5432
external_database_password = "secure-password"

external_redis_host = "prod-redis.internal"
external_redis_port = 6379

external_nats_host = "prod-nats.internal"
external_nats_port = 4222

external_ldap_url = "ldap://prod-ldap.internal:389"
```

Apply:

```bash
terraform apply -var-file=terraform.tfvars
```

---

## Bare Metal / systemd

Set environment variables in the systemd unit file:

```ini
[Service]
Environment=DB_HOST=prod-db.internal
Environment=DB_PORT=5432
Environment=DB_USER=ggid
Environment=DB_PASSWORD=secure-password
Environment=DB_SSL_MODE=require
Environment=REDIS_HOST=prod-redis.internal
Environment=NATS_URL=nats://prod-nats.internal:4222
Environment=LDAP_URL=ldap://prod-ldap.internal:389
ExecStart=/usr/local/bin/ggid-gateway
```

---

## Database Initialization

GGID requires these databases/schemas to be pre-created:

```sql
CREATE DATABASE ggid;
CREATE USER ggid WITH PASSWORD 'secure-password';
GRANT ALL PRIVILEGES ON DATABASE ggid TO ggid;

-- Enable RLS extension
\c ggid
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

The migration tool runs automatically on startup (idempotent — skips if tables exist).

---

## Verification

After starting with external DB:

```bash
# Check health (should report healthy)
curl http://localhost:8080/healthz

# Register a test user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@test.com","password":"Test1234!"}'

# If 201 Created → external DB is working
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `connection refused` | DB not reachable | Check firewall, security groups, `DB_HOST` |
| `SSL required` | DB requires SSL | Set `DB_SSL_MODE=require` |
| `authentication failed` | Wrong credentials | Verify `DB_USER`, `DB_PASSWORD` |
| `NATS: no servers available` | NATS URL wrong | Check `NATS_URL` format (`nats://host:port`) |
| `LDAP: connection error` | LDAP unreachable | Verify `LDAP_URL`, `LDAP_START_TLS=true` if needed |

---

*See: [Docker Compose Override](../deploy/docker-compose-override.md) | [Helm Chart Guide](../deploy/helm-chart-guide.md) | [Production Checklist](../deploy/production-checklist.md)*

*Last updated: 2025-07-11*
