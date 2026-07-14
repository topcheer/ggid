# Identity Provider Configuration Guide

SAML IdP, OIDC provider, social login, LDAP directory, multi-IdP routing, discovery, and failover configuration.

## Overview

GGID can consume authentication from multiple upstream identity providers. Users from different sources (SAML, OIDC, LDAP, social) are unified into GGID's identity model.

## SAML IdP Configuration

### Import IdP Metadata

```bash
POST /api/v1/identity/federation/saml/import
{
  "metadata_url": "https://idp.partner.com/metadata.xml",
  "name": "Partner SAML IdP",
  "attribute_mapping": {
    "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "email",
    "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": "display_name",
    "urn:oid:2.5.4.11": "department"
  }
}
# → 201 {id: "idp-saml-uuid", status: "active"}
```

### Manual Configuration

```bash
POST /api/v1/identity/federation/saml
{
  "name": "Corporate ADFS",
  "entity_id": "https://adfs.corp.com",
  "sso_url": "https://adfs.corp.com/adfs/ls/",
  "slo_url": "https://adfs.corp.com/adfs/ls/?wa=wsignout1.0",
  "signing_cert": "-----BEGIN CERTIFICATE-----\n...",
  "name_id_format": "urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
  "want_assertions_signed": true,
  "want_assertions_encrypted": true
}
```

### SP Metadata (GGID's endpoints)

```bash
GET /saml/metadata.xml
# → EntityDescriptor with:
#   EntityID: https://auth.ggid.dev/saml
#   ACS URL: https://auth.ggid.dev/saml/acs
#   SLO URL: https://auth.ggid.dev/saml/slo
```

## OIDC Provider Configuration

### Add OIDC IdP

```bash
POST /api/v1/identity/federation/oidc
{
  "name": "Corporate Okta",
  "issuer": "https://corp.okta.com",
  "authorization_endpoint": "https://corp.okta.com/oauth2/v1/authorize",
  "token_endpoint": "https://corp.okta.com/oauth2/v1/token",
  "userinfo_endpoint": "https://corp.okta.com/oauth2/v1/userinfo",
  "jwks_uri": "https://corp.okta.com/oauth2/v1/keys",
  "client_id": "ggid-client-id",
  "client_secret": "...",
  "scope": "openid profile email groups",
  "claim_mapping": {
    "email": "email",
    "display_name": "name",
    "department": "department"
  }
}
```

### Auto-Discovery

```bash
POST /api/v1/identity/federation/oidc/discover
{"issuer": "https://corp.okta.com"}
# → Auto-fetches /.well-known/openid-configuration
# → Returns discovered endpoints for confirmation
```

## Social Login Configuration

```bash
POST /api/v1/identity/federation/social
{
  "provider": "google",       // google, github, microsoft, apple, etc.
  "client_id": "...",
  "client_secret": "...",
  "scope": "openid email profile"
}
```

### Supported Social Providers

| Provider | Scopes Available | Claim Mapping |
|----------|----------------|---------------|
| Google | openid, email, profile | email, name, picture |
| GitHub | user:email, read:user | email, login, name |
| Microsoft | openid, email, profile | email, name, department |
| Apple | openid, email, name | email, name |
| GitLab | openid, email, profile | email, username, name |
| LinkedIn | r_liteprofile, r_emailaddress | email, name |
| Discord | identify, email | email, username |

### Social Provider Registry

```go
type SocialProvider struct {
    Name         string
    AuthURL      string
    TokenURL     string
    UserInfoURL  string
    Scopes       []string
    ClaimMapper  func(map[string]interface{}) UserClaims
}

var Registry = map[string]SocialProvider{
    "google":    {AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", ...},
    "github":    {AuthURL: "https://github.com/login/oauth/authorize", ...},
    "microsoft": {AuthURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize", ...},
}
```

## LDAP Directory Configuration

```bash
POST /api/v1/identity/federation/ldap
{
  "name": "Corporate LDAP",
  "url": "ldap://ldap.corp.com:389",
  "start_tls": true,
  "bind_dn": "cn=ggid,ou=service,dc=corp,dc=com",
  "bind_password": "...",
  "base_dn": "ou=users,dc=corp,dc=com",
  "user_filter": "(uid={username})",
  "auto_provision": true,
  "attribute_mapping": {
    "mail": "email",
    "cn": "display_name",
    "department": "department",
    "memberOf": "groups"
  }
}
```

### LDAP Sync (Scheduled)

```bash
# Configure scheduled sync
POST /api/v1/identity/federation/ldap/{id}/sync-config
{
  "schedule": "0 */6 * * *",    // Every 6 hours
  "create_missing": true,        // Auto-provision new users
  "deactivate_missing": true,    // Deactivate users removed from LDAP
  "update_attributes": true      // Sync attribute changes
}
```

## Multi-IdP Routing

### Domain-Based Routing

```bash
POST /api/v1/identity/federation/routing
{
  "rules": [
    {
      "match": {"domain": "@corp.com"},
      "idp_id": "idp-saml-corporate"
    },
    {
      "match": {"domain": "@partner.com"},
      "idp_id": "idp-oidc-partner"
    },
    {
      "match": {"email_regex": ".*@gmail\\.com"},
      "idp_id": "social-google"
    }
  ],
  "default": "local"   // Fall back to local password auth
}
```

### Login Flow with Routing

```
User enters email
    │
    ▼
Router evaluates domain/regex
    │
    ├── @corp.com → Redirect to SAML IdP
    ├── @partner.com → Redirect to OIDC IdP
    ├── @gmail.com → Google social login
    └── Other → Local password login
```

## IdP Discovery

### Email-Based Discovery

```bash
GET /api/v1/auth/discover?email=user@corp.com
# → {
#   "providers": [
#     {"type": "saml", "name": "Corporate SSO", "redirect_url": "..."},
#     {"type": "local", "name": "Password"}
#   ]
# }
```

### Client-Based Discovery

OAuth clients can specify which IdPs they support:

```bash
GET /api/v1/auth/discover?client_id=app-123
# → Returns only IdPs this client is authorized to use
```

## IdP Failover

```yaml
failover:
  primary:
    idp_id: "idp-saml-corporate"
    health_check: true
    check_interval: 30s

  fallback:
    idp_id: "idp-oidc-backup"
    trigger: "primary_unhealthy"
    auto_promote: true

  rules:
    - If primary fails 3 consecutive health checks → use fallback
    - When primary recovers → wait 5 min before switching back
    - Log all failover events to audit
```

### Health Check

```bash
GET /api/v1/identity/federation/health
# → [
#   {"idp_id": "idp-saml-corporate", "status": "healthy", "latency_ms": 145},
#   {"idp_id": "idp-oidc-backup", "status": "healthy", "latency_ms": 89}
# ]
```

## Monitoring

| Metric | Alert |
|--------|-------|
| IdP response time | >2s → investigate |
| IdP auth failures | >5% → check cert expiry |
| LDAP sync failures | Any → investigate connectivity |
| Failover triggered | Any → page ops |
| Provisioning errors | >1% → check attribute mapping |

## See Also

- [Identity Federation Architecture](identity-federation-architecture.md)
- [SCIM 2.0 Implementation](scim-2-0-implementation.md)
- [Identity Lifecycle Automation](identity-lifecycle-automation.md)
- [Authentication Flows](authentication-flows.md)
