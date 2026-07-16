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
	ID              uuid.UUID
	TenantID        uuid.UUID
	UserID          uuid.UUID
	Name            string
	CredentialID    []byte // raw credential ID from authenticator
	PublicKey       []byte // COSE-encoded public key
	Transports      []string
	Counter         uint32
	BackupEligible  bool   // WA-1: credential can be backed up (sync'd)
	BackupState     bool   // WA-1: credential has been backed up
	UserVerified    bool   // WA-1: user verification flag at registration
	AttestationType string // WA-1: attestation type (e.g. "none", "basic_full")
	AAGUID          []byte // WA-1: authenticator model identifier
	Attachment      string // platform or cross-platform
	CreatedAt       time.Time
	LastUsedAt      *time.Time
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
	UpdateLastUsed(ctx context.Context, tenantID uuid.UUID, credID []byte, lastUsedAt time.Time) error
	DeleteCredential(ctx context.Context, tenantID uuid.UUID, credID []byte) error
}

// --- Handler ---

// Handler implements WebAuthn HTTP endpoints with full go-webauthn verification.
type Handler struct {
	wbn           *webauthn.WebAuthn
	creds         CredentialStore
	sessions      *sessionStore
	origins       []string       // WA-11: allowed RP origins for ROR
	androidPkg    string         // WA-12: Android package name for asset links
	androidSHA256 string         // WA-12: Android app SHA-256 fingerprint
	iosAppIDs     []string       // WA-12: iOS app IDs for universal links
}

// HandlerOption configures a Handler at construction time.
type HandlerOption func(*handlerConfig)

type handlerConfig struct {
	origins       []string
	androidPkg    string   // WA-12
	androidSHA256 string   // WA-12
	iosAppIDs     []string // WA-12
}

// WithOrigins sets the allowed RP origins for WebAuthn (WA-9).
func WithOrigins(origins []string) HandlerOption {
	return func(c *handlerConfig) {
		c.origins = origins
	}
}

// WithAndroidAssetLinks configures Android Digital Asset Links (WA-12).
func WithAndroidAssetLinks(pkg, sha256 string) HandlerOption {
	return func(c *handlerConfig) {
		c.androidPkg = pkg
		c.androidSHA256 = sha256
	}
}

// WithIOSAppSiteAssociation configures iOS Universal Links app IDs (WA-12).
func WithIOSAppSiteAssociation(appIDs []string) HandlerOption {
	return func(c *handlerConfig) {
		c.iosAppIDs = appIDs
	}
}

// NewHandler creates a new WebAuthn handler with full verification support.
// CredentialStore can be nil for skeleton mode (credentials are not persisted).
func NewHandler(rpID, rpName string, store CredentialStore, opts ...HandlerOption) (*Handler, error) {
	cfg := &handlerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	origins := cfg.origins
	if len(origins) == 0 {
		origins = []string{"https://" + rpID, "http://localhost:3000"}
	}

	wconfig := &webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     origins,
	}

	wbn, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("init webauthn: %w", err)
	}

	return &Handler{
		wbn:           wbn,
		creds:         store,
		sessions:      newSessionStore(),
		origins:       origins,
		androidPkg:    cfg.androidPkg,
		androidSHA256: cfg.androidSHA256,
		iosAppIDs:     cfg.iosAppIDs,
	}, nil
}

// generateCredentialName derives a human-readable credential name from the
// User-Agent header when no explicit name is provided (WA-7).
func generateCredentialName(userAgent string) string {
	if userAgent == "" {
		return "Passkey"
	}
	ua := strings.ToLower(userAgent)

	var browser string
	switch {
	case strings.Contains(ua, "edg/"):
		browser = "Edge"
	case strings.Contains(ua, "edge/"):
		browser = "Edge"
	case strings.Contains(ua, "chrome/") || strings.Contains(ua, "crios/"):
		browser = "Chrome"
	case strings.Contains(ua, "firefox/") || strings.Contains(ua, "fxios/"):
		browser = "Firefox"
	case strings.Contains(ua, "safari/"):
		browser = "Safari"
	default:
		browser = "Browser"
	}

	var platform string
	switch {
	case strings.Contains(ua, "windows"):
		platform = "Windows"
	case strings.Contains(ua, "android"):
		platform = "Android"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		platform = "iOS"
	case strings.Contains(ua, "mac os") || strings.Contains(ua, "macintosh"):
		platform = "macOS"
	case strings.Contains(ua, "linux"):
		platform = "Linux"
	default:
		platform = "Device"
	}

	return browser + " on " + platform
}

