# ABAC Condition Builder Guide

This guide covers building Attribute-Based Access Control (ABAC) conditions in GGID — attribute sources, operators, nesting, time/geo/device conditions, testing, and best practices.

## Overview

ABAC extends RBAC by evaluating dynamic attributes (user properties, resource properties, environmental context) to make access decisions.

```
RBAC: "Is user a developer?" → Yes → Allow
ABAC: "Is user a developer AND in Engineering dept AND accessing during business hours from a trusted device?" → All true → Allow
```

## Attribute Sources

### User Attributes

| Attribute | Source | Example |
|-----------|--------|---------|
| `department` | User profile | `engineering` |
| `level` | User profile | `senior` |
| `clearance` | Custom field | `secret` |
| `manager_id` | Org service | `uuid` |
| `country` | Profile | `US` |

### Resource Attributes

| Attribute | Source | Example |
|-----------|--------|---------|
| `owner_id` | Resource metadata | `uuid` |
| `classification` | Resource tag | `confidential` |
| `department` | Resource tag | `engineering` |
| `created_at` | Resource metadata | `2025-01-01` |

### Environmental Attributes

| Attribute | Source | Example |
|-----------|--------|---------|
| `time` | Server clock | `business_hours` |
| `day_of_week` | Server clock | `weekday` |
| `ip_address` | Request | `192.168.1.50` |
| `ip_range` | CIDR lookup | `10.0.0.0/8` |
| `geo_country` | GeoIP | `US` |
| `device_trusted` | Device registry | `true` |
| `mfa_verified` | Session | `true` |

## Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `department eq "engineering"` |
| `ne` | Not equals | `clearance ne "top_secret"` |
| `in` | In list | `country in ["US", "CA", "UK"]` |
| `not_in` | Not in list | `country not_in ["CN", "RU"]` |
| `gt` | Greater than | `level gt 5` |
| `gte` | Greater or equal | `level gte 3` |
| `lt` | Less than | `risk_score lt 50` |
| `lte` | Less or equal | `risk_score lte 20` |
| `contains` | String contains | `resource contains "document"` |
| `starts_with` | Prefix match | `ip starts_with "10.0"` |
| `cidr` | IP in CIDR | `ip cidr "192.168.0.0/16"` |

## AND/OR Nesting

### Simple AND

```json
{
  "operator": "and",
  "items": [
    {"attribute": "department", "operator": "eq", "value": "engineering"},
    {"attribute": "level", "operator": "gte", "value": 3}
  ]
}
```

### Mixed AND/OR

```json
{
  "operator": "and",
  "items": [
    {
      "operator": "or",
      "items": [
        {"attribute": "department", "operator": "eq", "value": "engineering"},
        {"attribute": "department", "operator": "eq", "value": "research"}
      ]
    },
    {"attribute": "country", "operator": "in", "value": ["US", "CA"]},
    {"attribute": "mfa_verified", "operator": "eq", "value": true}
  ]
}
```

Meaning: `(engineering OR research) AND (US OR CA) AND MFA verified`

## Time-Based Conditions

```json
{
  "attribute": "time",
  "operator": "in",
  "value": "business_hours"
}
```

| Time Value | Definition |
|-----------|------------|
| `business_hours` | Mon-Fri 09:00-18:00 (tenant timezone) |
| `off_hours` | Outside business_hours |
| `weekend` | Saturday-Sunday |
| `weekday` | Monday-Friday |

### Custom Time Windows

```json
{
  "attribute": "time",
  "operator": "between",
  "value": {"start": "09:00", "end": "17:00", "timezone": "America/New_York"}
}
```

## Geo-Based Conditions

```json
{
  "attribute": "geo_country",
  "operator": "in",
  "value": ["US", "CA", "UK", "AU"]
}
```

Or using IP CIDR:
```json
{
  "attribute": "ip_address",
  "operator": "cidr",
  "value": "10.0.0.0/8"
}
```

## Device Conditions

```json
{
  "attribute": "device_trusted",
  "operator": "eq",
  "value": true
}
```

Combine with MFA for high-security:
```json
{
  "operator": "and",
  "items": [
    {"attribute": "device_trusted", "operator": "eq", "value": true},
    {"attribute": "mfa_verified", "operator": "eq", "value": true}
  ]
}
```

## Creating Policies via API

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "Engineering internal docs (business hours)",
    "effect": "allow",
    "resource": "document:internal/*",
    "action": "read",
    "conditions": [{
      "operator": "and",
      "items": [
        {"attribute": "department", "operator": "eq", "value": "engineering"},
        {"attribute": "time", "operator": "in", "value": "business_hours"},
        {"attribute": "ip_address", "operator": "cidr", "value": "10.0.0.0/8"}
      ]
    }]
  }'
```

## Testing with Dry-Run

```bash
curl -X POST https://api.ggid.example.com/api/v1/policies/dry-run \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "rules": [{
      "effect": "allow", "resource": "document:*", "action": "read",
      "conditions": [{"operator": "and", "items": [
        {"attribute": "department", "operator": "eq", "value": "engineering"}
      ]}]
    }],
    "test_cases": [
      {"user_id": "eng-user-uuid", "resource": "document:spec", "action": "read", "context": {"department": "engineering"}},
      {"user_id": "sales-user-uuid", "resource": "document:spec", "action": "read", "context": {"department": "sales"}}
    ]
  }'
```

**Response**:
```json
{
  "results": [
    {"user_id": "eng-user", "allowed": true},
    {"user_id": "sales-user", "allowed": false}
  ]
}
```

## Best Practices

1. **Start with RBAC, add ABAC selectively** — Don't replace RBAC, extend it
2. **Test before deploy** — Always use dry-run first
3. **Keep conditions simple** — If nesting > 3 levels, refactor
4. **Document each policy** — Name, purpose, owner
5. **Review quarterly** — Remove stale conditions
6. **Monitor deny rates** — High deny rate may indicate misconfiguration
7. **Use environmental attributes** — Time, geo, device are powerful and low-maintenance

## See Also

- [Policy API](../api/policy.md)
- [Policy API (detailed)](../api/policy-api.md)
- [Delegation Guide](delegation-guide.md)
- [Access Reviews](access-reviews.md)
