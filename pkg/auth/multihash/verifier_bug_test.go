package multihash

import (
	"encoding/base64"
	"fmt"
	"testing"

	"golang.org/x/crypto/argon2"
)

// BUG 1: verifyGGIDArgon2id uses StdEncoding instead of RawStdEncoding
// This is incompatible with crypto.HashPassword which uses RawStdEncoding
func TestVerifyGGIDArgon2id_Base64EncodingBug(t *testing.T) {
	pw := "test-password"
	salt := []byte("argonsalt1234567")
	iter, mem, par := 3, 65536, 2

	// Hash using RawStdEncoding (like crypto.HashPassword does)
	hashed := argon2.IDKey([]byte(pw), salt, uint32(iter), uint32(mem), uint8(par), 32)

	// Encode with RawStdEncoding (correct way, matches crypto module)
	encodedCorrect := fmt.Sprintf("argon2id$%d$%d$%d$%s.%s",
		iter, mem, par,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hashed))

	// This should work because multihash should accept both encodings
	ok, format, err := VerifyPassword(pw, encodedCorrect)
	if err != nil {
		t.Errorf("BUG: Failed to verify hash encoded with RawStdEncoding: %v", err)
	}
	if !ok {
		t.Error("BUG: Hash with RawStdEncoding should verify")
	}
	if format != FormatArgon2id {
		t.Errorf("Expected argon2id format, got %s", format)
	}
}

// BUG 2: verifyPHCArgon2id swaps t and mem parameters
// The code uses argon2.IDKey(password, salt, t, mem, p, keyLen)
// But it should be (password, salt, iter, mem, p, keyLen) where t is iter
func TestVerifyPHCArgon2id_ParameterOrderBug(t *testing.T) {
	pw := "test-password"
	salt := []byte("phcsalt1234567")

	// Create hash with different t and mem values
	iter := 3      // t = iterations
	mem := 65536   // m = memory
	p := 2
	hashed := argon2.IDKey([]byte(pw), salt, uint32(iter), uint32(mem), uint8(p), 32)

	// Encode in PHC format
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hashed)
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", mem, iter, p, saltB64, hashB64)

	// This should work
	ok, format, err := VerifyPassword(pw, encoded)
	if err != nil {
		t.Errorf("Failed to verify PHC format: %v", err)
	}
	if !ok {
		t.Error("PHC format should verify")
	}
	if format != FormatArgon2id {
		t.Errorf("Expected argon2id format, got %s", format)
	}

	// Now test with swapped parameters to demonstrate the bug
	// If the code incorrectly swaps t and mem in argon2.IDKey call, this would fail
	// Create hash with swapped values: use mem as iterations and iter as memory
	hashedSwapped := argon2.IDKey([]byte(pw), salt, uint32(mem), uint32(iter), uint8(p), 32)
	hashB64Swapped := base64.RawStdEncoding.EncodeToString(hashedSwapped)
	encodedSwapped := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", mem, iter, p, saltB64, hashB64Swapped)

	// This should NOT verify with the original password
	// (unless the bug exists and swaps the parameters back)
	okSwapped, _, _ := VerifyPassword(pw, encodedSwapped)
	if okSwapped {
		t.Log("WARNING: This might indicate the parameter swap bug exists")
		t.Log("The hash with swapped parameters verified, which suggests the code incorrectly swaps t and mem")
	}
}

// BUG 3: verifyPHCArgon2id tries RawStdEncoding then StdEncoding for salt
// but doesn't fall back correctly if the first fails
func TestVerifyPHCArgon2id_Base64Fallback(t *testing.T) {
	pw := "test-password"
	salt := []byte("phcsalt")
	hashed := argon2.IDKey([]byte(pw), salt, 3, 65536, 2, 32)

	// Test with RawStdEncoding (correct for PHC)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hashed)
	encoded := fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", saltB64, hashB64)

	ok, _, err := VerifyPassword(pw, encoded)
	if err != nil {
		t.Errorf("Failed to verify PHC with RawStdEncoding: %v", err)
	}
	if !ok {
		t.Error("PHC with RawStdEncoding should verify")
	}

	// Test with StdEncoding (has padding, might be used by some systems)
	saltB64Std := base64.StdEncoding.EncodeToString(salt)
	hashB64Std := base64.StdEncoding.EncodeToString(hashed)
	encodedStd := fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", saltB64Std, hashB64Std)

	okStd, _, errStd := VerifyPassword(pw, encodedStd)
	if errStd != nil {
		t.Logf("StdEncoding variant failed (might be expected): %v", errStd)
	} else if !okStd {
		t.Log("StdEncoding variant didn't verify (expected if code only handles RawStdEncoding)")
	} else {
		t.Log("StdEncoding variant worked - code handles both encodings")
	}
}

