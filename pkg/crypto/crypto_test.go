package crypto

import (
	"strings"
	"testing"
)

func TestHashPassword_VerifyPassword_Success(t *testing.T) {
	password := "MySecureP@ssw0rd!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if !strings.HasPrefix(hash, "argon2id$") {
		t.Fatalf("hash should start with argon2id$, got: %s", hash)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if !ok {
		t.Fatal("password should match")
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}
	if ok {
		t.Fatal("wrong password should not match")
	}
}

func TestVerifyPassword_InvalidFormat(t *testing.T) {
	ok, err := VerifyPassword("password", "invalid-hash-format")
	if err == nil {
		t.Fatal("should error on invalid format")
	}
	if ok {
		t.Fatal("should not verify with invalid hash")
	}
}

func TestHashPassword_DifferentSalts(t *testing.T) {
	hash1, _ := HashPassword("same-password")
	hash2, _ := HashPassword("same-password")

	if hash1 == hash2 {
		t.Fatal("same password should produce different hashes (different salts)")
	}

	ok1, _ := VerifyPassword("same-password", hash1)
	ok2, _ := VerifyPassword("same-password", hash2)
	if !ok1 || !ok2 {
		t.Fatal("both hashes should verify the same password")
	}
}

func TestAESEncrypt_AESDecrypt_RoundTrip(t *testing.T) {
	plaintext := []byte("sensitive-data-such-as-totp-secret")
	key := []byte("my-encryption-key")

	ciphertext, err := AESEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("AESEncrypt failed: %v", err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := AESDecrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("AESDecrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}
}

func TestAESDecrypt_WrongKey(t *testing.T) {
	plaintext := []byte("secret")
	ciphertext, _ := AESEncrypt(plaintext, []byte("correct-key"))

	_, err := AESDecrypt(ciphertext, []byte("wrong-key"))
	if err == nil {
		t.Fatal("should fail with wrong key")
	}
}

func TestAESDecrypt_ShortCiphertext(t *testing.T) {
	_, err := AESDecrypt([]byte("short"), []byte("key"))
	if err == nil {
		t.Fatal("should fail on short ciphertext")
	}
}

func TestGenerateRandomToken(t *testing.T) {
	token1, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("GenerateRandomToken failed: %v", err)
	}
	if len(token1) == 0 {
		t.Fatal("token should not be empty")
	}

	token2, _ := GenerateRandomToken(32)
	if token1 == token2 {
		t.Fatal("tokens should be unique")
	}
}
