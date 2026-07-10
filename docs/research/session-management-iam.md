# Session Management Architecture for IAM Systems

> Architecture patterns, timeout strategies, concurrent session management, and
> GGID-specific gap analysis. Complements three existing research documents:
>
> - `docs/research/openid-connect-session-management.md` — OIDC check_session iframe,
>   postMessage, cookie coordination.
> - `docs/research/session-fixation-prevention.md` — Session fixation vectors,
>   session ID regeneration.
> - `docs/research/session-management-design.md` — GGID's current JWT+opaque hybrid,
>   refresh token rotation, revocation roadmap.

---

## Table of Contents

1. [Stateless JWT vs Server-Side Sessions](#1-stateless-jwt-vs-server-side-sessions)
2. [Sliding vs Absolute Timeout](#2-sliding-vs-absolute-timeout)
3. [Concurrent Session Limits](#3-concurrent-session-limits)
4. [Session Token Storage Strategies](#4-session-token-storage-strategies)
5. [Session Revocation Architecture](#5-session-revocation-architecture)
6. [Session Analytics](#6-session-analytics)
7. [Device-Bound Sessions](#7-device-bound-sessions)
8. [GGID Session Management Gap Analysis](#8-ggid-session-management-gap-analysis)
9. [Gap Analysis & Recommendations](#9-gap-analysis--recommendations)

---

## 1. Stateless JWT vs Server-Side Sessions

The fundamental architectural decision in IAM session management is **stateless vs stateful**.

### Comparison Matrix

| Dimension | Stateless JWT | Server-Side Session (Redis) |
|---|---|---|
| Validation cost | Signature verification (CPU, ~0.1ms) | Redis GET (~0.5ms) |
| Horizontal scaling | Trivial (no shared state) | Requires shared Redis cluster |
| Revocation latency | Cannot revoke before `exp` | Immediate (delete key) |
| Token size | ~1-2 KB (RS256 with claims) | ~32-64 bytes (opaque ID) |
| Payload leakage | Claims visible to client (base64) | Zero (only random ID transmitted) |
| Clock skew sensitivity | High (`exp`/`iat`/`nbf`) | None |
| Offline validation | Yes (public key only) | No (requires Redis) |
| Infrastructure | JWKS endpoint only | Redis cluster + failover |

### Hybrid Architecture (Industry Standard)

The recommended pattern combines the strengths of both:

- **Access token**: short-lived JWT (5-15 min). Stateless verification at the API gateway.
  No DB/Redis lookup. Revocation is not possible before expiry, but the short TTL
  limits exposure.
- **Refresh token**: opaque, server-side stored (Redis + DB). Long-lived (7-30 days).
  Validated on every refresh request. Revocable immediately.

```
┌──────────────┐     ┌───────────┐     ┌────────┐
│   Client     │────▶│  Gateway  │────▶│  API   │
│ (AT + RT)    │     │ (verify   │     │ Service│
│              │     │  RS256)   │     │        │
└──────────────┘     └───────────┘     └────────┘
      │                                       │
      │  POST /refresh (sends RT)             │
      ▼                                       ▼
┌──────────────┐                        ┌────────┐
│ Auth Service │◀───────────────────────│ Redis  │
│ (rotate RT)  │                        │ (RT)   │
└──────────────┘                        └────────┘
```

### When to Use Each

| Scenario | Recommended Approach | Rationale |
|---|---|---|
| High-traffic read API | Stateless JWT | Avoid Redis bottleneck; cache TTL handles stale reads |
| Financial / admin operations | Server-side session | Need instant revocation for security |
| Mobile app with offline support | Hybrid (JWT + RT) | App can show cached data offline |
| Third-party API integration | Stateless JWT | Third party needs to verify without calling back |
| Cross-domain SSO | Opaque session cookie + SAML/OIDC | Central session store enables SLO |

### Decision Framework

```go
// SessionStrategy determines the session validation approach per tenant or route.
type SessionStrategy int

const (
    // StrategyStateless: JWT signature verification only. Lowest latency,
    // but tokens cannot be revoked before expiry.
    StrategyStateless SessionStrategy = iota

    // StrategyStateful: Redis lookup on every request. Highest security,
    // supports instant revocation, but adds ~0.5ms per request.
    StrategyStateful

    // StrategyHybrid: Stateless JWT for normal requests, stateful
    // validation for high-security routes (admin, financial, etc.)
    StrategyHybrid
)

// SessionPolicy maps route patterns to strategies.
type SessionPolicy struct {
    Default        SessionStrategy
    HighSecurityRx map[string]SessionStrategy // regex → strategy
}

// Evaluate determines the strategy for a given request path.
func (p *SessionPolicy) Evaluate(path string) SessionStrategy {
    for rx, strategy := range p.HighSecurityRx {
        if matched, _ := regexp.MatchString(rx, path); matched {
            return strategy
        }
    }
    return p.Default
}
```

---

## 2. Sliding vs Absolute Timeout

### Definitions

- **Sliding (idle) timeout**: Session extends each time the user is active.
  If the user is inactive for the idle period, the session expires.
  Risk: a perpetually active user never gets logged out.
- **Absolute timeout**: Session expires after a fixed duration regardless of
  activity. Risk: user friction — being logged out mid-task.

### Recommended Policy: Sliding with Absolute Cap

The industry standard is a **dual-window** approach:

| Parameter | Typical Value | Example |
|---|---|---|
| Idle timeout | 15-60 min | User idle → session expires |
| Absolute timeout | 8-24 hours | Forced re-login regardless of activity |
| Refresh token TTL | 7-30 days | Refresh tokens outlive sessions for seamless re-auth |

```
Session Timeline:
  t=0h   Login → session created (abs cap = 8h, idle = 30m)
  t=0.5h Activity → idle window slides to t=1h
  t=1h   Activity → idle window slides to t=1.5h
  ...
  t=7.5h Activity → idle window WOULD slide to t=8h
         BUT abs cap (8h) reached → session expires at t=8h
```

### Go Implementation: Timeout Enforcement Middleware

GGID already implements a version of this in
`auth_service.go:CheckSessionTimeout()`. Below is a generalized middleware
pattern suitable for the API gateway:

```go
package middleware

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
)

// TimeoutConfig controls session expiration windows.
type TimeoutConfig struct {
    IdleTimeout     time.Duration // e.g. 30 * time.Minute
    AbsoluteTimeout time.Duration // e.g. 8 * time.Hour
}

// TimeoutEnforcer validates sessions against idle and absolute timeouts.
type TimeoutEnforcer struct {
    rdb *redis.Client
    cfg TimeoutConfig
}

func NewTimeoutEnforcer(rdb *redis.Client, cfg TimeoutConfig) *TimeoutEnforcer {
    return &TimeoutEnforcer{rdb: rdb, cfg: cfg}
}

// Check validates that a session is within both timeout windows.
// On success, updates the last-activity timestamp (sliding window).
// Returns an error if the session has expired.
func (te *TimeoutEnforcer) Check(ctx context.Context, sessionID, createdAtStr string) error {
    // 1. Absolute timeout check
    createdAt, err := time.Parse(time.RFC3339, createdAtStr)
    if err != nil {
        return fmt.Errorf("invalid session timestamp")
    }
    if te.cfg.AbsoluteTimeout > 0 && time.Since(createdAt) > te.cfg.AbsoluteTimeout {
        return ErrSessionAbsoluteExpired
    }

    // 2. Idle (sliding) timeout check
    if te.cfg.IdleTimeout > 0 {
        activityKey := fmt.Sprintf("ggid:session_activity:%s", sessionID)
        lastActiveStr, err := te.rdb.Get(ctx, activityKey).Result()
        if err == nil {
            lastActive, _ := time.Parse(time.RFC3339, lastActiveStr)
            if time.Since(lastActive) > te.cfg.IdleTimeout {
                return ErrSessionIdleExpired
            }
        }
        // Slide the window: update last-activity to now.
        te.rdb.Set(ctx, activityKey, time.Now().Format(time.RFC3339), te.cfg.IdleTimeout)
    }

    return nil
}

// Middleware wraps an HTTP handler with session timeout enforcement.
// Expects session_id and session_created_at in the request context
// (set by JWTAuth middleware).
func (te *TimeoutEnforcer) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        sessionID, _ := r.Context().Value(SessionIDKey).(string)
        createdAt, _ := r.Context().Value(SessionCreatedAtKey).(string)

        // Skip if no session (public endpoints or pure JWT without session binding)
        if sessionID == "" {
            next.ServeHTTP(w, r)
            return
        }

        if err := te.Check(r.Context(), sessionID, createdAt); err != nil {
            writeSessionError(w, err.Error())
            return
        }

        next.ServeHTTP(w, r)
    })
}

var ErrSessionAbsoluteExpired = fmt.Errorf("session exceeded absolute timeout")
var ErrSessionIdleExpired = fmt.Errorf("session expired due to inactivity")

type sessionCreatedAtKey string

var SessionCreatedAtKey sessionCreatedAtKey = "session_created_at"
```

---

## 3. Concurrent Session Limits

### Purpose

Limiting concurrent sessions per user:
- Prevents credential sharing across multiple people.
- Limits blast radius of compromised credentials.
- Enables "sign out everywhere" semantics.

### Design

```
Login Flow:
  1. Authenticate user credentials
  2. Create new session S_new
  3. Query active session count for user: count = SELECT COUNT(*) ...
  4. If count > MaxSessions:
     a. Find the oldest active session S_oldest
     b. Revoke S_oldest (DB + Redis)
     c. Revoke all refresh tokens for S_oldest
     d. Publish "session_evicted" event to NATS
     e. Send notification to user: "Session signed out on {device}"
  5. Return tokens for S_new
```

### Go Implementation: Concurrent Session Manager with Redis

```go
package session

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
)

// ConcurrentSessionManager enforces max concurrent sessions per user.
type ConcurrentSessionManager struct {
    rdb         *redis.Client
    maxSessions int
}

func NewConcurrentSessionManager(rdb *redis.Client, maxSessions int) *ConcurrentSessionManager {
    return &ConcurrentSessionManager{rdb: rdb, maxSessions: maxSessions}
}

// Register adds a new session to the user's active set and evicts
// the oldest session if the limit is exceeded.
// Returns the evicted session IDs (if any) for notification.
func (csm *ConcurrentSessionManager) Register(
    ctx context.Context,
    tenantID, userID, sessionID uuid.UUID,
    deviceInfo map[string]string,
) ([]uuid.UUID, error) {
    userKey := fmt.Sprintf("ggid:sessions:%s:%s", tenantID, userID)
    now := time.Now().Unix()

    // Store session metadata: score = creation timestamp, member = sessionID
    sessionMeta := fmt.Sprintf("%s|%d|%s", sessionID, now, deviceInfo["browser"])
    pipe := csm.rdb.TxPipeline()
    pipe.ZAdd(ctx, userKey, redis.Z{Score: float64(now), Member: sessionMeta})
    pipe.Expire(ctx, userKey, 30*24*time.Hour)
    if _, err := pipe.Exec(ctx); err != nil {
        return nil, fmt.Errorf("register session: %w", err)
    }

    // Count active sessions
    count, err := csm.rdb.ZCard(ctx, userKey).Result()
    if err != nil {
        return nil, err
    }

    var evicted []uuid.UUID
    // Evict oldest sessions if over limit
    for int(count) > csm.maxSessions {
        // Get the oldest entry (lowest score)
        oldest, err := csm.rdb.ZRangeWithScores(ctx, userKey, 0, 0).Result()
        if err != nil || len(oldest) == 0 {
            break
        }
        oldestMeta, ok := oldest[0].Member.(string)
        if !ok {
            break
        }
        // Parse session ID from "sessionID|timestamp|browser"
        evictedID := uuid.MustParse(extractSessionID(oldestMeta))
        evicted = append(evicted, evictedID)

        // Remove from the sorted set
        csm.rdb.ZRem(ctx, userKey, oldestMeta)
        // Delete session data
        csm.rdb.Del(ctx, fmt.Sprintf("ggid:session:%s", evictedID))
        count--
    }

    return evicted, nil
}

// Unregister removes a session from the user's active set (on logout).
func (csm *ConcurrentSessionManager) Unregister(ctx context.Context, tenantID, userID, sessionID uuid.UUID) {
    userKey := fmt.Sprintf("ggid:sessions:%s:%s", tenantID, userID)
    // ZRem needs exact member match — iterate to find matching prefix
    members, _ := csm.rdb.ZRange(ctx, userKey, 0, -1).Result()
    for _, m := range members {
        if len(m) >= 36 && m[:36] == sessionID.String() {
            csm.rdb.ZRem(ctx, userKey, m)
            break
        }
    }
}

// ActiveCount returns the number of concurrent active sessions.
func (csm *ConcurrentSessionManager) ActiveCount(ctx context.Context, tenantID, userID uuid.UUID) (int, error) {
    userKey := fmt.Sprintf("ggid:sessions:%s:%s", tenantID, userID)
    count, err := csm.rdb.ZCard(ctx, userKey).Result()
    return int(count), err
}

func extractSessionID(meta string) string {
    for i, c := range meta {
        if c == '|' {
            return meta[:i]
        }
    }
    return meta
}
```

### Notification on New Device Login

```go
// NotifyNewDevice sends an alert when a login occurs from an unrecognized device.
func (csm *ConcurrentSessionManager) NotifyNewDevice(
    ctx context.Context,
    rdb *redis.Client,
    tenantID, userID uuid.UUID,
    deviceInfo, ip, location string,
) error {
    // Check if this device is recognized (seen in the last 30 days)
    fp := deviceFingerprint(deviceInfo, ip)
    knownKey := fmt.Sprintf("ggid:known_devices:%s:%s:%s", tenantID, userID, fp)

    if rdb.Exists(ctx, knownKey).Val() > 0 {
        // Known device — refresh TTL
        rdb.Expire(ctx, knownKey, 30*24*time.Hour)
        return nil
    }

    // New device — send notification
    event := map[string]any{
        "type":       "new_device_login",
        "user_id":    userID.String(),
        "device":     deviceInfo,
        "ip":         ip,
        "location":   location,
        "timestamp":  time.Now().UTC(),
    }
    // Publish to NATS for notification service to consume
    // (implementation depends on NATS client setup)

    // Mark device as known
    rdb.Set(ctx, knownKey, "1", 30*24*time.Hour)
    return nil
}
```

---

## 4. Session Token Storage Strategies

### Browser Storage Options

| Storage | Accessible via JS | Sent with requests | Size limit | Recommended for session tokens |
|---|---|---|---|---|
| `localStorage` | Yes (XSS risk) | No (manual) | 5-10 MB | No |
| `sessionStorage` | Yes (XSS risk) | No (manual) | 5-10 MB | No |
| `HttpOnly` cookie | No | Yes (automatic) | 4 KB | **Yes** |
| `Secure` cookie | No | Yes (HTTPS only) | 4 KB | **Yes** |
| `SameSite` cookie | No | Conditional | 4 KB | **Yes** |

### Why HttpOnly + Secure + SameSite Cookies?

1. **HttpOnly**: JavaScript cannot read the cookie, preventing XSS-based token theft.
   This is the single most important defense against session hijacking via XSS.
2. **Secure**: Cookie is only transmitted over HTTPS, preventing MITM attacks.
3. **SameSite=Lax/Strict**: Prevents CSRF attacks by restricting cross-origin cookie
   transmission.

### CSRF Implications of Cookie-Based Sessions

When using cookies for session management, CSRF is a concern because browsers
automatically include cookies in cross-origin requests. Mitigations:

- **SameSite=Strict**: Most restrictive. Cookies never sent cross-origin.
  Breaks SSO flows where the user arrives from a different domain.
- **SameSite=Lax**: Cookies sent on top-level navigations (GET). Adequate for most
  applications. POST/PUT/DELETE are protected.
- **Double-submit cookie**: Include a CSRF token in both a cookie and a request header.
  Server validates that they match.

### Go Code: Session Cookie Management

```go
package session

import (
    "crypto/rand"
    "encoding/hex"
    "net/http"
    "time"
)

// CookieConfig controls session cookie attributes.
type CookieConfig struct {
    Name     string        // e.g. "ggid_session"
    Domain   string        // e.g. ".example.com" for cross-subdomain
    Path     string        // typically "/"
    Secure   bool          // true in production (HTTPS only)
    HTTPOnly bool          // always true for session tokens
    SameSite http.SameSite // Lax, Strict, or None
    MaxAge   time.Duration // cookie max-age
}

// SetSessionCookie sets an HttpOnly, Secure, SameSite cookie.
func SetSessionCookie(w http.ResponseWriter, cfg CookieConfig, token string) {
    http.SetCookie(w, &http.Cookie{
        Name:     cfg.Name,
        Value:    token,
        Domain:   cfg.Domain,
        Path:     cfg.Path,
        Secure:   cfg.Secure,
        HttpOnly: cfg.HTTPOnly,
        SameSite: cfg.SameSite,
        MaxAge:   int(cfg.MaxAge.Seconds()),
        Expires:  time.Now().Add(cfg.MaxAge),
    })
}

// ClearSessionCookie removes a session cookie by setting MaxAge to -1.
func ClearSessionCookie(w http.ResponseWriter, cfg CookieConfig) {
    http.SetCookie(w, &http.Cookie{
        Name:     cfg.Name,
        Value:    "",
        Domain:   cfg.Domain,
        Path:     cfg.Path,
        Secure:   cfg.Secure,
        HttpOnly: cfg.HTTPOnly,
        SameSite: cfg.SameSite,
        MaxAge:   -1,
        Expires:  time.Unix(0, 0),
    })
}

// DefaultCookieConfig returns a production-ready cookie configuration.
func DefaultCookieConfig() CookieConfig {
    return CookieConfig{
        Name:     "ggid_session",
        Path:     "/",
        Secure:   true,
        HTTPOnly: true,
        SameSite: http.SameSiteLaxMode,
        MaxAge:   8 * time.Hour,
    }
}

// GenerateCSRFToken creates a random 32-byte hex token for double-submit CSRF.
func GenerateCSRFToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

// ValidateCSRFToken compares the request header token against the cookie token.
func ValidateCSRFToken(r *http.Request, cookieName, headerName string) bool {
    cookie, err := r.Cookie(cookieName)
    if err != nil {
        return false
    }
    headerToken := r.Header.Get(headerName)
    if headerToken == "" || cookie.Value == "" {
        return false
    }
    // Constant-time comparison to prevent timing attacks
    return subtleEqual(cookie.Value, headerToken)
}

func subtleEqual(a, b string) bool {
    if len(a) != len(b) {
        return false
    }
    var result byte
    for i := 0; i < len(a); i++ {
        result |= a[i] ^ b[i]
    }
    return result == 0
}
```

### GGID-Specific Note

GGID currently returns tokens as JSON in the response body (`TokenSet.AccessToken`,
`TokenSet.RefreshToken`), relying on the client to store them. For browser-based
clients (the Admin Console), switching to HttpOnly cookie delivery for the refresh
token would significantly improve security posture. The access token can remain
in-memory (JavaScript variable) since it is short-lived.

---

## 5. Session Revocation Architecture

### The Core Problem with JWT

JWTs are stateless — once issued, they are valid until `exp`. There is no "delete"
operation. This creates a window where a revoked user's JWT remains usable for up
to `access_token_ttl` (15 minutes in GGID's default config).

### Revocation Strategies

| Strategy | Latency | Complexity | Scalability | Notes |
|---|---|---|---|---|
| Redis jti blacklist | ~1ms | Low | Good (per-request lookup) | Add ~0.5ms to each request |
| NATS event broadcast | ~10-50ms | Medium | Excellent (push model) | Best for multi-node |
| Bloom filter (Cuckoo) | <0.1ms | High | Excellent (in-memory) | Probabilistic, false positives |
| Short TTL + accept gap | 0ms | Lowest | Best | Just use very short JWT (5 min) |
| Token introspection (RFC 7662) | ~5ms | Medium | Good | Centralized, always-authoritative |

### Redis jti Blacklist (Recommended for GGID)

Every JWT includes a unique `jti` (JWT ID). On revocation, add the `jti` to a Redis
set with TTL = remaining token lifetime. The gateway checks this set on each request.

```go
package revocation

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// RevocationService manages JWT revocation via a Redis blacklist.
type RevocationService struct {
    rdb *redis.Client
}

func NewRevocationService(rdb *redis.Client) *RevocationService {
    return &RevocationService{rdb: rdb}
}

// RevokeJWT adds a JWT's jti to the blacklist with TTL = remaining token lifetime.
func (rs *RevocationService) RevokeJWT(ctx context.Context, jti string, expiresAt time.Time) error {
    ttl := time.Until(expiresAt)
    if ttl <= 0 {
        return nil // already expired, no need to blacklist
    }
    key := fmt.Sprintf("ggid:jti_blacklist:%s", jti)
    return rs.rdb.Set(ctx, key, "revoked", ttl).Err()
}

// IsRevoked checks if a JWT's jti is in the blacklist.
func (rs *RevocationService) IsRevoked(ctx context.Context, jti string) bool {
    key := fmt.Sprintf("ggid:jti_blacklist:%s", jti)
    val, err := rs.rdb.Exists(ctx, key).Result()
    if err != nil {
        return false // fail open on Redis errors (don't block all traffic)
    }
    return val > 0
}

// RevokeUserSessions revokes all sessions for a user by:
// 1. Revoking all refresh tokens
// 2. Revoking all sessions in the DB
// 3. Publishing a revocation event to NATS for other services
func (rs *RevocationService) RevokeUserSessions(
    ctx context.Context,
    tenantID, userID string,
    publishFn func(subject string, data []byte) error,
) error {
    // The actual DB/Redis cleanup happens in AuthService.LogoutAll().
    // Here we broadcast the revocation event so that:
    // - Other gateway instances can update their local caches
    // - Downstream services (OAuth, Policy) can invalidate their session caches

    event := fmt.Sprintf(`{"tenant_id":"%s","user_id":"%s","timestamp":"%s","action":"revoke_all"}`,
        tenantID, userID, time.Now().UTC().Format(time.RFC3339))

    return publishFn("ggid.session.revoked", []byte(event))
}
```

### NATS Event Broadcast for Distributed Revocation

In a multi-node deployment, a revocation on one gateway instance must propagate
to all others. NATS provides pub/sub for this:

```go
package revocation

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
)

// RevocationEvent is broadcast over NATS when a session or JWT is revoked.
type RevocationEvent struct {
    Type      string    `json:"type"`       // "jwt_revoked", "session_revoked", "user_revoked"
    JTI       string    `json:"jti,omitempty"`
    SessionID string    `json:"session_id,omitempty"`
    UserID    string    `json:"user_id,omitempty"`
    TenantID  string    `json:"tenant_id"`
    Timestamp time.Time `json:"timestamp"`
}

// DistributedRevocationListener subscribes to NATS revocation events
// and updates the local Redis blacklist.
type DistributedRevocationListener struct {
    rs       *RevocationService
    subFn    func(subject string, handler func([]byte)) error
}

func NewDistributedRevocationListener(rs *RevocationService) *DistributedRevocationListener {
    return &DistributedRevocationListener{rs: rs}
}

// Start subscribes to the revocation event stream.
func (dl *DistributedRevocationListener) Start(subject string) error {
    return dl.subFn(subject, func(data []byte) {
        var event RevocationEvent
        if err := json.Unmarshal(data, &event); err != nil {
            log.Printf("revocation event parse error: %v", err)
            return
        }

        ctx := context.Background()
        switch event.Type {
        case "jwt_revoked":
            // Add to local blacklist with a conservative TTL (access token max lifetime)
            dl.rs.rdb.Set(ctx,
                fmt.Sprintf("ggid:jti_blacklist:%s", event.JTI),
                "revoked",
                15*time.Minute, // access token max TTL
            )
        case "session_revoked":
            dl.rs.rdb.Del(ctx, fmt.Sprintf("ggid:session:%s", event.SessionID))
        case "user_revoked":
            // Mark all sessions for user as revoked — downstream services
            // should check on next request
            dl.rs.rdb.Set(ctx,
                fmt.Sprintf("ggid:user_revoked:%s:%s", event.TenantID, event.UserID),
                event.Timestamp.Format(time.RFC3339),
                24*time.Hour,
            )
        }
    })
}
```

### Revocation Propagation Latency Targets

| Operation | Target Latency | Acceptable | Mechanism |
|---|---|---|---|
| Single JWT revocation | <100ms | <500ms | Redis SET (synchronous) |
| User-wide revocation | <500ms | <2s | Redis + NATS broadcast |
| Cross-system (CAEP/RISC) | <5s | <30s | CAEP SSE event feed |
| Gateway cache invalidation | <200ms | <1s | NATS subscriber |

### CAEP (Cross-Application Event Protocol)

For federated IAM deployments, the [CAEP/SSF](https://openid.net/sgf/) specification
enables cross-system session revocation. When a user is revoked in GGID, a CAEP
event can trigger revocation in all connected applications:

```json
{
  "iss": "https://auth.ggid.dev",
  "aud": "https://app.example.com",
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {
        "subject_type": "iss_sub",
        "iss": "https://auth.ggid.dev",
        "sub": "user-uuid-here"
      },
      "reason": "admin_revocation"
    }
  }
}
```

---

## 6. Session Analytics

### Metrics to Track

| Metric | Source | Use Case |
|---|---|---|
| Active sessions per user | Redis sorted set | Credential sharing detection |
| Active sessions per tenant | Redis SCAN | Capacity planning |
| Session duration distribution | Session created_at → revoked_at | UX tuning (are sessions too short/long?) |
| Concurrent geographic sessions | IP geolocation | Impossible travel detection |
| New device login rate | Device fingerprint registry | Security alerts |
| Session revocation rate | Revocation events | Incident response metrics |

### Go Code: Session Analytics with Prometheus Metrics

```go
package analytics

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/redis/go-redis/v9"
)

var (
    // SessionsGauge tracks the current number of active sessions per tenant.
    SessionsGauge = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ggid_active_sessions",
            Help: "Number of active sessions",
        },
        []string{"tenant_id"},
    )

    // SessionDuration tracks session lifetime from creation to revocation.
    SessionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "ggid_session_duration_seconds",
            Help:    "Session lifetime in seconds",
            Buckets: []float64{60, 300, 900, 1800, 3600, 7200, 21600, 43200, 86400},
        },
        []string{"tenant_id"},
    )

    // NewDeviceLogins tracks logins from previously unseen devices.
    NewDeviceLogins = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ggid_new_device_logins_total",
            Help: "Total logins from new devices",
        },
        []string{"tenant_id", "device_type"},
    )

    // ConcurrentGeoSessions tracks sessions from multiple countries.
    ConcurrentGeoSessions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ggid_concurrent_geo_sessions_total",
            Help: "Sessions with concurrent activity from different countries",
        },
        []string{"tenant_id"},
    )
)

// SessionAnalytics collects and exposes session metrics.
type SessionAnalytics struct {
    rdb *redis.Client
}

func NewSessionAnalytics(rdb *redis.Client) *SessionAnalytics {
    return &SessionAnalytics{rdb: rdb}
}

// RecordSessionEnd records session duration when a session is revoked or expires.
func (sa *SessionAnalytics) RecordSessionEnd(tenantID uuid.UUID, createdAt, endedAt time.Time) {
    duration := endedAt.Sub(createdAt).Seconds()
    SessionDuration.WithLabelValues(tenantID.String()).Observe(duration)
}

// UpdateActiveCount refreshes the gauge for active sessions per tenant.
// Intended to be called by a periodic background job.
func (sa *SessionAnalytics) UpdateActiveCount(ctx context.Context, tenantID uuid.UUID) error {
    // Count active sessions across all users in the tenant.
    // This uses a Redis SCAN pattern over ggid:session:* keys.
    var cursor uint64
    count := 0
    pattern := fmt.Sprintf("ggid:sessions:%s:*", tenantID)
    for {
        keys, next, err := sa.rdb.Scan(ctx, cursor, pattern, 100).Result()
        if err != nil {
            return err
        }
        for _, key := range keys {
            n, _ := sa.rdb.ZCard(ctx, key).Result()
            count += int(n)
        }
        cursor = next
        if cursor == 0 {
            break
        }
    }
    SessionsGauge.WithLabelValues(tenantID.String()).Set(float64(count))
    return nil
}

// DetectImpossibleTravel flags sessions where the same user has concurrent
// sessions from geographically distant locations within an impossible timeframe.
func (sa *SessionAnalytics) DetectImpossibleTravel(
    ctx context.Context,
    tenantID, userID uuid.UUID,
    currentIP, currentCountry string,
    knownCountries map[string]time.Time,
) bool {
    now := time.Now()
    for country, lastSeen := range knownCountries {
        if country == currentCountry {
            continue
        }
        // If the user was seen in another country <2 hours ago and the
        // distance implies >1000 km/h travel, flag as impossible.
        timeDiff := now.Sub(lastSeen)
        if timeDiff < 2*time.Hour {
            ConcurrentGeoSessions.WithLabelValues(tenantID.String()).Inc()
            return true
        }
    }
    return false
}
```

---

## 7. Device-Bound Sessions

### Concept

Binding a session to a device fingerprint means that even if an attacker steals
the JWT, they cannot use it from a different device. The fingerprint mismatch
triggers step-up authentication.

### Fingerprint Signals

| Signal | Stability | Privacy | Collection method |
|---|---|---|---|
| User-Agent | Low (changes on browser update) | Low | HTTP header |
| IP address | Low (changes on network switch) | Medium | TCP connection |
| IP range (/24) | Medium | Low | Subnet extraction |
| TLS channel ID | High (session-level) | Low | TLS extension |
| FingerprintJS / client-side | High | Medium | JavaScript canvas/WebGL hash |
| Accept-Language | Medium | Low | HTTP header |

### Recommended Composite Fingerprint

```go
// A composite fingerprint uses multiple signals with partial matching:
// If 2 of 3 signals match, consider the device "same".
type DeviceFingerprint struct {
    UserAgentHash string // SHA-256(user-agent) first 16 bytes
    IPRange       string // /24 subnet (e.g. "192.168.1.0/24")
    AcceptLang    string // Accept-Language header value
}
```

### Go Code: Device-Bound Session Validation

```go
package session

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "net"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
)

// DeviceFingerprint captures stable device signals.
type DeviceFingerprint struct {
    UserAgentHash string
    IPRange       string
    AcceptLang    string
}

// ComputeFingerprint generates a DeviceFingerprint from request signals.
func ComputeFingerprint(userAgent, clientIP, acceptLang string) DeviceFingerprint {
    h := sha256.Sum256([]byte(userAgent))
    uaHash := hex.EncodeToString(h[:16])

    // Extract /24 subnet
    ip := net.ParseIP(clientIP)
    ipRange := clientIP
    if ip != nil {
        if ip4 := ip.To4(); ip4 != nil {
            ipRange = fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
        }
    }

    return DeviceFingerprint{
        UserAgentHash: uaHash,
        IPRange:       ipRange,
        AcceptLang:    strings.ToLower(acceptLang),
    }
}

// String serializes the fingerprint for storage.
func (df DeviceFingerprint) String() string {
    return fmt.Sprintf("%s|%s|%s", df.UserAgentHash, df.IPRange, df.AcceptLang)
}

// Matches returns true if at least 2 of 3 signals match (partial match policy).
func (df DeviceFingerprint) Matches(other DeviceFingerprint) bool {
    score := 0
    if df.UserAgentHash == other.UserAgentHash {
        score++
    }
    if df.IPRange == other.IPRange {
        score++
    }
    if df.AcceptLang == other.AcceptLang {
        score++
    }
    return score >= 2
}

// DeviceBoundSessionValidator validates that requests come from the
// device that originally authenticated.
type DeviceBoundSessionValidator struct {
    rdb *redis.Client
}

func NewDeviceBoundSessionValidator(rdb *redis.Client) *DeviceBoundSessionValidator {
    return &DeviceBoundSessionValidator{rdb: rdb}
}

// BindOnLogin stores the device fingerprint at session creation time.
func (dv *DeviceBoundSessionValidator) BindOnLogin(
    ctx context.Context,
    sessionID uuid.UUID,
    fp DeviceFingerprint,
) error {
    key := fmt.Sprintf("ggid:device_fp:%s", sessionID)
    return dv.rdb.Set(ctx, key, fp.String(), 24*time.Hour).Err()
}

// Validate checks if the current request matches the bound device.
// Returns:
//   - true if device matches (proceed normally)
//   - false if device changed (trigger step-up auth)
//   - error if Redis lookup fails (fail open)
func (dv *DeviceBoundSessionValidator) Validate(
    ctx context.Context,
    sessionID uuid.UUID,
    currentFP DeviceFingerprint,
) (bool, error) {
    key := fmt.Sprintf("ggid:device_fp:%s", sessionID)
    stored, err := dv.rdb.Get(ctx, key).Result()
    if err != nil {
        if err == redis.Nil {
            // No fingerprint bound — allow (backward compatibility)
            return true, nil
        }
        return true, err // fail open on Redis errors
    }

    // Parse stored fingerprint
    parts := strings.SplitN(stored, "|", 3)
    if len(parts) != 3 {
        return true, nil
    }
    boundFP := DeviceFingerprint{
        UserAgentHash: parts[0],
        IPRange:       parts[1],
        AcceptLang:    parts[2],
    }

    // Partial match: allow minor changes (IP switch, browser update)
    // but require at least 2 of 3 signals to match.
    if !currentFP.Matches(boundFP) {
        // Device mismatch — potential token theft
        // Trigger step-up auth: require MFA or password re-entry
        return false, nil
    }

    return true, nil
}

// StepUpRequired is returned when device validation fails.
type StepUpRequired struct {
    Reason string
}

func (e StepUpRequired) Error() string {
    return fmt.Sprintf("step-up authentication required: %s", e.Reason)
}
```

### Token Theft Detection

When a device fingerprint mismatch is detected, the system should:
1. Immediately revoke the session and all associated tokens.
2. Notify the user that a suspicious login was detected.
3. Require re-authentication with MFA.

```go
// HandleDeviceMismatch is called when a request arrives from a device
// that doesn't match the bound fingerprint.
func HandleDeviceMismatch(
    ctx context.Context,
    sessionID uuid.UUID,
    revokeSession func(uuid.UUID) error,
    notifyUser func(string),
) {
    // Revoke the session immediately
    _ = revokeSession(sessionID)
    // Notify the user
    notifyUser("Suspicious activity detected on your account. Please log in again.")
}
```

---

## 8. GGID Session Management Gap Analysis

### Current Implementation Inventory

The following table maps each session management concern to what GGID currently
implements, based on review of `services/auth/`, `services/gateway/internal/middleware/`,
and `services/auth/internal/conf/conf.go`.

| Feature | Status | Location | Notes |
|---|---|---|---|
| JWT access token (RS256, 15min TTL) | **Implemented** | `token_service.go:IssueAccessToken` | Stateless, signature verified at gateway |
| Opaque refresh token (30d TTL, SHA-256 hashed) | **Implemented** | `token_service.go:IssueRefreshToken` | Stored in DB + Redis |
| Refresh token rotation with reuse detection | **Implemented** | `token_service.go:RotateRefreshToken` | Revoke entire chain on reuse (RFC 6749 10.4) |
| Session record (DB-backed, revocable) | **Implemented** | `session_repo.go`, `session_service.go` | Sessions table with token_hash, device_info, expires_at |
| Session revocation (single + all-for-user) | **Implemented** | `session_repo.go:Revoke`, `RevokeAllForUser` | DB-level `revoked_at` column |
| Absolute timeout config | **Implemented** | `conf.go:SessionTimeoutConfig.AbsoluteTimeout` | Default: 8h |
| Idle timeout config | **Implemented** | `conf.go:SessionTimeoutConfig.IdleTimeout` | Default: 30m |
| Session timeout enforcement | **Partial** | `auth_service.go:CheckSessionTimeout` | Logic exists but is NOT called from gateway middleware — gap |
| Max concurrent sessions config | **Implemented** | `conf.go:SessionTimeoutConfig.MaxSessions` | Default: 0 (unlimited) |
| Concurrent session limit enforcement | **Implemented** | `session_management.go:EnforceSessionLimit` | Evicts oldest sessions when over limit |
| Device fingerprint generation | **Implemented** | `session_management.go:GenerateDeviceFingerprint` | UA + IP hash |
| Device fingerprint binding | **Implemented** | `session_management.go:BindFingerprintToSession` | Redis `ggid:session_fp:{id}` |
| Device fingerprint verification | **Implemented** | `session_management.go:VerifySessionFingerprint` | Redis lookup, exact match |
| Device tracking (Redis) | **Implemented** | `device_tracking.go:TrackDevice` | Per-user Redis hash |
| Force logout (admin) | **Implemented** | `session_management.go:ForceLogout` | Revokes all sessions + refresh tokens |
| Logout all (user-initiated) | **Implemented** | `logout_all.go:LogoutAll` | Sessions + refresh tokens |
| Gateway session validation (Redis) | **Implemented** | `gateway/middleware/session.go` | Checks Redis key `ggid:session:{id}` |
| JTI replay tracking | **Implemented** | `gateway/middleware/jti_replay.go` | In-memory (not Redis — multi-node gap) |
| Trusted device (MFA bypass) | **Implemented** | `auth_service.go:RememberTrustedDevice` | 30-day TTL in Redis |
| HttpOnly cookie delivery | **Missing** | — | Tokens returned as JSON body; client manages storage |
| JWT jti blacklist (pre-expiry revocation) | **Missing** | — | No mechanism to revoke JWTs before `exp` |
| NATS revocation broadcast | **Missing** | — | Revocation is DB/Redis only; no pub/sub propagation |
| Sliding window enforcement at gateway | **Missing** | — | `CheckSessionTimeout` exists but is not wired into gateway pipeline |
| Session analytics (Prometheus metrics) | **Missing** | — | No session-related metrics exported |
| Impossible travel detection | **Missing** | — | No geo-IP session correlation |
| CAEP cross-system revocation | **Missing** | — | No SSE/CAEP event publisher |
| CSRF double-submit token | **Missing** | — | No CSRF protection (cookies not used, so lower priority) |

### Key Gaps

1. **CheckSessionTimeout is not wired into the gateway pipeline.** The function
   exists in `auth_service.go` but the gateway's `SessionMiddleware` does not call
   it. Sessions are validated only by Redis key existence — the idle/absolute
   timeout logic in `CheckSessionTimeout` is dead code from the gateway's perspective.

2. **JTI replay tracker is in-memory only.** `jti_replay.go` uses a `sync.Mutex`
   + `map[string]time.Time`. In a multi-node gateway deployment, each instance has
   its own tracker. A replayed token would be detected on one instance but accepted
   on another. The code comment acknowledges this: "In production, replace with
   Redis SETNX."

3. **No JWT revocation before `exp`.** There is no jti blacklist. Once a JWT is
   issued, it is valid for the full 15-minute TTL regardless of logout. The `Logout`
   function only revokes the refresh token.

4. **Concurrent session limit default is 0 (unlimited).** `MaxSessions` defaults to
   0, meaning credential sharing is not prevented by default.

5. **Device fingerprint verification uses exact match.** `VerifySessionFingerprint`
   does `stored == fingerprint` — a network change (IP changes → fingerprint changes)
   would fail, but the function returns `true` when no fingerprint is found (fail open).
   There is no partial-match policy or step-up auth trigger.

---

## 9. Gap Analysis & Recommendations

### Priority Action Items

| # | Action | Effort | Impact | Files Affected |
|---|---|---|---|---|
| 1 | **Wire `CheckSessionTimeout` into the gateway session middleware** so idle/absolute timeouts are actually enforced on every request. Currently the logic exists but is dead code from the gateway's perspective. | **S** (2h) | High — closes the "infinite session" gap | `gateway/internal/middleware/session.go` |
| 2 | **Replace in-memory JTI replay tracker with Redis SETNX** for multi-node consistency. Current `jti_replay.go` only works for single-node deployments. | **M** (4h) | High — prevents replay across gateway instances | `gateway/internal/middleware/jti_replay.go` |
| 3 | **Add JWT jti blacklist to the gateway pipeline.** On logout/revocation, add the JWT's jti to a Redis set with TTL = remaining token lifetime. Gateway checks this on each request. Eliminates the 15-minute revocation window. | **M** (1d) | High — enables pre-expiry JWT revocation | `gateway/internal/middleware/`, `auth/service/token_service.go` |
| 4 | **Add NATS revocation event broadcast** so revocations propagate to all gateway instances within seconds. Subscribe on gateway startup, publish from auth service. | **L** (2d) | Medium — multi-node revocation consistency | `auth/service/`, `gateway/internal/middleware/` |
| 5 | **Export Prometheus session metrics** (active sessions gauge, session duration histogram, new device login counter). Wire into the existing metrics endpoint. | **M** (4h) | Medium — observability for security operations | `auth/service/`, `gateway/internal/` |

### Effort Legend

- **S** = Small (<4 hours, <100 lines changed)
- **M** = Medium (4h-1d, 100-300 lines changed)
- **L** = Large (1-3d, 300+ lines, multiple packages)

### Summary

GGID has a solid foundation for session management: hybrid JWT+opaque refresh tokens,
rotation with reuse detection, DB-backed sessions with revocation, device tracking,
and timeout configuration. The primary gaps are:

1. **Wiring**: timeout enforcement exists but is not called from the gateway.
2. **Multi-node**: JTI replay tracking is in-memory only.
3. **Pre-expiry JWT revocation**: no jti blacklist.
4. **Observability**: no session metrics exported.
5. **Cross-system revocation**: no NATS broadcast or CAEP events.

These gaps are addressable incrementally — each is independently deployable and
does not require changes to the token format or session schema. The highest-ROI
item is #1 (wiring timeout enforcement), which closes a security gap with a
single middleware change.

---

*See also: `docs/research/session-management-design.md` for GGID's token lifecycle
and rotation details, `docs/research/session-fixation-prevention.md` for fixation
attack vectors, and `docs/research/openid-connect-session-management.md` for OIDC
back-channel session coordination.*
