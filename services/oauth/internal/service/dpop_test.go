package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateDPoPKey generates an ECDSA P-256 key pair for DPoP testing.
func generateDPoPKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

// makeDPoPProof creates a signed DPoP proof JWT for testing.
func makeDPoPProof(t *testing.T, key *ecdsa.PrivateKey, jti, htm, htu string, iat time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"jti": jti,
		"htm": htm,
		"htu": htu,
		"iat": iat.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	// Add JWK public key to header.
	pub := key.Public().(*ecdsa.PublicKey)
	token.Header["jwk"] = map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64RawURL(pub.X.Bytes()),
		"y":   base64RawURL(pub.Y.Bytes()),
	}
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign proof: %v", err)
	}
	return signed
}

func base64RawURL(b []byte) string {
	return strings.TrimRight(strings.TrimRight(
		func() string {
			const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
			// Simple base64url without padding
			var sb strings.Builder
			for i := 0; i < len(b); i += 3 {
				var n uint32
				for j := 0; j < 3 && i+j < len(b); j++ {
					n |= uint32(b[i+j]) << (16 - uint(j*8))
			}
			sb.WriteByte(alphabet[(n>>18)&0x3F])
			sb.WriteByte(alphabet[(n>>12)&0x3F])
			if i+1 < len(b) {
				sb.WriteByte(alphabet[(n>>6)&0x3F])
			}
			if i+2 < len(b) {
				sb.WriteByte(alphabet[n&0x3F])
			}
		}
		return sb.String()
		}(), "="), "=")
}

func TestDPoP_MissingProof(t *testing.T) {
	_, err := ParseDPoPHeader("", "POST", "https://example.com/token")
	if err == nil {
		t.Fatal("expected error for missing proof")
	}
	if !strings.Contains(err.Error(), "missing_proof") {
		t.Errorf("expected missing_proof error, got %v", err)
	}
}

func TestDPoP_InvalidJWT(t *testing.T) {
	_, err := ParseDPoPHeader("not.a.jwt", "POST", "https://example.com/token")
	if err == nil {
		t.Fatal("expected error for invalid JWT")
	}
}

func TestDPoP_ValidProof(t *testing.T) {
	key := generateDPoPKey(t)
	proof := makeDPoPProof(t, key, "test-jti-1", "POST", "https://example.com/token", time.Now())

	parsed, err := ParseDPoPHeader(proof, "POST", "https://example.com/token")
	if err != nil {
		t.Fatalf("ParseDPoPHeader: %v", err)
	}
	if parsed.JTI != "test-jti-1" {
		t.Errorf("expected jti 'test-jti-1', got %q", parsed.JTI)
	}
	if parsed.HTTPMethod != "POST" {
		t.Errorf("expected htm POST, got %q", parsed.HTTPMethod)
	}
	if parsed.PublicKey == nil {
		t.Error("expected non-nil public key")
	}
}

func TestDPoP_HTMismatch(t *testing.T) {
	key := generateDPoPKey(t)
	proof := makeDPoPProof(t, key, "test-jti-2", "POST", "https://example.com/token", time.Now())

	_, err := ParseDPoPHeader(proof, "GET", "https://example.com/token")
	if err == nil {
		t.Fatal("expected error for htm mismatch")
	}
	if !strings.Contains(err.Error(), "htm_mismatch") {
		t.Errorf("expected htm_mismatch, got %v", err)
	}
}

func TestDPoP_HTUMismatch(t *testing.T) {
	key := generateDPoPKey(t)
	proof := makeDPoPProof(t, key, "test-jti-3", "POST", "https://example.com/token", time.Now())

	_, err := ParseDPoPHeader(proof, "POST", "https://other.com/token")
	if err == nil {
		t.Fatal("expected error for htu mismatch")
	}
	if !strings.Contains(err.Error(), "htu_mismatch") {
		t.Errorf("expected htu_mismatch, got %v", err)
	}
}

func TestDPoP_ExpiredProof(t *testing.T) {
	key := generateDPoPKey(t)
	proof := makeDPoPProof(t, key, "test-jti-4", "POST", "https://example.com/token", time.Now().Add(-10*time.Minute))

	_, err := ParseDPoPHeader(proof, "POST", "https://example.com/token")
	if err == nil {
		t.Fatal("expected error for expired proof")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expired error, got %v", err)
	}
}

func TestDPoP_IsDPoPTokenRequest(t *testing.T) {
	r := httptest.NewRequest("POST", "/token", nil)
	if IsDPoPTokenRequest(r) {
		t.Error("expected false without DPoP header")
	}
	r.Header.Set("DPoP", "some-proof")
	if !IsDPoPTokenRequest(r) {
		t.Error("expected true with DPoP header")
	}
}

