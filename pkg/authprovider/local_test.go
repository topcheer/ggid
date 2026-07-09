package authprovider

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/crypto"
	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// --- Mock credential store ---

type mockCredentialStore struct {
	credential *LocalCredential
	err        error
	called     bool
}

func (m *mockCredentialStore) GetCredentialByUsername(_ context.Context, _ uuid.UUID, _ string) (*LocalCredential, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return m.credential, nil
}

// --- Helpers ---

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := crypto.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return hash
}

func localCtx(t *testing.T) context.Context {
	t.Helper()
	return WithTenantContext(context.Background(), uuid.New())
}

// --- Tests ---

func TestLocalProvider_Type(t *testing.T) {
	p := NewLocalProvider(nil)
	if p.Type() != ProviderLocal {
		t.Errorf("expected %s, got %s", ProviderLocal, p.Type())
	}
}

func TestLocalProvider_Name(t *testing.T) {
	p := NewLocalProvider(nil)
	if p.Name() != "local" {
		t.Errorf("expected 'local', got '%s'", p.Name())
	}
}

func TestLocalProvider_Authenticate_Success(t *testing.T) {
	userID := uuid.New()
	hash := mustHashPassword(t, "correctPass123")

	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       userID,
			Username:     "testuser",
			Email:        "test@example.com",
			Status:       "active",
			PasswordHash: hash,
		},
	}

	provider := NewLocalProvider(store)
	result, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "testuser",
		Password: "correctPass123",
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result.Provider != ProviderLocal {
		t.Errorf("expected provider local, got %s", result.Provider)
	}
	if result.LinkedUser == nil || *result.LinkedUser != userID {
		t.Errorf("expected linked user %s, got %+v", userID, result.LinkedUser)
	}
	if result.Attributes["username"] != "testuser" {
		t.Errorf("expected username attribute, got %v", result.Attributes["username"])
	}
	if result.Attributes["email"] != "test@example.com" {
		t.Errorf("expected email attribute, got %v", result.Attributes["email"])
	}
}

func TestLocalProvider_Authenticate_WrongPassword(t *testing.T) {
	hash := mustHashPassword(t, "correctPass123")

	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       uuid.New(),
			Username:     "testuser",
			Status:       "active",
			PasswordHash: hash,
		},
	}

	provider := NewLocalProvider(store)
	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "testuser",
		Password: "wrongPassword",
	})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	ge, ok := gerr.AsGGIDError(err)
	if !ok {
		t.Fatalf("expected GGIDError, got %T: %v", err, err)
	}
	if ge.Code != gerr.ErrUnauthenticated {
		t.Errorf("expected ErrUnauthenticated, got %s", ge.Code)
	}
}

func TestLocalProvider_Authenticate_UserNotFound(t *testing.T) {
	store := &mockCredentialStore{
		err: gerr.NotFound("user", "nonexistent"),
	}

	provider := NewLocalProvider(store)
	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "nonexistent",
		Password: "somepassword",
	})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}

	// Should return unauthenticated, not the original not-found error.
	ge, ok := gerr.AsGGIDError(err)
	if !ok {
		t.Fatalf("expected GGIDError, got %T: %v", err, err)
	}
	if ge.Code != gerr.ErrUnauthenticated {
		t.Errorf("expected ErrUnauthenticated (not not-found to avoid user enumeration), got %s", ge.Code)
	}
}

func TestLocalProvider_Authenticate_LockedUser(t *testing.T) {
	hash := mustHashPassword(t, "correctPass123")

	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       uuid.New(),
			Username:     "lockeduser",
			Status:       "locked",
			PasswordHash: hash,
		},
	}

	provider := NewLocalProvider(store)
	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "lockeduser",
		Password: "correctPass123",
	})
	if err == nil {
		t.Fatal("expected error for locked user")
	}

	ge, ok := gerr.AsGGIDError(err)
	if !ok {
		t.Fatalf("expected GGIDError, got %T: %v", err, err)
	}
	if ge.Code != gerr.ErrFailedPrecondition {
		t.Errorf("expected ErrFailedPrecondition, got %s", ge.Code)
	}
}

func TestLocalProvider_Authenticate_DisabledUser(t *testing.T) {
	hash := mustHashPassword(t, "correctPass123")

	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       uuid.New(),
			Username:     "disableduser",
			Status:       "disabled",
			PasswordHash: hash,
		},
	}

	provider := NewLocalProvider(store)
	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "disableduser",
		Password: "correctPass123",
	})
	if err == nil {
		t.Fatal("expected error for disabled user")
	}
}

func TestLocalProvider_Authenticate_EmptyCredentials(t *testing.T) {
	store := &mockCredentialStore{}
	provider := NewLocalProvider(store)

	// Empty username.
	_, err := provider.Authenticate(localCtx(t), Credentials{Password: "pass"})
	if err == nil {
		t.Fatal("expected error for empty username")
	}

	// Empty password.
	_, err = provider.Authenticate(localCtx(t), Credentials{Username: "user"})
	if err == nil {
		t.Fatal("expected error for empty password")
	}

	// Store should never have been called.
	if store.called {
		t.Error("credential store should not be called with empty credentials")
	}
}

func TestLocalProvider_Authenticate_MissingTenantContext(t *testing.T) {
	store := &mockCredentialStore{}
	provider := NewLocalProvider(store)

	_, err := provider.Authenticate(context.Background(), Credentials{
		Username: "testuser",
		Password: "password",
	})
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestLocalProvider_Authenticate_CorruptedHash(t *testing.T) {
	store := &mockCredentialStore{
		credential: &LocalCredential{
			UserID:       uuid.New(),
			Username:     "testuser",
			Status:       "active",
			PasswordHash: "invalid-hash-format",
		},
	}

	provider := NewLocalProvider(store)
	_, err := provider.Authenticate(localCtx(t), Credentials{
		Username: "testuser",
		Password: "somepassword",
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
