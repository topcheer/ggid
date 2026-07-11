package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
)

// TestPasswordPepper_HashAndVerify proves that pepper set via crypto.SetPepper
// is transparently applied through HashPassword and VerifyPassword — which is
// exactly what the auth service Register and Login flows use.
func TestPasswordPepper_HashAndVerify(t *testing.T) {
	crypto.EnableTestFastHash()
	defer crypto.SetPepper("") // reset pepper after test

	password := "MySecurePass123!"

	// Without pepper
	crypto.SetPepper("")
	hashNoPepper, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("hash without pepper: %v", err)
	}
	ok, err := crypto.VerifyPassword(password, hashNoPepper)
	if err != nil || !ok {
		t.Fatalf("verify without pepper should succeed: ok=%v err=%v", ok, err)
	}

	// With pepper
	crypto.SetPepper("secret-pepper-value")
	hashWithPepper, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("hash with pepper: %v", err)
	}
	ok, err = crypto.VerifyPassword(password, hashWithPepper)
	if err != nil || !ok {
		t.Fatalf("verify with pepper should succeed: ok=%v err=%v", ok, err)
	}

	// Hashes should differ (pepper changes the HMAC'd input)
	if hashNoPepper == hashWithPepper {
		t.Error("hashes with and without pepper should differ")
	}

	// Verify with wrong pepper should fail
	crypto.SetPepper("wrong-pepper")
	ok, err = crypto.VerifyPassword(password, hashWithPepper)
	if ok {
		t.Error("verify with wrong pepper should fail")
	}
}

// TestPasswordPepper_BackwardCompatible proves that empty pepper = no-op,
// so existing deployments without PASSWORD_PEPPER env continue to work.
func TestPasswordPepper_BackwardCompatible(t *testing.T) {
	crypto.EnableTestFastHash()
	crypto.SetPepper("") // empty = no-op

	password := "BackwardCompatible123!"
	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	ok, err := crypto.VerifyPassword(password, hash)
	if err != nil || !ok {
		t.Errorf("verify should succeed without pepper: ok=%v err=%v", ok, err)
	}

	// Wrong password should fail
	ok, _ = crypto.VerifyPassword("wrong-password", hash)
	if ok {
		t.Error("verify with wrong password should fail")
	}
}

// TestPasswordPepper_RegisterUsesPepper proves the auth service Register path
// transparently applies pepper via crypto.HashPassword.
func TestPasswordPepper_RegisterUsesPepper(t *testing.T) {
	crypto.EnableTestFastHash()

	svc, cr, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	tenantID := uuid.New()
	userID := uuid.New()

	// Register without pepper
	crypto.SetPepper("")
	err := svc.Register(ctx, tenantID, userID, "pepper-test-user", "StrongPass!1")
	if err != nil {
		t.Fatalf("register without pepper: %v", err)
	}

	cred := cr.byName["pepper-test-user"]
	if cred == nil {
		t.Fatal("credential should exist after register")
	}
	hashNoPepper := cred.Secret

	// Now register a different user WITH pepper
	crypto.SetPepper("register-pepper")
	err = svc.Register(ctx, tenantID, uuid.New(), "pepper-test-user2", "StrongPass!1")
	if err != nil {
		t.Fatalf("register with pepper: %v", err)
	}

	cred2 := cr.byName["pepper-test-user2"]
	if cred2 == nil {
		t.Fatal("second credential should exist")
	}
	hashWithPepper := cred2.Secret

	// The two hashes should be different even though password is identical
	if hashNoPepper == hashWithPepper {
		t.Error("hashes should differ when pepper is applied vs not")
	}

	// Reset pepper
	crypto.SetPepper("")
}
