package middleware

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"

	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
)

// publicKeyToJWK converts a crypto.PublicKey to a JWK map suitable for JWKS responses.
// Supports RSA and ECDSA public keys.
func publicKeyToJWK(kid string, pub crypto.PublicKey) (map[string]any, error) {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return map[string]any{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": kid,
			"n":   base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes()),
		}, nil
	case *ecdsa.PublicKey:
		byteLen := (k.Curve.Params().BitSize + 7) / 8
		return map[string]any{
			"kty": "EC",
			"use": "sig",
			"alg": jwtAlgorithmForECDSA(k.Curve),
			"kid": kid,
			"x":   base64.RawURLEncoding.EncodeToString(padBytes(k.X.Bytes(), byteLen)),
			"y":   base64.RawURLEncoding.EncodeToString(padBytes(k.Y.Bytes(), byteLen)),
			"crv": crvForECDSA(k.Curve),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported public key type: %T", pub)
	}
}

func jwtAlgorithmForECDSA(curve elliptic.Curve) string {
	switch curve {
	case elliptic.P256():
		return "ES256"
	case elliptic.P384():
		return "ES384"
	case elliptic.P521():
		return "ES512"
	default:
		return "ES256"
	}
}

func crvForECDSA(curve elliptic.Curve) string {
	switch curve {
	case elliptic.P256():
		return "P-256"
	case elliptic.P384():
		return "P-384"
	case elliptic.P521():
		return "P-521"
	default:
		return "P-256"
	}
}

func padBytes(b []byte, length int) []byte {
	if len(b) >= length {
		return b
	}
	padded := make([]byte, length)
	copy(padded[length-len(b):], b)
	return padded
}

// SetKeyProvider wires a KeyProvider into the JWKS client so the JWKS endpoint
// is served from the provider's public key dynamically.
func (c *JWKSClient) SetKeyProvider(kp pkgcrypto.KeyProvider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keyProvider = kp
}
