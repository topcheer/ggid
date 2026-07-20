package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SecurityIncident tracks a security incident.
type SecurityIncident struct {
	ID              string     `json:"id"`
	TenantID        string     `json:"tenant_id"`
	Severity        string     `json:"severity"`
	Type            string     `json:"type"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	AffectedUsers   []string   `json:"affected_users"`
	Status          string     `json:"status"`
	ResolutionNotes string     `json:"resolution_notes,omitempty"`
	AssignedTo      string     `json:"assigned_to,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

func incidentToMap(inc *SecurityIncident) map[string]any {
	return map[string]any{
		"id":               inc.ID,
		"tenant_id":        inc.TenantID,
		"severity":         inc.Severity,
		"type":             inc.Type,
		"title":            inc.Title,
		"description":      inc.Description,
		"affected_users":   inc.AffectedUsers,
		"status":           inc.Status,
		"resolution_notes": inc.ResolutionNotes,
		"assigned_to":      inc.AssignedTo,
	}
}

func mapToIncident(row map[string]any) *SecurityIncident {
	inc := &SecurityIncident{}
	inc.ID = amGetString(row, "id")
	inc.TenantID = amGetString(row, "tenant_id")
	inc.Severity = amGetString(row, "severity")
	inc.Type = amGetString(row, "type")
	inc.Title = amGetString(row, "title")
	inc.Description = amGetString(row, "description")
	inc.Status = amGetString(row, "status")
	inc.ResolutionNotes = amGetString(row, "resolution_notes")
	inc.AssignedTo = amGetString(row, "assigned_to")
	if raw, ok := row["affected_users"].([]any); ok {
		for _, a := range raw {
			if s, ok := a.(string); ok {
				inc.AffectedUsers = append(inc.AffectedUsers, s)
			}
		}
	}
	return inc
}

// incidentListFromDB reads all incidents from the audit_incidents table.
func (s *HTTPServer) incidentListFromDB(r *http.Request) []*SecurityIncident {
	if s.pool == nil {
		return nil
	}
	rows, err := s.pool.Query(r.Context(), `SELECT data::text FROM audit_incidents ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []*SecurityIncident
	for rows.Next() {
		var dataJSON string
		if err := rows.Scan(&dataJSON); err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(dataJSON), &m) == nil {
			result = append(result, mapToIncident(m))
		}
	}
	return result
}

// incidentGetFromDB reads a single incident from DB.
func (s *HTTPServer) incidentGetFromDB(r *http.Request, id string) *SecurityIncident {
	if s.pool == nil {
		return nil
	}
	var dataJSON string
	err := s.pool.QueryRow(r.Context(), `SELECT data::text FROM audit_incidents WHERE id = $1`, id).Scan(&dataJSON)
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal([]byte(dataJSON), &m) != nil {
		return nil
	}
	return mapToIncident(m)
}

// incidentSaveDB writes an incident to the audit_incidents table.
func (s *HTTPServer) incidentSaveDB(r *http.Request, inc *SecurityIncident) bool {
	if s.pool == nil {
		return false
	}
	dataBytes, _ := json.Marshal(incidentToMap(inc))
	_, err := s.pool.Exec(r.Context(),
		`INSERT INTO audit_incidents (id, data) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET data = $2`,
		inc.ID, dataBytes)
	return err == nil
}

// POST /api/v1/audit/incidents — create incident
// GET /api/v1/audit/incidents/active — list active incidents
// POST /api/v1/audit/incidents/{id}/resolve — resolve incident
func (s *HTTPServer) handleIncidents(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/incidents")

	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				TenantID      string   `json:"tenant_id"`
				Severity      string   `json:"severity"`
				Type          string   `json:"type"`
				Title         string   `json:"title"`
				Description   string   `json:"description"`
				AffectedUsers []string `json:"affected_users"`
				AssignedTo    string   `json:"assigned_to"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
			if req.Title == "" {
				writeJSONError(w, http.StatusBadRequest, "title is required")
				return
			}
			if req.Severity == "" {
				req.Severity = "medium"
			}
			if req.Type == "" {
				req.Type = "anomaly"
			}
			inc := &SecurityIncident{
				ID: uuid.New().String(), TenantID: req.TenantID,
				Severity: req.Severity, Type: req.Type, Title: req.Title,
				Description: req.Description, AffectedUsers: req.AffectedUsers,
				Status: "open", AssignedTo: req.AssignedTo, CreatedAt: time.Now().UTC(),
			}
			// Save to DB (primary) + memMapRepo (fallback cache)
			s.incidentSaveDB(r, inc)
			if s.memMapRepo2 != nil {
				s.memMapRepo2.StoreJSON(r.Context(), "audit_incidents", inc.ID, incidentToMap(inc))
			}
			writeJSON(w, http.StatusCreated, inc)
		case http.MethodGet:
			// Try DB first
			result := s.incidentListFromDB(r)
			if result == nil && s.memMapRepo2 != nil {
				// Fallback to memMapRepo
				rows, _ := s.memMapRepo2.ListJSON(r.Context(), "audit_incidents")
				for _, row := range rows {
					result = append(result, mapToIncident(row))
				}
			}
			if result == nil {
				result = []*SecurityIncident{}
			}
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Try DB first, then memMapRepo
		inc := s.incidentGetFromDB(r, parts[0])
		if inc == nil && s.memMapRepo2 != nil {
			rows, _ := s.memMapRepo2.ListJSON(r.Context(), "audit_incidents")
			for _, row := range rows {
				if amGetString(row, "id") == parts[0] {
					inc = mapToIncident(row)
					break
				}
			}
		}
		if inc == nil {
			writeJSONError(w, http.StatusNotFound, "incident not found")
			return
		}
		now := time.Now().UTC()
		inc.Status = "resolved"
		inc.ResolutionNotes = req.ResolutionNotes
		inc.ResolvedAt = &now
		s.incidentSaveDB(r, inc)
		if s.memMapRepo2 != nil {
			s.memMapRepo2.StoreJSON(r.Context(), "audit_incidents", inc.ID, incidentToMap(inc))
		}
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

	// Try DB first
	all := s.incidentListFromDB(r)
	if all == nil && s.memMapRepo2 != nil {
		rows, _ := s.memMapRepo2.ListJSON(r.Context(), "audit_incidents")
		for _, row := range rows {
			all = append(all, mapToIncident(row))
		}
	}

	result := []*SecurityIncident{}
	for _, inc := range all {
		if inc.Status != "open" && inc.Status != "investigating" {
			continue
		}
		if tenantID != "" && inc.TenantID != tenantID {
			continue
		}
		result = append(result, inc)
	}
	writeJSON(w, http.StatusOK, map[string]any{"incidents": result, "count": len(result)})
}
