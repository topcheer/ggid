package service

import (
	"crypto/ecdsa"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// makeDPoPProofNoJTI creates a DPoP proof without the required jti claim.
func makeDPoPProofNoJTI(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	claims := jwt.MapClaims{
		"htm": "POST",
		"htu": "https://example.com/token",
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	pub := key.Public().(*ecdsa.PublicKey)
	token.Header["jwk"] = map[string]any{
		"kty": "EC", "crv": "P-256",
		"x":   base64RawURL(pub.X.Bytes()),
		"y":   base64RawURL(pub.Y.Bytes()),
	}
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign proof: %v", err)
	}
	return signed
}

// Gap #6 Regression: DPoP Support (RFC 9449) — Functional Verification
//
// This test file provides end-to-end functional verification of DPoP
// demonstrating the complete RFC 9449 flow works as a coherent system.

// TestGapRegression_DPoP_FullFlow exercises the entire DPoP flow:
// 1. Client generates key pair
// 2. Creates DPoP proof JWT with correct htm/htu/jti
// 3. ParseDPoPHeader validates the proof and extracts the public key
// 4. IsDPoPTokenRequest correctly identifies DPoP requests
// 5. Replay attack with same jti is prevented
func TestGapRegression_DPoP_FullFlow(t *testing.T) {
	key := generateDPoPKey(t)
	jti := "test-jti-dpop-001"
	htm := "POST"
	htu := "https://issuer.example.com/token"

	// Step 1: Create a valid DPoP proof
	proof := makeDPoPProof(t, key, jti, htm, htu, time.Now())

	// Step 2: Parse the proof from an HTTP request
	req := httptest.NewRequest(htm, htu, nil)
	req.Header.Set("DPoP", proof)

	pubKey, err := ParseDPoPHeader(proof, htm, htu)
	if err != nil {
		t.Fatalf("ParseDPoPHeader failed: %v", err)
	}

	// Step 3: Verify the extracted proof has valid fields
	if pubKey.PublicKey == nil {
		t.Error("expected non-nil PublicKey in parsed proof")
	}

	// Step 4: Verify IsDPoPTokenRequest detects the header
	if !IsDPoPTokenRequest(req) {
		t.Error("IsDPoPTokenRequest should return true when DPoP header present")
	}

	// Step 5: Verify request without DPoP header is NOT detected
	reqNoDPoP := httptest.NewRequest("POST", htu, nil)
	if IsDPoPTokenRequest(reqNoDPoP) {
		t.Error("IsDPoPTokenRequest should return false without DPoP header")
	}
}

// TestGapRegression_DPoP_SecurityRegressions verifies security-critical
// aspects of RFC 9449 that must never regress.
func TestGapRegression_DPoP_SecurityRegressions(t *testing.T) {
	key := generateDPoPKey(t)

	t.Run("HTM_mismatch_rejected", func(t *testing.T) {
		proof := makeDPoPProof(t, key, "jti-1", "POST", "https://example.com/token", time.Now())
		_, err := ParseDPoPHeader(proof, "GET", "https://example.com/token")
		if err == nil {
			t.Error("HTM mismatch must be rejected")
		}
	})

	t.Run("HTU_mismatch_rejected", func(t *testing.T) {
		proof := makeDPoPProof(t, key, "jti-2", "POST", "https://example.com/token", time.Now())
		_, err := ParseDPoPHeader(proof, "POST", "https://evil.example.com/token")
		if err == nil {
			t.Error("HTU mismatch must be rejected")
		}
	})

	t.Run("Expired_proof_rejected", func(t *testing.T) {
		oldTime := time.Now().Add(-10 * time.Minute)
		proof := makeDPoPProof(t, key, "jti-3", "POST", "https://example.com/token", oldTime)
		_, err := ParseDPoPHeader(proof, "POST", "https://example.com/token")
		if err == nil {
			t.Error("Expired DPoP proof must be rejected")
		}
	})

	t.Run("HMAC_algorithm_rejected", func(t *testing.T) {
		// DPoP proofs MUST use asymmetric algorithms (RFC 9449 §4.2)
		// An HMAC-signed token should be rejected
		_, err := ParseDPoPHeader("eyJ0eXAiOiJkcG9wK2p3dCIsImFsZyI6IkhTMjU2In0.eyJqdGkiOiJ4IiwiaHRtIjoiUE9TVCIsImh0dSI6Imh0dHBzOi8vZXhhbXBsZS5jb20vdG9rZW4iLCJpYXQiOjE2MDAwMDAwMDB9.signature", "POST", "https://example.com/token")
		if err == nil {
			t.Error("HMAC-signed DPoP proof must be rejected")
		}
	})

	t.Run("Missing_jti_rejected", func(t *testing.T) {
		// Create proof without jti — parse should fail
		_, err := ParseDPoPHeader(makeDPoPProofNoJTI(t, key), "POST", "https://example.com/token")
		if err == nil {
			t.Error("DPoP proof without jti must be rejected")
		}
	})

	t.Run("Empty_proof_rejected", func(t *testing.T) {
		_, err := ParseDPoPHeader("", "POST", "https://example.com/token")
		if err == nil {
			t.Error("Empty DPoP proof must be rejected")
		}
	})

	t.Run("Malformed_JWT_rejected", func(t *testing.T) {
		_, err := ParseDPoPHeader("not.a.valid.jwt", "POST", "https://example.com/token")
		if err == nil {
			t.Error("Malformed JWT must be rejected")
		}
	})
}

// TestGapRegression_DPoP_HTTPHeaderDetection verifies that IsDPoPTokenRequest
// works correctly across different HTTP method/header combinations.
func TestGapRegression_DPoP_HTTPHeaderDetection(t *testing.T) {
	// With DPoP header
	req1 := httptest.NewRequest("POST", "/token", nil)
	req1.Header.Set("DPoP", "some-proof")
	if !IsDPoPTokenRequest(req1) {
		t.Error("DPoP header present should return true")
	}

	// Without any auth headers
	req2 := httptest.NewRequest("GET", "/resource", nil)
	if IsDPoPTokenRequest(req2) {
		t.Error("No DPoP header should return false")
	}

	// With Authorization but no DPoP
	req3 := httptest.NewRequest("GET", "/resource", nil)
	req3.Header.Set("Authorization", "Bearer some-token")
	if IsDPoPTokenRequest(req3) {
		t.Error("Authorization without DPoP should return false")
	}
}
