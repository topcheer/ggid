package server

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type AttributeChange struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedBy string    `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
}

func RecordAttributeChange(userID, field, oldVal, newVal, changedBy string) {
	c := &AttributeChange{
		ID: uuid.New().String(), UserID: userID, Field: field,
		OldValue: oldVal, NewValue: newVal, ChangedBy: changedBy,
		ChangedAt: time.Now().UTC(),
	}
	if globalIdentityMap != nil {
		globalIdentityMap.Store(nil, "identity_attribute_history", c.ID, map[string]any{
			"user_id": c.UserID, "field": c.Field, "old_value": c.OldValue,
			"new_value": c.NewValue, "changed_by": c.ChangedBy,
		})
	}
}

var globalIdentityMap *identityPolicyMapRepo

func SetGlobalIdentityMap(repo *identityPolicyMapRepo) {
	globalIdentityMap = repo
}

func (h *HTTPHandler) handleAttributeHistory(ctx context.Context, userID uuid.UUID, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	field := r.URL.Query().Get("field")
	var result []map[string]any
	if h.identityPolicyMap != nil {
		rows, _ := h.identityPolicyMap.List(r.Context(), "identity_attribute_history")
		for _, row := range rows {
			if getString(row, "user_id") != userID.String() {
				continue
			}
			if field != "" && getString(row, "field") != field {
				continue
			}
			result = append(result, row)
		}
	}
	if result == nil { result = []map[string]any{} }
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID.String(), "changes": result, "count": len(result)})
}
