# Identity Proofing and Verification

> Research document for integrating identity proofing into GGID IAM Suite.
> Scope: NIST IAL framework, document verification, biometric liveness, KBV,
> lifecycle management, and GGID integration design.

---

## 1. Overview

**Identity proofing** is the process of verifying that a real-world person is
who they claim to be **before** issuing any credential. It is a one-time (or
periodic re-verification) gate; authentication is the ongoing process of
confirming the same person returns.

### Key Distinction: Proofing vs Authentication

| Concept | Identity Proofing (IAL) | Authentication (AAL) |
|---|---|---|
| **When** | Once at enrollment / re-proof | Every session |
| **Question** | "Are you who you claim to be?" | "Are you the same person?" |
| **Standard** | NIST SP 800-63A | NIST SP 800-63B |

### NIST SP 800-63A

Defines three **Identity Assurance Levels (IAL)**. A revision (SP 800-63-4) is
in draft but the three-level structure remains.

### Use Cases

- **Government**: tax filing, benefits, voting (IAL2/IAL3)
- **Financial KYC**: bank accounts, wire transfers (IAL2)
- **Healthcare**: patient portals, e-prescribing (IAL2/IAL3)
- **Education**: enrollment, credential issuance (IAL2)
- **Low-risk**: social apps, forums (IAL1)

---

## 2. NIST IAL Levels

### IAL1 — Self-Asserted

The applicant self-asserts their identity attributes (name, email, date of
birth). No independent verification is required.

- **Evidence required**: none
- **Verification**: none; attributes are self-claimed
- **Use case**: low-risk services, social apps, discussion forums
- **GGID today**: current registration (email + password) is IAL1

### IAL2 — Remote Evidence

Requires at least one piece of **SUPERIOR** or **STRONG** evidence validated or
verified by a trusted authority.

- **Strong evidence**: remote government ID submission validated against authoritative source + selfie biometric match
- **Superior evidence**: in-person or validated remote proofing with strong biometric comparison
- **Typical flow**: upload ID → vendor OCR + doc authentication → live selfie with liveness → face match
- **Use case**: financial services, healthcare portals, government, online notarization

### IAL3 — In-Person or Supervised Remote

Adds physical presence requirements to IAL2. Requires biometric collection
under supervision.

- **In-person**: trained agent inspects physical documents and collects biometrics
- **Supervised remote**: live video session with agent overseeing document presentation
- **Use case**: high-assurance government, defense, healthcare provider credentialing

### IAL vs AAL Comparison

| IAL | Min AAL | What's Verified | Example Application |
|-----|---------|-----------------|---------------------|
| IAL1 | AAL1 | Nothing (self-asserted) | Social media account |
| IAL2 | AAL2 | Government ID + biometric match | Bank account, patient portal |
| IAL3 | AAL2 | In-person/supervised ID + biometrics | Defense systems, licensed professionals |

> IAL and AAL are independent — a service may require IAL2 with AAL3 or any
> combination appropriate to the risk profile.

---

## 3. Document Verification

### Process Flow

1. **Upload**: user captures government ID via in-app camera (license, passport, national ID)
2. **OCR extraction**: extract name, DOB, document number, issue/expiry, address
3. **Document authentication**: detect security features — holograms, UV ink, microprint, MRZ checksum
4. **Template matching**: compare layout, fonts, colors against known templates per issuing country
5. **Face match**: compare ID photo to live selfie with liveness detection

### Vendor Comparison

| Vendor | Strengths | Liveness | Pricing Model | Best For |
|--------|-----------|----------|---------------|----------|
| **Onfido** | Document + facial similarity; global ID coverage | Active + passive | Per-verification | Global onboarding |
| **Jumio** | ID + liveness; 200+ countries; strong compliance | Active (liveness) | Per-verification | Financial services, gaming |
| **Socure** | ID + fraud scoring + email/phone signals | Passive | Per-verification | High-fraud-risk verticals |
| **Persona** | Customizable workflows; no-code orchestrator | Active + passive | Tiered per-verification | Startups, flexible flows |
| **Stripe Identity** | Embedded in Stripe payment flow | Active | Per-verification | Stripe-based businesses |
| **AWS Rekognition** | Face comparison API (lower-level) | N/A (DIY) | Per-API-call | Custom integrations |
| **Veriff** | Video-based verification; strong EU coverage | Passive (video) | Per-verification | European markets |

### Integration Pattern in GGID

```go
// IdentityProofingClient defines the vendor-agnostic interface.
type IdentityProofingClient interface {
    InitProofing(ctx context.Context, userID uuid.UUID, method string) (sessionID string, err error)
    VerifyDocument(ctx context.Context, doc []byte, docType string) (*DocResult, error)
    VerifyBiometric(ctx context.Context, selfie []byte, docFace []byte) (*BiometricResult, error)
    GetStatus(ctx context.Context, sessionID string) (*ProofingStatus, error)
}
```

