package server

import (
	"testing"
)

func TestRAR_ValidateUserProfile_Valid(t *testing.T) {
	r := NewRARRegistry()
	details := []AuthorizationDetail{
		{Type: "user_profile", Actions: []string{"read"}, Fields: []string{"email", "phone"}},
	}
	if err := r.ValidateDetails(details, "client-1"); err != nil {
		t.Fatalf("valid user_profile should pass: %v", err)
	}
}

func TestRAR_ValidateUserProfile_InvalidAction(t *testing.T) {
	r := NewRARRegistry()
	details := []AuthorizationDetail{
		{Type: "user_profile", Actions: []string{"delete"}},
	}
	if err := r.ValidateDetails(details, "client-1"); err == nil {
		t.Fatal("delete action should be rejected for user_profile")
	}
}

func TestRAR_ValidateUnknownType(t *testing.T) {
	r := NewRARRegistry()
	details := []AuthorizationDetail{
		{Type: "unknown_type", Actions: []string{"read"}},
	}
	if err := r.ValidateDetails(details, "client-1"); err == nil {
		t.Fatal("unknown type should be rejected")
	}
}

func TestRAR_RenderConsentLines(t *testing.T) {
	r := NewRARRegistry()
	details := []AuthorizationDetail{
		{Type: "user_profile", Actions: []string{"read"}, Fields: []string{"email"}},
		{Type: "audit_events", Actions: []string{"read", "export"}},
	}
	lines := r.RenderConsentLines(details)
	if len(lines) != 2 {
		t.Fatalf("expected 2 consent lines, got %d", len(lines))
	}
	if lines[0].Title != "Read Profile" {
		t.Errorf("expected 'Read Profile', got '%s'", lines[0].Title)
	}
}

func TestRAR_PaymentInitiation(t *testing.T) {
	r := NewRARRegistry()
	details := []AuthorizationDetail{
		{
			Type:    "payment_initiation",
			Actions: []string{"initiate"},
			Constraints: map[string]any{
				"instructedAmount": map[string]any{"currency": "EUR", "amount": "100.00"},
			},
		},
	}
	if err := r.ValidateDetails(details, "client-1"); err != nil {
		t.Fatalf("payment_initiation should validate: %v", err)
	}
	lines := r.RenderConsentLines(details)
	if lines[0].Title != "Payment" {
		t.Errorf("expected 'Payment', got '%s'", lines[0].Title)
	}
}

func TestRAR_AllSixTypes(t *testing.T) {
	r := NewRARRegistry()
	types := []string{"user_profile", "user_roles", "audit_events", "app_access", "payment_initiation", "vc_issue"}
	for _, typeName := range types {
		_, ok := r.Get(typeName)
		if !ok {
			t.Errorf("type '%s' should be registered", typeName)
		}
	}
}
