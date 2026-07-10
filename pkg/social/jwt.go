package social

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// parseJWTClaims extracts claims from a JWT without verification (caller must trust the source).
func parseJWTClaims(jwtStr string) (map[string]any, error) {
	parts := splitJWT(jwtStr)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Try with padding
		padded := parts[1]
		for len(padded)%4 != 0 {
			padded += "="
		}
		payload, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return nil, fmt.Errorf("decode JWT payload: %w", err)
		}
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parse JWT claims JSON: %w", err)
	}
	return claims, nil
}

func splitJWT(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
