package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// errorInjectRepo wraps mockRepo and overrides specific methods to return
// injected errors. Real parameters are forwarded to the embedded mockRepo
// when no error is injected, preserving correct internal state.
type errorInjectRepo struct {
	*mockRepo
	listEmailsErr        error
	listEmailsEmpty      bool
	createUserErr2       error
	addEmailErr2         error
	setPrimaryEmailErr2  error
	createTokenErr2      error
	consumeTokenErr2     error
	updateUserErr        error
	deleteUserErr        error
	listUsersErr         error
	setStatusErr         error
	removeEmailErr       error
	getUserByEmailIDErr  error
	listExtIdentityErr   error
	findExtIdentityErr2  error
	linkExtIdentityErr2  error
}

func newErrorInjectRepo() *errorInjectRepo {
	return &errorInjectRepo{mockRepo: newMockRepo()}
}

func (m *errorInjectRepo) CreateUser(ctx context.Context, user *domain.User) error {
	if m.createUserErr2 != nil {
		return m.createUserErr2
	}
	return m.mockRepo.CreateUser(ctx, user)
}

func (m *errorInjectRepo) ListUserEmails(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.UserEmail, error) {
	if m.listEmailsErr != nil {
		return nil, m.listEmailsErr
	}
	if m.listEmailsEmpty {
		return []*domain.UserEmail{}, nil
	}
	return m.mockRepo.ListUserEmails(ctx, tenantID, userID)
}

func (m *errorInjectRepo) AddUserEmail(ctx context.Context, tenantID, userID uuid.UUID, email string) (*domain.UserEmail, error) {
	if m.addEmailErr2 != nil {
		return nil, m.addEmailErr2
	}
	return m.mockRepo.AddUserEmail(ctx, tenantID, userID, email)
}

func (m *errorInjectRepo) SetPrimaryEmail(ctx context.Context, tenantID, userID, emailID uuid.UUID) (*domain.UserEmail, error) {
	if m.setPrimaryEmailErr2 != nil {
		return nil, m.setPrimaryEmailErr2
	}
	return m.mockRepo.SetPrimaryEmail(ctx, tenantID, userID, emailID)
}

func (m *errorInjectRepo) CreateEmailVerificationToken(ctx context.Context, token *domain.EmailVerificationToken) error {
	if m.createTokenErr2 != nil {
		return m.createTokenErr2
	}
	return m.mockRepo.CreateEmailVerificationToken(ctx, token)
}

func (m *errorInjectRepo) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (*domain.EmailVerificationToken, error) {
	if m.consumeTokenErr2 != nil {
		return nil, m.consumeTokenErr2
	}
	return m.mockRepo.ConsumeEmailVerificationToken(ctx, tokenHash)
}

func (m *errorInjectRepo) UpdateUser(ctx context.Context, tenantID, id uuid.UUID, input *domain.UpdateUserInput) (*domain.User, error) {
	if m.updateUserErr != nil {
		return nil, m.updateUserErr
	}
	return m.mockRepo.UpdateUser(ctx, tenantID, id, input)
}

func (m *errorInjectRepo) DeleteUser(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteUserErr != nil {
		return m.deleteUserErr
	}
	return m.mockRepo.DeleteUser(ctx, tenantID, id)
}

func (m *errorInjectRepo) ListUsers(ctx context.Context, filter *domain.ListUsersFilter) (*domain.ListUsersResult, error) {
	if m.listUsersErr != nil {
		return nil, m.listUsersErr
	}
	return m.mockRepo.ListUsers(ctx, filter)
}

func (m *errorInjectRepo) SetUserStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.UserStatus) (*domain.User, error) {
	if m.setStatusErr != nil {
		return nil, m.setStatusErr
	}
	return m.mockRepo.SetUserStatus(ctx, tenantID, id, status)
}

func (m *errorInjectRepo) RemoveUserEmail(ctx context.Context, tenantID, userID uuid.UUID, email string) error {
	if m.removeEmailErr != nil {
		return m.removeEmailErr
	}
	return m.mockRepo.RemoveUserEmail(ctx, tenantID, userID, email)
}

