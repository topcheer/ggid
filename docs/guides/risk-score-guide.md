# Risk Score Dashboard — User Guide

> Feature: F-44 Risk Scoring Engine
> Location: **Security > Risk Score** (`/security/risk-score`)
> Alias: `/risk-scoring` (route alias)

## What It Does

The Risk Score Dashboard provides a real-time view of user risk across the organization. It aggregates signals from multiple sources — authentication patterns, device trust, geographic anomalies, MFA status, and access behaviors — into a composite risk score per user. Administrators can identify high-risk users, drill into contributing factors, and trigger risk recalculation.

## How to Access

1. Log in to the GGID Admin Console.
2. Navigate to **Security** in the sidebar.
3. Click **Risk Score**.

Alternatively, go to `/security/risk-score` or `/risk-scoring` directly.

## Page Layout

### Risk Summary (Top Section)

Displays organization-wide risk metrics:

- **Risk Gauge**: Visual semicircle gauge showing average risk score (0-100).
- **Total Users**: Number of users with risk scores.
- **Average Score**: Mean risk score across all users.
- **High Risk Count**: Users with score >= 50.
- **Critical Count**: Users with score >= 75.

### Risk Distribution

Shows how users are distributed across risk levels:

- **Low (0-24)**: Green — minimal risk indicators.
- **Medium (25-49)**: Yellow — some risk factors present.
- **High (50-74)**: Orange — multiple risk factors, review recommended.
- **Critical (75-100)**: Red — immediate action required.

### Top Risk Factors

Lists the most impactful risk factors across the organization:

- **Factor Name**: e.g., "Impossible Travel", "MFA Not Enabled", "Credential Stuffing Pattern".
- **Weight**: How much this factor contributes to the overall score.
- **Average Value**: Mean contribution across affected users.
- **Description**: What the factor measures.

### High-Risk Users Table

A sortable table of users sorted by risk score (highest first):

- **Username and Email**: User identification.
- **Score**: Numeric risk score with color-coded level badge.
- **Level**: Low, Medium, High, or Critical badge.
- **Last Updated**: When the risk score was last recalculated.
- **Actions**: Recalculate button and View Details button.

**Workflow — Identify and investigate a high-risk user:**
1. Open the Risk Score Dashboard.
2. Check the High Risk Count and Critical Count in the summary.
3. Scroll to the High-Risk Users table.
4. Click **View Details** on the highest-risk user.
5. The detail modal opens showing:
   - Individual risk score gauge.
   - Breakdown of each contributing factor.
   - Factor weight and current value.
   - Factor description.
6. Assess whether the risk is legitimate (e.g., user traveling) or suspicious.
7. Take action: enforce MFA, reset password, or revoke sessions.

**Workflow — Recalculate risk for a user:**
1. Find the user in the table.
2. Click the **Recalculate** button (refresh icon).
3. The system re-evaluates all risk factors for that user.
4. The updated score appears in the table.

## Risk Factors

The risk scoring engine evaluates these factors:

| Factor | Description | Weight |
|--------|-------------|--------|
| Impossible Travel | Login from geographically impossible locations within a short timeframe | High |
| MFA Not Enabled | User has not enrolled in multi-factor authentication | High |
| Credential Stuffing | Repeated failed login attempts suggesting credential stuffing | Medium |
| Device Anomaly | Login from an unrecognized or unmanaged device | Medium |
| Off-Hours Access | Login outside the user's typical activity hours | Low |
| Privileged Access | User has administrative or sensitive role assignments | Medium |
| Stale Sessions | Long-lived sessions without re-authentication | Low |
| Geographic Risk | Login from high-risk countries or anonymizing services | High |

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/policy/risk-score/summary` | GET | Get organization-wide risk summary |
| `/api/v1/policy/risk-score/users` | GET | List all users with risk scores |
| `/api/v1/policy/risk-score/recalculate` | POST | Recalculate risk for a specific user |

### curl Examples

```bash
TOKEN="your-jwt-token"
TENANT="00000000-0000-0000-0000-000000000001"

# Get risk summary
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/policy/risk-score/summary" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# List high-risk users
NEW_TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/policy/risk-score/users" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "X-Tenant-ID: $TENANT"

# Recalculate risk for a user
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/policy/risk-score/recalculate" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"user_id":"user-123"}'
```

**Expected response (summary):**
```json
{
  "total_users": 150,
  "average_score": 22,
  "high_risk_count": 8,
  "critical_count": 2,
  "top_factors": [
    {"name": "MFA Not Enabled", "weight": 30, "value": 25, "description": "User has not enrolled in MFA"}
  ],
  "distribution": [
    {"level": "low", "count": 120},
    {"level": "medium", "count": 20},
    {"level": "high", "count": 8},
    {"level": "critical", "count": 2}
  ]
}
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Dashboard shows "Failed to load" | Policy service unreachable | Check `ggid-policy` pod: `kubectl get pod -n ggid -l app=ggid-policy` |
| All scores are 0 | Risk engine not configured or no risk factors enabled | Verify risk scoring is enabled in Policy settings |
| Recalculate has no effect | No new events since last calculation | Wait for new auth events or check audit pipeline |
| User not in list | User has no recent activity | Risk scores require recent authentication events |
| Score seems inaccurate | Stale data or missing signals | Click Recalculate to force re-evaluation |

## Best Practices

- **Review daily**: Check the Critical Count daily for immediate threats.
- **Act on critical users**: Users scoring 75+ should be investigated immediately.
- **Enforce MFA**: The "MFA Not Enabled" factor is often the easiest to remediate.
- **Recalculate after changes**: After policy changes or security incidents, recalculate affected users.
- **Correlate with incidents**: Use the Audit Explorer to correlate high-risk users with security events.
- **Trend monitoring**: Track the average score over time — rising scores indicate deteriorating security posture.
