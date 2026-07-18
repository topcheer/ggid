// Package crypto provides cryptographic utilities for GGID.
// Includes password hashing (Argon2id), token generation, and AES encryption.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters — tuned for interactive login (<100ms target).
// OWASP-recommended first-line params: m=19456, t=2, p=1.
// Previous: m=65536, t=3, p=2 (~250ms) — too slow for interactive login.
// Stored hashes embed their params, so existing hashes verify with their
// original settings. Only new password sets use these tuned params.
const (
	argonMemory      = 19 * 1024 // 19 MB (OWASP recommendation)
	argonIterations  = 2         // 2 passes (OWASP recommendation)
	argonParallelism = 1         // single lane (low-overhead)
	argonKeyLength   = 32
	argonSaltLength  = 16
)

// argonParams returns effective params with env overrides for tuning.
// ARGON2_ITERATIONS, ARGON2_MEMORY_KB, ARGON2_PARALLELISM override defaults.
func argonParams() (iter, mem, par, keyLen uint32) {
	iter, mem, par, keyLen = uint32(argonIterations), uint32(argonMemory), uint32(argonParallelism), uint32(argonKeyLength)
	if v := os.Getenv("ARGON2_ITERATIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			iter = uint32(n)
		}
	}
	if v := os.Getenv("ARGON2_MEMORY_KB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1024 {
			mem = uint32(n)
		}
	}
	if v := os.Getenv("ARGON2_PARALLELISM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			par = uint32(n)
		}
	}
	return
}

// testFastHash controls fast hashing for tests (set via init in test files).
// When true, uses minimal Argon2id params (1 iteration, 4KB memory) to avoid
// timeouts when hundreds of tests call HashPassword under the race detector.
var testFastHash bool

// EnableTestFastHash enables fast password hashing for tests.
// This MUST only be called from test files (TestMain or init()).
func EnableTestFastHash() {
	testFastHash = true
}

// pepper is an optional HMAC-SHA256 pre-hash pepper.
// When set, passwords are HMAC'd before Argon2id, adding a server-side secret.
// Configure via SetPepper() at startup from environment variable.
var pepper []byte

// SetPepper configures the password pepper. Must be called once at startup
// before any HashPassword/VerifyPassword calls. The pepper adds a server-side
// HMAC-SHA256 step before Argon2id hashing, protecting against rainbow table
// attacks even if the database is compromised without the app config.
func SetPepper(p string) {
	if p != "" {
		pepper = []byte(p)
	}
}

// applyPepper applies HMAC-SHA256 pepper if configured.
func applyPepper(password string) []byte {
	pw := []byte(password)
	if len(pepper) > 0 {
		mac := hmac.New(sha256.New, pepper)
		mac.Write(pw)
		pw = mac.Sum(nil)
	}
	return pw
}

// HashPassword hashes a plaintext password using Argon2id.
// If pepper is set via SetPepper(), the password is HMAC-SHA256'd first.
// Returns a base64-encoded string: salt + hash, prefixed with algorithm info.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	iter, mem, par, keyLen := argonParams()
	if testFastHash {
		iter, mem, par = 1, 4*1024, 1 // Minimal params for test speed
	}
	hash := argon2.IDKey(applyPepper(password), salt, iter, mem, uint8(par), keyLen)

	// Format: argon2id$iterations$memory$parallelism$salt.hash
	encoded := fmt.Sprintf("argon2id$%d$%d$%d$%s.%s",
		iter, mem, par,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword compares a plaintext password against a stored Argon2id hash.
func VerifyPassword(password, encoded string) (bool, error) {
	var iter, mem uint32
	var par uint8
	var saltB64, hashB64 string

	_, err := fmt.Sscanf(encoded, "argon2id$%d$%d$%d$%s",
		&iter, &mem, &par, &saltB64)
	if err != nil {
		return false, fmt.Errorf("invalid hash format: %w", err)
	}

	// Split salt.hash
	parts := splitLast(saltB64, ".")
	if len(parts) != 2 {
		return false, errors.New("invalid hash encoding")
	}
	saltB64 = parts[0]
	hashB64 = parts[1]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	computedHash := argon2.IDKey(applyPepper(password), salt, iter, mem, par, uint32(len(expectedHash)))

	// Constant-time comparison
	return constantTimeCompare(computedHash, expectedHash), nil
}

// AESEncrypt encrypts plaintext using AES-256-GCM with the given key.
func AESEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(hashKey(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// AESDecrypt decrypts ciphertext using AES-256-GCM with the given key.
func AESDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(hashKey(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateRandomToken generates a URL-safe random token of the given byte length.
func GenerateRandomToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashKey derives a 32-byte AES key from an arbitrary-length key using SHA-256.
func hashKey(key []byte) []byte {
	h := sha256.Sum256(key)
	return h[:]
}

func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func splitLast(s, sep string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
