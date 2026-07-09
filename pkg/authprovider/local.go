package authprovider

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
)

// LocalCredential is returned by LocalCredentialStore for a matched user.
type LocalCredential struct {
	UserID       uuid.UUID
	Username     string
	Email        string
	Status       string // active, locked, disabled, deleted
	PasswordHash string // Argon2id encoded hash
}

// LocalCredentialStore is implemented by the Identity repository.
// The LocalProvider uses it to look up stored password hashes.
type LocalCredentialStore interface {
	// GetCredentialByUsername looks up a user by username OR email within a tenant.
	GetCredentialByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*LocalCredential, error)
}

// LocalProvider authenticates users whose credentials are stored in the local database.
// Passwords are hashed with Argon2id via pkg/crypto.
type LocalProvider struct {
	store LocalCredentialStore
}

// NewLocalProvider creates a new LocalProvider.
func NewLocalProvider(store LocalCredentialStore) *LocalProvider {
	return &LocalProvider{store: store}
}

// Type returns the provider type.
func (p *LocalProvider) Type() ProviderType { return ProviderLocal }

// Name returns the human-readable name.
func (p *LocalProvider) Name() string { return "local" }

// Authenticate verifies username/password against local stored credentials.
func (p *LocalProvider) Authenticate(ctx context.Context, creds Credentials) (*AuthResult, error) {
	if creds.Username == "" || creds.Password == "" {
		return nil, errors.Unauthenticated("username and password are required")
	}

	tenantID, err := resolveTenantID(ctx)
	if err != nil {
		return nil, err
	}

	lc, err := p.store.GetCredentialByUsername(ctx, tenantID, creds.Username)
	if err != nil {
		// Intentionally do not reveal whether the user exists.
		return nil, errors.Unauthenticated("invalid credentials")
	}

	if lc.Status != "active" {
		return nil, errors.New(errors.ErrFailedPrecondition,
			fmt.Sprintf("user account is %s", lc.Status))
	}

	ok, err := crypto.VerifyPassword(creds.Password, lc.PasswordHash)
	if err != nil {
		return nil, errors.Wrap(errors.ErrInternal, "password verification failed", err)
	}
	if !ok {
		return nil, errors.Unauthenticated("invalid credentials")
	}

	uid := lc.UserID
	return &AuthResult{
		Provider:   ProviderLocal,
		LinkedUser: &uid,
		Attributes: map[string]any{
			"username": lc.Username,
			"email":    lc.Email,
		},
	}, nil
}
