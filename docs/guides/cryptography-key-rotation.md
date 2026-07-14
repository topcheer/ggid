# Cryptography Key Rotation

Rotation cadence by key type, zero-downtime overlap, automated pipeline, emergency revocation, backward compat, and HSM-backed rotation.

## Rotation Cadence

| Key Type | Period | Grace Period | Method |
|----------|--------|-------------|--------|
| JWT signing (RS256) | 90 days | 15 days | Dual-publish JWKS |
| JWT signing (ES256) | 90 days | 15 days | Dual-publish JWKS |
| SAML signing | 365 days | 30 days | Dual cert in metadata |
| SAML encryption | 365 days | 30 days | Dual cert in metadata |
| mTLS service cert | 90 days | 15 days | cert-manager auto |
| DB encryption (DEK) | Per-write (fresh) | — | Envelope encryption |
| Tenant encryption (TEK) | 90 days | 7 days | Re-encrypt DEKs |
| Master encryption (MEK) | 365 days | 30 days | Re-encrypt TEKs |
| OAuth client secret | 180 days | 0 (grace via API) | Generate new, revoke old |
| API key | 90 days | 0 (rolling) | Generate new, revoke old |

## Zero-Downtime Rotation (Overlap Window)

```
Day 0:    Publish new key alongside old key
          JWKS contains: [old-key, new-key]
          Services accept tokens signed by either

Day 1-15: Start signing with new key only
          Still accept old key tokens (grace period)
          Old tokens expire naturally (15min TTL)

Day 15:   Remove old key from JWKS
          Old key revoked from signing
```

### Implementation

```go
type KeyManager struct {
    current  *SigningKey
    previous *SigningKey  // Still valid, used for verification only
}

func (km *KeyManager) Rotate() error {
    newKey, err := generateSigningKey()
    if err != nil { return err }

    // Old key moves to "previous" (verify only)
    km.previous = km.current
    km.current = newKey

    // Publish both in JWKS
    publishJWKS(km.current, km.previous)

    // Schedule old key removal after grace period
    time.AfterFunc(gracePeriod, func() {
        km.previous = nil
        publishJWKS(km.current)  // Only new key now
        audit.Log("key.removed_old", newKey.ID)
    })

    audit.Log("key.rotated", map[string]interface{}{
        "old_kid": km.previous.ID,
        "new_kid": km.current.ID,
    })
    return nil
}
```

## Automated Pipeline

```yaml
rotation_pipeline:
  schedule: "0 3 1 * *"  # 1st of month, 3am

  steps:
    - name: check_keys_needing_rotation
      query: "SELECT * FROM signing_keys WHERE expires_at < NOW() + INTERVAL '15 days'"

    - name: generate_new_key
      action: HSM generate ECDSA P-256

    - name: publish_dual_jwks
      action: Update /.well-known/jwks.json with both keys

    - name: switch_signing
      action: Start signing new tokens with new key only

    - name: wait_grace_period
      duration: 15_days

    - name: remove_old_key
      action: Remove from JWKS, revoke signing capability

    - name: audit
      action: Log rotation event with timestamps
```

## Emergency Revocation

```bash
# Immediate key revocation (suspected compromise)
POST /api/v1/admin/keys/revoke
{
  "key_id": "key-abc",
  "reason": "Suspected key compromise",
  "invalidate_tokens": true   # All tokens signed by this key → revoked
}
```

### Emergency Steps

```
1. Revoke compromised key (remove from JWKS immediately)
2. All tokens signed by that key become invalid
3. Users must re-authenticate
4. Generate new key immediately
5. Publish new JWKS
6. Notify all resource servers to re-fetch JWKS
7. Audit log + security incident
```

## Backward Compatibility

During rotation, two key states coexist:

```go
func VerifyToken(token string) (*Claims, error) {
    // Parse header to get kid
    header := parseHeader(token)
    kid := header["kid"].(string)

    // Try current key
    if kid == km.current.ID {
        return verifyWithKey(token, km.current)
    }

    // Try previous key (grace period)
    if km.previous != nil && kid == km.previous.ID {
        return verifyWithKey(token, km.previous)
    }

    // Unknown key → re-fetch JWKS (maybe rotation just happened)
    jwks := fetchJWKS()
    if key := jwks.Find(kid); key != nil {
        return verifyWithKey(token, key)
    }

    return nil, ErrUnknownKeyID
}
```

## HSM-Backed Rotation

```go
func generateSigningKey() (*SigningKey, error) {
    // Key generated inside HSM, private key never leaves
    keyHandle, err := hsm.GenerateECDSA("P-256")
    if err != nil { return nil, err }

    // Export public key only
    pubKey, err := hsm.ExportPublic(keyHandle)
    if err != nil { return nil, err }

    return &SigningKey{
        ID:        uuid.New(),
        PublicKey: pubKey,
        Handle:    keyHandle,  // HSM reference
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
    }, nil
}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Key nearing expiry | <15 days → rotate now |
| JWKS fetch failures | Any → cached JWKS expiring |
| Unknown kid errors | Spike → clients didn't refresh |
| Emergency revocation | Any → security incident |
| Rotation automation failure | Any → manual intervention needed |

## See Also

- [Credential Vault Architecture](credential-vault-architecture.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [Key Rotation Procedure](key-rotation-procedure.md)
- [JWT Security Best Practices](jwt-security-best-practices.md)
- [Post-Quantum Crypto Migration](post-quantum-crypto-migration.md)
