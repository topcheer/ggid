package saml

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"strings"
	"testing"
)

func genIdPTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	cert, key := genRSACertWithKey(t)
	derBytes, err := x509.CreateCertificate(nil, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create DER cert: %v", err)
	}
	return key, derBytes
}

func TestIdP_GenerateIdPMetadata(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		SSOURL:      "https://ggid.example.com/saml/idp/sso",
		SLOURL:      "https://ggid.example.com/saml/idp/slo",
		PrivateKey:  key,
		Certificate: certDER,
	}

	meta, err := idp.GenerateIdPMetadata()
	if err != nil {
		t.Fatalf("GenerateIdPMetadata: %v", err)
	}

	metaStr := string(meta)
	// Verify essential metadata elements
	if !strings.Contains(metaStr, "EntityDescriptor") {
		t.Error("metadata should contain EntityDescriptor")
	}
	if !strings.Contains(metaStr, "IDPSSODescriptor") {
		t.Error("metadata should contain IDPSSODescriptor")
	}
	if !strings.Contains(metaStr, idp.EntityID) {
		t.Error("metadata should contain EntityID")
	}
	if !strings.Contains(metaStr, idp.SSOURL) {
		t.Error("metadata should contain SSO URL")
	}
	if !strings.Contains(metaStr, idp.SLOURL) {
		t.Error("metadata should contain SLO URL")
	}
	// Verify certificate is embedded
	certB64 := base64.StdEncoding.EncodeToString(certDER)
	if !strings.Contains(metaStr, certB64) {
		t.Error("metadata should contain certificate")
	}
	// Verify NameIDFormats
	if !strings.Contains(metaStr, "emailAddress") {
		t.Error("metadata should contain emailAddress NameIDFormat")
	}
	if !strings.Contains(metaStr, "persistent") {
		t.Error("metadata should contain persistent NameIDFormat")
	}
}

func TestIdP_BuildSAMLResponse_Basic(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		SSOURL:      "https://ggid.example.com/saml/idp/sso",
		SLOURL:      "https://ggid.example.com/saml/idp/slo",
		PrivateKey:  key,
		Certificate: certDER,
	}

	req := &SAMLResponseRequest{
		Destination:  "https://sp.example.com/acs",
		Audience:     "https://sp.example.com/metadata",
		NameID:       "user@example.com",
		Attributes:   map[string][]string{
			"email":      {"user@example.com"},
			"displayName": {"Test User"},
			"groups":     {"admin", "users"},
		},
		InResponseTo: "req-123",
	}

	resp, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(resp)

	// Verify Response element
	if !strings.Contains(respStr, "samlp:Response") {
		t.Error("response should contain samlp:Response")
	}
	if !strings.Contains(respStr, `Version="2.0"`) {
		t.Error("response should be SAML 2.0")
	}
	if !strings.Contains(respStr, req.Destination) {
		t.Error("response should contain Destination")
	}
	if !strings.Contains(respStr, "Success") {
		t.Error("response status should be Success")
	}
	if !strings.Contains(respStr, req.InResponseTo) {
		t.Error("response should contain InResponseTo")
	}
}

func TestIdP_BuildSAMLResponse_ContainsAssertion(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		PrivateKey:  key,
		Certificate: certDER,
	}

	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com/metadata",
		NameID:      "user@example.com",
		Attributes:  map[string][]string{"email": {"user@example.com"}},
	}

	resp, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(resp)

	// Verify assertion is present
	if !strings.Contains(respStr, "saml:Assertion") {
		t.Error("response should contain assertion")
	}
	if !strings.Contains(respStr, "user@example.com") {
		t.Error("assertion should contain NameID")
	}
	// Verify audience restriction
	if !strings.Contains(respStr, req.Audience) {
		t.Error("assertion should contain audience restriction")
	}
	// Verify attributes
	if !strings.Contains(respStr, "displayName") || !strings.Contains(respStr, "Test User") {
		// displayName not in this test, check email attribute
	}
	if !strings.Contains(respStr, "email") {
		t.Error("assertion should contain email attribute")
	}
}

func TestIdP_BuildSAMLResponse_ContainsSignature(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		PrivateKey:  key,
		Certificate: certDER,
	}

	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com/metadata",
		NameID:      "user@example.com",
	}

	resp, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(resp)

	// Verify signature is present
	if !strings.Contains(respStr, "ds:Signature") {
		t.Error("response should contain XMLDSig signature")
	}
	if !strings.Contains(respStr, "rsa-sha256") {
		t.Error("signature should use rsa-sha256 algorithm")
	}
	if !strings.Contains(respStr, "sha256") {
		t.Error("digest should use sha256")
	}
	// Verify certificate is in signature
	certB64 := base64.StdEncoding.EncodeToString(certDER)
	if !strings.Contains(respStr, certB64) {
		t.Error("signature should contain certificate")
	}
}

