package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type VelocityRule struct {
	RuleID       string  `json:"rule_id"`
	Metric       string  `json:"metric"`
	Window       string  `json:"window"`
	Threshold    int     `json:"threshold"`
	Action       string  `json:"action"`
	Enabled      bool    `json:"enabled"`
	Triggered24h int     `json:"triggered_24h"`
	PerScope     string  `json:"per_scope"`
}

type VelocityRulesResult struct {
	Rules    []VelocityRule `json:"rules"`
	TotalRules int          `json:"total_rules"`
	EnabledCount int        `json:"enabled_count"`
}

type VelocityRuleRequest struct {
	Metric    string `json:"metric"`
	Window    string `json:"window"`
	Threshold int    `json:"threshold"`
	Action    string `json:"action"`
	PerScope  string `json:"per_scope"`
}

var velocityRulesStore sync.Map

func init() {
	for _, r := range []VelocityRule{
		{RuleID: "vr-001", Metric: "login_attempts", Window: "60s", Threshold: 5, Action: "block", Enabled: true, Triggered24h: 142, PerScope: "per_ip"},
		{RuleID: "vr-002", Metric: "login_attempts", Window: "1h", Threshold: 20, Action: "captcha", Enabled: true, Triggered24h: 89, PerScope: "per_account"},
		{RuleID: "vr-003", Metric: "password_reset", Window: "1h", Threshold: 3, Action: "block", Enabled: true, Triggered24h: 12, PerScope: "per_account"},
		{RuleID: "vr-004", Metric: "mfa_attempts", Window: "5m", Threshold: 5, Action: "lockout", Enabled: true, Triggered24h: 34, PerScope: "per_account"},
		{RuleID: "vr-005", Metric: "token_refresh", Window: "1m", Threshold: 10, Action: "rate_limit", Enabled: false, Triggered24h: 0, PerScope: "per_client"},
	} {
		velocityRulesStore.Store(r.RuleID, r)
	}
}

func (h *Handler) handleVelocityRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var rules []VelocityRule
		velocityRulesStore.Range(func(_, v any) bool {
			rules = append(rules, v.(VelocityRule))
			return true
		})
		enabledCount := 0
		for _, r := range rules {
			if r.Enabled {
				enabledCount++
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(VelocityRulesResult{Rules: rules, TotalRules: len(rules), EnabledCount: enabledCount})
	case http.MethodPost:
		var req VelocityRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		if req.Metric == "" {
			req.Metric = "login_attempts"
		}
		if req.Window == "" {
			req.Window = "60s"
		}
		if req.Threshold == 0 {
			req.Threshold = 5
		}
		if req.Action == "" {
			req.Action = "block"
		}
		if req.PerScope == "" {
			req.PerScope = "per_ip"
		}
		rule := VelocityRule{
			RuleID: fmt.Sprintf("vr-%d", time.Now().UnixNano()%100000),
			Metric: req.Metric, Window: req.Window, Threshold: req.Threshold,
			Action: req.Action, Enabled: true, Triggered24h: 0, PerScope: req.PerScope,
		}
		velocityRulesStore.Store(rule.RuleID, rule)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rule)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
