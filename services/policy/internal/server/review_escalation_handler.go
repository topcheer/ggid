package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// escalationRecord represents an escalated access review.
type escalationRecord struct {
	ID           string `json:"id"`
	ReviewID     string `json:"review_id"`
	EscalatedTo  string `json:"escalated_to"`
	Reason       string `json:"reason"`
	EscalatedAt  string `json:"escalated_at"`
	Status       string `json:"status"` // escalated, resolved, re_escalated
	OriginalReviewer string `json:"original_reviewer"`
}

var escalationStore = struct {
	sync.RWMutex
	data map[string]*escalationRecord
}{data: make(map[string]*escalationRecord)}

// POST /api/v1/policies/access-reviews/escalate
// GET  /api/v1/policies/access-reviews/escalated
func (s *HTTPServer) handleReviewEscalation(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			ReviewID         string `json:"review_id"`
			EscalatedTo      string `json:"escalated_to"`
			Reason           string `json:"reason"`
			OriginalReviewer string `json:"original_reviewer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.ReviewID == "" || req.EscalatedTo == "" {
			writeJSONError(w, http.StatusBadRequest, "review_id and escalated_to are required")
			return
		}
		if req.Reason == "" {
			req.Reason = "timeout_no_response"
		}

		rec := &escalationRecord{
			ID: uuid.New().String(), ReviewID: req.ReviewID,
			EscalatedTo: req.EscalatedTo, Reason: req.Reason,
			OriginalReviewer: req.OriginalReviewer,
			Status: "escalated", EscalatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		escalationStore.Lock()
		escalationStore.data[rec.ID] = rec
		escalationStore.Unlock()

		writeJSON(w, http.StatusCreated, rec)

	case http.MethodGet:
		statusFilter := r.URL.Query().Get("status")

		escalationStore.RLock()
		result := []*escalationRecord{}
		for _, e := range escalationStore.data {
			if statusFilter != "" && e.Status != statusFilter {
				continue
			}
			result = append(result, e)
		}
		escalationStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"escalations": result,
			"total":       len(result),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
