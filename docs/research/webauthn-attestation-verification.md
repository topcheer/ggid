# WebAuthn Attestation Verification Implementation for IAM

> **Scope:** Verification code, trust scoring algorithms, per-tenant policy,
> and GGID-specific gap analysis. For attestation format descriptions
> (packed/tpm/android-key/apple/fido-u2f), see
> [webauthn-attestation-chain.md](./webauthn-attestation-chain.md).
> For FIDO MDS blob fetch/verify and AAGUID lookup, see
> [fido-mds-attestation.md](./fido-mds-attestation.md).

---

## 1. Attestation Verification Pipeline

The end-to-end flow takes a raw `attestationObject` (CBOR-encoded) from the
authenticator and produces a `VerificationResult` containing the attestation
type, trust score, and authenticator metadata. The pipeline has five stages:

```
authData          attStmt (CBOR map)
    |                   |
    v                   v
[1. Parse AuthData] --> [2. Extract Format] --> [3. Format-Specific Verify]
                                                        |
                                                        v
                              [4. AAGUID Lookup] --> [5. Trust Score]
                                                        |
                                                        v
                                              VerificationResult
```

### Unified Dispatcher

```go
package webauthn

import (
	"crypto/sha256"
	"encoding/asn1"
	"fmt"
)

// VerificationResult holds the output of a full attestation verification.
type VerificationResult struct {
	Format          string         // "packed", "tpm", "apple", "none", etc.
	AAGUID          string         // canonical UUID form
	TrustScore      int            // 0-100
	AttestationType AttestationKind // full, basic, self, none
	Authenticator   *AuthenticatorInfo
	StatusReport    *StatusReport  // from MDS, may be nil
	Warning         []string       // non-fatal issues (e.g., self-signed cert)
}

// AttestationKind classifies the attestation trust level.
type AttestationKind string

const (
	AttestationNone     AttestationKind = "none"
	AttestationSelf     AttestationKind = "self"
	AttestationBasic    AttestationKind = "basic"
	AttestationAttCA    AttestationKind = "attca"  // full chain
	AttestationUnknown  AttestationKind = "unknown"
)

// AttestationInput is the raw data needed to perform verification.
type AttestationInput struct {
	AuthData       []byte
	ClientDataHash []byte // SHA-256 of clientDataJSON
	Format         string
	AttStmt        map[string]interface{} // parsed CBOR attStmt
}

// VerifyAttestation is the unified dispatcher for all attestation formats.
// It orchestrates the five-stage pipeline and returns a complete result.
func VerifyAttestation(
	input AttestationInput,
	registry *AAGUIDRegistry,
	policy *TenantPolicy,
) (*VerificationResult, error) {

	result := &VerificationResult{Format: input.Format}

	// Stage 1: Parse authenticator data to extract AAGUID.
	aaguidBytes := ExtractAAGUIDFromAuthData(input.AuthData)
	result.AAGUID = formatAAGUID(aaguidBytes)

	// Stage 2: Format-specific cryptographic verification.
	kind, err := verifyByFormat(input)
	if err != nil {
		return result, fmt.Errorf("attestation verification (%s): %w",
			input.Format, err)
	}
	result.AttestationType = kind

	// Stage 3: AAGUID lookup against registered metadata.
	if registry != nil {
		result.Authenticator = registry.Lookup(result.AAGUID)
		result.StatusReport = registry.GetStatus(result.AAGUID)
	}

	// Stage 4: Compute trust score.
	result.TrustScore = ComputeTrustScore(result)

	// Stage 5: Enforce tenant policy (if provided).
	if policy != nil {
		if err := policy.Enforce(result); err != nil {
			return result, fmt.Errorf("policy violation: %w", err)
		}
	}

	return result, nil
}

func verifyByFormat(input AttestationInput) (AttestationKind, error) {
	switch input.Format {
	case "none":
		return AttestationNone, nil

	case "packed":
		return verifyPackedFull(input)

	case "tpm":
		return verifyTPMFull(input)

	case "android-key":
		return verifyAndroidKeyFull(input)

	case "apple":
		return verifyAppleFull(input)

	case "fido-u2f":
		return verifyFidoU2FFull(input)

	case "android-safetynet":
		return verifyAndroidSafetynet(input)

	default:
		return AttestationUnknown, fmt.Errorf(
			"unsupported attestation format: %s", input.Format)
	}
}
```

The dispatcher decouples format-specific logic from the higher-level
pipeline. Each `verify*Full` function returns an `AttestationKind` so the
trust score calculator can distinguish self-attestation from full attestation
without re-parsing.

---

## 2. Per-Format Verification Implementations

> Format descriptions (field layout, signature schemes, trust models) are
> covered in [webauthn-attestation-chain.md](./webauthn-attestation-chain.md).
> This section focuses on **verification code**.

### 2a. Packed Attestation (x5c chain or self/ECDAA)

