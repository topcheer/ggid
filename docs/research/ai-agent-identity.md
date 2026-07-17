# AI Agent Identity & Delegated Access: Production Implementation Guide for GGID

> **Focus**: A production-grade agent identity system — first-class AI agent principals with workload identity, delegated authorization via RFC 8693 token exchange, per-task scope bounding, multi-agent delegation chains, behavioral anomaly detection, and full dual-attribution audit trails. This document focuses on **implementation specifics** — DB schema, API contracts, code integration points, and DoD — building on the theoretical analysis in `ai-agent-identity-analysis.md` (2108 lines).
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§12), curl commands (§8).
>
> **Related**: `ai-agent-identity-analysis.md` (theory + competitive), `oauth21-mcp-agent-auth-pqc-migration.md` (MCP auth), `token_exchange_delegation.go` (existing impl).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [GGID Current State: Agent Infrastructure](#2-ggid-current-state-agent-infrastructure)
3. [Gap Analysis](#3-gap-analysis)
4. [Proposed Architecture](#4-proposed-architecture)
5. [Agent Identity Model](#5-agent-identity-model)
6. [Delegated Authorization Flow](#6-delegated-authorization-flow)
7. [Endpoint Precondition Check](#7-endpoint-precondition-check)
8. [API Design + Curl Commands](#8-api-design--curl-commands)
9. [Database Schema](#9-database-schema)
10. [Multi-Agent Delegation Chains](#10-multi-agent-delegation-chains)
11. [Agent Behavioral Anomaly Detection](#11-agent-behavioral-anomaly-detection)
12. [Implementation Backlog with DoD](#12-implementation-backlog-with-dod)
13. [Competitive Differentiation](#13-competitive-differentiation)
14. [Security Considerations](#14-security-considerations)

---

## 1. Executive Summary

AI agents are becoming first-class digital workers — autonomous software that reads emails, manages infrastructure, executes trades, and interacts with APIs on behalf of humans. These agents need **identity** (who they are), **authorization** (what they can do), and **attribution** (who they act for).

GGID has partial agent infrastructure:
- **Token exchange delegation** (`token_exchange_delegation.go:18`) — RFC 8693 with delegation chains ✅
- **Agent consent** (`agent_consent_handler.go:18`) — Human approves agent's scope requests ⚠️ (in-memory)
- **Scope delegation** (`scope_delegation_handler.go`) — Scope narrowing per delegation ⚠️
- **Delegation store** (`delegation_pg.go:13`) — PG-backed delegation chains ✅
- **DPoP** (`dpop_pg.go`) — Proof-of-possession tokens ✅
- **MCP tools** (`services/mcp/`) — MCP server with audit/policy/users tools ✅

However, critical pieces are missing:
1. **No agent registry** — agents have no persistent identity (no DB table)
2. **No workload identity** — no SPIFFE/SPIRE or equivalent attestation
3. **Agent consent in-memory** — `PendingConsentRequest` uses `sync.RWMutex` + map
4. **No per-task tokens** — agents get broad scopes, not JIT per-task grants
5. **No agent behavioral baselines** — can't detect agent gone rogue
6. **No rate limiting per agent** — agents can make unlimited API calls
7. **No dual-attribution audit** — actions not tagged with both agent_id + delegator_id
8. **No agent lifecycle management** — no registration/attestation/revocation

**Recommendation**: Build an **Agent Identity Service** with: agent registry (DB-backed), workload attestation, per-task delegated tokens, human-in-the-loop consent, behavioral anomaly detection, per-agent rate limiting, and dual-attribution audit.

**Estimated effort**: 4 sprints for MVP (registry + delegation + consent + audit + rate limiting).

---

## 2. GGID Current State: Agent Infrastructure

### Existing Components

| Component | File:Line | Status | Issue |
|-----------|-----------|--------|-------|
| Token exchange delegation | `oauth/server/token_exchange_delegation.go:18` | **Works** ✅ | RFC 8693 with DelegationEntry chain |
| DelegationEntry | `token_exchange_delegation.go:14` | **Works** ✅ | Actor + Subject + Scope + Reason |
| PG delegation store | `oauth/server/delegation_pg.go:13` | **DB-backed** ✅ | delegation_chains table |
| Agent consent handler | `oauth/server/agent_consent_handler.go:18` | **In-memory** ❌ | `sync.RWMutex` + map |
| Scope delegation | `oauth/server/scope_delegation_handler.go` | **Implemented** | Scope narrowing |
| DPoP store | `oauth/server/dpop_pg.go` | **DB-backed** ✅ | Token binding |
| RAR handler | `oauth/server/rar_handler.go:203` | **Works** ✅ | Rich authorization requests |
| RAR consent preview | `oauth/server/rar_handler.go:203` | **Works** ✅ | Human-readable consent |
| MCP server | `services/mcp/` | **Implemented** ✅ | Agent tools gateway |
| MCP client | `services/mcp/internal/client/` | **Works** ✅ | GGID API client |
| ABAC conditions | `policy/abac_condition_config_handler.go` | **Works** ✅ | env.risk_score in conditions |
| Feature flag (agents) | `policy/feature_flags_handler.go` | **Hardcoded** ❌ | Mock rollout state |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No agent registry** | Agents have no persistent identity |
| 2 | **No workload attestation** | Can't verify agent binary/runtime |
| 3 | **Agent consent in-memory** | Consent requests lost on restart |
| 4 | **No per-task tokens** | Agents get broad persistent scopes |
| 5 | **No agent rate limiting** | Agents can overwhelm APIs |
| 6 | **No behavioral baselines** | Can't detect rogue agents |
| 7 | **No dual-attribution audit** | Can't distinguish agent vs human action |
| 8 | **No lifecycle management** | No registration/attestation/revocation |
| 9 | **No agent-to-agent delegation** | Multi-agent chains not validated |
| 10 | **No human-in-the-loop** | Sensitive operations auto-approved |

---

## 3. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "Register an AI agent 'InvoiceProcessor'" | No API | POST /agents → DB-backed identity |
| 2 | "Agent requests read access to invoices for user Alice" | In-memory consent (lost on restart) | DB-backed consent with approval flow |
| 3 | "Agent gets a token scoped to only invoices:read for 1 hour" | Broad scopes | Per-task JIT token with narrowest scope |
| 4 | "Detect if agent is making unusual API calls" | No baseline | Behavioral anomaly: agent deviates from normal pattern |
| 5 | "Limit agent to 100 API calls per minute" | No rate limit | Per-agent Redis rate limiter |
| 6 | "Audit trail shows: agent_id + delegator_id" | Single attribution | Dual-attribution: both agent and human |
| 7 | "Agent A delegates to Agent B" | No validation | Delegation chain with max depth + cycle detection |
| 8 | "Revoke all tokens for a compromised agent" | No API | POST /agents/{id}/revoke → cascading revocation |

---

## 4. Proposed Architecture

```
                    ┌──────────────────────────────────────────────┐
                    │       Agent Identity Service                  │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Agent Registry (PostgreSQL)          │    │
                    │  │  - agents table (identity + metadata) │    │
                    │  │  - agent_attestations (workload proof) │    │
                    │  │  - agent_consent_requests (DB-backed)  │    │
                    │  │  - agent_delegation_chains (max depth) │    │
                    │  │  - agent_rate_limits (Redis counters)  │    │
                    │  │  - agent_behavior_log (baselines)      │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Delegated Authorization Pipeline     │    │
                    │  │                                      │    │
                    │  │  1. Agent registers + attests         │    │
                    │  │  2. Agent requests delegation token   │    │
                    │  │  3. Human approves consent (DB)       │    │
                    │  │  4. Token exchange (RFC 8693)         │    │
                    │  │     - act claim = agent_id            │    │
                    │  │     - sub claim = human_id            │    │
                    │  │     - scope = narrowest privilege     │    │
                    │  │  5. Per-task JIT token (short TTL)    │    │
                    │  │  6. Every call: dual-attribution audit│    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────┐  ┌─────────────────┐   │
                    │  │  Rate Limiter    │  │  Anomaly Detect │   │
                    │  │  (Redis per      │  │  (behavioral    │   │
                    │  │   agent)         │  │   baseline dev) │   │
                    │  └──────────────────┘  └─────────────────┘   │
                    └──────────────────────────────────────────────┘
```

---

## 5. Agent Identity Model

### Agent vs Human vs Service Account

| Property | Human | Service Account | **AI Agent** |
|----------|-------|-----------------|-------------|
| Identity type | `user` | `service_account` | `agent` |
| Authentication | Password + MFA | Client credentials | **Workload attestation + client credentials** |
| Delegation | N/A | Own identity | **Acts on behalf of user (act claim)** |
| Token scope | User's roles | Fixed service scopes | **Per-task narrowest scope** |
| TTL | Hours (session) | Days (service token) | **Minutes (per-task)** |
| Rate limit | 100/min | 1000/min | **Configurable per agent** |
| Consent | At login | At registration | **Per sensitive operation** |
| Behavioral baseline | Login patterns | API patterns | **Per-operation patterns** |
| Revocation | Disable user | Delete SA | **Revoke all delegated tokens** |

### Agent Registration Flow

```
1. Admin creates agent: POST /api/v1/agents
   - name, description, owner_id
   - allowed_scopes (max scope ceiling)
   - max_delegation_depth (default: 3)
   - rate_limit_per_minute (default: 100)
   - requires_human_consent (default: true for sensitive scopes)

2. Agent receives credentials:
   - client_id + client_secret (for client_credentials flow)
   - OR workload identity (SPIFFE SVID / k8s service account token)

3. Agent attests its workload:
   - POST /agents/{id}/attest
   - Body: { attestation_type: "k8s_sa", token: "..." }
   - System verifies attestation (k8s token review, SPIFFE bundle)

4. Agent is now ACTIVE and can request delegated tokens
```

---

## 6. Delegated Authorization Flow

### RFC 8693 Token Exchange with `act` Claim

```
Agent wants to read invoices on behalf of Alice:

1. Agent authenticates (client_credentials → agent access token)

2. Agent requests delegated token:
   POST /oauth/token
   grant_type=urn:ietf:params:oauth:grant-type:token-exchange
   subject_token=ALICE_USER_TOKEN     (or user_id + consent)
   actor_token=AGENT_ACCESS_TOKEN
   scope=invoices:read
   requested_token_type=urn:ietf:params:oauth:token-type:access_token

3. GGID validates:
   - Agent is registered + active
   - Alice has consented to this agent for invoices:read
   - invoices:read is within agent's allowed_scopes ceiling
   - Rate limit not exceeded
   - (If sensitive) Human-in-the-loop consent request issued

4. GGID issues delegated token with:
   {
     "sub": "alice_user_id",           // Who the agent acts for
     "act": {                          // RFC 8693 act claim
       "sub": "agent_invoice_processor",
       "delegator": "alice_user_id",
       "chain_depth": 1,
       "task_id": "task_abc123",
       "consent_id": "consent_xyz"
     },
     "scope": "invoices:read",         // Narrowest privilege
     "exp": 1700000000,                // Short TTL (15 min)
     "cnf": { "jkt": "..." }           // DPoP binding
   }

5. Agent calls API with delegated token:
   GET /api/v1/invoices
   Authorization: DPoP token
   Dual-attribution: agent_id=agent_invoice_processor, delegator=alice_user_id
```

---

## 7. Endpoint Precondition Check

### Existing Endpoints (Reusable)

| Endpoint | File:Line | Status | Reusable? |
|----------|-----------|--------|-----------|
| `POST /api/v1/oauth/token-exchange-delegation` | `token_exchange_delegation.go:18` | **Works** ✅ | Yes — core delegation |
| Agent consent (in-memory) | `agent_consent_handler.go:18` | **In-memory** ❌ | Replace with DB |
| Scope delegation | `scope_delegation_handler.go` | **Works** | Yes — scope narrowing |
| Delegation chain store | `delegation_pg.go:13` | **DB-backed** ✅ | Yes — chain tracking |
| DPoP validation | `dpop_pg.go` | **DB-backed** ✅ | Yes — token binding |
| RAR + consent preview | `rar_handler.go:203` | **Works** ✅ | Yes — human-readable consent |
| MCP server | `services/mcp/` | **Works** ✅ | Yes — agent tools |
| OAuth token endpoint | `oauth/server.go` | **Works** ✅ | Yes — token exchange |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/agents` | POST | Register agent | P0 |
| `/api/v1/agents` | GET | List agents | P0 |
| `/api/v1/agents/{id}` | GET | Get agent details | P0 |
| `/api/v1/agents/{id}` | PUT | Update agent config | P0 |
| `/api/v1/agents/{id}` | DELETE | Revoke/delete agent | P0 |
| `/api/v1/agents/{id}/attest` | POST | Submit workload attestation | P0 |
| `/api/v1/agents/{id}/consent-requests` | POST | Agent requests consent from user | P0 |
| `/api/v1/agents/{id}/consent-requests` | GET | List pending consent requests | P0 |
| `/api/v1/agents/{id}/consent-requests/{rid}/approve` | POST | User approves consent | P0 |
| `/api/v1/agents/{id}/consent-requests/{rid}/deny` | POST | User denies consent | P0 |
| `/api/v1/agents/{id}/tokens` | GET | List active delegated tokens | P1 |
| `/api/v1/agents/{id}/revoke-all` | POST | Revoke all agent tokens | P0 |
| `/api/v1/agents/{id}/behavior` | GET | Agent behavioral profile | P1 |
| `/api/v1/agents/{id}/rate-limit` | GET/PUT | Per-agent rate config | P1 |

---

## 8. API Design + Curl Commands

### Register Agent

```bash
curl -X POST https://ggid.corp.com/api/v1/agents \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "invoice-processor",
    "description": "Reads and processes invoices from S3",
    "owner_id": "uuid-alice",
    "allowed_scopes": ["invoices:read", "invoices:write"],
    "max_delegation_depth": 3,
    "rate_limit_per_minute": 100,
    "requires_human_consent": true,
    "consent_scopes": ["invoices:write"],
    "token_ttl_minutes": 15
  }'

# Response:
{
  "id": "agt_7f3a2b1c-...",
  "name": "invoice-processor",
  "client_id": "agt_7f3a...",
  "client_secret": "ggid_agt_5f8a3b2c1d4e...",
  "status": "active",
  "created_at": "2026-07-17T10:00:00Z"
}
```

### Agent Requests Consent (DB-backed)

```bash
curl -X POST https://ggid.corp.com/api/v1/agents/agt_7f3a/consent-requests \
  -H "Authorization: Bearer $AGENT_TOKEN" \
  -d '{
    "user_id": "uuid-alice",
    "requested_scopes": ["invoices:write"],
    "resource": "/api/v1/invoices/INV-2026-001",
    "justification": "Processing invoice INV-2026-001 for Q3 reconciliation",
    "task_id": "task_abc123",
    "expires_in_minutes": 30
  }'

# Response:
{
  "request_id": "cr_9e8f7g6h-...",
  "status": "pending_user_approval",
  "user_notified": true,
  "expires_at": "2026-07-17T10:30:00Z"
}
```

### User Approves Consent

```bash
curl -X POST https://ggid.corp.com/api/v1/agents/agt_7f3a/consent-requests/cr_9e8f/approve \
  -H "Authorization: Bearer $USER_TOKEN" \
  -d '{}'

# Response:
{
  "status": "approved",
  "delegated_token": "eyJhbGci...",
  "scope": "invoices:write",
  "expires_at": "2026-07-17T10:15:00Z",
  "task_id": "task_abc123"
}
```

### Get Agent Audit Trail (Dual-Attribution)

```bash
curl "https://ggid.corp.com/api/v1/agents/agt_7f3a/audit?limit=10" \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Response:
{
  "events": [
    {
      "timestamp": "2026-07-17T10:05:00Z",
      "action": "invoices:write",
      "resource": "/api/v1/invoices/INV-2026-001",
      "agent_id": "agt_7f3a",
      "agent_name": "invoice-processor",
      "delegator_id": "uuid-alice",
      "delegator_name": "alice@corp.com",
      "delegation_depth": 1,
      "task_id": "task_abc123",
      "consent_id": "cr_9e8f",
      "result": "success",
      "ip_address": "10.0.1.42"
    }
  ]
}
```

### Revoke All Agent Tokens

```bash
curl -X POST https://ggid.corp.com/api/v1/agents/agt_7f3a/revoke-all \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"reason": "suspected_compromise"}'

# Response:
{
  "revoked_tokens": 7,
  "revoked_consents": 2,
  "status": "revoked",
  "agent_status": "suspended"
}
```

---

## 9. Database Schema

```sql
-- Agent registry
CREATE TABLE agents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    description         TEXT,
    owner_id            UUID NOT NULL,

    -- Credentials
    client_id           VARCHAR(128) NOT NULL UNIQUE,
    client_secret_hash  VARCHAR(256),                 -- bcrypt hash

    -- Scopes
    allowed_scopes      JSONB NOT NULL DEFAULT '[]',  -- Max scope ceiling
    consent_scopes      JSONB NOT NULL DEFAULT '[]',  -- Scopes requiring human consent

    -- Constraints
    max_delegation_depth INT DEFAULT 3,
    rate_limit_per_minute INT DEFAULT 100,
    token_ttl_minutes   INT DEFAULT 15,
    requires_human_consent BOOLEAN DEFAULT true,

    -- Attestation
    attestation_type    VARCHAR(32),                  -- 'k8s_sa', 'spiffe', 'static'
    attestation_data    JSONB,
    attested_at         TIMESTAMPTZ,

    -- State
    status              VARCHAR(16) NOT NULL DEFAULT 'active',
    -- 'active', 'suspended', 'revoked', 'expired'

    -- Audit
    created_by          UUID NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,

    UNIQUE(tenant_id, name)
);

-- Agent consent requests (replaces in-memory handler)
CREATE TABLE agent_consent_requests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    agent_id            UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL,                -- Human delegator

    requested_scopes    JSONB NOT NULL,
    resource            VARCHAR(512),
    justification       TEXT,
    task_id             VARCHAR(128),

    status              VARCHAR(16) NOT NULL DEFAULT 'pending',
    -- 'pending', 'approved', 'denied', 'expired', 'revoked'

    delegated_token_hash VARCHAR(256),                -- Hash of issued token
    token_expires_at    TIMESTAMPTZ,

    decided_at          TIMESTAMPTZ,
    decided_by          UUID,

    expires_at          TIMESTAMPTZ NOT NULL,         -- Consent request TTL
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agent delegation chain tracking
CREATE TABLE agent_delegation_chains (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    chain_id            VARCHAR(128) NOT NULL,        -- Groups entries in same chain

    depth               INT NOT NULL,                 -- 1 = direct, 2 = agent-to-agent
    agent_id            UUID NOT NULL REFERENCES agents(id),
    delegator_agent_id  UUID REFERENCES agents(id),   -- NULL if delegator is human
    delegator_user_id   UUID,                         -- NULL if delegator is agent

    scope               VARCHAR(512) NOT NULL,
    token_hash          VARCHAR(256),
    expires_at          TIMESTAMPTZ NOT NULL,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agent behavioral log (for anomaly detection)
CREATE TABLE agent_behavior_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    agent_id            UUID NOT NULL,
    user_id             UUID,                         -- Delegator (for dual-attribution)
    task_id             VARCHAR(128),

    action              VARCHAR(256) NOT NULL,
    resource            VARCHAR(512),
    method              VARCHAR(8),
    status_code         INT,

    -- Context
    ip_address          VARCHAR(45),
    user_agent          TEXT,

    -- Risk
    risk_score          INT,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Agent behavioral baselines
CREATE TABLE agent_behavior_baselines (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    agent_id            UUID NOT NULL,

    metric              VARCHAR(64) NOT NULL,         -- 'calls_per_hour', 'unique_endpoints', 'error_rate'
    mean                DOUBLE PRECISION NOT NULL,
    stddev              DOUBLE PRECISION NOT NULL,
    p95                 DOUBLE PRECISION,

    last_computed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, agent_id, metric)
);

-- Indexes
CREATE INDEX idx_agents_tenant ON agents (tenant_id, status);
CREATE INDEX idx_agents_client_id ON agents (client_id) WHERE status = 'active';
CREATE INDEX idx_consent_agent_user ON agent_consent_requests (tenant_id, agent_id, user_id, status);
CREATE INDEX idx_consent_pending ON agent_consent_requests (tenant_id, status) WHERE status = 'pending';
CREATE INDEX idx_delegation_chain ON agent_delegation_chains (tenant_id, chain_id, depth);
CREATE INDEX idx_behavior_agent_time ON agent_behavior_log (tenant_id, agent_id, created_at DESC);
CREATE INDEX idx_behavior_dual_attr ON agent_behavior_log (tenant_id, agent_id, user_id, created_at DESC);
CREATE INDEX idx_baselines_agent ON agent_behavior_baselines (tenant_id, agent_id);
```

---

## 10. Multi-Agent Delegation Chains

### Chain Validation Rules

| Rule | Enforcement |
|------|------------|
| **Max depth** | Configurable per agent (default: 3). Chain rejected if depth exceeds. |
| **No cycles** | Agent A → Agent B → Agent A = rejected (detected via visited set) |
| **Scope narrowing** | Each link in chain must have ⊆ scope of parent |
| **Scope ceiling** | Each agent's scope ≤ its `allowed_scopes` |
| **TTL cascade** | Child token TTL ≤ parent token TTL |

### Chain Example

```
Alice (human)
  └─→ Agent A "orchestrator" (depth=1)
        scope: tasks:manage, invoices:read
        └─→ Agent B "invoice-processor" (depth=2)
              scope: invoices:read (narrowed from parent)
              └─→ Agent C "OCR-extractor" (depth=3)
                    scope: invoices:read (same as parent — max depth reached)

JWT act claim (depth=3):
{
  "sub": "alice",
  "act": {
    "sub": "agent_c_ocr",
    "act": {
      "sub": "agent_b_invoice",
      "act": {
        "sub": "agent_a_orchestrator"
      }
    }
  }
}
```

### Cycle Detection

```go
func validateDelegationChain(chain []DelegationEntry) error {
    visited := make(map[string]bool)
    for _, entry := range chain {
        if visited[entry.Actor] {
            return fmt.Errorf("circular delegation detected: agent %s appears twice", entry.Actor)
        }
        visited[entry.Actor] = true
    }
    return nil
}
```

---

## 11. Agent Behavioral Anomaly Detection

### Why Agent Behavior Differs from Human

| Metric | Human Typical | Agent Typical | Anomaly Signal |
|--------|-------------|---------------|---------------|
| Requests/hour | 10-50 | 100-500 | >1000 = suspicious |
| Unique endpoints/hour | 5-20 | 3-10 | >50 = scope scanning |
| Error rate | 2-5% | <1% | >10% = malfunctioning |
| Request interval | Irregular | Regular/predictable | Chaotic = testing |
| User-Agent | Browser | Library/curl | Changes mid-session = hijack |
| Session duration | Minutes-hours | Minutes (per-task) | Hours = persistent token misuse |

### Detection Implementation

```go
func (d *AgentAnomalyDetector) Evaluate(ctx context.Context, agentID uuid.UUID) *AgentRiskAssessment {
    baseline := d.getBaseline(agentID)
    recent := d.getRecentActivity(agentID, 1*time.Hour)

    riskScore := 0
    reasons := []string{}

    // Call velocity anomaly
    if recent.CallCount > baseline.P95("calls_per_hour")*2 {
        riskScore += 30
        reasons = append(reasons, "call_velocity_anomaly")
    }

    // Endpoint diversity anomaly (scope scanning)
    if recent.UniqueEndpoints > baseline.P95("unique_endpoints")*1.5 {
        riskScore += 25
        reasons = append(reasons, "endpoint_diversity_anomaly")
    }

    // Error rate spike
    if recent.ErrorRate > baseline.Mean("error_rate")*3 {
        riskScore += 20
        reasons = append(reasons, "error_rate_spike")
    }

    // Off-schedule activity (agent acting at unusual times)
    if d.isOffSchedule(agentID) {
        riskScore += 15
        reasons = append(reasons, "off_schedule_activity")
    }

    return &AgentRiskAssessment{
        AgentID:   agentID,
        RiskScore: riskScore,
        Reasons:   reasons,
        Action:    decisionFromScore(riskScore), // throttle/alert/suspend
    }
}
```

---

## 12. Implementation Backlog with DoD

### P0 — Agent Registry + DB-Backed Consent (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Agent DB schema | ✅ CREATE TABLE in migration ✅ go build PASS ✅ No in-memory map | 2d |
| 2 | Agent registry + CRUD API | ✅ 6 endpoints registered ✅ DB-backed ✅ curl test PASS ✅ ≥3 tests | 4d |
| 3 | Replace in-memory agent consent | ✅ Uses agent_consent_requests table ✅ No sync.RWMutex ✅ ≥3 tests | 3d |
| 4 | Token exchange integration | ✅ Token exchange issues act claim ✅ Delegated token has agent_id + delegator_id ✅ ≥3 tests | 3d |
| 5 | Workload attestation | ✅ k8s SA token validation ✅ Agent status set to 'attested' ✅ ≥3 tests | 3d |

### P1 — Delegation Chains + Rate Limiting + Audit (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 6 | Multi-agent delegation chain validation | ✅ Max depth enforced ✅ Cycle detection ✅ Scope narrowing validated ✅ ≥3 tests | 4d |
| 7 | Per-agent rate limiting | ✅ Redis counter per agent per minute ✅ 429 on exceed ✅ ≥3 tests | 2d |
| 8 | Dual-attribution audit | ✅ Every agent action logged with agent_id + delegator_id ✅ DB-backed ✅ ≥3 tests | 3d |
| 9 | Agent revocation cascade | ✅ Revoke-all invalidates all tokens + consents ✅ Redis cache flush ✅ ≥3 tests | 2d |

### P2 — Behavioral Detection + Console UI (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 10 | Agent behavioral baselines | ✅ 30-day trailing per-agent baselines ✅ DB-backed ✅ ≥3 tests | 3d |
| 11 | Agent anomaly detection | ✅ Detects velocity/diversity/error anomalies ✅ Auto-throttle on high risk ✅ ≥3 tests | 3d |
| 12 | Console agent management | ✅ Register/list/revoke agents ✅ Consent approval queue ✅ Agent audit trail ✅ ≥3 tests | 4d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 13 | SPIFFE/SPIRE integration | Native SVID workload identity |
| 14 | Per-task token issuance | Ultra-short TTL (1-5 min) per specific task |
| 15 | Agent reputation scoring | Cross-tenant reputation from anomaly patterns |
| 16 | Human-in-the-loop for sensitive ops | Real-time consent push (mobile/web notification) |
| 17 | Agent policy templates | Pre-configured scope bundles per agent type |
| 18 | Agent federation | Cross-org agent delegation with external trust |

---

## 13. Competitive Differentiation

| Feature | GGID (target) | Auth0 | Okta | AWS IAM | Azure | Google WIF |
|---------|---------------|-------|------|---------|-------|-----------|
| **Agent registry** | **DB-backed** | Custom | Custom | IAM Role | App Registration | Service Account |
| **Delegated auth (act claim)** | **RFC 8693** ✅ | Via Actions | Partial | STS AssumeRole | OBO flow | STS |
| **Per-task JIT tokens** | **15-min TTL** | Custom | Custom | STS session | Custom | STS |
| **Multi-agent chains** | **Max depth + cycle detection** | No | No | Role chaining | No | Role chaining |
| **Workload attestation** | **k8s SA + SPIFFE** | No | No | IRSA | Managed Identity | WIF |
| **Human consent (per-op)** | **DB-backed queue** | Actions | No | No | No | No |
| **Dual-attribution audit** | **agent_id + delegator_id** | Custom | Partial | CloudTrail | Activity Log | Audit Logs |
| **Behavioral anomaly** | **Per-agent baselines** | Custom | No | GuardDuty | Defender | Eventarc |
| **Rate limiting per agent** | **Redis-backed** | Custom | API limits | API quotas | API limits | API quotas |
| **Open source** | **Yes (Apache 2.0)** | No | No | No | No | No |

**Key differentiator**: GGID would be the only open-source IAM with native first-class AI agent identity — including workload attestation, per-task JIT delegated tokens, multi-agent delegation chains with cycle detection, and per-agent behavioral anomaly detection. No competitor offers all of these in one system.

---

## 14. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Agent token theft** | DPoP binding + short TTL (15 min) + single-use consent tokens |
| **Scope escalation** | Per-task tokens with narrowest scope; agent's allowed_scopes is ceiling |
| **Delegation cycle abuse** | Cycle detection in chain validation; max depth enforced |
| **Rogue agent** | Behavioral baselines detect anomalous patterns; auto-suspend on critical risk |
| **Consent fatigue** | Consent only for sensitive scopes; non-sensitive operations auto-approved within ceiling |
| **Token replay** | DPoP proof-of-possession required; token hash in audit log for detection |
| **Agent impersonation** | Workload attestation verifies agent binary/runtime; client_secret bcrypt-hashed |
| **Rate limit bypass** | Rate limit keyed by agent_id, not by IP; can't bypass by rotating IP |

---

## References

- [RFC 8693: Token Exchange](https://datatracker.ietf.org/doc/html/rfc8693) — `act` claim for delegation
- [RFC 8705: Mutual TLS](https://datatracker.ietf.org/doc/html/rfc8705) — Client cert auth for agents
- [SPIFFE/SPIRE](https://spiffe.io/) — Workload identity standard
- [OAuth 2.0 for Browser-Based Apps (BCP)](https://datatracker.ietf.org/doc/draft-ietf-oauth-browser-based-apps/) — Agent auth patterns
- [Google Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation) — External workload identity
- [AWS IAM Roles for Service Accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) — k8s workload identity
- [GGID Token Exchange Delegation](../services/oauth/internal/server/token_exchange_delegation.go) — Existing RFC 8693 impl at line 18
- [GGID Agent Consent Handler](../services/oauth/internal/server/agent_consent_handler.go) — In-memory consent at line 18
- [GGID Delegation PG Store](../services/oauth/internal/server/delegation_pg.go) — DB-backed chain store at line 13
- [GGID DPoP Store](../services/oauth/internal/server/dpop_pg.go) — Token binding store
- [GGID RAR Handler](../services/oauth/internal/server/rar_handler.go) — Rich auth requests at line 203
- [GGID MCP Server](../services/mcp/) — Agent tools gateway
- [GGID AI Agent Analysis](./ai-agent-identity-analysis.md) — Theoretical analysis (2108 lines)
- [GGID OAuth 2.1 + MCP + Agent Auth](./oauth21-mcp-agent-auth-pqc-migration.md) — MCP auth migration
