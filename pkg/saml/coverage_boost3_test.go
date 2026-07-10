package saml

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

// --- parseSignature additional coverage (70% → higher) ---

func TestCovS3_ParseSignature_InvalidXML(t *testing.T) {
	_, err := parseSignature([]byte("<<<invalid xml>>>"))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestCovS3_ParseSignature_NonNamespaced_EmptySigValue(t *testing.T) {
	xml := []byte(`<Assertion><Signature><SignatureValue></SignatureValue></Signature></Assertion>`)
	_, err := parseSignature(xml)
	if err == nil {
		t.Error("expected error for empty SignatureValue")
	}
}

func TestCovS3_ParseSignature_NonNamespaced_Success(t *testing.T) {
	sigVal := base64.StdEncoding.EncodeToString([]byte("sig"))
	digestVal := base64.StdEncoding.EncodeToString([]byte("digest"))
	xml := []byte(`<Assertion>
  <Signature>
    <SignedInfo>
      <SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <Reference URI="#_abc">
        <DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <DigestValue>` + digestVal + `</DigestValue>
      </Reference>
    </SignedInfo>
    <SignatureValue>` + sigVal + `</SignatureValue>
  </Signature>
</Assertion>`)
	info, err := parseSignature(xml)
	if err != nil {
		t.Fatalf("parseSignature non-NS: %v", err)
	}
	if info.referencedID != "#_abc" {
		t.Errorf("expected '#_abc', got '%s'", info.referencedID)
	}
}

// --- verifyCryptoSignature additional coverage (60% → higher) ---

func TestCovS3_VerifyCryptoSignature_ECDSA_InvalidSig(t *testing.T) {
	cert := genECDSACert(t)
	info := &signatureInfo{
		signedInfoBytes: []byte("<ds:SignedInfo>test</ds:SignedInfo>"),
		signatureValue:  []byte("invalid-signature-bytes"),
		signatureMethod: "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256",
	}
	err := verifyCryptoSignature(info, cert)
	if err == nil {
		t.Error("expected ECDSA verification failure")
	}
}

func TestCovS3_VerifyCryptoSignature_RSA_NonRSAMethod(t *testing.T) {
	// RSA cert with recognized (non-rsa) algorithm → "unsupported RSA signature method"
	cert := genRSACert(t)
	info := &signatureInfo{
		signedInfoBytes: []byte("<ds:SignedInfo>test</ds:SignedInfo>"),
		signatureValue:  []byte("sig"),
		signatureMethod: "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256",
	}
	err := verifyCryptoSignature(info, cert)
	if err == nil {
		t.Error("expected error for RSA cert with non-rsa method")
	}
}

func TestCovS3_VerifyCryptoSignature_UnsupportedAlgorithm(t *testing.T) {
	cert := genRSACert(t)
	info := &signatureInfo{
		signedInfoBytes: []byte("data"),
		signatureValue:  []byte("sig"),
		signatureMethod: "http://unknown/algorithm",
	}
	err := verifyCryptoSignature(info, cert)
	if err == nil {
		t.Error("expected error for unsupported signature algorithm")
	}
}

// --- VerifySignedAssertion additional coverage (88.2% → higher) ---

func TestCovS3_VerifySignedAssertion_NilCert(t *testing.T) {
	_, err := VerifySignedAssertion([]byte("<x/>"), nil)
	if err == nil {
		t.Error("expected error for nil cert")
	}
}

func TestCovS3_VerifySignedAssertion_EmptyXML(t *testing.T) {
	cert := genRSACert(t)
	_, err := VerifySignedAssertion([]byte{}, cert)
	if err == nil {
		t.Error("expected error for empty XML")
	}
}

func TestCovS3_VerifySignedAssertion_ReferenceMismatch(t *testing.T) {
	cert := genRSACert(t)
	sigVal := base64.StdEncoding.EncodeToString([]byte("sig"))
	digestVal := base64.StdEncoding.EncodeToString([]byte("digest"))
	xml := []byte(`<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_correct">
  <Issuer>https://idp.example.com</Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_wrong">
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>` + digestVal + `</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>` + sigVal + `</ds:SignatureValue>
  </ds:Signature>
  <Subject><NameID>user@example.com</NameID></Subject>
</Assertion>`)
	_, err := VerifySignedAssertion(xml, cert)
	if err == nil {
		t.Error("expected error for reference URI mismatch")
	}
}

// --- VerifySignedAssertionWithDigest additional coverage (58.8% → higher) ---

func TestCovS3_VerifySignedAssertionWithDigest_DigestMismatch(t *testing.T) {
	cert := genRSACert(t)
	sigVal := base64.StdEncoding.EncodeToString([]byte("sig"))
	digestVal := base64.StdEncoding.EncodeToString([]byte("wrong-digest"))
	xml := []byte(`<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc">
  <Issuer>https://idp.example.com</Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_abc">
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>` + digestVal + `</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>` + sigVal + `</ds:SignatureValue>
  </ds:Signature>
  <Subject><NameID>user@example.com</NameID></Subject>
</Assertion>`)
	_, err := VerifySignedAssertionWithDigest(xml, cert)
	if err == nil {
		t.Error("expected digest mismatch error")
	}
}

func TestCovS3_VerifySignedAssertionWithDigest_SigFailure(t *testing.T) {
	cert := genRSACert(t)
	sigVal := base64.StdEncoding.EncodeToString([]byte("bad-sig"))
	digestVal := base64.StdEncoding.EncodeToString([]byte("wrong"))
	xml := []byte(`<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_abc">
  <Issuer>idp</Issuer>
  <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
    <ds:SignedInfo>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference URI="#_abc">
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>` + digestVal + `</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>` + sigVal + `</ds:SignatureValue>
  </ds:Signature>
</Assertion>`)
	_, err := VerifySignedAssertionWithDigest(xml, cert)
	if err == nil {
		t.Error("expected error (digest or signature failure)")
	}
}

// --- EncodeForRedirect success path (71.4% → higher) ---

func TestCovS3_EncodeForRedirect_Success(t *testing.T) {
	sp := &ServiceProvider{
		EntityID: "https://sp.example.com",
		ACSURL:   "https://sp.example.com/acs",
	}
	req := BuildAuthnRequest(sp, "https://idp.example.com/sso")
	encoded, err := req.EncodeForRedirect()
	if err != nil {
		t.Fatalf("EncodeForRedirect: %v", err)
	}
	if encoded == "" {
		t.Error("expected non-empty encoded string")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("encoded result is not valid base64: %v", err)
	}
	xmlBytes, err := flateDecompress(decoded)
	if err != nil {
		t.Fatalf("flateDecompress: %v", err)
	}
	if !strings.Contains(string(xmlBytes), "AuthnRequest") {
		t.Error("expected XML to contain AuthnRequest")
	}
}

// --- Marshal success path (80% → higher) ---

func TestCovS3_Marshal_Success(t *testing.T) {
	req := &AuthnRequest{
		ID:      "_test123",
		Version: "2.0",
		Issuer:  Issuer{Value: "https://sp.example.com"},
	}
	data, err := req.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), "_test123") {
		t.Error("expected XML to contain ID")
	}
	if !strings.Contains(string(data), `<?xml`) {
		t.Error("expected XML header")
	}
}

