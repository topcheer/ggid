package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
)

// POST /api/v1/users/jit-provision
func (h *HTTPHandler) handleJITProvision(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Provider    string         `json:"provider"`
		ExternalID  string         `json:"external_id"`
		Email       string         `json:"email"`
		Username    string         `json:"username"`
		DisplayName string         `json:"display_name"`
		Attributes  map[string]any `json:"attributes"`
		TenantID    string         `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Provider == "" || req.ExternalID == "" {
		writeError(w, http.StatusBadRequest, "provider and external_id are required")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	tenantID := uuid.Nil
	if req.TenantID != "" {
		tenantID, _ = uuid.Parse(req.TenantID)
	}
	if req.Username == "" {
		req.Username = req.Email
	}

	// Check existing by email
	searchFilter := &domain.ListUsersFilter{Search: req.Email}
	result, err := h.svc.ListUsers(ctx, searchFilter)
	if err == nil {
		for _, u := range result.Users {
			if u.Email == req.Email {
				writeJSON(w, http.StatusOK, map[string]any{
					"status":   "existing",
					"user_id":  u.ID.String(),
					"username": u.Username,
					"email":    u.Email,
					"provider": req.Provider,
					"linked":   true,
				})
				return
			}
		}
	}

	// Create new user
	input := &domain.CreateUserInput{
		TenantID:    tenantID,
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
	}

	user, err := h.svc.CreateUser(ctx, input)
	if err != nil {
		log.Printf("jit provision error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to provision user")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":         "created",
		"user_id":        user.ID.String(),
		"username":       user.Username,
		"email":          user.Email,
		"display_name":   user.DisplayName,
		"provider":       req.Provider,
		"external_id":    req.ExternalID,
		"email_verified": true,
		"user_status":    string(user.Status),
		"provisioned_at": time.Now().UTC().Format(time.RFC3339),
	})
}
