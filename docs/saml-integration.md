# SAML 2.0 Integration Guide

How to configure GGID as a SAML 2.0 Service Provider and integrate with external Identity Providers.

---

## Overview

GGID acts as a **SAML 2.0 Service Provider (SP)**. When a user attempts to access a resource, GGID redirects to a configured Identity Provider (IdP) for authentication. The IdP sends a signed SAML Assertion back, which GGID verifies to log the user in.

### SAML Roles

| Role | Who | Responsibility |
|------|-----|----------------|
| Service Provider (SP) | GGID | Requests authentication, verifies assertion |
| Identity Provider (IdP) | Okta/Azure AD/ADFS | Authenticates user, issues SAML assertion |

---

## SP-Initiated Flow

```
Browser              GGID (SP)              IdP (Okta/Azure AD)
  │                     │                        │
  │ 1. Access resource  │                        │
  ├────────────────────►│                        │
  │                     │ User not authenticated │
  │                     │ Generate SAML AuthnRequest             │
  │ 2. 302 Redirect     │                        │
  │   to IdP SSO URL    │                        │
  │◄────────────────────┤                        │
  │                     │                        │
  │ 3. AuthnRequest     │                        │
  ├─────────────────────────────────────────────►│
  │                     │                        │ User sees IdP login page
  │ 4. Login form       │                        │
  │◄─────────────────────────────────────────────┤
  │                     │                        │
  │ 5. User authenticates                       │
  ├─────────────────────────────────────────────►│
  │                     │                        │ Verify credentials
  │                     │                        │ Generate SAML Assertion
  │                     │                        │ Sign with IdP private key
  │ 6. POST SAMLResponse│                        │
  │   to GGID ACS URL   │                        │
  │◄─────────────────────────────────────────────┤
  │                     │                        │
  │ 7. SAMLResponse     │                        │
  ├────────────────────►│                        │
  │                     │ Verify signature       │
  │                     │ (using IdP public cert)│
  │                     │ Validate conditions    │
  │                     │ Extract attributes     │
  │                     │ Map to GGID user       │
  │                     │ JIT provision (if new) │
  │                     │ Issue JWT              │
  │ 8. 302 Redirect +   │                        │
  │   JWT cookie/token  │                        │
  │◄────────────────────┤                        │
```

---

## GGID SAML Configuration

### SP Metadata

GGID exposes SP metadata at:

```
GET /saml/metadata
```

This returns XML metadata containing:
- Entity ID
- ACS (Assertion Consumer Service) URL
- SP X.509 certificate
- Supported bindings (HTTP-POST, HTTP-Redirect)

### Configure IdP Connection

```bash
POST /api/v1/idp/config
{
  "type": "saml",
  "name": "Corporate Okta",
  "entity_id": "http://www.okta.com/exkabc123",
  "sso_url": "https://corp.okta.com/app/corp/ggid/exkabc123/sso/saml",
  "slo_url": "https://corp.okta.com/app/corp/ggid/exkabc123/slo/saml",
  "x509_cert": "MIIDpDCCAoygAwIBAgIGAX9...",
  "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
}
```

| Field | Description |
|-------|-------------|
| `entity_id` | IdP's unique identifier |
| `sso_url` | IdP's Single Sign-On endpoint |
| `slo_url` | IdP's Single Logout endpoint |
| `x509_cert` | IdP's public certificate (for signature verification) |
| `name_id_format` | Format of the NameID in the assertion |

### Attribute Mapping

Map SAML assertion attributes to GGID user fields:

```bash
PUT /api/v1/idp/config/{id}/attribute-mapping
{
  "email": "email",
  "username": "email",
  "first_name": "givenName",
  "last_name": "surname",
  "department": "department",
  "display_name": "displayName"
}
```

---

## Okta Integration

### 1. Get Okta Metadata

In Okta Admin:
1. **Applications** → **Create App Integration** → **SAML 2.0**
2. Configure:
   - **Single sign-on URL (ACS):** `https://iam.example.com/saml/acs`
   - **Audience URI (SP Entity ID):** `https://iam.example.com/saml/metadata`
   - **Name ID format:** Email
   - **Application username:** Email
3. **Attribute Statements:**
   - `email` = `user.email`
   - `givenName` = `user.firstName`
   - `surname` = `user.lastName`
   - `department` = `user.department`
4. Download the **Identity Provider metadata** or certificate