The packed format supports both attestation certificate chains (x5c array)
and self-attestation (ecdak or direct signing with the credential public key).
GGID's existing `VerifyPackedAttestation` handles only the signature check.
This extended version also validates the certificate chain to an MDS root.

```go
// verifyPackedFull performs full packed attestation verification including
// certificate chain validation and self-attestation handling.
func verifyPackedFull(input AttestationInput) (AttestationKind, error) {
	alg, _ := input.AttStmt["alg"].(int64)
	sig, _ := input.AttStmt["sig"].([]byte)
	if len(sig) == 0 {
		return AttestationUnknown, fmt.Errorf("packed: missing signature")
	}

	signedData := append(input.AuthData, input.ClientDataHash...)

	// Case 1: x5c certificate chain present -> full attestation.
	if x5cRaw, ok := input.AttStmt["x5c"].([]interface{}); ok && len(x5cRaw) > 0 {
		certs, err := parseX5CChain(x5cRaw)
		if err != nil {
			return AttestationUnknown, fmt.Errorf("packed: parse x5c: %w", err)
		}

		// Verify chain terminates at an MDS root (see Section 5).
		if err := ValidateCertChain(certs, MDSRootPool); err != nil {
			return AttestationUnknown, fmt.Errorf(
				"packed: cert chain validation: %w", err)
		}

		// Verify the leaf cert's public key signed the attestation data.
		if err := verifySignatureWithCert(certs[0], int(alg),
			signedData, sig); err != nil {
			return AttestationUnknown, fmt.Errorf(
				"packed: signature verification: %w", err)
		}

		// Verify the AAGUID in the cert extension matches authData AAGUID.
		if err := verifyCertAAGUID(certs[0], input.AuthData); err != nil {
			return AttestationUnknown, fmt.Errorf(
				"packed: AAGUID mismatch: %w", err)
		}

		return AttestationAttCA, nil
	}

	// Case 2: No x5c -> self-attestation. The credential public key itself
	// signed the attestation. Verify against COSE key in attStmt.
	if coseKey, ok := input.AttStmt["ecdaaKeyId"]; ok {
		// ECDAA anonymous attestation — requires ECDAA verification library.
		_ = coseKey
		return AttestationSelf, fmt.Errorf(
			"packed: ECDAA not yet implemented")
	}

	// Self-attestation: verify using the credential's own COSE public key.
	// The COSE key is NOT in attStmt for packed self-attestation; it must
	// be extracted from the authData's attestedCredentialData.
	pubKey, err := extractCOSEKeyFromAuthData(input.AuthData)
	if err != nil {
		return AttestationUnknown, fmt.Errorf(
			"packed self: extract public key: %w", err)
	}
	if err := verifySignatureWithCOSEKey(pubKey, int(alg),
		signedData, sig); err != nil {
		return AttestationUnknown, fmt.Errorf(
			"packed self: signature verification: %w", err)
	}

	return AttestationSelf, nil
}
```

### 2b. TPM Attestation

TPM attestation verifies that the credential was created inside a hardware
TPM. The verification requires parsing the TPM `pubArea` and `certInfo`
structures from the attStmt and validating the cert chain against the TPM
vendor root.

```go
// verifyTPMFull validates TPM attestation.
func verifyTPMFull(input AttestationInput) (AttestationKind, error) {
	version, _ := input.AttStmt["ver"].(string)
	if version != "2.0" {
		return AttestationUnknown, fmt.Errorf(
			"tpm: unsupported version %q", version)
	}

	sig, _ := input.AttStmt["sig"].([]byte)
	certInfo, _ := input.AttStmt["certInfo"].([]byte) // TPMS_ATTEST
	pubArea, _ := input.AttStmt["pubArea"].([]byte)    // TPM2B_PUBLIC

	if len(sig) == 0 || len(certInfo) == 0 || len(pubArea) == 0 {
		return AttestationUnknown, fmt.Errorf(
			"tpm: missing certInfo, pubArea, or sig")
	}

	// Parse x5c chain (TPM attestation always uses x5c).
	x5cRaw, _ := input.AttStmt["x5c"].([]interface{})
	certs, err := parseX5CChain(x5cRaw)
	if err != nil {
		return AttestationUnknown, fmt.Errorf("tpm: parse x5c: %w", err)
	}

	// Validate cert chain to MDS root.
	if err := ValidateCertChain(certs, MDSRootPool); err != nil {
		return AttestationUnknown, fmt.Errorf(
			"tpm: cert chain: %w", err)
	}

	// Parse TPM2B_PUBLIC to extract the public key and its parameters.
	tpmPub, err := parseTPMPubArea(pubArea)
	if err != nil {
		return AttestationUnknown, fmt.Errorf(
			"tpm: parse pubArea: %w", err)
	}

	// The signed data is the SHA-256 of (authData || clientDataHash).
	signedData := append(input.AuthData, input.ClientDataHash...)
	digest := sha256.Sum256(signedData)

	// Verify the TPM EK certificate signed certInfo, which contains
	// a hash of the signed data.
	if err := verifyTPMAttestationSig(
		certs[0], tpmPub, certInfo, digest[:], sig); err != nil {
		return AttestationUnknown, fmt.Errorf(
			"tpm: signature verification: %w", err)
	}

	return AttestationAttCA, nil
}

// parseTPMPubArea parses the TPM2B_PUBLIC structure to extract the
// public area used for verification.
func parseTPMPubArea(data []byte) (*TPMPublicArea, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("pubArea too short")
	}
	// TPM2B_PUBLIC = 2-byte size + TPMU_PUBLIC_PARMS + TPMU_PUBLIC_ID
	// This is a simplified parser; production should use a full TPM struct.
	return &TPMPublicArea{
		Raw: data,
	}, nil
}

type TPMPublicArea struct {
	Raw []byte
}
```

