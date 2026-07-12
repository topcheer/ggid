# SAML SP Implementation Guide

Metadata generation, AuthnRequest, ACS handler, signature verification, decryption, logout, and IdP-initiated SSO handling.

## Overview

GGID acts as a SAML 2.0 Service Provider (SP), consuming assertions from external Identity Providers (IdPs). This guide covers the full SP implementation.

## SP Metadata Generation

```bash
GET /saml/metadata.xml
```

```xml
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
  entityID="https://auth.ggid.dev/saml">

  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    
    <KeyDescriptor use="signing">
      <KeyInfo>
        <X509Data>
          <X509Certificate>MIID...base64...</X509Certificate>
        </X509Data>
      </KeyInfo>
    </KeyDescriptor>
    
    <KeyDescriptor use="encryption">
      <KeyInfo>
        <X509Data>
          <X509Certificate>MIID...base64...</X509Certificate>
        </X509Data>
      </KeyInfo>
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

## AuthnRequest (SP-Initiated SSO)

### Generate AuthnRequest

```go
func GenerateAuthnRequest(idpID, acsURL string) (string, error) {
    requestID := generateID() // UUID
    
    authnRequest := saml.AuthnRequest{
        ID:           requestID,
        Version:      "2.0",
        IssueInstant: time.Now().UTC(),
        Destination:  idpSSOURL,
        Issuer: &saml.Issuer{
            Value: spEntityID, // "https://auth.ggid.dev/saml"
        },
        AssertionConsumerServiceURL: acsURL,
        NameIDPolicy: &saml.NameIDPolicy{
            AllowCreate: true,
            Format:      "urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
        },
        RequestedAuthnContext: &saml.RequestedAuthnContext{
            Comparison: "minimum",
            AuthnContextClassRef: []string{
                "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
            },
        },
    }
    
    // Sign and encode as redirect URL
    return EncodeForRedirect(authnRequest, spPrivateKey)
}
```

### Redirect User

```
302 Location: https://idp.com/sso?SAMLRequest=base64-deflate-encoded&RelayState=return-url
```

### HTTP-POST Binding (Alternative)

```html
<form method="POST" action="https://idp.com/sso">
  <input type="hidden" name="SAMLRequest" value="base64-encoded-xml"/>
  <input type="hidden" name="RelayState" value="return-url"/>
