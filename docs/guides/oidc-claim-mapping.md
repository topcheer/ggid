# OIDC Claim Mapping

This guide covers OIDC claim types, mapping rules, transformation pipelines, and per-client customization in GGID.

## Claim Types

### Standard Claims (RFC 7519 + OIDC)

| Claim | Description | Source |
|---|---|---|
| `sub` | Subject identifier | User ID |
| `name` | Full name | display_name |
| `given_name` | First name | first_name |
| `family_name` | Last name | last_name |
| `middle_name` | Middle name | middle_name |
| `nickname` | Casual name | nickname |
| `preferred_username` | Username | username |
| `profile` | Profile URL | profile_url |
| `picture` | Avatar URL | avatar_url |
| `website` | Personal URL | website |
| `email` | Email address | email |
| `email_verified` | Email verification status | boolean |
| `gender` | Gender | gender |
| `birthdate` | Birth date | birthdate |
| `zoneinfo` | Timezone | timezone |
| `locale` | Locale preference | locale |
| `phone_number` | Phone number | phone |
| `phone_number_verified` | Phone verification status | boolean |
| `address` | Mailing address | address JSON |
| `updated_at` | Last update timestamp | unix epoch |

### Custom Claims

GGID-specific claims beyond the standard set:

| Claim | Description |
|---|---|
| `tenant_id` | Tenant identifier |
| `roles` | Array of role names |
| `permissions` | Array of permission strings |
| `groups` | Array of group names |
| `department` | Organizational unit |
| `security_level` | Clearance level |
| `mfa_verified` | MFA completion flag |
| `delegation_chain` | Agent delegation chain |

### Multi-Valued Claims

Claims with multiple values are represented as JSON arrays:

```json
{
  "roles": ["admin", "auditor"],
  "groups": ["engineering", "platform-team"],
  "permissions": ["users:read", "users:write", "policy:read"]
}
```

## Claim Mapping Rules

### IdP Attribute → GGID Claim

```yaml
claim_mapping:
  # LDAP attributes
  "uid": "preferred_username"
  "mail": "email"
  "cn": "name"
  "givenName": "given_name"
  "sn": "family_name"
  "telephoneNumber": "phone_number"
  "title": "job_title"
  "departmentNumber": "department"

  # SAML attributes
  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "email"
  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": "name"
  "http://schemas.microsoft.com/ws/2008/06/identity/claims/role": "roles"

  # SCIM attributes
  "userName": "preferred_username"
  "displayName": "name"
  "emails[type='work'].value": "email"
  "phoneNumbers[type='mobile'].value": "phone_number"
```

### Mapping Configuration

```yaml
oidc:
  claim_mapping:
    source: "ldap"  # or "saml", "scim", "database"
    rules:
      - source_attr: "mail"
        target_claim: "email"
        required: true
      - source_attr: "departmentNumber"
        target_claim: "department"
        required: false
        default: "unassigned"
```

## Scope-to-Claim Mapping

GGID maps OAuth/OIDC scopes to claim sets:

| Scope | Claims Released |
|---|---|
| `openid` | `sub` |
| `profile` | `name`, `family_name`, `given_name`, `middle_name`, `nickname`, `preferred_username`, `profile`, `picture`, `website`, `gender`, `birthdate`, `zoneinfo`, `locale`, `updated_at` |
| `email` | `email`, `email_verified` |
| `address` | `address` |
| `phone` | `phone_number`, `phone_number_verified` |
| `groups` | `groups`, `roles` |
| `tenant` | `tenant_id`, `department` |

```yaml
oidc:
  scope_claims:
    openid: ["sub"]
    profile: ["name", "given_name", "family_name", "preferred_username", "picture", "locale"]
    email: ["email", "email_verified"]
    phone: ["phone_number", "phone_number_verified"]
    groups: ["groups", "roles", "permissions"]
    tenant: ["tenant_id", "department"]
```

### Custom Scopes

```yaml
oidc:
  custom_scopes:
    "hr:data":
      claims: ["employee_id", "department", "manager"]
      description: "HR-related user attributes"
    "security:info":
      claims: ["security_level", "mfa_verified", "last_login"]
      description: "Security-related attributes"
```

## Claim Transformation Pipeline

### Pipeline Stages

```
Source Attributes → Mapping → Transformation → Filtering → Token
```

### Regex Transformation

```yaml
transformations:
  - claim: "email"
    type: "regex"
    pattern: "@(.*)$"
    replacement: "@$1"
    # Normalize domain: user@EXAMPLE.COM → user@example.com
```

### Lookup Transformation

```yaml
transformations:
  - claim: "department"
    type: "lookup"
    table:
      "eng": "Engineering"
      "sales": "Sales"
      "ops": "Operations"
    default: "Unknown"
```

### Computed Transformation

