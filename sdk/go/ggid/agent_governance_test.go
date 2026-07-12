package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetectAgentDrift(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing auth header")
		}
		json.NewEncoder(w).Encode(DriftReport{
			AgentID:        "agent-1",
			AgentName:      "CodeBot",
			DetectedScopes: []string{"repo:write", "repo:read"},
			DeclaredScopes: []string{"repo:read"},
			DriftType:      "scope_expansion",
			Severity:       "high",
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	report, err := c.DetectAgentDrift(context.Background(), "agent-1", "test-token")
	if err != nil {
		t.Fatalf("DetectAgentDrift: %v", err)
	}
	if report.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", report.AgentID)
	}
	if report.DriftType != "scope_expansion" {
		t.Errorf("expected scope_expansion, got %s", report.DriftType)
	}
	if len(report.DetectedScopes) != 2 {
		t.Errorf("expected 2 detected scopes, got %d", len(report.DetectedScopes))
	}
}

func TestScanShadowAgents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ShadowScanResult{
			TotalTokens:   50,
			TotalShadows:  3,
			UnknownAgents: []ShadowAgent{
				{AgentID: "unknown-1", TokenCount: 5, RiskLevel: "high"},
				{AgentID: "unknown-2", TokenCount: 2, RiskLevel: "medium"},
			},
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	result, err := c.ScanShadowAgents(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("ScanShadowAgents: %v", err)
	}
	if result.TotalShadows != 3 {
		t.Errorf("expected 3 shadows, got %d", result.TotalShadows)
	}
	if len(result.UnknownAgents) != 2 {
		t.Errorf("expected 2 unknown agents, got %d", len(result.UnknownAgents))
	}
}

func TestCreateAgentReview(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(AgentReview{
			ID:       "rev-1",
			AgentID:  "agent-1",
			Reviewer: "admin",
			Decision: "approve",
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	review, err := c.CreateAgentReview(context.Background(), &AgentReview{
		AgentID:        "agent-1",
		Reviewer:       "admin",
		ScopesReviewed: []string{"repo:read"},
		Decision:       "approve",
	}, "test-token")
	if err != nil {
		t.Fatalf("CreateAgentReview: %v", err)
	}
	if review.ID != "rev-1" {
		t.Errorf("expected rev-1, got %s", review.ID)
	}
}

func TestListAgentReviews(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]AgentReview{
			{ID: "rev-1", AgentID: "agent-1", Decision: "approve"},
			{ID: "rev-2", AgentID: "agent-2", Decision: "reject"},
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	reviews, err := c.ListAgentReviews(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("ListAgentReviews: %v", err)
	}
	if len(reviews) != 2 {
		t.Errorf("expected 2 reviews, got %d", len(reviews))
	}
}

func TestGetAgentReview(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AgentReview{
			ID: "rev-1", AgentID: "agent-1", Decision: "approve",
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	review, err := c.GetAgentReview(context.Background(), "rev-1", "test-token")
	if err != nil {
		t.Fatalf("GetAgentReview: %v", err)
	}
	if review.ID != "rev-1" {
		t.Errorf("expected rev-1, got %s", review.ID)
	}
}

func TestUpdateAgentReview(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(AgentReview{
			ID: "rev-1", Decision: "reject", Comment: "too broad",
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	review, err := c.UpdateAgentReview(context.Background(), "rev-1", &AgentReview{
		Decision: "reject", Comment: "too broad",
	}, "test-token")
	if err != nil {
		t.Fatalf("UpdateAgentReview: %v", err)
	}
	if review.Decision != "reject" {
		t.Errorf("expected reject, got %s", review.Decision)
	}
}

func TestListNHI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(NHIInventory{
			Total: 10, Active: 7, Orphaned: 3,
			Entries: []NHIEntry{
				{ID: "nhi-1", Name: "svc-account-1", Type: NHITypeServiceAccount, Status: "active"},
				{ID: "nhi-2", Name: "api-key-prod", Type: NHITypeAPIKey, Status: "orphaned"},
			},
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	inv, err := c.ListNHI(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("ListNHI: %v", err)
	}
	if inv.Total != 10 {
		t.Errorf("expected 10 total, got %d", inv.Total)
	}
	if inv.Orphaned != 3 {
		t.Errorf("expected 3 orphaned, got %d", inv.Orphaned)
	}
}

func TestDetectOrphans(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]NHIEntry{
			{ID: "nhi-2", Name: "api-key-old", Type: NHITypeAPIKey, Status: "orphaned"},
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	orphans, err := c.DetectOrphans(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("DetectOrphans: %v", err)
	}
	if len(orphans) != 1 {
		t.Errorf("expected 1 orphan, got %d", len(orphans))
	}
}

func TestDecommissionNHI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	err := c.DecommissionNHI(context.Background(), "nhi-2", "orphaned 90+ days", "test-token")
	if err != nil {
		t.Fatalf("DecommissionNHI: %v", err)
	}
}

func TestScheduleRotation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RotationSchedule{
			CredentialID:   "cred-1",
			CredentialType: "api-key",
			Policy:         RotationPolicy{IntervalDays: 90, AutoRotate: true, NotifyBeforeDays: 7},
			Status:         "scheduled",
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	sched, err := c.ScheduleRotation(context.Background(), "cred-1", &RotationPolicy{
		IntervalDays:     90,
		AutoRotate:       true,
		NotifyBeforeDays: 7,
	}, "test-token")
	if err != nil {
		t.Fatalf("ScheduleRotation: %v", err)
	}
	if sched.Status != "scheduled" {
		t.Errorf("expected scheduled, got %s", sched.Status)
	}
	if sched.Policy.IntervalDays != 90 {
		t.Errorf("expected 90 days, got %d", sched.Policy.IntervalDays)
	}
}

func TestCheckDueRotations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]RotationSchedule{
			{CredentialID: "cred-1", Status: "due"},
			{CredentialID: "cred-2", Status: "overdue"},
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	schedules, err := c.CheckDueRotations(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("CheckDueRotations: %v", err)
	}
	if len(schedules) != 2 {
		t.Errorf("expected 2 schedules, got %d", len(schedules))
	}
}

func TestExecuteRotation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(RotationSchedule{
			CredentialID: "cred-1",
			Status:       "rotated",
		})
	}))
	defer srv.Close()

	c := &Client{gatewayURL: srv.URL, httpClient: &http.Client{}}
	sched, err := c.ExecuteRotation(context.Background(), "cred-1", "test-token")
	if err != nil {
		t.Fatalf("ExecuteRotation: %v", err)
	}
	if sched.Status != "rotated" {
		t.Errorf("expected rotated, got %s", sched.Status)
	}
}