func (m *errorInjectRepo) GetUserByEmailID(ctx context.Context, tenantID, emailID uuid.UUID) (*domain.UserEmail, error) {
	if m.getUserByEmailIDErr != nil {
		return nil, m.getUserByEmailIDErr
	}
	return m.mockRepo.GetUserByEmailID(ctx, tenantID, emailID)
}

func (m *errorInjectRepo) ListExternalIdentities(ctx context.Context, tenantID, userID uuid.UUID) ([]*domain.ExternalIdentity, error) {
	if m.listExtIdentityErr != nil {
		return nil, m.listExtIdentityErr
	}
	return m.mockRepo.ListExternalIdentities(ctx, tenantID, userID)
}

func (m *errorInjectRepo) FindExternalIdentity(ctx context.Context, tenantID uuid.UUID, provider, externalID string) (*domain.ExternalIdentity, error) {
	if m.findExtIdentityErr2 != nil {
		return nil, m.findExtIdentityErr2
	}
	return m.mockRepo.FindExternalIdentity(ctx, tenantID, provider, externalID)
}

func (m *errorInjectRepo) LinkExternalIdentity(ctx context.Context, ei *domain.ExternalIdentity) (*domain.ExternalIdentity, error) {
	if m.linkExtIdentityErr2 != nil {
		return nil, m.linkExtIdentityErr2
	}
	return m.mockRepo.LinkExternalIdentity(ctx, ei)
}

// --- CreateUser error path tests ---

func TestCreateUser_RepoCreateUserError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.createUserErr2 = errors.New("db connection lost")
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	_, err := svc.CreateUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error from repo.CreateUser")
	}
}

func TestCreateUser_AddEmailError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.addEmailErr2 = errors.New("email insert failed")
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	_, err := svc.CreateUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error from repo.AddUserEmail")
	}
}

func TestCreateUser_SetPrimaryEmailError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.setPrimaryEmailErr2 = errors.New("set primary failed")
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	_, err := svc.CreateUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error from repo.SetPrimaryEmail")
	}
}

func TestCreateUser_WithPhoneAndTimezone(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username:    "phoneuser",
		Email:       "phone@example.com",
		Password:    "SecurePassword123!",
		Phone:       "+1234567890",
		DisplayName: "Phone User",
		Locale:      "fr",
		Timezone:    "Europe/Paris",
	}

	user, err := svc.CreateUser(testCtx(testTenantID), input)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Phone != "+1234567890" {
		t.Errorf("expected phone '+1234567890', got '%s'", user.Phone)
	}
	if user.Timezone != "Europe/Paris" {
		t.Errorf("expected timezone 'Europe/Paris', got '%s'", user.Timezone)
	}
	if user.Locale != "fr" {
		t.Errorf("expected locale 'fr', got '%s'", user.Locale)
	}
	if user.PrimaryEmailID == nil {
		t.Error("expected PrimaryEmailID to be set after CreateUser")
	}
}

// --- RegisterUser error path tests ---

func TestRegisterUser_ListEmailsEmpty(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.listEmailsEmpty = true
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "SecurePassword123!",
	}

	_, _, err := svc.RegisterUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error when ListUserEmails returns empty")
	}
}

func TestRegisterUser_ListEmailsError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.listEmailsErr = errors.New("db query failed")
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "SecurePassword123!",
	}

	_, _, err := svc.RegisterUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error when ListUserEmails fails")
	}
}

func TestRegisterUser_CreateTokenError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.createTokenErr2 = errors.New("token insert failed")
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "SecurePassword123!",
	}

	_, _, err := svc.RegisterUser(testCtx(testTenantID), input)
	if err == nil {
		t.Fatal("expected error when CreateEmailVerificationToken fails")
	}
}

func TestRegisterUser_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "SecurePassword123!",
	}

	_, _, err := svc.RegisterUser(context.Background(), input)
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

// --- UpdateUser error path tests ---

