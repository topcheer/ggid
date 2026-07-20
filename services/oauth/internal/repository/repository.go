// Package repository defines data-access interfaces for the OAuth Service.
package repository

import (
	"context"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// ClientRepository manages OAuth client registrations.
type ClientRepository interface {
	CreateClient(ctx context.Context, client *domain.OAuthClient) error
	GetClientByID(ctx context.Context, tenantID uuid.UUID, clientID string) (*domain.OAuthClient, error)
	ListClients(ctx context.Context, tenantID uuid.UUID, pageSize, offset int) ([]*domain.OAuthClient, int, error)
	UpdateClient(ctx context.Context, tenantID uuid.UUID, clientID string, client *domain.OAuthClient) (*domain.OAuthClient, error)
	DeleteClient(ctx context.Context, tenantID uuid.UUID, clientID string) error
}

// AuthorizationCodeRepository manages short-lived authorization codes.
type AuthorizationCodeRepository interface {
	CreateCode(ctx context.Context, code *domain.AuthorizationCode) error
	ConsumeCode(ctx context.Context, codeHash string) (*domain.AuthorizationCode, error)
	ResolveTenantFromCode(ctx context.Context, codeHash string) (uuid.UUID, error)
}

// IDTokenRepository stores ID token records for audit (the tokens themselves are stateless JWTs).
// It also manages refresh token records for rotation and reuse detection.
type IDTokenRepository interface {
	RecordIDToken(ctx context.Context, record *domain.IDTokenRecord) error
	StoreRefreshToken(ctx context.Context, record *domain.RefreshTokenRecord) error
	GetRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) (*domain.RefreshTokenRecord, error)
	RevokeRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) error
	RevokeAllRefreshTokens(ctx context.Context, tenantID, clientID uuid.UUID) error
}
