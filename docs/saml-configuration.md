# SAML Configuration Reference

Deep-dive configuration guide for SAML 2.0 in GGID. Covers IdP metadata import,
SP metadata generation, NameID format selection, attribute mapping, certificate
rotation, and HTTP-POST vs HTTP-Redirect binding selection.

> For SP setup tutorials (Grafana, Jenkins, Tableau), see
> [SAML Service Provider Setup](saml-sp-setup.md). For GGID-as-SP integration
> with external IdPs, see [SAML Integration Guide](saml-integration.md).

---

## Table of Contents

- [Configuration Model](#configuration-model)
- [IdP Metadata Import](#idp-metadata-import)
- [SP Metadata Generation](#sp-metadata-generation)
- [NameID Format](#nameid-format)
- [Attribute Mapping](#attribute-mapping)
- [Certificate Rotation](#certificate-rotation)
- [Binding Selection](#binding-selection)
- [Signature and Encryption](#signature-and-encryption)
- [Session Configuration](#session-configuration)
- [Troubleshooting](#troubleshooting)

---

## Configuration Model

GGID supports SAML in both directions:

| Mode | Role | Direction |
|------|------|-----------|
| **IdP mode** | GGID is the Identity Provider | Applications (SPs) delegate auth to GGID |
| **SP mode** | GGID is the Service Provider | GGID delegates auth to external IdPs (Okta, Azure AD) |

### Configuration File

```yaml
saml:
  # IdP mode: GGID as Identity Provider
  idp:
    enabled: true
    entity_id: "https://iam.example.com/saml/metadata"
    sign_response: true
    sign_assertion: true
    encrypt_assertion: false
    want_authn_request_signed: true
    default_nameid_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
    default_binding: "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
    cert: "/etc/ggid/saml/idp-cert.pem"
    key: "/etc/ggid/saml/idp-key.pem"
    metadata_validity: "72h"

  # SP mode: GGID as Service Provider
  sp:
    enabled: true
    entity_id: "https://iam.example.com/saml/sp"
    acs_url: "https://iam.example.com/saml/acs"
    want_assertion_signed: true
    want_response_signed: true
    want_assertion_encrypted: false
    nameid_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
    metadata_url: "https://iam.example.com/saml/sp/metadata"
```

---

## IdP Metadata Import

When GGID operates as a Service Provider, it needs the IdP's metadata to know
where to send AuthnRequests and how to verify assertions.

### Import Methods

#### 1. URL Import (Auto-Refresh)

```bash
# Via API
curl -X POST https://iam.example.com/api/v1/saml/idps \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_id": "http://www.okta.com/exk1a2b3c4d5",
    "metadata_url": "https://example.okta.com/app/exk1a2b3c4d5/sso/saml/metadata",
    "refresh_interval": "24h",
    "name": "Corporate Okta"
  }'
```

GGID periodically fetches and re-parses the metadata. If the cert changes, it
is picked up automatically.

#### 2. File Import

```bash
curl -X POST https://iam.example.com/api/v1/saml/idps \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: multipart/form-data" \
  -F "name=Azure AD" \
  -F "entity_id=https://sts.windows.net/tenant-id/" \
  -F "metadata=@azure-ad-metadata.xml"
```

#### 3. Manual Configuration

For IdPs that don't publish metadata:

```bash
curl -X POST https://iam.example.com/api/v1/saml/idps \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ADFS",
    "entity_id": "https://adfs.example.com/adfs/services/trust",
    "sso_url": "https://adfs.example.com/adfs/ls/",
    "slo_url": "https://adfs.example.com/adfs/ls/?wa=wsignout1.0",
    "signing_cert": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
    "encryption_cert": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
    "nameid_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
  }'
```

### Metadata Structure

GGID expects standard SAML 2.0 metadata XML:

```xml
<EntityDescriptor entityID="https://idp.example.com">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo>
        <X509Data>
          <X509Certificate>MIID...</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://idp.example.com/sso" />
    <SingleSignOnService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://idp.example.com/sso" />
    <SingleLogoutService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://idp.example.com/slo" />
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
  </IDPSSODescriptor>
</EntityDescriptor>
```

### Validation Checks on Import

| Check | Action on Failure |
|-------|-------------------|
| XML well-formed | Reject with 400 |
| EntityID present | Reject with 400 |
| At least one SSO service | Reject with 400 |
| Signing certificate present | Reject with 400 |
| Certificate not expired | Warn but accept |
| Certificate expiry < 30 days | Warn admin |

---

## SP Metadata Generation

When GGID operates as an Identity Provider, SPs need GGID's metadata to know
where to send assertions.

### Generated Metadata

```
GET https://iam.example.com/saml/metadata
Content-Type: application/xml
```

```xml
<EntityDescriptor entityID="https://iam.example.com/saml/metadata">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIID...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <SingleSignOnService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://iam.example.com/saml/sso" />
    <SingleSignOnService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://iam.example.com/saml/sso" />
    <SingleLogoutService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://iam.example.com/saml/slo" />
  </IDPSSODescriptor>
</EntityDescriptor>
```

### SP Registration

Register each SP in GGID:

```bash
curl -X POST https://iam.example.com/api/v1/saml/sps \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_id": "https://grafana.example.com/saml/metadata",
    "acs_url": "https://grafana.example.com/saml/acs",
    "acs_binding": "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
    "slo_url": "https://grafana.example.com/saml/sls",
    "name": "Grafana",
    "nameid_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    "attributes": {
      "email": "$user.email",
      "display_name": "$user.display_name",
      "groups": "$user.groups"
    }
  }'
```

---

## NameID Format

The NameID identifies the subject of a SAML assertion. GGID supports all
standard formats:

| Format URI | When to Use |
|------------|-------------|
| `emailAddress` | Default. User email as identifier. |
| `unspecified` | Custom identifier. |
| `persistent` | Stable opaque ID per SP. Privacy-preserving. |
| `transient` | Random one-time ID. Per-session only. |
| `entity` | Identifies an entity, not a user. |
| `X509SubjectName` | X.509 certificate subject. |
| `WindowsDomainQualifiedName` | `DOMAIN\username` format. |
| `kerberos` | Kerberos principal name. |

### Persistent NameID

For privacy, use persistent NameID to give each SP a different opaque
identifier:

```yaml
saml:
  idp:
    default_nameid_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:persistent"
```

```sql
-- GGID generates a per-SP persistent ID
SELECT saml_generate_persistent_id(user_id, sp_entity_id);
-- Result: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
```

### Per-SP Override

```bash
curl -X PATCH https://iam.example.com/api/v1/saml/sps/{id} \
  -d '{ "nameid_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:persistent" }'
```

---

## Attribute Mapping

GGID maps user profile fields to SAML attributes in the assertion.

### Default Mapping

| SAML Attribute | Source | Description |
|----------------|--------|-------------|
| `email` | `$user.email` | Primary email |
| `name` | `$user.display_name` | Full display name |
| `given_name` | `$user.first_name` | First name |
| `family_name` | `$user.last_name` | Last name |
| `groups` | `$user.groups` | Group memberships |
| `roles` | `$user.roles` | Assigned roles |

### Custom Mapping

```bash
curl -X PATCH https://iam.example.com/api/v1/saml/sps/{id} \
  -H "Authorization: Bearer <admin-token>" \
  -d '{
    "attributes": {
      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "$user.email",
      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": "$user.display_name",
      "http://schemas.microsoft.com/ws/2008/06/identity/claims/role": "$user.roles",
      "http://schemas.xmlsoap.org/claims/Group": "$user.groups",
      "department": "$user.metadata.department",
      "employee_id": "$user.metadata.employee_id"
    }
  }'
```

### Conditional Mapping

```json
{
  "attributes": {
    "groups": {
      "source": "$user.groups",
      "filter": { "prefix": "grafana-" },
      "transform": "strip_prefix"
    }
  }
}
```

### Attribute Statement Example

```xml
<saml:AttributeStatement>
  <saml:Attribute Name="email"
    NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
    <saml:AttributeValue>jane@example.com</saml:AttributeValue>
  </saml:Attribute>
  <saml:Attribute Name="groups"
    NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
    <saml:AttributeValue>admins</saml:AttributeValue>
    <saml:AttributeValue>developers</saml:AttributeValue>
  </saml:Attribute>
</saml:AttributeStatement>
```

---

## Certificate Rotation

### Signing Certificate Lifecycle

```
Generate new key pair → Add to metadata (dual cert) → Wait for SPs to pick up
→ Switch to signing with new key → Remove old cert after grace period
```

### Step 1: Generate New Key Pair

```bash
openssl req -x509 -newkey rsa:2048 -keyout new-key.pem -out new-cert.pem \
  -days 3650 -nodes -subj "/CN=iam.example.com"
```

### Step 2: Publish Both Certificates (Dual Mode)

```yaml
saml:
  idp:
    signing_certs:
      - cert: "/etc/ggid/saml/idp-cert-old.pem"
        status: "retiring"
      - cert: "/etc/ggid/saml/idp-cert-new.pem"
        status: "active"
```

GGID metadata includes both certificates:

```xml
<KeyDescriptor use="signing">
  <ds:KeyInfo>
    <ds:X509Data><ds:X509Certificate>old-cert-base64</ds:X509Certificate></ds:X509Data>
  </ds:KeyInfo>
</KeyDescriptor>
<KeyDescriptor use="signing">
  <ds:KeyInfo>
    <ds:X509Data><ds:X509Certificate>new-cert-base64</ds:X509Certificate></ds:X509Data>
  </ds:KeyInfo>
</KeyDescriptor>
```

### Step 3: Switch Signing Key

After all SPs have refreshed metadata (wait at least `metadata_validity`):

```yaml
saml:
  idp:
    cert: "/etc/ggid/saml/idp-cert-new.pem"
    key: "/etc/ggid/saml/idp-key-new.pem"
    signing_certs:
      - cert: "/etc/ggid/saml/idp-cert-new.pem"
        status: "active"
```

### Step 4: Remove Old Certificate

After a grace period (recommend 2 weeks):

```yaml
saml:
  idp:
    signing_certs:
      - cert: "/etc/ggid/saml/idp-cert-new.pem"
        status: "active"
```

### Rotation Schedule

| Certificate | Recommended Rotation |
|-------------|---------------------|
| IdP signing | Every 2 years |
| IdP encryption | Every 2 years |
| SP signing | As needed by SP |
| SP encryption | Every 2 years |

---

## Binding Selection

SAML defines two primary bindings for SSO:

| Binding | Request Direction | Response Direction | Characteristics |
|---------|-------------------|--------------------|-----------------|
| HTTP-Redirect | Browser → IdP | Browser → SP (limited) | URL-based, size-limited, signed query string |
| HTTP-POST | Browser → IdP | Browser → SP | Form POST, larger payloads, signed XML |

### Recommendation Matrix

| Scenario | Request Binding | Response Binding |
|----------|----------------|-----------------|
| Standard SSO | HTTP-Redirect | HTTP-POST |
| Large assertions | HTTP-POST | HTTP-POST |
| Encrypted assertions | HTTP-POST | HTTP-POST |
| Minimal round-trips | HTTP-Redirect | HTTP-POST |
| Legacy SP | HTTP-POST | HTTP-POST |

### HTTP-Redirect Signing

In Redirect binding, the message is signed as a query parameter:

```
https://idp.example.com/sso?
  SAMLRequest=base64-deflated-authn-request&
  RelayState=state-token&
  SigAlg=rsa-sha256&
  Signature=base64-signature
```

### HTTP-POST Signing

In POST binding, the message is base64-encoded in a form field, and the XML
itself is signed:

```html
<form method="POST" action="https://sp.example.com/acs">
  <input type="hidden" name="SAMLResponse" value="base64-assertion-xml" />
  <input type="hidden" name="RelayState" value="state-token" />
</form>
```

### Configuring Binding per SP

```bash
curl -X PATCH https://iam.example.com/api/v1/saml/sps/{id} \
  -d '{
    "preferred_request_binding": "HTTP-Redirect",
    "response_binding": "HTTP-POST"
  }'
```

---

## Signature and Encryption

### Response vs Assertion Signing

| Option | Signs | Protection Level |
|--------|-------|------------------|
| Sign Response only | `<samlp:Response>` | Message integrity |
| Sign Assertion only | `<saml:Assertion>` | Assertion integrity |
| Sign Both | Response + Assertion | Maximum integrity |

```yaml
saml:
  idp:
    sign_response: true
    sign_assertion: true   # Recommended for production
```

### Assertion Encryption

For high-security environments, GGID can encrypt the assertion using the SP's
public key:

```xml
<EncryptedAssertion>
  <EncryptedData>
    <EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes256-cbc" />
    <KeyInfo>
      <EncryptedKey>
        <EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p" />
      </EncryptedKey>
    </KeyInfo>
    <!-- encrypted assertion content -->
  </EncryptedData>
</EncryptedAssertion>
```

```yaml
saml:
  idp:
    encrypt_assertion: true
    encryption_algorithm: "aes256-cbc"
    key_transport: "rsa-oaep-mgf1p"
```

### Signature Algorithm

```yaml
saml:
  idp:
    signature_algorithm: "rsa-sha256"
    digest_algorithm: "sha256"
```

| Algorithm | Status |
|-----------|--------|
| `rsa-sha256` | **Recommended** |
| `rsa-sha384` | Acceptable |
| `rsa-sha512` | Acceptable |
| `rsa-sha1` | **Deprecated** — reject in production |

---

## Session Configuration

### SAML Session

```yaml
saml:
  session:
    max_age: "8h"          # Maximum session duration
    lifetime: "4h"         # Default session lifetime
    force_reauth: false    # Require fresh authentication each SSO
```

### Session NotOnOrAfter

GGID sets `SessionNotOnOrAfter` in the assertion to enforce session expiry:

```xml
<AuthnStatement
  AuthnInstant="2024-01-15T10:00:00Z"
  SessionIndex="session-id"
  SessionNotOnOrAfter="2024-01-15T18:00:00Z">
```

### ForceAuthn

Set `ForceAuthn="true"` in the AuthnRequest to require the IdP to
re-authenticate the user, ignoring existing sessions:

```xml
<samlp:AuthnRequest
  ForceAuthn="true"
  IsPassive="false"
  ...>
```

---

## Troubleshooting

### "Signature validation failed"

| Cause | Fix |
|-------|-----|
| Certificate mismatch | Re-import IdP metadata with correct cert |
| Wrong signature algorithm | Check `signature_algorithm` config |
| Assertion signed but response expected | Toggle `sign_response` / `sign_assertion` |

### "Audience restriction failed"

The SP's entity ID is not in the `Audience` element.

```xml
<!-- Correct -->
<AudienceRestriction>
  <Audience>https://grafana.example.com/saml/metadata</Audience>
</AudienceRestriction>
```

**Fix**: Ensure SP `entity_id` matches the expected audience.

### "Clock skew detected"

| Cause | Fix |
|-------|-----|
| Server clocks differ | Configure NTP on all servers |
| Conditions NotBefore/NotOnOrAfter | Increase `clock_skew_tolerance` |

```yaml
saml:
  clock_skew_tolerance: "60s"
```

### "NameID format not supported"

The IdP sent a NameID format GGID doesn't expect.

**Fix**: Update `nameid_format` to match the IdP's format, or add multiple
acceptable formats.

### "Certificate expired"

```bash
# Check certificate expiry
openssl x509 -in idp-cert.pem -noout -dates

notBefore=Jan  1 00:00:00 2022 GMT
notAfter=Jan  1 00:00:00 2024 GMT  ← Expired
```

**Fix**: Request new certificate from IdP administrator, re-import metadata.

### Metadata Refresh Stale

If URL-based metadata import stops refreshing:

```bash
# Force manual refresh
curl -X POST https://iam.example.com/api/v1/saml/idps/{id}/refresh-metadata \
  -H "Authorization: Bearer <admin-token>"
```
