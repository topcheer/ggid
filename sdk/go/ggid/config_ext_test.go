package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserLifecycleConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/identity/user-lifecycle/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(UserLifecycleConfig{
				AutoDeactivateAfterDays: 90,
				NotificationBefore:      7,
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetUserLifecycleConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetUserLifecycleConfig: %v", err)
	}
	if cfg.AutoDeactivateAfterDays != 90 {
		t.Errorf("expected 90, got %d", cfg.AutoDeactivateAfterDays)
	}

	err = c.UpdateUserLifecycleConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateUserLifecycleConfig: %v", err)
	}
}

func TestABACConditionConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy/abac/condition-config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(ABACConditionConfig{
				AttributeSources:   []string{"ldap", "hr_db"},
				EvaluationCacheTTL: 300,
				DefaultDeny:        true,
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetABACConditionConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetABACConditionConfig: %v", err)
	}
	if len(cfg.AttributeSources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(cfg.AttributeSources))
	}

	err = c.UpdateABACConditionConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateABACConditionConfig: %v", err)
	}
}

func TestSCIMProvisioningConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/identity/scim/provisioning-config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(SCIMProvisioningConfig{
				Endpoint:             "https://scim.example.com/v2",
				SyncDirection:        "bidirectional",
				DeprovisionOnDisable: true,
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetSCIMProvisioningConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetSCIMProvisioningConfig: %v", err)
	}
	if cfg.Endpoint != "https://scim.example.com/v2" {
		t.Errorf("unexpected endpoint: %s", cfg.Endpoint)
	}

	err = c.UpdateSCIMProvisioningConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateSCIMProvisioningConfig: %v", err)
	}
}

func TestAuditExportConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/audit/export/schedule-config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(AuditExportConfig{
				MaxConcurrent: 3,
				Jobs: []ExportJob{
					{Name: "daily-export", Cron: "0 2 * * *", Format: "json"},
				},
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetAuditExportConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetAuditExportConfig: %v", err)
	}
	if len(cfg.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(cfg.Jobs))
	}

	err = c.UpdateAuditExportConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateAuditExportConfig: %v", err)
	}
}

func TestTokenRotationConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/token-rotation/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(TokenRotationConfig{
				PerClient: []ClientTokenRotation{
					{ClientID: "web-app", RotationInterval: 3600, AutoRotate: true},
				},
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetTokenRotationConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetTokenRotationConfig: %v", err)
	}
	if len(cfg.PerClient) != 1 {
		t.Errorf("expected 1 client, got %d", len(cfg.PerClient))
	}

	err = c.UpdateTokenRotationConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateTokenRotationConfig: %v", err)
	}
}

func TestRiskScoringConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/identity/risk-scoring/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(RiskScoringConfig{
				RiskFactors:      map[string]float64{"login_location": 0.3, "failed_attempts": 0.4},
				AdaptiveLearning: true,
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetRiskScoringConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetRiskScoringConfig: %v", err)
	}
	if !cfg.AdaptiveLearning {
		t.Error("expected adaptive learning to be true")
	}

	err = c.UpdateRiskScoringConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateRiskScoringConfig: %v", err)
	}
}

func TestSODConflictConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy/sod/conflict-detection-config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(SODConflictConfig{
				SensitivityLevels: map[string]string{"payments": "critical", "hr": "high"},
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetSODConflictConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetSODConflictConfig: %v", err)
	}
	if cfg.SensitivityLevels["payments"] != "critical" {
		t.Error("expected payments=critical")
	}

	err = c.UpdateSODConflictConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateSODConflictConfig: %v", err)
	}
}

func TestSIEMForwarderConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/audit/siem/forwarder-config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode(SIEMForwarderConfig{
				Destinations: []SIEMDestination{
					{SIEMType: "splunk", Protocol: "https", Host: "siem.internal:8088", Format: "cef"},
				},
				HealthCheckInterval: 60,
			})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, WithTenantID("test-tenant"))

	cfg, err := c.GetSIEMForwarderConfig(context.Background(), "token")
	if err != nil {
		t.Fatalf("GetSIEMForwarderConfig: %v", err)
	}
	if len(cfg.Destinations) != 1 || cfg.Destinations[0].SIEMType != "splunk" {
		t.Error("expected 1 splunk destination")
	}

	err = c.UpdateSIEMForwarderConfig(context.Background(), "token", cfg)
	if err != nil {
		t.Fatalf("UpdateSIEMForwarderConfig: %v", err)
	}
}
