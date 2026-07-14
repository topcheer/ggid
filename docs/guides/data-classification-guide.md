# Data Classification Guide

This guide covers GGID's 4-tier data classification model, classification criteria, automation, labeling, access control, encryption, and retention policies.

## 4-Tier Classification Model

```
┌──────────────────────────────────────────────────┐
│  RESTRICTED  │ Highest sensitivity, severe impact │
├──────────────────────────────────────────────────┤
│  CONFIDENTIAL │ High sensitivity, significant impact│
├──────────────────────────────────────────────────┤
│  INTERNAL    │ Moderate sensitivity, limited impact │
├──────────────────────────────────────────────────┤
│  PUBLIC      │ No sensitivity, no impact            │
└──────────────────────────────────────────────────┘
```

### Tier Definitions

| Tier | Description | Examples | Breach Impact |
|---|---|---|---|
| Restricted | Most sensitive data, legally regulated | SSN, medical records, encryption keys | Severe: legal, financial, reputational |
| Confidential | Sensitive business data | PII (email, phone), financial data, API keys | Significant: regulatory, competitive |
| Internal | Internal business data | Org charts, internal docs, project plans | Moderate: operational, minor reputational |
| Public | Approved for public release | Marketing materials, public APIs, open docs | None |

## Classification Criteria

### Data Type Criteria

| Data Type | Default Tier | Rationale |
|---|---|---|
| Passwords / secrets | Restricted | Authentication credentials |
| Encryption keys | Restricted | Key material |
| SSN / National ID | Restricted | Legally regulated PII |
| Medical records (PHI) | Restricted | HIPAA regulated |
| Credit card numbers | Restricted | PCI-DSS regulated |
| Email addresses | Confidential | PII under GDPR |
| Phone numbers | Confidential | PII under GDPR |
| Home addresses | Confidential | PII under GDPR |
| Date of birth | Confidential | PII |
| Financial records | Confidential | Business sensitive |
| API keys / tokens | Confidential | Credential material |
| Audit logs | Confidential | Contains user activity |
| Internal documentation | Internal | Not for external release |
| Organization structure | Internal | Business operational |
| Public API responses | Public | Intended for public |
| Marketing content | Public | Approved for release |

### Criteria Questions

1. **Is it legally regulated?** (GDPR, HIPAA, PCI-DSS, SOX) → Restricted or Confidential
2. **Does it contain PII?** → Confidential minimum
3. **Could it cause financial loss if leaked?** → Confidential minimum
4. **Is it meant for public consumption?** → Public
5. **Is it internal operational data?** → Internal

## PII vs Sensitive vs Regulated Data

### PII (Personally Identifiable Information)

Data that can identify an individual:

| Field | PII? | Tier |
|---|---|---|
| Full name | Yes | Confidential |
| Email | Yes | Confidential |
| Phone | Yes | Confidential |
| Address | Yes | Confidential |
| IP address | Yes (under GDPR) | Confidential |
| User ID | Yes | Internal (if pseudonymized) |
| SSN | Yes | Restricted |
| Biometric data | Yes | Restricted |

### Sensitive Data (Non-PII)

Business data that's sensitive but not personal:

| Data | Tier |
|---|---|
| Source code (proprietary) | Confidential |
| Business plans | Confidential |
| Financial projections | Confidential |
| Security configurations | Restricted |
| Private keys | Restricted |

### Regulated Data

Data subject to legal/regulatory requirements:

| Regulation | Data Types | Tier |
|---|---|---|
| GDPR | PII of EU residents | Confidential+ |
| HIPAA | Protected Health Information | Restricted |
| PCI-DSS | Cardholder data | Restricted |
| SOX | Financial controls data | Confidential |
| CCPA | California consumer PII | Confidential+ |

## Classification Automation

### DLP Scanner

```go
type ClassificationScanner struct {
    patterns map[string]*regexp.Regexp
}

func NewScanner() *ClassificationScanner {
    return &ClassificationScanner{
        patterns: map[string]*regexp.Regexp{
            "ssn":          regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
            "credit_card":  regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`),
            "email":        regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
            "phone":        regexp.MustCompile(`\b\+?\d{1,3}[-.\s]?\(?\d{1,4}\)?[-.\s]?\d{3,4}[-.\s]?\d{4}\b`),
            "api_key":      regexp.MustCompile(`\b(ggid_[a-zA-Z0-9]{32,})\b`),
            "private_key":  regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`),
        },
    }
}

func (s *ClassificationScanner) Classify(data string) string {
    // Check for restricted patterns first
    if s.patterns["ssn"].MatchString(data) || s.patterns["private_key"].MatchString(data) {
        return "restricted"
    }
    if s.patterns["credit_card"].MatchString(data) || s.patterns["api_key"].MatchString(data) {
        return "confidential"
    }
    if s.patterns["email"].MatchString(data) || s.patterns["phone"].MatchString(data) {
        return "confidential"
    }
    return "internal"
}
```

### Pattern Matching

```yaml
classification:
  auto_scan: true
  patterns:
    restricted:
      - pattern: "\\b\\d{3}-\\d{2}-\\d{4}\\b"
        name: "SSN"
      - pattern: "-----BEGIN.*PRIVATE KEY-----"
        name: "Private Key"
      - pattern: "\\b\\d{16}\\b"
        name: "Credit Card"
    confidential:
      - pattern: "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}"
        name: "Email"
      - pattern: "ggid_[a-zA-Z0-9]{32,}"
        name: "API Key"
```

## Labeling Standards

### Metadata Tags

