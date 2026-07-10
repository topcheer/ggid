# GGID Configuration Reference

Complete configuration reference for all GGID services.

---

## Table of Contents

- [Configuration Sources](#configuration-sources)
- [Gateway](#gateway)
- [Auth Service](#auth-service)
- [Identity Service](#identity-service)
- [OAuth Service](#oauth-service)
- [Policy Service](#policy-service)
- [Org Service](#org-service)
- [Audit Service](#audit-service)
- [Console](#console)

---

## Configuration Sources

GGID services read configuration in this order (later overrides earlier):

1. Built-in defaults
2. Environment variables
3. Command-line flags

Environment variables are the primary configuration method in Docker/K8s deployments.

---

## Gateway

### Core

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `GATEWAY_ADDR` | string | `:8080` | HTTP listen address |
| `GATEWAY_DOMAIN_SUFFIX` | string | _(empty)_ | Domain suffix for routing (e.g., `.iam.example.com`) |

### JWT / Authentication

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `GATEWAY_JWKS_URL` | string | _(empty)_ | JWKS endpoint URL. If set, overrides local key |
| `GATEWAY_JWT_ISSUER` | string | `ggid-auth` | Expected `iss` claim in JWT |
| `JWT_PUBLIC_KEY_PATH` | string | `configs/rsa_public.pem` | RSA public key file path |

### Backend Service URLs

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `AUTH_SERVICE_URL` | string | `http://auth:9001` | Auth service URL |
| `IDENTITY_SERVICE_URL` | string | `http://identity:8080` | Identity service URL |
| `ROLES_SERVICE_URL` | string | `http://policy:8070` | Policy service URL (roles route) |
| `PERMISSIONS_SERVICE_URL` | string | `http://policy:8070` | Policy service URL (permissions route) |
| `POLICY_SERVICE_URL` | string | `http://policy:8070` | Policy service URL (policies route) |
| `ORG_SERVICE_URL` | string | `http://org:8071` | Org service URL |
| `AUDIT_SERVICE_URL` | string | `http://audit:8072` | Audit service URL |
| `OAUTH_SERVICE_URL` | string | `http://oauth:9005` | OAuth service URL |
| `SAML_SERVICE_URL` | string | `http://oauth:9005` | SAML service URL |

### Observability

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | string | _(empty)_ | OTLP HTTP endpoint for tracing |
| `OTEL_SERVICE_NAME` | string | `ggid-gateway` | Service name in traces |

### Example

```bash
GATEWAY_ADDR=:8080
JWT_PUBLIC_KEY_PATH=/configs/rsa_public.pem
GATEWAY_JWT_ISSUER=ggid-auth
AUTH_SERVICE_URL=http://auth:9001
IDENTITY_SERVICE_URL=http://identity:8080
POLICY_SERVICE_URL=http://policy:8070
ORG_SERVICE_URL=http://org:8071
AUDIT_SERVICE_URL=http://audit:8072
OAUTH_SERVICE_URL=http://oauth:9005
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

---

## Auth Service

### Core

| Env Var | Type | Default | Required | Description |
|---------|------|---------|:--------:|-------------|
| `AUTH_HTTP_ADDR` | string | `:9001` | No | HTTP listen address |
| `DATABASE_URL` | string | _(none)_ | **Yes** | PostgreSQL connection string |
| `REDIS_ADDR` | string | `redis:6379` | No | Redis address |
| `REDIS_PASSWORD` | string | _(empty)_ | No | Redis auth password |

### JWT / Crypto

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `JWT_PRIVATE_KEY_PATH` | string | `/configs/rsa_private.pem` | RSA private key for JWT signing |
| `JWT_PUBLIC_KEY_PATH` | string | `/configs/rsa_public.pem` | RSA public key |
| `JWT_ACCESS_TOKEN_TTL` | duration | `1h` | Access token lifetime |
| `JWT_REFRESH_TOKEN_TTL` | duration | `720h` (30d) | Refresh token lifetime |

### LDAP / AD

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `LDAP_URL` | string | _(empty)_ | LDAP URL. If empty, LDAP disabled |
| `LDAP_BIND_DN` | string | _(empty)_ | LDAP bind DN |
| `LDAP_BIND_PASSWORD` | string | _(empty)_ | LDAP bind password |
| `LDAP_BASE_DN` | string | _(empty)_ | LDAP search base DN |
| `LDAP_USER_FILTER` | string | `(uid=%s)` | User filter template |
| `LDAP_START_TLS` | bool | `false` | Enable StartTLS |
| `LDAP_AUTO_PROVISION` | bool | `false` | Auto-create user on first LDAP login |

### Social Login

| Env Var | Type | Description |
|---------|------|-------------|
| `AUTH_GOOGLE_CLIENT_ID` | string | Google OAuth client ID |
| `AUTH_GOOGLE_CLIENT_SECRET` | string | Google OAuth client secret |
| `AUTH_GOOGLE_REDIRECT_URL` | string | Google callback URL |
| `AUTH_GITHUB_CLIENT_ID` | string | GitHub OAuth client ID |
| `AUTH_GITHUB_CLIENT_SECRET` | string | GitHub OAuth client secret |
| `AUTH_MICROSOFT_CLIENT_ID` | string | Microsoft OAuth client ID |
| `AUTH_MICROSOFT_CLIENT_SECRET` | string | Microsoft OAuth client secret |
| `AUTH_DISCORD_CLIENT_ID` | string | Discord OAuth client ID |
| `AUTH_DISCORD_CLIENT_SECRET` | string | Discord OAuth client secret |

### SMTP (Email)

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `SMTP_HOST` | string | _(empty)_ | SMTP server hostname |
| `SMTP_PORT` | int | `587` | SMTP port |
| `SMTP_USERNAME` | string | _(empty)_ | SMTP auth username |
| `SMTP_PASSWORD` | string | _(empty)_ | SMTP auth password |
| `SMTP_FROM_EMAIL` | string | `noreply@ggid.dev` | Sender address |
| `SMTP_FROM_NAME` | string | `GGID` | Sender display name |

### NATS (Audit Events)

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `NATS_URL` | string | `nats://nats:4222` | NATS connection URL |

### Example

```bash
AUTH_HTTP_ADDR=:9001
DATABASE_URL=postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable
REDIS_ADDR=redis:6379
JWT_PRIVATE_KEY_PATH=/configs/rsa_private.pem
JWT_PUBLIC_KEY_PATH=/configs/rsa_public.pem

LDAP_URL=ldap://ldap:389
LDAP_BIND_DN=cn=admin,dc=ggid,dc=local
LDAP_BIND_PASSWORD=admin
LDAP_BASE_DN=dc=ggid,dc=local
LDAP_USER_FILTER=(uid=%s)

AUTH_GOOGLE_CLIENT_ID=xxx.apps.googleusercontent.com
AUTH_GOOGLE_CLIENT_SECRET=xxx

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=noreply@example.com
SMTP_PASSWORD=app-password

NATS_URL=nats://nats:4222
```

---

## Identity Service

| Env Var | Type | Default | Required | Description |
|---------|------|---------|:--------:|-------------|
| `DATABASE_URL` | string | _(none)_ | **Yes** | PostgreSQL connection string |
| `NATS_URL` | string | `nats://nats:4222` | No | NATS for audit events |

### Example

```bash
DATABASE_URL=postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable
NATS_URL=nats://nats:4222
```

---

## OAuth Service

| Env Var | Type | Default | Required | Description |
|---------|------|---------|:--------:|-------------|
| `DATABASE_URL` | string | _(none)_ | **Yes** | PostgreSQL connection string |
| `OAUTH_PRIVATE_KEY_PATH` | string | `/configs/rsa_private.pem` | RSA signing key |
| `OAUTH_PUBLIC_KEY_PATH` | string | `/configs/rsa_public.pem` | RSA public key |

### Example

```bash
DATABASE_URL=postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable
OAUTH_PRIVATE_KEY_PATH=/configs/rsa_private.pem
OAUTH_PUBLIC_KEY_PATH=/configs/rsa_public.pem
```

---

## Policy Service

Uses individual DB environment variables (not `DATABASE_URL`):

| Env Var | Type | Default | Required | Description |
|---------|------|---------|:--------:|-------------|
| `DB_HOST` | string | `postgres` | No | Database host |
| `DB_PORT` | int | `5432` | No | Database port |
| `DB_USER` | string | `ggid` | **Yes** | Database username |
| `DB_PASSWORD` | string | _(none)_ | **Yes** | Database password |
| `DB_DATABASE` | string | `ggid` | No | Database name |
| `DB_SSL_MODE` | string | `disable` | No | SSL mode: `disable\|require\|verify-full` |
| `GRPC_PORT` | string | `:9070` | No | gRPC listen address |

### Example

```bash
DB_HOST=postgres
DB_PORT=5432
DB_USER=ggid
DB_PASSWORD=ggid
DB_DATABASE=ggid
DB_SSL_MODE=disable
GRPC_PORT=:9070
```

---

## Org Service

Same DB env var pattern as Policy:

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `DB_HOST` | string | `postgres` | Database host |
| `DB_PORT` | int | `5432` | Database port |
| `DB_USER` | string | `ggid` | Database username |
| `DB_PASSWORD` | string | _(none)_ | Database password |
| `DB_DATABASE` | string | `ggid` | Database name |
| `DB_SSL_MODE` | string | `disable` | SSL mode |
| `GRPC_PORT` | string | `:9071` | gRPC listen address |

---

## Audit Service

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `DB_HOST` | string | `postgres` | Database host |
| `DB_PORT` | int | `5432` | Database port |
| `DB_USER` | string | `ggid` | Database username |
| `DB_PASSWORD` | string | _(none)_ | Database password |
| `DB_DATABASE` | string | `ggid` | Database name |
| `DB_SSL_MODE` | string | `disable` | SSL mode |
| `NATS_URL` | string | `nats://nats:4222` | NATS for consuming events |
| `GRPC_PORT` | string | `:9072` | gRPC listen address |

---

## Console

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `GATEWAY_URL` | string | `http://gateway:8080` | Gateway URL (server-side) |
| `NEXT_PUBLIC_GATEWAY_URL` | string | `http://localhost:8080` | Gateway URL (browser-side) |
| `HOSTNAME` | string | `0.0.0.0` | Next.js listen hostname |
| `PORT` | int | `3000` | Next.js listen port |

### Example

```bash
GATEWAY_URL=http://gateway:8080
NEXT_PUBLIC_GATEWAY_URL=https://iam.example.com
HOSTNAME=0.0.0.0
PORT=3000
```

---

## DATABASE_URL Format

Auth, Identity, and OAuth services use `DATABASE_URL`:

```
postgres://USER:PASSWORD@HOST:PORT/DATABASE?sslmode=SSL_MODE
```

Examples:

```bash
# Development (no TLS)
DATABASE_URL=postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable

# Production (TLS required)
DATABASE_URL=postgres://ggid_app:s3cret@db.internal:5432/ggid?sslmode=require

# Production (full verification)
DATABASE_URL=postgres://ggid_app:s3cret@db.internal:5432/ggid?sslmode=verify-full
```

### SSL Modes

| Mode | Description |
|------|-------------|
| `disable` | No TLS (development only) |
| `require` | TLS required, no certificate verification |
| `verify-ca` | TLS required, verify server CA |
| `verify-full` | TLS required, verify CA + hostname (recommended) |

---

## Duration Format

Duration values support Go duration syntax:

| Value | Meaning |
|-------|---------|
| `30s` | 30 seconds |
| `5m` | 5 minutes |
| `1h` | 1 hour |
| `720h` | 30 days |
| `1h30m` | 1 hour 30 minutes |

---

## Boolean Format

Boolean values accept:

| True | False |
|------|-------|
| `true` | `false` |
| `1` | `0` |
| `yes` | `no` |
| `on` | `off` |

---

## Complete Docker Compose Example

```yaml
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
      LDAP_URL: ldap://ldap:389
      LDAP_BIND_DN: cn=admin,dc=ggid,dc=local
      LDAP_BIND_PASSWORD: admin
      LDAP_BASE_DN: dc=ggid,dc=local
      LDAP_AUTO_PROVISION: "true"
      NATS_URL: nats://nats:4222

  identity:
    environment:
      DATABASE_URL: postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable

  oauth:
    environment:
      DATABASE_URL: postgres://ggid:ggid@postgres:5432/ggid?sslmode=disable
      OAUTH_PRIVATE_KEY_PATH: /configs/rsa_private.pem
      OAUTH_PUBLIC_KEY_PATH: /configs/rsa_public.pem

  policy:
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: ggid
      DB_PASSWORD: ggid
      DB_DATABASE: ggid

  org:
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: ggid
      DB_PASSWORD: ggid
      DB_DATABASE: ggid

  audit:
    environment:
      DB_HOST: postgres
      DB_PORT: "5432"
      DB_USER: ggid
      DB_PASSWORD: ggid
      DB_DATABASE: ggid
      NATS_URL: nats://nats:4222

  console:
    environment:
      GATEWAY_URL: http://gateway:8080
      NEXT_PUBLIC_GATEWAY_URL: http://localhost:8080
      HOSTNAME: "0.0.0.0"
      PORT: "3000"
```
