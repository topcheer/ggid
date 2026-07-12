# Mobile Authentication Guide

## Overview

Mobile authentication presents unique challenges and opportunities compared to web authentication. This guide covers biometric authentication, OAuth for mobile apps, WebAuthn platform authenticators, push MFA, app security (secure storage, certificate pinning), device attestation, and how GGID supports mobile authentication.

## Mobile Authentication Methods

### Biometric Authentication

Biometric authentication uses device-native biometric sensors for local user verification.

- **Face ID (iOS)**: Facial recognition via TrueDepth camera
- **Touch ID (iOS)**: Fingerprint recognition via Home button or sensor
- **Fingerprint (Android)**: Fingerprint via biometric sensor
- **Face unlock (Android)**: Facial recognition via front camera
- **Iris/retina (Samsung)**: Iris scanning on supported devices

#### Implementation Pattern

Biometrics in mobile apps follow a two-tier model:

1. **Primary authentication**: User authenticates to the app using biometrics (local device verification)
2. **Token issuance**: App uses a stored refresh token or key to obtain/refresh access tokens

The biometric verification is local to the device - the server trusts the device's attestation rather than the biometric data itself.

```
User -> Biometric Prompt -> Device OS verifies -> App unlocks stored credential -> App calls GGID token endpoint -> Access token issued
```

#### iOS (LocalAuthentication)

```swift
import LocalAuthentication

func authenticateWithBiometrics() {
    let context = LAContext()
    var error: NSError?

    guard context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: &error) else {
        // Fallback to passcode
        return
    }

    context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics,
                          localizedReason: "Authenticate to access GGID") { success, error in
        DispatchQueue.main.async {
            if success {
                // Retrieve stored refresh token from Keychain
                let token = KeychainHelper.shared.read(key: "ggid_refresh_token")
                // Exchange for access token
                self.refreshAccessToken(refreshToken: token)
            } else {
                // Handle failure
            }
        }
    }
}
```

#### Android (BiometricPrompt)

```kotlin
val biometricPrompt = BiometricPrompt(
    activity,
    ContextCompat.getMainExecutor(context),
    object : BiometricPrompt.AuthenticationCallback() {
        override fun onAuthenticationSucceeded(result: BiometricPrompt.AuthenticationResult) {
            // Retrieve stored token from Keystore
            val token = keystoreHelper.getDecryptedToken("ggid_refresh_token")
            // Exchange for access token
            refreshAccessToken(token)
        }
    }
)

val promptInfo = BiometricPrompt.PromptInfo.Builder()
    .setTitle("Authenticate")
    .setSubtitle("Access GGID")
    .setNegativeButtonText("Use password")
    .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
    .build()

biometricPrompt.authenticate(promptInfo)
```

### OAuth for Mobile Apps

Mobile apps are public clients - they cannot securely store a client secret. OAuth 2.1 requires PKCE for all mobile authorization flows.

#### Authorization Code + PKCE

```
1. App generates code_verifier (random 43-128 char string)
2. App computes code_challenge = BASE64URL(SHA256(code_verifier))
3. App opens system browser: GET /authorize?response_type=code&client_id=...&redirect_uri=com.example.app://callback&code_challenge=...&code_challenge_method=S256&state=...
4. User authenticates in browser (existing session, MFA)
5. GGID redirects to com.example.app://callback?code=AUTH_CODE&state=...
6. App exchanges code for tokens: POST /token with code_verifier
7. GGID verifies code_challenge matches
8. App receives access_token + refresh_token
```

#### Custom URL Scheme vs App Links

| Method | Platform | Security | UX |
|--------|----------|---------|-----|
| Custom URL scheme | iOS + Android | Medium (app hijack risk) | Simple |
| Universal Links (iOS) | iOS | High (verified domain) | Seamless |
| App Links (Android) | Android | High (verified domain) | Seamless |

**Recommendation**: Use Universal Links (iOS) and App Links (Android) for production apps. Custom URL schemes are acceptable for development.

#### iOS PKCE Implementation

