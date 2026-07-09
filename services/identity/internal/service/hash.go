package service

import (
	"crypto/sha256"
	"encoding/hex"
)

// hashTokenSHA256 returns a hex-encoded SHA-256 hash of the plaintext token.
func hashTokenSHA256(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
