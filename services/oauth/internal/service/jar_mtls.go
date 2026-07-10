package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// --- RFC 9101: JWT-Secured Authorization Request (JAR) ---

// ValidateAuthorizationRequest validates a JWT-secured authorization request
// per RFC 9101. Handles both `request` (inline JWT) and `request_uri` (PAR reference).
//
// Validation rules:
//   - If both request and request_uri present → reject (RFC 9101 §2.2)
//   - iss MUST equal client_id (RFC 9101 §3.1)
//   - aud MUST be the authorization server issuer (RFC 9101 §3.2)
//   - exp MUST be present; default max lifetime is 10 minutes (RFC 9101 §3.3)
func (s *OAuthService) ValidateAuthorizationRequest(ctx context.Context, clientID, request, requestURI string) (jwt.MapClaims, error) {
	if request != "" && requestURI != "" {
		return nil, errors.InvalidArgument("request and request_uri MUST NOT be present together")
	}

	if requestURI != "" {
		pushed, err := s.GetPushedAuthorizationRequest(requestURI)
		if err != nil {
			return nil, errors.InvalidArgument("unable to resolve request_uri")
		}
		return jwt.MapClaims{
			"iss":           pushed.ClientID,
			"response_type": pushed.ResponseType,
			"redirect_uri":  pushed.RedirectURI,
			"scope":         pushed.Scope,
			"state":         pushed.State,
			"nonce":         pushed.Nonce,
		}, nil
	}

	if request == "" {
		return nil, nil
	}

	claims := jwt.MapClaims{}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	_, _, err := parser.ParseUnverified(request, claims)
	if err != nil {
		return nil, errors.InvalidArgument("invalid request JWT: " + err.Error())
	}

	iss, _ := claims["iss"].(string)
	if iss != clientID {
		return nil, errors.InvalidArgument("request JWT iss MUST equal client_id")
	}

	if expClaim, ok := claims["exp"]; ok {
		switch exp := expClaim.(type) {
		case float64:
			if time.Now().Unix() > int64(exp) {
				return nil, errors.InvalidArgument("request JWT has expired")
			}
		case int64:
			if time.Now().Unix() > exp {
				return nil, errors.InvalidArgument("request JWT has expired")
			}
		default:
			return nil, errors.InvalidArgument("request JWT has invalid exp claim")
		}
	} else {
		return nil, errors.InvalidArgument("request JWT MUST contain exp claim")
	}

	if aud, ok := claims["aud"].(string); ok && aud != "" && aud != s.issuer {
		return nil, errors.InvalidArgument("request JWT aud MUST be the authorization server")
	}

	return claims, nil
}

// JARClaims holds the validated claims from a JWT-Secured Authorization Request.
type JARClaims struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	State               string
	Nonce               string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
}

// ValidateJARRequest validates a JAR request JWT and returns the extracted claims.
// This is the simplified entry point for direct JWT validation (without request_uri).
func (s *OAuthService) ValidateJARRequest(ctx context.Context, clientID, requestJWT string) (*JARClaims, error) {
	if requestJWT == "" {
		return nil, errors.InvalidArgument("request parameter is required")
	}

	claims := jwt.MapClaims{}
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	_, _, err := parser.ParseUnverified(requestJWT, claims)
	if err != nil {
		return nil, errors.InvalidArgument("invalid request JWT: " + err.Error())
	}

	// Validate iss == client_id.
	iss, _ := claims["iss"].(string)
	if iss != clientID {
		return nil, errors.InvalidArgument("request JWT iss MUST equal client_id")
	}

	// Validate exp — default 10 minutes if missing.
	var expiresAt time.Time
	if expClaim, ok := claims["exp"]; ok {
		switch exp := expClaim.(type) {
		case float64:
			expiresAt = time.Unix(int64(exp), 0)
		case int64:
			expiresAt = time.Unix(exp, 0)
		default:
			return nil, errors.InvalidArgument("invalid exp claim type")
		}
	} else {
		expiresAt = time.Now().Add(10 * time.Minute)
	}
	if time.Now().After(expiresAt) {
		return nil, errors.InvalidArgument("request JWT has expired")
	}

	// Validate aud.
	if aud, ok := claims["aud"].(string); ok && aud != "" && aud != s.issuer {
		return nil, errors.InvalidArgument("request JWT aud MUST be the authorization server")
	}

	return &JARClaims{
		ClientID:            clientID,
		RedirectURI:         getStringClaim(claims, "redirect_uri"),
		ResponseType:        getStringClaim(claims, "response_type"),
		State:               getStringClaim(claims, "state"),
		Nonce:               getStringClaim(claims, "nonce"),
		Scope:               getStringClaim(claims, "scope"),
		CodeChallenge:       getStringClaim(claims, "code_challenge"),
		CodeChallengeMethod: getStringClaim(claims, "code_challenge_method"),
		ExpiresAt:           expiresAt,
	}, nil
}

