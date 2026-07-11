# Attribute-Based Access Control (ABAC) Guide

> Define policies based on user attributes, resource properties, and environmental conditions. Includes policy syntax, dry-run testing, and compliance templates.

---

## Overview

While RBAC checks "does the user have role X?", ABAC asks "do the user's attributes, resource properties, and environmental conditions satisfy policy Y?" This enables fine-grained, context-aware access control.

```
ABAC = Subject Attributes + Resource Attributes + Environment + Policy Rules → Allow/Deny
```

**When to use ABAC instead of (or alongside) RBAC:**

| Scenario | RBAC | ABAC |
|----------|------|------|
| "Editors can write users" | Yes | — |
| "Only doctors can access patient records during business hours" | No | Yes |
| "Deny access if user hasn't completed MFA" | No | Yes |
| "Allow data deletion if user is the data owner" | No | Yes |
| "Block production access outside business hours" | No | Yes |

---

## Prerequisites

- GGID Gateway running at `http://localhost:8080`
- Admin JWT and tenant ID
- Complete [RBAC Guide](./role-based-access.md) first (roles are still used as attributes)

---

## 1. Policy Structure

Every ABAC policy has this structure:

```json
{
  "name": "Deny access outside business hours",
  "effect": "deny",
  "actions": ["read", "write"],
  "resources": ["card_data"],
  "priority": 100,
  "description": "PCI-DSS compliance: block card data access after hours"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Human-readable policy name |
| `description` | string | Optional explanation |
| `effect` | `allow` \| `deny` | Decision when policy matches |
| `actions` | string[] | Actions this policy applies to (`*` = all) |
| `resources` | string[] | Resources this policy applies to (`*` = all) |
| `priority` | int | Higher = evaluated first (default 0) |

### Attribute Sources

ABAC evaluates attributes from three sources:

| Prefix | Source | Example |
|--------|--------|---------|
| `user.*` | User profile / JWT claims | `user.role`, `user.mfa_verified` |
| `resource.*` | Target resource properties | `resource.owner_id`, `resource.classification` |
| `env.*` / `request.*` | Environmental / request context | `env.time`, `request.ip`, `request.approved` |

---

## 2. Create an ABAC Policy

```bash
JWT="your-admin-jwt"
TENANT="00000000-0000-0000-0000-000000000001"

curl -s -X POST http://localhost:8080/api/v1/policies \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "'"$TENANT"'",
    "name": "Require MFA for sensitive data",
    "description": "Deny read access to patient_records if user has not verified MFA",
    "effect": "deny",
    "actions": ["read"],
    "resources": ["patient_records"],
    "priority": 200
  }' | jq .
```

**Response (201 Created):**

```json
{
  "id": "770e8400-e29b-41d4-a716-446655440002",
  "name": "Require MFA for sensitive data",
  "effect": "deny",
  "actions": ["read"],
  "resources": ["patient_records"],
  "priority": 200
}
```

### List Policies

```bash
curl -s "http://localhost:8080/api/v1/policies?tenant_id=$TENANT" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

### Delete a Policy

```bash
POLICY_ID="770e8400-e29b-41d4-a716-446655440002"

curl -s -X DELETE "http://localhost:8080/api/v1/policies/$POLICY_ID" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

---

## 3. Evaluate with Attributes

The evaluate endpoint checks a request against all policies with attribute context:

```bash
USER_ID="usr_abc123def456"

curl -s -X POST http://localhost:8080/api/v1/policies/evaluate \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "'"$USER_ID"'",
    "tenant_id": "'"$TENANT"'",
    "resource_type": "patient_records",
    "action": "read",
    "resource": "patient_records",
    "attributes": {
      "user.mfa_verified": false,
      "user.role": "nurse",
      "env.time": "02:00",
      "request.external": false
    }
  }' | jq .
```

**Response (Denied):**

```json
{
  "allowed": false,
  "reason": "policy_denied:require_mfa_for_sensitive_data",
  "matched_by": "policy:770e8400...",
  "evaluation_time_ms": 0.3
}
```

**Response (Allowed — when MFA is verified):**

```json
{
  "allowed": true,
  "reason": "no_deny_matched",
  "matched_by": "default_allow",
  "evaluation_time_ms": 0.2
}
```

---

## 4. Dry-Run Testing

The dry-run endpoint evaluates a hypothetical request without affecting production. Perfect for testing policy changes before deployment.

```bash
curl -s -X POST http://localhost:8080/api/v1/policies/dry-run \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "'"$USER_ID"'",
    "tenant_id": "'"$TENANT"'",
    "resource": "card_data",
    "action": "read",
    "attributes": {
      "user.role": "analyst",
      "user.mfa_verified": true,
      "env.time": "14:30",
      "request.external": false
    }
  }' | jq .
