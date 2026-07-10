# GGID Environment Variables Reference

Complete reference for all environment variables across every GGID service.

---

## Table of Contents

- [Gateway](#gateway)
- [Auth Service](#auth-service)
- [Identity Service](#identity-service)
- [OAuth Service](#oauth-service)
- [Policy Service](#policy-service)
- [Org Service](#org-service)
- [Audit Service](#audit-service)
- [Console](#console)
- [Infrastructure](#infrastructure)

---

## Gateway

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `GATEWAY_ADDR` | No | `:8080` | HTTP listen address |
| `GATEWAY_DOMAIN_SUFFIX` | No | _(empty)_ | Domain suffix for routing (e.g. `.iam.example.com`) |
| `JWT_PUBLIC_KEY_PATH` | No | `configs/rsa_public.pem` | RSA public key file for JWT verification |
| `GATEWAY_JWKS_URL` | No | _(empty)_ | JWKS endpoint URL. If set, overrides local key file |
| `GATEWAY_JWT_ISSUER` | No | `ggid-auth` | Expected `iss` claim in JWT |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | _(empty)_ | OTLP HTTP endpoint for OpenTelemetry tracing |
| `AUTH_SERVICE_URL` | No | `http://auth:9001` | Auth service backend URL |
| `IDENTITY_SERVICE_URL` | No | `http://identity:8080` | Identity service backend URL |
| `ROLES_SERVICE_URL` | No | `http://policy:8070` | Policy service URL (roles route) |
| `PERMISSIONS_SERVICE_URL` | No | `http://policy:8070` | Policy service URL (permissions route) |
| `POLICY_SERVICE_URL` | No | `http://policy:8070` | Policy service URL (policies route) |
| `ORG_SERVICE_URL` | No | `http://org:8071` | Org service backend URL |
| `AUDIT_SERVICE_URL` | No | `http://audit:8072` | Audit service backend URL |
| `OAUTH_SERVICE_URL` | No | `http://oauth:9005` | OAuth service backend URL |
| `SAML_SERVICE_URL` | No | `http://oauth:9005` | SAML service backend URL |

---

## Auth Service

### Core

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `AUTH_HTTP_ADDR` | No | `:9001` | HTTP listen address |
| `DATABASE_URL` | **Yes** | _(none)_ | PostgreSQL connection string |

### Redis

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `REDIS_ADDR` | No | `redis:6379` | Redis address for rate limiting and sessions |
| `REDIS_PASSWORD` | No | _(empty)_ | Redis auth password |

### JWT / Crypto

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `JWT_PRIVATE_KEY_PATH` | No | `/configs/rsa_private.pem` | RSA private key for JWT signing |
| `JWT_PUBLIC_KEY_PATH` | No | `/configs/rsa_public.pem` | RSA public key |

### LDAP / AD

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `LDAP_URL` | No | _(empty)_ | LDAP server URL (e.g. `ldap://host:389`). If empty, LDAP is disabled |
| `LDAP_BIND_DN` | No | _(empty)_ | LDAP bind DN (e.g. `cn=admin,dc=corp,dc=local`) |
| `LDAP_BIND_PASSWORD` | No | _(empty)_ | LDAP bind password |
| `LDAP_BASE_DN` | No | _(empty)_ | LDAP search base (e.g. `dc=corp,dc=local`) |
| `LDAP_USER_FILTER` | No | `(uid=%s)` | LDAP user filter template |
| `LDAP_START_TLS` | No | `false` | Enable StartTLS for LDAP connection |
| `LDAP_AUTO_PROVISION` | No | `false` | Auto-create GGID user on first LDAP login |

### Social Login

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `AUTH_GOOGLE_CLIENT_ID` | No | _(empty)_ | Google OAuth client ID |
| `AUTH_GOOGLE_CLIENT_SECRET` | No | _(empty)_ | Google OAuth client secret |
| `AUTH_GOOGLE_REDIRECT_URL` | No | _(auto)_ | Google OAuth callback URL |
| `AUTH_GITHUB_CLIENT_ID` | No | _(empty)_ | GitHub OAuth client ID |
| `AUTH_GITHUB_CLIENT_SECRET` | No | _(empty)_ | GitHub OAuth client secret |
| `AUTH_MICROSOFT_CLIENT_ID` | No | _(empty)_ | Microsoft OAuth client ID |
| `AUTH_MICROSOFT_CLIENT_SECRET` | No | _(empty)_ | Microsoft OAuth client secret |
| `AUTH_DISCORD_CLIENT_ID` | No | _(empty)_ | Discord OAuth client ID |
| `AUTH_DISCORD_CLIENT_SECRET` | No | _(empty)_ | Discord OAuth client secret |

### SMTP (Email)

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `SMTP_HOST` | No | _(empty)_ | SMTP server hostname |
| `SMTP_PORT` | No | `587` | SMTP server port |
| `SMTP_USERNAME` | No | _(empty)_ | SMTP auth username |
| `SMTP_PASSWORD` | No | _(empty)_ | SMTP auth password |
| `SMTP_FROM_EMAIL` | No | `noreply@ggid.dev` | Sender email address |
| `SMTP_FROM_NAME` | No | `GGID` | Sender display name |

---

## Identity Service

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `DATABASE_URL` | **Yes** | _(none)_ | PostgreSQL connection string |

---

## OAuth Service

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `DATABASE_URL` | **Yes** | _(none)_ | PostgreSQL connection string |
| `OAUTH_PRIVATE_KEY_PATH` | No | `/configs/rsa_private.pem` | RSA private key for token signing |
| `OAUTH_PUBLIC_KEY_PATH` | No | `/configs/rsa_public.pem` | RSA public key |

---

## Policy Service

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `DB_HOST` | No | `postgres` | Database host |
| `DB_PORT` | No | `5432` | Database port |
| `DB_USER` | **Yes** | _(none)_ | Database username |
| `DB_PASSWORD` | **Yes** | _(none)_ | Database password |
| `DB_DATABASE` | No | `ggid` | Database name |
| `DB_SSL_MODE` | No | `disable` | PostgreSQL SSL mode (`disable\|require\|verify-full`) |

---

## Org Service

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `DB_HOST` | No | `postgres` | Database host |
| `DB_PORT` | No | `5432` | Database port |
| `DB_USER` | **Yes** | _(none)_ | Database username |
| `DB_PASSWORD` | **Yes** | _(none)_ | Database password |
| `DB_DATABASE` | No | `ggid` | Database name |
| `DB_SSL_MODE` | No | `disable` | PostgreSQL SSL mode |

---

## Audit Service

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `DB_HOST` | No | `postgres` | Database host |
| `DB_PORT` | No | `5432` | Database port |
| `DB_USER` | **Yes** | _(none)_ | Database username |
| `DB_PASSWORD` | **Yes** | _(none)_ | Database password |
| `DB_DATABASE` | No | `ggid` | Database name |
| `DB_SSL_MODE` | No | `disable` | PostgreSQL SSL mode |
| `NATS_URL` | No | `nats://nats:4222` | NATS connection URL |

---

## Console

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `GATEWAY_URL` | No | `http://gateway:8080` | Gateway URL (internal Docker network) |
| `NEXT_PUBLIC_GATEWAY_URL` | No | `http://localhost:8080` | Gateway URL (browser-side) |
| `HOSTNAME` | No | `0.0.0.0` | Next.js listen hostname (Docker binding) |
| `PORT` | No | `3000` | Next.js listen port |

---

## Infrastructure

### PostgreSQL

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| `POSTGRES_USER` | **Yes** | `ggid` | Superuser username |
| `POSTGRES_PASSWORD` | **Yes** | _(none)_ | Superuser password (**change in production**) |
| `POSTGRES_DB` | No | `ggid` | Default database name |

### Redis

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| _(configured via `redis.conf`)_ | — | — | See [Security Hardening](./security-hardening.md#redis-hardening) |

### NATS

| Variable | Required | Default | Description |
|----------|:--------:|---------|-------------|
| _(configured via `nats-server.conf`)_ | — | — | See [Security Hardening](./security-hardening.md#nats-hardening) |

---

## Connection String Format

### Auth / Identity / OAuth (DATABASE_URL)

```
postgres://USER:PASSWORD@HOST:PORT/DATABASE?sslmode=SSL_MODE
```

Example:
```
DATABASE_URL=postgres://ggid_app:s3cret@postgres:5432/ggid?sslmode=require
```

### Policy / Org / Audit (individual DB_ vars)

```bash
DB_HOST=postgres
DB_PORT=5432
DB_USER=ggid_app
DB_PASSWORD=s3cret
DB_DATABASE=ggid
DB_SSL_MODE=require
```

---

## Docker Compose Example

```yaml
# deploy/docker-compose.yaml (excerpt)

services:
  gateway:
    environment:
      GATEWAY_ADDR: ":8080"
      JWT_PUBLIC_KEY_PATH: /configs/rsa_public.pem
      GATEWAY_JWT_ISSUER: ggid-auth
      AUTH_SERVICE_URL: http://auth:9001
      IDENTITY_SERVICE_URL: http://identity:8080
      POLICY_SERVICE_URL: http://policy:8070
      ORG_SERVICE_URL: http://org:8071
      AUDIT_SERVICE_URL: http://audit:8072
      OAUTH_SERVICE_URL: http://oauth:9005

  auth:
    environment:
      AUTH_HTTP_ADDR: ":9001"
      DATABASE_URL: postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable
      REDIS_ADDR: redis:6379
      JWT_PRIVATE_KEY_PATH: /configs/rsa_private.pem
      JWT_PUBLIC_KEY_PATH: /configs/rsa_public.pem

  policy:
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: ggid
      DB_PASSWORD: ggid
      DB_DATABASE: ggid
```

---

## Kubernetes Secret Example

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ggid-auth-secrets
type: Opaque
stringData:
  DATABASE_URL: postgres://ggid_app:s3cret@postgres:5432/ggid?sslmode=require
  REDIS_PASSWORD: strong-redis-password
  LDAP_BIND_PASSWORD: ldap-secret
  SMTP_PASSWORD: smtp-secret
---
# In deployment:
spec:
  containers:
    - name: auth
      envFrom:
        - secretRef:
            name: ggid-auth-secrets
```
