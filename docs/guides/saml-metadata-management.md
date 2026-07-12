# SAML Metadata Management

SP metadata generation, IdP metadata import, refresh schedule, signature on metadata, entity categories, and federation metadata aggregation.

## Overview

SAML metadata is the XML document that describes a SAML entity (SP or IdP) — its endpoints, certificates, bindings, and capabilities. Metadata exchange establishes trust between SP and IdP.

## SP Metadata Generation

GGID generates SP metadata at a well-known URL:

```bash
GET /saml/metadata.xml
```

### Metadata Structure

```xml
<EntityDescriptor entityID="https://auth.ggid.dev/saml"
  xmlns="urn:oasis:names:tc:SAML:2.0:metadata">

  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">

    <KeyDescriptor use="signing">
      <KeyInfo><X509Data><X509Certificate>...</X509Certificate></X509Data></KeyInfo>
    </KeyDescriptor>

    <KeyDescriptor use="encryption">
      <KeyInfo><X509Data><X509Certificate>...</X509Certificate></X509Data></KeyInfo>
      <EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes256-gcm"/>
    </KeyDescriptor>

    <NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDFormat>

    <AssertionConsumerService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      Location="https://auth.ggid.dev/saml/acs"
      index="0" isDefault="true"/>

    <SingleLogoutService
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
      Location="https://auth.ggid.dev/saml/slo"/>

  </SPSSODescriptor>
</EntityDescriptor>
```

### Signed SP Metadata

```go
func GenerateSignedMetadata() ([]byte, error) {
    metadata := buildMetadataXML()
    return SignXML(metadata, spPrivateKey, spCert)
}
```

Signed metadata prevents tampering during exchange. GGID signs all SP metadata by default.

## IdP Metadata Import

### URL-Based Import

```bash
POST /api/v1/identity/federation/saml/import
{"metadata_url": "https://idp.partner.com/metadata.xml"}
# → Fetches, parses, validates, stores
```

### File Upload

```bash
POST /api/v1/identity/federation/saml/import-file
# multipart: metadata.xml
```

### Manual Configuration

```bash
POST /api/v1/identity/federation/saml
{
  "entity_id": "https://idp.corp.com",
  "sso_url": "https://idp.corp.com/sso",
  "slo_url": "https://idp.corp.com/slo",
  "signing_cert": "...",
  "want_assertions_signed": true
}
```

## Parsed Metadata Elements

| Element | GGID Usage |
|---------|-----------|
| EntityID | Unique identifier for trust relationship |
| SingleSignOnService URL | Redirect target for AuthnRequest |
| SingleLogoutService URL | SLO request target |
| Signing certificate | Verify IdP-issued assertions |
| Encryption certificate | GGID encrypts requests to IdP |
| NameIDFormat | Expected NameID format |
| Attribute profiles | Expected attribute names |

## Metadata Refresh

```yaml
refresh:
  schedule: "0 */6 * * *"  # Every 6 hours
  strategy: "if_modified_since"
  on_change:
    - update_certificates
    - notify_admin
    - log_diff
    - test_connection
```

### Change Detection

```go
func RefreshMetadata(idpID string) error {
    oldMetadata := store.GetMetadata(idpID)
    
    newMetadata, err := fetchMetadata(idpID)
    if err != nil { return err }
    
    if !metadataChanged(oldMetadata, newMetadata) {
        return nil // No change
    }
    
    // Validate new metadata
    if err := validateMetadata(newMetadata); err != nil {
        alert.Send("metadata_validation_failed", idpID, err)
        return err // Keep old metadata
    }
    
    // Apply new metadata
    store.UpdateMetadata(idpID, newMetadata)
    
    // Diff for audit
    diff := diffMetadata(oldMetadata, newMetadata)
    audit.Log("metadata.updated", map[string]interface{}{
        "idp": idpID,
        "changes": diff,
    })
    
    if diff.CertChanged {
        alert.Send("saml_cert_changed", idpID, diff)
    }
    
    return nil
}
```

## Metadata Signature Verification

```go
func ImportMetadata(xml []byte) (*EntityDescriptor, error) {
    doc := parseXML(xml)
    
    // Check if metadata is signed
    sig := findSignature(doc)
    if sig != nil {
        // Verify signature against trusted CA or known cert
        if err := verifyXMLSignature(doc, sig, trustedCAs); err != nil {
            return nil, ErrMetadataSignatureInvalid
        }
    } else if requireSignedMetadata {
        return nil, ErrMetadataNotSigned
    }
    
    return parseEntityDescriptor(doc)
}
```

| Policy | When to Use |
|--------|-------------|
| Require signed metadata | Production, federation |
| Allow unsigned (manual verify) | Testing, well-known partners |
| Pin certificate fingerprint | High-security bilateral trust |

## Entity Categories

Entity categories declare compliance with specific federation profiles:

```xml
<md:EntityAttributes>
  <saml:Attribute Name="http://macedir.org/entity-category">
    <saml:AttributeValue>http://refeds.org/category/research-and-scholarship</saml:AttributeValue>
    <saml:AttributeValue>http://www.geant.net/uri/dataprotection-code-of-conduct/v1</saml:AttributeValue>
  </saml:Attribute>
</md:EntityAttributes>
```

| Category | Meaning |
|----------|---------|
| Research & Scholarship | eduGAIN R&S profile |
| Data Protection CoC | GDPR compliance attestation |
| Sirtfi | Security incident response trust framework |

GGID respects entity categories for automatic attribute release decisions.

## Federation Metadata Aggregation

For multi-party federations (eduGAIN, InCommon):

```bash
# Import aggregated federation metadata
POST /api/v1/identity/federation/saml/import-aggregate
{
  "aggregate_url": "https://metadata.federation.org/aggregate.xml",
  "trust_filter": "research-and-scholarship",  # Only accept R&S entities
  "refresh_interval_hours": 6
}
```

### Aggregation Flow

```
Federation Authority publishes aggregate metadata (1000s of entities)
  ↓
GGID downloads aggregate
  ↓
Applies trust filter (entity categories, SP entity allowlist)
  ↓
Stores only approved entities
  ↓
Refreshes on schedule
```

## Certificate Rotation in Metadata

When IdP rotates signing certificate:

```
1. New metadata published with BOTH old + new certificates
2. GGID refreshes → accepts assertions signed by either
3. After grace period (2 weeks), old certificate removed from metadata
4. GGID refreshes → accepts only new certificate
```

GGID logs certificate changes and alerts admins.

## Monitoring

| Metric | Alert |
|--------|-------|
| Metadata refresh failure | Any → stale config risk |
| Certificate expiry | <30 days → notify |
| Certificate changed | Any → review + test |
| Import validation failure | Any → bad metadata |
| Entity count change (aggregate) | >10% → investigate |

## See Also

- [SAML SP Implementation](saml-sp-implementation.md)
- [Identity Federation Patterns](identity-federation-patterns.md)
- [Identity Federation Architecture](identity-federation-architecture.md)
- [Identity Provider Configuration](identity-provider-configuration.md)
