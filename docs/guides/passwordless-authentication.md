# Passwordless Authentication

This guide covers WebAuthn platform authenticators, passkey sync, FIDO2 cross-device auth, passwordless migration strategy, UX design patterns, fallback methods, enterprise deployment, and GGID's passwordless implementation.

## Overview

Passwordless authentication eliminates passwords entirely, replacing them with cryptographic credentials (passkeys) stored on user devices. This reduces phishing, credential stuffing, and password fatigue while improving user experience.

## WebAuthn Platform Authenticators

### Built-in Biometric Authenticators

| Platform | Authenticator | Biometric |
|---|---|---|
| macOS | Touch ID / Face ID (Apple Silicon) | Fingerprint / Face |
| iOS | Face ID / Touch ID | Face / Fingerprint |
| Windows | Windows Hello | Face / Fingerprint / PIN |
| Android | Pixel Imprint / Face Unlock | Fingerprint / Face |
| Chrome OS | Chromebook PIN | PIN |

### How Platform Authenticators Work

```
1. Registration:
   - Server sends challenge
   - Browser prompts biometric (Face ID/Touch ID)
   - Platform creates key pair in secure enclave
   - Private key never leaves device
   - Public key + attestation sent to server

2. Authentication:
   - Server sends challenge
   - Browser prompts biometric
   - Platform signs challenge with private key
   - Signature sent to server
   - Server verifies with stored public key
```

### Platform vs Roaming

| Type | Example | Use Case |
|---|---|---|
| Platform | Touch ID, Windows Hello | Same device, convenient |
| Roaming | YubiKey, hardware token | Cross-device, high security |

## Passkey Sync

### What is Passkey Sync?

Passkeys can be synced across a user's devices via cloud keychain, enabling seamless cross-device authentication without each device needing its own credential.

### Sync Providers

| Provider | Platform | Storage | Multi-platform |
|---|---|---|---|
| Apple iCloud Keychain | iOS, macOS | Apple servers | Apple ecosystem only |
| Google Password Manager | Android, Chrome | Google servers | Android + Chrome |
| Microsoft | Windows | Microsoft account | Windows ecosystem |
| 1Password | Cross-platform | 1Password vault | Yes |
| Bitwarden | Cross-platform | Bitwarden vault | Yes |

### Sync Architecture

```
Device A (iPhone)
  └── Passkey synced via iCloud Keychain
      └── Device B (MacBook) receives same passkey
          └── Device C (iPad) receives same passkey
```

### Security Implications

| Aspect | Synced Passkey | Device-bound |
|---|---|---|
| Convenience | High (works on all devices) | Low (one device only) |
| Recovery | Automatic (new device gets passkey) | Manual (re-enroll needed) |
| Security | Good (encrypted in cloud) | Higher (key never leaves device) |
| Phishing resistance | Excellent | Excellent |
| Use case | Consumer | High-security enterprise |

## FIDO2 Cross-Device Authentication

### Hybrid Transport (CABLE)

FIDO2 cross-device authentication (formerly CA BLE, now "hybrid transport") allows a phone to authenticate for a desktop browser:

```
1. User opens login page on desktop browser
2. Desktop shows QR code
3. User scans QR code with phone
4. Phone authenticates with biometric
5. Phone sends signed assertion to desktop via BLE/internet
6. Desktop forwards assertion to server
7. Server verifies → user logged in on desktop
```

### Flow Detail

```
Desktop Browser          Phone                Server
     │                      │                    │
     │── request auth ──────────────────────────▶│
     │◀── challenge + QR ────────────────────────│
     │                      │                    │
     │── show QR ──▶ scan QR │                    │
     │                      │── biometric ──▶    │
     │                      │◀── sign challenge │
     │◀── assertion ────────│                    │
     │── assertion ──────────────────────────▶   │
     │◀── authenticated ─────────────────────────│
```

### Use Cases

| Scenario | Method |
|---|---|
| Desktop login with phone | QR code + hybrid transport |
| Public kiosk login | QR code + personal phone |
| New device setup | QR code + existing device |

## Passwordless Migration Strategy

### Phase 1: Add Passwordless (Coexist)

```yaml
passwordless:
  migration:
    phase: 1
    mode: "optional"
    allow_password: true
    allow_passkey: true
    encourage_passkey: true  # Show "try passkey" prompt
```

### Phase 2: Default Passwordless

```yaml
passwordless:
  migration:
    phase: 2
    mode: "preferred"
    allow_password: true  # Fallback
    default_method: "passkey"
    show_passkey_first: true
    password_link: "Use password instead"  # Small link
```

### Phase 3: Passwordless Required (New Users)

```yaml
passwordless:
  migration:
    phase: 3
    mode: "required_for_new"
    new_users: "passkey_only"  # No password for new accounts
    existing_users: "preferred"  # Still allow password
```

### Phase 4: Fully Passwordless

```yaml
passwordless:
  migration:
    phase: 4
    mode: "required"
    allow_password: false  # No passwords at all
    fallback: "recovery_key"  # Recovery via key, not password
```

## UX Design Patterns

### Registration UX

```
"Set up passkey"
┌─────────────────────────────────┐
│  Create your passkey            │
│                                 │
│  Use your device's biometric    │
│  to sign in without a password. │
│                                 │
│  [Create Passkey]               │
│                                 │
│  Maybe later                    │
└─────────────────────────────────┘
```

### Login UX

