# Federation Metadata Management

This guide covers SAML metadata, OIDC discovery and JWKS, metadata refresh strategies, signing and validation, aggregation, trust chain verification, versioning, expiry handling, and GGID's metadata management.

## SAML Metadata

### EntityDescriptor

The root element of SAML metadata describes a federation entity (SP or IdP):

```xml
<EntityDescriptor entityID="https://auth.ggid.example.com/saml/metadata"
                  xmlns="urn:oasis:names:tc:SAML:2.0:metadata">

  <!-- IDP role -->
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <KeyDescriptor use="encryption">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
                         Location="https://idp.example.com/sso"/>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                         Location="https://idp.example.com/sso"/>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
  </IDPSSODescriptor>

  <!-- SP role -->
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <ds:KeyInfo>
        <ds:X509Data>
          <ds:X509Certificate>...</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </KeyDescriptor>
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
                              Location="https://auth.ggid.example.com/saml/acs"
                              index="0" isDefault="true"/>
    <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
                         Location="https://auth.ggid.example.com/saml/slo"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

### Key Metadata Elements

| Element | Purpose |
|---|---|
| `EntityDescriptor` | Root — identifies entity by entityID |
| `IDPSSODescriptor` | IdP role — SSO service endpoints, signing certs |
| `SPSSODescriptor` | SP role — ACS endpoints, signing/encryption certs |
| `KeyDescriptor` | Certificate for signing or encryption |
| `SingleSignOnService` | IdP SSO endpoint with binding |
| `AssertionConsumerService` | SP ACS endpoint with binding |
| `SingleLogoutService` | SLO endpoint with binding |
| `NameIDFormat` | Supported NameID formats |

### Metadata Extensions

```xml
<Extensions>
  <!-- UI information -->
  <mdui:UIInfo>
    <mdui:DisplayName xml:lang="en">GGID Identity Platform</mdui:DisplayName>
    <mdui:Logo width="100" height="40">https://auth.ggid.example.com/logo.png</mdui:Logo>
  </mdui:UIInfo>

  <!-- Discovery service -->
  <mdui:DiscoHints>
    <mdui:DomainHint>example.com</mdui:DomainHint>
  </mdui:DiscoHints>
</Extensions>
```

## OIDC Discovery + JWKS

### Discovery Document

```bash
GET /.well-known/openid-configuration
```

```json
{
  "issuer": "https://auth.ggid.example.com",
  "authorization_endpoint": "https://auth.ggid.example.com/oauth/authorize",
  "token_endpoint": "https://auth.ggid.example.com/oauth/token",
  "userinfo_endpoint": "https://auth.ggid.example.com/oauth/userinfo",
  "jwks_uri": "https://auth.ggid.example.com/.well-known/jwks.json",
  "response_types_supported": ["code", "code id_token"],
  "grant_types_supported": ["authorization_code", "refresh_token", "client_credentials"],
  "subject_types_supported": ["public", "pairwise"],
  "id_token_signing_alg_values_supported": ["RS256", "ES256", "EdDSA"],
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "private_key_jwt"],
  "claims_supported": ["sub", "name", "email", "roles", "tenant_id"],
  "code_challenge_methods_supported": ["S256"],
  "require_pkce": true
}
```

### JWKS (JSON Web Key Set)

```bash
GET /.well-known/jwks.json
```

```json
{
  "keys": [
    {
      "kid": "2026-01",
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "n": "modulus-base64url",
      "e": "AQAB"
    },
    {
      "kid": "2026-07",
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "n": "modulus-base64url-new",
      "e": "AQAB"
    }
  ]
}
```

### JWKS Key Selection

Clients select the correct key for JWT verification by matching the `kid` header:

```go
func getKeyFromJWKS(kid string, jwks *JWKSet) (interface{}, error) {
    for _, key := range jwks.Keys {
        if key.KID == kid {
            return parseRSAKey(key.N, key.E)
        }
    }
    return nil, ErrKeyNotFound
}
```

## Metadata Refresh Strategy

### Polling

Periodic fetch of metadata at configured intervals:

```yaml
federation:
  refresh:
    strategy: "polling"
    interval: 24h
    jitter: 1h  # Random ±1h to avoid thundering herd
    timeout: 30s
    retry:
      max_attempts: 3
      backoff: 5s
