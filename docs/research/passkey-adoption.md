# Passkey Adoption Analysis

> Passkey ecosystem analysis and GGID's passkey-first authentication strategy.

---

## Passkey Ecosystem (2025)

### Platform Support

| Platform | Sync | Cross-Device | Status |
|----------|------|-------------|--------|
| Apple (iOS/macOS) | iCloud Keychain | Yes (QR) | GA since iOS 16 |
| Google (Android/Chrome) | Google Password Manager | Yes (QR) | GA since Android 9 |
| Microsoft (Windows) | Windows Hello | Planned | GA since Win 10 |
| 1Password | Cloud sync | Yes | GA |
| Bitwarden | Cloud sync | Yes | GA |

### Adoption Stats

- Apple: 60%+ of iOS users have passkey-capable devices
- Google: 400M+ passkey enrollments (2025)
- WebAuthn: W3C standard since 2019, FIDO2 certified
- Major sites: Google, Apple, Microsoft, GitHub, TikTok, Amazon

---

## GGID Passkey Strategy

### Current State

| Feature | Status |
|---------|--------|
| WebAuthn registration | Done |
| WebAuthn login | Done |
| Multiple credentials per user | Done |
| Device management (list/delete) | Done |
| Passkey-first policy | Via ABAC |
| Cross-device auth (hybrid) | Browser-native |

### Recommended: Passkey-First Auth Flow

```
1. User enters username
2. GGID checks if user has registered passkey
3. If yes → WebAuthn login (no password prompt)
4. If no → Password login + offer passkey registration
5. After 2 successful passkey logins → remove password option
```

### Implementation

```go
func (s *AuthService) Login(ctx context.Context, username string) (*LoginOptions, error) {
    user, err := s.repo.GetByUsername(ctx, username)
    if err != nil {
        return &LoginOptions{Methods: []string{"password"}}, nil
    }

    creds, _ := s.webauthn.ListCredentials(ctx, user.ID)
    if len(creds) > 0 {
        return &LoginOptions{
            Methods:         []string{"webauthn"},
            PasswordAllowed: false, // Passkey-first
        }, nil
    }
    return &LoginOptions{Methods: []string{"password"}}, nil
}
```

---

## Competitive Position

| Feature | GGID | Auth0 | Keycloak | Clerk |
|--------|------|-------|----------|-------|
| WebAuthn | Yes | Yes | Yes | Yes |
| Passkey-first flow | Via ABAC | Yes | No | Yes |
| Device management | Yes | Yes | Limited | Yes |
| Biometric | Platform-native | Platform-native | Platform-native | Platform-native |

---

## Recommendation

1. **Add `passkey_first` tenant setting** — auto-detect passkey users, skip password
2. **Console UI toggle** — "Require passkey for all users"
3. **Promote passkey at registration** — default option, password as fallback
4. **Track passkey adoption** — audit event `webauthn.register` vs `user.register`

Effort: ~3 days for passkey-first flow + Console UI.

---

*See: [WebAuthn Setup](../guides/webauthn-setup.md) | [Passwordless Setup](../guides/passwordless-setup.md) | [Device-Bound SSO Analysis](device-bound-sso-analysis.md)*

*Last updated: 2025-07-11*
