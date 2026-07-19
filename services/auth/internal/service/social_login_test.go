package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/google/uuid"
)

// --- Mock IdentityClient for SocialLogin tests ---

type mockSocialIdentityClient struct {
	mu                sync.Mutex
	users             map[string]*UserInfo            // keyed by email
	byID              map[uuid.UUID]*UserInfo          // keyed by userID
	externalIdentities map[string]*ExternalIdentityLink // keyed by "provider:externalID"
	createdUsers      []*UserInfo                       // track JIT-provisioned users
	linkedIdentities  []*ExternalIdentityLink           // track linked identities
}

func newMockSocialIdentityClient() *mockSocialIdentityClient {
	return &mockSocialIdentityClient{
		users:              make(map[string]*UserInfo),
		byID:               make(map[uuid.UUID]*UserInfo),
		externalIdentities: make(map[string]*ExternalIdentityLink),
	}
}

func (m *mockSocialIdentityClient) GetUser(_ context.Context, _ uuid.UUID, identifier string) (*UserInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[identifier]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found: %s", identifier)
}

func (m *mockSocialIdentityClient) GetUserByID(_ context.Context, _ uuid.UUID, userID uuid.UUID) (*UserInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.byID[userID]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("user not found: %s", userID)
}

func (m *mockSocialIdentityClient) FindExternalIdentity(_ context.Context, _ uuid.UUID, provider, externalID string) (*ExternalIdentityLink, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := provider + ":" + externalID
	if link, ok := m.externalIdentities[key]; ok {
		return link, nil
	}
	return nil, nil // not found = nil, nil
}

func (m *mockSocialIdentityClient) LinkExternalIdentity(_ context.Context, _ uuid.UUID, userID uuid.UUID, provider, externalID string, _ map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	link := &ExternalIdentityLink{
		UserID:     userID,
		Provider:   provider,
		ExternalID: externalID,
	}
	m.externalIdentities[provider+":"+externalID] = link
	m.linkedIdentities = append(m.linkedIdentities, link)
	return nil
}

func (m *mockSocialIdentityClient) CreateUserFromSocial(_ context.Context, tenantID uuid.UUID, username, email, displayName, provider, externalID string, _ map[string]any) (*UserInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user := &UserInfo{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Username:    username,
		Email:       email,
		Status:      "active",
		DisplayName: displayName,
	}
	m.users[email] = user
	m.byID[user.ID] = user

	// Also link the external identity.
	link := &ExternalIdentityLink{
		UserID:     user.ID,
		Provider:   provider,
		ExternalID: externalID,
	}
	m.externalIdentities[provider+":"+externalID] = link

	m.createdUsers = append(m.createdUsers, user)
	return user, nil
}

func (m *mockSocialIdentityClient) GetUserRoles(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]string, error) {
	return []string{"admin"}, nil
}

func (m *mockSocialIdentityClient) ResolveTenantBySlug(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, fmt.Errorf("not implemented in mock")
}

// --- Tests ---

func newSocialTestAuthService(t *testing.T, idClient IdentityClient) *AuthService {
	t.Helper()

	credRepo := &mockCredentialRepo{}
	refreshRepo := newMockRefreshTokenRepo()
	tokenSvc, rdb := newTestTokenSvc(t, refreshRepo)
	sessionRepo := newMockSessionRepo()
	sessionSvc := NewSessionService(sessionRepo)
	passwordSvc := NewPasswordService(conf.Default().Password, credRepo, rdb)
	rateLimiter := NewRateLimiter(rdb)
	chain := authprovider.NewChain()

	return &AuthService{
		cfg:             conf.Default(),
		chain:           chain,
		credentialRepo:  credRepo,
		tokenService:    tokenSvc,
		sessionService:  sessionSvc,
		passwordService: passwordSvc,
		rateLimiter:     rateLimiter,
		identityClient:  idClient,
		mfaService:      nil,
	}
}

func TestSocialLogin_JITProvisioning(t *testing.T) {
	idClient := newMockSocialIdentityClient()
	svc := newSocialTestAuthService(t, idClient)

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.New(),
		IsolationLevel: tenant.IsolationShared,
	})

	tokens, err := svc.SocialLogin(ctx, "google", "g-12345", "newuser@test.com", "New User", "https://avatar.test/g.png", "1.2.3.4", "TestAgent")
	if err != nil {
		t.Fatalf("SocialLogin failed: %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if tokens.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %s", tokens.TokenType)
	}

	// Verify JIT provisioning happened.
	if len(idClient.createdUsers) != 1 {
		t.Fatalf("expected 1 created user, got %d", len(idClient.createdUsers))
	}
	created := idClient.createdUsers[0]
	if created.Email != "newuser@test.com" {
		t.Errorf("expected email 'newuser@test.com', got '%s'", created.Email)
	}
}

