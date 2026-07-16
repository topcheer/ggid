package crypto

import (
	"crypto/x509"
	"encoding/pem"
	"os"
)

// decodePEM decodes a PEM block from a string.
func decodePEM(pemStr string) *pem.Block {
	block, _ := pem.Decode([]byte(pemStr))
	return block
}

// parseX509PublicKey parses a DER-encoded public key (SPKI format).
func parseX509PublicKey(derBytes []byte) (any, error) {
	return x509.ParsePKIXPublicKey(derBytes)
}

// osReadFile wraps os.ReadFile for testability.
var osReadFile = os.ReadFile

// getEnv wraps os.Getenv for testability.
var getEnv = os.Getenv
