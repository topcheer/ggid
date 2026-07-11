# OAuth Scopes Design Patterns

> Best practices for resource:action scope convention, wildcard matching, and scope vs role hierarchy.

---

## Scope Naming Convention

Use `resource:action` format consistently:

```
read:users          # Read user profiles
write:users         # Create/update users
delete:users        # Delete users
read:roles          # View roles
write:roles         # Create/modify roles
read:audit          # Query audit events
write:policies      # Create ABAC policies
read:orgs           # List organizations
*:*                 # Admin (all resources, all actions)
```

---

## Wildcard Matching Rules

| Scope Pattern | Matches | Does NOT Match |
|---------------|---------|----------------|
| `read:*` | `read:users`, `read:roles` | `write:users` |
| `*:users` | `read:users`, `write:users`, `delete:users` | `read:roles` |
| `*:*` | Everything | — |
| `read:users` | Only `read:users` | `write:users` |

### Implementation

```go
func hasScope(granted, required string) bool {
    if granted == "*:*" { return true }
    gParts := strings.Split(granted, ":")
    rParts := strings.Split(required, ":")
    match := func(g, r string) bool { return g == "*" || g == r }
    return match(gParts[0], rParts[0]) && match(gParts[1], rParts[1])
}
```

---

## Scope vs Role vs Permission

```
User ──assigned──▶ Role ──has──▶ Permissions ──checked as──▶ JWT scope

Example:
  User alice → Role "Editor" → Permissions [read:users, write:users]
       → JWT scope claim: "read:users write:users"
```

| Layer | Where | Example |
|-------|------|---------|
| Permission | role_permissions table | `read:users` |
| Role | roles table | `editor` |
| Scope (JWT) | JWT `scope` claim | `read:users write:users` |
| Policy | ABAC engine | Context-aware deny/allow |

---

## Design Best Practices

1. **One resource per scope group**: `users`, `roles`, `orgs`, `audit`
2. **Standard actions**: `read`, `write`, `delete` per resource
3. **No overlapping scopes**: Don't have both `manage:users` and `write:users`
4. **Least privilege**: Start with `read:*`, escalate to `write:*` only when needed
5. **Document required scope per endpoint**: Each API must declare its scope
6. **Custom scopes**: `deploy:production`, `rollback:production` for app-specific needs

---

## Anti-Patterns

| Anti-Pattern | Why Bad | Fix |
|-------------|---------|-----|
| `admin` (no resource:action) | Opaque, can't audit | Use `*:*` or specific scopes |
| `manage:users` (overlapping) | Overlaps read+write+delete | Use specific `read:users write:users` |
| Scope per user | Not scalable | Scope per role |
| No wildcard support | Verbose JWT | Support `read:*` pattern |

---

## GGID Implementation

```bash
# Create role with permissions
curl -X POST .../api/v1/roles \
  -d '{"name":"Editor","key":"editor"}'

curl -X POST .../api/v1/roles/$ROLE_ID/permissions \
  -d '{"permissions":["read:users","write:users"]}'
```

Gateway checks scope before forwarding:
```go
mux.Handle("/api/v1/users", RequirePermission("users", "read", handler))
```

---

*See: [RBAC Guide](role-based-access.md) | [OAuth Scopes Guide](../guides/oauth-scopes-design.md) | [REST API Reference](../api/rest-api.md)*

*Last updated: 2025-07-11*