// --- GenerateSPMetadata error paths (94.4% → higher) ---

func TestCovS3_GenerateSPMetadata_NilSP(t *testing.T) {
	_, err := GenerateSPMetadata(nil)
	if err == nil {
		t.Error("expected error for nil SP")
	}
}

func TestCovS3_GenerateSPMetadata_EmptyEntityID(t *testing.T) {
	_, err := GenerateSPMetadata(&ServiceProvider{
		ACSURL: "https://sp.example.com/acs",
	})
	if err == nil {
		t.Error("expected error for empty entity ID")
	}
}

func TestCovS3_GenerateSPMetadata_EmptyACSURL(t *testing.T) {
	_, err := GenerateSPMetadata(&ServiceProvider{
		EntityID: "https://sp.example.com",
	})
	if err == nil {
		t.Error("expected error for empty ACS URL")
	}
}

func TestCovS3_GenerateSPMetadata_InvalidCert(t *testing.T) {
	_, err := GenerateSPMetadata(&ServiceProvider{
		EntityID:        "https://sp.example.com",
		ACSURL:          "https://sp.example.com/acs",
		X509Certificate: []byte("not-a-valid-cert"),
	})
	if err == nil {
		t.Error("expected error for invalid cert")
	}
}

func TestCovS3_GenerateSPMetadata_WithSLO(t *testing.T) {
	sp := &ServiceProvider{
		EntityID: "https://sp.example.com",
		ACSURL:   "https://sp.example.com/acs",
		SLOURL:   "https://sp.example.com/slo",
	}
	data, err := GenerateSPMetadata(sp)
	if err != nil {
		t.Fatalf("GenerateSPMetadata with SLO: %v", err)
	}
	if !strings.Contains(string(data), "SingleLogoutService") {
		t.Error("expected metadata to contain SingleLogoutService")
	}
}

