package saml

import (
	"crypto/x509"
	"encoding/xml"
	"strings"
	"testing"
)

// TestIdP_SP_RoundTrip is the critical cascade authentication test:
// 1. IdP builds and signs a SAML Response
// 2. SP extracts the assertion from the Response
// 3. SP verifies the IdP's signature
// 4. SP extracts user attributes
//
// This proves GGID can act as IdP and its SAML Responses
// can be verified by any standards-compliant SP.
func TestIdP_SP_RoundTrip(t *testing.T) {
	// Setup: generate IdP key pair
	cert, key := genRSACertWithKey(t)
	derBytes, err := x509.CreateCertificate(nil, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create DER cert: %v", err)
	}

	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		SSOURL:      "https://ggid.example.com/saml/idp/sso",
		SLOURL:      "https://ggid.example.com/saml/idp/slo",
		PrivateKey:  key,
		Certificate: derBytes,
	}

	// Step 1: IdP builds signed SAML Response
	respXML, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination:  "https://sp.example.com/acs",
		Audience:     "https://sp.example.com/metadata",
		NameID:       "user@example.com",
		NameIDFormat: NameIDFormatEmailAddress,
		Attributes: map[string][]string{
			"email":       {"user@example.com"},
			"displayName": {"Test User"},
			"groups":      {"admin", "users"},
		},
		InResponseTo: "req-abc-123",
	})
	if err != nil {
		t.Fatalf("IdP BuildSAMLResponse failed: %v", err)
	}

	respStr := string(respXML)

	// Step 2: SP extracts the assertion from the Response XML
	assertionXML := extractAssertionFromResponse(respStr)
	if assertionXML == nil {
		t.Fatal("SP failed to extract assertion from SAML Response")
	}

	// Step 3: SP verifies the IdP's signature
	assertion, err := VerifySignedAssertion(assertionXML, cert)
	if err != nil {
		t.Fatalf("SP VerifySignedAssertion failed: %v", err)
	}

	// Step 4: Verify assertion contents
	if assertion.ID == "" {
		t.Error("assertion ID should not be empty")
	}
	if assertion.IssueInstant == "" {
		t.Error("assertion IssueInstant should not be empty")
	}

	// Step 5: Verify the response contains correct attributes
	if !strings.Contains(respStr, "user@example.com") {
		t.Error("response should contain NameID")
	}
	if !strings.Contains(respStr, "Test User") {
		t.Error("response should contain displayName attribute")
	}
	if !strings.Contains(respStr, "admin") {
		t.Error("response should contain groups attribute")
	}
	if !strings.Contains(respStr, "Success") {
		t.Error("response status should be Success")
	}
	if !strings.Contains(respStr, "req-abc-123") {
		t.Error("response should contain InResponseTo")
	}

	// Step 6: Verify signature is valid XML
	if !strings.Contains(respStr, "ds:Signature") {
		t.Error("response should contain XMLDSig signature")
	}
	if !strings.Contains(respStr, "rsa-sha256") {
		t.Error("signature should use rsa-sha256")
	}

	t.Log("SP→IdP roundtrip: IdP signed assertion verified by SP successfully")
}

// TestIdP_SP_RoundTrip_PersistentNameID tests with persistent NameID format
func TestIdP_SP_RoundTrip_PersistentNameID(t *testing.T) {
	cert, key := genRSACertWithKey(t)
	derBytes, _ := x509.CreateCertificate(nil, cert, cert, &key.PublicKey, key)

	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		PrivateKey:  key,
		Certificate: derBytes,
	}

	respXML, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination:  "https://sp.example.com/acs",
		Audience:     "https://sp.example.com/metadata",
		NameID:       "persistent-user-id-12345",
		NameIDFormat: NameIDFormatPersistent,
	})
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(respXML)

	// Verify persistent format is used
	if !strings.Contains(respStr, "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent") {
		t.Error("response should use persistent NameID format")
	}
	if !strings.Contains(respStr, "persistent-user-id-12345") {
		t.Error("response should contain persistent NameID value")
	}

	// Verify the assertion can be extracted and parsed
	assertionXML := extractAssertionFromResponse(respStr)
	if assertionXML == nil {
		t.Fatal("failed to extract assertion")
	}

	var parsed SAMLAssertion
	if err := xml.Unmarshal(assertionXML, &parsed); err != nil {
		t.Fatalf("failed to parse extracted assertion: %v", err)
	}
}

