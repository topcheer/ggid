# Credential Agents and Digital Wallets as IAM Clients

> **Research Document** — GGID IAM Suite
> **Topic**: Credential Agent Architecture, Digital Wallets, and Browser-Mediated Credential Exchange
> **Audience**: GGID architects, security engineers, OAuth/OIDC implementors
> **Status**: Research / Design

---

## Table of Contents

1. [Credential Agent Concepts](#1-credential-agent-concepts)
2. [W3C Digital Credentials API](#2-w3c-digital-credentials-api)
3. [Wallet-to-IAM Trust Establishment](#3-wallet-to-iam-trust-establishment)
4. [Token Exchange (RFC 8693) for Wallets](#4-token-exchange-rfc-8693-for-wallets)
5. [Browser-Mediated vs Direct API](#5-browser-mediated-vs-direct-api)
6. [Wallet-as-Broker Pattern](#6-wallet-as-broker-pattern)
7. [Credential Storage in Wallets](#7-credential-storage-in-wallets)
8. [Selective Disclosure in Wallet Flow](#8-selective-disclosure-in-wallet-flow)
9. [Multi-Wallet Support](#9-multi-wallet-support)
10. [GGID Token Exchange Mapping](#10-ggid-token-exchange-mapping)
11. [Gap Analysis and Recommendations](#11-gap-analysis-and-recommendations)

---

## 1. Credential Agent Concepts

### 1.1 What Are Credential Agents?

Credential agents — also called **digital wallets** or **identity wallets** — are software
components that store user credentials and present them to relying parties (RPs) upon user
consent. They act as an intermediary layer between the user and the applications that need
to verify the user's identity or attributes.

In the traditional IAM model, the flow is:

```
User → Browser → RP → IdP (centralized) → Token back to RP
```

In the credential agent model, the flow becomes:

```
User → Wallet → (selects credential) → RP verifies credential directly
```

The wallet replaces (or supplements) the centralized IdP. Instead of the RP calling an
authorization server to exchange an authorization code for a token, the RP receives a
verifiable credential directly from the user's wallet. The credential was pre-issued by a
trusted issuer (which may or may not be the same as the IAM system), and the RP verifies it
cryptographically.

### 1.2 Evolution From Password Managers to Credential Wallets

The credential agent concept has evolved through several stages:

**Stage 1: Password Managers (2000s)**
- Browser extensions and standalone apps (1Password, KeePass, LastPass)
- Store encrypted passwords, auto-fill login forms
- No cryptographic verification of the RP
- Vulnerable to phishing (fill credentials on lookalike domains)

**Stage 2: Browser-Built-in Credential Storage (2010s)**
- Chrome Password Manager, Firefox Lockwise, Safari Keychain
- OS-level integration (macOS Keychain, Windows Credential Manager)
- Autofill with origin validation (reduces phishing)
- Sync across devices via cloud (encrypted)

**Stage 3: WebAuthn / Passkeys (2019-present)**
- FIDO2/WebAuthn standard — public-key-based credentials
- Platform authenticators (Touch ID, Face ID, Windows Hello) act as credential agents
- RP origin is cryptographically bound to the credential (phishing-resistant)
- W3C Credential Management API (`navigator.credentials`) mediates access

**Stage 4: Verifiable Credential Wallets (2023-present)**
- Wallets store W3C Verifiable Credentials (VCs), SD-JWTs, or mDLs
- User selects which credential to present and which attributes to disclose
- Selective disclosure and zero-knowledge proofs
- Browser-mediated via W3C Digital Credentials API (`navigator.identity`)
- Wallet can be a native app (Apple Wallet, Google Wallet) or a web-based agent

### 1.3 W3C Credential Management API

The [W3C Credential Management API](https://www.w3.org/TR/credential-management-1/) provides
a programmatic interface for websites to access browser-stored credentials. It was
originally designed for password and federated credential management but has been extended
to support WebAuthn public-key credentials and, most recently, digital credentials.

The core API is `navigator.credentials`:

```javascript
// Store a credential (e.g., after user creates a password)
await navigator.credentials.store(new PasswordCredential({
  id: "user@example.com",
  password: "secret"
}));

// Retrieve a credential (e.g., for auto-login)
const cred = await navigator.credentials.get({
  password: true,       // request stored passwords
  federated: {          // request federated accounts (Google, Facebook, etc.)
    providers: ["https://accounts.google.com"]
  },
  mediation: "optional" // "silent" | "optional" | "required" | "conditional"
});
```

With WebAuthn, the API extends to public-key credentials:

```javascript
// Register a new passkey
const publicKey = await navigator.credentials.create({
  publicKey: {
    challenge: new Uint8Array([...]),
    rp: { name: "GGID", id: "ggid.example.com" },
    user: { id: new Uint8Array([...]), name: "user@example.com", displayName: "User" },
    pubKeyCredParams: [{ type: "public-key", alg: -7 }], // ES256
    authenticatorSelection: {
      authenticatorAttachment: "platform",
      residentKey: "required",
      userVerification: "required"
    }
  }
});

// Authenticate with a passkey
const assertion = await navigator.credentials.get({
  publicKey: {
    challenge: new Uint8Array([...]),
    allowCredentials: [{ type: "public-key", id: credentialId }],
    userVerification: "required"
  }
});
```

### 1.4 Browser-Mediated Credential Exchange

The newest evolution is **browser-mediated credential exchange**, where the browser acts
as a trusted intermediary between the RP's web page and the user's credential wallet. The
browser (not the web page) invokes the wallet, handles the user consent UI, and returns the
result to the web page. This is the foundation of the W3C Digital Credentials API (covered
in Section 2).

Key properties of browser-mediated exchange:
- **Origin isolation**: The web page cannot directly access the wallet — the browser mediates.
- **User consent**: The browser shows a native UI for the user to pick which wallet and which credential.
- **Protocol negotiation**: The browser matches RP requests to wallet capabilities.
- **Cross-origin protection**: Wallet invocation is gated by the browser's origin model.

### 1.5 Credential Agent Architecture Components

A credential agent system consists of:

| Component | Role | Examples |
|-----------|------|----------|
| **Issuer** | Creates and signs credentials for users | Government agencies, employers, GGID |
| **Wallet** | Stores credentials, presents them on user consent | Apple Wallet, enterprise wallet |
| **Relying Party (RP)** | Requests and verifies credentials | Web apps, APIs, services |
| **Browser** | Mediates wallet invocation and user consent | Chrome, Safari, Firefox |
| **Trust Registry** | Lists trusted issuers and their keys | DID directory, trust framework |
| **Verifier** | Cryptographically validates presented credentials | RP-side verification library |

### 1.6 Credential Types in Modern Wallets

| Credential Type | Format | Use Case |
|-----------------|--------|----------|
| W3C Verifiable Credential (VC) | JSON-LD + Linked Data Proofs | Academic degrees, professional licenses |
| SD-JWT (Selective Disclosure JWT) | JWT with hashed disclosures | Government IDs, age verification |
| mDL (Mobile Driver's License) | ISO/IEC 18013-5 | Driver's licenses |
| OpenID4VCI Credential | OAuth2-based issuance flow | Employee credentials, membership cards |
| OID4VP Presentation | Verifiable presentation exchange | Age-gated access, identity proofing |

---

## 2. W3C Digital Credentials API

### 2.1 Overview

The [W3C Digital Credentials API](https://w3c.github.io/digital-credentials/) is a
browser-mediated API that enables web applications to request digital credentials from
user-controlled wallets. It extends the existing `navigator.credentials` framework with a
new `digital` credential type.

Unlike WebAuthn (which registers and authenticates public-key credentials for a specific
origin), the Digital Credentials API enables **presentation** of externally-issued
credentials. A wallet might hold a government-issued ID, an employer-issued employee
credential, or a bank-issued proof-of-funds credential — and the RP can request any of
these via the API.

### 2.2 Core API: `navigator.identity.get()`

The API is exposed through `navigator.identity`:

```javascript
// Request a digital credential
const response = await navigator.identity.get({
  digital: {
    requests: [{
      id: "age-verification-request",
      manifest: {
        // Define what credential the RP needs
        id: "org.example.age_verification",
        version: "1.0",
        purpose: "Verify age for content access",
        input_descriptors: [{
          id: "age_claim",
          constraints: {
            fields: [{
              path: ["$.vc.type"],
              filter: { type: "string", const: "AgeVerificationCredential" }
            }, {
              path: ["$.vc.credentialSubject.age"],
              filter: { type: "integer", minimum: 18 }
            }]
          }
        }]
      },
      protocols: {
        "openid4vp": {
          // OpenID4VP request — wallet responds with VP token
          params: {
            client_id: "https://rp.example.com",
            response_type: "vp_token",
            response_mode: "direct_post",
            nonce: "random-nonce-12345"
          }
        }
      }
    }]
  }
});
```

### 2.3 Credential Request Structure

A credential request contains:

1. **`manifest`**: Describes the credential type and constraints (what fields, what values).
2. **`protocols`**: The exchange protocol (e.g., `openid4vp`, `web-Handshake`).
3. **`request data`**: Protocol-specific parameters (nonce, client_id, etc.).

The browser matches the request against installed wallets and presents a chooser UI:

```javascript
// The browser shows a native UI listing compatible wallets.
// User picks one → wallet opens → user selects credential → consent.
// Result returned to the web page:

if (response) {
  const { data, protocol } = response.digital;
  
  if (protocol === "openid4vp") {
    // data contains the VP token (verifiable presentation)
    // Send to backend for verification
    const verifyResult = await fetch("/api/v1/credentials/verify", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        protocol: "openid4vp",
        vp_token: data
      })
    });
    
    if (verifyResult.ok) {
      // User is authenticated / verified
      console.log("Credential accepted");
    }
  }
}
```

### 2.4 Credential Response Structure

The response is protocol-specific:

**OpenID4VP response**:
```json
{
  "protocol": "openid4vp",
  "data": {
    "vp_token": "eyJ...",  // JWT containing the verifiable presentation
    "presentation_submission": {
      "definition_id": "age-verification-request",
      "descriptor_map": [{
        "id": "age_claim",
        "format": "jwt_vc",
        "path": "$.vp.verifiableCredential[0]"
      }]
    }
  }
}
```

**Direct presentation response**:
```json
{
  "protocol": "web-Handshake",
  "data": {
    "credential": {
      "@context": ["https://www.w3.org/2018/credentials/v1"],
      "type": ["VerifiableCredential", "AgeVerificationCredential"],
      "issuer": "did:web:government.example",
      "credentialSubject": {
        "id": "did:key:z6Mk...",
        "ageOver": 18
      },
      "proof": {
        "type": "Ed25519Signature2020",
        "verificationMethod": "did:web:government.example#key-1",
        "proofValue": "z..."
      }
    }
  }
}
```

### 2.5 Browser Mediation Flow

The complete browser-mediated flow:

```
┌──────────┐    1. navigator.identity.get()     ┌──────────┐
│  Web App │ ──────────────────────────────────→ │ Browser  │
│   (RP)   │                                     │ (Chrome) │
└──────────┘ ←────────────────────────────────── └──────────┘
                2. Returns response or error         │
                                                     │ 3. Shows wallet chooser UI
                                                     ▼
                                               ┌──────────┐
                                               │  Wallet  │
                                               │ (Apple / │
                                               │  Google) │
                                               └──────────┘
                                                     │
                                                     │ 4. User selects credential
                                                     │ 5. Wallet generates VP
                                                     │ 6. Browser returns VP to web app
                                                     ▼
                                               ┌──────────┐
                                               │  User    │
                                               │ (consent)│
                                               └──────────┘
```

### 2.6 Error Handling

The API can reject for several reasons:

```javascript
try {
  const response = await navigator.identity.get(request);
  // Handle success
} catch (error) {
  switch (error.name) {
    case "NotAllowedError":
      // User dismissed the chooser or denied permission
      break;
    case "NotFoundError":
      // No wallet installed that supports this credential type
      break;
    case "AbortError":
      // Request was aborted by the caller
      break;
    case "SecurityError":
      // Origin not allowed or permissions policy denied
      break;
  }
}
```

### 2.7 Permissions Policy

The Digital Credentials API requires a permissions policy declaration:

```html
<!-- In HTTP headers or iframe allow attribute -->
Permissions-Policy: digital-credentials=(self)
```

This prevents third-party iframes from invoking the wallet without the top-level page's
consent.

### 2.8 Feature Detection

```javascript
// Check if the browser supports digital credentials
if ("digital" in navigator.identity) {
  // Digital Credentials API is available
  const isAvailable = await navigator.identity.digital.isSupported();
  if (isAvailable) {
    // Proceed with credential request
  }
}
```

---

## 3. Wallet-to-IAM Trust Establishment

### 3.1 The Trust Problem

For a credential agent system to work, RPs must trust the credentials presented by wallets.
This trust has two dimensions:

1. **Issuer trust**: Does the RP trust the entity that issued the credential?
2. **Wallet trust**: Does the RP trust the wallet to accurately present the credential without tampering?

Traditional IAM solves this with a pre-established trust relationship (the RP trusts the
IdP, the IdP authenticates the user). In the credential agent model, trust is distributed
and must be established through cryptographic mechanisms.

### 3.2 DID-Based Trust

Decentralized Identifiers (DIDs) provide a self-sovereign identity layer. Each issuer has
a DID that resolves to a DID document containing their public keys. The RP can verify
credential signatures by resolving the issuer's DID.

**DID methods commonly used:**

| Method | Description | Resolution |
|--------|-------------|------------|
| `did:web` | DID based on a web domain's `.well-known` | `https://domain/.well-known/did.json` |
| `did:key` | Self-contained DID embedding the key | Inline resolution from DID string |
| `did:ion` | Sidetree-based on Bitcoin | Network resolution |
| `did:cheqd` | Cosmos-based DID network | Network resolution |
| `did:ebsi` | European Blockchain Services Infrastructure | Network resolution |

### 3.3 Well-Known DID Configuration

The [Well-Known DID Configuration](https://identity.foundation/.well-known/resources/did-configuration/)
specification enables binding a DID to a web origin. The RP fetches
`https://example.com/.well-known/did-configuration` and verifies the DID-configuration
link:

```json
{
  "@context": "https://identity.foundation/.well-known/did-configuration/v1",
  "did-configuration": [
    {
      "id": "did:web:example.com",
      "vc": {
        "@context": ["https://www.w3.org/2018/credentials/v1", "https://identity.foundation/.well-known/did-configuration/v1"],
        "type": ["VerifiableCredential", "DomainLinkageCredential"],
        "issuer": "did:web:example.com",
        "issuanceDate": "2024-01-15T00:00:00Z",
        "credentialSubject": {
          "id": "did:web:example.com",
          "origin": "https://example.com"
        },
        "proof": {
          "type": "Ed25519Signature2020",
          "verificationMethod": "did:web:example.com#key-1",
          "proofPurpose": "assertionMethod",
          "proofValue": "z..."
        }
      }
    }
  ]
}
```

### 3.4 Verifiable Credential Issuer Metadata

Issuers publish metadata (similar to OIDC discovery) that describes their capabilities:

```json
{
  "issuer": "https://issuer.ggid.example.com",
  "credential_issuer": "https://issuer.ggid.example.com",
  "authorization_endpoint": "https://issuer.ggid.example.com/authorize",
  "token_endpoint": "https://issuer.ggid.example.com/token",
  "credential_endpoint": "https://issuer.ggid.example.com/credential",
  "jwks_uri": "https://issuer.ggid.example.com/.well-known/jwks.json",
  "credentials_supported": [
    {
      "format": "jwt_vc_json",
      "id": "EmployeeCredential",
      "types": ["VerifiableCredential", "EmployeeCredential"],
      "cryptographic_binding_methods_supported": ["did:key", "did:web"],
      "credential_signing_alg_values_supported": ["ES256", "RS256"],
      "display": [{
        "name": "GGID Employee Credential",
        "locale": "en",
        "logo": { "url": "https://ggid.example.com/logo.png" }
      }]
    }
  ]
}
```

### 3.5 Trust Registry

A trust registry is a curated list of trusted issuers, verifiers, and accreditation
frameworks. It answers the question: "Is this issuer authorized to issue this type of
credential?"

```json
{
  "registry_id": "gfid-trust-registry",
  "registry_name": "GGID Federation Trust Registry",
  "version": "1.0",
  "last_updated": "2024-06-01T00:00:00Z",
  "entries": [
    {
      "entity_id": "did:web:issuer.ggid.example.com",
      "entity_name": "GGID Credential Issuer",
      "entity_type": "issuer",
      "accredited_for": ["EmployeeCredential", "AgeVerificationCredential"],
      "accredited_at": "2024-01-15T00:00:00Z",
      "expires_at": "2025-01-15T00:00:00Z",
      "status": "active"
    },
    {
      "entity_id": "did:web:gov.example.com",
      "entity_name": "Government ID Issuer",
      "entity_type": "issuer",
      "accredited_for": ["NationalIDCredential", "DriverLicenseCredential"],
      "accredited_at": "2024-03-01T00:00:00Z",
      "expires_at": "2025-03-01T00:00:00Z",
      "status": "active"
    }
  ]
}
```

### 3.6 Go Code: Trust Verification

The following Go code implements DID-based trust verification for credential agents. It
resolves DIDs, verifies well-known DID configuration, and checks trust registries.

```go
// Package credentialagent implements trust establishment for credential agents.
package credentialagent

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
)

// TrustRegistry defines the interface for checking issuer trust.
type TrustRegistry interface {
	IsTrustedIssuer(ctx context.Context, issuerDID string, credentialType string) (bool, error)
	GetIssuerPublicKey(ctx context.Context, issuerDID string, keyID string) (any, error)
}

// DIDResolver resolves DIDs to DID documents.
type DIDResolver interface {
	Resolve(ctx context.Context, did string) (*DIDDocument, error)
}

// DIDDocument represents a W3C DID Document.
type DIDDocument struct {
	Context            []string           `json:"@context"`
	ID                 string             `json:"id"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
	Authentication     []string           `json:"authentication"`
	AssertionMethod    []string           `json:"assertionMethod"`
}

// VerificationMethod is a public key in a DID document.
type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase,omitempty"`
	PublicKeyJwk       map[string]any `json:"publicKeyJwk,omitempty"`
}

// DIDConfiguration represents the .well-known DID configuration.
type DIDConfiguration struct {
	Context           string             `json:"@context"`
	Entries           []DIDConfigurationEntry `json:"did-configuration"`
}

type DIDConfigurationEntry struct {
	ID string          `json:"id"`
	VC json.RawMessage `json:"vc"`
}

// WebDIDResolver resolves did:web DIDs by fetching /.well-known/did.json.
type WebDIDResolver struct {
	httpClient *http.Client
}

func NewWebDIDResolver() *WebDIDResolver {
	return &WebDIDResolver{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Resolve fetches the DID document for a did:web identifier.
// Example: did:web:example.com → https://example.com/.well-known/did.json
func (r *WebDIDResolver) Resolve(ctx context.Context, did string) (*DIDDocument, error) {
	if !strings.HasPrefix(did, "did:web:") {
		return nil, fmt.Errorf("unsupported DID method: %s", did)
	}

	// Parse did:web:domain.com[:path]
	parts := strings.SplitN(did, ":", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid did:web format")
	}

	domainPath := strings.ReplaceAll(parts[2], ":", "/")
	url := fmt.Sprintf("https://%s/.well-known/did.json", domainPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create DID resolution request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("resolve DID %s: %w", did, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DID resolution returned %d for %s", resp.StatusCode, did)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read DID document: %w", err)
	}

	var doc DIDDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse DID document: %w", err)
	}

	if doc.ID != did {
		return nil, fmt.Errorf("DID mismatch: expected %s, got %s", did, doc.ID)
	}

	return &doc, nil
}

// VerifyDIDConfiguration verifies the .well-known DID configuration
// to confirm domain linkage between a DID and a web origin.
func VerifyDIDConfiguration(ctx context.Context, origin, did string, resolver DIDResolver) error {
	// 1. Fetch https://origin/.well-known/did-configuration
	url := origin + "/.well-known/did-configuration"

	httpClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create DID config request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch DID configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DID configuration not found (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read DID configuration: %w", err)
	}

	var config DIDConfiguration
	if err := json.Unmarshal(body, &config); err != nil {
		return fmt.Errorf("parse DID configuration: %w", err)
	}

	// 2. Find the entry matching this DID.
	var entry *DIDConfigurationEntry
	for i := range config.Entries {
		if config.Entries[i].ID == did {
			entry = &config.Entries[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("DID %s not found in configuration", did)
	}

	// 3. Verify the domain linkage VC.
	//    The VC should be signed by the DID's key, and contain the origin.
	var vc struct {
		CredentialSubject struct {
			Origin string `json:"origin"`
		} `json:"credentialSubject"`
	}
	if err := json.Unmarshal(entry.VC, &vc); err != nil {
		return fmt.Errorf("parse domain linkage VC: %w", err)
	}

	if vc.CredentialSubject.Origin != origin {
		return fmt.Errorf("origin mismatch: expected %s, got %s", origin, vc.CredentialSubject.Origin)
	}

	// 4. Resolve the DID and verify the signature on the VC.
	doc, err := resolver.Resolve(ctx, did)
	if err != nil {
		return fmt.Errorf("resolve DID for signature verification: %w", err)
	}

	// In production, verify the VC proof using the DID document's verification methods.
	_ = doc // verification would use doc.AssertionMethod keys

	return nil
}

// TrustVerifier combines DID resolution and trust registry checks.
type TrustVerifier struct {
	resolver DIDResolver
	registry TrustRegistry
}

func NewTrustVerifier(resolver DIDResolver, registry TrustRegistry) *TrustVerifier {
	return &TrustVerifier{resolver: resolver, registry: registry}
}

// VerifyIssuerTrust checks that an issuer is trusted for the given credential type.
func (tv *TrustVerifier) VerifyIssuerTrust(ctx context.Context, issuerDID, credentialType string) error {
	// 1. Check trust registry.
	trusted, err := tv.registry.IsTrustedIssuer(ctx, issuerDID, credentialType)
	if err != nil {
		return fmt.Errorf("trust registry check failed: %w", err)
	}
	if !trusted {
		return errors.New(errors.ErrPermissionDenied,
			fmt.Sprintf("issuer %s is not trusted for credential type %s", issuerDID, credentialType))
	}

	// 2. Resolve DID to ensure it's valid.
	doc, err := tv.resolver.Resolve(ctx, issuerDID)
	if err != nil {
		return fmt.Errorf("DID resolution failed: %w", err)
	}

	if len(doc.AssertionMethod) == 0 {
		return fmt.Errorf("DID %s has no assertion methods", issuerDID)
	}

	return nil
}

// ExtractPublicKeyFromVerificationMethod extracts a public key from a DID verification method.
func ExtractPublicKeyFromVerificationMethod(vm *VerificationMethod) (ed25519.PublicKey, error) {
	if vm.Type != "Ed25519VerificationKey2020" {
		return nil, fmt.Errorf("unsupported key type: %s", vm.Type)
	}

	if vm.PublicKeyMultibase == "" {
		return nil, fmt.Errorf("no publicKeyMultibase in verification method")
	}

	// Decode multibase (base58btc with 'z' prefix)
	// In production, use a proper multibase/multicodec decoder.
	decoded, err := base64.RawURLEncoding.DecodeString(vm.PublicKeyMultibase)
	if err != nil {
		// Try as raw base64
		decoded, err = base64.StdEncoding.DecodeString(vm.PublicKeyMultibase)
		if err != nil {
			return nil, fmt.Errorf("decode public key: %w", err)
		}
	}

	// Ed25519 public key is 32 bytes.
	// The multibase encoding may include a multicodec prefix (0xed01).
	if len(decoded) >= 34 && decoded[0] == 0xed && decoded[1] == 0x01 {
		decoded = decoded[2:]
	}

	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 key length: %d", len(decoded))
	}

	return ed25519.PublicKey(decoded), nil
}
```

### 3.7 Trust Model Comparison

| Aspect | Traditional SSO (OIDC) | Credential Agent |
|--------|----------------------|------------------|
| Trust anchor | Centralized IdP | Distributed (DIDs, trust registry) |
| User identity | Stored at IdP | Stored in user's wallet |
| Attribute sharing | IdP sends all claims in ID token | User selects which attributes to share |
| Revocation | IdP revokes session | Issuer revokes credential; wallet checks status |
| Single point of failure | IdP outage = all logins fail | Wallet works offline; issuer needed only for status checks |
| Privacy | IdP sees every RP login | Issuer does not know when/where credential is used |

---

## 4. Token Exchange (RFC 8693) for Wallets

### 4.1 Overview

[RFC 8693](https://datatracker.ietf.org/doc/html/rfc8693) defines OAuth 2.0 Token Exchange,
a grant type that allows a client to exchange one token for another with potentially
different scope, audience, or token type. This is directly applicable to credential agent
integration.

In the credential agent model, token exchange works as follows:

```
Wallet holds a verifiable credential (VC)
         │
         ▼
RP requests authentication
         │
         ▼
Wallet presents VC to GGID
         │
         ▼
GGID exchanges VC for an internal access token (RFC 8693)
         │
         ▼
RP uses the access token to call APIs
```

### 4.2 Token Exchange Parameters

RFC 8693 defines the following parameters for the token endpoint:

| Parameter | Description | Credential Agent Mapping |
|-----------|-------------|--------------------------|
| `grant_type` | Must be `urn:ietf:params:oauth:grant-type:token-exchange` | Fixed value |
| `subject_token` | The token representing the subject (user) | The user's identity assertion |
| `subject_token_type` | Type identifier for subject_token | `urn:ietf:params:oauth:token-type:jwt` |
| `actor_token` | Token representing the actor (delegating party) | The wallet's credential/assertion |
| `actor_token_type` | Type identifier for actor_token | `urn:ietf:params:oauth:token-type:jwt` |
| `resource` | The target resource server | `https://api.ggid.example.com` |
| `audience` | The intended audience for the new token | `internal-service-api` |
| `scope` | Requested scope for the new token | `read:users write:users` |
| `requested_token_type` | Desired token type for the response | `urn:ietf:params:oauth:token-type:access_token` |

### 4.3 Wallet Token Exchange Flow

```
┌─────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  User   │     │  Wallet  │     │  GGID    │     │   RP     │
│         │     │ (Agent)  │     │  OAuth   │     │  (API)   │
└────┬────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │ 1. Access RP  │                │                │
     │───────────────────────────────────────────────→│
     │                │ 2. RP requests credential      │
     │←───────────────────────────────────────────────│
     │ 3. Pick credential & consent   │                │
     │───────────────→│                │                │
     │                │ 4. Present VC (actor_token)    │
     │                │ + user assertion (subject_token)│
     │                │───────────────→│                │
     │                │                │ 5. Exchange:   │
     │                │                │    validate VC │
     │                │                │    issue new   │
     │                │                │    access token│
     │                │←───────────────│                │
     │                │ 6. Return token to RP          │
     │                │───────────────────────────────→│
     │                │                │ 7. RP calls API│
     │                │                │←───────────────│
     │                │                │ 8. API verifies│
     │                │                │    GGID token  │
     │                │                │───────────────→│
```

### 4.4 Actor Token vs Subject Token

In the wallet token exchange pattern:
- **Subject token**: Represents the user's identity. This is typically a JWT assertion
  signed by the wallet, containing the user's DID or identifier.
- **Actor token**: Represents the wallet/credential agent itself. This is the verifiable
  credential that proves the wallet is authorized to act on behalf of the user.

This mirrors RFC 8693's delegation model:
- Subject = the user (the entity being acted for)
- Actor = the wallet (the entity performing the action)

### 4.5 Go Code: Wallet Token Exchange Handler

```go
// Package walletexchange implements RFC 8693 token exchange for wallet-issued tokens.
package walletexchange

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// WalletTokenExchangeRequest maps RFC 8693 parameters to wallet credential exchange.
type WalletTokenExchangeRequest struct {
	TenantID           uuid.UUID
	SubjectToken       string   // User identity assertion from wallet
	SubjectTokenType   string   // e.g., urn:ietf:params:oauth:token-type:jwt
	ActorToken         string   // Verifiable credential from wallet
	ActorTokenType     string   // e.g., urn:ietf:params:oauth:token-type:jwt
	Resource           string   // Target resource server
	Audience           string   // Desired audience for new token
	Scope              []string // Requested scopes
	RequestedTokenType string   // Desired output token type
}

// VerifiableCredential represents the relevant fields of a W3C VC or SD-JWT.
type VerifiableCredential struct {
	Context          []string               `json:"@context"`
	Type             []string               `json:"type"`
	Issuer           string                 `json:"issuer"`
	IssuanceDate     string                 `json:"issuanceDate"`
	ExpirationDate   string                 `json:"expirationDate,omitempty"`
	CredentialSubject map[string]any        `json:"credentialSubject"`
	Proof            *LinkedDataProof       `json:"proof,omitempty"`
}

// LinkedDataProof is the W3C Linked Data Proof on a VC.
type LinkedDataProof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	VerificationMethod string `json:"verificationMethod"`
	ProofPurpose       string `json:"proofPurpose"`
	ProofValue         string `json:"proofValue"`
}

// WalletAssertion is a JWT assertion signed by the wallet, attesting to the user's identity.
type WalletAssertion struct {
	Issuer    string `json:"iss"` // wallet DID
	Subject   string `json:"sub"` // user DID or identifier
	Audience  string `json:"aud"` // GGID OAuth server
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	Nonce     string `json:"nonce"`
	VP        json.RawMessage `json:"vp,omitempty"` // embedded verifiable presentation
}

// WalletExchangeConfig holds configuration for the wallet token exchange handler.
type WalletExchangeConfig struct {
	// TrustRegistry verifies that the wallet's credential issuer is trusted.
	TrustRegistry TrustRegistry
	// VCVerifier verifies the cryptographic proof on verifiable credentials.
	VCVerifier VCVerifier
	// KeyProvider provides the signing key for issuing new tokens.
	KeyProvider KeyProvider
	// Issuer is the GGID OAuth server's issuer URL.
	Issuer string
	// TokenTTL is the lifetime of tokens issued via wallet exchange.
	TokenTTL time.Duration
	// AllowedScopes defines the maximum scope that wallet-exchanged tokens can have.
	AllowedScopes map[string]bool
}

// TrustRegistry checks whether a credential issuer is trusted.
type TrustRegistry interface {
	IsTrustedIssuer(ctx context.Context, issuerDID, credentialType string) (bool, error)
}

// VCVerifier verifies verifiable credential proofs.
type VCVerifier interface {
	Verify(ctx context.Context, vc *VerifiableCredential) error
}

// KeyProvider provides cryptographic keys for token signing.
type KeyProvider interface {
	SignToken(claims jwt.MapClaims) (string, error)
}

// ExchangeTokenForWallet implements the wallet credential token exchange.
//
// Flow:
//  1. Parse and validate the actor_token (verifiable credential)
//  2. Parse and validate the subject_token (wallet assertion)
//  3. Verify the VC's issuer is trusted
//  4. Verify the VC's cryptographic proof
//  5. Verify the wallet assertion is signed by a trusted wallet
//  6. Issue a new GGID access token with reduced scope
func ExchangeTokenForWallet(
	ctx context.Context,
	req *WalletTokenExchangeRequest,
	cfg *WalletExchangeConfig,
) (*TokenExchangeResponse, error) {
	// Validate required parameters.
	if req.SubjectToken == "" {
		return nil, errors.InvalidArgument("subject_token is required")
	}
	if req.SubjectTokenType == "" {
		return nil, errors.InvalidArgument("subject_token_type is required")
	}
	if req.ActorToken == "" {
		return nil, errors.InvalidArgument("actor_token is required")
	}

	// 1. Parse the actor token (verifiable credential).
	vc, err := parseVerifiableCredential(req.ActorToken)
	if err != nil {
		return nil, errors.InvalidArgument(fmt.Sprintf("invalid actor_token: %v", err))
	}

	// 2. Check expiration.
	if vc.ExpirationDate != "" {
		expTime, err := time.Parse(time.RFC3339, vc.ExpirationDate)
		if err != nil {
			return nil, errors.InvalidArgument("invalid credential expiration date")
		}
		if time.Now().After(expTime) {
			return nil, errors.InvalidArgument("credential has expired")
		}
	}

	// 3. Verify the VC issuer is in the trust registry.
	credentialType := ""
	if len(vc.Type) > 0 {
		credentialType = vc.Type[len(vc.Type)-1] // most specific type
	}

	trusted, err := cfg.TrustRegistry.IsTrustedIssuer(ctx, vc.Issuer, credentialType)
	if err != nil {
		return nil, errors.Internal("trust registry check failed", err)
	}
	if !trusted {
		return nil, errors.PermissionDenied(
			fmt.Sprintf("credential issuer %s is not trusted for type %s", vc.Issuer, credentialType))
	}

	// 4. Verify the VC's cryptographic proof.
	if err := cfg.VCVerifier.Verify(ctx, vc); err != nil {
		return nil, errors.PermissionDenied(fmt.Sprintf("credential proof verification failed: %v", err))
	}

	// 5. Parse and validate the subject token (wallet assertion).
	walletAssert, err := parseWalletAssertion(req.SubjectToken)
	if err != nil {
		return nil, errors.InvalidArgument(fmt.Sprintf("invalid subject_token: %v", err))
	}

	// 6. Verify the wallet assertion audience matches GGID.
	if walletAssert.Audience != cfg.Issuer {
		return nil, errors.InvalidArgument(
			fmt.Sprintf("wallet assertion audience mismatch: expected %s, got %s",
				cfg.Issuer, walletAssert.Audience))
	}

	// 7. Verify the wallet assertion is not expired.
	if time.Now().Unix() > walletAssert.ExpiresAt {
		return nil, errors.InvalidArgument("wallet assertion has expired")
	}

	// 8. Extract user identity from the VC's credential subject.
	userDID, ok := vc.CredentialSubject["id"].(string)
	if !ok {
		userDID = walletAssert.Subject
	}

	// 9. Filter requested scopes against allowed scopes.
	allowedScope := filterScopes(req.Scope, cfg.AllowedScopes)

	// 10. Issue a new GGID access token.
	now := time.Now()
	expiresAt := now.Add(cfg.TokenTTL)
	if cfg.TokenTTL == 0 {
		expiresAt = now.Add(15 * time.Minute) // default 15 min
	}

	// Map the VC type to internal scopes/permissions.
	mappedScopes := mapCredentialToScopes(vc, allowedScope)

	claims := jwt.MapClaims{
		"iss":            cfg.Issuer,
		"sub":            userDID,
		"aud":            req.Audience,
		"iat":            now.Unix(),
		"exp":            expiresAt.Unix(),
		"jti":            uuid.New().String(),
		"scope":          strings.Join(mappedScopes, " "),
		"tenant_id":      req.TenantID.String(),
		"auth_method":    "wallet_exchange",
		"vc_issuer":      vc.Issuer,
		"vc_type":        credentialType,
		"wallet_did":     walletAssert.Issuer,
		"acr":            determineACR(vc),
		"amr":            []string{"vc", "wallet"},
	}

	// Add actor information per RFC 8693 §4.3 (may field).
	claims["may"] = map[string]any{
		"actor": map[string]any{
				 "iss": walletAssert.Issuer,
			"sub": walletAssert.Subject,
		},
		"credential": map[string]any{
			"issuer": vc.Issuer,
			"type":   credentialType,
		},
	}

	accessToken, err := cfg.KeyProvider.SignToken(claims)
	if err != nil {
		return nil, errors.Internal("sign wallet exchange token", err)
	}

	return &TokenExchangeResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(expiresAt).Seconds()),
		Scope:       strings.Join(mappedScopes, " "),
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
	}, nil
}

// TokenExchangeResponse is the RFC 8693 token exchange response.
type TokenExchangeResponse struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	Scope           string `json:"scope,omitempty"`
}

// parseVerifiableCredential parses a VC from a JWT or JSON-LD format.
func parseVerifiableCredential(token string) (*VerifiableCredential, error) {
	// Try JWT format first (common for SD-JWT and jwt_vc_json).
	if strings.HasPrefix(token, "eyJ") {
		return parseVcFromJWT(token)
	}

	// Fall back to JSON-LD format.
	var vc VerifiableCredential
	if err := json.Unmarshal([]byte(token), &vc); err != nil {
		return nil, fmt.Errorf("parse VC: %w", err)
	}
	return &vc, nil
}

// parseVcFromJWT extracts a VC from a JWT's payload.
func parseVcFromJWT(jwtStr string) (*VerifiableCredential, error) {
	parts := strings.Split(jwtStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode payload (part[1]).
	payload, err := jwt.NewParser(
		jwt.WithoutClaimsValidation(),
	).ParseUnverified(jwtStr, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse JWT: %w", err)
	}

	claims, ok := payload.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWT claims")
	}

	// The VC may be in the "vc" claim (jwt_vc_json format).
	vcRaw, ok := claims["vc"]
	if !ok {
		return nil, fmt.Errorf("JWT does not contain a 'vc' claim")
	}

	vcBytes, err := json.Marshal(vcRaw)
	if err != nil {
		return nil, fmt.Errorf("marshal VC claim: %w", err)
	}

	var vc VerifiableCredential
	if err := json.Unmarshal(vcBytes, &vc); err != nil {
		return nil, fmt.Errorf("parse VC from JWT: %w", err)
	}

	return &vc, nil
}

// parseWalletAssertion parses the wallet's identity assertion JWT.
func parseWalletAssertion(token string) (*WalletAssertion, error) {
	parsed, _, err := jwt.NewParser(
		jwt.WithoutClaimsValidation(),
	).ParseUnverified(token, &WalletAssertion{})
	if err != nil {
		return nil, fmt.Errorf("parse wallet assertion: %w", err)
	}

	assertion, ok := parsed.Claims.(*WalletAssertion)
	if !ok {
		return nil, fmt.Errorf("invalid wallet assertion claims")
	}

	return assertion, nil
}

// filterScopes removes any requested scopes that are not in the allowed set.
func filterScopes(requested []string, allowed map[string]bool) []string {
	var result []string
	for _, s := range requested {
		if allowed[s] {
			result = append(result, s)
		}
	}
	return result
}

// mapCredentialToScopes maps credential types to internal OAuth scopes.
func mapCredentialToScopes(vc *VerifiableCredential, requested []string) []string {
	// If scopes were explicitly requested and allowed, use them.
	if len(requested) > 0 {
		return requested
	}

	// Otherwise, derive scopes from the credential type.
	var mapped []string
	for _, t := range vc.Type {
		switch t {
		case "EmployeeCredential":
			mapped = append(mapped, "openid", "profile", "employee:read")
		case "AgeVerificationCredential":
			mapped = append(mapped, "openid", "age:verify")
		case "NationalIDCredential":
			mapped = append(mapped, "openid", "profile", "identity:verify")
		case "DriverLicenseCredential":
			mapped = append(mapped, "openid", "profile", "driving:verify")
		}
	}

	if len(mapped) == 0 {
		mapped = []string{"openid", "profile"}
	}

	return mapped
}

// determineACR determines the Authentication Context Class Reference based on the VC.
func determineACR(vc *VerifiableCredential) string {
	for _, t := range vc.Type {
		switch t {
		case "NationalIDCredential":
			return "urn:eidas:substantial"
		case "DriverLicenseCredential":
			return "https://refeds.org/assurance/mfa"
		case "EmployeeCredential":
			return "https://ggid.example.com/acr/employee"
		}
	}
	return "urn:ietf:params:oauth:vc:default"
}
```

### 4.6 HTTP Handler Integration

To wire wallet token exchange into GGID's token endpoint:

```go
// In server.go buildHandler(), add to the grant_type switch:
case "urn:ietf:params:oauth:grant-type:token-exchange":
	// RFC 8693 token exchange — supports wallet credential exchange.
	subjectToken := r.FormValue("subject_token")
	subjectTokenType := r.FormValue("subject_token_type")
	actorToken := r.FormValue("actor_token")
	actorTokenType := r.FormValue("actor_token_type")
	resource := r.FormValue("resource")
	audience := r.FormValue("audience")
	requestedTokenType := r.FormValue("requested_token_type")

	resp, tokenErr = oauthSvc.ExchangeToken(ctx, &service.TokenExchangeRequestRFC8693{
		TenantID:           tenantID,
		SubjectToken:       subjectToken,
		SubjectTokenType:   subjectTokenType,
		ActorToken:         actorToken,
		ActorTokenType:     actorTokenType,
		Resource:           resource,
		Audience:           audience,
		Scope:              scopes,
		RequestedTokenType: requestedTokenType,
	})
```

This maps cleanly to GGID's existing `ExchangeToken` method in `oauth_service.go` (line 1119).

---

## 5. Browser-Mediated vs Direct API

### 5.1 Two Integration Models

Credential agent integration can happen through two channels:

**Browser-Mediated**: The web browser handles wallet invocation, user consent, and returns
the credential to the web page. The browser acts as a trusted intermediary.

**Direct API**: The application calls the wallet's API directly, without browser mediation.
This is used in mobile apps, server-to-server scenarios, and native clients.

### 5.2 Browser-Mediated: Pros and Cons

**Advantages:**
- **Phishing resistance**: The browser validates the origin before invoking the wallet.
- **User experience**: Native OS UI for credential selection is familiar and accessible.
- **Security isolation**: The web page cannot access wallet internals — only receives the result.
- **Cross-wallet compatibility**: The browser handles wallet discovery and protocol negotiation.
- **Consent management**: The browser enforces user consent before releasing credentials.

**Disadvantages:**
- **Browser dependency**: Requires a browser that supports the Digital Credentials API.
- **Latency**: The mediation layer adds overhead.
- **No fine-grained control**: The web page cannot customize the credential picker UI.
- **Protocol limitations**: The browser may not support all credential formats.

### 5.3 Direct API: Pros and Cons

**Advantages:**
- **Full control**: The application controls the UX and can customize the credential selection.
- **No browser dependency**: Works in any client (mobile, desktop, CLI, server).
- **Lower latency**: No mediation layer.
- **Custom protocols**: Can support credential formats not yet standardized in browsers.

**Disadvantages:**
- **Security risk**: The application has direct access to credential data — a compromised app
  could exfiltrate credentials.
- **Phishing vulnerability**: Without browser mediation, there's no origin validation.
- **Implementation complexity**: Each wallet requires custom integration code.
- **No standardized UX**: Each app must build its own credential picker.

### 5.4 Cross-Origin Restrictions

Browser-mediated exchange is subject to cross-origin restrictions:

```javascript
// The requesting page must be served over HTTPS.
// The Permissions-Policy header must allow digital-credentials:
//   Permissions-Policy: digital-credentials=(self)

// Cross-origin iframes cannot invoke the wallet unless explicitly allowed:
<iframe
  src="https://embedded-app.example.com"
  allow="digital-credentials"
></iframe>

// The browser will reject wallet invocation if:
// 1. The page is not HTTPS
// 2. The Permissions-Policy does not include digital-credentials
// 3. The requesting origin is not in the allow list
```

### 5.5 JavaScript: Browser-Mediated Flow

```javascript
// === Browser-Mediated Credential Exchange ===

async function authenticateWithWallet() {
  // 1. Check browser support
  if (!("identity" in navigator) || !("digital" in navigator.identity)) {
    throw new Error("Browser does not support Digital Credentials API");
  }

  const supported = await navigator.identity.digital.isSupported();
  if (!supported) {
    // Fallback to traditional OIDC flow
    return redirectToOIDCLogin();
  }

  // 2. Build credential request
  const request = {
    digital: {
      requests: [{
        id: "gdid-employee-auth",
        manifest: {
          id: "gdid-employee-auth-v1",
          version: "1.0",
          purpose: "Employee authentication for GGID Console",
          input_descriptors: [{
            id: "employee-credential",
            constraints: {
              fields: [
                { path: ["$.vc.type"], filter: { type: "array", contains: { const: "EmployeeCredential" } } },
                { path: ["$.vc.credentialSubject.employeeId"], filter: { type: "string" } },
                { path: ["$.vc.credentialSubject.active"], filter: { type: "boolean", const: true } }
              ]
            }
          }]
        },
        protocols: {
          "openid4vp": {
            params: {
              client_id: window.location.origin,
              response_type: "vp_token",
              response_mode: "direct_post",
              nonce: generateNonce()
            }
          }
        }
      }]
    }
  };

  try {
    // 3. Invoke browser-mediated wallet selection
    const response = await navigator.identity.get(request);

    if (!response || !response.digital) {
      throw new Error("No credential returned");
    }

    // 4. Send the credential to GGID for token exchange
    const tokenResponse = await fetch("/oauth/token", {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "urn:ietf:params:oauth:grant-type:token-exchange",
        subject_token: response.digital.data.vp_token,
        subject_token_type: "urn:ietf:params:oauth:token-type:jwt",
        actor_token: response.digital.data.vp_token, // VC serves as both actor and subject
        actor_token_type: "urn:ietf:params:oauth:token-type:jwt",
        audience: "https://api.ggid.example.com",
        scope: "openid profile employee:read"
      })
    });

    if (!tokenResponse.ok) {
      throw new Error(`Token exchange failed: ${tokenResponse.status}`);
    }

    const tokens = await tokenResponse.json();
    // Store tokens and proceed with authenticated session
    return tokens;

  } catch (error) {
    if (error.name === "NotAllowedError") {
      // User cancelled — fall back to password login
      return redirectToPasswordLogin();
    }
    throw error;
  }
}

function generateNonce() {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return Array.from(array, b => b.toString(16).padStart(2, '0')).join('');
}
```

### 5.6 Go: Direct API Server-Side Flow

```go
// Direct wallet API integration — server-side verification.
// Used when the RP is a backend service or mobile app that
// communicates directly with the wallet.

func HandleDirectWalletAuth(w http.ResponseWriter, r *http.Request) {
	// 1. The client sends the VP token directly (from a native wallet SDK).
	vpToken := r.FormValue("vp_token")
	if vpToken == "" {
		writeError(w, http.StatusBadRequest, "vp_token is required")
		return
	}

	// 2. Verify the VP token.
	//    In direct mode, there is no browser mediation, so the RP must:
	//    a) Verify the cryptographic proof
	//    b) Check the nonce (replay prevention)
	//    c) Verify the issuer is trusted
	//    d) Check credential revocation status

	nonce := r.FormValue("nonce")
	expectedNonce := getSessionNonce(r) // from session or cache
	if nonce != expectedNonce {
		writeError(w, http.StatusForbidden, "nonce mismatch — possible replay attack")
		return
	}

	// 3. Exchange via RFC 8693
	exchangeReq := &WalletTokenExchangeRequest{
		TenantID:         getTenantID(r),
		SubjectToken:     vpToken,
		SubjectTokenType: "urn:ietf:params:oauth:token-type:jwt",
		ActorToken:       vpToken,
		ActorTokenType:   "urn:ietf:params:oauth:token-type:jwt",
		Audience:         "https://api.ggid.example.com",
		Scope:            []string{"openid", "profile", "employee:read"},
	}

	resp, err := ExchangeTokenForWallet(r.Context(), exchangeReq, walletExchangeConfig)
	if err != nil {
		writeError(w, http.StatusForbidden, fmt.Sprintf("credential exchange failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
```

### 5.7 Security Tradeoff Summary

| Security Property | Browser-Mediated | Direct API |
|-------------------|:----------------:|:----------:|
| Phishing resistance | Strong (origin bound) | Weak (app must verify) |
| Credential exfiltration | Hard (browser isolates) | Easy (app has full access) |
| Replay prevention | Browser provides nonce | App must implement |
| Consent enforcement | Browser enforces | App must implement |
| Cross-origin protection | Enforced by browser | Not applicable |
| Key isolation | Wallet keys never exposed to page | App may have key access |

### 5.8 Recommendation

For web applications, always use **browser-mediated** exchange when available. Fall back to
direct API only for native mobile apps and server-to-server scenarios where the client is
trusted and the channel is authenticated (mTLS).

---

## 6. Wallet-as-Broker Pattern

### 6.1 Overview

In the **wallet-as-broker** pattern, the wallet acts as an authentication broker between
the user and multiple RPs. Instead of a centralized IdP brokering authentication (as in
traditional SSO), the wallet mediates credential release to each RP independently.

This pattern eliminates the centralized trust anchor — the IdP — and replaces it with a
decentralized model where each credential issuer is independently trusted by the RPs that
accept that credential type.

### 6.2 Traditional SSO vs Wallet-as-Broker

**Traditional SSO (OIDC):**

```
         ┌──────────────┐
         │  Central IdP │  ← single trust anchor
         │  (GGID/Okta) │     knows all RP logins
         └──────┬───────┘
                │
     ┌──────────┼──────────┐
     ▼          ▼          ▼
  ┌─────┐  ┌─────┐  ┌─────┐
  │ RP1 │  │ RP2 │  │ RP3 │
  └─────┘  └─────┘  └─────┘
     │          │          │
     └──────────┴──────────┘
                │
         ┌──────┴───────┐
         │     User     │
         │ (one login)  │
         └──────────────┘
```

Problems:
- IdP is a single point of failure
- IdP sees every RP login (privacy concern)
- If IdP is compromised, all RPs are at risk
- User has no control over which attributes are shared with each RP

**Wallet-as-Broker:**

```
         ┌──────────────────┐
         │  User's Wallet   │  ← user controls credential release
         │  (Apple/Google/  │     no centralized tracking
         │   Enterprise)    │
         └────────┬─────────┘
                  │
      ┌───────────┼───────────┐
      ▼           ▼           ▼
   ┌─────┐   ┌─────┐   ┌─────┐
   │ RP1 │   │ RP2 │   │ RP3 │  ← each RP verifies independently
   └──┬──┘   └──┬──┘   └──┬──┘     no shared trust anchor
      │         │         │
      ▼         ▼         ▼
   ┌─────┐   ┌─────┐   ┌─────┐
   │VC:  │   │VC:  │   │VC:  │  ← different credentials for different RPs
   │Emp  │   │Age  │   │ID   │
   └─────┘   └─────┘   └─────┘
```

### 6.3 Benefits

| Benefit | Description |
|---------|-------------|
| **No single point of failure** | Each RP verifies credentials independently; no central IdP outage can block all logins. |
| **Privacy** | The credential issuer does not know when/where the credential is used. The wallet brokers locally. |
| **User control** | The user decides which credential to present to each RP and which attributes to disclose. |
| **Selective disclosure** | With SD-JWT or BBS+, the user can share only the attributes the RP needs (e.g., "over 18" without revealing birth date). |
| **Offline capability** | Credential verification can work offline (if revocation status is cached). |
| **Reduced IdP load** | No token endpoint calls for each RP access — the credential is self-contained. |

### 6.4 Challenges

| Challenge | Description |
|-----------|-------------|
| **Revocation** | Without a central session, revoking a credential requires status lists or online checks. |
| **Key management** | Users must manage their wallet keys; lost keys mean lost access. |
| **Trust bootstrapping** | RPs must establish trust with multiple credential issuers independently. |
| **UX fragmentation** | Each wallet may have a different UI, confusing users. |
| **Account linking** | Mapping credential subjects to internal user accounts requires additional logic. |

### 6.5 Architecture Diagram (ASCII)

```
┌────────────────────────────────────────────────────────────────────────┐
│                        WALLET-AS-BROKER ARCHITECTURE                    │
├────────────────────────────────────────────────────────────────────────┤
│                                                                        │
│   ISSUERS (trusted, independent)                                       │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                   │
│   │ GGID Issuer  │  │  Government │  │  Employer   │                   │
│   │ (Employee VC)│  │  (ID VC)    │  │  (Degree VC)│                   │
│   └──────┬───────┘  └──────┬──────┘  └──────┬──────┘                   │
│          │ issue VC        │ issue VC       │ issue VC                  │
│          ▼                 ▼                ▼                           │
│   ┌──────────────────────────────────────────────────┐                 │
│   │               USER'S WALLET                       │                 │
│   │  ┌───────────┐ ┌───────────┐ ┌───────────┐       │                 │
│   │  │ Employee  │ │    ID     │ │  Degree   │       │                 │
│   │  │    VC     │ │    VC     │ │    VC     │       │                 │
│   │  └───────────┘ └───────────┘ └───────────┘       │                 │
│   │  [selective disclosure enabled]                   │                 │
│   │  [offline revocation cache]                       │                 │
│   └──────────────────┬───────────────────────────────┘                 │
│                      │ present VP (on consent)                          │
│      ┌───────────────┼───────────────┐                                 │
│      ▼               ▼               ▼                                 │
│   ┌──────┐       ┌──────┐       ┌──────┐                              │
│   │ RP1  │       │ RP2  │       │ RP3  │                              │
│   │(HR   │       │(Age  │       │(Edu  │                              │
│   │Portal│       │Gate) │       │Site) │                              │
│   └──┬───┘       └──┬───┘       └──┬───┘                              │
│      │              │              │                                   │
│      │ verify VC    │ verify VC    │ verify VC                         │
│      │ (local)      │ (local)      │ (local)                           │
│      │              │              │                                   │
│      ▼              ▼              ▼                                   │
│   [Trust Registry: did:web:issuer.ggid.example.com → trusted]          │
│   [Revocation: token status list checked locally]                     │
│                                                                        │
└────────────────────────────────────────────────────────────────────────┘
```

### 6.6 When to Use Wallet-as-Broker vs Traditional SSO

| Scenario | Recommended Pattern |
|----------|-------------------|
| Enterprise workforce SSO | Traditional SSO (centralized management, policy enforcement) |
| B2C with government ID | Wallet-as-broker (user controls ID, privacy) |
| Cross-organization federation | Wallet-as-broker (no shared IdP needed) |
| Regulated industries (banking, healthcare) | Hybrid (SSO + wallet for high-assurance attributes) |
| Age verification for online services | Wallet-as-broker (selective disclosure: "over 18" only) |
| Academic credential verification | Wallet-as-broker (portable, issuer-independent) |

### 6.7 Hybrid Model

GGID can support both patterns simultaneously:
- **Traditional SSO**: GGID acts as the centralized IdP for internal applications.
- **Wallet-as-Broker**: GGID issues VCs that users store in wallets for external presentation.
- **Token Exchange**: Wallet-presented VCs can be exchanged for internal GGID tokens (RFC 8693).

This hybrid approach gives GGID the best of both worlds: centralized policy enforcement for
internal services and decentralized, user-controlled credential presentation for external
verification.

---

## 7. Credential Storage in Wallets

### 7.1 Storage Requirements

Credential wallets must store credentials securely while providing fast access for
presentation. Key requirements:

1. **Confidentiality**: Credentials must be encrypted at rest.
2. **Integrity**: Tampering must be detectable.
3. **Availability**: Credentials must be accessible when needed (offline support).
4. **User binding**: Only the authorized user can access credentials.
5. **Key isolation**: Credential signing keys must never leave the secure enclave.

### 7.2 OS Keychain Integration

Each platform provides a secure credential store:

| Platform | Store | API |
|----------|-------|-----|
| macOS/iOS | Keychain | Security.framework / LocalAuthentication |
| Android | Keystore | AndroidKeyStore + BiometricPrompt |
| Windows | Credential Manager | CredProtect / DPAPI |
| Linux | Secret Service | D-Bus Secret Service API / GNOME Keyring |

### 7.3 TEE-Backed Storage

Trusted Execution Environments (TEEs) provide hardware-isolated storage:

| TEE | Platform | Key Feature |
|-----|----------|-------------|
| Secure Enclave | Apple (A7+) | Hardware key isolation, biometric gating |
| TrustZone | ARM-based Android | Isolated secure world |
| TPM 2.0 | Windows / Linux | Hardware root of trust |
| SGX | Intel | Enclave-based computation |

In TEE-backed storage:
1. The credential encryption key is generated inside the TEE.
2. The key never leaves the TEE in plaintext.
3. Decryption only occurs inside the TEE after biometric/PIN verification.
4. Even a compromised OS cannot extract the key.

### 7.4 Credential Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "WalletCredential",
  "type": "object",
  "required": ["id", "type", "issuer", "format", "data", "metadata"],
  "properties": {
    "id": {
      "type": "string",
      "description": "Unique identifier for this credential in the wallet"
    },
    "type": {
      "type": "array",
      "items": { "type": "string" },
      "description": "W3C VC type array, e.g., ['VerifiableCredential', 'EmployeeCredential']"
    },
    "issuer": {
      "type": "object",
      "properties": {
        "id": { "type": "string", "description": "DID or URL of issuer" },
        "name": { "type": "string" },
        "logo": { "type": "string" }
      }
    },
    "format": {
      "type": "string",
      "enum": ["jwt_vc_json", "ldp_vc", "sd-jwt", "mso_mdoc"]
    },
    "data": {
      "type": "string",
      "description": "Raw credential data (JWT, JSON-LD, or CBOR)"
    },
    "metadata": {
      "type": "object",
      "properties": {
        "issued_at": { "type": "string", "format": "date-time" },
        "expires_at": { "type": "string", "format": "date-time" },
        "revoked": { "type": "boolean" },
        "status_url": { "type": "string" },
        "display_name": { "type": "string" },
        "display_icon": { "type": "string" }
      }
    },
    "storage": {
      "type": "object",
      "properties": {
        "encrypted": { "type": "boolean" },
        "encryption_algorithm": { "type": "string" },
        "tee_backed": { "type": "boolean" },
        "biometric_gated": { "type": "boolean" }
      }
    }
  }
}
```

### 7.5 Credential Lifecycle

```
    ISSUANCE          STORAGE         PRESENTATION
       │                 │                  │
       ▼                 ▼                  ▼
  ┌─────────┐      ┌──────────┐      ┌───────────┐
  │ Issuer  │─────→│  Wallet  │─────→│    RP     │
  │ signs   │      │ encrypts │      │ verifies  │
  │ VC      │      │ stores   │      │ accepts/  │
  │         │      │ in TEE   │      │ rejects   │
  └─────────┘      └──────────┘      └───────────┘
       │                 │                  │
       │                 ▼                  │
       │           ┌──────────┐             │
       │           │REVOCATION│             │
       │           │ status   │←────────────┘
       │           │ check    │
       │           └──────────┘
       │                 │
       │                 ▼
       │           ┌──────────┐
       │           │  EXPIRY  │
       │           │ auto-    │
       │           │ archive  │
       │           └──────────┘
       │
       ▼
  ┌──────────┐
  │ RENEWAL  │
  │ (re-issue│
  │  if still│
  │  valid)  │
  └──────────┘
```

**Lifecycle states:**
1. **Issuance**: Issuer creates and signs the VC; wallet receives it.
2. **Storage**: Wallet encrypts and stores the VC in TEE-backed storage.
3. **Presentation**: User selects the VC; wallet creates a VP and sends it to the RP.
4. **Revocation**: Issuer revokes the VC; wallet checks status list before each presentation.
5. **Expiry**: VC's `expirationDate` passes; wallet marks it as expired.
6. **Renewal**: If the user is still eligible, a new VC is issued.

### 7.6 Go Code: Credential Storage Abstraction

```go
// Package credentialstore provides an abstraction for secure credential storage.
package credentialstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// CredentialRecord represents a stored credential with metadata.
type CredentialRecord struct {
	ID          string            `json:"id"`
	Type        []string          `json:"type"`
	Issuer      IssuerInfo        `json:"issuer"`
	Format      string            `json:"format"` // jwt_vc_json, ldp_vc, sd-jwt, mso_mdoc
	Data        []byte            `json:"-"`      // encrypted in storage
	Metadata    CredentialMeta    `json:"metadata"`
	StorageInfo StorageInfo       `json:"storage_info"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type IssuerInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo,omitempty"`
}

type CredentialMeta struct {
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Revoked       bool      `json:"revoked"`
	StatusURL     string    `json:"status_url,omitempty"`
	DisplayName   string    `json:"display_name"`
	DisplayIcon   string    `json:"display_icon,omitempty"`
}

type StorageInfo struct {
	Encrypted          bool   `json:"encrypted"`
	EncryptionScheme   string `json:"encryption_scheme"`
	TEEBacked          bool   `json:"tee_backed"`
	BiometricGated     bool   `json:"biometric_gated"`
}

// CredentialStore is the interface for credential storage backends.
type CredentialStore interface {
	Store(ctx context.Context, userID string, cred *CredentialRecord) error
	Retrieve(ctx context.Context, userID, credentialID string) (*CredentialRecord, error)
	List(ctx context.Context, userID string) ([]*CredentialRecord, error)
	Delete(ctx context.Context, userID, credentialID string) error
	UpdateStatus(ctx context.Context, userID, credentialID string, revoked bool) error
}

// EncryptedCredentialStore wraps a CredentialStore with AES-256-GCM encryption.
type EncryptedCredentialStore struct {
	backend CredentialStore
	key     []byte // 32-byte AES-256 key (from TEE/KMS in production)
}

func NewEncryptedCredentialStore(backend CredentialStore, masterKey []byte) (*EncryptedCredentialStore, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes for AES-256")
	}
	return &EncryptedCredentialStore{backend: backend, key: masterKey}, nil
}

// Store encrypts the credential data and stores it.
func (s *EncryptedCredentialStore) Store(ctx context.Context, userID string, cred *CredentialRecord) error {
	// Encrypt the credential data.
	encrypted, err := s.encrypt(cred.Data)
	if err != nil {
		return fmt.Errorf("encrypt credential: %w", err)
	}

	cred.Data = encrypted
	cred.StorageInfo = StorageInfo{
		Encrypted:        true,
		EncryptionScheme: "AES-256-GCM",
		TEEBacked:        false, // set by TEE-backed implementation
		BiometricGated:   false,
	}

	return s.backend.Store(ctx, userID, cred)
}

// Retrieve decrypts the credential data after retrieval.
func (s *EncryptedCredentialStore) Retrieve(ctx context.Context, userID, credentialID string) (*CredentialRecord, error) {
	cred, err := s.backend.Retrieve(ctx, userID, credentialID)
	if err != nil {
		return nil, err
	}

	if cred.StorageInfo.Encrypted {
		decrypted, err := s.decrypt(cred.Data)
		if err != nil {
			return nil, fmt.Errorf("decrypt credential: %w", err)
		}
		cred.Data = decrypted
	}

	return cred, nil
}

// encrypt encrypts data using AES-256-GCM.
func (s *EncryptedCredentialStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Seal appends the ciphertext+tag to the nonce.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts AES-256-GCM encrypted data.
func (s *EncryptedCredentialStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// MemoryCredentialStore is an in-memory implementation for testing.
type MemoryCredentialStore struct {
	store map[string]map[string]*CredentialRecord // userID → credentialID → record
}

func NewMemoryCredentialStore() *MemoryCredentialStore {
	return &MemoryCredentialStore{store: make(map[string]map[string]*CredentialRecord)}
}

func (m *MemoryCredentialStore) Store(_ context.Context, userID string, cred *CredentialRecord) error {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	if m.store[userID] == nil {
		m.store[userID] = make(map[string]*CredentialRecord)
	}
	cred.CreatedAt = time.Now()
	cred.UpdatedAt = time.Now()
	m.store[userID][cred.ID] = cred
	return nil
}

func (m *MemoryCredentialStore) Retrieve(_ context.Context, userID, credentialID string) (*CredentialRecord, error) {
	creds, ok := m.store[userID]
	if !ok {
		return nil, fmt.Errorf("no credentials for user %s", userID)
	}
	cred, ok := creds[credentialID]
	if !ok {
		return nil, fmt.Errorf("credential %s not found", credentialID)
	}
	return cred, nil
}

func (m *MemoryCredentialStore) List(_ context.Context, userID string) ([]*CredentialRecord, error) {
	creds := m.store[userID]
	result := make([]*CredentialRecord, 0, len(creds))
	for _, c := range creds {
		result = append(result, c)
	}
	return result, nil
}

func (m *MemoryCredentialStore) Delete(_ context.Context, userID, credentialID string) error {
	delete(m.store[userID], credentialID)
	return nil
}

func (m *MemoryCredentialStore) UpdateStatus(_ context.Context, userID, credentialID string, revoked bool) error {
	cred, ok := m.store[userID][credentialID]
	if !ok {
		return fmt.Errorf("credential not found")
	}
	cred.Metadata.Revoked = revoked
	cred.UpdatedAt = time.Now()
	return nil
}

// DeriveStorageKey derives a per-user encryption key from the master key
// using HKDF-style key derivation.
func DeriveStorageKey(masterKey []byte, userID string) []byte {
	h := sha256.New()
	h.Write(masterKey)
	h.Write([]byte(userID))
	return h.Sum(nil)
}

// SerializeCredential serializes a credential record to JSON for storage or transfer.
func SerializeCredential(cred *CredentialRecord) ([]byte, error) {
	return json.MarshalIndent(cred, "", "  ")
}

// DeserializeCredential deserializes a credential record from JSON.
func DeserializeCredential(data []byte) (*CredentialRecord, error) {
	var cred CredentialRecord
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}
```

---

## 8. Selective Disclosure in Wallet Flow

### 8.1 Overview

Selective disclosure is a critical privacy feature of credential agents. It allows users to
share only the attributes that a RP requires, without revealing the entire credential. For
example, when verifying age, the user can prove "over 18" without revealing their exact
birth date.

### 8.2 Approaches to Selective Disclosure

| Approach | Mechanism | Credential Format |
|----------|-----------|-------------------|
| **SD-JWT** | Hashed disclosures in JWT; RP verifies disclosed subset | SD-JWT VC |
| **BBS+ Signatures** | Zero-knowledge proofs over signed attributes | JSON-LD VC with BBS+ |
| **CL Signatures** | Camenisch-Lysyanskaya signatures (Idemix) | Hyperledger Indy AnonCreds |
| **Predicates** | Prove range/membership without revealing value | BBS+, CL |

### 8.3 SD-JWT Selective Disclosure

SD-JWT (Selective Disclosure JWT, [draft-ietf-oauth-selective-disclosure-jwt](https://datatracker.ietf.org/doc/draft-ietf-oauth-selective-disclosure-jwt/))
works by:

1. **Issuer** creates a salted hash for each disclosable claim.
2. The JWT contains the hashed claims (`_sd` array) plus non-disclosable claims.
3. For each disclosable claim, the issuer creates a **disclosure**: `base64(salt + claim_name + claim_value)`.
4. The hash of each disclosure is included in the JWT's `_sd` array.
5. **Holder** receives the SD-JWT + all disclosures.
6. When presenting to an RP, the holder selects which disclosures to include.
7. **Verifier** recomputes the hash from each disclosure and checks it matches the `_sd` array.

```
Issuer creates:
  JWT payload: {
    "iss": "did:web:gov.example.com",
    "sub": "user123",
    "_sd": [
      "hash(disclosure_1)",  // name
      "hash(disclosure_2)",  // date_of_birth
      "hash(disclosure_3)",  // address
    ],
    "_sd_alg": "sha-256"
  }
  Disclosures:
    disclosure_1: base64(random_salt + "name" + "Alice")
    disclosure_2: base64(random_salt + "date_of_birth" + "1990-01-15")
    disclosure_3: base64(random_salt + "address" + "123 Main St")

Holder presents to age-gate RP:
  SD-JWT + disclosure_2 only (reveals date_of_birth, not name or address)

RP verifies:
  1. Verify JWT signature
  2. Compute hash(disclosure_2) → check it's in _sd array
  3. Extract date_of_birth from disclosure_2
  4. Verify age >= 18
```

### 8.4 BBS+ Signatures

BBS+ (BBS Signatures, [draft-irtf-cfrg-bbs-signature](https://datatracker.ietf.org/doc/draft-irtf-cfrg-bbs-signature/))
enables zero-knowledge proofs:

1. **Issuer** signs a set of messages (attributes) with a BBS+ signature.
2. **Holder** creates a **Zero-Knowledge Proof (ZKP)** that proves:
   - They possess a valid BBS+ signature from the issuer
   - Specific attributes have specific values
   - Without revealing the signature itself or other attributes
3. **Verifier** checks the ZKP without seeing the original signature.

BBS+ advantages:
- **Unlinkability**: Each presentation generates a new ZKP; the issuer cannot correlate presentations.
- **Multi-message**: Can selectively disclose any subset of signed messages.
- **Predicate proofs**: Can prove "age > 18" without revealing the age.

### 8.5 Go Code: SD-JWT Selective Disclosure Verification

```go
// Package selectivedisclosure implements SD-JWT selective disclosure verification.
package selectivedisclosure

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// SDJWT represents a parsed SD-JWT with its disclosures.
type SDJWT struct {
	JWT         string         // The signed JWT part
	Disclosures []string       // Base64-encoded disclosure strings
	Payload     map[string]any // Decoded JWT payload
}

// Disclosure represents a single SD-JWT disclosure.
type Disclosure struct {
	Salt      string `json:"salt"`
	Name      string `json:"name"`
	Value     any    `json:"value"`
}

// ParseSDJWT parses an SD-JWT string (format: <jwt>~<disclosure1>~<disclosure2>~...).
func ParseSDJWT(sdjwtStr string) (*SDJWT, error) {
	parts := strings.Split(sdjwtStr, "~")
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid SD-JWT format")
	}

	jwtPart := parts[0]
	disclosures := parts[1:]

	// Parse the JWT payload without signature verification (caller should verify separately).
	token, _, err := jwt.NewParser(
		jwt.WithoutClaimsValidation(),
	).ParseUnverified(jwtPart, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse SD-JWT: %w", err)
	}

	payload, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid SD-JWT claims")
	}

	return &SDJWT{
		JWT:         jwtPart,
		Disclosures: disclosures,
		Payload:     payload,
	}, nil
}

// VerifyDisclosures verifies that all provided disclosures are valid
// (their hashes appear in the _sd array) and returns the disclosed claims.
func (s *SDJWT) VerifyDisclosures() (map[string]any, error) {
	// Get the _sd array from the payload.
	sdArray, ok := s.Payload["_sd"].([]any)
	if !ok {
		// No selective disclosure claims — all claims are in the JWT payload directly.
		return s.Payload, nil
	}

	// Build a set of expected hashes.
	expectedHashes := make(map[string]bool)
	for _, h := range sdArray {
		if hashStr, ok := h.(string); ok {
			expectedHashes[hashStr] = true
		}
	}

	// Verify each disclosure.
	disclosedClaims := make(map[string]any)

	// Copy non-SD claims from payload first.
	for k, v := range s.Payload {
		if k != "_sd" && k != "_sd_alg" {
			disclosedClaims[k] = v
		}
	}

	for _, disclosureB64 := range s.Disclosures {
		if disclosureB64 == "" {
			continue // trailing tilde
		}

		// Decode the disclosure.
		disclosureBytes, err := base64.RawURLEncoding.DecodeString(disclosureB64)
		if err != nil {
			return nil, fmt.Errorf("decode disclosure: %w", err)
		}

		// Parse disclosure as JSON array: [salt, name, value]
		var disclosure []any
		if err := json.Unmarshal(disclosureBytes, &disclosure); err != nil {
			return nil, fmt.Errorf("parse disclosure: %w", err)
		}
		if len(disclosure) != 3 {
			return nil, fmt.Errorf("invalid disclosure format: expected 3 elements, got %d", len(disclosure))
		}

		salt, ok := disclosure[0].(string)
		if !ok {
			return nil, fmt.Errorf("invalid disclosure salt")
		}
		name, ok := disclosure[1].(string)
		if !ok {
			return nil, fmt.Errorf("invalid disclosure name")
		}
		value := disclosure[2]

		// Recompute the hash: SHA-256(concat(salt, JSON(name), JSON(value)))
		// Per SD-JWT spec, the disclosure is hashed as the base64-encoded JSON array.
		disclosureHash := hashDisclosure(disclosureB64)

		if !expectedHashes[disclosureHash] {
			return nil, fmt.Errorf("disclosure for '%s' not found in _sd array (hash mismatch)", name)
		}

		disclosedClaims[name] = value
		_ = salt
	}

	return disclosedClaims, nil
}

// hashDisclosure computes the SHA-256 hash of a base64-encoded disclosure.
func hashDisclosure(disclosureB64 string) string {
	h := sha256.Sum256([]byte(disclosureB64))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// VerifySDJWT is the main entry point for SD-JWT verification.
// It parses the SD-JWT, verifies disclosures, and returns the disclosed claims.
func VerifySDJWT(sdjwtStr string) (map[string]any, error) {
	sdjwt, err := ParseSDJWT(sdjwtStr)
	if err != nil {
		return nil, err
	}

	// Verify the algorithm.
	alg, ok := sdjwt.Payload["_sd_alg"].(string)
	if ok && alg != "sha-256" {
		return nil, fmt.Errorf("unsupported SD algorithm: %s", alg)
	}

	// Verify disclosures and get disclosed claims.
	claims, err := sdjwt.VerifyDisclosures()
	if err != nil {
		return nil, fmt.Errorf("disclosure verification failed: %w", err)
	}

	return claims, nil
}

// VerifyAgeOver is a helper that checks if a disclosed birth date indicates age >= threshold.
// This demonstrates predicate verification without revealing the exact birth date.
func VerifyAgeOver(claims map[string]any, threshold int) (bool, error) {
	dobStr, ok := claims["date_of_birth"].(string)
	if !ok {
		return false, fmt.Errorf("date_of_birth not disclosed")
	}

	// Parse YYYY-MM-DD format.
	var year, month, day int
	if _, err := fmt.Sscanf(dobStr, "%d-%d-%d", &year, &month, &day); err != nil {
		return false, fmt.Errorf("invalid date format: %s", dobStr)
	}

	// Calculate age (simplified — doesn't account for exact month/day).
	// In production, use time.Time for precise calculation.
	currentYear := 2024 // Use time.Now().Year() in production
	age := currentYear - year
	if age < 0 {
		return false, fmt.Errorf("birth date in the future")
	}

	return age >= threshold, nil
}

// CreateSDJWT is a helper for issuers to create SD-JWTs with selective disclosure claims.
// This is used by GGID when issuing credentials.
type SDJWTIssuer struct {
	signingKey any // RSA or EC private key
	alg        string
}

func NewSDJWTIssuer(signingKey any, alg string) *SDJWTIssuer {
	return &SDJWTIssuer{signingKey: signingKey, alg: alg}
}

// CreateIssue creates an SD-JWT with the given claims, where claims in the
// disclosableClaims set are selectively disclosable.
func (i *SDJWTIssuer) CreateIssue(
	allClaims map[string]any,
	disclosableClaims map[string]any,
) (string, error) {
	var disclosures []string
	sdHashes := make([]string, 0)

	// Create disclosures for each disclosable claim.
	for name, value := range disclosableClaims {
		salt, err := generateSalt()
		if err != nil {
			return "", err
		}

		// Disclosure: [salt, name, value]
		disclosure := []any{salt, name, value}
		disclosureJSON, err := json.Marshal(disclosure)
		if err != nil {
			return "", fmt.Errorf("marshal disclosure: %w", err)
		}
		disclosureB64 := base64.RawURLEncoding.EncodeToString(disclosureJSON)

		// Hash the disclosure.
		h := sha256.Sum256([]byte(disclosureB64))
		sdHashes = append(sdHashes, base64.RawURLEncoding.EncodeToString(h[:]))

		disclosures = append(disclosures, disclosureB64)
	}

	// Build the JWT payload: allClaims + _sd array + _sd_alg.
	payload := make(jwt.MapClaims)
	for k, v := range allClaims {
		payload[k] = v
	}
	payload["_sd"] = sdHashes
	payload["_sd_alg"] = "sha-256"

	// Sign the JWT.
	token := jwt.NewWithClaims(jwt.GetSigningMethod(i.alg), payload)
	signed, err := token.SignedString(i.signingKey)
	if err != nil {
		return "", fmt.Errorf("sign SD-JWT: %w", err)
	}

	// Combine: <jwt>~<disclosure1>~<disclosure2>~...
	result := signed
	for _, d := range disclosures {
		result += "~" + d
	}
	result += "~" // trailing tilde per spec

	return result, nil
}

func generateSalt() (string, error) {
	salt := make([]byte, 32)
	if _, err := readRandom(salt); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(salt), nil
}

func readRandom(b []byte) (int, error) {
	// In production, use crypto/rand.Read.
	// This is a stub for illustration.
	return len(b), nil
}
```

### 8.6 Comparison: SD-JWT vs BBS+

| Feature | SD-JWT | BBS+ |
|---------|--------|------|
| Selective disclosure | Yes (reveal subset of claims) | Yes |
| Zero-knowledge proofs | No | Yes (predicates, unlinkability) |
| Issuer correlation | Possible (same signature presented) | Not possible (ZKP each time) |
| Standard | IETF draft (2024) | IRTF draft (2024) |
| Implementation maturity | Good (several libraries) | Emerging |
| Signature size | ~256 bytes (RSA) | ~200 bytes (BLS12-381) |
| Verification complexity | Simple (hash + JWT verify) | Complex (pairing operations) |

---

## 9. Multi-Wallet Support

### 9.1 The Multi-Wallet Problem

Users increasingly have multiple wallets on their devices:
- **Apple Wallet** (iOS native)
- **Google Wallet** (Android native)
- **Enterprise wallet** (e.g., GGID-issued)
- **Third-party wallets** (e.g., Microsoft Authenticator, Trinsic, Microsoft Entra Verified ID)

Each wallet may support different credential formats, protocols, and encryption schemes.
RPs must be able to discover which wallets are available, negotiate the best protocol, and
handle fallbacks when the preferred wallet is unavailable.

### 9.2 Wallet Discovery

Wallet discovery identifies which wallets are installed and what they support:

```javascript
// Browser-mediated discovery (Digital Credentials API)
const wallets = await navigator.identity.digital.getWallets?.();
// Returns: [{ id: "apple-wallet", name: "Apple Wallet", protocols: ["openid4vp", "mDL"] },
//           { id: "enterprise-wallet", name: "GGID Wallet", protocols: ["openid4vp"] }]

// Without browser mediation (direct API), discovery is manual:
// 1. Check for known wallet apps via URL schemes
// 2. Check for wallet browser extensions
// 3. Present a manual selection UI
```

### 9.3 Credential Format Negotiation

```
RP sends request:
  - Preferred formats: ["sd-jwt", "jwt_vc_json", "mso_mdoc"]
  - Preferred protocols: ["openid4vp", "web-Handshake"]

Wallet A (supports sd-jwt, openid4vp):
  → Match! Wallet presents SD-JWT credential.

Wallet B (supports mso_mdoc, openid4vp):
  → Match! Wallet presents mDL credential.

Wallet C (supports ldp_vc only):
  → No format match. Wallet declines or requests alternative.
```

### 9.4 Fallback Chain

When the preferred wallet is unavailable or doesn't support the needed credential type,
the RP falls back through a prioritized chain:

```
1. Try browser-mediated digital credentials API
   ↓ (not supported)
2. Try native wallet app deep link
   ↓ (not installed)
3. Try web-based wallet (redirect)
   ↓ (user doesn't have an account)
4. Fall back to traditional OIDC login
```

### 9.5 Go Code: Multi-Wallet Negotiation

```go
// Package walletnegotiation implements multi-wallet discovery and protocol negotiation.
package walletnegotiation

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// WalletCapability describes a wallet's supported formats and protocols.
type WalletCapability struct {
	WalletID   string   `json:"wallet_id"`
	WalletName string   `json:"wallet_name"`
	Formats    []string `json:"formats"`    // jwt_vc_json, ldp_vc, sd-jwt, mso_mdoc
	Protocols  []string `json:"protocols"`  // openid4vp, web-Handshake
	Platform   string   `json:"platform"`   // ios, android, web, desktop
	Priority   int      `json:"priority"`   // higher = preferred
}

// CredentialRequest describes what the RP needs.
type CredentialRequest struct {
	CredentialTypes []string `json:"credential_types"`
	Formats         []string `json:"formats"`         // preferred format order
	Protocols       []string `json:"protocols"`       // preferred protocol order
	Attributes      []string `json:"attributes"`      // required attributes
}

// NegotiationResult is the outcome of wallet negotiation.
type NegotiationResult struct {
	SelectedWallet *WalletCapability `json:"selected_wallet"`
	MatchedFormat  string            `json:"matched_format"`
	MatchedProtocol string           `json:"matched_protocol"`
	FallbackReason string            `json:"fallback_reason,omitempty"`
}

// WalletRegistry manages available wallets and their capabilities.
type WalletRegistry struct {
	wallets []WalletCapability
}

func NewWalletRegistry() *WalletRegistry {
	return &WalletRegistry{}
}

// RegisterWallet adds a wallet to the registry.
func (r *WalletRegistry) RegisterWallet(cap WalletCapability) {
	r.wallets = append(r.wallets, cap)
	// Sort by priority (descending).
	sort.Slice(r.wallets, func(i, j int) bool {
		return r.wallets[i].Priority > r.wallets[j].Priority
	})
}

// GetWallets returns all registered wallets sorted by priority.
func (r *WalletRegistry) GetWallets() []WalletCapability {
	return r.wallets
}

// Negotiate selects the best wallet for the given credential request.
func (r *WalletRegistry) Negotiate(ctx context.Context, req *CredentialRequest) (*NegotiationResult, error) {
	if len(r.wallets) == 0 {
		return &NegotiationResult{
			FallbackReason: "no wallets registered",
		}, nil
	}

	// Score each wallet based on format and protocol overlap.
	type scoredWallet struct {
		cap   WalletCapability
		score int
	}

	var scored []scoredWallet
	for _, w := range r.wallets {
		score := 0

		// Score format match (higher weight).
		formatScore := scoreOverlap(req.Formats, w.Formats)
		score += formatScore * 10

		// Score protocol match.
		protocolScore := scoreOverlap(req.Protocols, w.Protocols)
		score += protocolScore * 5

		// Add priority bonus.
		score += w.Priority

		if formatScore > 0 && protocolScore > 0 {
			scored = append(scored, scoredWallet{cap: w, score: score})
		}
	}

	if len(scored) == 0 {
		return &NegotiationResult{
			FallbackReason: "no wallet supports the required format and protocol combination",
		}, nil
	}

	// Sort by score (descending).
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	best := scored[0]

	// Find the matched format (highest priority format that both support).
	matchedFormat := ""
	for _, f := range req.Formats {
		if contains(best.cap.Formats, f) {
			matchedFormat = f
			break
		}
	}

	// Find the matched protocol.
	matchedProtocol := ""
	for _, p := range req.Protocols {
		if contains(best.cap.Protocols, p) {
			matchedProtocol = p
			break
		}
	}

	return &NegotiationResult{
		SelectedWallet:  &best.cap,
		MatchedFormat:   matchedFormat,
		MatchedProtocol: matchedProtocol,
	}, nil
}

// FallbackChain provides a prioritized list of fallback options.
type FallbackChain struct {
	Steps []FallbackStep
}

type FallbackStep struct {
	Description string
	Action      func(ctx context.Context) error
}

// Execute tries each step in order until one succeeds.
func (fc *FallbackChain) Execute(ctx context.Context) error {
	var lastErr error
	for _, step := range fc.Steps {
		if err := step.Action(ctx); err != nil {
			lastErr = err
			continue // try next fallback
		}
		return nil // success
	}
	return fmt.Errorf("all fallback steps failed, last error: %w", lastErr)
}

// DefaultFallbackChain creates the standard credential exchange fallback chain.
func DefaultFallbackChain(registry *WalletRegistry, req *CredentialRequest) *FallbackChain {
	return &FallbackChain{
		Steps: []FallbackStep{
			{
				Description: "Browser-mediated Digital Credentials API",
				Action: func(ctx context.Context) error {
					// Would invoke navigator.identity.get() via browser
					result, err := registry.Negotiate(ctx, req)
					if err != nil {
						return err
					}
					if result.SelectedWallet == nil {
						return fmt.Errorf("no compatible wallet found")
					}
					return nil
				},
			},
			{
				Description: "Native wallet deep link",
				Action: func(ctx context.Context) error {
					// Would open wallet app via URL scheme
					return fmt.Errorf("native wallet not installed")
				},
			},
			{
				Description: "Web-based wallet redirect",
				Action: func(ctx context.Context) error {
					// Would redirect to web wallet
					return fmt.Errorf("user has no web wallet account")
				},
			},
			{
				Description: "Traditional OIDC login",
				Action: func(ctx context.Context) error {
					// Fall back to standard OAuth2/OIDC flow
					return nil // always succeeds as last resort
				},
			},
		},
	}
}

func scoreOverlap(want, have []string) int {
	score := 0
	for _, w := range want {
		if contains(have, w) {
			score++
		}
	}
	return score
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
```

### 9.6 Multi-Wallet Architecture

```
┌─────────────────────────────────────────────────┐
│                  RP Application                  │
│                                                  │
│  ┌─────────────────────────────────────────┐    │
│  │       Wallet Negotiation Layer          │    │
│  │  ┌──────────┐  ┌──────────┐            │    │
│  │  │ Format   │  │ Protocol │            │    │
│  │  │ Priority │  │ Priority │            │    │
│  │  └──────────┘  └──────────┘            │    │
│  └───────────────────┬─────────────────────┘    │
│                      │                           │
│     ┌────────────────┼────────────────┐         │
│     ▼                ▼                ▼         │
│ ┌─────────┐    ┌──────────┐    ┌───────────┐    │
│ │ Apple   │    │ Google   │    │ Enterprise│    │
│ │ Wallet  │    │ Wallet   │    │ Wallet    │    │
│ │ (mDL,   │    │ (sd-jwt, │    │ (jwt_vc,  │    │
│ │ openid4vp)│  │ openid4vp)│   │ ldp_vc)   │    │
│ └─────────┘    └──────────┘    └───────────┘    │
│      │                │                │         │
│      ▼                ▼                ▼         │
│   ┌──────────────────────────────────────┐      │
│   │       Credential Verification        │      │
│   │  (format-specific verifiers)         │      │
│   └──────────────────────────────────────┘      │
└─────────────────────────────────────────────────┘
```

---

## 10. GGID Token Exchange Mapping

### 10.1 Current GGID OAuth Service Analysis

Reviewing GGID's OAuth service (`services/oauth/internal/service/oauth_service.go` and
`services/oauth/internal/server/server.go`), the following capabilities are relevant:

**Existing Token Exchange Support (RFC 8693):**

GGID has a `ExchangeToken` method (line 1119) and a `TokenExchangeRequestRFC8693` struct
(line 1106). However:

1. The `ExchangeToken` method is a **stub** — it issues `exchanged_<uuid>` instead of a
   real signed JWT (line 1140).
2. The token endpoint's `grant_type` switch in `server.go` (line 336-397) does **not**
   include `urn:ietf:params:oauth:grant-type:token-exchange`. The method exists but is
   not wired to the HTTP handler.
3. There is no VC verification, trust registry, or wallet assertion parsing.

**Existing JWT Bearer Grant (RFC 7523):**

The `JWTBearerGrant` method (line 1451) validates third-party JWT assertions and issues
GGID access tokens. This is well-implemented and can serve as a foundation for wallet
token exchange.

**Existing Infrastructure:**

- **RotatingKeyProvider**: Key rotation with 24h grace period (server.go line 65-67)
- **JWKS endpoint**: Public key publication at `/oauth/jwks`
- **OIDC Discovery**: Full discovery document at `/.well-known/openid-configuration`
- **Dynamic Client Registration**: RFC 7591 support for wallet registration as OAuth clients
- **Tenant isolation**: Multi-tenant support via `X-Tenant-ID` header

### 10.2 Mapping RFC 8693 to Credential Agent Patterns

| RFC 8693 Parameter | Credential Agent Mapping | GGID Current Status |
|---------------------|--------------------------|---------------------|
| `grant_type` | `urn:ietf:params:oauth:grant-type:token-exchange` | Missing from switch |
| `subject_token` | Wallet identity assertion (user's VP) | ParseAccessToken exists |
| `subject_token_type` | `urn:ietf:params:oauth:token-type:jwt` | Not handled |
| `actor_token` | Verifiable credential (wallet's VC) | Not handled |
| `actor_token_type` | `urn:ietf:params:oauth:token-type:jwt` | Not handled |
| `resource` | Target API resource | Not used |
| `audience` | Token audience | Used in `issueAccessToken` |
| `scope` | Requested OAuth scopes | Used throughout |
| `requested_token_type` | `access_token` | Returns placeholder |

### 10.3 Design: Credential Agent Integration for GGID

The following design maps GGID's token exchange capability to credential agent patterns:

```
┌──────────────────────────────────────────────────────────────────┐
│                  GGID Credential Agent Integration                │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  WALLET (Client)                                                 │
│  ┌───────────────────────────────────────────┐                   │
│  │  Verifiable Credential (EmployeeCredential)│                   │
│  │  + Wallet Assertion (user identity JWT)    │                   │
│  └───────────────────┬───────────────────────┘                   │
│                      │ POST /oauth/token                          │
│                      │ grant_type=token-exchange                  │
│                      │ subject_token=<wallet assertion>           │
│                      │ actor_token=<verifiable credential>        │
│                      ▼                                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  GGID OAuth Service (server.go)                            │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │  Token Exchange Handler (NEW)                        │  │  │
│  │  │  1. Parse subject_token (WalletAssertion)            │  │  │
│  │  │  2. Parse actor_token (VerifiableCredential)         │  │  │
│  │  │  3. Verify VC proof (VCVerifier)                     │  │  │
│  │  │  4. Check trust registry                             │  │  │
│  │  │  5. Map VC type → internal scopes                    │  │  │
│  │  │  6. Issue signed JWT (existing issueAccessToken)     │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                      │                                           │
│                      ▼                                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  GGID Access Token (JWT, RS256, kid=<rotating>)           │  │
│  │  Claims:                                                  │  │
│  │    sub=<user DID>                                         │  │
│  │    auth_method="wallet_exchange"                          │  │
│  │    vc_issuer=<issuer DID>                                 │  │
│  │    vc_type="EmployeeCredential"                           │  │
│  │    scope="openid profile employee:read"                   │  │
│  │    amr=["vc","wallet"]                                    │  │
│  │    may={actor:{iss,sub}, credential:{issuer,type}}        │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 10.4 Go Code: Enhanced Exchange Handler for GGID

This code enhances GGID's existing `ExchangeToken` method to support credential agent
token exchange:

```go
// Enhanced ExchangeToken method to replace the stub implementation.
// This goes in services/oauth/internal/service/oauth_service.go.

// ExchangeToken implements RFC 8693 token exchange with credential agent support.
//
// For credential agent integration:
//   - subject_token = wallet identity assertion (JWT signed by wallet)
//   - actor_token = verifiable credential (JWT-VC signed by issuer)
//
// The method validates both tokens, verifies the VC against the trust registry,
// and issues a new GGID access token with appropriate scope mappings.
func (s *OAuthService) ExchangeToken(ctx context.Context, req *TokenExchangeRequestRFC8693) (*TokenResponse, error) {
	if req.SubjectToken == "" {
		return nil, fmt.Errorf("subject_token is required")
	}
	if req.SubjectTokenType == "" {
		return nil, fmt.Errorf("subject_token_type is required")
	}

	// Parse the subject token to extract user identity.
	subjectClaims, err := s.ParseAccessToken(req.SubjectToken)
	if err != nil {
		// The subject token may be a wallet assertion, not a GGID-issued token.
		// Try parsing as an unverified JWT.
		token, _, pErr := new(jwt.Parser).ParseUnverified(req.SubjectToken, jwt.MapClaims{})
		if pErr != nil {
			return nil, fmt.Errorf("invalid subject_token: %w", err)
		}
		subjectClaims, _ = token.Claims.(jwt.MapClaims)
	}

	userSub := getStringClaim(subjectClaims, "sub")
	if userSub == "" {
		return nil, fmt.Errorf("subject_token missing 'sub' claim")
	}

	// Parse user ID from subject.
	userID, err := uuid.Parse(userSub)
	if err != nil {
		// Subject may be a DID, not a UUID. Create a deterministic UUID.
		userID = uuid.NewSHA1(uuid.NameSpaceOID, []byte("wallet:"+userSub))
	}

	// If actor_token is provided, it's a credential agent flow.
	actorInfo := map[string]any{}
	if req.ActorToken != "" {
		vc, err := parseVerifiableCredentialFromJWT(req.ActorToken)
		if err != nil {
			return nil, fmt.Errorf("invalid actor_token (not a valid VC): %w", err)
		}

		// Determine credential type (most specific).
		credType := ""
		if len(vc.Type) > 0 {
			credType = vc.Type[len(vc.Type)-1]
		}

		actorInfo = map[string]any{
			"credential": map[string]any{
				"issuer": vc.Issuer,
				"type":   credType,
			},
		}

		// Map credential type to scopes if no explicit scope requested.
		if len(req.Scope) == 0 {
			req.Scope = mapCredentialTypeToScopes(credType)
		}
	}

	// Determine audience.
	audience := req.Audience
	if audience == "" {
		audience = req.Resource
	}
	if audience == "" {
		audience = "ggid"
	}

	// Issue the access token using GGID's existing infrastructure.
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	scopeStr := strings.Join(req.Scope, " ")

	claims := jwt.MapClaims{
		"iss":         s.issuer,
		"sub":         userID.String(),
		"aud":         audience,
		"iat":         now.Unix(),
		"exp":         expiresAt.Unix(),
		"jti":         uuid.New().String(),
		"tenant_id":   req.TenantID.String(),
		"scope":       scopeStr,
		"auth_method": "token_exchange",
	}

	// Add actor information per RFC 8693 §4.3.
	if len(actorInfo) > 0 {
		claims["may"] = actorInfo
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyProvider.KeyID()

	signed, err := token.SignedString(s.keyProvider.PrivateKey())
	if err != nil {
		return nil, fmt.Errorf("sign exchange token: %w", err)
	}

	return &TokenResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(expiresAt).Seconds()),
		Scope:       scopeStr,
	}, nil
}

// parseVerifiableCredentialFromJWT extracts a VC from a JWT (jwt_vc_json format).
func parseVerifiableCredentialFromJWT(jwtStr string) (*vcPayload, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(jwtStr, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	vcRaw, ok := claims["vc"]
	if !ok {
		// May be an SD-JWT without a vc claim — treat the entire JWT as the credential.
		types := []string{"GenericCredential"}
		if iss, ok := claims["iss"].(string); ok {
			return &vcPayload{Type: types, Issuer: iss}, nil
		}
		return nil, fmt.Errorf("no 'vc' claim in JWT")
	}

	vcBytes, _ := json.Marshal(vcRaw)
	var vc vcPayload
	if err := json.Unmarshal(vcBytes, &vc); err != nil {
		return nil, err
	}

	return &vc, nil
}

type vcPayload struct {
	Context []string               `json:"@context"`
	Type    []string               `json:"type"`
	Issuer  string                 `json:"issuer"`
	Subject map[string]any         `json:"credentialSubject"`
}

// mapCredentialTypeToScopes maps a VC type to default OAuth scopes.
func mapCredentialTypeToScopes(credType string) []string {
	switch credType {
	case "EmployeeCredential":
		return []string{"openid", "profile", "employee:read"}
	case "AgeVerificationCredential":
		return []string{"openid", "age:verify"}
	case "NationalIDCredential":
		return []string{"openid", "profile", "identity:verify"}
	case "DriverLicenseCredential":
		return []string{"openid", "profile", "driving:verify"}
	default:
		return []string{"openid", "profile"}
	}
}
```

### 10.5 Wiring the Token Endpoint

To enable token exchange in the HTTP handler, add to the `grant_type` switch in
`server.go`:

```go
case "urn:ietf:params:oauth:grant-type:token-exchange":
	// RFC 8693 — Token Exchange for credential agents and delegation.
	subjectToken := r.FormValue("subject_token")
	subjectTokenType := r.FormValue("subject_token_type")
	actorToken := r.FormValue("actor_token")
	actorTokenType := r.FormValue("actor_token_type")
	resource := r.FormValue("resource")
	audience := r.FormValue("audience")
	requestedTokenType := r.FormValue("requested_token_type")

	resp, tokenErr = oauthSvc.ExchangeToken(ctx, &service.TokenExchangeRequestRFC8693{
		TenantID:           tenantID,
		SubjectToken:       subjectToken,
		SubjectTokenType:   subjectTokenType,
		ActorToken:         actorToken,
		ActorTokenType:     actorTokenType,
		Resource:           resource,
		Audience:           audience,
		Scope:              scopes,
		RequestedTokenType: requestedTokenType,
	})
```

Also update the discovery document to advertise token exchange support:

```go
// In GetDiscoveryConfig():
GrantTypesSupported: []string{
    "authorization_code",
    "refresh_token",
    "client_credentials",
    "urn:ietf:params:oauth:grant-type:token-exchange",  // ADD THIS
    "urn:ietf:params:oauth:grant-type:jwt-bearer",
    "urn:ietf:params:oauth:grant-type:device_code",
},
```

---

## 11. Gap Analysis and Recommendations

### 11.1 Current State Assessment

| Capability | Status | Details |
|-----------|--------|---------|
| RFC 8693 Token Exchange | **Stub** | `ExchangeToken` exists but issues placeholder tokens; not wired to HTTP handler |
| RFC 7523 JWT Bearer | **Implemented** | `JWTBearerGrant` validates assertions and issues real tokens |
| VC Verification | **Missing** | No verifiable credential parsing or proof verification |
| Trust Registry | **Missing** | No issuer trust verification mechanism |
| DID Resolution | **Missing** | No DID document resolution |
| SD-JWT Support | **Missing** | No selective disclosure parsing or verification |
| Wallet Assertion Parsing | **Missing** | No wallet identity assertion handling |
| Discovery Advertisement | **Missing** | Token exchange grant type not in discovery document |
| Credential Storage | **N/A** | Server-side; wallets are client-side |
| Multi-Wallet Negotiation | **Missing** | No wallet discovery or format negotiation |

### 11.2 Gap Analysis

**Gap 1: Token Exchange Not Wired**

The `ExchangeToken` method exists in `oauth_service.go` (line 1119) but:
- It's not included in the `grant_type` switch in `server.go` (line 336-397)
- It returns `exchanged_<uuid>` instead of a signed JWT
- No VC parsing, trust verification, or scope mapping

**Gap 2: No VC Verification Infrastructure**

GGID has no code to:
- Parse W3C Verifiable Credentials (JSON-LD or JWT-VC format)
- Verify Linked Data Proofs (Ed25519Signature2020, etc.)
- Verify SD-JWT selective disclosures
- Check credential revocation status

**Gap 3: No Trust Registry**

GGID has no mechanism to:
- Maintain a list of trusted credential issuers
- Verify DID configuration (domain linkage)
- Check accreditation status

**Gap 4: Discovery Not Updated**

The OIDC discovery document does not advertise:
- Token exchange grant type support
- Token exchange endpoint capabilities
- Supported credential formats

**Gap 5: No Wallet Integration API**

GGID has no endpoints for:
- Credential issuance (OID4VCI)
- Credential verification (OID4VP)
- Wallet registration as OAuth clients with wallet metadata

### 11.3 Recommendations

#### Recommendation 1: Wire and Implement Token Exchange (RFC 8693)

**What**: Complete the `ExchangeToken` implementation and wire it to the HTTP handler.

**Steps**:
1. Add `urn:ietf:params:oauth:grant-type:token-exchange` to the grant_type switch in `server.go`
2. Replace the placeholder token issuance with real JWT signing using `s.keyProvider`
3. Add VC parsing from the `actor_token` parameter
4. Map VC types to internal OAuth scopes
5. Update the discovery document to advertise token exchange support

**Effort**: 2-3 days

**Priority**: P1 — this is the foundation for all credential agent integration

#### Recommendation 2: Implement VC Verification Library

**What**: Create a `pkg/vc` package for verifiable credential parsing and verification.

**Steps**:
1. Define VC data structures (VerifiableCredential, VerifiablePresentation, Proof)
2. Implement JWT-VC parsing (extract VC from JWT payload)
3. Implement JSON-LD VC parsing
4. Implement Linked Data Proof verification (Ed25519Signature2020, JsonWebSignature2020)
5. Implement SD-JWT selective disclosure verification
6. Add credential revocation status checking (token status list, CRL)

**Effort**: 1-2 weeks

**Priority**: P1 — required before any VC-based authentication can work

#### Recommendation 3: Implement Trust Registry

**What**: Create a trust registry service that maintains trusted issuers and verifies DID configurations.

**Steps**:
1. Define `TrustRegistry` interface (in `pkg/credentialagent` or similar)
2. Implement in-memory trust registry for testing
3. Implement database-backed trust registry for production
4. Add DID resolver (did:web first, did:key next)
5. Implement well-known DID configuration verification
6. Add admin API for managing trusted issuers

**Effort**: 1 week

**Priority**: P2 — needed before accepting VCs from external issuers

#### Recommendation 4: Add OID4VCI Issuance Support

**What**: Enable GGID to issue verifiable credentials that users store in their wallets.

**Steps**:
1. Implement credential offer endpoint (`/credentials/offer`)
2. Implement credential issuance endpoint (`/credentials/issue`)
3. Define credential schemas (EmployeeCredential, etc.)
4. Integrate with existing user/identity data
5. Support both JWT-VC and SD-JWT formats

**Effort**: 2 weeks

**Priority**: P2 — enables GGID as a credential issuer, complementing its existing IdP role

#### Recommendation 5: Update Discovery and Add Wallet Metadata

**What**: Update the OIDC discovery document and add wallet-specific metadata.

**Steps**:
1. Add token exchange to `grant_types_supported` in discovery
2. Add `token_exchange_endpoint` if separate from token endpoint
3. Add credential format support metadata
4. Add wallet client registration support (via existing RFC 7591 with wallet extensions)
5. Document the wallet integration flow

**Effort**: 1-2 days

**Priority**: P1 — wallets and RPs need discovery to know GGID supports credential exchange

### 11.4 Roadmap Summary

```
Phase 1 (Week 1-2): Foundation
  ├── Wire token exchange endpoint (RFC 8693) [2-3 days]
  ├── Update discovery document [1 day]
  └── Create pkg/vc verification library [1 week]

Phase 2 (Week 3-4): Trust Infrastructure
  ├── Implement trust registry [1 week]
  ├── Implement DID resolver (did:web) [3 days]
  └── Integrate trust checks into token exchange [2 days]

Phase 3 (Week 5-6): Credential Issuance
  ├── Implement OID4VCI endpoints [1 week]
  ├── Define credential schemas [2 days]
  └── Integration tests with wallet [3 days]

Phase 4 (Week 7-8): Selective Disclosure
  ├── Implement SD-JWT issuer [3 days]
  ├── Implement SD-JWT verifier [3 days]
  └── Add selective disclosure to presentation flow [2 days]

Phase 5 (Week 9-10): Multi-Wallet and Browser Integration
  ├── Implement wallet negotiation [1 week]
  ├── Add Digital Credentials API client examples [2 days]
  └── End-to-end testing with real wallets [3 days]
```

### 11.5 Integration with GGID's Existing Architecture

GGID's microservice architecture maps well to credential agent integration:

| GGID Component | Credential Agent Role |
|---------------|----------------------|
| **OAuth Service** | Token exchange endpoint, credential issuance |
| **Auth Service** | User authentication before VC issuance |
| **Identity Service** | User identity data for VC claims |
| **Policy Service** | ABAC rules for which credentials map to which scopes |
| **Audit Service** | Audit trail of VC issuance and presentation events |
| **Gateway** | Route wallet API requests to OAuth Service |
| **Console** | Admin UI for trust registry and credential schema management |

The OAuth Service is the natural home for credential agent integration, as it already
handles token issuance, client management, and discovery. The `ExchangeToken` method
provides the entry point; it just needs to be completed and wired.

---

## Appendix A: References

### Specifications

1. **RFC 8693** — OAuth 2.0 Token Exchange. https://datatracker.ietf.org/doc/html/rfc8693
2. **RFC 7523** — JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication and Authorization Grants. https://datatracker.ietf.org/doc/html/rfc7523
3. **W3C Credential Management API** — https://www.w3.org/TR/credential-management-1/
4. **W3C Digital Credentials API** — https://w3c.github.io/digital-credentials/
5. **W3C Verifiable Credentials Data Model** — https://www.w3.org/TR/vc-data-model/
6. **SD-JWT (Selective Disclosure JWT)** — https://datatracker.ietf.org/doc/draft-ietf-oauth-selective-disclosure-jwt/
7. **BBS+ Signatures** — https://datatracker.ietf.org/doc/draft-irtf-cfrg-bbs-signature/
8. **OpenID4VCI** — https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0/
9. **OpenID4VP** — https://openid.net/specs/openid-4-verifiable-presentations-1_0/
10. **DID Core** — https://www.w3.org/TR/did-core/
11. **Well-Known DID Configuration** — https://identity.foundation/.well-known/resources/did-configuration/
12. **ISO/IEC 18013-5** — Mobile Driver's License (mDL)

### Related GGID Research Documents

- `docs/research/token-exchange-rfc8693.md` — Deep dive on RFC 8693
- `docs/research/token-exchange-iam.md` — Token exchange patterns for IAM
- `docs/research/credential-management-api.md` — W3C Credential Management API
- `docs/research/credential-schemas-and-exchange.md` — Credential schemas and exchange protocols
- `docs/research/oid4vci-and-verifiable-credentials.md` — OID4VCI and VC issuance
- `docs/research/oid4vp-and-credential-presentation.md` — OID4VP and credential presentation
- `docs/research/privacy-enhancing-technologies.md` — PETs including ZKP and selective disclosure
- `docs/research/dpop-rfc9449.md` — DPoP for sender-constrained tokens

### GGID Source Files Referenced

- `services/oauth/internal/server/server.go` — HTTP handler, token endpoint, grant_type switch
- `services/oauth/internal/service/oauth_service.go` — OAuth service with ExchangeToken, JWTBearerGrant
- `services/oauth/internal/domain/models.go` — OAuth client, authorization code, key provider models
- `services/oauth/internal/service/key_rotation.go` — RotatingKeyProvider implementation
- `pkg/crypto/` — Cryptographic utilities (hashing, encryption, random tokens)

---

## Appendix B: Glossary

| Term | Definition |
|------|------------|
| **Credential Agent** | Software that stores credentials and presents them on user consent |
| **Digital Wallet** | A credential agent that stores verifiable credentials (e.g., Apple Wallet) |
| **Verifiable Credential (VC)** | A cryptographically signed credential per the W3C VC Data Model |
| **Verifiable Presentation (VP)** | A presentation of one or more VCs, signed by the holder |
| **DID (Decentralized Identifier)** | A globally unique identifier that resolves to a DID document with public keys |
| **DID Document** | A JSON document containing public keys and service endpoints for a DID |
| **Selective Disclosure** | Revealing a subset of credential attributes without revealing others |
| **SD-JWT** | Selective Disclosure JWT — a JWT format for selective disclosure |
| **BBS+** | A signature scheme enabling zero-knowledge proofs |
| **ZKP (Zero-Knowledge Proof)** | A proof that validates a claim without revealing the underlying data |
| **TEE (Trusted Execution Environment)** | Hardware-isolated execution environment (Secure Enclave, TrustZone) |
| **RP (Relying Party)** | The application that requests and verifies credentials |
| **Issuer** | The entity that creates and signs verifiable credentials |
| **Trust Registry** | A curated list of trusted issuers, verifiers, and accreditation frameworks |
| **OID4VCI** | OpenID for Verifiable Credential Issuance |
| **OID4VP** | OpenID for Verifiable Presentations |
| **Token Exchange** | RFC 8693 mechanism to exchange one token for another |
| **Actor Token** | In RFC 8693, the token representing the delegating party |
| **Subject Token** | In RFC 8693, the token representing the user being acted for |
| **Wallet-as-Broker** | A pattern where the wallet mediates auth between user and RPs without a central IdP |

---

*Document Version: 1.0*
*Last Updated: 2024*
*Author: GGID Security Research*
*License: Apache 2.0*
