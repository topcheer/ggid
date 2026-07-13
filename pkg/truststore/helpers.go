package truststore

import (
	"crypto/sha256"
	"encoding/hex"
)

// sha256SumImpl computes SHA-256 sum.
func sha256SumImpl(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// hexEncodeImpl encodes bytes to lowercase hex string.
func hexEncodeImpl(data []byte) string {
	return hex.EncodeToString(data)
}
