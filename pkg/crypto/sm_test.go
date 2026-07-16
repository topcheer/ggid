package crypto

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestSM4EncryptDecrypt_Roundtrip(t *testing.T) {
	key := []byte("0123456789abcdef")
	plaintext := []byte("GGID 国密 SM4-GCM test payload")

	ct, err := SM4Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("SM4Encrypt: %v", err)
	}
	if string(ct) == string(plaintext) {
		t.Fatal("ciphertext equals plaintext")
	}

	pt, err := SM4Decrypt(ct, key)
	if err != nil {
		t.Fatalf("SM4Decrypt: %v", err)
	}
	if string(pt) != string(plaintext) {
		t.Fatalf("roundtrip mismatch: got %q", pt)
	}
}

func TestSM4Decrypt_WrongKey(t *testing.T) {
	ct, err := SM4Encrypt([]byte("secret"), []byte("key-one-12345678"))
	if err != nil {
		t.Fatalf("SM4Encrypt: %v", err)
	}
	if _, err := SM4Decrypt(ct, []byte("key-two-12345678")); err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestSM4Decrypt_Tampered(t *testing.T) {
	key := []byte("0123456789abcdef")
	ct, err := SM4Encrypt([]byte("integrity check"), key)
	if err != nil {
		t.Fatalf("SM4Encrypt: %v", err)
	}
	ct[len(ct)-1] ^= 0xff
	if _, err := SM4Decrypt(ct, key); err == nil {
		t.Fatal("expected GCM tag verification failure on tampered ciphertext")
	}
}

func TestSM4_ArbitraryKeyLength(t *testing.T) {
	// Keys of non-16-byte length are normalized via hash.
	ct, err := SM4Encrypt([]byte("data"), []byte("short"))
	if err != nil {
		t.Fatalf("SM4Encrypt with short key: %v", err)
	}
	pt, err := SM4Decrypt(ct, []byte("short"))
	if err != nil || string(pt) != "data" {
		t.Fatalf("roundtrip with short key failed: %v %q", err, pt)
	}
}

func TestSM2KeyProvider_Generate(t *testing.T) {
	p, err := newSM2KeyProvider(SM2KeyProviderConfig{Generate: true, KeyID: "test-sm2"})
	if err != nil {
		t.Fatalf("newSM2KeyProvider: %v", err)
	}
	if p.Metadata().Algorithm != SM2SM3 {
		t.Fatalf("algorithm = %v, want SM2SM3", p.Metadata().Algorithm)
	}
	if p.Metadata().KeyID != "test-sm2" {
		t.Fatalf("keyID = %v", p.Metadata().KeyID)
	}
	if p.Public() == nil || p.Signer() == nil {
		t.Fatal("Public/Signer must not be nil")
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestSM2KeyProvider_PEMRoundtrip(t *testing.T) {
	priv, err := GenerateSM2KeyPair()
	if err != nil {
		t.Fatalf("GenerateSM2KeyPair: %v", err)
	}
	privPEM, err := MarshalSM2PrivateKeyPEM(priv)
	if err != nil {
		t.Fatalf("MarshalSM2PrivateKeyPEM: %v", err)
	}

	dir := t.TempDir()
	privPath := filepath.Join(dir, "sm2.pem")
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		t.Fatalf("write PEM: %v", err)
	}

	p, err := newSM2KeyProvider(SM2KeyProviderConfig{PrivateKeyPath: privPath})
	if err != nil {
		t.Fatalf("newSM2KeyProvider from PEM: %v", err)
	}
	if p.Metadata().Algorithm != SM2SM3 {
		t.Fatalf("algorithm = %v", p.Metadata().Algorithm)
	}
}

func TestSM2KeyProvider_RequiresConfig(t *testing.T) {
	if _, err := newSM2KeyProvider(SM2KeyProviderConfig{}); err == nil {
		t.Fatal("expected error when neither path nor generate is set")
	}
}

func TestNewKeyProvider_SM2Factory(t *testing.T) {
	kp, err := NewKeyProvider(t.Context(), KeyProviderConfig{
		Provider: "sm2",
		SM2:      SM2KeyProviderConfig{Generate: true},
	})
	if err != nil {
		t.Fatalf("NewKeyProvider(sm2): %v", err)
	}
	if kp.Metadata().Algorithm != SM2SM3 {
		t.Fatalf("algorithm = %v", kp.Metadata().Algorithm)
	}
}

func TestInferAlgorithm_SM2(t *testing.T) {
	priv, err := GenerateSM2KeyPair()
	if err != nil {
		t.Fatalf("GenerateSM2KeyPair: %v", err)
	}
	pub, ok := priv.Public().(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("SM2 public key type = %T", priv.Public())
	}
	if got := inferAlgorithm(pub); got != SM2SM3 {
		t.Fatalf("inferAlgorithm(SM2) = %v, want SM2SM3", got)
	}
}

func TestSM2JWT_SignVerifyRoundtrip(t *testing.T) {
	priv, err := GenerateSM2KeyPair()
	if err != nil {
		t.Fatalf("GenerateSM2KeyPair: %v", err)
	}

	claims := jwt.MapClaims{
		"sub": "user-123",
		"iss": "ggid-test",
	}
	token := jwt.NewWithClaims(SigningMethodSM2, claims)
	token.Header["kid"] = "sm2-key-1"

	tokenStr, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	// Parse and verify with the public key.
	parsed, err := jwt.Parse(tokenStr, func(tok *jwt.Token) (interface{}, error) {
		if tok.Method.Alg() != SM2SM3Alg {
			t.Fatalf("unexpected alg: %v", tok.Method.Alg())
		}
		return priv.Public().(*ecdsa.PublicKey), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("parsed token invalid")
	}
	if sub, _ := parsed.Claims.(jwt.MapClaims)["sub"].(string); sub != "user-123" {
		t.Fatalf("sub = %v", sub)
	}
}

func TestSM2JWT_WrongKeyFails(t *testing.T) {
	priv1, _ := GenerateSM2KeyPair()
	priv2, _ := GenerateSM2KeyPair()

	token := jwt.NewWithClaims(SigningMethodSM2, jwt.MapClaims{"sub": "x"})
	tokenStr, err := token.SignedString(priv1)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = jwt.Parse(tokenStr, func(tok *jwt.Token) (interface{}, error) {
		return priv2.Public().(*ecdsa.PublicKey), nil
	})
	if err == nil {
		t.Fatal("expected signature verification failure with wrong key")
	}
}

func TestSM2JWT_RegisteredMethod(t *testing.T) {
	m := jwt.GetSigningMethod(SM2SM3Alg)
	if m == nil {
		t.Fatal("SM2SM3 signing method not registered")
	}
	if m.Alg() != SM2SM3Alg {
		t.Fatalf("alg = %v", m.Alg())
	}
}
