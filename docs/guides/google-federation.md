# Google Workspace Federation with GGID

Integrate GGID as a SAML 2.0 Identity Provider for Google Workspace SSO, enabling employees to access Google apps with GGID credentials.

## Architecture

```
┌──────────┐   SAML 2.0 SSO         ┌──────────────────┐
│  GGID    │ ─────────────────────> │  Google Workspace│
│  IdP     │                        │  (Admin Console) │
│          │ <───────────────────── │                  │
│  SAML    │   Google SAML Metadata │  Gmail, Drive,   │
│ Metadata │                        │  Calendar, etc.  │
└──────────┘                        └──────────────────┘
```

**Flow**: User goes to gmail.com -> redirected to GGID SAML login -> GGID issues assertion -> Google grants access.

## Prerequisites

- GGID deployed with HTTPS endpoint
- Google Workspace (formerly G Suite) with Super Admin access
- Domain verification completed in Google

## Step 1: Download Google SP Metadata

1. Log into **Google Admin Console** (admin.google.com)
2. Go to **Security** -> **Set up single sign-on (SSO)**
3. Under **SSO profile for your organization**, download the **Google IDP metadata**
4. Save as `google-sp-metadata.xml`

## Step 2: Register Google as SP in GGID

Register Google Workspace as a SAML Service Provider in GGID:

```bash
curl -X POST "https://auth.yourcompany.com/api/v1/identity/federation/entities" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_id": "google.com",
    "entity_name": "Google Workspace",
    "entity_type": "sp",
    "protocol": "saml",
    "metadata_url": "https://www.google.com/a/<your-domain>/acs",
    "trust_level": "high",
    "trust_direction": "outbound"
  }'
```

## Step 3: Configure SAML Attribute Mapping in GGID

Map GGID user attributes to Google's expected SAML claims:

```bash
curl -X PUT "https://auth.yourcompany.com/api/v1/identity/saml/attribute-mapping" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "mappings": [
      {"saml_attribute": "email", "source": "user.email"},
      {"saml_attribute": "first_name", "source": "user.first_name"},
      {"saml_attribute": "last_name", "source": "user.last_name"},
      {"saml_attribute": "groups", "source": "user.groups"}
    ]
  }'
```

Google expects these attributes:
- `Subject` (NameID): User's primary email (format: `emailAddress`)
- `email`: Primary email
- `first_name`: Given name
- `last_name`: Family name

## Step 4: Configure Google Admin Console SSO

1. In **Google Admin Console** -> **Security** -> **Set up single sign-on**
2. Check **Set up SSO with third-party IdP**
3. Fill in:
   - **Sign-in page URL**: `https://auth.yourcompany.com/api/v1/identity/saml/sso`
   - **Sign-out page URL**: `https://auth.yourcompany.com/login?logged_out=true`
   - **Change password URL**: `https://auth.yourcompany.com/profile/password`
   - **Verification certificate**: Upload GGID SAML signing certificate
4. Check **Use a domain-specific issuer**
5. Save

## Step 5: Upload GGID Certificate to Google

1. Download GGID SAML signing certificate:
```bash
curl -o ggid-saml-cert.pem \
  https://auth.yourcompany.com/api/v1/auth/certificates?type=saml
```
2. In Google Admin Console SSO settings, upload `ggid-saml-cert.pem`

## Step 6: Test SSO

**Browser test**:
1. Open incognito window
2. Go to `https://mail.google.com/a/<your-domain>`
3. Should redirect to GGID login
4. After authentication, redirect to Gmail inbox

**Test with forced SSO**:
```
https://www.google.com/a/<your-domain>/ServiceLogin?continue=https://mail.google.com/mail/u/0/
```

## Verification

```bash
# Verify GGID SAML metadata
curl -s https://auth.yourcompany.com/api/v1/identity/saml/metadata | head -5

# Verify Google SP is registered in GGID
curl -s https://auth.yourcompany.com/api/v1/identity/federation/entities \
  -H "Authorization: Bearer $TOKEN" | jq '.[] | select(.entity_id=="google.com")'

# Verify attribute mapping
curl -s https://auth.yourcompany.com/api/v1/identity/saml/attribute-mapping \
  -H "Authorization: Bearer $TOKEN" | jq .mappings
```

## Troubleshooting

### "This app isn't verified"

- **Cause**: GGID SSO URL not whitelisted in Google
- **Fix**: Verify Sign-in page URL in Google Admin matches GGID SSO endpoint exactly

### SAML loop (redirects between GGID and Google)

- **Cause**: NameID format mismatch or certificate validation failure
- **Fix**: Ensure NameID is `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress`; verify certificate fingerprint matches

### Groups not syncing

- **Cause**: Google expects Google Groups, not SAML groups
- **Fix**: Use Google Directory Sync (GADS) with GGID SCIM endpoint `/api/v1/identity/scim/v2/Groups` for group provisioning

### SSO works for some users but not others

- **Cause**: Users with `admin` flag bypass SSO in Google
- **Fix**: In Google Admin Console -> **Security** -> **SSO** -> uncheck **Network masks** or ensure admin accounts are tested separately

### Certificate expiry

- GGID SAML certificates expire. Monitor via:
```bash
curl -s https://auth.yourcompany.com/api/v1/auth/certificates | jq '.[] | select(.type=="SAML") | .expiry'
```
- Re-upload new certificate to Google Admin Console before expiry
