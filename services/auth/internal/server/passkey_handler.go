package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PasskeyRegistrationSession struct {
	SessionID    string    `json:"session_id"`
	UserID       string    `json:"user_id"`
	Challenge    string    `json:"challenge"`
	RPID         string    `json:"rp_id"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}

type PasskeyAuthSession struct {
	SessionID    string    `json:"session_id"`
	Challenge    string    `json:"challenge"`
	RPID         string    `json:"rp_id"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}

type PasskeyCredential struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	PublicKey  string    `json:"public_key"`
	Counter    int       `json:"counter"`
	CreatedAt  time.Time `json:"created_at"`
	Revoked    bool      `json:"revoked"`
}

var (
	pkRegSessions   = make(map[string]*PasskeyRegistrationSession)
	pkAuthSessions  = make(map[string]*PasskeyAuthSession)
	pkCredentials   = make(map[string]*PasskeyCredential)
	pkMu            sync.RWMutex
	pkSeq           int
)

// resolveWebAuthnRPID reads RP ID from sys_config DB table first, falls back to env.
// Returns error if neither source provides a value.
func resolveWebAuthnRPID(h *Handler) (string, error) {
	// 1. Try DB
	if h.pool != nil {
		var configJSON []byte
		err := h.pool.QueryRow(context.Background(),
			`SELECT value::text FROM sys_config WHERE key = 'webauthn_config'`).Scan(&configJSON)
		if err == nil && len(configJSON) > 0 {
			var cfg struct {
				RPID string `json:"rp_id"`
			}
			if json.Unmarshal(configJSON, &cfg) == nil && cfg.RPID != "" {
				return cfg.RPID, nil
			}
		}
	}
	// 2. Fallback to env
	if rpID := os.Getenv("WEBAUTHN_RP_ID"); rpID != "" {
		return rpID, nil
	}
	return "", fmt.Errorf("WebAuthn RP ID not configured — set via /api/v1/system/config or WEBAUTHN_RP_ID env")
}

// resolveRPIDForConfig returns RP ID for display purposes (no error).
func resolveRPIDForConfig(h *Handler) string {
	rpID, err := resolveWebAuthnRPID(h)
	if err != nil {
		return ""
	}
	return rpID
}

