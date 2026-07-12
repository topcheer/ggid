# Data Governance Framework

This guide covers data ownership roles, classification policy, lineage tracking, quality metrics, lifecycle management, sharing agreements, sovereignty compliance, governance council, and GGID's implementation.

## Data Ownership Roles

### Role Hierarchy

| Role | Responsibility | Who |
|---|---|---|
| Data Owner | Accountable for data, approves access | Business unit head |
| Data Steward | Manages data quality, metadata, policies | Data team |
| Data Custodian | Technical storage, security, backup | IT/Ops |
| Data User | Consumes data per approved access | Employees |

### Responsibilities Matrix

| Activity | Owner | Steward | Custodian | User |
|---|---|---|---|---|
| Classify data | Approves | Defines | Implements | N/A |
| Grant access | Approves | Recommends | Executes | Requests |
| Quality rules | Approves | Defines | Implements | Reports issues |
| Retention policy | Approves | Defines | Executes | N/A |
| Encryption | Approves | Specifies | Implements | N/A |
| Backup | Approves | Specifies | Executes | N/A |

## Data Classification Policy

### 4-Tier Model

| Tier | Owner Approval for Access | Encryption | Retention |
|---|---|---|---|
| Restricted | Data Owner + Security | AES-256 + HSM keys | 7y (regulated) |
| Confidential | Data Owner | AES-256 + KMS keys | 3y |
| Internal | Steward | AES-128 | 1y |
| Public | Steward | Optional | Indefinite |

### Classification Assignment

```yaml
data_governance:
  classification:
    auto_classify: true
    rules:
      - field: "ssn"
        tier: "restricted"
      - field: "email"
        tier: "confidential"
      - field: "phone"
        tier: "confidential"
      - field: "department"
        tier: "internal"
      - field: "company_name"
        tier: "public"
    review_cycle: 90d
```

## Data Lineage Tracking

### What is Lineage?

Data lineage tracks the flow of data from origin through transformations to consumption:

```
Source: LDAP → GGID User Import → DB users table → API /users → Console display
                                                     ↓
                                                  Audit log
```

### Lineage Metadata

```json
{
  "data_element": "user.email",
  "source": "LDAP (mail attribute)",
  "transformations": [
    {"step": "import", "function": "lowercase", "system": "scim-service"},
    {"step": "store", "function": "encrypt", "system": "identity-service"},
    {"step": "release", "function": "scope_filter", "system": "oauth-service"}
  ],
  "consumers": [
    {"system": "console", "access": "read", "scope": "email"},
    {"system": "audit", "access": "masked", "scope": "pii"}
  ]
}
```

### Lineage Tracking Implementation

```go
type DataLineage struct {
    ElementID    string
    Source       string
    Transformations []Transformation
    Consumers    []Consumer
    UpdatedAt    time.Time
}

func RecordLineage(element string, transform Transformation) {
    lineage := getOrCreateLineage(element)
    lineage.Transformations = append(lineage.Transformations, transform)
    lineage.UpdatedAt = time.Now()
    save(lineage)
}
```

## Data Quality Metrics

### Quality Dimensions

| Dimension | Metric | Target |
|---|---|---|
| Completeness | % non-null fields | >99% |
| Accuracy | % verified against source | >98% |
| Consistency | % consistent across systems | >99% |
| Timeliness | Avg sync latency | <5 min |
| Uniqueness | Duplicate record rate | <0.1% |
| Validity | % valid format | >99.5% |

### Quality Dashboard

```yaml
data_governance:
  quality:
    metrics:
      completeness:
        query: "SELECT count(null) / count(*) FROM users WHERE email IS NULL"
        target: 0.99
        alert_below: 0.95
      accuracy:
        query: "SELECT count(*) FROM users WHERE email_verified = false"
        target: 0.98
      uniqueness:
        query: "SELECT email, count(*) FROM users GROUP BY email HAVING count(*) > 1"
        target: 0.999
    reporting:
      frequency: "weekly"
      recipients: ["data-stewards", "data-owner"]
```

## Data Lifecycle Management

### Lifecycle Stages

```
Create → Store → Use → Share → Archive → Destroy
  ↑                                    ↓
  └────────── Retention Policy ────────┘
```

