// Package service implements the business logic for the Identity Service.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/crypto"
	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/services/identity/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdentityService implements the core identity management operations.
type IdentityService struct {
	repo repository.UserRepository
}

// NewIdentityService creates a new IdentityService.
func NewIdentityService(repo repository.UserRepository) *IdentityService {
	return &IdentityService{repo: repo}
}

// Pool returns the underlying connection pool for direct queries.
func (s *IdentityService) Pool() *pgxpool.Pool { return s.repo.Pool() }

// --- User CRUD ---

// CreateUser creates a new user with a hashed password.
func (s *IdentityService) CreateUser(ctx context.Context, input *domain.CreateUserInput) (*domain.User, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}

	// Check for existing username or email.
	if existing, _ := s.repo.GetUserByUsername(ctx, tc.TenantID, input.Username); existing != nil {
		return nil, gerr.AlreadyExists("user", input.Username)
	}
	if existing, _ := s.repo.GetUserByEmail(ctx, tc.TenantID, input.Email); existing != nil {
		return nil, gerr.AlreadyExists("email", input.Email)
	}

	// Hash the password.
	hash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, gerr.Internal("hash password", err)
	}

	if input.Locale == "" {
		input.Locale = "en"
	}

	user := &domain.User{
		ID:           uuid.New(),
		TenantID:     tc.TenantID,
		Username:     input.Username,
		Email:        input.Email,
		Phone:        input.Phone,
		Status:       domain.UserStatusActive,
		EmailVerified: false,
		DisplayName:  input.DisplayName,
		Locale:       input.Locale,
		Timezone:     input.Timezone,
		PasswordHash: hash,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Also create a credential record in the credentials table so the auth
	// service can authenticate this user (auth queries credentials, not users).
	if pool := s.Pool(); pool != nil {
		_, _ = pool.Exec(ctx, `
			INSERT INTO credentials (tenant_id, user_id, type, identifier, secret, enabled)
			VALUES ($1, $2, 'password', $3, $4, true)
			ON CONFLICT DO NOTHING
		`, tc.TenantID, user.ID, input.Username, hash)
	}

	// Create the primary email record.
	pEmail, err := s.repo.AddUserEmail(ctx, tc.TenantID, user.ID, input.Email)
	if err != nil {
		return nil, err
	}

	// Set as primary.
	_, err = s.repo.SetPrimaryEmail(ctx, tc.TenantID, user.ID, pEmail.ID)
	if err != nil {
		return nil, err
	}

	user.PrimaryEmailID = &pEmail.ID
	return user, nil
}

// GetUser retrieves a user by ID.
func (s *IdentityService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.GetUserByID(ctx, tc.TenantID, id)
}

// ListUsers returns a paginated list of users.
func (s *IdentityService) ListUsers(ctx context.Context, filter *domain.ListUsersFilter) (*domain.ListUsersResult, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	filter.TenantID = tc.TenantID
	return s.repo.ListUsers(ctx, filter)
}

// UpdateUser updates mutable user fields.
func (s *IdentityService) UpdateUser(ctx context.Context, id uuid.UUID, input *domain.UpdateUserInput) (*domain.User, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.UpdateUser(ctx, tc.TenantID, id, input)
}

// DeleteUser soft-deletes a user.
func (s *IdentityService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.DeleteUser(ctx, tc.TenantID, id)
}

// RestoreUser restores a soft-deleted user (status → active).
func (s *IdentityService) RestoreUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.setStatus(ctx, id, domain.UserStatusActive)
}

// LockUser locks a user account.
func (s *IdentityService) LockUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.setStatus(ctx, id, domain.UserStatusLocked)
}

// UnlockUser reactivates a locked user.
func (s *IdentityService) UnlockUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.setStatus(ctx, id, domain.UserStatusActive)
}

// DisableUser disables a user account (admin action).
func (s *IdentityService) DisableUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.setStatus(ctx, id, domain.UserStatusDisabled)
}

// DeactivateUser deactivates a user account (sets status to inactive/disabled,
// prevents authentication). Alias for DisableUser with semantic clarity.
func (s *IdentityService) DeactivateUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.setStatus(ctx, id, domain.UserStatusDisabled)
}

// ActivateUser re-activates a deactivated user (sets status to active).
func (s *IdentityService) ActivateUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.setStatus(ctx, id, domain.UserStatusActive)
}

func (s *IdentityService) setStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus) (*domain.User, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.SetUserStatus(ctx, tc.TenantID, id, status)
}

// --- Registration & Email Verification ---

