# Certificate Pinning Guide

This guide covers pinning strategies, pin locations, pin rotation, TOFU vs preload, HPKP deprecation, mobile app pinning, service-to-service mTLS pinning, pin failure handling, and GGID's pinning strategy.

## Overview

Certificate pinning associates a host with its expected certificate or public key. When a pinned certificate doesn't match, the connection is rejected, preventing man-in-the-middle (MITM) attacks even if the attacker has a trusted CA certificate.

## Pinning Strategies

### 1. SPKI (Subject Public Key Info) Pin

Pins the hash of the public key (recommended):

```
pin-sha256="sha256/base64-of-spki"
```

```go
func computeSPKIHash(cert *x509.Certificate) string {
    spki := cert.RawSubjectPublicKeyInfo
    hash := sha256.Sum256(spki)
    return base64.StdEncoding.EncodeToString(hash[:])
}
```

### 2. Certificate Pin

Pins the full certificate hash:

```
pin-sha256="sha256/base64-of-full-cert"
```

**Not recommended** — breaks on certificate renewal even with same key.

### 3. Public Key Pin

Pins the raw public key:

```
pin-sha256="sha256/base64-of-public-key"
```

### Strategy Comparison

| Strategy | Rotation Survivable? | Security | Complexity |
|---|---|---|---|
| SPKI hash | Yes (same key, new cert) | High | Low |
| Certificate hash | No | High | Low |
| Public key hash | Yes | High | Medium |
| CA pin | Yes (any cert from CA) | Low | Low |

**Recommendation**: Use SPKI pinning — survives certificate renewal while maintaining security.

## Pin Locations

### In Code (Mobile Apps)

```swift
// iOS - URLSession delegate
func urlSession(_ session: URLSession,
                didReceive challenge: URLAuthenticationChallenge,
                completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
    guard let trust = challenge.protectionSpace.serverTrust,
          let cert = SecTrustGetCertificateAtIndex(trust, 0) else {
        completionHandler(.cancelAuthenticationChallenge, nil)
        return
    }

    let serverSPKI = computeSPKIHash(cert)
    let pinnedSPKIs = ["sha256/base64-of-expected-spki"]

    if pinnedSPKIs.contains(serverSPKI) {
        completionHandler(.useCredential, URLCredential(trust: trust))
    } else {
        completionHandler(.cancelAuthenticationChallenge, nil)
    }
}
```

```kotlin
// Android - OkHttp CertificatePinner
val pinner = CertificatePinner.Builder()
    .add("auth.ggid.example.com", "sha256/base64-of-spki")
    .add("auth.ggid.example.com", "sha256/backup-spki")  // Backup pin
    .build()

val client = OkHttpClient.Builder()
    .certificatePinner(pinner)
    .build()
```

### In Configuration (Services)

```yaml
certificate_pinning:
  hosts:
    - host: "auth.ggid.example.com"
      pins:
        - "sha256/primary-spki-hash"
        - "sha256/backup-spki-hash"
    - host: "api.ggid.example.com"
      pins:
        - "sha256/primary-spki-hash"
        - "sha256/backup-spki-hash"
```

### HTTP Header (HPKP — Deprecated)

```
Public-Key-Pins: pin-sha256="base64=="; max-age=5184000; includeSubDomains
```

**Note**: HPKP is deprecated due to pinning attacks (attacker pins their key). Do not use for web browsers.

## Pin Rotation

### Backup Pin Strategy

Always include a backup pin during deployment:

```
Primary pin: sha256/current-spki
Backup pin:  sha256/future-spki  (pre-generated key pair)
```

### Rotation Process

```
1. Generate new key pair (backup key)
2. Deploy with both pins: [current, backup]
3. Wait for all clients to update (max-age period)
4. Switch to new certificate (using backup key)
5. Generate new backup key pair
6. Deploy with both pins: [new-current, new-backup]
7. Remove old pin after next deployment cycle
```

### Rotation Timeline

```
Day 0:   Deploy pins [A, B]  (A=current, B=backup)
Day 30:  All clients have both pins
Day 60:  Switch to cert B, generate C
Day 90:  Deploy pins [B, C]
Day 120: Remove pin A from config
```

