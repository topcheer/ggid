# GGID v2 Multi-Role Review — Task Tracker

> Source: docs/v2-fullstack-review.md (2026-07-23)
> Last updated: 2026-07-23

## Developer (D1-D5)

| ID | Title | Priority | Status | Owner |
|----|-------|----------|--------|-------|
| D1 | OpenAPI→SDK drift detection CI | P1 | ✅ Done | ggcxf_cli |
| D2 | API Breaking Change detection CI | P1 | ✅ Done | ggcxf_cli |
| D3 | Console API Explorer (Try-it-now) | P2 | 🔲 Pending | — |
| D4 | Frontend SDK tree-shaking optimization | P3 | 🔲 Deferred (low impact) | — |
| D5 | Webhook payload signature verification SDK helper | P2 | 🔲 Pending | — |

## Admin (A1-A4)

| ID | Title | Priority | Status | Owner |
|----|-------|----------|--------|-------|
| A1 | Console 中文 i18n | P1 | ✅ Done | shen_frontend |
| A2 | Batch operations UI (bulk disable/delete/role) | P2 | 🔲 Pending | — |
| A3 | Role/permission export to PDF/Excel | P3 | 🔲 Deferred (low impact) | — |
| A4 | SCIM reverse sync confirmation | P2 | 🔲 Pending | — |

## DevOps (O1-O5)

| ID | Title | Priority | Status | Owner |
|----|-------|----------|--------|-------|
| O1 | Prometheus metrics standardization | P1 | ✅ Done | ggcxf_cli |
| O2 | DB backup auto-verification | P1 | ✅ Done | ggcxf_cli |
| O3 | Blue-green/canary deploy templates | P2 | ✅ Done | ggcxf_cli |
| O4 | values-dev.yaml for dev environments | P3 | 🔲 Pending | — |
| O5 | SLI/SLO definition + error budget dashboard | P2 | 🔲 Pending | — |

## Security (S1-S3)

| ID | Title | Priority | Status | Owner |
|----|-------|----------|--------|-------|
| S1 | SOC2/ISO27001 control mapping | P2 | ✅ Done | ggcxf_cli + guardian |
| S2 | Data residency policy enforcement | P3 | 🔲 Pending | — |
| S3 | Automated penetration testing scripts | P3 | 🔲 Pending | — |

## Product (P1-P4)

| ID | Title | Priority | Status | Owner |
|----|-------|----------|--------|-------|
| P1 | API Rate Limit visualization panel | P2 | 🔲 Pending | — |
| P2 | Feature Flag console enhancement | P2 | 🔲 Pending | — |
| P3 | NPS/CSAT feedback collection | P3 | 🔲 Deferred (needs user base) | — |
| P4 | Multi-tenant usage metering | P2 | ✅ Done | ggcxf_cli |

## Summary

- **Done**: 8 (D1, D2, A1, O1, O2, O3, S1, P4)
- **Pending**: 10 (D3, D5, A2, A4, O4, O5, S2, S3, P1, P2)
- **Deferred**: 3 (D4, A3, P3 — low impact / needs user base)
