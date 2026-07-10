# GGID Configuration Reference

Complete reference for all environment variables across all GGID services.
Required variables must be set; optional variables have sensible defaults.

---

## Gateway Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP listen port |
| `AUTH_SERVICE_URL` | Yes | — | Auth service address (`host:9001`) |
| `IDENTITY_SERVICE_URL` | Yes | — | Identity gRPC address (`host:50051`) |
| `POLICY_SERVICE_URL` | Yes | — | Policy gRPC address (`host:9070`) |
| `ORG_SERVICE_URL` | Yes | — | Org gRPC address (`host:9071`) |
| `AUDIT_SERVICE_URL` | Yes | — | Audit gRPC address (`host:9072`) |
| `OAUTH_SERVICE_URL` | Yes | — | OAuth service address (`host:9005`) |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `REDIS_URL` | No | `redis://localhost:6379` | Redis connection string |
| `JWT_ISSUER` | No | `https://iam.example.com` | JWT issuer claim |
| `JWKS_URL` | No | Auto-discovered | JWKS endpoint URL |
| `NATS_URL` | No | `nats://localhost:4222` | NATS connection string |
| `RATELIMIT_FAIL_MODE` | No | `open` | `open` or `closed` when Redis down |
| `LOG_LEVEL` | No | `info` | `debug`/`info`/`warn`/`error` |
| `LOG_FORMAT` | No | `json` | `json` or `text` |
| `CORS_ALLOWED_ORIGINS` | No | `*` | Comma-separated origin list |
| `TLS_CERT` | No | — | TLS certificate path |
| `TLS_KEY` | No | — | TLS private key path |

---

## Auth Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `9001` | Listen port |
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `REDIS_URL` | No | `redis://localhost:6379` | Redis for sessions/rate limit |
| `JWT_SECRET` | Conditional | — | HS256 signing key (if using HS256) |
| `JWT_PRIVATE_KEY` | Conditional | — | RS256 private key (PEM) |
| `JWT_PUBLIC_KEY` | Conditional | — | RS256 public key (PEM) |
| `JWT_ISSUER` | No | `https://iam.example.com` | JWT issuer claim |
| `JWT_ACCESS_TTL` | No | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | No | `168h` | Refresh token lifetime (7 days) |
| `BCRYPT_COST` | No | `12` | bcrypt cost factor (4-31) |
| `ARGON2_MEMORY` | No | `65536` | Argon2id memory in KB |
| `ARGON2_ITERATIONS` | No | `3` | Argon2id iterations |
| `ARGON2_PARALLELISM` | No | `2` | Argon2id parallelism |
| `PASSWORD_MIN_LENGTH` | No | `12` | Minimum password length |
| `PASSWORD_REQUIRE_UPPERCASE` | No | `true` | Require uppercase letter |
| `PASSWORD_REQUIRE_LOWERCASE` | No | `true` | Require lowercase letter |
| `PASSWORD_REQUIRE_DIGIT` | No | `true` | Require digit |
| `PASSWORD_REQUIRE_SYMBOL` | No | `false` | Require special character |
| `LOCKOUT_THRESHOLD` | No | `5` | Failed attempts before lockout |
| `LOCKOUT_DURATION` | No | `15m` | Lockout duration |
| `LDAP_URL` | No | — | LDAP server URL (`ldap://host:389`) |
| `LDAP_BIND_DN` | No | — | LDAP service account DN |
| `LDAP_BIND_PASSWORD` | No | — | LDAP service account password |
| `LDAP_BASE_DN` | No | — | LDAP base DN for user search |
| `LDAP_USER_FILTER` | No | `(uid=%s)` | LDAP user search filter |
| `LDAP_START_TLS` | No | `false` | Use START_TLS for LDAP |
| `LDAP_AUTO_PROVISION` | No | `false` | Auto-create users on LDAP login |
| `MFA_TOTP_ISSUER` | No | `GGID` | TOTP app display name |
| `MFA_ENABLED` | No | `true` | Enable MFA features |
| `WEBAUTHN_RP_ID` | No | `localhost` | WebAuthn relying party ID |
| `WEBAUTHN_RP_NAME` | No | `GGID` | WebAuthn display name |
| `WEBAUTHN_ORIGIN` | No | `http://localhost:8080` | WebAuthn expected origin |
| `NATS_URL` | No | `nats://localhost:4222` | NATS for audit events |
| `LOG_LEVEL` | No | `info` | Log level |

---

## Identity Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GRPC_PORT` | No | `50051` | gRPC listen port |
| `HTTP_PORT` | No | `8081` | HTTP listen port |
| `DB_HOST` | Yes | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | Yes | — | PostgreSQL user |
| `DB_PASSWORD` | Yes | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `DB_SSLMODE` | No | `disable` | `disable`/`require`/`verify-full` |
| `SCIM_ENABLED` | No | `true` | Enable SCIM 2.0 endpoints |
| `LOG_LEVEL` | No | `info` | Log level |

