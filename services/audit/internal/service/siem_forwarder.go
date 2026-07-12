package service

import (
	"crypto/tls"
	"fmt"
	"bytes"
	"net/http"
	"sync"
	"time"
)

type SIEMType string

const (
	SIEMSplunk     SIEMType = "splunk"
	SIEMQRadar     SIEMType = "qradar"
	SIEMElastic    SIEMType = "elastic"
	SIEMSyslog     SIEMType = "syslog"
)

type SIEMDestination struct {
	Name      string    `json:"name"`
	Type      SIEMType  `json:"type"`
	URL       string    `json:"url"`
	AuthToken string    `json:"auth_token"`
	BatchSize int       `json:"batch_size"`
	Enabled   bool      `json:"enabled"`
}

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

type SIEMForwarder struct {
	mu            sync.RWMutex
	destinations  map[string]*SIEMDestination
	circuitStates map[string]CircuitState
	failCounts    map[string]int
	client        *http.Client
	stats         map[string]*ForwardStats
}

type ForwardStats struct {
	Forwarded int `json:"forwarded"`
	Failed    int `json:"failed"`
	Retried   int `json:"retried"`
}

func NewSIEMForwarder() *SIEMForwarder {
	return &SIEMForwarder{
		destinations:  make(map[string]*SIEMDestination),
		circuitStates: make(map[string]CircuitState),
		failCounts:    make(map[string]int),
		stats:         make(map[string]*ForwardStats),
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
		},
	}
}

func (f *SIEMForwarder) AddDestination(dest SIEMDestination) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.destinations[dest.Name] = &dest
	f.circuitStates[dest.Name] = CircuitClosed
	f.failCounts[dest.Name] = 0
	f.stats[dest.Name] = &ForwardStats{}
}

func (f *SIEMForwarder) ForwardEvent(event []byte, destName string) error {
	f.mu.Lock()
	dest, ok := f.destinations[destName]
	if !ok || !dest.Enabled {
		f.mu.Unlock()
		return fmt.Errorf("destination not found or disabled")
	}
	state := f.circuitStates[destName]
	if state == CircuitOpen {
		f.mu.Unlock()
		return fmt.Errorf("circuit breaker open for %s", destName)
	}
	f.mu.Unlock()

	return f.forwardWithRetry(destName, dest, event, 3)
}

func (f *SIEMForwarder) BatchForward(events [][]byte, destName string) error {
	f.mu.Lock()
	dest, ok := f.destinations[destName]
	if !ok || !dest.Enabled {
		f.mu.Unlock()
		return fmt.Errorf("destination not found or disabled")
	}
	f.mu.Unlock()

	batchSize := dest.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		for _, evt := range events[i:end] {
			if err := f.forwardWithRetry(destName, dest, evt, 2); err != nil {
				continue // skip failed events in batch
			}
		}
	}
	return nil
}

func (f *SIEMForwarder) forwardWithRetry(destName string, dest *SIEMDestination, payload []byte, maxRetries int) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
			f.mu.Lock()
			f.stats[destName].Retried++
			f.mu.Unlock()
		}
		req, err := http.NewRequest(http.MethodPost, dest.URL, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if dest.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+dest.AuthToken)
		}
		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			f.mu.Lock()
			f.stats[destName].Forwarded++
			f.failCounts[destName] = 0
			f.circuitStates[destName] = CircuitClosed
			f.mu.Unlock()
			return nil
		}
		lastErr = fmt.Errorf("SIEM returned %d", resp.StatusCode)
	}

	f.mu.Lock()
	f.failCounts[destName]++
	f.stats[destName].Failed++
	if f.failCounts[destName] >= 5 {
		f.circuitStates[destName] = CircuitOpen
	}
	f.mu.Unlock()
	return lastErr
}

func (f *SIEMForwarder) HealthCheck(destName string) bool {
	f.mu.RLock()
	dest, ok := f.destinations[destName]
	f.mu.RUnlock()
	if !ok {
		return false
	}
	resp, err := f.client.Get(dest.URL + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 400
}

func (f *SIEMForwarder) GetStats(destName string) *ForwardStats {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.stats[destName]
}

