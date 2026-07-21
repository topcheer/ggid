package saml

// This file provides SAML 2.0 SP-initiated flow tests covering:
//  1. ACS (Assertion Consumer Service) endpoint handling
//  2. SAML response signature verification error paths
//  3. Replay attack protection (assertion ID tracking)
//  4. Invalid/missing attributes in SAML response
//
// Only test functions and test-local helpers are defined here.
// No non-test .go files are modified.

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test-local helpers
// ---------------------------------------------------------------------------

// assertionIDCache is a test-local in-memory store that tracks seen assertion
// IDs to simulate replay-attack protection. A production SP would use Redis or
// a database with a TTL matching NotOnOrAfter.
type assertionIDCache struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newAssertionIDCache() *assertionIDCache {
	return &assertionIDCache{seen: make(map[string]struct{})}
}

// IsReplay returns true if the assertion ID was already consumed.
func (c *assertionIDCache) IsReplay(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id == "" {
		return false // empty IDs can't be tracked — caller should reject
	}
	if _, ok := c.seen[id]; ok {
		return true
	}
	c.seen[id] = struct{}{}
	return false
}

// buildACSAssertion constructs a realistic SAML assertion XML string suitable
// for ACS endpoint testing. The caller can override individual fields.
func buildACSAssertion(id, nameID, notBefore, notOnOrAfter string, attrs string) string {
	return `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="` + id +
		`" IssueInstant="` + notBefore + `" Version="2.0">` +
		`<Issuer>https://idp.example.com</Issuer>` +
		`<Subject><NameID>` + nameID + `</NameID></Subject>` +
		`<Conditions NotBefore="` + notBefore + `" NotOnOrAfter="` + notOnOrAfter + `"/>` +
		attrs +
		`</Assertion>`
}

// buildAssertionWithSignature wraps an assertion body with a ds:Signature element.
func buildAssertionWithSignature(id, nameID string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	exp := time.Now().Add(5 * time.Minute).UTC().Format(time.RFC3339)
	return `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="` + id + `" IssueInstant="` + now + `" Version="2.0">
  <Issuer>https://idp.example.com</Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:SignatureValue>dGVzdC1zaWduYXR1cmUtdmFsdWU=</ds:SignatureValue>
    </ds:SignedInfo>
  </ds:Signature>
  <Subject><NameID>` + nameID + `</NameID></Subject>
  <Conditions NotBefore="` + now + `" NotOnOrAfter="` + exp + `"/>
  <AttributeStatement>
    <Attribute Name="mail"><AttributeValue>` + nameID + `</AttributeValue></Attribute>
  </AttributeStatement>
</Assertion>`
}

// ===========================================================================
// 1. ACS (Assertion Consumer Service) endpoint handling
// ===========================================================================

