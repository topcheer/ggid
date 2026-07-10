package service

import (
	"testing"

	"github.com/ggid/ggid/services/auth/internal/conf"
)

func TestPasswordPolicy_Validate_MinLength(t *testing.T) {
	svc := NewPasswordService(conf.PasswordPolicy{MinLength: 8}, nil, nil)
	if err := svc.Validate("short"); err != ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
	if err := svc.Validate("longenough"); err != nil {
		t.Errorf("expected nil for valid length, got %v", err)
	}
}

func TestPasswordPolicy_Validate_RequireUpper(t *testing.T) {
	svc := NewPasswordService(conf.PasswordPolicy{MinLength: 4, RequireUpper: true}, nil, nil)
	if err := svc.Validate("abcd"); err == nil {
		t.Error("expected error for missing uppercase")
	}
	if err := svc.Validate("Abcd"); err != nil {
		t.Errorf("expected nil for valid password, got %v", err)
	}
}

func TestPasswordPolicy_Validate_RequireLower(t *testing.T) {
	svc := NewPasswordService(conf.PasswordPolicy{MinLength: 4, RequireLower: true}, nil, nil)
	if err := svc.Validate("ABCD"); err == nil {
		t.Error("expected error for missing lowercase")
	}
	if err := svc.Validate("ABcD"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestPasswordPolicy_Validate_RequireDigit(t *testing.T) {
	svc := NewPasswordService(conf.PasswordPolicy{MinLength: 4, RequireDigit: true}, nil, nil)
	if err := svc.Validate("abcd"); err == nil {
		t.Error("expected error for missing digit")
	}
	if err := svc.Validate("ab1d"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestPasswordPolicy_Validate_RequireSpecial(t *testing.T) {
	svc := NewPasswordService(conf.PasswordPolicy{MinLength: 4, RequireSpecial: true}, nil, nil)
	if err := svc.Validate("abcd1234"); err == nil {
		t.Error("expected error for missing special char")
	}
	if err := svc.Validate("abcd!234"); err != nil {
		t.Errorf("expected nil for password with special char, got %v", err)
	}
}

func TestPasswordPolicy_Validate_AllRequirements(t *testing.T) {
	svc := NewPasswordService(conf.PasswordPolicy{
		MinLength:      12,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
	}, nil, nil)

	// Missing special
	if err := svc.Validate("Abcd1234abcd"); err == nil {
		t.Error("expected error for missing special")
	}
	// Meets all requirements
	if err := svc.Validate("Abcd1234!@#$"); err != nil {
		t.Errorf("expected nil for compliant password, got %v", err)
	}
	// Too short despite meeting all char types
	if err := svc.Validate("Aa1!"); err == nil {
		t.Error("expected error for too-short password")
	}
}