// --- constantTimeEqual edge cases ---

func TestCovS3_ConstantTimeEqual_DifferentLengths(t *testing.T) {
	if constantTimeEqual([]byte("a"), []byte("ab")) {
		t.Error("expected false for different lengths")
	}
}

func TestCovS3_ConstantTimeEqual_SameData(t *testing.T) {
	if !constantTimeEqual([]byte("abc"), []byte("abc")) {
		t.Error("expected true for same data")
	}
}

func TestCovS3_ConstantTimeEqual_DifferentData(t *testing.T) {
	if constantTimeEqual([]byte("abc"), []byte("abd")) {
		t.Error("expected false for different data")
	}
}

// --- cryptoHashForSignature SHA1 ---

func TestCovS3_CryptoHashForSignature_RSA_SHA1(t *testing.T) {
	h, err := cryptoHashForSignature("http://www.w3.org/2000/09/xmldsig#rsa-sha1")
	if err != nil {
		t.Fatalf("rsa-sha1: %v", err)
	}
	if h.Size() != 20 {
		t.Errorf("expected SHA1 size 20, got %d", h.Size())
	}
}

func TestCovS3_CryptoHashForSignature_ECDSA_SHA256(t *testing.T) {
	h, err := cryptoHashForSignature("http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256")
	if err != nil {
		t.Fatalf("ecdsa-sha256: %v", err)
	}
	if h.Size() != 32 {
		t.Errorf("expected SHA256 size 32, got %d", h.Size())
	}
}

// --- generateID uniqueness ---

func TestCovS3_GenerateID_MultipleUnique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		if ids[id] {
			t.Fatalf("duplicate ID: %s", id)
		}
		ids[id] = true
		if !strings.HasPrefix(id, "_") {
			t.Errorf("ID prefix: %s", id)
		}
	}
}

// --- BuildAuthnRequest completeness ---

func TestCovS3_BuildAuthnRequest_Fields(t *testing.T) {
	sp := &ServiceProvider{
		EntityID: "https://sp.example.com",
		ACSURL:   "https://sp.example.com/acs",
	}
	req := BuildAuthnRequest(sp, "https://idp.example.com/sso")
	if req.ID == "" {
		t.Error("expected non-empty ID")
	}
	if req.Version != "2.0" {
		t.Errorf("expected version 2.0, got %s", req.Version)
	}
	if req.Destination != "https://idp.example.com/sso" {
		t.Errorf("unexpected destination: %s", req.Destination)
	}
	if _, err := time.Parse(time.RFC3339, req.IssueInstant); err != nil {
		t.Errorf("IssueInstant not valid RFC3339: %s", req.IssueInstant)
	}
}

// --- flateCompress/decompress large data ---

func TestCovS3_FlateCompress_LargeData(t *testing.T) {
	original := make([]byte, 10000)
	for i := range original {
		original[i] = byte(i % 256)
	}
	compressed, err := flateCompress(original)
	if err != nil {
		t.Fatalf("flateCompress: %v", err)
	}
	decompressed, err := flateDecompress(compressed)
	if err != nil {
		t.Fatalf("flateDecompress: %v", err)
	}
	if len(decompressed) != len(original) {
		t.Errorf("length mismatch: %d vs %d", len(decompressed), len(original))
	}
}

// --- extractSignedInfoBytes bare SignedInfo ---

func TestCovS3_ExtractSignedInfoBytes_BareSignedInfo(t *testing.T) {
	xml := []byte(`<root><SignedInfo>content</SignedInfo></root>`)
	result := extractSignedInfoBytes(xml)
	if result == nil {
		t.Error("expected non-nil for bare SignedInfo")
	}
}

func TestCovS3_ExtractSignedInfoBytes_BareSignedInfo_NoClose(t *testing.T) {
	xml := []byte(`<root><SignedInfo>data</root>`)
	result := extractSignedInfoBytes(xml)
	if result != nil {
		t.Error("expected nil for unclosed bare SignedInfo")
	}
}

// --- SHA256 unused import fix ---

var _ = sha256.New
