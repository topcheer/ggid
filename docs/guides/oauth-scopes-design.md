# OAuth Scopes & Permissions Design

> Scope naming conventions, wildcard matching, and the scope vs role vs permission model.

---

## Scope Naming Convention

GGID uses `resource:action` format:

```
read:users       — Read user profiles
write:users      — Create/update users
delete:users     — Delete users
read:roles       — View roles
write:roles      — Create/modify roles
*:*              — Admin (all resources, all actions)
```

### Wildcard Matching

| Scope | Matches |
|-------|----------|
| `read:*` | All read actions |
| `*:users` | All actions on users |
| `*:*` | Everything |
| `read:users` | Only user reads |

---

## Scope vs Role vs Permission

```
Role (e.g. "Editor")
  └── has Permissions (read:users, write:users)
       └── checked via Scopes in JWT

User → Assigned Role → Inherits Permissions → JWT contains scope claim
```

| Concept | Where Defined | Where Used |
|---------|--------------|------------|
| **Permission** | `role_permissions` table | Assigned to roles |
| **Role** | `roles` table | Assigned to users |
| **Scope** | JWT `scope` claim | Checked by Gateway |

---

## API Design Best Practices

1. **One scope per action**: `read:users` for GET, `write:users` for POST/PUT
2. **Granular resources**: `read:audit` separate from `read:users`
3. **No overlap**: Don't have both `manage:users` and `write:users`
4. **Document scopes**: Each API endpoint declares required scope

### Example: Protected Endpoint

```go
// Gateway checks scope before forwarding
mux.HandleFunc("/api/v1/users", client.RequirePermission("users", "read", handler))
// Requires scope: read:users
```

### Scope Hierarchy

```
admin role → *:* (everything)
manager role → read:*, write:users
editor role → read:users, write:users
viewer role → read:users
```

---

## Custom Scopes

```bash
curl -X POST .../api/v1/roles/{id}/permissions \
  -d '{"permissions":["deploy:production","rollback:production"]}'
```

---

*See: [RBAC Guide](role-based-access.md) | [ABAC Policy](abac-policy.md) | [REST API Reference](../api/rest-api.md)*

*Last updated: 2025-07-11*
