# Idempotency for IAM API Operations

> **Research Document** — GGID IAM Suite
> Topic: Ensuring at-most-once semantics for identity, auth, and audit operations
> Status: Active

---

## Table of Contents

1. [Why Idempotency Matters for IAM](#1-why-idempotency-matters-for-iam)
2. [Idempotency-Key Header Pattern](#2-idempotency-key-header-pattern)
3. [Idempotency Storage](#3-idempotency-storage)
4. [Natural Idempotency Keys](#4-natural-idempotency-keys)
5. [Idempotency for NATS Events](#5-idempotency-for-nats-events)
6. [Idempotency for Token Issuance](#6-idempotency-for-token-issuance)
7. [Idempotency for Role/Policy Operations](#7-idempotency-for-rolepolicy-operations)
8. [Idempotency vs Distributed Locks](#8-idempotency-vs-distributed-locks)
9. [Idempotency Audit Trail](#9-idempotency-audit-trail)
10. [GGID Idempotency Gap Analysis](#10-ggid-idempotency-gap-analysis)
11. [Gap Analysis and Recommendations](#11-gap-analysis-and-recommendations)

---

## 1. Why Idempotency Matters for IAM

Identity and Access Management systems deal with state mutations that carry
security and compliance consequences. Unlike a blog comment where a duplicate
is merely annoying, a duplicate user creation or token issuance can create
security holes, compliance violations, and audit trail corruption.

### 1.1 The Retry Problem

Network communication is inherently unreliable. The classic double-submit
scenario:

```
Client                    Gateway                    Identity Service
  |                          |                            |
  |--- POST /users --------->|                            |
  |                          |--- CreateUser() ---------->|
  |                          |                            |-- INSERT user
  |     [timeout]            |<--- 201 Created -----------|
  |<-- (response lost) ------|                            |
  |                          |                            |
  |--- POST /users (retry) ->|                            |
  |                          |--- CreateUser() ---------->|
  |                          |                            |-- INSERT user (DUPLICATE!)
  |                          |<--- 201 Created -----------|
  |<-- 201 ------------------|                            |
```

The client cannot distinguish between "request never arrived" and "response
was lost." Without idempotency, the safe choice (retry) causes harm.

### 1.2 IAM-Specific Duplicate Risks

| Operation | Duplicate Impact | Severity |
|---|---|---|
| User registration | Two accounts with same email | Critical — security bypass |
| Role assignment | User gains role twice (usually harmless) | Low — but pollutes audit |
| Token issuance | Two valid tokens from one auth code | Critical — RFC 6749 violation |
| Audit event | Two records for one action | Medium — false-positive alerts |
| Policy creation | Two identical policies active | Medium — double-deny/double-allow |
| Webhook delivery | Downstream system processes event twice | High — double-charge, double-provision |

### 1.3 The Problem in Go (Without Idempotency)

```go
// POST /users handler — NO idempotency protection
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var input CreateUserInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    // Check for existing user
    existing, _ := h.repo.GetByEmail(ctx, input.Email)
    if existing != nil {
        writeJSON(w, http.StatusConflict, errorResp("email exists"))
        return
    }

    // RACE CONDITION: between GetByEmail and CreateUser,
    // another request can insert the same email.
    user, err := h.service.CreateUser(ctx, &input)
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, errorResp(err.Error()))
        return
    }

    // On client retry (timeout + retry), this handler runs again.
    // If the first request succeeded but the response was lost,
    // this returns 409 Conflict — which is WRONG.
    // The client should get the original 201 Created.
    writeJSON(w, http.StatusCreated, user)
}
```

The fundamental issue: the server has no memory of whether it already processed
this request. Without that memory, it cannot return the original response.

---

## 2. Idempotency-Key Header Pattern

### 2.1 The Standard

The `Idempotency-Key` header is an emerging standard (used by Stripe, AWS,
and others, documented in IETF drafts). The client generates a UUID per
logical operation and sends it with the request. The server uses it to
deduplicate.

```
POST /api/v1/users HTTP/1.1
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
Content-Type: application/json

{"username": "alice", "email": "alice@example.com"}
```

### 2.2 Server-Side Flow

```
1. Request arrives with Idempotency-Key
2. Check: have we seen this key before?
   - YES: return the cached response (original status + body)
   - NO: proceed to execute, then store key → response
3. Concurrent requests with the same key:
   - First request acquires the key (in-flight)
   - Second request waits or returns 409
```

### 2.3 Go Middleware Implementation

```go
package middleware

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
)

const idempotencyHeader = "Idempotency-Key"
const idemTTL = 24 * time.Hour

// IdempotencyResponse is the cached response stored in Redis.
type IdempotencyResponse struct {
    Status  int               `json:"status"`
    Body    json.RawMessage   `json:"body"`
    Headers map[string]string `json:"headers,omitempty"`
}

// IdempotencyMiddleware wraps POST/PUT/PATCH handlers with idempotency
// protection. GET/DELETE are naturally idempotent and skip this middleware.
func IdempotencyMiddleware(rdb *redis.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Only protect non-idempotent methods.
            if r.Method != http.MethodPost &&
               r.Method != http.MethodPut &&
               r.Method != http.MethodPatch {
                next.ServeHTTP(w, r)
                return
            }

            key := r.Header.Get(idempotencyHeader)
            if key == "" {
                // No idempotency key — pass through (no protection).
                next.ServeHTTP(w, r)
                return
            }

            ctx := r.Context()
            redisKey := "idem:" + key

            // Step 1: Try to load a completed response.
            val, err := rdb.Get(ctx, redisKey).Bytes()
            if err == nil {
                // Key exists — return cached response.
                var cached IdempotencyResponse
                if json.Unmarshal(val, &cached) == nil {
                    for k, v := range cached.Headers {
                        w.Header().Set(k, v)
                    }
                    w.Header().Set("X-Idempotent-Replay", "true")
                    w.WriteHeader(cached.Status)
                    w.Write(cached.Body)
                    return
                }
            }

            // Step 2: Try to claim the key atomically (SETNX).
            // We use a sentinel value "processing" to mark in-flight state.
            claimed, err := rdb.SetNX(ctx, redisKey, "processing", idemTTL).Result()
            if err != nil {
                // Redis error — fail open (process without idempotency).
                next.ServeHTTP(w, r)
                return
            }
            if !claimed {
                // Key exists but value is "processing" — another request
                // is in-flight. Return 409 Conflict.
                w.Header().Set("Retry-After", "5")
                writeJSON(w, http.StatusConflict, map[string]string{
                    "error": "idempotency key is in-flight, retry shortly",
                })
                return
            }

            // Step 3: Execute the actual handler.
            rec := &responseRecorder{
                ResponseWriter: w,
                body:           &bytes.Buffer{},
                status:         http.StatusOK,
            }
            next.ServeHTTP(rec, r)

            // Step 4: Store the completed response.
            cached := IdempotencyResponse{
                Status: rec.status,
                Body:   rec.body.Bytes(),
            }
            data, _ := json.Marshal(cached)
            rdb.Set(ctx, redisKey, data, idemTTL)
        })
    }
}

// responseRecorder captures the handler's response for caching.
type responseRecorder struct {
    http.ResponseWriter
    body   *bytes.Buffer
    status int
}

func (r *responseRecorder) WriteHeader(code int) {
    r.status = code
    r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
    r.body.Write(b)
    return r.ResponseWriter.Write(b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}
```

### 2.4 Client Retry Example

```go
// Client-side: retry with idempotency key.
func create_user_WithRetry(client *http.Client, url string, user *UserInput) (*User, error) {
    idemKey := uuid.New().String() // Generate once, reuse on retries

    for attempt := 0; attempt < 3; attempt++ {
        body, _ := json.Marshal(user)
        req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Idempotency-Key", idemKey) // Same key every attempt

        resp, err := client.Do(req)
        if err != nil {
            time.Sleep(time.Duration(attempt+1) * time.Second)
            continue
        }
        defer resp.Body.Close()

        // Same response whether first attempt or retry.
        var result User
        json.NewDecoder(resp.Body).Decode(&result)
        return &result, nil
    }
    return nil, fmt.Errorf("max retries exceeded")
}
```

---

## 3. Idempotency Storage

### 3.1 Redis as Idempotency Store

Redis is the ideal idempotency store because:

- **SETNX** provides atomic claim — only one request wins the race.
- **TTL** provides automatic cleanup — old keys expire without manual sweeping.
- **Sub-millisecond latency** — negligible overhead per request.
- **Distributed** — works across multiple gateway/service instances.

### 3.2 Key Design

```
idem:{idempotency-key}    → JSON response or "processing"
idem:{key}:inflight        → (optional) separate key for in-flight tracking
```

The TTL should cover the maximum expected client retry window:
- **24 hours** for most operations (Stripe's default).
- **7 days** for long-running operations (SCIM bulk provisioning).

### 3.3 Full Redis-Backed Idempotency Store

```go
package idempotency

import (
    "context"
    "encoding/json"
    "errors"
    "time"

    "github.com/redis/go-redis/v9"
)

var (
    ErrKeyInFlight = errors.New("idempotency key is in-flight")
    ErrKeyNotFound = errors.New("no cached response for key")
)

// Store provides Redis-backed idempotency key management.
type Store struct {
    rdb *redis.Client
    ttl time.Duration
}

func NewStore(rdb *redis.Client, ttl time.Duration) *Store {
    if ttl == 0 {
        ttl = 24 * time.Hour
    }
    return &Store{rdb: rdb, ttl: ttl}
}

// CachedResponse holds the full HTTP response for replay.
type CachedResponse struct {
    Status      int               `json:"status"`
    Body        []byte            `json:"body"`
    Headers     map[string]string `json:"headers,omitempty"`
    CachedAt    time.Time         `json:"cached_at"`
}

// Claim atomically marks a key as "in-flight". Returns:
//   - (true, nil) if the key was successfully claimed (first caller).
//   - (false, ErrKeyInFlight) if another request holds the key.
//   - (false, error) on Redis failure.
func (s *Store) Claim(ctx context.Context, key string) (bool, error) {
    ok, err := s.rdb.SetNX(ctx, s.key(key), "processing", s.ttl).Result()
    if err != nil {
        return false, err
    }
    if !ok {
        return false, ErrKeyInFlight
    }
    return true, nil
}

// Get retrieves a cached response for a previously-seen key.
func (s *Store) Get(ctx context.Context, key string) (*CachedResponse, error) {
    val, err := s.rdb.Get(ctx, s.key(key)).Bytes()
    if err == redis.Nil {
        return nil, ErrKeyNotFound
    }
    if err != nil {
        return nil, err
    }

    // Check if still "processing" (in-flight sentinel).
    if string(val) == "processing" {
        return nil, ErrKeyInFlight
    }

    var cached CachedResponse
    if err := json.Unmarshal(val, &cached); err != nil {
        return nil, err
    }
    return &cached, nil
}

// Store saves the completed response for future replays.
func (s *Store) Store(ctx context.Context, key string, resp *CachedResponse) error {
    data, err := json.Marshal(resp)
    if err != nil {
        return err
    }
    return s.rdb.Set(ctx, s.key(key), data, s.ttl).Err()
}

// Release removes the in-flight marker without storing a response
// (used on handler error to allow retry).
func (s *Store) Release(ctx context.Context, key string) error {
    return s.rdb.Del(ctx, s.key(key)).Err()
}

func (s *Store) key(k string) string {
    return "ggid:idem:" + k
}
```

### 3.4 Handling Large Responses

Storing full response bodies in Redis can be problematic for large payloads
(e.g., SCIM bulk operations returning hundreds of users). Strategies:

```go
// Strategy 1: Store a hash reference for large bodies (> 4 KB).
const inlineThreshold = 4 * 1024 // 4 KB

func (s *Store) Store(ctx context.Context, key string, resp *CachedResponse) error {
    if len(resp.Body) > inlineThreshold {
        // Store body in a separate key, reference it in the metadata.
        bodyKey := s.key(key) + ":body"
        if err := s.rdb.Set(ctx, bodyKey, resp.Body, s.ttl).Err(); err != nil {
            return err
        }
        resp.Body = nil
        resp.Headers["X-Idempotency-Body-Key"] = bodyKey
    }
    data, _ := json.Marshal(resp)
    return s.rdb.Set(ctx, s.key(key), data, s.ttl).Err()
}

// Strategy 2: For truly massive responses, only cache the status + Location
// header. The client can follow the redirect to get the full resource.
// This is appropriate for 201 Created responses where Location is set.
```

---

## 4. Natural Idempotency Keys

### 4.1 Business Keys vs UUIDs

Not all idempotency requires a client-supplied UUID. Many IAM operations have
natural business keys:

| Operation | Natural Key | Idempotency Mechanism |
|---|---|---|
| Create user | email (per tenant) | UNIQUE constraint |
| Create role | role key (per tenant) | UNIQUE constraint |
| Assign role | (user_id, role_id, scope) | ON CONFLICT DO NOTHING |
| Create policy | policy name (per tenant) | UNIQUE constraint |
| Create org | org slug (per tenant) | UNIQUE constraint |

Natural keys are superior because they work even without the client sending
an Idempotency-Key header. They enforce idempotency at the data layer.

### 4.2 Upsert Pattern in SQL

```sql
-- User creation with natural key idempotency.
-- If the email already exists, return the existing user.
INSERT INTO users (id, tenant_id, username, email, status, created_at)
VALUES ($1, $2, $3, $4, 'active', NOW())
ON CONFLICT (tenant_id, email) DO NOTHING
RETURNING id, username, email;

-- If RETURNING returns nothing (conflict happened), do a SELECT.
SELECT id, username, email FROM users
WHERE tenant_id = $1 AND email = $2;
```

### 4.3 Go Implementation

```go
// CreateUserIdempotent creates a user or returns the existing one if the
// email is already registered. This is truly idempotent — calling it N times
// with the same input produces the same result.
func (r *UserRepository) CreateUserOrGet(ctx context.Context, user *domain.User) (*domain.User, bool, error) {
    query := `
        INSERT INTO users (id, tenant_id, username, email, status, password_hash, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())
        ON CONFLICT (tenant_id, email) DO NOTHING
        RETURNING id, created_at`

    var created bool
    err := r.db.QueryRow(ctx, query,
        user.ID, user.TenantID, user.Username, user.Email,
        user.Status, user.PasswordHash,
    ).Scan(&user.ID, &user.CreatedAt)

    if err != nil {
        // ON CONFLICT DO NOTHING returns pgx.ErrNoRows — the user already exists.
        // Fetch the existing user.
        existing, getErr := r.GetUserByEmail(ctx, user.TenantID, user.Email)
        if getErr != nil {
            return nil, false, getErr
        }
        return existing, false, nil
    }

    created = true
    return user, created, nil
}
```

### 4.4 When to Use Natural Keys vs Idempotency-Key Headers

| Scenario | Preferred Approach |
|---|---|
| User registration (email unique) | Natural key (UNIQUE constraint) |
| Role assignment (composite unique) | Natural key (ON CONFLICT) |
| Login (no business uniqueness) | Idempotency-Key header |
| Token refresh (opaque token) | Natural key (token hash) |
| Bulk SCIM operations | Idempotency-Key header |
| Webhook delivery (external system) | Event ID deduplication |

---

## 5. Idempotency for NATS Events

### 5.1 The Double-Delivery Problem

NATS JetStream provides at-least-once delivery. In practice this means
a consumer may receive the same message more than once:

```
Producer → NATS JetStream → Consumer
                              |
                              ├─ Process → DB insert → ACK
                              │   (ACK lost in network)
                              |
                              ├─ Redelivery (same message)
                              │   Process → DB insert → ACK (DUPLICATE!)
```

JetStream guarantees redelivery, not exactly-once. The consumer must be
idempotent.

### 5.2 NATS Built-in Deduplication

JetStream supports server-side deduplication via the `Nats-Msg-Id` header.
When publishing:

```go
// Publisher: set a unique message ID for deduplication.
headers := nats.Header{}
headers.Set(nats.MsgIdHdr, event.ID.String()) // Use event UUID as Msg-Id
msg := nats.NewMsg(subject)
msg.Data = data
msg.Header = headers

_, err = js.PublishMsg(ctx, msg)
```

JetStream deduplicates messages with the same `Nats-Msg-Id` within the
stream's `DuplicateWindow` (default 2 minutes). This prevents the producer
from accidentally publishing duplicates.

### 5.3 Consumer-Side Deduplication

Server-side deduplication handles producer retries but not consumer-side
redelivery (different delivery attempts of the same stored message). For
consumer-side deduplication:

```go
// IdempotentNATSConsumer wraps the standard consumer with deduplication.
type IdempotentConsumer struct {
    rdb       *redis.Client
    dedupTTL  time.Duration
}

// ProcessMessage handles a NATS message idempotently.
func (c *IdempotentConsumer) ProcessMessage(ctx context.Context, msg jetstream.Msg) error {
    var event AuditEvent
    if err := json.Unmarshal(msg.Data(), &event); err != nil {
        return nil // Poison message — ACK and drop.
    }

    // Dedup key: event ID (should be globally unique).
    dedupKey := "audit:dedup:" + event.ID.String()

    // SETNX — if we already processed this event ID, skip.
    claimed, err := c.rdb.SetNX(ctx, dedupKey, "processed", c.dedupTTL).Result()
    if err != nil {
        return fmt.Errorf("dedup check: %w", err)
    }
    if !claimed {
        // Already processed — ACK without re-inserting.
        log.Printf("Audit: skipping duplicate event %s", event.ID)
        return nil
    }

    // Insert into database — safe because we hold the dedup key.
    if err := c.repo.Insert(ctx, &event); err != nil {
        // On failure, release the dedup key so NATS redelivery can retry.
        c.rdb.Del(ctx, dedupKey)
        return err
    }

    return nil
}
```

### 5.4 Database-Level Deduplication

The strongest guarantee is a UNIQUE constraint on the event ID:

```sql
-- Audit events table with unique event ID.
CREATE TABLE audit_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID NOT NULL,   -- external event ID from NATS
    tenant_id   UUID NOT NULL,
    action      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Prevent duplicate inserts for the same external event.
    UNIQUE (event_id)
);

-- Consumer insert with conflict handling.
INSERT INTO audit_events (event_id, tenant_id, action)
VALUES ($1, $2, $3)
ON CONFLICT (event_id) DO NOTHING;
```

This provides idempotency even if Redis dedup fails or the consumer restarts.

---

## 6. Idempotency for Token Issuance

### 6.1 Authorization Code Flow (Already Idempotent)

The OAuth authorization code flow is idempotent by design:

1. User visits `/authorize` — gets a one-time code.
2. Client exchanges code for tokens at `/token`.
3. **The code is single-use** — a second exchange with the same code fails.

This built-in single-use property makes the code exchange naturally
idempotent. If the client retries the token request (e.g., timeout), the
second attempt gets an error ("invalid_grant"), and the client knows the
first attempt either succeeded or failed.

### 6.2 Refresh Token Rotation (NOT Idempotent by Default)

Refresh token rotation is where idempotency breaks:

```
1. Client sends refresh_token=A → server rotates A→B, returns B
2. Client times out, retries with refresh_token=A
3. Server sees A is already revoked → returns error
4. But client needs the new token B!
```

Without special handling, the client is stuck: the old token is revoked,
the new token was issued but lost.

### 6.3 Token Family Tracking (GGID's Approach)

GGID already implements replay detection via token family tracking
(see `services/auth/internal/service/token_service.go`):

```go
// RotateRefreshToken revokes the old token and issues a new one.
// If the old token was already revoked, it detects a replay attack
// and revokes the ENTIRE session.
func (ts *TokenService) RotateRefreshToken(ctx context.Context, plaintext string) (string, *domain.RefreshToken, error) {
    tokenHash := hashToken(plaintext)

    // Check DB for the token's state.
    oldToken, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
    if err != nil {
        return "", nil, fmt.Errorf("find refresh token: %w", err)
    }
    if oldToken == nil {
        return "", nil, fmt.Errorf("refresh token is invalid or expired")
    }

    // Replay detection: if already revoked, revoke entire session.
    if !oldToken.IsActive() {
        _ = ts.refreshRepo.RevokeAllForSession(ctx, oldToken.SessionID)
        return "", nil, fmt.Errorf("refresh token replay detected — session revoked")
    }

    // Revoke old, issue new.
    ts.refreshRepo.Revoke(ctx, oldToken.ID)
    // ... issue new token with RotatedFrom = oldToken.ID ...
}
```

This is a **security-first** approach: a replayed token revokes the session.
However, it is NOT idempotent from the client's perspective — a legitimate
retry (lost response) is treated as a replay attack and locks the user out.

### 6.4 Idempotent Refresh Token Rotation

To make refresh truly idempotent, we cache the first rotation result:

```go
// IdempotentRotate checks if a rotation already happened for this token.
// If so, returns the cached result instead of treating it as a replay.
func (ts *TokenService) IdempotentRotate(ctx context.Context, plaintext string) (*TokenSet, error) {
    tokenHash := hashToken(plaintext)
    idemKey := "rotate:" + tokenHash

    // Check if we already rotated this token (response was cached).
    cached, err := ts.rdb.Get(ctx, idemKey).Result()
    if err == nil {
        // Return cached token set from the previous successful rotation.
        var set TokenSet
        if json.Unmarshal([]byte(cached), &set) == nil {
            return &set, nil
        }
    }

    // Check if a rotation is in-flight (another concurrent request).
    claimed, _ := ts.rdb.SetNX(ctx, idemKey+":lock", "1", 30*time.Second).Result()
    if !claimed {
        return nil, ErrRotationInProgress
    }
    defer ts.rdb.Del(ctx, idemKey+":lock")

    // Perform the actual rotation.
    oldToken, err := ts.refreshRepo.FindByHash(ctx, tokenHash)
    if err != nil || oldToken == nil {
        return nil, ErrInvalidRefreshToken
    }

    if !oldToken.IsActive() {
        // Distinguish replay from legitimate retry:
        // If we have a cached rotation, return it.
        // Otherwise, this is a genuine replay — revoke session.
        return nil, ErrReplayDetected
    }

    // Rotate: revoke old, issue new.
    newPlaintext, newToken, err := ts.doRotate(ctx, oldToken)
    if err != nil {
        return nil, err
    }

    // Cache the result for 60 seconds — long enough for a client retry.
    set := &TokenSet{
        AccessToken:  accessToken,
        RefreshToken: newPlaintext,
        ExpiresIn:    accessTTL,
    }
    data, _ := json.Marshal(set)
    ts.rdb.Set(ctx, idemKey, data, 60*time.Second)

    return set, nil
}
```

This approach balances security (replay detection still fires for genuinely
stolen tokens used after the 60-second window) with usability (legitimate
retries within 60 seconds succeed).

---

## 7. Idempotency for Role/Policy Operations

### 7.1 AssignRole Should Be Idempotent

Assigning the same role to the same user in the same scope is conceptually
a no-op. GGID already implements this correctly at the database level:

```go
// GGID's existing Assign method (user_role_policy_repo.go):
func (r *UserRoleRepository) Assign(ctx context.Context, ur *domain.UserRole) error {
    query := `
        INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (user_id, role_id, scope_type, scope_id) DO UPDATE
            SET granted_by = EXCLUDED.granted_by, expires_at = EXCLUDED.expires_at
        RETURNING created_at`
    return r.db.QueryRow(ctx, query, ...).Scan(&ur.CreatedAt)
}
```

This `ON CONFLICT DO UPDATE` makes `AssignRole` idempotent: calling it twice
with the same (user, role, scope) tuple updates the row rather than creating
a duplicate.

### 7.2 DELETE Is Naturally Idempotent

```go
// Deleting a role assignment twice is harmless.
// First call: DELETE FROM user_roles WHERE ... → deletes 1 row.
// Second call: DELETE FROM user_roles WHERE ... → deletes 0 rows.
// Both return success (or 404 if the caller expects the resource to exist).
```

The policy repository already handles this:

```go
func (r *PolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
    cmd, err := r.db.Exec(ctx, `DELETE FROM policies WHERE id = $1`, id)
    if cmd.RowsAffected() == 0 {
        return notFound("policy", id.String())
    }
    return err
}
```

### 7.3 UPDATE: Last-Write-Wins vs Optimistic Locking

UPDATE operations present a challenge. Consider two concurrent updates:

```
T1: UPDATE user SET display_name='Alice' WHERE id=1  (version=1)
T2: UPDATE user SET display_name='Bob' WHERE id=1    (version=1)
```

Without versioning, this is a lost-update problem. The last writer wins,
and the first update is silently overwritten.

```go
// Last-write-wins (no version check) — simple but dangerous.
func (r *UserRepository) UpdateUser(ctx context.Context, id uuid.UUID, input *UpdateInput) error {
    _, err := r.db.Exec(ctx,
        `UPDATE users SET display_name = $2, updated_at = NOW() WHERE id = $1`,
        id, input.DisplayName)
    return err
}

// Optimistic locking (version check) — safe for concurrent updates.
func (r *UserRepository) UpdateUserVersioned(ctx context.Context, id uuid.UUID, version int, input *UpdateInput) error {
    cmd, err := r.db.Exec(ctx,
        `UPDATE users
         SET display_name = $3, version = version + 1, updated_at = NOW()
         WHERE id = $1 AND version = $2`,
        id, version, input.DisplayName)
    if cmd.RowsAffected() == 0 {
        return ErrConcurrentModification // version mismatch
    }
    return err
}
```

### 7.4 Policy Attachment Idempotency

GGID's policy attachment uses `ON CONFLICT DO NOTHING`, which is correctly
idempotent:

```go
func (r *PolicyRepository) AttachPolicy(ctx context.Context, attachment *domain.PolicyAttachment) error {
    query := `
        INSERT INTO policy_attachments (policy_id, principal_type, principal_id)
        VALUES ($1, $2, $3)
        ON CONFLICT DO NOTHING`
    _, err := r.db.Exec(ctx, query, ...)
    return err
}
```

---

## 8. Idempotency vs Distributed Locks

### 8.1 When to Use Which

| Aspect | Idempotency | Distributed Lock |
|---|---|---|
| Strategy | Optimistic | Pessimistic |
| Mechanism | Detect duplicate after execution | Prevent concurrent execution |
| Scalability | High (parallel execution OK) | Low (serialized) |
| Complexity | Moderate (need key management) | High (deadlock, lease renewal) |
| Failure mode | Duplicate execution (then cached) | Lock holder crash → stuck |
| Best for | POST/PUT operations | Read-modify-write sequences |

### 8.2 When Locks Are Necessary

Idempotency handles duplicate requests, but some operations require
mutual exclusion even for first-time requests:

- **Counter increment**: "increment user count" is not idempotent — the
  second call changes the value again.
- **Resource allocation**: "assign next available license slot" — two
  concurrent requests might grab the same slot.
- **Sequential operations**: "generate invoice number" — must be unique and sequential.

For these, a distributed lock is the right tool.

### 8.3 Redis Distributed Lock Pattern

```go
package lock

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
)

var ErrLockNotAcquired = errors.New("could not acquire lock")

// DistributedLock provides a Redis-based mutex with TTL.
type DistributedLock struct {
    rdb    *redis.Client
    key    string
    token  string // unique token to prevent releasing another holder's lock
    ttl    time.Duration
}

// Acquire tries to get a lock. Returns ErrLockNotAcquired if already held.
func Acquire(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration) (*DistributedLock, error) {
    token := uuid.New().String()
    ok, err := rdb.SetNX(ctx, "lock:"+key, token, ttl).Result()
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, ErrLockNotAcquired
    }
    return &DistributedLock{rdb: rdb, key: key, token: token, ttl: ttl}, nil
}

// Release removes the lock. Uses a Lua script to ensure we only release
// our own lock (not another process that acquired it after expiry).
func (l *DistributedLock) Release(ctx context.Context) error {
    // Lua script: only delete if the stored value matches our token.
    script := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("DEL", KEYS[1])
        else
            return 0
        end`
    _, err := l.rdb.Eval(ctx, script, []string{"lock:" + l.key}, l.token).Result()
    return err
}

// WithLock executes a function while holding a distributed lock.
func WithLock(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration, fn func() error) error {
    lock, err := Acquire(ctx, rdb, key, ttl)
    if err != nil {
        return err
    }
    defer lock.Release(ctx)
    return fn()
}
```

### 8.4 Usage Example: Sequential License Assignment

```go
func (s *LicenseService) AssignLicense(ctx context.Context, userID uuid.UUID) error {
    // Use a lock to prevent two concurrent assignments from getting the
    // same slot.
    return lock.WithLock(ctx, s.rdb, "license-assign:"+userID.String(), 10*time.Second, func() error {
        // Check current license count.
        count, err := s.repo.GetActiveLicenseCount(ctx)
        if err != nil {
            return err
        }
        if count >= s.maxLicenses {
            return ErrNoLicensesAvailable
        }
        // Assign — safe because we hold the lock.
        return s.repo.AssignLicense(ctx, userID)
    })
}
```

### 8.5 Performance Comparison

| Operation | Idempotency (Redis SETNX) | Lock (Redis SETNX + TTL + Lua) |
|---|---|---|
| Latency per request | 0.3 ms (1 SETNX) | 0.6 ms (SETNX + Lua DEL) |
| Throughput | Unlimited (parallel) | Serialized (1 at a time) |
| Correctness | At-most-once for duplicates | At-most-one concurrent |

For IAM operations, **prefer idempotency** for user/role/policy CRUD.
Use locks only for resource-constrained operations (license allocation,
sequence generation).

---

## 9. Idempotency Audit Trail

### 9.1 Why Log Duplicates?

When an idempotency key results in a cached replay, the operation is NOT
re-executed — but it should still be logged. The audit trail should record:

1. **First execution**: normal audit event (action=user.create, result=success).
2. **Duplicate/replay**: flagged event (action=user.create, result=success,
   metadata.is_idempotent_replay=true).

### 9.2 What Duplicate Patterns Reveal

| Pattern | Possible Cause |
|---|---|
| Many replays from same client | Client-side timeout too short |
| Replays across different IPs | Possible replay attack or load balancer issue |
| Replays > 5 minutes apart | Client retry logic is overly aggressive |
| Replays for specific endpoints | Network issue between gateway and that service |

### 9.3 Go Implementation

```go
// IdempotencyAuditLogger wraps the audit publisher to log idempotency events.
type IdempotencyAuditLogger struct {
    publisher *audit.Publisher
}

// LogReplay records that an idempotent request was served from cache.
func (l *IdempotencyAuditLogger) LogReplay(ctx context.Context, key string, action string, originalTime time.Time) {
    event := audit.Event{
        ID:        uuid.New(),
        Action:    action,
        Result:    "success",
        Metadata: map[string]any{
            "is_idempotent_replay": true,
            "idempotency_key":      key,
            "original_time":        originalTime,
            "replay_delay_ms":      time.Since(originalTime).Milliseconds(),
        },
        CreatedAt: time.Now(),
    }
    // Publish asynchronously — don't block the response.
    go l.publisher.PublishAsync(event)
}
```

### 9.4 Integration with Middleware

```go
// In the IdempotencyMiddleware, after returning a cached response:
if cached, ok := store.Get(ctx, key); ok {
    // Log the replay for observability.
    auditLogger.LogReplay(ctx, key, actionFromPath(r.URL.Path), cached.CachedAt)

    // Return cached response.
    writeResponse(w, cached)
    return
}
```

### 9.5 Monitoring Queries

```sql
-- Count idempotent replays by action (last 24 hours).
SELECT
    metadata->>'action' AS action,
    COUNT(*) AS replay_count,
    AVG((metadata->>'replay_delay_ms')::int) AS avg_delay_ms
FROM audit_events
WHERE metadata->>'is_idempotent_replay' = 'true'
    AND created_at > NOW() - INTERVAL '24 hours'
GROUP BY metadata->>'action'
ORDER BY replay_count DESC;

-- Alert: a single client generating excessive replays may indicate a bug.
```

---

## 10. GGID Idempotency Gap Analysis

### 10.1 What Exists

| Component | File | Idempotency Mechanism | Status |
|---|---|---|---|
| Gateway coalesce | `services/gateway/internal/middleware/coalesce.go` | In-memory request coalescing for GET + Idempotency-Key POST/PUT/PATCH | Partial — in-memory only (single instance) |
| Role assignment | `services/policy/internal/repository/user_role_policy_repo.go` | `ON CONFLICT DO UPDATE` on (user_id, role_id, scope_type, scope_id) | Correct |
| Policy attachment | `services/policy/internal/repository/user_role_policy_repo.go` | `ON CONFLICT DO NOTHING` | Correct |
| User registration | `services/identity/internal/service/identity_service.go` | Check-then-insert (GetUserByEmail → CreateUser) | Weak — TOCTOU race |
| Refresh token rotation | `services/auth/internal/service/token_service.go` | Token family tracking + replay detection | Security-first — not client-idempotent |
| LDAP JIT provisioning | `services/identity/internal/service/identity_service.go` | Check external identity before create | Partial — race possible |
| NATS consumer | `services/audit/internal/consumer/nats_consumer.go` | None — inserts every delivered message | Missing |
| NATS publisher | `pkg/audit/publisher.go` | None — no Nats-Msg-Id header | Missing |

### 10.2 What's Missing

#### Gap 1: User Registration TOCTOU Race

The current `CreateUser` checks for existing email via a separate query,
then inserts. Under concurrent requests, two goroutines can pass the check
simultaneously and both insert:

```go
// Current (race-prone):
if existing, _ := s.repo.GetUserByEmail(ctx, tc.TenantID, input.Email); existing != nil {
    return nil, gerr.AlreadyExists("email", input.Email)
}
// ... another request can insert between here ...
user := &domain.User{...}
s.repo.CreateUser(ctx, user) // May violate UNIQUE constraint → 500 error
```

The database UNIQUE constraint prevents duplicate data but the error is
returned as a 500 Internal Server Error rather than a clean 409 Conflict.

#### Gap 2: No Redis-Backed Idempotency Store

The gateway coalesce middleware uses in-memory maps (`sync.Mutex` + `map`).
This only works for single-instance deployments. In a multi-instance
deployment (Kubernetes with multiple gateway pods), an idempotency key on
instance A is invisible to instance B.

#### Gap 3: No NATS Message Deduplication

The audit publisher does not set `Nats-Msg-Id` headers. The consumer does
not check for duplicate event IDs before inserting. If the publisher retries
(NATS reconnect + publish), duplicate audit events are inserted.

The audit consumer (`processMessage`) directly calls `repo.Insert` without
any dedup check:

```go
func (c *EventConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
    var event domain.AuditEvent
    json.Unmarshal(msg.Data(), &event)
    // No dedup check — duplicate events are inserted.
    return c.repo.Insert(ctx, &event)
}
```

#### Gap 4: Refresh Token Rotation Is Not Client-Idempotent

When a client retries a refresh request (response lost), the server treats
it as a replay attack and revokes the entire session. This is
security-correct but user-hostile — the user is logged out because of a
network timeout.

#### Gap 5: No Idempotency-Key Header Propagation

The `Idempotency-Key` header is only processed by the gateway coalesce
middleware. Individual services (identity, auth, policy) do not check for
or propagate this header. If the gateway coalesce cache misses (in-memory,
no Redis), the downstream service receives and processes the duplicate.

---

## 11. Gap Analysis and Recommendations

### Recommendation Summary

| # | Action | Priority | Effort | Impact |
|---|---|---|---|---|
| 1 | Add UNIQUE constraint + clean error mapping for user registration | P0 | 2 hours | Fixes TOCTOU race, returns 409 instead of 500 |
| 2 | Implement Redis-backed idempotency store for gateway middleware | P1 | 1 day | Enables multi-instance idempotency for all POST/PUT |
| 3 | Add NATS Msg-Id deduplication for audit events | P1 | 4 hours | Prevents duplicate audit events on publisher retry |
| 4 | Add consumer-side deduplication + DB UNIQUE constraint for audit events | P1 | 4 hours | Prevents duplicate inserts on consumer redelivery |
| 5 | Implement client-idempotent refresh token rotation (60s grace cache) | P2 | 1 day | Prevents user lockout on legitimate retries |

### Detailed Action Items

#### Action 1: Fix User Registration TOCTOU (P0, ~2 hours)

Add a `ON CONFLICT` handler to `CreateUser` in
`services/identity/internal/repository/pg_repo.go`. Map PostgreSQL unique
violation errors to `gerr.AlreadyExists` so the API returns 409 Conflict
instead of 500 Internal Server Error.

```go
// In pg_repo.go CreateUser:
err := r.db.QueryRow(ctx, query, ...).Scan(...)
if err != nil {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
        return nil, gerr.AlreadyExists("user", input.Email)
    }
    return nil, err
}
```

#### Action 2: Redis-Backed Idempotency Store (P1, ~1 day)

Replace the in-memory cache in `coalesce.go` with the Redis-backed
`IdempotencyStore` from Section 3.3. This makes idempotency work across
multiple gateway instances. The middleware already handles the
`Idempotency-Key` header — just swap the storage backend.

Key changes:
- Add `redis.Client` to `RequestCoalescer` struct.
- Replace `cache map[string]*cachedResponse` with Redis GET/SET calls.
- Keep in-memory map for GET request coalescing (in-flight only), use Redis
  for idempotency-key response caching.

#### Action 3: NATS Publisher Deduplication (P1, ~4 hours)

In `pkg/audit/publisher.go`, add `Nats-Msg-Id` header to published messages
and set `DuplicateWindow` on the stream:

```go
func (p *Publisher) Publish(ctx context.Context, event Event) error {
    // ... existing code ...
    opts := []jetstream.PublishOpt{
        jetstream.WithMsgID(event.ID.String()), // Dedup key
    }
    _, err = p.js.Publish(ctx, p.subject, data, opts...)
    // ...
}
```

Also update `ensureStream` to include `DuplicateWindow: 5 * time.Minute`.

#### Action 4: Consumer-Side Audit Deduplication (P1, ~4 hours)

In `services/audit/internal/consumer/nats_consumer.go`:
1. Add a UNIQUE constraint on `event_id` in the audit events table.
2. Change `repo.Insert` to use `ON CONFLICT (event_id) DO NOTHING`.
3. Add a Redis dedup layer (Section 5.3) for fast-path dedup before DB hit.

#### Action 5: Idempotent Refresh Token Rotation (P2, ~1 day)

In `services/auth/internal/service/token_service.go`, add a 60-second grace
cache for rotation results (Section 6.4). This distinguishes legitimate
client retries (within 60s) from genuine replay attacks (after 60s).

The grace period is deliberately short to maintain security: a stolen token
used after 60 seconds still triggers full session revocation.

---

## Appendix: Idempotency Method Reference

| HTTP Method | Naturally Idempotent | With Idempotency-Key | Recommendation |
|---|---|---|---|
| GET | Yes | N/A (cache instead) | Standard |
| DELETE | Yes | Optional | Standard |
| PUT | Yes (replaces state) | Optional | Standard |
| POST | No | Yes | Required for safety |
| PATCH | No | Yes | Required for safety |

---

*End of document.*
