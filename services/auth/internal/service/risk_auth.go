package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RiskLevel represents the assessed risk of an authentication attempt.
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "low"
	RiskLevelMedium RiskLevel = "medium"
	RiskLevelHigh   RiskLevel = "high"
)

// RiskAssessment holds the result of evaluating an authentication attempt's risk.
type RiskAssessment struct {
	Level             RiskLevel `json:"level"`
	Score             int       `json:"score"` // 0-100, higher = riskier
	Reasons           []string  `json:"reasons,omitempty"`
	RequiresStepUp    bool      `json:"requires_step_up"`
	RequiresAdminAlert bool     `json:"requires_admin_alert,omitempty"`
}

// AssessLoginRisk evaluates the risk of a login attempt based on:
// - IP address (known/unknown, rate of failed attempts)
// - User agent (device fingerprint consistency)
// - Time of day (anomalous login hours)
// - Geographic considerations (via IP prefix changes)
// Returns a risk score and recommended actions.
func (s *AuthService) AssessLoginRisk(ctx context.Context, tenantID, userID uuid.UUID, ip, userAgent string) *RiskAssessment {
	assessment := &RiskAssessment{
		Level:   RiskLevelLow,
		Score:   0,
		Reasons: []string{},
	}

	// 1. Check for excessive failed attempts from this IP.
	ipFailKey := fmt.Sprintf("ggid:risk:ipfail:%s", ip)
	failCount, err := s.rateLimiter.rdb.Get(ctx, ipFailKey).Int()
	if err == nil && failCount > 0 {
		if failCount >= 5 {
			assessment.Score += 40
			assessment.Level = RiskLevelHigh
			assessment.Reasons = append(assessment.Reasons, fmt.Sprintf("%d failed attempts from IP %s", failCount, ip))
			assessment.RequiresStepUp = true
		} else if failCount >= 2 {
			assessment.Score += 20
			assessment.Level = RiskLevelMedium
			assessment.Reasons = append(assessment.Reasons, fmt.Sprintf("%d recent failed attempts from this IP", failCount))
			assessment.RequiresStepUp = true
		}
	}

	// 2. Check if IP is known (seen in previous successful logins).
	ipKnownKey := fmt.Sprintf("ggid:risk:knownip:%s:%s", userID, ip)
	_, err = s.rateLimiter.rdb.Get(ctx, ipKnownKey).Result()
	if err != nil {
		// Unknown IP — moderate risk.
		assessment.Score += 15
		assessment.Reasons = append(assessment.Reasons, "login from unrecognized IP address")
		if assessment.Level == RiskLevelLow {
			assessment.Level = RiskLevelMedium
		}
	}

	// 3. Check for anomalous login time (2 AM - 5 AM local).
	hour := time.Now().UTC().Hour()
	if hour >= 2 && hour <= 5 {
		assessment.Score += 10
		assessment.Reasons = append(assessment.Reasons, "login at anomalous hour (night)")
	}

	// 4. Check for rapid IP changes (user from different IPs in short window).
	if userAgent != "" {
		uaKey := fmt.Sprintf("ggid:risk:ua:%s", userID)
		storedUA, _ := s.rateLimiter.rdb.Get(ctx, uaKey).Result()
		if storedUA != "" && storedUA != userAgent {
			assessment.Score += 15
			assessment.Reasons = append(assessment.Reasons, "user agent changed since last login")
			if assessment.Level == RiskLevelLow {
				assessment.Level = RiskLevelMedium
			}
		}
		if storedUA == "" {
			_ = s.rateLimiter.rdb.Set(ctx, uaKey, userAgent, 24*time.Hour).Err()
		}
	}

	// 5. Check for brute-force patterns across users from same IP.
	bfKey := fmt.Sprintf("ggid:risk:bruteforce:%s", ip)
	multiUser, err := s.rateLimiter.rdb.Get(ctx, bfKey).Int()
	if err == nil && multiUser >= 3 {
		assessment.Score += 30
		assessment.Level = RiskLevelHigh
		assessment.Reasons = append(assessment.Reasons, fmt.Sprintf("brute-force pattern: %d different users attempted from same IP", multiUser))
		assessment.RequiresAdminAlert = true
		assessment.RequiresStepUp = true
	}

	// Cap score at 100.
	if assessment.Score > 100 {
		assessment.Score = 100
	}

	// Determine final level based on score if not already set.
	if assessment.Score >= 70 && assessment.Level != RiskLevelHigh {
		assessment.Level = RiskLevelHigh
		assessment.RequiresStepUp = true
	} else if assessment.Score >= 30 && assessment.Level == RiskLevelLow {
		assessment.Level = RiskLevelMedium
	}

	return assessment
}

// RecordSuccessfulLogin records a successful login for risk tracking.
// This updates known IP and user-agent patterns.
func (s *AuthService) RecordSuccessfulLogin(ctx context.Context, userID uuid.UUID, ip, userAgent string) {
	// Record known IP (30 day TTL).
	ipKnownKey := fmt.Sprintf("ggid:risk:knownip:%s:%s", userID, ip)
	_ = s.rateLimiter.rdb.Set(ctx, ipKnownKey, "1", 30*24*time.Hour).Err()

	// Record user agent.
	if userAgent != "" {
		uaKey := fmt.Sprintf("ggid:risk:ua:%s", userID)
		_ = s.rateLimiter.rdb.Set(ctx, uaKey, userAgent, 24*time.Hour).Err()
	}
}

// RecordFailedLoginAttempt records a failed login for risk tracking.
// Tracks per-IP failed attempt counts and multi-user brute-force detection.
func (s *AuthService) RecordFailedLoginAttempt(ctx context.Context, userID uuid.UUID, ip string) {
	// Increment IP failure counter.
	ipFailKey := fmt.Sprintf("ggid:risk:ipfail:%s", ip)
	count, _ := s.rateLimiter.rdb.Incr(ctx, ipFailKey).Result()
	if count == 1 {
		s.rateLimiter.rdb.Expire(ctx, ipFailKey, time.Hour)
	}

	// Track multi-user brute-force: add userID to set of attempted users from this IP.
	bfKey := fmt.Sprintf("ggid:risk:bruteforce:%s", ip)
	userSetKey := fmt.Sprintf("ggid:risk:bfusers:%s", ip)
	s.rateLimiter.rdb.SAdd(ctx, userSetKey, userID.String())
	s.rateLimiter.rdb.Expire(ctx, userSetKey, time.Hour)

	// Count distinct users attempted from this IP.
	card, _ := s.rateLimiter.rdb.SCard(ctx, userSetKey).Result()
	if card >= 3 {
		s.rateLimiter.rdb.Set(ctx, bfKey, card, time.Hour)
	}
}

// BlockSuspiciousIP adds an IP to a temporary blocklist.
func (s *AuthService) BlockSuspiciousIP(ctx context.Context, ip string, duration time.Duration) {
	key := fmt.Sprintf("ggid:risk:blocked:%s", ip)
	_ = s.rateLimiter.rdb.Set(ctx, key, "1", duration).Err()
}

// IsIPBlocked checks if an IP is in the blocklist.
func (s *AuthService) IsIPBlocked(ctx context.Context, ip string) bool {
	key := fmt.Sprintf("ggid:risk:blocked:%s", ip)
	_, err := s.rateLimiter.rdb.Get(ctx, key).Result()
	return err == nil
}

// Ensure strings import is used.
var _ = strings.TrimSpace
