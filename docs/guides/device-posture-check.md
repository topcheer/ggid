# Device Posture Check Guide

This guide covers device posture checking in GGID — posture rules per platform, compliance thresholds, jailbreak detection, auto-remediation, and enforcement.

## Overview

Device posture verifies that the device accessing GGID meets security requirements (OS version, encryption, screen lock, no jailbreak) before granting access.

## Posture Rules by Platform

### iOS / iPadOS

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| OS version | >= latest - 1 | Warn → Block after 30d |
| Passcode | 6+ digits | Block |
| Biometric | Enabled (Face ID/Touch ID) | Warn |
| Jailbreak | Not detected | Block |
| Screen lock | <= 5 min | Warn |
| Disk encryption | Enabled (default) | Warn |
| MDM enrollment | Required (enterprise) | Block |

### Android

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| OS version | >= latest - 2 | Warn → Block after 30d |
| Screen lock | Required | Block |
| Root | Not detected | Block |
| Play Protect | Enabled | Warn |
| Encryption | Enabled | Block |
| MDM/Work Profile | Required (enterprise) | Block |
| USB debugging | Disabled | Warn |

### macOS

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| OS version | >= latest - 1 | Warn |
| FileVault | Enabled | Block |
| Screen lock | <= 10 min | Warn |
| Firewall | Enabled | Warn |
| Gatekeeper | Enabled | Block |
| MDM enrollment | Required (enterprise) | Block |

### Windows

| Rule | Requirement | Enforcement |
|------|-------------|-------------|
| OS build | Supported (Win 10/11) | Warn → Block |
| BitLocker | Enabled | Block |
| Antivirus | Active (Defender/3rd party) | Warn |
| Firewall | Enabled | Block |
| Screen lock | <= 10 min | Warn |
| MDM/Intune | Required (enterprise) | Block |
| TPM 2.0 | Present | Warn |

## Compliance Thresholds

```yaml
device_posture:
  min_compliance_score: 70  # Block if below 70%
  enforcement_mode: enforce  # enforce | monitor | disabled
  grace_period_days: 7      # Warn before blocking
```

### Scoring

| Category | Weight |
|----------|--------|
| Jailbreak/Root | Critical (0 = fail) |
| Encryption | 30 points |
| OS version | 20 points |
| Screen lock | 15 points |
| MDM enrollment | 20 points |
| Firewall/AV | 15 points |

## Jailbreak/Root Detection

### iOS Jailbreak

| Signal | Detection |
|--------|-----------|
| Cydia installed | Check file existence |
| SSH daemon | Port 22 open |
| /Applications writable | Filesystem check |
| App Sandbox broken | Sandbox test fail |

### Android Root

| Signal | Detection |
|--------|-----------|
| su binary | `/system/bin/su` or `/system/xbin/su` |
| Magisk | Magisk package detected |
| SELinux permissive | `getenforce` returns `Permissive` |
| Test-keys build | `Build.TAGS = test-keys` |

## Enforcement Points

### At Authentication

```
Login → Device fingerprint check → Posture evaluation
  ↓
  Compliant → Issue JWT (normal)
  Non-compliant → Step-up: require MFA or block
  Unknown device → Require device enrollment
```
### At API Gateway

```
Request → JWT verified → Device posture claim checked
  ↓
  Posture valid → Allow
  Posture expired → Step-up (re-attest)
  Posture revoked → Block (403)
```

## Configuration

```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/device-posture \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "enforcement": "enforce",
    "platforms": {
      "ios": {"min_os": "17.0", "require_passcode": true},
      "android": {"min_os": "13.0", "require_encryption": true},
      "macos": {"require_filevault": true, "require_mdm": true},
      "windows": {"require_bitlocker": true, "require_firewall": true}
    },
    "grace_period_days": 7
  }'
```

## Auto-Remediation

For compliant MDM-managed devices, GGID can trigger auto-remediation:

| Issue | Auto-Remediation |
|-------|----------------|
| Screen lock disabled | MDM push config |
| Firewall disabled | MDM enable firewall |
| OS outdated | Notify user to update |
| Encryption off | MDM enable + reboot prompt |

## See Also

- [Adaptive Authentication](../research/adaptive-authentication.md)
- [Zero Trust Architecture](../research/zero-trust-architecture.md)
- [Fraud Detection](fraud-detection.md)
