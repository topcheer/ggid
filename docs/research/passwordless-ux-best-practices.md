# Passwordless UX Best Practices

> Research document for GGID IAM Console passwordless authentication design.
> Covers WebAuthn conditional mediation, passkey enrollment UX, cross-device
> hybrid transport, platform ecosystem practices, and conversion data.

---

## 1. Overview

Passwordless authentication eliminates the password as a user-facing credential
while improving security posture. The core insight is that passwords are both
the weakest security link and the largest source of user friction.

**Key technologies:**

- **WebAuthn / Passkeys** — public-key credentials stored on the user's device,
  authenticated via biometrics (Face ID, Touch ID, Windows Hello) or device PIN.
- **Conditional Mediation (Autofill)** — the browser surfaces passkeys inside
  the username input's autofill dropdown, blending with saved passwords.
- **Hybrid Transport** — cross-device auth via QR code + BLE proximity,
  allowing a phone passkey to authenticate a desktop session.

**Business case:**

- **Support cost reduction** — password resets are the #1 help-desk ticket
  category, typically 20-50% of all IT support volume.
- **Conversion improvement** — passwordless sign-in reduces abandonment at
  login and registration checkpoints.
- **Credential stuffing elimination** — passkeys are phishing-resistant and
  cannot be replayed, eliminating the entire class of automated attacks.
- **Compliance** — NIST 800-63B explicitly recommends verifier impersonation
  resistance, which WebAuthn provides by design.

As of December 2024, over 15 billion online accounts can leverage passkeys
(FIDO Alliance), doubling from the prior year.

---

## 2. WebAuthn Autofill / Conditional Mediation

Conditional mediation is the single most impactful UX improvement for
passwordless login. Instead of requiring a separate "Sign in with passkey"
button, passkeys appear directly in the browser's autofill dropdown alongside
saved passwords.

**How it works:**

1. The username `<input>` has `autocomplete="username webauthn"`.
2. JavaScript calls `navigator.credentials.get({ mediation: "conditional" })`
   on page load — this is non-blocking.
3. When the user focuses the username field, the browser shows a unified
   dropdown: saved passwords + available passkeys.
4. User selects a passkey → biometric prompt appears → authenticated.
5. If no passkey is available, the user types their username and proceeds
   with password-based auth as the fallback.

**HTML — login form with conditional mediation:**

```html
<form id="login-form">
  <input name="username" autocomplete="username webauthn" required />
  <input type="password" name="password" autocomplete="current-password" />
  <button type="submit">Sign In</button>
</form>
```

**JavaScript — conditional mediation (fires on page load):**

```javascript
async function startConditionalMediation() {
  if (!window.PublicKeyCredential?.isConditionalMediationAvailable) return;
  if (!(await PublicKeyCredential.isConditionalMediationAvailable())) return;
  try {
    const cred = await navigator.credentials.get({
      mediation: "conditional",
      publicKey: {
        challenge: base64ToArrayBuffer(challengeFromServer),
        timeout: 60000, rpId: window.location.hostname,
        allowCredentials: [], userVerification: "preferred",
      },
    });
    await submitPasskeyAssertion(cred); // verify on server, complete login
  } catch (err) {
    // NotAllowedError: user cancelled — fall back to password silently
    // AbortError: timeout — user can retry. SecurityError: not HTTPS
  }
}
window.addEventListener("DOMContentLoaded", startConditionalMediation);
```

**Key principle:** The password field is always visible as fallback. Conditional
mediation never blocks — it augments the existing form.

---

## 3. First-Time Passkey Registration UX

The golden rule: **never block authentication with passkey enrollment.**
Prompt for passkey creation *after* a successful login, not during.

**Timing:**

- Show the enrollment prompt on the post-login dashboard or redirect page.
- Do not interrupt checkout, onboarding flows, or critical tasks.

**Pitch and copy:**

```
+---------------------------------------------------+
|  Create a passkey for faster, more secure sign-in |
|                                                   |
|  [Touch ID icon]   No password needed             |
|                    Use your biometrics to sign in |
|                                                   |
|  Your passkey stays on this device.               |
|                                                   |
|  [ Create Passkey ]    [ Not now ]                |
+---------------------------------------------------+
```

**Behavioral guidelines:**

- **Opt-in, not forced.** Users can dismiss. Re-prompt on subsequent logins,
  up to 3 times, then stop unless the user visits Settings.
- **Show the device name.** Display "This iPhone", "Touch ID on MacBook Pro",
  or "Windows Hello" so users know what they're enrolling.
- **Privacy reassurance.** Explicitly state the passkey never leaves the device
  and is not shared with the server.
- **Post-enrollment.** After successful registration, show a success toast:
  "Passkey created. Next time, just use Face ID to sign in."

**Registration prompt (React):**

