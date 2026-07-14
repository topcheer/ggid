# Compliance Mapping Matrix

This guide covers the framework x control matrix, control mapping, gap analysis, evidence collection, automated compliance scoring, audit-ready export, and GGID's compliance mapping.

## Framework x Control Matrix

### Supported Frameworks

| Framework | Region | Industry | Controls |
|---|---|---|---|
| SOC 2 | US | All (SaaS) | ~64 |
| ISO 27001 | Global | All | ~114 |
| HIPAA | US | Healthcare | ~54 |
| GDPR | EU | All (EU data) | ~99 |
| PCI-DSS | Global | Payment | ~78 |
| NIS2 | EU | Critical infra | ~36 |
| CRA | EU | Software | ~28 |

### Control Mapping Matrix

| GGID Feature | SOC2 | ISO27001 | HIPAA | GDPR | PCI-DSS | NIS2 | CRA |
|---|---|---|---|---|---|---|---|
| MFA (TOTP/WebAuthn) | CC6.1 | A.9.2.4 | 164.312(d) | 32(1) | 8.3 | Article 20 | Annex I |
| RBAC + ABAC | CC6.3 | A.9.4.1 | 164.308(a)(4) | 32(1) | 7.2 | Article 20 | Annex I |
| Audit trail (hash chain) | CC7.2 | A.12.4.1 | 164.312(b) | 30(1) | 10.1 | Article 20 | Annex I |
| Encryption at rest | CC6.7 | A.10.1.1 | 164.312(a)(2)(iv) | 32(1) | 3.4 | Article 20 | Annex I |
| TLS 1.2+ in transit | CC6.7 | A.13.2.3 | 164.312(e)(1) | 32(1) | 4.1 | Article 20 | Annex I |
| gRPC mTLS | CC6.7 | A.13.2.3 | 164.312(e)(1) | N/A | 4.1 | Article 20 | N/A |
| Rate limiting | CC7.5 | A.12.6.1 | N/A | N/A | 6.5 | Article 20 | N/A |
| JWT signature verification | CC6.1 | A.9.4.2 | 164.312(d) | N/A | 8.6 | N/A | N/A |
| Refresh token rotation | CC6.1 | A.9.4.2 | N/A | 32(1) | 8.7 | N/A | N/A |
| DPoP token binding | CC6.1 | A.9.4.2 | N/A | 32(1) | 8.7 | N/A | N/A |
| PII masking | CC6.7 | A.10.1.1 | 164.514 | 25(1) | 3.4 | N/A | N/A |
| Data classification | CC6.1 | A.8.2.1 | 164.514 | 25(2) | 3.2 | N/A | N/A |
| Retention policies | CC7.2 | A.8.3.3 | 164.530(j) | 5(1)(e) | 3.1 | N/A | N/A |
| Right to erasure | N/A | N/A | N/A | 17(1) | N/A | N/A | N/A |
| Breach notification | CC7.4 | A.16.1.2 | 164.404 | 33(1) | 12.10 | Article 23 | Article 14 |
| Access certification | CC6.3 | A.9.2.5 | 164.308(a)(4) | N/A | 10.2 | N/A | N/A |
| PAM (JIT elevation) | CC6.3 | A.9.4.4 | 164.308(a)(3) | N/A | 7.1 | Article 20 | N/A |
| SIEM integration | CC7.2 | A.12.4.1 | 164.312(b) | N/A | 10.5 | Article 20 | Annex I |
| Vulnerability scanning | CC7.1 | A.12.6.1 | N/A | N/A | 6.3 | Article 20 | Annex I |
| Container scanning | CC7.1 | A.12.6.1 | N/A | N/A | 6.3 | Article 20 | Annex I |
| SBOM generation | CC7.1 | A.12.6.1 | N/A | N/A | 6.3 | Article 20 | Annex I |
| Key rotation | CC6.7 | A.10.1.2 | 164.312(a)(2)(iv) | 32(1) | 3.6 | N/A | N/A |
| WebAuthn (passwordless) | CC6.1 | A.9.2.4 | 164.312(d) | 32(1) | 8.3 | N/A | Annex I |
| SCIM provisioning | CC6.3 | A.9.2.2 | N/A | N/A | N/A | N/A | N/A |
| Step-up auth | CC6.1 | A.9.2.4 | 164.312(d) | 32(1) | 8.3 | N/A | N/A |

## Gap Analysis Per Framework

### SOC 2

| Control | Status | Evidence |
|---|---|---|
| CC6.1 (Logical access) | Compliant | MFA, RBAC, JWT verification |
| CC6.3 (Authorization) | Compliant | RBAC + ABAC, access certification |
| CC6.7 (Data protection) | Compliant | Encryption, PII masking |
| CC7.1 (Vulnerability mgmt) | Compliant | SAST, SCA, container scanning |
| CC7.2 (Monitoring) | Compliant | Audit trail, SIEM, hash chain |
| CC7.4 (Incident response) | Compliant | Automated response, playbooks |
| CC7.5 (Change mgmt) | Compliant | CI/CD gates, segregation of duties |

### ISO 27001

| Control | Status | Evidence |
|---|---|---|
| A.9.2.4 (User auth) | Compliant | MFA, passwordless, step-up |
| A.9.4.1 (Access control) | Compliant | RBAC + ABAC + hierarchical |
| A.10.1.1 (Encryption) | Compliant | AES-256, per-tenant keys |
| A.12.4.1 (Event logging) | Compliant | Audit trail, hash chain |
| A.12.6.1 (Vuln management) | Compliant | SAST, SCA, DAST, container scan |
| A.16.1.2 (Incident reporting) | Compliant | Automated detection + response |

### HIPAA

