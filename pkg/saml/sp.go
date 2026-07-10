package saml

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"time"
)

// SAML protocol and metadata XML namespaces.
const (
	NSProtocol   = "urn:oasis:names:tc:SAML:2.0:protocol"
	NSAssertion  = "urn:oasis:names:tc:SAML:2.0:assertion"
	NSMetadata   = "urn:oasis:names:tc:SAML:2.0:metadata"
	NSXMLSig     = "http://www.w3.org/2000/09/xmldsig#"
)

// NameID format constants defined by SAML 2.0.
const (
	NameIDFormatUnspecified       = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
	NameIDFormatEmailAddress     = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	NameIDFormatX509SubjectName  = "urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName"
	NameIDFormatWindowsDNQ       = "urn:oasis:names:tc:SAML:1.1:nameid-format:WindowsDomainQualifiedName"
	NameIDFormatTransient        = "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"
	NameIDFormatPersistent       = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
	NameIDFormatEntity           = "urn:oasis:names:tc:SAML:2.0:nameid-format:entity"
)

// ServiceProvider describes a SAML 2.0 Service Provider configuration.
type ServiceProvider struct {
	EntityID         string // Unique SP identifier (typically the SP metadata URL)
	ACSURL           string // Assertion Consumer Service URL
	SLOURL           string // Single Logout Service URL
	X509Certificate  []byte // DER-encoded X.509 certificate for signing
	WantAssertionsSigned bool // Whether the SP requires signed assertions
}

// NameIDPolicy controls the name identifier requested from the IdP.
type NameIDPolicy struct {
	Format      string `xml:"Format,attr,omitempty"`
	SPNameQual  string `xml:"SPNameQualifier,attr,omitempty"`
	AllowCreate bool  `xml:"AllowCreate,attr"`
}

// AuthnRequest represents a SAML 2.0 authentication request (SP-initiated SSO).
type AuthnRequest struct {
	XMLName      xml.Name     `xml:"samlp:AuthnRequest"`
	ID           string       `xml:"ID,attr"`
	Version      string       `xml:"Version,attr"`
	IssueInstant string       `xml:"IssueInstant,attr"`
	Destination  string       `xml:"Destination,attr,omitempty"`
	ProtocolBinding string    `xml:"ProtocolBinding,attr,omitempty"`
	AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr,omitempty"`
	Issuer       Issuer       `xml:"saml:Issuer"`
	NameIDPolicy NameIDPolicy `xml:"samlp:NameIDPolicy,omitempty"`
	RequestedAuthnContext *RequestedAuthnContext `xml:"samlp:RequestedAuthnContext,omitempty"`
}

// Issuer is the entity that created the message.
type Issuer struct {
	Value string `xml:",chardata"`
}

// RequestedAuthnContext specifies the requested authentication context.
type RequestedAuthnContext struct {
	AuthnContextClassRef []string `xml:"saml:AuthnContextClassRef"`
}

// BuildAuthnRequest constructs a SAML 2.0 AuthnRequest for SP-initiated SSO.
// The request is suitable for HTTP-Redirect or HTTP-POST binding to the IdP.
func BuildAuthnRequest(sp *ServiceProvider, idpSSOURL string) *AuthnRequest {
	return &AuthnRequest{
		ID:                       generateID(),
		Version:                  "2.0",
		IssueInstant:             time.Now().UTC().Format(time.RFC3339),
		Destination:              idpSSOURL,
		ProtocolBinding:          "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
		AssertionConsumerServiceURL: sp.ACSURL,
		Issuer: Issuer{Value: sp.EntityID},
		NameIDPolicy: NameIDPolicy{
			Format:      NameIDFormatTransient,
			AllowCreate: true,
		},
		RequestedAuthnContext: &RequestedAuthnContext{
			AuthnContextClassRef: []string{
				"urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport",
			},
		},
	}
}

// Marshal produces the XML representation of the AuthnRequest.
func (r *AuthnRequest) Marshal() ([]byte, error) {
	data, err := xml.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal AuthnRequest: %w", err)
	}
	// Prepend XML declaration and namespace declarations.
	header := []byte(xml.Header)
	return append(header, data...), nil
}

// EncodeForRedirect produces the deflate+base64 encoded SAMLRequest value
// suitable for use in an HTTP-Redirect binding query parameter.
func (r *AuthnRequest) EncodeForRedirect() (string, error) {
	xmlBytes, err := r.Marshal()
	if err != nil {
		return "", err
	}
	compressed, err := deflate(xmlBytes)
	if err != nil {
		return "", fmt.Errorf("deflate AuthnRequest: %w", err)
	}
	return base64.StdEncoding.EncodeToString(compressed), nil
}

// ---------------------------------------------------------------------------
// SP Metadata Generation
// ---------------------------------------------------------------------------

// generateID creates a SAML-compatible unique identifier.
func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails.
		return fmt.Sprintf("_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("_%x", b)
}

// generateValidDuration returns a validity window string for metadata.
func generateValidDuration(d time.Duration) string {
	hours := int(d.Hours())
	return fmt.Sprintf("PT%dH", hours)
}

