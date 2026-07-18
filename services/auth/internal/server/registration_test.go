package server

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	valid := []string{"user@example.com", "test.user@domain.org", "a@b.co"}
	for _, e := range valid {
		if !validateEmail(e) { t.Errorf("%s should be valid", e) }
	}
	invalid := []string{"", "notanemail", "@nodomain.com", "user@"}
	for _, e := range invalid {
		if validateEmail(e) { t.Errorf("%s should be invalid", e) }
	}
}

func TestValidatePassword(t *testing.T) {
	valid := []string{"Password1", "Secure123", "Abcdefg1"}
	for _, p := range valid {
		if !validatePassword(p) { t.Errorf("%s should be valid", p) }
	}
	invalid := []string{"", "short", "alllowercase", "NoDigits!", "12345678", "ONLYUPPER1"}
	for _, p := range invalid {
		if validatePassword(p) { t.Errorf("%s should be invalid", p) }
	}
}

func TestVerificationRepo_NilPool(t *testing.T) {
	repo := newVerificationRepo(nil)
	token, err := repo.CreateToken(nil, "user-1", "email_verification", 0)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if token.Token == "" { t.Error("token should be generated") }
	if token.Type != "email_verification" { t.Error("type mismatch") }
}

func TestVerificationRepo_ValidateNilPool(t *testing.T) {
	repo := newVerificationRepo(nil)
	_, err := repo.ValidateToken(nil, "any-token", "email_verification")
	if err == nil { t.Error("nil pool should return error") }
}

func TestRegisterRequest_Validation(t *testing.T) {
	req := RegisterRequest{Username: "testuser", Email: "test@example.com", Password: "Password1"}
	if req.Username == "" || !validateEmail(req.Email) || !validatePassword(req.Password) {
		t.Error("valid registration request failed validation")
	}
}
