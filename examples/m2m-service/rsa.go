package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
)

// buildRSAPublicKey constructs an *rsa.PublicKey from JWK modulus (n) and
// exponent (e) big integers.
func buildRSAPublicKey(n, e *big.Int) (*rsa.PublicKey, error) {
	if n == nil || e == nil {
		return nil, errors.New("nil modulus or exponent")
	}
	if n.Sign() <= 0 {
		return nil, errors.New("invalid modulus")
	}
	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// verifyRS256Signature verifies an RSA-SHA256 signature.
func verifyRS256Signature(pubKey *rsa.PublicKey, signingInput, signature []byte) error {
	if pubKey == nil {
		return errors.New("nil public key")
	}

	// Compute SHA-256 hash of the signing input
	hash := sha256.Sum256(signingInput)

	// Verify the PKCS#1 v1.5 signature
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("PKCS1v15 verification: %w", err)
	}
	return nil
}

// pemFromRSAPublicKey converts an RSA public key to PEM format (for debugging).
func pemFromRSAPublicKey(key *rsa.PublicKey) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", err
	}
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}
	return string(pem.EncodeToMemory(pemBlock)), nil
}
