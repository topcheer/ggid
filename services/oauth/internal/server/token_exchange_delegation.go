package server

import (
	"encoding/json"
	"net/http"
	"strings"
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

// scopeClaimSet extracts the scope claim from JWT claims as a set.
func scopeClaimSet(claims map[string]any) map[string]bool {
	set := map[string]bool{}
	if s, ok := claims["scope"].(string); ok {
		for _, f := range strings.Fields(s) {
			set[f] = true
		}
	}
	return set
}

// POST /api/v1/oauth/token-exchange-delegation — RFC 8693 extension with delegation_chain.
//
// SECURITY (R5 P0): both tokens are signature-validated and the requested
// scope must be a subset of BOTH the subject token's scope and (when the
// actor token carries a scope) the actor token's scope. Previously any
// string was accepted as a token and req.Scope was echoed verbatim,
// allowing scope escalation claims in persisted delegation chains.
func handleTokenExchangeDelegation(svc *service.OAuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Validate both tokens (signature + expiry).
		subjectClaims, err := svc.ParseAccessToken(req.SubjectToken)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid subject_token"})
			return
		}
		actorClaims, err := svc.ParseAccessToken(req.ActorToken)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid actor_token"})
			return
		}

		// Scope narrowing (RFC 8693): requested ⊆ subject, and ⊆ actor when
		// the actor token carries a scope claim.
		subjectScopes := scopeClaimSet(subjectClaims)
		actorScopes := scopeClaimSet(actorClaims)
		requested := strings.Fields(req.Scope)
		if len(requested) == 0 {
			// Inherit the intersection of subject and actor scopes.
			for sc := range subjectScopes {
				if len(actorScopes) == 0 || actorScopes[sc] {
					requested = append(requested, sc)
				}
			}
		} else {
			for _, sc := range requested {
				if !subjectScopes[sc] {
					writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_scope: requested scope exceeds subject token scope"})
					return
				}
				if len(actorScopes) > 0 && !actorScopes[sc] {
					writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_scope: requested scope exceeds actor token scope"})
					return
				}
			}
		}
		narrowedScope := strings.Join(requested, " ")

		// Build delegation chain entry (token fragments only, for audit).
		sub, _ := actorClaims["sub"].(string)
		if sub == "" {
			sub = "unknown"
		}
		entry := DelegationEntry{
			Actor:   "actor:" + sub,
			Subject: "subject:" + func() string { s, _ := subjectClaims["sub"].(string); return s }(),
			Scope:   narrowedScope,
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
			"access_token":     "", // no dev tokens in production code,
			"token_type":       "Bearer",
			"expires_in":       3600,
			"scope":            narrowedScope,
			"delegation_chain": []DelegationEntry{entry},
			"act":              actClaim,
			"chain_id":         chainID,
		})
	}
}
