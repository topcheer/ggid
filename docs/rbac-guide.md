# RBAC Guide: Role-Based Access Control

> Complete guide to GGID's hybrid RBAC + ABAC policy engine: role hierarchy, permission model, policy evaluation, attribute-based rules, and common patterns.

---

## Table of Contents

1. [Overview](#overview)
2. [Core Concepts](#core-concepts)
3. [Role Hierarchy](#role-hierarchy)
4. [Permission Model](#permission-model)
5. [Policy Engine Architecture](#policy-engine-architecture)
6. [ABAC Rules](#abac-rules)
7. [REST API Reference](#rest-api-reference)
8. [gRPC API Reference](#grpc-api-reference)
9. [Common Patterns](#common-patterns)
10. [Best Practices](#best-practices)
11. [Debugging Access Decisions](#debugging-access-decisions)

---

## Overview

GGID implements a **hybrid RBAC + ABAC** policy engine:

- **RBAC (Role-Based Access Control)**: Users are assigned roles. Roles contain permissions. Simple, coarse-grained, easy to manage.
- **ABAC (Attribute-Based Access Control)**: Fine-grained rules based on user attributes, resource attributes, and environmental conditions (time, IP, risk score).

**Evaluation order**: RBAC check first → ABAC refinement second.

```
Request → Has required role? → Has required permission? → ABAC conditions met? → ALLOW
              ↓ No                  ↓ No                       ↓ No
             DENY                  DENY                       DENY
```

---

## Core Concepts

### Entity Model

```
┌──────────┐     ┌──────────┐     ┌────────────┐
│  Tenant  │────▶│   Role   │────▶│ Permission │
│          │     │          │     │            │
└──────────┘     └────┬─────┘     └────────────┘
                      │
                      │ assign
                      │
                 ┌────▼─────┐
                 │   User   │
                 │          │
                 └──────────┘
```

| Entity | Description | Scope |
|--------|-------------|-------|
| **Tenant** | Isolated organization | Root-level |
| **Role** | Named collection of permissions | Tenant-scoped |
| **Permission** | Action + resource pair (e.g., `read:users`) | Global or tenant-scoped |
| **User** | Identity assigned one or more roles | Tenant-scoped |

### Key Principles

1. **Tenant isolation**: Every role and permission is scoped to a specific tenant
2. **Explicit grant**: No permissions are granted by default
3. **Unique role key**: Each role has a unique `key` within a tenant (UNIQUE constraint)
4. **Immediate effect**: Policy changes take effect on the next request (no recompilation)

---

## Role Hierarchy

### Standard Roles

| Role Key | Role Name | Description | Key Permissions |
|----------|-----------|-------------|-----------------|
| `super_admin` | Super Admin | Full system access | `*:*` (all actions, all resources) |
| `tenant_admin` | Tenant Admin | Manage everything within a tenant | `*:users`, `*:roles`, `*:orgs`, `*:audit` |
| `security_admin` | Security Admin | Security and audit management | `read:audit`, `write:webhooks`, `read:security` |
| `user_admin` | User Admin | User lifecycle management | `read:users`, `write:users`, `delete:users` |
| `developer` | Developer | API access for integration | `read:users`, `read:roles`, `read:orgs` |
| `auditor` | Auditor | Read-only audit access | `read:audit`, `read:users` (no PII) |
| `end_user` | End User | Basic self-service | `read:self`, `write:self` |

### Custom Roles

Create custom roles via the REST API:

```bash
curl -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer <JWT>" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Content Editor",
    "key": "content_editor",
    "description": "Edit and publish content",
    "permissions": [
      "read:users",
      "write:content",
      "publish:content",
      "delete:own_content"
    ]
  }'
```

### Role Inheritance

Roles do not have built-in inheritance. To simulate hierarchy:
1. Create parent role with broad permissions
2. Create child roles that include a subset of parent permissions
3. Assign users to the most specific role

---

## Permission Model

### Permission Format

Permissions follow the `<action>:<resource>` pattern:

| Action | Description |
|--------|-------------|
| `read` | View/list resource |
| `write` | Create or update resource |
| `delete` | Remove resource |
| `publish` | Make resource public/active |
| `*` | All actions (wildcard) |

| Resource | Description |
|----------|-------------|
| `users` | User accounts |
| `roles` | Role definitions |
| `orgs` | Organizations |
| `audit` | Audit events |
| `content` | Content resources |
| `security` | Security settings |
| `self` | Own user data |
| `*` | All resources (wildcard) |

### Permission Examples

| Permission | Meaning |
|------------|---------|
| `read:users` | Can view user list and details |
| `write:users` | Can create and update users |
| `delete:users` | Can delete users |
| `*:users` | All actions on users |
| `read:*` | Read any resource |
| `*:*` | Full access (super_admin only) |
| `read:self` | Can view own profile |
| `write:self` | Can update own profile |

### Scopes in JWT

Permissions are embedded in JWT as space-delimited scopes:

```json
{
  "sub": "usr_abc123",
  "scope": "read:users write:roles delete:users",
  "roles": ["user_admin"]
}
```

### Scope Check Implementation

The gateway checks scopes using the `HasScope()` function:

```go
// In middleware
func HasScope(requiredScope string, claims JWTClaims) bool {
    for _, scope := range strings.Fields(claims.Scope) {
        if matchScope(scope, requiredScope) {
            return true
        }
    }
    return false
}

// Wildcard matching
func matchScope(owned, required string) bool {
    // "read:*" matches "read:users"
    // "*:*" matches anything
    parts := strings.SplitN(owned, ":", 2)
    if len(parts) == 2 && parts[1] == "*" {
        return strings.HasPrefix(required, parts[0]+":")
    }
    return owned == required
}
```

---

## Policy Engine Architecture

### Evaluation Flow

```
┌──────────────────────────────────────────────────────┐
│                    Incoming Request                    │
│                                                        │
│  1. Extract JWT from Authorization header              │
│  2. Parse tenant_id, user_id, scope, roles from JWT    │
│  3. Determine required permission for endpoint          │
│                                                        │
│  ┌────────────────────────────────────────────────┐   │
│  │              RBAC CHECK                          │   │
│  │                                                  │   │
│  │  Does user's JWT scope contain required scope?  │   │
│  │  • Yes → proceed to ABAC                         │   │
│  │  • No  → 403 Forbidden                          │   │
│  └───────────────────────┬────────────────────────┘   │
│                          │                             │
│  ┌───────────────────────▼────────────────────────┐   │
│  │              ABAC CHECK (if rules exist)          │   │
│  │                                                  │   │
│  │  Evaluate attribute conditions:                   │   │
│  │  • Time of day (business hours only?)             │   │
│  │  • Client IP (allowed IP range?)                  │   │
│  │  • Risk score (step-up auth required?)            │   │
│  │  • Resource ownership (is owner?)                 │   │
│  │  • Yes → ALLOW                                   │   │
│  │  • No  → 403 Forbidden                           │   │
│  └───────────────────────┬────────────────────────┘   │
│                          │                             │
│  ┌───────────────────────▼────────────────────────┐   │
│  │              ADMIN SCOPE CHECK                    │   │
│  │                                                  │   │
│  │  For /api/v1/admin/* endpoints:                   │   │
│  │  Does user have hasAdminScope()?                  │   │
│  │  • Yes → ALLOW                                   │   │
│  │  • No  → 403 Forbidden                           │   │
│  └────────────────────────────────────────────────┘   │
│                                                        │
│                    → Backend Service                    │
└──────────────────────────────────────────────────────┘
```

### Policy Service Endpoints

The policy service runs on:
- **HTTP**: `:8070`
- **gRPC**: `:9070`

---

## ABAC Rules

### Attribute Sources

| Source | Attributes | Example |
|--------|-----------|---------|
| **User** | Department, clearance level, MFA enrolled | `user.department = "Engineering"` |
| **Resource** | Owner, sensitivity, tenant | `resource.owner_id = user.id` |
| **Environment** | Time, IP, geo-location, risk score | `env.time.hour >= 9 AND env.time.hour <= 17` |
| **Request** | Method, path, headers | `request.method = "DELETE" AND user.role = "admin"` |

### ABAC Rule Format

Rules are expressed as conditional expressions:

```json
{
  "rule_name": "business_hours_only",
  "description": "Delete operations allowed only during business hours",
  "condition": "env.time.hour >= 9 AND env.time.hour <= 17",
  "action": "ALLOW",
  "priority": 100
}
```

### Common ABAC Patterns

#### Time-Based Access

```json
{
  "rule_name": "admin_hours",
  "condition": "user.role = 'admin' AND env.time.hour >= 9 AND env.time.hour <= 17",
  "action": "ALLOW"
}
```

#### IP-Based Access

```json
{
  "rule_name": "office_ip_only",
  "condition": "user.role = 'admin' AND env.ip IN ['10.0.0.0/8', '192.168.1.0/24']",
  "action": "ALLOW"
}
```

#### Resource Ownership

```json
{
  "rule_name": "own_data_only",
  "condition": "resource.owner_id = user.id",
  "action": "ALLOW"
}
```

#### Step-Up Authentication

```json
{
  "rule_name": "high_risk_step_up",
  "condition": "env.risk_score > 0.7 AND user.mfa_enrolled = true",
  "action": "REQUIRE_MFA"
}
```

---

## REST API Reference

### Create Role

```bash
POST /api/v1/roles
Content-Type: application/json
X-Tenant-ID: <uuid>

{
  "name": "Content Editor",
  "key": "content_editor",
  "description": "Edit and publish content",
  "permissions": ["read:users", "write:content"]
}
```

Response: `201 Created`
```json
{
  "id": "role_xyz789",
  "name": "Content Editor",
  "key": "content_editor",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "permissions": ["read:users", "write:content"],
  "created_at": "2025-07-11T12:00:00Z"
}
```

**Note**: The `key` field is required and must be unique within the tenant. Empty key causes a 500 error (UNIQUE constraint violation).

### List Roles

```bash
GET /api/v1/roles?tenant_id=<uuid>
```

Response: `200 OK`
```json
{
  "roles": [
    {
      "id": "role_001",
      "name": "Super Admin",
      "key": "super_admin",
      "permissions": ["*:*"]
    },
    {
      "id": "role_002",
      "name": "User Admin",
      "key": "user_admin",
      "permissions": ["read:users", "write:users", "delete:users"]
    }
  ],
  "total": 2
}
```

### Assign Role to User

```bash
POST /api/v1/users/{user_id}/roles
Content-Type: application/json

{
  "role_id": "role_xyz789"
}
```

### Check Permission

```bash
POST /api/v1/policies/check
Content-Type: application/json

{
  "user_id": "usr_abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "action": "write",
  "resource": "users"
}
```

Response: `200 OK`
```json
{
  "allowed": true,
  "reason": "role_permission_match",
  "role": "user_admin"
}
```

### Delete Role

```bash
DELETE /api/v1/roles/{role_id}
```

---

## gRPC API Reference

```protobuf
service PolicyService {
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse);
  rpc CreateRole(CreateRoleRequest) returns (Role);
  rpc ListRoles(ListRolesRequest) returns (ListRolesResponse);
  rpc AssignRole(AssignRoleRequest) returns (google.protobuf.Empty);
  rpc ListABACRules(ListABACRulesRequest) returns (ListABACRulesResponse);
}

message CheckPermissionRequest {
  string user_id = 1;
  string tenant_id = 2;
  string action = 3;
  string resource = 4;
  map<string, string> attributes = 5;
}

message CheckPermissionResponse {
  bool allowed = 1;
  string reason = 2;
  string matched_role = 3;
}
```

---

## Common Patterns

### Pattern 1: Multi-Tier Admin

```
Super Admin (*:*)
  └── Tenant Admin (*:users, *:roles, *:orgs)
       └── Department Admin (read:users, write:users, delete:users)
            └── Team Lead (read:users, write:users)
                 └── End User (read:self, write:self)
```

### Pattern 2: Least Privilege Service Account

```bash
# Create a service account with minimal permissions
curl -X POST .../api/v1/roles \
  -d '{
    "name": "CI/CD Pipeline",
    "key": "cicd_pipeline",
    "permissions": ["read:users", "write:users"]
  }'

# Assign to service account
curl -X POST .../api/v1/users/svc_cicd/roles \
  -d '{"role_id": "role_cicd"}'
```

### Pattern 3: Temporary Elevated Access

1. User normally has `developer` role (`read:users`, `read:roles`)
2. For emergency fix, assign `user_admin` temporarily
3. Remove `user_admin` after fix is complete
4. All actions are audit-logged with the active role

### Pattern 4: Regional Access Control

ABAC rule restricting access by IP region:

```json
{
  "rule_name": "us_only_access",
  "condition": "user.role = 'admin' AND env.geo_country = 'US'",
  "action": "ALLOW",
  "priority": 200
}
```

### Pattern 5: Data Residency Enforcement

```json
{
  "rule_name": "eu_data_only",
  "condition": "resource.region = 'EU' AND user.region = 'EU'",
  "action": "ALLOW"
}
```

---

## Best Practices

### Do

- **Use least privilege**: Grant only the minimum permissions needed
- **Create specific roles**: Prefer specific roles over wildcard permissions
- **Audit role assignments**: Regularly review who has what roles
- **Use ABAC for exceptions**: Keep RBAC simple, use ABAC for edge cases
- **Document custom roles**: Maintain a role catalog with descriptions
- **Separate admin roles**: Don't combine security admin with user admin

### Don't

- **Don't use `*:*` for non-super-admin roles**: Too dangerous
- **Don't assign multiple overlapping roles**: Causes confusion in debugging
- **Don't forget to remove stale roles**: Clean up after project completion
- **Don't rely solely on RBAC**: Use ABAC for fine-grained control
- **Don't hardcode permissions in application code**: Use the policy engine

---

## Debugging Access Decisions

### Permission Denied? Debug Steps

1. **Check JWT scopes**:
   ```bash
   # Decode JWT payload
   echo "<jwt_token>" | cut -d. -f2 | base64 -d | jq .scope
   ```

2. **Verify role assignment**:
   ```bash
   curl .../api/v1/users/{user_id}/roles
   ```

3. **Check policy evaluation**:
   ```bash
   curl -X POST .../api/v1/policies/check \
     -d '{"user_id":"usr_123","action":"write","resource":"users"}'
   ```

4. **Review ABAC rules**:
   ```bash
   curl .../api/v1/policies/rules
   ```

5. **Check audit log for the denial**:
   ```bash
   curl ".../api/v1/audit/events?user_id=usr_123&status_code=403"
   ```

### Common Access Issues

| Symptom | Cause | Fix |
|---------|-------|-----|
| 403 on all endpoints | JWT expired or invalid | Re-authenticate |
| 403 on admin endpoints | Missing admin scope | Assign admin role |
| 403 intermittently | ABAC time/IP rule | Check rule conditions |
| 403 on specific resource | Resource ownership check | Verify resource.owner_id |
| 403 after role change | JWT cached old scopes | Wait for token refresh |

---

*Last updated: 2025-07-11*
