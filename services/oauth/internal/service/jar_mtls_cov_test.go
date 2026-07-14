package service

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Coverage tests for ValidateAuthorizationRequest and ValidateMTLSBinding (both at 0%).

// --- ValidateAuthorizationRequest coverage ---

func TestCovJAR_AuthReq_BothParams(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", "jwt", "urn:ietf:params:oauth:request_uri:x")
	if err == nil {
		t.Error("expected error when both request and request_uri present")
	}
}

func TestCovJAR_AuthReq_NeitherParam(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	claims, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", "", "")
	if err != nil {
		t.Fatalf("expected nil error: %v", err)
	}
	if claims != nil {
		t.Error("expected nil claims when no JAR")
	}
}

func TestCovJAR_AuthReq_RequestURI_Unresolvable(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", "", "urn:ietf:params:oauth:request_uri:nonexistent")
	if err == nil {
		t.Error("expected error for unresolvable request_uri")
	}
}

func TestCovJAR_AuthReq_RequestURI_Resolvable(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// Inject directly into parStore
	uri := "urn:ietf:params:oauth:request_uri:" + "test-par-resolvable"
	parStore.Store(uri, parEntry{
		Request: &PushedAuthorizationRequest{
			ClientID:     "c1",
			RedirectURI:  "https://app.example.com/callback",
			ResponseType: "code",
			Scope:        "openid",
		},
		ExpiresAt: time.Now().Add(5 * time.Minute),
	})

	claims, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", "", uri)
	if err != nil {
		t.Fatalf("expected success for resolvable request_uri: %v", err)
	}
	if claims == nil {
		t.Fatal("expected non-nil claims")
	}
	if claims["response_type"] != "code" {
		t.Errorf("expected response_type=code, got %v", claims["response_type"])
	}
}

func TestCovJAR_AuthReq_InvalidJWT(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", "not.a.jwt", "")
	if err == nil {
		t.Error("expected error for invalid JWT")
	}
}

func TestCovJAR_AuthReq_WrongIss(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeSimpleJARJWT(jwt.MapClaims{
		"iss": "wrong-client",
		"aud": "https://test.ggid.dev",
		"exp": float64(time.Now().Add(5 * time.Minute).Unix()),
	})
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "correct-client", token, "")
	if err == nil {
		t.Error("expected error for iss mismatch")
	}
}

func TestCovJAR_AuthReq_Expired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeSimpleJARJWT(jwt.MapClaims{
		"iss": "c1",
		"aud": "https://test.ggid.dev",
		"exp": float64(1),
	})
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", token, "")
	if err == nil {
		t.Error("expected error for expired JWT")
	}
}

func TestCovJAR_AuthReq_MissingExp(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeSimpleJARJWT(jwt.MapClaims{
		"iss": "c1",
		"aud": "https://test.ggid.dev",
	})
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", token, "")
	if err == nil {
		t.Error("expected error for missing exp")
	}
}

func TestCovJAR_AuthReq_WrongAud(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeSimpleJARJWT(jwt.MapClaims{
		"iss": "c1",
		"aud": "https://wrong.example.com",
		"exp": float64(time.Now().Add(5 * time.Minute).Unix()),
	})
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", token, "")
	if err == nil {
		t.Error("expected error for wrong aud")
	}
}

func TestCovJAR_AuthReq_Valid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := makeSimpleJARJWT(jwt.MapClaims{
		"iss":           "c1",
		"aud":           "https://test.ggid.dev",
		"exp":           float64(time.Now().Add(5 * time.Minute).Unix()),
		"response_type": "code",
		"scope":         "openid",
	})
	claims, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", token, "")
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if claims["response_type"] != "code" {
		t.Errorf("expected code, got %v", claims["response_type"])
	}
}

func TestCovJAR_AuthReq_InvalidExpType(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Build a JWT with a string exp (invalid type)
	header := `{"alg":"none","typ":"JWT"}`
	payload := `{"iss":"c1","aud":"https://test.ggid.dev","exp":"not-a-number"}`
	h := encodeBase64(header)
	p := encodeBase64(payload)
	token := h + "." + p + "."
	_, err := svc.ValidateAuthorizationRequest(context.Background(), "c1", token, "")
	if err == nil {
		t.Error("expected error for invalid exp type")
	}
}

// --- ValidateMTLSBinding coverage ---

func TestCovMTLS_Binding_EmptyToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	err := svc.ValidateMTLSBinding("", "thumbprint")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestCovMTLS_Binding_EmptyThumbprint(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	err := svc.ValidateMTLSBinding("token", "")
	if err == nil {
		t.Error("expected error for empty thumbprint")
	}
}

func TestCovMTLS_Binding_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	err := svc.ValidateMTLSBinding("invalid-token", "thumbprint")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestCovMTLS_Binding_NoCnfClaim(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := signTestToken(svc, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
	})
	err := svc.ValidateMTLSBinding(token, "x5t#S256:abc")
	if err == nil {
		t.Error("expected error for token without cnf claim")
	}
}

func TestCovMTLS_Binding_Matching(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	thumb := "x5t#S256:" + hashTokenSHA256("cert-data")
	token := signTestToken(svc, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
		"cnf": map[string]any{"x5t#S256": thumb},
	})
	err := svc.ValidateMTLSBinding(token, thumb)
	if err != nil {
		t.Errorf("expected success for matching thumbprint: %v", err)
	}
}

func TestCovMTLS_Binding_Mismatch(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	token := signTestToken(svc, jwt.MapClaims{
		"sub": "user-1",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iss": "https://test.ggid.dev",
		"cnf": map[string]any{"x5t#S256": "x5t#S256:different"},
	})
	err := svc.ValidateMTLSBinding(token, "x5t#S256:abc")
	if err == nil {
		t.Error("expected error for thumbprint mismatch")
	}
}

// signTestToken signs a JWT with the service's RSA key for testing.
func signTestToken(svc *OAuthService, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, _ := token.SignedString(svc.keyProvider.Signer())
	return signed
}

// encodeBase64 base64url-encodes a string without padding.
func encodeBase64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
