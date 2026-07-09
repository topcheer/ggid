package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/ggid/ggid/pkg/crypto"
	"github.com/google/uuid"
)

// mockRepo is a configurable in-memory mock for UserRepository.
type mockRepo struct {
	users             map[uuid.UUID]*domain.User
	emails            map[uuid.UUID]*domain.UserEmail
	externalIdentities []*domain.ExternalIdentity
	verificationTokens []*domain.EmailVerificationToken
	// errors to return (nil = success)
	createUserErr       error
	findExternalErr     error
	createEmailErr      error
	setPrimaryEmailErr  error
	createTokenErr      error
	consumeTokenErr     error
	linkExtIdentityErr  error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:              make(map[uuid.UUID]*domain.User),
		emails:             make(map[uuid.UUID]*domain.UserEmail),
		externalIdentities: []*domain.ExternalIdentity{},
		verificationTokens: []*domain.EmailVerificationToken{},
	}
}

func (m *mockRepo) CreateUser(_ context.Context, user *domain.User) error {
	if m.createUserErr != nil {
		return m.createUserErr
	}
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	return nil
}

func (m *mockRepo) GetUserByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, gerr.NotFound("user", id.String())
	}
	return u, nil
}

func (m *mockRepo) GetUserByUsername(_ context.Context, _ uuid.UUID, username string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, gerr.NotFound("user", username)
}

func (m *mockRepo) GetUserByEmail(_ context.Context, _ uuid.UUID, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, gerr.NotFound("email", email)
}

func (m *mockRepo) UpdateUser(_ context.Context, _ uuid.UUID, id uuid.UUID, input *domain.UpdateUserInput) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, gerr.NotFound("user", id.String())
	}
	if input.Phone != nil {
		u.Phone = *input.Phone
	}
	if input.DisplayName != nil {
		u.DisplayName = *input.DisplayName
	}
	if input.AvatarURL != nil {
		u.AvatarURL = *input.AvatarURL
	}
	if input.Locale != nil {
		u.Locale = *input.Locale
	}
	if input.Timezone != nil {
		u.Timezone = *input.Timezone
	}
	return u, nil
}

func (m *mockRepo) DeleteUser(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	u, ok := m.users[id]
	if !ok {
		return gerr.NotFound("user", id.String())
	}
	u.Status = domain.UserStatusDeleted
	now := time.Now()
	u.DeletedAt = &now
	return nil
}

func (m *mockRepo) ListUsers(_ context.Context, filter *domain.ListUsersFilter) (*domain.ListUsersResult, error) {
	var result []*domain.User
	for _, u := range m.users {
		if filter.Status != nil && u.Status != *filter.Status {
			continue
		}
		result = append(result, u)
	}
	return &domain.ListUsersResult{Users: result, Total: len(result)}, nil
}

func (m *mockRepo) SetUserStatus(_ context.Context, _ uuid.UUID, id uuid.UUID, status domain.UserStatus) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, gerr.NotFound("user", id.String())
	}
	u.Status = status
	return u, nil
}

func (m *mockRepo) UpdateLastLogin(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockRepo) UpdatePassword(_ context.Context, _ uuid.UUID, id uuid.UUID, hash string) error {
	u, ok := m.users[id]
	if !ok {
		return gerr.NotFound("user", id.String())
	}
	u.PasswordHash = hash
	return nil
}

func (m *mockRepo) GetCredentialByUsername(_ context.Context, _ uuid.UUID, username string) (*authprovider.LocalCredential, error) {
	for _, u := range m.users {
		if u.Username == username || u.Email == username {
			return &authprovider.LocalCredential{
				UserID:       u.ID,
				Username:     u.Username,
				Email:        u.Email,
				Status:       string(u.Status),
				PasswordHash: u.PasswordHash,
			}, nil
		}
	}
	return nil, gerr.NotFound("user", username)
}

