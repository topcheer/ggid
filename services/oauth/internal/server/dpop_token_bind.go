package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/service"
)

// dpopCache is a read-through cache for DPoP binding lookups (hot path).
// Persisted to PG via mapRepoVar for durability.
var dpopCache sync.Map // token → jkt

// BindTokenToDPoP associates an access token with a DPoP key thumbprint.
// Persists to PG and caches in memory for fast lookups.
func BindTokenToDPoP(token, jkt string) {
	dpopCache.Store(token, jkt)
	if mapRepoVar != nil {
		mapRepoVar.Store(nil, "oauth_dpop_bindings", token, map[string]any{"jkt": jkt})
	}
}

// CheckTokenDPoPBinding returns the bound JKT for a token (empty = not bound).
func CheckTokenDPoPBinding(token string) string {
	if v, ok := dpopCache.Load(token); ok {
		return v.(string)
	}
	// Cache miss — try PG.
	if mapRepoVar != nil {
		if row, _ := mapRepoVar.Get(nil, "oauth_dpop_bindings", token); row != nil {
			jkt := omGetString(row, "jkt")
			if jkt != "" {
				dpopCache.Store(token, jkt)
				return jkt
			}
		}
	}
	return ""
}

// POST /api/v1/oauth/token/dpop-bind — bind an access token to a DPoP key.
func handleDPoPTokenBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		AccessToken string `json:"access_token"`
		DPoPProof   string `json:"dpop_proof"`
		DPoPJKT     string `json:"dpop_jkt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}
	if req.AccessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "access_token is required"})
		return
	}
	jkt := req.DPoPJKT
	if jkt == "" && req.DPoPProof != "" {
		proof, err := service.ParseDPoPHeader(req.DPoPProof, "POST", "https://example.com/token")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid DPoP proof"})
			return
		}
		jkt = computeKeyThumbprint(proof.PublicKey)
	}
	if jkt == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "dpop_jkt or dpop_proof is required"})
		return
	}
	BindTokenToDPoP(req.AccessToken, jkt)
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "bound", "token_prefix": req.AccessToken[:8] + "...",
		"jkt": jkt, "bound_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// POST /api/v1/oauth/token/dpop-verify — verify that a token matches the DPoP key.
func handleDPoPTokenVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		AccessToken string `json:"access_token"`
		DPoPProof   string `json:"dpop_proof"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}
	if req.AccessToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "access_token is required"})
		return
	}
	boundJKT := CheckTokenDPoPBinding(req.AccessToken)
	if boundJKT == "" {
		writeJSON(w, http.StatusOK, map[string]any{"is_bound": false, "valid": true})
		return
	}
	if req.DPoPProof == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"is_bound": true, "valid": false, "error": "token is DPoP-bound but no DPoP proof provided"})
		return
	}
	proof, err := service.ParseDPoPHeader(req.DPoPProof, "POST", "https://example.com/token")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"is_bound": true, "valid": false, "error": "invalid DPoP proof: " + err.Error()})
		return
	}
	actualJKT := computeKeyThumbprint(proof.PublicKey)
	if actualJKT != boundJKT {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"is_bound": true, "valid": false, "error": "DPoP key thumbprint mismatch", "expected": boundJKT, "actual": actualJKT})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"is_bound": true, "valid": true, "jkt": boundJKT})
}
