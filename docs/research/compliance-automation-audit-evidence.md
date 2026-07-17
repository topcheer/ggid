# Compliance Automation & Audit Evidence Collection: Continuous Compliance for GGID

> **Focus**: Upgrading GGID's extensive compliance handler suite (37 handlers) from hardcoded/mock data to production-grade continuous compliance monitoring (CCM), automated evidence collection, and audit-ready reporting.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Checklist Compliance**: Endpoint precondition check (§6), DoD per backlog item (§7).

---

## 1. Executive Summary

GGID has a **massive compliance handler suite** — 37 files covering evidence collection, compliance dashboards, heatmaps, gaps, drift, auto-collection, certification export, regulatory reports, and more. However, most return **hardcoded mock data** rather than real evidence from actual system state.

**Existing handlers (37 files):**
- Compliance dashboard, heatmap, gaps, drift, config ✅ (hardcoded ⚠️)
- Evidence collection, auto-tag, versioning, attachment, chain, integrity, expiry ✅ (hardcoded ⚠️)
- Compliance mapping, framework coverage, widget ✅ (hardcoded ⚠️)
- Report handler, custom report, cert export, regulatory report ✅ (hardcoded ⚠️)
- Auto-score, schedule collect, schedule CRUD ✅ (hardcoded ⚠️)
- Forensics timeline, remediation progress, score history ✅ (hardcoded ⚠️)
- Export schedule config ✅ (hardcoded ⚠️)

**The pattern**: GGID has built the **entire API surface** for compliance automation but hasn't connected it to real data sources.

**Recommendation**: Wire the 37 handlers to real data via: automated evidence collectors (querying audit log, policy service, access reviews), continuous control monitoring engine, framework-to-control mapping (SOC2/ISO/NIST), and PDF report generation with evidence attachments.

---

## 2. Compliance Framework Mapping

### GGID Features → Control Mapping

| Framework | Control | GGID Feature | Evidence Source |
|-----------|---------|-------------|-----------------|
| **SOC2 CC6.1** | Logical access controls | OAuth 2.1 + RBAC/ABAC | Audit log: auth events |
| **SOC2 CC6.2** | User authentication | MFA + WebAuthn | Audit log: MFA challenges |
| **SOC2 CC6.3** | Access authorization | ReBAC + PDP decisions | Policy decisions log |
| **SOC2 CC6.6** | Access reviews | Lifecycle + dormant detection | Access review records |
| **SOC2 CC7.1** | System monitoring | ITDR detection + audit | Detection events |
| **SOC2 CC7.2** | Incident detection | ITDR rules + SOAR | Incident records |
| **SOC2 CC7.3** | Incident response | SOAR playbooks | Response logs |
| **SOC2 CC8.1** | Change management | Policy versioning | Policy change diff |
| **ISO A.9** | Access control | OAuth + RBAC + JIT | Access provisioning log |
| **ISO A.12.4** | Logging/monitoring | Audit service | Audit event count |
| **ISO A.16** | Incident management | ITDR + SOAR | Incident timeline |
| **ISO A.18.1** | Compliance (GDPR) | Consent + DSR + DLP | Consent records |

---

## 3. Evidence Collection Architecture

