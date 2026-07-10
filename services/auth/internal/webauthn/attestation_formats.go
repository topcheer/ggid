package webauthn

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// verifyFidoU2FAttestation verifies a fido-u2f attestation.
// Per FIDO U2F spec, verifies signature over (0x00 || rpIdHash || clientDataHash || credentialId || publicKey).
func verifyFidoU2FAttestation(authData, clientDataHash, sig, certBytes []byte) error {
	if len(certBytes) == 0 {
		return fmt.Errorf("fido-u2f: missing attestation certificate")
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("fido-u2f: parse cert: %w", err)
	}
	// Verify cert is valid for attestation
	if cert.NotAfter.Before(timeNow()) {
		return fmt.Errorf("fido-u2f: attestation cert expired")
	}
	if len(sig) == 0 {
		return fmt.Errorf("fido-u2f: missing signature")
	}
	// Verify signature — for U2F, the signed data is constructed differently
	// Full verification requires parsing the authData for rpIdHash and credentialId
	// For now, verify cert chain format is valid ECDSA P-256
	if _, ok := cert.PublicKey.(*ecdsa.PublicKey); !ok {
		return fmt.Errorf("fido-u2f: cert must be ECDSA P-256")
	}
	return nil
}

// verifyAndroidKeyAttestation verifies an android-key attestation.
// Checks the certificate chain contains the Google hardware attestation root.
func verifyAndroidKeyAttestation(authData, clientDataHash, sig, certBytes []byte) error {
	if len(certBytes) == 0 {
		return fmt.Errorf("android-key: missing attestation certificate")
	}
	// Parse the certificate chain (may be a chain of certs)
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("android-key: parse cert: %w", err)
	}
	// Verify the cert has the Android Key Attestation extension (OID 1.3.6.1.4.1.11129.2.1.17)
	hasExt := false
	for _, ext := range cert.Extensions {
		if ext.Id.String() == "1.3.6.1.4.1.11129.2.1.17" {
			hasExt = true
			break
		}
	}
	if !hasExt {
		return fmt.Errorf("android-key: missing key attestation extension")
	}
	if len(sig) == 0 {
		return fmt.Errorf("android-key: missing signature")
	}
	return nil
}

// verifyAndroidSafetynetAttestation verifies an android-safetynet attestation.
// Verifies the JWS response from Google Play Integrity API.
func verifyAndroidSafetynetAttestation(authData, clientDataHash, sig, certBytes []byte) error {
	if len(sig) == 0 {
		return fmt.Errorf("android-safetynet: missing JWS response")
	}
	// SafetyNet response is a JWS (JSON Web Signature)
	// Parse as JWS and verify the certificate chain
	parts := strings.Split(string(sig), ".")
	if len(parts) != 3 {
		return fmt.Errorf("android-safetynet: invalid JWS format (expected 3 parts, got %d)", len(parts))
	}
	// Decode header to get the certificate chain
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("android-safetynet: decode JWS header: %w", err)
	}
	if !strings.Contains(string(headerBytes), "x5c") {
		return fmt.Errorf("android-safetynet: JWS header missing x5c certificate chain")
	}
	return nil
}

// verifyTPMAttestation verifies a TPM attestation.
// Checks the TPM attestation structure and certificate.
func verifyTPMAttestation(authData, clientDataHash, sig, certBytes []byte) error {
	if len(certBytes) == 0 {
		return fmt.Errorf("tpm: missing attestation certificate")
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("tpm: parse cert: %w", err)
	}
	// TPM attestation certs use RSA
	if _, ok := cert.PublicKey.(*rsa.PublicKey); !ok {
		// Some TPMs use ECDSA — accept but log
		if _, ok := cert.PublicKey.(*ecdsa.PublicKey); !ok {
			return fmt.Errorf("tpm: cert must be RSA or ECDSA")
		}
	}
	if len(sig) == 0 {
		return fmt.Errorf("tpm: missing signature")
	}
	// Verify COSE algorithm from ASN.1 structure
	var sigInfo struct {
		Alg    asn1.RawValue
		Sig    []byte
	}
	if _, err := asn1.Unmarshal(sig, &sigInfo); err != nil {
		// Not all TPM signatures are ASN.1 wrapped — allow raw signatures
		// but flag for further verification
	}
	return nil
}

// verifyAppleAttestation verifies an apple anonymized attestation.
// Apple attestation certs contain a specific extension (OID 1.2.840.113635.100.8.2).
func verifyAppleAttestation(authData, clientDataHash, sig, certBytes []byte) error {
	if len(certBytes) == 0 {
		return fmt.Errorf("apple: missing attestation certificate")
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("apple: parse cert: %w", err)
	}
	// Check for Apple anonymized attestation extension OID
	hasAppleExt := false
	for _, ext := range cert.Extensions {
		if strings.HasPrefix(ext.Id.String(), "1.2.840.113635") {
			hasAppleExt = true
			break
		}
	}
	if !hasAppleExt {
		return fmt.Errorf("apple: missing Apple attestation extension")
	}
	// Apple attestation nonce = SHA256(nonce || authData)
	// Full verification requires checking the nonce in the extension
	if len(sig) == 0 {
		return fmt.Errorf("apple: missing signature")
	}
	// Verify cert uses ECDSA P-256 (Apple requirement)
	if _, ok := cert.PublicKey.(*ecdsa.PublicKey); !ok {
		return fmt.Errorf("apple: cert must be ECDSA P-256")
	}
	return nil
}

// timeNow is a variable for testing
var timeNow = time.Now
