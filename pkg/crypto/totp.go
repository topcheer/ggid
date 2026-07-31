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
	"strings"
	"sync"
)

var (
	keyCache   []byte
	keyCacheMu sync.Mutex
)

// encryptedPrefix marks TOTP secrets that were encrypted with AES-GCM.
// Values without this prefix are treated as legacy plaintext for backward compat.
const encryptedPrefix = "enc:"

// getKey resolves the AES-256 encryption key from GGID_ENCRYPTION_KEY env.
// Returns nil if not set (caller must handle — EncryptTOTPSecret is fail-closed).
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
// SECURITY: Returns error if no encryption key is configured (fail-closed).
func EncryptTOTPSecret(plaintext string) (string, error) {
	key := getKey()
	if key == nil {
		return "", fmt.Errorf("GGID_ENCRYPTION_KEY not set — refusing to store TOTP secret as plaintext")
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
	return encryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret decrypts. Falls back to plaintext only for legacy rows
// (values without the "enc:" prefix). Encrypted values that fail decryption
// return an error rather than silently exposing potentially tampered data.
func DecryptTOTPSecret(stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	// Legacy plaintext: no prefix. Safe fallback for pre-migration rows.
	if !strings.HasPrefix(stored, encryptedPrefix) {
		return stored, nil
	}
	// Encrypted value: strip prefix and decrypt. Fail on errors — don't
	// silently return tampered data as plaintext.
	key := getKey()
	if key == nil {
		return "", fmt.Errorf("encryption key not configured")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encryptedPrefix))
	if err != nil {
		return "", fmt.Errorf("invalid TOTP secret encoding: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("TOTP secret too short")
	}
	plaintext, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("TOTP secret decryption failed: %w", err)
	}
	return string(plaintext), nil
}
