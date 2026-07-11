package crypto

// Password Pepper Functional Verification Tests
// Verifies: pepper is correctly applied in hash/verify lifecycle
// Date: 2026-07-25

import (
	"strings"
	"testing"
)

// TestPepper_Functional_FullLifecycle verifies the complete pepper lifecycle:
// SetPepper → HashPassword → VerifyPassword (correct) → VerifyPassword (wrong).
func TestPepper_Functional_FullLifecycle(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	SetPepper("functional-test-pepper-secret")

	hash, err := HashPassword("MySecurePass123!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "argon2id$") {
		t.Errorf("hash should have argon2id prefix, got %s", hash[:10])
	}

	// Correct password with correct pepper → verify succeeds
	ok, err := VerifyPassword("MySecurePass123!", hash)
	if err != nil || !ok {
		t.Error("verify with correct password + pepper should succeed")
	}

	// Wrong password → verify fails
	ok, _ = VerifyPassword("WrongPassword", hash)
	if ok {
		t.Error("verify with wrong password should fail")
	}
}

// TestPepper_Functional_DifferentPeppersProduceDifferentHashes verifies that
// the same password hashed with different peppers produces different hashes.
func TestPepper_Functional_DifferentPeppersProduceDifferentHashes(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	SetPepper("pepper-A")
	hashA, _ := HashPassword("shared-password")

	SetPepper("pepper-B")
	hashB, _ := HashPassword("shared-password")

	if hashA == hashB {
		t.Fatal("same password with different peppers should produce different hashes")
	}

	// hashA should only verify with pepper-A
	SetPepper("pepper-A")
	ok, _ := VerifyPassword("shared-password", hashA)
	if !ok {
		t.Error("hashA should verify with pepper-A")
	}

	// hashA should NOT verify with pepper-B
	SetPepper("pepper-B")
	ok, _ = VerifyPassword("shared-password", hashA)
	if ok {
		t.Error("hashA should NOT verify with pepper-B")
	}
}

// TestPepper_Functional_NoPepperBackwardCompat verifies that without pepper,
// hash/verify work normally (backward compatible).
func TestPepper_Functional_NoPepperBackwardCompat(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	pepper = nil

	hash, err := HashPassword("nop-pepper-pass")
	if err != nil {
		t.Fatalf("HashPassword without pepper: %v", err)
	}

	ok, _ := VerifyPassword("nop-pepper-pass", hash)
	if !ok {
		t.Error("verify without pepper should succeed")
	}
}

// TestPepper_Functional_EmptySetPepperNoop verifies SetPepper("") doesn't change pepper.
func TestPepper_Functional_EmptySetPepperNoop(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	SetPepper("initial-pepper")
	SetPepper("") // should be no-op
	if string(pepper) != "initial-pepper" {
		t.Error("empty SetPepper should be a no-op")
	}
}

// TestPepper_Functional_HashFormatStable verifies pepper doesn't change hash format.
func TestPepper_Functional_HashFormatStable(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	// With pepper
	SetPepper("format-test-pepper")
	hashWithPepper, _ := HashPassword("format-pass")
	if !strings.HasPrefix(hashWithPepper, "argon2id$") {
		t.Error("hash with pepper should have argon2id$ prefix")
	}

	// Without pepper
	pepper = nil
	hashNoPepper, _ := HashPassword("format-pass")
	if !strings.HasPrefix(hashNoPepper, "argon2id$") {
		t.Error("hash without pepper should have argon2id$ prefix")
	}

	// Both should have same format structure (algorithm$params$salt.hash)
	partsWith := strings.Split(hashWithPepper, "$")
	partsWithout := strings.Split(hashNoPepper, "$")
	if len(partsWith) != len(partsWithout) {
		t.Errorf("hash format part count should match: %d vs %d",
			len(partsWith), len(partsWithout))
	}
}
