package crypto

import (
	"strings"
	"testing"
)

func TestCrypto_HashPassword_VerifyPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("MySecurePass123!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "argon2id$") {
		t.Errorf("expected argon2id prefix, got %s", hash[:10])
	}

	ok, err := VerifyPassword("MySecurePass123!", hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Error("expected correct password to verify")
	}

	// Wrong password should fail
	okWrong, _ := VerifyPassword("WrongPass", hash)
	if okWrong {
		t.Error("wrong password should not verify")
	}

	// Different hashes for same password (random salt)
	hash2, _ := HashPassword("MySecurePass123!")
	if hash == hash2 {
		t.Error("hashes should differ due to random salt")
	}
}

func TestCrypto_VerifyPassword_InvalidFormats(t *testing.T) {
	tests := []struct {
		name  string
		hash  string
		errSub string
	}{
		{"invalid format", "not-a-hash", "invalid hash format"},
		{"empty hash", "", "invalid hash format"},
		{"missing separator", "argon2id$3$65536$2$nopdot", "invalid hash encoding"},
		{"invalid base64 salt", "argon2id$3$65536$2$!!!invalid!!!.hash", "failed to decode salt"},
		{"invalid base64 hash", "argon2id$3$65536$2$c29tZXNhbHQ.!!!invalid!!!", "failed to decode hash"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyPassword("pass", tt.hash)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errSub) {
				t.Errorf("expected '%s', got: %v", tt.errSub, err)
			}
		})
	}
}

func TestCrypto_AES_RoundTrip(t *testing.T) {
	key := []byte("any-key")
	plaintext := []byte("sensitive data")

	ciphertext, err := AESEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("AESEncrypt: %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext should differ from plaintext")
	}

	decrypted, err := AESDecrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("AESDecrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("got '%s', want '%s'", decrypted, plaintext)
	}
}

func TestCrypto_AES_DifferentEachTime(t *testing.T) {
	key := []byte("key")
	c1, _ := AESEncrypt([]byte("data"), key)
	c2, _ := AESEncrypt([]byte("data"), key)
	if string(c1) == string(c2) {
		t.Error("same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestCrypto_AES_EmptyPlaintext(t *testing.T) {
	key := []byte("key")
	ct, err := AESEncrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("AESEncrypt empty: %v", err)
	}
	// Even empty plaintext produces nonce+tag
	if len(ct) < 12 {
		t.Errorf("expected at least nonce bytes, got %d", len(ct))
	}
	decrypted, err := AESDecrypt(ct, key)
	if err != nil {
		t.Fatalf("AESDecrypt empty: %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty plaintext, got %s", decrypted)
	}
}

func TestCrypto_AES_WrongKey(t *testing.T) {
	ct, _ := AESEncrypt([]byte("secret"), []byte("correct-key"))
	_, err := AESDecrypt(ct, []byte("wrong-key!!"))
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestCrypto_AES_CorruptedCiphertext(t *testing.T) {
	key := []byte("key")
	ct, _ := AESEncrypt([]byte("data"), key)
	ct[0] ^= 0xFF
	_, err := AESDecrypt(ct, key)
	if err == nil {
		t.Fatal("expected error for corrupted ciphertext")
	}
}

func TestCrypto_AES_TooShort(t *testing.T) {
	_, err := AESDecrypt([]byte{1, 2, 3}, []byte("key"))
	if err == nil {
		t.Fatal("expected error for too-short ciphertext")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected 'too short', got: %v", err)
	}
}

func TestCrypto_AES_EmptyCiphertext(t *testing.T) {
	_, err := AESDecrypt([]byte{}, []byte("key"))
	if err == nil {
		t.Fatal("expected error for empty ciphertext")
	}
}

func TestCrypto_GenerateRandomToken(t *testing.T) {
	token, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatalf("GenerateRandomToken: %v", err)
	}
	if len(token) < 32 {
		t.Errorf("token too short: %d", len(token))
	}

	// Uniqueness
	t2, _ := GenerateRandomToken(32)
	if token == t2 {
		t.Error("tokens should be unique")
	}

	// Zero length
	zeroToken, _ := GenerateRandomToken(0)
	if zeroToken != "" {
		t.Errorf("expected empty token for 0 bytes, got %s", zeroToken)
	}

	// Short token
	short, _ := GenerateRandomToken(1)
	if len(short) == 0 {
		t.Error("expected non-empty short token")
	}
}

func TestCrypto_ConstantTimeCompare(t *testing.T) {
	if !constantTimeCompare([]byte{1, 2, 3}, []byte{1, 2, 3}) {
		t.Error("equal slices should match")
	}
	if constantTimeCompare([]byte{1, 2, 3}, []byte{1, 2, 4}) {
		t.Error("different slices should not match")
	}
	if constantTimeCompare([]byte{1, 2, 3}, []byte{1, 2}) {
		t.Error("different lengths should not match")
	}
	if !constantTimeCompare(nil, nil) {
		t.Error("nil slices should match")
	}
}

func TestCrypto_SplitLast(t *testing.T) {
	// Normal case: last separator splits
	parts := splitLast("a.b.c", ".")
	if len(parts) != 2 || parts[0] != "a.b" || parts[1] != "c" {
		t.Errorf("expected [a.b c], got %v", parts)
	}

	// Single separator
	parts = splitLast("foo.bar", ".")
	if len(parts) != 2 || parts[0] != "foo" || parts[1] != "bar" {
		t.Errorf("expected [foo bar], got %v", parts)
	}

	// No separator
	parts = splitLast("nosep", ".")
	if len(parts) != 1 || parts[0] != "nosep" {
		t.Errorf("expected [nosep], got %v", parts)
	}

	// Empty string
	parts = splitLast("", ".")
	if len(parts) != 1 {
		t.Errorf("expected 1 part for empty, got %d", len(parts))
	}

	// Ends with separator
	parts = splitLast("foo.", ".")
	if len(parts) != 2 || parts[0] != "foo" || parts[1] != "" {
		t.Errorf("expected [foo ''], got %v", parts)
	}
}

func TestCrypto_HashKey(t *testing.T) {
	k1 := hashKey([]byte("test-key"))
	k2 := hashKey([]byte("test-key"))

	// Deterministic
	if string(k1) != string(k2) {
		t.Error("hashKey should be deterministic")
	}

	// Always 32 bytes (SHA-256)
	if len(k1) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(k1))
	}

	// Different inputs → different outputs
	k3 := hashKey([]byte("different"))
	if string(k1) == string(k3) {
		t.Error("different inputs should produce different hashes")
	}
}