```
┌──────────────────────────────────────────────┐
│     Evidence Collection Engine                │
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │  Evidence Collectors (scheduled)      │    │
│  │  ├── Auth Evidence Collector          │    │
│  │  │   → query audit_events (MFA rate)  │    │
│  │  ├── Access Review Collector          │    │
│  │  │   → query policy decisions         │    │
│  │  ├── Incident Response Collector      │    │
│  │  │   → query detections + SOAR        │    │
│  │  ├── Change Management Collector      │    │
│  │  │   → query policy_versions diff     │    │
│  │  └── Configuration Collector          │    │
│  │      → export current policy configs  │    │
│  └──────────────┬───────────────────────┘    │
│                 │                            │
│  ┌──────────────▼───────────────────────┐    │
│  │  Evidence Store (PostgreSQL)          │    │
│  │  - evidence_records table             │    │
│  │  - Hash-chained for tamper-evidence   │    │
│  │  - Auto-tagged by control ID          │    │
│  └──────────────┬───────────────────────┘    │
│                 │                            │
│  ┌──────────────▼───────────────────────┐    │
│  │  Continuous Control Monitoring (CCM)  │    │
│  │                                      │    │
│  │  For each control:                   │    │
│  │  - Latest evidence timestamp         │    │
│  │  - Pass/fail/missing status          │    │
│  │  - Gap detection                     │    │
│  │  - Compliance score                  │    │
│  └──────────────────────────────────────┘    │
└──────────────────────────────────────────────┘
```

---

## 4. Endpoint Precondition Check

### Existing Handlers (Wire to Real Data)

| Handler | File | Current | Target |
|---------|------|---------|--------|
| compliance_dashboard | `compliance_dashboard_handler.go` | Hardcoded | Query CCM engine |
| compliance_heatmap | `compliance_heatmap_handler.go` | Hardcoded | Query control status |
| compliance_gaps | `compliance_gaps_handler.go` | Hardcoded | Query missing evidence |
| compliance_mapping | `compliance_mapping_handler.go` | Hardcoded | Real framework mapping |
| evidence_collection | `evidence_collection_handler.go` | Hardcoded | Real evidence from audit |
| evidence_auto_tag | `evidence_auto_tag_handler.go` | Hardcoded | Auto-tag by control ID |
| evidence_chain | `evidence_chain_handler.go` | Hardcoded | Hash-chain verification |
| evidence_integrity | `evidence_integrity_handler.go` | Hardcoded | Real integrity check |
| compliance_autocollect | `compliance_autocollect_handler.go` | Hardcoded | Real scheduled collection |
| auto_score | `auto_score_handler.go` | Hardcoded | Real compliance score |
| report | `report_handler.go` | Hardcoded | PDF with real evidence |
| cert_export | `cert_export_handler.go` | Hardcoded | Real certification export |
| framework_coverage | `framework_coverage_handler.go` | Hardcoded | Real coverage % |

---

## 5. Database Schema

```sql
CREATE TABLE compliance_controls (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    framework           VARCHAR(32) NOT NULL,     -- 'SOC2', 'ISO27001', 'NIST800-53'
    control_id          VARCHAR(32) NOT NULL,     -- 'CC6.1', 'A.9.1'
    title               VARCHAR(256) NOT NULL,
    description         TEXT,
    category            VARCHAR(64),
    evidence_source     VARCHAR(128),             -- 'audit_log', 'policy_decisions', 'access_review'
    collection_frequency VARCHAR(32) DEFAULT 'daily',
    status              VARCHAR(16) DEFAULT 'unknown', -- 'pass', 'fail', 'missing', 'unknown'
    last_collected_at   TIMESTAMPTZ,
    UNIQUE(tenant_id, framework, control_id)
);

CREATE TABLE compliance_evidence (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    control_id          UUID REFERENCES compliance_controls(id),
    evidence_type       VARCHAR(64),             -- 'log_query', 'config_export', 'screenshot'
    evidence_data       JSONB NOT NULL,           -- Actual evidence content
    evidence_hash       VARCHAR(64) NOT NULL,     -- SHA-256 hash
    prev_hash           VARCHAR(64),              -- Hash chain
    collected_by        VARCHAR(32) DEFAULT 'auto',
    collected_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE compliance_reports (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    framework           VARCHAR(32) NOT NULL,
    period_start        DATE NOT NULL,
    period_end          DATE NOT NULL,
    status              VARCHAR(16) DEFAULT 'draft',
    pdf_url             TEXT,
    controls_total      INT,
    controls_passing    INT,
    controls_failing    INT,
    controls_missing    INT,
    generated_at        TIMESTAMPTZ,
    generated_by        UUID
);

CREATE INDEX idx_controls_tenant_framework ON compliance_controls (tenant_id, framework);
CREATE INDEX idx_evidence_control_time ON compliance_evidence (tenant_id, control_id, collected_at DESC);
CREATE INDEX idx_reports_tenant ON compliance_reports (tenant_id, generated_at DESC);
```

