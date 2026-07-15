package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SecurityIncident tracks a security incident.
type SecurityIncident struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	Severity         string    `json:"severity"` // low, medium, high, critical
	Type             string    `json:"type"`     // breach, anomaly, intrusion, etc.
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	AffectedUsers    []string  `json:"affected_users"`
	Status           string    `json:"status"` // open, investigating, resolved
	ResolutionNotes  string    `json:"resolution_notes,omitempty"`
	AssignedTo       string    `json:"assigned_to,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
}

var (
	incidentMu sync.RWMutex
	incidents  = make(map[string]*SecurityIncident)
)

// POST /api/v1/audit/incidents — create incident
// GET /api/v1/audit/incidents/active — list active incidents
// POST /api/v1/audit/incidents/{id}/resolve — resolve incident
func (s *HTTPServer) handleIncidents(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/incidents")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				TenantID string   `json:"tenant_id"`
				Severity string   `json:"severity"`
				Type     string   `json:"type"`
				Title    string   `json:"title"`
				Description string `json:"description"`
				AffectedUsers []string `json:"affected_users"`
				AssignedTo string `json:"assigned_to"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			if req.Title == "" {
				writeJSONError(w, http.StatusBadRequest, "title is required")
				return
			}
			if req.Severity == "" { req.Severity = "medium" }
			if req.Type == "" { req.Type = "anomaly" }
			inc := &SecurityIncident{
				ID: uuid.New().String(), TenantID: req.TenantID,
				Severity: req.Severity, Type: req.Type, Title: req.Title,
				Description: req.Description, AffectedUsers: req.AffectedUsers,
				Status: "open", AssignedTo: req.AssignedTo, CreatedAt: time.Now().UTC(),
			}
			incidentMu.Lock(); incidents[inc.ID] = inc; incidentMu.Unlock()
			writeJSON(w, http.StatusCreated, inc)
		case http.MethodGet:
			incidentMu.RLock()
			result := []*SecurityIncident{}
			for _, inc := range incidents {
				result = append(result, inc)
			}
			incidentMu.RUnlock()
			writeJSON(w, http.StatusOK, map[string]any{"incidents": result, "count": len(result)})
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 2 && parts[1] == "resolve" {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req struct {
			ResolutionNotes string `json:"resolution_notes"`
			ResolvedBy      string `json:"resolved_by"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid request body"); return }
		incidentMu.Lock()
		inc, ok := incidents[parts[0]]
		if !ok {
			incidentMu.Unlock()
			writeJSONError(w, http.StatusNotFound, "incident not found")
			return
		}
		now := time.Now().UTC()
		inc.Status = "resolved"
		inc.ResolutionNotes = req.ResolutionNotes
		inc.ResolvedAt = &now
		incidentMu.Unlock()
		writeJSON(w, http.StatusOK, inc)
		return
	}
	if len(parts) == 2 && parts[1] == "timeline" {
		s.handleIncidentTimeline(w, r)
		return
	}
	writeJSONError(w, http.StatusNotFound, "not found")
}

func (s *HTTPServer) handleIncidentsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	incidentMu.RLock()
	result := []*SecurityIncident{}
	for _, inc := range incidents {
		if inc.Status != "open" && inc.Status != "investigating" {
			continue
		}
		if tenantID != "" && inc.TenantID != tenantID {
			continue
		}
		result = append(result, inc)
	}
	incidentMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{"incidents": result, "count": len(result)})
}
