package server

import (
	"encoding/json"
	"net/http"

	"github.com/ggid/ggid/pkg/auth/multihash"
	"github.com/ggid/ggid/pkg/crypto"
)

// MultiHashVerifyRequest is the body for POST /api/v1/auth/multi-hash/verify.
type MultiHashVerifyRequest struct {
	Hash     string `json:"hash"`
	Password string `json:"password"`
}

// MultiHashVerifyResponse contains verification result and rehash info.
type MultiHashVerifyResponse struct {
	Match       bool   `json:"match"`
	Format      string `json:"format"`
	NeedsRehash bool   `json:"needs_rehash"`
	Rehashed    string `json:"rehashed,omitempty"` // new Argon2id hash if rehashing was performed
}

// handleMultiHashVerify tests a password against a multi-format hash.
// POST /api/v1/auth/multi-hash/verify
//
// This endpoint is primarily for testing/migration validation.
// It auto-detects the hash format and verifies the password.
// If the hash is in a legacy format and matches, it returns a new Argon2id hash.
func (h *Handler) handleMultiHashVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req MultiHashVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Hash == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "hash and password are required")
		return
	}

	// Verify using multihash verifier.
	match, format, err := multihash.VerifyPassword(req.Password, req.Hash)
	if err != nil && !match {
		writeJSON(w, http.StatusOK, MultiHashVerifyResponse{
			Match:       false,
			Format:      format,
			NeedsRehash: multihash.NeedsRehash(req.Hash),
		})
		return
	}

	resp := MultiHashVerifyResponse{
		Match:       match,
		Format:      format,
		NeedsRehash: multihash.NeedsRehash(req.Hash),
	}

	// Transparent rehashing: if old format matched, generate new Argon2id hash.
	if match && resp.NeedsRehash {
		newHash, err := crypto.HashPassword(req.Password)
		if err == nil {
			resp.Rehashed = newHash
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
