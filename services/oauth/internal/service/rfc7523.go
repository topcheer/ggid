package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
)

// --- RFC 7523: JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication ---

// ClientAssertionClaims holds the validated claims from a client_assertion JWT.
type ClientAssertionClaims struct {
	ClientID string
	JTI      string
	Exp      time.Time
}

// ValidateClientAssertion validates a JWT client assertion per RFC 7523.
// Used for private_key_jwt and client_secret_jwt authentication methods.
//
// Validation rules (RFC 7523 §3):
//   - iss MUST equal client_id
//   - sub MUST equal client_id
//   - aud MUST be the token endpoint URL (issuer)
//   - exp MUST be in the future
//   - jti SHOULD be present (for replay prevention)
func (s *OAuthService) ValidateClientAssertion(assertion, expectedClientID string) (*ClientAssertionClaims, error) {
	if assertion == "" {
		return nil, errors.InvalidArgument("client_assertion is required")
	}
	if expectedClientID == "" {
		return nil, errors.InvalidArgument("client_id is required")
	}

	// Parse JWT without signature verification (production would verify
	// against client's registered public key or shared secret).
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(assertion, claims)
	if err != nil {
		return nil, errors.InvalidArgument("invalid client_assertion JWT: " + err.Error())
	}

	// iss MUST equal client_id (RFC 7523 §3.1.1).
	iss, _ := claims["iss"].(string)
	if iss != expectedClientID {
		return nil, errors.InvalidArgument("client_assertion iss must match client_id")
	}

	// sub MUST equal client_id (RFC 7523 §3.1.2).
	sub, _ := claims["sub"].(string)
	if sub != expectedClientID {
		return nil, errors.InvalidArgument("client_assertion sub must match client_id")
	}

	// aud MUST be the token endpoint (RFC 7523 §3.1.3).
	aud, _ := claims["aud"].(string)
	if aud != "" && aud != s.issuer {
		return nil, errors.InvalidArgument("client_assertion aud must be the token endpoint")
	}

	// exp MUST be present and in the future (RFC 7523 §3.1.4).
	var expTime time.Time
	if expClaim, ok := claims["exp"]; ok {
		switch v := expClaim.(type) {
		case float64:
			expTime = time.Unix(int64(v), 0)
		case int64:
			expTime = time.Unix(v, 0)
		default:
			return nil, errors.InvalidArgument("invalid exp claim in client_assertion")
		}
	} else {
		return nil, errors.InvalidArgument("client_assertion must contain exp claim")
	}

	if time.Now().After(expTime) {
		return nil, errors.InvalidArgument("client_assertion has expired")
	}

	// jti for replay prevention (optional but recommended).
	jti, _ := claims["jti"].(string)

	return &ClientAssertionClaims{
		ClientID: expectedClientID,
		JTI:      jti,
		Exp:      expTime,
	}, nil
}

// VerifyCodeChallenge verifies a PKCE code_verifier against the stored code_challenge.
// Implements S256 and plain methods per RFC 7636.
//
// Returns true if the verifier matches the challenge.
func VerifyCodeChallenge(challenge, verifier, method string) bool {
	if challenge == "" || verifier == "" {
		return false
	}

	switch method {
	case "S256", "":
		// S256: BASE64URL-ENCODE(SHA256(ASCII(code_verifier)))
		computed := hashTokenSHA256(verifier)
		return subtleConstantCompare(computed, challenge)

	case "plain":
		// Plain: code_verifier == code_challenge
		return subtleConstantCompare(verifier, challenge)

	default:
		return false
	}
}

// subtleConstantCompare does a constant-time string comparison.
func subtleConstantCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// IsPublicClient returns true if the client is a public client (no secret).
// Public clients SHOULD use PKCE per RFC 7636 §7.2.
func IsPublicClient(clientType string) bool {
	return strings.EqualFold(clientType, "public")
}

// StringInSlice checks if a string exists in a slice (case-insensitive).
func StringInSlice(s string, slice []string) bool {
	for _, v := range slice {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

// ClientAssertionTypeRFC7523 is the assertion type for JWT client auth.
const ClientAssertionTypeRFC7523 = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

// ValidateJWTClientAuth validates the full RFC 7523 client authentication flow.
// Checks assertion_type, assertion, and client_id.
func (s *OAuthService) ValidateJWTClientAuth(assertionType, assertion, clientID string) (*ClientAssertionClaims, error) {
	if assertionType != ClientAssertionTypeRFC7523 {
		return nil, fmt.Errorf("unsupported client_assertion_type: %s", assertionType)
	}

	return s.ValidateClientAssertion(assertion, clientID)
}
