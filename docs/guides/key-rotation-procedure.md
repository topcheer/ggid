# Key Rotation Procedure

Step-by-step zero-downtime rotation for JWT signing keys, JWKS, SAML certs, encryption keys, and mTLS certificates.

## JWT Signing Key Rotation

### Zero-Downtime Procedure

```
T+0:   Generate new RSA-2048 key pair
T+0:   Add new public key to JWKS endpoint (alongside old key)
T+0:   Switch signing to new key (new kid)
T+0:   Publish both keys in JWKS
T+24h: Old tokens expired (TTL: 15min, but allow 24h for cached JWKS)
T+24h: Remove old key from JWKS
T+24h: Destroy old private key
```

### GGID Rotation Command

```bash
# Trigger rotation
curl -X POST https://api.ggid.example.com/api/v1/admin/keys/rotate \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"type":"jwt-signing","grace_period_hours":24}'
```

## SAML Certificate Rotation

```
1. Generate new cert: openssl req -x509 -newkey rsa:2048 -keyout saml-new.key -out saml-new.crt -days 365 -nodes
2. Upload new cert to IdP (keep old during transition)
3. Deploy new cert to GGID
4. Test SAML login
5. Remove old cert from IdP after 48h
```

## Encryption Key (AES-256-GCM) Rotation

```
1. Generate new key
2. Decrypt with old key → re-encrypt with new key (batch)
3. Deploy new key
4. Verify all data accessible
5. Destroy old key
```

## mTLS Certificate Rotation

```
1. Generate new cert signed by same CA
2. Add new cert to trust pool (alongside old)
3. Deploy new cert to servers + clients
4. After 24h: remove old cert
```

### cert-manager (Kubernetes)

```yaml
spec:
  duration: 2160h      # 90 days
  renewBefore: 360h     # 15 days before expiry
```

## Rotation Schedule

| Key Type | Frequency | Grace Period |
|---------|----------|-------------|
| JWT signing | 90 days | 24h |
| TLS server | 90 days | 7d |
| gRPC mTLS | 90 days | 24h |
| SAML signing | 365 days | 7d |
| Encryption (AES) | 365 days | N/A (re-encrypt) |
| Password pepper | 365 days | Until all re-hash |
| Client secrets | 90 days | 24h |
| API keys | 90 days | Immediate (old revoked) |

## Checklist

- [ ] Rotation automated where possible
- [ ] Grace period configured per key type
- [ ] Old keys destroyed after grace period
- [ ] Rotation tested in staging first
- [ ] Emergency rotation procedure documented
- [ ] Audit log records all rotations

## See Also

- [Key Management Lifecycle](../research/key-management-lifecycle.md)
- [HSM Integration](hsm-integration.md)
- [Secrets Management](secrets-management.md)
