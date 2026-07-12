# Data Loss Prevention Strategy

This guide covers DLP architecture, policy engine, data discovery, leak channels, incident response, and GGID's DLP integration.

## DLP Architecture

### Layers

| Layer | Scope | Examples |
|---|---|---|
| Endpoint | Devices | USB copy, clipboard, print |
| Network | Traffic | Email, web upload, API calls |
| Cloud | Cloud services | SaaS upload, cloud storage |
| Data center | Internal | DB export, log files, backups |

## Policy Engine

### Rule Structure

```yaml
dlp:
  policies:
    - name: "prevent-ssn-leak"
      description: "Block SSN in any outbound channel"
      conditions:
        - pattern: "\\b\\d{3}-\\d{2}-\\d{4}\\b"
          data_classification: "restricted"
      actions:
        - block
        - alert_security
        - log_event
    - name: "prevent-pii-email"
      conditions:
        - pattern: "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+"
          count: "> 50"  # More than 50 emails = bulk
          channel: "email"
      actions:
        - block
        - require_approval
    - name: "prevent-large-export"
      conditions:
        - operation: "data_export"
          records: "> 10000"
      actions:
        - block
        - alert_admin
        - require_approval
```

### Actions

| Action | Description |
|---|---|
| block | Prevent the operation |
| alert_security | Notify security team |
| alert_admin | Notify admin |
| log_event | Record in audit |
| require_approval | Require manual approval |
| encrypt | Encrypt before allowing |
| quarantine | Isolate the data |
| redact | Remove sensitive content |

## Data Discovery

### Scan + Classify + Inventory

```go
func scanForSensitiveData(data string) []DataFinding {
    var findings []DataFinding
    patterns := map[string]*regexp.Regexp{
        "ssn":          regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
        "credit_card":  regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`),
        "email":        regexp.MustCompile(`\b[\w.]+@[\w.]+\.\w+\b`),
        "api_key":      regexp.MustCompile(`\b(ggid_[a-zA-Z0-9]{32,})\b`),
        "private_key":  regexp.MustCompile(`-----BEGIN.*PRIVATE KEY-----`),
    }
    for name, pattern := range patterns {
        if pattern.MatchString(data) {
            findings = append(findings, DataFinding{
                Type: name, Count: len(pattern.FindAllString(data, -1)),
            })
        }
    }
    return findings
}
```

## Leak Channels

| Channel | Detection | Prevention |
|---|---|---|
| Email | Pattern scan | Block + redact |
| Web upload | Content inspection | Block + alert |
| USB | Endpoint agent | Block + log |
| Cloud upload | API proxy | Block + require approval |
| API response | Response filter | Redact PII fields |
| Print | Endpoint agent | Watermark + log |
| Screenshot | Endpoint agent | Detect + log |

## GGID DLP Integration

```yaml
dlp:
  enabled: true
  channels:
    api_response:
      scan: true
      redact_pii: true
      block_patterns: ["ssn", "credit_card", "private_key"]
    data_export:
      scan: true
      max_records: 10000
      require_approval_above: 1000
    audit_log:
      scan: true
      mask_pii: true
  policy_engine:
    rules: ["prevent-ssn-leak", "prevent-pii-email", "prevent-large-export"]
  incident_response:
    alert: true
    block: true
    quarantine: false
    notify: ["security-team"]
```

## Best Practices

1. **Scan all outbound channels** — Email, web, API, file
2. **Block restricted data** — SSN, credit cards, private keys
3. **Require approval for bulk** — Large exports need sign-off
4. **Redact PII in responses** — Don't leak via API
5. **Log all DLP events** — Full audit trail
6. **Update patterns regularly** — New data formats emerge
7. **Test with real data** — Verify detection works
8. **Educate users** — DLP is a safety net, not a substitute for training
9. **Balance security and productivity** — Don't block legitimate work
10. **Integrate with SIEM** — Forward DLP events for correlation