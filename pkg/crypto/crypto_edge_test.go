package crypto

import (
	"strings"
	"testing"
)

// --- HashPassword edge cases ---

func TestHashPassword_DifferentSalats(t *testing.T) {
	h1, err := HashPassword("samepassword")
	if err != nil {
		t.Fatal(err)
	}
	h2, err := HashPassword("samepassword")
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Error("different salts should produce different hashes")
	}
}

func TestHashPassword_FormatPrefix(t *testing.T) {
	h, err := HashPassword("test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "argon2id$") {
		t.Errorf("hash should start with 'argon2id$', got: %s", h[:20])
	}
}

func TestHashPassword_LongPassword(t *testing.T) {
	longPassword := strings.Repeat("a", 10000)
	h, err := HashPassword(longPassword)
	if err != nil {
		t.Fatalf("long password failed: %v", err)
	}
	ok, err := VerifyPassword(longPassword, h)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("long password should verify")
	}
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	h, err := HashPassword("")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("", h)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("empty password should verify")
	}
}

func TestHashPassword_UnicodePassword(t *testing.T) {
	password := "パスワード123"
	h, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword(password, h)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("unicode password should verify")
	}
}

// --- VerifyPassword edge cases ---

func TestVerifyPassword_NoSeparator(t *testing.T) {
	// Missing the '.' separator between salt and hash
	ok, err := VerifyPassword("test", "argon2id$3$65536$2$invalidsaltandhash")
	if err == nil {
		t.Error("expected error for missing separator")
	}
	if ok {
		t.Error("should not verify")
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	h, err := HashPassword("correct")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("wrong", h)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("wrong password should not verify")
	}
}

func TestVerifyPassword_InvalidSaltBase64(t *testing.T) {
	// Valid format but invalid base64 in salt
	ok, err := VerifyPassword("test", "argon2id$3$65536$2$!!!.validbase64")
	if err == nil {
		t.Error("expected error for invalid base64 salt")
	}
	if ok {
		t.Error("should not verify")
	}
}

func TestVerifyPassword_InvalidHashBase64(t *testing.T) {
	// Valid salt but invalid hash
	h, err := HashPassword("test")
	if err != nil {
		t.Fatal(err)
	}
	// Tamper: replace hash part with invalid base64
	parts := strings.SplitN(h, ".", 2)
	if len(parts) != 2 {
		t.Fatal("unexpected hash format")
	}
	tampered := parts[0] + ".!!!invalid!!!"
	ok, err := VerifyPassword("test", tampered)
	if err == nil {
		t.Error("expected error for invalid base64 hash")
	}
	if ok {
		t.Error("should not verify")
	}
}

func TestVerifyPassword_CompletelyInvalidFormat(t *testing.T) {
	ok, err := VerifyPassword("test", "just-a-string")
	if err == nil {
		t.Error("expected error for invalid format")
	}
	if ok {
		t.Error("should not verify")
	}
}

func TestVerifyPassword_EmptyString(t *testing.T) {
	ok, err := VerifyPassword("test", "")
	if err == nil {
		t.Error("expected error for empty hash")
	}
	if ok {
		t.Error("should not verify")
	}
}

func TestVerifyPassword_DifferentKeyLength(t *testing.T) {
	h, err := HashPassword("test")
	if err != nil {
		t.Fatal(err)
	}
	// Should still verify correctly
	ok, err := VerifyPassword("test", h)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("should verify")
	}
}

// --- AES edge cases ---

func TestAES_LargePlaintext(t *testing.T) {
	plaintext := make([]byte, 100000) // 100KB
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	key := []byte("secret-key")

	ct, err := AESEncrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := AESDecrypt(ct, key)
	if err != nil {
		t.Fatal(err)
	}
	if len(pt) != len(plaintext) {
		t.Fatalf("length mismatch: %d vs %d", len(pt), len(plaintext))
	}
}

func TestAES_SameKey(t *testing.T) {
	key := []byte("same-key")
	ct1, _ := AESEncrypt([]byte("data"), key)
	ct2, _ := AESEncrypt([]byte("data"), key)
	// Nonces are random so ciphertexts should differ
	if string(ct1) == string(ct2) {
		t.Error("same data+key should produce different ciphertext (random nonce)")
	}
}

