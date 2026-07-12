# SAML Troubleshooting Guide

This guide helps diagnose and resolve common SAML 2.0 integration issues in GGID.

## Common Issues

### 1. Certificate Mismatch

**Symptom**: `Signature validation failed` or `Invalid signature`

**Cause**: The IdP's signing certificate doesn't match the one in GGID's metadata configuration.

**Diagnosis**:
```bash
# Extract certificate from IdP metadata
xmlstarlet sel -t -v "//*[local-name()='X509Certificate']" idp-metadata.xml | base64 -d | openssl x509 -fingerprint -sha256 -noout

# Check GGID's configured certificate
openssl x509 -fingerprint -sha256 -noout -in /etc/ggid/saml/idp-cert.pem
```

**Fix**: Update the IdP certificate in GGID configuration. IdP certificates rotate periodically — set up monitoring.

### 2. Clock Skew

**Symptom**: `Assertion expired` or `NotBefore condition not met`

**Cause**: Server clocks between IdP and GGID differ by more than the allowed skew.

**Default skew**: 60 seconds

```yaml
saml:
  clock_skew: 120s  # Increase if needed
```

**Fix**:
- Ensure NTP is running on all servers
- Increase `clock_skew` temporarily if NTP is unavailable
- Check container timezones (use UTC)

### 3. NameID Format Mismatch

**Symptom**: `Unable to map NameID to user` or empty username after SAML login

**Cause**: IdP sends a NameID format GGID doesn't expect.

| Format | URN |
|---|---|
| Unspecified | `urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified` |
| Email | `urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress` |
| X509 | `urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName` |
| Persistent | `urn:oasis:names:tc:SAML:2.0:nameid-format:persistent` |
| Transient | `urn:oasis:names:tc:SAML:2.0:nameid-format:transient` |

```yaml
saml:
  name_id_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
  name_id_attribute: "email"  # Map to GGID user field
```

### 4. Audience Restriction

**Symptom**: `Audience restriction failed`

**Cause**: The SAML assertion's `<Audience>` doesn't match GGID's entity ID.

**Diagnosis**:
```xml
<!-- In assertion -->
<saml:AudienceRestriction>
  <saml:Audience>https://auth.ggid.example.com/saml/metadata</saml:Audience>
</saml:AudienceRestriction>
```

**Fix**: Ensure GGID's entity ID matches the IdP's configured audience:

```yaml
saml:
  entity_id: "https://auth.ggid.example.com/saml/metadata"
```

### 5. Destination URL Mismatch

**Symptom**: `Destination does not match expected URL`

**Cause**: The `Destination` attribute in the SAML response doesn't match GGID's ACS URL.

**Fix**: Configure the IdP to send responses to the correct ACS URL:

```
ACS URL: https://auth.ggid.example.com/saml/acs
```

## SP Metadata Validation

### Validate SP Metadata

```bash
# Download GGID SP metadata
curl https://auth.ggid.example.com/saml/metadata > sp-metadata.xml

# Validate XML structure
xmllint --noout sp-metadata.xml

# Check key elements
xmlstarlet sel -t -v "//*[local-name()='EntityID']" sp-metadata.xml
xmlstarlet sel -t -v "//*[local-name()='AssertionConsumerService']/@Location" sp-metadata.xml
```

### Common SP Metadata Issues

| Issue | Fix |
|---|---|
| Wrong entity ID | Update `saml.entity_id` in config |
| Wrong ACS URL | Update `saml.acs_url` in config |
| Missing certificate | Generate and export SP signing cert |
| Wrong bindings | Ensure HTTP-POST binding for ACS |

## IdP Metadata Refresh

GGID automatically refreshes IdP metadata:

```yaml
saml:
  idp_metadata_url: "https://idp.example.com/metadata"
  metadata_refresh_interval: 24h
```

**Manual refresh**:
```bash
curl -X POST http://localhost:9005/admin/saml/refresh-metadata
```

**Symptoms of stale metadata**:
- Certificate mismatch after IdP rotation
- New IdP endpoints not recognized
- Intermittent authentication failures

## Signed vs Encrypted Assertions

### Signed Assertions

GGID requires signed assertions by default:

```yaml
saml:
  want_signed_assertions: true
```

Verification steps:
1. Extract signature from `<ds:Signature>` element
2. Verify signature using IdP's public certificate
3. Check signature covers the assertion (references)

