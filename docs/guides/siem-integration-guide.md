# SIEM Integration Guide

This guide covers GGID audit event schema, Splunk HEC integration, Elastic Beats + Logstash, Datadog logs API, QRadar syslog, filter rules, batch forwarding, TLS mutual auth, circuit breaker, retry queue, forward stats, and dashboard templates.

## Audit Event Schema

### JSON Format (Native)

```json
{
  "timestamp": "2026-07-12T10:00:00.123Z",
  "event_id": "evt-uuid-1234",
  "event_type": "user.login",
  "severity": "info",
  "tenant_id": "tenant-uuid-5678",
  "user_id": "user-uuid-9012",
  "user_name": "jdoe@example.com",
  "action": "login",
  "resource": "auth-service",
  "ip": "192.168.1.50",
  "user_agent": "Mozilla/5.0",
  "result": "success",
  "details": {
    "method": "password+mfa",
    "mfa_type": "totp"
  },
  "hash_chain": {
    "sequence": 12345,
    "prev_hash": "abc123...",
    "block_hash": "def456..."
  }
}
```

### CEF Format (Common Event Format)

```
CEF:0|GGID|Identity Platform|1.0|100|User Login|3|suser=jdoe@example.com act=login dst=auth-service src=192.168.1.50 rt=Jul 12 2026 10:00:00.123Z outcome=success cs1Label=TenantID cs1=tenant-uuid-5678 cs2Label=Method cs2=password+mfa
```

### LEEF Format (Log Event Extended Format)

```
LEEF:1.0|GGID|Identity Platform|1.0|100|src=192.168.1.50 suser=jdoe@example.com act=login resource=auth-service outcome=success sev=3
```

## Splunk HEC Integration

### Configuration

```yaml
siem:
  splunk:
    enabled: true
    hec_url: "https://splunk.example.com:8088/services/collector"
    hec_token: "<splunk-hec-token>"
    index: "gcid-audit"
    sourcetype: "gcid:audit"
    batch_size: 100
    flush_interval: 5s
    tls:
      verify_server: true
      ca_cert: "/etc/ggid/siem/splunk-ca.pem"
```

### Forwarder Implementation

```go
type SplunkForwarder struct {
    hecURL   string
    token    string
    index    string
    sourceType string
    client   *http.Client
    batch    []map[string]interface{}
    batchMu  sync.Mutex
    flushTimer *time.Ticker
}

func (f *SplunkForwarder) Forward(event *AuditEvent) error {
    splunkEvent := map[string]interface{}{
        "time": float64(event.Timestamp.UnixNano()) / 1e9,
        "host": event.IP,
        "source": event.Resource,
        "sourcetype": f.sourceType,
        "index": f.index,
        "event": map[string]interface{}{
            "event_id":   event.ID,
            "event_type": event.Type,
            "severity":   event.Severity,
            "tenant_id":  event.TenantID,
            "user_id":    event.UserID,
            "user_name":  event.UserName,
            "action":     event.Action,
            "result":     event.Result,
            "ip":         event.IP,
            "details":    event.Details,
        },
    }
    
    f.batchMu.Lock()
    f.batch = append(f.batch, splunkEvent)
    if len(f.batch) >= f.config.BatchSize {
        return f.flush()
    }
    f.batchMu.Unlock()
    return nil
}

func (f *SplunkForwarder) flush() error {
    if len(f.batch) == 0 {
        return nil
    }
    
    body, _ := json.Marshal(f.batch)
    req, _ := http.NewRequest("POST", f.hecURL, bytes.NewReader(body))
    req.Header.Set("Authorization", "Splunk "+f.token)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := f.client.Do(req)
    if err != nil {
        return f.retry(f.batch, err)
    }
    
    if resp.StatusCode >= 400 {
        return f.retry(f.batch, fmt.Errorf("splunk HEC returned %d", resp.StatusCode))
    }
    
    f.batch = f.batch[:0]  // Clear batch
    return nil
}
```

## Elastic Beats + Logstash

### Filebeat Configuration

```yaml
# /etc/filebeat/filebeat.yml
filebeat.inputs:
  - type: log
    paths:
      - /var/log/ggid/audit.log
    json.keys_under_root: true
    json.add_error_key: true
    fields:
      source: ggid
      environment: production
    fields_under_root: true

output.logstash:
  hosts: ["logstash.example.com:5044"]
  ssl.certificate_authorities: ["/etc/filebeat/ca.pem"]
  ssl.certificate: "/etc/filebeat/client.pem"
  ssl.key: "/etc/filebeat/client-key.pem"
```

