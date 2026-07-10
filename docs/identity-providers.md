# Identity Provider Integration

External IdP integration guide: SAML IdP configuration, OIDC provider setup,
social login federation, trust relationships, and attribute mapping.

> **See also**: [SAML Guide](saml-guide.md), [Social Login](social-login-guide.md),
> [OAuth Flows](oauth-flows.md), [LDAP Integration](ldap-integration.md).

---

## Table of Contents

- [Federation Model](#federation-model)
- [SAML IdP Configuration](#saml-idp-configuration)
- [OIDC Provider Setup](#oidc-provider-setup)
- [Social Login Federation](#social-login-federation)
- [LDAP/Active Directory](#ldapactive-directory)
- [Federation Trust](#federation-trust)
- [Attribute Mapping Reference](#attribute-mapping-reference)

---

## Federation Model

GGID supports multiple external IdPs simultaneously. Users authenticate
against their home IdP, and GGID creates/federates their identity.

```
┌─────────────────────────────────────────────┐
│               GGID (SP / RP)                 │
│                                              │
│  ┌─────────┐ ┌─────────┐ ┌────────────────┐ │
│  │ SAML    │ │ OIDC    │ │ Social (OAuth) │ │
│  │ IdP     │ │ Provider│ │ Connectors     │ │
│  │ Module  │ │ Module  │ │                │ │
│  └────┬────┘ └────┬────┘ └───────┬────────┘ │
│       │           │              │          │
│  ┌────▼────────────▼──────────────▼───────┐ │
│  │     Attribute Mapping Engine           │ │
│  │     (normalize → map → provision)      │ │
│  └───────────────────────────────────────┘ │
│       │                                     │
│  ┌────▼───────────────────────────────────┐ │
│  │     GGID User Directory                │ │
│  │     (users, groups, roles)             │ │
│  └───────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

### Supported Protocols

| Protocol | Use Case | Examples |
|----------|----------|----------|
| SAML 2.0 | Enterprise SSO | Okta, Azure AD, AD FS |
| OIDC | Federated SSO | Auth0, Keycloak, Google |
| OAuth 2.0 | Social login | GitHub, Discord, Slack |
| LDAP | Directory sync | Active Directory, OpenLDAP |

---

## SAML IdP Configuration

### Register External SAML IdP

```bash
curl -X POST https://iam.example.com/api/v1/admin/saml/idp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "name": "Corporate Okta",
    "metadata_url": "https://corp.okta.com/app/exkabc/federationmetadata",
    "name_id_format": "emailAddress",
    "attribute_mapping": {
      "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
      "first_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
      "last_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"
    }
  }'
```

### Multiple IdPs

GGID supports multiple SAML IdPs per tenant. A discovery page lets users
choose their IdP:

```
Login Page:
  ┌─────────────────────────────┐
  │  Sign in with:              │
  │  ┌───────────────────────┐  │
  │  │ GGID Account          │  │
  │  └───────────────────────┘  │
  │  ┌───────────────────────┐  │
  │  │ Corporate Okta (SAML) │  │
  │  └───────────────────────┘  │
  │  ┌───────────────────────┐  │
  │  │ Azure AD (SAML)       │  │
  │  └───────────────────────┘  │
  └─────────────────────────────┘
```

---

## OIDC Provider Setup

### Register External OIDC Provider

```bash
curl -X POST .../admin/oidc/providers \
  -d '{
    "name": "Auth0 Tenant",
    "issuer": "https://login.auth0.com",
    "client_id": "your-client-id",
    "client_secret": "your-client-secret",
    "scopes": ["openid", "email", "profile"],
    "redirect_url": "https://iam.example.com/oauth/callback/auth0"
  }'
```

### OIDC Discovery

GGID automatically fetches the provider's `.well-known/openid-configuration`
to discover endpoints:

```json
{
  "issuer": "https://login.auth0.com",
  "authorization_endpoint": "https://login.auth0.com/authorize",
  "token_endpoint": "https://login.auth0.com/oauth/token",
  "userinfo_endpoint": "https://login.auth0.com/userinfo",
  "jwks_uri": "https://login.auth0.com/.well-known/jwks.json"
}
```

---

## Social Login Federation

### Configure Social Providers

See [Social Login Guide](social-login-guide.md) for detailed setup of each
provider (Google, GitHub, Microsoft, Apple, GitLab, Discord, Slack, LinkedIn, OIDC).

### Federation Flow

```
1. User clicks "Sign in with Google"
2. GGID redirects to Google OAuth consent
3. User authenticates with Google
4. Google redirects back with authorization code
5. GGID exchanges code for access_token + id_token
6. GGID verifies id_token signature (Google JWKS)
7. GGID extracts claims (email, name, picture)
8. GGID links or creates user account
9. GGID issues own JWT for the user
```

---

## LDAP/Active Directory

See [LDAP Integration](ldap-integration.md) and [LDAP Directory Sync](ldap-directory-sync.md)
for configuring Active Directory / OpenLDAP as an authentication provider.

### Authentication Chain Order

```yaml
auth:
  provider_chain:
    - local          # GGID internal database (priority 10)
    - ldap           # Active Directory (priority 20)
    - saml:okta      # Okta SAML (priority 30)
    - oidc:auth0     # Auth0 OIDC (priority 40)
```

The chain tries each provider in order. First successful authentication wins.

---

## Federation Trust

### Trust Model

| Trust Type | Description |
|-----------|-------------|
| Implicit | GGID trusts IdP's authentication, creates user if missing |
| Explicit | GGID only authenticates users that exist in its directory |
| JIT Provisioning | GGID auto-creates user on first login from trusted IdP |

### JIT Provisioning

```yaml
federation:
  jit_provisioning: true
  create_user_if_missing: true
  default_role: "viewer"
  default_org: "federated"
  sync_attributes: ["email", "name", "department", "manager"]
```

### Attribute Query

```bash
# Query which IdP authenticated a user
curl .../users/{user_id}/federation \
  -H "Authorization: Bearer $TOKEN"
```

```json
{
  "provider": "saml:okta",
  "provider_user_id": "okta-12345",
  "first_auth": "2024-01-10T10:00:00Z",
  "last_auth": "2024-01-20T09:00:00Z",
  "attributes_synced": ["email", "name", "department"]
}
```

---

## Attribute Mapping Reference

### Standard SAML Attributes

| SAML Attribute URI | GGID Field |
|-------------------|------------|
| `.../claims/emailaddress` | email |
| `.../claims/givenname` | first_name |
| `.../claims/surname` | last_name |
| `.../claims/name` | display_name |
| `.../claims/department` | department |
| `.../claims/groups` | groups (→ role mapping) |
| `.../claims/role` | direct role assignment |

### OIDC Claims

| OIDC Claim | GGID Field |
|-----------|------------|
| `sub` | provider_user_id |
| `email` | email |
| `email_verified` | email_verified |
| `name` | display_name |
| `given_name` | first_name |
| `family_name` | last_name |
| `picture` | avatar_url |
| `locale` | locale |
| `preferred_username` | username |

### LDAP Attributes

| LDAP Attribute | GGID Field |
|---------------|------------|
| `sAMAccountName` (AD) | username |
| `uid` (OpenLDAP) | username |
| `mail` | email |
| `displayName` | display_name |
| `givenName` | first_name |
| `sn` | last_name |
| `memberOf` | groups (→ role mapping) |
| `department` | department |
| `manager` | manager_dn |

### Social Provider Attributes

| Provider | Username | Email | Name |
|----------|----------|-------|------|
| Google | (from email) | email | name |
| GitHub | login | email | name |
| Microsoft | upn | email | displayName |
| Apple | (relay) | email | name (first only) |
| GitLab | username | email | name |
| Discord | username | email | username |
| Slack | name | email | real_name |
| LinkedIn | (from email) | email | localizedFirstName + localizedLastName |

### Group-to-Role Mapping

```bash
curl -X PATCH .../admin/saml/idp/{id} \
  -d '{
    "group_mapping": {
      "Domain Admins": "admin",
      "Developers": "developer",
      "Viewers": "viewer"
    }
  }'
```

When a SAML assertion or LDAP group membership includes "Domain Admins",
GGID automatically assigns the `admin` role.
