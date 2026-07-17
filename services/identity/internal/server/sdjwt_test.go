package server

import (
	"testing"
)

func TestSimpleHash_Deterministic(t *testing.T) {
	h1 := simpleHash("test-value")
	h2 := simpleHash("test-value")
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
	if simpleHash("different") == h1 {
		t.Error("different input should produce different hash")
	}
}

func TestBase64RoundTrip(t *testing.T) {
	original := `{"alg":"none","typ":"sd-jwt"}`
	encoded := base64Encode(original)
	decoded, err := base64Decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded != original {
		t.Errorf("round-trip mismatch: %s != %s", decoded, original)
	}
}

func TestSDJWTIssueResponse_Structure(t *testing.T) {
	resp := SDJWTIssueResponse{
		SJWT: "header.payload.",
		Disclosures: []Disclosure{
			{Claim: "email", Hash: "abc123", Value: "alice@example.com"},
		},
	}
	if resp.SJWT == "" {
		t.Error("SJWT should be set")
	}
	if len(resp.Disclosures) != 1 {
		t.Error("should have 1 disclosure")
	}
}

func TestSDJWTVerifyResponse_Invalid(t *testing.T) {
	resp := SDJWTVerifyResponse{Valid: false, Error: "expired"}
	if resp.Valid {
		t.Error("should be invalid")
	}
}