### 2. Configure in GGID

```bash
POST /api/v1/idp/config
{
  "type": "saml",
  "name": "Okta SSO",
  "entity_id": "http://www.okta.com/exkabc123",
  "sso_url": "https://corp.okta.com/app/corp/ggid/exkabc123/sso/saml",
  "x509_cert": "<cert-from-okta>",
  "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
}
```

### 3. Test Login

```
https://iam.example.com/saml/login?idp=okta
→ Redirects to Okta login
→ After auth, redirected back with JWT
```

---

## Azure AD (Entra ID) Integration

### 1. Create Enterprise App

1. Azure Portal → **Enterprise Applications** → **New Application** → **Create your own**
2. Select **Integrate any other application you don't find in the gallery**
3. Configure SAML:
   - **Identifier (Entity ID):** `https://iam.example.com/saml/metadata`
   - **Reply URL (ACS):** `https://iam.example.com/saml/acs`
   - **Logout URL:** `https://iam.example.com/saml/slo`
4. Download **Federation Metadata XML** from the SAML certificates section

### 2. Attribute Claims

| Claim Name | Source |
|-----------|--------|
| `email` | `user.userprincipalname` |
| `givenName` | `user.givenname` |
| `surname` | `user.surname` |
| `department` | `user.department` |

### 3. Configure in GGID

```bash
POST /api/v1/idp/config
{
  "type": "saml",
  "name": "Azure AD",
  "entity_id": "https://sts.windows.net/{tenant-id}/",
  "sso_url": "https://login.microsoftonline.com/{tenant-id}/saml2",
  "x509_cert": "<cert-from-metadata-xml>"
}
```

---

## ADFS Integration

### 1. Add Relying Party Trust

In ADFS Management:
1. **Relying Party Trusts** → **Add**
2. Import GGID SP metadata from `https://iam.example.com/saml/metadata`
3. Configure Claim Rules:
   - `email` ← `E-Mail-Address`
   - `givenName` ← `Given-Name`
   - `surname` ← `Surname`

### 2. Export ADFS Certificate

```bash
# Export token signing certificate
# ADFS Management → Service → Certificates → Token-signing
```

### 3. Configure in GGID

```bash
POST /api/v1/idp/config
{
  "type": "saml",
  "name": "ADFS",
  "entity_id": "https://adfs.corp.local/federationmetadata/2007-06/federationmetadata.xml",
  "sso_url": "https://adfs.corp.local/adfs/ls/",
  "x509_cert": "<adfs-signing-cert>"
}
```

---

## Assertion Parsing

GGID parses SAML assertions using `pkg/saml`:

```go
// Parse assertion from SAML response
assertion, err := saml.ParseAssertion(samlResponseXML)
if err != nil {
    return err
}

// Validate conditions (time window, audience)
if err := assertion.ValidateConditions(); err != nil {
    return err
}

// Verify signature with IdP certificate
if err := saml.ValidateSignature(assertion, idpCert); err != nil {
    return err
}

// Extract attributes
attrs := saml.ExtractAttributes(assertion)
email := saml.GetAttribute(assertion, "email")
```

### Verified Attributes

| Attribute | Required | Default NameID |
|-----------|:--------:|----------------|
| Email | Yes | NameID value |
| Username | No | Falls back to email |
| First Name | No | — |
| Last Name | No | — |
| Groups | No | For role mapping |

---

## JIT Provisioning

When a user authenticates via SAML for the first time and doesn't exist in GGID:

1. GGID checks if JIT is enabled for the IdP
2. Creates user from SAML attributes
3. Sets `email_verified = true` (IdP verified the email)
4. Publishes audit event: `user.register` (method: saml)
5. Issues JWT

```bash
# Enable JIT for an IdP
PUT /api/v1/idp/config/{id}
{"auto_provision": true}
```

---

## Security Considerations

### Signature Verification

- GGID verifies every SAML assertion signature using the IdP's public certificate
- Unsigned assertions are **rejected** (no unsigned fallback)
- Certificate rotation: update via API, dual-cert period supported

### Replay Protection

- GGID checks the assertion's `NotOnOrAfter` time window
- Assertion ID stored in Redis to prevent replay (TTL = assertion validity)

### Certificate Rotation

```
1. Add new IdP certificate to GGID (without removing old)
2. Both certificates are valid during overlap period
3. IdP switches to new signing certificate
4. Remove old certificate from GGID
```