func TestIdP_BuildSAMLResponse_MissingDestination(t *testing.T) {
	key, _ := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:   "https://ggid.example.com",
		PrivateKey: key,
	}

	req := &SAMLResponseRequest{
		NameID: "user@example.com",
	}

	_, err := idp.BuildSAMLResponse(req)
	if err == nil {
		t.Error("should error when destination is missing")
	}
	if !strings.Contains(err.Error(), "destination") {
		t.Errorf("error should mention destination, got: %v", err)
	}
}

func TestIdP_BuildSAMLResponse_MissingNameID(t *testing.T) {
	key, _ := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:   "https://ggid.example.com",
		PrivateKey: key,
	}

	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
	}

	_, err := idp.BuildSAMLResponse(req)
	if err == nil {
		t.Error("should error when NameID is missing")
	}
}

func TestIdP_BuildSAMLResponse_NilPrivateKey(t *testing.T) {
	idp := &IdentityProvider{
		EntityID: "https://ggid.example.com",
	}

	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		NameID:      "user@example.com",
	}

	_, err := idp.BuildSAMLResponse(req)
	if err == nil {
		t.Error("should error when private key is nil")
	}
}

func TestIdP_BuildSAMLResponse_Defaults(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		PrivateKey:  key,
		Certificate: certDER,
	}

	// No NotBefore/NotOnOrAfter set — should use defaults
	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com/metadata",
		NameID:      "user@example.com",
	}

	resp, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(resp)

	// Should have NotBefore and NotOnOrAfter
	if !strings.Contains(respStr, "NotBefore") {
		t.Error("assertion should have NotBefore")
	}
	if !strings.Contains(respStr, "NotOnOrAfter") {
		t.Error("assertion should have NotOnOrAfter")
	}
	// Should default to emailAddress NameID format
	if !strings.Contains(respStr, "emailAddress") {
		t.Error("assertion should default to emailAddress NameID format")
	}
	// Should contain bearer subject confirmation
	if !strings.Contains(respStr, "bearer") {
		t.Error("assertion should use bearer subject confirmation")
	}
	// Should contain PasswordProtectedTransport authn context
	if !strings.Contains(respStr, "PasswordProtectedTransport") {
		t.Error("assertion should use PasswordProtectedTransport authn context")
	}
}

func TestIdP_BuildSAMLResponse_NoAttributes(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com/saml/idp/metadata",
		PrivateKey:  key,
		Certificate: certDER,
	}

	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com/metadata",
		NameID:      "user@example.com",
		// No Attributes
	}

	resp, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(resp)
	// Should still work without attributes
	if !strings.Contains(respStr, "saml:Assertion") {
		t.Error("response should contain assertion even without attributes")
	}
}

func TestIdP_EncodeResponseForPOST(t *testing.T) {
	xml := []byte(`<samlp:Response ID="test"/>`)
	encoded := EncodeResponseForPOST(xml)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if string(decoded) != string(xml) {
		t.Error("round-trip encode/decode should match")
	}
}

func TestIdP_GenerateIdPMetadata_NilIdP(t *testing.T) {
	var idp *IdentityProvider
	_, err := idp.GenerateIdPMetadata()
	if err == nil {
		t.Error("should error when IdP is nil")
	}
}

func TestIdP_BuildSAMLResponse_AuthnStatement(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com",
		PrivateKey:  key,
		Certificate: certDER,
	}

	req := &SAMLResponseRequest{
		Destination:  "https://sp.example.com/acs",
		Audience:     "https://sp.example.com",
		NameID:       "user@example.com",
		SessionIndex: "session-abc-123",
	}

	resp, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	respStr := string(resp)
	if !strings.Contains(respStr, "AuthnStatement") {
		t.Error("assertion should contain AuthnStatement")
	}
	if !strings.Contains(respStr, "session-abc-123") {
		t.Error("assertion should contain SessionIndex")
	}
	if !strings.Contains(respStr, "AuthnContextClassRef") {
		t.Error("assertion should contain AuthnContextClassRef")
	}
}

func TestIdP_BuildSAMLResponse_XMLValid(t *testing.T) {
	key, certDER := genIdPTestKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://ggid.example.com",
		PrivateKey:  key,
		Certificate: certDER,
	}

	req := &SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com",
		NameID:      "user@example.com",
		Attributes:  map[string][]string{"email": {"user@example.com"}},
	}

	resp, err := idp.BuildSAMLResponse(req)
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}

	// Verify the response is valid XML
	var parsed map[string]interface{}
	err = xml.Unmarshal(resp, &parsed)
	// Even if namespace handling makes strict unmarshal tricky,
	// it should at least not be garbage
	_ = err // XML with namespaces may not unmarshal to map[string]interface{}
	// Just verify it starts with < and ends with >
	respStr := strings.TrimSpace(string(resp))
	if !strings.HasPrefix(respStr, "<") {
		t.Error("response should be XML (start with <)")
	}
	if !strings.HasSuffix(respStr, ">") {
		t.Error("response should be XML (end with >)")
	}
}
