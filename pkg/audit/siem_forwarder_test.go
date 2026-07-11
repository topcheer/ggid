package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestSIEMForwarder_DefaultConfig(t *testing.T) {
	cfg := DefaultSIEMConfig()
	if cfg.Provider != SIEMProviderGeneric {
		t.Errorf("expected generic provider, got %s", cfg.Provider)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("expected batch size 100, got %d", cfg.BatchSize)
	}
	if cfg.FlushInterval != 5*time.Second {
		t.Errorf("expected flush interval 5s, got %v", cfg.FlushInterval)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", cfg.MaxRetries)
	}
}

func TestSIEMForwarder_FormatSplunk(t *testing.T) {
	f := NewSIEMForwarder(SIEMConfig{
		Provider:  SIEMProviderSplunk,
		IndexName: "main",
	})
	event := Event{
		Action:    "user.login",
		Result:    "success",
		IPAddress: "192.168.1.1",
		ActorType: "user",
	}
	event.CreatedAt = time.Now()

	data, err := f.formatSplunk([]Event{event})
	if err != nil {
		t.Fatalf("formatSplunk failed: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(data[:len(data)-1], &entry); err != nil {
		t.Fatalf("unmarshal splunk entry: %v", err)
	}
	if entry["sourcetype"] != "ggid:audit" {
		t.Errorf("expected sourcetype ggid:audit, got %v", entry["sourcetype"])
	}
	if entry["index"] != "main" {
		t.Errorf("expected index main, got %v", entry["index"])
	}
}

func TestSIEMForwarder_FormatDatadog(t *testing.T) {
	f := NewSIEMForwarder(SIEMConfig{
		Provider:  SIEMProviderDatadog,
		IndexName: "ggid",
	})
	event := Event{
		Action:       "user.login",
		Result:       "success",
		ActorType:    "user",
		ResourceType: "session",
	}
	event.CreatedAt = time.Now()

	data, err := f.formatDatadog([]Event{event})
	if err != nil {
		t.Fatalf("formatDatadog failed: %v", err)
	}

	var logs []map[string]any
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("unmarshal datadog logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0]["ddsource"] != "ggid" {
		t.Errorf("expected ddsource ggid, got %v", logs[0]["ddsource"])
	}
	if logs[0]["service"] != "ggid" {
		t.Errorf("expected service ggid, got %v", logs[0]["service"])
	}
}

func TestSIEMForwarder_FormatElasticsearch(t *testing.T) {
	f := NewSIEMForwarder(SIEMConfig{
		Provider:  SIEMProviderElasticsearch,
		IndexName: "audit-logs",
	})
	event := Event{
		Action: "user.login",
		Result: "success",
	}
	event.CreatedAt = time.Now()

	data, err := f.formatElasticsearch([]Event{event})
	if err != nil {
		t.Fatalf("formatElasticsearch failed: %v", err)
	}

	lines := splitNewlines(string(data))
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (action+source), got %d", len(lines))
	}

	var action map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &action); err != nil {
		t.Fatalf("unmarshal es action: %v", err)
	}
	idx, ok := action["index"].(map[string]any)
	if !ok {
		t.Fatal("expected index key in action")
	}
	if idx["_index"] != "audit-logs" {
		t.Errorf("expected index audit-logs, got %v", idx["_index"])
	}
}

func TestSIEMForwarder_FormatGeneric(t *testing.T) {
	f := NewSIEMForwarder(DefaultSIEMConfig())
	event := Event{Action: "test", Result: "success"}

	data, err := f.formatGeneric([]Event{event})
	if err != nil {
		t.Fatalf("formatGeneric failed: %v", err)
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("unmarshal generic: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Action != "test" {
		t.Errorf("expected action test, got %s", events[0].Action)
	}
}

func TestSIEMForwarder_SendSuccess(t *testing.T) {
	var receivedCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("expected non-empty body")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer auth, got %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.APIKey = "test-key"
	cfg.BatchSize = 1
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.MaxRetries = 1

	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test.event", Result: "success"})

	// Manually flush since we don't Start()
	time.Sleep(50 * time.Millisecond)
	f.flush()

	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Errorf("expected 1 received, got %d", atomic.LoadInt32(&receivedCount))
	}
}

func TestSIEMForwarder_SendRetry(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.APIKey = "key"
	cfg.MaxRetries = 3
	cfg.Timeout = 5 * time.Second

	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "success"})

	// flush manually
	f.flush()

	if atomic.LoadInt32(&attempts) < 2 {
		t.Errorf("expected at least 2 attempts (retry), got %d", atomic.LoadInt32(&attempts))
	}
}

func TestSIEMForwarder_SendAllRetriesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.MaxRetries = 2
	cfg.Timeout = 2 * time.Second

	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "fail"})

	// Should not panic, just log error
	f.flush()
}

func TestSIEMForwarder_SplunkAuth(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Provider = SIEMProviderSplunk
	cfg.Endpoint = srv.URL
	cfg.APIKey = "splunk-token"
	cfg.MaxRetries = 1

	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "ok"})
	f.flush()

	if authHeader != "Splunk splunk-token" {
		t.Errorf("expected 'Splunk splunk-token', got '%s'", authHeader)
	}
}

func TestSIEMForwarder_DatadogAuth(t *testing.T) {
	var apiKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey = r.Header.Get("DD-API-KEY")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Provider = SIEMProviderDatadog
	cfg.Endpoint = srv.URL
	cfg.APIKey = "dd-key"
	cfg.MaxRetries = 1

	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "ok"})
	f.flush()

	if apiKey != "dd-key" {
		t.Errorf("expected DD-API-KEY dd-key, got '%s'", apiKey)
	}
}

// splitNewlines splits a byte payload into non-empty lines.
func splitNewlines(data string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
