package authprovider

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TestTransparentRehash verifies that when a legacy bcrypt hash is verified
// successfully, the rehash callback is invoked asynchronously.
func TestTransparentRehash(t *testing.T) {
	pw := "testpw-rehash-1"
	bcryptHash, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)

	var rehashCalled int32
	var capturedUserID atomic.Value // stores uuid.UUID

	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       uuid.New(),
			Username:     "rehash-test-user",
			Status:       "active",
			PasswordHash: string(bcryptHash),
		},
	}

	provider := NewLocalProvider(store)
	provider.SetRehashCallback(func(ctx context.Context, userID uuid.UUID, plainPw, oldHash string) {
		atomic.StoreInt32(&rehashCalled, 1)
		capturedUserID.Store(userID)
		// Generate new Argon2id hash (simulating DB update).
		_, _ = crypto.HashPassword(plainPw)
	})

	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "rehash-test-user",
		Password: pw,
	})
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	// The rehash callback runs asynchronously; wait for it.
	for i := 0; i < 200; i++ {
		if atomic.LoadInt32(&rehashCalled) == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	if atomic.LoadInt32(&rehashCalled) != 1 {
		t.Error("rehash callback was not invoked for legacy bcrypt hash")
	}
	if capturedUserID.Load().(uuid.UUID) != store.credential.UserID {
		t.Error("rehash callback received wrong user ID")
	}
}

// TestCorruptedHashReturnsErrInternal verifies that a corrupted/unrecognized
// hash format returns ErrInternal (not ErrUnauthenticated).
func TestCorruptedHashReturnsErrInternal(t *testing.T) {
	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       uuid.New(),
			Username:     "corrupt-user",
			Status:       "active",
			PasswordHash: "totally-corrupted-hash-string",
		},
	}

	provider := NewLocalProvider(store)
	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "corrupt-user",
		Password: "somepw",
	})
	if err == nil {
		t.Fatal("expected error for corrupted hash")
	}

	ge, ok := gerr.AsGGIDError(err)
	if !ok {
		t.Fatalf("expected GGIDError, got %T: %v", err, err)
	}
	if ge.Code != gerr.ErrInternal {
		t.Errorf("expected ErrInternal for corrupted hash, got %s", ge.Code)
	}
}

// TestArgon2idNoRehash verifies that Argon2id hashes do NOT trigger rehash.
func TestArgon2idNoRehash(t *testing.T) {
	pw := "testpw-argon-norehash"
	argonHash, _ := crypto.HashPassword(pw)

	var rehashCalled int32
	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       uuid.New(),
			Username:     "argon-user",
			Status:       "active",
			PasswordHash: argonHash,
		},
	}

	provider := NewLocalProvider(store)
	provider.SetRehashCallback(func(ctx context.Context, userID uuid.UUID, plainPw, oldHash string) {
		atomic.StoreInt32(&rehashCalled, 1)
	})

	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "argon-user",
		Password: pw,
	})
	if err != nil {
		t.Fatalf("authentication failed: %v", err)
	}

	// Give goroutine time to run (it shouldn't).
	for i := 0; i < 50; i++ {
		if atomic.LoadInt32(&rehashCalled) != 0 {
			break
		}
	}

	if atomic.LoadInt32(&rehashCalled) != 0 {
		t.Error("rehash callback should NOT be invoked for Argon2id hash")
	}
}
