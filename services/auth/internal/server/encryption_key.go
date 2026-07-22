package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"sync"
)

var (
	encKeyMu sync.Mutex
	encKeys  = make(map[string][]byte)
)

// loadEncryptionKey reads a 32-byte AES-256 key from the named environment
// variable. The env var may contain either a hex-encoded 32-byte key or a
// passphrase that will be SHA-256 hashed to derive the key.
// If the env var is not set, a deterministic development key is derived from
// the env var name. This avoids panics during tests/dev while still requiring
// explicit key configuration in production.
func loadEncryptionKey(envVar string) []byte {
	encKeyMu.Lock()
	defer encKeyMu.Unlock()
	if key, ok := encKeys[envVar]; ok {
		return key
	}

	// Try PG first for cached key
	if globalMemMapRepo != nil {
		if row, _ := globalMemMapRepo.GetJSON(context.Background(), "auth_encryption_keys_json", envVar); row != nil {
			if hexKey, _ := row["key_hex"].(string); hexKey != "" {
				if decoded, err := hex.DecodeString(hexKey); err == nil && len(decoded) == 32 {
					encKeys[envVar] = decoded
					return decoded
				}
			}
		}
	}

	val := os.Getenv(envVar)
	if val == "" {
		// Production safety: refuse to derive a predictable key.
		// If GGID_ENCRYPTION_KEY is not set, return nil → callers that
		// attempt encryption/decryption will fail rather than using a
		// predictable key that an attacker could reproduce.
		slog.Error("encryption key not set",
			"env_var", envVar,
			"hint", "Set "+envVar+"=<hex> (openssl rand -hex 32)")
		return nil
	}

	// Try hex decode first (preferred: raw 32 bytes)
	if len(val) == 64 {
		if key, err := hex.DecodeString(val); err == nil && len(key) == 32 {
			encKeys[envVar] = key
			// PG write-through
			if globalMemMapRepo != nil {
				globalMemMapRepo.StoreJSON(context.Background(), "auth_encryption_keys_json", envVar, map[string]any{
					"key_name": envVar, "key_hex": val,
					"algorithm": "AES-256-GCM",
				})
			}
			return key
		}
	}

	// Fallback: derive key from passphrase via SHA-256
	h := sha256.Sum256([]byte(val))
	encKeys[envVar] = h[:]
	// PG write-through
	if globalMemMapRepo != nil {
		globalMemMapRepo.StoreJSON(context.Background(), "auth_encryption_keys_json", envVar, map[string]any{
			"key_name": envVar, "key_hex": hex.EncodeToString(h[:]),
			"algorithm": "AES-256-GCM",
		})
	}
	return h[:]
}