### 2c. Android Key Attestation

Android KeyStore attestation uses an X.509 certificate chain rooted at
Google's hardware attestation root. The leaf certificate contains key
description extension (OID 1.3.6.1.4.1.11129.2.1.17) that must match the
challenge and RP ID hash.

```go
// verifyAndroidKeyFull validates Android KeyStore attestation.
func verifyAndroidKeyFull(input AttestationInput) (AttestationKind, error) {
	sig, _ := input.AttStmt["sig"].([]byte)
	if len(sig) == 0 {
		return AttestationUnknown, fmt.Errorf("android-key: missing sig")
	}

	x5cRaw, _ := input.AttStmt["x5c"].([]interface{})
	certs, err := parseX5CChain(x5cRaw)
	if err != nil {
		return AttestationUnknown, fmt.Errorf("android-key: x5c: %w", err)
	}

	// Validate chain to Google hardware attestation root.
	if err := ValidateCertChain(certs, GoogleRootPool); err != nil {
		return AttestationUnknown, fmt.Errorf(
			"android-key: chain validation: %w", err)
	}

	// Parse the key description extension from the leaf cert.
	keyDesc, err := parseAndroidKeyDescription(certs[0])
	if err != nil {
		return AttestationUnknown, fmt.Errorf(
			"android-key: key description extension: %w", err)
	}

	// Verify the challenge field matches clientDataHash.
	signedData := append(input.AuthData, input.ClientDataHash...)
	digest := sha256.Sum256(signedData)

	if len(keyDesc.Challenge) != len(digest) ||
		!constantTimeEqual(keyDesc.Challenge, digest[:]) {
		return AttestationUnknown, fmt.Errorf(
			"android-key: challenge mismatch")
	}

	// Verify the keymaster software enforcement level is acceptable.
	if keyDesc.SoftwareEnforcedFlags&1 != 0 {
		// Key is software-backed, not hardware-backed.
		return AttestationBasic, nil
	}

	return AttestationAttCA, nil
}

// AndroidKeyDescription is parsed from the key description extension.
type AndroidKeyDescription struct {
	Challenge              []byte
	HardwareEnforcedFlags  uint32
	SoftwareEnforcedFlags  uint32
}

func parseAndroidKeyDescription(cert *x509.Certificate) (*AndroidKeyDescription, error) {
	const keyDescOID = "1.3.6.1.4.1.11129.2.1.17"
	ext, err := extractExtension(cert, keyDescOID)
	if err != nil {
		return nil, err
	}
	// Parse the ASN.1 sequence (simplified).
	// Production code should use full ASN.1 structure parsing.
	_ = ext
	return &AndroidKeyDescription{}, nil
}
```

### 2d. Apple Anonymous Attestation

Apple attestation uses a single X.509 certificate with an extension (OID
`1.2.840.113635.100.8.2`) that contains a nonce proving the credential was
created on Apple hardware. There is no chain — the certificate is validated
by its extension alone.

