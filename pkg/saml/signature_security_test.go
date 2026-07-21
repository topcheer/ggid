package saml

import (
	"encoding/base64"
	"strings"
	"testing"
)

// These tests prove that the XML-DSig verification is real: forged,
// self-signed, or tampered assertions must be rejected even when they are
// structurally well-formed and contain a Signature element.

// signWithIdP is intentionally inlined in each test for clarity.

func TestSecurity_ValidIdPSignatureAccepted(t *testing.T) {
	cert, privKey := genRSACertWithKey(t)
	idp := &IdentityProvider{
		EntityID:    "https://idp.example.com/saml",
		PrivateKey:  privKey,
		Certificate: cert.Raw,
		KeyID:       "sec-test-key",
	}
	resp, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		Audience:    "https://sp.example.com/saml",
		NameID:      "alice@corp.com",
	})
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}
	assertionXML := extractAssertionFromResponse(t, resp)

	a, err := VerifySignedAssertion(assertionXML, cert)
	if err != nil {
		t.Fatalf("valid IdP-signed assertion rejected: %v", err)
	}
	if a.Subject.NameID != "alice@corp.com" {
		t.Errorf("NameID = %q, want alice@corp.com", a.Subject.NameID)
	}
}

// TestSecurity_ForgedSignatureValueRejected: attacker takes a genuinely
// signed assertion and replaces the SignatureValue with garbage.
func TestSecurity_ForgedSignatureValueRejected(t *testing.T) {
	cert, privKey := genRSACertWithKey(t)
	idp := &IdentityProvider{
		EntityID: "https://idp.example.com/saml", PrivateKey: privKey,
		Certificate: cert.Raw, KeyID: "k",
	}
	resp, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		NameID:      "alice@corp.com",
	})
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}
	assertionXML := string(extractAssertionFromResponse(t, resp))

	// Replace the real signature value with a forged one.
	forged := base64.StdEncoding.EncodeToString([]byte("forged-signature-bytes-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789ab"))
	start := strings.Index(assertionXML, "<ds:SignatureValue>")
	end := strings.Index(assertionXML, "</ds:SignatureValue>")
	if start < 0 || end < 0 {
		t.Fatal("no SignatureValue in signed assertion")
	}
	tampered := assertionXML[:start+len("<ds:SignatureValue>")] + forged + assertionXML[end:]

	if _, err := VerifySignedAssertion([]byte(tampered), cert); err == nil {
		t.Fatal("forged SignatureValue must be rejected")
	}
}

// TestSecurity_SelfSignedForgeryRejected: attacker generates their own key
// pair, signs a forged assertion, and embeds their own cert in KeyInfo.
// Verification against the trusted IdP cert must fail.
func TestSecurity_SelfSignedForgeryRejected(t *testing.T) {
	trustedCert, _ := genRSACertWithKey(t)

	// Attacker's own IdP with attacker's key.
	attackerCert, attackerKey := genRSACertWithKey(t)
	attackerIdP := &IdentityProvider{
		EntityID: "https://idp.example.com/saml", // same issuer string!
		PrivateKey: attackerKey, Certificate: attackerCert.Raw, KeyID: "attacker",
	}
	resp, err := attackerIdP.BuildSAMLResponse(&SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		NameID:      "admin@corp.com", // privilege escalation attempt
	})
	if err != nil {
		t.Fatalf("attacker BuildSAMLResponse: %v", err)
	}
	forgedAssertion := extractAssertionFromResponse(t, resp)

	if _, err := VerifySignedAssertion(forgedAssertion, trustedCert); err == nil {
		t.Fatal("self-signed forgery verified against trusted IdP cert must be rejected")
	}
}

// TestSecurity_ContentTamperingRejected: signature is valid but the attacker
// modifies the NameID after signing — digest mismatch must reject it.
func TestSecurity_ContentTamperingRejected(t *testing.T) {
	cert, privKey := genRSACertWithKey(t)
	idp := &IdentityProvider{
		EntityID: "https://idp.example.com/saml", PrivateKey: privKey,
		Certificate: cert.Raw, KeyID: "k",
	}
	resp, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		NameID:      "user@corp.com",
	})
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}
	assertionXML := string(extractAssertionFromResponse(t, resp))

	tampered := strings.Replace(assertionXML, "user@corp.com", "admin@corp.com", 1)
	if _, err := VerifySignedAssertion([]byte(tampered), cert); err == nil {
		t.Fatal("content-tampered assertion must be rejected (digest mismatch)")
	}
}

// TestSecurity_UnsignedAssertionRejected: no Signature element at all.
func TestSecurity_UnsignedAssertionRejected(t *testing.T) {
	cert, _ := genRSACertWithKey(t)
	unsigned := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_unsigned" Version="2.0">
  <Issuer>https://idp.example.com</Issuer>
  <Subject><NameID>admin@corp.com</NameID></Subject>
</Assertion>`
	if _, err := VerifySignedAssertion([]byte(unsigned), cert); err == nil {
		t.Fatal("unsigned assertion must be rejected")
	}
}

// TestSecurity_ValidateSignatureForgedRejected exercises the ValidateSignature
// public API with a forged signature.
func TestSecurity_ValidateSignatureForgedRejected(t *testing.T) {
	cert, _ := genRSACertWithKey(t)
	forged := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_forged" Version="2.0">
  <Issuer>https://idp.example.com</Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_forged">
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=</ds:SignatureValue>
  </ds:Signature>
  <Subject><NameID>admin@corp.com</NameID></Subject>
</Assertion>`
	a, err := ParseAssertion([]byte(forged))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := ValidateSignature(a, cert); err == nil {
		t.Fatal("ValidateSignature must reject forged signature")
	}
}

// TestSecurity_ReferenceIDMismatchRejected: signature references a different
// assertion ID than the one presented.
func TestSecurity_ReferenceIDMismatchRejected(t *testing.T) {
	cert, privKey := genRSACertWithKey(t)
	idp := &IdentityProvider{
		EntityID: "https://idp.example.com/saml", PrivateKey: privKey,
		Certificate: cert.Raw, KeyID: "k",
	}
	resp, err := idp.BuildSAMLResponse(&SAMLResponseRequest{
		Destination: "https://sp.example.com/acs",
		NameID:      "user@corp.com",
	})
	if err != nil {
		t.Fatalf("BuildSAMLResponse: %v", err)
	}
	assertionXML := string(extractAssertionFromResponse(t, resp))

	// Swap the assertion ID attribute — reference URI no longer matches.
	swapped := strings.Replace(assertionXML, `ID="`, `ID="_evil`, 1)
	if _, err := VerifySignedAssertion([]byte(swapped), cert); err == nil {
		t.Fatal("assertion with mismatched reference ID must be rejected")
	}
}
