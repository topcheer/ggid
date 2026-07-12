# Compliance Framework Mapping

Mapping GGID controls to SOC 2, ISO 27001, GDPR, CCPA, HIPAA, and PIPL requirements.

## Overview

This matrix maps each compliance control to its GGID implementation, evidence location, and gap status. Use this for audit preparation and gap remediation.

## SOC 2 (Type II)

| Control | GGID Implementation | Evidence | Status |
|---------|-------------------|----------|--------|
| CC6.1: Logical access | RBAC + ABAC enforcement, JWT scopes | `pkg/auth`, `pkg/policy` | ✅ Implemented |
| CC6.2: User authentication | Password + MFA + WebAuthn + LDAP/OAuth/SAML | `services/auth` | ✅ Implemented |
| CC6.3: Access revocation | Token revocation (RFC 7009), session kill | `services/auth/internal/server` | ✅ Implemented |
| CC6.6: Boundary protection | API gateway, mTLS between services, WAF rules | `services/gateway` | ✅ Implemented |
| CC6.7: Transmission encryption | TLS 1.3 external, mTLS internal | `pkg/transport` | ✅ Implemented |
| CC6.8: Encryption at rest | PostgreSQL TDE, pgcrypto columns, backup AES-256 | `deploy/` | ✅ Implemented |
| CC7.1: System monitoring | Structured logging, OpenTelemetry tracing, metrics | `pkg/healthcheck` | ✅ Implemented |
| CC7.2: Anomaly detection | Rate limiting, brute force detection, risk scoring | `services/gateway/internal/middleware` | ✅ Implemented |
| CC7.3: Event logging | NATS audit events, 7-year retention | `services/audit` | ✅ Implemented |
| CC7.4: Incident response | Alerting, break-glass, SIEM forwarder | `pkg/audit/publisher` | ✅ Implemented |
| CC8.1: Change management | CI/CD pipeline, code review, deployment gating | `.github/workflows/` | ✅ Implemented |
| CC9.1: Risk assessment | Threat modeling docs, pentest reports | `docs/research/threat-modeling.md` | ✅ Implemented |

## ISO 27001

| Control | GGID Implementation | Evidence | Status |
|---------|-------------------|----------|--------|
| A.5.15: Access control | RBAC + ABAC, JIT elevation, delegated admin | `services/policy`, `docs/guides/delegated-administration.md` | ✅ |
| A.5.16: Identity management | Identity lifecycle automation, SCIM provisioning | `services/identity`, `docs/guides/identity-lifecycle-automation.md` | ✅ |
| A.5.17: Authentication info | Password pepper, HSM-stored keys, secret rotation | `pkg/crypto`, `docs/guides/secrets-rotation-automation.md` | ✅ |
| A.5.18: Access rights | Quarterly access reviews, automated dormancy detection | `services/audit` | ✅ |
| A.5.23: Cloud services | Data residency controls, multi-region deployment | `docs/guides/multi-region-deployment.md` | ✅ |
| A.5.34: Privacy & PII | PII obfuscation, data minimization, consent mgmt | `pkg/pii` | ✅ Implemented (wiring pending) |
| A.6.3: Cryptography | RS256/ES256 JWT, AES-256-GCM, mTLS, HSM integration | `pkg/crypto` | ✅ |
| A.8.2: Privileged access | Admin scope hierarchy, dual-control break-glass | `docs/guides/delegated-administration.md` | ✅ |
| A.8.5: Secure auth | Passwordless, WebAuthn, MFA step-up, adaptive auth | `docs/guides/passwordless-auth-architecture.md` | ✅ |
| A.8.16: Monitoring | Audit hash chain, SIEM forwarding, anomaly alerts | `services/audit` | ✅ |

## GDPR

