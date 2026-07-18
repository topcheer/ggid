// Package multihash provides multi-format password hash verification.
// It supports bcrypt, PBKDF2, scrypt, SSHA, and Argon2id formats,
// enabling migration from legacy systems with transparent re-hashing to Argon2id.
package multihash

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
	"crypto/sha1"
	"crypto/sha256"
)

// Format names for supported hash algorithms.
const (
	FormatBcrypt   = "bcrypt"
	FormatPBKDF2   = "pbkdf2"
	FormatScrypt   = "scrypt"
	FormatSSHA     = "ssha"
	FormatArgon2id = "argon2id"
	FormatUnknown  = "unknown"
)

// ErrInvalidHash indicates the hash string is malformed or uses an unsupported format.
var ErrInvalidHash = errors.New("invalid or unsupported hash format")

// DetectFormat inspects a hash string and returns the algorithm name.
// Supported prefixes:
//   - "$2a$", "$2b$", "$2y$" → bcrypt
//   - "$pbkdf2$"             → PBKDF2 (passlib format)
//   - "$scrypt$"             → scrypt (passlib format)
//   - "{SSHA}", "{ssha}"     → SSHA (LDAP)
//   - "$argon2id$"           → Argon2id (PHC format)
//   - "argon2id$"            → Argon2id (GGID internal format)
func DetectFormat(encoded string) string {
	switch {
	case strings.HasPrefix(encoded, "$2a$"), strings.HasPrefix(encoded, "$2b$"), strings.HasPrefix(encoded, "$2y$"):
		return FormatBcrypt
	case strings.HasPrefix(encoded, "$pbkdf2"), strings.HasPrefix(encoded, "$pbkdf2-"):
		return FormatPBKDF2
	case strings.HasPrefix(encoded, "$scrypt$"):
		return FormatScrypt
	case strings.HasPrefix(encoded, "{SSHA}"), strings.HasPrefix(encoded, "{ssha}"):
		return FormatSSHA
	case strings.HasPrefix(encoded, "$argon2id$"), strings.HasPrefix(encoded, "argon2id$"):
		return FormatArgon2id
	default:
		return FormatUnknown
	}
}

// VerifyPassword checks a plaintext password against a stored hash of any supported format.
// Returns (matched bool, format string, err error).
// The format is returned even on failure for logging/metrics.
func VerifyPassword(password, encoded string) (bool, string, error) {
	format := DetectFormat(encoded)
	switch format {
	case FormatBcrypt:
		ok, err := verifyBcrypt(password, encoded)
		return ok, FormatBcrypt, err
	case FormatPBKDF2:
		ok, err := verifyPBKDF2(password, encoded)
		return ok, FormatPBKDF2, err
	case FormatScrypt:
		ok, err := verifyScrypt(password, encoded)
		return ok, FormatScrypt, err
	case FormatSSHA:
		ok, err := verifySSHA(password, encoded)
		return ok, FormatSSHA, err
	case FormatArgon2id:
		ok, err := verifyArgon2id(password, encoded)
		return ok, FormatArgon2id, err
	default:
		return false, FormatUnknown, ErrInvalidHash
	}
}

// NeedsRehash returns true if the hash is NOT in Argon2id format and should be re-hashed.
func NeedsRehash(encoded string) bool {
	return DetectFormat(encoded) != FormatArgon2id
}

// --- bcrypt ---

func verifyBcrypt(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return false, nil // bcrypt returns error on mismatch; not a real error
	}
	return true, nil
}

// --- PBKDF2 (passlib format: $pbkdf2$iterations$saltHex$hashHex) ---

func verifyPBKDF2(password, encoded string) (bool, error) {
	// Format: $pbkdf2-sha256$iterations$saltBase64$hashBase64
	// Also support simple format: $pbkdf2$iter$saltHex$hashHex
	parts := strings.Split(encoded, "$")
	if len(parts) < 5 {
		return false, fmt.Errorf("pbkdf2: invalid format")
	}

	// parts[0] = "", parts[1] = "pbkdf2-sha256" or "pbkdf2", parts[2] = iterations
	iterations := 0
	if _, err := fmt.Sscanf(parts[2], "%d", &iterations); err != nil || iterations < 1 {
		return false, fmt.Errorf("pbkdf2: invalid iterations: %w", err)
	}

	saltStr := parts[3]
	hashStr := parts[4]

	// Try hex first (common in LDAP/legacy), then base64.
	salt, err := hex.DecodeString(saltStr)
	if err != nil {
		salt, err = base64.StdEncoding.DecodeString(saltStr)
		if err != nil {
			return false, fmt.Errorf("pbkdf2: invalid salt encoding: %w", err)
		}
	}

	expectedHash, err := hex.DecodeString(hashStr)
	if err != nil {
		expectedHash, err = base64.StdEncoding.DecodeString(hashStr)
		if err != nil {
			return false, fmt.Errorf("pbkdf2: invalid hash encoding: %w", err)
		}
	}

	// Determine hash function by variant in parts[1].
	h := sha256.New
	keyLen := len(expectedHash)

	computed := pbkdf2.Key([]byte(password), salt, iterations, keyLen, h)
	return subtle.ConstantTimeCompare(computed, expectedHash) == 1, nil
}

// --- scrypt (passlib format: $scrypt$N$r$p$saltHex$hashHex) ---

