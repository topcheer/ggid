# WebAuthn Deployment Guide

Registration/authentication flows, transport types, conditional UI, resident vs non-resident keys, RP ID config, timeout, user verification policy, backup eligibility, and multi-device sync.

## Registration Flow

```
1. POST /webauthn/register/begin → challenge + options
2. navigator.credentials.create(options) → credential
3. POST /webauthn/register/complete → verify + store
```

### Server Options

```json
{
  "challenge": "base64url-random",
  "rp": {"name": "GGID", "id": "ggid.dev"},
  "user": {"id": "base64url", "name": "jane@corp.com", "displayName": "Jane Doe"},
  "pubKeyCredParams": [
    {"type": "public-key", "alg": -7},
    {"type": "public-key", "alg": -257}
  ],
  "timeout": 60000,
  "attestation": "none",
  "excludeCredentials": [{"type": "public-key", "id": "..."}],
  "authenticatorSelection": {
    "authenticatorAttachment": "platform",
    "userVerification": "required",
    "residentKey": "preferred"
  }
}
```

## Authentication Flow

```
1. POST /webauthn/auth/begin → challenge + allowCredentials
2. navigator.credentials.get(options) → assertion
3. POST /webauthn/auth/complete → verify + issue session
```

## Transport Types

| Transport | Description | Example |
|-----------|------------|---------|
| `internal` | Platform authenticator (built-in) | Touch ID, Face ID |
| `hybrid` | Cross-device via QR | Phone scans laptop |
| `cross-platform` | External device | YubiKey, security key |
| `usb` | USB-connected | YubiKey USB |
| `nfc` | NFC tap | YubiKey NFC |
| `ble` | Bluetooth | (deprecated) |

## Conditional UI (Autofill)

```javascript
// Browser shows WebAuthn credential in autofill dropdown
const assertion = await navigator.credentials.get({
  mediation: "conditional",  // Non-blocking, shows in autofill
  publicKey: {
    challenge: decode(challenge),
    userVerification: "required",
    // No allowCredentials → discoverable credentials
  }
});
```

User types username → WebAuthn credential appears in autofill → user selects → Face ID/Fingerprint.

## Resident vs Non-Resident Keys

| Type | Stored Where | RP ID Needed | Privacy |
|------|-------------|-------------|---------|
| Non-resident (server-side) | Server stores credential ID | Yes (send allowCredentials) | High (device has no user info) |
| Resident (discoverable) | Authenticator stores credential + RP ID | No (discoverable) | Medium (device knows user) |

GGID defaults to `residentKey: "preferred"` — allows discoverable credentials when authenticator supports.

## RP ID Configuration

```
RP ID = ggid.dev (effective domain)
→ Works on: auth.ggid.dev, app.ggid.dev, console.ggid.dev
→ Does NOT work on: ggid.com, other-domain.com
```

RP ID must be a registrable domain suffix of the origin. All subdomains share the same RP ID.

## Timeout Settings

| Flow | Timeout | Rationale |
|------|---------|-----------|
| Registration | 60s | User needs time to interact with device |
| Authentication | 30s | Should be quick if device is available |
| Conditional UI | No timeout | Passive — waits for user to select |

## User Verification Policy

| Policy | Behavior | Use Case |
|--------|----------|---------|
| `required` | Must verify (biometric/PIN) | High-security, admin |
| `preferred` | Verify if supported | Default |
| `discouraged` | Don't verify (user presence only) | Low-risk read-only |

## Backup Eligibility

Modern authenticators (Apple iCloud Keychain, Google Password Manager) sync passkeys across devices:

```json
{
  "authenticatorAttachment": "platform",
  "backed_up": true,  // Credential is synced/backed up
  "backup_eligible": true
}
```

### Backup State Handling

| State | Meaning | Action |
|-------|---------|--------|
| `backup_eligible: true` | Authenticator supports backup | Log |
| `backed_up: true` | Credential is actually backed up | Lower risk of lockout |
| `backed_up: false` | Not backed up | Warn user about device loss risk |

## Multi-Device Sync

```
Device 1: Register passkey (iPhone) → synced to iCloud Keychain
Device 2: Passkey available on iPad (via sync) → can authenticate
Device 3: Mac → same passkey via iCloud → can authenticate
```

### Hybrid Transport (Cross-Device)

```
Laptop shows QR code → Phone scans → Phone authenticates →
Assertion sent to laptop via BLE/Wi-Fi → Laptop completes auth
```

## Best Practices

1. **Offer WebAuthn as primary** — not just MFA add-on
2. **Allow multiple credentials** — user can register phone + YubiKey
3. **Don't require attestation** — `"none"` for privacy
4. **Provide recovery path** — backup factors + admin-assisted
5. **Track backup state** — warn users with non-backed-up credentials
6. **Support conditional UI** — smoothest UX for returning users

## Monitoring

| Metric | Alert |
|--------|-------|
| Registration completion rate | <80% → UX issue |
| Authentication success rate | <95% → device or config issue |
| Non-backed-up credential count | Track → prompt for backup |
| Conditional UI adoption | Track browser support |

## See Also

- [WebAuthn Server Implementation](webauthn-server-implementation.md)
- [Multi-Factor Auth Strategy](multi-factor-auth-strategy.md)
- [Passkey Recovery Strategy](passkey-recovery-strategy.md)
- [Passwordless Auth Architecture](passwordless-auth-architecture.md)
