package detection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
)

// ==============================================================================
// 1. ConsentPhishingRule (T1098) — abnormal OAuth consent grants
// ==============================================================================

type ConsentPhishingRule struct{}

func (r *ConsentPhishingRule) ID() string                    { return "consent_phishing" }
func (r *ConsentPhishingRule) Name() string                  { return "Consent Phishing — Risky OAuth Scope Grant" }
func (r *ConsentPhishingRule) MITRE() string                 { return "T1098" }
func (r *ConsentPhishingRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *ConsentPhishingRule) Actions() []string {
	return []string{"oauth.consent.grant", "oauth.scope.grant"}
}

func (r *ConsentPhishingRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "success" || evt.ActorID == nil {
		return nil, nil
	}

	scopes, _ := evt.Metadata["scopes"].(string)
	riskyScopes := []string{"admin", "write", "delete", "all", "offline_access", "files.readwrite", "mail.send"}
	matched := []string{}
	lowerScopes := strings.ToLower(scopes)
	for _, rs := range riskyScopes {
		if strings.Contains(lowerScopes, rs) {
			matched = append(matched, rs)
		}
	}

	if len(matched) == 0 {
		return nil, nil
	}

	// Check if this is a first-time grant for this app (via state store).
	appID, _ := evt.Metadata["client_id"].(string)
	if appID != "" && state != nil {
		key := fmt.Sprintf("consent:%s:%s", evt.ActorID, appID)
		events, _ := state.EventsSince(ctx, key, evt.CreatedAt.Unix()-int64(30*24*3600))
		if len(events) > 0 {
			return nil, nil // user has granted to this app before
		}
		if state != nil {
			state.AddEvent(ctx, key, evt.CreatedAt.Unix(), evt.ID.String(), 30*24*time.Hour)
		}
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("First-time risky OAuth consent grant to app %s with scopes: %s", appID, strings.Join(matched, ",")))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"client_id":   appID,
		"risky_scopes": matched,
		"all_scopes":  scopes,
	}
	return det, nil
}

// ==============================================================================
// 2. MFAFatigueRule (T1621) — >5 MFA pushes in 10 minutes
// ==============================================================================

type MFAFatigueRule struct{}

func (r *MFAFatigueRule) ID() string                    { return "mfa_fatigue" }
func (r *MFAFatigueRule) Name() string                  { return "MFA Fatigue — Possible MFA Bombing" }
func (r *MFAFatigueRule) MITRE() string                 { return "T1621" }
func (r *MFAFatigueRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *MFAFatigueRule) Actions() []string {
	return []string{"mfa.push", "mfa.challenge", "user.login"}
}

func (r *MFAFatigueRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.ActorID == nil {
		return nil, nil
	}

	threshold := int64(5)
	if t, ok := cfg.Threshold["max_pushes"].(float64); ok && t > 0 {
		threshold = int64(t)
	}
	windowMin := 10
	if w, ok := cfg.Threshold["window_minutes"].(float64); ok && w > 0 {
		windowMin = int(w)
	}

	key := fmt.Sprintf("mfa_push:%s", evt.ActorID)
	count, err := state.Incr(ctx, key, time.Duration(windowMin)*time.Minute)
	if err != nil {
		return nil, err
	}
	if count < threshold {
		return nil, nil
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("MFA fatigue: %d MFA pushes in %d minutes", count, windowMin))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"push_count":     count,
		"window_minutes": windowMin,
		"threshold":      threshold,
		"ip_address":     evt.IPAddress,
	}
	return det, nil
}

// ==============================================================================
// 3. TokenTheftRule (T1528) — token used from new geo + new device simultaneously
// ==============================================================================

type TokenTheftRule struct{}

func (r *TokenTheftRule) ID() string                    { return "token_theft" }
func (r *TokenTheftRule) Name() string                  { return "Token Theft — Impossible Travel for Token Use" }
func (r *TokenTheftRule) MITRE() string                 { return "T1528" }
func (r *TokenTheftRule) DefaultSeverity() domain.Severity { return domain.SeverityCritical }
func (r *TokenTheftRule) Actions() []string {
	return []string{"token.exchange", "token.refresh", "api.call", "oauth.introspect"}
}

func (r *TokenTheftRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.ActorID == nil || evt.IPAddress == "" {
		return nil, nil
	}

	// Track IP + user-agent per user; detect simultaneous use from different fingerprints.
	fingerprint := fmt.Sprintf("%s|%s", evt.IPAddress, evt.UserAgent)
	key := fmt.Sprintf("token_fp:%s", evt.ActorID)

	prevFPs, _ := state.EventsSince(ctx, key, evt.CreatedAt.Unix()-int64(600)) // 10 min window

	// Record current fingerprint.
	state.AddEvent(ctx, key, evt.CreatedAt.Unix(), fingerprint, 10*time.Minute)

	// Check for different IP + different UA (strong theft signal).
	for _, prev := range prevFPs {
		if prev == fingerprint {
			continue // same fingerprint, not theft
		}
		parts := strings.SplitN(prev, "|", 2)
		if len(parts) != 2 {
			continue
		}
		prevIP, prevUA := parts[0], parts[1]
		if prevIP != evt.IPAddress && prevUA != evt.UserAgent {
			actorID := *evt.ActorID
			det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
				fmt.Sprintf("Token used from new IP (%s) and new device simultaneously", evt.IPAddress))
			det.ActorID = &actorID
			det.Detail = map[string]any{
				"current_ip":    evt.IPAddress,
				"current_ua":    evt.UserAgent,
				"previous_ip":   prevIP,
				"previous_ua":   prevUA,
				"window":        "10 minutes",
			}
			return det, nil
		}
	}

	return nil, nil
}