### Logstash Pipeline

```
# /etc/logstash/conf.d/ggid-audit.conf
input {
  beats {
    port => 5044
    ssl => true
    ssl_certificate => "/etc/logstash/server.pem"
    ssl_key => "/etc/logstash/server-key.pem"
    ssl_ca => "/etc/logstash/ca.pem"
  }
}

filter {
  if [source] == "gcid" {
    date {
      match => ["timestamp", "ISO8601"]
    }
    mutate {
      add_field => { "application" => "gcid" }
    }
    geoip {
      source => "ip"
      target => "geoip"
    }
  }
}

output {
  elasticsearch {
    hosts => ["https://es.example.com:9200"]
    index => "gcid-audit-%{+YYYY.MM.dd}"
    user => "gcid_logstash"
    password => "${LOGSTASH_PASSWORD}"
    ssl => true
    cacert => "/etc/logstash/es-ca.pem"
  }
}
```

## Datadog Logs API

### Configuration

```yaml
siem:
  datadog:
    enabled: true
    api_url: "https://http-intake.logs.datadoghq.com/api/v2/logs"
    api_key: "<datadog-api-key>"
    service: "gcid"
    env: "production"
    batch_size: 50
    flush_interval: 5s
```

### Forwarder

```go
func (f *DatadogForwarder) Forward(event *AuditEvent) error {
    ddEvent := map[string]interface{}{
        "ddsource": "gcid",
        "ddtags":   fmt.Sprintf("service:gcid,env:production,tenant:%s,severity:%s",
                        event.TenantID, event.Severity),
        "hostname": event.IP,
        "message":  event,
        "service":  "gcid",
    }
    // Batch and send to Datadog intake API
    return f.batchAndSend(ddEvent)
}
```

## QRadar Syslog

### Configuration

```yaml
siem:
  qradar:
    enabled: true
    syslog_host: "qradar.example.com"
    syslog_port: 514
    protocol: "tcp"
    format: "cef"
    facility: "local0"
    severity: "info"
```

### Syslog Forwarder

```go
func (f *SyslogForwarder) Forward(event *AuditEvent) error {
    cefEvent := toCEF(event)
    syslogMsg := fmt.Sprintf("<%d>%s %s gcid[%d]: %s",
        f.priority(),
        time.Now().Format("Jan 2 15:04:05"),
        f.hostname,
        f.pid,
        cefEvent,
    )
    
    conn, err := net.Dial(f.protocol, f.addr)
    if err != nil {
        return f.retry(event, err)
    }
    defer conn.Close()
    
    _, err = fmt.Fprintln(conn, syslogMsg)
    return err
}
```

## Filter Rules

### Forward Only Relevant Events

```yaml
siem:
  filters:
    include:
      severity: ["warn", "error", "critical"]
      event_types:
        - "user.login"
        - "user.login_failed"
        - "user.logout"
        - "admin.*"
        - "security.*"
        - "mfa.*"
        - "policy.*"
      tenants: ["*"]  # or specific tenant IDs
    exclude:
      event_types:
        - "user.heartbeat"
        - "system.health_check"
      ip_ranges:
        - "127.0.0.0/8"
        - "10.0.0.0/8"
```

### Filter Implementation

```go
func shouldForward(event *AuditEvent, config *FilterConfig) bool {
    // Check severity
    if !contains(config.Include.Severity, event.Severity) {
        return false
    }
    
    // Check event type
    if !matchesPatterns(config.Include.EventTypes, event.Type) {
        return false
    }
    
    // Check exclusions
    if contains(config.Exclude.EventTypes, event.Type) {
        return false
    }
    
    if inCIDRRange(config.Exclude.IPRanges, event.IP) {
        return false
    }
    
    return true
}
```

## Batch Forwarding

### Configuration

```yaml
siem:
  batching:
    enabled: true
    max_batch_size: 100
    flush_interval: 5s
    max_queue_size: 10000
    overflow_action: "drop_oldest"  # or "block"
```

### Batch Queue

```go
type BatchQueue struct {
    queue    chan *AuditEvent
    batch    []*AuditEvent
    mu       sync.Mutex
    config   BatchConfig
}

func (q *BatchQueue) Add(event *AuditEvent) error {
    select {
    case q.queue <- event:
        return nil
    default:
        if q.config.OverflowAction == "drop_oldest" {
            <-q.queue  // Drop oldest
            q.queue <- event
            return nil
        }
        return ErrQueueFull
    }
}

func (q *BatchQueue) Flush() error {
    q.mu.Lock()
    defer q.mu.Unlock()
    
    if len(q.batch) == 0 {
        return nil
    }
    
    err := q.forward(q.batch)
    if err != nil {
        q.moveToRetry(q.batch)
    }
    q.batch = q.batch[:0]
    return err
}
```

