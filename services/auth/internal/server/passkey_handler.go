package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
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
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	pkMu.Lock()
	defer pkMu.Unlock()
	sess, ok := pkRegSessions[req.SessionID]
	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}
	pkSeq++
	cred := &PasskeyCredential{
		ID:        req.Credential.ID,
		UserID:    sess.UserID,
		PublicKey: req.Credential.PublicKey,
		CreatedAt: time.Now(),
	}
	pkCredentials[cred.ID] = cred
	sess.Status = "completed"
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
	sess, ok := pkAuthSessions[req.SessionID]
	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}
	cred, ok := pkCredentials[req.CredentialID]
	if !ok || cred.Revoked {
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