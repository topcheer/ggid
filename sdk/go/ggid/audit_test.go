package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAuditEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/audit/events" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("event_type") != "user.login" {
			t.Errorf("expected event_type filter")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]AuditEvent{
			{ID: "evt-1", EventType: "user.login", ActorID: "user-1"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	events, err := c.ListAuditEvents(context.Background(), "test-token", AuditEventFilter{EventType: "user.login"})
	if err != nil {
		t.Fatalf("ListAuditEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "evt-1" {
		t.Errorf("expected evt-1, got %s", events[0].ID)
	}
}

func TestGetComplianceReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "soc2" {
			t.Errorf("expected type=soc2")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ComplianceReport{
			Type: "soc2",
			Period: map[string]string{"start": "2025-01-01", "end": "2025-12-31"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	report, err := c.GetComplianceReport(context.Background(), "tok", "soc2", "", "")
	if err != nil {
		t.Fatalf("GetComplianceReport failed: %v", err)
	}
	if report.Type != "soc2" {
		t.Errorf("expected soc2, got %s", report.Type)
	}
}

func TestGetAlertRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Rules []AlertRule `json:"rules"`
		}{
			Rules: []AlertRule{
				{ID: "rule-1", Name: "Failed Logins", Threshold: 5, Enabled: true},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	rules, err := c.GetAlertRules(context.Background(), "tok")
	if err != nil {
		t.Fatalf("GetAlertRules failed: %v", err)
	}
	if len(rules) != 1 || rules[0].Name != "Failed Logins" {
		t.Fatalf("unexpected rules: %+v", rules)
	}
}

func TestUpsertAlertRule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	err := c.UpsertAlertRule(context.Background(), "tok", AlertRule{
		Name: "Test Rule", Condition: "failed_login", Threshold: 3,
	})
	if err != nil {
		t.Fatalf("UpsertAlertRule failed: %v", err)
	}
}

func TestRetentionPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(RetentionPolicy{MaxAgeDays: 90})
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	policy, err := c.GetRetentionPolicy(context.Background(), "tok")
	if err != nil {
		t.Fatalf("GetRetentionPolicy failed: %v", err)
	}
	if policy.MaxAgeDays != 90 {
		t.Errorf("expected 90, got %d", policy.MaxAgeDays)
	}

	err = c.UpdateRetentionPolicy(context.Background(), "tok", RetentionPolicy{MaxAgeDays: 180})
	if err != nil {
		t.Fatalf("UpdateRetentionPolicy failed: %v", err)
	}
}

func TestExportAuditEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "csv" {
			t.Errorf("expected format=csv")
		}
		w.Write([]byte("id,event_type\n1,login\n"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	data, err := c.ExportAuditEvents(context.Background(), "tok", "csv")
	if err != nil {
		t.Fatalf("ExportAuditEvents failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty export data")
	}
}

func TestVerifyAuditIntegrity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Valid   bool   `json:"valid"`
			Message string `json:"message"`
		}{Valid: true, Message: "chain intact"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	valid, err := c.VerifyAuditIntegrity(context.Background(), "tok")
	if err != nil {
		t.Fatalf("VerifyAuditIntegrity failed: %v", err)
	}
	if !valid {
		t.Error("expected valid chain")
	}
}

func TestListAccessRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]AccessRequest{
			{ID: "ar-1", Resource: "admin-panel", Action: "read", Status: "pending"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	reqs, err := c.ListAccessRequests(context.Background(), "tok", "pending")
	if err != nil {
		t.Fatalf("ListAccessRequests failed: %v", err)
	}
	if len(reqs) != 1 || reqs[0].Status != "pending" {
		t.Fatalf("unexpected: %+v", reqs)
	}
}

func TestSubmitAccessRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(AccessRequest{
			ID: "ar-2", Resource: "db-prod", Action: "write", Status: "pending",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.SubmitAccessRequest(context.Background(), "tok", AccessRequest{
		Resource: "db-prod", Action: "write",
	})
	if err != nil {
		t.Fatalf("SubmitAccessRequest failed: %v", err)
	}
	if result.ID != "ar-2" {
		t.Errorf("expected ar-2, got %s", result.ID)
	}
}

func TestApproveAccessRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/access-requests/ar-1/approve" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.ApproveAccessRequest(context.Background(), "tok", "ar-1"); err != nil {
		t.Fatalf("ApproveAccessRequest failed: %v", err)
	}
}

func TestDenyAccessRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/access-requests/ar-1/deny" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.DenyAccessRequest(context.Background(), "tok", "ar-1"); err != nil {
		t.Fatalf("DenyAccessRequest failed: %v", err)
	}
}

func TestBranding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(BrandingConfig{
				LogoURL: "https://example.com/logo.png", PrimaryColor: "#1a73e8",
			})
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	config, err := c.GetBranding(context.Background(), "tok", "tenant-1")
	if err != nil {
		t.Fatalf("GetBranding failed: %v", err)
	}
	if config.PrimaryColor != "#1a73e8" {
		t.Errorf("expected #1a73e8, got %s", config.PrimaryColor)
	}

	err = c.UpdateBranding(context.Background(), "tok", "tenant-1", BrandingConfig{
		LogoURL: "https://example.com/new-logo.png",
	})
	if err != nil {
		t.Fatalf("UpdateBranding failed: %v", err)
	}
}

func TestTestAlert(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.TestAlert(context.Background(), "tok"); err != nil {
		t.Fatalf("TestAlert failed: %v", err)
	}
}

func TestListAuditEventsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid token"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.ListAuditEvents(context.Background(), "bad-token", AuditEventFilter{})
	if err == nil {
		t.Error("expected error for 401 response")
	}
}