### Encrypted Assertions

For encrypted assertions, GGID uses its SP decryption key:

```yaml
saml:
  want_encrypted_assertions: false  # Optional
  sp_decrypt_key: /etc/ggid/saml/sp-key.pem
  sp_decrypt_cert: /etc/ggid/saml/sp-cert.pem
```

**Common encryption issue**: `Failed to decrypt assertion`
- Ensure SP decryption key matches the one in SP metadata
- Check key algorithm compatibility (RSA-OAEP, AES-CBC)

## SLO (Single Logout) Debugging

### SP-Initiated SLO

```
1. User clicks logout in GGID
2. GGID sends LogoutRequest to IdP SLO endpoint
3. IdP validates request, terminates session
4. IdP sends LogoutResponse to GGID SLO endpoint
5. GGID clears local session
```

### Common SLO Issues

| Issue | Cause | Fix |
|---|---|---|
| LogoutResponse signature invalid | IdP uses different signing cert for SLO | Check SLO signing cert |
| User not logged out from IdP | IdP doesn't support SLO | Use front-channel logout |
| Redirect loop | ACS and SLO URLs confused | Verify endpoint URLs |
| `Destination` mismatch | Wrong SLO endpoint URL | Update `saml.slo_url` |

## XML Signature Verification

### Verification Steps

1. Parse the SAML response XML
2. Locate `<ds:Signature>` element
3. Extract signature value and signing info
4. Canonicalize the signed element (C14N)
5. Verify signature using IdP certificate

```go
func verifySignedAssertion(response *saml.Response, cert *x509.Certificate) error {
    sig := response.Signature
    if sig == nil {
        return ErrMissingSignature
    }
    // Canonicalize signed element
    signedBytes, err := canonicalize(response.Assertion)
    // Verify signature
    return cert.CheckSignature(sig.Algorithm, signedBytes, sig.Value)
}
```

### Common Signature Issues

- **Wrapping attacks**: Signature covers a different element than expected. Always verify the signature references the assertion ID.
- **Canonicalization mismatch**: Use exclusive C14N (`http://www.w3.org/2001/10/xml-exc-c14n#`) as specified in SAML 2.0.
- **Algorithm downgrade**: Reject SHA-1 signatures in favor of SHA-256+.

## Browser Redirect Limits

SAML responses are sent via browser redirects, which have URL length limits (~2048 bytes for older browsers). For large responses:

- Use HTTP-POST binding (no size limit) instead of HTTP-Redirect
- Configure GGID to prefer POST binding:

```yaml
saml:
  preferred_binding: "POST"  # or "Redirect"
```

## Session Lifetime Mismatch

**Issue**: IdP session lasts longer than GGID session, causing unexpected re-authentication.

**Config**:
```yaml
saml:
  session_lifetime: 8h        # GGID session
  idp_session_lifetime: 8h    # Should match IdP
```

## Debugging Tools

### SAML Tracer (Browser Extension)

Firefox/Chrome extension that captures SAML messages:
1. Install SAML Tracer
2. Initiate SAML login
3. View captured requests/responses
4. Inspect raw XML, certificates, and attributes

### openssl Commands

```bash
# Inspect SAML response
echo "<saml-response-base64>" | base64 -d | xmllint --format -

# Verify certificate
openssl x509 -in idp-cert.pem -text -noout

# Check certificate dates
openssl x509 -in idp-cert.pem -dates -noout
```

### GGID Debug Logging

```yaml
log:
  level: debug
  saml: true
```

```
DEBUG saml: received response from IdP https://idp.example.com
DEBUG saml: assertion ID=_abc123, issuer=https://idp.example.com
DEBUG saml: verifying signature with cert CN=idp-signing,OU=IdP
DEBUG saml: signature verified, extracting attributes
DEBUG saml: NameID format=emailAddress, value=user@example.com
DEBUG saml: mapped 5 attributes, creating session
```

## Quick Diagnostic Checklist

- [ ] NTP synchronized on both servers
- [ ] IdP certificate valid and not expired
- [ ] Entity ID matches between SP and IdP
- [ ] ACS URL reachable from IdP
- [ ] SP metadata exported and imported in IdP
- [ ] Firewall allows HTTPS between IdP and SP
- [ ] Clock skew within tolerance
- [ ] NameID format configured correctly
- [ ] Attribute mappings defined
- [ ] SLO endpoint configured (if needed)