Data objects include classification metadata:

```json
{
  "data": { ... },
  "classification": {
    "tier": "confidential",
    "contains_pii": true,
    "pii_types": ["email", "phone"],
    "regulations": ["GDPR"],
    "classified_at": "2026-07-12T10:00:00Z",
    "classified_by": "auto-scanner"
  }
}
```

### HTTP Headers

API responses include classification headers:

```
X-Data-Classification: confidential
X-Data-Contains-PII: true
X-Data-Regulations: GDPR
```

### Database Labels

```sql
-- Column-level classification
COMMENT ON COLUMN users.email IS 'CLASSIFICATION: confidential, PII: true, REGULATION: GDPR';
COMMENT ON COLUMN users.ssn IS 'CLASSIFICATION: restricted, PII: true, REGULATION: HIPAA';
```

## Access Control by Classification

### Role-Based Access

| Tier | Who Can Access | Required Conditions |
|---|---|---|
| Restricted | Specific roles (security-admin) | MFA + step-up + audit |
| Confidential | Authorized roles | MFA + audit |
| Internal | All authenticated users | Standard auth |
| Public | Anyone | No auth required |

### Implementation

```go
func RequireClassificationAccess(tier string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromContext(r)

        switch tier {
        case "restricted":
            if !user.HasRole("security-admin") {
                writeError(w, 403, "insufficient_classification_access")
                return
            }
            if !user.MFAVerified || !user.StepUpVerified {
                writeError(w, 403, "step_up_required_for_restricted")
                return
            }
        case "confidential":
            if !user.HasAnyRole("admin", "security-admin", "user-admin") {
                writeError(w, 403, "insufficient_classification_access")
                return
            }
            if !user.MFAVerified {
                writeError(w, 403, "mfa_required_for_confidential")
                return
            }
        case "internal":
            if user == nil {
                writeError(w, 401, "authentication_required")
                return
            }
        case "public":
            // No restrictions
        }

        // Audit access to restricted/confidential
        if tier == "restricted" || tier == "confidential" {
            audit.Log(AuditEvent{
                Type:         "data_access",
                UserID:       user.ID,
                Classification: tier,
                Resource:     r.URL.Path,
                IP:           clientIP(r),
            })
        }

        next.ServeHTTP(w, r)
    })
}
```

## Encryption Requirements Per Tier

| Tier | At Rest | In Transit | Key Management |
|---|---|---|---|
| Restricted | AES-256-GCM + tenant-specific key | TLS 1.3 | HSM-backed, per-tenant keys |
| Confidential | AES-256-GCM | TLS 1.2+ | KMS-managed, per-tenant keys |
| Internal | AES-128-GCM | TLS 1.2 | Shared key |
| Public | Optional | TLS 1.2 | Shared key |

### Per-Tenant Encryption Keys

```go
func encryptRestrictedData(data []byte, tenantID string) ([]byte, error) {
    key := getTenantKey(tenantID)  // From HSM/KMS
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    return gcm.Seal(nonce, nonce, data, nil), nil
}
```

## Retention Policy Per Tier

| Tier | Default Retention | Legal Hold | Disposal Method |
|---|---|---|---|
| Restricted | 7 years (regulated) | Supported | Cryptographic erase |
| Confidential | 3 years | Supported | Secure overwrite + verify |
| Internal | 1 year | Supported | Standard deletion |
| Public | Indefinite | N/A | Standard deletion |

### Retention Configuration

```yaml
retention:
  tiers:
    restricted:
      default: 7y
      regulations:
        HIPAA: 7y
        SOX: 7y
      disposal: "crypto_erase"
      purge_after: true
    confidential:
      default: 3y
      regulations:
        GDPR: "user_request"  # Delete on user request
      disposal: "secure_overwrite"
      purge_after: true
    internal:
      default: 1y
      disposal: "standard_delete"
      purge_after: false
    public:
      default: indefinite
      disposal: "standard_delete"
      purge_after: false
```

### Automated Retention Enforcement

```go
func enforceRetention() {
    // Runs daily via cron
    now := time.Now()

    // Restricted data
    purgeOlderThan("restricted", now.AddDate(-7, 0, 0), cryptoErase)

    // Confidential data
    purgeOlderThan("confidential", now.AddDate(-3, 0, 0), secureOverwrite)

    // Internal data
    purgeOlderThan("internal", now.AddDate(-1, 0, 0), standardDelete)
}
```

## PII Obfuscation

GGID's PII package automatically obfuscates sensitive fields in logs and audit trails:

```go
import "github.com/ggid/ggid/pkg/pii"

func logUserData(user *User) {
    safe := pii.Obfuscate(map[string]interface{}{
        "email":    user.Email,     // j***@example.com
        "phone":    user.Phone,     // +1-*-555-****
        "name":     user.Name,      // J** D**
        "ssn":      user.SSN,       // ***-**-****
    })
    log.Info("user data", "data", safe)
}
```

## Best Practices

1. **Classify at creation** — Assign tier when data is first stored
2. **Automate classification** — Use DLP scanner for automatic tier assignment
3. **Label everything** — Metadata tags on all data objects
4. **Enforce access control** — Tier-based authorization on all endpoints
5. **Encrypt appropriately** — Per-tier encryption requirements
6. **Audit access** — Log all access to restricted and confidential data
7. **Enforce retention** — Automated purge of expired data
8. **Review periodically** — Reassess classification as regulations change
9. **Train users** — Ensure all users understand classification tiers
10. **Monitor for violations** — Alert on unauthorized access attempts
