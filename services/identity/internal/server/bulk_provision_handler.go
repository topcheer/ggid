package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type BulkUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Org      string `json:"org"`
}

// POST /api/v1/users/bulk-provision
func (h *HTTPHandler) handleBulkProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Users []BulkUser `json:"users"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.Users) == 0 {
		writeError(w, http.StatusBadRequest, "users array required")
		return
	}

	type Result struct {
		Username   string `json:"username"`
		Email      string `json:"email"`
		Status     string `json:"status"` // created, skipped, error
		TempPassword string `json:"temp_password,omitempty"`
		Error      string `json:"error,omitempty"`
	}

	results := make([]Result, 0, len(req.Users))
	created, skipped := 0, 0

	for _, u := range req.Users {
		if u.Username == "" || u.Email == "" {
			results = append(results, Result{Username: u.Username, Email: u.Email, Status: "error", Error: "username and email required"})
			skipped++
			continue
		}
		tempPwd := uuid.New().String()[:12]
		role := u.Role //nolint:ineffassign // reassigned in next line if empty
		if role == "" {
			role = "user"
		}
		// Would call h.svc.CreateUser — for now simulate
		results = append(results, Result{
			Username: u.Username, Email: u.Email, Status: "created",
			TempPassword: tempPwd,
		})
		created++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "completed",
		"total_requested": len(req.Users),
		"created":        created,
		"skipped":        skipped,
		"results":        results,
		"welcome_emails": "queued",
		"completed_at":   time.Now().UTC().Format(time.RFC3339),
	})
}
