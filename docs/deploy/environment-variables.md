# Environment Variables Reference

> Complete reference of ALL environment variables across all 7 GGID services.

---

## Gateway Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | Gateway HTTP listen port |
| `JWT_SECRET` | **Yes** | — | HMAC-SHA256 signing key (shared with auth) |
| `REDIS_URL` | **Yes** | — | Redis connection string |
| `DATABASE_URL` | **Yes** | — | PostgreSQL connection string |
| `NATS_URL` | No | `nats://localhost:4222` | NATS connection string |
| `JWKS_REFRESH_INTERVAL` | No | `15m` | JWKS cache refresh interval |
| `LOG_LEVEL` | No | `info` | Log level: debug, info, warn, error |
| `TENANT_HEADER` | No | `X-Tenant-ID` | Header name for tenant context |
| `RATE_LIMIT_ENABLED` | No | `true` | Enable rate limiting |
| `RATE_LIMIT_RPS` | No | `100` | Requests per second per IP |
| `RATE_LIMIT_BURST` | No | `200` | Burst capacity |
| `CIRCUIT_BREAKER_ENABLED` | No | `true` | Enable circuit breaker |
| `CB_FAILURE_THRESHOLD` | No | `5` | Failures before opening circuit |
| `CB_RESET_TIMEOUT` | No | `30s` | Time before half-open probe |
| `CB_MAX_REQUESTS` | No | `1` | Max requests in half-open state |
| `COMPRESSION_ENABLED` | No | `true` | Enable gzip/brotli compression |
| `CORS_ALLOWED_ORIGINS` | No | `*` | Comma-separated allowed origins |
| `BODY_LIMIT` | No | `10MB` | Max request body size |
| `TIMEOUT_READ` | No | `5s` | Read timeout |
| `TIMEOUT_WRITE` | No | `30s` | Write timeout |
| `TIMEOUT_IDLE` | No | `120s` | Idle timeout |

---

## Auth Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `9001` | Auth HTTP listen port |
| `JWT_SECRET` | **Yes** | — | JWT signing key (MUST be non-empty, log.Fatal if empty) |
| `DATABASE_URL` | **Yes** | — | PostgreSQL connection string |
| `REDIS_URL` | **Yes** | — | Redis connection string |
| `JWT_ACCESS_TTL` | No | `15m` | Access token TTL |
| `JWT_REFRESH_TTL` | No | `168h` | Refresh token TTL (7 days) |
| `JWT_ISSUER` | No | `ggid-auth` | JWT issuer claim |
| `JWT_AUDIENCE` | No | `ggid-gateway` | JWT audience claim |
| `PASSWORD_MIN_LENGTH` | No | `8` | Minimum password length |
| `PASSWORD_BCRYPT_COST` | No | `12` | bcrypt cost factor |
| `PASSWORD_PEPPER` | No | — | Server-side password pepper |
| `MFA_ISSUER` | No | `GGID` | TOTP issuer name |
| `LDAP_URL` | No | — | LDAP server URL (empty = disabled) |
| `LDAP_BIND_DN` | No | — | LDAP service account DN |
| `LDAP_BIND_PASSWORD` | No | — | LDAP service account password |
| `LDAP_BASE_DN` | No | — | LDAP search base DN |
| `LDAP_USER_FILTER` | No | `(uid=%s)` | LDAP user search filter |
| `LDAP_START_TLS` | No | `false` | Enable STARTTLS for LDAP |
| `LDAP_AUTO_PROVISION` | No | `false` | Auto-create local user on LDAP login |
| `LOG_LEVEL` | No | `info` | Log level |

---

