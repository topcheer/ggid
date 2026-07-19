package detection

import (
	"log/slog"
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
)

// CredentialStuffingRule: same IP, 10+ different accounts in 10 minutes.
type CredentialStuffingRule struct{}

func (r *CredentialStuffingRule) ID() string         { return "credential_stuffing" }
func (r *CredentialStuffingRule) Name() string       { return "Credential Stuffing" }
func (r *CredentialStuffingRule) MITRE() string      { return "T1110.004" }
func (r *CredentialStuffingRule) DefaultSeverity() domain.Severity { return domain.SeverityCritical }
func (r *CredentialStuffingRule) Actions() []string  { return []string{"user.login"} }

func (r *CredentialStuffingRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "failure" || evt.IPAddress == "" {
		return nil, nil
	}

	// Track unique accounts per IP.
	key := fmt.Sprintf("cs:%s", evt.IPAddress)
	windowMin := 10
	threshold := 10
	if t, ok := cfg.Threshold["min_accounts"].(float64); ok && t > 0 {
		threshold = int(t)
	}

	// Use EventsSince to count unique actor names in window.
	member := evt.ActorName
	if member == "" && evt.ActorID != nil {
		member = evt.ActorID.String()
	}
	if member == "" {
		return nil, nil
	}

	if err := state.AddEvent(ctx, key, evt.CreatedAt.Unix(), member, time.Duration(windowMin)*time.Minute); err != nil { slog.Debug("detection: AddEvent failed", "error", err) }
	members, err := state.EventsSince(ctx, key, evt.CreatedAt.Add(-time.Duration(windowMin)*time.Minute).Unix())
	if err != nil {
		return nil, err
	}

	// Count unique accounts.
	seen := make(map[string]bool)
	for _, m := range members {
		seen[m] = true
	}
	if len(seen) < threshold {
		return nil, nil
	}

	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(), "Credential stuffing attack detected")
	det.Detail = map[string]any{
		"unique_accounts": len(seen),
		"window_minutes":  windowMin,
		"ip_address":      evt.IPAddress,
		"threshold":       threshold,
	}
	if evt.ActorID != nil {
		actorID := *evt.ActorID
		det.ActorID = &actorID
	}
	return det, nil
}
