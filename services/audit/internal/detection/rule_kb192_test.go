package detection

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// fakeState implements StateStore for testing.
type fakeState struct {
	counters map[string]int64
	events   map[string][]string // key → []member
}

func newFakeState() *fakeState {
	return &fakeState{
		counters: make(map[string]int64),
		events:   make(map[string][]string),
	}
}

func (s *fakeState) AddEvent(_ context.Context, key string, _ int64, member string, _ time.Duration) error {
	s.events[key] = append(s.events[key], member)
	return nil
}

func (s *fakeState) EventsSince(_ context.Context, key string, _ int64) ([]string, error) {
	return s.events[key], nil
}

func (s *fakeState) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	s.counters[key]++
	return s.counters[key], nil
}

func makeEvent(action string, result domain.EventResult) *domain.AuditEvent {
	actorID := uuid.New()
	return &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		ActorID:   &actorID,
		ActorName: "test@example.com",
		Action:    action,
		Result:    result,
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		CreatedAt: time.Now(),
		Metadata:  map[string]any{},
	}
}

// --- 1. Consent Phishing (T1098) ---

func TestConsentPhishing_RiskyScope(t *testing.T) {
	rule := &ConsentPhishingRule{}
	state := newFakeState()
	evt := makeEvent("oauth.consent.grant", "success")
	evt.Metadata["scopes"] = "admin files.readwrite offline_access"
	evt.Metadata["client_id"] = "app-123"

	det, err := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if det == nil {
		t.Fatal("expected detection for risky scopes")
	}
	if det.RuleID != "consent_phishing" {
		t.Fatalf("got %s", det.RuleID)
	}
}

func TestConsentPhishing_SafeScope(t *testing.T) {
	rule := &ConsentPhishingRule{}
	state := newFakeState()
	evt := makeEvent("oauth.consent.grant", "success")
	evt.Metadata["scopes"] = "openid profile"

	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("safe scopes should not trigger")
	}
}

// --- 2. MFA Fatigue (T1621) ---

func TestMFAFatigue_BelowThreshold(t *testing.T) {
	rule := &MFAFatigueRule{}
	state := newFakeState()

	for i := 0; i < 4; i++ {
		evt := makeEvent("mfa.push", "failure")
		det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
		if det != nil {
			t.Fatalf("should not trigger at count %d", i+1)
		}
	}
}

func TestMFAFatigue_AtThreshold(t *testing.T) {
	rule := &MFAFatigueRule{}
	state := newFakeState()
	var lastDet *domain.Detection

	for i := 0; i < 6; i++ {
		evt := makeEvent("mfa.push", "failure")
		lastDet, _ = rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	}
	if lastDet == nil {
		t.Fatal("expected detection at 6 pushes (threshold 5)")
	}
}

// --- 3. Token Theft (T1528) ---

func TestTokenTheft_DifferentIPAndDevice(t *testing.T) {
	rule := &TokenTheftRule{}
	state := newFakeState()

	// First use from IP1 + UA1.
	evt1 := makeEvent("token.exchange", "success")
	evt1.IPAddress = "10.0.0.1"
	evt1.UserAgent = "Chrome/120"
	rule.Evaluate(context.Background(), evt1, state, domain.RuleConfig{Enabled: true})

	// Second use from different IP + different UA.
	evt2 := makeEvent("token.exchange", "success")
	evt2.ActorID = evt1.ActorID
	evt2.IPAddress = "10.0.0.2"
	evt2.UserAgent = "Firefox/120"
	det, _ := rule.Evaluate(context.Background(), evt2, state, domain.RuleConfig{Enabled: true})
	if det == nil {
		t.Fatal("expected token theft detection for different IP + device")
	}
}

func TestTokenTheft_SameFingerprint(t *testing.T) {
	rule := &TokenTheftRule{}
	state := newFakeState()

	evt := makeEvent("api.call", "success")
	rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})

	// Same fingerprint — no theft.
	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("same IP + UA should not trigger")
	}
}

// --- 4. Session Hijack (T1539) ---

func TestSessionHijack_IPChange(t *testing.T) {
	rule := &SessionHijackRule{}
	state := newFakeState()

	evt1 := makeEvent("session.validate", "success")
	evt1.Metadata["session_id"] = "sess-123"
	evt1.IPAddress = "10.0.0.1"
	rule.Evaluate(context.Background(), evt1, state, domain.RuleConfig{Enabled: true})

	// Same session, different IP.
	evt2 := makeEvent("session.validate", "success")
	evt2.Metadata["session_id"] = "sess-123"
	evt2.IPAddress = "10.0.0.99"
	det, _ := rule.Evaluate(context.Background(), evt2, state, domain.RuleConfig{Enabled: true})
	if det == nil {
		t.Fatal("expected session hijack detection for IP change")
	}
}

