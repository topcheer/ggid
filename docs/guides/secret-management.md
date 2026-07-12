# Secret Management

Secret types, vault patterns, rotation automation, dynamic secrets, access policies, audit trail, leak detection, and GGID integration.

## Secret Types

| Type | Example | Rotation |
|------|---------|----------|
| API keys | `ak-uuid:secret` | 90 days |
| Database passwords | `pg://user:pass@host` | 90 days |
| TLS private keys | `-----BEGIN RSA...` | 90 days |
| OAuth client secrets | `client_secret:...` | 180 days |
| JWT signing keys | RSA/ECDSA private key | 90 days |
| Connection strings | `redis://:pass@host` | 90 days |
| Encryption keys | AES-256, DEK/TEK/MEK | Per policy |

## Vault Patterns

### HashiCorp Vault

```bash
# Store secret
vault kv put secret/ggid/prod/db password="..."

# Read secret (app uses this at runtime)
vault kv get secret/ggid/prod/db
# → {"password": "..."}

# Dynamic secret (database credentials generated on-demand)
vault write database/roles/ggid-app \
  db_name=postgres \
  creation_statements="CREATE ROLE \"{{name}}\"..."
# → Vault creates temporary DB user, TTL 1h
```

### AWS Secrets Manager

```bash
# Store
aws secretsmanager create-secret --name ggid/prod/db --secret-string '{"password":"..."}'

# Retrieve (app code)
aws secretsmanager get-secret-value --secret-id ggid/prod/db
```

### Azure Key Vault

```bash
az keyvault secret set --vault-name ggid-kv --name db-password --value "..."
az keyvault secret show --vault-name ggid-kv --name db-password
```

## Rotation Automation

```yaml
rotation:
  schedule: "0 3 1 * *"
  secrets:
    - path: "secret/ggid/prod/db"
      rotate_every: 90d
      generator: "random_32_char"
      consumers: ["identity-svc", "auth-svc"]
      notify_before: 7d
      
    - path: "secret/ggid/prod/jwt-signing"
      rotate_every: 90d
      generator: "ecdsa_p256"
      consumers: ["auth-svc", "oauth-svc"]
      grace_period: 15d
```

### Zero-Downtime Rotation

```
1. Generate new secret
2. Store in vault (new version)
3. Signal consumers to re-read (SIGHUP or webhook)
4. Both old + new valid during grace period
5. After grace period → revoke old
```

## Dynamic Secrets

| Secret | TTL | How |
|--------|-----|-----|
| DB credentials | 1 hour | Vault creates temp DB user |
| AWS STS | 1 hour | Vault generates STS token |
| PKI cert | 24 hours | Vault issues short-lived cert |

Benefits: no long-lived secrets, automatic expiry, per-request credentials.

## Access Policies

```yaml
secret_access:
  - identity-svc:
      allowed: ["secret/ggid/prod/db", "secret/ggid/prod/jwt-signing"]
      denied: ["secret/ggid/prod/payment-*"]
      
  - audit-svc:
      allowed: ["secret/ggid/prod/audit-db"]
      denied: ["*"]
```

## Leak Detection

| Method | How |
|--------|-----|
| Pre-commit scan | gitleaks scans git diff |
| CI scan | TruffleHog scans repo |
| Runtime scan | Scan env vars for known secret patterns |
| Vault audit | Compare vault access logs vs actual usage |
| Public exposure | GitHub secret scanning alerts |

## GGID Integration

```go
// External Secrets Operator syncs Vault → K8s Secret → pod env
type SecretProvider interface {
    Get(path string) (string, error)
    Rotate(path string) error
}

// Default: Vault
type VaultProvider struct { client *vault.Client }
// Fallback: AWS Secrets Manager
type AWSProvider struct { client *secretsmanager.Client }
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Secret access failures | >1% → check policy or vault |
| Secrets nearing rotation | <7 days → verify auto |
| Leak detected | Any → immediate rotation |
| Dynamic secret TTL expiry | Track rate |
| Unauthorized access attempts | Any → security alert |

## See Also

- [Secret Sprawl Prevention](secret-sprawl-prevention.md)
- [Credential Vault Architecture](credential-vault-architecture.md)
- [KMS Integration](kms-integration.md)
- [Cryptography Key Rotation](cryptography-key-rotation.md)