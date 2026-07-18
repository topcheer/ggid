package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/ggid/ggid/pkg/errors"
)

func (s *HTTPServer) handleCCM(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/api/v1/audit/ccm/results" && r.Method == http.MethodGet:
		s.ccmResults(w, r)
	case path == "/api/v1/audit/ccm/history" && r.Method == http.MethodGet:
		s.ccmHistory(w, r)
	case path == "/api/v1/audit/ccm/run" && r.Method == http.MethodPost:
		s.ccmRun(w, r)
	case path == "/api/v1/audit/ccm/summary" && r.Method == http.MethodGet:
		s.ccmSummary(w, r)
	default:
		errors.WriteSimpleAPIError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

// ccmResults returns the latest result for each compliance control.
func (s *HTTPServer) ccmResults(w http.ResponseWriter, r *http.Request) {
	if s.ccmEngine == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	results := s.ccmEngine.GetResults()
	writeJSON(w, http.StatusOK, results)
}

// ccmHistory returns historical compliance results with optional filtering.
func (s *HTTPServer) ccmHistory(w http.ResponseWriter, r *http.Request) {
	if s.ccmEngine == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	controlID := r.URL.Query().Get("control_id")
	limit := 100

	results := s.ccmEngine.GetHistory(controlID, limit)
	if results == nil {
		results = []CCMResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

// ccmRun triggers a manual compliance scan of all controls.
func (s *HTTPServer) ccmRun(w http.ResponseWriter, r *http.Request) {
	if s.ccmEngine == nil {
		s.ccmEngine = NewCCMEngine()
	}

	var req struct {
		Controls []string `json:"controls"`
	}
	// Body is optional — empty body runs all controls.
	_ = json.NewDecoder(r.Body).Decode(&req)

	results := s.ccmEngine.RunAll()

	writeJSON(w, http.StatusOK, map[string]any{
		"results":        results,
		"controls_run":   len(results),
		"summary":        s.ccmEngine.GetSummary(),
	})
}

// ccmSummary returns a high-level compliance dashboard summary.
func (s *HTTPServer) ccmSummary(w http.ResponseWriter, r *http.Request) {
	if s.ccmEngine == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"total_controls":   0,
			"pass":             0,
			"warn":             0,
			"fail":             0,
			"compliance_score": 0,
			"message":          "CCM engine not configured — run POST /ccm/run to initialize",
		})
		return
	}

	writeJSON(w, http.StatusOK, s.ccmEngine.GetSummary())
}
