# Key Management Lifecycle

This guide covers key types, lifecycle stages, key storage, key access control, key usage audit, emergency key rotation, dual control, and GGID's key management.

## Key Types

| Key Type | Purpose | Algorithm | Rotation |
|---|---|---|---|
| JWT signing | Token signing | RS256/ES256/EdDSA | 90 days |
| JWT encryption | Token encryption | RSA-OAEP | 90 days |
| Session signing | Session cookie signing | HS256/RS256 | 90 days |
| Refresh token | Token signing | RS256 | 90 days |
| Agent token | Agent identity | RS256 | 90 days |
| DPoP key | Client proof-of-possession | ES256/EdDSA | Per client |
| SAML signing | Assertion signing | RSA-SHA256 | 365 days |
| SAML encryption | Assertion encryption | RSA-OAEP | 365 days |
| Database DEK | Data encryption | AES-256-GCM | 365 days |
| Database KEK | Key encryption | AES-256 | 180 days |
| Database MEK | Master encryption | AES-256 | 90 days |
| Webhook signing | HMAC verification | HMAC-SHA256 | 90 days |
| Audit hash | Hash chain | SHA-256 | N/A |
| Password pepper | Password hashing | Random bytes | 365 days |

## Lifecycle Stages

### 1. Generate

```go
// RSA key pair
func generateRSAKey(bits int) (*rsa.PrivateKey, error) {
    return rsa.GenerateKey(rand.Reader, bits)
}

// ECDSA key pair
func generateECDSAKey() (*ecdsa.PrivateKey, error) {
    return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// Ed25519 key pair
func generateEd25519Key() (ed25519.PrivateKey, ed25519.PublicKey, error) {
    return ed25519.GenerateKey(rand.Reader)
}

// AES symmetric key
func generateAESKey() ([]byte, error) {
    key := make([]byte, 32)  // 256-bit
    _, err := rand.Read(key)
    return key, err
}
```

### 2. Distribute

| Method | Use Case | Security |
|---|---|---|
| KMS API call | Cloud-managed keys | High |
| HSM PKCS#11 | Hardware-backed | Very High |
| Vault transit | HashiCorp Vault | High |
| Kubernetes secret | Container env | Medium |
| Config file | On-premise | Low (not recommended) |

### 3. Use

```go
// Sign JWT
token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
token.Header["kid"] = keyID
signed, err := token.SignedString(privateKey)

// Verify JWT
parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
    kid := t.Header["kid"].(string)
    key := jwks.GetKey(kid)
    return key, nil
})
```

### 4. Rotate

```
1. Generate new key pair
2. Publish new public key to JWKS
3. Dual signing period (both keys valid)
4. Switch signing to new key
5. Wait for old tokens to expire
6. Archive old key
7. Remove old key from JWKS
```

### 5. Archive

```yaml
key_management:
  archive:
    enabled: true
    storage: "encrypted_s3"
    retention: 7y  # Keep archived keys for 7 years
    encryption: "AES-256-GCM"
    access_log: true
```

### 6. Destroy

```go
func destroyKey(keyID string) error {
    // Verify key is past retention requirement
    key := getKey(keyID)
    if time.Since(key.ArchivedAt) < 7*365*24*time.Hour {
        return ErrKeyWithinRetention
    }
    
    // Secure wipe from HSM/KMS
    err := kms.DestroyKey(keyID)
    if err != nil {
        return err
    }
    
    // Audit
    audit.Log("key_destroyed", keyID, "key_management")
    
    return nil
}
```

## Key Storage

### HSM (Hardware Security Module)

| Feature | Description |
|---|---|
| Security | Highest — keys never leave hardware |
| Performance | Fast (hardware accelerated) |
| Cost | High |
| Use case | Master keys, signing keys |
| Standards | FIPS 140-2 Level 3+ |

### KMS (Key Management Service)

| Provider | Features |
|---|---|
| AWS KMS | HSM-backed, per-key IAM, audit trail |
| Google Cloud KMS | HSM-backed, per-key IAM, rotation |
| Azure Key Vault | HSM-backed, per-key RBAC, backup |
| HashiCorp Vault | Self-hosted, transit engine, auto-rotation |

### Software (In-Memory)

| Feature | Description |
|---|---|
| Security | Lowest — keys in process memory |
| Performance | Fastest |
| Use case | Session keys, ephemeral keys |
| Risk | Key extraction via memory dump |

## Key Access Control

### Per-Key RBAC

```yaml
key_management:
  access_control:
    jwt_signing:
      roles: ["auth-service"]
      operations: ["sign", "verify"]
    jwt_verify:
      roles: ["gateway", "all-services"]
      operations: ["verify"]
    saml_signing:
      roles: ["oauth-service"]
      operations: ["sign"]
    database_mek:
      roles: ["kms-service"]
      operations: ["encrypt", "decrypt"]
      require_dual_control: true
```

### Access Enforcement

