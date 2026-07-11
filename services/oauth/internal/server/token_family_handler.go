package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type TokenFamilyMember struct {
	TokenID    string    `json:"token_id"`
	IssuedAt   time.Time `json:"issued_at"`
	Status     string    `json:"status"` // active, rotated, revoked
	RotatedTo  string    `json:"rotated_to,omitempty"`
}

type TokenFamily struct {
	FamilyID    string             `json:"family_id"`
	Tokens      []TokenFamilyMember `json:"tokens"`
	TheftDetected bool             `json:"theft_detected"`
	CreatedAt   time.Time          `json:"created_at"`
}

var (
	tokenFamilyMu sync.RWMutex
	tokenFamilies = make(map[string]*TokenFamily) // keyed by refresh_token_id (first token)
)

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

	tokenFamilyMu.RLock()
	family, ok := tokenFamilies[rtID]
	tokenFamilyMu.RUnlock()

	if !ok {
		// Return empty family structure
		writeJSON(w, http.StatusOK, map[string]any{
			"refresh_token_id": rtID,
			"family_id":        "",
			"tokens":           []TokenFamilyMember{},
			"theft_detected":   false,
			"note":             "no token family found for this refresh token",
		})
		return
	}

	writeJSON(w, http.StatusOK, family)
}

// RegisterTokenFamily registers a token rotation event (helper for internal use).
func RegisterTokenFamily(familyID, oldTokenID, newTokenID string) {
	tokenFamilyMu.Lock()
	defer tokenFamilyMu.Unlock()

	if _, ok := tokenFamilies[familyID]; !ok {
		tokenFamilies[familyID] = &TokenFamily{
			FamilyID:  familyID,
			CreatedAt: time.Now().UTC(),
		}
	}

	family := tokenFamilies[familyID]

	// Check for theft: if oldTokenID is already rotated, reuse = theft
	for i := range family.Tokens {
		if family.Tokens[i].TokenID == oldTokenID && family.Tokens[i].Status == "rotated" {
			family.TheftDetected = true
			family.Tokens[i].Status = "revoked" // revoke on theft detection
		}
	}

	// Add old token as rotated
	found := false
	for i := range family.Tokens {
		if family.Tokens[i].TokenID == oldTokenID {
			family.Tokens[i].Status = "rotated"
			family.Tokens[i].RotatedTo = newTokenID
			found = true
			break
		}
	}
	if !found {
		family.Tokens = append(family.Tokens, TokenFamilyMember{
			TokenID:   oldTokenID,
			IssuedAt:  time.Now().UTC(),
			Status:    "rotated",
			RotatedTo: newTokenID,
		})
	}

	// Add new token as active
	family.Tokens = append(family.Tokens, TokenFamilyMember{
		TokenID:  newTokenID,
		IssuedAt: time.Now().UTC(),
		Status:   "active",
	})
}
