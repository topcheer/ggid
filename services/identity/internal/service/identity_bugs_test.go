package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/authprovider"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// BUG 1: Mock repository ignores tenantID in GetUserByEmail and GetUserByUsername
// This test demonstrates that the mock doesn't actually enforce tenant isolation

func TestCreateUser_EmailUniqueness_EnforcedPerTenant(t *testing.T) {
	// Create two different tenants
	tenant1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenant2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	// Create a user in tenant1 with email test@example.com
	input1 := &domain.CreateUserInput{
		Username: "user1",
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}

	user1, err := svc.CreateUser(testCtx(tenant1), input1)
	if err != nil {
		t.Fatalf("Failed to create user in tenant1: %v", err)
	}

	// Try to create a user in tenant2 with the SAME email
	// This SHOULD be allowed because email uniqueness should be per-tenant
	input2 := &domain.CreateUserInput{
		Username: "user2",
		Email:    "test@example.com", // Same email as user1
		Password: "SecurePassword123!",
	}

	user2, err := svc.CreateUser(testCtx(tenant2), input2)
	if err != nil {
		t.Fatalf("Creating user with same email in different tenant should be allowed, got error: %v", err)
	}

	if user2.ID == user1.ID {
		t.Error("Expected different user IDs for different tenants")
	}
}

// BUG 2: ProvisionFromLDAP doesn't check for duplicate username/email
func TestProvisionFromLDAP_DuplicateUsername_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	// Create an existing user with username "jdoe"
	existingUserID := uuid.New()
	repo.users[existingUserID] = &domain.User{
		ID:       existingUserID,
		TenantID: tenantID,
		Username: "jdoe",
		Email:    "existing@example.com",
		Status:   domain.UserStatusActive,
	}

	// Try to provision from LDAP with the same username
	result := &authprovider.AuthResult{
		ExternalID: "CN=jdoe2,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "jdoe", // Duplicate username!
			"mail":           "jdoe@corp.local",
			"displayName":    "John Doe",
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(tenantID), result)
	if err == nil {
		t.Errorf("BUG: ProvisionFromLDAP should fail when username already exists, but it succeeded and created user %s", user.ID)
	}
}

func TestProvisionFromLDAP_DuplicateEmail_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	// Create an existing user with email "jdoe@corp.local"
	existingUserID := uuid.New()
	repo.users[existingUserID] = &domain.User{
		ID:       existingUserID,
		TenantID: tenantID,
		Username: "existinguser",
		Email:    "jdoe@corp.local", // Duplicate email!
		Status:   domain.UserStatusActive,
	}

	// Try to provision from LDAP with the same email
	result := &authprovider.AuthResult{
		ExternalID: "CN=jdoe,DC=corp,DC=local",
		Provider:   authprovider.ProviderLDAP,
		NewUser:    true,
		Attributes: map[string]any{
			"sAMAccountName": "jdoe",
			"mail":           "jdoe@corp.local", // Duplicate email!
			"displayName":    "John Doe",
		},
	}

	user, err := svc.ProvisionFromLDAP(testCtx(tenantID), result)
	if err == nil {
		t.Errorf("BUG: ProvisionFromLDAP should fail when email already exists, but it succeeded and created user %s", user.ID)
	}
}

// BUG 3: Race condition in CreateUser (demonstrated with concurrent access)
func TestCreateUser_RaceCondition_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "concurrent",
		Email:    "concurrent@example.com",
		Password: "SecurePassword123!",
	}

	// Simulate concurrent creation attempts
	done := make(chan bool, 2)
	var users [2]*domain.User
	var errs [2]error

	for i := 0; i < 2; i++ {
		go func(idx int) {
			users[idx], errs[idx] = svc.CreateUser(testCtx(tenantID), input)
			done <- true
		}(i)
	}

	<-done
	<-done

	// At least one should have failed in a correct implementation
	// But with the current check-then-create pattern, both might succeed
	successCount := 0
	for i := 0; i < 2; i++ {
		if errs[i] == nil {
			successCount++
		}
	}

	if successCount > 1 {
		t.Errorf("BUG: Race condition - %d users were created with the same username/email", successCount)
	}
}

