package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// --- Mock AuditRepo ---

type mockRepo struct {
	events     []*domain.AuditEvent
	stats      *domain.Stats
	cleanupN   int64
	cleanupErr error
	listErr    error
	statsErr   error
	getErr     error
}

func (m *mockRepo) Insert(ctx interface{ Done() <-chan struct{} }, e *domain.AuditEvent) error {
	return nil
}

func (m *mockRepo) GetByID(ctx interface{ Done() <-chan struct{} }, id uuid.UUID) (*domain.AuditEvent, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, e := range m.events {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, m.getErr
}

func (m *mockRepo) List(ctx interface{ Done() <-chan struct{} }, filter domain.ListFilter, limit, offset int) ([]*domain.AuditEvent, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	return m.events, len(m.events), nil
}

func (m *mockRepo) GetStats(ctx interface{ Done() <-chan struct{} }, tenantID uuid.UUID, since time.Time) (*domain.Stats, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.stats, nil
}

func (m *mockRepo) DeleteOlderThan(ctx interface{ Done() <-chan struct{} }, before time.Time) (int64, error) {
	if m.cleanupErr != nil {
		return 0, m.cleanupErr
	}
	return m.cleanupN, nil
}

// --- Test helpers ---

func newTestServer(events []*domain.AuditEvent, stats *domain.Stats) *HTTPServer {
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	if events == nil {
		actorID := uuid.New()
		events = []*domain.AuditEvent{
			{
				ID:        uuid.New(),
				TenantID:  tenantID,
				ActorType: domain.ActorUser,
				ActorID:   &actorID,
				ActorName: "admin",
				Action:    "user.login",
				Result:    domain.ResultSuccess,
				CreatedAt: time.Now().UTC(),
				IPAddress: "192.168.1.1",
			},
		}
	}

	if stats == nil {
		stats = &domain.Stats{
			TotalEvents24h:  10,
			FailedLogins24h: 2,
			EventsByAction:  map[string]int{"user.login": 8},
			HourlyDistribution: []domain.HourlyCount{
				{Hour: time.Now().UTC().Truncate(time.Hour), Count: 3},
			},
			TopActors: []domain.ActorActivity{
				{ActorID: uuid.New(), ActorName: "admin", Count: 5},
			},
		}
	}

	repo := &mockRepo{events: events, stats: stats}
	svc := newTestService(repo)
	return NewHTTPServer(svc)
}

// newTestService creates an AuditService using the mock repo.
// We need to use a type assertion since AuditRepo is an interface.
func newTestService(repo *mockRepo) *service.AuditService {
	return service.NewAuditService(repo)
}

func doRequest(t *testing.T, srv *HTTPServer, method, path string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.RegisterRoutes(http.NewServeMux())

	// Use a mux that's already registered
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	mux.ServeHTTP(w, req)
	return w
}

// --- Tests ---

func TestHandleEvents_GetList(t *testing.T) {
	srv := newTestServer(nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"

	w := doRequest(t, srv, "GET", "/api/v1/audit/events?tenant_id="+tenantID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	events, ok := resp["events"].([]any)
	if !ok {
		t.Fatal("expected events array")
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestHandleEvents_MissingTenantID(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "GET", "/api/v1/audit/events", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleEvents_InvalidTenantID(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "GET", "/api/v1/audit/events?tenant_id=invalid", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleEvents_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"
	w := doRequest(t, srv, "POST", "/api/v1/audit/events?tenant_id="+tenantID, "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleStats_Get(t *testing.T) {
	srv := newTestServer(nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"
	w := doRequest(t, srv, "GET", "/api/v1/audit/stats?tenant_id="+tenantID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total_events_24h"].(float64) != 10 {
		t.Fatalf("expected 10 events, got %v", resp["total_events_24h"])
	}
	if resp["failed_logins_24h"].(float64) != 2 {
		t.Fatalf("expected 2 failed logins, got %v", resp["failed_logins_24h"])
	}
}

func TestHandleStats_MissingTenantID(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "GET", "/api/v1/audit/stats", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleRetention_Get(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "GET", "/api/v1/audit/retention", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["retention_days"].(float64) != 90 {
		t.Fatalf("expected 90 days default, got %v", resp["retention_days"])
	}
	if resp["enabled"] != true {
		t.Fatalf("expected enabled=true, got %v", resp["enabled"])
	}
}

func TestHandleRetention_Put(t *testing.T) {
	srv := newTestServer(nil, nil)
	enabled := false
	body := `{"retention_days": 30, "enabled": false}`
	_ = enabled

	w := doRequest(t, srv, "PUT", "/api/v1/audit/retention", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["retention_days"].(float64) != 30 {
		t.Fatalf("expected 30 days, got %v", resp["retention_days"])
	}
	if resp["enabled"] != false {
		t.Fatalf("expected enabled=false, got %v", resp["enabled"])
	}

	// Verify GET returns updated value
	w2 := doRequest(t, srv, "GET", "/api/v1/audit/retention", "")
	json.Unmarshal(w2.Body.Bytes(), &resp)
	if resp["retention_days"].(float64) != 30 {
		t.Fatalf("expected 30 after update, got %v", resp["retention_days"])
	}
}

func TestHandleRetention_Put_InvalidJSON(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "PUT", "/api/v1/audit/retention", "not json")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleRetention_Post(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "POST", "/api/v1/audit/retention?days=7", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "completed" {
		t.Fatalf("expected completed, got %v", resp["status"])
	}
	if resp["retention_days"].(float64) != 7 {
		t.Fatalf("expected 7 days, got %v", resp["retention_days"])
	}
}

func TestHandleRetention_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "DELETE", "/api/v1/audit/retention", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleAnomalyRules_Get(t *testing.T) {
	// Reset global rules
	anomalyRules = []map[string]any{}
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "GET", "/api/v1/audit/rules", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	rules, ok := resp["rules"].([]any)
	if !ok {
		t.Fatal("expected rules array")
	}
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(rules))
	}
}

func TestHandleAnomalyRules_Create(t *testing.T) {
	anomalyRules = []map[string]any{}
	srv := newTestServer(nil, nil)
	body := `{"name": "Brute Force", "action": "user.login", "threshold": 10, "window_minutes": 5, "severity": "critical"}`

	w := doRequest(t, srv, "POST", "/api/v1/audit/rules", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "Brute Force" {
		t.Fatalf("expected name, got %v", resp["name"])
	}
	if resp["severity"] != "critical" {
		t.Fatalf("expected critical, got %v", resp["severity"])
	}
}

func TestHandleAnomalyRules_Create_MissingFields(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "POST", "/api/v1/audit/rules", `{"threshold": 5}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAnomalyRules_Create_Defaults(t *testing.T) {
	anomalyRules = []map[string]any{}
	srv := newTestServer(nil, nil)
	body := `{"name": "Test", "action": "user.login"}`

	w := doRequest(t, srv, "POST", "/api/v1/audit/rules", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["threshold"].(float64) != 5 {
		t.Fatalf("expected default threshold 5, got %v", resp["threshold"])
	}
	if resp["window_minutes"].(float64) != 5 {
		t.Fatalf("expected default window 5, got %v", resp["window_minutes"])
	}
	if resp["severity"] != "warning" {
		t.Fatalf("expected default severity warning, got %v", resp["severity"])
	}
}

func TestHandleAnomalyRules_Create_InvalidJSON(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "POST", "/api/v1/audit/rules", "bad json")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAnomalyRules_Delete(t *testing.T) {
	// Add a rule first
	ruleID := uuid.New().String()
	anomalyRules = []map[string]any{
		{"id": ruleID, "name": "Test Rule", "action": "user.login", "threshold": 5, "window_minutes": 5, "severity": "warning"},
	}
	srv := newTestServer(nil, nil)

	w := doRequest(t, srv, "DELETE", "/api/v1/audit/rules?id="+ruleID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(anomalyRules) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(anomalyRules))
	}
}

func TestHandleAnomalyRules_Delete_MissingID(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "DELETE", "/api/v1/audit/rules", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleAnomalyRules_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "PATCH", "/api/v1/audit/rules", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleExport_JSON(t *testing.T) {
	srv := newTestServer(nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"
	w := doRequest(t, srv, "GET", "/api/v1/audit/export?tenant_id="+tenantID+"&format=json", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "events") {
		t.Fatal("expected events in response")
	}
}

func TestHandleExport_CSV(t *testing.T) {
	srv := newTestServer(nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"
	w := doRequest(t, srv, "GET", "/api/v1/audit/export?tenant_id="+tenantID+"&format=csv", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/csv" {
		t.Fatalf("expected text/csv, got %s", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "user.login") {
		t.Fatal("expected CSV body to contain action")
	}
}

func TestHandleExport_InvalidFormat(t *testing.T) {
	srv := newTestServer(nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"
	w := doRequest(t, srv, "GET", "/api/v1/audit/export?tenant_id="+tenantID+"&format=xml", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleExport_MissingTenantID(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "GET", "/api/v1/audit/export", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleExport_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	tenantID := "00000000-0000-0000-0000-000000000001"
	w := doRequest(t, srv, "POST", "/api/v1/audit/export?tenant_id="+tenantID, "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandleEventByID(t *testing.T) {
	events := newTestServer(nil, nil).retention.days // just to have server
	_ = events
	srv := newTestServer(nil, nil)

	// Get the event ID from the mock events
	tenantID := "00000000-0000-0000-0000-000000000001"
	w := doRequest(t, srv, "GET", "/api/v1/audit/events?tenant_id="+tenantID, "")
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	eventsArr := resp["events"].([]any)
	firstEvent := eventsArr[0].(map[string]any)
	eventID := firstEvent["id"].(string)

	// Fetch by ID
	w2 := doRequest(t, srv, "GET", "/api/v1/audit/events/"+eventID, "")
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandleEventByID_InvalidID(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(t, srv, "GET", "/api/v1/audit/events/not-a-uuid", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestEventToJSON(t *testing.T) {
	actorID := uuid.New()
	resourceID := uuid.New()
	e := &domain.AuditEvent{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ActorType:    domain.ActorUser,
		ActorID:      &actorID,
		ActorName:    "testuser",
		Action:       "user.register",
		ResourceType: "user",
		ResourceID:   &resourceID,
		ResourceName: "testuser",
		Result:       domain.ResultSuccess,
		IPAddress:    "10.0.0.1",
		UserAgent:    "test-agent",
		RequestID:    "req-123",
		Metadata:     map[string]any{"key": "value"},
		CreatedAt:    time.Now().UTC(),
	}

	m := eventToJSON(e)
	if m["action"] != "user.register" {
		t.Fatalf("expected user.register, got %v", m["action"])
	}
	if m["actor_name"] != "testuser" {
		t.Fatalf("expected testuser, got %v", m["actor_name"])
	}
	if m["resource_id"] != resourceID.String() {
		t.Fatalf("expected resource_id, got %v", m["resource_id"])
	}
	if m["actor_id"] != actorID.String() {
		t.Fatalf("expected actor_id, got %v", m["actor_id"])
	}
	meta, ok := m["metadata"].(map[string]any)
	if !ok || meta["key"] != "value" {
		t.Fatalf("expected metadata, got %v", m["metadata"])
	}
}

func TestStatsToJSON(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()
	stats := &domain.Stats{
		TotalEvents24h:  42,
		FailedLogins24h: 3,
		EventsByAction:  map[string]int{"user.login": 20, "role.assign": 5},
		HourlyDistribution: []domain.HourlyCount{
			{Hour: time.Now().UTC(), Count: 10},
		},
		TopActors: []domain.ActorActivity{
			{ActorID: actorID, ActorName: "admin", Count: 15},
		},
	}

	m := statsToJSON(stats)
	if m["total_events_24h"] != 42 {
		t.Fatalf("expected 42, got %v", m["total_events_24h"])
	}
	if m["failed_logins_24h"] != 3 {
		t.Fatalf("expected 3, got %v", m["failed_logins_24h"])
	}

	actions, ok := m["events_by_action"].(map[string]any)
	if !ok {
		t.Fatal("expected events_by_action map")
	}
	if actions["user.login"].(int) != 20 {
		t.Fatalf("expected 20 logins, got %v", actions["user.login"])
	}

	hourly, ok := m["hourly_distribution"].([]map[string]any)
	if !ok || len(hourly) != 1 {
		t.Fatalf("expected 1 hourly entry, got %v", m["hourly_distribution"])
	}

	actors, ok := m["top_actors"].([]map[string]any)
	if !ok || len(actors) != 1 {
		t.Fatalf("expected 1 actor, got %v", m["top_actors"])
	}
	if actors[0]["actor_name"] != "admin" {
		t.Fatalf("expected admin, got %v", actors[0]["actor_name"])
	}

	_ = tenantID // keep tenantID for potential future use
}
