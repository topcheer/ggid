package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLoginAttempts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login-attempts/user-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(LoginAttemptInfo{
			FailedAttempts:  3,
			LockedUntil:     "2025-01-01T00:00:00Z",
			LastAttemptIP:   "192.168.1.1",
			LastAttemptTime: "2025-01-01T00:00:00Z",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	info, err := client.GetLoginAttempts(context.Background(), "token", "user-123")
	if err != nil {
		t.Fatalf("GetLoginAttempts failed: %v", err)
	}
	if info.FailedAttempts != 3 {
		t.Errorf("expected 3 attempts, got %d", info.FailedAttempts)
	}
}

func TestResetLoginAttempts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login-attempts/user-123/reset" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.ResetLoginAttempts(context.Background(), "token", "user-123")
	if err != nil {
		t.Fatalf("ResetLoginAttempts failed: %v", err)
	}
}

func TestPasswordHistoryCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/password-history-check" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_repeated":   true,
			"history_count": 5,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	isRepeated, err := client.PasswordHistoryCheck(context.Background(), "token", "user-123", "newpass")
	if err != nil {
		t.Fatalf("PasswordHistoryCheck failed: %v", err)
	}
	if !isRepeated {
		t.Error("expected is_repeated=true")
	}
}

func TestLinkAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/user-123/link" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.LinkAccount(context.Background(), "token", "user-123", LinkedAccount{
		Provider:   "google",
		ExternalID: "google-123",
	})
	if err != nil {
		t.Fatalf("LinkAccount failed: %v", err)
	}
}

func TestUnlinkAccount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/user-123/link/google" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.UnlinkAccount(context.Background(), "token", "user-123", "google")
	if err != nil {
		t.Fatalf("UnlinkAccount failed: %v", err)
	}
}

func TestListConsents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/consent/list" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("user_id") != "user-123" {
			t.Errorf("expected user_id query param")
		}
		json.NewEncoder(w).Encode([]Consent{
			{ID: "consent-1", ClientID: "client-1", Scopes: []string{"openid", "email"}},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	consents, err := client.ListConsents(context.Background(), "token", "user-123")
	if err != nil {
		t.Fatalf("ListConsents failed: %v", err)
	}
	if len(consents) != 1 {
		t.Fatalf("expected 1 consent, got %d", len(consents))
	}
	if consents[0].ClientID != "client-1" {
		t.Errorf("expected client_id 'client-1', got %q", consents[0].ClientID)
	}
}

func TestRevokeConsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/consent/consent-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.RevokeConsent(context.Background(), "token", "consent-123")
	if err != nil {
		t.Fatalf("RevokeConsent failed: %v", err)
	}
}

func TestEvaluateABAC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policies/abac/evaluate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(ABACEvalResult{
			Matched:      true,
			MatchedRules: []string{"rule-1", "rule-2"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	result, err := client.EvaluateABAC(context.Background(), "token", ABACEvalRequest{
		Attributes: map[string]string{"department": "engineering"},
		Conditions: []ABACCondition{{Field: "department", Operator: "eq", Value: "engineering"}},
	})
	if err != nil {
		t.Fatalf("EvaluateABAC failed: %v", err)
	}
	if !result.Matched {
		t.Error("expected matched=true")
	}
	if len(result.MatchedRules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(result.MatchedRules))
	}
}

func TestValidateDelegation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy/delegation/validate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":          true,
			"depth":          3,
			"cycle_detected": false,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	valid, err := client.ValidateDelegation(context.Background(), "token", []string{"a", "b", "c"}, 5)
	if err != nil {
		t.Fatalf("ValidateDelegation failed: %v", err)
	}
	if !valid {
		t.Error("expected valid=true")
	}
}

func TestGetSIEMHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/audit/siem/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(SIEMHealth{
			PendingEvents: 5,
			ErrorCount:    0,
			DestURL:       "https://siem.example.com/***",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	health, err := client.GetSIEMHealth(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetSIEMHealth failed: %v", err)
	}
	if health.PendingEvents != 5 {
		t.Errorf("expected 5 pending, got %d", health.PendingEvents)
	}
}

func TestCreateAlertWebhook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/audit/alert-webhooks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.CreateAlertWebhook(context.Background(), "token", AlertWebhook{
		URL:    "https://hooks.example.com/alert",
		Events: []string{"user.locked", "auth.failed"},
	})
	if err != nil {
		t.Fatalf("CreateAlertWebhook failed: %v", err)
	}
}

func TestListComplianceSchedules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/audit/compliance-schedules" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]ComplianceSchedule{
			{ID: "sched-1", ReportType: "soc2", Frequency: "weekly"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	schedules, err := client.ListComplianceSchedules(context.Background(), "token")
	if err != nil {
		t.Fatalf("ListComplianceSchedules failed: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}
	if schedules[0].ReportType != "soc2" {
		t.Errorf("expected soc2, got %q", schedules[0].ReportType)
	}
}

func TestValidateUserImport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/import/validate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(ImportValidationResult{
			ValidCount:   8,
			InvalidCount: 2,
			Errors: []ImportValidationError{
				{Row: 3, Field: "email", Error: "invalid format"},
				{Row: 7, Field: "username", Error: "already exists"},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	result, err := client.ValidateUserImport(context.Background(), "token", []map[string]string{
		{"username": "user1", "email": "user1@test.com"},
	})
	if err != nil {
		t.Fatalf("ValidateUserImport failed: %v", err)
	}
	if result.ValidCount != 8 {
		t.Errorf("expected 8 valid, got %d", result.ValidCount)
	}
	if result.InvalidCount != 2 {
		t.Errorf("expected 2 invalid, got %d", result.InvalidCount)
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors))
	}
}
