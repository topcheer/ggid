# AWS IAM Identity Center Federation with GGID

Integrate GGID as a SAML 2.0 Identity Provider for AWS IAM Identity Center (formerly AWS SSO), enabling single sign-on (SSO) for all AWS accounts in your organization.

## Architecture

```
┌──────────┐     SAML Assertion     ┌──────────────────┐
│  GGID    │ ─────────────────────> │   AWS IAM        │
│  IdP     │                        │ Identity Center  │
│          │ <───────────────────── │                  │
│  SAML    │     SSO Redirect       │  AWS Console     │
│ Metadata │                        │  CLI (sso login) │
└──────────┘                        └──────────────────┘
```

**Flow**: User accesses AWS Console -> redirected to GGID login -> GGID issues SAML assertion -> AWS grants console access.

## Prerequisites

- GGID deployed and accessible via HTTPS
- AWS Organizations account with admin access
- GGID SAML IdP configured (see Identity > Federation > Entities)

## Step 1: Export GGID SAML IdP Metadata

GGID exposes SAML IdP metadata at:

```
https://<your-ggid-domain>/api/v1/identity/saml/metadata
```

Download the metadata XML:

```bash
curl -o ggid-idp-metadata.xml \
  https://auth.yourcompany.com/api/v1/identity/saml/metadata
```

Verify the metadata contains:
- `<EntityDescriptor>` with GGID's entity ID
- `<IDPSSODescriptor>` with signing certificate
- `<SingleSignOnService>` binding `HTTP-Redirect` and `HTTP-POST`

## Step 2: Create SAML Provider in AWS

1. Log into **AWS Console** -> **IAM Identity Center**
2. Under **Settings** -> **Identity source** -> click **Change**
3. Select **External identity provider**
4. Upload the GGID metadata XML (`ggid-idp-metadata.xml`)
5. Note the **AWS SSO SAML metadata** download URL -- you'll need this for GGID's SP config

## Step 3: Configure Attribute Mapping in AWS

In IAM Identity Center -> **Attribute mapping**, map GGID SAML attributes:

| AWS Attribute | GGID SAML Claim | Format |
|---------------|----------------|--------|
| `Subject` | `${user:email}` | `urn:oasis:names:tc:SAML:2.0:nameid-format:email` |
| `Email` | `${user:email}` | basic |
| `FirstName` | `${user:first_name}` | basic |
| `LastName` | `${user:last_name}` | basic |
| `Groups` | `${user:groups}` | basic |

## Step 4: Configure SAML Attribute Mapping in GGID

In GGID console -> **Identity** -> **SAML Attribute Mapping**:

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

## Step 5: Configure Permission Sets in AWS

1. In IAM Identity Center -> **Permission sets** -> Create
2. Create permission sets mapping to AWS roles (e.g., `AdministratorAccess`, `ReadOnlyAccess`)
3. Assign permission sets to users/groups that match GGID's SAML `Groups` attribute

## Step 6: Test SSO

**Browser test**:
1. Go to AWS SSO start URL: `https://<aws-sso-portal>.awsapps.com/start`
2. You should be redirected to GGID login
3. After authentication, redirected back to AWS with available accounts

**CLI test**:
```bash
aws sso login --profile ggid-aws
# Opens browser -> GGID login -> AWS CLI receives temporary credentials
aws s3 ls --profile ggid-aws
```

## Verification

```bash
# Verify SAML metadata is accessible
curl -s https://auth.yourcompany.com/api/v1/identity/saml/metadata | grep EntityDescriptor

# Verify attribute mapping is configured
curl -s https://auth.yourcompany.com/api/v1/identity/saml/attribute-mapping \
  -H "Authorization: Bearer $TOKEN" | jq .mappings
```

## Troubleshooting

### SAML response not accepted by AWS

- **Cause**: Clock skew between GGID and AWS, or certificate mismatch
- **Fix**: Ensure NTP is synced on GGID server; verify signing cert in metadata matches GGID certificate

### Groups not mapping

- **Cause**: AWS expects `Groups` attribute as multi-value; GGID sends as comma-separated
- **Fix**: Configure GGID SAML attribute mapping to send groups as `<AttributeValue>` entries

### "Invalid Issuer" error

- **Cause**: Entity ID in GGID metadata doesn't match what AWS expects
- **Fix**: Compare `entityID` in GGID metadata with AWS IAM Identity Center configuration

### Certificate rotation

- When GGID rotates SAML signing certificate, download new metadata and re-upload in AWS
- GGID exposes certificate info at `/api/v1/auth/certificates`