// Metadata represents the SAML SP EntityDescriptor XML structure.
type Metadata struct {
	XMLName           xml.Name        `xml:"EntityDescriptor"`
	Xmlns             string          `xml:"xmlns,attr"`
	EntityID          string          `xml:"entityID,attr"`
	ValidUntil        string          `xml:"validUntil,attr,omitempty"`
	SPSSODescriptor   *SPSSODescriptor `xml:"SPSSODescriptor,omitempty"`
}

// SPSSODescriptor contains the SP's SSO configuration.
type SPSSODescriptor struct {
	AuthnRequestsSigned     bool             `xml:"AuthnRequestsSigned,attr"`
	WantAssertionsSigned    bool             `xml:"WantAssertionsSigned,attr"`
	ProtocolSupportProtocol string           `xml:"protocolSupportEnumeration,attr"`
	KeyDescriptor           []KeyDescriptor  `xml:"KeyDescriptor"`
	AssertionConsumerService []ACSDescriptor `xml:"AssertionConsumerService"`
	SingleLogoutService     []SLODescriptor  `xml:"SingleLogoutService"`
}

// KeyDescriptor describes a key used for signing or encryption.
type KeyDescriptor struct {
	Use       string      `xml:"use,attr"`
	KeyInfo   KeyInfoData `xml:"ds:KeyInfo"`
}

// KeyInfoData wraps the X509 certificate data.
type KeyInfoData struct {
	Xmlns      string          `xml:"xmlns:ds,attr"`
	X509Data   X509DataWrapper `xml:"ds:X509Data"`
}

// X509DataWrapper holds the certificate.
type X509DataWrapper struct {
	X509Certificate string `xml:"ds:X509Certificate"`
}

// ACSDescriptor describes an Assertion Consumer Service endpoint.
type ACSDescriptor struct {
	Index         int    `xml:"index,attr"`
	Binding       string `xml:"Binding,attr"`
	Location      string `xml:"Location,attr"`
	IsDefault     bool   `xml:"isDefault,attr,omitempty"`
}

// SLODescriptor describes a Single Logout Service endpoint.
type SLODescriptor struct {
	Binding  string `xml:"Binding,attr"`
	Location string `xml:"Location,attr"`
}

// GenerateSPMetadata produces a SAML 2.0 SP metadata EntityDescriptor XML document.
// The metadata includes the SP's entity ID, ACS endpoint, SLO endpoint,
// and signing certificate (if provided).
func GenerateSPMetadata(sp *ServiceProvider) ([]byte, error) {
	if sp == nil {
		return nil, fmt.Errorf("service provider is nil")
	}
	if sp.EntityID == "" {
		return nil, fmt.Errorf("entity ID is required")
	}
	if sp.ACSURL == "" {
		return nil, fmt.Errorf("ACS URL is required")
	}

	meta := &Metadata{
		Xmlns:      NSMetadata,
		EntityID:   sp.EntityID,
		ValidUntil: time.Now().UTC().Add(365 * 24 * time.Hour).Format(time.RFC3339),
		SPSSODescriptor: &SPSSODescriptor{
			AuthnRequestsSigned:     sp.X509Certificate != nil,
			WantAssertionsSigned:    sp.WantAssertionsSigned,
			ProtocolSupportProtocol: NSProtocol,
			AssertionConsumerService: []ACSDescriptor{
				{
					Index:     0,
					Binding:   "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST",
					Location:  sp.ACSURL,
					IsDefault: true,
				},
			},
			SingleLogoutService: []SLODescriptor{},
		},
	}

	// Add SLO endpoint if configured.
	if sp.SLOURL != "" {
		meta.SPSSODescriptor.SingleLogoutService = append(meta.SPSSODescriptor.SingleLogoutService,
			SLODescriptor{
				Binding:  "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect",
				Location: sp.SLOURL,
			},
		)
	}

	// Add signing certificate if provided.
	if len(sp.X509Certificate) > 0 {
		certB64, err := extractCertBase64(sp.X509Certificate)
		if err != nil {
			return nil, fmt.Errorf("extract certificate: %w", err)
		}
		meta.SPSSODescriptor.KeyDescriptor = []KeyDescriptor{
			{
				Use: "signing",
				KeyInfo: KeyInfoData{
					Xmlns: NSXMLSig,
					X509Data: X509DataWrapper{
						X509Certificate: certB64,
					},
				},
			},
		}
	}

	data, err := xml.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal SP metadata: %w", err)
	}
	return append([]byte(xml.Header), data...), nil
}

// extractCertBase64 accepts either a DER-encoded certificate or a parsed
// *x509.Certificate and returns the base64-encoded DER for XML embedding.
func extractCertBase64(derBytes []byte) (string, error) {
	// Validate it's a parseable DER certificate.
	_, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return "", fmt.Errorf("invalid DER certificate: %w", err)
	}
	return base64.StdEncoding.EncodeToString(derBytes), nil
}

// deflate is a placeholder hook; actual compression for HTTP-Redirect uses
// raw DEFLATE (RFC 1951) without zlib header. We use compress/flate internally.
// We declare it here to keep the API clean.
func deflate(data []byte) ([]byte, error) {
	return flateCompress(data)
}
