# Internal Certificate Authority for IAM Service-to-Service TLS

> Research document for the GGID IAM platform. Covers the design, implementation,
> and operational lifecycle of an internal PKI for encrypting and authenticating
> all inter-service traffic (gRPC + REST) with mutual TLS.

---

## Table of Contents

1. [Why IAM Needs an Internal CA](#1-why-iam-needs-an-internal-ca)
2. [Root CA Bootstrap](#2-root-ca-bootstrap)
3. [Service Certificate Issuance](#3-service-certificate-issuance)
4. [Automated Certificate Rotation](#4-automated-certificate-rotation)
5. [OCSP and CRL](#5-ocsp-and-crl)
6. [Vault PKI Secrets Engine](#6-vault-pki-secrets-engine)
7. [SPIFFE/SPIRE Integration](#7-spiffespire-integration)
8. [Certificate Trust Distribution](#8-certificate-trust-distribution)
9. [GGID TLS Roadmap](#9-ggid-tls-roadmap)
10. [Gap Analysis & Recommendations](#10-gap-analysis--recommendations)

---

## 1. Why IAM Needs an Internal CA

### The Threat Model

An IAM platform like GGID processes credentials, JWTs, OAuth tokens, and user
PII at every layer. The seven microservices (gateway, identity, auth, oauth,
policy, org, audit) communicate over internal networks that are often treated
as trusted. This is a dangerous assumption:

- **Container escape** — if one container is compromised, plaintext traffic
  to other services is trivially sniffable.
- **Lateral movement** — an attacker inside the cluster can impersonate any
  service by sending HTTP requests with forged headers.
- **Insider threat** — anyone with network-level access (NOC engineers,
  cloud admins) can intercept credentials in transit.

Service-to-service mTLS solves both confidentiality (encryption) and
authentication (cryptographic identity) simultaneously.

### Public CA vs Internal CA vs Self-Signed

| Approach | Identity Binding | Trust Chain | Cost | Automation |
|---|---|---|---|---|
| **Public CA** (Let's Encrypt, DigiCert) | Public DNS names | Globally trusted | $0–$thousands | ACME (limited to port 80/443) |
| **Internal CA** (self-hosted PKI) | Internal service names, SPIFFE IDs | Private trust store | $0 software + ops cost | Full programmatic control |
| **Self-signed** (per-service) | None | No chain, every peer hardcoded | $0 | Manual, fragile, no revocation |

**Why public CAs don't work for internal services:**

1. **No public DNS** — Internal services resolve via service discovery
   (`identity:8081`, `auth:9001`), not FQDNs in public DNS zones.
2. **Domain validation fails** — Public CAs require proof of domain ownership.
   Internal service names are not publicly resolvable.
3. **Rate limits** — Let's Encrypt caps certificate issuance at 50 per
   registered domain per week. A 100-service cluster with hourly rotation
   exceeds this instantly.
4. **Cost** — Commercial CAs charge $200–$600/year per certificate. With
   7 GGID services, that's $1,400–$4,200/year — and that's just the current
   scale. Certificate Transparency logging also leaks internal architecture.
5. **Latency** — OCSP checks to public responders add 50–200ms per
   connection. Internal OCSP resolves in <1ms.

### SPIFFE/SPIRE as Alternative

SPIFFE (Secure Production Identity Framework for Everyone) eliminates the
need for a traditional CA by providing a workload identity framework:

- **SPIFFE ID** — A URI-format identity: `spiffe://ggid.dev/service/auth`
- **SVID** — A short-lived x509 or JWT credential proving the identity
- **SPIRE Agent** — Runs on each node, attests workloads, and issues SVIDs
  automatically without CSRs

SPIRE is ideal for dynamic environments (Kubernetes, Nomad) where services
scale up/down frequently. For GGID's current Docker Compose deployment with
fixed services, a traditional internal CA is simpler. SPIRE is the target
state for Kubernetes deployment (see Section 7).

---

## 2. Root CA Bootstrap

### Architecture: Offline Root + Online Intermediate

The standard two-tier PKI model separates the ultimate trust anchor (root
CA) from the operational signing key (intermediate CA):

```
                    Root CA (offline, HSM, 20-year)
                         |
                    Intermediate CA (online, 5-year)
                    /    |    |    \
               Identity Auth OAuth  Gateway ...
```

**Why two tiers?**

- The root CA key is the "crown jewels" — if compromised, the entire PKI
  is untrusted. By keeping it offline (air-gapped machine, HSM), the attack
  surface is minimized.
- The intermediate CA signs day-to-day CSRs. If its key is compromised,
  the root can revoke it and issue a new intermediate — no client trust
  store changes needed.
- Intermediate keys live on the signing server (or in Vault). Root keys
  never touch a networked machine.

### Root CA Key Generation

Root CA keys should be generated on an air-gapped machine, preferably with
an HSM (Hardware Security Module) or at minimum a YubiKey. For GGID's
open-source context, we show the software-based path with strong key
parameters.

```go
package ca

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

// GenerateRootCA creates a self-signed root CA certificate and private key.
// The key is ECDSA P-256 (preferred for performance and smaller signature size).
// The certificate is valid for 20 years.
//
// SECURITY: This function must be run on an offline machine. The returned
// key bytes should be stored in an HSM or encrypted at rest immediately.
func GenerateRootCA(commonName string) ([]byte, []byte, error) {
	// Generate ECDSA P-256 private key.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate root key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"GGID"},
		},
		NotBefore:             time.Now().Add(-time.Hour), // small clock skew tolerance
		NotAfter:              time.Now().AddDate(20, 0, 0), // 20 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1, // allow one level of intermediates
		MaxPathLenZero:        false,
	}

	// Self-sign: parent == template, pub == privKey.PublicKey
	certDER, err := x509.CreateCertificate(rand.Reader, template, template,
		&privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create root cert: %w", err)
	}

	// Encode to PEM.
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal root key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	return certPEM, keyPEM, nil
}
```

### Intermediate CA Creation

The intermediate CA is signed by the root CA but operates online. Its
certificate is shorter-lived (5 years) and marked as a CA with
`MaxPathLen: 0` (cannot sign further CAs).

```go
// GenerateIntermediateCA creates an intermediate CA certificate signed by
// the root CA. The intermediate is valid for 5 years.
func GenerateIntermediateCA(rootCert *x509.Certificate, rootKey *ecdsa.PrivateKey,
	commonName string) (*x509.Certificate, *ecdsa.PrivateKey, error) {

	interKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"GGID"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(5, 0, 0), // 5 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0, // leaf certs only, no further sub-CAs
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, rootCert,
		&interKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, err
	}

	interCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, err
	}

	return interCert, interKey, nil
}
```

### Offline Root Operational Procedure

1. Boot air-gapped machine from read-only media (Ubuntu live USB).
2. Run `GenerateRootCA("GGID Root CA")` — produces `root-ca.pem` and
   `root-ca-key.pem`.
3. Copy `root-ca.pem` (public cert) to a USB drive. This will be distributed
   to all service trust stores.
4. Store `root-ca-key.pem` in an HSM or encrypt with `age`/GPG and store
   in a physical safe. This key is only needed when creating or rotating
   the intermediate CA (every 5 years).
5. Power off the machine. Wipe all temp files.

---

## 3. Service Certificate Issuance

### CSR Flow

Each GGID service generates a keypair locally and submits a Certificate
Signing Request (CSR) to the CA. The CA validates the request, signs it,
and returns a short-lived leaf certificate.

```
Service                    Intermediate CA
  |                              |
  |-- generate keypair --------->|
  |-- submit CSR --------------->|
  |   (SAN: auth.ggid.svc)       |
  |                              |-- validate CSR
  |                              |-- sign with intermediate key
  |<------- leaf cert -----------|
  |   (valid 48h)                |
```

### Certificate Properties

| Property | Value | Rationale |
|---|---|---|
| Key type | ECDSA P-256 | Fast signing, small signatures |
| Validity | 24–48 hours | Limits blast radius of key compromise |
| SAN (DNS) | `auth`, `auth.ggid`, `auth.ggid.svc.cluster.local` | Service mesh DNS names |
| SAN (URI) | `spiffe://ggid.dev/service/auth` (optional) | SPIFFE-compatible identity |
| Key Usage | `digitalSignature`, `keyEncipherment` | TLS handshake + key exchange |
| Extended Key Usage | `serverAuth`, `clientAuth` | Bidirectional mTLS |
| Basic Constraints | `CA:FALSE` | Leaf certs cannot sign other certs |
| CRL Distribution Point | `http://ca.ggid.svc/crl/intermediate.crl` | Revocation checking |
| OCSP URL | `http://ocsp.ggid.svc` | Real-time status |

### Go Code: CSR Generation (Service Side)

```go
// GenerateCSR creates a CSR for a GGID service.
// The service keeps the private key; only the CSR is sent to the CA.
func GenerateCSR(serviceName string, dnsNames []string) (*x509.CertificateRequest, *ecdsa.PrivateKey, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   serviceName,
			Organization: []string{"GGID"},
		},
		DNSNames:       dnsNames,
		IPAddresses:    []net.IP{}, // optionally add pod IPs
		EmailAddresses: []string{},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, key)
	if err != nil {
		return nil, nil, err
	}

	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, nil, err
	}

	return csr, key, nil
}
```

### Go Code: CSR Signing (CA Side)

```go
// SignCSR signs a service CSR using the intermediate CA, producing a
// short-lived leaf certificate valid for 48 hours.
func SignCSR(csr *x509.CertificateRequest, interCert *x509.Certificate,
	interKey *ecdsa.PrivateKey, ttl time.Duration) (*x509.Certificate, error) {

	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("invalid CSR signature: %w", err)
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               csr.Subject,
		DNSNames:              csr.DNSNames,
		IPAddresses:           csr.IPAddresses,
		NotBefore:             time.Now().Add(-5 * time.Minute), // clock skew tolerance
		NotAfter:              time.Now().Add(ttl),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		CRLDistributionPoints: []string{"http://ca.ggid.svc/crl/intermediate.crl"},
		OCSPServer:            []string{"http://ocsp.ggid.svc"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, interCert,
		csr.PublicKey, interKey)
	if err != nil {
		return nil, fmt.Errorf("sign CSR: %w", err)
	}

	return x509.ParseCertificate(certDER)
}
```

### GGID Service DNS Names

For GGID's Docker Compose deployment, each service container is reachable
by its service name:

| Service | Primary DNS | Additional SAN |
|---|---|---|
| Gateway | `gateway` | `gateway.ggid.local` |
| Identity | `identity` | `identity.ggid.local` |
| Auth | `auth` | `auth.ggid.local` |
| OAuth | `oauth` | `oauth.ggid.local` |
| Policy | `policy` | `policy.ggid.local` |
| Org | `org` | `org.ggid.local` |
| Audit | `audit` | `audit.ggid.local` |

For Kubernetes, add the FQDN: `<service>.ggid.svc.cluster.local`.

---

## 4. Automated Certificate Rotation

### Why Short-Lived?

Short-lived certificates (24–48h) provide:

1. **Bounded key compromise window** — A stolen key is valid for at most
   48 hours, reducing the value of theft.
2. **Forced automation** — If rotation is manual, ops will forget. Short
   TTLs make automation mandatory.
3. **No CRL/OCSP needed for revocation** — If certs expire in hours, the
   CRL is always small and stale revocation data matters less.

### Zero-Downtime Rotation

The rotation process must not interrupt service:

```
Time →
T-0h:    Service running with cert A (valid until T+48h)
T+24h:   Agent fetches cert B (valid T+24h to T+72h)
T+24h+ε: Service loads both A and B into TLS config
T+24h+2ε: Service drains old connections, accepts new with B
T+25h:   Service drops A from active set (A still valid but unused)
T+48h:   Cert A expires — no impact, B is already serving
```

### Go Code: Hot-Reloading TLS Certificates

```go
package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"sync"
	"time"
)

// CertReloader manages hot-reloading of TLS certificates without dropping
// active connections. It implements the tls.Config.GetCertificate interface.
type CertReloader struct {
	mu          sync.RWMutex
	cert        *tls.Certificate
	certPath    string
	keyPath     string
	lastModTime time.Time
}

// NewCertReloader creates a reloader that reads certs from the given paths.
func NewCertReloader(certPath, keyPath string) (*CertReloader, error) {
	r := &CertReloader{certPath: certPath, keyPath: keyPath}
	if err := r.reload(); err != nil {
		return nil, err
	}
	return r, nil
}

// GetCertificate implements tls.Config.GetCertificate.
// Called on every TLS handshake.
func (r *CertReloader) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.RLock()
	cert := r.cert
	r.mu.RUnlock()
	return cert, nil
}

// GetClientCertificate implements tls.Config.GetClientCertificate (for mTLS).
func (r *CertReloader) GetClientCertificate(hello *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	r.mu.RLock()
	cert := r.cert
	r.mu.RUnlock()
	return cert, nil
}

// reload reads the cert+key from disk and atomically swaps the in-memory copy.
// Called by the background watcher.
func (r *CertReloader) reload() error {
	cert, err := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	if err != nil {
		return fmt.Errorf("load cert pair: %w", err)
	}

	r.mu.Lock()
	r.cert = &cert
	r.mu.Unlock()

	log.Printf("TLS certificate reloaded from %s", r.certPath)
	return nil
}

// StartWatcher polls for cert file changes and reloads automatically.
// Run as a goroutine.
func (r *CertReloader) StartWatcher(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		info, err := os.Stat(r.certPath)
		if err != nil {
			log.Printf("cert watcher stat error: %v", err)
			continue
		}
		if info.ModTime().After(r.lastModTime) {
			if err := r.reload(); err != nil {
				log.Printf("cert reload failed: %v", err)
			} else {
				r.lastModTime = info.ModTime()
			}
		}
	}
}

// BuildServerTLSConfig creates a tls.Config for an mTLS server.
func BuildServerTLSConfig(reloader *CertReloader, clientCAs *x509.CertPool) *tls.Config {
	return &tls.Config{
		GetCertificate:             reloader.GetCertificate,
		ClientCAs:                  clientCAs,
		ClientAuth:                 tls.RequireAndVerifyClientCert,
		MinVersion:                 tls.VersionTLS13,
		PreferServerCipherSuites:   true,
		SessionTicketsDisabled:     true, // disable for security: no resumption across cert rotation
	}
}

// BuildClientTLSConfig creates a tls.Config for an mTLS client (service-to-service).
func BuildClientTLSConfig(reloader *CertReloader, rootCAs *x509.CertPool) *tls.Config {
	return &tls.Config{
		GetClientCertificate: reloader.GetClientCertificate,
		RootCAs:              rootCAs,
		MinVersion:           tls.VersionTLS13,
	}
}
```

### Rotation Patterns by Platform

| Platform | Tool | Mechanism |
|---|---|---|
| Docker Compose | File watcher (shown above) | Watch cert dir, reload on change |
| Kubernetes | cert-manager + trust-manager | Kubernetes TLS secret mounted as volume, kubelet auto-reloads |
| Kubernetes | SPIRE | Workload API streams SVIDs |
| HashiCorp Nomad | Vault Agent template | Sidecar writes certs to `secrets/`, consul-template triggers reload |

---

## 5. OCSP and CRL

### OCSP (Online Certificate Status Protocol)

OCSP allows a client to query a CA's responder in real-time: "Is this
certificate still valid?" Defined in RFC 6960.

**Problem with plain OCSP:** The client sends the certificate serial number
to the CA's HTTP endpoint. This creates:

- **Privacy leak** — The CA knows which service is connecting to which.
- **Latency** — An extra HTTP round-trip per TLS handshake.
- **Availability** — If the OCSP responder is down, clients must decide:
  fail hard (break services) or fail soft (accept revoked certs).

### OCSP Stapling

**OCSP stapling** (RFC 6066) solves these problems: the server pre-fetches
the OCSP response and "staples" it to the TLS handshake. The client verifies
the stapled response without contacting the CA.

```
Without stapling:
  Client → Server: ClientHello
  Client → Server: TLS handshake
  Client → OCSP Responder: "Is cert #1234 valid?"   ← privacy leak + latency
  OCSP Responder → Client: "Yes, valid until T+1h"

With stapling:
  Server → OCSP Responder: "Give me status for #1234"  (pre-fetched)
  Server → Client: Certificate + OCSP Response (stapled)   ← no client→CA leak
```

### Go Code: OCSP Server

```go
package ocsp

import (
	"crypto"
	"crypto/x509"
	"net/http"
	"time"

	"golang.org/x/crypto/ocsp"
)

// OCSPResponder handles OCSP requests for the intermediate CA.
type OCSPResponder struct {
	caCert    *x509.Certificate
	caKey     crypto.Signer
	revoked   map[string]time.Time // serial → revocation time
}

// ServeHTTP handles OCSP requests per RFC 6960.
func (r *OCSPResponder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, 10*1024))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ocspReq, err := ocsp.ParseRequest(body)
	if err != nil {
		http.Error(w, "invalid OCSP request", http.StatusBadRequest)
		return
	}

	// Check revocation status.
	var status int
	var revokedAt time.Time
	if revTime, ok := r.revoked[ocspReq.SerialNumber.String()]; ok {
		status = ocsp.Revoked
		revokedAt = revTime
	} else {
		status = ocsp.Good
	}

	// Build OCSP response signed by the CA.
	tmpl := ocsp.Response{
		Status:           status,
		SerialNumber:     ocspReq.SerialNumber,
		ThisUpdate:       time.Now().Add(-time.Minute),
		NextUpdate:       time.Now().Add(1 * time.Hour),
		Certificate:      r.caCert,
		IssuerHash:       crypto.SHA256,
		RevokedAt:        revokedAt,
		RevocationReason: ocsp.KeyCompromise,
	}

	respBytes, err := tmpl.Sign(r.caCert, r.caKey)
	if err != nil {
		http.Error(w, "sign OCSP response failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/ocsp-response")
	w.Write(respBytes)
}
```

### Go Code: OCSP Client (Stapling Verification)

```go
// VerifyStapledOCSP verifies a stapled OCSP response from a TLS connection.
func VerifyStapledOCSP(stapledOCSP []byte, cert, issuer *x509.Certificate) error {
	if len(stapledOCSP) == 0 {
		return fmt.Errorf("no OCSP response stapled")
	}

	resp, err := ocsp.ParseResponse(stapledOCSP, issuer)
	if err != nil {
		return fmt.Errorf("parse OCSP: %w", err)
	}

	if resp.Status == ocsp.Revoked {
		return fmt.Errorf("certificate %s revoked at %v (reason: %d)",
			cert.SerialNumber, resp.RevokedAt, resp.RevocationReason)
	}

	if time.Now().After(resp.NextUpdate) {
		return fmt.Errorf("OCSP response expired (next update was %v)", resp.NextUpdate)
	}

	return nil
}
```

### CRL (Certificate Revocation List)

CRLs are signed lists of revoked serial numbers, distributed via HTTP. While
older than OCSP, CRLs are simpler and work without network connectivity
during TLS handshake (the list is pre-fetched).

```go
// GenerateCRL creates a DER-encoded CRL signed by the intermediate CA.
func GenerateCRL(caCert *x509.Certificate, caKey crypto.Signer,
	revoked []pkix.RevokedCertificate) ([]byte, error) {

	template := &x509.RevocationList{
		Number:             big.NewInt(time.Now().Unix()), // CRL number
		ThisUpdate:         time.Now().Add(-time.Minute),
		NextUpdate:         time.Now().Add(24 * time.Hour),
		RevokedCertificates: revoked,
	}

	return x509.CreateRevocationList(rand.Reader, template, caCert, caKey)
}
```

**Recommendation for GGID:** Use OCSP stapling as the primary mechanism
(with 1-hour OCSP response lifetime) and publish a daily CRL as a fallback.
The short-lived (48h) service certs make revocation rare — by the time a
revocation propagates, the cert has likely expired.

---

## 6. Vault PKI Secrets Engine

HashiCorp Vault's PKI secrets engine provides a managed internal CA with
role-based certificate issuance, automatic revocation tracking, and audit
logging — without managing raw keys.

### Architecture

```
Vault Server
  ├── PKI Engine (intermediate CA key stored in Vault)
  │   ├── pki/root       → root CA cert (imported, root key in HSM/safe)
  │   ├── pki/issue/auth → role: allowed_domains="auth.ggid", ttl=48h
  │   ├── pki/issue/identity → role: allowed_domains="identity.ggid", ttl=48h
  │   └── pki/issue/gateway → role: allowed_domains="gateway.ggid", ttl=48h
  │
Vault Agent (sidecar per service)
  ├── Requests cert from pki/issue/<service>
  ├── Writes cert+key to /opt/ggid/certs/
  └── Triggers service reload on file change
```

### Vault Setup Commands

```bash
# Enable PKI engine
vault secrets enable -path=pki pki
vault secrets tune -max-lease-ttl=87600h pki  # 10 years max

# Generate intermediate CA (root is imported from offline)
vault write pki/intermediate/generate/internal \
    common_name="GGID Intermediate CA" \
    ttl=43800h  # 5 years

# Sign intermediate with root CA (offline)
vault write -format=json pki/intermediate/generate/internal \
    common_name="GGID Intermediate CA" \
    ttl=43800h | jq -r .data.csr > inter.csr

# (on offline root machine) sign the CSR
# Then import the signed cert:
vault write pki/intermediate/set-signed certificate=@inter-signed.pem

# Create per-service roles
vault write pki/roles/auth \
    allowed_domains="auth.ggid,auth,auth.ggid.svc.cluster.local" \
    allow_subdomains=false \
    max_ttl=48h \
    key_type=ec \
    key_bits=256

vault write pki/roles/identity \
    allowed_domains="identity.ggid,identity,identity.ggid.svc.cluster.local" \
    allow_subdomains=false \
    max_ttl=48h \
    key_type=ec \
    key_bits=256
```

### Go Code: Requesting Certs from Vault PKI

```go
package vaultpki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// VaultClient requests certificates from Vault PKI secrets engine.
type VaultClient struct {
	addr   string
	token  string
	client *http.Client
}

// NewVaultClient creates a Vault PKI client.
func NewVaultClient(addr, token string) *VaultClient {
	return &VaultClient{
		addr:  addr,
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IssueRequest is the body sent to Vault's pki/issue/<role> endpoint.
type IssueRequest struct {
	CommonName string `json:"common_name"`
	TTL        string `json:"ttl"`
	AltNames   string `json:"alt_names,omitempty"`
}

// IssueResponse contains the cert, key, and CA chain.
type IssueResponse struct {
	Data struct {
		Certificate    string `json:"certificate"`
		PrivateKey     string `json:"private_key"`
		PrivateKeyType string `json:"private_key_type"`
		IssuingCA      string `json:"issuing_ca"`
		CAChain        []string `json:"ca_chain"`
		SerialNumber   string `json:"serial_number"`
	} `json:"data"`
}

// IssueCertificate requests a new certificate from Vault.
func (v *VaultClient) IssueCertificate(ctx context.Context, role string,
	req IssueRequest) (*IssueResponse, error) {

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/v1/pki/issue/%s", v.addr, role)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("X-Vault-Token", v.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("vault request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vault returned %d: %s", resp.StatusCode, respBody)
	}

	var result IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode vault response: %w", err)
	}

	return &result, nil
}
```

### Vault Agent Auto-Renewal

Deploy Vault Agent as a sidecar container. It uses templates to fetch
certs and write them to a shared volume:

```hcl
# vault-agent-config.hcl
template {
  source      = "/etc/vault/templates/cert.tpl"
  destination = "/certs/auth.pem"
  command     = "/usr/local/bin/reload-ggid-service.sh auth"
}
```

```gotemplate
{{ with secret "pki/issue/auth" "common_name=auth.ggid" "ttl=48h" "alt_names=auth,auth.ggid.svc.cluster.local" }}
{{ .Data.certificate }}
{{ .Data.private_key }}
{{ end }}
```

---

## 7. SPIFFE/SPIRE Integration

### SPIFFE Identity Model

SPIFFE provides a framework for workload identity that replaces traditional
x509-based PKI with a standardized identity format:

```
SPIFFE ID: spiffe://ggid.dev/service/auth
           ^^^^^^^^^^^^  ^^^^^^^^^^^^^^^^
           trust domain   workload path
```

Each service gets a **SPIFFE Verifiable Identity Document (SVID)** — a
short-lived credential proving its identity. Two SVID formats exist:

| Format | Use Case | Lifetime | Verification |
|---|---|---|---|
| **x509-SVID** | Service-to-service mTLS | 1 hour | Standard x509 path validation |
| **JWT-SVID** | External API calls, federation | 5–15 min | JWT signature validation |

### SPIRE Architecture

```
┌─────────────────────────────────────────┐
│              SPIRE Server               │
│  (signs SVIDs, maintains trust bundle)  │
│  trust domain: ggid.dev                 │
└──────────────┬──────────────────────────┘
               │ workload API (Unix socket)
    ┌──────────┼──────────────┐
    │          │              │
┌───▼───┐ ┌───▼───┐  ┌──────▼──────┐
│ SPIRE │ │ SPIRE │  │   SPIRE     │
│ Agent │ │ Agent │  │   Agent     │
│ node1 │ │ node2 │  │   node3     │
└───┬───┘ └───┬───┘  └──────┬──────┘
    │          │              │
 ┌──▼──┐  ┌───▼──┐     ┌─────▼─────┐
 │auth │  │gateway│    │  identity │
 │svc  │  │ svc   │    │   svc     │
 └─────┘  └──────┘     └───────────┘
```

### SPIRE Attestation

SPIRE uses attestation to verify workload identity:

1. **Node attestation** — The SPIRE Agent proves it's running on an
   authorized node (Kubernetes Service Account Token, AWS IID, etc.).
2. **Workload attestation** — The Agent identifies workloads by their
   Unix UID, Kubernetes Service Account, or Docker container ID.
3. **SVID issuance** — The Agent mints an x509-SVID with the appropriate
   SPIFFE ID and delivers it via the Workload API (Unix domain socket).

### Go Code: SPIFFE-Based mTLS

```go
package spiffe

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

// NewSPIFFEServerTLSConfig creates a tls.Config that:
//   - Gets its SVID from the SPIRE Workload API
//   - Verifies client SVIDs from the same trust domain
//   - Enforces mTLS
func NewSPIFFEServerTLSConfig(ctx context.Context, socketPath string,
	trustDomain string) (*tls.Config, error) {

	// Connect to SPIRE Workload API.
	source, err := workloadapi.NewX509Source(ctx,
		workloadapi.WithClientOptions(
			workloadapi.WithAddr(socketPath),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to SPIRE: %w", err)
	}

	td, err := spiffeid.TrustDomainFromString(trustDomain)
	if err != nil {
		return nil, err
	}

	// Authorize: only services with this trust domain can connect.
	authorizer := tlsconfig.AdaptMatcher(func(id spiffeid.ID) error {
		if id.TrustDomain() != td {
			return fmt.Errorf("untrusted domain: %s", id.TrustDomain())
		}
		return nil
	})

	return tlsconfig.MTLSServerConfig(source, source, authorizer), nil
}

// DialWithSPIFFE creates an mTLS gRPC client connection using SPIFFE SVIDs.
func DialWithSPIFFE(ctx context.Context, target, socketPath string,
	trustDomain string) (*grpc.ClientConn, error) {

	source, err := workloadapi.NewX509Source(ctx,
		workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath)))
	if err != nil {
		return nil, err
	}

	td, _ := spiffeid.TrustDomainFromString(trustDomain)

	creds := credentials.NewTLS(tlsconfig.MTLSClientConfig(
		source,
		source,
		tlsconfig.AuthorizeMemberOf(td),
	))

	return grpc.DialContext(ctx, target, grpc.WithTransportCredentials(creds))
}
```

### How SPIRE Replaces Traditional CA

| Concern | Traditional CA | SPIRE |
|---|---|---|
| Identity issuance | Service submits CSR manually | Agent auto-issues SVID via attestation |
| Key storage | Service stores private key on disk | Private key never leaves SPIRE Agent memory |
| Rotation | File watcher or Vault Agent polls | Workload API streams updates (push, not poll) |
| Trust distribution | Manually deploy root CA cert | Trust bundle fetched from SPIRE Server |
| Authorization | Based on DNS name in SAN | Based on SPIFFE ID (cryptographic workload identity) |
| Revocation | OCSP/CRL (slow, unreliable) | Short-lived SVIDs (1h) — revocation is implicit |

---

## 8. Certificate Trust Distribution

### Trust Bundle

The **trust bundle** is the set of root CA (and optionally intermediate CA)
certificates that a service trusts. Every service must have the trust bundle
installed to verify peer certificates.

### Distribution Mechanisms

| Method | Refresh | Platform | Notes |
|---|---|---|---|
| Docker volume mount | Manual (restart) | Docker Compose | Current GGID approach |
| Kubernetes ConfigMap | Automatic (kubelet) | K8s | ConfigMap updates trigger pod restart |
| trust-manager | Automatic | K8s | Operator that syncs bundle to all namespaces |
| SPIRE Bundle API | Streaming | All | SPIRE Server pushes bundle updates |
| HTTP bundle endpoint | Polling | All | `GET /.well-known/spiffe/bundle.json` |

### Go Code: Trust Bundle Manager

```go
package trustbundle

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// BundleManager maintains an in-memory trust store of root/intermediate
// CA certificates. It supports hot-reloading and HTTP bundle fetching.
type BundleManager struct {
	mu       sync.RWMutex
	pool     *x509.CertPool
	source   string // URL or file path
}

// NewBundleManager creates a manager that loads from the given PEM source.
func NewBundleManager(source string) *BundleManager {
	return &BundleManager{
		pool:   x509.NewCertPool(),
		source: source,
	}
}

// LoadFromPEM adds PEM-encoded certificates to the trust pool.
func (m *BundleManager) LoadFromPEM(pemData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemData) {
		return fmt.Errorf("no valid certificates found in PEM data")
	}

	m.pool = pool
	return nil
}

// LoadFromHTTP fetches a trust bundle from an HTTP endpoint (SPIFFE bundle API).
func (m *BundleManager) LoadFromHTTP(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, m.source, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch trust bundle: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return err
	}

	// Try SPIFFE bundle format (JSON with x509 certificates).
	var spiffeBundle struct {
		TrustDomain string            `json:"trust_domain"`
		Keys        []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(body, &spiffeBundle); err == nil && spiffeBundle.TrustDomain != "" {
		return m.parseSPIFFEBundle(body)
	}

	// Fall back to PEM.
	return m.LoadFromPEM(body)
}

// GetPool returns the current x509.CertPool (thread-safe).
func (m *BundleManager) GetPool() *x509.CertPool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pool
}

// StartAutoRefresh polls the source endpoint at the given interval.
func (m *BundleManager) StartAutoRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.LoadFromHTTP(ctx); err != nil {
				// Log but keep old bundle
				continue
			}
		}
	}
}

// parseSPIFFEBundle parses a SPIFFE trust bundle JSON document.
func (m *BundleManager) parseSPIFFEBundle(data []byte) error {
	var bundle struct {
		TrustDomain string   `json:"trust_domain"`
		Sequence    uint64   `json:"sequence_number"`
		Keys        []struct {
			Use string   `json:"use"`
			X509 []string `json:"x509_certificate"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(data, &bundle); err != nil {
		return err
	}

	pool := x509.NewCertPool()
	for _, key := range bundle.Keys {
		for _, certB64 := range key.X509 {
			certDER, err := base64.StdEncoding.DecodeString(certB64)
			if err != nil {
				continue
			}
			cert, err := x509.ParseCertificate(certDER)
			if err != nil {
				continue
			}
			pool.AddCert(cert)
		}
	}

	m.mu.Lock()
	m.pool = pool
	m.mu.Unlock()
	return nil
}
```

### Root CA Rotation

Root CA rotation is the most delicate PKI operation. The standard approach:

1. **Publish new root** — Generate new root CA, distribute to all trust
   bundles alongside the old root.
2. **Accept both roots** — All services now trust both old and new roots.
   This "overlap period" lasts 6–12 months.
3. **Sign new intermediates** — Issue new intermediate CAs signed by the
   new root. New leaf certs are signed by the new intermediate.
4. **Revoke old intermediate** — Stop signing with the old intermediate.
   Existing leaf certs expire naturally (24–48h).
5. **Remove old root** — After all leaf certs have rotated through the
   new chain, remove the old root from trust bundles.

```
Month 0:   Trust = [Root-A]
Month 1:   Trust = [Root-A, Root-B]
           New intermediates signed by Root-B
           New leaf certs signed by Intermediate-B
Month 3:   Trust = [Root-A, Root-B]
           All leaf certs now from Intermediate-B
Month 12:  Trust = [Root-B]
           Root-A removed
```

---

## 9. GGID TLS Roadmap

### Current State Assessment

Based on a review of the GGID codebase:

#### What Exists

| Component | TLS Status | Evidence |
|---|---|---|
| **Nginx (edge)** | TLS termination at edge | `deploy/nginx/nginx.conf` — `ssl_certificate /certs/fullchain.pem`, TLS 1.2+1.3, HSTS |
| **Gateway HTTP/3** | TLS required but not provisioned | `services/gateway/internal/http3/server.go` — `cfg.TLSConfig` required, `NextProtos = ["h3"]` |
| **Gateway transport pool** | TLS timeout configured, no cert | `services/gateway/internal/transport/pool.go` — `TLSHandshakeTimeout: 5s`, no `tls.Config` |
| **LDAP provider** | StartTLS supported | `pkg/authprovider/ldap.go` — `StartTLS bool`, `TLSConfig *tls.Config` |
| **SMTP email** | TLS supported | `pkg/email/sender.go` — `crypto/tls` import, TLS config |
| **OAuth mTLS (RFC 8705)** | Client cert thumbprint binding | `services/oauth/internal/service/jar_mtls.go` — `ExtractCertThumbprint`, `ValidateMTLSClientAuth` |
| **SAML** | x509 cert generation for testing | `pkg/saml/` — test cert helpers only |

#### What Does NOT Exist (Gaps)

| Gap | Impact | Current Code |
|---|---|---|
| **gRPC plaintext** | All gRPC traffic (identity, policy, org, audit) is unencrypted | `grpc.NewServer()` with no `credentials.NewTLS()`, `grpc.Dial` with `Insecure` |
| **HTTP plaintext between services** | Gateway → backend HTTP calls have no TLS | `httputil.ReverseProxy` targets `http://service:port` |
| **No internal CA** | No certificate generation, signing, or management | No `x509.CreateCertificate` for service certs |
| **No cert rotation** | If certs were deployed, they'd be static | No file watcher, no Vault Agent, no cert-manager |
| **No trust bundle distribution** | Services have no mechanism to receive CA certs | No trust store in any service config |
| **Docker Compose: no certs** | No cert volumes or TLS env vars in compose file | `deploy/docker-compose.yaml` — no TLS configuration |
| **No service identity** | Services cannot cryptographically prove identity | No SPIFFE/SPIRE integration |

### Target State

```
                    Internet
                       │
                 ┌─────▼─────┐
                 │   Nginx   │  (existing: TLS termination)
                 │  :443     │
                 └─────┬─────┘
                       │ HTTPS (existing)
                 ┌─────▼─────┐
                 │  Gateway  │  ── mTLS (NEW) ──┐
                 │  :8080    │                 │
                 └───────────┘                 │
                    │     │     │              │
              mTLS  │  mTLS│  mTLS│             │
            (NEW)   │ (NEW)│ (NEW)│             │
         ┌──────────┤      │      ├─── ... ─────┘
         │ Identity │   Auth│   OAuth│
         │ :8081    │  :9001│   :9005│
         │ :50051   │       │       │
         └──────────┴───────┴───────┘
                    gRPC + REST all over mTLS
```

---

## 10. Gap Analysis & Recommendations

### Summary of Findings

GGID currently has **edge TLS only** (Nginx termination). All internal
service-to-service traffic — both HTTP and gRPC — flows in **plaintext**.
The OAuth service has RFC 8705 mTLS sender-constrained token support, but
this is for OAuth client binding, not for encrypting inter-service traffic.

### Action Items

| # | Action | Priority | Effort | Impact |
|---|--------|----------|--------|--------|
| 1 | **Add TLS config to gRPC servers** | P0 | 2 days | Encrypts all gRPC traffic (identity, policy, org, audit) |
| 2 | **Add mTLS to Gateway→backend HTTP** | P0 | 3 days | Encrypts all REST proxy traffic |
| 3 | **Implement internal CA bootstrap** | P1 | 3 days | Self-contained root+intermediate CA generation tool |
| 4 | **Add cert rotation (file watcher)** | P1 | 2 days | Automatic cert reload without downtime |
| 5 | **Evaluate SPIRE for K8s deployment** | P2 | 1 week research | Stateless workload identity, eliminates cert file management |

### Detailed Action Items

#### Action 1: Add TLS to gRPC Servers (P0, 2 days)

Modify each service's `internal/server/server.go` to create gRPC servers
with TLS credentials:

```go
// Current (plaintext):
grpcSrv := grpc.NewServer()

// Target (mTLS):
creds, err := credentials.NewServerTLSFromFile(certPath, keyPath)
grpcSrv := grpc.NewServer(grpc.Creds(creds))
```

For client-side (gateway → backend):
```go
creds := credentials.NewClientTLSFromFile(caPath, "")
conn, _ := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
```

Files to modify: `identity/internal/server/server.go`, `policy/cmd/main.go`,
`org/cmd/main.go`, `audit/cmd/main.go`, `gateway/internal/middleware/grpc_interceptor.go`.

#### Action 2: Add mTLS to Gateway HTTP Reverse Proxy (P0, 3 days)

Modify `services/gateway/internal/transport/pool.go` to add a `tls.Config`
to the `http.Transport`:

```go
// Add TLSConfig field to PoolConfig struct
type PoolConfig struct {
    // ... existing fields ...
    TLSConfig *tls.Config
}

// In NewTransport:
transport := &http.Transport{
    // ... existing fields ...
    TLSClientConfig: cfg.TLSConfig,
}
```

#### Action 3: Implement Internal CA Bootstrap (P1, 3 days)

Create `pkg/ca/` package with:
- `GenerateRootCA()` — offline root generation
- `GenerateIntermediateCA()` — intermediate signing
- `SignCSR()` — service certificate issuance
- CLI tool: `cmd/ggid-ca/main.go`

#### Action 4: Add Cert Rotation (P1, 2 days)

Create `pkg/tlsutil/cert_reloader.go` with the `CertReloader` type from
Section 4. Integrate into each service's server startup.

#### Action 5: SPIRE Evaluation for Kubernetes (P2, 1 week)

For the Helm chart (`deploy/helm/ggid/`), evaluate:
- SPIRE Server deployment (Helm chart: `spiffe/spire`)
- SPIRE Agent DaemonSet
- Kubernetes workload registrar
- Replace cert files with SPIFFE Workload API integration

This eliminates all cert file management — SVIDs are issued and rotated
automatically by the SPIRE Agent.

### Recommended Phasing

```
Phase 1 (Week 1-2): gRPC + HTTP mTLS with static certs
  → Proves the TLS plumbing works end-to-end

Phase 2 (Week 3-4): Internal CA + cert rotation
  → Automates cert lifecycle, no manual cert management

Phase 3 (Week 5-8): SPIRE for Kubernetes
  → Stateless identity, zero-touch cert management at scale
```

---

## Appendix: Certificate Hierarchy for GGID

```
GGID Root CA (offline, 20yr)
  └── GGID Intermediate CA (online, 5yr)
        ├── gateway.ggid (48h, SAN: gateway, gateway.ggid.svc)
        ├── identity.ggid (48h, SAN: identity, identity.ggid.svc)
        ├── auth.ggid (48h, SAN: auth, auth.ggid.svc)
        ├── oauth.ggid (48h, SAN: oauth, oauth.ggid.svc)
        ├── policy.ggid (48h, SAN: policy, policy.ggid.svc)
        ├── org.ggid (48h, SAN: org, org.ggid.svc)
        └── audit.ggid (48h, SAN: audit, audit.ggid.svc)
```

Each leaf certificate has:
- `KeyUsage: digitalSignature | keyEncipherment`
- `ExtKeyUsage: serverAuth, clientAuth` (for bidirectional mTLS)
- `SAN: <service>, <service>.ggid, <service>.ggid.svc.cluster.local`
- `OCSP: http://ocsp.ggid.svc`
- `CRL: http://ca.ggid.svc/crl/intermediate.crl`
- ECDSA P-256 key, 48-hour validity

---

*Document version: 1.0 | GGID IAM Platform | Apache 2.0 License*
