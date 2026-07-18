package server

import (
	"encoding/json"
	"net/http"
)

type LateralPattern struct {
	UserID         string   `json:"user_id"`
	ResourceChain  []string `json:"resource_chain"`
	Timeline       string   `json:"timeline"`
	AccessVelocity float64  `json:"access_velocity"`
	Severity       string   `json:"severity"`
}

type LateralMovementResult struct {
	DetectedPatterns []LateralPattern `json:"detected_patterns"`
	MITREMapping     []string         `json:"mitre_mapping"`
	ConfidenceScore  float64          `json:"confidence_score"`
	TotalDetected    int              `json:"total_detected"`
}

func (h *Handler) handleLateralMovementDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := LateralMovementResult{
		DetectedPatterns: []LateralPattern{
			{UserID: "u-0342", ResourceChain: []string{"web-app", "db-proxy", "file-share", "dc"}, Timeline: "2025-01-15T03:00-03:30Z", AccessVelocity: 8.5, Severity: "critical"},
			{UserID: "u-0517", ResourceChain: []string{"api-gateway", "internal-api", "vault"}, Timeline: "2025-01-15T03:10-03:20Z", AccessVelocity: 6.2, Severity: "high"},
			{UserID: "u-0891", ResourceChain: []string{"ssh-bastion", "k8s-node", "etcd"}, Timeline: "2025-01-14T22:00-22:15Z", AccessVelocity: 4.8, Severity: "high"},
		},
		MITREMapping: []string{"T1021.004 - Remote Services: SSH", "T1078 - Valid Accounts", "T1550.002 - Pass the Hash"},
		ConfidenceScore: 0.84,
		TotalDetected:   3,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
