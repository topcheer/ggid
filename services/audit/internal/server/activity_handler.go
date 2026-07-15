package httpserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
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

	// Aggregate recent events from audit service if available
	if s.svc != nil {
		tenantIDStr := r.URL.Query().Get("tenant_id")
		if tenantIDStr != "" {
			tenantID, err := uuid.Parse(tenantIDStr)
			if err == nil {
				ctx := r.Context()
				since := time.Now().Add(-7 * 24 * time.Hour) // last 7 days
				filter := domain.ListFilter{
					TenantID:   tenantID,
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
		}
	}

	// Fallback: return empty activity list
	writeJSON(w, http.StatusOK, map[string]any{
		"activity": []map[string]any{},
		"total":    0,
	})
}
