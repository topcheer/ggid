# Data Sovereignty

Data residency by region, cross-border transfer mechanisms, localization mandates, sovereign cloud, data flow mapping, and tenant data partitioning.

## Regional Requirements

| Region | Law | Key Requirements |
|--------|-----|-----------------|
| EU | GDPR | Data stays in EU, SCCs for transfer, right to erasure |
| China | PIPL | Data stored in China, government approval for export |
| Russia | 152-FZ | Personal data stored in Russia |
| Brazil | LGPD | Data stored in Brazil or adequate countries |
| India | DPDP 2023 | Consent-based, data localization for certain categories |

## Cross-Border Transfer Mechanisms

| Mechanism | Legal Basis | Use Case |
|-----------|------------|---------|
| Adequacy decision | GDPR Art. 45 | EU→US (Data Privacy Framework) |
| Standard Contractual Clauses | GDPR Art. 46 | EU→non-adequate countries |
| Binding Corporate Rules | GDPR Art. 47 | Intra-group transfers |
| Explicit consent | GDPR Art. 49(1)(a) | One-time transfer |
| Government approval | PIPL Art. 38 | China cross-border |

## Data Localization Mandates

```yaml
localization:
  cn-north-1:
    mode: strict
    requirements:
      - data_stored_in_china: true
      - processing_in_china: true
      - no_cross_border_replication: true
      - encryption_keys_in_china: true
      - admin_access_from_china_only: true

  eu-west-1:
    mode: moderate
    requirements:
      - data_stored_in_eu: true
      - scc_for_transfer: required
      - encryption_keys_in_eu: true
```

## Tenant Data Partitioning by Geography

```sql
-- User assigned to home region at registration
INSERT INTO users (id, email, home_region) VALUES (
  'uuid', 'user@corp.com', determine_region(client_ip)
);

-- RLS ensures data never crosses region boundary
CREATE POLICY region_isolation ON users
  USING (home_region = current_setting('app.region'));
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Cross-region data access | Any → review legal basis |
| Localization violation | Critical → regulatory risk |
| Transfer without SCC | Any → legal review |

## See Also

- [Data Residency Architecture](data-residency-architecture.md)
- [Privacy by Design](privacy-by-design.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
