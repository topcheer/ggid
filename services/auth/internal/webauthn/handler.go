// Package webauthn implements Passkey/WebAuthn registration and authentication
// using the go-webauthn library for full cryptographic verification.
package webauthn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// --- Domain Types ---

// Credential represents a registered WebAuthn credential (passkey).
type Credential struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	UserID       uuid.UUID
	Name         string
	CredentialID []byte // raw credential ID from authenticator
	PublicKey    []byte // COSE-encoded public key
	Transports   []string
	Counter      uint32
	CreatedAt    time.Time
	LastUsedAt   *time.Time
}

// webAuthnUser implements webauthn.User to bridge our domain user with the library.
type webAuthnUser struct {
	id          uuid.UUID
	username    string
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte { return u.id[:] }

func (u *webAuthnUser) WebAuthnName() string { return u.username }

func (u *webAuthnUser) WebAuthnDisplayName() string { return u.displayName }

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// --- Session Store (in-memory, ephemeral — production would use Redis) ---

type sessionData struct {
	userID    uuid.UUID
	tenantID  uuid.UUID
	challenge string
	data      *webauthn.SessionData
	createdAt time.Time
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]*sessionData
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]*sessionData)}
}

func (s *sessionStore) save(key string, sd *sessionData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sd.createdAt = time.Now()
	s.sessions[key] = sd
}

func (s *sessionStore) get(key string) (*sessionData, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sd, ok := s.sessions[key]
	if !ok {
		return nil, false
	}
	// Expire after 5 minutes.
	if time.Since(sd.createdAt) > 5*time.Minute {
		delete(s.sessions, key)
		return nil, false
	}
	return sd, true
}

func (s *sessionStore) delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, key)
}

// --- Credential Store Interface ---

// CredentialStore manages persisted WebAuthn credentials.
type CredentialStore interface {
	SaveCredential(ctx context.Context, cred *Credential) error
	GetCredentialsByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*Credential, error)
	GetCredentialByID(ctx context.Context, tenantID uuid.UUID, credID []byte) (*Credential, error)
	UpdateCounter(ctx context.Context, tenantID uuid.UUID, credID []byte, counter uint32) error
	DeleteCredential(ctx context.Context, tenantID uuid.UUID, credID []byte) error
}

// --- Handler ---

// Handler implements WebAuthn HTTP endpoints with full go-webauthn verification.
type Handler struct {
	wbn          *webauthn.WebAuthn
	creds        CredentialStore
	sessions     *sessionStore
}

// NewHandler creates a new WebAuthn handler with full verification support.
// CredentialStore can be nil for skeleton mode (credentials are not persisted).
func NewHandler(rpID, rpName string, store CredentialStore) (*Handler, error) {
	wconfig := &webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     []string{"https://" + rpID, "http://localhost:3000"},
	}

	wbn, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("init webauthn: %w", err)
	}

	return &Handler{
		wbn:      wbn,
		creds:    store,
		sessions: newSessionStore(),
	}, nil
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func getTenantAndUser(r *http.Request) (context.Context, uuid.UUID, uuid.UUID, error) {
	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, fmt.Errorf("missing or invalid X-Tenant-ID")
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		// Try POST body for user_id
		// Fallback: parse from Authorization context (in production, JWT would provide this)
		return nil, uuid.Nil, uuid.Nil, fmt.Errorf("user_id is required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, uuid.Nil, uuid.Nil, fmt.Errorf("invalid user_id")
	}

	tc := &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	}
	ctx := ggidtenant.WithContext(r.Context(), tc)
	return ctx, tenantID, userID, nil
}

func (h *Handler) buildWebAuthnUser(ctx context.Context, tenantID, userID uuid.UUID) (*webAuthnUser, error) {
	var wcreds []webauthn.Credential

	if h.creds != nil {
		creds, err := h.creds.GetCredentialsByUser(ctx, tenantID, userID)
		if err == nil {
			for _, c := range creds {
				var transports []protocol.AuthenticatorTransport
				for _, t := range c.Transports {
					transports = append(transports, protocol.AuthenticatorTransport(t))
				}
				wcreds = append(wcreds, webauthn.Credential{
					ID:              c.CredentialID,
					PublicKey:       c.PublicKey,
					AttestationType: "none",
					Authenticator: webauthn.Authenticator{
						AAGUID:    make([]byte, 16),
						SignCount: c.Counter,
						Attachment: protocol.Platform,
					},
				})
			}
		}
	}

	return &webAuthnUser{
		id:          userID,
		username:    userID.String(),
		displayName: userID.String(),
		credentials: wcreds,
	}, nil
}

// --- Registration ---

func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, tenantID, userID, err := getTenantAndUser(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.buildWebAuthnUser(ctx, tenantID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	options, sessData, err := h.wbn.BeginRegistration(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("begin registration: %v", err))
		return
	}

	// Store session keyed by the challenge.
	challenge := options.Response.Challenge.String()
	h.sessions.save("reg:"+challenge, &sessionData{
		userID:    userID,
		tenantID:  tenantID,
		challenge: challenge,
		data:      sessData,
	})

	writeJSON(w, http.StatusOK, options)
}

