# Policy Service API Reference

Complete REST API for GGID's Policy service — RBAC, ABAC, policy CRUD, permission check, SoD, delegation.

**Base URL**: `https://api.ggid.example.com/api/v1`

## Roles

### CRUD
```
GET /api/v1/roles?page=1&page_size=20
POST /api/v1/roles   {"key":"developer","name":"Developer","permissions":["users:read"]}
GET /api/v1/roles/{id}
PUT /api/v1/roles/{id}
DELETE /api/v1/roles/{id}
```

### Assignment
```
POST /api/v1/users/{user_id}/roles   {"role_id":"uuid","expires_at":"2025-12-31T23:59:59Z"}
DELETE /api/v1/users/{user_id}/roles/{role_id}
GET /api/v1/users/{user_id}/roles
```

## Permissions

```
GET /api/v1/permissions
```
```json
{"permissions":[
  {"resource":"users","action":"read"},
  {"resource":"users","action":"write"},
  {"resource":"audit","action":"read"},
  {"resource":"policy","action":"check"}
]}
```

## Policy Evaluation

### Check Permission
```
POST /api/v1/policies/check
```
```json
{"user_id":"uuid","resource":"document:report","action":"read","context":{"ip":"192.168.1.50"}}
```
**Response**: `{"allowed":true,"reason":"role:developer permits document:read","policy_id":"uuid"}`

### Dry-Run
```
POST /api/v1/policies/dry-run
```
```json
{"rules":[{"effect":"allow","resource":"document:*","action":"read"}],"test_cases":[{"user_id":"uuid","resource":"document:spec","action":"read"}]}
```
**Response**: `{"results":[{"allowed":true}]}`

## Policy CRUD

```
GET /api/v1/policies
POST /api/v1/policies   {"name":"Eng docs","effect":"allow","resource":"document:*","action":"read"}
DELETE /api/v1/policies/{id}
```

## Segregation of Duties (SoD)

```
GET /api/v1/policies/sod
POST /api/v1/policies/sod   {"name":"No self-grant","conflicting_roles":["admin","auditor"],"action":"deny"}
POST /api/v1/policies/sod/check   {"user_id":"uuid","proposed_role_id":"uuid"}
```
**Response**: `{"violated":false,"conflicts":[]}`

## Delegation

```
POST /api/v1/policies/delegation
```
```json
{"from_user_id":"manager-uuid","to_user_id":"delegate-uuid","role_id":"approver-uuid","expires_at":"2025-06-30T23:59:59Z"}
```

```
GET /api/v1/policies/delegation?user_id={user_id}
DELETE /api/v1/policies/delegation/{id}
```

## Coverage Matrix

```
GET /api/v1/policies/coverage
```
Returns which roles cover which resources/actions.

## Access Graph

```
GET /api/v1/policies/access-graph?user_id={user_id}
```
Returns effective permissions via role inheritance.

## Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 422 | `sod_violation` | SoD conflict |
| 403 | `forbidden` | Insufficient scope |

## See Also
- [Policy API (detailed)](policy-api.md)
- [Delegation Guide](../guides/delegation-guide.md)
- [Access Reviews](../guides/access-reviews.md)
