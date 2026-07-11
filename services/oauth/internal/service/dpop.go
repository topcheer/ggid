package service

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DPoPProof represents a parsed and validated DPoP proof JWT (RFC 9449).
type DPoPProof struct {
	PublicKey  interface{} // ECDSA public key, RSA public key, or Ed25519 public key
	JTI        string
	IssuedAt   time.Time
	HTTPMethod string
	HTTPURI    string
	AccessToken string
}

// DPoPError describes a DPoP validation failure.
type DPoPError struct {
	Code        string
	Description string
}

func (e *DPoPError) Error() string {
	return fmt.Sprintf("dpop_%s: %s", e.Code, e.Description)
}

// ParseDPoPHeader extracts and validates the DPoP proof JWT from the
// DPoP HTTP header. The proof must be a valid JWT signed with the
// client's private key.
//
// RFC 9449 §4.2: The DPoP proof JWT contains:
//   - htm: the HTTP method (case-insensitive)
//   - htu: the HTTP URI (without query/fragment)
//   - iat: creation time (Unix timestamp)
//   - jti: unique token identifier
//   - ath (optional): hash of the access token
func ParseDPoPHeader(headerValue, httpMethod, httpURI string) (*DPoPProof, error) {
	if headerValue == "" {
		return nil, &DPoPError{Code: "missing_proof", Description: "DPoP header is required"}
	}

	// Parse the JWT without signature verification first to extract the header.
	// The header contains the "jwk" field with the public key.
	claims := jwt.MapClaims{}
	token, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseWithClaims(headerValue, claims, func(t *jwt.Token) (interface{}, error) {
		// Extract the public key from the JWT header.
		jwk, ok := t.Header["jwk"].(map[string]interface{})
		if !ok {
			return nil, &DPoPError{Code: "invalid_key", Description: "missing jwk in DPoP header"}
		}
		return extractJWKPublicKey(jwk)
	})
	if err != nil {
		return nil, &DPoPError{Code: "invalid_proof", Description: fmt.Sprintf("JWT parse/signature: %v", err)}
	}

	// Verify the signing algorithm is asymmetric (ES256, RS256, EdDSA).
	alg, _ := token.Header["alg"].(string)
	if !isAsymmetricAlgorithm(alg) {
		return nil, &DPoPError{Code: "alg_mismatch", Description: fmt.Sprintf("DPoP proof must use asymmetric algorithm, got %s", alg)}
	}

	// Extract the public key from the header for the return value.
	jwk, _ := token.Header["jwk"].(map[string]interface{})
	pubKey, _ := extractJWKPublicKey(jwk)

	proof := &DPoPProof{
		PublicKey: pubKey,
	}

	// Extract jti.
	if jti, ok := claims["jti"].(string); ok {
		proof.JTI = jti
	} else {
		return nil, &DPoPError{Code: "invalid_claims", Description: "missing jti"}
	}

	// Extract and validate iat.
	iat, ok := claims["iat"]
	if !ok {
		return nil, &DPoPError{Code: "invalid_claims", Description: "missing iat"}
	}
	switch v := iat.(type) {
	case float64:
		proof.IssuedAt = time.Unix(int64(v), 0)
	case int64:
		proof.IssuedAt = time.Unix(v, 0)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			proof.IssuedAt = time.Unix(n, 0)
		} else {
			return nil, &DPoPError{Code: "invalid_claims", Description: "invalid iat format"}
		}
	default:
		return nil, &DPoPError{Code: "invalid_claims", Description: "invalid iat type"}
	}

	// Check proof freshness (within 5 minutes).
	if time.Since(proof.IssuedAt) > 5*time.Minute {
		return nil, &DPoPError{Code: "expired", Description: "DPoP proof is too old (>5min)"}
	}

	// Extract and validate htm (HTTP method).
	htm, _ := claims["htm"].(string)
	if !strings.EqualFold(htm, httpMethod) {
		return nil, &DPoPError{Code: "htm_mismatch", Description: fmt.Sprintf("htm %q does not match %q", htm, httpMethod)}
	}
	proof.HTTPMethod = htm

	// Extract and validate htu (HTTP URI without query/fragment).
	htu, _ := claims["htu"].(string)
	if stripQueryFragment(htu) != stripQueryFragment(httpURI) {
		return nil, &DPoPError{Code: "htu_mismatch", Description: fmt.Sprintf("htu %q does not match %q", htu, httpURI)}
	}
	proof.HTTPURI = htu

	// Extract ath (access token hash) if present.
	if ath, ok := claims["ath"].(string); ok {
		proof.AccessToken = ath
	}

	return proof, nil
}

