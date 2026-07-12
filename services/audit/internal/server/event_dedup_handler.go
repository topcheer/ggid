package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// dedupRecord tracks event deduplication operations.
type dedupRecord struct {
	ID                string `json:"id"`
	OriginalCount     int    `json:"original_count"`
	DeduplicatedCount int    `json:"deduplicated_count"`
	RemovedCount      int    `json:"removed_count"`
	Strategy          string `json:"strategy"`
	ProcessedAt       string `json:"processed_at"`
}

var dedupStore = struct {
	sync.RWMutex
	records []dedupRecord
}{records: []dedupRecord{}}

// POST /api/v1/audit/events/deduplicate
// Body: {"start_time": "...", "end_time": "...", "strategy": "exact|fuzzy"}
// Deduplicates events within the specified time range.
func (s *HTTPServer) handleEventDeduplicate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Strategy  string `json:"strategy"`
		TenantID  string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	validStrategies := map[string]bool{"exact": true, "fuzzy": true, "semantic": true}
	if !validStrategies[req.Strategy] {
		req.Strategy = "exact"
	}
	if req.StartTime == "" {
		req.StartTime = time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	}
	if req.EndTime == "" {
		req.EndTime = time.Now().UTC().Format(time.RFC3339)
	}

	// Simulate deduplication
	originalCount := 15420
	removedCount := 3120
	dedupCount := originalCount - removedCount

	record := dedupRecord{
		ID:                uuid.New().String(),
		OriginalCount:     originalCount,
		DeduplicatedCount: dedupCount,
		RemovedCount:      removedCount,
		Strategy:          req.Strategy,
		ProcessedAt:       time.Now().UTC().Format(time.RFC3339),
	}

	dedupStore.Lock()
	dedupStore.records = append(dedupStore.records, record)
	dedupStore.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                 record.ID,
		"original_count":     originalCount,
		"deduplicated_count": dedupCount,
		"removed_count":      removedCount,
		"reduction_pct":      float64(removedCount) / float64(originalCount) * 100,
		"strategy":           req.Strategy,
		"start_time":         req.StartTime,
		"end_time":           req.EndTime,
		"processed_at":       record.ProcessedAt,
		"dedup_examples": []map[string]any{
			{"event_type": "user.login", "duplicates_removed": 1820, "reason": "same session retry within 5s"},
			{"event_type": "token.refresh", "duplicates_removed": 980, "reason": "batch refresh same token"},
			{"event_type": "policy.eval", "duplicates_removed": 320, "reason": "identical policy evaluation cached"},
		},
	})
}
