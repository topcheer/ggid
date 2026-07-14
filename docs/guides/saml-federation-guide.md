# SAML Federation Guide

This guide covers configuring SAML 2.0 federation in GGID — Service Provider (SP) and Identity Provider (IdP) initiated SSO, metadata exchange, attribute mapping, and certificate rotation.

> **Related**: [SAML Federation Ecosystem](../research/saml-federation-ecosystem.md)

## Overview

GGID implements SAML 2.0 in `pkg/saml/` with support for both SP-initiated and IdP-initiated SSO flows.

## Roles

```
Service Provider (SP) = GGID (relies on identity assertion)
Identity Provider (IdP) = External system (Okta, Azure AD, ADFS, etc.)

OR

Service Provider (SP) = External application (relies on GGID for identity)
Identity Provider (IdP) = GGID (issues assertions)
```

## SP-Initiated SSO

GGID as Service Provider — user starts at GGID, gets redirected to IdP.

```
User          GGID (SP)           IdP (Okta/Azure)
 │               │                      │
 │── GET /saml/login ─→                 │
 │               │  Generate AuthnRequest
 │               │  Sign with SP private key
 │               │── Redirect ────────→ │
 │←──── Redirect to IdP login ──────────│
 │               │                      │
 │── Authenticate at IdP ─────────────→ │
 │               │                      │
 │←── Redirect with SAML Response ──────│
 │   (signed assertion)                 │
 │               │                      │
 │── POST /saml/acs ──→                 │
 │   {SAMLResponse}                     │
 │               │  1. Verify XML signature
 │               │  2. Validate conditions (NotBefore/NotOnOrAfter)
 │               │  3. Extract attributes
 │               │  4. Auto-provision user (if new)
 │               │  5. Issue GGID JWT
 │←─── 200 {access_token} ─────────────│
```

### Configuration (GGID as SP)

```yaml
saml:
  sp:
    entity_id: "https://ggid.example.com/saml/metadata"
    acs_url: "https://ggid.example.com/saml/acs"
    slo_url: "https://ggid.example.com/saml/slo"
    cert_file: "/keys/saml-sp.crt"      # SP public cert
    key_file: "/keys/saml-sp.key"        # SP private key
  idp:
    entity_id: "https://okta.com/saml2/idp"
    sso_url: "https://company.okta.com/app/company/ggid/sso/saml"
    slo_url: "https://company.okta.com/app/company/ggid/slo/saml"
    cert_file: "/keys/idp-okta.crt"     # IdP public cert
```

### SP Metadata

GGID generates SP metadata for the IdP to consume:

```bash
# Download GGID SP metadata
curl https://ggid.example.com/.well-known/saml-metadata

# Or from the API
curl https://api.ggid.example.com/api/v1/saml/sp-metadata \
  -H "X-Tenant-ID: $TENANT_ID"
```

```xml
<EntityDescriptor entityID="https://ggid.example.com/saml/metadata">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo>
        <X509Data>
          <X509Certificate>MIID...</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    <AssertionConsumerService
      Location="https://ggid.example.com/saml/acs"
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
      index="0" isDefault="true"/>
    <SingleLogoutService
      Location="https://ggid.example.com/saml/slo"
      Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

## IdP-Initiated SSO

User starts at the IdP, which sends an unsolicited SAML Response to GGID.

```
User          IdP (Okta)          GGID (SP)
 │               │                     │
 │── Click GGID app in Okta ──→        │
 │               │                     │
 │               │  Generate unsolicited
 │               │  SAML Response
 │←── Redirect to GGID ACS ────────────│
 │   {SAMLResponse}                    │
 │               │                     │
 │── POST /saml/acs ──────────────────→│
 │               │                     │
 │               │  1. Verify signature
 │               │  2. Validate conditions
 │               │  3. Check InResponseTo (if present)
 │               │  4. Extract attributes
 │               │  5. Issue JWT
 │←─── 200 {access_token} ─────────────│
```

GGID handles unsolicited responses in `pkg/saml/idp_initiated.go`:

```go
type IdPInitiatedSSORequest struct {
    XMLName     xml.Name `xml:"Response"`
    Destination string   `xml:"Destination,attr"`
    Issuer      string   `xml:"Issuer>value"`
    Assertion   *SAMLAssertion
}
```

### Security: IdP-Initiated Risk

IdP-initiated SSO is vulnerable to:
- **Login CSRF**: Attacker injects their SAML response into victim's browser
- **Replay**: Replaying a captured SAML response

**Mitigations**: GGID validates `NotOnOrAfter` conditions and checks response freshness.

## Attribute Mapping

Map IdP attributes to GGID user fields:

| SAML Attribute | GGID Field | Example |
|----------------|------------|---------|
| `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress` | email | alice@example.com |
| `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name` | display_name | Alice Chen |
| `http://schemas.xmlsoap.org/claims/Group` | groups[] | Engineering |
| `http://schemas.microsoft.com/ws/2008/06/identity/claims/role` | roles[] | admin |

```yaml
saml:
  attribute_mapping:
    email: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
    name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
    groups: "http://schemas.xmlsoap.org/claims/Group"
```

## Certificate Rotation

### GGID SP Certificate Rotation

1. **Generate new key pair**:
```bash
openssl req -x509 -newkey rsa:2048 -keyout saml-sp-new.key \
  -out saml-sp-new.crt -days 365 -nodes
```

2. **Update IdP**: Upload new SP certificate to IdP (Okta/Azure AD). Keep old cert during transition.

3. **Deploy new GGID cert**: Replace key files, restart service. Both old and new certs valid during grace period.

4. **Verify**: Test SAML login. Check logs for signature verification success.

5. **Remove old cert from IdP**: After grace period (24-48h), remove old certificate.

### IdP Certificate Rotation

When the IdP (Okta/Azure) rotates their signing certificate:

1. **Download new IdP metadata** (includes new cert)
2. **Add new IdP cert** to GGID alongside old cert (grace period)
3. **Monitor**: SAML logins should work with both certs
4. **Remove old cert** after grace period

> GGID supports multiple trusted IdP certificates simultaneously to enable zero-downtime rotation.

## SAML Security Checklist

- [ ] Assertions signed with RSA-SHA256 (not SHA1)
- [ ] Assertions encrypted (optional but recommended for high-security)
- [ ] `NotBefore` and `NotOnOrAfter` conditions enforced
- [ ] Replay detection (assertion ID tracking)
- [ ] Audience restriction validated (`audience` must match SP entity ID)
- [ ] SP metadata signed
- [ ] Clock skew tolerance configured (default: 60 seconds)
- [ ] Certificate chain validated to trusted root

## Troubleshooting

### Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| "Invalid signature" | Wrong IdP cert in GGID | Re-download IdP metadata |
| "Conditions not met" | Clock skew | Check NTP on both servers |
| "Audience restriction" | Entity ID mismatch | Verify SP entity ID matches |
| "Response expired" | `NotOnOrAfter` passed | Check clock sync |
| "No NameID" | Missing NameID format | Configure IdP NameID policy |

### Debug SAML Response

```bash
# Decode SAML response from base64
echo "PHNhbWxwOlJlc3BvbnNl..." | base64 -d | xmllint --format -

# Check signature
openssl dgst -sha256 -verify idp-pubkey.pem -signature sig.bin response.xml
```

## See Also

- [Per-Tenant IdP](per-tenant-idp.md)
- SSO Configuration
- [SAML Federation Ecosystem](../research/saml-federation-ecosystem.md)
- Certificates Management
