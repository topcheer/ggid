# Organization Management

Organization management: create org tree, member management, cross-org roles,
org-level policies, and org admin delegation.

---

## Table of Contents

- [Organization Tree](#organization-tree)
- [Create Organization](#create-organization)
- [Member Management](#member-management)
- [Cross-Org Roles](#cross-org-roles)
- [Org-Level Policies](#org-level-policies)
- [Org Admin Delegation](#org-admin-delegation)

---

## Organization Tree

```
Company (root org)
├── Engineering
│   ├── Platform Team
│   ├── Frontend Team
│   └── DevOps
├── Sales
│   ├── Inside Sales
│   └── Field Sales
├── Marketing
└── Finance
```

Organizations form a tree. Users belong to one or more orgs. Roles can be
scoped to an org, limiting the user's permissions to that org's resources.

---

## Create Organization

```bash
curl -X POST https://iam.example.com/api/v1/orgs \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Engineering",
    "description": "Engineering Department",
    "parent_id": "root-org-uuid",
    "metadata": { "cost_center": "CC-1001" }
  }'
```

### View Org Tree

```bash
curl .../admin/orgs/tree \
  -H "Authorization: Bearer $TOKEN"
```

```json
{
  "org": {
    "id": "root",
    "name": "Company",
    "children": [
      { "id": "eng", "name": "Engineering", "children": [...] },
      { "id": "sales", "name": "Sales" }
    ]
  }
}
```

---

## Member Management

### Add User to Org

```bash
curl -X POST .../admin/users/{user_id}/orgs \
  -d '{ "org_id": "engineering-org-uuid" }'
```

### List Org Members

```bash
curl .../admin/orgs/{org_id}/members \
  -H "Authorization: Bearer $TOKEN"
```

### Remove from Org

```bash
curl -X DELETE .../admin/orgs/{org_id}/members/{user_id}
```

### Move User Between Orgs

```bash
curl -X POST .../admin/users/{user_id}/orgs \
  -d '{ "org_id": "new-org-id", "remove_from_previous": true }'
```

---

## Cross-Org Roles

Users can belong to multiple orgs with different roles in each:

```
User: Jane Doe
  ├── Engineering org: editor
  ├── Marketing org: viewer
  └── QA org: admin
```

### Assign Org-Scoped Role

```bash
curl -X POST .../admin/users/{user_id}/roles \
  -d '{
    "role_id": "editor-role-id",
    "scope": "org",
    "scope_id": "engineering-org-uuid"
  }'
```

### List All Org Roles for User

```bash
curl .../admin/users/{user_id}/roles?include_orgs=true \
  -H "Authorization: Bearer $TOKEN"
```

```json
{
  "roles": [
    { "role": "editor", "scope": "org", "org": "Engineering", "org_id": "eng-uuid" },
    { "role": "viewer", "scope": "org", "org": "Marketing", "org_id": "mktg-uuid" }
  ]
}
```

---

## Org-Level Policies

ABAC policies can be scoped to an organization:

```bash
curl -X POST .../admin/policies \
  -d '{
    "name": "Eng Only Access",
    "effect": "allow",
    "actions": ["documents:read"],
    "resources": ["documents/*"],
    "conditions": {
      "subject.org_id": { "equals": "engineering-org-uuid" }
    },
    "scope": "org",
    "scope_id": "engineering-org-uuid"
  }'
```

---

## Org Admin Delegation

Tenant admins can delegate org-level administration to org admins:

### Appoint Org Admin

```bash
curl -X POST .../admin/orgs/{org_id}/admins \
  -d '{
    "user_id": "user-uuid",
    "permissions": ["users:read", "users:write", "roles:assign"]
  }'
```

### Org Admin Capabilities

Org admins can (within their org only):
- Add/remove members from their org
- Assign roles scoped to their org
- Create org-level policies
- View audit logs for their org

### Limits

Org admins CANNOT:
- Delete the organization
- Create child organizations
- Assign roles outside their org
- Modify tenant-wide settings
