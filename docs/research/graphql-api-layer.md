# GraphQL API Layer for Identity Queries: Schema-Driven, Authorized, Optimized

> **Focus**: A production GraphQL API layer sitting in front of GGID's REST microservices — with a typed schema for identity/policy/oauth/audit domains, role-based field-level authorization, dataloader batch optimization, and federation-ready architecture.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Includes endpoint precondition check (§6), DoD per backlog item (§12), curl verification commands (§7).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Why GraphQL for IAM](#2-why-graphql-for-iam)
3. [GGID Current State: Gateway GraphQL Proxy](#3-ggid-current-state-gateway-graphql-proxy)
4. [Gap Analysis](#4-gap-analysis)
5. [Proposed Architecture: Typed GraphQL Layer](#5-proposed-architecture-typed-graphql-layer)
6. [Endpoint Precondition Check](#6-endpoint-precondition-check)
7. [API Design + Curl Commands](#7-api-design--curl-commands)
8. [Schema Design](#8-schema-design)
9. [Field-Level Authorization](#9-field-level-authorization)
10. [Dataloader Optimization](#10-dataloader-optimization)
11. [Database Schema](#11-database-schema)
12. [Implementation Backlog with DoD](#12-implementation-backlog-with-dod)
13. [Performance Considerations](#13-performance-considerations)
14. [Competitive Differentiation](#14-competitive-differentiation)
15. [Security Considerations](#15-security-considerations)

---

## 1. Executive Summary

GraphQL solves the over-fetching/under-fetching problem of REST APIs. For identity systems, it lets a frontend request exactly the user data it needs — user profile + roles + groups + recent sessions + MFA devices — in a single round-trip, instead of 5 separate REST calls.

GGID has a **basic GraphQL-to-REST proxy** in the gateway (`services/gateway/internal/middleware/graphql.go:34` — `GraphQLResolver`). It:
- Accepts POST `/graphql` with a query string
- Parses top-level field names using a simple brace-depth tracker
- Maps type names to REST backend URLs
- Proxies each field as a separate HTTP request to the backend

However, this is **not a real GraphQL engine**. It lacks:
1. **No type system / schema** — no SDL, no introspection, no type validation
2. **No nested field resolution** — can't resolve `user { roles { permissions } }`
3. **No dataloaders** — N+1 query problem for list-with-relations
4. **No field-level auth** — can't hide `user.ssn` from non-admin callers
5. **No mutations** — query-only, no create/update/delete
6. **No subscriptions** — no real-time updates
7. **No query complexity limits** — vulnerable to deeply nested queries
8. **No persisted queries** — no query allow-listing for security

**Recommendation**: Replace the simple proxy with a **typed GraphQL layer** using [gqlgen](https://gqlgen.com/) (Go code generation from SDL), with role-based field-level authorization, dataloaders for batch optimization, query complexity analysis, and persisted query support.

**Estimated effort**: 4 sprints for MVP (schema + auth + dataloaders + mutations + complexity).

---

## 2. Why GraphQL for IAM

### The N+1 API Problem in Identity

```
Frontend needs: "Show me user Alice, her roles, her groups, and recent logins"

REST approach (5 round-trips):
  1. GET /api/v1/identity/users/alice         → user profile
  2. GET /api/v1/policy/users/alice/roles     → roles
  3. GET /api/v1/identity/users/alice/groups  → groups
  4. GET /api/v1/audit/users/alice/events     → recent events
  5. GET /api/v1/auth/users/alice/mfa-devices → MFA devices

GraphQL approach (1 round-trip):
  query {
    user(id: "alice") {
      email, displayName, status
      roles { name, permissions }
      groups { name, memberCount }
      recentEvents(limit: 10) { type, createdAt }
      mfaDevices { type, name, enrolledAt }
    }
  }
```

### GraphQL vs REST for IAM

| Property | REST | GraphQL |
|----------|------|---------|
| Round-trips for complex view | 5-10 | 1 |
| Over-fetching | Common (full user for just email) | Eliminated (request only needed fields) |
| Under-fetching | Common (need follow-up calls) | Eliminated (nested resolution) |
| Versioning | /v1/, /v2/ URLs | Schema evolution (deprecation) |
| Schema discovery | OpenAPI (separate) | Introspection (built-in) |
| Type safety | Runtime validation | Compile-time (generated types) |
| Client flexibility | Fixed endpoints | Arbitrary queries |
| Caching | HTTP-level (simple) | Client-side (Apollo cache) |
| File uploads | Multipart (simple) | Multipart spec extension |
| Error handling | HTTP status codes | Partial data + errors array |

### When NOT to Use GraphQL

| Scenario | Better with REST |
|----------|-----------------|
| File uploads/downloads | REST multipart is simpler |
| Server-to-server webhooks | REST POST |
| Simple CRUD with no relations | REST |
| HTTP caching critical (CDN) | REST (GraphQL is POST-only) |

**Recommendation**: GraphQL as the **primary read API** for dashboards and Console UI. REST remains for writes, webhooks, and server-to-server. Both coexist.

---

## 3. GGID Current State: Gateway GraphQL Proxy

### Existing Implementation

| Component | File:Line | Status |
|-----------|-----------|--------|
| GraphQLRequest struct | `graphql.go:15` | **Implemented** — query + variables |
| GraphQLResponse struct | `graphql.go:22` | **Implemented** — data + errors |
| GraphQLResolver | `graphql.go:34` | **Implemented** — backend URL map |
| GraphQLHandler | `graphql.go:54` | **Implemented** — POST `/graphql` |
| resolveQuery | `graphql.go:88` | **Implemented** — parse fields, resolve each |
| resolveField | `graphql.go:121` | **Implemented** — proxy to REST backend |
| parseGraphQLFields | `graphql.go:169` | **Implemented** — simple brace-depth parser |
| substituteVariables | `graphql.go:285` | **Implemented** — `$var` substitution |
| Gateway route | `router.go:366` | **Wired** — `/graphql` endpoint |
| Tests | `coverage_boost_test.go:531+` | **Comprehensive** — 15+ test cases |

### What the Proxy CAN Do

```graphql
# This works today:
{ user(id: "123") { } }           # → GET /api/v1/users/123
{ users { } }                      # → GET /api/v1/users
{ org(id: "acme") { } }           # → GET /api/v1/orgs/acme
```

### What the Proxy CANNOT Do

```graphql
# These all FAIL today:
query {
  user(id: "123") {
    email          # Can't return specific fields — returns full REST response
    roles { name } # Can't resolve nested relations
  }
}

mutation {         # No mutation support
  createUser(input: { email: "a@b.com" }) { id }
}

# No type safety, no schema validation, no introspection
```

---

## 4. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No typed schema** | No introspection, no type validation, clients can't discover API |
| 2 | **No nested resolution** | Can't traverse relations (user → roles → permissions) |
| 3 | **No dataloaders** | N+1 problem: fetching 50 users with their roles = 51 backend calls |
| 4 | **No field-level auth** | All fields visible to all callers |
| 5 | **No mutations** | Query-only; can't create/update/delete via GraphQL |
| 6 | **No complexity analysis** | Vulnerable to `query { users { groups { users { groups { ... } } } } }` DoS |
| 7 | **No persisted queries** | Arbitrary queries from clients = security risk |
| 8 | **No subscriptions** | No real-time updates (WebSocket) |
| 9 | **Returns full REST response** | No field selection — over-fetching not solved |
| 10 | **No federation** | Can't compose schemas from multiple services |

---

## 5. Proposed Architecture: Typed GraphQL Layer

```
                    ┌──────────────────────────────────────────────┐
                    │         GraphQL API Layer (gqlgen)            │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  SDL Schema (schema.graphql)          │    │
                    │  │  - User, Role, Group, Session types   │    │
                    │  │  - Policy, OAuthClient, AuditEvent    │    │
                    │  │  - Queries + Mutations                │    │
                    │  │  - Field-level @auth directives       │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────────────────────────┐    │
                    │  │  Resolver Layer (generated by gqlgen) │    │
                    │  │                                      │    │
                    │  │  Query resolvers:                    │    │
                    │  │  ├── user(id) → User                 │    │
                    │  │  ├── users(filter) → [User]          │    │
                    │  │  ├── role(id) → Role                 │    │
                    │  │  ├── group(id) → Group               │    │
                    │  │  ├── auditEvents(filter) → [Event]   │    │
                    │  │  └── me → current User               │    │
                    │  │                                      │    │
                    │  │  Mutation resolvers:                 │    │
                    │  │  ├── createUser(input) → User        │    │
                    │  │  ├── updateUser(id, input) → User    │    │
                    │  │  ├── assignRole(userID, roleID)      │    │
                    │  │  └── revokeSession(sessionID)        │    │
                    │  │                                      │    │
                    │  │  Field resolvers (nested):           │    │
                    │  │  ├── User.roles → [Role]             │    │
                    │  │  ├── User.groups → [Group]           │    │
                    │  │  ├── User.sessions → [Session]       │    │
                    │  │  ├── Role.permissions → [Permission] │    │
                    │  │  └── Group.members → [User]          │    │
                    │  └──────────────────────────────────────┘    │
                    │                                              │
                    │  ┌──────────────────┐  ┌─────────────────┐   │
                    │  │  Dataloaders     │  │  Auth Middleware │   │
                    │  │  (batch resolve) │  │  (field-level)  │   │
                    │  └────────┬─────────┘  └────────┬────────┘   │
                    │           │                     │            │
                    │  ┌────────▼─────────────────────▼────────┐   │
                    │  │  REST Backend Clients (HTTP)           │   │
                    │  │  → Identity Service                    │   │
                    │  │  → Policy Service                      │   │
                    │  │  → Auth Service                        │   │
                    │  │  → OAuth Service                       │   │
                    │  │  → Audit Service                       │   │
                    │  └───────────────────────────────────────┘   │
                    └──────────────────────────────────────────────┘
```

---

## 6. Endpoint Precondition Check

### Existing Infrastructure (Reusable)

| Component | File:Line | Status | Reusable? |
|-----------|-----------|--------|-----------|
| `/graphql` route | `gateway/router/router.go:366` | **Works** | Yes — same endpoint |
| GraphQLResolver | `gateway/middleware/graphql.go:34` | **Works** | Replace with gqlgen |
| Backend URL map | `gateway/middleware/graphql.go:40` | **Works** | Reuse for REST client config |
| JWT auth middleware | `gateway/middleware/jwt_auth.go` | **Works** | Yes — extract claims for field auth |
| Tenant middleware | `gateway/middleware/tenant.go` | **Works** | Yes — tenant context |
| Policy PDP | `services/policy/` | **Works** | Yes — field-level auth checks |
| REST APIs (all services) | Various | **Works** | Yes — backend data source |

### New Components Required

| Component | Purpose | Priority |
|-----------|---------|----------|
| `schema.graphql` | SDL type definitions | P0 |
| Generated resolvers (gqlgen) | Type-safe resolver functions | P0 |
| Dataloader package | Batch field resolution | P0 |
| Field-level auth directive | `@auth(requires: "admin")` | P0 |
| Complexity analyzer | Query depth/cost limits | P0 |
| Persisted query store | Query allow-listing | P1 |
| Subscription handler | WebSocket real-time | P2 |

---

## 7. API Design + Curl Commands

### Single Endpoint

All GraphQL operations go through a single endpoint:

```bash
# Query: Get current user with nested data
curl -X POST https://ggid.corp.com/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query Me { me { id email displayName status roles { name permissions } groups { name } mfaDevices { type name } recentEvents(limit: 5) { type createdAt } } }"
  }'

# Response:
{
  "data": {
    "me": {
      "id": "uuid",
      "email": "alice@corp.com",
      "displayName": "Alice Chen",
      "status": "active",
      "roles": [
        { "name": "developer", "permissions": ["doc:read", "doc:write"] }
      ],
      "groups": [
        { "name": "engineering" }
      ],
      "mfaDevices": [
        { "type": "passkey", "name": "MacBook Pro" }
      ],
      "recentEvents": [
        { "type": "auth.login", "createdAt": "2026-07-17T10:00:00Z" }
      ]
    }
  }
}

# Query: Batch users with roles (dataloader prevents N+1)
curl -X POST https://ggid.corp.com/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query Users { users(limit: 50) { id email roles { name } } }"
  }'

# Mutation: Assign role
curl -X POST https://ggid.corp.com/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation AssignRole($userId: ID!, $roleId: ID!) { assignRole(userId: $userId, roleId: $roleId) { id roles { name } } }",
    "variables": { "userId": "uuid", "roleId": "uuid" }
  }'

# Introspection (disabled in production)
curl -X POST https://ggid.corp.com/graphql \
  -H "Authorization: Bearer $TOKEN" \
  -d '{ "query": "{ __schema { types { name } } }" }'

# Error response format:
{
  "data": null,
  "errors": [
    {
      "message": "Unauthorized: field 'users' requires role 'admin'",
      "path": ["users"],
      "extensions": { "code": "FORBIDDEN" }
    }
  ]
}
```

---

## 8. Schema Design

```graphql
# schema.graphql — GGID Identity GraphQL Schema

directive @auth(requires: Role = ADMIN) on FIELD_DEFINITION
directive @tenantScoped on FIELD_DEFINITION

enum Role { VIEWER DEVELOPER ADMIN SUPER_ADMIN }
enum UserStatus { ACTIVE DISABLED PENDING }
enum MfaDeviceType { PASSKEY TOTP SMS EMAIL }

# === Core Types ===

type User @key(fields: "id") {
  id: ID!
  email: String! @auth(requires: ADMIN)
  username: String!
  displayName: String!
  status: UserStatus!
  avatarUrl: String
  locale: String
  emailVerified: Boolean!
  createdAt: DateTime!
  updatedAt: DateTime
  
  # Nested relations (resolved via dataloaders)
  roles: [Role!]!          # Batch resolved
  groups: [Group!]!        # Batch resolved
  sessions: [Session!]! @auth(requires: ADMIN)
  mfaDevices: [MfaDevice!]! @auth(requires: ADMIN)
  externalIdentities: [ExternalIdentity!]! @auth(requires: ADMIN)
  recentEvents(limit: Int = 10): [AuditEvent!]! @auth(requires: ADMIN)
}

type Role {
  id: ID!
  name: String!
  description: String
  permissions: [String!]!
  usersCount: Int!
  createdAt: DateTime!
}

type Group {
  id: ID!
  name: String!
  description: String
  memberCount: Int!
  members(limit: Int = 50, offset: Int = 0): [User!]!
  parent: Group
  children: [Group!]!
}

type Session {
  id: ID!
  userId: ID!
  ipAddress: String
  userAgent: String
  createdAt: DateTime!
  expiresAt: DateTime!
  isActive: Boolean!
}

type MfaDevice {
  id: ID!
  type: MfaDeviceType!
  name: String!
  enrolledAt: DateTime!
  lastUsedAt: DateTime
}

type AuditEvent {
  id: ID!
  type: String!
  userId: ID
  action: String!
  resourceType: String
  resourceId: String
  ipAddress: String
  success: Boolean
  createdAt: DateTime!
}

type ExternalIdentity {
  id: ID!
  provider: String!
  externalId: String!
  externalEmail: String
  lastSyncedAt: DateTime!
}

# === Queries ===

type Query {
  # Current user (from JWT)
  me: User!
  
  # User queries
  user(id: ID!): User @auth(requires: ADMIN)
  users(filter: UserFilter, limit: Int = 50, offset: Int = 0): UserConnection! @auth(requires: ADMIN)
  
  # Role queries
  role(id: ID!): Role @auth(requires: VIEWER)
  roles(limit: Int = 50): [Role!]! @auth(requires: VIEWER)
  
  # Group queries
  group(id: ID!): Group @auth(requires: VIEWER)
  groups(limit: Int = 50): [Group!]! @auth(requires: VIEWER)
  
  # Audit
  auditEvents(filter: AuditFilter, limit: Int = 50, offset: Int = 0): [AuditEvent!]! @auth(requires: ADMIN)
}

# === Mutations ===

type Mutation {
  createUser(input: CreateUserInput!): User! @auth(requires: ADMIN)
  updateUser(id: ID!, input: UpdateUserInput!): User! @auth(requires: ADMIN)
  disableUser(id: ID!, reason: String): User! @auth(requires: ADMIN)
  
  assignRole(userId: ID!, roleId: ID!): User! @auth(requires: ADMIN)
  revokeRole(userId: ID!, roleId: ID!): User! @auth(requires: ADMIN)
  
  addToGroup(userId: ID!, groupId: ID!): User! @auth(requires: ADMIN)
  removeFromGroup(userId: ID!, groupId: ID!): User! @auth(requires: ADMIN)
  
  revokeSession(sessionId: ID!): Boolean! @auth(requires: ADMIN)
}

# === Inputs ===

input UserFilter {
  search: String
  status: UserStatus
  groupId: ID
  roleId: ID
}

input AuditFilter {
  userId: ID
  type: String
  from: DateTime
  to: DateTime
  success: Boolean
}

input CreateUserInput {
  email: String!
  username: String!
  displayName: String!
  password: String
  roleId: ID
  groupId: ID
}

input UpdateUserInput {
  email: String
  displayName: String
  status: UserStatus
  locale: String
}

# === Pagination ===

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type UserEdge {
  node: User!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

scalar DateTime
```

---

## 9. Field-Level Authorization

### Directive-Based Auth

```go
// gateway/internal/graphql/auth_directive.go

// AuthDirective implements @auth(requires: Role) field directive.
type AuthDirective struct {
    policyClient *policy.Client
}

func (d *AuthDirective) EnsureRole(ctx context.Context, obj interface{}, next graphql.Resolver, requires Role) (interface{}, error) {
    // Extract user from JWT context
    user := GetUserFromContext(ctx)
    if user == nil {
        return nil, gqlerror.Errorf("Unauthorized: authentication required")
    }
    
    // Check if user has required role
    if !user.HasRole(requires) {
        // Also check via PDP for fine-grained access
        allowed, err := d.policyClient.Check(ctx, &policy.CheckRequest{
            Subject:  user.ID,
            Action:   "graphql:field",
            Resource: graphql.GetFieldContext(ctx).FieldName(),
        })
        if err != nil || !allowed {
            return nil, gqlerror.Errorf("Forbidden: field requires role %s", requires)
        }
    }
    
    return next(ctx)
}
```

### PDP Integration

The GraphQL layer integrates with GGID's existing Policy Decision Point:

```
GraphQL query: { users { email roles { name } } }
                      ↑       ↑      ↑
                      │       │      └── @auth(requires: VIEWER)
                      │       └── @auth(requires: ADMIN)
                      └── @auth(requires: ADMIN)

Each field with @auth directive calls:
  PolicyService.Check(subject=userID, action="graphql:field", resource=fieldName)
  
If denied, that field returns null + error, but other fields still resolve
(partial data — GraphQL's key advantage over REST for auth failures)
```

---

## 10. Dataloader Optimization

### The N+1 Problem

```
Query: { users(limit: 50) { id roles { name } } }

Without dataloader (N+1 = 51 requests):
  1. GET /api/v1/identity/users?limit=50        → 50 users
  2. GET /api/v1/policy/users/1/roles            → user 1's roles
  3. GET /api/v1/policy/users/2/roles            → user 2's roles
  ...
  51. GET /api/v1/policy/users/50/roles          → user 50's roles

With dataloader (2 requests):
  1. GET /api/v1/identity/users?limit=50         → 50 users
  2. POST /api/v1/policy/users/batch-roles       → all 50 users' roles in one call
     body: { "user_ids": ["1","2",...,"50"] }
```

### Dataloader Implementation

```go
// gateway/internal/graphql/dataloaders.go

func (r *Resolver) UserRolesLoader(ctx context.Context) *dataloader.Loader {
    return dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
        userIDs := make([]string, len(keys))
        for i, key := range keys {
            userIDs[i] = key.String()
        }
        
        // Single batch call to policy service
        rolesByUser, err := r.policyClient.GetBatchUserRoles(ctx, userIDs)
        if err != nil {
            results := make([]*dataloader.Result, len(keys))
            for i := range results {
                results[i] = &dataloader.Result{Error: err}
            }
            return results
        }
        
        results := make([]*dataloader.Result, len(keys))
        for i, userID := range userIDs {
            results[i] = &dataloader.Result{Data: rolesByUser[userID]}
        }
        return results
    }, dataloader.WithWait(2*time.Millisecond)) // Batch window
}
```

---

## 11. Database Schema

```sql
-- GraphQL persisted queries (for production query allow-listing)
CREATE TABLE graphql_persisted_queries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID,
    query_hash          VARCHAR(64) NOT NULL UNIQUE,    -- SHA-256 of query
    query_text          TEXT NOT NULL,
    operation_name      VARCHAR(128),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- GraphQL query log (for analytics + debugging)
CREATE TABLE graphql_query_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID,
    operation_name      VARCHAR(128),
    query_hash          VARCHAR(64),
    complexity_score    INT,                            -- Computed complexity
    depth               INT,                            -- Query depth
    duration_ms         INT,
    error_count         INT DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_gql_persisted_hash ON graphql_persisted_queries (query_hash);
CREATE INDEX idx_gql_log_tenant_time ON graphql_query_log (tenant_id, created_at DESC);
CREATE INDEX idx_gql_log_user ON graphql_query_log (tenant_id, user_id, created_at DESC);
```

---

## 12. Implementation Backlog with DoD

### P0 — Core GraphQL Layer (3 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | gqlgen setup + SDL schema | ✅ schema.graphql compiles ✅ go generate produces types ✅ go build PASS ✅ No log.Printf/内存 map | 3d |
| 2 | Query resolvers (user/users/me/role/roles) | ✅ Resolves from REST backends ✅ Field selection works ✅ curl test PASS ✅ ≥3 tests | 4d |
| 3 | Nested field resolvers (user.roles, user.groups) | ✅ Relations resolve correctly ✅ curl test PASS ✅ ≥3 tests | 3d |
| 4 | Dataloaders | ✅ Batch resolution prevents N+1 ✅ ≥3 tests verify batching ✅ No per-item HTTP calls | 4d |
| 5 | Field-level @auth directive | ✅ Unauthorized fields return error ✅ Authorized fields resolve ✅ PDP integration ✅ ≥3 tests | 3d |
| 6 | Query complexity analysis | ✅ Deep queries rejected (depth > 10) ✅ Complex queries rejected (cost > 1000) ✅ ≥3 tests | 2d |

### P1 — Mutations + Advanced (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | Mutation resolvers (create/update/disable user, assign/revoke role) | ✅ Mutations write to backends ✅ Audit events emitted ✅ ≥3 tests | 4d |
| 8 | Pagination (Relay cursor-based) | ✅ Cursor pagination works ✅ hasNextPage correct ✅ ≥3 tests | 2d |
| 9 | Persisted queries | ✅ Production mode rejects non-persisted queries ✅ Registration API works ✅ ≥3 tests | 3d |
| 10 | GraphQL query log | ✅ All queries logged with complexity ✅ DB-backed ✅ ≥3 tests | 2d |

### P2 — Console Integration + Subscriptions (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 11 | Console GraphQL playground | ✅ GraphiQL/Playground accessible ✅ Introspection in dev mode ✅ Auth token injection | 2d |
| 12 | Console migration to GraphQL | ✅ Dashboard uses GraphQL for data fetching ✅ Reduces API calls by 60%+ | 5d |
| 13 | Subscriptions (WebSocket) | ✅ Real-time user.status updates ✅ WebSocket transport ✅ ≥3 tests | 4d |

### P3 — Federation + Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 14 | Apollo Federation | Federate schemas from identity/policy/audit services |
| 15 | Automatic persisted queries | APQ protocol support (Apollo client) |
| 16 | GraphQL Defer directive | Stream partial results for slow fields |
| 17 | Schema stitching | Combine GGID GraphQL with external APIs |
| 18 | GraphQL rate limiting | Per-operation rate limits based on complexity |

---

## 13. Performance Considerations

| Operation | Latency | Notes |
|-----------|---------|-------|
| Simple query (1 field, no relations) | 2-5ms | Single REST proxy |
| Complex query (5 fields, 3 relations) | 5-15ms | Parallel resolution |
| Batch query (50 users with roles) | 8-20ms | Dataloader batching |
| Mutation (create user) | 5-10ms | REST POST + audit |
| Query with 10 nested levels | Blocked | Complexity limit (depth ≤ 10) |

### Optimization Strategies

1. **Parallel field resolution**: gqlgen resolves sibling fields concurrently
2. **Dataloader batching**: Batch N+1 into single backend call per relation
3. **Response caching**: Cache stable queries (user profile) at Redis layer
4. **Persisted queries**: Skip parsing/validating known queries
5. **Query complexity cap**: Prevent DoS via deeply nested queries
6. **Field-level defer**: Stream slow fields while fast fields return first

---

## 14. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Keycloak | WorkOS |
|---------|---------------|------|-------|----------|--------|
| **GraphQL API** | **Full (gqlgen)** | No (REST only) | Custom | No | No |
| **Field-level auth** | **@auth directive** | N/A | N/A | N/A | N/A |
| **Nested resolution** | **Dataloaders** | N/A | N/A | N/A | N/A |
| **Mutations** | **Full CRUD** | N/A | N/A | N/A | N/A |
| **Subscriptions** | **WebSocket** | N/A | N/A | N/A | N/A |
| **Query complexity** | **Depth + cost** | N/A | N/A | N/A | N/A |
| **Persisted queries** | **Yes** | N/A | N/A | N/A | N/A |
| **Open source** | **Yes (Apache 2.0)** | No | No | Yes | No |

**Key differentiator**: GGID would be the only open-source IAM with a typed GraphQL API layer — solving the N+1 API problem that plagues REST-based identity systems.

---

## 15. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Query DoS (deep nesting)** | Query depth limit (10) + complexity score limit (1000) |
| **Information disclosure** | Field-level @auth directive hides sensitive fields per role |
| **Introspection in production** | Disabled by default; only enabled with `GRAPHQL_INTROSPECTION=true` |
| **Arbitrary queries** | Persisted query mode (production) only accepts registered queries |
| **Batch abuse** | Rate limit on query complexity, not just request count |
| **Data leakage via relations** | Every nested resolver checks auth before returning data |
| **Tenant isolation** | Tenant ID from JWT injected into every backend call |

---

## References

- [gqlgen: Go GraphQL Server](https://gqlgen.com/) — Code-first GraphQL for Go
- [GraphQL Specification](https://spec.graphql.org/) — Official GraphQL spec
- [GraphQL Authorization Patterns](https://www.osohq.com/post/graphql-authorization) — Field-level auth strategies
- [GraphQL DataLoader](https://github.com/graph-gophers/dataloader) — Batch resolution for Go
- [Cerbos GraphQL Authorization](https://www.cerbos.dev/blog/graphql-authorization) — Authorization in GraphQL
- [Apollo Federation](https://www.apollographql.com/docs/federation/) — Schema composition
- [GraphQL Complexity Analysis](https://github.com/ivpusic/graphql-complexity) — Query cost analysis
- [GGID GraphQLResolver](../services/gateway/internal/middleware/graphql.go) — Existing proxy at line 34
- [GGID GraphQL Tests](../services/gateway/internal/middleware/coverage_boost_test.go) — 15+ existing tests at line 531
- [GGID Gateway Router](../services/gateway/internal/router/router.go) — `/graphql` route at line 366
- [GGID Policy Service](../services/policy/) — PDP for field-level auth checks