```yaml
transformations:
  - claim: "full_name"
    type: "computed"
    expression: "{{.given_name}} {{.family_name}}"
  - claim: "display_label"
    type: "computed"
    expression: "{{.preferred_username}} ({{.department}})"
```

### Conditional Transformation

```yaml
transformations:
  - claim: "security_level"
    type: "conditional"
    conditions:
      - if: "roles contains 'admin'"
        value: "high"
      - if: "roles contains 'developer'"
        value: "medium"
      - else: "low"
```

## Multi-IdP Claim Normalization

When users authenticate through different IdPs, claim names may differ:

| IdP | Email Attribute | Role Attribute |
|---|---|---|
| LDAP | `mail` | `memberUid` → group lookup |
| SAML (AD) | `http://.../emailaddress` | `http://.../role` |
| SAML (Okta) | `email` | `groups` |
| Google | `email` | `hd` (hosted domain) |
| GitHub | `email` | `organizations` |

### Normalization Rules

```yaml
normalization:
  email:
    sources: ["mail", "email", "http://.../emailaddress", "emails[0].value"]
    transform: "lowercase"
    required: true
  roles:
    sources: ["groups", "http://.../role", "memberUid"]
    transform: "flatten"  # Convert nested arrays to flat array
    deduplicate: true
  name:
    sources: ["cn", "displayName", "name", "http://.../name"]
    transform: "trim"
    required: false
    default: "Unknown"
```

## Claim Restrictions (PII vs Non-PII)

### PII Classification

| Claim | PII Level | Restrictions |
|---|---|---|
| `email` | PII | Encrypted at rest, audit on access |
| `phone_number` | PII | Encrypted at rest, audit on access |
| `address` | PII | Encrypted at rest, audit on access |
| `birthdate` | PII | Encrypted at rest, audit on access |
| `sub` | Non-PII | No restrictions |
| `roles` | Non-PII | No restrictions |
| `tenant_id` | Non-PII | No restrictions |

### PII Protection Policy

```yaml
pii:
  sensitive_claims: ["email", "phone_number", "address", "birthdate"]
  encryption: true
  audit_access: true
  masking_in_logs: true
  require_explicit_consent: true
```

### Consent-Based Claim Release

Claims marked as PII require user consent before release to a client:

```
User logs in → GGID shows consent screen:
  "App X wants to access: email, profile"
  [Allow] [Deny] [Allow once]
```

## Claim Caching Strategy

```yaml
claim_cache:
  enabled: true
  ttl_active: 300s    # Short TTL for active tokens
  ttl_inactive: 3600s # Longer TTL for inactive/expired
  max_entries: 100000
  eviction: "lru"
  invalidation:
    on_user_update: true
    on_role_change: true
    on_tenant_config_change: true
```

### Cache Key

```
cache_key = hash(user_id + tenant_id + client_id + scope_set)
```

### Invalidation Triggers

- User profile update → invalidate all claims for user
- Role assignment change → invalidate `roles`, `permissions`
- Tenant config change → invalidate all claims in tenant
- Password change → invalidate all claims for user

## Per-Client Claim Customization

### Client-Specific Claim Rules

```yaml
client_claims:
  "web-frontend":
    extra_claims:
      ui_theme: "dark"
      feature_flags: ["beta_dashboard", "new_nav"]
    suppress_claims:
      - phone_number
      - address
    scope_override:
      profile: ["name", "preferred_username", "picture"]

  "mobile-app":
    extra_claims:
      app_version_min: "2.0.0"
    suppress_claims:
      - address
      - birthdate

  "admin-cli":
    extra_claims:
      admin_session: true
      audit_level: "detailed"
    allow_all_claims: true
```

### Client Claim Policy

```yaml
client_policies:
  default:
    max_claims: 20
    require_consent_for_pii: true
    allow_custom_claims: false
  trusted:
    max_claims: 50
    require_consent_for_pii: false
    allow_custom_claims: true
```

## Token Types and Claims

| Token | Claims Included |
|---|---|
| Access Token | `sub`, `iss`, `aud`, `exp`, `iat`, `scope`, `tenant_id`, `roles` |
| ID Token | All profile/email/phone claims per granted scopes |
| Refresh Token | `sub`, `iss`, `aud`, `exp`, `jti`, `tenant_id` |
| Agent Token | `sub`, `iss`, `aud`, `exp`, `delegation_chain`, `mcp_servers`, `max_delegation_depth` |

## Best Practices

1. **Minimal claim release** — Only include claims the client needs
2. **Scope-based filtering** — Map scopes to claim sets, don't release everything
3. **PII consent** — Require explicit consent for PII claims
4. **Cache with invalidation** — Cache for performance, invalidate on changes
5. **Normalize across IdPs** — Ensure consistent claim names regardless of source
6. **Audit claim access** — Log when PII claims are released to which client
7. **Per-client customization** — Tailor claim sets per client use case
8. **Test claim mappings** — Validate mappings with real IdP data before production