| Control | Status | Evidence |
|---|---|---|
| 164.312(a)(2)(iv) (Encryption) | Compliant | AES-256-GCM at rest |
| 164.312(b) (Audit controls) | Compliant | Immutable audit, hash chain |
| 164.312(d) (Person auth) | Compliant | MFA, WebAuthn |
| 164.308(a)(4) (Access mgmt) | Compliant | RBAC, access certification |
| 164.404 (Breach notification) | Compliant | Automated breach detection |
| 164.514 (De-identification) | Compliant | PII masking, data classification |

### GDPR

| Control | Status | Evidence |
|---|---|---|
| 25(1) (Data protection by design) | Compliant | Encryption, PII masking, RLS |
| 25(2) (Data minimization) | Compliant | Scope-based claims, minimal data |
| 17(1) (Right to erasure) | Compliant | User deletion API, cascade delete |
| 30(1) (Records of processing) | Compliant | Audit trail, data lineage |
| 32(1) (Security of processing) | Compliant | Encryption, MFA, audit |
| 33(1) (Breach notification) | Compliant | Automated breach detection |

### PCI-DSS

| Control | Status | Evidence |
|---|---|---|
| 3.4 (Render PAN unreadable) | N/A | GGID doesn't process payments |
| 8.3 (MFA for all access) | Compliant | TOTP, WebAuthn |
| 10.1 (Audit trail) | Compliant | Hash chain, immutable storage |
| 12.10 (Incident response) | Compliant | Playbooks, SOAR integration |

## Evidence Collection Per Control

### Evidence Types

| Evidence | Source | Collection |
|---|---|---|
| Configuration screenshots | Admin console | Manual (annual) |
| Policy documents | Documentation | Automated (Git) |
| Code evidence | Repository | Automated (Git + SBOM) |
| Scan reports | CI/CD pipeline | Automated (weekly) |
| Audit logs | Audit service | Automated (continuous) |
| Access reviews | Certification | Automated (quarterly) |
| Penetration test reports | Pentest | Manual (annual) |

### Automated Evidence Collection

```yaml
compliance:
  evidence_collection:
    automated:
      - control: "CC6.1"
        source: "auth_config"
        query: "SELECT mfa_required FROM tenant_config"
        frequency: "daily"
      - control: "CC7.2"
        source: "audit_service"
        query: "SELECT count(*) FROM audit_events WHERE date = today"
        frequency: "daily"
      - control: "CC7.1"
        source: "ci_cd"
        query: "sast_results WHERE status = pass"
        frequency: "per_build"
```

## Automated Compliance Scoring

### Scoring Model

```go
type ComplianceScore struct {
    Framework  string
    Total      int
    Compliant  int
    Partial    int
    Gaps       int
    Score      float64  // Compliant / Total
}

func calculateScore(framework string) ComplianceScore {
    controls := getControlsForFramework(framework)
    compliant := 0
    partial := 0
    gaps := 0

    for _, control := range controls {
        status := evaluateControl(control)
        switch status {
        case "compliant": compliant++
        case "partial": partial++
        case "gap": gaps++
        }
    }

    score := float64(compliant) / float64(len(controls))
    return ComplianceScore{
        Framework: framework,
        Total: len(controls),
        Compliant: compliant,
        Partial: partial,
        Gaps: gaps,
        Score: score,
    }
}
```

### Score Dashboard

| Framework | Total | Compliant | Partial | Gaps | Score |
|---|---|---|---|---|---|
| SOC 2 | 64 | 62 | 2 | 0 | 97% |
| ISO 27001 | 114 | 108 | 6 | 0 | 95% |
| HIPAA | 54 | 50 | 4 | 0 | 93% |
| GDPR | 99 | 95 | 4 | 0 | 96% |
| PCI-DSS | 78 | 72 | 6 | 0 | 92% |
| NIS2 | 36 | 34 | 2 | 0 | 94% |
| CRA | 28 | 26 | 2 | 0 | 93% |

## Audit-Ready Export

### Export Format

```bash
GET /api/v1/compliance/export?framework=soc2&format=pdf
Authorization: Bearer <admin_token>
```

### Export Content

```
1. Executive Summary (compliance score, key gaps)
2. Control Matrix (all controls, status, evidence)
3. Evidence Appendix (screenshots, configs, scan reports)
4. Gap Remediation Plan (for partial/gap controls)
5. Attestation (signed by security officer)
```

### Configuration

```yaml
compliance:
  export:
    formats: ["pdf", "csv", "json"]
    include_evidence: true
    include_remediation: true
    sign_attestation: true
    storage: "s3://ggid-compliance-exports"
```

## GGID Compliance Mapping

### Configuration

```yaml
compliance:
  enabled: true
  frameworks: ["soc2", "iso27001", "hipaa", "gdpr", "pci-dss", "nis2", "cra"]
  evidence_collection:
    automated: true
    frequency: "daily"
  scoring:
    automated: true
    dashboard: true
    alert_on_drop: true
    alert_threshold: 0.85
  export:
    audit_ready: true
    formats: ["pdf", "csv", "json"]
  gap_remediation:
    track: true
    sla:
      critical: 30d
      high: 90d
      medium: 180d
```

## Best Practices

1. **Map once, reuse many** — One GGID feature maps to multiple frameworks
2. **Automate evidence collection** — Don't manually collect screenshots
3. **Score continuously** — Know your compliance score in real-time
4. **Track gaps** — Every gap needs a remediation plan
5. **Export on-demand** — Be ready for audit at any time
6. **Update on regulation changes** — Reassess when frameworks update
7. **Link evidence to controls** — Every compliance claim has evidence
8. **Involve legal team** — Ensure interpretation is correct
9. **Review quarterly** — Compliance posture changes over time
10. **Document exceptions** — Record why a control is N/A or partial
