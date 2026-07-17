package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
)

// OffHoursAdminRule detects privileged operations during off-hours (22:00-06:00 local).
// Severity: medium (may be legitimate, but warrants review).
type OffHoursAdminRule struct{}

func (r *OffHoursAdminRule) ID() string         { return "offhours_admin" }
func (r *OffHoursAdminRule) Name() string       { return "Off-Hours Administrative Activity" }
func (r *OffHoursAdminRule) MITRE() string      { return "T1078" }
func (r *OffHoursAdminRule) DefaultSeverity() domain.Severity { return domain.SeverityMedium }
func (r *OffHoursAdminRule) Actions() []string {
	return []string{"role.assign", "role.revoke", "user.create", "user.delete", "policy.update", "break_glass.activate"}
}

func (r *OffHoursAdminRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "success" {
		return nil, nil
	}
	if evt.ActorID == nil {
		return nil, nil
	}

	// Check off-hours window: 22:00-06:00 UTC by default.
	startHour := 22
	endHour := 6
	if h, ok := cfg.Threshold["start_hour"].(float64); ok {
		startHour = int(h)
	}
	if h, ok := cfg.Threshold["end_hour"].(float64); ok {
		endHour = int(h)
	}

	hour := evt.CreatedAt.Hour()
	inOffHours := hour >= startHour || hour < endHour
	if !inOffHours {
		return nil, nil
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("Administrative action '%s' performed during off-hours (%02d:00 UTC)", evt.Action, hour))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"action":    evt.Action,
		"hour_utc":  hour,
		"ip_address": evt.IPAddress,
		"actor_name": evt.ActorName,
	}
	return det, nil
}

// NewDevicePrivilegedRule detects privileged actions from a previously unseen device/IP.
// Uses StateStore to track seen IPs per user. First appearance triggers a detection.
type NewDevicePrivilegedRule struct{}

func (r *NewDevicePrivilegedRule) ID() string         { return "new_device_privileged" }
func (r *NewDevicePrivilegedRule) Name() string       { return "Privileged Action from New Device" }
func (r *NewDevicePrivilegedRule) MITRE() string      { return "T1078" }
func (r *NewDevicePrivilegedRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *NewDevicePrivilegedRule) Actions() []string {
	return []string{"role.assign", "policy.update", "break_glass.activate", "session.revoke"}
}

func (r *NewDevicePrivilegedRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "success" {
		return nil, nil
	}
	if evt.ActorID == nil || evt.IPAddress == "" {
		return nil, nil
	}

	// Track seen device+user pairs in StateStore (7-day window).
	key := fmt.Sprintf("dev:%s:%s", evt.ActorID, evt.IPAddress)
	windowTTL := 7 * 24 * time.Hour
	if d, ok := cfg.Threshold["device_remember_days"].(float64); ok && d > 0 {
		windowTTL = time.Duration(d) * 24 * time.Hour
	}

	// Check if this IP is already seen for this user.
	known, err := state.EventsSince(ctx, key, evt.CreatedAt.Add(-windowTTL).Unix())
	if err != nil {
		return nil, err
	}

	// If we've seen this device before, no detection.
	if len(known) > 0 {
		return nil, nil
	}

	// Record this device for future checks.
	_ = state.AddEvent(ctx, key, evt.CreatedAt.Unix(), evt.ID.String(), windowTTL)

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("Privileged action '%s' from new IP %s", evt.Action, evt.IPAddress))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"action":     evt.Action,
		"ip_address": evt.IPAddress,
		"user_agent": evt.UserAgent,
		"actor_name": evt.ActorName,
	}
	return det, nil
}

// TokenReplayRule detects token replay attempts: the same jti appearing in
// multiple requests in a short window. Relies on audit events with action
// "cae.jti_revoke" or "session.revoke" followed by continued API usage.
type TokenReplayRule struct{}

func (r *TokenReplayRule) ID() string         { return "token_replay" }
func (r *TokenReplayRule) Name() string       { return "Revoked Token Replay Attempt" }
func (r *TokenReplayRule) MITRE() string      { return "T1550" }
func (r *TokenReplayRule) DefaultSeverity() domain.Severity { return domain.SeverityCritical }
func (r *TokenReplayRule) Actions() []string {
	return []string{"api.request", "user.api_call"}
}

func (r *TokenReplayRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "denied" {
		return nil, nil
	}
	if evt.ActorID == nil {
		return nil, nil
	}

	// Check if the metadata contains a CAE revocation reason.
	caeReason, _ := evt.Metadata["cae_reason"].(string)
	if caeReason == "" {
		// Also check for "session revoked" in the metadata.
		reason, _ := evt.Metadata["reason"].(string)
		if reason == "" {
			return nil, nil
		}
		caeReason = reason
	}

	// Count denied requests with revocation reason per user.
	threshold := int64(3) // 3 denied requests after revocation = replay attempt
	if t, ok := cfg.Threshold["min_denied"].(float64); ok && t > 0 {
		threshold = int64(t)
	}

	key := fmt.Sprintf("replay:%s", evt.ActorID)
	windowMin := 10
	count, err := state.Incr(ctx, key, time.Duration(windowMin)*time.Minute)
	if err != nil {
		return nil, err
	}
	if count < threshold {
		return nil, nil
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("Token replay: %d denied requests after session revocation", count))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"denied_count":   count,
		"window_minutes": windowMin,
		"cae_reason":     caeReason,
		"ip_address":     evt.IPAddress,
	}
	return det, nil
}