```swift
import CryptoKit

func generatePKCE() -> (verifier: String, challenge: String) {
    var verifierBytes = [UInt8](repeating: 0, count: 32)
    _ = SecRandomCopyBytes(kSecRandomDefault, 32, &verifierBytes)
    let verifier = Data(verifierBytes).base64URLEncodedString()

    let challenge = Data(SHA256.hash(data: Data(verifier.utf8)))
        .base64URLEncodedString()

    return (verifier, challenge)
}

func startAuthFlow() {
    let pkce = generatePKCE()
    // Store verifier securely
    KeychainHelper.shared.save(key: "pkce_verifier", value: pkce.verifier)

    var components = URLComponents(string: "https://ggid.example.com/oauth/authorize")!
    components.queryItems = [
        URLQueryItem(name: "response_type", value: "code"),
        URLQueryItem(name: "client_id", value: clientId),
        URLQueryItem(name: "redirect_uri", value: "https://app.example.com/oauth/callback"),
        URLQueryItem(name: "code_challenge", value: pkce.challenge),
        URLQueryItem(name: "code_challenge_method", value: "S256"),
        URLQueryItem(name: "state", value: state),
    ]
    UIApplication.shared.open(components.url!)
}
```

### WebAuthn Platform Authenticators

Mobile devices can serve as WebAuthn platform authenticators using device biometrics.

- **iOS**: Safari + SFSafariViewController support WebAuthn with Face ID/Touch ID
- **Android**: Chrome supports WebAuthn with biometric verification
- **Registration**: Device creates a credential key pair, private key in Secure Enclave/Keystore
- **Authentication**: User verifies biometrically, device signs challenge with private key

#### Mobile WebAuthn Flow

```
1. App/Site requests WebAuthn registration
2. Browser invokes platform authenticator (Face ID/Touch ID/biometric)
3. User verifies biometric
4. Device generates key pair, stores private key in hardware-backed keystore
5. Public key sent to GGID
6. Future auth: GGID sends challenge, device signs with biometric verification, GGID verifies
```

### Push MFA

Push-based MFA sends a notification to the user's mobile device for out-of-band authentication.

- **Flow**: User enters password, GGID sends push notification, user approves/denies on device
- **Security**: Push approval requires device possession factor
- **Context**: Include context (location, IP, resource) in push notification for user awareness
- **Anti-fatigue**: Number matching to prevent MFA fatigue attacks (user must match a number shown on screen)

#### Implementation Considerations

- **Push delivery**: APNs (iOS), FCM (Android) for notification delivery
- **Timeout**: Push approval should expire after 60-120 seconds
- **Fallback**: Allow fallback to TOTP if push delivery fails
- **Rate limiting**: Limit push attempts to prevent fatigue
- **Device binding**: Bind push MFA to a specific enrolled device

## App Security

### Secure Storage

#### iOS Keychain

```swift
// Store token securely in Keychain
func saveToken(_ token: String, key: String) {
    let data = token.data(using: .utf8)!
    let query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrAccount as String: key,
        kSecValueData as String: data,
        kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
    ]
    SecItemAdd(query as CFDictionary, nil)
}

// Retrieve token from Keychain
func loadToken(key: String) -> String? {
    let query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrAccount as String: key,
        kSecReturnData as String: true,
        kSecMatchLimit as String: kSecMatchLimitOne,
    ]
    var item: CFTypeRef?
    SecItemCopyMatching(query as CFDictionary, &item)
    return (item as? Data)?.map { String(format: "%c", $0) }.joined()
}
```

**Keychain access control flags**:
- `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`: Token accessible when device unlocked, not backed up
- `kSecAttrAccessibleWhenPasscodeSetThisDeviceOnly`: Only if passcode is set, not backed up (recommended for tokens)
- `kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly`: Accessible after first unlock, for background tasks

#### Android Keystore

```kotlin
// Store encrypted token using Keystore
fun saveEncryptedToken(key: String, token: String) {
    val masterKey = MasterKey.Builder(context)
        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
        .setUserAuthenticationRequired(true)
        .build()

    val encryptedPrefs = EncryptedSharedPreferences.create(
        context,
        "ggid_secure_prefs",
        masterKey,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    encryptedPrefs.edit().putString(key, token).apply()
}
```

