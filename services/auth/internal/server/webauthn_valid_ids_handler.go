package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/webauthn"
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
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user_id from JWT.
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == "" || tokenStr == authHeader {
		writeJSONError(w, http.StatusUnauthorized, "missing Authorization header")
		return
	}

	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(tok *jwt.Token) (any, error) {
		return h.authSvc.PublicKey(), nil
	})
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	userIDStr, _ := claims["sub"].(string)
	if userIDStr == "" {
		writeJSONError(w, http.StatusUnauthorized, "token missing sub claim")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id in token")
		return
	}

	// Require tenant from context — no fallback to default tenant.
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil || tc == nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	// Get user display info from identity service.
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

	// Collect credential IDs from the DB-backed credential store.
	var credentialIDs []string

	if h.waCredStore != nil {
		creds, err := h.waCredStore.GetCredentialsByUser(r.Context(), tc.TenantID, userID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		for _, c := range creds {
			if len(c.CredentialID) > 0 {
				credentialIDs = append(credentialIDs, base64.RawURLEncoding.EncodeToString(c.CredentialID))
			}
		}
	}

	// Also check in-memory passkey store as fallback (for dev/test without DB).
	pkMu.RLock()
	for _, cred := range pkCredentials {
		if cred.UserID == userIDStr && !cred.Revoked {
			// Avoid duplicates.
			found := false
			for _, existing := range credentialIDs {
				if existing == cred.ID {
					found = true
					break
				}
			}
			if !found {
				credentialIDs = append(credentialIDs, cred.ID)
			}
		}
	}
	pkMu.RUnlock()

	if credentialIDs == nil {
		credentialIDs = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validIDsResponse{
		UserID:        userHandle,
		UserName:      userName,
		DisplayName:   displayName,
		CredentialIDs: credentialIDs,
	})
}

// Ensure webauthn package is imported for CredentialStore interface usage.
var _ webauthn.CredentialStore = (webauthn.CredentialStore)(nil)
var _ = context.Background
