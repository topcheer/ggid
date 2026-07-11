# Data Protection

> How GGID protects sensitive data: encryption at rest, encryption in transit, PII handling, data retention policies, GDPR compliance, and data residency.

---

## Table of Contents

1. [Encryption Overview](#encryption-overview)
2. [Encryption at Rest](#encryption-at-rest)
3. [Encryption in Transit](#encryption-in-transit)
4. [PII Handling](#pii-handling)
5. [Data Retention](#data-retention)
6. [Data Deletion](#data-deletion)
7. [GDPR Compliance](#gdpr-compliance)
8. [Data Residency](#data-residency)
9. [Backup Encryption](#backup-encryption)
10. [Key Management](#key-management)

---

## Encryption Overview

```
┌──────────────────────────────────────────────────┐
│ Layer 1: Transport (TLS 1.3)                     │
│ Client ←──TLS──→ Gateway ←──TLS/planned──→ Svc  │
├──────────────────────────────────────────────────┤
│ Layer 2: Application (JWT signing, HMAC)         │
│ Tokens signed with HMAC-SHA256 / RS256           │
├──────────────────────────────────────────────────┤
│ Layer 3: Database (PostgreSQL encryption)        │
│ Column-level encryption for PII, TDE for disk    │
├──────────────────────────────────────────────────┤
│ Layer 4: Infrastructure (disk encryption)        │
│ LUKS / cloud provider KMS                        │
└──────────────────────────────────────────────────┘
```

---

## Encryption at Rest

### Database Level

| Data Type | Protection | Implementation |
|-----------|-----------|----------------|
| Passwords | bcrypt hash (cost 12) | One-way hash, never decryptable |
| OAuth client secrets | AES-256-GCM | Symmetric encryption with server key |
| WebAuthn credentials | Stored as-is | Credential ID + public key (no secret) |
| JWT signing key | Environment variable | Stored in secrets manager (recommended) |
| LDAP bind password | Environment variable | Stored in secrets manager (recommended) |
| Session data | Redis (in-memory) | Redis AOF encryption (recommended) |
| Audit events | Plaintext in PostgreSQL | Sensitive fields encrypted at column level |

### Column-Level Encryption

Sensitive PII fields are encrypted before storage:

```sql
-- Email addresses encrypted with AES-256-GCM
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(255),
    email BYTEA,  -- AES-256-GCM encrypted
    phone BYTEA,  -- AES-256-GCM encrypted
    ...
);
```

### Disk Encryption

| Environment | Method |
|-------------|--------|
| Docker (local) | Host disk encryption (FileVault on macOS, LUKS on Linux) |
| Kubernetes | StorageClass with encryption (AWS EBS encryption, GCP PD encryption) |
| Bare metal | LUKS full-disk encryption |

---

## Encryption in Transit

### External Traffic

```
Client ───TLS 1.3───▶ Load Balancer ───HTTP───▶ Gateway
```

- **TLS 1.3** required (1.2 minimum for legacy)
- **HSTS**: `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- **Certificate management**: Let's Encrypt (automated) or commercial CA
- **Cipher suites**: Strong ciphers only, AEAD preferred

### Internal Traffic

```
Gateway ───HTTP (plaintext)───▶ Backend Services
Backend ───TCP (plaintext)───▶ PostgreSQL / Redis / NATS
```

**Current state**: Internal traffic is plaintext. Acceptable when:
- Services run on same host (Docker Compose)
- Network is isolated (Kubernetes pod network, VPC)

**Planned**: mTLS between all services via service mesh (Istio/Linkerd)

---

## PII Handling

### What is PII?

Personally Identifiable Information includes:

| Category | Examples |
|----------|----------|
| Direct identifiers | Email, phone, SSN, passport number |
| Indirect identifiers | Username, IP address, user agent |
| Sensitive | Biometric data, health info, religious beliefs |
| Authentication | Passwords, MFA secrets, security questions |

### PII Obfuscation

GGID includes a `pii.Obfuscate()` function for masking sensitive fields in logs and responses:

```go
pii.Obfuscate("john@example.com")  // → "j***@e******.com"
pii.Obfuscate("+1234567890")      // → "+1234***890"
pii.Obfuscate("192.168.1.50")     // → "192.168.x.x"
```

### Where PII Appears

| Location | PII Present | Protection |
|----------|-------------|------------|
| Database | Email, phone, name | Column encryption |
| JWT payload | User ID, tenant ID, email | Signed (not encrypted) — keep minimal |
| Audit logs | User ID, IP, user agent | IP obfuscation in exports |
| Application logs | Depends on log level | PII redaction middleware |
| Webhook payloads | User ID, email | HMAC signed, TLS delivered |
| Backups | All user data | Encrypted backup files |

### PII Redaction in Logs

The logging middleware redacts sensitive fields:

```go
// Before logging a request body
redacted := redactFields(body, []string{
    "password", "secret", "token", "refresh_token",
    "client_secret", "code", "mfa_code",
})
logger.Info("request", "body", redacted)
```

---

## Data Retention

### Default Retention Policy

| Data Type | Retention Period | Action After |
|-----------|-----------------|--------------|
| User accounts | Until deletion | Manual delete |
| Audit events | 90 days | Auto-delete |
| Session data | 8 hours | Redis TTL expiry |
| Refresh tokens | 7 days | Redis TTL expiry |
| JTI anti-replay | 15 minutes | Redis TTL expiry |
| Rate limit counters | 1 minute | Redis TTL expiry |
| OAuth states | 10 minutes | Redis TTL expiry |
| Password reset tokens | 1 hour | Redis TTL expiry |
| Webhook deliveries | 30 days | Auto-delete |
| System logs | 30 days | Log rotation |

### Configurable Retention

```bash
AUDIT_RETENTION_DAYS=90       # Audit event retention
WEBHOOK_DELIVERY_DAYS=30      # Webhook delivery log retention
LOG_RETENTION_DAYS=30         # Application log retention
```

---

## Data Deletion

### User Deletion

When a user is deleted:

1. User record marked as `deleted_at = NOW()` (soft delete)
2. After 30-day grace period, record is hard-deleted
3. All sessions revoked immediately
4. All refresh tokens invalidated
5. All MFA devices removed
6. WebAuthn credentials removed
7. Webhook `user.deleted` event emitted

### Right to Erasure (GDPR Article 17)

```bash
curl -X DELETE http://localhost:8080/api/v1/users/{user_id}?hard=true \
  -H "Authorization: Bearer <admin-JWT>"
```

With `hard=true`:
- User record deleted immediately (no grace period)
- Audit events anonymized (user_id replaced with "deleted_user")
- All related data removed

### Data Export (GDPR Article 20)

```bash
curl http://localhost:8080/api/v1/users/{user_id}/export \
  -H "Authorization: Bearer <admin-JWT>" \
  -o user_data_export.json
```

---

## GDPR Compliance

### Data Subject Rights

| Right | GGID Support |
|-------|-------------|
| Access (Art. 15) | `GET /api/v1/users/{id}/export` |
| Rectification (Art. 16) | `PUT /api/v1/users/{id}` |
| Erasure (Art. 17) | `DELETE /api/v1/users/{id}?hard=true` |
| Restriction (Art. 18) | `POST /api/v1/users/{id}/suspend` |
| Portability (Art. 20) | `GET /api/v1/users/{id}/export` (JSON format) |
| Object (Art. 21) | `DELETE /api/v1/users/{id}/consent` |

### Lawful Basis

GGID processes personal data under:
- **Contract** (Art. 6(1)(b)): User account management
- **Legal obligation** (Art. 6(1)(c)): Audit log retention
- **Legitimate interest** (Art. 6(1)(f)): Security monitoring

### Data Processing Records

Audit events serve as processing records:
- Who accessed what data
- When and from where
- For what purpose

### Data Protection Officer (DPO)

For enterprise deployments, export audit events to your SIEM for DPO review:

```bash
curl ".../api/v1/audit/events?from=2025-01-01&to=2025-07-11&format=csv" \
  -o audit_export.csv
```

---

## Data Residency

### Multi-Region Considerations

| Deployment | Data Location | Compliance |
|-----------|---------------|------------|
| Single-region | One data center | Simplest, data stays in one jurisdiction |
| Multi-region (active-passive) | Primary + replica | Replicas may be in different regions |
| Multi-region (active-active) | All regions | Data may be replicated globally |

### Region Pinning

For compliance with EU GDPR, China PIPL, etc.:

```bash
# Configure tenant to use EU-only infrastructure
TENANT_REGION_MAPPING='{"eu-tenant-001": "eu-central-1"}'
```

ABAC rule to enforce data residency:

```json
{
  "rule_name": "eu_data_only",
  "condition": "resource.region = 'EU' AND user.region = 'EU'",
  "action": "ALLOW"
}
```

---

## Backup Encryption

### Backup Files

Database backups are encrypted using AES-256-CBC:

```bash
# Backup script generates encrypted backup
pg_dump ggid | openssl enc -aes-256-cbc -salt -pass file:/etc/ggid/backup.key \
  > backup_$(date +%Y%m%d).sql.enc
```

### Key Rotation for Backups

- Backup encryption key rotated every 90 days
- Old key retained for 1 year (to decrypt old backups)
- Keys stored in HashiCorp Vault or cloud KMS

See: [Backup & Recovery](backup-recovery.md) for backup procedures.

---

## Key Management

### Current State

| Key Type | Storage | Rotation |
|----------|---------|----------|
| JWT signing secret | Environment variable | Manual |
| AES encryption key | Environment variable | Manual |
| Database password | Environment variable | Manual |
| Redis password | Environment variable | Manual |
| TLS private key | File / KMS | Per certificate renewal |
| Backup encryption key | File | Every 90 days |

### Recommended (Production)

| Key Type | Storage | Rotation |
|----------|---------|----------|
| JWT signing secret | HashiCorp Vault / AWS KMS | Every 90 days |
| AES encryption key | HashiCorp Vault / AWS KMS | Every 90 days |
| Database password | HashiCorp Vault / AWS Secrets Manager | Every 90 days |
| TLS certificates | cert-manager (K8s) / Let's Encrypt | Every 90 days |

### JWT Key Rotation (Planned)

1. Generate new signing key
2. Keep old key active for grace period (token lifetime)
3. New tokens signed with new key
4. Old tokens verified with old key during grace period
5. After grace period, remove old key

---

*Last updated: 2025-07-11*
