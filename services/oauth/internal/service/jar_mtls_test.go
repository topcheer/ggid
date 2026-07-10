package service

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// --- JAR (RFC 9101) Tests ---

func makeJARJWT(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	header := `{"alg":"none","typ":"JWT"}`
	payload := `{"iss":"` + claims["iss"].(string)
	if v, ok := claims["aud"].(string); ok {
		payload += `","aud":"` + v
	}
	if v, ok := claims["redirect_uri"].(string); ok {
		payload += `","redirect_uri":"` + v
	}
	if v, ok := claims["response_type"].(string); ok {
		payload += `","response_type":"` + v
	}
	if v, ok := claims["state"].(string); ok {
		payload += `","state":"` + v
	}
	if v, ok := claims["scope"].(string); ok {
		payload += `","scope":"` + v
	}
	if v, ok := claims["nonce"].(string); ok {
		payload += `","nonce":"` + v
	}
	if v, ok := claims["code_challenge"].(string); ok {
		payload += `","code_challenge":"` + v
	}
	if v, ok := claims["code_challenge_method"].(string); ok {
		payload += `","code_challenge_method":"` + v
	}
	if v, ok := claims["exp"].(float64); ok {
		payload += `","exp":` + strings.TrimSpace(strings.Replace(
			strings.Replace(` `+formatFloat(v), " ", "", -1), "\n", "", -1))
	}
	payload += `"}`
	h := base64.RawURLEncoding.EncodeToString([]byte(header))
	p := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return h + "." + p + "."
}

func formatFloat(f float64) string {
	return strings.TrimSpace(strings.Replace(
		strings.Replace(` `+formatFloatHelper(f), " ", "", -1), "\n", "", -1))
}

func formatFloatHelper(f float64) string {
	// Simple JSON number formatting
	if f == float64(int64(f)) {
		return formatInt(int64(f))
	}
	return formatFloat(f)
}

func formatInt(i int64) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func TestJAR_ValidRequest(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	claims := jwt.MapClaims{
		"iss":          "client-123",
		"aud":          "https://test.ggid.dev",
		"redirect_uri": "https://app.example.com/callback",
		"response_type": "code",
		"state":        "xyz",
		"scope":        "openid profile",
	}

	tokenStr := makeSimpleJARJWT(claims)
	result, err := svc.ValidateJARRequest(context.Background(), "client-123", tokenStr)
	if err != nil {
		t.Fatalf("ValidateJARRequest: %v", err)
	}
	if result.ClientID != "client-123" {
		t.Errorf("expected client-123, got %s", result.ClientID)
	}
	if result.RedirectURI != "https://app.example.com/callback" {
		t.Errorf("expected redirect_uri, got %s", result.RedirectURI)
	}
}

func TestJAR_EmptyRequest(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateJARRequest(context.Background(), "client-1", "")
	if err == nil {
		t.Error("expected error for empty request")
	}
}

func TestJAR_InvalidJWT(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ValidateJARRequest(context.Background(), "client-1", "not.a.jwt")
	if err == nil {
		t.Error("expected error for invalid JWT")
	}
}

func TestJAR_IssMismatch(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	claims := jwt.MapClaims{
		"iss": "wrong-client",
		"aud": "https://test.ggid.dev",
	}
	tokenStr := makeSimpleJARJWT(claims)
	_, err := svc.ValidateJARRequest(context.Background(), "correct-client", tokenStr)
	if err == nil {
		t.Error("expected error for iss mismatch")
	}
}

func TestJAR_WrongAudience(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	claims := jwt.MapClaims{
		"iss": "client-1",
		"aud": "https://wrong-issuer.example.com",
	}
	tokenStr := makeSimpleJARJWT(claims)
	_, err := svc.ValidateJARRequest(context.Background(), "client-1", tokenStr)
	if err == nil {
		t.Error("expected error for wrong audience")
	}
}

func TestJAR_Expired(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	claims := jwt.MapClaims{
		"iss": "client-1",
		"aud": "https://test.ggid.dev",
		"exp": float64(1), // Unix epoch 1970
	}
	tokenStr := makeSimpleJARJWT(claims)
	_, err := svc.ValidateJARRequest(context.Background(), "client-1", tokenStr)
	if err == nil {
		t.Error("expected error for expired JWT")
	}
}

func TestRejectBothRequestParams_BothPresent(t *testing.T) {
	err := RejectBothRequestParams("request-param", "urn:ietf:params:oauth:request_uri:abc")
	if err == nil {
		t.Error("expected error when both request and request_uri present")
	}
}

func TestRejectBothRequestParams_OnlyRequest(t *testing.T) {
	err := RejectBothRequestParams("request-param", "")
	if err != nil {
		t.Errorf("expected nil: %v", err)
	}
}

