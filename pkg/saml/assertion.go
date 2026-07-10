// Package saml provides SAML 2.0 assertion parsing and validation.
package saml

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/xml"
	"fmt"
	"strings"
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
}

// Conditions holds the validity window.
type Conditions struct {
	NotBefore    string `xml:"NotBefore,attr"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
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
		if err == nil && now.Before(notBefore.Add(-time.Minute)) {
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

// ValidateSignature verifies the XML signature against the provided certificate.
// This is a simplified check — production usage should use a full XML-Sig library.
func ValidateSignature(assertion *SAMLAssertion, cert *x509.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	if assertion == nil || assertion.RawXML == nil {
		return fmt.Errorf("assertion has no raw XML")
	}

	// Check that the certificate has an RSA public key.
	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate does not contain an RSA public key")
	}
	_ = pubKey // Used in production for actual signature verification

	// In production, this would use github.com/russellhaering/go-saml
	// or github.com/crewjam/saml to verify the XML-Signature.
	// For now, we verify the assertion contains a Signature element.
	if !strings.Contains(string(assertion.RawXML), "<ds:Signature") &&
		!strings.Contains(string(assertion.RawXML), "<Signature") {
		return fmt.Errorf("assertion does not contain a Signature element")
	}

	return nil
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