func TestSessionHijack_NoSessionID(t *testing.T) {
	rule := &SessionHijackRule{}
	state := newFakeState()
	evt := makeEvent("api.call", "success")
	// No session_id in metadata.
	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not trigger without session_id")
	}
}

// --- 5. Mass Creation (T1136) ---

func TestMassCreation_BelowThreshold(t *testing.T) {
	rule := &MassCreationRule{}
	state := newFakeState()

	for i := 0; i < 9; i++ {
		evt := makeEvent("user.create", "success")
		det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
		if det != nil {
			t.Fatal("should not trigger below 10")
		}
	}
}

func TestMassCreation_AtThreshold(t *testing.T) {
	rule := &MassCreationRule{}
	state := newFakeState()
	var det *domain.Detection

	for i := 0; i < 11; i++ {
		evt := makeEvent("user.create", "success")
		det, _ = rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	}
	if det == nil {
		t.Fatal("expected detection at 11 creations")
	}
}

func TestMassCreation_AdminExempt(t *testing.T) {
	rule := &MassCreationRule{}
	state := newFakeState()
	evt := makeEvent("user.create", "success")
	evt.Metadata["actor_role"] = "admin"

	for i := 0; i < 20; i++ {
		det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
		if det != nil {
			t.Fatal("admin should be exempt from mass creation")
		}
	}
}

// --- 6. Federation Anomaly (T1606) ---

func TestFederationAnomaly_NewIdP(t *testing.T) {
	rule := &FederationAnomalyRule{}
	state := newFakeState()
	evt := makeEvent("saml.acs", "success")
	evt.Metadata["idp_entity_id"] = "https://new-idp.example.com"
	evt.Metadata["saml_issuer"] = "new-idp"

	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det == nil {
		t.Fatal("expected detection for new IdP")
	}
}

func TestFederationAnomaly_KnownIdP(t *testing.T) {
	rule := &FederationAnomalyRule{}
	state := newFakeState()
	idp := "https://known-idp.example.com"

	// First call registers it.
	evt1 := makeEvent("saml.acs", "success")
	evt1.Metadata["idp_entity_id"] = idp
	rule.Evaluate(context.Background(), evt1, state, domain.RuleConfig{Enabled: true})

	// Second call should not trigger (known IdP).
	evt2 := makeEvent("saml.acs", "success")
	evt2.Metadata["idp_entity_id"] = idp
	det, _ := rule.Evaluate(context.Background(), evt2, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("known IdP should not trigger")
	}
}

// --- 7. MFA Bypass (T1098.001) ---

func TestMFABypass_RequiredNotPerformed(t *testing.T) {
	rule := &MFABypassRule{}
	state := newFakeState()
	evt := makeEvent("user.login", "success")
	evt.Metadata["mfa_required"] = true
	evt.Metadata["mfa_performed"] = false

	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det == nil {
		t.Fatal("expected detection for MFA bypass")
	}
}

func TestMFABypass_RequiredAndPerformed(t *testing.T) {
	rule := &MFABypassRule{}
	state := newFakeState()
	evt := makeEvent("user.login", "success")
	evt.Metadata["mfa_required"] = true
	evt.Metadata["mfa_performed"] = true
	evt.Metadata["mfa_method"] = "totp"

	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not trigger when MFA was performed")
	}
}

func TestMFABypass_NotRequired(t *testing.T) {
	rule := &MFABypassRule{}
	state := newFakeState()
	evt := makeEvent("user.login", "success")
	evt.Metadata["mfa_required"] = false

	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not trigger when MFA not required")
	}
}

// --- 8. Mass Export (T1005) ---

func TestMassExport_BelowThreshold(t *testing.T) {
	rule := &MassExportRule{}
	state := newFakeState()
	for i := 0; i < 4; i++ {
		evt := makeEvent("data.export", "success")
		det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
		if det != nil {
			t.Fatal("should not trigger below threshold")
		}
	}
}

func TestMassExport_AtThreshold(t *testing.T) {
	rule := &MassExportRule{}
	state := newFakeState()
	var det *domain.Detection
	for i := 0; i < 6; i++ {
		evt := makeEvent("audit.export", "success")
		det, _ = rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	}
	if det == nil {
		t.Fatal("expected detection at 6 exports (threshold 5)")
	}
}