func TestUpdateUser_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	newPhone := "555-1234"
	_, err := svc.UpdateUser(testCtx(testTenantID), uuid.New(), &domain.UpdateUserInput{
		Phone: &newPhone,
	})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestUpdateUser_AllFields(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Status:   domain.UserStatusActive,
	}

	phone := "555-0000"
	display := "New Display"
	avatar := "https://example.com/avatar.png"
	locale := "de"
	tz := "UTC"

	user, err := svc.UpdateUser(testCtx(testTenantID), userID, &domain.UpdateUserInput{
		Phone:       &phone,
		DisplayName: &display,
		AvatarURL:   &avatar,
		Locale:      &locale,
		Timezone:    &tz,
	})
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if user.Phone != phone {
		t.Errorf("expected phone '%s', got '%s'", phone, user.Phone)
	}
	if user.DisplayName != display {
		t.Errorf("expected displayName '%s', got '%s'", display, user.DisplayName)
	}
	if user.AvatarURL != avatar {
		t.Errorf("expected avatarURL '%s', got '%s'", avatar, user.AvatarURL)
	}
	if user.Locale != locale {
		t.Errorf("expected locale '%s', got '%s'", locale, user.Locale)
	}
	if user.Timezone != tz {
		t.Errorf("expected timezone '%s', got '%s'", tz, user.Timezone)
	}
}

func TestUpdateUser_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.UpdateUser(context.Background(), uuid.New(), &domain.UpdateUserInput{})
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestUpdateUser_RepoError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.updateUserErr = errors.New("update failed")
	svc := NewIdentityService(repo)

	newName := "X"
	_, err := svc.UpdateUser(testCtx(testTenantID), uuid.New(), &domain.UpdateUserInput{
		DisplayName: &newName,
	})
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

// --- DeleteUser error path tests ---

func TestDeleteUser_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	err := svc.DeleteUser(testCtx(testTenantID), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestDeleteUser_RepoError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.deleteUserErr = errors.New("db error")
	svc := NewIdentityService(repo)

	err := svc.DeleteUser(testCtx(testTenantID), uuid.New())
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

// --- Status change error paths ---

func TestRestoreUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "deleteduser",
		Status:   domain.UserStatusDeleted,
	}

	user, err := svc.RestoreUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("RestoreUser failed: %v", err)
	}
	if user.Status != domain.UserStatusActive {
		t.Errorf("expected status active, got %s", user.Status)
	}
}

func TestDisableUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Status:   domain.UserStatusActive,
	}

	user, err := svc.DisableUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("DisableUser failed: %v", err)
	}
	if user.Status != domain.UserStatusDisabled {
		t.Errorf("expected status disabled, got %s", user.Status)
	}
}

func TestDeactivateUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Status:   domain.UserStatusActive,
	}

	user, err := svc.DeactivateUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("DeactivateUser failed: %v", err)
	}
	if user.Status != domain.UserStatusDisabled {
		t.Errorf("expected status disabled, got %s", user.Status)
	}
}

func TestActivateUser_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	repo.users[userID] = &domain.User{
		ID:       userID,
		Username: "testuser",
		Status:   domain.UserStatusDisabled,
	}

	user, err := svc.ActivateUser(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("ActivateUser failed: %v", err)
	}
	if user.Status != domain.UserStatusActive {
		t.Errorf("expected status active, got %s", user.Status)
	}
}

func TestSetStatus_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.LockUser(testCtx(testTenantID), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestSetStatus_RepoError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.setStatusErr = errors.New("status update failed")
	svc := NewIdentityService(repo)

	_, err := svc.LockUser(testCtx(testTenantID), uuid.New())
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

// --- ListUsers error paths ---

func TestListUsers_RepoError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.listUsersErr = errors.New("query failed")
	svc := NewIdentityService(repo)

	_, err := svc.ListUsers(testCtx(testTenantID), &domain.ListUsersFilter{})
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

func TestListUsers_Pagination(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	for i := 0; i < 5; i++ {
		repo.users[uuid.New()] = &domain.User{
			Username: "user" + string(rune('a'+i)),
			Status:   domain.UserStatusActive,
		}
	}

	result, err := svc.ListUsers(testCtx(testTenantID), &domain.ListUsersFilter{
		PageSize: 2,
		Offset:   0,
	})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected total 5, got %d", result.Total)
	}
}

// --- Multi-Email management error paths ---

