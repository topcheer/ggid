# Device-Bound SSO Design

**Date**: 2025-07-11
**Status**: Architecture design — not yet implemented

## Overview

Device-bound SSO ties authentication sessions to a specific device via hardware-backed cryptographic keys (WebAuthn/TPM/Secure Enclave). Unlike traditional SSO where any device with the session cookie can authenticate, device-bound SSO requires proof of possession of a device-bound key.

## Auth0 Approach

Auth0's device-bound SSO uses:
- **Private key stored in device hardware** (TPM 2.0, Secure Enclave, Android Keystore)
- **Certificate binding** to the device's hardware attestation
- **Session token contains device assertion** — server verifies on each request

## GGID Implementation Plan

### Phase 1: WebAuthn Device Registration (partially exists)

GGID already has WebAuthn registration/authentication in `services/auth/internal/webauthn/`. Extend it:

1. **Device registration during login**: After password auth, prompt for WebAuthn device binding
2. **Device ID in JWT claims**: Add `device_id` claim to access tokens
3. **Session validation**: Verify device credential on token refresh

### Phase 2: SSO Flow with Device Binding

```
User → IdP (password) → WebAuthn challenge → Device-bound token
  ↓                                           ↓
  Service A ← SSO token + device assertion ← Token has device_id
  Service B ← Same SSO token + device assertion ← Verifies device_id matches
```

### API Changes

- `POST /api/v1/auth/device-bind` — Register device after login
- `POST /api/v1/auth/device-verify` — Verify device on SSO propagation
- JWT claim: `"device_id": "webauthn-credential-id"`
- Refresh token: requires WebAuthn assertion

### Security Properties

- **Session hijacking prevention**: Stolen token can't be used on different device
- **Cross-device SSO**: Each device must independently authenticate with WebAuthn
- **Revocation**: Device can be revoked → all device-bound sessions invalidated

### Comparison

| Feature | Auth0 | GGID (planned) | GGID (current) |
|---|---|---|---|
| Hardware key binding | Yes | Phase 1 | WebAuthn exists, not bound to SSO |
| Device-bound session | Yes | Phase 2 | No |
| Token refresh with device proof | Yes | Phase 2 | No |
| Cross-device SSO blocking | Yes | Phase 2 | No |

## Recommendation

This is a P1 item for competitive parity. GGID's WebAuthn infrastructure provides 70% of the needed code. The missing piece is binding WebAuthn credential IDs to JWT sessions and requiring assertion on token refresh.