### Stage Management

| Stage | Actions | Owner |
|---|---|---|
| Create | Validate, classify, encrypt | Steward |
| Store | Encrypt at rest, access control | Custodian |
| Use | Audit access, mask PII | Custodian |
| Share | Agreement check, DLP scan | Owner + Steward |
| Archive | Move to cold storage, retain | Custodian |
| Destroy | Crypto-erase, verify, audit | Owner + Custodian |

## Data Sharing Agreements

### Agreement Structure

```yaml
data_governance:
  sharing:
    agreements:
      - name: "HR-IT Data Sharing"
        provider: "HR Department"
        consumer: "IT Department"
        data: ["employee_id", "department", "manager"]
        purpose: "User provisioning"
        expiry: "2026-12-31"
        legal_basis: "employment_contract"
        security_requirements:
          encryption: "AES-256"
          access_control: "RBAC"
          audit: "required"
        data_minimization: true
        retention: "30d after termination"
```

### Agreement Enforcement

```go
func checkSharingAgreement(consumer, dataField string) error {
    agreement := getActiveAgreement(consumer)
    if agreement == nil {
        return ErrNoSharingAgreement
    }
    if !contains(agreement.Data, dataField) {
        return ErrDataNotInAgreement
    }
    if time.Now().After(agreement.Expiry) {
        return ErrAgreementExpired
    }
    return nil
}
```

## Data Sovereignty Compliance

### Regulations by Region

| Regulation | Region | Key Requirement |
|---|---|---|
| GDPR | EU | EU personal data stays in EU |
| CCPA | California | Right to delete, right to know |
| PIPL | China | Data localization for China |
| LGPD | Brazil | Similar to GDPR |
| PDPA | Singapore | Consent for personal data |

### Data Residency Configuration

```yaml
data_governance:
  sovereignty:
    enabled: true
    regions:
      eu:
        data_types: ["pii", "financial"]
        storage: "eu-west-1"
        replication: "eu-west-1, eu-central-1"
      us:
        data_types: ["all"]
        storage: "us-east-1"
      cn:
        data_types: ["all"]
        storage: "cn-north-1"
        no_cross_border: true
```

## Data Governance Council

### Council Composition

| Member | Role | Responsibility |
|---|---|---|
| CISO | Chair | Security oversight |
| Data Owner | Business rep | Data access decisions |
| Data Steward | Technical rep | Quality and metadata |
| Legal Counsel | Compliance | Regulatory compliance |
| Privacy Officer | Privacy | GDPR/CCPA compliance |
| IT Director | Operations | Infrastructure |

### Council Cadence

| Meeting | Frequency | Topics |
|---|---|---|
| Monthly review | Monthly | Quality metrics, access requests |
| Quarterly assessment | Quarterly | Policy review, classification audit |
| Annual strategy | Annually | Framework update, regulation changes |

## GGID Data Governance Implementation

### Configuration

```yaml
data_governance:
  enabled: true
  classification:
    auto_classify: true
    tiers: [public, internal, confidential, restricted]
    review_cycle: 90d
  lineage:
    track: true
    storage: "postgresql"
  quality:
    metrics: [completeness, accuracy, consistency, timeliness, uniqueness, validity]
    reporting: weekly
  lifecycle:
    retention:
      restricted: 7y
      confidential: 3y
      internal: 1y
    auto_purge: true
  sharing:
    agreements_required: true
    dlp_scan: true
  sovereignty:
    enabled: true
    region_routing: true
  council:
    monthly_review: true
    quarterly_assessment: true
```

## Best Practices

1. **Assign clear ownership** — Every data element has an owner
2. **Classify automatically** — DLP scanner for consistent classification
3. **Track lineage** — Know where data comes from and goes
4. **Measure quality** — Track metrics and alert on degradation
5. **Enforce retention** — Don't keep data longer than needed
6. **Document sharing** — Every data share has an agreement
7. **Respect sovereignty** — Keep regulated data in its region
8. **Review regularly** — Governance council meets monthly
9. **Automate where possible** — Classification, quality, retention
10. **Train users** — Everyone understands their data responsibilities