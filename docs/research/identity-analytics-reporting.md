# Identity Analytics and Reporting: Metrics, Dashboards, and Insights for GGID

> **Focus**: A comprehensive identity analytics platform — time-series metrics, behavioral analytics, authentication trends, compliance reporting, anomaly detection, and custom dashboards — giving organizations real-time visibility into their identity infrastructure.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Includes endpoint precondition check (§7), DoD per backlog item (§16), curl verification commands (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Why Identity Analytics Matters](#2-why-identity-analytics-matters)
3. [Industry Landscape](#3-industry-landscape)
4. [GGID Current State Analysis](#4-ggid-current-state-analysis)
5. [Gap Analysis](#5-gap-analysis)
6. [Proposed Architecture: Identity Analytics Engine](#6-proposed-architecture-identity-analytics-engine)
7. [Endpoint Precondition Check](#7-endpoint-precondition-check)
8. [API Design + Curl Commands](#8-api-design--curl-commands)
9. [Database Schema](#9-database-schema)
10. [Metrics Catalog](#10-metrics-catalog)
11. [Anomaly Detection](#11-anomaly-detection)
12. [Compliance Reporting](#12-compliance-reporting)
13. [Console UI Design](#13-console-ui-design)
14. [Performance Considerations](#14-performance-considerations)
15. [Competitive Differentiation](#15-competitive-differentiation)
16. [Implementation Backlog with DoD](#16-implementation-backlog-with-dod)

---

## 1. Executive Summary

Identity systems generate mountains of data every day — logins, device signals, access requests, policy evaluations, MFA challenges. Organizations need this data transformed into **actionable insights**: who's logging in, from where, what methods they're using, which policies are blocking access, and where anomalies suggest threats.

The IAM behavioral analytics market is projected to grow from $538M (2024) to $1.8B (2030). Okta, Microsoft Entra, and Auth0 all ship built-in analytics dashboards. Gartner's 2026 IAM Planning Guide emphasizes **native behavioral analytics** as a top evaluation criterion.

GGID has basic analytics foundations:
- `dashboard_stats_handler.go` — aggregate counts (users, sessions, MFA devices)
- `login_analytics_handler.go` — login events with date filtering
- Audit service with report generation (`audit/http.go:178-192`)
- Prometheus metrics at `/metrics` (`gateway/middleware/metrics.go:50`)
- MCP tool for dashboard stats (`mcp/tools/audit.go:36`)

However, GGID is **missing the analytics platform layer**:
1. **No time-series aggregation** — data is point-in-time, no trend analysis
2. **No behavioral analytics** — no baseline establishment or deviation detection
3. **No authentication method analytics** — can't track passkey vs password adoption over time
4. **No geographic/IP analytics** — no login location heatmaps or impossible-travel detection
5. **No policy decision analytics** — can't see allow/deny rates per policy
6. **No custom dashboards** — only fixed dashboard stats endpoint
7. **No scheduled reports** — no email/PDF report delivery
8. **No export capabilities** — no CSV/JSON data export for BI tools
9. **Hardcoded stats** — PAR/JAR usage stats use hardcoded values, not real data
10. **No data retention pipeline** — no aggregation/rollup for long-term analytics

**Recommendation**: Build an **Identity Analytics Engine** with time-series aggregation, behavioral baselining, policy decision tracking, anomaly detection, compliance reporting, custom dashboards, and export APIs.

**Estimated effort**: 4 sprints for MVP (event pipeline + aggregation + dashboards + reports).

---

## 2. Why Identity Analytics Matters

### The Identity Data Problem

```
A 10,000-user organization generates daily:
├── ~50,000 authentication events
├── ~200,000 policy evaluations
├── ~15,000 MFA challenges
├── ~500 failed login attempts
├── ~50 password resets
├── ~20 account lockouts
├── ~5 suspicious activity alerts
└── ~2 potential security incidents

Without analytics, this data is:
├── Trapped in audit logs (unreadable at scale)
├── No trends visible (is MFA adoption increasing?)
├── No baselines (what's "normal" for this tenant?)
├── No early warning (brute-force patterns invisible)
└── No compliance reporting (auditors need summaries)
```

### Key Questions Analytics Must Answer

| Question | Analytics Capability | Business Value |
|----------|---------------------|---------------|
| "How many users adopted passkeys this month?" | Auth method trends | Security posture |
| "Is there a spike in failed logins from Russia?" | Geographic anomaly detection | Threat detection |
| "Which policy denies the most requests?" | Policy decision analytics | Policy tuning |
| "What % of users use MFA?" | MFA adoption metrics | Compliance |
| "How fast is our average login?" | Performance analytics | UX optimization |
| "Show me all admin actions last quarter" | Compliance report | Audit/SOC2 |
| "Is impossible travel happening?" | Behavioral analytics | Account takeover |
| "Generate a GDPR data processing report" | Compliance reporting | Legal compliance |

---

## 3. Industry Landscape

### Market Data

- IAM behavioral analytics market: **$538M (2024) → $1.8B (2030)**, CAGR 22%
- Overall IAM market: **$26B (2025) → $42.6B (2030)**
- 47% of organizations cite **improved security posture** as top analytics benefit
- Gartner 2026: **native behavioral analytics** is a top-3 IAM evaluation criterion

### Comparison Matrix

| Feature | Okta | Microsoft Entra | Auth0 | Keycloak | **GGID (target)** |
|---------|------|-----------------|-------|----------|-------------------|
| **Auth method trends** | Yes (detailed) | Yes (Sign-in logs) | Yes (dashboard) | No | **Target** |
| **Geographic analytics** | Yes (map + heatmap) | Yes (sign-in locations) | Custom | No | **Target** |
| **Policy decision analytics** | Yes (policy insights) | Yes (Conditional Access insights) | No | No | **Target** |
| **Behavioral analytics** | Yes (risk events) | Yes (Identity Protection) | Custom | No | **Target** |
| **Anomaly detection** | Yes (automated) | Yes (risk detection) | Custom | No | **Target** |
| **Compliance reports** | Yes (scheduled PDF) | Yes (usage reports) | No | No | **Target** |
| **Custom dashboards** | Yes (workflows) | Yes (workbooks) | Fixed dashboards | No | **Target** |
| **Data export** | CSV/JSON/Splunk | Log Analytics/Sentinel | Splunk/Datadog | No | **Target** |
| **Scheduled email reports** | Yes | Yes | No | No | **Target** |
| **API for analytics** | Yes (Reports API) | Yes (Graph API) | Yes (Stats API) | No | **Target** |
| **Open source** | No | No | No | Yes | **Yes** |

---

## 4. GGID Current State Analysis

### Existing Analytics Infrastructure

| Component | File:Line | Status |
|-----------|-----------|--------|
| Dashboard stats | `identity/server/dashboard_stats_handler.go:11` | **Implemented** — aggregate counts |
| Login analytics | `auth/server/login_analytics_handler.go:7` | **Implemented** — login events by date |
| Audit report generation | `audit/server/http.go:191` | **Implemented** — `/audit/reports/generate` |
| Audit report download | `audit/server/http.go:192` | **Implemented** — `/audit/reports/` |
| Regulatory report | `audit/server/http.go:178` | **Implemented** — `/audit/regulatory/report` |
| Prometheus metrics | `gateway/middleware/metrics.go:50` | **Implemented** — `/metrics` endpoint |
| MCP dashboard tool | `mcp/tools/audit.go:36` | **Implemented** — `get_dashboard_stats` |
| PAR usage stats | `oauth/server/par_config_handler.go:14` | **Hardcoded** — values not real |
| JAR usage stats | `oauth/server/jar_config_handler.go:14` | **Hardcoded** — values not real |
| SIEM connector | `docs/research/siem-connector-design.md` | **Researched** — event streaming |
| Real-time alerting | `docs/research/realtime-alerting-design.md` | **Researched** — threshold alerts |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No time-series aggregation** | Can't show trends over time (weekly/monthly) |
| 2 | **No behavioral baselines** | Can't detect deviations from normal patterns |
| 3 | **No auth method analytics** | Can't track passkey/password/OTP adoption trends |
| 4 | **No geographic analytics** | Can't show login locations or detect impossible travel |
| 5 | **No policy decision tracking** | Can't see which policies allow/deny most |
| 6 | **No custom dashboards** | Only fixed endpoint, no configurable widgets |
| 7 | **No scheduled reports** | No automated email/PDF delivery |
| 8 | **No data export** | Can't export to CSV/JSON for external BI |
| 9 | **Hardcoded stats** | PAR/JAR usage use fake numbers |
| 10 | **No event aggregation pipeline** | Raw events stored but not aggregated for analytics |

---

## 5. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "Show passkey adoption trend over 6 months" | No data | Time-series chart: weekly enrollments |
| 2 | "Alert when failed logins spike 300%" | No baseline | Behavioral anomaly: 3σ deviation alert |
| 3 | "Which policy denies 40% of requests?" | No tracking | Policy decision breakdown with allow/deny rates |
| 4 | "Generate Q3 compliance report for SOC2" | Manual audit log review | One-click scheduled PDF report |
| 5 | "Export all login events to Splunk" | No export API | CSV/JSON streaming or webhook to SIEM |
| 6 | "Show heatmap of login locations" | No geo data | Geographic visualization with anomaly markers |
| 7 | "Average login latency this week" | No performance tracking | P50/P95/P99 login latency trend |

---

## 6. Proposed Architecture: Identity Analytics Engine

```
                    ┌──────────────────────────────────────────────┐
                    │       Identity Analytics Engine               │
                    │                                              │
                    │  ┌───────────────────────────────────────┐   │
                    │  │  Event Collector                       │   │
                    │  │  (NATS subscriber / direct insert)    │   │
                    │  │                                       │   │
                    │  │  Events:                              │   │
                    │  │  ├── auth.login (success/fail)        │   │
                    │  │  ├── auth.mfa_challenge               │   │
                    │  │  ├── auth.password_reset              │   │
                    │  │  ├── policy.decision (allow/deny)     │   │
                    │  │  ├── oauth.token_issued               │   │
                    │  │  ├── identity.user_created            │   │
                    │  │  ├── identity.user_disabled           │   │
                    │  │  ├── delegation.granted               │   │
                    │  │  └── session.created/revoked          │   │
                    │  └──────────────┬────────────────────────┘   │
                    │                 │                            │
                    │  ┌──────────────▼────────────────────────┐   │
                    │  │  Aggregation Pipeline                  │   │
                    │  │                                       │   │
                    │  │  ├── Raw events → event_store          │   │
                    │  │  ├── Hourly rollups → metrics_hourly   │   │
                    │  │  ├── Daily rollups → metrics_daily     │   │
                    │  │  └── Monthly rollups → metrics_monthly │   │
                    │  └──────────────┬────────────────────────┘   │
                    │                 │                            │
                    │  ┌──────────────▼────────────────────────┐   │
                    │  │  Analytics API                         │   │
                    │  │                                       │   │
                    │  │  ├── /api/v1/analytics/overview        │   │
                    │  │  ├── /api/v1/analytics/auth-methods    │   │
                    │  │  ├── /api/v1/analytics/geographic      │   │
                    │  │  ├── /api/v1/analytics/policy          │   │
                    │  │  ├── /api/v1/analytics/anomalies       │   │
                    │  │  ├── /api/v1/analytics/trends          │   │
                    │  │  ├── /api/v1/analytics/reports         │   │
                    │  │  └── /api/v1/analytics/export          │   │
                    │  └───────────────────────────────────────┘   │
                    └──────────────────────────────────────────────┘
```

### Data Flow

```
1. User logs in → Auth Service publishes auth.login event to NATS
2. Event Collector subscribes → writes to analytics_event_store
3. Hourly aggregation job → computes rollups into metrics_hourly
4. Analytics API → queries aggregated data for dashboards
5. Anomaly detector → runs baseline comparison, flags deviations
6. Report scheduler → generates PDF, sends via email/webhook
```

---

## 7. Endpoint Precondition Check

### Existing Endpoints (Reusable)

| Endpoint | File:Line | Status | Reusable? |
|----------|-----------|--------|-----------|
| `GET /api/v1/identity/dashboard/stats` | `identity/server/dashboard_stats_handler.go:11` | **Works** | Yes — base for overview |
| `GET /api/v1/auth/login-analytics` | `auth/server/login_analytics_handler.go:7` | **Works** | Yes — login event source |
| `POST /api/v1/audit/reports/generate` | `audit/server/http.go:191` | **Works** | Yes — report framework |
| `GET /api/v1/audit/reports/{id}` | `audit/server/http.go:192` | **Works** | Yes — report download |
| `GET /api/v1/audit/regulatory/report` | `audit/server/http.go:178` | **Works** | Yes — compliance base |
| `GET /metrics` | `gateway/middleware/metrics.go:50` | **Works** | Yes — Prometheus scrape |
| MCP `get_dashboard_stats` | `mcp/tools/audit.go:36` | **Works** | Yes — AI analytics |

### New Endpoints Required

| Endpoint | Method | Purpose | Priority |
|----------|--------|---------|----------|
| `/api/v1/analytics/overview` | GET | Time-series dashboard overview | P0 |
| `/api/v1/analytics/auth-methods` | GET | Auth method adoption trends | P0 |
| `/api/v1/analytics/geographic` | GET | Login location analytics | P1 |
| `/api/v1/analytics/policy-decisions` | GET | Policy allow/deny breakdown | P1 |
| `/api/v1/analytics/anomalies` | GET | Detected anomalies list | P1 |
| `/api/v1/analytics/trends` | GET | Configurable trend query | P0 |
| `/api/v1/analytics/reports/schedule` | POST | Schedule recurring report | P1 |
| `/api/v1/analytics/export` | GET | CSV/JSON data export | P1 |
| `/api/v1/analytics/performance` | GET | Auth latency percentiles | P2 |

---

## 8. API Design + Curl Commands

### Analytics Overview

```bash
# Get analytics overview with time range
curl "https://ggid.corp.com/api/v1/analytics/overview?from=2026-07-01&to=2026-07-17" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Response:
{
  "period": { "from": "2026-07-01", "to": "2026-07-17" },
  "summary": {
    "total_logins": 487230,
    "unique_users": 4821,
    "failed_logins": 3217,
    "success_rate": 99.34,
    "mfa_challenges": 89200,
    "mfa_success_rate": 97.8,
    "avg_login_ms": 245,
    "p95_login_ms": 890
  },
  "daily_trend": [
    { "date": "2026-07-01", "logins": 28100, "failures": 190, "unique_users": 4100 },
    { "date": "2026-07-02", "logins": 29500, "failures": 210, "unique_users": 4250 }
  ]
}
```

### Auth Method Analytics

```bash
curl "https://ggid.corp.com/api/v1/analytics/auth-methods?period=30d" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Response:
{
  "methods": [
    { "method": "password", "count": 185000, "percentage": 38.0, "trend": "-5.2%" },
    { "method": "passkey", "count": 142000, "percentage": 29.2, "trend": "+12.8%" },
    { "method": "totp", "count": 98000, "percentage": 20.1, "trend": "-2.1%" },
    { "method": "magic_link", "count": 38000, "percentage": 7.8, "trend": "+1.5%" },
    { "method": "sms_otp", "count": 24230, "percentage": 4.9, "trend": "-7.3%" }
  ],
  "weekly_trend": [
    { "week": "2026-W27", "password": 42000, "passkey": 28000, "totp": 22000 },
    { "week": "2026-W28", "password": 39000, "passkey": 33000, "totp": 21000 }
  ]
}
```

### Geographic Analytics

```bash
curl "https://ggid.corp.com/api/v1/analytics/geographic?period=7d" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Response:
{
  "countries": [
    { "country": "US", "logins": 210000, "failures": 1200, "unique_ips": 4200 },
    { "country": "CN", "logins": 1800, "failures": 950, "unique_ips": 3 },
    { "country": "DE", "logins": 45000, "failures": 200, "unique_ips": 1800 }
  ],
  "impossible_travel": [
    {
      "user_id": "uuid",
      "user_email": "alice@corp.com",
      "login1": { "ip": "1.2.3.4", "city": "New York", "time": "2026-07-17T08:00:00Z" },
      "login2": { "ip": "5.6.7.8", "city": "Moscow", "time": "2026-07-17T08:30:00Z" },
      "distance_km": 7500,
      "time_delta_min": 30,
      "required_speed_kmh": 15000
    }
  ]
}
```

### Policy Decision Analytics

```bash
curl "https://ggid/analytics/policy-decisions?period=30d&group_by=policy" \
  -H "Authorization: Bearer $TOKEN"

# Response:
{
  "policies": [
    { "policy": "admin-access", "total": 4500, "allow": 3800, "deny": 700, "deny_rate": 15.6 },
    { "policy": "data-access", "total": 89000, "allow": 87000, "deny": 2000, "deny_rate": 2.2 },
    { "policy": "after-hours", "total": 12000, "allow": 8000, "deny": 4000, "deny_rate": 33.3 }
  ]
}
```

### Data Export

```bash
# Export login events as CSV
curl "https://ggid.corp.com/api/v1/analytics/export?format=csv&type=logins&from=2026-07-01&to=2026-07-17" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -o logins_export.csv

# Export as JSON
curl "https://ggid.corp.com/api/v1/analytics/export?format=json&type=policy-decisions&from=2026-07-01&to=2026-07-17" \
  -H "Authorization: Bearer $TOKEN" \
  -o policy_decisions.json
```

### Scheduled Reports

```bash
# Schedule weekly compliance report
curl -X POST "https://ggid.corp.com/api/v1/analytics/reports/schedule" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Weekly SOC2 Report",
    "type": "compliance",
    "schedule": "0 9 * * 1",
    "format": "pdf",
    "recipients": ["security@corp.com"],
    "filters": { "include_failed_logins": true, "include_policy_denials": true }
  }'
```

---

## 9. Database Schema

```sql
-- Analytics event store (raw events for querying)
CREATE TABLE analytics_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    event_type          VARCHAR(64) NOT NULL,       -- 'auth.login', 'policy.decision', etc.
    user_id             UUID,
    user_email          VARCHAR(256),
    
    -- Event data
    success             BOOLEAN,
    method              VARCHAR(32),                 -- 'password', 'passkey', etc.
    ip_address          VARCHAR(45),
    country             VARCHAR(2),
    city                VARCHAR(128),
    user_agent          TEXT,
    
    -- Performance
    duration_ms         INT,
    
    -- Policy context (for policy events)
    policy_name         VARCHAR(128),
    decision            VARCHAR(16),                 -- 'allow', 'deny'
    deny_reason         TEXT,
    
    -- Metadata
    metadata            JSONB DEFAULT '{}',
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hourly rollups (aggregated metrics per hour)
CREATE TABLE analytics_metrics_hourly (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    metric_date         DATE NOT NULL,
    metric_hour         INT NOT NULL,                -- 0-23
    
    -- Dimensions
    event_type          VARCHAR(64) NOT NULL,
    method              VARCHAR(32),
    country             VARCHAR(2),
    success             BOOLEAN,
    
    -- Aggregated values
    count               BIGINT NOT NULL,
    unique_users        INT NOT NULL,
    avg_duration_ms     INT,
    p95_duration_ms     INT,
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, metric_date, metric_hour, event_type, method, country, success)
);

-- Daily rollups
CREATE TABLE analytics_metrics_daily (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    metric_date         DATE NOT NULL,
    
    event_type          VARCHAR(64) NOT NULL,
    method              VARCHAR(32),
    country             VARCHAR(2),
    success             BOOLEAN,
    
    count               BIGINT NOT NULL,
    unique_users        INT NOT NULL,
    avg_duration_ms     INT,
    p95_duration_ms     INT,
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, metric_date, event_type, method, country, success)
);

-- Behavioral baselines (for anomaly detection)
CREATE TABLE analytics_baselines (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    metric              VARCHAR(128) NOT NULL,       -- 'logins_per_hour', 'failures_per_hour'
    
    -- Baseline statistics (computed over trailing 30 days)
    mean                DOUBLE PRECISION NOT NULL,
    stddev              DOUBLE PRECISION NOT NULL,
    p95                 DOUBLE PRECISION,
    p99                 DOUBLE PRECISION,
    
    -- Dimensions
    dimension_key       VARCHAR(256),                -- e.g. 'country:CN' or 'method:password'
    
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, metric, dimension_key)
);

-- Anomaly detection results
CREATE TABLE analytics_anomalies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    type                VARCHAR(64) NOT NULL,        -- 'spike', 'impossible_travel', 'new_location'
    severity            VARCHAR(16) NOT NULL,        -- 'low', 'medium', 'high', 'critical'
    
    -- Details
    metric              VARCHAR(128),
    observed_value      DOUBLE PRECISION,
    baseline_mean       DOUBLE PRECISION,
    baseline_stddev     DOUBLE PRECISION,
    z_score             DOUBLE PRECISION,
    
    -- Context
    user_id             UUID,
    ip_address          VARCHAR(45),
    country             VARCHAR(2),
    description         TEXT,
    
    -- State
    status              VARCHAR(16) DEFAULT 'open',  -- 'open', 'acknowledged', 'resolved'
    resolved_at         TIMESTAMPTZ,
    resolved_by         UUID,
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Scheduled reports
CREATE TABLE analytics_scheduled_reports (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(256) NOT NULL,
    type                VARCHAR(64) NOT NULL,        -- 'compliance', 'summary', 'custom'
    schedule_cron       VARCHAR(64) NOT NULL,        -- '0 9 * * 1' = every Monday 9am
    format              VARCHAR(16) DEFAULT 'pdf',   -- 'pdf', 'csv', 'json'
    recipients          JSONB DEFAULT '[]',           -- ["security@corp.com"]
    filters             JSONB DEFAULT '{}',
    enabled             BOOLEAN DEFAULT true,
    last_run_at         TIMESTAMPTZ,
    next_run_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Custom dashboard configurations
CREATE TABLE analytics_dashboards (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    name                VARCHAR(128) NOT NULL,
    owner_id            UUID NOT NULL,
    widgets             JSONB NOT NULL DEFAULT '[]', -- Widget configs
    shared              BOOLEAN DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_events_tenant_time ON analytics_events (tenant_id, created_at DESC);
CREATE INDEX idx_events_type_time ON analytics_events (tenant_id, event_type, created_at DESC);
CREATE INDEX idx_events_user ON analytics_events (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_events_country ON analytics_events (tenant_id, country, created_at DESC);
CREATE INDEX idx_hourly_tenant_date ON analytics_metrics_hourly (tenant_id, metric_date, metric_hour);
CREATE INDEX idx_daily_tenant_date ON analytics_metrics_daily (tenant_id, metric_date);
CREATE INDEX idx_baselines_tenant ON analytics_baselines (tenant_id, metric);
CREATE INDEX idx_anomalies_tenant_time ON analytics_anomalies (tenant_id, created_at DESC);
CREATE INDEX idx_anomalies_open ON analytics_anomalies (tenant_id, status) WHERE status = 'open';
CREATE INDEX idx_reports_tenant ON analytics_scheduled_reports (tenant_id, enabled, next_run_at);
```

---

## 10. Metrics Catalog

### Authentication Metrics

| Metric | Description | Granularity |
|--------|-------------|-------------|
| `login_total` | Total login attempts | Per hour/day/month |
| `login_success_rate` | % successful logins | Per hour/day |
| `login_failure_rate` | % failed logins | Per hour/day |
| `unique_users` | Unique users logging in | Per day |
| `avg_login_duration_ms` | Average login time | Per hour/day |
| `p95_login_duration_ms` | 95th percentile login time | Per hour/day |
| `method_distribution` | Auth method usage breakdown | Per day |
| `method_adoption_trend` | Method adoption over time | Weekly trend |

### MFA Metrics

| Metric | Description |
|--------|-------------|
| `mfa_challenge_total` | Total MFA challenges issued |
| `mfa_success_rate` | % MFA challenges that succeeded |
| `mfa_method_distribution` | TOTP vs SMS vs passkey breakdown |
| `mfa_avg_duration_ms` | Average MFA completion time |

### Policy Metrics

| Metric | Description |
|--------|-------------|
| `policy_total` | Total policy evaluations |
| `policy_allow_rate` | % of decisions that allowed |
| `policy_deny_rate` | % of decisions that denied |
| `policy_top_denied` | Most frequently denied policy |
| `policy_latency_ms` | Average policy evaluation time |

### Geographic Metrics

| Metric | Description |
|--------|-------------|
| `logins_by_country` | Login count per country |
| `failures_by_country` | Failed login rate per country |
| `unique_ips_by_country` | Unique IP addresses per country |
| `impossible_travel_events` | Detected impossible-travel events |

---

## 11. Anomaly Detection

### Detection Methods

| Type | Method | Threshold |
|------|--------|-----------|
| **Volume spike** | Z-score > 3σ on hourly login volume | `observed > mean + 3*stddev` |
| **Failure spike** | Z-score > 3σ on hourly failure rate | `observed > mean + 3*stddev` |
| **New country** | Login from country not in 30-day baseline | `country ∉ baseline_countries` |
| **New IP for user** | User logging in from unseen IP | `ip ∉ user_historical_ips` |
| **Impossible travel** | Two logins from distant locations within impossible timeframe | `speed > 900 km/h` |
| **Off-hours access** | Login outside user's typical hours | `hour ∉ user_typical_hours` |
| **Brute force** | >10 failed logins from same IP in 5 minutes | Rate-based threshold |

### Implementation

```go
// services/audit/internal/service/anomaly_detector.go

func (s *AnomalyDetector) RunHourly(ctx context.Context, tenantID uuid.UUID) error {
    // 1. Get current hour metrics
    current := s.getHourlyMetrics(ctx, tenantID, time.Now())
    
    // 2. Get baseline (30-day average for same hour)
    baseline := s.getBaseline(ctx, tenantID, "logins_per_hour")
    
    // 3. Compute z-score
    zScore := (current.LoginCount - baseline.Mean) / baseline.Stddev
    
    if zScore > 3.0 {
        s.createAnomaly(ctx, &Anomaly{
            TenantID:       tenantID,
            Type:           "volume_spike",
            Severity:       severityFromZScore(zScore),
            Metric:         "logins_per_hour",
            ObservedValue:  current.LoginCount,
            BaselineMean:   baseline.Mean,
            BaselineStddev: baseline.Stddev,
            ZScore:         zScore,
            Description:    fmt.Sprintf("Login volume %.0f is %.1fσ above baseline", current.LoginCount, zScore),
        })
    }
    
    return nil
}
```

---

## 12. Compliance Reporting

### Report Types

| Report | Audience | Content | Format |
|--------|----------|---------|--------|
| **SOC2 Access Report** | Auditors | User access changes, admin actions, failed logins | PDF |
| **GDPR Data Processing** | DPO | Data access events, consent changes, data exports | PDF + CSV |
| **ISO 27001 Access Control** | Auditors | Policy decisions, MFA coverage, access reviews | PDF |
| **Monthly Security Summary** | CISO | Anomalies, incidents, threat indicators, trends | PDF |
| **Custom Report** | Any | Configurable filters and widgets | PDF/CSV/JSON |

### Scheduled Report Pipeline

```
1. Cron triggers report generation at scheduled time
2. Query analytics_events + metrics tables with report filters
3. Aggregate data into report sections
4. Render to PDF (using go-pdf or similar)
5. Store in audit/reports/ with expiry
6. Email to recipients with download link
7. Log report generation to audit trail
```

---

## 13. Console UI Design

### Analytics Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  Identity Analytics                                              │
│  [Today] [7 Days] [30 Days] [90 Days] [Custom]                  │
│                                                                  │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐    │
│  │ Logins     │ │ Success    │ │ MFA Rate   │ │ Avg Latency│    │
│  │ 28,450     │ │ 99.3%      │ │ 87.2%      │ │ 245ms      │    │
│  │ +5.2% vs  │ │ +0.1%      │ │ +3.1%      │ │ -12ms      │    │
│  └────────────┘ └────────────┘ └────────────┘ └────────────┘    │
│                                                                  │
│  Login Trend (30 days)                                           │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │     ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁▂▃▄▅▆▇█▇                              │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Authentication Methods                                          │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ Passkey   ████████████████████░░  29% (+12.8%)            │  │
│  │ Password  ██████████████████████  38% (-5.2%)             │  │
│  │ TOTP      ████████████░░░░░░░░░░  20% (-2.1%)             │  │
│  │ Magic Link████████░░░░░░░░░░░░░   8% (+1.5%)              │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Top Countries by Login Volume          Anomalies (3 open)       │
│  ┌──────────────────────────┐           ┌──────────────────────┐ │
│  │ US  ████████████ 210K    │           │ ⚠ Spike: CN failures │ │
│  │ DE  ████ 45K             │           │ ⚠ New IP: admin user │ │
│  │ UK  ███ 32K              │           │ ⚠ Off-hours: 3AM     │ │
│  └──────────────────────────┘           └──────────────────────┘ │
│                                                                  │
│  [Export Data] [Schedule Report] [Create Dashboard]              │
└──────────────────────────────────────────────────────────────────┘
```

---

## 14. Performance Considerations

| Operation | Latency | Strategy |
|-----------|---------|---------|
| Dashboard overview (aggregated) | <50ms | Query hourly/daily rollups |
| Trend query (30-day daily) | <100ms | Pre-aggregated daily rollups |
| Raw event query | 100-500ms | Indexed by tenant + time + type |
| Export (10K events) | 1-3s | Streaming CSV writer |
| Anomaly detection run | <5s | Hourly job, reads baselines |
| Report PDF generation | 5-15s | Background job |

### Data Retention Strategy

| Data Type | Hot Storage | Warm Storage | Cold Storage |
|-----------|------------|-------------|-------------|
| Raw events | 30 days (PostgreSQL) | 90 days (compressed) | 1 year (archive table) |
| Hourly rollups | 90 days | 1 year | 5 years |
| Daily rollups | 1 year | 5 years | Indefinite |
| Anomalies | Indefinite | — | — |

---

## 15. Competitive Differentiation

| Feature | GGID (target) | Okta | Microsoft Entra | Auth0 |
|---------|---------------|------|-----------------|-------|
| **Time-series analytics** | **Built-in** | Yes | Yes | Yes |
| **Behavioral baselining** | **3σ anomaly detection** | Yes (advanced) | Yes (ML-based) | Custom |
| **Auth method trends** | **Passkey adoption tracking** | Yes | Yes | Yes |
| **Geographic analytics** | **Impossible travel** | Yes | Yes | Custom |
| **Policy decision analytics** | **Per-policy allow/deny** | Yes | Yes | No |
| **Compliance reports** | **SOC2/GDPR/ISO templates** | Yes | Yes | No |
| **Scheduled email reports** | **Yes** | Yes | Yes | No |
| **Data export** | **CSV/JSON/Streaming** | Yes | Yes | Yes |
| **Custom dashboards** | **Widget-based** | Yes (workflows) | Yes (workbooks) | Fixed |
| **Open source** | **Yes (Apache 2.0)** | No | No | No |

---

## 16. Implementation Backlog with DoD

### P0 — Event Pipeline + Core Analytics (3 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Analytics DB schema | ✅ CREATE TABLE in migration ✅ go build PASS ✅ No log.Printf/内存 map | 2d |
| 2 | Event collector service | ✅ Subscribes to NATS events ✅ Writes to analytics_events ✅ DB-backed ✅ ≥3 tests | 4d |
| 3 | Hourly/daily aggregation jobs | ✅ Background goroutine ✅ Writes to rollup tables ✅ No hardcoded values ✅ ≥3 tests | 3d |
| 4 | Analytics overview API | ✅ `/api/v1/analytics/overview` registered in server.go ✅ DB-backed (queries rollups) ✅ curl test PASS ✅ ≥3 tests | 3d |
| 5 | Auth method analytics API | ✅ Returns real data from events ✅ Trend computation ✅ curl test PASS | 2d |
| 6 | Trends API | ✅ Configurable metric + time range ✅ DB-backed ✅ ≥3 tests | 2d |

### P1 — Anomaly Detection + Reporting (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 7 | Behavioral baseline computation | ✅ 30-day trailing statistics ✅ DB-backed ✅ ≥3 tests | 3d |
| 8 | Anomaly detector (3σ + impossible travel) | ✅ Detects spikes/new locations/travel ✅ Writes anomalies table ✅ ≥3 tests | 4d |
| 9 | Anomaly API + Console alerts | ✅ `/api/v1/analytics/anomalies` ✅ Acknowledge/resolve flow ✅ ≥3 tests | 3d |
| 10 | Scheduled report generator | ✅ Cron-triggered ✅ PDF output ✅ Email delivery ✅ ≥3 tests | 4d |
| 11 | Geographic analytics API | ✅ Country breakdown + impossible travel ✅ DB-backed ✅ ≥3 tests | 3d |
| 12 | Policy decision analytics API | ✅ Per-policy allow/deny rates ✅ DB-backed ✅ ≥3 tests | 2d |

### P2 — Console UI + Export (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 13 | Analytics dashboard | ✅ Overview cards + trend chart ✅ Method distribution ✅ Real API data | 4d |
| 14 | Anomaly management UI | ✅ List + acknowledge/resolve ✅ Severity indicators ✅ ≥3 tests | 3d |
| 15 | Report scheduling UI | ✅ Create/edit schedules ✅ Format/recipient config ✅ ≥3 tests | 3d |
| 16 | Data export (CSV/JSON) | ✅ Streaming download ✅ Tenant-scoped ✅ Rate-limited ✅ ≥3 tests | 2d |
| 17 | Custom dashboard builder | ✅ Widget configuration ✅ Drag-drop layout ✅ Share with team | 4d |

### P3 — Advanced Features (Future)

| # | Task | DoD |
|---|------|-----|
| 18 | ML-based anomaly detection | Replace 3σ with isolation forest / LSTM |
| 19 | User behavior profiling | Per-user typical hours, locations, methods |
| 20 | Real-time alerting integration | Wire anomalies to existing alerting system |
| 21 | Performance analytics (P50/P95/P99) | Auth latency percentiles per method |
| 22 | Federation analytics | SAML/OIDC federation login analytics |
| 23 | Data retention automation | Auto-archive events older than 30/90/365 days |
| 24 | Analytics webhook | Push metrics to external BI systems |

---

## References

- [Gartner 2026 IAM Planning Guide](https://www.gartner.com/en/documents/6993966) — Analytics as top evaluation criterion
- [IAM Behavioral Analytics Market](https://www.grandviewresearch.com/horizon/statistics/behavior-analytics-market/application/identity-access-management-iam/global) — $538M → $1.8B by 2030
- [Okta Sign-in Analytics](https://help.okta.com/en-us/Content/Topics/Analytics/analytics.htm) — Auth method trends and insights
- [Microsoft Entra Sign-in Logs](https://learn.microsoft.com/en-us/entra/identity/monitoring-health/concept-sign-ins) — Geographic and risk analytics
- [Auth0 Dashboard Statistics](https://auth0.com/docs/customize/dashboard) — Login analytics
- [GGID Dashboard Stats Handler](../services/identity/internal/server/dashboard_stats_handler.go) — Existing aggregate stats at line 11
- [GGID Login Analytics](../services/auth/internal/server/login_analytics_handler.go) — Login event analytics at line 7
- [GGID Audit Reports](../services/audit/internal/server/http.go) — Report generation at line 191
- [GGID Prometheus Metrics](../services/gateway/internal/middleware/metrics.go) — Metrics endpoint at line 50
- [GGID SIEM Connector Design](./siem-connector-design.md) — Event streaming architecture
- [GGID Real-time Alerting](./realtime-alerting-design.md) — Threshold-based alerting
