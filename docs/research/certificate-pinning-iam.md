# TLS Certificate Pinning and Trust Management for IAM Systems

> Research document for GGID — a Go-based Identity & Access Management platform.
> Covers certificate pinning strategies, IdP trust management, mTLS, JWKS
> validation, and a concrete audit of GGID's current cryptographic trust model.

---

## Table of Contents

1. [Certificate Pinning Concepts](#1-certificate-pinning-concepts)
2. [SAML SP Metadata Signing Cert Pinning](#2-saml-sp-metadata-signing-cert-pinning)
3. [IdP Certificate Rotation](#3-idp-certificate-rotation)
4. [mTLS Client Certificate Validation](#4-mtls-client-certificate-validation)
5. [Trust Chain Verification](#5-trust-chain-verification)
6. [JWKS Key Pinning](#6-jwks-key-pinning)
7. [GGID SAML/OIDC Certificate Handling Audit](#7-ggid-samloidc-certificate-handling-audit)
8. [Gap Analysis and Recommendations](#8-gap-analysis-and-recommendations)

---

## 1. Certificate Pinning Concepts

### 1.1 What Is Certificate Pinning?

Certificate pinning is the practice of hard-coding or dynamically caching a
known-good cryptographic identity for a remote endpoint, then rejecting any
certificate that does not match the pin even if the certificate chains to a
trusted root CA. The goal is to defeat man-in-the-middle (MITM) attacks that
rely on a compromised or maliciously issued CA certificate.

For IAM systems, the stakes are uniquely high. A successful MITM against an
authentication flow allows the attacker to intercept credentials, session
tokens, SAML assertions, and authorization codes — the full identity lifecycle.
Pinning the IdP/SP certificates adds a defense-in-depth layer that survives CA
compromise.

### 1.2 SPKI Pinning vs Full Certificate Pinning

**Full certificate pinning** compares the entire DER-encoded leaf certificate
against a stored copy. This is brittle: any re-issuance (even with the same key)
invalidates the pin.

**SPKI (Subject Public Key Info) pinning** hashes only the public key portion of
the certificate:

```go
func spkiPin(cert *x509.Certificate) (string, error) {
    spkiDER, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
    if err != nil {
        return "", fmt.Errorf("marshal SPKI: %w", err)
    }
    sum := sha256.Sum256(spkiDER)
    return base64.StdEncoding.EncodeToString(sum[:]), nil
}
```

SPKI pins survive certificate renewal as long as the key pair is reused. This
is the approach used by HTTP Public Key Pinning (HPKP, RFC 7469) and TACK.

### 1.3 Static Pins vs Dynamic Pin Discovery

| Approach | Description | Risk |
|----------|-------------|------|
| **Static pins** | Hard-coded pins shipped in the binary/config | If the key rotates and pins are not updated, connections break |
| **Dynamic (TOFU)** | Trust on first use: pin the cert seen on first connection | Vulnerable to MITM on the first connection |
| **Backup pins** | Ship a primary + one or more backup pins | Reduces bricking risk during rotation |

Best practice for IAM: ship static pins with at least one backup pin, and rotate
backup pins proactively before they are needed.

### 1.4 Risks of Pinning (Bricking)

The most notorious risk is **certificate bricking**: if the pinned certificate
or key is lost or rotated without updating the pin, the client cannot connect.
HPKP was deprecated largely because misconfigured pins locked users out of
their own services.

Mitigations:
- Always include a backup pin pointing to a different key.
- Set reasonable max-age values for dynamic pins.
- Use a test rollout before enforcing pins in production.
- For IAM systems, prefer SPKI pinning over full-cert pinning.

### 1.5 Go Code: TLS Dial with Certificate Pinning

```go
package tlspin

import (
    "crypto/sha256"
    "crypto/tls"
    "crypto/x509"
    "encoding/base64"
    "fmt"
    "net"
)

// PinValidator holds a set of allowed SPKI SHA-256 pins.
type PinValidator struct {
    pins map[string]bool // base64-encoded SHA-256 of SPKI DER
}

func NewPinValidator(pins []string) *PinValidator {
    m := make(map[string]bool, len(pins))
    for _, p := range pins {
        m[p] = true
    }
    return &PinValidator{pins: m}
}

// VerifyConn checks that at least one certificate in the peer chain
// matches a known SPKI pin.
func (pv *PinValidator) VerifyConn(state tls.ConnectionState) error {
    for _, cert := range state.PeerCertificates {
        spkiDER, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
        if err != nil {
            continue
        }
        sum := sha256.Sum256(spkiDER)
        pinStr := base64.StdEncoding.EncodeToString(sum[:])
        if pv.pins[pinStr] {
            return nil // Pin matched
        }
    }
    return fmt.Errorf("no certificate in chain matches any pinned SPKI")
}

// PinnedDialer returns a tls.Dial wrapper that enforces SPKI pinning
// in addition to standard chain validation.
func PinnedDialer(network, addr string, pv *PinValidator) (*tls.Conn, error) {
    rawConn, err := net.Dial(network, addr)
    if err != nil {
        return nil, err
    }
    host, _, _ := net.SplitHostPort(addr)
    tlsConn := tls.Client(rawConn, &tls.Config{
        ServerName:         host,
        InsecureSkipVerify: false, // Standard chain validation still runs
    })
    if err := tlsConn.Handshake(); err != nil {
        rawConn.Close()
        return nil, err
    }
    if err := pv.VerifyConn(tlsConn.ConnectionState()); err != nil {
        tlsConn.Close()
        return nil, err
    }
    return tlsConn, nil
}
```

---

## 2. SAML SP Metadata Signing Cert Pinning

### 2.1 Why Pinning IdP Signing Certificates Prevents MITM

In a SAML flow, the IdP signs assertions with its private key. The SP verifies
the signature using the IdP's public certificate. If an attacker can substitute
their own certificate (e.g., by tampering with metadata exchange or DNS
rebinding), they can forge assertions and impersonate any user.

Pinning the IdP signing certificate at the SP ensures that only the expected
certificate — or its successor during an authorized rotation window — is
accepted for assertion verification.

### 2.2 Metadata Refresh and Pin Rotation

SAML metadata contains KeyDescriptor elements with `use="signing"` certificates.
When the IdP rotates its signing key, it publishes the new certificate in
metadata ahead of time. The SP should:

1. Fetch metadata periodically (e.g., every 24h).
2. Accept assertions signed by **both** old and new certificates during the
   overlap window.
3. After the cutover date, reject assertions signed with the old certificate.

### 2.3 How GGID Stores SAML Certificates

GGID's SAML package (`pkg/saml/`) stores the SP signing certificate as a
DER-encoded byte slice in the `ServiceProvider.X509Certificate` field:

```go
// pkg/saml/sp.go
type ServiceProvider struct {
    EntityID            string
    ACSURL              string
    SLOURL              string
    X509Certificate     []byte // DER-encoded X.509 certificate
    WantAssertionsSigned bool
}
```

The `GenerateSPMetadata` function embeds this certificate in the SP's metadata
EntityDescriptor XML:

```go
// pkg/saml/sp.go — GenerateSPMetadata
meta.SPSSODescriptor.KeyDescriptor = []KeyDescriptor{
    {Use: "signing", KeyInfo: KeyInfoData{...}},
}
```

On the verification side, `VerifySignedAssertion` accepts a single `*x509.Certificate`:

```go
// pkg/saml/signed_assertion.go
func VerifySignedAssertion(rawXML []byte, cert *x509.Certificate) (*SAMLAssertion, error)
```

This single-cert model means there is **no built-in support for accepting
multiple certificates during a rotation window**.

### 2.4 Gap: No IdP Cert Store

GGID currently has no persistent store for IdP certificates. The SP metadata
generation is one-directional (producing SP metadata), but there is no
`IdPMetadata` parser or trust store that would allow automatic certificate
discovery and rotation.

---

## 3. IdP Certificate Rotation

### 3.1 Certificate Lifecycle Management

Trust relationships in federated identity depend on a well-managed certificate
lifecycle:

```
Phase 1: PUBLISH  — IdP publishes new cert in metadata alongside old cert
Phase 2: OVERLAP  — SP accepts assertions signed by either cert (weeks)
Phase 3: CUTOVER  — IdP stops signing with old key
Phase 4: REVOKE   — SP removes old cert from trust store
```

The overlap window is critical. If too short, some SPs will reject valid
assertions during the transition. If too long, a compromised old key remains
usable.

### 3.2 Key Rollover Timeline

```
Day 0:   IdP generates new key pair
Day 7:   IdP publishes both old and new cert in metadata
Day 14:  IdP starts signing with new key (old still accepted by SPs)
Day 28:  IdP stops publishing old cert in metadata
Day 35:  SPs purge old cert from trust store
```

### 3.3 Go Code: Multi-Cert Trust Validation

```go
package samltrust

import (
    "crypto/x509"
    "fmt"
    "sync"
    "time"
)

// CertEntry represents a trusted IdP signing certificate with metadata.
type CertEntry struct {
    Cert      *x509.Certificate
    AddedAt   time.Time
    ExpiresAt time.Time // When the SP should stop accepting this cert
    Active    bool      // Whether the IdP is currently signing with this key
}

// TrustStore manages multiple IdP signing certificates for rotation.
type TrustStore struct {
    mu    sync.RWMutex
    certs map[string]*CertEntry // fingerprint → entry
}

func NewTrustStore() *TrustStore {
    return &TrustStore{certs: make(map[string]*CertEntry)}
}

// AddCert registers a new certificate in the trust store.
func (ts *TrustStore) AddCert(cert *x509.Certificate, overlap time.Duration) {
    ts.mu.Lock()
    defer ts.mu.Unlock()
    fp := certFingerprint(cert)
    ts.certs[fp] = &CertEntry{
        Cert:      cert,
        AddedAt:   time.Now(),
        ExpiresAt: time.Now().Add(overlap),
        Active:    true,
    }
}

// VerifyAgainstAny checks the assertion signature against all active certs.
func (ts *TrustStore) VerifyAgainstAny(verifyFn func(*x509.Certificate) error) error {
    ts.mu.RLock()
    defer ts.mu.RUnlock()
    now := time.Now()
    var lastErr error
    for _, entry := range ts.certs {
        if now.After(entry.ExpiresAt) {
            continue // Expired overlap window
        }
        if err := verifyFn(entry.Cert); err == nil {
            return nil // Signature verified
        } else {
            lastErr = err
        }
    }
    if lastErr != nil {
        return fmt.Errorf("no trusted certificate verified the assertion: %w", lastErr)
    }
    return fmt.Errorf("no active certificates in trust store")
}

// PurgeExpired removes certificates past their overlap window.
func (ts *TrustStore) PurgeExpired() int {
    ts.mu.Lock()
    defer ts.mu.Unlock()
    now := time.Now()
    purged := 0
    for fp, entry := range ts.certs {
        if now.After(entry.ExpiresAt) {
            delete(ts.certs, fp)
            purged++
        }
    }
    return purged
}

func certFingerprint(cert *x509.Certificate) string {
    return fmt.Sprintf("%X", cert.SerialNumber)
}
```

---

## 4. mTLS Client Certificate Validation

### 4.1 mTLS for Service-to-Service Auth

Mutual TLS (mTLS) authenticates both parties in a TLS handshake. In a
microservice architecture like GGID, mTLS ensures that only known services can
call each other directly, even if the network is compromised.

GGID's OAuth service already implements RFC 8705 sender-constrained tokens via
mTLS thumbprint binding (`services/oauth/internal/service/jar_mtls.go`):

```go
func ValidateMTLSClientAuth(claims jwt.MapClaims, certThumbprint string) error {
    cnf, ok := claims["cnf"].(map[string]any)
    x5t, _ := cnf["x5t#S256"].(string)
    if !strings.EqualFold(x5t, certThumbprint) {
        return fmt.Errorf("client certificate thumbprint mismatch")
    }
    return nil
}
```

However, this validates the **token binding**, not the TLS handshake itself.

### 4.2 Go Code: mTLS Server Configuration

```go
package mtls

import (
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "net/http"
    "os"
)

// NewMTLSServer creates an HTTP server that requires client certificate
// authentication.
func NewMTLSServer(addr, serverCertFile, serverKeyFile, clientCAFile string,
    handler http.Handler) (*http.Server, error) {

    // Load server certificate and key.
    serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
    if err != nil {
        return nil, fmt.Errorf("load server cert: %w", err)
    }

    // Load the CA that signs client certificates.
    clientCAPEM, err := os.ReadFile(clientCAFile)
    if err != nil {
        return nil, fmt.Errorf("read client CA: %w", err)
    }
    clientCAs := x509.NewCertPool()
    if !clientCAs.AppendCertsFromPEM(clientCAPEM) {
        return nil, fmt.Errorf("failed to parse client CA bundle")
    }

    tlsCfg := &tls.Config{
        Certificates: []tls.Certificate{serverCert},
        ClientAuth:   tls.RequireAndVerifyClientCert,
        ClientCAs:    clientCAs,
        MinVersion:   tls.VersionTLS13,
    }

    return &http.Server{
        Addr:      addr,
        Handler:   handler,
        TLSConfig: tlsCfg,
    }, nil
}
```

### 4.3 Go Code: mTLS Client

```go
// NewMTLSClient creates an HTTP client that presents a client certificate.
func NewMTLSClient(clientCertFile, clientKeyFile, serverCAFile string) (*http.Client, error) {
    clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
    if err != nil {
        return nil, fmt.Errorf("load client cert: %w", err)
    }

    serverCAPEM, err := os.ReadFile(serverCAFile)
    if err != nil {
        return nil, fmt.Errorf("read server CA: %w", err)
    }
    serverCAs := x509.NewCertPool()
    serverCAs.AppendCertsFromPEM(serverCAPEM)

    tlsCfg := &tls.Config{
        Certificates: []tls.Certificate{clientCert},
        RootCAs:      serverCAs,
        MinVersion:   tls.VersionTLS13,
    }

    return &http.Client{
        Transport: &http.Transport{TLSClientConfig: tlsCfg},
    }, nil
}
```

### 4.4 Certificate Revocation (CRL/OCSP)

mTLS should validate that client certificates have not been revoked:

```go
// VerifyWithRevocation adds CRL checking to standard chain validation.
func VerifyWithRevocation(cert *x509.Certificate, roots *x509.CertPool,
    crlURL string) error {

    // Fetch CRL.
    resp, err := http.Get(crlURL)
    if err != nil {
        return fmt.Errorf("fetch CRL: %w", err)
    }
    defer resp.Body.Close()

    crlDER, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("read CRL: %w", err)
    }

    crl, err := x509.ParseRevocationList(crlDER)
    if err != nil {
        return fmt.Errorf("parse CRL: %w", err)
    }

    // Check if the certificate is revoked.
    for _, revoked := range crl.RevokedCertificates {
        if cert.SerialNumber.Cmp(revoked.SerialNumber) == 0 {
            return fmt.Errorf("certificate serial %s is revoked", cert.SerialNumber)
        }
    }

    // Standard chain verification.
    opts := x509.VerifyOptions{Roots: roots}
    _, err = cert.Verify(opts)
    return err
}
```

For latency-sensitive paths, OCSP stapling (TLS extension) is preferred over
fetching CRLs on every connection. Go's `tls.Config` supports OCSP stapling
automatically when the server provides a stapled response.

---

## 5. Trust Chain Verification

### 5.1 Full Chain Validation

A TLS certificate chain consists of:

```
Root CA (self-signed, in trust store)
  └── Intermediate CA
        └── Leaf Certificate (server or client identity)
```

Go's `x509.Certificate.Verify()` performs full chain validation against the
system or custom root pool. However, IAM systems often need **custom
verification logic** to enforce organizational policies.

### 5.2 Pinning Intermediate CAs vs Root CAs

| Pin Level | Durability | Security Tradeoff |
|-----------|------------|-------------------|
| Root CA | Most durable — roots rotate every 10-20 years | Weakest pinning — any cert from that CA is accepted |
| Intermediate CA | Moderate — rotates every 1-5 years | Better — limits accepted certs to one intermediate's scope |
| Leaf/SPKI | Least durable — rotates annually | Strongest — pins exact identity |

For IAM trust relationships, **pinning intermediate CAs** offers the best
balance: it constrains which CA hierarchy can issue trusted certificates while
remaining durable through leaf certificate renewals.

### 5.3 Go Code: Custom Certificate Verification

```go
package chainverify

import (
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "fmt"
)

// ChainVerifier performs standard chain validation plus intermediate CA pinning.
type ChainVerifier struct {
    Roots         *x509.CertPool
    PinnedInterCA string // base64 SHA-256 of allowed intermediate's SPKI
}

func (cv *ChainVerifier) Verify(cert *x509.Certificate) error {
    // Step 1: Standard chain validation.
    chains, err := cert.Verify(x509.VerifyOptions{
        Roots: cv.Roots,
    })
    if err != nil {
        return fmt.Errorf("chain validation failed: %w", err)
    }

    // Step 2: Check that at least one chain includes the pinned intermediate.
    for _, chain := range chains {
        for _, c := range chain {
            if c.IsCA && !c.CheckSignatureFrom(c) { // Not self-signed (intermediate)
                spkiDER, _ := x509.MarshalPKIXPublicKey(c.PublicKey)
                sum := sha256.Sum256(spkiDER)
                pin := base64.StdEncoding.EncodeToString(sum[:])
                if pin == cv.PinnedInterCA {
                    return nil
                }
            }
        }
    }

    return fmt.Errorf("no chain contains the pinned intermediate CA")
}
```

### 5.4 Go Code: Custom VerifyPeerCertificate Callback

For TLS connections where you need pinning at the connection level:

```go
func pinnedVerifyPeer(pinnedSPKIs []string) func([][]byte, [][]*x509.Certificate) error {
    allowed := make(map[string]bool)
    for _, p := range pinnedSPKIs {
        allowed[p] = true
    }
    return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
        for _, rawCert := range rawCerts {
            cert, err := x509.ParseCertificate(rawCert)
            if err != nil {
                continue
            }
            spkiDER, _ := x509.MarshalPKIXPublicKey(cert.PublicKey)
            sum := sha256.Sum256(spkiDER)
            pin := base64.StdEncoding.EncodeToString(sum[:])
            if allowed[pin] {
                return nil
            }
        }
        return fmt.Errorf("peer certificate does not match any pinned SPKI")
    }
}
```

---

## 6. JWKS Key Pinning

### 6.1 The JWKS Substitution Attack

If an attacker can tamper with the JWKS endpoint response (via DNS spoofing, CA
compromise, or proxy injection), they can substitute their own public key. Any
JWT signed with the attacker's private key would then validate successfully.

### 6.2 Key ID (kid) Validation

Each JWKS key has a `kid` field. JWT headers reference the `kid` to indicate
which key was used for signing. Attacks include:

- **Key substitution**: Replace the key at a known `kid` with an attacker key.
- **Algorithm confusion**: Declare `alg: none` or switch from RS256 to HS256
  to abuse the public key as an HMAC secret.
- **Unknown kid**: Submit a JWT with a `kid` not in the JWKS to trigger
  fallback behavior.

### 6.3 Algorithm Confusion Prevention

GGID's OAuth service currently validates the signing method:

```go
// services/oauth/internal/service/oauth_service.go
token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
    if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return s.keyProvider.PublicKey(), nil
})
```

This correctly rejects non-RSA algorithms. The explicit `*jwt.SigningMethodRSA`
type assertion prevents HS256 confusion attacks.

### 6.4 Go Code: JWKS Fetching with Pinning

```go
package jwkspin

import (
    "crypto"
    "crypto/rsa"
    "crypto/sha256"
    "crypto/x509"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "math/big"
    "net/http"
    "time"
)

// PinnedJWKSClient fetches JWKS from a pinned endpoint, validating that
// the returned keys match expected SPKI hashes.
type PinnedJWKSClient struct {
    endpoint   string
    pinnedKeys map[string]string // kid → base64 SHA-256 of expected SPKI
    client     *http.Client
}

type jwksResponse struct {
    Keys []struct {
        KTY string `json:"kty"`
        Kid string `json:"kid"`
        Alg string `json:"alg"`
        N   string `json:"n"`
        E   string `json:"e"`
    } `json:"keys"`
}

func NewPinnedJWKSClient(endpoint string, pinnedKeys map[string]string) *PinnedJWKSClient {
    return &PinnedJWKSClient{
        endpoint:   endpoint,
        pinnedKeys: pinnedKeys,
        client:     &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *PinnedJWKSClient) FetchAndVerify() (map[string]*rsa.PublicKey, error) {
    resp, err := c.client.Get(c.endpoint)
    if err != nil {
        return nil, fmt.Errorf("fetch JWKS: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
    }

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read JWKS body: %w", err)
    }

    var jwks jwksResponse
    if err := json.Unmarshal(body, &jwks); err != nil {
        return nil, fmt.Errorf("parse JWKS: %w", err)
    }

    keys := make(map[string]*rsa.PublicKey)
    for _, jwk := range jwks.Keys {
        // Validate algorithm whitelist.
        if jwk.Alg != "RS256" {
            return nil, fmt.Errorf("key %s: unsupported alg %s", jwk.Kid, jwk.Alg)
        }

        // Validate kid is pinned.
        expectedPin, ok := c.pinnedKeys[jwk.Kid]
        if !ok {
            continue // Skip unknown keys rather than failing
        }

        // Reconstruct RSA public key from JWK.
        nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
        if err != nil {
            return nil, fmt.Errorf("decode n for kid %s: %w", jwk.Kid, err)
        }
        eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
        if err != nil {
            return nil, fmt.Errorf("decode e for kid %s: %w", jwk.Kid, err)
        }
        pubKey := &rsa.PublicKey{
            N: new(big.Int).SetBytes(nBytes),
            E: int(new(big.Int).SetBytes(eBytes).Int64()),
        }

        // Verify SPKI pin.
        spkiDER, err := x509.MarshalPKIXPublicKey(pubKey)
        if err != nil {
            return nil, fmt.Errorf("marshal SPKI for kid %s: %w", jwk.Kid, err)
        }
        sum := sha256.Sum256(spkiDER)
        actualPin := base64.StdEncoding.EncodeToString(sum[:])
        if actualPin != expectedPin {
            return nil, fmt.Errorf("key %s: SPKI pin mismatch (possible substitution)", jwk.Kid)
        }

        keys[jwk.Kid] = pubKey
    }

    if len(keys) == 0 {
        return nil, fmt.Errorf("no keys matched pinned entries")
    }

    return keys, nil
}

// VerifyJWT verifies a JWT signature against a pinned key set.
func VerifyJWT(tokenStr string, keys map[string]*rsa.PublicKey, kid string) error {
    key, ok := keys[kid]
    if !ok {
        return fmt.Errorf("unknown kid: %s", kid)
    }
    // In production, use jwt.ParseWithClaims with the selected key.
    _ = key
    _ = crypto.SHA256
    return nil
}
```

---

## 7. GGID SAML/OIDC Certificate Handling Audit

### 7.1 SAML Package (`pkg/saml/`)

**Strengths:**
- `VerifySignedAssertion` performs XML digital signature verification using
  RSA-PKCS1v15 and ECDSA.
- Supports SHA-1, SHA-256, SHA-384, SHA-512 digest algorithms.
- `VerifySignedAssertionWithDigest` adds digest verification as defense-in-depth.
- Uses constant-time comparison for digest verification
  (`constantTimeEqual`).
- SP metadata generation correctly embeds the signing certificate in KeyDescriptor.

**Gaps:**
1. **Single-certificate verification** — `VerifySignedAssertion` accepts one
   `*x509.Certificate`. No support for multi-cert trust during rotation.
2. **No certificate chain validation** — The cert is used raw for signature
   verification without checking expiry, revocation, or chain. A caller could
   pass an expired or revoked certificate.
3. **No IdP metadata parser** — GGID can generate SP metadata but cannot parse
   IdP metadata to discover signing certificates automatically.
4. **No metadata signature verification** — If metadata is fetched over HTTP,
   there is no verification of the metadata document's own XML signature.
5. **SHA-1 acceptance** — `hashForAlgorithm` and `cryptoHashForSignature`
  still accept SHA-1 algorithms. SHA-1 is collision-vulnerable; consider
  deprecating for production deployments.

### 7.2 OAuth/OIDC Service (`services/oauth/`)

**Strengths:**
- JWT signing uses RS256 exclusively with RSA key pairs.
- `ParseAccessToken` explicitly checks for `*jwt.SigningMethodRSA`, preventing
  algorithm confusion attacks.
- JWKS endpoint exposes only the RSA public key with `kid` identification.
- mTLS sender-constrained tokens (RFC 8705) implemented via
  `ValidateMTLSClientAuth` with `x5t#S256` thumbprint binding.
- OIDC discovery config exposes `id_token_signing_alg_values_supported`.

**Gaps:**
1. **No JWKS endpoint pinning** — Clients fetching from `/oauth/jwks` have no
   mechanism to pin expected key SPKI hashes.
2. **No certificate revocation checking** — mTLS thumbprint validation does not
   check CRL or OCSP for the client certificate.
3. **No mTLS at the transport layer** — The RFC 8705 implementation validates
   token claims but does not configure `tls.Config` with
  `RequireAndVerifyClientCert` for the actual TLS handshake.
4. **Static key provider** — The `KeyProvider` interface returns a single key
   pair. There is no key rotation or multi-key support.

### 7.3 WebAuthn (`services/auth/internal/webauthn/`)

**Strengths:**
- Packed attestation verifies ECDSA, RSA, and EdDSA signatures with proper
  algorithm dispatch.
- AAGUID lookup allows authenticator metadata enrichment.
- Supports "none" attestation format per WebAuthn spec.

**Gaps:**
1. **No attestation chain verification** — `VerifyPackedAttestation` parses the
   attestation certificate and checks the signature, but does not verify the
   certificate chain back to a trusted attestation root (FIDO MDS).
2. **Platform formats accepted without verification** — `fido-u2f`,
   `android-key`, `android-safetynet`, `tpm`, `apple` formats are accepted
   with `return nil` — no chain or signature verification.
3. **No FIDO Metadata Service integration** — Cannot validate attestation
   roots against the FIDO Alliance metadata blob.

### 7.4 Gateway HTTP/3 (`services/gateway/internal/http3/`)

**Strengths:**
- TLS configuration is required for HTTP/3 (QUIC).
- NextProtos set correctly for `h3`.

**Gaps:**
1. **No client certificate configuration** — Gateway TLS config does not set
   `ClientAuth` or `ClientCAs` for mTLS.
2. **No certificate pinning callbacks** — `VerifyPeerCertificate` is not
   configured.

---

## 8. Gap Analysis and Recommendations

### 8.1 Current State Summary

| Area | Status | Risk Level |
|------|--------|------------|
| SAML assertion signature verification | Implemented (single cert) | Medium — no rotation support |
| SAML cert chain validation | Not implemented | High — expired/revoked certs accepted |
| SAML IdP metadata parsing | Not implemented | Medium — manual cert import required |
| OAuth JWKS key rotation | Not implemented | Medium — single key, manual rotation |
| OAuth algorithm confusion prevention | Implemented | Low — properly enforced |
| mTLS token binding (RFC 8705) | Implemented (claims only) | Medium — no transport-layer mTLS |
| mTLS transport enforcement | Not implemented | High — no service-to-service auth |
| WebAuthn attestation chain | Not implemented | Medium — no root validation |
| Certificate revocation (CRL/OCSP) | Not implemented | High — revoked certs accepted |
| Certificate pinning (any type) | Not implemented | High — vulnerable to CA compromise |

### 8.2 Implementation Roadmap

#### Action 1: Add SAML Multi-Certificate Trust Store (Effort: 3-5 days)

Implement a `TrustStore` type in `pkg/saml/` that accepts multiple IdP signing
certificates with overlap windows. Update `VerifySignedAssertion` to iterate
over active certificates. Add `PurgeExpired()` for cleanup. This enables
zero-downtime IdP certificate rotation.

**Files:** `pkg/saml/trust_store.go` (new), `pkg/saml/signed_assertion.go`
(modify signature), `pkg/saml/trust_store_test.go` (new).

#### Action 2: Add Certificate Chain Validation for SAML (Effort: 2-3 days)

Before using an IdP signing certificate for assertion verification, call
`cert.Verify()` against a configurable root CA pool. Add `CertValidator` with
options for expiry checking, revocation (CRL fetch), and key usage validation.

**Files:** `pkg/saml/cert_validator.go` (new), `pkg/saml/signed_assertion.go`
(integrate).

#### Action 3: Implement Transport-Layer mTLS for Internal Services (Effort: 5-7 days)

Create a shared mTLS configuration package in `pkg/mtls/` that provides:
- Server-side `tls.Config` with `RequireAndVerifyClientCert` and internal CA.
- Client-side `http.Client` that presents service identity certificates.
- Certificate auto-rotation watcher for renewed certs.

Wire this into all 7 microservices' `cmd/main.go` entry points.

**Files:** `pkg/mtls/` (new package), all `services/*/cmd/main.go` (integrate).

#### Action 4: Add JWKS Key Pinning and Rotation Support (Effort: 3-4 days)

Extend the `KeyProvider` interface to support multiple active keys with
overlap windows. Add a `PinnedJWKSClient` for SDK consumers that validates
SPKI hashes against configured pins. Add key rotation API in the OAuth service.

**Files:** `services/oauth/internal/domain/models.go` (extend KeyProvider),
`services/oauth/internal/service/key_rotation.go` (new),
`sdk/go/jwks_pin.go` (new).

#### Action 5: Add WebAuthn Attestation Chain Verification (Effort: 4-5 days)

Implement FIDO Metadata Service (MDS) integration to validate attestation
certificate chains against the official FIDO root certificates. Add chain
verification for `packed` and `fido-u2f` formats. Remove the blanket `return
nil` for platform-specific formats and add at minimum signature verification.

**Files:** `services/auth/internal/webauthn/attestation.go` (extend),
`services/auth/internal/webauthn/fido_mds.go` (new).

### 8.3 Priority Ordering

1. **mTLS transport enforcement** (Action 3) — highest risk; service-to-service
   traffic is currently unauthenticated at the transport layer.
2. **Certificate revocation checking** (part of Actions 2-3) — revoked
   certificates are silently accepted today.
3. **SAML multi-cert trust store** (Action 1) — enables safe IdP cert rotation.
4. **JWKS rotation support** (Action 4) — needed for OAuth key lifecycle.
5. **WebAuthn attestation chains** (Action 5) — important for high-assurance
   deployments but lower risk for most use cases.

---

## References

- RFC 7469: Public Key Pinning Extension for HTTP (HPKP)
- RFC 8705: OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens
- RFC 9101: The OAuth 2.0 Authorization Framework: JWT-Secured Authorization Request (JAR)
- NIST SP 800-63-3: Digital Identity Guidelines
- OWASP: Certificate and Public Key Pinning
- FIDO Alliance: Metadata Service (MDS) Specification
- OASIS SAML 2.0 Metadata (sstc-saml-metadata-2.0-os)

---

*Document generated as part of GGID security research. Review the referenced
source files for implementation details and update this document when the
codebase changes.*
