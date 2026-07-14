# RBAC Design Patterns

Role hierarchy, permission inheritance, role mining, dynamic roles, template roles, separation of duties, and delegation.

## Role Hierarchy

```
Super Admin
├── Tenant Admin
│   ├── Department Admin
│   │   ├── Team Lead
│   │   └── Member
│   └── Auditor (read-only)
└── Security Admin
    └── Security Analyst
```

### Inheritance Model

A child role inherits ALL permissions from its parent, plus adds its own:

```go
type Role struct {
    ID          string
    Name        string
    ParentID    *string  // nil = root role
    Permissions []Permission
}

func resolvePermissions(role *Role, roleStore RoleStore) []Permission {
    perms := role.Permissions

    // Walk up the hierarchy
    if role.ParentID != nil {
        parent := roleStore.Get(*role.ParentID)
        perms = append(perms, resolvePermissions(parent, roleStore)...)
    }

    return dedup(perms)
}
```

### Hierarchy Depth

GGID limits hierarchy depth to 5 levels to prevent complexity:

```
Level 0: Super Admin (org-wide)
Level 1: Tenant Admin
Level 2: Department Admin
Level 3: Team Lead
Level 4: Member (leaf)
```

## Permission Inheritance

### Permission Structure

```go
type Permission struct {
    Resource string   // "users", "roles", "documents", "*"
    Actions  []string // "read", "write", "delete", "*"
    Scope    string   // "self", "department", "tenant", "all"
}
```

### Scope-Based Inheritance

| Scope | Can Access |
|-------|-----------|
| `self` | Only own resources |
| `department` | Resources owned by users in same department |
| `tenant` | All resources within tenant |
| `all` | All tenants (super admin only) |

### Example: Department Admin

```yaml
role: department_admin
inherits: team_lead
permissions:
  - resource: users
    actions: [read, write]
    scope: department    # Can manage users in own department
  - resource: roles
    actions: [read]
    scope: tenant        # Can see all roles in tenant
  - resource: roles
    actions: [assign]
    scope: department    # Can assign roles within department
```

## Dynamic Roles

Roles computed at runtime based on user attributes (combines with ABAC):

```yaml
dynamic_roles:
  - name: "project_manager_role"
    condition: |
      user.attributes.project_id != "" &&
      user.attributes.title == "Manager"
    permissions:
      - resource: projects
        actions: [read, write]
        scope: department

  - name: "on_call_role"
    condition: |
      schedule.is_on_call(user.id) &&
      time.hour >= 18 || time.hour < 8
    permissions:
      - resource: systems
        actions: [read, write, restart]
        scope: tenant
```

The user doesn't explicitly hold these roles — they're granted when conditions match.

## Template Roles

Pre-defined role bundles for common patterns:

```bash
# Create roles from template
POST /api/v1/policy/roles/from-template
{
  "template": "developer",
  "name": "Backend Developer",
  "params": {
    "project": "api-gateway",
    "environment": "staging"
  }
}
```

### Built-in Templates

| Template | Included Permissions |
|----------|---------------------|
| `developer` | Code repos (read/write), CI/CD pipelines, staging deploy |
| `reviewer` | Code repos (read), PR approval, merge to protected branches |
| `on_call` | Systems (read/restart), incident tickets, runbooks |
| `auditor` | All resources (read), audit logs (read), no write/delete |
| `contractor` | Limited repos (read), no prod access, time-boxed |
| `service_account` | API access (scoped), no console access |

## Role Mining

Automated discovery of optimal role assignments from access patterns:

```bash
# Run role mining analysis
POST /api/v1/policy/roles/mine
{
  "analysis_period": "90d",
  "method": "role_entropy",
  "min_users_per_role": 5
}
# → {
#   "suggested_roles": [
#     {
#       "name": "Suggested: Data Team Read-Only",
#       "users": 23,
#       "permissions": ["datasets:read", "reports:read", "metrics:read"],
#       "current_overgrant": "users:write"  # Currently have but don't use
#     }
#   ],
#   "cleanup_suggestions": [
#     {"user": "uuid", "remove": "users:delete", "reason": "never used in 90d"}
#   ]
# }
```

### Mining Process

```
1. Collect 90 days of access decisions
2. Cluster users by similar permission patterns
3. Identify common permission sets → candidate roles
4. Detect overgranted permissions (have but never use)
5. Detect undergranted permissions (denied frequently)
6. Present recommendations for admin review
```

## Separation of Duties (SoD)

Prevent fraud by requiring multiple people for critical operations.

### Static SoD

Two roles that cannot be held simultaneously:

```yaml
sod_constraints:
  - name: "no_same_person_create_and_approve"
    description: "Cannot both create and approve purchase orders"
    role_a: "po_creator"
    role_b: "po_approver"
    action: "deny_assignment"
```

```go
func checkSoDViolation(userID string, newRoleID string) error {
    conflictingRoles := sodMap[newRoleID]
    userRoles := getUserRoles(userID)

    for _, r := range userRoles {
        if contains(conflictingRoles, r) {
            return ErrSoDViolation
        }
    }
    return nil
}
```

### Dynamic SoD

Checked at runtime, not assignment time:

```yaml
sod_dynamic:
  - name: "no_self_approval"
    description: "Cannot approve own request"
    check: |
      resource.created_by != user.id
    applies_to: "approvals"
```

### Common SoD Rules

| Constraint | Roles |
|-----------|-------|
| Create + Approve | Initiator ≠ Approver |
| Deploy + Review | Developer ≠ Code Reviewer |
| Payment + Audit | Payment Approver ≠ Auditor |
| Admin + Audit | System Admin ≠ Security Auditor |

## Delegation Patterns

### Role Delegation Chain

```
Tenant Admin → (delegates "Team Lead" role) → Team Lead → (delegates "Member" role) → Member
```

Rules:
- Can only delegate roles you hold (or subset)
- Max delegation depth: 3
- Time-boxed delegations (expire automatically)
- All delegations logged in audit trail

```bash
POST /api/v1/policy/delegations
{
  "granter": "admin-uuid",
  "grantee": "user-uuid",
  "role_id": "team_lead",
  "expires_at": "2025-01-16T00:00:00Z",
  "constraints": {
    "department": "engineering"
  }
}
```

### Delegation vs Proxy

| Pattern | Delegation | Proxy |
|---------|-----------|-------|
| Who acts? | Grantee (with own identity) | Granter (on behalf of grantee) |
| Audit | Shows grantee as actor | Shows granter as actor, grantee in `act` claim |
| Token | Grantee's own token | Proxy token with `act` claim |
| Use case | Ongoing responsibility | Temporary assistance |

## Monitoring

| Metric | Alert |
|--------|-------|
| Role explosion | >100 roles per tenant → review |
| Overgranted roles | Role has permission used by <5% of holders |
| SoD violations | Blocked assignment attempts |
| Stale roles | No users assigned >90 days |
| Delegation depth | >3 → blocked |

## See Also

- [Policy Engine Internals](policy-engine-internals.md)
- [Delegated Administration](delegated-administration.md)
- [Conditional Access](conditional-access.md)
- [OAuth Client Scoped Permissions](oauth-client-scoped-permissions.md)
- [Access Reviews](access-reviews.md)