## TLS Mutual Authentication

```yaml
siem:
  tls:
    enabled: true
    mutual_auth: true
    client_cert: "/etc/ggid/siem/client.pem"
    client_key: "/etc/ggid/siem/client-key.pem"
    ca_cert: "/etc/ggid/siem/ca.pem"
    min_version: "TLS1.2"
    verify_server: true
```

```go
func newTLSClient(config *TLSConfig) *http.Client {
    cert, _ := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
    caCert, _ := os.ReadFile(config.CACert)
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)
    
    return &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                Certificates: []tls.Certificate{cert},
                RootCAs:      caPool,
                MinVersion:   tls.VersionTLS12,
            },
        },
        Timeout: 30 * time.Second,
    }
}
```

## Circuit Breaker

```yaml
siem:
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    reset_timeout: 30s
    half_open_max: 3
```

```go
type CircuitBreaker struct {
    failures     int
    state        string  // "closed", "open", "half_open"
    lastFailure  time.Time
    config       CircuitConfig
    mu           sync.Mutex
}

func (cb *CircuitBreaker) Allow() bool {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    switch cb.state {
    case "closed":
        return true
    case "open":
        if time.Since(cb.lastFailure) > cb.config.ResetTimeout {
            cb.state = "half_open"
            return true
        }
        return false
    case "half_open":
        return true
    }
    return false
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failures = 0
    cb.state = "closed"
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    cb.failures++
    cb.lastFailure = time.Now()
    if cb.failures >= cb.config.FailureThreshold {
        cb.state = "open"
    }
}
```

## Retry Queue

```yaml
siem:
  retry:
    max_attempts: 3
    backoff: "exponential"
    initial_delay: 1s
    max_delay: 30s
    queue_size: 50000
    persistent: true  # Survive restarts
    persistence_path: "/var/lib/ggid/siem-retry"
```

## Forward Statistics

```yaml
siem:
  stats:
    enabled: true
    metrics:
      - "forwarded_total"
      - "forwarded_success"
      - "forwarded_failed"
      - "batch_size_avg"
      - "latency_ms"
      - "circuit_breaker_state"
      - "queue_depth"
      - "retry_count"
    prometheus:
      enabled: true
      namespace: "gcid"
      subsystem: "siem"
```

## Dashboard Templates

### Splunk Dashboard

```json
{
  "dashboard": {
    "title": "GGID Security Events",
    "panels": [
      {
        "title": "Login Failures by IP (24h)",
        "query": "index=gcid-audit event_type=user.login_failed | stats count by ip | sort -count"
      },
      {
        "title": "Admin Actions (1h)",
        "query": "index=gcid-audit event_type=admin.* | table timestamp user_name action resource result"
      },
      {
        "title": "MFA Failures",
        "query": "index=gcid-audit event_type=mfa.* result=failed | timechart count"
      }
    ]
  }
}
```

## GGID SIEM Implementation

### Configuration

```yaml
siem:
  enabled: true
  forwarder: "splunk"  # or "elastic", "datadog", "qradar", "syslog"
  format: "json"  # or "cef", "leef"
  batching:
    max_batch_size: 100
    flush_interval: 5s
  filters:
    include:
      severity: ["warn", "error", "critical"]
  tls:
    enabled: true
    mutual_auth: true
  circuit_breaker:
    enabled: true
    failure_threshold: 5
    reset_timeout: 30s
  retry:
    max_attempts: 3
    backoff: "exponential"
  stats:
    enabled: true
    prometheus: true
```

## Best Practices

1. **Filter at source** — Don't forward every event, only relevant ones
2. **Batch for efficiency** — Reduce API calls with batching
3. **Use circuit breaker** — Don't let SIEM downtime block GGID
4. **Retry with backoff** — Exponential backoff prevents overwhelming SIEM
5. **Mutual TLS** — Authenticate both directions for security
6. **Monitor forward stats** — Track success rate, latency, queue depth
7. **Test failover** — Verify circuit breaker and retry work
8. **Use CEF for multi-SIEM** — Common format works with many SIEMs
9. **Include hash chain** — Forward chain metadata for tamper detection
10. **Document event schema** — Help SIEM team build dashboards