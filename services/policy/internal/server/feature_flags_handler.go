package httpserver

import (
	"encoding/json"
	"net/http"
	"time"
)

type FeatureFlag struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Enabled        bool    `json:"enabled"`
	RolloutPct     float64 `json:"rollout_pct"`
	TargetAudience string  `json:"target_audience"`
	Environment    string  `json:"environment"`
}

type FeatureFlagsResult struct {
	Flags    []FeatureFlag `json:"flags"`
	AuditLog []struct {
		Timestamp string `json:"timestamp"`
		Flag      string `json:"flag"`
		Action    string `json:"action"`
		By        string `json:"by"`
	} `json:"audit_log"`
}

func (s *HTTPServer) handleFeatureFlags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		result := FeatureFlagsResult{
			Flags: []FeatureFlag{
				{Name: "webauthn_login", Description: "WebAuthn passwordless login", Enabled: true, RolloutPct: 100, TargetAudience: "all", Environment: "prod"},
				{Name: "scim_2_0", Description: "SCIM 2.0 provisioning", Enabled: true, RolloutPct: 100, TargetAudience: "all", Environment: "prod"},
				{Name: "adaptive_auth", Description: "Risk-based adaptive authentication", Enabled: true, RolloutPct: 50, TargetAudience: "beta", Environment: "prod"},
				{Name: "dpop_tokens", Description: "DPoP proof-of-possession tokens", Enabled: true, RolloutPct: 25, TargetAudience: "beta", Environment: "prod"},
				{Name: "pipl_compliance", Description: "PIPL data governance", Enabled: false, RolloutPct: 0, TargetAudience: "specific", Environment: "staging"},
				{Name: "agent_identity", Description: "AI Agent identity & MCP auth", Enabled: true, RolloutPct: 100, TargetAudience: "all", Environment: "prod"},
			},
			AuditLog: []struct {
				Timestamp string `json:"timestamp"`
				Flag      string `json:"flag"`
				Action    string `json:"action"`
				By        string `json:"by"`
			}{
				{Timestamp: time.Now().Format(time.RFC3339), Flag: "dpop_tokens", Action: "rollout_pct_changed:10→25", By: "admin@GGID"},
				{Timestamp: "2025-01-14T10:00:00Z", Flag: "adaptive_auth", Action: "rollout_pct_changed:0→50", By: "admin@GGID"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case http.MethodPut:
		var req FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "updated", "flag": req.Name, "enabled": req.Enabled})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
