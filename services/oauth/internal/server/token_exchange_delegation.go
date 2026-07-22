package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

// DelegationEntry tracks one link in a delegation chain.
type DelegationEntry struct {
	Actor   string `json:"actor"`
	Subject string `json:"subject"`
	Scope   string `json:"scope"`
	Reason  string `json:"reason,omitempty"`
}

// POST /api/v1/oauth/token-exchange-delegation — RFC 8693 extension with delegation_chain.
func handleTokenExchangeDelegation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		SubjectToken string `json:"subject_token"`
		ActorToken   string `json:"actor_token"`
		Scope        string `json:"scope"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
		return
	}
	if req.SubjectToken == "" || req.ActorToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "subject_token and actor_token are required"})
		return
	}

	// Build delegation chain entry.
	entry := DelegationEntry{
		Actor:   "actor:" + req.ActorToken[:8],
		Subject: "subject:" + req.SubjectToken[:8],
		Scope:   req.Scope,
		Reason:  req.Reason,
	}

	// Persist delegation chain to PG.
	chainID := uuid.New().String()
	if mapRepoVar != nil {
		mapRepoVar.Store(r.Context(), "oauth_delegation_chains", chainID, map[string]any{
			"actor": entry.Actor, "subject": entry.Subject,
			"scope": entry.Scope, "reason": entry.Reason,
			"created_at": time.Now().UTC(),
		})
	}

	// Build act claim for the token.
	actClaim := map[string]any{
		"sub": entry.Actor,
	}

	// Return a simulated token response with delegation info.
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":      "", // no dev tokens in production code,
		"token_type":        "Bearer",
		"expires_in":        3600,
		"scope":             req.Scope,
		"delegation_chain":  []DelegationEntry{entry},
		"act":               actClaim,
		"chain_id":          chainID,
	})

	_ = service.OAuthService{} // suppress unused import
}