func TestAES_NilKey(t *testing.T) {
	// nil key should still work (hashKey produces SHA256 of empty)
	ct, err := AESEncrypt([]byte("data"), nil)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := AESDecrypt(ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != "data" {
		t.Errorf("decrypted = %q", pt)
	}
}

func TestAES_TamperedCiphertext(t *testing.T) {
	key := []byte("key")
	ct, err := AESEncrypt([]byte("secret"), key)
	if err != nil {
		t.Fatal(err)
	}
	// Flip a byte in ciphertext (after nonce)
	if len(ct) > 12 {
		ct[13] ^= 0xFF
	}
	_, err = AESDecrypt(ct, key)
	if err == nil {
		t.Error("tampered ciphertext should fail decryption")
	}
}

func TestAES_DifferentKeyLengths(t *testing.T) {
	tests := [][]byte{
		[]byte("a"),
		[]byte("ab"),
		[]byte("abcdefghijklmnop"), // 16 chars
		[]byte("abcdefghijklmnopqrstuvwxyz123456"), // 32 chars
	}
	for _, key := range tests {
		ct, err := AESEncrypt([]byte("data"), key)
		if err != nil {
			t.Errorf("key len %d: encrypt failed: %v", len(key), err)
			continue
		}
		pt, err := AESDecrypt(ct, key)
		if err != nil {
			t.Errorf("key len %d: decrypt failed: %v", len(key), err)
			continue
		}
		if string(pt) != "data" {
			t.Errorf("key len %d: decrypted = %q", len(key), pt)
		}
	}
}

// --- GenerateRandomToken edge cases ---

func TestGenerateRandomToken_Lengths(t *testing.T) {
	for _, n := range []int{1, 8, 16, 32, 64, 128} {
		token, err := GenerateRandomToken(n)
		if err != nil {
			t.Errorf("byteLen %d: %v", n, err)
		}
		// Base64 URL encoding: n bytes -> ceil(n*4/3) chars (with padding stripped)
		if len(token) == 0 {
			t.Errorf("byteLen %d: empty token", n)
		}
	}
}

func TestGenerateRandomToken_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateRandomToken(32)
		if err != nil {
			t.Fatal(err)
	}
		if seen[token] {
			t.Error("duplicate token generated")
		}
		seen[token] = true
	}
}

func TestGenerateRandomToken_ZeroBytes(t *testing.T) {
	token, err := GenerateRandomToken(0)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		t.Errorf("0 bytes should produce empty string, got %q", token)
	}
}

// --- splitLast edge cases ---

func TestSplitLast_NoMatch(t *testing.T) {
	result := splitLast("hello", ".")
	if len(result) != 1 || result[0] != "hello" {
		t.Errorf("splitLast without separator should return single element: %v", result)
	}
}

func TestSplitLast_MultipleMatches(t *testing.T) {
	result := splitLast("a.b.c", ".")
	if len(result) != 2 || result[0] != "a.b" || result[1] != "c" {
		t.Errorf("splitLast with multiple separators: %v", result)
	}
}

func TestSplitLast_SingleMatch(t *testing.T) {
	result := splitLast("hello.world", ".")
	if len(result) != 2 || result[0] != "hello" || result[1] != "world" {
		t.Errorf("splitLast with single separator: %v", result)
	}
}

func TestSplitLast_EmptyString(t *testing.T) {
	result := splitLast("", ".")
	if len(result) != 1 || result[0] != "" {
		t.Errorf("splitLast on empty string: %v", result)
	}
}

// --- constantTimeCompare edge cases ---

func TestConstantTimeCompare_DifferentLengths(t *testing.T) {
	if constantTimeCompare([]byte{1, 2, 3}, []byte{1, 2}) {
		t.Error("different lengths should return false")
	}
}

func TestConstantTimeCompare_BothEmpty(t *testing.T) {
	if !constantTimeCompare([]byte{}, []byte{}) {
		t.Error("both empty should return true")
	}
}

func TestConstantTimeCompare_SameContent(t *testing.T) {
	if !constantTimeCompare([]byte{1, 2, 3, 4, 5}, []byte{1, 2, 3, 4, 5}) {
		t.Error("same content should return true")
	}
}

// --- hashKey edge cases ---

func TestHashKey_EmptyKey(t *testing.T) {
	h := hashKey(nil)
	if len(h) != 32 {
		t.Errorf("hashKey(nil) length = %d, want 32", len(h))
	}
	// SHA256 of empty string
	h2 := hashKey([]byte{})
	if string(h) != string(h2) {
		t.Error("hashKey(nil) and hashKey([]byte{}) should be the same")
	}
}

func TestHashKey_ConsistentOutput(t *testing.T) {
	h1 := hashKey([]byte("key"))
	h2 := hashKey([]byte("key"))
	if string(h1) != string(h2) {
		t.Error("same input should produce same hash")
	}
}

func TestHashKey_DifferentInputs(t *testing.T) {
	h1 := hashKey([]byte("key1"))
	h2 := hashKey([]byte("key2"))
	if string(h1) == string(h2) {
		t.Error("different inputs should produce different hashes")
	}
}
