package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

func TestMergeUsers_Success(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)
	ctx := testCtx(uuid.New())

	primary, _ := svc.CreateUser(ctx, &domain.CreateUserInput{
		Username: "primary-user", Email: "primary@test.com", Password: "Pass123!",
	})
	secondary, _ := svc.CreateUser(ctx, &domain.CreateUserInput{
		Username: "secondary-user", Email: "secondary@test.com", Password: "Pass456!",
	})

	result, err := svc.MergeUsers(ctx, primary.ID, secondary.ID)
	if err != nil {
		t.Fatalf("MergeUsers: %v", err)
	}
	if result.PrimaryUserID != primary.ID {
		t.Error("primary ID mismatch")
	}
	if result.SecondaryUserID != secondary.ID {
		t.Error("secondary ID mismatch")
	}
	if result.AuditNote == "" {
		t.Error("audit note should not be empty")
	}
}

func TestMergeUsers_SameUser(t *testing.T) {
	svc := NewIdentityService(newMockRepo())
	ctx := testCtx(uuid.New())

	id := uuid.New()
	_, err := svc.MergeUsers(ctx, id, id)
	if err == nil {
		t.Error("should reject merging user with self")
	}
}

func TestMergeUsers_NoTenant(t *testing.T) {
	svc := NewIdentityService(newMockRepo())
	_, err := svc.MergeUsers(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("should require tenant context")
	}
}

func TestMergeUsers_PrimaryNotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)
	ctx := testCtx(uuid.New())

	secondary, _ := svc.CreateUser(ctx, &domain.CreateUserInput{
		Username: "secondary", Email: "sec@test.com", Password: "Pass123!",
	})

	_, err := svc.MergeUsers(ctx, uuid.New(), secondary.ID)
	if err == nil {
		t.Error("should fail when primary not found")
	}
}

func TestMergeUsers_SecondaryNotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewIdentityService(repo)
	ctx := testCtx(uuid.New())

	primary, _ := svc.CreateUser(ctx, &domain.CreateUserInput{
		Username: "primary", Email: "p@test.com", Password: "Pass123!",
	})

	_, err := svc.MergeUsers(ctx, primary.ID, uuid.New())
	if err == nil {
		t.Error("should fail when secondary not found")
	}
}
