# NIS2/CRA Compliance Dashboard — User Guide

> Feature: F-51 NIS2/CRA Compliance Dashboard
> Location: **Settings > Compliance Dashboard** (`/settings/compliance-dashboard`)

## What It Does

The NIS2/CRA Compliance Dashboard provides a visual overview of your organization's compliance posture across multiple regulatory frameworks. It shows coverage percentages, control status breakdowns, and compliance gaps for frameworks like NIS2 Directive, Cyber Resilience Act (CRA), ISO 27001, SOC 2, and more.

## How to Access

1. Log in to the GGID Admin Console.
2. Navigate to **Settings** in the sidebar.
3. Click **Compliance Dashboard**.

Alternatively, go to `/settings/compliance-dashboard` directly.

## Page Layout

### Framework Cards

Each compliance framework is displayed as a card with:

- **Donut Chart**: Visual coverage percentage (color-coded: green >=80%, yellow 60-79%, red <60%).
- **Framework Name**: NIS2, CRA, ISO 27001, SOC 2, etc.
- **Control Breakdown**: Compliant / total controls, partial count, missing count.
- **Gap Count**: Number of compliance gaps that need remediation.
- **Last Assessed**: Date of last compliance assessment.

### Gap Banner

If any framework has gaps, an orange banner appears at the top showing the total number of gaps across all frameworks.

### Expanded View

Click any framework card to expand it and see detailed gap information including:
- Specific control IDs that are missing or partial.
- Recommended remediation actions.
- Assessment history.

## Workflows

### Assess Compliance Posture

1. Open the Compliance Dashboard.
2. Review each framework's donut chart.
3. Identify frameworks with coverage below 80%.
4. Click the framework to expand and see specific gaps.
5. Prioritize remediation based on gap severity.

### Track Remediation Progress

1. Note the current coverage percentage for each framework.
2. After implementing fixes, click **Refresh** to reload.
3. The coverage should increase as gaps are resolved.
4. Monitor the gap count trending toward zero.

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/audit/compliance-dashboard` | GET | Get all framework compliance summaries |

### curl Example

```bash
TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/audit/compliance-dashboard" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

**Expected response:**
```json
[
  {
    "framework": "NIS2",
    "coverage_pct": 85,
    "total_controls": 40,
    "compliant": 34,
    "partial": 4,
    "missing": 2,
    "gap_count": 2,
    "last_assessed": "2026-07-18T00:00:00Z"
  }
]
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Dashboard shows "No data" | Audit service not returning compliance data | Check `ggid-audit` pod; ensure compliance assessment has been run |
| All frameworks at 0% | No controls mapped or assessment not run | Run compliance assessment in Audit > Compliance section first |
| Donut chart not rendering | SVG rendering issue in older browsers | Update browser to latest version |
| Coverage not updating after fixes | Assessment cached or not re-run | Restart audit pod or trigger re-assessment |

## Best Practices

- **Regular assessments**: Run compliance assessments quarterly or after major changes.
- **Prioritize gaps**: Focus on frameworks with legal deadlines (NIS2 has October 2024 deadline).
- **Track trends**: Monitor coverage percentages over time to ensure improvement.
- **Cross-reference**: Use the Audit Explorer to trace specific control evidence.
- **Document remediation**: Keep records of how each gap was addressed for auditor reviews.

## NIS2 Specific Requirements

The NIS2 Directive (EU 2022/2555) requires:
- **Risk management measures**: Article 21 — implement appropriate technical and operational measures.
- **Incident reporting**: Article 23 — report significant incidents within 24 hours.
- **Supply chain security**: Article 21(2)(d) — secure supply chains and vendor relationships.
- **Training**: Article 21(2)(g) — regular cybersecurity training.

## CRA Specific Requirements

The Cyber Resilience Act (Regulation EU 2024/2847) requires:
- **Security by design**: Products must be designed with appropriate security levels.
- **Vulnerability disclosure**: Manufacturers must have disclosure processes.
- **Lifecycle support**: Security updates throughout the product lifecycle.
- **Conformity assessment**: CE marking requirements for digital products.