```
"Sign in with passkey"
┌─────────────────────────────────┐
│  Sign in to GGID                │
│                                 │
│  [🔒 Use passkey]               │
│                                 │
│  Use password instead →         │
└─────────────────────────────────┘
```

### Re-authentication UX

```
User already has passkey on this device:
→ Automatic biometric prompt (no clicks needed)
→ Touch ID / Face ID / Windows Hello appears
→ User authenticates in <1 second
```

### Cross-Device UX

```
Desktop browser (no passkey on this device):
┌─────────────────────────────────┐
│  Sign in with passkey            │
│                                 │
│  Scan QR code with your phone    │
│                                 │
│  ┌───────────┐                  │
│  │  QR Code  │                  │
│  │           │                  │
│  └───────────┘                  │
│                                 │
│  Or use password →              │
└─────────────────────────────────┘
```

## Fallback Methods

### When Passkey Is Unavailable

| Scenario | Fallback |
|---|---|
| New device (no passkey yet) | QR code + existing device |
| Phone lost | Recovery key / backup codes |
| Biometric failure | Device PIN |
| Browser doesn't support WebAuthn | TOTP / password |
| All passkeys lost | Admin reset + re-enrollment |

### Recovery Flow

```yaml
passwordless:
  recovery:
    methods:
      - recovery_key  # Long passphrase generated at enrollment
      - backup_codes  # One-time codes
      - admin_reset   # Admin-assisted
      - secondary_email  # Email verification
    require_identity_verification: true
    delay_period: 24h  # For high-security
```

## Enterprise Deployment

### Device Requirements

```yaml
passwordless:
  enterprise:
    device_requirements:
      managed: true  # Must be MDM-enrolled
      attestation: "packed"  # Require attestation
      trust_threshold: "basic"
    allow_personal_devices: true  # BYOD
    require_screen_lock: true
    require_disk_encryption: true
```

### Rollout Strategy

```yaml
passwordless:
  rollout:
    groups:
      - name: "pilot"
        members: ["security-team"]
        phase: 4  # Fully passwordless
      - name: "engineering"
        members: ["eng-department"]
        phase: 2  # Preferred
      - name: "all-users"
        members: ["*"]
        phase: 1  # Optional
    schedule:
      pilot: "2026-08-01"
      engineering: "2026-09-01"
      all-users: "2026-10-01"
```

## GGID Passwordless Implementation

### Configuration

```yaml
passwordless:
  enabled: true
  webauthn:
    rp_id: "ggid.example.com"
    rp_name: "GGID Identity Platform"
    origins:
      - "https://auth.ggid.example.com"
    user_verification: "required"
    timeout: 60000
    attestation: "none"  # or "direct" for enterprise
  passkey_sync:
    allow: true  # Allow synced passkeys
  cross_device:
    enabled: true
  migration:
    phase: 2
    allow_password: true
    default_method: "passkey"
  recovery:
    methods: ["recovery_key", "backup_codes", "admin_reset"]
    delay_period: 24h
```

### API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/auth/passkey/register/begin` | POST | Start passkey registration |
| `/api/v1/auth/passkey/register/finish` | POST | Complete registration |
| `/api/v1/auth/passkey/login/begin` | POST | Start passkey login |
| `/api/v1/auth/passkey/login/finish` | POST | Complete login |
| `/api/v1/auth/passkey/list` | GET | List user's passkeys |
| `/api/v1/auth/passkey/{id}` | DELETE | Remove a passkey |

### Registration Handler

```go
func (s *AuthService) BeginPasskeyRegistration(userID string) (*RegistrationOptions, error) {
    user := s.getUser(userID)
    
    options := &webauthn.PublicKeyCredentialCreationOptions{
        Challenge:        generateChallenge(32),
        RP: webauthn.RelyingParty{
            ID:   s.config.RPID,
            Name: s.config.RPName,
        },
        User: webauthn.User{
            ID:          []byte(user.ID),
            Name:        user.Email,
            DisplayName: user.Name,
        },
        PubKeyCredParams: []webauthn.PublicKeyCredentialParameters{
            {Type: "public-key", Alg: -7},   // ES256
            {Type: "public-key", Alg: -257}, // RS256
            {Type: "public-key", Alg: -8},   // EdDSA
        },
        AuthenticatorSelection: webauthn.AuthenticatorSelection{
            AuthenticatorAttachment: "platform",
            UserVerification:        "required",
            ResidentKey:             "required",
        },
        Timeout:         s.config.Timeout,
        Attestation:     s.config.Attestation,
        ExcludeCredentials: s.getExistingCredentials(user.ID),
    }
    
    // Store challenge for verification
    s.storeChallenge(userID, options.Challenge)
    
    return options, nil
}
```

## Best Practices

1. **Make passkey default** — Users prefer convenience, make it the primary option
2. **Allow coexistence** — Don't remove passwords until all users have passkeys
3. **Provide recovery** — Users will lose devices, have recovery flow ready
4. **Support cross-device** — QR code flow for new/unfamiliar devices
5. **Require UV** — User verification (biometric/PIN) on every authentication
6. **Allow synced passkeys for consumer** — Convenience > maximum security
7. **Require device-bound for enterprise** — Security > convenience
8. **Generate recovery key at enrollment** — User must save it before proceeding
9. **Track passkey enrollment rate** — Monitor migration progress
10. **Educate users** — Explain what passkeys are and why they're better