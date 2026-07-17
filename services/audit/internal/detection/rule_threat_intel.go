package detection

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// ThreatIntelChecker is the interface for looking up threat indicators during ITDR evaluation.
// Implemented by repository.ThreatIntelRepository (adapted via wrapper).
type ThreatIntelChecker interface {
	// CheckIndicator queries for a specific indicator match (ip/email/domain).
	// Returns a ThreatIntelHit if found, or nil if no match.
	CheckIndicator(ctx context.Context, tenantID uuid.UUID, indType, value string) (*ThreatIntelHit, error)
}

// ThreatIntelHit is a lightweight representation of a matched indicator.
type ThreatIntelHit struct {
	IndicatorType  string
	IndicatorValue string
	Severity       string  // low | medium | high | critical
	Confidence     int     // 0-100
	SourceID       uuid.UUID
}

// ThreatIntelRule checks whether the event's source IP or actor email
// appears in the threat_indicators table. If matched, raises a detection
// enriched with the threat intel details.
type ThreatIntelRule struct {
	checker ThreatIntelChecker
}

func NewThreatIntelRule(checker ThreatIntelChecker) *ThreatIntelRule {
	return &ThreatIntelRule{checker: checker}
}

func (r *ThreatIntelRule) ID() string                    { return "threat_intel_hit" }
func (r *ThreatIntelRule) Name() string                  { return "Threat Intelligence Match" }
func (r *ThreatIntelRule) MITRE() string                 { return "T1589" }
func (r *ThreatIntelRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *ThreatIntelRule) Actions() []string {
	return []string{"user.login", "user.access", "api.call", "token.exchange", "sso.assertion"}
}

// Evaluate checks the event's IP address and actor name (email) against threat indicators.
func (r *ThreatIntelRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if r.checker == nil {
		return nil, nil
	}

	// Check source IP.
	if evt.IPAddress != "" {
		hit, err := r.checker.CheckIndicator(ctx, evt.TenantID, "ip", evt.IPAddress)
		if err == nil && hit != nil {
			return r.buildDetection(evt, hit, "ip"), nil
		}
	}

	// Check actor email/name (may be an email or credential hash).
	if evt.ActorName != "" {
		hit, err := r.checker.CheckIndicator(ctx, evt.TenantID, "email", evt.ActorName)
		if err == nil && hit != nil {
			return r.buildDetection(evt, hit, "email"), nil
		}
	}

	// Check user agent as domain indicator.
	if evt.UserAgent != "" {
		hit, err := r.checker.CheckIndicator(ctx, evt.TenantID, "domain", evt.UserAgent)
		if err == nil && hit != nil {
			return r.buildDetection(evt, hit, "domain"), nil
		}
	}

	return nil, nil
}

func (r *ThreatIntelRule) buildDetection(evt *domain.AuditEvent, hit *ThreatIntelHit, indType string) *domain.Detection {
	severity := domain.Severity(hit.Severity)
	if severity == "" {
		severity = r.DefaultSeverity()
	}

	title := fmt.Sprintf("Threat intel match: %s %s in feed", indType, hit.IndicatorValue)

	det := domain.NewDetection(evt.TenantID, r.ID(), severity, title)
	det.ActorID = evt.ActorID
	det.Detail = map[string]any{
		"indicator_type":   hit.IndicatorType,
		"indicator_value":  hit.IndicatorValue,
		"confidence":       hit.Confidence,
		"source_id":        hit.SourceID,
		"event_ip":         evt.IPAddress,
		"event_action":     evt.Action,
		"matched_on":       indType,
		"recommendation":   recommendForSeverity(severity),
	}
	return det
}

// recommendForSeverity returns the CAE action for a given threat intel severity.
func recommendForSeverity(s domain.Severity) string {
	switch s {
	case domain.SeverityCritical:
		return "block_session"
	case domain.SeverityHigh:
		return "step_up_mfa"
	case domain.SeverityMedium:
		return "require_mfa"
	default:
		return "log_only"
	}
}
