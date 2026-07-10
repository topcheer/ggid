# Mobile Biometric Authentication for IAM Systems

> Research document for GGID — Go-based multi-tenant IAM platform.

---

## 1. WebAuthn Platform Authenticators

### 1.1 How Touch ID / Face ID Work

A **platform authenticator** is a WebAuthn/FIDO2 authenticator built into the
client device. Unlike roaming authenticators (YubiKey), platform authenticators
are **device-bound**: the private key never leaves the secure enclave / TEE,
the authenticator enforces a local biometric/PIN check before signing (the UV
flag in authenticator data), and the credential lives on one device.

| Platform | Authenticator | Modality |
|----------|--------------|----------|
| macOS | Touch ID | Fingerprint |
| iOS | Face ID / Touch ID | Face / Fingerprint |
| Android | Fingerprint / Face Unlock | Fingerprint / Face |

Apple iCloud Keychain sync makes "sync fabric passkeys" — still platform
authenticators but the private key syncs across devices in the same account.

### 1.2 Assertion Flow

```
Client                  GGID Server            Authenticator
  │ 1. POST /webauthn/auth/options                │
  ├──────────────────────►│ 2. challenge, allowCreds, UV:"required"
  │◄──────────────────────┤                        │
  │ 3. credentials.get({ challenge })              │
  ├──────────────────────────────────────────────►│ 4. Biometric prompt (UV)
  │ 5. assertion = { authData, clientData, sig }   │
  │◄──────────────────────────────────────────────┤
  │ 6. POST /webauthn/auth/verify                  │
  ├──────────────────────►│ 7. Verify sig, check UV flag, check counter
  │                       │ 8. Issue JWT / session
  │◄──────────────────────┤
```

### 1.3 Attestation vs Assertion

Attestation (registration) proves authenticator provenance and may leak device
model via AAGUID. Assertion (authentication) proves user possession — no device
model info, only a counter. GGID already supports attestation verification in
`services/auth/internal/webauthn/attestation.go` (packed/none/fido-u2f with
ES256, RS256, EdDSA).

### 1.4 Why Platform Authenticators Are Device-Bound

The private key is generated **inside** the Secure Enclave (Apple) or TEE
(Android StrongBox). It never exists in application memory in plaintext, is
marked non-exportable, and is accessible only through a hardware-verified
biometric path. Even a compromised OS kernel cannot extract the raw key.

### 1.5 Go Code: Verifying a Platform Authenticator Assertion

```go
package biometric

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-webauthn/webauthn/protocol"
)

func VerifyPlatformAssertion(input VerifyAssertionInput, storedPubKey []byte,
	expectedOrigin, expectedRPID string) error {
	// 1. Decode client data — verify type is "webauthn.get" and origin matches.
	clientDataBytes, _ := base64.RawURLEncoding.DecodeString(input.ClientDataJSON)
	var cd protocol.CollectedClientData
	json.Unmarshal(clientDataBytes, &cd)
	if cd.Type != "webauthn.get" || cd.Origin != expectedOrigin {
		return fmt.Errorf("invalid client data")
	}

	// 2. Parse authenticator data — UV flag MUST be set for biometric auth.
	authDataBytes, _ := base64.RawURLEncoding.DecodeString(input.AuthenticatorData)
	authData, err := protocol.ParseAuthenticatorData(authDataBytes)
	if err != nil || !authData.Flags.UserVerified {
		return fmt.Errorf("user verification required")
	}

	// 3. Verify signature over authData || SHA256(clientDataJSON).
	sigBytes, _ := base64.RawURLEncoding.DecodeString(input.Signature)
	clientDataHash := sha256.Sum256(clientDataBytes)
	signedData := append(authDataBytes, clientDataHash[:]...)
	// GGID's attestation.go already has ECDSA/RSA/EdDSA verification logic.
	if err := verifyCOSESignature(storedPubKey, signedData, sigBytes); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// 4. Check RP ID hash.
	expectedRPIDHash := sha256.Sum256([]byte(expectedRPID))
	if !bytes.Equal(authData.RPIDHash, expectedRPIDHash[:]) {
		return fmt.Errorf("RP ID hash mismatch")
	}
	return nil
}
```

---

## 2. Android BiometricPrompt Integration

`BiometricPrompt` (Android 9+) is the unified biometric API handling fingerprint,
face, and iris. The system renders the prompt, preventing UI spoofing.
`CryptoObject` binds authentication to a cryptographic operation — the key is
only usable after biometric verification.

