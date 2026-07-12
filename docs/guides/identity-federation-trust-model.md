# Identity Federation Trust Model

This guide covers trust relationship types, trust establishment, trust lifecycle, trust chain validation, trust framework interoperability, and GGID's trust model.

## Trust Relationship Types

| Type | Description | Example |
|---|---|---|
| Bilateral | Direct trust between two parties | GGID ↔ Okta |
| Federated | Trust within a federation of multiple parties | eduGAIN, InCommon |
| Bridged | Trust through a bridge/hub | GGID → Bridge → Multiple IdPs |
| Transitive | A trusts B, B trusts C → A trusts C | Federation cascading |

### Comparison

| Type | Scalability | Trust Scope | Complexity | Use Case |
|---|---|---|---|---|
| Bilateral | Low (O(n²)) | Two parties | Low | Enterprise SSO |
| Federated | High | All federation members | Medium | Research/education |
| Bridged | Medium | Via bridge | Medium | Cross-federation |
| Transitive | High but risky | Cascading | High | Rare, risky |

## Trust Establishment

### Steps

```
1. Metadata Exchange → Both parties share SAML/OIDC metadata
2. Certificate Pinning → Pin IdP signing certificate
3. Configuration → Configure entity ID, endpoints, certificates
4. Testing → Verify authentication flow works
5. Activation → Enable trust relationship for production
```

### Metadata Exchange

```yaml
federation:
  trust:
    establishment:
      metadata_exchange:
        method: "url"  # or "file"
        sp_metadata_url: "https://auth.ggid.example.com/saml/metadata"
        idp_metadata_url: "https://idp.example.com/metadata"
      cert_pinning:
        enabled: true
        pin_algorithm: "sha256"
        pin_location: "config"  # or "database"
      testing:
        required: true
        test_users: ["test@example.com"]
      activation:
        requires_approval: true
        approver: "security-admin"
```

### Certificate Pinning

```go
func establishTrust(idpMetadata []byte) error {
    cert := extractCertificate(idpMetadata)
    pin := sha256.Sum256(cert.Raw)
    storePin("idp-signing-cert-pin", hex.EncodeToString(pin[:]))
    audit.Log("trust_established", "cert_pinned", hex.EncodeToString(pin[:]))
    return nil
}
```

## Trust Lifecycle

```
Establish → Monitor → Renew → Revoke
    ↑                        ↓
    └── Re-establish ←───────┘
```

### Establish

- Exchange metadata
- Pin certificates
- Configure endpoints
- Test authentication
- Activate

### Monitor

- Verify metadata freshness
- Check certificate expiry
- Monitor auth success rate
- Alert on auth failure spike
- Detect metadata changes

### Renew

```
1. IdP generates new signing certificate
2. IdP publishes new metadata with both old + new cert
3. GGID fetches new metadata, sees both certs
4. Both certs valid during overlap period (7-30 days)
5. IdP switches to new cert for signing
6. After overlap, old cert removed from metadata
7. GGID removes old pin
```

### Revoke

| Trigger | Action |
|---|---|
| Security incident | Immediately revoke trust + block all auth |
| Certificate compromise | Revoke cert + re-establish with new cert |
| Contract termination | Graceful shutdown + revoke after transition |
| IdP compromise | Revoke all trust + notify users |

```go
func revokeTrust(entityID string, reason string) error {
    // Block all authentication from this entity
    redis.Set(ctx, "trust:revoked:"+entityID, reason, 0)
    
    // Remove pinned certificates
    removePin(entityID)
    
    // Invalidate cached metadata
    invalidateMetadata(entityID)
    
    // Terminate active sessions from this IdP
    terminateFederatedSessions(entityID)
    
    // Audit
    audit.Log("trust_revoked", entityID, reason)
    
    // Notify
    notifyAdmin("Trust revoked: " + entityID + " — " + reason)
    
    return nil
}
```

## Trust Chain Validation

### Validation Steps

1. Verify entity metadata signature
2. Verify metadata signed by federation root
3. Check entity is not revoked
4. Check metadata is not expired
5. Verify certificate chain to trusted root
6. Pin check: certificate matches pinned value

```go
func validateTrustChain(entity *FederationEntity) error {
    // Check revocation
    if isRevoked(entity.ID) { return ErrRevoked }
    
    // Check expiry
    if time.Now().After(entity.ValidUntil) { return ErrExpired }
    
    // Verify metadata signature
    if err := verifyMetadataSignature(entity); err != nil {
        return err
    }
    
    // Verify cert chain
    if err := verifyCertChain(entity.Cert, trustedRoots); err != nil {
        return err
    }
    
    // Check pin
    if entity.PinnedHash != "" {
        certHash := sha256.Sum256(entity.Cert.Raw)
        if hex.EncodeToString(certHash[:]) != entity.PinnedHash {
            return ErrPinMismatch
        }
    }
    
    return nil
}
```

## Trust Framework Interoperability

### Frameworks

| Framework | Region | Purpose | Members |
|---|---|---|---|
| eIDAS | EU | Electronic identity | EU member states |
| eduGAIN | Global | Research/education federation | 70+ federations |
| InCommon | US | US education federation | 400+ institutions |
| PICS | Canada | Pan-Canadian trust | Canadian gov |
| Kantara Initiative | Global | Identity assurance | Various |

### Interoperability Configuration

```yaml
federation:
  frameworks:
    eidas:
      assurance_levels: ["substantial", "high"]
      mutual_recognition: true
    eduGAIN:
      metadata_feed: "https://metadata.edugain.org"
      refresh: 24h
      entity_category_support: true
    incommon:
      metadata_feed: "https://incommon.org/metadata"
      refresh: 24h
```

## GGID Trust Model

### Configuration

```yaml
federation:
  trust_model:
    type: "bilateral"  # Default; "federated" for multi-party
    establishment:
      metadata_exchange: true
      cert_pinning: true
      testing_required: true
      approval_required: true
    lifecycle:
      monitoring: true
      auto_renewal: true
      renewal_overlap: 7d
      expiry_warning: 30d
    revocation:
      immediate: true
      terminate_sessions: true
      notify_admin: true
    validation:
      verify_signature: true
      verify_chain: true
      check_revocation: true
      check_expiry: true
      pin_check: true
    frameworks:
      eidas: false
      eduGAIN: false
      incommon: false
```

### Trust Store API

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/federation/trust` | GET | List trust relationships |
| `/api/v1/federation/trust` | POST | Establish new trust |
| `/api/v1/federation/trust/{id}` | GET | Get trust details |
| `/api/v1/federation/trust/{id}` | DELETE | Revoke trust |
| `/api/v1/federation/trust/{id}/renew` | POST | Renew trust |

## Best Practices

1. **Always pin certificates** — Don't trust metadata blindly
2. **Monitor trust health** — Track auth success rates per IdP
3. **Plan cert renewal** — Overlap period prevents downtime
4. **Revoke quickly on incident** — Don't leave compromised trust active
5. **Test before activation** — Verify auth works before production
6. **Require approval** — Trust establishment needs admin approval
7. **Document trust relationships** — Track why each was established
8. **Audit trust changes** — Log all establish/renew/revoke events
9. **Support multiple frameworks** — Enable interop as needed
10. **Alert on metadata changes** — Unexpected changes may indicate attack