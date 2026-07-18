package httpserver

import (
	"encoding/json"
	"net/http"
)

type CollaborationPattern struct {
	Pattern string  `json:"pattern"`
	Score   float64 `json:"score"`
}

type ExpertiseDist struct {
	Area     string `json:"area"`
	Members  int    `json:"members"`
	Coverage string `json:"coverage"`
}

type TeamInsightsResult struct {
	TeamID                string                `json:"team_id"`
	TeamName              string                `json:"team_name"`
	CohesionScore         float64               `json:"cohesion_score"`
	CollaborationPatterns []CollaborationPattern `json:"collaboration_patterns"`
	SiloDetection         []string              `json:"silo_detection"`
	CrossTeamDeps         []string              `json:"cross_team_deps"`
	ExpertiseDistribution []ExpertiseDist       `json:"expertise_distribution"`
	RiskOfAttrition       float64               `json:"risk_of_attrition"`
	MemberCount           int                   `json:"member_count"`
}

func (s *HTTPServer) handleTeamInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	result := TeamInsightsResult{
		TeamID:        "team-001",
		TeamName:      "Platform Engineering",
		CohesionScore: 0.78,
		CollaborationPatterns: []CollaborationPattern{
			{Pattern: "pair_programming", Score: 0.65},
			{Pattern: "code_review_coverage", Score: 0.89},
			{Pattern: "cross_functional_tickets", Score: 0.42},
			{Pattern: "knowledge_sharing", Score: 0.71},
		},
		SiloDetection: []string{"backend-vs-frontend: low interaction (0.3)", "devops-vs-sre: moderate silo (0.45)"},
		CrossTeamDeps: []string{"depends on team-security for auth", "blocked by team-infra for k8s upgrades"},
		ExpertiseDistribution: []ExpertiseDist{
			{Area: "Go", Members: 8, Coverage: "strong"},
			{Area: "Kubernetes", Members: 5, Coverage: "adequate"},
			{Area: "PostgreSQL", Members: 3, Coverage: "thin"},
			{Area: "Frontend", Members: 1, Coverage: "risk"},
		},
		RiskOfAttrition: 0.22,
		MemberCount:     12,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
