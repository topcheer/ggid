# Per-Tenant Identity Provider Configuration

> Configure SAML and OIDC identity providers per tenant for enterprise SSO.

---

## Overview

Each tenant can have its own set of identity providers (IdPs). Tenant A can use Okta SAML while Tenant B uses Azure AD OIDC — independently configured.

```
Tenant A (acme.com)  → SAML IdP: Okta
Tenant B (globex.com) → OIDC IdP: Azure AD
Tenant C (initech.com) → LDAP: Active Directory
```

---

## Configure SAML per Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT_A/saml/config \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_A" \
  -d '{
    "entity_id": "https://acme.com/saml/sp",
    "idp_metadata_url": "https://acme.okta.com/app/exk123/sso/saml/metadata",
    "acs_url": "https://ggid.example.com/saml/acs",
    "cert": "-----BEGIN CERTIFICATE-----\n...",
    "sign_assertions": true
  }'
```

## Configure OIDC per Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT_B/oidc/config \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_B" \
  -d '{
    "issuer": "https://login.microsoftonline.com/$AZURE_TENANT/v2.0",
    "client_id": "azure-client-id",
    "client_secret": "azure-secret",
    "redirect_uri": "https://ggid.example.com/oidc/callback",
    "scopes": ["openid", "email", "profile"]
  }'
```

---

## Domain-Based Routing

GGID routes users to the correct tenant IdP based on email domain:

```bash
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT_A/domains \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_A" \
  -d '{"domains": ["acme.com", "acme-corp.com"]}'
```

When user enters `alice@acme.com` at login, GGID auto-redirects to Acme's SAML IdP.

---

## JIT Provisioning

When a user authenticates via an external IdP for the first time, GGID auto-creates their local account:

```bash
# Enable JIT provisioning per tenant
curl -X PUT http://localhost:8080/api/v1/tenants/$TENANT_A/settings \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -d '{"jit_provisioning": true, "default_role": "member"}'
```

---

*See: [Social Login Setup](social-login-setup.md) | [Multi-Tenant Guide](multi-tenant-guide.md) | [Authentication Guide](../authentication-guide.md)*

*Last updated: 2025-07-11*
