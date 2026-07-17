package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/services/audit/internal/repository"
	"github.com/google/uuid"
)

func (s *HTTPServer) handleThreatIntel(w http.ResponseWriter, r *http.Request) {
	if s.threatIntelRepo == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "threat intel not configured")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/audit/threat-intel")
	tenantID := tenantIDFromRequest(r)

	switch {
	case path == "/sources" || path == "/sources/":
		s.handleThreatIntelSources(w, r, tenantID)
	case strings.HasPrefix(path, "/sources/"):
		s.handleThreatIntelSources(w, r, tenantID)
	case path == "/indicators" || path == "/indicators/":
		s.handleThreatIntelIndicators(w, r, tenantID)
	case path == "/check":
		s.handleThreatIntelCheck(w, r, tenantID)
	case path == "/stats":
		s.handleThreatIntelStats(w, r, tenantID)
	default:
		writeJSONError(w, http.StatusNotFound, "unknown threat-intel endpoint")
	}
}

func (s *HTTPServer) handleThreatIntelSources(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	switch r.Method {
	case http.MethodGet:
		sources, err := s.threatIntelRepo.ListSources(r.Context(), tenantID, false)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list sources")
			return
		}
		if sources == nil {
			sources = []repository.ThreatIntelSource{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"sources": sources, "total": len(sources)})

	case http.MethodPost:
		var req struct {
			Name         string `json:"name"`
			SourceType   string `json:"source_type"`
			APIEndpoint  string `json:"api_endpoint"`
			APIKeyRef    string `json:"api_key_ref"`
			PollInterval string `json:"poll_interval"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" || req.SourceType == "" || req.APIEndpoint == "" {
			writeJSONError(w, http.StatusBadRequest, "name, source_type, and api_endpoint are required")
			return
		}
		if req.PollInterval == "" {
			req.PollInterval = "1 hour"
		}

		src := repository.ThreatIntelSource{
			ID:           uuid.New(),
			TenantID:     tenantID,
			Name:         req.Name,
			SourceType:   req.SourceType,
			APIEndpoint:  req.APIEndpoint,
			APIKeyRef:     req.APIKeyRef,
			PollInterval: req.PollInterval,
			Enabled:      true,
			CreatedAt:    time.Now(),
		}
		if err := s.threatIntelRepo.CreateSource(r.Context(), &src); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create source")
			return
		}
		writeJSON(w, http.StatusCreated, src)

	case http.MethodDelete:
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/audit/threat-intel/sources/"), "/")
		if len(parts) < 1 || parts[0] == "" {
			writeJSONError(w, http.StatusBadRequest, "source id required")
			return
		}
		srcID, err := uuid.Parse(parts[0])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid source id")
			return
		}
		if err := s.threatIntelRepo.DeleteSource(r.Context(), tenantID, srcID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to delete source")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleThreatIntelIndicators(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	indType := r.URL.Query().Get("type")
	indicators, total, err := s.threatIntelRepo.ListIndicators(r.Context(), tenantID, indType, 100)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list indicators")
		return
	}
	if indicators == nil {
		indicators = []repository.ThreatIndicator{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"indicators": indicators, "total": total})
}

func (s *HTTPServer) handleThreatIntelCheck(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Indicator     string `json:"indicator"`
		IndicatorType string `json:"indicator_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Indicator == "" || req.IndicatorType == "" {
		writeJSONError(w, http.StatusBadRequest, "indicator and indicator_type are required")
		return
	}

	match, err := s.threatIntelRepo.CheckIndicator(r.Context(), tenantID, req.IndicatorType, req.Indicator)
	if err != nil || match == nil {
		writeJSON(w, http.StatusOK, map[string]any{"matched": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"matched":    true,
		"severity":   match.Severity,
		"confidence": match.Confidence,
		"source_id":  match.SourceID,
		"last_seen":  match.LastSeen,
	})
}

func (s *HTTPServer) handleThreatIntelStats(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	stats, err := s.threatIntelRepo.Stats(r.Context(), tenantID)
	if err != nil || stats == nil {
		writeJSON(w, http.StatusOK, map[string]any{"sources_enabled": 0, "indicators_total": 0, "hits_24h": 0, "by_type": map[string]int{}})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func tenantIDFromRequest(r *http.Request) uuid.UUID {
	if tid := r.Header.Get("X-Tenant-ID"); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			return id
		}
	}
	return uuid.Nil
}
