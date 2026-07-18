package server

import (
	"encoding/json"
	"net/http"
)

type IntelSource struct {
	SourceName string `json:"source_name"`
	SourceType string `json:"source_type"`
	Status     string `json:"status"`
	LastSync   string `json:"last_sync"`
	Indicators int    `json:"indicators_imported"`
}

type ThreatIndicator struct {
	IndicatorType string   `json:"type"`
	Value         string   `json:"value"`
	Confidence    float64  `json:"confidence"`
	Tags          []string `json:"tags"`
	FirstSeen     string   `json:"first_seen"`
}

type AutoBlockRule struct {
	RuleID    string `json:"rule_id"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Enabled   bool   `json:"enabled"`
}

type ThreatIntelFeedResult struct {
	IntelSources    []IntelSource    `json:"intel_sources"`
	Indicators      []ThreatIndicator `json:"indicators"`
	AutoBlockRules  []AutoBlockRule  `json:"auto_block_rules"`
	TotalIndicators int              `json:"total_indicators"`
	GeneratedAt     string           `json:"generated_at"`
}

func (h *Handler) handleThreatIntelFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := ThreatIntelFeedResult{
		IntelSources: []IntelSource{
			{SourceName: "AlienVault OTX", SourceType: "threat_feed", Status: "active", LastSync: "2025-01-15T09:30:00Z", Indicators: 45200},
			{SourceName: "AbuseIPDB", SourceType: "ip_reputation", Status: "active", LastSync: "2025-01-15T09:45:00Z", Indicators: 12800},
			{SourceName: "HaveIBeenPwned", SourceType: "breach_data", Status: "active", LastSync: "2025-01-15T08:00:00Z", Indicators: 8900},
			{SourceName: "MISP Feed", SourceType: "stix/taxii", Status: "syncing", LastSync: "2025-01-15T07:00:00Z", Indicators: 23000},
		},
		Indicators: []ThreatIndicator{
			{IndicatorType: "ip", Value: "203.0.113.50", Confidence: 0.95, Tags: []string{"botnet", "credential_stuffing"}, FirstSeen: "2025-01-10T00:00:00Z"},
			{IndicatorType: "ip", Value: "198.51.100.12", Confidence: 0.88, Tags: []string{"brute_force", "proxy"}, FirstSeen: "2025-01-08T00:00:00Z"},
			{IndicatorType: "domain", Value: "evil-phish.example", Confidence: 0.92, Tags: []string{"phishing", "credential_harvest"}, FirstSeen: "2025-01-12T00:00:00Z"},
			{IndicatorType: "user_agent", Value: "python-requests/2.28", Confidence: 0.75, Tags: []string{"automated", "scraper"}, FirstSeen: "2025-01-05T00:00:00Z"},
			{IndicatorType: "email", Value: "test@temp-mail.dev", Confidence: 0.81, Tags: []string{"disposable", "fraud"}, FirstSeen: "2025-01-11T00:00:00Z"},
		},
		AutoBlockRules: []AutoBlockRule{
			{RuleID: "abr-001", Condition: "confidence >= 0.9 AND type=ip", Action: "block_ip", Enabled: true},
			{RuleID: "abr-002", Condition: "confidence >= 0.8 AND type=domain", Action: "block_redirect", Enabled: true},
			{RuleID: "abr-003", Condition: "tag=disposable AND type=email", Action: "flag_for_review", Enabled: true},
			{RuleID: "abr-004", Condition: "confidence >= 0.7 AND type=user_agent", Action: "challenge_captcha", Enabled: false},
		},
		TotalIndicators: 89900,
		GeneratedAt:     "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
