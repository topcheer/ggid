# Role Administration

Role management guide: create/edit/delete roles, permission assignment, role
hierarchy, user-role assignment, default roles, and system roles.

> **See also**: [RBAC/ABAC Guide](rbac-abac-guide.md) for ABAC policy syntax
> and condition operators.

---

## Table of Contents

- [System Roles](#system-roles)
- [Create Custom Role](#create-custom-role)
- [Role Hierarchy](#role-hierarchy)
- [Permission Assignment](#permission-assignment)
- [User-Role Assignment](#user-role-assignment)
- [Default Roles](#default-roles)

---

## System Roles

GGID ships with built-in system roles that cannot be deleted:

| Role | Key | Level | Permissions |
|------|-----|-------|------------|
| Super Admin | `super_admin` | 100 | All resources, all actions, all tenants |
| Admin | `admin` | 90 | User/org/role management, audit, tenant config |
| Security Admin | `security_admin` | 80 | MFA enforcement, impersonation, security monitoring |
| Editor | `editor` | 50 | Create/update resources |
| Viewer | `viewer` | 10 | Read-only access |
| Service Account | `service_account` | 0 | M2M API access (no UI login) |

> System roles can be modified (add permissions) but not deleted.

---

## Create Custom Role

```bash
curl -X POST https://iam.example.com/api/v1/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Content Manager",
    "key": "content_manager",
    "description": "Manage content, not users",
    "permissions": ["content:read", "content:write", "content:publish"],
    "parent_role": "editor"
  }'
```

### Edit Role

```bash
curl -X PATCH .../roles/{role_id} \
  -d '{
    "description": "Updated description",
    "permissions": ["content:read", "content:write", "content:publish", "content:archive"]
  }'
```

### Delete Role

```bash
curl -X DELETE .../roles/{role_id}
```

> Deleting a role automatically revokes it from all assigned users.

---

## Role Hierarchy

```
super_admin (level 100)
├── admin (level 90)
│   ├── security_admin (level 80)
│   └── content_manager (custom, level 60)
│       └── editor (level 50)
│           └── viewer (level 10)
└── service_account (level 0)
```

### Permission Inheritance

- Child roles inherit all permissions from ancestors
- `viewer` inherits nothing (leaf role for reads)
- `editor` inherits `viewer` + adds write
- `admin` inherits `editor` + adds delete + management

---

## Permission Assignment

### Permission Naming Convention

```
{resource}:{action}
```

| Resource | Actions | Example Permissions |
|---------|---------|---------------------|
| users | read, write, delete | `users:read`, `users:delete` |
| roles | read, write, delete | `roles:write` |
| orgs | read, write, delete | `orgs:read` |
| policies | read, write, delete, evaluate | `policies:evaluate` |
| audit | read, export | `audit:read`, `audit:export` |
| content | read, write, publish, archive | `content:publish` |
| tenant | read, write | `tenant:write` (config) |
| `*` | `*` | Super admin (all) |

---

## User-Role Assignment

### Assign Role

```bash
curl -X POST .../admin/users/{user_id}/roles \
  -d '{
    "role_id": "role-uuid",
    "scope": "tenant",
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

### Scope Types

| Scope | Description |
|-------|-------------|
| `tenant` | Applies across entire tenant |
| `org` | Scoped to a specific organization |
| `resource` | Scoped to a specific resource |

### List User Roles

```bash
curl .../admin/users/{user_id}/roles \
  -H "Authorization: Bearer $TOKEN"
```

### Revoke Role

```bash
curl -X DELETE .../admin/users/{user_id}/roles/{role_id}
```

### Bulk Assignment

```bash
curl -X POST .../admin/roles/{role_id}/assign-bulk \
  -d '{ "user_ids": ["user-1", "user-2", "user-3"] }'
```

---

## Default Roles

### New User Default Role

```yaml
tenant:
  default_role: "viewer"
  registration:
    default_role: "viewer"
    auto_assign: true
```

### Per-Provider Default Roles

```yaml
federation:
  default_roles:
    local_registration: "viewer"
    saml_okta: "editor"
    oidc_auth0: "viewer"
    social_google: "viewer"
    ldap_ad: "editor"
```

### Role Expiry Management

```bash
# Find roles expiring within 30 days
curl ".../admin/roles/expiring?days=30" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```json
{
  "expiring": [
    { "user_id": "u1", "username": "temp_admin", "role": "admin", "expires_at": "2024-02-01T00:00:00Z" }
  ]
}
```
