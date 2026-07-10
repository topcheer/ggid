# Access Key Management for IAM Systems

> Research document for GGID — API Access Key Architecture, Security, and Implementation
> Status: Research / Design
> Date: 2025-07-11

## Table of Contents

1. [Access Key Architecture](#1-access-key-architecture)
2. [Key Hashing and Storage](#2-key-hashing-and-storage)
3. [Scoped Keys](#3-scoped-keys)
4. [Key Rotation](#4-key-rotation)
5. [IP Binding](#5-ip-binding)
6. [Rate Limiting Per Key](#6-rate-limiting-per-key)
7. [Key Lifecycle Management](#7-key-lifecycle-management)
8. [Key Authentication Flow](#8-key-authentication-flow)
9. [Audit Trail for API Keys](#9-audit-trail-for-api-keys)
10. [GGID API Key Gap Analysis](#10-ggid-api-key-gap-analysis)
11. [Gap Analysis & Recommendations](#11-gap-analysis--recommendations)

---

## 1. Access Key Architecture

### Long-Lived API Keys vs Short-Lived OAuth Tokens

OAuth 2.0 access tokens are short-lived (minutes to hours) and are refreshed
via refresh tokens or client credentials grants. This model is ideal for
interactive applications and delegated access. However, many machine-to-machine
(M2M) scenarios benefit from long-lived API keys:

| Use Case | Token Type | Rationale |
|---|---|---|
| Web SPA accessing APIs | OAuth JWT (15 min) | Interactive, user-bound, refreshable |
| Mobile app | OAuth JWT + refresh | User session lifecycle |
| CI/CD pipeline | API key (long-lived) | No human to re-auth; static credential |
| Legacy system integration | API key | No OAuth client support; simpler HTTP header |
| Cron job / scheduled task | API key | Fires unattended; needs always-on credential |
| Internal service mesh | mTLS or OAuth client creds | Depends on service capability |

API keys trade the security of automatic expiry for operational simplicity. The
mitigation for this risk is a layered defense: key hashing at rest, scoped
permissions, IP binding, rate limiting, rotation policies, and comprehensive
audit trails.

### Key Format Design

A well-designed key format serves three purposes: human readability (quick
identification), service routing (prefix identifies the issuer), and integrity
verification (checksum catches transcription errors).

**Format:** `ggid_<base64url-body>_<checksum>`

- **Prefix** (`ggid_`): Identifies the key as belonging to GGID. Enables secret
  scanners (e.g., GitHub secret scanning, TruffleHog) to detect leaked keys.
  Also enables multi-tenant gateways to route keys to the correct issuer.
- **Body**: 32 bytes of cryptographic randomness, base64url-encoded (43 chars).
  This provides 256 bits of entropy — far beyond brute-force feasibility.
- **Checksum**: Last 6 characters are a CRC32 of the body, used purely as a
  transcription-error guard. Not a security mechanism — it prevents typos from
  hitting the database, not attacks.

```
ggid_Aa1Bb2Cc3Dd4Ee5Ff6Gg7Hh8Ii9Jj0Kk1Ll2Mm3Nn4_abc123
^^^^^                                               ^^^^^^
prefix                                               checksum
      <------------- 43-char body ---------------->
```

### Go Code: Key Generation

```go
package apikey

import (
	"crypto/rand"
	"encoding/base64"
	"hash/crc32"
)

const (
	keyPrefix = "ggid_"
	bodyBytes = 32 // 256 bits of entropy
)

// GenerateKey creates a new API key in the format:
//   ggid_<base64url-body>_<crc32-checksum>
//
// The caller MUST store the hash (see HashKey) and never retain the plaintext
// after initial display to the user.
func GenerateKey() (string, error) {
	raw := make([]byte, bodyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	checksum := crc32.ChecksumIEEE([]byte(body))
	return keyPrefix + body + "_" + encodeChecksum(checksum), nil
}

func encodeChecksum(c uint32) string {
	// 6-char base32-ish encoding for compactness
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := [6]byte{}
	for i := 5; i >= 0; i-- {
		b[i] = chars[c%36]
		c /= 36
	}
	return string(b[:])
}

// ValidateFormat checks that a key has the correct structural format
// (prefix, body, checksum) without looking it up in the database.
func ValidateFormat(key string) bool {
	if len(key) < len(keyPrefix)+44+7 {
		return false // too short
	}
	if key[:len(keyPrefix)] != keyPrefix {
		return false
	}
	rest := key[len(keyPrefix):]
	// Split at last underscore
	idx := -1
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == '_' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	body := rest[:idx]
	checksumStr := rest[idx+1:]

	// Verify checksum
	expected := encodeChecksum(crc32.ChecksumIEEE([]byte(body)))
	return checksumStr == expected
}
```

### Service Routing via Prefix

In a multi-tenant or multi-product deployment, the key prefix can encode the
service that issued it. For example:

- `ggid_` — GGID core IAM service
- `ggpay_` — Payment service
- `gghost_` — Hosting service

The gateway inspects the prefix before performing a database lookup, routing
the validation to the correct service's key store.

---

## 2. Key Hashing and Storage

### Never Store Plaintext

API keys are bearer tokens — anyone who possesses the key string can use it.
If the database is compromised and plaintext keys are stored, the attacker
gains immediate access to all systems those keys authenticate.

The correct approach is identical to password hashing: store only a one-way
hash. When a key is presented for validation, hash it and compare against
stored hashes.

### SHA-256 with Lookup Prefix

Unlike passwords, API keys have high entropy (256 bits), so a fast hash
(SHA-256) is acceptable — rainbow tables and brute force are infeasible.
Argon2id is unnecessary overhead for 256-bit inputs.

**Lookup prefix** solves the database index problem. If you store only the full
SHA-256 hash, you must scan the entire table to find a match (or use the hash
itself as the primary key). The lookup prefix — the first 8 characters of the
hex-encoded hash — narrows the search dramatically while revealing nothing
about the full key.

```
Full hash:   3f7a8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a
Lookup:      3f7a8b2c  (first 8 hex chars = 32 bits → narrows to ~1 in 4 billion)
```

### Go Code: Key Hashing and Storage

```go
package apikey

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// HashKey returns the full SHA-256 hex hash and the 8-char lookup prefix.
func HashKey(plaintext string) (fullHash string, lookupPrefix string) {
	h := sha256.Sum256([]byte(plaintext))
	full := hex.EncodeToString(h[:])
	return full, full[:8]
}

// VerifyKeyConstantTime compares a computed hash against a stored hash in
// constant time to prevent timing-based enumeration attacks.
func VerifyKeyConstantTime(computed, stored string) bool {
	if len(computed) != len(stored) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(stored)) == 1
}
```

### Database Schema

```sql
CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    user_id         UUID,                          -- NULL for service accounts
    name            TEXT NOT NULL,                  -- human-readable label
    key_prefix      TEXT NOT NULL,                  -- first 4 chars of body for display ("Aa1B")
    lookup_prefix   CHAR(8) NOT NULL,               -- first 8 hex chars of hash
    key_hash        TEXT NOT NULL,                  -- full SHA-256 hex hash
    scopes          TEXT[] NOT NULL DEFAULT '{}',   -- e.g., {"users:read","roles:write"}
    ip_allowlist    TEXT[] NOT NULL DEFAULT '{}',   -- CIDR strings, e.g., {"10.0.0.0/8"}
    rate_limit_rps  INT NOT NULL DEFAULT 100,       -- requests per second
    rate_limit_burst INT NOT NULL DEFAULT 200,      -- burst capacity
    status          TEXT NOT NULL DEFAULT 'active', -- active|suspended|rotated|revoked
    expires_at      TIMESTAMPTZ,                    -- NULL = no expiry
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_from    UUID REFERENCES api_keys(id),   -- predecessor key if rotated
    replaced_by     UUID REFERENCES api_keys(id),   -- successor key after rotation
    grace_until     TIMESTAMPTZ                     -- old key valid until this time during rotation
);

CREATE INDEX idx_api_keys_lookup ON api_keys (lookup_prefix) WHERE status IN ('active', 'rotated');
CREATE INDEX idx_api_keys_tenant ON api_keys (tenant_id);
CREATE INDEX idx_api_keys_user ON api_keys (user_id);
```

The partial index on `lookup_prefix WHERE status IN ('active', 'rotated')`
ensures fast lookups only for keys that could be valid, keeping the index small.

### Lookup Flow

1. Extract key from request.
2. Compute `HashKey(plaintext)` → `(fullHash, lookupPrefix)`.
3. `SELECT * FROM api_keys WHERE lookup_prefix = $1 AND status IN ('active', 'rotated')`.
4. Iterate results (typically 0-1 row), call `VerifyKeyConstantTime(fullHash, row.key_hash)`.
5. If match, check expiry, IP allowlist, and scopes.

---

## 3. Scoped Keys

### Binding Keys to Permissions

An API key without scopes is an all-or-nothing credential — it inherits the
full permissions of its owner. This violates the principle of least privilege.
Scoped keys restrict what a key can do, following the same model as OAuth scopes.

**Scope format:** `<resource>:<action>` (e.g., `users:read`, `roles:write`,
`audit:read`). A wildcard scope `*` grants full access (use sparingly).

| Key Purpose | Recommended Scopes |
|---|---|
| CI/CD read-only deploy | `users:read`, `roles:read` |
| Audit log exporter | `audit:read` |
| User provisioning bot | `users:read`, `users:write` |
| Monitoring health check | `health:read` |
| Full admin (emergency) | `*` |

### Scope Enforcement on Key-Authenticated Requests

The gateway middleware extracts scopes from the validated key and injects them
into the request context. Downstream middleware or handlers check scopes before
processing. GGID's existing `HasScope()` function already implements this
pattern with deny-by-default semantics.

### Go Code: Scoped Key Validation Middleware

```go
package middleware

import (
	"context"
	"net/http"
)

// ScopeKey is the context key for API key scopes.
type ScopeKey struct{}

// RequireScope returns middleware that rejects requests lacking the given scope.
// Must be placed AFTER API key / JWT authentication middleware.
func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, ok := r.Context().Value(ScopeKey{}).([]string)
			if !ok || !containsScope(scopes, scope) {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"error": "insufficient scope: requires " + scope,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyScope rejects requests that lack ALL of the listed scopes.
func RequireAnyScope(required ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, ok := r.Context().Value(ScopeKey{}).([]string)
			if !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"error": "no scopes in context",
				})
				return
			}
			for _, req := range required {
				if containsScope(scopes, req) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "insufficient scope",
			})
		})
	}
}

func containsScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target || s == "*" {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}
```

**Route wiring example:**

```go
mux.Handle("/api/v1/users", RequireScope("users:read")(apiKeyAuth(usersHandler)))
mux.Handle("/api/v1/users", RequireScope("users:write")(apiKeyAuth(createUserHandler)))
mux.Handle("/api/v1/audit", RequireScope("audit:read")(apiKeyAuth(auditHandler)))
```

---

## 4. Key Rotation

### Why Rotate?

| Trigger | Urgency | Description |
|---|---|---|
| Scheduled (90 days) | Low | Regular rotation as defense-in-depth |
| Personnel departure | Medium | Developer with key access leaves the team |
| Suspected compromise | Critical | Key may have been leaked (e.g., committed to Git) |
| Security incident | Critical | Breach detected; rotate all keys as containment |
| Policy compliance | Low | SOC 2 / PCI-DSS mandate periodic rotation |

### Dual-Key Overlap Strategy

The fundamental challenge of key rotation is zero downtime. If you revoke the
old key before the consumer updates their configuration, requests fail. The
solution is a **grace period** where both old and new keys are valid:

```
Timeline:
  T0        T1              T2                 T3
  |---------|---------------|-------------------|
  Create    Old key still   Old key revoked     Grace expires
  new key   works (grace)   (hard stop)
            New key also
            works
```

1. **T0**: Generate new key. Mark old key as `rotated` with `grace_until = T2`.
2. **T0–T2**: Both keys are valid. Consumer updates their config at any point.
3. **T2**: Old key stops working. Log a warning if still in use.
4. **T3**: Old key record is eligible for deletion (retained for audit).

### Go Code: Rotation Handler

```go
package apikey

import (
	"context"
	"time"
)

// RotationService handles API key rotation with a dual-key grace period.
type RotationService struct {
	store    KeyStore
	notifier RotationNotifier
}

type KeyStore interface {
	GetByID(ctx context.Context, id string) (*APIKey, error)
	Update(ctx context.Context, key *APIKey) error
	Create(ctx context.Context, key *APIKey) error
}

type RotationNotifier interface {
	NotifyKeyRotation(ctx context.Context, oldKey, newKey *APIKey) error
}

// RotateKey creates a new key, marks the old key as rotated with a grace
// period, and returns the new plaintext key. The caller is responsible for
// securely delivering the new key to the consumer.
func (rs *RotationService) RotateKey(
	ctx context.Context,
	oldKeyID string,
	gracePeriod time.Duration,
) (string, error) {
	old, err := rs.store.GetByID(ctx, oldKeyID)
	if err != nil {
		return "", err
	}
	if old.Status != "active" {
		return "", ErrKeyNotActive
	}

	// Generate new key
	plaintext, err := GenerateKey()
	if err != nil {
		return "", err
	}
	fullHash, lookup := HashKey(plaintext)

	now := time.Now()
	newKey := &APIKey{
		TenantID:      old.TenantID,
		UserID:        old.UserID,
		Name:          old.Name + " (rotated)",
		LookupPrefix:  lookup,
		KeyHash:       fullHash,
		Scopes:        old.Scopes,
		IPAllowlist:   old.IPAllowlist,
		RateLimitRPS:  old.RateLimitRPS,
		Status:        "active",
		ExpiresAt:     old.ExpiresAt,
		CreatedAt:     now,
		RotatedFrom:   &old.ID,
	}
	if err := rs.store.Create(ctx, newKey); err != nil {
		return "", err
	}

	// Mark old key as rotated with grace period
	old.Status = "rotated"
	old.ReplacedBy = &newKey.ID
	old.GraceUntil = now.Add(gracePeriod)
	if err := rs.store.Update(ctx, old); err != nil {
		// Best-effort: new key is already active, old key will expire naturally
		_ = err
	}

	// Fire notification (webhook, email, Slack) asynchronously
	go rs.notifier.NotifyKeyRotation(ctx, old, newKey)

	return plaintext, nil
}

// EmergencyRotate performs immediate rotation with a minimal grace period
// (or zero grace). Used when a key is known to be compromised.
func (rs *RotationService) EmergencyRotate(ctx context.Context, oldKeyID string) (string, error) {
	// 1-hour grace for emergency: enough time to update critical consumers
	// but short enough to limit attacker window
	return rs.RotateKey(ctx, oldKeyID, 1*time.Hour)
}
```

### Automated Rotation Reminders

A background job checks for keys approaching their rotation deadline:

```go
// CheckRotationDue returns keys that should be rotated within the next 7 days.
func (rs *RotationService) CheckRotationDue(ctx context.Context, tenantID string) ([]*APIKey, error) {
	threshold := time.Now().AddDate(0, 0, 7)
	return rs.store.ListByExpiry(ctx, tenantID, threshold)
}
```

---

## 5. IP Binding

### Restricting Keys to Known Sources

IP binding constrains a key to only work from specific IP addresses or CIDR
ranges. This is one of the most effective controls for keys used in fixed
infrastructure (CI/CD servers, on-prem integrations, cloud NAT gateways).

| Scenario | Binding | Example |
|---|---|---|
| GitHub Actions CI | AWS NAT gateway IP | `203.0.113.45/32` |
| Corporate VPN | Office subnet | `10.0.0.0/8` |
| Cloud function | Cloud provider range | `35.230.0.0/16` |
| Unrestricted (trusted) | None | — |

### CIDR Matching

GGID already implements CIDR matching in its `IPAllowlist` middleware
(`services/gateway/internal/middleware/ipallowlist.go`). The logic parses CIDR
strings into `net.IPNet` structures and uses `net.IPNet.Contains()` for matching.

The existing implementation handles `X-Forwarded-For` and `X-Real-IP` headers
to extract the real client IP behind load balancers, falling back to
`RemoteAddr`. This is critical for correct IP enforcement in production.

### Go Code: IP-Restricted Key Validation

```go
package apikey

import (
	"net"
	"net/http"
	"strings"
)

// ValidateIPBinding checks that the request's client IP is within the key's
// allowed CIDR ranges. If the key has no IP allowlist, all IPs are permitted.
func ValidateIPBinding(r *http.Request, allowedCIDRs []string) bool {
	if len(allowedCIDRs) == 0 {
		return true // no restriction
	}

	clientIP := extractClientIP(r)
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, cidrStr := range allowedCIDRs {
		// Normalize single IPs to /32 or /128
		if !strings.Contains(cidrStr, "/") {
			if ip.To4() != nil {
				cidrStr += "/32"
			} else {
				cidrStr += "/128"
			}
		}
		_, ipNet, err := net.ParseCIDR(cidrStr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// extractClientIP extracts the real client IP, respecting proxy headers.
// This mirrors the logic in services/gateway/internal/middleware/ipallowlist.go.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Strip port from RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
```

### IPv6 Considerations

IPv6 addresses are 128 bits, so single-IP bindings should use `/128` instead of
`/32`. The `ParseCIDRs` function in GGID's `ipallowlist.go` already handles
this correctly by checking `ip.To4()` to determine the address family.

---

## 6. Rate Limiting Per Key

### Per-Key vs Per-Tenant Limits

GGID currently implements per-tenant rate limiting via `TenantBucketLimiter`
and per-tier limits via `tier_ratelimit.go`. Per-key rate limiting adds another
dimension: individual keys within a tenant can have distinct quotas.

This is important when a tenant has multiple integrations with different
criticality levels:

| Key | Limit | Burst | Purpose |
|---|---|---|---|
| `prod-cicd` | 100 req/s | 200 | High-volume deployment pipeline |
| `audit-export` | 5 req/s | 10 | Nightly export job |
| `monitoring` | 2 req/s | 5 | Health check poller |
| `legacy-sync` | 50 req/s | 100 | Legacy system data sync |

### Go Code: Per-Key Rate Limiter with Redis

```go
package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements a sliding-window rate limiter backed by Redis.
// Each key gets its own counter namespace, enabling per-key quotas.
type RedisRateLimiter struct {
	rdb *redis.Client
}

func NewRedisRateLimiter(rdb *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{rdb: rdb}
}

// Allow checks whether the given API key ID is within its rate limit.
// Returns (allowed, retryAfter, error).
func (rl *RedisRateLimiter) Allow(
	ctx context.Context,
	keyID string,
	rps int,
	burst int,
) (bool, time.Duration, error) {
	// Lua script: atomic token bucket check
	// KEYS[1] = counter key
	// KEYS[2] = timestamp key
	// ARGV[1] = current time (microseconds)
	// ARGV[2] = refill rate (tokens per microsecond)
	// ARGV[3] = burst capacity
	const script = `
		local now = tonumber(ARGV[1])
		local rate = tonumber(ARGV[2])
		local burst = tonumber(ARGV[3])
		local counter = redis.call('GET', KEYS[1])
		local last_time = redis.call('GET', KEYS[2])

		local tokens = burst
		if counter and last_time then
			local elapsed = now - tonumber(last_time)
			tokens = math.min(burst, tonumber(counter) + elapsed * rate)
		end

		if tokens >= 1 then
			tokens = tokens - 1
			redis.call('SET', KEYS[1], tokens)
			redis.call('SET', KEYS[2], now)
			redis.call('EXPIRE', KEYS[1], 60)
			redis.call('EXPIRE', KEYS[2], 60)
			return {1, tostring(tokens)}
		else
			local retry = (1 - tokens) / rate
			return {0, tostring(retry)}
		end
	`

	counterKey := fmt.Sprintf("ggid:rl:%s:count", keyID)
	timeKey := fmt.Sprintf("ggid:rl:%s:ts", keyID)
	now := time.Now().UnixMicro()
	ratePerMicro := float64(rps) / 1e6

	result, err := rl.rdb.Eval(ctx, script, []string{counterKey, timeKey},
		now, ratePerMicro, burst).Result()
	if err != nil {
		// Fail-open: allow request if Redis is down (log alert separately)
		return true, 0, nil
	}

	vals, ok := result.([]any)
	if !ok || len(vals) < 1 {
		return true, 0, nil
	}

	allowed, _ := vals[0].(int64)
	if allowed == 1 {
		return true, 0, nil
	}

	// Calculate retry duration
	retryMicro, _ := vals[1].(string)
	retry := time.Duration(0)
	if retryMicro != "" {
		var micro int64
		fmt.Sscanf(retryMicro, "%d", &micro)
		retry = time.Duration(micro) * time.Microsecond
	}
	return false, retry, nil
}

// QuotaTracker tracks daily and monthly usage quotas per key.
type QuotaTracker struct {
	rdb *redis.Client
}

// IncrementUsage bumps the daily and monthly counters for a key.
func (qt *QuotaTracker) IncrementUsage(ctx context.Context, keyID string) error {
	now := time.Now().UTC()
	dailyKey := fmt.Sprintf("ggid:quota:daily:%s:%s", keyID, now.Format("2006-01-02"))
	monthlyKey := fmt.Sprintf("ggid:quota:monthly:%s:%s", keyID, now.Format("2006-01"))

	pipe := qt.rdb.Pipeline()
	pipe.Incr(ctx, dailyKey)
	pipe.Expire(ctx, dailyKey, 48*time.Hour)
	pipe.Incr(ctx, monthlyKey)
	pipe.Expire(ctx, monthlyKey, 35*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}
```

---

## 7. Key Lifecycle Management

### Key States

```
                    ┌──────────┐
   Create ─────────►│  active  │
                    └────┬─────┘
                         │
              ┌──────────┼──────────┐
              ▼          ▼          ▼
        ┌──────────┐ ┌────────┐ ┌──────────┐
        │ suspended│ │rotated │ │ revoked  │
        └────┬─────┘ └───┬────┘ └────┬─────┘
             │           │           │
             └─────┬─────┘           │
                   ▼                 ▼
             ┌──────────┐      ┌──────────┐
             │  active  │      │ deleted  │
             │(resume)  │      │(purged)  │
             └──────────┘      └──────────┘
```

| State | Validates? | Description |
|---|---|---|
| `active` | Yes | Normal operation |
| `suspended` | No | Temporarily disabled (admin action, billing) |
| `rotated` | Yes (grace) | Superseded by new key; valid during grace period |
| `revoked` | No | Permanently disabled (compromise, user request) |
| `deleted` | No | Record purged after retention period (90 days) |

### Expiry Enforcement

Keys should have explicit expiry dates. The gateway checks expiry on every
key-authenticated request. A background job marks expired keys as `revoked`
and notifies the owner.

**Recommended expiry policy:**

| Key Type | Default Expiry | Max Expiry |
|---|---|---|
| Production | 90 days | 365 days |
| Staging | 30 days | 90 days |
| Development | 7 days | 30 days |
| Emergency | 24 hours | 7 days |

### Go Code: Key Lifecycle Manager

```go
package apikey

import (
	"context"
	"log"
	"time"
)

// LifecycleManager handles key state transitions and automated expiry.
type LifecycleManager struct {
	store    KeyStore
	notifier LifecycleNotifier
	tickRate time.Duration
}

type LifecycleNotifier interface {
	NotifyExpiryWarning(ctx context.Context, key *APIKey, daysUntilExpiry int) error
	NotifyExpired(ctx context.Context, key *APIKey) error
}

// APIKey represents the persisted key record (no plaintext).
type APIKey struct {
	ID           string
	TenantID     string
	UserID       string
	Name         string
	LookupPrefix string
	KeyHash      string
	Scopes       []string
	IPAllowlist  []string
	RateLimitRPS int
	Status       string
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	RotatedFrom  *string
	ReplacedBy   *string
	GraceUntil   *time.Time
	LastUsedAt   *time.Time
}

// RunExpiryJob is a background goroutine that checks for keys approaching
// or past their expiry date and transitions them appropriately.
func (lm *LifecycleManager) RunExpiryJob(ctx context.Context) {
	ticker := time.NewTicker(lm.tickRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lm.processExpiry(ctx)
		}
	}
}

func (lm *LifecycleManager) processExpiry(ctx context.Context) {
	now := time.Now()

	// Warn 7 days before expiry
	warnThreshold := now.AddDate(0, 0, 7)
	warnKeys, err := lm.store.ListExpiringBefore(ctx, warnThreshold)
	if err != nil {
		log.Printf("lifecycle: failed to list expiring keys: %v", err)
		return
	}

	for _, key := range warnKeys {
		days := int(key.ExpiresAt.Sub(now).Hours() / 24)
		if days <= 0 {
			// Key has expired — transition to revoked
			key.Status = "revoked"
			if err := lm.store.Update(ctx, key); err != nil {
				log.Printf("lifecycle: failed to revoke expired key %s: %v", key.ID, err)
			}
			_ = lm.notifier.NotifyExpired(ctx, key)
		} else {
			// Key expiring soon — send warning
			_ = lm.notifier.NotifyExpiryWarning(ctx, key, days)
		}
	}

	// Purge deleted keys past retention
	retentionCutoff := now.AddDate(0, 0, -90)
	if err := lm.store.PurgeBefore(ctx, retentionCutoff); err != nil {
		log.Printf("lifecycle: purge failed: %v", err)
	}
}

// Create activates a new key. Returns the plaintext key (shown once).
func (lm *LifecycleManager) Create(
	ctx context.Context,
	tenantID, userID, name string,
	scopes, ipAllowlist []string,
	expiresAt *time.Time,
) (string, *APIKey, error) {
	plaintext, err := GenerateKey()
	if err != nil {
		return "", nil, err
	}
	fullHash, lookup := HashKey(plaintext)

	key := &APIKey{
		TenantID:     tenantID,
		UserID:       userID,
		Name:         name,
		LookupPrefix: lookup,
		KeyHash:      fullHash,
		Scopes:       scopes,
		IPAllowlist:  ipAllowlist,
		Status:       "active",
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
	}

	if err := lm.store.Create(ctx, key); err != nil {
		return "", nil, err
	}
	return plaintext, key, nil
}

// Revoke permanently disables a key. Cannot be undone.
func (lm *LifecycleManager) Revoke(ctx context.Context, keyID string) error {
	key, err := lm.store.GetByID(ctx, keyID)
	if err != nil {
		return err
	}
	key.Status = "revoked"
	return lm.store.Update(ctx, key)
}

// Suspend temporarily disables a key. Can be resumed.
func (lm *LifecycleManager) Suspend(ctx context.Context, keyID string) error {
	key, err := lm.store.GetByID(ctx, keyID)
	if err != nil {
		return err
	}
	key.Status = "suspended"
	return lm.store.Update(ctx, key)
}
```

---

## 8. Key Authentication Flow

### Gateway Request Flow

When a request arrives at the gateway with an API key, the following sequence
executes:

```
Client Request
     │
     ▼
┌─────────────────────┐
│ Extract API Key     │  Header: X-API-Key: ggid_xxx
│ from request        │  OR query: ?api_key=ggid_xxx
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Validate Format     │  Check prefix, body length, checksum
│ (fast reject)       │  Reject → 401 (format error)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Hash & Lookup       │  SHA-256(key) → lookup_prefix
│ in database         │  SELECT WHERE lookup_prefix = ?
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Verify Hash         │  Constant-time compare with stored hash
│ (constant-time)     │  Reject → 401 (invalid key)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Check Status        │  active or rotated (within grace)?
│ & Expiry            │  Reject → 401 (expired/revoked)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Validate Scope      │  Required scope present in key scopes?
│                     │  Reject → 403 (insufficient scope)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Check IP Binding    │  Client IP in allowed CIDRs?
│                     │  Reject → 403 (IP not allowed)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Check Rate Limit    │  Within per-key quota?
│ (Redis)             │  Reject → 429 (too many requests)
└────────┬────────────┘
         │
         ▼
┌─────────────────────┐
│ Inject Identity     │  ctx.tenant_id, ctx.user_id, ctx.scopes
│ & Forward           │  → Proxy to backend service
└─────────────────────┘
```

### Go Code: Complete API Key Authentication Middleware

```go
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// KeyValidator is the interface for persistent API key validation.
type KeyValidator interface {
	Lookup(ctx context.Context, key string) (*KeyInfo, error)
}

// KeyInfo contains the resolved key metadata for request processing.
type KeyInfo struct {
	KeyID        string
	TenantID     string
	UserID       string
	Scopes       []string
	IPAllowlist  []string
	RateLimitRPS int
	IsRotated    bool
}

// RateLimitChecker checks whether a key is within its rate limit.
type RateLimitChecker interface {
	Allow(ctx context.Context, keyID string, rps, burst int) (bool, time.Duration, error)
}

// APIKeyMiddleware provides full API key authentication with scope, IP,
// and rate limit enforcement. Falls through to JWT if no API key present.
func APIKeyMiddleware(
	validator KeyValidator,
	rateLimiter RateLimitChecker,
	auditLogger AuditLogger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Step 1: Extract key
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}
			if apiKey == "" {
				// No API key — fall through to JWT auth
				next.ServeHTTP(w, r)
				return
			}

			// Step 2: Lookup & verify hash
			info, err := validator.Lookup(r.Context(), apiKey)
			if err != nil {
				rejectAPIKey(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			// Step 3: IP binding check
			if !ValidateIPBinding(r, info.IPAllowlist) {
				rejectAPIKey(w, http.StatusForbidden, "IP address not allowed for this key")
				return
			}

			// Step 4: Rate limit check
			allowed, retry, err := rateLimiter.Allow(r.Context(), info.KeyID, info.RateLimitRPS, info.RateLimitRPS*2)
			if err == nil && !allowed {
				w.Header().Set("Retry-After", formatRetry(retry))
				rejectAPIKey(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			// Step 5: Inject identity into context
			ctx := context.WithValue(r.Context(), TenantIDKey, info.TenantID)
			ctx = context.WithValue(ctx, UserIDKey, info.UserID)
			ctx = context.WithValue(ctx, ScopeKey{}, info.Scopes)
			ctx = context.WithValue(ctx, APIKeyIDKey, info.KeyID)

			// Step 6: Audit (async to avoid blocking request)
			if info.IsRotated {
				w.Header().Set("X-API-Key-Rotation-Warning",
					"This key has been rotated. Update to the new key.")
			}
			go auditLogger.LogKeyUsage(context.Background(), info.KeyID, r)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var APIKeyIDKey apiKeyIDCtx = "api_key_id"
type apiKeyIDCtx string

func rejectAPIKey(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func formatRetry(d time.Duration) string {
	return time.Now().Add(d).Format(time.RFC1123)
}
```

---

## 9. Audit Trail for API Keys

### What to Log

Every API key-authenticated request should produce an audit record. GGID's
existing audit infrastructure (NATS JetStream publisher, audit query API)
provides the transport; this section defines the key-specific events.

**Per-request events:**

| Field | Example | Purpose |
|---|---|---|
| `key_id` | `a3f8c1d2-...` | Identify which key made the request |
| `key_name` | `prod-cicd-deploy` | Human-readable label |
| `tenant_id` | `00000000-...-001` | Tenant context |
| `endpoint` | `POST /api/v1/users` | What was accessed |
| `client_ip` | `203.0.113.45` | Source IP |
| `status_code` | `200` | Response status |
| `latency_ms` | `42` | Performance metric |
| `timestamp` | `2025-07-11T10:30:00Z` | When |

**Key lifecycle events:**

| Event | When | Severity |
|---|---|---|
| `key.created` | New key generated | INFO |
| `key.rotated` | Key replaced with new key | INFO |
| `key.revoked` | Key permanently disabled | WARNING |
| `key.expired` | Key reached expiry date | WARNING |
| `key.suspended` | Temporarily disabled | WARNING |
| `key.deleted` | Record purged | INFO |
| `key.anomaly.new_ip` | Key used from unseen IP | CRITICAL |
| `key.anomaly.volume_spike` | Usage exceeds baseline | WARNING |
| `key.anomaly.off_hours` | Usage outside expected window | INFO |

### Anomaly Detection

GGID's auth service already has an `anomaly_detection.go` module. The same
principles apply to API key usage:

1. **New IP detection**: Maintain a Redis set of IPs seen per key. When a
   request arrives from an IP not in the set, log a `key.anomaly.new_ip` event.
   After the first sighting, add the IP to the set.

2. **Volume spike**: Track rolling 5-minute request counts per key. If the
   current count exceeds 3x the rolling 24-hour average, flag as anomaly.

3. **Off-hours usage**: For keys with a known usage schedule (e.g., CI/CD that
   runs 9-5), flag requests outside that window.

### Go Code: Key Usage Audit

```go
package apikey

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// KeyUsageAudit logs API key usage and performs lightweight anomaly detection.
type KeyUsageAudit struct {
	seenIPs *sync.Map  // key: keyID, value: map[string]bool (IP set)
	stats   *UsageStats
}

// UsageEntry captures a single key usage event.
type UsageEntry struct {
	KeyID      string    `json:"key_id"`
	TenantID   string    `json:"tenant_id"`
	Endpoint   string    `json:"endpoint"`
	Method     string    `json:"method"`
	StatusCode int       `json:"status_code"`
	ClientIP   string    `json:"client_ip"`
	LatencyMS  int64     `json:"latency_ms"`
	Timestamp  time.Time `json:"timestamp"`
	Anomaly    string    `json:"anomaly,omitempty"` // "new_ip", "volume_spike", etc.
}

// LogKeyUsage records a key usage event. Called asynchronously after the
// request is forwarded.
func (a *KeyUsageAudit) LogKeyUsage(ctx context.Context, keyID, tenantID string, r *http.Request) {
	ip := extractClientIP(r)

	// Anomaly detection: new IP
	anomaly := a.detectNewIP(keyID, ip)

	entry := UsageEntry{
		KeyID:     keyID,
		TenantID:  tenantID,
		Endpoint:  r.URL.Path,
		Method:    r.Method,
		ClientIP:  ip,
		Timestamp: time.Now().UTC(),
		Anomaly:   anomaly,
	}

	// Publish to audit pipeline (NATS, Kafka, etc.)
	_ = a.stats.Record(ctx, entry)
}

// detectNewIP checks whether the client IP has been seen for this key before.
// Returns an anomaly label if this is a new IP.
func (a *KeyUsageAudit) detectNewIP(keyID, ip string) string {
	ipSetIface, _ := a.seenIPs.LoadOrStore(keyID, &sync.Map{})
	ipSet := ipSetIface.(*sync.Map)

	if _, loaded := ipSet.LoadOrStore(ip, true); loaded {
		return "" // IP already known
	}
	return "new_ip"
}

// UsageStats tracks per-key statistics for dashboards and anomaly detection.
type UsageStats struct {
	mu      sync.Mutex
	window  map[string]*keyStats // key: keyID
}

type keyStats struct {
	totalRequests  int64
	errorCount     int64
	lastSeen       time.Time
	topEndpoints   map[string]int64
}

// GetKeyStats returns aggregated usage statistics for a key.
func (s *UsageStats) GetKeyStats(keyID string) *KeyStatsReport {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats, ok := s.window[keyID]
	if !ok {
		return nil
	}

	report := &KeyStatsReport{
		TotalRequests: stats.totalRequests,
		ErrorCount:    stats.errorCount,
		ErrorRate:     float64(stats.errorCount) / float64(stats.totalRequests),
		LastSeen:      stats.lastSeen,
	}
	return report
}

type KeyStatsReport struct {
	TotalRequests int64     `json:"total_requests"`
	ErrorCount    int64     `json:"error_count"`
	ErrorRate     float64   `json:"error_rate"`
	LastSeen      time.Time `json:"last_seen"`
}
```

---

## 10. GGID API Key Gap Analysis

### What GGID Already Has

The GGID codebase already contains significant API key infrastructure in the
gateway middleware layer:

**`services/gateway/internal/middleware/apikey.go`:**
- `APIKeyAuth` middleware that extracts keys from `X-API-Key` header or
  `api_key` query parameter
- `APIKeyValidator` interface for pluggable validation backends
- `MemoryAPIKeyValidator` for testing (in-memory key store)
- `HasScope()` function with deny-by-default semantics (P0 security fix)
- Pass-through to JWT auth when no API key is present

**`services/gateway/internal/middleware/apikey_rotation.go`:**
- `RotatableAPIKeyValidator` with dual-key grace period support
- `RotateKey()` method that marks old key as rotated and registers replacement
- `IsRotated()` and `ReplacementKey()` query helpers
- Configurable grace period (default 7 days)

**`services/gateway/internal/middleware/ipallowlist.go`:**
- Per-tenant CIDR matching with `X-Forwarded-For` / `X-Real-IP` support
- `ParseCIDRs()` handles both IPv4 (`/32`) and IPv6 (`/128`) single IPs
- Per-tenant IP restriction enforcement

**`pkg/crypto/crypto.go`:**
- `GenerateRandomToken(byteLen)` using `crypto/rand` (base64url-encoded)
- `constantTimeCompare()` for timing-safe comparison
- `HashPassword()` / `VerifyPassword()` using Argon2id with optional pepper
- AES-256-GCM encryption for secrets at rest

**`services/auth/internal/service/token_service.go`:**
- Refresh token hashing pattern: `hashToken()` uses SHA-256 hex encoding
- Redis-backed token caching with TTL (`rt:{hash}` → token ID)
- Token rotation with replay detection (revokes entire session on replay)
- This exact pattern should be replicated for API key persistence

### What's Missing

| Gap | Current State | Impact | Effort |
|---|---|---|---|
| **No `api_keys` table** | Keys stored in-memory only (`MemoryAPIKeyValidator`) | Keys lost on restart; no multi-instance support | Medium |
| **No key generation** | No `ggid_` prefix or checksum | Keys lack identifiable format and typo protection | Small |
| **No hashed storage** | Plaintext keys stored in validator map | Database compromise exposes all keys | Medium |
| **No key management API** | No REST endpoints for CRUD operations | Cannot create/list/revoke keys programmatically | Medium |
| **No per-key rate limiting** | Only per-tenant (`TenantBucketLimiter`) | One key can consume entire tenant quota | Medium |
| **No per-key usage tracking** | No audit trail for key usage | Cannot detect anomalies or report on usage | Medium |
| **No lifecycle management** | No expiry enforcement or automated rotation | Keys persist indefinitely until manual revocation | Medium |
| **Rotation is in-memory** | `RotatableAPIKeyValidator` doesn't persist | Rotation lost on restart | Small |
| **No IP binding per key** | `IPAllowlist` is per-tenant, not per-key | Cannot restrict individual keys to IPs | Small |
| **No key display format** | Generated keys shown raw, no structured format | Poor UX; no key identification | Small |

### Design for Adding API Key Support

The implementation should follow the existing token service pattern
(`token_service.go`) which already demonstrates hash-based storage with Redis
caching and PostgreSQL persistence.

**Phase 1: Core Infrastructure**
1. Create `api_keys` table migration (following the schema in Section 2)
2. Implement key generation with `ggid_` prefix and checksum
3. Implement hash-based storage with lookup prefix
4. Replace `MemoryAPIKeyValidator` with `DatabaseAPIKeyValidator`

**Phase 2: Management API**
5. Add REST endpoints: `POST /api/v1/api-keys`, `GET /api/v1/api-keys`,
   `DELETE /api/v1/api-keys/{id}`, `POST /api/v1/api-keys/{id}/rotate`
6. Wire API key auth middleware to use `DatabaseAPIKeyValidator`

**Phase 3: Advanced Features**
7. Add per-key rate limiting (Redis token bucket per key)
8. Add per-key IP binding (extend `IPAllowlist` to key level)
9. Add usage tracking and audit trail
10. Add lifecycle manager with automated expiry

---

## 11. Gap Analysis & Recommendations

### Summary of Current State

GGID has a **solid foundation** for API key authentication in its gateway
middleware layer. The `APIKeyAuth` middleware, `RotatableAPIKeyValidator`,
`IPAllowlist`, and `HasScope` functions cover the core authentication and
authorization flow. However, the implementation is **in-memory only** — it lacks
persistence, lifecycle management, and operational tooling.

The most critical gap is the absence of a persistent key store. Without a
database table and hashed storage, keys are lost on restart and cannot be
shared across gateway instances. This makes the current implementation
unsuitable for production.

### Action Items

| # | Action | Effort | Priority | Description |
|---|---|---|---|---|
| 1 | **Create `api_keys` table and `DatabaseAPIKeyValidator`** | 3 days | P0 | Migration + repository + validator implementing `APIKeyValidator` interface. Follows `token_service.go` pattern: SHA-256 hash storage with Redis cache. This unblocks everything else. |
| 2 | **Implement key generation with `ggid_` prefix** | 0.5 days | P0 | Add `GenerateKey()` to a new `pkg/apikey` package. Wire into management API. Plumb `ValidateFormat()` into middleware for fast-reject. |
| 3 | **Build key management REST API** | 2 days | P1 | CRUD endpoints under `/api/v1/api-keys`. Include scope assignment, IP binding configuration, expiry setting. Return plaintext key only on creation. |
| 4 | **Add per-key rate limiting and usage tracking** | 2 days | P1 | Redis token bucket per key (Section 6). Usage audit to NATS (Section 9). Dashboard in admin console. |
| 5 | **Implement lifecycle manager with automated expiry** | 1.5 days | P2 | Background goroutine for expiry enforcement. Rotation reminders via email/webhook. Integrate with existing `RotatableAPIKeyValidator` for persisted rotation. |

**Total estimated effort: ~9 developer-days**

### Risk Assessment

- **Low risk**: The `APIKeyValidator` interface already exists, so swapping
  the in-memory validator for a database-backed one requires no changes to
  the gateway middleware chain.

- **Medium risk**: Per-key rate limiting adds a Redis round-trip to every
  key-authenticated request. This should be benchmarked; if latency is a
  concern, the rate limit check can be pipelined with the hash lookup.

- **Zero risk to existing JWT auth**: The `APIKeyAuth` middleware falls
  through to JWT when no API key is present, so adding persistent key support
  does not affect existing token-based authentication flows.

### Alignment with Existing Patterns

The proposed implementation closely mirrors patterns already established in GGID:

1. **Hash storage**: `token_service.go` already uses `hashToken()` (SHA-256 hex)
   for refresh tokens. API keys should use the identical approach.
2. **Redis caching**: Refresh tokens use `rt:{hash}` → token ID in Redis with
   TTL. API keys should use `apikey:{hash}` → key ID.
3. **Rotation**: `RotatableAPIKeyValidator` already implements the dual-key
   grace period. The work is to persist this state, not redesign the logic.
4. **Scope enforcement**: `HasScope()` already checks both API key scopes and
   JWT scopes with deny-by-default. No changes needed.
5. **IP enforcement**: `IPAllowlist` already does per-tenant CIDR matching.
   Extending to per-key CIDRs is a straightforward generalization.

### Conclusion

GGID's API key infrastructure is approximately 40% complete. The gateway
middleware layer is well-designed with proper interfaces and security-first
patterns (deny-by-default scope checking, constant-time comparison, IP
allowlists). The primary gap is persistence — once the `api_keys` table and
`DatabaseAPIKeyValidator` are implemented, the existing middleware chain
handles everything else with minimal changes. The estimated 9 developer-days
of work would bring GGID to production-ready API key management with scoped
keys, per-key rate limiting, IP binding, lifecycle automation, and full audit
trail.
