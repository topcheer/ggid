# Device-Bound SSO Analysis

> Analysis of Auth0 Device-Bound SSO and GGID's WebAuthn-based approach.

---

## What is Device-Bound SSO?

Device-Bound SSO ties a session to a specific device via a hardware-backed credential (passkey/security key). Even if a session token is stolen, it cannot be used from a different device.

---

## Auth0 Implementation

Auth0 announced Device-Bound SSO in 2025:
- Uses **WebAuthn platform authenticators** (Touch ID, Face ID, Windows Hello)
- SSO session is bound to the device's private key
- Cross-device SSO requires re-authentication on the new device
- Managed entirely in Auth0's hosted login

## GGID Implementation

GGID provides equivalent functionality via:

| Feature | Status | Location |
|---------|--------|----------|
| WebAuthn registration | Done | `services/auth/internal/webauthn/` |
| WebAuthn login flow | Done | `/api/v1/auth/webauthn/login/begin` + `/finish` |
| Device credential storage | Done | `webauthn_credentials` table |
| Per-device session binding | Partial | JWT `amr` claim marks WebAuthn-used sessions |
| Cross-device SSO block | TODO | No policy to require device for SSO |

### Gap: SSO Session Binding

GGID currently allows a WebAuthn-authenticated session to be used across devices if the JWT is copied. To match Auth0's Device-Bound SSO:

1. **Add `device_id` claim to JWT** — bind token to originating device
2. **Gateway verifies `device_id`** — reject if device fingerprint doesn't match
3. **Policy: require WebAuthn for SSO** — ABAC policy denying SSO without `user.mfa_verified: true`

---

## Recommendation

GGID has all the building blocks (WebAuthn, ABAC, JWT claims). Implementation effort:

| Task | Effort |
|------|--------|
| Add `device_id` to JWT claims | 1 day |
| Gateway device fingerprint check | 2 days |
| ABAC policy template "Require device-bound SSO" | 0.5 days |
| Console UI: "Require passkey for SSO" toggle | 1 day |
| **Total** | **~4.5 days** |

Priority: P1 (competitive feature, not blocking).

---

*See: [WebAuthn Setup](../guides/webauthn-setup.md) | [Security Overview](../architecture/security-overview.md) | [Gap Closure Report](gap-closure-report.md)*

*Last updated: 2025-07-11*
