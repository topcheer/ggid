package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CompositeRule defines a multi-signal detection rule (N signals × time window → critical).
type CompositeRule struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Signals     []string       `json:"signals"`     // ITDR rule IDs that must trigger
	MinSignals  int            `json:"min_signals"`  // threshold (e.g., 3 of 5 signals)
	WindowMin   int            `json:"window_minutes"` // time window for correlation
	Severity    string         `json:"severity"`     // resulting severity
	Actions     []string       `json:"actions"`      // auto-response (isolate, alert, block)
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
}

// IncidentListEntry is a lightweight incident for list view.
type IncidentListEntry struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Severity     string         `json:"severity"`
	Status       string         `json:"status"`
	TriggeredRules []string     `json:"triggered_rules"`
	UserIDs      []string       `json:"user_ids,omitempty"`
	IPAddresses  []string       `json:"ip_addresses,omitempty"`
	DetectionCount int          `json:"detection_count"`
	FirstDetected time.Time     `json:"first_detected"`
	LastUpdated   time.Time     `json:"last_updated"`
	Timeline     []TimelineEntry `json:"timeline,omitempty"`
}

type TimelineEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	Source    string    `json:"source"`
	Detail    string    `json:"detail,omitempty"`
}

// Playbook defines automated incident response.
type Playbook struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Trigger     string         `json:"trigger"` // rule ID or severity
	Steps       []PlaybookStep `json:"steps"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
}

type PlaybookStep struct {
	Order   int    `json:"order"`
	Action  string `json:"action"` // isolate_user, revoke_sessions, notify_soc, disable_account
	Target  string `json:"target"`
	Delay   int    `json:"delay_seconds,omitempty"`
}

// ThreatHeatmapEntry represents a cell in the user×resource detection heatmap.
type ThreatHeatmapEntry struct {
	UserID       string `json:"user_id"`
	ResourceType string `json:"resource_type"`
	Detections   int    `json:"detections"`
	Severity     string `json:"severity"`
}

var (
	compositeRulesMu  sync.RWMutex
	compositeRules    = map[string]*CompositeRule{}
	itdrIncidentsMu       sync.RWMutex
	itdrIncidents      = map[string]*IncidentListEntry{}
	playbooksMu       sync.RWMutex
	playbooks         = map[string]*Playbook{}
)

// handleITDRComposite handles composite detection rules CRUD.
func (s *HTTPServer) handleITDRComposite(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var result []*CompositeRule
		if s.compositeRepo != nil {
			rules, _ := s.compositeRepo.List(r.Context())
			result = rules
		} else {
			compositeRulesMu.RLock()
			for _, cr := range compositeRules {
				result = append(result, cr)
			}
			compositeRulesMu.RUnlock()
		}
		if result == nil {
			result = []*CompositeRule{}
		}
		writeJSON2(w, http.StatusOK, map[string]any{"rules": result, "total": len(result)})
	case http.MethodPost:
		var rule CompositeRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSON2(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		if rule.Name == "" || len(rule.Signals) == 0 {
			writeJSON2(w, http.StatusBadRequest, map[string]string{"error": "name and signals required"})
			return
		}
		rule.ID = uuid.New().String()
		if rule.MinSignals == 0 {
			rule.MinSignals = len(rule.Signals)
		}
		if rule.WindowMin == 0 {
			rule.WindowMin = 30
		}
		if rule.Severity == "" {
			rule.Severity = "critical"
		}
		rule.Enabled = true
		rule.CreatedAt = time.Now()
		if s.compositeRepo != nil {
			if err := s.compositeRepo.Create(r.Context(), &rule); err != nil {
				writeJSON2(w, http.StatusInternalServerError, map[string]string{"error": "failed"})
				return
			}
		} else {
			compositeRulesMu.Lock()
			compositeRules[rule.ID] = &rule
			compositeRulesMu.Unlock()
		}
		writeJSON2(w, http.StatusCreated, rule)
	case http.MethodPut:
		parts := strings.Split(r.URL.Path, "/")
		id := parts[len(parts)-1]
		var update CompositeRule
		json.NewDecoder(r.Body).Decode(&update)
		update.ID = id
		if s.compositeRepo != nil {
			s.compositeRepo.Update(r.Context(), &update)
		} else {
			compositeRulesMu.Lock()
			if rule, ok := compositeRules[id]; ok {
				if update.Name != "" {
					rule.Name = update.Name
				}
				rule.Signals = update.Signals
				rule.MinSignals = update.MinSignals
				rule.Enabled = update.Enabled
			}
			compositeRulesMu.Unlock()
		}
		writeJSON2(w, http.StatusOK, update)
	case http.MethodDelete:
		parts := strings.Split(r.URL.Path, "/")
		id := parts[len(parts)-1]
		if s.compositeRepo != nil {
			s.compositeRepo.Delete(r.Context(), id)
		} else {
			compositeRulesMu.Lock()
			delete(compositeRules, id)
			compositeRulesMu.Unlock()
		}
		writeJSON2(w, http.StatusOK, map[string]bool{"deleted": true})
	}
}

// handleITDRIncidents returns active security itdrIncidents.
func (s *HTTPServer) handleITDRIncidents(w http.ResponseWriter, r *http.Request) {
	itdrIncidentsMu.RLock()
	result := make([]*IncidentListEntry, 0, len(itdrIncidents))
	status := r.URL.Query().Get("status")
	for _, inc := range itdrIncidents {
		if status != "" && inc.Status != status {
			continue
		}
		result = append(result, inc)
	}
	itdrIncidentsMu.RUnlock()
	writeJSON2(w, http.StatusOK, map[string]any{"itdrIncidents": result, "total": len(result)})
}

// handleITDRKillChain returns the kill chain timeline for an incident.
func (s *HTTPServer) handleITDRKillChain(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]
	itdrIncidentsMu.RLock()
	inc, ok := itdrIncidents[id]
	itdrIncidentsMu.RUnlock()
	if !ok {
		writeJSON2(w, http.StatusNotFound, map[string]string{"error": "incident not found"})
		return
	}
	writeJSON2(w, http.StatusOK, map[string]any{
		"incident_id": id,
		"timeline":    inc.Timeline,
		"stages":      []string{"reconnaissance", "credential_access", "lateral_movement", "exfiltration", "impact"},
	})
}

// handleITDRPlaybooks handles playbook CRUD.
func (s *HTTPServer) handleITDRPlaybooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		playbooksMu.RLock()
		result := make([]*Playbook, 0, len(playbooks))
		for _, pb := range playbooks {
			result = append(result, pb)
		}
		playbooksMu.RUnlock()
		writeJSON2(w, http.StatusOK, map[string]any{"playbooks": result, "total": len(result)})
	case http.MethodPost:
		var pb Playbook
		if err := json.NewDecoder(r.Body).Decode(&pb); err != nil {
			writeJSON2(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
			return
		}
		if pb.Name == "" || pb.Trigger == "" {
			writeJSON2(w, http.StatusBadRequest, map[string]string{"error": "name and trigger required"})
			return
		}
		pb.ID = uuid.New().String()
		pb.Enabled = true
		pb.CreatedAt = time.Now()
		playbooksMu.Lock()
		playbooks[pb.ID] = &pb
		playbooksMu.Unlock()
		writeJSON2(w, http.StatusCreated, pb)
	}
}

// handleITDRThreatHeatmap returns user×resource detection heatmap data.
func (s *HTTPServer) handleITDRThreatHeatmap(w http.ResponseWriter, r *http.Request) {
	// In production: aggregate from detections table GROUP BY user_id, resource_type.
	// For now returns structured response ready for DB wiring.
	writeJSON2(w, http.StatusOK, map[string]any{
		"entries":  []ThreatHeatmapEntry{},
		"total":    0,
		"generated_at": time.Now().UTC(),
	})
}

// writeJSON2 is a local JSON writer to avoid collision with existing writeJSON.
func writeJSON2(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
