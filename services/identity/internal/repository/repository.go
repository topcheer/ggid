// Package repository defines the data-access interfaces for the Identity Service.
// Interfaces live here so the service layer can depend on abstractions.
package repository

import (
	"context"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// UserRepository is the data-access interface for users and related entities.
type UserRepository interface {
	// --- User CRUD ---

	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error)
	GetUserByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateUserInput) (*domain.User, error)
	DeleteUser(ctx context.Context, tenantID, id uuid.UUID) error
	ListUsers(ctx context.Context, filter *domain.ListUsersFilter) (*domain.ListUsersResult, error)
	SetUserStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.UserStatus) (*domain.User, error)
	UpdateLastLogin(ctx context.Context, tenantID, id uuid.UUID, ip string) error
	UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error

	// --- Credential lookup (for auth providers) ---

	GetCredentialByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*authprovider.LocalCredential, error)

	// --- Email management ---

	ListUserEmails(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.UserEmail, error)
	AddUserEmail(ctx context.Context, tenantID, userID uuid.UUID, email string) (*domain.UserEmail, error)
	RemoveUserEmail(ctx context.Context, tenantID, userID uuid.UUID, email string) error
	SetPrimaryEmail(ctx context.Context, tenantID, userID, emailID uuid.UUID) (*domain.UserEmail, error)
	GetUserByEmailID(ctx context.Context, tenantID, emailID uuid.UUID) (*domain.UserEmail, error)

	// --- Email verification ---

	CreateEmailVerificationToken(ctx context.Context, token *domain.EmailVerificationToken) error
	ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error)

	// --- External identities ---

	ListExternalIdentities(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.ExternalIdentity, error)
	LinkExternalIdentity(ctx context.Context, ei *domain.ExternalIdentity) (*domain.ExternalIdentity, error)
	UnlinkExternalIdentity(ctx context.Context, tenantID, userID, identityID uuid.UUID) error
	FindExternalIdentity(ctx context.Context, tenantID uuid.UUID, provider, externalID string) (*domain.ExternalIdentity, error)
}
