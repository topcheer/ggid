package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// validIDsResponse is the JSON response for GET /api/v1/auth/webauthn/credentials/valid-ids.
type validIDsResponse struct {
	UserID        string   `json:"user_id"`
	UserName      string   `json:"user_name"`
	DisplayName   string   `json:"display_name"`
	CredentialIDs []string `json:"credential_ids"`
}

// handleWebAuthnValidIDs returns the user's WebAuthn user handle and active credential IDs.
// Used by frontend for signalAllAcceptedCredentials and signalCurrentUserDetails.
// JWT-protected: extracts user_id from Bearer token sub claim.
func (h *Handler) handleWebAuthnValidIDs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user_id from JWT.
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == "" || tokenStr == authHeader {
		writeError(w, http.StatusUnauthorized, "missing Authorization header")
		return
	}

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
		return h.authSvc.PublicKey(), nil
	})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	userIDStr, _ := claims["sub"].(string)
	if userIDStr == "" {
		writeError(w, http.StatusUnauthorized, "token missing sub claim")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id in token")
		return
	}

	// Get tenant from context or use default.
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil || tc == nil {
		tc = &ggidtenant.Context{TenantID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}
	}

	// Get user display info.
	var userName, displayName string
	if h.authSvc != nil {
		if ic := h.authSvc.IdentityClient(); ic != nil {
			if user, err := ic.GetUserByID(r.Context(), tc.TenantID, userID); err == nil && user != nil {
				userName = user.Email
				if userName == "" {
					userName = user.Username
				}
				displayName = user.DisplayName
				if displayName == "" {
					displayName = user.Username
				}
			}
		}
	}
	if userName == "" {
		userName = userIDStr
	}
	if displayName == "" {
		displayName = userName
	}

	// WebAuthn user handle: user UUID bytes as base64url.
	userHandle := base64.RawURLEncoding.EncodeToString(userID[:])

	// Collect credential IDs.
	credIDSet := make(map[string]bool)

	// 1. Check in-memory passkey store.
	pkMu.RLock()
	for _, cred := range pkCredentials {
		if cred.UserID == userIDStr && !cred.Revoked {
			credIDSet[cred.ID] = true
		}
	}
	pkMu.RUnlock()

	// 2. DB-backed webauthn credentials would be queried here if pool is available.
	// The pool is accessed via the repository layer, not directly on authSvc.
	// For now, we rely on the in-memory passkey store. Production will use the
	// pgWebAuthnCredentialStore via the webauthn handler.

	// Convert set to slice.
	credentialIDs := make([]string, 0, len(credIDSet))
	for id := range credIDSet {
		credentialIDs = append(credentialIDs, id)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validIDsResponse{
		UserID:        userHandle,
		UserName:      userName,
		DisplayName:   displayName,
		CredentialIDs: credentialIDs,
	})
}
