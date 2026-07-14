# FIDO2 Server Certification Guide for IAM Systems

> **Scope**: The complete certification process for a FIDO2 Server (Relying Party
> server) — Conformance Tools, Functional Certification, Security Requirements,
> documentation, timeline, cost, and a readiness roadmap for GGID.
>
> **Companion document**: `docs/research/fido-alliance-interop-report.md` covers
> authenticator matrix, attestation format details, AAGUID lookup, and MDS
> integration internals. This document focuses on the **certification process**
> itself and does not repeat those details.

---

## Table of Contents

1. [FIDO2 Certification Overview](#1-fido2-certification-overview)
2. [FIDO Conformance Tools](#2-fido-conformance-tools)
3. [FIDO2 Server Functional Certification](#3-fido2-server-functional-certification)
4. [FIDO Security Requirements (FIDO Sec Req)](#4-fido-security-requirements-fido-sec-req)
5. [Authenticator Certification Levels](#5-authenticator-certification-levels)
6. [Documentation Requirements](#6-documentation-requirements)
7. [Certification Timeline](#7-certification-timeline)
8. [Cost Analysis](#8-cost-analysis)
9. [Interoperability Testing](#9-interoperability-testing)
10. [GGID WebAuthn Implementation Assessment](#10-ggid-webauthn-implementation-assessment)
11. [Certification Readiness Roadmap](#11-certification-readiness-roadmap)
12. [Gap Analysis and Recommendations](#12-gap-analysis-and-recommendations)
13. [Appendix A: Go Code Examples](#appendix-a-go-code-examples)
14. [Appendix B: Test Configurations](#appendix-b-test-configurations)

---

## 1. FIDO2 Certification Overview

### 1.1 The FIDO Alliance

The Fast IDentity Online (FIDO) Alliance is an open industry association founded
in 2013 with a focused mission: develop and promote authentication standards
that reduce the world's over-reliance on passwords. The Alliance maintains two
primary specification families:

- **FIDO U2F** (Universal 2nd Factor) — the original hardware-key second-factor
  standard (now legacy but still supported).
- **FIDO2** — the modern passwordless standard comprising:
  - **WebAuthn** (W3C Web Authentication API) — browser-side specification
  - **CTAP** (Client-to-Authenticator Protocol) — device-side specification

FIDO2 is recognized by major browser vendors (Google Chrome, Apple Safari,
Mozilla Firefox, Microsoft Edge) and platform providers (Apple, Google,
Microsoft). The Alliance operates a certification program that validates
compliance and interoperability across the ecosystem.

### 1.2 Certification Programs

The FIDO Alliance operates multiple certification programs that cover different
layers of the FIDO2 stack:

```
┌─────────────────────────────────────────────────────────────────────┐
│                     FIDO2 Ecosystem Layers                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐    ┌─────────────────┐    ┌────────────────┐ │
│  │ Authenticator   │    │  Client /       │    │  Server (RP)   │ │
│  │  Certification  │    │  Host Platform  │    │  Certification │ │
│  │  (L1, L2, L3)   │    │  Certification  │    │                │ │
│  └────────┬────────┘    └────────┬────────┘    └───────┬────────┘ │
│           │                       │                     │          │
│           ▼                       ▼                     ▼          │
│     YubiKey 5              Windows Hello          GGID Server     │
│     Touch ID               Chrome                 Auth0           │
│     Titan                  Android                Okta            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

#### Authenticator Certification (L1, L2, L3)

Certifies that a hardware or software authenticator meets specific security
requirements. This is the most well-known program — YubiKeys, Touch ID, and
Windows Hello all carry authenticator certifications.

| Level | Name | Security Posture | Typical Examples |
|-------|------|-----------------|-----------------|
| **L1** | FIDO Certified | Basic: software or hardware key storage | Software authenticators, entry-level tokens |
| **L2** | FIDO Certified+ | Advanced: user verification (biometric/PIN), secure hardware | YubiKey Bio, Windows Hello, Touch ID |
| **L3** | FIDO Certified++ | High assurance: certified secure element, anti-physical attacks | YubiKey 5 FIPS, hardware security modules |

#### Server Certification (FIDO2 Server)

Certifies that a Relying Party server correctly implements WebAuthn/FIDO2
protocols. This is the certification path relevant to GGID. A server
certification demonstrates that:

1. The server correctly processes registration ceremonies (attestation
   verification, credential storage).
2. The server correctly processes authentication ceremonies (assertion
   verification, counter checking).
3. The server handles all standard attestation formats.
4. The server integrates with the FIDO Metadata Service (MDS).
5. The server passes the FIDO Conformance Tools test suite.

#### Client / Host Platform Certification

Certifies that a browser, OS, or middleware correctly mediates between
authenticators and servers. Chrome, Safari, Firefox, and Windows all carry this
certification. GGID does not build a client, so this program is not applicable.

### 1.3 What GGID Would Certify As

GGID would pursue **FIDO2 Server Certification**. In FIDO terminology, GGID acts
as a **Relying Party (RP)** — the server that a user authenticates *to*. The
certification validates GGID's WebAuthn endpoints:

- **Registration**: `POST /api/v1/webauthn/register/begin` and
  `POST /api/v1/webauthn/register/finish` — server generates challenges,
  verifies attestation, stores credentials.
- **Authentication**: `POST /api/v1/webauthn/auth/begin` and
  `POST /api/v1/webauthn/auth/finish` — server generates challenges, verifies
  assertions, checks counters.
- **Well-known endpoints**: `/.well-known/webauthn` (Related Origin Requests),
  `/.well-known/assetlinks.json` (Android), `/.well-known/apple-app-site-association`
  (iOS).

The certification scope covers the server-side protocol handling, not the
frontend JavaScript that calls `navigator.credentials.create()` /
`navigator.credentials.get()`. However, the server must produce correct
`PublicKeyCredentialCreationOptions` and `PublicKeyCredentialRequestOptions`
JSON that any standards-compliant browser can consume.

### 1.4 Why Certification Matters

#### Enterprise Trust

Large enterprise customers — especially in finance, healthcare, government, and
critical infrastructure — require FIDO2 certification as a procurement
gatekeeper. A FIDO Certified server logo on the GGID marketing site signals:

- The implementation has been independently tested by the FIDO Alliance.
- It interoperates with any FIDO Certified authenticator (hundreds of devices).
- It follows the specification precisely, reducing integration risk.

Without certification, many RFPs will filter GGID out before evaluation.

#### Interoperability Guarantee

The FIDO ecosystem has grown to include hundreds of authenticators from dozens
of manufacturers. A certification proves that GGID can interoperate with all of
them — not just the ones GGID's team has manually tested. This is backed by the
Conformance Tools test suite, which exercises edge cases that real-world testing
might miss.

#### Marketing and Competitive Positioning

Competitors that hold FIDO2 Server Certification:

| Vendor | FIDO2 Server Certified | Notes |
|--------|----------------------|-------|
| Auth0 (Okta) | Yes | Certified across multiple product tiers |
| Okta | Yes | Full FIDO2/W3C WebAuthn support |
| Duo (Cisco) | Yes | Strong in the authenticator certification space |
| Microsoft Entra ID | Yes | Azure AD / Entra ID |
| Google Cloud Identity | Yes | Google Workspace |
| Ping Identity | Yes | PingFederate / PingOne |
| Keycloak | No (community) | Community-driven, not formally certified |
| Ory | No (community) | Open-source, not formally certified |

GGID can differentiate from community projects (Keycloak, Ory) and match
enterprise vendors (Auth0, Okta, Duo) by achieving certification.

#### Regulatory Compliance

Several regulatory frameworks increasingly reference FIDO2:

- **PSD2 SCA** (Payment Services Directive 2, Strong Customer Authentication) —
  FIDO2 is recognized as a compliant strong authentication mechanism.
- **NIST SP 800-63B** — AAL2/AAL3 can be achieved with FIDO2 authenticators.
- **HIPAA** — FIDO2 provides strong authentication for healthcare access.
- **FedRAMP** — FIDO2 is accepted for federal cloud authentication.

Certification provides audit-ready evidence that GGID meets these requirements.

---

## 2. FIDO Conformance Tools

### 2.1 Overview

The FIDO Alliance provides **FIDO Conformance Tools** — an online test suite
that validates server implementations against the FIDO2 specification. The tools
are available to FIDO Alliance members (or via the FIDO Dev Days program for
non-members).

Access URL: `https://fidoalliance.org/fido-conformance-tools/`

The Conformance Tools operate as a **virtual authenticator** — they act as both
the client (browser) and the authenticator, sending carefully crafted
WebAuthn requests to the server under test. This means the server must expose
standard WebAuthn endpoints that the tools can target.

### 2.2 Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                     Conformance Tools Architecture                    │
│                                                                       │
│  ┌──────────────────┐     HTTPS      ┌──────────────────────────┐   │
│  │                  │ ──────────────►│                          │   │
│  │  FIDO Conformance│   (WebAuthn)   │  Your Server (GGID)      │   │
│  │  Tools Server    │◄──────────────│  /api/v1/webauthn/*      │   │
│  │                  │                │                          │   │
│  │  • Virtual       │                │  Returns standard JSON:  │   │
│  │    Authenticator │                │  • PublicKeyCredential   │   │
│  │  • Test Scripts  │                │    CreationOptions       │   │
│  │  • Result Tracker│                │  • PublicKeyCredential   │   │
│  │                  │                │    RequestOptions        │   │
│  └──────────────────┘                └──────────────────────────┘   │
│                                                                       │
│  The tools send crafted WebAuthn requests and verify the server's     │
│  responses conform to the specification.                              │
└──────────────────────────────────────────────────────────────────────┘
```

### 2.3 Test Categories

The Conformance Tools are organized into categories that mirror the WebAuthn
specification structure:

#### Registration Tests

| Test Group | Description | Count (approx.) |
|-----------|-------------|----------------|
| `P-1` | Server returns valid `PublicKeyCredentialCreationOptions` | 5 |
| `P-2` | Server correctly processes attestation formats | 12 |
| `P-3` | Server correctly verifies authenticator data flags | 8 |
| `P-4` | Server handles excludeCredentials properly | 4 |
| `P-5` | Server validates origin and RP ID hash | 6 |
| `P-6` | Server handles error cases (malformed input) | 10 |
| `P-7` | Attestation statement verification | 15 |

#### Authentication Tests

| Test Group | Description | Count (approx.) |
|-----------|-------------|----------------|
| `F-1` | Server returns valid `PublicKeyCredentialRequestOptions` | 4 |
| `F-2` | Server correctly verifies assertion signatures | 8 |
| `F-3` | Server checks user present / user verified flags | 6 |
| `F-4` | Server validates sign counter (clone detection) | 5 |
| `F-5` | Server handles allowCredentials properly | 4 |
| `F-6` | Server handles error cases (malformed assertion) | 8 |

#### Assertion / Edge Case Tests

| Test Group | Description | Count (approx.) |
|-----------|-------------|----------------|
| `A-1` | Empty attestation (`none` format) | 3 |
| `A-2` | Extension support (`credProtect`, `hmac-secret`) | 5 |
| `A-3` | Large credential IDs and public keys | 3 |
| `A-4` | Unicode user names and display names | 4 |
| `A-5` | Concurrent registration/authentication sessions | 3 |

### 2.4 Setup Guide

#### Step 1: Register Server URL

Before testing, register your server's base URL with the FIDO Alliance. The
Conformance Tools will only send requests to this registered URL.

```yaml
# Example server registration with FIDO Alliance
server_metadata:
  rp_id: "ggid.example.com"
  rp_name: "GGID Identity Platform"
  origins:
    - "https://ggid.example.com"
  endpoints:
    registration_begin: "/api/v1/webauthn/register/begin"
    registration_finish: "/api/v1/webauthn/register/finish"
    authentication_begin: "/api/v1/webauthn/auth/begin"
    authentication_finish: "/api/v1/webauthn/auth/finish"
```

#### Step 2: Configure Test Metadata

The Conformance Tools need to know which attestation formats and authenticator
types to test. GGID must expose a metadata endpoint or configure this manually:

```json
{
  "supported_attestation_formats": [
    "none",
    "packed",
    "fido-u2f",
    "android-key",
    "android-safetynet",
    "tpm",
    "apple"
  ],
  "supported_algorithms": [
    -7,   // ES256 (ECDSA P-256)
    -257, // RS256 (RSA 2048)
    -8    // EdDSA
  ],
  "supported_transports": [
    "usb", "nfc", "ble", "internal", "hybrid", "smart-card"
  ],
  "user_verification": "preferred",
  "resident_key": "preferred"
}
```

#### Step 3: Ensure Session Management

The Conformance Tools perform registration and authentication as multi-step
ceremonies. The server must maintain session state between the "begin" and
"finish" calls. GGID's in-memory session store with 5-minute expiry is
sufficient for conformance testing.

#### Step 4: Disable Production Security Controls (Temporarily)

For conformance testing, some production security controls must be relaxed:

```go
// Conformance test mode configuration
type ConformanceConfig struct {
    // Disable rate limiting — tools send rapid bursts
    DisableRateLimit bool `yaml:"disable_rate_limit"`

    // Disable tenant isolation — tools don't send X-Tenant-ID
    DefaultTenantID  string `yaml:"default_tenant_id"`

    // Disable CSRF — tools use direct API calls
    DisableCSRF      bool `yaml:"disable_csrf"`

    // Accept test origins
    AllowedOrigins   []string `yaml:"allowed_origins"`
}
```

#### Step 5: Run Tests

Navigate to the FIDO Conformance Tools portal, select the test category, and
initiate a test run. The tools will:

1. Call your `/register/begin` endpoint
2. Parse the returned options
3. Construct a virtual authenticator response
4. POST it to your `/register/finish` endpoint
5. Verify the server's response
6. Report pass/fail with detailed error messages

### 2.5 Interpreting Results

The Conformance Tools produce detailed reports:

```
FIDO2 Server Conformance Test Report
=====================================
Server: ggid.example.com
Date: 2025-01-24 14:32:00 UTC
Overall Result: FAIL (87/105 passed)

Registration Tests:
  P-1.1: PublicKeyCredentialCreationOptions valid .......... PASS
  P-1.2: Challenge is cryptographically random ............. PASS
  P-1.3: RP ID matches registered server ................... PASS
  P-1.4: User verification set correctly ................... PASS
  P-1.5: pubKeyCredParams includes ES256 ................... PASS
  P-2.1: "none" attestation accepted ....................... PASS
  P-2.2: "packed" attestation - ECDSA verified ............. PASS
  P-2.3: "packed" attestation - RSA verified ............... PASS
  P-2.4: "packed" attestation - EdDSA verified ............. PASS
  P-2.5: "fido-u2f" attestation verified ................... FAIL
         Error: Server did not verify attestation signature
  P-2.6: "android-key" attestation verified ................ FAIL
         Error: Extension OID not checked properly
  ...

Authentication Tests:
  F-1.1: PublicKeyCredentialRequestOptions valid ........... PASS
  F-2.1: Assertion signature verified ...................... PASS
  F-2.2: Sign counter checked .............................. PASS
  ...
```

### 2.6 Common Failures

Based on analysis of the WebAuthn specification and common implementation
pitfalls:

| # | Failure | Root Cause | Fix |
|---|---------|-----------|-----|
| 1 | Challenge not cryptographically random | Using sequential or timestamp-based challenges | Use `crypto/rand` (GGID already does via go-webauthn) |
| 2 | RP ID hash not verified | Missing or incorrect hash comparison | Verify `authData[0:32] == SHA256(rpId)` |
| 3 | Origin not validated | Accepting any origin | Check `clientData.origin` against allowed list |
| 4 | Attestation signature not verified | Only checking cert format, not signature | Implement full signature verification for each format |
| 5 | Sign counter not checked | Ignoring `authenticatorData.signCount` | Compare against stored counter, reject if not increasing |
| 6 | `flags.UP` not checked on authentication | Accepting assertions without user presence | Verify `authData[32] & 0x01 != 0` |
| 7 | Extensions not handled | Crashing on unknown extensions | Ignore unknown extensions gracefully |
| 8 | User verification flag ignored | Not checking `flags.UV` when required | Verify UV based on `userVerification` policy |
| 9 | Malformed CBOR crashes server | No input validation | Validate CBOR structure before parsing |
| 10 | Session not invalidated after use | Reusing challenge across ceremonies | Delete session after finish (GGID does this) |

### 2.7 Go Code: Test-Compatible Server Setup

```go
// Package conformance provides a WebAuthn server configured for FIDO
// Conformance Tools testing.
package conformance

import (
    "crypto/rand"
    "encoding/base64"
    "net/http"
    "time"

    "github.com/go-webauthn/webauthn/protocol"
    "github.com/go-webauthn/webauthn/webauthn"
)

// ConformanceServer wraps a WebAuthn handler configured for FIDO
// Conformance Tools.
type ConformanceServer struct {
    wbn      *webauthn.WebAuthn
    sessions map[string]*ConformanceSession
}

// ConformanceSession holds the state between begin and finish calls.
type ConformanceSession struct {
    Challenge  string
    UserID     []byte
    CreatedAt  time.Time
}

// NewConformanceServer creates a server ready for FIDO Conformance Tools.
//
// rpID must match the registered server domain.
// origins must include all origins the conformance tools will use.
func NewConformanceServer(rpID, rpName string, origins []string) (*ConformanceServer, error) {
    config := &webauthn.Config{
        RPDisplayName: rpName,
        RPID:          rpID,
        RPOrigins:     origins,
        // AttestationPreference: This controls what the server requests.
        // For conformance testing, accept all attestation formats.
        AttestationPreference: protocol.PreferDirectAttestation,

        // Authenticator selection — conformance tools test all variants
        AuthenticatorSelection: protocol.AuthenticatorSelection{
            ResidentKey:      protocol.ResidentKeyRequirementPreferred,
            UserVerification: protocol.VerificationPreferred,
        },

        // Debug mode — log all attestation details for troubleshooting
        Timeout: 60000, // 60 seconds (conformance tools may be slow)
    }

    wbn, err := webauthn.New(config)
    if err != nil {
        return nil, err
    }

    return &ConformanceServer{
        wbn:      wbn,
        sessions: make(map[string]*ConformanceSession),
    }, nil
}

// GenerateChallenge creates a cryptographically random challenge.
// FIDO requires at least 16 bytes of entropy.
func GenerateChallenge() (string, error) {
    buf := make([]byte, 32)
    if _, err := rand.Read(buf); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(buf), nil
}

// VerifyChallenge checks that the challenge in the client response matches
// the stored challenge. This prevents replay attacks.
func (s *ConformanceServer) VerifyChallenge(challenge string) bool {
    sess, ok := s.sessions[challenge]
    if !ok {
        return false
    }
    // Check expiry
    if time.Since(sess.CreatedAt) > 5*time.Minute {
        delete(s.sessions, challenge)
        return false
    }
    // Consume the challenge (one-time use)
    delete(s.sessions, challenge)
    return true
}
```

---

## 3. FIDO2 Server Functional Certification

### 3.1 Requirements Overview

To achieve FIDO2 Server Functional Certification, a server must:

1. **Pass the FIDO Conformance Tools** — all mandatory test cases must pass.
2. **Support all standard attestation formats** — at minimum: `none`, `packed`,
   `fido-u2f`, `tpm`, `android-key`, `android-safetynet`, and `apple`.
3. **Integrate with the FIDO Metadata Service (MDS)** — download and cache MDS
   BLOB, look up authenticator metadata by AAGUID.
4. **Handle registration and authentication ceremonies** — full implementation
   of WebAuthn Level 2 (or later).
5. **Support extensions** — at least `appid` and `credProtect`.

### 3.2 Mandatory Server Capabilities

| Capability | Requirement | GGID Status |
|-----------|-------------|-------------|
| Generate challenges | ≥16 bytes random | PASS (go-webauthn) |
| Verify clientDataJSON | Origin, type, challenge | PASS (go-webauthn) |
| Verify RP ID hash | SHA256(rpId) == authData[0:32] | PASS (go-webauthn) |
| Verify user present flag | flags.UP == 1 | PASS (go-webauthn) |
| Verify user verified flag | flags.UV based on policy | PASS (go-webauthn) |
| Verify attestation signature | Per-format verification | PARTIAL (see Section 10) |
| Store credential | ID, public key, counter | PASS |
| Verify assertion signature | ECDSA/RSA/EdDSA | PASS (go-webauthn) |
| Check sign counter | Monotonic increase | PASS |
| Handle "none" attestation | Accept without verification | PASS |
| Handle "packed" attestation | Full verification | PASS |
| Handle "fido-u2f" attestation | Full verification | PARTIAL |
| Handle "tpm" attestation | Full verification | PARTIAL |
| Handle "android-key" attestation | Full verification | PARTIAL |
| Handle "android-safetynet" | JWS verification | PARTIAL |
| Handle "apple" attestation | Nonce verification | PARTIAL |
| MDS integration | Download, cache, lookup | MISSING |
| Extension support | appid, credProtect | PARTIAL |
| Related Origin Requests | /.well-known/webauthn | PASS |

### 3.3 Test Matrix for Functional Certification

The certification test matrix covers combinations of:

#### Authenticator Types

| Type | Attachment | Resident Key | User Verification |
|------|-----------|-------------|-------------------|
| Platform authenticator | platform | required | required |
| Roaming authenticator (USB) | cross-platform | preferred | preferred |
| Roaming authenticator (NFC) | cross-platform | discouraged | discouraged |
| Hybrid (phone) | cross-platform | preferred | required |
| Software authenticator | platform | preferred | discouraged |

#### Attestation Formats

Each authenticator type is tested across all attestation formats:

| Format | Authenticator | Crypto | Test Focus |
|--------|--------------|--------|-----------|
| `none` | Any | N/A | Server accepts without verification |
| `packed` (self) | Same device | ES256/RS256 | Server verifies self-attestation |
| `packed` (basic) | Manufacturer cert | ES256/RS256/EdDSA | Server verifies cert chain |
| `packed` (attCA) | CA-issued cert | ES256/RS256 | Server verifies CA chain |
| `fido-u2f` | U2F device | ECDSA P-256 | Server verifies U2F register response |
| `tpm` | Windows Hello | RSA | Server verifies TPM structure |
| `android-key` | Android device | ECDSA | Server verifies key attestation extension |
| `android-safetynet` | Android device | JWS | Server verifies JWS response |
| `apple` | Apple device | ECDSA P-256 | Server verifies nonce extension |

#### Transport Types

| Transport | Protocol | Test Method |
|-----------|---------|-------------|
| `usb` | CTAP over USB | Virtual USB authenticator |
| `nfc` | CTAP over NFC | Virtual NFC authenticator |
| `ble` | CTAP over BLE | Virtual BLE authenticator |
| `internal` | Platform API | Virtual platform authenticator |
| `hybrid` | caBLE | Virtual hybrid authenticator |
| `smart-card` | APDU | Virtual smart card |

### 3.4 Submission Process

```
┌─────────────────────────────────────────────────────────────┐
│                 Certification Submission Flow                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Become FIDO Alliance Member                             │
│     └─► Annual membership fee ($5K-$50K)                    │
│                                                             │
│  2. Access Conformance Tools                                │
│     └─► Configure server URL and metadata                   │
│                                                             │
│  3. Run Conformance Tests                                   │
│     └─► All mandatory tests must PASS                      │
│     └─► Save test results report                            │
│                                                             │
│  4. Submit Certification Application                        │
│     └─► Include: conformance report                         │
│     └─► Include: server documentation                       │
│     └─► Include: security documentation                     │
│     └─► Include: interoperability test results              │
│                                                             │
│  5. FIDO Alliance Review                                    │
│     └─► Technical review (2-4 weeks)                        │
│     └─► Possible clarification requests                     │
│                                                             │
│  6. Certification Granted                                   │
│     └─► Listed on fidoalliance.org/certified                │
│     └─► Permission to use FIDO Certified logo               │
│                                                             │
│  7. Annual Maintenance                                      │
│     └─► Re-run conformance on new versions                  │
│     └─► Report significant changes                          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 3.5 Certification Marks and Usage

Upon certification, GGID receives:

- **FIDO Certified Server** logo for use in marketing materials
- Listing in the FIDO Alliance Certified Products registry
- A certificate number for audit and procurement purposes
- Access to the FIDO Alliance trademark usage guidelines

The certification is valid for the specific software version tested. Minor
updates (bug fixes, security patches) do not require re-certification, but
major version changes or significant protocol modifications do.

---

## 4. FIDO Security Requirements (FIDO Sec Req)

### 4.1 Overview

The FIDO Security Requirements document (`FIDO Security Requirements`) defines
the security controls that certified products must implement. While the
Functional Certification validates protocol correctness, the Security
Requirements validate the underlying security posture of the implementation.

The document is organized into requirement categories, each with specific
controls that must be met or exceeded.

### 4.2 Requirement Categories

#### REQ-1: Key Protection

**Requirement**: Cryptographic keys used for WebAuthn operations must be
protected against unauthorized access, extraction, or modification.

| Control | Description | GGID Status |
|---------|-------------|-------------|
| REQ-1.1 | Private keys stored in encrypted form at rest | PASS — credentials stored with encrypted public key in PostgreSQL |
| REQ-1.2 | Key access requires authentication | PASS — DB access requires authenticated connection |
| REQ-1.3 | Key material never logged | PARTIAL — AAGUID and attestation type are logged; key material is not |
| REQ-1.4 | Key rotation capability | GAP — no WebAuthn-specific key rotation (the credential keys are user-held) |
| REQ-1.5 | HSM or secure element for server-side keys | GAP — RP attestation keys not using HSM (not required for RP servers) |

**Gap Analysis for GGID**:
- The RP (server) does not hold private keys for user credentials — those are on
  the authenticator. The server stores only public keys.
- If GGID were to support RP attestation (server-side attestation signing), it
  would need an HSM. Currently, GGID delegates attestation verification to
  go-webauthn and does not sign RP attestation.

#### REQ-2: Audit Logging

**Requirement**: All security-relevant operations must be logged with sufficient
detail for forensic analysis.

| Control | Description | GGID Status |
|---------|-------------|-------------|
| REQ-2.1 | Log all registration ceremonies | GAP — no WebAuthn-specific audit logging |
| REQ-2.2 | Log all authentication ceremonies | GAP — no WebAuthn-specific audit logging |
| REQ-2.3 | Log credential deletion | GAP — not logged |
| REQ-2.4 | Log attestation verification results | GAP — not logged |
| REQ-2.5 | Log counter mismatch (clone detection) | GAP — returns error but doesn't log |
| REQ-2.6 | Logs tamper-resistant | PASS — GGID uses NATS JetStream audit with hash chain |
| REQ-2.7 | Logs retained per compliance requirements | PASS — configurable retention |

**Gap Analysis for GGID**:
GGID has a robust audit infrastructure (NATS JetStream + hash chain + REST query)
but WebAuthn operations are NOT wired to it. This is a significant gap that
requires adding audit events for all WebAuthn ceremonies.

```go
// Required audit events for FIDO compliance
type WebAuthnAuditEvent struct {
    EventType   string    // "webauthn.register", "webauthn.authenticate", etc.
    TenantID    uuid.UUID
    UserID      uuid.UUID
    CredentialID string
    AAGUID      string
    Success     bool
    ErrorReason string
    Timestamp   time.Time
    IPAddress   string
    UserAgent   string
}
```

#### REQ-3: Authentication of Operations

**Requirement**: Sensitive operations must require appropriate authentication
and authorization.

| Control | Description | GGID Status |
|---------|-------------|-------------|
| REQ-3.1 | Registration requires authenticated session | PARTIAL — uses X-Tenant-ID + user_id (no JWT verification on WebAuthn endpoints) |
| REQ-3.2 | Credential deletion requires owner verification | PARTIAL — uses X-Tenant-ID + user_id |
| REQ-3.3 | Admin operations require admin role | GAP — no admin-level WebAuthn operations |
| REQ-3.4 | Rate limiting on authentication | PARTIAL — gateway rate limits, but not WebAuthn-specific |

**Gap Analysis for GGID**:
WebAuthn endpoints accept `X-Tenant-ID` header and `user_id` query parameter
without JWT verification. In production, an authenticated user could register
WebAuthn credentials for another user by spoofing the user_id. This is a P0
security gap.

#### REQ-4: Vulnerability Management

**Requirement**: Certified products must have a vulnerability management process.

| Control | Description | GGID Status |
|---------|-------------|-------------|
| REQ-4.1 | Security vulnerability response process | PASS — documented in incident-response.md |
| REQ-4.2 | Dependency scanning | PARTIAL — Go modules, no automated scanning |
| REQ-4.3 | Penetration testing | GAP — no FIDO-specific pentest |
| REQ-4.4 | Security advisories | PARTIAL — GitHub security advisories |
| REQ-4.5 | Coordinated disclosure | PASS — security policy documented |

#### REQ-5: Cryptographic Implementation

**Requirement**: Cryptographic operations must use vetted, standards-compliant
implementations.

| Control | Description | GGID Status |
|---------|-------------|-------------|
| REQ-5.1 | Use NIST-approved algorithms | PASS — ES256, RS256, EdDSA |
| REQ-5.2 | No custom crypto | PASS — uses Go crypto stdlib |
| REQ-5.3 | Constant-time comparisons | PASS — uses `crypto/subtle` in key paths |
| REQ-5.4 | Secure random number generation | PASS — uses `crypto/rand` |
| REQ-5.5 | Key sizes meet minimum requirements | PASS — P-256, RSA-2048, Ed25519 |

#### REQ-6: Operational Security

**Requirement**: The production deployment must meet operational security
standards.

| Control | Description | GGID Status |
|---------|-------------|-------------|
| REQ-6.1 | TLS for all external communication | PASS — HTTPS enforced at gateway |
| REQ-6.2 | Internal service encryption | GAP — gRPC plaintext between services |
| REQ-6.3 | Secrets management | PASS — env vars / keys.env |
| REQ-6.4 | Access controls on infrastructure | PASS — Docker network isolation |
| REQ-6.5 | Monitoring and alerting | PARTIAL — healthchecks, no WebAuthn-specific alerts |

### 4.3 Security Requirements Mapping Summary

```
┌─────────────────────────────────────────────────────────────────┐
│              FIDO Sec Req Compliance Scorecard                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  REQ-1: Key Protection         ████████████████░░░░  80%        │
│  REQ-2: Audit Logging          ██████░░░░░░░░░░░░░░  30%        │
│  REQ-3: Auth of Operations     ██████████░░░░░░░░░░  50%        │
│  REQ-4: Vulnerability Mgmt     ████████████████░░░░  80%        │
│  REQ-5: Crypto Implementation  ████████████████████  100%       │
│  REQ-6: Operational Security   ████████████████░░░░  80%        │
│                                                                 │
│  Overall Compliance:                                     70%    │
│                                                                 │
│  Critical gaps:                                                 │
│  • Audit logging for WebAuthn ceremonies (REQ-2)                │
│  • JWT verification on WebAuthn endpoints (REQ-3)              │
│  • gRPC TLS between services (REQ-6)                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Authenticator Certification Levels

### 5.1 Overview

While GGID certifies as a **Server** (not an authenticator), understanding
authenticator certification levels is critical for two reasons:

1. **Policy enforcement** — GGID's per-tenant policy should be able to enforce
   a minimum authenticator certification level.
2. **Metadata-based decisions** — When a user registers with an authenticator,
   GGID should check its certification level via the FIDO Metadata Service.

### 5.2 Level 1 (L1) — FIDO Certified

**Security posture**: Basic security. The authenticator stores keys in
software or basic hardware. No user verification requirement.

| Aspect | Requirement |
|--------|-------------|
| Key storage | Software or hardware |
| User verification | Not required |
| Secure element | Not required |
| Anti-cloning | Basic (sign counter) |
| Typical examples | Software tokens, entry-level USB keys |

**Use cases**: Consumer applications, low-risk authentication, second-factor
scenarios where a password is still primary.

### 5.3 Level 2 (L2) — FIDO Certified+

**Security posture**: Advanced security. The authenticator requires user
verification (biometric or PIN) and uses secure hardware for key storage.

| Aspect | Requirement |
|--------|-------------|
| Key storage | Secure hardware (TEE or secure element) |
| User verification | Required (local biometric or PIN) |
| Secure element | Required (Common Criteria EAL2+ equivalent) |
| Anti-cloning | Hardware-backed |
| Typical examples | YubiKey Bio, Windows Hello, Apple Touch ID/Face ID |

**Use cases**: Enterprise authentication, financial services, healthcare. This
is the minimum level most enterprises require for passwordless authentication.

### 5.4 Level 3 (L3) — FIDO Certified++

**Security posture**: High assurance. The authenticator uses a certified
secure element (Common Criteria EAL4+ or higher) and provides protection
against physical attacks.

| Aspect | Requirement |
|--------|-------------|
| Key storage | Certified secure element (EAL4+) |
| User verification | Required (with on-chip matching) |
| Physical attack resistance | Side-channel, fault injection protections |
| Certification evidence | Formal evaluation report |
| Typical examples | YubiKey 5 FIPS, hardware security modules |

**Use cases**: Government, defense, high-value financial transactions. Required
for FIDO2-based PSD2 SCA at the highest assurance levels.

### 5.5 Why GGID Should Care About Authenticator Levels

#### Policy Enforcement

Enterprise tenants need to control which authenticators their employees can use.
GGID should support per-tenant policies:

```go
// Per-tenant authenticator policy
type WebAuthnPolicy struct {
    TenantID              uuid.UUID

    // Minimum authenticator certification level
    // 0 = any, 1 = L1, 2 = L2, 3 = L3
    MinAuthenticatorLevel int `json:"min_authenticator_level"`

    // Allowed attestation formats (empty = all allowed)
    AllowedAttestationFormats []string `json:"allowed_attestation_formats"`

    // Require user verification
    RequireUserVerification bool `json:"require_user_verification"`

    // Require backup-eligible credentials
    RequireBackupEligible bool `json:"require_backup_eligible"`

    // Allowed AAGUIDs (empty = all allowed, populated = whitelist)
    AllowedAAGUIDs []string `json:"allowed_aaguids"`

    // Blocked AAGUIDs (specific models banned)
    BlockedAAGUIDs []string `json:"blocked_aaguids"`

    // Authenticator attachment preference
    AttachmentPreference string `json:"attachment_preference"`
    // "any", "platform", "cross-platform"
}
```

#### Metadata-Based Enforcement

When a user registers, GGID should:

1. Extract the AAGUID from the attestation.
2. Look up the AAGUID in the FIDO Metadata Service.
3. Check the authenticator's certification level.
4. Reject or warn if below the tenant's minimum.

```go
// EnforceAuthenticatorPolicy checks a credential against tenant policy.
func EnforceAuthenticatorPolicy(
    policy *WebAuthnPolicy,
    aaguid []byte,
    attestationFormat string,
    userVerified bool,
    backupEligible bool,
    mds *MetadataService,
) error {
    // 1. Check attestation format whitelist
    if len(policy.AllowedAttestationFormats) > 0 {
        allowed := false
        for _, f := range policy.AllowedAttestationFormats {
            if f == attestationFormat {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("attestation format %s not allowed by policy", attestationFormat)
        }
    }

    // 2. Check AAGUID whitelist/blocklist
    aaguidStr := formatAAGUID(aaguid)

    for _, blocked := range policy.BlockedAAGUIDs {
        if blocked == aaguidStr {
            return fmt.Errorf("authenticator %s is blocked by policy", aaguidStr)
        }
    }

    if len(policy.AllowedAAGUIDs) > 0 {
        allowed := false
        for _, a := range policy.AllowedAAGUIDs {
            if a == aaguidStr {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("authenticator %s not in allowed list", aaguidStr)
        }
    }

    // 3. Check user verification
    if policy.RequireUserVerification && !userVerified {
        return fmt.Errorf("user verification required but not performed")
    }

    // 4. Check backup eligibility
    if policy.RequireBackupEligible && !backupEligible {
        return fmt.Errorf("backup-eligible credential required")
    }

    // 5. Check certification level via MDS
    if policy.MinAuthenticatorLevel > 0 && mds != nil {
        meta := mds.Lookup(aaguidStr)
        if meta == nil {
            // Unknown authenticator — reject if policy requires known level
            if policy.MinAuthenticatorLevel > 0 {
                return fmt.Errorf("authenticator not found in metadata service")
            }
        } else {
            certLevel := extractCertificationLevel(meta)
            if certLevel < policy.MinAuthenticatorLevel {
                return fmt.Errorf(
                    "authenticator certification level %d below required %d",
                    certLevel, policy.MinAuthenticatorLevel,
                )
            }
        }
    }

    return nil
}
```

### 5.6 Certification Level Display in Admin Console

GGID's console should display the certification level of each registered
credential:

```
┌─────────────────────────────────────────────────────────────────┐
│  WebAuthn Credentials                          [+ Add Passkey]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Name              │ Type      │ Level │ Last Used │ Actions     │
│  ─────────────────────────────────────────────────────────────  │
│  Chrome on macOS   │ Platform  │ L2    │ 2 min ago │ [Delete]   │
│  YubiKey 5 NFC     │ Roaming   │ L2    │ Yesterday │ [Delete]   │
│  Backup key        │ Roaming   │ L1    │ Never     │ [Delete]   │
│                                                                 │
│  Policy: Minimum Level L2 (configured by admin)                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. Documentation Requirements

### 6.1 Overview

FIDO Alliance requires specific documentation as part of the certification
submission. These documents demonstrate that the implementation is not only
functionally correct but also well-documented, secure, and operationally sound.

### 6.2 Required Documents

| # | Document | Purpose | Est. Effort |
|---|---------|---------|-------------|
| 1 | Server Security Documentation | Describe security architecture | 2-3 weeks |
| 2 | Cryptographic Implementation Description | Detail crypto operations | 1-2 weeks |
| 3 | Operational Procedures | Deployment, monitoring, incident response | 1-2 weeks |
| 4 | Incident Response Plan | Security incident handling | 1 week |
| 5 | Conformance Test Report | Results from Conformance Tools | Automated |
| 6 | Interoperability Test Report | Real-device testing results | 1-2 weeks |
| 7 | Bill of Materials | Software components and dependencies | 3-5 days |

### 6.3 Document Templates

#### 6.3.1 Server Security Documentation

```markdown
# FIDO2 Server Security Documentation
# GGID Identity Platform

## 1. Architecture Overview

### 1.1 System Components
- [Diagram of WebAuthn flow]
- [List of components: gateway, auth service, database, Redis]

### 1.2 Trust Boundaries
- [Network segmentation]
- [Service-to-service authentication]
- [External trust boundary (user ↔ server)]

### 1.3 Key Management
- Server does not hold user private keys
- Public keys stored encrypted in PostgreSQL
- Database encryption at rest (AES-256)
- TLS 1.3 for all external communication

## 2. WebAuthn Implementation

### 2.1 Registration Ceremony
1. Client requests challenge from /api/v1/webauthn/register/begin
2. Server generates 32-byte random challenge
3. Server returns PublicKeyCredentialCreationOptions
4. Client creates credential via navigator.credentials.create()
5. Client posts attestation to /api/v1/webauthn/register/finish
6. Server verifies attestation (per-format)
7. Server stores credential (ID, public key, counter)

### 2.2 Authentication Ceremony
1. Client requests challenge from /api/v1/webauthn/auth/begin
2. Server generates challenge
3. Server returns PublicKeyCredentialRequestOptions
4. Client gets assertion via navigator.credentials.get()
5. Client posts assertion to /api/v1/webauthn/auth/finish
6. Server verifies assertion signature
7. Server checks sign counter (clone detection)

## 3. Attestation Verification

### 3.1 Supported Formats
| Format | Algorithm | Verification Method |
|--------|-----------|-------------------|
| none | N/A | Accepted without verification |
| packed | ES256/RS256/EdDSA | Signature over authData || clientDataHash |
| fido-u2f | ECDSA P-256 | Signature over 0x00 || rpIdHash || ... |
| tpm | RSA | TPM attestation structure |
| android-key | ECDSA | Key attestation extension |
| android-safetynet | JWS | JWS verification with Google root |
| apple | ECDSA P-256 | Nonce in extension |

### 3.2 Certificate Chain Validation
- [Root CA store used]
- [Revocation checking]
- [Trust anchor management]

## 4. Metadata Service Integration
- [MDS BLOB download schedule]
- [AAGUID lookup process]
- [Policy enforcement]

## 5. Access Controls
- JWT verification on all WebAuthn endpoints
- Tenant isolation enforced
- Admin operations require admin role
```

#### 6.3.2 Cryptographic Implementation Description

```markdown
# Cryptographic Implementation Description
# GGID FIDO2 Server

## 1. Cryptographic Libraries

### 1.1 Core Library
- go-webauthn/webauthn (Go module)
- Go standard library crypto package
- No custom cryptographic implementations

### 1.2 Algorithms
| Algorithm | Use Case | Key Size | Standard |
|-----------|---------|----------|----------|
| ECDSA P-256 (ES256) | Attestation/assertion verification | 256-bit | FIPS 186-4 |
| RSA PKCS#1v1.5 (RS256) | Attestation verification | 2048-bit | PKCS#1 v2.1 |
| Ed25519 (EdDSA) | Attestation verification | 256-bit | RFC 8032 |
| SHA-256 | Hashing | 256-bit | FIPS 180-4 |
| AES-256-GCM | Database encryption | 256-bit | FIPS 197 |

## 2. Random Number Generation
- Source: crypto/rand (OS-provided CSPRNG)
- Challenge size: 32 bytes (256 bits)
- Credential IDs: authenticator-generated

## 3. Key Storage
- User credential public keys: PostgreSQL with AES-256 encryption
- No private keys stored server-side
- Database backup encryption: AES-256-CBC

## 4. Certificate Validation
- Root CA: FIDO Alliance Metadata Service root certificates
- Intermediate: per-authenticator manufacturer
- Revocation: CRL / OCSP where available
- Pinning: not used (MDS provides trust anchors)

## 5. Constant-Time Operations
- Challenge comparison: crypto/subtle.ConstantTimeCompare
- AAGUID comparison: crypto/subtle.ConstantTimeCompare
- Counter comparison: standard uint32 comparison (no timing sensitivity)
```

#### 6.3.3 Operational Procedures

```markdown
# Operational Procedures
# GGID FIDO2 Server

## 1. Deployment

### 1.1 Prerequisites
- TLS certificate for RP domain
- DNS A record for RP ID
- PostgreSQL 16+
- Redis 7+
- NATS JetStream (for audit)

### 1.2 Configuration
- WEBAUTHN_RP_ID: set to your domain
- WEBAUTHN_RP_NAME: display name
- WEBAUTHN_ORIGINS: list of allowed origins
- WEBAUTHN_TIMEOUT: 60000 (ms)
- WEBAUTHN_USER_VERIFICATION: preferred

### 1.3 Health Checks
- GET /api/v1/webauthn/register/begin (should return 400 without params)
- GET /.well-known/webauthn (should return JSON)

## 2. Monitoring

### 2.1 Metrics
- webauthn_registrations_total{tenant, success}
- webauthn_authentications_total{tenant, success}
- webauthn_attestation_failures_total{format}
- webauthn_clone_detections_total

### 2.2 Alerts
- Registration failure rate > 5%
- Authentication failure rate > 3%
- Clone detection events > 0
- Session expiry rate > 10%

## 3. Incident Response

### 3.1 Clone Detection
1. Alert fires on sign counter regression
2. Verify via audit log
3. If confirmed: invalidate credential, notify user
4. User must re-register credential

### 3.2 Attestation Failure Spike
1. Check MDS BLOB freshness
2. Check certificate root store
3. If MDS issue: re-download BLOB
4. If cert issue: update root store

### 3.3 Compromised Credential
1. Mark credential as revoked in database
2. Reject all future assertions with that credential ID
3. Notify user to re-register
4. Audit log entry for compliance
```

#### 6.3.4 Incident Response Plan

```markdown
# Incident Response Plan
# GGID FIDO2 Security Incidents

## 1. Severity Levels
| Level | Description | Response Time | Examples |
|-------|-------------|--------------|---------|
| P0 | Critical | 1 hour | Mass authentication bypass |
| P1 | High | 4 hours | Attestation verification bypass |
| P2 | Medium | 24 hours | Clone detection anomaly |
| P3 | Low | 72 hours | Individual credential issue |

## 2. Response Process
1. Detect (monitoring, user report)
2. Triage (severity assignment)
3. Contain (disable affected feature)
4. Investigate (root cause analysis)
5. Remediate (fix vulnerability)
6. Notify (affected users, FIDO Alliance if certified)
7. Post-mortem (document lessons learned)

## 3. FIDO Alliance Notification
For certified products, significant security incidents must be reported to the
FIDO Alliance within 72 hours. Report includes:
- Incident description
- Affected components
- Root cause
- Remediation status
- Impact assessment
```

### 6.4 Estimated Documentation Effort

| Document | Pages | Effort (person-weeks) | Dependencies |
|----------|-------|-----------------------|-------------|
| Server Security Documentation | 30-40 | 2-3 | Architecture finalized |
| Crypto Implementation | 15-20 | 1-2 | Crypto code reviewed |
| Operational Procedures | 20-25 | 1-2 | Production deployment |
| Incident Response Plan | 10-15 | 1 | Security team input |
| Interoperability Report | 15-20 | 1-2 | Device testing complete |
| **Total** | **90-120** | **6-10** | |

---

## 7. Certification Timeline

### 7.1 Phase Overview

The certification process spans approximately 6 months from kick-off to
certification grant, followed by ongoing maintenance.

```
Phase 1: PREPARE          Phase 2: TEST          Phase 3: SUBMIT        Phase 4: MAINTAIN
(Weeks 1-8)               (Weeks 9-12)           (Weeks 13-18)          (Weeks 19+)

├── Gap analysis          ├── Run conformance    ├── Submit application ├── Annual re-test
├── Fix P0 gaps           ├── Fix test failures  ├── FIDO review        ├── Version updates
├── Implement MDS         ├── Real device tests  ├── Clarifications     ├── Report changes
├── Audit logging         ├── Document results   ├── Certification      ├── Logo usage
├── Security docs         └── 90%+ pass rate     └── Listed on site     └── Renewal
├── Policy enforcement
└── JWT verification
```

### 7.2 Detailed Timeline

#### Phase 1: Prepare (Weeks 1-8)

| Week | Task | Owner | Deliverable |
|------|------|-------|-------------|
| 1 | Gap analysis: compare current impl vs FIDO Sec Req | Security Eng | Gap report |
| 1-2 | Fix JWT verification on WebAuthn endpoints | Backend Eng | PR merged |
| 2-3 | Complete attestation verification for all 7 formats | Backend Eng | PR merged |
| 3-4 | Implement FIDO Metadata Service integration | Backend Eng | PR merged |
| 4-5 | Add audit logging for all WebAuthn ceremonies | Backend Eng | PR merged |
| 5-6 | Implement per-tenant authenticator policy | Backend Eng | PR merged |
| 6 | Set up conformance test environment | DevOps | Staging URL |
| 7 | Write Server Security Documentation | Security Eng | Draft document |
| 7-8 | Write Crypto Implementation Description | Backend Eng | Draft document |
| 8 | Write Operational Procedures | DevOps | Draft document |

#### Phase 2: Test (Weeks 9-12)

| Week | Task | Owner | Deliverable |
|------|------|-------|-------------|
| 9 | Register server URL with FIDO Alliance | PM | Confirmation |
| 9-10 | Run Conformance Tools — Registration tests | QA Eng | Test report |
| 10 | Fix failing registration tests | Backend Eng | All P-1 to P-7 pass |
| 10-11 | Run Conformance Tools — Authentication tests | QA Eng | Test report |
| 11 | Fix failing authentication tests | Backend Eng | All F-1 to F-6 pass |
| 11-12 | Real-device interoperability testing | QA Eng | Device test report |
| 12 | Achieve ≥95% conformance pass rate | QA Eng | Final report |

#### Phase 3: Submit (Weeks 13-18)

| Week | Task | Owner | Deliverable |
|------|------|-------|-------------|
| 13 | Compile certification application | PM | Application package |
| 13-14 | Internal review of all documents | Security Eng | Approved docs |
| 14 | Submit to FIDO Alliance | PM | Submission receipt |
| 14-18 | FIDO Alliance technical review | FIDO Alliance | Review feedback |
| 15-16 | Address clarification requests | Backend Eng | Responses |
| 17-18 | Final review and certification grant | FIDO Alliance | Certificate |

#### Phase 4: Maintain (Weeks 19+, ongoing)

| Cadence | Task | Owner |
|---------|------|-------|
| Per release | Re-run conformance tests | QA Eng |
| Quarterly | Review MDS BLOB freshness | DevOps |
| Annually | Re-certification (if major version) | PM |
| As needed | Report security incidents (72h SLA) | Security Eng |

### 7.3 Gantt Chart

```
Week:  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19+
       │  │  │  │  │  │  │  │  │  │  │  │  │  │  │  │  │  │  │
P1: Gap Analysis     ████████
P1: Fix P0 Gaps      ████████████████
P1: Implement MDS         ████████
P1: Audit Logging             ████████
P1: Policy Enforce               ████████
P1: Documentation                      ████████████
P2: Conformance Setup                       ████
P2: Reg Tests                                ████████
P2: Auth Tests                                    ████████
P2: Device Tests                                       ████
P2: 95% Pass                                              ██
P3: Compile App                                              ████
P3: Submit                                                      ██
P3: FIDO Review                                                   ████████████
P3: Certified                                                            ██
P4: Maintain                                                                 ████→
```

### 7.4 Critical Path

The critical path runs through:

1. **JWT verification fix** (Weeks 1-2) — blocks conformance testing
2. **Attestation completion** (Weeks 2-3) — blocks conformance testing
3. **MDS integration** (Weeks 3-4) — blocks functional certification
4. **Conformance testing** (Weeks 9-12) — blocks submission
5. **FIDO review** (Weeks 14-18) — external dependency

Any delay in the critical path pushes the certification date. The FIDO Alliance
review (Weeks 14-18) is the least controllable element — plan for potential
clarification rounds.

---

## 8. Cost Analysis

### 8.1 FIDO Alliance Membership Fees

The FIDO Alliance offers multiple membership tiers. The relevant tiers for a
company pursuing Server Certification:

| Tier | Annual Fee | Voting Rights | Certification Access | Best For |
|------|-----------|---------------|---------------------|----------|
| **Sponsor** | $50,000 | Yes (Board) | Full | Large enterprises |
| **Contributor** | $25,000 | Yes (Working Groups) | Full | Mid-size companies |
| **Associate** | $10,000 | Limited | Full | Small companies |
| **Non-Profit/Gov** | $2,500 | Limited | Full | Non-profits, government |
| **Community** | Free | No | No certification access | Individuals |

For GGID (open-source, seeking enterprise adoption), **Contributor** ($25,000/year)
is the recommended tier. It provides full certification access, working group
participation, and the ability to influence specifications.

### 8.2 Certification Testing Fees

| Item | Cost | Notes |
|------|------|-------|
| Conformance Tools access | Included with membership | No additional fee |
| Certification application fee | $5,000-$10,000 | One-time per certification |
| Re-certification fee | $2,000-$5,000 | Per major version |
| Logo licensing | Included | With certification |

### 8.3 Engineering Effort Estimate

| Role | Allocation | Duration | Cost (loaded) |
|------|-----------|----------|---------------|
| Security Engineer (lead) | 50% | 6 months | $60,000 |
| Backend Engineer | 75% | 3 months | $45,000 |
| QA Engineer | 50% | 2 months | $15,000 |
| DevOps Engineer | 25% | 2 months | $7,500 |
| Technical Writer | 50% | 2 months | $10,000 |
| Project Manager | 15% | 6 months | $9,000 |
| **Total Engineering** | | | **$146,500** |

### 8.4 Total Cost Summary

| Cost Category | Year 1 | Year 2+ |
|--------------|--------|---------|
| FIDO Alliance membership | $25,000 | $25,000/year |
| Certification application | $7,500 | $3,500 (re-cert) |
| Engineering effort | $146,500 | $20,000 (maintenance) |
| Device testing (hardware) | $5,000 | $1,000/year |
| **Total** | **$184,000** | **$49,500/year** |

### 8.5 ROI Analysis

#### Scenario: Enterprise Deal Enabled by Certification

| Metric | Value |
|--------|-------|
| Enterprise deal size (avg) | $150,000/year |
| Deals requiring FIDO cert | 3/year |
| Revenue enabled | $450,000/year |
| Certification cost (Year 1) | $184,000 |
| **Net Year 1 benefit** | **$266,000** |
| Certification cost (Year 2+) | $49,500/year |
| **Net annual benefit** | **$400,500/year** |

The ROI is strongly positive if certification enables even a single enterprise
deal per year. The break-even point is ~1.2 enterprise deals in Year 1.

#### Scenario: No Certification (Opportunity Cost)

Without certification, GGID loses deals to certified competitors:

| Competitor | FIDO2 Certified | Customer Overlap |
|-----------|----------------|-----------------|
| Auth0/Okta | Yes | High |
| Duo | Yes | High |
| Microsoft Entra | Yes | Medium |
| Keycloak | No | Low (different market) |
| Ory | No | Medium |

Enterprise customers who mandate FIDO2 certification will not evaluate GGID.
Based on market analysis, approximately 30% of enterprise IAM RFPs include
FIDO2 certification requirements.

### 8.6 Cost Comparison with Competitors

| Vendor | Approximate Certification Investment | Certification Level |
|--------|-------------------------------------|-------------------|
| Auth0 (Okta) | $500K+ (enterprise scale) | FIDO2 Server Certified |
| Okta | $500K+ (enterprise scale) | FIDO2 Server Certified |
| Duo (Cisco) | $300K+ (authenticator + server) | L2 Authenticator + Server |
| Microsoft | $1M+ (multiple products) | Full ecosystem |
| Keycloak | $0 (not certified) | None |
| Ory | $0 (not certified) | None |
| **GGID (estimated)** | **$184K** | **Targeting FIDO2 Server** |

GGID's estimated cost is significantly lower than enterprise vendors because:

1. GGID is smaller (fewer products to certify).
2. GGID already has a functional WebAuthn implementation.
3. Open-source development reduces documentation costs (code is public).

---

## 9. Interoperability Testing

### 9.1 Beyond Conformance Tools

Conformance Tools validate protocol correctness but cannot fully replicate
real-world authenticator behavior. Interoperability testing with physical
devices is essential for production readiness and is a certification
requirement.

### 9.2 Test Device Matrix

| Device | Platform | Attestation | Transport | UV | Backup | Cost |
|--------|---------|------------|-----------|----|--------| ------|
| YubiKey 5 NFC | Cross-platform | packed | usb, nfc | PIN | No | $50 |
| YubiKey 5C NFC | Cross-platform | packed | usb, nfc | PIN | No | $50 |
| YubiKey Bio | Cross-platform | packed | usb | Fingerprint | No | $80 |
| Google Titan | Cross-platform | packed | usb, ble | No | No | $30 |
| Feitian ePass FIDO | Cross-platform | packed | usb | PIN | No | $25 |
| SoloKey V2 | Cross-platform | packed | usb, nfc | No | No | $20 |
| iPhone (Touch ID) | Platform | apple | internal | Face/Touch | Yes | $400+ |
| Mac (Touch ID) | Platform | apple | internal | Touch | Yes | $1000+ |
| Windows Hello | Platform | tpm | internal | Face/PIN | No | included |
| Android Pixel | Platform | android-key | internal | Fingerprint | Yes | $400+ |
| Android Samsung | Platform | android-key | internal | Fingerprint | Yes | $400+ |
| Chrome (Linux) | Platform | none | internal | PIN | No | $0 |

**Minimum required test set** (cost ~$255):

1. YubiKey 5 NFC (USB + NFC)
2. iPhone or Mac (Apple platform)
3. Windows machine (Windows Hello / TPM)
4. Android device (android-key attestation)

### 9.3 Test Scenarios

#### Scenario 1: Cross-Platform Authenticator Registration

```
Browser: Chrome on macOS
Authenticator: YubiKey 5 NFC
Transport: USB

1. User navigates to GGID passkey registration page
2. Server returns PublicKeyCredentialCreationOptions
3. Browser prompts: "Insert your security key"
4. User taps YubiKey
5. YubiKey LED flashes — user touches metal contact
6. Browser sends attestation to server
7. Server verifies "packed" attestation
8. Server stores credential
Expected: Registration succeeds, attestation format = "packed"
```

#### Scenario 2: Platform Authenticator Registration

```
Browser: Safari on macOS
Authenticator: Touch ID
Transport: internal

1. User navigates to GGID passkey registration page
2. Server returns PublicKeyCredentialCreationOptions
3. Browser prompts: "Use Touch ID to create a passkey"
4. User touches Touch ID sensor
5. Browser sends attestation to server
6. Server verifies "apple" attestation
7. Server stores credential
Expected: Registration succeeds, attestation format = "apple", backup_eligible = true
```

#### Scenario 3: Discoverable Credential Authentication

```
Browser: Chrome on Android
Authenticator: Android Biometric
Transport: internal

1. User navigates to GGID login page
2. Server returns PublicKeyCredentialRequestOptions (no allowCredentials)
3. Browser shows account picker (discoverable credentials)
4. User selects account and authenticates with fingerprint
5. Browser sends assertion to server
6. Server verifies assertion
7. Server identifies user from credential ID
Expected: Authentication succeeds without prior user_id
```

#### Scenario 4: Hybrid Transport (Phone as Authenticator)

```
Desktop Browser: Chrome on Windows
Authenticator: iPhone (via caBLE/hybrid)
Transport: hybrid

1. User navigates to GGID passkey registration page on Windows
2. Server returns PublicKeyCredentialCreationOptions
3. Browser shows QR code
4. User scans QR code with iPhone
5. iPhone authenticates with Face ID
6. Credential is created on iPhone, sent to desktop browser via caBLE
7. Browser sends attestation to server
8. Server verifies attestation
Expected: Registration succeeds, transport = "hybrid"
```

### 9.4 Known Interoperability Issues

| Issue | Affected Devices | Root Cause | Workaround |
|-------|-----------------|-----------|-----------|
| Empty AAGUID | Apple platform authenticators | Apple privacy by design | Accept all-zeros AAGUID |
| Transport not returned | Some Chrome versions | Browser bug | Default to "internal" |
| UV flag missing | Older YubiKey firmware | No biometric | Set UV to "preferred" |
| Sign counter always 0 | Touch ID, Face ID | Platform authenticators don't count | Don't enforce counter for counter=0 |
| `android-safetynet` deprecation | Android 13+ | Replaced by Play Integrity API | Support both formats |
| BLE pairing failures | Some BLE keys | BLE stack bugs | Recommend USB/NFC |
| Cross-origin ROR | Multiple subdomains | Origin validation | Implement /.well-known/webauthn |
| RP ID mismatch | localhost testing | RP ID must be valid domain | Use test domain or hosts file |

### 9.5 Go Test Code for Multi-Authenticator Testing

```go
// Package webauthn_interop provides integration tests for real authenticators.
// These tests require physical devices and a browser driver.
package webauthn_interop

import (
    "context"
    "testing"
    "time"
)

// AuthenticatorTestCase defines a test case for a specific authenticator.
type AuthenticatorTestCase struct {
    Name                string
    Browser             string   // "chrome", "safari", "firefox", "edge"
    Platform            string   // "macos", "windows", "android", "ios", "linux"
    AuthenticatorType   string   // "platform", "cross-platform"
    AttestationFormat   string   // expected format
    Transport           string   // expected transport
    UserVerification    bool     // expected UV
    BackupEligible      bool     // expected backup eligible
    Discoverable        bool     // supports discoverable credentials
}

// TestMatrix defines all authenticator combinations to test.
var TestMatrix = []AuthenticatorTestCase{
    {
        Name: "YubiKey 5 on Chrome/macOS (USB)",
        Browser: "chrome", Platform: "macOS",
        AuthenticatorType: "cross-platform",
        AttestationFormat: "packed",
        Transport: "usb",
        UserVerification: true,
        BackupEligible: false,
        Discoverable: true,
    },
    {
        Name: "Touch ID on Safari/macOS",
        Browser: "safari", Platform: "macOS",
        AuthenticatorType: "platform",
        AttestationFormat: "apple",
        Transport: "internal",
        UserVerification: true,
        BackupEligible: true,
        Discoverable: true,
    },
    {
        Name: "Windows Hello on Edge/Windows",
        Browser: "edge", Platform: "Windows",
        AuthenticatorType: "platform",
        AttestationFormat: "tpm",
        Transport: "internal",
        UserVerification: true,
        BackupEligible: false,
        Discoverable: true,
    },
    {
        Name: "Android Biometric on Chrome/Android",
        Browser: "chrome", Platform: "Android",
        AuthenticatorType: "platform",
        AttestationFormat: "android-key",
        Transport: "internal",
        UserVerification: true,
        BackupEligible: true,
        Discoverable: true,
    },
    {
        Name: "YubiKey 5 NFC on Chrome/Android",
        Browser: "chrome", Platform: "Android",
        AuthenticatorType: "cross-platform",
        AttestationFormat: "packed",
        Transport: "nfc",
        UserVerification: true,
        BackupEligible: false,
        Discoverable: true,
    },
    {
        Name: "Face ID on Safari/iOS",
        Browser: "safari", Platform: "iOS",
        AuthenticatorType: "platform",
        AttestationFormat: "apple",
        Transport: "internal",
        UserVerification: true,
        BackupEligible: true,
        Discoverable: true,
    },
    {
        Name: "Hybrid (iPhone as auth for Chrome/macOS)",
        Browser: "chrome", Platform: "macOS",
        AuthenticatorType: "cross-platform",
        AttestationFormat: "apple", // or "none" depending on iOS version
        Transport: "hybrid",
        UserVerification: true,
        BackupEligible: true,
        Discoverable: true,
    },
}

// TestRegistrationWithAuthenticator tests the full registration ceremony
// with a real authenticator via browser automation.
func TestRegistrationWithAuthenticator(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping interop test in short mode")
    }

    serverURL := getEnvOrDefault("WEBAUTHN_TEST_SERVER", "https://ggid.local:8080")

    for _, tc := range TestMatrix {
        tc := tc
        t.Run(tc.Name, func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
            defer cancel()

            // Step 1: Call /register/begin
            beginResp, err := callRegistrationBegin(ctx, serverURL)
            if err != nil {
                t.Fatalf("register/begin failed: %v", err)
            }

            // Step 2: Verify PublicKeyCredentialCreationOptions
            if beginResp.PublicKey.RP.ID == "" {
                t.Error("RP ID is empty")
            }
            if len(beginResp.PublicKey.User.ID) == 0 {
                t.Error("User ID is empty")
            }

            // Step 3: Use browser automation to create credential
            // (requires Selenium or similar browser driver)
            attestation, err := browserCreateCredential(ctx, tc, beginResp.PublicKey)
            if err != nil {
                t.Fatalf("browser create credential failed: %v", err)
            }

            // Step 4: Call /register/finish with attestation
            finishResp, err := callRegistrationFinish(ctx, serverURL, attestation)
            if err != nil {
                t.Fatalf("register/finish failed: %v", err)
            }

            // Step 5: Verify response
            if finishResp.Status != "registered" {
                t.Errorf("expected status 'registered', got '%s'", finishResp.Status)
            }

            // Step 6: Verify stored credential properties
            cred := getCredentialFromDB(t, finishResp.CredentialID)

            if cred.AttestationType == "" {
                t.Error("attestation type not stored")
            }

            if tc.AttestationFormat != "" && cred.AttestationFormat != tc.AttestationFormat {
                t.Errorf("expected attestation format %s, got %s",
                    tc.AttestationFormat, cred.AttestationFormat)
            }

            if cred.UserVerified != tc.UserVerification {
                t.Errorf("expected UV=%v, got %v", tc.UserVerification, cred.UserVerified)
            }
        })
    }
}

// TestAuthenticationWithAuthenticator tests the full authentication ceremony.
func TestAuthenticationWithAuthenticator(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping interop test in short mode")
    }

    serverURL := getEnvOrDefault("WEBAUTHN_TEST_SERVER", "https://ggid.local:8080")

    for _, tc := range TestMatrix {
        tc := tc
        t.Run(tc.Name, func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
            defer cancel()

            // Step 1: Register a credential first
            credentialID := registerTestCredential(t, ctx, serverURL, tc)

            // Step 2: Call /auth/begin
            beginResp, err := callAuthenticationBegin(ctx, serverURL, credentialID)
            if err != nil {
                t.Fatalf("auth/begin failed: %v", err)
            }

            // Step 3: Use browser to get assertion
            assertion, err := browserGetAssertion(ctx, tc, beginResp.PublicKey)
            if err != nil {
                t.Fatalf("browser get assertion failed: %v", err)
            }

            // Step 4: Call /auth/finish
            finishResp, err := callAuthenticationFinish(ctx, serverURL, assertion)
            if err != nil {
                t.Fatalf("auth/finish failed: %v", err)
            }

            // Step 5: Verify
            if finishResp.Status != "authenticated" {
                t.Errorf("expected status 'authenticated', got '%s'", finishResp.Status)
            }
        })
    }
}

// TestCloneDetection verifies that the server detects a cloned authenticator
// via sign counter regression.
func TestCloneDetection(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping interop test in short mode")
    }

    serverURL := getEnvOrDefault("WEBAUTHN_TEST_SERVER", "https://ggid.local:8080")

    // Register a credential
    credID := registerTestCredential(t, context.Background(), serverURL, TestMatrix[0])

    // First authentication — counter increases
    authenticateAndAssertCounter(t, serverURL, credID, 1)

    // Replay the same assertion — should be rejected
    err := replayAssertion(t, serverURL, credID)
    if err == nil {
        t.Error("expected clone detection error, but authentication succeeded")
    }
}

// TestDiscoverableCredentialLogin tests login without providing user_id.
func TestDiscoverableCredentialLogin(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping interop test in short mode")
    }

    serverURL := getEnvOrDefault("WEBAUTHN_TEST_SERVER", "https://ggid.local:8080")

    // Register a discoverable credential
    registerDiscoverableCredential(t, context.Background(), serverURL)

    // Authenticate without user_id
    resp, err := callAuthenticationBeginNoUser(context.Background(), serverURL)
    if err != nil {
        t.Fatalf("auth/begin without user_id failed: %v", err)
    }

    // The response should not contain allowCredentials (discoverable flow)
    if len(resp.PublicKey.AllowCredentials) > 0 {
        t.Error("expected empty allowCredentials for discoverable flow")
    }
}
```

### 9.6 CI/CD Integration for Conformance

```yaml
# .github/workflows/fido-conformance.yml
name: FIDO2 Conformance Tests

on:
  push:
    branches: [main]
    paths:
      - 'services/auth/internal/webauthn/**'
  pull_request:
    branches: [main]
    paths:
      - 'services/auth/internal/webauthn/**'
  schedule:
    # Run nightly conformance tests
    - cron: '0 2 * * *'

jobs:
  conformance:
    name: FIDO2 Server Conformance
    runs-on: ubuntu-latest
    timeout-minutes: 30

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Build server
        run: |
          go build -o bin/ggid-auth ./services/auth/cmd

      - name: Start test server
        run: |
          ./bin/ggid-auth &
          sleep 5
          # Verify server is up
          curl -f http://localhost:9001/healthz

      - name: Run FIDO Conformance Tools
        env:
          FIDO_TOOLS_API_KEY: ${{ secrets.FIDO_TOOLS_API_KEY }}
          FIDO_SERVER_URL: http://localhost:9001
        run: |
          # Run conformance tools (requires FIDO Alliance access)
          go run ./test/fido-conformance/ -server=$FIDO_SERVER_URL \
            -category=registration \
            -category=authentication \
            -format=json \
            -output=conformance-report.json

      - name: Check pass rate
        run: |
          PASS_RATE=$(jq '.pass_rate' conformance-report.json)
          echo "Pass rate: $PASS_RATE"
          # Require 95% pass rate
          if (( $(echo "$PASS_RATE < 0.95" | bc -l) )); then
            echo "Conformance pass rate below 95%"
            exit 1
          fi

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: fido-conformance-report
          path: conformance-report.json
```

---

## 10. GGID WebAuthn Implementation Assessment

### 10.1 Implementation Overview

GGID's WebAuthn implementation resides in `services/auth/internal/webauthn/`
and uses the `go-webauthn/webauthn` library for core protocol handling. The
implementation consists of:

| File | Purpose | Lines |
|------|---------|-------|
| `handler.go` | HTTP endpoints, session management, ceremony orchestration | ~850 |
| `attestation.go` | Attestation format dispatch, AAGUID lookup | ~185 |
| `attestation_formats.go` | Per-format verification (fido-u2f, android-key, etc.) | ~155 |
| `handler_test.go` | Unit tests | ~700 |
| `attestation_test.go` | Attestation verification tests | ~200 |
| `handler_coverage_test.go` | Coverage tests | ~500 |
| `handler_p1_test.go` | Phase 1 feature tests | ~50 |
| `handler_p2_test.go` | Phase 2 feature tests | ~300 |

### 10.2 What Passes (Conformance-Ready)

| Capability | Status | Evidence |
|-----------|--------|---------|
| Registration begin/finish endpoints | PASS | `handler.go:443-585` |
| Authentication begin/finish endpoints | PASS | `handler.go:589-752` |
| Challenge generation (32 bytes) | PASS | Delegated to go-webauthn |
| Session management (5-min expiry) | PASS | `handler.go:60-103` |
| Credential store interface | PASS | `handler.go:108-115` |
| "none" attestation | PASS | `attestation.go:26-28` |
| "packed" attestation (ES256, RS256, EdDSA) | PASS | `attestation.go:32-73` |
| AAGUID extraction from authData | PASS | `attestation.go:100-110` |
| Sign counter parsing | PASS | `attestation.go:113-118` |
| Clone detection (counter check) | PASS | `handler.go:727-738` |
| Authenticator selection (resident key, UV) | PASS | `handler.go:471-474` |
| Exclude existing credentials | PASS | `handler.go:462-468` |
| Discoverable credential support | PASS | `handler.go:639-649` |
| Allow credentials with transports | PASS | `handler.go:619-633` |
| Credential name auto-generation | PASS | `handler.go:199-238` |
| Related Origin Requests (.well-known) | PASS | `handler.go:261-275` |
| Android Digital Asset Links | PASS | `handler.go:279-301` |
| Apple App Site Association | PASS | `handler.go:305-330` |
| Error classification | PASS | `handler.go:346-373` |
| Credential list/delete endpoints | PASS | `handler.go:756-846` |

### 10.3 What Needs Work

#### P0: Critical (Blocks Conformance)

| Gap | Current State | Required | Effort |
|-----|--------------|----------|--------|
| **"fido-u2f" attestation** | Checks cert is ECDSA P-256, does NOT verify signature over `(0x00 \|\| rpIdHash \|\| clientDataHash \|\| credentialId \|\| publicKey)` | Full signature verification per FIDO U2F spec | 2-3 days |
| **"android-key" attestation** | Checks for extension OID, does NOT verify signature or parse key attestation extension payload | Full verification including challenge check in extension | 3-5 days |
| **"android-safetynet" attestation** | Checks JWS has 3 parts and x5c header, does NOT verify JWS signature or check nonce | Full JWS verification with Google root CA, nonce check | 3-5 days |
| **"tpm" attestation** | Checks cert key type and ASN.1 structure, does NOT verify COSE algorithm or TPM attestation structure | Full TPM attestation verification per spec | 5-7 days |
| **"apple" attestation** | Checks Apple extension OID and ECDSA key, does NOT verify nonce in extension | Full nonce verification: `SHA256(nonce \|\| authData)` | 2-3 days |
| **JWT verification on WebAuthn endpoints** | Accepts X-Tenant-ID + user_id without authentication | Verify JWT token, extract tenant/user from claims | 1-2 days |
| **Audit logging** | No WebAuthn-specific audit events | Log all ceremonies to NATS audit | 2-3 days |

#### P1: Important (Improves Robustness)

| Gap | Current State | Required | Effort |
|-----|--------------|----------|--------|
| **MDS integration** | Hardcoded AAGUID lookup (5 entries) | Download and parse FIDO MDS BLOB, cache, lookup | 5-7 days |
| **Per-tenant authenticator policy** | No policy enforcement | Min cert level, allowed formats, AAGUID whitelist | 3-5 days |
| **Session store in Redis** | In-memory (lost on restart) | Redis-backed session store | 1-2 days |
| **Certificate chain validation** | None (only self-attestation for packed) | Full chain validation against root CA store | 3-5 days |
| **Rate limiting on WebAuthn endpoints** | Gateway-level only | Endpoint-specific rate limiting | 1 day |
| **Extension support** | None (ignores extensions) | Handle `appid`, `credProtect` | 2-3 days |

#### P2: Nice-to-Have (For L2+ or Competitive Edge)

| Gap | Current State | Required | Effort |
|-----|--------------|----------|--------|
| **RP attestation** | Not implemented | Server signs its own attestation (for L2+) | 5-7 days |
| **Conditional UI** | Not implemented | Support `mediation: "conditional"` | 2-3 days |
| **Credential backup state tracking** | Stored but not acted upon | Enforce backup policy, monitor backup state changes | 1-2 days |
| **Multi-device credential management** | Not implemented | List/manage credentials across user devices | 3-5 days |
| **WebAuthn extensions API** | Not implemented | Expose extension configuration to admin console | 3-5 days |

### 10.4 Attestation Format Verification Status

```
┌─────────────────────────────────────────────────────────────────────┐
│              Attestation Format Verification Status                  │
├──────────────────────┬──────────┬────────────┬──────────────────────┤
│ Format               │ Parsed   │ Verified   │ Certification Ready  │
├──────────────────────┼──────────┼────────────┼──────────────────────┤
│ none                 │ N/A      │ N/A        │ YES                  │
│ packed (self)        │ YES      │ FULL       │ YES                  │
│ packed (basic)       │ YES      │ FULL       │ YES                  │
│ packed (attCA)       │ YES      │ FULL       │ YES                  │
│ fido-u2f             │ PARTIAL  │ PARTIAL    │ NO — missing sig     │
│ tpm                  │ PARTIAL  │ PARTIAL    │ NO — missing TPM str │
│ android-key          │ PARTIAL  │ PARTIAL    │ NO — missing ext val │
│ android-safetynet    │ PARTIAL  │ PARTIAL    │ NO — missing JWS val │
│ apple                │ PARTIAL  │ PARTIAL    │ NO — missing nonce   │
│ android-dot          │ NO       │ NO         │ NO — not implemented │
└──────────────────────┴──────────┴────────────┴──────────────────────┘
```

### 10.5 Library Assessment

GGID uses `github.com/go-webauthn/webauthn` which provides:

- WebAuthn ceremony orchestration (BeginRegistration, CreateCredential, etc.)
- Client data parsing and verification
- AuthData parsing
- Assertion signature verification
- Session data management

What the library does NOT provide (GGID must implement):

- Attestation format-specific verification beyond "packed" and "none"
- FIDO Metadata Service integration
- Certificate chain validation
- Audit logging
- Per-tenant policy enforcement

The library's `CreateCredential` method does verify the assertion signature and
client data, but delegates attestation verification to a callback or default
implementation. GGID's custom `VerifyAttestationFormat` function fills this gap
but is incomplete for 5 of 8 formats.

---

## 11. Certification Readiness Roadmap

### 11.1 Prioritized Action Items

#### P0: Must-Fix for Conformance (Blocks Certification)

```
┌─────────────────────────────────────────────────────────────────────┐
│  P0-1: Complete fido-u2f attestation verification                   │
│  ─────────────────────────────────────────────────────────────────  │
│  File: services/auth/internal/webauthn/attestation_formats.go       │
│  Function: verifyFidoU2FAttestation                                 │
│  Current: Checks cert is ECDSA P-256 only                           │
│  Required: Verify signature over:                                   │
│    0x00 || rpIdHash || clientDataHash || credentialId || publicKey  │
│  Effort: 2-3 days                                                   │
│  Blocks: Conformance test P-2.5                                     │
├─────────────────────────────────────────────────────────────────────┤
│  P0-2: Complete android-key attestation verification                │
│  ─────────────────────────────────────────────────────────────────  │
│  File: services/auth/internal/webauthn/attestation_formats.go       │
│  Function: verifyAndroidKeyAttestation                              │
│  Current: Checks for extension OID existence only                   │
│  Required: Parse key attestation extension, verify challenge,       │
│    check key authorization list, verify signature                   │
│  Effort: 3-5 days                                                   │
│  Blocks: Conformance test P-2.6                                     │
├─────────────────────────────────────────────────────────────────────┤
│  P0-3: Complete android-safetynet attestation verification          │
│  ─────────────────────────────────────────────────────────────────  │
│  File: services/auth/internal/webauthn/attestation_formats.go       │
│  Function: verifyAndroidSafetynetAttestation                        │
│  Current: Checks JWS has 3 parts and x5c header                     │
│  Required: Verify JWS signature with Google root CA, check nonce,   │
│    verify certificate chain, check timestamps                       │
│  Effort: 3-5 days                                                   │
│  Blocks: Conformance test P-2.7                                     │
├─────────────────────────────────────────────────────────────────────┤
│  P0-4: Complete TPM attestation verification                        │
│  ─────────────────────────────────────────────────────────────────  │
│  File: services/auth/internal/webauthn/attestation_formats.go       │
│  Function: verifyTPMAttestation                                     │
│  Current: Checks cert key type and ASN.1 structure                  │
│  Required: Parse TPM attestation structure (TPMS_ATTEST),           │
│    verify COSE algorithm, check certificate chain                   │
│  Effort: 5-7 days                                                   │
│  Blocks: Conformance test P-2.8                                     │
├─────────────────────────────────────────────────────────────────────┤
│  P0-5: Complete Apple attestation verification                      │
│  ─────────────────────────────────────────────────────────────────  │
│  File: services/auth/internal/webauthn/attestation_formats.go       │
│  Function: verifyAppleAttestation                                   │
│  Current: Checks Apple extension OID and ECDSA key                  │
│  Required: Extract nonce from extension, verify:                    │
│    SHA256(nonce || authData) matches extension value                │
│  Effort: 2-3 days                                                   │
│  Blocks: Conformance test P-2.9                                     │
├─────────────────────────────────────────────────────────────────────┤
│  P0-6: JWT verification on WebAuthn endpoints                       │
│  ─────────────────────────────────────────────────────────────────  │
│  File: services/auth/internal/webauthn/handler.go                   │
│  Function: getTenantAndUser                                         │
│  Current: Reads X-Tenant-ID header and user_id query param          │
│  Required: Verify JWT Bearer token, extract tenant/user from claims │
│  Effort: 1-2 days                                                   │
│  Blocks: Security requirement REQ-3.1                               │
├─────────────────────────────────────────────────────────────────────┤
│  P0-7: Audit logging for WebAuthn ceremonies                        │
│  ─────────────────────────────────────────────────────────────────  │
│  File: services/auth/internal/webauthn/handler.go                   │
│  Required: Publish events to NATS audit for:                        │
│    - registration_begin, registration_success, registration_failed  │
│    - authentication_begin, authentication_success, authentication_  │
│      failed                                                         │
│    - credential_deleted                                             │
│    - clone_detected                                                 │
│  Effort: 2-3 days                                                   │
│  Blocks: Security requirement REQ-2                                 │
└─────────────────────────────────────────────────────────────────────┘

Total P0 effort: ~20-30 person-days
```

#### P1: Should-Fix for Robustness

```
┌─────────────────────────────────────────────────────────────────────┐
│  P1-1: FIDO Metadata Service integration                            │
│  Required: Download MDS BLOB, parse, cache in Redis, lookup by      │
│    AAGUID, check certification status                               │
│  Effort: 5-7 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P1-2: Per-tenant authenticator policy                              │
│  Required: Min cert level, allowed formats, AAGUID whitelist/       │
│    blocklist, UV requirement, backup requirement                    │
│  Effort: 3-5 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P1-3: Redis-backed session store                                   │
│  Required: Replace in-memory map with Redis for session persistence │
│  Effort: 1-2 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P1-4: Certificate chain validation                                 │
│  Required: Build root CA store from MDS, validate cert chains       │
│  Effort: 3-5 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P1-5: Rate limiting on WebAuthn endpoints                          │
│  Required: Per-IP and per-user rate limiting                        │
│  Effort: 1 day                                                      │
├─────────────────────────────────────────────────────────────────────┤
│  P1-6: Extension support (appid, credProtect)                       │
│  Required: Parse and honor standard extensions                      │
│  Effort: 2-3 days                                                   │
└─────────────────────────────────────────────────────────────────────┘

Total P1 effort: ~15-23 person-days
```

#### P2: Nice-to-Have for L2+ or Competitive Edge

```
┌─────────────────────────────────────────────────────────────────────┐
│  P2-1: RP attestation (server-side attestation signing)             │
│  Required: Server signs its own attestation using HSM-held key      │
│  Effort: 5-7 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P2-2: Conditional UI support                                       │
│  Required: Support mediation: "conditional" for autofill UI         │
│  Effort: 2-3 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P2-3: Credential backup state tracking                             │
│  Required: Monitor backup state changes, enforce backup policy      │
│  Effort: 1-2 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P2-4: Multi-device credential management UI                        │
│  Required: Admin console for cross-device credential management     │
│  Effort: 3-5 days                                                   │
├─────────────────────────────────────────────────────────────────────┤
│  P2-5: android-dot attestation format                               │
│  Required: Implement Android 14+ attestation format                 │
│  Effort: 2-3 days                                                   │
└─────────────────────────────────────────────────────────────────────┘

Total P2 effort: ~13-20 person-days
```

### 11.2 Six-Month Plan to Certification

```
Month 1: P0 Fixes
├── Week 1-2: Complete attestation verification (P0-1 through P0-5)
├── Week 2: JWT verification (P0-6)
├── Week 3: Audit logging (P0-7)
└── Week 4: Integration testing, code review

Month 2: P1 Features
├── Week 1-2: MDS integration (P1-1)
├── Week 2-3: Per-tenant policy (P1-2)
├── Week 3: Redis session store (P1-3)
├── Week 3-4: Cert chain validation (P1-4)
└── Week 4: Rate limiting + extensions (P1-5, P1-6)

Month 3: Documentation
├── Week 1: Server Security Documentation
├── Week 2: Crypto Implementation Description
├── Week 3: Operational Procedures
└── Week 4: Incident Response Plan

Month 4: Conformance Testing
├── Week 1: Register server with FIDO Alliance
├── Week 1-2: Run registration conformance tests
├── Week 2-3: Run authentication conformance tests
├── Week 3-4: Fix failures, re-run

Month 5: Interoperability Testing
├── Week 1: Acquire test devices ($255 budget)
├── Week 1-2: Run device matrix tests
├── Week 2-3: Fix interop issues
└── Week 3-4: Compile interop report

Month 6: Submission
├── Week 1: Compile certification application
├── Week 2: Internal review
├── Week 3: Submit to FIDO Alliance
└── Week 4: Begin FIDO review (continues into Month 7-8)
```

### 11.3 Resource Allocation

| Month | Backend Eng | Security Eng | QA Eng | DevOps | PM |
|-------|------------|-------------|--------|--------|-----|
| 1 | 100% | 50% | 25% | 25% | 15% |
| 2 | 100% | 50% | 25% | 25% | 15% |
| 3 | 25% | 75% | 10% | 25% | 15% |
| 4 | 75% | 25% | 100% | 25% | 15% |
| 5 | 50% | 25% | 100% | 10% | 15% |
| 6 | 25% | 25% | 50% | 10% | 50% |

---

## 12. Gap Analysis and Recommendations

### 12.1 Gap Summary

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Certification Readiness Dashboard                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Protocol Implementation              ████████████████░░░░  80%      │
│  ├── Registration ceremony            ████████████████████  100%     │
│  ├── Authentication ceremony          ████████████████████  100%     │
│  ├── Attestation verification (7/8)   ████████████░░░░░░░  60%      │
│  └── Extensions                        ░░░░░░░░░░░░░░░░░░░   0%      │
│                                                                     │
│  Security Requirements                ████████████████░░░░  70%      │
│  ├── Key protection                   ████████████████░░░░  80%      │
│  ├── Audit logging                    ██████░░░░░░░░░░░░░  30%      │
│  ├── Auth of operations               ██████████░░░░░░░░░  50%      │
│  ├── Vulnerability mgmt               ████████████████░░░░  80%      │
│  ├── Crypto implementation            ████████████████████  100%     │
│  └── Operational security             ████████████████░░░░  80%      │
│                                                                     │
│  Metadata Service                     ██░░░░░░░░░░░░░░░░░  10%      │
│  ├── MDS BLOB download                ░░░░░░░░░░░░░░░░░░░   0%      │
│  ├── AAGUID lookup                    ████░░░░░░░░░░░░░░░  20%      │
│  └── Certification level check        ░░░░░░░░░░░░░░░░░░░   0%      │
│                                                                     │
│  Policy Enforcement                   ████░░░░░░░░░░░░░░░  20%      │
│  ├── Per-tenant authenticator policy  ░░░░░░░░░░░░░░░░░░░   0%      │
│  ├── UV enforcement                   ████░░░░░░░░░░░░░░░  20%      │
│  └── AAGUID whitelist                 ██░░░░░░░░░░░░░░░░░  10%      │
│                                                                     │
│  Documentation                        ████░░░░░░░░░░░░░░░  20%      │
│  ├── Security documentation           ██░░░░░░░░░░░░░░░░░  10%      │
│  ├── Crypto description               ░░░░░░░░░░░░░░░░░░░   0%      │
│  ├── Operational procedures           ██░░░░░░░░░░░░░░░░░  10%      │
│  └── Incident response                ██████░░░░░░░░░░░░░  30%      │
│                                                                     │
│  ─────────────────────────────────────────────────────────────────  │
│  Overall Readiness:                   ████████████░░░░░░░  53%      │
│                                                                     │
│  Estimated time to certification: 6 months                          │
│  Estimated cost (Year 1): $184,000                                  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 12.2 Critical Path Gaps

The following gaps are on the critical path and MUST be resolved first:

1. **Attestation verification completeness** (P0-1 through P0-5) — Without
   full verification of all 7 attestation formats, conformance tests will fail
   and certification cannot proceed.

2. **JWT verification** (P0-6) — The current implementation accepts unauthenticated
   requests. This is a security vulnerability that blocks REQ-3 compliance and
   must be fixed before any certification submission.

3. **Audit logging** (P0-7) — FIDO Sec Req mandates comprehensive audit logging.
   The current implementation has zero WebAuthn audit events.

4. **MDS integration** (P1-1) — Functional certification requires MDS integration.
   Without it, the submission will be rejected.

### 12.3 Recommendations

#### Immediate Actions (Next 30 Days)

1. **Start with P0-6 (JWT verification)** — This is the quickest fix (1-2 days)
   and has the highest security impact.

2. **Allocate a backend engineer full-time** to attestation verification
   (P0-1 through P0-5). This is ~15-20 days of work and is the largest P0 item.

3. **Wire audit events** (P0-7) — The audit infrastructure already exists (NATS
   JetStream + hash chain). Only the WebAuthn event publishing needs to be added.

#### Medium-Term Actions (Months 2-3)

4. **Implement MDS integration** (P1-1) — This is the most complex P1 item and
   should start early.

5. **Begin documentation** — Security documentation can be written in parallel
   with engineering work. Start with the Server Security Documentation template.

6. **Join the FIDO Alliance** — Membership is required for certification access.
   The Contributor tier ($25K/year) is recommended.

#### Long-Term Actions (Months 4-6)

7. **Run conformance tests** — After P0 fixes, run the Conformance Tools and
   iterate until ≥95% pass rate.

8. **Acquire test devices** — Purchase the minimum test set (~$255) for
   interoperability testing.

9. **Submit certification application** — Compile all documents, conformance
   report, and interop results into the application package.

#### Strategic Recommendations

10. **Position certification as a competitive differentiator** — Keycloak and
    Ory are not certified. Certification would place GGID above all open-source
    IAM alternatives.

11. **Leverage certification for enterprise deals** — Use the FIDO Certified
    logo in enterprise RFPs and sales materials.

12. **Plan for ongoing maintenance** — Budget $49,500/year for re-certification
    and maintenance after Year 1.

### 12.4 Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|-----------|
| Attestation verification harder than estimated | Medium | High | Start early, allocate buffer time |
| FIDO Alliance review takes longer than 4 weeks | Medium | Medium | Submit early, respond to clarifications promptly |
| MDS BLOB format changes | Low | Medium | Monitor FIDO Alliance announcements |
| New attestation format required | Low | Medium | Subscribe to spec updates |
| Engineering resources unavailable | Medium | High | Secure budget commitment before starting |
| Cost exceeds estimate | Low | Medium | 20% contingency in budget |

### 12.5 Final Recommendation

**GGID should pursue FIDO2 Server Certification.** The business case is strong:

- **Cost**: $184K Year 1, $49.5K/year ongoing
- **Revenue impact**: Enables enterprise deals worth $150K+ each
- **Competitive advantage**: Differentiates from Keycloak, Ory (not certified)
- **Security improvement**: Fixes critical gaps (JWT verification, audit logging)
- **Market positioning**: Matches Auth0, Okta, Duo in certification status

The certification process itself will improve GGID's code quality, security
posture, and documentation — providing value beyond the certification itself.

**Recommended next step**: Allocate a backend engineer to begin P0 fixes
(starting with JWT verification), and initiate FIDO Alliance membership
discussions.

---

## Appendix A: Go Code Examples

### A.1 Complete fido-u2f Attestation Verification

```go
// verifyFidoU2FAttestationComplete performs full FIDO U2F attestation
// verification per the FIDO U2F Raw Message Formats specification.
//
// The signed data is:
//   0x00 || rpIdHash || clientDataHash || credentialId || publicKey
//
// Where:
//   - rpIdHash = SHA256(rpId) from authData[0:32]
//   - clientDataHash = SHA256(clientDataJSON)
//   - credentialId = from authData
//   - publicKey = COSE-encoded ECDSA P-256 public key (65 bytes, uncompressed)
func verifyFidoU2FAttestationComplete(
    authData, clientDataHash, sig, certBytes, publicKeyBytes []byte,
) error {
    if len(certBytes) == 0 {
        return fmt.Errorf("fido-u2f: missing attestation certificate")
    }

    cert, err := x509.ParseCertificate(certBytes)
    if err != nil {
        return fmt.Errorf("fido-u2f: parse cert: %w", err)
    }

    // Verify certificate validity period
    if time.Now().After(cert.NotAfter) {
        return fmt.Errorf("fido-u2f: attestation cert expired")
    }

    // Verify cert is ECDSA P-256
    pubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
    if !ok {
        return fmt.Errorf("fido-u2f: cert must be ECDSA P-256")
    }

    // Check curve is P-256
    if pubKey.Curve != elliptic.P256() {
        return fmt.Errorf("fido-u2f: cert must use P-256 curve")
    }

    // Extract rpIdHash from authData
    if len(authData) < 32 {
        return fmt.Errorf("fido-u2f: authData too short")
    }
    rpIdHash := authData[:32]

    // Extract credential ID from authData
    // authData structure: rpIdHash(32) || flags(1) || signCount(4) || attData
    // attData: aaguid(16) || credIdLen(2) || credId || pubKey
    if len(authData) < 55 {
        return fmt.Errorf("fido-u2f: authData missing attested credential data")
    }
    credIdLen := binary.BigEndian.Uint16(authData[53:55])
    if len(authData) < 55+int(credIdLen) {
        return fmt.Errorf("fido-u2f: authData credential ID truncated")
    }
    credentialId := authData[55 : 55+int(credIdLen)]

    // Construct verification data
    // 0x00 || rpIdHash || clientDataHash || credentialId || publicKey
    verificationData := make([]byte, 0, 1+32+32+len(credentialId)+65)
    verificationData = append(verificationData, 0x00)
    verificationData = append(verificationData, rpIdHash...)
    verificationData = append(verificationData, clientDataHash...)
    verificationData = append(verificationData, credentialId...)
    verificationData = append(verificationData, publicKeyBytes...)

    // Hash the verification data
    hash := sha256.Sum256(verificationData)

    // Verify signature
    if !ecdsa.VerifyASN1(pubKey, hash[:], sig) {
        return fmt.Errorf("fido-u2f: signature verification failed")
    }

    return nil
}
```

### A.2 FIDO Metadata Service Client

```go
// Package fidomds provides a client for the FIDO Alliance Metadata Service.
package fidomds

import (
    "compress/gzip"
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// MetadataService downloads and caches the FIDO MDS BLOB.
type MetadataService struct {
    token     string
    baseURL   string
    cache     *MetadataCache
    client    *http.Client
}

// MetadataCache holds the parsed MDS BLOB.
type MetadataCache struct {
    BLOB       *FIDOMetadataBLOB
    UpdatedAt  time.Time
    Entries    map[string]*MetadataBLOBPayloadEntry // keyed by AAGUID
}

// FIDOMetadataBLOB is the top-level MDS BLOB structure.
type FIDOMetadataBLOB struct {
    NextUpdate time.Time                  `json:"nextUpdate"`
    LegalHeader string                    `json:"legalHeader"`
    Num        int                        `json:"no"`
    Entries    []MetadataBLOBPayloadEntry `json:"entries"`
}

// MetadataBLOBPayloadEntry describes one authenticator.
type MetadataBLOBPayloadEntry struct {
    AAGUID          string                 `json:"aaguid"`
    MetadataStatement MetadataStatement     `json:"metadataStatement"`
    StatusReports   []StatusReport          `json:"statusReports"`
    TimeOfLastUpdate time.Time             `json:"timeOfLastStatusChange"`
}

// MetadataStatement contains authenticator details.
type MetadataStatement struct {
    Description               string                 `json:"description"`
    AttestationTypes         []string               `json:"attestationTypes"`
    UserVerificationDetails  [][]string             `json:"userVerificationDetails"`
    AssertionScheme          string                 `json:"assertionScheme"`
    AuthenticationAlgorithms []int                  `json:"authenticationAlgorithms"`
    PublicKeyAlgAndEncodings []int                  `json:"publicKeyAlgAndEncodings"`
    AttestationRootCertificates []string            `json:"attestationRootCertificates"`
}

// StatusReport contains certification status.
type StatusReport struct {
    Status           string    `json:"status"`
    Certificate      string    `json:"certificate,omitempty"`
    CertificationLevel string  `json:"certificationLevel,omitempty"`
    EffectiveDate    time.Time `json:"effectiveDate"`
}

// New creates a new MetadataService client.
// token is the FIDO Alliance access token for the MDS API.
func New(token string) *MetadataService {
    return &MetadataService{
        token:   token,
        baseURL: "https://mds.fidoalliance.org",
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        cache: &MetadataCache{
            Entries: make(map[string]*MetadataBLOBPayloadEntry),
        },
    }
}

// Download fetches the latest MDS BLOB from the FIDO Alliance.
func (s *MetadataService) Download(ctx context.Context) error {
    url := fmt.Sprintf("%s/?token=%s", s.baseURL, s.token)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := s.client.Do(req)
    if err != nil {
        return fmt.Errorf("download MDS BLOB: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("MDS API returned status %d", resp.StatusCode)
    }

    // Response is a JWT containing the BLOB as payload
    // Parse JWT (simplified — production should verify signature)
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("read response: %w", err)
    }

    // Decode JWT payload
    parts := strings.Split(string(body), ".")
    if len(parts) != 3 {
        return fmt.Errorf("invalid JWT format")
    }

    payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
    if err != nil {
        return fmt.Errorf("decode JWT payload: %w", err)
    }

    var blob FIDOMetadataBLOB
    if err := json.Unmarshal(payloadBytes, &blob); err != nil {
        return fmt.Errorf("parse MDS BLOB: %w", err)
    }

    // Build cache entries
    entries := make(map[string]*MetadataBLOBPayloadEntry)
    for i := range blob.Entries {
        entry := &blob.Entries[i]
        if entry.AAGUID != "" {
            entries[strings.ToLower(entry.AAGUID)] = entry
        }
    }

    s.cache = &MetadataCache{
        BLOB:      &blob,
        UpdatedAt: time.Now(),
        Entries:   entries,
    }

    return nil
}

// Lookup retrieves authenticator metadata by AAGUID.
func (s *MetadataService) Lookup(aaguid string) *MetadataBLOBPayloadEntry {
    return s.cache.Entries[strings.ToLower(aaguid)]
}

// GetCertificationLevel returns the highest certification level for an AAGUID.
func (s *MetadataService) GetCertificationLevel(aaguid string) int {
    entry := s.Lookup(aaguid)
    if entry == nil {
        return 0
    }

    maxLevel := 0
    for _, report := range entry.StatusReports {
        switch report.CertificationLevel {
        case "L1":
            if maxLevel < 1 {
                maxLevel = 1
            }
        case "L2", "L2plus":
            if maxLevel < 2 {
                maxLevel = 2
            }
        case "L3", "L3plus":
            if maxLevel < 3 {
                maxLevel = 3
            }
        }
    }

    return maxLevel
}

// IsRevoked checks if an authenticator has been revoked.
func (s *MetadataService) IsRevoked(aaguid string) bool {
    entry := s.Lookup(aaguid)
    if entry == nil {
        return false
    }

    for _, report := range entry.StatusReports {
        if report.Status == "REVOKED" {
            return true
        }
    }

    return false
}

// NeedsUpdate checks if the cache should be refreshed.
func (s *MetadataService) NeedsUpdate() bool {
    if s.cache.BLOB == nil {
        return true
    }
    if time.Since(s.cache.UpdatedAt) > 24*time.Hour {
        return true
    }
    if time.Now().After(s.cache.BLOB.NextUpdate) {
        return true
    }
    return false
}
```

### A.3 Conformance Test Server Configuration

```go
// Package confserver provides a WebAuthn server configured specifically
// for FIDO Conformance Tools testing.
package confserver

import (
    "log"
    "net/http"
    "os"

    "github.com/ggid/ggid/services/auth/internal/webauthn"
)

// Config holds conformance test server configuration.
type Config struct {
    Port       string
    RPID       string
    RPName     string
    Origins    []string

    // For conformance testing, use a simple in-memory credential store
    // rather than connecting to PostgreSQL
    UseMemoryStore bool

    // Disable security controls that interfere with conformance testing
    DisableJWT      bool
    DisableCSRF     bool
    DisableRateLimit bool

    // Default tenant for conformance (tools don't send X-Tenant-ID)
    DefaultTenantID string
}

// DefaultConfig returns sensible defaults for conformance testing.
func DefaultConfig() *Config {
    return &Config{
        Port:       getEnv("CONF_PORT", "9001"),
        RPID:       getEnv("CONF_RP_ID", "localhost"),
        RPName:     getEnv("CONF_RP_NAME", "GGID Conformance Test Server"),
        Origins:    []string{"https://localhost", "http://localhost:3000"},
        UseMemoryStore: true,
        DisableJWT:     true,
        DisableCSRF:    true,
        DefaultTenantID: "00000000-0000-0000-0000-000000000001",
    }
}

// Start launches the conformance test server.
func Start(cfg *Config) error {
    handler, err := webauthn.NewHandler(
        cfg.RPID,
        cfg.RPName,
        nil, // nil store = in-memory mode
        webauthn.WithOrigins(cfg.Origins),
    )
    if err != nil {
        return err
    }

    mux := http.NewServeMux()
    handler.RegisterRoutes(mux)

    // Add conformance-specific endpoints
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    addr := ":" + cfg.Port
    log.Printf("FIDO Conformance Test Server listening on %s", addr)
    log.Printf("RP ID: %s", cfg.RPID)
    log.Printf("Origins: %v", cfg.Origins)

    return http.ListenAndServe(addr, mux)
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

func main() {
    cfg := DefaultConfig()
    if err := Start(cfg); err != nil {
        log.Fatal(err)
    }
}
```

---

## Appendix B: Test Configurations

### B.1 FIDO Conformance Tools Configuration File

```json
{
  "server": {
    "url": "https://ggid-conformance.example.com",
    "rp_id": "ggid-conformance.example.com",
    "rp_name": "GGID Conformance Test Server",
    "endpoints": {
      "registration_begin": "/api/v1/webauthn/register/begin",
      "registration_finish": "/api/v1/webauthn/register/finish",
      "authentication_begin": "/api/v1/webauthn/auth/begin",
      "authentication_finish": "/api/v1/webauthn/auth/finish"
    }
  },
  "test_categories": [
    "P-1",
    "P-2",
    "P-3",
    "P-4",
    "P-5",
    "P-6",
    "P-7",
    "F-1",
    "F-2",
    "F-3",
    "F-4",
    "F-5",
    "F-6",
    "A-1",
    "A-2",
    "A-3",
    "A-4",
    "A-5"
  ],
  "authenticator_types": [
    {
      "name": "Virtual Platform Authenticator",
      "attachment": "platform",
      "transport": "internal",
      "attestation": "none",
      "user_verification": true,
      "resident_key": true
    },
    {
      "name": "Virtual Roaming Authenticator (USB)",
      "attachment": "cross-platform",
      "transport": "usb",
      "attestation": "packed",
      "user_verification": false,
      "resident_key": false
    },
    {
      "name": "Virtual U2F Authenticator",
      "attachment": "cross-platform",
      "transport": "usb",
      "attestation": "fido-u2f",
      "user_verification": false,
      "resident_key": false
    },
    {
      "name": "Virtual TPM Authenticator",
      "attachment": "platform",
      "transport": "internal",
      "attestation": "tpm",
      "user_verification": true,
      "resident_key": true
    },
    {
      "name": "Virtual Android Key Authenticator",
      "attachment": "platform",
      "transport": "internal",
      "attestation": "android-key",
      "user_verification": true,
      "resident_key": true
    },
    {
      "name": "Virtual Android SafetyNet Authenticator",
      "attachment": "platform",
      "transport": "internal",
      "attestation": "android-safetynet",
      "user_verification": true,
      "resident_key": true
    },
    {
      "name": "Virtual Apple Authenticator",
      "attachment": "platform",
      "transport": "internal",
      "attestation": "apple",
      "user_verification": true,
      "resident_key": true
    }
  ],
  "algorithms": [-7, -257, -8],
  "timeout_ms": 60000
}
```

### B.2 Interoperability Test Configuration

```yaml
# test/configs/webauthn-interop.yaml
interop_test:
  server_url: "https://ggid-staging.example.com"
  admin_token: "${ADMIN_TOKEN}"

  devices:
    - name: "YubiKey 5 NFC"
      browser: "chrome"
      platform: "macOS"
      transport: "usb"
      attestation_format: "packed"
      user_verification: true

    - name: "Touch ID"
      browser: "safari"
      platform: "macOS"
      transport: "internal"
      attestation_format: "apple"
      user_verification: true
      backup_eligible: true

    - name: "Windows Hello"
      browser: "edge"
      platform: "Windows"
      transport: "internal"
      attestation_format: "tpm"
      user_verification: true

    - name: "Android Biometric"
      browser: "chrome"
      platform: "Android"
      transport: "internal"
      attestation_format: "android-key"
      user_verification: true
      backup_eligible: true

  scenarios:
    - registration
    - authentication
    - discoverable_login
    - clone_detection
    - credential_deletion
    - uv_required
    - uv_preferred
    - uv_discouraged

  assertions:
    - "registration returns 201 with credential_id"
    - "authentication returns 200 with sign_count"
    - "clone detection returns 401"
    - "credential deletion returns 200"
    - "UV required rejects non-UV assertion"
```

### B.3 CI Pipeline Configuration

```yaml
# .github/workflows/webauthn-conformance.yml
name: WebAuthn Conformance Check

on:
  push:
    paths:
      - 'services/auth/internal/webauthn/**'
      - 'test/fido-conformance/**'
  pull_request:
    paths:
      - 'services/auth/internal/webauthn/**'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Build
        run: go build ./services/auth/...

      - name: Vet
        run: go vet ./services/auth/internal/webauthn/...

      - name: Unit tests
        run: go test -v -race ./services/auth/internal/webauthn/...

      - name: Coverage
        run: |
          go test -coverprofile=coverage.out ./services/auth/internal/webauthn/...
          go tool cover -func=coverage.out | tail -1

      - name: Start conformance server
        run: |
          go run ./services/auth/cmd &
          sleep 5
          curl -f http://localhost:9001/healthz || exit 1

      - name: Run conformance tests (mock)
        run: |
          go test -v -tags=conformance ./test/fido-conformance/...
        env:
          WEBAUTHN_TEST_SERVER: "http://localhost:9001"
```

---

## Appendix C: Certification Checklist

```
FIDO2 Server Certification Checklist
=====================================

[ ] Phase 1: Preparation
    [ ] Gap analysis completed
    [ ] All attestation formats fully verified (7/7)
    [ ] JWT verification on all WebAuthn endpoints
    [ ] Audit logging wired for all ceremonies
    [ ] MDS integration implemented
    [ ] Per-tenant authenticator policy implemented
    [ ] Server Security Documentation written
    [ ] Crypto Implementation Description written
    [ ] Operational Procedures written
    [ ] Incident Response Plan written

[ ] Phase 2: Testing
    [ ] FIDO Alliance membership active
    [ ] Server URL registered with FIDO Alliance
    [ ] Conformance test environment configured
    [ ] Registration conformance tests pass (P-1 through P-7)
    [ ] Authentication conformance tests pass (F-1 through F-6)
    [ ] Edge case tests pass (A-1 through A-5)
    [ ] Overall pass rate ≥ 95%
    [ ] Interoperability tests with ≥ 4 device types
    [ ] Interoperability report compiled

[ ] Phase 3: Submission
    [ ] Certification application compiled
    [ ] All documents reviewed internally
    [ ] Application submitted to FIDO Alliance
    [ ] Clarification requests addressed
    [ ] Certification granted

[ ] Phase 4: Maintenance
    [ ] Conformance tests in CI pipeline
    [ ] MDS BLOB auto-refresh configured
    [ ] Annual re-certification scheduled
    [ ] Security incident reporting process documented
    [ ] FIDO Certified logo added to marketing materials
```

---

## References

1. **FIDO Alliance** — [https://fidoalliance.org](https://fidoalliance.org)
2. **W3C Web Authentication (WebAuthn) Level 2** — W3C Recommendation, April 2021
3. **FIDO Security Requirements** — FIDO Alliance, latest version
4. **FIDO Metadata Service 3.0** — FIDO Alliance specification
5. **CTAP 2.1** — Client-to-Authenticator Protocol, FIDO Alliance
6. **FIDO Conformance Tools** — [https://fidoalliance.org/fido-conformance-tools](https://fidoalliance.org/fido-conformance-tools)
7. **RFC 8809: COSE Algorithms** — IETF
8. **GGID WebAuthn Implementation** — `services/auth/internal/webauthn/`
9. **GGID FIDO Alliance Interoperability Report** — `docs/research/fido-alliance-interop-report.md`
10. **GGID FIDO Metadata Service Research** — `docs/research/fido-metadata-service.md`
11. **GGID WebAuthn Attestation Chain** — `docs/research/webauthn-attestation-chain.md`
12. **GGID WebAuthn Attestation Verification** — `docs/research/webauthn-attestation-verification.md`
13. **GGID Passkey/FIDO2 Deployment Guide** — `docs/research/passkey-fido2-deployment.md`

---

*Document version: 1.0*
*Last updated: 2025-01-24*
*Author: GGID Security Research*
*License: Apache 2.0*
