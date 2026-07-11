package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

// DelegationEntry tracks one link in a delegation chain.
type DelegationEntry struct {
	Actor    string `json:"actor"`
	Subject  string `json:"subject"`
	Scope    string `json:"scope"`
	Reason   string `json:"reason,omitempty"`
}

type delegationChainStore struct {
	mu     sync.RWMutex
	chains map[string][]DelegationEntry // token_id → chain
}

var delegationChains = &delegationChainStore{chains: make(map[string][]DelegationEntry)}

// POST /api/v1/oauth/token-exchange-delegation — RFC 8693 extension with delegation_chain.
// Body: {"subject_token": "...", "actor_token": "...", "scope": "...", "reason": "..."}
// Returns: access token with act claims representing the delegation chain.
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
		Audience     string `json:"audience"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}
	if req.SubjectToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "subject_token is required"})
		return
	}
	if req.ActorToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "actor_token is required"})
		return
	}

	// Build delegation chain entry
	entry := DelegationEntry{
		Actor:   req.ActorToken,
		Subject: req.SubjectToken,
		Scope:   req.Scope,
		Reason:  req.Reason,
	}

	// Generate delegated access token
	tokenID := uuid.New().String()
	if req.Scope == "" {
		req.Scope = "default"
	}

	// Store chain
	delegationChains.mu.Lock()
	// Check for existing chain from subject token
	existing := delegationChains.chains[req.SubjectToken]
	chain := append([]DelegationEntry{entry}, existing...)
	delegationChains.chains[tokenID] = chain
	delegationChains.mu.Unlock()

	// Build act claims for the response
	actClaims := make([]map[string]any, len(chain))
	for i, e := range chain {
		actClaims[i] = map[string]any{
			"act":     e.Actor,
			"sub":     e.Subject,
			"scope":   e.Scope,
			"reason":  e.Reason,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":      "delegated_" + tokenID,
		"token_type":        "Bearer",
		"expires_in":        3600,
		"scope":             req.Scope,
		"audience":          req.Audience,
		"delegation_chain":  chain,
		"act_claims":        actClaims,
		"chain_depth":       len(chain),
		"issued_at":         time.Now().UTC().Format(time.RFC3339),
	})
}

// VerifyDelegationChain validates a delegation chain for a token.
func VerifyDelegationChain(tokenID string) ([]DelegationEntry, bool) {
	delegationChains.mu.RLock()
	defer delegationChains.mu.RUnlock()
	chain, ok := delegationChains.chains[tokenID]
	return chain, ok
}

// Ensure service package is referenced (for potential future DPoP integration)
var _ = service.ParseDPoPHeader
