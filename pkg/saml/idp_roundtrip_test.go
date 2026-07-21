package saml

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

// extractAssertionFromResponse extracts the <Assertion> element from a
// <samlp:Response> XML document and returns it as standalone XML.
func extractAssertionFromResponse(t *testing.T, responseXML []byte) []byte {
	t.Helper()
	// We need the raw bytes, so find <Assertion> in the XML string
	s := string(responseXML)
	startTag := "<Assertion"
	endTag := "</Assertion>"
	startIdx := strings.Index(s, startTag)
	if startIdx < 0 {
		// Try with namespace prefix
		startTag = "<saml:Assertion"
		endTag = "</saml:Assertion>"
		startIdx = strings.Index(s, startTag)
	}
	if startIdx < 0 {
		t.Fatal("no <Assertion> element found in Response XML")
	}
	endIdx := strings.Index(s[startIdx:], endTag)
	if endIdx < 0 {
		t.Fatal("no closing </Assertion> found")
	}
	endIdx += startIdx + len(endTag)
	// Return the original assertion bytes unchanged. ParseAssertion matches
	// XML local names regardless of namespace prefix, and signature
	// verification requires the exact signed bytes (stripping prefixes
	// would break the digest).
	return responseXML[startIdx:endIdx]
}

var _ = xml.Marshal // keep import for potential future use

func TestIdP_SP_RoundTrip_BasicAttributes(t *testing.T) {
	// Generate RSA key + cert for IdP
	cert, privKey := genRSACertWithKey(t)

	idp := &IdentityProvider{
		EntityID:    "https://idp.example.com/saml",
		SSOURL:      "https://idp.example.com/sso",
		SLOURL:      "https://idp.example.com/slo",
		PrivateKey:  privKey,
		Certificate: cert.Raw,
		KeyID:       "test-key-id",
	}

	req := &SAMLResponseRequest{
		Destination:  "https://sp.example.com/acs",
		Audience:     "https://sp.example.com/saml",
		NameID:       "user@example.com",
		NameIDFormat: NameIDFormatEmailAddress,
		Attributes: map[string][]string{
			"email":        {"user@example.com"},
			"displayName":  {"Test User"},
			"role":         {"admin", "editor"},
		},
	}

	// Step 1: IdP builds signed SAML Response
	responseXML, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}
	if len(responseXML) == 0 {
		t.Fatal("empty response XML")
	}

	// Verify it's a SAML Response
	if !strings.Contains(string(responseXML), "samlp:Response") {
		t.Error("response XML does not contain samlp:Response")
	}

	// Step 2: Extract the signed Assertion from the Response
	assertionXML := extractAssertionFromResponse(t, responseXML)
	if len(assertionXML) == 0 {
		t.Fatal("extracted assertion XML is empty")
	}

	// Step 3: SP verifies the signed assertion
	assertion, err := VerifySignedAssertion(assertionXML, cert)
	if err != nil {
		t.Fatalf("VerifySignedAssertion: %v", err)
	}

	// Step 4: Verify NameID
	if assertion.Subject.NameID != "user@example.com" {
		t.Errorf("NameID = %s, want user@example.com", assertion.Subject.NameID)
	}

	// Step 5: Verify attributes
	attrs := ExtractAttributes(assertion)
	if email, ok := attrs["email"]; !ok || len(email) == 0 || email[0] != "user@example.com" {
		t.Errorf("email attribute = %v, want [user@example.com]", attrs["email"])
	}
	if name, ok := attrs["displayName"]; !ok || len(name) == 0 || name[0] != "Test User" {
		t.Errorf("displayName attribute = %v, want [Test User]", attrs["displayName"])
	}
	if roles, ok := attrs["role"]; !ok || len(roles) != 2 {
		t.Errorf("role attribute = %v, want 2 values", attrs["role"])
	} else {
		if roles[0] != "admin" || roles[1] != "editor" {
			t.Errorf("roles = %v, want [admin editor]", roles)
		}
	}
}

