# Threat Intelligence Hub — Technical Guide

> Feature: B-37 Threat Intelligence Integration Hub
> Location: Audit service (`services/audit/internal/`)
> Console: `/security/threat-intel`

## What It Does

The Threat Intelligence Hub aggregates indicators of compromise (IOCs) from multiple external threat intel sources (OTX, AbuseIPDB, HIBP, MISP) and correlates them with internal ITDR detections. It provides real-time threat checking, source management, and statistical analysis.

## Architecture

```
External Sources (OTX/AbuseIPDB/HIBP/MISP)
         ↓
    Threat Intel Collector (periodic sync)
         ↓
    PostgreSQL (indicators + sources tables)
         ↓
    ┌─────────────────────────────┐
    │  Threat Intel API Endpoints  │
    └─────────────────────────────┘
         ↓              ↓              ↓
    Source CRUD    IOC Query     ITDR Correlation
```

## Components

### 1. Intel Sources

Manage external threat intelligence providers:

- **OTX (AlienVault)**: Pulse-based threat data, community-sourced.
- **AbuseIPDB**: IP reputation database.
- **HIBP (Have I Been Pwned)**: Breach data and compromised credentials.
- **MISP**: Open-source threat sharing platform.

Each source tracks:
- Connection status (connected, syncing, error)
- API key and sync interval
- Last sync timestamp and indicator count
- Auto-block rules (automatically block IOCs from this source)

### 2. Indicators (IOCs)

Threat indicators stored in PostgreSQL:

| IOC Type | Examples |
|----------|----------|
| **IP** | Malicious IPs, known C2 servers |
| **Domain** | Phishing domains, malware domains |
| **Email** | Spam senders, phishing emails |
| **Hash** | File hashes (MD5, SHA256) of known malware |
| **URL** | Malicious URLs, phishing links |
| **User Agent** | Known malicious user agent strings |

Each indicator has:
- **Confidence Score** (0-100): How reliable the indicator is.
- **Tags**: Categorization (e.g., `botnet`, `phishing`, `ransomware`).
- **Source**: Which intel provider contributed it.
- **First/Last Seen**: When the indicator was first and most recently observed.

### 3. Real-Time Threat Checker

Query all configured sources for a specific indicator:

### 4. ITDR Correlation

Cross-reference external threat intel with internal ITDR detections:
- Match internal login anomalies against external IOC databases.
- Identify users who logged in from known-malicious IPs.
- Correlate credential stuffing patterns with breach databases.

## API Endpoints

All endpoints are under the audit service:

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/audit/threat-intel/sources` | GET | List all configured sources |
| `/api/v1/audit/threat-intel/sources` | POST | Add a new source |
| `/api/v1/audit/threat-intel/sources/:id` | DELETE | Remove a source |
| `/api/v1/audit/threat-intel/indicators` | GET | Query indicators (filter by type, search) |
| `/api/v1/audit/threat-intel/check` | POST | Real-time check (IP, email, hash) |
| `/api/v1/audit/threat-intel/stats` | GET | Statistics and coverage metrics |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# List all intel sources
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/threat-intel/sources" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Query IP indicators
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/threat-intel/indicators?type=ip&limit=50" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Real-time threat check
NEW_TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/audit/threat-intel/check" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"type":"ip","value":"192.168.1.100"}'

# Get statistics
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/threat-intel/stats" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Source sync fails | Invalid API key or network issue | Verify API key; check network connectivity from audit pod |
| No indicators returned | Source never synced or filters too narrow | Check source sync status; broaden query filters |
| Check returns no match | IOC not in database or not yet synced | Try a different source or wait for next sync cycle |
| Stats show 0 indicators | Database not populated | Trigger manual sync from the console UI |

## Best Practices

- **Multiple sources**: Configure at least 3 intel sources for broader coverage.
- **Regular sync**: Set sync intervals to 15-30 minutes for near-real-time data.
- **Auto-block rules**: Enable auto-block for high-confidence sources like AbuseIPDB.
- **Confidence threshold**: Only act on indicators with confidence >= 70.
- **Correlate with ITDR**: Regularly review the ITDR correlation panel for cross-referenced threats.
- **Monitor API quotas**: External sources have API rate limits — monitor usage to avoid throttling.
