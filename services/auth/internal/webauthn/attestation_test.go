package webauthn

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func TestVerifyNoneAttestation(t *testing.T) {
	if err := VerifyNoneAttestation(); err != nil {
		t.Fatalf("expected nil: %v", err)
	}
}

func TestVerifyAttestationFormat_None(t *testing.T) {
	if err := VerifyAttestationFormat("none", nil, nil, 0, nil, nil); err != nil {
		t.Fatalf("none should pass: %v", err)
	}
}

func TestVerifyAttestationFormat_FidoU2F(t *testing.T) {
	if err := VerifyAttestationFormat("fido-u2f", nil, nil, 0, nil, nil); err != nil {
		t.Fatalf("fido-u2f should pass: %v", err)
	}
}

func TestVerifyAttestationFormat_Apple(t *testing.T) {
	if err := VerifyAttestationFormat("apple", nil, nil, 0, nil, nil); err != nil {
		t.Fatalf("apple should pass: %v", err)
	}
}

func TestVerifyAttestationFormat_Empty(t *testing.T) {
	if err := VerifyAttestationFormat("", nil, nil, 0, nil, nil); err == nil {
		t.Fatal("expected error for empty format")
	}
}

func TestVerifyAttestationFormat_Unknown(t *testing.T) {
	if err := VerifyAttestationFormat("bogus", nil, nil, 0, nil, nil); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestVerifyPackedAttestation_EmptyCert(t *testing.T) {
	err := VerifyPackedAttestation(nil, nil, -7, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty cert")
	}
}

func TestVerifyPackedAttestation_BadCert(t *testing.T) {
	err := VerifyPackedAttestation(nil, nil, -7, nil, []byte{0x00, 0x01})
	if err == nil {
		t.Fatal("expected error for bad cert")
	}
}

func TestVerifyPackedAttestation_EC2(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"},
		NotBefore: time.Now(), NotAfter: time.Now().Add(1 * time.Hour),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)

	authData := make([]byte, 37)
	clientDataHash := sha256.Sum256([]byte("test"))
	signedData := append(authData, clientDataHash[:]...)
	hash := sha256.Sum256(signedData)
	sig, _ := ecdsa.SignASN1(rand.Reader, privKey, hash[:])

	err := VerifyPackedAttestation(authData, clientDataHash[:], -7, sig, certDER)
	if err != nil {
		t.Fatalf("EC2 packed: %v", err)
	}
}

func TestVerifyPackedAttestation_RSA(t *testing.T) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "rsa-test"},
		NotBefore: time.Now(), NotAfter: time.Now().Add(1 * time.Hour),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)

	authData := make([]byte, 37)
	clientDataHash := sha256.Sum256([]byte("test"))
	signedData := append(authData, clientDataHash[:]...)
	hash := sha256.Sum256(signedData)
	sig, _ := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, hash[:])

	err := VerifyPackedAttestation(authData, clientDataHash[:], -257, sig, certDER)
	if err != nil {
		t.Fatalf("RSA packed: %v", err)
	}
}

func TestVerifyPackedAttestation_BadSig(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"},
		NotBefore: time.Now(), NotAfter: time.Now().Add(1 * time.Hour),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)

	err := VerifyPackedAttestation(nil, nil, -7, []byte{0x00}, certDER)
	if err == nil {
		t.Fatal("expected error for bad signature")
	}
}

func TestVerifyPackedAttestation_UnsupportedAlg(t *testing.T) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"},
		NotBefore: time.Now(), NotAfter: time.Now().Add(1 * time.Hour),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)

	err := VerifyPackedAttestation(nil, nil, -999, nil, certDER)
	if err == nil {
		t.Fatal("expected error for unsupported alg")
	}
}

func TestLookupAAGUID_TouchID(t *testing.T) {
	aaguid := []byte{0xfb, 0xfc, 0x30, 0x07, 0x15, 0x4e, 0x4e, 0xcc, 0x8c, 0x0b, 0x6e, 0x02, 0x43, 0xc0, 0xed, 0x8c}
	info := LookupAAGUID(aaguid)
	if info == nil {
		t.Fatal("expected Touch ID")
	}
	if info.Name != "Touch ID" {
		t.Errorf("name = %s", info.Name)
	}
}

