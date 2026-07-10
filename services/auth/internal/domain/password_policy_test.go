package domain

import (
	"testing"
)

func TestPasswordPolicy_Validate_AllRules(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:      12,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}

	tests := []struct {
		name    string
		pw      string
		wantErr error
	}{
		{"valid", "Abcd1234!@#$", nil},
		{"too short", "Aa1!", ErrPolicyTooShort},
		{"no upper", "abcd1234!@#$", ErrPolicyNoUpper},
		{"no lower", "ABCD1234!@#$", ErrPolicyNoLower},
		{"no digit", "AbcdAbcd!@#$", ErrPolicyNoDigit},
		{"no special", "Abcd1234Abcd", ErrPolicyNoSpecial},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := policy.Validate(tc.pw)
			if err != tc.wantErr {
				t.Errorf("Validate(%q) = %v, want %v", tc.pw, err, tc.wantErr)
			}
		})
	}
}

func TestPasswordPolicy_Validate_MinLength(t *testing.T) {
	policy := PasswordPolicy{MinLength: 8}
	if err := policy.Validate("short"); err != ErrPolicyTooShort {
		t.Errorf("expected ErrPolicyTooShort, got %v", err)
	}
	if err := policy.Validate("longenough"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestPasswordPolicy_Validate_Blacklist(t *testing.T) {
	policy := PasswordPolicy{
		MinLength: 4,
		Blacklist: []string{"password", "123456", "Password1"},
	}

	if err := policy.Validate("password"); err != ErrPolicyBlacklisted {
		t.Errorf("expected ErrPolicyBlacklisted for 'password', got %v", err)
	}
	if err := policy.Validate("PASSWORD"); err != ErrPolicyBlacklisted {
		t.Errorf("expected ErrPolicyBlacklisted for 'PASSWORD' (case-insensitive), got %v", err)
	}
	if err := policy.Validate("Password1"); err != ErrPolicyBlacklisted {
		t.Errorf("expected ErrPolicyBlacklisted for 'Password1', got %v", err)
	}
	if err := policy.Validate("goodpass"); err != nil {
		t.Errorf("expected nil for 'goodpass', got %v", err)
	}
}

func TestPasswordPolicy_Validate_EmptyPolicy(t *testing.T) {
	var policy PasswordPolicy // zero-value policy
	// With MinLength=0, any password passes length check.
	if err := policy.Validate(""); err != nil {
		t.Errorf("expected nil for empty policy, got %v", err)
	}
	if err := policy.Validate("anything"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestPasswordPolicy_StrengthScore(t *testing.T) {
	policy := PasswordPolicy{}

	tests := []struct {
		pw   string
		want int
	}{
		{"", 0},
		{"a", 0},
		{"abcdefgh", 1},          // 8+ chars, lowercase only (variety=1)
		{"abcdefghij12", 2},      // 12+ chars, lowercase+digits (variety=2)
		{"Abcdefgh1", 2},         // 8+ chars, upper+lower+digit (variety=3)
		{"Abcdefgh1!", 3},        // 8+ chars, all 4 types (variety=4)
		{"Ab1!2345678ab", 4},     // 12+ chars, all 4 types
	}

	for _, tc := range tests {
		t.Run(tc.pw, func(t *testing.T) {
			got := policy.StrengthScore(tc.pw)
			if got != tc.want {
				t.Errorf("StrengthScore(%q) = %d, want %d", tc.pw, got, tc.want)
			}
		})
	}
}
