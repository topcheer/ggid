# Passkey / FIDO2 Enterprise Deployment at Scale

> Practical deployment guide for rolling out passkeys across thousands of users and
> heterogeneous device fleets. This is **not** a WebAuthn spec deep-dive — see the
> existing FIDO/WebAuthn research docs for protocol internals and attestation formats.

---

## 1. Overview

Deploying passkeys at enterprise scale means supporting **thousands of users** across
multiple device types — managed laptops, BYOD phones, hardware security keys, and
shared kiosks. The core challenges are not cryptographic; they are operational:

- **Device diversity**: macOS Touch ID, Windows Hello, Android biometrics, YubiKeys —
  each behaves differently during enrollment and authentication.
- **BYOD vs managed**: users bring personal devices whose sync accounts you don't control.
- **Recoverability**: when a device is lost, users must regain access quickly without
  compromising security.
- **Policy enforcement**: which authenticators are allowed, who can enroll, what
  attestation level is required — all per-tenant.

The goal is to **maximize passkey adoption** (reducing password-reset tickets and
phishing risk) while maintaining security and a manageable support load. This document
provides the rollout strategy, policy configuration, checklists, and GGID-specific
implementation guidance needed to get there.

---

## 2. Platform Authenticator Sync Management

Synced passkeys (Apple iCloud Keychain, Google Password Manager, Microsoft) eliminate
device-loss lockout but introduce a policy question: **do you allow credentials to leave
the device?**

### Apple iCloud Keychain
- **Sync scope**: iPhone, iPad, Mac sharing the same Apple ID.
- **Security model**: end-to-end encrypted; device passcode is the recovery key.
- **Enterprise concern**: personal Apple ID vs Managed Apple ID. With ABM/ASM, Managed
  Apple IDs can sync within the org; personal IDs are user-controlled.
- **MDM posture**: `allowAccountModification` can restrict iCloud sign-in on supervised
  devices. Apple does not provide a policy to block *only* Keychain sync while allowing
  other iCloud services.
- **Recommendation**: allow sync by default (dramatically better UX, still E2E
  encrypted). Restrict to device-bound keys only for classified/air-gapped environments.

### Google Password Manager
- **Sync scope**: Android devices + Chrome on any OS sharing a Google account.
- **Security model**: E2E encrypted via Google account; screen lock required on mobile.
- **Enterprise**: Google Workspace accounts sync within the org domain; admins can
  enable/disable passkey sync via Workspace policy.
- **BYOD**: personal Google accounts sync passkeys to the user's personal ecosystem —
  you cannot prevent this and should not try.

### Microsoft Authenticator / Entra ID
- **Entra ID FIDO2**: passkeys are managed server-side in Entra; policies control
  AAGUID allowlists, key restrictions, and enforcement.
- **Authenticator app backup**: newer feature; passkeys backed up to the user's
  Microsoft account with E2E encryption.
- **Windows Hello**: device-bound by default (not synced), though sync is expanding.

### Sync Policy Decision Matrix

| Scenario | Allow Sync? | Rationale |
|---|---|---|
| Corporate device + corp account | Yes | Better UX; still E2E encrypted |
| BYOD + personal account | Yes | Can't prevent; sync reduces lockout risk |
| Classified / regulated environment | No | Device-bound only; no credential leaves device |
| Shared kiosk | No | Temporary credential; no sync, auto-expiring |

> **Key insight**: synced passkeys report `backup_eligible = true` and
> `backup_state = true`. GGID already stores these flags on each `Credential` — use
> them to report sync status in the admin dashboard, not to block enrollment.

---

## 3. AAID / AAGUID Filtering

The relying party can restrict **which authenticator models** are accepted, giving
enterprises control over their hardware ecosystem.

### What to Filter
- **AAID** (FIDO U2F, legacy): identifies an authenticator type by key handle length
  and attestation root.
- **AAGUID** (FIDO2): a 128-bit identifier unique to each authenticator *model*
  (e.g., all YubiKey 5 NFC devices share one AAGUID).