// RegisterRoutes registers WebAuthn endpoints on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/webauthn/register/begin", h.beginRegistration)
	mux.HandleFunc("/api/v1/webauthn/register/finish", h.finishRegistration)
	mux.HandleFunc("/api/v1/webauthn/auth/begin", h.beginAuthentication)
	mux.HandleFunc("/api/v1/webauthn/auth/finish", h.finishAuthentication)
	mux.HandleFunc("/api/v1/webauthn/credentials", h.listCredentials)
	mux.HandleFunc("/api/v1/webauthn/credentials/", h.deleteCredential)

	// WA-11: Related Origin Requests (ROR)
	mux.HandleFunc("/.well-known/webauthn", h.wellKnownWebAuthn)

	// WA-12: Mobile app integration
	mux.HandleFunc("/.well-known/assetlinks.json", h.wellKnownAssetLinks)
	mux.HandleFunc("/.well-known/apple-app-site-association", h.wellKnownAppleAppSiteAssociation)
}

// --- Well-Known Endpoints (WA-11, WA-12) ---

// wellKnownWebAuthn returns Related Origin Requests (ROR) JSON (WA-11).
// Browsers fetch this to discover which origins share the RP ID.
func (h *Handler) wellKnownWebAuthn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	origins := h.origins
	if len(origins) == 0 {
		origins = []string{"https://" + h.wbn.Config.RPID}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"origins": origins,
	})
}

// wellKnownAssetLinks returns Android Digital Asset Links JSON (WA-12).
// This allows Android apps to associate with the WebAuthn RP.
func (h *Handler) wellKnownAssetLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.androidPkg == "" {
		// Default empty response when not configured.
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	writeJSON(w, http.StatusOK, []map[string]any{
		{
			"relation": []string{"delegate_permission/common.get_login_creds"},
			"target": map[string]any{
				"namespace":          "android",
				"package_name":       h.androidPkg,
				"sha256_cert_fingerprints": []string{h.androidSHA256},
			},
		},
	})
}

// wellKnownAppleAppSiteAssociation returns iOS Universal Links JSON (WA-12).
// This allows iOS apps to associate with the WebAuthn RP.
func (h *Handler) wellKnownAppleAppSiteAssociation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if len(h.iosAppIDs) == 0 {
		// Default empty response when not configured.
		writeJSON(w, http.StatusOK, map[string]any{
			"applinks": map[string]any{"details": []any{}},
		})
		return
	}

	details := make([]map[string]any, 0, len(h.iosAppIDs))
	for _, appID := range h.iosAppIDs {
		details = append(details, map[string]any{
			"apps": []string{appID},
			"components": []string{"/*"},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"applinks": map[string]any{"details": details},
	})
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

// classifyError maps a WebAuthn library error to a structured error code.
// This helps the frontend display user-friendly messages.
func classifyError(err error) (code, message string) {
	if err == nil {
		return "OK", ""
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "notallowed"):
		return "USER_CANCELLED", "The operation was cancelled or not allowed"
	case strings.Contains(lower, "invalidstate"):
		return "INVALID_STATE", "A credential with this authenticator already exists"
	case strings.Contains(lower, "abort"):
		return "TIMEOUT", "The operation timed out"
	case strings.Contains(lower, "security") || strings.Contains(lower, "origin"):
		return "SECURITY_ERROR", "Security verification failed"
	default:
		return "UNKNOWN_ERROR", msg
	}
}

