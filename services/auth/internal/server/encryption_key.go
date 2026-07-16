package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
// Panics at startup if the env var is not set.
func loadEncryptionKey(envVar string) []byte {
	encKeyMu.Lock()
	defer encKeyMu.Unlock()
	if key, ok := encKeys[envVar]; ok {
		return key
	}

	val := os.Getenv(envVar)
	if val == "" {
		panic(fmt.Sprintf(
			"FATAL: %s environment variable not set. "+
				"Generate a key with: openssl rand -hex 32 "+
				"and set %s=<hex> in your environment.",
			envVar, envVar,
		))
	}

	// Try hex decode first (preferred: raw 32 bytes)
	if len(val) == 64 {
		if key, err := hex.DecodeString(val); err == nil && len(key) == 32 {
			encKeys[envVar] = key
			return key
		}
	}

	// Fallback: derive key from passphrase via SHA-256
	h := sha256.Sum256([]byte(val))
	encKeys[envVar] = h[:]
	return h[:]
}
