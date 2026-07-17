package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DashboardWidget defines a custom dashboard widget configuration.
type DashboardWidget struct {
	ID              string    `json:"id"`
	DashboardID     string    `json:"dashboard_id"`
	Title           string    `json:"title"`
	Query           string    `json:"query"`
	ChartType       string    `json:"chart_type"`
	RefreshInterval int       `json:"refresh_interval"`
	Position        int       `json:"position"`
	CreatedAt       time.Time `json:"created_at"`
}

func (s *HTTPServer) handleDashboardWidgets(w http.ResponseWriter, r *http.Request) {
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
		if req.ChartType == "" { req.ChartType = "line" }
		if req.RefreshInterval <= 0 { req.RefreshInterval = 60 }
		widget := &DashboardWidget{
			ID: uuid.New().String(), DashboardID: dashboardID, Title: req.Title,
			Query: req.Query, ChartType: req.ChartType,
			RefreshInterval: req.RefreshInterval, Position: req.Position,
			CreatedAt: time.Now().UTC(),
		}
		if s.memMapRepo2 != nil {
			s.memMapRepo2.StoreJSON(r.Context(), "dashboard_widgets", widget.ID, map[string]any{
				"dashboard_id": widget.DashboardID, "title": widget.Title,
				"query": widget.Query, "chart_type": widget.ChartType,
				"refresh_interval": widget.RefreshInterval, "position": widget.Position,
			})
		}
		writeJSON(w, http.StatusCreated, map[string]any{"widget": widget})

	case http.MethodGet:
		var result []*DashboardWidget
		if s.memMapRepo2 != nil {
			rows, _ := s.memMapRepo2.ListJSON(r.Context(), "dashboard_widgets")
			for _, row := range rows {
				if amGetString(row, "dashboard_id") != dashboardID {
					continue
				}
				result = append(result, &DashboardWidget{
					ID: amGetString(row, "id"), DashboardID: amGetString(row, "dashboard_id"),
					Title: amGetString(row, "title"), Query: amGetString(row, "query"),
					ChartType: amGetString(row, "chart_type"),
				})
			}
		}
		if result == nil { result = []*DashboardWidget{} }
		writeJSON(w, http.StatusOK, map[string]any{"widgets": result, "count": len(result)})

	case http.MethodDelete:
		if widgetID == "" {
			writeJSONError(w, http.StatusBadRequest, "widget_id is required")
			return
		}
		if s.memMapRepo2 != nil {
			s.memMapRepo2.DeleteJSON(r.Context(), "dashboard_widgets", widgetID)
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "widget_id": widgetID})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func amGetString(m map[string]any, key string) string {
	if v, ok := m[key]; ok { return fmt.Sprintf("%v", v) }
	return ""
}

func amGetBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case bool: return val
		case string: return val == "true"
		}
	}
	return false
}
