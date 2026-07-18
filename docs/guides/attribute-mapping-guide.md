# Attribute Mapping Guide (KB-063)

## Overview

GGID's attribute mapping engine transforms identity attributes from external sources (SAML, OIDC, SCIM, HR connectors) into internal GGID user attributes — enabling consistent policy evaluation regardless of IdP.

## Mapping Sources

| Source | Protocol | Example |
|--------|----------|---------|
| SAML IdP | SAML 2.0 | `saml:Attribute → ggid:department` |
| OIDC Provider | OpenID Connect | `id_token.groups → ggid:roles` |
| SCIM | SCIM 2.0 | `scim:userName → ggid:email` |
| HR Connector | API/CSV | `workday.title → ggid:job_title` |

## Configuration

### Create Mapping
```http
POST /api/v1/policies/attribute-mapping
Content-Type: application/json

{
  "source": "saml:azure-ad",
  "source_attribute": "http://schemas.xmlsoap.org/claims/Group",
  "target_attribute": "roles",
  "transform": "split:comma",
  "default_value": "viewer"
}
```

### List Mappings
```http
GET /api/v1/policies/attribute-mapping?source=saml
```

## Transform Functions

| Function | Input → Output | Description |
|----------|----------------|-------------|
| `identity` | `Alice` → `Alice` | No transform |
| `lower` | `Alice` → `alice` | Lowercase |
| `split:delim` | `a,b,c` → `[a,b,c]` | Split to array |
| `prefix:str` | `admin` → `role:admin` | Prepend prefix |
| `regex:pattern` | `CN=Admins` → `Admins` | Extract match |

## Evaluation Order

1. SAML/OIDC assertion received at login
2. Attribute mapping engine loads tenant mappings
3. Each source attribute is transformed → target
4. Merged into user's session claims
5. CAE engine evaluates policies against mapped attributes

## Best Practices

- Map to standard GGID attributes (`email`, `roles`, `department`, `manager`)
- Use `default_value` for missing attributes to prevent policy gaps
- Test mappings with dry-run before enabling
- Audit mapping changes — they affect access decisions
