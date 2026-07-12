package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/services/oauth/internal/service"
)

var driftDetector = service.NewDriftDetector()
var shadowScanner = service.NewShadowScanner(nil)

func handleAgentDriftDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, `{"error":"agent id required"}`, http.StatusBadRequest)
		return
	}
	agentID := parts[len(parts)-1]
	reports := driftDetector.GetReports(agentID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}

func handleAgentShadows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	report := shadowScanner.ScanShadows(nil)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func handleAgentDriftReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"message":   "full drift report endpoint",
		"reports":   []any{},
	})
}