```

**Response:**

```json
{
  "allowed": true,
  "reason": "no_deny_matched",
  "matched_by": "default_allow",
  "conditions_evaluated": 4
}
```

Use dry-run to:
- Test what happens if you add a new deny policy
- Verify a user will still have access after a policy change
- Simulate attribute values (e.g., `user.mfa_verified: false`)

---

## 5. Compliance Templates

GGID ships with pre-built policy templates for common regulatory frameworks. These can be applied with a single API call.

### List Available Templates

```bash
curl -s http://localhost:8080/api/v1/policies/templates \
  -H "Authorization: Bearer $JWT" | jq .
```

**Response:**

```json
{
  "templates": [
    {
      "id": "pci-dss",
      "name": "PCI-DSS Access Control",
      "compliance": "PCI-DSS v4.0",
      "policy_count": 2
    },
    {
      "id": "hipaa",
      "name": "HIPAA Healthcare Privacy",
      "compliance": "HIPAA 2023",
      "policy_count": 2
    },
    {
      "id": "soc2",
      "name": "SOC 2 Security",
      "compliance": "SOC 2 Type II",
      "policy_count": 2
    },
    {
      "id": "gdpr",
      "name": "GDPR Data Protection",
      "compliance": "GDPR 2024",
      "policy_count": 2
    }
  ],
  "count": 4
}
```

### Apply a Template

```bash
# Apply PCI-DSS baseline policies
curl -s -X POST http://localhost:8080/api/v1/policies/from-template/pci-dss \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d "{\"tenant_id\":\"$TENANT\"}" | jq .
```

**Response (201 Created):**

```json
{
  "template": "pci-dss",
  "created": [
    { "id": "...", "name": "[PCI-DSS v4.0] Deny card data access outside business hours" },
    { "id": "...", "name": "[PCI-DSS v4.0] Require MFA for card data access" }
  ]
}
```

### Template Policies

| Template | Policies Created |
|----------|-----------------|
| **PCI-DSS** | Deny card data outside business hours; Require MFA for card data |
| **HIPAA** | Deny PHI without medical role; Deny PHI export to external |
| **SOC 2** | Require strong auth for production; Deny production write without approval |
| **GDPR** | Deny personal data without consent; Allow erasure for data owners |

---

## 6. Policy Export / Import

### Export All Policies

```bash
curl -s "http://localhost:8080/api/v1/policies/export?tenant_id=$TENANT" \
  -H "Authorization: Bearer $JWT" > policies_backup.json

jq '.policies | length' policies_backup.json
# → 12
```

### Import Policies

```bash
curl -s -X POST http://localhost:8080/api/v1/policies/import \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d @policies_backup.json | jq .
```

---

## 7. Policy Evaluation Order

Policies are evaluated in priority order (highest first). The decision algorithm:

```
1. Evaluate all DENY policies (highest priority first)
   → If any deny matches: DENY immediately
2. Evaluate all ALLOW policies
   → If any allow matches: ALLOW
3. Default action (configurable: deny or allow)
```

### Check Default Action

```bash
curl -s "http://localhost:8080/api/v1/policies/default-action?tenant_id=$TENANT" \
  -H "Authorization: Bearer $JWT" | jq .
# → {"default_action": "deny"}
```

---

## 8. Common ABAC Patterns

### Time-Based Access Control

```json
{
  "name": "Deny outside business hours",
  "effect": "deny",
  "actions": ["read", "write"],
  "resources": ["financial_data"],
  "priority": 300
}
```

Evaluate with `env.time` attribute to enforce.

### Owner-Based Access

```json
{
  "name": "Allow owner to delete own data",
  "effect": "allow",
  "actions": ["delete"],
  "resources": ["documents"],
  "priority": 150
}
```

Evaluate with `user.is_owner: true` attribute.

### IP-Based Restrictions

```json
{
  "name": "Deny access from external networks",
  "effect": "deny",
  "actions": ["*"],
  "resources": ["internal:*"],
  "priority": 400
}
```

Evaluate with `request.external: true` attribute.

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/policies` | Create policy |
| `GET` | `/api/v1/policies?tenant_id=X` | List policies |
| `DELETE` | `/api/v1/policies/{id}` | Delete policy |
| `POST` | `/api/v1/policies/check` | Check permission (RBAC + conditions) |
| `POST` | `/api/v1/policies/evaluate` | Evaluate with full attributes |
| `POST` | `/api/v1/policies/dry-run` | Test without affecting production |
| `GET` | `/api/v1/policies/templates` | List compliance templates |
| `POST` | `/api/v1/policies/from-template/{id}` | Apply template |
| `GET` | `/api/v1/policies/export?tenant_id=X` | Export policies as JSON |
| `POST` | `/api/v1/policies/import` | Import policies |
| `GET` | `/api/v1/policies/default-action?tenant_id=X` | Get default action |
| `GET` | `/api/v1/policies/versions?policy_id=X` | Policy version history |

---

*See also: [RBAC Guide](./role-based-access.md) | [Security Hardening](./security-hardening.md) | [API Reference](../api-reference.md)*

*Last updated: 2025-07-11*
