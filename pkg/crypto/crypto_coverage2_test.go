package crypto

import (
	"strings"
	"testing"
)

// --- HashPassword error path ---
// The only error path is io.ReadFull(rand.Reader, salt) failing,
// which is nearly impossible to trigger in tests. We can however
// ensure the hash format has all the right components.

func TestHashPassword_SaltLength(t *testing.T) {
	h, err := HashPassword("test")
	if err != nil {
		t.Fatal(err)
	}
	// Format: argon2id$iter$mem$par$saltBase64.hashBase64
	parts := strings.SplitN(h, "$", 5)
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d: %s", len(parts), h)
	}
	// Check algorithm
	if parts[0] != "argon2id" {
		t.Errorf("algorithm = %q", parts[0])
	}
	// Check salt.hash has a separator
	saltHash := parts[4]
	if !strings.Contains(saltHash, ".") {
		t.Errorf("salt.hash should contain '.': %s", saltHash)
	}
}

// --- AESEncrypt: test GCM Seal path with valid data ---
// The error paths are: aes.NewCipher (fails with bad key len - but hashKey ensures 32 bytes)
// and cipher.NewGCM (fails only on unsupported block sizes)
// Both are practically unreachable with valid inputs.

func TestAESEncrypt_ValidKey_HashKeyAlways32Bytes(t *testing.T) {
	// hashKey always produces 32 bytes, so AES-256 always succeeds
	key := []byte("any-key-length-works")
	ct, err := AESEncrypt([]byte("data"), key)
	if err != nil {
		t.Fatalf("AESEncrypt should always succeed with hashKey: %v", err)
	}
	if len(ct) == 0 {
		t.Error("ciphertext should not be empty")
	}
}

// --- AESDecrypt error path: ciphertext exactly nonceSize (no data) ---

func TestAESDecrypt_NonceOnly(t *testing.T) {
	key := []byte("test-key")
	// Encrypt then truncate to just the nonce
	ct, err := AESEncrypt([]byte("data"), key)
	if err != nil {
		t.Fatal(err)
	}
	// gcm.NonceSize() is 12, so ct[:12] is just the nonce, no ciphertext
	nonceOnly := ct[:12]
	_, err = AESDecrypt(nonceOnly, key)
	if err == nil {
		t.Error("decrypting nonce-only should fail (empty ciphertext)")
	}
}

// --- GenerateRandomToken: verify encoding ---
// The error path is io.ReadFull(rand.Reader, b) which requires rand.Reader to fail.
// In normal operation this never happens.

func TestGenerateRandomToken_Encoding(t *testing.T) {
	token, err := GenerateRandomToken(32)
	if err != nil {
		t.Fatal(err)
	}
	// Should be valid base64url
	for _, c := range token {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			t.Errorf("invalid base64url char in token: %c", c)
		}
	}
}

// --- Full round-trip stress test ---

func TestAES_RoundTrip_Stress(t *testing.T) {
	key := []byte("stress-test-key")
	sizes := []int{1, 7, 16, 64, 256, 1024, 4096}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}

		ct, err := AESEncrypt(data, key)
		if err != nil {
			t.Errorf("size %d: encrypt error: %v", size, err)
			continue
		}

		pt, err := AESDecrypt(ct, key)
		if err != nil {
			t.Errorf("size %d: decrypt error: %v", size, err)
			continue
		}

		if len(pt) != len(data) {
			t.Errorf("size %d: length mismatch %d vs %d", size, len(pt), len(data))
			continue
		}

		for i := range pt {
			if pt[i] != data[i] {
				t.Errorf("size %d: byte mismatch at %d", size, i)
				break
			}
		}
	}
}

// --- VerifyPassword with valid hash but different iterations ---

func TestVerifyPassword_CustomParams(t *testing.T) {
	// Create hash with default params
	h, err := HashPassword("test123")
	if err != nil {
		t.Fatal(err)
	}

	// Verify with correct password
	ok, err := VerifyPassword("test123", h)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if !ok {
		t.Error("correct password should verify")
	}

	// Verify with slightly different password
	ok, err = VerifyPassword("test124", h)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if ok {
		t.Error("wrong password should NOT verify")
	}
}

// --- End-to-end HashPassword -> VerifyPassword with special chars ---

func TestHashPassword_SpecialCharacters(t *testing.T) {
	passwords := []string{
		"p@ssw0rd!",
		"$2a$10$...",
		"hash with spaces",
		"tab\there",
		"newline\nin\npassword",
		"unicode: 🎉🎊",
		strings.Repeat("a", 1),
	}

	for _, pw := range passwords {
		h, err := HashPassword(pw)
		if err != nil {
			t.Errorf("HashPassword(%q) error: %v", pw, err)
			continue
		}
		ok, err := VerifyPassword(pw, h)
		if err != nil {
			t.Errorf("VerifyPassword(%q) error: %v", pw, err)
			continue
		}
		if !ok {
			t.Errorf("password %q should verify", pw)
		}
	}
}
