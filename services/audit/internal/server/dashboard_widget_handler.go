package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DashboardWidget defines a custom dashboard widget configuration.
type DashboardWidget struct {
	ID              string    `json:"id"`
	DashboardID     string    `json:"dashboard_id"`
	Title           string    `json:"title"`
	Query           string    `json:"query"`
	ChartType       string    `json:"chart_type"` // line, bar, pie, table, gauge
	RefreshInterval int       `json:"refresh_interval"` // seconds
	Position        int       `json:"position"`
	CreatedAt       time.Time `json:"created_at"`
}

type widgetStore struct {
	mu      sync.RWMutex
	widgets map[string]*DashboardWidget
}

var dashboardWidgets = &widgetStore{widgets: make(map[string]*DashboardWidget)}

// POST/GET/DELETE /api/v1/audit/dashboards/{id}/widgets
func (s *HTTPServer) handleDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	// Extract dashboard_id from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/dashboards/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "widgets" {
		writeJSONError(w, http.StatusBadRequest, "invalid path")
		return
	}
	dashboardID := parts[0]
	widgetID := r.URL.Query().Get("widget_id")

	switch r.Method {
	case http.MethodPost:
		var req struct {
			Title           string `json:"title"`
			Query           string `json:"query"`
			ChartType       string `json:"chart_type"`
			RefreshInterval int    `json:"refresh_interval"`
			Position        int    `json:"position"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Title == "" {
			writeJSONError(w, http.StatusBadRequest, "title is required")
			return
		}
		if req.ChartType == "" {
			req.ChartType = "line"
		}
		if req.RefreshInterval <= 0 {
			req.RefreshInterval = 60
		}

		widget := &DashboardWidget{
			ID:              uuid.New().String(),
			DashboardID:     dashboardID,
			Title:           req.Title,
			Query:           req.Query,
			ChartType:       req.ChartType,
			RefreshInterval: req.RefreshInterval,
			Position:        req.Position,
			CreatedAt:       time.Now().UTC(),
		}

		dashboardWidgets.mu.Lock()
		dashboardWidgets.widgets[widget.ID] = widget
		dashboardWidgets.mu.Unlock()

		// Return widget config + simulated live data
		writeJSON(w, http.StatusCreated, map[string]any{
			"widget": widget,
			"live_data": map[string]any{
				"labels": []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00"},
				"values": []int{12, 19, 45, 67, 82, 56},
			},
		})

	case http.MethodGet:
		dashboardWidgets.mu.RLock()
		result := []*DashboardWidget{}
		for _, wgt := range dashboardWidgets.widgets {
			if wgt.DashboardID != dashboardID {
				continue
			}
			result = append(result, wgt)
		}
		dashboardWidgets.mu.RUnlock()

		// Include live data for each widget
		widgetsWithData := make([]map[string]any, 0, len(result))
		for _, wgt := range result {
			widgetsWithData = append(widgetsWithData, map[string]any{
				"widget": wgt,
				"live_data": map[string]any{
					"labels": []string{"00:00", "04:00", "08:00", "12:00", "16:00", "20:00"},
					"values": []int{12, 19, 45, 67, 82, 56},
				},
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"widgets": widgetsWithData,
			"count":   len(widgetsWithData),
		})

	case http.MethodDelete:
		if widgetID == "" {
			writeJSONError(w, http.StatusBadRequest, "widget_id is required")
			return
		}
		dashboardWidgets.mu.Lock()
		if _, ok := dashboardWidgets.widgets[widgetID]; !ok {
			dashboardWidgets.mu.Unlock()
			writeJSONError(w, http.StatusNotFound, "widget not found")
			return
		}
		delete(dashboardWidgets.widgets, widgetID)
		dashboardWidgets.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "widget_id": widgetID})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
