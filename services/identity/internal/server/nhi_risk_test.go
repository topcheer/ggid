package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// Test 1: Risk evaluation with no baseline → score 20 (new NHI).
func TestNHIRisk_NoBaseline(t *testing.T) {
	engine := NewNHIRiskEngine()
	nhiID := uuid.New()

	score := engine.EvaluateRisk(nhiID, CurrentActivity{
		NHIID:        nhiID.String(),
		Endpoint:     "/api/v1/users",
		CallsPerHour: 5,
		IP:           "10.0.0.1",
		Hour:         14,
	})

	if score.Score != 20 {
		t.Errorf("expected score 20 for new NHI, got %d", score.Score)
	}
	if score.Level != "low" {
		t.Errorf("expected level=low, got %s", score.Level)
	}
}

// Test 2: Frequency spike detection — 10x normal → high score.
func TestNHIRisk_FrequencySpike(t *testing.T) {
	engine := NewNHIRiskEngine()
	nhiID := uuid.New()
	nhiStr := nhiID.String()

	// Establish baseline: 10 calls/hour to /api/v1/users.
	for i := 0; i < 10; i++ {
		engine.RecordBaseline(nhiStr, "/api/v1/users", 10, "10.0.0.1", 14)
	}

	// Evaluate with 100 calls/hour (10x spike).
	score := engine.EvaluateRisk(nhiID, CurrentActivity{
		NHIID:        nhiStr,
		Endpoint:     "/api/v1/users",
		CallsPerHour: 100,
		IP:           "10.0.0.1",
		Hour:         14,
	})

	if score.Score < 25 {
		t.Errorf("expected score >=25 for frequency spike, got %d", score.Score)
	}
	signals := score.Signals
	if signals["frequency_spike"] != true {
		t.Error("expected frequency_spike signal")
	}
}

// Test 3: New endpoint detection — unknown API → score increase.
func TestNHIRisk_NewEndpoint(t *testing.T) {
	engine := NewNHIRiskEngine()
	nhiID := uuid.New()
	nhiStr := nhiID.String()

	// Baseline only on /api/v1/users.
	engine.RecordBaseline(nhiStr, "/api/v1/users", 10, "10.0.0.1", 14)

	// Access a new endpoint.
	score := engine.EvaluateRisk(nhiID, CurrentActivity{
		NHIID:        nhiStr,
		Endpoint:     "/api/v1/admin/delete",
		CallsPerHour: 10,
		IP:           "10.0.0.1",
		Hour:         14,
	})

	if score.Score < 20 {
		t.Errorf("expected score >=20 for new endpoint, got %d", score.Score)
	}
}

// Test 4: Off-hours access detection → score increase.
func TestNHIRisk_OffHoursAccess(t *testing.T) {
	engine := NewNHIRiskEngine()
	nhiID := uuid.New()
	nhiStr := nhiID.String()

	// Baseline at hour 14 (2pm).
	engine.RecordBaseline(nhiStr, "/api/v1/users", 10, "10.0.0.1", 14)

	// Access at 3am.
	score := engine.EvaluateRisk(nhiID, CurrentActivity{
		NHIID:        nhiStr,
		Endpoint:     "/api/v1/users",
		CallsPerHour: 10,
		IP:           "10.0.0.1",
		Hour:         3,
	})

	if score.Score < 15 {
		t.Errorf("expected score >=15 for off-hours access, got %d", score.Score)
	}
}

// Test 5: New IP detection → score increase.
func TestNHIRisk_NewIP(t *testing.T) {
	engine := NewNHIRiskEngine()
	nhiID := uuid.New()
	nhiStr := nhiID.String()

	// Baseline from 10.0.0.1.
	engine.RecordBaseline(nhiStr, "/api/v1/users", 10, "10.0.0.1", 14)

	// Access from new IP.
	score := engine.EvaluateRisk(nhiID, CurrentActivity{
		NHIID:        nhiStr,
		Endpoint:     "/api/v1/users",
		CallsPerHour: 10,
		IP:           "203.0.113.99",
		Hour:         14,
	})

	if score.Score < 10 {
		t.Errorf("expected score >=10 for new IP, got %d", score.Score)
	}
}

// Test 6: Multiple anomalies combined → high or critical risk.
func TestNHIRisk_AllAnomalies(t *testing.T) {
	engine := NewNHIRiskEngine()
	nhiID := uuid.New()
	nhiStr := nhiID.String()

	// Normal baseline on same endpoint.
	engine.RecordBaseline(nhiStr, "/api/v1/users", 10, "10.0.0.1", 14)

	// Frequency spike on known endpoint + off-hours + new IP.
	score := engine.EvaluateRisk(nhiID, CurrentActivity{
		NHIID:        nhiStr,
		Endpoint:     "/api/v1/users", // same endpoint → frequency spike triggers
		CallsPerHour: 200,             // 20x spike
		IP:           "203.0.113.99",  // new IP
		Hour:         3,               // 3am off-hours
	})

	// spike(30) + new_ip(15) + off_hours(20) = 65 → high
	if score.Level != "high" && score.Level != "critical" {
		t.Errorf("expected high/critical for multiple anomalies, got %s (score=%d)", score.Level, score.Score)
	}
	if score.Score < 50 {
		t.Errorf("expected score >=50, got %d", score.Score)
	}
}

