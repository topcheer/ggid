# Credential Management API

> Research note on the W3C Credential Management API and its role in
> password + passkey unified authentication for the GGID Admin Console.

---

## 1. Overview

The **Credential Management API** (CMA) is a W3C specification that gives
web applications an imperative interface to create, store, and retrieve
user credentials through the browser's native credential manager.

**Core methods:**

| Method | Purpose |
|---|---|
| `navigator.credentials.get()` | Request a credential from the user (password, federated, or WebAuthn) |
| `navigator.credentials.store()` | Save a credential after successful authentication |
| `navigator.credentials.create()` | Create a new credential (currently delegates to WebAuthn) |

**Problem solved:** Before CMA, browsers had opaque password managers
and WebAuthn used a separate API surface. CMA unifies them — one request
can ask for a saved password, a federated login, or a passkey.

**Spec status:** W3C Credential Management Level 1 (Working Draft).
WebAuthn Level 2/3 extends it with `PublicKeyCredential`. All major
browsers implement `get()` and `create()`; `store()` and
`PasswordCredential` are Chromium-only.

> Reference: <https://www.w3.org/TR/credential-management-1/> —
> MDN: <https://developer.mozilla.org/en-US/docs/Web/API/Credential_Management_API>

---

## 2. API Surface

### 2.1 navigator.credentials.get()

Returns a `Promise<Credential | null>`. The `password` and `federated`
options are gated behind feature detection; `publicKey` (WebAuthn) is
universally supported.

```javascript
const cred = await navigator.credentials.get({
  password: true,              // PasswordCredential
  federated: { providers: ['https://accounts.google.com'] },
  publicKey: {                 // PublicKeyCredential (WebAuthn/passkey)
    challenge: new Uint8Array([/* server challenge */]),
    allowCredentials: [],      // empty = discoverable (passkey) login
    userVerification: 'preferred',
  },
  mediation: 'optional',       // 'silent'|'optional'|'conditional'|'required'
  signal: abortController.signal,
});
```

**Mediation values:**

| Value | Behaviour |
|---|---|
| `"silent"` | Return only if no user interaction is needed; reject otherwise |
| `"optional"` | Browser decides whether to show UI (default) |
| `"conditional"` | Credential request runs in background; UI appears as autofill suggestions in input fields |
| `"required"` | Always require explicit user action |

**AbortSignal:** Pass an `AbortController.signal` to cancel a pending
request. This is critical for conditional mediation lifecycle management
(see section 6).

### 2.2 navigator.credentials.store()

Called **after** a successful traditional login to ask the browser's
password manager to save (or update) the credential.

```javascript
// After a successful username/password login
const passwordCred = new PasswordCredential({
  id: userEmail,
  password: userPassword,
  name: userDisplayName,
});
await navigator.credentials.store(passwordCred);
// Browser shows "Save password?" prompt (if not already saved)
```

For federated logins, store a `FederatedCredential` so the browser can
offer "Sign in with Google" in future autofill suggestions.

> **Browser support:** `store()` works in Chrome and Edge. Safari
> supports it partially (password store only). Firefox does not implement
> `PasswordCredential` or `FederatedCredential`.

### 2.3 navigator.credentials.create()

Creates a new credential. Currently this exclusively delegates to the
WebAuthn API (`PublicKeyCredential`).

```javascript
const newCred = await navigator.credentials.create({
  publicKey: {
    challenge: serverChallenge,
    rp: { name: 'GGID' },
    user: { id: userIdBuffer, name: userEmail, displayName: userDisplayName },
    pubKeyCredParams: [{ type: 'public-key', alg: -7 }, { type: 'public-key', alg: -257 }],
    authenticatorSelection: { residentKey: 'preferred', userVerification: 'preferred' },
    excludeCredentials: existingCredentialIds,
  },
});
```

Future spec versions may allow `create()` to mint other credential types
(digital wallet credentials, OIDC VP tokens), but today it is WebAuthn-only.

---

## 3. Autofill Integration

The most impactful feature of CMA is **conditional mediation** with HTML
autofill integration. This is the mechanism that makes passkey-first
login feel seamless to end users.

**How it works:**

1. The page includes an `<input>` with `autocomplete="username webauthn"`.
2. JavaScript calls `navigator.credentials.get({ mediation: "conditional" })`.
3. The browser focuses the input and shows a **unified autofill dropdown**
   containing saved passwords **and** available passkeys side by side.
4. The user selects a credential (or types their username).
5. If a credential is selected, the promise resolves with that credential.
   If the user submits the form manually, the conditional request is
   aborted.

