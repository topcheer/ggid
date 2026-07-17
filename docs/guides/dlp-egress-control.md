# DLP Egress Control — Technical Guide

> Feature: Data Loss Prevention (DLP) Egress Middleware
> Location: Gateway middleware (`services/gateway/internal/middleware/dlp_egress.go`)

## What It Does

The DLP Egress middleware intercepts API responses leaving GGID services and scans for Personally Identifiable Information (PII). Detected PII is automatically redacted according to configurable strategies, preventing accidental data leakage through API responses.

## Architecture

```
Client Request → Gateway → Backend Service → Response
                                              ↓
                                    DLP Egress Middleware
                                              ↓
                                    Scan JSON fields for PII
                                              ↓
                               Apply redaction strategy per field
                                              ↓
                                    Redacted Response → Client
```

## PII Detection Patterns

The middleware detects these PII types using regex patterns:

| PII Type | Pattern | Default Strategy |
|----------|---------|-----------------|
| **SSN** | `\d{3}-\d{2}-\d{4}` | Full mask |
| **Credit Card** | `\d{13,19}` (Luhn validated) | Partial mask |
| **Email** | RFC 5322 compliant | Email mask |
| **Phone** | International format | Partial mask |
| **API Key** | `sk_live_*`, `AKIA*` | Tokenize |
| **IBAN** | International bank account | Full mask |

## Redaction Strategies

| Strategy | Description | Example |
|----------|-------------|--------|
| `full_mask` | Replace entire value with `***REDACTED***` | `4111-1111-1111-1111` → `***REDACTED***` |
| `partial_mask` | Show first/last characters | `4111-****-****-1111` |
| `email_mask` | Mask local part | `j***@example.com` |
| `tokenize` | Replace with a token reference | `[TOKEN:abc123]` |
| `remove` | Remove the field entirely | Field omitted from JSON |

## Configuration

The `DLPEgressConfig` struct controls middleware behavior:

```go
type DLPEgressConfig struct {
    Enabled           bool                        `json:"enabled"`
    Patterns          map[string]*regexp.Regexp   `json:"patterns"`
    Strategies        map[string]RedactionStrategy `json:"strategies"`
    ExcludedPaths     []string                    `json:"excluded_paths"`
    AuditCallback     func(match PIIMatch)        `json:"-"`
}
```

- **Enabled**: Master toggle for the middleware.
- **Patterns**: Custom regex patterns keyed by PII type.
- **Strategies**: Per-PII-type redaction strategy overrides.
- **ExcludedPaths**: API paths that bypass DLP scanning (e.g., admin endpoints).
- **AuditCallback**: Function called for each PII match (for logging/metrics).

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/dlp/policies` | GET | List DLP policies |
| `/api/v1/dlp/policies` | POST | Create a DLP policy |
| `/api/v1/dlp/policies/:id` | PUT | Update a policy |
| `/api/v1/dlp/policies/:id` | DELETE | Delete a policy |
| `/api/v1/dlp/scan` | POST | Scan content for PII (on-demand) |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Scan content for PII
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/dlp/scan" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"content":"My SSN is 123-45-6789 and email is john@acme.com"}'

# List DLP policies
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/dlp/policies" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Create a DLP policy
NEW_TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/dlp/policies" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"SSN Detection","data_class":"pii","scope":"api","action":"block","enabled":true}'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| PII not being redacted | Middleware disabled or path excluded | Check `DLPEgressConfig.Enabled` and `ExcludedPaths` |
| False positive redaction | Regex too broad | Refine pattern specificity |
| Performance degradation | Large response bodies | Enable streaming scan or reduce response size |
| Missing PII type | Pattern not configured | Add custom regex to `Patterns` map |

## Best Practices

- **Start conservative**: Enable for `email` and `SSN` first, then expand.
- **Audit matches**: Always configure `AuditCallback` to log PII exposure.
- **Exclude admin paths**: Administrative endpoints may need raw data.
- **Test patterns**: Validate regex against test data before enabling in production.
- **Monitor performance**: Large JSON payloads can slow scanning. Benchmark at expected scale.
