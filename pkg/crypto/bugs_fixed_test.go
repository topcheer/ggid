package crypto

import (
	"strings"
	"testing"

	"github.com/ggid/ggid/pkg/auth/multihash"
	"golang.org/x/crypto/bcrypt"
)

// TestBcryptErrorHandlingFix verifies that the bcrypt error handling bug is fixed.
//
// BUG HISTORY: crypto.VerifyPassword previously returned (false, err) for bcrypt
// mismatch, treating wrong password as an error. This was inconsistent with
// multihash.verifyBcrypt which correctly returns (false, nil).
//
// FIX: Changed line 125 in crypto.go from:
//   return err == nil, err
// to:
//   return err == nil, nil
//
// This ensures wrong passwords return (false, nil) consistently across both modules.
func TestBcryptErrorHandlingFix(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	// Test with wrong password - should return (false, nil)
	ok, err := VerifyPassword("wrong-password", string(hash))
	if ok {
		t.Error("wrong password should not match")
	}
	if err != nil {
		t.Errorf("BUG NOT FIXED: VerifyPassword returned error for bcrypt mismatch: %v", err)
		t.Error("Expected (false, nil) for wrong password")
	}

	// Test with correct password - should return (true, nil)
	ok, err = VerifyPassword("correct-password", string(hash))
	if !ok {
		t.Error("correct password should match")
	}
	if err != nil {
		t.Errorf("VerifyPassword returned error for correct password: %v", err)
	}
}

// TestBcryptConsistency verifies consistency between crypto and multihash modules.
//
// BUG HISTORY: The two modules had inconsistent error handling for bcrypt verification.
// crypto returned the bcrypt error, while multihash returned nil for mismatches.
//
// FIX: Now both modules return (false, nil) for wrong passwords.
func TestBcryptConsistency(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	hashStr := string(hash)

	// crypto.VerifyPassword should return (false, nil) for wrong password
	cryptoOk, cryptoErr := VerifyPassword("wrong", hashStr)

	// multihash.VerifyPassword should also return (false, nil) for wrong password
	multiOk, _, multiErr := multihash.VerifyPassword("wrong", hashStr)

	t.Logf("crypto.VerifyPassword: ok=%v, err=%v", cryptoOk, cryptoErr)
	t.Logf("multihash.VerifyPassword: ok=%v, err=%v", multiOk, multiErr)

	if cryptoErr != nil {
		t.Error("BUG NOT FIXED: crypto returns error on bcrypt mismatch")
	}
	if multiErr != nil {
		t.Error("multihash returns error on bcrypt mismatch")
	}
	if cryptoErr != nil && multiErr == nil {
		t.Error("INCONSISTENCY: crypto returns error, multihash does not")
	}
}

// TestArgon2idBase64Compatibility verifies that crypto and multihash use compatible encodings.
//
// BUG HISTORY: multihash.verifyGGIDArgon2id used base64.StdEncoding while
// crypto.HashPassword used base64.RawStdEncoding. This caused hashes created
// by crypto to fail verification by multihash.
//
// FIX: Changed multihash.verifyGGIDArgon2id to try RawStdEncoding first,
// then fall back to StdEncoding, accepting both formats.
func TestArgon2idBase64Compatibility(t *testing.T) {
	// Create a hash using crypto.HashPassword (uses RawStdEncoding)
	hash, err := HashPassword("test-password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !strings.HasPrefix(hash, "argon2id$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}

	// Verify with crypto.VerifyPassword (uses RawStdEncoding)
	ok, err := VerifyPassword("test-password", hash)
	if err != nil {
		t.Errorf("crypto.VerifyPassword failed: %v", err)
	}
	if !ok {
		t.Error("crypto.VerifyPassword failed to verify correct password")
	}

	// Verify with multihash.VerifyPassword (should now support RawStdEncoding)
	multiOk, format, multiErr := multihash.VerifyPassword("test-password", hash)
	if multiErr != nil {
		t.Errorf("multihash.VerifyPassword failed: %v", multiErr)
	}
	if !multiOk {
		t.Error("multihash.VerifyPassword failed to verify correct password")
	}
	if format != multihash.FormatArgon2id {
		t.Errorf("expected argon2id format, got %s", format)
	}

	// Wrong password should fail with both modules
	wrongOk, _ := VerifyPassword("wrong-password", hash)
	if wrongOk {
		t.Error("crypto.VerifyPassword should fail for wrong password")
	}

	wrongMultiOk, _, _ := multihash.VerifyPassword("wrong-password", hash)
	if wrongMultiOk {
		t.Error("multihash.VerifyPassword should fail for wrong password")
	}
}

// TestSplitLastCorrectness verifies the splitLast function works correctly.
//
// ANALYSIS: Initial analysis suggested splitLast had a bug because it converts
// a byte to string before comparison. However, testing revealed this works
// correctly for single-byte separators like ".".
//
// The function is CORRECT as written.
func TestSplitLastCorrectness(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		sep    string
		expect []string
	}{
		{
			name:   "multiple dots",
			input:  "a.b.c",
			sep:    ".",
			expect: []string{"a.b", "c"},
		},
		{
			name:   "single dot (hash format)",
			input:  "salt.hash",
			sep:    ".",
			expect: []string{"salt", "hash"},
		},
		{
			name:   "no separator",
			input:  "nosep",
			sep:    ".",
			expect: []string{"nosep"},
		},
		{
			name:   "ends with separator",
			input:  "foo.",
			sep:    ".",
			expect: []string{"foo", ""},
		},
		{
			name:   "empty string",
			input:  "",
			sep:    ".",
			expect: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitLast(tt.input, tt.sep)
			if len(parts) != len(tt.expect) {
				t.Errorf("splitLast(%q, %q) = %v (len=%d), want %v (len=%d)",
					tt.input, tt.sep, parts, len(parts), tt.expect, len(tt.expect))
				return
			}
			for i := range parts {
				if parts[i] != tt.expect[i] {
					t.Errorf("parts[%d] = %q, want %q", i, parts[i], tt.expect[i])
				}
			}
		})
	}
}

// TestEmptyInputHandling verifies graceful handling of empty inputs.
//
// ANALYSIS: Both modules handle empty passwords and hashes correctly.
// No bugs found in this area.
func TestEmptyInputHandling(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{
			name:     "empty password with valid argon2id hash",
			password: "",
			hash:     "argon2id$3$65536$2$c29tZXNhbHQ.c29tZWhhc2g",
			wantErr:  false,
		},
		{
			name:     "empty hash",
			password: "password",
			hash:     "",
			wantErr:  true,
		},
		{
			name:     "both empty",
			password: "",
			hash:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := VerifyPassword(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword(%q, %q) error = %v, wantErr %v",
					tt.password, tt.hash, err, tt.wantErr)
			}
		})
	}
}