func (m *mockRepo) ListUserEmails(_ context.Context, _ uuid.UUID, userID uuid.UUID) ([]*domain.UserEmail, error) {
	var result []*domain.UserEmail
	for _, e := range m.emails {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockRepo) AddUserEmail(_ context.Context, _ uuid.UUID, userID uuid.UUID, email string) (*domain.UserEmail, error) {
	if m.createEmailErr != nil {
		return nil, m.createEmailErr
	}
	e := &domain.UserEmail{
		ID:        uuid.New(),
		UserID:    userID,
		Email:     email,
		CreatedAt: time.Now(),
	}
	m.emails[e.ID] = e
	return e, nil
}

func (m *mockRepo) RemoveUserEmail(_ context.Context, _ uuid.UUID, userID uuid.UUID, email string) error {
	for id, e := range m.emails {
		if e.UserID == userID && e.Email == email {
			delete(m.emails, id)
			return nil
		}
	}
	return gerr.NotFound("email", email)
}

func (m *mockRepo) SetPrimaryEmail(_ context.Context, _ uuid.UUID, userID uuid.UUID, emailID uuid.UUID) (*domain.UserEmail, error) {
	if m.setPrimaryEmailErr != nil {
		return nil, m.setPrimaryEmailErr
	}
	// Clear all primary flags for this user.
	for _, e := range m.emails {
		if e.UserID == userID {
			e.IsPrimary = (e.ID == emailID)
		}
	}
	e, ok := m.emails[emailID]
	if !ok {
		return nil, gerr.NotFound("email", emailID.String())
	}
	return e, nil
}

func (m *mockRepo) GetUserByEmailID(_ context.Context, _ uuid.UUID, emailID uuid.UUID) (*domain.UserEmail, error) {
	e, ok := m.emails[emailID]
	if !ok {
		return nil, gerr.NotFound("email", emailID.String())
	}
	return e, nil
}

func (m *mockRepo) CreateEmailVerificationToken(_ context.Context, token *domain.EmailVerificationToken) error {
	if m.createTokenErr != nil {
		return m.createTokenErr
	}
	token.CreatedAt = time.Now()
	m.verificationTokens = append(m.verificationTokens, token)
	return nil
}

func (m *mockRepo) ConsumeEmailVerificationToken(_ context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	if m.consumeTokenErr != nil {
		return nil, m.consumeTokenErr
	}
	for _, t := range m.verificationTokens {
		if t.TokenHash == tokenHash && t.ConsumedAt == nil {
			now := time.Now()
			t.ConsumedAt = &now
			return t, nil
		}
	}
	return nil, gerr.InvalidArgument("invalid or expired verification token")
}

func (m *mockRepo) ListExternalIdentities(_ context.Context, _ uuid.UUID, userID uuid.UUID) ([]*domain.ExternalIdentity, error) {
	var result []*domain.ExternalIdentity
	for _, ei := range m.externalIdentities {
		if ei.UserID == userID {
			result = append(result, ei)
		}
	}
	return result, nil
}

func (m *mockRepo) LinkExternalIdentity(_ context.Context, ei *domain.ExternalIdentity) (*domain.ExternalIdentity, error) {
	if m.linkExtIdentityErr != nil {
		return nil, m.linkExtIdentityErr
	}
	ei.LinkedAt = time.Now()
	m.externalIdentities = append(m.externalIdentities, ei)
	return ei, nil
}

func (m *mockRepo) UnlinkExternalIdentity(_ context.Context, _ uuid.UUID, _ uuid.UUID, identityID uuid.UUID) error {
	for i, ei := range m.externalIdentities {
		if ei.ID == identityID {
			m.externalIdentities = append(m.externalIdentities[:i], m.externalIdentities[i+1:]...)
			return nil
		}
	}
	return gerr.NotFound("external identity", identityID.String())
}

func (m *mockRepo) FindExternalIdentity(_ context.Context, _ uuid.UUID, provider, externalID string) (*domain.ExternalIdentity, error) {
	if m.findExternalErr != nil {
		return nil, m.findExternalErr
	}
	for _, ei := range m.externalIdentities {
		if ei.Provider == provider && ei.ExternalID == externalID {
			return ei, nil
		}
	}
	return nil, gerr.NotFound("external identity", provider+":"+externalID)
}

// --- Test Helpers ---

func testCtx(tenantID uuid.UUID) context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})
}

var testTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// --- Tests ---

func TestCreateUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "SecurePassword123!",
		DisplayName: "Test User",
	}

	user, err := svc.CreateUser(testCtx(testTenantID), input)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("expected non-nil user ID")
	}
	if user.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", user.Username)
	}
	if user.Status != domain.UserStatusActive {
		t.Errorf("expected status active, got %s", user.Status)
	}
	if user.PasswordHash == "" {
		t.Error("expected password hash to be set")
	}
	if user.PasswordHash == "SecurePassword123!" {
		t.Error("password hash should not be plaintext")
	}

	// Verify password hash is valid Argon2id.
	ok, err := crypto.VerifyPassword("SecurePassword123!", user.PasswordHash)
	if err != nil || !ok {
		t.Errorf("password verification failed: ok=%v err=%v", ok, err)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	// Pre-create a user.
	repo.users[uuid.New()] = &domain.User{
		Username: "existing",
		Email:    "existing@example.com",
	}

	input := &domain.CreateUserInput{
		Username: "existing",
		Email:    "new@example.com",
		Password: "SecurePassword123!",
	}

	_, err := svc.CreateUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error for duplicate username")
	}

	ge, ok := gerr.AsGGIDError(err)
	if !ok {
		t.Fatalf("expected GGIDError, got %T: %v", err, err)
	}
	if ge.Code != gerr.ErrAlreadyExists {
		t.Errorf("expected ErrAlreadyExists, got %s", ge.Code)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	repo.users[uuid.New()] = &domain.User{
		Username: "someone",
		Email:    "taken@example.com",
	}

	input := &domain.CreateUserInput{
		Username: "newuser",
		Email:    "taken@example.com",
		Password: "SecurePassword123!",
	}

	_, err := svc.CreateUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestCreateUser_MissingTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	_, err := svc.CreateUser(context.Background(), input)
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestCreateUser_DefaultLocale(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "SecurePassword123!",
		// Locale intentionally empty
	}

	user, err := svc.CreateUser(testCtx(testTenantID), input)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Locale != "en" {
		t.Errorf("expected default locale 'en', got '%s'", user.Locale)
	}
}

func TestGetUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		Status:   domain.UserStatusActive,
	}

	user, err := svc.GetUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.ID != userID {
		t.Errorf("expected user ID %s, got %s", userID, user.ID)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.GetUser(testCtx(testTenantID), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestDeleteUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Status:   domain.UserStatusActive,
	}

	err := svc.DeleteUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	if repo.users[userID].Status != domain.UserStatusDeleted {
		t.Error("expected user status to be deleted")
	}
	if repo.users[userID].DeletedAt == nil {
		t.Error("expected deleted_at to be set")
	}
}

func TestLockUnlockUser(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Status:   domain.UserStatusActive,
	}

	// Lock
	user, err := svc.LockUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("LockUser failed: %v", err)
	}
	if user.Status != domain.UserStatusLocked {
		t.Errorf("expected status locked, got %s", user.Status)
	}

	// Unlock
	user, err = svc.UnlockUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("UnlockUser failed: %v", err)
	}
	if user.Status != domain.UserStatusActive {
		t.Errorf("expected status active, got %s", user.Status)
	}
}

func TestRegisterUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "SecurePassword123!",
	}

	user, token, err := svc.RegisterUser(testCtx(testTenantID), input)
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}

	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if token == "" {
		t.Error("expected non-empty verification token")
	}

	// Verify token was stored.
	if len(repo.verificationTokens) != 1 {
		t.Fatalf("expected 1 verification token, got %d", len(repo.verificationTokens))
	}

	storedToken := repo.verificationTokens[0]
	if storedToken.UserID != user.ID {
		t.Error("token user ID mismatch")
	}
	if storedToken.ConsumedAt != nil {
		t.Error("token should not be consumed yet")
	}
}