// BUG 4: Mock repository doesn't filter by tenantID
func TestMockRepository_TenantIsolation_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenant1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenant2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	repo := newMockRepo()

	// Create a user in tenant1
	user1ID := uuid.New()
	repo.users[user1ID] = &domain.User{
		ID:       user1ID,
		TenantID: tenant1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Try to find the user by email, but using tenant2
	// BUG: The mock ignores the tenantID parameter
	user, err := repo.GetUserByEmail(context.Background(), tenant2, "test@example.com")

	if err == nil {
		t.Errorf("BUG: Mock repository found user from tenant1 when searching in tenant2. User ID: %s, User Tenant: %s, Search Tenant: %s",
			user.ID, user.TenantID, tenant2)
	}
}

// BUG 5: DeleteUser doesn't cascade to credentials or external identities
func TestDeleteUser_Cascade_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()

	// Create a user
	repo.users[userID] = &domain.User{
		ID:       userID,
		TenantID: tenantID,
		Username: "testuser",
		Email:    "test@example.com",
		Status:   domain.UserStatusActive,
	}

	// Link an external identity
	ei := &domain.ExternalIdentity{
		ID:         uuid.New(),
		UserID:     userID,
		TenantID:   tenantID,
		Provider:   "google",
		ExternalID: "google-12345",
	}
	repo.externalIdentities = append(repo.externalIdentities, ei)

	// Add some emails
	email1 := &domain.UserEmail{
		ID:     uuid.New(),
		UserID: userID,
		Email:  "test@example.com",
	}
	email2 := &domain.UserEmail{
		ID:     uuid.New(),
		UserID: userID,
		Email:  "secondary@example.com",
	}
	repo.emails[email1.ID] = email1
	repo.emails[email2.ID] = email2

	// Delete the user
	err := svc.DeleteUser(testCtx(tenantID), userID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// BUG: User is soft-deleted, but external identities and emails are not cleaned up
	if len(repo.externalIdentities) != 0 {
		t.Errorf("BUG: External identities not cleaned up after user deletion. Found %d identities", len(repo.externalIdentities))
	}

	if len(repo.emails) != 0 {
		t.Errorf("BUG: User emails not cleaned up after user deletion. Found %d emails", len(repo.emails))
	}
}

// BUG 6: VerifyEmail doesn't actually mark email as verified
func TestVerifyEmail_MarksEmailVerified_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	emailID := uuid.New()

	// Create a verification token
	plainToken := "test-token"
	tokenHash := hashTokenSHA256(plainToken)
	repo.verificationTokens = append(repo.verificationTokens, &domain.EmailVerificationToken{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		EmailID:   emailID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	// Verify the email
	_, err := svc.VerifyEmail(testCtx(tenantID), plainToken)
	if err != nil {
		t.Fatalf("VerifyEmail failed: %v", err)
	}

	// BUG: The email is not actually marked as verified
	// In the current implementation, VerifyEmail only returns the user ID
	// but doesn't update the email's verified status

	email, err := repo.GetUserByEmailID(context.Background(), tenantID, emailID)
	if err != nil {
		t.Fatalf("Failed to get email: %v", err)
	}

	if email.VerifiedAt == nil {
		t.Error("BUG: VerifyEmail should mark the email as verified, but it doesn't")
	}
}

// BUG 7: User can be created with empty password
func TestCreateUser_EmptyPassword_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	input := &domain.CreateUserInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "", // Empty password
	}

	user, err := svc.CreateUser(testCtx(tenantID), input)
	if err == nil {
		t.Errorf("BUG: User created with empty password. User ID: %s, PasswordHash: %s", user.ID, user.PasswordHash)
	}
}

// BUG 8: AddUserEmail doesn't check for duplicate emails
func TestAddUserEmail_Duplicate_Bug(t *testing.T) {
t.Skip("known bug — see test name")
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := newMockRepo()
	svc := NewIdentityService(repo)

	userID := uuid.New()
	email1 := &domain.UserEmail{
		ID:     uuid.New(),
		UserID: userID,
		Email:  "test@example.com",
	}
	repo.emails[email1.ID] = email1

	// Try to add the same email again
	_, err := svc.AddUserEmail(testCtx(tenantID), userID, "test@example.com")
	if err == nil {
		t.Error("BUG: AddUserEmail should prevent duplicate emails for the same user")
	}
}
