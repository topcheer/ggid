# FIDO2/Passkey Ecosystem 2026

> Passkey adoption landscape and GGID WebAuthn positioning.

---

## Platform Passkey Support (2026)

| Platform | Sync Method | Cross-Device Auth | Users |
|----------|------------|------------------|-------|
| Apple (iOS 17+/macOS 14+) | iCloud Keychain | QR + Bluetooth | 1.2B+ devices |
| Google (Android 9+/Chrome) | Google Password Manager | QR + Bluetooth | 3B+ devices |
| Microsoft (Windows 11) | Windows Hello | Planned | 200M+ |
| 1Password | Cloud vault | Yes | 5M+ |
| Bitwarden | Cloud vault | Yes | 3M+ |

### Passkey Sync Benefits
- Device loss recovery (synced to new device)
- Cross-platform (Apple↔Google via QR)
- No password to phish

---

## Cross-Device Authentication (Hybrid Transport)

```
1. User on laptop visits login page
2. Laptop shows QR code
3. User scans QR with phone
4. Phone uses Face ID/Touch ID
5. Phone signs challenge via Bluetooth
6. Laptop receives authenticated session
```

This is **browser-native** — GGID doesn't need custom implementation. The browser handles FIDO2 hybrid transport automatically.

---

## GGID WebAuthn Implementation

### Current Capabilities

| Feature | Status |
|---------|--------|
| Registration (platform + roaming) | Done |
| Authentication (platform + roaming) | Done |
| Multiple credentials per user | Done |
| Credential management (list/delete) | Done |
| Attestation verification | Partial (5 of 6 formats) |
| Backup eligibility flags | Done |

### vs Competitors

| Feature | GGID | Auth0 | Keycloak | Clerk |
|--------|------|-------|----------|-------|
| WebAuthn registration | Yes | Yes | Yes | Yes |
| Passkey-first flow | Via ABAC | Native | No | Native |
| Cross-device auth | Browser-native | Browser-native | Browser-native | Browser-native |
| Credential labels | Yes | Yes | No | Yes |
| Attestation formats | 5/6 | 6/6 | 4/6 | 6/6 |

---

## Passkey Adoption Strategy for GGID

1. **Default to passkey at registration** — show passkey as primary, password as fallback
2. **Passkey-first login** — if user has passkey, skip password entirely
3. **Promote upgrade** — after successful password login, offer passkey registration
4. **Track adoption** — audit events `webauthn.register` vs `user.register`
5. **Complete attestation** — implement Apple anonymized attestation format

---

## FIDO Metadata Service

FIDO MDS provides attestation root certificates for verifying authenticator authenticity. GGID should cache MDS blob and validate attestation certificates against it.

Effort: 2 days (MDS fetch + cache + validation).

---

*See: [Passkey Adoption](passkey-adoption.md) | [WebAuthn Setup](../guides/webauthn-setup.md) | [Device-Bound SSO Analysis](device-bound-sso-analysis.md)*

*Last updated: 2025-07-11*