func TestSocialLogin_ExistingIdentityLinked(t *testing.T) {
	idClient := newMockSocialIdentityClient()
	svc := newSocialTestAuthService(t, idClient)

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	// Pre-link an external identity.
	userID := uuid.New()
	idClient.byID[userID] = &UserInfo{
		ID:       userID,
		TenantID: tenantID,
		Username: "existing_user",
		Email:    "existing@test.com",
		Status:   "active",
	}
	idClient.externalIdentities["github:gh-789"] = &ExternalIdentityLink{
		UserID:     userID,
		Provider:   "github",
		ExternalID: "gh-789",
	}

	tokens, err := svc.SocialLogin(ctx, "github", "gh-789", "existing@test.com", "Existing User", "", "1.2.3.4", "TestAgent")
	if err != nil {
		t.Fatalf("SocialLogin failed: %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}

	// Should NOT have created a new user.
	if len(idClient.createdUsers) != 0 {
		t.Errorf("expected 0 created users, got %d", len(idClient.createdUsers))
	}
}

func TestSocialLogin_EmailMatch_LinkIdentity(t *testing.T) {
	idClient := newMockSocialIdentityClient()
	svc := newSocialTestAuthService(t, idClient)

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	// Pre-create a user with an email (but no external identity link).
	existingUserID := uuid.New()
	idClient.users["matched@test.com"] = &UserInfo{
		ID:       existingUserID,
		TenantID: tenantID,
		Username: "matched_user",
		Email:    "matched@test.com",
		Status:   "active",
	}
	idClient.byID[existingUserID] = idClient.users["matched@test.com"]

	tokens, err := svc.SocialLogin(ctx, "google", "g-link-me", "matched@test.com", "Matched User", "", "1.2.3.4", "TestAgent")
	if err != nil {
		t.Fatalf("SocialLogin failed: %v", err)
	}

	if tokens.AccessToken == "" {
		t.Error("expected non-empty access token")
	}

	// Should NOT have created a new user.
	if len(idClient.createdUsers) != 0 {
		t.Errorf("expected 0 created users (linked instead), got %d", len(idClient.createdUsers))
	}

	// Should have linked the identity.
	if len(idClient.linkedIdentities) != 1 {
		t.Fatalf("expected 1 linked identity, got %d", len(idClient.linkedIdentities))
	}
	link := idClient.linkedIdentities[0]
	if link.Provider != "google" || link.ExternalID != "g-link-me" {
		t.Errorf("unexpected link: %+v", link)
	}
	if link.UserID != existingUserID {
		t.Errorf("expected link to user %s, got %s", existingUserID, link.UserID)
	}
}

func TestSocialLogin_NoTenantContext(t *testing.T) {
	idClient := newMockSocialIdentityClient()
	svc := newSocialTestAuthService(t, idClient)

	_, err := svc.SocialLogin(context.Background(), "google", "g-1", "test@test.com", "Test", "", "1.2.3.4", "UA")
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestSocialLogin_SecondLogin_FindsExistingLink(t *testing.T) {
	idClient := newMockSocialIdentityClient()
	svc := newSocialTestAuthService(t, idClient)

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	// First login — JIT provision.
	tokens1, err := svc.SocialLogin(ctx, "google", "g-second", "second@test.com", "Second User", "", "1.2.3.4", "UA")
	if err != nil {
		t.Fatalf("first SocialLogin failed: %v", err)
	}
	if tokens1.AccessToken == "" {
		t.Error("expected non-empty access token on first login")
	}
	if len(idClient.createdUsers) != 1 {
		t.Fatalf("expected 1 created user after first login, got %d", len(idClient.createdUsers))
	}

	// Second login — should find the existing link.
	tokens2, err := svc.SocialLogin(ctx, "google", "g-second", "second@test.com", "Second User", "", "1.2.3.4", "UA")
	if err != nil {
		t.Fatalf("second SocialLogin failed: %v", err)
	}
	if tokens2.AccessToken == "" {
		t.Error("expected non-empty access token on second login")
	}
	// Should NOT create another user.
	if len(idClient.createdUsers) != 1 {
		t.Errorf("expected still 1 created user after second login, got %d", len(idClient.createdUsers))
	}
}
