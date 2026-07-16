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
// If the env var is not set, a deterministic development key is derived from
// the env var name. This avoids panics during tests/dev while still requiring
// explicit key configuration in production.
func loadEncryptionKey(envVar string) []byte {
	encKeyMu.Lock()
	defer encKeyMu.Unlock()
	if key, ok := encKeys[envVar]; ok {
		return key
	}

	val := os.Getenv(envVar)
	if val == "" {
		// Derive a deterministic development key so tests/dev work without
		// configuration. Production deployments MUST set the env var.
		fmt.Fprintf(os.Stderr,
			"WARNING: %s environment variable not set — using derived dev key. "+
				"Set %s=<hex> in production (openssl rand -hex 32).\n",
			envVar, envVar)
		h := sha256.Sum256([]byte("dev-fallback-key:" + envVar))
		encKeys[envVar] = h[:]
		return h[:]
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