func TestRejectBothRequestParams_OnlyURI(t *testing.T) {
	err := RejectBothRequestParams("", "urn:ietf:params:oauth:request_uri:abc")
	if err != nil {
		t.Errorf("expected nil: %v", err)
	}
}

// --- mTLS (RFC 8705) Tests ---

func TestExtractCertThumbprint_Empty(t *testing.T) {
	result := ExtractCertThumbprint(nil)
	if result != "" {
		t.Errorf("expected empty, got %s", result)
	}
}

func TestExtractCertThumbprint_Valid(t *testing.T) {
	result := ExtractCertThumbprint([]byte("fake-cert-der"))
	if !strings.HasPrefix(result, "x5t#S256:") {
		t.Errorf("expected x5t#S256 prefix, got %s", result)
	}
}

func TestValidateMTLSClientAuth_NoCert(t *testing.T) {
	claims := jwt.MapClaims{}
	err := ValidateMTLSClientAuth(claims, "")
	if err == nil {
		t.Error("expected error for no cert")
	}
}

func TestValidateMTLSClientAuth_NoCNF(t *testing.T) {
	claims := jwt.MapClaims{}
	err := ValidateMTLSClientAuth(claims, "x5t#S256:abc")
	if err == nil {
		t.Error("expected error for no cnf claim")
	}
}

func TestValidateMTLSClientAuth_NoX5T(t *testing.T) {
	claims := jwt.MapClaims{
		"cnf": map[string]any{"foo": "bar"},
	}
	err := ValidateMTLSClientAuth(claims, "x5t#S256:abc")
	if err == nil {
		t.Error("expected error for no x5t#S256")
	}
}

func TestValidateMTLSClientAuth_ThumbprintMismatch(t *testing.T) {
	claims := jwt.MapClaims{
		"cnf": map[string]any{"x5t#S256": "different-thumbprint"},
	}
	err := ValidateMTLSClientAuth(claims, "x5t#S256:abc")
	if err == nil {
		t.Error("expected error for thumbprint mismatch")
	}
}

func TestValidateMTLSClientAuth_Match(t *testing.T) {
	claims := jwt.MapClaims{
		"cnf": map[string]any{"x5t#S256": "x5t#S256:abc123"},
	}
	err := ValidateMTLSClientAuth(claims, "x5t#S256:abc123")
	if err != nil {
		t.Errorf("expected nil for matching thumbprint: %v", err)
	}
}

func TestIsMTLSClient_TLS(t *testing.T) {
	client := &domain.OAuthClient{
		TokenEndpointAuthMethod: ClientAuthMethodTLS,
	}
	if !IsMTLSClient(client) {
		t.Error("expected true for tls_client_auth")
	}
}

func TestIsMTLSClient_SelfSigned(t *testing.T) {
	client := &domain.OAuthClient{
		TokenEndpointAuthMethod: ClientAuthMethodSelfSignedTLS,
	}
	if !IsMTLSClient(client) {
		t.Error("expected true for self_signed_tls_client_auth")
	}
}

func TestIsMTLSClient_NotMTLS(t *testing.T) {
	client := &domain.OAuthClient{
		TokenEndpointAuthMethod: "client_secret_basic",
	}
	if IsMTLSClient(client) {
		t.Error("expected false for client_secret_basic")
	}
}

// makeSimpleJARJWT creates an unsigned JWT with the given claims for JAR testing.
func makeSimpleJARJWT(claims jwt.MapClaims) string {
	// Build JSON payload manually for reliability
	var parts []string
	parts = append(parts, `"iss":"`+getStringClaim(claims, "iss")+`"`)
	if v := getStringClaim(claims, "aud"); v != "" {
		parts = append(parts, `"aud":"`+v+`"`)
	}
	if v := getStringClaim(claims, "redirect_uri"); v != "" {
		parts = append(parts, `"redirect_uri":"`+v+`"`)
	}
	if v := getStringClaim(claims, "response_type"); v != "" {
		parts = append(parts, `"response_type":"`+v+`"`)
	}
	if v := getStringClaim(claims, "state"); v != "" {
		parts = append(parts, `"state":"`+v+`"`)
	}
	if v := getStringClaim(claims, "scope"); v != "" {
		parts = append(parts, `"scope":"`+v+`"`)
	}
	if v := getStringClaim(claims, "nonce"); v != "" {
		parts = append(parts, `"nonce":"`+v+`"`)
	}
	if v := getStringClaim(claims, "code_challenge"); v != "" {
		parts = append(parts, `"code_challenge":"`+v+`"`)
	}
	if v := getStringClaim(claims, "code_challenge_method"); v != "" {
		parts = append(parts, `"code_challenge_method":"`+v+`"`)
	}
	if exp, ok := claims["exp"].(float64); ok {
		parts = append(parts, `"exp":`+formatInt(int64(exp)))
	}

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte("{" + strings.Join(parts, ",") + "}"))
	return header + "." + payload + "."
}