```

### Webhook (Push)

Metadata changes trigger immediate notification:

```yaml
federation:
  refresh:
    strategy: "webhook"
    webhook_endpoint: "https://auth.ggid.example.com/federation/webhook"
    webhook_secret: "<hmac-secret>"
```

### Conditional Fetch (ETag/Last-Modified)

```go
func refreshMetadata(url string, etag string) (*Metadata, string, error) {
    req, _ := http.NewRequest("GET", url, nil)
    if etag != "" {
        req.Header.Set("If-None-Match", etag)
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, "", err
    }
    if resp.StatusCode == 304 {
        return nil, etag, nil  // Not modified
    }
    metadata := parseMetadata(resp.Body)
    return metadata, resp.Header.Get("ETag"), nil
}
```

## Metadata Signing and Validation

### Signed Metadata

Federation metadata is signed by the federation operator:

```xml
<EntityDescriptor entityID="...">
  <!-- Content -->
  <ds:Signature>
    <ds:SignedInfo>
      <ds:CanonicalizationMethod Algorithm="..."/>
      <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
      <ds:Reference>
        <ds:Transforms>...</ds:Transforms>
        <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
        <ds:DigestValue>...</ds:DigestValue>
      </ds:Reference>
    </ds:SignedInfo>
    <ds:SignatureValue>...</ds:SignatureValue>
    <ds:KeyInfo>
      <ds:X509Data><ds:X509Certificate>...</ds:X509Certificate></ds:X509Data>
    </ds:KeyInfo>
  </ds:Signature>
</EntityDescriptor>
```

### Validation

```go
func validateMetadata(metadata []byte, trustedRoots []*x509.Certificate) error {
    // Parse XML and extract signature
    doc := parseXML(metadata)
    sig := extractSignature(doc)
    if sig == nil {
        return ErrUnsignedMetadata
    }

    // Verify signature
    cert := extractCertificate(sig)
    if err := verifyCertChain(cert, trustedRoots); err != nil {
        return fmt.Errorf("cert chain verification: %w", err)
    }

    // Verify signature over metadata content
    signedContent := extractSignedContent(doc)
    if err := verifyXMLSignature(sig, cert, signedContent); err != nil {
        return fmt.Errorf("signature verification: %w", err)
    }

    return nil
}
```

## Metadata Aggregation

### Federation Aggregate

A federation operator aggregates metadata from all members:

```xml
<EntitiesDescriptor Name="urn:ggid:federation:2026">
  <EntityDescriptor entityID="https://idp1.example.com">...</EntityDescriptor>
  <EntityDescriptor entityID="https://idp2.example.com">...</EntityDescriptor>
  <EntityDescriptor entityID="https://sp1.example.com">...</EntityDescriptor>
</EntitiesDescriptor>
```

### GGID as Aggregator

```yaml
federation:
  aggregation:
    enabled: true
    members:
      - entity_id: "https://idp.example.com"
        metadata_url: "https://idp.example.com/metadata"
        refresh_interval: 24h
      - entity_id: "https://sp.example.com"
        metadata_url: "https://sp.example.com/metadata"
        refresh_interval: 24h
    aggregate_url: "https://auth.ggid.example.com/federation/aggregate"
    sign_aggregate: true
```

## Trust Chain Verification

### Trust Chain

```
Federation Root CA
    └── Federation Operator Certificate
        └── Entity Metadata Signature
```

### Verification Steps

1. Verify entity metadata signature
2. Verify signing certificate chains to federation root
3. Check federation root is in trusted roots list
4. Verify certificate not expired
5. Check entity is not revoked

```go
func verifyTrustChain(metadata []byte, config *FederationConfig) error {
    // Step 1: Verify metadata signature
    if err := validateMetadata(metadata, config.TrustedRoots); err != nil {
        return err
    }

    // Step 2: Check entity is trusted
    entityID := extractEntityID(metadata)
    if !config.IsTrustedEntity(entityID) {
        return ErrUntrustedEntity
    }

    // Step 3: Check revocation
    if config.IsRevoked(entityID) {
        return ErrRevokedEntity
    }

    // Step 4: Check validity period
    validUntil := extractValidUntil(metadata)
    if time.Now().After(validUntil) {
        return ErrMetadataExpired
    }

    return nil
}
```

## Metadata Versioning

### Version Tracking

```yaml
federation:
  versioning:
    track_changes: true
    store_history: true
    max_versions: 10  # Keep last 10 versions
    diff_on_change: true  # Log what changed
