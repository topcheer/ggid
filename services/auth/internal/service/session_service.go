package service

import (
	"context"
	"strings"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
)

// SessionService manages the session lifecycle: create, query, revoke, cleanup.
type SessionService struct {
	sessionRepo SessionRepo
}

func NewSessionService(sessionRepo SessionRepo) *SessionService {
	return &SessionService{
		sessionRepo: sessionRepo,
	}
}

// CreateSessionParams holds the data needed to create a new session.
type CreateSessionParams struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	IPAddress string
	UserAgent string
	TTL       time.Duration
}

// Create creates a new session with a random session token.
// Returns the session token (plaintext) and the Session domain object.
func (ss *SessionService) Create(ctx context.Context, p CreateSessionParams) (string, *domain.Session, error) {
	token, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", nil, err
	}

	tokenHash := hashToken(token)
	session := &domain.Session{
		ID:        uuid.New(),
		TenantID:  p.TenantID,
		UserID:    p.UserID,
		TokenHash: tokenHash,
		IPAddress: p.IPAddress,
		UserAgent: p.UserAgent,
		ExpiresAt: time.Now().Add(p.TTL),
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"mfa_verified": false},
		DeviceInfo: parseDeviceInfo(p.UserAgent),
	}

	if err := ss.sessionRepo.Create(ctx, session); err != nil {
		return "", nil, err
	}
	return token, session, nil
}

// FindByID looks up a session by ID.
func (ss *SessionService) FindByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	s, err := ss.sessionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ErrSessionNotFound
	}
	return s, nil
}

// ListByUser returns all active sessions for a user.
func (ss *SessionService) ListByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.Session, error) {
	return ss.sessionRepo.ListByUser(ctx, tenantID, userID)
}

// Revoke revokes a session by ID.
func (ss *SessionService) Revoke(ctx context.Context, id uuid.UUID) error {
	return ss.sessionRepo.Revoke(ctx, id)
}

// RevokeAllForUser revokes all sessions for a user except the given one.
func (ss *SessionService) RevokeAllForUser(ctx context.Context, tenantID, userID, exceptID uuid.UUID) error {
	return ss.sessionRepo.RevokeAllForUser(ctx, tenantID, userID, exceptID)
}

// UpdateSessionJTI writes the JTI and token expiry back to the session record (CAE Phase 2).
func (ss *SessionService) UpdateSessionJTI(ctx context.Context, sessionID uuid.UUID, jti string, tokenExp time.Time) error {
	if ss.sessionRepo == nil {
		return nil
	}
	return ss.sessionRepo.UpdateJTI(ctx, sessionID, jti, tokenExp)
}

// ListActiveJTIForUser returns JTI + token expiry for all active sessions of a user.
func (ss *SessionService) ListActiveJTIForUser(ctx context.Context, tenantID, userID uuid.UUID) ([]domain.SessionJTI, error) {
	if ss.sessionRepo == nil {
		return nil, nil
	}
	return ss.sessionRepo.ListActiveJTIForUser(ctx, tenantID, userID)
}

// CleanupExpired removes expired and revoked sessions. Returns count of deleted rows.
func (ss *SessionService) CleanupExpired(ctx context.Context, retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)
	return ss.sessionRepo.DeleteExpired(ctx, cutoff)
}

// parseDeviceInfo extracts basic info from a User-Agent string.
func parseDeviceInfo(ua string) map[string]any {
	info := map[string]any{}
	uaLower := strings.ToLower(ua)
	switch {
	case strings.Contains(uaLower, "chrome"):
		info["browser"] = "Chrome"
	case strings.Contains(uaLower, "firefox"):
		info["browser"] = "Firefox"
	case strings.Contains(uaLower, "safari"):
		info["browser"] = "Safari"
	case strings.Contains(uaLower, "edge"):
		info["browser"] = "Edge"
	default:
		info["browser"] = "Unknown"
	}
	switch {
	case strings.Contains(uaLower, "windows"):
		info["os"] = "Windows"
	case strings.Contains(uaLower, "mac"):
		info["os"] = "macOS"
	case strings.Contains(uaLower, "linux"):
		info["os"] = "Linux"
	case strings.Contains(uaLower, "android"):
		info["os"] = "Android"
	case strings.Contains(uaLower, "iphone") || strings.Contains(uaLower, "ios"):
		info["os"] = "iOS"
	default:
		info["os"] = "Unknown"
	}
	return info
}
