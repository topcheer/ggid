package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type QuarantineRequest struct {
	Reason          string   `json:"reason"`
	DurationHours   int      `json:"duration_hours"`
	AffectedEntities []string `json:"affected_entities"`
}

type QuarantineResult struct {
	PolicyID        string    `json:"policy_id"`
	Status          string    `json:"status"`
	Reason          string    `json:"reason"`
	QuarantinedAt   string    `json:"quarantined_at"`
	AutoReenableAt  string    `json:"auto_reenable_at"`
	AffectedEntities []string  `json:"affected_entities"`
	RollbackPlan    string    `json:"rollback_plan"`
	DurationHours   int       `json:"duration_hours"`
}

var quarantineStore sync.Map

func (s *HTTPServer) handlePolicyQuarantine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract policy ID from path /api/v1/policy/{id}/quarantine
	path := r.URL.Path
	idx := lastNth(path, "/", 2)
	policyID := ""
	if idx > 0 {
		policyID = path[idx+1 : lastNth(path, "/", 1)]
	}
	if policyID == "" {
		policyID = "unknown"
	}

	var req QuarantineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req = QuarantineRequest{Reason: "manual quarantine", DurationHours: 24}
	}
	if req.DurationHours == 0 {
		req.DurationHours = 24
	}
	if req.Reason == "" {
		req.Reason = "security hold"
	}

	now := time.Now().UTC()
	result := QuarantineResult{
		PolicyID:         policyID,
		Status:           "quarantined",
		Reason:           req.Reason,
		QuarantinedAt:    now.Format(time.RFC3339),
		AutoReenableAt:   now.Add(time.Duration(req.DurationHours) * time.Hour).Format(time.RFC3339),
		AffectedEntities: req.AffectedEntities,
		RollbackPlan:     fmt.Sprintf("Re-enable policy %s after %dh or manually via /api/v1/policy/%s/quarantine/remove", policyID, req.DurationHours, policyID),
		DurationHours:    req.DurationHours,
	}
	quarantineStore.Store(policyID, result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// lastNth returns the byte index of the nth-to-last "/" in s, or -1 if not found.
// lastNth(path, "/", 1) returns position of last "/".
// lastNth(path, "/", 2) returns position of second-to-last "/".
func lastNth(s, sep string, n int) int {
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}
