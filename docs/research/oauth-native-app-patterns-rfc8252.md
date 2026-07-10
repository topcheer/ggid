# OAuth 2.0 for Native Apps — RFC 8252 Patterns for GGID

## 1. Overview

**RFC 8252** (OAuth 2.0 for Native Apps, October 2017) defines OAuth 2.0 best
practices for applications running directly on the user's device:

- **Mobile**: iOS (iPhone/iPad), Android (phone/tablet)
- **Desktop**: macOS, Windows, Linux (CLI tools, Electron, native GUI apps)

The core challenge: **native apps are public clients**. They cannot keep a
`client_secret` confidential — the binary is distributed to end users and can be
decompiled, reverse-engineered, or extracted. Any compiled-in secret is public.

RFC 8252's solution combines two mechanisms:

1. **PKCE** (RFC 7636) — replaces `client_secret` as proof-of-possession during
   authorization code exchange.
2. **System browser** — authorization requests must go through the OS-default
   browser (Safari/Chrome), not embedded WebViews, enabling cross-app SSO and
   isolating credentials from the app process.

This document defines mobile SDK patterns for GGID covering PKCE generation,
redirect URI selection, system browser integration, and secure token storage.

---

## 2. PKCE Requirement for Native Apps

### Why No client_secret?

Native apps ship as binaries — APK files (Android) unpack with standard tools,
IPA files (iOS) extract from device, Electron apps bundle plaintext JavaScript.
Any embedded `client_secret` is recoverable via static analysis and must be
treated as **public knowledge**.

### PKCE as Replacement

PKCE provides **proof of key possession** without a shared secret:

1. App generates random `code_verifier` (43–128 chars, high entropy)
2. App computes `code_challenge = BASE64URL(SHA256(code_verifier))`
3. App sends `code_challenge` + `method=S256` in authorize request
4. App sends `code_verifier` in token exchange; AS verifies SHA256 match

Only the app holds the `code_verifier`. An interceptor who captures the
authorization code at the redirect URI cannot complete the exchange.

### OAuth 2.1: PKCE for All Clients

OAuth 2.1 makes PKCE **mandatory for every client type** using the authorization
code flow — including confidential server-side apps. RFC 8252 pioneered this for
native apps; OAuth 2.1 generalizes it for defense-in-depth.

### SDK Implication

GGID SDKs must **auto-generate PKCE** without developer opt-in:

```go
func generatePKCE() (verifier, challenge string, err error) {
    b := make([]byte, 32)
    if _, err = rand.Read(b); err != nil {
        return "", "", err
    }
    verifier = base64.RawURLEncoding.EncodeToString(b) // 43 chars
    h := sha256.Sum256([]byte(verifier))
    challenge = base64.RawURLEncoding.EncodeToString(h[:])
    return verifier, challenge, nil
}
```

---

## 3. Redirect URI Strategies

### Custom URI Scheme

**Format**: `com.example.app:/oauth2redirect`

The app registers a custom URL scheme with the OS. When the system browser
redirects to this URI, the OS launches the registered app.

- **iOS**: `Info.plist` → `CFBundleURLTypes`
- **Android**: `AndroidManifest.xml` → `<intent-filter>` with `<data android:scheme="...">`

**Security concern**: Any app can register the same scheme. On Android, the
last-registered app wins; on iOS, behavior is undefined with conflicts.

**Mitigation**: PKCE — the interceptor captures the code but cannot exchange it
without the `code_verifier`.

**Recommendation**: Acceptable with PKCE; claimed HTTPS preferred.

### Claimed HTTPS Redirect

**Format**: `https://app.example.com/oauth2redirect`

The developer owns the domain. Other apps cannot claim this URI. Android **App Links** verify `.well-known/assetlinks.json`; iOS **Universal Links** verify `apple-app-site-association` (AASA).

OS-level cryptographic verification prevents interception. **Most secure option**.
**Recommendation**: Preferred for production native apps.

### Loopback Redirect

**Format**: `http://127.0.0.1:{port}/callback`

The app starts a local HTTP server on a random port. The browser redirects to
localhost and the server receives the authorization code.

- **Works for**: Desktop apps (CLI tools, Electron)
- **Does NOT work for**: Mobile (background server restrictions, port conflicts)
- **RFC 8252**: Prefer `127.0.0.1` over `localhost`

```go
func startLocalCallbackServer() (string, chan string, error) {
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { return "", nil, err }
    port := ln.Addr().(*net.TCPAddr).Port
    redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
    codeCh := make(chan string, 1)
    go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        codeCh <- r.URL.Query().Get("code")
        w.Write([]byte("Done."))
        ln.Close()
    }))
    return redirectURI, codeCh, nil
}
```

### Comparison Table

| Method | Security | Mobile | Desktop | Setup | Recommendation |
|---|---|---|---|---|---|
| Custom URI Scheme | Medium (needs PKCE) | Yes | Yes | Low | Acceptable with PKCE |
| Claimed HTTPS | High (OS-verified) | Yes | No | Medium | Preferred for production |
| Loopback (127.0.0.1) | High (localhost only) | No | Yes | Low | Preferred for desktop/CLI |

