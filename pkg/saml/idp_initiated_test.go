package saml

import (
	"crypto/x509"
	"encoding/base64"
	"testing"
)

func TestHandleIdPInitiatedSSO_EmptyResponse(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}
	_, err := sp.HandleIdPInitiatedSSO(&IdPInitiatedSSORequest{}, nil)
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestHandleIdPInitiatedSSO_NilRequest(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}
	_, err := sp.HandleIdPInitiatedSSO(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestHandleIdPInitiatedSSO_InvalidBase64(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}
	_, err := sp.HandleIdPInitiatedSSO(&IdPInitiatedSSORequest{
		SAMLResponse: "!!!invalid base64!!!",
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestHandleIdPInitiatedSSO_InvalidXML(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}
	encoded := base64.StdEncoding.EncodeToString([]byte("<not valid saml"))
	_, err := sp.HandleIdPInitiatedSSO(&IdPInitiatedSSORequest{
		SAMLResponse: encoded,
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
}

func TestHandleIdPInitiatedSSO_SuccessResponse(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}

	samlXML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" 
  ID="_resp1" IssueInstant="2025-01-01T00:00:00Z" Version="2.0"
  Destination="https://sp.example.com/acs">
  <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status>
  <saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" ID="_a1" IssueInstant="2025-01-01T00:00:00Z" Version="2.0">
    <saml:Issuer>https://idp.example.com</saml:Issuer>
    <saml:Subject>
      <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">user@example.com</saml:NameID>
      <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml:SubjectConfirmationData Recipient="https://sp.example.com/acs" NotOnOrAfter="2030-01-01T00:00:00Z"/>
      </saml:SubjectConfirmation>
    </saml:Subject>
    <saml:Conditions NotBefore="2020-01-01T00:00:00Z" NotOnOrAfter="2030-01-01T00:00:00Z"/>
    <saml:AuthnStatement SessionIndex="_session1" AuthnInstant="2025-01-01T00:00:00Z">
      <saml:AuthnContext><saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport</saml:AuthnContextClassRef></saml:AuthnContext>
    </saml:AuthnStatement>
    <saml:AttributeStatement>
      <saml:Attribute Name="email"><saml:AttributeValue>user@example.com</saml:AttributeValue></saml:Attribute>
      <saml:Attribute Name="role"><saml:AttributeValue>admin</saml:AttributeValue></saml:Attribute>
    </saml:AttributeStatement>
  </saml:Assertion>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(samlXML))
	result, err := sp.HandleIdPInitiatedSSO(&IdPInitiatedSSORequest{
		SAMLResponse: encoded,
		RelayState:   "state123",
	}, nil)
	if err != nil {
		t.Fatalf("HandleIdPInitiatedSSO failed: %v", err)
	}
	if result.NameID != "user@example.com" {
		t.Errorf("expected NameID user@example.com, got %q", result.NameID)
	}
	if result.Issuer != "https://idp.example.com" {
		t.Errorf("expected Issuer, got %q", result.Issuer)
	}
	if result.SessionIndex != "_session1" {
		t.Errorf("expected SessionIndex _session1, got %q", result.SessionIndex)
	}
	if result.RelayState != "state123" {
		t.Errorf("expected RelayState state123, got %q", result.RelayState)
	}
	if emails, ok := result.Attributes["email"]; !ok || len(emails) != 1 {
		t.Errorf("expected email attribute, got %v", result.Attributes)
	}
}

func TestHandleIdPInitiatedSSO_DestinationMismatch(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}

	samlXML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" 
  ID="_resp2" IssueInstant="2025-01-01T00:00:00Z" Version="2.0"
  Destination="https://wrong.example.com/acs">
  <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(samlXML))
	_, err := sp.HandleIdPInitiatedSSO(&IdPInitiatedSSORequest{
		SAMLResponse: encoded,
	}, nil)
	if err == nil {
		t.Fatal("expected error for destination mismatch")
	}
}

func TestHandleIdPInitiatedSSO_NonSuccessStatus(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}

	samlXML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" 
  ID="_resp3" IssueInstant="2025-01-01T00:00:00Z" Version="2.0">
  <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Requester"/></samlp:Status>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(samlXML))
	_, err := sp.HandleIdPInitiatedSSO(&IdPInitiatedSSORequest{
		SAMLResponse: encoded,
	}, nil)
	if err == nil {
		t.Fatal("expected error for non-success status")
	}
}

func TestHandleIdPInitiatedSSO_RejectInResponseTo(t *testing.T) {
	sp := &ServiceProvider{EntityID: "https://sp.example.com", ACSURL: "https://sp.example.com/acs"}

	samlXML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" 
  ID="_resp4" InResponseTo="_someprior" IssueInstant="2025-01-01T00:00:00Z" Version="2.0"
  Destination="https://sp.example.com/acs">
  <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</saml:Issuer>
  <samlp:Status><samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/></samlp:Status>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(samlXML))
	_, err := sp.HandleIdPInitiatedSSO(&IdPInitiatedSSORequest{
		SAMLResponse: encoded,
	}, nil)
	if err == nil {
		t.Fatal("expected error for InResponseTo in IdP-initiated SSO")
	}
}

func TestIsTrustedIdP(t *testing.T) {
	trusted := []string{"https://idp1.example.com", "https://idp2.example.com"}
	if !IsTrustedIdP("https://idp1.example.com", trusted) {
		t.Error("expected idp1 to be trusted")
	}
	if IsTrustedIdP("https://untrusted.example.com", trusted) {
		t.Error("expected untrusted IdP to be rejected")
	}
	// Case insensitive
	if !IsTrustedIdP("HTTPS://IDP1.EXAMPLE.COM", trusted) {
		t.Error("expected case-insensitive match")
	}
}

// Suppress unused import guard for x509 (used in production signature verification).
var _ = x509.NewCertPool