```

### Change Detection

```go
func detectMetadataChanges(old, new *EntityDescriptor) []MetadataChange {
    var changes []MetadataChange

    // Check certificate changes
    oldCerts := extractCertificates(old)
    newCerts := extractCertificates(new)
    if !certsEqual(oldCerts, newCerts) {
        changes = append(changes, MetadataChange{
            Type:     "certificate",
            Old:      oldCerts,
            New:      newCerts,
            Severity: "high",
        })
    }

    // Check endpoint changes
    oldEndpoints := extractEndpoints(old)
    newEndpoints := extractEndpoints(new)
    if !endpointsEqual(oldEndpoints, newEndpoints) {
        changes = append(changes, MetadataChange{
            Type:     "endpoint",
            Old:      oldEndpoints,
            New:      newEndpoints,
            Severity: "medium",
        })
    }

    return changes
}
```

## Expiry Handling

### Valid Until

SAML metadata includes `validUntil` attribute:

```xml
<EntityDescriptor entityID="..." validUntil="2026-08-12T00:00:00Z">
```

### Cache Duration

```xml
<EntityDescriptor entityID="..." cacheDuration="PT24H">
```

### Expiry Strategy

```go
func handleMetadataExpiry(entity *FederationEntity) error {
    if time.Now().After(entity.ValidUntil) {
        // Metadata expired
        switch entity.Config.OnExpiry {
        case "use_cached":
            // Continue using cached metadata
            log.Warn("metadata expired, using cached: " + entity.EntityID)
        case "reject":
            // Reject all assertions from this entity
            log.Error("metadata expired, rejecting: " + entity.EntityID)
            entity.Trusted = false
        case "refresh":
            // Attempt immediate refresh
            return refreshEntityMetadata(entity)
        }
    }
    return nil
}
```

### Configuration

```yaml
federation:
  expiry:
    warning_threshold: 7d  # Warn 7 days before expiry
    on_expiry: "refresh"   # refresh, use_cached, or reject
    max_cache_duration: 48h  # Don't use cached metadata > 48h past expiry
    alert_admin: true
```

## GGID Metadata Management

### Configuration

```yaml
federation:
  enabled: true
  saml:
    sp_metadata_url: "https://auth.ggid.example.com/saml/metadata"
    idp_metadata:
      - entity_id: "https://idp.example.com"
        metadata_url: "https://idp.example.com/metadata"
        refresh_interval: 24h
        signed: true
        trusted_roots: "/etc/ggid/federation/roots/"
  oidc:
    discovery_url: "https://auth.ggid.example.com/.well-known/openid-configuration"
    jwks_url: "https://auth.ggid.example.com/.well-known/jwks.json"
    jwks_refresh_interval: 1h
  refresh:
    strategy: "polling"
    interval: 24h
    jitter: 1h
    timeout: 30s
  validation:
    require_signed: true
    verify_trust_chain: true
    check_revocation: true
    check_expiry: true
  versioning:
    track_changes: true
    store_history: true
    max_versions: 10
  expiry:
    warning_threshold: 7d
    on_expiry: "refresh"
    alert_admin: true
```

### API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/federation/entities` | GET | List federation entities |
| `/api/v1/federation/entities/{id}` | GET | Get entity metadata |
| `/api/v1/federation/refresh` | POST | Force metadata refresh |
| `/api/v1/federation/aggregate` | GET | Get aggregated metadata |
| `/api/v1/federation/changes` | GET | Get metadata change history |

## Best Practices

1. **Always validate metadata signatures** — Never trust unsigned metadata
2. **Refresh regularly** — Don't use stale metadata
3. **Handle expiry gracefully** — Don't break on metadata expiration
4. **Track changes** — Log when metadata changes for security analysis
5. **Alert on cert changes** — Certificate changes may indicate rotation or attack
6. **Use signed aggregates** — Federation aggregates must be signed
7. **Cache with TTL** — Don't fetch on every request, but don't cache forever
8. **Verify trust chain** — Don't just check the leaf certificate
9. **Monitor for revocation** — Check if entities have been revoked
10. **Test metadata refresh** — Ensure refresh works before metadata expires