// ValidateDPoPForToken validates a DPoP proof for a token request or
// protected resource access. Checks htm, htu, and optionally ath.
func ValidateDPoPForToken(r *http.Request, accessToken string) (*DPoPProof, error) {
	dpopHeader := r.Header.Get("DPoP")
	if dpopHeader == "" {
		return nil, &DPoPError{Code: "missing_proof", Description: "DPoP header required for sender-constrained token"}
	}

	// Construct the full URI (without query/fragment for htu validation).
	fullURI := r.URL.String()
	if r.TLS == nil && r.Host != "" {
		fullURI = "http://" + r.Host + r.URL.Path
	} else if r.Host != "" {
		fullURI = "https://" + r.Host + r.URL.Path
	}

	proof, err := ParseDPoPHeader(dpopHeader, r.Method, fullURI)
	if err != nil {
		return nil, err
	}

	// If an access token is provided, validate ath.
	if accessToken != "" {
		expectedATH := hashAccessToken(accessToken)
		if proof.AccessToken != expectedATH {
			return nil, &DPoPError{Code: "ath_mismatch", Description: "DPoP ath does not match access token"}
		}
	}

	return proof, nil
}

// ComputeDPoPTokenType returns "DPoP" for DPoP-bound tokens.
const DPoPTokenType = "DPoP"

// IsDPoPTokenRequest checks if the request contains a DPoP proof header.
func IsDPoPTokenRequest(r *http.Request) bool {
	return r.Header.Get("DPoP") != ""
}

// --- Helpers ---

// isAsymmetricAlgorithm returns true for algorithms suitable for DPoP.
// Per RFC 9449 §4.2, only asymmetric algorithms are valid.
func isAsymmetricAlgorithm(alg string) bool {
	switch alg {
	case "ES256", "ES384", "ES512", "RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "EdDSA":
		return true
	default:
		return false
	}
}

// extractJWKPublicKey converts a JWK map from the JWT header into a Go crypto public key.
func extractJWKPublicKey(jwk map[string]interface{}) (interface{}, error) {
	kty, _ := jwk["kty"].(string)
	switch kty {
	case "EC":
		crv, _ := jwk["crv"].(string)
		xStr, _ := jwk["x"].(string)
		yStr, _ := jwk["y"].(string)
		xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
		if err != nil {
			return nil, fmt.Errorf("invalid EC x: %w", err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
		if err != nil {
			return nil, fmt.Errorf("invalid EC y: %w", err)
		}
		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		return &ecdsa.PublicKey{Curve: getECurve(crv), X: x, Y: y}, nil
	case "RSA":
		nStr, _ := jwk["n"].(string)
		eStr, _ := jwk["e"].(string)
		nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
		if err != nil {
			return nil, fmt.Errorf("invalid RSA n: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
		if err != nil {
			return nil, fmt.Errorf("invalid RSA e: %w", err)
		}
		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)
		return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
	case "OKP":
		crv, _ := jwk["crv"].(string)
		xStr, _ := jwk["x"].(string)
		xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
		if err != nil {
			return nil, fmt.Errorf("invalid OKP x: %w", err)
		}
		if crv == "Ed25519" {
			return ed25519.PublicKey(xBytes), nil
		}
		return nil, fmt.Errorf("unsupported OKP curve: %s", crv)
	default:
		return nil, fmt.Errorf("unsupported JWK kty: %s", kty)
	}
}

func getECurve(crv string) elliptic.Curve {
	switch crv {
	case "P-256":
		return elliptic.P256()
	case "P-384":
		return elliptic.P384()
	case "P-521":
		return elliptic.P521()
	default:
		return elliptic.P256()
	}
}

// stripQueryFragment removes query params and fragments from a URI.
func stripQueryFragment(uri string) string {
	if i := strings.Index(uri, "?"); i >= 0 {
		uri = uri[:i]
	}
	if i := strings.Index(uri, "#"); i >= 0 {
		uri = uri[:i]
	}
	return strings.TrimRight(uri, "/")
}

// hashAccessToken computes the base64url-encoded SHA-256 hash of the access token.
func hashAccessToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