func TestLookupAAGUID_WindowsHello(t *testing.T) {
	aaguid := []byte{0x08, 0x98, 0x70, 0x58, 0xca, 0xdc, 0x4b, 0x81, 0xb6, 0xe1, 0x30, 0xde, 0x50, 0xdc, 0xbe, 0x16}
	info := LookupAAGUID(aaguid)
	if info == nil {
		t.Fatal("expected Windows Hello")
	}
}

func TestLookupAAGUID_Anonymous(t *testing.T) {
	info := LookupAAGUID(make([]byte, 16))
	if info == nil {
		t.Fatal("expected anonymous authenticator")
	}
}

func TestLookupAAGUID_Unknown(t *testing.T) {
	aaguid := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	if LookupAAGUID(aaguid) != nil {
		t.Fatal("expected nil for unknown")
	}
}

func TestLookupAAGUID_Nil(t *testing.T) {
	if LookupAAGUID(nil) != nil {
		t.Fatal("expected nil")
	}
}

func TestLookupAAGUID_Short(t *testing.T) {
	if LookupAAGUID([]byte{0x01}) != nil {
		t.Fatal("expected nil for short input")
	}
}

func TestRegisterAuthenticator(t *testing.T) {
	aaguid := []byte{0xaa, 0xaa, 0xaa, 0xaa, 0xbb, 0xbb, 0xcc, 0xcc, 0xdd, 0xdd, 0xee, 0xee, 0xee, 0xee, 0xee, 0xee}
	RegisterAuthenticator("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", &AuthenticatorInfo{
		Name: "Test Key", Manufacturer: "TestCorp",
	})
	info := LookupAAGUID(aaguid)
	if info == nil {
		t.Fatal("expected registered authenticator")
	}
}

func TestFormatAAGUID(t *testing.T) {
	aaguid := []byte{0xfb, 0xfc, 0x30, 0x07, 0x15, 0x4e, 0x4e, 0xcc, 0x8c, 0x0b, 0x6e, 0x02, 0x43, 0xc0, 0xed, 0x8c}
	result := formatAAGUID(aaguid)
	expected := "fbfc3007-154e-4ecc-8c0b-6e0243c0ed8c"
	if result != expected {
		t.Errorf("formatAAGUID = %s, want %s", result, expected)
	}
}

func TestFormatAAGUID_Short(t *testing.T) {
	result := formatAAGUID([]byte{0x01, 0x02})
	if result == "" {
		t.Error("expected non-empty")
	}
}

func TestExtractAAGUIDFromAuthData_Valid(t *testing.T) {
	authData := make([]byte, 53)
	authData[32] = 0x40 // AT flag
	copy(authData[37:53], []byte{0xfb, 0xfc, 0x30, 0x07, 0x15, 0x4e, 0x4e, 0xcc, 0x8c, 0x0b, 0x6e, 0x02, 0x43, 0xc0, 0xed, 0x8c})
	aaguid := ExtractAAGUIDFromAuthData(authData)
	if aaguid == nil {
		t.Fatal("expected non-nil")
	}
}

func TestExtractAAGUIDFromAuthData_NoATFlag(t *testing.T) {
	authData := make([]byte, 53)
	authData[32] = 0x00
	if ExtractAAGUIDFromAuthData(authData) != nil {
		t.Fatal("expected nil when AT flag not set")
	}
}

func TestExtractAAGUIDFromAuthData_Short(t *testing.T) {
	if ExtractAAGUIDFromAuthData([]byte{0x01}) != nil {
		t.Fatal("expected nil for short data")
	}
}

func TestParseSignCount(t *testing.T) {
	authData := make([]byte, 37)
	authData[33] = 0x00
	authData[34] = 0x00
	authData[35] = 0x01
	authData[36] = 0x2C
	if count := ParseSignCount(authData); count != 300 {
		t.Errorf("signCount = %d, want 300", count)
	}
}

func TestParseSignCount_Short(t *testing.T) {
	if count := ParseSignCount([]byte{0x01}); count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}
