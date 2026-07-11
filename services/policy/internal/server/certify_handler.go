package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Certification struct {
	ID          string    `json:"id"`
	CampaignID  string    `json:"campaign_id"`
	ReviewerID  string    `json:"reviewer_id"`
	UserID      string    `json:"user_id"`
	Decision    string    `json:"decision"` // certify, revoke, modify
	Comment     string    `json:"comment"`
	CertifiedAt time.Time `json:"certified_at"`
}

var (
	certMu  sync.RWMutex
	certs   = make(map[string]*Certification)
)

// POST /api/v1/policies/access-reviews/{campaign_id}/certify
func (s *HTTPServer) handleCertify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	campaignID := ""
	if len(parts) >= 5 {
		campaignID = parts[4]
	}
	if campaignID == "" {
		writeJSONError(w, http.StatusBadRequest, "campaign_id required")
		return
	}
	var req struct {
		ReviewerID string `json:"reviewer_id"`
		UserID     string `json:"user_id"`
		Decision   string `json:"decision"`
		Comment    string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Decision == "" {
		writeJSONError(w, http.StatusBadRequest, "decision required")
		return
	}
	cert := &Certification{
		ID: uuid.New().String(), CampaignID: campaignID, ReviewerID: req.ReviewerID,
		UserID: req.UserID, Decision: req.Decision, Comment: req.Comment,
		CertifiedAt: time.Now().UTC(),
	}
	certMu.Lock(); certs[cert.ID] = cert; certMu.Unlock()
	writeJSON(w, http.StatusCreated, cert)
}
