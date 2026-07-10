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

## Feature Flags

GGID supports runtime feature flags for gradual rollouts and A/B testing.

### Available Feature Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `FEATURE_WEBAUTHN` | bool | `true` | Enable WebAuthn/passkey registration and login |
| `FEATURE_MFA_TOTP` | bool | `true` | Enable TOTP-based MFA |
| `FEATURE_LDAP` | bool | `false` | Enable LDAP authentication |
| `FEATURE_SAML` | bool | `false` | Enable SAML 2.0 SSO |
| `FEATURE_SOCIAL_LOGIN` | bool | `false` | Enable social login connectors |
| `FEATURE_SCIM` | bool | `true` | Enable SCIM 2.0 provisioning API |
| `FEATURE_SSE_AUDIT` | bool | `true` | Enable SSE audit event streaming |
| `FEATURE_PASSWORD_HISTORY` | bool | `true` | Enforce password history on change |
| `FEATURE_BREACH_DETECTION` | bool | `false` | Check passwords against HIBP API |
| `FEATURE_STEPUP_AUTH` | bool | `true` | Enable step-up authentication |
| `FEATURE_ANOMALY_DETECTION` | bool | `false` | ML-based login anomaly scoring |
| `FEATURE_CUSTOM_CLAIMS` | bool | `true` | Allow custom JWT claims via hooks |

### Configure Feature Flags

Feature flags can be set via environment variables or managed at runtime via
the admin API:

```bash
# Via environment variable
FEATURE_WEBAUTHN=true
FEATURE_SAML=false

# Via admin API (runtime toggle)
curl -X PUT $API/api/v1/settings/features \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "webauthn": true,
    "mfa_totp": true,
    "saml": true,
    "social_login": false
  }'

# Get current feature flags
curl $API/api/v1/settings/features \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Runtime changes take effect immediately without service restart.

---

## Security Defaults

GGID ships with secure defaults. Override only when you understand the
implications.

### Authentication Security

| Setting | Default | Description |
|---------|---------|-------------|
| `JWT_ACCESS_TOKEN_TTL` | `15m` | Access token lifetime (short-lived) |
| `JWT_REFRESH_TOKEN_TTL` | `168h` (7d) | Refresh token lifetime |
| `JWT_SIGNING_ALG` | `RS256` | JWT signing algorithm |
| `BCRYPT_COST` | `12` | bcrypt cost factor (higher = slower) |
| `SESSION_MAX_CONCURRENT` | `5` | Max active sessions per user |
| `SESSION_IDLE_TIMEOUT` | `30m` | Idle session timeout |
| `ACCOUNT_LOCKOUT_THRESHOLD` | `5` | Failed attempts before lockout |
| `ACCOUNT_LOCKOUT_DURATION` | `15m` | Lockout period |
| `PASSWORD_MIN_LENGTH` | `12` | Minimum password length |
| `PASSWORD_REQUIRE_UPPER` | `true` | Require uppercase letter |
| `PASSWORD_REQUIRE_LOWER` | `true` | Require lowercase letter |
| `PASSWORD_REQUIRE_DIGIT` | `true` | Require at least one digit |
| `PASSWORD_REQUIRE_SPECIAL` | `true` | Require special character |
| `PASSWORD_HISTORY_COUNT` | `5` | Reject reuse of last N passwords |
| `PASSWORD_MAX_AGE` | `90d` | Password expiry (set 0 to disable) |

### Rate Limiting Defaults

| Setting | Default | Description |
|---------|---------|-------------|
| `RATE_LIMIT_AUTH` | `10/min` | Auth endpoints (login, register) |
| `RATE_LIMIT_API` | `60/min` | General API endpoints |
| `RATE_LIMIT_POLICY_CHECK` | `100/min` | Policy check endpoint |
| `RATE_LIMIT_SCIM` | `100/min` | SCIM provisioning endpoints |
| `RATE_LIMIT_REFRESH` | `30/min` | Token refresh endpoint |
| `RATE_LIMIT_BURST` | `1.5x` | Burst multiplier above steady rate |
| `RATE_LIMIT_FAIL_MODE` | `open` | Behavior when Redis is unavailable |

### TLS/Network

| Setting | Default | Description |
|---------|---------|-------------|
| `TLS_MIN_VERSION` | `1.2` | Minimum TLS version |
| `CORS_ALLOW_CREDENTIALS` | `true` | Allow cookies in CORS |
| `CORS_MAX_AGE` | `24h` | Preflight cache duration |
| `COOKIE_SECURE` | `true` (prod) | Set Secure flag on cookies |
| `COOKIE_HTTP_ONLY` | `true` | Set HttpOnly flag |
| `COOKIE_SAME_SITE` | `strict` | SameSite attribute |
| `HSTS_MAX_AGE` | `31536000` | HSTS header max age (1 year) |
| `HSTS_INCLUDE_SUBDOMAINS` | `true` | Include subdomains in HSTS |
| `HSTS_PRELOAD` | `true` | Enable HSTS preload |

### Audit

| Setting | Default | Description |
|---------|---------|-------------|
| `AUDIT_ENABLED` | `true` | Enable audit event collection |
| `AUDIT_RETENTION_DAYS` | `90` | Hot retention period |
| `AUDIT_PII_REDACTION` | `true` | Redact PII in audit logs |
| `AUDIT_NATS_STREAM` | `GGID_EVENTS` | JetStream stream name |
| `AUDIT_NATS_RETENTION` | `7d` | JetStream message retention |

---

## Multi-Tenancy Settings

### Tenant Isolation

GGID enforces tenant isolation at three layers:

1. **Application layer** — Every query includes `tenant_id` from JWT context
2. **Database layer** — PostgreSQL Row-Level Security (RLS) policies
3. **Network layer** — Tenant ID in `X-Tenant-ID` header, validated by Gateway

| Setting | Default | Description |
|---------|---------|-------------|
| `MULTI_TENANT_MODE` | `true` | Enable multi-tenancy (set false for single-tenant) |
| `DEFAULT_TENANT_ID` | `00000000-0000-0000-0000-000000000001` | Default tenant UUID |
| `TENANT_HEADER` | `X-Tenant-ID` | Header name for tenant identification |
| `TENANT_ISOLATION_STRICT` | `true` | Reject requests without valid tenant header |
| `TENANT_AUTO_CREATE` | `false` | Auto-create tenant on first request |

### Tenant Management API

```bash
# List tenants
curl $API/api/v1/tenants \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN"

# Create tenant
curl -X POST $API/api/v1/tenants \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
  -d '{
    "name": "Acme Corp",
    "tier": "pro",
    "features": {
      "webauthn": true,
      "saml": true
    }
  }'

# Configure tenant-specific rate limits
curl -X PUT $API/api/v1/tenants/$TENANT_ID \
  -H "Authorization: Bearer $SUPERADMIN_TOKEN" \
  -d '{
    "rate_limits": {
      "auth_endpoints": "20/min",
      "crud_endpoints": "60/min"
    }
  }'
```

### Single-Tenant Mode

For deployments that don't need multi-tenancy:

```bash
MULTI_TENANT_MODE=false
DEFAULT_TENANT_ID=00000000-0000-0000-0000-000000000001
```

In single-tenant mode:
- The `X-Tenant-ID` header becomes optional
- RLS still applies but with a fixed tenant_id
- All data belongs to the default tenant

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
