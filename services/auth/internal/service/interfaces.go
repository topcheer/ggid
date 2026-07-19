package service

import (
	"context"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// CredentialRepo is the interface for credential persistence used by the service layer.
// The concrete *repository.CredentialRepository satisfies this interface.
type CredentialRepo interface {
	FindByIDentifier(ctx context.Context, tenantID uuid.UUID, identifier string) (*domain.Credential, error)
	FindByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Credential, error)
	Create(ctx context.Context, c *domain.Credential) error
	UpdateFailedAttempts(ctx context.Context, id uuid.UUID, attempts int, lockedUntil *time.Time) error
	UpdateSecret(ctx context.Context, id uuid.UUID, secret string) error
	AddToHistory(ctx context.Context, tenantID, userID uuid.UUID, secret string) error
	GetHistory(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]domain.CredentialHistoryEntry, error)
}

// SessionRepo is the interface for session persistence.
// The concrete *repository.SessionRepository satisfies this interface.
type SessionRepo interface {
	Create(ctx context.Context, s *domain.Session) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, tenantID, userID, exceptSessionID uuid.UUID) error
	DeleteExpired(ctx context.Context, cutoff time.Time) (int64, error)
	RevokeOldestForUser(ctx context.Context, tenantID, userID uuid.UUID, keepCount int) error

	// UpdateJTI writes the JTI and token expiry back to the session record (CAE Phase 2).
	UpdateJTI(ctx context.Context, sessionID uuid.UUID, jti string, tokenExp time.Time) error
	// ListActiveJTIForUser returns JTI + token expiry for all active sessions of a user.
	ListActiveJTIForUser(ctx context.Context, tenantID, userID uuid.UUID) ([]domain.SessionJTI, error)
}

// RefreshTokenRepo is the interface for refresh-token persistence.
// The concrete *repository.RefreshTokenRepository satisfies this interface.
type RefreshTokenRepo interface {
	Create(ctx context.Context, t *domain.RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeAllForSession(ctx context.Context, sessionID uuid.UUID) error
	RevokeAllForUser(ctx context.Context, tenantID, userID uuid.UUID) error
}
