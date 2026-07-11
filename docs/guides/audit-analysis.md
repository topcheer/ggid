# Audit Log Analysis Guide

> Query, filter, export, and forward GGID audit logs for security analysis and compliance.

---

## Query Audit Events

### Basic Query

```bash
curl -s "http://localhost:8080/api/v1/audit/events?limit=50" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

### Filter by Action

```bash
# All failed logins
curl -s "http://localhost:8080/api/v1/audit/events?action=user.login&limit=100" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  | jq '.events[] | select(.metadata.success == false)'
```

### Filter by Time Range

```bash
curl -s "http://localhost:8080/api/v1/audit/events?start_time=2025-07-01T00:00:00Z&end_time=2025-07-31T23:59:59Z" \
  -H "Authorization: Bearer $JWT" | jq '.total'
```

### Filter by Actor

```bash
curl -s "http://localhost:8080/api/v1/audit/events?actor_id=usr_abc123" \
  -H "Authorization: Bearer $JWT" | jq '.events[] | {action, timestamp}'
```

---

## Verify Hash Chain

```bash
curl -s http://localhost:8080/api/v1/audit/verify \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
# {"verified":true,"total_events":15423,"tampered_events":[]}
```

---

## Export for Compliance

### PCI-DSS Quarterly Report

```bash
curl -s "http://localhost:8080/api/v1/audit/events?resource_type=card_data&start_time=2025-04-01T00:00:00Z&limit=5000" \
  -H "Authorization: Bearer $JWT" | jq '.events' > pci-q2.json
```

### SOC2 Change History

```bash
curl -s "http://localhost:8080/api/v1/audit/events?action=role.update&action=user.delete&limit=1000" \
  -H "Authorization: Bearer $JWT" | jq '.events' > soc2-changes.json
```

---

## SIEM Forwarding

See [SIEM Integration Guide](siem-integration.md) for Splunk/Datadog/Elasticsearch setup.

Quick start:
```bash
export SIEM_SPLUNK_HEC_URL=https://splunk.internal:8088/services/collector
export SIEM_SPLUNK_TOKEN=your-token
cd deploy && docker compose up -d siem-connector
```

---

## Common Analysis Patterns

| Question | Query |
|----------|-------|
| Who accessed user data? | `?resource_type=users&action=user.read` |
| Failed admin logins? | `?action=user.login&actor_type=admin` → filter `success=false` |
| Role changes this week? | `?action=role.assign&start_time=...` |
| SCIM operations? | `?action=scim.*` |

---

*See: [Audit Compliance](audit-compliance.md) | [SIEM Integration](siem-integration.md) | [Event-Driven Architecture](../architecture/event-driven.md)*

*Last updated: 2025-07-11*