// BUG 4: Malformed inputs should be handled gracefully
func TestVerifyArgon2id_MalformedInputs(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		password string
		wantErr  bool
	}{
		{
			name:     "missing hash part",
			hash:     "argon2id$3$65536$2$salt",
			password: "test",
			wantErr:  true,
		},
		{
			name:     "wrong base64 in salt",
			hash:     "argon2id$3$65536$2$!!!invalid!!!.hash",
			password: "test",
			wantErr:  true,
		},
		{
			name:     "wrong base64 in hash",
			hash:     "argon2id$3$65536$2$c29tZXNhbHQ.!!!invalid!!!",
			password: "test",
			wantErr:  true,
		},
		{
			name:     "PHC missing hash",
			hash:     "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA",
			password: "test",
			wantErr:  true,
		},
		{
			name:     "PHC missing params",
			hash:     "$argon2id$v=19$c2FsdA$aGFzaA",
			password: "test",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, format, err := VerifyPassword(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPassword(%q, %q) error = %v, wantErr %v", tt.password, tt.hash, err, tt.wantErr)
			}
			if tt.wantErr && ok {
				t.Error("should not verify with malformed input")
			}
			if !tt.wantErr && !ok {
				t.Error("should verify with valid input")
			}
			if format == "" && !tt.wantErr {
				t.Error("format should not be empty for valid input")
			}
		})
	}
}

// BUG 5: verifyPBKDF2 and verifyScrypt don't validate iteration counts
func TestVerifyPBKDF2_InvalidIterations(t *testing.T) {
	tests := []struct {
		name        string
		iterations  int
		shouldFail  bool
	}{
		{"zero iterations", 0, true},
		{"negative iterations", -1, true},
		{"very large iterations", 999999999, false}, // Should accept but be slow
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := fmt.Sprintf("$pbkdf2$%d$salt$hash", tt.iterations)
			_, _, err := VerifyPassword("test", encoded)
			if (err != nil) != tt.shouldFail {
				t.Errorf("VerifyPassword with iterations=%d error = %v, shouldFail %v", tt.iterations, err, tt.shouldFail)
			}
		})
	}
}

// BUG 6: verifySSHA doesn't handle truncated hashes gracefully
func TestVerifySSHA_TruncatedHash(t *testing.T) {
	// SSHA requires at least sha1.Size (20 bytes) for the hash
	// Test with less data
	shortData := []byte("tooshort") // Only 8 bytes
	encoded := "{SSHA}" + base64.StdEncoding.EncodeToString(shortData)

	_, _, err := VerifyPassword("test", encoded)
	if err == nil {
		t.Error("Should fail with truncated SSHA hash")
	}
}

// BUG 7: pepper inconsistency
// crypto module applies pepper to passwords before hashing
// multihash module does NOT apply pepper
// This means passwords hashed with crypto won't verify with multihash if pepper is set
func TestPepperInconsistency(t *testing.T) {
	// This test documents the inconsistency
	// If pepper is set in crypto module via SetPepper(), then:
	// - crypto.HashPassword hashes: HMAC-SHA256(password, pepper) -> Argon2id
	// - crypto.VerifyPassword verifies: HMAC-SHA256(password, pepper) -> Argon2id -> compare
	// - multihash.VerifyPassword verifies: password -> Argon2id -> compare (NO PEPPER!)
	//
	// This is a SECURITY ISSUE: the same hash string will verify differently
	// depending on which module is used.

	t.Log("DOCUMENTED BUG: Pepper inconsistency between crypto and multihash modules")
	t.Log("crypto module applies pepper via applyPepper() before Argon2id")
	t.Log("multihash module does NOT apply pepper")
	t.Log("This causes verification inconsistency when pepper is configured")

	// To fix this, multihash should either:
	// 1. Apply the same pepper, or
	// 2. Reject pepper-protected hashes with a clear error, or
	// 3. Document that it only works with unpeppered hashes
}