- GGID delegates document/biometric processing to the vendor API
- User experience: in-app camera capture via vendor SDK or web component
- Result: verification status + extracted attributes stored in GGID user profile
- Vendor implementations: `OnfidoClient`, `PersonaClient`, `JumioClient`

---

## 4. Knowledge-Based Verification (KBV)

### How KBV Works

KBV asks questions that only the real person should know, sourced from credit
header data and public records:

- "Which of these addresses did you live at in 2019?"
- "What was the approximate monthly payment on your auto loan?"

Typically 3–5 multiple-choice questions; all must be answered correctly.

### NIST Position on KBV (SP 800-63A, 2017 revision)

- **KBV alone is INSUFFICIENT for IAL2.** It may only be used as supplementary
  evidence alongside a primary document-based verification.
- **Rationale**: massive data breaches (Equifax 2017, etc.) made KBV answer data
  widely available to attackers, destroying the assumption that "only the real
  person knows these answers."
- **Trend**: KBV is being phased out in favor of document verification +
  biometric liveness. Some legacy systems still use it as a fallback.

### Implementation Notes

- **Vendors**: LexisNexis Instant Identify, Experian Precise ID, Equifax
- Questions generated dynamically from the applicant's credit file
- **Privacy concern**: using KBV requires sharing PII with credit bureaus, which
  may conflict with GDPR data minimization principles
- **GGID recommendation**: do NOT implement KBV as primary verification; use
  only as supplementary factor if a specific tenant requires it

---

## 5. Biometric Liveness Detection

### Purpose

Liveness detection prevents **presentation attacks** — spoofing the camera
with a photo, video, mask, or deepfake. Ensures a live human is physically
present during selfie capture.

### Active vs Passive Liveness

| Type | How It Works | UX | Security |
|------|-------------|-----|----------|
| **Active** | User performs actions: blink, turn head, read digits | Higher friction | Moderate |
| **Passive** | Analyzes video frames for texture, depth, motion | Invisible to user | Higher |

**Recommendation**: passive liveness for better UX and stronger security.

### Standards and Certification

- **ISO/IEC 30107-3**: Presentation Attack Detection (PAD) standard
- **PAD Levels**: Level 1 (basic — detects photos, simple masks) → Level 2
  (advanced — detects sophisticated silicone masks, deepfakes)
- **iBeta**: independent lab that certifies PAD systems to ISO 30107-3
- Look for vendors with iBeta Level 1 and Level 2 certification

---

## 6. Identity Proofing Lifecycle

```
Initiate ──▶ Collect Evidence ──▶ Verify (Vendor) ──▶ Store Result ──▶ Issue Credential
                                                                   │
Revoke ◀── Expiry Check ◀───────────────────────────────────────────┘
```

### Enrollment

1. **Initiate**: user requests verified status
2. **Collect evidence**: document upload + biometric selfie
3. **Verify**: vendor processes, returns pass/fail + extracted attributes
4. **Store**: verification status, IAL level, evidence type, expiry, verified attributes
5. **Issue**: IAL claim in JWT, verified badge in UI

### Maintenance

- **Re-proofing**: periodic (e.g., every 2 years for IAL2, annually for IAL3)
- **Attribute changes**: legal name change requires new document verification
- **Document expiry**: when the government ID used expires, flag for re-proofing

### Revocation

- **Fraud detected**: revoke verified status immediately
- **Impact**: user loses access to IAL2+ services until re-proofed
- **Audit trail**: every proofing event logged with timestamp, vendor, result, evidence type

---

## 7. GGID Integration Design

### Data Model

```sql
CREATE TABLE identity_verifications (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id),
    tenant_id     UUID NOT NULL,
    ial_level     SMALLINT NOT NULL DEFAULT 1,   -- 1, 2, or 3
    method        VARCHAR(50) NOT NULL,           -- 'document', 'in_person', 'kbv'
    vendor        VARCHAR(50),                    -- 'onfido', 'persona', 'jumio'
    vendor_session_id VARCHAR(255),
    status        VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending/verified/failed/revoked
    evidence_type VARCHAR(50),                    -- 'passport', 'drivers_license', 'national_id'
    verified_at   TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,                    -- re-proof deadline
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE verified_attributes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    attribute_name  VARCHAR(50) NOT NULL,   -- 'legal_name', 'date_of_birth', 'address'
    verified_value  TEXT NOT NULL,
    verified_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    verification_id UUID REFERENCES identity_verifications(id)
);
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/identity/proof/init` | Start proofing session (returns vendor SDK token/URL) |
| GET | `/api/v1/identity/proof/status` | Check current user's verification status |
| GET | `/api/v1/identity/proof/{user_id}` | Admin view for compliance audit |
| POST | `/api/v1/identity/proof/callback` | Vendor webhook — receives verification result |
| POST | `/api/v1/identity/proof/consent` | Record user consent for data sharing |

### Go Implementation Sketch

