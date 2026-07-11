# EU Digital Identity Wallet (eIDAS 2.0) — Integration for IAM Systems

> **Research document for GGID IAM.** Covers the EU Digital Identity Wallet
> regulation (eIDAS 2.0 / Regulation 2024/1183), wallet architecture, PID,
> OpenID4VP, SIOPv2, SD-JWT VC, Status List 2021, cross-border
> interoperability, and a concrete design for GGID as a Relying Party.
> All spec references as of 2025.

---

## Table of Contents

1. [eIDAS 2.0 Regulation](#1-eidas-20-regulation)
2. [Wallet Architecture](#2-wallet-architecture)
3. [PID (Personal Identification Data)](#3-pid-personal-identification-data)
4. [OpenID4VP Integration](#4-openid4vp-integration)
5. [SIOPv2 (Self-Issued OpenID Provider)](#5-siopv2-self-issued-openid-provider)
6. [Wallet-to-RP Trust Establishment](#6-wallet-to-rp-trust-establishment)
7. [Verifiable Credentials Format](#7-verifiable-credentials-format)
8. [Status List 2021 (Revocation)](#8-status-list-2021-revocation)
9. [Cross-Border Interoperability](#9-cross-border-interoperability)
10. [GGID as Relying Party Design](#10-ggid-as-relying-party-design)
11. [Gap Analysis and Recommendations](#11-gap-analysis-and-recommendations)
12. [References](#references)

---

## 1. eIDAS 2.0 Regulation

### 1.1 Legislative Background

The **European Digital Identity Regulation**, informally known as **eIDAS 2.0**,
was published as **Regulation (EU) 2024/1183** on 11 April 2024. It amends
and significantly extends the original eIDAS Regulation (EU  No 910/2014)
which established the legal framework for electronic identification, trust
services, and electronic transactions in the EU's internal market.

eIDAS 1.0 (2014) created the regulatory foundation for:
- Mutual recognition of national eID schemes across borders
- Qualified trust services (eSignatures, eSeals, time stamps)
- Legal equivalence of electronic signatures to handwritten ones

However, adoption was uneven. Only 15 of 27 member states notified
eID schemes, and cross-border usage was limited. The European
Commission estimated that only 60% of EU citizens had access to a
national eID, and cross-border eID authentication volumes remained
negligible.

eIDAS 2.0 addresses these shortcomings by mandating a unified wallet
approach, setting explicit deadlines, and introducing Person
Identification Data (PID) as a standardized, legally recognized
identity attribute set.

### 1.2 Key Provisions

#### 1.2.1 Mandatory EU Digital Identity Wallet

Article 5a requires every member state to offer an **EU Digital Identity
Wallet** to all citizens, residents, and businesses within its territory.
The deadline is structured in two phases:

| Milestone | Deadline | Requirement |
|-----------|----------|-------------|
| Notification | 2025-Q3 | Each MS notifies the Commission of its wallet scheme |
| Provision | **End of 2026** | Wallets must be available to citizens |
| Interoperability | 2027+ | Cross-border full operability validated through conformance |

Member states may develop their own wallet app (e.g., Germany's *BundID
Wallet*, France's *France Identite*) or adopt the EUDI Reference
Implementation. Both paths must comply with the Architecture Reference
Framework (ARF) published by the Commission.

#### 1.2.2 Personal Identification Data (PID)

eIDAS 2.0 introduces **PID** as a regulated, standardized credential that
replaces the patchwork of national identity cards in digital interactions.
PID is issued by a **PID Provider** — typically the national government
authority responsible for civil registry or identity documents.

The minimum PID attribute set (Annex VII) includes:

| Attribute | Mandatory | Notes |
|-----------|-----------|-------|
| `family_name` | Yes | As recorded in the civil registry |
| `given_name` | Yes | As recorded in the civil registry |
| `birth_date` | Yes | ISO 8601 date |
| `age_birth_year` | Conditional | Required if `age_over_18` not derivable |
| `age_over_18` | Conditional | Boolean; sufficient for age verification |
| `family_name_birth` | Optional | Maiden name / birth family name |
| `given_name_birth` | Optional | Birth given name |
| `place_of_birth` | Optional | City, country, state/province |
| `current_address` | Optional | Street, city, postal code, country |
| `nationality` | Yes | ISO 3166-1 alpha-2 country code(s) |
| `personal_number` | Conditional | National identifier (varies by country) |
| `sex` | Optional | Per ISO/IEC 5218 |
| `email_address` | Optional | If available from the trusted source |
| `mobile_phone_number` | Optional | If available |

The PID carries a **Qualified Electronic Seal** from the PID Provider,
giving it the legal standing of a qualified trust service under eIDAS 2.0.

#### 1.2.3 Qualified Trust Service Providers (QTSP)

QTSPs are entities that provide qualified trust services and are supervised
by a national supervisory body. Under eIDAS 2.0, QTSPs play a central role:

- **PID Issuance**: Some member states designate QTSPs (notably national
  mint/security printing offices like *Bundesdruckerei* in Germany) as
  PID Providers.
- **Qualified Electronic Attestations of Attributes (QEAA)**: QTSPs issue
  qualified credentials beyond PID — university diplomas, professional
  licenses, bank account confirmations.
- **Wallet Remote Signing**: The wallet uses a QTSP-issued qualified
  signature creation device (QSCD) for remote qualified eSignatures.

Each QTSP is listed in the **EU Trust List** (EUTL), a machine-readable
registry published by each member state and aggregated by the Commission.
Relying parties use the EUTL to establish trust chains for credential
verification.

#### 1.2.4 Relying Party Registration

A major innovation in eIDAS 2.0 is the **mandatory RP registration
framework**. Under Article 5b, every relying party that wishes to request
PID or other credentials from the wallet must:

1. **Register** with its national supervisory body (or a designated
   accreditation authority).
2. **Identify itself** with verifiable RP metadata: legal name, registration
   number, sector, purpose of data access.
3. **Obtain a certificate** or digital attestation proving registration,
   which the wallet verifies before presenting any credential.

This mechanism serves two critical purposes:
- **Anti-phishing**: The wallet displays the verified RP identity to the
  user, making it visually obvious when a fraudulent app is masquerading.
- **Accountability**: Relying parties that misuse personal data can be
  traced through their registration. This creates a legal audit trail.

Registered RPs are published in a **RP Registry** accessible to wallets.
Each registry entry includes the RP's public key (or certificate chain)
for mutual TLS or signed authorization request verification.

#### 1.2.5 Legal Certainty and Cross-Border Recognition

Article 6 establishes the principle that PID and QEAAs presented from any
member state's wallet must be recognized by relying parties in all other
member states. This is the cornerstone of cross-border digital identity
in the EU.

Key legal effects:
- **Same legal value as physical documents**: A PID presented digitally
  from the wallet has the same legal standing as a physical national
  identity card.
- **Non-discrimination**: Service providers cannot refuse wallet-based
  authentication if they accept national eID schemes.
- **Liability framework**: Relying parties that rely on a validly
  presented PID are protected from liability for the accuracy of the
  underlying data (the PID Provider bears this responsibility).

### 1.3 Relationship to GDPR

eIDAS 2.0 is designed to be **GDPR-compatible by default**:

| Principle | eIDAS 2.0 Mechanism |
|-----------|---------------------|
| Data minimization | PID supports selective disclosure via SD-JWT |
| Purpose limitation | RP must declare purpose during registration; wallet displays it |
| Storage limitation | RP cannot retain data without user consent (`intent_to_retain`) |
| User consent | Wallet displays what data is being shared before presentation |
| Interoperability | Standardized PID format across all member states |

The wallet acts as a **privacy-enhancing intermediary**: the user sees
exactly what data flows to which RP, and can withhold non-essential
attributes through selective disclosure.

### 1.4 Regulatory Timeline

```
2024 Q2   ─── eIDAS 2.0 published (Regulation 2024/1183)
2024 Q3   ─── ARF v1.0 published (Architecture Reference Framework)
2025 Q3   ─── Member states notify Commission of wallet schemes
2025 Q4   ─── Conformance testing infrastructure operational
2026 H2   ─── Wallets available to citizens (hard deadline)
2027      ─── Cross-border interoperability validated
2028+     ─── Enterprise adoption expected to accelerate
```

### 1.5 Impact on IAM Vendors

For identity and access management platforms like GGID:

1. **Relying Party integration** is the primary use case — organizations
   will need to accept wallet-based authentication alongside passwords,
   SSO, and WebAuthn.

2. **OID4VP is the transport protocol** — the EUDI Wallet ecosystem
   standardizes on OpenID for Verifiable Presentations for
   wallet-to-RP communication.

3. **SD-JWT VC is the credential format** — the EU ARF mandates SD-JWT
   for PID and QEAA, not JSON-LD.

4. **Status List 2021 is the revocation mechanism** — bitstring-based
   status lists replace per-credential OCSP-style checks.

5. **RP registration is mandatory** — IAM vendors must support
   presenting RP certificates and metadata to the wallet for
   trust establishment.

---

## 2. Wallet Architecture

### 2.1 EUDI Wallet Reference Implementation

The European Commission funded an **open-source reference implementation**
of the EUDI Wallet, maintained in the `eu-digital-identity-wallet`
GitHub organization. The reference implementation (written in Kotlin
for Android and Swift for iOS) demonstrates all core wallet functions
and serves as a compliance template for national implementations.

The reference wallet supports:
- PID storage and presentation (SD-JWT format)
- QEAA issuance and management
- OpenID4VP as both verifier and wallet role
- SIOPv2 for self-issued authentication
- Status List 2021 status checking
- Remote qualified eSignature (QSCD)
- BLE proximity-based presentation (ISO 18013-5 inspired)

### 2.2 Core Components

```
┌─────────────────────────────────────────────────────────────────────┐
│                    EU Digital Identity Wallet Ecosystem             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────┐    ┌──────────────────┐   ┌────────────────┐ │
│  │   Wallet App     │    │   PID Provider   │   │     QTSP       │ │
│  │   (Holder)       │    │  (Government)    │   │ (Trust Service)│ │
│  │                  │    │                  │   │                │ │
│  │  - Secure Enclave│    │  - Civil Registry│   │ - QEAA Issuance│ │
│  │  - Credential    │◄───┤  2. Issues PID   │   │ - Cert Issuance│ │
│  │    Store         │    │    (SD-JWT VC)   │   │ - QSCD Access  │ │
│  │  - Presentation  │    │                  │   │                │ │
│  │    Engine        │    └──────────────────┘   └────────────────┘ │
│  │  - User Consent  │           1. User requests PID                  │
│  │  - Status Check  │                                                │
│  └────────┬─────────┘                                                │
│           │                                                          │
│           │ 3. Presentation Request / Response                       │
│           │   (OpenID4VP or SIOPv2)                                  │
│           │                                                          │
│           ▼                                                          │
│  ┌──────────────────┐          ┌────────────────────────────────┐   │
│  │  Relying Party   │          │    RP Registry                 │   │
│  │  (e.g., GGID)    │◄─────────┤    (National Authority)        │   │
│  │                  │  4. Verify│                                │   │
│  │  - Auth Request  │  RP cert  │  - Registered RP List          │   │
│  │  - VP Verifier   │          │  - RP Certificates             │   │
│  │  - Session Mgmt  │          │  - Trust Chain                 │   │
│  └──────────────────┘          └────────────────────────────────┘   │
│           │                                                          │
│           │ 5. Status Check (cached)                                 │
│           ▼                                                          │
│  ┌──────────────────┐                                                │
│  │  Status List     │                                                │
│  │  (Issuer-hosted) │                                                │
│  │  Bitstring (TSL) │                                                │
│  └──────────────────┘                                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.3 Detailed Component Responsibilities

#### Wallet App (Holder Device)

The wallet runs on the user's mobile device and serves as the holder
of verifiable credentials. Key subsystems:

| Subsystem | Responsibility |
|-----------|----------------|
| Secure Storage | Credential store backed by platform secure enclave (iOS Keychain, Android Keystore) |
| Key Management | Holder binding keys — ECDSA P-256 or Ed25519 for holder proof-of-possession |
| Presentation Engine | Constructs Verifiable Presentations per OID4VP; applies selective disclosure |
| Consent UI | Displays RP identity, requested attributes, and purpose; requires biometric unlock |
| Status Checker | Fetches and caches Status List bitstrings for credential validation |
| QR Scanner | Parses credential offers and authorization requests from QR codes |
| BLE Transport | Optional proximity-based presentation for privacy-preserving cross-device flows |

#### PID Provider (Trusted Source)

The PID Provider is typically the national civil registry or identity
authority. It:

1. Authenticates the citizen (in-person, video verification, or national
   eID scheme).
2. Issues a PID credential in SD-JWT VC format, signed with the
   provider's qualified seal.
3. Maintains a Status List for PID revocation (lost device, identity
   fraud, death).
4. Publishes its public keys via the EU Trust List and `.well-known`
   OID4VCI metadata.

#### QTSP (Qualified Trust Service Provider)

QTSPs extend the wallet ecosystem beyond government-issued PID:

- **QEAA issuance**: Professional licenses, educational credentials,
  bank account attestations, age verifications.
- **Qualified eSignatures**: The wallet can invoke a remote QSCD to
  sign documents with legal equivalence to handwritten signatures.
- **Trust anchoring**: QTSP certificates are anchored in the EUTL,
  providing the root of trust for credential verification.

#### Relying Party (Verifier)

The RP requests credentials from the wallet and verifies them. In this
document, GGID acts as the RP. Key RP functions:

1. Generate authorization requests (OID4VP / SIOPv2).
2. Present RP identity via registered certificate.
3. Verify received Verifiable Presentations (signature, trust, status).
4. Map credential claims to application user identity.
5. Create authentication sessions for verified users.

### 2.4 Communication Protocols

The EUDI Wallet ecosystem uses three primary communication patterns:

```
 Protocol        Direction             Use Case
─────────────── ───────────────────── ──────────────────────────
 OID4VCI         Issuer → Wallet       Credential issuance
 OID4VP          RP ↔ Wallet           Credential presentation
 SIOPv2          Wallet → RP           Self-issued authentication
```

**OpenID4VP** is the workhorse for RP-to-wallet interactions. The RP
sends a signed authorization request containing a
`presentation_definition` that specifies which credentials and attributes
it needs. The wallet responds with a Verifiable Presentation containing
the requested credentials.

**SIOPv2** complements OID4VP for cases where the RP only needs to
authenticate the user (no specific credentials required). The wallet
acts as its own OpenID Provider and issues an ID token directly. This
is useful for "sign in with wallet" scenarios.

### 2.5 Transport Modes

| Mode | Device Relationship | Mechanism | Privacy |
|------|---------------------|-----------|---------|
| Same-device | Wallet and RP on same device | App switch / deep link | Moderate |
| Cross-device QR | Desktop RP, mobile wallet | QR code encodes request URI | Low (session linking) |
| Cross-device BLE | Desktop RP, mobile wallet | BLE proximity + encrypted channel | High (no server relay) |
| Digital Credentials API | Browser RP, platform wallet | W3C `navigator.credentials.get()` | High (browser-mediated) |

---

## 3. PID (Personal Identification Data)

### 3.1 What PID Contains

PID is the standardized, government-issued identity credential that
every EUDI Wallet must support. Unlike ad-hoc verifiable credentials
(which any issuer can create), PID has a legally mandated attribute
set defined in Annex VII of eIDAS 2.0.

The full PID attribute set:

| Attribute | Type | Mandatory | Description |
|-----------|------|-----------|-------------|
| `family_name` | string | Yes | Current family name (surname) |
| `given_name` | string | Yes | Current given name (first name) |
| `birth_date` | string | Yes | ISO 8601 full date of birth |
| `age_over_18` | bool | Conditional | Sufficient for adult-only services |
| `age_birth_year` | int | Conditional | Year of birth (4-digit) |
| `family_name_birth` | string | Optional | Family name at birth |
| `given_name_birth` | string | Optional | Given name at birth |
| `place_of_birth` | object | Optional | { locality, region, country } |
| `current_address` | object | Optional | { formatted, street, locality, postal_code, country } |
| `nationality` | string[] | Yes | ISO 3166-1 alpha-2 codes |
| `personal_number` | string | Conditional | National ID number |
| `sex` | int | Optional | ISO/IEC 5218 (0=unknown, 1=male, 2=female, 9=not applicable) |
| `email_address` | string | Optional | Verified email |
| `mobile_phone_number` | string | Optional | Verified phone (E.164) |

The `age_over_18` attribute deserves special attention. The EU ARF
recommends that wallets compute `age_over_18` (and other age thresholds)
from `birth_date` at presentation time, avoiding the need to disclose
the actual date of birth. This is a **data minimization** feature: an
RP that only needs to verify adulthood receives a boolean, not a date.

### 3.2 PID Issuance Flow

```
 Citizen                    PID Provider              Wallet
   │                            │                         │
   │ 1. Identity proofing       │                         │
   │  (in-person / video / eID) │                         │
   ├───────────────────────────>│                         │
   │                            │                         │
   │ 2. Credential offer        │                         │
   │<───────────────────────────┤                         │
   │                            │                         │
   │ 3. Open wallet, scan offer │                         │
   ├────────────────────────────────────────────────────>│
   │                            │                         │
   │                            │ 4. OID4VCI token req    │
   │                            │<────────────────────────┤
   │                            │ 5. access_token         │
   │                            │────────────────────────>│
   │                            │                         │
   │                            │ 6. POST /credential     │
   │                            │   (with proof of key)   │
   │                            │<────────────────────────┤
   │                            │ 7. PID (SD-JWT VC)      │
   │                            │────────────────────────>│
   │                            │                         │
   │ 8. PID stored in wallet    │                         │
   │<─────────────────────────────────────────────────────│
   │                            │                         │
```

The PID Provider authenticates the citizen through a **trusted source**
process — typically the national civil registry verification or an
in-person identity check at a government office. Once identity is
confirmed, the provider issues the PID credential using OID4VCI
(pre-authorized code flow is common for in-person issuance).

### 3.3 PID Storage in Wallet

PID is stored as an SD-JWT VC credential. The wallet's secure storage
ensures:

- **Confidentiality**: Credential data is encrypted at rest using a key
  in the device's secure enclave (iOS SEP / Android StrongBox).
- **Integrity**: The SD-JWT signature prevents tampering.
- **Holder binding**: The credential contains a `cnf` claim linking it
  to a key the wallet controls, preventing export-and-replay attacks.

### 3.4 Data Minimization via Selective Disclosure

The SD-JWT format enables **per-attribute selective disclosure**. When
an RP requests `family_name` and `age_over_18`, the wallet reveals only
those two attributes from a PID that contains 14+ fields.

Example: A bar verifying age:

```
PID in wallet (14 attributes):
  family_name, given_name, birth_date, place_of_birth, nationality,
  personal_number, current_address, email, phone, ...

RP request (presentation_definition):
  Input descriptor: PID
  Constraints: [ $.age_over_18 ]   ← only this field

Wallet reveals:
  age_over_18: true                ← nothing else
  + SD-JWT proof of integrity
  + Holder binding proof
```

The RP never sees the birth date, address, national ID number, or any
other attribute. The SD-JWT hash mechanism (see Section 7) guarantees
that the disclosed `age_over_18` value was indeed signed by the PID
Provider.

### 3.5 Go Data Model for PID

```go
// Package pid defines the Personal Identification Data (PID) data model
// per eIDAS 2.0 Annex VII.
package pid

import "time"

// PlaceOfBirth represents the place of birth as specified in the PID.
type PlaceOfBirth struct {
	Locality string `json:"locality,omitempty"`
	Region   string `json:"region,omitempty"`
	Country  string `json:"country,omitempty"`
}

// Address represents the current address in the PID.
type Address struct {
	Formatted  string `json:"formatted,omitempty"`
	Street     string `json:"street_address,omitempty"`
	Locality   string `json:"locality,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

// PID represents the Personal Identification Data credential payload.
// This is the credentialSubject of the SD-JWT VC.
type PID struct {
	// Mandatory attributes
	FamilyName  string   `json:"family_name"`
	GivenName   string   `json:"given_name"`
	BirthDate   string   `json:"birth_date"` // ISO 8601 date
	Nationality []string `json:"nationality"` // ISO 3166-1 alpha-2

	// Conditional attributes
	AgeOver18     bool `json:"age_over_18,omitempty"`
	AgeBirthYear  int  `json:"age_birth_year,omitempty"`
	PersonalNumber string `json:"personal_number,omitempty"`

	// Optional attributes
	FamilyNameBirth  string  `json:"family_name_birth,omitempty"`
	GivenNameBirth   string  `json:"given_name_birth,omitempty"`
	PlaceOfBirth     *PlaceOfBirth `json:"place_of_birth,omitempty"`
	CurrentAddress   *Address `json:"current_address,omitempty"`
	Sex              int     `json:"sex,omitempty"`
	EmailAddress     string  `json:"email_address,omitempty"`
	MobilePhoneNumber string `json:"mobile_phone_number,omitempty"`
}

// PIDProviderInfo describes the issuing PID Provider.
type PIDProviderInfo struct {
	ID          string `json:"iss"`          // PID provider identifier (URL)
	Name        string `json:"name"`         // Human-readable name
	Country     string `json:"country"`      // ISO 3166-1 alpha-2
	OrganizationIdentifier string `json:"organization_id,omitempty"`
}

// PIDCredential wraps the full SD-JWT VC PID credential.
type PIDCredential struct {
	Issuer         PIDProviderInfo `json:"iss"`
	Subject        string          `json:"sub"`         // holder DID or key hash
	IssuedAt       time.Time       `json:"iat"`
	ExpiresAt      time.Time       `json:"exp"`
	StatusIndex    int             `json:"status_idx"`  // index in Status List
	PID            PID             `json:"pid"`         // selective-disclosure wrapped
	Confirmation   *KeyInfo        `json:"cnf"`         // holder binding key
}

// KeyInfo describes the holder's key for holder binding (cnf claim).
type KeyInfo struct {
	JWK map[string]any `json:"jwk"` // JSON Web Key
}

// PresentationRequest defines what PID attributes an RP needs.
type PresentationRequest struct {
	DefinitionID   string   `json:"definition_id"`
	RequestedFields []string `json:"requested_fields"` // e.g., ["family_name", "age_over_18"]
	Purpose        string   `json:"purpose"`
}

// VerifyAgeOver checks if the PID holder meets the age threshold.
// Uses age_over_18 if available; falls back to birth_date calculation.
func (p *PID) VerifyAgeOver(threshold int) bool {
	if threshold == 18 && p.AgeOver18 {
		return p.AgeOver18
	}
	birthDate, err := time.Parse("2006-01-02", p.BirthDate)
	if err != nil {
		return false
	}
	now := time.Now()
	age := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		age--
	}
	return age >= threshold
}

// SelectAttributes returns only the requested attributes as a map,
// suitable for selective disclosure presentation.
func (p *PID) SelectAttributes(requested []string) map[string]any {
	all := map[string]any{
		"family_name":         p.FamilyName,
		"given_name":          p.GivenName,
		"birth_date":          p.BirthDate,
		"nationality":         p.Nationality,
		"age_over_18":         p.AgeOver18,
		"personal_number":     p.PersonalNumber,
		"email_address":       p.EmailAddress,
		"mobile_phone_number": p.MobilePhoneNumber,
	}

	result := make(map[string]any, len(requested))
	for _, field := range requested {
		if val, ok := all[field]; ok {
			result[field] = val
		}
	}
	return result
}
```

### 3.6 PID Verification Rules

When verifying a PID, the RP must check:

1. **Issuer trust**: The PID was issued by a recognized PID Provider
   listed in the EU Trust List.
2. **Signature validity**: The SD-JWT signature verifies against the
   PID Provider's public key.
3. **Status**: The PID's status index in the Status List is 0 (not revoked).
4. **Expiry**: `exp` claim is in the future.
5. **Holder binding**: The holder proof-of-possession in the Verifiable
   Presentation matches the `cnf` key in the PID.
6. **Nonce**: The presentation proof includes a nonce generated by the
   RP to prevent replay.

---

## 4. OpenID4VP Integration

### 4.1 Protocol Overview

OpenID for Verifiable Presentations (OID4VP) is the protocol that the
EUDI Wallet ecosystem uses for RP-to-wallet credential exchange. It is
an extension of OAuth 2.0 where the "authorization response" is a
Verifiable Presentation token rather than an authorization code.

The core flow:

```
 Relying Party (GGID)                Wallet
      │                                │
      │  1. Generate presentation      │
      │     definition                 │
      │                                │
      │  2. Build authorization req    │
      │     (JWT-Signed, with          │
      │     presentation_definition)   │
      │                                │
      │  3. Send request               │
      ├───────────────────────────────>│
      │     (QR code / deep link /     │
      │      Digital Credentials API)  │
      │                                │
      │                     4. Parse &  │
      │                        verify   │
      │                        RP cert  │
      │                                │
      │                     5. Display  │
      │                        consent  │
      │                        (RP name,│
      │                        attrs)   │
      │                                │
      │                     6. User     │
      │                        approves │
      │                        (biom.)  │
      │                                │
      │  7. VP Token response          │
      │<───────────────────────────────┤
      │     (SD-JWT VC + holder proof) │
      │                                │
      │  8. Verify signature           │
      │  9. Check issuer trust         │
      │ 10. Check status list          │
      │ 11. Create session             │
      │                                │
```

### 4.2 Presentation Definition

The presentation definition tells the wallet what credentials the RP
needs. It uses the DIF Presentation Exchange format.

Example: GGID requesting PID for user registration:

```json
{
  "presentation_definition": {
    "id": "ggid-pid-verification",
    "input_descriptors": [
      {
        "id": "eu-pid",
        "format": {
          "dc+sd-jwt": {
            "sd-jwt_alg_values": ["ES256", "RS256"],
            "kb-jwt_alg_values": ["ES256"]
          }
        },
        "constraints": {
          "fields": [
            {
              "path": ["$.family_name"],
              "purpose": "Identity verification for account creation"
            },
            {
              "path": ["$.given_name"],
              "purpose": "Identity verification for account creation"
            },
            {
              "path": ["$.age_over_18"],
              "purpose": "Verify user is an adult",
              "intent_to_retain": false
            }
          ],
          "limit_disclosure": "required"
        }
      }
    ]
  }
}
```

Key fields:
- `format.dc+sd-jwt`: Specifies SD-JWT VC format with supported algorithms.
- `constraints.limit_disclosure: "required"`: Forces selective disclosure —
  the wallet must NOT reveal attributes beyond what is listed in `fields`.
- `intent_to_retain`: Whether the RP will store the data (false = transient
  verification only).

### 4.3 Authorization Request

The RP sends a signed authorization request. Large requests use
`request_uri` (a URL pointing to the full request as a JWT) to keep
QR codes short.

```
GET openid4vp://authorize?
  response_type=vp_token
  &client_id=https://ggid.example.com
  &redirect_uri=https://ggid.example.com/wallet/callback
  &request_uri=https://ggid.example.com/oid4vp/request/jwt-a1b2c3d4
```

The JWT at `request_uri` contains:

```json
{
  "iss": "https://ggid.example.com",
  "aud": "https://wallet.example",
  "response_type": "vp_token",
  "response_uri": "https://ggid.example.com/oid4vp/response",
  "response_mode": "direct_post",
  "client_id": "https://ggid.example.com",
  "client_metadata": {
    "vp_formats_supported": {
      "dc+sd-jwt": {
        "sd-jwt_alg_values": ["ES256", "RS256"]
      }
    }
  },
  "nonce": "n-47s8d2j3kl4h5j6",
  "presentation_definition": { ... },
  "exp": 1735689600
}
```

The request is signed with the RP's registered key (from RP Registry).
The wallet verifies this signature before showing consent to the user.

### 4.4 VP Token Response

The wallet responds with a VP Token containing the Verifiable Presentation.
Using the `direct_post` response mode, the wallet POSTs directly to the
RP's `response_uri`:

```json
{
  "vp_token": "<SD-JWT VC with holder binding>",
  "presentation_submission": {
    "definition_id": "ggid-pid-verification",
    "id": "submission-001",
    "descriptor_map": [
      {
        "id": "eu-pid",
        "format": "dc+sd-jwt",
        "path": "$"
      }
    ]
  }
}
```

The `vp_token` is an SD-JWT VC with:
1. The PID credential (disclosing only the requested attributes).
2. A Key Binding JWT (KB-JWT) proving the holder controls the key
   referenced in the PID's `cnf` claim, and including the RP's nonce.

### 4.5 Go Code: OpenID4VP Verifier

```go
// Package oid4vp implements the Relying Party side of OpenID for
// Verifiable Presentations (OID4VP) for EU Digital Identity Wallet.
package oid4vp

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// PresentationRequest describes what the RP needs from the wallet.
type PresentationRequest struct {
	DefinitionID    string            `json:"definition_id"`
	InputDescriptors []InputDescriptor `json:"input_descriptors"`
	Nonce           string            `json:"nonce"`
	ResponseURI     string            `json:"response_uri"`
}

// InputDescriptor specifies a credential constraint.
type InputDescriptor struct {
	ID         string                 `json:"id"`
	Format     map[string]any         `json:"format"`
	Constraints InputConstraints      `json:"constraints"`
}

// InputConstraints defines field-level requirements.
type InputConstraints struct {
	Fields          []FieldConstraint `json:"fields"`
	LimitDisclosure string            `json:"limit_disclosure,omitempty"`
}

// FieldConstraint maps a JSONPath to a filter and purpose.
type FieldConstraint struct {
	Path           string         `json:"path"`
	Purpose        string         `json:"purpose,omitempty"`
	Filter         map[string]any `json:"filter,omitempty"`
	IntentToRetain bool           `json:"intent_to_retain,omitempty"`
}

// VPTokenResponse is the wallet's response.
type VPTokenResponse struct {
	VPToken              string                `json:"vp_token"`
	PresentationSubmission PresentationSubmission `json:"presentation_submission"`
}

// PresentationSubmission maps descriptors to presented credentials.
type PresentationSubmission struct {
	DefinitionID string           `json:"definition_id"`
	ID           string           `json:"id"`
	DescriptorMap []DescriptorMap `json:"descriptor_map"`
}

// DescriptorMap links an input descriptor to a path in the VP token.
type DescriptorMap struct {
	ID     string `json:"id"`
	Format string `json:"format"`
	Path   string `json:"path"`
}

// Verifier is the RP-side OID4VP verifier.
type Verifier struct {
	issuer         string
	rpPrivateKey   interface{} // RP signing key for authz requests
	rpKeyID        string
	trustedIssuers map[string]*IssuerMetadata // keyed by issuer URL
	httpClient     *http.Client
}

// IssuerMetadata holds the public keys and metadata for a trusted issuer.
type IssuerMetadata struct {
	IssuerURL   string
	JWKSURI     string
	PublicKeys  map[string]interface{} // kid -> public key
	Country     string
	TrustStatus string // "qualified", "trusted", "untrusted"
}

// VerificationResult is the output of verifying a VP token.
type VerificationResult struct {
	Issuer      string            `json:"issuer"`
	IssuerName  string            `json:"issuer_name"`
	HolderKey   string            `json:"holder_key"`
	Claims      map[string]any    `json:"claims"`
	ExpiresAt   time.Time         `json:"expires_at"`
	StatusIndex int               `json:"status_index"`
	StatusURI   string            `json:"status_uri"`
	IsValid     bool              `json:"is_valid"`
}

// NewVerifier creates an OID4VP verifier.
func NewVerifier(issuer string, rpPrivateKey interface{}, rpKeyID string) *Verifier {
	return &Verifier{
		issuer:         issuer,
		rpPrivateKey:   rpPrivateKey,
		rpKeyID:        rpKeyID,
		trustedIssuers: make(map[string]*IssuerMetadata),
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

// AddTrustedIssuer registers an issuer as trusted.
func (v *Verifier) AddTrustedIssuer(meta *IssuerMetadata) {
	v.trustedIssuers[meta.IssuerURL] = meta
}

// CreateAuthorizationRequest generates a signed JWT authorization request
// for the wallet.
func (v *Verifier) CreateAuthorizationRequest(req *PresentationRequest) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":                      v.issuer,
		"response_type":            "vp_token",
		"response_uri":             req.ResponseURI,
		"response_mode":            "direct_post",
		"client_id":                v.issuer,
		"nonce":                    req.Nonce,
		"iat":                      now.Unix(),
		"exp":                      now.Add(5 * time.Minute).Unix(),
		"presentation_definition": map[string]any{
			"id":                req.DefinitionID,
			"input_descriptors": req.InputDescriptors,
		},
		"client_metadata": map[string]any{
			"vp_formats_supported": map[string]any{
				"dc+sd-jwt": map[string]any{
					"sd-jwt_alg_values": []string{"ES256", "RS256"},
					"kb-jwt_alg_values": []string{"ES256"},
				},
			},
			"jwks": map[string]any{
				// RP public key for the wallet to verify the request signature
			},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = v.rpKeyID
	return token.SignedString(v.rpPrivateKey)
}

// VerifyPresentation verifies a VP token from the wallet.
// Returns the extracted claims and verification metadata.
func (v *Verifier) VerifyPresentation(ctx context.Context, vpToken string, expectedNonce string) (*VerificationResult, error) {
	// Step 1: Parse the SD-JWT VC structure.
	// Format: <header>.<payload>.<signature>~<disclosure1>~<disclosure2>~...~<kb-jwt>
	parts := strings.Split(vpToken, "~")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid SD-JWT format: expected at least 1 disclosure + KB-JWT")
	}

	sdjwt := parts[0]
	disclosures := parts[1 : len(parts)-1] // all but last
	kbJWT := parts[len(parts)-1]

	// Step 2: Parse and verify the SD-JWT header + payload.
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
	}
	headerBytes, err := decodeJWTPart(strings.Split(sdjwt, ".")[0])
	if err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}

	// Parse payload to get issuer.
	var payload struct {
		Iss  string         `json:"iss"`
		Sub  string         `json:"sub"`
		Iat  int64          `json:"iat"`
		Exp  int64          `json:"exp"`
		Cnf  json.RawMessage `json:"cnf"`
		Status struct {
			StatusList struct {
				URI string `json:"uri"`
				Idx int    `json:"idx"`
			} `json:"status_list"`
		} `json:"status"`
		SD         []string `json:"_sd"`
	}
	payloadBytes, err := decodeJWTPart(strings.Split(sdjwt, ".")[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	// Step 3: Check issuer trust.
	issuerMeta, ok := v.trustedIssuers[payload.Iss]
	if !ok {
		return nil, fmt.Errorf("untrusted issuer: %s", payload.Iss)
	}
	if issuerMeta.TrustStatus == "untrusted" {
		return nil, fmt.Errorf("issuer %s is not a qualified/trusted issuer", payload.Iss)
	}

	// Step 4: Verify the SD-JWT signature using issuer's public key.
	pubKey, ok := issuerMeta.PublicKeys[header.Kid].(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("issuer key not found: kid=%s", header.Kid)
	}

	_, err = jwt.Parse(sdjwt, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("SD-JWT signature verification failed: %w", err)
	}

	// Step 5: Check expiry.
	if payload.Exp > 0 && time.Now().Unix() > payload.Exp {
		return nil, fmt.Errorf("credential expired at %s", time.Unix(payload.Exp, 0))
	}

	// Step 6: Verify disclosures (reconstruct disclosed claims).
	disclosedClaims, err := verifyDisclosures(payloadBytes, disclosures, payload.SD)
	if err != nil {
		return nil, fmt.Errorf("disclosure verification failed: %w", err)
	}

	// Step 7: Verify the Key Binding JWT (holder proof-of-possession).
	holderKey, err := v.verifyKeyBinding(kbJWT, payload.Cnf, expectedNonce, sdjwt)
	if err != nil {
		return nil, fmt.Errorf("holder binding verification failed: %w", err)
	}

	result := &VerificationResult{
		Issuer:      payload.Iss,
		IssuerName:  issuerMeta.IssuerURL,
		HolderKey:   holderKey,
		Claims:      disclosedClaims,
		ExpiresAt:   time.Unix(payload.Exp, 0),
		StatusIndex: payload.Status.StatusList.Idx,
		StatusURI:   payload.Status.StatusList.URI,
		IsValid:     true,
	}

	return result, nil
}

// verifyKeyBinding verifies the Key Binding JWT from the wallet.
// The KB-JWT proves the holder controls the key in the PID's cnf claim.
func (v *Verifier) verifyKeyBinding(kbJWT string, cnfRaw json.RawMessage, expectedNonce, sdjwtHash string) (string, error) {
	var cnf struct {
		JWK struct {
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
			Kid string `json:"kid"`
		} `json:"jwk"`
	}
	if err := json.Unmarshal(cnfRaw, &cnf); err != nil {
		return "", fmt.Errorf("parse cnf: %w", err)
	}

	// Reconstruct the ECDSA public key from JWK.
	xBytes, err := base64.RawURLEncoding.DecodeString(cnf.JWK.X)
	if err != nil {
		return "", fmt.Errorf("decode JWK x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(cnf.JWK.Y)
	if err != nil {
		return "", fmt.Errorf("decode JWK y: %w", err)
	}

	pubKey := &ecdsa.PublicKey{
		Curve: getCurve(cnf.JWK.Crv),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	// Parse and verify the KB-JWT.
	token, err := jwt.Parse(kbJWT, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected KB-JWT signing method")
		}
		return pubKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("KB-JWT signature invalid: %w", err)
	}

	// Verify the nonce and the SD-JWT hash in the KB-JWT payload.
	claims := token.Claims.(jwt.MapClaims)
	nonce, ok := claims["nonce"].(string)
	if !ok || nonce != expectedNonce {
		return "", fmt.Errorf("nonce mismatch: expected %s", expectedNonce)
	}

	sdHash, ok := claims["sd_hash"].(string)
	if !ok {
		return "", fmt.Errorf("missing sd_hash in KB-JWT")
	}

	// Verify sd_hash matches the hash of the SD-JWT.
	computedHash := sha256.Sum256([]byte(sdjwt))
	computedB64 := base64.RawURLEncoding.EncodeToString(computedHash[:])
	if sdHash != computedB64 {
		return "", fmt.Errorf("sd_hash mismatch: KB-JWT not bound to this credential")
	}

	return cnf.JWK.Kid, nil
}

// verifyDisclosures reconstructs and validates selectively disclosed claims.
// Each disclosure is a base64url-encoded JSON array: [salt, key, value].
// The _sd array in the payload contains SHA-256 hashes of these disclosures.
func verifyDisclosures(payloadJSON []byte, disclosures []string, sdHashes []string) (map[string]any, error) {
	result := make(map[string]any)

	sdHashSet := make(map[string]bool, len(sdHashes))
	for _, h := range sdHashes {
		sdHashSet[h] = true
	}

	for _, disclosure := range disclosures {
		raw, err := base64.RawURLEncoding.DecodeString(disclosure)
		if err != nil {
			return nil, fmt.Errorf("decode disclosure: %w", err)
		}

		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("parse disclosure array: %w", err)
		}
		if len(arr) != 3 {
			return nil, fmt.Errorf("invalid disclosure: expected 3 elements, got %d", len(arr))
		}

		// Compute SHA-256 hash of the disclosure and verify against _sd.
		disclosureHash := sha256.Sum256([]byte(disclosure))
		hashB64 := base64.RawURLEncoding.EncodeToString(disclosureHash[:])
		if !sdHashSet[hashB64] {
			return nil, fmt.Errorf("disclosure hash not found in _sd array")
		}

		// Extract key and value.
		var key string
		json.Unmarshal(arr[1], &key)

		var value interface{}
		json.Unmarshal(arr[2], &value)

		result[key] = value
	}

	return result, nil
}

// decodeJWTPart decodes a base64url JWT segment.
func decodeJWTPart(part string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(part)
}

// getCurve returns the elliptic curve for the given JWK crv value.
func getCurve(crv string) elliptic.Curve {
	switch crv {
	case "P-256":
		return elliptic.P256()
	case "P-384":
		return elliptic.P384()
	case "P-521":
		return elliptic.P521()
	default:
		return elliptic.P256()
	}
}
```

### 4.6 Integration with GGID OAuth Service

The current GGID OAuth service (`services/oauth/internal/server/server.go`)
already supports the authorization code flow, token endpoint, JWKS, and
OIDC discovery. The OID4VP verifier adds a new authentication path:

| Existing Endpoint | New OID4VP Endpoint |
|-------------------|---------------------|
| `GET /oauth/authorize` | `GET /oid4vp/request/:id` (signed authz request for wallet) |
| `POST /oauth/token` | `POST /oid4vp/response` (receive VP token from wallet) |
| `GET /.well-known/openid-configuration` | Extended with `vp_endpoint` |

The OAuth `domain.KeyProvider` interface (defined in
`services/oauth/internal/domain/models.go`) can be reused for signing
OID4VP authorization requests. The existing RSA key management and
rotation infrastructure applies directly.

---

## 5. SIOPv2 (Self-Issued OpenID Provider)

### 5.1 Concept

In traditional OpenID Connect, the user authenticates through an
external Identity Provider (Google, Okta, the GGID Auth Service). The
Relying Party redirects the user to the IdP, which authenticates them
and returns an ID Token.

**SIOPv2** (Self-Issued OpenID Provider v2) flips this model: the
**wallet itself acts as the OpenID Provider**. There is no external IdP.
The wallet:
1. Generates its own DID or key pair.
2. Authenticates the user locally (biometric).
3. Issues an ID Token signed by the wallet's key.
4. Sends it directly to the RP.

This eliminates the need for a trusted third-party IdP for wallet-based
authentication. The RP verifies the ID Token using the wallet's public
key, which it discovers through DID resolution or the authorization
request metadata.

### 5.2 SIOPv2 vs OID4VP

| Feature | SIOPv2 | OID4VP |
|---------|--------|--------|
| Purpose | User authentication | Credential presentation |
| Token type | ID Token (JWT) | VP Token (SD-JWT VC) |
| Issuer of token | Wallet itself | Credential issuer (e.g., PID Provider) |
| External trust needed | No (self-signed) | Yes (issuer must be trusted) |
| Data richness | Basic claims only | Full credential attributes |
| Use case | "Sign in with wallet" | "Prove your identity" |

In practice, SIOPv2 and OID4VP are often combined: the wallet first
authenticates via SIOPv2 (proving it controls a key), then presents
credentials via OID4VP (proving the user has a valid PID). The combined
flow is called **SIOPv2 + OID4VP**.

### 5.3 SIOPv2 Flow

```
 Relying Party (GGID)                 Wallet
      │                                 │
      │  1. SIOPv2 authz request        │
      │     client_id=urn:ietf:params:  │
      │       oauth:client-id:type:     │
      │       self-issued:metadata      │
      ├────────────────────────────────>│
      │                                 │
      │                    2. User auth │
      │                       (biometric)│
      │                                 │
      │  3. ID Token (self-signed)      │
      │<────────────────────────────────┤
      │     sub = wallet DID             │
      │     iss = wallet DID             │
      │     aud = RP client_id           │
      │     nonce = RP challenge         │
      │                                 │
      │  4. Verify ID Token              │
      │     (using wallet's public key   │
      │      from DID / request metadata)│
      │                                 │
      │  5. Create session               │
      │     (sub = wallet DID → user)    │
      │                                 │
```

### 5.4 Authorization Request for SIOPv2

The RP sends a standard OIDC authorization request but with special
parameters:

```
GET openid://?
  scope=openid
  &response_type=id_token
  &client_id=https://ggid.example.com
  &redirect_uri=https://ggid.example.com/siop/callback
  &response_mode=direct_post
  &nonce=s-n3k4j5h6l7
  &request_uri=https://ggid.example.com/siop/request/jwt-xyz
```

The key differences from standard OIDC:
- `response_type=id_token` (no authorization code — the response IS the
  ID Token, returned directly)
- `response_mode=direct_post` (wallet POSTs directly to RP, no browser
  redirect)
- No `client_secret` (the wallet doesn't register with the RP)

### 5.5 ID Token from Wallet

The wallet issues a self-signed ID Token:

```json
{
  "iss": "did:key:z6MkpTHR8VNsBxYAA3ituYQ4fN5h5sB7Vp3J5S4rNxQi7eKm",
  "sub": "did:key:z6MkpTHR8VNsBxYAA3ituYQ4fN5h5sB7Vp3J5S4rNxQi7eKm",
  "aud": "https://ggid.example.com",
  "nonce": "s-n3k4j5h6l7",
  "iat": 1735689600,
  "exp": 1735689900,
  "_sd_alg": "sha-256",
  "claims": {
    "name": "Max Mustermann"
  }
}
```

The `iss` and `sub` are the same — the wallet's DID. The wallet signs
this token with its private key, and the RP verifies using the public
key derived from the DID.

### 5.6 Go Code: SIOPv2 Verifier

```go
// Package siop implements the SIOPv2 (Self-Issued OpenID Provider) flow
// for EU Digital Identity Wallet authentication.
package siop

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SIOPVerifier handles the RP side of SIOPv2 authentication.
type SIOPVerifier struct {
	rpID         string // RP client_id (URL)
	rpPrivateKey interface{}
	rpKeyID      string
	httpClient   *http.Client
	didResolver  DIDResolver
}

// DIDResolver resolves a DID to its verification methods (public keys).
type DIDResolver interface {
	Resolve(ctx context.Context, did string) (*DIDDocument, error)
}

// DIDDocument is a minimal DID document for key extraction.
type DIDDocument struct {
	ID                   string            `json:"id"`
	VerificationMethod   []VerificationMethod `json:"verificationMethod"`
	Authentication       []interface{}     `json:"authentication"`
	AssertionMethod      []interface{}     `json:"assertionMethod"`
}

// VerificationMethod describes a public key in the DID document.
type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
	PublicKeyJWK       map[string]any `json:"publicKeyJwk,omitempty"`
}

// SIOPAuthRequest is the SIOPv2 authorization request sent to the wallet.
type SIOPAuthRequest struct {
	Scope        string `json:"scope"`
	ResponseType string `json:"response_type"`
	ClientID     string `json:"client_id"`
	RedirectURI  string `json:"redirect_uri"`
	Nonce        string `json:"nonce"`
}

// SIOPIDToken is the self-issued ID Token from the wallet.
type SIOPIDToken struct {
	Iss   string `json:"iss"`   // wallet DID
	Sub   string `json:"sub"`   // wallet DID (same as iss)
	Aud   string `json:"aud"`   // RP client_id
	Nonce string `json:"nonce"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

// AuthResult is the verified SIOPv2 authentication result.
type AuthResult struct {
	WalletDID  string         `json:"wallet_did"`
	Claims     map[string]any `json:"claims"`
	ExpiresAt  time.Time      `json:"expires_at"`
}

// NewSIOPVerifier creates a new SIOPv2 verifier.
func NewSIOPVerifier(rpID string, rpPrivateKey interface{}, rpKeyID string, resolver DIDResolver) *SIOPVerifier {
	return &SIOPVerifier{
		rpID:         rpID,
		rpPrivateKey: rpPrivateKey,
		rpKeyID:      rpKeyID,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		didResolver:  resolver,
	}
}

// CreateAuthRequest generates the SIOPv2 authorization request as a signed JWT.
func (v *SIOPVerifier) CreateAuthRequest(nonce, redirectURI string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"scope":         "openid",
		"response_type": "id_token",
		"client_id":     v.rpID,
		"redirect_uri":  redirectURI,
		"response_mode": "direct_post",
		"nonce":         nonce,
		"iss":           v.rpID,
		"iat":           now.Unix(),
		"exp":           now.Add(5 * time.Minute).Unix(),
		"client_metadata": map[string]any{
			"jwks": map[string]any{
				"keys": []map[string]any{
					{
						"kty": "EC",
						"crv": "P-256",
						"kid": v.rpKeyID,
						"use": "sig",
						"alg": "ES256",
					},
				},
			},
			"id_token_signing_alg_values_supported": []string{"ES256", "RS256"},
			"subject_syntax_types_supported":       []string{"did:key", "did:web", "did:jwk"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = v.rpKeyID
	return token.SignedString(v.rpPrivateKey)
}

// VerifyIDToken verifies a self-issued ID Token from the wallet.
func (v *SIOPVerifier) VerifyIDToken(ctx context.Context, idToken string, expectedNonce string) (*AuthResult, error) {
	// Step 1: Parse without verification to extract issuer (wallet DID).
	var unverified SIOPIDToken
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	unverifiedToken, _, err := parser.ParseUnverified(idToken, &unverified)
	if err != nil {
		return nil, fmt.Errorf("parse ID token: %w", err)
	}
	_ = unverifiedToken

	// Step 2: Resolve the wallet DID to get the public key.
	didDoc, err := v.didResolver.Resolve(ctx, unverified.Iss)
	if err != nil {
		return nil, fmt.Errorf("resolve wallet DID %s: %w", unverified.Iss, err)
	}

	// Step 3: Extract the public key from the DID document.
	pubKey, err := extractPublicKey(didDoc)
	if err != nil {
		return nil, fmt.Errorf("extract public key from DID: %w", err)
	}

	// Step 4: Verify the ID Token signature.
	token, err := jwt.Parse(idToken, func(t *jwt.Token) (interface{}, error) {
		// Verify the algorithm matches expected ECDSA.
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("ID token signature verification failed: %w", err)
	}

	// Step 5: Validate claims.
	claims := token.Claims.(jwt.MapClaims)

	// Check audience.
	aud, ok := claims["aud"].(string)
	if !ok || aud != v.rpID {
		return nil, fmt.Errorf("audience mismatch: expected %s, got %v", v.rpID, claims["aud"])
	}

	// Check nonce.
	nonce, ok := claims["nonce"].(string)
	if !ok || nonce != expectedNonce {
		return nil, fmt.Errorf("nonce mismatch: expected %s", expectedNonce)
	}

	// Check expiry.
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing exp claim")
	}
	expiresAt := time.Unix(int64(exp), 0)
	if time.Now().After(expiresAt) {
		return nil, fmt.Errorf("ID token expired at %s", expiresAt)
	}

	// Check that iss == sub (self-issued property).
	iss, _ := claims["iss"].(string)
	sub, _ := claims["sub"].(string)
	if iss != sub {
		return nil, fmt.Errorf("iss != sub in self-issued token: %s != %s", iss, sub)
	}

	// Extract any additional claims the wallet included.
	extraClaims := make(map[string]any)
	for k, v := range claims {
		switch k {
		case "iss", "sub", "aud", "nonce", "iat", "exp", "_sd_alg":
			continue
		default:
			extraClaims[k] = v
		}
	}

	return &AuthResult{
		WalletDID: iss,
		Claims:    extraClaims,
		ExpiresAt: expiresAt,
	}, nil
}

// extractPublicKey extracts the ECDSA public key from a DID document.
func extractPublicKey(doc *DIDDocument) (*ecdsa.PublicKey, error) {
	for _, vm := range doc.VerificationMethod {
		if vm.PublicKeyJWK != nil {
			return jwkToECDSA(vm.PublicKeyJWK)
		}
		if vm.PublicKeyMultibase != "" {
			return multibaseToECDSA(vm.PublicKeyMultibase)
		}
	}
	return nil, fmt.Errorf("no usable public key in DID document")
}

// jwkToECDSA converts a JWK map to an ECDSA public key.
func jwkToECDSA(jwk map[string]any) (*ecdsa.PublicKey, error) {
	crvStr, _ := jwk["crv"].(string)
	xStr, _ := jwk["x"].(string)
	yStr, _ := jwk["y"].(string)

	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, err
	}

	curve := getCurve(crvStr)
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// multibaseToECDSA decodes a multibase-encoded public key (used by did:key).
func multibaseToECDSA(mb string) (*ecdsa.PublicKey, error) {
	// Multibase prefix for base58btc is 'z'.
	// For did:key with P-256, the multicodec prefix is 0x1200.
	if !strings.HasPrefix(mb, "z") {
		return nil, fmt.Errorf("unsupported multibase encoding")
	}
	// In production, use go-multibase + go-multicodec for proper decoding.
	// This is a simplified version.
	return nil, fmt.Errorf("multibase key decoding requires multicodec library")
}

// getCurve returns the elliptic curve for a JWK crv value.
func getCurve(crv string) elliptic.Curve {
	switch crv {
	case "P-256":
		return elliptic.P256()
	case "P-384":
		return elliptic.P384()
	default:
		return elliptic.P256()
	}
}
```

### 5.7 Combined SIOPv2 + OID4VP

For full wallet-based authentication with credential verification, the
EUDI Wallet supports a combined flow where both an ID Token (SIOPv2)
and a VP Token (OID4VP) are returned:

```json
{
  "response_type": "id_token vp_token",
  "nonce": "combined-nonce-123",
  "presentation_definition": { ... }
}
```

The response contains both:
```json
{
  "id_token": "<SIOPv2 self-issued JWT>",
  "vp_token": "<SD-JWT VC with holder binding>"
}
```

The RP verifies the ID Token (proving wallet control) and the VP Token
(proving credential validity). This provides both authentication and
identity proofing in one exchange.

---

## 6. Wallet-to-RP Trust Establishment

### 6.1 Why RP Registration Matters

In the traditional OAuth/OIDC model, the RP registers with the IdP.
The IdP trusts the RP because of this registration. But in the wallet
model, there is no central IdP — the wallet communicates directly with
the RP.

This creates a critical trust gap: **how does the wallet know the RP is
legitimate?** Without trust establishment:
- A phishing app could impersonate a bank and steal the user's PID.
- A malicious website could request excessive personal data.
- There would be no accountability for data misuse.

eIDAS 2.0 solves this with **mandatory RP registration** in a national
trust framework.

### 6.2 RP Registration Process

```
 Relying Party              National Authority          Wallet
 (e.g., GGID)               (RP Registry)              (at presentation time)
     │                            │                          │
     │ 1. Submit registration     │                          │
     │    - Legal entity name     │                          │
     │    - Registration number   │                          │
     │    - Sector (banking,     │                          │
     │      healthcare, etc.)    │                          │
     │    - Purpose of data      │                          │
     │      processing           │                          │
     │    - Public key / CSR     │                          │
     ├───────────────────────────>│                          │
     │                            │                          │
     │ 2. Authority verifies     │                          │
     │    entity exists, reviews │                          │
     │    purpose                 │                          │
     │                            │                          │
     │ 3. Issues RP certificate   │                          │
     │<───────────────────────────┤                          │
     │    (X.509 or JWT attestation)                         │
     │    + adds to RP Registry  │                          │
     │                            │                          │
     │                            │                          │
     │ 4. RP signs authz request  │                          │
     │    with RP cert            │                          │
     │                                                       │
     │ 5. Wallet receives         │                          │
     │    authz request           │                          │
     ├───────────────────────────────────────────────────────>│
     │                            │                          │
     │                            │ 6. Wallet checks RP      │
     │                            │    cert against Registry │
     │                            │<─────────────────────────┤
     │                            │                          │
     │                            │ 7. RP is registered ✓    │
     │                            │─────────────────────────>│
     │                            │                          │
     │                            │    8. Wallet displays:   │
     │                            │    "GGID Inc. (verified) │
     │                            │     wants: family_name,  │
     │                            │     age_over_18          │
     │                            │     Purpose: Account     │
     │                            │     registration"        │
     │                            │                          │
     │                            │    9. User approves ✓    │
     │  10. VP Token              │                          │
     │<───────────────────────────────────────────────────────┤
     │                                                       │
```

### 6.3 RP Certificate Format

The RP certificate can be either an X.509 certificate or a JWT
attestation. The EU ARF prefers JWT-based attestations for flexibility:

```json
{
  "iss": "https://registry.eidas.gov.de",
  "sub": "https://ggid.example.com",
  "rp_legal_name": "GGID GmbH",
  "rp_legal_id": "DE-123456789",
  "rp_sector": "private",
  "rp_purpose": "Identity verification for account registration",
  "rp_attributes_requested": ["family_name", "given_name", "age_over_18"],
  "rp_public_key": {
    "kty": "EC",
    "crv": "P-256",
    "x": "...",
    "y": "...",
    "kid": "ggid-rp-key-1"
  },
  "iat": 1735689600,
  "exp": 1767225600
}
```

This attestation is signed by the national authority's key, which is
itself anchored in the EU Trust List.

### 6.4 Anti-Phishing Properties

The RP registration framework provides strong anti-phishing guarantees:

1. **Visual verification**: The wallet displays the RP's verified legal
   name, not just a URL. Users see "GGID GmbH" (verified entity), not
   "ggid.example.com" (potentially spoofed URL).

2. **Certificate pinning**: The wallet only sends data to the RP whose
   key matches the certificate. A phishing site with a different key
   cannot receive any presentation.

3. **Purpose display**: The wallet shows the declared purpose of data
   processing. If the purpose doesn't match user expectations, they
   can decline.

4. **Audit trail**: Every data access is logged with the RP's
   registration number. Misuse can be traced to a legal entity.

### 6.5 Go Code: RP Registration Verification

```go
// Package rpregistry implements Relying Party registration verification
// for the EU Digital Identity Wallet ecosystem.
package rpregistry

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// RPRegistry verifies Relying Party registrations against a national
// or EU-wide registry.
type RPRegistry struct {
	registryURL string        // National RP registry base URL
	authorityKey *ecdsa.PublicKey // National authority's signing key
	cache       *RPCache
	httpClient  *http.Client
}

// RPCache caches RP registrations with TTL to avoid repeated lookups.
type RPCache struct {
	mu      sync.RWMutex
	entries map[string]*RPRegistration // keyed by RP client_id
	ttl     time.Duration
}

// RPRegistration represents a registered Relying Party.
type RPRegistration struct {
	ClientID        string    `json:"sub"`
	LegalName       string    `json:"rp_legal_name"`
	LegalID         string    `json:"rp_legal_id"`
	Sector          string    `json:"rp_sector"`
	Purpose         string    `json:"rp_purpose"`
	AttributesReq   []string  `json:"rp_attributes_requested"`
	PublicKey       RPKey     `json:"rp_public_key"`
	ExpiresAt       time.Time `json:"exp"`
	IssuedAt        time.Time `json:"iat"`
}

// RPKey is the RP's public key for verifying authorization request signatures.
type RPKey struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Kid string `json:"kid"`
}

// NewRPRegistry creates a new RP registry client.
func NewRPRegistry(registryURL string, authorityKey *ecdsa.PublicKey) *RPRegistry {
	return &RPRegistry{
		registryURL:  registryURL,
		authorityKey: authorityKey,
		cache: &RPCache{
			entries: make(map[string]*RPRegistration),
			ttl:     1 * time.Hour,
		},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// VerifyRP checks if an RP is registered and returns its registration.
// Uses cached results when available; falls back to registry lookup.
func (r *RPRegistry) VerifyRP(ctx context.Context, clientID string) (*RPRegistration, error) {
	// Check cache first.
	if reg := r.cache.get(clientID); reg != nil {
		return reg, nil
	}

	// Fetch from registry.
	reg, err := r.fetchRegistration(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("RP not found in registry: %w", err)
	}

	// Cache the result.
	r.cache.set(clientID, reg)
	return reg, nil
}

// VerifyAuthzRequest verifies that an authorization request is signed by
// a registered RP, and that the requested attributes match what the RP
// is authorized to request.
func (r *RPRegistry) VerifyAuthzRequest(
	ctx context.Context,
	authzRequestJWT string,
	requestedAttributes []string,
) (*RPRegistration, error) {
	// Step 1: Parse the JWT header to get the RP's kid.
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	headerBytes, err := decodeJWTHeader(authzRequestJWT)
	if err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("unmarshal header: %w", err)
	}

	// Step 2: Extract issuer (client_id) from the JWT payload.
	var payload struct {
		Iss      string `json:"iss"`
		ClientID string `json:"client_id"`
	}
	payloadBytes, err := decodeJWTPayload(authzRequestJWT)
	if err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	clientID := payload.ClientID
	if clientID == "" {
		clientID = payload.Iss
	}

	// Step 3: Verify RP is registered.
	reg, err := r.VerifyRP(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("unregistered RP: %w", err)
	}

	// Step 4: Verify the authorization request signature using RP's key.
	rpPubKey, err := rpKeyToECDSA(&reg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("convert RP key: %w", err)
	}

	_, err = jwt.Parse(authzRequestJWT, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		// Verify the key ID matches.
		kid, _ := t.Header["kid"].(string)
		if kid != reg.PublicKey.Kid {
			return nil, fmt.Errorf("key ID mismatch: expected %s, got %s", reg.PublicKey.Kid, kid)
		}
		return rpPubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("authorization request signature invalid: %w", err)
	}

	// Step 5: Verify requested attributes are authorized.
	for _, attr := range requestedAttributes {
		if !contains(reg.AttributesReq, attr) {
			return nil, fmt.Errorf("RP %s is not authorized to request attribute: %s", reg.LegalName, attr)
		}
	}

	// Step 6: Check registration hasn't expired.
	if time.Now().After(reg.ExpiresAt) {
		return nil, fmt.Errorf("RP registration expired at %s", reg.ExpiresAt)
	}

	return reg, nil
}

// fetchRegistration fetches the RP registration JWT from the registry.
func (r *RPRegistry) fetchRegistration(ctx context.Context, clientID string) (*RPRegistration, error) {
	url := fmt.Sprintf("%s/rp/%s", r.registryURL, clientID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned %d for RP %s", resp.StatusCode, clientID)
	}

	// The response is a JWT attestation signed by the national authority.
	var attestation string
	if err := json.NewDecoder(resp.Body).Decode(&attestation); err != nil {
		return nil, fmt.Errorf("decode attestation: %w", err)
	}

	// Verify the attestation signature.
	token, err := jwt.ParseWithClaims(attestation, &RPRegistration{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected authority signing method")
		}
		return r.authorityKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("attestation signature invalid: %w", err)
	}

	reg, ok := token.Claims.(*RPRegistration)
	if !ok {
		return nil, fmt.Errorf("invalid attestation claims")
	}

	return reg, nil
}

// --- Cache helpers ---

func (c *RPCache) get(clientID string) *RPRegistration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[clientID]
	if !ok {
		return nil
	}
	// Check TTL (simplified — in production, track set time per entry).
	return entry
}

func (c *RPCache) set(clientID string, reg *RPRegistration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[clientID] = reg
}

// --- Utility functions ---

func rpKeyToECDSA(key *RPKey) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, err
	}
	curve := elliptic.P256()
	if key.Crv == "P-384" {
		curve = elliptic.P384()
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// decodeJWTHeader and decodeJWTPayload extract raw bytes from a JWT.
func decodeJWTHeader(token string) ([]byte, error) {
	parts := splitJWT(token)
	return base64.RawURLEncoding.DecodeString(parts[0])
}

func decodeJWTPayload(token string) ([]byte, error) {
	parts := splitJWT(token)
	return base64.RawURLEncoding.DecodeString(parts[1])
}

func splitJWT(token string) []string {
	return strings.SplitN(token, ".", 3)
}
```

### 6.6 Trust Chain

The full trust chain from root to presentation:

```
EU Trust List (root)
  └── National Authority (RP Registry signing key)
      └── RP Registration Attestation (JWT)
          └── RP Authorization Request (JWT, signed by RP key)
              └── VP Token Response (SD-JWT VC, from wallet)

Each level is cryptographically verified:
  1. National Authority key ← verified against EU Trust List
  2. RP Registration ← verified against National Authority key
  3. RP Authorization Request ← verified against RP public key (from registration)
  4. VP Token ← verified against PID Provider key (from EU Trust List / issuer metadata)
```

---

## 7. Verifiable Credentials Format

### 7.1 Three Credential Formats

The EUDI Wallet ecosystem supports three credential formats. The EU ARF
specifies SD-JWT VC as the primary format for PID and QEAA.

| Format | Specification | Complexity | EU ARF Status |
|--------|---------------|------------|---------------|
| **SD-JWT VC** | RFC 9901 + draft-ietf-oauth-sd-jwt-vc | Low | **Primary** (PID, QEAA) |
| W3C VC JSON-LD | W3C VC DM v2.0 | High | Supported (interoperability) |
| ISO mDL | ISO/IEC 18013-5 | Medium | Supported (mobile driving license) |

### 7.2 W3C Verifiable Credentials Data Model

The W3C VC Data Model (v2.0, 2024) defines a general framework for
expressing credentials. The core structure:

```json
{
  "@context": [
    "https://www.w3.org/ns/credentials/v2",
    "https://w3id.org/eidas2/credentials/v1"
  ],
  "id": "urn:uuid:550e8400-e29b-41d4-a716-446655440000",
  "type": ["VerifiableCredential", "PersonalIdentificationData"],
  "issuer": {
    "id": "https://pid-provider.bund.de",
    "name": "Federal Office of Administration (Germany)",
    "country": "DE"
  },
  "validFrom": "2025-01-15T10:00:00Z",
  "validUntil": "2030-01-15T10:00:00Z",
  "credentialSchema": {
    "type": "JsonSchema",
    "id": "https://pid-provider.bund.de/schemas/pid-v1.json"
  },
  "credentialSubject": {
    "id": "did:key:z6MkpTHR8VNsBxYAA3...",
    "family_name": "Mustermann",
    "given_name": "Max",
    "birth_date": "1990-05-15",
    "nationality": ["DE"],
    "age_over_18": true
  },
  "proof": {
    "type": "Ed25519Signature2020",
    "created": "2025-01-15T10:00:05Z",
    "verificationMethod": "https://pid-provider.bund.de/keys/key-1",
    "proofPurpose": "assertionMethod",
    "proofValue": "z58DAdFto9oqpHDJ5udt..."
  }
}
```

### 7.3 SD-JWT (Selective Disclosure JWT)

SD-JWT (RFC 9901, published 2025) is the EU's preferred format because:

1. **JWT-compatible**: Uses the existing JWT infrastructure (JWS, JWKS,
   `alg` algorithms). No new crypto or serialization needed.
2. **Simpler than JSON-LD**: No RDF canonicalization, no linked data
   proofs. Just JWTs with hash references.
3. **Selective disclosure built-in**: Each claim can be individually
   revealed or withheld using salted hashes.
4. **Compact**: Typical SD-JWT PID is ~2-4 KB; JSON-LD VCs with proofs
   are larger.

#### How SD-JWT Selective Disclosure Works

```
Issuance:
  Credential claims: { family_name: "Mustermann", given_name: "Max",
                       birth_date: "1990-05-15", nationality: ["DE"] }

  For each disclosable claim, create a disclosure:
    disclosure_1 = base64url([random_salt_1, "family_name", "Mustermann"])
    disclosure_2 = base64url([random_salt_2, "given_name", "Max"])
    disclosure_3 = base64url([random_salt_3, "birth_date", "1990-05-15"])
    disclosure_4 = base64url([random_salt_4, "nationality", ["DE"]])

  Compute hashes:
    hash_1 = SHA-256(disclosure_1)
    hash_2 = SHA-256(disclosure_2)
    hash_3 = SHA-256(disclosure_3)
    hash_4 = SHA-256(disclosure_4)

  SD-JWT payload:
    {
      "_sd": [hash_1, hash_2, hash_3, hash_4],
      "_sd_alg": "sha-256",
      "iss": "https://pid-provider.bund.de",
      "cnf": { "jwk": { "kty": "EC", "crv": "P-256", ... } },
      "status": { ... },
      "iat": 1735689600,
      "exp": 1798876800
    }

  The SD-JWT is: header.payload.signature
  The full credential is: header.payload.signature~disclosure_1~disclosure_2~disclosure_3~disclosure_4

Presentation (selective disclosure):
  Wallet reveals only family_name and age_over_18:
    header.payload.signature~disclosure_1~disclosure_3~<kb-jwt>

  RP verifies:
    1. Verify SD-JWT signature (issuer's key)
    2. Recompute SHA-256(disclosure_1) and check it's in _sd array
    3. Recompute SHA-256(disclosure_3) and check it's in _sd array
    4. Verify KB-JWT (holder's key, proving possession)
    5. Claims not disclosed (given_name, nationality) are never seen
```

### 7.4 Why SD-JWT Is Preferred for EU Wallet

| Criterion | SD-JWT VC | JSON-LD VC | ISO mDL |
|-----------|-----------|------------|---------|
| Implementation complexity | Low | High | Medium |
| Existing JWT infrastructure | **Yes** | No (custom) | No (CBOR) |
| Selective disclosure | **Built-in** | With BBS+ | Built-in |
| Key binding | **KB-JWT** | LD-Proof | Device Auth |
| Library availability | **Go: jwt/v5 + custom** | Aries Go | iOS/Android native |
| EU ARF recommendation | **Primary** | Alternative | mDL specific |
| Verifier learning curve | **Low** (JWT knowledge) | High (DID, LD) | Medium (CBOR) |

### 7.5 SD-JWT VC Structure for PID

The complete PID in SD-JWT VC format:

```
<JWT header>.<JWT payload>.<JWT signature>~<disclosure>~<disclosure>~...~<KB-JWT>

JWT header: { "alg": "ES256", "typ": "dc+sd-jwt", "kid": "pid-key-1" }

JWT payload: {
  "_sd": [
    "hash_of_family_name_disclosure",
    "hash_of_given_name_disclosure",
    "hash_of_birth_date_disclosure",
    "hash_of_nationality_disclosure",
    "hash_of_age_over_18_disclosure"
  ],
  "_sd_alg": "sha-256",
  "iss": "https://pid-provider.bund.de",
  "iat": 1735689600,
  "exp": 1798876800,
  "cnf": {
    "jwk": { "kty": "EC", "crv": "P-256", "x": "...", "y": "...", "kid": "holder-key-1" }
  },
  "status": {
    "status_list": { "uri": "https://pid-provider.bund.de/status/list-1", "idx": 42 }
  },
  "type": "urn:eu.europa.ec.eudi:pid:1"
}

Disclosure (base64url of [salt, key, value]):
  e30...base64...~   [ "abc123salt", "family_name", "Mustermann" ]
  e30...base64...~   [ "def456salt", "given_name", "Max" ]
  e30...base64...~   [ "ghi789salt", "birth_date", "1990-05-15" ]
  e30...base64...~   [ "jkl012salt", "nationality", ["DE"] ]
  e30...base64...~   [ "mno345salt", "age_over_18", true ]

KB-JWT (Key Binding, signed by wallet):
  { "alg": "ES256", "typ": "kb+jwt" }.
  {
    "iss": "did:key:z6MkpTHR8VNsBxYAA3...",
    "aud": "https://ggid.example.com",
    "nonce": "n-47s8d2j3kl4h5j6",
    "iat": 1735689700,
    "sd_hash": "sha256 of the SD-JWT part (before ~)"
  }.
  <ECDSA signature with wallet's holder key>
```

### 7.6 Go Code: SD-JWT Verification

```go
// Package sdjwt implements SD-JWT (RFC 9901) verification for the
// EU Digital Identity Wallet credential format.
package sdjwt

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// SDJWT represents a parsed and verified SD-JWT credential.
type SDJWT struct {
	Issuer        string
	Subject       string
	IssuedAt      int64
	ExpiresAt     int64
	IssuerClaims  map[string]any // non-selective claims in the payload
	DisclosedClaims map[string]any // selectively disclosed claims
	HolderKey     map[string]any // cnf.jwk
	StatusURI     string
	StatusIndex   int
	SDHashes      []string       // _sd array
}

// Disclosure represents a decoded disclosure element.
type Disclosure struct {
	Salt   string
	Key    string
	Value  interface{}
	Raw    string // base64url-encoded original
}

// ParseAndVerify parses an SD-JWT VC, verifies the issuer signature,
// and returns the disclosed claims.
func ParseAndVerify(
	sdjwtCompact string,
	issuerPublicKey *ecdsa.PublicKey,
) (*SDJWT, error) {
	// Split by ~ to separate the SD-JWT from disclosures and KB-JWT.
	parts := strings.Split(sdjwtCompact, "~")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid SD-JWT: expected at least 1 disclosure + KB-JWT")
	}

	sdjwtPart := parts[0]
	disclosureParts := parts[1:]

	// Separate disclosures from the KB-JWT (last element if it's a JWT).
	var disclosures []string
	var kbJWT string
	if len(disclosureParts) > 0 {
		last := disclosureParts[len(disclosureParts)-1]
		if strings.Count(last, ".") == 2 {
			// Last part is a JWT (KB-JWT).
			kbJWT = last
			disclosures = disclosureParts[:len(disclosureParts)-1]
		} else {
			disclosures = disclosureParts
		}
	}

	// Parse and verify the SD-JWT.
	token, err := jwt.Parse(sdjwtPart, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unsupported signing algorithm: %v", t.Header["alg"])
			}
		}
		return issuerPublicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("SD-JWT signature verification failed: %w", err)
	}

	claims := token.Claims.(jwt.MapClaims)

	// Extract issuer-level claims.
	issuer, _ := claims["iss"].(string)
	subject, _ := claims["sub"].(string)
	iat, _ := claims["iat"].(float64)
	exp, _ := claims["exp"].(float64)

	// Extract _sd hashes.
	sdHashes := extractStringArray(claims, "_sd")

	// Extract holder key from cnf.
	holderKey := extractMap(claims, "cnf")

	// Extract status info.
	statusURI, statusIdx := extractStatus(claims)

	// Verify disclosures.
	disclosedClaims, err := verifyDisclosures(sdHashes, disclosures)
	if err != nil {
		return nil, fmt.Errorf("disclosure verification failed: %w", err)
	}

	result := &SDJWT{
		Issuer:          issuer,
		Subject:         subject,
		IssuedAt:        int64(iat),
		ExpiresAt:       int64(exp),
		IssuerClaims:    extractIssuerClaims(claims),
		DisclosedClaims: disclosedClaims,
		HolderKey:       holderKey,
		StatusURI:       statusURI,
		StatusIndex:     statusIdx,
		SDHashes:        sdHashes,
	}

	// Verify KB-JWT if present.
	if kbJWT != "" {
		if err := verifyKBJWT(kbJWT, holderKey, sdjwtPart); err != nil {
			return nil, fmt.Errorf("KB-JWT verification failed: %w", err)
		}
	}

	return result, nil
}

// verifyDisclosures reconstructs selectively disclosed claims.
func verifyDisclosures(sdHashes []string, disclosures []string) (map[string]any, error) {
	// Build a set of expected hashes.
	hashSet := make(map[string]bool)
	for _, h := range sdHashes {
		hashSet[h] = true
	}

	result := make(map[string]any)

	for _, disclosure := range disclosures {
		// Decode the disclosure.
		raw, err := base64.RawURLEncoding.DecodeString(disclosure)
		if err != nil {
			return nil, fmt.Errorf("decode disclosure: %w", err)
		}

		// Parse as [salt, key, value].
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, fmt.Errorf("parse disclosure: %w", err)
		}
		if len(arr) != 3 {
			return nil, fmt.Errorf("invalid disclosure format: expected 3 elements")
		}

		// Verify the hash.
		hash := sha256.Sum256([]byte(disclosure))
		hashB64 := base64.RawURLEncoding.EncodeToString(hash[:])
		if !hashSet[hashB64] {
			return nil, fmt.Errorf("disclosure hash not in _sd array (possible tampering)")
		}

		// Extract key and value.
		var key string
		json.Unmarshal(arr[1], &key)

		var value interface{}
		json.Unmarshal(arr[2], &value)

		result[key] = value
	}

	return result, nil
}

// verifyKBJWT verifies the Key Binding JWT.
func verifyKBJWT(kbJWT string, holderKey map[string]any, sdjwtPart string) error {
	// Reconstruct the holder's public key from cnf.jwk.
	jwk, ok := holderKey["jwk"].(map[string]any)
	if !ok {
		return fmt.Errorf("no jwk in cnf claim")
	}

	pubKey, err := jwkToPublicKey(jwk)
	if err != nil {
		return fmt.Errorf("extract holder key: %w", err)
	}

	// Parse and verify the KB-JWT.
	token, err := jwt.Parse(kbJWT, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unsupported KB-JWT algorithm")
		}
		return pubKey, nil
	})
	if err != nil {
		return fmt.Errorf("KB-JWT signature invalid: %w", err)
	}

	// Verify sd_hash matches the SD-JWT.
	claims := token.Claims.(jwt.MapClaims)
	sdHash, ok := claims["sd_hash"].(string)
	if !ok {
		return fmt.Errorf("missing sd_hash in KB-JWT")
	}

	computedHash := sha256.Sum256([]byte(sdjwtPart))
	computedB64 := base64.RawURLEncoding.EncodeToString(computedHash[:])
	if sdHash != computedB64 {
		return fmt.Errorf("sd_hash mismatch: KB-JWT not bound to this SD-JWT")
	}

	return nil
}

// jwkToPublicKey converts a JWK map to a crypto public key.
func jwkToPublicKey(jwk map[string]any) (*ecdsa.PublicKey, error) {
	kty, _ := jwk["kty"].(string)
	if kty != "EC" {
		return nil, fmt.Errorf("unsupported key type: %s", kty)
	}

	crv, _ := jwk["crv"].(string)
	xStr, _ := jwk["x"].(string)
	yStr, _ := jwk["y"].(string)

	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}

	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported curve: %s", crv)
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

// --- Helpers ---

func extractStringArray(claims jwt.MapClaims, key string) []string {
	arr, ok := claims[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func extractMap(claims jwt.MapClaims, key string) map[string]any {
	if m, ok := claims[key].(map[string]any); ok {
		return m
	}
	return nil
}

func extractStatus(claims jwt.MapClaims) (string, int) {
	status, ok := claims["status"].(map[string]any)
	if !ok {
		return "", 0
	}
	sl, ok := status["status_list"].(map[string]any)
	if !ok {
		return "", 0
	}
	uri, _ := sl["uri"].(string)
	idx, _ := sl["idx"].(float64)
	return uri, int(idx)
}

func extractIssuerClaims(claims jwt.MapClaims) map[string]any {
	result := make(map[string]any)
	for k, v := range claims {
		if k == "_sd" || k == "_sd_alg" || k == "cnf" || k == "status" {
			continue
		}
		result[k] = v
	}
	return result
}
```

---

## 8. Status List 2021 (Revocation)

### 8.1 Overview

Status List 2021 (IETF draft `draft-ietf-oauth-status-list`, near final)
provides a privacy-preserving, scalable revocation mechanism. Instead
of checking each credential individually (which would leak which
credential is being checked to the issuer), the verifier fetches the
entire status list and checks a bit position locally.

> See our dedicated research document:
> [token-status-list.md](token-status-list.md) for full protocol details.
> This section focuses on the EUDI Wallet integration aspects.

### 8.2 How It Works in the Wallet Context

```
PID Provider                    Wallet                     Relying Party
(issuer of status list)                                    (verifier)
     │                           │                              │
     │ Issues PID                │                              │
     │ (status.idx = 42)         │                              │
     ├──────────────────────────>│                              │
     │                           │                              │
     │                           │ 1. RP requests presentation  │
     │                           │<─────────────────────────────│
     │                           │                              │
     │                           │ 2. Wallet checks status      │
     │                           │    before presenting         │
     │                           │                              │
     │ 3. Fetch status list      │                              │
     │<──────────────────────────┤                              │
     │                           │                              │
     │ 4. Compressed bitstring   │                              │
     │──────────────────────────>│                              │
     │                           │                              │
     │                           │ 5. Check bit at idx=42       │
     │                           │    → 0 (valid)               │
     │                           │                              │
     │                           │ 6. Present credential to RP  │
     │                           │─────────────────────────────>│
     │                           │                              │
     │                           │    7. RP also checks status  │
     │                           │    (fetches same list)       │
     │                           │    → bit at idx=42 = 0 (ok)  │
     │                           │                              │
     │                           │    8. Accept presentation    │
     │                           │<─────────────────────────────│
```

Both the wallet (before presenting) and the RP (after receiving) should
check the status. This double-check ensures the credential hasn't been
revoked between the wallet's check and the RP's verification.

### 8.3 Status List Structure

The status list is a compressed bitstring published as a JWT:

```json
{
  "iss": "https://pid-provider.bund.de",
  "sub": "https://pid-provider.bund.de/status/list-1",
  "iat": 1735689600,
  "bits": 1,
  "lst": "eNrbuRgAAhcBXQ"
}
```

- `bits`: Number of bits per credential (1 = valid/revoked only).
- `lst`: Base64url-encoded ZLIB-compressed bitstring.

For 131,072 credentials with 1-bit status, the uncompressed bitstring
is 16,384 bytes. After ZLIB compression, it's typically 100-500 bytes
(most bits are 0 for active credentials).

### 8.4 Go Code: Status List Verification

```go
// Package statuslist implements Status List 2021 verification for
// EU Digital Identity Wallet credential revocation checking.
package statuslist

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// StatusListVerifier checks credential revocation status using Status List 2021.
type StatusListVerifier struct {
	issuerKeys map[string]*ecdsa.PublicKey // keyed by issuer URL
	cache      *StatusListCache
	httpClient *http.Client
}

// StatusListCache caches decompressed status lists with TTL.
type StatusListCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedList // keyed by status list URI
	ttl     time.Duration
}

type cachedList struct {
	list      *StatusList
	fetchedAt time.Time
}

// StatusList represents a decompressed status list.
type StatusList struct {
	Issuer    string `json:"iss"`
	Subject   string `json:"sub"`
	Bits      int    `json:"bits"`
	Lst       string `json:"lst"` // compressed, base64url-encoded
	bitstring []byte               // decompressed
}

// NewStatusListVerifier creates a new status list verifier.
func NewStatusListVerifier() *StatusListVerifier {
	return &StatusListVerifier{
		issuerKeys: make(map[string]*ecdsa.PublicKey),
		cache: &StatusListCache{
			entries: make(map[string]*cachedList),
			ttl:     5 * time.Minute,
		},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// AddIssuerKey registers the public key for a status list issuer.
func (v *StatusListVerifier) AddIssuerKey(issuer string, key *ecdsa.PublicKey) {
	v.issuerKeys[issuer] = key
}

// CheckStatus verifies whether a credential at the given index is valid.
// Returns true if valid (not revoked), false if revoked.
func (v *StatusListVerifier) CheckStatus(ctx context.Context, statusURI string, idx int) (bool, error) {
	list, err := v.getOrFetch(ctx, statusURI)
	if err != nil {
		return false, fmt.Errorf("fetch status list: %w", err)
	}

	// Check the bit at position idx.
	// Bits are packed MSB-first within each byte.
	byteIdx := idx / 8
	bitIdx := 7 - (idx % 8)

	if byteIdx >= len(list.bitstring) {
		return false, fmt.Errorf("status index %d out of range (list size: %d bits)", idx, len(list.bitstring)*8)
	}

	// 0 = valid, 1 = revoked.
	isRevoked := (list.bitstring[byteIdx]>>bitIdx)&1 == 1
	return !isRevoked, nil
}

// getOrFetch returns the cached status list or fetches and verifies it.
func (v *StatusListVerifier) getOrFetch(ctx context.Context, uri string) (*StatusList, error) {
	// Check cache.
	v.cache.mu.RLock()
	if entry, ok := v.cache.entries[uri]; ok {
		if time.Since(entry.fetchedAt) < v.cache.ttl {
			v.cache.mu.RUnlock()
			return entry.list, nil
		}
	}
	v.cache.mu.RUnlock()

	// Fetch from issuer.
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status list endpoint returned %d", resp.StatusCode)
	}

	// Read the JWT status list.
	var statusListJWT string
	if err := json.NewDecoder(resp.Body).Decode(&statusListJWT); err != nil {
		// Try as raw JWT string (not JSON-wrapped).
		body, _ := io.ReadAll(resp.Body)
		statusListJWT = string(body)
	}

	// Verify the JWT signature.
	token, err := jwt.Parse(statusListJWT, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		// Get the issuer from claims to look up the key.
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid claims")
		}
		issuer, _ := claims["iss"].(string)
		key, ok := v.issuerKeys[issuer]
		if !ok {
			return nil, fmt.Errorf("unknown status list issuer: %s", issuer)
		}
		return key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("status list signature invalid: %w", err)
	}

	// Extract claims.
	claims := token.Claims.(jwt.MapClaims)
	lst, _ := claims["lst"].(string)
	bits, _ := claims["bits"].(float64)
	issuer, _ := claims["iss"].(string)
	subject, _ := claims["sub"].(string)

	// Decompress the bitstring.
	raw, err := base64.RawURLEncoding.DecodeString(lst)
	if err != nil {
		return nil, fmt.Errorf("decode lst: %w", err)
	}

	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}
	defer zr.Close()

	bitstring, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("read decompressed data: %w", err)
	}

	list := &StatusList{
		Issuer:    issuer,
		Subject:   subject,
		Bits:      int(bits),
		Lst:       lst,
		bitstring: bitstring,
	}

	// Cache the result.
	v.cache.mu.Lock()
	v.cache.entries[uri] = &cachedList{list: list, fetchedAt: time.Now()}
	v.cache.mu.Unlock()

	return list, nil
}

// SetCacheTTL sets the cache time-to-live for status lists.
func (v *StatusListVerifier) SetCacheTTL(ttl time.Duration) {
	v.cache.ttl = ttl
}
```

### 8.5 Privacy Properties

The Status List 2021 design provides strong privacy guarantees:

1. **k-anonymity**: The verifier fetches the entire list (potentially
   containing millions of credentials). The issuer cannot determine
   which specific credential is being checked.

2. **No phone-home**: Verification is offline once the list is cached.
   The verifier never contacts the issuer about a specific credential.

3. **Cached freshness**: The verifier caches the list with a TTL
   (typically 5-15 minutes). Within the TTL window, all checks are
   purely local.

4. **Compressed delivery**: The list is ZLIB-compressed. For 131K
   credentials with mostly-valid status, the compressed payload is
   a few hundred bytes.

### 8.6 Status Values Beyond Revocation

The `bits` field supports richer status models:

| Bits | Values | Statuses |
|------|--------|----------|
| 1 | 2 | 0=valid, 1=revoked |
| 2 | 4 | 0=valid, 1=revoked, 2=suspended, 3=pending |
| 4 | 16 | Application-defined |
| 8 | 256 | Full application-defined |

For EU PID, 1-bit (valid/revoked) is the standard. Suspended status
(2-bit) may be used for temporary holds during fraud investigations.

---

## 9. Cross-Border Interoperability

### 9.1 The Challenge

A German citizen traveling to France should be able to use their
German-issued wallet (containing a German PID) with a French service
provider. Similarly, a Dutch business should accept a Spanish
professional license credential presented via a Spanish wallet.

This requires:

1. **Technical interoperability**: All wallets speak the same protocol
   (OID4VP, SIOPv2) and use the same credential formats (SD-JWT VC).
2. **Trust interoperability**: An RP in France trusts a PID issued by
   the German PID Provider, verified through the EU Trust List.
3. **Semantic interoperability**: The attribute `family_name` means the
   same thing regardless of issuing country.
4. **Legal interoperability**: eIDAS 2.0 Article 6 mandates mutual
   recognition — French RPs cannot refuse German PID.

### 9.2 Interoperability Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                  EU-Wide Interoperability Layer                   │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐│
│   │ Germany     │ │ France      │ │ Netherlands │ │ Spain     ││
│   │ Wallet      │ │ Wallet      │ │ Wallet      │ │ Wallet    ││
│   │ (BundID)    │ │(France ID)  │ │ (DigiD)     │ │ (Cl@ve)   ││
│   └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └─────┬─────┘│
│          │                │                │               │     │
│   ┌──────┴──────┐ ┌──────┴──────┐ ┌──────┴──────┐ ┌─────┴─────┐│
│   │ German      │ │ French      │ │ Dutch       │ │ Spanish   ││
│   │ PID Provider│ │ PID Provider│ │ PID Provider│ │ PID Prov. ││
│   └──────┬──────┘ └──────┬──────┘ └──────┬──────┘ └─────┬─────┘│
│          │                │                │               │     │
│          └────────────────┼────────────────┼───────────────┘     │
│                           │                │                     │
│          ┌────────────────┴────────────────┴────────────┐       │
│          │          EU Trust List (EUTL)               │       │
│          │    (Aggregated by European Commission)       │       │
│          │                                              │       │
│          │  - All member states' PID Provider keys      │       │
│          │  - All QTSP certificates                      │       │
│          │  - National RP Registry keys                  │       │
│          └──────────────────────────────────────────────┘       │
│                                                                  │
│   Any RP in any member state can verify credentials from        │
│   any wallet in any member state through the EUTL.              │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 9.3 Cross-Border Flow: German Wallet → French RP

```
 German Citizen           French RP (Bank)          German PID Provider
 (German Wallet)                                     (Bundesdruckerei)
      │                       │                              │
      │ 1. Access French      │                              │
      │    bank online        │                              │
      ├──────────────────────>│                              │
      │                       │                              │
      │ 2. Bank sends OID4VP  │                              │
      │    authorization      │                              │
      │    request (signed    │                              │
      │    by French RP cert) │                              │
      │<──────────────────────┤                              │
      │                       │                              │
      │ 3. Wallet verifies    │                              │
      │    French RP cert     │                              │
      │    against French RP  │                              │
      │    Registry → EUTL    │                              │
      │                       │                              │
      │ 4. Wallet presents    │                              │
      │    German PID         │                              │
      │    (SD-JWT VC)        │                              │
      ├──────────────────────>│                              │
      │                       │                              │
      │                       │ 5. RP verifies SD-JWT:       │
      │                       │    - Signature valid?        │
      │                       │    - Look up German PID      │
      │                       │      Provider key in EUTL    │
      │                       │      → Found & verified      │
      │                       │                              │
      │                       │ 6. RP checks status list     │
      │                       │    (hosted by German PID     │
      │                       │     Provider)                │
      │                       ├─────────────────────────────>│
      │                       │<─────────────────────────────┤
      │                       │    Status: idx=42 → 0 (ok)   │
      │                       │                              │
      │                       │ 7. RP creates session        │
      │                       │    (PID claims → user)       │
      │  8. Access granted    │                              │
      │<──────────────────────┤                              │
```

Key steps for interoperability:
- Step 3: The wallet resolves the French RP's registration through the
  French RP Registry, which is anchored in the EUTL.
- Step 5: The French RP resolves the German PID Provider's key through
  the EUTL. Both national trust lists converge on the same EUTL root.
- Step 6: The status list is hosted by the credential issuer (German
  PID Provider). Any RP worldwide can fetch it.

### 9.4 EUTL Resolution

The EU Trust List is a hierarchical X.509 PKI:

```
European Commission Root CA
  ├── German National Trust List (signed by EC root)
  │     ├── German PID Provider certificate
  │     ├── German QTSP certificates
  │     └── German RP Registry signing key
  ├── French National Trust List
  │     ├── French PID Provider certificate
  │     └── French RP Registry signing key
  ├── Dutch National Trust List
  │     └── ...
  └── ... (27 member states)
```

Relying parties fetch the EUTL (or their national subset) at startup
and cache it with periodic refresh. Key resolution follows the chain:

1. RP receives a credential from `iss: https://pid-provider.bund.de`.
2. RP looks up `bund.de` in the German national trust list.
3. The German list is signed by the EC root → verify the chain.
4. Extract the PID Provider's public key → verify the credential signature.

### 9.5 Conformance Testing

The European Commission operates a **conformance testing infrastructure**
that validates wallet implementations against the ARF. National wallets
must pass conformance tests before they are certified.

The conformance suite covers:
- PID issuance (OID4VCI pre-authorized + authorization code flows)
- PID presentation (OID4VP with various presentation definitions)
- SIOPv2 self-issued authentication
- Status list verification
- Cross-border scenario testing (wallet from country A, RP from country B)
- Selective disclosure verification (all disclosed claims verify, no
  undisclosed claims leak)

### 9.6 Known Interoperability Challenges

| Challenge | Status | Mitigation |
|-----------|--------|------------|
| DID method fragmentation | Ongoing | EU ARF recommends `did:web` and `did:jwk` |
| Attribute naming variations | Mitigated | ARF Annex VII defines standard names |
| mDL coexistence | Active | ISO 18013-5 mDL support mandated alongside SD-JWT |
| Different crypto per country | Mitigated | ARF mandates ES256 (P-256) as baseline |
| Network latency for status checks | Operational | Cache with TTL; offline grace period |
| RP Registry lookup latency | Operational | Cache with periodic refresh |

---

## 10. GGID as Relying Party Design

### 10.1 Current GGID OAuth Architecture

GGID's OAuth service (`services/oauth/`) currently supports:

| Component | File | Function |
|-----------|------|----------|
| Authorization Server | `internal/server/server.go` | `/oauth/authorize`, `/oauth/token` |
| Client Management | `internal/service/oauth_service.go` | OAuth client registration (RFC 7591/7592) |
| Discovery | `internal/domain/models.go` | `OIDCDiscoveryConfig` struct |
| JWKS | `internal/server/server.go` | `/oauth/jwks` endpoint |
| Key Management | `internal/domain/models.go` | `KeyProvider` interface |
| Token Introspection | `internal/server/server.go` | `/oauth/introspect` |
| CIBA | `internal/service/ciba.go` | Client-initiated backchannel auth |
| DPoP | `internal/service/dpop.go` | Demonstrating Proof-of-Possession |
| JAR | `internal/service/jar_mtls.go` | JWT-Secured Authorization Requests |
| PAR | `internal/service/par.go` | Pushed Authorization Requests |
| Key Rotation | `internal/service/key_rotation.go` | RotatingKeyProvider with grace period |

The service uses `domain.KeyProvider` for JWT signing:

```go
// From services/oauth/internal/domain/models.go
type KeyProvider interface {
    PublicKey() *rsa.PublicKey
    PrivateKey() *rsa.PrivateKey
    KeyID() string
}
```

This interface can be extended to support ECDSA keys (needed for SD-JWT
and OID4VP, which primarily use P-256).

### 10.2 What GGID Needs to Add

To support EUDI Wallet as a Relying Party, GGID needs:

```
┌──────────────────────────────────────────────────────────────────┐
│                    GGID as EUDI Wallet RP                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Existing OAuth Service         New Components                   │
│  ┌─────────────────┐           ┌──────────────────────────────┐ │
│  │ /oauth/authorize│           │ /oid4vp/request/:id          │ │
│  │ /oauth/token    │           │   (generates signed authz    │ │
│  │ /oauth/jwks     │           │    request for wallet)       │ │
│  │ /.well-known/   │           │                              │ │
│  │  openid-config  │           │ /oid4vp/response             │ │
│  └─────────────────┘           │   (receives VP token)        │ │
│                                │                              │ │
│  ┌─────────────────┐           │ /siop/request/:id            │ │
│  │ Auth Service    │           │   (SIOPv2 authz request)     │ │
│  │ (JWT issuance)  │           │                              │ │
│  └─────────────────┘           │ /siop/callback               │ │
│                                │   (receives ID token)        │ │
│                                └──────────────────────────────┘ │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              EUDI Wallet RP Module                       │   │
│  │                                                          │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐  │   │
│  │  │ OID4VP       │  │ SIOPv2       │  │ RP Registry   │  │   │
│  │  │ Verifier     │  │ Verifier     │  │ Client        │  │   │
│  │  │              │  │              │  │               │  │   │
│  │  │ - SD-JWT VC  │  │ - DID resolve│  │ - Fetch RP    │  │   │
│  │  │   verify     │  │ - Key verify │  │   attestation │  │   │
│  │  │ - Disclosures│  │ - Nonce check│  │ - Verify sig  │  │   │
│  │  │ - KB-JWT     │  │ - Claims     │  │ - Cache       │  │   │
│  │  └──────────────┘  └──────────────┘  └───────────────┘  │   │
│  │                                                          │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────────┐  │   │
│  │  │ Status List  │  │ SD-JWT       │  │ PID Mapping   │  │   │
│  │  │ Verifier     │  │ Parser       │  │ Service       │  │   │
│  │  │              │  │              │  │               │  │   │
│  │  │ - Fetch list │  │ - Parse VC   │  │ - PID → user  │  │   │
│  │  │ - Check bit  │  │ - Verify sig │  │ - Create      │  │   │
│  │  │ - Cache TTL  │  │ - Extract    │  │   session     │  │   │
│  │  └──────────────┘  └──────────────┘  └───────────────┘  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                 Shared Infrastructure                    │   │
│  │                                                          │   │
│  │  ┌──────────┐  ┌──────────┐  ┌────────────────────────┐ │   │
│  │  │ EU Trust │  │ Issuer   │  │ Trusted Issuer         │ │   │
│  │  │ List     │  │ Metadata │  │ Registry               │ │   │
│  │  │ Cache    │  │ Cache    │  │ (per-tenant config)    │ │   │
│  │  └──────────┘  └──────────┘  └────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 10.3 Registration Flow

Before GGID can accept wallet presentations, it must register as a
Relying Party in the national RP Registry:

```go
// RP registration is done out-of-band (admin operation, not per-request).
// This code generates the registration request and CSR.

type RPRegistrationRequest struct {
	LegalName         string   `json:"legal_name"`
	LegalID           string   `json:"legal_id"`
	Sector            string   `json:"sector"`
	Purpose           string   `json:"purpose"`
	AttributesNeeded  []string `json:"attributes_needed"`
	ClientID          string   `json:"client_id"` // GGID OAuth client_id
	RedirectURIs      []string `json:"redirect_uris"`
	PublicKeyJWK      map[string]any `json:"public_key_jwk"`
}

func GenerateRPKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func CreateRegistrationRequest(
	legalName, legalID, sector, purpose string,
	attrs []string,
	rpClientID string,
	redirectURIs []string,
	privKey *ecdsa.PrivateKey,
) (*RPRegistrationRequest, error) {
	pubJWK, err := publicKeyToJWK(&privKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &RPRegistrationRequest{
		LegalName:        legalName,
		LegalID:          legalID,
		Sector:           sector,
		Purpose:          purpose,
		AttributesNeeded: attrs,
		ClientID:         rpClientID,
		RedirectURIs:     redirectURIs,
		PublicKeyJWK:     pubJWK,
	}, nil
}
```

The registration is submitted to the national RP Registry API. Once
approved, GGID receives a signed attestation (JWT) that it includes in
authorization requests to wallets.

### 10.4 Credential Request Flow

When a user selects "Sign in with EU Wallet" on the GGID login page:

```
 User Browser                GGID Gateway              GGID OAuth Service
      │                           │                          │
      │ 1. Click "Sign in with    │                          │
      │    EU Digital Wallet"     │                          │
      ├──────────────────────────>│                          │
      │                           │                          │
      │                           │ 2. Create OID4VP         │
      │                           │    authorization request │
      │                           ├─────────────────────────>│
      │                           │                          │
      │                           │    3. Generate nonce,    │
      │                           │       build presentation │
      │                           │       definition, sign   │
      │                           │       with RP key        │
      │                           │<─────────────────────────┤
      │                           │                          │
      │ 4. Redirect to wallet     │                          │
      │    (QR code / deep link)  │                          │
      │<──────────────────────────┤                          │
      │                           │                          │
      │ 5. User opens wallet,     │                          │
      │    scans QR / clicks      │                          │
      │    deep link              │                          │
      │                           │                          │
      │ 6. Wallet verifies GGID   │                          │
      │    RP certificate, shows  │                          │
      │    consent                │                          │
      │                           │                          │
      │ 7. User approves          │                          │
      │    (biometric)            │                          │
      │                           │                          │
      │ 8. Wallet POSTs VP token  │                          │
      │    to GGID callback       │                          │
      ├──────────────────────────────────────────────────────>│
      │                           │                          │
      │                           │ 9. Verify VP token:      │
      │                           │    - SD-JWT signature    │
      │                           │    - Disclosures         │
      │                           │    - KB-JWT holder bind  │
      │                           │    - Status list check   │
      │                           │    - Issuer trust (EUTL) │
      │                           │                          │
      │                           │ 10. Map PID claims →     │
      │                           │     GGID user            │
      │                           │     (create or find)     │
      │                           │                          │
      │                           │ 11. Issue GGID session   │
      │                           │     JWT / auth code      │
      │ 12. Redirect with session │                          │
      │<──────────────────────────┤                          │
      │                           │                          │
```

### 10.5 Verification Flow Implementation

```go
// Package wallet implements the EUDI Wallet Relying Party integration
// for the GGID OAuth service.
package wallet

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
)

// WalletAuthHandler orchestrates the full wallet authentication flow.
type WalletAuthHandler struct {
	oid4vpVerifier  *OID4VPVerifier
	siopVerifier    *SIOPVerifier
	statusVerifier  *StatusListVerifier
	rpRegistry      *RPRegistryClient
	pidMapper       *PIDMapper
	keyProvider     domain.KeyProvider
	issuer          string
}

// WalletAuthResult is the outcome of a successful wallet authentication.
type WalletAuthResult struct {
	UserID       uuid.UUID
	TenantID     uuid.UUID
	PIDClaims    map[string]any
	Issuer       string
	IssuerCountry string
	AuthMethod   string // "oid4vp", "siopv2", "combined"
	ExpiresAt    time.Time
}

// HandlePresentation processes a VP token from the wallet and creates
// a GGID authentication session.
func (h *WalletAuthHandler) HandlePresentation(
	ctx context.Context,
	tenantID uuid.UUID,
	vpToken string,
	presentationSubmission string,
	sessionNonce string,
) (*WalletAuthResult, error) {
	// Step 1: Parse the presentation submission to get the format.
	submission, err := parsePresentationSubmission(presentationSubmission)
	if err != nil {
		return nil, errors.BadRequest("invalid presentation submission")
	}

	// Step 2: Verify the VP token based on format.
	var result *OID4VPVerificationResult
	switch submission.DescriptorMap[0].Format {
	case "dc+sd-jwt":
		result, err = h.oid4vpVerifier.VerifyPresentation(ctx, vpToken, sessionNonce)
		if err != nil {
			return nil, errors.Unauthorized(fmt.Sprintf("VP verification failed: %v", err))
		}
	default:
		return nil, errors.BadRequest("unsupported VP format: " + submission.DescriptorMap[0].Format)
	}

	// Step 3: Check credential status.
	if result.StatusURI != "" {
		valid, err := h.statusVerifier.CheckStatus(ctx, result.StatusURI, result.StatusIndex)
		if err != nil {
			return nil, errors.Internal("status check failed", err)
		}
		if !valid {
			return nil, errors.Unauthorized("credential is revoked")
		}
	}

	// Step 4: Map PID claims to GGID user.
	pidResult, err := h.pidMapper.MapOrCreateUser(ctx, tenantID, result)
	if err != nil {
		return nil, err
	}

	return &WalletAuthResult{
		UserID:        pidResult.UserID,
		TenantID:      tenantID,
		PIDClaims:     result.Claims,
		Issuer:        result.Issuer,
		AuthMethod:    "oid4vp",
		ExpiresAt:     result.ExpiresAt,
	}, nil
}

// CreateAuthzRequest generates an OID4VP authorization request for the wallet.
func (h *WalletAuthHandler) CreateAuthzRequest(
	ctx context.Context,
	tenantID uuid.UUID,
	requestedAttributes []string,
	purpose string,
) (string, error) {
	// Generate a nonce for this session.
	nonce, err := generateNonce()
	if err != nil {
		return "", errors.Internal("generate nonce", err)
	}

	// Store nonce in session for later verification.
	if err := h.storeSessionNonce(ctx, tenantID, nonce); err != nil {
		return "", errors.Internal("store nonce", err)
	}

	// Build the presentation definition.
	definition := buildPresentationDefinition(requestedAttributes, purpose)

	// Build and sign the authorization request.
	authzReq := &PresentationRequest{
		DefinitionID:    "ggid-eudi-wallet-auth",
		InputDescriptors: definition,
		Nonce:           nonce,
		ResponseURI:     fmt.Sprintf("%s/oid4vp/response", h.issuer),
	}

	jwtReq, err := h.oid4vpVerifier.CreateAuthorizationRequest(authzReq)
	if err != nil {
		return "", errors.Internal("create authz request", err)
	}

	return jwtReq, nil
}

// PIDMapper maps verified PID claims to GGID users.
type PIDMapper struct {
	// Dependencies on identity service, user repository, etc.
}

// PIDMappingResult is the result of mapping PID to a GGID user.
type PIDMappingResult struct {
	UserID    uuid.UUID
	IsNewUser bool
}

// MapOrCreateUser finds an existing user matching the PID or creates a new one.
func (m *PIDMapper) MapOrCreateUser(
	ctx context.Context,
	tenantID uuid.UUID,
	verification *OID4VPVerificationResult,
) (*PIDMappingResult, error) {
	// Extract PID claims for user lookup.
	familyName, _ := verification.Claims["family_name"].(string)
	givenName, _ := verification.Claims["given_name"].(string)

	if familyName == "" || givenName == "" {
		return nil, errors.BadRequest("PID must contain family_name and given_name")
	}

	// Look up user by wallet holder key (for returning users).
	// In production, this would query the identity service.
	holderKey := verification.HolderKey

	// If no existing user, create one from PID claims.
	// The user's identity is verified by the PID — no password needed.
	userID := uuid.New() // Simplified — production would check for existing user

	return &PIDMappingResult{
		UserID:    userID,
		IsNewUser: true,
	}, nil
}
```

### 10.6 HTTP Endpoints

New endpoints to add to the OAuth server's `buildHandler`:

```go
// In services/oauth/internal/server/server.go, add to buildHandler():

// EUDI Wallet endpoints
mux.HandleFunc("/oid4vp/request/", func(w http.ResponseWriter, r *http.Request) {
	// GET /oid4vp/request/:tenantID
	// Returns a signed OID4VP authorization request for the wallet.
	requestID := r.URL.Path[len("/oid4vp/request/"):]
	// ... generate authz request, return as JWT
})

mux.HandleFunc("/oid4vp/response", func(w http.ResponseWriter, r *http.Request) {
	// POST /oid4vp/response
	// Receives VP token from the wallet.
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	// ... parse response, verify VP token, create session
})

mux.HandleFunc("/siop/request/", func(w http.ResponseWriter, r *http.Request) {
	// GET /siop/request/:tenantID
	// Returns a signed SIOPv2 authorization request.
})

mux.HandleFunc("/siop/callback", func(w http.ResponseWriter, r *http.Request) {
	// POST /siop/callback
	// Receives self-issued ID token from the wallet.
})

// Extend discovery endpoint to advertise wallet support
// In GetDiscoveryConfig():
config["vp_formats_supported"] = map[string]any{
	"dc+sd-jwt": map[string]any{
		"sd-jwt_alg_values": []string{"ES256", "RS256"},
		"kb-jwt_alg_values": []string{"ES256"},
	},
}
config["request_uri_parameter_supported"] = true
config["require_pushed_authorization_requests"] = true
```

### 10.7 Extending KeyProvider for ECDSA

The current `domain.KeyProvider` interface only supports RSA. SD-JWT
and EUDI Wallet operations prefer ECDSA (P-256). The interface needs
extension:

```go
// Extended KeyProvider supporting both RSA and ECDSA.
type KeyProvider interface {
	// Existing RSA support
	PublicKey() *rsa.PublicKey
	PrivateKey() *rsa.PrivateKey
	KeyID() string

	// New ECDSA support for wallet operations
	ECDSAPublicKey() *ecdsa.PublicKey
	ECDSAPrivateKey() *ecdsa.PrivateKey
	ECDSAKeyID() string

	// Algorithm
	SigningAlgorithm() string // "RS256" or "ES256"
}
```

Alternatively, a separate `WalletKeyProvider` interface can be defined
to avoid breaking the existing RSA-only interface:

```go
// WalletKeyProvider supplies ECDSA keys for EUDI Wallet operations.
type WalletKeyProvider interface {
	PublicKey() *ecdsa.PublicKey
	PrivateKey() *ecdsa.PrivateKey
	KeyID() string
}
```

### 10.8 Database Schema

```sql
-- Trusted issuers (PID Providers, QTSPs)
CREATE TABLE wallet_trusted_issuers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    issuer_url      TEXT NOT NULL,
    issuer_name     TEXT NOT NULL,
    issuer_country  TEXT NOT NULL,   -- ISO 3166-1 alpha-2
    trust_status    TEXT NOT NULL DEFAULT 'qualified', -- qualified, trusted
    jwks_uri        TEXT NOT NULL,
    public_keys     JSONB NOT NULL DEFAULT '{}',  -- cached keys
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, issuer_url)
);

-- Wallet authentication sessions (nonce tracking)
CREATE TABLE wallet_auth_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    nonce           TEXT NOT NULL UNIQUE,
    presentation_definition JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, completed, expired
    user_id         UUID,                  -- set after successful verification
    vp_token_hash   TEXT,                  -- hash of received VP token
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL
);

-- RP registration metadata (per tenant)
CREATE TABLE wallet_rp_registrations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL UNIQUE,
    rp_client_id    TEXT NOT NULL,         -- GGID client_id for wallet
    rp_legal_name   TEXT NOT NULL,
    rp_legal_id     TEXT NOT NULL,
    rp_sector       TEXT NOT NULL,
    rp_purpose      TEXT NOT NULL,
    rp_attributes   TEXT[] NOT NULL DEFAULT '{}',
    rp_public_key   JSONB NOT NULL,        -- JWK
    attestation_jwt TEXT,                  -- from national RP registry
    registered      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Credential presentation records (audit trail)
CREATE TABLE wallet_presentations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    issuer          TEXT NOT NULL,
    issuer_country  TEXT,
    holder_key_hash TEXT NOT NULL,         -- SHA-256 of holder key
    claims_json     JSONB NOT NULL,        -- disclosed claims only
    auth_method     TEXT NOT NULL,         -- oid4vp, siopv2, combined
    status_verified BOOLEAN NOT NULL DEFAULT false,
    presented_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallet_presentations_tenant ON wallet_presentations(tenant_id);
CREATE INDEX idx_wallet_presentations_user ON wallet_presentations(user_id);
CREATE INDEX idx_wallet_auth_sessions_nonce ON wallet_auth_sessions(nonce);
```

---

## 11. Gap Analysis and Recommendations

### 11.1 Current State Assessment

| Capability | Status | Notes |
|-----------|--------|-------|
| OAuth 2.0 Authorization Code | **Implemented** | `services/oauth/internal/service/oauth_service.go` |
| OIDC Discovery | **Implemented** | `OIDCDiscoveryConfig` in domain models |
| JWT Signing (RSA) | **Implemented** | `KeyProvider` interface, `RotatingKeyProvider` |
| PKCE | **Implemented** | S256 enforcement for public clients |
| DPoP | **Implemented** | `services/oauth/internal/service/dpop.go` |
| JAR (JWT Authz Requests) | **Implemented** | `jar_mtls.go` |
| PAR | **Implemented** | `par.go` |
| CIBA | **Implemented** | `ciba.go` |
| OID4VP | **Not started** | No wallet presentation support |
| SIOPv2 | **Not started** | No self-issued OP support |
| SD-JWT VC | **Not started** | No selective disclosure parsing |
| Status List 2021 | **Not started** | No revocation checking |
| RP Registry | **Not started** | No wallet trust framework integration |
| ECDSA Key Support | **Partial** | Interface is RSA-only |
| EU Trust List | **Not started** | No EUTL resolution/caching |

### 11.2 Action Items

#### Action 1: Implement SD-JWT VC Parser and Verifier
**Effort**: ~2 weeks
**Priority**: P0 (foundation for all wallet features)

Implement the SD-JWT parsing, signature verification, disclosure
reconstruction, and key binding JWT verification as described in
Section 7.6. This is the foundation — without SD-JWT support, no
wallet credential can be verified.

Deliverables:
- `pkg/sdjwt/` package with `ParseAndVerify`, `verifyDisclosures`,
  `verifyKBJWT` functions
- Support for ES256 (P-256) and RS256 algorithms
- Unit tests with known SD-JWT test vectors from the EUDI conformance suite
- Integration with the existing `domain.KeyProvider` pattern

#### Action 2: Add OID4VP and SIOPv2 Endpoints
**Effort**: ~3 weeks
**Priority**: P0 (core wallet authentication)

Add the `/oid4vp/request/:id`, `/oid4vp/response`, `/siop/request/:id`,
and `/siop/callback` endpoints to the OAuth server. Implement the
verifier logic as described in Sections 4.5 and 5.6.

Deliverables:
- `services/oauth/internal/wallet/` package with `OID4VPVerifier` and
  `SIOPVerifier`
- Presentation definition builder API
- Nonce generation and session storage
- PID claim → user mapping (integrate with identity service)
- HTTP handlers wired into `buildHandler`
- Extend `OIDCDiscoveryConfig` to advertise wallet support

#### Action 3: Implement Status List 2021 Verification
**Effort**: ~1 week
**Priority**: P1 (required for production revocation)

Implement the status list fetcher, decompressor, and bit checker
as described in Section 8.4. This prevents acceptance of revoked
credentials.

Deliverables:
- `pkg/statuslist/` package with `StatusListVerifier`
- ZLIB decompression + base64url decoding
- Cache with configurable TTL (default 5 min)
- JWT signature verification for status list authenticity
- Configurable fail-open/fail-closed behavior on fetch failure

#### Action 4: Extend KeyProvider for ECDSA
**Effort**: ~3 days
**Priority**: P1 (SD-JWT and wallet ops require P-256)

The current `domain.KeyProvider` interface (in
`services/oauth/internal/domain/models.go`) only exposes RSA keys.
SD-JWT VC verification, OID4VP authorization request signing, and
SIOPv2 all use ECDSA P-256.

Deliverables:
- Add `ECDSAPublicKey()`, `ECDSAPrivateKey()`, `ECDSAKeyID()` to
  `KeyProvider` (or define a separate `WalletKeyProvider`)
- Generate ECDSA P-256 keypair at startup alongside RSA keys
- Extend JWKS endpoint to serve both RSA and EC keys
- Update `RotatingKeyProvider` to rotate ECDSA keys

#### Action 5: Implement Trusted Issuer Registry and EUTL Integration
**Effort**: ~2 weeks
**Priority**: P2 (required for cross-border scenarios)

Build a per-tenant trusted issuer registry that can be populated from
the EU Trust List. Each tenant configures which issuers (PID Providers,
QTSPs) it accepts credentials from.

Deliverables:
- `wallet_trusted_issuers` table with per-tenant configuration
- Admin API for adding/removing trusted issuers (`POST /api/v1/wallet/issuers`)
- EUTL fetcher and parser (X.509 / XML trust list format)
- Periodic refresh of issuer public keys (background job)
- Console UI for trusted issuer management

### 11.3 Implementation Roadmap

```
Phase 1 (Weeks 1-3): Foundation
  ┌─────────────────────────────────────────────────┐
  │ Action 1: SD-JWT Parser        (Weeks 1-2)      │
  │ Action 4: ECDSA KeyProvider    (Week 3)         │
  └─────────────────────────────────────────────────┘
         │
         ▼
Phase 2 (Weeks 4-7): Core Wallet Auth
  ┌─────────────────────────────────────────────────┐
  │ Action 2: OID4VP + SIOPv2       (Weeks 4-6)     │
  │ Action 3: Status List 2021     (Week 7)         │
  └─────────────────────────────────────────────────┘
         │
         ▼
Phase 3 (Weeks 8-10): Trust & Interop
  ┌─────────────────────────────────────────────────┐
  │ Action 5: Trusted Issuer Registry (Weeks 8-9)   │
  │ Console UI for wallet config     (Week 10)      │
  │ Integration tests with EUDI ref impl            │
  └─────────────────────────────────────────────────┘
         │
         ▼
Phase 4 (Weeks 11-12): Polish & Hardening
  ┌─────────────────────────────────────────────────┐
  │ Security review                                  │
  │ Conformance test suite                           │
  │ Documentation & tutorials                        │
  │ Production deployment guide                      │
  └─────────────────────────────────────────────────┘
```

### 11.4 Total Effort Estimate

| Phase | Duration | Team Size | Output |
|-------|----------|-----------|--------|
| Phase 1 | 3 weeks | 1 dev | SD-JWT + ECDSA keys |
| Phase 2 | 4 weeks | 1 dev | OID4VP + SIOPv2 + Status |
| Phase 3 | 3 weeks | 1 dev + 1 frontend | Trust registry + Console |
| Phase 4 | 2 weeks | 1 dev + 1 QA | Hardening + docs |
| **Total** | **12 weeks** | | **Full EUDI Wallet RP support** |

### 11.5 Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| eIDAS 2.0 timeline slips | Medium | Low | Design is spec-driven, not deadline-driven |
| Spec changes (OID4VP draft) | Medium | Medium | Pin to a specific draft version; update later |
| EUDI conformance test availability | High | Medium | Test against reference wallet independently |
| Cross-border interoperability gaps | Medium | High | Start with same-country flows; expand later |
| ECDSA migration breaks existing flows | Low | High | Keep RSA for existing OAuth flows; add ECDSA alongside |

---

## References

### Regulations and Standards

1. **eIDAS 2.0** — Regulation (EU) 2024/1183:
   https://eur-lex.europa.eu/eli/reg/2024/1183/oj

2. **EU Digital Identity Wallet Architecture Reference Framework (ARF)**:
   https://github.com/eu-digital-identity-wallet/architecture-and-reference-framework

3. **Implementing Regulation on PID** (Commission Implementing Regulation
   supplementing eIDAS 2.0):
   https://digital-strategy.ec.europa.eu/en/library/european-digital-identity-wallet

4. **RFC 9901 — SD-JWT (Selective Disclosure for JWTs)**:
   https://datatracker.ietf.org/doc/rfc9901/

5. **SD-JWT VC (draft-ietf-oauth-sd-jwt-vc)**:
   https://datatracker.ietf.org/doc/draft-ietf-oauth-sd-jwt-vc/

6. **OID4VP (OpenID for Verifiable Presentations)**:
   https://openid.net/specs/openid-4-verifiable-presentations-1_0.html

7. **SIOPv2 (Self-Issued OpenID Provider v2)**:
   https://openid.net/specs/openid-connect-self-issued-v2-1_0.html

8. **Token Status List (draft-ietf-oauth-status-list)**:
   https://datatracker.ietf.org/doc/draft-ietf-oauth-status-list/

9. **W3C Verifiable Credentials Data Model v2.0**:
   https://www.w3.org/TR/vc-data-model-2.0/

10. **DIF Presentation Exchange v2**:
    https://identity.foundation/presentation-exchange/

### Related GGID Research Documents

- [OID4VCI and Verifiable Credentials](oid4vci-and-verifiable-credentials.md) —
  Credential issuance flows, VC data model, OID4VCI protocol
- [OID4VP and Credential Presentation](oid4vp-and-credential-presentation.md) —
  Verifier flow, presentation definitions, selective disclosure, Digital
  Credentials API
- [Token Status List](token-status-list.md) — Full TSL protocol details,
  bitstring format, privacy properties
- [OIDC Federation](oidc-federation.md) — Federation trust models
- [FIDO2 Certification Guide](fido2-certification-guide.md) —
  WebAuthn/FIDO2 attestation verification

### EU Ecosystem Resources

- **EUDI Wallet Reference Implementation**:
  https://github.com/eu-digital-identity-wallet/eudi-reference-implementation

- **EUDI Wallet Conformance Testing**:
  https://github.com/eu-digital-identity-wallet/eudi-conformance-testing

- **EU Trust List Browser**:
  https://webgate.ec.europa.eu/tl-browser/

- **eIDAS 2.0 Official Page**:
  https://digital-strategy.ec.europa.eu/en/policies/european-digital-identity-wallet

### GGID Source Code References

- OAuth service main: `services/oauth/cmd/main.go`
- OAuth server (HTTP handlers): `services/oauth/internal/server/server.go`
- OAuth business logic: `services/oauth/internal/service/oauth_service.go`
- Domain models (KeyProvider, OAuthClient): `services/oauth/internal/domain/models.go`
- Key rotation: `services/oauth/internal/service/key_rotation.go`
- JAR/mTLS: `services/oauth/internal/service/jar_mtls.go`
- DPoP: `services/oauth/internal/service/dpop.go`
- CIBA: `services/oauth/internal/service/ciba.go`

---

> **Document version**: 1.0
> **Last updated**: 2025
> **Author**: GGID Security Research
> **Status**: Research / Design — not yet implemented
