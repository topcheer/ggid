package server

import (
	"encoding/json"
	"net/http"
)

type ThreatDetection struct {
	Type            string `json:"type"`
	Severity        string `json:"severity"`
	MITRETechnique  string `json:"mitre_technique"`
	Description     string `json:"description"`
	DetectedCount   int    `json:"detected_count_24h"`
}

type DetectionRule struct {
	RuleID   string `json:"rule_id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Severity string `json:"severity"`
}

type ResponsePlaybook struct {
	PlaybookID string   `json:"playbook_id"`
	Name       string   `json:"name"`
	Trigger    string   `json:"trigger"`
	Steps      []string `json:"steps"`
}

type ITDRDetectionsResult struct {
	ThreatDetections   []ThreatDetection  `json:"threat_detections"`
	DetectionRules     []DetectionRule    `json:"detection_rules"`
	ResponsePlaybooks  []ResponsePlaybook `json:"response_playbooks"`
	TotalDetected24h   int                `json:"total_detected_24h"`
	CriticalCount      int                `json:"critical_count"`
}

func (h *Handler) handleITDRDetections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := ITDRDetectionsResult{
		ThreatDetections: []ThreatDetection{
			{Type: "credential_stuffing", Severity: "critical", MITRETechnique: "T1110.004", Description: "Credential Stuffing", DetectedCount: 42},
			{Type: "pass_the_hash", Severity: "high", MITRETechnique: "T1550.002", Description: "Pass the Hash detected", DetectedCount: 3},
			{Type: "golden_ticket", Severity: "critical", MITRETechnique: "T1558.001", Description: "Kerberos Golden Ticket", DetectedCount: 1},
			{Type: "anomalous_ldap", Severity: "medium", MITRETechnique: "T1018", Description: "Anomalous LDAP queries", DetectedCount: 18},
			{Type: "lateral_movement", Severity: "high", MITRETechnique: "T1021", Description: "Lateral movement via SSH", DetectedCount: 7},
		},
		DetectionRules: []DetectionRule{
			{RuleID: "dr-001", Name: "Impossible Travel", Enabled: true, Severity: "high"},
			{RuleID: "dr-002", Name: "Brute Force Pattern", Enabled: true, Severity: "critical"},
			{RuleID: "dr-003", Name: "Off-hours Admin Access", Enabled: true, Severity: "medium"},
			{RuleID: "dr-004", Name: "New Device + Privileged Action", Enabled: true, Severity: "high"},
			{RuleID: "dr-005", Name: "Token Replay", Enabled: false, Severity: "critical"},
		},
		ResponsePlaybooks: []ResponsePlaybook{
			{PlaybookID: "pb-001", Name: "Lock and Investigate", Trigger: "critical", Steps: []string{"lock_account", "revoke_sessions", "notify_soc", "create_incident"}},
			{PlaybookID: "pb-002", Name: "Step-up Auth", Trigger: "high", Steps: []string{"require_mfa", "alert_admin", "log_event"}},
			{PlaybookID: "pb-003", Name: "Monitor and Log", Trigger: "medium", Steps: []string{"increase_monitoring", "log_event"}},
		},
		TotalDetected24h: 71,
		CriticalCount:    43,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
