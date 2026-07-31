package server

import (
	"net/http"
	"sync"
)

// scopeDiffResult holds the comparison of two tokens' scopes.
type scopeDiffResult struct {
	TokenID string
	Scopes  map[string]bool
}

// tokenScopeStore is currently a stub — the store is never populated.
// This handler is a placeholder for a future token scope diff feature.
// When implemented, populate the store from actual OAuth token introspection
// (not hardcoded mock data).
var tokenScopeStore = struct {
	sync.RWMutex
	tokens map[string]*scopeDiffResult
}{tokens: make(map[string]*scopeDiffResult)}

// GET /api/v1/oauth/token-scope-diff?token_a=X&token_b=Y
func handleTokenScopeDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	tokenA := r.URL.Query().Get("token_a")
	tokenB := r.URL.Query().Get("token_b")
	if tokenA == "" || tokenB == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "token_a and token_b are required"})
		return
	}

	tokenScopeStore.RLock()
	defer tokenScopeStore.RUnlock()

	aData, aExists := tokenScopeStore.tokens[tokenA]
	bData, bExists := tokenScopeStore.tokens[tokenB]

	if !aExists || !bExists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "one or both tokens not found"})
		return
	}

	common := []string{}
	onlyA := []string{}
	onlyB := []string{}

	// Collect all scope keys
	allScopes := map[string]bool{}
	for s := range aData.Scopes {
		allScopes[s] = true
	}
	for s := range bData.Scopes {
		allScopes[s] = true
	}

	for scope := range allScopes {
		aHas := aData.Scopes[scope]
		bHas := bData.Scopes[scope]
		if aHas && bHas {
			common = append(common, scope)
		} else if aHas && !bHas {
			onlyA = append(onlyA, scope)
		} else if !aHas && bHas {
			onlyB = append(onlyB, scope)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token_a":      tokenA,
		"token_b":      tokenB,
		"common":       common,
		"only_a":       onlyA,
		"only_b":       onlyB,
		"common_count": len(common),
		"only_a_count": len(onlyA),
		"only_b_count": len(onlyB),
		"total_unique": len(allScopes),
	})
}