```kotlin
// Generate an ECDSA P-256 key in Android Keystore (TEE/StrongBox).
fun generateBiometricKey(): KeyPair {
    val gen = KeyPairGenerator.getInstance("EC", "AndroidKeyStore")
    gen.initialize(
        KeyGenParameterSpec.Builder("ggid_biometric_key",
            KeyProperties.PURPOSE_SIGN or KeyProperties.PURPOSE_VERIFY)
            .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
            .setDigests(KeyProperties.DIGEST_SHA256)
            .setUserAuthenticationRequired(true)
            .setIsStrongBoxBacked(true)           // hardware-backed if available
            .setInvalidatedByBiometricEnrollment(true)
            .build()
    )
    return gen.generateKeyPair()
}

// Trigger biometric prompt with a locked Signature object.
fun authenticate(activity: FragmentActivity, challenge: ByteArray,
                 onSuccess: (ByteArray) -> Unit, onError: (String) -> Unit) {
    val ks = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
    val sig = Signature.getInstance("SHA256withECDSA")
    sig.initSign(ks.getKey("ggid_biometric_key", null) as PrivateKey)

    val prompt = BiometricPrompt(activity, ContextCompat.getMainExecutor(activity),
        object : BiometricPrompt.AuthenticationCallback() {
            override fun onAuthenticationSucceeded(r: BiometricPrompt.AuthenticationResult) {
                val unlocked = r.cryptoObject!!.signature  // unlocked only after biometric
                unlocked.update(challenge)
                onSuccess(unlocked.sign())
            }
            override fun onAuthenticationError(code: Int, err: CharSequence) = onError(err.toString())
        })
    prompt.authenticate(
        BiometricPrompt.PromptInfo.Builder()
            .setTitle("Authenticate to GGID")
            .setNegativeButtonText("Cancel")
            .setAllowedAuthenticators(BiometricManager.Authenticators.BIOMETRIC_STRONG)
            .build(),
        BiometricPrompt.CryptoObject(sig)
    )
}
```

The critical property: `CryptoObject` ties the key unlock to the specific
biometric event. If the prompt fails, the `Signature` remains locked.

---

## 3. iOS LocalAuthentication & Secure Enclave

```swift
import LocalAuthentication
import CryptoKit

class BiometricAuth {
    // ECDSA P-256 key generated in the Secure Enclave — non-extractable.
    var sepKey: P256.Signing.PrivateKey {
        try! P256.Signing.PrivateKey(
            secureEnclave: .init(),
            accessControl: SecAccessControl(
                kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
                [.userPresence, .biometryCurrentSet]  // invalidated on re-enrollment
            )!
        )
    }

    func authenticate(challenge: Data, completion: @escaping (Result<Data, Error>) -> Void) {
        let context = LAContext()
        guard context.canEvaluatePolicy(.deviceOwnerAuthenticationWithBiometrics, error: nil) else {
            completion(.failure(BiometricError.notAvailable)); return
        }
        context.evaluatePolicy(.deviceOwnerAuthenticationWithBiometrics,
            localizedReason: "Authenticate to GGID") { success, err in
            guard success else { completion(.failure(err!)); return }
            let sig = try? self.sepKey.signature(for: challenge)
            completion(sig.map { .success($0.derRepresentation) }
                       ?? .failure(BiometricError.signingFailed))
        }
    }
}
```

The Secure Enclave is a dedicated hardware subsystem with its own crypto engine,
isolated from the main CPU. Keys are non-extractable — key material never leaves
the SEP even from a compromised kernel. Biometric template comparison happens
inside the SEP; the OS only receives a boolean result.

---

## 4. Device Attestation

### 4.1 Android Play Integrity API (replaces SafetyNet)

| Verdict | Meaning |
|---------|---------|
| **App Integrity** | Genuine Play-signed binary? Detects repackaged APKs. |
| **Device Integrity** | Genuine Android with Play Services? Detects rooted/emulated. Values: `MEETS_DEVICE_INTEGRITY`, `MEETS_STRONG_INTEGRITY`. |
| **Account Details** | Google account licensed/approved. |

### 4.2 iOS App Attest

App generates a key in the Secure Enclave; Apple countersigns attestations.
Flow: `generateKey()` → `attestation(keyId:)` (verify chain with Apple root CA
server-side) → `generateAssertion(clientDataHash:)` per authentication.