---

## OAuth Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `9005` | Listen port |
| `DB_HOST` | Yes | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | Yes | — | PostgreSQL user |
| `DB_PASSWORD` | Yes | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `JWT_PRIVATE_KEY` | Yes | — | RS256 private key (PEM) |
| `JWT_PUBLIC_KEY` | Yes | — | RS256 public key (PEM) |
| `JWT_ISSUER` | No | `https://iam.example.com` | JWT issuer |
| `AUTHCODE_TTL` | No | `10m` | Authorization code lifetime |
| `ACCESSTOKEN_TTL` | No | `1h` | OAuth access token lifetime |
| `REFRESHTOKEN_TTL` | No | `720h` | Refresh token lifetime (30 days) |
| `PKCE_REQUIRED` | No | `false` | Require PKCE for public clients |
| `LOG_LEVEL` | No | `info` | Log level |

---

## Policy Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `8070` | HTTP listen port |
| `GRPC_PORT` | No | `9070` | gRPC listen port |
| `DB_HOST` | Yes | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | Yes | — | PostgreSQL user |
| `DB_PASSWORD` | Yes | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `POLICY_CACHE_SIZE` | No | `10000` | LRU cache entries for policy eval |
| `POLICY_CACHE_TTL` | No | `300s` | Cache entry TTL |
| `LOG_LEVEL` | No | `info` | Log level |

---

## Org Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `8071` | HTTP listen port |
| `GRPC_PORT` | No | `9071` | gRPC listen port |
| `DB_HOST` | Yes | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | Yes | — | PostgreSQL user |
| `DB_PASSWORD` | Yes | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `LOG_LEVEL` | No | `info` | Log level |

---

## Audit Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `8072` | HTTP listen port |
| `GRPC_PORT` | No | `9072` | gRPC listen port |
| `DB_HOST` | Yes | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | Yes | — | PostgreSQL user |
| `DB_PASSWORD` | Yes | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `NATS_URL` | Yes | — | NATS JetStream URL |
| `NATS_STREAM` | No | `GGID_EVENTS` | JetStream stream name |
| `AUDIT_RETENTION_DAYS` | No | `365` | Event retention period |
| `LOG_LEVEL` | No | `info` | Log level |

---

## Console (Admin UI)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `3000` | Next.js listen port |
| `NEXT_PUBLIC_GATEWAY_URL` | Yes | — | Gateway URL for API calls |
| `NEXT_PUBLIC_TENANT_ID` | Yes | — | Default tenant ID |

---

## Infrastructure

### PostgreSQL

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_USER` | Yes | — | Admin user |
| `POSTGRES_PASSWORD` | Yes | — | Admin password |
| `POSTGRES_DB` | No | `ggid` | Default database |

### Redis

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `REDIS_PASSWORD` | No | — | Auth password (recommended) |
| `REDIS_MAXMEMORY` | No | `256mb` | Max memory |
| `REDIS_MAXMEMORY_POLICY` | No | `allkeys-lru` | Eviction policy |

### NATS

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NATS_JETSTREAM_STORE_DIR` | No | `/data` | JetStream persistence dir |
| `NATS_MAX_FILE_STORE` | No | `10GB` | Max JetStream storage |

---

## Configuration Priority

GGID resolves configuration in this order (highest priority first):

1. **Environment variables** (production)
2. **`.env` file** (Docker Compose)
3. **Config file** (`ggid.yaml`, if present)
4. **Built-in defaults** (lowest)

---

## Docker Compose `.env` Example

```bash
# deploy/.env

# Database
POSTGRES_USER=ggid
POSTGRES_PASSWORD=change-me-in-production
POSTGRES_DB=ggid

# Redis
REDIS_PASSWORD=change-me-redis

# Auth Service
JWT_PRIVATE_KEY=/run/secrets/jwt-private.pem
JWT_PUBLIC_KEY=/run/secrets/jwt-public.pem
BCRYPT_COST=12
LOCKOUT_THRESHOLD=5

# OAuth
PKCE_REQUIRED=true

# LDAP (optional)
LDAP_URL=ldap://openldap:389
LDAP_BIND_DN=cn=admin,dc=example,dc=com
LDAP_BIND_PASSWORD=change-me-ldap
LDAP_BASE_DN=dc=example,dc=com
LDAP_USER_FILTER=(uid=%s)
LDAP_START_TLS=true

# NATS
NATS_URL=nats://nats:4222

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## References

- [Getting Started](./getting-started.md) — 5-minute quickstart
- [Deployment Guide](./deployment-guide.md) — Production deployment
- [Security Hardening](./security-hardening.md) — Production security checklist
