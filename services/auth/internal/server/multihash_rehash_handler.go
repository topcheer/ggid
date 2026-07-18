package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// MultiHashRehashRequest is the body for POST /api/v1/auth/multi-hash/rehash/:user_id.
type MultiHashRehashRequest struct {
	Password string `json:"password"` // plaintext password for rehashing
	OldHash  string `json:"old_hash"`  // current hash (optional; if omitted, lookup by user_id)
}

// handleMultiHashRehash manually triggers rehashing of a user's password hash.
// POST /api/v1/auth/multi-hash/rehash/:user_id
//
// This endpoint is for administrators to manually trigger rehashing
// during migration. It verifies the password against the old hash,
// then generates and returns a new Argon2id hash.
func (h *Handler) handleMultiHashRehash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user_id from path.
	userIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/multi-hash/rehash/")
	if userIDStr == "" || strings.Contains(userIDStr, "/") {
		writeError(w, http.StatusBadRequest, "valid user_id required in path")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid user_id required")
		return
	}

	var req MultiHashRehashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	// If old_hash is provided, verify against it directly.
	if req.OldHash != "" {
		// Import multihash inline to avoid circular dependency at package level.
		returnRehashResult(w, req.Password, req.OldHash)
		return
	}

	// Look up user's credential to get current hash.
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid X-Tenant-ID header required")
		return
	}

	cred, err := h.authSvc.LookupCredential(r.Context(), tenantID, userID)
	if err != nil || cred == nil {
		slog.Warn("rehash: credential lookup failed",
			"user_id", userID, "error", err)
		writeError(w, http.StatusNotFound, "credential not found")
		return
	}

	returnRehashResult(w, req.Password, cred.Secret)
}

// returnRehashResult verifies password against oldHash and returns the rehashed result.
func returnRehashResult(w http.ResponseWriter, password, oldHash string) {
	resp, err := rehashPassword(password, oldHash)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