```go
// verifyAppleFull validates Apple anonymous attestation.
func verifyAppleFull(input AttestationInput) (AttestationKind, error) {
	x5cRaw, _ := input.AttStmt["x5c"].([]interface{})
	certs, err := parseX5CChain(x5cRaw)
	if err != nil || len(certs) != 1 {
		return AttestationUnknown, fmt.Errorf(
			"apple: expected single attestation certificate")
	}

	appleCert := certs[0]

	// Verify the nonce extension (OID 1.2.840.113635.100.8.2).
	const appleNonceOID = "1.2.840.113635.100.8.2"
	extData, err := extractExtension(appleCert, appleNonceOID)
	if err != nil {
		return AttestationUnknown, fmt.Errorf(
			"apple: nonce extension missing: %w", err)
	}

	// The nonce is the SHA-256 of (authData || clientDataHash).
	nonceData := append(input.AuthData, input.ClientDataHash...)
	expectedNonce := sha256.Sum256(nonceData)

	// Extract the nonce from the extension and compare.
	extractedNonce, err := extractAppleNonce(extData)
	if err != nil {
		return AttestationUnknown, fmt.Errorf(
			"apple: parse nonce: %w", err)
	}

	if !constantTimeEqual(extractedNonce, expectedNonce[:]) {
		return AttestationUnknown, fmt.Errorf(
			"apple: nonce mismatch")
	}

	// Apple attestation is anonymous — no chain to a vendor root.
	// Trust is based on the extension alone.
	return AttestationBasic, nil
}

func extractAppleNonce(extData []byte) ([]byte, error) {
	// The extension wraps the nonce in an ASN.1 SEQUENCE.
	// Parse the outer SEQUENCE, then the inner OCTET STRING.
	var wrapped asn1.RawValue
	if _, err := asn1.Unmarshal(extData, &wrapped); err != nil {
		return nil, err
	}
	if len(wrapped.Bytes) < 35 { // SEQUENCE + tag + nonce
		return nil, fmt.Errorf("nonce extension too short")
	}
	// Skip ASN.1 wrapper to extract the 32-byte nonce.
	nonceStart := len(wrapped.Bytes) - 32
	return wrapped.Bytes[nonceStart:], nil
}
```

---

## 3. Trust Score Computation

Trust scoring combines three factors into a 0-100 score:

| Factor | Weight | Range |
|--------|--------|-------|
| Attestation type | 40% | none=0, self=15, basic=30, attca=40 |
| Certification level | 35% | L1=15, L2=25, L3=35, unknown=10 |
| AAGUID status | 25% | certified=25, self=10, revoked=0, unknown=12 |

```go
package webauthn

// CertificationLevel from FIDO Authenticator Certification Levels.
type CertificationLevel int

const (
	CertUnknown CertificationLevel = 0
	CertL1      CertificationLevel = 1 // FIDO Functional Certification
	CertL2      CertificationLevel = 2 // FIDO Security Certification
	CertL3      CertificationLevel = 3 // FIDO Biometric Certification
)

// AuthenticatorStatus from FIDO MDS status reports.
type AuthenticatorStatus string

const (
	StatusNotFidoCertified AuthenticatorStatus = "NOT_FIDO_CERTIFIED"
	StatusFidoCertified    AuthenticatorStatus = "FIDO_CERTIFIED"
	StatusSelfAsserted     AuthenticatorStatus = "SELF_ASSERTION_RECEIVED"
	StatusRevoked          AuthenticatorStatus = "REVOKED"
)

// StatusReport mirrors the MDS status report structure.
type StatusReport struct {
	Status AuthenticatorStatus
}

// ComputeTrustScore produces a 0-100 trust score from a verification result.
func ComputeTrustScore(result *VerificationResult) int {
	score := 0

	// Factor 1: Attestation type (max 40).
	switch result.AttestationType {
	case AttestationAttCA:
		score += 40
	case AttestationBasic:
		score += 30
	case AttestationSelf:
		score += 15
	case AttestationNone:
		score += 0
	}

	// Factor 2: Certification level (max 35).
	level := certLevelFromAuthenticator(result.Authenticator)
	switch level {
	case CertL3:
		score += 35
	case CertL2:
		score += 25
	case CertL1:
		score += 15
	default:
		score += 10 // unknown but not zero — the device may be certified
		             // but not in MDS yet
	}

	// Factor 3: AAGUID status (max 25).
	status := StatusNotFidoCertified
	if result.StatusReport != nil {
		status = result.StatusReport.Status
	}
	switch status {
	case StatusFidoCertified:
		score += 25
	case StatusNotFidoCertified:
		score += 12
	case StatusSelfAsserted:
		score += 10
	case StatusRevoked:
		score += 0 // revoked authenticators get zero trust
	}

	if score > 100 {
		score = 100
	}
	return score
}

func certLevelFromAuthenticator(info *AuthenticatorInfo) CertificationLevel {
	if info == nil {
		return CertUnknown
	}
	// In production, this would come from MDS metadata's
	// certificationReports field. For now, use a heuristic.
	switch info.Manufacturer {
	case "Yubico", "Google", "Microsoft", "Apple":
		return CertL2
	default:
		return CertL1
	}
}
```

**Score interpretation guide:**

| Range | Meaning | Action |
|-------|---------|--------|
| 85-100 | Hardware-attested, FIDO-certified | Accept all registrations |
| 60-84 | Platform authenticator, basic attestation | Accept with logging |
| 35-59 | Self-attested or unknown device | Accept but flag for review |
| 0-34 | None attestation or revoked | Require step-up auth or reject |

---

## 4. AAGUID Registration and Management

GGID currently hardcodes 5 AAGUID entries in `attestation.go init()`. A
production system needs a registry that can be updated from FIDO MDS and
supports per-tenant overrides.