### 4.3 Go: Verifying Play Integrity Token

```go
func VerifyPlayIntegrity(ctx context.Context, token string) error {
	body, _ := json.Marshal(map[string]string{"integrity_token": token})
	req, _ := http.NewRequestWithContext(ctx, "POST",
		"https://playintegrity.googleapis.com/v1/decodeIntegrityToken",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return fmt.Errorf("call play integrity: %w", err) }
	defer resp.Body.Close()

	var result struct {
		AppIntegrity struct{ AppRecognitionVerdict string } `json:"appIntegrity"`
		DeviceIntegrity struct{ DeviceRecognitionVerdict []string } `json:"deviceIntegrity"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.AppIntegrity.AppRecognitionVerdict != "PLAY_RECOGNIZED" {
		return fmt.Errorf("app integrity failed")
	}
	for _, v := range result.DeviceIntegrity.DeviceRecognitionVerdict {
		if v == "MEETS_DEVICE_INTEGRITY" || v == "MEETS_STRONG_INTEGRITY" {
			return nil // pass
		}
	}
	return fmt.Errorf("device integrity failed")
}
```

### 4.4 Go: Verifying App Attest Assertion

```go
func VerifyAppAttestAssertion(pubKey *ecdsa.PublicKey, signedData, sig []byte) error {
	hash := sha256.Sum256(signedData)
	if !ecdsa.VerifyASN1(pubKey, hash[:], sig) {
		return fmt.Errorf("app attest assertion verification failed")
	}
	return nil
}
```

---

## 5. Secure Enclave / TEE Key Storage

### 5.1 Hardware-Backed Architecture

```
┌── Rich OS (REE) ──────────────────────┐
│ App → Keymaster HAL proxy              │
├── Secure World / TEE ──────────────────┤
│ Keymaster TA / StrongBox (Titan M)     │
│  • Device-Unique Key (fused at factory)│
│  • Biometric Template Data             │
│  • Private Keys (non-exportable)       │
└────────────────────────────────────────┘
```

### 5.2 Why TEE/SEP Prevents Key Extraction

1. **Hardware isolation**: TEE runs in a separate CPU mode (ARM TrustZone /
   Apple SEP). The rich OS has no direct memory access.
2. **Non-exportable keys**: Keys are wrapped with a device-unique hardware key.
3. **Auth-gated**: Key usage requires a hardware-verified biometric match.
   Biometric comparison happens inside the TEE — the OS only receives a boolean.
4. **Key attestation certs**: The TEE emits a certificate chain proving a key
   was generated in hardware, signed by the manufacturer's root CA.

### 5.3 GGID's Crypto Package

GGID's `pkg/crypto/crypto.go` provides Argon2id hashing, AES-256-GCM encryption,
and CSPRNG token generation. The WebAuthn attestation code
(`services/auth/internal/webauthn/attestation.go`) already handles ES256/RS256/
EdDSA signature verification and COSE algorithm dispatch via
`VerifyPackedAttestation()`.

**Gap**: No generalized functions for verifying mobile-issued biometric
assertions, no COSE/CBOR key parsing in `pkg/crypto`, and no Play Integrity /
App Attest token decoding. The existing attestation verification logic could be
extracted into `pkg/crypto` and generalized.

---

## 6. GGID Mobile SDK Design

### 6.1 Architecture

```
┌────────── Mobile App ──────────────────────────┐
│  ┌──────────── GGID Mobile SDK ──────────────┐ │
│  │ Biometric Mgr │ Token Mgr │ Device Attest │ │ │
│  │  iOS SEP key  │  PKCE     │  Play Integ.  │ │ │
│  │  Android KSt  │  JWT Store│  App Attest   │ │ │
│  │           HTTP Client (TLS pinning)        │ │
│  └───────────────────┬────────────────────────┘ │
└──────────────────────┼──────────────────────────┘
                       │ HTTPS
