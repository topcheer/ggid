package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestValidatePKCE_LengthAndCharSet(t *testing.T) {
	s256Challenge := func(verifier string) string {
		h := sha256.Sum256([]byte(verifier))
		return base64.RawURLEncoding.EncodeToString(h[:])
	}

	valid := strings.Repeat("a", 43) // exactly 43 chars, minimum allowed

	cases := []struct {
		name     string
		verifier string
		method   string
		chal     string
		want     bool
	}{
		// RFC 7636 §4.1: 43*128unreserved
		{"valid 43 chars S256", valid, "S256", s256Challenge(valid), true},
		{"valid 128 chars S256", strings.Repeat("a", 128), "S256", s256Challenge(strings.Repeat("a", 128)), true},
		{"too short (42 chars)", strings.Repeat("a", 42), "S256", s256Challenge(strings.Repeat("a", 42)), false},
		{"too long (129 chars)", strings.Repeat("a", 129), "S256", s256Challenge(strings.Repeat("a", 129)), false},
		{"empty verifier", "", "S256", s256Challenge(valid), false},
		{"invalid char (slash)", strings.Repeat("a", 42) + "/", "S256", "somechallenge", false},
		{"invalid char (equals)", strings.Repeat("a", 42) + "=", "plain", "somechallenge", false},
		{"tilde allowed", strings.Repeat("~", 43), "plain", strings.Repeat("~", 43), true},
		{"correct S256 match", valid, "S256", s256Challenge(valid), true},
		{"wrong S256 challenge", valid, "S256", "wrongchallenge", false},
		{"no PKCE required", valid, "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := &AuthorizationCode{
				CodeChallenge:       tc.chal,
				CodeChallengeMethod: tc.method,
			}
			// For "no PKCE required", leave CodeChallenge empty.
			if tc.name == "no PKCE required" {
				code.CodeChallenge = ""
			}
			got := code.ValidatePKCE(tc.verifier)
			if got != tc.want {
				t.Errorf("ValidatePKCE(%q) = %v, want %v", tc.verifier, got, tc.want)
			}
		})
	}
}