func TestDPoP_MissingJTI(t *testing.T) {
	key := generateDPoPKey(t)
	claims := jwt.MapClaims{
		"htm": "POST",
		"htu": "https://example.com/token",
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	pub := key.Public().(*ecdsa.PublicKey)
	token.Header["jwk"] = map[string]interface{}{
		"kty": "EC", "crv": "P-256",
		"x": base64RawURL(pub.X.Bytes()), "y": base64RawURL(pub.Y.Bytes()),
	}
	signed, _ := token.SignedString(key)

	_, err := ParseDPoPHeader(signed, "POST", "https://example.com/token")
	if err == nil {
		t.Fatal("expected error for missing jti")
	}
	if !strings.Contains(err.Error(), "jti") {
		t.Errorf("expected jti error, got %v", err)
	}
}

func TestDPoP_MissingJWK(t *testing.T) {
	key := generateDPoPKey(t)
	claims := jwt.MapClaims{
		"jti": "test-jti",
		"htm": "POST",
		"htu": "https://example.com/token",
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	signed, _ := token.SignedString(key)

	_, err := ParseDPoPHeader(signed, "POST", "https://example.com/token")
	if err == nil {
		t.Fatal("expected error for missing jwk")
	}
	if !strings.Contains(err.Error(), "invalid_key") {
		t.Errorf("expected invalid_key error, got %v", err)
	}
}

func TestDPoP_RejectHMACAlgorithm(t *testing.T) {
	key := generateDPoPKey(t)
	claims := jwt.MapClaims{
		"jti": "test-jti-hmac",
		"htm": "POST",
		"htu": "https://example.com/token",
		"iat": time.Now().Unix(),
	}
	// Use HS256 — should be rejected.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = "dpop+jwt"
	pub := key.Public().(*ecdsa.PublicKey)
	token.Header["jwk"] = map[string]interface{}{
		"kty": "EC", "crv": "P-256",
		"x": base64RawURL(pub.X.Bytes()), "y": base64RawURL(pub.Y.Bytes()),
	}
	signed, _ := token.SignedString([]byte("secret"))

	_, err := ParseDPoPHeader(signed, "POST", "https://example.com/token")
	if err == nil {
		t.Fatal("expected error for HMAC algorithm")
	}
}

func TestDPoP_TokenType(t *testing.T) {
	if DPoPTokenType != "DPoP" {
		t.Errorf("expected DPoP, got %s", DPoPTokenType)
	}
}

func TestDPoP_ValidateDPoPForToken_NoHeader(t *testing.T) {
	r := httptest.NewRequest("POST", "/token", nil)
	_, err := ValidateDPoPForToken(r, "")
	if err == nil {
		t.Fatal("expected error without DPoP header")
	}
}

func TestDPoP_ValidateDPoPForToken_WithAccessToken(t *testing.T) {
	key := generateDPoPKey(t)
	r := httptest.NewRequest("POST", "https://example.com/token", nil)
	htu := "https://example.com/token"
	ath := hashAccessToken("my-access-token")

	claims := jwt.MapClaims{
		"jti": "test-jti-ath",
		"htm": "POST",
		"htu": htu,
		"iat": time.Now().Unix(),
		"ath": ath,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	pub := key.Public().(*ecdsa.PublicKey)
	token.Header["jwk"] = map[string]interface{}{
		"kty": "EC", "crv": "P-256",
		"x": base64RawURL(pub.X.Bytes()), "y": base64RawURL(pub.Y.Bytes()),
	}
	signed, _ := token.SignedString(key)
	r.Header.Set("DPoP", signed)

	proof, err := ValidateDPoPForToken(r, "my-access-token")
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if proof.AccessToken != ath {
		t.Errorf("expected ath %q, got %q", ath, proof.AccessToken)
	}
}

func TestDPoP_ValidateDPoPForToken_WrongATH(t *testing.T) {
	key := generateDPoPKey(t)
	r := httptest.NewRequest("POST", "https://example.com/token", nil)

	claims := jwt.MapClaims{
		"jti": "test-jti-wath",
		"htm": "POST",
		"htu": "https://example.com/token",
		"iat": time.Now().Unix(),
		"ath": hashAccessToken("correct-token"),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	pub := key.Public().(*ecdsa.PublicKey)
	token.Header["jwk"] = map[string]interface{}{
		"kty": "EC", "crv": "P-256",
		"x": base64RawURL(pub.X.Bytes()), "y": base64RawURL(pub.Y.Bytes()),
	}
	signed, _ := token.SignedString(key)
	r.Header.Set("DPoP", signed)

	_, err := ValidateDPoPForToken(r, "wrong-token")
	if err == nil {
		t.Fatal("expected ath_mismatch error")
	}
	if !strings.Contains(err.Error(), "ath_mismatch") {
		t.Errorf("expected ath_mismatch, got %v", err)
	}
}

// Suppress unused import guard.
var _ = http.MethodPost
