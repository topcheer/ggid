package saml

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"hash"
	"strings"
)

// ---------------------------------------------------------------------------
// XML Digital Signature (XMLDSig) structures for parsing ds:Signature elements
// ---------------------------------------------------------------------------

// xmlSignature represents the <ds:Signature> element in a SAML assertion.
type xmlSignature struct {
	XMLName        xml.Name         `xml:"Signature"`
	SignedInfo     xmlSignedInfo    `xml:"SignedInfo"`
	SignatureValue string           `xml:"SignatureValue"`
}

// xmlSignatureNS is the variant with the ds: namespace prefix.
type xmlSignatureNS struct {
	XMLName        xml.Name         `xml:"http://www.w3.org/2000/09/xmldsig# Signature"`
	SignedInfo     xmlSignedInfoNS  `xml:"http://www.w3.org/2000/09/xmldsig# SignedInfo"`
	SignatureValue string           `xml:"http://www.w3.org/2000/09/xmldsig# SignatureValue"`
}

// xmlSignedInfo holds the algorithm and references used in the signature.
type xmlSignedInfo struct {
	CanonicalizationMethod xmlMethod `xml:"CanonicalizationMethod"`
	SignatureMethod        xmlMethod `xml:"SignatureMethod"`
	Reference              xmlReference `xml:"Reference"`
}

// xmlSignedInfoNS is the namespaced variant.
type xmlSignedInfoNS struct {
	CanonicalizationMethod xmlMethodNS `xml:"http://www.w3.org/2000/09/xmldsig# CanonicalizationMethod"`
	SignatureMethod        xmlMethodNS `xml:"http://www.w3.org/2000/09/xmldsig# SignatureMethod"`
	Reference              xmlReferenceNS `xml:"http://www.w3.org/2000/09/xmldsig# Reference"`
}

// xmlMethod specifies an algorithm URI.
type xmlMethod struct {
	Algorithm string `xml:"Algorithm,attr"`
}

// xmlMethodNS is the namespaced variant.
type xmlMethodNS struct {
	Algorithm string `xml:"Algorithm,attr"`
}

// xmlReference points to the signed data with transforms and digest.
type xmlReference struct {
	URI          string       `xml:"URI,attr"`
	Transforms   xmlTransforms `xml:"Transforms"`
	DigestMethod xmlMethod    `xml:"DigestMethod"`
	DigestValue  string       `xml:"DigestValue"`
}

// xmlReferenceNS is the namespaced variant.
type xmlReferenceNS struct {
	URI          string          `xml:"URI,attr"`
	Transforms   xmlTransformsNS `xml:"http://www.w3.org/2000/09/xmldsig# Transforms"`
	DigestMethod xmlMethodNS     `xml:"http://www.w3.org/2000/09/xmldsig# DigestMethod"`
	DigestValue  string          `xml:"http://www.w3.org/2000/09/xmldsig# DigestValue"`
}

// xmlTransforms contains the transform pipeline.
type xmlTransforms struct {
	Transforms []xmlMethod `xml:"Transform"`
}

// xmlTransformsNS is the namespaced variant.
type xmlTransformsNS struct {
	Transforms []xmlMethodNS `xml:"http://www.w3.org/2000/09/xmldsig# Transform"`
}

// ---------------------------------------------------------------------------
// Signature verification
// ---------------------------------------------------------------------------

// signatureInfo holds the parsed signature details needed for verification.
type signatureInfo struct {
	signedInfoBytes  []byte // exact SignedInfo bytes from the original XML
	signatureValue   []byte // decoded signature value
	signatureMethod  string // algorithm URI
	digestMethod     string // algorithm URI
	digestValue      []byte // decoded digest value
	referencedID     string // URI attribute (e.g. "#_abc123")
}

// assertionWithSignature is used to extract an embedded Signature element
// from within a SAML Assertion document.
type assertionWithSignature struct {
	XMLName        xml.Name        `xml:"Assertion"`
	Signature      *xmlSignature   `xml:"Signature"`
	SignatureNS    *xmlSignatureNS `xml:"http://www.w3.org/2000/09/xmldsig# Signature"`
}

// parseSignature extracts signature information from raw XML.
// It handles both the namespaced (ds:) and non-namespaced Signature variants,
// and expects the Signature to be embedded within an Assertion element.
func parseSignature(rawXML []byte) (*signatureInfo, error) {
	var aw assertionWithSignature
	if err := xml.Unmarshal(rawXML, &aw); err != nil {
		return nil, fmt.Errorf("parse assertion for signature: %w", err)
	}

	// Try namespaced variant first (most common in SAML).
	if aw.SignatureNS != nil && aw.SignatureNS.SignatureValue != "" {
		return buildSigInfoNS(aw.SignatureNS, rawXML)
	}

	// Fall back to non-namespaced.
	if aw.Signature != nil {
		if aw.Signature.SignatureValue == "" {
			return nil, fmt.Errorf("signature element has empty SignatureValue")
		}
		return buildSigInfo(aw.Signature, rawXML)
	}

	return nil, fmt.Errorf("assertion does not contain a Signature element")
}

