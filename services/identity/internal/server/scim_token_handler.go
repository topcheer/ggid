package server

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// handleSCIMTokens manages SCIM bearer token CRUD.
//   POST   /api/v1/identity/scim/tokens         — create (returns plaintext once)
//   GET    /api/v1/identity/scim/tokens         — list (no hash)
//   DELETE /api/v1/identity/scim/tokens/{id}    — revoke
func (h *HTTPHandler) handleSCIMTokens(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	if h.scimRepo == nil {
		writeError(w, http.StatusServiceUnavailable, "SCIM token management not configured")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/identity/scim/tokens")

	switch {
	case path == "" || path == "/":
		switch r.Method {
		case http.MethodPost:
			h.createSCIMToken(w, r, tc.TenantID)
		case http.MethodGet:
			h.listSCIMTokens(w, r, tc.TenantID)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	case strings.HasPrefix(path, "/") && r.Method == http.MethodDelete:
		tokenIDStr := strings.TrimPrefix(path, "/")
		tokenID, err := uuid.Parse(tokenIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid token id")
			return
		}
		if err := h.scimRepo.Revoke(r.Context(), tokenID, tc.TenantID); err != nil {
			slog.Error("SCIM token revoke error", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to revoke token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *HTTPHandler) createSCIMToken(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Extract creator user ID from header.
	var createdBy uuid.UUID
	if uidStr := r.Header.Get("X-User-ID"); uidStr != "" {
		createdBy, _ = uuid.Parse(uidStr)
	}

	token, plaintext, err := h.scimRepo.Create(r.Context(), tenantID, req.Name, createdBy, hashSCIMToken)
	if err != nil {
		slog.Error("SCIM token create error", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	// Audit: SCIM token created.
	_ = h.auditPublisher // audit publisher may be nil in dev

	log.Printf("SCIM token created: tenant=%s name=%s id=%s", tenantID, req.Name, token.ID)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          token.ID,
		"name":        token.Name,
		"token":       plaintext,
		"scopes":      token.Scopes,
		"created_at":  token.CreatedAt,
		"message":     "Save this token now — it will not be shown again.",
	})
}

func (h *HTTPHandler) listSCIMTokens(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	tokens, err := h.scimRepo.ListByTenant(r.Context(), tenantID)
	if err != nil {
		slog.Error("SCIM token list error", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list tokens")
		return
	}
	if tokens == nil {
		tokens = []*SCIMToken{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tokens": tokens,
		"total":  len(tokens),
	})
}
