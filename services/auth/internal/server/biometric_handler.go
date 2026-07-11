package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BiometricTemplate stores an encrypted biometric template.
type BiometricTemplate struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	TemplateEnc   string    `json:"template_encrypted"`
	DeviceType    string    `json:"device_type"` // fingerprint, face, voice
	EnrolledAt    time.Time `json:"enrolled_at"`
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	VerifyCount   int       `json:"verify_count"`
}

var (
	biometricMu  sync.RWMutex
	biometrics   = make(map[string]*BiometricTemplate)
	bioKey       = []byte("32-byte-key-for-aes-256-enc!!") // production: from KMS
)

// POST /api/v1/auth/biometric/enroll — store encrypted biometric template.
// POST /api/v1/auth/biometric/verify — verify biometric against stored template.
func (h *Handler) handleBiometricEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID      string `json:"user_id"`
		TemplateB64 string `json:"template"` // base64-encoded raw template
		DeviceType  string `json:"device_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.UserID == "" || req.TemplateB64 == "" {
		writeError(w, http.StatusBadRequest, "user_id and template are required")
		return
	}
	if req.DeviceType == "" {
		req.DeviceType = "fingerprint"
	}

	// Encrypt template
	rawTemplate, err := base64.StdEncoding.DecodeString(req.TemplateB64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template encoding")
		return
	}
	encTemplate, err := encryptAESGCM(bioKey, rawTemplate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encryption failed")
		return
	}

 tmpl := &BiometricTemplate{
		ID:          uuid.New().String(),
		UserID:      req.UserID,
		TemplateEnc: base64.StdEncoding.EncodeToString(encTemplate),
		DeviceType:  req.DeviceType,
		EnrolledAt:  time.Now().UTC(),
	}
	biometricMu.Lock()
	biometrics[tmpl.ID] = tmpl
	biometricMu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":       "enrolled",
		"template_id":  tmpl.ID,
		"device_type":  tmpl.DeviceType,
		"enrolled_at":  tmpl.EnrolledAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleBiometricVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID      string `json:"user_id"`
		TemplateB64 string `json:"template"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.UserID == "" || req.TemplateB64 == "" {
		writeError(w, http.StatusBadRequest, "user_id and template are required")
		return
	}

	// Find template for user
	biometricMu.Lock()
	defer biometricMu.Unlock()

	var found *BiometricTemplate
	for _, t := range biometrics {
		if t.UserID == req.UserID {
			found = t
			break
		}
	}
	if found == nil {
		writeError(w, http.StatusNotFound, "no biometric template enrolled")
		return
	}

	// Decrypt stored template
	encData, _ := base64.StdEncoding.DecodeString(found.TemplateEnc)
	stored, err := decryptAESGCM(bioKey, encData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "decryption failed")
		return
	}

	// Compare (in production: use fuzzy matching, not exact)
	provided, _ := base64.StdEncoding.DecodeString(req.TemplateB64)
	match := len(stored) == len(provided)
	if match {
		for i := range stored {
			if stored[i] != provided[i] {
				match = false
				break
			}
		}
	}

	if !match {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"verified": false,
			"reason":   "template_mismatch",
		})
		return
	}

	now := time.Now().UTC()
	found.VerifiedAt = &now
	found.VerifyCount++

	writeJSON(w, http.StatusOK, map[string]any{
		"verified":      true,
		"template_id":   found.ID,
		"device_type":   found.DeviceType,
		"verify_count":  found.VerifyCount,
		"verified_at":   now.Format(time.RFC3339),
	})
}

func encryptAESGCM(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptAESGCM(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, io.ErrUnexpectedEOF
	}
	return gcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
}