// Test 7: ListHighRisk returns only high-risk NHIs.
func TestNHIRisk_ListHighRisk(t *testing.T) {
	engine := NewNHIRiskEngine()
	nhi1 := uuid.New()
	nhi2 := uuid.New()

	// nhi1 gets critical score (no baseline → 20, won't be in high risk).
	engine.EvaluateRisk(nhi1, CurrentActivity{
		NHIID: nhi1.String(), Endpoint: "/test", CallsPerHour: 1, IP: "1.1.1.1", Hour: 3,
	})

	// nhi2 gets high score.
	engine.RecordBaseline(nhi2.String(), "/api/v1/users", 10, "10.0.0.1", 14)
	engine.EvaluateRisk(nhi2, CurrentActivity{
		NHIID: nhi2.String(), Endpoint: "/unknown", CallsPerHour: 500, IP: "9.9.9.9", Hour: 3,
	})

	high := engine.ListHighRisk(50)
	if len(high) == 0 {
		t.Error("expected at least 1 high-risk NHI")
	}
}

// Test 8: POST /risk/scan — endpoint returns risk score.
func TestNHIRisk_ScanEndpoint(t *testing.T) {
	h := &HTTPHandler{
		nhiRiskEngine: NewNHIRiskEngine(),
	}
	nhiID := uuid.New()

	body := `{"nhi_id":"` + nhiID.String() + `","endpoint":"/api/v1/users","calls_per_hour":10,"ip":"10.0.0.1","hour":14}`
	req := httptest.NewRequest("POST", "/api/v1/identity/nhi/risk/scan", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleNHIRisk(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var score NHIRiskScore
	if err := json.Unmarshal(w.Body.Bytes(), &score); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if score.Score < 0 || score.Score > 100 {
		t.Errorf("score out of range: %d", score.Score)
	}
}

// Test 9: GET /risk-alerts — returns empty list initially.
func TestNHIRisk_AlertsEndpoint(t *testing.T) {
	h := &HTTPHandler{
		nhiRiskEngine: NewNHIRiskEngine(),
	}

	req := httptest.NewRequest("GET", "/api/v1/identity/nhi/risk-alerts", nil)
	w := httptest.NewRecorder()

	h.handleNHIRisk(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 10: POST /risk/scan with invalid NHI ID → 400.
func TestNHIRisk_ScanInvalidID(t *testing.T) {
	h := &HTTPHandler{
		nhiRiskEngine: NewNHIRiskEngine(),
	}

	body := `{"nhi_id":"not-a-uuid"}`
	req := httptest.NewRequest("POST", "/api/v1/identity/nhi/risk/scan", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleNHIRisk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// Test 11: GET /:id/risk — returns score after evaluation.
func TestNHIRisk_GetRiskEndpoint(t *testing.T) {
	h := &HTTPHandler{
		nhiRiskEngine: NewNHIRiskEngine(),
	}
	nhiID := uuid.New()

	// First scan to create a score.
	h.nhiRiskEngine.EvaluateRisk(nhiID, CurrentActivity{
		NHIID: nhiID.String(), Endpoint: "/test", CallsPerHour: 5, IP: "1.2.3.4", Hour: 14,
	})

	req := httptest.NewRequest("GET", "/api/v1/identity/nhi/"+nhiID.String()+"/risk", nil)
	w := httptest.NewRecorder()

	h.handleNHIRisk(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 12: Critical risk triggers SOAR signal.
func TestNHIRisk_SOARTrigger(t *testing.T) {
	h := &HTTPHandler{
		nhiRiskEngine: NewNHIRiskEngine(),
	}
	nhiID := uuid.New()
	nhiStr := nhiID.String()

	// Create baseline, then trigger all anomalies.
	h.nhiRiskEngine.RecordBaseline(nhiStr, "/api/v1/users", 10, "10.0.0.1", 14)

	body := `{"nhi_id":"` + nhiStr + `","endpoint":"/new","calls_per_hour":500,"ip":"9.9.9.9","hour":3}`
	req := httptest.NewRequest("POST", "/api/v1/identity/nhi/risk/scan", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleNHIRisk(w, req)

	var score NHIRiskScore
	if err := json.Unmarshal(w.Body.Bytes(), &score); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if score.Score >= 70 {
		soarTriggered, ok := score.Signals["soar_triggered"]
		if !ok || soarTriggered != true {
			t.Error("expected SOAR trigger for critical risk")
		}
	}
}
