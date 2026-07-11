package service

// Gap Regression Verification Test
// Verifies: Gap #7 — Refresh Token Rotation (DONE, grep-only → functional)
// Date: 2026-07-24

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// ========== GAP #7: Refresh Token Rotation — Functional Verification ==========

func TestGapRegression_RefreshRotation_ValidRotation(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, _ := newTestTokenSvc(t, refreshRepo)

	ctx := context.Background()
	plaintext := "valid-refresh-token-for-rotation-test"
	oldToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TenantID:  uuid.New(),
		SessionID: uuid.New(),
		TokenHash: hashToken(plaintext),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = refreshRepo.Create(ctx, oldToken)

	newToken, rotated, err := tokenSvc.RotateRefreshToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}
	if newToken == "" {
		t.Fatal("expected non-empty new token")
	}
	if rotated == nil {
		t.Fatal("expected non-nil rotated token record")
	}
}

func TestGapRegression_RefreshRotation_ReplayDetection(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, _ := newTestTokenSvc(t, refreshRepo)

	ctx := context.Background()
	plaintext := "token-to-replay"
	oldToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TenantID:  uuid.New(),
		SessionID: uuid.New(),
		TokenHash: hashToken(plaintext),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = refreshRepo.Create(ctx, oldToken)

	_, _, err := tokenSvc.RotateRefreshToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("first rotation should succeed: %v", err)
	}

	_, _, err = tokenSvc.RotateRefreshToken(ctx, plaintext)
	if err == nil {
		t.Fatal("REPLAY ATTACK: second rotation with revoked token should FAIL")
	}
}

func TestGapRegression_RefreshRotation_InvalidToken(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, _ := newTestTokenSvc(t, refreshRepo)

	_, _, err := tokenSvc.RotateRefreshToken(context.Background(), "nonexistent-token")
	if err == nil {
		t.Fatal("invalid refresh token should be rejected")
	}
}

func TestGapRegression_RefreshRotation_EmptyToken(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, _ := newTestTokenSvc(t, refreshRepo)

	_, _, err := tokenSvc.RotateRefreshToken(context.Background(), "")
	if err == nil {
		t.Fatal("empty token should be rejected")
	}
}

func TestGapRegression_RefreshRotation_NewTokenDiffersFromOld(t *testing.T) {
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, _ := newTestTokenSvc(t, refreshRepo)

	ctx := context.Background()
	plaintext := "old-token-value"
	oldToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TenantID:  uuid.New(),
		SessionID: uuid.New(),
		TokenHash: hashToken(plaintext),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = refreshRepo.Create(ctx, oldToken)

	newToken, _, err := tokenSvc.RotateRefreshToken(ctx, plaintext)
	if err != nil {
		t.Fatalf("rotation failed: %v", err)
	}
	if newToken == plaintext {
		t.Fatal("new token must differ from old — rotation must produce fresh token")
	}
}