// RejectBothRequestParams implements RFC 9101 §2.2: if both request and
// request_uri are present in the authorization request, the server MUST reject.
func RejectBothRequestParams(requestParam, requestURI string) error {
	if requestParam != "" && requestURI != "" {
		return errors.InvalidArgument("request and request_uri MUST NOT be both present")
	}
	return nil
}

// --- RFC 8705: mTLS Sender-Constrained Tokens ---

const (
	// ClientAuthMethodTLS is the mTLS client authentication method (RFC 8705 §2).
	ClientAuthMethodTLS = "tls_client_auth"
	// ClientAuthMethodSelfSignedTLS is the self-signed certificate mTLS method.
	ClientAuthMethodSelfSignedTLS = "self_signed_tls_client_auth"
)

// ExtractCertThumbprint extracts the x5t#S256 thumbprint from a TLS client
// certificate's DER-encoded bytes. Returns "x5t#S256:<sha256-hash>".
func ExtractCertThumbprint(certDER []byte) string {
	if len(certDER) == 0 {
		return ""
	}
	return "x5t#S256:" + hashTokenSHA256(string(certDER))
}

// ValidateMTLSClientAuth validates that the access token's cnf.x5t#S256 claim
// matches the TLS client certificate thumbprint from the request.
// This implements sender-constrained token verification per RFC 8705 §3.
func ValidateMTLSClientAuth(claims jwt.MapClaims, certThumbprint string) error {
	if certThumbprint == "" {
		return fmt.Errorf("no client certificate provided")
	}

	cnfRaw, ok := claims["cnf"]
	if !ok {
		return fmt.Errorf("token is not sender-constrained (missing cnf claim)")
	}

	cnf, ok := cnfRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid cnf claim format")
	}

	x5t, ok := cnf["x5t#S256"].(string)
	if !ok || x5t == "" {
		return fmt.Errorf("token not bound to client certificate (no x5t#S256)")
	}

	if !strings.EqualFold(x5t, certThumbprint) {
		return fmt.Errorf("client certificate thumbprint mismatch")
	}

	return nil
}

// ValidateMTLSBinding validates that the access token's cnf.x5t#S256 claim
// matches the TLS client certificate thumbprint from the request (RFC 8705 §3).
func (s *OAuthService) ValidateMTLSBinding(tokenStr, certThumbprint string) error {
	if tokenStr == "" || certThumbprint == "" {
		return fmt.Errorf("token and certificate thumbprint are required")
	}

	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return fmt.Errorf("invalid access token")
	}

	return ValidateMTLSClientAuth(claims, certThumbprint)
}

// IsMTLSClient checks if the client uses mTLS sender-constrained tokens.
func IsMTLSClient(client *domain.OAuthClient) bool {
	return client.TokenEndpointAuthMethod == ClientAuthMethodTLS ||
		client.TokenEndpointAuthMethod == ClientAuthMethodSelfSignedTLS
}
