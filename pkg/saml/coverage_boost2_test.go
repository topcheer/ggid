package saml

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

// --- generateValidDuration (0% → 100%) ---
func TestCovS2_GenerateValidDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{24 * time.Hour, "PT24H"},
		{0, "PT0H"},
		{90 * time.Hour, "PT90H"},
	}
	for _, tt := range tests {
		got := generateValidDuration(tt.d)
		if got != tt.want {
			t.Errorf("generateValidDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// --- flateCompress (60% → 100%) ---
func TestCovS2_FlateCompress_EmptyInput(t *testing.T) {
	result, err := flateCompress([]byte{})
	if err != nil {
		t.Fatalf("flateCompress(empty): %v", err)
	}
	if len(result) == 0 {
		t.Error("expected non-empty output for empty input")
	}
}

func TestCovS2_FlateCompress_RoundTrip(t *testing.T) {
	original := []byte("<saml:Assertion>test data</saml:Assertion>")
	compressed, err := flateCompress(original)
	if err != nil {
		t.Fatalf("flateCompress: %v", err)
	}
	decompressed, err := flateDecompress(compressed)
	if err != nil {
		t.Fatalf("flateDecompress: %v", err)
	}
	if string(decompressed) != string(original) {
		t.Errorf("round-trip mismatch")
	}
}

// --- hashForAlgorithm (83.3% → 100%) ---
func TestCovS2_HashForAlgorithm_Unsupported(t *testing.T) {
	_, err := hashForAlgorithm("http://unknown/alg")
	if err == nil {
		t.Error("expected error for unsupported algorithm")
	}
}

func TestCovS2_HashForAlgorithm_SHA384(t *testing.T) {
	h, err := hashForAlgorithm("http://www.w3.org/2001/04/xmldsig-more#sha384")
	if err != nil || h == nil {
		t.Errorf("SHA384: err=%v, h=%v", err, h)
	}
}

func TestCovS2_HashForAlgorithm_SHA512(t *testing.T) {
	h, err := hashForAlgorithm("http://www.w3.org/2001/04/xmlenc#sha512")
	if err != nil || h == nil {
		t.Errorf("SHA512: err=%v, h=%v", err, h)
	}
}

// --- cryptoHashForSignature (66.7% → 100%) ---
func TestCovS2_CryptoHashForSignature_Unsupported(t *testing.T) {
	_, err := cryptoHashForSignature("http://unknown/sig")
	if err == nil {
		t.Error("expected error for unsupported signature algorithm")
	}
}

func TestCovS2_CryptoHashForSignature_SHA384(t *testing.T) {
	h, err := cryptoHashForSignature("http://www.w3.org/2001/04/xmldsig-more#rsa-sha384")
	if err != nil {
		t.Fatalf("rsa-sha384: %v", err)
	}
	if h.Size() != 48 {
		t.Errorf("expected size 48, got %d", h.Size())
	}
}

func TestCovS2_CryptoHashForSignature_SHA512(t *testing.T) {
	_, err := cryptoHashForSignature("http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512")
	if err != nil {
		t.Fatalf("ecdsa-sha512: %v", err)
	}
}

// --- verifyDigest (0% → 100%) ---
func TestCovS2_VerifyDigest_Success(t *testing.T) {
	data := []byte("<test>content</test>")
	h := sha256.Sum256(data)
	info := &signatureInfo{
		digestMethod: "http://www.w3.org/2001/04/xmlenc#sha256",
		digestValue:  h[:],
	}
	if err := verifyDigest(info, data); err != nil {
		t.Fatalf("verifyDigest: %v", err)
	}
}

func TestCovS2_VerifyDigest_Mismatch(t *testing.T) {
	info := &signatureInfo{
		digestMethod: "http://www.w3.org/2001/04/xmlenc#sha256",
		digestValue:  []byte("wrong-digest"),
	}
	err := verifyDigest(info, []byte("test data"))
	if err == nil {
		t.Error("expected digest mismatch error")
	}
}

func TestCovS2_VerifyDigest_UnsupportedAlgorithm(t *testing.T) {
	info := &signatureInfo{
		digestMethod: "http://unknown/alg",
		digestValue:  []byte("x"),
	}
	err := verifyDigest(info, []byte("data"))
	if err == nil {
		t.Error("expected error for unsupported algorithm")
	}
}

func TestCovS2_VerifyDigest_DefaultSHA1(t *testing.T) {
	data := []byte("test")
	h, _ := hashForAlgorithm("")
	h.Write(data)
	info := &signatureInfo{
		digestMethod: "",
		digestValue:  h.Sum(nil),
	}
	if err := verifyDigest(info, data); err != nil {
		t.Fatalf("verifyDigest SHA1 default: %v", err)
	}
}

// --- VerifySignedAssertionWithDigest (0% → 100%) ---
func TestCovS2_VerifySignedAssertionWithDigest_NilCert(t *testing.T) {
	_, err := VerifySignedAssertionWithDigest([]byte("<test/>"), nil)
	if err == nil {
		t.Error("expected error for nil cert")
	}
}

func TestCovS2_VerifySignedAssertionWithDigest_EmptyXML(t *testing.T) {
	cert := genRSACert(t)
	_, err := VerifySignedAssertionWithDigest([]byte{}, cert)
	if err == nil {
		t.Error("expected error for empty XML")
	}
}

func TestCovS2_VerifySignedAssertionWithDigest_InvalidXML(t *testing.T) {
	cert := genRSACert(t)
	_, err := VerifySignedAssertionWithDigest([]byte("not xml"), cert)
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestCovS2_VerifySignedAssertionWithDigest_NoSignature(t *testing.T) {
	cert := genRSACert(t)
	xml := []byte(`<Assertion xmlns="urn:oasis:names:tc:SAML:2.0:assertion" ID="_123"><Issuer>idp</Issuer></Assertion>`)
	_, err := VerifySignedAssertionWithDigest(xml, cert)
	if err == nil {
		t.Error("expected error for assertion without signature")
	}
}

// --- extractSignedInfoBytes (64.3% → 100%) ---
func TestCovS2_ExtractSignedInfoBytes_NoSignedInfo(t *testing.T) {
	result := extractSignedInfoBytes([]byte("<root>no signed info</root>"))
	if result != nil {
		t.Error("expected nil for XML without SignedInfo")
	}
}

func TestCovS2_ExtractSignedInfoBytes_EmptyInput(t *testing.T) {
	result := extractSignedInfoBytes([]byte(""))
	if result != nil {
		t.Error("expected nil for empty input")
	}
}

func TestCovS2_ExtractSignedInfoBytes_NoClosingTag(t *testing.T) {
	xml := []byte("<root><ds:SignedInfo>data without closing</root>")
	result := extractSignedInfoBytes(xml)
	if result != nil {
		t.Error("expected nil for unclosed SignedInfo")
	}
}

// --- buildSigInfo error paths (71.4% → 100%) ---
func TestCovS2_BuildSigInfo_InvalidBase64SigValue(t *testing.T) {
	sig := &xmlSignature{SignatureValue: "!!!invalid!!!"}
	_, err := buildSigInfo(sig, []byte("<x/>"))
	if err == nil {
		t.Error("expected error for invalid base64 SignatureValue")
	}
}

func TestCovS2_BuildSigInfo_InvalidBase64DigestValue(t *testing.T) {
	sig := &xmlSignature{
		SignatureValue: base64.StdEncoding.EncodeToString([]byte("sig")),
		SignedInfo: xmlSignedInfo{
			Reference: xmlReference{DigestValue: "!!!invalid!!!"},
		},
	}
	_, err := buildSigInfo(sig, []byte("<x/>"))
	if err == nil {
		t.Error("expected error for invalid base64 DigestValue")
	}
}

// --- buildSigInfoNS (0% → 100%) ---
func TestCovS2_BuildSigInfoNS_Success(t *testing.T) {
	sigVal := base64.StdEncoding.EncodeToString([]byte("signature"))
	digestVal := base64.StdEncoding.EncodeToString([]byte("digest"))
	sig := &xmlSignatureNS{
		SignatureValue: sigVal,
		SignedInfo: xmlSignedInfoNS{
			SignatureMethod: xmlMethodNS{Algorithm: "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"},
			Reference: xmlReferenceNS{
				URI:          "#_abc",
				DigestMethod: xmlMethodNS{Algorithm: "http://www.w3.org/2001/04/xmlenc#sha256"},
				DigestValue:  digestVal,
			},
		},
	}
	info, err := buildSigInfoNS(sig, []byte("<ds:SignedInfo>test</ds:SignedInfo>"))
	if err != nil {
		t.Fatalf("buildSigInfoNS: %v", err)
	}
	if info.referencedID != "#_abc" {
		t.Errorf("expected '#_abc', got '%s'", info.referencedID)
	}
}

func TestCovS2_BuildSigInfoNS_InvalidBase64Sig(t *testing.T) {
	sig := &xmlSignatureNS{SignatureValue: "!!!invalid!!!"}
	_, err := buildSigInfoNS(sig, nil)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestCovS2_BuildSigInfoNS_InvalidBase64Digest(t *testing.T) {
	sig := &xmlSignatureNS{
		SignatureValue: base64.StdEncoding.EncodeToString([]byte("sig")),
		SignedInfo: xmlSignedInfoNS{
			Reference: xmlReferenceNS{DigestValue: "!!!invalid!!!"},
		},
	}
	_, err := buildSigInfoNS(sig, nil)
	if err == nil {
		t.Error("expected error for invalid base64 DigestValue")
	}
}

// --- generateID (75% → higher) ---
func TestCovS2_GenerateID_Format(t *testing.T) {
	id := generateID()
	if !strings.HasPrefix(id, "_") {
		t.Errorf("expected ID to start with '_', got '%s'", id)
	}
	id2 := generateID()
	if id == id2 {
		t.Error("expected unique IDs")
	}
}
