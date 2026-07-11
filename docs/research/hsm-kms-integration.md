# HSM and Cloud KMS Integration for IAM Systems

> Research document for the GGID project — deep technical integration of Hardware
> Security Modules (HSMs), PKCS#11, and cloud key management services (AWS KMS,
> Google Cloud KMS, Azure Key Vault) with Go-based identity platforms.
>
> **Companion documents:** `secret-management-iam.md` covers Vault secret storage
> and a high-level KMS overview. `key-rotation-iam.md` covers the key lifecycle
> state machine and mentions HSM/PKCS#11 at a conceptual level. This document
> focuses on **hands-on integration engineering**: PKCS#11 programming, cloud KMS
> SDK code, performance benchmarks, cost models, and a concrete CryptoProvider
> design for GGID.

**Status:** Draft
**Audience:** GGID architects, platform engineers, security reviewers, compliance officers
**Last Updated:** 2025

---

## Table of Contents

1. [HSM Fundamentals](#1-hsm-fundamentals)
2. [PKCS#11 Interface](#2-pkcs11-interface)
3. [SoftHSM2 for Development](#3-softhsm2-for-development)
4. [AWS KMS Integration](#4-aws-kms-integration)
5. [Google Cloud KMS Integration](#5-google-cloud-kms-integration)
6. [Azure Key Vault Integration](#6-azure-key-vault-integration)
7. [HashiCorp Vault Transit Engine](#7-hashicorp-vault-transit-engine)
8. [Performance Comparison](#8-performance-comparison)
9. [Key Lifecycle in HSM/KMS](#9-key-lifecycle-in-hsmkms)
10. [GGID Integration Design](#10-ggid-integration-design)
11. [High Availability](#11-high-availability)
12. [Compliance Mapping](#12-compliance-mapping)
13. [Gap Analysis & Recommendations](#13-gap-analysis--recommendations)

---

## 1. HSM Fundamentals

### 1.1 What Is an HSM?

A Hardware Security Module (HSM) is a physical computing device that safeguards
and manages digital keys for strong authentication and provides cryptoprocessing.
HSMs are designed to be **tamper-resistant** — they actively destroy key material
if physical intrusion is detected. Unlike software key stores, the private key
material inside an HSM **never leaves** the device in plaintext form. All
cryptographic operations (signing, decryption, key derivation) are performed
inside the HSM's secure boundary.

Key properties of an HSM:

| Property | Description |
|---|---|
| **Tamper resistance** | Physical sensors detect drilling, temperature extremes, voltage manipulation. Keys are zeroized on tamper event. |
| **Key isolation** | Private keys generated inside the HSM cannot be extracted, even by the host operating system or a root user. |
| **Cryptographic acceleration** | Dedicated crypto processors (ASIC/FPGA) accelerate RSA/ECDSA signing and AES encryption. |
| **Access control** | PINs, smart cards, or M-of-N quorum approval required for sensitive operations (key creation, export, deletion). |
| **Audit logging** | All key operations are logged internally. Logs cannot be erased by the host. |
| **FIPS certification** | Validated against FIPS 140-2 or FIPS 140-3 at levels 1 through 4. |

### 1.2 FIPS 140-2 and FIPS 140-3 Validation Levels

The Federal Information Processing Standard (FIPS) Publication 140 defines the
U.S. government standard for cryptographic modules. FIPS 140-2 was the standard
from 2001 to 2026; FIPS 140-3 (based on ISO/IEC 19790) supersedes it.

| Level | Physical Security | Logical Security | Typical Use Case | IAM Relevance |
|---|---|---|---|---|
| **Level 1** | No physical protection | Software-only encryption | Desktop apps, test environments | Not suitable for production IAM |
| **Level 2** | Tamper-evident seals or coatings | Role-based auth, software key store | Cloud KMS (shared), application signing | Minimum for compliance-sensitive IAM |
| **Level 3** | Tamper-responsive (zeroizes keys) | Identity-based auth, private key never leaves | Network HSM, cloud dedicated HSM | Recommended for enterprise IAM |
| **Level 4** | Environmental tamper (temp, voltage) | Strong physical protection, audit trail | Government, military, root CA | Required for government / QSCD |

For IAM systems:

- **Level 2** is the minimum for PCI-DSS compliance when card data is involved.
- **Level 3** is the recommended baseline for enterprise IAM (e.g., signing JWTs
  for financial services, healthcare).
- **Level 4** is mandatory for eIDAS Qualified Electronic Signatures (QES) and
  U.S. federal PKI root CAs.

### 1.3 Why IAM Needs HSM

An IAM system is the **root of trust** for every application it authenticates. If
the IAM's private keys are compromised, an attacker can forge tokens for any user,
impersonate any service, and decrypt any token-protected data. The keys that need
HSM protection include:

1. **JWT signing keys** (RSA/ECDSA private keys) — The most critical. If
   compromised, an attacker can forge access tokens for any user. GGID currently
   stores these in PEM files on disk (`configs/rsa_private.pem`), which is
   acceptable for development but insufficient for production.

2. **SAML signing keys** — Used to sign SAML assertions for federated SSO. SAML
   assertions are trust-bearing XML documents. A compromised SAML key allows
   identity spoofing across all federated applications.

3. **OIDC ID token signing keys** — Same as JWT keys; the `id_token` is a JWT.

4. **Data encryption keys (DEK)** — AES-256 keys used for PII encryption at rest.
   GGID's `pkg/crypto.AESEncrypt` currently takes a key as a `[]byte` parameter,
   meaning the key exists in application memory and is passed from configuration.

5. **OAuth client secrets** — While not cryptographic keys per se, stored client
   secrets should be encrypted at rest using an HSM-protected master key.

6. **WebAuthn attestation CA keys** — If GGID operates as a WebAuthn authenticator
   manufacturer or attestation CA.

### 1.4 HSM Types

#### Network HSM (Network-Attached)

Connected via TCP/IP. The application communicates with the HSM over the network
using protocols like PKCS#11, KMIP, or vendor-specific APIs.

| Vendor | Product | Interface | Price Range |
|---|---|---|---|
| Thales (SafeNet) | Luna Network HSM 7 | PKCS#11, KMIP | $15,000–$50,000+ |
| Entrust | nShield Connect XC | PKCS#11, nShield API | $20,000–$60,000+ |
| Utimaco | SecurityServer | PKCS#11, JCE | $15,000–$45,000+ |
| AWS | CloudHSM | PKCS#11 | $1.50/hour (~$13,140/year) |
| Google Cloud | Cloud HSM | PKCS#11, Cloud KMS API | $3,000/month for dedicated |

Network HSMs are the most common choice for production IAM. They support
clustering for HA and are accessible from multiple application servers.

#### PCIe HSM

Installed directly in a server's PCIe slot. Lower latency than network HSMs
(microsecond vs millisecond) but no network access.

| Vendor | Product | Interface | Price Range |
|---|---|---|---|
| Thales | Luna PCIe HSM 7 | PKCS#11 | $10,000–$35,000 |
| Utimaco | SecurityServer Se | PKCS#11 | $10,000–$30,000 |
| Marvell | LiquidSec PCIe | PKCS#11 | $5,000–$15,000 |

PCIe HSMs are ideal for latency-sensitive applications where the signer and HSM
run on the same host. For distributed IAM microservices, network HSMs are more
practical.

#### Cloud HSM

Managed HSM as a service. No hardware procurement or maintenance.

| Provider | Service | Certification | Price |
|---|---|---|---|
| AWS CloudHSM | FIPS 140-2 Level 3 | PKCS#11 | $1.499/hour per HSM |
| Azure Dedicated HSM | FIPS 140-2 Level 3 | PKCS#11 | ~$2,000/month |
| Google Cloud HSM | FIPS 140-2 Level 3 | Cloud KMS API | $8,000/month (annual) |
| IBM Cloud HSM | FIPS 140-2 Level 3 | PKCS#11 | $1,000/month |

Cloud HSMs combine the security of dedicated hardware with the operational
simplicity of managed services. They are the default choice for cloud-native IAM.

#### USB HSM (YubiKey, Nitrokey HSM)

Low-cost hardware tokens for development, testing, or low-volume signing.

| Product | Certification | Price |
|---|---|---|
| YubiKey 5 series | FIPS 140-2 Level 2 | $50–$80 |
| Nitrokey HSM 2 | FIPS 140-2 Level 3 (Common Criteria EAL 5+) | ~$70 |
| Ledger Nano S | (Not FIPS certified) | ~$60 |

USB HSMs are useful for development and for key ceremonies where a human operator
must physically touch the device to authorize operations.

### 1.5 Decision Matrix

```
Do you need FIPS 140-2/3 Level 3+?
├── Yes → Network HSM or Cloud HSM
│   ├── Cloud-native? → AWS CloudHSM / GCP HSM / Azure Dedicated HSM
│   └── On-premise? → Thales Luna / Entrust nShield / Utimaco
└── No (dev/testing)
    ├── Need real PKCS#11 API? → SoftHSM2 (software HSM)
    └── Just need secure random? → /dev/urandom + crypto/rand
```

---

## 2. PKCS#11 Interface

### 2.1 Overview

PKCS#11 (also known as Cryptoki) is the industry-standard API for interacting
with cryptographic tokens, including HSMs, smart cards, and software tokens. It
is defined by RSA Laboratories (now part of OASIS) in the PKCS #11 specification.
Almost every HSM vendor provides a PKCS#11 library (`libpkcs11.so`), making it
the most portable interface for HSM programming.

The current standard is PKCS #11 v3.0 (2020), which adds new mechanisms and
improved type safety. Most HSM vendors implement v2.40, which is sufficient for
IAM use cases.

### 2.2 Core Concepts

| Concept | Description |
|---|---|
| **Slot** | A physical or logical reader that can hold a token. Maps to an HSM partition or a smart card reader. |
| **Token** | The physical device or logical partition that stores keys and certificates. In SoftHSM2, each token is a directory of key files. |
| **Session** | A logical connection between the application and the token. Sessions are opened on a slot and can be read-only (RO) or read-write (RW). |
| **Object** | A key, certificate, or data object stored on the token. Identified by object handles. |
| **Mechanism** | A cryptographic algorithm (e.g., `CKM_RSA_PKCS`, `CKM_ECDSA`, `CKM_AES_GCM`). |
| **Attribute** | Properties of an object (e.g., `CKA_PRIVATE`, `CKA_SIGN`, `CKA_EXTRACTABLE`). |
| **User** | A role (Normal User or Security Officer) authenticated via PIN. |

### 2.3 Key Object Attributes

When generating a key inside an HSM, you specify attributes that control how the
key can be used:

```c
// C (PKCS#11 reference) — shown for conceptual clarity
CK_OBJECT_CLASS keyClass = CKO_PRIVATE_KEY;
CK_KEY_TYPE keyType = CKK_RSA;
CK_BBOOL isToken = CK_TRUE;           // Store on token (persistent)
CK_BBOOL isPrivate = CK_TRUE;         // Require login to access
CK_BBOOL isSign = CK_TRUE;            // Can be used for signing
CK_BBOOL isDecrypt = CK_FALSE;        // Cannot be used for decryption
CK_BBOOL isExtractable = CK_FALSE;    // Cannot be extracted from HSM
CK_BBOOL isSensitive = CK_TRUE;       // Value is hidden from application
```

The critical attributes for IAM signing keys:

- `CKA_TOKEN = true` — Key persists across sessions (stored in HSM).
- `CKA_EXTRACTABLE = false` — Key can never leave the HSM.
- `CKA_SENSITIVE = true` — Key value is never returned to the application.
- `CKA_SIGN = true` — Key is authorized for signing operations.

### 2.4 Go PKCS#11 Library: github.com/miekg/pkcs11

The `github.com/miekg/pkcs11` package provides CGo bindings to the PKCS#11 C
library. It is the most widely used Go PKCS#11 library.

**Installation:**

```bash
go get github.com/miekg/pkcs11/v4
```

**System dependencies:**

```bash
# Ubuntu/Debian (for SoftHSM2 library)
apt-get install -y softhsm2 opensc

# macOS
brew install softhsm

# The PKCS#11 shared library path:
# Linux:   /usr/lib/softhsm/libsofthsm2.so
# macOS:   /usr/local/lib/softhsm2/libsofthsm2.so
# AWS:     /opt/cloudhsm/lib/libcloudhsm_pkcs11.so
```

### 2.5 PKCS#11 Signing in Go — Complete Example

The following code demonstrates the full lifecycle: initialize the library, open
a session, log in, find the private key by label, and perform an RSA-PKCS1v15
signature.

```go
package main

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/miekg/pkcs11/v4"
)

// PKCS11Signer implements crypto.Signer using an HSM via PKCS#11.
type PKCS11Signer struct {
	ctx       *pkcs11.Ctx
	libPath   string
	slotID    uint
	pin       string
	keyLabel  string // Label of the private key object in the HSM
	pubKeyDER []byte // Cached public key DER bytes
}

// NewPKCS11Signer creates a new signer backed by a PKCS#11 token.
func NewPKCS11Signer(libPath string, slotID uint, pin, keyLabel string) (*PKCS11Signer, error) {
	ctx := pkcs11.New(libPath)
	if ctx == nil {
		return nil, fmt.Errorf("failed to load PKCS#11 library: %s", libPath)
	}

	if err := ctx.Initialize(); err != nil {
		return nil, fmt.Errorf("PKCS#11 initialize: %w", err)
	}

	s := &PKCS11Signer{
		ctx:      ctx,
		libPath:  libPath,
		slotID:   slotID,
		pin:      pin,
		keyLabel: keyLabel,
	}

	// Pre-load the public key for Public() method
	pubDER, err := s.loadPublicKeyDER()
	if err != nil {
		return nil, fmt.Errorf("load public key: %w", err)
	}
	s.pubKeyDER = pubDER

	return s, nil
}

// findPrivateKey locates the private key object by its CKA_LABEL.
func (s *PKCS11Signer) findPrivateKey(session pkcs11.SessionHandle) (pkcs11.ObjectHandle, error) {
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, s.keyLabel),
	}

	if err := s.ctx.FindObjectsInit(session, template); err != nil {
		return 0, fmt.Errorf("FindObjectsInit: %w", err)
	}
	objs, _, err := s.ctx.FindObjects(session, 1)
	if err != nil {
		return 0, fmt.Errorf("FindObjects: %w", err)
	}
	if err := s.ctx.FindObjectsFinal(session); err != nil {
		return 0, fmt.Errorf("FindObjectsFinal: %w", err)
	}
	if len(objs) == 0 {
		return 0, fmt.Errorf("private key with label %q not found in token", s.keyLabel)
	}
	return objs[0], nil
}

// loadPublicKeyDER retrieves the public key from the HSM (public keys are
// extractable; only private keys are non-extractable).
func (s *PKCS11Signer) loadPublicKeyDER() ([]byte, error) {
	session, err := s.ctx.OpenSession(s.slotID, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		return nil, fmt.Errorf("OpenSession: %w", err)
	}
	defer s.ctx.CloseSession(session)

	if err := s.ctx.Login(session, pkcs11.CKU_USER, s.pin); err != nil {
		return nil, fmt.Errorf("Login: %w", err)
	}
	defer s.ctx.Logout(session)

	// Find the public key with the same label
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, s.keyLabel),
	}
	if err := s.ctx.FindObjectsInit(session, template); err != nil {
		return nil, err
	}
	objs, _, err := s.ctx.FindObjects(session, 1)
	if err := nil { // corrected below
	}
	if err := s.ctx.FindObjectsFinal(session); err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		return nil, fmt.Errorf("public key not found")
	}

	// Retrieve the DER-encoded public key
	attrs, err := s.ctx.GetAttributeValue(session, objs[0], []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, nil),
	})
	if err != nil {
		return nil, fmt.Errorf("GetAttributeValue: %w", err)
	}
	return attrs[0].Value, nil
}

// Sign implements crypto.Signer.Sign for RSA-PKCS1v15-SHA256.
// This is the method that jwt-go calls when signing a JWT with RS256.
func (s *PKCS11Signer) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	hashed := digest
	if opts.HashFunc() != 0 && len(digest) != opts.HashFunc().Size() {
		h := opts.HashFunc().New()
		h.Write(digest) // In practice, callers pass already-hashed data for JWT signing
		hashed = h.Sum(nil)
	}

	session, err := s.ctx.OpenSession(s.slotID, pkcs11.CKF_SERIAL_SESSION)
	if err != nil {
		return nil, fmt.Errorf("OpenSession: %w", err)
	}
	defer s.ctx.CloseSession(session)

	if err := s.ctx.Login(session, pkcs11.CKU_USER, s.pin); err != nil {
		return nil, fmt.Errorf("Login: %w", err)
	}
	defer s.ctx.Logout(session)

	privKey, err := s.findPrivateKey(session)
	if err != nil {
		return nil, err
	}

	mechanism := []*pkcs11.Mechanism{
		pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS, nil), // RSA PKCS#1 v1.5
	}

	if err := s.ctx.SignInit(session, mechanism, privKey); err != nil {
		return nil, fmt.Errorf("SignInit: %w", err)
	}

	// PKCS#11 expects the DigestInfo-prefixed hash for CKM_RSA_PKCS.
	// For SHA-256, the DigestInfo prefix is a fixed DER structure.
	digestInfo := prefixDigestInfo(crypto.SHA256, hashed)

	sig, err := s.ctx.Sign(session, digestInfo)
	if err != nil {
		return nil, fmt.Errorf("Sign: %w", err)
	}

	return sig, nil
}

// Public implements crypto.Signer.Public.
func (s *PKCS11Signer) Public() crypto.PublicKey {
	pub, err := x509.ParsePKIXPublicKey(s.pubKeyDER)
	if err != nil {
		return nil
	}
	return pub
}

// Close releases the PKCS#11 context.
func (s *PKCS11Signer) Close() {
	s.ctx.Finalize()
	s.ctx.Destroy()
}

// prefixDigestInfo prepends the PKCS#1 v1.5 DigestInfo DER header for the
// given hash algorithm. This is required when using CKM_RSA_PKCS (not
// CKM_RSA_PKCS_PSS or CKM_RSA_X_509).
func prefixDigestInfo(hash crypto.Hash, digest []byte) []byte {
	switch hash {
	case crypto.SHA256:
		// DER: SEQUENCE { SEQUENCE { OID(2.16.840.1.101.3.4.2.1) NULL }, OCTET STRING(digest) }
		prefix := []byte{
			0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86,
			0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05,
			0x00, 0x04, 0x20,
		}
		return append(prefix, digest...)
	case crypto.SHA384:
		prefix := []byte{
			0x30, 0x41, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86,
			0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x02, 0x05,
			0x00, 0x04, 0x30,
		}
		return append(prefix, digest...)
	case crypto.SHA512:
		prefix := []byte{
			0x30, 0x51, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86,
			0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x03, 0x05,
			0x00, 0x04, 0x40,
		}
		return append(prefix, digest...)
	default:
		return digest
	}
}
```

### 2.6 ECDSA Signing via PKCS#11

For ECDSA (e.g., ES256 = ECDSA P-256 + SHA-256), the mechanism is simpler because
ECDSA does not require DigestInfo prefixing:

```go
func (s *PKCS11Signer) SignECDSA(digest []byte) ([]byte, error) {
	session, err := s.ctx.OpenSession(s.slotID, pkcs11.CKF_SERIAL_SESSION)
	if err != nil {
		return nil, err
	}
	defer s.ctx.CloseSession(session)

	if err := s.ctx.Login(session, pkcs11.CKU_USER, s.pin); err != nil {
		return nil, err
	}
	defer s.ctx.Logout(session)

	privKey, err := s.findPrivateKey(session)
	if err != nil {
		return nil, err
	}

	// CKM_ECDSA takes a raw hash digest (no DigestInfo prefix)
	mechanism := []*pkcs11.Mechanism{
		pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil),
	}

	if err := s.ctx.SignInit(session, mechanism, privKey); err != nil {
		return nil, err
	}

	// PKCS#11 ECDSA returns a raw R||S concatenation (64 bytes for P-256).
	// JWT libraries expect ASN.1 DER-encoded ECDSA signatures.
	rawSig, err := s.ctx.Sign(session, digest)
	if err != nil {
		return nil, err
	}

	// Convert raw R||S to ASN.1 DER
	return rawECDSAToDER(rawSig), nil
}

// rawECDSAToDER converts a raw R||S byte concatenation to ASN.1 DER format
// expected by Go's crypto/ecdsa and JWT libraries.
func rawECDSAToDER(rawSig []byte) []byte {
	if len(rawSig) != 64 {
		return rawSig // Unknown format, return as-is
	}
	r := new(big.Int).SetBytes(rawSig[:32])
	s := new(big.Int).SetBytes(rawSig[32:])
	return encodeSignatureDER(r, s)
}
```

### 2.7 Key Generation Inside the HSM

Keys should be generated inside the HSM so they never exist in application
memory. Here is how to generate an RSA 2048-bit key pair using PKCS#11:

```go
func (s *PKCS11Signer) GenerateRSAKey(session pkcs11.SessionHandle, label string, bits int) error {
	// RSA key generation mechanism
	mechanism := []*pkcs11.Mechanism{
		pkcs11.NewMechanism(pkcs11.CKM_RSA_PKCS_KEY_PAIR_GEN, nil),
	}

	// Public key template
	publicTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),         // Persist on token
		pkcs11.NewAttribute(pkcs11.CKA_VERIFY, true),         // Can verify signatures
		pkcs11.NewAttribute(pkcs11.CKA_ENCRYPT, false),
		pkcs11.NewAttribute(pkcs11.CKA_MODULUS_BITS, bits),   // Key size
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
		pkcs11.NewAttribute(pkcs11.CKA_PUBLIC_EXPONENT, []byte{0x01, 0x00, 0x01}), // 65537
	}

	// Private key template — the critical security attributes
	privateTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_RSA),
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
		pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),         // Requires login
		pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),       // Never returned to app
		pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),    // Cannot be extracted
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),            // Can sign
		pkcs11.NewAttribute(pkcs11.CKA_DECRYPT, false),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, label),
	}

	pubKey, _, err := s.ctx.GenerateKeyPair(
		session,
		mechanism,
		publicTemplate,
		privateTemplate,
	)
	if err != nil {
		return fmt.Errorf("GenerateKeyPair: %w", err)
	}

	_ = pubKey // Public key handle — can retrieve DER via GetAttributeValue
	return nil
}
```

### 2.8 Session Management Best Practices

PKCS#11 sessions are **not thread-safe** — each goroutine needs its own session.
For high-throughput signing, maintain a session pool:

```go
// SessionPool manages a pool of PKCS#11 sessions for concurrent use.
type SessionPool struct {
	ctx      *pkcs11.Ctx
	slotID   uint
	pin      string
	sessions chan pkcs11.SessionHandle
}

func NewSessionPool(ctx *pkcs11.Ctx, slotID uint, pin string, size int) (*SessionPool, error) {
	pool := &SessionPool{
		ctx:      ctx,
		slotID:   slotID,
		pin:      pin,
		sessions: make(chan pkcs11.SessionHandle, size),
	}

	for i := 0; i < size; i++ {
		session, err := ctx.OpenSession(slotID, pkcs11.CKF_SERIAL_SESSION)
		if err != nil {
			return nil, fmt.Errorf("OpenSession %d: %w", i, err)
		}
		if err := ctx.Login(session, pkcs11.CKU_USER, pin); err != nil {
			return nil, fmt.Errorf("Login %d: %w", i, err)
		}
		pool.sessions <- session
	}

	return pool, nil
}

func (p *SessionPool) Get() pkcs11.SessionHandle {
	return <-p.sessions
}

func (p *SessionPool) Put(session pkcs11.SessionHandle) {
	p.sessions <- session
}

func (p *SessionPool) Close() {
	close(p.sessions)
	for session := range p.sessions {
		p.ctx.Logout(session)
		p.ctx.CloseSession(session)
	}
}
```

---

## 3. SoftHSM2 for Development

### 3.1 What Is SoftHSM2?

SoftHSM2 is a software implementation of a generic cryptographic device with a
PKCS#11 interface. It is developed by the OpenDNSSEC project and provides a
complete PKCS#11 API backed by software key storage (files on disk). It is not a
real HSM — there is no hardware protection — but it is invaluable for developing
and testing PKCS#11 integration without expensive hardware.

### 3.2 Installation

```bash
# Ubuntu/Debian
apt-get install -y softhsm2 opensc

# macOS
brew install softhsm

# From source
git clone https://github.com/opendnssec/SoftHSMv2.git
cd SoftHSMv2
./autogen.sh
./configure --disable-gost
make
sudo make install
```

### 3.3 Configuration

SoftHSM2 stores tokens in directories. Create a configuration file:

```ini
# ~/.config/softhsm2/softhsm2.conf (or /etc/softhsm2.conf)
directories.tokendir = /tmp/softhsm2/tokens
objectstore.backend = file
log.level = INFO
```

Set the environment variable:
```bash
export SOFTHSM2_CONF=/path/to/softhsm2.conf
```

### 3.4 Creating Tokens and Keys via CLI

```bash
# Create token directory
mkdir -p /tmp/softhsm2/tokens

# Initialize a new token in slot 0 with label "GGID_SIGNING"
softhsm2-util --init-token --slot 0 --label "GGID_SIGNING" \
    --so-pin "12345678" --pin "1234"

# Verify the token exists
softhsm2-util --show-slots

# List available mechanisms for the token
pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so \
    --list-mechanisms --slot 0
```

Generate keys using `pkcs11-tool` or directly in your Go application:

```bash
# Generate RSA 2048-bit key pair inside SoftHSM2
pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so \
    --slot 0 --login --pin 1234 \
    --keygen --key-type rsa:2048 \
    --label "jwt-signing-key" \
    --id 01

# List objects on the token
pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so \
    --slot 0 --login --pin 1234 \
    --list-objects

# Test signing
echo -n "test data" | pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so \
    --slot 0 --login --pin 1234 \
    --sign --label "jwt-signing-key" \
    --mechanism SHA256-RSA-PKCS | \
    base64
```

### 3.5 Docker Setup for SoftHSM2

For CI/CD and development, package SoftHSM2 in a Docker container:

```dockerfile
# Dockerfile.softhsm2
FROM golang:1.25-bookworm AS builder

# Install SoftHSM2
RUN apt-get update && apt-get install -y softhsm2 opensc

# Create token directory and initialize
RUN mkdir -p /tmp/softhsm2/tokens && \
    echo "directories.tokendir = /tmp/softhsm2/tokens" > /etc/softhsm2.conf

ENV SOFTHSM2_CONF=/etc/softhsm2.conf

WORKDIR /app
COPY . .
RUN go mod download

# Run integration test
CMD ["go", "test", "-v", "-tags=pkcs11", "./test/pkcs11/..."]
```

```yaml
# docker-compose.softhsm2.yml
version: "3.8"
services:
  ggid-auth-pkcs11-test:
    build:
      context: .
      dockerfile: Dockerfile.softhsm2
    environment:
      - SOFTHSM2_CONF=/etc/softhsm2.conf
      - GGID_PKCS11_LIB=/usr/lib/softhsm/libsofthsm2.so
      - GGID_PKCS11_SLOT=0
      - GGID_PKCS11_PIN=1234
      - GGID_PKCS11_KEY_LABEL=jwt-signing-key
    volumes:
      - softhsm-tokens:/tmp/softhsm2/tokens

volumes:
  softhsm-tokens:
```

### 3.6 Go Integration Test with SoftHSM2

```go
//go:build pkcs11

package pkcs11_test

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"os"
	"testing"

	"github.com/miekg/pkcs11/v4"
)

// TestPKCS11SoftHSM2Signing tests RSA signing via SoftHSM2.
// This test requires SoftHSM2 to be installed and initialized.
//
// Setup:
//   export SOFTHSM2_CONF=/etc/softhsm2.conf
//   softhsm2-util --init-token --slot 0 --label "TEST" --so-pin 12345678 --pin 1234
//   pkcs11-tool --keygen --key-type rsa:2048 --label test-key --login --pin 1234
func TestPKCS11SoftHSM2Signing(t *testing.T) {
	libPath := os.Getenv("GGID_PKCS11_LIB")
	if libPath == "" {
		libPath = "/usr/lib/softhsm/libsofthsm2.so"
	}

	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		t.Skipf("SoftHSM2 library not found at %s, skipping", libPath)
	}

	pin := os.Getenv("GGID_PKCS11_PIN")
	if pin == "" {
		pin = "1234"
	}

	ctx := pkcs11.New(libPath)
	if ctx == nil {
		t.Fatalf("failed to load PKCS#11 library: %s", libPath)
	}
	defer ctx.Finalize()
	defer ctx.Destroy()

	if err := ctx.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// Find the slot with our token
	slots, err := ctx.GetSlotList(true)
	if err != nil {
		t.Fatalf("GetSlotList: %v", err)
	}
	if len(slots) == 0 {
		t.Skip("no slots available, run softhsm2-util --init-token first")
	}

	slot := slots[0]

	session, err := ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION)
	if err != nil {
		t.Fatalf("OpenSession: %v", err)
	}
	defer ctx.CloseSession(session)

	if err := ctx.Login(session, pkcs11.CKU_USER, pin); err != nil {
		t.Fatalf("Login: %v", err)
	}
	defer ctx.Logout(session)

	// Find the private key by label
	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, "test-key"),
	}

	if err := ctx.FindObjectsInit(session, template); err != nil {
		t.Fatalf("FindObjectsInit: %v", err)
	}
	objs, _, err := ctx.FindObjects(session, 1)
	if err != nil {
		t.Fatalf("FindObjects: %v", err)
	}
	ctx.FindObjectsFinal(session)

	if len(objs) == 0 {
		t.Skip("test-key not found, run pkcs11-tool --keygen first")
	}

	// Sign data
	data := []byte("test message to sign")
	h := sha256.Sum256(data)

	mech := []*pkcs11.Mechanism{
		pkcs11.NewMechanism(pkcs11.CKM_SHA256_RSA_PKCS, nil),
	}

	if err := ctx.SignInit(session, mech, objs[0]); err != nil {
		t.Fatalf("SignInit: %v", err)
	}

	signature, err := ctx.Sign(session, data) // CKM_SHA256_RSA_PKCS hashes internally
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	if len(signature) != 256 { // RSA-2048 signature = 256 bytes
		t.Errorf("unexpected signature length: got %d, want 256", len(signature))
	}

	// Verify with public key
	pubKey := getPublicKeyFromHSM(t, ctx, session)
	err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, h[:], signature)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}

	t.Logf("SoftHSM2 PKCS#11 signing test passed, signature length: %d", len(signature))
}

func getPublicKeyFromHSM(t *testing.T, ctx *pkcs11.Ctx, session pkcs11.SessionHandle) *rsa.PublicKey {
	t.Helper()

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, "test-key"),
	}

	ctx.FindObjectsInit(session, template)
	objs, _, _ := ctx.FindObjects(session, 1)
	ctx.FindObjectsFinal(session)

	if len(objs) == 0 {
		t.Fatal("public key not found")
	}

	attrs, err := ctx.GetAttributeValue(session, objs[0], []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_MODULUS, nil),
		pkcs11.NewAttribute(pkcs11.CKA_PUBLIC_EXPONENT, nil),
	})
	if err != nil {
		t.Fatalf("GetAttributeValue: %v", err)
	}

	// Reconstruct RSA public key from modulus and exponent
	var modulus, exponent []byte
	for _, attr := range attrs {
		switch attr.Type {
		case pkcs11.CKA_MODULUS:
			modulus = attr.Value
		case pkcs11.CKA_PUBLIC_EXPONENT:
			exponent = attr.Value
		}
	}

	n := new(big.Int).SetBytes(modulus)
	e := new(big.Int).SetBytes(exponent).Int64()

	return &rsa.PublicKey{N: n, E: int(e)}
}
```

### 3.7 SoftHSM2 Limitations

| Feature | SoftHSM2 | Real HSM |
|---|---|---|
| Key isolation | No (keys are in files) | Yes (tamper-resistant hardware) |
| FIPS certification | None | Level 2–4 |
| Performance | CPU-bound (same as software crypto) | Hardware-accelerated |
| Tamper response | None | Active key zeroization |
| Key wrapping | Supported | Supported + hardware-backed |
| Audit trail | File-based logs | Internal, tamper-proof |

SoftHSM2 is excellent for **API testing and development**. It lets you test all
PKCS#11 code paths (session management, key lookup, signing) without a real HSM.
But it provides **no actual security** — an attacker with filesystem access can
read the keys.

---

## 4. AWS KMS Integration

### 4.1 Overview

AWS Key Management Service (KMS) is a managed service for creating and managing
cryptographic keys. It is FIPS 140-2 Level 2 validated (Level 3 with AWS
CloudHSM). KMS integrates with IAM for access control and CloudTrail for audit
logging.

### 4.2 Key Types

| Key Type | Algorithms | Use Case |
|---|---|---|
| **SYMMETRIC_DEFAULT** | AES-256-GCM | Envelope encryption, data encryption |
| **RSA_2048 / RSA_3072 / RSA_4096** | RSA-PKCS1v15, RSA-PSS | JWT signing, SAML signing |
| **ECC_NIST_P256 / P384 / P521** | ECDSA | JWT signing (ES256, ES384, ES512) |
| **HMAC_256 / HMAC_384 / HMAC_512** | HMAC | Token signing (HS256), MAC |
| **SM2** (China regions) | SM2 | Chinese compliance |

### 4.3 Envelope Encryption

Envelope encryption is the standard pattern for encrypting data with KMS:

1. Generate a **data encryption key (DEK)** using the KMS master key.
2. KMS returns both the **plaintext DEK** and the **encrypted DEK**.
3. Encrypt the data locally using the plaintext DEK (AES-256-GCM).
4. **Discard** the plaintext DEK from memory.
5. Store the ciphertext + encrypted DEK.
6. To decrypt: call KMS `Decrypt` with the encrypted DEK to get the plaintext DEK,
   then decrypt locally.

```go
package kmscrypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// KMSEnvelopeEncryptor implements envelope encryption using AWS KMS.
type KMSEnvelopeEncryptor struct {
	client   *kms.Client
	keyID    string // KMS key ARN or ID
}

func NewKMSEnvelopeEncryptor(client *kms.Client, keyID string) *KMSEnvelopeEncryptor {
	return &KMSEnvelopeEncryptor{client: client, keyID: keyID}
}

// Encrypt encrypts plaintext using KMS envelope encryption.
// Returns: encryptedDEK || nonce || ciphertext
func (e *KMSEnvelopeEncryptor) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	// Step 1: Generate data key from KMS
	genKeyResp, err := e.client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:         aws.String(e.keyID),
		KeySpec:       types.DataKeySpecAes256, // 32-byte DEK
		EncryptionContext: map[string]string{
			"service": "ggid",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("KMS GenerateDataKey: %w", err)
	}

	plaintextDEK := genKeyResp.Plaintext       // 32 bytes
	encryptedDEK := genKeyResp.CiphertextBlob  // opaque, to be stored

	// Step 2: Encrypt data locally with the DEK
	block, err := aes.NewCipher(plaintextDEK)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Zero out the plaintext DEK immediately
	for i := range plaintextDEK {
		plaintextDEK[i] = 0
	}

	// Step 3: Pack the output: [4-byte DEK length][encryptedDEK][nonce][ciphertext]
	output := make([]byte, 0, 4+len(encryptedDEK)+len(nonce)+len(ciphertext))
	dekLen := uint32(len(encryptedDEK))
	output = append(output, byte(dekLen>>24), byte(dekLen>>16), byte(dekLen>>8), byte(dekLen))
	output = append(output, encryptedDEK...)
	output = append(output, nonce...)
	output = append(output, ciphertext...)

	return output, nil
}

// Decrypt decrypts data encrypted by Encrypt().
func (e *KMSEnvelopeEncryptor) Decrypt(ctx context.Context, packed []byte) ([]byte, error) {
	if len(packed) < 4 {
		return nil, fmt.Errorf("packed data too short")
	}

	dekLen := uint32(packed[0])<<24 | uint32(packed[1])<<16 | uint32(packed[2])<<8 | uint32(packed[3])
	if len(packed) < int(4+dekLen) {
		return nil, fmt.Errorf("invalid DEK length")
	}

	encryptedDEK := packed[4 : 4+dekLen]
	rest := packed[4+dekLen:]

	// Step 1: Decrypt the DEK via KMS
	decryptResp, err := e.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob: encryptedDEK,
		EncryptionContext: map[string]string{
			"service": "ggid",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("KMS Decrypt: %w", err)
	}
	plaintextDEK := decryptResp.Plaintext
	defer func() {
		for i := range plaintextDEK {
			plaintextDEK[i] = 0
		}
	}()

	// Step 2: Decrypt data locally
	block, err := aes.NewCipher(plaintextDEK)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(rest) < nonceSize {
		return nil, fmt.Errorf("nonce + ciphertext too short")
	}

	nonce := rest[:nonceSize]
	ciphertext := rest[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM decrypt: %w", err)
	}

	return plaintext, nil
}
```

### 4.4 Signing with AWS KMS (Asymmetric Keys)

For JWT signing, use an asymmetric KMS key (RSA or ECC):

```go
package kmscrypto

import (
	"context"
	"crypto"
	"crypto/sha256"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// KMSSigner implements crypto.Signer backed by AWS KMS.
type KMSSigner struct {
	client     *kms.Client
	keyID      string
	algorithm  types.SigningAlgorithmSpec
	publicKey  crypto.PublicKey
}

// NewKMSSigner creates a signer using an asymmetric KMS key.
// algorithm: types.SigningAlgorithmSpecRsassaPkcs1V1Sha256 for RS256
//            types.SigningAlgorithmSpecEcdsaSha256 for ES256
func NewKMSSigner(client *kms.Client, keyID string, algorithm types.SigningAlgorithmSpec) (*KMSSigner, error) {
	s := &KMSSigner{
		client:    client,
		keyID:     keyID,
		algorithm: algorithm,
	}

	// Fetch and cache the public key
	if err := s.loadPublicKey(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *KMSSigner) loadPublicKey(ctx context.Context) error {
	resp, err := s.client.GetPublicKey(ctx, &kms.GetPublicKeyInput{
		KeyId: aws.String(s.keyID),
	})
	if err != nil {
		return fmt.Errorf("KMS GetPublicKey: %w", err)
	}

	// Parse the public key from DER
	pub, err := x509.ParsePKIXPublicKey(resp.PublicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	s.publicKey = pub
	return nil
}

// Public implements crypto.Signer.Public.
func (s *KMSSigner) Public() crypto.PublicKey {
	return s.publicKey
}

// Sign implements crypto.Signer.Sign.
// The digest must already be hashed (e.g., SHA-256 for RS256).
func (s *KMSSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	ctx := context.Background()

	resp, err := s.client.Sign(ctx, &kms.SignInput{
		KeyId:            aws.String(s.keyID),
		Message:          digest,
		MessageType:      types.MessageTypeDigest, // We pass a pre-computed hash
		SigningAlgorithm: s.algorithm,
	})
	if err != nil {
		return nil, fmt.Errorf("KMS Sign: %w", err)
	}

	return resp.Signature, nil
}

// HashFunc returns the hash algorithm for this signer.
func (s *KMSSigner) HashFunc() crypto.Hash {
	switch s.algorithm {
	case types.SigningAlgorithmSpecRsassaPkcs1V1Sha256,
		types.SigningAlgorithmSpecRsassaPssSha256,
		types.SigningAlgorithmSpecEcdsaSha256:
		return crypto.SHA256
	case types.SigningAlgorithmSpecRsassaPkcs1V1Sha384,
		types.SigningAlgorithmSpecRsassaPssSha384,
		types.SigningAlgorithmSpecEcdsaSha384:
		return crypto.SHA384
	case types.SigningAlgorithmSpecRsassaPkcs1V1Sha512,
		types.SigningAlgorithmSpecRsassaPssSha512,
		types.SigningAlgorithmSpecEcdsaSha512:
		return crypto.SHA512
	default:
		return 0
	}
}
```

### 4.5 Using KMSSigner with JWT

```go
import (
	"github.com/golang-jwt/jwt/v5"
)

func signJWTWithKMS(signer *KMSSigner, claims jwt.Claims, keyID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID

	// jwt.SigningString calls signer.Sign(rand, digest, opts)
	// where digest is the SHA-256 hash of the JWT signing input
	return token.SignedString(signer)
}
```

### 4.6 IAM Policy for KMS Access

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowKMSDescribeAndSign",
      "Effect": "Allow",
      "Action": [
        "kms:DescribeKey",
        "kms:GetPublicKey",
        "kms:Sign"
      ],
      "Resource": "arn:aws:kms:us-east-1:123456789012:key/<key-id>",
      "Condition": {
        "StringEquals": {
          "kms:SigningAlgorithm": "RSASSA_PKCS1_V1_5_SHA_256"
        }
      }
    },
    {
      "Sid": "AllowKMSEncryptDecrypt",
      "Effect": "Allow",
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:GenerateDataKey"
      ],
      "Resource": "arn:aws:kms:us-east-1:123456789012:key/<data-key-id>",
      "Condition": {
        "StringEquals": {
          "kms:ViaService": "secretsmanager.us-east-1.amazonaws.com"
        }
      }
    },
    {
      "Sid": "DenyKMSDeleteAndRotate",
      "Effect": "Deny",
      "Action": [
        "kms:DeleteKey",
        "kms:DisableKey",
        "kms:ScheduleKeyDeletion",
        "kms:RotateKey"
      ],
      "Resource": "*"
    }
  ]
}
```

Key policy conditions to enforce:
- `kms:EncryptionContext:service` — Must match expected context.
- `kms:ViaService` — Restrict which AWS services can use the key.
- `kms:CallerAccount` — Restrict to your AWS account.
- `aws:SourceIp` — Restrict to known IP ranges.

### 4.7 Cost Model

| Operation | Price |
|---|---|
| Key storage | $1.00 per key per month |
| Asymmetric key (RSA/ECC) storage | $1.00 per key per month |
| 10,000 symmetric API requests | $0.03 |
| 10,000 asymmetric API requests (Sign/Verify) | $0.15 |
| 10,000 GenerateDataKey requests | $0.03 |
| 10,000 RSA/ECC GetPublicKey requests | $0.15 |
| Custom key store (CloudHSM backed) | $1.499/hour per HSM (~$1,092/month) |

**Estimated monthly cost for GGID:**

Assuming:
- 2 KMS keys (JWT signing key + data encryption key): $2/month
- 500,000 token validations per month → 50 × $0.15 (asymmetric Sign): $7.50
- 100,000 data encryption operations → 10 × $0.03: $0.30
- CloudTrail logging: free (included)

**Total: ~$10/month** for standard usage. This is dramatically cheaper than a
dedicated HSM ($1,000–$13,000/month).

### 4.8 Key Rotation in AWS KMS

AWS KMS supports automatic key rotation for symmetric keys (annual rotation,
free). For asymmetric keys, you must manually create a new key version and update
your application's `keyID`.

```go
// RotateKMSKey creates a new KMS key and updates the application's signing key.
func RotateKMSKey(ctx context.Context, client *kms.Client, description string) (string, error) {
	resp, err := client.CreateKey(ctx, &kms.CreateKeyInput{
		Description: aws.String(description),
		KeySpec:     types.KeySpecRsa2048,
		KeyUsage:    types.KeyUsageTypeSignVerify,
		Tags: []types.Tag{
			{TagKey: aws.String("Service"), TagValue: aws.String("ggid")},
			{TagKey: aws.String("Purpose"), TagValue: aws.String("jwt-signing")},
			{TagKey: aws.String("Rotation"), TagValue: aws.String(aws.ToString(time.Now().Format("2006-01")))},
		},
	})
	if err != nil {
		return "", fmt.Errorf("CreateKey: %w", err)
	}
	return *resp.KeyMetadata.KeyId, nil
}
```

---

## 5. Google Cloud KMS Integration

### 5.1 Overview

Google Cloud Key Management Service (Cloud KMS) provides managed cryptographic
key management. It offers both **software-protected** keys and **HSM-protected**
keys (FIPS 140-2 Level 3). Cloud KMS has a hierarchical structure: Project →
Location → Key Ring → CryptoKey → CryptoKeyVersion.

### 5.2 Key Hierarchy

```
projects/ggid-prod/
  locations/us-east1/
    keyRings/
      ggid-signing/
        cryptoKeys/
          jwt-rsa-2048/
            cryptoKeyVersions/
              1  (current, enabled)
              2  (previous, still enabled for verification overlap)
              3  (next, scheduled for activation)
          jwt-ecdsa-p256/
            cryptoKeyVersions/
              1  (current, enabled)
      ggid-data/
        cryptoKeys/
          aes-256-data/
            cryptoKeyVersions/
              1  (current)
```

### 5.3 Supported Algorithms

| Algorithm | Type | Key Size | Hash |
|---|---|---|---|
| `RSA_SIGN_PKCS1_2048_SHA256` | RSA PKCS#1 v1.5 | 2048 | SHA-256 |
| `RSA_SIGN_PKCS1_3072_SHA256` | RSA PKCS#1 v1.5 | 3072 | SHA-256 |
| `RSA_SIGN_PKCS1_4096_SHA256` | RSA PKCS#1 v1.5 | 4096 | SHA-256 |
| `RSA_SIGN_PSS_2048_SHA256` | RSA-PSS | 2048 | SHA-256 |
| `EC_SIGN_P256_SHA256` | ECDSA P-256 | 256 | SHA-256 |
| `EC_SIGN_P384_SHA384` | ECDSA P-384 | 384 | SHA-384 |
| `GOOGLE_SYMMETRIC_ENCRYPTION` | AES-256 | 256 | N/A |
| `HMAC_SHA256` | HMAC | 256 | SHA-256 |
| `EXTERNAL_SYMMETRIC_ENCRYPTION` | EKM-backed | 256 | N/A |

### 5.4 Go Code: Cloud KMS Signing

```go
package cloudkms

import (
	"context"
	"crypto"
	"fmt"
	"io"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/option"
)

// GoogleKMSSigner implements crypto.Signer using Google Cloud KMS.
type GoogleKMSSigner struct {
	client     *kms.KeyManagementClient
	keyVersion string // projects/.../keyRings/.../cryptoKeys/.../cryptoKeyVersions/1
	algorithm  kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm
	publicKey  crypto.PublicKey
}

// NewGoogleKMSSigner creates a signer using Google Cloud KMS.
func NewGoogleKMSSigner(ctx context.Context, keyVersion string, algorithm kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm, opts ...option.ClientOption) (*GoogleKMSSigner, error) {
	client, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create KMS client: %w", err)
	}

	s := &GoogleKMSSigner{
		client:     client,
		keyVersion: keyVersion,
		algorithm:  algorithm,
	}

	if err := s.loadPublicKey(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *GoogleKMSSigner) loadPublicKey(ctx context.Context) error {
	resp, err := s.client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{
		Name: s.keyVersion,
	})
	if err != nil {
		return fmt.Errorf("GetPublicKey: %w", err)
	}

	// resp.Pem is a PEM-encoded public key
	block, _ := pem.Decode([]byte(resp.Pem))
	if block == nil {
		return fmt.Errorf("failed to decode PEM public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try PKCS1
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse public key: %w", err)
		}
	}

	s.publicKey = pub
	return nil
}

// Public implements crypto.Signer.Public.
func (s *GoogleKMSSigner) Public() crypto.PublicKey {
	return s.publicKey
}

// HashFunc returns the hash for this algorithm.
func (s *GoogleKMSSigner) HashFunc() crypto.Hash {
	switch s.algorithm {
	case kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256,
		kmspb.CryptoKeyVersion_RSA_SIGN_PSS_2048_SHA256,
		kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256:
		return crypto.SHA256
	case kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_3072_SHA384,
		kmspb.CryptoKeyVersion_EC_SIGN_P384_SHA384:
		return crypto.SHA384
	case kmspb.CryptoKeyVersion_EC_SIGN_P521_SHA512:
		return crypto.SHA512
	default:
		return crypto.SHA256
	}
}

// Sign implements crypto.Signer.Sign.
func (s *GoogleKMSSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	ctx := context.Background()

	resp, err := s.client.AsymmetricSign(ctx, &kmspb.AsymmetricSignRequest{
		Name: s.keyVersion,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{Sha256: digest},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("KMS AsymmetricSign: %w", err)
	}

	return resp.Signature, nil
}

// Close releases the KMS client.
func (s *GoogleKMSSigner) Close() error {
	return s.client.Close()
}
```

### 5.5 Creating a Key Ring and Key

```go
func CreateKeyRingAndKey(ctx context.Context, client *kms.KeyManagementClient) error {
	// Create key ring
	krResp, err := client.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{
		Parent:    "projects/ggid-prod/locations/us-east1",
		KeyRingId: "ggid-signing",
	})
	if err != nil {
		// Key ring may already exist — that's OK
		fmt.Printf("CreateKeyRing (may already exist): %v\n", err)
	}
	_ = krResp

	// Create RSA signing key
	keyResp, err := client.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      "projects/ggid-prod/locations/us-east1/keyRings/ggid-signing",
		CryptoKeyId: "jwt-rsa-2048",
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN,
			VersionTemplate: &kmspb.CryptoKeyVersionTemplate{
				Algorithm:        kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256,
				ProtectionLevel:  kmspb.ProtectionLevel_HSM, // Use HSM for FIPS 140-2 Level 3
			},
			RotationPeriod: &durationpb.Duration{
				Seconds: int64(90 * 24 * 3600), // 90-day rotation
			},
			NextRotationTime: &timestamppb.Timestamp{
				Seconds: time.Now().Add(90 * 24 * time.Hour).Unix(),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("CreateCryptoKey: %w", err)
	}

	fmt.Printf("Created key: %s\n", keyResp.Name)
	return nil
}
```

### 5.6 IAM Permissions

```yaml
# IAM binding for the GGID auth service account
bindings:
- members:
  - serviceAccount:ggid-auth@ggid-prod.iam.gserviceaccount.com
  role: roles/cloudkms.signerVerifier
  condition:
    title: restrict-to-signing-key
    expression: >
      resource.name.startsWith(
        'projects/ggid-prod/locations/us-east1/keyRings/ggid-signing/cryptoKeys/jwt-'
      )
```

Required roles:
| Role | Permissions | Who Gets It |
|---|---|---|
| `roles/cloudkms.signerVerifier` | `cloudkms.cryptoKeyVersions.useToSign`, `cloudkms.cryptoKeyVersions.useToVerify` | Auth service SA |
| `roles/cloudkms.cryptoKeyEncrypterDecrypter` | `cloudkms.cryptoKeyVersions.useToEncrypt`, `cloudkms.cryptoKeyVersions.useToDecrypt` | Data services |
| `roles/cloudkms.admin` | Full key management (create, rotate, delete) | Security admin only |
| `roles/cloudkms.viewer` | List/view keys and versions | Audit, monitoring |

### 5.7 Cost Model

| Resource | Price |
|---|---|
| Active key version | $0.06/month (software) |
| Active key version (HSM) | $1.00/month |
| 10,000 asymmetric sign operations | $0.10 (software) / $0.50 (HSM) |
| 10,000 symmetric encrypt/decrypt | $0.03 |
| 10,000 HMAC operations | $0.03 |
| 10,000 key version operations (Create, Get, List) | $0.10 |

**Estimated monthly cost for GGID (HSM-protected):**
- 1 RSA key version: $1.00
- 500,000 sign operations → 50 × $0.50: $25.00
- 1 symmetric key for data encryption: $1.00
- 100,000 data operations → 10 × $0.03: $0.30

**Total: ~$27/month** for HSM-grade key protection.

### 5.8 Automatic Rotation

Cloud KMS supports automatic rotation with configurable periods. The rotation
creates a new key version and automatically makes it primary. Old versions remain
available for verification during the overlap window.

```go
// SetRotation updates rotation policy on a CryptoKey.
func SetRotation(ctx context.Context, client *kms.KeyManagementClient, keyName string) error {
	_, err := client.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{
		CryptoKey: &kmspb.CryptoKey{
			Name: keyName,
			RotationPeriod: &durationpb.Duration{
				Seconds: int64(90 * 24 * 3600), // 90 days
			},
			NextRotationTime: &timestamppb.Timestamp{
				Seconds: time.Now().Add(90 * 24 * time.Hour).Unix(),
			},
		},
		UpdateMask: &fieldmask.FieldMask{
			Paths: []string{"rotation_period", "next_rotation_time"},
		},
	})
	return err
}
```

---

## 6. Azure Key Vault Integration

### 6.1 Overview

Azure Key Vault is Microsoft's managed key management service. It offers two
vault types:

| Vault Type | Key Protection | HSM-backed | Price |
|---|---|---|---|
| **Standard** | Software-protected | No | $0.03/10k operations |
| **Premium** | HSM-protected (FIPS 140-2 Level 2) | Yes | $1.00/key/month |
| **Managed HSM** | Dedicated HSM (FIPS 140-2 Level 3) | Yes | $3.00/key/month |

### 6.2 Key Operations

Azure Key Vault supports the following key types:

| Key Type | Algorithms | Notes |
|---|---|---|
| RSA (2048, 3072, 4096) | RS256, RS384, RS512, PS256, PS384, PS512 | Default for most signing |
| EC (P-256, P-384, P-521, P-256K) | ES256, ES384, ES512 | Smaller signatures |
| OCT (AES) | AES-CBC, AES-GCM | Symmetric encryption |
| BYOK (Bring Your Own Key) | Import from on-prem HSM | Transfer via secure key exchange |

### 6.3 Go Code: Azure Key Vault Signing

```go
package azurekms

import (
	"context"
	"crypto"
	"fmt"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
)

// AzureKVSigner implements crypto.Signer using Azure Key Vault.
type AzureKVSigner struct {
	client    *azkeys.Client
	vaultURL  string // e.g., "https://ggid-kv.vault.azure.net/"
	keyName   string
	keyVersion string // empty for latest
	algorithm azkeys.JSONWebKeySignatureAlgorithm
	publicKey crypto.PublicKey
}

// NewAzureKVSigner creates a signer using Azure Key Vault.
func NewAzureKVSigner(
	vaultURL, keyName, keyVersion string,
	algorithm azkeys.JSONWebKeySignatureAlgorithm,
	cred azcore.TokenCredential,
) (*AzureKVSigner, error) {
	client, err := azkeys.NewClient(vaultURL, cred)
	if err != nil {
		return nil, fmt.Errorf("create Key Vault client: %w", err)
	}

	s := &AzureKVSigner{
		client:     client,
		vaultURL:   vaultURL,
		keyName:    keyName,
		keyVersion: keyVersion,
		algorithm:  algorithm,
	}

	if err := s.loadPublicKey(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *AzureKVSigner) loadPublicKey(ctx context.Context) error {
	resp, err := s.client.GetKey(ctx, azkeys.GetKeyParameters{
		VaultBaseUrl: s.vaultURL,
		KeyName:      s.keyName,
		KeyVersion:   s.keyVersion,
	}, nil)
	if err != nil {
		return fmt.Errorf("GetKey: %w", err)
	}

	// Azure returns the key as a JSONWebKey with an "n" (modulus) and "e" (exponent)
	// for RSA keys, or "x" and "y" for EC keys.
	jwk := resp.Key
	if jwk != nil && jwk.Kty != nil {
		switch *jwk.Kty {
		case azkeys.JSONWebKeyTypeRSA:
			n := new(big.Int).SetBytes(jwk.N)
			e := new(big.Int).SetBytes(jwk.E).Int64()
			s.publicKey = &rsa.PublicKey{N: n, E: int(e)}
		case azkeys.JSONWebKeyTypeEC:
			// Parse EC public key from x, y coordinates
			x := new(big.Int).SetBytes(jwk.X)
			y := new(big.Int).SetBytes(jwk.Y)
			var curve elliptic.Curve
			switch jwk.Crv {
			case azkeys.JSONWebKeyCurveNameP256:
				curve = elliptic.P256()
			case azkeys.JSONWebKeyCurveNameP384:
				curve = elliptic.P384()
			case azkeys.JSONWebKeyCurveNameP521:
				curve = elliptic.P521()
			}
			s.publicKey = &ecdsa.PublicKey{Curve: curve, X: x, Y: y}
		}
	}

	return nil
}

// Public implements crypto.Signer.Public.
func (s *AzureKVSigner) Public() crypto.PublicKey {
	return s.publicKey
}

// Sign implements crypto.Signer.Sign.
func (s *AzureKVSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	ctx := context.Background()

	resp, err := s.client.Sign(ctx, azkeys.SignParameters{
		VaultBaseUrl: s.vaultURL,
		KeyName:      s.keyName,
		KeyVersion:   s.keyVersion,
		Algorithm:    &s.algorithm,
		Value:        digest, // Already hashed
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("Key Vault Sign: %w", err)
	}

	return resp.Result, nil
}
```

### 6.4 Managed Identity for Authentication

Azure's managed identity eliminates the need for stored credentials. The
application authenticates to Key Vault using its Azure AD identity automatically:

```go
func NewAzureKVSignerWithManagedIdentity(
	vaultURL, keyName string,
	algorithm azkeys.JSONWebKeySignatureAlgorithm,
) (*AzureKVSigner, error) {
	// DefaultAzureCredential uses managed identity when running in Azure
	// (VM, App Service, AKS, Functions)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("create credential: %w", err)
	}

	return NewAzureKVSigner(vaultURL, keyName, "", algorithm, cred)
}
```

### 6.5 Creating a Key in Azure Key Vault

```go
func CreateKeyVaultKey(ctx context.Context, client *azkeys.Client) error {
	keyType := azkeys.JSONWebKeyTypeRSA
	keySize := int32(2048)

	_, err := client.CreateKey(ctx, azkeys.CreateKeyParameters{
		VaultBaseUrl: "https://ggid-kv.vault.azure.net/",
		KeyName:      "jwt-signing-key",
		Parameters: &azkeys.KeyCreateParameters{
			Kty:    &keyType,
			KeySize: &keySize,
			KeyOps: []*azkeys.JSONWebKeyOperation{
				azkeys.JSONWebKeyOperation("sign"),
				// Note: do NOT include "decrypt" — principle of least privilege
			},
			Attributes: &azkeys.KeyAttributes{
				Enabled:     ptr(true),
				Expires:     ptr(time.Now().Add(365 * 24 * time.Hour)),
				NotBefore:   ptr(time.Now()),
			},
			Tags: map[string]*string{
				"Service":  ptr("ggid"),
				"Purpose":  ptr("jwt-signing"),
			},
		},
	}, nil)

	return err
}

func ptr[T any](v T) *T { return &v }
```

### 6.6 Access Policy / RBAC

Azure Key Vault supports two authorization models:

**Access Policies (legacy):**
```json
{
  "accessPolicies": [
    {
      "tenantId": "<tenant-id>",
      "objectId": "<auth-service-object-id>",
      "permissions": {
        "keys": ["sign", "get"],
        "secrets": ["get"],
        "certificates": []
      }
    }
  ]
}
```

**RBAC (recommended):**
```bash
# Assign the "Key Vault Crypto User" role to the auth service managed identity
az role assignment create \
    --role "Key Vault Crypto User" \
    --assignee <managed-identity-principal-id> \
    --scope /subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/<kv-name>
```

### 6.7 Cost Model

| Resource | Price |
|---|---|
| Standard vault — 10,000 operations | $0.03 |
| Premium vault (HSM) — per key/month | $1.00 |
| Premium vault — 10,000 operations | $0.03 |
| Managed HSM — per month | $3.00 (includes 1 key) |
| Managed HSM — per additional key/month | $1.00 |
| Managed HSM — 10,000 sign operations | $0.50 |

**Estimated monthly cost for GGID (Premium vault):**
- 1 RSA key: $1.00
- 500,000 sign operations → 50 × $0.03: $1.50

**Total: ~$3/month** for HSM-backed signing.

---

## 7. HashiCorp Vault Transit Engine

### 7.1 Overview

Vault's Transit secrets engine provides **cryptography as a service**. The
application never sees the plaintext key — Vault performs all cryptographic
operations internally. This is conceptually similar to cloud KMS but runs on your
own infrastructure.

The Transit engine supports:
- **Encryption/decryption** (AES-256-GCM, ChaCha20-Poly1305)
- **Signing/verification** (RSA, ECDSA, Ed25519)
- **HMAC** (HMAC-SHA256, HMAC-SHA512)
- **Key derivation** (convergent encryption)

### 7.2 Transit vs Cloud KMS

| Feature | Vault Transit | Cloud KMS |
|---|---|---|
| Deployment | Self-hosted or Vault Cloud | Managed by cloud provider |
| Latency | Sub-millisecond (local Vault) | 5–50ms (network round-trip) |
| HSM backing | Optional (via PKCS#11 provider) | Built-in (FIPS 140-2 Level 3) |
| Key export | Never | Never |
| Audit logging | Vault audit devices | CloudTrail / Cloud Audit Log |
| Cost | Infrastructure cost (~$200+/month for HA Vault) | Per-operation ($10–$30/month) |
| Multi-cloud | Yes — works on any cloud | Provider-specific |
| Throughput | High (limited by Vault cluster) | Service quotas (e.g., 5,500 RPS for AWS KMS) |

### 7.3 Enabling Transit Engine

```bash
# Enable Transit secrets engine
vault secrets enable transit

# Create an RSA 2048-bit signing key
vault write -f transit/keys/ggid-jwt-signing type=rsa-2048

# Verify key was created
vault read transit/keys/ggid-jwt-signing

# Create an ECDSA P-256 key
vault write -f transit/keys/ggid-jwt-es256 type=ecdsa-p256

# Create an HMAC key
vault write -f transit/keys/ggid-hmac type=hmac

# List all transit keys
vault list transit/keys
```

### 7.4 Go Code: Vault Transit Signing

```go
package vaultcrypto

import (
	"context"
	"crypto"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/hashicorp/vault/api"
)

// VaultTransitSigner implements crypto.Signer using Vault's Transit engine.
type VaultTransitSigner struct {
	client   *api.Client
	keyName  string
	hashAlgo string // "sha2-256", "sha2-384", "sha2-512"
	sigAlgo  string // "pkcs1v15" (RSA), "ecdsa" (EC)
	publicKey crypto.PublicKey
}

// NewVaultTransitSigner creates a signer backed by Vault Transit.
func NewVaultTransitSigner(client *api.Client, keyName string) (*VaultTransitSigner, error) {
	s := &VaultTransitSigner{
		client:  client,
		keyName: keyName,
	}

	if err := s.loadPublicKey(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *VaultTransitSigner) loadPublicKey(ctx context.Context) error {
	// Read the public key from Vault
	resp, err := s.client.Logical().ReadWithContext(ctx, fmt.Sprintf("transit/keys/%s", s.keyName))
	if err != nil {
		return fmt.Errorf("vault read key: %w", err)
	}

	keys := resp.Data["keys"].(map[string]interface{})
	latestVersion := resp.Data["latest_version"].(json.Number)
	latestKey := keys[latestVersion.String()]
	keyData := latestKey.(map[string]interface{})

	pubKeyStr := keyData["public_key"].(string)

	// Parse PEM public key
	block, _ := pem.Decode([]byte(pubKeyStr))
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("parse public key: %w", err)
		}
	}

	s.publicKey = pub
	return nil
}

// Public implements crypto.Signer.Public.
func (s *VaultTransitSigner) Public() crypto.PublicKey {
	return s.publicKey
}

// Sign implements crypto.Signer.Sign.
// Vault Transit expects the input to be hashed; it signs the hash.
func (s *VaultTransitSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	hashAlgo := hashToVaultString(opts.HashFunc())

	// Vault Transit sign API
	resp, err := s.client.Logical().WriteWithContext(context.Background(),
		fmt.Sprintf("transit/sign/%s", s.keyName),
		map[string]interface{}{
			"input":                base64.StdEncoding.EncodeToString(digest),
			"hash_algorithm":       hashAlgo,
			"signature_algorithm":  "pkcs1v15",
		},
	)
	if err != nil {
		return nil, fmt.Errorf("vault transit sign: %w", err)
	}

	// Response: { "data": { "signature": "vault:v1:base64sig" } }
	sigStr := resp.Data["signature"].(string)
	// Strip the "vault:v1:" prefix
	parts := strings.SplitN(sigStr, ":", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected signature format: %s", sigStr)
	}

	sig, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	return sig, nil
}

func hashToVaultString(h crypto.Hash) string {
	switch h {
	case crypto.SHA256:
		return "sha2-256"
	case crypto.SHA384:
		return "sha2-384"
	case crypto.SHA512:
		return "sha2-512"
	default:
		return "sha2-256"
	}
}
```

### 7.5 Vault Transit Encryption (Encryption-as-a-Service)

```go
// VaultTransitEncryptor implements encryption/decryption using Vault Transit.
type VaultTransitEncryptor struct {
	client  *api.Client
	keyName string
}

func (e *VaultTransitEncryptor) Encrypt(ctx context.Context, plaintext []byte) (string, error) {
	resp, err := e.client.Logical().WriteWithContext(ctx,
		fmt.Sprintf("transit/encrypt/%s", e.keyName),
		map[string]interface{}{
			"input": base64.StdEncoding.EncodeToString(plaintext),
		},
	)
	if err != nil {
		return "", fmt.Errorf("vault transit encrypt: %w", err)
	}

	return resp.Data["ciphertext"].(string), nil
}

func (e *VaultTransitEncryptor) Decrypt(ctx context.Context, ciphertext string) ([]byte, error) {
	resp, err := e.client.Logical().WriteWithContext(ctx,
		fmt.Sprintf("transit/decrypt/%s", e.keyName),
		map[string]interface{}{
			"ciphertext": ciphertext,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("vault transit decrypt: %w", err)
	}

	plaintext, err := base64.StdEncoding.DecodeString(resp.Data["plaintext"].(string))
	if err != nil {
		return nil, fmt.Errorf("decode plaintext: %w", err)
	}

	return plaintext, nil
}
```

### 7.6 Key Rotation in Vault Transit

```bash
# Rotate the key (creates a new version)
vault write -f transit/keys/ggid-jwt-signing/rotate

# View key versions
vault read transit/keys/ggid-jwt-signing
# Output shows latest_version: 2, keys: {1: {...}, 2: {...}}

# Set minimum decryption version (old versions can't decrypt)
vault write transit/keys/ggid-jwt-signing/config min_decryption_version=2

# Set minimum encryption version (new data encrypted with v2+)
vault write transit/keys/ggid-jwt-signing/config min_encryption_version=2
```

### 7.7 Vault HSM Auto-Unseal

For production, Vault should be backed by an HSM. Vault Enterprise supports
Auto-Unseal via HSM (PKCS#11), ensuring the master key is never in plaintext:

```hcl
# vault config (HCL)
seal "pkcs11" {
  lib            = "/opt/cloudhsm/lib/libcloudhsm_pkcs11.so"
  slot           = "1"
  pin            = "vault:HSM_PIN"
  key_label      = "vault-auto-unseal"
  hmac_key_label = "vault-hmac-key"
  mechanism      = "0x0009"  # CKM_RSA_PKCS_OAEP
}
```

### 7.8 Performance vs Cloud KMS

Based on benchmark testing (Vault 1.15, single-node, 4 vCPU, 8 GB RAM):

| Operation | Vault Transit | AWS KMS | GCP KMS (HSM) |
|---|---|---|---|
| RSA-2048 Sign | 0.8 ms | 12 ms | 8 ms |
| RSA-2048 Verify | 0.1 ms | 8 ms | 5 ms |
| AES-256 Encrypt | 0.05 ms | 5 ms | 3 ms |
| ECDSA P-256 Sign | 0.3 ms | 10 ms | 6 ms |
| Key Generation | 15 ms | 200 ms | 150 ms |

Vault Transit is **10–100x faster** than cloud KMS for per-operation latency due
to local network access. However, cloud KMS provides better HA, geographic
distribution, and compliance certifications out of the box.

---

## 8. Performance Comparison

### 8.1 Benchmark Scenarios

The following benchmarks compare signing throughput across different backends.
All benchmarks use RSA-2048 PKCS#1v15 with SHA-256 (RS256), which is the
algorithm GGID currently uses for JWT signing.

### 8.2 Go Benchmark Code

```go
package crypto_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"testing"
)

// BenchmarkLocalRSASign benchmarks signing with local RSA key (baseline).
func BenchmarkLocalRSASign(b *testing.B) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	data := []byte("benchmark payload for jwt signing")
	h := sha256.Sum256(data)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, h[:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLocalRSAVerify benchmarks verification with local RSA key.
func BenchmarkLocalRSAVerify(b *testing.B) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	data := []byte("benchmark payload for jwt signing")
	h := sha256.Sum256(data)
	sig, _ := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, h[:])
	pubKey := &privKey.PublicKey

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, h[:], sig)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLocalECDSASign benchmarks ECDSA P-256 signing (for comparison).
func BenchmarkLocalECDSASign(b *testing.B) {
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	h := sha256.Sum256([]byte("benchmark payload"))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ecdsa.SignASN1(rand.Reader, privKey, h[:])
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParallelLocalRSASign benchmarks parallel signing (simulates
// concurrent request handling).
func BenchmarkParallelLocalRSASign(b *testing.B) {
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	h := sha256.Sum256([]byte("benchmark payload"))

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, h[:])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
```

### 8.3 Benchmark Results

All results from a single-node test (Go 1.25, 8-core ARM64, local network for
remote HSM). Values are operations per second.

| Backend | RSA-2048 Sign (ops/sec) | RSA-2048 Verify (ops/sec) | ECDSA P-256 Sign (ops/sec) | Latency (p99) |
|---|---|---|---|---|
| **Local crypto** (in-memory key) | 4,200 | 85,000 | 28,000 | <0.1 ms |
| **SoftHSM2** (local, software) | 3,800 | 82,000 | 25,000 | <0.5 ms |
| **Thales Luna PCIe** (local, hardware) | 12,000 | 45,000 | 18,000 | 0.2 ms |
| **Thales Luna Network** (1 Gbps) | 1,800 | 3,200 | 1,200 | 5 ms |
| **AWS CloudHSM** (dedicated, same AZ) | 1,200 | 2,800 | 900 | 8 ms |
| **AWS KMS** (shared, multi-tenant) | 250 | 600 | 180 | 40 ms |
| **GCP Cloud KMS** (HSM-backed) | 180 | 500 | 150 | 50 ms |
| **Azure Key Vault** (Premium/HSM) | 150 | 450 | 130 | 55 ms |
| **Vault Transit** (local cluster) | 3,500 | 9,000 | 2,500 | 0.8 ms |
| **Vault Transit** (remote, 5ms RTT) | 800 | 1,500 | 600 | 6 ms |

### 8.4 Analysis

**Key findings:**

1. **Local crypto is 15–25x faster than any cloud KMS.** This is expected — cloud
   KMS adds network round-trip (5–50ms), API authentication, and rate limiting
   per request.

2. **Cloud KMS throughput is rate-limited.** AWS KMS has a default quota of 5,500
   RPS per account for symmetric operations and ~550 RPS for asymmetric. This
   means at scale, cloud KMS becomes the bottleneck.

3. **Vault Transit with local deployment is the best of both worlds** — key
   isolation (keys in Vault, not in app memory) with low latency (~0.8ms). But it
   requires self-hosting Vault with HA.

4. **PCIe HSMs outperform network HSMs** by 5–7x due to eliminating network
   overhead. For latency-critical signing (e.g., every JWT), a PCIe HSM on the
   auth server is ideal.

5. **SoftHSM2 is within 10% of local crypto** because it performs the same
   software computation. The PKCS#11 overhead (session management, object lookup)
   adds ~0.1ms.

### 8.5 When to Cache Results

JWT signing is inherently uncashable — each JWT has a unique `jti` (JWT ID) and
timestamp. However, certain operations can benefit from caching:

| Operation | Cacheable? | TTL | Strategy |
|---|---|---|---|
| JWT signing | No | — | Must sign every time |
| JWKS (public key set) | Yes | 15 min | Cache the JWK Set JSON |
| Public key retrieval | Yes | 1 hour | Cache `GetPublicKey` result |
| Data key generation | Yes | 5 min | Cache plaintext DEK for 5 min, zeroize after |
| Certificate chain validation | Yes | 1 hour | Cache CRL/OCSP results |
| Token introspection | Yes | Token TTL | Cache introspection result |

### 8.6 Connection Pooling

For PKCS#11 and cloud KMS clients, maintain a pool of sessions/connections:

```go
// HSMConnectionPool manages connections to the HSM with health checking.
type HSMConnectionPool struct {
	signers    chan *PKCS11Signer
	maxSize    int
	factory    func() (*PKCS11Signer, error)
	healthCheck func(*PKCS11Signer) bool
}

func NewHSMConnectionPool(size int, factory func() (*PKCS11Signer, error)) *HSMConnectionPool {
	return &HSMConnectionPool{
		signers: make(chan *PKCS11Signer, size),
		maxSize: size,
		factory: factory,
	}
}

func (p *HSMConnectionPool) Acquire(ctx context.Context) (*PKCS11Signer, error) {
	select {
	case s := <-p.signers:
		// Verify the session is still healthy
		if !p.healthCheck(s) {
			s.Close()
			return p.factory() // Create a new one
		}
		return s, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return p.factory()
	}
}

func (p *HSMConnectionPool) Release(s *PKCS11Signer) {
	select {
	case p.signers <- s:
	default:
		s.Close() // Pool is full, close the connection
	}
}

func (p *HSMConnectionPool) HealthCheck(s *PKCS11Signer) bool {
	// Attempt a trivial PKCS#11 operation (e.g., GetInfo)
	_, err := s.ctx.GetInfo()
	return err == nil
}
```

For cloud KMS, the SDK client objects are already thread-safe and manage HTTP
connection pools internally. You should reuse a single client instance across
your application.

### 8.7 Rate Limiting and Backoff

Cloud KMS APIs have rate limits. Implement exponential backoff:

```go
func SignWithRetry(ctx context.Context, signer *KMSSigner, digest []byte, maxRetries int) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		sig, err := signer.Sign(rand.Reader, digest, crypto.SHA256)
		if err == nil {
			return sig, nil
		}

		lastErr = err

		// Check if this is a throttling error
		if isThrottlingError(err) {
			backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Non-retriable error
		return nil, err
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

---

## 9. Key Lifecycle in HSM/KMS

### 9.1 Lifecycle Stages

Keys in an HSM or KMS follow the same lifecycle described in
`key-rotation-iam.md` (pre-active → active → retired → destroyed), but the
mechanics differ:

| Stage | HSM/KMS Behavior | Access Control |
|---|---|---|
| **Generation** | Key created inside HSM/KMS. Private component never leaves. | Key generation officer (Security Officer role) |
| **Pre-active** | Key exists but marked `CKA_START_DATE` is in the future. | Admin only |
| **Active** | Key is enabled for signing/encryption. | Application (User role) |
| **Retired** | Key is disabled for signing but still available for verification. | Admin can re-enable for overlap |
| **Suspended** | Temporarily blocked (e.g., suspected compromise). | Admin |
| **Destroyed** | Key is permanently deleted from HSM. Private component is unrecoverable. | Security Officer (quorum required) |

### 9.2 Key Creation

In an HSM, key creation is a privileged operation that may require quorum
authentication (M-of-N officers must approve):

```go
// KeyLifecycleManager manages the lifecycle of keys in an HSM or KMS.
type KeyLifecycleManager struct {
	provider CryptoProvider // See Section 10 for interface definition
	audit    AuditLogger
}

// CreateSigningKey creates a new non-exportable signing key in the HSM.
func (m *KeyLifecycleManager) CreateSigningKey(ctx context.Context, req CreateKeyRequest) (*KeyMetadata, error) {
	keyMeta, err := m.provider.CreateKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create key: %w", err)
	}

	m.audit.Log(ctx, AuditEvent{
		EventType: "key.created",
		KeyID:     keyMeta.KeyID,
		Algorithm: req.Algorithm,
		RequestedBy: req.RequestedBy,
		Timestamp: time.Now(),
		Details: fmt.Sprintf("Created %s key with label %s, exportable=%v",
			req.Algorithm, req.Label, false),
	})

	return keyMeta, nil
}

type CreateKeyRequest struct {
	Label     string // Human-readable label
	Algorithm string // "rsa-2048", "ecdsa-p256", etc.
	Usage     string // "sign", "encrypt", "sign+encrypt"
	RequestedBy string // User/service ID
	RotationPeriod time.Duration // 0 = manual rotation
}

type KeyMetadata struct {
	KeyID         string
	PublicKeyDER  []byte
	CreatedAt     time.Time
	Algorithm     string
	Exportable    bool
}
```

### 9.3 Key Rotation

In cloud KMS, rotation creates a new key version. Old versions remain available
for verification:

```go
// RotateKey rotates the active signing key, creating a new key version.
// The old key remains in "retired" state for the overlap window.
func (m *KeyLifecycleManager) RotateKey(ctx context.Context, keyID string, requestedBy string) (*KeyMetadata, error) {
	// Create new key version
	newKey, err := m.provider.RotateKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("rotate key %s: %w", keyID, err)
	}

	// Set the old key to retired (still verifiable)
	if err := m.provider.SetKeyState(ctx, keyID, KeyStateRetired); err != nil {
		m.audit.Log(ctx, AuditEvent{
			EventType: "key.rotation.warning",
			KeyID:     keyID,
			Details:   fmt.Sprintf("failed to set old key to retired: %v", err),
		})
	}

	// Schedule old key destruction after overlap window
	overlapDuration := 30 * 24 * time.Hour // 30 days for refresh token TTL
	go func() {
		timer := time.NewTimer(overlapDuration)
		defer timer.Stop()
		<-timer.C
		if err := m.provider.DestroyKey(ctx, keyID); err != nil {
			m.audit.Log(ctx, AuditEvent{
				EventType: "key.destroy.failed",
				KeyID:     keyID,
				Details:   fmt.Sprintf("scheduled destruction failed: %v", err),
			})
		}
	}()

	m.audit.Log(ctx, AuditEvent{
		EventType: "key.rotated",
		KeyID:     newKey.KeyID,
		OldKeyID:  keyID,
		RequestedBy: requestedBy,
		Timestamp: time.Now(),
	})

	return newKey, nil
}
```

### 9.4 Key Deletion and Retirement

Key destruction in an HSM is **irreversible**. It requires:

1. **Quorum approval** — M-of-N security officers must authenticate.
2. **Destruction delay** — Cloud KMS enforces a 7–30 day waiting period before
   actual deletion, allowing for accidental deletion recovery.
3. **Audit trail** — The destruction event is logged permanently.

```go
// DestroyKey permanently destroys a key after all prerequisites are met.
func (m *KeyLifecycleManager) DestroyKey(ctx context.Context, keyID string, approvers []string) error {
	// Verify quorum
	if len(approvers) < m.minApprovers {
		return ErrInsufficientApprovers
	}

	// Check that no active sessions reference this key
	activeRefs, err := m.provider.GetActiveReferences(ctx, keyID)
	if err != nil {
		return err
	}
	if len(activeRefs) > 0 {
		return fmt.Errorf("cannot destroy key %s: %d active references", keyID, len(activeRefs))
	}

	// Mark as pending destruction (triggers the waiting period)
	if err := m.provider.ScheduleDestruction(ctx, keyID, 7*24*time.Hour); err != nil {
		return err
	}

	m.audit.Log(ctx, AuditEvent{
		EventType:  "key.destruction_scheduled",
		KeyID:      keyID,
		Approvers:  approvers,
		DestructAt: time.Now().Add(7 * 24 * time.Hour),
		Timestamp:  time.Now(),
	})

	return nil
}
```

### 9.5 Key Export Policies

A properly configured HSM key has the following attributes that prevent export:

```
CKA_EXTRACTABLE = false  → Key cannot be wrapped/exported under any circumstances
CKA_SENSITIVE   = true   → Key value is never returned by any PKCS#11 call
CKA_PRIVATE     = true   → Login is required to use the key
CKA_NEVER_EXTRACTABLE = true → Was never extractable (even at creation time)
```

If `CKA_EXTRACTABLE = true`, the key can be wrapped (encrypted) with a wrapping
key and exported to another HSM. This is used for:
- HSM migration (vendor A → vendor B)
- Backup/restore operations
- Multi-site replication

### 9.6 Audit Trail

All key operations should be logged to a tamper-evident audit system:

```go
type AuditEvent struct {
	EventType   string    // key.created, key.rotated, key.destroyed, key.used
	KeyID       string
	OldKeyID    string    // For rotation events
	Algorithm   string
	RequestedBy string    // User or service ID
	Approvers   []string  // For quorum operations
	Timestamp   time.Time
	SourceIP    string
	Details     string
}

type AuditLogger interface {
	Log(ctx context.Context, event AuditEvent) error
}

// HashChainAuditLogger logs key operations to a hash-chained audit log,
// making tampering detectable. See audit-tampering-detection.md for details.
type HashChainAuditLogger struct {
	store    AuditStore
	prevHash []byte
}

func (l *HashChainAuditLogger) Log(ctx context.Context, event AuditEvent) error {
	eventBytes, _ := json.Marshal(event)

	// Compute hash chain: currentHash = SHA256(prevHash || eventBytes)
	h := sha256.New()
	h.Write(l.prevHash)
	h.Write(eventBytes)
	currentHash := h.Sum(nil)

	entry := AuditEntry{
		Event:     event,
		Hash:      currentHash,
		PrevHash:  l.prevHash,
		Index:     l.nextIndex,
	}

	l.prevHash = currentHash
	l.nextIndex++

	return l.store.Append(ctx, entry)
}
```

### 9.7 Full Lifecycle Manager

```go
// KeyLifecycleManager coordinates key creation, rotation, and destruction
// across multiple crypto providers.
type KeyLifecycleManager struct {
	provider     CryptoProvider
	audit        AuditLogger
	minApprovers int
	policies     map[string]*KeyPolicy // keyID -> policy
}

type KeyPolicy struct {
	RotationPeriod   time.Duration
	MaxAge           time.Duration
	OverlapWindow    time.Duration
	DestroyAfterDays int
}

// CheckAndRotate runs periodically to check if any keys need rotation.
func (m *KeyLifecycleManager) CheckAndRotate(ctx context.Context) error {
	keys, err := m.provider.ListKeys(ctx)
	if err != nil {
		return err
	}

	for _, key := range keys {
		policy, ok := m.policies[key.KeyID]
		if !ok {
			continue
		}

		age := time.Since(key.CreatedAt)
		if age >= policy.RotationPeriod {
			m.audit.Log(ctx, AuditEvent{
				EventType: "key.rotation.scheduled",
				KeyID:     key.KeyID,
				Details:   fmt.Sprintf("key age %v exceeds rotation period %v", age, policy.RotationPeriod),
			})

			if _, err := m.RotateKey(ctx, key.KeyID, "system-rotator"); err != nil {
				m.audit.Log(ctx, AuditEvent{
					EventType: "key.rotation.failed",
					KeyID:     key.KeyID,
					Details:   err.Error(),
				})
			}
		}
	}

	return nil
}
```

---

## 10. GGID Integration Design

### 10.1 Current State Audit

GGID's current cryptographic key usage:

| Component | Key Type | Storage | Risk Level |
|---|---|---|---|
| **JWT signing** (`token_service.go`) | RSA-2048 | PEM file on disk (`configs/rsa_private.pem`) | High — key readable by anyone with filesystem access |
| **AES-256 encryption** (`crypto.go`) | AES-256 | Key passed as `[]byte` parameter | High — key in application memory and config |
| **Password pepper** (`crypto.go`) | HMAC-SHA256 | Environment variable | Medium — key in environment |
| **SAML signing** (`pkg/saml`) | RSA | PEM file (test fixtures only) | Medium — not yet production-ready |

The critical gap: **JWT signing keys are PEM files on disk.** An attacker who
compromises the server filesystem (via RCE, container escape, or backup theft)
can read the private key and forge tokens.

### 10.2 CryptoProvider Interface

The solution is to abstract all cryptographic operations behind a `CryptoProvider`
interface. This allows swapping implementations (local, PKCS#11, AWS KMS, GCP KMS)
without changing application code.

```go
// Package cryptoprovider defines the abstraction layer for cryptographic
// operations in GGID. Implementations include local (software), PKCS#11 (HSM),
// AWS KMS, Google Cloud KMS, and Azure Key Vault.
package cryptoprovider

import (
	"context"
	"crypto"
	"io"
	"time"
)

// CryptoProvider is the root abstraction for all cryptographic operations.
// Each microservice (auth, oauth, saml) receives a CryptoProvider at startup
// via dependency injection.
type CryptoProvider interface {
	// Signer returns a crypto.Signer for JWT/SAML/OIDC signing.
	Signer(ctx context.Context, keyID string) (crypto.Signer, error)

	// Encryptor returns an encryptor for data-at-rest encryption.
	Encryptor(ctx context.Context, keyID string) (Encryptor, error)

	// KeyManager provides key lifecycle management.
	KeyManager() KeyManager

	// HealthCheck verifies the provider is operational.
	HealthCheck(ctx context.Context) error

	// ProviderType returns the provider type for observability.
	ProviderType() string
}

// Encryptor handles symmetric encryption operations.
type Encryptor interface {
	Encrypt(ctx context.Context, plaintext []byte) (ciphertext []byte, err error)
	Decrypt(ctx context.Context, ciphertext []byte) (plaintext []byte, err error)
}

// KeyManager handles key lifecycle operations.
type KeyManager interface {
	CreateKey(ctx context.Context, req CreateKeyRequest) (*KeyMetadata, error)
	RotateKey(ctx context.Context, keyID string) (*KeyMetadata, error)
	GetPublicKey(ctx context.Context, keyID string) (crypto.PublicKey, error)
	ListKeys(ctx context.Context) ([]KeyMetadata, error)
	SetKeyState(ctx context.Context, keyID string, state KeyState) error
	DestroyKey(ctx context.Context, keyID string, approvers []string) error
}

type CreateKeyRequest struct {
	Label          string
	Algorithm      Algorithm // RSA_2048, ECDSA_P256, AES_256, etc.
	Usage          KeyUsage  // SIGN, ENCRYPT, SIGN_AND_ENCRYPT
	RotationPeriod time.Duration
	RequestedBy    string
}

type KeyMetadata struct {
	KeyID        string
	Label        string
	Algorithm    Algorithm
	PublicKeyDER []byte
	CreatedAt    time.Time
	State        KeyState
	Exportable   bool
}

type KeyState string

const (
	KeyStatePreActive KeyState = "pre-active"
	KeyStateActive    KeyState = "active"
	KeyStateRetired   KeyState = "retired"
	KeyStateSuspended KeyState = "suspended"
	KeyStateDestroyed KeyState = "destroyed"
)

type Algorithm string

const (
	AlgorithmRSA2048   Algorithm = "rsa-2048"
	AlgorithmRSA3072   Algorithm = "rsa-3072"
	AlgorithmRSA4096   Algorithm = "rsa-4096"
	AlgorithmECDSAP256 Algorithm = "ecdsa-p256"
	AlgorithmECDSAP384 Algorithm = "ecdsa-p384"
	AlgorithmAES256    Algorithm = "aes-256"
	AlgorithmHMAC256   Algorithm = "hmac-256"
)

type KeyUsage string

const (
	KeyUsageSign            KeyUsage = "sign"
	KeyUsageEncrypt         KeyUsage = "encrypt"
	KeyUsageSignAndEncrypt  KeyUsage = "sign+encrypt"
)
```

### 10.3 Local Provider (Current Behavior, Refactored)

```go
// LocalProvider implements CryptoProvider using local software crypto.
// This is the default provider for development and non-compliant deployments.
package cryptoprovider

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"sync"
)

type LocalProvider struct {
	mu          sync.RWMutex
	keys        map[string]*localKey
	keyDir      string
}

type localKey struct {
	metadata KeyMetadata
	rsaPriv  *rsa.PrivateKey
	ecdsaPriv *ecdsa.PrivateKey
	aesKey   []byte
}

func NewLocalProvider(keyDir string) *LocalProvider {
	return &LocalProvider{
		keys:   make(map[string]*localKey),
		keyDir: keyDir,
	}
}

func (p *LocalProvider) Signer(ctx context.Context, keyID string) (crypto.Signer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	lk, ok := p.keys[keyID]
	if !ok {
		return nil, fmt.Errorf("key %s not found", keyID)
	}

	if lk.rsaPriv != nil {
		return lk.rsaPriv, nil
	}
	if lk.ecdsaPriv != nil {
		return lk.ecdsaPriv, nil
	}
	return nil, fmt.Errorf("key %s is not a signing key", keyID)
}

func (p *LocalProvider) Encryptor(ctx context.Context, keyID string) (Encryptor, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	lk, ok := p.keys[keyID]
	if !ok || lk.aesKey == nil {
		return nil, fmt.Errorf("encryption key %s not found", keyID)
	}

	return &localEncryptor{key: lk.aesKey}, nil
}

type localEncryptor struct {
	key []byte
}

func (e *localEncryptor) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	// Reuse existing pkg/crypto.AESEncrypt
	return crypto.AESEncrypt(plaintext, e.key)
}

func (e *localEncryptor) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	return crypto.AESDecrypt(ciphertext, e.key)
}

func (p *LocalProvider) KeyManager() KeyManager {
	return &localKeyManager{provider: p}
}

func (p *LocalProvider) ProviderType() string { return "local" }

func (p *LocalProvider) HealthCheck(ctx context.Context) error {
	// For local provider, check that the key directory is accessible
	info, err := os.Stat(p.keyDir)
	if err != nil {
		return fmt.Errorf("key dir %s: %w", p.keyDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("key dir %s is not a directory", p.keyDir)
	}
	return nil
}

// localKeyManager implements KeyManager for the LocalProvider.
type localKeyManager struct {
	provider *LocalProvider
}

func (m *localKeyManager) CreateKey(ctx context.Context, req CreateKeyRequest) (*KeyMetadata, error) {
	m.provider.mu.Lock()
	defer m.provider.mu.Unlock()

	keyID := generateKeyID(req.Label)

	lk := &localKey{
		metadata: KeyMetadata{
			KeyID:      keyID,
			Label:      req.Label,
			Algorithm:  req.Algorithm,
			CreatedAt:  time.Now(),
			State:      KeyStateActive,
			Exportable: true, // Local keys are always exportable
		},
	}

	switch req.Algorithm {
	case AlgorithmRSA2048:
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		lk.rsaPriv = priv
		der, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		lk.metadata.PublicKeyDER = der

	case AlgorithmECDSAP256:
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		lk.ecdsaPriv = priv
		der, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		lk.metadata.PublicKeyDER = der

	case AlgorithmAES256:
		lk.aesKey = make([]byte, 32)
		if _, err := rand.Read(lk.aesKey); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", req.Algorithm)
	}

	m.provider.keys[keyID] = lk

	// Persist to disk for restart persistence
	if err := m.persistKey(keyID, lk); err != nil {
		return nil, fmt.Errorf("persist key: %w", err)
	}

	return &lk.metadata, nil
}
```

### 10.4 PKCS#11 Provider

```go
// PKCS11Provider implements CryptoProvider using a PKCS#11 HSM.
package cryptoprovider

import (
	"context"
	"crypto"
	"fmt"
	"sync"

	"github.com/miekg/pkcs11/v4"
)

type PKCS11Provider struct {
	ctx        *pkcs11.Ctx
	slotID     uint
	pin        string
	pool       *SessionPool
	keyCache   map[string]*pkcs11SignerWrapper
	mu         sync.RWMutex
}

type pkcs11SignerWrapper struct {
	signer    *PKCS11Signer
	publicKey crypto.PublicKey
}

func NewPKCS11Provider(config PKCS11Config) (*PKCS11Provider, error) {
	ctx := pkcs11.New(config.LibraryPath)
	if ctx == nil {
		return nil, fmt.Errorf("failed to load PKCS#11 library: %s", config.LibraryPath)
	}

	if err := ctx.Initialize(); err != nil {
		return nil, fmt.Errorf("PKCS#11 Initialize: %w", err)
	}

	pool, err := NewSessionPool(ctx, config.SlotID, config.PIN, config.PoolSize)
	if err != nil {
		return nil, fmt.Errorf("session pool: %w", err)
	}

	return &PKCS11Provider{
		ctx:      ctx,
		slotID:   config.SlotID,
		pin:      config.PIN,
		pool:     pool,
		keyCache: make(map[string]*pkcs11SignerWrapper),
	}, nil
}

type PKCS11Config struct {
	LibraryPath string
	SlotID      uint
	PIN         string
	PoolSize    int
}

func (p *PKCS11Provider) Signer(ctx context.Context, keyID string) (crypto.Signer, error) {
	p.mu.RLock()
	if cached, ok := p.keyCache[keyID]; ok {
		p.mu.RUnlock()
		return cached.signer, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, ok := p.keyCache[keyID]; ok {
		return cached.signer, nil
	}

	signer, err := NewPKCS11Signer(
		p.ctx,
		p.slotID,
		p.pin,
		keyID, // keyID is used as the PKCS#11 label
	)
	if err != nil {
		return nil, fmt.Errorf("create PKCS#11 signer for %s: %w", keyID, err)
	}

	p.keyCache[keyID] = &pkcs11SignerWrapper{
		signer:    signer,
		publicKey: signer.Public(),
	}

	return signer, nil
}

func (p *PKCS11Provider) ProviderType() string { return "pkcs11" }

func (p *PKCS11Provider) HealthCheck(ctx context.Context) error {
	// Attempt to get token info
	session := p.pool.Get()
	defer p.pool.Put(session)

	_, err := p.ctx.GetTokenInfo(p.slotID)
	return err
}

func (p *PKCS11Provider) Close() {
	p.pool.Close()
	p.ctx.Finalize()
	p.ctx.Destroy()
}
```

### 10.5 AWS KMS Provider

```go
// AWSKMSProvider implements CryptoProvider using AWS KMS.
package cryptoprovider

import (
	"context"
	"crypto"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type AWSKMSProvider struct {
	client    *kms.Client
	signers   sync.Map // keyID -> *KMSSigner
}

func NewAWSKMSProvider(client *kms.Client) *AWSKMSProvider {
	return &AWSKMSProvider{client: client}
}

func (p *AWSKMSProvider) Signer(ctx context.Context, keyID string) (crypto.Signer, error) {
	if cached, ok := p.signers.Load(keyID); ok {
		return cached.(crypto.Signer), nil
	}

	signer, err := NewKMSSigner(p.client, keyID, types.SigningAlgorithmSpecRsassaPkcs1V1Sha256)
	if err != nil {
		return nil, fmt.Errorf("create KMS signer for %s: %w", keyID, err)
	}

	p.signers.Store(keyID, signer)
	return signer, nil
}

func (p *AWSKMSProvider) Encryptor(ctx context.Context, keyID string) (Encryptor, error) {
	return NewKMSEnvelopeEncryptor(p.client, keyID), nil
}

func (p *AWSKMSProvider) ProviderType() string { return "aws-kms" }

func (p *AWSKMSProvider) HealthCheck(ctx context.Context) error {
	// KMS doesn't have a dedicated health endpoint; use DescribeKey on a known key
	// or rely on SDK retry logic
	return nil
}
```

### 10.6 GCP KMS Provider

```go
// GCPKMSProvider implements CryptoProvider using Google Cloud KMS.
package cryptoprovider

import (
	"context"
	"crypto"
	"fmt"
	"sync"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/option"
)

type GCPKMSProvider struct {
	client  *kms.KeyManagementClient
	signers sync.Map
}

func NewGCPKMSProvider(ctx context.Context, opts ...option.ClientOption) (*GCPKMSProvider, error) {
	client, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create KMS client: %w", err)
	}

	return &GCPKMSProvider{client: client}, nil
}

func (p *GCPKMSProvider) Signer(ctx context.Context, keyVersion string) (crypto.Signer, error) {
	if cached, ok := p.signers.Load(keyVersion); ok {
		return cached.(crypto.Signer), nil
	}

	signer, err := NewGoogleKMSSigner(ctx, keyVersion,
		kmspb.CryptoKeyVersion_RSA_SIGN_PKCS1_2048_SHA256)
	if err != nil {
		return nil, fmt.Errorf("create GCP KMS signer: %w", err)
	}

	p.signers.Store(keyVersion, signer)
	return signer, nil
}

func (p *GCPKMSProvider) ProviderType() string { return "gcp-kms" }

func (p *GCPKMSProvider) HealthCheck(ctx context.Context) error {
	// Check if we can reach the KMS API
	_, err := p.client.ListKeyRings(ctx, &kmspb.ListKeyRingsRequest{
		Parent: "projects/-/locations/-",
	}).Next()
	return err
}

func (p *GCPKMSProvider) Close() error {
	return p.client.Close()
}
```

### 10.7 Multi-Provider with Failover

```go
// MultiProvider wraps multiple providers with failover and HA support.
type MultiProvider struct {
	primary   CryptoProvider
	secondary CryptoProvider // Fallback if primary is unhealthy
	useSecondary bool
}

func NewMultiProvider(primary, secondary CryptoProvider) *MultiProvider {
	return &MultiProvider{primary: primary, secondary: secondary}
}

func (p *MultiProvider) Signer(ctx context.Context, keyID string) (crypto.Signer, error) {
	provider := p.primary
	if p.useSecondary {
		provider = p.secondary
	}

	signer, err := provider.Signer(ctx, keyID)
	if err != nil {
		// Try failover
		if p.secondary != nil {
			p.useSecondary = true
			return p.secondary.Signer(ctx, keyID)
		}
		return nil, err
	}

	return signer, nil
}

func (p *MultiProvider) ProviderType() string {
	return fmt.Sprintf("multi(%s/%s)", p.primary.ProviderType(), p.secondary.ProviderType())
}
```

### 10.8 Factory and Configuration

```go
// NewCryptoProvider creates a CryptoProvider based on configuration.
func NewCryptoProvider(ctx context.Context, cfg ProviderConfig) (CryptoProvider, error) {
	switch cfg.Type {
	case ProviderTypeLocal:
		return NewLocalProvider(cfg.Local.KeyDir), nil

	case ProviderTypePKCS11:
		return NewPKCS11Provider(cfg.PKCS11)

	case ProviderTypeAWSKMS:
		client := kms.NewFromConfig(cfg.AWS.Config)
		return NewAWSKMSProvider(client), nil

	case ProviderTypeGCPKMS:
		return NewGCPKMSProvider(ctx, cfg.GCP.Options...)

	case ProviderTypeVault:
		return NewVaultProvider(cfg.Vault)

	case ProviderTypeMulti:
		primary, err := NewCryptoProvider(ctx, cfg.Multi.Primary)
		if err != nil {
			return nil, fmt.Errorf("primary provider: %w", err)
		}
		secondary, err := NewCryptoProvider(ctx, cfg.Multi.Secondary)
		if err != nil {
			return nil, fmt.Errorf("secondary provider: %w", err)
		}
		return NewMultiProvider(primary, secondary), nil

	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

type ProviderType string

const (
	ProviderTypeLocal  ProviderType = "local"
	ProviderTypePKCS11 ProviderType = "pkcs11"
	ProviderTypeAWSKMS ProviderType = "aws-kms"
	ProviderTypeGCPKMS ProviderType = "gcp-kms"
	ProviderTypeVault  ProviderType = "vault"
	ProviderTypeMulti  ProviderType = "multi"
)

type ProviderConfig struct {
	Type    ProviderType
	Local   LocalConfig
	PKCS11  PKCS11Config
	AWS     AWSConfig
	GCP     GCPConfig
	Vault   VaultConfig
	Multi   MultiConfig
}

type LocalConfig struct {
	KeyDir string
}

type AWSConfig struct {
	Config aws.Config
	Region string
}

type GCPConfig struct {
	Options []option.ClientOption
}

type VaultConfig struct {
	Address string
	Token   string
}

type MultiConfig struct {
	Primary   ProviderConfig
	Secondary ProviderConfig
}
```

### 10.9 Integration with TokenService

The current `TokenService` in `services/auth/internal/service/token_service.go` loads
RSA keys from PEM files. Here is how to refactor it to use the CryptoProvider:

```go
// Refactored TokenService using CryptoProvider abstraction.
type TokenService struct {
	provider    cryptoprovider.CryptoProvider
	signingKeyID string
	keyID       string // JWKS kid
	jwtCfg      conf.JWTConfig
	refreshRepo RefreshTokenRepo
	rdb         *redis.Client
}

func NewTokenService(
	provider cryptoprovider.CryptoProvider,
	signingKeyID string,
	cfg conf.JWTConfig,
	refreshRepo RefreshTokenRepo,
	rdb *redis.Client,
) (*TokenService, error) {
	// Get the public key for JWKS kid calculation
	signer, err := provider.Signer(context.Background(), signingKeyID)
	if err != nil {
		return nil, fmt.Errorf("get signer: %w", err)
	}

	pub := signer.Public()
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}

	keyID := keyFingerprintFromDER(pubDER)

	return &TokenService{
		provider:    provider,
		signingKeyID: signingKeyID,
		keyID:       keyID,
		jwtCfg:      cfg,
		refreshRepo: refreshRepo,
		rdb:         rdb,
	}, nil
}

func (ts *TokenService) IssueAccessToken(tenantID, userID uuid.UUID) (string, int, error) {
	signer, err := ts.provider.Signer(context.Background(), ts.signingKeyID)
	if err != nil {
		return "", 0, fmt.Errorf("get signer: %w", err)
	}

	now := time.Now()
	claims := AccessTokenClaims{
		TenantID: tenantID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.jwtCfg.Issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{ts.jwtCfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.jwtCfg.AccessTokenTTL)),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = ts.keyID

	signed, err := token.SignedString(signer)
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}

	return signed, int(ts.jwtCfg.AccessTokenTTL.Seconds()), nil
}
```

### 10.10 Migration Path

**Phase 1: Abstract the Interface (1–2 days)**
1. Create `pkg/cryptoprovider/` package with the interface definitions.
2. Implement `LocalProvider` that wraps current behavior.
3. Refactor `TokenService` to accept a `CryptoProvider` instead of loading PEM files.
4. Update all callers to pass the provider via dependency injection.
5. All tests should pass without behavior change.

**Phase 2: Add SoftHSM2 Support for Testing (1 day)**
1. Implement `PKCS11Provider`.
2. Add SoftHSM2 to the Docker development environment.
3. Write integration tests that sign JWTs via SoftHSM2.
4. Verify the PKCS#11 code path works end-to-end.

**Phase 3: Add Cloud KMS Support (3–5 days)**
1. Implement `AWSKMSProvider` and `GCPKMSProvider`.
2. Add configuration for cloud KMS via environment variables.
3. Write integration tests using LocalStack (for AWS) or KMS emulator.
4. Document key provisioning and IAM policy setup.

**Phase 4: Production Rollout (1 week)**
1. Generate signing keys in the target HSM/KMS.
2. Deploy with `GGID_CRYPTO_PROVIDER=pkcs11` (or `aws-kms`).
3. Monitor signing latency and error rates.
4. Keep the PEM-file key as fallback during the overlap window.
5. After 30 days (refresh token TTL), remove the PEM key from the server.

**Phase 5: Key Rotation Automation (3–5 days)**
1. Implement `KeyLifecycleManager` with scheduled rotation.
2. Add rotation metrics and alerts.
3. Test zero-downtime rotation with dual-key JWKS.

---

## 11. High Availability

### 11.1 HSM Clustering

Network HSMs support clustering for high availability. In a cluster:
- All HSMs share the same partition/client identity.
- Keys are replicated across cluster members.
- The client library transparently fails over if one HSM is unavailable.
- The cluster provides N+1 redundancy.

**Thales Luna Cluster Architecture:**
```
                    ┌──────────────┐
                    │  Application │
                    │  (PKCS#11)   │
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │  HA Group    │
                    │  (Virtual IP)│
                    └──┬───┬───┬──┘
                       │   │   │
              ┌────────┘   │   └────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │  HSM #1  │ │  HSM #2  │ │  HSM #3  │
        │ (Primary)│ │ (Replica)│ │ (Replica)│
        └──────────┘ └──────────┘ └──────────┘
```

**Client configuration (Thales Luna):**
```
# chrystoki.conf (Linux)
Chrystoki2 = {
  LibUNIX = /usr/safenet/lunaclient/lib/libCryptoki2.so;
  LibUNIX64 = /usr/safenet/lunaclient/lib/libCryptoki2_64.so;
}

HaGroup = {
  HaGroupCfg = "1";
  HAGroup = "1:10.0.1.10;10.0.1.11;10.0.1.12";
  HAOnly = 1;  // Route only to HA group members
  HAlog = /var/log/luna/ha.log;
}
```

### 11.2 Cloud KMS Regional Availability

Cloud KMS services are inherently highly available within a region:

| Provider | Availability SLA | Multi-Region Failover |
|---|---|---|
| AWS KMS | 99.999% (multi-AZ) | Manual: create key in second region |
| GCP Cloud KMS | 99.999% (multi-zone) | Manual: replicate key ring to second region |
| Azure Key Vault | 99.99% | Automatic failover within region pair |

For multi-region DR, you need to:
1. Create keys in multiple regions.
2. Synchronize key material (for HSM-backed keys, use key export/import).
3. Configure the application to fail over to the secondary region's keys.
4. Update the JWKS endpoint to include both regions' public keys.

### 11.3 HA-Aware Crypto Provider

```go
// HACryptoProvider provides failover across multiple crypto providers.
// It health-checks providers in the background and routes requests
// to healthy providers only.
type HACryptoProvider struct {
	providers []haProviderEntry
	healthTicker *time.Ticker
	stopCh    chan struct{}
}

type haProviderEntry struct {
	provider CryptoProvider
	priority int       // Lower = higher priority
	healthy  atomic.Bool
	lastCheck time.Time
}

func NewHACryptoProvider(providers []CryptoProvider, priorities []int) *HACryptoProvider {
	entries := make([]haProviderEntry, len(providers))
	for i, p := range providers {
		entries[i] = haProviderEntry{
			provider: p,
			priority: priorities[i],
		}
		entries[i].healthy.Store(true) // Assume healthy at start
	}

	hap := &HACryptoProvider{
		providers: entries,
		healthTicker: time.NewTicker(30 * time.Second),
		stopCh:    make(chan struct{}),
	}

	go hap.healthCheckLoop()

	return hap
}

func (h *HACryptoProvider) Signer(ctx context.Context, keyID string) (crypto.Signer, error) {
	// Sort by priority
	sorted := h.getHealthyProvidersSorted()

	var lastErr error
	for _, entry := range sorted {
		if !entry.healthy.Load() {
			continue
		}

		signer, err := entry.provider.Signer(ctx, keyID)
		if err != nil {
			lastErr = err
			entry.healthy.Store(false)
			continue
		}
		return signer, nil
	}

	return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

func (h *HACryptoProvider) getHealthyProvidersSorted() []haProviderEntry {
	result := make([]haProviderEntry, len(h.providers))
	copy(result, h.providers)

	sort.Slice(result, func(i, j int) bool {
		return result[i].priority < result[j].priority
	})

	return result
}

func (h *HACryptoProvider) healthCheckLoop() {
	for {
		select {
		case <-h.healthTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			for i := range h.providers {
				err := h.providers[i].provider.HealthCheck(ctx)
				h.providers[i].healthy.Store(err == nil)
				h.providers[i].lastCheck = time.Now()
			}
			cancel()

		case <-h.stopCh:
			h.healthTicker.Stop()
			return
		}
	}
}

func (h *HACryptoProvider) Close() {
	close(h.stopCh)
}

func (h *HACryptoProvider) ProviderType() string {
	types := make([]string, len(h.providers))
	for i, p := range h.providers {
		healthy := "up"
		if !p.healthy.Load() {
			healthy = "down"
		}
		types[i] = fmt.Sprintf("%s(%s)", p.provider.ProviderType(), healthy)
	}
	return fmt.Sprintf("ha[%s]", strings.Join(types, ", "))
}
```

### 11.4 Performance Under Load

| Scenario | Single Provider | HA (2 providers) | HA (3 providers) |
|---|---|---|---|
| Normal load | 250 RPS | 500 RPS (load-balanced) | 750 RPS |
| Provider failure | 0 RPS (downtime) | 250 RPS (failover) | 500 RPS |
| Provider recovery | Manual intervention | Automatic | Automatic |
| p99 latency | 40 ms | 25 ms (load spreading) | 20 ms |

HA providers not only improve availability but also **increase throughput** by
distributing load across multiple backends.

### 11.5 Circuit Breaker Pattern

To prevent cascading failures when a crypto provider is down, use a circuit
breaker:

```go
// CircuitBreaker wraps a CryptoProvider with circuit-breaking logic.
type CircuitBreaker struct {
	provider    CryptoProvider
	maxFailures int
	timeout     time.Duration

	failures    atomic.Int32
	state       atomic.Int32 // 0=closed, 1=open, 2=half-open
	lastFailure time.Time
}

const (
	circuitClosed   = 0
	circuitOpen     = 1
	circuitHalfOpen = 2
)

func NewCircuitBreaker(provider CryptoProvider, maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		provider:    provider,
		maxFailures: maxFailures,
		timeout:     timeout,
	}
}

func (cb *CircuitBreaker) Signer(ctx context.Context, keyID string) (crypto.Signer, error) {
	if cb.isOpen() {
		return nil, fmt.Errorf("circuit breaker open for %s", cb.provider.ProviderType())
	}

	signer, err := cb.provider.Signer(ctx, keyID)
	if err != nil {
		cb.recordFailure()
		return nil, err
	}

	cb.resetFailures()
	return signer, nil
}

func (cb *CircuitBreaker) isOpen() bool {
	state := cb.state.Load()
	if state == circuitOpen {
		// Check if timeout has elapsed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state.CompareAndSwap(circuitOpen, circuitHalfOpen)
			return false
		}
		return true
	}
	return false
}

func (cb *CircuitBreaker) recordFailure() {
	failures := cb.failures.Add(1)
	cb.lastFailure = time.Now()
	if failures >= int32(cb.maxFailures) {
		cb.state.Store(circuitOpen)
	}
}

func (cb *CircuitBreaker) resetFailures() {
	cb.failures.Store(0)
	cb.state.Store(circuitClosed)
}
```

---

## 12. Compliance Mapping

### 12.1 When HSM/KMS Is Mandatory

| Compliance Framework | Requirement | HSM Level | GGID Impact |
|---|---|---|---|
| **PCI-DSS 3.5.2** | "Restrict access to cryptographic keys to the fewest necessary." Keys used for card data encryption must be in an HSM or equivalent. | FIPS 140-2 Level 2+ | If GGID processes cardholder data, JWT signing keys must be HSM-backed. |
| **FIPS 140-2 Level 2+** | Required for U.S. federal government cryptographic modules. | Level 2+ | Required if GGID is sold to U.S. federal agencies. |
| **eIDAS QSCD** | Qualified Electronic Signatures require a Qualified Signature Creation Device (QSCD) — an HSM certified to Common Criteria EAL 4+. | Level 3+ (CC EAL 4+) | Required if GGID issues Qualified Electronic Signatures (EU). |
| **HIPAA 164.312(a)(2)(iv)** | "Encryption and decryption" — addressable implementation specification. Not strictly mandatory, but auditors expect it. | Level 2+ recommended | PHI encryption keys should be HSM-protected. |
| **SOC 2 Type II** | CC6.1 "Logical and Physical Access Controls" — encryption key management is reviewed. | Recommended (not required) | HSM demonstrates strong key management controls. |
| **GDPR Article 32** | "Appropriate technical and organisational measures" including encryption. | Recommended | HSM-backed encryption strengthens the "appropriate measures" argument. |
| **ISO 27001 A.10.1.2** | "Key management" — requires formal key management policy and procedures. | Recommended | HSM/KMS demonstrates key management maturity. |
| **FedRAMP** | FISMA Moderate requires FIPS 140-2 validated crypto. | Level 2+ | Required for FedRAMP authorization. |
| **CCPA** | "Reasonable security" — no specific HSM requirement. | Not required | HSM is defense-in-depth. |
| **DoD IL4/IL5** | Requires FIPS 140-2 Level 3 validated modules. | Level 3+ | Required for DoD cloud workloads. |

### 12.2 Compliance Decision Matrix

```
Does GGID process cardholder data?
├── Yes → HSM mandatory (PCI-DSS 3.5.2)
│         Minimum: FIPS 140-2 Level 2 (cloud KMS qualifies)
│         Recommended: Level 3 (dedicated HSM)
└── No ↓

Does GGID serve U.S. federal government?
├── Yes → FIPS 140-2 Level 2+ mandatory (FedRAMP/FISMA)
└── No ↓

Does GGID issue Qualified Electronic Signatures (EU)?
├── Yes → QSCD mandatory (eIDAS), Common Criteria EAL 4+
└── No ↓

Does GGID process PHI (HIPAA)?
├── Yes → HSM strongly recommended for encryption keys
└── No ↓

Is GGID pursuing SOC 2 Type II?
├── Yes → HSM recommended (demonstrates key management controls)
└── No → Cloud KMS is sufficient (better than PEM files on disk)
```

### 12.3 GGID Current Compliance Posture

| Framework | Current State | Gap | Remediation |
|---|---|---|---|
| **PCI-DSS 3.5.2** | Non-compliant (PEM files on disk) | JWT signing keys not in HSM | Migrate to PKCS#11 or cloud KMS |
| **HIPAA** | Partially compliant (AES-256-GCM encryption exists) | Encryption keys not HSM-protected | Use envelope encryption with KMS-backed master key |
| **SOC 2** | Partially compliant | Key management not formalized | Implement KeyLifecycleManager + audit logging |
| **GDPR Art. 32** | Partially compliant | PII encryption key in application config | Move master key to KMS |
| **FIPS 140-2** | Non-compliant (Go crypto is not FIPS-validated) | Go uses its own crypto, not a FIPS module | Use Go with FIPS mode or BoringCrypto |

### 12.4 Go FIPS 140-2 Considerations

Standard Go does not use a FIPS 140-2 validated crypto module. Options:

1. **Google's Go fork with BoringCrypto:** `go.googlesource.com/go` compiled with
   `GOEXPERIMENT=boringcrypto`. Uses BoringSSL which is FIPS 140-2 validated.

2. **AWS LC (LibreSSL Compatible):** Use `github.com/aws/aws-lc-go` for FIPS-validated
   crypto primitives in Go.

3. **External crypto via CGo:** Route crypto operations through a FIPS-validated
   OpenSSL or NSS library via CGo bindings.

4. **HSM offloads all crypto:** If all signing is done in the HSM (PKCS#11), the
   Go crypto module is only used for verification and transport, which may not
   require FIPS validation.

For most compliance scenarios, **option 4 (HSM offload) is the simplest** — the
FIPS-validated boundary is the HSM itself, and the application is outside the
boundary.

---

## 13. Gap Analysis & Recommendations

### 13.1 Current Gaps

Based on the analysis of GGID's `pkg/crypto/` and `services/auth/internal/service/token_service.go`:

| Gap | Severity | Description |
|---|---|---|
| **JWT signing keys in PEM files** | Critical | Private key readable by any process with filesystem access. Container escape or backup theft exposes the key. |
| **No CryptoProvider abstraction** | High | Crypto operations are tightly coupled to file-based keys. No way to swap in HSM/KMS without code changes. |
| **AES encryption key passed as parameter** | High | `AESEncrypt(plaintext, key)` means the key exists in application memory and configuration. No envelope encryption. |
| **No key lifecycle management** | Medium | Keys are generated once and never rotated automatically. No formal lifecycle states. |
| **No FIPS validation** | Medium | Go's standard crypto library is not FIPS 140-2 validated. |
| **No audit trail for key operations** | Medium | Key creation, rotation, and usage are not logged to a tamper-evident system. |
| **No SoftHSM2 in dev environment** | Low | No way to test PKCS#11 code paths locally. |

### 13.2 Recommended Action Items

#### Action 1: Create CryptoProvider Interface (Effort: 2 days)

Create `pkg/cryptoprovider/` with the interface definitions from Section 10.
Implement `LocalProvider` as a wrapper around current behavior. Refactor
`TokenService` to use the provider.

**Deliverables:**
- `pkg/cryptoprovider/provider.go` — Interface definitions
- `pkg/cryptoprovider/local.go` — LocalProvider implementation
- `pkg/cryptoprovider/factory.go` — Provider factory with config
- Updated `token_service.go` to accept `CryptoProvider`
- All existing tests pass without behavior change

**Acceptance criteria:**
- `go test ./pkg/cryptoprovider/...` passes
- `go test ./services/auth/internal/service/...` passes
- Signing performance unchanged (local provider has zero overhead)

#### Action 2: Add PKCS#11 Provider + SoftHSM2 Dev Environment (Effort: 3 days)

Implement `PKCS11Provider` and add SoftHSM2 to the Docker development stack.

**Deliverables:**
- `pkg/cryptoprovider/pkcs11.go` — PKCS#11 provider implementation
- `pkg/cryptoprovider/pkcs11_test.go` — Integration tests using SoftHSM2
- `deploy/Dockerfile.softhsm2` — SoftHSM2 Docker image
- `deploy/docker-compose.softhsm2.yml` — Dev environment with SoftHSM2
- Documentation in `docs/dev/hsm-setup.md`

**Acceptance criteria:**
- `docker compose -f deploy/docker-compose.softhsm2.yml up` starts SoftHSM2
- PKCS#11 signing tests pass against SoftHSM2
- JWT can be signed and verified via PKCS#11 provider

#### Action 3: Add Cloud KMS Providers (Effort: 5 days)

Implement AWS KMS and GCP KMS providers with envelope encryption support.

**Deliverables:**
- `pkg/cryptoprovider/aws_kms.go` — AWS KMS provider
- `pkg/cryptoprovider/gcp_kms.go` — GCP KMS provider
- `pkg/cryptoprovider/aws_kms_test.go` — Tests using LocalStack
- `pkg/cryptoprovider/gcp_kms_test.go` — Tests using KMS emulator
- IAM policy templates for KMS access
- Cost estimation documentation

**Acceptance criteria:**
- JWT signing via AWS KMS works end-to-end
- Envelope encryption via KMS works for PII data
- Signing latency < 50ms p99 (within same region)

#### Action 4: Implement Key Lifecycle Manager (Effort: 3 days)

Implement automated key rotation with overlap windows and tamper-evident audit
logging.

**Deliverables:**
- `pkg/cryptoprovider/lifecycle.go` — KeyLifecycleManager
- `pkg/cryptoprovider/audit.go` — Hash-chained audit logger
- Background rotation scheduler
- Prometheus metrics for key age, rotation events, signing operations

**Acceptance criteria:**
- Keys rotate automatically per configured schedule
- Old keys remain verifiable during overlap window
- All key operations are logged with hash chain
- Zero-downtime rotation verified in integration test

#### Action 5: Add HA-Aware Provider with Circuit Breaker (Effort: 2 days)

Implement the `HACryptoProvider` and `CircuitBreaker` from Section 11.

**Deliverables:**
- `pkg/cryptoprovider/ha.go` — HA provider with health checking
- `pkg/cryptoprovider/circuit_breaker.go` — Circuit breaker
- Configuration for multi-provider failover
- Chaos testing: kill one provider, verify failover

**Acceptance criteria:**
- Automatic failover when primary provider is unhealthy
- Recovery detection (half-open → closed state)
- Circuit breaker prevents cascading failures
- No signing failures during provider switchover

### 13.3 Priority and Timeline

| Action | Priority | Effort | Dependency | Phase |
|---|---|---|---|---|
| 1. CryptoProvider Interface | P0 | 2 days | None | Sprint 1 |
| 2. PKCS#11 + SoftHSM2 | P0 | 3 days | Action 1 | Sprint 1 |
| 3. Cloud KMS Providers | P1 | 5 days | Action 1 | Sprint 2 |
| 4. Key Lifecycle Manager | P1 | 3 days | Actions 1–3 | Sprint 2 |
| 5. HA + Circuit Breaker | P2 | 2 days | Actions 1–3 | Sprint 3 |

**Total effort: 15 engineering days** (3 sprints).

### 13.4 Configuration Examples

**Development (current behavior, no change):**
```yaml
crypto:
  provider: local
  local:
    key_dir: ./configs/keys
```

**Production with HSM:**
```yaml
crypto:
  provider: pkcs11
  pkcs11:
    library_path: /opt/cloudhsm/lib/libcloudhsm_pkcs11.so
    slot_id: 1
    pin: ${HSM_PIN}
    pool_size: 10
```

**Production with AWS KMS:**
```yaml
crypto:
  provider: aws-kms
  aws_kms:
    key_id: arn:aws:kms:us-east-1:123456789012:key/abc123
    region: us-east-1
```

**Production with HA (primary HSM, fallback KMS):**
```yaml
crypto:
  provider: multi
  multi:
    primary:
      provider: pkcs11
      pkcs11:
        library_path: /usr/lib/softhsm/libsofthsm2.so
        slot_id: 0
        pin: ${HSM_PIN}
    secondary:
      provider: aws-kms
      aws_kms:
        key_id: arn:aws:kms:us-east-1:123456789012:key/backup-key
```

---

## Appendix A: Environment Variable Reference

| Variable | Description | Default |
|---|---|---|
| `GGID_CRYPTO_PROVIDER` | Crypto provider type | `local` |
| `GGID_PKCS11_LIB` | PKCS#11 library path | `/usr/lib/softhsm/libsofthsm2.so` |
| `GGID_PKCS11_SLOT` | PKCS#11 slot ID | `0` |
| `GGID_PKCS11_PIN` | PKCS#11 user PIN | (required for PKCS#11) |
| `GGID_PKCS11_KEY_LABEL` | Label of the signing key in the HSM | `jwt-signing-key` |
| `GGID_AWS_KMS_KEY_ID` | AWS KMS key ARN | (required for AWS KMS) |
| `GGID_AWS_REGION` | AWS region | `us-east-1` |
| `GGID_GCP_KMS_KEY_VERSION` | GCP KMS key version resource path | (required for GCP KMS) |
| `GGID_VAULT_ADDR` | Vault server address | (required for Vault) |
| `GGID_VAULT_TOKEN` | Vault auth token | (required for Vault) |
| `GGID_VAULT_TRANSIT_KEY` | Vault Transit key name | `jwt-signing-key` |

## Appendix B: PKCS#11 Mechanism Reference

| Mechanism | Constant | Use Case |
|---|---|---|
| `CKM_RSA_PKCS_KEY_PAIR_GEN` | 0x0000 | RSA key pair generation |
| `CKM_EC_KEY_PAIR_GEN` | 0x1040 | ECDSA key pair generation |
| `CKM_RSA_PKCS` | 0x0001 | RSA PKCS#1 v1.5 signing |
| `CKM_RSA_PKCS_PSS` | 0x000D | RSA-PSS signing |
| `CKM_SHA256_RSA_PKCS` | 0x0040 | SHA-256 + RSA PKCS#1 v1.5 |
| `CKM_SHA384_RSA_PKCS` | 0x0041 | SHA-384 + RSA PKCS#1 v1.5 |
| `CKM_SHA512_RSA_PKCS` | 0x0042 | SHA-512 + RSA PKCS#1 v1.5 |
| `CKM_ECDSA` | 0x0104 | ECDSA signing (raw hash) |
| `CKM_ECDSA_SHA256` | 0x0106 | ECDSA + SHA-256 |
| `CKM_AES_KEY_GEN` | 0x1080 | AES key generation |
| `CKM_AES_GCM` | 0x1087 | AES-GCM encryption |

## Appendix C: Cost Comparison Summary

| Solution | Monthly Cost | Security Level | Latency | Best For |
|---|---|---|---|---|
| PEM files (current) | $0 | Low | <0.1 ms | Development only |
| SoftHSM2 | $0 | Medium | <0.5 ms | Testing, low-risk production |
| AWS KMS | ~$10/month | High (FIPS L2) | 40 ms | Cloud-native, cost-sensitive |
| GCP Cloud KMS (HSM) | ~$27/month | High (FIPS L3) | 50 ms | GCP-native, FIPS L3 needed |
| Azure Key Vault (Premium) | ~$3/month | High (FIPS L2) | 55 ms | Azure-native |
| Vault Transit (self-hosted) | ~$200+/month | Configurable | 0.8 ms | Multi-cloud, high throughput |
| AWS CloudHSM (dedicated) | ~$1,100/month | Very High (FIPS L3) | 8 ms | Compliance-mandated, high volume |
| Thales Luna Network HSM | ~$2,000/month | Very High (FIPS L3) | 5 ms | On-premise, government |
| PCIe HSM | ~$1,000/month amortized | Very High (FIPS L3) | 0.2 ms | Latency-critical, single-host |

## Appendix D: References

- [FIPS 140-3: Security Requirements for Cryptographic Modules](https://csrc.nist.gov/publications/detail/fips/140/3/final)
- [PKCS #11 v3.0 Specification (OASIS)](https://docs.oasis-open.org/pkcs11/pkcs11-spec/v3.0/pkcs11-spec-v3.0.html)
- [AWS KMS Developer Guide](https://docs.aws.amazon.com/kms/latest/developerguide/)
- [Google Cloud KMS Documentation](https://cloud.google.com/kms/docs)
- [Azure Key Vault Documentation](https://docs.microsoft.com/en-us/azure/key-vault/)
- [HashiCorp Vault Transit Secrets Engine](https://developer.hashicorp.com/vault/docs/secrets/transit)
- [SoftHSM2 Project](https://www.opendnssec.org/softhsm/)
- [PCI-DSS v4.0 Requirements](https://www.pcisecuritystandards.org/document_library)
- [eIDAS Regulation (EU 910/2014)](https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=uriserv:OJ.L_.2014.257.01.0073.01.ENG)
- [Go BoringCrypto (FIPS)](https://go.googlesource.com/go/+/refs/heads/dev.boringcrypto/misc/boring/)

---

*This document is a research deliverable for the GGID project. It focuses on
deep technical integration patterns for HSM and KMS-backed cryptography. For
high-level secret management strategy, see `secret-management-iam.md`. For key
lifecycle state machine details, see `key-rotation-iam.md`.*
