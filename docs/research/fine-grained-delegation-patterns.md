# Fine-Grained Delegation Patterns: User-to-User Permission Delegation for GGID

> **Focus**: A comprehensive delegation framework — letting users grant scoped subsets of their permissions to other users (or services) for limited time, with revocation, audit trails, depth limiting, and per-resource constraints. Covers RFC 8693 `act` claim chains, OAuth 2.0 token exchange, admin impersonation, and user-driven delegation workflows.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `token-exchange-rfc8693.md` covers the protocol. This document covers the **delegation model and UX** — how users manage, grant, revoke, and audit delegated access.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What is Fine-Grained Delegation?](#2-what-is-fine-grained-delegation)
3. [Delegation Patterns](#3-delegation-patterns)
4. [Industry Landscape](#4-industry-landscape)
5. [GGID Current State Analysis](#5-ggid-current-state-analysis)
6. [Gap Analysis](#6-gap-analysis)
7. [Proposed Architecture: Delegation Framework](#7-proposed-architecture-delegation-framework)
8. [Delegation Policy DSL](#8-delegation-policy-dsl)
9. [Database Schema](#9-database-schema)
10. [API Design](#10-api-design)
11. [JWT act Claim Chain](#11-jwt-act-claim-chain)
12. [Security Considerations](#12-security-considerations)
13. [Performance Considerations](#13-performance-considerations)
14. [Console UI Design](#14-console-ui-design)
15. [Competitive Differentiation](#15-competitive-differentiation)
16. [Implementation Backlog](#16-implementation-backlog)

---

## 1. Executive Summary

Delegation lets User A grant User B a **scoped subset** of their permissions — for a limited time, with revocation capability, and a full audit trail of "who acted on behalf of whom." This is essential for:

- **Vacation coverage**: "Let my deputy approve expense reports while I'm away"
- **Task delegation**: "Grant contractor access to Project X for 2 weeks"
- **Admin impersonation**: "Support engineer impersonates user to debug their issue"
- **Service-to-service**: "Microservice A calls Microservice B on behalf of User C"
- **AI agent acting for user**: "AI assistant reads user's documents on their behalf"

GGID already has significant delegation infrastructure:
- `DelegationValidator` with depth limiting, scope narrowing, circular detection, expiry checking (`delegation_validator.go:27`)
- `DelegatePermissions()` for granting permission subsets (`delegation.go:38`)
- `IssueImpersonationToken()` for admin impersonation (`impersonation.go:29`)
- RFC 8693 token exchange with `act` claim chain (`oauth_service.go`)
- Token exchange tests with delegation semantics (`gap_regression_token_exchange_test.go:17`)

However, the implementation is **in-memory only, fragmented, and lacks a unified management layer**:
1. Delegations stored in `globalDelegationStore` (in-memory map) — lost on restart
2. No delegation management API (create/list/revoke via REST)
3. No Console UI for users to manage their delegations
4. No delegation policy (who can delegate what to whom)
5. No delegation audit trail in the database
6. No per-resource delegation (can only delegate permission keys, not specific resources)
7. Delegation not wired into policy evaluator for enforcement

**Recommendation**: Build a **Delegation Framework** that unifies the existing components with PostgreSQL persistence, a management API, Console UI, policy engine, and audit logging.

**Estimated effort**: 3 sprints for MVP (DB persistence + API + policy + evaluator wiring + Console UI).

---

## 2. What is Fine-Grained Delegation?

### Definition

Delegation is the act of **transferring a subset of one's permissions to another principal** (user or service) for a bounded scope and duration. The delegator retains their own permissions; the delegatee gains a scoped subset.

### Key Properties

| Property | Description |
|----------|-------------|
| **Subset (scope narrowing)** | Delegatee gets ≤ delegator's permissions, never more |
| **Time-bounded** | Delegation has start time and expiry |
| **Revocable** | Delegator can revoke at any time |
| **Auditable** | Every action via delegation is logged: "B acted on behalf of A" |
| **Depth-limited** | A→B→C chains limited to configurable max depth (prevents infinite loops) |
| **Circular-safe** | A→B→A cycles detected and rejected |
| **Per-resource** | Delegation scoped to specific resources, not just permission keys |

### The Delegation Triangle

```
                    ┌──────────────────┐
                    │    Delegator     │
                    │    (User A)      │
                    │                  │
                    │ Has permissions: │
                    │  - doc:read      │
                    │  - doc:write     │
                    │  - admin:users   │
                    └────────┬─────────┘
                             │
                    grants scoped subset
                             │
                    ┌────────▼─────────┐
                    │    Delegatee     │
                    │    (User B)      │
                    │                  │
                    │ Receives:        │
                    │  - doc:read      │ (scoped to Project X)
                    │  - doc:write     │ (scoped to Project X)
                    │  Duration: 7 days│
                    │  Revocable by A  │
                    └──────────────────┘
                             │
                    acts on behalf of A
                             │
                    ┌────────▼─────────┐
                    │   Audit Log      │
                    │                  │
                    │ "User B performed│
                    │  doc:write on    │
                    │  doc:report.doc  │
                    │  on behalf of    │
                    │  User A"         │
                    └──────────────────┘
```

---

## 3. Delegation Patterns

### Pattern 1: Vacation / Out-of-Office Delegation

```
User A goes on vacation → delegates "approve:expenses" to User B for 7 days
→ User B can approve expenses as if they were User A
→ JWT contains: { "sub": "B", "act": { "sub": "A" } }
→ On return, User A revokes delegation
```

### Pattern 2: Task-Based Delegation

```
Manager assigns project → delegates "project:X:read,write" to Contractor for 2 weeks
→ Contractor can access project resources
→ Delegation auto-expires after 2 weeks
→ Per-resource: only Project X, not all projects
```

### Pattern 3: Admin Impersonation (Break-Glass)

```
Support admin needs to debug user's issue → impersonates user
→ Admin gets temporary token with user's identity
→ All actions logged as "admin acted as user"
→ Time-limited (15 min), reason required, supervisor may need approval
→ Different from delegation: full identity assumption, not permission subset
```

### Pattern 4: Service-to-Service Delegation

```
Frontend service calls Backend API on behalf of user
→ Frontend has user's token → exchanges for backend-scoped token via RFC 8693
→ New token: { "sub": "user", "act": { "sub": "frontend-service" } }
→ Backend sees: user is acting through frontend-service
→ Scope reduced to only what backend needs
```

### Pattern 5: AI Agent Delegation

```
User authorizes AI assistant to read their documents
→ Delegation: assistant can perform "doc:read" on behalf of user
→ JWT: { "sub": "user", "act": { "sub": "ai-assistant" } }
→ Audit: every document the AI reads is logged as "ai-assistant on behalf of user"
```

---

## 4. Industry Landscape

### Comparison Matrix

| Feature | Okta | Auth0 | AWS IAM | Keycloak | **GGID (existing)** | **GGID (target)** |
|---------|------|-------|---------|----------|---------------------|-------------------|
| **User-to-user delegation** | Yes (delegated admin) | Via Actions | No (roles only) | No | Partial (in-memory) | **Full (DB-persisted)** |
| **Admin impersonation** | Yes | Yes | No | Yes | **Yes** | **Yes** |
| **RFC 8693 token exchange** | Via API | Yes | STS AssumeRole | Partial | **Yes** | **Yes** |
| **Scope narrowing** | Yes | Yes | IAM policies | Yes | **Yes** (validator) | **Yes** |
| **Depth limiting** | Configurable | Custom | Role chain limit | No | **Yes** (validator) | **Yes** |
| **Circular detection** | Yes | Custom | N/A | No | **Yes** (validator) | **Yes** |
| **Per-resource scoping** | Yes | Custom | Resource ARNs | No | No | **Yes** |
| **Self-service UI** | Yes | No | No | No | No | **Yes** |
| **Delegation audit trail** | Yes | Custom | CloudTrail | No | No | **Yes** |
| **Open source** | No | No | No | Yes | **Yes** | **Yes** |

---

## 5. GGID Current State Analysis

### Existing Delegation Infrastructure

| Component | File | Status |
|-----------|------|--------|
| Delegation model | `services/policy/internal/service/delegation.go:14` | **In-memory** — struct + store |
| DelegatePermissions | `services/policy/internal/service/delegation.go:38` | **Implemented** — grant permission subset |
| GetDelegation / ListDelegations | `services/policy/internal/service/delegation.go:73,84` | **Implemented** — in-memory lookup |
| RevokeDelegation | `services/policy/internal/service/delegation.go:117` | **Implemented** — in-memory delete |
| DelegationValidator | `services/policy/internal/service/delegation_validator.go:27` | **Implemented** — depth, scope, expiry, circular |
| Scope narrowing check | `services/policy/internal/service/delegation_validator.go:69` | **Implemented** — `CheckScopeNarrowing()` |
| Circular detection | `services/policy/internal/service/delegation_validator.go:93` | **Implemented** — `CheckCircularDelegation()` |
| Delegation depth check | `services/policy/internal/service/delegation_validator.go:58` | **Implemented** — `CheckDelegationDepth()` |
| Impersonation | `services/auth/internal/service/impersonation.go:29` | **Implemented** — `IssueImpersonationToken()` |
| Token exchange (RFC 8693) | `services/oauth/internal/service/oauth_service.go` | **Implemented** — `act` claim chain |
| Delegation test coverage | `delegation_test.go`, `delegation_validator_test.go` | **Comprehensive** — 13 test cases |
| Gateway delegation route | `services/gateway/internal/router/router.go:337` | **Routed** — `/api/v1/delegation` |
| Delegation handlers | `services/policy/internal/server/delegation_*handler.go` | **Multiple** — 6 handler files |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **In-memory storage** | Delegations lost on service restart |
| 2 | **No DB persistence** | No PostgreSQL tables for delegations |
| 3 | **No delegation policy** | Can't restrict "who can delegate what" |
| 4 | **No per-resource scoping** | Can only delegate permission keys, not specific resources |
| 5 | **No delegation management API** | Users can't self-serve create/revoke via REST |
| 6 | **No delegation in JWT** | Delegation not reflected in access tokens (no `act` claim in access token) |
| 7 | **No delegation audit in DB** | Actions performed via delegation not logged to DB |
| 8 | **No Console UI** | No interface for users to manage delegations |
| 9 | **Not wired to evaluator** | Policy evaluator doesn't check delegation store |
| 10 | **No delegation approval workflow** | Delegations are instant; no approval step for sensitive delegations |

---

## 6. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "User restarts service — are delegations preserved?" | Lost | Persisted in PostgreSQL |
| 2 | "User delegates doc:read for Project X only" | Can't scope to resource | Per-resource delegation |
| 3 | "Policy evaluator checks delegated permissions" | Not wired | Evaluator queries delegation store |
| 4 | "JWT shows 'acting on behalf of'" | Only in token exchange | `act` claim in standard access token |
| 5 | "User views their active delegations in Console" | No UI | Self-service delegation dashboard |
| 6 | "Require manager approval for delegation" | Instant approval | Approval workflow for sensitive delegations |

---

## 7. Proposed Architecture: Delegation Framework

```
                    ┌───────────────────────────────────────────────┐
                    │            Policy Service                      │
                    │                                               │
                    │  ┌─────────────────────────────────────────┐  │
                    │  │     Delegation Framework                │  │
                    │  │                                         │  │
                    │  │  ┌──────────────┐ ┌──────────────────┐ │  │
                    │  │  │ Delegation   │ │ Delegation       │ │  │
                    │  │  │ Manager      │ │ Policy Engine    │ │  │
                    │  │  │ (CRUD)       │ │ (who→what→whom)  │ │  │
                    │  │  └──────┬───────┘ └────────┬─────────┘ │  │
                    │  │         │                  │           │  │
                    │  │  ┌──────┴──────────────────┴────────┐  │  │
                    │  │  │ PostgreSQL Delegation Store      │  │  │
                    │  │  │ (replaces in-memory store)       │  │  │
                    │  │  └─────────────────────────────────┘  │  │
                    │  │                                         │  │
                    │  │  ┌──────────────┐ ┌──────────────────┐ │  │
                    │  │  │ Validator    │ │ Audit Logger     │ │  │
                    │  │  │ (existing)   │ │ (DB-persisted)   │ │  │
                    │  │  └──────────────┘ └──────────────────┘ │  │
                    │  └─────────────────────────────────────────┘  │
                    │                      │                        │
                    │  ┌───────────────────▼───────────────────┐    │
                    │  │ Policy Evaluator                      │    │
                    │  │ (now checks delegation store          │    │
                    │  │  for delegated permissions)           │    │
                    │  └───────────────────────────────────────┘    │
                    └───────────────────────────────────────────────┘
                                       │
                    ┌──────────────────┼──────────────────┐
                    │                  │                  │
                    ▼                  ▼                  ▼
             ┌────────────┐   ┌──────────────┐   ┌────────────┐
             │ OAuth Svc  │   │ Console      │   │ Audit Svc  │
             │ (act claim │   │ (delegation  │   │ (delegation│
             │  in token) │   │  dashboard)  │   │  events)   │
             └────────────┘   └──────────────┘   └────────────┘
```

---

## 8. Delegation Policy DSL

A YAML-based policy that controls who can delegate what to whom:

```yaml
# Per-tenant delegation policy
delegation_policy:
  enabled: true
  max_depth: 3                    # A→B→C→D max chain length
  max_duration: 604800            # 7 days max delegation
  default_duration: 86400         # 24 hours default

  # Who can delegate?
  allowed_delegators:
    - role: manager              # Managers can delegate
    - role: team_lead
    - role: admin

  # What can be delegated?
  delegatable_permissions:
    - "doc:read"
    - "doc:write"
    - "report:approve"
    - "project:*:read"           # Wildcard per-resource

  # What CANNOT be delegated?
  non_delegatable_permissions:
    - "admin:users"              # Admin powers can't be delegated
    - "admin:billing"
    - "security:audit"

  # Sensitive delegations require approval
  approval_required:
    - permission: "report:approve"
      approver_role: "director"
    - permission: "project:*:write"
      approver_role: "manager"

  # Per-resource constraints
  resource_constraints:
    - resource_type: "project"
      max_delegations_per_resource: 5
    - resource_type: "document"
      max_delegations_per_resource: 10
```

---

## 9. Database Schema

```sql
-- Delegations (replaces in-memory store)
CREATE TABLE delegations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    delegator_id        UUID NOT NULL,               -- User A (grants)
    delegatee_id        UUID NOT NULL,               -- User B (receives)

    -- What is delegated
    permissions         JSONB NOT NULL DEFAULT '[]',  -- ["doc:read", "doc:write"]
    resource_type       VARCHAR(64),                  -- "project", "document" (null = all)
    resource_id         VARCHAR(256),                 -- specific resource ID (null = all of type)

    -- Scope constraints
    scopes              JSONB DEFAULT '[]',           -- OAuth scopes being delegated

    -- Time bounds
    starts_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,

    -- State
    status              VARCHAR(32) NOT NULL DEFAULT 'active',  -- 'pending', 'active', 'revoked', 'expired'
    revoked_at          TIMESTAMPTZ,
    revoked_by          UUID,                          -- Who revoked (delegator or admin)
    revoke_reason       TEXT,

    -- Metadata
    reason              TEXT,                          -- "Vacation coverage"
    approval_status     VARCHAR(32) DEFAULT 'approved', -- 'pending', 'approved', 'rejected'
    approver_id         UUID,
    approved_at         TIMESTAMPTZ,

    -- Audit
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Delegation chain (for depth checking)
CREATE TABLE delegation_chains (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delegation_id       UUID NOT NULL REFERENCES delegations(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL,
    chain_depth         INT NOT NULL,                  -- 0 = direct, 1 = re-delegated
    chain_path          JSONB NOT NULL,                -- [{delegator, delegatee}, ...]
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Delegation usage log (actions performed via delegation)
CREATE TABLE delegation_usage_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    delegation_id       UUID NOT NULL REFERENCES delegations(id),
    delegatee_id        UUID NOT NULL,                -- Who acted
    delegator_id        UUID NOT NULL,                -- On whose behalf
    action              VARCHAR(128) NOT NULL,         -- "doc:write"
    resource_type       VARCHAR(64),
    resource_id         VARCHAR(256),
    jwt_jti             VARCHAR(256),                  -- Token ID used
    ip_address          VARCHAR(45),
    user_agent          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Delegation policy (per-tenant config)
CREATE TABLE delegation_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL UNIQUE,
    config_yaml         TEXT NOT NULL,
    enabled             BOOLEAN DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_delegations_tenant ON delegations (tenant_id, status);
CREATE INDEX idx_delegations_delegator ON delegations (tenant_id, delegator_id, status);
CREATE INDEX idx_delegations_delegatee ON delegations (tenant_id, delegatee_id, status);
CREATE INDEX idx_delegations_resource ON delegations (tenant_id, resource_type, resource_id) WHERE resource_id IS NOT NULL;
CREATE INDEX idx_delegations_expiry ON delegations (expires_at) WHERE status = 'active';
CREATE INDEX idx_delegation_usage_delegation ON delegation_usage_log (delegation_id, created_at DESC);
CREATE INDEX idx_delegation_usage_delegatee ON delegation_usage_log (tenant_id, delegatee_id, created_at DESC);
```

---

## 10. API Design

### Delegation Management

```
# Create delegation
POST /api/v1/delegation
{
    "delegatee_id": "uuid",
    "permissions": ["doc:read", "doc:write"],
    "resource_type": "project",
    "resource_id": "project-alpha",
    "scopes": ["project:alpha:read", "project:alpha:write"],
    "duration_seconds": 604800,
    "reason": "Vacation coverage for Alice"
}

Response:
{
    "id": "uuid",
    "status": "active",             // or "pending" if approval required
    "delegator_id": "uuid",
    "delegatee_id": "uuid",
    "permissions": ["doc:read", "doc:write"],
    "resource": { "type": "project", "id": "project-alpha" },
    "expires_at": "2026-07-24T10:00:00Z"
}

# List my delegations (as delegator)
GET /api/v1/delegation?as=delegator&status=active

# List delegations I've received (as delegatee)
GET /api/v1/delegation?as=delegatee&status=active

# Revoke delegation
DELETE /api/v1/delegation/{id}
{
    "reason": "Returned from vacation"
}

# Check if user has delegated permission
POST /api/v1/delegation/check
{
    "delegatee_id": "uuid",
    "permission": "doc:write",
    "resource_type": "project",
    "resource_id": "project-alpha"
}

Response:
{
    "has_permission": true,
    "via_delegation": true,
    "delegation_id": "uuid",
    "delegator_id": "uuid",
    "expires_at": "2026-07-24T10:00:00Z"
}
```

### Impersonation (existing, enhanced)

```
# Admin requests impersonation token
POST /api/v1/auth/impersonate
{
    "target_user_id": "uuid",
    "reason": "Debug user's dashboard rendering issue",
    "duration_minutes": 15
}

Response:
{
    "access_token": "eyJ...",
    "expires_in": 900,
    "impersonating": "user@example.com",
    "reason": "Debug user's dashboard rendering issue"
}
```

---

## 11. JWT act Claim Chain

When a user acts via delegation, their access token includes the `act` claim per RFC 8693:

### Direct User (No Delegation)

```json
{
  "sub": "user-a-uuid",
  "iss": "https://ggid.corp.com/oauth",
  "scope": "doc:read doc:write",
  "tenant_id": "tenant-uuid"
}
```

### Delegated Access (User B acting on behalf of User A)

```json
{
  "sub": "user-a-uuid",           // Original user (whose permissions)
  "act": {
    "sub": "user-b-uuid"          // Acting user (who is performing the action)
  },
  "iss": "https://ggid.corp.com/oauth",
  "scope": "doc:read",            // Narrowed scope (delegated subset only)
  "tenant_id": "tenant-uuid",
  "delegation_id": "delegation-uuid"  // Reference to delegation record
}
```

### Service-to-Service (Frontend calling Backend for User)

```json
{
  "sub": "user-a-uuid",
  "act": {
    "sub": "frontend-service"
  },
  "aud": "backend-api",          // Audience-restricted
  "scope": "api:read",           // Narrowed for backend only
}
```

### Delegation Chain (A → B → C)

```json
{
  "sub": "user-a-uuid",
  "act": {
    "sub": "user-b-uuid",
    "act": {
      "sub": "user-c-uuid"       // C is acting as B who is acting as A
    }
  }
}
```

---

## 12. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Privilege escalation via delegation** | Scope narrowing enforced: delegatee gets ≤ delegator's permissions |
| **Infinite delegation chains** | Max depth (default 3), circular detection (existing `CheckCircularDelegation`) |
| **Stale delegations** | Auto-expiry via `expires_at`; background job marks expired |
| **Delegation after revocation** | Evaluator checks delegation status in real-time (Redis-cached) |
| **Delegation of admin powers** | `non_delegatable_permissions` in policy (admin:users, admin:billing) |
| **Impersonation abuse** | Admin-only, time-limited, reason required, all actions audited |
| **Token replay via delegation** | Delegated tokens have short TTL + `delegation_id` claim for tracking |

---

## 13. Performance Considerations

| Operation | Latency | Notes |
|-----------|---------|-------|
| Create delegation | 3-5ms | INSERT + validation |
| Check delegated permission (Redis cache) | <1ms | Cache hit on active delegations |
| Check delegated permission (DB) | 2-5ms | Cache miss fallback |
| Revoke delegation | 2-3ms | UPDATE status + Redis invalidation |
| List active delegations | 3-5ms | Indexed by delegator/delegatee |
| Policy evaluation with delegation | +2ms overhead | Additional delegation check in evaluator |

---

## 14. Console UI Design

### Delegation Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  My Delegations                                                 │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │  Granted By Me │  │  Granted To Me │  │  Total Active  │     │
│  │  (Delegator)   │  │  (Delegatee)   │  │  Delegations   │     │
│  │  3 active      │  │  2 active      │  │  5             │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│                                                                  │
│  Granted By Me (I delegated my permissions)                      │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ → Bob Smith    doc:read, doc:write  Project Alpha          │  │
│  │   Expires: Jul 24   Status: Active   [Revoke]              │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ → Carol Jones  report:approve       All Reports            │  │
│  │   Expires: Jul 20   Status: Active   [Revoke]              │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Granted To Me (Others delegated to me)                          │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ ← Alice Chen   doc:read, doc:write  Project Alpha          │  │
│  │   Expires: Jul 24   From: Alice's vacation coverage        │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ← Dave Wilson  admin:dashboard       Global                │  │
│  │   Expires: Jul 18   From: Conference coverage              │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  + New Delegation                                                │
│    [Select user] [Select permissions] [Resource] [Duration]     │
│                                                                  │
│  Recent Delegation Activity                                      │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ 10:15  Bob S.   used doc:write  Project Alpha  via Alice   │  │
│  │ 09:45  Carol J. used report:approve  Report Q3  via Alice  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 15. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Keycloak |
|---------|---------------|------|-------|----------|
| **User-to-user delegation** | **Full + per-resource** | Delegated admin | Custom | No |
| **Delegation policy DSL** | **YAML declarative** | Visual config | Actions (JS) | No |
| **Per-resource scoping** | **Yes** | Limited | Custom | No |
| **Delegation approval** | **Yes (workflow)** | No | Custom | No |
| **Self-service UI** | **Yes (dashboard)** | Admin only | No | No |
| **Depth + circular checks** | **Yes (existing)** | Yes | Custom | No |
| **act claim in JWT** | **Yes** | Yes | Yes | Partial |
| **Delegation audit trail** | **DB-persisted** | Yes | Custom | No |
| **Open source** | **Yes (Apache 2.0)** | No | No | Yes |

**Key differentiator**: GGID would be the only open-source IAM with user-to-user delegation + per-resource scoping + delegation policy DSL + approval workflow + self-service Console UI.

---

## 16. Implementation Backlog

### P0 — Persistence + API + Evaluator Wiring (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | Delegation DB schema | PostgreSQL tables for delegations, chains, usage log, policy | 2 days |
| 2 | Delegation repository | Replace in-memory store with PostgreSQL-backed repo | 3 days |
| 3 | Delegation management API | REST CRUD: create, list, revoke, check | 3 days |
| 4 | Policy evaluator integration | Wire delegation check into Check() pipeline | 2 days |
| 5 | Delegation policy engine | Parse YAML policy, enforce who→what→whom rules | 3 days |
| 6 | Per-resource scoping | Support resource_type + resource_id in delegation | 2 days |
| 7 | Auto-expiry | Background job marks expired delegations | 1 day |
| 8 | Unit tests | 90%+ coverage for repo, policy, API | 3 days |

### P1 — Enhanced Features (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 9 | JWT act claim injection | Add `act` claim to access tokens when acting via delegation | 3 days |
| 10 | Delegation audit logging | Log all delegated actions to delegation_usage_log | 2 days |
| 11 | Approval workflow | Sensitive delegations require manager/director approval | 3 days |
| 12 | Delegation Redis cache | Cache active delegations for <1ms evaluator lookups | 2 days |
| 13 | Integration tests | End-to-end delegation + evaluation + revocation | 3 days |

### P2 — Console UI (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 14 | Delegation dashboard | "Granted by me" / "Granted to me" cards + lists | 3 days |
| 15 | Create delegation wizard | User picker, permission selector, resource scope, duration | 3 days |
| 16 | Delegation activity log | Recent delegated actions with delegator context | 2 days |
| 17 | Admin delegation policy editor | YAML editor with live validation | 2 days |
| 18 | Impersonation controls (admin) | Enhanced impersonation panel with audit view | 2 days |

### P3 — Advanced Features (Future)

| # | Task | Description |
|---|------|-------------|
| 19 | Delegation templates | Pre-configured delegation presets ("vacation coverage") |
| 20 | Bulk delegation | Delegate to multiple users at once (group delegation) |
| 21 | Delegation chain visualization | Visual graph of A→B→C delegation chains |
| 22 | Cross-tenant delegation | Delegate to users in partner organizations |
| 23 | Delegation analytics | Most-delegated permissions, average duration, usage frequency |
| 24 | OAuth scope delegation | Let users delegate specific OAuth scopes to third-party apps |

---

## References

- [RFC 8693: OAuth 2.0 Token Exchange](https://www.rfc-editor.org/rfc/rfc8693) — `act` claim and delegation semantics
- [RFC 8707: Resource Indicators](https://www.rfc-editor.org/rfc/rfc8707) — Audience restriction for delegated tokens
- [Okta Delegated Administration](https://help.okta.com/en-us/Content/Topics/Security/delegated-admin/delegated-admin.htm) — Admin delegation model
- [AWS IAM Role Chaining](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_terms-and-concepts.html) — STS AssumeRole chains
- [GGID Delegation Validator](../services/policy/internal/service/delegation_validator.go) — Existing depth/scope/circular checks at line 27
- [GGID Delegation Service](../services/policy/internal/service/delegation.go) — In-memory delegation store at line 14
- [GGID Impersonation](../services/auth/internal/service/impersonation.go) — Admin impersonation at line 29
- [GGID Token Exchange Tests](../services/oauth/internal/service/gap_regression_token_exchange_test.go) — RFC 8693 delegation flow tests
