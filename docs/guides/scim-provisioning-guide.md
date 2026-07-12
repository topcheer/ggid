# SCIM Provisioning Guide

SCIM 2.0 spec overview, resource models, CRUD endpoints, bulk operations, filtering, pagination, PATCH, enterprise schemas, and IdP integration.

## SCIM 2.0 Overview

SCIM (RFC 7643/7644) automates user provisioning across applications. GGID implements SCIM Provider (server) and Consumer (client).

## Resource Models

| Resource | Endpoint | Purpose |
|----------|----------|---------|
| User | /scim/v2/Users | User CRUD |
| Group | /scim/v2/Groups | Group CRUD |
| ServiceProviderConfig | /scim/v2/ServiceProviderConfig | Capabilities |
| Bulk | /scim/v2/Bulk | Batch operations |

## CRUD Endpoints

| Method | Path | Action |
|--------|------|--------|
| GET | /scim/v2/Users | List/search |
| POST | /scim/v2/Users | Create |
| GET | /scim/v2/Users/{id} | Get single |
| PUT | /scim/v2/Users/{id} | Full replace |
| PATCH | /scim/v2/Users/{id} | Partial update |
| DELETE | /scim/v2/Users/{id} | Remove |

## Filtering (SCIM Filter Language)

```bash
# Exact match
GET /scim/v2/Users?filter=userName eq "jane@corp.com"

# Contains
GET /scim/v2/Users?filter=displayName co "Jane"

# Multiple conditions
GET /scim/v2/Users?filter=active eq true and emails.value co "@corp.com"
```

Operators: `eq`, `ne`, `co`, `sw`, `ew`, `pr`, `gt`, `ge`, `lt`, `le`, `and`, `or`, `not`

## Pagination

```bash
GET /scim/v2/Users?startIndex=1&count=100
# → {"totalResults":1547, "startIndex":1, "itemsPerPage":100, "Resources":[...]}
```

Max count: 1000. Default: 100.

## PATCH Operations

```bash
PATCH /scim/v2/Users/{id}
{"Operations": [
  {"op":"replace", "path":"emails[type eq \"work\"].value", "value":"new@corp.com"},
  {"op":"add", "path":"groups", "value":[{"value":"grp-sales"}]},
  {"op":"remove", "path":"phoneNumbers[type eq \"mobile\"]"}
]}
```

## Enterprise Schema Extension

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User",
              "urn:ietf:params:scim:schemas:extension:enterprise:2.0"],
  "userName": "jane@corp.com",
  "urn:ietf:params:scim:schemas:extension:enterprise:2.0": {
    "department": "Engineering",
    "manager": {"value": "manager-uuid"},
    "costCenter": "CC-1001"
  }
}
```

## IdP Integration

| IdP | SCIM Config | Notes |
|-----|------------|-------|
| Okta | App → Provisioning → SCIM | Auto-discovers endpoints |
| Azure AD | Enterprise App → Provisioning → SCIM | Needs bearer token |
| OneLogin | App → Configuration → SCIM | Standard SCIM 2.0 |
| Workday | Integration → SCIM | Full lifecycle sync |

### Okta Integration Example

```
1. Configure SCIM app in Okta
2. Set SCIM base URL: https://auth.ggid.dev/scim/v2
3. Set auth: Bearer token (from GGID admin)
4. Okta auto-discovers ServiceProviderConfig
5. Assign users/groups in Okta → GGID receives SCIM requests
```

## Monitoring

| Metric | Alert |
|--------|-------|
| SCIM request latency | >500ms → optimize |
| SCIM errors | >1% → check payload or auth |
| Provisioning lag | >30s → queue backlog |
| Filter query timeout | Any → add DB indexes |

## See Also

- [SCIM 2.0 Implementation](scim-2-0-implementation.md)
- [User Provisioning Pipeline](user-provisioning-pipeline.md)
- [Digital Identity Lifecycle](digital-identity-lifecycle.md)
- [Identity Provider Configuration](identity-provider-configuration.md)