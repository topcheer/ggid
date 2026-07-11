package crypto

import (
	"bytes"
	"strings"
	"testing"
)

// TestAESEncryptDecrypt_RoundTrip verifies the full encrypt→decrypt cycle.
func TestAESEncryptDecrypt_RoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 16) // any length — hashKey normalizes
	plaintext := []byte("sensitive data")

	ct, err := AESEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("AESEncrypt failed: %v", err)
	}
	if len(ct) == 0 {
		t.Fatal("expected non-empty ciphertext")
	}
	// Ciphertext should differ from plaintext
	if bytes.Equal(ct, plaintext) {
		t.Fatal("ciphertext equals plaintext — encryption failed")
	}

	pt, err := AESDecrypt(ct, key)
	if err != nil {
		t.Fatalf("AESDecrypt failed: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("decrypted %q, expected %q", pt, plaintext)
	}
}

// TestAESEncrypt_DifferentNonces verifies that encrypting the same data
// twice produces different ciphertexts (nonce randomness).
func TestAESEncrypt_DifferentNonces(t *testing.T) {
	key := []byte("test-key-1234567890")
	pt := []byte("same plaintext")

	ct1, _ := AESEncrypt(pt, key)
	ct2, _ := AESEncrypt(pt, key)
	if bytes.Equal(ct1, ct2) {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

// TestAESDecrypt_TooShort verifies error for short ciphertext.
func TestAESDecrypt_TooShort(t *testing.T) {
	key := []byte("test-key")
	_, err := AESDecrypt([]byte("short"), key)
	if err == nil {
		t.Fatal("expected error for too-short ciphertext")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Fatalf("expected 'too short' error, got %v", err)
	}
}

// TestAESDecrypt_WrongKey verifies decryption fails with wrong key.
func TestAESDecrypt_WrongKey(t *testing.T) {
	key1 := []byte("correct-key-1234")
	key2 := []byte("wrong-key-123456")
	pt := []byte("secret")

	ct, _ := AESEncrypt(pt, key1)
	_, err := AESDecrypt(ct, key2)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

// TestAESDecrypt_Tampered verifies decryption fails with tampered ciphertext.
func TestAESDecrypt_Tampered(t *testing.T) {
	key := []byte("test-key-12345678")
	ct, _ := AESEncrypt([]byte("secret"), key)

	// Flip a byte in the ciphertext
	ct[len(ct)-1] ^= 0xFF
	_, err := AESDecrypt(ct, key)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

// TestGenerateRandomToken_Uniqueness verifies tokens are unique.
func TestGenerateRandomToken_Uniq(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateRandomToken(32)
		if err != nil {
			t.Fatalf("GenerateRandomToken failed: %v", err)
		}
		if tokens[token] {
			t.Fatalf("duplicate token at iteration %d", i)
		}
		tokens[token] = true
	}
}

// TestGenerateRandomToken_Length verifies the base64url-encoded output length.
func TestGenerateRandomToken_Length(t *testing.T) {
	token, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 32 bytes → base64url = ceil(32*4/3) = 43 chars (no padding)
	if len(token) != 43 {
		t.Fatalf("expected 43 chars for 32-byte token, got %d", len(token))
	}
}

// TestGenerateRandomToken_ZeroLength verifies 0-byte token works.
func TestGenerateRandomToken_ZeroLength(t *testing.T) {
	token, err := GenerateRandomToken(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "" {
		t.Fatalf("expected empty string for 0-byte token, got %q", token)
	}
}

// TestHashPassword_DifferentPasswords verifies different inputs → different hashes.
func TestHashPassword_DifferentPasswords(t *testing.T) {
	h1, _ := HashPassword("password1")
	h2, _ := HashPassword("password2")
	if h1 == h2 {
		t.Fatal("expected different hashes for different passwords")
	}
}

// TestHashPassword_SamePassword verifies bcrypt salt uniqueness.
func TestHashPassword_SamePassword(t *testing.T) {
	h1, _ := HashPassword("samepass")
	h2, _ := HashPassword("samepass")
	if h1 == h2 {
		t.Fatal("expected different hashes due to salt")
	}
}

// TestEnableTestFastHash verifies the test-fast-hash flag.
func TestEnableTestFastHash(t *testing.T) {
	EnableTestFastHash()
	// Verify it doesn't panic and hashing still works
	h, err := HashPassword("test")
	if err != nil {
		t.Fatalf("HashPassword with fast hash failed: %v", err)
	}
	if h == "" {
		t.Fatal("expected non-empty hash")
	}
	// Verify the hash is still verifiable
	ok, err := VerifyPassword("test", h)
	if err != nil || !ok {
		t.Fatal("fast hash should still verify")
	}
}
