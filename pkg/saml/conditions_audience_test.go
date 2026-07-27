package saml

import (
	"testing"
	"time"
)

func TestValidateConditionsWithAudience_Success(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(55 * time.Minute).Format(time.RFC3339),
			AudienceRestriction: AudienceRestriction{
				Audience: "https://sp.example.com/saml/metadata",
			},
		},
	}
	err := a.ValidateConditionsWithAudience("https://sp.example.com/saml/metadata")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateConditionsWithAudience_Mismatch(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(55 * time.Minute).Format(time.RFC3339),
			AudienceRestriction: AudienceRestriction{
				Audience: "https://wrong-sp.example.com/saml/metadata",
			},
		},
	}
	err := a.ValidateConditionsWithAudience("https://sp.example.com/saml/metadata")
	if err == nil {
		t.Error("expected audience mismatch error")
	}
}

func TestValidateConditionsWithAudience_NoRestriction(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(55 * time.Minute).Format(time.RFC3339),
		},
	}
	err := a.ValidateConditionsWithAudience("https://sp.example.com/saml/metadata")
	if err == nil {
		t.Error("expected error for missing AudienceRestriction")
	}
}

func TestValidateConditionsWithAudience_EmptyAudience(t *testing.T) {
	// When expectedAudience is empty, skip audience check
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(55 * time.Minute).Format(time.RFC3339),
		},
	}
	err := a.ValidateConditionsWithAudience("")
	if err != nil {
		t.Errorf("expected no error with empty audience (skip check), got: %v", err)
	}
}

func TestValidateConditionsWithAudience_Expired(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
	}
	err := a.ValidateConditionsWithAudience("")
	if err == nil {
		t.Error("expected expiry error")
	}
}

func TestValidateConditionsWithAudience_NotYetValid(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		},
	}
	err := a.ValidateConditionsWithAudience("")
	if err == nil {
		t.Error("expected not-yet-valid error")
	}
}

func TestValidateConditionsWithAudience_SubjectConfirmation_Bearer(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(55 * time.Minute).Format(time.RFC3339),
		},
		Subject: Subject{
			SubjectConfirmation: SubjectConfirmation{
				Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
			},
		},
	}
	err := a.ValidateConditionsWithAudience("")
	if err != nil {
		t.Errorf("expected no error for bearer method, got: %v", err)
	}
}

func TestValidateConditionsWithAudience_SubjectConfirmation_InvalidMethod(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(55 * time.Minute).Format(time.RFC3339),
		},
		Subject: Subject{
			SubjectConfirmation: SubjectConfirmation{
				Method: "urn:oasis:names:tc:SAML:2.0:cm:holder-of-key",
			},
		},
	}
	err := a.ValidateConditionsWithAudience("")
	if err == nil {
		t.Error("expected error for non-bearer SubjectConfirmation method")
	}
}

func TestValidateConditionsWithAudience_SubjectConfirmation_Expired(t *testing.T) {
	a := &SAMLAssertion{
		Conditions: Conditions{
			NotBefore:    time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			NotOnOrAfter: time.Now().Add(55 * time.Minute).Format(time.RFC3339),
		},
		Subject: Subject{
			SubjectConfirmation: SubjectConfirmation{
				Method: "urn:oasis:names:tc:SAML:2.0:cm:bearer",
				SubjectConfirmationData: SubjectConfirmationData{
					NotOnOrAfter: time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
				},
			},
		},
	}
	err := a.ValidateConditionsWithAudience("")
	if err == nil {
		t.Error("expected error for expired SubjectConfirmation")
	}
}

func TestExtractAttributes_MultiValue(t *testing.T) {
	a := &SAMLAssertion{
		AttributeStatement: AttributeStatement{
			Attributes: []Attribute{
				{Name: "email", Values: []string{"user@example.com"}},
				{Name: "groups", Values: []string{"admin", "dev"}},
				{Name: "department", Values: []string{"engineering"}},
			},
		},
	}
	attrs := ExtractAttributes(a)

	if len(attrs) != 3 {
		t.Fatalf("expected 3 attributes, got %d", len(attrs))
	}
	if attrs["email"][0] != "user@example.com" {
		t.Errorf("email mismatch: %v", attrs["email"])
	}
	if len(attrs["groups"]) != 2 {
		t.Errorf("expected 2 groups, got %d", len(attrs["groups"]))
	}
}

func TestGetAttribute_FirstValue(t *testing.T) {
	a := &SAMLAssertion{
		AttributeStatement: AttributeStatement{
			Attributes: []Attribute{
				{Name: "email", Values: []string{"user@example.com", "user@backup.com"}},
			},
		},
	}
	// GetAttribute returns first value
	val := GetAttribute(a, "email")
	if val != "user@example.com" {
		t.Errorf("expected first email, got %s", val)
	}
	// Non-existent attribute
	val = GetAttribute(a, "nonexistent")
	if val != "" {
		t.Errorf("expected empty string for missing attribute, got %s", val)
	}
}

func TestInResponseTo_Value(t *testing.T) {
	a := &SAMLAssertion{
		Subject: Subject{
			SubjectConfirmation: SubjectConfirmation{
				SubjectConfirmationData: SubjectConfirmationData{
					InResponseTo: "request-12345",
				},
			},
		},
	}
	if got := a.InResponseTo(); got != "request-12345" {
		t.Errorf("InResponseTo = %q, want %q", got, "request-12345")
	}
}