func (h *Handler) finishRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, tenantID, userID, err := getTenantAndUser(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse the credential creation response from the authenticator.
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("parse credential creation: %v", err))
		return
	}

	// Find the session by challenge.
	challenge := parsedResponse.Response.CollectedClientData.Challenge
	sd, ok := h.sessions.get("reg:" + challenge)
	if !ok {
		writeError(w, http.StatusBadRequest, "session expired or not found")
		return
	}
	defer h.sessions.delete("reg:" + challenge)

	user, err := h.buildWebAuthnUser(ctx, tenantID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Verify the attestation — this is the core cryptographic check.
	credential, err := h.wbn.CreateCredential(user, *sd.data, parsedResponse)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("verify attestation: %v", err))
		return
	}

	// Persist the credential if store is available.
	if h.creds != nil {
		cred := &Credential{
			ID:           uuid.New(),
			TenantID:     tenantID,
			UserID:       userID,
			Name:         r.URL.Query().Get("name"),
			CredentialID: credential.ID,
			PublicKey:    credential.PublicKey,
			Transports:   []string{string(credential.Authenticator.Attachment)},
			Counter:      credential.Authenticator.SignCount,
			CreatedAt:    time.Now(),
		}
		if err := h.creds.SaveCredential(ctx, cred); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("save credential: %v", err))
			return
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":         "registered",
		"credential_id":  base64.RawURLEncoding.EncodeToString(credential.ID),
		"sign_count":     credential.Authenticator.SignCount,
	})
}

// --- Authentication ---

func (h *Handler) beginAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID")
		return
	}

	ctx := ggidtenant.WithContext(r.Context(), &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	})

	// For discoverable credentials (passkeys), we don't need the user upfront.
	// Create an ephemeral user with no credentials — go-webauthn will handle it.
	ephemeralUser := &webAuthnUser{
		id:          uuid.New(),
		username:    "discoverable",
		displayName: "Discoverable Credential",
	}

	options, sessData, err := h.wbn.BeginLogin(ephemeralUser)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("begin login: %v", err))
		return
	}

	challenge := options.Response.Challenge.String()
	h.sessions.save("auth:"+challenge, &sessionData{
		tenantID:  tenantID,
		challenge: challenge,
		data:      sessData,
	})

	_ = ctx // context for credential lookup in finish step
	writeJSON(w, http.StatusOK, options)
}

func (h *Handler) finishAuthentication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing or invalid X-Tenant-ID")
		return
	}

	// Parse the assertion response from the authenticator.
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("parse assertion: %v", err))
		return
	}

	// Find the session by challenge.
	challenge := parsedResponse.Response.CollectedClientData.Challenge
	sd, ok := h.sessions.get("auth:" + challenge)
	if !ok {
		writeError(w, http.StatusBadRequest, "session expired or not found")
		return
	}
	defer h.sessions.delete("auth:" + challenge)

	// Look up the credential to build the user for verification.
	ctx := ggidtenant.WithContext(r.Context(), &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	})

	// Build user from the credential referenced in the assertion.
	var user *webAuthnUser
	if h.creds != nil {
		cred, err := h.creds.GetCredentialByID(ctx, tenantID, parsedResponse.RawID)
		if err == nil && cred != nil {
			user, _ = h.buildWebAuthnUser(ctx, tenantID, cred.UserID)
		}
	}
	if user == nil {
		// No stored credential — create ephemeral user (assertion will fail without valid credential).
		user = &webAuthnUser{
			id:          uuid.New(),
			username:    "discoverable",
			displayName: "Discoverable Credential",
		}
	}

	// Verify the assertion — this is the core cryptographic check.
	credential, err := h.wbn.ValidateLogin(user, *sd.data, parsedResponse)
	if err != nil {
		writeError(w, http.StatusUnauthorized, fmt.Sprintf("verify assertion: %v", err))
		return
	}

	// Update credential counter.
	if h.creds != nil {
		_ = h.creds.UpdateCounter(ctx, tenantID, credential.ID, credential.Authenticator.SignCount)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "authenticated",
		"credential_id": base64.RawURLEncoding.EncodeToString(credential.ID),
		"sign_count":    credential.Authenticator.SignCount,
	})
}

// --- Credential Management ---

func (h *Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing X-Tenant-ID")
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		writeJSON(w, http.StatusOK, map[string]any{"credentials": []any{}})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if h.creds == nil {
		writeJSON(w, http.StatusOK, map[string]any{"credentials": []any{}})
		return
	}

	ctx := ggidtenant.WithContext(r.Context(), &ggidtenant.Context{
		TenantID:       tenantID,
		IsolationLevel: ggidtenant.IsolationShared,
	})

	creds, err := h.creds.GetCredentialsByUser(ctx, tenantID, userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"credentials": []any{}})
		return
	}

	result := make([]map[string]any, 0, len(creds))
	for _, c := range creds {
		result = append(result, map[string]any{
			"id":            c.ID.String(),
			"name":          c.Name,
			"credential_id": base64.RawURLEncoding.EncodeToString(c.CredentialID),
			"created_at":    c.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"credentials": result})
}

func (h *Handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.Header.Get("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing X-Tenant-ID")
		return
	}

	// Extract credential ID from URL path.
	credIDStr := strings.TrimPrefix(r.URL.Path, "/api/v1/webauthn/credentials/")
	credID, err := base64.RawURLEncoding.DecodeString(credIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid credential ID")
		return
	}

	if h.creds != nil {
		ctx := ggidtenant.WithContext(r.Context(), &ggidtenant.Context{
			TenantID:       tenantID,
			IsolationLevel: ggidtenant.IsolationShared,
		})
		if err := h.creds.DeleteCredential(ctx, tenantID, credID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