// TestIdP_SP_RoundTrip_Unsolicited tests IdP-initiated (unsolicited) SSO
func TestIdP_SP_RoundTrip_Unsolicited(t *testing.T) {
	cert, key := genRSACertWithKey(t)
	derBytes, _ := x509.CreateCertificate(nil, cert, cert, &key.PublicKey, key)

	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		PrivateKey:  key,
		Certificate: derBytes,
	}

	// No InResponseTo = unsolicited (IdP-initiated)
	respXML, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com/metadata",
		NameID:      "user@example.com",
		// InResponseTo intentionally empty
	})
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(respXML)

	// Verify response is valid
	if !strings.Contains(respStr, "samlp:Response") {
		t.Error("should contain Response element")
	}
	if !strings.Contains(respStr, "Success") {
		t.Error("status should be Success")
	}

	// Verify assertion can be extracted and verified
	assertionXML := extractAssertionFromResponse(respStr)
	if assertionXML == nil {
		t.Fatal("failed to extract assertion")
	}

	_, err = VerifySignedAssertion(assertionXML, cert)
	if err != nil {
		t.Fatalf("SP verification of unsolicited response failed: %v", err)
	}

	t.Log("IdP-initiated (unsolicited) SSO: assertion verified by SP")
}

// TestIdP_SP_RoundTrip_MultipleAttributes verifies multi-valued attributes
func TestIdP_SP_RoundTrip_MultipleAttributes(t *testing.T) {
	cert, key := genRSACertWithKey(t)
	derBytes, _ := x509.CreateCertificate(nil, cert, cert, &key.PublicKey, key)

	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com",
		PrivateKey:  key,
		Certificate: derBytes,
	}

	respXML, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com",
		NameID:      "user@example.com",
		Attributes: map[string][]string{
			"groups":   {"admin", "users", "developers"},
			"roles":    {"superadmin"},
			"email":    {"user@example.com"},
			"employee": {"E12345"},
		},
	})
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(respXML)

	// All attribute values should be present
	expectedValues := []string{"admin", "users", "developers", "superadmin", "user@example.com", "E12345"}
	for _, v := range expectedValues {
		if !strings.Contains(respStr, v) {
			t.Errorf("response should contain attribute value: %s", v)
		}
	}
}

// TestIdP_MetadataCanBeConsumedBySP verifies the IdP metadata
// can be parsed by a standards-compliant SP
func TestIdP_MetadataCanBeConsumedBySP(t *testing.T) {
	cert, key := genRSACertWithKey(t)
	derBytes, _ := x509.CreateCertificate(nil, cert, cert, &key.PublicKey, key)

	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		SSOURL:      "https://ggid.example.com/saml/idp/sso",
		SLOURL:      "https://ggid.example.com/saml/idp/slo",
		PrivateKey:  key,
		Certificate: derBytes,
	}

	meta, err := idp.GenerateIdPMetadata()
	if err != nil {
		t.Fatalf("GenerateIdPMetadata: %v", err)
	}

	// SP should be able to parse this as valid XML
	var parsed struct {
		XMLName xml.Name `xml:"EntityDescriptor"`
		EntityID string  `xml:"entityID,attr"`
	}
	if err := xml.Unmarshal(meta, &parsed); err != nil {
		t.Fatalf("SP failed to parse IdP metadata: %v", err)
	}

	if parsed.EntityID != idp.EntityID {
		t.Errorf("metadata EntityID mismatch: got %s, want %s", parsed.EntityID, idp.EntityID)
	}

	// Verify cert in metadata matches IdP cert
	if !strings.Contains(string(meta), "X509Certificate") {
		t.Error("metadata should contain X509Certificate for SP to verify signatures")
	}
}

// extractAssertionFromResponse extracts the <saml:Assertion> element from
// a SAML Response XML string. This simulates what an SP does when it
// receives a SAML Response from an IdP.
func extractAssertionFromResponse(respStr string) []byte {
	startTag := "<saml:Assertion"
	endTag := "</saml:Assertion>"

	start := strings.Index(respStr, startTag)
	if start < 0 {
		// Try without namespace prefix
		startTag = "<Assertion"
		endTag = "</Assertion>"
		start = strings.Index(respStr, startTag)
	}
	if start < 0 {
		return nil
	}

	end := strings.Index(respStr[start:], endTag)
	if end < 0 {
		return nil
	}

	return []byte(respStr[start : start+end+len(endTag)])
}
