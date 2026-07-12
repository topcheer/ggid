# EU CRA Compliance

Security-by-design, vulnerability disclosure, SBOM, product lifecycle, incident reporting, CE marking, GGID checklist, and NIS2 alignment.

## Overview

The EU Cyber Resilience Act (CRA) mandates cybersecurity requirements for products with digital elements placed on the EU market.

## Security-by-Design Requirements

| Requirement | CRA Article | GGID Implementation |
|-------------|------------|---------------------|
| Secure by design | Art. 13 | Threat modeling per service, secure SDLC |
| Secure by default | Art. 13 | MFA on by default, least privilege |
| No known vulnerabilities | Art. 13 | Trivy scanning in CI, dependency updates |
| Secure update mechanism | Art. 13 | Rolling updates, signed images (Cosign) |
| Vulnerability disclosure | Art. 14 | security@ggid.dev, 90-day disclosure |

## SBOM (Software Bill of Materials)

```bash
# Generate CycloneDX SBOM
POST /api/v1/admin/compliance/sbom/generate
{"format": "cyclonedx", "version": "1.5"}
# → {
#   "bomFormat": "CycloneDX",
#   "specVersion": "1.5",
#   "components": [
#     {"type": "library", "name": "pgx", "version": "5.5.1", "purl": "pkg:golang/github.com/jackc/pgx@v5.5.1"},
#     ...
#   ]
# }
```

### SBOM Requirements

| Field | Required |
|-------|----------|
| Component name | ✅ |
| Version | ✅ |
| Supplier | ✅ |
| PURL (package URL) | ✅ |
| License | ✅ |
| Known vulnerabilities | ✅ (VEX) |

## Product Lifecycle Obligations

| Phase | Obligation | Duration |
|-------|-----------|----------|
| Placement on market | CE marking, DoC, tech documentation | — |
| Support & updates | Security patches | Product lifetime + 5 years |
| Vulnerability monitoring | Continuous CVE tracking | Active support period |
| EOL notification | Notify users 12 months before EOL | — |

## Incident Reporting Timeline

| Event | Deadline | Recipient |
|-------|---------|-----------|
| Active exploitation discovered | 24 hours | EU CSIRT |
| Severe vulnerability | 24 hours | ENISA |
| Full incident report | 72 hours | EU CSIRT |
| Patch available | Notify within 15 days | Users + ENISA |

```bash
POST /api/v1/admin/compliance/cra/incident-report
{
  "incident_id": "INC-2025-0001",
  "severity": "severe",
  "affected_products": ["ggid-2.5.x"],
  "cve": "CVE-2025-1234",
  "patch_available": true,
  "report_type": "initial"
}
# → Submitted to ENISA portal
```

## CE Marking

Requirements for CE mark:
1. Conformity assessment (internal or notified body)
2. Technical documentation (architecture, security measures, SBOM)
3. EU Declaration of Conformity
4. CE mark on product/packaging/documentation

## GGID Compliance Checklist

| # | Requirement | Status | Evidence |
|---|------------|--------|----------|
| 1 | Security-by-design process | ✅ | Threat models per service |
| 2 | Secure SDLC (CI scanning) | ✅ | Trivy + gitleaks in pipeline |
| 3 | SBOM generation | ✅ | CycloneDX per release |
| 4 | Vulnerability disclosure policy | ✅ | security@ggid.dev + 90-day SLA |
| 5 | Incident reporting procedure | ✅ | 24h/72h timeline documented |
| 6 | Secure update mechanism | ✅ | Signed images, rolling deploys |
| 7 | Support period documented | ✅ | 5 years post-release |
| 8 | CE marking process | ⚠️ | Pending conformity assessment |
| 9 | Tech documentation maintained | ✅ | 474+ docs in docs/ |
| 10 | NIS2 alignment | ✅ | See below |

## NIS2 Alignment

| NIS2 Requirement | CRA Overlap | GGID |
|-----------------|------------|------|
| Risk management | ✅ Shared | Threat modeling |
| Incident handling | ✅ Shared | 24h/72h reporting |
| Supply chain security | ✅ Shared | SBOM + vendor assessment |
| Vulnerability disclosure | ✅ Shared | 90-day policy |
| Security training | NIS2 specific | Annual training |
| Business continuity | NIS2 specific | DR drills, RTO/RPO |

## Monitoring

| Metric | Alert |
|--------|-------|
| Unpatched critical CVE | >24h → CRA violation risk |
| SBOM staleness | >30 days since last generation |
| Incident report deadline | <24h remaining → urgent |
| Support period ending | <12 months → notify users |

## See Also

- [Compliance Framework Mapping](compliance-framework-mapping.md)
- [Compliance Automation](compliance-automation.md)
- [Security Hardening Checklist](security-hardening-checklist.md)
- [Incident Response Playbook](incident-response-playbook.md)