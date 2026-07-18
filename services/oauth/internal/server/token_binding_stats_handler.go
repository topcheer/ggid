package server

import (
	"encoding/json"
	"net/http"
	"sync"
)

type BindingMethodStat struct {
	Method   string  `json:"method"`
	Count    int     `json:"count"`
	Pct      float64 `json:"pct"`
}

type ClientBindingStat struct {
	ClientID       string  `json:"client_id"`
	ClientName     string  `json:"client_name"`
	Bound          int     `json:"bound_tokens"`
	Unbound        int     `json:"unbound_tokens"`
	BindingMethods []string `json:"binding_methods"`
	Compliant      bool    `json:"compliant"`
}

type TokenBindingStats struct {
	TotalTokens     int                  `json:"total_tokens"`
	BoundTokens     int                  `json:"bound_tokens"`
	UnboundTokens   int                  `json:"unbound_tokens"`
	CompliancePct   float64              `json:"compliance_pct"`
	BindingMethods  []BindingMethodStat  `json:"binding_methods"`
	ByClient        []ClientBindingStat   `json:"by_client"`
	GeneratedAt     string               `json:"generated_at"`
}

var tokenBindingOnce sync.Once
var tokenBindingData TokenBindingStats

func initTokenBindingData() {
	tokenBindingOnce.Do(func() {
		tokenBindingData = TokenBindingStats{
			TotalTokens:   4280,
			BoundTokens:   3520,
			UnboundTokens: 760,
			CompliancePct: 82.2,
			BindingMethods: []BindingMethodStat{
				{Method: "DPoP", Count: 1850, Pct: 52.6},
				{Method: "mTLS", Count: 1120, Pct: 31.8},
				{Method: "PKI", Count: 550, Pct: 15.6},
			},
			ByClient: []ClientBindingStat{
				{ClientID: "c-001", ClientName: "web-app", Bound: 980, Unbound: 120, BindingMethods: []string{"DPoP"}, Compliant: true},
				{ClientID: "c-002", ClientName: "mobile-app", Bound: 720, Unbound: 280, BindingMethods: []string{"DPoP"}, Compliant: false},
				{ClientID: "c-003", ClientName: "api-gateway", Bound: 650, Unbound: 50, BindingMethods: []string{"mTLS"}, Compliant: true},
				{ClientID: "c-004", ClientName: "service-account", Bound: 520, Unbound: 0, BindingMethods: []string{"mTLS", "PKI"}, Compliant: true},
				{ClientID: "c-005", ClientName: "legacy-app", Bound: 0, Unbound: 310, BindingMethods: []string{}, Compliant: false},
			},
			GeneratedAt: "2025-01-15T10:00:00Z",
		}
	})
}

func handleTokenBindingStats(w http.ResponseWriter, r *http.Request) {
	initTokenBindingData()
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenBindingData)
}
