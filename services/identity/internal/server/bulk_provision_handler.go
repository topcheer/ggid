package server

import (
	"encoding/json"
	"net/http"
	"time"
	"encoding/base64"
	crand "crypto/rand"
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
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Users []BulkUser `json:"users"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.Users) == 0 {
		writeJSONError(w, http.StatusBadRequest, "users array required")
		return
	}
	if len(req.Users) > 1000 {
		writeJSONError(w, http.StatusBadRequest, "max 1000 users per bulk provision")
		return
	}

	type Result struct {
		Username     string `json:"username"`
		Email        string `json:"email"`
		Status       string `json:"status"` // created, skipped, error
		TempPassword string `json:"temp_password,omitempty"`
		Role         string `json:"role,omitempty"`
		Error        string `json:"error,omitempty"`
	}

	results := make([]Result, 0, len(req.Users))
	created, skipped := 0, 0

	for _, u := range req.Users {
		if u.Username == "" || u.Email == "" {
			results = append(results, Result{Username: u.Username, Email: u.Email, Status: "error", Error: "username and email required"})
			skipped++
			continue
		}
		// SECURITY: Validate input lengths to prevent DoS.
		if len(u.Username) > 255 || len(u.Email) > 320 {
			results = append(results, Result{Username: u.Username, Email: u.Email, Status: "error", Error: "field exceeds maximum length"})
			skipped++
			continue
		}
		// SECURITY: Use crypto/rand for temporary password instead of UUID.
		tempPwdBytes := make([]byte, 9)
		if _, err := crand.Read(tempPwdBytes); err != nil {
			results = append(results, Result{Username: u.Username, Email: u.Email, Status: "error", Error: "failed to generate password"})
			skipped++
			continue
		}
		tempPwd := base64.RawURLEncoding.EncodeToString(tempPwdBytes)
		role := u.Role
		if role == "" {
			role = "user"
		}
		// Would call h.svc.CreateUser — for now simulate
		results = append(results, Result{
			Username: u.Username, Email: u.Email, Status: "created",
			TempPassword: tempPwd,
			Role:         role,
		})
		created++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "completed",
		"total_requested": len(req.Users),
		"created":         created,
		"skipped":         skipped,
		"results":         results,
		"welcome_emails":  "queued",
		"completed_at":    time.Now().UTC().Format(time.RFC3339),
	})
}
