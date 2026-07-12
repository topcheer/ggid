package server

import (
	"encoding/json"
	"net/http"
)

type SignalBreakdown struct {
	Signal   string  `json:"signal"`
	Score    float64 `json:"score"`
	Weight   float64 `json:"weight"`
	Detail   string  `json:"detail"`
}

type FlaggedAccount struct {
	AccountID string  `json:"account_id"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason"`
}

type BlockedEntity struct {
	EntityType string `json:"entity_type"`
	Value      string `json:"value"`
	Reason     string `json:"reason"`
	BlockedAt  string `json:"blocked_at"`
}

type FraudScoreResult struct {
	CompositeScore   float64          `json:"composite_score"`
	RiskLevel        string           `json:"risk_level"`
	SignalBreakdown  []SignalBreakdown `json:"signal_breakdown"`
	FlaggedAccounts  []FlaggedAccount `json:"flagged_accounts"`
	BlockedEntities  []BlockedEntity  `json:"blocked_entities"`
	GeneratedAt      string           `json:"generated_at"`
}

func (h *Handler) handleFraudScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := FraudScoreResult{
		CompositeScore: 0.72,
		RiskLevel:      "high",
		SignalBreakdown: []SignalBreakdown{
			{Signal: "device_anomaly", Score: 0.65, Weight: 0.25, Detail: "New device fingerprint from unknown location"},
			{Signal: "velocity_anomaly", Score: 0.85, Weight: 0.30, Detail: "12 login attempts in 60s from different IPs"},
			{Signal: "ip_reputation", Score: 0.78, Weight: 0.25, Detail: "Source IP on 3 threat lists"},
			{Signal: "behavioral_anomaly", Score: 0.55, Weight: 0.20, Detail: "Unusual access pattern: off-hours admin endpoints"},
		},
		FlaggedAccounts: []FlaggedAccount{
			{AccountID: "u-0342", Score: 0.91, Reason: "impossible_travel + credential_stuffing"},
			{AccountID: "u-0517", Score: 0.83, Reason: "device_mismatch + off_hours_access"},
			{AccountID: "u-0891", Score: 0.76, Reason: "ip_reputation + velocity_spike"},
		},
		BlockedEntities: []BlockedEntity{
			{EntityType: "ip", Value: "203.0.113.50", Reason: "credential_stuffing_source", BlockedAt: "2025-01-15T03:30:00Z"},
			{EntityType: "device", Value: "fp-unknown-chrome-win", Reason: "fraud_device", BlockedAt: "2025-01-15T03:25:00Z"},
		},
		GeneratedAt: "2025-01-15T10:00:00Z",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
