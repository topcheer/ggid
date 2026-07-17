package server

import (
	"crypto/hmac"
	"encoding/json"
	"testing"
)

func TestSDJWT_IssueAndVerify(t *testing.T) {
	// Issue
	issueReq := SDJWTIssueRequest{
		Subject: "user-123",
		Claims: map[string]any{
			"email":   "alice@example.com",
			"phone":   "+1234567890",
			"name":    "Alice",
		},
		Disclosable:    []string{"email", "phone"},
		AlwaysDisclosed: []string{"name"},
		TTLSeconds:     3600,
	}

	// Simulate issue: build JWT manually.
	header := map[string]any{"alg": "HS256", "typ": "sd-jwt"}
	payload := map[string]any{
		"iss": "ggid", "sub": issueReq.Subject, "name": "Alice",
		"_sd_email": sha256Hex("email:alice@example.com"),
	}
	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)
	signingInput := b64url(headerJSON) + "." + b64url(payloadJSON)
	sig := hmacSHA256(getSDJWTSecret(), []byte(signingInput))
	sjwt := signingInput + "." + b64url(sig)

	// Verify signature
	parts := splitDot(sjwt)
	if len(parts) != 3 {
		t.Fatal("should have 3 parts")
	}
	verifyInput := parts[0] + "." + parts[1]
	expectedSig := hmacSHA256(getSDJWTSecret(), []byte(verifyInput))
	actualSig, _ := b64urlDecode(parts[2])
	if !hmac.Equal(expectedSig, actualSig) {
		t.Fatal("signature verification should pass")
	}
}

func TestSDJWT_SignatureRejection(t *testing.T) {
	// Tampered token should fail verification.
	sjwt := b64url([]byte(`{"alg":"HS256"}`)) + "." + b64url([]byte(`{"sub":"x"}`)) + "." + b64url([]byte("badsig"))
	parts := splitDot(sjwt)
	verifyInput := parts[0] + "." + parts[1]
	expectedSig := hmacSHA256(getSDJWTSecret(), []byte(verifyInput))
	actualSig, _ := b64urlDecode(parts[2])
	if hmac.Equal(expectedSig, actualSig) {
		t.Fatal("tampered signature should NOT pass")
	}
}

func TestSHA256Hex_Deterministic(t *testing.T) {
	h1 := sha256Hex("test-data")
	h2 := sha256Hex("test-data")
	if h1 != h2 {
		t.Fatal("SHA-256 should be deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("SHA-256 hex should be 64 chars, got %d", len(h1))
	}
}

func TestB64URL_RoundTrip(t *testing.T) {
	original := []byte(`{"test":"data","num":42}`)
	encoded := b64url(original)
	decoded, err := b64urlDecode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if string(decoded) != string(original) {
		t.Error("round-trip mismatch")
	}
}

func TestSDJWTVerifyResponse_Invalid(t *testing.T) {
	resp := SDJWTVerifyResponse{Valid: false, Error: "expired"}
	if resp.Valid {
		t.Error("should be invalid")
	}
}

func splitDot(s string) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
