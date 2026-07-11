package audit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSIEMForwarder_DefaultConfig(t *testing.T) {
	cfg := DefaultSIEMConfig()
	if cfg.Provider != SIEMProviderGeneric {
		t.Errorf("expected generic, got %s", cfg.Provider)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("expected batch 100, got %d", cfg.BatchSize)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected retries 3, got %d", cfg.MaxRetries)
	}
}

func TestSIEMForwarder_FormatSplunk(t *testing.T) {
	f := NewSIEMForwarder(SIEMConfig{Provider: SIEMProviderSplunk, IndexName: "main"})
	e := Event{ID: uuid.New(), Action: "login", Result: "success", CreatedAt: time.Now()}
	data, err := f.formatSplunk([]Event{e})
	if err != nil {
		t.Fatal(err)
	}
	var entry map[string]any
	if err := json.Unmarshal(data[:len(data)-1], &entry); err != nil {
		t.Fatal(err)
	}
	if entry["sourcetype"] != "ggid:audit" {
		t.Errorf("got %v", entry["sourcetype"])
	}
}

func TestSIEMForwarder_FormatDatadog(t *testing.T) {
	f := NewSIEMForwarder(SIEMConfig{Provider: SIEMProviderDatadog, IndexName: "ggid"})
	e := Event{Action: "login", Result: "success", CreatedAt: time.Now()}
	data, err := f.formatDatadog([]Event{e})
	if err != nil {
		t.Fatal(err)
	}
	var logs []map[string]any
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatal(err)
	}
	if logs[0]["ddsource"] != "ggid" {
		t.Errorf("got %v", logs[0]["ddsource"])
	}
}

func TestSIEMForwarder_FormatElasticsearch(t *testing.T) {
	f := NewSIEMForwarder(SIEMConfig{Provider: SIEMProviderElasticsearch, IndexName: "audit"})
	e := Event{ID: uuid.New(), Action: "login", Result: "success", CreatedAt: time.Now()}
	data, err := f.formatElasticsearch([]Event{e})
	if err != nil {
		t.Fatal(err)
	}
	// ES bulk = 2 lines per event
	lines := splitNL(string(data))
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	var action map[string]any
	json.Unmarshal([]byte(lines[0]), &action)
	idx := action["index"].(map[string]any)
	if idx["_index"] != "audit" {
		t.Errorf("got %v", idx["_index"])
	}
}

func TestSIEMForwarder_FormatGeneric(t *testing.T) {
	f := NewSIEMForwarder(DefaultSIEMConfig())
	events := []Event{{Action: "a", Result: "ok"}, {Action: "b", Result: "ok"}}
	data, err := f.formatGeneric(events)
	if err != nil {
		t.Fatal(err)
	}
	var result []Event
	json.Unmarshal(data, &result)
	if len(result) != 2 {
		t.Errorf("got %d", len(result))
	}
}

func TestSIEMForwarder_SendSuccess(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Error("missing auth")
		}
		io.ReadAll(r.Body)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.APIKey = "key"
	cfg.MaxRetries = 1
	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "ok"})
	f.flush()
	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestSIEMForwarder_RetryThenSucceed(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&attempts, 1)
		if c < 2 {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.MaxRetries = 3
	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "ok"})
	f.flush()
	if atomic.LoadInt32(&attempts) < 2 {
		t.Errorf("expected retry, got %d attempts", attempts)
	}
}

func TestSIEMForwarder_AllRetriesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.MaxRetries = 2
	cfg.Timeout = 1 * time.Second
	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "ok"})
	f.flush() // should not panic
}

func TestSIEMForwarder_StartStopFlushes(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.BatchSize = 100
	cfg.FlushInterval = 10 * time.Second
	cfg.MaxRetries = 1
	f := NewSIEMForwarder(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.Start(ctx)

	f.Forward(Event{Action: "a", Result: "ok"})
	f.Forward(Event{Action: "b", Result: "ok"})
	f.Stop()

	if atomic.LoadInt32(&count) == 0 {
		t.Error("Stop should flush events")
	}
}

func TestSIEMForwarder_SplunkAuth(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Provider = SIEMProviderSplunk
	cfg.Endpoint = srv.URL
	cfg.APIKey = "hec-token"
	cfg.MaxRetries = 1
	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "t", Result: "ok"})
	f.flush()
	if auth != "Splunk hec-token" {
		t.Errorf("got %s", auth)
	}
}

func TestSIEMForwarder_DatadogAuth(t *testing.T) {
	var key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("DD-API-KEY")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Provider = SIEMProviderDatadog
	cfg.Endpoint = srv.URL
	cfg.APIKey = "dd-key"
	cfg.MaxRetries = 1
	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "t", Result: "ok"})
	f.flush()
	if key != "dd-key" {
		t.Errorf("got %s", key)
	}
}

func TestSIEMForwarder_EmptyFlush(t *testing.T) {
	f := NewSIEMForwarder(DefaultSIEMConfig())
	f.flush() // should not panic
}

func splitNL(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
