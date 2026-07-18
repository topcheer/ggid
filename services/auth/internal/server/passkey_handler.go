package server

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func (h *Handler) handlePasskeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	pkMu.Lock()
	pkSeq++
	sess := &PasskeyRegistrationSession{
		SessionID: fmtPKID(pkSeq),
		UserID:    req.UserID,
		Challenge: "challenge-" + fmtPKID(pkSeq),
		RPID:      "auth.ggid.example",
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
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
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
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
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
			http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
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
			http.Error(w, `{"error":"authenticator_not_approved","message":"This authenticator is not in the approved device list"}`, http.StatusForbidden)
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
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	pkMu.Lock()
	pkSeq++
	sess := &PasskeyAuthSession{
		SessionID: fmtPKID(pkSeq),
		Challenge: "auth-challenge-" + fmtPKID(pkSeq),
		RPID:      "auth.ggid.example",
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
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		SessionID    string `json:"session_id"`
		CredentialID string `json:"credential_id"`
		Assertion    string `json:"assertion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
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
			http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
			return
		}
	}
	if cred == nil {
		var ok bool
		cred, ok = pkCredentials[req.CredentialID]
		if !ok || cred.Revoked {
			http.Error(w, `{"error":"credential not found or revoked"}`, http.StatusUnauthorized)
			return
		}
	} else if cred.Revoked {
		http.Error(w, `{"error":"credential not found or revoked"}`, http.StatusUnauthorized)
		return
	}
	cred.Counter++
	sess.Status = "verified"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "user_id": cred.UserID})
}

func (h *Handler) handlePasskeyRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, `{"error":"credential id required"}`, http.StatusBadRequest)
		return
	}
	id := parts[len(parts)-1]
	pkMu.Lock()
	defer pkMu.Unlock()
	cred, ok := pkCredentials[id]
	if !ok {
		http.Error(w, `{"error":"credential not found"}`, http.StatusNotFound)
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
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
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