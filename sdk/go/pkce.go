package ggid

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCEPair holds a PKCE code verifier, challenge, and method.
type PKCEPair struct {
	CodeVerifier  string
	CodeChallenge string
	Method        string
}

// GenerateCodeVerifier generates a cryptographically random PKCE code verifier
// per RFC 7636 §4.1 (43-128 chars from unreserved set).
func GenerateCodeVerifier() string {
	b := make([]byte, 64)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b) // 86 chars
}

// GenerateCodeChallenge computes the S256 code challenge from a verifier.
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// GeneratePKCEPair creates a complete PKCE pair with S256 method.
func GeneratePKCEPair() *PKCEPair {
	v := GenerateCodeVerifier()
	return &PKCEPair{
		CodeVerifier:  v,
		CodeChallenge: GenerateCodeChallenge(v),
		Method:        "S256",
	}
}
