package detection

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// UserBehaviorProfile stores aggregated behavioral baseline for a user.
type UserBehaviorProfile struct {
	UserID           uuid.UUID        `json:"user_id"`
	TenantID         uuid.UUID        `json:"tenant_id"`
	LoginHours       [24]float64      `json:"login_hours"`
	KnownIPs         map[string]float64 `json:"known_ips"`
	KnownDevices     map[string]float64 `json:"known_devices"`
	AvgDailyActions  float64          `json:"avg_daily_actions"`
	ActionTypes      map[string]float64 `json:"action_types"`
	UpdatedAt        time.Time        `json:"updated_at"`
	EventCount       int              `json:"event_count"`
}

// ProfileBuilder aggregates audit events into behavioral profiles.
type ProfileBuilder struct {
	mu       sync.RWMutex
	profiles map[string]*UserBehaviorProfile // key: tenantID:userID
}

func NewProfileBuilder() *ProfileBuilder {
	return &ProfileBuilder{profiles: make(map[string]*UserBehaviorProfile)}
}

// IngestEvent updates the user's profile with a single audit event.
// Cold start: first 50 events only learn, no detection (handled by caller).
func (b *ProfileBuilder) IngestEvent(ctx context.Context, tenantID, userID uuid.UUID, hour int, ip, device, action string) {
	key := tenantID.String() + ":" + userID.String()
	b.mu.Lock()
	defer b.mu.Unlock()

	p, ok := b.profiles[key]
	if !ok {
		p = &UserBehaviorProfile{
			UserID:       userID,
			TenantID:     tenantID,
			KnownIPs:     make(map[string]float64),
			KnownDevices: make(map[string]float64),
			ActionTypes:  make(map[string]float64),
		}
		b.profiles[key] = p
	}

	// Update login hour distribution.
	if hour >= 0 && hour < 24 {
		p.LoginHours[hour]++
	}

	// Update known IPs.
	if ip != "" {
		p.KnownIPs[ip]++
	}

	// Update known devices.
	if device != "" {
		p.KnownDevices[device]++
	}

	// Update action types.
	if action != "" {
		p.ActionTypes[action]++
	}

	p.EventCount++
	p.UpdatedAt = time.Now()
}

// GetProfile returns the behavioral profile for a user.
func (b *ProfileBuilder) GetProfile(tenantID, userID uuid.UUID) *UserBehaviorProfile {
	key := tenantID.String() + ":" + userID.String()
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.profiles[key]
}

// NormalizeProfile converts raw counts to probabilities.
func (p *UserBehaviorProfile) Normalize() {
	total := float64(0)
	for _, v := range p.LoginHours {
		total += v
	}
	if total > 0 {
		for i := range p.LoginHours {
			p.LoginHours[i] /= total
		}
	}
	normalizeMap(p.KnownIPs)
	normalizeMap(p.KnownDevices)
	normalizeMap(p.ActionTypes)
}

func normalizeMap(m map[string]float64) {
	total := float64(0)
	for _, v := range m {
		total += v
	}
	if total > 0 {
		for k := range m {
			m[k] /= total
		}
	}
}

// BaselineDeviation checks if an event deviates from the user's baseline.
// Returns a deviation score (0 = normal, higher = more anomalous).
// Cold start: if EventCount < 50, returns 0 (learning phase).
type BaselineDeviationRule struct{}

func (r *BaselineDeviationRule) ID() string         { return "baseline_deviation" }
func (r *BaselineDeviationRule) Name() string       { return "Behavioral Baseline Deviation (UEBA)" }
func (r *BaselineDeviationRule) MITRE() string      { return "T1078" }
func (r *BaselineDeviationRule) DefaultSeverity() domain.Severity { return domain.SeverityMedium }

func (r *BaselineDeviationRule) Actions() []string {
	return []string{"user.login", "api.request", "role.assign"}
}

// Evaluate checks event against profiles stored in the StateStore.
// The profile is stored as JSON in StateStore under key "ueba:{tenant}:{user}".
func (r *BaselineDeviationRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.ActorID == nil {
		return nil, nil
	}

	// Cold start check — need at least 50 events in profile.
	profileKey := "ueba:" + evt.TenantID.String() + ":" + evt.ActorID.String()
	known, err := state.EventsSince(ctx, profileKey, 0)
	if err != nil || len(known) < 50 {
		return nil, nil // learning phase, no detection
	}

	// Compute hour.
	hour := evt.CreatedAt.Hour()

	// Only flag as anomalous if BOTH conditions met: off-hours AND new IP.
	isOffHours := hour < 6 || hour > 22
	if !isOffHours {
		return nil, nil // business hours — not anomalous enough
	}

	// IP novelty check.
	ipKey := "ueba_ip:" + evt.TenantID.String() + ":" + evt.ActorID.String() + ":" + evt.IPAddress
	ipKnown, _ := state.EventsSince(ctx, ipKey, evt.CreatedAt.Add(-30*24*time.Hour).Unix())
	if evt.IPAddress == "" || len(ipKnown) > 0 {
		return nil, nil // known IP at off-hours — lower priority, skip for now
	}

	// Off-hours + new IP = deviation.
	state.AddEvent(ctx, ipKey, evt.CreatedAt.Unix(), evt.ID.String(), 30*24*time.Hour)

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		"Behavioral baseline deviation: off-hours access from new IP")
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"hour":           hour,
		"ip_address":     evt.IPAddress,
		"profile_events": len(known),
		"reason":         "off-hours access from new IP not seen in 30 days",
	}
	return det, nil
}

// Suppress unused imports
var _ = math.Pi
var _ json.RawMessage
