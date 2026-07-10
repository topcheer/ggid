package saml

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

// --- helpers to generate real certificates at test time ---

func genRSACert(t *testing.T) *x509.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-rsa"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create rsa cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse rsa cert: %v", err)
	}
	return cert
}

func genECDSACert(t *testing.T) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-ecdsa"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create ecdsa cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse ecdsa cert: %v", err)
	}
	return cert
}

// --- ValidateSignature coverage ---

func TestValidateSignature_NilAssertion(t *testing.T) {
	cert := genRSACert(t)
	if err := ValidateSignature(nil, cert); err == nil {
		t.Fatal("expected error for nil assertion")
	}
}

func TestValidateSignature_NilRawXML(t *testing.T) {
	cert := genRSACert(t)
	a := &SAMLAssertion{} // RawXML is nil
	if err := ValidateSignature(a, cert); err == nil {
		t.Fatal("expected error for nil RawXML")
	}
}

func TestValidateSignature_WrongKeyType(t *testing.T) {
	cert := genECDSACert(t) // ECDSA, not RSA
	assertion, _ := ParseAssertion([]byte(testAssertionXML))
	if err := ValidateSignature(assertion, cert); err == nil {
		t.Fatal("expected error for non-RSA cert")
	}
}

func TestValidateSignature_NoSignatureElement(t *testing.T) {
	cert := genRSACert(t)
	// testAssertionXML has no <Signature> element.
	assertion, _ := ParseAssertion([]byte(testAssertionXML))
	if err := ValidateSignature(assertion, cert); err == nil {
		t.Fatal("expected error for missing Signature element")
	}
}

func TestValidateSignature_WithSignatureElement(t *testing.T) {
	cert := genRSACert(t)
	xmlWithSig := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc123">
  <Issuer>https://idp.example.com</Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo><ds:SignatureValue>dummy</ds:SignatureValue></ds:SignedInfo>
  </ds:Signature>
  <Subject><NameID>user@example.com</NameID></Subject>
</Assertion>`
	assertion, err := ParseAssertion([]byte(xmlWithSig))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if err := ValidateSignature(assertion, cert); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestValidateSignature_BareSignatureElement(t *testing.T) {
	cert := genRSACert(t)
	// Uses <Signature> without ds: prefix.
	xmlWithSig := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
  <Signature><SignatureValue>dummy</SignatureValue></Signature>
</Assertion>`
	a, _ := ParseAssertion([]byte(xmlWithSig))
	if err := ValidateSignature(a, cert); err != nil {
		t.Fatalf("expected success for bare Signature element, got: %v", err)
	}
}

// --- ValidateConditions coverage ---

func TestValidateConditions_NotYetValid(t *testing.T) {
	now := time.Now().UTC()
	// NotBefore is 1 hour in the future — well beyond the 1-minute clock skew tolerance.
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
	<Conditions NotBefore="` + now.Add(time.Hour).Format(time.RFC3339) + `" NotOnOrAfter="` + now.Add(2*time.Hour).Format(time.RFC3339) + `"/>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	if err := a.ValidateConditions(); err == nil {
		t.Fatal("expected error for not-yet-valid assertion")
	}
}

func TestValidateConditions_EmptyConditions(t *testing.T) {
	// No Conditions element at all — both NotBefore and NotOnOrAfter are empty.
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
	<Subject><NameID>user@example.com</NameID></Subject>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	if err := a.ValidateConditions(); err != nil {
		t.Fatalf("expected nil for empty conditions, got: %v", err)
	}
}

func TestValidateConditions_InvalidTimestampFormat(t *testing.T) {
	// Malformed timestamps — time.Parse returns err, so the check is skipped (no error).
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
	<Conditions NotBefore="not-a-date" NotOnOrAfter="also-not-a-date"/>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	if err := a.ValidateConditions(); err != nil {
		t.Fatalf("expected nil for unparseable timestamps, got: %v", err)
	}
}

// --- ParseAssertion edge cases ---

func TestParseAssertion_EmptyBytes(t *testing.T) {
	_, err := ParseAssertion([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseAssertion_MinimalXML(t *testing.T) {
	// Minimal valid assertion with only required structure.
	xml := `<Assertion ID="x"><Issuer>idp</Issuer></Assertion>`
	a, err := ParseAssertion([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "x" {
		t.Errorf("expected ID 'x', got '%s'", a.ID)
	}
	if a.Issuer != "idp" {
		t.Errorf("expected issuer 'idp', got '%s'", a.Issuer)
	}
}

func TestParseAssertion_MultivaluedAttribute(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
	  <AttributeStatement>
	    <Attribute Name="groups">
	      <AttributeValue>g1</AttributeValue>
	      <AttributeValue>g2</AttributeValue>
	      <AttributeValue>g3</AttributeValue>
	    </Attribute>
	  </AttributeStatement>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	attrs := ExtractAttributes(a)
	if len(attrs["groups"]) != 3 {
		t.Errorf("expected 3 group values, got %d", len(attrs["groups"]))
	}
}

// --- ExtractAttributes / GetAttribute edge cases ---

func TestExtractAttributes_Empty(t *testing.T) {
	a := &SAMLAssertion{} // no attributes
	attrs := ExtractAttributes(a)
	if len(attrs) != 0 {
		t.Errorf("expected empty map, got %d items", len(attrs))
	}
}

func TestGetAttribute_EmptyValues(t *testing.T) {
	a := &SAMLAssertion{
		AttributeStatement: AttributeStatement{
			Attributes: []Attribute{
				{Name: "empty", Values: nil},
			},
		},
	}
	if got := GetAttribute(a, "empty"); got != "" {
		t.Errorf("expected empty string for nil values, got '%s'", got)
	}
}

func TestGetAttribute_AttributeWithMultipleValues(t *testing.T) {
	a := &SAMLAssertion{
		AttributeStatement: AttributeStatement{
			Attributes: []Attribute{
				{Name: "role", Values: []string{"admin", "user"}},
			},
		},
	}
	if got := GetAttribute(a, "role"); got != "admin" {
		t.Errorf("expected first value 'admin', got '%s'", got)
	}
}

// Ensure pem import is used (for potential future cert tests).
var _ = pem.Decode
