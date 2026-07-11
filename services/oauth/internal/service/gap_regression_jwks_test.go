package service

// JWKS Endpoint Functional Verification Tests
// Verifies: Gap #2 — JWKS endpoint returns valid JSON Web Key Set (was DONE via grep, now functionally verified)
// Flow: GetJWKS → verify keys array with correct fields (kty, use, kid, n, e, alg).
// Date: 2026-07-25

import (
	"encoding/base64"
	"math/big"
	"testing"
)

// ========== JWKS Functional Tests ==========

// TestJWKS_ReturnsValidKeySet verifies GetJWKS returns a properly formatted JWKS response.
func TestJWKS_ReturnsValidKeySet(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	jwks := svc.GetJWKS()
	if jwks == nil {
		t.Fatal("JWKS response should not be nil")
	}

	if len(jwks.Keys) == 0 {
		t.Fatal("JWKS should contain at least one key")
	}

	key := jwks.Keys[0]

	// Verify required fields per RFC 7517
	if key.KTY != "RSA" {
		t.Errorf("kty should be 'RSA', got '%s'", key.KTY)
	}
	if key.Use != "sig" {
		t.Errorf("use should be 'sig', got '%s'", key.Use)
	}
	if key.Alg != "RS256" {
		t.Errorf("alg should be 'RS256', got '%s'", key.Alg)
	}
	if key.KID == "" {
		t.Error("kid should not be empty")
	}
	if key.N == "" {
		t.Error("n (modulus) should not be empty")
	}
	if key.E == "" {
		t.Error("e (exponent) should not be empty")
	}
}

// TestJWKS_ModulusIsValidBase64 verifies the RSA modulus is valid base64url.
func TestJWKS_ModulusIsValidBase64(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	jwks := svc.GetJWKS()
	key := jwks.Keys[0]

	// N should be valid base64url encoding of the RSA public key modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		t.Fatalf("failed to decode n (modulus): %v", err)
	}

	if len(nBytes) < 128 {
		t.Errorf("RSA modulus should be at least 128 bytes (1024-bit), got %d bytes", len(nBytes))
	}

	// Convert to big.Int and verify it's positive
	n := new(big.Int).SetBytes(nBytes)
	if n.Sign() <= 0 {
		t.Error("RSA modulus should be a positive integer")
	}
}

// TestJWKS_ExponentIsValid verifies the RSA exponent is correct.
func TestJWKS_ExponentIsValid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	jwks := svc.GetJWKS()
	key := jwks.Keys[0]

	// E should be "AQAB" (65537 in base64url)
	if key.E != "AQAB" {
		t.Errorf("e (exponent) should be 'AQAB' (65537), got '%s'", key.E)
	}

	// Decode and verify
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		t.Fatalf("failed to decode e: %v", err)
	}

	e := new(big.Int).SetBytes(eBytes)
	if e.Int64() != 65537 {
		t.Errorf("exponent should be 65537, got %d", e.Int64())
	}
}

// TestJWKS_KeyIDMatchesProvider verifies KID matches the key provider.
func TestJWKS_KeyIDMatchesProvider(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	jwks := svc.GetJWKS()
	expectedKID := svc.keyProvider.KeyID()

	if jwks.Keys[0].KID != expectedKID {
		t.Errorf("kid should be '%s', got '%s'", expectedKID, jwks.Keys[0].KID)
	}
}

// TestJWKS_KeyMatchesProviderPublicKey verifies the JWKS modulus matches the
// key provider's actual RSA public key.
func TestJWKS_KeyMatchesProviderPublicKey(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	jwks := svc.GetJWKS()
	key := jwks.Keys[0]

	pub := svc.keyProvider.PublicKey()

	// Compare modulus
	expectedN := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	if key.N != expectedN {
		t.Errorf("modulus mismatch: JWKS has different modulus than key provider's public key")
	}

	// Compare exponent
	expectedE := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	if key.E != expectedE {
		t.Errorf("exponent mismatch: JWKS=%s, expected=%s", key.E, expectedE)
	}
}

// TestJWKS_KeyRotation verifies that rotating the key provider changes the JWKS.
func TestJWKS_KeyRotation(t *testing.T) {
	// Original key
	kp1 := newMockKeyProvider()
	svc1 := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp1, "https://test.ggid.dev")
	jwks1 := svc1.GetJWKS()

	// Rotated key
	kp2 := &mockKeyProvider{priv: kp1.priv, pub: kp1.pub, kid: "rotated-kid-2"}
	svc2 := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp2, "https://test.ggid.dev")
	jwks2 := svc2.GetJWKS()

	if jwks1.Keys[0].KID == jwks2.Keys[0].KID {
		t.Error("KID should change after key rotation")
	}
}

// TestJWKS_DiscoveryURI verifies the discovery config references the JWKS endpoint.
func TestJWKS_DiscoveryURI(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	cfg := svc.GetDiscoveryConfig()

	if cfg.JwksURI == "" {
		t.Fatal("jwks_uri should not be empty in discovery config")
	}

	// JWKS URI should end with /oauth/jwks
	if cfg.JwksURI != "https://test.ggid.dev/oauth/jwks" {
		t.Errorf("jwks_uri should be 'https://test.ggid.dev/oauth/jwks', got '%s'", cfg.JwksURI)
	}
}