```jsx
function PasskeyEnrollPrompt({ user, onDismiss }) {
  const [count] = useState(() => Number(localStorage.getItem("passkey_prompt_count") || 0));
  if (count >= 3) return null; // stop nagging after 3 dismissals

  const handleCreate = async () => {
    try {
      const cred = await createPasskey(user.id, user.email);
      await saveCredentialToServer(cred);
      toast.success("Passkey created! Next sign-in is passwordless.");
      onDismiss();
    } catch (err) {
      if (err.name !== "NotAllowedError") toast.error("Setup failed.");
      onDismiss();
    }
  };

  return (
    <Card>
      <BiometricIcon />
      <h3>Create a passkey for faster sign-in</h3>
      <p>No password needed. Your passkey stays on this device.</p>
      <Button onClick={handleCreate}>Create Passkey</Button>
      <Link onClick={() => { localStorage.setItem("passkey_prompt_count", count + 1); onDismiss(); }}>Not now</Link>
    </Card>
  );
}
```

**Server-side creation:** `fetch("/api/webauthn/generate-registration")` returns options,
then `navigator.credentials.create({ publicKey: options })` triggers the biometric prompt.

---

## 4. Cross-Device Auth: QR Code (Hybrid Transport)

Hybrid transport allows a user to authenticate on a desktop using a passkey
stored on their phone — without any passwords or manual codes.

**When to show it:**

- User is on desktop, no matching passkey is found via conditional mediation.
- User selects "Sign in with a passkey from another device."

**Flow:**

1. Desktop browser displays a QR code (contains a session challenge + relay URL).
2. User scans the QR code with their phone's camera (iOS/Android native).
3. Phone and desktop establish BLE proximity (confirms physical co-presence).
4. Cloud relay (Apple/Google) transmits the WebAuthn assertion.
5. Desktop receives the authenticated assertion → login complete.

**UX copy:**

```
+--------------------------------------------+
|                                            |
|         [ QR Code Image ]                  |
|                                            |
|   Scan this QR code with your phone        |
|   camera to sign in with a passkey.        |
|                                            |
|   --- or ---                               |
|                                            |
|   [ Insert a security key (USB/NFC) ]     |
|                                            |
|   [ Cancel ]                               |
+--------------------------------------------+
```

**Platform support:**

| Desktop (Relying Party) | Phone (Authenticator) | Status |
|------------------------|----------------------|--------|
| Chrome / Edge           | iOS 16+ (Safari)     | Supported |
| Chrome / Edge           | Android (GMS)        | Supported |
| Safari (macOS)          | iOS 16+               | Supported |
| Firefox                 | Any                  | Not yet supported |

**Known issues:**

- **Windows 10 Bluetooth** — some BLE drivers require pairing; Windows 11 is
  more reliable. Always show a fallback (security key / password).
- **Safari timing** — the BLE proximity check can time out if the phone is
  locked; instruct users to unlock before scanning.
- **Corporate networks** — BLE may be restricted by endpoint security policies.
- **Local network restrictions** — cloud relay requires outbound HTTPS; some
  captive portals block it.

**Accessibility:** Always provide a manual code entry alternative for users
who cannot scan QR codes (e.g., screen reader users, damaged cameras).

---

## 5. Platform Ecosystem Best Practices

### Apple

- **iCloud Keychain sync:** Passkeys sync automatically across all Apple
  devices signed into the same Apple ID. Recoverable via iCloud Data Recovery.
- **AutoFill in Safari:** Passkeys appear in the QuickType bar above the
  keyboard on iOS, and in Safari's autofill dropdown on macOS.
- **No app installation needed:** The OS-level authenticator handles
  biometric prompts natively (Face ID / Touch ID).
- **UX:** Single-tap sign-in with Face ID. Apple reports 2x faster login
  compared to passwords.
- **Share via AirDrop:** iOS 17+ allows securely sharing passkeys with
  contacts via AirDrop.

### Google

- **Google Password Manager:** Cross-device sync on Android + Chrome.
  Passkeys appear alongside saved passwords in autofill.
- **Android screen lock:** Any Android device with a screen lock (PIN,
  pattern, biometric) can serve as a passkey authenticator.
- **Pixel biometrics:** Fingerprint and Face Unlock used for passkey
  verification without additional setup.
- **QR code flow:** Desktop Chrome shows a QR code → Android phone scans →
  BLE proximity + Google cloud relay authenticates.
- **Google accounts:** Google made passkeys the default sign-in method for
  personal accounts in October 2023.

### Microsoft

- **Windows Hello:** Face, fingerprint, or PIN as the primary authenticator
  on Windows 10/11. No additional hardware needed.
- **Microsoft Authenticator app:** Recent updates add passkey support
  (FIDO2 credentials) for cross-device scenarios.
- **Edge + Windows integration:** Passkeys stored in Windows credential
  manager; Edge surfaces them in autofill.
- **Enterprise (Entra ID):** Azure AD / Entra ID supports FIDO2 security keys
  and platform authenticators for enterprise SSO. Conditional Access policies
  can mandate passkey-only sign-in for sensitive roles.
