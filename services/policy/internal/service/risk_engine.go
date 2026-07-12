package service

import (
	"sync"
	"time"
)

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type RiskAction string

const (
	ActionAllow    RiskAction = "allow"
	ActionStepUp   RiskAction = "step_up"
	ActionBlock    RiskAction = "block"
)

type RiskFactor struct {
	Name    string  `json:"name"`
	Weight  float64 `json:"weight"`
	Enabled bool    `json:"enabled"`
}

type RiskScore struct {
	Score            float64      `json:"score"`
	Level            RiskLevel    `json:"level"`
	Action           RiskAction   `json:"action"`
	TriggeredFactors []string     `json:"triggered_factors"`
}

type RiskContext struct {
	UserID          string    `json:"user_id"`
	IPAddress       string    `json:"ip_address"`
	GeoLocation     string    `json:"geo_location"`
	LastGeoLocation string    `json:"last_geo_location"`
	LastLoginAt     time.Time `json:"last_login_at"`
	DeviceKnown     bool      `json:"device_known"`
	FailedAttempts  int       `json:"failed_attempts"`
	HourOfDay       int       `json:"hour_of_day"`
}

type RiskEngine struct {
	mu      sync.RWMutex
	factors []RiskFactor
}

func NewRiskEngine() *RiskEngine {
	return &RiskEngine{
		factors: []RiskFactor{
			{Name: "geo_velocity", Weight: 0.25, Enabled: true},
			{Name: "impossible_travel", Weight: 0.35, Enabled: true},
			{Name: "new_device", Weight: 0.15, Enabled: true},
			{Name: "anomalous_time", Weight: 0.10, Enabled: true},
			{Name: "failed_attempts", Weight: 0.15, Enabled: true},
		},
	}
}

func (re *RiskEngine) EvaluateRisk(ctx RiskContext) RiskScore {
	re.mu.RLock()
	defer re.mu.RUnlock()

	var score float64
	var triggered []string

	for _, f := range re.factors {
		if !f.Enabled {
			continue
		}
		if isFactorTriggered(f.Name, ctx) {
			score += f.Weight
			triggered = append(triggered, f.Name)
		}
	}

	level, action := scoreToLevel(score)
	return RiskScore{
		Score:            score,
		Level:            level,
		Action:           action,
		TriggeredFactors: triggered,
	}
}

func isFactorTriggered(name string, ctx RiskContext) bool {
	switch name {
	case "geo_velocity":
		return ctx.GeoLocation != "" && ctx.LastGeoLocation != "" && ctx.GeoLocation != ctx.LastGeoLocation
	case "impossible_travel":
		if ctx.LastLoginAt.IsZero() || ctx.GeoLocation == "" || ctx.LastGeoLocation == "" {
			return false
		}
		elapsed := time.Since(ctx.LastLoginAt)
		return elapsed < 30*time.Minute && ctx.GeoLocation != ctx.LastGeoLocation
	case "new_device":
		return !ctx.DeviceKnown
	case "anomalous_time":
		return ctx.HourOfDay < 6 || ctx.HourOfDay > 22
	case "failed_attempts":
		return ctx.FailedAttempts >= 3
	}
	return false
}

func scoreToLevel(score float64) (RiskLevel, RiskAction) {
	switch {
	case score >= 0.7:
		return RiskCritical, ActionBlock
	case score >= 0.5:
		return RiskHigh, ActionBlock
	case score >= 0.3:
		return RiskMedium, ActionStepUp
	default:
		return RiskLow, ActionAllow
	}
}