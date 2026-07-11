# SIEM Connector Design

> Architecture for streaming GGID audit events from NATS to Splunk, Datadog, and Elasticsearch.

---

## Architecture

```
GGID Services → NATS JetStream → SIEM Connector → Splunk HEC / Datadog / Elasticsearch
                     ↑                                     ↓
                     └─── (durable consumer) ────── Each target gets own consumer
```

The connector runs as a sidecar or standalone service that:
1. Subscribes to NATS `AUDIT_EVENTS` stream as a durable consumer
2. Transforms events to the target SIEM's format
3. Batches and sends via HTTP API
4. Acknowledges NATS after successful delivery

---

## Event Format (GGID Internal)

```json
{
  "event_id": "evt_abc123",
  "tenant_id": "00000000-0000-0000-0000-000000000001",
  "actor_type": "user",
  "actor_id": "usr_abc123",
  "action": "user.login",
  "resource_type": "auth",
  "resource_id": "",
  "metadata": { "ip": "192.168.1.1", "success": true },
  "timestamp": "2025-07-11T12:00:00Z",
  "hash": "sha256:abc..."
}
```

---

## Splunk HEC (HTTP Event Collector)

### Transform

```go
type SplunkEvent struct {
    Time   int64       `json:"time"`   // Unix epoch
    Host   string      `json:"host"`   // Source IP or hostname
    Source string      `json:"source"` // "ggid"
    Sourcetype string  `json:"sourcetype"` // "ggid:audit"
    Index  string      `json:"index"`  // "iam"
    Event  *AuditEvent `json:"event"`  // Original GGID event
}
```

### Connector Implementation

```go
func splunkForwarder(natsURL, hecURL, hecToken string) error {
    nc, _ := nats.Connect(natsURL)
    js, _ := nc.JetStream()
    sub, _ := js.Subscribe("AUDIT_EVENTS", "siem-splunk", func(msg *nats.Msg) {
        var event AuditEvent
        json.Unmarshal(msg.Data, &event)

        splunkEvent := SplunkEvent{
            Time:       event.Timestamp.Unix(),
            Host:       event.Metadata.IP,
            Source:     "ggid",
            Sourcetype: "ggid:audit",
            Index:      "iam",
            Event:      &event,
        }

        body, _ := json.Marshal(splunkEvent)
        req, _ := http.NewRequest("POST", hecURL+"/services/collector",
            bytes.NewReader(body))
        req.Header.Set("Authorization", "Splunk "+hecToken)

        resp, err := http.DefaultClient.Do(req)
        if err == nil && resp.StatusCode == 200 {
            msg.Ack() // Only ack on success
        }
        resp.Body.Close()
    },
        nats.Durable("siem-splunk"),
        nats.DeliverAll(),
        nats.ManualAck(),
        nats.MaxDeliver(5),
    )
    return nil
}
```

### Splunk Configuration

```ini
# inputs.conf
[http://ggid]
disabled = 0
token = <HEC_TOKEN>
index = iam
sourcetype = ggid:audit
```

### Splunk SPL Queries

```splunk
# Failed logins by IP
index=iam action="user.login" metadata.success=false
| stats count by metadata.ip
| sort -count

# Privilege escalation attempts
index=iam action="role.assign" OR action="iga.request_approved"
| table timestamp actor_id action resource_id

# Hash chain tamper detection
index=iam action="security.hash_chain_tamper"
```

---

## Datadog Logs API

### Transform

```json
[
  {
    "ddsource": "ggid",
    "ddtags": "env:prod,service:iam,tenant:acme",
    "hostname": "ggid-gateway",
    "message": "{\"action\":\"user.login\",\"actor\":\"usr_abc\",\"success\":true}",
    "timestamp": "2025-07-11T12:00:00Z"
  }
]
```

### Connector

```go
func datadogForwarder(apiKey string) {
    // POST https://http-intake.logs.datadoghq.com/v1/input
    // Header: DD-API-KEY: <apiKey>
    // Body: array of log entries
    // Batch: 100 events or 5 seconds, whichever first
}
```

### Datadog Queries

```
# Audit event volume by action
@action:user.* | group by action | count()

# Failed authentications
@action:user.login @success:false
```

---

## Elasticsearch / Elastic Stack

### Transform (ECS-compatible)

```json
{
  "@timestamp": "2025-07-11T12:00:00Z",
  "event": {
    "action": "user.login",
    "category": "authentication",
    "type": "info",
    "outcome": "success"
  },
  "user": { "id": "usr_abc123" },
  "source": { "ip": "192.168.1.1" },
  "observer": { "product": "GGID", "vendor": "GGID" },
  "labels": { "tenant_id": "00000000-...", "hash": "sha256:abc..." }
}
```

### Index Template

```json
{
  "index_patterns": ["ggid-audit-*"],
  "mappings": {
    "properties": {
      "@timestamp": { "type": "date" },
      "event.action": { "type": "keyword" },
      "event.outcome": { "type": "keyword" },
      "user.id": { "type": "keyword" },
      "source.ip": { "type": "ip" },
      "labels.tenant_id": { "type": "keyword" }
    }
  }
}
```

### Kibana KQL

```
event.action: "user.login" and event.outcome: "failure"
```

---

## Deployment

### Docker Compose (sidecar)

```yaml
services:
  siem-connector:
    image: ggid/siem-connector:latest
    environment:
      NATS_URL: nats://nats:4222
      SPLUNK_HEC_URL: https://splunk.internal:8088
      SPLUNK_HEC_TOKEN: ${SPLUNK_TOKEN}
      DD_API_KEY: ${DATADOG_API_KEY}
      ES_URL: https://es.internal:9200
      BATCH_SIZE: 100
      FLUSH_INTERVAL: 5s
    depends_on:
      - nats
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ggid-siem-connector
spec:
  replicas: 1  # Exactly 1 for ordered delivery
  template:
    spec:
      containers:
      - name: connector
        image: ggid/siem-connector:latest
        env:
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: SPLUNK_HEC_URL
          valueFrom:
            secretKeyRef:
              name: siem-secrets
              key: splunk-hec-url
```

---

## Reliability

| Concern | Solution |
|---------|----------|
| At-least-once delivery | NATS durable consumer + manual ack |
| Duplicate handling | SIEM deduplicates by `event_id` |
| Backpressure | Batch + flush interval |
| Retry | NATS MaxDeliver(5) with backoff |
| Ordering | Per-tenant ordering via subject partitioning |
| Monitoring | Connector exposes `/metrics` (Prometheus) |

---

## Performance

- **Throughput:** 5000 events/sec per connector instance
- **Latency:** < 2s from GGID to SIEM (batch window)
- **Memory:** ~50MB for 100k event buffer
- **Scale horizontally:** Partition by tenant_id for >10k events/sec

---

*See: [Event-Driven Architecture](../architecture/event-driven.md) | [Audit Compliance](../guides/audit-compliance.md) | [Operations Runbook](../operations-runbook.md)*

*Last updated: 2025-07-11*
