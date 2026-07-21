// Package saml provides SAML 2.0 assertion parsing and validation.
package saml

import (
	"crypto/x509"
	"encoding/xml"
	"fmt"
	"time"
)

// SAMLAssertion represents a parsed SAML 2.0 assertion.
type SAMLAssertion struct {
	XMLName        xml.Name `xml:"Assertion"`
	ID             string   `xml:"ID,attr"`
	IssueInstant   string   `xml:"IssueInstant,attr"`
	Version        string   `xml:"Version,attr"`
	Issuer         string   `xml:"Issuer"`
	Subject        Subject  `xml:"Subject"`
	Conditions     Conditions `xml:"Conditions"`
	AttributeStatement AttributeStatement `xml:"AttributeStatement"`
	// Raw XML for signature verification.
	RawXML []byte `xml:"-"`
}

// Subject holds the assertion subject (name identifier).
type Subject struct {
	NameID string `xml:"NameID"`
	SubjectConfirmation SubjectConfirmation `xml:"SubjectConfirmation"`
}

// SubjectConfirmation describes how the relying party may confirm the subject.
type SubjectConfirmation struct {
	Method string `xml:"Method,attr"`
	SubjectConfirmationData SubjectConfirmationData `xml:"SubjectConfirmationData"`
}

// SubjectConfirmationData carries bearer-confirmation constraints.
type SubjectConfirmationData struct {
	InResponseTo string `xml:"InResponseTo,attr"`
	Recipient    string `xml:"Recipient,attr"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
}

// Conditions holds the validity window.
type Conditions struct {
	NotBefore           string              `xml:"NotBefore,attr"`
	NotOnOrAfter        string              `xml:"NotOnOrAfter,attr"`
	AudienceRestriction AudienceRestriction `xml:"AudienceRestriction"`
}

// AudienceRestriction enforces that the assertion is intended for a specific SP.
type AudienceRestriction struct {
	Audience string `xml:"Audience"`
}

// AttributeStatement holds the collection of SAML attributes.
type AttributeStatement struct {
	Attributes []Attribute `xml:"Attribute"`
}

// Attribute represents a single SAML attribute.
type Attribute struct {
	Name       string   `xml:"Name,attr"`
	NameFormat string   `xml:"NameFormat,attr"`
	Values    []string `xml:"AttributeValue"`
}

// ParseAssertion parses a raw SAML 2.0 assertion XML document.
func ParseAssertion(rawXML []byte) (*SAMLAssertion, error) {
	var assertion SAMLAssertion
	if err := xml.Unmarshal(rawXML, &assertion); err != nil {
		return nil, fmt.Errorf("parse SAML assertion XML: %w", err)
	}
	assertion.RawXML = rawXML
	return &assertion, nil
}

// ValidateConditions checks that the assertion is within its validity window.
func (a *SAMLAssertion) ValidateConditions() error {
	now := time.Now().UTC()

	if a.Conditions.NotBefore != "" {
		notBefore, err := time.Parse(time.RFC3339, a.Conditions.NotBefore)
		if err == nil && now.Before(notBefore.Add(-2 * time.Minute)) {
			return fmt.Errorf("assertion not yet valid (NotBefore=%s)", a.Conditions.NotBefore)
		}
	}

	if a.Conditions.NotOnOrAfter != "" {
		notOnOrAfter, err := time.Parse(time.RFC3339, a.Conditions.NotOnOrAfter)
		if err == nil && !now.Before(notOnOrAfter) {
			return fmt.Errorf("assertion expired (NotOnOrAfter=%s)", a.Conditions.NotOnOrAfter)
		}
	}

	return nil
}

// ValidateSignature performs real XML-DSig verification of the assertion's
// signature against the provided certificate. It extracts the <ds:Signature>
// element, checks the reference URI matches the assertion ID, and verifies
// the RSA/ECDSA signature over the SignedInfo element. Assertions without a
// Signature element, or with a forged/invalid signature, are rejected.
func ValidateSignature(assertion *SAMLAssertion, cert *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	if assertion == nil || assertion.RawXML == nil {
		return fmt.Errorf("assertion has no raw XML")
	}

	info, err := parseSignature(assertion.RawXML)
	if err != nil {
		return fmt.Errorf("extract signature: %w", err)
	}

	if info.referencedID != "" && assertion.ID != "" && info.referencedID != "#"+assertion.ID {
		return fmt.Errorf("signature reference URI %q does not match assertion ID %q",
			info.referencedID, assertion.ID)
	}

	if err := verifyCryptoSignature(info, cert); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// InResponseTo returns the ID of the AuthnRequest this assertion answers,
// or empty for IdP-initiated (unsolicited) assertions.
func (a *SAMLAssertion) InResponseTo() string {
	return a.Subject.SubjectConfirmation.SubjectConfirmationData.InResponseTo
}

// ExtractAttributes converts the assertion's attributes into a map of string slices.
// Multi-valued attributes are returned as []string.
func ExtractAttributes(assertion *SAMLAssertion) map[string][]string {
	attrs := make(map[string][]string)
	for _, attr := range assertion.AttributeStatement.Attributes {
		attrs[attr.Name] = attr.Values
	}
	return attrs
}

// GetAttribute returns the first value of a named attribute, or empty string.
func GetAttribute(assertion *SAMLAssertion, name string) string {
	for _, attr := range assertion.AttributeStatement.Attributes {
		if attr.Name == name && len(attr.Values) > 0 {
			return attr.Values[0]
		}
	}
	return ""
}
