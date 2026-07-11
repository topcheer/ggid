# SIEM Integration Guide

> Step-by-step guide to forward GGID audit events to Splunk, Datadog, and Elasticsearch.

---

## Prerequisites

- GGID running with NATS JetStream (`NATS_URL=nats://localhost:4222`)
- SIEM destination accessible from GGID network

---

## Splunk HEC

### 1. Create HEC Token

Splunk → Settings → Data Inputs → HTTP Event Collector → New Token.

### 2. Configure GGID

```bash
export SIEM_SPLUNK_HEC_URL=https://splunk.internal:8088/services/collector
export SIEM_SPLUNK_TOKEN=your-hec-token-here
cd deploy && docker compose up -d siem-connector
```

### 3. Verify

```bash
# Check connector is consuming
docker logs ggid-siem-connector | tail -5
# → "forwarded 42 events to Splunk"

# Search in Splunk
index=iam sourcetype=ggid:audit | head 10
```

---

## Datadog Logs

### 1. Get API Key

Datadog → Integrations → APIs → New API Key.

### 2. Configure GGID

```bash
export SIEM_DATADOG_API_KEY=dd-api-key-here
export SIEM_DATADOG_SITE=datadoghq.com
cd deploy && docker compose up -d siem-connector
```

### 3. Verify

```bash
docker logs ggid-siem-connector | tail -5
# → "forwarded 42 events to Datadog"
```

Datadog query: `@action:user.* | group by action | count()`

---

## Elasticsearch

### 1. Create Index Template

```bash
curl -X PUT https://es.internal:9200/_index_template/ggid-audit \
  -H 'Content-Type: application/json' \
  -d '{
    "index_patterns": ["ggid-audit-*"],
    "template": {
      "mappings": {
        "properties": {
          "@timestamp": {"type": "date"},
          "event.action": {"type": "keyword"},
          "user.id": {"type": "keyword"},
          "source.ip": {"type": "ip"}
        }
      }
    }
  }'
```

### 2. Configure GGID

```bash
export SIEM_ES_URL=https://es.internal:9200
export SIEM_ES_INDEX=ggid-audit
cd deploy && docker compose up -d siem-connector
```

### 3. Verify

```bash
curl https://es.internal:9200/ggid-audit-*/_count
# → {"count": 15423}
```

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| No events in SIEM | Connector not started | `docker ps`, check logs |
| 401 from Splunk | Wrong HEC token | Regenerate token, update env |
| 403 from Datadog | Invalid API key | Verify key in Datadog dashboard |
| Events delayed | NATS consumer lag | Check `NATS_URL`, increase batch size |

---

*See: [SIEM Connector Design](../research/siem-connector-design.md) | [Audit Compliance](audit-compliance.md) | [Event-Driven Architecture](../architecture/event-driven.md)*

*Last updated: 2025-07-11*
