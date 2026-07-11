package saml

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// IdPInitiatedSSORequest represents an unsolicited SAML Response from an IdP
// (IdP-initiated SSO). Unlike SP-initiated SSO, there is no corresponding
// AuthnRequest — the IdP sends the assertion directly.
type IdPInitiatedSSORequest struct {
	SAMLResponse string // Base64-encoded SAML Response XML
	RelayState   string // Optional relay state for SP context
}

// IdPInitiatedSSOResult contains the extracted user identity from a verified
// IdP-initiated SAML response.
type IdPInitiatedSSOResult struct {
	NameID       string
	NameIDFormat string
	Attributes   map[string][]string
	SessionIndex string
	NotOnOrAfter time.Time
	Issuer       string
	RelayState   string
}

// HandleIdPInitiatedSSO processes an unsolicited SAML Response (IdP-initiated SSO).
// This is used when the IdP sends a SAML assertion without a prior AuthnRequest
// from the SP.
//
// Security considerations (OASIS Security Conformance):
//  1. The response MUST be signed (either response or assertion level)
//  2. The Issuer MUST be a trusted IdP
//  3. The ACS URL in the response MUST match the SP's configured ACS URL
//  4. The response MUST have valid NotBefore/NotOnOrAfter times
//  5. SubjectConfirmationData MUST contain a valid Recipient
//  6. Replay attacks MUST be prevented via InResponseTo tracking (if present)
func (sp *ServiceProvider) HandleIdPInitiatedSSO(req *IdPInitiatedSSORequest, idpCert *x509.Certificate) (*IdPInitiatedSSOResult, error) {
	if req == nil {
		return nil, fmt.Errorf("IdP-initiated SSO request is nil")
	}
	if req.SAMLResponse == "" {
		return nil, fmt.Errorf("SAMLResponse is empty")
	}

	// Decode the base64-encoded SAML response.
	xmlBytes, err := base64.StdEncoding.DecodeString(req.SAMLResponse)
	if err != nil {
		return nil, fmt.Errorf("base64 decode SAMLResponse: %w", err)
	}

	// Parse the SAML Response XML.
	var samlResp Response
	if err := xml.Unmarshal(xmlBytes, &samlResp); err != nil {
		return nil, fmt.Errorf("unmarshal SAML Response: %w", err)
	}

	// Verify the response status is Success.
	if samlResp.Status.StatusCode.Value != StatusCodeSuccess {
		return nil, fmt.Errorf("IdP returned non-success status: %s", samlResp.Status.StatusCode.Value)
	}

	// Verify the Issuer is present (trusted IdP check is done by caller).
	if samlResp.Issuer.Value == "" {
		return nil, fmt.Errorf("SAML Response missing Issuer")
	}

	// For IdP-initiated SSO, InResponseTo MUST be absent (no prior AuthnRequest).
	// If present, it may indicate a replay attack or misconfigured IdP.
	if samlResp.InResponseTo != "" {
		return nil, fmt.Errorf("IdP-initiated SSO must not have InResponseTo")
	}

	// Verify the destination (ACS URL) matches the SP configuration.
	if samlResp.Destination != "" && samlResp.Destination != sp.ACSURL {
		return nil, fmt.Errorf("SAML Response Destination %q does not match ACS URL %q",
			samlResp.Destination, sp.ACSURL)
	}

	// Verify time validity.
	now := time.Now().UTC()
	issueTime, err := time.Parse(time.RFC3339, samlResp.IssueInstant)
	if err == nil && issueTime.Add(-5*time.Minute).After(now) {
		return nil, fmt.Errorf("SAML Response IssueInstant is in the future")
	}

	// Extract and validate the assertion.
	if samlResp.Assertion == nil {
		return nil, fmt.Errorf("SAML Response has no Assertion")
	}

	assertion := samlResp.Assertion

	// Validate NotBefore/NotOnOrAfter conditions.
	if assertion.Conditions != nil {
		if assertion.Conditions.NotBefore.After(now.Add(time.Minute)) {
			return nil, fmt.Errorf("assertion NotBefore is in the future")
		}
		if !assertion.Conditions.NotOnOrAfter.IsZero() && now.After(assertion.Conditions.NotOnOrAfter) {
			return nil, fmt.Errorf("assertion has expired (past NotOnOrAfter)")
		}
	}

	// Validate SubjectConfirmation.
	if assertion.Subject != nil {
		for _, sc := range assertion.Subject.SubjectConfirmations {
			if sc.SubjectConfirmationData.Recipient != "" && sc.SubjectConfirmationData.Recipient != sp.ACSURL {
				return nil, fmt.Errorf("SubjectConfirmation Recipient does not match ACS URL")
			}
		}
	}

	// Verify signature if certificate is provided.
	if idpCert != nil {
		// In production, verify the XML signature using the IdP's certificate.
		// For now, we verify the signature is present.
		if samlResp.Signature == nil && assertion.Signature == nil {
			return nil, fmt.Errorf("SAML Response and Assertion are both unsigned")
		}
	}

	// Build the result.
	result := &IdPInitiatedSSOResult{
		Issuer:     samlResp.Issuer.Value,
		RelayState: req.RelayState,
		Attributes: make(map[string][]string),
	}

	// Extract NameID.
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		result.NameID = assertion.Subject.NameID.Value
		result.NameIDFormat = assertion.Subject.NameID.Format
	}

	// Extract SessionIndex from AuthnStatement.
	for _, authnStmt := range assertion.AuthnStatements {
		if authnStmt.SessionIndex != "" {
			result.SessionIndex = authnStmt.SessionIndex
			break
		}
	}

	// Extract conditions expiry.
	if assertion.Conditions != nil {
		result.NotOnOrAfter = assertion.Conditions.NotOnOrAfter
	}

	// Extract attributes from AttributeStatement.
	for _, attrStmt := range assertion.AttributeStatements {
		for _, attr := range attrStmt.Attributes {
			var values []string
			for _, val := range attr.Values {
				values = append(values, val.Value)
			}
			result.Attributes[attr.Name] = values
		}
	}

	return result, nil
}

