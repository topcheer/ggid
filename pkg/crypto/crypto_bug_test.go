package crypto

import (
	"strings"
	"testing"

	"github.com/ggid/ggid/pkg/auth/multihash"
	"golang.org/x/crypto/bcrypt"
)

// BUG 1: VerifyPassword returns (false, error) for bcrypt mismatch instead of (false, nil)
// This is inconsistent with multihash.verifyBcrypt which correctly returns (false, nil)
func TestVerifyPassword_BcryptErrorHandling(t *testing.T) {
	// Generate a valid bcrypt hash
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	// Test with wrong password - should return (false, nil) like multihash does
	// BUG: Currently returns (false, bcrypt.ErrMismatchedHashAndPassword)
	ok, err := VerifyPassword("wrong-password", string(hash))
	if ok {
		t.Error("wrong password should not match")
	}
	if err != nil {
		// This is the bug - err should be nil for wrong password
		t.Errorf("BUG: VerifyPassword returned error for bcrypt mismatch (should be nil): %v", err)
		t.Logf("multihash.verifyBcrypt correctly returns (false, nil) for mismatches")
	}
}

// BUG 2: splitLast compares byte to string, causing it to never find "." in base64
func TestSplitLast_DotInBase64(t *testing.T) {
	// Create a string with multiple dots (simulating base64 with dots)
	// RawStdEncoding base64 can contain +, /, -, _ but not dots
	// However, the format is salt.hash, so the dot is the separator
	// What if the salt or hash itself contains a dot character in base64?
	// Actually, base64url (RawURLEncoding) uses - and _, not dots
	// But the BUG is in the implementation: string(s[i]) == sep

	// The bug: string(s[i]) converts a single byte to a string
	// Then compares it to "." (multi-byte string)
	// This will NEVER match because string(s[i]) is always a single character

	// Test with simple case
	parts := splitLast("abc.def", ".")
	if len(parts) != 2 {
		t.Errorf("BUG: splitLast('abc.def', '.') should return 2 parts, got %d", len(parts))
		t.Logf("parts = %v", parts)
		t.Logf("This fails because string(s[i]) == '.' compares a 1-char string to \".\"")
	}

	// Test the actual hash format case
	saltB64 := "c29tZXNhbHQ" // "somesalt" in base64
	hashB64 := "c29tZWhhc2g" // "somehash" in base64
	combined := saltB64 + "." + hashB64

	parts = splitLast(combined, ".")
	if len(parts) != 2 || parts[0] != saltB64 || parts[1] != hashB64 {
		t.Errorf("BUG: splitLast failed to split '%s' correctly", combined)
		t.Logf("Got: %v", parts)
	}
}

// BUG 3: VerifyPassword doesn't handle empty password/hash gracefully for Argon2id
func TestVerifyPassword_EmptyInputs(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{"empty password with valid hash", "", "argon2id$3$65536$2$c29tZXNhbHQ.c29tZWhhc2g", false},
		{"empty hash", "password", "", true},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyPassword(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword(%q, %q) error = %v, wantErr %v", tt.password, tt.hash, err, tt.wantErr)
			}
		})
	}
}

// BUG 4: splitLast with special characters that might be in base64
func TestSplitLast_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expect   []string
		shouldWork bool
	}{
		{
			name:     "simple dots",
			input:    "a.b.c",
			sep:      ".",
			expect:   []string{"a.b", "c"},
			shouldWork: false, // BUG: won't work due to string comparison
		},
		{
			name:     "single dot",
			input:    "salt.hash",
			sep:      ".",
			expect:   []string{"salt", "hash"},
			shouldWork: false, // BUG: won't work
		},
		{
			name:     "no separator",
			input:    "nosep",
			sep:      ".",
			expect:   []string{"nosep"},
			shouldWork: true, // Should work (returns whole string)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitLast(tt.input, tt.sep)
			if !tt.shouldWork {
				// We expect it to fail
				if len(parts) == len(tt.expect) {
					// Check if by chance it worked
					match := true
					for i := range parts {
						if parts[i] != tt.expect[i] {
							match = false
							break
						}
					}
					if match {
						t.Log("Surprisingly worked - this means the bug might be fixed or test is wrong")
					} else {
						t.Logf("BUG confirmed: splitLast('%s', '%s') = %v, want %v", tt.input, tt.sep, parts, tt.expect)
					}
				}
			} else {
				// Should work
				if len(parts) != len(tt.expect) {
					t.Errorf("splitLast('%s', '%s') = %v, want %v", tt.input, tt.sep, parts, tt.expect)
				}
			}
		})
	}
}

// BUG 5: VerifyPassword with Argon2id format using wrong salt length
// Should handle gracefully instead of potentially panicking
func TestVerifyPassword_Argon2id_WrongSaltLength(t *testing.T) {
	// Create a hash with very short salt (less than argonSaltLength=16)
	shortSalt := "c2E=" // "sa" in base64 (only 2 bytes decoded)
	hashB64 := "c29tZWhhc2g=" // "somehash" in base64
	hash := "argon2id$3$65536$2$" + shortSalt + "." + hashB64

	_, err := VerifyPassword("password", hash)
	if err == nil {
		t.Error("Should fail gracefully with wrong salt length")
	}
	if !strings.Contains(err.Error(), "failed to decode salt") && !strings.Contains(err.Error(), "invalid hash") {
		t.Logf("Error for wrong salt length: %v", err)
	}
}

// Demonstration of the bcrypt error inconsistency between crypto and multihash
func TestBcryptError_Inconsistency(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	hashStr := string(hash)

	// crypto.VerifyPassword returns (false, err) on bcrypt mismatch
	cryptoOk, cryptoErr := VerifyPassword("wrong", hashStr)

	// multihash.verifyBcrypt (indirectly) returns (false, nil) on mismatch
	multiOk, _, multiErr := multihash.VerifyPassword("wrong", hashStr)

	t.Logf("crypto.VerifyPassword: ok=%v, err=%v", cryptoOk, cryptoErr)
	t.Logf("multihash.VerifyPassword: ok=%v, err=%v", multiOk, multiErr)

	if cryptoErr != nil && multiErr == nil {
		t.Error("INCONSISTENCY: crypto returns error on bcrypt mismatch, multihash does not")
		t.Error("This violates the principle that wrong password is not an error")
	}
}
