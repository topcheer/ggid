package server

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type TokenFamilyMember struct {
	TokenID    string    `json:"token_id"`
	IssuedAt   time.Time `json:"issued_at"`
	Status     string    `json:"status"` // active, rotated, revoked
	RotatedTo  string    `json:"rotated_to,omitempty"`
}

type TokenFamily struct {
	FamilyID      string              `json:"family_id"`
	Tokens        []TokenFamilyMember `json:"tokens"`
	TheftDetected bool                `json:"theft_detected"`
	CreatedAt     time.Time           `json:"created_at"`
}

// GET /api/v1/oauth/token-families/{refresh_token_id}
func handleTokenFamily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	rtID := strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/token-families/")
	if rtID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "refresh_token_id required"})
		return
	}

	if mapRepoVar != nil {
		data, err := mapRepoVar.Get(r.Context(), "oauth_token_families", rtID)
		if err == nil {
			writeJSON(w, http.StatusOK, data)
			return
		}
	}

	// Return empty family structure
	writeJSON(w, http.StatusOK, map[string]any{
		"refresh_token_id": rtID,
		"family_id":        "",
		"tokens":           []TokenFamilyMember{},
		"theft_detected":   false,
		"note":             "no token family found for this refresh token",
	})
}

// RegisterTokenFamily registers a token rotation event (helper for internal use).
func RegisterTokenFamily(familyID, oldTokenID, newTokenID string) {
	if mapRepoVar == nil {
		return
	}
	ctx := context.Background()

	var data map[string]any
	if existing, err := mapRepoVar.Get(ctx, "oauth_token_families", familyID); err == nil {
		data = existing
	} else {
		data = map[string]any{
			"family_id":      familyID,
			"created_at":     time.Now().UTC(),
			"tokens":         []map[string]any{},
			"theft_detected": false,
		}
	}

	tokensRaw, _ := data["tokens"].([]any)

	// Check for theft: if oldTokenID is already rotated, reuse = theft
	for i, t := range tokensRaw {
		if tm, ok := t.(map[string]any); ok {
			if tid, _ := tm["token_id"].(string); tid == oldTokenID {
				if status, _ := tm["status"].(string); status == "rotated" {
					data["theft_detected"] = true
					tm["status"] = "revoked"
					tokensRaw[i] = tm
				}
			}
		}
	}

	// Add old token as rotated
	found := false
	for i, t := range tokensRaw {
		if tm, ok := t.(map[string]any); ok {
			if tid, _ := tm["token_id"].(string); tid == oldTokenID {
				tm["status"] = "rotated"
				tm["rotated_to"] = newTokenID
				tokensRaw[i] = tm
				found = true
				break
			}
		}
	}
	if !found {
		tokensRaw = append(tokensRaw, map[string]any{
			"token_id":   oldTokenID,
			"issued_at":  time.Now().UTC(),
			"status":     "rotated",
			"rotated_to": newTokenID,
		})
	}

	// Add new token as active
	tokensRaw = append(tokensRaw, map[string]any{
		"token_id":  newTokenID,
		"issued_at": time.Now().UTC(),
		"status":    "active",
	})

	data["tokens"] = tokensRaw
	mapRepoVar.Store(ctx, "oauth_token_families", familyID, data)
}