```go
package proofing

type Service struct {
    repo   Repository
    vendor IdentityProofingClient
    audit  audit.Publisher
}

func (s *Service) InitProofing(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (string, error) {
    // 1. Check existing verification status
    existing, err := s.repo.GetLatest(ctx, userID)
    if err != nil { return "", err }
    if existing != nil && existing.Status == "verified" && existing.ExpiresAt.After(time.Now()) {
        return "", ErrAlreadyVerified
    }

    // 2. Create vendor session
    sessionID, err := s.vendor.InitProofing(ctx, userID, "document")
    if err != nil { return "", err }

    // 3. Persist pending record
    v := &IdentityVerification{
        UserID: userID, TenantID: tenantID,
        IALLevel: 2, Method: "document",
        VendorSessionID: sessionID, Status: "pending",
    }
    return sessionID, s.repo.Create(ctx, v)
}

func (s *Service) HandleCallback(ctx context.Context, cb VendorCallback) error {
    v, err := s.repo.GetBySessionID(ctx, cb.SessionID)
    if err != nil { return err }

    if cb.Passed {
        v.Status = "verified"
        v.VerifiedAt = time.Now()
        v.ExpiresAt = time.Now().AddDate(2, 0, 0) // 2-year re-proof
    } else {
        v.Status = "failed"
    }

    s.audit.Publish(ctx, audit.Event{
        Type: "identity.proofed", UserID: v.UserID,
        Metadata: map[string]string{"ial": fmt.Sprintf("%d", v.IALLevel), "result": v.Status},
    })
    return s.repo.Update(ctx, v)
}
```

### JWT IAL Claim

```json
{
  "sub": "user-uuid",
  "ial": 2,
  "verified_claims": {
    "legal_name": "Jane Doe",
    "date_of_birth": "1990-01-15"
  },
  "iat": 1700000000,
  "exp": 1700003600
}
```

### Policy Integration

- **ABAC rule**: `require_ial(user, required_level)` — policy engine checks
  the user's current IAL against the required level for a resource
- **Per-tenant config**: each tenant configures which vendor to use, which IAL
  level is required, and re-proofing interval
- **Gateway enforcement**: middleware checks IAL claim in JWT; routes like
  `/admin/*` or `/transfers/*` can require `ial >= 2`

```go
// Middleware: enforce minimum IAL on protected routes
func RequireIAL(minLevel int) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := GetClaims(r.Context())
            if claims.IAL < minLevel {
                http.Error(w, "insufficient identity assurance", http.StatusForbidden)
                return
            }
            next.ServeHTTP, r)
        })
    }
}
```

### Consent

- User must consent before PII is shared with a verification vendor
- Consent record stored with timestamp, vendor, scope — GDPR-compliant flow

---

## 8. Privacy and Compliance

| Concern | Requirement | GGID Approach |
|---------|-------------|---------------|
| **Data minimization** | Store only what's needed | Store verification status + verified attributes, NOT raw documents |
| **Vendor data retention** | Limit how long vendor keeps data | Configure short retention (30–90 days); delete after verification |
| **GDPR** | Right to erasure | Verification data included in user deletion; vendor deletion API called |
| **HIPAA** | PHI protection | If healthcare tenant, verification data is PHI — encrypt at rest, BAA with vendor |
| **SOC 2** | Audit trail | All proofing events logged to audit service (NATS JetStream) |
| **Data residency** | EU-only processing | Some vendors (Veriff, Onfido) offer EU data residency |

### Key Privacy Principles

1. **Never store raw ID documents** in GGID — only verification result metadata
2. **Vendor acts as data processor** under DPA; GGID is controller
3. **Biometric templates are NOT stored** — vendor processes face match, returns only pass/fail
4. **Right to erasure** propagates to vendor via deletion API call
5. **Cross-border transfers** require SCC or adequacy decisions (GDPR)

---

## 9. Roadmap

| Phase | Deliverable | Effort |
|-------|-------------|--------|
| **Phase 1** | DB schema + status tracking + API skeleton | 3–4 days |
| **Phase 2** | Vendor integration (Onfido or Persona as first vendor) | 1–2 weeks |
| **Phase 3** | IAL claim in JWT + gateway route enforcement | 3–5 days |
| **Phase 4** | Re-proofing scheduler + expiry notifications | 3–5 days |
| **Phase 5** | Multi-vendor support + per-tenant vendor config | 1 week |

### Phase 1–2 Detail (~2 weeks)

- Create DB tables, implement `ProofingService` (InitProofing, GetStatus, HandleCallback)
- Integrate one vendor SDK (Persona for easiest API, or Onfido for global coverage)
- Add consent flow, audit logging, wire endpoints through gateway

### Phase 3 Detail (~1 week)

- Add `ial` claim to JWT, implement `RequireIAL` middleware, ABAC policy rule,
  console verification badge

### Future Considerations

- **Reusable credentials**: accept verified credentials from NIST-aligned providers (Login.gov)
- **Verifiable credentials**: issue W3C VC reflecting IAL level for cross-domain trust
- **Step-up proofing**: allow users to upgrade from IAL1 to IAL2 on demand
