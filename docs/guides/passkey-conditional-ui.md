# Passkey Conditional UI Guide

This guide covers implementing Passkey Conditional UI in GGID — the WebAuthn mediation API, autofill, hybrid transport, cross-device sync, browser support, and troubleshooting.

## What Is Conditional UI?

Conditional UI allows passkeys to appear in the browser's native autofill dropdown on login forms. Users select their passkey and authenticate with biometrics — no password entry.

```
User clicks username field
  ↓ Browser autofill dropdown shows saved passkeys
  ↓ User selects passkey
  ↓ Biometric prompt (Face ID / Touch ID / Windows Hello)
  ↓ GGID verifies WebAuthn assertion
  ↓ JWT issued — login complete
```

## Implementation

### Frontend — Enable Conditional Mediation

```javascript
// Check browser support
if (!window.PublicKeyCredential ||
    !PublicKeyCredential.isConditionalMediationAvailable()) {
  // Fallback to traditional login
  showPasswordForm();
}

const isConditional = await PublicKeyCredential.isConditionalMediationAvailable();

if (isConditional) {
  // Get assertion with conditional mediation
  const credential = await navigator.credentials.get({
    publicKey: {
      challenge: base64urlDecode(challenge),
      allowCredentials: [],  // Empty = use discovered credentials
      userVerification: 'required',
      mediation: 'conditional'  // Enable autofill
    }
  });

  // Send assertion to GGID for verification
  const response = await fetch('/api/v1/webauthn/auth/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      assertion: base64urlEncode(credential.response.authenticatorData),
      client_data_json: base64urlEncode(credential.response.clientDataJSON),
      signature: base64urlEncode(credential.response.signature)
    })
  });

  const { access_token } = await response.json();
}
```

### HTML Form

```html
<!-- The autocomplete attribute triggers passkey autofill -->
<input
  type="text"
  id="username"
  autocomplete="username webauthn"
  placeholder="Username"
/>
```

The `webauthn` autocomplete token tells the browser to show passkeys in the autofill dropdown.

## Hybrid Transport (Cross-Device)

Hybrid transport lets users authenticate with a passkey stored on their phone while logging in on a desktop:

```
Desktop browser shows QR code
  ↓ Phone scans QR code
  ↓ Phone authenticates user (Face ID / fingerprint)
  ↓ Bluetooth LE confirms physical proximity
  ↓ Cloud relay passes assertion to desktop
  ↓ Desktop receives WebAuthn assertion
  ↓ GGID verifies — login complete
```

**No server configuration needed** — the browser handles cross-device auth automatically.

### Enabling in GGID

```yaml
webauthn:
  hybrid_transport: true  # Enable cross-device auth
```

## Browser Support Matrix

| Browser | Conditional UI | Hybrid Transport | Platform Auth | Cross-Platform |
|---------|---------------|-----------------|-------------|----------------|
| Chrome 121+ | Yes | Yes | Yes | Yes |
| Edge 121+ | Yes | Yes | Yes | Yes |
| Safari 16+ | Yes | Yes | Yes | Yes |
| Firefox | No | No | Yes | Yes |
| Chrome Android | No | Yes (as phone) | Yes | No |
| Safari iOS | Yes | Yes (as phone) | Yes | No |

### Feature Detection

```javascript
const features = {
  conditionalUI: await PublicKeyCredential.isConditionalMediationAvailable(),
  hybrid: 'Bluetooth' in navigator,  // Approximate
  webauthn: window.PublicKeyCredential !== undefined
};
```

## Cross-Device Sync

Passkeys sync across devices via platform sync:

| Platform | Sync Service |
|----------|-------------|
| Apple | iCloud Keychain |
| Google | Google Password Manager |
| Microsoft | Windows Hello (coming) |

Synced passkeys work on any device signed into the same account.

## Server-Side: Discoverable Credentials

Conditional UI works best with **discoverable credentials** (resident keys). GGID stores credential IDs server-side:

```bash
# Begin auth (server returns challenge, no credential list)
curl -X POST https://api.ggid.example.com/api/v1/webauthn/auth/begin \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":""}'  # Empty = discoverable
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| No passkey in autofill | `autocomplete` missing | Add `webauthn` token |
| "Not supported" error | Firefox or old browser | Feature-detect, fallback to password |
| QR code doesn't appear | Bluetooth disabled | Enable Bluetooth on both devices |
| Cross-device fails | Different Apple/Google accounts | Ensure same account on both |
| UV failed | Biometric unavailable | Set `userVerification: 'preferred'` |
| Timeout | User didn't interact in 60s | Increase timeout or re-prompt |

## Best Practices

- [ ] Always provide password fallback (not all browsers support conditional UI)
- [ ] Use `autocomplete="username webauthn"` on username fields
- [ ] Feature-detect before calling conditional mediation API
- [ ] Set `userVerification: 'preferred'` (not 'required') for broader compatibility
- [ ] Test on Chrome, Safari, Edge, Firefox
- [ ] Educate users about passkey enrollment
- [ ] Track passkey enrollment rate as a metric

## See Also

- [WebAuthn Deploy Guide](webauthn-deploy.md)
- [WebAuthn Deep Dive](../research/webauthn-deep-dive.md)
- [Authentication Flows](authentication-flows.md)
- [Passwordless Setup](passwordless-setup.md)
