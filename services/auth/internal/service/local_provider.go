package service

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

// LocalProvider implements authprovider.Provider for local DB password authentication.
type LocalProvider struct {
	credRepo *repository.CredentialRepository
	policy   conf.PasswordPolicy
}

// NewLocalProvider creates a LocalProvider backed by the credential repository.
func NewLocalProvider(credRepo *repository.CredentialRepository, policy conf.PasswordPolicy) *LocalProvider {
	return &LocalProvider{credRepo: credRepo, policy: policy}
}

func (p *LocalProvider) Type() authprovider.ProviderType { return authprovider.ProviderLocal }
func (p *LocalProvider) Name() string                    { return "local" }

// Authenticate verifies credentials against the local database.
func (p *LocalProvider) Authenticate(ctx context.Context, creds authprovider.Credentials) (*authprovider.AuthResult, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("local auth requires tenant context: %w", err)
	}

	cred, err := p.credRepo.FindByIDentifier(ctx, tc.TenantID, creds.Username)
	if err != nil {
		return nil, fmt.Errorf("lookup credential: %w", err)
	}
	if cred == nil {
		// Credential not found — let the chain try other providers
		return nil, fmt.Errorf("credential not found for %s", creds.Username)
	}

	if !cred.Enabled {
		return nil, fmt.Errorf("credential is disabled")
	}

	if cred.IsLocked() {
		return nil, ErrAccountLocked
	}

	match, err := crypto.VerifyPassword(creds.Password, cred.Secret)
	if err != nil || !match {
		// Increment failed attempts and persist
		cred.RegisterFailedAttempt(p.policy.MaxAttempts, p.policy.LockDuration)
		_ = p.credRepo.UpdateFailedAttempts(ctx, cred.ID, cred.FailedAttempts, cred.LockedUntil)
		return nil, ErrInvalidCredentials
	}

	// Success — reset failed attempts
	cred.ResetFailedAttempts()
	_ = p.credRepo.UpdateFailedAttempts(ctx, cred.ID, 0, nil)

	userID := cred.UserID
	return &authprovider.AuthResult{
		Provider:   authprovider.ProviderLocal,
		LinkedUser: &userID,
		Attributes: map[string]any{
			"identifier": cred.Identifier,
			"type":       cred.Type,
		},
	}, nil
}

// interface compliance
var _ authprovider.Provider = (*LocalProvider)(nil)

// CreateCredentialParams holds data for creating a new local credential.
type CreateCredentialParams struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	Identifier string
	Password   string
	Type       domain.CredentialType
}

// CreateCredential creates a new local credential after validating the password.
func (p *LocalProvider) CreateCredential(ctx context.Context, params CreateCredentialParams) error {
	hash, err := crypto.HashPassword(params.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	credType := params.Type
	if credType == "" {
		credType = domain.CredentialPassword
	}

	cred := &domain.Credential{
		TenantID:   params.TenantID,
		UserID:     params.UserID,
		Type:       credType,
		Identifier: params.Identifier,
		Secret:     hash,
		Enabled:    true,
	}
	return p.credRepo.Create(ctx, cred)
}
