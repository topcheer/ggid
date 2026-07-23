package service

import (
	"crypto/rand"
	"crypto/rsa"
	"sync"
	"testing"
	"time"
)

func TestRotatingKeyProvider_Initial(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 24*time.Hour)

	if kp.Metadata().KeyID == "" {
		t.Error("expected non-empty KeyID")
	}
	if kp.Public() == nil {
		t.Error("expected non-nil PublicKey")
	}
	if kp.Signer() != key {
		t.Error("expected same private key pointer")
	}
	if kp.PreviousPublicKey() != nil {
		t.Error("expected nil previous key before rotation")
	}
}

func TestRotatingKeyProvider_RotateKey(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 24*time.Hour)

	oldKID := kp.Metadata().KeyID
	oldPub := kp.Public()

	if err := kp.RotateKey(); err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}

	if kp.Metadata().KeyID == oldKID {
		t.Error("expected new KeyID after rotation")
	}
	if kp.Public() == oldPub {
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

	oldKID := kp.Metadata().KeyID

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
	if kp.ResolveKeyByID(kp.Metadata().KeyID) == nil {
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

	oldKID := kp.Metadata().KeyID
	stop := kp.StartRotationTicker(50 * time.Millisecond)
	time.Sleep(120 * time.Millisecond)
	stop()

	if kp.Metadata().KeyID == oldKID {
		t.Error("expected key to rotate via ticker")
	}
}

func TestKeyAge(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 1*time.Hour)

	// Key just initialized: age should be small
	age1 := kp.KeyAge()
	if age1 < 0 || age1 > 2*time.Second {
		t.Errorf("expected small age at init, got %v", age1)
	}

	// Rotate and check age resets to near-zero
	_ = kp.RotateKey()
	age2 := kp.KeyAge()
	if age2 < 0 || age2 > 2*time.Second {
		t.Errorf("expected small age after rotation, got %v", age2)
	}
}

func TestRotatedAt(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 1*time.Hour)

	// rotatedAt is set at init time now
	if kp.RotatedAt().IsZero() {
		t.Error("expected non-zero RotatedAt at init")
	}
	_ = kp.RotateKey()
	if kp.RotatedAt().IsZero() {
		t.Error("expected non-zero RotatedAt after rotation")
	}
}

func TestStartAutoRotation_RotatesWhenOld(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 1*time.Hour)

	oldKID := kp.KeyID()

	var mu sync.Mutex
	var auditCalls []struct{ old, new string }

	// maxAge = 5ms, checkInterval = 10ms — should rotate quickly
	// (5ms gives enough headroom for slow CI runners to exceed maxAge)
	rotated := make(chan struct{}, 4)
	stop := kp.StartAutoRotation(10*time.Millisecond, 5*time.Millisecond, func(old, new string) {
		mu.Lock()
		auditCalls = append(auditCalls, struct{ old, new string }{old, new})
		mu.Unlock()
		rotated <- struct{}{}
	})

	// Wait for at least one rotation (up to 2s for slow CI)
	select {
	case <-rotated:
	case <-time.After(2 * time.Second):
		stop()
		t.Fatal("expected at least one rotation within 2s")
	}
	stop()

	newKID := kp.KeyID()
	if newKID == oldKID {
		t.Error("expected key to auto-rotate when age exceeds maxAge")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(auditCalls) == 0 {
		t.Fatal("expected at least one audit callback on rotation")
	}
	// First audit call should have the original old KID
	if auditCalls[0].old != oldKID {
		t.Errorf("first audit old_kid: want %s, got %s", oldKID, auditCalls[0].old)
	}
	// Last audit call should have the current new KID
	if auditCalls[len(auditCalls)-1].new != newKID {
		t.Errorf("last audit new_kid: want %s, got %s", newKID, auditCalls[len(auditCalls)-1].new)
	}
}

func TestStartAutoRotation_SkipsWhenNew(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kp := NewRotatingKeyProvider(key, 1*time.Hour)

	oldKID := kp.KeyID()

	// maxAge = 10s, checkInterval = 50ms — should NOT rotate
	stop := kp.StartAutoRotation(50*time.Millisecond, 10*time.Second, nil)
	time.Sleep(200 * time.Millisecond)
	stop()

	if kp.KeyID() != oldKID {
		t.Error("expected key to NOT rotate when age < maxAge")
	}
}
