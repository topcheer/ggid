# Advanced WebAuthn Features Roadmap v2 — 2025

> **Scope:** This document covers **advanced** WebAuthn capabilities that go beyond
> the basics in `webauthn-passkey-best-practices.md`. It focuses on extensions,
> transport internals, signature counter policy, user-verification strategy, and
> authentication attachment hints — features a production IAM like GGID needs to
> implement or track for 2025–2026.
>
> **Companion docs:**
> - `webauthn-passkey-best-practices.md` — registration/login flows, attestation, conditional UI basics
> - `webauthn-roadmap.md` — initial backlog items (backup flags, cred naming, etc.)
> - `docs/research/webauthn-roadmap-v2.md` — **this document**

---

## Table of Contents

1. [PRF Extension (Pseudo-Random Function)](#1-prf-extension-pseudo-random-function--webauthn-l3)
2. [Large Blob Storage](#2-large-blob-storage--webauthn-l2)
3. [Conditional Mediation Internals (Deep Dive)](#3-conditional-mediation-internals-deep-dive)
4. [Hybrid Transport UX Deep Dive](#4-hybrid-transport-ux-deep-dive)
5. [Credential Properties (credProps) Extension](#5-credential-properties-credprops-extension)
6. [User Verification Platform Authenticator Policy (uvpa)](#6-user-verification-platform-authenticator-policy-uvpa)
7. [Authenticator Attachment Hints](#7-authenticator-attachment-hints)
8. [Signature Counter Deep Dive](#8-signature-counter-deep-dive)
9. [WebAuthn Extensions Status Table (2025)](#9-webauthn-extensions-status-table-2025)
10. [GGID Advanced WebAuthn Roadmap v2](#10-ggid-advanced-webauthn-roadmap-v2)

---

## 1. PRF Extension (Pseudo-Random Function) — WebAuthn L3

### 1.1 What It Is

The **Pseudo-Random Function (PRF) extension** is defined in the WebAuthn Level 3
specification (`[W3C] §6.4.10`). It allows a Relying Party to request the
evaluation of a pseudo-random function bound to a specific WebAuthn credential
during authentication (`navigator.credentials.get()`) or, on newer authenticators,
during registration (`navigator.credentials.create()`).

Conceptually, PRF is a **random oracle**:
- Input: one or more caller-provided **salt** values (ArrayBuffer / TypedArray).
- Output: a **deterministic 32-byte** cryptographic value.
- The same authenticator + credential + RP ID + salt always produces the same output.
- The underlying secret key **never leaves** the authenticator.

Internally, PRF maps to the CTAP2 `hmac-secret` extension. The browser applies
a domain-separation step — hashing the caller's salt with the context string
`"WebAuthn PRF"` followed by a null byte — before passing it to the
authenticator's HMAC function. This ensures web-derived PRF outputs cannot
collide with outputs used in non-web contexts (e.g., local OS login).

### 1.2 Key Use Cases

| Use Case | Description |
|---|---|
| **End-to-end encryption (E2EE)** | Derive a per-user symmetric encryption key during login. Encrypt/decrypt user data client-side using WebCrypto (AES-GCM). The server never sees the plaintext or the key. |
| **Passwordless vault decryption** | Replace master passwords in password managers (e.g., Dashlane, Bitwarden roadmap). The passkey authentication itself unlocks the vault. |
| **Secure key rotation** | PRF accepts two salts (`first` and `second`). Use `first` for the current encryption key, `second` for the next. Rotate by promoting `second` to `first` on the server side. |
| **Per-site deterministic secrets** | Generate unique, non-guessable secrets per RP without storing them server-side. |
| **Identity wallets / non-custodial** | Derive private keys for digital wallets without ever exposing raw key material to the server. |

### 1.3 How It Works — Two Modes

#### Mode 1: `eval` (Single Derivation)

```javascript
// Client-side: navigator.credentials.get() with PRF eval
const assertion = await navigator.credentials.get({
  publicKey: {
    challenge: serverChallenge,
    rpId: "auth.ggid.dev",
    allowCredentials: [{ type: "public-key", id: credentialId }],

    extensions: {
      prf: {
        eval: {
          first: saltForEncryptionKey,  // ArrayBuffer, 32 bytes recommended
          // second is optional — used for key rotation
        },
      },
    },
  },
});

// Extract the PRF output
const prfResults = assertion.getClientExtensionResults();
const derivedKey32 = prfResults.prf.results.first; // ArrayBuffer (32 bytes)
```

The output is **deterministic**: the same `first` salt + same credential will
always produce the same 32-byte output. Use this as HKDF input to derive a
256-bit AES-GCM key via the WebCrypto API.

#### Mode 2: `evalByCredential` (Per-Credential Salts)

When multiple credentials are in `allowCredentials`, the RP can specify
different salts per credential ID:

```javascript
const assertion = await navigator.credentials.get({
  publicKey: {
    challenge: serverChallenge,
    rpId: "auth.ggid.dev",
    allowCredentials: [
      { type: "public-key", id: credId1 },
      { type: "public-key", id: credId2 },
    ],

    extensions: {
      prf: {
        evalByCredential: {
          [base64url(credId1)]: { first: saltForCred1 },
          [base64url(credId2)]: { first: saltForCred2 },
        },
      },
    },
  },
});
```

**Constraints:**
- If `evalByCredential` is non-empty, `allowCredentials` must **not** be empty.
- Keys in `evalByCredential` must be valid Base64URL encodings matching
  credential IDs in `allowCredentials`.
- `eval` and `evalByCredential` are mutually exclusive.
- On `create()`, only `eval` is allowed (not `evalByCredential`).

### 1.4 PRF at Registration Time

Newer platform authenticators (iCloud Keychain, Google Password Manager) can
return the first PRF output **during credential creation**:

```javascript
const newCredential = await navigator.credentials.create({
  publicKey: {
    // ... standard registration options ...
    extensions: {
      prf: {
        eval: { first: registrationSalt },
      },
    },
  },
});

const results = newCredential.getClientExtensionResults();
if (results.prf?.enabled) {
  // PRF is supported by this authenticator
  const firstPRF = results.prf.results.first; // 32 bytes available immediately
}
```

The `enabled: true` flag in the response confirms that the authenticator
supports PRF evaluation. If `enabled: false`, the credential was created but
PRF is not supported — you can still try PRF during `get()` later.

### 1.5 Platform Support (as of early 2025, evolving)

| Platform | Browser | Platform Authenticator | Security Key | Cross-Device (Hybrid) |
|---|---|---|---|---|
| **Android** | Chrome / Edge / Samsung Internet | Full (GPM default) | Yes | Yes |
| **Android** | Firefox | No | No | No |
| **macOS 15+** | Safari 18+ | Yes (iCloud Keychain) | No | Yes |
| **macOS 15+** | Chrome 132+ | Yes (iCloud Keychain) | Yes | Yes |
| **macOS 15+** | Firefox 139+ | Yes | Yes | Yes |
| **Windows 11 (pre-25H2)** | All | No (Windows Hello lacks hmac-secret) | Yes | Yes |
| **Windows 11 25H2 (Feb 2026+)** | Chrome 147+ / Firefox 148+ | Yes | Yes | Yes |
| **iOS/iPadOS 18.4+** | Safari / Chrome | Yes (iCloud Keychain) | No | Yes (fixed in 18.4) |

**Key takeaways:**
- Android has the most robust PRF support out of the box.
- macOS 15+ added PRF via iCloud Keychain.
- Windows 11 25H2 with KB5077181 patch (Feb 2026) enables PRF on Windows Hello.
- Always **probe** PRF support at runtime rather than assuming.

### 1.6 Security Considerations

1. **Keys never leave the authenticator.** The RP only sees deterministic outputs.
2. **Passkey loss = data loss.** Data encrypted with a PRF-derived key is
   irrecoverable if the credential is lost. **Always implement backup/recovery**
   mechanisms (e.g., additional PRF-encrypted recovery key stored server-side).
3. **Synced passkeys and PRF.** When a passkey syncs across devices (iCloud
   Keychain, Google Password Manager), the PRF secret syncs with it. This means
   a new device can derive the same encryption key — desirable for E2EE but
   changes the threat model.
4. **Do not derive keys from other WebAuthn response fields** (signature,
   authenticatorData, public key). These are either public or designed for
   verification — they are not secret and must not be used as key material.
5. **Use HKDF on the PRF output.** The raw 32-byte PRF output should be fed into
   a Key Derivation Function (e.g., HKDF-SHA-256) to derive the actual encryption
   key with proper domain separation.

### 1.7 Go Server Implementation

#### PRF Extension Struct

```go
package webauthn

// PRFSalt holds the salts for a PRF evaluation request.
// The server stores these per-session and includes them in the
// PublicKeyCredentialRequestOptions extensions.
type PRFSalt struct {
    First  []byte `json:"first,omitempty"`
    Second []byte `json:"second,omitempty"`
}

// PRFEval is the PRF extension input for navigator.credentials.get().
type PRFEval struct {
    Eval              map[string]*PRFSalt `json:"eval,omitempty"` // single-credential mode
    EvalByCredential  map[string]*PRFSalt `json:"evalByCredential,omitempty"` // per-credential mode
}

// PRFResult holds the PRF output returned by the authenticator.
type PRFResult struct {
    Enabled bool             `json:"enabled,omitempty"` // only in create() response
    Results *PRFResultValues `json:"results,omitempty"`
}

type PRFResultValues struct {
    First  []byte `json:"first,omitempty"`
    Second []byte `json:"second,omitempty"`
}
```

#### Salt Management and Key Derivation

```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"

    "golang.org/x/crypto/hkdf"
)

// GeneratePRFSalt creates a random 32-byte salt for PRF evaluation.
func GeneratePRFSalt() ([]byte, error) {
    salt := make([]byte, 32)
    if _, err := rand.Read(salt); err != nil {
        return nil, fmt.Errorf("generate PRF salt: %w", err)
    }
    return salt, nil
}

// DeriveKeyFromPRF uses HKDF-SHA256 to derive an encryption key from the
// 32-byte PRF output. The info string provides domain separation.
func DeriveKeyFromPRF(prfOutput []byte, info string) ([]byte, error) {
    derived := make([]byte, 32) // AES-256 key
    if _, err := hkdf.New(sha256.New, prfOutput, nil, []byte(info)).Read(derived); err != nil {
        return nil, fmt.Errorf("HKDF derive: %w", err)
    }
    return derived, nil
}

// StoreSaltForSession associates a PRF salt with an authentication session.
// The salt is sent to the client in the get() options; the server stores
// it to associate the PRF output with the correct encryption context.
func (h *Handler) StoreSaltForSession(challenge string, salt []byte) {
    h.sessions.save("prf:"+challenge, &sessionData{
        challenge: challenge,
        // Store salt alongside session for later verification
    })
}
```

#### Including PRF in Authentication Options

```go
// BeginAuthenticationWithPRF starts a WebAuthn authentication that requests
// PRF evaluation. The client derives a secret from the user's passkey.
func (h *Handler) BeginAuthenticationWithPRF(
    ctx context.Context,
    tenantID, userID uuid.UUID,
    existingSalt []byte,
) (*protocol.PublicKeyCredentialRequestOptions, error) {

    user, err := h.buildWebAuthnUser(ctx, tenantID, userID)
    if err != nil {
        return nil, err
    }

    // Generate a new salt if none provided
    salt := existingSalt
    if salt == nil {
        salt, err = GeneratePRFSalt()
        if err != nil {
            return nil, err
        }
    }

    // Build PRF extension input
    prfInput := map[string]any{
        "prf": map[string]any{
            "eval": map[string][]byte{
                "first": salt,
            },
        },
    }

    options, sessData, err := h.wbn.BeginLogin(user,
        webauthn.WithExtensions(prfInput),
    )
    if err != nil {
        return nil, fmt.Errorf("begin PRF login: %w", err)
    }

    // Store the salt with the session
    challenge := options.Response.Challenge.String()
    h.sessions.save("prf:"+challenge, &sessionData{
        userID:    userID,
        tenantID:  tenantID,
        challenge: challenge,
        data:      sessData,
    })

    // Return options with PRF extension visible to client
    options.Response.Extensions = prfInput
    return &options.Response, nil
}
```

#### Parsing PRF Output from Authentication Response

```go
// FinishPRFAuthentication verifies the assertion and extracts the PRF output.
func (h *Handler) FinishPRFAuthentication(
    ctx context.Context,
    parsedResponse *protocol.ParsedCredentialAssertionData,
) (*PRFResult, error) {

    challenge := parsedResponse.Response.CollectedClientData.Challenge
    sd, ok := h.sessions.get("prf:" + challenge)
    if !ok {
        return nil, fmt.Errorf("PRF session not found")
    }
    defer h.sessions.delete("prf:" + challenge)

    // Standard WebAuthn assertion verification
    user, err := h.buildWebAuthnUser(ctx, sd.tenantID, sd.userID)
    if err != nil {
        return nil, err
    }

    credential, err := h.wbn.ValidateLogin(user, *sd.data, parsedResponse)
    if err != nil {
        return nil, fmt.Errorf("verify assertion: %w", err)
    }

    // Extract PRF results from client extension results
    // The client must send the PRF output in the response body
    var prfResult PRFResult
    if raw, ok := parsedResponse.ClientExtensionResults["prf"]; ok {
        resultJSON, _ := json.Marshal(raw)
        if err := json.Unmarshal(resultJSON, &prfResult); err != nil {
            return nil, fmt.Errorf("parse PRF result: %w", err)
        }
    }

    return &prfResult, nil
}
```

### 1.8 GGID Use Case: Per-User Data-at-Rest Encryption

```go
// GGID PRF flow for encrypting tenant data at rest:
//
// 1. User registers a passkey with prf extension requested
// 2. On login, GGID sends a tenant-specific salt in the PRF eval
// 3. Client derives a 32-byte secret and sends it to GGID over TLS
// 4. GGID uses HKDF to derive an AES-256-GCM key
// 5. GGID encrypts/decrypts sensitive PII columns using this key
// 6. The key is ephemeral — derived fresh on each login
//    (same output due to PRF determinism)

// Example: encrypt a PII field
func EncryptPIIWithPRF(prfOutput []byte, plaintext []byte) ([]byte, error) {
    key, err := DeriveKeyFromPRF(prfOutput, "ggid-pii-encryption-v1")
    if err != nil {
        return nil, err
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }

    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

### 1.9 PRF JSON Examples

#### Registration with PRF (client request)

```json
{
  "publicKey": {
    "challenge": "base64url-encoded-challenge",
    "rp": { "id": "auth.ggid.dev", "name": "GGID" },
    "user": {
      "id": "base64url-user-id",
      "name": "user@example.com",
      "displayName": "User Name"
    },
    "pubKeyCredParams": [
      { "type": "public-key", "alg": -7 },
      { "type": "public-key", "alg": -257 }
    ],
    "authenticatorSelection": {
      "residentKey": "preferred",
      "userVerification": "preferred"
    },
    "extensions": {
      "prf": {
        "eval": { "first": "base64url-salt-32-bytes" }
      },
      "credProps": true
    }
  }
}
```

#### Registration response with PRF output

```json
{
  "getClientExtensionResults()": {
    "prf": {
      "enabled": true,
      "results": {
        "first": "base64url-32-byte-derived-secret"
      }
    },
    "credProps": {
      "rk": true
    }
  }
}
```

#### Authentication with evalByCredential

```json
{
  "publicKey": {
    "challenge": "base64url-challenge",
    "rpId": "auth.ggid.dev",
    "allowCredentials": [
      {
        "type": "public-key",
        "id": "base64url-cred-id-1",
        "transports": ["internal", "hybrid"]
      }
    ],
    "extensions": {
      "prf": {
        "evalByCredential": {
          "base64url-cred-id-1": {
            "first": "base64url-salt-for-cred-1"
          }
        }
      }
    }
  }
}
```

---

## 2. Large Blob Storage — WebAuthn L2+

### 2.1 What It Is

The **Large Blob Storage extension** (`largeBlob`) is defined in WebAuthn Level 2.
It allows a Relying Party to store an arbitrary blob of data (up to ~1024 bytes)
**on the authenticator**, associated with a specific discoverable credential.

Unlike PRF (which derives secrets on demand), `largeBlob` stores **static** data
at write time and retrieves it at read time.

### 2.2 Key Use Cases

| Use Case | Description |
|---|---|
| **Certificate storage** | Store an X.509 client certificate on the authenticator for mutual TLS. |
| **Credential metadata backup** | Store a backup of credential metadata (e.g., RP-specific config). |
| **Multi-device config sync** | For roaming authenticators, store app config that travels with the key. |

> **Important note from Chrome team:** Chrome developers have explicitly favored
> focusing on PRF over `largeBlob` for encryption use cases. `largeBlob` is better
> suited for non-secret auxiliary data (certificates, config) rather than
> cryptographic key material.

### 2.3 How It Works

#### During Registration (`create()`)

```javascript
const credential = await navigator.credentials.create({
  publicKey: {
    // ... standard registration options ...
    extensions: {
      largeBlob: {
        support: "preferred",  // or "required"
      },
    },
  },
});

const results = credential.getClientExtensionResults();
// results.largeBlob.supported === true  → blob storage is available
```

- `"preferred"`: Creates the credential even if blob storage isn't available.
  The `supported` output tells the RP whether it worked.
- `"required"`: Fails the `create()` call if the authenticator can't store blobs.

#### During Authentication — Read

```javascript
const assertion = await navigator.credentials.get({
  publicKey: {
    challenge: serverChallenge,
    rpId: "auth.ggid.dev",
    allowCredentials: [{ type: "public-key", id: credId }],

    extensions: {
      largeBlob: {
        read: true,
      },
    },
  },
});

const results = assertion.getClientExtensionResults();
if (results.largeBlob?.blob) {
  const blobData = new Uint8Array(results.largeBlob.blob);
  // Use the blob data (e.g., parse as certificate)
}
```

#### During Authentication — Write

```javascript
const assertion = await navigator.credentials.get({
  publicKey: {
    challenge: serverChallenge,
    rpId: "auth.ggid.dev",
    // MUST contain exactly ONE credential for write to work
    allowCredentials: [{ type: "public-key", id: credId }],

    extensions: {
      largeBlob: {
        write: blobArrayBuffer,  // up to ~1024 bytes
      },
    },
  },
});

const results = assertion.getClientExtensionResults();
// results.largeBlob.written === true → write succeeded
```

**Constraint:** A write operation requires `allowCredentials` to contain exactly
one element.

### 2.4 Platform Support

| Platform | Browser | Support |
|---|---|---|
| macOS 14+ | Safari 17+ | Yes (iCloud Keychain) |
| macOS | Chrome 113+ | Partial (security keys only on older versions) |
| Android | Chrome | Limited (not Google Password Manager) |
| Windows | Chrome | Security keys with firmware support |
| iOS 17+ | Safari | Yes (iCloud Keychain) |

**Limitations:**
- Not all authenticators support blob storage.
- Maximum blob size is authenticator-dependent (typically ~1024 bytes).
- Google Password Manager does **not** support `largeBlob`.
- Blob data is **not encrypted** by default — the RP should encrypt sensitive
  data before writing.

### 2.5 Go Server Implementation

```go
package webauthn

// LargeBlobConfig configures the largeBlob extension for registration.
type LargeBlobConfig struct {
    Support string `json:"support"` // "preferred" or "required"
}

// LargeBlobReadRequest requests blob read during authentication.
type LargeBlobReadRequest struct {
    Read bool `json:"read"`
}

// LargeBlobWriteRequest sends blob data during authentication.
type LargeBlobWriteRequest struct {
    Write []byte `json:"write"` // base64url-encoded in JSON
}

// LargeBlobResponse holds the extension output.
type LargeBlobResponse struct {
    Supported bool   `json:"supported,omitempty"` // from create()
    Blob      []byte `json:"blob,omitempty"`      // from get() read
    Written   bool   `json:"written,omitempty"`   // from get() write
}
```

#### Including largeBlob in Registration

```go
func (h *Handler) BeginRegistrationWithLargeBlob(
    ctx context.Context,
    tenantID, userID uuid.UUID,
) error {
    user, err := h.buildWebAuthnUser(ctx, tenantID, userID)
    if err != nil {
        return err
    }

    extensions := map[string]any{
        "largeBlob": map[string]string{
            "support": "preferred",
        },
        "credProps": true,
    }

    options, sessData, err := h.wbn.BeginRegistration(user,
        webauthn.WithExtensions(extensions),
    )
    if err != nil {
        return fmt.Errorf("begin registration with largeBlob: %w", err)
    }

    // Store session...
    challenge := options.Response.Challenge.String()
    h.sessions.save("reg:"+challenge, &sessionData{
        userID:   userID,
        tenantID: tenantID,
        data:     sessData,
    })

    return nil
}
```

### 2.6 largeBlob vs PRF — When to Use Which

| Aspect | `largeBlob` | `prf` |
|---|---|---|
| **Purpose** | Store static data on authenticator | Derive secret keys on demand |
| **Data type** | Arbitrary blob (cert, config) | Cryptographic key material |
| **Size limit** | ~1024 bytes | 32 bytes per salt (deterministic) |
| **Encryption** | RP must encrypt | Inherently secret (HMAC output) |
| **Chrome stance** | Deprioritized for secrets | Recommended for key derivation |
| **Authenticator support** | Limited (iCloud Keychain yes, GPM no) | Growing (Android, macOS, Windows 25H2) |

---

## 3. Conditional Mediation Internals (Deep Dive)

> **Note:** This section covers the **internals and advanced behavior** of
> conditional mediation. For basic usage (adding `autocomplete="username webauthn"`
> and `mediation: "conditional"`), see `webauthn-passkey-best-practices.md`.

### 3.1 How Conditional Mediation Works Under the Hood

Conditional mediation (`mediation: "conditional"`) changes the WebAuthn
`navigator.credentials.get()` lifecycle to integrate with the browser's native
**autofill** infrastructure instead of showing a modal dialog immediately.

#### The Lifecycle

```
1. Page loads → JavaScript calls navigator.credentials.get({ mediation: "conditional" })
2. Browser does NOT show a modal — returns a pending Promise
3. Browser injects passkey options into the autofill UI of <input autocomplete="webauthn">
4. User focuses the username field → autofill dropdown appears
5. User sees passkey option(s) alongside saved passwords
6. User selects a passkey → browser resolves the Promise with the assertion
7. OR: user types a username instead → no passkey UI interaction (Promise stays pending)
8. OR: user navigates away / new get() starts → Promise rejects with AbortError
```

#### Key Internals

1. **Silent until interaction.** The browser does **not** reveal to the page
   whether the user has a passkey until the user actively selects one from the
   autofill menu. This is a **privacy** property: the page cannot fingerprint
   users based on passkey presence.

2. **Autofill integration.** The browser hooks into the platform's autofill
   system (Chrome's autofill service on Android, Keychain on macOS/iOS,
   Credential Manager on Windows). Passkey entries appear as a special
   autofill suggestion type.

3. **RP ID → keychain lookup.** The browser uses the `rpId` from the
   `get()` options to filter the local credential store. Only passkeys
   whose RP ID matches are shown in the autofill dropdown.

4. **Discoverable credentials only.** Conditional mediation **requires**
   discoverable (resident) credentials. The `allowCredentials` array must
   be **empty** (or omitted). If `allowCredentials` is non-empty, the
   browser falls back to normal (modal) mediation.

5. **Single active request.** Only one conditional mediation `get()` can
   be active at a time. Starting a new one aborts the previous.

### 3.2 AbortController Lifecycle

```javascript
let abortController = null;

async function startConditionalUI() {
    // Abort any previous conditional request
    if (abortController) {
        abortController.abort();
    }

    abortController = new AbortController();

    try {
        const assertion = await navigator.credentials.get({
            publicKey: {
                challenge: await fetchChallenge(),
                rpId: "auth.ggid.dev",
                // Empty allowCredentials → discoverable credential flow
                allowCredentials: [],
                userVerification: "preferred",

                extensions: {
                    // Conditional UI can be combined with PRF!
                    prf: {
                        eval: { first: encryptionSalt },
                    },
                },
            },

            mediation: "conditional",
            signal: abortController.signal,
        });

        // User selected a passkey from autofill
        await sendAssertionToServer(assertion);
    } catch (err) {
        if (err.name === "AbortError") {
            // Expected: user started a new flow or navigated away
            return;
        }
        // Other errors: NotAllowedError (user cancelled modal),
        // SecurityError (rpId mismatch), etc.
        console.error("Conditional UI error:", err);
    }
}

// Call on page load
startConditionalUI();

// On form submit (user typed username/password fallback):
form.addEventListener("submit", () => {
    if (abortController) {
        abortController.abort(); // Clean up pending conditional request
    }
});
```

### 3.3 Requirements Checklist

| Requirement | Description |
|---|---|
| `mediation: "conditional"` | Must be set in `get()` options |
| Empty `allowCredentials` | Must be `[]` or omitted |
| Discoverable credentials | User must have at least one resident credential for this RP ID |
| `autocomplete="webauthn"` | Input field must have this attribute for autofill integration |
| `autocomplete="username webauthn"` | Recommended: combines username autofill with passkey option |
| HTTPS | Page must be served over HTTPS (or localhost) |
| Single active request | Only one conditional `get()` at a time |

### 3.4 Privacy Properties

Conditional mediation is designed with strict privacy guarantees:

1. **No presence signal.** The browser does not notify the page that a passkey
   exists until the user selects one. The autofill UI is rendered by the browser's
   native UI, not by page JavaScript.

2. **No timing attacks.** The `get()` Promise resolves immediately after the
   user's selection — there's no observable timing difference between "user has
   a passkey" and "user doesn't" from the page's perspective.

3. **Cross-origin isolation.** The RP ID scoping ensures that a passkey for
   `auth.ggid.dev` cannot appear in the autofill dropdown when visiting
   `evil.example.com`.

### 3.5 Error Handling Matrix

| Error | When | Action |
|---|---|---|
| `AbortError` | New `get()` started, or `AbortController.abort()` called | Silent — expected lifecycle |
| `NotAllowedError` | User dismissed the browser's passkey modal (after selecting from autofill) | Show fallback UI |
| `SecurityError` | RP ID doesn't match the page origin | Fix RP ID configuration |
| `NotSupportedError` | Browser doesn't support conditional mediation | Fall back to modal `get()` |
| `InvalidStateError` | Another `get()` with `mediation: "conditional"` is already active | Abort previous first |

### 3.6 Advanced: Combining Conditional UI with Extensions

Conditional mediation can be combined with PRF and largeBlob:

```javascript
// Conditional UI + PRF: derive encryption key during autofill login
const assertion = await navigator.credentials.get({
    publicKey: {
        challenge: challenge,
        rpId: "auth.ggid.dev",
        allowCredentials: [], // required for conditional
        userVerification: "preferred",

        extensions: {
            prf: {
                eval: { first: salt },
            },
        },
    },
    mediation: "conditional",
});

// If user selected a passkey AND it supports PRF:
const results = assertion.getClientExtensionResults();
if (results.prf?.results?.first) {
    const encryptionKey = results.prf.results.first;
    // Decrypt user data client-side
}
```

**Caveat:** If the user's passkey doesn't support PRF, `results.prf` will be
`{}` (empty object). Always handle this gracefully.

---

## 4. Hybrid Transport UX Deep Dive

### 4.1 What Hybrid Transport Is

The **hybrid** transport (also known as Cross-Device Authentication, CDA, or
"caBLE" — Cloud-assisted Bluetooth Low Energy) enables a passkey stored on a
**phone** to be used to authenticate on a **different device** (e.g., a laptop
or desktop computer that doesn't have a passkey for the current RP).

This is the flow behind "Scan QR code to sign in" UIs in Chrome and Safari.

### 4.2 Full Flow

```
┌──────────────────────────────────────────────────────────────┐
│  LAPTOP (auth.ggid.dev)              PHONE (passkey stored)   │
│                                                              │
│  1. navigator.credentials.get() called                        │
│     → browser has no local passkey for this RP               │
│     → browser shows "Use phone to sign in" / QR option        │
│                                                              │
│  2. QR code displayed on laptop ←────── 3. Phone scans QR     │
│     (contains rendezvous info via                               using camera/OS
│      FIDO Cloud Auth API / Google/Apple                       │
│      cloud relay service)                                      │
│                                                              │
│  4. BLE proximity check ────────────→ 4. BLE handshake       │
│     (both devices measure BLE                                     (proves physical
│      signal strength)                                              proximity)       │
│                                                              │
│  5. Cloud relay establishes ─────────→ 5. Phone authenticates │
│     encrypted tunnel via cloud         user with biometric      │
│     (Google FIDO API or Apple           (Face ID, fingerprint) │
│      relay)                                                               │
│                                                              │
│  6. Phone sends signed ──────────────→ 6. Assertion signed    │
│     assertion via cloud                  with phone's           │
│     relay to laptop                     passkey private key     │
│                                                              │
│  7. Laptop receives assertion → 8. POST to GGID → verified    │
└──────────────────────────────────────────────────────────────┘
```

### 4.3 Implementation Details for RP (GGID)

#### Store Transports from Registration Response

GGID's handler already stores transports (lines 410-417 in `handler.go`):

```go
// Current GGID code (handler.go finishRegistration):
var transports []string
for _, t := range credential.Transport {
    transports = append(transports, string(t))
}
if len(transports) == 0 {
    transports = []string{string(credential.Authenticator.Attachment)}
}
```

The transports array will typically contain values like:
- `"internal"` — platform authenticator (Touch ID, Face ID, Windows Hello)
- `"hybrid"` — cross-device via phone QR scan
- `"usb"` — USB security key (YubiKey)
- `"nfc"` — NFC security key
- `"ble"` — Bluetooth Low Energy (legacy, rare)

#### Include Transports in get() Options

When building `allowCredentials` for authentication, include the stored transports:

```go
// Current GGID code (handler.go beginAuthentication):
for _, wc := range user.credentials {
    var transports []protocol.AuthenticatorTransport
    for _, t := range wc.Transport {
        transports = append(transports, t)
    }
    allowCreds = append(allowCreds, protocol.CredentialDescriptor{
        Type:         protocol.PublicKeyCredentialType,
        CredentialID: wc.ID,
        Transport:    transports, // This tells the browser which methods to offer
    })
}
```

**Why this matters:** If a credential's transport includes `"hybrid"`, the browser
knows it can offer the QR code flow. If transports are missing, the browser may
not show the cross-device option.

#### RP ID / Domain Guidance

| Guideline | Why |
|---|---|
| RP ID = registrable domain (e.g., `ggid.dev`) | Passkeys are scoped to the RP ID. If you use `auth.ggid.dev`, the passkey won't work on `app.ggid.dev`. |
| Use a shared RP ID across subdomains | Set RP ID to the registrable domain so passkeys work on all subdomains. |
| `.well-known/webauthn` file | If using multiple origins, publish a WebAuthn Related Origins file to allow cross-origin assertion. |
| Ensure HTTPS | Hybrid transport requires a secure context. |

```json
// https://ggid.dev/.well-known/webauthn
{
  "origins": [
    "https://auth.ggid.dev",
    "https://app.ggid.dev",
    "https://admin.ggid.dev"
  ]
}
```

### 4.4 Known UX Issues

| Issue | Platform | Description | Mitigation |
|---|---|---|---|
| **Windows 10 Bluetooth dead-end** | Windows 10 | Windows 10 shows QR code but Bluetooth handshake fails silently or hangs. | Detect Windows 10 and show manual fallback; recommend upgrading to Windows 11. |
| **Safari QR timing** | macOS Safari | QR code may expire before phone scans it; no visual countdown. | Show retry button; instruct users to scan promptly. |
| **BLE permission prompts** | Android/Chrome | First use prompts for Bluetooth permission on the phone. | Educate users in onboarding flow. |
| **Network requirements** | Both devices | Cloud relay requires internet on both devices. | Show network error message; offer local USB/NFC option for security keys. |
| **Phone must be unlocked** | Phone | Phone must be unlocked and have the passkey in Google Password Manager or iCloud Keychain. | Document this in help text. |

### 4.5 Go Server: Transports Storage and Return

```go
// Credential already stores Transports []string in GGID's handler.go.
// The key requirement is to include them in allowCredentials during login.

// Enhanced: Add hybrid transport detection for UX hints
func (c *Credential) SupportsHybridTransport() bool {
    for _, t := range c.Transports {
        if t == "hybrid" {
            return true
        }
    }
    return false
}

// Enhanced: Add transport-based UX metadata to list credentials response
func (h *Handler) listCredentialsEnhanced(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    for _, c := range creds {
        entry := map[string]any{
            "id":               c.ID.String(),
            "name":             c.Name,
            "credential_id":    base64.RawURLEncoding.EncodeToString(c.CredentialID),
            "transports":       c.Transports,
            "supports_hybrid":  c.SupportsHybridTransport(),
            "backup_eligible":  c.BackupEligible,
            "backup_state":     c.BackupState,
            // UX hint: "This passkey can be used on other devices via QR scan"
        }
        result = append(result, entry)
    }
}
```

---

## 5. Credential Properties (credProps) Extension

### 5.1 What It Is

The **Credential Properties extension** (`credProps`) is defined in WebAuthn Level 2.
During credential creation, it tells the RP whether the created credential is
**discoverable** (resident / client-side discoverable) or not.

This is critical because **conditional mediation requires discoverable credentials**.
Without `credProps`, the RP doesn't know whether a newly created passkey will work
with autofill-based conditional UI.

### 5.2 How It Works

```javascript
// Request credProps during registration
const credential = await navigator.credentials.create({
    publicKey: {
        // ... standard options ...
        authenticatorSelection: {
            residentKey: "preferred", // request discoverable credential
        },
        extensions: {
            credProps: true,
        },
    },
});

const results = credential.getClientExtensionResults();
// results.credProps.rk === true  → credential is discoverable
// results.credProps.rk === false → credential is server-side (non-discoverable)
// results.credProps.rk === undefined → unknown (browser doesn't support credProps)
```

### 5.3 Output Interpretation

| `credProps.rk` Value | Meaning | Conditional UI Works? |
|---|---|---|
| `true` | Client-side discoverable credential | Yes |
| `false` | Server-side credential (non-discoverable) | No |
| `undefined` / absent | Browser doesn't support `credProps` | Unknown — test separately |

### 5.4 Why It Matters for GGID

1. **Conditional UI readiness.** If `credProps.rk === true`, GGID knows the
   user can use autofill-based conditional mediation. If `false`, the credential
   requires a modal `get()` flow.

2. **UX optimization.** GGID can show different UI hints: "This passkey supports
   autofill sign-in" vs. "This passkey requires a dialog prompt."

3. **Credential inventory.** The admin console can display which credentials are
   discoverable, helping admins identify which users have the best UX.

### 5.5 Platform Support

| Browser | Version | Support |
|---|---|---|
| Chrome | 108+ | Yes |
| Edge | 108+ | Yes |
| Safari | 16+ | Yes |
| Firefox | 116+ | Yes |

### 5.6 Go Implementation

```go
// credProps response from getClientExtensionResults()
type CredPropsResult struct {
    RK *bool `json:"rk"` // pointer: nil means "unknown"
}

// ParseCredProps extracts the credProps extension result from a registration response.
func ParseCredProps(clientExtensionResults map[string]any) (*CredPropsResult, error) {
    raw, ok := clientExtensionResults["credProps"]
    if !ok {
        return &CredPropsResult{RK: nil}, nil // extension not present
    }

    resultJSON, err := json.Marshal(raw)
    if err != nil {
        return nil, fmt.Errorf("marshal credProps: %w", err)
    }

    var result CredPropsResult
    if err := json.Unmarshal(resultJSON, &result); err != nil {
        return nil, fmt.Errorf("unmarshal credProps: %w", err)
    }

    return &result, nil
}

// StoreDiscoverableFlag updates the credential record with the rk flag.
// This should be called in finishRegistration after parsing the response.
func (h *Handler) finishRegistrationWithCredProps(
    ctx context.Context,
    tenantID, userID uuid.UUID,
    parsedResponse *protocol.ParsedCredentialCreationData,
    clientExtensions map[string]any,
) error {
    credProps, err := ParseCredProps(clientExtensions)
    if err != nil {
        return fmt.Errorf("parse credProps: %w", err)
    }

    // Verify the attestation
    user, err := h.buildWebAuthnUser(ctx, tenantID, userID)
    if err != nil {
        return err
    }

    challenge := parsedResponse.Response.CollectedClientData.Challenge
    sd, ok := h.sessions.get("reg:" + challenge)
    if !ok {
        return fmt.Errorf("session expired")
    }

    credential, err := h.wbn.CreateCredential(user, *sd.data, parsedResponse)
    if err != nil {
        return fmt.Errorf("verify attestation: %w", err)
    }

    // Determine if discoverable
    isDiscoverable := false
    if credProps.RK != nil {
        isDiscoverable = *credProps.RK
    }

    cred := &Credential{
        ID:              uuid.New(),
        TenantID:        tenantID,
        UserID:          userID,
        CredentialID:    credential.ID,
        PublicKey:       credential.PublicKey,
        Transports:      transportsToStrings(credential.Transport),
        Counter:         credential.Authenticator.SignCount,
        BackupEligible:  credential.Flags.BackupEligible,
        BackupState:     credential.Flags.BackupState,
        UserVerified:    credential.Flags.UserVerified,
        AttestationType: credential.AttestationType,
        AAGUID:          credential.Authenticator.AAGUID,
        Discoverable:    isDiscoverable, // NEW FIELD
        CreatedAt:       time.Now(),
    }

    return h.creds.SaveCredential(ctx, cred)
}

func transportsToStrings(t []protocol.AuthenticatorTransport) []string {
    var result []string
    for _, tr := range t {
        result = append(result, string(tr))
    }
    return result
}
```

#### Updated Credential Struct

```go
type Credential struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    UserID          uuid.UUID
    Name            string
    CredentialID    []byte
    PublicKey       []byte
    Transports      []string
    Counter         uint32
    BackupEligible  bool
    BackupState     bool
    UserVerified    bool
    AttestationType string
    AAGUID          []byte
    Discoverable    bool // NEW: from credProps.rk
    CreatedAt       time.Time
    LastUsedAt      *time.Time
}
```

---

## 6. User Verification Platform Authenticator Policy (uvpa)

### 6.1 The Three UV Values

| Value | Behavior | Use When |
|---|---|---|
| `"preferred"` (default) | Asks for UV if the authenticator supports it; succeeds without it if not. Best for most flows. | Default for most authentication — balance UX and security. |
| `"required"` | Fails if the authenticator cannot perform user verification. Forces biometric/PIN every time. | High-value operations: fund transfer, admin privilege escalation, password reset. |
| `"discouraged"` | Explicitly requests the authenticator to **not** perform user verification. | Low-risk convenience flows (e.g., session renewal when already authenticated via other means). |

### 6.2 Platform Authenticator Behavior

| Platform Authenticator | `"preferred"` | `"required"` | `"discouraged"` |
|---|---|---|---|
| **Touch ID / Face ID** (macOS/iOS) | Biometric prompt (may skip if recent) | Always biometric | No biometric, assertion has UV=false |
| **Windows Hello** | Biometric/PIN prompt | Always biometric/PIN | Falls back to platform default |
| **Android Biometric** | Biometric prompt | Always biometric | No biometric |
| **Security Key (YubiKey)** | Touch + PIN if configured | Touch + PIN required | Touch only, no PIN |

### 6.3 Impact on UX

- **`"required"`**: Every authentication forces a biometric/PIN interaction. This
  is the most secure but creates friction if overused. Users on devices without
  biometrics will fail.
- **`"preferred"`**: The authenticator decides. Platform authenticators typically
  prompt for biometrics, but the browser may apply caching heuristics (e.g., skip
  if the user recently authenticated). The assertion will include `UV=true` if
  verification was performed.
- **`"discouraged"`**: The fastest flow but provides the weakest assurance. Only
  suitable when another factor has already verified the user.

### 6.4 Risk-Based UV Selection

GGID can dynamically set the UV policy based on a risk score:

```go
// RiskScore represents the authentication risk level.
type RiskScore int

const (
    RiskLow    RiskScore = iota // Routine login, same device/IP
    RiskMedium                  // New device, unusual location
    RiskHigh                    // Sensitive operation, new geo, impossible travel
)

// UserVerificationForRisk returns the appropriate UV policy for a given risk score.
func UserVerificationForRisk(risk RiskScore) protocol.UserVerificationRequirement {
    switch risk {
    case RiskLow:
        return protocol.VerificationPreferred // Fast, still verifies if supported
    case RiskMedium:
        return protocol.VerificationPreferred // Same, but we'll enforce step-up on server
    case RiskHigh:
        return protocol.VerificationRequired // Force biometric
    default:
        return protocol.VerificationPreferred
    }
}

// CalculateRiskScore determines the risk level for an authentication attempt.
func (h *Handler) CalculateRiskScore(
    userID uuid.UUID,
    ipAddress string,
    userAgent string,
    operation string,
) RiskScore {
    // Factors:
    // - Is this IP known for this user?
    // - Is this device known?
    // - Is the operation sensitive (password reset, admin access)?
    // - Impossible travel detection
    // - Time of day anomalies

    if operation == "admin_access" || operation == "password_reset" {
        return RiskHigh
    }

    // Simplified example — real implementation uses risk engine
    if h.isNewIP(userID, ipAddress) {
        return RiskMedium
    }

    return RiskLow
}

// BeginRiskAwareAuth sets UV based on risk score.
func (h *Handler) BeginRiskAwareAuth(
    ctx context.Context,
    tenantID, userID uuid.UUID,
    ipAddress, userAgent string,
) error {
    risk := h.CalculateRiskScore(userID, ipAddress, userAgent, "login")
    uvPolicy := UserVerificationForRisk(risk)

    user, _ := h.buildWebAuthnUser(ctx, tenantID, userID)

    authSel := protocol.AuthenticatorSelection{
        UserVerification: uvPolicy,
    }

    options, sessData, err := h.wbn.BeginLogin(user,
        webauthn.WithUserVerification(uvPolicy),
    )
    if err != nil {
        return err
    }

    // Store session with risk metadata
    challenge := options.Response.Challenge.String()
    h.sessions.save("auth:"+challenge, &sessionData{
        userID:   userID,
        tenantID: tenantID,
        data:     sessData,
        // Store risk score for post-auth audit
    })

    return nil
}
```

### 6.5 Go: Setting UserVerification

```go
// In PublicKeyCredentialRequestOptions / CreationOptions
type WebAuthnLoginOptions struct {
    UserVerification string `json:"userVerification"` // "preferred" | "required" | "discouraged"
}

// go-webauthn library usage:
options, sessData, err := h.wbn.BeginLogin(user,
    webauthn.WithUserVerification(protocol.VerificationRequired), // force UV
)

// For registration:
authSel := protocol.AuthenticatorSelection{
    ResidentKey:      protocol.ResidentKeyRequirementPreferred,
    UserVerification: protocol.VerificationRequired,
}
options, sessData, err := h.wbn.BeginRegistration(user,
    webauthn.WithAuthenticatorSelection(authSel),
)
```

---

## 7. Authenticator Attachment Hints

### 7.1 The Three Attachment Values

| Value | Description | Examples |
|---|---|---|
| `"platform"` | Authenticator built into the device | Touch ID, Face ID, Windows Hello, Android Biometric |
| `"cross-platform"` | Roaming authenticator (external hardware) | YubiKey (USB/NFC), Titan Security Key, FIDO2 smart card |
| `""` (empty/unset) | No preference — any authenticator type | Default behavior |

### 7.2 When to Guide Attachment

| Scenario | Recommended Attachment | Reasoning |
|---|---|---|
| **Consumer registration** | `"platform"` | Best UX: no extra hardware, biometric, syncs across devices |
| **Enterprise / high-security** | `"cross-platform"` | Hardware-bound keys can't be phished or synced; physical possession required |
| **Mixed / BYOD** | `""` (empty) | Let user choose: platform for convenience, security key for high-assurance |
| **Financial services** | `"cross-platform"` for admin, `"platform"` for users | Tiered: admins get hardware keys, customers get biometrics |

### 7.3 Important: Attachment Is a Hint, Not a Guarantee

The `authenticatorAttachment` field in `AuthenticatorSelection` is a **hint** to the
browser. The browser may show a picker allowing the user to override it. Some
platforms may not strictly enforce it.

### 7.4 Go Implementation

```go
// Registration with platform attachment hint (consumer flow)
func (h *Handler) BeginRegistrationPlatform(
    ctx context.Context,
    tenantID, userID uuid.UUID,
) error {
    user, err := h.buildWebAuthnUser(ctx, tenantID, userID)
    if err != nil {
        return err
    }

    platformTrue := true
    authSel := protocol.AuthenticatorSelection{
        AuthenticatorAttachment: protocol.Platform,
        ResidentKey:             protocol.ResidentKeyRequirementPreferred,
        UserVerification:        protocol.VerificationPreferred,
    }

    // Note: go-webauthn uses AuthenticatorAttachment as a string field,
    // not a pointer. Check the library version for exact API.
    options, sessData, err := h.wbn.BeginRegistration(user,
        webauthn.WithAuthenticatorSelection(authSel),
    )
    if err != nil {
        return err
    }

    challenge := options.Response.Challenge.String()
    h.sessions.save("reg:"+challenge, &sessionData{
        userID:   userID,
        tenantID: tenantID,
        data:     sessData,
    })

    return nil
}

// Registration with cross-platform attachment hint (enterprise flow)
func (h *Handler) BeginRegistrationSecurityKey(
    ctx context.Context,
    tenantID, userID uuid.UUID,
) error {
    user, err := h.buildWebAuthnUser(ctx, tenantID, userID)
    if err != nil {
        return err
    }

    authSel := protocol.AuthenticatorSelection{
        AuthenticatorAttachment: protocol.CrossPlatform,
        ResidentKey:             protocol.ResidentKeyRequirementPreferred,
        UserVerification:        protocol.VerificationRequired, // enforce UV for security keys
    }

    options, sessData, err := h.wbn.BeginRegistration(user,
        webauthn.WithAuthenticatorSelection(authSel),
    )
    if err != nil {
        return err
    }

    challenge := options.Response.Challenge.String()
    h.sessions.save("reg:"+challenge, &sessionData{
        userID:   userID,
        tenantID: tenantID,
        data:     sessData,
    })

    return nil
}
```

### 7.5 Registration Options JSON

```json
// Platform (consumer) registration
{
  "authenticatorSelection": {
    "authenticatorAttachment": "platform",
    "residentKey": "preferred",
    "userVerification": "preferred"
  }
}

// Cross-platform (enterprise) registration
{
  "authenticatorSelection": {
    "authenticatorAttachment": "cross-platform",
    "residentKey": "preferred",
    "userVerification": "required"
  }
}
```

---

## 8. Signature Counter Deep Dive

### 8.1 What the Signature Counter Is

Every WebAuthn assertion includes a `signCount` field — a monotonically increasing
32-bit unsigned integer maintained by the authenticator. Its purpose is **clone
detection**: if the same credential is used on two devices (i.e., the authenticator
was cloned), the counter values will diverge, revealing the clone.

### 8.2 Clone Detection Logic

```
┌──────────────────────────────────────────────────┐
│  Authentication Assertion Received                │
│  response.signCount = 42                          │
│                                                  │
│  Lookup stored credential:                        │
│  stored.counter = 40                             │
│                                                  │
│  Is response.signCount > stored.counter?          │
│  ┌─────────────────────────────────────────────┐ │
│  │ YES → Normal: update stored.counter = 42     │ │
│  │       Authentication succeeds                │ │
│  ├─────────────────────────────────────────────┤ │
│  │ NO (response <= stored) → Possible clone!    │ │
│  │   - stored.counter = 40, response = 39       │ │
│  │   - The authenticator may have been cloned   │ │
│  │   - REJECT authentication                    │ │
│  │   - Alert security team                      │ │
│  └─────────────────────────────────────────────┘ │
│                                                  │
│  Special case: stored.counter == 0               │
│  ┌─────────────────────────────────────────────┐ │
│  │ Authenticator doesn't support counters       │ │
│  │ (synced passkeys always return 0)            │ │
│  │ → Skip counter check, allow authentication   │ │
│  └─────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

### 8.3 Current GGID Implementation

GGID already implements basic clone detection (handler.go lines 588-599):

```go
// Current GGID code (handler.go finishAuthentication):
if h.creds != nil {
    storedCred, getErr := h.creds.GetCredentialByID(ctx, tenantID, credential.ID)
    if getErr == nil && storedCred != nil && storedCred.Counter > 0 {
        if credential.Authenticator.SignCount <= storedCred.Counter {
            writeError(w, http.StatusUnauthorized, "possible credential clone detected")
            return
        }
    }
}
```

**What's correct:** The check skips counter enforcement when `storedCred.Counter == 0`
(synced passkeys).

**What's missing:**
1. No handling of the case where `response.signCount == 0` but `stored.counter > 0`
   (credential was on a device-bound key, then moved to a synced provider).
2. No audit log of clone detection events.
3. No dynamic policy based on backup_state.

### 8.4 Enhanced Signature Counter with Synced-Aware Policy

```go
// SignatureCounter defines the interface for clone detection.
type SignatureCounter interface {
    // CheckAndUpdate validates the response counter against the stored counter.
    // Returns (newCounter, cloned bool, err).
    CheckAndUpdate(storedCounter uint32, responseCounter uint32, backupEligible bool) (uint32, bool, error)
}

// DefaultCounterChecker implements synced-aware clone detection.
type DefaultCounterChecker struct{}

func (c *DefaultCounterChecker) CheckAndUpdate(
    storedCounter uint32,
    responseCounter uint32,
    backupEligible bool,
) (uint32, bool, error) {

    // Case 1: Synced credential (backup eligible) — counters are unreliable.
    // Synced passkeys (iCloud Keychain, GPM) always return signCount = 0.
    // Do NOT enforce counter monotonicity for these.
    if backupEligible {
        // Don't update counter — always 0 for synced
        return storedCounter, false, nil
    }

    // Case 2: Device-bound credential with stored counter = 0
    // The authenticator may not support counters at all.
    if storedCounter == 0 && responseCounter == 0 {
        return 0, false, nil // No counter support — skip
    }

    // Case 3: Device-bound credential with counters
    if storedCounter > 0 {
        if responseCounter <= storedCounter {
            // Counter went backward or stayed same — possible clone!
            return storedCounter, true, fmt.Errorf(
                "possible credential clone: stored=%d, response=%d",
                storedCounter, responseCounter,
            )
        }
    }

    // Normal case: response counter is higher than stored
    return responseCounter, false, nil
}

// Enhanced finishAuthentication with synced-aware clone detection.
func (h *Handler) finishAuthenticationEnhanced(w http.ResponseWriter, r *http.Request) {
    // ... parse and verify assertion as before ...

    if h.creds != nil {
        storedCred, _ := h.creds.GetCredentialByID(ctx, tenantID, credential.ID)
        if storedCred != nil {
            checker := &DefaultCounterChecker{}

            newCounter, cloned, err := checker.CheckAndUpdate(
                storedCred.Counter,
                credential.Authenticator.SignCount,
                storedCred.BackupEligible, // NEW: pass backup flag
            )

            if cloned {
                // Log security event
                // h.audit.LogCloneDetection(tenantID, userID, credential.ID)
                writeError(w, http.StatusUnauthorized,
                    "authentication denied: possible credential clone detected")
                return
            }

            if err != nil {
                writeError(w, http.StatusInternalServerError, err.Error())
                return
            }

            // Update counter only for device-bound credentials
            if !storedCred.BackupEligible && newCounter > storedCred.Counter {
                _ = h.creds.UpdateCounter(ctx, tenantID, credential.ID, newCounter)
            }
        }
    }

    // ... update last used, return success ...
}
```

### 8.5 Counter Policy Decision Matrix

| Credential Type | `backup_eligible` | `stored.counter` | `response.counter` | Action |
|---|---|---|---|---|
| Synced passkey | `true` | 0 | 0 | Skip check (always 0 for synced) |
| Synced passkey | `true` | > 0 (legacy) | 0 | Skip check, update to 0 |
| Device-bound | `false` | 0 | 0 | Skip check (no counter support) |
| Device-bound | `false` | 0 | > 0 | First use — store counter |
| Device-bound | `false` | > 0 | > stored | Normal — update counter |
| Device-bound | `false` | > 0 | <= stored | **Clone detected** — reject |

---

## 9. WebAuthn Extensions Status Table (2025)

### Complete Extensions Reference

| Extension | Spec Level | Chrome | Safari | Firefox | Status | GGID Priority |
|---|---|---|---|---|---|---|
| **appid** | L1 | All | All | All | Stable — legacy U2F backward compat | Low (legacy) |
| **appidExclude** | L1 | All | All | All | Stable — exclude legacy U2F during registration | Low (legacy) |
| **credProps** | L2 | 108+ | 16+ | 116+ | Stable — discoverable credential detection | **P0** |
| **largeBlob** | L2 | 113+ | 17+ (macOS/iOS) | No | Growing — blob storage on authenticator | P3 |
| **prf** | L3 | 118+ (auth), 147+ (create on Win) | 18+ (macOS 15+) | 139+ | Maturing — key derivation from passkey | **P2** |
| **credProtect** | L2 (CTAP) | Yes | Yes | Yes | Stable — credential protection policy | P2 |
| **minPinLength** | L2 (CTAP) | Yes | No | No | Limited — security keys only | P3 |
| **uvm** | L2 | Yes | No | No | Limited — user verification method reporting | P3 |
| **payment** (SPC) | Separate spec | Yes | No | No | Specialized — Secure Payment Confirmation | N/A |
| **hybridTransport** | L2 (transports) | All | All | All | Stable — QR / cross-device auth | **P0** |
| **act** (attestation cert) | Draft | No | No | No | Experimental | N/A |
| **credBlob** | CTAP 2.1 | Limited | No | No | Experimental — 32-byte static blob | P3 |
| **userVerification** | L1 (core) | All | All | All | Core — not an extension, but UV policy | **P2** |

### Extension Categories

#### Client-Side Extensions (processed by user agent only)
These are handled entirely by the browser:
- `credProps` — discoverability flag
- `appid` / `appidExclude` — legacy U2F

#### Authenticator-Side Extensions (processed by authenticator)
These require the authenticator to support the extension via CTAP:
- `prf` — pseudo-random function (hmac-secret)
- `largeBlob` — blob storage
- `credProtect` — credential protection policy
- `minPinLength` — minimum PIN length
- `credBlob` — small static blob
- `uvm` — user verification method

#### Transport-Related (not formal extensions)
These are part of the core spec, not the extensions mechanism:
- `transports` — array in credential descriptor (internal, hybrid, usb, nfc, ble)
- `authenticatorAttachment` — hint in authenticator selection (platform, cross-platform)

---

## 10. GGID Advanced WebAuthn Roadmap v2

### Priority Matrix

| Priority | Feature | Effort | Impact | Dependency |
|---|---|---|---|---|
| **P0** | Hybrid transport support (store/return transports) | Low | High — enables cross-device auth | None (partially done) |
| **P0** | credProps support (determine rk flag) | Low | High — enables conditional UI detection | None |
| **P1** | Signature counter with synced-aware policy | Medium | High — security (clone detection) | None (partial impl) |
| **P1** | Conditional UI deep integration | Medium | High — UX improvement | credProps (P0) |
| **P2** | PRF extension (encryption key derivation) | High | Medium — E2EE use cases | None |
| **P2** | Risk-based UV selection | Medium | Medium — adaptive security | Risk engine |
| **P3** | Large blob storage | Medium | Low — limited authenticator support | None |
| **P3** | uvm extension (auth method reporting) | Low | Low — audit/compliance | None |

---

### P0-A: Hybrid Transport Support

**Status:** Partially implemented in GGID (transports stored at registration,
returned at authentication).

**Remaining work:**

```go
// 1. Add Discoverable field to Credential struct
type Credential struct {
    // ... existing fields ...
    Discoverable bool `json:"discoverable"` // from credProps.rk
}

// 2. Store transports properly (already done in handler.go)
// Verify transports are populated in the database

// 3. Add .well-known/webauthn file support
// File: ggid/.well-known/webauthn
// {
//   "origins": ["https://auth.ggid.dev", "https://app.ggid.dev"]
// }

// 4. Enhance credential listing to show transport info
func (h *Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
    // ... existing code, add transport metadata to response ...
    entry := map[string]any{
        // ... existing fields ...
        "transports":       c.Transports,
        "supports_hybrid":  contains(c.Transports, "hybrid"),
        "supports_usb":     contains(c.Transports, "usb"),
        "supports_nfc":     contains(c.Transports, "nfc"),
    }
}
```

**Acceptance criteria:**
- Transports array stored in DB from registration response
- Transports included in `allowCredentials` during `get()`
- `.well-known/webauthn` file served for cross-origin support
- Credential list API returns transport metadata

---

### P0-B: credProps Support

**Status:** Not implemented.

**Implementation:**

```go
// 1. Add Discoverable field to Credential struct (see P0-A above)

// 2. Request credProps during registration
func (h *Handler) beginRegistration(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    extensions := map[string]any{
        "credProps": true,
    }

    options, sessData, err := h.wbn.BeginRegistration(user,
        webauthn.WithExtensions(extensions),
        webauthn.WithAuthenticatorSelection(authSel),
    )
    // ... rest of existing code ...

    // Include extensions in the response sent to client
    options.Response.Extensions = extensions
}

// 3. Parse credProps from client response in finishRegistration
func (h *Handler) finishRegistration(w http.ResponseWriter, r *http.Request) {
    // Parse the body to extract clientExtensionResults
    var requestBody struct {
        Response           json.RawMessage `json:"response"`
        ClientExtensionResults map[string]any `json:"clientExtensionResults"`
    }
    // ... parse ...

    // Extract credProps
    isDiscoverable := false
    if credPropsRaw, ok := requestBody.ClientExtensionResults["credProps"]; ok {
        if cp, ok := credPropsRaw.(map[string]any); ok {
            if rk, ok := cp["rk"].(bool); ok {
                isDiscoverable = rk
            }
        }
    }

    // Store in credential
    cred := &Credential{
        // ... existing fields ...
        Discoverable: isDiscoverable,
    }
}

// 4. Use discoverable flag for conditional UI readiness
func (c *Credential) SupportsConditionalUI() bool {
    return c.Discoverable
}
```

**Acceptance criteria:**
- `credProps: true` requested during registration
- `Discoverable` field populated from response
- Credential list API shows discoverable status
- Admin console can filter by discoverable credentials

---

### P1-A: Signature Counter with Synced-Aware Policy

**Status:** Partially implemented (basic clone detection exists).

**Implementation:** See [Section 8.4](#84-enhanced-signature-counter-with-synced-aware-policy) above.

**Remaining work:**
1. Add `DefaultCounterChecker` implementation
2. Pass `BackupEligible` flag to counter checker
3. Skip counter update for synced credentials
4. Add audit logging for clone detection events
5. Handle edge case: device-bound → synced migration (counter resets to 0)

**Acceptance criteria:**
- Counter enforcement skipped for `backup_eligible = true`
- Clone detection still works for device-bound credentials
- Audit events logged for clone detection
- Counter not updated for synced passkeys (always 0)

---

### P1-B: Conditional UI Deep Integration

**Status:** Basic conditional UI is documented in best-practices doc.

**Advanced integration:**

```go
// Server-side: detect conditional UI support from client capabilities
type ClientCapabilities struct {
    ConditionalMediation bool `json:"conditionalMediation"`
    UserVerifyingPlatformAuth bool `json:"userVerifyingPlatformAuth"`
    PRF                  bool `json:"prf"`
    LargeBlob            bool `json:"largeBlob"`
}

// Endpoint: GET /api/v1/webauthn/capabilities
// Returns server's extension support for client capability negotiation
func (h *Handler) getCapabilities(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]any{
        "extensions": map[string]bool{
            "credProps":   true,
            "prf":         true,
            "largeBlob":   false, // not yet supported
            "hybridTransport": true,
        },
        "conditionalMediation": true,
        "riskBasedUV":          true,
    })
}

// Endpoint: POST /api/v1/webauthn/auth/conditional/begin
// Returns options specifically for conditional mediation
func (h *Handler) beginConditionalAuth(w http.ResponseWriter, r *http.Request) {
    tenantIDStr := r.Header.Get("X-Tenant-ID")
    tenantID, err := uuid.Parse(tenantIDStr)
    if err != nil {
        writeError(w, http.StatusBadRequest, "missing X-Tenant-ID")
        return
    }

    // Conditional mediation requires:
    // 1. Empty allowCredentials
    // 2. Discoverable credentials only
    // 3. userVerification: preferred or required

    ephemeralUser := &webAuthnUser{
        id:          uuid.New(),
        username:    "conditional",
        displayName: "Conditional UI",
    }

    options, sessData, err := h.wbn.BeginLogin(ephemeralUser,
        webauthn.WithUserVerification(protocol.VerificationPreferred),
        // Do NOT add WithAllowedCredentials — must be empty for conditional
    )
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // Explicitly clear allowCredentials in response
    options.Response.AllowCredentials = nil

    challenge := options.Response.Challenge.String()
    h.sessions.save("conditional:"+challenge, &sessionData{
        tenantID:  tenantID,
        challenge: challenge,
        data:      sessData,
    })

    writeJSON(w, http.StatusOK, options.Response)
}
```

**Acceptance criteria:**
- Dedicated endpoint for conditional UI options
- Empty `allowCredentials` guaranteed
- Compatible with PRF extension combination
- AbortController guidance documented for client SDK

---

### P2-A: PRF Extension Implementation

**Status:** Not implemented.

**Implementation plan:**

1. Add PRF types (see [Section 1.7](#17-go-server-implementation))
2. Add salt management to session store
3. Add `BeginAuthenticationWithPRF` endpoint
4. Add `FinishPRFAuthentication` handler
5. Store PRF support flag per credential
6. Implement HKDF key derivation utility
7. Add client SDK method for PRF flow

**Acceptance criteria:**
- PRF salt generated per session
- PRF extension included in `get()` options when requested
- PRF output parsed from response
- HKDF key derivation available as utility
- Graceful fallback when PRF not supported (empty result)

---

### P2-B: Risk-Based UV Selection

**Status:** Not implemented.

**Implementation plan:**

1. Add `RiskScore` type and `CalculateRiskScore` method (see [Section 6.4](#64-risk-based-uv-selection))
2. Store IP/device history for risk calculation
3. Integrate with existing auth risk engine (if available)
4. Add configurable UV policy thresholds per tenant
5. Audit log UV decisions

**Acceptance criteria:**
- Risk score calculated from IP, device, operation type
- UV policy dynamically set based on risk
- High-risk operations force `uv: required`
- Configuration per tenant (risk thresholds)

---

### P3-A: Large Blob Storage

**Status:** Not implemented. Low priority due to limited authenticator support.

**Implementation plan:**
1. Add `LargeBlobConfig` types
2. Add `BeginRegistrationWithLargeBlob` (support: "preferred")
3. Parse `largeBlob.supported` from registration response
4. Add blob read/write endpoints
5. Store blob support flag per credential

**Note:** Chrome team recommends PRF over largeBlob for encryption use cases.
Only implement largeBlob for certificate storage or credential metadata backup.

---

### P3-B: uvm Extension

**Status:** Not implemented. Low priority.

**What it does:** Reports which user verification method was used (e.g.,
fingerprint, PIN, voice). Useful for audit and compliance.

```go
// UVMExtension result
type UVMEntry struct {
    UserVerificationMethod int `json:"userVerificationMethod"`
    CAGUID                 int `json:"caGuid"`
    KeyProtectionType      int `json:"keyProtectionType"`
    MatcherProtectionType  int `json:"matcherProtectionType"`
}
```

UV method values (from FIDO Registry):
- `1` — presence (touch)
- `2` — fingerprint
- `3` — passphrase (PIN)
- `4` — voice
- `5` — face (Face ID)
- `6` — location

---

## Appendix A: Complete Browser Support Matrix (2025)

### Extensions Support by Browser

| Extension | Chrome | Edge | Safari | Firefox | Samsung Internet |
|---|---|---|---|---|---|
| appid | All | All | All | All | All |
| appidExclude | All | All | All | All | All |
| credProps | 108+ | 108+ | 16+ | 116+ | 21+ |
| largeBlob (create) | 113+ | 113+ | 17+ | No | 21+ |
| largeBlob (read/write) | 113+ | 113+ | 17+ | No | 21+ |
| prf (get) | 118+ | 118+ | 18+ (macOS 15+) | 139+ | 25+ |
| prf (create) | 147+ (Win 25H2) | 147+ | 18+ (macOS 15+) | 148+ | 25+ |
| credProtect | Yes | Yes | Yes | Yes | Yes |
| minPinLength | Yes | Yes | No | No | Yes |
| uvm | Yes | Yes | No | No | Yes |

### Platform Authenticator PRF Support

| Platform | Provider | PRF on get() | PRF on create() |
|---|---|---|---|
| Android | Google Password Manager | Yes | Yes |
| macOS 15+ | iCloud Keychain | Yes | Yes |
| macOS | Chrome profile | No | No |
| Windows 11 pre-25H2 | Windows Hello | No | No |
| Windows 11 25H2+ | Windows Hello | Yes (Firefox 148+) | Yes (Chrome 147+) |
| iOS 18.4+ | iCloud Keychain | Yes | Yes |
| 1Password | Cross-platform | Yes | Yes |

### Transport Support

| Transport | Chrome | Safari | Firefox |
|---|---|---|---|
| internal | Yes | Yes | Yes |
| hybrid (CDA/QR) | Yes | Yes (iOS 16+) | No |
| usb | Yes | Yes (macOS) | Yes |
| nfc | Yes (Android) | Yes (iOS 16+) | Yes (Android) |
| ble (legacy) | Deprecated | No | No |

---

## Appendix B: JSON Extension Quick Reference

### All Extensions in One Registration Request

```json
{
  "publicKey": {
    "challenge": "base64url-challenge",
    "rp": { "id": "ggid.dev", "name": "GGID IAM" },
    "user": {
      "id": "base64url-user-id",
      "name": "user@example.com",
      "displayName": "User Name"
    },
    "pubKeyCredParams": [
      { "type": "public-key", "alg": -7 },
      { "type": "public-key", "alg": -257 }
    ],
    "authenticatorSelection": {
      "authenticatorAttachment": "platform",
      "residentKey": "preferred",
      "userVerification": "preferred"
    },
    "extensions": {
      "credProps": true,
      "prf": {
        "eval": {
          "first": "base64url-32-byte-salt"
        }
      },
      "largeBlob": {
        "support": "preferred"
      }
    }
  }
}
```

### All Extensions in One Authentication Request

```json
{
  "publicKey": {
    "challenge": "base64url-challenge",
    "rpId": "ggid.dev",
    "allowCredentials": [],
    "userVerification": "preferred",
    "extensions": {
      "prf": {
        "eval": {
          "first": "base64url-32-byte-salt",
          "second": "base64url-32-byte-rotation-salt"
        }
      },
      "largeBlob": {
        "read": true
      }
    }
  },
  "mediation": "conditional"
}
```

---

## Appendix C: GGID Handler Enhancement Checklist

Based on the current `handler.go` (707 lines), the following enhancements are needed:

| # | Enhancement | Current State | Target |
|---|---|---|---|
| 1 | Store `Discoverable` from `credProps` | Missing | P0 |
| 2 | Store transports array (already done) | Done (lines 410-417) | Verify DB schema |
| 3 | Return transports in `allowCredentials` (already done) | Done (lines 480-489) | Verify |
| 4 | Synced-aware signature counter | Partial (lines 588-599) | Enhance with `backup_eligible` check |
| 5 | PRF extension endpoints | Missing | P2 |
| 6 | Conditional UI dedicated endpoint | Missing | P1 |
| 7 | Risk-based UV selection | Missing | P2 |
| 8 | largeBlob extension endpoints | Missing | P3 |
| 9 | uvm extension parsing | Missing | P3 |
| 10 | `.well-known/webauthn` support | Missing | P0 (cross-origin) |
| 11 | Client capabilities endpoint | Missing | P1 |
| 12 | Audit logging for clone detection | Missing | P1 |
| 13 | Credential attachment hint per tenant policy | Missing | P2 |

---

## Appendix D: References

### Specifications
- **WebAuthn Level 3 (PRF):** https://www.w3.org/TR/webauthn-3/
- **MDN WebAuthn Extensions:** https://developer.mozilla.org/en-US/docs/Web/API/Web_Authentication_API/WebAuthn_extensions
- **CTAP 2.2 (hmac-secret, largeBlob):** https://fidoalliance.org/specs/fido-v2.2-.../fido-client-to-authenticator-protocol-v2.2-...html

### Research Sources
- **Corbado — PRF Extension Guide:** https://www.corbado.com/blog/passkeys-prf-webauthn
- **Yubico — PRF Extension:** https://developers.yubico.com/WebAuthn/Concepts/PRF_Extension/
- **Corbado — WebAuthn Transports:** https://www.corbado.com/blog/webauthn-transports-internal-hybrid
- **Google Chrome — Conditional Mediation:** https://github.com/GoogleChrome/modern-web-guidance-src/blob/main/skills-src/passkeys/references/conditional-mediation.md

### GGID Internal
- `services/auth/internal/webauthn/handler.go` — current WebAuthn handler (707 lines)
- `docs/webauthn-passkey-best-practices.md` — basics (1186 lines)
- `docs/research/webauthn-roadmap.md` — initial backlog

---

## Document Info

- **Created:** 2025
- **Author:** GGID Research
- **Version:** v2.0
- **Related docs:** `webauthn-passkey-best-practices.md`, `webauthn-roadmap.md`
- **Review cycle:** Quarterly (browser support changes rapidly)
