package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// --- ITDR Dashboard Alias Route Tests ---

func TestITDRThreatHeatmap_DashboardAlias(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/threat-heatmap", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if _, ok := resp["zones"]; !ok {
		t.Error("expected 'zones' key in response")
	}
	if _, ok := resp["total_threats"]; !ok {
		t.Error("expected 'total_threats' key in response")
	}
	if _, ok := resp["by_severity"]; !ok {
		t.Error("expected 'by_severity' key in response")
	}
}

func TestITDRThreatHeatmap_WithIncidents(t *testing.T) {
	srv := newTestServer(nil, nil)
	// Seed a critical incident
	itdrIncidentsMu.Lock()
	itdrIncidents["inc-1"] = &IncidentListEntry{
		ID:       "inc-1",
		Title:    "Test Critical Incident",
		Severity: "critical",
		Status:   "open",
	}
	itdrIncidentsMu.Unlock()
	defer func() {
		itdrIncidentsMu.Lock()
		delete(itdrIncidents, "inc-1")
		itdrIncidentsMu.Unlock()
	}()

	w := doRequest(srv, "GET", "/api/v1/audit/threat-heatmap", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	// total_threats is float64 in JSON
	if int(resp["total_threats"].(float64)) != 1 {
		t.Errorf("expected total_threats=1, got %v", resp["total_threats"])
	}
}

func TestITDRKillChainSummary_DashboardAlias(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/kill-chain", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	stages, ok := resp["stages"].([]any)
	if !ok {
		t.Fatal("expected 'stages' array in response")
	}
	if len(stages) != 5 {
		t.Errorf("expected 5 kill chain stages, got %d", len(stages))
	}
	if _, ok := resp["total_attacks"]; !ok {
		t.Error("expected 'total_attacks' key in response")
	}
}

func TestITDRTimelineFeed_DashboardAlias(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/incident-timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if _, ok := resp["events"]; !ok {
		t.Error("expected 'events' key in response")
	}
	if _, ok := resp["total"]; !ok {
		t.Error("expected 'total' key in response")
	}
}

func TestITDRTimelineFeed_WithIncidentTimeline(t *testing.T) {
	srv := newTestServer(nil, nil)
	// Seed an incident with timeline entries
	itdrIncidentsMu.Lock()
	itdrIncidents["inc-tl"] = &IncidentListEntry{
		ID:       "inc-tl",
		Title:    "Test Incident With Timeline",
		Severity: "high",
		Status:   "investigating",
		Timeline: []TimelineEntry{
			{Timestamp: time.Now().UTC(), Event: "reconnaissance detected", Source: "anomaly_engine"},
			{Timestamp: time.Now().UTC(), Event: "credential_access attempt", Source: "correlation_engine"},
		},
	}
	itdrIncidentsMu.Unlock()
	defer func() {
		itdrIncidentsMu.Lock()
		delete(itdrIncidents, "inc-tl")
		itdrIncidentsMu.Unlock()
	}()

	w := doRequest(srv, "GET", "/api/v1/audit/incident-timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	events := resp["events"].([]any)
	if len(events) != 2 {
		t.Errorf("expected 2 timeline events, got %d", len(events))
	}
}

func TestITDRIncidents_ReturnsBothKeys(t *testing.T) {
	srv := newTestServer(nil, nil)
	// Seed an incident
	itdrIncidentsMu.Lock()
	itdrIncidents["inc-keys"] = &IncidentListEntry{
		ID:       "inc-keys",
		Title:    "Test",
		Severity: "medium",
		Status:   "open",
	}
	itdrIncidentsMu.Unlock()
	defer func() {
		itdrIncidentsMu.Lock()
		delete(itdrIncidents, "inc-keys")
		itdrIncidentsMu.Unlock()
	}()

	w := doRequest(srv, "GET", "/api/v1/audit/itdr/incidents", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	// Both "incidents" and "itdrIncidents" keys should be present
	if _, ok := resp["incidents"]; !ok {
		t.Error("expected 'incidents' key in response")
	}
	if _, ok := resp["itdrIncidents"]; !ok {
		t.Error("expected 'itdrIncidents' key in response (legacy compat)")
	}
}

func TestSecurityOverviewAlias(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/security/overview", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Should return same data as /security/dashboard
	w2 := doRequest(srv, "GET", "/api/v1/security/dashboard", "")
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for dashboard, got %d", w2.Code)
	}
}
