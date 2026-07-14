package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Claims represents the decoded JWT payload.
type Claims struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Audience  string `json:"aud"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	TenantID  string `json:"tenant_id"`
	ClientID  string `json:"client_id"`
	Scope     string `json:"scope"`
}

// verifyJWT verifies a JWT token against the GGID JWKS without external
// dependencies. It decodes the header, fetches the matching JWK, and
// verifies the RSA signature.
func verifyJWT(token string, jwksCache *JWKSKeyCache) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format: expected 3 segments, got %d", len(parts))
	}

	// 1. Decode header to get kid
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	var header map[string]any
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	kid, ok := header["kid"].(string)
	if !ok || kid == "" {
		return nil, fmt.Errorf("token header missing kid")
	}

	alg, _ := header["alg"].(string)
	if alg != "RS256" {
		return nil, fmt.Errorf("unsupported algorithm: %s (expected RS256)", alg)
	}

	// 2. Decode payload to check expiry early
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	// 3. Check expiry
	if claims.ExpiresAt > 0 && time.Now().Unix() >= claims.ExpiresAt {
		return nil, fmt.Errorf("token expired at %d", claims.ExpiresAt)
	}

	// 4. Get the JWK for this kid
	keyData, err := jwksCache.GetKey(kid)
	if err != nil {
		return nil, fmt.Errorf("find signing key: %w", err)
	}

	// 5. Verify the signature
	// For RS256, we need the RSA public key from the JWK (n and e)
	nStr, ok := keyData["n"].(string)
	if !ok {
		return nil, fmt.Errorf("JWK missing 'n' parameter")
	}
	eStr, ok := keyData["e"].(string)
	if !ok {
		return nil, fmt.Errorf("JWK missing 'e' parameter")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("decode JWK n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("decode JWK e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	rsaKey, err := buildRSAPublicKey(n, e)
	if err != nil {
		return nil, fmt.Errorf("build RSA public key: %w", err)
	}

	// Verify signature: parts[0].parts[1] is the signing input
	signingInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	if err := verifyRS256Signature(rsaKey, []byte(signingInput), signature); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return &claims, nil
}
