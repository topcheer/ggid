package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
)

func TestAuthService_GetPasswordPolicy_NilCov(t *testing.T) {
	svc := &AuthService{}
	policy := svc.GetPasswordPolicy()
	if policy.MinLength != 0 {
		t.Errorf("expected zero policy, got %+v", policy)
	}
}

func TestAuthService_SetPasswordPolicy_Cov(t *testing.T) {
	svc := &AuthService{cfg: &conf.Config{}}
	policy := conf.PasswordPolicy{MinLength: 16, RequireUpper: true}
	svc.SetPasswordPolicy(policy)
	if svc.cfg.Password.MinLength != 16 {
		t.Errorf("expected 16, got %d", svc.cfg.Password.MinLength)
	}
}

func TestPasswordService_CheckPasswordBreach_NetworkError(t *testing.T) {
	ps := &PasswordService{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ps.CheckPasswordBreach(ctx, "testpassword")
	if err != nil {
		t.Errorf("expected nil (fail open), got %v", err)
	}
}

func TestObfuscateForLog(t *testing.T) {
	result := obfuscateForLog("user@example.com")
	if result == "user@example.com" {
		t.Error("expected email to be obfuscated")
	}
}

func TestObfuscateEmail(t *testing.T) {
	result := obfuscateEmail("user@example.com")
	if result == "user@example.com" {
		t.Error("expected email to be masked")
	}
}

func TestTokenSet_Struct(t *testing.T) {
	ts := &domain.TokenSet{
		AccessToken:  "abc",
		RefreshToken: "def",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}
	if ts.AccessToken != "abc" {
		t.Error("expected abc")
	}
	if ts.MFARequired {
		t.Error("expected MFARequired false")
	}
}

func TestPasswordService_GetPolicy_Cov(t *testing.T) {
	ps := &PasswordService{policy: conf.PasswordPolicy{MinLength: 12}}
	p := ps.GetPolicy()
	if p.MinLength != 12 {
		t.Errorf("expected 12, got %d", p.MinLength)
	}
}

func TestPasswordService_UpdatePolicy_Cov(t *testing.T) {
	ps := &PasswordService{}
	ps.UpdatePolicy(conf.PasswordPolicy{MinLength: 20})
	if ps.policy.MinLength != 20 {
		t.Errorf("expected 20, got %d", ps.policy.MinLength)
	}
}
