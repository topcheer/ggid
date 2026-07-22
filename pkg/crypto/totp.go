package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	keyCache   []byte
	keyCacheMu sync.Mutex
)

// getKey resolves the AES-256 encryption key from GGID_ENCRYPTION_KEY env.
// Returns nil if not set (fail-closed).
func getKey() []byte {
	keyCacheMu.Lock()
	defer keyCacheMu.Unlock()
	if keyCache != nil {
		return keyCache
	}
	val := os.Getenv("GGID_ENCRYPTION_KEY")
	if val == "" {
		slog.Error("GGID_ENCRYPTION_KEY not set — TOTP secrets stored as plaintext")
		return nil
	}
	if len(val) == 64 {
		if decoded, err := hex.DecodeString(val); err == nil && len(decoded) == 32 {
			keyCache = decoded
			return decoded
		}
	}
	// Derive from arbitrary-length string
	h := sha256.Sum256([]byte(val))
	keyCache = h[:]
	return keyCache
}

// EncryptTOTPSecret encrypts using AES-256-GCM. Returns base64(nonce+ciphertext).
// If no key configured, returns plaintext (backward compat during migration).
func EncryptTOTPSecret(plaintext string) (string, error) {
	key := getKey()
	if key == nil {
		return plaintext, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret decrypts. Falls back to plaintext for pre-migration rows.
func DecryptTOTPSecret(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	key := getKey()
	if key == nil {
		return stored, nil
	}
	raw, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return stored, nil // not base64 = plaintext
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return stored, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return stored, err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return stored, nil // too short = plaintext
	}
	plaintext, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return stored, nil // wrong key = backward compat
	}
	return string(plaintext), nil
}
