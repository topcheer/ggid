package service

import (
	"context"
	"fmt"
	"time"

	gerr "github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// MergeResult records what was transferred during a merge.
type MergeResult struct {
	PrimaryUserID    uuid.UUID
	SecondaryUserID  uuid.UUID
	EmailTransferred bool
	AuditNote        string
	MergedAt         time.Time
}

// MergeUsers merges the secondary user into the primary user.
// The secondary account is soft-deleted after merging profile data.
// The primary user's existing data is preserved unless empty.
func (s *IdentityService) MergeUsers(ctx context.Context, primaryID, secondaryID uuid.UUID) (*MergeResult, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, gerr.New(gerr.ErrFailedPrecondition, "missing tenant context")
	}
	if primaryID == secondaryID {
		return nil, gerr.InvalidArgument("cannot merge user with self")
	}

	primary, err := s.repo.GetUserByID(ctx, tc.TenantID, primaryID)
	if err != nil {
		return nil, gerr.Wrap(gerr.ErrNotFound, "primary user not found", err)
	}
	secondary, err := s.repo.GetUserByID(ctx, tc.TenantID, secondaryID)
	if err != nil {
		return nil, gerr.Wrap(gerr.ErrNotFound, "secondary user not found", err)
	}

	result := &MergeResult{
		PrimaryUserID:   primaryID,
		SecondaryUserID: secondaryID,
		MergedAt:        time.Now().UTC(),
	}

	// Fill missing fields on primary from secondary
	updated := false
	updateInput := &domain.UpdateUserInput{}
	if primary.DisplayName == "" && secondary.DisplayName != "" {
		updateInput.DisplayName = &secondary.DisplayName
		updated = true
	}
	if primary.Phone == "" && secondary.Phone != "" {
		updateInput.Phone = &secondary.Phone
		updated = true
	}

	if updated {
		if _, err := s.repo.UpdateUser(ctx, tc.TenantID, primaryID, updateInput); err != nil {
			return nil, gerr.Wrap(gerr.ErrInternal, "update primary user during merge", err)
		}
	}

	// Soft-delete the secondary user
	if err := s.repo.DeleteUser(ctx, tc.TenantID, secondaryID); err != nil {
		return nil, gerr.Wrap(gerr.ErrInternal, "delete secondary user during merge", err)
	}

	result.AuditNote = fmt.Sprintf("Merged user %s (%s) into %s (%s)",
		secondaryID, secondary.Username, primaryID, primary.Username)

	return result, nil
}
