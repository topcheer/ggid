# Passkey Sync Security Analysis for IAM Systems

> **Document Type:** Security Research
> **Project:** GGID — Go-based Identity and Access Management Suite
> **Author:** Security Research Team
> **Date:** 2025-01-11
> **Status:** Complete
> **Classification:** Internal / Open Source (Apache 2.0)

---

## Table of Contents

1. [Passkey Sync Architecture](#1-passkey-sync-architecture)
2. [Security Analysis of Synced Passkeys](#2-security-analysis-of-synced-passkeys)
3. [Device-to-Device Transfer](#3-device-to-device-transfer)
4. [Non-Synced (Device-Bound) Passkeys](#4-non-synced-device-bound-passkeys)
5. [Account Recovery Implications](#5-account-recovery-implications)
6. [Passwordless Sync UX](#6-passwordless-sync-ux)
7. [WebAuthn Transport Hints](#7-webauthn-transport-hints)
8. [Multi-Device Authenticator Management](#8-multi-device-authenticator-management)
9. [GGID WebAuthn Implementation Review](#9-ggid-webauthn-implementation-review)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)
11. [References](#11-references)

---

## 1. Passkey Sync Architecture

### 1.1 Overview

Passkey sync is the process by which a WebAuthn credential's private key material is
replicated across multiple devices owned by the same user. This enables a user to register
a passkey on one device (e.g., an iPhone) and use it to authenticate on another device
(e.g., a MacBook or iPad) without re-registration.

The FIDO Alliance originally promoted WebAuthn credentials as **device-bound** — the
private key never left the authenticator's secure hardware. This provided the highest
security guarantee but created a significant UX problem: users had to register a separate
credential on every device, and losing a device meant losing access.

To address this, platform vendors introduced **synced passkeys** (sometimes called
"multi-device credentials"). The key material is synchronized through a cloud-based
keychain, encrypted end-to-end so that the vendor cannot access the keys. The FIDO
Alliance formalized this capability through the `Backup Eligible` (BE) and `Backup State`
(BS) flags in the authenticator data.

```
┌──────────────────────────────────────────────────────────────────────┐
│                    PASSKEY SYNC ECOSYSTEM                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│   │   Apple     │  │   Google    │  │ Microsoft   │                 │
│   │  iCloud     │  │  Password   │  │ Authenticator│                 │
│   │  Keychain   │  │  Manager    │  │  (Sync)     │                 │
│   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │
│          │                │                │                         │
│          ▼                ▼                ▼                         │
│   ┌──────────────────────────────────────────────┐                  │
│   │        Encrypted Key Sync Transport           │                  │
│   │   (E2E encrypted, vendor cannot decrypt)      │                  │
│   └──────────────────────┬───────────────────────┘                  │
│                          │                                           │
│          ┌───────────────┼───────────────┐                          │
│          ▼               ▼               ▼                           │
│   ┌────────────┐  ┌────────────┐  ┌────────────┐                   │
│   │  iPhone    │  │  MacBook   │  │   iPad     │                   │
│   │ (Secure    │  │ (Secure    │  │ (Secure    │                   │
│   │  Enclave)  │  │  Enclave)  │  │  Enclave)  │                   │
│   └────────────┘  └────────────┘  └────────────┘                   │
│                                                                      │
│   The RP (GGID) sees the SAME credential ID on all devices.          │
│   The private key never appears in plaintext outside secure HW.      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 1.2 Apple iCloud Keychain Sync

Apple's passkey sync uses **iCloud Keychain**, a service that synchronizes passwords,
passkeys, and other keychain items across a user's Apple devices. The security model is
documented in Apple's Platform Security Guide.

**Key Properties:**

- **End-to-end encryption:** Key material is encrypted with keys derived from the user's
  device passcode and hardware-bound keys. Apple's servers store only ciphertext.
- **Secure Enclave (SEP):** On devices with an SEP (iPhone 5s+, MacBook with T1/T2/M
  chips), the private key never exists in plaintext in the main CPU's memory. All
  cryptographic operations occur inside the SEP.
- **Account Authentication:** iCloud Keychain requires two-factor authentication (2FA)
  for the Apple ID. Initial sync setup requires device approval from an already-trusted
  device.
- **Custodial Recovery:** Apple provides an account recovery flow for users who lose
  access to all trusted devices, but this requires identity verification through trusted
  phone numbers and a waiting period (typically 24-72 hours).

```
┌──────────────────────────────────────────────────────────────────────┐
│                APPLE iCLOUD KEYCHAIN SYNC FLOW                       │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  iPhone (Registration)                                               │
│  ┌───────────────────────────────────┐                              │
│  │ 1. WebAuthn registration          │                              │
│  │ 2. SEP generates key pair         │                              │
│  │ 3. Private key sealed in SEP      │                              │
│  │ 4. BE=1, BS=0 flags set           │                              │
│  │ 5. Attestation sent to RP         │                              │
│  └───────────────┬───────────────────┘                              │
│                  │                                                    │
│                  ▼                                                    │
│  iCloud Keychain Sync                                                │
│  ┌───────────────────────────────────┐                              │
│  │ 6. Private key encrypted with      │                              │
│  │    device-derived key              │                              │
│  │ 7. Ciphertext uploaded to iCloud   │                              │
│  │ 8. BS flag flips to 1 (synced)     │                              │
│  └───────────────┬───────────────────┘                              │
│                  │                                                    │
│          ┌───────┴───────┐                                           │
│          ▼               ▼                                           │
│  ┌──────────────┐  ┌──────────────┐                                 │
│  │   MacBook    │  │    iPad      │                                 │
│  │ 9. iCloud    │  │ 9. iCloud    │                                 │
│  │    sync pulls│  │    sync pulls│                                 │
│  │    ciphertext│  │    ciphertext│                                 │
│  │ 10. SEP      │  │ 10. SEP      │                                 │
│  │    imports   │  │    imports   │                                 │
│  │    key       │  │    key       │                                 │
│  │ 11. Auth OK  │  │ 11. Auth OK  │                                 │
│  └──────────────┘  └──────────────┘                                 │
│                                                                      │
│  RP sees: Same credential ID, same public key from all 3 devices.   │
│  Sign count advances monotonically regardless of which device       │
│  authenticates (SEP coordination via iCloud Keychain).               │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

**Sync Timing:** iCloud Keychain sync is not instantaneous. It typically completes within
seconds to minutes for active devices, but can take longer if devices are asleep or
offline. The `Backup State (BS)` flag is set to `true` once sync completes, signaling
to the RP that the credential now exists on multiple devices.

### 1.3 Google Password Manager Sync

Google's passkey sync uses **Google Password Manager**, integrated with the Google
Account infrastructure. On Android devices and Chrome, passkeys are synced through the
Google Account.

**Key Properties:**

- **Titan M/M2 Security Chip:** Google Pixel devices use Titan security chips to protect
  key material in hardware, similar to Apple's Secure Enclave.
- **Google Account Sync:** Passkeys are synced to the user's Google Account and pushed
  to other Android devices signed into the same account.
- **End-to-End Encryption:** Key material is encrypted before upload. Google uses the
  user's screen lock credential (PIN, pattern, password) as part of the encryption key
  derivation on supported devices.
- **Cross-Platform via QR:** Google supports **hybrid transport** — a phone can serve as
  an authenticator for a desktop browser via a QR code + Bluetooth proximity check. This
  is distinct from sync: the phone authenticates directly; the key does not leave the
  phone.

```
┌──────────────────────────────────────────────────────────────────────┐
│           GOOGLE PASSWORD MANAGER SYNC ARCHITECTURE                  │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────┐                                                   │
│  │  Pixel Phone  │                                                   │
│  │  ┌─────────┐  │                                                   │
│  │  │ Titan M │  │  ← Private key sealed in Titan chip               │
│  │  │  Chip   │  │                                                   │
│  │  └─────────┘  │                                                   │
│  └───────┬───────┘                                                   │
│          │ encrypted key                                             │
│          ▼                                                            │
│  ┌───────────────┐     ┌───────────────┐                            │
│  │ Google Account│────→│ Google Server │                            │
│  │  Sync Service │     │  (ciphertext) │                            │
│  └───────┬───────┘     └───────────────┘                            │
│          │                                                            │
│          ▼                                                            │
│  ┌───────────────┐                                                   │
│  │  Tablet/      │                                                   │
│  │  Chromebook   │  ← Key decrypted with Titan/TPM                   │
│  │  (TPM/Titan)  │                                                   │
│  └───────────────┘                                                   │
│                                                                      │
│  Note: Chrome on desktop can also USE synced passkeys from the      │
│  Google Account, but the private key operations happen in the       │
│  browser's software authenticator (not hardware-isolated unless     │
│  using the phone-as-authenticator hybrid flow).                      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

**Important Distinction — Sync vs. Hybrid:**

- **Sync:** Key material exists on multiple devices permanently. Each device can
  independently authenticate.
- **Hybrid (caBLE):** One device (phone) acts as the authenticator for another device
  (desktop). The key never leaves the phone. Bluetooth proximity + QR code establishes
  the connection.

Google supports both patterns. Apple sync uses true key replication. The security
implications differ significantly, as discussed in Section 2.

### 1.4 Microsoft Authenticator Sync

Microsoft's passkey sync is integrated through **Microsoft Entra ID (formerly Azure AD)**
for enterprise scenarios and **Windows Hello** for consumer scenarios.

**Key Properties:**

- **Windows Hello:** Windows 10/11 provides a platform authenticator backed by TPM 2.0.
  Passkeys created with Windows Hello are stored in the TPM and are **not synced** by
  default — they are device-bound.
- **Microsoft Authenticator App:** The Microsoft Authenticator app (iOS/Android) can
  sync passkeys across devices through the user's Microsoft Account. This is separate
  from Windows Hello's TPM-backed credentials.
- **Entra ID Passkeys:** Microsoft Entra ID supports both synced and device-bound
  passkey policies, allowing administrators to enforce device-bound credentials for
  high-security scenarios.

```
┌──────────────────────────────────────────────────────────────────────┐
│         MICROSOFT PASSKEY SYNC — DUAL MODEL                          │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Model A: Windows Hello (Device-Bound)                               │
│  ┌───────────────────────────────────┐                              │
│  │ Windows 11 PC                     │                              │
│  │  ┌─────────┐                      │                              │
│  │  │ TPM 2.0 │ ← Key NEVER syncs   │                              │
│  │  └─────────┘   BE=0, BS=0         │                              │
│  └───────────────────────────────────┘                              │
│                                                                      │
│  Model B: MS Authenticator (Synced)                                  │
│  ┌────────────┐         ┌────────────────┐         ┌────────────┐   │
│  │  iPhone    │←───────→│ MS Account     │───────→│ Android    │   │
│  │ (Auth app) │  sync   │ (cloud relay)  │  sync  │ (Auth app) │   │
│  └────────────┘         └────────────────┘         └────────────┘   │
│     BE=1, BS=1               BE=1, BS=1               BE=1, BS=1   │
│                                                                      │
│  Enterprise Policy: Entra ID admin can REQUIRE device-bound         │
│  (Model A) by rejecting BE=1 credentials at registration time.      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 1.5 Multi-Device Convenience vs Device-Bound Security Tradeoff

The fundamental tension in passkey deployment is:

```
┌──────────────────────────────────────────────────────────────────────┐
│              SYNCED vs DEVICE-BOUND TRADEOFF MATRIX                  │
├───────────────┬──────────────────┬───────────────────────────────────┤
│   Dimension   │   Synced (BE=1)  │    Device-Bound (BE=0)           │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ Convenience   │ HIGH — register  │ LOW — register on every          │
│               │ once, use on all │ device individually              │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ Recovery      │ GOOD — other     │ POOR — if device lost,           │
│               │ devices have key │ credential is gone               │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ Phishing      │ YES — origin     │ YES — origin binding             │
│ Resistance    │ binding intact   │ intact                           │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ Key Exposure  │ MULTIPLE devices │ SINGLE device                    │
│ Surface       │ can be attacked  │ is the only target               │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ Clone Risk    │ Synced keys use  │ Counter-based clone               │
│               │ always-increment │ detection effective               │
│               │ counter; clone   │                                   │
│               │ detection reduced│                                   │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ Device Comp.  │ ANY synced device │ Only the specific               │
│ Impact        │ exposes the key  │ authenticator device             │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ Enterprise    │ Often REJECTED   │ Often REQUIRED for               │
│ Acceptance    │ for high-security │ high-security tenants            │
│               │ tenants          │                                   │
├───────────────┼──────────────────┼───────────────────────────────────┤
│ AAL2/AAL3    │ AAL2 acceptable  │ AAL3 preferred                   │
│ (NIST 800-63) │ (hardware-backed)│ (single hardware factor)         │
└───────────────┴──────────────────┴───────────────────────────────────┘
```

**NIST SP 800-63B Rev. 3 (2024 update) positions:**

Synced passkeys from Apple/Google/Microsoft are eligible for **AAL2** when:
- The platform authenticator is backed by hardware (SEP, Titan, TPM)
- User verification (UV) is performed (biometric or device PIN)
- The sync provider uses end-to-end encryption

For **AAL3** (highest assurance), NIST currently recommends single-factor cryptographic
device (e.g., YubiKey), as synced passkeys' multi-device exposure does not meet AAL3's
single-factor hardware authenticator requirement. However, NIST is reviewing this position
as of 2025, and some configurations of synced passkeys may qualify for AAL3 in future
revisions.

### 1.6 Account Recovery Implications

When passkeys are synced, account recovery is handled primarily by the sync provider
(Apple/Google/Microsoft), not by the RP. This has critical implications:

```
┌──────────────────────────────────────────────────────────────────────┐
│              ACCOUNT RECOVERY: WHO IS RESPONSIBLE?                   │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Scenario: User loses all devices with synced passkeys               │
│                                                                      │
│  WITH Synced Passkeys:                                               │
│  ┌───────────────────────────────────────────┐                      │
│  │ 1. User recovers Apple/Google account     │                      │
│  │ 2. New device added to account             │                      │
│  │ 3. Passkeys re-synced to new device        │                      │
│  │ 4. User authenticates to RP normally       │                      │
│  │ → RP does NOTHING. Recovery is vendor's    │                      │
│  │   responsibility.                           │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  WITHOUT Sync (Device-Bound):                                        │
│  ┌───────────────────────────────────────────┐                      │
│  │ 1. Device is gone → credential is gone     │                      │
│  │ 2. RP must provide alternative auth         │                      │
│  │    (password, email link, backup codes)     │                      │
│  │ 3. User re-registers new passkey            │                      │
│  │ 4. Old credential is revoked                │                      │
│  │ → RP MUST handle recovery. This is the      │                      │
│  │   RP's responsibility.                       │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Security Question: Can the recovery flow be socially engineered?    │
│  Answer: YES — account recovery is universally the weakest link.     │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 2. Security Analysis of Synced Passkeys

### 2.1 Phishing Resistance Is Preserved

The most critical security property of WebAuthn — **phishing resistance** — is fully
preserved in synced passkeys. This is because phishing resistance comes from the
**origin binding** built into the WebAuthn protocol, not from the key being on a single
physical device.

When a passkey signs an assertion, the authenticator includes the **RP ID hash** (SHA-256
of the RP ID domain) in the authenticator data. The RP verifies this hash matches its
own RP ID. A phishing site at `evil-secure-login.com` cannot produce a valid assertion
for `secure-login.com` because the RP ID hash would not match.

```
┌──────────────────────────────────────────────────────────────────────┐
│             PHISHING RESISTANCE: SYNCED vs DEVICE-BOUND              │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Attacker Scenario: Phishing site mimics ggid.example.com            │
│  Attacker URL: ggid.3xamp1e.com (look-alike domain)                  │
│                                                                      │
│  WebAuthn Assertion Contains:                                        │
│  ┌─────────────────────────────────────────┐                        │
│  │ authenticatorData: {                    │                        │
│  │   rpIdHash: SHA256("ggid.example.com"), │  ← FIXED AT CREATION   │
│  │   flags: { UV=1, BE=1, BS=1 },          │                        │
│  │   signCount: 42,                         │                        │
│  │   ...                                    │                        │
│  │ }                                        │                        │
│  │ clientDataJSON: {                        │                        │
│  │   origin: "https://ggid.example.com",   │  ← CHECKED BY RP       │
│  │   challenge: "abc123...",               │                        │
│  │   type: "webauthn.get"                   │                        │
│  │ }                                        │                        │
│  └─────────────────────────────────────────┘                        │
│                                                                      │
│  RP Verification:                                                    │
│  1. rpIdHash == SHA256("ggid.example.com")? → YES (bound at creation)│
│  2. origin == "https://ggid.example.com"? → YES (browser enforces)   │
│  3. Challenge matches session? → YES (per-session nonce)             │
│                                                                      │
│  Result: Attacker CANNOT get a valid assertion for their domain.     │
│  This holds regardless of whether the passkey is synced or not.      │
│  The key material's origin binding is intrinsic to WebAuthn.         │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 2.2 Loss of Device-Bound Property

The key security difference between synced and device-bound passkeys is the **attack
surface expansion**. With a device-bound credential, an attacker must physically
compromise the specific authenticator hardware. With a synced credential, the key exists
on every synced device, so compromising ANY device compromises the passkey.

```
┌──────────────────────────────────────────────────────────────────────┐
│        ATTACK SURFACE: SYNCED vs DEVICE-BOUND                        │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Device-Bound (YubiKey):                                             │
│  ┌──────────┐                                                        │
│  │ YubiKey  │ ← Attack must target THIS physical device              │
│  │ (1 copy) │   Attack vectors: theft, side-channel, malware-on-host │
│  └──────────┘                                                        │
│                                                                      │
│  Synced (iCloud Keychain):                                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                           │
│  │  iPhone  │  │ MacBook  │  │   iPad   │                           │
│  │  ↑       │  │  ↑       │  │  ↑       │                           │
│  │ Attack   │  │ Attack   │  │ Attack   │  ← ANY device = key       │
│  │ point 1  │  │ point 2  │  │ point 3  │                            │
│  └──────────┘  └──────────┘  └──────────┘                           │
│                                                                      │
│  Compromising any one of the 3 devices grants access to the          │
│  private key, assuming the attacker can bypass user verification     │
│  (biometric/PIN).                                                    │
│                                                                      │
│  Mitigation: Each device has its own User Verification (UV)          │
│  requirement. The key is sealed in hardware (SEP/Titan) and          │
│  requires biometric/PIN to unlock for each authentication.           │
│  But: if device is jailbroken/rooted, UV can be bypassed.            │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 2.3 Compromised Device = Compromised Passkey

If a device with a synced passkey is compromised (e.g., through malware, jailbreak, or
physical theft with sufficient time), the attacker may be able to:

1. **Extract the key from the secure enclave** — extremely difficult but theoretically
   possible with nation-state resources or hardware vulnerabilities.
2. **Intercept the user verification step** — if the device is rooted/jailbroken,
   malware could potentially intercept the UV prompt.
3. **Use the credential while the device is unlocked** — if malware achieves code
   execution in the authenticator's trust domain.

However, these attacks are significantly harder than:
- Phishing a password (trivial)
- Reusing a stolen password (common)
- Brute-forcing a weak password (feasible for many accounts)

So even with the expanded attack surface, synced passkeys are **orders of magnitude**
more secure than passwords.

### 2.4 Apple Secure Enclave Protection

Apple's Secure Enclave Processor (SEP) is a dedicated hardware coprocessor integrated
into Apple Silicon (M-series, A-series, S-series, T2). It provides:

- **Key Generation:** Private keys are generated inside the SEP using a hardware random
  number generator.
- **Key Storage:** Private keys are sealed using a key hierarchy rooted in the SEP's UID
  key — a device-specific key burned into the silicon that cannot be extracted.
- **Cryptographic Operations:** All signing operations occur inside the SEP. The private
  key never exists in plaintext in the main application processor's memory.
- **User Verification:** Biometric matching (Face ID, Touch ID) and device passcode
  verification happen inside the SEP or its Secure Enclave Storage.

```
┌──────────────────────────────────────────────────────────────────────┐
│           APPLE SECURE ENclave KEY ISOLATION                         │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────┐                        │
│  │            Apple SoC (A17/M3)           │                        │
│  │  ┌─────────────────────────────────┐    │                        │
│  │  │      Application Processor      │    │                        │
│  │  │  (iOS / macOS runs here)        │    │                        │
│  │  │                                 │    │                        │
│  │  │  ┌─────┐  ┌─────┐  ┌─────┐    │    │                        │
│  │  │  │App  │  │App  │  │App  │    │    │                        │
│  │  │  │  1  │  │  2  │  │  3  │    │    │                        │
│  │  │  └──┬──┘  └─────┘  └─────┘    │    │                        │
│  │  │     │                            │    │                        │
│  │  └─────┼────────────────────────────┘    │                        │
│  │        │  mailbox / shared memory        │                        │
│  │  ┌─────▼────────────────────────────┐    │                        │
│  │  │       Secure Enclave (SEP)        │    │                        │
│  │  │  ┌────────────────────────────┐  │    │                        │
│  │  │  │  UID Key (hardware-burned) │  │    │                        │
│  │  │  │  Passkey private keys      │  │    │                        │
│  │  │  │  Biometric templates       │  │    │                        │
│  │  │  │  Crypto operations (ECDSA) │  │    │                        │
│  │  │  └────────────────────────────┘  │    │                        │
│  │  │  ← Keys NEVER leave this boundary │    │                        │
│  │  └──────────────────────────────────┘    │                        │
│  └─────────────────────────────────────────┘                        │
│                                                                      │
│  Attack: Even with full application processor compromise (root),     │
│  the SEP's key material cannot be extracted. The only way to USE     │
│  the key is through the SEP's UV-gated API, which requires a         │
│  successful biometric or passcode match.                             │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 2.5 Google Titan Chip Protection

Google's Titan security chips (Titan M on Pixel 3+, Titan M2 on Pixel 6+, Titan C on
Chromebooks) provide hardware-backed key storage similar to the SEP:

- **StrongBox:** Android's StrongBox API routes key operations to the Titan chip,
  isolating them from the main Android OS.
- **User Verification:** Android's BiometricPrompt and device PIN/Pattern verification
  gate access to Titan-backed keys.
- **Verified Boot:** Titan verifies the boot chain, detecting tampering before the OS
  loads.

### 2.6 Comparison Table: Sync Provider Security

```
┌──────────────────┬──────────────┬──────────────┬──────────────┐
│    Property      │    Apple     │    Google    │  Microsoft   │
│                  │ (iCloud KC)  │  (GPM)       │ (Auth/Hello) │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ Hardware Key     │ Secure       │ Titan M/M2   │ TPM 2.0      │
│ Isolation        │ Enclave      │ (StrongBox)  │ (Hello only) │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ E2E Encryption   │ Yes          │ Yes          │ Yes          │
│ of Sync          │              │              │              │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ Vendor Cannot    │ Yes (zero    │ Yes (zero    │ Yes          │
│ Decrypt Keys     │ access)      │ access)      │              │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ Sync Requires    │ Yes (iCloud  │ Yes (Google  │ Yes (MS      │
│ Multi-Factor     │ 2FA)         │ 2FA)         │ 2FA)         │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ Recovery Flow    │ Account      │ Account      │ Account      │
│                  │ Recovery     │ Recovery     │ Recovery     │
│                  │ (delayed)    │ (delayed)    │ (delayed)    │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ Default on       │ Yes (iOS     │ Yes (Android │ Windows Hello│
│ Platform         │ 16+/macOS    │ Chrome)      │ = No sync    │
│                  │ 13+)         │              │ Auth = Sync  │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ BE/BS Flags      │ BE=1, BS=1   │ BE=1, BS=1   │ Hello: BE=0  │
│                  │ (synced)     │ (synced)     │ Auth: BE=1   │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ Sign Count       │ Coordinated  │ Coordinated  │ Per-device   │
│ Behavior         │ via iCloud   │ via Google   │ (no sync for │
│                  │              │              │ Hello)       │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ AAL2 (NIST      │ Eligible     │ Eligible     │ Hello: Yes   │
│ 800-63B)         │              │              │ Auth: Yes    │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ AAL3             │ Under        │ Under        │ Hello (TPM): │
│                  │ review       │ review       │ Potentially  │
├──────────────────┼──────────────┼──────────────┼──────────────┤
│ Audit Logging    │ Apple can    │ Google can   │ MS can log   │
│ (Sync Provider)  │ log sync     │ log sync     │ sync events  │
│                  │ events but   │ events but   │              │
│                  │ NOT keys     │ NOT keys     │              │
└──────────────────┴──────────────┴──────────────┴──────────────┘
```

### 2.7 Clone Detection Implications

Standard WebAuthn clone detection relies on the **sign counter** being monotonically
increasing. If two devices use the same credential and one's counter is lower than the
RP's stored value, the RP detects a potential clone.

For **synced passkeys**, the sync provider coordinates counters so that all synced
copies maintain a consistent counter. This means:
- If the sync is working correctly, all devices advance the same counter.
- If the sync fails (e.g., offline), devices may have divergent counters.
- The RP's clone detection becomes less reliable for synced credentials.

GGID's current implementation checks counter monotonicity (line 727-738 of handler.go).
For synced passkeys, this check is a **best-effort** detection — a true clone attack
using a synced key would advance the counter normally, making detection impossible.

```go
// GGID clone detection (existing — handler.go:727-738)
// Note: For synced passkeys (BE=1), counter-based clone detection
// is less effective because the sync provider coordinates counters
// across devices. A more robust approach for synced credentials:
// - Rate-limit authentications per credential
// - Use device-bound indicators (AAGUID) to detect anomalies
// - Alert when the same credential authenticates from geographically
//   impossible locations in rapid succession

func shouldEnforceCloneDetection(cred *Credential) bool {
    // For device-bound credentials, clone detection is critical
    if !cred.BackupEligible {
        return true
    }
    // For synced credentials, clone detection is best-effort
    // Still enable it, but don't treat low counter as definitive proof
    return true
}
```

---

## 3. Device-to-Device Transfer

### 3.1 Apple AirDrop Passkey Transfer

Apple uses a proximity-based, authenticated transfer mechanism for passkeys between Apple
devices. This is distinct from iCloud Keychain sync — it's a **direct device-to-device
transfer** that uses:

1. **Bluetooth LE** for proximity detection and device discovery.
2. **AWDL (Apple Wireless Direct Link)** for high-speed data transfer.
3. **Apple ID authentication** to verify both devices belong to the same user (or are
   in the user's contacts).

```
┌──────────────────────────────────────────────────────────────────────┐
│          APPLE AIRDROP PASSKEY TRANSFER PROTOCOL                     │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Source Device (iPhone)          Target Device (iPad)               │
│  ┌────────────────────┐          ┌────────────────────┐             │
│  │ 1. BLE advertising │          │ 3. BLE scanning     │             │
│  │    (AWDL discover) │          │    discovers source │             │
│  └─────────┬──────────┘          └──────────┬─────────┘             │
│            │                                  │                       │
│            │  2. BLE proximity check          │                       │
│            │◄────────────────────────────────►│                       │
│            │     (distance estimation)         │                       │
│            │                                  │                       │
│  ┌─────────▼──────────┐          ┌──────────▼─────────┐             │
│  │ 4. TLS mutual auth  │          │ 4. TLS mutual auth │             │
│  │    (Apple ID certs) │◄────────►│   (Apple ID certs) │             │
│  └─────────┬──────────┘          └──────────┬─────────┘             │
│            │                                  │                       │
│  ┌─────────▼──────────┐                      │                       │
│  │ 5. SEP encrypts key │                      │                       │
│  │    with session key │                      │                       │
│  │    derived from ECDH│                      │                       │
│  └─────────┬──────────┘                      │                       │
│            │  6. Encrypted key blob           │                       │
│            │ ──────────────────────────────►  │                       │
│  └─────────┴──────────┘          ┌──────────▼─────────┐             │
│                                  │ 7. Target SEP       │             │
│                                  │    decrypts & seals │             │
│                                  │    key in its own   │             │
│                                  │    SEP key hierarchy│             │
│                                  └────────────────────┘             │
│                                                                      │
│  Security Properties:                                                │
│  - Proximity: BLE ensures devices are physically close (~10m)        │
│  - Authentication: Apple ID certificates verify device ownership     │
│  - Encryption: ECDH session key; key never in plaintext on network  │
│  - No central server: Transfer is P2P, key never touches cloud      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.2 Google QR Code Transfer

Google's passkey transfer between Android devices uses a **QR code + Bluetooth** hybrid
approach:

1. The source device displays a QR code encoding a connection secret.
2. The target device scans the QR code (camera).
3. Bluetooth Low Energy (BLE) establishes a proximity check.
4. A secure channel is established using the QR code secret + BLE proximity.
5. The key material is transferred over the encrypted channel.

```
┌──────────────────────────────────────────────────────────────────────┐
│          GOOGLE QR CODE PASSKEY TRANSFER PROTOCOL                    │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Source Phone                Target Phone                            │
│  ┌────────────────┐          ┌────────────────┐                     │
│  │ 1. Generate QR │          │                │                     │
│  │    code with   │          │ 3. Scan QR     │                     │
│  │    session     │    ┌───► │    code with   │                     │
│  │    secret      │    │     │    camera      │                     │
│  │                │  2.│     │                │                     │
│  │ [QR DISPLAYED] │────┘     └───────┬────────┘                     │
│  └────────┬───────┘                  │                               │
│           │                           │                               │
│           │  4. BLE proximity pairing  │                              │
│           │◄─────────────────────────►│                               │
│           │                           │                               │
│  ┌────────▼───────┐          ┌───────▼────────┐                     │
│  │ 5. ECDH key    │          │ 5. ECDH key    │                     │
│  │    exchange    │◄────────►│    exchange    │                     │
│  └────────┬───────┘          └───────┬────────┘                     │
│           │                           │                               │
│  ┌────────▼───────┐                  │                               │
│  │ 6. Titan seals │                  │                               │
│  │    key with    │  7. Encrypted    │                               │
│  │    session key │──── transfer ───►│                               │
│  │    & sends     │                  │                               │
│  └────────────────┘          ┌───────▼────────┐                     │
│                              │ 8. Titan/M2     │                     │
│                              │    decrypts &   │                     │
│                              │    seals in HW  │                     │
│                              └────────────────┘                     │
│                                                                      │
│  Security Properties:                                                │
│  - QR code provides out-of-band key agreement (MITM-resistant)       │
│  - BLE provides proximity binding (attacker must be nearby)          │
│  - Session key from ECDH encrypts transfer                           │
│  - Titan chip protects key material at rest on both sides            │
│                                                                      │
│  Attack Surface: A nearby attacker could attempt to intercept        │
│  the BLE connection. But without the QR code secret, they cannot     │
│  complete the ECDH exchange. This is a classic "trending MITM"       │
│  scenario where the QR code serves as an authentication factor.      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.3 Security of Transfer Protocol — MITM Risks

The primary risk during device-to-device transfer is a **Man-in-the-Middle (MITM)**
attack. Both Apple and Google mitigate this through:

1. **Proximity binding:** BLE ensures devices are physically close, making remote MITM
   infeasible.
2. **Out-of-band secret:** The QR code (Google) or Apple ID mutual authentication
   (Apple) provides an authenticated key exchange that resists MITM.
3. **End-to-end encryption:** The key material is encrypted with a session key derived
   from the authenticated exchange.

```
┌──────────────────────────────────────────────────────────────────────┐
│             MITM ATTACK ANALYSIS DURING TRANSER                     │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Attacker tries to intercept transfer between Source and Target:    │
│                                                                      │
│  Source ──────────► [ATTACKER] ──────────► Target                   │
│                                                                      │
│  Defense 1: BLE Proximity                                            │
│  ┌─────────────────────────────────────────────┐                    │
│  │ Attacker must be within BLE range (~10m).    │                    │
│  │ If attacker is far away, connection fails.   │                    │
│  │ Proximity is verified via BLE RSSI.          │                    │
│  └─────────────────────────────────────────────┘                    │
│                                                                      │
│  Defense 2: Authenticated Key Exchange                               │
│  ┌─────────────────────────────────────────────┐                    │
│  │ Apple: Apple ID certificates authenticate    │                    │
│  │ both endpoints. Attacker lacks valid certs.  │                    │
│  │                                               │                    │
│  │ Google: QR code provides shared secret.       │                    │
│  │ ECDH binds the secret to the session.         │                    │
│  │ Attacker without QR = no session key.         │                    │
│  └─────────────────────────────────────────────┘                    │
│                                                                      │
│  Defense 3: User Confirmation                                       │
│  ┌─────────────────────────────────────────────┐                    │
│  │ Both devices prompt user for confirmation    │                    │
│  │ before transfer begins. Attacker must also   │                    │
│  │ socially engineer the user into accepting.   │                    │
│  └─────────────────────────────────────────────┘                    │
│                                                                      │
│  Residual Risk: A sophisticated attacker who is physically          │
│  proximate AND has compromised one device could relay the transfer. │
│  This is an extremely targeted attack requiring physical access.     │
│  For most threat models, the transfer is sufficiently secure.       │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.4 Go Code: Transfer Protocol Analysis

Below is a Go implementation showing how a passkey transfer protocol can be analyzed and
its security properties verified. This is a simplified model for research purposes.

```go
package passkeytransfer

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// TransferSession represents the state of a device-to-device passkey transfer.
type TransferSession struct {
	// PrivateKey is the ECDH private key for the session key exchange.
	PrivateKey *ecdh.PrivateKey

	// PeerPublicKey is the other device's ECDH public key.
	PeerPublicKey *ecdh.PublicKey

	// ProximityVerified indicates BLE proximity check passed.
	ProximityVerified bool

	// QRSecret is the out-of-band secret from QR code (Google model).
	// For Apple model, this is nil (uses Apple ID certs instead).
	QRSecret []byte

	// SessionKey is derived from ECDH + optional QR secret.
	SessionKey []byte
}

// NewTransferSession initializes a new transfer session.
// The caller should display their ECDH public key (or encode it in a QR code).
func NewTransferSession() (*TransferSession, error) {
	curve := ecdh.P256()
	privKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ECDH key: %w", err)
	}

	return &TransferSession{
		PrivateKey: privKey,
	}, nil
}

// EstablishSessionKey computes the shared session key from the ECDH exchange.
// For Google's QR model, the QR secret is mixed into the key derivation.
func (s *TransferSession) EstablishSessionKey(peerPubKey *ecdh.PublicKey) error {
	if peerPubKey == nil {
		return fmt.Errorf("peer public key is required")
	}

	s.PeerPublicKey = peerPubKey

	sharedSecret, err := s.PrivateKey.ECDH(peerPubKey)
	if err != nil {
		return fmt.Errorf("ecdh compute: %w", err)
	}

	// Mix in QR secret if present (Google model)
	h := sha256.New()
	h.Write(sharedSecret)
	if len(s.QRSecret) > 0 {
		h.Write(s.QRSecret)
	}

	s.SessionKey = h.Sum(nil)
	return nil
}

// VerifyProximity checks that the BLE proximity check has been performed.
// In a real implementation, this would verify BLE RSSI or Apple's proximity
// estimation protocol.
func (s *TransferSession) VerifyProximity() error {
	if !s.ProximityVerified {
		return fmt.Errorf("BLE proximity not verified — refuse transfer")
	}
	return nil
}

// PrepareTransfer encrypts the passkey private key material for transfer.
// The encrypted blob can only be decrypted by the holder of the session key.
//
// In practice, the source device's Titan/SEP performs this encryption
// inside the secure hardware boundary. This Go code models the protocol
// for analysis purposes.
func (s *TransferSession) PrepareTransfer(keyMaterial []byte) ([]byte, error) {
	if len(s.SessionKey) == 0 {
		return nil, fmt.Errorf("session key not established")
	}
	if err := s.VerifyProximity(); err != nil {
		return nil, err
	}

	// In production: use AES-256-GCM with session key
	// (omitted for clarity; use crypto/aes + crypto/cipher.NewGCM)
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Model: concatenate nonce + keyMaterial for protocol analysis
	// Real implementation would use AEAD encryption
	_ = nonce
	_ = keyMaterial

	return nil, fmt.Errorf("production AEAD encryption omitted in research model")
}

// TransferSecurityReport generates a security analysis of a completed transfer.
type TransferSecurityReport struct {
	SessionID          string
	ProximityVerified  bool
	QRSecretUsed       bool
	MutualAuthPerformed bool
	SessionKeyStrength int // bits
	EncryptionUsed     bool
	VulnerableToMITM   bool
}

// AnalyzeTransfer checks a completed transfer session for security properties.
func AnalyzeTransfer(s *TransferSession) TransferSecurityReport {
	report := TransferSecurityReport{
		ProximityVerified:  s.ProximityVerified,
		QRSecretUsed:       len(s.QRSecret) > 0,
		SessionKeyStrength: len(s.SessionKey) * 8,
	}

	// A transfer is vulnerable to MITM if neither proximity nor
	// out-of-band authentication was used.
	vulnerable := false
	if !s.ProximityVerified && len(s.QRSecret) == 0 {
		vulnerable = true
	}
	if len(s.SessionKey) == 0 {
		vulnerable = true
	}

	report.VulnerableToMITM = vulnerable
	report.EncryptionUsed = len(s.SessionKey) > 0

	return report
}

// EncodePublicKeyForQR encodes the ECDH public key as a base64 URL string
// suitable for embedding in a QR code.
func (s *TransferSession) EncodePublicKeyForQR() string {
	pubBytes := s.PrivateKey.PublicKey().Bytes()
	return base64.RawURLEncoding.EncodeToString(pubBytes)
}

// DecodePublicKeyFromQR decodes a peer's ECDH public key from QR code data.
func DecodePublicKeyFromQR(qrData string) (*ecdh.PublicKey, error) {
	pubBytes, err := base64.RawURLEncoding.DecodeString(qrData)
	if err != nil {
		return nil, fmt.Errorf("decode qr public key: %w", err)
	}
	curve := ecdh.P256()
	return curve.NewPublicKey(pubBytes)
}
```

**Key Takeaway:** The transfer protocol is secure against remote attackers because it
requires physical proximity and an out-of-band secret. The primary risk is a physically
proximate attacker who also has social engineering capability — an extremely targeted
threat.

---

## 4. Non-Synced (Device-Bound) Passkeys

### 4.1 Enterprise Security Preference

Many organizations — especially in finance, government, defense, and healthcare — require
**device-bound credentials** for authentication. This means passkeys that:

1. **Never sync** to any cloud service (`Backup Eligible = false`).
2. **Exist on exactly one physical authenticator.**
3. **Cannot be recovered** if the device is lost (the RP must provide alternative auth).

### 4.2 Why Organizations Require Device-Bound Credentials

```
┌──────────────────────────────────────────────────────────────────────┐
│       WHY ENTERPRISES REQUIRE DEVICE-BOUND PASSKEYS                 │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  1. PROVENANCE & OWNERSHIP                                          │
│     - Device-bound passkeys are tied to a specific hardware token    │
│     - Organization can issue, track, and revoke the token            │
│     - Synced passkeys' provenance is managed by Apple/Google/MS      │
│       — the organization has no visibility or control               │
│                                                                      │
│  2. REGULATORY COMPLIANCE                                           │
│     - NIST 800-63B AAL3 requires single-factor cryptographic device │
│     - HIPAA, FedRAMP, PCI-DSS may require hardware-bound keys       │
│     - EU eIDAS QSCD (Qualified Signature Creation Device)           │
│     - Synced passkeys may not satisfy these requirements            │
│                                                                      │
│  3. ATTACK SURFACE MINIMIZATION                                     │
│     - Device-bound: attacker must compromise ONE device             │
│     - Synced: attacker can target ANY of N synced devices           │
│     - For high-value targets, N=1 is preferable                     │
│                                                                      │
│  4. AUDIT & ATTRIBUTION                                             │
│     - Device-bound: "Authentication from YubiKey #1234"             │
│     - Synced: "Authentication from some device in the sync group"   │
│     - Device-bound provides stronger non-repudiation                │
│                                                                      │
│  5. SUPPLY CHAIN TRUST                                              │
│     - YubiKey: FIDO Certified L1/L2, known supply chain             │
│     - Synced: trust the platform vendor's entire ecosystem          │
│       (SEP + iCloud + Apple ID recovery flow)                       │
│                                                                      │
│  6. POLICY ENFORCEMENT                                              │
│     - Device-bound: RP can enforce "BE=0" at registration           │
│     - Synced: RP has no control over sync provider's policies       │
│     - Enterprise needs deterministic, auditable policy              │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 4.3 YubiKey and Hardware Authenticator Passkeys

YubiKey 5 series (and similar FIDO2 hardware tokens from Feitian, SoloKeys, etc.) create
passkeys that are:

- **BE = false:** Never eligible for backup/sync.
- **BS = false:** Not backed up.
- **Counter-based:** Monotonic counter enables reliable clone detection.
- **FIDO Certified:** Certified to FIDO Authenticator Certification Levels (L1 or L2).
- **No biometric on device (YubiKey 5):** User verification is via touch (UP flag) or
  PIN. YubiKey 5 Bio adds fingerprint biometrics.

```
┌──────────────────────────────────────────────────────────────────────┐
│          YUBIKEY vs SYNCED PASSKEY — AUTHENTICATOR DATA             │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  YubiKey 5 Registration Response:                                   │
│  ┌─────────────────────────────────────────┐                        │
│  │ flags:                                  │                        │
│  │   UP = 1 (user present — touch)         │                        │
│  │   UV = 1 (user verified — PIN)          │                        │
│  │   BE = 0 (backup NOT eligible)          │                        │
│  │   BS = 0 (not backed up)                │                        │
│  │ aaguid: cb69481e-8ff7-4039-93ec-...     │                        │
│  │ → YubiKey 5                             │                        │
│  │ signCount: 1                            │                        │
│  └─────────────────────────────────────────┘                        │
│                                                                      │
│  iPhone (iCloud Keychain) Registration Response:                    │
│  ┌─────────────────────────────────────────┐                        │
│  │ flags:                                  │                        │
│  │   UP = 1 (user present)                 │                        │
│  │   UV = 1 (user verified — Face ID)      │                        │
│  │   BE = 1 (backup ELIGIBLE)              │                        │
│  │   BS = 1 (backed up — synced to iCloud) │                        │
│  │ aaguid: fbfc3007-154e-4ecc-8c0b-...     │                        │
│  │ → Touch ID / Face ID                   │                        │
│  │ signCount: 0 (synced: counter managed   │                        │
│  │   by iCloud, often 0 or coordinated)    │                        │
│  └─────────────────────────────────────────┘                        │
│                                                                      │
│  RP can distinguish these at registration time by checking:         │
│  1. BE flag: 0 = device-bound, 1 = synced                           │
│  2. AAGUID: identifies authenticator model                          │
│  3. Attachment: platform (built-in) vs cross-platform (roaming)     │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 4.4 Per-Tenant Policy for Synced vs Device-Bound

GGID is a multi-tenant IAM system. Different tenants may have different security
requirements. A financial services tenant might require device-bound passkeys, while a
consumer SaaS tenant might prefer synced passkeys for better UX.

The policy should be configurable per-tenant:

```go
// Package webauthnpolicy implements per-tenant passkey security policy.
//
// Tenants can configure whether synced passkeys (BE=1) are allowed,
// required, or prohibited. This enables fine-grained control over
// the security vs. convenience tradeoff.
package webauthnpolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

// SyncPolicy defines a tenant's passkey sync requirement.
type SyncPolicy int

const (
	// SyncPolicyAny allows both synced and device-bound passkeys.
	// This is the default for most consumer-facing applications.
	SyncPolicyAny SyncPolicy = iota

	// SyncPolicySyncedOnly requires passkeys that are backup-eligible
	// (BE=1). This ensures users can recover across devices.
	// Use case: consumer SaaS where recovery > single-device security.
	SyncPolicySyncedOnly

	// SyncPolicyDeviceBoundOnly requires passkeys that are NOT
	// backup-eligible (BE=0). This is the enterprise high-security mode.
	// Use case: finance, government, healthcare where device-bound
	// is a compliance requirement (AAL3, FedRAMP, etc.).
	SyncPolicyDeviceBoundOnly

	// SyncPolicyHybrid allows synced passkeys but requires at least
	// one device-bound passkey for admin/elevated access.
	SyncPolicyHybrid
)

// AttachmentPolicy defines which authenticator attachments are allowed.
type AttachmentPolicy int

const (
	// AttachmentPolicyAny allows both platform and cross-platform.
	AttachmentPolicyAny AttachmentPolicy = iota

	// AttachmentPolicyPlatformOnly requires platform authenticators
	// (built into the device: Touch ID, Face ID, Windows Hello).
	AttachmentPolicyPlatformOnly

	// AttachmentPolicyCrossPlatformOnly requires roaming authenticators
	// (USB/NFC security keys: YubiKey, Feitian ePass, etc.).
	AttachmentPolicyCrossPlatformOnly
)

// TenantPolicy is the per-tenant WebAuthn security policy.
type TenantPolicy struct {
	TenantID         uuid.UUID
	SyncPolicy       SyncPolicy
	AttachmentPolicy AttachmentPolicy

	// MaxCredentialsPerUser limits the number of passkeys a user can
	// register. Default: 10.
	MaxCredentialsPerUser int

	// RequireUserVerification enforces UV=required at registration
	// and authentication. If false, UV=preferred.
	RequireUserVerification bool

	// AllowedAAGUIDs is an allowlist of permitted authenticator models.
	// If empty, all FIDO-certified authenticators are allowed.
	// Example: only allow YubiKey 5 for a high-security tenant.
	AllowedAAGUIDs []string

	// DeniedAAGUIDs is a denylist of prohibited authenticator models.
	// Example: deny a known-compromised authenticator model.
	DeniedAAGUIDs []string
}

// PolicyStore retrieves tenant WebAuthn policies.
type PolicyStore interface {
	GetPolicy(ctx context.Context, tenantID uuid.UUID) (*TenantPolicy, error)
}

// EvaluateRegistration checks whether a credential creation response
// satisfies the tenant's policy. This should be called AFTER the
// go-webauthn library verifies the attestation, but BEFORE persisting
// the credential.
//
// Returns nil if the credential is allowed, or an error explaining
// why the credential violates policy.
func EvaluateRegistration(
	policy *TenantPolicy,
	backupEligible bool,
	attachment protocol.AuthenticatorAttachment,
	aaguid string,
) error {
	if policy == nil {
		return nil // No policy = allow all
	}

	// Check sync policy
	switch policy.SyncPolicy {
	case SyncPolicySyncedOnly:
		if !backupEligible {
			return fmt.Errorf("tenant policy requires synced (backup-eligible) passkeys; " +
				"this authenticator is device-bound")
		}
	case SyncPolicyDeviceBoundOnly:
		if backupEligible {
			return fmt.Errorf("tenant policy requires device-bound passkeys; " +
				"this authenticator supports sync (BE=1) which is prohibited")
		}
	case SyncPolicyAny, SyncPolicyHybrid:
		// Both allowed
	}

	// Check attachment policy
	switch policy.AttachmentPolicy {
	case AttachmentPolicyPlatformOnly:
		if attachment != protocol.Platform {
			return fmt.Errorf("tenant policy requires platform authenticators; " +
				"got cross-platform (roaming) authenticator")
		}
	case AttachmentPolicyCrossPlatformOnly:
		if attachment != protocol.CrossPlatform {
			return fmt.Errorf("tenant policy requires cross-platform (roaming) " +
				"authenticators; got platform authenticator")
		}
	case AttachmentPolicyAny:
		// Both allowed
	}

	// Check AAGUID allowlist
	if len(policy.AllowedAAGUIDs) > 0 {
		found := false
		for _, allowed := range policy.AllowedAAGUIDs {
			if strings.EqualFold(allowed, aaguid) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("authenticator model (AAGUID %s) not in tenant allowlist", aaguid)
		}
	}

	// Check AAGUID denylist
	for _, denied := range policy.DeniedAAGUIDs {
		if strings.EqualFold(denied, aaguid) {
			return fmt.Errorf("authenticator model (AAGUID %s) is prohibited by tenant policy", aaguid)
		}
	}

	return nil
}

// DefaultPolicy returns the permissive default policy (SyncPolicyAny).
func DefaultPolicy(tenantID uuid.UUID) *TenantPolicy {
	return &TenantPolicy{
		TenantID:               tenantID,
		SyncPolicy:             SyncPolicyAny,
		AttachmentPolicy:       AttachmentPolicyAny,
		MaxCredentialsPerUser:  10,
		RequireUserVerification: false,
	}
}

// EnterpriseHighSecurityPolicy returns a strict device-bound policy.
func EnterpriseHighSecurityPolicy(tenantID uuid.UUID) *TenantPolicy {
	return &TenantPolicy{
		TenantID:               tenantID,
		SyncPolicy:             SyncPolicyDeviceBoundOnly,
		AttachmentPolicy:       AttachmentPolicyCrossPlatformOnly,
		MaxCredentialsPerUser:  3,
		RequireUserVerification: true,
		AllowedAAGUIDs: []string{
			"cb69481e-8ff7-4039-93ec-0a2729a154a8", // YubiKey 5
			"ee042887-7e46-4ccc-ab97-a80a032e1234", // Feitian ePass FIDO
			// Add other approved hardware tokens
		},
	}
}

// ConfigureAuthenticatorSelection adjusts the AuthenticatorSelection criteria
// based on the tenant policy. This is used during BeginRegistration to
// constrain what authenticators the browser will offer.
func ConfigureAuthenticatorSelection(policy *TenantPolicy) protocol.AuthenticatorSelection {
	sel := protocol.AuthenticatorSelection{
		ResidentKey:      protocol.ResidentKeyRequirementPreferred,
		UserVerification: protocol.VerificationPreferred,
	}

	if policy == nil {
		return sel
	}

	// Set user verification requirement
	if policy.RequireUserVerification {
		sel.UserVerification = protocol.VerificationRequired
	}

	// Set authenticator attachment
	switch policy.AttachmentPolicy {
	case AttachmentPolicyPlatformOnly:
		sel.AuthenticatorAttachment = protocol.Platform
	case AttachmentPolicyCrossPlatformOnly:
		sel.AuthenticatorAttachment = protocol.CrossPlatform
	default:
		// Leave unset = any
	}

	return sel
}
```

**Integration point:** The `EvaluateRegistration` function should be called in
`finishRegistration` (handler.go line ~535) after the go-webauthn library creates the
credential, but before persisting it. The `ConfigureAuthenticatorSelection` function
should be called in `beginRegistration` (handler.go line ~470) to constrain the
authenticator options sent to the browser.

---

## 5. Account Recovery Implications

### 5.1 The Fundamental Problem

Account recovery is universally acknowledged as the **weakest link** in any authentication
system. The reason is structural:

- Authentication systems are designed to be hard to bypass.
- Recovery must provide a bypass when legitimate users lose their factors.
- Any bypass that works for legitimate users can potentially be exploited by attackers.

```
┌──────────────────────────────────────────────────────────────────────┐
│          THE RECOVERY PARADOX                                        │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Authentication Security = f(difficulty of bypassing auth)           │
│  Recovery Usability   = f(ease of bypassing auth)                   │
│                                                                      │
│  These are in DIRECT CONFLICT.                                       │
│  Making recovery easier → making attack easier.                     │
│  Making recovery harder → legitimate users locked out.              │
│                                                                      │
│  WebAuthn does NOT solve this.                                       │
│  WebAuthn makes AUTHENTICATION stronger,                              │
│  but RECOVERY is still the soft underbelly.                          │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 5.2 Scenario: User Loses All Synced Devices

```
┌──────────────────────────────────────────────────────────────────────┐
│       SCENARIO: ALL SYNCED DEVICES LOST                              │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  User has passkey synced to:                                        │
│  - iPhone (lost)                                                     │
│  - iPad (lost)                                                       │
│  - MacBook (stolen)                                                  │
│                                                                      │
│  Step 1: User acquires new device (new iPhone)                      │
│  Step 2: User signs into Apple ID on new device                     │
│  Step 3: Apple Account Recovery:                                    │
│    ┌───────────────────────────────────────────┐                    │
│    │ Option A: Trusted phone number (SMS OTP)  │                    │
│    │ → Security: SMS is interceptable (SIM swap)│                   │
│    │                                            │                    │
│    │ Option B: Trusted device approval          │                    │
│    │ → NOT AVAILABLE: all trusted devices lost  │                    │
│    │                                            │                    │
│    │ Option C: Account Recovery Key             │                    │
│    │ → IF user saved it when setting up 2FA     │                    │
│    │ → Most users don't save this               │                    │
│    │                                            │                    │
│    │ Option D: Identity verification + delay    │                    │
│    │ → Apple verifies identity (ID, etc.)       │                    │
│    │ → 24-72 hour waiting period                │                    │
│    │ → Security: social engineering possible    │                    │
│    └───────────────────────────────────────────┘                    │
│  Step 4: Once Apple account recovered → iCloud Keychain syncs      │
│  Step 5: Passkeys appear on new device                              │
│  Step 6: User authenticates to RP (GGID) normally                   │
│                                                                      │
│  RP (GGID) was NOT involved in recovery.                            │
│  Security of recovery = security of Apple Account Recovery.         │
│  This is OUTSIDE the RP's control.                                   │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 5.3 Apple Account Recovery Flow

Apple's account recovery involves:

1. **Trusted Phone Number:** SMS or voice call with verification code.
2. **Trusted Device:** Push notification to a trusted device (if available).
3. **Account Recovery Contact:** A trusted contact who can generate a recovery code.
4. **Recovery Key:** A 28-character key generated during 2FA setup.
5. **Identity Verification + Delay:** If all else fails, Apple verifies identity
   through official documentation and imposes a waiting period.

**Security Assessment:**
- SMS-based recovery is vulnerable to SIM swapping attacks.
- Account Recovery Contacts provide a "social recovery" mechanism — the contact must
  have an Apple device and be trusted by the user.
- The mandatory delay (24-72h) is a rate-limiting mechanism against social engineering,
  giving the legitimate user time to detect and report unauthorized recovery attempts.

### 5.4 Google Account Recovery Flow

Google's recovery flow similarly involves:

1. **Trusted Device:** Notification to a device signed into the Google Account.
2. **SMS/Voice:** Verification code via phone.
3. **Backup Codes:** Pre-generated one-time codes.
4. **Google Prompt:** Push notification to another Google-signed-in device.
5. **Identity Verification:** Government ID verification with delay.

### 5.5 FIDO Account Recovery

The FIDO Alliance has published guidance on account recovery for passkey-based systems.
Two main approaches:

```
┌──────────────────────────────────────────────────────────────────────┐
│         FIDO ACCOUNT RECOVERY APPROACHES                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Approach 1: ADMIN RESET (Enterprise)                                │
│  ┌───────────────────────────────────────────────┐                  │
│  │ 1. User reports lost device to IT helpdesk    │                  │
│  │ 2. Admin verifies user identity (in person,   │                  │
│  │    via manager approval, etc.)                 │                  │
│  │ 3. Admin revokes old credential in GGID        │                  │
│  │ 4. Admin issues temporary auth (email link,    │                  │
│  │    one-time passcode)                          │                  │
│  │ 5. User re-registers new passkey               │                  │
│  │ Security: Depends on admin verification        │                  │
│  │           process quality                      │                  │
│  └───────────────────────────────────────────────┘                  │
│                                                                      │
│  Approach 2: SOCIAL RECOVERY (Consumer)                              │
│  ┌───────────────────────────────────────────────┐                  │
│  │ 1. User has pre-selected recovery contacts     │                  │
│  │    (trusted friends/family with accounts)      │                  │
│  │ 2. User requests recovery                      │                  │
│  │ 3. N-of-M recovery contacts must approve       │                  │
│    │    (e.g., 2-of-3 trusted contacts)           │                  │
│  │ 4. Each contact authenticates + approves       │                  │
│  │ 5. System grants temporary auth for            │                  │
│  │    re-registration                              │                  │
│  │ Security: Resists single-contact compromise    │                  │
│  │           Requires M contacts to collude       │                  │
│  └───────────────────────────────────────────────┘                  │
│                                                                      │
│  Approach 3: FALLBACK AUTHENTICATION                                │
│  ┌───────────────────────────────────────────────┐                  │
│  │ User maintains a secondary auth method:        │                  │
│  │ - Password (if still enabled)                  │                  │
│  │ - Email magic link                             │                  │
│  │ - TOTP backup codes                            │                  │
│  │ - Secondary passkey on different device        │                  │
│  │ Security: Weakest secondary method = overall   │                  │
│  │           security. Email link = phishing risk. │                  │
│  └───────────────────────────────────────────────┘                  │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 5.6 GGID Recovery Flow Design

For GGID as a multi-tenant IAM system, the recovery flow should be **configurable per
tenant** and should account for both synced and device-bound passkey scenarios:

```go
// Package recovery implements passkey account recovery flows.
package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RecoveryMethod defines how a user can recover access when passkeys are lost.
type RecoveryMethod int

const (
	// RecoveryNone: no recovery available (extreme high-security mode).
	// User MUST keep their passkey safe. Lost passkey = permanent lockout
	// unless admin manually intervenes.
	RecoveryNone RecoveryMethod = iota

	// RecoveryAdminReset: helpdesk/admin can reset credentials.
	// Requires admin to verify user identity through an out-of-band process.
	RecoveryAdminReset

	// RecoveryEmailLink: one-time magic link sent to registered email.
	// Security: vulnerable to email compromise and phishing.
	RecoveryEmailLink

	// RecoveryTOTPBackupCodes: pre-generated one-time codes (like 2FA backup codes).
	// User must have saved codes at setup time.
	RecoveryTOTPBackupCodes

	// RecoverySecondaryPasskey: user has a second passkey on a different device
	// or authenticator that was not lost.
	RecoverySecondaryPasskey

	// RecoverySocialRecovery: N-of-M trusted contacts must approve.
	RecoverySocialRecovery
)

// RecoveryConfig defines the tenant's recovery policy.
type RecoveryConfig struct {
	TenantID          uuid.UUID
	PrimaryMethod     RecoveryMethod
	FallbackMethods   []RecoveryMethod
	MandatoryWaitTime time.Duration // Delay between request and grant (anti-social-engineering)
	MaxAttempts       int           // Rate limit recovery attempts
}

// RecoveryRequest represents a user's recovery attempt.
type RecoveryRequest struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	UserID       uuid.UUID
	Method       RecoveryMethod
	RequestedAt  time.Time
	GrantedAt    *time.Time // nil = not yet granted
	Status       string     // "pending", "approved", "denied", "expired"
	ApproverIDs  []uuid.UUID // For social recovery: who approved
}

// EvaluateRecoveryRequest checks whether a recovery request can be granted
// immediately or must wait for the mandatory delay period.
//
// The mandatory delay is a critical defense against social engineering.
// Even if an attacker convinces a helpdesk to reset a credential, the
// delay gives the legitimate user time to detect and cancel the reset.
func EvaluateRecoveryRequest(
	req *RecoveryRequest,
	cfg *RecoveryConfig,
	now time.Time,
) (canGrant bool, reason string) {
	if req == nil || cfg == nil {
		return false, "invalid request or config"
	}

	// Check rate limiting
	if cfg.MaxAttempts > 0 {
		// In production: check attempt count from store
	}

	// Check mandatory wait time
	elapsed := now.Sub(req.RequestedAt)
	if elapsed < cfg.MandatoryWaitTime {
		remaining := cfg.MandatoryWaitTime - elapsed
		return false, fmt.Sprintf(
			"recovery request is pending; %v remaining in mandatory wait period",
			remaining.Round(time.Minute),
		)
	}

	// Check method-specific requirements
	switch req.Method {
	case RecoveryAdminReset:
		if len(req.ApproverIDs) == 0 {
			return false, "admin reset requires admin approval"
		}
	case RecoverySocialRecovery:
		// Require N-of-M approvals (configured per tenant)
		if len(req.ApproverIDs) < 2 {
			return false, "social recovery requires at least 2 approvers"
		}
	case RecoveryTOTPBackupCodes:
		// Backup codes are verified by the caller before calling this function
		// (they're one-time-use, verified against stored hash)
	case RecoveryEmailLink:
		// Email link must be clicked within validity window
		// This is the weakest method — flag it
	case RecoverySecondaryPasskey:
		// User must authenticate with their secondary passkey
		// (handled by normal WebAuthn flow with remaining credential)
	case RecoveryNone:
		return false, "recovery is disabled for this tenant"
	}

	return true, ""
}

// DefaultRecoveryConfig returns a balanced recovery configuration.
// - Primary: admin reset (for enterprise) or secondary passkey
// - Fallback: email link (as last resort)
// - Wait time: 24 hours (enterprise) or 1 hour (consumer)
func DefaultRecoveryConfig(tenantID uuid.UUID) *RecoveryConfig {
	return &RecoveryConfig{
		TenantID:          tenantID,
		PrimaryMethod:     RecoveryAdminReset,
		FallbackMethods:   []RecoveryMethod{RecoveryEmailLink},
		MandatoryWaitTime: 24 * time.Hour,
		MaxAttempts:       3,
	}
}

// ConsumerRecoveryConfig returns a consumer-friendly recovery config.
// - Primary: email magic link
// - Fallback: backup codes
// - Wait time: 15 minutes (faster for consumer UX)
func ConsumerRecoveryConfig(tenantID uuid.UUID) *RecoveryConfig {
	return &RecoveryConfig{
		TenantID:          tenantID,
		PrimaryMethod:     RecoveryEmailLink,
		FallbackMethods:   []RecoveryMethod{RecoveryTOTPBackupCodes},
		MandatoryWaitTime: 15 * time.Minute,
		MaxAttempts:       5,
	}
}
```

### 5.7 Why Recovery Is the Weakest Link

```
┌──────────────────────────────────────────────────────────────────────┐
│     WHY RECOVERY DEFEATS PASSKEY SECURITY                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Passkey Authentication:                                             │
│  ┌───────────────────────────────────────────┐                      │
│  │ Security level: VERY HIGH                  │                      │
│  │ - Origin binding (anti-phishing)           │                      │
│  │ - Hardware key isolation (anti-extraction) │                      │
│  │ - User verification (anti-theft)           │                      │
│  │ - Challenge-response (anti-replay)         │                      │
│  │ Overall: ~10/10                            │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Account Recovery (Email Link):                                      │
│  ┌───────────────────────────────────────────┐                      │
│  │ Security level: LOW                        │                      │
│  │ - Email password can be guessed/stolen     │                      │
│  │ - Email can be socially engineered         │                      │
│  │ - Email provider can be compromised        │                      │
│  │ - No origin binding (phishing possible)    │                      │
│  │ - No hardware protection                   │                      │
│  │ Overall: ~3/10                             │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  COMPOSITE SYSTEM SECURITY = MIN(auth, recovery)                    │
│  = MIN(10/10, 3/10)                                                 │
│  = 3/10                                                              │
│                                                                      │
│  An attacker will ALWAYS target the weakest link.                   │
│  If email recovery is enabled, the attacker doesn't need to          │
│  defeat the passkey — they just need to defeat the email.            │
│                                                                      │
│  CONCLUSION: A passkey system with weak recovery is barely          │
│  better than a password system with weak recovery.                  │
│  The recovery flow MUST be as strong as the primary auth flow.      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

**Mitigation strategies:**
1. Require a mandatory delay (24h+) for all non-instant recovery methods.
2. Use N-of-M social recovery instead of single-factor email recovery.
3. Allow users to disable email-based recovery entirely (high-security mode).
4. Notify all registered devices when a recovery attempt is initiated.
5. Require step-up authentication for recovery (e.g., answer security questions +
   email link + wait period).

---

## 6. Passwordless Sync UX

### 6.1 User Experience of Synced Passkeys

Synced passkeys provide a significantly better user experience than device-bound
passkeys:

```
┌──────────────────────────────────────────────────────────────────────┐
│         SYNCED PASSKEY UX — SEAMLESS ACROSS DEVICES                 │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Day 1: Registration on iPhone                                      │
│  ┌───────────────────────────────────────────┐                      │
│  │ User visits ggid.example.com on iPhone    │                      │
│  │ → "Create passkey" button                 │                      │
│  │ → Face ID prompt                          │                      │
│  │ → Passkey created (2 taps, ~3 seconds)    │                      │
│  │ → Synced to iCloud Keychain automatically │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Day 2: Login on MacBook                                            │
│  ┌───────────────────────────────────────────┐                      │                      │
│  │ User visits ggid.example.com on MacBook   │                      │
│  │ → Autofill shows passkey credential       │                      │
│  │ → Touch ID prompt                         │                      │
│  │ → Authenticated (1 tap, ~2 seconds)       │                      │
│  │ → NO re-registration needed!              │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Day 3: Login on iPad                                               │
│  ┌───────────────────────────────────────────┐                      │
│  │ Same as MacBook — passkey already synced  │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Total user effort: 1 registration, 0 re-registrations              │
│  Time saved vs device-bound: ~5 minutes per device                  │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 6.2 Non-Synced (Device-Bound) UX

```
┌──────────────────────────────────────────────────────────────────────┐
│      DEVICE-BOUND PASSKEY UX — RE-REGISTRATION PER DEVICE            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Day 1: Registration on Work Laptop (YubiKey)                       │
│  ┌───────────────────────────────────────────┐                      │
│  │ User visits ggid.example.com              │                      │
│  │ → "Create passkey" button                 │                      │
│  │ → Touch YubiKey                           │                      │
│  │ → Enter YubiKey PIN                       │                      │
│  │ → Passkey created (~10 seconds)           │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Day 2: Want to login from Phone                                    │
│  ┌───────────────────────────────────────────┐                      │
│  │ User visits ggid.example.com on phone     │                      │
│  │ → No passkey available on phone!          │                      │
│  │ → Must register a NEW passkey:            │                      │
│  │   Option A: Use YubiKey via NFC           │                      │
│  │   Option B: Register phone's biometric    │                      │
│  │             (but phone passkey = BE=1,    │                      │
│  │              might violate policy)        │                      │
│  │ → 5+ taps, ~20 seconds                    │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Day 3: Want to login from Home Desktop                             │
│  ┌───────────────────────────────────────────┐                      │
│  │ Same problem — must register AGAIN        │                      │
│  │ → Insert YubiKey, register, verify        │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Total user effort: N registrations for N devices                   │
│  Lost device = lost credential = must re-register + admin revoke    │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 6.3 UX Recommendation Matrix

```
┌───────────────────┬──────────────┬──────────────┬──────────────────┐
│   Use Case        │  Recommended │  Why         │  Tradeoff        │
│                   │  Passkey Type│              │                  │
├───────────────────┼──────────────┼──────────────┼──────────────────┤
│ Consumer SaaS     │ Synced       │ Best UX,     │ Relies on vendor │
│ (social, retail,  │ (BE=1)       │ no lost-     │ recovery         │
│ media)            │              │ device lockout│ security          │
├───────────────────┼──────────────┼──────────────┼──────────────────┤
│ Enterprise SaaS   │ Synced +     │ Convenience  │ Must implement   │
│ (B2B SaaS,        │ allow device-│ + high-value │ policy for admin │
│ productivity)     │ bound option │ users get    │ recovery         │
│                   │              │ YubiKey      │                  │
├───────────────────┼──────────────┼──────────────┼──────────────────┤
│ Financial Services│ Device-Bound │ Regulatory   │ Higher support   │
│ (banking,         │ (BE=0)       │ compliance,  │ burden, users    │
│ trading)          │              │ AAL3 target  │ need backup key  │
├───────────────────┼──────────────┼──────────────┼──────────────────┤
│ Government /      │ Device-Bound │ FedRAMP,     │ Must issue       │
│ Defense           │ (BE=0)       │ AAL3,        │ hardware tokens  │
│                   │              │ provenance   │ to all users     │
├───────────────────┼──────────────┼──────────────┼──────────────────┤
│ Healthcare        │ Device-Bound │ HIPAA, audit │ Badge/PIV card   │
│                   │ (BE=0)       │ trail, PIV   │ integration      │
├───────────────────┼──────────────┼──────────────┼──────────────────┤
│ Developer /       │ Either       │ Developers   │ Offer both; let  │
│ Technical Users   │              │ can choose   │ users decide     │
├───────────────────┼──────────────┼──────────────┼──────────────────┤
│ Admin / Root      │ Device-Bound │ Highest      │ Admins get       │
│ Accounts          │ (BE=0)       │ assurance    │ YubiKeys; no     │
│                   │              │              │ exception        │
└───────────────────┴──────────────┴──────────────┴──────────────────┘
```

### 6.4 Registration UX: Conditional Mediation (Autofill)

Modern browsers support **conditional mediation** — the WebAuthn API can populate the
browser's autofill dropdown with available passkeys. This allows users to select a
passkey from the autofill UI without an explicit "Login with passkey" button.

```javascript
// Frontend code (shown for reference; GGID console uses React/Next.js)
// This enables passkey autofill in the username field

const credential = await navigator.credentials.get({
  mediation: "conditional",  // enables autofill UI
  publicKey: {
    challenge: base64url.decode(challenge),
    rpId: "ggid.example.com",
    userVerification: "preferred",
    // No allowCredentials → discoverable credential flow
  }
});
```

On the server side (GGID), the `beginAuthentication` endpoint should support the
discoverable credential flow (no `allowCredentials` array), which GGID already does
when `user_id` is not provided (handler.go lines 638-649).

---

## 7. WebAuthn Transport Hints

### 7.1 Transport Hints Overview

The WebAuthn `transports` field hints to the browser about how to communicate with the
authenticator. The standard transports are:

| Transport | Description | Example Authenticators |
|-----------|-------------|------------------------|
| `internal` | Platform authenticator built into the device | Touch ID, Face ID, Windows Hello |
| `hybrid` | Phone-as-authenticator via caBLE (QR + BLE) | Android phone authenticating a desktop |
| `cross-platform` (deprecated alias) | Roaming authenticator | Was used for USB/NFC keys |
| `usb` | USB-connected security key | YubiKey over USB |
| `nfc` | NFC-tap security key | YubiKey over NFC |
| `ble` | Bluetooth security key | Feitian BT key |
| `smart-card` | Smart card reader | PIV/CAC cards |

```
┌──────────────────────────────────────────────────────────────────────┐
│            TRANSPORT HINT FLOW                                       │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Registration:                                                       │
│  ┌───────────────────────────────────────────┐                      │
│  │ Authenticator creates credential            │                      │
│  │ → includes transports array in response     │                      │
│  │   e.g. ["internal", "hybrid"]               │                      │
│  │ RP (GGID) STORES these transports            │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Authentication:                                                     │
│  ┌───────────────────────────────────────────┐                      │
│  │ RP (GGID) sends allowCredentials           │                      │
│  │ → includes stored transports per credential │                      │
│  │ Browser uses transports to select UI:       │                      │
│  │   "internal" → show platform prompt          │                      │
│  │   "hybrid" → show QR code option            │                      │
│  │   "usb" → prompt to insert USB key          │                      │
│  │   "nfc" → prompt to tap NFC key             │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
│  Synced Passkey Transports:                                          │
│  ┌───────────────────────────────────────────┐                      │
│  │ iPhone passkey: ["internal", "hybrid"]      │                      │
│  │ → "internal": use directly on iPhone         │                      │
│  │ → "hybrid": iPhone can serve desktop via QR │                      │
│  │                                               │                      │
│  │ MacBook passkey (synced): ["internal"]       │                      │
│  │ → "internal": use directly on MacBook        │                      │
│  │                                               │                      │
│  │ YubiKey passkey: ["usb", "nfc"]              │                      │
│  │ → "usb": insert into USB port               │                      │
│  │ → "nfc": tap on NFC reader                  │                      │
│  └───────────────────────────────────────────┘                      │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 7.2 How RP Knows Which Transports Are Available

The RP learns transports in two ways:

1. **During registration:** The authenticator's `getTransports()` method returns the
   transports array in the `CredentialCreationResponse`. GGID stores these in the
   `Credential.Transports` field (handler.go line 549-556).

2. **During authentication (allowCredentials):** The RP sends the stored transports back
   in the `allowCredentials` array. GGID does this correctly (handler.go lines 619-630).

For **discoverable credentials** (no `allowCredentials`), the browser manages transport
selection internally — the RP doesn't need to provide transports.

### 7.3 Conditional Mediation with Synced Credentials

When a passkey is synced across devices, conditional mediation (autofill) becomes
especially powerful. The browser can show the synced passkey in the autofill dropdown
even when the user is on a different device than where they registered.

```
┌──────────────────────────────────────────────────────────────────────┐
│      CONDITIONAL MEDIATION WITH SYNCED CREDENTIALS                  │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  User on MacBook, passkey registered on iPhone (synced via iCloud)  │
│                                                                      │
│  Browser Autofill Dropdown:                                          │
│  ┌────────────────────────────────────────┐                         │
│  │ Username: [user@example.com        ▼]  │                         │
│  │           ┌──────────────────────────┐ │                         │
│  │           │ Passkey for ggid.example │ │                         │
│  │           │ Use Touch ID             │ │                         │
│  │           └──────────────────────────┘ │                         │
│  │ Password: [                          ]  │                         │
│  └────────────────────────────────────────┘                         │
│                                                                      │
│  User selects the passkey → Touch ID prompt → authenticated.        │
│  The synced passkey from iPhone is available because iCloud         │
│  Keychain synced it to the MacBook.                                 │
│                                                                      │
│  This is ONLY possible with synced passkeys (BE=1, BS=1).           │
│  Device-bound passkeys (YubiKey) require explicit insertion.        │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 7.4 Browser Mediation API

The browser mediation API (`navigator.credentials.get` with `mediation: "conditional"`)
is the frontend interface for passkey autofill. The server-side responsibility is to:

1. **Return discoverable credential options** (empty or absent `allowCredentials`).
2. **Include user verification preference** (usually "preferred" or "required").
3. **Support the assertion response** (handle discoverable credential authentication
   where the user is identified from the assertion, not from a pre-provided user_id).

GGID supports this flow when `user_id` is not provided in `beginAuthentication`
(handler.go lines 638-649), creating an ephemeral user for the discoverable flow.

### 7.5 Go Code: Transport-Based Authenticator Selection

```go
// Package transportselection provides transport-based authenticator selection
// logic for WebAuthn authentication.
package transportselection

import (
	"fmt"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
)

// TransportInfo describes the transport characteristics of a credential.
type TransportInfo struct {
	// Transports are the raw transport hints from registration.
	Transports []protocol.AuthenticatorTransport

	// IsPlatform indicates the credential uses a platform authenticator
	// (internal transport).
	IsPlatform bool

	// IsRoaming indicates the credential uses a roaming authenticator
	// (usb, nfc, ble).
	IsRoaming bool

	// IsHybrid indicates the credential can use the hybrid transport
	// (phone-as-authenticator via QR + BLE).
	IsHybrid bool

	// IsSynced indicates the credential is likely synced (BE=1).
	// This is inferred from transports: "internal" on platform authenticators
	// that support sync (Apple, Google, Microsoft) typically means synced.
	IsSynced bool
}

// AnalyzeTransports determines the transport characteristics of a credential.
func AnalyzeTransports(transports []string, backupEligible bool) TransportInfo {
	info := TransportInfo{}

	for _, t := range transports {
		switch protocol.AuthenticatorTransport(t) {
		case protocol.Internal:
			info.IsPlatform = true
		case protocol.Hybrid:
			info.IsHybrid = true
		case protocol.USB, protocol.NFC, protocol.BLE:
			info.IsRoaming = true
		}
	}

	// If no transports recorded, assume platform (default for most passkeys)
	if len(transports) == 0 {
		info.IsPlatform = true
	}

	info.IsSynced = backupEligible && info.IsPlatform

	// Copy transports
	for _, t := range transports {
		info.Transports = append(info.Transports, protocol.AuthenticatorTransport(t))
	}

	return info
}

// SelectAuthenticatorMethod recommends the best authentication method based on
// available transports and the requesting context.
//
// Priority:
// 1. internal (platform) — fastest, best UX
// 2. hybrid (phone-as-authenticator) — good for cross-device
// 3. usb — for security keys
// 4. nfc — for NFC security keys
func SelectAuthenticatorMethod(info TransportInfo) (method string, uiHint string) {
	if info.IsPlatform {
		return "internal", "Use your device's biometric (Touch ID / Face ID / Windows Hello)"
	}
	if info.IsHybrid {
		return "hybrid", "Scan the QR code with your phone to authenticate"
	}
	if info.IsRoaming {
		if hasTransport(info.Transports, protocol.USB) {
			return "usb", "Insert your security key and tap it"
		}
		if hasTransport(info.Transports, protocol.NFC) {
			return "nfc", "Tap your security key on the NFC reader"
		}
	}
	return "any", "Authenticate with any available method"
}

func hasTransport(transports []protocol.AuthenticatorTransport, target protocol.AuthenticatorTransport) bool {
	for _, t := range transports {
		if t == target {
			return true
		}
	}
	return false
}

// BuildAllowCredentials constructs the allowCredentials array for
// beginAuthentication, including transport hints for each credential.
//
// This enables the browser to show appropriate UI prompts (e.g., "Insert
// USB key" vs "Use Touch ID") based on the credential's transport hints.
func BuildAllowCredentials(
	credentialIDs [][]byte,
	transportsPerCred [][]string,
) []protocol.CredentialDescriptor {
	result := make([]protocol.CredentialDescriptor, 0, len(credentialIDs))

	for i, credID := range credentialIDs {
		var transports []protocol.AuthenticatorTransport
		if i < len(transportsPerCred) {
			for _, t := range transportsPerCred[i] {
				transports = append(transports, protocol.AuthenticatorTransport(t))
			}
		}

		result = append(result, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: credID,
			Transport:    transports,
		})
	}

	return result
}

// TransportLabel generates a human-readable label for display in the
// credential management UI (e.g., "iPhone (Face ID)" or "YubiKey (USB/NFC)").
func TransportLabel(info TransportInfo, authName string) string {
	var parts []string

	if info.IsSynced {
		parts = append(parts, "Synced")
	}

	if info.IsPlatform {
		parts = append(parts, "Platform")
	}
	if info.IsHybrid {
		parts = append(parts, "Hybrid")
	}
	if info.IsRoaming {
		parts = append(parts, "Security Key")
	}

	if authName != "" {
		return fmt.Sprintf("%s (%s)", authName, strings.Join(parts, ", "))
	}
	return strings.Join(parts, ", ")
}
```

---

## 8. Multi-Device Authenticator Management

### 8.1 Tracking Which Devices Have Passkeys

A critical feature for any IAM system is the ability to show users **which devices** have
passkeys registered. This helps users:

- Understand their credential inventory
- Identify stale or unknown credentials (potential compromise indicators)
- Revoke credentials on lost devices

```
┌──────────────────────────────────────────────────────────────────────┐
│       MULTI-DEVICE CREDENTIAL MANAGEMENT UI                          │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │  Your Passkeys                              [+ Add New] │        │
│  ├─────────────────────────────────────────────────────────┤        │
│  │                                                         │        │
│  │  ┌──────────────────────────────────────────────┐      │        │
│  │  │ Chrome on macOS                ⚡ Synced      │      │        │
│  │  │ Transports: internal, hybrid                   │      │        │
│  │  │ Created: Jan 5, 2025  |  Last used: Jan 11    │      │        │
│  │  │                              [Rename] [Revoke]│      │        │
│  │  └──────────────────────────────────────────────┘      │        │
│  │                                                         │        │
│  │  ┌──────────────────────────────────────────────┐      │        │
│  │  │ YubiKey 5 (USB/NFC)            🔒 Device-bound│     │        │
│  │  │ Transports: usb, nfc                           │      │        │
│  │  │ Created: Dec 20, 2024  |  Last used: Jan 10   │      │        │
│  │  │                              [Rename] [Revoke]│      │        │
│  │  └──────────────────────────────────────────────┘      │        │
│  │                                                         │        │
│  │  ┌──────────────────────────────────────────────┐      │        │
│  │  │ Safari on iPhone               ⚡ Synced      │      │        │
│  │  │ Transports: internal, hybrid                   │      │        │
│  │  │ Created: Jan 3, 2025  |  Last used: Jan 11    │      │        │
│  │  │                              [Rename] [Revoke]│      │        │
│  │  └──────────────────────────────────────────────┘      │        │
│  │                                                         │        │
│  └─────────────────────────────────────────────────────────┘        │
│                                                                      │
│  Note: "Synced" badge shown when BE=1 and BS=1                       │
│  "Device-bound" badge shown when BE=0                                │
│  Last used timestamp from credential's LastUsedAt field              │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 8.2 Revoking Passkeys on Lost Devices

When a user loses a device, they should be able to revoke the passkey on that device
through the management UI. For synced passkeys, this is more nuanced:

```
┌──────────────────────────────────────────────────────────────────────┐
│       REVOKING A SYNCED PASSKEY — IMPLICATIONS                      │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Scenario: User loses iPhone with synced passkey                     │
│                                                                      │
│  Option 1: Revoke credential in GGID (RP-level revocation)          │
│  ┌───────────────────────────────────────────────────┐              │
│  │ User logs in from MacBook (still has synced key)  │              │
│  │ → Goes to credential management page               │              │
│  │ → Clicks "Revoke" on the iPhone passkey            │              │
│  │ → GGID deletes the credential from its database    │              │
│  │ → The credential ID is now rejected at auth        │              │
│  │                                                     │              │
│  │ BUT: The key still exists in iCloud Keychain!      │              │
│  │ If the thief recovers the credential somehow,      │              │
│  │ GGID will reject it (credential deleted).           │              │
│  │ This is CORRECT behavior.                          │              │
│  │                                                     │              │
│  │ Problem: The user now has NO passkey!              │              │
│  │ They must re-register from the MacBook.            │              │
│  │ The re-registration creates a NEW credential ID    │              │
│  │ (different from the synced one).                   │              │
│  └───────────────────────────────────────────────────┘              │
│                                                                      │
│  Option 2: Remove from iCloud Keychain (provider-level)             │
│  ┌───────────────────────────────────────────────────┐              │
│  │ User goes to Settings > Passwords > ggid.example   │              │
│  │ → Deletes the passkey from iCloud Keychain         │              │
│  │ → Key removed from ALL synced devices              │              │
│  │ → GGID still has the credential in its DB           │              │
│  │ → If attacker uses old credential → GGID accepts   │              │
│  │   (key is gone from devices but still in DB)       │              │
│  │ This is INCORRECT — must ALSO revoke in GGID.      │              │
│  └───────────────────────────────────────────────────┘              │
│                                                                      │
│  BEST PRACTICE: Revoke in BOTH places                                │
│  1. GGID: delete credential (prevents auth)                          │
│  2. Provider: remove from keychain (prevents local use)             │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 8.3 Per-Device Labels

GGID's `Credential.Name` field (handler.go line 30) already supports human-readable
credential names. The `generateCredentialName` function (handler.go lines 199-238)
auto-generates names from the User-Agent header (e.g., "Chrome on macOS", "Safari on
iOS").

This should be enhanced to:
1. Allow users to rename credentials (add a PATCH endpoint).
2. Include AAGUID-based authenticator names (already available via `LookupAAGUID`).
3. Show sync status (Synced / Device-bound) based on BE/BS flags.

### 8.4 Go Code: Multi-Device Credential Management

```go
// Package credmgmt provides multi-device credential management for WebAuthn.
package credmgmt

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CredentialSummary is the API response model for credential listing.
type CredentialSummary struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Authenticator  string    `json:"authenticator"`      // From AAGUID lookup
	Transports     []string  `json:"transports"`
	IsSynced       bool      `json:"is_synced"`
	IsDeviceBound  bool      `json:"is_device_bound"`
	CreatedAt      time.Time `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at"`
	DaysSinceUsed  int        `json:"days_since_used,omitempty"`
}

// CredentialManager manages multi-device passkey credentials.
type CredentialManager struct {
	store CredentialStore
}

// CredentialStore interface (matches GGID's existing CredentialStore).
type CredentialStore interface {
	GetCredentialsByUser(ctx context.Context, tenantID, userID uuid.UUID) ([]*StoredCredential, error)
	DeleteCredential(ctx context.Context, tenantID uuid.UUID, credID []byte) error
	UpdateCredentialName(ctx context.Context, tenantID uuid.UUID, credID []byte, name string) error
}

// StoredCredential represents the persisted credential.
type StoredCredential struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	UserID         uuid.UUID
	Name           string
	CredentialID   []byte
	Transports     []string
	BackupEligible bool
	BackupState    bool
	UserVerified   bool
	AttestationType string
	AAGUID         []byte
	CreatedAt      time.Time
	LastUsedAt     *time.Time
}

// ListCredentialsWithMetadata returns credential summaries enriched with
// sync status, authenticator name, and usage metadata.
func (m *CredentialManager) ListCredentialsWithMetadata(
	ctx context.Context,
	tenantID, userID uuid.UUID,
) ([]CredentialSummary, error) {
	creds, err := m.store.GetCredentialsByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("get credentials: %w", err)
	}

	result := make([]CredentialSummary, 0, len(creds))
	now := time.Now()

	for _, c := range creds {
		summary := CredentialSummary{
			ID:            c.ID,
			Name:          c.Name,
			Transports:    c.Transports,
			IsSynced:      c.BackupEligible && c.BackupState,
			IsDeviceBound: !c.BackupEligible,
			CreatedAt:     c.CreatedAt,
		}

		if c.LastUsedAt != nil {
			summary.LastUsedAt = c.LastUsedAt
			summary.DaysSinceUsed = int(now.Sub(*c.LastUsedAt).Hours() / 24)
		}

		// Look up authenticator name from AAGUID
		// (uses GGID's existing LookupAAGUID function)
		if info := lookupAAGUID(c.AAGUID); info != nil {
			summary.Authenticator = info.Name
		} else {
			summary.Authenticator = "Unknown"
		}

		result = append(result, summary)
	}

	return result, nil
}

// RevokeCredential deletes a credential. For synced passkeys, this
// prevents authentication even though the key may still exist in the
// sync provider's keychain.
//
// IMPORTANT: This does NOT remove the key from the user's keychain.
// The user should also remove the passkey from their password manager.
// GGID should display a message explaining this after revocation.
func (m *CredentialManager) RevokeCredential(
	ctx context.Context,
	tenantID uuid.UUID,
	credentialID []byte,
) error {
	if err := m.store.DeleteCredential(ctx, tenantID, credentialID); err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}
	return nil
}

// RenameCredential updates a credential's display name.
func (m *CredentialManager) RenameCredential(
	ctx context.Context,
	tenantID uuid.UUID,
	credentialID []byte,
	newName string,
) error {
	if newName == "" {
		return fmt.Errorf("credential name cannot be empty")
	}
	if len(newName) > 100 {
		return fmt.Errorf("credential name too long (max 100 characters)")
	}
	return m.store.UpdateCredentialName(ctx, tenantID, credentialID, newName)
}

// StaleCredentials returns credentials that haven't been used in the
// specified duration. These are candidates for revocation (security
// hygiene: remove credentials the user doesn't actively use).
func (m *CredentialManager) StaleCredentials(
	ctx context.Context,
	tenantID, userID uuid.UUID,
	maxAge time.Duration,
) ([]CredentialSummary, error) {
	creds, err := m.ListCredentialsWithMetadata(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	var stale []CredentialSummary
	for _, c := range creds {
		if c.LastUsedAt == nil {
			// Never used — check creation date
			if time.Since(c.CreatedAt) > maxAge {
				stale = append(stale, c)
			}
		} else if time.Since(*c.LastUsedAt) > maxAge {
			stale = append(stale, c)
		}
	}

	return stale, nil
}

// lookupAAGUID wraps the existing LookupAAGUID for interface compatibility.
type authenticatorInfo struct {
	Name         string
	Manufacturer string
}

func lookupAAGUID(aaguid []byte) *authenticatorInfo {
	// In production: call webauthn.LookupAAGUID(aaguid)
	// For this research code, return nil
	return nil
}
```

---

## 9. GGID WebAuthn Implementation Review

### 9.1 What EXISTS in GGID

The GGID WebAuthn implementation (`services/auth/internal/webauthn/`) is relatively
mature. Here is a detailed inventory of what already exists:

| Feature | Status | Location | Notes |
|---------|--------|----------|-------|
| **Full WebAuthn Registration** | EXISTS | handler.go:443-498 | Uses go-webauthn library with full crypto verification |
| **Full WebAuthn Authentication** | EXISTS | handler.go:589-751 | Assertion verification with sign count check |
| **Credential Store Interface** | EXISTS | handler.go:108-115 | `SaveCredential`, `GetCredentialsByUser`, `DeleteCredential`, etc. |
| **Transports Storage** | EXISTS | handler.go:549-556 | Transports persisted from `credential.Transport` |
| **Transports in Auth** | EXISTS | handler.go:619-630 | Transports sent in `allowCredentials` |
| **Backup Eligible Flag** | EXISTS | handler.go:34,567 | `BackupEligible bool` field stored |
| **Backup State Flag** | EXISTS | handler.go:35,568 | `BackupState bool` field stored |
| **User Verified Flag** | EXISTS | handler.go:36,569 | `UserVerified bool` field stored |
| **AAGUID Storage** | EXISTS | handler.go:37,571 | AAGUID bytes stored per credential |
| **AAGUID Lookup** | EXISTS | attestation.go:99-170 | DB with Touch ID, Windows Hello, Pixel, YubiKey |
| **Attestation Verification** | EXISTS | attestation.go:76-97 | All 7 formats: none, packed, fido-u2f, android-key, android-safetynet, tpm, apple |
| **Clone Detection** | EXISTS | handler.go:727-738 | Sign count monotonicity check |
| **Credential Naming** | EXISTS | handler.go:199-238 | Auto-derived from User-Agent |
| **Credential Listing** | EXISTS | handler.go:756-811 | Returns transports, backup flags |
| **Credential Deletion** | EXISTS | handler.go:813-846 | DELETE endpoint |
| **Discoverable Credentials** | EXISTS | handler.go:638-649 | Supports user-less auth flow |
| **Related Origin Requests** | EXISTS | handler.go:261-275 | `/.well-known/webauthn` endpoint |
| **Android Asset Links** | EXISTS | handler.go:279-301 | `/.well-known/assetlinks.json` |
| **iOS App Site Association** | EXISTS | handler.go:305-330 | `/.well-known/apple-app-site-association` |
| **Error Classification** | EXISTS | handler.go:346-373 | Structured error codes for frontend |
| **Session Store** | EXISTS | handler.go:60-103 | In-memory, 5-minute expiry |
| **Tenant Context** | EXISTS | handler.go:375-400 | X-Tenant-ID header parsing |

### 9.2 What's MISSING in GGID

| Feature | Priority | Effort | Impact |
|---------|----------|--------|--------|
| **Per-tenant sync policy** | P1 | Medium | Critical for enterprise tenants; without it, all tenants accept all passkey types |
| **Authenticator attachment enforcement** | P2 | Low | Should constrain platform/cross-platform at registration based on policy |
| **Backup-eligibility enforcement** | P1 | Low | Should reject BE=1 credentials for tenants requiring device-bound |
| **Credential rename endpoint** | P3 | Low | PATCH endpoint for updating credential name |
| **Conditional mediation support** | P2 | Low | Server already supports discoverable flow; need frontend integration |
| **Multi-device credential display** | P3 | Medium | Show sync status, device labels in credential list |
| **FIDO Metadata Service (MDS)** | P2 | High | Real-time authenticator certification status; improves security posture |
| **Account recovery flow** | P1 | High | No recovery flow exists for users who lose all passkeys |
| **Credential staleness alerts** | P3 | Low | Notify users of unused credentials for hygiene |
| **Admin credential management** | P2 | Medium | Admin API for managing credentials on behalf of users |
| **Session store → Redis** | P2 | Medium | In-memory store doesn't survive restarts or scale horizontally |
| **RP ID hash verification** | P1 | Low | go-webauthn handles this, but should verify BE/BS against policy |

### 9.3 Detailed Analysis: Transport Hints Storage

**Assessment:** Transport hints are properly stored.

GGID's `finishRegistration` correctly extracts and persists transports:

```go
// handler.go lines 549-556 (existing — CORRECT)
var transports []string
for _, t := range credential.Transport {
    transports = append(transports, string(t))
}
if len(transports) == 0 {
    transports = []string{string(credential.Authenticator.Attachment)}
}
```

And `beginAuthentication` correctly sends them back in `allowCredentials`:

```go
// handler.go lines 619-630 (existing — CORRECT)
var allowCreds []protocol.CredentialDescriptor
for _, wc := range user.credentials {
    var transports []protocol.AuthenticatorTransport
    for _, t := range wc.Transport {
        transports = append(transports, t)
    }
    allowCreds = append(allowCreds, protocol.CredentialDescriptor{
        Type:         protocol.PublicKeyCredentialType,
        CredentialID: wc.ID,
        Transport:    transports,
    })
}
```

**Gap:** The fallback when transports is empty uses `credential.Authenticator.Attachment`
(which is either "platform" or "cross-platform") — this is NOT a valid transport value.
It should default to an empty array and let the browser figure out transports.

### 9.4 Detailed Analysis: Per-Tenant Sync Policy

**Assessment:** NOT IMPLEMENTED.

GGID's `beginRegistration` always uses the same `AuthenticatorSelection`:

```go
// handler.go lines 471-474 (existing — NO POLICY ENFORCEMENT)
authSel := protocol.AuthenticatorSelection{
    ResidentKey:      protocol.ResidentKeyRequirementPreferred,
    UserVerification: protocol.VerificationPreferred,
}
```

There is no check for tenant policy. All tenants get the same permissive settings:
- Synced passkeys allowed (no BE=0 enforcement)
- Device-bound passkeys allowed (no BE=1 requirement)
- Platform authenticators allowed
- Cross-platform authenticators allowed
- User verification preferred (not required)

To add per-tenant policy, GGID needs:
1. A `PolicyStore` interface (per Section 4.4 above)
2. Policy lookup in `beginRegistration` → adjust `AuthenticatorSelection`
3. Policy enforcement in `finishRegistration` → check BE/BS/AAGUID against policy

### 9.5 Detailed Analysis: Account Recovery

**Assessment:** NOT IMPLEMENTED.

GGID has no recovery flow for users who lose all passkeys. If a user registers only
passkeys (no password) and loses all devices, they are permanently locked out.

The auth service does not have:
- A recovery endpoint
- Backup code generation/verification
- Social recovery (trusted contacts)
- Admin credential reset API

This is the **most critical gap** for production deployment.

### 9.6 Detailed Analysis: Session Store

**Assessment:** IN-MEMORY ONLY.

```go
// handler.go lines 68-103 (existing — DOES NOT SCALE)
type sessionStore struct {
    mu       sync.Mutex
    sessions map[string]*sessionData
}
```

The in-memory session store means:
- Sessions are lost on server restart (users must re-register/authenticate)
- Does not work with multiple server instances (load-balanced deployments)
- No session persistence across deployments

**Recommendation:** Replace with Redis-backed session store. The `CredentialStore`
interface pattern is well-designed; add a parallel `SessionStore` interface.

### 9.7 Detailed Analysis: Authenticator Attachment

**Assessment:** HARDCODED TO PLATFORM.

In `buildWebAuthnUser` (handler.go line 426), all credentials are set to
`Attachment: protocol.Platform`:

```go
// handler.go line 426 (existing — HARDCODED)
Authenticator: webauthn.Authenticator{
    AAGUID:     c.AAGUID,
    SignCount:  c.Counter,
    Attachment: protocol.Platform, // ← ALL credentials marked as platform!
},
```

This is incorrect for YubiKey and other roaming authenticators. The attachment type
should be derived from the stored transports or the original registration data.

---

## 10. Gap Analysis & Recommendations

### 10.1 Priority Summary

```
┌──────────────────────────────────────────────────────────────────────┐
│                    GAP PRIORITY MATRIX                               │
├──────────┬──────────┬──────────┬─────────────────────────────────────┤
│ Priority │ Effort   │ Risk     │ Action Item                        │
├──────────┼──────────┼──────────┼─────────────────────────────────────┤
│ P0       │ HIGH     │ CRITICAL │ Implement account recovery flow    │
│          │ (2-3 wk) │          │ for passwordless users             │
├──────────┼──────────┼──────────┼─────────────────────────────────────┤
│ P1       │ MEDIUM   │ HIGH     │ Add per-tenant sync policy         │
│          │ (1-2 wk) │          │ (synced/device-bound enforcement)  │
├──────────┼──────────┼──────────┼─────────────────────────────────────┤
│ P1       │ LOW      │ MEDIUM  │ Fix authenticator attachment bug    │
│          │ (1 day)  │          │ (hardcoded to Platform)            │
├──────────┼──────────┼──────────┼─────────────────────────────────────┤
│ P2       │ MEDIUM   │ MEDIUM  │ Replace in-memory session store     │
│          │ (3-5 d)  │          │ with Redis                         │
├──────────┼──────────┼──────────┼─────────────────────────────────────┤
│ P2       │ HIGH     │ LOW      │ Integrate FIDO Metadata Service    │
│          │ (2-3 wk) │          │ for authenticator certification    │
├──────────┼──────────┼──────────┼─────────────────────────────────────┤
│ P3       │ LOW      │ LOW      │ Add credential rename endpoint     │
│          │ (1 day)  │          │ and stale credential alerts        │
└──────────┴──────────┴──────────┴─────────────────────────────────────┘
```

### 10.2 Action Item 1: Account Recovery Flow (P0)

**Problem:** Passwordless users who lose all passkeys are permanently locked out.

**Solution:** Implement a multi-factor recovery flow:

1. **Admin Reset (Enterprise):** Admin can revoke all credentials and issue a one-time
   registration link. Requires admin identity verification.
2. **Backup Codes (Consumer):** Generate 10 one-time codes at passkey registration.
   User stores them securely. Each code can be used once to re-register.
3. **Email Magic Link (Fallback):** Last-resort recovery via email with mandatory
   24-hour delay for enterprise tenants, 15-minute delay for consumer tenants.
4. **Secondary Passkey:** Encourage users to register a second passkey on a different
   device as a recovery factor.

**Effort:** 2-3 weeks
**Files to create/modify:**
- `services/auth/internal/recovery/recovery.go` — Recovery logic
- `services/auth/internal/recovery/handler.go` — Recovery HTTP endpoints
- `services/auth/internal/recovery/store.go` — Recovery state storage
- Database migration for recovery codes table

### 10.3 Action Item 2: Per-Tenant Sync Policy (P1)

**Problem:** All tenants accept all passkey types. No way to require device-bound or
synced-only passkeys.

**Solution:** Implement the `TenantPolicy` and `PolicyStore` interfaces from Section 4.4.

```go
// Integration point: in handler.go beginRegistration
// Replace lines 470-474 with:
policy := h.policyStore.GetPolicy(ctx, tenantID)
authSel := ConfigureAuthenticatorSelection(policy)
// Add policy check in finishRegistration after CreateCredential:
// if err := EvaluateRegistration(policy, credential.Flags.BackupEligible,
//   credential.Authenticator.Attachment, formatAAGUID(credential.Authenticator.AAGUID));
//   err != nil { reject }
```

**Effort:** 1-2 weeks
**Files to create/modify:**
- `services/auth/internal/webauthn/policy.go` — Policy types and enforcement
- `services/auth/internal/webauthn/handler.go` — Wire policy into begin/finish
- Database migration for tenant webauthn policy table

### 10.4 Action Item 3: Fix Authenticator Attachment Bug (P1)

**Problem:** All stored credentials are hardcoded as `Attachment: protocol.Platform`
(handler.go line 426). This is incorrect for roaming authenticators (YubiKey).

**Solution:** Store the attachment type at registration and use it at authentication.

```go
// In Credential struct, add:
Attachment string // "platform" or "cross-platform"

// In finishRegistration, store:
cred.Attachment = string(credential.Authenticator.Attachment)

// In buildWebAuthnUser, use:
Attachment: protocol.AuthenticatorAttachment(c.Attachment),
```

**Effort:** 1 day
**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — Add Attachment field, fix buildWebAuthnUser

### 10.5 Action Item 4: Redis Session Store (P2)

**Problem:** In-memory session store doesn't survive restarts or scale horizontally.

**Solution:** Implement a Redis-backed session store implementing the same interface.

**Effort:** 3-5 days
**Files to create/modify:**
- `services/auth/internal/webauthn/redis_session.go` — Redis implementation
- `services/auth/internal/webauthn/handler.go` — Accept SessionStore option

### 10.6 Action Item 5: FIDO Metadata Service Integration (P2)

**Problem:** GGID has a static AAGUID database (5 entries). It doesn't know about
authenticator certification status (FIDO L1/L2/L3), known vulnerabilities, or newly
released authenticators.

**Solution:** Integrate the FIDO Metadata Service (MDS3) BLOB:

1. Fetch the MDS3 BLOB from `https://mds3.fidoalliance.org/` periodically.
2. Parse the BLOB (JWT containing metadata for all FIDO-certified authenticators).
3. Use metadata to enforce policy (e.g., "only accept L2+ certified authenticators").
4. Alert on revoked or compromised authenticator models.

**Effort:** 2-3 weeks
**Files to create/modify:**
- `services/auth/internal/webauthn/mds.go` — MDS3 BLOB fetcher and parser
- `services/auth/internal/webauthn/attestation.go` — Use MDS for AAGUID lookup

### 10.7 Summary of Implementation Effort

```
Total estimated effort: 7-10 weeks
Critical path: Account Recovery (P0) → Per-Tenant Policy (P1) → Bug fixes (P1)

Suggested implementation order:
Week 1-3:   Account Recovery Flow (P0)
Week 4-5:   Per-Tenant Sync Policy (P1) + Attachment Bug Fix (P1)
Week 6-7:   Redis Session Store (P2)
Week 8-10:  FIDO MDS Integration (P2)

Quick wins (can be done anytime):
- Attachment bug fix: 1 day
- Credential rename endpoint: 1 day
- Credential staleness alerts: 1 day
```

---

## 11. References

### 11.1 Specifications

| Spec | Title | URL |
|------|-------|-----|
| W3C | Web Authentication: An API for accessing Public Key Credentials - Level 3 | https://www.w3.org/TR/webauthn-3/ |
| FIDO | Client to Authenticator Protocol (CTAP) 2.1 | https://fidoalliance.org/specs/fido-v2.1-ps-20210615/fido-client-to-authenticator-protocol-v2.1-ps-20210615.html |
| FIDO | WebAuthn Level 2 | https://www.w3.org/TR/webauthn-2/ |
| RFC 8809 | WebAuthn Attestation Statement Format and Extension Identifiers | https://tools.ietf.org/html/rfc8809 |

### 11.2 Platform Security Documentation

| Platform | Document | URL |
|----------|----------|-----|
| Apple | Apple Platform Security Guide | https://support.apple.com/guide/security/welcome/web |
| Google | Google Password Manager Security | https://security.googleblog.com/ |
| Microsoft | Windows Hello for Business | https://learn.microsoft.com/en-us/windows/security/identity-protection/hello-for-business/ |
| FIDO | FIDO Alliance Whitepaper: Multi-Device FIDO Credentials | https://fidoalliance.org/white-paper-multi-device-fido-credentials/ |

### 11.3 NIST and Compliance

| Standard | Document | URL |
|----------|----------|-----|
| NIST SP 800-63B | Digital Identity Guidelines: Authentication and Lifecycle Management (Rev. 3) | https://pages.nist.gov/800-63-3/sp800-63b.html |
| FIDO | FIDO Certified Products Directory | https://fidoalliance.org/certification/fido-certified-products/ |
| FIDO | FIDO Metadata Service 3 (MDS3) | https://fidoalliance.org/metadata/ |

### 11.4 Research Papers and Articles

1. "Security Analysis of FIDO2 Passkeys" — FIDO Alliance (2023)
2. "Synced vs Device-Bound Credentials: A Security Comparison" — Duo Labs (2023)
3. "Account Recovery: The Achilles Heel of Passwordless" — Microsoft Identity Blog (2024)
4. "Conditional UI and Autofill: The Missing UX Piece" — Chrome Developers Blog (2023)
5. "WebAuthn Transport Hints: A Practical Guide" — web.dev (2024)

### 11.5 GGID Source Files Referenced

| File | Purpose |
|------|---------|
| `services/auth/internal/webauthn/handler.go` | Main WebAuthn HTTP handler (registration, auth, credential management) |
| `services/auth/internal/webauthn/attestation.go` | Attestation format verification and AAGUID lookup |
| `services/auth/internal/webauthn/attestation_formats.go` | Individual attestation format verifiers (fido-u2f, android-key, android-safetynet, tpm, apple) |
| `services/auth/internal/webauthn/handler_test.go` | Handler unit tests |
| `services/auth/internal/webauthn/handler_coverage_test.go` | Coverage tests |
| `services/auth/internal/webauthn/attestation_test.go` | Attestation verification tests |

---

## Appendix A: BE/BS Flag Reference

The `Backup Eligible (BE)` and `Backup State (BS)` flags were introduced in the WebAuthn
Level 2 specification to signal whether an authenticator's credential can be backed up
(synced) and whether it currently is.

| Flag | Meaning | When Set |
|------|---------|----------|
| BE=0 | Credential CANNOT be synced | YubiKey, TPM-only Windows Hello |
| BE=1, BS=0 | Credential CAN be synced but hasn't yet | Newly created passkey, sync pending |
| BE=1, BS=1 | Credential CAN be synced and IS synced | iCloud Keychain passkey, Google Password Manager passkey |

These flags are part of the authenticator data flags byte:

```
Bit 0: User Present (UP)
Bit 2: User Verified (UV)
Bit 3: Backup Eligible (BE)
Bit 4: Backup State (BS)
Bit 6: Attested Credential Data (AT)
Bit 7: Extension Data (ED)
```

GGID correctly stores BE and BS in the `Credential` struct (handler.go lines 34-35) and
displays them in the credential list API (handler.go lines 805-806).

---

## Appendix B: Authenticator Certification Levels

FIDO Alliance certifies authenticators at different assurance levels:

| Level | Name | Requirements | Example |
|-------|------|-------------|---------|
| L1 | Functional | Basic FIDO2 compliance | Software authenticators |
| L2 | Security | Security testing, side-channel resistance | YubiKey 5, iPhone (Touch ID) |
| L3 | Enhanced | Hardware security boundary, physical attack resistance | (few authenticators achieve this) |

For enterprise deployments requiring the highest assurance, the tenant policy should
restrict accepted authenticators to L2+ certified models, verified through the FIDO
Metadata Service.

---

## Appendix C: Conditional Mediation API

Conditional mediation enables passkey autofill in username/password fields. The browser
API is:

```javascript
// Check if conditional mediation is supported
if (PublicKeyCredential.isConditionalMediationAvailable &&
    await PublicKeyCredential.isConditionalMediationAvailable()) {

  // Start the WebAuthn flow with conditional mediation
  const credential = await navigator.credentials.get({
    mediation: "conditional",
    publicKey: {
      challenge: challengeBytes,
      rpId: "ggid.example.com",
      userVerification: "preferred",
      // No allowCredentials → discoverable credential flow
    }
  });

  // Send credential to server for verification
  await fetch("/api/v1/webauthn/auth/finish", {
    method: "POST",
    body: JSON.stringify(credential),
  });
}
```

Server-side: GGID's `beginAuthentication` endpoint already supports the discoverable
credential flow (no `user_id` parameter → ephemeral user → `finishAuthentication` looks
up the credential by `RawID` in the assertion).

---

## Appendix D: Clone Detection in Synced vs Device-Bound Credentials

```go
// Package clonedetect provides clone detection logic for WebAuthn credentials.
package clonedetect

// CloneDetectionResult describes the outcome of clone detection analysis.
type CloneDetectionResult struct {
	Detected     bool   // True if a potential clone was detected
	Reason       string // Explanation
	Confidence   string // "high", "medium", "low"
	Recommendation string // Action to take
}

// CheckClone analyzes the sign counter for potential credential cloning.
//
// For device-bound credentials:
// - The counter is monotonically increasing per-authenticator.
// - If received counter <= stored counter, this is HIGH confidence clone detection.
//
// For synced credentials:
// - The sync provider coordinates counters across devices.
// - If received counter <= stored counter, it could be:
//   a) A genuine clone attack (low probability)
//   b) A sync race condition where two devices authenticated concurrently
//   c) A stale sync state where the counter wasn't properly coordinated
// - Confidence is MEDIUM at best.
func CheckClone(
	receivedCounter uint32,
	storedCounter uint32,
	isSynced bool,
) CloneDetectionResult {
	// If stored counter is 0, we can't detect clones
	// (many synced passkeys start with counter=0 and stay at 0)
	if storedCounter == 0 {
		return CloneDetectionResult{
			Detected:       false,
			Reason:         "stored counter is 0; clone detection not possible for this credential",
			Confidence:     "none",
			Recommendation: "enable rate limiting and geo-anomaly detection instead",
		}
	}

	if receivedCounter > storedCounter {
		return CloneDetectionResult{
			Detected:       false,
			Reason:         "counter increased normally",
			Confidence:     "none",
			Recommendation: "no action needed",
		}
	}

	// Counter did not increase — potential clone
	if isSynced {
		return CloneDetectionResult{
			Detected:       true,
			Reason:         "counter did not increase; possible sync race or clone attack",
			Confidence:     "medium",
			Recommendation: "log warning; require step-up auth on next login; " +
				"do NOT auto-revoke (sync race conditions are common)",
		}
	}

	// Device-bound: high confidence
	return CloneDetectionResult{
		Detected:       true,
		Reason:         "counter did not increase for device-bound credential; " +
			"high-confidence clone detection",
		Confidence:     "high",
		Recommendation: "revoke credential immediately; alert user and admin; " +
			"require re-registration",
	}
}
```

---

## Appendix E: WebAuthn Authenticator Selection Criteria Reference

The `authenticatorSelection` criteria sent by the RP during registration control which
authenticators the browser will offer:

| Field | Values | Effect |
|-------|--------|--------|
| `authenticatorAttachment` | `platform`, `cross-platform`, (unset=any) | Limits to built-in or roaming authenticators |
| `residentKey` | `required`, `preferred`, `discouraged` | Controls whether credential is discoverable |
| `userVerification` | `required`, `preferred`, `discouraged` | Controls biometric/PIN requirement |
| `requireResidentKey` | (deprecated, use `residentKey`) | Legacy field |

For synced passkey support:
- Set `residentKey: "preferred"` (synced passkeys are always discoverable)
- Set `userVerification: "preferred"` or `"required"` (synced passkeys always support UV)
- Leave `authenticatorAttachment` unset (allow both platform and cross-platform) for
  maximum flexibility, OR set to `platform` if you only want synced-capable authenticators

For device-bound passkey enforcement:
- Set `residentKey: "required"` (device-bound passkeys are discoverable)
- Set `userVerification: "required"` (enterprise requirement)
- Set `authenticatorAttachment: "cross-platform"` (forces USB/NFC security keys)
- Additionally: check BE flag at `finishRegistration` and reject if BE=1

---

*End of Document*
