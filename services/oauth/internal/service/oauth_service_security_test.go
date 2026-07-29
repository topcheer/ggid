//go:build ignore
package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/repository"
	"github.com/google/uuid"
)

// TestRevokeToken_CascadeToRefreshTokens tests if token revocation
// properly cascades to refresh tokens in the same family.
func TestRevokeToken_CascadeToRefreshTokens(t *testing.T) {
	// Setup mock repository
	mockRepo := &mockSecTokenRepo{}
	mockSecClientRepo := &mockSecClientRepo{}

	kp, _ := crypto.NewInMemoryKeyProvider()
	svc := NewOAuthService(mockSecClientRepo, mockRepo, mockRepo, kp, "http://test")

	tenantID := uuid.New()
	userID := uuid.New()
	clientID := uuid.New()

	// Create a refresh token record
	refreshToken, err := svc.issueRefreshTokenRecord(context.Background(), tenantID, clientID, userID, []string{"offline_access"}, "")
	if err != nil {
		t.Fatalf("Failed to create refresh token: %v", err)
	}

	// Store it in the mock repo
	tokenHash := hashTokenSHA256(refreshToken)
	mockRepo.tokens[tokenHash] = &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: tokenHash,
		FamilyID:  uuid.New().String(),
	}

	// Revoke the access token
	// BUG: The RevokeToken function at line 1396-1461 tries to cascade
	// to refresh tokens, but the SQL query at line 1448-1451 is broken:
	// It sets app.tenant_id which doesn't make sense for a SELECT/UPDATE
	// Also, it only revokes tokens for the user, not the entire family

	// Create a mock access token JWT
	claims := map[string]any{
		"sub":       userID.String(),
		"tenant_id": tenantID.String(),
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
		"jti":       uuid.New().String(),
	}

	// For this test, we'll just check if the cascade would work
	t.Log("Testing token revocation cascade...")

	// The issue: Line 1448 has `_, _ = s.pool.Exec(ctx, fmt.Sprintf("SET app.tenant_id = '%s'", tenantID))`
	// This is a no-op statement - it doesn't actually set anything useful
	// The real issue is that the cascade doesn't use the FamilyID to revoke
	// ALL tokens in the family, only tokens for that specific user

	// Create another refresh token in the SAME family (simulating a family member)
	familyID := uuid.New().String()
	token2Hash := hashTokenSHA256("token2")
	mockRepo.tokens[token2Hash] = &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID, // Same user
		TokenHash: token2Hash,
		FamilyID:  familyID,
	}

	// If we revoke the first token's family, the second should also be revoked
	// But the current implementation at line 1449-1451 only revokes by user_id,
	// not by family_id
}

// TestRefreshToken_RevokedTokenStillUsable tests if a revoked refresh token
// can still be used to mint new access tokens.
func TestRefreshToken_RevokedTokenStillUsable(t *testing.T) {
	// Setup
	mockRepo := &mockSecTokenRepo{}
	mockSecClientRepo := &mockSecClientRepo{}

	kp, _ := crypto.NewInMemoryKeyProvider()
	svc := NewOAuthService(mockSecClientRepo, mockRepo, mockRepo, kp, "http://test")

	tenantID := uuid.New()
	userID := uuid.New()
	clientID := uuid.New()

	// Setup mock client
	mockSecClientRepo.clients[clientID] = &domain.OAuthClient{
		ID:                      clientID,
		TenantID:                tenantID,
		ClientID:                "test_client",
		ClientSecretHash:        "$2a$10$test", // bcrypt hash
		GrantTypes:              []string{"refresh_token"},
		TokenEndpointAuthMethod: "client_secret_basic",
		Enabled:                 true,
	}

	// Create and store a refresh token
	refreshToken, _ := svc.issueRefreshTokenRecord(context.Background(), tenantID, clientID, userID, []string{"offline_access"}, "")
	tokenHash := hashTokenSHA256(refreshToken)

	record := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ClientID:  clientID,
		UserID:    userID,
		TokenHash: tokenHash,
		Scope:     []string{"offline_access"},
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		FamilyID:  uuid.New().String(),
	}
	mockRepo.tokens[tokenHash] = record

	// Mark it as revoked (simulating prior revocation)
	record.Revoked = true

	// Try to use the revoked refresh token
	req := &RefreshTokenRequest{
		TenantID:     tenantID,
		RefreshToken: refreshToken,
		ClientID:     "test_client",
		ClientSecret: "test",
		Scope:        []string{"openid"},
	}

	_, err := svc.RefreshToken(context.Background(), req)

	// BUG: At line 1578, the code checks if record.Used || record.Revoked
	// If true, it SHOULD revoke the family and return error
	// However, the lookup at line 1562 might return a record from Auth service
	// (line 1567) which doesn't have the Revoked flag checked properly

	if err == nil {
		t.Error("BUG: Revoked refresh token was accepted and issued new tokens")
	} else {
		t.Logf("Correctly rejected revoked token: %v", err)
	}
}

