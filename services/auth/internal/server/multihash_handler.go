package server

import (
	"encoding/json"
	"fmt"
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

// rehashPassword verifies a password against an old-format hash and generates
// a new Argon2id hash if the old format matched. Used by both the verify
// and rehash endpoints.
func rehashPassword(password, oldHash string) (*MultiHashVerifyResponse, error) {
	match, format, err := multihash.VerifyPassword(password, oldHash)
	if err != nil && !match {
		return &MultiHashVerifyResponse{
			Match:       false,
			Format:      format,
			NeedsRehash: multihash.NeedsRehash(oldHash),
		}, nil
	}

	resp := &MultiHashVerifyResponse{
		Match:       match,
		Format:      format,
		NeedsRehash: multihash.NeedsRehash(oldHash),
	}

	// Transparent rehashing: if old format matched, generate new Argon2id hash.
	if match && resp.NeedsRehash {
		newHash, err := crypto.HashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("rehash failed: %w", err)
		}
		resp.Rehashed = newHash
	}

	return resp, nil
}

// handleMultiHashVerify tests a password against a multi-format hash.
// POST /api/v1/auth/multi-hash/verify
func (h *Handler) handleMultiHashVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req MultiHashVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Hash == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "hash and password are required")
		return
	}

	resp, err := rehashPassword(req.Password, req.Hash)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
