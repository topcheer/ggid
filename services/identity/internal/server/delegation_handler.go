package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Delegation struct {
	ID          string    `json:"id"`
	DelegatedTo string    `json:"delegated_to"`
	Scope       []string  `json:"scope"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *HTTPHandler) handleDelegations(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	uid := userID.String()
	switch r.Method {
	case http.MethodPost:
		var req struct {
			DelegatedTo string   `json:"delegated_to"`
			Scope       []string `json:"scope"`
			StartDate   string   `json:"start_date"`
			EndDate     string   `json:"end_date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.DelegatedTo == "" {
			writeJSONError(w, http.StatusBadRequest, "delegated_to required")
			return
		}
		now := time.Now().UTC()
		end, _ := time.Parse(time.RFC3339, req.EndDate)
		if end.IsZero() { end = now.Add(7 * 24 * time.Hour) }
		start, _ := time.Parse(time.RFC3339, req.StartDate)
		if start.IsZero() { start = now }
		d := &Delegation{ID: "dlg-" + uuid.New().String()[:8], DelegatedTo: req.DelegatedTo, Scope: req.Scope, StartDate: start, EndDate: end, Status: "active", CreatedAt: now}
		if h.identityPolicyMap != nil {
			h.identityPolicyMap.Store(r.Context(), "identity_delegations", d.ID, map[string]any{
				"user_id": uid, "delegated_to": d.DelegatedTo, "scope": d.Scope,
				"start_date": d.StartDate, "end_date": d.EndDate, "status": d.Status,
			})
		}
		writeJSON(w, http.StatusCreated, d)
	case http.MethodGet:
		var result []map[string]any
		if h.identityPolicyMap != nil {
			rows, _ := h.identityPolicyMap.List(r.Context(), "identity_delegations")
			for _, row := range rows {
				if getString(row, "user_id") == uid {
					result = append(result, row)
				}
			}
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"delegations": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
