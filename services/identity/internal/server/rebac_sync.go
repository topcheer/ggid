package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// ReBACSyncRequest controls RBAC→ReBAC tuple sync.
type ReBACSyncRequest struct {
	DryRun bool `json:"dry_run"`
}

// ReBACSyncResult reports sync outcome.
type ReBACSyncResult struct {
	SyncedCount   int             `json:"synced_count"`
	SkippedCount  int             `json:"skipped_count"`
	Errors        []string        `json:"errors,omitempty"`
	DryRun        bool            `json:"dry_run"`
	SampleTuples  []RelationTuple `json:"sample_tuples,omitempty"`
}

// handleReBACSyncRBAC syncs existing RBAC role assignments into ReBAC tuples.
// For each (user, role): tuple(role, role_id, member, user:{user_id})
//
// POST /api/v1/identity/rebac/sync-rbac
func (h *HTTPHandler) handleReBACSyncRBAC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	var req ReBACSyncRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if h.rebacRepo == nil {
		writeJSON(w, http.StatusOK, map[string]any{"relations": []any{}, "count": 0})
		return
	}

	// Framework ready — actual role sync requires policy service DB access.
	// Future: call policy service internal API to get all user-role assignments,
	// then write tuples. For now returns structured empty result.
	slog.Info("ReBAC sync: framework ready, needs policy service integration", "tenant", tc.TenantID, "dry_run", req.DryRun)

	writeJSON(w, http.StatusOK, ReBACSyncResult{
		SyncedCount: 0,
		SkippedCount: 0,
		Errors:      []string{},
		DryRun:      req.DryRun,
	})
}
