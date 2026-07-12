package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type DPoPHeader struct {
	Type      string                 `json:"typ"`
	Algorithm string                 `json:"alg"`
	KeyJWK    map[string]string      `json:"jwk"`
}

type DPoPClaims struct {
	JWTID      string `json:"jti"`
	HTTPMethod string `json:"htm"`
	HTTPURI    string `json:"htu"`
	IssuedAt   int64  `json:"iat"`
	Nonce      string `json:"nonce"`
}

type DPoPVerifier struct {
	mu           sync.RWMutex
	usedNonces   map[string]time.Time
	maxProofAge  time.Duration
}

func NewDPoPVerifier() *DPoPVerifier {
	return &DPoPVerifier{
		usedNonces:  make(map[string]time.Time),
		maxProofAge: 60 * time.Second,
	}
}

func (v *DPoPVerifier) VerifyDPoPProof(proofJWT, httpMethod, url, accessToken string) error {
	parts := strings.Split(proofJWT, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format")
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid header encoding: %w", err)
	}
	var header DPoPHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("invalid header JSON: %w", err)
	}
	if header.Type != "dpop+jwt" {
		return fmt.Errorf("invalid token type: %s", header.Type)
	}

	// Decode claims
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("invalid claims encoding: %w", err)
	}
	var claims DPoPClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return fmt.Errorf("invalid claims JSON: %w", err)
	}

	// Verify htm
	if claims.HTTPMethod != httpMethod {
		return fmt.Errorf("htm mismatch: expected %s, got %s", httpMethod, claims.HTTPMethod)
	}

	// Verify htu
	if claims.HTTPURI != url {
		return fmt.Errorf("htu mismatch: expected %s, got %s", url, claims.HTTPURI)
	}

	// Verify timestamp (iat within 60 seconds)
	issuedAt := time.Unix(claims.IssuedAt, 0)
	if time.Since(issuedAt) > v.maxProofAge {
		return fmt.Errorf("proof token expired")
	}

	// Check nonce replay
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, used := v.usedNonces[claims.JWTID]; used {
		return fmt.Errorf("nonce reuse detected (replay attack)")
	}
	v.usedNonces[claims.JWTID] = time.Now()
	// Cleanup old nonces
	cutoff := time.Now().Add(-v.maxProofAge * 2)
	for nonce, t := range v.usedNonces {
		if t.Before(cutoff) {
			delete(v.usedNonces, nonce)
		}
	}

	// Verify JWK exists (public key extraction)
	if len(header.KeyJWK) == 0 {
		return fmt.Errorf("missing jwk in header")
	}

	return nil
}

func (v *DPoPVerifier) ExtractPublicKey(proofJWT string) (map[string]string, error) {
	parts := strings.Split(proofJWT, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header DPoPHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}
	return header.KeyJWK, nil
}