func TestRegisterUser_DuplicateFails(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	repo.users[uuid.New()] = &domain.User{
		Username: "existing",
		Email:    "existing@example.com",
	}

	input := &domain.CreateUserInput{
		Username: "existing",
		Email:    "new@example.com",
		Password: "SecurePassword123!",
	}

	_, _, err := svc.RegisterUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error for duplicate user")
	}
}

func TestVerifyEmail_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	// Pre-create a verification token.
	plainToken := "test-verification-token"
	tokenHash := hashTokenSHA256(plainToken)
	repo.verificationTokens = append(repo.verificationTokens, &domain.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	returnedUserID, err := svc.VerifyEmail(testCtx(testTenantID), plainToken)
	if err != nil {
		t.Fatalf("VerifyEmail failed: %v", err)
	}
	if *returnedUserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, *returnedUserID)
	}

	// Verify token is consumed.
	if repo.verificationTokens[0].ConsumedAt == nil {
		t.Error("expected token to be consumed")
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.VerifyEmail(testCtx(testTenantID), "nonexistent-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestVerifyEmail_AlreadyConsumed(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	plainToken := "consumed-token"
	tokenHash := hashTokenSHA256(plainToken)
	now := time.Now()
	repo.verificationTokens = append(repo.verificationTokens, &domain.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: tokenHash,
		ConsumedAt: &now,
	})

	_, err := svc.VerifyEmail(testCtx(testTenantID), plainToken)
	if err == nil {
		t.Fatal("expected error for already-consumed token")
	}
}

func TestProvisionFromLDAP_NewUser(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	result := &authprovider.AuthResult{
		ExternalID: "CN=jdoe,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "jdoe",
			"mail":           "jdoe@corp.local",
			"displayName":    "John Doe",
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err != nil {
		t.Fatalf("ProvisionFromLDAP failed: %v", err)
	}

	if user.Username != "jdoe" {
		t.Errorf("expected username 'jdoe', got '%s'", user.Username)
	}
	if user.Email != "jdoe@corp.local" {
		t.Errorf("expected email 'jdoe@corp.local', got '%s'", user.Email)
	}
	if user.DisplayName != "John Doe" {
		t.Errorf("expected displayName 'John Doe', got '%s'", user.DisplayName)
	}
	if !user.EmailVerified {
		t.Error("expected email_verified to be true for LDAP users")
	}
	if user.PasswordHash != "" {
		t.Error("expected empty password hash for LDAP users")
	}

	// Verify external identity was linked.
	if len(repo.externalIdentities) != 1 {
		t.Fatalf("expected 1 external identity, got %d", len(repo.externalIdentities))
	}
	ei := repo.externalIdentities[0]
	if ei.Provider != "ldap" {
		t.Errorf("expected provider 'ldap', got '%s'", ei.Provider)
	}
	if ei.ExternalID != "CN=jdoe,DC=corp,DC=local" {
		t.Errorf("expected external ID 'CN=jdoe,DC=corp,DC=local', got '%s'", ei.ExternalID)
	}
	if ei.UserID != user.ID {
		t.Error("external identity user ID mismatch")
	}
}

func TestProvisionFromLDAP_AlreadyLinked(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	// Pre-create a user and link an LDAP identity.
	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "jdoe",
		Email:    "jdoe@corp.local",
		Status:   domain.UserStatusActive,
	}
	repo.externalIdentities = append(repo.externalIdentities, &domain.ExternalIdentity{
		ID:         uuid.New(),
		UserID:     userID,
		Provider:   "ldap",
		ExternalID: "CN=jdoe,DC=corp,DC=local",
	})

	result := &authprovider.AuthResult{
		ExternalID: "CN=jdoe,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "jdoe",
			"mail":           "jdoe@corp.local",
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err != nil {
		t.Fatalf("ProvisionFromLDAP failed: %v", err)
	}
	if user.ID != userID {
		t.Errorf("expected existing user ID %s, got %s", userID, user.ID)
	}

	// Should not create a new user.
	if len(repo.users) != 1 {
		t.Errorf("expected 1 user, got %d", len(repo.users))
	}
}

func TestProvisionFromLDAP_FallbackUsername(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	result := &authprovider.AuthResult{
		ExternalID: "CN=users,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			// No sAMAccountName — should fall back to ExternalID
			"mail": "user@corp.local",
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err != nil {
		t.Fatalf("ProvisionFromLDAP failed: %v", err)
	}

	if user.Username != result.ExternalID {
		t.Errorf("expected fallback username '%s', got '%s'", result.ExternalID, user.Username)
	}
}

func TestListUsers(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	repo.users[uuid.New()] = &domain.User{Username: "user1", Status: domain.UserStatusActive}
	repo.users[uuid.New()] = &domain.User{Username: "user2", Status: domain.UserStatusActive}
	repo.users[uuid.New()] = &domain.User{Username: "user3", Status: domain.UserStatusLocked}

	result, err := svc.ListUsers(testCtx(testTenantID), &domain.ListUsersFilter{})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected 3 users, got %d", result.Total)
	}

	// Filter by status.
	activeStatus := domain.UserStatusActive
	result, err = svc.ListUsers(testCtx(testTenantID), &domain.ListUsersFilter{Status: &activeStatus})
	if err != nil {
		t.Fatalf("ListUsers with filter failed: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 active users, got %d", result.Total)
	}
}

func TestUpdateUser(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Locale:   "en",
	}

	newLocale := "zh-CN"
	newName := "Updated Name"
	input := &domain.UpdateUserInput{
		Locale:      &newLocale,
		DisplayName: &newName,
	}

	user, err := svc.UpdateUser(testCtx(testTenantID), userID, input)
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if user.Locale != "zh-CN" {
		t.Errorf("expected locale 'zh-CN', got '%s'", user.Locale)
	}
	if user.DisplayName != "Updated Name" {
		t.Errorf("expected displayName 'Updated Name', got '%s'", user.DisplayName)
	}
}