## OAuth Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `9005` | OAuth HTTP listen port |
| `DATABASE_URL` | **Yes** | — | PostgreSQL connection string |
| `REDIS_URL` | **Yes** | — | Redis connection string |
| `JWT_SECRET` | **Yes** | — | JWT signing key (shared) |
| `OAUTH_ISSUER` | No | `http://localhost:9005` | OAuth issuer URL |
| `OAUTH_CODE_TTL` | No | `10m` | Authorization code TTL |
| `OAUTH_ACCESS_TOKEN_TTL` | No | `15m` | Access token TTL |
| `OAUTH_REFRESH_TOKEN_TTL` | No | `168h` | Refresh token TTL |
| `OAUTH_PKCE_REQUIRED` | No | `true` | Require PKCE (OAuth 2.1) |
| `DPoP_ENABLED` | No | `false` | Enable DPoP (RFC 9449) |
| `LOG_LEVEL` | No | `info` | Log level |

---

## Identity Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP listen port |
| `GRPC_PORT` | No | `50051` | gRPC listen port |
| `DATABASE_URL` | **Yes** | — | PostgreSQL connection string |
| `REDIS_URL` | No | — | Redis connection string |
| `SCIM_ENABLED` | No | `true` | Enable SCIM 2.0 endpoints |
| `SCIM_BULK_MAX_SIZE` | No | `100` | Max records per bulk operation |
| `LOG_LEVEL` | No | `info` | Log level |

---

## Policy Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `8070` | HTTP listen port |
| `GRPC_PORT` | No | `9070` | gRPC listen port |
| `DB_HOST` | **Yes** | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | **Yes** | — | PostgreSQL username |
| `DB_PASSWORD` | **Yes** | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `DB_SSLMODE` | No | `disable` | SSL mode |
| `LOG_LEVEL` | No | `info` | Log level |

> **Note**: Policy service uses individual `DB_*` vars, NOT `DATABASE_URL`.

---

## Org Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `8071` | HTTP listen port |
| `GRPC_PORT` | No | `9071` | gRPC listen port |
| `DB_HOST` | **Yes** | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | **Yes** | — | PostgreSQL username |
| `DB_PASSWORD` | **Yes** | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `DB_SSLMODE` | No | `disable` | SSL mode |
| `LOG_LEVEL` | No | `info` | Log level |

> **Note**: Org service uses individual `DB_*` vars, NOT `DATABASE_URL`.

---

## Audit Service

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HTTP_PORT` | No | `8072` | HTTP listen port |
| `GRPC_PORT` | No | `9072` | gRPC listen port |
| `DB_HOST` | **Yes** | — | PostgreSQL host |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | **Yes** | — | PostgreSQL username |
| `DB_PASSWORD` | **Yes** | — | PostgreSQL password |
| `DB_NAME` | No | `ggid` | Database name |
| `DB_SSLMODE` | No | `disable` | SSL mode |
| `NATS_URL` | **Yes** | — | NATS connection string |
| `AUDIT_RETENTION_DAYS` | No | `90` | Days to retain audit events |
| `LOG_LEVEL` | No | `info` | Log level |

> **Note**: Audit service uses individual `DB_*` vars, NOT `DATABASE_URL`.

---

## Console

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `3000` | Next.js listen port |
| `NEXT_PUBLIC_GGID_URL` | **Yes** | — | Gateway URL for API calls |
| `NEXT_PUBLIC_GGID_WS_URL` | No | — | WebSocket URL for real-time updates |
| `NODE_ENV` | No | `production` | Environment |

---

## Shared Configuration Notes

### JWT_SECRET

- **MUST** be the same across gateway, auth, and oauth services
- **MUST** be non-empty (auth service calls `log.Fatal` if empty)
- Recommended: 32+ character random string
- Generate: `openssl rand -base64 32`

### DATABASE_URL Format

```
postgres://USER:PASSWORD@HOST:PORT/DBNAME?sslmode=disable
```

- Used by: Gateway, Auth, OAuth, Identity
- NOT used by: Policy, Org, Audit (they use individual `DB_*` vars)

### Default Tenant

```
TENANT_ID=00000000-0000-0000-0000-000000000001
```

All requests must include `X-Tenant-ID` header with this UUID.

---

*Last updated: 2025-07-11*