```go
package webauthn

import (
	"context"
	"sync"
)

// AAGUIDRegistry manages authenticator metadata and status reports.
// It is safe for concurrent use.
type AAGUIDRegistry struct {
	mu       sync.RWMutex
	entries  map[string]*registryEntry // keyed by lowercase AAGUID UUID string
}

type registryEntry struct {
	Info   *AuthenticatorInfo
	Status *StatusReport
	// CertLevel from MDS certification reports.
	CertLevel CertificationLevel
	// TrustedCAs is the set of root CA fingerprints for this authenticator.
	TrustedCAs [][]byte // SHA-256 fingerprints of root CA public keys
}

// NewAAGUIDRegistry creates an empty registry.
func NewAAGUIDRegistry() *AAGUIDRegistry {
	return &AAGUIDRegistry{entries: make(map[string]*registryEntry)}
}

// Register adds or updates an authenticator in the registry.
func (r *AAGUIDRegistry) Register(
	aaguid string,
	info *AuthenticatorInfo,
	status *StatusReport,
	certLevel CertificationLevel,
	trustedCAs [][]byte,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[normalizeAAGUID(aaguid)] = &registryEntry{
		Info:       info,
		Status:     status,
		CertLevel:  certLevel,
		TrustedCAs: trustedCAs,
	}
}

// Lookup retrieves authenticator metadata by AAGUID UUID string.
func (r *AAGUIDRegistry) Lookup(aaguid string) *AuthenticatorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if entry, ok := r.entries[normalizeAAGUID(aaguid)]; ok {
		return entry.Info
	}
	return nil
}

// GetStatus retrieves the current status report for an AAGUID.
func (r *AAGUIDRegistry) GetStatus(aaguid string) *StatusReport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if entry, ok := r.entries[normalizeAAGUID(aaguid)]; ok {
		return entry.Status
	}
	return nil
}

// IsRevoked checks if an authenticator's status is REVOKED.
func (r *AAGUIDRegistry) IsRevoked(aaguid string) bool {
	sr := r.GetStatus(aaguid)
	return sr != nil && sr.Status == StatusRevoked
}

// UpdateFromMDS bulk-updates the registry from a parsed FIDO MDS payload.
// This is typically called on a weekly schedule (see fido-mds-attestation.md
// for blob fetch/verify details).
func (r *AAGUIDRegistry) UpdateFromMDS(ctx context.Context,
	payload *MetadataBLOBPayload) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range payload.Entries {
		if entry.AAGUID == "" {
			continue // metadata for U2F authenticators, skip
		}
		aaguid := normalizeAAGUID(entry.AAGUID)
		r.entries[aaguid] = &registryEntry{
			Info: &AuthenticatorInfo{
				AAGUID:       aaguid,
				Name:         entry.MetadataStatement.Description,
				Manufacturer: entry.MetadataStatement.AttachmentHint,
			},
			Status: &StatusReport{
				Status: AuthenticatorStatus(entry.StatusReports[0].Status),
			},
		}
	}
}

// ExportAll returns all entries for admin/UI display.
func (r *AAGUIDRegistry) ExportAll() []registryEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]registryEntry, 0, len(r.entries))
	for _, e := range r.entries {
		result = append(result, *e)
	}
	return result
}

func normalizeAAGUID(aaguid string) string {
	return strings.ToLower(strings.TrimSpace(aaguid))
}
```

---

## 5. Attestation Certificate Chain Validation

> For MDS root CA management and blob verification, see
> [fido-mds-attestation.md](./fido-mds-attestation.md). This section covers
> the Go code for building and validating the attestation cert chain.

The attestation certificate chain goes from the leaf (authenticator's
attestation cert) through zero or more intermediates to a vendor root CA
that is either pinned in the MDS metadata or in a global FIDO root pool.

