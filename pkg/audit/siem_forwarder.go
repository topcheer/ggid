package audit

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SIEMProvider identifies the SIEM destination type.
type SIEMProvider string

const (
	SIEMProviderSplunk      SIEMProvider = "splunk"
	SIEMProviderDatadog     SIEMProvider = "datadog"
	SIEMProviderElasticsearch SIEMProvider = "elasticsearch"
	SIEMProviderGeneric     SIEMProvider = "generic"
)

// SIEMConfig configures the SIEM forwarder.
type SIEMConfig struct {
	Provider    SIEMProvider
	Endpoint    string        // HEC URL, Datadog API endpoint, Elasticsearch _bulk URL
	APIKey      string        // Splunk HEC token, Datadog API key, Elasticsearch auth
	IndexName   string        // Splunk index, Datadog service name, Elasticsearch index
	BatchSize   int           // events per batch (default: 100)
	FlushInterval time.Duration // how often to flush (default: 5s)
	MaxRetries  int           // retry count on failure (default: 3)
	Timeout     time.Duration // HTTP client timeout (default: 10s)
}

// DefaultSIEMConfig returns a config with sensible defaults.
func DefaultSIEMConfig() SIEMConfig {
	return SIEMConfig{
		Provider:      SIEMProviderGeneric,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
	}
}

// SIEMForwarder subscribes to audit events and forwards them to an external SIEM.
// It batches events for efficiency and retries on failure.
type SIEMForwarder struct {
	config   SIEMConfig
	client   *http.Client
	logger   *slog.Logger

	mu       sync.Mutex
	buffer   []Event
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
}

// NewSIEMForwarder creates a new SIEM forwarder.
func NewSIEMForwarder(cfg SIEMConfig) *SIEMForwarder {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SIEMForwarder{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		logger: slog.Default(),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// SetCAPool configures a custom CA certificate pool for TLS connections to the SIEM endpoint.
func (f *SIEMForwarder) SetCAPool(pool *x509.CertPool) {
	f.client = &http.Client{
		Timeout: f.config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}
}

// Forward adds an event to the buffer for batch forwarding.
func (f *SIEMForwarder) Forward(event Event) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.buffer = append(f.buffer, event)

	if len(f.buffer) >= f.config.BatchSize {
		go f.flush()
	}
}

// Start begins the periodic flush goroutine.
// Call Stop() to cleanly shut down.
func (f *SIEMForwarder) Start(ctx context.Context) {
	go func() {
		defer close(f.doneCh)
		ticker := time.NewTicker(f.config.FlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				f.flush()
			case <-f.stopCh:
				f.flush() // final flush
				return
			case <-ctx.Done():
				f.flush()
				return
			}
		}
	}()
}

// Stop gracefully shuts down the forwarder, flushing remaining events.
// Safe to call multiple times.
func (f *SIEMForwarder) Stop() {
	f.stopOnce.Do(func() {
		close(f.stopCh)
	})
	select {
	case <-f.doneCh:
	case <-time.After(5 * time.Second):
	}
}

// flush sends all buffered events to the configured SIEM endpoint.
func (f *SIEMForwarder) flush() {
	f.mu.Lock()
	if len(f.buffer) == 0 {
		f.mu.Unlock()
		return
	}
	batch := f.buffer
	f.buffer = make([]Event, 0, f.config.BatchSize)
	f.mu.Unlock()

	payload, err := f.formatBatch(batch)
	if err != nil {
		f.logger.Error("SIEM forwarder: format batch failed", "error", err, "events", len(batch))
		return
	}

	for attempt := 0; attempt < f.config.MaxRetries; attempt++ {
		if err := f.send(context.Background(), payload); err != nil {
			f.logger.Warn("SIEM forwarder: send failed, retrying",
				"attempt", attempt+1, "error", err)
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		f.logger.Info("SIEM forwarder: batch sent", "events", len(batch), "provider", f.config.Provider)
		return
	}

	f.logger.Error("SIEM forwarder: exhausted retries, dropping batch",
		"events", len(batch), "provider", f.config.Provider)
}

// formatBatch converts events to the provider-specific format.
func (f *SIEMForwarder) formatBatch(events []Event) ([]byte, error) {
	switch f.config.Provider {
	case SIEMProviderSplunk:
		return f.formatSplunk(events)
	case SIEMProviderDatadog:
		return f.formatDatadog(events)
	case SIEMProviderElasticsearch:
		return f.formatElasticsearch(events)
	default:
		return f.formatGeneric(events)
	}
}

// formatSplunk formats events as Splunk HEC JSON lines.
func (f *SIEMForwarder) formatSplunk(events []Event) ([]byte, error) {
	var buf bytes.Buffer
	for _, e := range events {
		entry := map[string]any{
			"time":       e.CreatedAt.Unix(),
			"host":       e.IPAddress,
			"source":     "ggid",
			"sourcetype": "ggid:audit",
			"index":      f.config.IndexName,
			"event":      e,
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// formatDatadog formats events as Datadog Logs API payload.
func (f *SIEMForwarder) formatDatadog(events []Event) ([]byte, error) {
	logs := make([]map[string]any, 0, len(events))
	for _, e := range events {
		logs = append(logs, map[string]any{
			"message":  fmt.Sprintf("%s %s %s", e.Action, e.ResourceType, e.Result),
			"ddsource": "ggid",
			"service":  f.config.IndexName,
			"timestamp": e.CreatedAt.UnixMilli(),
			"ddtags":   fmt.Sprintf("tenant:%s,actor:%s,result:%s", e.TenantID, e.ActorType, e.Result),
			"audit":    e,
		})
	}
	return json.Marshal(logs)
}

// formatElasticsearch formats events as Elasticsearch bulk index operations.
func (f *SIEMForwarder) formatElasticsearch(events []Event) ([]byte, error) {
	var buf bytes.Buffer
	for _, e := range events {
		// Action line
		action := map[string]any{
			"index": map[string]any{
				"_index": f.config.IndexName,
				"_id":    e.ID.String(),
			},
		}
		actionData, err := json.Marshal(action)
		if err != nil {
			return nil, err
		}
		buf.Write(actionData)
		buf.WriteByte('\n')

		// Source line
		sourceData, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		buf.Write(sourceData)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// formatGeneric formats events as a JSON array.
func (f *SIEMForwarder) formatGeneric(events []Event) ([]byte, error) {
	return json.Marshal(events)
}

// send posts the formatted payload to the SIEM endpoint with provider-specific auth.
func (f *SIEMForwarder) send(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.config.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	switch f.config.Provider {
	case SIEMProviderSplunk:
		req.Header.Set("Authorization", "Splunk "+f.config.APIKey)
	case SIEMProviderDatadog:
		req.Header.Set("DD-API-KEY", f.config.APIKey)
		req.Header.Set("Content-Type", "application/json")
	case SIEMProviderElasticsearch:
		req.Header.Set("Authorization", "Bearer "+f.config.APIKey)
		req.Header.Set("Content-Type", "application/x-ndjson")
	default:
		req.Header.Set("Authorization", "Bearer "+f.config.APIKey)
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("send to SIEM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SIEM returned HTTP %d", resp.StatusCode)
	}

	return nil
}
