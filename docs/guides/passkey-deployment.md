# Passkey Deployment

Passkey vs traditional WebAuthn, platform authenticator sync, enrollment UX, conditional UI, multi-device sync, recovery, migration strategy, FIDO stats, and implementation checklist.

## Passkey vs Traditional WebAuthn

| Aspect | Traditional WebAuthn | Passkey |
|--------|---------------------|---------|
| Credential sync | No (device-bound) | Yes (cloud sync) |
| Multi-device | Re-register per device | Automatic via sync |
| Recovery | Manual (backup factor) | Cloud account recovery |
| UX | USB tap / biometric per device | One-tap enroll, autofill login |
| Backing | Hardware key | Apple iCloud / Google / Microsoft |

Passkeys ARE WebAuthn credentials with platform-managed sync.

## Platform Authenticator Sync

| Platform | Sync Service | Devices |
|----------|-------------|---------|
| Apple | iCloud Keychain | iPhone, iPad, Mac (same Apple ID) |
| Google | Google Password Manager | Android, Chrome OS (same Google account) |
| Microsoft | Windows Hello | Windows devices (same MS account) |

### Sync Benefits

- User registers passkey on iPhone → available on iPad + Mac
- Device lost → passkey recovered via cloud account
- No re-registration needed across devices

## Enrollment UX (One-Tap)

```javascript
// Simplified enrollment — minimal friction
async function enrollPasskey() {
  const options = await fetch('/webauthn/register/begin').then(r => r.json());
  
  // Browser handles Face ID / Touch ID / Windows Hello
  const credential = await navigator.credentials.create({ publicKey: options });
  
  // Send to server — done!
  await fetch('/webauthn/register/complete', {
    method: 'POST',
    body: JSON.stringify(credential)
  });
  
  showSuccess("Passkey created! It syncs to your devices automatically.");
}
```

### UX Flow

```
1. User clicks "Set up passkey" button
2. Browser shows native biometric prompt (Face ID / fingerprint)
3. User authenticates with biometric
4. Passkey created + synced to cloud
5. "Success! You can now sign in with biometrics."
Total time: ~3 seconds
```

## Conditional UI Autofill

```html
<input type="text" autocomplete="username webauthn">
<input type="password" autocomplete="current-password webauthn">
```

```javascript
// Conditional UI — non-blocking, shows passkey in autofill
navigator.credentials.get({
  mediation: "conditional",
  publicKey: { challenge: ..., userVerification: "required" }
});
```

User focuses username field → passkey option appears in autofill dropdown → user selects → biometric → logged in.

## Multi-Device Sync

### Within Ecosystem (Automatic)

```
iPhone: Register passkey → iCloud Keychain syncs
iPad:   Passkey appears automatically → can authenticate
Mac:    Same passkey → Touch ID or Apple Watch
```

### Cross-Ecosystem (Hybrid Transport)

```
Android phone + Mac laptop:
1. Mac shows QR code
2. Android scans QR
3. Android authenticates with fingerprint
4. Assertion sent to Mac via BLE/Wi-Fi
5. Mac completes authentication
```

## Recovery

| Method | How | When |
|--------|-----|------|
| Cloud sync | Restore via Apple/Google account | New device, same ecosystem |
| Device-to-device | QR scan from existing device | New device, same user |
| Admin-assisted | Identity verification + re-enroll | Lost all devices |
| Backup factor | TOTP or recovery codes | Passkey unavailable temporarily |

## Migration Strategy

```
Phase 1: Add passkey as MFA option (alongside TOTP)
  ↓ Adoption ~20%
Phase 2: Promote passkey as primary, password as backup
  ↓ Adoption ~50%
Phase 3: Passkey-only login (password optional)
  ↓ Adoption ~80%
Phase 4: Passwordless default (password deprecated)
  ↓ Adoption ~95%
Phase 5: Password removed entirely
```

### Per-User Migration

```
Password only → Add TOTP → Add passkey → Remove password (opt-in)
```

## FIDO Alliance Stats

| Metric | Value | Source |
|--------|-------|--------|
| Passkey success rate | 4x higher than passwords | FIDO 2024 |
| Sign-in time | 0.3s (passkey) vs 12s (password+MFA) | FIDO |
| Account takeover reduction | 95% with passkeys | Google |
| Consumer awareness | 78% know "passkey" | FIDO 2024 survey |
| Passkey support | Apple (2022), Google (2023), MS (2023) | Platform |

## Implementation Checklist

- [ ] WebAuthn server implementation (registration + auth ceremonies)
- [ ] RP ID configured (e.g., `ggid.dev`)
- [ ] `userVerification: "preferred"` (biometric when available)
- [ ] `residentKey: "preferred"` (discoverable credentials)
- [ ] Conditional UI enabled (autofill with `mediation: conditional`)
- [ ] Multiple passkeys allowed per user (phone + YubiKey)
- [ ] Backup factor (TOTP or recovery codes) for passkey loss
- [ ] Track `backup_eligible` and `backed_up` flags
- [ ] Passkey management UI (list, rename, delete)
- [ ] Migration UX (password → passkey prompt)
- [ ] Fallback login (password) during transition
- [ ] Audit logging for all passkey operations

## Monitoring

| Metric | Target |
|--------|--------|
| Passkey enrollment rate | >50% of active users |
| Passkey login success rate | >95% |
| Average passkey login time | <2s |
| Recovery rate (passkey lost) | <5%/month |
| Conditional UI usage | Track browser support |

## See Also

- [WebAuthn Server Implementation](webauthn-server-implementation.md)
- [WebAuthn Deployment Guide](webauthn-deployment-guide.md)
- [Multi-Factor Auth Strategy](multi-factor-auth-strategy.md)
- [Passkey Recovery Strategy](passkey-recovery-strategy.md)
- [Passwordless Auth Architecture](passwordless-auth-architecture.md)