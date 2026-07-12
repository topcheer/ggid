# SOC 2 Audit Preparation Guide

This guide covers preparing for a SOC 2 Type II audit with GGID — trust service criteria, evidence collection, control mapping, gap remediation, auditor liaison, and timeline.

## Trust Service Criteria Mapping

### CC6: Logical and Physical Access

| Control | GGID Feature | Evidence |
|---------|-------------|----------|
| CC6.1: User identification | JWT with sub claim | Auth config, sample JWTs |
| CC6.2: User authentication | Password + MFA | MFA enrollment rates, policy config |
| CC6.3: Access authorization | RBAC + ABAC | Role assignment export, policy config |
| CC6.4: Access restriction | Per-request scope check | Gateway middleware config |
| CC6.5: Access removal | User delete/lock + session revoke | Termination procedure, revoke audit log |
| CC6.8: Access points | Gateway as sole entry | Network diagram, firewall rules |

### CC7: System Operations

| Control | GGID Feature | Evidence |
|---------|-------------|----------|
| CC7.1: Infrastructure management | Docker/K8s manifests | Deployment YAMLs |
| CC7.2: Incident detection | Audit alerts + SIEM | Alert config, SIEM forwarder logs |
| CC7.3: Monitoring | Health endpoints + metrics | Monitoring dashboard, uptime reports |
| CC7.4: Anomaly handling | Rate limit + circuit breaker | Config + alert history |

### CC8: Change Management

| Control | GGID Feature | Evidence |
|---------|-------------|----------|
| CC8.1: Authorize changes | Audit trail for all config changes | Git history + audit log |

### CC9: Risk Mitigation

| Control | GGID Feature | Evidence |
|---------|-------------|----------|
| CC9.1: Risk identification | Threat model | Threat modeling doc |
| CC9.2: Vendor risk | Dependency inventory | go.mod, SBOM |

## Evidence Collection

### Automated Evidence

```bash
# 1. Access review evidence (quarterly)
curl https://api.ggid.example.com/api/v1/audit/compliance/report?type=soc2 \
  -G --data-urlencode "start_date=2025-01-01" \
      --data-urlencode "end_date=2025-03-31" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o soc2_q1_2025.json

# 2. Audit integrity proof
curl https://api.ggid.example.com/api/v1/audit/integrity/verify \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o audit_integrity.json

# 3. User access export
curl https://api.ggid.example.com/api/v1/users/export?format=csv \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o user_access_q1.csv

# 4. Role assignments
curl https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o roles.json

# 5. Audit log export
https://api.ggid.example.com/api/v1/audit/export?format=csv&start_date=2025-01-01 \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -o audit_log_q1.csv
```

### Evidence Binder Structure

```
soc2-audit-2025/
├── access-reviews/
│   ├── q1-access-review.json
│   ├── q2-access-review.json
│   └── quarterly-summary.md
├── audit-logs/
│   ├── q1-audit-export.csv
│   ├── q2-audit-export.csv
│   └── integrity-verification.json
├── policies/
│   ├── password-policy.yaml
│   ├── mfa-enrollment-report.json
│   └── access-control-policy.md
├── incidents/
│   ├── incident-response-plan.md
│   └── incident-log-2025.md
├── infrastructure/
│   ├── deployment-manifests/
│   ├── network-diagram.png
│   └── backup-test-results.md
└── change-management/
    ├── git-change-log.txt
    └── change-approval-process.md
```

## Gap Remediation

### Common Gaps & Fixes

| Gap | SOC 2 Impact | Remediation |
|-----|-------------|-------------|
| No MFA enforcement | CC6.2 | Enforce MFA for all admins |
| No access reviews | CC6.3 | Quarterly certification campaigns |
| No change approval | CC8.1 | PR approval workflow |
| No incident plan | CC7.4 | Document IR playbook |
| No backup testing | CC9.1 | Monthly restore drill |
| No vendor assessment | CC9.2 | Annual vendor risk review |

## Auditor Liaison

### Pre-Audit (4-6 weeks)

1. Select auditor (CPA firm with SOC 2 experience)
2. Define scope (systems, locations, period)
3. Provide evidence binder
4. Schedule walkthroughs

### Fieldwork (2-4 weeks)

1. Control walkthroughs (live demos)
2. Evidence sampling (auditor selects items)
3. Interviews with personnel
4. Observation of operations

### Reporting (2-4 weeks)

1. Draft report review
2. Management responses
3. Final report issued

## Timeline

```
Month 1-2: Gap assessment + remediation
Month 3:   Evidence collection (quarter)
Month 4:   Pre-audit prep + auditor selection
Month 5:   Fieldwork
Month 6:   Report

Total: ~6 months for first SOC 2 Type II
```

## Checklist

- [ ] Scope defined (systems, period)
- [ ] Gap assessment completed
- [ ] All gaps remediated
- [ ] Evidence binder assembled
- [ ] Access reviews conducted quarterly
- [ ] Backup restore tested
- [ ] Incident response plan documented
- [ ] Change management process documented
- [ ] Auditor selected
- [ ] Walkthrough demos prepared

## See Also

- [Compliance Guide](compliance-guide.md)
- [Security Audit Checklist](security-audit-checklist.md)
- [Access Reviews](access-reviews.md)
- [Threat Modeling](threat-modeling.md)
