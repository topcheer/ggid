# Tutorial: SAML SP Configuration

> Step-by-step SAML 2.0 Service Provider configuration: generate SP metadata, configure IdP (Okta/ADFS), import IdP metadata, test SSO.

---

## Overview

In SAML 2.0, GGID acts as the **Service Provider (SP)** and trusts an external **Identity Provider (IdP)** like Okta, Azure AD, or ADFS.

```
User → GGID (SP) → Redirect to IdP → User authenticates → IdP sends assertion → GGID validates → Session created
```

---

## Step 1: Generate SP Metadata

GGID generates SP metadata for the IdP administrator:

```bash
curl http://localhost:8080/api/v1/saml/metadata \
  -H "X-Tenant-ID: $TENANT_ID" \
  -o sp_metadata.xml
```

### SP Metadata Format

```xml
<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
  entityID="https://ggid.example.com">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>

    <AssertionConsumerService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://ggid.example.com/api/v1/saml/acs"
      index="0" />

    <SingleLogoutService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://ggid.example.com/api/v1/saml/slo" />

    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <X509Data>
          <X509Certificate>MIID...</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
  </SPSSODescriptor>
</EntityDescriptor>
```

---

## Step 2: Configure IdP (Okta Example)

### In Okta Admin Console

1. Go to **Applications** → **Create App Integration**
2. Select **SAML 2.0**
3. General Settings:
   - App name: `GGID`
   - App logo: (optional)
4. SAML Settings:
   - **Single sign on URL**: `https://ggid.example.com/api/v1/saml/acs`
   - **Audience URI (SP Entity ID)**: `https://ggid.example.com`
   - **Name ID format**: `EmailAddress`
   - **Application username**: `Email`
5. Attribute Statements (optional):
   - `firstName` = `user.firstName`
   - `lastName` = `user.lastName`
   - `department` = `user.department`
6. Click **Next** → **Finish**

### Download IdP Metadata

In Okta: **Sign On** tab → **Identity Provider metadata** → right-click → **Save Link As...**

Save as `idp_metadata.xml`.

---

## Step 3: Configure IdP (ADFS Example)

### In ADFS Management Console

1. **Add Relying Party Trust** → **Start**
2. Select **Enter data about the relying party manually**
3. Display name: `GGID`
4. Profile: **AD FS profile**
5. Certificate: (skip encryption)
6. URL: Enable, set to `https://ggid.example.com/api/v1/saml/acs`
7. Identifier: `https://ggid.example.com`
8. Issuance Authorization Rules: **Permit all users**
9. Issuance Transform Rules:
   - Send LDAP Attributes as Claims:
     - E-Mail-Addresses → Name ID
     - Given-Name → firstName
     - Surname → lastName
10. Finish

### Export IdP Metadata

ADFS metadata URL: `https://adfs.example.com/FederationMetadata/2007-06/FederationMetadata.xml`

---

## Step 4: Import IdP Metadata into GGID

```bash
curl -X POST http://localhost:8080/api/v1/saml/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "idp_metadata_url": "https://idp.example.com/metadata.xml",
    "sp_entity_id": "https://ggid.example.com",
    "acs_url": "https://ggid.example.com/api/v1/saml/acs",
    "idp_cert": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----"
  }' | jq .
```

### Alternative: Upload Metadata File

```bash
curl -X POST http://localhost:8080/api/v1/saml/config/upload \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "metadata=@idp_metadata.xml" | jq .
```

---

## Step 5: Test SSO Login

### Initiate SSO

```bash
# SP-initiated SSO — redirect user to this URL
echo "Open in browser:"
echo "https://ggid.example.com/api/v1/saml/login?return_to=/dashboard"
```

### Expected Flow

1. Browser redirects to IdP (Okta/ADFS)
2. User authenticates at IdP
3. IdP POSTs SAML assertion to `https://ggid.example.com/api/v1/saml/acs`
4. GGID validates:
   - Signature (using IdP certificate)
   - Time window (NotBefore/NotOnOrAfter)
   - Audience (must match SP entity ID)
   - Recipient (must match ACS URL)
5. GGID creates session, redirects to `/dashboard`

### Verify with curl

```bash
# Check SAML config status
curl http://localhost:8080/api/v1/saml/config \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq .
```

---

## Step 6: Attribute Mapping

Map SAML attributes to GGID user fields:

```bash
curl -X PUT http://localhost:8080/api/v1/saml/attribute-mapping \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name_id": "email",
    "firstName": "first_name",
    "lastName": "last_name",
    "department": "department",
    "groups": "groups"
  }' | jq .
```

---

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| `Invalid signature` | Wrong IdP certificate | Re-import IdP metadata with correct cert |
| `Audience mismatch` | SP entity ID differs | Ensure `sp_entity_id` matches in both SP and IdP |
| `Time skew` | Clock drift between SP and IdP | Ensure NTP is running; tolerance is ±60s |
| `User not provisioned` | Auto-provisioning off | Enable `SAML_AUTO_PROVISION=true` or pre-create user |
| `Redirect loop` | ACS URL not matching | Verify `acs_url` matches IdP configuration |
| `No NameID` | NameID format mismatch | Set IdP to use `emailAddress` format |

---

## SAML Configuration Reference

| Setting | Value |
|---------|-------|
| SP Entity ID | `https://ggid.example.com` |
| ACS URL | `https://ggid.example.com/api/v1/saml/acs` |
| SLO URL | `https://ggid.example.com/api/v1/saml/slo` |
| NameID Format | `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress` |
| Binding | `HTTP-POST` (preferred), `HTTP-Redirect` |
| Signature | RSA-SHA256 |
| Digest | SHA256 |

---

## Summary

1. Generate SP metadata from GGID
2. Create SAML app in IdP (Okta/ADFS) with SP metadata
3. Download IdP metadata
4. Import IdP metadata into GGID
5. Test SSO login flow
6. Configure attribute mapping

See: [SAML Configuration](../saml-configuration.md) for full reference.

---

*Last updated: 2025-07-11*
