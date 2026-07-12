package server

import (
	"encoding/json"
	"net/http"
)

type DLPPolicy struct {
	PolicyID      string   `json:"policy_id"`
	Name          string   `json:"name"`
	Enabled       bool     `json:"enabled"`
	DetectionRules []string `json:"detection_rules"`
	Actions       []string `json:"actions"`
	Channels      []string `json:"channels"`
	Violations24h int      `json:"violations_24h"`
}

type DLPResult struct {
	Policies       []DLPPolicy `json:"policies"`
	TotalViolations int        `json:"total_violations"`
	BlockedCount    int        `json:"blocked_count"`
	QuarantinedCount int       `json:"quarantined_count"`
	GeneratedAt    string      `json:"generated_at"`
}

func (h *Handler) handleDLPPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := DLPResult{
		Policies: []DLPPolicy{
			{PolicyID: "dlp-001", Name: "SSN Detection", Enabled: true, DetectionRules: []string{"regex: \\d{3}-\\d{2}-\\d{4}"}, Actions: []string{"block", "alert"}, Channels: []string{"email", "api"}, Violations24h: 12},
			{PolicyID: "dlp-002", Name: "Credit Card Numbers", Enabled: true, DetectionRules: []string{"regex: \\d{13,19}", "luhn_check"}, Actions: []string{"block", "quarantine"}, Channels: []string{"email", "api", "webhook"}, Violations24h: 8},
			{PolicyID: "dlp-003", Name: "API Keys", Enabled: true, DetectionRules: []string{"pattern: sk_live_", "pattern: AKIA"}, Actions: []string{"alert", "auto_rotate"}, Channels: []string{"api", "logs"}, Violations24h: 5},
			{PolicyID: "dlp-004", Name: "Health Records (PHI)", Enabled: true, DetectionRules: []string{"nlp_model: phi_detector"}, Actions: []string{"block", "quarantine", "alert"}, Channels: []string{"email", "api", "export"}, Violations24h: 3},
		},
		TotalViolations:  28,
		BlockedCount:     15,
		QuarantinedCount: 8,
		GeneratedAt:      "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