- The RP receives the AAGUID in the attestation object at registration and can accept
  or reject it before persisting the credential.

### Policy Levels
| Level | Behavior | Use Case |
|---|---|---|
| `open` | Accept all authenticators | Consumer / default |
| `certified` | Accept only FIDO-certified (check FIDO MDS) | Standard enterprise |
| `allowlist` | Accept only specified AAGUIDs | Regulated / locked-down |
| `blocklist` | Accept all except specified AAGUIDs | Revoked / compromised devices |

Use the [FIDO Metadata Service (MDS)](https://fidoalliance.org/metadata/) to discover
AAGUIDs and certification status for each authenticator model, and to detect when a
model has been revoked.

### Enterprise Allowlist Example
A regulated org standardizes on YubiKey 5 series + Windows Hello:

```json
{
  "policy": "allowlist",
  "aaguids": [
    "cb69481e-8ff7-4039-93ec-0a2729a154a8",
    "08987058-cadc-49b2-ab47-67039c501e03",
    "9ddd1817-af5a-4672-a2b9-3e3dd95000a9"
  ],
  "exclude": [
    "00000000-0000-0000-0000-000000000000"
  ]
}
```

### GGID Implementation

GGID's `Credential` struct already stores `AAGUID []byte`. Add a per-tenant
authenticator policy and check it during `finishRegistration`:

```go
// AuthenticatorPolicy defines which authenticators a tenant accepts.
type AuthenticatorPolicy struct {
    Mode    string   `json:"policy"`          // open | certified | allowlist | blocklist
    AAGUIDs []string `json:"aaguids"`         // allowlist entries (hex-encoded AAGUIDs)
    Exclude []string `json:"exclude"`         // blocklist entries
}

// ShouldAccept returns true if the given AAGUID is acceptable under this policy.
func (p AuthenticatorPolicy) ShouldAccept(aaguid []byte) bool {
    id := fmt.Sprintf("%x", aaguid)
    switch p.Mode {
    case "", "open":
        return true
    case "blocklist":
        for _, ex := range p.Exclude {
            if ex == id { return false }
        }
        return true
    case "allowlist":
        for _, ok := range p.AAGUIDs {
            if ok == id { return true }
        }
        return false
    case "certified":
        return mds.IsCertified(aaguid) // integrate FIDO MDS blob
    }
    return true // fail-open default
}
```

Wire this into `finishRegistration` right after `h.wbn.CreateCredential` succeeds:

```go
// After verifying attestation, enforce authenticator policy.
if !policy.ShouldAccept(credential.Authenticator.AAGUID) {
    writeError(w, http.StatusForbidden, "authenticator not permitted by tenant policy")
    return
}
```

---

## 4. Enterprise Policy Controls

Beyond AAGUID filtering, enterprise deployment needs controls over authenticator type,
user verification, attestation, and enrollment limits.

### Authenticator Attachment Policy
| Value | Examples | Enterprise Use |
|---|---|---|
| `platform` | Touch ID, Face ID, Windows Hello | Passkey-first consumer UX |
| `cross-platform` | YubiKey, Google Titan | Hardware-key requirement (regulated) |

### User Verification Policy
| Value | Behavior | Recommendation |
|---|---|---|
| `required` | Biometric/PIN on every auth | Admin / privileged access |
| `preferred` | Biometric when available, fallback OK | Standard users |
| `discouraged` | Minimize prompts | Testing / low-security |

> GGID currently hardcodes `UserVerification: protocol.VerificationPreferred` (line 473).
> Phase 5 makes this per-tenant.

### Attestation Policy
| Value | Behavior | Use Case |
|---|---|---|
| `none` | No device verification (privacy-first) | Consumer default |
| `direct` | Full attestation returned to RP | Enterprise device verification |
| `enterprise` | RP receives authenticator info for inventory | Regulated industries |

> GGID currently uses the go-webauthn default (no explicit attestation conveyance =
> `none`). Enterprises needing `direct` pass it as a `RegistrationOption`.

### Enrollment Policy
- **Max credentials per user**: cap at 10 to prevent abuse (GGID has no limit today).
- **Re-auth before enrollment**: require a valid session or step-up auth before allowing
  new credential registration — prevents session-hijack → credential injection.
- **Admin approval mode**: new passkey requires admin sign-off before activation (high
  security / VIP accounts).

### Go Implementation Sketch

```go
type WebAuthnPolicy struct {
    Attachment       string `json:"attachment"`         // platform | cross-platform | ""
    UserVerification string `json:"user_verification"`   // required | preferred | discouraged
    Attestation      string `json:"attestation"`         // none | direct | enterprise
    MaxCredentials   int    `json:"max_credentials"`     // 0 = unlimited
    RequireReAuth    bool   `json:"require_reauth"`
    AdminApproval    bool   `json:"admin_approval"`
}
```

Load per-tenant at registration time and translate to `webauthn.RegistrationOption`
values. GGID already builds `AuthenticatorSelection` — extend it with policy-driven
fields:

```go
authSel := protocol.AuthenticatorSelection{
    ResidentKey:      protocol.ResidentKeyRequirementPreferred,
    UserVerification: policy.UserVerificationPreference(),
    AuthenticatorAttachment: policy.AttachmentValue(),
}
```

---

## 5. Recoverability Planning

Device loss is the #1 operational risk in passkey deployment. Plan for it before rollout.

### Device Loss Scenarios
| Scenario | Impact | Resolution |
|---|---|---|
| Corporate device lost (synced) | Low | Replacement auto-restores synced passkeys |
| Device-bound key lost | High | User locked out; needs fallback auth |
| Personal device lost (BYOD) | Medium | User recovers via personal sync account |
| All devices lost | Critical | Admin-assisted recovery required |

### Fallback Chain (priority order)
1. **Synced passkey** on another device — automatic if synced, zero support cost.
2. **Recovery code** — user-generated at enrollment, stored offline.
3. **TOTP backup** — if the user enrolled a TOTP authenticator before passkey.
4. **Admin-assisted recovery** — admin issues a temporary credential after identity
   verification.
5. **Password fallback** — if password auth is still enabled (recommended for first year).

### Enterprise Recovery SLA
- **Target**: user recovered within **15 minutes** of reporting device loss.
- **Process**: user calls helpdesk -> helpdesk verifies identity (manager callback or
  knowledge-based verification) -> admin removes the lost credential -> user re-enrolls
  on a replacement device.
- **Audit**: all recovery events logged with actor, timestamp, and verification method
  for compliance (SOC 2, ISO 27001).

### Preventive Measures
- Encourage **multi-credential enrollment** (passkey + TOTP, or two passkeys on
  different devices).
- Require recovery codes for **passkey-only accounts** (password disabled).
- Run **quarterly recovery drills** — test the fallback chain with a subset of users.

---

## 6. Deployment Checklist

### Pre-Deployment
- [ ] WebAuthn RP ID set to the registrable domain (e.g., `example.com`, not
      `https://app.example.com/login`)
- [ ] HTTPS endpoints configured (WebAuthn requires secure context)
- [ ] Session store moved to Redis (GGID uses in-memory `sessionStore` — replace for
      production multi-instance)
- [ ] FIDO MDS blob fetched and cached for AAGUID lookups
- [ ] Per-tenant authenticator policy defined (open vs allowlist)
- [ ] Recovery code system implemented and tested
- [ ] User education materials: "What is a passkey?" one-pager, enrollment screenshots

### Pilot Phase (10-20 users)
- [ ] Enable for IT/admin team first
- [ ] Monitor: enrollment success rate, auth success rate, support tickets
- [ ] Test: cross-device authentication, device-loss recovery, sync behavior
- [ ] Collect feedback: UX friction points, confusion, browser compatibility issues

### Rollout Phase (all users, opt-in)
- [ ] Enable passkey enrollment for all users — **opt-in, not mandatory**
- [ ] Track: passkey adoption rate, auth success rate, password usage decline
- [ ] Train helpdesk: passkey enrollment assistance, recovery procedures
- [ ] Email campaign: explain passkey benefits with step-by-step enrollment guide

### Post-Deployment
- [ ] Monitor authenticator diversity (which AAGUIDs users actually have)
- [ ] Re-check FIDO MDS quarterly for newly revoked authenticators
- [ ] Measure support ticket reduction (password-reset tickets should drop)
- [ ] Evaluate: disable password auth for passkey-only power users

---

## 7. Metrics and Success Criteria

### Key Metrics
| Metric | Target | How to Measure |
|---|---|---|
| Enrollment rate | >70% in 6 months | `credentials / total_users` per tenant |
| Auth success rate | Passkey > password | Compare finish-auth 200 rate |
| Average auth time | Passkey < password | Client-side timing telemetry |
| Recovery rate | <2% of passkey users | Count admin-assisted recoveries |
| Password-reset tickets | Decrease 40%+ | Compare pre/post deployment |

### GGID Dashboard
- **Tenant-level**: enrollment rate, auth success rate, device/AAGUID breakdown,
  synced vs device-bound ratio (from `backup_state`).
- **Global**: cross-tenant adoption trend, top authenticator models.
- **Alerts**: enrollment rate drops below threshold, recovery rate spikes above 2%,
  new AAGUID appears outside allowlist.

GGID already exposes `backup_eligible` and `backup_state` in the credential list
endpoint (`/api/v1/webauthn/credentials`) — use these for the sync-status dashboard.

---

## 8. Common Pitfalls

1. **Forcing passkey enrollment** — mandatory on day one frustrates users who hit
   browser/OS incompatibilities. Always start opt-in.
2. **No recovery fallback** — users locked out flood the helpdesk. Always have
   TOTP or password fallback during the transition period.
3. **Wrong RP ID** — using `https://app.example.com` instead of `example.com`.
   The browser will reject the ceremony silently, and debugging is painful.
4. **Rejecting "none" attestation** — consumer platform authenticators (Face ID,
   Windows Hello) often return `attestation: none`. If you require attestation, you
   block the majority of platform passkeys. Use `none` as default; enforce `direct`
   only for hardware keys in regulated environments.
5. **Ignoring sync status** — treating a synced passkey as device-bound leads to
   wrong recovery procedures. Check `backup_state` before advising users.
6. **No user education** — users don't know what a passkey is and skip enrollment.
   A 30-second explainer at the enrollment prompt increases adoption 3-5x.

---

## 9. GGID Deployment Roadmap

| Phase | Scope | Status | Effort |
|---|---|---|---|
| 1 | Basic passkey enrollment + auth | **Done** | - |
| 2 | Per-tenant AAGUID filtering | Planned | 3-5 days |
| 3 | Recovery code system | Planned | 3-5 days |
| 4 | Deployment metrics dashboard | Planned | 3-5 days |
| 5 | Enterprise policy controls (attachment, UV, attestation, limits) | Planned | 5-7 days |

**Total effort for Phase 2-5**: approximately 2-3 weeks of focused development.

### Current GGID WebAuthn Capabilities (from `handler.go`)
- Full registration + authentication with go-webauthn cryptographic verification
- Credential exclusion (prevents duplicate enrollment on same authenticator)
- Clone detection via sign-count monotonicity
- Backup eligibility/state tracking (synced vs device-bound)
- Related Origin Requests (ROR) support for multi-origin RPs
- Android Digital Asset Links + iOS Universal Links (mobile app integration)
- Auto-generated credential names from User-Agent
- Discoverable credential login (server-side resident key support)

### Gaps to Close for Enterprise
- In-memory session store -> Redis-backed for horizontal scaling
- Hardcoded `UserVerification: preferred` -> per-tenant policy
- No AAGUID filtering at registration -> add `AuthenticatorPolicy`
- No max-credential limit -> add enrollment guard
- No attestation conveyance config -> expose as policy option

---

*See also: [passkey-recovery-architecture.md](./passkey-recovery-architecture.md),
[webauthn-implementation-guide.md](./webauthn-implementation-guide.md),
[fido2-attestation-formats.md](./fido2-attestation-formats.md).*
