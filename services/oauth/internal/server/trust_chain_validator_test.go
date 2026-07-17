package server

import (
	"testing"
)

func TestExtractSAMLIssuer_ResponseLevel(t *testing.T) {
	xml := []byte(`<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"><saml:Issuer>https://idp.example.com</saml:Issuer></samlp:Response>`)
	issuer := extractSAMLIssuer(xml)
	if issuer != "https://idp.example.com" {
		t.Errorf("expected https://idp.example.com, got %s", issuer)
	}
}

func TestExtractSAMLIssuer_AssertionLevel(t *testing.T) {
	xml := []byte(`<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion"><Issuer>https://idp2.example.com</Issuer></Assertion>`)
	issuer := extractSAMLIssuer(xml)
	if issuer != "https://idp2.example.com" {
		t.Errorf("expected https://idp2.example.com, got %s", issuer)
	}
}

func TestExtractSAMLIssuer_Empty(t *testing.T) {
	issuer := extractSAMLIssuer([]byte("<invalid>"))
	if issuer != "" {
		t.Errorf("expected empty issuer, got %s", issuer)
	}
}

func TestTrustChainValidator_NilPool(t *testing.T) {
	v := NewTrustChainValidator(nil)
	if err := v.ValidateSAMLIssuer(nil, "tenant", "issuer"); err != nil {
		t.Errorf("nil pool should not error: %v", err)
	}
	if err := v.ValidateOIDCClient(nil, "tenant", "client"); err != nil {
		t.Errorf("nil pool should not error: %v", err)
	}
}

func TestTrustChainValidator_NilReceiver(t *testing.T) {
	var v *TrustChainValidator
	if err := v.ValidateSAMLIssuer(nil, "tenant", "issuer"); err != nil {
		t.Errorf("nil receiver should not error: %v", err)
	}
}