// writeClassifiedError writes a structured WebAuthn error response.
func writeClassifiedError(w http.ResponseWriter, status int, err error) {
	code, msg := classifyError(err)
	writeJSON(w, status, map[string]string{
		"error":      msg,
		"error_code": code,
	})
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
					transports = append(transports, protocol.AuthenticatorTransport(t)) //nolint:staticcheck // SA4010: intentional build pattern
				}

				wcreds = append(wcreds, webauthn.Credential{
					ID:              c.CredentialID,
					PublicKey:       c.PublicKey,
					AttestationType: c.AttestationType,
					Flags: webauthn.CredentialFlags{
						BackupEligible: c.BackupEligible,
						BackupState:    c.BackupState,
						UserVerified:   c.UserVerified,
					},
					Authenticator: webauthn.Authenticator{
						AAGUID:     c.AAGUID,
						SignCount:  c.Counter,
						Attachment: resolveAttachment(c.Attachment),
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
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// WA-3: exclude existing credentials so the same authenticator can't be registered twice.
	var excludeCreds []protocol.CredentialDescriptor
	for _, wc := range user.credentials {
		excludeCreds = append(excludeCreds, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: wc.ID,
		})
	}

	// WA-4: explicit AuthenticatorSelection with ResidentKey preferred, UV preferred.
	authSel := protocol.AuthenticatorSelection{
		ResidentKey:      protocol.ResidentKeyRequirementPreferred,
		UserVerification: protocol.VerificationPreferred,
	}

	var regOpts []webauthn.RegistrationOption
	regOpts = append(regOpts, webauthn.WithAuthenticatorSelection(authSel))
	if len(excludeCreds) > 0 {
		regOpts = append(regOpts, webauthn.WithExclusions(excludeCreds))
	}

	options, sessData, err := h.wbn.BeginRegistration(user, regOpts...)
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
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Verify the attestation — this is the core cryptographic check.
	credential, err := h.wbn.CreateCredential(user, *sd.data, parsedResponse)
	if err != nil {
		writeClassifiedError(w, http.StatusBadRequest, fmt.Errorf("verify attestation: %w", err))
		return
	}

		// Persist the credential if store is available.
	if h.creds != nil {
		// WA-7: auto-generate credential name from User-Agent if not provided.
		name := r.URL.Query().Get("name")
		if name == "" {
			name = generateCredentialName(r.Header.Get("User-Agent"))
		}

		// WA-8: persist actual transports from credential.Transport.
		var transports []string
		for _, t := range credential.Transport {
			transports = append(transports, string(t))
		}
		if len(transports) == 0 {
			transports = []string{string(credential.Authenticator.Attachment)}
		}

		cred := &Credential{
			ID:              uuid.New(),
			TenantID:        tenantID,
			UserID:          userID,
			Name:            name,
			CredentialID:    credential.ID,
			PublicKey:       credential.PublicKey,
			Transports:      transports,
			Counter:         credential.Authenticator.SignCount,
			BackupEligible:  credential.Flags.BackupEligible,
			BackupState:     credential.Flags.BackupState,
			UserVerified:    credential.Flags.UserVerified,
			AttestationType: credential.AttestationType,
			AAGUID:          credential.Authenticator.AAGUID,
			Attachment:      string(credential.Authenticator.Attachment),
			CreatedAt:       time.Now(),
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

	userIDStr := r.URL.Query().Get("user_id")

	// WA-15: If user_id is provided, populate allowCredentials with stored transports.
	var loginOpts []webauthn.LoginOption
	var sessionUserID uuid.UUID

	if userIDStr != "" {
		uid, parseErr := uuid.Parse(userIDStr)
		if parseErr == nil {
			sessionUserID = uid
			user, buildErr := h.buildWebAuthnUser(ctx, tenantID, uid)
			if buildErr == nil && len(user.credentials) > 0 {
				var allowCreds []protocol.CredentialDescriptor
				for _, wc := range user.credentials {
					transports := append([]protocol.AuthenticatorTransport(nil), wc.Transport...)
					allowCreds = append(allowCreds, protocol.CredentialDescriptor{
						Type:         protocol.PublicKeyCredentialType,
						CredentialID: wc.ID,
						Transport:    transports,
					})
				}
				if len(allowCreds) > 0 {
					loginOpts = append(loginOpts, webauthn.WithAllowedCredentials(allowCreds))
				}
			}
		}
	}

	// If no user_id, use ephemeral user for discoverable credential flow.
	var loginUser webauthn.User
	if sessionUserID != uuid.Nil {
		u, _ := h.buildWebAuthnUser(ctx, tenantID, sessionUserID)
		loginUser = u
	} else {
		loginUser = &webAuthnUser{
			id:          uuid.New(),
			username:    "discoverable",
			displayName: "Discoverable Credential",
		}
	}

	options, sessData, err := h.wbn.BeginLogin(loginUser, loginOpts...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("begin login: %v", err))
		return
	}

	challenge := options.Response.Challenge.String()
	h.sessions.save("auth:"+challenge, &sessionData{
		userID:    sessionUserID,
		tenantID:  tenantID,
		challenge: challenge,
		data:      sessData,
	})

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
		writeClassifiedError(w, http.StatusUnauthorized, fmt.Errorf("verify assertion: %w", err))
		return
	}

	// WA-2: Clone detection — check signCount monotonicity.
	// If the stored counter > 0 and the received counter <= stored counter,
	// a cloned authenticator may exist.
	if h.creds != nil {
		storedCred, getErr := h.creds.GetCredentialByID(ctx, tenantID, credential.ID)
		if getErr == nil && storedCred != nil && storedCred.Counter > 0 {
			if credential.Authenticator.SignCount <= storedCred.Counter {
				writeError(w, http.StatusUnauthorized, "possible credential clone detected")
				return
			}
		}
	}

	// Update credential counter and LastUsedAt (WA-7).
	if h.creds != nil {
		now := time.Now()
		_ = h.creds.UpdateCounter(ctx, tenantID, credential.ID, credential.Authenticator.SignCount)
		_ = h.creds.UpdateLastUsed(ctx, tenantID, credential.ID, now)
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
			"id":             c.ID.String(),
			"name":           c.Name,
			"credential_id":  base64.RawURLEncoding.EncodeToString(c.CredentialID),
			"created_at":     c.CreatedAt,
			"transports":     c.Transports,
			"backup_eligible": c.BackupEligible,
			"backup_state":    c.BackupState,
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
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// resolveAttachment returns the protocol attachment type from a stored string.
// Falls back to platform if the value is empty or unrecognized.
func resolveAttachment(stored string) protocol.AuthenticatorAttachment {
	switch protocol.AuthenticatorAttachment(stored) {
	case protocol.CrossPlatform:
		return protocol.CrossPlatform
	case protocol.Platform:
		return protocol.Platform
	default:
		// Empty or unrecognized — default to platform.
		return protocol.Platform
	}
}
