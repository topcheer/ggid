package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type SimulateBatchRequest struct {
	Subjects []string `json:"subjects"`
	Resources []string `json:"resources"`
	Actions   []string `json:"actions"`
}

type SubjectResult struct {
	Subject  string                  `json:"subject"`
	Results  []ResourceActionResult   `json:"results"`
}

type ResourceActionResult struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

type AggregateStats struct {
	TotalChecks   int     `json:"total_checks"`
	AllowedCount  int     `json:"allowed_count"`
	DeniedCount   int     `json:"denied_count"`
	AllowRate     float64 `json:"allow_rate"`
}

type SimulateBatchResult struct {
	PerSubject     []SubjectResult `json:"per_subject"`
	AggregateStats AggregateStats  `json:"aggregate_stats"`
	MismatchCount  int             `json:"mismatch_count"`
	SimulatedAt    string          `json:"simulated_at"`
}

var simulateBatchStore sync.Map

func (s *HTTPServer) handlePolicySimulateBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req SimulateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.Subjects) == 0 {
		req.Subjects = []string{"user:alice", "user:bob", "role:admin"}
	}
	if len(req.Resources) == 0 {
		req.Resources = []string{"doc:1", "folder:2", "system:settings"}
	}
	if len(req.Actions) == 0 {
		req.Actions = []string{"read", "write", "delete"}
	}

	var perSubject []SubjectResult
	totalChecks := 0
	allowed := 0
	denied := 0
	mismatches := 0

	for _, subj := range req.Subjects {
		sr := SubjectResult{Subject: subj}
		isAdmin := subj == "role:admin" || subj == "user:bob"
		for _, res := range req.Resources {
			for _, act := range req.Actions {
				decision := "deny"
				if isAdmin {
					decision = "allow"
				} else if act == "read" && res != "system:settings" {
					decision = "allow"
				}
				if decision == "allow" {
					allowed++
				} else {
					denied++
				}
				totalChecks++
				sr.Results = append(sr.Results, ResourceActionResult{
					Resource: res, Action: act, Decision: decision,
				})
			}
		}
		perSubject = append(perSubject, sr)
	}

	allowRate := 0.0
	if totalChecks > 0 {
		allowRate = float64(allowed) / float64(totalChecks)
	}

	result := SimulateBatchResult{
		PerSubject: perSubject,
		AggregateStats: AggregateStats{
			TotalChecks: totalChecks, AllowedCount: allowed,
			DeniedCount: denied, AllowRate: allowRate,
		},
		MismatchCount: mismatches,
		SimulatedAt:   "2025-01-15T10:00:00Z",
	}

	simulateBatchStore.Store("latest", result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
