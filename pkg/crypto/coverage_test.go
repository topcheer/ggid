package crypto

import (
	"strings"
	"testing"
)

// TestHashPassword_EmptyPassword tests hashing an empty password.
func TestHashPassword_EmptyPassword2(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword(''): %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash for empty password")
	}
	if !strings.HasPrefix(hash, "argon2id$") {
		t.Errorf("expected argon2id$ prefix, got: %s", hash[:20])
	}
}

// TestHashPassword_DifferentSalts tests that same password produces different hashes.
func TestHashPassword_DifferentSalts(t *testing.T) {
	h1, _ := HashPassword("test123")
	h2, _ := HashPassword("test123")
	if h1 == h2 {
		t.Error("expected different hashes due to random salt")
	}
}

// TestAESEncryptDecrypt_Roundtrip tests encrypt then decrypt returns original.
func TestAESEncryptDecrypt_Roundtrip(t *testing.T) {
	plaintext := []byte("sensitive data")
	key := []byte("my-secret-key")
	ct, err := AESEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("AESEncrypt: %v", err)
	}
	pt, err := AESDecrypt(ct, key)
	if err != nil {
		t.Fatalf("AESDecrypt: %v", err)
	}
	if string(pt) != "sensitive data" {
		t.Errorf("expected 'sensitive data', got '%s'", pt)
	}
}

// TestAESEncrypt_DifferentCiphertexts tests same plaintext encrypts differently.
func TestAESEncrypt_DifferentCiphertexts(t *testing.T) {
	pt := []byte("test")
	key := []byte("key")
	c1, _ := AESEncrypt(pt, key)
	c2, _ := AESEncrypt(pt, key)
	if string(c1) == string(c2) {
		t.Error("expected different ciphertexts due to random nonce")
	}
}

// TestAESDecrypt_WrongKey_DifferentKey tests decryption with a completely different key.
func TestAESDecrypt_WrongKey_DifferentKey(t *testing.T) {
	pt := []byte("secret")
	ct, _ := AESEncrypt(pt, []byte("correct-key"))
	_, err := AESDecrypt(ct, []byte("wrong-key"))
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}

// TestAESDecrypt_ShortCiphertext tests decryption of too-short ciphertext.
func TestAESDecrypt_ShortCiphertext(t *testing.T) {
	_, err := AESDecrypt([]byte{1, 2, 3}, []byte("key"))
	if err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

// TestAESDecrypt_EmptyCiphertext tests decryption of empty ciphertext.
func TestAESDecrypt_EmptyCiphertext(t *testing.T) {
	_, err := AESDecrypt([]byte{}, []byte("key"))
	if err == nil {
		t.Error("expected error for empty ciphertext")
	}
}

// TestGenerateRandomToken tests basic token generation.
func TestGenerateRandomToken(t *testing.T) {
	tok, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("GenerateRandomToken: %v", err)
	}
	if tok == "" {
		t.Error("expected non-empty token")
	}
}

// TestGenerateRandomToken_Uniqueness tests tokens are unique.
func TestGenerateRandomToken_Uniqueness2(t *testing.T) {
	t1, _ := GenerateRandomToken(32)
	t2, _ := GenerateRandomToken(32)
	if t1 == t2 {
		t.Error("expected different tokens")
	}
}

// TestGenerateRandomToken_ZeroLength tests zero-length token.
func TestGenerateRandomToken_ZeroLength2(t *testing.T) {
	tok, err := GenerateRandomToken(0)
	if err != nil {
		t.Fatalf("GenerateRandomToken(0): %v", err)
	}
	if tok != "" {
		t.Errorf("expected empty token for 0 length, got %s", tok)
	}
}

// TestVerifyPassword_InvalidFormat tests verification with malformed hash.
func TestVerifyPassword_InvalidFormat(t *testing.T) {
	_, err := VerifyPassword("test", "not-a-valid-hash")
	if err == nil {
		t.Error("expected error for invalid hash format")
	}
}

// TestVerifyPassword_CorruptedHash tests verification with invalid base64.
func TestVerifyPassword_CorruptedHash(t *testing.T) {
	// Use invalid base64 characters that will fail decoding
	_, err := VerifyPassword("test", "argon2id$3$65536$2$!!!!.@@@@")
	if err == nil {
		t.Error("expected error for corrupted hash")
	}
}

// TestVerifyPassword_NoDot tests hash without dot separator.
func TestVerifyPassword_NoDot(t *testing.T) {
	_, err := VerifyPassword("test", "argon2id$3$65536$2$nodot")
	if err == nil {
		t.Error("expected error for hash without dot separator")
	}
}

// TestConstantTimeCompare tests the constant time comparison helper.
func TestConstantTimeCompare(t *testing.T) {
	if !constantTimeCompare([]byte("abc"), []byte("abc")) {
		t.Error("expected true for equal bytes")
	}
	if constantTimeCompare([]byte("abc"), []byte("abd")) {
		t.Error("expected false for different bytes")
	}
	if constantTimeCompare([]byte("abc"), []byte("ab")) {
		t.Error("expected false for different lengths")
	}
}

// TestSplitLast tests the split helper.
func TestSplitLast(t *testing.T) {
	parts := splitLast("salt.hash", ".")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[0] != "salt" || parts[1] != "hash" {
		t.Errorf("expected ['salt','hash'], got %v", parts)
	}

	// No separator found
	parts = splitLast("noSeparator", ".")
	if len(parts) != 1 {
		t.Errorf("expected 1 part, got %d", len(parts))
	}
}

// TestHashKey tests key hashing produces consistent output.
func TestHashKey(t *testing.T) {
	k1 := hashKey([]byte("test-key"))
	k2 := hashKey([]byte("test-key"))
	if len(k1) != 32 {
		t.Errorf("expected 32-byte key, got %d", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Error("expected same key hash for same input")
			break
		}
	}
}
