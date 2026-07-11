package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type PolicyConflict struct {
	ID            string `json:"id"`
	Resource      string `json:"resource"`
	Action        string `json:"action"`
	Principal     string `json:"principal"`
	AllowPolicy   string `json:"allow_policy"`
	DenyPolicy    string `json:"deny_policy"`
	Resolution    string `json:"resolution"` // deny_wins, allow_wins, manual_review
	Description   string `json:"description"`
}

// POST /api/v1/policies/conflicts/resolve
func (s *HTTPServer) handleConflictResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Policies []struct {
			ID        string `json:"id"`
			Effect    string `json:"effect"` // allow, deny
			Resource  string `json:"resource"`
			Action    string `json:"action"`
			Principal string `json:"principal"`
			Priority  int    `json:"priority"`
		} `json:"policies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Detect allow+deny conflicts on same resource+action+principal
	conflicts := []PolicyConflict{}
	for i, a := range req.Policies {
		for _, b := range req.Policies[i+1:] {
			if a.Resource == b.Resource && a.Action == b.Action && a.Principal == b.Principal && a.Effect != b.Effect {
				allowPol, denyPol := a.ID, b.ID
				if a.Effect == "deny" {
					allowPol, denyPol = b.ID, a.ID
				}
				// Default resolution: deny wins (least privilege)
				resolution := "deny_wins"
				if a.Priority > 0 || b.Priority > 0 {
					// Higher priority wins
					if a.Priority > b.Priority && a.Effect == "allow" {
						resolution = "allow_wins"
					}
				}
				conflicts = append(conflicts, PolicyConflict{
					ID: uuid.New().String(), Resource: a.Resource, Action: a.Action,
					Principal: a.Principal, AllowPolicy: allowPol, DenyPolicy: denyPol,
					Resolution: resolution,
					Description: "conflicting allow/deny on " + a.Resource + ":" + a.Action,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"conflicts":           conflicts,
		"conflict_count":      len(conflicts),
		"policies_analyzed":   len(req.Policies),
		"default_resolution":  "deny_wins",
		"analyzed_at":         time.Now().UTC().Format(time.RFC3339),
	})
}