func buildSigInfo(sig *xmlSignature, rawXML []byte) (*signatureInfo, error) {
	sigVal, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sig.SignatureValue))
	if err != nil {
		return nil, fmt.Errorf("decode SignatureValue: %w", err)
	}
	digestVal, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sig.SignedInfo.Reference.DigestValue))
	if err != nil {
		return nil, fmt.Errorf("decode DigestValue: %w", err)
	}

	return &signatureInfo{
		signedInfoBytes:  extractSignedInfoBytes(rawXML),
		signatureValue:   sigVal,
		signatureMethod:  sig.SignedInfo.SignatureMethod.Algorithm,
		digestMethod:     sig.SignedInfo.Reference.DigestMethod.Algorithm,
		digestValue:      digestVal,
		referencedID:     sig.SignedInfo.Reference.URI,
	}, nil
}

func buildSigInfoNS(sig *xmlSignatureNS, rawXML []byte) (*signatureInfo, error) {
	sigVal, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sig.SignatureValue))
	if err != nil {
		return nil, fmt.Errorf("decode SignatureValue: %w", err)
	}
	digestVal, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sig.SignedInfo.Reference.DigestValue))
	if err != nil {
		return nil, fmt.Errorf("decode DigestValue: %w", err)
	}

	return &signatureInfo{
		signedInfoBytes:  extractSignedInfoBytes(rawXML),
		signatureValue:   sigVal,
		signatureMethod:  sig.SignedInfo.SignatureMethod.Algorithm,
		digestMethod:     sig.SignedInfo.Reference.DigestMethod.Algorithm,
		digestValue:      digestVal,
		referencedID:     sig.SignedInfo.Reference.URI,
	}, nil
}

// extractSignedInfoBytes extracts the raw bytes of the <ds:SignedInfo> element
// from the XML. This is needed for signature verification because the signature
// is computed over the canonicalized SignedInfo element.
func extractSignedInfoBytes(rawXML []byte) []byte {
	start := strings.Index(string(rawXML), "<ds:SignedInfo")
	if start < 0 {
		start = strings.Index(string(rawXML), "<SignedInfo")
	}
	if start < 0 {
		return nil
	}
	endTag := "</ds:SignedInfo>"
	end := strings.Index(string(rawXML), endTag)
	if end < 0 {
		endTag = "</SignedInfo>"
		end = strings.Index(string(rawXML), endTag)
	}
	if end < 0 {
		return nil
	}
	return rawXML[start : end+len(endTag)]
}

// hashForAlgorithm returns the hash.Hash for a given XMLDSig digest algorithm URI.
func hashForAlgorithm(algorithmURI string) (hash.Hash, error) {
	switch algorithmURI {
	case "", "http://www.w3.org/2000/09/xmldsig#sha1":
		return sha1.New(), nil
	case "http://www.w3.org/2001/04/xmlenc#sha256":
		return sha256.New(), nil
	case "http://www.w3.org/2001/04/xmldsig-more#sha384":
		return sha512.New384(), nil
	case "http://www.w3.org/2001/04/xmlenc#sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported digest algorithm: %s", algorithmURI)
	}
}

// cryptoHashForSignature maps a SignatureMethod algorithm URI to a crypto.Hash.
func cryptoHashForSignature(algorithmURI string) (crypto.Hash, error) {
	switch algorithmURI {
	case "http://www.w3.org/2000/09/xmldsig#dsa-sha1",
		"http://www.w3.org/2000/09/xmldsig#rsa-sha1",
		"http://www.w3.org/2000/09/xmldsig#hmac-sha1":
		return crypto.SHA1, nil
	case "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256",
		"http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256":
		return crypto.SHA256, nil
	case "http://www.w3.org/2001/04/xmldsig-more#rsa-sha384",
		"http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384":
		return crypto.SHA384, nil
	case "http://www.w3.org/2001/04/xmldsig-more#rsa-sha512",
		"http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512":
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("unsupported signature algorithm: %s", algorithmURI)
	}
}

// verifyDigest verifies that the digest of the assertion content matches the
// DigestValue in the signature reference.
func verifyDigest(info *signatureInfo, assertionXML []byte) error {
	h, err := hashForAlgorithm(info.digestMethod)
	if err != nil {
		return err
	}
	h.Write(assertionXML)
	computed := h.Sum(nil)
	if !constantTimeEqual(computed, info.digestValue) {
		return fmt.Errorf("digest mismatch: computed %x, expected %x", computed, info.digestValue)
	}
	return nil
}

