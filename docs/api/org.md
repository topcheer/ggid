# Org Service API Reference

Complete REST API for GGID's Organization service — tenants, org tree, departments, teams, memberships.

**Base URL**: `https://api.ggid.example.com/api/v1`

## Tenants

### List/Create/Delete (Super Admin)
```
GET /api/v1/tenants
POST /api/v1/tenants   {"name":"Acme Corp","plan":"enterprise"}
DELETE /api/v1/tenants/{id}
```

### Tenant Export
```
GET /api/v1/tenants/{id}/export
```

## Organizations

### CRUD
```
GET /api/v1/organizations
POST /api/v1/organizations   {"name":"Engineering","description":"Engineering team","parent_id":"parent-org-uuid"}
GET /api/v1/organizations/{id}
DELETE /api/v1/organizations/{id}
```

**Response** (201):
```json
{"id":"org-uuid","name":"Engineering","parent_id":"parent-uuid","path":"root.engineering","created_at":"2025-01-01T00:00:00Z"}
```

### Org Tree

```
GET /api/v1/organizations/tree
```
Returns nested org hierarchy using PostgreSQL LTREE.

**Response**:
```json
{
  "id":"root-uuid","name":"Acme Corp","children":[
    {"id":"eng-uuid","name":"Engineering","children":[
      {"id":"backend-uuid","name":"Backend","children":[]}
    ]},
    {"id":"sales-uuid","name":"Sales","children":[]}
  ]}
}
```

## Departments

```
GET /api/v1/organizations/{id}/departments
POST /api/v1/organizations/{id}/departments   {"name":"Backend","description":"Backend team"}
```

## Teams

```
GET /api/v1/organizations/{id}/teams
POST /api/v1/organizations/{id}/teams   {"name":"Platform","description":"Platform team"}
GET /api/v1/organizations/{id}/teams/{team_id}
DELETE /api/v1/organizations/{id}/teams/{team_id}
```

## Memberships

### Add Member
```
POST /api/v1/organizations/{id}/members
{"user_id":"uuid","role":"developer"}
```

### List Members
```
GET /api/v1/organizations/{id}/members
```
**Response**:
```json
{
  "members":[
    {"user_id":"uuid","username":"alice","role":"developer","joined_at":"2025-01-01T00:00:00Z"}
  ]
}
```

### Remove Member
```
DELETE /api/v1/organizations/{id}/members/{user_id}
```

### Transfer Member
```
POST /api/v1/organizations/{id}/members/{user_id}/transfer
{"to_org_id":"target-org-uuid"}
```

## Reporting Structure

```
GET /api/v1/organizations/{id}/reporting-structure
```
Returns manager → reportee hierarchy.

## Cost Centers

```
GET /api/v1/organizations/{id}/cost-centers
POST /api/v1/organizations/{id}/cost-centers   {"code":"ENG-001","name":"Engineering"}
```

## Statistics

```
GET /api/v1/organizations/{id}/stats
```
**Response**: `{"member_count":42,"department_count":5,"team_count":8,"active":true}`

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `invalid_argument` | Circular parent reference |
| 404 | `not_found` | Org not found |
| 409 | `already_exists` | Duplicate org name |

## See Also
- [REST API Reference](rest-api.md)
- [Multi-Tenant Architecture](../guides/multi-tenant-architecture.md)
- [Identity API](identity.md)
