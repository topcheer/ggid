package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
)

// GET /api/v1/audit/activity
// Returns recent audit events as an activity feed for the frontend activity page.
// Routed via gateway /api/v1/activity prefix → audit service (rewritten to /api/v1/audit/activity).
func (s *HTTPServer) handleActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pageSize := 50
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 500 {
			pageSize = v
		}
	}

	// SECURITY: Use validated tenant from header, not query param.
	tenantUUID, ok := resolveValidatedTenant(w, r)
	if !ok {
		return
	}
	if s.svc != nil {
		ctx := r.Context()
		since := time.Now().Add(-7 * 24 * time.Hour)
		filter := domain.ListFilter{
			TenantID:   tenantUUID,
			StartTime:  &since,
			Descending: true,
		}
		events, total, err := s.svc.ListEvents(ctx, filter, 1, pageSize)
		if err == nil {
			items := make([]map[string]any, 0, len(events))
			for _, e := range events {
				items = append(items, map[string]any{
					"id":            e.ID,
					"action":        string(e.Action),
					"actor_type":    string(e.ActorType),
					"actor_name":    e.ActorName,
					"resource_type": e.ResourceType,
					"result":        string(e.Result),
					"ip_address":    e.IPAddress,
					"created_at":    e.CreatedAt,
				})
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"activity": items,
				"total":    total,
			})
			return
		}
	}

	// Fallback: return empty activity list
	writeJSON(w, http.StatusOK, map[string]any{
		"activity": []map[string]any{},
		"total":    0,
	})
}
