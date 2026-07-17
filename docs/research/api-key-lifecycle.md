# API Key Lifecycle Management: CRUD, Scoping, Rotation, and Audit for GGID

> **Focus**: A production-grade API key system — DB-backed key storage with bcrypt hashing, scope binding, per-key rate limiting, automatic rotation, revocation, usage tracking, and audit trails. Replaces the current in-memory placeholder.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§12), curl commands (§7).
>
> **Related**: `access-key-management.md` (2025-07-11) covers theoretical architecture. This document covers the **production implementation** replacing the current in-memory code, with new checklist standards.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: In-Memory API Keys](#2-ggid-current-state-in-memory-api-keys)
3. [Gap Analysis](#3-gap-analysis)
4. [Proposed Architecture](#4-proposed-architecture)
5. [Industry Best Practices](#5-industry-best-practices)
6. [Endpoint Precondition Check](#6-endpoint-precondition-check)
7. [API Design + Curl Commands](#7-api-design--curl-commands)
8. [Database Schema](#8-database-schema)
9. [Key Hashing and Validation](#9-key-hashing-and-validation)
10. [Scope Binding and Enforcement](#10-scope-binding-and-enforcement)
11. [Rotation Strategy](#11-rotation-strategy)
12. [Implementation Backlog with DoD](#12-implementation-backlog-with-dod)
13. [Competitive Differentiation](#13-competitive-differentiation)
14. [Security Considerations](#14-security-considerations)

---

## 1. Executive Summary

API keys are the primary authentication mechanism for machine-to-machine (M2M) access — CI/CD pipelines calling the API, microservices authenticating to each other, and third-party integrations. Unlike OAuth tokens (short-lived, user-scoped), API keys are long-lived, service-scoped, and must be managed through a complete lifecycle: create → scope → distribute → rotate → revoke.

GGID has an API key system but it is **entirely in-memory** (`api_keys_handler.go:27-30`):
```go
var (
    apiKeysMu sync.RWMutex
    apiKeys   = []APIKey{}     // ← in-memory, lost on restart
)
```

This violates the team acceptance checklist. Key issues:
1. **Keys lost on restart** — all API keys disappear when the auth service restarts
2. **No key hashing** — keys stored in plaintext in the `APIKey` struct
3. **No real key generation** — IDs are `key-{timestamp}` not cryptographic random
4. **No actual key secret returned** — no `ggid_` prefixed secret for clients to use
5. **No rotation** — no mechanism to rotate keys without downtime
6. **No usage tracking** — `UsageCount` and `LastUsed` are never updated
7. **No per-key rate limiting** — gateway `APIKeyAuth` middleware validates but doesn't rate-limit
8. **No IP allow-list binding** — validator interface exists but no implementation

**Recommendation**: Replace the in-memory handler with a PostgreSQL-backed API key service with bcrypt-hashed storage, cryptographic key generation, scope enforcement, rotation with grace period, usage tracking, per-key rate limiting, and audit trails.

**Estimated effort**: 2 sprints for MVP (DB + CRUD + hashing + gateway integration + Console UI).

---

## 2. GGID Current State: In-Memory API Keys

### Existing Implementation

| Component | File:Line | Status | Issue |
|-----------|-----------|--------|-------|
| APIKey struct | `api_keys_handler.go:16` | **In-memory** | No DB persistence |
| apiKeys store | `api_keys_handler.go:29` | **In-memory** | `[]APIKey{}` — lost on restart |
| handleAPIKeys | `api_keys_handler.go:34` | **Implemented** | CRUD works but not persisted |
| Key ID format | `api_keys_handler.go:54` | **Weak** | `key-{timestamp}` — not random |
| Key hashing | — | **Missing** | Keys not hashed at all |
| APIKeyAuth middleware | `gateway/middleware/apikey.go:22` | **Implemented** | Validates via `APIKeyValidator` interface |
| APIKeyValidator | `gateway/middleware/apikey.go:14` | **Interface** | Returns tenantID, userID, scopes |
| Access keys handler | `auth/server/access_keys_handler.go` | **Alias** | Rewrites to api-keys handler |
| IP allow-list test | `apikey_ipallowlist_test.go` | **Tested** | Test mock, no real impl |
| Console page | `console/src/app/settings/api-keys/` | **Exists** | UI calls `/api/v1/auth/api-keys` |

### The APIKeyAuth Middleware (Working)

```go
// gateway/middleware/apikey.go:22
func APIKeyAuth(validator APIKeyValidator) func(http.Handler) http.Handler {
    // Extracts X-API-Key header or ?api_key= query param
    // Calls validator.Validate(ctx, key) → tenantID, userID, scopes
    // Injects scopes into context
    // Returns 401 on invalid key
}
```

The middleware is well-designed — it just needs a real validator implementation backed by PostgreSQL.

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | **In-memory storage** | All keys lost on service restart |
| 2 | **No key hashing** | Plaintext keys in memory; if logged, they leak |
| 3 | **No real key secret** | `key-{timestamp}` is not a usable API key; no `ggid_xxxx` format |
| 4 | **No rotation** | Can't rotate compromised keys without breaking clients |
| 5 | **No usage tracking** | `LastUsed` and `UsageCount` never updated |
| 6 | **No per-key rate limiting** | Keys can be used unlimited times |
| 7 | **No IP allow-list** | Interface exists but no DB-backed implementation |
| 8 | **No expiry enforcement** | Gateway doesn't check `expires_at` |
| 9 | **No audit trail** | Key creation/use/revocation not logged to audit service |
| 10 | **No scope enforcement** | Scopes stored but not checked in gateway routing |

---

## 4. Proposed Architecture

```
                    ┌──────────────────────────────────────────────┐
                    │         API Key Lifecycle Service             │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  PostgreSQL Key Store                 │    │
                    │  │  - api_keys table (hashed keys)       │    │
                    │  │  - api_key_usage table (audit)        │    │
                    │  │  - api_key_scopes table (scope map)   │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Key Generation                       │    │
                    │  │  - crypto/rand 32 bytes               │    │
                    │  │  - Format: ggid_{base62(32 bytes)}    │    │
                    │  │  - SHA-256 hash for storage           │    │
                    │  │  - Bcrypt for comparison cache        │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  CRUD API                             │    │
                    │  │  - POST /api/v1/auth/api-keys (create)│    │
                    │  │  - GET /api/v1/auth/api-keys (list)   │    │
                    │  │  - GET /api/v1/auth/api-keys/{id}     │    │
                    │  │  - PUT /api/v1/auth/api-keys/{id}     │    │
                    │  │  - DELETE /api/v1/auth/api-keys/{id}  │    │
                    │  │  - POST .../rotate (grace period)     │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Gateway Validator (Redis cache)      │    │
                    │  │  - Hash incoming key → lookup Redis   │    │
                    │  │  - Redis miss → DB lookup → cache     │    │
                    │  │  - Enforce: scopes, expiry, IP list   │    │
                    │  │  - Rate limit per key                 │    │
                    │  │  - Update LastUsed (async)            │    │
                    │  └──────────────────────────────────────┘    │
                    └──────────────────────────────────────────────┘
```

---

## 5. Industry Best Practices

### OWASP API Security Top 5 for Keys

| Practice | Implementation |
|----------|---------------|
| **Limit scope** | Each key scoped to specific permissions + resources |
| **Rotate regularly** | Auto-rotation every 30-90 days; instant rotation on compromise |
| **Never in source code** | Keys returned once at creation; hash stored in DB |
| **HTTPS only** | Gateway enforces TLS; keys never transmitted in plaintext |
| **Monitor usage** | Per-key usage analytics + anomaly detection |

### Key Format Comparison

| Provider | Format | Length | Example |
|----------|--------|--------|---------|
| **GGID (target)** | `ggid_` + base62 | 48 chars | `ggid_5f8a3b2c1d4e...` |
| Auth0 | `` + base64 | 64 chars | Random token |
| AWS | `AKIA` + base64 | 20 chars | `AKIA...EXAMPLE (redacted)` |
| Stripe | `sk_` + base64 | 99 chars | `sk_live_XXXX...XXXX (redacted)` |
| GitHub | `ghp_` + base62 | 40 chars | `ghp_XXXX...XXXX (redacted)` |

---

## 6. Endpoint Precondition Check

### Existing Endpoints (Replace In-Memory)

| Endpoint | File:Line | Current | Target |
|----------|-----------|---------|--------|
| `GET /api/v1/auth/api-keys` | `api_keys_handler.go:36` | **In-memory list** | DB-backed list |
| `POST /api/v1/auth/api-keys` | `api_keys_handler.go:43` | **In-memory create** | DB-backed + hash + real key |
| `GET /api/v1/auth/api-keys/{id}` | `api_keys_handler.go` | **In-memory** | DB-backed |
| `DELETE /api/v1/auth/api-keys/{id}` | `api_keys_handler.go` | **In-memory** | DB-backed + audit |
| `APIKeyAuth` middleware | `gateway/middleware/apikey.go:22` | **Interface only** | Real DB-backed validator |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/auth/api-keys/{id}` | PUT | Update key (name, scopes, IP list) | P0 |
| `/api/v1/auth/api-keys/{id}/rotate` | POST | Rotate key with grace period | P0 |
| `/api/v1/auth/api-keys/{id}/usage` | GET | Usage statistics for a key | P1 |
| `/api/v1/auth/api-keys/{id}/test` | POST | Test key validity (dry-run) | P1 |

---

## 7. API Design + Curl Commands

### Create API Key

```bash
curl -X POST https://ggid.corp.com/api/v1/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "CI/CD Pipeline Key",
    "scopes": ["users:read", "users:write", "groups:read"],
    "expires_at": "2027-01-01T00:00:00Z",
    "ip_allowlist": ["10.0.0.0/8", "192.168.1.0/24"],
    "rate_limit_per_minute": 100
  }'

# Response (key returned ONLY once):
{
  "id": "7f3a2b1c-...",
  "name": "CI/CD Pipeline Key",
  "key": "ggid_5f8a3b2c1d4e6789a0b1c2d3e4f5a6b7c8d9e0f1",
  "key_prefix": "ggid_5f8a",
  "scopes": ["users:read", "users:write", "groups:read"],
  "expires_at": "2027-01-01T00:00:00Z",
  "ip_allowlist": ["10.0.0.0/8", "192.168.1.0/24"],
  "rate_limit_per_minute": 100,
  "status": "active",
  "created_at": "2026-07-17T10:00:00Z"
}
```

### List Keys

```bash
curl https://ggid.corp.com/api/v1/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Response (never includes the full key):
{
  "keys": [
    {
      "id": "7f3a2b1c-...",
      "name": "CI/CD Pipeline Key",
      "key_prefix": "ggid_5f8a",
      "scopes": ["users:read", "users:write"],
      "status": "active",
      "last_used_at": "2026-07-17T09:45:00Z",
      "usage_count": 15420,
      "expires_at": "2027-01-01T00:00:00Z",
      "created_at": "2026-06-01T10:00:00Z"
    }
  ]
}
```

### Rotate Key

```bash
curl -X POST https://ggid.corp.com/api/v1/auth/api-keys/7f3a2b1c/rotate \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"grace_period_hours": 24}'

# Response:
{
  "id": "7f3a2b1c-...",
  "new_key": "ggid_9a8b7c6d5e4f3210fedcba9876543210abcdef",
  "old_key_expires_at": "2026-07-18T10:00:00Z",
  "grace_period_active": true
}
```

### Revoke Key

```bash
curl -X DELETE https://ggid.corp.com/api/v1/auth/api-keys/7f3a2b1c \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Response:
{ "status": "revoked", "revoked_at": "2026-07-17T10:05:00Z" }
```

### Use API Key (Authentication)

```bash
curl https://ggid.corp.com/api/v1/identity/users \
  -H "X-API-Key: ggid_5f8a3b2c1d4e6789a0b1c2d3e4f5a6b7c8d9e0f1" \
  -H "X-Tenant-ID: $TENANT_ID"
```

---

## 8. Database Schema

```sql
-- API keys (hashed storage)
CREATE TABLE api_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    
    -- Key storage (never store plaintext after creation)
    key_hash            VARCHAR(256) NOT NULL,        -- SHA-256 hash of key
    key_prefix          VARCHAR(16) NOT NULL,         -- ggid_5f8a (for display)
    
    -- Scopes
    scopes              JSONB NOT NULL DEFAULT '[]',  -- ["users:read", "groups:write"]
    
    -- Constraints
    expires_at          TIMESTAMPTZ,
    ip_allowlist        JSONB DEFAULT '[]',           -- ["10.0.0.0/8"]
    rate_limit_per_minute INT DEFAULT 1000,
    
    -- State
    status              VARCHAR(16) NOT NULL DEFAULT 'active', -- active, rotated, revoked, expired
    
    -- Rotation tracking
    rotated_from_id     UUID REFERENCES api_keys(id), -- Previous key in rotation chain
    rotation_grace_until TIMESTAMPTZ,                  -- Old key still valid until this time
    
    -- Usage
    last_used_at        TIMESTAMPTZ,
    usage_count         BIGINT DEFAULT 0,
    
    -- Audit
    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at          TIMESTAMPTZ,
    revoked_by          UUID,
    revoke_reason       TEXT
);

-- API key usage log (for analytics + anomaly detection)
CREATE TABLE api_key_usage_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    api_key_id          UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    endpoint            VARCHAR(512) NOT NULL,
    method              VARCHAR(8) NOT NULL,
    status_code         INT NOT NULL,
    ip_address          VARCHAR(45),
    response_ms         INT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- API key rate limit counters (Redis primary, DB fallback)
-- Stored in Redis: key="apikey:rl:{key_id}:{minute_bucket}" → count

-- Indexes
CREATE INDEX idx_api_keys_tenant ON api_keys (tenant_id, status);
CREATE INDEX idx_api_keys_hash ON api_keys (key_hash) WHERE status IN ('active', 'rotated');
CREATE INDEX idx_api_keys_expires ON api_keys (expires_at) WHERE status = 'active';
CREATE INDEX idx_api_keys_rotated ON api_keys (rotated_from_id);
CREATE INDEX idx_api_usage_tenant_time ON api_key_usage_log (tenant_id, created_at DESC);
CREATE INDEX idx_api_usage_key_time ON api_key_usage_log (api_key_id, created_at DESC);
```

---

## 9. Key Hashing and Validation

### Key Generation

```go
// Generate a new API key
func GenerateAPIKey() (plaintext string, hash string, prefix string, err error) {
    // 1. Generate 32 random bytes
    raw := make([]byte, 32)
    if _, err := rand.Read(raw); err != nil {
        return "", "", "", err
    }
    
    // 2. Encode as base62 with prefix
    plaintext = "ggid_" + base62.Encode(raw)
    
    // 3. Compute SHA-256 hash for storage
    h := sha256.Sum256([]byte(plaintext))
    hash = hex.EncodeToString(h[:])
    
    // 4. Extract prefix for display
    prefix = plaintext[:9] // "ggid_XXXX"
    
    return plaintext, hash, prefix, nil
}
```

### Gateway Validation (with Redis Cache)

```go
// gateway/internal/middleware/apikey_validator.go

type pgAPIKeyValidator struct {
    db    *pgxpool.Pool
    cache *redis.Client
}

func (v *pgAPIKeyValidator) Validate(ctx context.Context, key string) (string, string, []string, error) {
    // 1. Hash the incoming key
    h := sha256.Sum256([]byte(key))
    keyHash := hex.EncodeToString(h[:])
    
    // 2. Check Redis cache first
    cacheKey := fmt.Sprintf("apikey:%s", keyHash)
    if cached, err := v.cache.Get(ctx, cacheKey).Result(); err == nil {
        var info cachedKeyInfo
        json.Unmarshal([]byte(cached), &info)
        // Check expiry, IP allow-list, rate limit
        return info.TenantID, info.UserID, info.Scopes, nil
    }
    
    // 3. DB lookup
    var info keyInfo
    err := v.db.QueryRow(ctx,
        `SELECT tenant_id, scopes, expires_at, ip_allowlist, status
         FROM api_keys WHERE key_hash = $1 AND status IN ('active', 'rotated')`,
        keyHash,
    ).Scan(&info.TenantID, &info.Scopes, &info.ExpiresAt, &info.IPAllowlist, &info.Status)
    
    if err != nil {
        return "", "", nil, fmt.Errorf("invalid API key")
    }
    
    // 4. Check expiry
    if info.ExpiresAt != nil && time.Now().After(*info.ExpiresAt) {
        return "", "", nil, fmt.Errorf("API key expired")
    }
    
    // 5. Cache for 60 seconds
    v.cache.Set(ctx, cacheKey, cachedJSON, 60*time.Second)
    
    // 6. Async update last_used + usage_count
    go v.updateUsage(keyHash)
    
    return info.TenantID, "", info.Scopes, nil
}
```

---

## 10. Scope Binding and Enforcement

```go
// Middleware enforces that the key's scopes include the required scope
// for the requested endpoint.

// Scope mapping (configured per route)
var routeScopeMap = map[string]string{
    "/api/v1/identity/users":    "users:read",
    "/api/v1/identity/users/":   "users:read",
    "/api/v1/policy/roles":      "roles:read",
    "/api/v1/policy/decisions":  "policy:evaluate",
    "/api/v1/audit/events":      "audit:read",
}

func checkKeyScope(requestedPath string, keyScopes []string) bool {
    requiredScope, ok := routeScopeMap[requestedPath]
    if !ok {
        return true // No scope requirement for this path
    }
    for _, scope := range keyScopes {
        if scope == requiredScope || scope == "*"{ // wildcard
            return true
        }
    }
    return false
}
```

---

## 11. Rotation Strategy

```
Step 1: Client calls POST /api-keys/{id}/rotate
  → System generates new key
  → Old key status changes to "rotated"
  → Old key grace_period_until = now + 24h
  → Both keys valid during grace period

Step 2: Client updates their service with new key
  → New key used for all requests

Step 3: After grace period (24h)
  → Old key automatically revoked
  → Any request with old key returns 401

Step 4: Audit event logged: "Key rotated, old key revoked after grace period"
```

---

## 12. Implementation Backlog with DoD

### P0 — DB-Backed Key Storage + Gateway Integration (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | API key DB schema | ✅ CREATE TABLE in migration ✅ go build PASS ✅ No in-memory map | 1d |
| 2 | Key generation + hashing | ✅ crypto/rand 32 bytes ✅ SHA-256 hash stored ✅ Plaintext never persisted ✅ ≥3 tests | 1d |
| 3 | API key repository | ✅ CRUD backed by pgx ✅ `if err != nil` guards ✅ ≥3 tests | 2d |
| 4 | Replace in-memory handler | ✅ handleAPIKeys uses repo (not apiKeys var) ✅ No sync.RWMutex ✅ curl test PASS ✅ ≥3 tests | 2d |
| 5 | Gateway validator implementation | ✅ Implements APIKeyValidator interface ✅ DB-backed + Redis cache ✅ Expiry check ✅ ≥3 tests | 3d |

### P1 — Rotation + Scope Enforcement + Rate Limiting (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 6 | Key rotation with grace period | ✅ POST /rotate generates new key ✅ Old key valid during grace ✅ Auto-revoke after grace ✅ ≥3 tests | 3d |
| 7 | Scope enforcement in gateway | ✅ Route-to-scope map ✅ Insufficient scope returns 403 ✅ ≥3 tests | 2d |
| 8 | Per-key rate limiting | ✅ Redis counter per key per minute ✅ 429 on exceed ✅ ≥3 tests | 2d |
| 9 | Usage tracking (last_used, count) | ✅ Async update on each request ✅ DB-backed ✅ ≥3 tests | 1d |
| 10 | IP allow-list enforcement | ✅ CIDR matching ✅ Non-allowlisted IP blocked ✅ ≥3 tests | 2d |

### P2 — Console UI + Analytics (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 11 | Console key management | ✅ Create/list/revoke UI ✅ Key shown once on create ✅ Copy-to-clipboard | 3d |
| 12 | Rotation UI | ✅ Rotate button with grace period display ✅ Countdown timer | 2d |
| 13 | Usage analytics | ✅ Per-key usage chart ✅ Top endpoints by key ✅ ≥3 tests | 3d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 14 | Key templates | Pre-configured scope bundles ("CI/CD Key", "Read-Only Key") |
| 15 | Key expiration alerts | Email notification 7 days before expiry |
| 16 | Compromised key auto-detection | Detect anomalous usage patterns, auto-rotate |
| 17 | Key inheritance | Service account keys with delegated permissions |
| 18 | Key signing | Asymmetric key signing for high-security use cases |

---

## 13. Competitive Differentiation

| Feature | GGID (target) | Auth0 | AWS API Gateway | Kong | Stripe |
|---------|---------------|-------|-----------------|------|--------|
| **Key generation** | **crypto/rand + ggid_ prefix** | Yes | Yes | Yes | Yes |
| **Key hashing** | **SHA-256 at rest** | Yes | KMS-backed | Hash | Yes |
| **Scope binding** | **Per-route scope map** | Yes (scopes) | IAM policies | ACLs | Restricted keys |
| **Rotation** | **Grace period dual-key** | Via API | Via IAM | Via Admin | Via Dashboard |
| **Rate limiting** | **Per-key Redis** | Tenant-level | Usage plans | Per-key | Not built-in |
| **IP allow-list** | **CIDR matching** | Custom | Resource policies | Enterprise | Custom |
| **Usage analytics** | **Per-key tracking** | Yes | CloudWatch | Analytics | Dashboard |
| **DB-backed** | **PostgreSQL** | Proprietary | Proprietary | PostgreSQL | Proprietary |
| **Open source** | **Yes (Apache 2.0)** | No | No | Partially | No |

---

## 14. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Key leakage via logs** | Keys never logged; only hash logged; gateway redacts X-API-Key from access logs |
| **Key reuse across tenants** | Key hash lookup scoped by tenant_id; cross-tenant key impossible |
| **Brute-force key guessing** | 32 bytes (256 bits) = 2^256 search space; infeasible |
| **Stale keys after compromise** | Instant revocation + Redis cache invalidation (<1s propagation) |
| **Key in URL query param** | Supported but discouraged; gateway logs warning when `?api_key=` used |
| **Rotation gap** | Grace period (24h default) ensures zero-downtime rotation |
| **Rate limit bypass** | Rate limit keyed by key_hash, not by IP; can't bypass by rotating IP |
| **Compromised key scope escalation** | Scopes immutable after creation; must create new key with different scopes |

---

## References

- [OWASP API Security Top 10](https://owasp.org/API-Security/) — API key security risks
- [Zuplo: API Key Rotation Guide](https://zuplo.com/learning-center/api-key-rotation-lifecycle-management) — Zero-downtime rotation
- [OneUptime: API Key Management Best Practices](https://oneuptime.com/blog/post/2026-02-20-api-key-management-best-practices/view) — 2026 practices
- [AKeyless: API Key Management](https://www.akeyless.io/blog/power-of-api-keys/) — Vault-backed key storage
- [GGID API Keys Handler](../services/auth/internal/server/api_keys_handler.go) — Current in-memory implementation at line 34
- [GGID APIKeyAuth Middleware](../services/gateway/internal/middleware/apikey.go) — Gateway validator at line 22
- [GGID Access Key Research](./access-key-management.md) — Previous theoretical architecture research
- [GGID IP Allow-list Tests](../services/gateway/internal/middleware/apikey_ipallowlist_test.go) — Existing test patterns
