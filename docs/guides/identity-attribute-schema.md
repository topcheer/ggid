# Identity Attribute Schema Guide

Standard attributes, custom attributes, lifecycle, multi-valued attributes, schema extension, validation rules, and privacy classification.

## Standard Attributes

| Attribute | Type | Required | Privacy Level | Example |
|-----------|------|----------|--------------|---------|
| `id` | UUID | Yes | L1 | `550e8400-e29b-41d4-a716-446655440000` |
| `email` | string | Yes | L3 | `jane@corp.com` |
| `email_verified` | boolean | Yes | L2 | `true` |
| `display_name` | string | Yes | L1 | `Jane Doe` |
| `first_name` | string | No | L1 | `Jane` |
| `last_name` | string | No | L1 | `Doe` |
| `phone` | string | No | L3 | `+1-555-0100` |
| `phone_verified` | boolean | No | L2 | `true` |
| `department` | string | No | L2 | `Engineering` |
| `title` | string | No | L2 | `Senior Engineer` |
| `status` | enum | Yes | L2 | `active` |
| `locale` | string | No | L1 | `en-US` |
| `timezone` | string | No | L1 | `America/New_York` |
| `avatar_url` | URL | No | L1 | `https://...` |
| `created_at` | timestamp | Yes | L2 | `2025-01-15T10:00:00Z` |
| `last_login_at` | timestamp | No | L3 | `2025-01-20T14:30:00Z` |

### Status Enum Values

| Value | Meaning | Login Allowed |
|-------|---------|--------------|
| `active` | Normal user | ✅ |
| `dormant` | No login in 90 days | ❌ |
| `suspended` | Admin-suspended | ❌ |
| `locked` | Security lock | ❌ |
| `pending` | Not yet activated | ❌ |
| `archived` | Anonymized | ❌ |

## Custom Attributes

### Define Custom Attribute

```bash
POST /api/v1/admin/attributes/custom
{
  "name": "clearance_level",
  "type": "enum",
  "enum_values": ["public", "internal", "confidential", "restricted", "secret"],
  "required": false,
  "privacy_level": "L3",
  "validation": {"max_length": 20},
  "searchable": true
}
```

### Custom Attribute Types

| Type | Validation | Example |
|------|-----------|---------|
| `string` | max_length, pattern | `"emp-1234"` |
| `integer` | min, max | `42` |
| `boolean` | — | `true` |
| `enum` | enum_values | `"secret"` |
| `date` | ISO 8601 | `"2025-01-15"` |
| `url` | valid URL | `"https://..."` |
| `json` | valid JSON | `{"key":"value"}` |
| `string[]` | max_items | `["a","b","c"]` |

## Multi-Valued Attributes

```json
{
  "emails": [
    {"value": "jane@corp.com", "type": "work", "primary": true},
    {"value": "jane.d@gmail.com", "type": "personal"}
  ],
  "phone_numbers": [
    {"value": "+1-555-0100", "type": "mobile", "primary": true},
    {"value": "+1-555-0200", "type": "office"}
  ],
  "addresses": [
    {"type": "home", "street": "123 Main St", "city": "NYC", "country": "US"}
  ],
  "groups": ["engineering", "developers", "on_call"]
}
```

### Querying Multi-Valued

```bash
# Find user by any email
GET /api/v1/identity/users?email=jane.d@gmail.com

# Find users in specific group
GET /api/v1/identity/users?group=on_call
```

## Schema Extension

### Enterprise Extension

```bash
POST /api/v1/admin/attributes/schema-extension
{
  "namespace": "enterprise",
  "attributes": [
    {"name": "employee_id", "type": "string", "required": true},
    {"name": "cost_center", "type": "string"},
    {"name": "manager_id", "type": "uuid"},
    {"name": "hire_date", "type": "date"},
    {"name": "termination_date", "type": "date"}
  ]
}
```

### Using Extension

```bash
GET /api/v1/identity/users/uuid
# → {
#   "id": "uuid",
#   "email": "jane@corp.com",
#   "enterprise": {
#     "employee_id": "EMP-1234",
#     "cost_center": "CC-1001",
#     "manager_id": "uuid-of-manager",
#     "hire_date": "2023-06-01"
#   }
# }
```

## Validation Rules

```yaml
validation:
  email:
    format: rfc5322
    max_length: 254
    unique: true
    normalize: lowercase
  
  phone:
    format: e164
    sanitize: strip_non_digits
    
  display_name:
    min_length: 1
    max_length: 100
    sanitize: trim_whitespace
    block_html: true
    
  password:
    min_length: 12
    max_length: 128
    require: [upper, lower, digit, special]
    pepper: true
    breach_check: hibp
  
  employee_id:
    pattern: "^EMP-[0-9]{4}$"
    unique: true
```

## Attribute Lifecycle

```
Created → Updated → Archived → Deleted (anonymized)
```

| Lifecycle Event | What Happens |
|----------------|-------------|
| Created | Default values set, required fields validated |
| Updated | Old value logged in audit trail |
| Archived | PII anonymized, non-PII retained |
| Deleted | All attributes hashed/anonymized |

### Attribute History

```sql
CREATE TABLE attribute_history (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    attribute_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    reason TEXT
);
```

## Privacy Classification per Attribute

| Level | Attributes | Masking | Access |
|-------|-----------|---------|--------|
| L1 (Public) | display_name, avatar_url | None | Anyone |
| L2 (Internal) | department, title, status | None | Authenticated |
| L3 (Confidential) | email, phone, last_login | In logs | Scoped access |
| L4 (Restricted) | password_hash, mfa_secret | Never exposed | Internal only |

### Scope-Based Release

```yaml
scope_attribute_mapping:
  openid: [sub]
  profile: [display_name, first_name, last_name, locale, timezone, avatar_url]
  email: [email, email_verified]
  phone: [phone, phone_verified]
  # Never released via OAuth: password_hash, mfa_secret, recovery_codes
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Custom attribute count | >50 per tenant → review |
| Validation failure rate | >5% → source data quality |
| Schema extension count | >5 per tenant → complexity |
| Unsearchable attribute usage | Any → add index |

## See Also

- [SCIM 2.0 Implementation](scim-2-0-implementation.md)
- [Privacy by Design](privacy-by-design.md)
- [Data Classification Implementation](data-classification-implementation.md)
- [User Provisioning Pipeline](user-provisioning-pipeline.md)