┌──────────────────────▼──────────────────────────┐
│           GGID Gateway (:8080)                   │
└──────────────┬──────────────┬───────────────────┘
┌──────────────▼────┐ ┌──────▼──────┐ ┌────────────┐
│ Auth (:9001)      │ │ OAuth (:9005│ │ Identity   │
│ WebAuthn Handler  │ │ PKCE Flow   │ │ (:8081)    │
│ + NEW:            │ │ + NEW:      │ │ User CRUD  │
│  /biometric/      │ │ grant_type: │ │            │
│   register/verify │ │ urn:ggid:bio│ │            │
└───────────────────┘ └─────────────┘ └────────────┘
```

### 6.2 OAuth Integration

GGID's OAuth service already supports PKCE and public clients
(`AuthorizationCode.ValidatePKCE` in `domain/models.go`). The SDK adds a new
grant type for biometric login:

```go
type BiometricTokenRequest struct {
	GrantType      string `json:"grant_type"`       // urn:ggid:params:oauth:grant-type:biometric
	ClientID       string `json:"client_id"`        // public client, no secret
	CredentialID   string `json:"credential_id"`    // WebAuthn credential ID
	Assertion      struct {
		AuthenticatorData string `json:"authenticator_data"`
		ClientDataJSON    string `json:"client_data_json"`
		Signature         string `json:"signature"`
	} `json:"assertion"`
	IntegrityToken string `json:"integrity_token"`  // Play Integrity / App Attest
	CodeVerifier   string `json:"code_verifier"`    // PKCE
}
```

The flow: (1) initial registration via standard OAuth code + PKCE, registering
a biometric credential through the Auth service; (2) biometric login via the new
grant type — GGID verifies assertion with stored public key, checks device
integrity, validates PKCE, issues tokens.

Tokens stored in Keychain (`kSecAttrAccessibleWhenUnlockedThisDeviceOnly`)
on iOS and EncryptedSharedPreferences on Android. GGID's `crypto.AESEncrypt`
can be reused server-side for encrypted token caching.

---

## 7. Gap Analysis & Recommendations

### 7.1 Current State

| Capability | Status |
|-----------|--------|
| WebAuthn registration + attestation verification | **Implemented** |
| OAuth2 PKCE + public clients | **Implemented** |
| AES-256-GCM, Argon2id, constant-time compare | **Implemented** |
| Mobile SDK (iOS/Android) | **Missing** |
| Biometric assertion REST endpoint for mobile | **Missing** |
| Custom `urn:ggid:...:biometric` grant type | **Missing** |
| Play Integrity / App Attest verification | **Missing** |
| Device-bound refresh tokens (`device_id` on `RefreshTokenRecord`) | **Missing** |

### 7.2 Implementation Roadmap

| # | Action Item | Effort | Priority |
|---|------------|--------|----------|
| 1 | **Biometric assertion endpoint** — New `POST /api/v1/biometric/verify` in Auth service. Extract COSE signature verification from `attestation.go` into `pkg/crypto`. | 3-5 days | P0 |
| 2 | **`urn:ggid:...:biometric` OAuth grant** — Extend token endpoint. Add `device_id` to `RefreshTokenRecord` for device-bound tokens. Enforce PKCE. | 3-5 days | P0 |
| 3 | **Play Integrity + App Attest verification** — New `pkg/attestation/` package. Store attested device keys in `device_credentials` table. Cache verdicts in Redis. | 5-7 days | P1 |
| 4 | **GGID Mobile SDK** — Swift + Kotlin SDKs: biometric key generation (SEP/Keystore), OAuth PKCE, token storage. Swift Package + Maven. | 10-15 days | P1 |
| 5 | **Device management API** — `GET/DELETE /api/v1/devices`. Device fingerprint metadata. Console UI. | 3-5 days | P2 |

### 7.3 Security Considerations

- **Replay protection**: Single-use challenges in Redis (30-60s TTL). Migrate
  WebAuthn's in-memory `sessionStore` to Redis.
- **Fallback**: SDK must fall back to password + TOTP if biometric fails, never
  silently skip.
- **Key invalidation**: `setInvalidatedByBiometricEnrollment(true)` (Android)
  and `.biometryCurrentSet` (iOS) to force re-registration on biometric changes.
- **Tenant isolation**: All biometric credential storage must include `tenant_id`.

---

## References

- W3C WebAuthn L2: https://www.w3.org/TR/webauthn-2/
- Android BiometricPrompt: https://developer.android.com/training/sign-in/biometric-auth
- Android Play Integrity: https://developer.android.com/google/play/integrity
- iOS LocalAuthentication: https://developer.apple.com/documentation/localauthentication
- iOS App Attest: https://developer.apple.com/documentation/devicecheck/appattest
- GGID source: `services/auth/internal/webauthn/`, `services/oauth/internal/service/`, `pkg/crypto/`