| Article | Requirement | GGID Implementation | Status |
|---------|------------|---------------------|--------|
| Art. 6 | Lawful basis tracking | Consent management endpoints, audit trail | ✅ |
| Art. 7 | Consent withdrawal | `DELETE /api/v1/consent/{id}` | ✅ |
| Art. 15 | Right of access | `GET /api/v1/identity/users/{id}?expand=all` | ✅ |
| Art. 16 | Right to rectification | `PATCH /api/v1/identity/users/{id}` | ✅ |
| Art. 17 | Right to erasure | `DELETE /api/v1/identity/users/{id}` (anonymize) | ✅ |
| Art. 18 | Right to restriction | `PATCH /api/v1/identity/users/{id} {"status":"restricted"}` | ✅ |
| Art. 20 | Data portability | `GET /api/v1/identity/users/{id}/export` (JSON) | ✅ |
| Art. 25 | Privacy by design | PII obfuscation, data minimization, pairwise sub | ✅ |
| Art. 32 | Security of processing | TLS, encryption at rest, RBAC, MFA | ✅ |
| Art. 33 | Breach notification | SIEM alerts, webhook to compliance team | ✅ |
| Art. 35 | DPIA | `docs/research/dpia-template.md` | ✅ |

## CCPA / CPRA

| Requirement | GGID Implementation | Status |
|------------|---------------------|--------|
| Right to know | Data export API | ✅ |
| Right to delete | Erasure endpoint (anonymize + retain audit hash) | ✅ |
| Right to opt-out | Consent management + global privacy control header | ✅ |
| Right to correct | User self-service + admin patch | ✅ |
| Sensitive PI limitation | PII classification + column-level encryption | ✅ |
| Service provider agreements | Webhook-based data flow agreements, audit trail | ✅ |

## HIPAA

| Safeguard | GGID Implementation | Status |
|-----------|---------------------|--------|
| Access control (164.312(a)) | RBAC + ABAC, unique user ID, auto logoff | ✅ |
| Audit controls (164.312(b)) | Comprehensive audit logging, 7-year retention | ✅ |
| Integrity (164.312(c)) | Audit hash chain, tamper-evident logs | ✅ |
| Person authentication (164.312(d)) | MFA, adaptive auth, WebAuthn | ✅ |
| Transmission security (164.312(e)) | TLS 1.3, mTLS internal | ✅ |
| Encryption (164.312(a)(2)(iv)) | AES-256 at rest, pgcrypto for PHI columns | ✅ |
| BAA readiness | Audit data classification, access logging | ✅ |

## PIPL (China)

| Requirement | GGID Implementation | Status |
|------------|---------------------|--------|
| Consent (Art. 13-16) | Granular consent management | ✅ |
| Data minimization (Art. 6) | Scope-based attribute release | ✅ |
| Cross-border transfer (Art. 38-42) | Data residency: CN region, no cross-border replication | ✅ |
| Right to delete (Art. 47) | Erasure endpoint | ✅ |
| Security assessment (Art. 55) | Threat modeling, pentest reports | ✅ |

## Audit Readiness Checklist

- [ ] Access reviews completed (quarterly)
- [ ] Vulnerability scan reports (monthly)
- [ ] Penetration test (annual)
- [ ] Incident response plan tested
- [ ] Change management logs reviewed
- [ ] Data retention policy enforced (automated)
- [ ] Encryption key rotation documented
- [ ] Backup restore tested (quarterly)
- [ ] Privacy policy up to date
- [ ] DPIA completed for new features

## Evidence Collection

```bash
# Automated evidence export for audit
ggid audit export --framework soc2 --period Q4-2024
# → ZIP with: access logs, config snapshots, review reports, rotation records

ggid audit export --framework gdpr --period 2024
# → ZIP with: consent records, DSAR logs, breach register, DPIAs
```

## Gap Remediation

| Gap | Framework | Priority | Status |
|-----|-----------|----------|--------|
| PII obfuscation not wired | ISO 27001 A.5.34 | P1 | In progress |
| gRPC TLS all services | SOC 2 CC6.6 | P0 | ✅ Closed |
| Audit hash chain | HIPAA 164.312(c) | P1 | ✅ Implemented |
| Introspection auth | SOC 2 CC6.1 | P0 | ✅ Closed |

## See Also

- [SOC2 Audit Preparation](soc2-audit-prep.md)
- [GDPR Compliance](gdpr-compliance.md)
- [Penetration Testing](penetration-testing.md)
- [Threat Modeling](../research/threat-modeling.md)
- [Data Retention Policy](data-retention-policy.md)