**This is the primary UX for passkey-first authentication in 2024+.**
Users do not need a separate "Sign in with passkey" button — the passkey
option appears inline in the username field alongside saved passwords.

```html
<form id="login-form">
  <input type="text" name="username" autocomplete="username webauthn" required />
  <input type="password" name="password" autocomplete="current-password" />
  <button type="submit">Sign in</button>
</form>
```

```javascript
const ac = new AbortController();
if (window.PublicKeyCredential?.isConditionalMediationAvailable?.()) {
  navigator.credentials
    .get({ mediation: 'conditional', publicKey: { challenge: await fetchChallenge(), userVerification: 'preferred' } })
    .then((cred) => cred && submitWebAuthnAssertion(cred))
    .catch((err) => { if (err.name !== 'AbortError') console.error(err); });
}
// Form submit aborts conditional mediation
document.getElementById('login-form').addEventListener('submit', (e) => {
  e.preventDefault(); ac.abort(); /* submit password... */
});
```

---

## 4. Browser Support Matrix

| Feature | Chrome | Edge | Safari | Firefox | Samsung |
|---|---|---|---|---|---|
| `navigator.credentials.get()` | Yes | Yes | Yes | Yes | Yes |
| `PublicKeyCredential` (WebAuthn) | Yes | Yes | Yes | Yes | Yes |
| `PasswordCredential` | Yes | Yes | No | No | Yes |
| `FederatedCredential` | Yes | Yes | No | No | Yes |
| `navigator.credentials.store()` | Yes | Yes | Partial | No | Yes |
| Conditional mediation | 107+ | 107+ | 16+ | 122+ | 21+ |
| `isConditionalMediationAvailable()` | Yes | Yes | Yes | Yes | Yes |

**Key takeaways:**

- **`get()` with `PublicKeyCredential`** is universally available — this
  is the safe path for all browsers.
- **`PasswordCredential` / `FederatedCredential`** are Chromium-only. On
  Safari and Firefox, the browser's built-in password manager handles
  password autofill through traditional HTML `autocomplete` attributes;
  CMA `store()` is not needed.
- **Conditional mediation** is the critical feature for passkey-first UX
  and is now available in all major browsers (Firefox 122 shipped in
  Jan 2024).
- Always **feature-detect** with `PublicKeyCredential.isConditionalMediationAvailable()`
  before relying on conditional mediation.

---

## 5. Password/Passkey Unified UX

CMA allows requesting **both** password and WebAuthn credentials in a
single `get()` call. The browser presents a unified picker:

```javascript
async function unifiedLogin() {
  const cred = await navigator.credentials.get({
    password: true,
    publicKey: { challenge: await fetchChallenge(), userVerification: 'preferred' },
    mediation: 'required', // always show the browser credential picker
  });
  if (!cred) return showLoginForm();
  switch (cred.type) {
    case 'password':    return submitPassword(cred.id, cred.password);
    case 'public-key':  return submitWebAuthnAssertion(cred);
    case 'federated':   return window.location.href = cred.provider;
  }
}
```

**Migration pattern:**

1. User logs in with password for the first time → call `store()` to
   save the `PasswordCredential`.
2. After login, offer passkey enrollment via `create()`.
3. On next visit, `get()` with `mediation: "required"` shows both the
   saved password and the enrolled passkey in one picker.
4. User picks whichever they prefer. Over time, most users gravitate
   to passkeys (faster, no typing).

**Fallback:** If `get()` returns `null` or throws, show the traditional
login form. Never assume a credential will be available.

---

## 6. WebAuthn Conditional Mediation Integration

Conditional mediation is not a separate API — it is a **mediation mode**
of `navigator.credentials.get()` applied to `PublicKeyCredential`. The
Credential Management API is the foundation; WebAuthn is a consumer.

**How they connect:**

```
navigator.credentials.get({
  mediation: 'conditional',   ← CMA mediation parameter
  publicKey: { ... }          ← WebAuthn options
})
```

The browser:
1. Does **not** show a modal immediately (unlike `mediation: "required"`).
2. Waits for focus on an `autocomplete="webauthn"` field, then shows
   passkey suggestions inline.
3. When the user selects a passkey, the WebAuthn ceremony (biometric,
   security key tap) executes.

**AbortController lifecycle:**

```javascript
let conditionalAC = null;
async function startConditionalLogin() {
  conditionalAC?.abort(); // abort any previous conditional request
  conditionalAC = new AbortController();
  return await navigator.credentials.get({
    mediation: 'conditional',
    signal: conditionalAC.signal,
    publicKey: { challenge: await fetchChallenge(), userVerification: 'preferred' },
  });
}
// When navigating away: conditionalAC?.abort();
```

