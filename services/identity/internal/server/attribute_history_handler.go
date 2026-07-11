package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AttributeChange tracks a user attribute modification for audit.
type AttributeChange struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedBy string    `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
}

var (
	attrHistoryMu sync.RWMutex
	attrHistory   = make(map[string][]*AttributeChange) // user_id → changes
)

// RecordAttributeChange logs a user attribute modification.
func RecordAttributeChange(userID, field, oldVal, newVal, changedBy string) {
	c := &AttributeChange{
		ID: uuid.New().String(), UserID: userID, Field: field,
		OldValue: oldVal, NewValue: newVal, ChangedBy: changedBy,
		ChangedAt: time.Now().UTC(),
	}
	attrHistoryMu.Lock()
	attrHistory[userID] = append(attrHistory[userID], c)
	attrHistoryMu.Unlock()
}

// GET /api/v1/users/{id}/attribute-history — returns attribute change history.
func (h *HTTPHandler) handleAttributeHistory(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	field := r.URL.Query().Get("field")
	limit := 100

	attrHistoryMu.RLock()
	changes := attrHistory[userID.String()]
	result := []*AttributeChange{}
	for i := len(changes) - 1; i >= 0 && len(result) < limit; i-- {
		if field != "" && changes[i].Field != field {
			continue
		}
		result = append(result, changes[i])
	}
	attrHistoryMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID.String(),
		"changes": result,
		"count":   len(result),
	})
}
