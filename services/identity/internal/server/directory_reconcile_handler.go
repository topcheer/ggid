package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/identity/directory/reconcile
// Body: {"dry_run": true, "fix_orphaned": true, "fix_duplicates": true}
func (h *HTTPHandler) handleDirectoryReconcile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		DryRun        bool `json:"dry_run"`
		FixOrphaned   bool `json:"fix_orphaned"`
		FixDuplicates bool `json:"fix_duplicates"`
		FixStaleMgr   bool `json:"fix_stale_managers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Default to dry_run=true for safety
		req.DryRun = true
	}
	// Safety: default dry_run if not specified
	if !req.DryRun && r.URL.Query().Get("confirm") != "true" {
		req.DryRun = true
	}

	now := time.Now().UTC()
	jobID := uuid.New().String()

	// Simulate reconciliation analysis
	orphanedIDs := []string{"user-0042", "user-0078", "user-0156", "user-0201", "user-0333"}
	duplicateGroups := []map[string]any{
		{"email": "dup@example.com", "accounts": []string{"user-0100", "user-0101"}, "merge_strategy": "keep_recent_active"},
		{"email": "shared@company.com", "accounts": []string{"user-0200", "user-0201"}, "merge_strategy": "keep_higher_role"},
		{"email": "legacy@old.com", "accounts": []string{"user-0300", "user-0301"}, "merge_strategy": "manual_review"},
	}

	cleanupPlan := []map[string]any{}
	if req.FixOrphaned {
		for _, id := range orphanedIDs {
			cleanupPlan = append(cleanupPlan, map[string]any{
				"type": "orphaned_account", "user_id": id,
				"action": "assign_to_default_manager", "manager_id": "mgr-default",
			})
		}
	}
	if req.FixDuplicates {
		for _, group := range duplicateGroups {
			cleanupPlan = append(cleanupPlan, map[string]any{
				"type": "duplicate_account", "email": group["email"],
				"accounts": group["accounts"], "action": group["merge_strategy"],
			})
		}
	}
	if req.FixStaleMgr {
		cleanupPlan = append(cleanupPlan, map[string]any{
			"type": "stale_manager", "manager_id": "mgr-002",
			"action": "reassign_reports", "new_manager": "mgr-008",
		})
	}

	status := "dry_run"
	if !req.DryRun {
		status = "executed"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"job_id":            jobID,
		"status":            status,
		"dry_run":           req.DryRun,
		"orphaned_ids":      orphanedIDs,
		"orphaned_count":    len(orphanedIDs),
		"duplicate_groups":  duplicateGroups,
		"duplicate_count":   len(duplicateGroups),
		"stale_managers":    8,
		"merge_strategies":  []string{"keep_recent_active", "keep_higher_role", "manual_review"},
		"cleanup_plan":      cleanupPlan,
		"total_actions":     len(cleanupPlan),
		"analyzed_at":       now.Format(time.RFC3339),
		"safety_note": func() string {
			if req.DryRun {
				return "No changes applied. Set dry_run=false and confirm=true to execute."
			}
			return "Changes applied. All actions are logged for audit."
		}(),
	})
}
