package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Attestation tracks periodic user profile attestation.
type Attestation struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Status     string    `json:"status"` // pending, confirmed, declined
	RequestedAt time.Time `json:"requested_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
}

var (
	attestationMu sync.RWMutex
	attestations  = make(map[string]*Attestation)
)

// POST /api/v1/users/{id}/attest — request profile attestation.
// GET /api/v1/users/attestation/pending — list pending attestations.
// The POST /attest endpoint is called from handleUserByID sub-path routing.
func (h *HTTPHandler) handleAttest(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		ExpiryDays int `json:"expiry_days"`
	}
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	if req.ExpiryDays <= 0 {
		req.ExpiryDays = 30
	}

	now := time.Now().UTC()
	a := &Attestation{
		ID: uuid.New().String(), UserID: userID.String(),
		Status: "pending", RequestedAt: now,
		ExpiresAt: now.Add(time.Duration(req.ExpiryDays) * 24 * time.Hour),
	}
	attestationMu.Lock()
	attestations[a.ID] = a
	attestationMu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":       "requested",
		"attestation_id": a.ID,
		"user_id":      userID.String(),
		"expires_at":   a.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *HTTPHandler) handleAttestationPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()
	attestationMu.RLock()
	result := []*Attestation{}
	for _, a := range attestations {
		if a.Status != "pending" {
			continue
		}
		if now.After(a.ExpiresAt) {
			a.Status = "expired"
			continue
		}
		result = append(result, a)
	}
	attestationMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"attestations": result,
		"count":        len(result),
	})
}

// CompleteAttestation marks an attestation as confirmed.
func CompleteAttestation(attestationID, status string) bool {
	attestationMu.Lock()
	defer attestationMu.Unlock()
	a, ok := attestations[attestationID]
	if !ok {
		return false
	}
	now := time.Now().UTC()
	a.Status = status
	a.CompletedAt = &now
	return true
}
