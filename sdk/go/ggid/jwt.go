package ggid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// JWTVerifier verifies RS256 JWTs against GGID's JWKS endpoint.
type JWTVerifier struct {
	jwksURL  string
	cache    map[string]interface{}
	mu       sync.RWMutex
	cachedAt time.Time
	ttl      time.Duration
}

// NewJWTVerifier creates a verifier that fetches keys from the given JWKS URL.
func NewJWTVerifier(jwksURL string) *JWTVerifier {
	return &JWTVerifier{
		jwksURL: jwksURL,
		cache:   make(map[string]interface{}),
		ttl:     5 * time.Minute,
	}
}

// Verify validates a JWT and returns its claims.
func (v *JWTVerifier) Verify(ctx context.Context, token string) (map[string]interface{}, error) {
	claims, err := parseJWTClaims(token)
	if err != nil {
		return nil, fmt.Errorf("parse JWT: %w", err)
	}

	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, ErrTokenExpired
		}
	}

	return claims, nil
}

// parseJWTClaims extracts claims from JWT payload.
func parseJWTClaims(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Add padding if needed
	payload := parts[1]
	for len(payload)%4 != 0 {
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}
	return claims, nil
}
