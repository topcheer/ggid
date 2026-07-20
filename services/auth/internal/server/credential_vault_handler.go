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
	vaultAESKey  = loadEncryptionKey("CRED_VAULT_AES_KEY")
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

		// DB persistence
		if h.pool != nil {
			vaultID := req.UserID + ":" + req.Key
			_, _ = h.pool.Exec(r.Context(), `
				INSERT INTO auth_credential_vault (id, user_id, cred_key, encrypted_value, cred_type)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (id) DO UPDATE SET encrypted_value = $4, updated_at = NOW()`,
				vaultID, req.UserID, req.Key, encVal, "secret")
		} else if h.memMapRepo != nil {
			vaultID := req.UserID + ":" + req.Key
			h.memMapRepo.StoreJSON(r.Context(), "auth_credvault_json", vaultID, map[string]any{
				"id": vaultID, "user_id": req.UserID,
				"cred_key": req.Key, "encrypted_value": encVal,
				"created_at": now, "updated_at": now,
			})
		}

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
		// Try DB first, then PG memMap, fall back to in-memory.
		var cred *StoredCredential
		if h.pool != nil {
			var encVal string
			err := h.pool.QueryRow(r.Context(), `
				SELECT encrypted_value FROM auth_credential_vault WHERE user_id = $1 AND cred_key = $2`,
				userID, key).Scan(&encVal)
			if err == nil && encVal != "" {
				cred = &StoredCredential{Key: key, Value: encVal}
			}
		}
		if cred == nil && h.memMapRepo != nil {
			vaultID := userID + ":" + key
			if row, _ := h.memMapRepo.GetJSON(r.Context(), "auth_credvault_json", vaultID); row != nil {
				cred = &StoredCredential{
					Key:   getString(row, "cred_key"),
					Value: getString(row, "encrypted_value"),
				}
			}
		}
		if cred == nil {
			credVaultMu.RLock()
			userVault, ok := credVault[userID]
			credVaultMu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "vault not found")
				return
			}
			credVaultMu.RLock()
			c, ok := userVault[key]
			credVaultMu.RUnlock()
			if !ok {
				writeError(w, http.StatusNotFound, "credential not found")
				return
			}
			cred = c
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