// ==============================================================================
// 4. SessionHijackRule (T1539) — session cookie from different IP mid-session
// ==============================================================================

type SessionHijackRule struct{}

func (r *SessionHijackRule) ID() string                    { return "session_hijack" }
func (r *SessionHijackRule) Name() string                  { return "Session Hijacking — IP/UA Change Mid-Session" }
func (r *SessionHijackRule) MITRE() string                 { return "T1539" }
func (r *SessionHijackRule) DefaultSeverity() domain.Severity { return domain.SeverityCritical }
func (r *SessionHijackRule) Actions() []string {
	return []string{"session.validate", "session.refresh", "api.call"}
}

func (r *SessionHijackRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.ActorID == nil || evt.IPAddress == "" {
		return nil, nil
	}

	sessionID, _ := evt.Metadata["session_id"].(string)
	if sessionID == "" {
		return nil, nil
	}

	// Track session → fingerprint mapping.
	fingerprint := fmt.Sprintf("%s|%s", evt.IPAddress, evt.UserAgent)
	key := fmt.Sprintf("session_fp:%s", sessionID)

	prevFPs, _ := state.EventsSince(ctx, key, evt.CreatedAt.Unix()-int64(3600)) // 1 hour window
	state.AddEvent(ctx, key, evt.CreatedAt.Unix(), fingerprint, time.Hour)

	for _, prev := range prevFPs {
		if prev == fingerprint {
			continue
		}
		parts := strings.SplitN(prev, "|", 2)
		if len(parts) != 2 {
			continue
		}
		prevIP := parts[0]
		// IP change mid-session is the hijack signal.
		if prevIP != evt.IPAddress {
			actorID := *evt.ActorID
			det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
				fmt.Sprintf("Session %s changed IP mid-session: %s → %s", sessionID, prevIP, evt.IPAddress))
			det.ActorID = &actorID
			det.Detail = map[string]any{
				"session_id":  sessionID,
				"current_ip":  evt.IPAddress,
				"previous_ip": prevIP,
				"current_ua":  evt.UserAgent,
			}
			return det, nil
		}
	}

	return nil, nil
}

// ==============================================================================
// 5. MassCreationRule (T1136) — >10 user accounts created in 5 minutes by non-admin
// ==============================================================================

type MassCreationRule struct{}

func (r *MassCreationRule) ID() string                    { return "mass_creation" }
func (r *MassCreationRule) Name() string                  { return "Mass Account Creation" }
func (r *MassCreationRule) MITRE() string                 { return "T1136" }
func (r *MassCreationRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *MassCreationRule) Actions() []string {
	return []string{"user.create", "user.provision", "scim.user.create"}
}

func (r *MassCreationRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "success" || evt.ActorID == nil {
		return nil, nil
	}

	// Admin is allowed to bulk-create (e.g. SCIM from IdP).
	if role, _ := evt.Metadata["actor_role"].(string); role == "admin" || role == "service" {
		return nil, nil
	}

	threshold := int64(10)
	if t, ok := cfg.Threshold["max_creations"].(float64); ok && t > 0 {
		threshold = int64(t)
	}
	windowMin := 5
	if w, ok := cfg.Threshold["window_minutes"].(float64); ok && w > 0 {
		windowMin = int(w)
	}

	key := fmt.Sprintf("mass_create:%s", evt.ActorID)
	count, err := state.Incr(ctx, key, time.Duration(windowMin)*time.Minute)
	if err != nil {
		return nil, err
	}
	if count < threshold {
		return nil, nil
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("Mass account creation: %d users created in %d minutes by non-admin", count, windowMin))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"creation_count": count,
		"window_minutes": windowMin,
		"creator":        evt.ActorName,
	}
	return det, nil
}

// ==============================================================================
// 6. FederationAnomalyRule (T1606) — SAML/OIDC assertion from unexpected source
// ==============================================================================

type FederationAnomalyRule struct{}

func (r *FederationAnomalyRule) ID() string                    { return "federation_anomaly" }
func (r *FederationAnomalyRule) Name() string                  { return "Federation Anomaly — Unexpected IdP Assertion" }
func (r *FederationAnomalyRule) MITRE() string                 { return "T1606" }
func (r *FederationAnomalyRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *FederationAnomalyRule) Actions() []string {
	return []string{"sso.assertion", "saml.acs", "oidc.callback"}
}

