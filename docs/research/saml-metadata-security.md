# SAML Metadata Security for IAM Systems

> **Focus:** Metadata security — XML signature wrapping, metadata freshness, trust chain validation, entity categories, certificate handling, and metadata injection prevention.
>
> **Out of scope:** Multi-tenant SAML architecture, SP/IdP configuration workflows, attribute mapping strategies. See `docs/research/multi-tenant-saml.md` for those topics.

---

## Table of Contents

1. [SAML Metadata Structure](#1-saml-metadata-structure)
2. [XML Signature Wrapping Attacks](#2-xml-signature-wrapping-attacks)
3. [Metadata Signature Validation](#3-metadata-signature-validation)
4. [Metadata Freshness and Refresh](#4-metadata-freshness-and-refresh)
5. [Trust Chain Validation](#5-trust-chain-validation)
6. [Entity Categories](#6-entity-categories)
7. [Metadata Injection Prevention](#7-metadata-injection-prevention)
8. [Certificate Handling in Metadata](#8-certificate-handling-in-metadata)
9. [GGID SAML Metadata Audit](#9-ggid-saml-metadata-audit)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. SAML Metadata Structure

### 1.1 EntityDescriptor

SAML metadata is expressed as XML conforming to the SAML 2.0 metadata schema (`urn:oasis:names:tc:SAML:2.0:metadata`). The root element is `EntityDescriptor`, which uniquely identifies a federation participant:

```xml
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"
                  entityID="https://sp.ggid.example.com/metadata"
                  validUntil="2026-01-01T00:00:00Z">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"
                   AuthnRequestsSigned="true"
                   WantAssertionsSigned="true">
    <KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDC...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <KeyDescriptor use="encryption">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>MIIDC...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <AssertionConsumerService index="0"
                              Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                              Location="https://sp.ggid.example.com/acs"
                              isDefault="true"/>
    <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
                         Location="https://sp.ggid.example.com/slo"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

### 1.2 Role Descriptors

Each `EntityDescriptor` contains one or more role descriptors:

| Descriptor | Role | Key Elements |
|---|---|---|
| `IDPSSODescriptor` | Identity Provider | `SingleSignOnService`, `SingleLogoutService`, `NameIDFormat`, `Attribute` |
| `SPSSODescriptor` | Service Provider | `AssertionConsumerService`, `SingleLogoutService`, `NameIDFormat` |
| `AttributeAuthorityDescriptor` | Attribute Authority | `AttributeService`, `NameIDFormat` |

### 1.3 Go Metadata Parsing

```go
package samlmeta

import (
    "crypto/x509"
    "encoding/base64"
    "encoding/xml"
    "fmt"
    "time"
)

// EntityDescriptor is the root metadata element.
type EntityDescriptor struct {
    XMLName        xml.Name         `xml:"EntityDescriptor"`
    EntityID       string           `xml:"entityID,attr"`
    ValidUntil     string           `xml:"validUntil,attr,omitempty"`
    CacheDuration  string           `xml:"cacheDuration,attr,omitempty"`
    IDPSSODescriptor  *IDPSSODescriptor  `xml:"IDPSSODescriptor,omitempty"`
    SPSSODescriptor   *SPSSODescriptor   `xml:"SPSSODescriptor,omitempty"`
}

// IDPSSODescriptor describes IdP capabilities.
type IDPSSODescriptor struct {
    WantAuthnRequestsSigned bool                 `xml:"wantAuthnRequestsSigned,attr"`
    KeyDescriptors           []KeyDescriptor       `xml:"KeyDescriptor"`
    NameIDFormats            []NameIDFormat        `xml:"NameIDFormat"`
    SingleSignOnServices     []EndpointDescriptor  `xml:"SingleSignOnService"`
    SingleLogoutServices     []EndpointDescriptor  `xml:"SingleLogoutService"`
}

// KeyDescriptor holds signing or encryption key info.
type KeyDescriptor struct {
    Use     string     `xml:"use,attr"`
    KeyInfo KeyInfo    `xml:"KeyInfo"`
}

// KeyInfo wraps the X.509 certificate.
type KeyInfo struct {
    X509Data X509Data `xml:"X509Data"`
}

// X509Data contains the certificate.
type X509Data struct {
    X509Certificate string `xml:"X509Certificate"`
}

// NameIDFormat declares supported name ID formats.
type NameIDFormat struct {
    Value string `xml:",chardata"`
}

// EndpointDescriptor describes a SAML endpoint.
type EndpointDescriptor struct {
    Binding  string `xml:"Binding,attr"`
    Location string `xml:"Location,attr"`
}

// ParseMetadata parses a metadata XML document.
func ParseMetadata(rawXML []byte) (*EntityDescriptor, error) {
    var ed EntityDescriptor
    if err := xml.Unmarshal(rawXML, &ed); err != nil {
        return nil, fmt.Errorf("parse metadata XML: %w", err)
    }
    if ed.EntityID == "" {
        return nil, fmt.Errorf("metadata missing entityID")
    }
    return &ed, nil
}

// ExtractSigningCertificates returns all certificates marked use="signing".
func (ed *EntityDescriptor) ExtractSigningCertificates() ([]*x509.Certificate, error) {
    var certs []*x509.Certificate
    for _, kd := range ed.IDPSSODescriptor.KeyDescriptors {
        if kd.Use != "signing" {
            continue
        }
        cert, err := parseCertFromBase64(kd.KeyInfo.X509Data.X509Certificate)
        if err != nil {
            return nil, fmt.Errorf("parse signing certificate: %w", err)
        }
        certs = append(certs, cert)
    }
    return certs, nil
}

// ValidUntilTime parses the validUntil attribute into time.Time.
func (ed *EntityDescriptor) ValidUntilTime() (time.Time, error) {
    if ed.ValidUntil == "" {
        return time.Time{}, fmt.Errorf("metadata has no validUntil")
    }
    return time.Parse(time.RFC3339, ed.ValidUntil)
}

func parseCertFromBase64(b64 string) (*x509.Certificate, error) {
    der, err := base64.StdEncoding.DecodeString(b64)
    if err != nil {
        return nil, fmt.Errorf("base64 decode certificate: %w", err)
    }
    return x509.ParseCertificate(der)
}
```

---

## 2. XML Signature Wrapping Attacks

### 2.1 The Core Vulnerability

XML Signature Wrapping (XSW) is the most critical class of SAML metadata and assertion attacks. The fundamental issue: **XML signature validation verifies that a signature element is mathematically valid, but standard XML-Sig processing does NOT verify that the application consumes the same data that was signed.**

An attacker exploits the gap between the signature verification layer and the application logic layer:

```
Original signed document:
<EntityDescriptor ID="_abc">
  <ds:Signature>  <!-- signs _abc -->
    <ds:Reference URI="#_abc"/>
  </ds:Signature>
  ...legitimate content...
</EntityDescriptor>

XSW-attacked document:
<EntityDescriptor ID="_original">      <!-- attacker clones root -->
  <EntityDescriptor ID="_abc">         <!-- attacker moves original here -->
    <ds:Signature>                      <!-- still valid! signs _abc -->
      <ds:Reference URI="#_abc"/>
    </ds:Signature>
    ...legitimate content...
  </EntityDescriptor>
  <EntityDescriptor ID="_malicious">   <!-- attacker's payload -->
    ...malicious endpoints/keys...      <!-- application reads THIS -->
  </EntityDescriptor>
</EntityDescriptor>
```

The signature library verifies the inner `_abc` element (valid). The application's `xml.Unmarshal` deserializes the outer document structure, potentially reading the malicious entity content.

### 2.2 Attack Variants

**XSW-1 (Wrapping Attack):** The attacker wraps the original signed element inside a new root. The signature remains valid because it references the inner element by ID. The application reads from the outer element.

**XSW-2 (Sibling Insertion):** The attacker inserts a malicious sibling after the signed element. Both elements share the same local name. The application may read the second (malicious) instance.

**XSW-3 (Signature Sibling):** The signature element is moved to be a sibling of the original. The attacker inserts new content between the signature and the original.

**XInclude/XSLT Attacks:** If the XML processor supports XInclude or XSLT transforms in the signature's `<ds:Transform>` pipeline, an attacker can use these to inject external entities or execute server-side transforms. These transforms should be disabled in all SAML processing.

### 2.3 Why Standard Validation Fails

Standard XML signature libraries (including Go's `encoding/xml` and most third-party XML-Sig implementations) follow the W3C XML Signature specification, which:

1. Locates the `<ds:Reference>` element and resolves its URI.
2. Applies transforms to the referenced element.
3. Computes a digest and compares it to `<ds:DigestValue>`.
4. Verifies the cryptographic signature over `<ds:SignedInfo>`.

The problem: **the specification does not mandate that the application consume the element referenced by the signature.** The library confirms a signature exists and is valid, but the application may use a completely different part of the document tree.

### 2.4 Countermeasure: Referenced Element Extraction

```go
package samlsec

import (
    "bytes"
    "encoding/xml"
    "fmt"
)

// XSWSafeUnmarshal validates that the parsed object's root element
// is the same element referenced by the signature. It performs:
//  1. Standard XML deserialization
//  2. Verification that the root element ID matches the signature Reference URI
//  3. Confirmation that no duplicate IDs exist in the document
//
// This is a defense-in-depth measure; production systems should use a
// dedicated XML-Sig library that performs strict referenced-element extraction.
func XSWSafeUnmarshal(rawXML []byte, target interface{}) error {
    // Step 1: Check for duplicate IDs (a hallmark of wrapping attacks).
    if err := detectDuplicateIDs(rawXML); err != nil {
        return fmt.Errorf("XSW protection: %w", err)
    }

    // Step 2: Standard unmarshal.
    if err := xml.Unmarshal(rawXML, target); err != nil {
        return fmt.Errorf("unmarshal: %w", err)
    }

    // Step 3: Verify that the signature Reference URI points to the root element.
    // This requires the target to implement IDHolder.
    if holder, ok := target.(IDHolder); ok {
        refURI, err := extractSignatureReferenceURI(rawXML)
        if err != nil {
            return fmt.Errorf("extract signature reference: %w", err)
        }
        expectedURI := "#" + holder.GetID()
        if refURI != expectedURI {
            return fmt.Errorf("signature references %s but root element ID is %s — possible wrapping attack",
                refURI, expectedURI)
        }
    }

    return nil
}

// IDHolder is implemented by types that have an ID attribute.
type IDHolder interface {
    GetID() string
}

// detectDuplicateIDs scans the XML for elements sharing the same ID attribute.
func detectDuplicateIDs(rawXML []byte) error {
    decoder := xml.NewDecoder(bytes.NewReader(rawXML))
    seen := make(map[string]string) // ID -> element name

    for {
        tok, err := decoder.Token()
        if err != nil {
            if err.Error() == "EOF" {
                break
            }
            return nil // Non-fatal: let unmarshal handle real errors
        }

        startEl, ok := tok.(xml.StartElement)
        if !ok {
            continue
        }

        for _, attr := range startEl.Attr {
            if attr.Name.Local == "ID" && attr.Value != "" {
                if prevName, exists := seen[attr.Value]; exists {
                    return fmt.Errorf("duplicate ID %q found in elements %q and %q",
                        attr.Value, prevName, startEl.Name.Local)
                }
                seen[attr.Value] = startEl.Name.Local
            }
        }
    }
    return nil
}

// extractSignatureReferenceURI extracts the URI from the first ds:Reference.
func extractSignatureReferenceURI(rawXML []byte) (string, error) {
    // Use xml.Unmarshal into a targeted structure.
    type ref struct {
        URI string `xml:"Reference,attr"`
    }
    type sig struct {
        Refs []ref `xml:"SignedInfo"`
    }
    var s sig
    if err := xml.Unmarshal(rawXML, &s); err != nil {
        return "", err
    }
    if len(s.Refs) == 0 {
        return "", fmt.Errorf("no Reference element found in signature")
    }
    return s.Refs[0].URI, nil
}
```

---

## 3. Metadata Signature Validation

### 3.1 Signature Coverage Verification

The most important metadata signature check: **the signature must cover the entire EntityDescriptor**, not just a sub-element. A signature that references only a KeyDescriptor or an endpoint, but not the root, is insufficient.

The `ds:Reference` element's URI attribute should either:
- Be empty (whole-document signature), or
- Reference the EntityDescriptor's `ID` attribute (e.g., `URI="#_abc123"`).

### 3.2 Key Resolution

Metadata signatures can be validated using two key resolution strategies:

| Strategy | Description | Security Level |
|---|---|---|
| **In-band** | Certificate embedded in the metadata's `KeyDescriptor` | Low — attacker controls the cert |
| **Out-of-band** | Certificate provisioned through a trusted channel | High |
| **Federation** | Certificate from the federation operator's metadata signing key | High |

For metadata obtained from a federation feed, the signature must be validated against the **federation operator's metadata signing certificate**, not against a certificate embedded within the metadata itself.

### 3.3 Canonicalization (C14N) Risks

XML signatures are computed over the canonicalized form of the signed element. The C14N algorithm strips certain XML variations (whitespace, namespace prefixes, attribute ordering) to produce a canonical byte representation. Risks:

- **C14N ambiguity:** Different XML libraries may canonicalize differently, causing signature validation to fail or pass incorrectly.
- **C14N injection:** In rare cases, namespace context differences between the canonicalization step and the application's parsing step can create exploitable gaps.
- **Exclusive C14N (`xml-c14n11`):** Preferred over inclusive C14N to prevent namespace injection.

### 3.4 Secure Metadata Signature Validation

```go
package samlmeta

import (
    "crypto"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "encoding/xml"
    "fmt"
    "strings"
)

// ValidateMetadataSignature verifies that metadata XML is signed by a
// trusted certificate and that the signature covers the root EntityDescriptor.
//
// trustRoots: the set of certificates trusted to sign this metadata
// (federation operator certs or directly trusted entity certs).
func ValidateMetadataSignature(rawXML []byte, trustRoots []*x509.Certificate) error {
    if len(trustRoots) == 0 {
        return fmt.Errorf("no trust roots provided")
    }

    // Step 1: Extract signature information from the metadata.
    sigInfo, err := extractMetadataSignatureInfo(rawXML)
    if err != nil {
        return fmt.Errorf("extract signature: %w", err)
    }

    // Step 2: Verify the signature reference covers the root element.
    // The URI should be empty (whole document) or reference the root ID.
    rootID, err := extractRootEntityID(rawXML)
    if err != nil {
        return fmt.Errorf("extract root ID: %w", err)
    }
    if sigInfo.referenceURI != "" {
        expectedRef := "#" + rootID
        if sigInfo.referenceURI != expectedRef {
            return fmt.Errorf("metadata signature references %q, expected %q — signature does not cover root",
                sigInfo.referenceURI, expectedRef)
        }
    }

    // Step 3: Attempt to verify the signature against each trusted root.
    verified := false
    for _, cert := range trustRoots {
        if err := verifyRSAMetadataSignature(sigInfo, cert); err == nil {
            verified = true
            break
        }
    }
    if !verified {
        return fmt.Errorf("metadata signature did not validate against any trusted root")
    }

    // Step 4: Verify digest (defense in depth).
    if err := verifyMetadataDigest(sigInfo, rawXML); err != nil {
        return fmt.Errorf("metadata digest verification failed: %w", err)
    }

    return nil
}

type metadataSigInfo struct {
    signedInfoBytes []byte
    signatureValue  []byte
    referenceURI    string
    digestValue     []byte
    digestAlgorithm string
}

func extractMetadataSignatureInfo(rawXML []byte) (*metadataSigInfo, error) {
    type ref struct {
        URI          string `xml:"URI,attr"`
        DigestValue  string `xml:"DigestValue"`
        DigestMethod struct {
            Algorithm string `xml:"Algorithm,attr"`
        } `xml:"DigestMethod"`
    }
    type signedInfo struct {
        Reference ref `xml:"Reference"`
    }
    type sig struct {
        SignedInfo     signedInfo `xml:"SignedInfo"`
        SignatureValue string     `xml:"SignatureValue"`
    }
    type entityWithSig struct {
        XMLName   xml.Name `xml:"EntityDescriptor"`
        ID        string   `xml:"ID,attr"`
        Signature sig      `xml:"Signature"`
    }

    var ews entityWithSig
    if err := xml.Unmarshal(rawXML, &ews); err != nil {
        return nil, fmt.Errorf("parse metadata for signature: %w", err)
    }

    sigVal, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ews.Signature.SignatureValue))
    if err != nil {
        return nil, fmt.Errorf("decode signature value: %w", err)
    }
    digVal, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ews.Signature.SignedInfo.Reference.DigestValue))
    if err != nil {
        return nil, fmt.Errorf("decode digest value: %w", err)
    }

    // Extract raw SignedInfo bytes for signature verification.
    signedInfoBytes := extractSignedInfoRaw(rawXML)

    return &metadataSigInfo{
        signedInfoBytes: signedInfoBytes,
        signatureValue:  sigVal,
        referenceURI:    ews.Signature.SignedInfo.Reference.URI,
        digestValue:     digVal,
        digestAlgorithm: ews.Signature.SignedInfo.Reference.DigestMethod.Algorithm,
    }, nil
}

func extractRootEntityID(rawXML []byte) (string, error) {
    type ed struct {
        XMLName xml.Name `xml:"EntityDescriptor"`
        ID      string   `xml:"ID,attr"`
    }
    var e ed
    if err := xml.Unmarshal(rawXML, &e); err != nil {
        return "", err
    }
    return e.ID, nil
}

func verifyRSAMetadataSignature(info *metadataSigInfo, cert *x509.Certificate) error {
    rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
    if !ok {
        return fmt.Errorf("certificate is not RSA")
    }
    h := sha256.New()
    h.Write(info.signedInfoBytes)
    hashed := h.Sum(nil)
    return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hashed, info.signatureValue)
}

func verifyMetadataDigest(info *metadataSigInfo, rawXML []byte) error {
    h := sha256.New()
    h.Write(rawXML)
    computed := h.Sum(nil)
    if len(info.digestValue) > 0 && !bytesEqual(computed, info.digestValue) {
        return fmt.Errorf("metadata digest mismatch")
    }
    return nil
}

func bytesEqual(a, b []byte) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}

func extractSignedInfoRaw(rawXML []byte) []byte {
    s := string(rawXML)
    start := strings.Index(s, "<ds:SignedInfo")
    if start < 0 {
        start = strings.Index(s, "<SignedInfo")
    }
    if start < 0 {
        return nil
    }
    endTag := "</ds:SignedInfo>"
    end := strings.Index(s[start:], endTag)
    if end < 0 {
        endTag = "</SignedInfo>"
        end = strings.Index(s[start:], endTag)
    }
    if end < 0 {
        return nil
    }
    return []byte(s[start : start+end+len(endTag)])
}
```

---

## 4. Metadata Freshness and Refresh

### 4.1 Why Stale Metadata Is Dangerous

Metadata describes the trust relationship between federated parties. When metadata becomes stale:

- **Revoked IdP remains trusted:** If an IdP's certificate is compromised and revoked, SPs with cached metadata continue to accept assertions signed by the compromised key.
- **Stale endpoints:** An IdP that moves its SSO endpoint leaves stale entries. Attackers can register the old domain and intercept authentication requests.
- **Expired certificate usage:** Certificates in metadata have their own validity period. If metadata is never refreshed, expired certificates may still be used in validation.
- **Removed entity trust:** A federation may remove an entity from its aggregate. If the SP doesn't refresh, it continues trusting the removed entity.

### 4.2 Metadata Validity Attributes

| Attribute | Semantics | Example |
|---|---|---|
| `validUntil` | Metadata must not be used after this time | `2025-12-31T23:59:59Z` |
| `cacheDuration` | Suggested maximum cache time (W3C duration format) | `PT24H` (24 hours) |

### 4.3 Metadata Refresh Manager

```go
package samlmeta

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "sync"
    "time"
)

// MetadataSource describes a federation or entity metadata source.
type MetadataSource struct {
    Name          string
    URL           string
    ExpectedSHA256 string // out-of-band hash for integrity verification
    TrustRoots    []*x509.Certificate
    RefreshInterval time.Duration
}

// MetadataCache stores the most recently fetched and validated metadata.
type MetadataCache struct {
    mu       sync.RWMutex
    entries  map[string]*cachedMetadata
    client   *http.Client
}

type cachedMetadata struct {
    entityID   string
    rawXML     []byte
    validUntil time.Time
    fetchedAt  time.Time
    contentHash string
}

// NewMetadataCache creates a metadata cache with sensible HTTP defaults.
func NewMetadataCache() *MetadataCache {
    return &MetadataCache{
        entries: make(map[string]*cachedMetadata),
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// FetchAndValidate retrieves metadata from a source, verifies its integrity
// hash, validates its signature, and checks freshness.
func (mc *MetadataCache) FetchAndValidate(ctx context.Context, src MetadataSource) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", src.URL, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("Accept", "application/xml")

    resp, err := mc.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch metadata from %s: %w", src.URL, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("metadata fetch returned HTTP %d", resp.StatusCode)
    }

    rawXML, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB max
    if err != nil {
        return nil, fmt.Errorf("read metadata body: %w", err)
    }

    // Step 1: Verify content hash (metadata injection prevention).
    if src.ExpectedSHA256 != "" {
        hash := sha256.Sum256(rawXML)
        actualHash := hex.EncodeToString(hash[:])
        if actualHash != src.ExpectedSHA256 {
            return nil, fmt.Errorf("metadata hash mismatch: expected %s, got %s — possible injection",
                src.ExpectedSHA256, actualHash)
        }
    }

    // Step 2: Verify metadata signature.
    if len(src.TrustRoots) > 0 {
        if err := ValidateMetadataSignature(rawXML, src.TrustRoots); err != nil {
            return nil, fmt.Errorf("metadata signature validation: %w", err)
        }
    }

    // Step 3: Parse metadata and check freshness.
    ed, err := ParseMetadata(rawXML)
    if err != nil {
        return nil, fmt.Errorf("parse metadata: %w", err)
    }

    validUntil, err := ed.ValidUntilTime()
    if err != nil {
        return nil, fmt.Errorf("parse validUntil: %w", err)
    }

    if time.Now().UTC().After(validUntil) {
        return nil, fmt.Errorf("metadata expired: validUntil %s", validUntil.Format(time.RFC3339))
    }

    // Step 4: Check validUntil is not too far in the future (sanity).
    maxValidity := 365 * 24 * time.Hour
    if validUntil.Sub(time.Now().UTC()) > maxValidity {
        return nil, fmt.Errorf("metadata validUntil unreasonably far: %s", validUntil)
    }

    // Step 5: Cache the validated metadata.
    mc.mu.Lock()
    mc.entries[src.Name] = &cachedMetadata{
        entityID:   ed.EntityID,
        rawXML:     rawXML,
        validUntil: validUntil,
        fetchedAt:  time.Now().UTC(),
    }
    mc.mu.Unlock()

    return rawXML, nil
}

// StartRefreshLoop periodically refreshes metadata from a source.
func (mc *MetadataCache) StartRefreshLoop(ctx context.Context, src MetadataSource) {
    ticker := time.NewTicker(src.RefreshInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if _, err := mc.FetchAndValidate(ctx, src); err != nil {
                // Log error; continue using cached metadata if still valid.
                // In production: emit structured log, metric increment, alert.
                fmt.Printf("metadata refresh for %s failed: %v\n", src.Name, err)
            }
        }
    }
}

// Get retrieves validated metadata from cache. Returns error if expired.
func (mc *MetadataCache) Get(name string) ([]byte, error) {
    mc.mu.RLock()
    defer mc.mu.RUnlock()

    entry, ok := mc.entries[name]
    if !ok {
        return nil, fmt.Errorf("no cached metadata for %s", name)
    }
    if time.Now().UTC().After(entry.validUntil) {
        return nil, fmt.Errorf("cached metadata for %s expired (validUntil %s)",
            name, entry.validUntil.Format(time.RFC3339))
    }
    return entry.rawXML, nil
}
```

---

## 5. Trust Chain Validation

### 5.1 Federation Trust Model

In a federation, trust is indirect:

```
SP  →  trusts  →  Federation Operator (signs metadata aggregate)
                         ↓
                    signs Entity Metadata
                         ↓
                    IdP Entity (within aggregate)
```

The federation operator signs the aggregate metadata document. The SP validates the aggregate signature against the federation operator's root certificate (distributed out-of-band). Individual entity metadata within the aggregate inherits trust from the aggregate signature.

### 5.2 Multiple Trust Roots

A production IAM system typically has multiple trust sources:

1. **Direct trust:** SP and IdP exchange metadata directly (bilateral trust).
2. **Federation trust:** Both are members of a federation (e.g., InCommon, eduGAIN).
3. **Cross-federation:** Federations bridge to each other (e.g., eduGAIN ↔ Kalmar2).

### 5.3 Trust Chain Validator

```go
package samlmeta

import (
    "crypto/x509"
    "fmt"
)

// TrustMode defines how metadata is trusted.
type TrustMode int

const (
    TrustModeDirect     TrustMode = iota // Bilateral metadata exchange
    TrustModeFederation                  // Via federation aggregate
)

// TrustRoot represents a trust anchor.
type TrustRoot struct {
    Mode         TrustMode
    Certificate  *x509.Certificate
    FederationID string // For federation trust roots
}

// TrustChainValidator evaluates whether an entity is trusted.
type TrustChainValidator struct {
    roots       []TrustRoot
    knownEntities map[string]bool // Directly trusted entityIDs
}

// NewTrustChainValidator creates a validator with the given trust roots.
func NewTrustChainValidator(roots []TrustRoot) *TrustChainValidator {
    return &TrustChainValidator{
        roots:         roots,
        knownEntities: make(map[string]bool),
    }
}

// AddDirectTrust registers a directly-trusted entity ID.
func (v *TrustChainValidator) AddDirectTrust(entityID string) {
    v.knownEntities[entityID] = true
}

// ValidateTrust evaluates whether an entity descriptor is trusted
// under any of the configured trust roots.
func (v *TrustChainValidator) ValidateTrust(ed *EntityDescriptor, rawXML []byte) error {
    // Direct trust: entity ID is pre-registered.
    if v.knownEntities[ed.EntityID] {
        // Still validate signature against direct trust roots.
        var directRoots []*x509.Certificate
        for _, r := range v.roots {
            if r.Mode == TrustModeDirect {
                directRoots = append(directRoots, r.Certificate)
            }
        }
        if len(directRoots) > 0 {
            return ValidateMetadataSignature(rawXML, directRoots)
        }
        return nil
    }

    // Federation trust: validate against federation aggregate signature.
    var fedRoots []*x509.Certificate
    for _, r := range v.roots {
        if r.Mode == TrustModeFederation {
            fedRoots = append(fedRoots, r.Certificate)
        }
    }
    if len(fedRoots) == 0 {
        return fmt.Errorf("entity %q not directly trusted and no federation roots configured", ed.EntityID)
    }

    if err := ValidateMetadataSignature(rawXML, fedRoots); err != nil {
        return fmt.Errorf("federation trust validation failed: %w", err)
    }

    return nil
}
```

---

## 6. Entity Categories

### 6.1 Purpose

Entity categories are metadata attributes that classify federation participants by their data handling practices. They allow SPs and IdPs to make attribute release decisions based on the category of the requesting party, rather than bilateral agreements.

### 6.2 Standard Categories

| Category | URI | Meaning |
|---|---|---|
| Research & Scholarship | `http://refeds.org/category/research-and-scholarship` | SP supports research collaboration; minimal attribute release |
| REFEDS CoCo | `https://refeds.org/sirtfi` | Security Incident Response Trust Framework |
| Personal Data | `http://macedir.org/entity-category-support/personalized` | SP handles personal data with appropriate privacy controls |
| Anonymous Access | `https://refeds.org/entity-category/anonymous` | No persistent identifier required |

### 6.3 Attribute Release Based on Categories

IdPs use entity categories to determine which attributes to release without requiring per-SP attribute release policies:

```xml
<!-- SP metadata declaring entity category -->
<EntitiesDescriptor>
  <EntityDescriptor entityID="https://research.example.com/sp">
    <Extensions>
      <mdattr:EntityAttributes xmlns:mdattr="urn:oasis:names:tc:SAML:metadata:attribute">
        <saml:Attribute Name="http://macedir.org/entity-category"
                        NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
          <saml:AttributeValue>http://refeds.org/category/research-and-scholarship</saml:AttributeValue>
        </saml:Attribute>
      </mdattr:EntityAttributes>
    </Extensions>
    ...
  </EntityDescriptor>
</EntitiesDescriptor>
```

### 6.4 Entity Category-Based Attribute Filtering

```go
package samlmeta

import (
    "encoding/xml"
    "fmt"
    "strings"
)

// EntityCategory constants from REFEDS and Macedir.
const (
    EntityCategoryResearchAndScholarship = "http://refeds.org/category/research-and-scholarship"
    EntityCategoryCoCo                   = "http://macedir.org/entity-category/coconut"
    EntityCategoryPersonalData           = "http://macedir.org/entity-category-support/personalized"
)

// EntityAttributes holds category attributes parsed from metadata extensions.
type EntityAttributes struct {
    Attributes []EntityAttribute `xml:"Attribute"`
}

// EntityAttribute is a single category or assurance attribute.
type EntityAttribute struct {
    Name   string   `xml:"Name,attr"`
    Values []string `xml:"AttributeValue"`
}

// HasCategory checks whether the entity declares a specific category.
func (ea *EntityAttributes) HasCategory(category string) bool {
    for _, attr := range ea.Attributes {
        if attr.Name == "http://macedir.org/entity-category" {
            for _, v := range attr.Values {
                if v == category {
                    return true
                }
            }
        }
    }
    return false
}

// ParseEntityAttributes extracts EntityAttributes from metadata extensions.
func ParseEntityAttributes(rawXML []byte) (*EntityAttributes, error) {
    type extHolder struct {
        XMLName  xml.Name         `xml:"EntityDescriptor"`
        Extensions *struct {
            EntityAttributes EntityAttributes `xml:"EntityAttributes"`
        } `xml:"Extensions"`
    }
    var h extHolder
    if err := xml.Unmarshal(rawXML, &h); err != nil {
        return nil, fmt.Errorf("parse entity attributes: %w", err)
    }
    if h.Extensions == nil {
        return &EntityAttributes{}, nil
    }
    return &h.Extensions.EntityAttributes, nil
}

// AttributeReleasePolicy determines which attributes to release based on
// entity categories and requested attributes.
type AttributeReleasePolicy struct {
    // DefaultAttributes are released to all authenticated entities.
    DefaultAttributes []string
    // RAndSAttributes are the R&S category attribute bundle.
    RAndSAttributes []string
}

// EvaluateAttributeRelease returns the set of attributes to release to a
// requesting SP based on its entity categories.
func (p *AttributeReleasePolicy) EvaluateAttributeRelease(ea *EntityAttributes) []string {
    released := make(map[string]bool)
    for _, a := range p.DefaultAttributes {
        released[a] = true
    }

    if ea.HasCategory(EntityCategoryResearchAndScholarship) {
        for _, a := range p.RAndSAttributes {
            released[a] = true
        }
    }

    var result []string
    for a := range released {
        result = append(result, a)
    }
    return result
}

// DefaultRAndSPolicy returns the standard Research & Scholarship
// attribute bundle as defined by REFEDS.
func DefaultRAndSPolicy() *AttributeReleasePolicy {
    return &AttributeReleasePolicy{
        DefaultAttributes: []string{
            "urn:oid:1.3.6.1.4.1.5923.1.1.1.10", // eduPersonTargetedID
        },
        RAndSAttributes: []string{
            "urn:oid:1.3.6.1.4.1.5923.1.1.1.6",  // eduPersonPrincipalName
            "urn:oid:1.3.6.1.4.1.5923.1.1.1.9",  // eduPersonScopedAffiliation
            "urn:oid:2.5.4.3",                     // cn (commonName)
            "urn:oid:0.9.2342.19200300.100.1.1",   // uid
            "urn:oid:0.9.2342.19200300.100.1.3",   // mail
        },
    }
}

// FormatAttributesForDisplay returns a human-readable attribute list.
func FormatAttributesForDisplay(attrs []string) string {
    return strings.Join(attrs, ", ")
}
```

---

## 7. Metadata Injection Prevention

### 7.1 Attack Vectors

Metadata injection occurs when an attacker substitutes legitimate metadata with their own:

1. **Compromised federation feed:** If the federation's metadata distribution server is compromised, attackers can publish malicious aggregate metadata. Mitigated by signature validation against the federation root.

2. **DNS poisoning of metadata URL:** If the SP fetches metadata from `https://idp.example.com/metadata`, DNS poisoning redirects the fetch to an attacker-controlled server. Mitigated by TLS certificate validation and DNSSEC.

3. **MITM on metadata fetch:** Without TLS, a man-in-the-middle can substitute metadata in transit. Mitigated by mandatory TLS.

4. **Metadata URL spoofing:** An attacker registers a similar-looking domain (e.g., `idp.examp1e.com`) and tricks an administrator into configuring it as the metadata source.

### 7.2 Metadata Integrity Validator

```go
package samlmeta

import (
    "context"
    "crypto/sha256"
    "crypto/x509"
    "encoding/hex"
    "fmt"
    "net/http"
    "time"
)

// MetadataIntegrityConfig configures metadata integrity verification.
type MetadataIntegrityConfig struct {
    // KnownHash is the expected SHA-256 hash of the metadata document.
    // This should be obtained out-of-band (e.g., via a phone call, signed email).
    KnownHash string

    // KnownCertFingerprint is the expected SHA-256 fingerprint of the
    // TLS certificate serving the metadata URL.
    KnownCertFingerprint string

    // RequireHTTPS forces HTTPS for metadata fetches.
    RequireHTTPS bool

    // MaxMetadataSize is the maximum allowed metadata document size.
    MaxMetadataSize int64
}

// MetadataIntegrityValidator verifies metadata integrity before acceptance.
type MetadataIntegrityValidator struct {
    config MetadataIntegrityConfig
    client *http.Client
}

// NewMetadataIntegrityValidator creates a validator with the given configuration.
func NewMetadataIntegrityValidator(config MetadataIntegrityConfig) *MetadataIntegrityValidator {
    return &MetadataIntegrityValidator{
        config: config,
        client: &http.Client{
            Timeout: 30 * time.Second,
            // In production: configure custom TLS with cert pinning.
        },
    }
}

// ValidateIntegrity fetches metadata and performs all integrity checks.
func (v *MetadataIntegrityValidator) ValidateIntegrity(ctx context.Context, metadataURL string) ([]byte, error) {
    // Step 1: Enforce HTTPS.
    if v.config.RequireHTTPS && !strings.HasPrefix(metadataURL, "https://") {
        return nil, fmt.Errorf("metadata URL must use HTTPS: %s", metadataURL)
    }

    // Step 2: Fetch metadata.
    req, err := http.NewRequestWithContext(ctx, "GET", metadataURL, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := v.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch metadata: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("metadata fetch returned HTTP %d", resp.StatusCode)
    }

    // Step 3: Verify TLS certificate fingerprint (if configured).
    if v.config.KnownCertFingerprint != "" && resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
        actualFp := certFingerprint(resp.TLS.PeerCertificates[0])
        if actualFp != v.config.KnownCertFingerprint {
            return nil, fmt.Errorf("TLS certificate fingerprint mismatch: expected %s, got %s",
                v.config.KnownCertFingerprint, actualFp)
        }
    }

    // Step 4: Read metadata with size limit.
    maxSize := v.config.MaxMetadataSize
    if maxSize == 0 {
        maxSize = 10 << 20 // Default 10MB
    }
    rawXML, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
    if err != nil {
        return nil, fmt.Errorf("read metadata: %w", err)
    }

    // Step 5: Verify content hash (out-of-band integrity check).
    if v.config.KnownHash != "" {
        hash := sha256.Sum256(rawXML)
        actualHash := hex.EncodeToString(hash[:])
        if actualHash != v.config.KnownHash {
            return nil, fmt.Errorf("metadata content hash mismatch: expected %s, got %s",
                v.config.KnownHash, actualHash)
        }
    }

    return rawXML, nil
}

// certFingerprint computes the SHA-256 fingerprint of a certificate.
func certFingerprint(cert *x509.Certificate) string {
    hash := sha256.Sum256(cert.Raw)
    return hex.EncodeToString(hash[:])
}

// VerifyOutOfBand performs manual out-of-band verification steps that
// an administrator should execute when initially establishing trust.
func VerifyOutOfBand(metadataURL, expectedHash, contactPerson string) error {
    fmt.Println("=== Out-of-Band Metadata Verification Checklist ===")
    fmt.Printf("1. Verify metadata URL: %s\n", metadataURL)
    fmt.Printf("2. Confirm expected SHA-256 hash: %s\n", expectedHash)
    fmt.Printf("3. Contact %s via known phone/email to confirm hash\n", contactPerson)
    fmt.Println("4. Verify the hash matches before configuring automated refresh")
    fmt.Println("5. Document the verification date and method for audit")
    return nil
}
```

---

## 8. Certificate Handling in Metadata

### 8.1 KeyDescriptor Usage

SAML metadata uses `KeyDescriptor` elements to publish the certificates a party uses for specific purposes:

| `use` attribute | Purpose | Who uses it |
|---|---|---|
| `signing` | Signing AuthnRequests, assertions, logout messages | Verifying party validates signatures with this cert |
| `encryption` | Encrypting assertions (XML Encryption) | IdP encrypts assertions to this cert |

### 8.2 Certificate Rotation

During rotation, metadata should contain **both** the old and new certificates simultaneously for an overlap period (typically 1-2 weeks). This prevents service disruption:

```xml
<!-- During rotation: both certs present -->
<KeyDescriptor use="signing">
  <ds:KeyInfo><ds:X509Data><ds:X509Certificate>OLD_CERT...</ds:X509Certificate></ds:X509Data></ds:KeyInfo>
</KeyDescriptor>
<KeyDescriptor use="signing">
  <ds:KeyInfo><ds:X509Data><ds:X509Certificate>NEW_CERT...</ds:X509Certificate></ds:X509Data></ds:KeyInfo>
</KeyDescriptor>
```

### 8.3 Certificate Extraction and Rotation Manager

```go
package samlmeta

import (
    "crypto/x509"
    "encoding/base64"
    "encoding/xml"
    "fmt"
    "strings"
)

// KeyDescriptorUse enumerates KeyDescriptor use values.
type KeyDescriptorUse string

const (
    KeyUseSigning    KeyDescriptorUse = "signing"
    KeyUseEncryption KeyDescriptorUse = "encryption"
)

// CertEntry represents a single certificate in metadata.
type CertEntry struct {
    Use        KeyDescriptorUse
    Certificate *x509.Certificate
    Base64DER  string
}

// ExtractAllCertificates extracts every KeyDescriptor certificate from metadata.
func ExtractAllCertificates(rawXML []byte) ([]CertEntry, error) {
    type keyInfo struct {
        X509Data struct {
            X509Certificate string `xml:"X509Certificate"`
        } `xml:"X509Data"`
    }
    type keyDescriptor struct {
        Use     string   `xml:"use,attr"`
        KeyInfo keyInfo  `xml:"KeyInfo"`
    }
    type spDescriptor struct {
        KeyDescriptors []keyDescriptor `xml:"KeyDescriptor"`
    }
    type entityDescriptor struct {
        SPSSODescriptor *spDescriptor `xml:"SPSSODescriptor,omitempty"`
        IDPSSODescriptor *spDescriptor `xml:"IDPSSODescriptor,omitempty"`
    }

    var ed entityDescriptor
    if err := xml.Unmarshal(rawXML, &ed); err != nil {
        return nil, fmt.Errorf("parse metadata: %w", err)
    }

    var entries []CertEntry

    collectFrom := func(desc *spDescriptor) {
        if desc == nil {
            return
        }
        for _, kd := range desc.KeyDescriptors {
            certB64 := strings.TrimSpace(kd.KeyInfo.X509Data.X509Certificate)
            if certB64 == "" {
                continue
            }
            der, err := base64.StdEncoding.DecodeString(certB64)
            if err != nil {
                continue // Skip malformed certs
            }
            cert, err := x509.ParseCertificate(der)
            if err != nil {
                continue // Skip unparseable certs
            }
            entries = append(entries, CertEntry{
                Use:         KeyDescriptorUse(kd.Use),
                Certificate: cert,
                Base64DER:   certB64,
            })
        }
    }

    collectFrom(ed.SPSSODescriptor)
    collectFrom(ed.IDPSSODescriptor)

    return entries, nil
}

// FindValidSigningCert returns a signing certificate that is currently valid
// (not expired). If multiple valid signing certs exist, the one with the
// latest NotBefore is preferred (newest key during rotation).
func FindValidSigningCert(entries []CertEntry) (*x509.Certificate, error) {
    var best *x509.Certificate
    now := time.Now()

    for _, entry := range entries {
        if entry.Use != KeyUseSigning {
            continue
        }
        cert := entry.Certificate
        if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
            continue // Skip expired or not-yet-valid certs
        }
        if best == nil || cert.NotBefore.After(best.NotBefore) {
            best = cert
        }
    }

    if best == nil {
        return nil, fmt.Errorf("no valid signing certificate found in metadata")
    }
    return best, nil
}

// CheckCertRotationStatus reports the rotation state of signing certificates.
func CheckCertRotationStatus(entries []CertEntry) (active, upcoming, expiring []*x509.Certificate) {
    now := time.Now()
    rotationWindow := 14 * 24 * time.Hour // 14-day overlap

    for _, entry := range entries {
        if entry.Use != KeyUseSigning {
            continue
        }
        cert := entry.Certificate

        if now.After(cert.NotAfter) {
            continue // Already expired
        }
        if now.Before(cert.NotBefore) {
            upcoming = append(upcoming, cert)
            continue
        }
        if cert.NotAfter.Sub(now) < rotationWindow {
            expiring = append(expiring, cert)
            continue
        }
        active = append(active, cert)
    }
    return
}
```

### 8.4 Self-Signed Certificates

SAML metadata certificates are typically self-signed — they are trust anchors, not PKI end-entity certificates. The certificate in metadata is trusted because it is in metadata signed by a trusted federation operator. The certificate's own CA chain is irrelevant. This is an important distinction from TLS certificate validation.

---

## 9. GGID SAML Metadata Audit

### 9.1 What Exists

Based on review of `pkg/saml/`, the following metadata-related capabilities are present:

**`sp.go` — SP Metadata Generation:**
- `GenerateSPMetadata()` produces a valid `EntityDescriptor` XML document with:
  - `entityID`, `validUntil` (365 days from generation)
  - `SPSSODescriptor` with `AuthnRequestsSigned`, `WantAssertionsSigned` flags
  - `AssertionConsumerService` endpoint (HTTP-POST binding)
  - `SingleLogoutService` endpoint (HTTP-Redirect binding)
  - Single `KeyDescriptor` with `use="signing"` (if certificate provided)
- `Metadata`, `SPSSODescriptor`, `KeyDescriptor`, `KeyInfoData`, `X509DataWrapper`, `ACSDescriptor`, `SLODescriptor` structs for XML serialization

**`signed_assertion.go` — Assertion Signature Verification:**
- `VerifySignedAssertion()` — verifies XML-Sig over a SAML assertion
- `parseSignature()` — extracts signature info from raw XML
- `verifyCryptoSignature()` — RSA/ECDSA signature verification
- `verifyDigest()` — digest value comparison
- `VerifySignedAssertionWithDigest()` — includes digest verification for defense in depth
- Checks that `Reference URI` matches assertion ID
- `detectDuplicateIDs` equivalent: checks `referencedID` vs assertion ID

**`assertion.go` — Assertion Parsing:**
- `ParseAssertion()`, `ValidateConditions()`, `ExtractAttributes()`, `GetAttribute()`
- `ValidateSignature()` — presence-only check (comments note production should use full XML-Sig library)

### 9.2 Security Findings

| # | Finding | Severity | Status |
|---|---|---|---|
| F-1 | **No IdP metadata parsing** — `Metadata` struct only models SP metadata. No `IDPSSODescriptor` parsing. | Medium | Gap |
| F-2 | **No metadata signature validation** — No function validates EntityDescriptor signatures. `GenerateSPMetadata` produces unsigned metadata. | High | Gap |
| F-3 | **Limited XSW protection** — `VerifySignedAssertion` checks `referencedID` vs assertion ID, but does not detect duplicate IDs or verify the application consumes the signed element. | Medium | Partial |
| F-4 | **No metadata refresh mechanism** — No cache, no periodic refresh, no `validUntil` enforcement on received metadata. | High | Gap |
| F-5 | **No federation trust chain** — No trust root management, no aggregate metadata handling. | Medium | Gap |
| F-6 | **No entity category support** — No parsing of `EntityAttributes` extensions. | Low | Gap |
| F-7 | **No encryption KeyDescriptor** — `GenerateSPMetadata` only emits `use="signing"`. Encrypted assertions not supported. | Medium | Gap |
| F-8 | **`extractSignedInfoBytes` uses string search** — Not a full XML parser, could miss edge cases in namespace handling. | Low | Known |
| F-9 | **`ValidateSignature` is presence-only** — `assertion.go` line 101 checks string contains `<ds:Signature`, does not cryptographically verify. | Medium | Known |
| F-10 | **No certificate rotation support** — Single `KeyDescriptor` emitted, no overlap period mechanism. | Low | Gap |
| F-11 | **validUntil hardcoded to 365 days** — No configurable metadata validity period. | Low | Minor |

### 9.3 Positive Findings

- **`VerifySignedAssertionWithDigest`** provides defense in depth by checking both the cryptographic signature and the digest value separately.
- **Constant-time comparison** (`constantTimeEqual`) used for digest comparison prevents timing attacks.
- **Reference URI validation** in `VerifySignedAssertion` catches the most basic signature wrapping attacks (wrong reference target).
- **Certificate validation** (`extractCertBase64`) validates the DER certificate before embedding.
- **Multiple hash algorithms** supported (`cryptoHashForSignature`): RSA-SHA1/256/384/512, ECDSA-SHA256/384/512.

---

## 10. Gap Analysis & Recommendations

### Priority Action Items

| # | Action Item | Effort | Impact |
|---|---|---|---|
| **A-1** | **Add IdP metadata parsing and signature validation** — Implement `ParseIDPMetadata()`, `ValidateMetadataSignature()` that verifies the EntityDescriptor signature covers the root element, using federation trust roots or directly trusted certificates. | 3-5 days | Critical — closes the biggest security gap |
| **A-2** | **Add metadata refresh manager with validUntil enforcement** — Implement `MetadataCache` with periodic refresh, automatic cache invalidation on `validUntil` expiry, and alerting on refresh failures. | 2-3 days | High — prevents stale trust relationships |
| **A-3** | **Strengthen XSW protection** — Add `detectDuplicateIDs()` to `VerifySignedAssertion`, verify the parsed assertion's element tree position matches the signed element. Consider adopting `github.com/russellhaering/go-saml` or `github.com/crewjam/saml` for production-grade XML-Sig. | 2-4 days | High — addresses F-3 |
| **A-4** | **Add encryption KeyDescriptor and certificate rotation support** — Emit both `use="signing"` and `use="encryption"` KeyDescriptors. Support multiple simultaneous signing certs during rotation. Add `CheckCertRotationStatus()` to alert admins of expiring certs. | 2-3 days | Medium — improves operational maturity |
| **A-5** | **Add entity category and federation trust chain support** — Parse `EntityAttributes` extensions, implement `AttributeReleasePolicy` for category-based attribute filtering, add `TrustChainValidator` with direct + federation trust modes. | 3-5 days | Medium — enables federation interoperability |

### Estimated Total Effort

- **P0 (A-1, A-2, A-3):** 7-12 engineering days — addresses critical metadata security gaps
- **P1 (A-4):** 2-3 engineering days — operational maturity
- **P2 (A-5):** 3-5 engineering days — federation readiness

### Recommended Library Strategy

GGID currently implements a custom partial XML-Sig validator. For production SAML deployments, consider adopting one of:

| Library | License | Maturity | XSW Protection |
|---|---|---|---|
| `github.com/crewjam/saml` | BSD-3 | High — used by major projects | Yes |
| `github.com/russellhaering/go-saml` | Apache-2.0 | High — dedicated XML canonicalization | Yes |
| `github.com/ucarion/saml` | MIT | Medium | Partial |

Adopting a library eliminates the maintenance burden of XML-Sig edge cases and provides battle-tested XSW countermeasures. The custom `signed_assertion.go` can be retained as a lightweight fallback for environments where external dependencies are restricted.

---

## References

- **OASIS SAML 2.0 Metadata** (sstc-saml-metadata-2.0-os): EntityDescriptor schema, KeyDescriptor semantics
- **W3C XML Signature Syntax and Processing** (xmldsig-core): Reference, SignedInfo, CanonicalizationMethod
- **W3C XML Signature Wrapping Attacks**: Somorovsky et al., "On Breaking SAML: Be Whoever You Want to Be" (USENIX Security 2012)
- **REFEDS Entity Categories**: https://refeds.org/category/research-and-scholarship
- **InCommon Federation**: Metadata aggregation and trust model practices
- **eduGAIN**: Cross-federation metadata distribution
- **NIST SP 800-63C**: Federation andAssertions guidelines

---

*Document version: 1.0 | Last reviewed: 2025-01-11 | GGID SAML package coverage: 91.1%*
