# Data Sovereignty & Residency Compliance

## Overview

Data sovereignty laws require that personal data of citizens stays within their country's borders. This document analyzes GDPR Chapter V (Articles 44-49), CCPA, and other data residency regulations, mapping them to GGID deployment configurations.

> **Related**: [GDPR Compliance](gdpr-compliance.md), [Multi-Region Deployment](multi-region.md), [Data Retention Policy](../guides/data-retention-policy.md)

## GDPR Chapter V: International Transfers

### Article 44: General Principle

Personal data may only be transferred to third countries (outside EEA) if adequate protection is ensured.

### Article 45: Adequacy Decisions

The EU Commission can declare a country "adequate" for data transfers:

| Country | Adequacy Status | Date |
|---------|---------------|------|
| United Kingdom | Yes | 2021 |
| Switzerland | Yes | 2000 |
| Japan | Yes | 2019 |
| South Korea | Yes | 2023 |
| New Zealand | Yes | 2012 |
| Uruguay | Yes | 2012 |
| USA (Data Privacy Framework) | Partial | 2023 (DPF only) |
| Canada (commercial orgs) | Partial | adequacy under review |
| Others | No | — |

### Article 46: Appropriate Safeguards

If no adequacy decision, transfers require safeguards:

| Safeguard | Description | GGID Relevance |
|-----------|-------------|----------------|
| Standard Contractual Clauses (SCCs) | EU-approved contract template | Legal agreement between controller and processor |
| Binding Corporate Rules (BCRs) | Internal rules for multinational corps | Enterprise deployments |
| Codes of Conduct | Industry codes approved by regulator | Sector-specific |
| Certification | Approved certification (e.g., EuroPriSe) | Optional |

### Article 47: Binding Corporate Rules

For multinational companies transferring data between their own entities. Requires DPA approval.

### Articles 48-49: Derogations

Transfers allowed without safeguards only if:
- Explicit consent (49(1)(a))
- Contract performance (49(1)(b))
- Public interest (49(1)(d))
- Legal claims (49(1)(e))
- Vital interests (49(1)(f))

**These are exceptions, not the rule.**

## Other Data Sovereignty Laws

| Law | Region | Key Requirement |
|-----|--------|----------------|
| GDPR | EU/EEA | Data stays in EEA or adequate protection |
| CCPA/CPRA | California | Right to know, delete, opt-out |
| PIPL | China | Data localization + export approval |
| LGPD | Brazil | Similar to GDPR |
| PIPEDA | Canada | Consent + transparency |
| APPI | Japan | Consent + anonymization |
| PDPA | Singapore | Consent + purpose limitation |
| DPDPA | India | Consent + data fiduciary obligations |
| Russian Data Law | Russia | Personal data of Russians stored in Russia |
| Australian Privacy Act | Australia | APP 8 (cross-border disclosure) |

## China PIPL Data Localization

**Requirement**: Personal information of Chinese nationals must be stored on servers located in mainland China.

**Impact**: Multinationals must run separate infrastructure in China or use a local cloud provider (Alibaba Cloud, Tencent Cloud).

**GGID**: Deploy a separate GGID instance in China region. No cross-border data transfer.

## GGID Data Residency Configuration

### Per-Tenant Region Pinning

GGID can pin tenant data to specific database instances:

```yaml
tenants:
  - id: "eu-tenant-uuid"
    region: "eu-west-1"
    database: "ggid-eu.postgres.eu-west-1"
    redis: "redis-eu.eu-west-1"
    nats: "nats-eu.eu-west-1"

  - id: "us-tenant-uuid"
    region: "us-east-1"
    database: "ggid-us.postgres.us-east-1"
    redis: "redis-us.us-east-1"
    nats: "nats-us.us-east-1"
```

### Gateway Region Routing

```go
func routeTenant(tenantID string) string {
    region := tenantRegionMap[tenantID]
    if region == "" {
        region = defaultRegion
    }
    return region
}
// Request for EU tenant → routed to EU gateway → EU database
```

### Encryption at Rest

All data residency regions enforce:
- PostgreSQL: Transparent Data Encryption (TDE) or disk encryption
- Redis: disk encryption
- NATS: file encryption
- Backups: encrypted with region-specific keys

## Compliance Checklist

- [ ] Data residency requirement documented per tenant
- [ ] Database deployed in correct geographic region
- [ ] No cross-region replication for EU/China data
- [ ] SCCs signed with all processors
- [ ] Data transfer impact assessment (TIA) completed
- [ ] Encryption at rest enabled
- [ ] Audit logs stored in-region
- [ ] Backup strategy respects residency
- [ ] DNS routing configured per tenant region
- [ ] Certificate per region

## Multi-Region Data Flow

```
User (EU) → EU Gateway → EU Database (EU data stays in EU)
User (US) → US Gateway → US Database (US data stays in US)
User (CN) → CN Gateway → CN Database (CN data stays in CN)

Global: Only non-personal metadata (tenant config, schema) replicates
```

## References

- [GDPR Chapter V](https://gdpr-info.eu/chapter-5/)
- [EU-US Data Privacy Framework](https://www.dataprivacyframework.gov/)
- [EDPB Transfer Guidelines](https://edpb.europa.eu/)

## See Also

- [GDPR Compliance](gdpr-compliance.md)
- [Multi-Region Deployment](multi-region.md)
- [Data Retention Policy](../guides/data-retention-policy.md)
