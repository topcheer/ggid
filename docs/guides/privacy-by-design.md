# Privacy by Design

Guide for embedding privacy into GGID's architecture: data minimization, purpose limitation, consent, and data subject rights.

## Seven Principles (Privacy by Design)

| Principle | GGID Implementation |
|-----------|-------------------|
| Proactive not reactive | Privacy impact assessments before features ship |
| Privacy as default | Minimal data collected, maximal protection by default |
| Privacy embedded into design | PII module, consent engine, encryption in core architecture |
| Full functionality | Security and privacy enhance — not degrade — user experience |
| End-to-end security | Encryption in transit + at rest, key rotation, audit chain |
| Visibility and transparency | Data subject rights API, audit logs, privacy policy |
| Respect for user privacy | Consent management, opt-out, data export/erasure |

## Data Minimization

GGID collects only what is necessary for each function:

```sql
-- Registration: only required fields
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL,          -- Required: login + communication
    display_name TEXT NOT NULL,   -- Required: identification
    phone TEXT,                   -- Optional: MFA (encrypted)
    -- NO: SSN, date of birth, biometric data, salary
);
```

| Data Point | Required? | Purpose | Retention |
|-----------|----------|---------|-----------|
| Email | Yes | Login, notifications | Account lifetime |
| Display name | Yes | UI display | Account lifetime |
| Phone | Optional | MFA factor | Until user removes |
| IP address | Auto-collected | Security, audit | 90 days |
| User agent | Auto-collected | Device fingerprinting | 30 days |
| Audit logs | Auto | Compliance, security | 7 years (anonymized) |

### Scope-Based Data Release

```yaml
scope_data_mapping:
  "openid": [sub]
  "profile": [display_name, locale]
  "email": [email, email_verified]
  "users:read": [display_name, email, department]
  # Never release: password_hash, mfa_secret, security_answers
```

## Purpose Limitation

Each data field is tagged with allowed purposes:

```go
type DataField struct {
    Name          string
    AllowedPurposes []string
}

var FieldPurposes = map[string]DataField{
    "email":    {AllowedPurposes: ["auth", "communication", "account_recovery"]},
    "phone":    {AllowedPurposes: ["mfa", "security_alerts"]},
    "audit":    {AllowedPurposes: ["compliance", "security_monitoring"]},
}
```

Data accessed for a purpose not in `AllowedPurposes` is a policy violation.

## Storage Limitation

```yaml
retention_policy:
  active_user_data: "account_lifetime"
  dormant_user_data: "365_days"  # Then anonymize
  audit_logs: "7_years"          # Then archive
  ip_addresses: "90_days"        # Then hash/truncate
  session_data: "session_ttl"    # Then destroy
  backup_snapshots: "30_days"    # Then purge
```

Automated deletion runs nightly:

```sql
-- Anonymize dormant users (>365 days inactive)
UPDATE users SET 
  email = 'anonymized_' || md5(email::text),
  display_name = 'Deleted User',
  phone = NULL,
  status = 'archived'
WHERE status = 'dormant'
  AND last_login_at < NOW() - INTERVAL '365 days';
```

## PII Inventory

GGID maintains a complete PII inventory:

| Category | Fields | Storage | Encryption |
|----------|--------|---------|------------|
| Identifiers | email, username, sub | PostgreSQL | Column-level (pgcrypto) |
| Contact | phone, address | PostgreSQL | Column-level |
| Biometric | WebAuthn public key | PostgreSQL | Encrypted at rest |
| Behavioral | login patterns, device fingerprints | Redis | In-memory (ephemeral) |
| Audit | actor, action, timestamp | PostgreSQL | Hash chain |

### PII Classification

```go
type PIIClassification int

const (
    Public PIIClassification = iota   // display_name
    Internal                          // department, role
    Confidential                      // email, phone
    Restricted                        // password_hash, mfa_secret, recovery_codes
)
```

## Consent Management

### Consent Record

```bash
# User grants consent for specific purposes
POST /api/v1/consent
{
  "user_id": "uuid",
  "purposes": ["marketing", "analytics"],
  "scopes": ["email", "profile"],
  "client_id": "oauth-client-123"
}
# → Returns consent_id with timestamp
```

### Consent Lifecycle

```
Requested → Granted → (withdrawn at any time) → Revoked
```

```bash
# List all active consents
GET /api/v1/consent

# Withdraw consent
DELETE /api/v1/consent/{consent_id}
# → Immediate: stop data sharing, trigger data purge workflow
```

### Global Privacy Control (GPC)

```go
// Respect browser GPC signal
func handleGPC(r *http.Request, user User) {
    if r.Header.Get("Sec-GPC") == "1" {
        // User opted out of sale/sharing globally
        optOutSale(user.ID)
        restrictDataSharing(user.ID)
    }
}
```

## Data Subject Rights

| Right (GDPR) | API Endpoint | SLA |
|---------------|-------------|-----|
| Access (Art. 15) | `GET /api/v1/identity/users/{id}?expand=all` | 30 days |
| Rectification (Art. 16) | `PATCH /api/v1/identity/users/{id}` | 30 days |
| Erasure (Art. 17) | `DELETE /api/v1/identity/users/{id}` | 30 days |
| Restriction (Art. 18) | `PATCH /api/v1/identity/users/{id} {"status":"restricted"}` | 30 days |
| Portability (Art. 20) | `GET /api/v1/identity/users/{id}/export` | 30 days |
| Object (Art. 21) | `POST /api/v1/consent/object` | 30 days |

### Data Export Format

```json
{
  "user": {
    "id": "uuid",
    "email": "user@corp.com",
    "display_name": "Jane Doe",
    "phone": "+1-555-0100"
  },
  "sessions": [...],
  "consents": [...],
  "audit_events": [...],
  "mfa_factors": [...],
  "oauth_grants": [...]
}
```

Export is JSON (machine-readable) for portability.

## Privacy Impact Assessment (PIA)

Required before shipping any feature that:
- Collects new personal data
- Changes data flow between services
- Enables new third-party data sharing
- Modifies retention periods

```markdown
## PIA Template

1. Feature description
2. Data collected (new/existing)
3. Purpose and legal basis
4. Data flow diagram
5. Third parties involved
6. Retention period
7. Risk assessment (likelihood × impact)
8. Mitigations
9. Data subject rights impact
10. Approval: Privacy Officer
```

## Privacy Architecture

```
User → Gateway → [PII classification + consent check] → Service
                         │
                    ┌────┴────┐
                    │ pii.Obfuscate() │ ← Mask PII before logging/audit
                    └─────────┘
```

PII is obfuscated before entering logs:
- Email: `j***@corp.com`
- Phone: `+1-555-01**`
- IP: `10.0.1.*`

## See Also

- [Compliance Framework Mapping](compliance-framework-mapping.md)
- [GDPR Compliance](gdpr-compliance.md)
- [Data Retention Policy](data-retention-policy.md)
- [Database Security](database-security.md)
- [OAuth Scope Design](oauth-scope-design.md)
