package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/audit"
	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// handleDelegations is the full CRUD handler for delegations.
// GET    /api/v1/auth/delegations       — list current user's delegations
// POST   /api/v1/auth/delegations       — create delegation
// DELETE /api/v1/auth/delegations/:id   — revoke delegation
// GET    /api/v1/auth/delegations/check — check if delegation is valid
func (h *Handler) handleDelegations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/delegations")
	path = strings.TrimPrefix(path, "/")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			h.delegationList(w, r)
		case http.MethodPost:
			h.delegationCreate(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if path == "check" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.delegationCheck(w, r)
		return
	}

	// It's a :id path.
	if r.Method == http.MethodDelete {
		h.delegationRevoke(w, r, path)
		return
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (h *Handler) delegationList(w http.ResponseWriter, r *http.Request) {
	if h.delRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}
	userIDStr := r.Header.Get("X-User-ID")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid X-User-ID header required")
		return
	}
	delegations, err := h.delRepo.ListByUser(r.Context(), tc.TenantID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list delegations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"delegations": delegations,
		"count":       len(delegations),
	})
}

func (h *Handler) delegationCreate(w http.ResponseWriter, r *http.Request) {
	if h.delRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	var req struct {
		DelegateeID string   `json:"delegatee_id"`
		Scopes      []string `json:"scopes"`
		ResourceID  string   `json:"resource_id"`
		ExpiresIn   int      `json:"expires_in_hours"` // convenience: hours from now
		ExpiresAt   string   `json:"expires_at"`       // ISO 8601
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	delegatorID := r.Header.Get("X-User-ID")
	if delegatorID == "" {
		writeError(w, http.StatusBadRequest, "X-User-ID header required")
		return
	}

	// Parse expiry.
	var expiresAt time.Time
	if req.ExpiresAt != "" {
		expiresAt, err = time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid expires_at (use RFC3339)")
			return
		}
	} else if req.ExpiresIn > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Hour)
	} else {
		writeError(w, http.StatusBadRequest, "expires_at or expires_in_hours required")
		return
	}

	d := &UserDelegation{
		TenantID:    tc.TenantID.String(),
		DelegatorID: delegatorID,
		DelegateeID: req.DelegateeID,
		Scopes:      req.Scopes,
		ResourceID:  req.ResourceID,
		ExpiresAt:   expiresAt,
	}
	if err := ValidateDelegation(d); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.delRepo.Create(r.Context(), d); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create delegation")
		return
	}

	// Audit event.
	h.publishAuditEvent("delegation.create", "success", tc.TenantID, uuid.MustParse(delegatorID))

	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) delegationRevoke(w http.ResponseWriter, r *http.Request, id string) {
	if h.delRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	if err := h.delRepo.Revoke(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke delegation")
		return
	}

	// Audit event.
	if tc, err := ggidtenant.FromContext(r.Context()); err == nil {
		userIDStr := r.Header.Get("X-User-ID")
		if uid, e := uuid.Parse(userIDStr); e == nil {
			h.publishAuditEvent("delegation.revoke", "success", tc.TenantID, uid)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked", "id": id})
}

func (h *Handler) delegationCheck(w http.ResponseWriter, r *http.Request) {
	if h.delRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{}); return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	delegatorID := r.URL.Query().Get("delegator_id")
	delegateeID := r.URL.Query().Get("delegatee_id")
	scope := r.URL.Query().Get("scope")

	if delegatorID == "" || delegateeID == "" || scope == "" {
		writeError(w, http.StatusBadRequest, "delegator_id, delegatee_id, and scope query params required")
		return
	}

	degID, err := uuid.Parse(delegatorID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid delegator_id")
		return
	}
	deeID, err := uuid.Parse(delegateeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid delegatee_id")
		return
	}

	valid, del := h.delRepo.CheckDelegation(r.Context(), tc.TenantID, degID, deeID, scope)
	if valid && del != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"valid":      true,
			"delegation": del,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"valid": false,
	})
}

// Ensure audit import is used.
var _ = audit.NewEvent
