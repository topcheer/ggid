package server

import (
	"testing"
)

func TestDetectHashType_Argon2id(t *testing.T) {
	if dt := DetectHashType("argon2id$1$4096$1$salt.hash", ""); dt != "argon2id" {
		t.Errorf("expected argon2id, got %s", dt)
	}
}

func TestDetectHashType_Bcrypt(t *testing.T) {
	if dt := DetectHashType("$2a$10$abc123def456", ""); dt != "bcrypt" {
		t.Errorf("expected bcrypt, got %s", dt)
	}
}

func TestDetectHashType_SSHA(t *testing.T) {
	if dt := DetectHashType("{SSHA}base64datahere", ""); dt != "ssha" {
		t.Errorf("expected ssha, got %s", dt)
	}
}

func TestDetectHashType_Plaintext(t *testing.T) {
	if dt := DetectHashType("shortpw", ""); dt != "plaintext" {
		t.Errorf("expected plaintext, got %s", dt)
	}
}

func TestDetectHashType_Explicit(t *testing.T) {
	if dt := DetectHashType("somedata", "bcrypt"); dt != "bcrypt" {
		t.Errorf("explicit type should override detection, got %s", dt)
	}
}

func TestDetectHashType_Unknown(t *testing.T) {
	if dt := DetectHashType("$unknown$hash$format$1234567890123456", ""); dt != "unknown" {
		t.Errorf("expected unknown, got %s", dt)
	}
}

func TestVerifyMultiHash_NeedsRehash(t *testing.T) {
	// Non-argon2id hashes should flag needsRehash=true.
	_, needsRehash := VerifyMultiHash("password", "$2a$10$abc123def456")
	if !needsRehash {
		t.Error("bcrypt hash should require rehash")
	}
}