```yaml
certificate_pinning:
  rotation:
    overlap_period: 30d  # Both pins valid for 30 days
    pre_generate_backup: true  # Generate backup key in advance
    max_age: 5184000  # 60 days for HTTP header
```

## TOFU vs Preload

### TOFU (Trust On First Use)

Client pins the certificate on first connection:

```go
func tofuPinning(host string, cert *x509.Certificate) error {
    existingPin := getStoredPin(host)
    if existingPin == "" {
        // First use — pin this certificate
        storePin(host, computeSPKIHash(cert))
        return nil
    }
    // Subsequent use — verify
    if computeSPKIHash(cert) != existingPin {
        return ErrPinMismatch
    }
    return nil
}
```

**Risk**: First connection is vulnerable to MITM.

### Preload List

Pins are compiled into the application:

```go
var preloadPins = map[string][]string{
    "auth.ggid.example.com": {
        "sha256/primary-spki",
        "sha256/backup-spki",
    },
}
```

**Benefit**: No TOFU vulnerability. Pins are known before first connection.

### Comparison

| Approach | First-Use Security | Update Flexibility | Complexity |
|---|---|---|---|
| TOFU | Vulnerable | Easy (auto-update) | Low |
| Preload | Secure | Requires app update | High |
| Hybrid | Secure + flexible | Config-driven pins | Medium |

**Recommendation**: Use preload for mobile apps, config-driven for services.

## HPKP Deprecation

### Why HPKP Was Deprecated

1. **Pinning attacks**: Attacker with a valid CA cert could pin their key, locking out the real site
2. **Brick attacks**: Malicious pins could make sites inaccessible
3. **No recovery**: If private key lost and no backup pin, site is bricked
4. **No user awareness**: Users didn't understand pin warnings

### What to Use Instead

| Context | Alternative |
|---|---|
| Web browsers | No pinning (rely on CT logs) |
| Mobile apps | In-code pinning |
| Service-to-service | mTLS with cert verification |
| API clients | SPKI pinning in client config |

## Mobile App Pinning

### iOS (Swift)

```swift
class PinningDelegate: NSObject, URLSessionDelegate {
    let pinnedSPKIs: Set<String> = [
        "sha256/primary-spki",
        "sha256/backup-spki"
    ]

    func urlSession(_ session: URLSession,
                    didReceive challenge: URLAuthenticationChallenge,
                    completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
        guard challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodServerTrust,
              let serverTrust = challenge.protectionSpace.serverTrust else {
            completionHandler(.cancelAuthenticationChallenge, nil)
            return
        }

        // Evaluate trust chain first
        var error: CFError?
        guard SecTrustEvaluateWithError(serverTrust, &error) else {
            completionHandler(.cancelAuthenticationChallenge, nil)
            return
        }

        // Check pin
        let cert = SecTrustGetCertificateAtIndex(serverTrust, 0)!
        let spkiHash = computeSPKIHash(cert)

        if pinnedSPKIs.contains(spkiHash) {
            completionHandler(.useCredential, URLCredential(trust: serverTrust))
        } else {
            completionHandler(.cancelAuthenticationChallenge, nil)
        }
    }
}
```

### Android (Kotlin)

```kotlin
class PinningInterceptor : Interceptor {
    private val pins = mapOf(
        "auth.ggid.example.com" to listOf("sha256/primary", "sha256/backup")
    )

    override fun intercept(chain: Interceptor.Chain): Response {
        val request = chain.request()
        val host = request.url.host

        val response = chain.proceed(request)

        // Verify pin
        val certs = response.handshake?.peerCertificates ?: emptyList()
        val pinnedHashes = pins[host] ?: return response

        val matched = certs.any { cert ->
            val spki = computeSPKIHash(cert)
            pinnedHashes.contains("sha256/$spki")
        }

        if (!matched) {
            response.close()
            throw SSLPeerUnverifiedException("Certificate pinning failure for $host")
        }

        return response
    }
}
```

## Service-to-Service mTLS Pinning

### Pinning in gRPC