---

## 6. Implementation Backlog with DoD

### P0 — Evidence Engine + CCM (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | Compliance DB schema | ✅ Migration ✅ go build PASS | 1d |
| 2 | Evidence collectors (5 types) | ✅ Query real data from audit/policy ✅ DB-backed ✅ ≥3 tests | 5d |
| 3 | Continuous control monitoring engine | ✅ Evaluate each control ✅ Pass/fail/missing ✅ DB-backed ✅ ≥3 tests | 3d |
| 4 | Replace 12 hardcoded compliance handlers | ✅ All use real CCM data ✅ No hardcoded ✅ ≥3 tests | 3d |

### P1 — Framework Mapping + Reports (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Framework-to-control mapping (SOC2 + ISO + NIST) | ✅ Pre-loaded mappings ✅ GGID feature → control ✅ ≥3 tests | 3d |
| 6 | PDF report generation | ✅ Evidence-attached PDF ✅ Audit-ready format ✅ ≥3 tests | 3d |
| 7 | Compliance gap detection API | ✅ Real gaps from CCM ✅ Missing evidence list ✅ ≥3 tests | 2d |
| 8 | Trust center page | ✅ Public compliance status ✅ Framework badges ✅ ≥3 tests | 2d |

### P2 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 9 | Compliance drift detection | Alert when control status changes |
| 10 | Evidence expiry + auto-refresh | Re-collect expired evidence |
| 11 | Multi-framework mapping | Map one evidence → multiple frameworks |
| 12 | Auditor portal | External auditor access for evidence review |

---

## 7. Competitive Differentiation

| Feature | GGID (target) | Vanta | Drata | Secureframe | AuditBoard |
|---------|---------------|-------|-------|-------------|------------|
| **Evidence collection** | **Native (IAM-sourced)** | Integration-based | Integration-based | Integration-based | Manual + integration |
| **CCM** | **Real-time** | Continuous | Continuous | Continuous | Periodic |
| **Frameworks** | SOC2/ISO/NIST | SOC2/ISO/HIPAA | SOC2/ISO/HIPAA | SOC2/ISO | Enterprise |
| **Report generation** | **PDF with evidence** | Yes | Yes | Yes | Yes |
| **Trust center** | **Public page** | Yes | Yes | No | No |
| **IAM-native** | **Yes (unique)** | No (3rd party) | No | No | No |
| **Open source** | **Yes** | No | No | No | No |

**Key differentiator**: GGID collects compliance evidence **natively** from its own IAM operations (audit log, policy decisions, access reviews) — no third-party integration needed. Vanta/Drata collect via external integrations; GGID IS the source of truth.

---

## References

- [AICPA SOC 2 Trust Services Criteria](https://www.aicpa.org/interestareas/frc/assuranceadvisoryservices/trustservices.html) — SOC2 framework
- [ISO/IEC 27001:2022](https://www.iso.org/standard/27001) — ISMS standard
- [NIST SP 800-53 Rev 5](https://nvd.nist.gov/800-53) — Security controls
- [Vanta](https://www.vanta.com/) — Compliance automation platform
- [Drata](https://drata.com/) — Compliance automation
- [GGID Compliance Handlers](../services/audit/internal/server/) — 37 files
- [GGID Compliance Dashboard](../services/audit/internal/server/compliance_dashboard_handler.go) — Hardcoded
- [GGID Evidence Collection](../services/audit/internal/server/evidence_collection_handler.go) — Hardcoded
- [GGID Compliance Mapping](../services/audit/internal/server/compliance_mapping_handler.go) — Hardcoded
