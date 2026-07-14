# GraphQL API Design for IAM

Schema design, query complexity analysis, rate limiting, field-level authorization, pagination, subscriptions, federation, and N+1 prevention.

## Schema Design

```graphql
type Query {
  user(id: ID!): User @auth(requires: "users:read")
  users(first: Int = 50, after: String): UserConnection! @auth(requires: "users:read")
  roles(first: Int = 50, after: String): RoleConnection! @auth(requires: "roles:read")
}

type Mutation {
  createUser(input: CreateUserInput!): User! @auth(requires: "users:write")
  assignRole(userId: ID!, roleId: ID!): User! @auth(requires: "roles:assign")
}

type Subscription {
  auditEvents(filter: String): AuditEvent! @auth(requires: "audit:read")
}

type User {
  id: ID!
  email: String! @auth(requires: "email")
  displayName: String!
  department: String
  roles: [Role!]!
  mfaEnabled: Boolean! @auth(requires: "users:read")
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}
```

## Field-Level Authorization

```go
func (r *userResolver) Email(ctx context.Context, obj *User) (string, error) {
    claims := getClaims(ctx)
    if !claims.HasScope("email") {
        return "", ErrInsufficientScope  // Field hidden
    }
    return obj.Email, nil
}
```

Sensitive fields (email, phone) only resolved if client has required scope.

## Query Complexity Analysis

```go
maxComplexity := 1000

func calculateComplexity(selections []Selection, depth int) int {
    complexity := 0
    for _, sel := range selections {
        complexity += 1  // Each field = 1 point
        if sel.IsList { complexity += 10 }  // Lists are expensive
        complexity += calculateComplexity(sel.SubFields, depth+1) * depth
    }
    return complexity
}

// Reject queries exceeding limit
if complexity > maxComplexity {
    return ErrQueryTooComplex
}
```

## Rate Limiting

| Limit Type | Default | Rationale |
|-----------|---------|-----------|
| Query depth | 7 | Prevent deep nested queries |
| Field count | 100 | Limit response size |
| Node count | 50 | Limit list results |
| Cost per request | 1000 points | Complexity-based |
| Requests per minute | 100/tenant | Fair usage |

## Pagination (Cursor-Based)

```graphql
query {
  users(first: 50, after: "cursor") {
    edges { cursor, node { id, displayName } }
    pageInfo { hasNextPage, endCursor }
    totalCount
  }
}
```

Cursor = base64 of last item's sort key. Consistent performance regardless of offset.

## N+1 Prevention (Dataloaders)

```go
func (r *queryResolver) Users(ctx context.Context) ([]*User, error) {
    users := fetchUsers(ctx)
    // BAD: N+1 queries for roles
    // for _, u := range users { u.Roles = fetchRoles(u.ID) }

    // GOOD: Batch with dataloader
    loader := dataloader.ForContext(ctx)
    for _, u := range users {
        loader.Prime(u.ID, fetchRolesBatch)
    }
    return users, nil
}
```

## Subscriptions (Real-Time Audit)

```graphql
subscription { auditEvents(filter: "action eq 'user.login'") { action, timestamp, actor } }
```

WebSocket transport. Backpressure: if client can't keep up, server drops oldest events.

## Federation

```graphql
# Gateway service extends types from federated services
type User @key(fields: "id") {
  id: ID! @external
  roles: [Role!]! @requires(field: "id")  # Resolved by policy service
}
```

Each service owns its domain. Gateway composes them.

## Monitoring

| Metric | Alert |
|--------|-------|
| Query complexity avg | Track trend |
| N+1 detection | Slow queries → check for missing dataloaders |
| Subscription connections | Track active count |
| Query depth violations | Spike → possible abuse |

## See Also

- [API Versioning Strategy](api-versioning-strategy.md)
- [API Rate Limit Tuning](api-rate-limit-tuning.md)
- [Audit Query API](audit-query-api.md)
- [gRPC vs REST](grpc-vs-rest.md)
