package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Delegation struct {
	ID         string    `json:"id"`
	DelegatedTo string   `json:"delegated_to"`
	Scope      []string  `json:"scope"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	delegationMu sync.RWMutex
	delegations  = make(map[string][]*Delegation) // userID → delegations
)

// GET/POST /api/v1/users/{id}/delegations
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
			writeError(w, http.StatusBadRequest, "invalid JSON"); return
		}
		if req.DelegatedTo == "" {
			writeError(w, http.StatusBadRequest, "delegated_to required"); return
		}
		now := time.Now().UTC()
		end, _ := time.Parse(time.RFC3339, req.EndDate)
		if end.IsZero() { end = now.Add(7 * 24 * time.Hour) }
		start, _ := time.Parse(time.RFC3339, req.StartDate)
		if start.IsZero() { start = now }
		d := &Delegation{ID: "dlg-" + uuid.New().String()[:8], DelegatedTo: req.DelegatedTo, Scope: req.Scope, StartDate: start, EndDate: end, Status: "active", CreatedAt: now}
		delegationMu.Lock(); delegations[uid] = append(delegations[uid], d); delegationMu.Unlock()
		writeJSON(w, http.StatusCreated, d)
	case http.MethodGet:
		delegationMu.RLock()
		result := delegations[uid]
		if result == nil { result = []*Delegation{} }
		delegationMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"delegations": result, "count": len(result)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
