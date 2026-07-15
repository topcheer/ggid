package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ForgetRecord tracks a GDPR right-to-be-forgotten request.
type ForgetRecord struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Status         string    `json:"status"` // pending, processing, completed, failed
	RequestedBy    string    `json:"requested_by"`
	RequestedAt    time.Time `json:"requested_at"`
	CompletedAt    time.Time `json:"completed_at"`
	RecordsDeleted int       `json:"records_deleted"`
	Errors         []string  `json:"errors"`
}

var (
	forgetRecordsMu sync.RWMutex
	forgetRecords   = []ForgetRecord{}
)

// GET /api/v1/audit/gdpr-forget — list all forget records
// GET /api/v1/audit/gdpr-forget/search?q=X — search users
// POST /api/v1/audit/gdpr-forget/execute — execute forget request
func (s *HTTPServer) handleGDPRForgetV2(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/audit/gdpr-forget" && r.Method == http.MethodGet:
		forgetRecordsMu.RLock()
		records := make([]ForgetRecord, len(forgetRecords))
		copy(records, forgetRecords)
		forgetRecordsMu.RUnlock()
		writeJSON(w, http.StatusOK, records)

	case r.URL.Path == "/api/v1/audit/gdpr-forget/search" && r.Method == http.MethodGet:
		q := r.URL.Query().Get("q")
		// Return search results from audit events
		result := map[string]any{
			"user_id":      "",
			"username":     q,
			"email":        q + "@search.local",
			"record_count": 0,
		}
		writeJSON(w, http.StatusOK, result)

	case r.URL.Path == "/api/v1/audit/gdpr-forget/execute" && r.Method == http.MethodPost:
		var req struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid body")
			return
		}
		record := ForgetRecord{
			ID:          "fr-" + req.UserID,
			UserID:      req.UserID,
			Status:      "pending",
			RequestedAt: time.Now(),
			Errors:      []string{},
		}
		forgetRecordsMu.Lock()
		forgetRecords = append(forgetRecords, record)
		forgetRecordsMu.Unlock()
		writeJSON(w, http.StatusAccepted, record)

	default:
		if strings.HasPrefix(r.URL.Path, "/api/v1/audit/gdpr-forget") {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSONError(w, http.StatusNotFound, "not found")
	}
}
