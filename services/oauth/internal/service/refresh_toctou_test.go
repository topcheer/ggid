package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// losingConsumeTokenRepo simulates the loser of the atomic-consume race:
// GetRefreshToken still returns the record as fresh (the racer read before
// the winner consumed it), but ConsumeRefreshToken reports not-consumed.
type losingConsumeTokenRepo struct {
	familyAwareTokenRepo
}

func (m *losingConsumeTokenRepo) ConsumeRefreshToken(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

// failingStoreTokenRepo fails StoreRefreshToken (simulating DB write outage
// during rotation).
type failingStoreTokenRepo struct {
	familyAwareTokenRepo
}

func (m *failingStoreTokenRepo) StoreRefreshToken(_ context.Context, _ *domain.RefreshTokenRecord) error {
	return fmt.Errorf("db write failed")
}

// TestRefreshToken_TOCTOU_LostRaceTreatedAsReuse verifies that when the
// atomic consume reports the token was already consumed (two concurrent
// requests both passed the read-time Used/Revoked check), the request is
// rejected as reuse and the whole family is revoked.
func TestRefreshToken_TOCTOU_LostRaceTreatedAsReuse(t *testing.T) {
	clientRepo := newMockClientRepo()
	tokenRepo := &losingConsumeTokenRepo{}
	fam := &fakeFamilyStore{}
	svc := NewOAuthService(clientRepo, newMockCodeRepo(), tokenRepo, newMockKeyProvider(), "https://test.ggid.dev")
	svc.SetTokenFamilyStore(fam)

	client := addRefreshClient(t, clientRepo, "gcid_toctou_1")
	familyID := uuid.New().String()
	seed := &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: testTenantID, ClientID: client.ID, UserID: uuid.New(),
		TokenHash: hashTokenSHA256("toctou-token"), ExpiresAt: time.Now().Add(24 * time.Hour),
		FamilyID: familyID,
	}
	_ = tokenRepo.StoreRefreshToken(context.Background(), seed)

	resp, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID: testTenantID, RefreshToken: "toctou-token", ClientID: "gcid_toctou_1",
	})
	if err == nil || !strings.Contains(err.Error(), "reuse detected") {
		t.Fatalf("expected reuse detection error, got resp=%v err=%v", resp, err)
	}
	if resp != nil {
		t.Fatal("response must be nil on reuse detection")
	}
	if len(fam.theft) != 1 || fam.theft[0] != familyID {
		t.Errorf("theft = %v, want [%s]", fam.theft, familyID)
	}
	if len(tokenRepo.familyRevoked) != 1 || tokenRepo.familyRevoked[0] != familyID {
		t.Errorf("familyRevoked = %v, want [%s]", tokenRepo.familyRevoked, familyID)
	}
}

// TestRefreshToken_StoreFailureReturnsError verifies that a failure to store
// the rotated refresh token fails the request instead of handing the client
// an unusable refresh token.
func TestRefreshToken_StoreFailureReturnsError(t *testing.T) {
	clientRepo := newMockClientRepo()
	tokenRepo := &failingStoreTokenRepo{}
	svc := NewOAuthService(clientRepo, newMockCodeRepo(), tokenRepo, newMockKeyProvider(), "https://test.ggid.dev")

	client := addRefreshClient(t, clientRepo, "gcid_toctou_2")
	seed := &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: testTenantID, ClientID: client.ID, UserID: uuid.New(),
		TokenHash: hashTokenSHA256("store-fail-token"), ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	// Seed via the embedded repo (bypass the failing override).
	_ = tokenRepo.familyAwareTokenRepo.StoreRefreshToken(context.Background(), seed)

	resp, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID: testTenantID, RefreshToken: "store-fail-token", ClientID: "gcid_toctou_2",
	})
	if err == nil {
		t.Fatalf("expected store failure error, got resp=%+v", resp)
	}
	if resp != nil {
		t.Fatal("response must be nil when rotated token cannot be stored")
	}
}
