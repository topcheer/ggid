# Data Residency Architecture

Region pinning, replication boundaries, encryption per region, cross-border transfer assessment, legal basis tracking, and data flow diagrams.

## Overview

Data residency ensures user data is stored and processed within specific geographic boundaries to comply with laws like GDPR (EU), PIPL (China), and CCPA (California).

## Region Pinning

### User-to-Region Assignment

```sql
-- At registration, assign user to home region based on geo-IP
INSERT INTO users (id, email, home_region, ...) VALUES (
  'uuid', 'user@corp.com',
  determine_region(client_ip)  -- 'eu-west-1', 'us-east-1', 'cn-north-1'
);
```

### Region Affinity Rules

```yaml
regions:
  eu-west-1:
    countries: [DE, FR, IT, ES, NL, BE, AT, IE, PT, FI, ...]
    data_law: GDPR
    replication: "eu-only"

  us-east-1:
    countries: [US, CA, MX, BR, AR, ...]
    data_law: CCPA/LGPD
    replication: "us-only"

  cn-north-1:
    countries: [CN]
    data_law: PIPL
    replication: "cn-only"

  ap-southeast-1:
    countries: [JP, KR, SG, AU, NZ, ...]
    data_law: APPI/Privacy Act
    replication: "apac-only"
```

Once assigned, user data never leaves its home region without explicit legal basis.

## Replication Boundaries

```
┌─────────┐  NO replication  ┌─────────┐
│ EU-Only │ ←──────────────→ │ US-Only │
│ Region  │                  │ Region  │
└─────────┘                  └─────────┘

┌─────────┐  NO replication  ┌─────────┐
│ CN-Only │ ←──────────────→ │ Any     │
│ Region  │                  │ Region  │
└─────────┘                  └─────────┘
```

### What CAN Cross Regions

| Data | Cross-Region? | Mechanism |
|------|--------------|-----------|
| User PII (email, name, phone) | ❌ Never | Pinned |
| Passwords, MFA secrets | ❌ Never | Pinned |
| Audit logs | ❌ Never | Regional storage |
| Auth config (non-PII) | ✅ Synced | etcd / config store |
| Schema migrations | ✅ Applied | CI/CD per region |
| Public keys (JWKS) | ✅ Replicated | All regions have all keys |

### Data Flow Diagram

```
EU User → EU Gateway → EU Auth → EU PostgreSQL (eu-only)
                                    │
                                    └── Audit events → EU NATS → EU Audit DB

US User → US Gateway → US Auth → US PostgreSQL (us-only)
                                    │
                                    └── Audit events → US NATS → US Audit DB
```

## Per-Region Configuration

### Encryption Keys

Each region has its own encryption keys:

```yaml
encryption:
  eu-west-1:
    kms_key_id: "arn:aws:kms:eu-west-1:.../eu-ggid-key"
    jwt_signing_key: "eu-signing-key"

  us-east-1:
    kms_key_id: "arn:aws:kms:us-east-1:.../us-ggid-key"
    jwt_signing_key: "us-signing-key"
```

A data breach in one region does NOT compromise keys in another.

### JWKS (Cross-Region)

JWKS (public keys) are replicated globally so any region can verify tokens from any region:

```bash
# Each region publishes its public keys
GET https://auth.eu.ggid.dev/.well-known/jwks.json  # EU keys
GET https://auth.us.ggid.dev/.well-known/jwks.json  # US keys

# Federation: global JWKS aggregator
GET https://auth.ggid.dev/.well-known/jwks.json
# → All regions' public keys with kid prefix: "eu-key-1", "us-key-1"
```

## Cross-Border Transfer Assessment

Before transferring data across borders:

```yaml
transfer_assessment:
  trigger: "data_access_request from different region"

  checks:
    - legal_basis:
        - "explicit_consent"           # GDPR Art. 49(1)(a)
        - "standard_contractual_clauses" # GDPR Art. 46
        - "adequacy_decision"           # GDPR Art. 45
        - "government_approval"         # PIPL Art. 38

    - data_minimization:
        - "anonymize"    # Remove PII before transfer
        - "aggregate"    # Only send aggregated stats
        - "pseudonymize" # Hash identifiers

    - documentation:
        - "record_transfer_in_audit"
        - "notify_dpo"
        - "update_data_flow_register"
```

### Transfer Approval Flow

```
Request for cross-region access
    │
    ▼
Check legal basis ──── No basis → DENY
    │ Yes
    ▼
Minimize data (anonymize/aggregate)
    │
    ▼
Log transfer in audit trail
    │
    ▼
Execute transfer
    │
    ▼
Notify Data Protection Officer
```

## Multi-Region Failover

When a region fails, users should NOT be redirected to another region (data residency):

```yaml
failover:
  strategy: "region_isolation"

  eu-west-1_down:
    action: "show_maintenance_page"
    redirect: "DO NOT redirect to us-east-1"
    message: "EU service temporarily unavailable"

  exception:
    # Only if explicit consent for cross-border
    consent_required: true
    fallback_region: "eu-central-1"  # Same legal jurisdiction
```

## Data Localization (Strict Mode)

Some jurisdictions require all processing (not just storage) to happen locally:

```yaml
localization:
  cn-north-1:
    mode: "strict"
    requirements:
      - "data stored in China"
      - "processing in China"
      - "no cross-border replication"
      - "encryption keys in China KMS"
      - "admin access from China only"
```

## Audit Evidence

```bash
# Data residency compliance report
GET /api/v1/admin/compliance/residency?period=2025-Q1
# → {
#   "eu_users_in_eu_storage": 15423,
#   "us_users_in_us_storage": 8721,
#   "cross_border_transfers": 0,
#   "localization_violations": 0
# }
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Cross-region data access | Any → review legal basis |
| User in wrong region | Any → data migration needed |
| Replication boundary breach | Critical → security incident |
| Key access from wrong region | Critical → possible breach |
| Localization violation | Critical → regulatory risk |

## See Also

- [Multi-Region Deployment](multi-region-deployment.md)
- [Privacy by Design](privacy-by-design.md)
- [Compliance Framework Mapping](compliance-framework-mapping.md)
- [Data Classification Implementation](data-classification-implementation.md)
- [Database Security](database-security.md)
