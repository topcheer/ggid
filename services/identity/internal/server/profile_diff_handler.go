package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// profileVersion stores a snapshot of a user's profile at a point in time.
type profileVersion struct {
	Version   int               `json:"version"`
	Timestamp string            `json:"timestamp"`
	Fields    map[string]string `json:"fields"`
	ChangedBy string            `json:"changed_by"`
}

var profileVersionStore = struct {
	sync.RWMutex
	data map[string][]profileVersion // userID → versions
}{data: map[string][]profileVersion{
	"00000000-0000-0000-0000-000000000001": {
		{Version: 1, Timestamp: time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339), ChangedBy: "system", Fields: map[string]string{"email": "alice@old.com", "department": "Sales", "role": "viewer", "status": "active"}},
		{Version: 2, Timestamp: time.Now().UTC().Add(-15*24*time.Hour).Format(time.RFC3339), ChangedBy: "hr-admin", Fields: map[string]string{"email": "alice@new.com", "department": "Sales", "role": "viewer", "status": "active"}},
		{Version: 3, Timestamp: time.Now().UTC().Add(-5*24*time.Hour).Format(time.RFC3339), ChangedBy: "sec-admin", Fields: map[string]string{"email": "alice@new.com", "department": "Engineering", "role": "editor", "status": "active"}},
	},
}}

// GET /api/v1/users/{id}/profile-diff?version_a=1&version_b=3
func (h *HTTPHandler) handleProfileDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID
	path := r.URL.Path
	userID := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if dIdx := strings.Index(rest, "/profile-diff"); dIdx >= 0 {
			userID = rest[:dIdx]
		}
	}
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user ID is required in path")
		return
	}

	vaStr := r.URL.Query().Get("version_a")
	vbStr := r.URL.Query().Get("version_b")
	if vaStr == "" {
		vaStr = "1"
	}
	if vbStr == "" {
		vbStr = "latest"
	}

	var va, vb int
	for _, c := range vaStr {
		if c >= '0' && c <= '9' {
			va = va*10 + int(c-'0')
		}
	}

	profileVersionStore.RLock()
	versions := profileVersionStore.data[userID]
	profileVersionStore.RUnlock()

	if len(versions) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":  userID,
			"diffs":    []map[string]any{},
			"message":  "no profile versions recorded",
		})
		return
	}

	if vbStr == "latest" {
		vb = versions[len(versions)-1].Version
	} else {
		for _, c := range vbStr {
			if c >= '0' && c <= '9' {
				vb = vb*10 + int(c-'0')
			}
		}
	}

	var vA, vB *profileVersion
	for i := range versions {
		if versions[i].Version == va {
			vA = &versions[i]
		}
		if versions[i].Version == vb {
			vB = &versions[i]
		}
	}

	if vA == nil || vB == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "version not found"})
		return
	}

	// Compute diff
	diffs := []map[string]any{}
	for field, newVal := range vB.Fields {
		oldVal, exists := vA.Fields[field]
		if !exists {
			diffs = append(diffs, map[string]any{"field": field, "old_value": "", "new_value": newVal, "changed_by": vB.ChangedBy, "changed_at": vB.Timestamp})
		} else if oldVal != newVal {
			diffs = append(diffs, map[string]any{"field": field, "old_value": oldVal, "new_value": newVal, "changed_by": vB.ChangedBy, "changed_at": vB.Timestamp})
		}
	}
	// Check for removed fields
	for field := range vA.Fields {
		if _, exists := vB.Fields[field]; !exists {
			diffs = append(diffs, map[string]any{"field": field, "old_value": vA.Fields[field], "new_value": "", "changed_by": vB.ChangedBy, "changed_at": vB.Timestamp})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":    userID,
		"version_a":  va,
		"version_b":  vb,
		"diffs":      diffs,
		"total_changes": len(diffs),
		"version_a_timestamp": vA.Timestamp,
		"version_b_timestamp": vB.Timestamp,
	})
}
