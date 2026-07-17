# ReBAC: Relationship-Based Access Control for GGID

> **Focus**: Adding Google Zanzibar-style fine-grained authorization to complement GGID's existing RBAC + ABAC policy engine.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-15 | **Status**: Research Complete

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What is ReBAC?](#2-what-is-rebac)
3. [ReBAC vs RBAC vs ABAC](#3-rebac-vs-rbac-vs-abac)
4. [The Google Zanzibar Model](#4-the-google-zanzibar-model)
5. [Industry Landscape](#5-industry-landscape)
6. [GGID Gap Analysis](#6-ggid-gap-analysis)
7. [Proposed Architecture](#7-proposed-architecture)
8. [Schema Design for GGID](#8-schema-design-for-ggid)
9. [API Design](#9-api-design)
10. [Performance Considerations](#10-performance-considerations)
11. [Migration Strategy](#11-migration-strategy)
12. [Competitive Differentiation](#12-competitive-differentiation)
13. [Implementation Backlog](#13-implementation-backlog)

---

## 1. Executive Summary

GGID's Policy service currently supports RBAC (role-permission checks with inheritance) and ABAC (attribute-based policy evaluation with deny override). This covers the majority of enterprise authorization use cases but has a critical gap: **fine-grained, relationship-aware permission checks** ("Can user X edit document Y because they are a member of the project that owns it?").

**ReBAC (Relationship-Based Access Control)**, based on Google's Zanzibar paper, solves this by modeling permissions as a graph of typed relationships (tuples). Instead of asking "what role does this user have?", ReBAC asks "how is this user related to this resource?" — enabling hierarchical inheritance, group-based sharing, and resource-ownership patterns at scale.

**Recommendation**: Add a ReBAC tuple store and graph-traversal engine to the Policy service as an optional third authorization layer, evaluated after RBAC allow and before final default-deny. This makes GGID one of the few IAM platforms offering all three models (RBAC + ABAC + ReBAC) in a unified evaluation pipeline.

**Estimated effort**: 3-4 sprints for MVP (tuple store, check API, schema DSL, integration with evaluator).

---

## 2. What is ReBAC?

Relationship-Based Access Control (ReBAC) determines access permissions through the **connections between entities** rather than through role assignments or attribute evaluations. The core idea: a user's permission on a resource is derived from their position in a graph of relationships.

### Core Concepts

| Concept | Description | Example |
|---------|-------------|---------|
| **Object** | A typed entity in the system | `document:quarterly-report` |
| **Subject** | The entity requesting access (user or group) | `user:alice` |
| **Relation** | A typed relationship between subject and object | `owner`, `editor`, `viewer`, `member` |
| **Tuple** | A stored fact: (subject, relation, object) | `user:alice, owner, document:report` |
| **Permission** | A computed rule derived from relations | `can_edit = editor + owner` |
| **Schema** | Type definitions declaring relations and permissions | `definition document { ... }` |

### Key Properties

- **Graph-based**: Permissions are computed by traversing a relationship graph, not by looking up a flat table.
- **Inheritance**: Permissions cascade through hierarchies naturally — folder → document, org → project → resource.
- **Composability**: Relations can reference other objects (`editor from parent_folder`), enabling transitive permissions.
- **Delegation**: Users can be granted permission on specific resources without role assignment.

---

## 3. ReBAC vs RBAC vs ABAC

| Dimension | RBAC | ABAC | ReBAC |
|-----------|------|------|-------|
| **Decision basis** | Role membership | Subject + Resource + Environment attributes | Relationship graph traversal |
| **Granularity** | Coarse (role-level) | Fine (attribute-level) | Very fine (relationship-level, per-resource) |
| **Hierarchical inheritance** | Limited (role hierarchy) | No | Native (folder → document, org → project) |
| **Resource ownership** | Via role on resource | Via resource attribute | First-class (`owner` relation) |
| **Group-based sharing** | Via role assignment | Via group attribute | Native (`member` relation) |
| **Dynamic context** | No | Yes (time, IP, device) | Limited (caveats in SpiceDB) |
| **Query pattern** | "Does user have role X?" | "Do attributes match policy?" | "Is user related to resource?" |
| **Best for** | Broad permission tiers | Contextual constraints | Fine-grained resource sharing |
| **Scalability concern** | Role explosion | Attribute resolution latency | Tuple explosion |
| **GGID status** | **Implemented** | **Implemented** | **Missing** |

### When Each Model Wins

**RBAC** is best for: "All admins can manage users. All editors can create content." — stable, coarse-grained tiers.

**ABAC** is best for: "Users can access financial records only during business hours from managed devices." — dynamic, context-dependent rules.

**ReBAC** is best for: "Alice can edit this document because she's the owner; Bob can view it because he's a member of the project that owns it; Carol can comment because she was directly granted commenter access." — per-resource, relationship-aware permissions.

### Why GGID Needs All Three

Real-world enterprise authorization requires all three working together:

```
1. Deny override (ABAC):  "Block access if user is on a flagged IP"
2. RBAC baseline:         "User has admin role → broad permissions"
3. ReBAC fine-grained:    "User is owner of this specific document → full access"
4. ABAC conditions:       "But only during business hours"
5. Default deny
```

---

## 4. The Google Zanzibar Model

Google's Zanzibar paper (2019) is the foundational design for modern ReBAC systems. It powers Google Drive, YouTube, Calendar, and Cloud IAM.

### Tuple Format

Zanzibar stores relationships as tuples:

```
⟨object, relation, subject⟩
```

Examples:
```
⟨doc:budget, owner, user:alice⟩          # Alice owns the budget doc
⟨doc:budget, viewer, group:finance⟩      # Finance group can view it
⟨doc:budget, parent, folder:q4-reports⟩   # Budget doc is in Q4 folder
⟨folder:q4-reports, viewer, group:leadership⟩  # Leadership can view Q4 folder
```

### Permission Computation

Permissions are computed from the relation graph using **set algebra** (union, intersection, difference):

```
# In SpiceDB/OpenFGA syntax
permission view = viewer + editor + parent->view
permission edit = editor + owner
permission delete = owner
```

When checking "can user:carol view doc:budget?":

1. Look up direct viewers of `doc:budget` → `group:finance`
2. Is `user:carol` in `group:finance`? → If yes, ALLOW
3. Check parent folder: `doc:budget` → `folder:q4-reports`
4. Look up viewers of `folder:q4-reports` → `group:leadership`
5. Is `user:carol` in `group:leadership`? → If yes, ALLOW

### Consistency Model (Zookies/ZedTokens)

Zanzibar uses **Zookies** (now called ZedTokens in SpiceDB) to provide tunable consistency:

- **Minimize latency**: Read from nearest replica with eventual consistency
- **Maximize consistency**: Wait for all replicas to converge before answering
- **Selective consistency**: Use a token from a recent write to ensure the check reflects that write

This is critical for authorization: a revoked permission must take effect immediately, even in a distributed system.

---

## 5. Industry Landscape

### OpenFGA (Auth0/Okta)

| Attribute | Value |
|-----------|-------|
| **License** | Apache 2.0 |
| **Maintainer** | Auth0/Okta (now Okta) |
| **Schema DSL** | Model language (`.fga` files) |
| **API** | gRPC + REST |
| **SDKs** | Go, JS, Python, Java, C#, .NET, Rust, PHP (8+ languages) |
| **Storage** | PostgreSQL, MySQL, SQLite |
| **Multi-tenancy** | Stores (isolated data per tenant) |
| **Consistency** | Eventual (with consistency tuning) |
| **Watch API** | No (planned) |
| **Caveats** | Conditions (beta) |
| **Managed Cloud** | Okta FGA |
| **Best for** | Teams in Auth0/Okta ecosystem |

**Model example (OpenFGA DSL)**:
```
model
  schema 1.1

type user

type organization
  relations
    define owner: [user]
    define admin: [user] or owner
    define member: [user] or admin

type document
  relations
    define parent: [folder]
    define owner: [user]
    define editor: [user] or owner
    define viewer: [user] or editor or viewer from parent
    define can_edit: editor
    define can_view: viewer
    define can_delete: owner or admin from organization
```

### SpiceDB (AuthZed)

| Attribute | Value |
|-----------|-------|
| **License** | Apache 2.0 |
| **Maintainer** | AuthZed |
| **Schema** | Zed schema (closest to Google Zanzibar) |
| **API** | gRPC + REST |
| **SDKs** | Go, JS, Python, Java, Rust, .NET |
| **Storage** | PostgreSQL, CockroachDB, MySQL, Spanner |
| **Multi-tenancy** | Namespaces (not true isolation) |
| **Consistency** | Configurable (ZedTokens for strong consistency) |
| **Watch API** | Yes (real-time cache invalidation) |
| **Caveats** | Yes (conditional permissions with CEL expressions) |
| **Managed Cloud** | AuthZed Dedicated |
| **Best for** | Zanzibar purists, strong consistency requirements |

**Key advantage**: SpiceDB's Watch API enables real-time cache invalidation — critical for high-throughput systems where stale permission caches are unacceptable.

### Permify

| Attribute | Value |
|-----------|-------|
| **License** | Apache 2.0 |
| **Maintainer** | Permify |
| **Schema** | YAML-based |
| **API** | gRPC + REST |
| **SDKs** | Go, JS, Python, Java |
| **Storage** | PostgreSQL, in-memory |
| **Multi-tenancy** | Built-in (first-class) |
| **Consistency** | Snapshot tokens |
| **Watch API** | No |
| **Visual Playground** | Yes (built-in UI) |
| **Best for** | Developer experience, fast iteration, multi-tenant SaaS |

### Ory Keto

| Attribute | Value |
|-----------|-------|
| **License** | Apache 2.0 |
| **Maintainer** | Ory |
| **Schema** | OPL (Ory Permission Language, based on TypeScript) |
| **Status** | Production-ready (v0.12+) |
| **Best for** | Teams already using Ory ecosystem (Kratos, Hydra) |

### Summary Comparison

| Feature | OpenFGA | SpiceDB | Permify | Ory Keto |
|---------|---------|---------|---------|----------|
| Zanzibar fidelity | High | Highest | High | Medium |
| Schema format | DSL | Zed | YAML | OPL |
| Strong consistency | Partial | Full | Partial | Partial |
| Watch API | No | Yes | No | No |
| Caveats/Conditions | Beta | Yes | ABAC rules | No |
| Multi-tenancy | Stores | Namespaces | Built-in | No |
| CockroachDB support | No | Yes | No | No |
| Community size | Large | Medium | Growing | Medium |

---

## 6. GGID Gap Analysis

### Current Authorization Architecture

GGID's Policy service implements a **hybrid RBAC + ABAC** evaluator:

```
CheckRequest → 
  1. Resolve user roles (with inheritance) → collect permissions
  2. Check if any permission matches (resource_type, action) → RBAC allow
  3. Evaluate ABAC policies (deny override) → may deny or allow
  4. Default deny
```

### What This Cannot Express

| Scenario | Why It Fails | ReBAC Solution |
|----------|-------------|----------------|
| "Alice can edit document X because she owns it" | No per-document ownership concept in RBAC | `⟨document:X, owner, user:alice⟩` tuple |
| "Bob can view all documents in folder Y" | No folder-to-document inheritance | `permission view = viewer + parent->view` |
| "Finance team members can access financial reports" | Group-based resource access not modeled | `⟨report:q4, viewer, group:finance⟩` |
| "Project members can create issues in their project" | No project-membership → resource-permission link | `permission create_issue = member from project` |
| "Document shared with specific user (not role-based)" | No ad-hoc sharing | Direct `viewer` tuple on document |
| "Department head can manage department resources" | No transitive ownership through hierarchy | `permission manage = head from department` |

### Current Workarounds (And Why They're Bad)

1. **Creating a role per resource**: "doc-X-editor" role → role explosion (N×M roles)
2. **Embedding ownership in resource attributes**: Custom logic per service → inconsistent enforcement
3. **Application-level checks**: `if doc.OwnerID == userID` → authorization logic scattered across services, not centralized, not auditable

---

## 7. Proposed Architecture

### Integration Point

Add ReBAC as a **third evaluation layer** in the Policy service's evaluator:

```
CheckRequest →
  1. Check explicit DENY policies (ABAC)         → if deny, DENY
  2. Resolve user roles → check permissions (RBAC) → if allow, candidate
  3. Evaluate ABAC allow policies with conditions → if match, ALLOW
  4. ★ NEW: Check ReBAC relationship graph ★     → if allow, ALLOW
  5. Default DENY
```

### Component Architecture

```
                    ┌─────────────────────────────────┐
                    │      Policy Service             │
                    │      (Evaluator)                │
                    └────────┬────────────────────────┘
                             │
           ┌─────────────────┼─────────────────┐
           │                 │                  │
           ▼                 ▼                  ▼
    ┌─────────────┐  ┌──────────────┐  ┌──────────────┐
    │  RBAC       │  │  ABAC        │  │  ReBAC       │
    │  Engine     │  │  Engine      │  │  Engine      │
    │             │  │              │  │              │
    │ Roles →     │  │ Attributes   │  │ Tuples →     │
    │ Permissions │  │ → Policies   │  │ Graph →      │
    │             │  │              │  │ Permissions  │
    └──────┬──────┘  └──────┬───────┘  └──────┬───────┘
           │                │                  │
           ▼                ▼                  ▼
    ┌─────────────┐  ┌──────────────┐  ┌──────────────┐
    │ role_repo   │  │ policy_repo  │  │ tuple_store  │
    │ (Postgres)  │  │ (Postgres)   │  │ (Postgres)   │
    └─────────────┘  └──────────────┘  └──────────────┘
```

### Tuple Store Schema (PostgreSQL)

```sql
-- Relationship tuples store
CREATE TABLE rebac_tuples (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL,
    -- The object being related to
    object_type VARCHAR(64) NOT NULL,
    object_id   VARCHAR(256) NOT NULL,
    -- The relation type
    relation    VARCHAR(64) NOT NULL,
    -- The subject (can be a user, group, or another object)
    subject_type   VARCHAR(64) NOT NULL,
    subject_id     VARCHAR(256) NOT NULL,
    subject_relation VARCHAR(64),  -- For usersets: "group:engineering#member"
    -- Metadata
    caveat      JSONB,  -- Optional caveat context
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at  TIMESTAMPTZ,
    
    UNIQUE(tenant_id, object_type, object_id, relation, subject_type, subject_id, subject_relation)
);

-- Indexes for efficient graph traversal
CREATE INDEX idx_rebac_lookup ON rebac_tuples (tenant_id, object_type, object_id, relation);
CREATE INDEX idx_rebac_subject ON rebac_tuples (tenant_id, subject_type, subject_id);
CREATE INDEX idx_rebac_tenant ON rebac_tuples (tenant_id);

-- Schema definitions (authorization models)
CREATE TABLE rebac_schemas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    version     INT NOT NULL,
    definition  JSONB NOT NULL,  -- Parsed schema definition
    raw_text    TEXT NOT NULL,    -- Original schema text
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, version)
);
```

### Graph Traversal Algorithm

```go
// Simplified recursive relationship check
func (e *RebacEvaluator) Check(
    ctx context.Context,
    tenantID uuid.UUID,
    objectType, objectID string,
    permission string,
    subjectType, subjectID string,
) (bool, error) {
    // 1. Look up tuples matching (object, permission)
    tuples, err := e.store.Read(ctx, tenantID, objectType, objectID, permission)
    if err != nil {
        return false, err
    }
    
    // 2. For each tuple, check if subject matches
    for _, t := range tuples {
        // Direct user match
        if t.SubjectType == subjectType && t.SubjectID == subjectID {
            return true, nil
        }
        
        // Userset match: subject is a group, check membership
        if t.SubjectType == "group" {
            member, err := e.Check(ctx, tenantID, 
                t.SubjectType, t.SubjectID,
                t.SubjectRelation,  // e.g., "member"
                subjectType, subjectID)
            if err != nil {
                return false, err
            }
            if member {
                return true, nil
            }
        }
        
        // Computed userset: traverse to parent object
        if t.SubjectType == objectType && t.SubjectID != objectID {
            // Recursive: check if subject has the permission on the parent
            allowed, err := e.Check(ctx, tenantID,
                t.SubjectType, t.SubjectID,
                permission,
                subjectType, subjectID)
            if err != nil {
                return false, err
            }
            if allowed {
                return true, nil
            }
        }
    }
    
    return false, nil
}
```

---

## 8. Schema Design for GGID

### GGID Authorization Model (Zanzibar-style)

```
definition user {}

definition organization {
    relation owner: user
    relation admin: user
    relation member: user | organization#member  // nested orgs
    
    permission manage = owner
    permission administer = admin + owner
    permission view = member + administer
}

definition department {
    relation org: organization
    relation head: user
    relation member: user
    
    permission manage = head + org->manage
    permission administer = head + org->administer
    permission view = member + org->view
}

definition team {
    relation dept: department
    relation lead: user
    relation member: user
    
    permission manage = lead + dept->manage
    permission view = member + dept->view
}

definition role {
    relation tenant: organization
    relation assignee: user
    
    permission assign = tenant->administer
    permission revoke = tenant->administer
}

definition resource {
    relation owner: user
    relation org: organization
    relation dept: department
    relation team: team
    
    permission manage = owner + org->administer
    permission view = owner + org->view + dept->view + team->view
    permission edit = owner + org->administer
    permission delete = owner + org->manage
}

definition document {
    relation parent: folder | resource
    relation owner: user
    relation editor: user
    relation viewer: user
    relation commenter: user
    
    permission edit = owner + editor + parent->edit
    permission comment = commenter + edit
    permission view = viewer + comment + parent->view
    permission delete = owner + parent->manage
    permission share = owner + editor
}

definition folder {
    relation parent: folder | resource
    relation org: organization
    relation owner: user
    relation editor: user
    relation viewer: user
    
    permission edit = owner + editor + parent->edit
    permission view = viewer + edit + parent->view + org->view
    permission manage = owner + org->administer
}
```

### Mapping Existing RBAC to ReBAC

The existing RBAC model maps cleanly:

| RBAC Concept | ReBAC Equivalent |
|-------------|-----------------|
| `UserRole{UserID, RoleID, ScopeOrg, OrgID}` | `⟨organization:OrgID, member, user:UserID⟩` |
| `UserRole{UserID, RoleID, ScopeDept, DeptID}` | `⟨department:DeptID, member, user:UserID⟩` |
| `Role.admin → Permission("users", "manage")` | `permission administer = admin + owner` |
| Role inheritance (`ParentRoleID`) | Computed permissions referencing parent relations |
| `Policy{Effect:Deny}` | Caveats or explicit deny in schema |

---

## 9. API Design

### Check Permission

```
POST /api/v1/policy/rebac/check
Content-Type: application/json

{
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "object": {
        "type": "document",
        "id": "quarterly-report"
    },
    "permission": "can_edit",
    "subject": {
        "type": "user",
        "id": "alice@example.com"
    },
    "context": {
        "time": "2026-07-15T14:30:00Z",
        "ip": "10.0.1.50"
    }
}

Response:
{
    "allowed": true,
    "matched_by": "editor (direct tuple)",
    "evaluated_at": "2026-07-15T14:30:00.001Z"
}
```

### Write Relationship Tuple

```
POST /api/v1/policy/rebac/tuples
Content-Type: application/json

{
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "tuples": [
        {
            "object": {"type": "document", "id": "spec"},
            "relation": "editor",
            "subject": {"type": "user", "id": "bob@example.com"}
        },
        {
            "object": {"type": "document", "id": "spec"},
            "relation": "parent",
            "subject": {"type": "folder", "id": "project-alpha"}
        }
    ]
}
```

### List Objects (What can this user access?)

```
POST /api/v1/policy/rebac/list-objects

{
    "tenant_id": "...",
    "object_type": "document",
    "permission": "can_view",
    "subject": {"type": "user", "id": "alice@example.com"}
}

Response:
{
    "objects": ["document:spec", "document:design", "document:budget"]
}
```

### List Subjects (Who has access?)

```
POST /api/v1/policy/rebac/list-subjects

{
    "tenant_id": "...",
    "object": {"type": "document", "id": "spec"},
    "permission": "can_edit",
    "subject_type": "user"
}

Response:
{
    "subjects": ["user:alice", "user:bob", "user:carol"]
}
```

### Schema Management

```
PUT /api/v1/policy/rebac/schema
Content-Type: text/plain

definition user {}
definition document {
    relation owner: user
    relation editor: user
    relation viewer: user
    permission edit = owner + editor
    permission view = viewer + edit
}
```

---

## 10. Performance Considerations

### Expected Latency

| Check Complexity | Expected Latency | Example |
|-----------------|-----------------|---------|
| Direct tuple lookup | < 1ms | "Is user owner of document?" |
| 1-hop traversal | 1-3ms | "Is user member of group that has access?" |
| Multi-hop traversal (3+ hops) | 5-15ms | "Is user member of org that owns the folder containing the document?" |
| Deep graph (10+ hops) | 20-50ms | Complex organizational hierarchy |

### Optimization Strategies

1. **Tuple Caching (Redis)**: Cache frequently-accessed tuples and check results with short TTL (5-30s). Cache invalidation on tuple write.

2. **Memoized Graph Traversal**: Within a single check, memoize visited nodes to avoid redundant lookups in cyclic graphs.

3. **Materialized Permission Views**: Pre-compute flattened permission sets for common patterns ("all documents user X can view") via background jobs.

4. **Bounded Depth Limit**: Enforce max traversal depth (default 25, configurable) to prevent infinite loops and bound latency.

5. **Batch Check API**: Accept multiple check requests in a single call to amortize connection overhead.

6. **Read Replicas**: Route check queries to read replicas for throughput; writes go to primary.

### Storage Estimation

| Scale | Tuples | Storage | Notes |
|-------|--------|---------|-------|
| Small (1K users, 10K resources) | ~100K | ~50MB | Most tuples are group memberships |
| Medium (10K users, 100K resources) | ~5M | ~2GB | Document sharing adds significant tuples |
| Large (100K users, 1M resources) | ~50M | ~20GB | Folder hierarchy adds parent relations |
| Enterprise (1M users, 10M resources) | ~500M | ~200GB | Needs partitioning by tenant_id |

### PostgreSQL-Specific Optimizations

- **Partition by tenant_id**: Each tenant's tuples in a separate partition for isolation and query pruning
- **BRIN index on created_at**: Efficient tuple expiry/cleanup
- **Connection pooling**: PgBouncer or pgxpool with prepared statements
- **EXPLAIN ANALYZE monitoring**: Ensure graph traversal queries use indexes, not seq scans

---

## 11. Migration Strategy

### Phase 1: Shadow Mode (No Enforcement)

1. Deploy ReBAC engine alongside existing evaluator
2. Sync existing RBAC data to tuples (roles → org memberships, scopes → relations)
3. On every Check() call, also run ReBAC check but don't use the result
4. Log discrepancies between RBAC decision and ReBAC decision
5. Fix schema/model until discrepancies converge to zero

### Phase 2: Opt-In Mode

1. Add per-resource-type opt-in flag: `rebac_enabled document = true`
2. For enabled types, ReBAC check runs as an additional allow source
3. RBAC still works as fallback — no breaking changes
4. Gradually enable for more resource types

### Phase 3: Full Integration

1. ReBAC becomes a first-class evaluation layer
2. New resource types default to ReBAC for fine-grained permissions
3. RBAC remains for coarse-grained administrative permissions
4. ABAC remains for contextual constraints (time, IP, device posture)

### Data Migration

```sql
-- Migrate existing user-role assignments to ReBAC tuples
INSERT INTO rebac_tuples (tenant_id, object_type, object_id, relation, subject_type, subject_id)
SELECT 
    ur.tenant_id,
    'organization', ur.scope_id::text,
    'member',
    'user', ur.user_id::text
FROM user_roles ur
WHERE ur.scope_type = 'organization';

-- Migrate department-scoped roles
INSERT INTO rebac_tuples (tenant_id, object_type, object_id, relation, subject_type, subject_id)
SELECT 
    ur.tenant_id,
    'department', ur.scope_id::text,
    'member',
    'user', ur.user_id::text
FROM user_roles ur
WHERE ur.scope_type = 'department';
```

---

## 12. Competitive Differentiation

### How ReBAC Makes GGID Stand Out

| Platform | RBAC | ABAC | ReBAC | Unified Pipeline |
|----------|------|------|-------|-----------------|
| **GGID** (proposed) | Yes | Yes | Yes | Yes — single evaluator |
| Auth0/Okta | Yes | Partial | Via OpenFGA (separate) | No — separate products |
| Keycloak | Yes | Partial | No | N/A |
| Ory Stack | No | No | Via Keto | No — separate services |
| AWS IAM | Yes | No | Partial (resource policies) | Partial |
| Azure AD | Yes | CA policies | Partial | Partial |
| Clerk | Yes | No | No | N/A |
| Casdoor | Yes | No | No | N/A |

**Key differentiator**: GGID would be the **only open-source IAM platform** offering RBAC + ABAC + ReBAC in a single unified evaluation pipeline, with a single Check API and a single decision log.

### Market Positioning

- **Enterprise**: ReBAC enables document-level sharing, folder hierarchies, and project-based access without role explosion
- **SaaS multi-tenant**: Built-in multi-tenancy through `tenant_id` partitioning
- **Developer-friendly**: Schema DSL is declarative and auditable
- **Compliance-ready**: Every check is logged with full decision trace

---

## 13. Implementation Backlog

### P0 — Core Engine (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | Tuple store | PostgreSQL tables + repository layer | 3 days |
| 2 | Schema DSL parser | Parse Zanzibar-style schema definitions | 4 days |
| 3 | Graph traversal engine | Recursive check algorithm with depth limiting | 5 days |
| 4 | Check API | REST + gRPC endpoints for permission checks | 2 days |
| 5 | Write/Delete API | Tuple CRUD endpoints | 2 days |
| 6 | Evaluator integration | Wire ReBAC into existing Check() pipeline | 2 days |
| 7 | Unit tests | 90%+ coverage for engine, parser, store | 3 days |

### P1 — Enhanced Features (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 8 | ListObjects API | "What can this user access?" reverse lookup | 3 days |
| 9 | ListSubjects API | "Who has access to this resource?" | 3 days |
| 10 | Redis caching | Tuple cache with write invalidation | 2 days |
| 11 | Caveats support | Conditional permissions (IP allowlist, time-based) | 4 days |
| 12 | Batch check | Multiple checks in single API call | 2 days |
| 13 | Schema versioning | Track schema versions, atomic updates | 2 days |

### P2 — Operational (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 14 | Console UI | ReBAC schema editor, tuple browser, check playground | 5 days |
| 15 | Migration tooling | RBAC→ReBAC tuple sync, shadow mode | 3 days |
| 16 | Watch API | Real-time tuple change notifications for cache invalidation | 4 days |
| 17 | Metrics & monitoring | Check latency, cache hit rate, tuple count dashboards | 2 days |
| 18 | SDK support | Go SDK methods for ReBAC check/write | 2 days |

### P3 — Future Enhancements

| # | Task | Description |
|---|------|-------------|
| 19 | CockroachDB support | Multi-region active-active tuple store |
| 20 | SPARQL/graph query | Advanced relationship queries for compliance |
| 21 | Policy import/export | Share authorization models across tenants |
| 22 | Visual schema builder | Drag-and-drop schema designer in console |
| 23 | ReBAC + ABAC caveats | Combine relationship checks with attribute conditions |

---

## References

- [Google Zanzibar Paper (2019)](https://research.google/pubs/pub48190/) — "Zanzibar: Google's Consistent, Global Authorization System"
- [OpenFGA Documentation](https://openfga.dev/docs) — Auth0/Okta's open-source Zanzibar implementation
- [SpiceDB Documentation](https://authzed.com/docs) — AuthZed's Zanzibar-faithful implementation
- [Permify Documentation](https://docs.permify.co) — Developer-friendly authorization service
- [Ory Keto](https://www.ory.sh/keto/docs/) — Ory ecosystem authorization service
- [NIST SP 800-162](https://csrc.nist.gov/publications/detail/sp/800-162/final) — Guide to ABAC Definition and Planning
- [Auth0: ReBAC vs ABAC](https://auth0.com/blog/rebac-abac-openfga-cedar/) — OpenFGA vs Cedar comparison
- [Oso Academy: ReBAC](https://www.osohq.com/academy/relationship-based-access-control-rebac) — ReBAC concepts and patterns
