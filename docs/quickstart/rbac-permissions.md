# RBAC Permissions Quickstart

> Create roles, assign permissions, and check user access in 5 minutes.

---

## Prerequisites

- Complete [5-Minute JWT](./5-minute-jwt.md) first
- You need a JWT with admin scope

---

## Create a Role

```bash
JWT="your-access-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Create "Editor" role
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"name":"Editor","key":"editor","permissions":["read:users","write:users"]}' | jq .
```

## Assign Role to User

```bash
ROLE_ID=$(curl -s http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq -r '.roles[] | select(.key=="editor") | .id')

USER_ID="usr_abc123"  # from registration

curl -s -X POST "http://localhost:8080/api/v1/users/$USER_ID/roles" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d "{\"role_id\":\"$ROLE_ID\"}" | jq .
```

## Check Permission

```bash
curl -s -X POST http://localhost:8080/api/v1/policies/check \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"'$USER_ID'","action":"write","resource":"users"}' | jq .
# → {"allowed":true,"reason":"role_permission_match"}
```

## Permission Format

```
<action>:<resource>

Actions: read, write, delete, publish, * (wildcard)
Resources: users, roles, orgs, audit, security, self, * (wildcard)
```

---

*See: [RBAC Guide](../rbac-guide.md) | Policy API*