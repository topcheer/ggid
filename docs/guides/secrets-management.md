# Secrets Management Guide

This guide covers managing secrets in GGID — rotation policies, storage options, zero-downtime rotation, audit trails, and emergency procedures.

## Secret Types

| Secret | Purpose | Rotation |
|--------|---------|----------|
| JWT signing key | Sign access/refresh tokens | 90 days |
| Password pepper | Append before password hash | 365 days |
| DB password | PostgreSQL connection | 90 days |
| Redis password | Redis AUTH | 90 days |
| NATS credentials | NATS authentication | 90 days |
| OAuth client secrets | Client authentication | 90 days |
| API keys | Service-to-service | 90 days |
| TLS private keys | HTTPS/gRPC | 90 days (Let's Encrypt: auto) |
| LDAP bind password | LDAP authentication | 180 days |
| SIEM API key | Forward audit events | 365 days |

## Storage Options

### Environment Variables (Development)

```bash
# .env file (NEVER commit)
DB_PASSWORD=str0ng_pass
JWT_SIGNING_KEY=path/to/key
PASSWORD_PEPPER=random_32_bytes
```

**Pros**: Simple, fast
**Cons**: Visible in process listing, no rotation, no audit

### Kubernetes Secrets

```bash
kubectl create secret generic ggid-secrets \
  --namespace ggid \
  --from-literal=db_password=xxx \
  --from-literal=password_pepper=xxx
```

**Pros**: Encrypted at rest (etcd encryption), native K8s
**Cons**: Cluster-admin can read, no auto-rotation

### HashiCorp Vault (Production)

```bash
vault kv put secret/ggid/db password=$(openssl rand -base64 32)
vault kv put secret/ggid/pepper pepper=$(openssl rand -base64 32)
```

```yaml
vault:
  address: https://vault.internal:8200
  auth: kubernetes  # Auto-auth via K8s service account
  secret_path: secret/data/ggid
```

**Pros**: Auto-rotation, dynamic secrets, full audit trail, leasing
**Cons**: Infrastructure overhead

### Cloud Secrets Manager

| Provider | Service | Auto-Rotation |
|----------|---------|---------------|
| AWS | Secrets Manager | Yes (Lambda) |
| Azure | Key Vault | Yes (Function) |
| GCP | Secret Manager | Manual |

## Zero-Downtime Rotation

### JWT Signing Key Rotation

```
1. Generate new key pair
2. Add new public key to JWKS (alongside old)
3. Switch signing to new key
4. Wait for old tokens to expire (TTL: 15min)
5. After 24h grace period: remove old key from JWKS
6. Destroy old private key
```

### DB Password Rotation

```
1. Create new DB user with new password
2. Grant same permissions
3. Deploy new password to GGID services
4. Verify new connections work
5. Revoke old user
6. Drop old user
```
### Password Pepper Rotation

See [Password Pepper Deploy](password-pepper-deploy.md) for detailed rotation procedure.

```
1. Set PASSWORD_PEPPER_NEW alongside PASSWORD_PEPPER_OLD
2. Verify: try NEW first, fall back to OLD
3. On login with OLD: re-hash with NEW
4. After all users migrated: remove OLD
```

## Audit Trail

Every secret access should be logged:

```
Timestamp | Secret Path | Operation | Identity | Source IP
2025-01-24T14:30Z | secret/ggid/db | read | ggid-auth-svc | 10.0.1.5
2025-01-24T14:31Z | secret/ggid/jwt | read | ggid-oauth-svc | 10.0.1.6
```

Vault provides native audit logging:
```bash
vault audit enable file file_path=/var/log/vault/audit.log
```

## Emergency Procedures

### Compromised Secret

```
1. IDENTIFY: Which secret? When was it compromised?
2. ROTATE: Generate new secret immediately
3. DEPLOY: Update all services with new secret
4. REVOKE: Invalidate old secret
5. AUDIT: Check for unauthorized access during exposure window
6. DOCUMENT: Post-incident report
```
### Lost Secret (Unrecoverable)

| Secret Lost | Impact | Recovery |
|------------|--------|----------|
| JWT key | Can't verify tokens | Generate new key, force re-login |
| Pepper | Can't verify passwords | Force password reset for all |
| DB password | Can't connect to DB | Reset via console, update services |
| TLS key | Can't serve TLS | Re-issue via ACME |

## Security Checklist

- [ ] No secrets in source code or Docker images
- [ ] `.gitignore` includes `.env*`, `*.key`, `*.pem`
- [ ] Secrets manager deployed (Vault/KMS/SM)
- [ ] Rotation policy documented and automated
- [ ] Audit logging on all secret access
- [ ] Emergency rotation tested quarterly
- [ ] Secret scanning on CI (git-secrets, truffleHog)
- [ ] Separate secrets per environment (dev/staging/prod)

## See Also

- [HSM Integration](hsm-integration.md)
- [Key Management Lifecycle](../research/key-management-lifecycle.md)
- [Password Pepper Deploy](password-pepper-deploy.md)
- [Production Checklist](production-checklist.md)