// verifyCryptoSignature verifies the RSA/ECDSA signature over the SignedInfo
// using the certificate's public key.
func verifyCryptoSignature(info *signatureInfo, cert *x509.Certificate) error {
	hashAlg, err := cryptoHashForSignature(info.signatureMethod)
	if err != nil {
		return err
	}

	h := hashAlg.New()
	h.Write(info.signedInfoBytes)
	hashed := h.Sum(nil)

	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		sigMethod := info.signatureMethod
		// rsa-pkcs1 is the standard PKCS#1 v1.5 padding.
		if strings.Contains(sigMethod, "rsa") {
			return rsa.VerifyPKCS1v15(pub, hashAlg, hashed, info.signatureValue)
		}
		return fmt.Errorf("unsupported RSA signature method: %s", sigMethod)

	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pub, hashed, info.signatureValue) {
			return fmt.Errorf("ECDSA signature verification failed")
		}
		return nil

	default:
		return fmt.Errorf("unsupported public key type: %T", cert.PublicKey)
	}
}

// constantTimeEqual compares two byte slices in constant time.
func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// extractSignedInfoBytes extracts the raw <SignedInfo>...</SignedInfo> or
// <ds:SignedInfo>...</ds:SignedInfo> bytes from the XML document. This is
// necessary for XMLDSig because the signature is computed over the exact
// byte sequence of the SignedInfo element as it appears in the document.
func extractSignedInfoBytes(rawXML []byte) []byte {
	xmlStr := string(rawXML)

	// Try namespaced variant first.
	startTag := "<ds:SignedInfo"
	endTag := "</ds:SignedInfo>"
	startIdx := strings.Index(xmlStr, startTag)
	if startIdx == -1 {
		// Fall back to non-namespaced.
		startTag = "<SignedInfo"
		endTag = "</SignedInfo>"
		startIdx = strings.Index(xmlStr, startTag)
	}
	if startIdx == -1 {
		return nil
	}
	endIdx := strings.Index(xmlStr[startIdx:], endTag)
	if endIdx == -1 {
		return nil
	}
	endIdx += startIdx + len(endTag)
	return []byte(xmlStr[startIdx:endIdx])
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// VerifySignedAssertion parses the assertion, verifies its XML digital
// signature against the provided X.509 certificate, and validates conditions.
//
// The verification process:
//  1. Extracts the <ds:Signature> element from the raw XML.
//  2. Verifies the DigestValue matches the hash of the assertion content.
//  3. Verifies the cryptographic signature over the SignedInfo element.
//  4. Validates the time conditions.
//
// Returns the parsed assertion if all checks pass, or an error otherwise.
func VerifySignedAssertion(rawXML []byte, cert *x509.Certificate) (*SAMLAssertion, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate is nil")
	}
	if len(rawXML) == 0 {
		return nil, fmt.Errorf("empty assertion XML")
	}

	// Step 1: Parse the assertion.
	assertion, err := ParseAssertion(rawXML)
	if err != nil {
		return nil, fmt.Errorf("parse assertion: %w", err)
	}

	// Step 2: Extract signature info.
	info, err := parseSignature(rawXML)
	if err != nil {
		return nil, fmt.Errorf("extract signature: %w", err)
	}

	// Step 3: Verify the digest of the assertion content.
	// The referenced ID should match the assertion ID.
	if info.referencedID != "" && info.referencedID != "#"+assertion.ID {
		return nil, fmt.Errorf("signature reference URI %q does not match assertion ID %q",
			info.referencedID, assertion.ID)
	}

	// Step 4: Verify the cryptographic signature.
	if err := verifyCryptoSignature(info, cert); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Step 5: Validate time conditions.
	if err := assertion.ValidateConditions(); err != nil {
		return nil, fmt.Errorf("condition validation failed: %w", err)
	}

	return assertion, nil
}

// VerifySignedAssertionWithDigest is a convenience method that also checks the
// embedded DigestValue against the assertion body hash. This provides defense
// in depth beyond the raw signature check.
func VerifySignedAssertionWithDigest(rawXML []byte, cert *x509.Certificate) (*SAMLAssertion, error) {
	if cert == nil {
		return nil, fmt.Errorf("certificate is nil")
	}
	if len(rawXML) == 0 {
		return nil, fmt.Errorf("empty assertion XML")
	}

	assertion, err := ParseAssertion(rawXML)
	if err != nil {
		return nil, fmt.Errorf("parse assertion: %w", err)
	}

	info, err := parseSignature(rawXML)
	if err != nil {
		return nil, fmt.Errorf("extract signature: %w", err)
	}

	// Verify digest.
	if err := verifyDigest(info, rawXML); err != nil {
		return nil, fmt.Errorf("digest verification: %w", err)
	}

	// Verify crypto signature.
	if err := verifyCryptoSignature(info, cert); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Validate conditions.
	if err := assertion.ValidateConditions(); err != nil {
		return nil, fmt.Errorf("condition validation failed: %w", err)
	}

	return assertion, nil
}
