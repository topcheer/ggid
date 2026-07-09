// Package webauthn implements Passkey/WebAuthn registration and authentication.
// This is a skeleton that provides the HTTP API surface; full WebAuthn
// cryptographic verification requires a frontend (authenticator) interaction.
package webauthn

import (
	"crypto/rand"
	"encoding/base64"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// Credential represents a registered WebAuthn credential (passkey).
type Credential struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	UserID       uuid.UUID
	Name         string
	CredentialID []byte    // base64url-encoded credential ID from authenticator
	PublicKey    []byte    // COSE-encoded public key
	Attestation  string    // attestation format
	Transports   []string  // internal, hybrid, usb, nfc
	Counter      uint32    // signature counter
	CreatedAt    time.Time
	LastUsedAt   *time.Time
}

// RegistrationChallenge is a server-issued challenge for credential registration.
type RegistrationChallenge struct {
	Challenge    string
	UserID       uuid.UUID
	TenantID     uuid.UUID
	RpID         string    // e.g. "ggid.dev"
	RpName       string    // e.g. "GGID Platform"
	Timeout      int       // milliseconds
	ExpiresAt    time.Time
}

// AuthenticationChallenge is a server-issued challenge for authentication.
type AuthenticationChallenge struct {
	Challenge    string
	TenantID     uuid.UUID
	RpID         string
	Timeout      int
	ExpiresAt    time.Time
}

// Handler implements WebAuthn HTTP endpoints.
type Handler struct {
	rpID   string
	rpName string
	// In production: CredentialStore for DB-backed credential storage
}

// NewHandler creates a new WebAuthn handler.
func NewHandler(rpID, rpName string) *Handler {
	return &Handler{rpID: rpID, rpName: rpName}
}

// RegisterRoutes registers WebAuthn endpoints on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/webauthn/register/begin", h.beginRegistration)
	mux.HandleFunc("/api/v1/webauthn/register/finish", h.finishRegistration)
	mux.HandleFunc("/api/v1/webauthn/auth/begin", h.beginAuthentication)
	mux.HandleFunc("/api/v1/webauthn/auth/finish", h.finishAuthentication)
	mux.HandleFunc("/api/v1/webauthn/credentials", h.listCredentials)
	mux.HandleFunc("/api/v1/webauthn/credentials/", h.deleteCredential)
}

// --- Helpers ---

func generateChallenge() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate challenge: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func getTenantCtx(r *http.Request) (bool, context.Context) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		return false, nil
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return false, nil
	}
	tc := &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	}
	return true, ggidtenant.WithContext(r.Context(), tc)
}

// --- Registration ---

// BeginRegistrationResponse is sent to the client to initiate passkey registration.
type BeginRegistrationResponse struct {
	Challenge       string `json:"challenge"`
	RpID            string `json:"rp_id"`
	RpName          string `json:"rp_name"`
	UserID          string `json:"user_id"`
	UserName        string `json:"user_name,omitempty"`
	Timeout         int    `json:"timeout"`
	AuthenticatorSelection map[string]any `json:"authenticatorSelection"`
}

func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ok, _ := getTenantCtx(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing X-Tenant-ID")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	challenge, err := generateChallenge()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := BeginRegistrationResponse{
		Challenge: challenge,
		RpID:      h.rpID,
		RpName:    h.rpName,
		UserID:    userID,
		Timeout:   60000,
		AuthenticatorSelection: map[string]any{
			"authenticatorAttachment": "platform",
			"userVerification":        "required",
			"residentKey":             "required",
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// FinishRegistrationRequest is sent by the client after authenticator interaction.
type FinishRegistrationRequest struct {
	UserID      string `json:"user_id"`
	Challenge   string `json:"challenge"`
	CredentialID string `json:"credential_id"`  // base64url
	PublicKey   string `json:"public_key"`      // base64url COSE key
	Attestation string `json:"attestation"`
	Transports  []string `json:"transports"`
	Name        string `json:"name"`
}

func (h *Handler) finishRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req FinishRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Skeleton: in production, verify the attestation here.
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":      "registered",
		"credential_id": req.CredentialID,
		"name":        req.Name,
		"note":        "WebAuthn attestation verification not yet implemented",
	})
}

// --- Authentication ---

func (h *Handler) beginAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ok, _ := getTenantCtx(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "missing X-Tenant-ID")
		return
	}

	challenge, err := generateChallenge()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"challenge": challenge,
		"rp_id":     h.rpID,
		"timeout":   60000,
		"user_verification": "required",
	})
}

type FinishAuthenticationRequest struct {
	Challenge    string `json:"challenge"`
	CredentialID string `json:"credential_id"`
	Signature    string `json:"signature"`
	AuthData     string `json:"authenticator_data"`
}

func (h *Handler) finishAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req FinishAuthenticationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Skeleton: in production, verify the assertion signature here.
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "authenticated",
		"note":   "WebAuthn assertion verification not yet implemented",
	})
}

// --- Credential Management ---

func (h *Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Skeleton: returns empty list until credential storage is implemented.
	writeJSON(w, http.StatusOK, map[string]any{
		"credentials": []any{},
	})
}

func (h *Handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Skeleton.
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
