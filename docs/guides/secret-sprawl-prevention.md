# Secret Sprawl Prevention

Secret scanning, env var inventory, rotation enforcement, vault migration, CI/CD secret detection, runtime secret validation, and audit of all secret stores.

## Overview

Secret sprawl — secrets scattered across env vars, config files, code, and CI — is the most common cause of credential breaches. This guide prevents and detects it.

## Secret Inventory

### What Counts as a Secret

| Type | Examples | Storage |
|------|----------|---------|
| DB credentials | `DATABASE_URL`, `DB_PASSWORD` | Vault |
| API keys | `GGID_API_KEY`, `STRIPE_KEY` | Vault |
| OAuth secrets | `CLIENT_SECRET` | Vault |
| TLS private keys | `*.key`, `*.pem` | HSM/Vault |
| JWT signing keys | RSA/EC private keys | HSM |
| Encryption keys | AES keys, pepper | KMS |
| Cloud credentials | `AWS_SECRET_ACCESS_KEY` | IAM role |
| Webhook secrets | `WEBHOOK_SIGNING_SECRET` | Vault |

### Automated Inventory

```bash
# Scan all config sources for secrets
ggid secrets inventory
# Output:
# {
#   "kubernetes_secrets": 23,
#   "env_vars": 47,
#   "vault_paths": 31,
#   "config_files": 12,
#   "total_secrets": 113,
#   "duplicates": 3,
#   "stale": 8,
#   "unrotated_90d": 5
# }
```

## Secret Scanning

### Pre-Commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit
if gitleaks protect --staged; then
    exit 0
else
    echo "SECRET DETECTED in staged files!"
    echo "Remove the secret and use vault/Vault instead."
    exit 1
fi
```

### CI/CD Pipeline Scanning

```yaml
# GitHub Actions
secret_scan:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0  # Full history

    - name: gitleaks
      uses: gitleaks/gitleaks-action@v2
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

    - name: trivy-fs-scan
      run: trivy fs --scanners secret .
```

### Patterns Detected

| Pattern | Regex Example |
|---------|--------------|
| AWS key | `AKIA[0-9A-Z]{16}` |
| Private key | `-----BEGIN (RSA\|EC\|OPENSSH) PRIVATE KEY-----` |
| JWT | `eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*` |
| Generic password | `password\s*[=:]\s*["'][^"']+["']` |
| Connection string | `(postgres\|mongodb\|redis)://[^:]+:[^@]+@` |

## Vault Migration

### Current State → Target State

| Current | Problem | Target |
|---------|---------|--------|
| `.env` files | Committed to git | Vault + env injection at runtime |
| Kubernetes Secrets (base64) | Not encrypted at rest | Vault CSI driver |
| Hardcoded in config | Visible in code | Vault reference (`vault://path/key`) |
| CI/CD secrets (GitHub) | Visible in logs | Vault OIDC auth |

### Migration Steps

```bash
# Step 1: Import secrets to Vault
vault kv put secret/ggid/db host="..." password="..." port="5432"

# Step 2: Update app to read from Vault
# Before:
# dbPass := os.Getenv("DB_PASSWORD")

# After:
secret, _ := vault.Read("secret/ggid/db")
dbPass := secret["password"]

# Step 3: Remove from env vars and config files
# Step 4: Verify app reads from Vault
# Step 5: Delete old secrets from CI/CD and env
```

## Rotation Enforcement

### Rotation Policy

| Secret Type | Max Age | Rotation Trigger |
|-------------|---------|-----------------|
| DB credentials | 90 days | Scheduled |
| API keys | 60 days | Scheduled |
| OAuth client secrets | 90 days | Scheduled |
| JWT signing keys | 30 days | Scheduled |
| TLS certificates | 90 days | Auto (ACM/LE) |
| Encryption keys | 365 days | Scheduled |
| Webhook secrets | 90 days | Manual |

### Automated Rotation Check

```go
func checkRotationCompliance() []RotationViolation {
    violations := []RotationViolation{}

    for _, secret := range getAllSecrets() {
        age := time.Since(secret.RotatedAt)
        maxAge := rotationPolicy[secret.Type]

        if age > maxAge {
            violations = append(violations, RotationViolation{
                Secret: secret.Path,
                Age: age,
                MaxAge: maxAge,
                Severity: age > maxAge*2 ? "critical" : "warning",
            })
        }
    }
    return violations
}
```

### Alert on Non-Compliance

```bash
GET /api/v1/admin/secrets/rotation-status
# → {
#   "compliant": 108,
#   "non_compliant": 5,
#   "violations": [
#     {"path": "secret/ggid/db/password", "age_days": 120, "max_age_days": 90},
#     {"path": "secret/ggid/oauth/client-123-secret", "age_days": 100, "max_age_days": 90}
#   ]
# }
```

## Runtime Secret Validation

### Verify Secrets at Startup

```go
func validateSecrets() error {
    required := []string{
        "JWT_SIGNING_KEY",
        "DATABASE_URL",
        "REDIS_URL",
        "NATS_URL",
    }

    for _, key := range required {
        val := os.Getenv(key)
        if val == "" {
            return fmt.Errorf("missing required secret: %s", key)
        }
        // Verify it's not a placeholder
        if val == "changeme" || val == "TODO" || val == "secret" {
            return fmt.Errorf("placeholder secret detected: %s", key)
        }
    }

    // Verify DB connectivity with the secret
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        return fmt.Errorf("DB connection failed with configured secret")
    }
    db.Close()

    return nil
}
```

### Secret Freshness Check

```go
// Every 5 minutes, verify secrets haven't been rotated externally
func checkSecretFreshness() {
    currentHash := hashSecret(os.Getenv("DB_PASSWORD"))
    if storedHash != currentHash {
        // Secret was rotated in Vault but app still has old one
        log.Warn("secret drift detected — reloading from Vault")
        reloadSecrets()
    }
}
```

## CI/CD Secret Detection

```yaml
# Block PRs with secrets
name: secret-detection
on: [pull_request]
jobs:
  detect:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Scan diff
        run: |
          if gitleaks detect --source . --report-path leaks.json; then
            echo "Clean"
          else
            echo "::error::Secrets detected in PR!"
            cat leaks.json
            exit 1
          fi
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Secrets not in Vault | Any → migrate |
| Secrets older than max age | Any → rotate |
| Secrets in git history | Any → purge + rotate |
| Duplicate secrets | Any → consolidate |
| Missing runtime validation | Any → add check |

## See Also

- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [Database Security](database-security.md)
- [Compliance Automation](compliance-automation.md)
- [Post-Quantum Crypto Migration](post-quantum-crypto-migration.md)