func (h *Handler) handlePasskeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	pkMu.Lock()
	pkSeq++
	rpID, err := resolveWebAuthnRPID(h)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sess := &PasskeyRegistrationSession{
		SessionID: fmtPKID(pkSeq),
		UserID:    req.UserID,
		Challenge: generateChallenge(),
		RPID:      rpID,
		CreatedAt: time.Now(),
		Status:    "pending",
	}
	pkRegSessions[sess.SessionID] = sess
	pkMu.Unlock()

	// PG write-through
	if h.memMapRepo != nil {
		h.memMapRepo.StoreJSON(r.Context(), "auth_passkey_json", "reg:"+sess.SessionID, map[string]any{
			"session_id": sess.SessionID, "user_id": sess.UserID,
			"challenge": sess.Challenge, "rp_id": sess.RPID,
			"created_at": sess.CreatedAt, "status": sess.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sess)
}

func (h *Handler) handlePasskeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		SessionID  string `json:"session_id"`
		Credential struct {
			ID        string `json:"id"`
			PublicKey string `json:"public_key"`
		} `json:"credential"`
		AAGUID string `json:"aaguid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	pkMu.Lock()
	defer pkMu.Unlock()
	// Try PG first for session lookup, fall back to in-memory.
	var sess *PasskeyRegistrationSession
	if h.memMapRepo != nil {
		if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_passkey_json", "reg:"+req.SessionID); row != nil {
			sess = &PasskeyRegistrationSession{
				SessionID: req.SessionID,
				UserID:    getString(row, "user_id"),
				Status:    getString(row, "status"),
			}
		}
	}
	if sess == nil {
		var ok bool
		sess, ok = pkRegSessions[req.SessionID]
		if !ok {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
	}
	pkSeq++
	cred := &PasskeyCredential{
		ID:        req.Credential.ID,
		UserID:    sess.UserID,
		PublicKey: req.Credential.PublicKey,
		CreatedAt: time.Now(),
	}

	// KB-078: AAGUID allowlist enforcement — check if the authenticator is approved.
	if req.AAGUID != "" && h.aaguidAllowlistRepo != nil {
		if !h.aaguidAllowlistRepo.IsApproved(r.Context(), req.AAGUID) {
			// Audit: registration denied due to unapproved authenticator.
			h.publishAuditEventWithMeta(r,
				"webauthn.aaguid.registration_denied", "denied",
				"passkey_registration", req.AAGUID, uuid.Nil,
				map[string]any{
					"aaguid":       req.AAGUID,
					"user_id":      sess.UserID,
					"credential_id": req.Credential.ID,
					"reason":       "authenticator_not_approved",
				},
			)
			writeError(w, http.StatusForbidden, "This authenticator is not in the approved device list")
			return
		}
	}

	pkCredentials[cred.ID] = cred
	sess.Status = "completed"
	// PG write-through for credential and session
	if h.memMapRepo != nil {
		h.memMapRepo.StoreJSON(r.Context(), "auth_passkey_json", "cred:"+cred.ID, map[string]any{
			"id": cred.ID, "user_id": cred.UserID,
			"public_key": cred.PublicKey, "counter": cred.Counter,
			"created_at": cred.CreatedAt, "revoked": cred.Revoked,
		})
		h.memMapRepo.StoreJSON(r.Context(), "auth_passkey_json", "reg:"+req.SessionID, map[string]any{
			"session_id": req.SessionID, "user_id": sess.UserID,
			"status": "completed",
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cred)
}

func (h *Handler) handlePasskeyAuthBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pkMu.Lock()
	pkSeq++
	rpID, err := resolveWebAuthnRPID(h)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sess := &PasskeyAuthSession{
		SessionID: fmtPKID(pkSeq),
		Challenge: generateChallenge(),
		RPID:      rpID,
		CreatedAt: time.Now(),
		Status:    "pending",
	}
	pkAuthSessions[sess.SessionID] = sess
	pkMu.Unlock()

	// PG write-through
	if h.memMapRepo != nil {
		h.memMapRepo.StoreJSON(r.Context(), "auth_passkey_json", "auth:"+sess.SessionID, map[string]any{
			"session_id": sess.SessionID, "challenge": sess.Challenge,
			"rp_id": sess.RPID, "created_at": sess.CreatedAt,
			"status": sess.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

func (h *Handler) handlePasskeyAuthFinish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		SessionID    string `json:"session_id"`
		CredentialID string `json:"credential_id"`
		Assertion    string `json:"assertion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	pkMu.Lock()
	defer pkMu.Unlock()
	// Try PG first for session and credential lookup.
	var sess *PasskeyAuthSession
	var cred *PasskeyCredential
	if h.memMapRepo != nil {
		if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_passkey_json", "auth:"+req.SessionID); row != nil {
			sess = &PasskeyAuthSession{SessionID: req.SessionID, Status: getString(row, "status")}
		}
		if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_passkey_json", "cred:"+req.CredentialID); row != nil {
			cred = &PasskeyCredential{
				ID:      req.CredentialID,
				UserID:  getString(row, "user_id"),
				Counter: getInt(row, "counter"),
				Revoked: getBool(row, "revoked"),
			}
		}
	}
	if sess == nil {
		var ok bool
		sess, ok = pkAuthSessions[req.SessionID]
		if !ok {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
	}
	if cred == nil {
		var ok bool
		cred, ok = pkCredentials[req.CredentialID]
		if !ok || cred.Revoked {
			writeError(w, http.StatusUnauthorized, "credential not found or revoked")
			return
		}
	} else if cred.Revoked {
		writeError(w, http.StatusUnauthorized, "credential not found or revoked")
		return
	}
	cred.Counter++
	sess.Status = "verified"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "user_id": cred.UserID})
}

func (h *Handler) handlePasskeyRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		writeError(w, http.StatusBadRequest, "credential id required")
		return
	}
	id := parts[len(parts)-1]
	pkMu.Lock()
	defer pkMu.Unlock()
	cred, ok := pkCredentials[id]
	if !ok {
		writeError(w, http.StatusNotFound, "credential not found")
		return
	}
	cred.Revoked = true
	// PG write-through
	if h.memMapRepo != nil {
		if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_passkey_json", "cred:"+id); row != nil {
			row["revoked"] = true
			h.memMapRepo.StoreJSON(r.Context(), "auth_passkey_json", "cred:"+id, row)
		} else {
			h.memMapRepo.StoreJSON(r.Context(), "auth_passkey_json", "cred:"+id, map[string]any{
				"id": id, "revoked": true,
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "revoked", "id": id})
}

func (h *Handler) handlePasskeyStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pkMu.RLock()
	defer pkMu.RUnlock()
	var active, revoked int
	for _, c := range pkCredentials {
		if c.Revoked {
			revoked++
		} else {
			active++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"active":         active,
		"revoked":        revoked,
		"total":          active + revoked,
		"reg_sessions":   len(pkRegSessions),
		"auth_sessions":  len(pkAuthSessions),
	})
}

func fmtPKID(n int) string {
	const hex = "0123456789abcdef"
	if n == 0 {
		return "pk_0"
	}
	var buf []byte
	for n > 0 {
		buf = append([]byte{hex[n%16]}, buf...)
		n /= 16
	}
	return "pk_" + string(buf)
}

// --- map[string]any type-assertion helpers ---

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch s := v.(type) {
		case string:
			return s
		case fmt.Stringer:
			return s.String()
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// generateChallenge creates a cryptographically random challenge for WebAuthn.
func generateChallenge() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to UUID-based (still unique, just not crypto-random)
		return base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}