```go
package webauthn

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"
)

// MDSRootPool is the global pool of MDS-trusted root CAs.
var MDSRootPool *x509.CertPool

// GoogleRootPool is the pool of Google hardware attestation roots.
var GoogleRootPool *x509.CertPool

// ValidateCertChain validates an attestation certificate chain against
// the given root CA pool. The chain must be ordered: leaf, intermediate(s),
// with roots supplied in rootPool.
func ValidateCertChain(certs []*x509.Certificate,
	rootPool *x509.CertPool) error {

	if len(certs) == 0 {
		return fmt.Errorf("empty certificate chain")
	}

	if rootPool == nil {
		return fmt.Errorf("no root CA pool configured")
	}

	// Build the intermediate pool from certs[1:] (skip leaf).
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}

	leaf := certs[0]

	// Verify the chain.
	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         rootPool,
		Intermediates: intermediates,
		KeyUsages: []x509.ExtKeyUsage{
			x509.ExtKeyUsageAny, // attestation certs may have unusual EKUs
		},
		CurrentTime: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("x509 verify: %w", err)
	}

	return nil
}

// parseX5CChain parses a CBOR x5c array (from attStmt) into X.509 certs.
// The array elements are byte strings (DER-encoded certificates).
func parseX5CChain(x5c []interface{}) ([]*x509.Certificate, error) {
	certs := make([]*x509.Certificate, 0, len(x5c))
	for i, raw := range x5c {
		der, ok := raw.([]byte)
		if !ok {
			return nil, fmt.Errorf("x5c[%d]: not a byte string", i)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("x5c[%d]: parse: %w", i, err)
		}
		certs = append(certs, cert)
	}
	return certs, nil
}

// CheckRevocationStatus performs CRL or OCSP revocation checking on the
// leaf attestation certificate. This is optional but recommended for
// enterprise deployments.
func CheckRevocationStatus(cert *x509.Certificate) error {
	// CRL checking.
	if cert.CRLDistributionPoints != nil {
		for _, crlURL := range cert.CRLDistributionPoints {
			crl, err := fetchCRL(crlURL)
			if err != nil {
				continue // fail open on network errors
			}
			for _, revoked := range crl.TBSCertList.RevokedCertificates {
				if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {
					return fmt.Errorf(
						"certificate revoked (CRL %s)", crlURL)
				}
			}
		}
	}

	// OCSP checking (preferred when OCSP server is available).
	if len(cert.OCSPServer) > 0 {
		for _, ocspURL := range cert.OCSPServer {
			status, err := checkOCSP(cert, ocspURL)
			if err != nil {
				continue // fail open
			}
			if status == "revoked" {
				return fmt.Errorf(
					"certificate revoked (OCSP %s)", ocspURL)
			}
		}
	}

	return nil
}

// verifyCertAAGUID checks that the AAGUID extension in an attestation
// certificate matches the AAGUID in the authenticator data.
// FIDO attestation certs embed the AAGUID at OID 1.3.6.1.4.1.45724.2.1.1
func verifyCertAAGUID(cert *x509.Certificate, authData []byte) error {
	const fidoAAGUIDOID = "1.3.6.1.4.1.45724.2.1.1"
	ext, err := extractExtension(cert, fidoAAGUIDOID)
	if err != nil {
		// Some authenticators omit this extension. Log a warning.
		return nil
	}
	certAAGUID := ext // 16-byte AAGUID
	authDataAAGUID := ExtractAAGUIDFromAuthData(authData)
	if certAAGUID == nil || authDataAAGUID == nil {
		return nil // can't verify, skip
	}
	if len(certAAGUID) == 16 && len(authDataAAGUID) == 16 {
		if !constantTimeEqual(certAAGUID, authDataAAGUID) {
			return fmt.Errorf(
				"AAGUID in cert does not match authData")
		}
	}
	return nil
}

// fetchCRL downloads and parses a CRL from the given URL.
func fetchCRL(url string) (*x509.RevocationList, error) {
	// Implementation omitted — use net/http to fetch and
	// x509.ParseRevocationList to parse.
	return nil, fmt.Errorf("not implemented")
}

// checkOCSP queries an OCSP responder for certificate status.
func checkOCSP(cert *x509.Certificate,
	url string) (string, error) {
	// Implementation omitted — use golang.org/x/crypto/ocsp.
	return "", fmt.Errorf("not implemented")
}
```

---

## 6. Per-Tenant Attestation Policy

Different tenants have different security requirements. An enterprise tenant
may require platform authenticators with full attestation chains, while a
consumer tenant allows any authenticator including `"none"`.