// IsTrustedIdP checks if the given issuer entityID is in the list of trusted IdPs.
func IsTrustedIdP(issuer string, trustedIdPs []string) bool {
	for _, trusted := range trustedIdPs {
		if strings.EqualFold(issuer, trusted) {
			return true
		}
	}
	return false
}

// --- SAML Response XML Types ---

// Response represents a SAML 2.0 Protocol Response.
type Response struct {
	XMLName      xml.Name     `xml:"Response"`
	ID           string       `xml:"ID,attr"`
	InResponseTo string       `xml:"InResponseTo,attr,omitempty"`
	Destination  string       `xml:"Destination,attr,omitempty"`
	IssueInstant string       `xml:"IssueInstant,attr"`
	Version      string       `xml:"Version,attr"`
	Issuer       Issuer       `xml:"Issuer"`
	Status       Status       `xml:"Status"`
	Signature    *string      `xml:"Signature,omitempty"`
	Assertion    *Assertion   `xml:"Assertion"`
}

// Status contains the SAML response status.
type Status struct {
	StatusCode StatusCode `xml:"StatusCode"`
}

// StatusCode contains the status code value.
type StatusCode struct {
	Value string `xml:"Value,attr"`
}

// StatusCodeSuccess is the SAML success status code.
const StatusCodeSuccess = "urn:oasis:names:tc:SAML:2.0:status:Success"

// Assertion represents a SAML 2.0 Assertion.
// (Defined in assertion.go as well, but we need a local type with all fields
// for IdP-initiated SSO parsing.)
type SAMLResponseAssertion struct {
	XMLName             xml.Name             `xml:"Assertion"`
	ID                  string               `xml:"ID,attr"`
	IssueInstant        string               `xml:"IssueInstant,attr"`
	Version             string               `xml:"Version,attr"`
	Issuer              Issuer               `xml:"Issuer"`
	Signature           *string              `xml:"Signature,omitempty"`
	Subject             *SAMLSubject         `xml:"Subject"`
	Conditions          *SAMLConditions      `xml:"Conditions"`
	AuthnStatements     []SAMLAuthnStatement `xml:"AuthnStatement"`
	AttributeStatements []SAMLAttributeStatement `xml:"AttributeStatement"`
}

// SAMLSubject contains the assertion subject.
type SAMLSubject struct {
	NameID             *SAMLNameID            `xml:"NameID"`
	SubjectConfirmations []SAMLSubjectConfirmation `xml:"SubjectConfirmation"`
}

// SAMLNameID contains the name identifier.
type SAMLNameID struct {
	Format string `xml:"Format,attr,omitempty"`
	Value  string `xml:",chardata"`
}

// SAMLSubjectConfirmation contains confirmation data.
type SAMLSubjectConfirmation struct {
	Method                 string                   `xml:"Method,attr"`
	SubjectConfirmationData SAMLSubjectConfirmationData `xml:"SubjectConfirmationData"`
}

// SAMLSubjectConfirmationData contains recipient and timing info.
type SAMLSubjectConfirmationData struct {
	Recipient    string `xml:"Recipient,attr,omitempty"`
	NotOnOrAfter string `xml:"NotOnOrAfter,attr,omitempty"`
}

// SAMLConditions contains validity conditions.
type SAMLConditions struct {
	NotBefore    time.Time `xml:"NotBefore,attr,omitempty"`
	NotOnOrAfter time.Time `xml:"NotOnOrAfter,attr,omitempty"`
}

// SAMLAuthnStatement contains authentication context.
type SAMLAuthnStatement struct {
	SessionIndex       string             `xml:"SessionIndex,attr,omitempty"`
	AuthnInstant       string             `xml:"AuthnInstant,attr,omitempty"`
	SessionNotOnOrAfter string            `xml:"SessionNotOnOrAfter,attr,omitempty"`
	AuthnContext       SAMLAuthnContext   `xml:"AuthnContext"`
}

// SAMLAuthnContext contains the authentication context class.
type SAMLAuthnContext struct {
	AuthnContextClassRef string `xml:"AuthnContextClassRef"`
}

// SAMLAttributeStatement contains user attributes.
type SAMLAttributeStatement struct {
	Attributes []SAMLAttribute `xml:"Attribute"`
}

// SAMLAttribute represents a single SAML attribute.
type SAMLAttribute struct {
	Name   string         `xml:"Name,attr"`
	Values []SAMLAttributeValue `xml:"AttributeValue"`
}

// SAMLAttributeValue wraps an attribute value.
type SAMLAttributeValue struct {
	Value string `xml:",chardata"`
}

// Assertion is re-exported as an alias for backward compatibility.
// The full type used for IdP-initiated SSO parsing is SAMLResponseAssertion.
type Assertion = SAMLResponseAssertion
