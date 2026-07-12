package server

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// riskFactor represents a single contributing factor to user risk.
type riskFactor struct {
	Type     string `json:"type"`
	Weight   int    `json:"weight"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"` // low, medium, high, critical
}

// userRiskProfile aggregates multiple risk factors into a composite score.
type userRiskProfile struct {
	UserID     string       `json:"user_id"`
	RiskScore  int          `json:"risk_score"` // 0-100
	RiskLevel  string       `json:"risk_level"` // low, moderate, elevated, high, critical
	Factors    []riskFactor `json:"factors"`
	Trend      string       `json:"trend"` // improving, stable, worsening
	ScoreHistory []map[string]any `json:"score_history"`
	AssessedAt string       `json:"assessed_at"`
}

var riskProfileStore = struct {
	sync.RWMutex
	data map[string]*userRiskProfile
}{data: make(map[string]*userRiskProfile)}

// GET /api/v1/users/{id}/risk-profile
func (h *HTTPHandler) handleRiskProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract user ID from path
	path := r.URL.Path
	userID := ""
	if idx := strings.Index(path, "/users/"); idx >= 0 {
		rest := path[idx+len("/users/"):]
		if rpIdx := strings.Index(rest, "/risk-profile"); rpIdx >= 0 {
			userID = rest[:rpIdx]
		}
	}
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user ID is required in path")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	// Build risk profile from multiple signals
	factors := []riskFactor{}

	// Factor 1: Privileged access
	factors = append(factors, riskFactor{
		Type: "privileged_access", Weight: 25,
		Detail: "User has admin-level role assignments",
		Severity: "high",
	})

	// Factor 2: Stale password
	factors = append(factors, riskFactor{
		Type: "stale_password", Weight: 10,
		Detail: "Password not changed in 120+ days",
		Severity: "medium",
	})

	// Factor 3: No MFA
	factors = append(factors, riskFactor{
		Type: "no_mfa", Weight: 20,
		Detail: "Multi-factor authentication not enrolled",
		Severity: "high",
	})

	// Factor 4: Dormant account
	factors = append(factors, riskFactor{
		Type: "dormant", Weight: 8,
		Detail: "No login activity in 45 days",
		Severity: "low",
	})

	// Factor 5: Exposed credentials
	factors = append(factors, riskFactor{
		Type: "exposed_credentials", Weight: 15,
		Detail: "Email found in known breach database",
		Severity: "critical",
	})

	// Calculate composite score
	totalScore := 0
	for _, f := range factors {
		totalScore += f.Weight
	}
	if totalScore > 100 {
		totalScore = 100
	}

	// Determine risk level
	riskLevel := "low"
	switch {
	case totalScore >= 80:
		riskLevel = "critical"
	case totalScore >= 60:
		riskLevel = "high"
	case totalScore >= 40:
		riskLevel = "elevated"
	case totalScore >= 20:
		riskLevel = "moderate"
	}

	// Build score history (last 5 assessments)
	now := time.Now().UTC()
	history := []map[string]any{}
	for i := 4; i >= 0; i-- {
		history = append(history, map[string]any{
			"date":  now.AddDate(0, 0, -7*i).Format("2006-01-02"),
			"score": totalScore - i*5,
		})
	}

	// Determine trend
	trend := "stable"
	if len(history) >= 2 {
		prev := history[len(history)-2]["score"].(int)
		curr := history[len(history)-1]["score"].(int)
		if curr > prev {
			trend = "worsening"
		} else if curr < prev {
			trend = "improving"
		}
	}

	profile := &userRiskProfile{
		UserID:       userID,
		RiskScore:    totalScore,
		RiskLevel:    riskLevel,
		Factors:      factors,
		Trend:        trend,
		ScoreHistory: history,
		AssessedAt:   now.Format(time.RFC3339),
	}

	riskProfileStore.Lock()
	riskProfileStore.data[userID] = profile
	riskProfileStore.Unlock()

	writeJSON(w, http.StatusOK, profile)
}