```go
package webauthn

import (
	"fmt"
)

// TenantPolicy defines per-tenant WebAuthn attestation requirements.
type TenantPolicy struct {
	TenantID       string
	MinTrustScore  int              // minimum trust score (0-100)
	AllowedFormats []string         // e.g., ["packed", "apple", "tpm"]
	RequirePlatform bool            // only platform authenticators (no roaming)
	RequireUV       bool            // user verification required
	RejectNone      bool            // reject "none" attestation
	RejectRevoked   bool            // reject revoked authenticators
	AllowedAAGUIDs  []string        // allowlist of specific AAGUIDs
	BlockedAAGUIDs  []string        // blocklist of specific AAGUIDs
}

// Enforce checks a verification result against this tenant policy.
// Returns nil if the result passes, an error otherwise.
func (p *TenantPolicy) Enforce(result *VerificationResult) error {
	// Reject revoked authenticators.
	if p.RejectRevoked && result.StatusReport != nil &&
		result.StatusReport.Status == StatusRevoked {
		return fmt.Errorf("authenticator %s is revoked", result.AAGUID)
	}

	// Reject "none" attestation if policy requires attestation.
	if p.RejectNone && result.AttestationType == AttestationNone {
		return fmt.Errorf("attestation required, got 'none'")
	}

	// Check minimum trust score.
	if result.TrustScore < p.MinTrustScore {
		return fmt.Errorf(
			"trust score %d below minimum %d",
			result.TrustScore, p.MinTrustScore)
	}

	// Check format allowlist.
	if len(p.AllowedFormats) > 0 && !containsString(
		p.AllowedFormats, result.Format) {
		return fmt.Errorf(
			"attestation format %q not allowed for tenant %s",
			result.Format, p.TenantID)
	}

	// Check AAGUID blocklist.
	if len(p.BlockedAAGUIDs) > 0 && containsString(
		p.BlockedAAGUIDs, result.AAGUID) {
		return fmt.Errorf(
			"authenticator %s is blocked for tenant %s",
			result.AAGUID, p.TenantID)
	}

	// Check AAGUID allowlist (if set, only these are allowed).
	if len(p.AllowedAAGUIDs) > 0 && !containsString(
		p.AllowedAAGUIDs, result.AAGUID) {
		return fmt.Errorf(
			"authenticator %s not in allowlist for tenant %s",
			result.AAGUID, p.TenantID)
	}

	return nil
}

// PolicyPreset returns sensible default policies for common use cases.
func PolicyPreset(preset string) *TenantPolicy {
	switch preset {
	case "enterprise":
		return &TenantPolicy{
			MinTrustScore:   60,
			AllowedFormats:  []string{"packed", "tpm", "apple"},
			RequirePlatform: true,
			RequireUV:       true,
			RejectNone:      true,
			RejectRevoked:   true,
		}
	case "consumer":
		return &TenantPolicy{
			MinTrustScore:   0,
			RejectRevoked:   true,
		}
	case "regulated":
		return &TenantPolicy{
			MinTrustScore:   75,
			AllowedFormats:  []string{"packed", "tpm"},
			RequirePlatform: true,
			RequireUV:       true,
			RejectNone:      true,
			RejectRevoked:   true,
		}
	default:
		return &TenantPolicy{
			MinTrustScore:   0,
			RejectRevoked:   true,
		}
	}
}

// PolicyStore manages per-tenant policies with database-backed storage.
type PolicyStore interface {
	GetPolicy(ctx context.Context, tenantID string) (*TenantPolicy, error)
	SavePolicy(ctx context.Context, policy *TenantPolicy) error
}

// InMemoryPolicyStore is a thread-safe in-memory implementation.
type InMemoryPolicyStore struct {
	mu       sync.RWMutex
	policies map[string]*TenantPolicy
}

func NewInMemoryPolicyStore() *InMemoryPolicyStore {
	return &InMemoryPolicyStore{policies: make(map[string]*TenantPolicy)}
}

func (s *InMemoryPolicyStore) GetPolicy(
	ctx context.Context, tenantID string) (*TenantPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.policies[tenantID]; ok {
		return p, nil
	}
	// Default to consumer preset for unconfigured tenants.
	return PolicyPreset("consumer"), nil
}

func (s *InMemoryPolicyStore) SavePolicy(
	ctx context.Context, policy *TenantPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[policy.TenantID] = policy
	return nil
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
```

---

## 7. GGID WebAuthn Attestation Gap Analysis

### Current Implementation: `services/auth/internal/webauthn/attestation.go`

The existing file (176 lines) implements:

| Feature | Status | Details |
|---------|--------|---------|
| `"none"` format | **Implemented** | Returns nil unconditionally |
| `"packed"` format | **Partial** | Verifies EC2/RSA/EdDSA signatures but does NOT validate the x5c cert chain to a root CA |
| AAGUID extraction | **Implemented** | `ExtractAAGUIDFromAuthData` parses the 16-byte AAGUID from authData |
| AAGUID registry | **Stub** | 5 hardcoded entries in `init()`; no MDS integration, no per-tenant policy |
| `"tpm"` format | **No verification** | `VerifyAttestationFormat` returns `nil` — accepts unconditionally |
| `"android-key"` format | **No verification** | Returns `nil` |
| `"android-safetynet"` format | **No verification** | Returns `nil` |
| `"apple"` format | **No verification** | Returns `nil` |
| `"fido-u2f"` format | **No verification** | Returns `nil` |
| Trust scoring | **Not implemented** | No concept of trust score |
| Cert chain validation | **Not implemented** | No x509 chain building, no MDS root pool |
| Revocation checking | **Not implemented** | No CRL/OCSP |
| Per-tenant policy | **Not implemented** | All tenants get the same (no) verification |
| MDS integration | **Not implemented** | No blob fetch, no AAGUID update |

### Critical Gap: Format-Specific Stubs

The most dangerous code in `attestation.go` is lines 82-83:

```go
case "fido-u2f", "android-key", "android-safetynet", "tpm", "apple":
    return nil // Platform-specific: accept without full chain verification
```

This **accepts any attestation** from these formats without any
cryptographic verification. A malicious authenticator can claim `"tpm"`
or `"apple"` format and bypass attestation entirely.