```go
func loadMTLSCredentialsWithPinning(certFile, keyFile, caFile string, pins map[string][]string) (credentials.TransportCredentials, error) {
    cert, _ := tls.LoadX509KeyPair(certFile, keyFile)
    caCert, _ := os.ReadFile(caFile)
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)

    creds := credentials.NewTLS(&tls.Config{
        Certificates: []tls.Certificate{cert},
        RootCAs:      caPool,
        ClientCAs:    caPool,
        ClientAuth:   tls.RequireAndVerifyClientCert,
        VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
            // Pin verification
            if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
                return errors.New("no verified chain")
            }
            cert := verifiedChains[0][0]
            spkiHash := computeSPKIHash(cert)

            // Check against pins for the peer
            // pins keyed by expected hostname or service name
            for _, pinSet := range pins {
                for _, pin := range pinSet {
                    if "sha256/"+spkiHash == pin {
                        return nil  // Pin matches
                    }
                }
            }
            return errors.New("certificate pin verification failed")
        },
    })

    return creds, nil
}
```

### Configuration

```yaml
grpc:
  mtls:
    enabled: true
    cert: /etc/ggid/tls/service.pem
    key: /etc/ggid/tls/service-key.pem
    ca: /etc/ggid/tls/ca.pem
    pinning:
      enabled: true
      services:
        auth:
          pins: ["sha256/auth-service-spki"]
        policy:
          pins: ["sha256/policy-service-spki"]
        audit:
          pins: ["sha256/audit-service-spki"]
```

## Pin Failure Handling

### Hard Fail (Recommended for Security)

```go
func pinVerification(cert *x509.Certificate, pins []string) error {
    spki := computeSPKIHash(cert)
    for _, pin := range pins {
        if "sha256/"+spki == pin {
            return nil
        }
    }
    // Hard fail: reject connection
    log.Security("certificate pin failure", "spki", spki)
    return ErrPinVerificationFailed
}
```

### Soft Fail (Fallback)

```go
func softFailPinning(cert *x509.Certificate, pins []string) error {
    spki := computeSPKIHash(cert)
    for _, pin := range pins {
        if "sha256/"+spki == pin {
            return nil
        }
    }
    // Soft fail: log but allow
    log.Security("certificate pin mismatch (soft fail)", "spki", spki)
    return nil  // Allow connection
}
```

### Comparison

| Strategy | Security | Availability | Use Case |
|---|---|---|---|
| Hard fail | Maximum | May break on rotation | Production, high-security |
| Soft fail | Reduced | High | Development, staging |

**Recommendation**: Hard fail in production, soft fail in development only.

## GGID Pinning Strategy

### Configuration

```yaml
certificate_pinning:
  enabled: true
  strategy: "spki"  # SPKI hash pinning
  locations:
    - type: "config"
      path: "/etc/ggid/pins/pins.yaml"
  failure_handling:
    production: "hard_fail"
    staging: "soft_fail"
    development: "disabled"
  rotation:
    overlap_period: 30d
    pre_generate_backup: true
    max_pins_per_host: 3  # Current + 2 backups

  mobile:
    ios: "in_code"
    android: "okhttp_certificate_pinner"
    preload: true

  service_to_service:
    enabled: true
    mtls: true
    pin_verification: "hard_fail"
```

### Pin Management API

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/admin/pins` | GET | List all pins |
| `/api/v1/admin/pins/{host}` | GET | Get pins for host |
| `/api/v1/admin/pins/{host}` | POST | Add pin for host |
| `/api/v1/admin/pins/{host}/{pin}` | DELETE | Remove a pin |

## Best Practices

1. **Always have backup pins** — Prevents bricking on key rotation
2. **Use SPKI pinning** — Survives certificate renewal with same key
3. **Pre-generate backup keys** — Don't wait until rotation to create backup
4. **Hard fail in production** — Don't silently allow pin mismatches
5. **Don't use HPKP for browsers** — Deprecated, use CT logs instead
6. **Pin in mobile apps** — Apps are high-risk for MITM
7. **Pin service-to-service** — mTLS with pinning prevents internal MITM
8. **Monitor pin failures** — Track and alert on mismatch events
9. **Plan rotation carefully** — Overlap period prevents downtime
10. **Test before deploying** — Verify pins work in staging first