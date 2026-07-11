package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ConsentReceipt is a GDPR-compliant record of user consent.
type ConsentReceipt struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ClientID       string    `json:"client_id"`
	Purpose        string    `json:"purpose"`
	DataCategories []string  `json:"data_categories"`
	ThirdParties   []string  `json:"third_parties"`
	Retention      string    `json:"retention"`
	WithdrawURL    string    `json:"withdraw_url"`
	GrantedAt      time.Time `json:"granted_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

var (
	consentReceiptMu sync.RWMutex
	consentReceipts  = make(map[string]*ConsentReceipt)
)

// RecordConsentReceipt stores a consent receipt (called during OAuth consent flow).
func RecordConsentReceipt(userID, clientID, purpose string, categories, thirdParties []string) *ConsentReceipt {
	r := &ConsentReceipt{
		ID:             uuid.New().String(),
		UserID:         userID,
		ClientID:       clientID,
		Purpose:        purpose,
		DataCategories: categories,
		ThirdParties:   thirdParties,
		Retention:      "365 days",
		WithdrawURL:    "/api/v1/oauth/consent/" + "withdraw",
		GrantedAt:      time.Now().UTC(),
	}
	expiry := time.Now().UTC().Add(365 * 24 * time.Hour)
	r.ExpiresAt = &expiry
	consentReceiptMu.Lock()
	consentReceipts[r.ID] = r
	consentReceiptMu.Unlock()
	return r
}

// GET /api/v1/oauth/consent/{id}/receipt — get GDPR-compliant consent receipt.
func handleConsentReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	// Extract consent ID from path: /api/v1/oauth/consent/{id}/receipt
	pathParts := splitPath(r.URL.Path)
	if len(pathParts) < 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "consent ID required"})
		return
	}
	consentID := pathParts[3]

	consentReceiptMu.RLock()
	rec, ok := consentReceipts[consentID]
	consentReceiptMu.RUnlock()

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "consent receipt not found"})
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

func splitPath(p string) []string {
	return strings.Split(strings.Trim(p, "/"), "/")
}