**Requirements for conditional mediation:**

- **HTTPS only** — the page must be served over a secure context.
- **Discoverable credentials** (passkeys) — the WebAuthn request must
  not specify `allowCredentials` (or pass an empty array). Resident
  credentials are required so the authenticator can identify the user
  without a credential ID hint.
- **Feature detection** — call `PublicKeyCredential.isConditionalMediationAvailable()`
  and `PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()`
  before starting.
- **One at a time** — only one conditional mediation request may be
  active per document. Starting a new one aborts the previous.

---

## 7. GGID Console Integration

GGID's Admin Console is built with **Next.js 15 + React 19**. The
Credential Management API integrates naturally into the login flow and
the user settings (passkey enrollment) page.

### 7.1 Login Page — Conditional Mediation

```tsx
// console/src/app/login/page.tsx
'use client';
import { useEffect, useRef } from 'react';

export default function LoginPage() {
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    async function tryConditionalMediation() {
      const ok = await window.PublicKeyCredential?.isConditionalMediationAvailable?.()
        ?? Promise.resolve(false);
      if (!ok) return;
      abortRef.current?.abort();
      abortRef.current = new AbortController();
      try {
        const challenge = await fetch('/api/auth/webauthn/challenge').then((r) => r.arrayBuffer());
        const cred = (await navigator.credentials.get({
          mediation: 'conditional',
          signal: abortRef.current.signal,
          publicKey: { challenge: new Uint8Array(challenge), userVerification: 'preferred' },
        })) as PublicKeyCredential | null;
        if (cred) {
          const res = await fetch('/api/auth/webauthn/login', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(serializeAssertion(cred)),
          });
          if (res.ok) window.location.href = '/dashboard';
        }
      } catch (err) {
        if ((err as Error).name !== 'AbortError') console.error('Conditional mediation:', err);
      }
    }
    tryConditionalMediation();
    return () => abortRef.current?.abort();
  }, []);

  return (
    <form onSubmit={handleSubmit}>
      <input name="username" autoComplete="username webauthn" placeholder="Email or username" />
      <input type="password" name="password" autoComplete="current-password" />
      <button type="submit">Sign in</button>
    </form>
  );
}
```

### 7.2 Settings Page — Passkey Enrollment

```tsx
async function enrollPasskey(userId: string) {
  try {
    const opts = await fetch('/api/auth/webauthn/begin-registration', {
      method: 'POST', body: JSON.stringify({ userId }),
    }).then((r) => r.json());
    const cred = (await navigator.credentials.create({
      publicKey: { ...opts, challenge: base64ToBuffer(opts.challenge),
        user: { ...opts.user, id: base64ToBuffer(opts.user.id) } },
    })) as PublicKeyCredential;
    await fetch('/api/auth/webauthn/finish-registration', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(serializeAttestation(cred)),
    });
  } catch (err) { handleCredentialError(err); }
}
```

### 7.3 Error Handling

| Error | Cause | Recovery |
|---|---|---|
| `NotAllowedError` | User cancelled or dismissed the browser prompt | Show fallback form; offer retry |
| `SecurityError` | Page not HTTPS, or cross-origin iframe | Ensure top-level secure context |
| `AbortError` | Conditional mediation aborted (expected) | No action needed |
| `InvalidStateError` | Authenticator already registered for this RP | Inform user; dedupe server-side |
| `NotSupportedError` | No compatible authenticator or transport | Suggest platform authenticator setup |

---

## 8. Recommendations

1. **Adopt conditional mediation as the primary login UX.** It provides
   the best passkey-first experience without sacrificing password
   users. Feature-detect and fall back gracefully.

2. **Use `store()` to save passwords for legacy users** after traditional
   login. This ensures the browser's autofill includes their password on
   future visits, even before they enroll a passkey.

3. **Enroll passkeys post-login.** After a successful password login,
   proactively offer passkey enrollment via `create()` — this is the
   single most effective migration lever.

4. **Never block on CMA.** All credential API calls should be
   non-blocking augmentations over a traditional form. If the API is
   unavailable or fails, the form must still work.

5. **Timeline:** Implement conditional mediation + passkey enrollment in
   **Console v2**. The GGID backend already supports WebAuthn
   registration/login endpoints (`/auth/webauthn/begin-registration`,
   `/auth/webauthn/login`). The frontend integration is the remaining
   gap.

---

*Last updated: 2025. Sources: W3C Credential Management Level 1, MDN Web
Docs, WebAuthn Level 2/3.*