func verifyScrypt(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) < 7 {
		return false, fmt.Errorf("scrypt: invalid format")
	}

	var N, r, p int
	if _, err := fmt.Sscanf(parts[2], "%d", &N); err != nil {
		return false, fmt.Errorf("scrypt: invalid N: %w", err)
	}
	if _, err := fmt.Sscanf(parts[3], "%d", &r); err != nil {
		return false, fmt.Errorf("scrypt: invalid r: %w", err)
	}
	if _, err := fmt.Sscanf(parts[4], "%d", &p); err != nil {
		return false, fmt.Errorf("scrypt: invalid p: %w", err)
	}

	saltStr := parts[5]
	hashStr := parts[6]

	salt, err := hex.DecodeString(saltStr)
	if err != nil {
		salt, err = base64.StdEncoding.DecodeString(saltStr)
		if err != nil {
			return false, fmt.Errorf("scrypt: invalid salt encoding: %w", err)
		}
	}

	expectedHash, err := hex.DecodeString(hashStr)
	if err != nil {
		expectedHash, err = base64.StdEncoding.DecodeString(hashStr)
		if err != nil {
			return false, fmt.Errorf("scrypt: invalid hash encoding: %w", err)
		}
	}

	computed, err := scrypt.Key([]byte(password), salt, N, r, p, len(expectedHash))
	if err != nil {
		return false, fmt.Errorf("scrypt: key derivation failed: %w", err)
	}
	return subtle.ConstantTimeCompare(computed, expectedHash) == 1, nil
}

// --- SSHA (LDAP: {SSHA}base64(salt+sha1(password+salt))) ---

func verifySSHA(password, encoded string) (bool, error) {
	if !strings.HasPrefix(encoded, "{SSHA}") && !strings.HasPrefix(encoded, "{ssha}") {
		return false, fmt.Errorf("ssha: missing {SSHA} prefix")
	}

	raw := encoded[strings.Index(encoded, "}")+1:]
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return false, fmt.Errorf("ssha: invalid base64: %w", err)
	}

	if len(data) < sha1.Size {
		return false, fmt.Errorf("ssha: hash too short")
	}

	hashLen := sha1.Size
	salt := data[hashLen:]
	expectedHash := data[:hashLen]

	h := sha1.New()
	h.Write([]byte(password))
	h.Write(salt)
	computed := h.Sum(nil)

	return subtle.ConstantTimeCompare(computed, expectedHash) == 1, nil
}

// --- Argon2id ($argon2id$ PHC format or argon2id$ GGID format) ---

func verifyArgon2id(password, encoded string) (bool, error) {
	// GGID internal format: argon2id$iter$mem$par$saltB64.hashB64
	if strings.HasPrefix(encoded, "argon2id$") {
		return verifyGGIDArgon2id(password, encoded)
	}

	// PHC standard format: $argon2id$v=19$m=...,t=...,p=...$saltB64$hashB64
	if strings.HasPrefix(encoded, "$argon2id$") {
		return verifyPHCArgon2id(password, encoded)
	}

	return false, ErrInvalidHash
}

func verifyGGIDArgon2id(password, encoded string) (bool, error) {
	var iter, mem, par int
	var saltB64, hashB64 string

	_, err := fmt.Sscanf(encoded, "argon2id$%d$%d$%d$%s",
		&iter, &mem, &par, &saltB64)
	if err != nil {
		return false, fmt.Errorf("argon2id: invalid GGID format: %w", err)
	}

	// saltB64 may contain the hash after a dot
	parts := strings.SplitN(saltB64, ".", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("argon2id: missing hash separator")
	}
	saltB64 = parts[0]
	hashB64 = parts[1]

	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return false, fmt.Errorf("argon2id: invalid salt base64: %w", err)
	}

	expectedHash, err := base64.StdEncoding.DecodeString(hashB64)
	if err != nil {
		return false, fmt.Errorf("argon2id: invalid hash base64: %w", err)
	}

	computed := argon2.IDKey([]byte(password), salt, uint32(iter), uint32(mem), uint8(par), uint32(len(expectedHash)))
	return subtle.ConstantTimeCompare(computed, expectedHash) == 1, nil
}

func verifyPHCArgon2id(password, encoded string) (bool, error) {
	// PHC format: $argon2id$v=19$m=65536,t=3,p=2$<base64 salt>$<base64 hash>
	parts := strings.Split(encoded, "$")
	if len(parts) < 5 {
		return false, fmt.Errorf("argon2id: invalid PHC format")
	}

	// parts[2] = "v=19"
	// parts[3] = "m=65536,t=3,p=2"
	params := parts[3]
	var mem, t, p uint32
	for _, kv := range strings.Split(params, ",") {
		kvParts := strings.SplitN(kv, "=", 2)
		if len(kvParts) != 2 {
			continue
		}
		var val uint32
		_, _ = fmt.Sscanf(kvParts[1], "%d", &val)
		switch kvParts[0] {
		case "m":
			mem = val
		case "t":
			t = val
		case "p":
			p = val
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		salt, err = base64.StdEncoding.DecodeString(parts[4])
		if err != nil {
			return false, fmt.Errorf("argon2id: invalid PHC salt: %w", err)
		}
	}

	if len(parts) < 6 {
		return false, fmt.Errorf("argon2id: missing PHC hash")
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		expectedHash, err = base64.StdEncoding.DecodeString(parts[5])
		if err != nil {
			return false, fmt.Errorf("argon2id: invalid PHC hash: %w", err)
		}
	}

	computed := argon2.IDKey([]byte(password), salt, t, mem, uint8(p), uint32(len(expectedHash)))
	return subtle.ConstantTimeCompare(computed, expectedHash) == 1, nil
}
