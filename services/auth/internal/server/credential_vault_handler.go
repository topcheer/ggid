package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type StoredCredential struct {
	Key       string    `json:"key"`
	Value     string    `json:"-"` // never expose ciphertext in JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	credVaultMu  sync.RWMutex
	credVault    = make(map[string]map[string]*StoredCredential) // userID -> key -> cred
	vaultAESKey  = []byte("0123456789abcdef0123456789abcdef")   // 32-byte AES-256 key
)

func encryptCredential(plaintext string) (string, error) {
	block, err := aes.NewCipher(vaultAESKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func decryptCredential(ciphertextB64 string) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(vaultAESKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(ct) < ns {
		return "", io.ErrUnexpectedEOF
	}
	pt, err := gcm.Open(nil, ct[:ns], ct[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// POST /api/v1/auth/credentials/store — store encrypted credential in per-user vault.
// GET /api/v1/auth/credentials/{key} — retrieve decrypted credential.
func (h *Handler) handleCredentialVault(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			UserID string `json:"user_id"`
			Key    string `json:"key"`
			Value  string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.UserID == "" || req.Key == "" || req.Value == "" {
			writeError(w, http.StatusBadRequest, "user_id, key, value required")
			return
		}
		encVal, err := encryptCredential(req.Value)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "encryption failed")
			return
		}
		now := time.Now().UTC()
		cred := &StoredCredential{Key: req.Key, Value: encVal, CreatedAt: now, UpdatedAt: now}
		credVaultMu.Lock()
		if credVault[req.UserID] == nil {
			credVault[req.UserID] = make(map[string]*StoredCredential)
		}
		credVault[req.UserID][req.Key] = cred
		credVaultMu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{
			"status": "stored", "user_id": req.UserID, "key": req.Key,
			"encryption": "AES-256-GCM", "stored_at": now.Format(time.RFC3339),
		})
	case http.MethodGet:
		key := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/credentials/")
		userID := r.URL.Query().Get("user_id")
		if key == "" || userID == "" {
			writeError(w, http.StatusBadRequest, "key and user_id required")
			return
		}
		credVaultMu.RLock()
		userVault, ok := credVault[userID]
		credVaultMu.RUnlock()
		if !ok {
			writeError(w, http.StatusNotFound, "vault not found")
			return
		}
		credVaultMu.RLock()
		cred, ok := userVault[key]
		credVaultMu.RUnlock()
		if !ok {
			writeError(w, http.StatusNotFound, "credential not found")
			return
		}
		decVal, err := decryptCredential(cred.Value)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "decryption failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"key": cred.Key, "value": decVal, "created_at": cred.CreatedAt,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
