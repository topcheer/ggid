package crypto

import (
	"strings"
	"testing"
)

func TestSetPepper(t *testing.T) {
	// Save original to restore after test
	origPepper := pepper
	defer func() { pepper = origPepper }()

	SetPepper("test-pepper-secret")
	if string(pepper) != "test-pepper-secret" {
		t.Errorf("expected pepper to be set, got %q", string(pepper))
	}

	// Empty string should not change pepper
	SetPepper("")
	if string(pepper) != "test-pepper-secret" {
		t.Errorf("empty pepper should not overwrite, got %q", string(pepper))
	}
}

func TestApplyPepper_NoPepper(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	pepper = nil
	result := applyPepper("password123")
	if string(result) != "password123" {
		t.Errorf("without pepper, expected identity, got %q", string(result))
	}
}

func TestApplyPepper_WithPepper(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	pepper = []byte("secret-pepper")
	result := applyPepper("password123")
	// With pepper, result should be HMAC-SHA256 (32 bytes), not original password
	if len(result) != 32 {
		t.Errorf("expected 32-byte HMAC output, got %d bytes", len(result))
	}
	if string(result) == "password123" {
		t.Error("pepper should transform password, not return as-is")
	}

	// Same input should produce same output (deterministic HMAC)
	result2 := applyPepper("password123")
	if string(result) != string(result2) {
		t.Error("HMAC should be deterministic")
	}

	// Different input should produce different output
	result3 := applyPepper("different")
	if string(result) == string(result3) {
		t.Error("different input should produce different HMAC")
	}
}

func TestHashPassword_WithPepper(t *testing.T) {
	origPepper := pepper
	defer func() { pepper = origPepper }()

	SetPepper("integration-test-pepper")
	hashed, err := HashPassword("mypass")
	if err != nil {
		t.Fatalf("HashPassword with pepper failed: %v", err)
	}
	if !strings.HasPrefix(hashed, "argon2id$") {
		t.Errorf("expected argon2id prefix, got %q", hashed[:10])
	}

	// Verify with pepper set should succeed
	ok, err := VerifyPassword("mypass", hashed)
	if err != nil || !ok {
		t.Error("VerifyPassword should succeed with pepper set")
	}

	// Verify with wrong pepper should fail
	pepper = []byte("wrong-pepper")
	ok, _ = VerifyPassword("mypass", hashed)
	if ok {
		t.Error("VerifyPassword should fail with wrong pepper")
	}
}

func TestGenerateRandomToken_Empty(t *testing.T) {
	token, err := GenerateRandomToken(0)
	if err != nil {
		t.Errorf("expected no error for 0-length token, got %v", err)
	}
	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}