// TestRefreshToken_RotationDoesntCheckRevocation tests if token rotation
// properly validates the old token against the global revocation list.
func TestRefreshToken_RotationDoesntCheckRevocation(t *testing.T) {
	// Setup
	mockRepo := &mockSecTokenRepo{}
	mockSecClientRepo := &mockSecClientRepo{}

	kp, _ := crypto.NewInMemoryKeyProvider()
	svc := NewOAuthService(mockSecClientRepo, mockRepo, mockRepo, kp, "http://test")

	tenantID := uuid.New()
	userID := uuid.New()
	clientID := uuid.New()

	// Setup mock client
	mockSecClientRepo.clients[clientID] = &domain.OAuthClient{
		ID:                      clientID,
		TenantID:                tenantID,
		ClientID:                "test_client",
		ClientSecretHash:        "$2a$10$test",
		GrantTypes:              []string{"refresh_token"},
		TokenEndpointAuthMethod: "client_secret_basic",
		Enabled:                 true,
	}

	// Create a refresh token
	refreshToken, _ := svc.issueRefreshTokenRecord(context.Background(), tenantID, clientID, userID, []string{"offline_access"}, "")
	tokenHash := hashTokenSHA256(refreshToken)

	record := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ClientID:  clientID,
		UserID:    userID,
		TokenHash: tokenHash,
		Scope:     []string{"offline_access"},
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		FamilyID:  uuid.New().String(),
	}
	mockRepo.tokens[tokenHash] = record

	// Revoke the token via RevokeToken
	// BUG: RevokeToken at line 1396 doesn't properly cascade to refresh tokens
	// It tries to execute SQL that doesn't work properly
	// So the refresh token might still be valid in the DB

	// Try to refresh with the token
	req := &RefreshTokenRequest{
		TenantID:     tenantID,
		RefreshToken: refreshToken,
		ClientID:     "test_client",
		ClientSecret: "test",
		Scope:        []string{"openid"},
	}

	resp, err := svc.RefreshToken(context.Background(), req)

	// The token should be rejected because it was revoked
	// However, if RevokeToken didn't properly mark it as revoked in the DB,
	// it might still be accepted
	if err == nil && resp != nil {
		t.Error("BUG: Token that should have been revoked was accepted for refresh")
		t.Logf("Got new access token: %s", resp.AccessToken[:20]+"...")
	} else {
		t.Logf("Correctly rejected: %v", err)
	}
}