func TestListUserEmails_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.ListUserEmails(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestAddUserEmail_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.AddUserEmail(context.Background(), uuid.New(), "test@example.com")
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestRemoveUserEmail_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	email, err := svc.AddUserEmail(testCtx(testTenantID), userID, "remove@example.com")
	if err != nil {
		t.Fatalf("AddUserEmail failed: %v", err)
	}

	err = svc.RemoveUserEmail(testCtx(testTenantID), userID, email.Email)
	if err != nil {
		t.Fatalf("RemoveUserEmail failed: %v", err)
	}

	emails, _ := svc.ListUserEmails(testCtx(testTenantID), userID)
	for _, e := range emails {
		if e.Email == "remove@example.com" {
			t.Error("email should have been removed")
		}
	}
}

func TestRemoveUserEmail_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	err := svc.RemoveUserEmail(testCtx(testTenantID), uuid.New(), "nonexistent@example.com")
	if err == nil {
		t.Fatal("expected error for non-existent email")
	}
}

func TestRemoveUserEmail_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	err := svc.RemoveUserEmail(context.Background(), uuid.New(), "test@example.com")
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestSetPrimaryEmail_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	email1, _ := svc.AddUserEmail(testCtx(testTenantID), userID, "first@example.com")
	email2, _ := svc.AddUserEmail(testCtx(testTenantID), userID, "second@example.com")

	result, err := svc.SetPrimaryEmail(testCtx(testTenantID), userID, email2.ID)
	if err != nil {
		t.Fatalf("SetPrimaryEmail failed: %v", err)
	}
	if !result.IsPrimary {
		t.Error("expected IsPrimary to be true")
	}

	emails, _ := svc.ListUserEmails(testCtx(testTenantID), userID)
	for _, e := range emails {
		if e.ID == email1.ID && e.IsPrimary {
			t.Error("first email should no longer be primary")
		}
	}
}

func TestSetPrimaryEmail_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.SetPrimaryEmail(testCtx(testTenantID), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent email")
	}
}

func TestSetPrimaryEmail_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.SetPrimaryEmail(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

// --- External Identity error paths ---

func TestListExternalIdentities_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	ei := &domain.ExternalIdentity{
		UserID:     userID,
		Provider:   "github",
		ExternalID: "gh-123",
	}
	_, _ = svc.LinkExternalIdentity(testCtx(testTenantID), ei)

	result, err := svc.ListExternalIdentities(testCtx(testTenantID), userID)
	if err != nil {
		t.Fatalf("ListExternalIdentities failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 identity, got %d", len(result))
	}
}

func TestListExternalIdentities_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.ListExternalIdentities(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestLinkExternalIdentity_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.LinkExternalIdentity(context.Background(), &domain.ExternalIdentity{
		UserID:     uuid.New(),
		Provider:   "google",
		ExternalID: "g-123",
	})
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestUnlinkExternalIdentity_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	ei, _ := svc.LinkExternalIdentity(testCtx(testTenantID), &domain.ExternalIdentity{
		UserID:     userID,
		Provider:   "google",
		ExternalID: "g-123",
	})

	err := svc.UnlinkExternalIdentity(testCtx(testTenantID), userID, ei.ID)
	if err != nil {
		t.Fatalf("UnlinkExternalIdentity failed: %v", err)
	}

	identities, _ := svc.ListExternalIdentities(testCtx(testTenantID), userID)
	if len(identities) != 0 {
		t.Errorf("expected 0 identities after unlink, got %d", len(identities))
	}
}

func TestUnlinkExternalIdentity_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	err := svc.UnlinkExternalIdentity(testCtx(testTenantID), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent identity")
	}
}

