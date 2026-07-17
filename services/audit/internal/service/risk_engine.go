package service

import (
	"sync"
	"time"
)

// RiskScore represents a computed risk assessment.
type RiskScore struct {
	Score            float64       // 0.0 (low) to 1.0 (high)
	Velocity         int           // events in last hour
	GeoAnomaly       bool          // impossible travel detected
	DeviceKnown      bool          // device fingerprint seen before
	NewIP            bool          // IP not seen before
	ThreatIntelHit   bool          // IP/email found in threat intel feed
	ThreatSeverity   string        // severity of the threat intel match
	Recommendations  []string      // e.g. "require_mfa", "block"
}

// RiskEngine evaluates risk based on velocity, geo, device, and threat intel signals.
type RiskEngine struct {
	mu             sync.RWMutex
	velocityStore  map[string][]time.Time // userID → event timestamps
	knownDevices   map[string]bool       // deviceFingerprint → seen
	knownIPs       map[string]bool       // ip → seen
	knownLocations map[string]string    // userID → last country
	threatChecker  ThreatIntelChecker   // optional: threat intel lookup
}

// ThreatIntelChecker allows the risk engine to query threat indicators.
type ThreatIntelChecker interface {
	CheckThreat(tenantID, indicatorType, value string) (severity string, confidence int, hit bool)
}

func NewRiskEngine() *RiskEngine {
	return &RiskEngine{
		velocityStore:  make(map[string][]time.Time),
		knownDevices:   make(map[string]bool),
		knownIPs:       make(map[string]bool),
		knownLocations: make(map[string]string),
	}
}

// RecordEvent tracks velocity and device/IP history.
func (e *RiskEngine) RecordEvent(userID, deviceFingerprint, ip string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()

	// Prune events older than 1 hour
	var recent []time.Time
	for _, t := range e.velocityStore[userID] {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	e.velocityStore[userID] = recent

	e.knownDevices[deviceFingerprint] = true
	e.knownIPs[ip] = true
}

// Evaluate computes a risk score from signals including threat intel.
func (e *RiskEngine) Evaluate(userID, deviceFingerprint, ip, country string) *RiskScore {
	e.mu.RLock()
	defer e.mu.RUnlock()

	score := &RiskScore{}

	// Velocity: count events in last hour
	now := time.Now()
	for _, t := range e.velocityStore[userID] {
		if now.Sub(t) < time.Hour {
			score.Velocity++
		}
	}
	if score.Velocity > 20 {
		score.Score += 0.4
		score.Recommendations = append(score.Recommendations, "require_mfa")
	}

	// Device fingerprint
	score.DeviceKnown = e.knownDevices[deviceFingerprint]
	if !score.DeviceKnown {
		score.Score += 0.3
		score.Recommendations = append(score.Recommendations, "verify_device")
	}

	// New IP
	score.NewIP = !e.knownIPs[ip]
	if score.NewIP {
		score.Score += 0.2
	}

	// Geo anomaly: impossible travel (different country than last known)
	lastCountry := e.knownLocations[userID]
	if lastCountry != "" && country != "" && lastCountry != country {
		score.GeoAnomaly = true
		score.Score += 0.3
		score.Recommendations = append(score.Recommendations, "require_mfa")
	}

	// Threat intel: check IP against known-bad indicators.
	if e.threatChecker != nil && ip != "" {
		severity, _, hit := e.threatChecker.CheckThreat("", "ip", ip)
		if hit {
			score.ThreatIntelHit = true
			score.ThreatSeverity = severity
			switch severity {
			case "critical":
				score.Score += 0.5
				score.Recommendations = append(score.Recommendations, "block_session")
			case "high":
				score.Score += 0.35
				score.Recommendations = append(score.Recommendations, "step_up_mfa")
			case "medium":
				score.Score += 0.2
				score.Recommendations = append(score.Recommendations, "require_mfa")
			default:
				score.Score += 0.1
			}
		}
	}

	// Cap at 1.0
	if score.Score > 1.0 {
		score.Score = 1.0
	}
	if score.Score >= 0.8 {
		score.Recommendations = append(score.Recommendations, "block")
	}

	return score
}

// SetKnownLocation records the user's last known country.
func (e *RiskEngine) SetKnownLocation(userID, country string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.knownLocations[userID] = country
}

// SetThreatIntelChecker injects the threat intel lookup for risk scoring.
func (e *RiskEngine) SetThreatIntelChecker(checker ThreatIntelChecker) {
	e.threatChecker = checker
}

// Reset clears all risk data (for testing).
func (e *RiskEngine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.velocityStore = make(map[string][]time.Time)
	e.knownDevices = make(map[string]bool)
	e.knownIPs = make(map[string]bool)
	e.knownLocations = make(map[string]string)
}
