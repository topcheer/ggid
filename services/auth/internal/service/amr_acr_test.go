package service

import (
	"testing"
)

func TestComputeAMR_PasswordOnly(t *testing.T) {
	amr := ComputeAMR([]string{"password"})
	if len(amr) != 1 || amr[0] != "pwd" {
		t.Errorf("expected [pwd], got %v", amr)
	}
}

func TestComputeAMR_MFA(t *testing.T) {
	amr := ComputeAMR([]string{"password", "totp"})
	hasMFA := false
	for _, m := range amr {
		if m == "mfa" {
			hasMFA = true
		}
	}
	if !hasMFA {
		t.Error("MFA flag should be present when TOTP used")
	}
}

func TestComputeACR_Levels(t *testing.T) {
	if ComputeACR([]string{"password"}) != AAL1 {
		t.Error("password only should be AAL1")
	}
	if ComputeACR([]string{"password", "totp"}) != AAL2 {
		t.Error("password+totp should be AAL2")
	}
	if ComputeACR([]string{"password", "webauthn"}) != AAL3 {
		t.Error("password+webauthn should be AAL3")
	}
}

func TestComputeACR_NoAuth(t *testing.T) {
	if ComputeACR([]string{}) != "" {
		t.Error("no auth methods should return empty ACR")
	}
}

func TestComputeAMR_WebAuthn(t *testing.T) {
	amr := ComputeAMR([]string{"webauthn"})
	hasFido := false
	for _, m := range amr {
		if m == "fpt" {
			hasFido = true
		}
	}
	if !hasFido {
		t.Error("WebAuthn should produce fpt AMR")
	}
}