func TestUnlinkExternalIdentity_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	err := svc.UnlinkExternalIdentity(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

// --- ProvisionFromLDAP additional edge cases ---

func TestProvisionFromLDAP_FallbackDisplayName(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	result := &authprovider.AuthResult{
		ExternalID: "CN=jdoe,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "jdoe",
			"mail":           "jdoe@corp.local",
			"cn":             "Jane Doe",
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err != nil {
		t.Fatalf("ProvisionFromLDAP failed: %v", err)
	}
	if user.DisplayName != "Jane Doe" {
		t.Errorf("expected displayName 'Jane Doe' (fallback to cn), got '%s'", user.DisplayName)
	}
}

func TestProvisionFromLDAP_NoEmail(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	result := &authprovider.AuthResult{
		ExternalID: "CN=noemail,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "noemail",
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err != nil {
		t.Fatalf("ProvisionFromLDAP failed: %v", err)
	}
	if user.Email != "" {
		t.Errorf("expected empty email, got '%s'", user.Email)
	}
	if user.PrimaryEmailID != nil {
		t.Error("expected nil PrimaryEmailID when no email")
	}
}

func TestProvisionFromLDAP_NoTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.ProvisionFromLDAP(context.Background(), &authprovider.AuthResult{
		ExternalID: "test",
		Provider:   authprovider.ProviderLDAP,
	})
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestProvisionFromLDAP_CreateUserError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.createUserErr2 = errors.New("create user failed")
	svc := NewIdentityService(repo)

	result := &authprovider.AuthResult{
		ExternalID: "CN=err,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "erruser",
		},
	}

	_, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err == nil {
		t.Fatal("expected error from CreateUser")
	}
}

func TestProvisionFromLDAP_LinkExternalIdentityError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.linkExtIdentityErr2 = errors.New("link failed")
	svc := NewIdentityService(repo)

	result := &authprovider.AuthResult{
		ExternalID: "CN=linkerr,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "linkuser",
			"mail":           "link@corp.local",
		},
	}

	_, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err == nil {
		t.Fatal("expected error from LinkExternalIdentity")
	}
}

func TestProvisionFromLDAP_NonStringAttr(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	result := &authprovider.AuthResult{
		ExternalID: "CN=attr,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": 12345,
			"mail":           "attr@corp.local",
			"displayName":    []string{"not", "a", "string"},
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(testTenantID), result)
	if err != nil {
		t.Fatalf("ProvisionFromLDAP failed: %v", err)
	}
	if user.Username != result.ExternalID {
		t.Errorf("expected username to fallback to ExternalID, got '%s'", user.Username)
	}
}

// --- VerifyEmail additional tests ---

func TestVerifyEmail_ConsumeTokenError(t *testing.T) {
	repo := newErrorInjectRepo()
	repo.consumeTokenErr2 = errors.New("consume error")
	svc := NewIdentityService(repo)

	_, err := svc.VerifyEmail(testCtx(testTenantID), "some-token")
	if err == nil {
		t.Fatal("expected error from ConsumeEmailVerificationToken")
	}
}

// --- GetUser error path ---

func TestGetUser_RepoError(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	_, err := svc.GetUser(testCtx(testTenantID), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

// --- Missing tenant context for all remaining methods ---

func TestAllMethodsMissingTenantContext(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)

	ctx := context.Background()

	if _, err := svc.ListUserEmails(ctx, uuid.New()); err == nil {
		t.Error("ListUserEmails should fail without tenant context")
	}
	if _, err := svc.AddUserEmail(ctx, uuid.New(), "x@example.com"); err == nil {
		t.Error("AddUserEmail should fail without tenant context")
	}
	if err := svc.RemoveUserEmail(ctx, uuid.New(), "x@example.com"); err == nil {
		t.Error("RemoveUserEmail should fail without tenant context")
	}
	if _, err := svc.SetPrimaryEmail(ctx, uuid.New(), uuid.New()); err == nil {
		t.Error("SetPrimaryEmail should fail without tenant context")
	}
	if _, err := svc.ListExternalIdentities(ctx, uuid.New()); err == nil {
		t.Error("ListExternalIdentities should fail without tenant context")
	}
	if _, err := svc.LinkExternalIdentity(ctx, &domain.ExternalIdentity{}); err == nil {
		t.Error("LinkExternalIdentity should fail without tenant context")
	}
	if err := svc.UnlinkExternalIdentity(ctx, uuid.New(), uuid.New()); err == nil {
		t.Error("UnlinkExternalIdentity should fail without tenant context")
	}
	if _, err := svc.RestoreUser(ctx, uuid.New()); err == nil {
		t.Error("RestoreUser should fail without tenant context")
	}
	if _, err := svc.DisableUser(ctx, uuid.New()); err == nil {
		t.Error("DisableUser should fail without tenant context")
	}
	if _, err := svc.DeactivateUser(ctx, uuid.New()); err == nil {
		t.Error("DeactivateUser should fail without tenant context")
	}
	if _, err := svc.ActivateUser(ctx, uuid.New()); err == nil {
		t.Error("ActivateUser should fail without tenant context")
	}
}
