package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignature_Valid(t *testing.T) {
	payload := []byte(`{"event":"user.created","user_id":"123"}`)
	secret := "test-webhook-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !VerifySignature(payload, sig, secret) {
		t.Error("expected signature to be valid")
	}
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	if VerifySignature(payload, "invalid-signature", "secret") {
		t.Error("expected signature to be invalid")
	}
}

func TestVerifySignature_EmptyInputs(t *testing.T) {
	if VerifySignature(nil, "", "secret") {
		t.Error("expected false for empty signature")
	}
	if VerifySignature(nil, "sig", "") {
		t.Error("expected false for empty secret")
	}
}

func TestVerifySignature_WrongSecret(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	sig := SignPayload(payload, "correct-secret")

	if VerifySignature(payload, sig, "wrong-secret") {
		t.Error("expected signature verification to fail with wrong secret")
	}
}

func TestSignPayload_Deterministic(t *testing.T) {
	payload := []byte("test payload")
	sig1 := SignPayload(payload, "secret")
	sig2 := SignPayload(payload, "secret")

	if sig1 != sig2 {
		t.Error("expected same payload+secret to produce same signature")
	}

	if !VerifySignature(payload, sig1, "secret") {
		t.Error("SignPayload output should pass VerifySignature")
	}
}