**Keystore best practices**:
- Use `setUserAuthenticationRequired(true)` to require biometric/passcode for key access
- Use hardware-backed Keystore when available (StrongBox on supported devices)
- Never store tokens in SharedPreferences without encryption
- Clear tokens on device wipe/remote wipe

### Certificate Pinning

Certificate pinning prevents MITM attacks by validating server certificates against known pins.

#### iOS (URLSession)

```swift
func urlSession(_ session: URLSession,
                didReceive challenge: URLAuthenticationChallenge,
                completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
    guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
          let serverTrust = challenge.protectionSpace.serverTrust else {
        completionHandler(.performDefaultHandling, nil)
        return
    }

    let certificate = SecTrustCopyCertificateChain(serverTrust) as! [SecCertificate]
    let serverCertData = SecCertificateCopyData(certificate[0]) as Data

    let pinnedCertData = loadPinnedCertificate()

    if serverCertData == pinnedCertData {
        completionHandler(.useCredential, URLCredential(trust: serverTrust))
    } else {
        completionHandler(.cancelAuthenticationChallenge, nil)
    }
}
```

#### Android (OkHttp)

```kotlin
val certificatePinner = CertificatePinner.Builder()
    .add("ggid.example.com", "sha256/ABC123...")
    .add("ggid.example.com", "sha256/DEF456...")  // backup pin
    .build()

val client = OkHttpClient.Builder()
    .certificatePinner(certificatePinner)
    .build()
```

**Pinning best practices**:
- Always include a backup pin in case of certificate rotation
- Pin the intermediate CA or public key, not just the leaf certificate
- Use SPKI (Subject Public Key Info) hash, not the full certificate
- Plan for pin rotation in your app update cycle
- Consider using a remote pin update mechanism for emergency rotation

### Secure Communication

- **TLS 1.2+ minimum**: Reject TLS 1.0/1.1 connections
- **Certificate validation**: Full chain validation + pinning
- **No certificate bypass**: Never disable certificate validation, even in debug builds
- **Perfect forward secrecy**: Ensure cipher suites support PFS
- **App transport security (iOS)**: Enable ATS, no exceptions for auth endpoints

## Device Attestation

Device attestation proves the integrity and identity of a device to the server.

### Android Play Integrity API

```kotlin
val integrityManager = IntegrityManagerFactory.create(context)
val tokenProvider = StandardIntegrityTokenProvider(
    integrityManager,
    StandardIntegrityTokenProvider.Request.Builder()
        .setCloudProjectNumber(cloudProjectNumber)
        .build()
)

// Request integrity token
val token = tokenProvider.request()
    .setRequestHash(hashOfRequestData)
    .build()
    .get()

// Send token to GGID server for verification
```

**Play Integrity verifies**:
- App is the genuine version from Google Play
- Device has a verified boot state
- Device has Google Play Protect active
- App has not been tampered with

### iOS DeviceCheck / App Attest

```swift
// DCAppAttestService (iOS 14+)
import DeviceCheck

func generateAttestation() {
    DCAppAttestService.shared.generateKey { keyId, error in
        guard let keyId = keyId else { return }

        // Store keyId securely
        self.storeKeyId(keyId)

        // Request attestation from Apple
        DCAppAttestService.shared.attestKey(keyId, clientDataHash: clientDataHash) { attestation, error in
            guard let attestation = attestation else { return }

            // Send attestation to GGID server
            self.sendAttestationToServer(attestation: attestation)
        }
    }
}

// For each sensitive operation, generate an assertion
func generateAssertion(clientDataHash: Data) {
    DCAppAttestService.shared.generateAssertion(keyId, clientDataHash: clientDataHash) { assertion, error in
        guard let assertion = assertion else { return }
        // Send assertion with the request to GGID
    }
}
```

**App Attest verifies**:
- App is genuine and from the App Store
- Device is genuine Apple hardware
- Key is hardware-backed (Secure Enclave)
- App has not been tampered with

