package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// MultiHashRehashRequest is the body for POST /api/v1/auth/multi-hash/rehash/:user_id.
type MultiHashRehashRequest struct {
	Password string `json:"password"` // plaintext password for rehashing
	OldHash  string `json:"old_hash"` // current hash (optional; if omitted, lookup by user_id)
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

	// SECURITY: Always look up the user's actual DB hash. Never accept
	// client-supplied old_hash — that would create a password verification
	// oracle (attacker tests arbitrary hashes against a known password).
	// The old_hash field is accepted for API compatibility but must match
	// the DB hash to proceed.
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusForbidden, "tenant context required")
		return
	}

	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var dbHash string
	err = h.pool.QueryRow(r.Context(),
		`SELECT password_hash FROM auth_credentials WHERE user_id = $1 AND tenant_id = $2`,
		userID, tenantID).Scan(&dbHash)
	if err != nil {
		writeError(w, http.StatusNotFound, "user credential not found")
		return
	}

	// If old_hash provided, verify it matches the DB hash (anti-oracle).
	if req.OldHash != "" && req.OldHash != dbHash {
		writeError(w, http.StatusBadRequest, "provided old_hash does not match stored hash")
		return
	}

	returnRehashResult(w, req.Password, dbHash)
}

// returnRehashResult verifies password against oldHash and returns the rehashed result.
func returnRehashResult(w http.ResponseWriter, password, oldHash string) {
	resp, err := rehashPassword(password, oldHash)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
