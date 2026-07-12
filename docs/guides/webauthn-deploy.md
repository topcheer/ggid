# WebAuthn / FIDO2 Production Deployment Guide

This guide covers deploying WebAuthn in production — server configuration, attestation format selection, device registration, recovery codes, conditional UI, and troubleshooting.

> **Related**: [WebAuthn Setup](webauthn-setup.md), [WebAuthn Deep Dive](../research/webauthn-deep-dive.md)

## Server Configuration

```yaml
webauthn:
  rp_id: "ggid.example.com"           # Must match browser origin domain
  rp_name: "GGID"
  origin: "https://ggid.example.com"  # Exact origin
  attestation_conveyance: "direct"    # none | indirect | direct
  user_verification: "required"       # required | preferred | discouraged
  resident_key: "preferred"           # required | preferred | discouraged
  timeout: 60000                       # 60 seconds
  aaguid_allowlist: []                # Empty = accept all authenticators
```

## Attestation Format Selection

| Setting | Attestation | Privacy | Enterprise Control | Use Case |
|---------|------------|---------|-------------------|----------|
| `none` | No attestation | Highest | None | Consumer apps |
| `indirect` | Anonymized | High | Limited | Privacy-conscious enterprise |
| `direct` | Full attestation | Lower | Full (AAGUID filtering) | Regulated enterprise |

**Enterprise**: Use `direct` with AAGUID allowlist to restrict to approved hardware keys.

## Device Registration Flow

### Platform Authenticator (Face ID / Touch ID / Windows Hello)

```bash
# Begin registration
curl -X POST https://api.ggid.example.com/api/v1/webauthn/register/begin \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"authenticator_attachment": "platform"}'
```

### Cross-Platform (Security Key — YubiKey)

```bash
curl -X POST https://api.ggid.example.com/api/v1/webauthn/register/begin \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"authenticator_attachment": "cross-platform"}'
```

### Complete Registration

```bash
curl -X POST https://api.ggid.example.com/api/v1/webauthn/register/finish \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "attestation_object": "base64url...",
    "client_data_json": "base64url...",
    "credential_id": "base64url..."
  }'
```

## Recovery Codes

When a user registers a WebAuthn credential, generate recovery codes:

```bash
curl -X POST https://api.ggid.example.com/api/v1/webauthn/recovery-codes \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

**Response**:
```json
{
  "recovery_codes": [
    "abc12-def34-ghi56",
    "jkl78-mno90-pqr12",
    "...8 more codes..."
  ]
}
```

**Rules**:
- 10 single-use codes per user
- Store as Argon2id hash (never plaintext)
- Display once at generation time
- Regenerable by admin

## Conditional UI (Autofill)

Enable passkey autofill in login forms:

```javascript
if (PublicKeyCredential.isConditionalMediationAvailable()) {
  const credential = await navigator.credentials.get({
    publicKey: { challenge, mediation: "conditional" }
  });
}
```

Browser support: Chrome 121+, Safari 16+, Edge 121+.

## Hybrid Transport (QR Code)

Works automatically — the browser handles cross-device auth via BLE + cloud relay. No server configuration needed.

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| "Origin mismatch" | RP ID doesn't match domain | Set rp_id to exact domain |
| "User verification failed" | UV=required but no biometric | Set UV=preferred |
| "Attestation format unsupported" | Unknown format | Check format is in 7 supported |
| "Credential excluded" | Already registered | Check excludeCredentials list |
| "Timeout" | User didn't interact in time | Increase timeout or re-prompt |
| signCount regression | Cloned credential | Revoke credential immediately |

## Production Checklist

- [ ] RP ID matches production domain exactly
- [ ] HTTPS enforced (WebAuthn requires secure context)
- [ ] Attestation conveyance configured per security policy
- [ ] User verification set appropriately
- [ ] Recovery codes generated and stored
- [ ] AAGUID allowlist configured (enterprise)
- [ ] Conditional UI tested on Chrome/Safari/Edge
- [ ] Fallback auth (password + TOTP) available
- [ ] Sign counter monitoring enabled

## See Also

- [WebAuthn Setup](webauthn-setup.md)
- [WebAuthn Deep Dive](../research/webauthn-deep-dive.md)
- [Passwordless Setup](passwordless-setup.md)
- [Authentication Flows](authentication-flows.md)
