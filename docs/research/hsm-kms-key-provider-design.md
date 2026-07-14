# HSM/KMS Key Provider Architecture Design

*Author: arch/PM*
*Date: 2026-07-15*
*Target: v0.2.0 release*
*Source: docs/research/hsm-kms-integration.md, docs/research/vault-kms-integration.md*

## 1. Problem

GGID currently stores JWT/SAML signing keys as PEM files on disk (`configs/rsa_private.pem`). This is acceptable for development but not for production because:

- Keys are exposed to filesystem-level attacks
- No HSM-backed protection for compliance-sensitive deployments
- No support for cloud-native KMS key rotation
- No centralized key provider abstraction

## 2. Goal

Introduce a `pkg/crypto.KeyProvider` interface so auth/oauth services can sign tokens using local PEM, PKCS#11 HSM, AWS KMS, GCP KMS, Azure Key Vault, or HashiCorp Vault Transit without changing call sites.

## 3. Design

### 3.1 Interface (already implemented in `pkg/crypto/key_provider.go`)

```go
type KeyProvider interface {
    Metadata() KeyMetadata
    Public() crypto.PublicKey
    Signer() crypto.Signer
    Close() error
}
```

### 3.2 Configuration

A single `KeyProviderConfig` selects the provider. Environment variable `GGID_KEY_PROVIDER` sets the provider type. Service-specific `GGID_KEY_ID` and provider-specific env vars are documented in `docs/deployment.md`.

### 3.3 Implementation phases

| Phase | Provider | Owner | Test Strategy |
|-------|----------|-------|---------------|
| 0 | Local PEM (default) | arch | Unit test in pkg/crypto |
| 1 | PKCS#11 (SoftHSM2) | backend | Integration test with build tag `pkcs11` |
| 2 | AWS KMS | backend | Mock unit test + manual integration |
| 3 | GCP KMS | backend | Mock unit test + manual integration |
| 4 | Azure Key Vault | backend | Mock unit test + manual integration |
| 5 | HashiCorp Vault Transit | backend | Mock unit test + Docker integration test |

### 3.4 Service integration

- `services/auth/cmd/main.go`: initialize `KeyProvider` from env, inject into `TokenService`
- `services/oauth/cmd/main.go`: initialize `KeyProvider`, inject into `OAuthService`
- `services/gateway/internal/middleware/jwks.go`: expose `KeyProvider.Public()` on JWKS endpoint
- `pkg/saml`: accept `KeyProvider` for SAML signing

### 3.5 Migration path

1. Default remains `local` (backward compatible)
2. New deployments can set `GGID_KEY_PROVIDER=pkcs11` and provider env vars
3. Existing `configs/rsa_private.pem` continues to work
4. When a cloud KMS is configured, signing latency increases by 5–50ms; caching public key is required

## 4. Verification

- `go test ./pkg/crypto/...` passes
- `make test` passes
- E2E: login → JWT → JWKS returns correct public key
- Release: tag v0.2.0, Docker images published, npm @ggid/sdk published

## 5. Deployment notes

- Add provider-specific env vars to `deploy/docker-compose.yaml` and `deploy/docker-compose.prod.yaml`
- Provide `deploy/softhsm2/` setup for development
- Update `docs/deployment.md`

## 6. Backlog tracking

- See `docs/platform-completeness-report.md` gap #14
- See `docs/team-backlog.md` for implementation tasks

## 7. Risks

- PKCS#11 requires CGo and system libraries; may complicate cross-compilation
- Cloud KMS adds latency and network dependency
- Key rotation must be coordinated across all services sharing the same key
- JWKS endpoint must serve the correct public key for the active key provider

## 8. Acceptance criteria

- [ ] `pkg/crypto` KeyProvider interface merged with tests
- [ ] At least one non-local provider (PKCS#11) implemented and tested
- [ ] Auth and OAuth services use `KeyProvider` for JWT signing
- [ ] JWKS endpoint exposes public key from `KeyProvider`
- [ ] Docker compose stack still works with default local PEM provider
- [ ] v0.2.0 released with release notes mentioning HSM/KMS support

## Next step

Backend team implements PKCS#11 provider using SoftHSM2 in CI.
