# Password Pepper Deployment Guide

This guide covers deploying password pepper in GGID — HSM integration, key rotation, backward compatibility, and migration.

## Overview

A password pepper is a server-side secret appended to user passwords before hashing. Unlike per-user salts (stored in the database), the pepper is never stored alongside the hash, making offline brute-force impossible even if the database is compromised.

```
Hash stored in DB:  argon2id(password + salt + pepper)
Attacker steals DB: Has salt + hash, but NOT pepper → cannot brute-force
```

## Configuration

### Environment Variable

```bash
# Generate a 32-byte random pepper
PEPPER=$(openssl rand -base64 32)
echo "PASSWORD_PEPPER=$PEPPER" >> keys.env
```

```yaml
# docker-compose.yaml
services:
  auth:
    environment:
      - PASSWORD_PEPPER=${PASSWORD_PEPPER}
```

### Kubernetes Secret

```bash
kubectl create secret generic ggid-pepper \
  --namespace ggid \
  --from-literal=password_pepper=$(openssl rand -base64 32)
```

```yaml
# deployment.yaml
env:
  - name: PASSWORD_PEPPER
    valueFrom:
      secretKeyRef:
        name: ggid-pepper
        key: password_pepper
```

## How GGID Uses Pepper

```go
// auth/internal/service/password_service.go
func HashPassword(password string) (string, error) {
    salt := generateSalt(16)
    peppered := password + salt + pepper  // Pepper from env
    hash := argon2id.Hash(peppered, ...)
    return encodeHash(hash, salt), nil
}

func VerifyPassword(password, storedHash string) (bool, error) {
    salt := extractSalt(storedHash)
    peppered := password + salt + pepper
    return argon2id.Verify(peppered, storedHash)
}
```

## HSM Integration

For maximum security, store the pepper in an HSM (Hardware Security Module):

### AWS KMS

```go
// Fetch pepper from KMS at startup
result, _ := kmsClient.Decrypt(ctx, &kms.DecryptInput{
    CiphertextBlob: encryptedPepper,
})
pepper := string(result.Plaintext)
```

### HashiCorp Vault

```bash
# Store pepper in Vault
vault kv put secret/ggid/password-pepper pepper=@/dev/stdin <<< "$PEPPER"

# Fetch at startup
PEPPER=$(vault kv get -field=pepper secret/ggid/password-pepper)
```

## Key Rotation

### Rotation Procedure (Zero-Downtime)

```
1. Set new pepper as PASSWORD_PEPPER_NEW
2. Keep old pepper as PASSWORD_PEPPER_OLD
3. Verify: try NEW pepper first, fall back to OLD
4. On successful login with OLD pepper: re-hash with NEW pepper
5. After all users migrated (monitor): remove OLD pepper
```

### Rotation Code Pattern

```go
func VerifyPassword(password, storedHash string) (bool, error) {
    // Try current pepper
    if verifyWithPepper(password, storedHash, currentPepper) {
        return true, nil
    }

    // Try previous pepper (during rotation grace period)
    if previousPepper != "" && verifyWithPepper(password, storedHash, previousPepper) {
        // Re-hash with new pepper
        newHash := hashWithPepper(password, currentPepper)
        repo.UpdatePasswordHash(userID, newHash)
        return true, nil
    }

    return false, nil
}
```

### Rotation Schedule

| Frequency | Trigger |
|----------|---------|
| Every 365 days | Scheduled |
| On security incident | Emergency |
| On personnel turnover | Precautionary |

## Backward Compatibility

### No Pepper → With Pepper (Initial Deployment)

Existing password hashes (without pepper) remain valid until users log in:

```go
func VerifyPassword(password, storedHash string) (bool, error) {
    // Try with pepper first
    if verifyWithPepper(password, storedHash, pepper) {
        return true, nil
    }

    // Try without pepper (backward compat)
    if verifyWithoutPepper(password, storedHash) {
        // Upgrade to peppered hash
        newHash := hashWithPepper(password, pepper)
        repo.UpdatePasswordHash(userID, newHash)
        return true, nil
    }

    return false, nil
}
```

Users are transparently migrated to peppered hashes on their next login.

## Disaster Recovery

**CRITICAL**: If the pepper is lost, ALL password hashes become unverifiable.

### Mitigations

1. Store pepper in multiple secrets managers (Vault + AWS SM)
2. Include pepper in database backup scripts
3. Document pepper in sealed envelope (offline backup)
4. Test pepper recovery quarterly

### Recovery Without Pepper

If pepper is irrecoverably lost:
1. Generate new pepper
2. Force password reset for ALL users
3. Send reset emails
4. Users set new passwords (hashed with new pepper)

## Security Checklist

- [ ] Pepper is 32+ bytes of crypto-random data
- [ ] Pepper stored in secrets manager (not in source code)
- [ ] `.gitignore` includes `keys.env`
- [ ] Pepper backed up to secondary location
- [ ] Rotation procedure documented
- [ ] Backward compatibility tested
- [ ] HSM/KMS integration evaluated for production

## See Also

- [Password Policy Guide](password-policy-guide.md)
- [Key Management Lifecycle](../research/key-management-lifecycle.md)
- [Security Audit Checklist](security-audit-checklist.md)