```go
func authorizeKeyAccess(userID, keyID, operation string) error {
    key := getKey(keyID)
    user := getUser(userID)
    
    // Check if user's role is allowed
    allowed := false
    for _, role := range key.AllowedRoles {
        if user.HasRole(role) {
            allowed = true
            break
        }
    }
    if !allowed {
        return ErrKeyAccessDenied
    }
    
    // Check operation
    if !contains(key.AllowedOperations, operation) {
        return ErrOperationNotAllowed
    }
    
    // Check dual control
    if key.RequireDualControl && !hasDualApproval(userID, keyID) {
        return ErrDualControlRequired
    }
    
    // Audit
    audit.Log("key_access", userID, keyID, operation)
    
    return nil
}
```

## Key Usage Audit

### What to Log

| Event | Fields |
|---|---|
| Key generation | key_id, type, algorithm, generated_by |
| Key access | key_id, user_id, operation, timestamp |
| Key rotation | old_key_id, new_key_id, rotated_by |
| Key archival | key_id, archived_by, archive_location |
| Key destruction | key_id, destroyed_by, verified_by |
| Access denied | key_id, user_id, reason |

### Audit Query

```bash
GET /api/v1/admin/keys/audit?key_id=jwt-2026-07&start=2026-07-01&end=2026-07-31
```

## Emergency Key Rotation

### Compromise Response

```
1. Detect compromise (key leak, unauthorized use, breach)
2. Immediately generate new key pair
3. Publish new public key to JWKS
4. Revoke all tokens signed with compromised key
5. Remove compromised key from JWKS
6. Force all users to re-authenticate
7. Audit all actions taken with compromised key
8. Notify security team and CISO
9. Document incident
10. Destroy compromised key after investigation
```

### Emergency Rotation Script

```bash
#!/bin/bash
# Emergency JWT key rotation
echo "EMERGENCY KEY ROTATION"

# Generate new key
openssl genrsa -out /etc/ggid/jwt/new-private.pem 2048
openssl rsa -in /etc/ggid/jwt/new-private.pem -pubout -out /etc/ggid/jwt/new-public.pem

# Update config to use new key
sed -i 's/private_key.*/private_key: \/etc\/ggid\/jwt\/new-private.pem/' /etc/ggid/config.yaml

# Reload service
systemctl reload ggid-auth

# Revoke all existing tokens
redis-cli SET "jwt:revoked:old-key-id" "1" EX 86400

# Notify
curl -X POST $SLACK_WEBHOOK -d '{"text":"EMERGENCY: JWT key rotated, all users must re-authenticate"}'
```

## Dual Control

### What is Dual Control?

Sensitive operations require two authorized people to approve:

| Operation | Dual Control? |
|---|---|
| Generate master key | Yes |
| Rotate master key | Yes |
| Destroy any key | Yes |
| Emergency rotation | No (single admin can act) |
| Access signing key | No (service uses automatically) |
| View key audit log | No |

### Implementation

```go
func requestDualControl(keyID, operation, requestorID string) error {
    request := &DualControlRequest{
        ID:          uuid.New().String(),
        KeyID:       keyID,
        Operation:   operation,
        Requestor:   requestorID,
        Status:      "pending_approval",
        CreatedAt:   time.Now(),
        ExpiresAt:   time.Now().Add(24 * time.Hour),
    }
    storeRequest(request)
    notifyApprovers(request)
    return nil
}

func approveDualControl(requestID, approverID string) error {
    request := getRequest(requestID)
    if request.Requestor == approverID {
        return ErrCannotSelfApprove  // Requestor can't approve own request
    }
    request.Approver = approverID
    request.Status = "approved"
    request.ApprovedAt = time.Now()
    return executeOperation(request)
}
```

## GGID Key Management

### Configuration

```yaml
key_management:
  provider: "aws-kms"  # or "vault", "azure-keyvault", "hsm"
  
  keys:
    jwt_signing:
      algorithm: "RS256"
      key_size: 2048
      rotation: 90d
      overlap: 7d
      storage: "kms"
    
    saml_signing:
      algorithm: "RSA-SHA256"
      key_size: 2048
      rotation: 365d
      overlap: 30d
      storage: "file"  # PEM files
    
    database_mek:
      algorithm: "AES-256"
      rotation: 90d
      storage: "kms"
      dual_control: true
    
    webhook_signing:
      algorithm: "HMAC-SHA256"
      rotation: 90d
      storage: "env"
  
  access_control:
    per_key_rbac: true
    audit_all_access: true
  
  dual_control:
    operations: ["generate_master", "rotate_master", "destroy_key"]
    approval_timeout: 24h
    cannot_self_approve: true
  
  emergency:
    auto_revoke_tokens: true
    force_reauth: true
    notify_ciso: true
  
  archive:
    enabled: true
    retention: 7y
    encrypted: true
```

## Best Practices

1. **Use HSM/KMS** — Never store master keys in files
2. **Rotate regularly** — 90 days for signing, 365 for SAML
3. **Dual control for masters** — Two people for sensitive key operations
4. **Audit all access** — Every key use is logged
5. **Plan emergency rotation** — Have script ready for compromise
6. **Overlap during rotation** — Both old and new keys valid during transition
7. **Archive before destroy** — Keep keys for audit/decryption of old data
8. **Never log key material** — Log key ID, not the key itself
9. **Separate duties** — Key generator ≠ key user ≠ key destroyer
10. **Test key recovery** — Ensure you can recover from KMS outage