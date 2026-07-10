# Zero Trust Architecture — Practical Implementation for GGID

> **Research Document** — Implementation patterns, Go code, and GGID-specific gap analysis.
>
> **Scope**: This document is implementation-focused. For the theoretical foundation
> (NIST SP 800-207, SPIFFE/SPIRE, BeyondCorp model, device posture frameworks, and the
> maturity model), see the companion doc:
> [`zero-trust-iam-patterns.md`](./zero-trust-iam-patterns.md) (913 lines).

---

## Table of Contents

1. [Microsegmentation for IAM Services](#1-microsegmentation-for-iam-services)
2. [Device Posture in Access Decisions](#2-device-posture-in-access-decisions)
3. [Continuous Authorization](#3-continuous-authorization)
4. [Least Privilege Enforcement](#4-least-privilege-enforcement)
5. [Identity-Aware Proxy Pattern](#5-identity-aware-proxy-pattern)
6. [Zero Trust Network Architecture for GGID](#6-zero-trust-network-architecture-for-ggid)
7. [GGID Zero Trust Gap Analysis](#7-ggid-zero-trust-gap-analysis)
8. [Implementation Roadmap](#8-implementation-roadmap)

---

## 1. Microsegmentation for IAM Services

GGID runs seven microservices (gateway, identity, auth, oauth, policy, org, audit) backed
by PostgreSQL, Redis, NATS JetStream, and OpenLDAP. In a perimeter-based model, all
services sit in one flat network; if an attacker breaches any service, they have east-west
access to all others. Microsegmentation divides the network into **trust zones** so that
lateral movement is contained.

### Trust Zone Architecture

```
                      ┌──────────────────────────────────────────────────────────┐
                      │                    INTERNET                              │
                      │              (untrusted — zero implicit trust)           │
                      └────────────────────────────┬─────────────────────────────┘
                                   TLS termination
                                   │
                      ┌────────────▼───────────────────────────────┐
                      │           ZONE 0: DMZ (Edge)                │
                      │  ┌──────────────────────────────────────┐   │
                      │  │         GATEWAY (:8080)              │   │
                      │  │  • JWT verification per request       │   │
                      │  │  • Rate limiting / WAF                │   │
                      │  │  • TLS 1.3 termination                │   │
                      │  │  • NO direct DB access                │   │
                      │  └──────────┬───────────┬───────────────┘   │
                      └─────────────┼───────────┼───────────────────┘
                                    │ mTLS      │ mTLS
                      ┌─────────────▼───────────▼───────────────────┐
                      │      ZONE 1: Auth Services (internal)       │
                      │  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
                      │  │  AUTH    │  │  OAUTH   │  │ IDENTITY  │  │
                      │  │ (:9001)  │  │ (:9005)  │  │ (:8081)   │  │
                      │  │          │  │          │  │           │  │
                      │  │ Issues   │  │ OIDC/    │  │ User/Group│  │
                      │  │ JWTs     │  │ SAML/    │  │ CRUD      │  │
                      │  │          │  │ Social   │  │           │  │
                      │  └────┬─────┘  └────┬─────┘  └─────┬─────┘  │
                      └───────┼─────────────┼──────────────┼────────┘
                              │             │              │ mTLS
                      ┌───────▼─────────────▼──────────────▼────────┐
                      │     ZONE 2: Policy & Org (internal)          │
                      │  ┌──────────┐           ┌──────────┐         │
                      │  │ POLICY   │ mTLS      │   ORG    │         │
                      │  │ (:8070)  │◄─────────►│ (:8071)  │         │
                      │  │ RBAC/ABAC│           │ Org tree │         │
                      │  └──────────┘           └──────────┘         │
                      └───────┬─────────────────────────┬────────────┘
                              │                         │
                      ┌───────▼─────────────────────────▼────────────┐
                      │        ZONE 3: Audit & Telemetry              │
                      │  ┌──────────┐        ┌─────────────────────┐  │
                      │  │  AUDIT   │──NATS─►│  JetStream          │  │
                      │  │ (:8072)  │        │  (event bus)        │  │
                      │  │ Append-  │        │  Risk scoring cons. │  │
                      │  │ only DB  │        │                     │  │
                      │  └──────────┘        └─────────────────────┘  │
                      └──────────────────────────────────────────────┘
                              │                         │
                      ┌───────▼─────────────────────────▼────────────┐
                      │     ZONE 4: Data Layer (most restricted)      │
                      │  ┌──────────┐  ┌─────────┐  ┌──────────────┐  │
                      │  │PostgreSQL│  │  Redis  │  │  OpenLDAP    │  │
                      │  │  RLS on  │  │ session │  │  user store  │  │
                      │  │tenant_id │  │  cache  │  │              │  │
                      │  └──────────┘  └─────────┘  └──────────────┘  │
                      └──────────────────────────────────────────────┘
```

### Zone Policies

| Zone | Services | Inbound From | Outbound To | Network Policy |
|------|----------|-------------|-------------|----------------|
| **0 — DMZ** | Gateway | Internet (443) | Zone 1, Zone 2 | Deny all to Zone 3/4 |
| **1 — Auth** | Auth, OAuth, Identity | Zone 0 only | Zone 4 (DB) | Deny Zone 2→1 except Gateway |
| **2 — Policy/Org** | Policy, Org | Zone 0, Zone 1 | Zone 4 (DB) | No direct internet egress |
| **3 — Audit** | Audit, NATS | Zone 0, 1, 2 (publish) | Zone 4 (DB) | Append-only writes |
| **4 — Data** | PostgreSQL, Redis, LDAP | Zone 1, 2 only | None | No outbound connections |

### East-West mTLS

All inter-service communication must use mTLS. Today GGID's gateway proxy uses
`httputil.NewSingleHostReverseProxy` with a plain `http.Transport` — no TLS client config:

```go
// services/gateway/internal/router/router.go (CURRENT — no mTLS)
proxy.Transport = &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    // No TLSClientConfig — traffic to backends is plaintext HTTP
}
```

To enable mTLS, add a `tls.Config` to the transport:

```go
// PROPOSED: services/gateway/internal/router/router.go
import "crypto/tls"

proxy.Transport = &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    TLSClientConfig: &tls.Config{
        MinVersion:         tls.VersionTLS13,
        GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
            return gw.serviceCert, nil // SPIFFE SVID or internal CA cert
        },
        RootCAs:            gw.internalCA,
        ServerName:         parsed.Hostname(),
        // Verify peer against expected service identity
        VerifyPeerCertificate: gw.verifyServiceIdentity(parsed.Hostname()),
    },
}
```

### Service Mesh Considerations

A service mesh (Istio, Linkerd, or Consul Connect) can enforce mTLS transparently
without modifying GGID application code. With a mesh:

- **Sidecar proxies** handle mTLS negotiation per connection.
- **NetworkPolicies** (Kubernetes) or **security groups** (VM) enforce zone boundaries at L3/L4.
- **AuthorizationPolicies** (Istio CRD) restrict which services can call which:
  ```yaml
  # Allow only gateway to call auth service
  apiVersion: security.istio.io/v1
  kind: AuthorizationPolicy
  metadata:
    name: auth-service-policy
    namespace: ggid
  spec:
    selector:
      matchLabels: { app: auth }
    action: ALLOW
    rules:
      - from:
          - source:
              principals: ["cluster.local/ns/ggid/sa/gateway"]
  ```

GGID's OAuth service already implements RFC 8705 mTLS sender-constrained tokens
(`services/oauth/internal/service/jar_mtls.go`), which validates client certificate
thumbprints against JWT `cnf.x5t#S256` claims. This is a strong foundation for
extending mTLS to all service-to-service calls.

---

## 2. Device Posture in Access Decisions

> The companion doc covers device posture **frameworks** (see section 5 of
> `zero-trust-iam-patterns.md`). This section provides the **implementation**:
> Go code for device trust scoring and conditional access integration.

### Signal Sources

| Signal | Source | Weight | Example |
|--------|--------|--------|---------|
| Disk encryption | MDM API (Jamf/Intune) | 20 | FileVault on, BitLocker on |
| OS version | MDM API | 15 | macOS 14.5+, Windows 11 23H2+ |
| Screen lock | MDM API | 10 | Auto-lock ≤ 5 min |
| Managed device | Enrollment status | 25 | Corporate-issued, supervised |
| EDR agent running | Endpoint agent | 15 | CrowdStrike, SentinelOne active |
| Rooted/jailbroken | Device attestation | -50 | Knox attestation fail |
| Unknown location | GeoIP + history | -20 | First login from new country |

### Device Trust Scoring Engine

```go
// pkg/devicetrust/score.go
package devicetrust

import (
    "context"
    "time"
)

// PostureSignal represents a single device posture attribute.
type PostureSignal struct {
    Key       string      `json:"key"`       // "disk_encrypted", "os_version", etc.
    Value     any         `json:"value"`     // true, "14.5", 300, ...
    Timestamp time.Time   `json:"timestamp"` // when MDM last reported this
    Source    string      `json:"source"`    // "jamf", "intune", "attestation"
}

// DevicePosture aggregates all known signals for a device.
type DevicePosture struct {
    DeviceID  string           `json:"device_id"`
    Managed   bool             `json:"managed"`
    Signals   []PostureSignal  `json:"signals"`
    UpdatedAt time.Time        `json:"updated_at"`
}

// ScoringRule maps a signal condition to a point delta.
type ScoringRule struct {
    SignalKey string
    Apply     func(value any, dp *DevicePosture) int
}

// DefaultRules defines the standard scoring weights.
var DefaultRules = []ScoringRule{
    {"disk_encrypted", func(v any, _ *DevicePosture) int {
        if b, ok := v.(bool); ok && b { return 20 }
        return -15
    }},
    {"managed", func(v any, _ *DevicePosture) int {
        if b, ok := v.(bool); ok && b { return 25 }
        return 0
    }},
    {"os_version", func(v any, _ *DevicePosture) int {
        ver, ok := v.(string)
        if !ok { return -10 }
        if ver == "" { return -10 }
        // Simplified — production would use semver comparison
        return 10
    }},
    {"screen_lock_seconds", func(v any, _ *DevicePosture) int {
        secs, ok := v.(int)
        if !ok { return 0 }
        if secs > 0 && secs <= 300 { return 10 } // auto-lock ≤ 5 min
        return -5
    }},
    {"edr_active", func(v any, _ *DevicePosture) int {
        if b, ok := v.(bool); ok && b { return 15 }
        return -20
    }},
    {"jailbroken", func(v any, _ *DevicePosture) int {
        if b, ok := v.(bool); ok && b { return -50 }
        return 0
    }},
}

// Score computes a 0–100 trust score for a device based on its posture signals.
// A score of 0 means no posture data; negative components can reduce the score.
func (dp *DevicePosture) Score(rules []ScoringRule) int {
    if dp == nil || len(dp.Signals) == 0 {
        return 0
    }
    signalMap := make(map[string]any, len(dp.Signals))
    for _, s := range dp.Signals {
        signalMap[s.Key] = s.Value
    }

    score := 30 // baseline for known device
    for _, rule := range rules {
        if val, ok := signalMap[rule.SignalKey]; ok {
            score += rule.Apply(val, dp)
        }
    }

    // Stale signals (>24h old) reduce confidence
    for _, s := range dp.Signals {
        if time.Since(s.Timestamp) > 24*time.Hour {
            score -= 5
        }
    }

    if score > 100 { score = 100 }
    if score < 0   { score = 0 }
    return score
}

// AccessTier maps a trust score to a resource access tier.
const (
    TierBlocked   = 0  // score < 30
    TierLimited   = 1  // score 30–59: read-only, no sensitive data
    TierStandard  = 2  // score 60–79: standard app access
    TierElevated  = 3  // score 80–100: full access including admin
)

func ScoreToTier(score int) int {
    switch {
    case score < 30:  return TierBlocked
    case score < 60:  return TierLimited
    case score < 80:  return TierStandard
    default:          return TierElevated
    }
}
```

### Gateway Middleware Integration

```go
// services/gateway/internal/middleware/device_posture.go
package middleware

import (
    "context"
    "net/http"
)

type devicePostureKey string
const DevicePostureCtxKey devicePostureKey = "device_posture"

// DeviceStore retrieves posture for a given device ID.
type DeviceStore interface {
    Get(ctx context.Context, deviceID string) (*DevicePosture, error)
}

// RequiredTierForPath returns the minimum device tier needed for a path.
type TierResolver interface {
    Resolve(path, method string) int
}

// DevicePostureCheck enforces device trust before forwarding to backends.
func DevicePostureCheck(store DeviceStore, resolver TierResolver) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            requiredTier := resolver.Resolve(r.URL.Path, r.Method)
            if requiredTier == 0 {
                next.ServeHTTP(w, r) // no device requirement for this path
                return
            }

            deviceID := r.Header.Get("X-Device-ID")
            if deviceID == "" {
                // Unmanaged device — score 0, allow only if tier requirement is ≤ limited
                if requiredTier <= TierLimited {
                    ctx := context.WithValue(r.Context(), DevicePostureCtxKey, 0)
                    next.ServeHTTP(w, r.WithContext(ctx))
                    return
                }
                respondStepUp(w, "device enrollment required", "/api/v1/devices/enroll")
                return
            }

            posture, err := store.Get(r.Context(), deviceID)
            if err != nil || posture == nil {
                if requiredTier <= TierLimited {
                    next.ServeHTTP(w, r)
                    return
                }
                respondStepUp(w, "device posture unknown", "/api/v1/devices/attest")
                return
            }

            score := posture.Score(nil) // uses DefaultRules
            tier := ScoreToTier(score)
            if tier < requiredTier {
                respondStepUp(w, "insufficient device trust score", "/api/v1/devices/attest")
                return
            }

            ctx := context.WithValue(r.Context(), DevicePostureCtxKey, score)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func respondStepUp(w http.ResponseWriter, msg, challengeURL string) {
    w.Header().Set("WWW-Authenticate", `Device challenge="`+challengeURL+`"`)
    w.WriteHeader(http.StatusUnauthorized)
    w.Write([]byte(`{"error":"` + msg + `"}`))
}
```

---

## 3. Continuous Authorization

> The companion doc covers continuous authentication **concepts** (section 4).
> This section provides **per-request trust evaluation** and **CAEP event consumption**
> implementation.

Traditional IAM authenticates once (at login) and trusts the token until expiry. Zero
trust requires re-evaluating trust on every request, incorporating signals that may have
changed since the token was issued: session revocation, device posture degradation,
anomalous behavior, or admin-triggered lockout.

### Token Validation: Per-Request vs Cached

GGID's gateway currently validates JWT signatures per-request (good), but does not check
whether the **session** backing the token has been revoked. A stolen token remains valid
until its `exp` claim expires.

```go
// CURRENT: services/gateway/internal/middleware/middleware.go
// JWTAuth() validates signature + claims (exp/nbf/iss/aud) per request.
// BUT: no session revocation check, no risk evaluation.
```

### Session Revocation Cache

```go
// pkg/zta/revocation.go
package zta

import (
    "context"
    "time"
)

// RevocationStore checks whether a token/session has been revoked.
// Implementations: Redis-backed set, Bloom filter, or CAEP-fed cache.
type RevocationStore interface {
    IsRevoked(ctx context.Context, tokenHash string) bool
    Revoke(ctx context.Context, tokenHash string, ttl time.Duration) error
}

// CAEPEvent represents a Shared Signals Framework security event (RFC CAEP).
type CAEPEvent struct {
    EventType   string    `json:"event_type"` // "session-revoked", "credential-change"
    Subject     Subject   `json:"subject"`    // user or session identifier
    Timestamp   time.Time `json:"timestamp"`
    Reason      string    `json:"reason"`
}

type Subject struct {
    Format string `json:"format"` // "iss_sub", "urn:ietf:params:scim:schemas:core:2.0:User"
    Issuer string `json:"iss"`
    Value  string `json:"sub"` // user ID or session ID
}

// CAEPConsumer subscribes to a CAEP feed (SSE, webhook, or NATS) and
// updates the revocation store in real-time.
type CAEPConsumer struct {
    store RevocationStore
    subs  []<-chan CAEPEvent
}

func (c *CAEPConsumer) Start(ctx context.Context) {
    for _, ch := range c.subs {
        go func(events <-chan CAEPEvent) {
            for ev := range events {
                if ev.EventType == "session-revoked" || ev.EventType == "credential-change" {
                    _ = c.store.Revoke(ctx, ev.Subject.Value, 24*time.Hour)
                }
            }
        }(ch)
    }
}
```

### Per-Request Trust Evaluation

```go
// pkg/zta/trust_evaluator.go
package zta

import (
    "context"
    "sync"
    "time"
)

// RequestSignals collects identity, device, and behavioral signals for a request.
type RequestSignals struct {
    UserID          string
    SessionID       string
    ClientIP        string
    UserAgent       string
    GeoCountry      string
    DeviceScore     int  // from device posture check
    TokenAgeSeconds int
}

// RiskRule evaluates a signal and returns a risk delta (positive = riskier).
type RiskRule func(s RequestSignals) int

// DefaultRiskRules: baseline behavioral risk heuristics.
var DefaultRiskRules = []RiskRule{
    // Token age: older tokens are riskier
    func(s RequestSignals) int {
        if s.TokenAgeSeconds > 3600 { return 15 }  // > 1h old
        return 0
    },
    // New geo: first time from this country for this user
    func(s RequestSignals) int {
        if s.GeoCountry != "" && !seenCountry(s.UserID, s.GeoCountry) {
            return 25
        }
        return 0
    },
    // Low device score
    func(s RequestSignals) int {
        if s.DeviceScore < 50 { return 20 }
        return 0
    },
    // Anonymous user agent (automated tooling)
    func(s RequestSignals) int {
        if s.UserAgent == "" || s.UserAgent == "curl/7.0" { return 30 }
        return 0
    },
}

// SlidingWindowTracker tracks request frequency per user in a sliding window
// to detect brute-force or anomalous velocity.
type SlidingWindowTracker struct {
    mu      sync.Mutex
    windows map[string][]time.Time // user → request timestamps
    window  time.Duration
    limit   int
}

func NewSlidingWindowTracker(window time.Duration, limit int) *SlidingWindowTracker {
    return &SlidingWindowTracker{
        windows: make(map[string][]time.Time),
        window:  window,
        limit:   limit,
    }
}

// Record adds a request timestamp and returns true if the rate limit is exceeded.
func (t *SlidingWindowTracker) Record(userID string) bool {
    t.mu.Lock()
    defer t.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-t.window)
    times := t.windows[userID]
    // Prune old entries
    kept := times[:0]
    for _, ts := range times {
        if ts.After(cutoff) { kept = append(kept, ts) }
    }
    kept = append(kept, now)
    t.windows[userID] = kept

    return len(kept) > t.limit
}

// TrustDecision is the output of per-request evaluation.
type TrustDecision struct {
    Allow      bool
    RiskScore  int
    RequireMFA bool
    Reason     string
}

// Evaluate computes a trust decision for an incoming request.
// Risk score 0-100: 0 = no risk, 100 = maximum risk.
func Evaluate(ctx context.Context, store RevocationStore,
    tracker *SlidingWindowTracker, s RequestSignals) TrustDecision {

    // 1. Hard deny: revoked session
    if store != nil && store.IsRevoked(ctx, s.SessionID) {
        return TrustDecision{Allow: false, RiskScore: 100, Reason: "session revoked"}
    }

    // 2. Hard deny: sliding window rate limit exceeded
    if tracker != nil && tracker.Record(s.UserID) {
        return TrustDecision{Allow: false, RiskScore: 100, Reason: "rate limit exceeded"}
    }

    // 3. Compute risk score from behavioral rules
    risk := 0
    for _, rule := range DefaultRiskRules {
        risk += rule(s)
    }
    if risk > 100 { risk = 100 }

    // 4. Decide: allow, step-up MFA, or deny
    switch {
    case risk >= 60:
        return TrustDecision{Allow: false, RiskScore: risk, Reason: "risk too high"}
    case risk >= 30:
        return TrustDecision{Allow: true, RiskScore: risk, RequireMFA: true,
            Reason: "step-up MFA required"}
    default:
        return TrustDecision{Allow: true, RiskScore: risk, Reason: "ok"}
    }
}

// seenCountry is a placeholder — production would query the audit service.
func seenCountry(userID, country string) bool { return true }
```

### Gateway Integration

```go
// PROPOSED: Wire trust evaluation into the gateway middleware chain
// Order: RequestID → RateLimit → JWTAuth → DevicePosture → TrustEvaluation → Proxy
handler := middleware.RequestID()(
    middleware.RateLimit(limiter)(
        middleware.JWTAuth(jwks, true, issuer, audience)(
            middleware.DevicePostureCheck(deviceStore, tierResolver)(
                TrustEvaluationMiddleware(revStore, tracker)(
                    gatewayHandler,
                ),
            ),
        ),
    ),
)
```

---

## 4. Least Privilege Enforcement

### JIT vs Standing Privileges

**Standing privileges** are roles assigned permanently (no expiry). **Just-in-time (JIT)**
privileges are granted for a limited duration and automatically expire. Zero trust mandates
minimizing standing privileges — admin access should be JIT with approval workflows.

### GGID's Existing Foundation

GGID's Policy service already supports **time-bound role assignments**:

```go
// services/policy/internal/service/role_service.go (EXISTING)
func (s *RoleService) AssignRole(ctx context.Context, userID, roleID uuid.UUID,
    scopeType domain.ScopeType, scopeID, grantedBy uuid.UUID,
    expiresAt *time.Time) error { ... }
```

The repository already filters expired assignments:
```go
// services/policy/internal/repository/user_role_policy_repo.go
WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
```

This means JIT role elevation is partially supported — but there is no approval workflow,
no self-service request mechanism, and no automatic expiry enforcement beyond the DB query.

### JIT Role Elevation with Approval

```go
// pkg/jit/elevation.go
package jit

import (
    "context"
    "time"
    "github.com/google/uuid"
)

// ElevationRequest represents a user's request for temporary elevated access.
type ElevationRequest struct {
    ID          uuid.UUID  `json:"id"`
    UserID      uuid.UUID  `json:"user_id"`
    TenantID    uuid.UUID  `json:"tenant_id"`
    RoleID      uuid.UUID  `json:"role_id"`
    ScopeID     uuid.UUID  `json:"scope_id"`
    Reason      string     `json:"reason"`
    Duration    time.Duration `json:"duration"`
    Status      string     `json:"status"` // pending, approved, denied, expired
    ApproverID  *uuid.UUID `json:"approver_id,omitempty"`
    ApprovedAt  *time.Time `json:"approved_at,omitempty"`
    ExpiresAt   time.Time  `json:"expires_at"`
    CreatedAt   time.Time  `json:"created_at"`
}

// RoleAssigner is implemented by the Policy service.
type RoleAssigner interface {
    AssignRole(ctx context.Context, userID, roleID uuid.UUID,
        scopeType string, scopeID, grantedBy uuid.UUID, expiresAt *time.Time) error
    RevokeRole(ctx context.Context, userID, roleID uuid.UUID,
        scopeType string, scopeID uuid.UUID) error
}

// Manager handles JIT elevation lifecycle: request → approve → grant → expire.
type Manager struct {
    store     ElevationStore
    assigner  RoleAssigner
    notifiers []Notifier
}

type ElevationStore interface {
    Create(ctx context.Context, req *ElevationRequest) error
    Get(ctx context.Context, id uuid.UUID) (*ElevationRequest, error)
    ListPending(ctx context.Context, tenantID uuid.UUID) ([]*ElevationRequest, error)
    Update(ctx context.Context, req *ElevationRequest) error
}

type Notifier interface {
    Notify(ctx context.Context, req *ElevationRequest) error
}

// Request creates a new JIT elevation request (does not grant the role yet).
func (m *Manager) Request(ctx context.Context, userID, roleID uuid.UUID,
    reason string, duration time.Duration) (*ElevationRequest, error) {

    if duration > 8*time.Hour {
        return nil, fmt.Errorf("JIT elevation max duration is 8 hours")
    }
    if duration < 5*time.Minute {
        return nil, fmt.Errorf("JIT elevation min duration is 5 minutes")
    }

    req := &ElevationRequest{
        ID:        uuid.New(),
        UserID:    userID,
        RoleID:    roleID,
        Reason:    reason,
        Duration:  duration,
        Status:    "pending",
        ExpiresAt: time.Now().Add(24 * time.Hour), // request itself expires in 24h if unapproved
        CreatedAt: time.Now(),
    }

    if err := m.store.Create(ctx, req); err != nil {
        return nil, err
    }
    for _, n := range m.notifiers {
        _ = n.Notify(ctx, req) // fire-and-forget approval notifications
    }
    return req, nil
}

// Approve grants the role with the requested expiry duration.
func (m *Manager) Approve(ctx context.Context, reqID, approverID uuid.UUID) error {
    req, err := m.store.Get(ctx, reqID)
    if err != nil {
        return err
    }
    if req.Status != "pending" {
        return fmt.Errorf("request is %s, not pending", req.Status)
    }
    if time.Now().After(req.ExpiresAt) {
        req.Status = "expired"
        _ = m.store.Update(ctx, req)
        return fmt.Errorf("request has expired")
    }

    now := time.Now()
    roleExpiry := now.Add(req.Duration)
    req.Status = "approved"
    req.ApproverID = &approverID
    req.ApprovedAt = &now
    req.ExpiresAt = roleExpiry

    // Grant the role with expiry — leverages existing Policy service AssignRole
    if err := m.assigner.AssignRole(ctx, req.UserID, req.RoleID, "org", req.ScopeID,
        approverID, &roleExpiry); err != nil {
        return fmt.Errorf("grant elevated role: %w", err)
    }

    return m.store.Update(ctx, req)
}

// Deny rejects the elevation request.
func (m *Manager) Deny(ctx context.Context, reqID, denierID uuid.UUID) error {
    req, err := m.store.Get(ctx, reqID)
    if err != nil {
        return err
    }
    if req.Status != "pending" {
        return fmt.Errorf("request is %s, not pending", req.Status)
    }
    req.Status = "denied"
    req.ApproverID = &denierID
    return m.store.Update(ctx, req)
}

// CleanupExpired revokes any approved elevations that have passed their expiry.
// Run this as a periodic background job.
func (m *Manager) CleanupExpired(ctx context.Context) error {
    // In production, query store for approved requests where ExpiresAt < NOW()
    // For each, call assigner.RevokeRole and update status to "expired"
    return nil
}
```

---

## 5. Identity-Aware Proxy Pattern

> The companion doc discusses BeyondCorp's Access Proxy conceptually (section 3).
> This section provides a concrete **Go implementation** for GGID.

The identity-aware proxy (IAP) replaces traditional VPN with per-request authentication
at the proxy layer. Users authenticate to the proxy; the proxy validates identity and
forwards authorized requests to internal applications with identity headers injected.

### How It Differs from GGID's Current Gateway

GGID's gateway already acts as a reverse proxy with JWT verification — this is close to
an IAP. The key difference: a full IAP also handles **session establishment** (login flow),
**device posture enforcement**, and **per-application access policies**, not just token
forwarding.

### Identity-Aware Reverse Proxy Implementation

```go
// pkg/iap/proxy.go
package iap

import (
    "context"
    "net/http"
    "net/http/httputil"
    "net/url"
    "strings"
    "time"
)

// AccessPolicy defines per-application access rules.
type AccessPolicy struct {
    AppName        string
    RequiredRoles  []string
    RequiredTier   int       // min device trust tier
    AllowedMethods []string  // empty = all
    PathPrefix     string    // e.g., "/admin"
}

// IdentityInfo is the authenticated user context extracted by the proxy.
type IdentityInfo struct {
    UserID     string
    TenantID   string
    Email      string
    Roles      []string
    DeviceTier int
}

// AccessChecker evaluates whether an identity can access a given policy.
type AccessChecker interface {
    Check(ctx context.Context, identity *IdentityInfo, policy *AccessPolicy,
        method, path string) bool
}

// Proxy is the identity-aware reverse proxy.
type Proxy struct {
    checker   AccessChecker
    targets   map[string]*httputil.ReverseProxy // app name → backend proxy
    policies  map[string]*AccessPolicy           // path prefix → policy
    authFn    func(*http.Request) (*IdentityInfo, error) // token → identity
}

func New(checker AccessChecker, authFn func(*http.Request) (*IdentityInfo, error)) *Proxy {
    return &Proxy{
        checker:  checker,
        targets:  make(map[string]*httputil.ReverseProxy),
        policies: make(map[string]*AccessPolicy),
        authFn:   authFn,
    }
}

// RegisterApp adds a backend application with its access policy.
func (p *Proxy) RegisterApp(name, backendURL string, policy *AccessPolicy) error {
    u, err := url.Parse(backendURL)
    if err != nil {
        return err
    }
    proxy := httputil.NewSingleHostReverseProxy(u)
    // Set short timeouts for internal calls
    proxy.Transport = &http.Transport{
        MaxIdleConns:        50,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    }
    p.targets[name] = proxy
    p.policies[policy.PathPrefix] = policy
    return nil
}

// ServeHTTP is the main proxy handler: authenticate → authorize → forward.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. Authenticate: extract identity from JWT or session cookie
    identity, err := p.authFn(r)
    if err != nil {
        redirectToLogin(w, r)
        return
    }

    // 2. Match path to application policy
    policy := p.matchPolicy(r.URL.Path)
    if policy == nil {
        http.NotFound(w, r)
        return
    }

    // 3. Authorize: check roles, device tier, HTTP method
    if !p.checker.Check(r.Context(), identity, policy, r.Method, r.URL.Path) {
        http.Error(w, `{"error":"forbidden","reason":"insufficient privileges or device trust"}`,
            http.StatusForbidden)
        return
    }

    // 4. Inject identity headers for the backend application
    r.Header.Set("X-Authenticated-User", identity.UserID)
    r.Header.Set("X-Authenticated-Tenant", identity.TenantID)
    r.Header.Set("X-Authenticated-Email", identity.Email)
    r.Header.Set("X-Authenticated-Roles", strings.Join(identity.Roles, ","))
    r.Header.Set("X-Device-Tier", fmt.Sprintf("%d", identity.DeviceTier))
    // Remove the original Authorization header — backends trust proxy-injected identity
    r.Header.Del("Authorization")

    // 5. Forward to backend
    proxy := p.targets[policy.AppName]
    if proxy == nil {
        http.Error(w, "backend not registered", http.StatusBadGateway)
        return
    }
    proxy.ServeHTTP(w, r)
}

func (p *Proxy) matchPolicy(path string) *AccessPolicy {
    // Longest prefix match
    var best *AccessPolicy
    bestLen := 0
    for prefix, pol := range p.policies {
        if strings.HasPrefix(path, prefix) && len(prefix) > bestLen {
            best = pol
            bestLen = len(prefix)
        }
    }
    return best
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
    loginURL := "/auth/login?redirect=" + url.QueryEscape(r.URL.String())
    http.Redirect(w, r, loginURL, http.StatusFound)
}
```

### App-Level RBAC After Proxy Auth

Backend services should enforce RBAC using the proxy-injected identity headers, not
re-validate the JWT. This avoids redundant signature checks while maintaining defense
in depth:

```go
// Example: backend service authorization using proxy headers
func RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            roles := strings.Split(r.Header.Get("X-Authenticated-Roles"), ",")
            for _, rl := range roles {
                if strings.TrimSpace(rl) == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            http.Error(w, "forbidden", http.StatusForbidden)
        })
    }
}
```

---

## 6. Zero Trust Network Architecture for GGID

### Service Exposure Classification

| Service | Internet-Facing? | Rationale |
|---------|:----------------:|-----------|
| **Gateway** | Yes (443 only) | Single entry point; handles TLS termination, auth, rate limiting |
| **Console** | Yes (behind Gateway) | Admin UI; proxied through Gateway with session auth |
| **Auth** | No (internal) | Issues tokens; should never be directly reachable from internet |
| **OAuth** | Partial (callback endpoints only) | `/callback` and `/authorize` must be reachable for OAuth flows; all other endpoints internal |
| **Identity** | No (internal) | User CRUD — accessed only through Gateway with admin auth |
| **Policy** | No (internal) | Policy evaluation — called by other services, not directly by clients |
| **Org** | No (internal) | Org tree management — accessed only through Gateway |
| **Audit** | No (internal) | Append-only event ingestion; query API behind admin auth |

### Internal Communication Patterns

```
Client ──HTTPS──► Gateway ──mTLS──► Auth/OAuth/Identity/Policy/Org/Audit
                                         │
                                         ├──mTLS──► PostgreSQL (per-service DB)
                                         ├──TLS───► Redis (session cache)
                                         ├──TLS───► NATS JetStream (events)
                                         └──TLS───► OpenLDAP (user store)
```

**Key rules**:
1. No service connects to PostgreSQL except its own service (connection-level DB user isolation).
2. The audit service only receives events via NATS — no other service writes to the audit DB.
3. Auth and Identity both access OpenLDAP, but with different bind DNs and restricted search bases.
4. Policy service is read-only from the perspective of Auth/Gateway (they call `Check`, never modify policies).

### Database Access Restrictions

```sql
-- Per-service PostgreSQL roles (not shared credentials)
CREATE ROLE gateway_ro LOGIN PASSWORD '...' CONNECTION LIMIT 50;
CREATE ROLE auth_rw   LOGIN PASSWORD '...' CONNECTION LIMIT 30;
CREATE ROLE audit_wo  LOGIN PASSWORD '...' CONNECTION LIMIT 10;  -- write-only

-- Audit table: only audit service can INSERT
GRANT INSERT ON audit_events TO audit_wo;
REVOKE SELECT, UPDATE, DELETE ON audit_events FROM audit_wo;
GRANT SELECT ON audit_events TO gateway_ro;  -- gateway can query for display
```

### NATS as the Internal Trust Boundary

GGID uses NATS JetStream for async communication. In a zero-trust model, NATS subjects
should be ACL-protected so only authorized services can publish or subscribe:

```
# NATS account-level permissions
# Auth service can publish session events
publish = ["ggid.auth.sessions.>"]
# Audit service subscribes to all security events
subscribe = ["ggid.>"]
# Policy service subscribes to role change events
subscribe = ["ggid.policy.roles.>"]
```

---

## 7. GGID Zero Trust Gap Analysis

This section reviews GGID's actual source code to identify what exists and what's missing
for zero trust implementation.

### What Exists (Foundation)

| Capability | Location | Assessment |
|-----------|----------|------------|
| **JWT verification per request** | `services/gateway/internal/middleware/middleware.go:499` (`JWTAuth`) | Solid — RS256, JWKS, exp/nbf/iss/aud validation |
| **API key auth** | `services/gateway/internal/middleware/apikey.go:22` | Good — supports rotatable keys |
| **Session domain with device info** | `services/auth/internal/domain/session.go:10` | Good — captures IP, UA, device metadata, expiry, revocation |
| **Refresh token rotation** | `services/auth/internal/service/token_service.go:137` | Strong — detects replay attacks |
| **Session revocation** | `services/auth/internal/service/token_service.go:200-228` | Good — per-token, per-session, per-user revocation |
| **Time-bound role assignments** | `services/policy/internal/service/role_service.go:156` | Good — `expiresAt` parameter with DB-level filtering |
| **RBAC + ABAC policy engine** | `services/policy/internal/` | Strong — conditions map supports arbitrary attributes |
| **mTLS sender-constrained tokens** | `services/oauth/internal/service/jar_mtls.go:162` | Good — RFC 8705 compliance for OAuth client certs |
| **Audit event pipeline** | `pkg/audit/`, NATS JetStream | Good — structured event capture with NATS transport |
| **Tenant isolation** | `pkg/tenant/` | Strong — tenant ID in JWT claims, Row-Level Security in PG |
| **Rate limiting** | `services/gateway/internal/middleware/token_bucket.go` | Good — per-tenant, per-IP bucket limiter |

### What's Missing (Gaps)

| Gap | Priority | Impact |
|-----|----------|--------|
| **No mTLS between services** | High | Gateway→backends use plaintext HTTP. Lateral movement risk if any pod is compromised. |
| **No session revocation check at gateway** | High | `JWTAuth()` validates signature + claims but never checks if the session was revoked. A stolen token is valid until expiry. |
| **No device posture** | High | No device registration, no MDM integration, no device trust scoring. |
| **No continuous authorization** | High | No CAEP event consumption, no real-time risk evaluation, no behavioral analytics. |
| **No JIT approval workflow** | Medium | `AssignRole` supports `expiresAt` but there's no self-service request mechanism, no approval chain, no automatic cleanup. |
| **No identity-aware proxy login flow** | Medium | Gateway is a token-forwarding proxy, not a full IAP with session management and device enforcement. |
| **No network-level segmentation** | Medium | Docker Compose puts all services in one network. No NetworkPolicies or security groups. |
| **No per-service DB credentials** | Medium | Services share DB connection patterns; no connection-level isolation. |
| **No NATS ACL enforcement** | Low | Any service can publish/subscribe to any subject. |
| **No service identity (SPIFFE)** | Medium | No workload identity framework — see companion doc section 2 for SPIFFE plan. |

### Code-Level Detail: Gateway Proxy Has No TLS

```go
// services/gateway/internal/router/router.go:101-113
proxy.Transport = &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    MaxConnsPerHost:     0,
    IdleConnTimeout:     to.Idle,
    DialContext: (&net.Dialer{
        Timeout:   to.Dial,
        KeepAlive: 30 * time.Second,
    }).DialContext,
    ForceAttemptHTTP2:     true,
    TLSHandshakeTimeout:   5 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
    // ← MISSING: TLSClientConfig for mTLS to backends
}
```

The `Director` function injects `X-User-ID` and `X-Tenant-ID` headers to backends, but
since there is no mTLS, a compromised pod could spoof these headers to impersonate any
user.

### Code-Level Detail: JWTAuth Missing Revocation Check

```go
// services/gateway/internal/middleware/middleware.go:499-569
// JWTAuth validates:
//   ✅ Signature (RS256 via JWKS)
//   ✅ Expiry (exp claim)
//   ✅ Not-before (nbf claim)
//   ✅ Issuer (iss claim)
//   ✅ Audience (aud claim)
//
// JWTAuth does NOT check:
//   ❌ Whether the session backing this token has been revoked
//   ❌ Whether the user's account has been locked/disabled since token issuance
//   ❌ Whether device posture has degraded below threshold
//   ❌ Whether CAEP events indicate the credential was compromised
```

---

## 8. Implementation Roadmap

Prioritized action items based on the gap analysis. Each item references the companion
doc's phased roadmap where applicable.

### Action 1: Gateway Session Revocation Check (Effort: 2 weeks, P0)

**Problem**: Stolen tokens remain valid until JWT expiry (typically 15-60 minutes).

**Solution**: Add a `RevocationCheck` middleware after `JWTAuth` in the gateway chain.
Use Redis to store revoked session IDs with TTL matching the access token lifetime.

```go
// Wire into gateway after JWTAuth
handler := middleware.JWTAuth(jwks, true, issuer, audience)(
    middleware.RevocationCheck(redisStore)( // NEW
        proxyHandler,
    ),
)
```

**Files to create/modify**:
- `services/gateway/internal/middleware/revocation_check.go` (new)
- `services/auth/internal/service/token_service.go` (publish revocation to Redis on `RevokeAllForUser`)

**Validation**: Unit test that a revoked session returns 401 within 1 second of revocation.

### Action 2: Inter-Service mTLS (Effort: 3 weeks, P0)

**Problem**: Gateway→backend traffic is plaintext HTTP; headers can be spoofed.

**Solution**: Add `tls.Config` with internal CA certificates to all gateway proxy transports.
Issue per-service certificates via an internal CA (or SPIFFE — see companion doc Phase 3).

**Files to modify**:
- `services/gateway/internal/router/router.go` (add `TLSClientConfig` to proxy transports)
- `services/*/cmd/main.go` (load TLS certs for each service's HTTP server)
- `deploy/docker-compose.yml` (add internal CA volume, cert generation init container)

**Validation**: Verify gateway→auth traffic uses TLS 1.3; verify backends reject connections
without valid client cert.

### Action 3: Device Posture Middleware + MDM Webhook (Effort: 4 weeks, P1)

**Problem**: No device trust in access decisions.

**Solution**: Implement `DevicePostureCheck` middleware (section 2) and MDM webhook receiver.
Add device registration API to the identity service. Store posture in Redis with 5-min TTL.

**Files to create**:
- `pkg/devicetrust/score.go` (device scoring engine)
- `services/gateway/internal/middleware/device_posture.go` (middleware)
- `services/identity/internal/handler/device_handler.go` (registration + posture update API)
- `services/identity/internal/handler/mdm_webhook.go` (Jamf/Intune webhook receiver)

**Validation**: Register a device, update posture via webhook, verify gateway enforces tier
requirements per path.

### Action 4: JIT Elevation with Approval (Effort: 3 weeks, P1)

**Problem**: Standing admin privileges; no self-service elevation workflow.

**Solution**: Implement `pkg/jit/elevation.go` (section 4). Add REST endpoints for request/
approve/deny. Wire into the existing `AssignRole` with `expiresAt`. Add a background cleanup
job that revokes expired elevations.

**Files to create**:
- `pkg/jit/elevation.go` (elevation manager)
- `services/policy/internal/handler/jit_handler.go` (REST: POST /elevations, POST /elevations/{id}/approve)
- `services/policy/cmd/main.go` (start cleanup goroutine)

**Validation**: Request elevation → admin approves → role granted with 1h expiry → verify
auto-revocation after expiry.

### Action 5: Per-Request Trust Evaluation (Effort: 4 weeks, P2)

**Problem**: No continuous authorization; no risk-based step-up after initial auth.

**Solution**: Implement `pkg/zta/trust_evaluator.go` (section 3). Subscribe to CAEP events
via NATS. Add `TrustEvaluationMiddleware` to gateway chain. Integrate with Policy service's
`CheckRequest` conditions.

**Files to create**:
- `pkg/zta/trust_evaluator.go` (risk scoring, sliding window)
- `pkg/zta/revocation.go` (revocation store, CAEP consumer)
- `services/gateway/internal/middleware/trust_eval.go` (gateway middleware)

**Validation**: Inject a CAEP "session-revoked" event via NATS; verify subsequent requests
with that session return 401 within 2 seconds.

---

## References

- **Companion doc**: [`zero-trust-iam-patterns.md`](./zero-trust-iam-patterns.md) — NIST SP 800-207,
  SPIFFE/SPIRE, BeyondCorp, device posture frameworks, maturity model
- **NIST SP 800-207**: Zero Trust Architecture (August 2020)
- **RFC 8705**: OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens
- **CAEP / SSE**: OpenID Shared Signals and Events Working Group
- **GGID source**: `services/gateway/`, `services/auth/`, `services/policy/`, `services/oauth/`, `pkg/`
