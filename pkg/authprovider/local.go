package authprovider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ggid/ggid/pkg/auth/multihash"
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

// RehashCallback is called when a legacy hash format is verified successfully
// and needs to be re-hashed to Argon2id. The callback should update the DB
// asynchronously (it must not block the login flow).
type RehashCallback func(ctx context.Context, userID uuid.UUID, plainPassword, oldHash string)

// LocalProvider authenticates users whose credentials are stored in the local database.
// Passwords are hashed with Argon2id via pkg/crypto.
type LocalProvider struct {
	store     LocalCredentialStore
	rehashCb  RehashCallback
}

// NewLocalProvider creates a new LocalProvider.
func NewLocalProvider(store LocalCredentialStore) *LocalProvider {
	return &LocalProvider{store: store}
}

// SetRehashCallback injects a callback for transparent password re-hashing.
// When a legacy format (bcrypt, PBKDF2, scrypt, SSHA) is verified successfully,
// this callback is invoked asynchronously to update the DB with an Argon2id hash.
func (p *LocalProvider) SetRehashCallback(cb RehashCallback) {
	p.rehashCb = cb
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

	// Try Argon2id first (native format).
	ok, err := crypto.VerifyPassword(creds.Password, lc.PasswordHash)
	if !ok && err != nil {
		// Argon2id verification failed — try multi-hash for legacy formats.
		mhOK, format, mhErr := multihash.VerifyPassword(creds.Password, lc.PasswordHash)

		if format == multihash.FormatUnknown {
			// Hash is corrupted or uses an unrecognized format — this is an
			// infrastructure error, not an authentication failure.
			return nil, errors.New(errors.ErrInternal,
				fmt.Sprintf("corrupted or unrecognized password hash for user %s", lc.UserID))
		}

		if mhErr != nil || !mhOK {
			return nil, errors.Unauthenticated("invalid credentials")
		}

		// Legacy format matched — trigger transparent rehashing.
		ok = true
		if p.rehashCb != nil {
			// Asynchronous rehash — must not block login.
			go func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("rehash callback panicked",
							"user_id", lc.UserID,
							"old_format", format,
							"panic", r)
					}
				}()
				slog.Info("transparent password rehash triggered",
					"user_id", lc.UserID,
					"old_format", format,
					"new_format", "argon2id")
				p.rehashCb(ctx, lc.UserID, creds.Password, lc.PasswordHash)
			}()
		}
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
			"email":     lc.Email,
		},
	}, nil
}
