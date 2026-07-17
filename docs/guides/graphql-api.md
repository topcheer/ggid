# GraphQL API — Technical Guide

> Feature: GraphQL Playground + Query Layer
> Location: `services/gateway/internal/middleware/graphql.go`
> Console: `/settings/graphql`

## What It Does

GGID provides a GraphQL API layer that sits alongside the REST API, allowing clients to query exactly the data they need in a single request. The gateway middleware handles query parsing, complexity limiting, field-level authorization, and persisted queries.

## Query Structure

### Basic Query

```graphql
query {
  users(page: 1, size: 10) {
    id
    username
    email
    roles { id name }
  }
}
```

### Mutations

```graphql
mutation {
  createUser(input: {
    username: "newuser"
    email: "newuser@example.com"
  }) {
    id
    username
  }
}
```

### Nested Queries with Fragments

```graphql
fragment UserFields on User {
  id
  username
  email
  mfaEnabled
}

query {
  users {
    ...UserFields
    sessions { id ipAddress lastActiveAt }
  }
}
```

## Complexity Limits

To prevent abusive queries, GGID enforces complexity limits:

| Metric | Limit | Description |
|--------|-------|-------------|
| **Query depth** | 7 | Maximum nesting level |
| **Field count** | 100 | Total fields per query |
| **Complexity score** | 1000 | Weighted score (lists × depth) |

Queries exceeding limits return:
```json
{"errors": [{"message": "Query complexity 1200 exceeds maximum 1000"}]}
```

## Persisted Queries

For production clients, use persisted queries to reduce payload size and improve performance:

1. **Register**: Client sends query hash + full query during build.
2. **Runtime**: Client sends only the hash — server looks up the stored query.
3. **Security**: Only registered queries are accepted in strict mode.

## Field-Level Authorization

Each field checks the caller's permissions:

| Field | Required Permission |
|-------|-------------------|
| `users.email` | `user:read` or self |
| `users.mfaEnabled` | `user:read` |
| `users.sessions` | `admin:read` |
| `auditEvents` | `audit:read` |
| `roles.permissions` | `role:read` |

Unauthorized fields return `null` with a permission error in the `errors` array.

## API Endpoint

```bash
TOKEN="your-jwt-token"

# Execute a GraphQL query
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/graphql" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"query":"{ users { id username email } }"}'

# Mutation example
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/graphql" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"query":"mutation { createUser(input:{username:\"testuser\",email:\"test@test.com\"}){id}}"}'
```

## Console Playground

The GraphQL Playground at `/settings/graphql` provides:
- Interactive query editor with syntax highlighting
- Schema explorer (Docs panel)
- Query history
- Variable input
- Response viewer with JSON formatting

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| 401 Unauthorized | Missing or expired token | Include valid Authorization header |
| Complexity exceeded | Query too deep or too many fields | Split into smaller queries; use fragments |
| Field returns null | Insufficient permissions for that field | Check required permission in schema docs |
| Persisted query not found | Query not registered | Register query before using hash-only mode |

## Best Practices

- **Use fragments**: Reduce duplication and payload size.
- **Persist queries**: Register production queries for performance and security.
- **Request only needed fields**: Avoid `query { users { ...all fields } }` — select explicitly.
- **Batch queries**: Combine multiple data needs into one query to reduce round trips.
- **Handle errors gracefully**: Check the `errors` array even on 200 responses.
