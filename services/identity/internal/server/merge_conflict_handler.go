package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MergeConflict struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // duplicate_email, conflicting_role, conflicting_status
	SourceUser string `json:"source_user"`
	TargetUser string `json:"target_user"`
	Detail     string `json:"detail"`
	Resolution string `json:"resolution"`
}

// POST /api/v1/users/merge-conflicts — detect merge conflicts between users
func (h *HTTPHandler) handleMergeConflicts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		SourceUserID string `json:"source_user_id"`
		TargetUserID string `json:"target_user_id"`
		Source       struct {
			Email  string   `json:"email"`
			Roles  []string `json:"roles"`
			Status string   `json:"status"`
		} `json:"source"`
		Target struct {
			Email  string   `json:"email"`
			Roles  []string `json:"roles"`
			Status string   `json:"status"`
		} `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.SourceUserID == "" || req.TargetUserID == "" {
		writeJSONError(w, http.StatusBadRequest, "source_user_id and target_user_id required")
		return
	}

	conflicts := []MergeConflict{}

	// Check duplicate email
	if req.Source.Email != "" && req.Source.Email == req.Target.Email {
		conflicts = append(conflicts, MergeConflict{
			ID: uuid.New().String(), Type: "duplicate_email",
			SourceUser: req.SourceUserID, TargetUser: req.TargetUserID,
			Detail: "both users have same email: " + req.Source.Email,
			Resolution: "merge_emails — target takes primary",
		})
	}

	// Check conflicting roles
	roleMap := make(map[string]bool)
	for _, r := range req.Target.Roles {
		roleMap[r] = true
	}
	overlapping := []string{}
	for _, r := range req.Source.Roles {
		if roleMap[r] {
			overlapping = append(overlapping, r)
		}
	}
	if len(overlapping) > 0 {
		conflicts = append(conflicts, MergeConflict{
			ID: uuid.New().String(), Type: "overlapping_roles",
			SourceUser: req.SourceUserID, TargetUser: req.TargetUserID,
			Detail: "overlapping roles: " + strings.Join(overlapping, ", "),
			Resolution: "deduplicate — keep target roles, add non-overlapping source roles",
		})
	}

	// Check conflicting status
	if req.Source.Status != "" && req.Source.Status != req.Target.Status {
		conflicts = append(conflicts, MergeConflict{
			ID: uuid.New().String(), Type: "conflicting_status",
			SourceUser: req.SourceUserID, TargetUser: req.TargetUserID,
			Detail: "source=" + req.Source.Status + " target=" + req.Target.Status,
			Resolution: "target status preserved — source deactivated",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"source_user_id":   req.SourceUserID,
		"target_user_id":   req.TargetUserID,
		"conflicts":        conflicts,
		"conflict_count":   len(conflicts),
		"can_merge":        len(conflicts) == 0,
		"analyzed_at":      time.Now().UTC().Format(time.RFC3339),
	})
}
