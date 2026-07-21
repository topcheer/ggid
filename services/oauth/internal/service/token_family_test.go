package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// --- Task-E test doubles ---

// familyAwareTokenRepo extends mockTokenRepo with FamilyRevoker capability
// and call recording.
type familyAwareTokenRepo struct {
	mockTokenRepo
	familyRevoked []string
	revokeAll     int
}

func (m *familyAwareTokenRepo) RevokeRefreshTokensByFamily(_ context.Context, _ uuid.UUID, familyID string) error {
	m.familyRevoked = append(m.familyRevoked, familyID)
	// Mirror real behavior: mark all tokens in the family revoked.
	for _, rt := range m.refreshTokens {
		if rt.FamilyID == familyID {
			rt.Revoked = true
		}
	}
	return nil
}

func (m *familyAwareTokenRepo) RevokeAllRefreshTokens(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	m.revokeAll++
	return nil
}

// fakeFamilyStore is an in-memory TokenFamilyStore.
type fakeFamilyStore struct {
	rotations [][3]string // familyID, oldTokenID, newTokenID
	theft     []string
}

func (f *fakeFamilyStore) RegisterRotation(_ context.Context, familyID, oldTokenID, newTokenID string) error {
	f.rotations = append(f.rotations, [3]string{familyID, oldTokenID, newTokenID})
	return nil
}

func (f *fakeFamilyStore) MarkTheft(_ context.Context, familyID string) error {
	f.theft = append(f.theft, familyID)
	return nil
}

func (f *fakeFamilyStore) GetFamily(_ context.Context, familyID string) (map[string]any, error) {
	return map[string]any{"family_id": familyID}, nil
}

func newFamilyTestService() (*OAuthService, *mockClientRepo, *familyAwareTokenRepo, *fakeFamilyStore) {
	clientRepo := newMockClientRepo()
	tokenRepo := &familyAwareTokenRepo{}
	fam := &fakeFamilyStore{}
	svc := NewOAuthService(clientRepo, newMockCodeRepo(), tokenRepo, newMockKeyProvider(), "https://test.ggid.dev")
	svc.SetTokenFamilyStore(fam)
	return svc, clientRepo, tokenRepo, fam
}

func addRefreshClient(t *testing.T, repo *mockClientRepo, clientID string) *domain.OAuthClient {
	t.Helper()
	c := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   clientID,
		Name:       "Family Test",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"},
		Enabled:    true,
	}
	if err := repo.CreateClient(context.Background(), c); err != nil {
		t.Fatalf("create client: %v", err)
	}
	return c
}

// --- Tests ---

func TestRefreshToken_FamilyAssignedOnFirstRotation(t *testing.T) {
	svc, clientRepo, tokenRepo, fam := newFamilyTestService()
	client := addRefreshClient(t, clientRepo, "gcid_family_1")

	// Seed a refresh token WITHOUT a family (legacy token, e.g. issued by
	// the auth-service fallback path).
	seed := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		ClientID:  client.ID,
		UserID:    uuid.New(),
		TokenHash: hashTokenSHA256("seed-token-1"),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.StoreRefreshToken(context.Background(), seed)

	resp, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID: testTenantID, RefreshToken: "seed-token-1", ClientID: "gcid_family_1",
	})
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if resp.RefreshToken == "" {
		t.Fatal("expected rotated refresh token")
	}

	// New token must be rooted at a family = old token's ID.
	var newRec *domain.RefreshTokenRecord
	for _, rt := range tokenRepo.refreshTokens {
		if rt.ID != seed.ID {
			newRec = rt
		}
	}
	if newRec == nil {
		t.Fatal("new refresh token not stored")
	}
	if newRec.FamilyID != seed.ID.String() {
		t.Errorf("FamilyID = %q, want %q (root = first token)", newRec.FamilyID, seed.ID.String())
	}

	// Registry recorded the rotation.
	if len(fam.rotations) != 1 || fam.rotations[0][0] != seed.ID.String() ||
		fam.rotations[0][1] != seed.ID.String() || fam.rotations[0][2] != newRec.ID.String() {
		t.Errorf("rotations = %+v", fam.rotations)
	}
}

