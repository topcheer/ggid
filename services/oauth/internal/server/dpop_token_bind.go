package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/service"
)

// dpopBindingStore tracks DPoP-bound tokens: token_hash → jkt
type dpopBindingStore struct {
	mu     sync.RWMutex
	binds  map[string]string // access_token → jkt (DPoP key thumbprint)
}

var dpopBindings = &dpopBindingStore{binds: make(map[string]string)}

// BindTokenToDPoP associates an access token with a DPoP key thumbprint.
func BindTokenToDPoP(token, jkt string) {
	dpopBindings.mu.Lock()
	dpopBindings.binds[token] = jkt
	dpopBindings.mu.Unlock()
}

// CheckTokenDPoPBinding returns the bound JKT for a token (empty = not bound).
func CheckTokenDPoPBinding(token string) string {
	dpopBindings.mu.RLock()
	defer dpopBindings.mu.RUnlock()
	return dpopBindings.binds[token]
}

// POST /api/v1/oauth/token/dpop-bind — bind an access token to a DPoP key.
// Body: {"access_token": "...", "dpop_proof": "..."}
// This is called after token issuance when DPoP is used.
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
		// Compute JKT from proof
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
		"status":       "bound",
		"token_prefix": req.AccessToken[:8] + "...",
		"jkt":          jkt,
		"bound_at":     time.Now().UTC().Format(time.RFC3339),
	})
}

// POST /api/v1/oauth/token/dpop-verify — verify that a token matches the DPoP key.
// Returns 401 if the token is bound but the DPoP key doesn't match.
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
		writeJSON(w, http.StatusOK, map[string]any{
			"is_bound": false,
			"valid":    true,
		})
		return
	}

	if req.DPoPProof == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"is_bound": true,
			"valid":    false,
			"error":    "token is DPoP-bound but no DPoP proof provided",
		})
		return
	}

	proof, err := service.ParseDPoPHeader(req.DPoPProof, "POST", "https://example.com/token")
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"is_bound": true,
			"valid":    false,
			"error":    "invalid DPoP proof: " + err.Error(),
		})
		return
	}

	actualJKT := computeKeyThumbprint(proof.PublicKey)
	if actualJKT != boundJKT {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"is_bound":  true,
			"valid":     false,
			"error":     "DPoP key thumbprint mismatch",
			"expected":  boundJKT,
			"actual":    actualJKT,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"is_bound": true,
		"valid":    true,
		"jkt":      boundJKT,
	})
}
