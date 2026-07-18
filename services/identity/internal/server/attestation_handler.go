package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Attestation struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requested_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
}

func (h *HTTPHandler) handleAttest(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct{ ExpiryDays int `json:"expiry_days"` }
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	if req.ExpiryDays <= 0 { req.ExpiryDays = 30 }
	now := time.Now().UTC()
	a := &Attestation{
		ID: uuid.New().String(), UserID: userID.String(),
		Status: "pending", RequestedAt: now,
		ExpiresAt: now.Add(time.Duration(req.ExpiryDays) * 24 * time.Hour),
	}
	if h.identityPolicyMap != nil {
		h.identityPolicyMap.Store(r.Context(), "identity_attestations", a.ID, map[string]any{
			"user_id": a.UserID, "status": a.Status,
			"requested_at": a.RequestedAt, "expires_at": a.ExpiresAt,
		})
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "requested", "attestation_id": a.ID,
		"user_id": userID.String(), "expires_at": a.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *HTTPHandler) handleAttestationPending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var result []map[string]any
	if h.identityPolicyMap != nil {
		rows, _ := h.identityPolicyMap.List(r.Context(), "identity_attestations")
		for _, row := range rows {
			if getString(row, "status") == "pending" {
				result = append(result, row)
			}
		}
	}
	if result == nil { result = []map[string]any{} }
	writeJSON(w, http.StatusOK, map[string]any{"attestations": result, "count": len(result)})
}