- **Admin controls:** Entra ID provides passkey lifecycle management,
  attestation enforcement, and key restrictions by AAGUID.

---

## 6. Conversion Rate Data

Passwordless authentication measurably improves both security and user
experience. Key industry data points:

| Metric | Data Point | Source |
|--------|-----------|--------|
| Login speed | 2x faster than passwords | Apple, 2023 |
| Sign-in failure reduction | 40% fewer failures | Google, 2023 |
| Account takeover prevention | 99.9% reduction | Microsoft |
| Global account passkey support | 15B+ accounts | FIDO Alliance, Dec 2024 |
| Consumer opt-in rate | 70%+ when prompted post-login | Industry surveys |
| Abandonment (passkey optional) | <5% | Deployment case studies |

**Key insights:**

- **Never block auth with passkey enrollment.** Conversion drops sharply when
  users are forced to create a passkey before completing their task.
- **Post-login prompting** achieves the highest opt-in rates (70%+) because
  users have already established trust and context.
- **Password fallback must remain available** during the transition period.
  Removing it too early causes user frustration and support tickets.
- **Education matters:** Users who understand "your passkey stays on your
  device" are significantly more likely to enroll.

---

## 7. GGID Console Passwordless UX Design

### Login Page — Conditional Mediation as Primary Auth

```tsx
// app/login/page.tsx (Next.js 15 App Router)
"use client";
import { useEffect, useState } from "react";

export default function LoginPage() {
  const [error, setError] = useState<string | null>(null);
  useEffect(() => { initConditionalMediation(); }, []);

  async function initConditionalMediation() {
    if (!window.PublicKeyCredential?.isConditionalMediationAvailable) return;
    if (!(await PublicKeyCredential.isConditionalMediationAvailable())) return;
    try {
      const challenge = await fetch("/api/webauthn/challenge").then(r => r.arrayBuffer());
      const cred = await navigator.credentials.get({
        mediation: "conditional",
        publicKey: { challenge, timeout: 120000, rpId: location.hostname,
          allowCredentials: [], userVerification: "preferred" },
      });
      const res = await fetch("/api/webauthn/verify-assertion", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify(cred),
      });
      if (res.ok) location.href = "/dashboard";
    } catch (err) {
      if (err.name === "SecurityError") setError("Passkey requires HTTPS.");
      // NotAllowedError: cancelled — fall back silently. AbortError: timeout.
    }
  }

  return (
    <form onSubmit={handlePasswordLogin}>
      <input name="username" autoComplete="username webauthn" />
      <input type="password" name="password" autoComplete="current-password" />
      {error && <p className="text-red-500">{error}</p>}
      <button type="submit">Sign In</button>
    </form>
  );
}
```

### Account Settings — Passkey Management

Simple list: show `deviceName` + `createdAt`, with Rename/Delete buttons and an
"Add Passkey" button calling the registration flow above.

### Error Handling Summary

| Error | Cause | UX Response |
|-------|-------|-------------|
| `NotAllowedError` | User cancelled biometric | Silently fall back to password |
| `SecurityError` | Non-HTTPS context | Show error: "Passkey requires HTTPS" |
| `AbortError` | Timeout (60-120s) | Allow retry, increase timeout |
| `InvalidStateError` | Credential already exists | Inform user, avoid duplicate |

### Recovery Fallback

- Always maintain password + recovery codes as backup.
- If user deletes all passkeys, revert to password login automatically.
- Enforce step-up verification before deleting the last passkey.

---

## 8. Recommendations

### Implementation Phases

| Phase | Feature | Priority | Effort |
|-------|---------|----------|--------|
| **Phase 1** | Conditional mediation on Console login | P0 | ~3 days |
| **Phase 2** | Post-login passkey enrollment prompt | P1 | ~2 days |
| **Phase 3** | Passkey-only accounts (disable password) | P2 | ~5 days |
| **Phase 4** | Hybrid transport QR code (cross-device) | P2 | ~5 days |

- **Phase 1 (P0):** Add `autocomplete="username webauthn"` + conditional `credentials.get`
  on login page. New endpoints: `/api/webauthn/challenge`, `/api/webauthn/verify-assertion`.
  Password form remains as fallback. ~3 days.
- **Phase 2 (P1):** Post-login enrollment prompt (max 3 dismissals). Track opt-in rate.
  Add passkey management UI to Settings. ~2 days.
- **Phase 3 (P2):** Allow disabling password after 1+ passkey registered. Enforce recovery
  codes before removal. Admin policy for org-wide passkey-only. ~5 days.
- **Phase 4 (P2):** Hybrid transport: "Sign in with passkey from another device" link +
  QR code generation. Test BLE on Windows 10/11 + macOS/iOS. ~5 days.

---

*Sources: FIDO Alliance (Dec 2024), Corbado WebAuthn guides, Apple/Google/Microsoft
passkey documentation, state-of-passkeys.io.*
