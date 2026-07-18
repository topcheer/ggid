package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
		"github.com/ggid/ggid/services/auth/internal/repository"
)

// aaguidRequest is the DTO for adding an AAGUID.
type aaguidRequest struct {
	AAGUID      string `json:"aaguid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// SetAAGUIDAllowlistRepo injects the DB-backed repository.
func (h *Handler) SetAAGUIDAllowlistRepo(repo *repository.AAGUIDAllowlistRepository) {
	h.aaguidAllowlistRepo = repo
}

// handleAAGUIDAllowlist handles GET/POST/DELETE for /api/v1/auth/webauthn/aaguid.
func (h *Handler) handleAAGUIDAllowlist(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/webauthn/aaguid")

	switch {
	case r.Method == http.MethodGet && (path == "" || path == "/"):
		h.listAAGUIDs(w, r)
	case r.Method == http.MethodPost && (path == "" || path == "/"):
		h.addAAGUID(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/"):
		h.removeAAGUID(w, r, strings.TrimPrefix(path, "/"))
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func (h *Handler) listAAGUIDs(w http.ResponseWriter, r *http.Request) {
	if h.aaguidAllowlistRepo == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	records, err := h.aaguidAllowlistRepo.List(r.Context())
	if err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to list AAGUIDs")
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (h *Handler) addAAGUID(w http.ResponseWriter, r *http.Request) {
	var req aaguidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.AAGUID == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "aaguid is required")
		return
	}
	if req.Name == "" {
		req.Name = req.AAGUID
	}
	if req.Status == "" {
		req.Status = repository.AAGUIDStatusApproved
	}

	rec := &repository.AAGUIDRecord{
		AAGUID:      req.AAGUID,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		AddedBy:     "admin",
	}

	if err := h.aaguidAllowlistRepo.Add(r.Context(), rec); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to add AAGUID")
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (h *Handler) removeAAGUID(w http.ResponseWriter, r *http.Request, aaguid string) {
	if aaguid == "" {
		errors.WriteSimpleAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "aaguid is required")
		return
	}
	if err := h.aaguidAllowlistRepo.Remove(r.Context(), aaguid); err != nil {
		errors.WriteSimpleAPIError(w, http.StatusInternalServerError, "INTERNAL", "failed to remove AAGUID")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

// CheckAAGUIDDuringRegistration is called from passkey registration to verify
// the authenticator's AAGUID is approved. If the allowlist is empty, all
// authenticators are allowed (default open policy).
// Returns true if approved, false otherwise.
func (h *Handler) CheckAAGUIDDuringRegistration(r *http.Request, aaguid string) bool {
	if h.aaguidAllowlistRepo == nil {
		return true // no repo = allow all.
	}
	return h.aaguidAllowlistRepo.IsApproved(r.Context(), aaguid)
}