// RegisterUser creates a new self-registered user and returns a verification token.
// In production, the token is sent via email; in dev mode it is returned directly.
func (s *IdentityService) RegisterUser(ctx context.Context, input *domain.CreateUserInput) (*domain.User, string, error) {
	user, err := s.CreateUser(ctx, input)
	if err != nil {
		return nil, "", err
	}

	// Generate email verification token.
	plaintextToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, "", gerr.Internal("generate token", err)
	}

	tc, _ := tenant.FromContext(ctx)

	// Find the primary email to attach the token to.
	emails, err := s.repo.ListUserEmails(ctx, tc.TenantID, user.ID)
	if err != nil || len(emails) == 0 {
		return nil, "", gerr.Internal("list user emails", err)
	}

	tokenHash := hashTokenSHA256(plaintextToken)
	vToken := &domain.EmailVerificationToken{
		ID:        uuid.New(),
		TenantID:  tc.TenantID,
		UserID:    user.ID,
		EmailID:   emails[0].ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.repo.CreateEmailVerificationToken(ctx, vToken); err != nil {
		return nil, "", err
	}

	return user, plaintextToken, nil
}

// VerifyEmail consumes a verification token and marks the email as verified.
func (s *IdentityService) VerifyEmail(ctx context.Context, token string) (*uuid.UUID, error) {
	tokenHash := hashTokenSHA256(token)
	vToken, err := s.repo.ConsumeEmailVerificationToken(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	// Mark email as verified via a direct SQL update.
	// This is handled at the repo level in a real implementation.
	// For now, we return the user ID so the caller can take action.
	return &vToken.UserID, nil
}

// --- Multi-Email Management ---

func (s *IdentityService) ListUserEmails(ctx context.Context, userID uuid.UUID) ([]*domain.UserEmail, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.ListUserEmails(ctx, tc.TenantID, userID)
}

func (s *IdentityService) AddUserEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.UserEmail, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.AddUserEmail(ctx, tc.TenantID, userID, email)
}

func (s *IdentityService) RemoveUserEmail(ctx context.Context, userID uuid.UUID, email string) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.RemoveUserEmail(ctx, tc.TenantID, userID, email)
}

func (s *IdentityService) SetPrimaryEmail(ctx context.Context, userID, emailID uuid.UUID) (*domain.UserEmail, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.SetPrimaryEmail(ctx, tc.TenantID, userID, emailID)
}

// --- External Identity Management ---

func (s *IdentityService) ListExternalIdentities(ctx context.Context, userID uuid.UUID) ([]*domain.ExternalIdentity, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.ListExternalIdentities(ctx, tc.TenantID, userID)
}

func (s *IdentityService) LinkExternalIdentity(ctx context.Context, ei *domain.ExternalIdentity) (*domain.ExternalIdentity, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	ei.TenantID = tc.TenantID
	return s.repo.LinkExternalIdentity(ctx, ei)
}

func (s *IdentityService) UnlinkExternalIdentity(ctx context.Context, userID, identityID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	return s.repo.UnlinkExternalIdentity(ctx, tc.TenantID, userID, identityID)
}

// --- JIT Provisioning for LDAP ---

// ProvisionFromLDAP creates a local user record from LDAP attributes.
// Called by the auth service when AutoProvision is enabled and the LDAP
// provider returns NewUser=true.
func (s *IdentityService) ProvisionFromLDAP(ctx context.Context, result *authprovider.AuthResult) (*domain.User, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}

	// Check if already linked.
	existing, _ := s.repo.FindExternalIdentity(ctx, tc.TenantID, "ldap", result.ExternalID)
	if existing != nil {
		return s.repo.GetUserByID(ctx, tc.TenantID, existing.UserID)
	}

	// Extract attributes.
	username := getStringAttr(result.Attributes, "sAMAccountName")
	if username == "" {
		username = result.ExternalID
	}
	email := getStringAttr(result.Attributes, "mail")
	displayName := getStringAttr(result.Attributes, "displayName")
	if displayName == "" {
		displayName = getStringAttr(result.Attributes, "cn")
	}

	user := &domain.User{
		ID:           uuid.New(),
		TenantID:     tc.TenantID,
		Username:     username,
		Email:        email,
		Status:       domain.UserStatusActive,
		EmailVerified: true, // LDAP emails are considered verified
		DisplayName:  displayName,
		Locale:       "en",
		// No PasswordHash — LDAP users authenticate via LDAP
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create LDAP user: %w", err)
	}

	// Create primary email record.
	if email != "" {
		pEmail, err := s.repo.AddUserEmail(ctx, tc.TenantID, user.ID, email)
		if err == nil {
			_, _ = s.repo.SetPrimaryEmail(ctx, tc.TenantID, user.ID, pEmail.ID)
			user.PrimaryEmailID = &pEmail.ID
		}
	}

	// Link the external identity.
	ei := &domain.ExternalIdentity{
		ID:         uuid.New(),
		TenantID:   tc.TenantID,
		UserID:     user.ID,
		Provider:   "ldap",
		ExternalID: result.ExternalID,
		Metadata:   result.Attributes,
	}
	if _, err := s.repo.LinkExternalIdentity(ctx, ei); err != nil {
		return nil, fmt.Errorf("link LDAP identity: %w", err)
	}

	return user, nil
}

func getStringAttr(attrs map[string]any, key string) string {
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}