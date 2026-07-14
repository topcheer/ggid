package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Gap Regression: Authorize Flow Stats (#session-verified)
// Validates: GET /api/v1/oauth/stats/authorize-flow returns structured stats
// with consent rate, abandonment steps, top clients, and PKCE adoption.

func TestGapRegression_AuthorizeFlowStats_GetOnly(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/oauth/stats/authorize-flow", nil)
	w := httptest.NewRecorder()
	handleAuthorizeFlowStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestGapRegression_AuthorizeFlowStats_Structure(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/authorize-flow", nil)
	w := httptest.NewRecorder()
	handleAuthorizeFlowStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp AuthorizeFlowStats
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TotalAttempts <= 0 {
		t.Errorf("expected total_attempts > 0, got %d", resp.TotalAttempts)
	}
	if resp.ConsentRate < 0 || resp.ConsentRate > 1 {
		t.Errorf("consent_rate should be 0-1, got %f", resp.ConsentRate)
	}
}

func TestGapRegression_AuthorizeFlowStats_AbandonmentSteps(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/authorize-flow", nil)
	w := httptest.NewRecorder()
	handleAuthorizeFlowStats(w, req)
	var resp AuthorizeFlowStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.AbandonmentAtStep) == 0 {
		t.Fatal("expected abandonment_at_step entries")
	}
	for _, step := range resp.AbandonmentAtStep {
		if step.Step == "" {
			t.Error("abandonment step missing name")
		}
		if step.Count <= 0 {
			t.Errorf("abandonment step %s count should be > 0", step.Step)
		}
	}
}

func TestGapRegression_AuthorizeFlowStats_TopClients(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/authorize-flow", nil)
	w := httptest.NewRecorder()
	handleAuthorizeFlowStats(w, req)
	var resp AuthorizeFlowStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.TopClients) == 0 {
		t.Fatal("expected top_clients entries")
	}
	for _, c := range resp.TopClients {
		if c.ClientID == "" || c.ClientName == "" {
			t.Error("top client missing id or name")
		}
		if c.Attempts <= 0 {
			t.Errorf("client %s attempts should be > 0", c.ClientID)
		}
	}
}

func TestGapRegression_AuthorizeFlowStats_PKCEAdoption(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/authorize-flow", nil)
	w := httptest.NewRecorder()
	handleAuthorizeFlowStats(w, req)
	var resp AuthorizeFlowStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.PKCEAdoptionPct < 0 || resp.PKCEAdoptionPct > 1 {
		t.Errorf("pkce_adoption should be 0-1, got %f", resp.PKCEAdoptionPct)
	}
}

func TestGapRegression_AuthorizeFlowStats_GeneratedAt(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/authorize-flow", nil)
	w := httptest.NewRecorder()
	handleAuthorizeFlowStats(w, req)
	var resp AuthorizeFlowStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.GeneratedAt == "" {
		t.Error("expected generated_at timestamp")
	}
}

// Gap Regression: Token Binding Stats (#session-verified)
// Validates: GET /api/v1/oauth/stats/token-binding returns bound/unbound
// token counts, binding method distribution, per-client compliance.

func TestGapRegression_TokenBindingStats_GetOnly(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/oauth/stats/token-binding", nil)
	w := httptest.NewRecorder()
	handleTokenBindingStats(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestGapRegression_TokenBindingStats_Structure(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/token-binding", nil)
	w := httptest.NewRecorder()
	handleTokenBindingStats(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp TokenBindingStats
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.TotalTokens <= 0 {
		t.Errorf("expected total_tokens > 0, got %d", resp.TotalTokens)
	}
}

func TestGapRegression_TokenBindingStats_BoundUnboundMath(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/token-binding", nil)
	w := httptest.NewRecorder()
	handleTokenBindingStats(w, req)
	var resp TokenBindingStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.BoundTokens+resp.UnboundTokens != resp.TotalTokens {
		t.Errorf("bound(%d)+unbound(%d) != total(%d)", resp.BoundTokens, resp.UnboundTokens, resp.TotalTokens)
	}
}

func TestGapRegression_TokenBindingStats_BindingMethods(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/token-binding", nil)
	w := httptest.NewRecorder()
	handleTokenBindingStats(w, req)
	var resp TokenBindingStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.BindingMethods) == 0 {
		t.Fatal("expected binding_methods entries")
	}
	for _, m := range resp.BindingMethods {
		if m.Method == "" {
			t.Error("binding method missing name")
		}
		if m.Count <= 0 {
			t.Errorf("binding method %s count should be > 0", m.Method)
		}
	}
}

func TestGapRegression_TokenBindingStats_ByClient(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/token-binding", nil)
	w := httptest.NewRecorder()
	handleTokenBindingStats(w, req)
	var resp TokenBindingStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.ByClient) == 0 {
		t.Fatal("expected by_client entries")
	}
	for _, c := range resp.ByClient {
		if c.ClientID == "" {
			t.Error("client stat missing id")
		}
	}
}

func TestGapRegression_TokenBindingStats_CompliancePct(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/oauth/stats/token-binding", nil)
	w := httptest.NewRecorder()
	handleTokenBindingStats(w, req)
	var resp TokenBindingStats
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.CompliancePct < 0 || resp.CompliancePct > 100 {
		t.Errorf("compliance_pct should be 0-100, got %f", resp.CompliancePct)
	}
}

// Gap Regression: Client Branding Persistence (#branding-verified)
// Validates: handleClientBranding uses brandingAdapterVar (PG-first, mem fallback).

func TestGapRegression_ClientBranding_UsesAdapter(t *testing.T) {
	brandingAdapterVar = newBrandingAdapter(nil)
	clientID := "gap-test-client"

	body := `{"logo_url":"https://example.com/logo.png","primary_color":"#ff0000","background_url":"","custom_css":""}`
	req := httptest.NewRequest("PUT", "/api/v1/oauth/clients/"+clientID+"/branding", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handleClientBranding(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT branding expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/v1/oauth/clients/"+clientID+"/branding", nil)
	w = httptest.NewRecorder()
	handleClientBranding(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET branding expected 200, got %d", w.Code)
	}
	var resp struct {
		ClientID string          `json:"client_id"`
		Branding *ClientBranding `json:"branding"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ClientID != clientID {
		t.Errorf("expected client_id %s, got %s", clientID, resp.ClientID)
	}
	if resp.Branding == nil || resp.Branding.LogoURL != "https://example.com/logo.png" {
		t.Errorf("branding adapter read did not return persisted value: %+v", resp.Branding)
	}
}