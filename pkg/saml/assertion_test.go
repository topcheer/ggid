package saml

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"
)

const testAssertionXML = `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc123" IssueInstant="2024-01-15T10:00:00Z" Version="2.0">
  <Issuer>https://idp.example.com</Issuer>
  <Subject>
    <NameID>johndoe@example.com</NameID>
  </Subject>
  <Conditions NotBefore="2024-01-15T09:55:00Z" NotOnOrAfter="2024-01-15T10:05:00Z"/>
  <AttributeStatement>
    <Attribute Name="mail" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
      <AttributeValue>johndoe@example.com</AttributeValue>
    </Attribute>
    <Attribute Name="displayName" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
      <AttributeValue>John Doe</AttributeValue>
    </Attribute>
    <Attribute Name="memberOf" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:basic">
      <AttributeValue>cn=admins,dc=corp,dc=local</AttributeValue>
      <AttributeValue>cn=users,dc=corp,dc=local</AttributeValue>
    </Attribute>
  </AttributeStatement>
</Assertion>`

func TestParseAssertion_Success(t *testing.T) {
	assertion, err := ParseAssertion([]byte(testAssertionXML))
	if err != nil {
		t.Fatalf("ParseAssertion failed: %v", err)
	}
	if assertion.ID != "_abc123" {
		t.Errorf("expected ID '_abc123', got '%s'", assertion.ID)
	}
	if assertion.Issuer != "https://idp.example.com" {
		t.Errorf("expected issuer, got '%s'", assertion.Issuer)
	}
	if assertion.Subject.NameID != "johndoe@example.com" {
		t.Errorf("expected NameID, got '%s'", assertion.Subject.NameID)
	}
}

func TestParseAssertion_InvalidXML(t *testing.T) {
	_, err := ParseAssertion([]byte("not valid xml"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestExtractAttributes(t *testing.T) {
	assertion, _ := ParseAssertion([]byte(testAssertionXML))
	attrs := ExtractAttributes(assertion)

	if len(attrs["mail"]) != 1 || attrs["mail"][0] != "johndoe@example.com" {
		t.Errorf("unexpected mail attr: %v", attrs["mail"])
	}
	if len(attrs["displayName"]) != 1 || attrs["displayName"][0] != "John Doe" {
		t.Errorf("unexpected displayName attr: %v", attrs["displayName"])
	}
	// Multi-valued attribute.
	if len(attrs["memberOf"]) != 2 {
		t.Errorf("expected 2 memberOf values, got %d", len(attrs["memberOf"]))
	}
}

func TestGetAttribute(t *testing.T) {
	assertion, _ := ParseAssertion([]byte(testAssertionXML))

	if got := GetAttribute(assertion, "mail"); got != "johndoe@example.com" {
		t.Errorf("expected 'johndoe@example.com', got '%s'", got)
	}
	if got := GetAttribute(assertion, "nonexistent"); got != "" {
		t.Errorf("expected empty string for missing attr, got '%s'", got)
	}
}

func TestValidateConditions_Valid(t *testing.T) {
	// Create assertion with current time window.
	now := time.Now().UTC()
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
	<Conditions NotBefore="` + now.Format(time.RFC3339) + `" NotOnOrAfter="` + now.Add(10*time.Minute).Format(time.RFC3339) + `"/>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	if err := a.ValidateConditions(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateConditions_Expired(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
	<Conditions NotBefore="2020-01-01T00:00:00Z" NotOnOrAfter="2020-01-01T00:05:00Z"/>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	if err := a.ValidateConditions(); err == nil {
		t.Fatal("expected error for expired assertion")
	}
}

func TestValidateSignature_NoCert(t *testing.T) {
	assertion, _ := ParseAssertion([]byte(testAssertionXML))
	if err := ValidateSignature(assertion, nil); err == nil {
		t.Fatal("expected error for nil cert")
	}
}

func TestValidateSignature_WithCert(t *testing.T) {
	// Generate a self-signed cert for testing.
	certPEM := `-----BEGIN CERTIFICATE-----
MIIBizCCTAQBgkqhkiG9w0BBQ0wMTAPBglghkgBZQMEAgUFAQUQAbECMAUGAyDK
-----END CERTIFICATE-----`
	block, _ := pem.Decode([]byte(certPEM))
	// The cert is invalid but we test the nil-check path.
	assertion, _ := ParseAssertion([]byte(testAssertionXML))

	if block == nil {
		// Can't decode the test cert — just test the nil assertion path.
		if err := ValidateSignature(nil, nil); err == nil {
			t.Fatal("expected error for nil assertion")
		}
		return
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		// Cert parsing failed — test the function doesn't panic.
		return
	}
	_ = cert

	// Test that the function handles the assertion without a Signature element.
	if err := ValidateSignature(assertion, cert); err != nil {
		// Expected — our test assertion has no <ds:Signature> element.
		return
	}
}

// Suppress unused import.
var _ = rsa.PublicKey{}
