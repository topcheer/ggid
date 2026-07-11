package webauthn

import (
	"encoding/json"
	"net/http"
)

// ConditionalUIRequest configures conditional mediation for passkey autofill.
type ConditionalUIRequest struct {
	Challenge        string `json:"challenge"`
	RPID             string `json:"rp_id"`
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	UserVerification string `json:"user_verification"`
}

// ConditionalUIResponse is the browser-facing credential request for autofill.
type ConditionalUIResponse struct {
	Mediation      string `json:"mediation"`       // "conditional"
	PublicKey      PublicKeyCredentialRequest `json:"publicKey"`
}

// PublicKeyCredentialRequest mirrors the WebAuthn navigator.credentials.get options.
type PublicKeyCredentialRequest struct {
	Challenge        []byte                  `json:"challenge"`
	RPID             string                  `json:"rpId"`
	AllowCredentials []PublicKeyDescriptor   `json:"allowCredentials,omitempty"`
	UserVerification string                  `json:"userVerification"`
}

// PublicKeyDescriptor identifies an allowed credential for autofill.
type PublicKeyDescriptor struct {
	Type string   `json:"type"`
	ID   []byte   `json:"id"`
}

// BeginConditionalUI generates a credential request for browser autofill.
// The frontend uses navigator.credentials.get({mediation: "conditional"}).
func BeginConditionalUI(req *ConditionalUIRequest, registeredCredentials [][]byte) *ConditionalUIResponse {
	allowed := make([]PublicKeyDescriptor, 0, len(registeredCredentials))
	for _, credID := range registeredCredentials {
		allowed = append(allowed, PublicKeyDescriptor{
			Type: "public-key",
			ID:   credID,
		})
	}

	uv := req.UserVerification
	if uv == "" {
		uv = "preferred"
	}

	return &ConditionalUIResponse{
		Mediation: "conditional",
		PublicKey: PublicKeyCredentialRequest{
			Challenge:        []byte(req.Challenge),
			RPID:             req.RPID,
			AllowCredentials: allowed,
			UserVerification: uv,
		},
	}
}

// HandleConditionalUIBegin is the HTTP handler for GET /webauthn/conditional-ui/begin.
func HandleConditionalUIBegin(registeredCreds map[string][][]byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}

		creds := registeredCreds[userID]
		if creds == nil {
			creds = [][]byte{}
		}

		req := &ConditionalUIRequest{
			Challenge:        r.URL.Query().Get("challenge"),
			RPID:             r.URL.Query().Get("rp_id"),
			UserID:           userID,
			UserVerification: r.URL.Query().Get("uv"),
		}

		resp := BeginConditionalUI(req, creds)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// IsConditionalMediationSupported checks the browser hint header.
func IsConditionalMediationSupported(r *http.Request) bool {
	return r.Header.Get("Sec-WebAuthn-Conditional-Mediation") == "true"
}