---

## 4. System Browser SSO

### Why System Browser (Not WebView)

RFC 8252 Section 4: authorization requests **must** use the system browser.

**WebView problems**: No shared cookie store (no SSO), AS can't distinguish from
app, app has DOM access (credential interception risk).

**System browser benefits**: Cookies shared across all apps (SSO), trusted
credential entry context, session reuse.

### Platform APIs

**iOS — ASWebAuthenticationSession** (iOS 12+):

```swift
let session = ASWebAuthenticationSession(
    url: authURL, callbackURLScheme: "com.example.app"
) { callbackURL, error in
    guard let code = extractCode(from: callbackURL) else { return }
    ggidSDK.exchangeCode(code) { token, _ in
        KeychainHelper.save(token.refreshToken, for: "ggid_token")
    }
}
session.presentationContextProvider = self
session.start()
```

**Android — Custom Tabs**:

```kotlin
val intent = CustomTabsIntent.Builder().build()
intent.launchUrl(context, Uri.parse(authURL))
// Intent-filter captures redirect → onNewIntent() extracts code
```

### Authentication Flow

```
1. App opens ASWebAuthenticationSession / Custom Tab
2. Browser navigates to GGID authorize endpoint with PKCE challenge
3. User authenticates (AS sets session cookies in system browser)
4. GGID redirects to app's redirect URI with authorization code
5. App receives code via OS callback
6. App exchanges code + code_verifier for tokens
7. SSO established — future apps reuse system browser session
```

### Go SDK Pattern (gomobile)

Go core handles PKCE, URL building, and token exchange. Platform layer handles
browser invocation and redirect capture:

```go
func (c *NativeAppClient) StartAuth(authServerURL string) string {
    verifier, challenge, _ := generatePKCE()
    c.pkceVerifier = verifier
    c.state = generateState()
    params := url.Values{
        "response_type":         {"code"},
        "client_id":             {c.clientID},
        "redirect_uri":          {c.redirectURI},
        "code_challenge":        {challenge},
        "code_challenge_method": {"S256"},
        "state":                 {c.state},
    }
    return fmt.Sprintf("%s/oauth2/authorize?%s", authServerURL, params.Encode())
}
```

---

## 5. Token Storage on Mobile

### iOS: Keychain

- **Hardware backing**: Secure Enclave on supported devices (A7+)
- **Access control**: Require biometric (Face ID/Touch ID) via `SecAccessControl`
- **Data protection**: `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`

```swift
let access = SecAccessControlCreateWithFlags(nil,
    kSecAttrAccessibleWhenUnlockedThisDeviceOnly, .biometryCurrentSet, nil)!
let query: [String: Any] = [
    kSecClass as String: kSecClassGenericPassword,
    kSecAttrAccount as String: "ggid_refresh_token",
    kSecAttrAccessControl as String: access,
    kSecValueData as String: token.data(using: .utf8)!
]
SecItemAdd(query as CFDictionary, nil)
```

### Android: EncryptedSharedPreferences

- **Keystore**: Hardware-backed (TEE/StrongBox) — stores encryption keys
- **EncryptedSharedPreferences**: Jetpack Security, uses Keystore-managed keys

```kotlin
val masterKey = MasterKey.Builder(context)
    .setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build()
val prefs = EncryptedSharedPreferences.create(
    context, "ggid_tokens", masterKey,
    EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
    EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
)
prefs.edit().putString("refresh_token", token).apply()
```

### Desktop: OS Keychain

Cross-platform via `github.com/zalando/go-keyring`:

```go
keyring.Set(service, user, token) // store
keyring.Get(service, user)         // retrieve
```

### Security Best Practices

- **Never** store in: `SharedPreferences`, `UserDefaults`, plain files, logs
- Bind refresh token access to **biometric/PIN** — prevent silent background refresh
- **Clear on logout**: remove from Keychain/Keystore entirely
- Keep access tokens **short-lived** (5–15 min); use refresh tokens for renewal
- iOS: `prefersEphemeralWebBrowserSession = true` for apps that shouldn't SSO

### Comparison Table

| Platform | Secure Storage | Hardware-Backed | Go Access |
|---|---|---|---|
| iOS | Keychain | Secure Enclave | Swift bridge / CGO |
| Android | EncryptedSharedPreferences | TEE / StrongBox | Kotlin bridge / JNI |
| macOS | Keychain | SEP | `go-keyring` or CGO |
| Windows | Credential Manager (DPAPI) | TPM | `go-keyring` |
| Linux | Secret Service (libsecret) | TPM (if configured) | `go-keyring` |

---

## 6. GGID Mobile SDK Patterns

### Architecture

**Split design**: Go core (shared logic) + platform shell (browser + storage).

- **Platform Shell (Swift/Kotlin)**: `ASWebAuthenticationSession`/Custom Tabs,
  Keychain/EncryptedSharedPreferences, redirect URI capture