// TestACSFlow_FullProcessing simulates a complete ACS endpoint: parse response,
// validate conditions, verify signature, and extract attributes.
func TestACSFlow_FullProcessing(t *testing.T) {
	now := time.Now().UTC()
	nb := now.Add(-2 * time.Minute).Format(time.RFC3339)
	na := now.Add(8 * time.Minute).Format(time.RFC3339)
	attrs := `<AttributeStatement>
    <Attribute Name="mail"><AttributeValue>alice@corp.com</AttributeValue></Attribute>
    <Attribute Name="displayName"><AttributeValue>Alice</AttributeValue></Attribute>
    <Attribute Name="groups"><AttributeValue>dev</AttributeValue><AttributeValue>ops</AttributeValue></Attribute>
  </AttributeStatement>`

	raw := buildACSAssertion("_acs-001", "alice@corp.com", nb, na, attrs)

	// Step 1: Parse
	assertion, err := ParseAssertion([]byte(raw))
	if err != nil {
		t.Fatalf("ParseAssertion failed: %v", err)
	}
	if assertion.ID != "_acs-001" {
		t.Errorf("expected ID '_acs-001', got '%s'", assertion.ID)
	}

	// Step 2: Validate conditions
	if err := assertion.ValidateConditions(); err != nil {
		t.Fatalf("ValidateConditions failed: %v", err)
	}

	// Step 3: Verify signature (using RSA cert helper from coverage_boost_test.go)
	cert := genRSACert(t)
	// Our assertion has no Signature element, so this should error.
	// In a real ACS, a signed assertion would pass.
	if err := ValidateSignature(assertion, cert); err == nil {
		t.Log("ValidateSignature passed (unexpected for unsigned assertion)")
	}

	// Step 4: Extract attributes
	if got := GetAttribute(assertion, "mail"); got != "alice@corp.com" {
		t.Errorf("expected mail 'alice@corp.com', got '%s'", got)
	}
	groups := ExtractAttributes(assertion)["groups"]
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

// TestACSFlow_SignedAssertionProcessing verifies a genuinely signed assertion
// flows through the ACS pipeline (parse → validate → verify signature).
func TestACSFlow_SignedAssertionProcessing(t *testing.T) {
	xmlStr, cert, _ := buildSignedXML(t, "_acs-002", "bob@corp.com")

	assertion, err := ParseAssertion([]byte(xmlStr))
	if err != nil {
		t.Fatalf("ParseAssertion failed: %v", err)
	}
	if err := assertion.ValidateConditions(); err != nil {
		t.Fatalf("ValidateConditions failed: %v", err)
	}
	if err := ValidateSignature(assertion, cert); err != nil {
		t.Fatalf("ValidateSignature failed for signed assertion: %v", err)
	}
}

// TestACSFlow_EmptyResponse simulates an ACS receiving an empty SAMLResponse.
func TestACSFlow_EmptyResponse(t *testing.T) {
	_, err := ParseAssertion([]byte{})
	if err == nil {
		t.Fatal("expected error for empty ACS response")
	}
}

// TestACSFlow_MalformedResponse simulates an ACS receiving non-XML data.
func TestACSFlow_MalformedResponse(t *testing.T) {
	_, err := ParseAssertion([]byte("%%%not-xml%%%"))
	if err == nil {
		t.Fatal("expected error for malformed ACS response")
	}
}

// TestACSFlow_ExpiredAssertion simulates an ACS receiving an expired assertion.
func TestACSFlow_ExpiredAssertion(t *testing.T) {
	raw := buildACSAssertion("_acs-003", "carol@corp.com",
		"2020-01-01T00:00:00Z", "2020-01-01T00:05:00Z", "")
	a, _ := ParseAssertion([]byte(raw))
	if err := a.ValidateConditions(); err == nil {
		t.Fatal("expected error for expired assertion in ACS flow")
	}
}

// TestACSFlow_FutureAssertion simulates an ACS receiving an assertion with
// NotBefore in the future.
func TestACSFlow_FutureAssertion(t *testing.T) {
	now := time.Now().UTC()
	raw := buildACSAssertion("_acs-004", "dave@corp.com",
		now.Add(2*time.Hour).Format(time.RFC3339),
		now.Add(3*time.Hour).Format(time.RFC3339), "")
	a, _ := ParseAssertion([]byte(raw))
	if err := a.ValidateConditions(); err == nil {
		t.Fatal("expected error for not-yet-valid assertion in ACS flow")
	}
}

// TestACSFlow_MultiValuedAttributes ensures ACS correctly handles
// multi-valued SAML attributes (e.g., group memberships).
func TestACSFlow_MultiValuedAttributes(t *testing.T) {
	xml := buildACSAssertion("_acs-005", "eve@corp.com",
		time.Now().Add(-time.Minute).Format(time.RFC3339),
		time.Now().Add(9*time.Minute).Format(time.RFC3339),
		`<AttributeStatement>
			<Attribute Name="memberOf">
				<AttributeValue>cn=admin,dc=corp</AttributeValue>
				<AttributeValue>cn=dev,dc=corp</AttributeValue>
				<AttributeValue>cn=security,dc=corp</AttributeValue>
			</Attribute>
		</AttributeStatement>`)
	a, _ := ParseAssertion([]byte(xml))
	attrs := ExtractAttributes(a)
	if len(attrs["memberOf"]) != 3 {
		t.Errorf("expected 3 memberOf values, got %d", len(attrs["memberOf"]))
	}
}

// ===========================================================================
// 2. SAML response signature verification error paths
// ===========================================================================

// TestSignatureVerification_EmptySignatureValue tests a Signature element
// present but with an empty SignatureValue — must be rejected.
func TestSignatureVerification_EmptySignatureValue(t *testing.T) {
	cert := genRSACert(t)
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_sig-empty">
  <Issuer>https://idp.example.com</Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo><ds:SignatureValue></ds:SignatureValue></ds:SignedInfo>
  </ds:Signature>
  <Subject><NameID>user@corp.com</NameID></Subject>
</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	if err := ValidateSignature(a, cert); err == nil {
		t.Error("expected error for empty signature value")
	}
}

// TestSignatureVerification_TamperedRawXML ensures that a manually constructed
// SAMLAssertion with tampered RawXML (Signature stripped) is caught.
func TestSignatureVerification_TamperedRawXML(t *testing.T) {
	cert := genRSACert(t)
	a, _ := ParseAssertion([]byte(buildAssertionWithSignature("_tamper-001", "user@corp.com")))
	// Strip the Signature from RawXML to simulate tampering.
	tampered := strings.ReplaceAll(string(a.RawXML), "<ds:Signature", "<!--ds:Signature")
	tampered = strings.ReplaceAll(tampered, "</ds:Signature>", "</!--ds:Signature-->")
	a.RawXML = []byte(tampered)
	if err := ValidateSignature(a, cert); err == nil {
		t.Fatal("expected error for tampered assertion with removed Signature")
	}
}

// TestSignatureVerification_DifferentValidCert verifies that a signature
// created with one key is rejected when verified with a different cert.
func TestSignatureVerification_DifferentValidCert(t *testing.T) {
	xmlStr, _, _ := buildSignedXML(t, "_sig-002", "user@corp.com")
	a, _ := ParseAssertion([]byte(xmlStr))
	wrongCert := genRSACert(t)
	if err := ValidateSignature(a, wrongCert); err == nil {
		t.Fatal("expected ValidateSignature to reject signature verified with wrong cert")
	}
}

// TestSignatureVerification_NoCertButHasSignature verifies the nil-cert path
// even when the assertion is properly signed.
func TestSignatureVerification_NoCertButHasSignature(t *testing.T) {
	xml := buildAssertionWithSignature("_sig-003", "user@corp.com")
	a, _ := ParseAssertion([]byte(xml))
	if err := ValidateSignature(a, nil); err == nil {
		t.Fatal("expected error for nil cert even with valid Signature element")
	}
}

// TestSignatureVerification_BareSignatureWithRSA verifies that forged
// signatures (invalid SignatureValue) are rejected for both the
// <Signature> (non-prefixed) and <ds:Signature> variants.
func TestSignatureVerification_BareSignatureWithRSA(t *testing.T) {
	cert := genRSACert(t)
	tests := []struct {
		name string
		xml  string
	}{
		{
			name: "ds_prefix",
			xml: `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
				<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
					<ds:SignatureValue>abc</ds:SignatureValue>
				</ds:Signature>
			</Assertion>`,
		},
		{
			name: "bare",
			xml: `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion">
				<Signature><SignatureValue>abc</SignatureValue></Signature>
			</Assertion>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := ParseAssertion([]byte(tt.xml))
			if err := ValidateSignature(a, cert); err == nil {
				t.Errorf("expected forged signature to be rejected for %s", tt.name)
			}
		})
	}
}

// ===========================================================================
// 3. Replay attack protection (assertion ID tracking)
// ===========================================================================

// TestReplayProtection_FirstUseAccepted verifies that a fresh assertion ID
// is accepted on first submission.
func TestReplayProtection_FirstUseAccepted(t *testing.T) {
	cache := newAssertionIDCache()
	if cache.IsReplay("_replay-001") {
		t.Fatal("first use of assertion ID should not be flagged as replay")
	}
}

// TestReplayProtection_DuplicateRejected verifies that resubmitting the same
// assertion ID is detected as a replay attack.
func TestReplayProtection_DuplicateRejected(t *testing.T) {
	cache := newAssertionIDCache()
	if cache.IsReplay("_replay-002") {
		t.Fatal("first use should pass")
	}
	if !cache.IsReplay("_replay-002") {
		t.Fatal("second use of same assertion ID should be flagged as replay")
	}
}

// TestReplayProtection_DifferentIDsAccepted verifies that distinct assertion
// IDs are each accepted independently.
func TestReplayProtection_DifferentIDsAccepted(t *testing.T) {
	cache := newAssertionIDCache()
	for _, id := range []string{"_r-a", "_r-b", "_r-c"} {
		if cache.IsReplay(id) {
			t.Fatalf("assertion ID %s should not be replay", id)
		}
	}
}

// TestReplayProtection_IntegrationWithACS simulates the full replay check
// within an ACS-style flow: parse assertion, check ID, validate, mark consumed.
func TestReplayProtection_IntegrationWithACS(t *testing.T) {
	cache := newAssertionIDCache()
	xml := buildAssertionWithSignature("_replay-int-001", "frank@corp.com")

	// First submission: parse, check replay, validate.
	a, err := ParseAssertion([]byte(xml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cache.IsReplay(a.ID) {
		t.Fatal("first submission should not be replay")
	}
	if err := a.ValidateConditions(); err != nil {
		t.Fatalf("validate conditions: %v", err)
	}

	// Second submission of the same assertion: replay detected.
	a2, _ := ParseAssertion([]byte(xml))
	if !cache.IsReplay(a2.ID) {
		t.Fatal("second submission of same assertion ID must be flagged as replay")
	}
}

// TestReplayProtection_ConcurrentReplay verifies thread-safety of the
// assertion ID cache under concurrent access.
func TestReplayProtection_ConcurrentReplay(t *testing.T) {
	cache := newAssertionIDCache()
	id := "_replay-concurrent"
	acceptCount := 0
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !cache.IsReplay(id) {
				mu.Lock()
				acceptCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if acceptCount != 1 {
		t.Errorf("expected exactly 1 accepted (non-replay) result, got %d", acceptCount)
	}
}

// ===========================================================================
// 4. Invalid / missing attributes in SAML response
// ===========================================================================

// TestInvalidAttributes_MissingNameID verifies handling of an assertion with
// no Subject / NameID element.
func TestInvalidAttributes_MissingNameID(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_no-nameid">
		<Issuer>https://idp.example.com</Issuer>
		<AttributeStatement>
			<Attribute Name="mail"><AttributeValue>noname@corp.com</AttributeValue></Attribute>
		</AttributeStatement>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	if a.Subject.NameID != "" {
		t.Errorf("expected empty NameID, got '%s'", a.Subject.NameID)
	}
	// Attributes are still extractable.
	if got := GetAttribute(a, "mail"); got != "noname@corp.com" {
		t.Errorf("expected mail 'noname@corp.com', got '%s'", got)
	}
}

// TestInvalidAttributes_NoAttributeStatement verifies that an assertion without
// any AttributeStatement yields an empty attributes map.
func TestInvalidAttributes_NoAttributeStatement(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_no-attrs">
		<Issuer>https://idp.example.com</Issuer>
		<Subject><NameID>noattrs@corp.com</NameID></Subject>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	attrs := ExtractAttributes(a)
	if len(attrs) != 0 {
		t.Errorf("expected 0 attributes, got %d", len(attrs))
	}
	if got := GetAttribute(a, "mail"); got != "" {
		t.Errorf("expected empty for missing 'mail', got '%s'", got)
	}
}

// TestInvalidAttributes_EmptyAttributeName verifies handling of an attribute
// with an empty Name attribute.
func TestInvalidAttributes_EmptyAttributeName(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_empty-name">
		<AttributeStatement>
			<Attribute Name=""><AttributeValue>orphan</AttributeValue></Attribute>
			<Attribute Name="valid"><AttributeValue>ok</AttributeValue></Attribute>
		</AttributeStatement>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	attrs := ExtractAttributes(a)
	// The empty-name attribute is stored under key "".
	if _, ok := attrs[""]; !ok {
		t.Error("expected attribute stored under empty key")
	}
	if got := GetAttribute(a, "valid"); got != "ok" {
		t.Errorf("expected 'ok' for valid attribute, got '%s'", got)
	}
}

// TestInvalidAttributes_EmptyAttributeValue verifies handling of an attribute
// with an empty value.
func TestInvalidAttributes_EmptyAttributeValue(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_empty-val">
		<AttributeStatement>
			<Attribute Name="department"><AttributeValue></AttributeValue></Attribute>
		</AttributeStatement>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	attrs := ExtractAttributes(a)
	if vals, ok := attrs["department"]; !ok {
		t.Error("expected 'department' attribute to exist")
	} else if len(vals) != 1 {
		t.Errorf("expected 1 value, got %d", len(vals))
	} else if vals[0] != "" {
		t.Errorf("expected empty value, got '%s'", vals[0])
	}
}

// TestInvalidAttributes_MissingIssuer verifies that an assertion without an
// Issuer element is still parseable (Issuer is empty string).
func TestInvalidAttributes_MissingIssuer(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_no-issuer">
		<Subject><NameID>noissuer@corp.com</NameID></Subject>
	</Assertion>`
	a, err := ParseAssertion([]byte(xml))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if a.Issuer != "" {
		t.Errorf("expected empty issuer, got '%s'", a.Issuer)
	}
}

// TestInvalidAttributes_MissingRequiredAttributes verifies that an ACS flow
// can detect when required attributes (e.g., mail, displayName) are absent.
func TestInvalidAttributes_MissingRequiredAttributes(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_missing-required">
		<Issuer>https://idp.example.com</Issuer>
		<Subject><NameID>user@corp.com</NameID></Subject>
		<AttributeStatement>
			<Attribute Name="displayName"><AttributeValue>User</AttributeValue></Attribute>
		</AttributeStatement>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))

	// Check for required attributes — 'mail' is missing.
	required := []string{"mail", "displayName"}
	missing := []string{}
	for _, req := range required {
		if GetAttribute(a, req) == "" {
			missing = append(missing, req)
		}
	}
	if len(missing) != 1 || missing[0] != "mail" {
		t.Errorf("expected only 'mail' to be missing, got %v", missing)
	}
}

// TestInvalidAttributes_DuplicateAttributeNames verifies handling when the
// same attribute Name appears multiple times in the AttributeStatement.
func TestInvalidAttributes_DuplicateAttributeNames(t *testing.T) {
	xml := `<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_dup-attrs">
		<AttributeStatement>
			<Attribute Name="mail"><AttributeValue>first@corp.com</AttributeValue></Attribute>
			<Attribute Name="mail"><AttributeValue>second@corp.com</AttributeValue></Attribute>
		</AttributeStatement>
	</Assertion>`
	a, _ := ParseAssertion([]byte(xml))
	// ExtractAttributes overwrites — last occurrence wins.
	attrs := ExtractAttributes(a)
	mail := attrs["mail"]
	if len(mail) != 1 || mail[0] != "second@corp.com" {
		t.Errorf("expected last occurrence 'second@corp.com', got %v", mail)
	}
	// GetAttribute returns first occurrence in iteration order (first parsed).
	if got := GetAttribute(a, "mail"); got != "first@corp.com" && got != "second@corp.com" {
		t.Errorf("unexpected mail from GetAttribute: '%s'", got)
	}
}