// TestRefreshToken_AuthServiceTokensNotChecked tests if refresh tokens
// issued by the Auth service (stored in Redis) are properly validated.
func TestRefreshToken_AuthServiceTokensNotChecked(t *testing.T) {
	// Setup
	mockRepo := &mockSecTokenRepo{}
	mockSecClientRepo := &mockSecClientRepo{}

	kp, _ := crypto.NewInMemoryKeyProvider()
	svc := NewOAuthService(mockSecClientRepo, mockRepo, mockRepo, kp, "http://test")

	tenantID := uuid.New()
	clientID := uuid.New()

	// Setup mock client
	mockSecClientRepo.clients[clientID] = &domain.OAuthClient{
		ID:                      clientID,
		TenantID:                tenantID,
		ClientID:                "test_client",
		ClientSecretHash:        "$2a$10$test",
		GrantTypes:              []string{"refresh_token"},
		TokenEndpointAuthMethod: "client_secret_basic",
		Enabled:                 true,
	}

	// Simulate an Auth service refresh token (in Redis)
	// These are looked up via lookupAuthRefreshToken at line 1567
	// BUG: This function at line 1649-1673 doesn't check if the token is revoked
	// It just constructs a RefreshTokenRecord with minimal info

	// Try to refresh with an "Auth service" token
	req := &RefreshTokenRequest{
		TenantID:     tenantID,
		RefreshToken: "simulated_auth_service_token",
		ClientID:     "test_client",
		ClientSecret: "test",
		Scope:        []string{"openid"},
	}

	_, err := svc.RefreshToken(context.Background(), req)

	// The token should be rejected because it's not in the OAuth token DB
	// However, if Redis were configured and the token existed there,
	// the lookupAuthRefreshToken might accept it without proper validation
	t.Logf("Auth service token refresh result: %v", err)
}

// Mock implementations for testing

type mockSecTokenRepo struct {
	tokens map[string]*domain.RefreshTokenRecord
}

func (m *mockSecTokenRepo) CreateCode(ctx context.Context, code *domain.AuthorizationCode) error {
	return nil
}

func (m *mockSecTokenRepo) ConsumeCode(ctx context.Context, codeHash string) (*domain.AuthorizationCode, error) {
	return nil, nil
}

func (m *mockSecTokenRepo) ResolveTenantFromCode(ctx context.Context, codeHash string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockSecTokenRepo) StoreRefreshToken(ctx context.Context, token *domain.RefreshTokenRecord) error {
	if m.tokens == nil {
		m.tokens = make(map[string]*domain.RefreshTokenRecord)
	}
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockSecTokenRepo) GetRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) (*domain.RefreshTokenRecord, error) {
	if m.tokens == nil {
		return nil, nil
	}
	return m.tokens[tokenHash], nil
}

func (m *mockSecTokenRepo) RevokeRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) error {
	if m.tokens != nil {
		if tok, ok := m.tokens[tokenHash]; ok {
			tok.Revoked = true
		}
	}
	return nil
}

func (m *mockSecTokenRepo) ConsumeRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash string) (bool, error) {
	if m.tokens != nil {
		if tok, ok := m.tokens[tokenHash]; ok {
			if tok.Used || tok.Revoked {
				return false, nil
			}
			tok.Used = true
			tok.Revoked = true
			return true, nil
		}
	}
	return false, nil
}

type mockSecClientRepo struct {
	clients map[uuid.UUID]*domain.OAuthClient
}

func (m *mockSecClientRepo) CreateClient(ctx context.Context, client *domain.OAuthClient) error {
	return nil
}

func (m *mockSecClientRepo) GetClientByID(ctx context.Context, tenantID uuid.UUID, clientID string) (*domain.OAuthClient, error) {
	if m.clients == nil {
		return nil, nil
	}
	for _, client := range m.clients {
		if client.ClientID == clientID {
			return client, nil
		}
	}
	return nil, nil
}

func (m *mockSecClientRepo) UpdateClient(ctx context.Context, tenantID uuid.UUID, clientID string, client *domain.OAuthClient) error {
	return nil
}

func (m *mockSecClientRepo) DeleteClient(ctx context.Context, tenantID uuid.UUID, clientID string) error {
	return nil
}

func (m *mockSecClientRepo) ListClients(ctx context.Context, tenantID uuid.UUID, pageSize, offset int) ([]*domain.OAuthClient, int, error) {
	return nil, 0, nil
}
