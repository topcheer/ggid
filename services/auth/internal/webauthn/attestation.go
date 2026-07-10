package webauthn

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

// ===========================================================================
// WebAuthn Attestation Format Verification (packed, none, fido-u2f, etc.)
// + AAGUID lookup for authenticator metadata
// ===========================================================================

// --- Attestation Format Verification ---

// VerifyNoneAttestation handles the "none" attestation format.
// Per spec, no attestation data is present — the authenticator is anonymous.
func VerifyNoneAttestation() error {
	return nil
}

// VerifyPackedAttestation verifies a packed attestation signature.
// Supports EC2 (ES256, alg=-7), RSA (RS256, alg=-257), and EdDSA (alg=-8).
func VerifyPackedAttestation(authData, clientDataHash []byte, alg int, sig, certBytes []byte) error {
	if len(certBytes) == 0 {
		return fmt.Errorf("packed attestation requires a certificate")
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("parse attestation certificate: %w", err)
	}

	signedData := append(authData, clientDataHash...)
	hash := sha256.Sum256(signedData)

	switch alg {
	case -7: // ES256
		pubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("expected ECDSA public key for alg -7")
		}
		if !ecdsa.VerifyASN1(pubKey, hash[:], sig) {
			return fmt.Errorf("ECDSA signature verification failed")
		}
	case -257: // RS256
		pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("expected RSA public key for alg -257")
		}
		if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], sig); err != nil {
			return fmt.Errorf("RSA signature verification failed: %w", err)
		}
	case -8: // EdDSA
		pubKey, ok := cert.PublicKey.(ed25519.PublicKey)
		if !ok {
			return fmt.Errorf("expected Ed25519 public key for alg -8")
		}
		if !ed25519.Verify(pubKey, signedData, sig) {
			return fmt.Errorf("Ed25519 signature verification failed")
		}
	default:
		return fmt.Errorf("unsupported packed attestation algorithm: %d", alg)
	}
	return nil
}

// VerifyAttestationFormat dispatches to the correct attestation verifier.
func VerifyAttestationFormat(format string, authData, clientDataHash []byte, alg int, sig, certBytes []byte) error {
	switch format {
	case "none":
		return VerifyNoneAttestation()
	case "packed":
		return VerifyPackedAttestation(authData, clientDataHash, alg, sig, certBytes)
	case "fido-u2f":
		return verifyFidoU2FAttestation(authData, clientDataHash, sig, certBytes)
	case "android-key":
		return verifyAndroidKeyAttestation(authData, clientDataHash, sig, certBytes)
	case "android-safetynet":
		return verifyAndroidSafetynetAttestation(authData, clientDataHash, sig, certBytes)
	case "tpm":
		return verifyTPMAttestation(authData, clientDataHash, sig, certBytes)
	case "apple":
		return verifyAppleAttestation(authData, clientDataHash, sig, certBytes)
	case "":
		return fmt.Errorf("missing attestation format")
	default:
		return fmt.Errorf("unsupported attestation format: %s", format)
	}
}

// ExtractAAGUIDFromAuthData extracts the 16-byte AAGUID from WebAuthn authData.
func ExtractAAGUIDFromAuthData(authData []byte) []byte {
	if len(authData) < 53 {
		return nil
	}
	if authData[32]&0x40 == 0 { // AT flag not set
		return nil
	}
	aaguid := make([]byte, 16)
	copy(aaguid, authData[37:53])
	return aaguid
}

// ParseSignCount extracts the signature counter from authenticator data.
func ParseSignCount(authData []byte) uint32 {
	if len(authData) < 37 {
		return 0
	}
	return binary.BigEndian.Uint32(authData[33:37])
}

// --- AAGUID Lookup ---

// AuthenticatorInfo holds metadata about a known authenticator model.
type AuthenticatorInfo struct {
	AAGUID       string
	Name         string
	Manufacturer string
	Algorithm    string
}

var (
	authenticatorDB   = make(map[string]*AuthenticatorInfo)
	authenticatorDBMu sync.RWMutex
)

func init() {
	RegisterAuthenticator("00000000-0000-0000-0000-000000000000", &AuthenticatorInfo{
		Name: "Anonymous", Manufacturer: "Unknown", Algorithm: "ES256",
	})
	RegisterAuthenticator("fbfc3007-154e-4ecc-8c0b-6e0243c0ed8c", &AuthenticatorInfo{
		Name: "Touch ID", Manufacturer: "Apple", Algorithm: "ES256",
	})
	RegisterAuthenticator("08987058-cadc-4b81-b6e1-30de50dcbe16", &AuthenticatorInfo{
		Name: "Windows Hello", Manufacturer: "Microsoft", Algorithm: "RS256",
	})
	RegisterAuthenticator("dd4fa67f-36c7-4814-b3a5-95b05c1e0c6b", &AuthenticatorInfo{
		Name: "Pixel", Manufacturer: "Google", Algorithm: "RS256",
	})
	RegisterAuthenticator("cb69481e-8ff7-4039-93ec-0a2729a154a8", &AuthenticatorInfo{
		Name: "YubiKey 5", Manufacturer: "Yubico", Algorithm: "ES256",
	})
}

// RegisterAuthenticator adds an authenticator to the metadata database.
func RegisterAuthenticator(aaguid string, info *AuthenticatorInfo) {
	authenticatorDBMu.Lock()
	defer authenticatorDBMu.Unlock()
	info.AAGUID = strings.ToLower(aaguid)
	authenticatorDB[strings.ToLower(aaguid)] = info
}

// LookupAAGUID retrieves authenticator metadata by AAGUID bytes.
func LookupAAGUID(aaguid []byte) *AuthenticatorInfo {
	if len(aaguid) != 16 {
		return nil
	}
	uuid := formatAAGUID(aaguid)
	authenticatorDBMu.RLock()
	defer authenticatorDBMu.RUnlock()
	return authenticatorDB[uuid]
}

// formatAAGUID converts 16 bytes to UUID string.
func formatAAGUID(aaguid []byte) string {
	if len(aaguid) != 16 {
		return hex.EncodeToString(aaguid)
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(aaguid[0:4]),
		hex.EncodeToString(aaguid[4:6]),
		hex.EncodeToString(aaguid[6:8]),
		hex.EncodeToString(aaguid[8:10]),
		hex.EncodeToString(aaguid[10:16]),
	)
}
