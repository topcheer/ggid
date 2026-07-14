# Secrets Rotation Automation

Guide for automated, zero-downtime rotation of JWT keys, DB credentials, API keys, and OAuth client secrets.

## Rotation Pipeline

```
┌────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Trigger     │───▶│ Generate New │───▶│ Dual Publish │───▶│ Verify New   │
│ (schedule/  │    │ Secret       │    │ (old + new)  │    │ (health      │
│  incident)  │    │              │    │              │    │  checks)     │
└────────────┘    └──────────────┘    └──────────────┘    └──────┬───────┘
                                                                  ▼
                    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
                    │ Rollback if  │◀───│ Retire Old   │◀───│ Cutover      │
                    │ health fails │    │ (grace end)  │    │ Complete     │
                    └──────────────┘    └──────────────┘    └──────────────┘
```

## Secret Types and Schedule

| Secret | Rotation Period | Grace Period | Method |
|--------|----------------|-------------|--------|
| JWT signing keys | 90 days | 24h overlap | JWKS multi-key |
| DB credentials | 90 days | 1h overlap | PGBouncer reload |
| API keys | 365 days | 7 day overlap | Dual key valid |
| OAuth client secrets | 90 days | 7 day overlap | Dual secret |
| mTLS certificates | 365 days | 24h overlap | Cert manager |
| Webhook signing secrets | 90 days | 24h overlap | Dual signature |
| Encryption keys | 365 days | Indefinite | Re-encrypt on read |

## JWT Signing Key Rotation

### Zero-Downtime Procedure

```bash
# Step 1: Generate new key pair
ggid keys generate --type rsa --kid new-key-2025-02
# → private key to KMS, public key to JWKS

# Step 2: Publish both keys in JWKS (old + new)
GET /.well-known/jwks.json
# → {keys: [{kid: "old-key", ...}, {kid: "new-key", ...}]}

# Step 3: Start signing with new key
# Verifiers accept both keys during grace period

# Step 4: After 24h, remove old key from JWKS
# Tokens signed with old key expire naturally (15-min TTL)
```

```go
type KeyRotator struct {
    current    *SigningKey
    previous   *SigningKey // Valid during grace period
    graceUntil time.Time
}

func (r *KeyRotator) Sign(claims jwt.Claims) (string, error) {
    return jwt.SignWithKey(r.current.PrivateKey, claims) // Always sign with current
}

func (r *KeyRotator) Verify(token string) (jwt.Claims, error) {
    // Accept both keys during grace period
    if claims, err := verifyWithKey(r.current.PublicKey, token); err == nil {
        return claims, nil
    }
    if time.Now().Before(r.graceUntil) {
        return verifyWithKey(r.previous.PublicKey, token) // Fallback to old
    }
    return nil, ErrInvalidToken
}
```

### Health Check

```bash
# After rotation, verify new key works
TOKEN=$(ggid token issue --kid new-key-2025-02)
ggid token verify "$TOKEN" --expect-kid new-key-2025-02
# Exit 0 = healthy
```

## DB Credential Rotation

```bash
# Step 1: Create new DB role
psql -c "CREATE ROLE ggid_app_new WITH PASSWORD '$NEW_PASS' LOGIN;"

# Step 2: Grant same permissions
psql -c "GRANT ggid_app_permissions TO ggid_app_new;"

# Step 3: Update Kubernetes secret
kubectl create secret generic ggid-db-secret \
  --from-literal=password="$NEW_PASS" \
  --dry-run=client -o yaml | kubectl apply -f -

# Step 4: Rolling restart (pods pick up new secret)
kubectl rollout restart deployment/ggid-auth

# Step 5: Verify health
curl https://auth.ggid.dev/healthz

# Step 6: Drop old role after all pods healthy
psql -c "DROP ROLE ggid_app_old;"
```

## API Key Rotation

```bash
# Generate new key, keep old valid for 7 days
POST /api/v1/admin/api-keys/rotate
{"key_id": "ak-abc123", "grace_days": 7}
# → {"new_key": "ak-xyz789", "old_key_valid_until": "2025-02-22"}

# Old key logged as "rotated, grace period active"
# New key immediately valid alongside old
```

## OAuth Client Secret Rotation

```bash
# Rotate client secret
POST /api/v1/oauth/clients/{client_id}/rotate-secret
{}
# → {"new_secret": "...", "old_secret_valid_until": "2025-02-22"}

# Client can use either secret during grace period
# After grace, old secret rejected with 401
```

## Rollback

If health checks fail after rotation:

```bash
# Immediate rollback — restore previous key/secret
ggid keys rollback --kid new-key-2025-02
# → Removes new key from JWKS
# → Restores previous key as current
# → Alert sent to security team
```

Rollback triggers:
- Health check failure (>5% error rate after cutover)
- Token verification failures spike
- DB connection failures
- User reports of auth failures

## Automation Pipeline

```yaml
# Cron triggers rotation
rotation_schedule:
  jwt_keys:
    cron: "0 3 1 */3 *"  # Quarterly at 3 AM
    health_check: true
    rollback_on_failure: true
    notify: [security@corp.com]

  db_credentials:
    cron: "0 4 1 */3 *"
    rolling_restart: true
    health_check: true

  api_keys:
    cron: "0 5 1 1 *"  # Yearly
    grace_days: 7

  oauth_secrets:
    cron: "0 3 15 */3 *"
    grace_days: 7
    notify_clients: true
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Rotation failure | Immediate → page ops |
| Old key usage after grace | Any → security alert |
| Health check post-rotation | >1% errors → rollback |
| Rotation overdue | >7 days past schedule |
| Rollback triggered | Any → root cause analysis |

## See Also

- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Database Security](database-security.md)
- [Secrets Management](secrets-management.md)
- [Key Rotation Procedure](key-rotation-procedure.md)
- [Password Pepper Deployment](password-pepper-deploy.md)
