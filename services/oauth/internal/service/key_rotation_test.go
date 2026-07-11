package service

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

func TestRotatingKeyProvider_Initial(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 24*time.Hour)

	if kp.KeyID() == "" {
		t.Error("expected non-empty KeyID")
	}
	if kp.PublicKey() == nil {
		t.Error("expected non-nil PublicKey")
	}
	if kp.PrivateKey() != key {
		t.Error("expected same private key pointer")
	}
	if kp.PreviousPublicKey() != nil {
		t.Error("expected nil previous key before rotation")
	}
}

func TestRotatingKeyProvider_RotateKey(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 24*time.Hour)

	oldKID := kp.KeyID()
	oldPub := kp.PublicKey()

	if err := kp.RotateKey(); err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	if kp.KeyID() == oldKID {
		t.Error("expected new KeyID after rotation")
	}
	if kp.PublicKey() == oldPub {
		t.Error("expected new PublicKey after rotation")
	}
	if kp.PreviousPublicKey() == nil {
		t.Error("expected non-nil previous key after rotation")
	}
	if kp.PreviousKeyID() != oldKID {
		t.Error("expected previous key ID to match old key")
	}
}

func TestRotatingKeyProvider_ResolveKeyByID(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 24*time.Hour)

	oldKID := kp.KeyID()

	if kp.ResolveKeyByID(oldKID) == nil {
		t.Error("expected to resolve current key by ID")
	}
	if kp.ResolveKeyByID("unknown-kid") != nil {
		t.Error("expected nil for unknown key ID")
	}

	kp.RotateKey()

	if kp.ResolveKeyByID(oldKID) == nil {
		t.Error("expected to resolve previous key by ID during grace period")
	}
	if kp.ResolveKeyByID(kp.KeyID()) == nil {
		t.Error("expected to resolve new current key by ID")
	}
}

func TestRotatingKeyProvider_GracePeriodExpiry(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 1*time.Millisecond)

	kp.RotateKey()
	time.Sleep(10 * time.Millisecond)

	if !kp.IsGracePeriodExpired() {
		t.Error("expected grace period to be expired")
	}
	if kp.PreviousPublicKey() != nil {
		t.Error("expected nil previous key after grace period")
	}
	kp.CleanupExpired()
	if kp.PreviousKeyID() != "" {
		t.Error("expected empty previous key ID after cleanup")
	}
}

func TestRotatingKeyProvider_DefaultGracePeriod(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 0)
	kp.RotateKey()
	if kp.PreviousPublicKey() == nil {
		t.Error("expected non-nil previous key with default grace period")
	}
}

func TestStartRotationTicker(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 1*time.Hour)

	oldKID := kp.KeyID()
	stop := kp.StartRotationTicker(50 * time.Millisecond)
	time.Sleep(120 * time.Millisecond)
	stop()

	if kp.KeyID() == oldKID {
		t.Error("expected key to rotate via ticker")
	}
}