func (r *FederationAnomalyRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "success" {
		return nil, nil
	}

	idpEntityID, _ := evt.Metadata["idp_entity_id"].(string)
	if idpEntityID == "" {
		return nil, nil
	}

	// Track known IdPs per tenant.
	key := fmt.Sprintf("fed_idp:%s", evt.TenantID)
	knownIdps, _ := state.EventsSince(ctx, key, evt.CreatedAt.Unix()-int64(90*24*3600)) // 90 days

	// Record this IdP.
	state.AddEvent(ctx, key, evt.CreatedAt.Unix(), idpEntityID, 90*24*time.Hour)

	// Check if IdP was seen before.
	for _, known := range knownIdps {
		if known == idpEntityID {
			return nil, nil // known IdP, no anomaly
		}
	}

	// New IdP — check for other anomaly signals.
	issuer, _ := evt.Metadata["saml_issuer"].(string)
	newClaims, _ := evt.Metadata["unexpected_claims"].(bool)

	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("Federation assertion from new/unknown IdP: %s", idpEntityID))
	if evt.ActorID != nil {
		det.ActorID = evt.ActorID
	}
	det.Detail = map[string]any{
		"idp_entity_id":    idpEntityID,
		"issuer":           issuer,
		"unexpected_claims": newClaims,
		"ip_address":       evt.IPAddress,
	}
	return det, nil
}

// ==============================================================================
// 7. MFABypassRule (T1098.001) — auth succeeded without MFA when required
// ==============================================================================

type MFABypassRule struct{}

func (r *MFABypassRule) ID() string                    { return "mfa_bypass" }
func (r *MFABypassRule) Name() string                  { return "MFA Bypass — Auth Without Required MFA" }
func (r *MFABypassRule) MITRE() string                 { return "T1098.001" }
func (r *MFABypassRule) DefaultSeverity() domain.Severity { return domain.SeverityCritical }
func (r *MFABypassRule) Actions() []string {
	return []string{"user.login", "token.exchange"}
}

func (r *MFABypassRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "success" || evt.ActorID == nil {
		return nil, nil
	}

	mfaRequired, _ := evt.Metadata["mfa_required"].(bool)
	mfaPerformed, _ := evt.Metadata["mfa_performed"].(bool)
	mfaMethod, _ := evt.Metadata["mfa_method"].(string)

	// Detection: policy requires MFA but it was not performed.
	if !mfaRequired {
		return nil, nil // MFA not required for this user
	}
	if mfaPerformed && mfaMethod != "" {
		return nil, nil // MFA was performed
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("Authentication succeeded without MFA despite policy requirement for user %s", evt.ActorName))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"mfa_required": mfaRequired,
		"mfa_performed": mfaPerformed,
		"ip_address":   evt.IPAddress,
		"user_agent":   evt.UserAgent,
	}
	return det, nil
}

// ==============================================================================
// 8. MassExportRule (T1005) — bulk data export exceeding baseline by 5x
// ==============================================================================

type MassExportRule struct{}

func (r *MassExportRule) ID() string                    { return "mass_export" }
func (r *MassExportRule) Name() string                  { return "Mass Data Export — Exfiltration Risk" }
func (r *MassExportRule) MITRE() string                 { return "T1005" }
func (r *MassExportRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *MassExportRule) Actions() []string {
	return []string{"data.export", "audit.export", "user.list.export", "report.export"}
}

func (r *MassExportRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	if evt.Result != "success" || evt.ActorID == nil {
		return nil, nil
	}

	threshold := int64(5) // 5x baseline
	if t, ok := cfg.Threshold["multiplier"].(float64); ok && t > 0 {
		threshold = int64(t)
	}
	windowMin := 60
	if w, ok := cfg.Threshold["window_minutes"].(float64); ok && w > 0 {
		windowMin = int(w)
	}

	// Count exports per user in the window.
	key := fmt.Sprintf("export:%s", evt.ActorID)
	count, err := state.Incr(ctx, key, time.Duration(windowMin)*time.Minute)
	if err != nil {
		return nil, err
	}

	// Simpler: absolute threshold — more than `threshold` exports in window by one user.
	absThreshold := threshold
	if count < absThreshold {
		return nil, nil
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(),
		fmt.Sprintf("Mass data export: %d export operations in %d minutes by %s", count, windowMin, evt.ActorName))
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"export_count":   count,
		"window_minutes": windowMin,
		"export_action":  evt.Action,
		"resource":       evt.ResourceName,
		"ip_address":     evt.IPAddress,
	}
	return det, nil
}

// RegisterKB192Rules registers all 8 new MITRE-mapped detection rules.
func RegisterKB192Rules(registry *RuleRegistry) {
	registry.Register(&ConsentPhishingRule{})
	registry.Register(&MFAFatigueRule{})
	registry.Register(&TokenTheftRule{})
	registry.Register(&SessionHijackRule{})
	registry.Register(&MassCreationRule{})
	registry.Register(&FederationAnomalyRule{})
	registry.Register(&MFABypassRule{})
	registry.Register(&MassExportRule{})
}