- **Go Core (gomobile)**: PKCE auto-generation, authorize URL, token exchange,
  refresh, state/nonce generation
- **GGID AS**: `/oauth2/authorize` + `/oauth2/token`, PKCE + redirect validation

### iOS Integration

```swift
// 1. Start auth via Go SDK (generates PKCE internally)
let authURL = ggidSDK.startAuth("https://auth.ggid.io")
// 2. Open system browser
let session = ASWebAuthenticationSession(url: URL(string: authURL)!,
    callbackURLScheme: "com.example.app") { callbackURL, _ in
    let code = extractCode(from: callbackURL!)
    ggidSDK.exchangeCode(code) { token, _ in
        KeychainHelper.save(token.refreshToken, for: "ggid_token")
    }
}
session.start()
```

### Android Integration

```kotlin
// 1. Start auth via Go SDK
val authURL = ggidSDK.startAuth("https://auth.ggid.io")
// 2. Open Custom Tab
CustomTabsIntent.Builder().build().launchUrl(context, Uri.parse(authURL))
// 3. onNewIntent captures redirect → extract code → exchange
ggidSDK.exchangeCode(code) { token ->
    encryptedPrefs.edit().putString("refresh_token", token.refreshToken).apply()
}
```

### NativeAppClient (Go SDK)

```go
type NativeAppClient struct {
    clientID, redirectURI, pkceVerifier, state string
    httpClient *http.Client
}

func (c *NativeAppClient) ExchangeCode(authServer, code string) (*TokenResponse, error) {
    body := url.Values{
        "grant_type": {"authorization_code"}, "code": {code},
        "redirect_uri": {c.redirectURI}, "client_id": {c.clientID},
        "code_verifier": {c.pkceVerifier},
    }
    resp, err := c.httpClient.PostForm(authServer+"/oauth2/token", body)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("exchange failed: %d", resp.StatusCode)
    }
    var token TokenResponse
    json.NewDecoder(resp.Body).Decode(&token)
    return &token, nil
}
```

---

## 7. Security Considerations

| Threat | Mitigation |
|---|---|
| Reverse engineering | Never embed `client_secret`; PKCE replaces it |
| Token theft from storage | Hardware-backed Keychain/Keystore; biometric binding |
| Redirect interception | PKCE prevents code use; prefer claimed HTTPS |
| Screen overlay/screenshot | Android: `FLAG_SECURE`; iOS: screenshot detection |
| Clipboard leakage | Never copy tokens to clipboard |
| Jailbreak/root | Detect and degrade or refuse to run |
| Auth code replay | PKCE + one-time-use codes; `state` for CSRF |
| Token replay on other devices | Device attestation (App Attest / Play Integrity) |

**Additional hardening**: `state` (random per request) for CSRF. DPoP (RFC 9449)
for sender-constrained tokens. Validate `iss` in ID tokens. Pin AS TLS.

---

## 8. GGID Roadmap

| Phase | Task | Effort |
|---|---|---|
| 1 | Verify OAuth service supports PKCE for public clients (no `client_secret`) | ~1 day |
| 2 | Configure redirect URI validation for custom schemes + claimed HTTPS | ~1 day |
| 3 | Go SDK native app support: PKCE auto-generation, `NativeAppClient` | ~3 days |
| 4a | iOS wrapper: Swift package, `ASWebAuthenticationSession`, Keychain | ~1 week |
| 4b | Android wrapper: Kotlin library, Custom Tabs, EncryptedSharedPreferences | ~1 week |
| 5 | Token storage guidance: docs, `go-keyring` desktop integration | ~2 days |
| 6 | Security hardening: `state` validation, DPoP, device attestation | ~1 week |

**Total**: ~4 weeks for full native SDK across iOS, Android, and desktop.

### Verification Checklist

- [ ] OAuth service accepts `code_challenge` with no `client_secret` for public clients
- [ ] Token endpoint rejects mismatched `code_verifier` / `code_challenge`
- [ ] Redirect URI validation supports custom schemes + App Links/Universal Links
- [ ] Go SDK generates PKCE transparently (no developer opt-in)
- [ ] iOS wrapper uses `ASWebAuthenticationSession`; Android uses Custom Tabs (not WebViews)
- [ ] Tokens in Keychain/EncryptedSharedPreferences; logout removes them entirely

---

## References

- **RFC 8252**: Native Apps — https://datatracker.ietf.org/doc/html/rfc8252
- **RFC 7636**: PKCE — https://datatracker.ietf.org/doc/html/rfc7636
- **OAuth 2.1**: https://oauth.net/2.1/
- **ASWebAuthenticationSession**: https://developer.apple.com/documentation/authenticationservices/aswebauthenticationsession
- **Android Custom Tabs**: https://developer.chrome.com/docs/android/custom-tabs
- **Jetpack Security**: https://developer.android.com/topic/security/data
- **go-keyring**: https://github.com/zalando/go-keyring
