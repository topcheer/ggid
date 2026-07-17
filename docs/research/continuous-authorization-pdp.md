# Continuous Authorization & Policy Decision Point (PDP): Architecture for Per-Request Authorization

> **Focus**: A centralized Policy Decision Point (PDP) architecture that evaluates every API request in real-time — combining RBAC, ABAC, ReBAC, risk signals, device posture, and time-based conditions into a single decision. Covers PDP/PEP/PIP architecture, OPA/Cedar comparison, latency optimization via Redis caching, and integration with GGID's existing policy engine.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§11), curl commands (§8).
>
> **Related**: `zero-trust-maturity-assessment.md` (CAE flagged as P0 gap), `risk-adaptive-auth-engine.md` (URE feeds into PDP), `rebac-zanzibar-fine-grained-authz.md` (ReBAC engine).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [XACML Architecture Refresher: PDP/PEP/PIP](#2-xacml-architecture-refresher-pdppeppip)
3. [GGID Current State: Fragmented Authorization](#3-ggid-current-state-fragmented-authorization)
4. [Gap Analysis](#4-gap-analysis)
5. [Proposed Architecture: Unified PDP](#5-proposed-architecture-unified-pdp)
6. [Policy Language Comparison: GGID vs OPA/Rego vs Cedar](#6-policy-language-comparison-ggid-vs-oparego-vs-cedar)
7. [Endpoint Precondition Check](#7-endpoint-precondition-check)
8. [API Design + Curl Commands](#8-api-design--curl-commands)
9. [Caching Strategy for <5ms Decisions](#9-caching-strategy-for-5ms-decisions)
10. [Decision Audit Pipeline](#10-decision-audit-pipeline)
11. [Implementation Backlog with DoD](#11-implementation-backlog-with-dod)
12. [Competitive Differentiation](#12-competitive-differentiation)

---

## 1. Executive Summary

Continuous Authorization (CA) means every API request is independently authorized — not just at login. The gateway (PEP) asks the policy service (PDP) "should this request proceed?" on every call, with context from user attributes, device posture, risk score, and threat intel (PIP). If any signal changes between requests (e.g., risk spikes, device falls out of compliance), the next request is denied even though the session is still valid.

GGID has strong policy infrastructure:
- **Policy evaluator** (`policy/service/evaluator.go:39`) — RBAC + ABAC with decision logging ✅
- **ReBAC engine** (`identity/server/rebac_cache.go:16`) — Zanzibar-style with Redis cache (60s TTL) ✅
- **ZTNA PDP** (`gateway/protected_app_router.go:206`) — Per-app policy evaluation with device posture ✅
- **Access broker PDP** (`identity/access_broker_handler.go:190`) — `evaluateAccessPolicy()` ✅
- **Policy gRPC handler** (`policy/handler/policy_handler.go:138`) — `Check()` RPC ✅
- **Decision log** (`policy/service/evaluator.go:54`) — In-memory decision log ⚠️
- **Policy simulation** (`policy/service/policy_simulation.go:74`) — Dry-run evaluation ✅

However, GGID has **3 separate authorization paths** with no unified PDP:
1. **Gateway ZTNA PDP** — per-app device posture + role checks (works for ZTNA apps only)
2. **Policy service evaluator** — RBAC + ABAC via gRPC (used by some services)
3. **ReBAC engine** — relationship-based checks (separate from RBAC/ABAC)

These are **not integrated**. A request through the gateway doesn't automatically call the policy service's evaluator. The ReBAC engine is separate from RBAC/ABAC. There's no unified per-request authorization pipeline.

**Recommendation**: Build a **Unified PDP** as an enhancement to the existing policy service that:
1. Combines RBAC + ABAC + ReBAC + risk + device posture into one decision
2. Exposes a single gRPC `Authorize()` RPC called by the gateway on every request
3. Uses Redis decision cache (5s TTL) for <5ms latency
4. Logs every decision to PostgreSQL (not in-memory)
5. Supports OPA Rego policies as an extensible policy layer (optional)

**Estimated effort**: 3 sprints for MVP (unified evaluator + gateway integration + Redis cache + DB audit).

---

## 2. XACML Architecture Refresher: PDP/PEP/PIP

### The Classic XACML Model

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│   PEP    │────▶│   PDP    │────▶│   PIP    │
│(Enforce) │     │(Decide)  │     │(Info)    │
└──────────┘     └──────────┘     └──────────┘
     │                │
     │           ┌────▼────┐
     │           │   PAP   │
     │           │(Admin)  │
     │           └─────────┘
     │
     ▼
┌──────────┐
│ Resource │
└──────────┘
```

| Component | Role | GGID Equivalent |
|-----------|------|-----------------|
| **PEP** (Policy Enforcement Point) | Intercepts request, asks PDP, enforces decision | Gateway middleware |
| **PDP** (Policy Decision Point) | Evaluates policies, returns allow/deny | Policy service evaluator |
| **PIP** (Policy Information Point) | Provides attributes (user, device, risk, time) | Identity + auth + audit services |
| **PAP** (Policy Administration Point) | Where admins define/manage policies | Console policy editor |

### Modern Zero Trust Adaptation

In Zero Trust, the PEP→PDP→PIP flow happens on **every request**, not just at login:

```
Every HTTP Request:
  1. PEP (gateway middleware) intercepts request
  2. PEP collects context: JWT claims, IP, device, path, method
  3. PEP calls PDP: "Authorize(subject, action, resource, context)"
  4. PDP calls PIP for missing attributes:
     - User roles (identity service)
     - Device posture (identity service)
     - Risk score (risk engine)
     - Time of day (system clock)
     - Threat intel (ITDR)
  5. PDP evaluates ALL applicable policies
  6. PDP returns: allow / deny / step_up + reason
  7. PEP enforces decision (proceed, block, redirect to MFA)
  8. Decision logged to audit trail
```

---

## 3. GGID Current State: Fragmented Authorization

### Existing Authorization Components

| Component | File:Line | Status | Role | Issue |
|-----------|-----------|--------|------|-------|
| Policy evaluator | `policy/service/evaluator.go:39` | **Works** ✅ | PDP (RBAC+ABAC) | Decision log in-memory |
| ReBAC engine | `identity/server/rebac_cache.go:16` | **Works** ✅ | PDP (ReBAC) | Separate from RBAC/ABAC |
| ZTNA PDP | `gateway/protected_app_router.go:206` | **Works** ✅ | PEP+PDP (ZTNA only) | Not unified with policy service |
| Access broker | `identity/access_broker_handler.go:190` | **Works** ✅ | PEP+PDP (app access) | Separate evaluation logic |
| Policy gRPC handler | `policy/handler/policy_handler.go:138` | **Works** ✅ | PDP entry point | Not called by gateway per-request |
| Decision log | `policy/service/evaluator.go:54` | **In-memory** ❌ | Decision audit | `sync.Mutex` + slice |
| Policy simulation | `policy/service/policy_simulation.go:74` | **Works** ✅ | Dry-run PDP | Not used in production path |
| Login security | `auth/service/login_security.go:41` | **Works** ✅ | PDP (login) | Login-specific, not per-request |
| JWT auth middleware | `gateway/middleware/jwt_auth.go` | **Works** ✅ | PEP (token validation) | Validates token only, no policy check |
| ReBAC cache | `identity/server/rebac_cache.go:16` | **Redis** ✅ | Cache (60s TTL) | Good pattern, extend to PDP |

### The Fragmentation Problem

```
Current request flow (3 disconnected authorization paths):

Path 1: Regular API request through gateway
  → JWT validation (jwt_auth.go) → proxy to backend
  → NO policy check! (unless ZTNA app)

Path 2: ZTNA protected app
  → ZTNA PDP (protected_app_router.go:206)
  → Checks device posture + role
  → Does NOT check ReBAC or ABAC

Path 3: Service-to-service via gRPC
  → Policy handler Check() (policy_handler.go:138)
  → RBAC + ABAC evaluation
  → Does NOT check ReBAC or device posture

Problem: A user could be denied by ZTNA PDP but allowed by Policy evaluator,
or vice versa. No unified decision.
```

---

## 4. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No unified PDP** | 3 separate authorization paths produce different results |
| 2 | **No per-request authz for regular API** | Gateway validates JWT but doesn't call policy service |
| 3 | **RBAC/ABAC + ReBAC not combined** | A user might have RBAC permission but no ReBAC relationship |
| 4 | **Decision log in-memory** | Decisions lost on restart; no compliance trail |
| 5 | **No PIP aggregation** | PDP must call multiple services for attributes (slow) |
| 6 | **No decision cache** | Every authorization hits the database |
| 7 | **No risk score in authz** | Policy evaluator doesn't consider risk score |
| 8 | **No time-based conditions** | No "deny after hours" policy support |
| 9 | **No policy versioning** | Policy changes are instant; no canary/bulkroll |
| 10 | **No OPA/Cedar integration** | Can't import external policies |

---

## 5. Proposed Architecture: Unified PDP

```
                    ┌──────────────────────────────────────────────┐
                    │         Unified Policy Decision Point         │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Gateway PEP Middleware               │    │
                    │  │  (intercepts EVERY request)           │    │
                    │  │                                      │    │
                    │  │  Request → extract context →          │    │
                    │  │  call PDP → enforce decision          │    │
                    │  └──────────────┬───────────────────────┘    │
                    │                 │                            │
                    │  ┌──────────────▼───────────────────────┐    │
                    │  │  Decision Cache (Redis)               │    │
                    │  │  Key: tenant:user:action:resource     │    │
                    │  │  TTL: 5 seconds                       │    │
                    │  │  Hit → return cached decision (<1ms)  │    │
                    │  │  Miss → call evaluator                │    │
                    │  └──────────────┬───────────────────────┘    │
                    │                 │ (cache miss)                │
                    │  ┌──────────────▼───────────────────────┐    │
                    │  │  Unified Evaluator                    │    │
                    │  │                                      │    │
                    │  │  Evaluation order:                   │    │
                    │  │  1. Check decision cache             │    │
                    │  │  2. Gather attributes from PIP       │    │
                    │  │  3. Evaluate explicit deny policies  │    │
                    │  │  4. Evaluate ReBAC relationships     │    │
                    │  │  5. Evaluate RBAC permissions        │    │
                    │  │  6. Evaluate ABAC conditions         │    │
                    │  │  7. Apply risk overlay               │    │
                    │  │  8. Default deny                     │    │
                    │  │  9. Cache + log decision             │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  PIP Aggregators                      │    │
                    │  │  ├── User attributes (identity svc)   │    │
                    │  │  ├── Device posture (identity svc)    │    │
                    │  │  ├── Risk score (risk engine)         │    │
                    │  │  ├── Time/geo (system + IP geo)       │    │
                    │  │  └── Threat intel (ITDR)              │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Decision Audit (PostgreSQL)          │    │
                    │  │  - policy_decisions table             │    │
                    │  │  - Every decision with full context   │    │
                    │  │  - Supports compliance reporting      │    │
                    │  └──────────────────────────────────────┘    │
                    └──────────────────────────────────────────────┘
```

### Request Flow

```
1. HTTP request arrives at gateway
2. JWT middleware extracts: tenant_id, user_id, scopes from token
3. Unified PEP middleware extracts: path, method, resource
4. PEP builds AuthorizeRequest:
   { tenant_id, user_id, action="api:call", resource="/api/v1/users", context }
5. PEP checks Redis decision cache (5s TTL)
6. Cache HIT → return cached decision (<1ms)
7. Cache MISS → call Unified Evaluator:
   a. PIP: fetch user roles, device posture, risk score (parallel, <5ms)
   b. Evaluate deny policies first (explicit deny wins)
   c. Evaluate ReBAC: does user have relationship to resource?
   d. Evaluate RBAC: does user's role include required permission?
   e. Evaluate ABAC: do conditions match (time, risk, device)?
   f. Apply risk overlay: if risk > threshold, upgrade to step_up
   g. Default: deny
8. Cache decision in Redis (5s TTL)
9. Log to PostgreSQL (async)
10. PEP enforces: allow → proceed, deny → 403, step_up → 401 + challenge
```

---

## 6. Policy Language Comparison: GGID vs OPA/Rego vs Cedar

| Feature | **GGID Engine** | **OPA / Rego** | **Cedar (Rust)** |
|---------|-----------------|-----------------|------------------|
| **Language** | Go-native (RBAC+ABAC+ReBAC) | Rego (Datalog-like) | Cedar (declarative) |
| **Evaluation** | Compiled Go | Interpreted | Compiled Rust |
| **Performance** | **~0.5ms** | ~2-5ms | ~0.1ms |
| **RBAC support** | **Native** | Via rules | Via policies |
| **ABAC support** | **Native** | Via data | Via context |
| **ReBAC support** | **Native (Zanzibar)** | Via data joins | Via hierarchy |
| **Data integration** | Direct DB queries | External data fetch | Context bundle |
| **Learning curve** | Low (config-based) | **High** (Datalog) | Medium |
| **Hot reload** | Via DB policy change | Via data update | Via policy update |
| **Audit** | Decision log | Decision log | Decision log |
| **Bundle/distribution** | N/A | Bundle server | Embedded |
| **Best for** | **IAM-native authz** | General-purpose | App-level authz |

### Recommendation: Keep GGID Engine as Primary

GGID's native engine is already comprehensive (RBAC + ABAC + ReBAC + Zanzibar). Adding OPA would:
- Add a Rego interpreter dependency (increases binary size)
- Require data sync between GGID DB and OPA data store
- Introduce a second policy language for admins to learn

**Better approach**: Enhance GGID's engine with:
1. Unified evaluation (combine RBAC + ABAC + ReBAC in one pass)
2. Risk overlay (apply risk score as an ABAC attribute)
3. Decision cache (Redis)
4. DB-backed decision audit

**Optional OPA**: Add OPA as a **plugin policy layer** for advanced use cases (custom Rego policies). This is the WASM plugin system already designed in `wasm-plugin-architecture.md` — Rego can compile to WASM.

---

## 7. Endpoint Precondition Check

### Existing Endpoints (Enhance)

| Endpoint | File:Line | Current | Target |
|----------|-----------|---------|--------|
| gRPC `Check` | `policy/handler/policy_handler.go:138` | **Works** ✅ | Enhance to unified `Authorize` |
| Policy evaluator | `policy/service/evaluator.go:39` | **Works** ✅ | Add ReBAC + risk overlay |
| ReBAC Check | `identity/server/rebac_cache.go:36` | **Works** ✅ | Integrate into evaluator |
| ZTNA PDP | `gateway/protected_app_router.go:206` | **Works** ✅ | Call unified PDP |
| Decision log stats | `policy/server/http.go:182` | **Hardcoded** ❌ | DB-backed |
| Decision log query | `policy/server/http.go:1841` | **In-memory** ❌ | DB-backed |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| gRPC `Authorize` | RPC | Unified per-request authorization | P0 |
| `/api/v1/policy/authorize` | POST | REST fallback for Authorize | P0 |
| `/api/v1/policy/decisions` | GET | Query decision audit trail | P0 |
| `/api/v1/policy/decisions/export` | GET | Export decisions (CSV/JSON) | P1 |
| `/api/v1/policy/cache/invalidate` | POST | Invalidate decision cache | P1 |

---

## 8. API Design + Curl Commands

### Unified Authorize

```bash
curl -X POST https://ggid.corp.com/api/v1/policy/authorize \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "subject": { "user_id": "uuid-alice" },
    "action": "api:call",
    "resource": "/api/v1/admin/users",
    "context": {
      "method": "GET",
      "ip_address": "203.0.113.42",
      "device_id": "dev_xyz",
      "session_id": "sess_abc"
    }
  }'

# Response:
{
  "decision": "allow",
  "reason": "rbac_allow: role 'admin' has 'users:read'",
  "evaluation": {
    "rbac": { "result": "allow", "matched_role": "admin", "matched_permission": "users:read" },
    "abac": { "result": "allow", "policies_evaluated": 3 },
    "rebac": { "result": "skip", "reason": "no relationship required" },
    "risk": { "score": 18, "level": "low", "overlay": "none" }
  },
  "cached": false,
  "evaluation_ms": 3.2,
  "decision_id": "dec_7f3a2b1c-..."
}
```

### Query Decision Audit Trail

```bash
curl "https://ggid.corp.com/api/v1/policy/decisions?user_id=uuid-alice&limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Response:
{
  "decisions": [
    {
      "decision_id": "dec_7f3a2b1c-...",
      "user_id": "uuid-alice",
      "action": "api:call",
      "resource": "/api/v1/admin/users",
      "decision": "allow",
      "reason": "rbac_allow",
      "risk_score": 18,
      "evaluation_ms": 3.2,
      "cached": false,
      "ip_address": "203.0.113.42",
      "created_at": "2026-07-17T10:05:00Z"
    }
  ],
  "total": 15420,
  "summary": { "allow": 15200, "deny": 180, "step_up": 40 }
}
```

### Invalidate Decision Cache

```bash
curl -X POST https://ggid.corp.com/api/v1/policy/cache/invalidate \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{ "user_id": "uuid-alice", "reason": "role_change" }'

# Response:
{ "invalidated_keys": 42, "status": "ok" }
```

---

## 9. Caching Strategy for <5ms Decisions

### Multi-Level Cache

```
Level 1: In-process LRU cache (1s TTL, 10K entries)
  → Sub-millisecond decisions for hot paths
  
Level 2: Redis decision cache (5s TTL)
  → Cross-process cache, <1ms with Redis
  
Level 3: Unified Evaluator (full evaluation)
  → 3-10ms, includes PIP calls

Distribution:
  ~80% of requests → L1 hit (0.01ms)
  ~15% of requests → L2 hit (0.5ms)  
  ~5% of requests  → L3 miss (5ms)
  Average: ~0.3ms per decision
```

### Cache Key Design

```
Key: ggid:pdp:{tenant}:{user}:{action}:{resource_hash}
TTL: 5 seconds (configurable)
Invalidation:
  - User role change → invalidate all keys for user
  - Policy change → invalidate all keys for tenant
  - Risk score change → don't invalidate (short TTL handles it)
```

### Cache Invalidation Events

| Event | Action | Scope |
|-------|--------|-------|
| User role assigned/revoked | Flush user's keys | Per-user |
| Policy created/updated/deleted | Flush tenant's keys | Per-tenant |
| Device posture change | Flush user's keys | Per-user |
| Session revoked | Flush user's keys | Per-user |
| Admin cache flush | Flush specified scope | Manual |

---

## 10. Decision Audit Pipeline

### Database Schema

```sql
-- Policy decisions (every PDP evaluation)
CREATE TABLE policy_decisions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID,
    session_id          VARCHAR(128),

    -- Request context
    action              VARCHAR(128) NOT NULL,
    resource            VARCHAR(512) NOT NULL,
    method              VARCHAR(8),
    ip_address          VARCHAR(45),
    device_id           VARCHAR(128),

    -- Decision
    decision            VARCHAR(16) NOT NULL,        -- 'allow', 'deny', 'step_up'
    reason              VARCHAR(256),
    evaluation_ms       DOUBLE PRECISION,
    cached              BOOLEAN DEFAULT false,

    -- Detailed evaluation
    rbac_result         VARCHAR(16),
    rbac_matched_role   VARCHAR(128),
    abac_result         VARCHAR(16),
    abac_policies_count INT,
    rebac_result        VARCHAR(16),
    risk_score          INT,
    risk_overlay        VARCHAR(32),

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_decisions_tenant_time ON policy_decisions (tenant_id, created_at DESC);
CREATE INDEX idx_decisions_user ON policy_decisions (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_decisions_resource ON policy_decisions (tenant_id, resource, created_at DESC);
CREATE INDEX idx_decisions_decision ON policy_decisions (tenant_id, decision, created_at DESC);
```

### Async Logging

Decisions are logged asynchronously via NATS to avoid blocking the request:

```
1. PEP calls PDP → decision returned (sync, 3ms)
2. PDP publishes decision to NATS (async, <0.1ms)
3. Audit subscriber writes to policy_decisions table (background)
4. Decisions searchable within 1 second
```

---

## 11. Implementation Backlog with DoD

### P0 — Unified PDP + Gateway Integration (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Unified `Authorize()` gRPC RPC | ✅ Combines RBAC + ABAC + ReBAC + risk ✅ gRPC handler registered ✅ ≥3 tests | 4d |
| 2 | Gateway PEP middleware (per-request authz) | ✅ Every API request calls PDP ✅ Configurable per-route ✅ ≥3 tests | 3d |
| 3 | Redis decision cache (5s TTL) | ✅ Cache hit returns <1ms ✅ Invalidation on role/policy change ✅ ≥3 tests | 3d |
| 4 | PIP aggregators (parallel attribute fetch) | ✅ User roles + device posture + risk in parallel ✅ <5ms total ✅ ≥3 tests | 3d |
| 5 | Replace in-memory decision log with DB | ✅ policy_decisions table ✅ No sync.Mutex/slice ✅ Async via NATS ✅ ≥3 tests | 3d |
| 6 | REST `/api/v1/policy/authorize` endpoint | ✅ HTTP fallback for non-gRPC clients ✅ curl test PASS ✅ ≥3 tests | 2d |

### P1 — Risk Overlay + Decision Analytics (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | Risk overlay in evaluator | ✅ Risk score upgrades decision to step_up ✅ Configurable threshold ✅ ≥3 tests | 3d |
| 8 | Decision audit trail query API | ✅ Filter by user/resource/decision/time ✅ DB-backed ✅ ≥3 tests | 2d |
| 9 | Replace hardcoded decision stats | ✅ /policy/stats/decision-log returns real data ✅ DB-backed ✅ ≥3 tests | 2d |
| 10 | Cache invalidation API | ✅ POST /policy/cache/invalidate ✅ Scoped (user/tenant) ✅ ≥3 tests | 1d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 11 | OPA Rego plugin layer | Compile Rego → WASM → execute in wazero sandbox |
| 12 | Policy versioning + canary | Version policies, route % traffic to new version |
| 13 | In-process LRU cache | Sub-millisecond decisions for hot paths |
| 14 | Decision analytics dashboard | Decision trends, deny rate, avg latency |
| 15 | External PDP mode | Deploy PDP as separate service (sidecar or central) |

---

## 12. Competitive Differentiation

| Feature | GGID (target) | OPA | AWS Verified Access | Cloudflare Access | Google BeyondCorp |
|---------|---------------|-----|---------------------|-------------------|-------------------|
| **Unified RBAC+ABAC+ReBAC** | **Native** | Via Rego | IAM + tags | Zero Trust rules | IAM + Context-Aware |
| **Risk overlay** | **Risk score → decision** | Custom data | Trust provider | Custom | Adaptive Access |
| **Decision latency** | **<5ms (cached <1ms)** | 2-5ms | ~10ms | ~5ms | ~10ms |
| **Decision audit** | **DB-backed (NATS async)** | Decision log | CloudTrail | Audit logs | Cloud Audit Logs |
| **Policy language** | **Config-based (no DSL)** | Rego | JSON policy | CEL expressions | IAM conditions |
| **PIP integration** | **Native (identity+risk+device)** | External data | Trust provider | Custom | IAM Context |
| **Cache strategy** | **Redis 5s + invalidation** | In-memory | N/A | N/A | N/A |
| **Per-request authz** | **Gateway PEP on every call** | Sidecar | Per-connection | Per-request | Per-request |
| **Open source** | **Yes (Apache 2.0)** | Yes | No | No | No |

**Key differentiator**: GGID would be the only open-source IAM with a **unified PDP** combining RBAC + ABAC + ReBAC + risk overlay + device posture in a single evaluation, with sub-millisecond cached decisions and full DB-backed audit. OPA is general-purpose but requires external data integration; GGID's PDP is IAM-native.

---

## References

- [NIST SP 800-207: Zero Trust Architecture](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-207.pdf) — PEP/PDP/PIP model
- [XACML 3.0](https://docs.oasis-open.org/xacml/3.0/xacml-3.0-core-spec-os-en.html) — Policy decision architecture
- [Open Policy Agent](https://www.openpolicyagent.org/) — General-purpose policy engine
- [Cedar Policy Language](https://www.cedarpolicy.com/) — Rust-based policy DSL (Amazon)
- [Google BeyondCorp](https://cloud.google.com/beyondcorp) — Per-request access evaluation
- [AWS Verified Access](https://aws.amazon.com/verified-access/) — Per-request ZTNA
- [GGID Policy Evaluator](../services/policy/internal/service/evaluator.go) — RBAC+ABAC engine at line 39
- [GGID ReBAC Cache](../services/identity/internal/server/rebac_cache.go) — Redis-cached Zanzibar at line 16
- [GGID ZTNA PDP](../services/gateway/internal/router/protected_app_router.go) — Per-app PDP at line 206
- [GGID Access Broker PDP](../services/identity/internal/server/access_broker_handler.go) — evaluateAccessPolicy at line 190
- [GGID Policy gRPC Handler](../services/policy/internal/handler/policy_handler.go) — Check() RPC at line 138
- [GGID Decision Log](../services/policy/internal/service/evaluator.go) — In-memory at line 54
- [GGID Policy Simulation](../services/policy/internal/service/policy_simulation.go) — Dry-run at line 74
- [GGID Zero Trust Maturity Assessment](./zero-trust-maturity-assessment.md) — CAE flagged as P0 gap
- [GGID Risk Adaptive Auth Engine](./risk-adaptive-auth-engine.md) — URE feeds into PDP
- [GGID ReBAC/Zanzibar](./rebac-zanzibar-fine-grained-authz.md) — Relationship-based authz
- [GGID WASM Plugin Architecture](./wasm-plugin-architecture.md) — Plugin policy layer (OPA → WASM)