func TestLinkExternalIdentity(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	ei := &domain.ExternalIdentity{
		UserID:     userID,
		Provider:   "google",
		ExternalID: "google-12345",
		Metadata:   map[string]any{"email": "user@gmail.com"},
	}

	result, err := svc.LinkExternalIdentity(testCtx(testTenantID), ei)
	if err != nil {
		t.Fatalf("LinkExternalIdentity failed: %v", err)
	}
	if result.ID == uuid.Nil {
		t.Error("expected non-nil identity ID")
	}
	if result.Provider != "google" {
		t.Errorf("expected provider 'google', got '%s'", result.Provider)
	}
}

func TestAddUserEmail(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()

	email, err := svc.AddUserEmail(testCtx(testTenantID), userID, "secondary@example.com")
	if err != nil {
		t.Fatalf("AddUserEmail failed: %v", err)
	}
	if email.Email != "secondary@example.com" {
		t.Errorf("expected email 'secondary@example.com', got '%s'", email.Email)
	}
	if email.IsPrimary {
		t.Error("new email should not be primary")
	}
}

func TestAllMethodsRequireTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	// All these should fail without tenant context.
	ctx := context.Background()

	if _, err := svc.GetUser(ctx, uuid.New()); err == nil {
		t.Error("GetUser should fail without tenant context")
	}
	if err := svc.DeleteUser(ctx, uuid.New()); err == nil {
		t.Error("DeleteUser should fail without tenant context")
	}
	if _, err := svc.ListUsers(ctx, &domain.ListUsersFilter{}); err == nil {
		t.Error("ListUsers should fail without tenant context")
	}
	if _, err := svc.LockUser(ctx, uuid.New()); err == nil {
		t.Error("LockUser should fail without tenant context")
	}
	if _, err := svc.UnlockUser(ctx, uuid.New()); err == nil {
		t.Error("UnlockUser should fail without tenant context")
	}
}
