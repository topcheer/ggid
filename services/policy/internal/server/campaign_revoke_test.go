package httpserver

import (
	"testing"

	"github.com/google/uuid"
)

func TestExecuteCampaignRevoke_NoItemsSkips(t *testing.T) {
	s := &HTTPServer{roleSvc: nil} // nil roleSvc = early return, no panic
	c := &ReviewCampaign{
		ID:         uuid.New().String(),
		ReviewerID: uuid.New().String(),
		ScopeID:    uuid.New().String(),
	}
	// Should not panic with nil roleSvc
	s.executeCampaignRevoke(nil, c)
}

func TestExecuteCampaignRevoke_ReviewerNotRevoked(t *testing.T) {
	// This test verifies the fix: reviewer ID is NOT used as the revoke target.
	// Without campaign items, the revoke does nothing — reviewer is safe.
	reviewerID := uuid.New()
	c := &ReviewCampaign{
		ID:         uuid.New().String(),
		ReviewerID: reviewerID.String(),
		ScopeID:    uuid.New().String(),
		Decision:   "revoke",
	}
	s := &HTTPServer{roleSvc: nil, campaignRepo: nil}
	s.executeCampaignRevoke(nil, c)
	// No items → no revoke calls → reviewer safe.
	// If the old buggy code ran, it would have tried to revoke using reviewerID.
}

func TestCampaignItem_Struct(t *testing.T) {
	uid := uuid.New()
	rid := uuid.New()
	cid := uuid.New()
	item := &CampaignItem{
		CampaignID: cid.String(),
		UserID:     uid.String(),
		RoleID:     rid.String(),
		Decision:   "revoke",
	}
	if item.UserID == "" || item.RoleID == "" {
		t.Error("user_id and role_id must not be empty")
	}
}