### Attestation in GGID

GGID can use device attestation as part of its risk engine:

1. **Registration**: App sends attestation token during MFA enrollment
2. **Authentication**: App sends assertion with high-risk authentication requests
3. **Verification**: GGID validates attestation via Apple/Google APIs
4. **Risk scoring**: Attestation result feeds into risk score (failed attestation = higher risk)
5. **Access control**: Deny or require additional MFA for non-attested devices on sensitive resources

## Offline Authentication

For scenarios where the device may not have network connectivity:

- **Cached tokens**: Store access and refresh tokens locally with expiration validation
- **Offline grace period**: Allow token use for a configurable period after expiration (e.g., 24 hours)
- **Refresh on reconnect**: Automatically refresh tokens when connectivity is restored
- **Local biometric unlock**: Use biometric verification to unlock cached credentials
- **Offline limits**: Restrict sensitive operations (e.g., admin actions) to online mode only
- **Conflict resolution**: Handle token revocation that occurred during offline period

## GGID Mobile Authentication

### Supported Methods

| Method | Implementation | Status |
|--------|---------------|--------|
| OAuth 2.1 + PKCE | Authorization Code flow with PKCE | Supported |
| Biometric (local) | Via app SDK using Keychain/Keystore | App-managed |
| WebAuthn | Platform authenticator via browser | Supported |
| Push MFA | Via notification service | Roadmap |
| TOTP | Via authenticator app | Supported |
| Device attestation | Play Integrity / App Attest | Risk engine integration |

### Mobile SDK

GGID provides SDKs for mobile integration:

- **Go SDK**: For backend-for-frontend (BFF) pattern
- **Node SDK**: For React Native / Node.js backends
- **Java SDK**: For Android native apps

**SDK capabilities**:
- OAuth 2.1 authorization code + PKCE flow
- Token refresh and management
- JWKS caching and validation
- UserInfo retrieval
- Device registration for push MFA
- Risk-aware authentication

### Security Recommendations for GGID Mobile Apps

1. **Always use PKCE**: Never embed client secrets in mobile apps
2. **Use system browser**: Prefer ASWebAuthenticationSession (iOS) / Custom Tabs (Android) over in-app webviews
3. **Store tokens in secure storage**: Keychain (iOS) or EncryptedSharedPreferences/Keystore (Android)
4. **Implement certificate pinning**: Pin GGID API endpoints in production
5. **Use biometric verification**: Require biometric unlock for token access
6. **Implement device attestation**: Use Play Integrity/App Attest for high-security scenarios
7. **Handle token revocation**: Clear local tokens on 401 response or revocation notification
8. **Secure deep links**: Validate redirect URI parameters to prevent injection
9. **Implement session timeout**: Force re-authentication after configurable idle period
10. **Log security events**: Send auth events to GGID audit service for analytics

## Best Practices

1. **Never store secrets in app code**: Use PKCE, not client secrets
2. **Use hardware-backed security**: Secure Enclave (iOS), StrongBox Keystore (Android)
3. **Implement defense in depth**: TLS + pinning + secure storage + biometric + attestation
4. **Plan for key rotation**: Certificate pins, signing keys, and attestation keys all rotate
5. **Test on real devices**: Emulators don't have biometrics or secure hardware
6. **Handle edge cases**: No biometrics enrolled, biometric changed, device not attested
7. **Provide fallback**: Always have a non-biometric fallback (passcode, password, TOTP)
8. **Respect user privacy**: Don't collect unnecessary device data
9. **Follow platform guidelines**: Apple Human Interface Guidelines, Material Design for auth flows
10. **Keep dependencies updated**: Mobile libraries have frequent security patches

## See Also

- [OAuth 2.1 Implementation Guide](./oauth-2-1-implementation.md)
- [WebAuthn Deployment Guide](./webauthn-deployment-guide.md)
- [MFA Implementation Guide](./mfa-implementation-guide.md)
- [API Rate Limiting Strategy](./api-rate-limiting-strategy.md)
- [Token Lifecycle Design](./token-lifecycle-design.md)