### Handler Integration Gap

The handler (`handler.go` line 535) delegates attestation verification to
the go-webauthn library:

```go
credential, err := h.wbn.CreateCredential(user, *sd.data, parsedResponse)
```

The go-webauthn library performs its own attestation verification, but GGID
does NOT:
1. Access the library's attestation verification result
2. Apply per-tenant policy to the result
3. Record the attestation format, AAGUID, or trust score
4. Check the AAGUID against a registry or MDS
5. Enforce a minimum trust score

The custom `VerifyAttestationFormat` function in `attestation.go` is **not
called by the handler at all** — it is dead code from a coverage perspective
(only exercised by unit tests in `attestation_test.go`).

### Credential Model Gap

The `Credential` struct (handler.go lines 24-40) stores `AttestationType`
and `AAGUID` but has no fields for:
- `TrustScore int` — computed at registration time
- `CertChainValidated bool` — whether the x5c chain was validated
- `CertLevel CertificationLevel` — FIDO certification level
- `StatusCheckedAt *time.Time` — last MDS status check

---

## 8. Gap Analysis and Recommendations

### Priority 1: Fix Unconditional Format Acceptance (Effort: 3 days)

**Problem:** `tpm`, `android-key`, `android-safetynet`, `apple`, and
`fido-u2f` formats all return `nil` without verification.

**Action:** Implement format-specific verifiers per Section 2 of this
document. Start with `apple` (simplest: single cert + nonce extension) and
`packed` (most common: x5c chain validation). Mark formats without
verification as explicitly rejected with a clear error message instead of
silently accepting.

**Files to modify:**
- `services/auth/internal/webauthn/attestation.go` — implement verify functions
- `services/auth/internal/webauthn/attestation_test.go` — add format tests

### Priority 2: Wire Verification into Handler (Effort: 2 days)

**Problem:** Custom attestation verification is dead code — the handler uses
`h.wbn.CreateCredential` and ignores the attestation result.

**Action:** After `CreateCredential` succeeds, extract the attestation format
and AAGUID from the parsed response, run `VerifyAttestation()` (Section 1),
apply tenant policy, and store the trust score with the credential. Add
`TrustScore`, `CertLevel`, and `StatusCheckedAt` fields to the `Credential`
struct.

**Files to modify:**
- `services/auth/internal/webauthn/handler.go` — integrate pipeline
- `services/auth/internal/webauthn/attestation.go` — add pipeline types

### Priority 3: Implement Trust Scoring (Effort: 2 days)

**Problem:** No trust score concept exists. All registrations are treated
identically regardless of authenticator quality.

**Action:** Implement `ComputeTrustScore` per Section 3. Add a
`TrustScore` column to the `webauthn_credentials` table. Surface the score
in the credential list API and admin console so security teams can audit
which authenticators are enrolled.

### Priority 4: Per-Tenant Policy Engine (Effort: 3 days)

**Problem:** All tenants get the same (non-existent) attestation enforcement.

**Action:** Implement `TenantPolicy` per Section 6. Store policies in
PostgreSQL keyed by `tenant_id`. Load policy at registration time and enforce
before credential persistence. Provide preset templates (enterprise, consumer,
regulated) that admins can select from the console.

### Priority 5: MDS Integration (Effort: 5 days)

**Problem:** AAGUID registry has 5 hardcoded entries. No FIDO MDS
integration.

**Action:** Implement the `UpdateFromMDS` method on `AAGUIDRegistry`
(Section 4). Fetch and verify the MDS blob per
[fido-mds-attestation.md](./fido-mds-attestation.md). Run the update on a
weekly cron schedule. Cache the parsed blob in Redis to avoid re-fetching on
every registration.

### Summary Table

| Item | Effort | Risk if Unfixed |
|------|--------|----------------|
| P1: Format verification stubs | 3 days | **Critical** — attestation bypass |
| P2: Handler wiring | 2 days | **High** — verification code never runs |
| P3: Trust scoring | 2 days | Medium — no authenticator quality tracking |
| P4: Tenant policy | 3 days | Medium — no per-tenant enforcement |
| P5: MDS integration | 5 days | Low — manual AAGUID management works for MVP |
| **Total** | **15 days** | |

---

## References

- [W3C WebAuthn Level 2 — Attestation](https://www.w3.org/TR/webauthn-2/#sctn-attestation)
- [FIDO Metadata Service 3.0 Specification](https://fidoalliance.org/specs/fido-v2.0-rd-20180502/fido-metadata-service-v2.0-rd-20180502.html)
- [webauthn-attestation-chain.md](./webauthn-attestation-chain.md) — Format descriptions and cert chain structure
- [fido-mds-attestation.md](./fido-mds-attestation.md) — MDS blob fetch/verify and AAGUID lookup
- [webauthn-passkey-best-practices.md](./webauthn-passkey-best-practices.md) — General passkey recommendations
