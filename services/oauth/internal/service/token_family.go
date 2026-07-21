package service

import (
	"context"

	"github.com/google/uuid"
)

// token_family.go — Task-E: RFC 6749 §10.4 refresh-token rotation families.
//
// A family groups all refresh tokens descending from one initial grant.
// On each rotation the family registry is updated; when a rotated/revoked
// token is presented (reuse = theft signal), the whole family is revoked.

// TokenFamilyStore persists token-family metadata. Implementations must be
// safe for concurrent use. The store is nil-safe: when nil, family tracking
// is skipped and reuse detection falls back to client-wide revocation.
type TokenFamilyStore interface {
	// RegisterRotation records that oldTokenID rotated into newTokenID.
	RegisterRotation(ctx context.Context, familyID, oldTokenID, newTokenID string) error
	// MarkTheft flags the family as compromised (reuse detected).
	MarkTheft(ctx context.Context, familyID string) error
	// GetFamily returns the family record (for the token-families API view).
	GetFamily(ctx context.Context, familyID string) (map[string]any, error)
}

// FamilyRevoker is an optional capability of the token repository for
// family-scoped revocation (PG repo implements it; memory/mocks may not).
type FamilyRevoker interface {
	RevokeRefreshTokensByFamily(ctx context.Context, tenantID uuid.UUID, familyID string) error
}

// revokeFamily revokes all tokens in a family when the repo supports it,
// otherwise revokes all tokens for the client (legacy behavior).
func (s *OAuthService) revokeFamily(ctx context.Context, tenantID, clientID uuid.UUID, familyID string) {
	if familyID != "" {
		if fr, ok := s.tokenRepo.(FamilyRevoker); ok {
			if err := fr.RevokeRefreshTokensByFamily(ctx, tenantID, familyID); err == nil {
				return
			}
		}
	}
	_ = s.tokenRepo.RevokeAllRefreshTokens(ctx, tenantID, clientID)
}

// SetTokenFamilyStore injects the family registry (PG-backed in production).
func (s *OAuthService) SetTokenFamilyStore(store TokenFamilyStore) {
	s.tokenFamilyStore = store
}
