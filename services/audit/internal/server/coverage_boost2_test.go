package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/ggid/ggid/services/audit/internal/service"
	"github.com/google/uuid"
)

// --- Compliance Report Handler Coverage ---

func TestHandleComplianceReport_SOC2(t *testing.T) {
	srv := newTestServer(nil, nil)
	body := `{"tenant_id":"` + testTenantID.String() + `","format":"soc2"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["format"] != "soc2" {
		t.Fatalf("expected soc2, got %v", resp["format"])
	}
	ctrls, ok := resp["compliance_controls"].(map[string]any)
	if !ok {
		t.Fatal("expected compliance_controls map")
	}
	if _, ok := ctrls["CC6_1_logical_access"]; !ok {
		t.Fatal("expected CC6_1_logical_access control")
	}
}

func TestHandleComplianceReport_GDPR(t *testing.T) {
	srv := newTestServer(nil, nil)
	body := `{"tenant_id":"` + testTenantID.String() + `","format":"gdpr"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["format"] != "gdpr" {
		t.Fatalf("expected gdpr, got %v", resp["format"])
	}
	ctrls, ok := resp["compliance_controls"].(map[string]any)
	if !ok {
		t.Fatal("expected compliance_controls map")
	}
	if _, ok := ctrls["art_32_security"]; !ok {
		t.Fatal("expected art_32_security control")
	}
}

func TestHandleComplianceReport_DefaultFormat(t *testing.T) {
	srv := newTestServer(nil, nil)
	body := `{"tenant_id":"` + testTenantID.String() + `"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["format"] != "soc2" {
		t.Fatalf("expected default soc2, got %v", resp["format"])
	}
}

func TestHandleComplianceReport_InvalidFormat(t *testing.T) {
	srv := newTestServer(nil, nil)
	body := `{"tenant_id":"` + testTenantID.String() + `","format":"iso27001"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleComplianceReport_MissingTenantID(t *testing.T) {
	srv := newTestServer(nil, nil)
	body := `{"format":"soc2"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleComplianceReport_InvalidTenantID(t *testing.T) {
	srv := newTestServer(nil, nil)
	body := `{"tenant_id":"not-a-uuid","format":"soc2"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleComplianceReport_InvalidJSON(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "POST", "/api/v1/audit/reports", "bad json")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleComplianceReport_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/reports", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleComplianceReport_WithTimeRange(t *testing.T) {
	srv := newTestServer(nil, nil)
	start := time.Now().UTC().Add(-72 * time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Format(time.RFC3339)
	body := `{"tenant_id":"` + testTenantID.String() + `","format":"soc2","start_time":"` + start + `","end_time":"` + end + `"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleComplianceReport_WithRichEvents(t *testing.T) {
	actorID := uuid.New()
	events := []*domain.AuditEvent{
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorUser, ActorID: &actorID,
			ActorName: "admin", Action: "user.login", Result: domain.ResultSuccess,
			IPAddress: "10.0.0.1", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorUser, ActorID: &actorID,
			ActorName: "admin", Action: "user.login", Result: domain.ResultFailure,
			IPAddress: "10.0.0.2", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorUser, ActorID: &actorID,
			ActorName: "admin", Action: "user.mfa.challenge", Result: domain.ResultSuccess,
			IPAddress: "10.0.0.1", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorUser, ActorID: &actorID,
			ActorName: "admin", Action: "admin.update_policy", Result: domain.ResultSuccess,
			IPAddress: "10.0.0.1", CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorUser, ActorID: &actorID,
			ActorName: "admin", Action: "data.export", Result: domain.ResultSuccess,
			IPAddress: "10.0.0.1", CreatedAt: time.Now().UTC()},
	}
	srv := newTestServer(events, nil)
	body := `{"tenant_id":"` + testTenantID.String() + `","format":"soc2"}`
	w := doRequest(srv, "POST", "/api/v1/audit/reports", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	summary, ok := resp["summary"].(map[string]any)
	if !ok {
		t.Fatal("expected summary")
	}
	if int(summary["total_auth_events"].(float64)) < 1 {
		t.Fatalf("expected auth events, got %v", summary["total_auth_events"])
	}
	if int(summary["mfa_challenges"].(float64)) < 1 {
		t.Fatalf("expected mfa events, got %v", summary["mfa_challenges"])
	}
}

// --- Alert Config Handler Coverage ---

func TestHandleAlertConfig_Get(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/alerts/config", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleAlertConfig_Post(t *testing.T) {
	alertCfg.mu.Lock()
	alertCfg.enabled = false
	alertCfg.webhookURL = ""
	alertCfg.emailTo = ""
	alertCfg.minSeverity = "warning"
	alertCfg.mu.Unlock()

	srv := newTestServer(nil, nil)
	body := `{"enabled":true,"webhook_url":"https://hooks.example.com","email_to":"admin@example.com","min_severity":"critical"}`
	w := doRequest(srv, "POST", "/api/v1/audit/alerts/config", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", resp["enabled"])
	}
	if resp["webhook_url"] != "https://hooks.example.com" {
		t.Fatalf("expected webhook URL, got %v", resp["webhook_url"])
	}
}

func TestHandleAlertConfig_Post_InvalidJSON(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "POST", "/api/v1/audit/alerts/config", "bad json")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAlertConfig_Post_InvalidSeverity(t *testing.T) {
	srv := newTestServer(nil, nil)
	body := `{"min_severity":"bogus"}`
	w := doRequest(srv, "POST", "/api/v1/audit/alerts/config", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (invalid severity ignored), got %d", w.Code)
	}
}

func TestHandleAlertConfig_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "DELETE", "/api/v1/audit/alerts/config", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// --- Alert Test Handler Coverage ---

func TestHandleAlertTest_Post(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "POST", "/api/v1/audit/alerts/test", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "test_alert_dispatched" {
		t.Fatalf("expected test_alert_dispatched, got %v", resp["status"])
	}
}

func TestHandleAlertTest_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/alerts/test", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// --- dispatchAlert coverage ---

func TestDispatchAlert_Disabled(t *testing.T) {
	alertCfg.mu.Lock()
	alertCfg.enabled = false
	alertCfg.mu.Unlock()

	srv := newTestServer(nil, nil)
	srv.dispatchAlert(map[string]any{"severity": "critical"})
	// Should not panic, should not send
}

func TestDispatchAlert_LowSeverity(t *testing.T) {
	alertCfg.mu.Lock()
	alertCfg.enabled = true
	alertCfg.minSeverity = "critical"
	alertCfg.webhookURL = ""
	alertCfg.emailTo = ""
	alertCfg.mu.Unlock()
	defer func() {
		alertCfg.mu.Lock()
		alertCfg.enabled = false
		alertCfg.minSeverity = "warning"
		alertCfg.mu.Unlock()
	}()

	srv := newTestServer(nil, nil)
	// Severity "info" < minSeverity "critical" → should be skipped
	srv.dispatchAlert(map[string]any{"severity": "info"})
}

func TestDispatchAlert_WithWebhook(t *testing.T) {
	// Start a test HTTP server to receive webhook
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv2.Close()

	alertCfg.mu.Lock()
	alertCfg.enabled = true
	alertCfg.minSeverity = "warning"
	alertCfg.webhookURL = srv2.URL
	alertCfg.emailTo = ""
	alertCfg.mu.Unlock()
	defer func() {
		alertCfg.mu.Lock()
		alertCfg.enabled = false
		alertCfg.webhookURL = ""
		alertCfg.minSeverity = "warning"
		alertCfg.mu.Unlock()
	}()

	srv := newTestServer(nil, nil)
	srv.dispatchAlert(map[string]any{
		"severity": "warning",
		"message":  "test alert",
	})
	// Give async webhook goroutine time to fire
	time.Sleep(100 * time.Millisecond)
}

// --- StartRetentionCleanup with cleanup error ---

func TestStartRetentionCleanup_WithCleanupError(t *testing.T) {
	repo := &mockRepo{cleanupErr: errSimple("db error")}
	svc := service.NewAuditService(repo)
	srv := NewHTTPServer(svc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv.StartRetentionCleanup(ctx, 50*time.Millisecond)
	time.Sleep(150 * time.Millisecond)

	// lastRun should NOT be set because cleanup returned error
	srv.retention.mu.RLock()
	defer srv.retention.mu.RUnlock()
	if !srv.retention.lastRun.IsZero() {
		t.Fatal("expected lastRun to be zero when cleanup errors")
	}
}

// --- generateComplianceReport direct test ---

func TestGenerateComplianceReport_EmptyEvents(t *testing.T) {
	srv := newTestServer(nil, nil)
	report := srv.generateComplianceReport("soc2", testTenantID, time.Now().Add(-24*time.Hour), time.Now(), nil)
	if report["format"] != "soc2" {
		t.Fatalf("expected soc2, got %v", report["format"])
	}
	summary, ok := report["summary"].(map[string]any)
	if !ok {
		t.Fatal("expected summary")
	}
	if summary["total_auth_events"].(int) != 0 {
		t.Fatalf("expected 0 auth events, got %v", summary["total_auth_events"])
	}
}

func TestGenerateComplianceReport_GDPR_WithEvents(t *testing.T) {
	events := []*domain.AuditEvent{
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorUser,
			Action: "user.login", Result: domain.ResultSuccess, IPAddress: "1.2.3.4",
			CreatedAt: time.Now().UTC()},
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorSystem,
			Action: "data.export", Result: domain.ResultSuccess,
			CreatedAt: time.Now().UTC()},
	}
	srv := newTestServer(nil, nil)
	report := srv.generateComplianceReport("gdpr", testTenantID, time.Now().Add(-24*time.Hour), time.Now(), events)
	ctrls, ok := report["compliance_controls"].(map[string]any)
	if !ok {
		t.Fatal("expected compliance_controls")
	}
	if _, ok := ctrls["art_30_records"]; !ok {
		t.Fatal("expected art_30_records")
	}
}

// --- pct helper ---

func TestPct(t *testing.T) {
	if pct(1, 4) != 25.0 {
		t.Fatalf("expected 25.0, got %v", pct(1, 4))
	}
	if pct(0, 0) != 0.0 {
		t.Fatalf("expected 0.0 for zero denom, got %v", pct(0, 0))
	}
}

// --- handleSearch with resource_name match ---

func TestHandleSearch_ResourceName(t *testing.T) {
	events := []*domain.AuditEvent{
		{ID: uuid.New(), TenantID: testTenantID, ActorType: domain.ActorUser,
			ActorName: "admin", Action: "user.update", Result: domain.ResultSuccess,
			ResourceName: "special-document.pdf", IPAddress: "10.0.0.1",
			CreatedAt: time.Now().UTC()},
	}
	srv := newTestServer(events, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/search?q=special-document&tenant_id="+testTenantID.String(), "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["count"].(float64) != 1 {
		t.Fatalf("expected 1 result matching resource name, got %v", resp["count"])
	}
}

// --- StreamHub coverage ---

func TestStreamHub_BroadcastFullBuffer(t *testing.T) {
	hub := NewStreamHub()
	id, ch := hub.Subscribe()

	// Fill the buffer (64 capacity)
	for i := 0; i < 64; i++ {
		hub.Broadcast(&domain.AuditEvent{ID: uuid.New(), TenantID: testTenantID})
	}
	// 65th event should be silently dropped (buffer full)
	hub.Broadcast(&domain.AuditEvent{ID: uuid.New(), TenantID: testTenantID})

	// Drain channel
	count := 0
	for range ch {
		count++
		if count == 64 {
			break
		}
	}
	if count != 64 {
		t.Fatalf("expected 64 events in buffer, got %d", count)
	}

	hub.Unsubscribe(id)
}

func TestStreamHub_SubscriberCount(t *testing.T) {
	hub := NewStreamHub()
	if hub.SubscriberCount() != 0 {
		t.Fatalf("expected 0, got %d", hub.SubscriberCount())
	}
	id1, _ := hub.Subscribe()
	id2, _ := hub.Subscribe()
	if hub.SubscriberCount() != 2 {
		t.Fatalf("expected 2, got %d", hub.SubscriberCount())
	}
	hub.Unsubscribe(id1)
	if hub.SubscriberCount() != 1 {
		t.Fatalf("expected 1, got %d", hub.SubscriberCount())
	}
	hub.Unsubscribe(id2)
	if hub.SubscriberCount() != 0 {
		t.Fatalf("expected 0, got %d", hub.SubscriberCount())
	}
}

func TestStreamHub_BroadcastToMultipleSubscribers(t *testing.T) {
	hub := NewStreamHub()
	id1, ch1 := hub.Subscribe()
	id2, ch2 := hub.Subscribe()
	defer hub.Unsubscribe(id1)
	defer hub.Unsubscribe(id2)

	event := &domain.AuditEvent{ID: uuid.New(), TenantID: testTenantID}
	hub.Broadcast(event)

	select {
	case e := <-ch1:
		if e.ID != event.ID {
			t.Fatal("subscriber 1 got wrong event")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber 1 did not receive event")
	}

	select {
	case e := <-ch2:
		if e.ID != event.ID {
			t.Fatal("subscriber 2 got wrong event")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber 2 did not receive event")
	}
}

// --- jsonNumber/formatClientID coverage ---

func TestJsonNumber(t *testing.T) {
	if jsonNumber(0) != "0" {
		t.Fatalf("expected '0', got %v", jsonNumber(0))
	}
	if jsonNumber(42) != "42" {
		t.Fatalf("expected '42', got %v", jsonNumber(42))
	}
	if jsonNumber(123) != "123" {
		t.Fatalf("expected '123', got %v", jsonNumber(123))
	}
}

func TestFormatClientID(t *testing.T) {
	id := formatClientID(1)
	if !strings.HasPrefix(id, "ws-") {
		t.Fatalf("expected ws- prefix, got %s", id)
	}
}

// --- writeAuditCSV with nil optionals ---

func TestWriteAuditCSV_NilOptionals(t *testing.T) {
	events := []*domain.AuditEvent{
		{
			ID:        uuid.New(),
			TenantID:  testTenantID,
			ActorType: domain.ActorSystem,
			Action:    "system.cleanup",
			Result:    domain.ResultSuccess,
			CreatedAt: time.Now().UTC(),
		},
	}
	w := httptest.NewRecorder()
	writeAuditCSV(w, events)
	body := w.Body.String()
	if !strings.Contains(body, "system.cleanup") {
		t.Fatal("expected action in CSV")
	}
	if !strings.Contains(body, "system") {
		t.Fatal("expected actor_type in CSV")
	}
}
