# SAML 2.0 Integration Guide

Practical guide for SAML 2.0 integration: SP metadata configuration, IdP
setup, attribute mapping, NameID formats, signing certificates, logout
flows, and troubleshooting.

> **See also**: [SAML Configuration](saml-configuration.md) for detailed
> cert rotation and encryption, [SAML SP Setup](saml-sp-setup.md) for
> service provider configuration.

---

## Table of Contents

- [SAML Roles](#saml-roles)
- [SP Metadata Configuration](#sp-metadata-configuration)
- [IdP Setup](#idp-setup)
- [NameID Formats](#nameid-formats)
- [Attribute Mapping](#attribute-mapping)
- [Signing Certificates](#signing-certificates)
- [Logout Flows](#logout-flows)
- [Troubleshooting](#troubleshooting)

---

## SAML Roles

```
┌──────────────┐                    ┌──────────────┐
│  Service     │  1. AuthnRequest   │  Identity    │
│  Provider    │◄──────────────────►│  Provider    │
│  (GGID SP)   │  2. SAML Response  │  (IdP)       │
│              │◄──────────────────►│              │
│  Your App    │  3. Assertion      │  Okta/Azure  │
│              │                    │  AD FS/OneLogin│
└──────────────┘                    └──────────────┘
```

GGID acts as the **Service Provider (SP)**. External systems (Okta, Azure
AD, AD FS) are **Identity Providers (IdP)**.

---

## SP Metadata Configuration

### Generate SP Metadata

```bash
curl https://iam.example.com/api/v1/saml/metadata
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://iam.example.com/saml/metadata">
  <SPSSODescriptor
      protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">

    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>MIIDQjCCAiqgAwIBA...</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>

    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>

    <AssertionConsumerService
        index="0"
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
        Location="https://iam.example.com/saml/acs" />

    <SingleLogoutService
        Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
        Location="https://iam.example.com/saml/sls" />
  </SPSSODescriptor>
</EntityDescriptor>
```

### Register SP in IdP

Upload the SP metadata XML to your IdP (Okta/Azure AD/AD FS). The IdP needs:
- **Entity ID**: `https://iam.example.com/saml/metadata`
- **ACS URL**: `https://iam.example.com/saml/acs`
- **SLS URL**: `https://iam.example.com/saml/sls`
- **Signing Certificate**: from SP metadata

---

## IdP Setup

### Import IdP Metadata (URL)

```bash
curl -X POST https://iam.example.com/api/v1/admin/saml/idp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "metadata_url": "https://dev-12345.okta.com/app/exkabc/federationmetadata",
    "name": "Okta Production"
  }'
```

### Import IdP Metadata (File Upload)

```bash
curl -X POST https://iam.example.com/api/v1/admin/saml/idp \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -F "metadata=@idp-metadata.xml" \
  -F "name=Azure AD"
```

### Manual IdP Configuration

```bash
curl -X POST https://iam.example.com/api/v1/admin/saml/idp \
  -d '{
    "entity_id": "https://sts.windows.net/tenant-uuid/",
    "sso_url": "https://login.microsoftonline.com/tenant-uuid/saml2",
    "slo_url": "https://login.microsoftonline.com/tenant-uuid/saml2",
    "signing_cert": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
    "name": "Azure AD",
    "name_id_format": "emailAddress"
  }'
```

---

## NameID Formats

| Format | URN | Use When |
|--------|-----|----------|
| Email | `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress` | Email is unique identifier |
| Unspecified | `urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified` | Let IdP decide |
| Persistent | `urn:oasis:names:tc:SAML:2.0:nameid-format:persistent` | Privacy (opaque ID) |
| Transient | `urn:oasis:names:tc:SAML:2.0:nameid-format:transient` | One-time use |
| X509 | `urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName` | Certificate-based |
| Windows | `urn:oasis:names:tc:SAML:1.1:nameid-format:WindowsDomainQualifiedName` | AD domain\\user |
| Kerberos | `urn:oasis:names:tc:SAML:2.0:nameid-format:kerberos` | Kerberos principal |
| Entity | `urn:oasis:names:tc:SAML:2.0:nameid-format:entity` | Entity identifier |

### Configure NameID Format

```bash
curl -X PATCH .../admin/saml/idp/{id} \
  -d '{ "name_id_format": "persistent" }'
```

---

## Attribute Mapping

Map IdP assertion attributes to GGID user fields:

```bash
curl -X PATCH .../admin/saml/idp/{id} \
  -d '{
    "attribute_mapping": {
      "email": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
      "first_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
      "last_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
      "display_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
      "groups": "http://schemas.microsoft.com/ws/2008/06/identity/claims/groups",
      "department": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/department"
    }
  }'
```

### Group-to-Role Mapping

```bash
curl -X PATCH .../admin/saml/idp/{id} \
  -d '{
    "group_mapping": {
      "Azure Admin": "admin",
      "Azure Developer": "developer",
      "Azure Viewer": "viewer"
    }
  }'
```

When a SAML assertion includes the group claim `Azure Admin`, GGID
auto-assigns the `admin` role.

---

## Signing Certificates

### SP Certificate (GGID-side)

GGID signs AuthnRequests with its SP certificate:

```yaml
saml:
  sp:
    entity_id: "https://iam.example.com/saml/metadata"
    cert: "/etc/ggid/saml/sp-cert.pem"
    private_key: "/etc/ggid/saml/sp-key.pem"
    sign_authn_request: true
```

### IdP Certificate Rotation

When the IdP rotates its signing certificate:

```bash
# 1. Add new cert (dual-key period)
curl -X POST .../admin/saml/idp/{id}/certificates \
  -F "cert=@new-idp-cert.pem" \
  -F "status=active"

# 2. Keep old cert for overlap
# 3. After all assertions use new cert, remove old:
curl -X DELETE .../admin/saml/idp/{id}/certificates/{old-cert-id}
```

---

## Logout Flows

### SP-Initiated Logout

```
User → GGID (SP) → IdP → All SPs
```

1. User clicks "Logout" in GGID
2. GGID sends `LogoutRequest` to IdP
3. IdP terminates session, sends `LogoutRequest` to all SPs
4. Each SP clears its session
5. IdP sends `LogoutResponse` back to GGID

### IdP-Initiated Logout

```
IdP → GGID (SP) → Clear session
```

1. User logs out from IdP portal
2. IdP sends `LogoutRequest` to GGID's SLS endpoint
3. GGID clears session
4. GGID sends `LogoutResponse` to IdP

### Configuration

```yaml
saml:
  sp:
    slo_enabled: true
    slo_url: "https://iam.example.com/saml/sls"
    slo_binding: "HTTP-Redirect"   # or HTTP-POST
```

---

## Troubleshooting

### Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `InvalidSignature` | Signing cert mismatch | Re-import IdP metadata |
| `InvalidIssuer` | Entity ID mismatch | Verify entityID in config |
| `AudienceRestriction` | SP entity ID not in audience | Add SP entityID to allowed audiences |
| `ReplayDetected` | Same assertion submitted twice | Check clock skew; don't replay |
| `NotOnOrAfter` | Response expired | Check clock skew between SP and IdP |
| `NameIDNotFound` | NameID format not configured | Set correct name_id_format |
| `AttributeNotMapped` | Assertion attribute has no mapping | Add to attribute_mapping |

### Clock Skew

SAML assertions have time windows (`NotBefore` / `NotOnOrAfter`). Clock
skew between SP and IdP causes failures.

```yaml
saml:
  clock_skew_tolerance: 60  # seconds (default: 60)
```

### Debug Mode

```bash
# Enable SAML debug logging
curl -X PATCH .../admin/saml/idp/{id} \
  -d '{ "debug": true }'

# View SAML traces
docker logs ggid-auth 2>&1 | grep -i saml
```

### Capturing SAML Response

Use browser DevTools or SAML tracer extension:

1. Install [SAML Tracer](https://addons.mozilla.org/en-US/firefox/addon/saml-tracer/)
2. Navigate to login page
3. Click the SAML Tracer icon
4. Look for `SAMLResponse` POST to ACS URL
5. Copy the base64-encoded response, decode:

```bash
echo 'PHNhbWxwOlJlc3BvbnNl...' | base64 -d | xmllint --format -
```

### Certificate Fingerprint Verification

```bash
# Get IdP cert fingerprint
openssl x509 -in idp-cert.pem -fingerprint -sha256 -noout
# SHA256 Fingerprint=AB:CD:EF:...

# Compare with fingerprint in GGID config
curl .../admin/saml/idp/{id} | jq .signing_cert_fingerprint
```