func TestIdP_SP_RoundTrip_UnsignedAssertionFails(t *testing.T) {
	// An unsigned assertion should fail verification
	cert, _ := genRSACertWithKey(t)

	unsignedXML := []byte(`<Assertion ID="abc" Version="2.0" IssueInstant="2025-01-01T00:00:00Z">
		<Issuer>https://idp.example.com</Issuer>
		<Subject><NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">user@test.com</NameID></Subject>
		<Conditions NotBefore="2025-01-01T00:00:00Z" NotOnOrAfter="2026-01-01T00:00:00Z"/>
	</Assertion>`)

	_, err := VerifySignedAssertion(unsignedXML, cert)
	if err == nil {
		t.Error("expected error for unsigned assertion, got nil")
	}
}

func TestIdP_SP_RoundTrip_WrongCertFails(t *testing.T) {
	// Sign with one key, verify with a different cert
	cert1, privKey := genRSACertWithKey(t)
	cert2, _ := genRSACertWithKey(t) // different key pair

	idp := &IdentityProvider{
		EntityID:    "https://idp.example.com/saml",
		PrivateKey:  privKey,
		Certificate: cert1.Raw,
		KeyID:       "key1",
	}

	req := &SAMLResponseRequest{
		Destination:  "https://sp.example.com/acs",
		Audience:     "https://sp.example.com",
		NameID:       "user@example.com",
	}

	responseXML, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	assertionXML := extractAssertionFromResponse(t, responseXML)

	// Verify with wrong cert — should fail
	_, err = VerifySignedAssertion(assertionXML, cert2)
	if err == nil {
		t.Error("expected signature verification failure with wrong cert, got nil")
	}
}

func TestIdP_SP_RoundTrip_MinimalRequest(t *testing.T) {
	cert, privKey := genRSACertWithKey(t)

	idp := &IdentityProvider{
		EntityID:    "https://idp.example.com",
		PrivateKey:  privKey,
		Certificate: cert.Raw,
		KeyID:       "minimal-key",
	}

	// Minimal request — only required fields
	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		NameID:      "minimal@test.com",
	}

	responseXML, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse minimal: %v", err)
	}

	assertionXML := extractAssertionFromResponse(t, responseXML)

	assertion, err := VerifySignedAssertion(assertionXML, cert)
	if err != nil {
		t.Fatalf("VerifySignedAssertion minimal: %v", err)
	}

	if assertion.Subject.NameID != "minimal@test.com" {
		t.Errorf("NameID = %s, want minimal@test.com", assertion.Subject.NameID)
	}

	// Default NameIDFormat should be emailAddress
	// (already tested in idp_test.go)
}

func TestIdP_SP_RoundTrip_WithAttributes(t *testing.T) {
	cert, privKey := genRSACertWithKey(t)

	idp := &IdentityProvider{
		EntityID:    "https://idp.example.com",
		PrivateKey:  privKey,
		Certificate: cert.Raw,
		KeyID:       "attr-key",
	}

	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		NameID:      "attr@test.com",
		Attributes: map[string][]string{
			"groups":   {"engineering", "security"},
			"department": {"IT"},
			"manager":  {"boss@example.com"},
		},
		InResponseTo: "req-123",
		RelayState:   "state-abc",
		NotBefore:    time.Now().Add(-5 * time.Minute),
		NotOnOrAfter: time.Now().Add(10 * time.Minute),
	}

	responseXML, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	assertionXML := extractAssertionFromResponse(t, responseXML)

	assertion, err := VerifySignedAssertion(assertionXML, cert)
	if err != nil {
		t.Fatalf("VerifySignedAssertion: %v", err)
	}

	attrs := ExtractAttributes(assertion)
	if groups, ok := attrs["groups"]; !ok || len(groups) != 2 {
		t.Errorf("groups = %v, want 2 values", attrs["groups"])
	}
	if dept, ok := attrs["department"]; !ok || len(dept) != 1 || dept[0] != "IT" {
		t.Errorf("department = %v, want [IT]", attrs["department"])
	}
	if mgr, ok := attrs["manager"]; !ok || len(mgr) != 1 || mgr[0] != "boss@example.com" {
		t.Errorf("manager = %v, want [boss@example.com]", attrs["manager"])
	}
}
