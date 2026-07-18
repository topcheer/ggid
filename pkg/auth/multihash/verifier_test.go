package multihash

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// === DetectFormat tests ===

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name   string
		hash   string
		expect string
	}{
		{"bcrypt $2b$", "$2b$10$somehashvalue", FormatBcrypt},
		{"bcrypt $2a$", "$2a$12$otherhash", FormatBcrypt},
		{"bcrypt $2y$", "$2y$05$legacyhash", FormatBcrypt},
		{"pbkdf2", "$pbkdf2$29000$salt$hash", FormatPBKDF2},
		{"scrypt", "$scrypt$1024$8$1$salt$hash", FormatScrypt},
		{"ssha upper", "{SSHA}base64data", FormatSSHA},
		{"ssha lower", "{ssha}base64data", FormatSSHA},
		{"argon2id PHC", "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$hAsh", FormatArgon2id},
		{"argon2id GGID", "argon2id$3$65536$2$salt.hash", FormatArgon2id},
		{"unknown", "plaintext", FormatUnknown},
		{"empty", "", FormatUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.hash)
			if got != tt.expect {
				t.Errorf("DetectFormat(%q) = %q, want %q", tt.hash, got, tt.expect)
			}
		})
	}
}

// === bcrypt verify ===

func TestVerifyBcrypt_Correct(t *testing.T) {
	pw := "testpw-bcrypt-1"
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt generate: %v", err)
	}
	ok, format, err := VerifyPassword(pw, string(hash))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected match")
	}
	if format != FormatBcrypt {
		t.Errorf("expected bcrypt, got %s", format)
	}
}

func TestVerifyBcrypt_Wrong(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-one"), bcrypt.MinCost)
	ok, _, _ := VerifyPassword("wrong-one", string(hash))
	if ok {
		t.Error("should not match")
	}
}

// === PBKDF2 verify ===

func TestVerifyPBKDF2_Correct(t *testing.T) {
	pw := "testpw-pbkdf2-1"
	salt := []byte("testsalt12345678")
	expected := pbkdf2.Key([]byte(pw), salt, 10000, 32, sha256.New)
	encoded := fmt.Sprintf("$pbkdf2$%d$%s$%s",
		10000, hex.EncodeToString(salt), hex.EncodeToString(expected))

	ok, format, err := VerifyPassword(pw, encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected match")
	}
	if format != FormatPBKDF2 {
		t.Errorf("expected pbkdf2, got %s", format)
	}
}

func TestVerifyPBKDF2_Wrong(t *testing.T) {
	salt := []byte("testsalt12345678")
	expected := pbkdf2.Key([]byte("correct-one"), salt, 10000, 32, sha256.New)
	encoded := fmt.Sprintf("$pbkdf2$%d$%s$%s", 10000, hex.EncodeToString(salt), hex.EncodeToString(expected))

	ok, _, _ := VerifyPassword("wrong-one", encoded)
	if ok {
		t.Error("should not match")
	}
}

// === scrypt verify ===

func TestVerifyScrypt_Correct(t *testing.T) {
	pw := "testpw-scrypt-1"
	salt := []byte("scryptsalt12345")
	N, r, p := 16, 8, 1
	expected, err := scrypt.Key([]byte(pw), salt, N, r, p, 32)
	if err != nil {
		t.Fatalf("scrypt key: %v", err)
	}
	encoded := fmt.Sprintf("$scrypt$%d$%d$%d$%s$%s",
		N, r, p, hex.EncodeToString(salt), hex.EncodeToString(expected))

	ok, format, err := VerifyPassword(pw, encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected match")
	}
	if format != FormatScrypt {
		t.Errorf("expected scrypt, got %s", format)
	}
}

func TestVerifyScrypt_Wrong(t *testing.T) {
	salt := []byte("scryptsalt12345")
	expected, _ := scrypt.Key([]byte("correct-one"), salt, 16, 8, 1, 32)
	encoded := fmt.Sprintf("$scrypt$%d$%d$%d$%s$%s", 16, 8, 1, hex.EncodeToString(salt), hex.EncodeToString(expected))

	ok, _, _ := VerifyPassword("wrong-one", encoded)
	if ok {
		t.Error("should not match")
	}
}

// === SSHA verify ===

func TestVerifySSHA_Correct(t *testing.T) {
	pw := "testpw-ssha-1"
	salt := []byte("randomsalt12")
	h := sha1.New()
	h.Write([]byte(pw))
	h.Write(salt)
	hashed := h.Sum(nil)
	data := append(hashed, salt...)
	encoded := "{SSHA}" + base64.StdEncoding.EncodeToString(data)

	ok, format, err := VerifyPassword(pw, encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected match")
	}
	if format != FormatSSHA {
		t.Errorf("expected ssha, got %s", format)
	}
}

func TestVerifySSHA_Wrong(t *testing.T) {
	salt := []byte("randomsalt12")
	h := sha1.New()
	h.Write([]byte("correct-one"))
	h.Write(salt)
	hashed := h.Sum(nil)
	data := append(hashed, salt...)
	encoded := "{SSHA}" + base64.StdEncoding.EncodeToString(data)

	ok, _, _ := VerifyPassword("wrong-one", encoded)
	if ok {
		t.Error("should not match")
	}
}

// === Argon2id GGID format verify ===

func TestVerifyArgon2idGGID_Correct(t *testing.T) {
	pw := "testpw-argon-1"
	salt := []byte("argonsalt1234567")
	iter, mem, par := 3, 65536, 2
	hashed := argon2.IDKey([]byte(pw), salt, uint32(iter), uint32(mem), uint8(par), 32)
	encoded := fmt.Sprintf("argon2id$%d$%d$%d$%s.%s",
		iter, mem, par,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hashed))

	ok, format, err := VerifyPassword(pw, encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected match")
	}
	if format != FormatArgon2id {
		t.Errorf("expected argon2id, got %s", format)
	}
}

func TestVerifyArgon2idGGID_Wrong(t *testing.T) {
	salt := []byte("argonsalt1234567")
	hashed := argon2.IDKey([]byte("correct-one"), salt, 3, 65536, 2, 32)
	encoded := fmt.Sprintf("argon2id$%d$%d$%d$%s.%s",
		3, 65536, 2,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hashed))

	ok, _, _ := VerifyPassword("wrong-one", encoded)
	if ok {
		t.Error("should not match")
	}
}

// === NeedsRehash ===

func TestNeedsRehash(t *testing.T) {
	if NeedsRehash("argon2id$3$65536$2$salt.hash") {
		t.Error("argon2id should not need rehash")
	}
	if !NeedsRehash("$2b$10$somehash") {
		t.Error("bcrypt should need rehash")
	}
	if !NeedsRehash("{SSHA}base64data") {
		t.Error("ssha should need rehash")
	}
}

// === Unknown format ===

func TestVerifyUnknownFormat(t *testing.T) {
	ok, format, err := VerifyPassword("test", "unknownformathash")
	if ok {
		t.Error("unknown format should not verify")
	}
	if format != FormatUnknown {
		t.Errorf("expected unknown, got %s", format)
	}
	if err == nil {
		t.Error("expected error for unknown format")
	}
}
