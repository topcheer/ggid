# Session Management & Token Lifecycle Security: Comprehensive Session Hardening for GGID

> **Focus**: A comprehensive assessment of GGID's session and token lifecycle security — session management, refresh token rotation, token revocation, introspection, back-channel logout, session hijacking defense, and token binding — with production hardening recommendations.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Session & Token Infrastructure](#2-ggid-current-state-session--token-infrastructure)
3. [Gap Analysis](#3-gap-analysis)
4. [Session Security Architecture](#4-session-security-architecture)
5. [Refresh Token Rotation](#5-refresh-token-rotation)
6. [Token Revocation & Introspection](#6-token-revocation--introspection)
7. [Back-Channel Logout](#7-back-channel-logout)
8. [Session Hijacking Defense](#8-session-hijacking-defense)
9. [Endpoint Precondition Check](#9-endpoint-precondition-check)
10. [API Design + Curl Commands](#10-api-design--curl-commands)
11. [Database Schema](#11-database-schema)
12. [Implementation Backlog with DoD](#12-implementation-backlog-with-dod)
13. [Competitive Differentiation](#13-competitive-differentiation)

---

## 1. Executive Summary

Session and token lifecycle management is the backbone of identity security — compromised sessions and stolen tokens are the #1 attack vector for account takeover (per Verizon DBIR 2026).

GGID has **extensive session and token infrastructure** across auth and oauth services:
- Session management service (`auth/service/session_management.go`) ✅
- Session timeout by risk (`auth/server/session_timeout_handler.go:9`) ✅
- Session limit enforcement (`auth/service/session_limit_test.go`) ✅
- Refresh token rotation (`auth/service/` — gap regression tests) ✅
- Token revocation (`oauth/service/token_revocation.go`) ✅
- Token introspection (`oauth/service/introspection_cache.go`) ✅
- Back-channel logout (`oauth/service/logout.go` + E2E test) ✅
- Session hijack detection (`auth/server/hijack_check_handler.go:14`) ⚠️ Hardcoded
- Session re-evaluation (`auth/server/session_reevaluate.go:14`) ⚠️ Basic
- Revoke cascade (`oauth/server/revoke_cascade_handler.go:62`) ✅
- Gateway session middleware (`gateway/middleware/session.go`) ✅
- Gateway session timeout (`gateway/middleware/session_timeout.go`) ✅
- Session binding config (`auth/server/session_binding_config_handler.go`) ✅
- Termination reasons (`auth/server/termination_reasons_handler.go`) ✅

**Key gaps:**
1. Hijack detection hardcoded — returns fake detections
2. Session re-evaluation is basic stub
3. No DPoP binding on all sessions (only OAuth tokens)
4. No continuous session risk scoring
5. No device fingerprint binding to sessions
6. No session geo-anomaly real-time detection

**Recommendation**: Wire hardcoded handlers to real data, add continuous session risk scoring via CAE, enforce DPoP binding on all token-bearing sessions, and integrate session signals with the Unified Risk Engine.

---

## 2. GGID Current State: Session & Token Infrastructure

### Existing Components (43 files matched)

| Component | File | Status | Notes |
|-----------|------|--------|-------|
| SessionManagement | `auth/service/session_management.go` | ✅ | Core session CRUD |
| Session timeout (risk) | `auth/server/session_timeout_handler.go:9` | ✅ | Dynamic timeout by risk score |
| Session limit | `auth/service/session_limit_test.go` | ✅ | Max concurrent sessions |
| Session binding config | `auth/server/session_binding_config_handler.go` | ✅ | Configurable binding |
| Session re-evaluate | `auth/server/session_reevaluate.go:14` | ⚠️ Stub | Basic risk re-score |
| Session inspect | `auth/server/session_inspect_handler.go:26` | ⚠️ Hardcoded | Returns mock data |
| Session anomaly | `auth/server/session_anomaly_handler.go:46` | ⚠️ Hardcoded | Returns default low-risk |
| Hijack detection | `auth/server/hijack_check_handler.go:14` | ❌ Hardcoded | 3 fake detections |
| Termination reasons | `auth/server/termination_reasons_handler.go` | ✅ | Session end reasons |
| Token revocation | `oauth/service/token_revocation.go` | ✅ | RFC 7009 |
| Revoke cascade | `oauth/server/revoke_cascade_handler.go:62` | ✅ | Cascading revocation |
| Token introspection | `oauth/service/introspection_cache.go` | ✅ | RFC 7662 |
| Refresh token rotation | `auth/service/auth_service.go` | ✅ | Opaque + hash stored |
| Back-channel logout | `oauth/service/logout.go` | ✅ | OIDC Back-Channel Logout |
| Gateway session MW | `gateway/middleware/session.go` | ✅ | Session validation |
| Gateway session timeout | `gateway/middleware/session_timeout.go` | ✅ | Timeout enforcement |
| Session risk score | `auth/server/session_timeout_handler.go` | ✅ | Risk → timeout mapping |

---

## 3. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | Hijack detection hardcoded | No real session hijack detection |
| 2 | Session inspect returns mock | Can't inspect real session details |
| 3 | Session anomaly returns default | No real anomaly scoring |
| 4 | Session re-evaluation basic | No continuous risk scoring |
| 5 | No DPoP on all sessions | Sessions not proof-of-possession bound |
| 6 | No device fingerprint binding | Session not tied to specific device |
| 7 | No real-time geo anomaly | No instant detection of geo change |
| 8 | No session token store | Sessions tracked but not in DB for audit |

---

## 4. Session Security Architecture

### Target: Continuous Session Protection

```
Login → Session Created
  │
  ├── Session bound to: user_id + device_fingerprint + IP + risk_score
  ├── DPoP key bound: token cnf.jkt = device public key
  ├── Risk score: initial (from login risk assessment)
  │
  ▼
Every subsequent request:
  │
  ├── Gateway session middleware: validate session exists + not expired
  ├── CAE middleware: re-evaluate session risk (every 15 min)
  │   ├── Check: same device fingerprint?
  │   ├── Check: same IP range?
  │   ├── Check: same geo?
  │   ├── Check: risk engine signals?
  │   └── If risk increased → shorten timeout / require step-up / revoke
  ├── PDP middleware: authorize this specific request
  └── If all pass → forward to backend
```

### Session Lifecycle States

| State | Trigger | Action |
|-------|---------|--------|
| `active` | Login success | Normal operation |
| `elevated` | Step-up completed | Higher privilege window |
| `risk_elevated` | CAE detects risk increase | Shortened timeout |
| `challenged` | Step-up required but not completed | Read-only until challenge |
| `suspended` | Admin action or critical risk | All requests blocked |
| `expired` | TTL exceeded | Must re-authenticate |
| `revoked` | Admin/logout/token revocation | Immediate termination |

---

## 5. Refresh Token Rotation

### Current Implementation

```go
// auth/service/auth_service.go
// Refresh tokens are opaque, SHA-256 hashed, stored in Redis
// Rotation: each use generates new refresh token + invalidates old
// Detection: if old token reused after rotation → revoke entire family
```

### Rotation Security

| Property | GGID Status | Standard |
|----------|-------------|----------|
| Opaque tokens (not JWT) | ✅ | Best practice |
| SHA-256 hashed at rest | ✅ | Required |
| Rotation on each use | ✅ | RFC 6749 §10.4 |
| Token family detection | ✅ | Reuse detection |
| Reuse → revoke family | ✅ | Critical security |
| Lifetime limit | ✅ | Configurable |
| Binding (DPoP/mTLS) | ⚠️ OAuth only | Should be all tokens |

---

## 6. Token Revocation & Introspection

### Existing (RFC 7009 + RFC 7662)

| Feature | Status | Implementation |
|---------|--------|----------------|
| `POST /oauth/revoke` (RFC 7009) | ✅ | `token_revocation.go` |
| `POST /oauth/introspect` (RFC 7662) | ✅ | `introspection_cache.go` |
| Cascading revocation | ✅ | `revoke_cascade_handler.go:62` |
| Revocation stats | ✅ | `token_revocation_stats_handler.go` |
| Introspection cache | ✅ | Redis-cached |

---

## 7. Back-Channel Logout

### OIDC Back-Channel Logout (RFC drafts)

```go
// oauth/service/logout.go
// When user logs out from one app:
// 1. GGID sends logout token to all registered back_channel_logout_uris
// 2. Each app receives logout_token (JWT with sub + sid + events)
// 3. App validates token → terminates local session
```

### E2E Test Exists

`oauth/service/backchannel_logout_e2e_test.go` — full end-to-end test ✅

---

## 8. Session Hijacking Defense

### Defense in Depth

| Layer | Defense | GGID Status |
|-------|---------|-------------|
| **Network** | TLS everywhere | ✅ |
| **Token** | DPoP proof-of-possession | ✅ (OAuth tokens) |
| **Token** | mTLS client cert binding | ✅ (OAuth clients) |
| **Session** | IP binding + anomaly detection | ⚠️ (hijack_check hardcoded) |
| **Session** | Device fingerprint binding | ❌ Missing |
| **Session** | Geo-anomaly detection | ⚠️ (impossible_travel exists but not per-session) |
| **Behavioral** | CAE continuous risk scoring | 📋 (researched) |
| **Detection** | ITDR session rules | ⚠️ (token_replay exists, session_hijack needed) |

### Session Hijack Detection (Upgrade)

```go
func detectSessionHijack(session *Session, request *http.Request) *HijackAlert {
    alerts := []string{}

    // 1. IP change mid-session
    if session.IPAddress != "" && request.RemoteAddr != session.IPAddress {
        if !sameSubnet(session.IPAddress, request.RemoteAddr) {
            alerts = append(alerts, "ip_address_changed_subnet")
        }
    }

    // 2. User-Agent change mid-session
    if session.UserAgent != "" && request.UserAgent() != session.UserAgent {
        alerts = append(alerts, "user_agent_changed")
    }

    // 3. Geo change (impossible travel)
    currentGeo := geoLookup(request.RemoteAddr)
    if session.GeoCountry != "" && currentGeo != session.GeoCountry {
        alerts = append(alerts, "geo_country_changed")
    }

    // 4. Device fingerprint mismatch
    if session.DeviceFingerprint != "" {
        currentFP := computeFingerprint(request)
        if currentFP != session.DeviceFingerprint {
            alerts = append(alerts, "device_fingerprint_mismatch")
        }
    }

    if len(alerts) >= 2 {
        return &HijackAlert{Severity: "critical", Signals: alerts,
            Action: "revoke_session_and_require_reauth"}
    }
    return nil
}
```

---

## 9. Endpoint Precondition Check

### Existing (Enhance)

| Component | File | Current | Target |
|----------|------|---------|--------|
| Session management | `auth/service/session_management.go` | ✅ | Add device binding |
| Session timeout (risk) | `auth/server/session_timeout_handler.go:9` | ✅ | Keep, wire CAE |
| Hijack detection | `auth/server/hijack_check_handler.go:14` | ❌ Hardcoded | Real detection |
| Session inspect | `auth/server/session_inspect_handler.go:26` | ⚠️ Hardcoded | Real data |
| Session anomaly | `auth/server/session_anomaly_handler.go:46` | ⚠️ Hardcoded | Real scoring |
| Session re-evaluate | `auth/server/session_reevaluate.go:14` | ⚠️ Stub | CAE integration |
| Token revocation | `oauth/service/token_revocation.go` | ✅ | Keep |
| Token introspection | `oauth/service/introspection_cache.go` | ✅ | Keep |
| Back-channel logout | `oauth/service/logout.go` | ✅ | Keep |
| Revoke cascade | `oauth/server/revoke_cascade_handler.go:62` | ✅ | Keep |

### New Components

| Component | Priority |
|-----------|----------|
| Real session hijack detection | P0 |
| Device fingerprint session binding | P0 |
| Session store (DB-backed audit) | P1 |
| CAE session risk scoring | P1 |
| Session security dashboard | P2 |

---

## 10. API Design + Curl Commands

### Session Inspection (Real Data)

```bash
curl "https://ggid.corp.com/api/v1/auth/sessions/{session_id}/inspect" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response (real data, not mock):
{
  "session_id": "sess_abc123",
  "user_id": "uuid-alice",
  "created_at": "2026-07-17T08:00:00Z",
  "last_activity": "2026-07-17T10:05:00Z",
  "expires_at": "2026-07-17T12:00:00Z",
  "ip_address": "203.0.113.42",
  "geo": { "country": "US", "city": "San Francisco" },
  "device_fingerprint": "fp_abc",
  "user_agent": "Mozilla/5.0...",
  "risk_score": 22,
  "state": "active",
  "privilege_level": "elevated",
  "step_up_completed_at": "2026-07-17T08:01:00Z"
}
```

### Session Revocation (Admin)

```bash
curl -X POST "https://ggid.corp.com/api/v1/auth/sessions/{session_id}/revoke" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"reason": "suspected_hijack", "notify_user": true}'

# Response:
{ "status": "revoked", "cascade": ["oauth_tokens_revoked": 3, "refresh_tokens_revoked": 1] }
```

### Hijack Detection Alert

```bash
curl "https://ggid.corp.com/api/v1/auth/sessions/hijack-alerts?severity=critical" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response (real alerts):
{
  "alerts": [
    {
      "session_id": "sess_xyz",
      "user_id": "uuid-bob",
      "severity": "critical",
      "signals": ["ip_address_changed_subnet", "geo_country_changed", "user_agent_changed"],
      "action_taken": "session_revoked",
      "detected_at": "2026-07-17T10:03:00Z"
    }
  ]
}
```

---

## 11. Database Schema

```sql
CREATE TABLE sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    session_token_hash  VARCHAR(256) NOT NULL,

    -- Binding
    ip_address          VARCHAR(45),
    geo_country         VARCHAR(2),
    geo_city            VARCHAR(128),
    device_fingerprint  VARCHAR(256),
    user_agent          TEXT,
    dpop_jkt            VARCHAR(256),           -- DPoP public key thumbprint

    -- Risk
    risk_score          INT DEFAULT 0,
    risk_level          VARCHAR(16) DEFAULT 'low',

    -- State
    status              VARCHAR(16) DEFAULT 'active',
    privilege_level     VARCHAR(16) DEFAULT 'normal',
    step_up_at          TIMESTAMPTZ,

    -- Timing
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_activity       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    revoke_reason       TEXT,

    UNIQUE(session_token_hash)
);

CREATE TABLE session_hijack_alerts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    session_id          UUID NOT NULL,
    user_id             UUID NOT NULL,
    severity            VARCHAR(16) NOT NULL,
    signals             JSONB NOT NULL,           -- ["ip_changed", "geo_changed"]
    action_taken        VARCHAR(32),
    detected_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at         TIMESTAMPTZ,
    resolved_by         UUID
);

CREATE INDEX idx_sessions_tenant_user ON sessions (tenant_id, user_id, status);
CREATE INDEX idx_sessions_hash ON sessions (session_token_hash) WHERE status = 'active';
CREATE INDEX idx_sessions_activity ON sessions (tenant_id, last_activity);
CREATE INDEX idx_hijack_tenant_time ON session_hijack_alerts (tenant_id, detected_at DESC);
CREATE INDEX idx_hijack_unresolved ON session_hijack_alerts (tenant_id, severity) WHERE resolved_at IS NULL;
```

---

## 12. Implementation Backlog with DoD

### P0 — Real Session Security (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Replace hijack detection hardcoded handler | ✅ Real signal detection (IP/UA/geo) ✅ DB-backed ✅ ≥3 tests | 3d |
| 2 | Replace session inspect hardcoded handler | ✅ Returns real session data ✅ DB-backed ✅ ≥3 tests | 2d |
| 3 | Replace session anomaly hardcoded handler | ✅ Real risk scoring ✅ DB-backed ✅ ≥3 tests | 2d |
| 4 | Device fingerprint session binding | ✅ Session bound to device ✅ Mismatch detected ✅ ≥3 tests | 3d |

### P1 — CAE Integration + Session Store (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Session store (DB-backed) | ✅ All sessions in PostgreSQL ✅ Audit trail ✅ ≥3 tests | 3d |
| 6 | CAE session risk re-evaluation | ✅ Every 15 min risk re-score ✅ Auto-shorten timeout on risk ↑ ✅ ≥3 tests | 3d |
| 7 | DPoP binding on all sessions | ✅ Session token bound to DPoP key ✅ ≥3 tests | 2d |

### P2 — Dashboard + Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 8 | Session security dashboard | Active sessions, risk distribution, hijack alerts |
| 9 | Geo-anomaly real-time | Instant detection + push notification |
| 10 | Session recording (PAM) | Record privileged session activity |
| 11 | Concurrent session policy | Per-user max session enforcement |
| 12 | Session analytics | Session duration, frequency, patterns |

---

## 13. Competitive Differentiation

| Feature | GGID (target) | Okta | Microsoft Entra | Auth0 | Keycloak |
|---------|---------------|------|-----------------|-------|----------|
| **Session management** | **DB-backed** ✅ | Yes | Yes | Yes | Yes |
| **Risk-based timeout** | **Existing** ✅ | Yes | Yes | Custom | No |
| **Refresh token rotation** | **Existing** ✅ | Yes | Yes | Yes | Yes |
| **DPoP binding** | **OAuth ✅ + sessions (target)** | No | No | No | No |
| **Hijack detection** | **Real signals (target)** | Yes | Yes | Custom | No |
| **Back-channel logout** | **Existing** ✅ | Yes | Yes | Yes | Yes |
| **Revoke cascade** | **Existing** ✅ | Yes | Yes | Yes | Partial |
| **Device fingerprint binding** | **Target** | Yes | Yes | Custom | No |
| **CAE integration** | **Target** | Partial | Yes | No | No |
| **Open source** | **Yes** | No | No | No | Yes |

**Key differentiator**: GGID already has best-in-class refresh token rotation, token revocation, introspection, and back-channel logout. The gaps are in session-level defense (hijack detection, device binding, CAE) — upgrading these makes GGID enterprise-ready without the gaps competitors fill with proprietary add-ons.

---

## References

- [RFC 6749 §10.4](https://datatracker.ietf.org/doc/html/rfc6749#section-10.4) — Refresh token rotation
- [RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009) — Token revocation
- [RFC 7662](https://datatracker.ietf.org/doc/html/rfc7662) — Token introspection
- [RFC 9126](https://datatracker.ietf.org/doc/html/rfc9126) — DPoP
- [OIDC Back-Channel Logout](https://openid.net/specs/openid-connect-backchannel-1_0.html) — Session management
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [GGID Session Management](../services/auth/internal/service/session_management.go) — Core session service
- [GGID Session Timeout](../services/auth/internal/server/session_timeout_handler.go) — Risk-based at line 9
- [GGID Token Revocation](../services/oauth/internal/service/token_revocation.go) — RFC 7009
- [GGID Introspection Cache](../services/oauth/internal/service/introspection_cache.go) — RFC 7662
- [GGID Back-Channel Logout](../services/oauth/internal/service/logout.go) — OIDC logout
- [GGID Hijack Detection](../services/auth/internal/server/hijack_check_handler.go) — Hardcoded at line 14
- [GGID Gateway Session MW](../services/gateway/internal/middleware/session.go) — Gateway enforcement
- [GGID Continuous Auth Research](./continuous-authorization-pdp.md) — CAE middleware
- [GGID Risk Adaptive Auth](./risk-adaptive-auth-engine.md) — Unified risk engine