func TestRefreshToken_FamilyInheritedAcrossRotations(t *testing.T) {
	svc, clientRepo, tokenRepo, fam := newFamilyTestService()
	client := addRefreshClient(t, clientRepo, "gcid_family_2")

	seed := &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: testTenantID, ClientID: client.ID, UserID: uuid.New(),
		TokenHash: hashTokenSHA256("seed-token-2"), ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.StoreRefreshToken(context.Background(), seed)

	// First rotation.
	resp1, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID: testTenantID, RefreshToken: "seed-token-2", ClientID: "gcid_family_2",
	})
	if err != nil {
		t.Fatalf("rotation 1: %v", err)
	}
	// Second rotation must keep the same family.
	resp2, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID: testTenantID, RefreshToken: resp1.RefreshToken, ClientID: "gcid_family_2",
	})
	if err != nil {
		t.Fatalf("rotation 2: %v", err)
	}
	_ = resp2

	families := map[string]bool{}
	for _, rt := range tokenRepo.refreshTokens {
		if rt.FamilyID != "" {
			families[rt.FamilyID] = true
		}
	}
	if len(families) != 1 {
		t.Errorf("expected exactly 1 family across rotations, got %v", families)
	}
	if len(fam.rotations) != 2 {
		t.Errorf("expected 2 registered rotations, got %d", len(fam.rotations))
	}
}

func TestRefreshToken_ReuseRevokesWholeFamily(t *testing.T) {
	svc, clientRepo, tokenRepo, fam := newFamilyTestService()
	client := addRefreshClient(t, clientRepo, "gcid_family_3")
	familyID := uuid.New().String()
	userID := uuid.New()

	// A rotated (used) token and its successor in the same family.
	used := &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: testTenantID, ClientID: client.ID, UserID: userID,
		TokenHash: hashTokenSHA256("reused-token"), Used: true,
		ExpiresAt: time.Now().Add(24 * time.Hour), FamilyID: familyID,
	}
	successor := &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: testTenantID, ClientID: client.ID, UserID: userID,
		TokenHash: hashTokenSHA256("successor-token"),
		ExpiresAt: time.Now().Add(24 * time.Hour), FamilyID: familyID,
	}
	_ = tokenRepo.StoreRefreshToken(context.Background(), used)
	_ = tokenRepo.StoreRefreshToken(context.Background(), successor)

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID: testTenantID, RefreshToken: "reused-token", ClientID: "gcid_family_3",
	})
	if err == nil || !strings.Contains(err.Error(), "reuse detected") {
		t.Fatalf("expected reuse detection error, got %v", err)
	}

	// Theft marked for the family.
	if len(fam.theft) != 1 || fam.theft[0] != familyID {
		t.Errorf("theft = %v, want [%s]", fam.theft, familyID)
	}
	// Family-scoped revocation used (NOT client-wide).
	if len(tokenRepo.familyRevoked) != 1 || tokenRepo.familyRevoked[0] != familyID {
		t.Errorf("familyRevoked = %v, want [%s]", tokenRepo.familyRevoked, familyID)
	}
	if tokenRepo.revokeAll != 0 {
		t.Error("client-wide revoke must NOT run when family is known")
	}
	// Successor token in the family is revoked too.
	if !successor.Revoked {
		t.Error("successor token should be revoked with the family")
	}
}

func TestRefreshToken_ReuseLegacyWithoutFamily(t *testing.T) {
	svc, clientRepo, tokenRepo, fam := newFamilyTestService()
	client := addRefreshClient(t, clientRepo, "gcid_family_4")

	// Revoked token with NO family → legacy client-wide revocation.
	legacy := &domain.RefreshTokenRecord{
		ID: uuid.New(), TenantID: testTenantID, ClientID: client.ID, UserID: uuid.New(),
		TokenHash: hashTokenSHA256("legacy-revoked"), Revoked: true,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.StoreRefreshToken(context.Background(), legacy)

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		TenantID: testTenantID, RefreshToken: "legacy-revoked", ClientID: "gcid_family_4",
	})
	if err == nil {
		t.Fatal("expected reuse detection error")
	}
	if tokenRepo.revokeAll != 1 {
		t.Errorf("legacy path: revokeAll = %d, want 1", tokenRepo.revokeAll)
	}
	if len(tokenRepo.familyRevoked) != 0 {
		t.Error("family revoke must not run without a family ID")
	}
	if len(fam.theft) != 0 {
		t.Error("no theft marking without a family ID")
	}
}
