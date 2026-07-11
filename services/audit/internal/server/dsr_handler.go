package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type DSRRequest struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"` // access, erasure, portability, rectification
	UserID     string    `json:"user_id"`
	Status     string    `json:"status"` // pending, processing, completed, rejected
	DueDate    time.Time `json:"due_date"`
	CreatedAt  time.Time `json:"created_at"`
}

var (
	dsrMu sync.RWMutex
	dsrRequests = make(map[string]*DSRRequest)
)

// POST/GET /api/v1/audit/dsr
func (s *HTTPServer) handleDSR(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Type   string `json:"type"`
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return
		}
		if req.Type == "" || req.UserID == "" {
			writeJSONError(w, http.StatusBadRequest, "type and user_id required"); return
		}
		now := time.Now().UTC()
		dsr := &DSRRequest{ID: "dsr-" + uuid.New().String()[:8], Type: req.Type, UserID: req.UserID, Status: "pending", DueDate: now.Add(30 * 24 * time.Hour), CreatedAt: now}
		dsrMu.Lock(); dsrRequests[dsr.ID] = dsr; dsrMu.Unlock()
		writeJSON(w, http.StatusCreated, dsr)
	case http.MethodGet:
		dsrMu.RLock()
		result := make([]*DSRRequest, 0, len(dsrRequests))
		for _, d := range dsrRequests { result = append(result, d) }
		dsrMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"dsr_requests": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