</form>
```

## ACS Handler (Assertion Consumer Service)

```go
func ACSHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Decode SAML response
    samlResponse := r.PostFormValue("SAMLResponse")
    response, err := decodeAndParse(samlResponse)
    if err != nil {
        http.Error(w, "invalid SAML response", 400)
        return
    }
    
    // 2. Verify response status
    if response.Status.StatusCode.Value != "urn:oasis:names:tc:SAML:2.0:status:Success" {
        handleSAMLFailure(w, response.Status)
        return
    }
    
    // 3. Verify conditions
    assertion := response.Assertion
    if err := verifyConditions(assertion, spEntityID); err != nil {
        http.Error(w, "assertion conditions not met", 403)
        return
    }
    
    // 4. Verify signature
    if err := verifySignature(assertion, idpCert); err != nil {
        http.Error(w, "invalid signature", 403)
        return
    }
    
    // 5. Decrypt assertion (if encrypted)
    if assertion.EncryptedAssertion != nil {
        assertion, err = decryptAssertion(assertion, spPrivateKey)
        if err != nil {
            http.Error(w, "decryption failed", 403)
            return
        }
    }
    
    // 6. Verify timestamps
    if time.Now().After(assertion.Conditions.NotOnOrAfter) {
        http.Error(w, "assertion expired", 403)
        return
    }
    
    // 7. Extract attributes
    attrs := extractAttributes(assertion)
    
    // 8. Provision/update user (JIT)
    user := provisionOrLookupUser(attrs)
    
    // 9. Issue GGID session
    issueSession(w, user)
}
```

## Signature Verification

```go
func verifySignature(assertion *saml.Assertion, cert *x509.Certificate) error {
    sig := assertion.Signature
    if sig == nil {
        return ErrNoSignature
    }
    
    // Extract SignedInfo
    signedInfo := sig.SignedInfo
    
    // Compute canonical XML (C14N) of SignedInfo
    canonical, err := canonicalize(signedInfo)
    if err != nil {
        return err
    }
    
    // Verify signature
    hash := crypto.SHA256.New()
    hash.Write(canonical)
    
    return cert.CheckSignature(x509.SHA256WithRSA, canonical, sig.SignatureValue)
}
```

### Signature Requirements

| Requirement | Enforcement |
|-------------|-------------|
| Response signed | Required (or assertion signed) |
| Assertion signed | Required if response not signed |
| Algorithm | SHA-256 or SHA-512 (SHA-1 rejected) |
| Certificate | Must match IdP metadata |

## Assertion Decryption

```go
func decryptAssertion(encrypted *saml.EncryptedAssertion, privKey *rsa.PrivateKey) (*saml.Assertion, error) {
    // 1. Decrypt symmetric key using RSA-OAEP
    encryptedKey := encrypted.EncryptedKey
    symmetricKey, err := rsa.DecryptOAEP(
        sha256.New(), rand.Reader, privKey, encryptedKey.CipherValue, nil,
    )
    if err != nil {
        return nil, ErrKeyDecryption
    }
    
    // 2. Decrypt assertion using AES-256-GCM
    block, _ := aes.NewCipher(symmetricKey)
    gcm, _ := cipher.NewGCM(block)
    
    plaintext, err := gcm.Open(nil, encrypted.IV, encrypted.CipherValue, encrypted.AuthTag)
    if err != nil {
        return nil, ErrAssertionDecryption
    }
    
    // 3. Parse decrypted assertion
    return parseAssertion(plaintext)
}
```

## Logout (SLO)

### SP-Initiated Logout

```go
func InitiateLogout(w http.ResponseWriter, r *http.Request) {
    logoutRequest := saml.LogoutRequest{
        ID:           generateID(),
        Version:      "2.0",
        IssueInstant: time.Now().UTC(),
        Destination:  idpSLOURL,
        Issuer:       &saml.Issuer{Value: spEntityID},
        NameID:       session.NameID,
        SessionIndex: session.SessionIndex,
    }
    
    // Redirect to IdP SLO endpoint
    redirectURL := EncodeForRedirect(logoutRequest, spPrivateKey)
    http.Redirect(w, r, redirectURL, 302)
}
```

### IdP-Initiated Logout

IdP sends LogoutRequest to GGID's SLO endpoint:

```go
func SLOHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Parse LogoutRequest
    req := decodeLogoutRequest(r)
    
    // 2. Terminate user session
    terminateSession(req.NameID, req.SessionIndex)
    
    // 3. Respond with LogoutResponse
    resp := saml.LogoutResponse{
        ID:           generateID(),
        InResponseTo: req.ID,
        Version:      "2.0",
        IssueInstant: time.Now().UTC(),
        Destination:  idpSLOURL,
        Status:       saml.Status{StatusCode: saml.Success},
    }
    
    // Redirect back to IdP
    redirectURL := EncodeForRedirect(resp, spPrivateKey)
    http.Redirect(w, r, redirectURL, 302)
}
```

## IdP-Initiated SSO

When IdP sends unsolicited SAML response (no AuthnRequest):

```go
func handleIdPInitiated(response *saml.Response) error {
    // 1. Verify InResponseTo is empty (unsolicited)
    if response.InResponseTo != "" {
        return ErrUnexpectedResponse // Should be empty for IdP-initiated
    }
    
    // 2. Verify assertion (same as ACS handler)
    // 3. Look up user by NameID or attributes
    // 4. If no matching AuthnRequest, check RelayState for target
    
    // Allow only if configured
    if !allowIdPInitiated {
        return ErrIdPInitiatedDisabled
    }
}
```

## Attribute Extraction

```go
func extractAttributes(assertion *saml.Assertion) map[string][]string {
    attrs := make(map[string][]string)
    for _, statement := range assertion.AttributeStatements {
        for _, attr := range statement.Attributes {
            attrs[attr.Name] = attr.Values
        }
    }
    return attrs
}

// Map to GGID canonical fields
func mapAttributes(attrs map[string][]string) UserClaims {
    return UserClaims{
        Email:       firstValue(attrs, claimEmailURI),
        DisplayName: firstValue(attrs, claimNameURI),
        Department:  firstValue(attrs, claimDeptURI),
    }
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| SAML response verification failures | >5% → check cert expiry |
| Assertion expired errors | Spike → clock skew between SP and IdP |
| Encryption failures | Any → check SP private key |
| IdP-initiated SSO blocked | Log for review |

## See Also

- [Identity Provider Configuration](identity-provider-configuration.md)
- [Identity Federation Architecture](identity-federation-architecture.md)
- [Authentication Flows](authentication-flows.md)
- [WebAuthn Server Implementation](webauthn-server-implementation.md)
